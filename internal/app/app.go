// Package app holds the Wails controller that binds Meridian's engine
// packages to the frontend. It contains no engine logic itself — each
// bound method delegates to a pure package under internal/.
package app

import (
	"context"
	"net/http"
	"net/netip"
	"sync"
	"time"

	"github.com/hriday/meridian/internal/collector"
	"github.com/hriday/meridian/internal/ipspace"
	"github.com/hriday/meridian/internal/lookup"
	"github.com/hriday/meridian/internal/netcat"
	"github.com/hriday/meridian/internal/netflow"
	"github.com/hriday/meridian/internal/pwaudit"
	"github.com/hriday/meridian/internal/scan"
	"github.com/hriday/meridian/internal/snmp"
	"github.com/hriday/meridian/internal/tftp"
)

// collectorBuffer is the rolling message cap for the syslog/trap collectors;
// netflowBuffer is larger since flow records are small and feed a windowed view.
const (
	collectorBuffer = 10000
	netflowBuffer   = 65536
)

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
	// test seam: build a sync backend from config; nil uses the real backends.
	ipMakeBackend func(cfg syncConfig) (ipspace.Backend, bool)
	// test seam: probe an SFTP server for its host key; nil uses the real dialer.
	ipProbeSFTP func(ctx context.Context, cfg ipspace.SFTPConfig) (string, error)
	// decrypted sync config, cached so IPSpaceSyncInfo doesn't re-run the KDF on
	// every refresh; nil+loaded means "not configured".
	ipSyncCfg       *syncConfig
	ipSyncCfgLoaded bool

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

	syslog  *collector.Collector[collector.SyslogMsg]
	traps   *collector.Collector[snmp.TrapMsg]
	netflow *collector.Collector[netflow.FlowRecord]
	tftpSrv *tftp.Server
}

// New returns a controller with no context; the context arrives at startup.
func New() *App {
	nfParser := netflow.NewParser() // stateful: per-exporter v9/IPFIX template cache
	return &App{
		lookup:      lookup.NewClient(&http.Client{Timeout: 10 * time.Second}, 10*time.Minute),
		ipPath:      ipspace.DefaultPath(),
		adCracker:   pwaudit.NewHashcatCracker(),
		wordlistDir: defaultWordlistDir(),
		newPinger:   func() scan.Pinger { return scan.NewICMPPinger() },
		syslog:      collector.New[collector.SyslogMsg](collectorBuffer, collector.ParseSyslog),
		traps:       collector.New[snmp.TrapMsg](collectorBuffer, snmp.ParseTrap),
		netflow: collector.NewMulti[netflow.FlowRecord](netflowBuffer,
			func(b []byte, src netip.Addr) []netflow.FlowRecord { return nfParser.Parse(b, src, time.Now()) }),
		tftpSrv: tftp.New(),
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
	return func(context.Context) {
		a.stopAllMonitors()
		a.syslog.Stop()
		a.traps.Stop()
		a.netflow.Stop()
		a.tftpSrv.Stop()
	}
}

// lookupCtx is the context for outbound lookups: the Wails startup context
// once set, else a background context (e.g. in tests).
func (a *App) lookupCtx() context.Context {
	if a.ctx != nil {
		return a.ctx
	}
	return context.Background()
}
