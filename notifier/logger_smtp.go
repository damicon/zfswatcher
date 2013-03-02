//
// logger_smtp.go
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
	"github.com/snabb/smtp"
	"net"
	"strings"
	"time"
)

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
			mbuf = append(mbuf, m.TimeString())
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
			sevsubject := subject + " [" + severityTrack.String() + "]"
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
		sevsubject := subject + " [" + severityTrack.String() + "]"
		n.wg.Add(1)
		go n.sendEmailSMTP(server, username, password, from, to, sevsubject, text)
	}
}

// AddLoggerEmailSMTP adds an e-mail logging output. The e-mails are sent
// with ESMTP/SMTP whenever Flush() is called.
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

// eof
