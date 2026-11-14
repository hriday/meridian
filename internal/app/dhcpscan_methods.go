package app

import "github.com/hriday/meridian/internal/dhcpscan"

// DHCPSupported reports whether DHCP discovery works on this OS (macOS / Linux).
func (a *App) DHCPSupported() bool { return dhcpscan.Supported() }

// DHCPDiscover broadcasts a DHCP DISCOVER and reports every server that answers,
// flagging a rogue (more than one). It needs elevated privileges to bind UDP
// port 68; without them it returns an error saying so.
func (a *App) DHCPDiscover() (dhcpscan.Result, error) {
	return dhcpscan.Run(dhcpscan.DefaultWindow)
}
