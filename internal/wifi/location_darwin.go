//go:build darwin

package wifi

/*
#cgo LDFLAGS: -framework CoreLocation -framework Foundation
#include "location_darwin.h"
*/
import "C"

// locationStatus reports the app's Location Services authorization.
func locationStatus() string { return C.GoString(C.meridian_location_status()) }

// requestLocation asks macOS for Location Services authorization (shows the
// system prompt the first time; otherwise a no-op the user resolves in
// System Settings). Returns immediately.
func requestLocation() { C.meridian_request_location() }
