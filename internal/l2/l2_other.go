//go:build !darwin && !linux && !windows

package l2

import "fmt"

const supported = false

// Conn is unavailable on this OS.
type Conn struct{}

// Open always fails on unsupported platforms.
func Open(iface string) (*Conn, error) {
	return nil, fmt.Errorf("raw packet access isn't supported on this operating system yet")
}

func (c *Conn) Close() error                { return nil }
func (c *Conn) Write(frame []byte) error    { return fmt.Errorf("unsupported") }
func (c *Conn) Read() ([][]byte, error)     { return nil, fmt.Errorf("unsupported") }
