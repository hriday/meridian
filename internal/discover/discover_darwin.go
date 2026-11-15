//go:build darwin

package discover

/*
#include <dns_sd.h>
#include <stdlib.h>
#include <string.h>
#include <errno.h>
#include <arpa/inet.h>
#include <sys/select.h>
#include <sys/time.h>

// Go-exported callbacks (defined in Go with //export, called back from the
// DNS-SD runtime). Declared non-const to match cgo's generated signatures; the
// wrappers below cast them to the API's reply types, which silences the const
// difference.
extern void goBrowseCB(DNSServiceRef, DNSServiceFlags, uint32_t, DNSServiceErrorType,
    char *, char *, char *, void *);
extern void goResolveCB(DNSServiceRef, DNSServiceFlags, uint32_t, DNSServiceErrorType,
    char *, char *, uint16_t, uint16_t, unsigned char *, void *);
extern void goAddrCB(DNSServiceRef, DNSServiceFlags, uint32_t, DNSServiceErrorType,
    char *, struct sockaddr *, uint32_t, void *);

static DNSServiceErrorType startBrowse(DNSServiceRef shared, const char *regtype, uintptr_t ctx) {
    DNSServiceRef ref = shared;
    return DNSServiceBrowse(&ref, kDNSServiceFlagsShareConnection,
        kDNSServiceInterfaceIndexAny, regtype, "local.",
        (DNSServiceBrowseReply)goBrowseCB, (void *)ctx);
}

static DNSServiceErrorType startResolve(DNSServiceRef shared, uint32_t ifIndex,
    const char *name, const char *regtype, const char *domain, uintptr_t ctx) {
    DNSServiceRef ref = shared;
    return DNSServiceResolve(&ref, kDNSServiceFlagsShareConnection, ifIndex,
        name, regtype, domain, (DNSServiceResolveReply)goResolveCB, (void *)ctx);
}

static DNSServiceErrorType startAddr(DNSServiceRef shared, uint32_t ifIndex,
    const char *hostname, uintptr_t ctx) {
    DNSServiceRef ref = shared;
    return DNSServiceGetAddrInfo(&ref, kDNSServiceFlagsShareConnection, ifIndex,
        kDNSServiceProtocol_IPv4, hostname,
        (DNSServiceGetAddrInfoReply)goAddrCB, (void *)ctx);
}

// pump drives the shared connection's socket for up to timeoutMs, dispatching
// results (which invoke the Go callbacks) as they arrive.
static void pump(DNSServiceRef shared, int timeoutMs) {
    int fd = DNSServiceRefSockFD(shared);
    if (fd < 0) return;
    struct timeval start;
    gettimeofday(&start, NULL);
    for (;;) {
        struct timeval now;
        gettimeofday(&now, NULL);
        long elapsed = (now.tv_sec - start.tv_sec) * 1000 +
                       (now.tv_usec - start.tv_usec) / 1000;
        long remain = timeoutMs - elapsed;
        if (remain <= 0) break;
        fd_set rset;
        FD_ZERO(&rset);
        FD_SET(fd, &rset);
        struct timeval tv;
        tv.tv_sec = remain / 1000;
        tv.tv_usec = (remain % 1000) * 1000;
        int r = select(fd + 1, &rset, NULL, NULL, &tv);
        if (r > 0 && FD_ISSET(fd, &rset)) {
            if (DNSServiceProcessResult(shared) != kDNSServiceErr_NoError) break;
        } else if (r < 0) {
            if (errno == EINTR) continue;
            break;
        }
    }
}

static uint16_t ntohs_go(uint16_t x) { return ntohs(x); }

// sockaddrV4 writes the dotted IPv4 string of sa into out; returns 1 on success.
static int sockaddrV4(const struct sockaddr *sa, char *out, int outlen) {
    if (sa == NULL || sa->sa_family != AF_INET) return 0;
    const struct sockaddr_in *sin = (const struct sockaddr_in *)sa;
    if (inet_ntop(AF_INET, &sin->sin_addr, out, outlen) == NULL) return 0;
    return 1;
}
*/
import "C"

import (
	"context"
	"fmt"
	"net"
	"runtime/cgo"
	"strings"
	"time"
	"unsafe"
)

// browseSession accumulates results across all the DNS-SD operations sharing one
// connection. Every callback runs synchronously inside pump() on the same
// goroutine, so no locking is needed. Services are held as pointers (stable
// across slice growth) so an outstanding address lookup can fill one in later.
type browseSession struct {
	shared  C.DNSServiceRef
	root    cgo.Handle
	svcs    []*Service
	handles []cgo.Handle
}

func (s *browseSession) newHandle(v any) C.uintptr_t {
	h := cgo.NewHandle(v)
	s.handles = append(s.handles, h)
	return C.uintptr_t(h)
}

type resolveCtx struct {
	sess     *browseSession
	instance string
	regtype  string
}

type addrCtx struct {
	svc *Service
}

// systemMDNS browses the LAN through the macOS system responder (mDNSResponder),
// which holds the aggregated multicast view a raw socket cannot get here.
func systemMDNS(ctx context.Context, window time.Duration) ([]Service, error) {
	var shared C.DNSServiceRef
	if e := C.DNSServiceCreateConnection(&shared); e != C.kDNSServiceErr_NoError {
		return nil, fmt.Errorf("DNSServiceCreateConnection failed (%d)", int(e))
	}
	defer C.DNSServiceRefDeallocate(shared)

	sess := &browseSession{shared: shared}
	sess.root = cgo.NewHandle(sess)
	sess.handles = append(sess.handles, sess.root)
	defer func() {
		for _, h := range sess.handles {
			h.Delete()
		}
	}()

	rootCtx := C.uintptr_t(sess.root)
	for _, t := range mdnsTypes {
		regtype := strings.TrimSuffix(strings.TrimSuffix(t, "."), ".local")
		cr := C.CString(regtype)
		C.startBrowse(shared, cr, rootCtx)
		C.free(unsafe.Pointer(cr))
	}

	ms := int(window / time.Millisecond)
	if ms < 1000 {
		ms = 1000
	}
	C.pump(shared, C.int(ms))

	out := make([]Service, 0, len(sess.svcs))
	for _, s := range sess.svcs {
		out = append(out, *s)
	}
	return dedupeServices(out), nil
}

//export goBrowseCB
func goBrowseCB(sdRef C.DNSServiceRef, flags C.DNSServiceFlags, ifIndex C.uint32_t,
	errCode C.DNSServiceErrorType, serviceName *C.char, regtype *C.char,
	replyDomain *C.char, ctx unsafe.Pointer) {
	if errCode != C.kDNSServiceErr_NoError {
		return
	}
	if flags&C.kDNSServiceFlagsAdd == 0 {
		return // a "remove" event; we only track current presence
	}
	sess, ok := cgo.Handle(uintptr(ctx)).Value().(*browseSession)
	if !ok {
		return
	}
	rc := &resolveCtx{
		sess:     sess,
		instance: C.GoString(serviceName),
		regtype:  strings.TrimSuffix(C.GoString(regtype), "."),
	}
	h := sess.newHandle(rc)
	C.startResolve(sess.shared, ifIndex, serviceName, regtype, replyDomain, h)
}

//export goResolveCB
func goResolveCB(sdRef C.DNSServiceRef, flags C.DNSServiceFlags, ifIndex C.uint32_t,
	errCode C.DNSServiceErrorType, fullname *C.char, hosttarget *C.char,
	port C.uint16_t, txtLen C.uint16_t, txt *C.uchar, ctx unsafe.Pointer) {
	if errCode != C.kDNSServiceErr_NoError {
		return
	}
	rc, ok := cgo.Handle(uintptr(ctx)).Value().(*resolveCtx)
	if !ok {
		return
	}
	svc := &Service{
		Instance: rc.instance,
		Type:     rc.regtype,
		Host:     strings.TrimSuffix(C.GoString(hosttarget), "."),
		Port:     int(C.ntohs_go(port)),
		Info:     cleanTXT(parseTXTRecord(txt, txtLen)),
		Source:   "mDNS",
	}
	rc.sess.svcs = append(rc.sess.svcs, svc)

	ac := &addrCtx{svc: svc}
	h := rc.sess.newHandle(ac)
	C.startAddr(rc.sess.shared, ifIndex, hosttarget, h)
}

//export goAddrCB
func goAddrCB(sdRef C.DNSServiceRef, flags C.DNSServiceFlags, ifIndex C.uint32_t,
	errCode C.DNSServiceErrorType, hostname *C.char, address *C.struct_sockaddr,
	ttl C.uint32_t, ctx unsafe.Pointer) {
	if errCode != C.kDNSServiceErr_NoError {
		return
	}
	ac, ok := cgo.Handle(uintptr(ctx)).Value().(*addrCtx)
	if !ok {
		return
	}
	var buf [C.INET_ADDRSTRLEN]C.char
	if C.sockaddrV4(address, &buf[0], C.int(len(buf))) == 0 {
		return
	}
	ip := C.GoString(&buf[0])
	parsed := net.ParseIP(ip)
	if parsed == nil || parsed.IsLoopback() || parsed.IsUnspecified() {
		return // drop 127.x / 0.0.0.0 noise the local host reports for its own services
	}
	ac.svc.Addrs = appendUnique(ac.svc.Addrs, ip)
}

// parseTXTRecord splits a raw DNS-SD TXT record (a sequence of length-prefixed
// strings) into its entries.
func parseTXTRecord(txt *C.uchar, n C.uint16_t) []string {
	if txt == nil || n == 0 {
		return nil
	}
	b := C.GoBytes(unsafe.Pointer(txt), C.int(n))
	var out []string
	for i := 0; i < len(b); {
		l := int(b[i])
		i++
		if l == 0 || i+l > len(b) {
			if i+l > len(b) {
				break
			}
			continue
		}
		out = append(out, string(b[i:i+l]))
		i += l
	}
	return out
}
