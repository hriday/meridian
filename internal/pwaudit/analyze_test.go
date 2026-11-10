package pwaudit

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClassify(t *testing.T) {
	cases := []struct {
		pw, pattern, strength string
	}{
		{"Summer2024!", "season+year", "weak"},
		{"qwerty", "keyboard walk", "weak"},
		{"12345678", "all digits", "weak"},
		{"Password1", "word+digit", "weak"},
		{"7hX!qp2@Lm9zRt4W", "none", "strong"}, // 16 chars, mixed, no pattern
		{"short", "none", "weak"},              // <8
	}
	for _, c := range cases {
		strength, pattern := Classify(c.pw)
		require.Equal(t, c.pattern, pattern, "pattern for %q", c.pw)
		require.Equal(t, c.strength, strength, "strength for %q", c.pw)
	}
}

func TestDetectReuse(t *testing.T) {
	const shared = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	results := []Result{
		{Account: Account{Name: "alice", NTHash: shared}, Cracked: true, Plaintext: "Welcome1"},
		{Account: Account{Name: "bob", NTHash: shared}, Cracked: true, Plaintext: "Welcome1"},
		{Account: Account{Name: "carol", NTHash: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}, Cracked: true, Plaintext: "unique"},
		{Account: Account{Name: "guest", NTHash: "31d6cfe0d16ae931b73c59d7e0c089c0"}},
	}
	groups := DetectReuse(results)
	require.Len(t, groups, 2) // the shared-hash pair + the empty-password account

	// reuse pair
	require.Equal(t, shared, groups[0].NTHash)
	require.ElementsMatch(t, []string{"alice", "bob"}, groups[0].Accounts)
	require.Equal(t, "Welcome1", groups[0].Plaintext)

	// empty-password account flagged even though it's alone
	require.Equal(t, "31d6cfe0d16ae931b73c59d7e0c089c0", groups[1].NTHash)
	require.Equal(t, "(empty password)", groups[1].Plaintext)
	require.Equal(t, []string{"guest"}, groups[1].Accounts)
}
