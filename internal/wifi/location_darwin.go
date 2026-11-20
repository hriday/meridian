//go:build darwin

package wifi

/*
#cgo LDFLAGS: -framework CoreLocation -framework CoreWLAN -framework Foundation
#include <stdlib.h>
#include "location_darwin.h"
*/
import "C"

import (
	"encoding/json"
	"unsafe"
)

// locationStatus reports the app's Location Services authorization.
func locationStatus() string { return C.GoString(C.meridian_location_status()) }

// requestLocation asks macOS for Location Services authorization (shows the
// system prompt the first time; otherwise a no-op the user resolves in
// System Settings). Returns immediately.
func requestLocation() { C.meridian_request_location() }

// currentSSID reads the connected network's SSID via CoreWLAN in this process,
// where the app's own Location grant applies (unlike the separate
// system_profiler process). Empty when not connected or not authorized.
func currentSSID() string {
	c := C.meridian_current_ssid()
	if c == nil {
		return ""
	}
	defer C.free(unsafe.Pointer(c))
	return C.GoString(c)
}

// scanNeighbors runs an in-process CoreWLAN scan and returns the neighboring
// networks with real SSIDs (the app's Location grant applies in-process).
// Entries with an empty SSID — hidden networks, or every network when the grant
// is absent — are dropped, so an unauthorized scan yields nil and the caller
// falls back to system_profiler. Noise/SNR/security are not read here (CoreWLAN
// scan results don't carry neighbor noise); the UI shows those for the
// connected network only.
func scanNeighbors() []Network {
	c := C.meridian_scan_networks()
	if c == nil {
		return nil
	}
	defer C.free(unsafe.Pointer(c))
	var raw []struct {
		SSID    string `json:"ssid"`
		Channel int    `json:"channel"`
		Band    string `json:"band"`
		Width   string `json:"width"`
		RSSI    int    `json:"rssi"`
	}
	if err := json.Unmarshal([]byte(C.GoString(c)), &raw); err != nil {
		return nil
	}
	out := make([]Network, 0, len(raw))
	for _, r := range raw {
		if r.SSID == "" { // hidden, or redacted because the grant is absent
			continue
		}
		out = append(out, Network{SSID: r.SSID, Channel: r.Channel, Band: r.Band, Width: r.Width, RSSI: r.RSSI})
	}
	return out
}
