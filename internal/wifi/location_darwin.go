//go:build darwin

package wifi

/*
#cgo LDFLAGS: -framework CoreLocation -framework CoreWLAN -framework Foundation
#include <stdlib.h>
#include "location_darwin.h"
*/
import "C"

import "unsafe"

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
