package ipspace

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// A realistic NetBox prefixes export: status as a {value,label} object, tags as
// {name,slug} objects, description as the label.
const netboxYAML = `results:
  - prefix: 10.10.0.0/16
    status:
      value: active
      label: Active
    description: Corp core
    tags:
      - name: prod
        slug: prod
      - name: core
        slug: core
  - prefix: 10.20.0.0/16
    status:
      value: reserved
      label: Reserved
    description: Future DMZ
    tags: []
  - prefix: bad-cidr
    description: should skip
`

func TestImportNetBoxYAML(t *testing.T) {
	s := NewStore("a")
	rep, err := s.ImportNetBoxYAML(strings.NewReader(netboxYAML))
	require.NoError(t, err)
	require.Equal(t, 2, rep.Added)
	require.Equal(t, 1, rep.Skipped)

	byPrefix := map[string]Record{}
	for _, r := range s.Records() {
		byPrefix[r.Prefix] = r
	}
	require.Equal(t, "Corp core", byPrefix["10.10.0.0/16"].Name)
	require.Equal(t, Active, byPrefix["10.10.0.0/16"].Status)
	require.Equal(t, []string{"core", "prod"}, byPrefix["10.10.0.0/16"].Tags)
	require.Equal(t, Reserved, byPrefix["10.20.0.0/16"].Status)
}

func TestImportNetBoxBareList(t *testing.T) {
	// A bare list with string status + string tags also works.
	y := "- prefix: 192.168.0.0/24\n  status: deprecated\n  description: Legacy\n  tags: [legacy]\n"
	s := NewStore("a")
	rep, err := s.ImportNetBoxYAML(strings.NewReader(y))
	require.NoError(t, err)
	require.Equal(t, 1, rep.Added)
	require.Equal(t, Deprecated, s.Records()[0].Status)
	require.Equal(t, []string{"legacy"}, s.Records()[0].Tags)
}

func TestNetBoxRoundTrip(t *testing.T) {
	s := NewStore("a")
	_, _ = s.Add("10.0.0.0/8", "Corp", Active, []string{"prod"}, "")
	var buf bytes.Buffer
	require.NoError(t, s.ExportNetBoxYAML(&buf))

	s2 := NewStore("b")
	rep, err := s2.ImportNetBoxYAML(&buf)
	require.NoError(t, err)
	require.Equal(t, 1, rep.Added)
	require.Equal(t, "Corp", s2.Records()[0].Name)
	require.Equal(t, []string{"prod"}, s2.Records()[0].Tags)
}

// A realistic Terraform file with a VPC, two subnets, and a list of CIDRs.
const tfConfig = `resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
}

resource "aws_subnet" "public" {
  vpc_id     = aws_vpc.main.id
  cidr_block = "10.0.1.0/24"
}

resource "aws_security_group" "web" {
  ingress {
    cidr_blocks = ["10.0.0.0/16", "192.168.0.0/24"]
  }
}
`

func TestImportTerraform(t *testing.T) {
	s := NewStore("a")
	rep, err := s.ImportTerraform(strings.NewReader(tfConfig))
	require.NoError(t, err)
	// vpc /16, subnet /24, and the SG's 192.168.0.0/24 (the /16 dup is skipped).
	require.Equal(t, 3, rep.Added)
	require.Equal(t, 1, rep.Skipped)

	byPrefix := map[string]Record{}
	for _, r := range s.Records() {
		byPrefix[r.Prefix] = r
	}
	require.Equal(t, "aws_vpc.main", byPrefix["10.0.0.0/16"].Name)
	require.Equal(t, "aws_subnet.public", byPrefix["10.0.1.0/24"].Name)
	require.Contains(t, byPrefix, "192.168.0.0/24")
}

func TestExportTerraform(t *testing.T) {
	s := NewStore("a")
	_, _ = s.Add("10.0.0.0/8", "Corp", Active, nil, "")
	var buf bytes.Buffer
	require.NoError(t, s.ExportTerraform(&buf))
	out := buf.String()
	require.Contains(t, out, "locals {")
	require.Contains(t, out, `prefix = "10.0.0.0/8"`)
	require.Contains(t, out, `status = "active"`)
}
