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

type cfgType struct {
	Main struct {
		Zpoolstatusrefresh uint
		Zpoolstatuscmd     string
		Zfslistrefresh     uint
		Zfslistcmd         string
		Zfslistusagecmd    string
		Pidfile            string
	}
	Severity struct { // unparsed
		Poolstatemap             string
		Pooladded                string
		Poolremoved              string
		Poolstatuschanged        string
		Poolstatuscleared        string
		Poolerrorschanged        string
		Devstatemap              string
		Devadded                 string
		Devremoved               string
		Devreaderrorsincreased   string
		Devwriteerrorsincreased  string
		Devcksumerrorsincreased  string
		Devadditionalinfochanged string
		Devadditionalinfocleared string
	}
	sev struct { // same as previous but parsed format
		poolStateMap             stateToSeverityMap
		poolAdded                notifier.Severity
		poolRemoved              notifier.Severity
		poolStatusChanged        notifier.Severity
		poolStatusCleared        notifier.Severity
		poolErrorsChanged        notifier.Severity
		devStateMap              stateToSeverityMap
		devAdded                 notifier.Severity
		devRemoved               notifier.Severity
		devReadErrorsIncreased   notifier.Severity
		devWriteErrorsIncreased  notifier.Severity
		devCksumErrorsIncreased  notifier.Severity
		devAdditionalInfoChanged notifier.Severity
		devAdditionalInfoCleared notifier.Severity
	}
	Leds struct {
		Enable      bool
		Ledctlcmd   string
		Devstatemap string
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
		Severitycssclassmap  string
		Poolstatecssclassmap string
		Devstatecssclassmap  string
	}
	Wwwuser map[string]*struct {
		Enable   bool
		Password string
	}
}

var cfg *cfgType

// Finds an entry in stateToSeverityMap, returns INFO as default.
func (ssmap stateToSeverityMap) getSeverity(str string) notifier.Severity {
	sev, ok := ssmap[str]
	if !ok {
		sev = notifier.INFO
	}
	return sev
}

// Parses a string which describes how to map "state" strings to severity levels.
func parseStateToSeverityMap(str string) (stateToSeverityMap, error) {
	ssmap := make(stateToSeverityMap)

	for _, entry := range strings.Fields(str) {
		pair := strings.SplitN(entry, ":", 2)
		if len(pair) < 2 {
			return nil, errors.New(`invalid map entry "` + entry + `"`)
		}
		sev, err := notifier.GetSeverityCode(pair[1])
		if err != nil {
			return nil, err
		}
		ssmap[pair[0]] = sev
	}
	return ssmap, nil
}

// Convert severity string to notifier.SEVERITY (or use default unless defined). Complain
// about errors.
func getSevCfg(sevStr string, cfgFile string, errLabel string, defaultSev notifier.Severity, errorSeen *bool) (sev notifier.Severity) {
	if sevStr != "" {
		var err error
		sev, err = notifier.GetSeverityCode(sevStr)
		checkCfgErr(cfgFile, "severity", "", errLabel, err, errorSeen)
	} else {
		sev = defaultSev
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

	// read configuration settings:
	err := gcfg.ReadFileInto(&c, cfgFile)
	checkCfgErr(cfgFile, "", "", "", err, &errorSeen)

	if c.Severity.Poolstatemap != "" {
		c.sev.poolStateMap, err = parseStateToSeverityMap(c.Severity.Poolstatemap)
		checkCfgErr(cfgFile, "severity", "", "poolstatemap", err, &errorSeen)
	}
	c.sev.poolAdded = getSevCfg(c.Severity.Pooladded,
		cfgFile, "pooladded", notifier.INFO, &errorSeen)
	c.sev.poolRemoved = getSevCfg(c.Severity.Poolremoved,
		cfgFile, "poolremoved", notifier.INFO, &errorSeen)
	c.sev.poolStatusChanged = getSevCfg(c.Severity.Poolstatuschanged,
		cfgFile, "poolstatuschanged", notifier.INFO, &errorSeen)
	c.sev.poolStatusCleared = getSevCfg(c.Severity.Poolstatuscleared,
		cfgFile, "poolstatuscleared", notifier.INFO, &errorSeen)
	c.sev.poolErrorsChanged = getSevCfg(c.Severity.Poolerrorschanged,
		cfgFile, "poolerrorschanged", notifier.INFO, &errorSeen)

	if c.Severity.Devstatemap != "" {
		c.sev.devStateMap, err = parseStateToSeverityMap(c.Severity.Devstatemap)
		checkCfgErr(cfgFile, "severity", "", "devstatemap", err, &errorSeen)
	}
	c.sev.devAdded = getSevCfg(c.Severity.Devadded,
		cfgFile, "devadded", notifier.INFO, &errorSeen)
	c.sev.devRemoved = getSevCfg(c.Severity.Devremoved,
		cfgFile, "devremoved", notifier.INFO, &errorSeen)
	c.sev.devReadErrorsIncreased = getSevCfg(c.Severity.Devreaderrorsincreased,
		cfgFile, "devreaderrorsincreased", notifier.INFO, &errorSeen)
	c.sev.devWriteErrorsIncreased = getSevCfg(c.Severity.Devwriteerrorsincreased,
		cfgFile, "devwriteerrorsincreased", notifier.INFO, &errorSeen)
	c.sev.devCksumErrorsIncreased = getSevCfg(c.Severity.Devcksumerrorsincreased,
		cfgFile, "devcksumerrorsincreased", notifier.INFO, &errorSeen)
	c.sev.devAdditionalInfoChanged = getSevCfg(c.Severity.Devadditionalinfochanged,
		cfgFile, "devadditionalinfochanged", notifier.INFO, &errorSeen)
	c.sev.devAdditionalInfoCleared = getSevCfg(c.Severity.Devadditionalinfocleared,
		cfgFile, "devadditionalinfocleared", notifier.INFO, &errorSeen)

	if c.Leds.Devstatemap != "" {
		devStateToIbpiMap, err = parseDevStateToIbpiMap(c.Leds.Devstatemap)
		checkCfgErr(cfgFile, "leds", "", "devstatemap", err, &errorSeen)
	}

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
	if c.Www.Enable {
		wwwSevClass, err = parseWwwSeverityToClassMap(c.Www.Severitycssclassmap)
		checkCfgErr(cfgFile, "www", "", "severitycssclassmap", err, &errorSeen)
		wwwPoolStateClass, err = parseStringMap(c.Www.Poolstatecssclassmap)
		checkCfgErr(cfgFile, "www", "", "poolstatecssclassmap", err, &errorSeen)
		wwwDevStateClass, err = parseStringMap(c.Www.Devstatecssclassmap)
		checkCfgErr(cfgFile, "www", "", "devstatecssclassmap", err, &errorSeen)
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
