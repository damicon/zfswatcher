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

/*
  zfswatcher - ZFS pool monitoring and notification daemon

  Please see the web site for more information:

  http://zfswatcher.damicon.fi/
*/
package main

import (
	"fmt"
	"github.com/damicon/zfswatcher/notifier"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"
)

// Other global variables.

var notify *notifier.Notifier
var optDebug bool

var currentState struct {
	state []*PoolType
	usage map[string]*PoolUsageType
	mutex sync.RWMutex
}
var iostat struct {
	process *BackgroundProcess
	ch      chan *ZpoolIostatTable
}

var startTime time.Time

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
			notify.Printf(cfg.Severity.Poolremoved, `pool "%s" removed`, name)
		}
	}
	// go though new pool list:
	for name := range ns_pools {
		// check for new pools:
		if os_pools[name] == nil {
			notify.Printf(cfg.Severity.Pooladded, `pool "%s" added`, name)
			trackNotifications(notificationSev, name, cfg.Severity.Pooladded)
			continue
		}
		// pre-existing pool

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
				notify.Printf(cfg.Severity.Devremoved, `pool "%s" device "%s" removed`,
					name, dname)
				trackNotifications(notificationSev, name, cfg.Severity.Devremoved)
			}
		}
		for dname := range ns_devs {
			if len(ns_devs[dname].subDevs) != 0 {
				// intermediary "virtual" device, such as mirror-N or so, skip
				continue
			}
			// check for new devices:
			if os_devs[dname] == nil {
				notify.Printf(cfg.Severity.Devadded, `pool "%s" device "%s" added`,
					name, dname)
				trackNotifications(notificationSev, name, cfg.Severity.Devadded)
				continue
			}
			// pre-existing device, perform checks to find changes:
			if ns_devs[dname].read > os_devs[dname].read {
				notify.Printf(cfg.Severity.Devreaderrorsincreased,
					`pool "%s" device "%s" read errors increased: %d -> %d`,
					name, dname, os_devs[dname].read, ns_devs[dname].read)
				trackNotifications(notificationSev, name, cfg.Severity.Devreaderrorsincreased)
			}
			if ns_devs[dname].write > os_devs[dname].write {
				notify.Printf(cfg.Severity.Devwriteerrorsincreased,
					`pool "%s" device "%s" write errors increased: %d -> %d`,
					name, dname, os_devs[dname].write, ns_devs[dname].write)
				trackNotifications(notificationSev, name, cfg.Severity.Devwriteerrorsincreased)
			}
			if ns_devs[dname].cksum > os_devs[dname].cksum {
				notify.Printf(cfg.Severity.Devcksumerrorsincreased,
					`pool "%s" device "%s" cksum errors increased: %d -> %d`,
					name, dname, os_devs[dname].cksum, ns_devs[dname].cksum)
				trackNotifications(notificationSev, name, cfg.Severity.Devcksumerrorsincreased)
			}
			if ns_devs[dname].state != os_devs[dname].state {
				severity := cfg.Severity.Devstatemap.getSeverity(ns_devs[dname].state)
				notify.Printf(severity, `pool "%s" device "%s" state changed: %s -> %s`,
					name, dname, os_devs[dname].state, ns_devs[dname].state)
				trackNotifications(notificationSev, name, severity)
				// set leds
				if cfg.Leds.Enable && len(ns_devs[dname].subDevs) == 0 {
					ledsToSet[dname] = cfg.Leds.Devstatemap.getIbpiId(ns_devs[dname].state)
				}
			}
			if ns_devs[dname].rest != os_devs[dname].rest {
				if ns_devs[dname].rest != "" {
					notify.Printf(cfg.Severity.Devadditionalinfochanged,
						`pool "%s" device "%s" new additional info: %s`,
						name, dname, ns_devs[dname].rest)
					trackNotifications(notificationSev, name,
						cfg.Severity.Devadditionalinfochanged)
				} else {
					notify.Printf(cfg.Severity.Devadditionalinfocleared,
						`pool "%s" device "%s" additional info cleared`,
						name, dname)
					trackNotifications(notificationSev, name,
						cfg.Severity.Devadditionalinfocleared)
				}
			}
		}
		// check changes in the general pool information:
		if ns_pools[name].status != os_pools[name].status {
			if ns_pools[name].status != "" {
				notify.Printf(cfg.Severity.Poolstatuschanged, `pool "%s" new status: %s`,
					name, ns_pools[name].status)
				trackNotifications(notificationSev, name, cfg.Severity.Poolstatuschanged)
			} else {
				notify.Printf(cfg.Severity.Poolstatuscleared, `pool "%s" status cleared`,
					name)
				trackNotifications(notificationSev, name, cfg.Severity.Poolstatuscleared)
			}
		}
		if ns_pools[name].errors != os_pools[name].errors {
			notify.Printf(cfg.Severity.Poolerrorschanged, `pool "%s" new errors: %s`,
				name, ns_pools[name].errors)
			trackNotifications(notificationSev, name, cfg.Severity.Poolerrorschanged)
		}
		if ns_pools[name].state != os_pools[name].state {
			severity := cfg.Severity.Poolstatemap.getSeverity(ns_pools[name].state)
			notify.Printf(severity, `pool "%s" state changed: %s -> %s`,
				name, os_pools[name].state, ns_pools[name].state)
			trackNotifications(notificationSev, name, severity)
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

// Check ZFS space usage and send notifications if needed.
func checkZfsUsage(oldusage, newusage map[string]*PoolUsageType) {
	if len(cfg.Severity.Usedspace) == 0 {
		return
	}
	for pool := range oldusage {
		if _, ok := newusage[pool]; !ok {
			continue
		}
		ou := oldusage[pool].GetUsedPercent()
		nu := newusage[pool].GetUsedPercent()
		if !(nu > ou) {
			continue
		}
		maxlevel := 0
		for level := range cfg.Severity.Usedspace {
			if ou < level && nu >= level && level > maxlevel {
				maxlevel = level
			}
		}
		if maxlevel != 0 {
			notify.Printf(cfg.Severity.Usedspace[maxlevel],
				`pool "%s" usage reached %d%%`,
				pool, maxlevel)
		}
	}
}

// Set the initial state of the LEDs.
func setupLeds(state []*PoolType) {
	ledsToSet := make(map[string]ibpiID)

	for _, pool := range state {
		for _, dev := range pool.devs {
			if len(dev.subDevs) == 0 {
				ledsToSet[dev.name] = cfg.Leds.Devstatemap.getIbpiId(dev.state)
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

	// get the initial zpool status:
	out, err := getCommandOutput(cfg.Main.Zpoolstatuscmd)
	if err != nil {
		notify.Print(notifier.CRIT, "exiting, getting ZFS status failed")
		goto EXIT
	}
	currentState.state, err = parseZpoolStatus(out)
	if err != nil {
		notify.Print(notifier.CRIT, "exiting, parsing ZFS status failed")
		goto EXIT
	}
	out, err = getCommandOutput(cfg.Main.Zfslistcmd)
	if err != nil {
		notify.Print(notifier.CRIT, "exiting, getting ZFS disk usage failed")
		goto EXIT
	}
	currentState.usage = parseZfsList(out)
	if err != nil {
		notify.Print(notifier.CRIT, "exiting, parsing ZFS disk usage failed")
		goto EXIT
	}

	// alert about big problems here if desired XXX
	// make device map XXX
	// load previous state from disk? XXX

	// set initial led states
	if cfg.Leds.Enable {
		setupLeds(currentState.state)
	}

	// start iostat goroutine:
	if cfg.Main.Zpooliostatcmd != "" {
		iostat.process, err = NewBackgroundProcess(cfg.Main.Zpooliostatcmd)
		if err != nil {
			notify.Print(notifier.ERR, "failed to start iostat command")
		} else {
			iostat.ch = make(chan *ZpoolIostatTable)
			go ZpoolIostatStreamReader(iostat.ch, iostat.process.Out)
			// for now just print the output to stdout: XXX
			go func() {
				for i := range iostat.ch {
					fmt.Printf("iostat output: %+v\n", *i)
				}
			}()
		}
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
			checkZpoolStatus(currentState.state, newstate)
			currentState.mutex.Lock()
			currentState.state = newstate
			currentState.mutex.Unlock()
		// get disk usage statistics:
		case <-zfslistTicker.C:
			zfsListOutput, err := getCommandOutput(cfg.Main.Zfslistcmd)
			if err != nil {
				notify.Print(notifier.CRIT, "getting ZFS disk usage failed")
				continue
			}
			newusage := parseZfsList(zfsListOutput)
			if err != nil {
				notify.Print(notifier.CRIT, "parsing ZFS disk usage failed")
				continue
			}
			checkZfsUsage(currentState.usage, newusage)
			currentState.mutex.Lock()
			currentState.usage = newusage
			currentState.mutex.Unlock()
		// signals:
		case <-sigCexit:
			break MAINLOOP
		case <-sigChup:
			notify.Print(notifier.DEBUG, "reconfiguring, reopening logs")
			reconfigure()
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

	if iostat.process != nil {
		iostat.process.Stop()
	}

	// XXX persist data?

	// ask logger to stop:
	notifyCloseC := notify.Close()

	// wait a moment for logger goroutines to quit so that we get the last log messages:
	select {
	case <-notifyCloseC:
		if optDebug {
			fmt.Println("exiting now: logger finished")
		}
	case <-sigCexit:
		if optDebug {
			fmt.Println("exiting now: got another signal")
		}
	case <-time.After(time.Second * 10):
		if optDebug {
			fmt.Println("exiting now: timer elapsed")
		}
	}
}

// eof
