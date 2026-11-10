# Local-network tranche (Ping sweep + Port scanner) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans (inline). Steps use checkbox (`- [ ]`) syntax.

**Goal:** Two Local-network tools — a ping sweep (live hosts on a CIDR, ICMP + TCP fallback) and a TCP-connect port scanner (open/closed/filtered) — engine, CLI, and UI.

**Architecture:** `internal/scan` — a TCP-connect `ScanPorts`, a `Sweep` over a CIDR that takes an injected `Pinger` (ICMP) plus a TCP fallback, and an `ICMPPinger` (x/net/icmp, unprivileged) behind that seam. Bounded worker pools, context-cancellable, streamed results. Thin CLI + App bindings (streaming `scan:*` events); two Svelte tools.

**Tech Stack:** Go 1.23 stdlib (`net`, `net/netip`, `context`, `sync`) + `golang.org/x/net/icmp`/`ipv4`/`ipv6` (existing dep); Wails + Svelte.

**Spec:** `docs/superpowers/specs/2026-11-09-meridian-localnet-slice1-design.md`

## Global Constraints
- Go **1.23**, no new deps, additive only. ICMP stays **unprivileged** and behind the `Pinger` seam; tests use a fake pinger + local listeners (no raw sockets / real network).
- Bounded concurrency; per-probe timeout; cancellable; a host-count cap on sweeps. Degrade to TCP when ICMP is unavailable; never crash.
- No stealth/evasion; the UI carries an authorized-use note.

## Review Focus
- **Closed vs filtered vs open** — a refused connection is `closed`, a timeout/cancel is `filtered`, a successful connect is `open`; classify from the dial error, don't mislabel. (Task 1 test, incl. a cancelled-context → filtered.)
- **Sweep concurrency is bounded** — a /24 does not spawn 254 simultaneous dials; a worker pool caps it, and the host cap refuses an oversized prefix. (Task 2 tests.)
- **Context cancellation** stops both scanners promptly with partial results, no goroutine leak. (Task 1/2 tests; `-race`.)
- **Host enumeration** skips the network/broadcast addresses appropriately and handles a single-host /32. (Task 2 test.)
- **ICMP unavailable** (socket open fails) → `Sweep` still runs via the TCP fallback and marks method `tcp`. (Task 2 test with a pinger that reports "unavailable".)

---

### Task 1: Port scan engine
**Files:** Create `internal/scan/portscan.go`, `portscan_test.go`
**Produces:**
```go
type PortResult struct { Port int; State, Service string } // open | closed | filtered
func ParsePorts(spec string) ([]int, error)                // "22,80,443,8000-8100"
func CommonPorts() []int
func ScanPorts(ctx context.Context, host string, ports []int, timeout time.Duration, concurrency int, onResult func(PortResult)) ([]PortResult, error)
```
- [ ] **Step 1: failing tests** — `TestParsePorts` ("22,80,8000-8002" → [22,80,8000,8001,8002]; rejects "bad", "9999999"); `TestScanPortStates`: a `net.Listen` tcp gives an **open** port; an unused 127.0.0.1 port gives **closed** (refused); a **cancelled context** gives **filtered**; `TestScanPortsService` (port 22→"ssh", 443→"https"). Assert results are sorted by port and `onResult` fires per port.
- [ ] **Step 2–4:** implement — `scanPort` dials `net.DialContext` with the timeout; connected→open(+Close); error `errors.Is(ECONNREFUSED)` or "refused"→closed; else→filtered. `ScanPorts` fans out over a buffered worker pool (`concurrency`), collects, sorts. Service map for common ports. Verify (`go test -race`).
- [ ] **Step 5: commit** `feat(scan): TCP-connect port scanner`.

### Task 2: Ping sweep engine
**Files:** Create `internal/scan/sweep.go`, `sweep_test.go`
**Produces:**
```go
type HostResult struct { IP string; Alive bool; RTT time.Duration; Method, Name string } // Method: icmp|tcp
type Pinger interface { Ping(ctx context.Context, ip netip.Addr, timeout time.Duration) (rtt time.Duration, ok, available bool) }
const MaxSweepHosts = 65536
func Sweep(ctx context.Context, cidr string, p Pinger, tcpPorts []int, timeout time.Duration, concurrency int, onResult func(HostResult)) ([]HostResult, error)
```
- [ ] **Step 1: failing tests** — `TestSweepICMPAndTCP`: a `/30` CIDR; a fake Pinger that reports one host alive via ICMP and another `available:false`; a `net.Listen` so the TCP fallback finds a second host alive on a tcpPort → both hosts Alive with Method icmp/tcp respectively; dead hosts Alive=false. `TestSweepRejectsHugeRange` (a `/8` → error, no scan). `TestSweepBoundedConcurrency` (concurrency honoured — e.g. a counting fake pinger whose max-in-flight never exceeds `concurrency`). `TestSweepHostEnumeration` (a `/30` yields the 2 usable hosts, not network/broadcast; a `/32` yields the single host).
- [ ] **Step 2–4:** implement — parse with `netip.ParsePrefix`; enumerate usable hosts (skip network/broadcast for IPv4 prefixes shorter than /31; /31 and /32 as-is); cap at `MaxSweepHosts`; worker pool: ICMP `Ping` first (when `available`), else/también TCP-connect to `tcpPorts`; reverse-DNS best-effort (`net.LookupAddr` with a short timeout). Verify (`-race`).
- [ ] **Step 5: commit** `feat(scan): CIDR ping sweep (ICMP + TCP fallback)`.

### Task 3: ICMP pinger (production Pinger)
**Files:** Create `internal/scan/icmp.go`, `icmp_test.go`
**Produces:** `func NewICMPPinger() *ICMPPinger` implementing `Pinger` via `icmp.ListenPacket` unprivileged echo (IPv4 `udp4`/IPv6 `udp6`), matching the traceroute approach; `Ping` reports `available:false` when the socket can't be opened.
- [ ] **Step 1: failing test** — `TestICMPPingerAvailability`: construct the pinger; `Ping` to `127.0.0.1` returns without panicking and, if the socket opened, `available:true` (loopback ping may or may not reply in CI — assert no panic + `available` reflects socket success, don't require `ok`). Mark live-verify in the ledger (real ICMP needs network/policy).
- [ ] **Step 2–4:** implement echo send + reply match (id/seq), degrade to `available:false` on `ListenPacket` error. Verify.
- [ ] **Step 5: commit** `feat(scan): unprivileged ICMP echo pinger`.

### Task 4: CLI commands
**Files:** Modify `internal/cli/cli.go` (+ `cli_scan_test.go`)
**Produces:** `meridian-cli pingsweep <cidr> [--ports 22,80,443] [--timeout 1s] [--concurrency 64] [--json]` and `meridian-cli portscan <host> <ports|--common> [--timeout] [--concurrency] [--json]`.
- [ ] **Step 1: failing tests** — `TestRunPortScan` (local listener → its port reported open in output); `TestRunPingSweep` (a `/31` with a fake pinger seam [`var newPinger = scan.NewICMPPinger`, overridden in test] + a local listener for the TCP fallback → a host reported up); arg-validation (`exit 2`).
- [ ] **Step 2–4:** implement `runPortScan`/`runPingSweep` with a `newPinger` seam; print tables (or JSON). Verify + `go test ./...`.
- [ ] **Step 5: commit** `feat(cli): pingsweep and portscan commands`.

### Task 5: App bindings + event streaming
**Files:** Modify `internal/app/app.go`; create `internal/app/scan_methods.go` (+ test)
**Produces:** `PortScan(host string, ports []int) ([]scan.PortResult, error)` + `PingSweep(cidr string, tcpPorts []int) ([]scan.HostResult, error)` streaming `scan:port` / `scan:host` events; `ScanCommonPorts() []int`; a cancellable active scan (`ScanStop()`), mutex-guarded like traceroute. A `newPinger`-style seam so the test injects a fake.
- [ ] **Step 1: failing test** — `TestAppPortScan`: local listener; `a.PortScan(host, [port])` returns it open. `TestAppPingSweep`: inject a fake pinger; a `/31` → a host up. (ctx nil → EventsEmit no-op.)
- [ ] **Step 2–4:** implement; verify + `go test ./...`.
- [ ] **Step 5: commit** `feat(app): ping sweep and port scan bindings`.

### Task 6: UI tools + Workbench wiring
**Files:** Create `frontend/src/lib/tools/PingSweep.svelte`, `PortScan.svelte`; modify `Workbench.svelte`; regenerated `wailsjs`.
> UI verified by `wails build` (config/generated TDD exception; ledgered).
- [ ] **Step 1:** regenerate bindings; grep for `PortScan`/`PingSweep`.
- [ ] **Step 2:** build both tools — an authorized-use note; **Ping sweep**: CIDR input + options (ports, concurrency), Run + live progress, a results table (host / RTT / method / name) showing live hosts; **Port scan**: host + ports (or "common") input, Run + progress, a results table (port / state[colored] / service). Subscribe `scan:host`/`scan:port`; unsubscribe on destroy. Flip the `ping`/`ports` Workbench entries `ready:true`, import + route both. Plain-language captions; theme tokens; no horizontal overflow.
- [ ] **Step 3:** `wails build` succeeds.
- [ ] **Step 4: commit** `feat(ui): ping sweep and port scanner tools`.

## Self-review notes
- **Spec coverage:** port scan (T1), ping sweep incl. ICMP+TCP+cap (T2), ICMP pinger (T3), CLI (T4), app streaming (T5), both UIs (T6). SYN/stealth/version-detection are explicit non-goals.
- **Type consistency:** `PortResult`/`HostResult`/`Pinger` (T1–T3) consumed by T4/T5/T6 unchanged.
- **Review Focus:** all five rows map to named T1/T2 tests.
- **Safety:** bounded pools + host cap + cancellation; no stealth; authorized-use note; ICMP unprivileged behind a seam so tests stay offline.
- **Proportion:** engine (T1–T3) is the substance; CLI/app thin; UI presentation.
