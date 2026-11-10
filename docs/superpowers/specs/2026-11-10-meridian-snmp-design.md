# Meridian — SNMP engine + live interface monitor

**Date:** 2026-11-10
**Status:** design, ready to plan (mockup validated by owner, all three themes)
**Foundation:** `docs/superpowers/specs/2026-11-08-meridian-design.md` §4.6
(SNMP monitors, sweep, MIB browser, walk, trap receiver). Builds on the
realtime-metrics foundation (`…-realtime-metrics-design.md`): reuses the gauges,
the uPlot stream chart, and the Wails event-poller pattern.

## Intent

An SNMP **engine** that the whole SNMP tool cluster will share, proven by its
first user-facing consumer: a **live interface monitor** — point Meridian at a
switch/router, pick an interface, and watch in/out throughput, link
utilisation, CPU and memory in realtime, with the same "known-good vs off"
read. The engine also seeds the later SNMP tools (sweep, MIB browser, walk,
trap receiver) and the CPU/memory monitor.

## The SNMP engine (shared)

- **Library:** `gosnmp` (the standard Go SNMP library) — a dependency.
- **Versions:** **SNMPv2c** (community string) in v1; **SNMPv3**
  (user + auth + priv) is a planned follow-up (it adds a credential/auth-model
  layer, not a new tool). *Owner's call: v2c first.*
- **Engine package** (`internal/snmp`) wraps a gosnmp session behind a small
  interface so tools call it and tests inject a fake:
  - `Get(oids…)` and `Walk(rootOID)` — the primitives.
  - Higher-level helpers the monitor needs: **ListInterfaces** (walk IF-MIB:
    `ifDescr`/`ifAlias`/`ifType`/`ifSpeed` or `ifHighSpeed`/`ifOperStatus` →
    one row per ifIndex), **InterfaceCounters** (`ifHCInOctets`/`ifHCOutOctets`
    for an ifIndex, with the 32-bit `ifInOctets` fallback), **System**
    (`sysDescr`/`sysName`/`sysUpTime`), and best-effort **CPU/Memory**
    (HOST-RESOURCES-MIB `hrProcessorLoad`, `hrStorage*`; absent on many devices).
- **Credentials are sensitive:** the community string (and later v3 secrets) are
  entered in-app, **masked** in the UI, sent only to the target device, never
  committed/logged. v1 may remember them locally per device via the existing
  `internal/settings` (0600) — the builder's call; otherwise per-session.

## First consumer — live interface monitor

- **Pick:** device IP + community → **ListInterfaces** populates an interface
  picker (name/alias, speed, up/up). Pick one, Start.
- **Poll (backend goroutine, ~2s):** read `ifHCInOctets`/`ifHCOutOctets` (and
  CPU/mem if present) each tick; a **throughput aggregator** turns the octet
  **counter deltas** into **in/out bps** (handling counter wrap and the first
  tick), computes **link utilisation** (bps ÷ ifSpeed), and keeps a rolling
  window + a known-good band + a saturation **state** (steady / busy /
  saturated). Streams a snapshot per tick on `metrics:<id>` — same pattern as
  the ping monitor (Stop blocks, OnShutdown stops all, emit/interval seams).
- **Screen (validated mockup):** device/community/interface header + Start/Pause;
  three state-colored **gauges** — link util, CPU, memory (CPU/mem hidden or
  "n/a" when the device doesn't expose them); a dual-line **StreamChart** — in
  (rx) and out (tx) Mbps with the normal band; an **interface detail row**
  (speed, oper status, errors, current in/out); the **state word** up top. Day /
  Night / Green-terminal.

## Architecture (Meridian conventions + reuse)

- `internal/snmp` — the engine (gosnmp behind an injected interface; no
  persisted state). Unit-tested with a fake client returning scripted OID
  values.
- A **throughput aggregator** (in `internal/metrics`, beside the ping one) —
  pure: counter samples in → in/out bps, util, rolling band + state out.
  Separate from the ping aggregator (different metrics); both feed the same
  frontend components.
- `internal/app` — `ListInterfaces(device, community)` (non-streaming, for the
  picker) and `StartIfMonitor`/`StopIfMonitor`/`SetIfBaseline` (the streaming
  goroutine), mirroring the ping-monitor methods.
- **Frontend reuse:** `Gauge.svelte` as-is; `metricStream.ts` as-is;
  **`StreamChart.svelte` gains a multi-series mode** (plot rx + tx, each its own
  color) — a small extension of the existing single-series component. A new
  `SnmpMonitor.svelte` tool in a new **"SNMP & monitoring"** group.

## Scope

**In scope (v1):** the `internal/snmp` engine (v2c get/walk + the interface/
system/CPU-mem helpers); the throughput aggregator; the app methods (list +
streaming monitor); the multi-series StreamChart; and the **live interface
monitor** tool, held to the Nielsen/perf bar.

**Non-goals / later (my calls):**
- **SNMPv3** — planned next on the same engine.
- The other SNMP tools (**sweep, MIB browser, walk, trap receiver**) and the
  standalone **CPU/memory monitor** — they reuse this engine; separate slices.
- Writing to devices (SNMP set), trap *sending*, and historical storage.
- A bundled MIB database / OID-name resolution beyond the handful of named OIDs
  the monitor needs (the MIB browser slice owns general MIB handling).

## Decided (my calls, from design review)

- **Counter width probed at Start.** Try `ifHCInOctets`/`ifHCOutOctets`; on
  `noSuchObject` fall back to 32-bit `ifInOctets`/`ifOutOctets`. The chosen
  width (64/32) is carried on each sample so the wrap handler knows the modulus;
  the UI notes when it's on 32-bit counters.
- **Errors + oper status each tick.** Poll `ifInErrors`/`ifOutErrors` (shown as
  deltas) and `ifOperStatus` every tick. **A link going down is a STATE**
  (`down`), not an error snapshot — so the user can watch a link drop live.
- **Throughput state = absolute utilisation thresholds** (`steady` < 70%,
  `busy` 70–90%, `saturated` > 90% of link speed); the chart's **known-good
  band is a rolling statistical range** on bps (median ± k·MAD), and
  **`SetIfBaseline` freezes that band** (visual reference), matching the ping
  tool. Util drives the gauges/state word; the band is the chart's reference.
- **Link speed:** prefer `ifHighSpeed`×1e6, else `ifSpeed`. If speed is 0 or
  unusable (tunnels, loopbacks), **util shows "n/a"** and the saturation state is
  dropped (throughput still graphs); no manual speed override in v1.
- **CPU/memory:** HOST-RESOURCES-MIB only in v1 — CPU = **mean** of
  `hrProcessorLoad` cores; memory = the `hrStorageRam`-typed `hrStorage` row
  (used ÷ size × allocation units). **"n/a"** when absent (vendor MIBs are a
  later addition), gauge greyed.
- **StreamChart multi-series:** add an optional `series` prop (array of
  `{values, color, label?}`) with a **single shared band**; the existing
  single-series props stay as a shim so the ping tool is unchanged.
  `metricStream.ts` is payload-agnostic — no change.
- **Credentials:** the community string is a **per-call argument**
  (`ListInterfaces(device, community)`, `StartIfMonitor(id, device, community,
  ifIndex)`), kept only in the tool's session state (v1 does **not** persist it;
  settings-0600 persistence is a fast-follow). It **never appears in an event
  snapshot or an error string** — gosnmp errors are scrubbed; a redaction test
  guards this.
- **Failure semantics:** `ListInterfaces` or the **first** poll failing → a hard
  "device unreachable" error. A **later** tick timing out → a transient
  `unreachable`/stale state, polling continues (recovers if the device returns).
  A counter **decrease** is treated as a wrap only when the previous value was
  near the counter max; otherwise it's a **reset** (device reboot / `sysUpTime`
  drop) and that sample is dropped (no bogus negative/huge bps).
- **gosnmp** is added as a pinned dependency (BSD-licensed) in the engine task.

## Testing posture

- The **engine** is tested with a **fake SNMP client** (injected): ListInterfaces
  parses a scripted IF-MIB walk into rows; InterfaceCounters returns the HC
  octets; CPU/mem helpers degrade gracefully when the OIDs are absent; a get/walk
  error surfaces as a clear error. No real device, no network.
- The **throughput aggregator** is pure-unit: two counter samples → correct
  bps (delta ÷ dt); counter **wrap** handled; the **first tick** emits no bogus
  spike; util = bps ÷ speed; band/state transitions (steady → saturated).
- The **monitor goroutine** is tested like the ping monitor (fake client + emit
  seam + short interval): Start streams, Stop blocks with no leak, unavailable/
  unreachable device surfaces an error snapshot.
- Frontend (StreamChart multi-series, the tool) verified by build + a **live
  manual check** — against `snmpsim` or a real device if available; otherwise the
  no-device/error path and the rendering in all three themes.

## Left to the builder

- The exact engine interface + the fake's shape; gosnmp session options
  (timeout, retries, port 161).
- The precise OID set and the CPU/mem detection heuristic (which MIBs to try,
  how to present "n/a").
- Counter-wrap + first-tick handling specifics in the aggregator; the util band
  formula.
- Whether community strings persist (settings, 0600) or are per-session.
- The multi-series StreamChart API (how the tool passes rx/tx series + colors).
- The "SNMP & monitoring" group placement and the tool id.
