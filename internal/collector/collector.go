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
	parse func(datagram []byte, src netip.Addr) (T, bool)

	mu      sync.Mutex
	running bool
	port    int
	epoch   int
	ring    *Ring[T]
	conn    *net.UDPConn
	done    chan struct{}
}

// New returns a collector that buffers up to capacity messages, parsing each
// datagram with parse (which returns ok=false to drop it).
func New[T any](capacity int, parse func(datagram []byte, src netip.Addr) (T, bool)) *Collector[T] {
	return &Collector[T]{cap: capacity, parse: parse}
}

// Start binds 0.0.0.0:port (port 0 → an OS-chosen port) and begins collecting.
// If already running it restarts on the new port, resetting the buffer and
// bumping the epoch. A bind failure (port taken/privileged) is returned and the
// collector stays stopped.
func (c *Collector[T]) Start(port int) error {
	c.Stop() // restart semantics; no-op when not running
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
			if m, ok := parse(buf[:n], ap.Addr().Unmap()); ok { // unmap ::ffff:v4 → v4
				ring.Add(m)
			}
		}
	}()
	return nil
}

// Stop closes the socket, blocks until the read loop exits, and discards the
// buffer. It holds no lock while waiting, so it can't deadlock OnShutdown.
func (c *Collector[T]) Stop() {
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
