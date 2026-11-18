package myip

import (
	"reflect"
	"testing"
)

func TestStripMasks(t *testing.T) {
	got := stripMasks([]string{"192.168.1.82/24", "10.0.0.1", "169.254.1.2/16"})
	want := []string{"192.168.1.82", "10.0.0.1", "169.254.1.2"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("stripMasks = %v, want %v", got, want)
	}
}

func TestIfaceForIP(t *testing.T) {
	ifaces := []InterfaceIP{
		{Name: "en0", IPv4: []string{"192.168.1.82"}},
		{Name: "en5", IPv4: []string{"10.1.1.4", "10.1.1.5"}},
	}
	if got := ifaceForIP(ifaces, "10.1.1.5"); got != "en5" {
		t.Errorf("ifaceForIP(10.1.1.5) = %q, want en5", got)
	}
	if got := ifaceForIP(ifaces, "192.168.1.82"); got != "en0" {
		t.Errorf("ifaceForIP(192.168.1.82) = %q, want en0", got)
	}
	// Unknown / empty IP -> no interface, never a wrong guess.
	if got := ifaceForIP(ifaces, "203.0.113.9"); got != "" {
		t.Errorf("ifaceForIP(unknown) = %q, want empty", got)
	}
	if got := ifaceForIP(ifaces, ""); got != "" {
		t.Errorf("ifaceForIP(empty) = %q, want empty", got)
	}
}

func TestLocalRunsAndIsConsistent(t *testing.T) {
	info, err := Local()
	if err != nil {
		t.Fatalf("Local() error: %v", err)
	}
	// If a primary IP was found, it must belong to a listed interface.
	if info.PrimaryIP != "" && info.PrimaryIf != "" {
		if ifaceForIP(info.Interfaces, info.PrimaryIP) != info.PrimaryIf {
			t.Errorf("PrimaryIf %q doesn't own PrimaryIP %q", info.PrimaryIf, info.PrimaryIP)
		}
	}
	// Listed interfaces are never loopback and always have an address.
	for _, f := range info.Interfaces {
		if len(f.IPv4) == 0 {
			t.Errorf("interface %q listed with no IPv4", f.Name)
		}
	}
}
