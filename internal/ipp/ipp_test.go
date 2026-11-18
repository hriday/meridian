package ipp

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestBuildRequestHeader pins the request's wire header against RFC 8010, so the
// decoder test below isn't just checking the encoder against itself.
func TestBuildRequestHeader(t *testing.T) {
	b := buildRequest("ipp://printer.local:631/ipp/print")
	require.GreaterOrEqual(t, len(b), 9)
	require.Equal(t, []byte{0x02, 0x00}, b[0:2], "version 2.0")
	require.Equal(t, uint16(0x000B), binary.BigEndian.Uint16(b[2:4]), "Get-Printer-Attributes")
	require.Equal(t, uint32(1), binary.BigEndian.Uint32(b[4:8]), "request-id")
	require.Equal(t, byte(0x01), b[8], "operation-attributes-tag")
	require.Equal(t, byte(0x03), b[len(b)-1], "end-of-attributes-tag")
}

// putAttr writes a TLV exactly as RFC 8010 §3.1.4 specifies (independent of the
// package's own writeAttr), so decode() is tested against the spec layout.
func putAttr(b *bytes.Buffer, tag byte, name string, val []byte) {
	b.WriteByte(tag)
	_ = binary.Write(b, binary.BigEndian, uint16(len(name)))
	b.WriteString(name)
	_ = binary.Write(b, binary.BigEndian, uint16(len(val)))
	b.Write(val)
}

func i32(n int32) []byte {
	out := make([]byte, 4)
	binary.BigEndian.PutUint32(out, uint32(n))
	return out
}

func sampleResponse() []byte {
	var b bytes.Buffer
	b.Write([]byte{0x02, 0x00})                              // version
	_ = binary.Write(&b, binary.BigEndian, uint16(0x0000))   // status successful-ok
	_ = binary.Write(&b, binary.BigEndian, uint32(1))        // request-id
	b.WriteByte(0x01)                                        // operation-attributes-tag
	putAttr(&b, 0x47, "attributes-charset", []byte("utf-8")) //
	putAttr(&b, 0x48, "attributes-natural-language", []byte("en"))
	b.WriteByte(0x04) // printer-attributes-tag
	putAttr(&b, 0x42, "printer-name", []byte("lobby"))
	putAttr(&b, 0x41, "printer-make-and-model", []byte("ACME LaserJet 9000"))
	putAttr(&b, 0x23, "printer-state", i32(3)) // enum: idle
	putAttr(&b, 0x44, "printer-state-reasons", []byte("media-empty"))
	putAttr(&b, 0x44, "", []byte("toner-low")) // additional value (empty name)
	putAttr(&b, 0x44, "", []byte("none"))      // "none" should be dropped
	putAttr(&b, 0x21, "queued-job-count", i32(2))
	putAttr(&b, 0x42, "marker-names", []byte("Black Toner"))
	putAttr(&b, 0x21, "marker-levels", i32(42))
	putAttr(&b, 0x44, "marker-colors", []byte("#000000"))
	b.WriteByte(0x03) // end-of-attributes
	return b.Bytes()
}

func TestDecodeAndMap(t *testing.T) {
	attrs, status, err := decode(sampleResponse())
	require.NoError(t, err)
	require.Equal(t, uint16(0), status, "successful-ok")

	var r Result
	mapAttrs(&r, attrs)
	require.Equal(t, "lobby", r.Name)
	require.Equal(t, "ACME LaserJet 9000", r.MakeModel)
	require.Equal(t, "idle", r.State)
	require.Equal(t, []string{"media-empty", "toner-low"}, r.StateReasons) // "none" dropped
	require.Equal(t, 2, r.QueuedJobs)
	require.Len(t, r.Supplies, 1)
	require.Equal(t, "Black Toner", r.Supplies[0].Name)
	require.Equal(t, 42, r.Supplies[0].Level)
	require.Equal(t, "#000000", r.Supplies[0].Color)
}

func TestSupplyColorFromName(t *testing.T) {
	// A real Samsung C460 reports garbage in marker-colors (black as #FFFFFF,
	// cyan as #FF5110, …), so the supply NAME is authoritative for the swatch.
	// The printer's reported colour is only a fallback when the name carries no
	// colour word.
	cases := []struct {
		name, reported, want string
	}{
		{"Black Toner_S/N_:CRUM-18040922935", "#FFFFFF", "#000000"},
		{"Cyan Toner_S/N_:CRUM-19051752210", "#FF5110", "#00AEEF"},
		{"Magenta Toner_S/N_:CRUM-19051750135", "#13FF73", "#EC008C"},
		{"Yellow Toner_S/N_:CRUM-19051750279", "#000DFF", "#FFF200"},
		{"Black Toner", "", "#000000"},         // name works with no reported value
		{"Photo Black", "garbage", "#000000"},  // substring match, invalid reported ignored
		{"Imaging Unit", "#3A5F9C", "#3A5F9C"}, // no colour word -> keep a valid reported hex
		{"Waste Toner Bottle", "", ""},         // no colour word, nothing to show
		{"Drum Kit", "none", ""},               // invalid reported -> empty, never faked
	}
	for _, c := range cases {
		if got := supplyColor(c.name, c.reported); got != c.want {
			t.Errorf("supplyColor(%q, %q) = %q, want %q", c.name, c.reported, got, c.want)
		}
	}
}

func TestDecodeRejectsShort(t *testing.T) {
	_, _, err := decode([]byte{0x02, 0x00, 0x00})
	require.Error(t, err)
}
