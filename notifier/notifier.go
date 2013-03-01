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
	"os"
	"strings"
	"sync"
	"time"
)

const retry_SLEEP = 500 // milliseconds
const chan_SIZE = 32

// Define the date/time format for outputs.
const (
	time_FORMAT      = "15:04:05"
	date_FORMAT      = "2006-01-02"
	date_time_FORMAT = "2006-01-02 15:04:05"
)

// private

func internalError(str string) {
	// This is an internal error in the notifier library.
	// What should be done with these errors?
	// Right now we just write the errors to STDERR.
	// This is not a good solution.
	fmt.Fprintf(os.Stderr, "%s [NOTIFIER] %s\n",
		time.Now().Format(date_time_FORMAT), str)
}

func checkInternalError(str string, err error) {
	if err != nil {
		internalError(fmt.Sprintf("%s: %s", str, err))
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
				// the zero value of Severity is EMERG, thus
				// this always forwards messages with
				// undefined severity
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

func sanitizeMessageText(str string) string {
	// XXX: how about other control characters?
	return strings.Replace(str, "\n", " ", -1)
}

func (n *Notifier) internal_send(msgtype MsgType, s Severity, t string) error {
	if s == SEVERITY_NONE {
		return nil // discard
	}
	if s < severity_MIN || s > severity_MAX {
		return errors.New(`invalid "severity"`)
	}
	n.ch <- &Msg{
		Time:     time.Now(),
		MsgType:  msgtype,
		Severity: s,
		Text:     sanitizeMessageText(t),
	}
	return nil
}

func (n *Notifier) Send(s Severity, t string) error {
	return n.internal_send(MSGTYPE_MESSAGE, s, t)
}

func (n *Notifier) Attach(s Severity, t string) error {
	return n.internal_send(MSGTYPE_ATTACHMENT, s, t)
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
