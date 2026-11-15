//go:build !darwin

package wifi

// Location Services gating of SSIDs is macOS-specific; elsewhere there's
// nothing to request.
func locationStatus() string { return "unsupported" }
func requestLocation()       {}
func currentSSID() string    { return "" }
