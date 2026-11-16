package tfscan

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFixForCoversEveryFindingTitle(t *testing.T) {
	// Every title emitted by the add() calls in checks.go must map to a fix.
	titles := []string{
		"Database is publicly accessible",
		"Database storage encryption not confirmed",
		"S3 bucket is public",
		"S3 public access block is weakened",
		"Instance gets a public IP",
		"EBS volume is not encrypted",
		"Firewall open to the internet",
		"Firewall exposes a sensitive port",
		"Storage bucket is public",
		"Cloud SQL open to the internet",
		"Instance gets an external IP",
		"NSG rule open to the internet",
		"NSG rule exposes a sensitive port",
		"Storage account permits public blobs",
		"Storage container is public",
		"Managed database allows public network access",
		"Database firewall allows the whole internet",
		"Public IP allocated",
	}
	for _, title := range titles {
		require.NotEmptyf(t, fixFor(title), "no fix mapped for finding title %q", title)
	}
	require.Empty(t, fixFor("Some unknown title"))
}
