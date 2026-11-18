package app

import (
	"context"
	"net/http"
	"time"

	"github.com/hriday/meridian/internal/macvendor"
	"github.com/hriday/meridian/internal/wwn"
)

// WWNDecode decodes a Fibre Channel World Wide Name (WWN / WWPN / WWNN): NAA
// format, embedded IEEE OUI, re-formatted spellings, and the OUI's vendor via
// the shared keyless OUI lookup. Offline except the optional vendor lookup.
func (a *App) WWNDecode(input string) (wwn.Result, error) {
	r, err := wwn.Parse(input)
	if err != nil {
		return wwn.Result{}, err
	}
	if r.OUIBare != "" && r.OUIBare != "000000" {
		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		defer cancel()
		if mv, mErr := macvendor.Lookup(ctx, &http.Client{Timeout: 5 * time.Second}, r.OUIBare); mErr == nil && mv.Found {
			r.Vendor = mv.Vendor
		}
	}
	return r, nil
}
