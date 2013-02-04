//
// zfswatcher.go
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
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"zfswatcher.damicon.fi/notifier"
)

// Config file processing.

const CFGFILE = "/etc/zfs/zfswatcher.conf"

var cfg struct {
	Main struct {
		Zpoolstatusrefresh uint
		Zpoolstatuscmd     string
		Zfslistrefresh     uint
		Zfslistcmd         string
		Zfslistusagecmd    string
		Pidfile            string
	}
	Severity struct {
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

// Other global variables.

var notify *notifier.Notifier
var optDebug *bool

var currentState struct {
	state []*PoolType
	usage map[string]*PoolUsageType
	mutex sync.RWMutex
}

var startTime time.Time

type stateToSeverityMap map[string]notifier.Severity

var sevCfg struct {
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

// ZFS pool disk usage.
type PoolUsageType struct {
	Name          string
	Avail         int64
	Used          int64
	Usedsnap      int64
	Usedds        int64
	Usedrefreserv int64
	Usedchild     int64
	Refer         int64
	Mountpoint    string
}

// Parse "zfs list -H -o name,avail,used,usedsnap,usedds,usedrefreserv,usedchild,refer,mountpoint" command output.
func parseZfsList(str string) map[string]*PoolUsageType {
	usagemap := make(map[string]*PoolUsageType)
	for lineno, line := range strings.Split(str, "\n") {
		if line == "" {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) != 9 {
			notify.Printf(notifier.CRIT, "invalid line %d in ZFS usage output: %s",
				lineno+1, line)
			notify.Attach(notifier.CRIT, str)
			continue
		}
		usagemap[f[0]] = &PoolUsageType{
			Name:          f[0],
			Avail:         unniceNumber(f[1]),
			Used:          unniceNumber(f[2]),
			Usedsnap:      unniceNumber(f[3]),
			Usedds:        unniceNumber(f[4]),
			Usedrefreserv: unniceNumber(f[5]),
			Usedchild:     unniceNumber(f[6]),
			Refer:         unniceNumber(f[7]),
			Mountpoint:    f[8],
		}
	}
	return usagemap
}

// This represents ZFS disk/container/whatever.
type DevEntry struct {
	name      string
	state     string
	read      int64
	write     int64
	cksum     int64
	rest      string
	subDevs   []int
	parentDev int
}

// Parse ZFS zpool status config section and return a tree of the volumes/containers/devices.
func parseConfstr(confstr string) (devs []*DevEntry, err error) {
	if confstr == "The configuration cannot be determined." {
		return nil, errors.New("configuration can not be determined")
	}
	var prevIndent int
	var devStack []int

	for _, line := range strings.Split(confstr, "\n") {
		if line == "" {
			continue
		}
		origlen := len(line)
		line = strings.TrimLeft(line, " ")
		indent := (origlen - len(line)) / 2
		f := strings.Fields(line)
		if f[0] == "NAME" && f[1] == "STATE" && f[2] == "READ" && f[3] == "WRITE" && f[4] == "CKSUM" {
			continue
		}
		// make a new dev entry
		var dev DevEntry
		dev.name = f[0]
		// set up defaults
		dev.read = -1
		dev.write = -1
		dev.cksum = -1
		if len(f) > 1 {
			dev.state = f[1]
		}
		if len(f) > 2 {
			dev.read = unniceNumber(f[2])
		}
		if len(f) > 3 {
			dev.write = unniceNumber(f[3])
		}
		if len(f) > 4 {
			dev.cksum = unniceNumber(f[4])
		}
		if len(f) > 5 {
			dev.rest = strings.Join(f[5:], " ")
		}

		switch {
		case indent == 0: // root level entry
			dev.parentDev = -1
			devs = append(devs, &dev)
			thisDev := len(devs) - 1
			devStack = []int{thisDev} // reset + push
		case indent > prevIndent: // subdev of the previous entry
			dev.parentDev = devStack[len(devStack)-1]
			devs = append(devs, &dev)
			thisDev := len(devs) - 1
			devs[dev.parentDev].subDevs = append(devs[dev.parentDev].subDevs, thisDev)
			devStack = append(devStack, thisDev) // push
		case indent == prevIndent: // same level as the previous entry
			devStack = devStack[:len(devStack)-1] // pop
			dev.parentDev = devStack[len(devStack)-1]
			devs = append(devs, &dev)
			thisDev := len(devs) - 1
			devs[dev.parentDev].subDevs = append(devs[dev.parentDev].subDevs, thisDev)
			devStack = append(devStack, thisDev) // push
		case indent < prevIndent: // dedent
			devStack = devStack[:len(devStack)-1-(prevIndent-indent)] // pop x N
			dev.parentDev = devStack[len(devStack)-1]
			devs = append(devs, &dev)
			thisDev := len(devs) - 1
			devs[dev.parentDev].subDevs = append(devs[dev.parentDev].subDevs, thisDev)
			devStack = append(devStack, thisDev) // push
		}
		prevIndent = indent
	}
	return devs, nil
}

// A single ZFS pool.
type PoolType struct {
	name    string
	state   string
	status  string
	action  string
	see     string
	scan    string
	devs    []*DevEntry
	errors  string
	infostr string
}

// Internal parser state for parseZpoolStatus() function.
type zpoolStatusParserState int

const (
	stSTART zpoolStatusParserState = iota
	stPOOL
	stSTATE
	stSTATUS
	stACTION
	stSEE
	stSCAN
	stCONFIG
	stERRORS
)

// Parse "zpool status" output.
func parseZpoolStatus(zpoolStatusOutput string) (pools []*PoolType, err error) {
	// catch a panic which might occur during parsing if we get something unexpected:
	defer func() {
		if p := recover(); p != nil {
			// get the panic location:
			buf := make([]byte, 4096)
			length := runtime.Stack(buf, false)
			buf = buf[:length] // truncate trailing garbage
			notify.Printf(notifier.CRIT, "panic parsing status output: %v", p)
			notify.Attach(notifier.CRIT, string(buf)+"\n"+zpoolStatusOutput)
			// force the return value err to be true:
			err = errors.New("panic parsing status output")
		}
	}()

	var curpool *PoolType
	var confstr string
	var poolinfostr string

	var s zpoolStatusParserState = stSTART

	for lineno, line := range strings.Split(zpoolStatusOutput, "\n") {
		poolinfostr += line + "\n"

		// this state machine implements a parser:
		switch {
		case s == stSTART && line == "no pools available":
			// pools will be empty slice
			return pools, nil
		case s == stSTART && len(line) >= 8 && line[:8] == "  pool: ":
			curpool = &PoolType{name: line[8:]}
			s = stPOOL
		case s == stPOOL && len(line) >= 8 && line[:8] == " state: ":
			curpool.state = line[8:]
			s = stSTATE
		case s == stSTATE && len(line) >= 8 && line[:8] == "status: ":
			curpool.status = line[8:]
			s = stSTATUS
		case s == stSTATUS && len(line) >= 1 && line[:1] == "\t":
			curpool.status += "\n" + line[1:]
		case s == stSTATUS && len(line) >= 8 && line[:8] == "action: ":
			curpool.action = line[8:]
			s = stACTION
		case s == stACTION && len(line) >= 1 && line[:1] == "\t":
			curpool.action += "\n" + line[1:]
		case (s == stSTATE || s == stACTION) && len(line) >= 8 &&
			line[:8] == "   see: ":
			curpool.see = line[8:]
			s = stSEE
		case (s == stSTATE || s == stACTION || s == stSEE) &&
			len(line) >= 7 && line[:7] == " scan: ":
			curpool.scan = line[7:]
			s = stSCAN
		// fix for 240245896aad46d0d41b0f9f257ff2abd09cb29b
		// released in zfs-0.6.0-rc14
		case (s == stSTATE || s == stACTION || s == stSEE) &&
			len(line) >= 8 && line[:8] == "  scan: ":
			curpool.scan = line[8:]
			s = stSCAN
		case s == stSCAN && len(line) >= 1 && line[:1] == "\t":
			curpool.scan += "\n" + line[1:]
		case s == stSCAN && len(line) >= 4 && line[:4] == "    ":
			curpool.scan += "\n" + line[4:]
		case (s == stSCAN || s == stSTATE || s == stACTION || s == stSEE) &&
			len(line) >= 7 && line[:7] == "config:":
			s = stCONFIG
			if line[7:] != "" {
				confstr = line[7:]
			}
		case s == stCONFIG && line == "":
			// skip
		case s == stCONFIG && len(line) >= 1 && line[:1] == "\t":
			confstr += "\n" + line[1:]
		case s == stCONFIG && len(line) >= 8 && line[:8] == "errors: ":
			curpool.errors = line[8:]
			s = stERRORS
		case s == stERRORS && line == "":
			// this is the end of a pool!
			curpool.devs, err = parseConfstr(confstr)
			if err != nil {
				notify.Print(notifier.ERR, "device configuration parse error: %s", err)
				notify.Attach(notifier.ERR, confstr)
			}
			confstr = ""
			curpool.infostr = poolinfostr
			poolinfostr = ""
			pools = append(pools, curpool)
			s = stSTART
		default:
			notify.Printf(notifier.CRIT, "invalid line %d in status output: %s", lineno+1, line)
			notify.Attach(notifier.CRIT, zpoolStatusOutput)
			return pools, errors.New("parser error")
		}
	}
	return pools, nil
}

// Keep track of highest (numerically lowest) severity level per pool.
func trackNotifications(notificationSev map[string]notifier.Severity, name string, s notifier.Severity) {
	if ns, ok := notificationSev[name]; ok {
		if s < ns {
			notificationSev[name] = s
		}
	} else {
		notificationSev[name] = s
	}
}

// Compare old state to new state and notify about differences.
func checkZpoolStatus(os, ns []*PoolType) {
	// os means "new state", ns means "new state"

	notificationSev := make(map[string]notifier.Severity) // notification messages sent per pool
	ledsToSet := make(map[string]ibpiID)

	// make a map of old pools:
	os_pools := map[string]*PoolType{}
	for _, pool := range os {
		os_pools[pool.name] = pool
	}
	// make a map of new pools:
	ns_pools := map[string]*PoolType{}
	for _, pool := range ns {
		ns_pools[pool.name] = pool
	}

	// go through old pool list to check for disappeared pools:
	for name := range os_pools {
		if ns_pools[name] == nil {
			notify.Printf(sevCfg.poolRemoved, `pool "%s" removed`, name)
		}
	}
	// go though new pool list:
	for name := range ns_pools {
		// check for new pools:
		if os_pools[name] == nil {
			notify.Printf(sevCfg.poolAdded, `pool "%s" added`, name)
			trackNotifications(notificationSev, name, sevCfg.poolAdded)
			continue
		}
		// pre-existing pool, perform checks to find changes:
		if ns_pools[name].state != os_pools[name].state {
			severity := sevCfg.poolStateMap.getSeverity(ns_pools[name].state)
			notify.Printf(severity, `pool "%s" state changed: %s -> %s`,
				name, os_pools[name].state, ns_pools[name].state)
			trackNotifications(notificationSev, name, severity)
		}
		if ns_pools[name].status != os_pools[name].status {
			if ns_pools[name].status != "" {
				notify.Printf(sevCfg.poolStatusChanged, `pool "%s" new status: %s`,
					name, ns_pools[name].status)
				trackNotifications(notificationSev, name, sevCfg.poolStatusChanged)
			} else {
				notify.Printf(sevCfg.poolStatusCleared, `pool "%s" status cleared`,
					name)
				trackNotifications(notificationSev, name, sevCfg.poolStatusCleared)
			}
		}
		if ns_pools[name].errors != os_pools[name].errors {
			notify.Printf(sevCfg.poolErrorsChanged, `pool "%s" new errors: %s`,
				name, ns_pools[name].errors)
			trackNotifications(notificationSev, name, sevCfg.poolErrorsChanged)
		}
		// make maps of devices in the pool:
		os_devs := map[string]*DevEntry{}
		for _, dev := range os_pools[name].devs {
			os_devs[dev.name] = dev
		}
		ns_devs := map[string]*DevEntry{}
		for _, dev := range ns_pools[name].devs {
			ns_devs[dev.name] = dev
		}
		// check for disappeared devices:
		for dname := range os_devs {
			if len(os_devs[dname].subDevs) != 0 {
				// intermediary "virtual" device, such as mirror-N or so, skip
				continue
			}
			if ns_devs[dname] == nil {
				notify.Printf(sevCfg.devRemoved, `pool "%s" device "%s" removed`,
					name, dname)
				trackNotifications(notificationSev, name, sevCfg.devRemoved)
			}
		}
		for dname := range ns_devs {
			if len(ns_devs[dname].subDevs) != 0 {
				// intermediary "virtual" device, such as mirror-N or so, skip
				continue
			}
			// check for new devices:
			if os_devs[dname] == nil {
				notify.Printf(sevCfg.devAdded, `pool "%s" device "%s" added`,
					name, dname)
				trackNotifications(notificationSev, name, sevCfg.devAdded)
				continue
			}
			// pre-existing device, perform checks to find changes:
			if ns_devs[dname].read > os_devs[dname].read {
				notify.Printf(sevCfg.devReadErrorsIncreased,
					`pool "%s" device "%s" read errors increased: %d -> %d`,
					name, dname, os_devs[dname].read, ns_devs[dname].read)
				trackNotifications(notificationSev, name, sevCfg.devReadErrorsIncreased)
			}
			if ns_devs[dname].write > os_devs[dname].write {
				notify.Printf(sevCfg.devWriteErrorsIncreased,
					`pool "%s" device "%s" write errors increased: %d -> %d`,
					name, dname, os_devs[dname].write, ns_devs[dname].write)
				trackNotifications(notificationSev, name, sevCfg.devWriteErrorsIncreased)
			}
			if ns_devs[dname].cksum > os_devs[dname].cksum {
				notify.Printf(sevCfg.devCksumErrorsIncreased,
					`pool "%s" device "%s" cksum errors increased: %d -> %d`,
					name, dname, os_devs[dname].cksum, ns_devs[dname].cksum)
				trackNotifications(notificationSev, name, sevCfg.devCksumErrorsIncreased)
			}
			if ns_devs[dname].state != os_devs[dname].state {
				severity := sevCfg.devStateMap.getSeverity(ns_devs[dname].state)
				notify.Printf(severity, `pool "%s" device "%s" state changed: %s -> %s`,
					name, dname, os_devs[dname].state, ns_devs[dname].state)
				trackNotifications(notificationSev, name, severity)
				// set leds
				if cfg.Leds.Enable && len(ns_devs[dname].subDevs) == 0 {
					ledsToSet[dname] = getIbpiIdFromDevState(ns_devs[dname].state)
				}
			}
			if ns_devs[dname].rest != os_devs[dname].rest {
				if ns_devs[dname].rest != "" {
					notify.Printf(sevCfg.devAdditionalInfoChanged,
						`pool "%s" device "%s" new additional info: %s`,
						name, dname, ns_devs[dname].rest)
					trackNotifications(notificationSev, name,
						sevCfg.devAdditionalInfoChanged)
				} else {
					notify.Printf(sevCfg.devAdditionalInfoCleared,
						`pool "%s" device "%s" additional info cleared`,
						name, dname)
					trackNotifications(notificationSev, name,
						sevCfg.devAdditionalInfoCleared)
				}
			}
		}
	}
	// attach complete pool status for pools which had notifications
	for name, severity := range notificationSev {
		notify.Attach(severity, ns_pools[name].infostr)
	}
	// update device LEDs
	if cfg.Leds.Enable && len(ledsToSet) > 0 {
		setDevLeds(ledsToSet)
	}
}

// Set the initial state of the LEDs.
func setupLeds(state []*PoolType) {
	ledsToSet := make(map[string]ibpiID)

	for _, pool := range state {
		for _, dev := range pool.devs {
			if len(dev.subDevs) == 0 {
				ledsToSet[dev.name] = getIbpiIdFromDevState(dev.state)
			}
		}
	}

	setDevLeds(ledsToSet)
}

// Print version information.
func version() {
	fmt.Println("zfswatcher", VERSION, "- ZFS pool monitoring and notification daemon")
	fmt.Println("Built with", getGoEnvironment())
	fmt.Println(`
Copyright © 2012-2013 Damicon Kraa Oy

Zfswatcher is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

Zfswatcher is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with zfswatcher. If not, see <http://www.gnu.org/licenses/>.
`)
}

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
func getSevCfg(sevStr string, cfgFile *string, errLabel string, defaultSev notifier.Severity) (sev notifier.Severity) {
	if sevStr != "" {
		var err error
		sev, err = notifier.GetSeverityCode(sevStr)
		checkCfgErr(*cfgFile, "severity", "", errLabel, err)
	} else {
		sev = defaultSev
	}
	return sev
}

// Check for and notify about configuration error.
func checkCfgErr(cfgfile, sect, prof, param string, err error) {
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
	os.Exit(2)
}

// Setup configuration and logging.
func setup() {
	// ensure that zpool/zfs commands do not use localized messages:
	os.Setenv("LC_ALL", "C")

	// command line flags:
	cfgFile := pflag.StringP("conf", "c", CFGFILE, "configuration file path")
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

	// set up some sane default configuration settings:
	cfg.Main.Zpoolstatusrefresh = 10
	cfg.Main.Zpoolstatuscmd = "zpool status"
	cfg.Main.Zfslistrefresh = 60
	cfg.Main.Zfslistcmd = "zfs list -H -o name,avail,used,usedsnap,usedds,usedrefreserv,usedchild,refer,mountpoint -d 0"
	cfg.Main.Zfslistusagecmd = "zfs list -H -o name,avail,used,usedsnap,usedds,usedrefreserv,usedchild,refer,mountpoint -r -t all"
	cfg.Leds.Ledctlcmd = "ledctl"

	// read configuration settings:
	err := gcfg.ReadFileInto(&cfg, *cfgFile)
	checkCfgErr(*cfgFile, "", "", "", err)

	if cfg.Severity.Poolstatemap != "" {
		sevCfg.poolStateMap, err = parseStateToSeverityMap(cfg.Severity.Poolstatemap)
		checkCfgErr(*cfgFile, "severity", "", "poolstatemap", err)
	}
	sevCfg.poolAdded = getSevCfg(cfg.Severity.Pooladded,
		cfgFile, "pooladded", notifier.INFO)
	sevCfg.poolRemoved = getSevCfg(cfg.Severity.Poolremoved,
		cfgFile, "poolremoved", notifier.INFO)
	sevCfg.poolStatusChanged = getSevCfg(cfg.Severity.Poolstatuschanged,
		cfgFile, "poolstatuschanged", notifier.INFO)
	sevCfg.poolStatusCleared = getSevCfg(cfg.Severity.Poolstatuscleared,
		cfgFile, "poolstatuscleared", notifier.INFO)
	sevCfg.poolErrorsChanged = getSevCfg(cfg.Severity.Poolerrorschanged,
		cfgFile, "poolerrorschanged", notifier.INFO)

	if cfg.Severity.Devstatemap != "" {
		sevCfg.devStateMap, err = parseStateToSeverityMap(cfg.Severity.Devstatemap)
		checkCfgErr(*cfgFile, "severity", "", "devstatemap", err)
	}
	sevCfg.devAdded = getSevCfg(cfg.Severity.Devadded,
		cfgFile, "devadded", notifier.INFO)
	sevCfg.devRemoved = getSevCfg(cfg.Severity.Devremoved,
		cfgFile, "devremoved", notifier.INFO)
	sevCfg.devReadErrorsIncreased = getSevCfg(cfg.Severity.Devreaderrorsincreased,
		cfgFile, "devreaderrorsincreased", notifier.INFO)
	sevCfg.devWriteErrorsIncreased = getSevCfg(cfg.Severity.Devwriteerrorsincreased,
		cfgFile, "devwriteerrorsincreased", notifier.INFO)
	sevCfg.devCksumErrorsIncreased = getSevCfg(cfg.Severity.Devcksumerrorsincreased,
		cfgFile, "devcksumerrorsincreased", notifier.INFO)
	sevCfg.devAdditionalInfoChanged = getSevCfg(cfg.Severity.Devadditionalinfochanged,
		cfgFile, "devadditionalinfochanged", notifier.INFO)
	sevCfg.devAdditionalInfoCleared = getSevCfg(cfg.Severity.Devadditionalinfocleared,
		cfgFile, "devadditionalinfocleared", notifier.INFO)

	if cfg.Leds.Devstatemap != "" {
		devStateToIbpiMap, err = parseDevStateToIbpiMap(cfg.Leds.Devstatemap)
		checkCfgErr(*cfgFile, "leds", "", "devstatemap", err)
	}

	for prof, c := range cfg.Logfile {
		if c.Enable {
			sev, err := notifier.GetSeverityCode(c.Level)
			checkCfgErr(*cfgFile, "logfile", prof, "level", err)
			err = notify.AddLoggerFile(sev, c.File)
			checkCfgErr(*cfgFile, "logfile", prof, "", err)
		}
	}
	for prof, c := range cfg.Syslog {
		if c.Enable {
			sev, err := notifier.GetSeverityCode(c.Level)
			checkCfgErr(*cfgFile, "syslog", prof, "level", err)
			fac, err := notifier.GetSyslogFacilityCode(c.Facility)
			checkCfgErr(*cfgFile, "syslog", prof, "facility", err)
			err = notify.AddLoggerSyslog(sev, c.Server, fac)
			checkCfgErr(*cfgFile, "syslog", prof, "", err)
		}
	}
	for prof, c := range cfg.Email {
		if c.Enable {
			sev, err := notifier.GetSeverityCode(c.Level)
			checkCfgErr(*cfgFile, "email", prof, "level", err)
			err = notify.AddLoggerEmailSMTP(sev,
				c.Server, c.Username, c.Password, c.From, c.To, c.Subject,
				time.Second*time.Duration(c.Throttle))
			checkCfgErr(*cfgFile, "email", prof, "", err)
		}
	}
	if cfg.Www.Enable && cfg.Www.Logbuffer > 0 {
		sev, err := notifier.GetSeverityCode(cfg.Www.Level)
		checkCfgErr(*cfgFile, "www", "", "level", err)
		err = notify.AddLoggerCallback(sev, wwwLogReceiver)
		checkCfgErr(*cfgFile, "www", "", "", err)
	}
	if cfg.Www.Enable {
		wwwSevClass, err = parseWwwSeverityToClassMap(cfg.Www.Severitycssclassmap)
		checkCfgErr(*cfgFile, "www", "", "severitycssclassmap", err)
		wwwPoolStateClass, err = parseStringMap(cfg.Www.Poolstatecssclassmap)
		checkCfgErr(*cfgFile, "www", "", "poolstatecssclassmap", err)
		wwwDevStateClass, err = parseStringMap(cfg.Www.Devstatecssclassmap)
		checkCfgErr(*cfgFile, "www", "", "devstatecssclassmap", err)
	}
	if *optTest {
		os.Exit(0)
	}
}

// The main program.
func main() {
	startTime = time.Now()

	// process command line arguments and config, initialize logging:
	setup()

	// setup signal handlers:
	sigCexit := make(chan os.Signal)
	signal.Notify(sigCexit, syscall.SIGTERM, syscall.SIGINT) // terminate gracefully
	sigChup := make(chan os.Signal)
	signal.Notify(sigChup, syscall.SIGHUP) // reopen log files
	sigCusr1 := make(chan os.Signal)
	signal.Notify(sigCusr1, syscall.SIGUSR1) // debug output

	// create a pid file if desired, remove it at the end of main()
	if cfg.Main.Pidfile != "" {
		makePidFile(cfg.Main.Pidfile)
		defer removePidFile(cfg.Main.Pidfile)
	}

	notify.Print(notifier.INFO, "zfswatcher starting")

	var statusTicker, zfslistTicker *time.Ticker
	var state []*PoolType

	// get the initial zpool status:
	zpoolStatusOutput, err := getCommandOutput(cfg.Main.Zpoolstatuscmd)
	if err != nil {
		notify.Print(notifier.CRIT, "exiting, getting ZFS status failed")
		goto EXIT
	}
	state, err = parseZpoolStatus(zpoolStatusOutput)
	if err != nil {
		notify.Print(notifier.CRIT, "exiting, parsing ZFS status failed")
		goto EXIT
	}

	// alert about big problems here if desired XXX
	// make device map XXX
	// load previous state from disk? XXX

	// set initial led states
	if cfg.Leds.Enable {
		setupLeds(state)
	}

	// start a web server goroutine:
	if cfg.Www.Enable {
		go webServer()
	}

	// initialize ticker timers and go in main loop:
	statusTicker = time.NewTicker(time.Duration(cfg.Main.Zpoolstatusrefresh) * time.Second)
	zfslistTicker = time.NewTicker(time.Duration(cfg.Main.Zfslistrefresh) * time.Second)

MAINLOOP:
	for {
		notify.Flush()

		select {
		// when the statusTicker ticks, get the new zpool status and compare:
		case <-statusTicker.C:
			zpoolStatusOutput, err := getCommandOutput(cfg.Main.Zpoolstatuscmd)
			if err != nil {
				notify.Print(notifier.CRIT, "getting ZFS status failed")
				continue
			}
			newstate, err := parseZpoolStatus(zpoolStatusOutput)
			if err != nil {
				notify.Print(notifier.CRIT, "parsing ZFS status failed")
				continue
			}
			checkZpoolStatus(state, newstate)
			state = newstate
			currentState.mutex.Lock()
			currentState.state = state
			currentState.mutex.Unlock()
		// get disk usage statistics:
		case <-zfslistTicker.C:
			zfsListOutput, err := getCommandOutput(cfg.Main.Zfslistcmd)
			if err != nil {
				notify.Print(notifier.CRIT, "getting ZFS disk usage failed")
				continue
			}
			usage := parseZfsList(zfsListOutput)
			if err != nil {
				notify.Print(notifier.CRIT, "parsing ZFS disk usage failed")
				continue
			}
			// checkZfsUsage() // XXX
			currentState.mutex.Lock()
			currentState.usage = usage
			currentState.mutex.Unlock()
		// signals:
		case <-sigCexit:
			break MAINLOOP
		case <-sigChup:
			notify.Print(notifier.DEBUG, "flushing and reopening logs")
			notify.Flush()
			notify.Reopen()
		case <-sigCusr1:
			var memstats runtime.MemStats
			runtime.ReadMemStats(&memstats)
			notify.Printf(notifier.DEBUG, "running with %d goroutines",
				runtime.NumGoroutine())
			notify.Printf(notifier.DEBUG, "memory statistics: %+v", memstats)
		}
	}
	// exiting, stop tickers, close everything, etc:
	statusTicker.Stop()
	zfslistTicker.Stop()
EXIT:
	notify.Print(notifier.INFO, "zfswatcher stopping")

	// XXX persist data?

	// ask logger to stop:
	notify.Flush()
	notifyCloseC := notify.Close()

	// wait a moment for logger goroutines to quit so that we get the last log messages:
	select {
	case <-notifyCloseC:
		if *optDebug {
			fmt.Println("exiting now: logger finished")
		}
	case <-sigCexit:
		if *optDebug {
			fmt.Println("exiting now: got another signal")
		}
	case <-time.After(time.Second * 10):
		if *optDebug {
			fmt.Println("exiting now: timer elapsed")
		}
	}
}

// eof
