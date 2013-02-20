//
// leds.go
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
	"errors"
	"fmt"
	"github.com/damicon/zfswatcher/notifier"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// List of devices, paths and current LED statuses.

type ibpiID int32

const (
	IBPI_UNKNOWN ibpiID = iota
	IBPI_NORMAL
	IBPI_LOCATE
	IBPI_LOCATE_OFF
	IBPI_FAIL
	IBPI_REBUILD_P
	IBPI_REBUILD_S
	IBPI_PFA
	IBPI_HOTSPARE
	IBPI_CRITICAL_ARRAY
	IBPI_FAILED_ARRAY
	IBPI_UNDEFINED
)

var ibpiNameToId = map[string]ibpiID{
	"normal":         IBPI_NORMAL,
	"locate":         IBPI_LOCATE,
	"fail":           IBPI_FAIL,
	"rebuild":        IBPI_REBUILD_P,
	"rebuild_p":      IBPI_REBUILD_P,
	"rebuild_s":      IBPI_REBUILD_S,
	"pfa":            IBPI_PFA,
	"hotspare":       IBPI_HOTSPARE,
	"critical_array": IBPI_CRITICAL_ARRAY,
	"failed_array":   IBPI_FAILED_ARRAY,
	"undefined":      IBPI_UNDEFINED,
}

var ibpiToLedCtl = map[ibpiID]string{
	IBPI_NORMAL:         "normal",
	IBPI_LOCATE:         "locate", // paalle kun disk offline?
	IBPI_LOCATE_OFF:     "locate_off",
	IBPI_FAIL:           "failure",
	IBPI_REBUILD_P:      "rebuild_p",
	IBPI_REBUILD_S:      "rebuild",
	IBPI_PFA:            "pfa", // ei toimi
	IBPI_HOTSPARE:       "hotspare",
	IBPI_CRITICAL_ARRAY: "degraded",     // ei toimi
	IBPI_FAILED_ARRAY:   "failed_array", // ei toimi
}

type devLed struct {
	name   string
	path   string
	state  ibpiID
	locate bool
}

var (
	devLeds      map[string]*devLed
	devLedsMutex sync.Mutex
)

type devStateToIbpiMap map[string]ibpiID

// Implement fmt.Scanner interface.
func (simapp *devStateToIbpiMap) Scan(state fmt.ScanState, verb rune) error {
	smap := stringToStringMap{}
	err := smap.Scan(state, verb)
	if err != nil {
		return err
	}
	simap := make(devStateToIbpiMap)
	for a, b := range smap {
		var id ibpiID
		if n, err := fmt.Sscan(b, &id); n != 1 {
			return err
		}
		simap[a] = id
	}
	*simapp = simap
	return nil
}

// Implement fmt.Scanner interface.
func (i *ibpiID) Scan(state fmt.ScanState, verb rune) error {
	ibpistr, err := state.Token(false, func(r rune) bool { return true })
	if err != nil {
		return err
	}
	ibpiid, ok := ibpiNameToId[string(ibpistr)]
	if !ok {
		return errors.New(`invalid IBPI string "` + string(ibpistr) + `"`)
	}
	*i = ibpiid
	return nil
}

func (simap devStateToIbpiMap) getIbpiId(str string) ibpiID {
	id, ok := simap[str]
	if !ok {
		return IBPI_NORMAL
	}
	return id
}

func setDevLeds(devandled map[string]ibpiID) {
	var cmds []string

	devLedsMutex.Lock()
	// set the new led state in our internal array
	for dev, led := range devandled {
		if err := ensureDevLeds(dev); err != nil {
			notify.Print(notifier.ERR, "failed setting LED: ", err)
			continue
		}
		devLeds[dev].state = led
	}
	// make ledctl command line based on the status array
	// note: must update all leds as otherwise ledctl turns the leds off!
	for dev, devled := range devLeds {
		// skip devices which are currently missing
		st, err := os.Stat(devled.path)
		if err != nil || st.Mode()&os.ModeDevice == 0 {
			notify.Print(notifier.DEBUG, "skipping missing device LED: ", dev)
			continue
		}
		// reset unknown status to normal
		if devled.state == IBPI_UNKNOWN {
			devled.state = IBPI_NORMAL
		}
		// locate overrides other status
		if devled.locate {
			cmds = append(cmds, ibpiToLedCtl[IBPI_LOCATE]+"="+devled.path)
		} else {
			cmds = append(cmds, ibpiToLedCtl[devled.state]+"="+devled.path)
		}
	}
	devLedsMutex.Unlock()

	if len(cmds) > 0 {
		// notify.Print(notifier.DEBUG, `running: `, cfg.Leds.Ledctlcmd+" "+strings.Join(cmds, " "))
		cmd := exec.Command(cfg.Leds.Ledctlcmd, cmds...)
		cmd.Stdout, cmd.Stderr = nil, nil // suppress useless output
		err := cmd.Run()
		if err != nil {
			notify.Print(notifier.ERR, `running "`,
				cfg.Leds.Ledctlcmd+" "+strings.Join(cmds, " "),
				`" failed: `, err)
		}
	}
}

func ensureDevLeds(dev string) error {
	// must be called when devLedsMutex is locked!

	if devLeds == nil {
		devLeds = make(map[string]*devLed, 0)
	}
	_, ok := devLeds[dev]
	if ok {
		return nil
	}
	path, err := findDevicePath(dev)
	_ = err
	if err != nil {
		return err
	}
	devLeds[dev] = &devLed{name: dev, path: path}

	return nil
}

func locateOn(dev string) error {
	devLedsMutex.Lock()

	if err := ensureDevLeds(dev); err != nil {
		devLedsMutex.Unlock()
		return err
	}
	devLeds[dev].locate = true

	devLedsMutex.Unlock()

	setDevLeds(nil)

	return nil
}

func locateOff(dev string) error {
	devLedsMutex.Lock()

	if err := ensureDevLeds(dev); err != nil {
		devLedsMutex.Unlock()
		return err
	}
	devLeds[dev].locate = false

	devLedsMutex.Unlock()

	setDevLeds(nil)

	return nil
}

func locateQuery(dev string) (bool, error) {
	devLedsMutex.Lock()
	defer devLedsMutex.Unlock()

	if err := ensureDevLeds(dev); err != nil {
		return false, err
	}
	return devLeds[dev].locate, nil
}

// eof
