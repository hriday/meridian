# Netcat (security slice 2) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans (inline) to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** A raw-connection tool — connect (TCP/UDP, optional TLS) or listen (TCP/UDP) on a local port, exchanging bytes both ways, shown as text or hex.

**Architecture:** A `net.Conn`-based `Session` in `internal/netcat` (Send + a read-loop callback + Close), `Dial` (TCP/UDP/`tls.Dial`, returning a cert summary for TLS) and `Listen` (TCP `Accept`; UDP via a packet-conn adapter) both yielding a `Session`. Thin CLI (`nc`, stdin↔conn) and App bindings that stream `netcat:recv`/`netcat:status` events (mirroring the traceroute streaming pattern); a `Netcat.svelte` tool.

**Tech Stack:** Go 1.23 stdlib (`net`, `crypto/tls`, `crypto/x509`, `bufio`, `encoding/hex`); Wails + Svelte.

**Spec:** `docs/superpowers/specs/2026-11-09-meridian-netcat-slice1-design.md`

## Global Constraints
- Go **1.23**, stdlib only, no new deps. Additive only.
- **Never execute a command** on connect/data. TLS defaults to verifying; an unverified connection is allowed only via an explicit `Insecure` flag and is labelled.
- Read loops are cancellable; `Close` stops them with no goroutine leak.

## Review Focus
- **Close during an active read** — closing mid-stream ends the read loop and fires `onClose` once, no panic on a closed conn, no leaked goroutine. (Task 1 test.)
- **TLS to an invalid/self-signed cert** — default (`Insecure:false`) returns a verification error; `Insecure:true` connects and reports `CertInfo.Verified=false` with the cert summary. (Task 1 test.)
- **UDP listen peer binding** — a datagram from a peer makes `Accept` return a session whose `Send` goes back to *that* peer. (Task 2 test.)
- **Partial/zero-length reads & binary data** — the read loop forwards exactly the bytes received (no text assumptions); hex dump handles non-printable/zero bytes. (Task 1/hexdump tests.)
- **Listen Close before any peer** — `Accept` unblocks with an error, no hang. (Task 2 test.)

---

### Task 1: Dial + Session + hexdump
**Files:** Create `internal/netcat/netcat.go`, `netcat_test.go`
**Produces:**
```go
type Config struct { Network, Addr string; TLS, Insecure bool } // Network: "tcp"|"udp"
type CertInfo struct { Subject, Issuer, NotAfter string; Verified bool }
type Session struct { /* wraps net.Conn */ }
func Dial(ctx context.Context, cfg Config) (*Session, *CertInfo, error) // CertInfo non-nil only for TLS
func (s *Session) Send(b []byte) (int, error)
func (s *Session) Start(onData func([]byte), onClose func(error))       // starts the read loop
func (s *Session) Close() error
func (s *Session) Peer() string
func HexDump(b []byte) string                                          // offset  hex  ascii
```
- [ ] **Step 1: failing tests** — `TestDialSendRecv` (start a local TCP echo listener; Dial, Send "ping", assert `onData` delivers "ping"); `TestSessionCloseStopsLoop` (Close mid-read → `onClose` fires once, no panic); `TestDialTLSVerifies` (a `httptest.NewTLSServer`; `Insecure:false` → error; `Insecure:true` → connects, `CertInfo.Verified==false`, Subject populated); `TestHexDump` (bytes incl. 0x00 and non-printable → offset/hex/ascii with dots for non-printable).
- [ ] **Step 2–4:** implement — `Dial`: `net.Dial`/`tls.Dial` (build `*tls.Config{InsecureSkipVerify: cfg.Insecure}`; after handshake read `ConnectionState().PeerCertificates[0]` for `CertInfo`, set `Verified` from whether a verify-enabled dial would succeed / `cfg.Insecure`); `Session.Start` runs a `bufio`/`conn.Read` loop in a goroutine, calling `onData` per chunk and `onClose` once on EOF/error/Close; `Close` closes the conn (idempotent). Verify.
- [ ] **Step 5: commit** `feat(netcat): dial (TCP/UDP/TLS) session with streaming read and hexdump`.

### Task 2: Listen
**Files:** Create `internal/netcat/listen.go`, `listen_test.go`
**Produces:**
```go
type Listener struct { /* TCP net.Listener or UDP packet adapter */ }
func Listen(ctx context.Context, cfg Config) (*Listener, error)
func (l *Listener) Accept() (*Session, error) // first peer; Session.Send replies to it
func (l *Listener) Close() error
```
- [ ] **Step 1: failing tests** — `TestListenTCPAccept` (Listen tcp on 127.0.0.1:0; Dial to its Addr; `Accept` returns a session; data flows both ways); `TestListenUDPAccept` (Listen udp; a datagram from a peer → `Accept` returns a session whose `Send` reaches that peer); `TestListenCloseUnblocksAccept` (Close before any peer → `Accept` errors, no hang). Expose the bound address (e.g. `l.Addr()`), since `:0` picks a port.
- [ ] **Step 2–4:** implement — TCP: `net.Listen` + `Accept` wrapped in a `Session`; UDP: `net.ListenPacket`, first `ReadFrom` captures the peer, wrap the packet-conn in a `net.Conn` adapter bound to that peer (`Read` filters/returns datagrams, `Write` → `WriteTo` peer). Verify.
- [ ] **Step 5: commit** `feat(netcat): TCP/UDP listen yielding a peer session`.

### Task 3: CLI `nc`
**Files:** Modify `internal/cli/cli.go` (+ `cli_nc_test.go`)
**Produces:** `meridian-cli nc <host> <port> [--udp] [--tls] [--insecure] [--listen]` — connect (or listen), then pipe stdin → Send and recv → stdout until EOF/close. (Hex view is a UI concern; the CLI is raw bytes.)
- [ ] **Step 1: failing test** — `TestRunNCConnect`: start a local echo listener in the test, run `nc 127.0.0.1 <port>` with a `strings.Reader` stdin ("hi\n") and a buffer stdout, assert the echoed bytes appear; `TestRunNCNeedsHostPort` (missing args → exit 2). Use the existing `Run(args, stdin?, stdout, stderr)` shape — if `Run` has no stdin param, add an injectable stdin seam for `nc` (package var) like `newCracker`/`newLookupClient`.
- [ ] **Step 2–4:** implement `runNC` using `netcat.Dial`/`Listen`; copy stdin↔session; respect flags. Verify + `go test ./...`.
- [ ] **Step 5: commit** `feat(cli): nc connect/listen command`.

### Task 4: App bindings + event streaming
**Files:** Modify `internal/app/app.go` (fields); create `internal/app/netcat_methods.go` (+ test)
**Produces:** App holds one active session/listener. Methods: `NetcatConnect(network, addr string, useTLS, insecure bool) (netcat.CertInfo, error)`, `NetcatListen(network, addr string) (string, error)` (returns bound addr), `NetcatSend(text string) error`, `NetcatClose()`. Streams `netcat:recv` (`{text, hex, n}`) and `netcat:status` (`{state, peer, err}`) via `wruntime.EventsEmit`, guarded on `a.ctx != nil` (as the trace methods do).
- [ ] **Step 1: failing test** — `TestAppNetcatConnectSendRecv`: start a local echo listener; `a.NetcatConnect("tcp", addr, false, false)`; `a.NetcatSend("ping")`; assert (via a channel the test wires through the session, or by polling a captured buffer) the echoed data returns; `a.NetcatClose()` ends it. (ctx is nil in the test so EventsEmit is a no-op; assert on session state, not events.)
- [ ] **Step 2–4:** implement with a mutex around the active session (like `traceMu`); `NetcatClose` closes + clears; the read-loop callback emits events. Verify + `go test ./...`.
- [ ] **Step 5: commit** `feat(app): netcat session bindings and event streaming`.

### Task 5: Netcat.svelte tool + Workbench wiring
**Files:** Create `frontend/src/lib/tools/Netcat.svelte`; modify `frontend/src/lib/Workbench.svelte`; regenerated `wailsjs` bindings.
> UI verified by `wails build` (no frontend unit harness — config/generated TDD exception; stated in ledger).
- [ ] **Step 1:** regenerate bindings (`wails generate module`); grep for `NetcatConnect`.
- [ ] **Step 2: build the tool** — a target row (host, port, TCP/UDP toggle, TLS checkbox + "allow unverified"), **Connect** and **Listen** buttons, a live **transcript** pane with a **text / hex** toggle and a byte counter, a **send** box with a line-ending selector (none/LF/CRLF) + Send, a status line (connecting/connected/closed/error + peer), and the **TLS cert summary** when present. Subscribe to `netcat:recv`/`netcat:status`; unsubscribe on destroy. Plain-language captions; theme tokens; no horizontal overflow.
- [ ] **Step 3:** wire Workbench — import `Netcat`, add `{ id:"nc", name:"Netcat", group:"Security", ready:true }`, route branch.
- [ ] **Step 4:** `wails build` succeeds.
- [ ] **Step 5: commit** `feat(ui): netcat tool`.

## Self-review notes
- **Spec coverage:** connect+TLS+cert (T1), listen TCP/UDP (T2), CLI (T3), app streaming (T4), UI incl. hex/text + line endings (T5). Exec-on-connect is an explicit non-goal (and never implemented).
- **Type consistency:** `Session`/`Config`/`CertInfo` (T1) consumed by T2/T3/T4; `CertInfo` surfaced to UI (T5).
- **Review Focus:** all five rows map to named Task 1/2 tests.
- **Safety:** no command execution anywhere; TLS verifies by default, unverified is explicit + labelled; read loops cancel on Close.
- **Proportion:** engine (T1–T2) is the substance; CLI/app are thin wiring; UI is presentation.
