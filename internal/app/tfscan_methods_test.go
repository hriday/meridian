package app

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAnalyzeTerraformBinding(t *testing.T) {
	cfg := `resource "aws_security_group" "web" {
  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
}`
	a, err := New().AnalyzeTerraform(cfg, "")
	require.NoError(t, err)
	require.Equal(t, "hcl", a.Source)
	require.Len(t, a.Resources, 1)
	require.GreaterOrEqual(t, a.Counts.Critical, 1)
	require.NotEmpty(t, a.Exposure)
}

func TestAnalyzeTerraformEmpty(t *testing.T) {
	_, err := New().AnalyzeTerraform("", "")
	require.Error(t, err)
}
