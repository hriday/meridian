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
	progressCalls := 0
	c := HashcatCracker{
		LookPath: func(string) (string, error) { return "/usr/bin/hashcat", nil },
		Run: func(_ context.Context, _ string, args ...string) ([]byte, error) {
			switch {
			case argsContain(args, "--version"):
				return []byte("v6.2.6"), nil
			case argsContain(args, "--show"):
				return []byte("8846f7eaee8fb117ad06bdd830b7586c:Password1\n"), nil
			default: // the crack run streams status-json
				return []byte(`{"status":5,"recovered_hashes":[1,1],"progress":[100,100]}` + "\n"), nil
			}
		},
	}
	got, err := c.Crack(context.Background(),
		[]string{"8846f7eaee8fb117ad06bdd830b7586c"}, "/tmp/wl.txt", nil,
		func(Progress) { progressCalls++ })
	require.NoError(t, err)
	require.Equal(t, "Password1", got["8846f7eaee8fb117ad06bdd830b7586c"])
	require.Greater(t, progressCalls, 0)
}

func TestHashcatCrackNoBinary(t *testing.T) {
	c := HashcatCracker{LookPath: func(string) (string, error) { return "", errors.New("not found") }}
	_, err := c.Crack(context.Background(), []string{"8846f7eaee8fb117ad06bdd830b7586c"}, "/tmp/wl.txt", nil, nil)
	require.Error(t, err)
}
