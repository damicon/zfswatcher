//
// logger_syslog.go
//
// Copyright © 2012-2013 Damicon Kraa Oy
//
// This file is part of zfswatcher.
//
// Zfswatcher is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Zfswatcher is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with zfswatcher. If not, see <http://www.gnu.org/licenses/>.
//

package notifier

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path"
	"strings"
	"time"
)

// This implements the BSD style syslog protocol.

func connectSyslog(address string) (net.Conn, bool, error) {
	conntype := "udp"
	sep := false // this is needed later if SOCK_STREAM connections are implemented

	if strings.Index(address, "/") != -1 {
		conntype = "unixgram" // XXX should try different types?
	}
	var c net.Conn
	var err error

	for retries := 0; true; retries++ {
		c, err = net.Dial(conntype, address)
		if err == nil {
			break
		}
		if retries < 3 {
			checkInternalError("error connecting syslog socket (retrying)", err)
			time.Sleep(retry_SLEEP * time.Millisecond)
		} else {
			checkInternalError("error connecting syslog socket (giving up)", err)
			break
		}
	}
	return c, sep, err
}

func (n *Notifier) loggerSyslog(ch chan *Msg, address string, facility SyslogFacility) {
	defer n.wg.Done()
	tag := fmt.Sprintf("%s[%d]", path.Base(os.Args[0]), os.Getpid())

	var c net.Conn
	var sep bool
	var err error
	var connected bool = false

	c, sep, err = connectSyslog(address)
	if err == nil {
		connected = true
	}

	for m := range ch {
		switch m.MsgType {
		case MSGTYPE_MESSAGE:
			buf := []byte(m.SyslogString(facility, tag))

			if sep {
				if len(buf) > 1023 {
					buf = buf[:1022]
				}
				buf = append(buf, '\n')
			} else {
				if len(buf) > 1024 {
					buf = buf[:1023]
				}
			}
			for retries := 0; retries < 2; retries++ {
				if connected {
					_, err = c.Write(buf)
					if err == nil {
						break
					}
					checkInternalError("error writing to syslog socket", err)
					// try to reopen the socket if there was error:
					c.Close()
					connected = false
				}
				c, sep, err = connectSyslog(address)
				if err == nil {
					connected = true
				}
			}
		case MSGTYPE_REOPEN:
			if connected {
				c.Close()
				connected = false
			}
			c, sep, err = connectSyslog(address)
			if err == nil {
				connected = true
			}
		}
	}
	if connected {
		c.Close()
		connected = false
	}
}

func (n *Notifier) AddLoggerSyslog(s Severity, address string, facility SyslogFacility) error {
	switch {
	case s < severity_MIN || s > severity_MAX:
		return errors.New(`invalid "severity"`)
	case address == "":
		return errors.New(`"address" not defined`)
	case facility < syslog_FACILITY_MIN || facility > syslog_FACILITY_MAX:
		return errors.New(`invalid "facility"`)
	}
	ch := make(chan *Msg, chan_SIZE)
	n.wg.Add(1)
	go n.loggerSyslog(ch, address, facility)
	n.out = append(n.out, notifyOutput{severity: s, ch: ch})
	return nil
}

// eof
