package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hriday/meridian/internal/rtp"
)

func TestPrintRTP(t *testing.T) {
	var buf bytes.Buffer
	printRTP(&buf, []rtp.Stream{{
		SSRC: 0xABCD1234, Src: "10.0.0.1:10000", Dst: "10.0.0.2:10002",
		PayloadType: 0, Codec: "G.711 PCMU", Packets: 500, DurationMs: 10000,
		Lost: 3, LossPct: 0.6, JitterMs: 2.4,
	}})
	out := buf.String()
	for _, want := range []string{"1 RTP stream", "0xABCD1234", "10.0.0.1:10000", "G.711 PCMU", "2.40 ms", "0.6%"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q\n%s", want, out)
		}
	}
}

func TestPrintRTPNone(t *testing.T) {
	var buf bytes.Buffer
	printRTP(&buf, nil)
	if !strings.Contains(buf.String(), "No RTP streams seen") {
		t.Errorf("empty should explain: %s", buf.String())
	}
}
