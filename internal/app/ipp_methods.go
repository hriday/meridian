package app

import (
	"context"
	"net/http"
	"time"

	"github.com/hriday/meridian/internal/ipp"
)

// PrinterStatus probes a networked printer over IPP (Get-Printer-Attributes) and
// checks the common print ports. Read-only — it never sends a print job.
func (a *App) PrinterStatus(host string) (ipp.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 14*time.Second)
	defer cancel()
	return ipp.Probe(ctx, &http.Client{Timeout: 8 * time.Second}, host)
}
