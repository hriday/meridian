package tfscan

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// helpers to build rawResources with mapBody
func sg(name string, ingress ...map[string]any) rawResource {
	blocks := make([]any, len(ingress))
	for i, b := range ingress {
		blocks[i] = b
	}
	return rawResource{typ: "aws_security_group", name: name, body: mapBody{"ingress": blocks}}
}
func ingress(from, to int, proto string, cidrs ...string) map[string]any {
	c := make([]any, len(cidrs))
	for i, s := range cidrs {
		c[i] = s
	}
	return map[string]any{"from_port": float64(from), "to_port": float64(to), "protocol": proto, "cidr_blocks": c}
}

func findingsWith(a Analysis, sub string) []Finding {
	var out []Finding
	for _, f := range a.Findings {
		if contains(f.Title+" "+f.Detail, sub) {
			out = append(out, f)
		}
	}
	return out
}
func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestCheckOpenSSH(t *testing.T) {
	a := check([]rawResource{sg("web", ingress(22, 22, "tcp", "0.0.0.0/0"))})
	f := findingsWith(a, "22")
	require.NotEmpty(t, f)
	require.Equal(t, "critical", f[0].Severity)
	require.Equal(t, "exposure", f[0].Category)
	require.Len(t, a.Exposure, 1)
	require.Equal(t, "aws_security_group.web", a.Exposure[0].Target)
}

func TestCheckAllPortsCritical(t *testing.T) {
	a := check([]rawResource{sg("open", ingress(0, 0, "-1", "0.0.0.0/0"))})
	require.NotEmpty(t, a.Exposure)
	require.Equal(t, "critical", a.Exposure[0].Severity)
}

func TestCheckWebPortsInfo(t *testing.T) {
	a := check([]rawResource{sg("web", ingress(443, 443, "tcp", "0.0.0.0/0"))})
	require.NotEmpty(t, a.Exposure)
	require.Equal(t, "info", a.Exposure[0].Severity)
	for _, f := range a.Findings {
		require.NotEqual(t, "critical", f.Severity) // web port open is info, not critical
	}
}

func TestCheckEgressAndSgRefSkipped(t *testing.T) {
	sgRef := rawResource{typ: "aws_security_group", name: "x", body: mapBody{"ingress": []any{
		map[string]any{"from_port": float64(22), "to_port": float64(22), "security_groups": []any{"sg-123"}},
	}}}
	egress := rawResource{typ: "aws_security_group_rule", name: "out", body: mapBody{
		"type": "egress", "from_port": float64(0), "to_port": float64(0), "protocol": "-1", "cidr_blocks": []any{"0.0.0.0/0"},
	}}
	a := check([]rawResource{sgRef, egress})
	require.Empty(t, a.Exposure)
}

func TestCheckPublicRDSAndBucket(t *testing.T) {
	db := rawResource{typ: "aws_db_instance", name: "main", body: mapBody{"publicly_accessible": true, "storage_encrypted": true}}
	bucket := rawResource{typ: "aws_s3_bucket_acl", name: "assets", body: mapBody{"acl": "public-read"}}
	a := check([]rawResource{db, bucket})
	require.Len(t, a.Exposure, 2)
	require.NotEmpty(t, findingsWith(a, "publicly accessible"))
	require.NotEmpty(t, findingsWith(a, "public-read"))
	for _, f := range a.Findings {
		require.Equal(t, "critical", f.Severity)
	}
}

func TestCheckUnencrypted(t *testing.T) {
	db := rawResource{typ: "aws_db_instance", name: "main", body: mapBody{"storage_encrypted": false}}
	vol := rawResource{typ: "aws_ebs_volume", name: "data", body: mapBody{"encrypted": false}}
	a := check([]rawResource{db, vol})
	enc := findingsWith(a, "encrypt")
	require.Len(t, enc, 2)
	for _, f := range enc {
		require.Equal(t, "warning", f.Severity)
		require.Equal(t, "encryption", f.Category)
	}
	require.Empty(t, a.Exposure) // encryption is not internet exposure
}

// A partially-dynamic cidr list (resolved=false but a literal 0.0.0.0/0 present)
// must still flag the open rule and mark the resource Dynamic.
type dynSG struct{ cidrs []string }

func (d dynSG) str(n string) (string, bool) { return "", false }
func (d dynSG) list(n string) ([]string, bool) {
	if n == "cidr_blocks" {
		return d.cidrs, false // resolved=false
	}
	return nil, false
}
func (d dynSG) blocks(n string) []rawBody {
	if n == "ingress" {
		return []rawBody{ingressBody{d.cidrs}}
	}
	return nil
}

type ingressBody struct{ cidrs []string }

func (b ingressBody) str(n string) (string, bool) {
	if n == "from_port" {
		return "22", true
	}
	if n == "to_port" {
		return "22", true
	}
	if n == "protocol" {
		return "tcp", true
	}
	return "", false
}
func (b ingressBody) list(n string) ([]string, bool) {
	if n == "cidr_blocks" {
		return b.cidrs, false
	}
	return nil, false
}
func (b ingressBody) blocks(n string) []rawBody { return nil }

func TestCheckPartialDynamicListFlagged(t *testing.T) {
	r := rawResource{typ: "aws_security_group", name: "web", dynamic: false, body: dynSG{cidrs: []string{"0.0.0.0/0"}}}
	a := check([]rawResource{r})
	require.NotEmpty(t, a.Exposure)
	require.Equal(t, "critical", a.Exposure[0].Severity)
	require.True(t, a.Resources[0].Dynamic) // marked dynamic from the unresolved list
}

func TestCheckCountsAndSort(t *testing.T) {
	a := check([]rawResource{
		sg("web", ingress(22, 22, "tcp", "0.0.0.0/0")),
		{typ: "aws_db_instance", name: "main", body: mapBody{"storage_encrypted": false}},
		{typ: "aws_vpc", name: "main", body: mapBody{}},
	})
	require.Equal(t, 3, a.Counts.Resources)
	require.Equal(t, 1, a.Counts.SecurityGroups)
	require.GreaterOrEqual(t, a.Counts.Exposed, 1)
	require.GreaterOrEqual(t, a.Counts.Critical, 1)
	// sorted: critical before warning
	require.Equal(t, "critical", a.Findings[0].Severity)
}

func TestCheckRDSEncryptionAbsentFlagged(t *testing.T) {
	db := rawResource{typ: "aws_db_instance", name: "m", body: mapBody{"publicly_accessible": false}}
	a := check([]rawResource{db})
	require.NotEmpty(t, findingsWith(a, "encrypt")) // absent → unencrypted by default
}

func TestCheckRDSEncryptedTrueNotFlagged(t *testing.T) {
	db := rawResource{typ: "aws_db_instance", name: "m", body: mapBody{"storage_encrypted": true}}
	a := check([]rawResource{db})
	require.Empty(t, findingsWith(a, "encrypt"))
}

func TestCheckClusterInstanceNotEncryptionFlagged(t *testing.T) {
	// a cluster instance inherits encryption from the cluster — don't flag it.
	ci := rawResource{typ: "aws_rds_cluster_instance", name: "i", body: mapBody{}}
	a := check([]rawResource{ci})
	require.Empty(t, findingsWith(a, "encrypt"))
}

func TestCheckSGCountExcludesRules(t *testing.T) {
	a := check([]rawResource{
		sg("web", ingress(22, 22, "tcp", "0.0.0.0/0")),
		{typ: "aws_security_group_rule", name: "r1", body: mapBody{"type": "egress", "cidr_blocks": []any{"0.0.0.0/0"}}},
	})
	require.Equal(t, 1, a.Counts.SecurityGroups)
}

func TestCheckFromPortAbsentNotAllPorts(t *testing.T) {
	r := rawResource{typ: "aws_security_group", name: "x", body: mapBody{"ingress": []any{
		map[string]any{"to_port": float64(65535), "protocol": "tcp", "cidr_blocks": []any{"0.0.0.0/0"}},
	}}}
	a := check([]rawResource{r})
	for _, e := range a.Exposure {
		require.NotEqual(t, "all ports", e.Label) // missing from_port must not read as all-ports
	}
}

func TestCheckNoSecretsSurfaced(t *testing.T) {
	db := rawResource{typ: "aws_db_instance", name: "main", body: mapBody{
		"publicly_accessible": true, "password": "SUPERSECRETpw", "username": "admin",
	}}
	a := check([]rawResource{db})
	blob, err := json.Marshal(a)
	require.NoError(t, err)
	require.NotContains(t, string(blob), "SUPERSECRETpw")
}
