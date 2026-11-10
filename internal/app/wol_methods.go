package app

import (
	"fmt"

	"github.com/hriday/meridian/internal/wol"
)

// WakeOnLAN broadcasts a Wake-on-LAN magic packet for mac to bcast:port
// (defaults 255.255.255.255:9). It returns a plain confirmation that the packet
// was sent — which is not a guarantee the machine woke (WoL must be enabled, and
// broadcasts don't cross subnets).
func (a *App) WakeOnLAN(mac, bcast string, port int) (string, error) {
	if err := wol.Send(a.lookupCtx(), mac, bcast, port); err != nil {
		return "", err
	}
	b := bcast
	if b == "" {
		b = "255.255.255.255"
	}
	p := port
	if p <= 0 {
		p = 9
	}
	return fmt.Sprintf("Magic packet sent to %s via %s:%d", mac, b, p), nil
}
