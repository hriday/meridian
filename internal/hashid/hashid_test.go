package hashid

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func names(cs []Candidate) []string {
	var n []string
	for _, c := range cs {
		n = append(n, c.Name)
	}
	return n
}
func modeOf(cs []Candidate, name string) int {
	for _, c := range cs {
		if c.Name == name {
			return c.Mode
		}
	}
	return -2
}

func TestIdentifyNTLMvsMD5(t *testing.T) {
	cs := Identify("32ed87bdb5fdc5e9cba88547376818d4") // 32 hex
	require.Equal(t, "NTLM", cs[0].Name, "NTLM ranked first for a 32-hex hash")
	require.Equal(t, 1000, cs[0].Mode)
	require.Contains(t, names(cs), "MD5")
	require.Equal(t, 0, modeOf(cs, "MD5"))
}

func TestIdentifyByLength(t *testing.T) {
	require.Equal(t, 100, modeOf(Identify("da39a3ee5e6b4b0d3255bfef95601890afd80709"), "SHA-1"))      // 40
	require.Equal(t, 1400, modeOf(Identify("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"), "SHA-256")) // 64
	require.Equal(t, 1700, modeOf(Identify("cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce47d0d13c5d85f2b0ff8318d2877eec2f63b931bd47417a81a538327af927da3e"), "SHA-512")) // 128
}

func TestIdentifyByPrefix(t *testing.T) {
	require.Equal(t, 3200, Identify("$2b$12$abcdefghijklmnopqrstuv")[0].Mode)     // bcrypt
	require.Equal(t, 1800, Identify("$6$salt$hash")[0].Mode)                      // sha512crypt
	require.Equal(t, 500, Identify("$1$salt$hash")[0].Mode)                       // md5crypt
	require.Equal(t, 400, Identify("$P$Bxxxxxxxxxxxxxxxxxxxxxxxxxx")[0].Mode)     // phpass
	require.Equal(t, 111, Identify("{SSHA}base64stuffhere==")[0].Mode)           // LDAP SSHA
	require.Equal(t, 13100, Identify("$krb5tgs$23$*user*$abcd")[0].Mode)          // kerberoast
	require.Equal(t, 300, Identify("*2470C0C06DEE42FD1618BB99005ADCA2EC9D1E19")[0].Mode) // MySQL 4.1+
	require.Equal(t, 9200, Identify("$8$saltsalt$hashhash")[0].Mode)              // Cisco type 8
}

func TestIdentifyDumpLine(t *testing.T) {
	cs := Identify("Administrator:500:aad3b435b51404eeaad3b435b51404ee:32ed87bdb5fdc5e9cba88547376818d4:::")
	require.Contains(t, cs[0].Name, "pwdump")
	require.Equal(t, 1000, cs[0].Mode, "point at the NT half")
}

func TestIdentifyUnknownAndEmpty(t *testing.T) {
	require.Nil(t, Identify("   "))
	require.Equal(t, "Unknown", Identify("not a hash!")[0].Name)
	require.Equal(t, -1, Identify("not a hash!")[0].Mode)
}
