# NetFlow collector — v5 + top talkers (slice 1 of 2) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** A NetFlow v5 collector with a live top-talkers view, reusing the collector framework (generalised to a multi-record parse) + a new aggregating poll.

**Architecture:** Generalise `internal/collector` to a multi-record parse (`NewMulti`, `Ring.AddAll`, `Ring.Values(keep)`, `Collector.Snapshot(keep)`); an `internal/netflow` engine (v5 parse → `FlowRecord`s, `TopTalkers` aggregation); a singleton `collector.Collector[netflow.FlowRecord]` on `App` with `NetflowStart/Stop/Top`; a `NetflowCollector` Svelte tool polling the aggregate. Slice 2 adds v9/IPFIX behind the version dispatch.

**Spec:** docs/superpowers/specs/2026-11-10-meridian-netflow-design.md

## Global Constraints

- In-memory, ephemeral: a bounded ring (~65536 flow records); cleared on Stop; auto-stop on shutdown. Default port 2055, editable, bind 0.0.0.0 (non-privileged).
- Defensive v5 parse of untrusted UDP: version==5, count 1–30, `len ≥ 24+48·count`, big-endian `encoding/binary`, no panic, no unsafe.
- Concurrency: aggregation copies in-window records **under the ring lock** then aggregates **outside** it (never stall the read loop). A datagram's records are added under one lock (`AddAll`).

## Review Focus

- **Collector generalisation doesn't break syslog/traps**: `New`'s single-parse path still works (existing collector/app tests green). (Task 1.)
- **v5 parse safety**: malformed/short/non-v5/huge-count datagram → no records, no panic. (Task 2 tests.)
- **Aggregation correctness**: group by `src→dst:port/proto`, sum bytes/packets/flows, window filter, sort desc, top-N; `WindowSecCovered` reports when the ring can't cover the window. (Task 2 tests.)
- **Counts bounded**: Flows/Exporters are in-window, not unbounded lifetime sets. (Task 3 test.)
- **Lifecycle**: Start/Stop/epoch reused (already reviewed); OnShutdown stops it. (Task 3.)

---

### Task 1: Generalise the collector to a multi-record parse

**Files:** Modify `internal/collector/collector.go`, `internal/collector/ring.go`; add tests to `collector_test.go`, `ring_test.go`.

**Interfaces (produced):**
- `Ring.AddAll(vs []T)` — append all under one lock, trim once, counting dropped.
- `Ring.Values(keep func(T) bool) []T` — plain values matching keep, copied under the lock.
- `func NewMulti[T any](capacity int, parse func([]byte, netip.Addr) []T) *Collector[T]`
- `func (c *Collector[T]) Snapshot(keep func(T) bool) []T` — empty when not running.
- `New` unchanged externally: internally the `Collector.parse` field is `func([]byte, netip.Addr) []T`; `New` wraps the single-record parse into a 0/1-element slice. The read loop does `ring.AddAll(c.parse(buf[:n], addr))`.

- [ ] **Step 1: Failing tests.**
```go
func TestRingAddAll(t)        // AddAll([a,b,c]) → Snapshot(0) 3 items seq 1,2,3; AddAll past cap → dropped counted
func TestRingValuesFilter(t)  // Values(keep even) returns only matching plain values
func TestCollectorMulti(t)    // NewMulti whose parse returns 2 records/datagram; send 1 datagram; Eventually Snapshot(all) len==2; running true
func TestCollectorSnapshotFilter(t) // Snapshot(keep) filters; empty after Stop
```
- [ ] **Step 2: Run, verify fail.**
- [ ] **Step 3: Implement.** Add `AddAll`/`Values` to Ring. Change `Collector.parse` to the slice form; `New` wraps (`if m,ok:=single(b,a); ok {return []T{m}}; return nil`); add `NewMulti`; read loop uses `AddAll(c.parse(...))`; add `Snapshot(keep)` (lock c.mu, nil-ring → nil, else `ring.Values(keep)`).
- [ ] **Step 4: Run with race, verify pass — INCLUDING the existing collector tests (regression).** Full `go test ./...` green (syslog/traps unaffected).
- [ ] **Step 5: Commit** — `feat(collector): multi-record parse + filtered snapshot`.

---

### Task 2: NetFlow v5 engine

**Files:** Create `internal/netflow/v5.go`, `internal/netflow/aggregate.go`, `internal/netflow/v5_test.go`, `internal/netflow/aggregate_test.go`.

**Interfaces (produced):**
- `type FlowRecord struct { Exporter, Src, Dst netip.Addr; SrcPort, DstPort uint16; Proto uint8; Bytes, Packets uint64; T time.Time }`
- `type Conversation struct { SrcIP, DstIP string \`json:"srcIp","dstIp"\`; DstPort int \`json:"dstPort"\`; Proto string \`json:"proto"\`; Bytes, Packets uint64 \`json:"bytes","packets"\`; Flows int \`json:"flows"\` }`
- `func ParseV5(datagram []byte, exporter netip.Addr, now time.Time) []FlowRecord` — []nil on a non-v5/short/bad-count packet.
- `func TopTalkers(records []FlowRecord, window time.Duration, now time.Time, byPackets bool, topN int) (top []Conversation, windowSecCovered int)` — filter T ≥ now−window, group by `{Src,Dst,DstPort,Proto}`, sum, sort desc by bytes (or packets), top N; windowSecCovered from the oldest kept record.

- [ ] **Step 1: Failing tests.** A `buildV5(t, recs...)` helper assembling a v5 datagram (24-byte header version 5 + N×48-byte records) via `encoding/binary` big-endian at the real offsets (srcaddr@0, dstaddr@4, dPkts@16, dOctets@20, srcport@32, dstport@34, prot@38).
```go
func TestParseV5(t)            // 2 records → 2 FlowRecords with right Src/Dst/ports/proto/bytes/packets; Exporter set
func TestParseV5Rejects(t)     // version!=5, count 0, count 99 with short body, truncated header → nil (no panic)
func TestTopTalkersGroups(t)   // several records across 2 conversations → grouped, summed, sorted desc by bytes, top-N honoured
func TestTopTalkersWindow(t)   // records older than the window excluded; windowSecCovered reflects the oldest kept
func TestTopTalkersByPackets(t)// byPackets=true sorts by packets
```
- [ ] **Step 2–4: fail → implement → pass (race).** `ParseV5`: bounds first; loop records at offset `24+48*i`; `netip.AddrFrom4`. `TopTalkers`: map `[convKey]*Conversation` (key is a comparable struct of Src,Dst netip.Addr + DstPort + Proto — no string alloc), sum, then slice + `sort.Slice`, cap topN; proto number → name ("TCP"/"UDP"/"ICMP"/number). Full `go test ./...` green.
- [ ] **Step 5: Commit** — `feat(netflow): v5 parser + top-talkers aggregation`.

---

### Task 3: App methods + NetFlow tool

**Files:** Create `internal/app/netflow_methods.go`, `internal/app/netflow_methods_test.go`, `frontend/src/lib/tools/NetflowCollector.svelte`; modify `internal/app/app.go` (a `netflow *collector.Collector[netflow.FlowRecord]` field + init with `NewMulti` + OnShutdown stop), `frontend/src/lib/Workbench.svelte`; regenerate bindings.

**Interfaces (produced):**
- `func (a *App) NetflowStart(port int) error`, `NetflowStop()`, `NetflowTop(windowSec int, byPackets bool) NetflowView`
- `type NetflowView struct { Running bool; Port int; Flows, Exporters, WindowSecCovered int; Top []netflow.Conversation }`

- [ ] **Step 1: Failing app test.** `TestNetflowStartTopStop` — `NetflowStart(0)`; build a v5 datagram (as Task 2) and send it UDP to the polled port; `Eventually` `NetflowTop(60,false)` has a Top row with the expected conversation and `Flows>=1`, `Exporters>=1`; `NetflowStop()` → Running false. `TestNetflowStartBindInUse` — held port → error.
- [ ] **Step 2–4: fail → implement → pass.** `app.go`: `netflow: collector.NewMulti[netflow.FlowRecord](65536, netflowParse)`, where `netflowParse(b, src) []netflow.FlowRecord { return netflow.ParseV5(b, src, time.Now()) }` (slice 2 extends to dispatch by version); OnShutdown stops it. `NetflowTop`: `recs := a.netflow.Snapshot(func(f) bool { return f.T.After(now-window) })` then `netflow.TopTalkers(...)`; `Flows=len(recs)`, `Exporters` = distinct `f.Exporter` in recs, `WindowSecCovered` from TopTalkers; running/port from a status snapshot (`a.netflow.PollSince(0)` gives running/port cheaply, or add a tiny `Status()` — builder's call). Full `go test ./...` green.
- [ ] **Step 5: Build `NetflowCollector.svelte`** — port (default 2055) + Start/Stop, a window segmented control (30/60/300 s), a by-bytes/by-packets toggle, a status line (flows · exporters · "covers last Xs" when < window), and the ranked conversation table (conversation · proto pill · bytes bar relative to the top row + human value · packets · flows). Polls `NetflowTop(window, byPackets)` every ~1500 ms (replace the table each poll — an aggregate, no cursor); restore on mount; clear on Stop; an in-app usage note. Human bytes/packets formatting in the tool.
- [ ] **Step 6: Wire Workbench** — import, add `{ id:"netflow", name:"NetFlow", group:"Collectors", ready:true }`, render branch.
- [ ] **Step 7: Regenerate bindings; `npm run build` green.** Confirm `NetflowStart/Stop/Top` + `netflow.Conversation`/`NetflowView` in the bindings.
- [ ] **Step 8: Manual check** (dev server + Playwright, real send): `NetflowStart` on 2055; send real v5 datagrams with a tiny Python UDP sender emitting the v5 wire format (header + records for a couple of conversations); confirm the top-talkers table ranks them by bytes, the window control + by-packets toggle work, the status shows flows/exporters, restore-across-switch, Stop clears; regression on another Collectors tool.
- [ ] **Step 9: Commit** — `feat(app,ui): NetFlow v5 collector tool (top talkers)`.

---

## Self-Review notes
- Spec coverage: Task 1 = framework generalisation; Task 2 = v5 parse + aggregation; Task 3 = app singleton + tool. v9/IPFIX = slice 2 (version dispatch in `netflowParse` + a decoder lib). Non-goals (persistence, sFlow, reverse-DNS) out.
- Review Focus: regression (Task 1), parse safety + aggregation (Task 2), bounded counts + lifecycle (Task 3 + manual).
- Types: `collector.NewMulti[netflow.FlowRecord]` ← Task 1; `netflow.Conversation`/`NetflowView` ← Tasks 2/3 consumed by the tool.
