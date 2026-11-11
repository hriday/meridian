# Collector framework + Syslog receiver Implementation Plan (slice 1 of 2)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A reusable in-memory UDP collector framework, proven by a live Syslog receiver (listen, live tail, filter), ephemeral and laptop-safe.

**Architecture:** A generic `internal/collector` core — a bounded sequenced `Ring` + a `Collector` that binds a UDP socket, runs a read loop that `parse`s datagrams into typed messages and buffers them, and stops cleanly. A singleton-per-kind lives on `App` with `SyslogStart/Stop/Poll` (poll-since-cursor, epoch on restart). A shared `LiveTail` Svelte component renders the tail + filter; the `SyslogReceiver` tool uses it. Slice 2 adds a trap `parse` + tool.

**Tech Stack:** Go 1.23 generics + Wails v2, Svelte/TS. Trap decoding (slice 2) reuses gosnmp.

**Spec:** docs/superpowers/specs/2026-11-10-meridian-collectors-design.md

## Global Constraints

- **Ephemeral, in-memory**: rolling ring of 10000 messages (oldest dropped); nothing to disk; buffer discarded on Stop; collector auto-stops on app shutdown (no orphaned socket).
- **High default port** (syslog 5514), editable, 1–65535; bind `0.0.0.0` (a device must reach it). Start binds synchronously; EADDRINUSE/EACCES → friendly text.
- **Poll contract**: `Poll(afterSeq) → {running, port, epoch, total, dropped, messages}`; per-poll cap = ring capacity (a mount-time restore from 0 is never truncated); Poll never blocks; after Stop → running=false, no messages. Start while running = restart (stop-wait, new port, bump epoch).
- **Defensive parsing** of unauthenticated UDP: bounded read (64 KB), truncate message to ~2 KB, hex for non-text, never panic on malformed input.
- **Concurrency**: the read loop `Add`s under the ring lock; Poll copies the slice under the lock and marshals outside it. Stop/OnShutdown wait on the goroutine WITHOUT holding the app mutex.

## Review Focus

- **No orphaned/leaked goroutine or socket**: Stop closes the conn and blocks until the read loop exits; OnShutdown stops the collector; a second Start restarts cleanly. (Task 1 + Task 3 tests.)
- **Bind failure** (port in use) → a clear error, collector not marked running. (Task 3 test.)
- **Ring bound + dropped accounting**: at capacity the oldest are evicted and `dropped` counts them; `Since(seq)` returns only newer items. (Task 1 tests.)
- **Restart detection**: epoch bumps on restart so a stale cursor is detected. (Task 1/3 test.)
- **Malformed datagram** doesn't panic and is truncated/hex as needed. (Task 2 tests.)
- **Concurrency**: `-race` clean with the read loop writing while Poll reads. (Task 1/3 tests.)

---

### Task 1: Collector core — Ring + UDP Collector

**Files:**
- Create: `internal/collector/ring.go`, `internal/collector/collector.go`
- Test: `internal/collector/ring_test.go`, `internal/collector/collector_test.go`

**Interfaces:**
- Produces:
  - `type Stamped[T any] struct { Seq int; Val T }`
  - `type Ring[T any]` with `NewRing[T](capacity int) *Ring[T]`, `Add(v T)`, `Snapshot(afterSeq int) (items []Stamped[T], total, dropped int)` (all lock-guarded; `Snapshot` returns items with Seq>afterSeq up to capacity).
  - `type Poll[T any] struct { Running bool; Port, Epoch, Total, Dropped int; Items []Stamped[T] }`
  - `type Collector[T any]` with `New[T](capacity int, parse func(datagram []byte, src netip.Addr) (T, bool)) *Collector[T]`, `Start(port int) error`, `Stop()`, `PollSince(afterSeq int) Poll[T]`.

- [ ] **Step 1: Write failing tests.**
```go
// ring_test.go
func TestRingSeqAndSince(t)        // Add 3; Snapshot(0) → 3 items seq 1,2,3; Snapshot(2) → 1 item seq 3
func TestRingEvictsAndCountsDropped(t) // cap 3, Add 5 → total 3, dropped 2, oldest retained seq is 3
// collector_test.go (bind an ephemeral UDP port on 127.0.0.1)
func TestCollectorReceivesAndPolls(t)  // parse = identity-ish; Start(0→actual port via Poll); send 2 UDP datagrams; Eventually PollSince(0).Items has 2; running true
func TestCollectorStopNoLeak(t)        // after Stop, running=false, PollSince(0) empty; sending more does nothing; (goroutine exited — Stop returned)
func TestCollectorRestartBumpsEpoch(t) // Start; note epoch; Start again → epoch increased, buffer reset
func TestCollectorBindInUse(t)         // Start on a port already held → error; running=false
```
- [ ] **Step 2: Run, verify fail** — `go test ./internal/collector/` → FAIL (undefined).
- [ ] **Step 3: Implement.** `Ring`: slice + `nextSeq` + `dropped`, mutex; `Add` appends `Stamped{++seq, v}`, trims front when over capacity (incrementing `dropped`); `Snapshot` copies matching items under the lock. `Collector`: mutex + `running/port/epoch` + `*Ring` + `conn *net.UDPConn` + `done chan struct{}`. `Start(port)`: if running, `Stop()`; `net.ListenUDP("udp", :port)` (return err on failure, stay not-running); reset ring; `epoch++`; spawn read loop (`ReadFromUDPAddrPort` into a 64 KB buf → `parse` → `ring.Add` on ok), `defer close(done)`; set running. `Stop()`: under mutex grab conn+done, set not-running, then (outside mutex) `conn.Close()` and `<-done`; drop the ring. `PollSince`: snapshot + running/port/epoch (cap = ring capacity).
- [ ] **Step 4: Run with race** — `go test -race ./internal/collector/` → PASS.
- [ ] **Step 5: Commit** — `feat(collector): in-memory UDP collector core (ring + lifecycle)`.

---

### Task 2: Syslog parser

**Files:**
- Create: `internal/collector/syslog.go`, `internal/collector/syslog_test.go`

**Interfaces:**
- Consumes: nothing beyond stdlib.
- Produces:
  - `type SyslogMsg struct { Seq int \`json:"seq"\`; T time.Time \`json:"t"\`; Source string \`json:"source"\`; Facility int \`json:"facility"\`; Severity int \`json:"severity"\`; SeverityName string \`json:"severityName"\`; Tag string \`json:"tag"\`; Message string \`json:"message"\` }`
  - `func ParseSyslog(datagram []byte, src netip.Addr) (SyslogMsg, bool)` — the `parse` for the Collector. `T` = `time.Now()` (receive time).

- [ ] **Step 1: Write failing tests.**
```go
func TestParseSyslogPRI(t)        // "<34>Oct 11 22:14:15 host app: msg" → Facility 4, Severity 2, SeverityName "crit", Message contains "msg", Source = src.String()
func TestParseSyslogSeverityNames(t) // <0>→emerg … <7>→debug mapping
func TestParseSyslogNoPRI(t)      // a line with no "<..>" → kept as raw Message, Severity defaults (e.g. 6/info), bool true
func TestParseSyslogTruncatesAndHex(t) // a 5 KB message truncates to ~2 KB; a datagram with NUL/binary → hex or safely truncated, no panic
func TestParseSyslogEmpty(t)      // empty datagram → ok=false (nothing added)
```
- [ ] **Step 2: Run, verify fail.**
- [ ] **Step 3: Implement** `ParseSyslog`: empty → false; read optional `<PRI>` (digits), `Facility = PRI/8`, `Severity = PRI%8`, name via a 0–7 table; best-effort skip an RFC3164 timestamp + host/tag (`tag:` prefix) for `Tag`/`Message`, else whole remainder is `Message`; truncate `Message` to the cap and hex-render if non-printable (reuse the printable/hex approach from `snmp.formatMIBValue`'s logic — reimplement locally to avoid a cross-package dep). `T = time.Now()`.
- [ ] **Step 4: Run, verify pass.** Full `go test ./...` green.
- [ ] **Step 5: Commit** — `feat(collector): RFC3164 syslog parser`.

---

### Task 3: App syslog methods (singleton + lifecycle)

**Files:**
- Create: `internal/app/syslog_methods.go`, `internal/app/syslog_methods_test.go`
- Modify: `internal/app/app.go` (a `syslog *collector.Collector[collector.SyslogMsg]` field + init; `OnShutdown` stops it)

**Interfaces:**
- Consumes: `collector.New`, `collector.ParseSyslog`, `collector.SyslogMsg`, `collector.Poll`.
- Produces: `func (a *App) SyslogStart(port int) error`, `func (a *App) SyslogStop()`, `func (a *App) SyslogPoll(afterSeq int) collector.Poll[collector.SyslogMsg]`.

- [ ] **Step 1: Write failing tests.**
```go
func TestSyslogStartPollStop(t)   // SyslogStart(0-or-free port); learn port via SyslogPoll; dial UDP + send "<11>hi"; Eventually SyslogPoll(0).Items has a msg; SyslogStop(); Poll running=false
func TestSyslogStartBindInUse(t)  // hold a UDP port; SyslogStart(thatPort) → error
func TestSyslogRestartEpoch(t)    // Start; Start again different free port → epoch bumped
```
(Use a helper to grab a free UDP port.)
- [ ] **Step 2: Run, verify fail.**
- [ ] **Step 3: Implement.** In `app.go` add the `syslog` collector (constructed in `New` with capacity 10000 + `collector.ParseSyslog`); `OnShutdown` also calls `a.syslog.Stop()`. The three methods delegate to it (`SyslogStart` returns the bind error; `SyslogPoll` returns the `Poll`). The collector's own mutex is the singleton guard.
- [ ] **Step 4: Run with race** — `go test -race ./internal/app/ -run TestSyslog` → PASS. Full `go test ./...` green.
- [ ] **Step 5: Commit** — `feat(app): syslog receiver methods (start/stop/poll)`.

---

### Task 4: LiveTail component + Syslog receiver tool

**Files:**
- Create: `frontend/src/lib/collectors/LiveTail.svelte` (shared), `frontend/src/lib/tools/SyslogReceiver.svelte`
- Modify: `frontend/src/lib/Workbench.svelte` (new "Collectors" group + entry + branch)
- Regenerate: `frontend/wailsjs` bindings

**Interfaces:**
- Consumes: regenerated `SyslogStart/SyslogStop/SyslogPoll`, `collector.Poll`/`SyslogMsg`.
- Produces: a reusable `LiveTail` (port field + Start/Stop, filter + optional severity floor, newest-first rows, status line, buffer note) driven by callbacks; `{ id: "syslog", name: "Syslog receiver", group: "Collectors", ready: true }`.

- [ ] **Step 1: Build `LiveTail.svelte`** — props: `port` (bind:), `running`, `placeholder`, columns/row renderer via a slot or a `rows` + `renderRow` prop, `filter` (bind:), an optional `severities` list for the floor dropdown, and `start`/`stop` callbacks. It owns the ~700 ms poll loop? No — keep LiveTail presentational; the tool owns polling. LiveTail renders: the control row (port, Start/Stop), the filter row (filter input + optional severity select + Clear), the status line (`listening on :<port> · N messages from M hosts · showing K`), and the scrollable newest-first list via a `{#each}` over the tool-supplied filtered rows. (Keep it dumb; the tool passes filtered rows + status numbers.)
- [ ] **Step 2: Build `SyslogReceiver.svelte`** — state: `port=5514`, `running`, `rows: SyslogMsg[]`, `lastSeq`, `epoch`, `filter`, `sevFloor`, `total`, `dropped`, `err`. `start()`: `await SyslogStart(port)`; on throw set `err`, stay stopped; else `running=true`, start the poll interval. Poll every ~700 ms: `const p = await SyslogPoll(lastSeq)`; if `p.epoch !== epoch` reset rows+lastSeq (restart detected); append `p.items` (map `{seq, val}`), update `lastSeq` to the max seq, `total`/`dropped`/`running`/`port`. Cap local rows at 10000. `stop()`: `await SyslogStop()`; clear interval; `running=false`. `onMount`: poll once from 0 to restore state after a tool switch; start interval only if running. `onDestroy`: clear interval (leave the collector running). Derived `shown` = rows filtered by `filter` (contains over source+message+tag) and `sevFloor` (severity ≤ floor), newest-first; `hosts` = distinct sources. Severity coloring: ≤3 bad, 4 warn, else dim. Render via `LiveTail`.
- [ ] **Step 3: Wire Workbench** — import, add the "Collectors" group entry, add the `{:else if active === "syslog"}<SyslogReceiver />` branch.
- [ ] **Step 4: Regenerate bindings** — `wails generate module`; `git checkout -- frontend/wailsjs/runtime`. Confirm `SyslogStart/Stop/Poll` in `App.d.ts` and `collector.SyslogMsg`/`collector.Poll` in `models.ts`.
- [ ] **Step 5: Build** — `cd frontend && npm run build` → green.
- [ ] **Step 6: Manual check** (dev server + Playwright, real backend): Start on a free high port (e.g. 5514); from a tiny UDP sender (a Go/bash one-liner, or `node -e`/`printf | nc -u`) send a few `<PRI>…` syslog lines to 127.0.0.1:5514; confirm rows appear newest-first with severity colors, the filter narrows, the severity floor works, the count/hosts update; switch to another tool and back — the tail is restored and still live; Stop clears; a bind-in-use Start shows the error. Regression: another tool still renders. (If driving a real UDP send is impractical in the harness, stub `SyslogPoll` to return canned `items`/`epoch` and verify the render/filter/restore paths; note which was used.)
- [ ] **Step 7: Commit** — `feat(app,ui): Syslog receiver tool + live-tail collector UI`.

---

## Self-Review notes

- Spec coverage: Task 1 = framework (ring + lifecycle, the shared base for traps); Task 2 = syslog parse; Task 3 = singleton app methods + shutdown; Task 4 = the shared LiveTail + syslog tool. Traps are slice 2 (reuse Task 1 + LiveTail with a trap parse + tool). Non-goals (persistence, NetFlow/TFTP, forwarding) are out.
- Review Focus: each line maps to a Task 1/2/3 test or the Task 4 manual check.
- Types: `collector.Poll[SyslogMsg]` (Task 1/3) is what the tool consumes (Task 4); `SyslogMsg` JSON tags match the row fields; `ParseSyslog` is the `parse` the Collector holds.
- Proportion: the plan is signatures + test names + the concurrency/lifecycle rules the implementer can't invent; bodies are left to the implementer.
