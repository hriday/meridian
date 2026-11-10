package app

import "github.com/hriday/meridian/internal/netif"

// Interfaces lists the host's network interfaces (name, MAC, MTU, flags, and
// IPv4/IPv6 addresses). Local and read-only.
func (a *App) Interfaces() ([]netif.Iface, error) {
	return netif.List()
}
