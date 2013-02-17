//
// notifier.go
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
	"github.com/snabb/smtp"
	"net"
	"os"
	"path"
	"strings"
	"sync"
	"time"
)

// Message severity levels, conforms to the syslog severity levels:

// Severity level.
type Severity uint32

// Implement fmt.Scanner interface.
func (s *Severity) Scan(state fmt.ScanState, verb rune) error {
	sevstr, err := state.Token(false, func(r rune) bool { return true })
	if err != nil {
		return err
	}
	sev, ok := severityCodes[string(sevstr)]
	if !ok {
		return errors.New(`invalid severity "` + string(sevstr) + `"`)
	}
	*s = sev
	return nil
}

const (
	severity_MIN Severity = 0
	EMERG        Severity = 0
	ALERT        Severity = 1
	CRIT         Severity = 2
	ERR          Severity = 3
	WARNING      Severity = 4
	NOTICE       Severity = 5
	INFO         Severity = 6
	DEBUG        Severity = 7
	severity_MAX Severity = 7
)

var severityStrings = []string{
	EMERG:   "emerg",
	ALERT:   "alert",
	CRIT:    "crit",
	ERR:     "err",
	WARNING: "warning",
	NOTICE:  "notice",
	INFO:    "info",
	DEBUG:   "debug",
}

var severityCodes = map[string]Severity{
	"emerg":   EMERG,
	"alert":   ALERT,
	"crit":    CRIT,
	"err":     ERR,
	"error":   ERR,
	"warn":    WARNING,
	"warning": WARNING,
	"notice":  NOTICE,
	"info":    INFO,
	"debug":   DEBUG,
}

// Syslog message facility codes:

type SyslogFacility uint32

const (
	syslog_FACILITY_MIN      SyslogFacility = 0
	syslog_FACILITY_KERN     SyslogFacility = 0
	syslog_FACILITY_USER     SyslogFacility = 1
	syslog_FACILITY_MAIL     SyslogFacility = 2
	syslog_FACILITY_DAEMON   SyslogFacility = 3
	syslog_FACILITY_AUTH     SyslogFacility = 4
	syslog_FACILITY_SYSLOG   SyslogFacility = 5
	syslog_FACILITY_LPR      SyslogFacility = 6
	syslog_FACILITY_NEWS     SyslogFacility = 7
	syslog_FACILITY_UUCP     SyslogFacility = 8
	syslog_FACILITY_CRON     SyslogFacility = 9
	syslog_FACILITY_AUTHPRIV SyslogFacility = 10
	syslog_FACILITY_FTP      SyslogFacility = 11
	syslog_FACILITY_LOCAL0   SyslogFacility = 16
	syslog_FACILITY_LOCAL1   SyslogFacility = 17
	syslog_FACILITY_LOCAL2   SyslogFacility = 18
	syslog_FACILITY_LOCAL3   SyslogFacility = 19
	syslog_FACILITY_LOCAL4   SyslogFacility = 20
	syslog_FACILITY_LOCAL5   SyslogFacility = 21
	syslog_FACILITY_LOCAL6   SyslogFacility = 22
	syslog_FACILITY_LOCAL7   SyslogFacility = 23
	syslog_FACILITY_MAX      SyslogFacility = 23
)

var syslogFacilityCodes = map[string]SyslogFacility{
	"kern":     syslog_FACILITY_KERN,
	"user":     syslog_FACILITY_USER,
	"mail":     syslog_FACILITY_MAIL,
	"daemon":   syslog_FACILITY_DAEMON,
	"auth":     syslog_FACILITY_AUTH,
	"syslog":   syslog_FACILITY_SYSLOG,
	"lpr":      syslog_FACILITY_LPR,
	"news":     syslog_FACILITY_NEWS,
	"uucp":     syslog_FACILITY_UUCP,
	"cron":     syslog_FACILITY_CRON,
	"authpriv": syslog_FACILITY_AUTHPRIV,
	"ftp":      syslog_FACILITY_FTP,
	"local0":   syslog_FACILITY_LOCAL0,
	"local1":   syslog_FACILITY_LOCAL1,
	"local2":   syslog_FACILITY_LOCAL2,
	"local3":   syslog_FACILITY_LOCAL3,
	"local4":   syslog_FACILITY_LOCAL4,
	"local5":   syslog_FACILITY_LOCAL5,
	"local6":   syslog_FACILITY_LOCAL6,
	"local7":   syslog_FACILITY_LOCAL7,
}

const retry_SLEEP = 500 // milliseconds
const chan_SIZE = 32

// Messages passed in the module internal channels:

type MsgType int

const (
	MSGTYPE_MESSAGE    MsgType = iota // normal log message
	MSGTYPE_ATTACHMENT                // additional verbose information
	MSGTYPE_FLUSH                     // send messages to delayed destinations (e-mail)
	MSGTYPE_REOPEN                    // re-open output file after log rotation etc
)

type Msg struct {
	Time     time.Time
	MsgType  MsgType
	Severity Severity
	Text     string
}

// private

func internalError(str string) {
	// This is an internal error in the notifier library.
	// What should be done with these errors?
	// Right now we just write the errors to STDERR.
	// This is not a good solution.
	fmt.Fprintf(os.Stderr, "%s [NOTIFIER] %s\n", time.Now().Format("2006-01-02 15:04:05"), str)
}

func checkInternalError(str string, err error) {
	if err != nil {
		internalError(fmt.Sprintf("%s: %s", str, err))
	}
}

func (m *Msg) MsgToStrings() (string, string, string) {
	return m.Time.Format("2006-01-02 15:04:05"),
		severityStrings[m.Severity],
		m.Text
}

func (m *Msg) MsgToStringDateTime() string {
	return m.Time.Format("2006-01-02 15:04:05") +
		" [" + severityStrings[m.Severity] + "] " +
		m.Text
}

func (m *Msg) MsgToStringTime() string {
	return m.Time.Format("15:04:05") +
		" [" + severityStrings[m.Severity] + "] " +
		m.Text
}

func (n *Notifier) loggerStdout(ch chan *Msg) {
	defer n.wg.Done()
	for m := range ch {
		switch m.MsgType {
		case MSGTYPE_MESSAGE:
			fmt.Println(m.MsgToStringDateTime())
		case MSGTYPE_ATTACHMENT:
			fmt.Println(">" + strings.Replace(strings.TrimRight(m.Text, "\n"), "\n", "\n>", -1))
		}
	}
}

func (n *Notifier) loggerFile(ch chan *Msg, filename string) {
	defer n.wg.Done()

	var fileopen bool

	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)

	if err == nil {
		fileopen = true
	} else {
		checkInternalError("error opening log file", err)
		fileopen = false
	}

	for m := range ch {
		switch m.MsgType {
		case MSGTYPE_MESSAGE:
			if !fileopen {
				continue
			}
			_, err = f.WriteString(m.MsgToStringDateTime() + "\n")
			checkInternalError("error writing log file", err)
		case MSGTYPE_ATTACHMENT:
			if !fileopen {
				continue
			}
			_, err = f.WriteString(">" +
				strings.Replace(strings.TrimRight(m.Text, "\n"), "\n", "\n>", -1) +
				"\n")
			checkInternalError("error writing log file", err)
		case MSGTYPE_REOPEN:
			if fileopen {
				err = f.Close()
				checkInternalError("error closing log file", err)
			}
			f, err = os.OpenFile(filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
			if err == nil {
				fileopen = true
			} else {
				checkInternalError("error re-opening log file", err)
				fileopen = false
			}
		}
	}
	if fileopen {
		err = f.Close()
		checkInternalError("error closing log file", err)
	}
}

// This implements the BSD style syslog protocol.

func (m *Msg) msgToSyslogString(facility SyslogFacility, tag string) string {
	return fmt.Sprintf("<%d>%s %s: %s", uint32(m.Severity)|(uint32(facility)<<3),
		m.Time.Format(time.Stamp), tag, m.Text)
}

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
			buf := []byte(m.msgToSyslogString(facility, tag))

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

func (n *Notifier) sendEmailSMTP(server, username, password, from, to, subject, text string) {
	defer n.wg.Done()
	var auth smtp.Auth

	text = "From: " + from + "\r\n" +
		"To: " + strings.Join(strings.Fields(to), ", ") + "\r\n" +
		"Date: " + time.Now().Format(time.RFC1123Z) + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" + strings.Replace(text, "\n", "\r\n", -1) + "\r\n"

	if username != "" {
		host, _, err := net.SplitHostPort(server)
		checkInternalError("error parsing mail host:port", err)
		auth = smtp.PlainAuth("", username, password, host)
	}
	for retries := 0; true; retries++ {
		err := smtp.SendMail(server, auth, from, strings.Fields(to), []byte(text))
		if err == nil {
			break
		}
		if retries < 3 {
			checkInternalError("error sending mail (retrying)", err)
			time.Sleep(retry_SLEEP * time.Millisecond)
		} else {
			checkInternalError("error sending mail (giving up)", err)
			break
		}
	}
}

func makeEmailText(mbuf, abuf []string) (text string) {
	if mbuf != nil {
		text += strings.Join(mbuf, "\n")
		text += "\n"
		mbuf = nil
	}
	if abuf != nil {
		text += strings.Join(abuf, "\n")
		text += "\n"
		abuf = nil
	}
	return text
}

func (n *Notifier) loggerEmailSMTP(ch chan *Msg, server, username, password, from, to, subject string, throttle time.Duration) {
	defer n.wg.Done()
	var mbuf []string
	var abuf []string
	var lastFlush time.Time
	var throttleTimer *time.Timer

	severityTrack := DEBUG

	for {
		var throttleC <-chan time.Time

		if throttleTimer != nil {
			throttleC = throttleTimer.C
		}
		var m *Msg
		var ok bool

		select {
		case m, ok = <-ch:
			// nothing
		case <-throttleC:
			m = &Msg{MsgType: MSGTYPE_FLUSH}
			ok = true
		}
		if !ok {
			break
		}
		switch m.MsgType {
		case MSGTYPE_MESSAGE:
			mbuf = append(mbuf, m.MsgToStringTime())
			if m.Severity < severityTrack {
				// keep track of worst severity within a batch of messages
				severityTrack = m.Severity
			}
		case MSGTYPE_ATTACHMENT:
			abuf = append(abuf, ">"+
				strings.Replace(strings.TrimRight(m.Text, "\n"), "\n", "\n>", -1)+"\n")
		case MSGTYPE_FLUSH:
			if len(mbuf) == 0 && len(abuf) == 0 {
				continue
			}
			text := makeEmailText(mbuf, abuf)
			if text == "" {
				continue
			}
			if throttle != 0 && time.Since(lastFlush) < throttle {
				if throttleTimer == nil {
					throttleTimer = time.NewTimer(throttle - time.Since(lastFlush))
				}
				continue
			}
			mbuf, abuf = nil, nil
			sevsubject := subject + " [" + severityStrings[severityTrack] + "]"
			severityTrack = DEBUG
			lastFlush = time.Now()
			if throttleTimer != nil {
				throttleTimer.Stop()
				throttleTimer = nil
			}
			n.wg.Add(1)
			go n.sendEmailSMTP(server, username, password, from, to, sevsubject, text)
		}
	}
	// exiting
	if throttleTimer != nil {
		throttleTimer.Stop()
	}
	// send the last entries:
	if text := makeEmailText(mbuf, abuf); text != "" {
		sevsubject := subject + " [" + severityStrings[severityTrack] + "]"
		n.wg.Add(1)
		go n.sendEmailSMTP(server, username, password, from, to, sevsubject, text)
	}
}

func (n Notifier) loggerCallback(ch chan *Msg, f func(*Msg)) {
	defer n.wg.Done()
	for m := range ch {
		f(m)
	}
}

func (n *Notifier) dispatcher() {
	defer n.wg.Done()
	// read messages from the input channel:
	for m := range n.ch {
		// forward the message to relevant loggers
		for _, out := range n.out {
			switch {
			case m.MsgType == MSGTYPE_ATTACHMENT && out.attachment == false:
				continue
			case m.MsgType == MSGTYPE_FLUSH && out.flush == false:
				continue
			case m.Severity <= out.severity:
				select {
				case out.ch <- m:
				default:
					checkInternalError("dispatcher error", errors.New("channel full"))
				}
			}
		}
	}
	// the input channel has been closed, so close the output channels then:
	n.ch = nil
	for _, out := range n.out {
		close(out.ch)
		out.ch = nil
	}
}

// public API

func GetSeverityCode(sevstr string) (sev Severity, err error) {
	sev, ok := severityCodes[sevstr]
	if !ok {
		return 0, errors.New(`invalid severity "` + sevstr + `"`)
	}
	return sev, nil
}

func GetSyslogFacilityCode(facstr string) (fac SyslogFacility, err error) {
	fac, ok := syslogFacilityCodes[facstr]
	if !ok {
		return 0, errors.New(`invalid facility "` + facstr + `"`)
	}
	return fac, nil
}

type notifyOutput struct {
	severity   Severity
	ch         chan *Msg
	attachment bool
	flush      bool
}

type Notifier struct {
	ch  chan *Msg
	out []notifyOutput
	wg  *sync.WaitGroup
}

func New() *Notifier {
	ch := make(chan *Msg, chan_SIZE)
	n := &Notifier{ch: ch, wg: &sync.WaitGroup{}}
	n.wg.Add(1)
	go n.dispatcher()
	return n
}

func (n *Notifier) AddLoggerStdout(s Severity) error {
	switch {
	case s < severity_MIN || s > severity_MAX:
		return errors.New(`invalid "severity"`)
	}
	ch := make(chan *Msg, chan_SIZE)
	n.wg.Add(1)
	go n.loggerStdout(ch)
	n.out = append(n.out, notifyOutput{severity: s, ch: ch, attachment: true})
	return nil
}

func (n *Notifier) AddLoggerFile(s Severity, file string) error {
	switch {
	case s < severity_MIN || s > severity_MAX:
		return errors.New(`invalid "severity"`)
	case file == "":
		return errors.New(`"file" not defined`)
	}
	ch := make(chan *Msg, chan_SIZE)
	n.wg.Add(1)
	go n.loggerFile(ch, file)
	n.out = append(n.out, notifyOutput{severity: s, ch: ch, attachment: true})
	return nil
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

func (n *Notifier) AddLoggerEmailSMTP(s Severity, server, user, pass, from, to, subject string, throttle time.Duration) error {
	switch {
	case s < severity_MIN || s > severity_MAX:
		return errors.New(`invalid "severity"`)
	case server == "":
		return errors.New(`"server" not defined`)
	case from == "":
		return errors.New(`"from" not defined`)
	case to == "":
		return errors.New(`"to" not defined`)
	case subject == "":
		return errors.New(`"subject" not defined`)
	}
	ch := make(chan *Msg, chan_SIZE)
	n.wg.Add(1)
	go n.loggerEmailSMTP(ch, server, user, pass, from, to, subject, throttle)
	n.out = append(n.out, notifyOutput{severity: s, ch: ch, attachment: true, flush: true})
	return nil
}

func (n *Notifier) AddLoggerCallback(s Severity, f func(*Msg)) error {
	ch := make(chan *Msg, chan_SIZE)
	n.wg.Add(1)
	go n.loggerCallback(ch, f)
	n.out = append(n.out, notifyOutput{severity: s, ch: ch, attachment: true})
	return nil
}

func sanitizeMessageText(str string) string {
	// XXX: how about other control characters?
	return strings.Replace(str, "\n", " ", -1)
}

func (n *Notifier) Send(s Severity, t string) error {
	if s < severity_MIN || s > severity_MAX {
		return errors.New(`invalid "severity"`)
	}
	n.ch <- &Msg{Time: time.Now(), MsgType: MSGTYPE_MESSAGE, Severity: s, Text: sanitizeMessageText(t)}
	return nil
}

func (n *Notifier) Attach(s Severity, t string) error {
	if s < severity_MIN || s > severity_MAX {
		return errors.New(`invalid "severity"`)
	}
	n.ch <- &Msg{Time: time.Now(), MsgType: MSGTYPE_ATTACHMENT, Severity: s, Text: t}
	return nil
}

func (n *Notifier) Flush() {
	n.ch <- &Msg{Time: time.Now(), MsgType: MSGTYPE_FLUSH}
}

func (n *Notifier) Reopen() {
	n.ch <- &Msg{Time: time.Now(), MsgType: MSGTYPE_REOPEN}
}

func (n *Notifier) Close() chan bool {
	// close the message channel to tell the goroutines they should quit:
	close(n.ch)
	// create a channel which can be used to wait for goroutines to quit:
	closeC := make(chan bool)
	// start a goroutine which closes the channel when all goroutines have quit:
	go func() {
		n.wg.Wait()
		close(closeC)
	}()
	// return that channel to the caller so they can wait on it if they want:
	return closeC
}

func (n *Notifier) Printf(s Severity, format string, v ...interface{}) {
	n.Send(s, fmt.Sprintf(format, v...))
}

func (n *Notifier) Print(s Severity, v ...interface{}) { n.Send(s, fmt.Sprint(v...)) }

// func (n *Notifier) Println(s Severity, v ...interface{}) { n.Send(s, fmt.Sprintln(v...)) }

// eof
