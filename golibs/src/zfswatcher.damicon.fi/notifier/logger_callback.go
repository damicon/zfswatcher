//
// logger_callback.go
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

func (n Notifier) loggerCallback(ch chan *Msg, f func(*Msg)) {
	defer n.wg.Done()
	for m := range ch {
		f(m)
	}
}

func (n *Notifier) AddLoggerCallback(s Severity, f func(*Msg)) error {
	ch := make(chan *Msg, chan_SIZE)
	n.wg.Add(1)
	go n.loggerCallback(ch, f)
	n.out = append(n.out, notifyOutput{severity: s, ch: ch, attachment: true})
	return nil
}

// eof
