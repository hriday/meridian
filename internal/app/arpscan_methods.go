package app

import (
	"context"
	"net/http"
	"time"

	"github.com/hriday/meridian/internal/arpscan"
)

// ARPSupported reports whether ARP scanning works on this OS (macOS / Linux).
func (a *App) ARPSupported() bool { return arpscan.Supported() }

// ARPScan sweeps the local subnet with ARP requests and returns the live hosts
// (IP + MAC + best-effort vendor). It needs elevated privileges (raw L2);
// without them it returns an error saying so.
func (a *App) ARPScan() (arpscan.Result, error) {
	ctx, cancel := context.WithTimeout(a.lookupCtx(), 25*time.Second)
	defer cancel()
	res, err := arpscan.Run(ctx, arpscan.DefaultWindow, arpscan.DefaultMaxHosts)
	if err != nil {
		return res, err
	}
	arpscan.LookupVendors(ctx, &http.Client{Timeout: 8 * time.Second}, res.Hosts)
	return res, nil
}
