# Realtime Metrics Foundation + Live Ping — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A reusable realtime-metrics foundation — state-colored SVG gauges, a uPlot streaming chart with a known-good band + anomaly/loss highlighting, and a Wails-event backend poller — proven by a **live ping** tool (RTT graph + loss, with "is this known-good or off?" at a glance).

**Architecture:** `internal/metrics` holds a pure, unit-tested **ping aggregator** (rolling window → avg/loss/jitter + known-good band + state). `internal/app` runs a **ping monitor**: a goroutine wrapping `ping.Run` (large count, ctx-cancel to stop) that feeds the aggregator and **emits snapshots via Wails events** (emit behind a seam). The frontend has reusable `Gauge.svelte`, `StreamChart.svelte` (uPlot), and a `metricStream` helper, consumed by a `PingMonitor.svelte` tool.

**Tech Stack:** Go 1.23 (`internal/ping`, goroutines, `context`), Wails v2 runtime events, Svelte + **uPlot** (bundled npm). Tests: `stretchr/testify` for Go; frontend by build + live manual check.

**Spec:** `docs/superpowers/specs/2026-11-10-meridian-realtime-metrics-design.md`

## Global Constraints

- **Pure engine:** `internal/metrics` does no I/O and no Wails — just numbers in, snapshot out.
- **Streaming, non-blocking:** samples are pushed backend→frontend via Wails events; the UI never polls or blocks. Serial probes (one echo per tick, per-probe timeout < interval).
- **Lifecycle:** Stop blocks until the goroutine exits (cancels in-flight); Wails OnShutdown stops all monitors; the Svelte tool calls Stop in `onDestroy`; Pause = Stop; Resume = fresh stream/window (re-applying a frozen baseline). Per-stream-id event name `metrics:<id>`; the frontend generates the id, subscribes, then Starts.
- **State:** `warming` (< ~10 samples) → then worst of {RTT vs band, loss}: `steady` / `anomaly` / `degraded` / `loss`. Loss judged separately from the RTT band. Band from the rolling window until Set baseline freezes it.
- CLI is **not** required for a streaming monitor (no one-shot JSON shape). Commit attribution: `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`; PR ends with the Generated-with line.

## Review Focus

- **Stop has no goroutine leak:** after Stop (or ctx-cancel / OnShutdown), the polling goroutine exits and the in-flight probe is cancelled; no further events emit. (Task 2 test.)
- **ICMP-unavailable at Start** (no raw-socket permission) surfaces a clear terminal error/snapshot, not silent 100% loss. (Task 2 test, via the `available=false` path.)
- **Warm-up & baseline:** before the warm-up count the state is `warming` (no band); Set baseline freezes the band and classification continues against it. (Task 1 tests.)
- **State precedence:** a run with packet loss reports a loss state even when the responding RTTs are in-band; an RTT spike with no loss reports `anomaly`. (Task 1 tests.)
- **Multi-instance events:** two monitors use distinct `metrics:<id>` names so stopping one doesn't kill the other's stream. (Task 2 test asserts per-id emit; frontend uses EventsOff on the specific name.)

---

### Task 1: `internal/metrics` — ping aggregator (pure)

**Files:**
- Create: `internal/metrics/pingagg.go`
- Test: `internal/metrics/pingagg_test.go`

**Interfaces:**
- Produces:
  - `type Snapshot struct { RTTMs float64; AvgMs, LossPct, JitterMs, BandLo, BandHi float64; State string; Warming bool; HasRTT bool }` (State: `warming|steady|degraded|loss|anomaly`; `HasRTT` false when the latest probe was lost)
  - `type Aggregator struct { … }` with `func NewAggregator(window int) *Aggregator`
  - `func (a *Aggregator) Add(rttMs float64, ok bool) Snapshot` — records a sample (ok=false → loss, rtt ignored), updates the rolling window (last `window` samples), returns the current Snapshot.
  - `func (a *Aggregator) SetBaseline()` — freezes the current band as known-good (until reset); classification continues against it.
  - Band (derived): robust center ± spread over the responding RTTs in the window (e.g. median ± k·MAD, k≈3; a sane floor so a dead-steady series still has a non-zero band). Loss%: lost/total over the window. Jitter: mean abs diff of consecutive responding RTTs. State: `warming` while samples < warmupN (≈10); else `loss` if the latest probe was lost or loss% ≥ lossHigh; `degraded` if 0 < loss% < lossHigh; `anomaly` if the latest responding RTT is outside the band; else `steady`.

- [ ] **Step 1: Write failing tests** `pingagg_test.go`:
  - `TestWarmingThenSteady`: feed ~12 in-band samples (e.g. 10±1ms, all ok) → first snapshots `Warming==true`/`warming`; after warmup, `steady`, band brackets 10, Avg≈10, Loss 0.
  - `TestAnomalyOnSpike`: steady warm-up then one 40ms ok sample → `anomaly`, latest RTT above BandHi.
  - `TestLossStates`: inject some `ok=false` → a lost latest sample → `loss`/`HasRTT==false`; a modest loss fraction with an in-band latest → `degraded`; LossPct computed.
  - `TestJitter`: alternating 10/20ms → JitterMs ≈ 10.
  - `TestSetBaselineFreezesBand`: warm up at ~10ms, `SetBaseline()`, then feed 18ms samples → band stays ~10 (frozen) and 18ms reads `anomaly` (would've been in-band if the band had drifted).
- [ ] **Step 2: Run, verify fail.**
- [ ] **Step 3: Implement** `pingagg.go` (ring/window slice; median+MAD; state machine).
- [ ] **Step 4: Run, verify pass**, then `-race`.
- [ ] **Step 5: Commit** (`feat(metrics): ping monitor aggregator`).

### Task 2: App ping monitor — goroutine + Wails event streaming

**Files:**
- Create: `internal/app/pingmon_methods.go`, `internal/app/pingmon_methods_test.go`
- Modify: `internal/app/app.go` (a monitors map + `OnShutdown`), `main.go` (wire `OnShutdown`).

**Interfaces:**
- Consumes: `internal/ping` (`Run`, `Pinger`, `Reply`), `internal/metrics` (Task 1).
- Produces:
  - seams: `var emitEvent = runtime.EventsEmit` (override in tests); `var newPinger = func() ping.Pinger { … }` (reuse the existing ICMP prober; override in tests).
  - `func (a *App) StartPingMonitor(streamID, target string) error` — validates target; spawns a goroutine that runs `ping.Run(ctx, pinger, target, large, 1s, ~900ms, onReply)`, feeding each `Reply` into an `Aggregator` and calling `emitEvent(a.ctx, "metrics:"+streamID, snapshot)` per sample; tracks `{streamID: cancel}` in a mutex-guarded map; if the first probe reports ICMP unavailable, emits a terminal error snapshot and returns/records the error.
  - `func (a *App) StopPingMonitor(streamID string)` — cancels the stream's ctx and waits for the goroutine to finish (blocks, no leak); removes it from the map.
  - `func (a *App) SetPingBaseline(streamID string)` — calls `SetBaseline()` on that stream's aggregator (guard concurrent access).
  - `func OnShutdown(a *App) func(context.Context)` — stops all monitors.

- [ ] **Step 1: Write failing tests** `pingmon_methods_test.go` (override `emitEvent` to a captured sink + `newPinger` to a fake; use a fake that returns scripted RTT/ok, and `ping.resolveIP` is already a seam — set it so no DNS):
  - `TestStartEmitsSnapshots`: a fake pinger returning steady RTTs → captured events on `metrics:<id>` with Snapshot payloads; stop; assert ≥N emitted.
  - `TestStopNoLeak`: Start then Stop → Stop returns after the goroutine exits; no events emit after Stop (capture count stable); (optional `goleak`-style: the stream is removed from the map).
  - `TestUnavailableSurfaces`: fake pinger with `available=false` → a terminal/error snapshot (State error or a returned error), not an endless 100%-loss stream.
  - `TestSetBaseline`: Start, feed, SetPingBaseline → later snapshots reflect the frozen band.
  - `TestTwoStreamsIndependent`: two Starts with different ids → events on both names; Stop one → the other keeps emitting.
- [ ] **Step 2–4:** fail → implement → pass; `-race`.
- [ ] **Step 5: Commit** (`feat(app): live ping monitor with Wails event streaming`).

### Task 3: Frontend foundation — Gauge, StreamChart (uPlot), metricStream

**Files:**
- Create: `frontend/src/lib/metrics/Gauge.svelte`, `frontend/src/lib/metrics/StreamChart.svelte`, `frontend/src/lib/metrics/metricStream.ts`
- Modify: `frontend/package.json` (add `uplot`), import its CSS.

**Interfaces:**
- Produces:
  - `Gauge.svelte` — props `{ value:number, unit:string, label:string, state:"good"|"warn"|"bad"|"idle" }`; a 270° SVG arc colored by state (CSS vars `--good/--warn/--bad`), value + unit + label. No data logic. (Mockup is the reference.)
  - `StreamChart.svelte` — props `{ points:number[], lost:boolean[], bandLo:number, bandHi:number }` (or a ring-buffer handle); a uPlot line that shades the band rectangle, draws out-of-band points red and in-band green, and marks lost samples; colors from CSS vars; **redraws on theme change** (re-read vars). Keeps a fixed-length window.
  - `metricStream.ts` — `subscribe(streamID, onSnapshot): () => void` wrapping Wails `EventsOn("metrics:"+id, …)` and returning an unsubscribe that calls `EventsOff("metrics:"+id)`.

- [ ] **Step 1: Add uPlot** (`cd frontend && npm install uplot@<pinned>`); import `uplot/dist/uPlot.min.css`. Verify it bundles (`npm run build`).
- [ ] **Step 2: Write the three modules.** For dev verification, a tiny throwaway harness (or the Task-4 tool) feeds static/synthetic data.
- [ ] **Step 3: Build** `npm run build`; verify green and that uPlot is in the bundle.
- [ ] **Step 4: Manual check** — render `Gauge` at good/warn/bad and `StreamChart` with a synthetic series (in-band + a spike + a loss) in all three themes; confirm colors, band, anomaly red, and a theme switch redraws correctly.
- [ ] **Step 5: Commit** (`feat(ui): realtime metrics components (gauge, uPlot stream chart)`).

### Task 4: `PingMonitor.svelte` live-ping tool

**Files:**
- Create: `frontend/src/lib/tools/PingMonitor.svelte`
- Modify: `frontend/src/lib/Workbench.svelte` (import; a `{ id:"pingmon", name:"Ping monitor", group:"Local network", ready:true }` entry near Ping; route)
- Regenerate: bindings (`StartPingMonitor`/`StopPingMonitor`/`SetPingBaseline` + the `metrics.Snapshot` model).

- [ ] **Step 1: Regenerate bindings** (`$(go env GOPATH)/bin/wails generate module`; `git checkout -- frontend/wailsjs/runtime`); verify the three methods + `Snapshot` type.
- [ ] **Step 2: Write `PingMonitor.svelte`**: target input, **Start/Pause**, **Set baseline**; on Start, generate a stream id, `subscribe(id, …)`, then `StartPingMonitor(id, target)`; on each snapshot, update the three **Gauge**s (avg RTT / loss% / jitter, `state` from the snapshot) and push the sample into **StreamChart** (with bandLo/hi); show the **state word** up top; Pause → unsubscribe + `StopPingMonitor`; `onDestroy` → Stop. Guard nulls; the three-theme visual language from the mockup.
- [ ] **Step 3: Wire into `Workbench.svelte`** (import, entry, route).
- [ ] **Step 4: Build** `npm run build`; verify green.
- [ ] **Step 5: Manual check** in the running app: Start a ping to a reachable host → gauges + chart stream live, state reads steady; unplug/ping a lossy or unreachable host → anomaly/loss shows; Pause stops the stream (no further updates); switching away and back starts fresh; no console errors; UI stays responsive. Confirm in all three themes.
- [ ] **Step 6: Commit** (`feat(app,ui): Ping monitor (live) tool`).

---

## Self-Review notes

- Spec coverage: aggregator (band/state/baseline) → T1; goroutine poller + Wails streaming + lifecycle/shutdown → T2; Gauge/StreamChart/metricStream + uPlot → T3; live-ping tool → T4. The foundation (T1–T3) is what the later SNMP throughput / CPU-memory monitors reuse.
- Review Focus → named tests: no-leak, unavailable, warm-up/baseline, state precedence, multi-instance.
- Proportion: signatures + test names; aggregator math, uPlot wiring, and gauge geometry left to the implementer (mockup is the reference).
- No CLI (streaming). Frontend verified by build + live manual check (repo has no FE unit tests), including all three themes and UI responsiveness while streaming.
