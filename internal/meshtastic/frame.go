package meshtastic

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"time"

	"google.golang.org/protobuf/proto"
)

// Meshtastic's stream transport (serial and TCP alike) frames each protobuf
// message with a 4-byte header: START1 START2 then a big-endian uint16 length,
// followed by that many payload bytes. The same bytes carry human-readable debug
// logs between frames, so a reader must resynchronise on the magic rather than
// assume the stream is packet-aligned.
//
// https://meshtastic.org/docs/development/device/client-api/
const (
	start1       = 0x94
	start2       = 0xc3
	maxFrameSize = 512 // Meshtastic PacketMTU; a longer length field means noise.
)

// errReadTimeout is returned when no frame arrives before the deadline. It is a
// normal end condition for a point-in-time read (the node has gone quiet), not a
// failure, so the caller assembles whatever it already collected.
var errReadTimeout = errors.New("meshtastic: read deadline exceeded")

func isReadTimeout(err error) bool {
	if errors.Is(err, errReadTimeout) {
		return true
	}
	var te interface{ Timeout() bool }
	return errors.As(err, &te) && te.Timeout()
}

// streamConn frames protobuf messages over any byte stream.
type streamConn struct {
	rw io.ReadWriteCloser
}

func newStreamConn(rw io.ReadWriteCloser) *streamConn { return &streamConn{rw: rw} }

// writeMsg marshals m and writes it as one framed packet.
func (c *streamConn) writeMsg(m proto.Message) error {
	payload, err := proto.Marshal(m)
	if err != nil {
		return err
	}
	if len(payload) > maxFrameSize {
		return fmt.Errorf("meshtastic: outbound frame too large (%d > %d)", len(payload), maxFrameSize)
	}
	hdr := [4]byte{start1, start2}
	binary.BigEndian.PutUint16(hdr[2:], uint16(len(payload)))
	if _, err := c.rw.Write(hdr[:]); err != nil {
		return err
	}
	_, err = c.rw.Write(payload)
	return err
}

// readMsg reads the next framed packet (resynchronising past any inter-frame
// noise) and unmarshals it into m. A frame whose payload won't parse — the
// resync having locked onto noise that merely looked like a header — is skipped
// rather than failed, so one false frame can't discard a dump in progress. It
// gives up with errReadTimeout once deadline passes with no complete frame.
func (c *streamConn) readMsg(m proto.Message, deadline time.Time) error {
	for {
		payload, err := c.readFrame(deadline)
		if err != nil {
			return err
		}
		proto.Reset(m)
		if err := proto.Unmarshal(payload, m); err != nil {
			continue // not a real frame; resync to the next.
		}
		return nil
	}
}

func (c *streamConn) readFrame(deadline time.Time) ([]byte, error) {
	for {
		// Hunt for START1.
		b, err := c.readByte(deadline)
		if err != nil {
			return nil, err
		}
		if b != start1 {
			continue
		}
		// START2 must follow; if another START1 shows up instead, keep it as the
		// new candidate rather than consuming it.
		b, err = c.readByte(deadline)
		if err != nil {
			return nil, err
		}
		for b == start1 {
			if b, err = c.readByte(deadline); err != nil {
				return nil, err
			}
		}
		if b != start2 {
			continue
		}
		var lenb [2]byte
		if err := c.readFull(lenb[:], deadline); err != nil {
			return nil, err
		}
		n := int(binary.BigEndian.Uint16(lenb[:]))
		if n == 0 || n > maxFrameSize {
			continue // implausible length: treat the header as noise and resync.
		}
		payload := make([]byte, n)
		if err := c.readFull(payload, deadline); err != nil {
			return nil, err
		}
		return payload, nil
	}
}

func (c *streamConn) readFull(p []byte, deadline time.Time) error {
	for i := range p {
		b, err := c.readByte(deadline)
		if err != nil {
			return err
		}
		p[i] = b
	}
	return nil
}

// readByte reads a single byte, tolerating the (0, nil) ticks that a serial port
// returns on its own read timeout and converting a passed deadline into
// errReadTimeout. TCP connections instead surface their own timeout error once
// their read deadline (set to the overall deadline) fires.
func (c *streamConn) readByte(deadline time.Time) (byte, error) {
	var b [1]byte
	for {
		// Check the deadline before every read, not only on an empty one — a
		// serial line that streams bytes without ever forming a valid frame (a
		// GPS, a console, a mis-identified device) would otherwise never let the
		// resync loop return, hanging past the budget and past ctx.
		if !time.Now().Before(deadline) {
			return 0, errReadTimeout
		}
		n, err := c.rw.Read(b[:])
		if n > 0 {
			return b[0], nil
		}
		if err != nil {
			if isReadTimeout(err) {
				return 0, errReadTimeout
			}
			return 0, err
		}
		// n == 0, err == nil: a serial inter-tick with no data; loop and re-check.
	}
}

func (c *streamConn) close() error { return c.rw.Close() }
