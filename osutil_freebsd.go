//
// osutil_freebsd.go
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

// #include <stdlib.h>
import "C"

import (
	"errors"
	"syscall"
	"time"
	"unsafe"
)

// Returns system uptime as time.Duration.
func getSystemUptime() (uptime time.Duration, err error) {
	val, err := syscall.Sysctl("kern.boottime")
	if err != nil {
		return 0, err
	}
	buf := []byte(val)
	tv := (*syscall.Timeval)(unsafe.Pointer(&buf[0]))

	return time.Since(time.Unix(tv.Unix())), nil
}

// Returns system load averages.
func getSystemLoadaverage() ([3]float32, error) {
	avg := []C.double{0, 0, 0}

	n := C.getloadavg(&avg[0], C.int(len(avg)))

	if n == -1 {
		return [3]float32{0, 0, 0}, errors.New("load average unavailable")
	}
        return [3]float32{float32(avg[0]), float32(avg[1]), float32(avg[2])}, nil

}

// Device lookup paths. (This list comes from lib/libzfs/libzfs_import.c)
var deviceLookupPaths = [...]string{
	"/dev",
}

// eof
