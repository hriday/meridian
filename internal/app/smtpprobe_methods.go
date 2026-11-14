package app

import (
	"context"
	"net"
	"time"

	"github.com/hriday/meridian/internal/smtpprobe"
)

// SMTPProbe opens an SMTP conversation with a mail server and reports its banner,
// EHLO capabilities, STARTTLS support, and (after upgrading, or on an implicit-
// TLS port) its certificate. With useMX, host is treated as a domain and its top
// MX is probed. It never authenticates or sends mail.
func (a *App) SMTPProbe(host, port string, implicitTLS, useMX bool) (smtpprobe.Result, error) {
	ctx, cancel := context.WithTimeout(a.lookupCtx(), 20*time.Second)
	defer cancel()

	target := host
	if useMX {
		mx, err := smtpprobe.TopMX(ctx, host)
		if err != nil {
			return smtpprobe.Result{}, err
		}
		target = mx
	}
	if port == "" {
		port = "25"
		if implicitTLS {
			port = "465"
		}
	}
	d := &net.Dialer{}
	return smtpprobe.Probe(ctx, d.DialContext, target, port, "", implicitTLS)
}
