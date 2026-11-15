package app

import (
	"context"
	"net/http"
	"time"

	"github.com/hriday/meridian/internal/arpscan"
	"github.com/hriday/meridian/internal/lanrecon"
)

// ReconcileLAN sweeps the local subnet with ARP and diffs the live hosts against
// the (unlocked) IP-space inventory — surfacing untracked hosts, MAC changes,
// and active records that didn't answer. It requires an unlocked store. The
// returned findings are passed back to ApplyReconcileLAN to write changes.
func (a *App) ReconcileLAN() (lanrecon.Report, error) {
	a.ipMu.Lock()
	if err := a.requireUnlocked(); err != nil {
		a.ipMu.Unlock()
		return lanrecon.Report{}, err
	}
	records := a.ipStore.Records()
	a.ipMu.Unlock() // don't hold the inventory lock during the ~25s sweep

	ctx, cancel := context.WithTimeout(a.lookupCtx(), 30*time.Second)
	defer cancel()
	res, err := arpscan.Run(ctx, arpscan.DefaultWindow, arpscan.DefaultMaxHosts)
	if err != nil {
		return lanrecon.Report{}, err
	}
	arpscan.LookupVendors(ctx, &http.Client{Timeout: 8 * time.Second}, res.Hosts)

	return lanrecon.Reconcile(res, records), nil
}

// ApplyReconcileLAN writes the chosen reconciliation findings (as returned by
// ReconcileLAN) back to the inventory and saves it.
func (a *App) ApplyReconcileLAN(findings []lanrecon.Finding, importNew, updateMAC, deprecateMissing bool) (lanrecon.ApplyResult, error) {
	a.ipMu.Lock()
	defer a.ipMu.Unlock()
	if err := a.requireUnlocked(); err != nil {
		return lanrecon.ApplyResult{}, err
	}
	res, err := lanrecon.Apply(a.ipStore, findings, lanrecon.ApplyOptions{
		ImportNew:        importNew,
		UpdateMAC:        updateMAC,
		DeprecateMissing: deprecateMissing,
	})
	if err != nil {
		return res, err
	}
	if err := a.ipSave(); err != nil {
		return res, err
	}
	return res, nil
}
