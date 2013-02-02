//
// util.go
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
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
	"zfswatcher.damicon.fi/notifier"
)

// Run external command and capture output.
func getCommandOutput(cmdstr string) (string, error) {
	cmd := strings.Fields(cmdstr)
	out, err := exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
	if err != nil {
		notify.Print(notifier.CRIT,
			`running "`, cmdstr, `" failed: `, err)
		if len(out) != 0 {
			notify.Attach(notifier.CRIT, string(out))
		}
		return "", err
	}
	return string(out), err
}

// Convert floating point number with an optional multiplier suffix to the
// proper int64 value.
// For example: 1.5M = 1.5 * 1024 * 1024
func unniceNumber(str string) int64 {
	if str == "-" {
		return -1
	}
	var mul string
	if mulpos := strings.IndexAny(str, "KMGTPE"); mulpos >= 0 {
		mul = str[mulpos : mulpos+1]
		str = str[:mulpos]
	}
	val, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return -1
	}
	if mul != "" {
		for i := 0; i <= strings.Index("KMGTPE", mul); i++ {
			val *= 1024
		}
	}
	return int64(val)
}

// Convert integer to a floating point string with a suffix denoting multiples.
// This should match the zfs_nicenum() implementation in ZoL
// lib/libzfs/libzfs_util.c
func niceNumber(num int64) (str string) {
	if num == -1 {
		return "-"
	}
	n := num
	var index uint64 = 0

	for n > 1024 {
		n /= 1024
		index++
	}
	u := " KMGTPE"[index : index+1]

	switch {
	case index == 0:
		str = fmt.Sprintf("%d", n)
	case num&((int64(1)<<(10*index))-1) == 0:
		str = fmt.Sprintf("%d%s", n, u)
	default:
		for i := 2; i >= 0; i-- {
			str = fmt.Sprintf("%.*f%s", i, float64(num)/float64(int64(1)<<(10*index)), u)
			if len(str) <= 5 {
				break
			}
		}
	}
	return str
}

// Returns Go environment information string.
func getGoEnvironment() string {
	return fmt.Sprintf("%s %s (%s/%s)", runtime.Compiler, runtime.Version(),
		runtime.GOOS, runtime.GOARCH)
}

// Returns system uptime as time.Duration.
func getSystemUptime() (uptime time.Duration, err error) {
	// XXX works only on linux for now

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
	// XXX works only on linux for now

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

// Utility function for myDurationString.
func fmtInt(buf []byte, v uint64) int {
	w := len(buf)
	if v == 0 {
		w--
		buf[w] = '0'
	} else {
		for v > 0 {
			w--
			buf[w] = byte(v%10) + '0'
			v /= 10
		}
	}
	return w
}

// Implementation of "func (d Duration) String() string" which returns the
// amount of days as well (but no fractions of seconds).
func myDurationString(d time.Duration) string {
	if d == time.Duration(0) {
		return "unknown"
	}
	// stolen from src/pkg/time/time.go:
	var buf [32]byte
	w := len(buf)

	u := uint64(d.Seconds())
	neg := d < 0
	if neg {
		u = -u
	}

	w--
	buf[w] = 's'

	// u is now integer seconds
	w = fmtInt(buf[:w], u%60)
	u /= 60

	// u is now integer minutes
	if u > 0 {
		w--
		buf[w] = 'm'
		w = fmtInt(buf[:w], u%60)
		u /= 60

		// u is now integer hours
		if u > 0 {
			w--
			buf[w] = 'h'
			w = fmtInt(buf[:w], u%24)
			u /= 24

			// u is now integer days
			if u > 0 {
				w--
				buf[w] = 'd'
				w = fmtInt(buf[:w], u)
			}
		}
	}
	if neg {
		w--
		buf[w] = '-'
	}

	return string(buf[w:])
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

// Find full device path.
func findDevicePath(dev string) (string, error) {
	for _, prefix := range deviceLookupPaths {
		path := prefix + "/" + dev
		st, err := os.Stat(path)
		if err == nil && st.Mode()&os.ModeDevice != 0 {
			return path, nil
		}
	}
	return "", errors.New(`device "` + dev + `" not found`)
}

// The pool name must begin with a letter, and can only contain alphanumeric
// characters as well as underscore ("_"), dash ("-"), period ("."),
// colon (":"), and space (" "). The pool names "mirror", "raidz", "spare"
// and "log" are reserved, as are names beginning with the pattern "c[0-9]".
var illegalPoolNameRegex = regexp.MustCompile(`^(?:mirror|raidz|spare|log|c[0-9])$`)
var legalPoolNameRegex = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_\-.: ]*$`)

// Verify if pool name is legal.
func legalPoolName(str string) bool {
	if str == "" {
		return false
	}
	if illegalPoolNameRegex.MatchString(str) {
		return false
	}
	if legalPoolNameRegex.MatchString(str) {
		return true
	}
	return false
}

// Parse a string map.
func parseStringMap(str string) (map[string]string, error) {
	smap := make(map[string]string)

	for _, entry := range strings.Fields(str) {
		pair := strings.SplitN(entry, ":", 2)
		if len(pair) < 2 {
			return nil, errors.New(`invalid map entry "` + entry + `"`)
		}
		smap[pair[0]] = pair[1]
	}
	return smap, nil
}

// Write our pid to a file.
func makePidFile(filename string) (err error) {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(f, "%d\n", os.Getpid())
	if err != nil {
		f.Close()
		os.Remove(filename)
		return err
	}
	err = f.Close()
	if err != nil {
		os.Remove(filename)
		return err
	}
	return nil
}

// Remove pid file.
func removePidFile(filename string) (err error) {
	err = os.Remove(filename)
	return err
}

// eof
