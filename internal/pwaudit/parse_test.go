package pwaudit

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// A trimmed secretsdump/pwdump fixture: a normal user, a DOMAIN\user, a machine
// account, a comment, and a blank line.
const dumpFixture = `Administrator:500:aad3b435b51404eeaad3b435b51404ee:31d6cfe0d16ae931b73c59d7e0c089c0:::
CORP\jdoe:1104:aad3b435b51404eeaad3b435b51404ee:8846F7EAEE8FB117AD06BDD830B7586C:::
WS01$:1105:aad3b435b51404eeaad3b435b51404ee:0102030405060708090a0b0c0d0e0f10:::
# comment

`

func TestParseDump(t *testing.T) {
	accs, err := ParseDump(strings.NewReader(dumpFixture))
	require.NoError(t, err)
	require.Len(t, accs, 3)

	require.Equal(t, "Administrator", accs[0].Name)
	require.Equal(t, "31d6cfe0d16ae931b73c59d7e0c089c0", accs[0].NTHash)
	require.Equal(t, "500", accs[0].RID)
	require.False(t, accs[0].Machine)

	// DOMAIN\ prefix stripped; uppercase NT hash normalised to lowercase.
	require.Equal(t, "jdoe", accs[1].Name)
	require.Equal(t, "8846f7eaee8fb117ad06bdd830b7586c", accs[1].NTHash)

	// Machine account flagged.
	require.Equal(t, "WS01$", accs[2].Name)
	require.True(t, accs[2].Machine)
}

func TestParseRaw(t *testing.T) {
	accs, err := ParseRaw("8846f7eaee8fb117ad06bdd830b7586c\nalice:31d6cfe0d16ae931b73c59d7e0c089c0\nnotahash\n")
	require.NoError(t, err)
	require.Len(t, accs, 2) // the non-hex line is skipped

	require.Equal(t, "", accs[0].Name) // bare hash: no name
	require.Equal(t, "8846f7eaee8fb117ad06bdd830b7586c", accs[0].NTHash)

	require.Equal(t, "alice", accs[1].Name)
	require.Equal(t, "31d6cfe0d16ae931b73c59d7e0c089c0", accs[1].NTHash)
}
