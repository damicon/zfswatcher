//
// webpagehandlers.go
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
	"fmt"
	auth "github.com/abbot/go-http-auth"
	"github.com/damicon/zfswatcher/notifier"
	"html/template"
	"net/http"
	"sync"
	"time"
)

var templates *template.Template

type severityToWwwClassMap map[notifier.Severity]string

// Implement fmt.Scanner interface.
func (scmapp *severityToWwwClassMap) Scan(state fmt.ScanState, verb rune) error {
	smap := stringToStringMap{}
	err := smap.Scan(state, verb)
	if err != nil {
		return err
	}
	scmap := make(severityToWwwClassMap)
	for a, b := range smap {
		var severity notifier.Severity
		if n, err := fmt.Sscan(a, &severity); n != 1 {
			return err
		}
		scmap[severity] = b
	}
	*scmapp = scmap
	return nil
}

type webNav struct {
	// active main menu entry
	Dashboard  bool
	PoolStatus bool
	Statistics bool
	Logs       bool
	About      bool
}

type webSubNav struct {
	Name   string
	Active bool
}

type webData struct {
	Nav    webNav
	SubNav []webSubNav
	Data   interface{}
}

type devStatusWeb struct {
	Indent     int
	Name       string
	EnableLed  bool
	Locate     bool
	State      string
	StateClass string
	Read       int64
	Write      int64
	Cksum      int64
	Rest       string
}

type poolStatusWeb struct {
	N            int
	Name         string
	State        string
	StateClass   string
	Status       string
	Action       string
	See          string
	Scan         string
	Devs         []devStatusWeb
	Errors       string
	Used         int64
	UsedPercent  int
	UsedClass    string
	Avail        int64
	AvailPercent int
	Total        int64
}

type dashboardWeb struct {
	SysUptime        string
	ZfswatcherUptime string
	SysLoadaverage   [3]float32
	Pools            []*poolStatusWeb
}

type logMsgWeb struct {
	Time       string
	Severity   string
	Class      string
	Text       string
	Attachment string
}

var (
	wwwLogBuffer []*logMsgWeb
	wwwLogMutex  sync.RWMutex
)

func wwwLogReceiver(m *notifier.Msg) {
	wwwLogMutex.Lock()
	if wwwLogBuffer == nil {
		wwwLogBuffer = make([]*logMsgWeb, 0, cfg.Www.Logbuffer+1)
	}
	switch m.MsgType {
	case notifier.MSGTYPE_MESSAGE:
		nm := &logMsgWeb{}
		nm.Time, nm.Severity, nm.Text = m.Strings()
		nm.Class = cfg.Www.Severitycssclassmap[m.Severity]
		wwwLogBuffer = append(wwwLogBuffer, nm)
	case notifier.MSGTYPE_ATTACHMENT:
		prev := len(wwwLogBuffer) - 1
		if len(wwwLogBuffer) > 0 && wwwLogBuffer[prev].Attachment == "" {
			// add the attachment to the previous message
			wwwLogBuffer[prev].Attachment = m.Text
		} else {
			// make a new entry only with the attachment
			nm := &logMsgWeb{}
			nm.Time, nm.Severity, nm.Attachment = m.Strings()
			nm.Class = cfg.Www.Severitycssclassmap[m.Severity]
			wwwLogBuffer = append(wwwLogBuffer, nm)
		}
	}
	if len(wwwLogBuffer) > cfg.Www.Logbuffer {
		wwwLogBuffer = wwwLogBuffer[len(wwwLogBuffer)-cfg.Www.Logbuffer:]
	}
	wwwLogMutex.Unlock()
}

func makePoolStatusWeb(pool *PoolType, usage map[string]*PoolUsageType) *poolStatusWeb {
	statusWeb := &poolStatusWeb{
		Name:       pool.name,
		State:      pool.state,
		StateClass: cfg.Www.Poolstatecssclassmap[pool.state],
		Status:     pool.status,
		Action:     pool.action,
		See:        pool.see,
		Scan:       pool.scan,
		Errors:     pool.errors,
	}
	statusWeb.Avail = -1
	statusWeb.Used = -1
	statusWeb.Total = -1
	if u, ok := usage[pool.name]; ok {
		statusWeb.Avail = u.Avail
		statusWeb.AvailPercent = u.GetAvailPercent()
		statusWeb.Used = u.Used
		usedPercent := u.GetUsedPercent()
		statusWeb.UsedPercent = usedPercent
		statusWeb.Total = u.Avail + u.Used
		usedSeverity, _ := cfg.Severity.Usedspace.GetByPercentage(usedPercent)
		statusWeb.UsedClass = cfg.Www.Usedstatecssclassmap[usedSeverity]
	}

	for n, dev := range pool.devs {
		devw := devStatusWeb{
			Name:       dev.name,
			State:      dev.state,
			StateClass: cfg.Www.Devstatecssclassmap[dev.state],
			Read:       dev.read,
			Write:      dev.write,
			Cksum:      dev.cksum,
			Rest:       dev.rest,
		}
		devw.Indent = 1
		for d := n; pool.devs[d].parentDev != -1; d = pool.devs[d].parentDev {
			devw.Indent += 2
		}

		if cfg.Leds.Enable && len(dev.subDevs) == 0 {
			loc, err := locateQuery(dev.name)
			if err == nil {
				fmt.Printf("Locate query %s success\n", dev.name)
				devw.EnableLed = true
				devw.Locate = loc
			}
		}
		statusWeb.Devs = append(statusWeb.Devs, devw)
	}
	return statusWeb
}

func statusHandler(w http.ResponseWriter, r *auth.AuthenticatedRequest) {
	wn := webNav{PoolStatus: true}

	pool := r.URL.Path[len("/status/"):]

	if !legalPoolName(pool) && !(pool == "") {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	currentState.mutex.RLock()
	state := currentState.state
	usage := currentState.usage
	currentState.mutex.RUnlock()

	if len(state) == 0 {
		err := templates.ExecuteTemplate(w, "status-none.html", &webData{Nav: wn})
		if err != nil {
			notify.Printf(notifier.ERR, "error executing template: %s", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	subnav := make([]webSubNav, 0, len(state))
	match := -1

	for n, s := range state {
		active := s.name == pool
		subnav = append(subnav, webSubNav{Name: s.name, Active: active})
		if active {
			match = n
		}
	}
	if pool == "" {
		http.Redirect(w, &r.Request, "/status/"+subnav[0].Name, http.StatusSeeOther)
		return
	}
	if match == -1 {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	ws := makePoolStatusWeb(state[match], usage)

	var err error

	switch {
	case len(subnav) > 1: // more than one pool
		err = templates.ExecuteTemplate(w, "status-many.html",
			&webData{Nav: wn, SubNav: subnav, Data: ws})
	case len(subnav) == 1: // a single pool
		err = templates.ExecuteTemplate(w, "status-single.html",
			&webData{Nav: wn, SubNav: subnav, Data: ws})
	}
	if err != nil {
		notify.Printf(notifier.ERR, "error executing template: %s", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func usageHandler(w http.ResponseWriter, r *auth.AuthenticatedRequest) {
	wn := webNav{} // not available in menu

	pool := r.URL.Path[len("/usage/"):]

	if pool == "" || !legalPoolName(pool) {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	zfsListOutput, err := getCommandOutput(cfg.Main.Zfslistusagecmd + " " + pool)
	if err != nil {
		notify.Print(notifier.ERR, "getting ZFS disk usage failed")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	usage := parseZfsList(zfsListOutput)
	if err != nil {
		notify.Print(notifier.ERR, "parsing ZFS disk usage failed")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	err = templates.ExecuteTemplate(w, "usage.html", &webData{Nav: wn, Data: usage})
	if err != nil {
		notify.Printf(notifier.ERR, "error executing template: %s", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func dashboardHandler(w http.ResponseWriter, r *auth.AuthenticatedRequest) {
	wn := webNav{Dashboard: true}

	uptime, err := getSystemUptime()
	if err != nil {
		notify.Printf(notifier.ERR, "can not get system uptime: %s", err)
	}
	loadavg, err := getSystemLoadaverage()
	if err != nil {
		notify.Printf(notifier.ERR, "can not get system load average: %s", err)
	}

	currentState.mutex.RLock()
	state := currentState.state
	usage := currentState.usage
	currentState.mutex.RUnlock()

	var ws []*poolStatusWeb

	for n, s := range state {
		ws = append(ws, makePoolStatusWeb(s, usage))
		ws[n].N = n
	}

	d := &dashboardWeb{
		SysUptime:        myDurationString(uptime),
		ZfswatcherUptime: myDurationString(time.Since(startTime)),
		SysLoadaverage:   loadavg,
		Pools:            ws,
	}

	err = templates.ExecuteTemplate(w, "dashboard.html", &webData{Nav: wn, Data: d})
	if err != nil {
		notify.Printf(notifier.ERR, "error executing template: %s", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func statisticsHandler(w http.ResponseWriter, r *auth.AuthenticatedRequest) {
	wn := webNav{Statistics: true}
	err := templates.ExecuteTemplate(w, "statistics.html", &webData{Nav: wn})
	if err != nil {
		notify.Printf(notifier.ERR, "error executing template: %s", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func logsHandler(w http.ResponseWriter, r *auth.AuthenticatedRequest) {
	wn := webNav{Logs: true}
	wwwLogMutex.RLock()
	err := templates.ExecuteTemplate(w, "logs.html", &webData{Nav: wn, Data: wwwLogBuffer})
	wwwLogMutex.RUnlock()
	if err != nil {
		notify.Printf(notifier.ERR, "error executing template: %s", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func aboutHandler(w http.ResponseWriter, r *auth.AuthenticatedRequest) {
	wn := webNav{About: true}
	err := templates.ExecuteTemplate(w, "about.html",
		&webData{Nav: wn,
			Data: map[string]string{
				"Version":       VERSION,
				"GoEnvironment": getGoEnvironment(),
			}})
	if err != nil {
		notify.Printf(notifier.ERR, "error executing template: %s", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func locateHandler(w http.ResponseWriter, r *auth.AuthenticatedRequest) {
	dev := r.FormValue("dev") // XXX validate, remove slashes etc
	state := r.FormValue("state")

	if _, err := locateQuery(dev); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	switch state {
	case "on":
		locateOn(dev)
	case "off":
		locateOff(dev)
	default:
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	http.Redirect(w, &r.Request, r.Referer(), http.StatusSeeOther)
}

// eof
