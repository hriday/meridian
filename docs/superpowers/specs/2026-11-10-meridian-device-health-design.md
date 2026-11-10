# Meridian — Device health (multi-device CPU/memory monitor) — Design

**Status:** approved mockup (all three themes), ready to plan.
**Author:** assistant, with owner approval (NOC view chosen over single-device; mockup approved).
**Depends on:** the SNMP engine + realtime-metrics foundation already shipped (PRs #34, #35).

## Intent / why

A network engineer wants one screen that keeps a chosen set of devices — core
switches, distribution/access switches, routers, firewalls — in front of them
and makes the box that is in trouble *stand out*. Not a per-interface view (the
interface monitor already covers that), and not a deep single-device dashboard:
a **NOC grid**. You add the devices you care about; each one shows CPU and
memory at a glance; anything past its threshold colours; anything that stops
answering is flagged. The one hot or unreachable box is obvious without reading
numbers.

## Goals

- Watch **several devices at once** over SNMPv2c, polled independently.
- Per device, show: **CPU %** and **memory %** as horizontal bar meters, a
  **~60 s sparkline**, and a **state word**.
- **Threshold colouring** (known-good vs attention): a metric past its warn
  threshold goes amber, past its critical threshold goes red, and the row is
  tinted so it stands out in the list.
- **Add / remove devices** freely; one device's failure never affects the others.
- **Unreachable** devices are flagged and keep retrying; a device that exposes
  no CPU or no memory OID shows **n/a** for that metric (not a fake 0 or an error).
- **Sort: worst first** toggle to float problem devices to the top.
- Offline, cross-platform, reuses the shipped SNMP engine + realtime foundation.
- The **community string is read-only**, masked on input, and never appears in a
  snapshot or in any error shown to the UI.

## Non-goals (this slice)

- Per-core / per-storage-area breakdown, or a single-device drill-down view.
- Persistence / historical storage (the sparkline is an in-memory window only).
- Alerting, notifications, or thresholds that page someone.
- SNMPv3 (v2c only, matching the engine as shipped).
- A learned per-device baseline band — **deferred**; this slice uses fixed
  thresholds. (Noted below as the obvious next enhancement.)
- Interface throughput (that is the interface monitor).

## Anti-goals (failure even if it "works")

- The UI thread blocking on polling.
- The community string leaking into any snapshot, error, event, or log.
- One dead/slow device stalling the polling or rendering of the others.

## Decisions already made (owner-approved via the mockup)

- **Rows with bar meters**, not the radial gauges the interface monitor uses —
  bars pack tighter and scan better down a list. (Owner picked the NOC view.)
- **Fixed thresholds** for the known-good read: **CPU 75 / 90**, **memory 80 / 90**
  (warn / critical). A metric ≥ warn is amber, ≥ critical is red; the worst of
  the two metrics sets the row's tint and state word.
- **Per-device community**, masked (`type="password"`), never displayed back.
- **One backend poller per device**, each streaming to its own `metrics:<id>`
  event, reusing the shipped `monitor{cancel,done,baseline}` lifecycle — so add
  and remove are independent and a dead device can't stall the rest.
- **5-second poll interval** (device health drifts slower than interface
  throughput and it's lighter on the gear), a separate seam from the 2 s
  interface interval.
- **Sparkline** plots the busier of CPU/memory over the window (or CPU when
  memory is absent), coloured by the same threshold state.
- **Worst-first sort** is a client-side toggle over the device list; default is
  insertion order.

## Constraints

- Go 1.23 + Wails v2 (Svelte/TS). Pure engine code stays under `internal/`; the
  app layer only binds. Offline — no CDN; reuse bundled deps.
- Reuse, do not rebuild: `snmp.ResourceUsage` (CPU = mean of `hrProcessorLoad`,
  memory from `hrStorageRam`, `Has*` false when the device omits them); the app
  `monitor` lifecycle + `registerMonitor`/`stopMonitor`/`stopAllMonitors`; the
  `dial`/`scrub`/`snmpTimeout` seams; `metricStream.ts` (`subscribe`,
  `newStreamID`).
- Backend stays dumb about thresholds: it emits CPU/mem + reachability; the
  **frontend** derives the warn/critical colouring and the sparkline window. This
  keeps thresholds a UI concern (and easy to make configurable later).

## Shape (what the builder implements)

**Backend** — a device-health poller alongside the interface one:
- A per-poll snapshot carrying `cpuPct`, `memPct`, `hasCpu`, `hasMem`, a
  reachability `state` (`ok` | `unreachable` | `error`), and a human `note`.
- `StartDeviceMonitor(streamID, device, community)` — dials, then every
  `deviceInterval` reads `snmp.ResourceUsage` and emits a snapshot on
  `metrics:<streamID>`. First dial/read failure → one `error` snapshot then the
  goroutine exits (the device never answered); a later failure → an
  `unreachable` snapshot and keep polling. Errors scrubbed of the community.
- `StopDeviceMonitor(streamID)` — cancels and blocks until the goroutine exits.
- No baseline method (fixed thresholds this slice).

**Frontend** — a `DeviceHealth` tool:
- An add-device row (device address + masked community + Add), building a client
  list. Each list entry owns its own `streamID`, subscription, latest snapshot,
  and sparkline ring buffer; it generates the id and subscribes *before* Start.
- Per-device row: name/address, CPU bar meter, memory bar meter, sparkline, state
  word + dot. A remove control. Amber/red colouring and row tint from the fixed
  thresholds; **n/a** per metric when `!hasCpu`/`!hasMem`; a greyed row + flag
  when `state` is `unreachable`/`error`.
- A "worst-first" sort toggle. `onDestroy` stops every device's monitor.
- Small reusable bits as needed (a bar-meter and a sparkline), kept dumb like the
  existing `Gauge`.
- New "SNMP & monitoring" group already exists; this tool joins it.

## Left to the builder

- Exact file/module names, the snapshot struct's Go type and JSON tags, and
  whether the bar meter / sparkline are separate components or inline.
- Whether the per-device community is held only in the row object (yes — never in
  a store that could be serialised) and how removal tears down the subscription.
- Sparkline window length and sampling (target ~60 s of history at the 5 s poll).
