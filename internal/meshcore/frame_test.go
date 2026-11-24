package meshcore

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"
)

func framed(payload []byte) []byte {
	out := []byte{markerIn, 0, 0}
	binary.LittleEndian.PutUint16(out[1:], uint16(len(payload)))
	return append(out, payload...)
}

func TestWriteFrame(t *testing.T) {
	var buf bytes.Buffer
	if err := writeFrame(&buf, []byte{cmdAppStart, 1}); err != nil {
		t.Fatal(err)
	}
	got := buf.Bytes()
	want := []byte{markerOut, 0x02, 0x00, cmdAppStart, 1}
	if !bytes.Equal(got, want) {
		t.Errorf("frame = % x, want % x", got, want)
	}
}

func TestFrameReader_ReadsFrame(t *testing.T) {
	stream := framed([]byte{respSelfInfo, 0xAA})
	fr := newFrameReader(bytes.NewReader(stream), time.Time{})
	p, err := fr.next()
	if err != nil {
		t.Fatal(err)
	}
	if len(p) != 2 || p[0] != respSelfInfo || p[1] != 0xAA {
		t.Errorf("payload = % x", p)
	}
}

// TestFrameReader_ResyncsPastDebugText is the one that matters for wadamesh: the
// firmware prints plain-text logs on the same port, so the reader must skip them
// and still find the real frame.
func TestFrameReader_ResyncsPastDebugText(t *testing.T) {
	var stream []byte
	stream = append(stream, []byte("[STALL] ui:timers 710ms\n")...)  // debug noise
	stream = append(stream, framed([]byte{respDeviceInfo, 0x0a})...) // a real frame
	fr := newFrameReader(bytes.NewReader(stream), time.Time{})
	p, err := fr.next()
	if err != nil {
		t.Fatal(err)
	}
	if p[0] != respDeviceInfo {
		t.Errorf("should have resynced to the device-info frame, got % x", p)
	}
}

// A stray '>' (0x3E) inside text, followed by an implausible length, must not be
// mistaken for a frame — the reader resyncs and finds the next real one.
func TestFrameReader_FalseMarkerResync(t *testing.T) {
	var stream []byte
	// '>' then length 0xFFFF (65535) > maxFrame → false marker, skipped.
	stream = append(stream, markerIn, 0xff, 0xff)
	stream = append(stream, framed([]byte{respSelfInfo, 0x01})...)
	fr := newFrameReader(bytes.NewReader(stream), time.Time{})
	p, err := fr.next()
	if err != nil {
		t.Fatal(err)
	}
	if p[0] != respSelfInfo {
		t.Errorf("false marker should be skipped, got % x", p)
	}
}
