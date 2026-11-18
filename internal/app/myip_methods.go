package app

import "github.com/hriday/meridian/internal/myip"

// LocalNetwork reports this host's primary local IPv4, its active interfaces,
// and the default gateway. It is fast and local — the public IP and NAT type
// come from STUNDiscover, which the UI fetches separately so the local facts
// render instantly.
func (a *App) LocalNetwork() (myip.Info, error) {
	return myip.Local()
}
