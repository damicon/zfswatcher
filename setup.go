//
// setup.go
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

package main

import (
	"code.google.com/p/gcfg"
	"errors"
	"fmt"
	"github.com/ogier/pflag"
	"os"
	"strings"
	"time"
	"zfswatcher.damicon.fi/notifier"
)

// Config file processing.

const CFGFILE = "/etc/zfs/zfswatcher.conf"

var cfgFile string

type stateToSeverityMap map[string]notifier.Severity
type stringToStringMap map[string]string

type cfgType struct {
	Main struct {
		Zpoolstatusrefresh uint
		Zpoolstatuscmd     string
		Zfslistrefresh     uint
		Zfslistcmd         string
		Zfslistusagecmd    string
		Pidfile            string
	}
	Severity struct {
		Poolstatemap             stateToSeverityMap
		Pooladded                notifier.Severity
		Poolremoved              notifier.Severity
		Poolstatuschanged        notifier.Severity
		Poolstatuscleared        notifier.Severity
		Poolerrorschanged        notifier.Severity
		Devstatemap              stateToSeverityMap
		Devadded                 notifier.Severity
		Devremoved               notifier.Severity
		Devreaderrorsincreased   notifier.Severity
		Devwriteerrorsincreased  notifier.Severity
		Devcksumerrorsincreased  notifier.Severity
		Devadditionalinfochanged notifier.Severity
		Devadditionalinfocleared notifier.Severity
	}
	Leds struct {
		Enable      bool
		Ledctlcmd   string
		Devstatemap devStateToIbpiMap
	}
	Logfile map[string]*struct {
		Enable bool
		Level  string
		File   string
	}
	Syslog map[string]*struct {
		Enable   bool
		Level    string
		Server   string
		Facility string
	}
	Email map[string]*struct {
		Enable   bool
		Level    string
		Server   string
		Username string
		Password string
		From     string
		To       string
		Subject  string
		Throttle int64
	}
	Www struct {
		Enable               bool
		Level                string
		Logbuffer            int
		Bind                 string
		Templatedir          string
		Resourcedir          string
		Severitycssclassmap  severityToWwwClassMap
		Poolstatecssclassmap stringToStringMap
		Devstatecssclassmap  stringToStringMap
	}
	Wwwuser map[string]*struct {
		Enable   bool
		Password string
	}
}

var cfg *cfgType

func ScanMapHelper(state fmt.ScanState, verb rune) (map[string]string, error) {
	smap := make(map[string]string)
	for {
		tok, err := state.Token(true, nil)
		if err != nil {
			return nil, err
		}
		if len(tok) == 0 { // end of string
			break
		}
		str := string(tok)
		pair := strings.SplitN(str, ":", 2)
		if len(pair) != 2 {
			return nil, errors.New(`invalid map entry "` + str + `"`)
		}
		smap[pair[0]] = pair[1]
	}
	return smap, nil
}

// Implement fmt.Scanner interface.
func (ssmapp *stringToStringMap) Scan(state fmt.ScanState, verb rune) error {
	smap, err := ScanMapHelper(state, verb)
	if err != nil {
		return err
	}
	*ssmapp = smap
	return nil
}

// Implement fmt.Scanner interface.
func (ssmapp *stateToSeverityMap) Scan(state fmt.ScanState, verb rune) error {
	smap, err := ScanMapHelper(state, verb)
	if err != nil {
		return err
	}
	ssmap := make(stateToSeverityMap)
	for a, b := range smap {
		var severity notifier.Severity
		if n, err := fmt.Sscan(b, &severity); n != 1 {
			return err
		}
		ssmap[a] = severity
	}
	*ssmapp = ssmap
	return nil
}

// Finds an entry in stateToSeverityMap, returns INFO as default.
func (ssmap stateToSeverityMap) getSeverity(str string) notifier.Severity {
	sev, ok := ssmap[str]
	if !ok {
		sev = notifier.INFO
	}
	return sev
}

// Check for and notify about configuration error.
func checkCfgErr(cfgfile, sect, prof, param string, err error, errorSeen *bool) {
	if err == nil {
		return
	}
	var sectprof string
	switch {
	case sect != "" && prof != "":
		sectprof = ` [` + sect + ` "` + prof + `"]`
	case sect != "":
		sectprof = ` [` + sect + `]`
	}
	if param != "" {
		param = ` parameter "` + param + `"`
	}
	fmt.Fprintf(os.Stderr, "%s: Error in %s%s%s: %s\n", os.Args[0], cfgfile, sectprof, param, err)
	*errorSeen = true
}

// Read configuration.
func getCfg() *cfgType {
	var c cfgType
	var errorSeen bool

	// set up some sane default configuration settings:
	c.Main.Zpoolstatusrefresh = 10
	c.Main.Zpoolstatuscmd = "zpool status"
	c.Main.Zfslistrefresh = 60
	c.Main.Zfslistcmd = "zfs list -H -o name,avail,used,usedsnap,usedds,usedrefreserv,usedchild,refer,mountpoint -d 0"
	c.Main.Zfslistusagecmd = "zfs list -H -o name,avail,used,usedsnap,usedds,usedrefreserv,usedchild,refer,mountpoint -r -t all"
	c.Leds.Ledctlcmd = "ledctl"
	c.Severity.Pooladded = notifier.INFO
	c.Severity.Poolremoved = notifier.INFO
	c.Severity.Poolstatuschanged = notifier.INFO
	c.Severity.Poolstatuscleared = notifier.INFO
	c.Severity.Poolerrorschanged = notifier.INFO
	c.Severity.Devadded = notifier.INFO
	c.Severity.Devremoved = notifier.INFO
	c.Severity.Devreaderrorsincreased = notifier.INFO
	c.Severity.Devwriteerrorsincreased = notifier.INFO
	c.Severity.Devcksumerrorsincreased = notifier.INFO
	c.Severity.Devadditionalinfochanged = notifier.INFO
	c.Severity.Devadditionalinfocleared = notifier.INFO

	// read configuration settings:
	err := gcfg.ReadFileInto(&c, cfgFile)
	checkCfgErr(cfgFile, "", "", "", err, &errorSeen)

	// setup logging
	for prof, s := range c.Logfile {
		if s.Enable {
			sev, err := notifier.GetSeverityCode(s.Level)
			checkCfgErr(cfgFile, "logfile", prof, "level", err, &errorSeen)
			err = notify.AddLoggerFile(sev, s.File)
			checkCfgErr(cfgFile, "logfile", prof, "", err, &errorSeen)
		}
	}
	for prof, s := range c.Syslog {
		if s.Enable {
			sev, err := notifier.GetSeverityCode(s.Level)
			checkCfgErr(cfgFile, "syslog", prof, "level", err, &errorSeen)
			fac, err := notifier.GetSyslogFacilityCode(s.Facility)
			checkCfgErr(cfgFile, "syslog", prof, "facility", err, &errorSeen)
			err = notify.AddLoggerSyslog(sev, s.Server, fac)
			checkCfgErr(cfgFile, "syslog", prof, "", err, &errorSeen)
		}
	}
	for prof, s := range c.Email {
		if s.Enable {
			sev, err := notifier.GetSeverityCode(s.Level)
			checkCfgErr(cfgFile, "email", prof, "level", err, &errorSeen)
			err = notify.AddLoggerEmailSMTP(sev,
				s.Server, s.Username, s.Password, s.From, s.To, s.Subject,
				time.Second*time.Duration(s.Throttle))
			checkCfgErr(cfgFile, "email", prof, "", err, &errorSeen)
		}
	}
	if c.Www.Enable && c.Www.Logbuffer > 0 {
		sev, err := notifier.GetSeverityCode(c.Www.Level)
		checkCfgErr(cfgFile, "www", "", "level", err, &errorSeen)
		err = notify.AddLoggerCallback(sev, wwwLogReceiver)
		checkCfgErr(cfgFile, "www", "", "", err, &errorSeen)
	}
	if errorSeen {
		return nil
	}
	return &c
}

// Initial setup when the program starts.
func setup() {
	// ensure that zpool/zfs commands do not use localized messages:
	os.Setenv("LC_ALL", "C")

	// command line flags:
	pflag.StringVarP(&cfgFile, "conf", "c", CFGFILE, "configuration file path")
	optDebug = pflag.BoolP("debug", "d", false, "print debug information to stdout")
	optHashPassword := pflag.BoolP("passwordhash", "P", false, "hash web password")
	optTest := pflag.BoolP("test", "t", false, "test configuration and exit")
	optVersion := pflag.BoolP("version", "v", false, "print version information and exit")

	pflag.Parse()

	if pflag.NArg() > 0 {
		pflag.Usage()
		os.Exit(2)
	}
	if *optVersion {
		version()
		os.Exit(0)
	}
	if *optHashPassword {
		wwwHashPassword()
		os.Exit(0)
	}

	// initialize logging & notification:
	notify = notifier.New()

	if *optDebug || *optTest {
		notify.AddLoggerStdout(notifier.DEBUG)
	}

	cfg = getCfg()

	if *optTest {
		os.Exit(0)
	}
}

// eof
