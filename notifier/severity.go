//
// severity.go
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
)

// Message severity levels, matches with syslog severity levels.
type Severity uint32

const (
	severity_MIN  Severity = 0
	EMERG         Severity = 0 // this is unfortunate
	ALERT         Severity = 1
	CRIT          Severity = 2
	ERR           Severity = 3
	WARNING       Severity = 4
	NOTICE        Severity = 5
	INFO          Severity = 6
	DEBUG         Severity = 7
	severity_MAX  Severity = 7
	SEVERITY_NONE Severity = 8 // discard these messages
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
	SEVERITY_NONE:    "none",
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
	"none":    SEVERITY_NONE,
}

// public API

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

// Implement fmt.Stringer interface.
func (s Severity) String() string {
	return severityStrings[s]
}

// eof
