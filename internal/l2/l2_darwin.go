//go:build darwin

package l2

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

const supported = true

// Conn is a BPF capture bound to an interface, able to read and write frames.
type Conn struct {
	fd      int
	bufLen  int
	readBuf []byte
}

// Open binds a /dev/bpf device to iface.
func Open(iface string) (*Conn, error) {
	var fd int
	var err error
	for i := 0; i < 64; i++ {
		fd, err = unix.Open(fmt.Sprintf("/dev/bpf%d", i), unix.O_RDWR, 0)
		if err == nil {
			break
		}
		if errors.Is(err, unix.EACCES) || errors.Is(err, unix.EPERM) {
			return nil, ErrNeedsPrivilege
		}
		// EBUSY: device in use, try the next one.
	}
	if err != nil {
		return nil, fmt.Errorf("couldn't open a BPF device: %w", err)
	}

	bl := int32(32768)
	_ = ioctlPtr(fd, unix.BIOCSBLEN, unsafe.Pointer(&bl)) // must precede BIOCSETIF
	if err := ioctlPtr(fd, unix.BIOCGBLEN, unsafe.Pointer(&bl)); err != nil || bl <= 0 {
		bl = 4096
	}

	var ifr ifreq
	copy(ifr.name[:], iface)
	if err := ioctlPtr(fd, unix.BIOCSETIF, unsafe.Pointer(&ifr)); err != nil {
		unix.Close(fd)
		return nil, fmt.Errorf("couldn't bind BPF to %s: %w", iface, err)
	}

	one := int32(1)
	_ = ioctlPtr(fd, unix.BIOCIMMEDIATE, unsafe.Pointer(&one))  // deliver frames promptly
	_ = ioctlPtr(fd, unix.BIOCSHDRCMPLT, unsafe.Pointer(&one))  // we supply the full ethernet header on write
	tv := unix.Timeval{Sec: 1}                                  // 1s read timeout so callers can poll
	_ = ioctlPtr(fd, unix.BIOCSRTIMEOUT, unsafe.Pointer(&tv))

	return &Conn{fd: fd, bufLen: int(bl), readBuf: make([]byte, int(bl))}, nil
}

func (c *Conn) Close() error { return unix.Close(c.fd) }

// Write sends one complete Ethernet frame.
func (c *Conn) Write(frame []byte) error {
	_, err := unix.Write(c.fd, frame)
	return err
}

// Read returns the frames available this interval, or ErrReadTimeout if none.
func (c *Conn) Read() ([][]byte, error) {
	n, err := unix.Read(c.fd, c.readBuf)
	if err != nil {
		if errors.Is(err, unix.EAGAIN) || errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, syscall.EINTR) {
			return nil, ErrReadTimeout
		}
		return nil, err
	}
	if n == 0 {
		return nil, ErrReadTimeout
	}

	var frames [][]byte
	hdrSize := int(unsafe.Sizeof(unix.BpfHdr{}))
	for off := 0; off+hdrSize <= n; {
		hdr := (*unix.BpfHdr)(unsafe.Pointer(&c.readBuf[off]))
		dataOff := off + int(hdr.Hdrlen)
		end := dataOff + int(hdr.Caplen)
		if dataOff < off || end > n {
			break
		}
		f := make([]byte, int(hdr.Caplen))
		copy(f, c.readBuf[dataOff:end])
		frames = append(frames, f)
		off += bpfWordAlign(int(hdr.Hdrlen) + int(hdr.Caplen))
	}
	if len(frames) == 0 {
		return nil, ErrReadTimeout
	}
	return frames, nil
}

// ifreq is the subset of struct ifreq needed for BIOCSETIF (just the name).
type ifreq struct {
	name [16]byte
	pad  [16]byte
}

func ioctlPtr(fd int, req uint, arg unsafe.Pointer) error {
	_, _, e := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), uintptr(req), uintptr(arg))
	if e != 0 {
		return e
	}
	return nil
}

func bpfWordAlign(x int) int { return (x + 3) &^ 3 }
