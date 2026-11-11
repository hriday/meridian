# LAN Speed Test Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** A LAN iperf3 speed-test tool: point Meridian at a host running `iperf3 -s`, run a live download/upload/bidir throughput test with a gauge + chart, and show a result summary — using the pure-Go `github.com/appshubcc/go-iperf3` client (self-contained, client-only).

**Architecture:** A thin engine wrapper `internal/speedtest` isolates the library behind `Run(ctx, Options, onSample) (Result, error)`. The app layer streams samples over the existing `monitor` infra (`metrics:<streamID>` events), exactly like the ping/SNMP monitors. A Svelte tool reuses `Gauge`/`StreamChart`/`metricStream`.

**Tech Stack:** Go 1.23, `github.com/appshubcc/go-iperf3` v0.1.0, Wails events, Svelte.

**Spec:** `docs/superpowers/specs/2026-11-10-lan-speedtest-design.md`

## Global Constraints

- **Client only** — the app never calls the library's `Server`/`Listen`; that appears only in tests (loopback 127.0.0.1).
- `Run` forwards `onSample` only for `connecting`/`running` phases; the terminal outcome is the returned `Result`/error (no double terminal event).
- Durations are whole seconds (library rounds up); UI caps duration (≤60s) and parallel (≤128).
- The library panic is recover-guarded in the app goroutine so it can't crash Meridian.
- On Stop (ctx cancel → `cancelled`), emit no terminal event (mirror PingMonitor's `if ctx.Err() != nil { return }`).

## Review Focus

- **No server listening** → a clear "no server answered" snapshot within ~6s, no hang. (Task 1/2.)
- **Stop mid-test** → prompt cancel, goroutine exits, no late event. (Task 2.)
- **Library panic / nil Result** → recover guard; build `Result` only after `State.Error==""`. (Task 1/2.)
- **Unresolvable host** → pre-resolve → "Couldn't resolve <host>". (Task 1.)
- **Direction semantics** → download=server→client, upload=client→server, bidir=both; live Mbps + labels match. (Task 1.)

---

### Task 1: `internal/speedtest` engine wrapper

**Files:** Create `internal/speedtest/speedtest.go`, `internal/speedtest/speedtest_test.go`.

**Interfaces:**
- Produces:
  - `Options{ Host string; Port int; Direction string; Parallel int; Duration time.Duration; UDP bool }`
  - `Sample{ Elapsed, DownMbps, UpMbps float64; Phase string }`
  - `Result{ DownMbps, UpMbps, Bytes float64; Retransmits int64; RTTMeanMs, JitterMs float64; Lost, Packets int64; Peer, Protocol string }`
  - `Run(ctx context.Context, o Options, onSample func(Sample)) (Result, error)` — pre-resolves `o.Host` (`net.DefaultResolver.LookupHost`, short ctx) → "couldn't resolve" error on failure; maps Direction ("download"→`iperf3.Download`, "upload"→`Upload`, "bidir"→`Bidir`), UDP→Protocol; `OnUpdate` forwards to `onSample` only for connecting/running (Mbps = bits/1e6); after `RunClient` returns, map `State.Error` (refused/timeout/unreachable→"no server answered…", busy→"server busy", cancelled/interrupted→ctx error) or build `Result` from `DownloadResult`/`UploadResult`.
- Consumes: `github.com/appshubcc/go-iperf3`.

- [ ] **Step 1: Write failing tests** using a loopback `iperf3.Listen("127.0.0.1:0")`:
  - `TestRunDownloadLoopback`: Direction download, 1s, parallel 2 → `Result.DownMbps > 0`, running `onSample`s received (DownMbps>0 on at least one).
  - `TestRunUploadLoopback`: upload → `Result.UpMbps > 0`.
  - `TestRunBidirLoopback`: bidir → both `DownMbps` and `UpMbps` > 0.
  - `TestRunNoTerminalSample`: no `onSample` has `Phase=="done"` (terminal suppressed).
  - `TestRunNoServer`: host 127.0.0.1 port 1 → error mentioning "server" (refused), returns within a couple seconds.
  - `TestRunBadHost`: Host `"no.such.host.invalid"` → error mentioning "resolve".
  - `TestRunCancel`: cancel the ctx ~200ms in → returns promptly with a cancellation error; running samples may have arrived.
- [ ] **Step 2: Run, verify fail.**
- [ ] **Step 3: Implement** `speedtest.go`.
- [ ] **Step 4: Run, verify pass** (`go test ./internal/speedtest/`).
- [ ] **Step 5: Commit** (`feat(speedtest): pure-Go iperf3 client engine wrapper`).

---

### Task 2: App streaming methods

**Files:** Create `internal/app/speedtest_methods.go`, `internal/app/speedtest_methods_test.go`.

**Interfaces:**
- Produces:
  - `SpeedSnapshot{ Elapsed, DownMbps, UpMbps float64; State string; Note, Peer, Protocol string; AvgDownMbps, AvgUpMbps, Bytes float64; Retransmits int64; RTTMeanMs, JitterMs float64; Lost, Packets int64 }`.
  - `SpeedtestStart(streamID, host string, port int, direction string, parallel, durationSec int, udp bool) error` — validates host; `registerMonitor`; recover-guarded goroutine runs `speedtest.Run`, emitting `metrics:<streamID>` with a running `SpeedSnapshot{State:"running"}` per sample; on return, if `ctx.Err()==nil` emit one terminal snapshot (`State:"done"` + Result, or `State:"error"`/`"unreachable"` + `Note`); `defer close(done)`.
  - `SpeedtestStop(streamID string)` → `stopMonitor`.
- Consumes: `internal/speedtest`, the `monitor` infra (`registerMonitor`/`stopMonitor`/`emitEvent`).

- [ ] **Step 1: Write failing tests** (use the `emitEvent` seam to capture snapshots; loopback `iperf3.Listen` as the server):
  - `TestSpeedtestStreamsAndCompletes`: Start against a loopback server → captures ≥1 `running` snapshot then exactly one `done` snapshot with `AvgDownMbps>0`.
  - `TestSpeedtestNoServerErrors`: Start against port 1 → a terminal snapshot with `State:"error"`/`"unreachable"` and a `Note`, no running crash.
  - `TestSpeedtestStopNoTerminal`: Start a longer test, `SpeedtestStop` quickly → the monitor goroutine exits (stopMonitor returns) and NO `done`/`error` snapshot is emitted after stop.
  - `TestSpeedtestDuplicateStart`: same streamID twice → error.
  - `TestSpeedtestBadHost`: empty host → error.
- [ ] **Step 2: Run, verify fail.**
- [ ] **Step 3: Implement** `speedtest_methods.go`.
- [ ] **Step 4: Run, verify pass** + `go test ./...`.
- [ ] **Step 5: Commit** (`feat(app): LAN speed test streaming methods`).

---

### Task 3: UI tool

**Files:** Create `frontend/src/lib/tools/SpeedTest.svelte`; modify `frontend/src/lib/Workbench.svelte`; regenerate bindings.

**Interfaces:** Consumes `SpeedtestStart`/`SpeedtestStop`, `metricStream.subscribe`/`newStreamID`, `Gauge`, `StreamChart`.

Per the design: host + port(5201) + direction select(download/upload/bidir) + parallel(4) + duration(10s, cap 60) + TCP/UDP toggle + Start/Stop. Subscribe to `metrics:<streamID>` BEFORE Start (unsub before Stop, like PingMonitor). Live `Gauge` (current Mbps, pct against the running peak, auto-scaling) + `StreamChart` (down/up series over elapsed) + a result card (avg down/up, retransmits, RTT; jitter/loss when UDP). A help line: the other end needs `iperf3 -s`.

- [ ] **Step 1: Implement** the Svelte tool; register under **Local network** (`{ id:"speedtest", name:"LAN speed test", group:"Local network", ready:true }`).
- [ ] **Step 2: Build** `npm run build` → no errors.
- [ ] **Step 3: Regenerate bindings** (`$(go env GOPATH)/bin/wails generate module`; `git checkout -- frontend/wailsjs/runtime`).
- [ ] **Step 4: Manual check** — can't fully run without an iperf3 server; confirm the tool renders, validates an empty host, and (if an `iperf3 -s`/loopback is reachable) streams. Document the limitation.
- [ ] **Step 5: Commit** (`feat(app,ui): LAN speed test tool`).

---

## Self-review notes

- Spec coverage: engine (T1), streaming methods (T2), UI (T3).
- Review Focus: no-server (T1/T2), stop (T2), panic/nil (T1/T2), bad host (T1), direction (T1).
- Types consistent: `Run`→`Sample`/`Result` consumed by the app's `SpeedSnapshot`; monitor infra reused.
- Proportion: plan gives the interfaces + forwarding rules + test names; the wrapper/goroutine bodies are the implementer's.
