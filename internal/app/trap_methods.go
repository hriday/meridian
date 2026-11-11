package app

import (
	"github.com/hriday/meridian/internal/collector"
	"github.com/hriday/meridian/internal/snmp"
)

// TrapStart binds the SNMP trap receiver to a UDP port (0 → an OS-chosen port);
// starting while running restarts it on the new port. A bind failure is returned.
func (a *App) TrapStart(port int) error { return a.traps.Start(port) }

// TrapStop stops the receiver and discards its buffer.
func (a *App) TrapStop() { a.traps.Stop() }

// TrapPoll returns the running state plus the traps newer than afterSeq.
func (a *App) TrapPoll(afterSeq int) collector.Poll[snmp.TrapMsg] {
	return a.traps.PollSince(afterSeq)
}
