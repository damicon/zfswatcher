//
// logger_stdout.go
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
	"strings"
)

func (n *Notifier) loggerStdout(ch chan *Msg) {
	defer n.wg.Done()
	for m := range ch {
		switch m.MsgType {
		case MSGTYPE_MESSAGE:
			fmt.Println(m.String())
		case MSGTYPE_ATTACHMENT:
			fmt.Println(">" + strings.Replace(strings.TrimRight(m.Text, "\n"), "\n", "\n>", -1))
		}
	}
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

// eof
