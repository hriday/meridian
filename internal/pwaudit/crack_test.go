package pwaudit

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func argsContain(args []string, s string) bool {
	for _, a := range args {
		if a == s {
			return true
		}
	}
	return false
}

func TestHashcatDetect(t *testing.T) {
	c := HashcatCracker{
		LookPath: func(string) (string, error) { return "/usr/bin/hashcat", nil },
		Run: func(_ context.Context, _ string, args ...string) ([]byte, error) {
			if argsContain(args, "--version") {
				return []byte("v6.2.6\n"), nil
			}
			return nil, nil
		},
	}
	info := c.Detect(context.Background())
	require.True(t, info.Available)
	require.Equal(t, "hashcat", info.Name)
	require.Contains(t, info.Version, "6.2.6")
	require.Equal(t, "/usr/bin/hashcat", info.Path)
}

func TestHashcatDetectMissing(t *testing.T) {
	c := HashcatCracker{LookPath: func(string) (string, error) { return "", errors.New("not found") }}
	require.False(t, c.Detect(context.Background()).Available)
}

func TestHashcatCrack(t *testing.T) {
	var progress []Progress
	c := HashcatCracker{
		LookPath: func(string) (string, error) { return "/usr/bin/hashcat", nil },
		Run: func(_ context.Context, _ string, args ...string) ([]byte, error) {
			if argsContain(args, "--show") {
				return []byte("8846f7eaee8fb117ad06bdd830b7586c:Password1\n"), nil
			}
			return []byte("v6.2.6"), nil // --version
		},
		// Stream fires status lines live, as the real process would.
		Stream: func(_ context.Context, _ string, _ []string, onLine func(string)) error {
			onLine(`{"status":3,"recovered_hashes":[0,1],"progress":[50,100]}`)
			onLine(`{"status":5,"recovered_hashes":[1,1],"progress":[100,100]}`)
			return nil
		},
	}
	got, err := c.Crack(context.Background(),
		[]string{"8846f7eaee8fb117ad06bdd830b7586c"}, "/tmp/wl.txt", nil,
		func(p Progress) { progress = append(progress, p) })
	require.NoError(t, err)
	require.Equal(t, "Password1", got["8846f7eaee8fb117ad06bdd830b7586c"])
	// progress is reported per streamed line, not in one post-hoc burst
	require.Len(t, progress, 2)
	require.InDelta(t, 50.0, progress[0].Percent, 0.1)
	require.Equal(t, 1, progress[1].Recovered)
}

func TestHashcatCrackNoBinary(t *testing.T) {
	c := HashcatCracker{LookPath: func(string) (string, error) { return "", errors.New("not found") }}
	_, err := c.Crack(context.Background(), []string{"8846f7eaee8fb117ad06bdd830b7586c"}, "/tmp/wl.txt", nil, nil)
	require.Error(t, err)
}

func TestParseShowColonInPassword(t *testing.T) {
	// a plaintext containing ':' must survive the first-colon split
	got := parseShow([]byte("8846f7eaee8fb117ad06bdd830b7586c:pa:ss:word\n"))
	require.Equal(t, "pa:ss:word", got["8846f7eaee8fb117ad06bdd830b7586c"])
}
