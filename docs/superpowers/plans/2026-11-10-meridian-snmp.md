# SNMP Engine + Live Interface Monitor — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A shared SNMP engine (gosnmp, v2c) and its first consumer — a live interface monitor streaming in/out throughput, link utilisation and CPU/memory with the realtime gauges + stream chart, "known-good vs off" at a glance.

**Architecture:** `internal/snmp` wraps gosnmp behind an injected `Client` interface (tools call it, tests fake it) with helpers (ListInterfaces, InterfaceCounters, System, ResourceUsage). A pure `metrics.ThruAgg` turns counter samples into in/out bps + utilisation + a band/state. `internal/app` binds `ListInterfaces` (picker) and `StartIfMonitor`/`StopIfMonitor`/`SetIfBaseline` (the streaming goroutine, mirroring the ping monitor, emitting on `metrics:<id>`). The frontend reuses `Gauge`/`metricStream`, extends `StreamChart` to multi-series (rx/tx), and adds `SnmpMonitor.svelte`.

**Tech Stack:** Go 1.23 + `gosnmp` (pinned, BSD), Wails v2 events, Svelte + uPlot. Tests: `stretchr/testify` with a fake SNMP client + emit seam; frontend by build + live manual check.

**Spec:** `docs/superpowers/specs/2026-11-10-meridian-snmp-design.md`

## Global Constraints

- **Engine deps injected:** `internal/snmp` and `internal/metrics` are unit-tested with no real device — the gosnmp session is behind a `Client` interface; the app seams `dial`.
- **Community string is sensitive:** passed per call, kept only in session state, and **never** placed in an event snapshot or an error string (gosnmp errors scrubbed). A redaction test guards it.
- **v2c only** in v1 (v3 later). Counter width probed at Start (HC → 32-bit fallback), carried on the sample for wrap math.
- **Graceful "n/a":** util n/a when speed is 0; CPU/mem n/a when the OIDs are absent; a tick timeout → transient `unreachable`, keep polling; first poll / ListInterfaces failure → hard error.
- Streaming reuses the ping-monitor lifecycle (Stop blocks, OnShutdown stops all, emit/interval seams). Commit attribution: `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`; PR ends with the Generated-with line.

## Review Focus

- **Counter wrap vs reset:** a decrease near the counter max is a wrap (add modulus); a decrease not near max is a reset (drop the sample, no bogus huge/negative bps). (Task 2 tests, both widths.)
- **First tick emits no bogus spike** (no previous sample → warming, bps 0). (Task 2 test.)
- **HC → 32-bit fallback:** a device returning noSuchObject for `ifHCInOctets` falls back to `ifInOctets` and sets the 32-bit width. (Task 1 test.)
- **Util n/a at speed 0** (tunnels/loopbacks) → no divide-by-zero, saturation state dropped. (Task 2 test.)
- **Community redaction:** the community never appears in any snapshot or error. (Task 3 test.)
- **Monitor Stop/no-leak + unreachable:** Stop blocks until the goroutine exits; a scripted get error → an error/unreachable snapshot, not a hang. (Task 3 tests.)

---

### Task 1: `internal/snmp` engine (gosnmp behind an injected Client)

**Files:**
- Create: `internal/snmp/snmp.go` (Client interface, Dial, PDU), `internal/snmp/ifmib.go` (helpers)
- Test: `internal/snmp/ifmib_test.go`
- Modify: `go.mod`/`go.sum` (add `gosnmp`).

**Interfaces:**
- Produces:
  - `type PDU struct { OID string; Value any }`
  - `type Client interface { Get(oids []string) ([]PDU, error); Walk(root string) ([]PDU, error); Close() error }`
  - `func Dial(device, community string, timeout time.Duration) (Client, error)` — a gosnmp v2c session (port 161, sane retries) wrapped as Client.
  - `type Interface struct { Index int; Descr, Alias string; SpeedBps uint64; OperStatus string }`
  - `func ListInterfaces(c Client) ([]Interface, error)` — walk `ifDescr`,`ifAlias`,`ifType`,`ifHighSpeed`(→×1e6)/`ifSpeed`,`ifOperStatus`; one row per ifIndex; speed prefers ifHighSpeed.
  - `type Counters struct { InOctets, OutOctets, InErrors, OutErrors uint64; OperStatus string; Bits int }`
  - `func InterfaceCounters(c Client, ifIndex int) (Counters, error)` — get `ifHCInOctets`/`ifHCOutOctets` (Bits=64); on noSuchObject fall back to `ifInOctets`/`ifOutOctets` (Bits=32); plus `ifInErrors`/`ifOutErrors`/`ifOperStatus`.
  - `type SysInfo struct { Descr, Name string; UptimeTicks uint32 }`; `func System(c Client) (SysInfo, error)`.
  - `type Resources struct { CPUPct, MemPct float64; HasCPU, HasMem bool }`; `func ResourceUsage(c Client) (Resources, error)` — HOST-RESOURCES: CPU = mean of `hrProcessorLoad` rows; Mem = the `hrStorageRam`-typed `hrStorage` (used÷size×units); Has* false when absent.

- [ ] **Step 1: Add gosnmp** (`go get github.com/gosnmp/gosnmp@<pinned>`); `go build ./...`.
- [ ] **Step 2: Write failing tests** `ifmib_test.go` with a `fakeClient` returning scripted PDUs:
  - `TestListInterfaces`: a scripted IF-MIB walk (2 interfaces, one with ifHighSpeed, one tunnel speed 0) → correct rows, speed from ifHighSpeed, oper status mapped.
  - `TestInterfaceCountersHC`: HC OIDs present → Bits 64, octet/error values.
  - `TestInterfaceCounters32Fallback`: HC returns noSuchObject → falls back to 32-bit, Bits 32.
  - `TestResourceUsagePresentAndAbsent`: hrProcessorLoad+hrStorageRam present → CPU mean + Mem%; absent → Has* false.
- [ ] **Step 3–4:** fail → implement → pass.
- [ ] **Step 5: Commit** (`feat(snmp): v2c engine + IF-MIB/host-resources helpers`).

### Task 2: `metrics.ThruAgg` — throughput aggregator (pure)

**Files:**
- Create: `internal/metrics/thruagg.go`
- Test: `internal/metrics/thruagg_test.go`

**Interfaces:**
- Produces:
  - `type ThruSample struct { InOctets, OutOctets, InErrors, OutErrors uint64; Bits int; OperStatus string; CPUPct, MemPct float64; HasCPU, HasMem bool; T time.Time }`
  - `type ThruSnapshot struct { InBps, OutBps, UtilPct float64; HasUtil bool; CPUPct, MemPct float64; HasCPU, HasMem bool; InErrDelta, OutErrDelta uint64; OperStatus string; BandLo, BandHi float64; State string; Warming bool }` (State: `warming|steady|busy|saturated|down`)
  - `func NewThruAgg(window int, speedBps uint64) *ThruAgg`
  - `func (a *ThruAgg) Add(s ThruSample) ThruSnapshot` — bps from octet deltas ÷ dt; wrap when the previous value was near `2^Bits` (add modulus) else drop (reset); first sample → warming, bps 0; util = max(in,out)÷speed (HasUtil when speed>0); error deltas; state = `down` if OperStatus≠up, else by util (>90 saturated, >70 busy, else steady), `warming` while < warmupN; the chart band = rolling median±k·MAD on total bps.
  - `func (a *ThruAgg) SetBaseline()` — freeze the band.

- [ ] **Step 1: Write failing tests** `thruagg_test.go`:
  - `TestFirstTickNoSpike`: one sample → Warming, InBps 0.
  - `TestBpsFromDelta`: two samples 1s apart, InOctets +125000 → InBps 1,000,000; util vs a 1e8 speed.
  - `TestCounterWrap32`: Bits 32, prev near 2³², next small → treated as wrap (plausible bps), not negative.
  - `TestCounterResetDropped`: a large decrease from a mid-range value → sample dropped (bps 0), no huge value.
  - `TestUtilNaAtZeroSpeed`: speed 0 → HasUtil false, no panic, state steady.
  - `TestDownState`: OperStatus "down" → State `down`.
  - `TestSaturated`: util > 90% → `saturated`.
  - `TestSetBaselineFreezes`: freeze band, then higher throughput reads against the frozen band.
- [ ] **Step 2–4:** fail → implement → pass; `-race`.
- [ ] **Step 5: Commit** (`feat(metrics): interface throughput aggregator`).

### Task 3: App — ListInterfaces + streaming interface monitor

**Files:**
- Create: `internal/app/snmpmon_methods.go`, `internal/app/snmpmon_methods_test.go`

**Interfaces:**
- Consumes: `internal/snmp`, `internal/metrics` (ThruAgg), the ping-monitor lifecycle (`emitEvent`, `pingInterval`→reuse or a new `snmpInterval` seam, the monitors map + Stop pattern).
- Produces:
  - seam `var dial = snmp.Dial` (tests override to a fake Client).
  - `func (a *App) ListInterfaces(device, community string) ([]snmp.Interface, error)` — dial, list, close; errors scrubbed of the community.
  - `func (a *App) StartIfMonitor(streamID, device, community string, ifIndex int, speedBps uint64) error` — goroutine: dial once; probe HC/32-bit; each `snmpInterval` tick read counters (+resources), feed a `ThruAgg`, emit a `ThruSnapshot` on `metrics:<id>`; first-poll/dial failure → error; later timeout → an `unreachable` snapshot, keep polling. Reuse the monitors map + Stop(blocks)/OnShutdown.
  - `func (a *App) StopIfMonitor(streamID string)` / `func (a *App) SetIfBaseline(streamID string)` (reuse the generic monitor map; the ping and SNMP monitors can share the `pingMonitor`→rename to a generic `monitor` holding a `cancel`+`done`+a `baseline func()`).

- [ ] **Step 1: Write failing tests** (fake Client via `dial` seam + captured `emitEvent` + short interval):
  - `TestListInterfacesScrubsCommunity`: fake Dial error containing the community → the returned error does NOT contain it.
  - `TestIfMonitorStreams`: fake counters increasing → snapshots on `metrics:<id>` with InBps>0 after the first tick.
  - `TestIfMonitorSnapshotNoCommunity`: no emitted snapshot JSON contains the community string.
  - `TestIfMonitorStopNoLeak`: Stop blocks; no events after Stop.
  - `TestIfMonitorUnreachable`: fake get error on first poll → an error snapshot; (a later-tick error → `unreachable`, still streaming).
  - `TestSetIfBaseline`: baseline affects later snapshots.
- [ ] **Step 2–4:** fail → implement → pass; `-race`. (If sharing the monitor struct with ping, keep the ping monitor tests green.)
- [ ] **Step 5: Commit** (`feat(app): SNMP interface list + live interface monitor`).

### Task 4: Frontend — multi-series StreamChart + `SnmpMonitor.svelte`

**Files:**
- Modify: `frontend/src/lib/metrics/StreamChart.svelte` (add a `series` prop; keep single-series shim)
- Create: `frontend/src/lib/tools/SnmpMonitor.svelte`
- Modify: `frontend/src/lib/Workbench.svelte` (new "SNMP & monitoring" group; entry; route)
- Regenerate: bindings (`ListInterfaces`/`StartIfMonitor`/`StopIfMonitor`/`SetIfBaseline` + `snmp.Interface`).

- [ ] **Step 1: Extend `StreamChart`** with an optional `series: {values:(number|null)[], color:string}[]` prop rendered as N uPlot series; when absent, fall back to the existing single `rtts` series (ping tool unchanged). Band stays one shared rect. `npm run build` green; confirm the ping monitor still renders (regression).
- [ ] **Step 2: Regenerate bindings**; verify the four methods + `snmp.Interface`.
- [ ] **Step 3: Write `SnmpMonitor.svelte`**: device + community (masked) inputs → **List** calls `ListInterfaces` → interface `<select>`; **Start** generates a stream id, subscribes, `StartIfMonitor(id, device, community, ifIndex, speedBps)`; render three gauges (link util / CPU / memory — grey "n/a" when `!hasUtil`/`!hasCpu`/`!hasMem`), a dual-series StreamChart (rx green, tx blue, Mbps), an interface detail row (speed, oper status, error deltas, in/out), and the state word (steady/busy/saturated/down/unreachable). Pause = stop; `onDestroy` stops; error-state teardown to a clean Start (as the ping fix did). Null guards.
- [ ] **Step 4: Wire into `Workbench.svelte`**; **build** green.
- [ ] **Step 5: Manual check** (live if a device/snmpsim is reachable; otherwise the no-device/error path): the picker lists interfaces, Start streams gauges + dual-line chart, "n/a" shows for missing CPU/mem, a bad device shows a clean error, no console errors, all three themes.
- [ ] **Step 6: Commit** (`feat(app,ui): SNMP interface monitor tool`).

---

## Self-Review notes

- Spec coverage: engine (v2c get/walk + helpers) → T1; throughput/util/band/state + wrap/reset → T2; list + streaming monitor + redaction + lifecycle → T3; multi-series chart + tool → T4. The engine seeds the later SNMP tools; the components are the realtime foundation's.
- Review Focus → named tests (wrap vs reset, first-tick, HC fallback, util n/a, redaction, stop/no-leak/unreachable).
- Proportion: signatures + test names; gosnmp wiring, OID specifics, uPlot multi-series, and CPU/mem heuristics left to the implementer.
- No CLI (streaming + a device-dependent list). Frontend verified by build + live manual check; community never persisted in v1.
