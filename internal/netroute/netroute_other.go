//go:build !darwin && !linux

package netroute

// defaultGateway is unsupported on this platform; "" means "unknown", and the
// UI simply omits the gateway rather than showing a guess.
func defaultGateway() string { return "" }
