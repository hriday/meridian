package meshcore

import (
	"bufio"
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
	cmdGetBatt     = 0x14
	cmdDeviceQuery = 0x16

	// Responses we read (radio -> app).
	respErr           = 0x01
	respContactsStart = 0x02
	respContact       = 0x03
	respEndContacts   = 0x04
	respSelfInfo      = 0x05
	respBatt          = 0x0c
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

// frameReader reads radio→app frames from a serial/TCP stream, skipping any
// non-frame bytes in between — MeshCore firmware (wadamesh especially) prints
// plain-text debug logs on the same port, so the reader resyncs on the 0x3E
// marker rather than trusting the stream to contain only frames. A frame whose
// length is implausible is treated as a false marker hit and the scan continues.
type frameReader struct {
	r        *bufio.Reader
	deadline time.Time
}

func newFrameReader(r io.Reader, deadline time.Time) *frameReader {
	return &frameReader{r: bufio.NewReader(r), deadline: deadline}
}

// next returns the next frame's payload (code byte first). It skips bytes until
// a plausible frame appears. io.EOF or a read deadline ends the scan.
func (fr *frameReader) next() ([]byte, error) {
	for {
		if !fr.deadline.IsZero() && time.Now().After(fr.deadline) {
			return nil, errTimeout
		}
		b, err := fr.r.ReadByte()
		if err != nil {
			return nil, err
		}
		if b != markerIn {
			continue // debug text or mid-frame noise — keep scanning for a marker.
		}
		var lenBuf [2]byte
		if _, err := io.ReadFull(fr.r, lenBuf[:]); err != nil {
			return nil, err
		}
		n := int(binary.LittleEndian.Uint16(lenBuf[:]))
		if n == 0 || n > maxFrame {
			continue // a 0x3E that wasn't a real frame header; resync.
		}
		payload := make([]byte, n)
		if _, err := io.ReadFull(fr.r, payload); err != nil {
			return nil, err
		}
		return payload, nil
	}
}
