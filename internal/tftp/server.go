// Package tftp is Meridian's receive-only, ephemeral TFTP "grab" server: it
// catches files devices push (WRQ) into a bounded in-memory store and refuses
// read requests (RRQ) so it never serves files. Built on the pin/tftp library.
package tftp

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"path"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	ptftp "github.com/pin/tftp/v3"
)

var (
	maxTftpFile  = 16 << 20 // per-file cap (bytes)
	maxTftpFiles = 64       // keep at most this many, oldest evicted
)

// File is the metadata of a received file (content is held internally).
type File struct {
	ID     int       `json:"id"`
	Name   string    `json:"name"`
	Source string    `json:"source"`
	Size   int       `json:"size"`
	T      time.Time `json:"t"`
}

type stored struct {
	File
	content []byte
}

// Server is a receive-only TFTP server with an in-memory, bounded store.
type Server struct {
	mu      sync.Mutex
	running bool
	port    int
	srv     *ptftp.Server
	done    chan struct{}
	files   []stored
	nextID  int
}

// New returns a stopped server.
func New() *Server { return &Server{} }

// Start binds 0.0.0.0:port (0 → OS-chosen) and serves; starting while running
// restarts it. A bind failure is returned.
func (s *Server) Start(port int) error {
	s.Stop()
	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: port})
	if err != nil {
		return err
	}
	srv := ptftp.NewServer(denyRead, s.onWrite)
	srv.SetTimeout(5 * time.Second)

	s.mu.Lock()
	s.srv = srv
	s.running = true
	s.port = conn.LocalAddr().(*net.UDPAddr).Port
	s.files = nil
	s.done = make(chan struct{})
	done := s.done
	s.mu.Unlock()

	go func() { defer close(done); _ = srv.Serve(conn) }() // Serve owns & closes conn on Shutdown
	return nil
}

// Stop stops the server (closing the socket), waits for it to exit, and clears
// the store.
func (s *Server) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	srv, done := s.srv, s.done
	s.running = false
	s.srv = nil
	s.port = 0
	s.files = nil
	s.mu.Unlock()

	srv.Shutdown() // closes the conn + waits for in-flight transfers
	<-done
}

// List returns the running state, port, and the received files' metadata.
func (s *Server) List() (bool, int, []File) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]File, len(s.files))
	for i := range s.files {
		out[i] = s.files[i].File
	}
	return s.running, s.port, out
}

// Content returns a copy of a received file's bytes.
func (s *Server) Content(id int) ([]byte, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.files {
		if s.files[i].ID == id {
			c := make([]byte, len(s.files[i].content))
			copy(c, s.files[i].content)
			return c, true
		}
	}
	return nil, false
}

// Preview returns a display string for a file: its text (capped) or hex when it
// isn't printable.
func (s *Server) Preview(id, max int) (string, bool) {
	c, ok := s.Content(id)
	if !ok {
		return "", false
	}
	if len(c) > max {
		c = c[:max]
	}
	str := string(c)
	if !utf8.ValidString(str) || hasControl(str) {
		return hex.EncodeToString(c), true
	}
	return str, true
}

func denyRead(string, io.ReaderFrom) error {
	return fmt.Errorf("receive-only TFTP grab: read requests are refused")
}

func (s *Server) onWrite(filename string, wt io.WriterTo) error {
	var source string
	if ra, ok := wt.(interface{ RemoteAddr() net.UDPAddr }); ok {
		source = ra.RemoteAddr().IP.String()
	}
	var buf bytes.Buffer
	if _, err := wt.WriteTo(&limitWriter{w: &buf, max: maxTftpFile}); err != nil {
		return err // oversize or aborted → not stored
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	s.files = append(s.files, stored{
		File:    File{ID: s.nextID, Name: baseName(filename), Source: source, Size: buf.Len(), T: time.Now()},
		content: buf.Bytes(),
	})
	if len(s.files) > maxTftpFiles {
		s.files = append(s.files[:0], s.files[len(s.files)-maxTftpFiles:]...)
	}
	return nil
}

// limitWriter errors once more than max bytes are written.
type limitWriter struct {
	w   io.Writer
	max int
	n   int
}

func (l *limitWriter) Write(p []byte) (int, error) {
	if l.n+len(p) > l.max {
		return 0, fmt.Errorf("file exceeds the %d-byte limit", l.max)
	}
	n, err := l.w.Write(p)
	l.n += n
	return n, err
}

// baseName strips any path and control characters from a client-supplied name.
func baseName(name string) string {
	b := path.Base(name)
	b = string([]rune(removeControl(b)))
	if b == "" || b == "." || b == "/" {
		return "upload"
	}
	return b
}

func removeControl(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if !unicode.IsControl(r) {
			out = append(out, r)
		}
	}
	return string(out)
}

func hasControl(s string) bool {
	for _, r := range s {
		if r == '\t' || r == '\n' || r == '\r' {
			continue
		}
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}
