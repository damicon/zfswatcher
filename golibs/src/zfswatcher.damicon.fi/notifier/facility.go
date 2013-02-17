//
// favility.go
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

// Syslog message facility codes.
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

// public API

// Implement fmt.Scanner interface.
func (f *SyslogFacility) Scan(state fmt.ScanState, verb rune) error {
	facstr, err := state.Token(false, func(r rune) bool { return true })
	if err != nil {
		return err
	}
	fac, ok := syslogFacilityCodes[string(facstr)]
	if !ok {
		return errors.New(`invalid facility "` + string(facstr) + `"`)
	}
	*f = fac
	return nil
}

// eof
