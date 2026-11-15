package tfscan

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func critical(a Analysis) []Finding {
	var out []Finding
	for _, f := range a.Findings {
		if f.Severity == "critical" {
			out = append(out, f)
		}
	}
	return out
}

func hasFinding(a Analysis, sub string) bool {
	for _, f := range a.Findings {
		if strings.Contains(f.Title+" "+f.Detail, sub) {
			return true
		}
	}
	return false
}

func TestGCPExposure(t *testing.T) {
	hcl := `
resource "google_compute_firewall" "open" {
  direction     = "INGRESS"
  source_ranges = ["0.0.0.0/0"]
  allow {
    protocol = "tcp"
    ports    = ["22"]
  }
}
resource "google_storage_bucket_iam_member" "pub" {
  member = "allUsers"
  role   = "roles/storage.objectViewer"
}
resource "google_sql_database_instance" "db" {
  settings {
    ip_configuration {
      authorized_networks {
        value = "0.0.0.0/0"
      }
    }
  }
}
`
	a, err := Analyze(hcl, "hcl")
	require.NoError(t, err)
	require.True(t, hasFinding(a, "Firewall exposes a sensitive port"), "firewall SSH")
	require.True(t, hasFinding(a, "Storage bucket is public"), "public bucket")
	require.True(t, hasFinding(a, "Cloud SQL open to the internet"), "public sql")
	require.GreaterOrEqual(t, len(critical(a)), 3)
}

func TestAzureExposure(t *testing.T) {
	hcl := `
resource "azurerm_network_security_rule" "open" {
  access                 = "Allow"
  direction              = "Inbound"
  source_address_prefix  = "*"
  destination_port_range = "3389"
}
resource "azurerm_storage_container" "pub" {
  container_access_type = "container"
}
resource "azurerm_mssql_server" "db" {
  public_network_access_enabled = true
}
`
	a, err := Analyze(hcl, "hcl")
	require.NoError(t, err)
	require.True(t, hasFinding(a, "NSG rule exposes a sensitive port"), "NSG RDP")
	require.True(t, hasFinding(a, "Storage container is public"), "public container")
	require.True(t, hasFinding(a, "Managed database allows public network access"), "public db")
	require.GreaterOrEqual(t, len(critical(a)), 3)
}

func TestAzureNSGEgressIgnored(t *testing.T) {
	hcl := `
resource "azurerm_network_security_rule" "egress" {
  access                 = "Allow"
  direction              = "Outbound"
  source_address_prefix  = "*"
  destination_port_range = "22"
}
`
	a, err := Analyze(hcl, "hcl")
	require.NoError(t, err)
	require.False(t, hasFinding(a, "NSG"), "outbound rules must not be flagged")
}
