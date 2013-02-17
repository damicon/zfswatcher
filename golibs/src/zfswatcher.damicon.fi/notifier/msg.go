//
// msg.go
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
	"fmt"
	"time"
)

// Messages passed in the channels:

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

// Implement the fmt.Stringer interface.
func (m *Msg) String() string {
	return m.Time.Format(date_time_FORMAT) +
		" [" + m.Severity.String() + "] " +
		m.Text
}

func (m *Msg) Strings() (date_time string, severity string, text string) {
	return m.Time.Format(date_time_FORMAT), m.Severity.String(), m.Text
}

func (m *Msg) TimeString() string {
	return m.Time.Format(time_FORMAT) +
		" [" + m.Severity.String() + "] " +
		m.Text
}

func (m *Msg) SyslogString(facility SyslogFacility, tag string) string {
	return fmt.Sprintf("<%d>%s %s: %s",
		uint32(m.Severity)|(uint32(facility)<<3),
		m.Time.Format(time.Stamp), tag, m.Text)
}

// eof
