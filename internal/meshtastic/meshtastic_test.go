package meshtastic

import (
	"testing"

	pb "buf.build/gen/go/meshtastic/protobufs/protocolbuffers/go/meshtastic"
	"google.golang.org/protobuf/proto"
)

func TestClassify(t *testing.T) {
	cases := []struct{ in, wantTransport, wantAddr string }{
		{"/dev/cu.usbserial-10", "serial", "/dev/cu.usbserial-10"},
		{"/dev/ttyUSB0", "serial", "/dev/ttyUSB0"},
		{"COM3", "serial", "COM3"},
		{"com4", "serial", "com4"},
		{"serial:///dev/ttyACM0", "serial", "/dev/ttyACM0"},
		{"192.168.1.50", "tcp", "192.168.1.50:4403"},
		{"192.168.1.50:4403", "tcp", "192.168.1.50:4403"},
		{"meshnode.local", "tcp", "meshnode.local:4403"},
		{"tcp://10.0.0.9", "tcp", "10.0.0.9:4403"},
		{"  10.0.0.9  ", "tcp", "10.0.0.9:4403"},
		{"2001:db8::1", "tcp", "[2001:db8::1]:4403"},
		{"[2001:db8::1]:4403", "tcp", "[2001:db8::1]:4403"},
	}
	for _, c := range cases {
		gotT, gotA := classify(c.in)
		if gotT != c.wantTransport || gotA != c.wantAddr {
			t.Errorf("classify(%q) = (%q, %q); want (%q, %q)", c.in, gotT, gotA, c.wantTransport, c.wantAddr)
		}
	}
}

func TestAssemble_BuildsSnapshotAndRanksNodes(t *testing.T) {
	me := (&pb.NodeInfo_builder{
		Num:  100,
		User: (&pb.User_builder{Id: "!00000064", LongName: "Base", ShortName: "BASE", HwModel: pb.HardwareModel_HELTEC_V3, Role: pb.Config_DeviceConfig_CLIENT}).Build(),
		DeviceMetrics: (&pb.DeviceMetrics_builder{
			ChannelUtilization: proto.Float32(12.53),
			AirUtilTx:          proto.Float32(3.21),
			BatteryLevel:       proto.Uint32(101),
			Voltage:            proto.Float32(4.19),
		}).Build(),
	}).Build()

	near := (&pb.NodeInfo_builder{
		Num:           200,
		Snr:           6.28,
		LastHeard:     1_700_000_200,
		HopsAway:      proto.Uint32(1),
		User:          (&pb.User_builder{Id: "!000000c8", LongName: "Hilltop", ShortName: "HILL", Role: pb.Config_DeviceConfig_CLIENT}).Build(),
		Position:      (&pb.Position_builder{LatitudeI: proto.Int32(375_000_000), LongitudeI: proto.Int32(-1_223_000_000), Altitude: proto.Int32(84)}).Build(),
		DeviceMetrics: (&pb.DeviceMetrics_builder{BatteryLevel: proto.Uint32(72)}).Build(),
	}).Build()

	far := (&pb.NodeInfo_builder{
		Num:       300,
		Snr:       -11.0,
		LastHeard: 1_700_000_300,
		HopsAway:  proto.Uint32(3),
		User:      (&pb.User_builder{Id: "!0000012c", LongName: "Ridge"}).Build(),
	}).Build()

	// No hop count reported — must rank after every node that has one.
	noHops := (&pb.NodeInfo_builder{
		Num:       400,
		LastHeard: 1_700_000_400,
		ViaMqtt:   true,
		User:      (&pb.User_builder{Id: "!00000190", LongName: "Gateway"}).Build(),
	}).Build()

	frames := []*pb.FromRadio{
		(&pb.FromRadio_builder{MyInfo: (&pb.MyNodeInfo_builder{MyNodeNum: 100}).Build()}).Build(),
		(&pb.FromRadio_builder{Metadata: (&pb.DeviceMetadata_builder{FirmwareVersion: "2.5.3.abcdef", HwModel: pb.HardwareModel_HELTEC_V3}).Build()}).Build(),
		(&pb.FromRadio_builder{NodeInfo: far}).Build(),
		(&pb.FromRadio_builder{NodeInfo: noHops}).Build(),
		(&pb.FromRadio_builder{NodeInfo: near}).Build(),
		(&pb.FromRadio_builder{NodeInfo: me}).Build(),
		(&pb.FromRadio_builder{ConfigCompleteId: proto.Uint32(42)}).Build(),
	}

	snap := assemble("tcp", "192.168.1.50:4403", frames)

	if snap.Transport != "tcp" || snap.Target != "192.168.1.50:4403" {
		t.Errorf("transport/target = %q/%q", snap.Transport, snap.Target)
	}
	if snap.MyNodeNum != 100 {
		t.Errorf("MyNodeNum = %d, want 100", snap.MyNodeNum)
	}
	if snap.Firmware != "2.5.3.abcdef" {
		t.Errorf("Firmware = %q, want 2.5.3.abcdef", snap.Firmware)
	}
	if snap.HwModel != "HELTEC_V3" {
		t.Errorf("HwModel = %q, want HELTEC_V3", snap.HwModel)
	}
	if snap.NodeCount != 4 {
		t.Errorf("NodeCount = %d, want 4 (me + 3 neighbours)", snap.NodeCount)
	}
	if snap.ChannelUtil != 12.5 {
		t.Errorf("ChannelUtil = %v, want 12.5", snap.ChannelUtil)
	}
	if snap.AirUtilTx != 3.2 {
		t.Errorf("AirUtilTx = %v, want 3.2", snap.AirUtilTx)
	}

	if snap.Me == nil {
		t.Fatal("Me is nil; want the connected node separated out")
	}
	if snap.Me.Num != 100 || snap.Me.Role != "Client" {
		t.Errorf("Me = {Num:%d Role:%q}, want {100 Client}", snap.Me.Num, snap.Me.Role)
	}

	// Me is pulled out; the rest rank nearest-first: near (1), far (3), noHops (∞).
	if len(snap.Nodes) != 3 {
		t.Fatalf("Nodes len = %d, want 3", len(snap.Nodes))
	}
	for i, want := range []uint32{200, 300, 400} {
		if snap.Nodes[i].Num != want {
			t.Errorf("Nodes[%d].Num = %d, want %d", i, snap.Nodes[i].Num, want)
		}
	}

	n := snap.Nodes[0] // near
	if n.SNR != 6.3 {
		t.Errorf("near.SNR = %v, want 6.3", n.SNR)
	}
	if n.HopsAway == nil || *n.HopsAway != 1 {
		t.Errorf("near.HopsAway = %v, want 1", n.HopsAway)
	}
	if !n.HasPosition || n.Lat != 37.5 || n.Lon != -122.3 || n.AltitudeM != 84 {
		t.Errorf("near position = {has:%v lat:%v lon:%v alt:%d}, want {true 37.5 -122.3 84}", n.HasPosition, n.Lat, n.Lon, n.AltitudeM)
	}
	if n.Battery == nil || *n.Battery != 72 {
		t.Errorf("near.Battery = %v, want 72", n.Battery)
	}

	if snap.Nodes[2].HopsAway != nil {
		t.Errorf("noHops.HopsAway = %v, want nil (unreported)", *snap.Nodes[2].HopsAway)
	}
	if !snap.Nodes[2].ViaMQTT {
		t.Error("noHops.ViaMQTT = false, want true")
	}
}

func TestAssemble_LaterNodeInfoSupersedesEarlier(t *testing.T) {
	first := (&pb.NodeInfo_builder{Num: 500, User: (&pb.User_builder{LongName: "Old name"}).Build()}).Build()
	second := (&pb.NodeInfo_builder{Num: 500, User: (&pb.User_builder{LongName: "New name"}).Build()}).Build()

	snap := assemble("serial", "/dev/ttyUSB0", []*pb.FromRadio{
		(&pb.FromRadio_builder{NodeInfo: first}).Build(),
		(&pb.FromRadio_builder{NodeInfo: second}).Build(),
	})

	if snap.NodeCount != 1 {
		t.Fatalf("NodeCount = %d, want 1 (deduped by node number)", snap.NodeCount)
	}
	if snap.Nodes[0].LongName != "New name" {
		t.Errorf("LongName = %q, want the later value %q", snap.Nodes[0].LongName, "New name")
	}
}
