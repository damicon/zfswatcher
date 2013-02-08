//
// osutil_linux.go
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
	"io/ioutil"
	"time"
)

// Returns system uptime as time.Duration.
func getSystemUptime() (uptime time.Duration, err error) {
	buf, err := ioutil.ReadFile("/proc/uptime")
	if err != nil {
		return uptime, err
	}
	var up, idle float64
	n, err := fmt.Sscanln(string(buf), &up, &idle)
	if err != nil {
		return uptime, err
	}
	if n != 2 {
		return uptime, errors.New("failed parsing /proc/uptime")
	}
	uptime = time.Duration(up) * time.Second

	return uptime, nil
}

// Returns system load averages.
func getSystemLoadaverage() (la [3]float32, err error) {
	buf, err := ioutil.ReadFile("/proc/loadavg")
	if err != nil {
		return la, err
	}
	n, err := fmt.Sscan(string(buf), &la[0], &la[1], &la[2])
	if err != nil {
		return la, err
	}
	if n != 3 {
		return la, errors.New("failed parsing /proc/loadavg")
	}

	return la, nil
}

// Device lookup paths. (This list comes from lib/libzfs/libzfs_import.c)
var deviceLookupPaths = [...]string{
	"/dev/disk/by-vdev",
	"/dev/disk/zpool",
	"/dev/mapper",
	"/dev/disk/by-uuid",
	"/dev/disk/by-id",
	"/dev/disk/by-path",
	"/dev/disk/by-label",
	"/dev",
}

// eof
