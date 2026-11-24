// Package meshcore takes a point-in-time read of a MeshCore LoRa node over its
// USB-serial (or TCP) companion-radio protocol: it connects, asks the node for
// its own identity and radio config and the contacts it currently knows, and
// returns a snapshot plus a mesh-health read. It keeps no history.
//
// MeshCore is a different project from Meshtastic and a different wire protocol;
// this talks the MeshCore companion protocol, which Meridian's Meshtastic tool
// cannot. Contacts carry no per-neighbour SNR or hop count (MeshCore only
// reports those on received messages), so the node list here is identity, type,
// stored-path length, last-heard and position — not live signal.
package meshcore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"regexp"
	"strings"
	"time"

	"go.bug.st/serial"
)

var errTimeout = errors.New("meshcore: read timed out")

// Snapshot is one read of a MeshCore node and the contacts it knows.
type Snapshot struct {
	Transport    string       `json:"transport"` // "serial" or "tcp"
	Target       string       `json:"target"`
	Name         string       `json:"name"`      // the node's advertised name
	PublicKey    string       `json:"publicKey"` // hex prefix identifying the node
	Firmware     string       `json:"firmware"`  // semantic version, if reported
	Model        string       `json:"model"`     // hardware/model string, if reported
	Radio        *RadioConfig `json:"radio,omitempty"`
	Contacts     []Contact    `json:"contacts"`
	ContactCount int          `json:"contactCount"`
	HasPosition  bool         `json:"hasPosition"`
	Lat          float64      `json:"lat,omitempty"`
	Lon          float64      `json:"lon,omitempty"`
	TakenAt      string       `json:"takenAt"`
	Findings     []Finding    `json:"findings,omitempty"`
}

const defaultBudget = 15 * time.Second

var comPort = regexp.MustCompile(`^(?i:com)\d+$`)

// classify splits target into a serial device or a TCP endpoint, mirroring the
// Meshtastic tool. MeshCore's companion TCP service defaults to port 5000.
func classify(target string) (transport, addr string) {
	t := strings.TrimSpace(target)
	switch {
	case strings.HasPrefix(t, "serial://"):
		return "serial", strings.TrimPrefix(t, "serial://")
	case strings.HasPrefix(t, "tcp://"):
		t = strings.TrimPrefix(t, "tcp://")
	default:
		if strings.HasPrefix(t, "/") || comPort.MatchString(t) {
			return "serial", t
		}
	}
	if t != "" {
		if _, _, err := net.SplitHostPort(t); err != nil {
			t = net.JoinHostPort(t, "5000")
		}
	}
	return "tcp", t
}

func open(ctx context.Context, transport, addr string) (io.ReadWriteCloser, error) {
	if addr == "" {
		return nil, errors.New("meshcore: enter a serial device (e.g. /dev/cu.usbmodem1101) or a host (e.g. 192.168.1.50)")
	}
	switch transport {
	case "serial":
		port, err := serial.Open(addr, &serial.Mode{BaudRate: 115200})
		if err != nil {
			return nil, fmt.Errorf("meshcore: opening serial port %s: %w", addr, err)
		}
		_ = port.SetReadTimeout(300 * time.Millisecond)
		return port, nil
	default:
		d := net.Dialer{Timeout: 5 * time.Second}
		conn, err := d.DialContext(ctx, "tcp", addr)
		if err != nil {
			return nil, fmt.Errorf("meshcore: connecting to %s: %w", addr, err)
		}
		return conn, nil
	}
}

// Read connects to a MeshCore node and returns a point-in-time Snapshot. target
// is a serial device path or a TCP host[:5000].
func Read(ctx context.Context, target string) (Snapshot, error) {
	transport, addr := classify(target)
	rwc, err := open(ctx, transport, addr)
	if err != nil {
		return Snapshot{}, err
	}
	defer rwc.Close()

	budget := defaultBudget
	if dl, ok := ctx.Deadline(); ok {
		if b := time.Until(dl); b < budget {
			budget = b
		}
	}
	deadline := time.Now().Add(budget)
	if d, ok := rwc.(interface{ SetReadDeadline(time.Time) error }); ok {
		_ = d.SetReadDeadline(deadline)
	}

	fr := newFrameReader(rwc)
	snap := Snapshot{Transport: transport, Target: addr, TakenAt: time.Now().UTC().Format(time.RFC3339)}

	// APP_START → SELF_INFO: the node's identity and radio config. This is the
	// handshake, so a failure here means we never reached a MeshCore node.
	si, err := exchange(rwc, fr, appStartCmd(), respSelfInfo, deadline)
	if err != nil {
		return Snapshot{}, fmt.Errorf("meshcore: no companion response from %s — is it a MeshCore node reachable over this port, awake, and not already in use by another app?", addr)
	}
	self, _ := parseSelfInfo(si)
	snap.Name, snap.PublicKey = self.Name, self.PubKey
	r := self.Radio
	snap.Radio = &r
	snap.HasPosition, snap.Lat, snap.Lon = self.HasPosition, self.Lat, self.Lon

	// DEVICE_QUERY → DEVICE_INFO: firmware and model, best-effort — on a firmware
	// that ignores the command this must NOT burn the whole budget, or contacts
	// would read as empty and diagnose would cry "no nodes" falsely. Bound it.
	devDeadline := time.Now().Add(3 * time.Second)
	if devDeadline.After(deadline) {
		devDeadline = deadline
	}
	if di, err := exchange(rwc, fr, []byte{cmdDeviceQuery, 3}, respDeviceInfo, devDeadline); err == nil {
		d, _ := parseDeviceInfo(di)
		snap.Firmware, snap.Model = d.Version, d.Model
	}

	// GET_CONTACTS → the node list.
	snap.Contacts = readContacts(rwc, fr, deadline)
	snap.ContactCount = len(snap.Contacts)
	rankContacts(snap.Contacts)

	snap.Findings = diagnose(snap)
	return snap, nil
}

// exchange sends one command and reads frames until the wanted response code
// arrives, skipping unsolicited pushes and debug noise. A device error frame
// ends the wait with an error.
func exchange(w io.Writer, fr *frameReader, cmd []byte, want byte, deadline time.Time) ([]byte, error) {
	if err := writeFrame(w, cmd); err != nil {
		return nil, err
	}
	for {
		p, err := fr.next(deadline)
		if err != nil {
			return nil, err
		}
		if len(p) == 0 {
			continue
		}
		switch p[0] {
		case want:
			return p, nil
		case respErr:
			return nil, fmt.Errorf("meshcore: device returned error code %d", errCode(p))
		default:
			continue // a push (advert, log, …) — not what we asked for.
		}
	}
}

// readContacts requests and collects the contact list. A read error or the
// end-of-contacts marker stops it; anything it already has is returned.
func readContacts(w io.Writer, fr *frameReader, deadline time.Time) []Contact {
	cs := []Contact{}
	if err := writeFrame(w, []byte{cmdGetContacts}); err != nil {
		return cs
	}
	for {
		p, err := fr.next(deadline)
		if err != nil {
			return cs
		}
		if len(p) == 0 {
			continue
		}
		switch p[0] {
		case respContact:
			if c, err := parseContact(p); err == nil {
				cs = append(cs, c)
			}
		case respEndContacts:
			return cs
		default:
			continue
		}
		if len(cs) > 5000 {
			return cs
		}
	}
}

// appStartCmd builds the APP_START frame: code, app version, six reserved bytes,
// then the client name.
func appStartCmd() []byte {
	b := []byte{cmdAppStart, 1, 0, 0, 0, 0, 0, 0}
	return append(b, []byte("meridian")...)
}

// errCode reads the error code byte from a RESP_CODE_ERR frame.
func errCode(p []byte) int {
	if len(p) >= 2 {
		return int(p[1])
	}
	return 0
}
