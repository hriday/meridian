//go:build windows

package discover

import (
	"context"
	"net"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Windows mDNS backend. It browses through the OS DNS-SD API (DnsServiceBrowse /
// DnsServiceResolve in dnsapi.dll, Windows 10+), which drives the system
// resolver — the same role mDNSResponder plays on macOS and Avahi on Linux. If
// the API is missing (older Windows) or anything goes wrong, it falls back to
// the raw multicast probe. The whole thing is wrapped in a recover so a
// marshalling mistake degrades to the fallback instead of taking down the app.

const (
	dnsRequestPending  = 9506 // DNS_REQUEST_PENDING
	dnsTypePTR         = 12   // DNS_TYPE_PTR
	dnsFreeRecordList  = 1    // DNS_FREE_TYPE: DnsFreeRecordList
	dnsQueryReqVersion = 1    // DNS_QUERY_REQUEST_VERSION1
)

var (
	dnsapi                     = windows.NewLazySystemDLL("dnsapi.dll")
	procDnsServiceBrowse       = dnsapi.NewProc("DnsServiceBrowse")
	procDnsServiceBrowseCancel = dnsapi.NewProc("DnsServiceBrowseCancel")
	procDnsServiceResolve      = dnsapi.NewProc("DnsServiceResolve")
	procDnsServiceFreeInstance = dnsapi.NewProc("DnsServiceFreeInstance")
	procDnsRecordListFree      = dnsapi.NewProc("DnsRecordListFree")
)

// DNS_SERVICE_BROWSE_REQUEST (64-bit). The callback field is the first arm of a
// union; we use the V1 DNS_SERVICE_BROWSE_CALLBACK.
type dnsServiceBrowseRequest struct {
	Version        uint32
	InterfaceIndex uint32
	QueryName      *uint16
	Callback       uintptr
	QueryContext   uintptr
}

type dnsServiceCancel struct {
	reserved uintptr
}

type dnsServiceResolveRequest struct {
	Version        uint32
	InterfaceIndex uint32
	QueryName      *uint16
	Callback       uintptr
	QueryContext   uintptr
}

// DNS_RECORD header (64-bit) up to the data union; for a PTR record the union's
// first pointer is pNameHost (PWSTR).
type dnsRecord struct {
	Next       *dnsRecord
	Name       *uint16
	Type       uint16
	DataLength uint16
	Flags      uint32
	Ttl        uint32
	Reserved   uint32
	DataPtr    *uint16 // union; for a PTR record this is pNameHost (PWSTR)
}

// DNS_SERVICE_INSTANCE (64-bit).
type dnsServiceInstance struct {
	InstanceName  *uint16
	HostName      *uint16
	Ip4Address    *uint32
	Ip6Address    *[16]byte
	Port          uint16
	Priority      uint16
	Weight        uint16
	PropertyCount uint32
	Keys          **uint16
	Values        **uint16
}

// winSession is the shared state a browse/resolve pair writes into. All access
// is guarded because the OS invokes the callbacks on threadpool threads.
type winSession struct {
	mu        sync.Mutex
	instances map[string]bool // instance FQDNs seen while browsing
	services  []Service
	rwg       sync.WaitGroup
}

var (
	winRegMu sync.Mutex
	winRegN  uintptr
	winReg   = map[uintptr]*winSession{}
)

func winRegister(s *winSession) uintptr {
	winRegMu.Lock()
	defer winRegMu.Unlock()
	winRegN++
	winReg[winRegN] = s
	return winRegN
}

func winLookup(id uintptr) *winSession {
	winRegMu.Lock()
	defer winRegMu.Unlock()
	return winReg[id]
}

func winUnregister(id uintptr) {
	winRegMu.Lock()
	delete(winReg, id)
	winRegMu.Unlock()
}

// Stable callback pointers (NewCallback must not be called per-op in a loop; the
// number of callbacks a process can create is limited).
var (
	browseCallbackPtr  = syscall.NewCallback(browseCallback)
	resolveCallbackPtr = syscall.NewCallback(resolveCallback)
)

func systemMDNS(ctx context.Context, window time.Duration) (svcs []Service, err error) {
	if procDnsServiceBrowse.Find() != nil || procDnsServiceResolve.Find() != nil {
		return queryMDNS(ctx, window) // API not present (pre-Windows 10)
	}
	// A marshalling bug must not crash the whole app: degrade to the fallback.
	defer func() {
		if r := recover(); r != nil {
			svcs, err = queryMDNS(ctx, window)
		}
	}()

	sess := &winSession{instances: map[string]bool{}}
	id := winRegister(sess)
	defer winUnregister(id)

	// Start a browse per known type, keeping each cancel handle so we can stop.
	// The browse is asynchronous and may reference QueryName for its lifetime, so
	// the name buffers are pinned until after the browses are cancelled.
	var cancels []*dnsServiceCancel
	var browsePins []*uint16
	for _, t := range mdnsTypes {
		qn := strings.TrimSuffix(t, ".") // "_airplay._tcp.local"
		namePtr, perr := windows.UTF16PtrFromString(qn)
		if perr != nil {
			continue
		}
		browsePins = append(browsePins, namePtr)
		req := dnsServiceBrowseRequest{
			Version:        dnsQueryReqVersion,
			InterfaceIndex: 0,
			QueryName:      namePtr,
			Callback:       browseCallbackPtr,
			QueryContext:   id,
		}
		cancel := &dnsServiceCancel{}
		r, _, _ := procDnsServiceBrowse.Call(
			uintptr(unsafe.Pointer(&req)), uintptr(unsafe.Pointer(cancel)))
		if r == 0 || r == dnsRequestPending {
			cancels = append(cancels, cancel)
		}
	}
	if len(cancels) == 0 {
		return queryMDNS(ctx, window)
	}

	// Let instances accumulate for the window, then stop browsing.
	select {
	case <-ctx.Done():
	case <-time.After(window):
	}
	for _, c := range cancels {
		procDnsServiceBrowseCancel.Call(uintptr(unsafe.Pointer(c)))
	}
	runtime.KeepAlive(browsePins)

	// Resolve everything we found.
	sess.mu.Lock()
	names := make([]string, 0, len(sess.instances))
	for n := range sess.instances {
		names = append(names, n)
	}
	sess.mu.Unlock()

	var resolvePins []*uint16
	for _, n := range names {
		namePtr, perr := windows.UTF16PtrFromString(n)
		if perr != nil {
			continue
		}
		resolvePins = append(resolvePins, namePtr)
		req := dnsServiceResolveRequest{
			Version:        dnsQueryReqVersion,
			InterfaceIndex: 0,
			QueryName:      namePtr,
			Callback:       resolveCallbackPtr,
			QueryContext:   id,
		}
		cancel := &dnsServiceCancel{}
		sess.rwg.Add(1)
		r, _, _ := procDnsServiceResolve.Call(
			uintptr(unsafe.Pointer(&req)), uintptr(unsafe.Pointer(cancel)))
		if r != 0 && r != dnsRequestPending {
			sess.rwg.Done() // never dispatched
		}
	}

	// Wait for resolves, bounded by whatever time the context still allows.
	done := make(chan struct{})
	go func() { sess.rwg.Wait(); close(done) }()
	wait := 4 * time.Second
	if dl, ok := ctx.Deadline(); ok {
		if rem := time.Until(dl); rem < wait {
			wait = rem
		}
	}
	select {
	case <-done:
	case <-time.After(wait):
	}
	runtime.KeepAlive(resolvePins)

	sess.mu.Lock()
	out := append([]Service(nil), sess.services...)
	sess.mu.Unlock()
	return dedupeServices(out), nil
}

func browseCallback(status uintptr, ctxID uintptr, recPtr uintptr) uintptr {
	if status != 0 || recPtr == 0 {
		return 0
	}
	sess := winLookup(ctxID)
	if sess == nil {
		return 0
	}
	for rec := (*dnsRecord)(unsafe.Pointer(recPtr)); rec != nil; rec = rec.Next {
		if rec.Type == dnsTypePTR && rec.DataPtr != nil {
			if name := windows.UTF16PtrToString(rec.DataPtr); name != "" {
				sess.mu.Lock()
				sess.instances[name] = true
				sess.mu.Unlock()
			}
		}
	}
	// The record list is ours to free.
	procDnsRecordListFree.Call(recPtr, uintptr(dnsFreeRecordList))
	return 0
}

func resolveCallback(status uintptr, ctxID uintptr, instPtr uintptr) uintptr {
	sess := winLookup(ctxID)
	if sess == nil {
		if instPtr != 0 {
			procDnsServiceFreeInstance.Call(instPtr)
		}
		return 0
	}
	defer sess.rwg.Done()
	if status != 0 || instPtr == 0 {
		return 0
	}
	defer procDnsServiceFreeInstance.Call(instPtr)

	inst := (*dnsServiceInstance)(unsafe.Pointer(instPtr))
	full := utf16ToString(inst.InstanceName)
	instance, typ := splitInstanceName(full)
	svc := Service{
		Instance: instance,
		Type:     typ,
		Host:     strings.TrimSuffix(utf16ToString(inst.HostName), "."),
		Port:     int(inst.Port),
		Source:   "mDNS",
	}
	if inst.Ip4Address != nil {
		v := *inst.Ip4Address // IP4_ADDRESS, network byte order
		ip := net.IPv4(byte(v), byte(v>>8), byte(v>>16), byte(v>>24))
		if !ip.IsLoopback() && !ip.IsUnspecified() {
			svc.Addrs = appendUnique(svc.Addrs, ip.String())
		}
	}
	svc.Info = cleanTXT(readTXTProperties(inst))

	sess.mu.Lock()
	sess.services = append(sess.services, svc)
	sess.mu.Unlock()
	return 0
}

func readTXTProperties(inst *dnsServiceInstance) []string {
	n := int(inst.PropertyCount)
	if n == 0 || inst.Keys == nil || inst.Values == nil {
		return nil
	}
	keys := unsafe.Slice(inst.Keys, n)
	vals := unsafe.Slice(inst.Values, n)
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		k := utf16ToString(keys[i])
		if k == "" {
			continue
		}
		if v := utf16ToString(vals[i]); v != "" {
			out = append(out, k+"="+v)
		} else {
			out = append(out, k)
		}
	}
	return out
}

func utf16ToString(p *uint16) string {
	if p == nil {
		return ""
	}
	return windows.UTF16PtrToString(p)
}

// splitInstanceName turns "Living Room._airplay._tcp.local" into
// ("Living Room", "_airplay._tcp"), tolerating dots inside the instance label.
func splitInstanceName(full string) (instance, typ string) {
	full = strings.TrimSuffix(full, ".")
	for _, marker := range []string{"._tcp.", "._udp."} {
		if i := strings.Index(full, marker); i >= 0 {
			head := full[:i] // "<instance>._airplay"
			if j := strings.LastIndex(head, "._"); j >= 0 {
				instance = head[:j]
				svc := head[j+1:] // "_airplay"
				typ = svc + strings.TrimSuffix(marker, ".")
				return
			}
			instance = head
			return
		}
	}
	return full, ""
}
