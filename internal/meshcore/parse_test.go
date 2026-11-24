package meshcore

import (
	"encoding/binary"
	"testing"
)

func buildSelfInfo(freqMHz, bwKHz float64, sf, cr, tx int, name string, lat, lon float64) []byte {
	p := make([]byte, 58)
	p[0] = respSelfInfo
	p[1] = 1 // adv type
	p[2] = byte(int8(tx))
	p[4], p[5] = 0xAB, 0xCD // public key bytes
	binary.LittleEndian.PutUint32(p[36:], uint32(int32(lat*1e6)))
	binary.LittleEndian.PutUint32(p[40:], uint32(int32(lon*1e6)))
	binary.LittleEndian.PutUint32(p[48:], uint32(freqMHz*1000))
	binary.LittleEndian.PutUint32(p[52:], uint32(bwKHz*1000))
	p[56], p[57] = byte(sf), byte(cr)
	return append(p, []byte(name)...)
}

func buildDeviceInfo(fwVer int, model, version string) []byte {
	p := make([]byte, 80)
	p[0] = respDeviceInfo
	p[1] = byte(fwVer)
	p[2] = 50 // max_contacts / 2
	p[3] = 8  // max_channels
	copy(p[20:60], model)
	copy(p[60:80], version)
	return p
}

func buildContact(name string, advType byte, pathLen int, lastSeen uint32, lat, lon float64) []byte {
	p := make([]byte, 148)
	p[0] = respContact
	p[33] = advType
	p[35] = byte(int8(pathLen))
	copy(p[100:132], name)
	binary.LittleEndian.PutUint32(p[132:], lastSeen)
	binary.LittleEndian.PutUint32(p[136:], uint32(int32(lat*1e6)))
	binary.LittleEndian.PutUint32(p[140:], uint32(int32(lon*1e6)))
	return p
}

func TestParseSelfInfo(t *testing.T) {
	si, err := parseSelfInfo(buildSelfInfo(910.525, 250, 11, 5, 22, "WadaNode", 37.5, -122.3))
	if err != nil {
		t.Fatal(err)
	}
	if si.Name != "WadaNode" {
		t.Errorf("name = %q", si.Name)
	}
	if si.Radio.FreqMHz != 910.525 {
		t.Errorf("freq = %v, want 910.525", si.Radio.FreqMHz)
	}
	if si.Radio.BandwidthKHz != 250 || si.Radio.SpreadFactor != 11 || si.Radio.CodingRate != 5 {
		t.Errorf("radio params wrong: %+v", si.Radio)
	}
	if si.Radio.TxPowerDBm != 22 {
		t.Errorf("tx power = %d, want 22", si.Radio.TxPowerDBm)
	}
	if !si.HasPosition || si.Lat < 37.49 || si.Lat > 37.51 {
		t.Errorf("position wrong: %+v", si)
	}
	if si.PubKey != "abcd000000000000" {
		t.Errorf("pubkey hex = %q", si.PubKey)
	}
}

func TestParseSelfInfo_ShortFrameDegrades(t *testing.T) {
	// A frame truncated before the radio block must not panic; it yields zeros.
	si, err := parseSelfInfo([]byte{respSelfInfo, 1, 20})
	if err != nil {
		t.Fatal(err)
	}
	if si.Radio.FreqMHz != 0 || si.Name != "" {
		t.Errorf("short frame should degrade to zero values: %+v", si)
	}
}

func TestParseDeviceInfo(t *testing.T) {
	di, err := parseDeviceInfo(buildDeviceInfo(10, "LilyGo T-Deck", "v1.2.3"))
	if err != nil {
		t.Fatal(err)
	}
	if di.FirmwareVer != 10 || di.Model != "LilyGo T-Deck" || di.Version != "v1.2.3" {
		t.Errorf("device info wrong: %+v", di)
	}
	if di.MaxContacts != 100 || di.MaxChannels != 8 {
		t.Errorf("counts wrong: %+v", di)
	}
}

func TestParseContact(t *testing.T) {
	c, err := parseContact(buildContact("Hilltop", 2, 3, 1700000000, 37.6, -122.4))
	if err != nil {
		t.Fatal(err)
	}
	if c.Name != "Hilltop" || c.Type != "Repeater" || c.PathLen != 3 || c.LastSeen != 1700000000 {
		t.Errorf("contact wrong: %+v", c)
	}
	if !c.HasPosition || c.Lon > -122.39 || c.Lon < -122.41 {
		t.Errorf("contact position wrong: %+v", c)
	}
	// A flood-reached contact (out_path_len -1) reports no stored path.
	flood, _ := parseContact(buildContact("Far", 1, -1, 0, 0, 0))
	if flood.PathLen != -1 || flood.Type != "Companion" || flood.HasPosition {
		t.Errorf("flood contact wrong: %+v", flood)
	}
}
