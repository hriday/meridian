// Package app holds the Wails controller that binds Meridian's engine
// packages to the frontend. It contains no engine logic itself — each
// bound method delegates to a pure package under internal/.
package app

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/hriday/meridian/internal/collector"
	"github.com/hriday/meridian/internal/ipspace"
	"github.com/hriday/meridian/internal/lookup"
	"github.com/hriday/meridian/internal/netcat"
	"github.com/hriday/meridian/internal/pwaudit"
	"github.com/hriday/meridian/internal/scan"
	"github.com/hriday/meridian/internal/snmp"
)

// collectorBuffer is the rolling message cap for each collector (in-memory).
const collectorBuffer = 10000

// App is the Wails-bound controller. Its public methods are exposed to the
// frontend; unexported fields and methods are not, which is why the context
// accessor below is kept out of the public surface.
type App struct {
	ctx    context.Context
	lookup *lookup.Client

	traceMu     sync.Mutex
	traceCancel context.CancelFunc
	traceGen    int

	ipPath     string
	ipPass     string
	ipStore    *ipspace.Store
	ipUnlocked bool

	ncMu       sync.Mutex
	ncSession  *netcat.Session
	ncListener *netcat.Listener
	ncRecvHook func([]byte) // test observation hook; nil in production

	adCracker   pwaudit.Cracker
	adReport    *pwaudit.AuditReport // last run, held for an encrypted save
	wordlistDir string

	scanMu     sync.Mutex
	scanCancel context.CancelFunc
	newPinger  func() scan.Pinger

	monMu    sync.Mutex
	monitors map[string]*monitor

	syslog *collector.Collector[collector.SyslogMsg]
	traps  *collector.Collector[snmp.TrapMsg]
}

// New returns a controller with no context; the context arrives at startup.
func New() *App {
	return &App{
		lookup:      lookup.NewClient(&http.Client{Timeout: 10 * time.Second}, 10*time.Minute),
		ipPath:      ipspace.DefaultPath(),
		adCracker:   pwaudit.NewHashcatCracker(),
		wordlistDir: defaultWordlistDir(),
		newPinger:   func() scan.Pinger { return scan.NewICMPPinger() },
		syslog:      collector.New[collector.SyslogMsg](collectorBuffer, collector.ParseSyslog),
		traps:       collector.New[snmp.TrapMsg](collectorBuffer, snmp.ParseTrap),
	}
}

// OnStartup returns the Wails OnStartup callback for a. startup stays
// unexported so Wails does not bind it (or a context accessor) as a
// frontend-callable method.
func OnStartup(a *App) func(context.Context) {
	return a.startup
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// OnShutdown returns the Wails OnShutdown callback: stop every running monitor
// so no polling goroutine outlives the app.
func OnShutdown(a *App) func(context.Context) {
	return func(context.Context) { a.stopAllMonitors(); a.syslog.Stop(); a.traps.Stop() }
}

// lookupCtx is the context for outbound lookups: the Wails startup context
// once set, else a background context (e.g. in tests).
func (a *App) lookupCtx() context.Context {
	if a.ctx != nil {
		return a.ctx
	}
	return context.Background()
}
