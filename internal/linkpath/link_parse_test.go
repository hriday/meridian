package linkpath

import (
	"testing"
	"testing/fstest"
)

func TestParseIfconfigMedia(t *testing.T) {
	wired := "en5: flags=8863 mtu 1500\n\tether a4:b1:c2:d3:e4:f5\n\tmedia: autoselect (1000baseT <full-duplex,flow-control>)\n\tstatus: active\n"
	ls := parseIfconfigMedia(wired)
	if ls.speedMbps != 1000 || ls.duplex != "full" || !ls.known {
		t.Errorf("wired gigabit full-duplex misread: %+v", ls)
	}

	halfhub := "en5: flags=8863 mtu 1500\n\tmedia: autoselect (100baseTX <half-duplex>)\n\tstatus: active\n"
	ls = parseIfconfigMedia(halfhub)
	if ls.speedMbps != 100 || ls.duplex != "half" {
		t.Errorf("100M half-duplex misread: %+v", ls)
	}

	wifi := "en0: flags=8863 mtu 1500\n\tmedia: autoselect\n\tstatus: active\n"
	ls = parseIfconfigMedia(wifi)
	if ls.known || ls.duplex != "" || ls.speedMbps != 0 {
		t.Errorf("a Wi-Fi/no-media link should read as unknown: %+v", ls)
	}

	tenG := "en5: flags=8863 mtu 1500\n\tmedia: autoselect (10Gbase-T <full-duplex>)\n\tstatus: active\n"
	ls = parseIfconfigMedia(tenG)
	if ls.speedMbps != 10000 || ls.duplex != "full" {
		t.Errorf("10Gbase-T should read as 10000 Mb/s full duplex, got %+v", ls)
	}
}

func TestParseHardwarePorts(t *testing.T) {
	out := "Hardware Port: Wi-Fi\nDevice: en0\nEthernet Address: a1:b2\n\nHardware Port: Thunderbolt Ethernet\nDevice: en5\nEthernet Address: c3:d4\n"
	m := parseHardwarePorts(out)
	if !isWirelessPort(m["en0"]) {
		t.Errorf("en0 should map to a Wi-Fi port, got %q", m["en0"])
	}
	if isWirelessPort(m["en5"]) {
		t.Errorf("en5 (Thunderbolt Ethernet) should not be wireless, got %q", m["en5"])
	}
}

func TestParseLinuxLink(t *testing.T) {
	fsys := fstest.MapFS{
		"eth0/speed":  {Data: []byte("1000\n")},
		"eth0/duplex": {Data: []byte("full\n")},
		// wlan0 is wireless and reports no usable speed/duplex.
		"wlan0/speed":        {Data: []byte("-1\n")},
		"wlan0/duplex":       {Data: []byte("unknown\n")},
		"wlan0/wireless/foo": {Data: []byte("")}, // makes wlan0/wireless a directory
	}
	eth := parseLinuxLink(fsys, "eth0")
	if eth.speedMbps != 1000 || eth.duplex != "full" || eth.wireless || !eth.known {
		t.Errorf("eth0 misread: %+v", eth)
	}
	w := parseLinuxLink(fsys, "wlan0")
	if !w.wireless || w.known || w.duplex != "" || w.speedMbps != 0 {
		t.Errorf("wlan0 should read wireless + unknown link: %+v", w)
	}
}

func TestParseNetAdapter(t *testing.T) {
	wired := `{"Name":"Ethernet","Speed":1000000000,"FullDuplex":true,"PhysicalMediaType":"802.3"}`
	ls := parseNetAdapter([]byte(wired))
	if ls.speedMbps != 1000 || ls.duplex != "full" || ls.wireless {
		t.Errorf("wired gigabit misread: %+v", ls)
	}

	half := `{"Name":"Ethernet","Speed":100000000,"FullDuplex":false,"PhysicalMediaType":"802.3"}`
	ls = parseNetAdapter([]byte(half))
	if ls.duplex != "half" || ls.speedMbps != 100 {
		t.Errorf("100M half-duplex misread: %+v", ls)
	}

	wifi := `{"Name":"Wi-Fi","Speed":866700000,"FullDuplex":null,"PhysicalMediaType":"Native 802.11"}`
	ls = parseNetAdapter([]byte(wifi))
	if !ls.wireless || ls.duplex != "" {
		t.Errorf("Wi-Fi adapter should read wireless with no duplex: %+v", ls)
	}
}
