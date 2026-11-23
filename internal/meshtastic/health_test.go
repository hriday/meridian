package meshtastic

import (
	"strings"
	"testing"

	pb "buf.build/gen/go/meshtastic/protobufs/protocolbuffers/go/meshtastic"
	"google.golang.org/protobuf/proto"
)

func myInfoFrame(num uint32) *pb.FromRadio {
	return (&pb.FromRadio_builder{MyInfo: (&pb.MyNodeInfo_builder{MyNodeNum: num}).Build()}).Build()
}
func nodeFrame(num uint32) *pb.FromRadio {
	return (&pb.FromRadio_builder{NodeInfo: (&pb.NodeInfo_builder{Num: num}).Build()}).Build()
}
func nodeWithMetrics(num uint32, chUtil, airTx float32) *pb.FromRadio {
	return (&pb.FromRadio_builder{NodeInfo: (&pb.NodeInfo_builder{
		Num:           num,
		DeviceMetrics: (&pb.DeviceMetrics_builder{ChannelUtilization: proto.Float32(chUtil), AirUtilTx: proto.Float32(airTx)}).Build(),
	}).Build()}).Build()
}
func loraFrame(region pb.Config_LoRaConfig_RegionCode, preset pb.Config_LoRaConfig_ModemPreset, usePreset, tx bool) *pb.FromRadio {
	return (&pb.FromRadio_builder{Config: (&pb.Config_builder{Lora: (&pb.Config_LoRaConfig_builder{
		Region: region, ModemPreset: preset, UsePreset: usePreset, TxEnabled: tx,
	}).Build()}).Build()}).Build()
}
func primaryChanFrame(name string) *pb.FromRadio {
	return (&pb.FromRadio_builder{Channel: (&pb.Channel_builder{
		Role: pb.Channel_PRIMARY, Settings: (&pb.ChannelSettings_builder{Name: name}).Build(),
	}).Build()}).Build()
}

func findingLevels(fs []Finding) string {
	var b []string
	for _, f := range fs {
		b = append(b, f.Level+":"+f.Title)
	}
	return strings.Join(b, " | ")
}
func hasFinding(fs []Finding, level, substr string) bool {
	for _, f := range fs {
		if f.Level == level && strings.Contains(strings.ToLower(f.Title+" "+f.Fix), strings.ToLower(substr)) {
			return true
		}
	}
	return false
}

// TestRadioConfigExtracted pins that the LoRa config is pulled out of the dump.
func TestRadioConfigExtracted(t *testing.T) {
	snap := assemble("serial", "/dev/x", []*pb.FromRadio{
		myInfoFrame(100),
		loraFrame(pb.Config_LoRaConfig_US, pb.Config_LoRaConfig_LONG_FAST, true, true),
		primaryChanFrame("Secret"),
		nodeFrame(100), nodeFrame(200),
	})
	r := snap.Radio
	if r == nil {
		t.Fatal("Radio config was not extracted from the dump")
	}
	if r.Region != "US" || !r.RegionSet {
		t.Errorf("region wrong: %+v", r)
	}
	if r.ModemPreset != "LongFast" || !r.UsePreset {
		t.Errorf("preset should render as LongFast: %+v", r)
	}
	if !r.TxEnabled {
		t.Errorf("tx should be enabled: %+v", r)
	}
	if r.PrimaryChannel != "Secret" || !r.ChannelKnown {
		t.Errorf("channel wrong: %+v", r)
	}
}

// TestDiagnose_RangeClaimNeedsMetrics: the "points at range" inference may only
// appear when the node actually reported metrics — absent metrics default to 0%
// and must never become a range verdict (failed lookup ≠ verdict).
func TestDiagnose_RangeClaimNeedsMetrics(t *testing.T) {
	withMetrics := assemble("serial", "/dev/x", []*pb.FromRadio{
		myInfoFrame(100),
		loraFrame(pb.Config_LoRaConfig_US, pb.Config_LoRaConfig_LONG_FAST, true, true),
		primaryChanFrame(""),
		nodeWithMetrics(100, 0, 0),
	})
	if !hasFinding(withMetrics.Findings, "check", "little or no traffic") {
		t.Errorf("measured 0%% should point at range: %s", findingLevels(withMetrics.Findings))
	}

	noMetrics := assemble("serial", "/dev/x", []*pb.FromRadio{
		myInfoFrame(100),
		loraFrame(pb.Config_LoRaConfig_US, pb.Config_LoRaConfig_LONG_FAST, true, true),
		primaryChanFrame(""),
		nodeFrame(100),
	})
	if hasFinding(noMetrics.Findings, "check", "little or no traffic") {
		t.Errorf("no metrics must not assert a range verdict: %s", findingLevels(noMetrics.Findings))
	}
	if !hasFinding(noMetrics.Findings, "check", "range") {
		t.Errorf("should still give the generic match-a-peer/in-range advice: %s", findingLevels(noMetrics.Findings))
	}
}

// TestDiagnose_ChannelUnknown: a dump with no channel frame must not be reported
// as the default public channel — it was simply never read.
func TestDiagnose_ChannelUnknown(t *testing.T) {
	snap := assemble("serial", "/dev/x", []*pb.FromRadio{
		myInfoFrame(100),
		loraFrame(pb.Config_LoRaConfig_US, pb.Config_LoRaConfig_LONG_FAST, true, true),
		nodeFrame(100),
	})
	if snap.Radio.ChannelKnown {
		t.Error("no channel frame → ChannelKnown should be false")
	}
	if hasFinding(snap.Findings, "check", "default public channel") {
		t.Errorf("must not claim the default public channel when it wasn't read: %s", findingLevels(snap.Findings))
	}
	if !hasFinding(snap.Findings, "check", "not in this read") {
		t.Errorf("should say the channel wasn't in the read: %s", findingLevels(snap.Findings))
	}
}

// TestDiagnose_RegionUnset: the headline case — region not set blocks everything.
func TestDiagnose_RegionUnset(t *testing.T) {
	snap := assemble("serial", "/dev/x", []*pb.FromRadio{
		myInfoFrame(100),
		loraFrame(pb.Config_LoRaConfig_UNSET, pb.Config_LoRaConfig_LONG_FAST, true, true),
		nodeFrame(100),
	})
	if snap.Radio.RegionSet {
		t.Fatal("UNSET region should read as not set")
	}
	if !hasFinding(snap.Findings, "problem", "region") {
		t.Errorf("expected a 'problem' finding about the region, got: %s", findingLevels(snap.Findings))
	}
}

// TestDiagnose_TxDisabled: a listen-only node that never announces itself.
func TestDiagnose_TxDisabled(t *testing.T) {
	snap := assemble("serial", "/dev/x", []*pb.FromRadio{
		myInfoFrame(100),
		loraFrame(pb.Config_LoRaConfig_US, pb.Config_LoRaConfig_LONG_FAST, true, false),
		nodeFrame(100),
	})
	if !hasFinding(snap.Findings, "problem", "transmit") {
		t.Errorf("expected a 'problem' finding about transmit, got: %s", findingLevels(snap.Findings))
	}
}

// TestDiagnose_CapableButAlone: config looks fine, node is alone → a "check"
// finding that names what a peer must match, never a "problem".
func TestDiagnose_CapableButAlone(t *testing.T) {
	snap := assemble("serial", "/dev/x", []*pb.FromRadio{
		myInfoFrame(100),
		loraFrame(pb.Config_LoRaConfig_US, pb.Config_LoRaConfig_LONG_FAST, true, true),
		primaryChanFrame(""),
		nodeFrame(100),
	})
	for _, f := range snap.Findings {
		if f.Level == "problem" {
			t.Errorf("a capable, in-config node must not get a 'problem' finding: %s", findingLevels(snap.Findings))
		}
	}
	if !hasFinding(snap.Findings, "check", "range") {
		t.Errorf("expected a 'check' finding mentioning range, got: %s", findingLevels(snap.Findings))
	}
	// The default public channel must be named as such in the advice.
	if !hasFinding(snap.Findings, "check", "default public channel") {
		t.Errorf("empty primary channel should read as the default public channel: %s", findingLevels(snap.Findings))
	}
}

// TestDiagnose_HealthyWithNeighbours: a node that already sees others is clean.
func TestDiagnose_HealthyWithNeighbours(t *testing.T) {
	snap := assemble("serial", "/dev/x", []*pb.FromRadio{
		myInfoFrame(100),
		loraFrame(pb.Config_LoRaConfig_US, pb.Config_LoRaConfig_LONG_FAST, true, true),
		nodeFrame(100), nodeFrame(200),
	})
	if len(snap.Findings) != 0 {
		t.Errorf("a node with neighbours and good config should have no findings, got: %s", findingLevels(snap.Findings))
	}
}
