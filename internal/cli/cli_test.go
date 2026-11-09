package cli_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/hriday/engineers_tools/internal/cli"
	"github.com/hriday/engineers_tools/internal/subnet"
	"github.com/stretchr/testify/require"
)

func TestRunSubnetJSON(t *testing.T) {
	var out, errb bytes.Buffer
	code := cli.Run([]string{"subnet", "10.0.0.0/24", "--json"}, &out, &errb)
	require.Equal(t, 0, code)
	var info subnet.Info
	require.NoError(t, json.Unmarshal(out.Bytes(), &info))
	require.Equal(t, "10.0.0.255", info.Broadcast)
}

func TestRunClassifyText(t *testing.T) {
	var out, errb bytes.Buffer
	code := cli.Run([]string{"classify", "AS13335"}, &out, &errb)
	require.Equal(t, 0, code)
	require.Contains(t, out.String(), "ASN")
	require.Contains(t, out.String(), "AS13335")
}

func TestRunBadArgs(t *testing.T) {
	var out, errb bytes.Buffer
	code := cli.Run([]string{"subnet"}, &out, &errb) // missing arg
	require.NotEqual(t, 0, code)
	require.NotEmpty(t, errb.String())
}
