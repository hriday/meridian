// Package meshtastic takes a point-in-time read of a Meshtastic LoRa mesh node:
// it connects to a node over USB serial or TCP, asks it to dump its own config
// and node database, and returns the local node plus the mesh it currently sees
// (neighbours, signal, channel airtime, positions). It keeps no history — go in,
// read the mesh, move on.
package meshtastic

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net"
	"regexp"
	"sort"
	"strings"
	"time"

	pb "buf.build/gen/go/meshtastic/protobufs/protocolbuffers/go/meshtastic"
	"go.bug.st/serial"
)

// Snapshot is one read of a node and the mesh around it.
type Snapshot struct {
	Transport   string  `json:"transport"`   // "serial" or "tcp"
	Target      string  `json:"target"`      // the device path or host:port actually used
	MyNodeNum   uint32  `json:"myNodeNum"`   // the connected node's number
	Firmware    string  `json:"firmware"`    // firmware version, if the node reported it
	HwModel     string  `json:"hwModel"`     // hardware model of the connected node
	Me          *Node   `json:"me"`          // the connected node itself (nil if it never identified)
	Nodes       []Node  `json:"nodes"`       // every other node in the database, ranked nearest-first
	ChannelUtil float64 `json:"channelUtil"` // my node's channel utilisation, percent
	AirUtilTx   float64 `json:"airUtilTx"`   // my node's transmit airtime, percent
	NodeCount   int     `json:"nodeCount"`   // total nodes seen, including me
	TakenAt     string  `json:"takenAt"`     // RFC3339 UTC instant of this read

	Radio    *RadioConfig `json:"radio,omitempty"`    // the node's LoRa setup, when the dump carried it
	Findings []Finding    `json:"findings,omitempty"` // mesh-health call-outs, most important first
}

// Node is one entry in the mesh node database.
type Node struct {
	Num         uint32  `json:"num"`
	ID          string  `json:"id"` // user id, e.g. "!a1b2c3d4"
	LongName    string  `json:"longName"`
	ShortName   string  `json:"shortName"`
	HwModel     string  `json:"hwModel"`
	Role        string  `json:"role"`
	SNR         float64 `json:"snr"`       // last-heard signal-to-noise, dB
	HopsAway    *int    `json:"hopsAway"`  // nil when the node didn't report it
	LastHeard   int64   `json:"lastHeard"` // epoch seconds; 0 = never/unknown
	ViaMQTT     bool    `json:"viaMqtt"`   // reached over an MQTT gateway, not direct RF
	Battery     *int    `json:"battery"`   // percent; nil unknown; 101 reported = powered
	Voltage     float64 `json:"voltage"`
	HasPosition bool    `json:"hasPosition"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	AltitudeM   int32   `json:"altitudeM"`
}

// defaultBudget bounds a single snapshot. A want_config dump of a small mesh
// completes in well under a second; a larger node DB or a slow serial link needs
// more headroom, and a silent node must still end the read.
const defaultBudget = 20 * time.Second

var comPort = regexp.MustCompile(`^(?i:com)\d+$`)

// classify decides whether target names a serial device or a TCP endpoint, and
// normalises it. A leading "serial://" or "tcp://" forces the choice; otherwise a
// path ("/dev/...") or a Windows "COM3" is serial and anything else is TCP, with
// Meshtastic's default port 4403 supplied when none is given.
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
	// Ensure a host:port, supplying Meshtastic's default 4403 when no port is
	// given. net.JoinHostPort brackets an IPv6 literal correctly; SplitHostPort
	// succeeds only when a port is already present.
	if t != "" {
		if _, _, err := net.SplitHostPort(t); err != nil {
			t = net.JoinHostPort(t, "4403")
		}
	}
	return "tcp", t
}

// deadliner is implemented by net.Conn.
type deadliner interface{ SetReadDeadline(time.Time) error }

func open(ctx context.Context, transport, addr string) (io.ReadWriteCloser, error) {
	if addr == "" {
		return nil, errors.New("meshtastic: enter a serial device (e.g. /dev/cu.usbserial-10) or a host (e.g. 192.168.1.50)")
	}
	switch transport {
	case "serial":
		port, err := serial.Open(addr, &serial.Mode{BaudRate: 115200})
		if err != nil {
			return nil, fmt.Errorf("meshtastic: opening serial port %s: %w", addr, err)
		}
		// Short tick so readByte can re-check the overall deadline on a quiet line.
		_ = port.SetReadTimeout(500 * time.Millisecond)
		return port, nil
	default:
		d := net.Dialer{Timeout: 5 * time.Second}
		conn, err := d.DialContext(ctx, "tcp", addr)
		if err != nil {
			return nil, fmt.Errorf("meshtastic: connecting to %s: %w", addr, err)
		}
		return conn, nil
	}
}

// Read connects to a Meshtastic node and returns a point-in-time Snapshot of it
// and the mesh it sees. target is a serial device path or a TCP host[:4403].
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
	if d, ok := rwc.(deadliner); ok {
		_ = d.SetReadDeadline(deadline) // TCP: the overall budget is the read deadline.
	}

	conn := newStreamConn(rwc)

	// Nudge the node awake, then ask it to stream its config and node DB. The id
	// is echoed back in config_complete_id, which marks the end of the dump.
	_, _ = rwc.Write([]byte{start1, start1, start1, start1})
	time.Sleep(100 * time.Millisecond)
	wantID := rand.Uint32()
	if wantID == 0 {
		wantID = 1
	}
	req := &pb.ToRadio{PayloadVariant: &pb.ToRadio_WantConfigId{WantConfigId: wantID}}
	if err := conn.writeMsg(req); err != nil {
		return Snapshot{}, fmt.Errorf("meshtastic: requesting config: %w", err)
	}

	var frames []*pb.FromRadio
	for {
		if err := ctx.Err(); err != nil {
			return Snapshot{}, err
		}
		var fr pb.FromRadio
		if err := conn.readMsg(&fr, deadline); err != nil {
			// Quiet, closed, or a mid-stream glitch after we already have useful
			// frames: stop and assemble what we collected rather than discard it.
			// Only an error before any frame arrived is reported as a failure.
			if len(frames) > 0 || isReadTimeout(err) || errors.Is(err, io.EOF) {
				break
			}
			return Snapshot{}, fmt.Errorf("meshtastic: reading from node: %w", err)
		}
		frames = append(frames, &fr)
		if fr.GetConfigCompleteId() == wantID {
			break // the node has finished its dump.
		}
		if len(frames) > 20000 {
			break // defensive: never collect unboundedly.
		}
	}

	if len(frames) == 0 {
		return Snapshot{}, fmt.Errorf("meshtastic: no response from node at %s — is it a Meshtastic device, awake, and not already in use by another client?", addr)
	}
	return assemble(transport, addr, frames), nil
}

// assemble folds a config/node-DB dump into a Snapshot. It is pure: given the
// FromRadio frames a node streamed, it produces the same result with no I/O, so
// the protocol mapping is testable without hardware.
func assemble(transport, addr string, frames []*pb.FromRadio) Snapshot {
	snap := Snapshot{
		Transport: transport,
		Target:    addr,
		TakenAt:   time.Now().UTC().Format(time.RFC3339),
	}
	byNum := map[uint32]*pb.NodeInfo{}
	var order []uint32
	var lora *pb.Config_LoRaConfig
	primaryChannel := ""
	activeChannels := 0
	for _, fr := range frames {
		if mi := fr.GetMyInfo(); mi != nil {
			snap.MyNodeNum = mi.GetMyNodeNum()
		}
		if md := fr.GetMetadata(); md != nil {
			snap.Firmware = md.GetFirmwareVersion()
			if snap.HwModel == "" {
				snap.HwModel = hwModel(md.GetHwModel())
			}
		}
		if cfg := fr.GetConfig(); cfg != nil {
			if l := cfg.GetLora(); l != nil {
				lora = l
			}
		}
		if ch := fr.GetChannel(); ch != nil {
			if ch.GetRole() != pb.Channel_DISABLED {
				activeChannels++
			}
			if ch.GetRole() == pb.Channel_PRIMARY {
				primaryChannel = ch.GetSettings().GetName()
			}
		}
		if ni := fr.GetNodeInfo(); ni != nil {
			if _, seen := byNum[ni.GetNum()]; !seen {
				order = append(order, ni.GetNum())
			}
			byNum[ni.GetNum()] = ni // a later entry supersedes an earlier one.
		}
	}
	snap.Radio = radioFromLoRa(lora, primaryChannel, activeChannels)

	snap.NodeCount = len(order)
	for _, num := range order {
		ni := byNum[num]
		node := toNode(ni)
		if num == snap.MyNodeNum {
			me := node
			snap.Me = &me
			if snap.HwModel == "" {
				snap.HwModel = node.HwModel
			}
			if dm := ni.GetDeviceMetrics(); dm != nil {
				snap.ChannelUtil = round1(float64(dm.GetChannelUtilization()))
				snap.AirUtilTx = round1(float64(dm.GetAirUtilTx()))
			}
			continue
		}
		snap.Nodes = append(snap.Nodes, node)
	}
	rankNodes(snap.Nodes)
	snap.Findings = diagnose(snap)
	return snap
}

func toNode(ni *pb.NodeInfo) Node {
	n := Node{
		Num:       ni.GetNum(),
		SNR:       round1(float64(ni.GetSnr())),
		LastHeard: int64(ni.GetLastHeard()),
		ViaMQTT:   ni.GetViaMqtt(),
	}
	// HopsAway is only meaningful when the node carried it; 0 legitimately means
	// a direct neighbour, so distinguish "0 hops" from "not reported" via the
	// protobuf presence bit.
	if ni.HasHopsAway() {
		h := int(ni.GetHopsAway())
		n.HopsAway = &h
	}
	if u := ni.GetUser(); u != nil {
		n.ID = u.GetId()
		n.LongName = u.GetLongName()
		n.ShortName = u.GetShortName()
		n.HwModel = hwModel(u.GetHwModel())
		n.Role = role(u.GetRole())
	}
	if p := ni.GetPosition(); p != nil && (p.GetLatitudeI() != 0 || p.GetLongitudeI() != 0) {
		n.HasPosition = true
		n.Lat = round6(float64(p.GetLatitudeI()) * 1e-7)
		n.Lon = round6(float64(p.GetLongitudeI()) * 1e-7)
		n.AltitudeM = p.GetAltitude()
	}
	if dm := ni.GetDeviceMetrics(); dm != nil {
		n.Voltage = round2(float64(dm.GetVoltage()))
		if dm.HasBatteryLevel() {
			b := int(dm.GetBatteryLevel())
			n.Battery = &b
		}
	}
	return n
}

// rankNodes orders the mesh nearest-first: fewest hops, then most recently heard,
// then by name — so the closest/liveliest neighbours sit at the top. Nodes that
// never reported a hop count sort after those that did.
func rankNodes(ns []Node) {
	sort.SliceStable(ns, func(i, j int) bool {
		a, b := ns[i], ns[j]
		ha, hb := hopKey(a.HopsAway), hopKey(b.HopsAway)
		if ha != hb {
			return ha < hb
		}
		if a.LastHeard != b.LastHeard {
			return a.LastHeard > b.LastHeard
		}
		return strings.ToLower(a.LongName) < strings.ToLower(b.LongName)
	})
}

func hopKey(h *int) int {
	if h == nil {
		return math.MaxInt32
	}
	return *h
}

func round1(f float64) float64 { return math.Round(f*10) / 10 }
func round2(f float64) float64 { return math.Round(f*100) / 100 }
func round6(f float64) float64 { return math.Round(f*1e6) / 1e6 }
