package app

import "github.com/hriday/meridian/internal/collector"

// SyslogStart binds the syslog receiver to a UDP port (0 → an OS-chosen port);
// starting while already running restarts it on the new port. A bind failure
// (port taken or privileged) is returned as an error.
func (a *App) SyslogStart(port int) error { return a.syslog.Start(port) }

// SyslogStop stops the receiver and discards its buffer.
func (a *App) SyslogStop() { a.syslog.Stop() }

// SyslogPoll returns the running state plus the messages newer than afterSeq.
func (a *App) SyslogPoll(afterSeq int) collector.Poll[collector.SyslogMsg] {
	return a.syslog.PollSince(afterSeq)
}
