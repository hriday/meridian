package meshcore

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"
)

// MeshCore companion-radio serial framing. Every frame on the wire is one
// direction marker, a little-endian uint16 length, then that many payload bytes;
// payload[0] is the command/response code. App→radio frames use marker 0x3C
// ('<'), radio→app frames use 0x3E ('>'). (Per the MeshCore wiki's Companion
// Radio Protocol; the protocol is firmware-versioned and still evolving.)
const (
	markerOut = 0x3C // app -> radio
	markerIn  = 0x3E // radio -> app

	// Commands we send (app -> radio).
	cmdAppStart    = 0x01
	cmdGetContacts = 0x04
	cmdDeviceQuery = 0x16

	// Responses we read (radio -> app).
	respErr           = 0x01
	respContactsStart = 0x02
	respContact       = 0x03
	respEndContacts   = 0x04
	respSelfInfo      = 0x05
	respDeviceInfo    = 0x0d

	maxFrame = 512 // generous ceiling; real frames are ~172 bytes
)

// writeFrame sends one app→radio frame: marker, uint16-LE length, payload.
func writeFrame(w io.Writer, payload []byte) error {
	if len(payload) > maxFrame {
		return fmt.Errorf("meshcore: frame too large (%d bytes)", len(payload))
	}
	hdr := [3]byte{markerOut, 0, 0}
	binary.LittleEndian.PutUint16(hdr[1:], uint16(len(payload)))
	if _, err := w.Write(hdr[:]); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}

// frameReader reads radio→app frames, skipping any non-frame bytes in between —
// MeshCore firmware (wadamesh especially) prints plain-text debug logs on the
// same port, so the reader resyncs on the 0x3E marker rather than trusting the
// stream to contain only frames. A frame whose length is implausible is treated
// as a false marker hit and the scan continues.
//
// It does its own buffering so the per-call deadline is honoured on every read:
// go.bug.st serial returns (0, nil) on its own short timeout, and bufio would
// retry that internally up to 100 times, blocking tens of seconds past the
// deadline. Here each empty read loops straight back to the deadline check.
type frameReader struct {
	r   io.Reader
	buf []byte
	pos int
}

func newFrameReader(r io.Reader) *frameReader { return &frameReader{r: r} }

// readByte returns the next byte, blocking only until deadline. A zero deadline
// means "no deadline" (used in tests over a bytes.Reader, which ends at io.EOF).
func (fr *frameReader) readByte(deadline time.Time) (byte, error) {
	for fr.pos >= len(fr.buf) {
		if !deadline.IsZero() && time.Now().After(deadline) {
			return 0, errTimeout
		}
		tmp := make([]byte, 256)
		n, err := fr.r.Read(tmp)
		if n > 0 {
			fr.buf, fr.pos = tmp[:n], 0
			break
		}
		if err != nil {
			return 0, err
		}
		// n == 0, err == nil: a serial read timeout — loop and recheck the deadline.
	}
	b := fr.buf[fr.pos]
	fr.pos++
	return b, nil
}

// next returns the next frame's payload (code byte first), honouring deadline.
func (fr *frameReader) next(deadline time.Time) ([]byte, error) {
	for {
		b, err := fr.readByte(deadline)
		if err != nil {
			return nil, err
		}
		if b != markerIn {
			continue // debug text or mid-frame noise — keep scanning for a marker.
		}
		lo, err := fr.readByte(deadline)
		if err != nil {
			return nil, err
		}
		hi, err := fr.readByte(deadline)
		if err != nil {
			return nil, err
		}
		n := int(lo) | int(hi)<<8
		if n == 0 || n > maxFrame {
			continue // a 0x3E that wasn't a real frame header; resync.
		}
		payload := make([]byte, n)
		for i := 0; i < n; i++ {
			if payload[i], err = fr.readByte(deadline); err != nil {
				return nil, err
			}
		}
		return payload, nil
	}
}
