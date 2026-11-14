package app

import (
	"context"
	"time"

	"github.com/hriday/meridian/internal/sshaudit"
)

// SSHAudit inspects an SSH server's offered algorithms and host keys without
// authenticating, and grades them — with a focus on post-quantum key-exchange
// readiness (the defense against harvest-now-decrypt-later). It opens a few
// short connections: one to read the cleartext KEXINIT, and one per host-key
// type (each aborted the instant the key is received, so no credentials are
// ever sent).
func (a *App) SSHAudit(host string) (sshaudit.Report, error) {
	ctx, cancel := context.WithTimeout(a.lookupCtx(), 20*time.Second)
	defer cancel()
	return sshaudit.Probe(ctx, host)
}
