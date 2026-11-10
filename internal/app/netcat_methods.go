package app

import (
	"errors"
	"io"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/hriday/meridian/internal/netcat"
)

func (a *App) ncEmit(event string, data any) {
	if a.ctx != nil {
		wruntime.EventsEmit(a.ctx, event, data)
	}
}

// NetcatConnect opens a connection and streams received bytes as `netcat:recv`
// events. For a TLS connection it returns the peer's certificate summary.
func (a *App) NetcatConnect(network, addr string, useTLS, insecure bool) (netcat.CertInfo, error) {
	a.ncMu.Lock()
	defer a.ncMu.Unlock()
	a.ncCloseLocked()

	s, cert, err := netcat.Dial(a.lookupCtx(), netcat.Config{
		Network: network, Addr: addr, TLS: useTLS, Insecure: insecure,
	})
	if err != nil {
		return netcat.CertInfo{}, err
	}
	a.ncSession = s
	a.ncStart(s)
	if cert != nil {
		return *cert, nil
	}
	return netcat.CertInfo{}, nil
}

// NetcatListen binds a local port; the first peer becomes the active session.
// Returns the bound address.
func (a *App) NetcatListen(network, addr string) (string, error) {
	a.ncMu.Lock()
	defer a.ncMu.Unlock()
	a.ncCloseLocked()

	l, err := netcat.Listen(a.lookupCtx(), netcat.Config{Network: network, Addr: addr})
	if err != nil {
		return "", err
	}
	a.ncListener = l
	bound := l.Addr()
	go func() {
		s, err := l.Accept()
		if err != nil {
			a.ncEmit("netcat:status", map[string]any{"state": "closed", "err": err.Error()})
			return
		}
		a.ncMu.Lock()
		if a.ncListener != l {
			// The user closed/replaced this listener while we were waiting;
			// don't adopt the late peer — close it.
			a.ncMu.Unlock()
			s.Close()
			return
		}
		a.ncSession = s
		a.ncMu.Unlock()
		a.ncStart(s)
	}()
	a.ncEmit("netcat:status", map[string]any{"state": "listening", "peer": bound})
	return bound, nil
}

// NetcatSend writes text to the active session.
func (a *App) NetcatSend(text string) error {
	a.ncMu.Lock()
	defer a.ncMu.Unlock()
	if a.ncSession == nil {
		return errors.New("not connected")
	}
	_, err := a.ncSession.Send([]byte(text))
	return err
}

// NetcatClose tears down the active session/listener.
func (a *App) NetcatClose() {
	a.ncMu.Lock()
	defer a.ncMu.Unlock()
	a.ncCloseLocked()
}

func (a *App) ncCloseLocked() {
	if a.ncSession != nil {
		a.ncSession.Close()
		a.ncSession = nil
	}
	if a.ncListener != nil {
		a.ncListener.Close()
		a.ncListener = nil
	}
}

func (a *App) ncStart(s *netcat.Session) {
	s.Start(
		func(b []byte) {
			if a.ncRecvHook != nil {
				a.ncRecvHook(b)
			}
			a.ncEmit("netcat:recv", map[string]any{"text": string(b), "hex": netcat.HexDump(b), "n": len(b)})
		},
		func(err error) {
			a.ncMu.Lock()
			if a.ncSession == s {
				a.ncSession = nil // drop the dead session so Send reports "not connected"
			}
			a.ncMu.Unlock()
			msg := ""
			if err != nil && err != io.EOF {
				msg = err.Error()
			}
			a.ncEmit("netcat:status", map[string]any{"state": "closed", "peer": s.Peer(), "err": msg})
		},
	)
	a.ncEmit("netcat:status", map[string]any{"state": "connected", "peer": s.Peer()})
}
