package tlsscan

import (
	"crypto/tls"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRemediateEmitsConcreteDirectives(t *testing.T) {
	// Legacy protocol + weak cipher present -> both nginx directives, plus reload.
	protos := []Protocol{
		{Name: "TLS 1.0", Version: tls.VersionTLS10, Legacy: true, Supported: true},
		{Name: "TLS 1.2", Version: tls.VersionTLS12, Supported: true},
	}
	ciphers := []Cipher{{Name: "TLS_RSA_WITH_AES_128_CBC_SHA", Secure: false}}
	out := strings.Join(remediate(protos, ciphers), "\n")
	require.Contains(t, out, "ssl_protocols TLSv1.2 TLSv1.3;")
	require.Contains(t, out, "ssl_ciphers ")
	require.Contains(t, out, "nginx -t")
}

func TestRemediateCleanConfigIsEmpty(t *testing.T) {
	protos := []Protocol{
		{Name: "TLS 1.2", Version: tls.VersionTLS12, Supported: true},
		{Name: "TLS 1.3", Version: tls.VersionTLS13, Supported: true},
	}
	ciphers := []Cipher{{Name: "TLS_AES_128_GCM_SHA256", Secure: true}}
	require.Empty(t, remediate(protos, ciphers))
}
