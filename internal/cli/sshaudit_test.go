package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hriday/meridian/internal/sshaudit"
)

func TestPrintSSHAudit(t *testing.T) {
	var buf bytes.Buffer
	printSSHAudit(&buf, sshaudit.Report{
		Host:     "ssh.example.com:22",
		Software: "OpenSSH_9.9",
		Grade:    "A",
		Kex:      []string{"mlkem768x25519-sha256", "curve25519-sha256"},
		Ciphers:  []string{"chacha20-poly1305@openssh.com"},
		MACs:     []string{"hmac-sha2-256-etm@openssh.com"},
		HostKeys: []sshaudit.HostKey{{Type: "ssh-ed25519", Bits: 256, Fingerprint: "SHA256:abc123"}},
		PQ:       sshaudit.PQStatus{Supported: true, Preferred: true, Verdict: "Post-quantum key exchange is the server's default."},
	})
	out := buf.String()
	for _, want := range []string{"OpenSSH_9.9", "mlkem768x25519-sha256", "ssh-ed25519", "SHA256:abc123", "Post-quantum"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q\n%s", want, out)
		}
	}
}
