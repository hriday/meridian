# Meridian — Realtime metrics foundation + live ping

**Date:** 2026-11-10
**Status:** design, ready to plan (mockup validated by owner, all three themes)
**Foundation:** `docs/superpowers/specs/2026-11-08-meridian-design.md` §4.6
(SNMP monitors, realtime graphs) and §4.9 (no network op blocks the UI;
interactions <100ms; results stream).

## Intent

Meridian's monitoring tools (live ping, interface throughput, CPU/memory, …)
all share one realtime shape: **gauges + a streaming chart that answer, at a
glance, "is this known-good, or is something off?"** — not just live numbers.
This slice builds that **shared realtime-metrics foundation once** and proves it
with its first, self-contained consumer: **live ping**. Interface throughput and
CPU/memory reuse the same components once the SNMP engine exists (a later
foundation); live ping comes first because it needs only ICMP, which Meridian
already has.

## The "known-good vs anomaly" principle (owner's framing)

A network engineer watches a realtime screen to confirm **known-good** or spot
an **anomaly**. So the view encodes that judgment, it doesn't make the engineer
do the math:

- A **known-good band** (a normal range) is shown behind the stream; the live
  line sits inside it when healthy.
- **Gauges carry state** — green inside the band, amber/red when a metric breaks
  out — so a glance reads good/off, not just a value.
- **Anomalies are highlighted** on the stream: an out-of-band excursion is drawn
  red, packet loss is marked, and a one-word **state** (steady / degraded /
  loss / anomaly) gives the verdict.
- A **Set baseline** control freezes the current window as the known-good band;
  until pressed, the band is derived from rolling stats over the window.

## Architecture — the foundation

Decided tech (owner): **uPlot** for the streaming line/area chart (tiny, MIT,
canvas, bundled — no CDN), **hand-rolled SVG** for the radial gauges, and
**Wails runtime events** for streaming (no frontend polling).

- **Backend monitor (the poller).** A Go goroutine polls a metric source on an
  interval, maintains a **rolling window**, and emits a **snapshot** per tick
  via `runtime.EventsEmit(ctx, "<stream>", snapshot)`. The snapshot carries the
  new sample, the rolling aggregates (avg / loss / jitter for ping), the
  known-good **band (lo/hi)**, and the derived **state**. All the math — rolling
  stats, band, anomaly classification — lives here in Go, so it is
  **unit-testable** (poller driven by an injected sampler + ticker + emit sink;
  no real Wails runtime in tests). Start/Stop/SetBaseline are App methods.
- **Frontend components (shared, dumb render):**
  - `Gauge.svelte` — a 270° SVG arc: value, unit, label, and a `state`
    (good/warn/bad) that colors the arc. No data logic.
  - `StreamChart.svelte` — a uPlot streaming line keeping a fixed **ring buffer**
    (time window); shades the known-good band, draws out-of-band segments red,
    and marks loss. Appends each snapshot's sample; redraws without blocking.
  - `metricStream.ts` — a tiny helper that subscribes to a stream's Wails events
    (`EventsOn`), hands snapshots to the gauges/chart, and unsubscribes on
    teardown (and on Pause).
- **Stream identity.** Each running monitor has a stream id so multiple tools /
  instances don't collide; Start returns it, Stop/SetBaseline take it.

This is the pattern every later monitor reuses: a different backend poller
(SNMP counter, CPU OID) emitting the same snapshot shape into the same
`Gauge` + `StreamChart`.

## First consumer — live ping

- **Backend:** a ping monitor that pings the target every interval (reusing the
  existing ICMP engine under `internal/ping` / `internal/scan`), tracking RTT
  and loss over the window. Emits snapshots: current RTT, avg RTT, loss %,
  jitter (RTT stddev), band lo/hi, state, and the loss flag for the latest
  sample.
- **Frontend tool** (a new "Ping monitor" / live ping, in **Diagnostics**,
  distinct from the existing one-shot "Ping"): target input, Start/Pause, **Set
  baseline**, the **three gauges** (avg RTT, loss %, jitter — each state-colored
  against the band), the **StreamChart** (RTT over time, known-good band, red
  anomaly segments, loss markers), and the **state word** up top. Validated
  mockup in Day / Night / Green-terminal.

## Scope

**In scope (v1):** the `Gauge` + `StreamChart` + `metricStream` frontend
foundation; the backend monitor/poller pattern with a testable rolling-stats +
band + anomaly aggregator and Wails-event streaming (emit behind a seam);
Start/Pause/SetBaseline; and the **live-ping** tool end to end, held to the
Nielsen/perf bar (UI never blocks; streams at ~1s).

**Non-goals / later (my calls):**
- **Interface throughput** and **CPU/memory** monitors — reuse these exact
  components, but need the **SNMP engine** (a separate foundation + slice) first.
- Persisting history across sessions, alerting/paging (Meridian captures &
  displays, it doesn't page — per the foundation design), and configurable
  multi-series charts.
- Sub-second intervals / very-high-frequency charts (uPlot can, but v1 targets
  ~1s).

## Decided (my calls, from design review)

- **Pinger / ICMP.** The monitor does **one echo per tick** with a per-probe
  timeout **below the interval**; a lost/timed-out probe is a sample with a
  `lost` flag (RTT omitted), not an error. If the existing `scan.Pinger` seam is
  count/sweep-based, a thin single-probe adapter (or method) on `internal/ping`
  is **in scope**. A raw-socket/permission failure **at Start** surfaces as a
  clear error to the user (not silent 100% loss).
- **Poller lifecycle.** **Serial loop** — never run probes concurrently; if a
  probe outlasts the tick, skip the next tick (keeps samples ordered so the ring
  buffer and jitter math stay correct). **Stop blocks** until the goroutine
  exits and cancels the in-flight probe (the leak test asserts this). Wails
  **OnShutdown stops all** monitors. The Svelte tool **calls Stop in
  `onDestroy`** so switching tools never orphans a poller. **Pause = Stop** the
  backend poller (not just a frontend unsubscribe); **Resume** starts a fresh
  stream + window, re-applying a frozen baseline if one was set.
- **Wails events.** **Per-stream-id event name** (`metrics:<id>`) so one
  instance's `EventsOff` can't remove another's listeners. The **frontend
  generates the id**, subscribes, **then** calls `Start(id, …)` — no first-tick
  race. Snapshots are **incremental** (new sample + rolling aggregates + the
  current band + state); the **chart keeps its own ring buffer**. A remount /
  reload starts a fresh stream (losing prior on-screen history is acceptable for
  a live monitor).
- **Aggregator & state.** Below a warm-up count (~10 samples) state is
  **`warming up`** (neutral gauges, no band yet). The band is **derived from the
  rolling window** (a robust spread, e.g. median ± k·MAD) until **Set baseline**
  **freezes** it (frozen until Stop or re-press; classification continues against
  the frozen band). **Loss is judged separately** from the RTT band; the emitted
  **state is the worst of {RTT-vs-band, loss}** with fixed thresholds pinned by
  the aggregator tests (e.g. `steady` in-band + ~no loss; `anomaly` RTT
  out-of-band; `degraded` for mild loss; `loss` when the current probe dropped or
  loss% is high).
- **uPlot.** An **npm dependency**, Vite-bundled (offline, same-origin — no CSP
  issue). The **band is one shaded rectangle** for the current band (not a
  per-sample series); out-of-band points draw red. A **theme change triggers a
  redraw** that re-reads the CSS-var colors (uPlot fixes colors at construction).
- **Backgrounding.** Leaving the tool stops its poller (via `onDestroy`); a
  minimized-but-mounted window keeps streaming (cheap at ~1s).

## Testing posture

- The **aggregator** (rolling avg/loss/jitter, band from the window or a frozen
  baseline, state classification) is a pure Go unit — direct tests: a steady
  series → `steady` + a tight band + green gauges; an injected spike → `anomaly`
  + the sample out-of-band; injected loss → loss counted and flagged; SetBaseline
  freezes the band. The **poller** is tested with an injected sampler + manual
  tick + a capturing emit sink (no real Wails runtime): start emits snapshots at
  the interval, Stop halts them, no goroutine leak.
- The **ping monitor** is tested with a fake pinger (the existing seam) feeding
  RTT/loss, asserting the snapshot fields.
- `Gauge` / `StreamChart` / the tool are verified by **build + live manual
  check** (no frontend unit tests in the repo), including all three themes and
  that the UI stays responsive while streaming.

## Left to the builder

- The exact snapshot struct and stream-id scheme; the Wails event names.
- The rolling-window size / chart time window and the band formula (e.g. median
  ± k·MAD vs mean ± 2σ) — pick one that's stable and legible.
- uPlot wiring (bundling, the streaming update call, theming via CSS vars) and
  the SVG gauge geometry (the mockup is the reference).
- Whether the live-ping tool is a new entry or an enhancement of the existing
  Ping tool (recommend new, so the one-shot stays).
- The poller's concurrency/stop semantics and context handling.
