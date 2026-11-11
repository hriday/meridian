package tfscan

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAnalyzeAutoDetect(t *testing.T) {
	aj, err := Analyze(planFixture, "")
	require.NoError(t, err)
	require.Equal(t, "json", aj.Source)

	ah, err := Analyze(hclFixture, "")
	require.NoError(t, err)
	require.Equal(t, "hcl", ah.Source)
	require.NotEmpty(t, ah.Resources)
}

func TestAnalyzeEmpty(t *testing.T) {
	_, err := Analyze("   \n ", "")
	require.Error(t, err)
}

func TestAnalyzeExplicitSourceAndDynamicNote(t *testing.T) {
	a, err := Analyze(hclFixture, "hcl")
	require.NoError(t, err)
	require.NotEmpty(t, a.Resources)
	// the fixture has a var-based cidr → a dynamic note is added
	joined := ""
	for _, n := range a.Notes {
		joined += n
	}
	require.Contains(t, joined, "plan")
}

func TestAnalyzeUnknownSource(t *testing.T) {
	_, err := Analyze(hclFixture, "yaml")
	require.Error(t, err)
}
