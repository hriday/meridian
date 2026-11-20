package meshtastic

import (
	"bytes"
	"io"
	"testing"
	"time"

	pb "buf.build/gen/go/meshtastic/protobufs/protocolbuffers/go/meshtastic"
	"google.golang.org/protobuf/proto"
)

type memConn struct{ *bytes.Buffer }

func (memConn) Close() error { return nil }

func completeFrame(id uint32) *pb.FromRadio {
	return (&pb.FromRadio_builder{ConfigCompleteId: proto.Uint32(id)}).Build()
}

func framedBytes(t *testing.T, m proto.Message) []byte {
	t.Helper()
	buf := &bytes.Buffer{}
	if err := newStreamConn(memConn{buf}).writeMsg(m); err != nil {
		t.Fatalf("writeMsg: %v", err)
	}
	return buf.Bytes()
}

func TestFrame_RoundTrip(t *testing.T) {
	buf := &bytes.Buffer{}
	c := newStreamConn(memConn{buf})
	if err := c.writeMsg(completeFrame(4242)); err != nil {
		t.Fatalf("writeMsg: %v", err)
	}
	var out pb.FromRadio
	if err := c.readMsg(&out, time.Now().Add(time.Second)); err != nil {
		t.Fatalf("readMsg: %v", err)
	}
	if got := out.GetConfigCompleteId(); got != 4242 {
		t.Fatalf("config_complete_id round trip: got %d, want 4242", got)
	}
}

func TestFrame_ResyncPastInterFrameNoise(t *testing.T) {
	// A node interleaves human-readable debug text between frames, and a lone
	// START1 byte can appear mid-noise. The reader must skip all of it and lock
	// onto the real frame.
	buf := &bytes.Buffer{}
	buf.WriteString("INFO  | Booted, hw HELTEC_V3\n")
	buf.WriteByte(start1)
	buf.WriteString(" DEBUG | building node db ")
	buf.Write(framedBytes(t, completeFrame(77)))

	var out pb.FromRadio
	if err := newStreamConn(memConn{buf}).readMsg(&out, time.Now().Add(time.Second)); err != nil {
		t.Fatalf("readMsg after noise: %v", err)
	}
	if got := out.GetConfigCompleteId(); got != 77 {
		t.Fatalf("resync: got %d, want 77", got)
	}
}

func TestFrame_DoubleStart1Resyncs(t *testing.T) {
	// START1 START1 START2 ... — the first START1 is not followed by START2, but
	// the second one is; the reader must treat the second as the frame start.
	buf := &bytes.Buffer{}
	buf.WriteByte(start1)
	buf.Write(framedBytes(t, completeFrame(33)))

	var out pb.FromRadio
	if err := newStreamConn(memConn{buf}).readMsg(&out, time.Now().Add(time.Second)); err != nil {
		t.Fatalf("readMsg: %v", err)
	}
	if got := out.GetConfigCompleteId(); got != 33 {
		t.Fatalf("got %d, want 33", got)
	}
}

func TestFrame_ImplausibleLengthResyncs(t *testing.T) {
	// START1 START2 then a 65535 length is noise, not a real frame; the reader
	// must discard the bogus header and find the next real frame.
	buf := &bytes.Buffer{}
	buf.Write([]byte{start1, start2, 0xff, 0xff})
	buf.Write(framedBytes(t, completeFrame(5)))

	var out pb.FromRadio
	if err := newStreamConn(memConn{buf}).readMsg(&out, time.Now().Add(time.Second)); err != nil {
		t.Fatalf("readMsg: %v", err)
	}
	if got := out.GetConfigCompleteId(); got != 5 {
		t.Fatalf("got %d, want 5", got)
	}
}

// tickReader returns (0, nil) a set number of times — exactly as a serial port
// does when its own read timeout fires with no byte available — before serving
// its data, then io.EOF. It proves readByte tolerates those ticks and still
// honours the overall deadline on a line that never produces a byte.
type tickReader struct {
	ticks int
	data  []byte
	pos   int
}

func (r *tickReader) Read(p []byte) (int, error) {
	if r.ticks > 0 {
		r.ticks--
		return 0, nil
	}
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	p[0] = r.data[r.pos]
	r.pos++
	return 1, nil
}
func (r *tickReader) Write(p []byte) (int, error) { return len(p), nil }
func (r *tickReader) Close() error                { return nil }

func TestReadByte_ToleratesSerialTimeoutTicks(t *testing.T) {
	c := newStreamConn(&tickReader{ticks: 5, data: framedBytes(t, completeFrame(9))})
	var out pb.FromRadio
	if err := c.readMsg(&out, time.Now().Add(time.Second)); err != nil {
		t.Fatalf("readMsg through ticks: %v", err)
	}
	if got := out.GetConfigCompleteId(); got != 9 {
		t.Fatalf("got %d, want 9", got)
	}
}

func TestReadByte_DeadlineStopsAQuietLine(t *testing.T) {
	c := newStreamConn(&tickReader{ticks: 1 << 30}) // ticks forever, never serves a byte
	var out pb.FromRadio
	err := c.readMsg(&out, time.Now().Add(40*time.Millisecond))
	if !isReadTimeout(err) {
		t.Fatalf("want read timeout on a silent line, got %v", err)
	}
}

// floodReader always returns a byte that is never START1 — a line that streams
// data endlessly but never forms a valid frame (a mis-identified serial device:
// a GPS, a console). The deadline must still bound the read.
type floodReader struct{ b byte }

func (r *floodReader) Read(p []byte) (int, error)  { p[0] = r.b; return 1, nil }
func (r *floodReader) Write(p []byte) (int, error) { return len(p), nil }
func (r *floodReader) Close() error                { return nil }

func TestReadByte_DeadlineStopsAFloodingLine(t *testing.T) {
	c := newStreamConn(&floodReader{b: 0x41}) // 'A', never START1
	start := time.Now()
	var out pb.FromRadio
	err := c.readMsg(&out, time.Now().Add(60*time.Millisecond))
	if !isReadTimeout(err) {
		t.Fatalf("want read timeout on a flooding line, got %v", err)
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Fatalf("read took %v; the deadline is not honoured when bytes keep arriving", d)
	}
}

func TestReadMsg_SkipsUnparseableFrameThenReadsGood(t *testing.T) {
	// A well-formed frame whose 2-byte payload is invalid protobuf — tag 0x08
	// (field 1, varint) followed by 0xFF, a varint byte that demands a
	// continuation the payload doesn't provide, so Unmarshal fails. The reader
	// must skip it and return the next, valid frame rather than erroring.
	buf := &bytes.Buffer{}
	buf.Write([]byte{start1, start2, 0x00, 0x02, 0x08, 0xFF})
	buf.Write(framedBytes(t, completeFrame(55)))

	var out pb.FromRadio
	if err := newStreamConn(memConn{buf}).readMsg(&out, time.Now().Add(time.Second)); err != nil {
		t.Fatalf("readMsg: %v", err)
	}
	if got := out.GetConfigCompleteId(); got != 55 {
		t.Fatalf("skip unparseable: got %d, want 55", got)
	}
}
