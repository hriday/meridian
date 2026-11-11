package tfscan

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

const hclFixture = `variable "allowed" { type = list(string) }

resource "aws_security_group" "web" {
  name = "web-sg"
  ingress {
    description = "ssh from the world"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = var.allowed
  }
}

resource "aws_db_instance" "main" {
  identifier          = "main-db"
  publicly_accessible = true
  storage_encrypted   = false
  username            = "admin"
  password            = "SUPERSECRETpw123"
}

resource "aws_s3_bucket_acl" "assets" {
  bucket = aws_s3_bucket.assets.id
  acl    = "public-read"
}`

func mustParseHCL(t *testing.T, s string) []rawResource {
	t.Helper()
	rs, err := parseHCL(s)
	require.NoError(t, err)
	return rs
}

func TestHCLParsesResources(t *testing.T) {
	rs := mustParseHCL(t, hclFixture)
	require.Len(t, rs, 3) // the variable block is not a resource
	types := map[string]bool{}
	for _, r := range rs {
		types[r.typ] = true
		require.NotZero(t, r.line)
	}
	require.True(t, types["aws_security_group"])
	require.True(t, types["aws_db_instance"])
	require.True(t, types["aws_s3_bucket_acl"])
}

func TestHCLLiteralAndDynamic(t *testing.T) {
	a := check(mustParseHCL(t, hclFixture))
	require.NotEmpty(t, findingsWith(a, "22")) // literal 0.0.0.0/0:22 flagged
	// the var.allowed ingress raised no false finding, but marked the SG Dynamic
	for _, r := range a.Resources {
		if r.Type == "aws_security_group" {
			require.True(t, r.Dynamic)
		}
	}
	// only one exposure edge from the SG (the literal :22), not the var one
	sgEdges := 0
	for _, e := range a.Exposure {
		if e.Target == "aws_security_group.web" {
			sgEdges++
		}
	}
	require.Equal(t, 1, sgEdges)
}

func TestHCLEndToEndFindings(t *testing.T) {
	a := check(mustParseHCL(t, hclFixture))
	require.NotEmpty(t, findingsWith(a, "publicly accessible"))
	require.NotEmpty(t, findingsWith(a, "public-read"))
	require.NotEmpty(t, findingsWith(a, "22"))
	crit := 0
	for _, f := range a.Findings {
		if f.Severity == "critical" {
			crit++
		}
	}
	require.GreaterOrEqual(t, crit, 3)
}

func TestHCLSecretNotSurfaced(t *testing.T) {
	a := check(mustParseHCL(t, hclFixture))
	blob, err := json.Marshal(a)
	require.NoError(t, err)
	require.NotContains(t, string(blob), "SUPERSECRETpw123")
}

func TestHCLMalformed(t *testing.T) {
	_, err := parseHCL(`resource "aws_x" "y" {`) // unclosed brace
	require.Error(t, err)
}
