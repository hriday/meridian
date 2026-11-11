package collector

import (
	"net"
	"net/netip"
	"sync"
)

// maxDatagram bounds one UDP read (syslog/traps are far smaller).
const maxDatagram = 64 * 1024

// Poll is the snapshot the frontend polls: the running state plus the messages
// newer than the caller's cursor. Epoch changes on restart so a stale cursor is
// detected.
type Poll[T any] struct {
	Running bool         `json:"running"`
	Port    int          `json:"port"`
	Epoch   int          `json:"epoch"`
	Total   int          `json:"total"`
	Dropped int          `json:"dropped"`
	Items   []Stamped[T] `json:"items"`
}

// Collector binds a UDP socket, parses each datagram into a T, and buffers it in
// a bounded ring. One instance is a singleton per kind on the App.
type Collector[T any] struct {
	cap   int
	parse func(datagram []byte, src netip.Addr) []T

	opMu    sync.Mutex // serializes Start/Stop so their sections can't interleave
	mu      sync.Mutex // guards the fields below (Poll reads them)
	running bool
	port    int
	epoch   int
	ring    *Ring[T]
	conn    *net.UDPConn
	done    chan struct{}
}

// New returns a collector that buffers up to capacity messages, parsing each
// datagram into at most one message (parse returns ok=false to drop it).
func New[T any](capacity int, parse func(datagram []byte, src netip.Addr) (T, bool)) *Collector[T] {
	return NewMulti[T](capacity, func(b []byte, a netip.Addr) []T {
		if m, ok := parse(b, a); ok {
			return []T{m}
		}
		return nil
	})
}

// NewMulti returns a collector whose parse yields zero or more messages per
// datagram (e.g. a NetFlow packet carrying many flow records).
func NewMulti[T any](capacity int, parse func(datagram []byte, src netip.Addr) []T) *Collector[T] {
	return &Collector[T]{cap: capacity, parse: parse}
}

// Start binds 0.0.0.0:port (port 0 → an OS-chosen port) and begins collecting.
// If already running it restarts on the new port, resetting the buffer and
// bumping the epoch. A bind failure (port taken/privileged) is returned and the
// collector stays stopped.
func (c *Collector[T]) Start(port int) error {
	c.opMu.Lock() // whole Start is atomic against another Start/Stop
	defer c.opMu.Unlock()
	c.stopLocked() // restart semantics; no-op when not running
	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: port})
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.conn = conn
	c.ring = NewRing[T](c.cap)
	c.running = true
	c.epoch++
	c.port = conn.LocalAddr().(*net.UDPAddr).Port
	c.done = make(chan struct{})
	done, ring, parse := c.done, c.ring, c.parse
	c.mu.Unlock()

	go func() {
		defer close(done)
		buf := make([]byte, maxDatagram)
		for {
			n, ap, rerr := conn.ReadFromUDPAddrPort(buf)
			if rerr != nil {
				return // conn closed by Stop
			}
			ring.AddAll(parse(buf[:n], ap.Addr().Unmap())) // unmap ::ffff:v4 → v4
		}
	}()
	return nil
}

// Stop closes the socket, blocks until the read loop exits, and discards the
// buffer. It holds no field lock while waiting, so it can't deadlock OnShutdown.
func (c *Collector[T]) Stop() {
	c.opMu.Lock()
	defer c.opMu.Unlock()
	c.stopLocked()
}

// stopLocked stops the collector; the caller must hold opMu.
func (c *Collector[T]) stopLocked() {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		return
	}
	conn, done := c.conn, c.done
	c.running = false
	c.conn = nil
	c.ring = nil
	c.port = 0
	c.mu.Unlock()

	conn.Close()
	<-done
}

// Snapshot returns the buffered values matching keep (copied under the ring
// lock), or nil when the collector isn't running. The caller processes them
// outside the lock.
func (c *Collector[T]) Snapshot(keep func(T) bool) []T {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ring == nil {
		return nil
	}
	return c.ring.Values(keep)
}

// PollSince returns the current state plus the buffered messages with Seq >
// afterSeq (up to the full ring). The item copy happens under the lock; the
// caller marshals the result outside it.
func (c *Collector[T]) PollSince(afterSeq int) Poll[T] {
	c.mu.Lock()
	defer c.mu.Unlock()
	p := Poll[T]{Running: c.running, Port: c.port, Epoch: c.epoch}
	if c.ring != nil {
		p.Items, p.Total, p.Dropped = c.ring.Snapshot(afterSeq)
	}
	return p
}
