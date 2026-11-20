package app

import (
	"context"
	"time"

	"github.com/hriday/meridian/internal/meshtastic"
)

// MeshtasticSnapshot connects to a Meshtastic LoRa mesh node over USB serial or
// TCP and returns a point-in-time read of the node and the mesh it sees — node
// database, per-neighbour signal, channel airtime and positions. Read-only; it
// only asks the node to dump its config. User-initiated.
func (a *App) MeshtasticSnapshot(target string) (meshtastic.Snapshot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return meshtastic.Read(ctx, target)
}
