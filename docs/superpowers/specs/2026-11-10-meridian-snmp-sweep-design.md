# Meridian — SNMP sweep — Design

**Status:** approved mockup (all three themes), ready to plan.
**Author:** assistant, owner-approved (mockup + defaults).
**Depends on:** the SNMP engine (`snmp.Dial`, `snmp.System`) shipped in #35; the `scan` package's sweep pattern (CIDR enumeration, bounded concurrency, `MaxSweepHosts` cap, `onResult`, sort-by-IP); the app's existing single-scan cancellation infra (`scanContext()`/`ScanStop()`, `scanEmit`).

## Intent / why

A network engineer pointed at an unfamiliar (or drifted) subnet wants to know
what's actually out there that speaks SNMP: the live inventory. SNMP sweep walks
a CIDR, probes every host for an SNMP response with a read community, and lists
the responders — IP, name, what it is, and how long it's been up. It answers
"what's on this /24?" the way the ping sweep answers "what's alive?", but with
device identity instead of just reachability.

## Goals

- Sweep a **CIDR** (e.g. `10.0.0.0/24`) with a read **community**, probing every
  host for an SNMP response.
- Per responder, show **IP**, **sysName**, **description** (model/OS from
  sysDescr), and **uptime**. Sorted by address.
- **Responders fill in live** as the scan runs; a progress read ("scanned N ·
  M responded") and a **Stop** button.
- Bounded and safe: the existing `MaxSweepHosts` cap rejects an over-broad
  prefix; a non-responding host costs one timeout, not a hang.
- Offline, cross-platform. The community is read-only, masked on input, and
  never shown back or written into an error.

## Non-goals (this slice)

- Writing to devices, or any SET. Read-only.
- SNMPv3 (v2c only, matching the engine).
- A full MIB walk / per-OID browsing (that's the separate MIB-browser tool).
- Vendor classification beyond what sysDescr already states (sysDescr carries
  model + OS; a sysObjectID→vendor lookup is deferred).
- Persisting results across runs.

## Anti-goals

- The UI blocking during the scan.
- The community leaking into a result row, an event, or an error.
- An over-broad CIDR fanning out unbounded.

## Decisions (owner-approved via the mockup)

- **Columns:** address, name, description, uptime. No separate vendor column
  (sysDescr already says the model/OS).
- **Live streaming** of per-host progress as the scan runs, plus a final
  authoritative sorted table; a **Stop** button while running.
- **Single community** for the whole sweep, masked.
- Sensible fixed defaults — **1 s timeout, ~64 parallel** — with no advanced
  panel this slice.

## Shape (what the builder implements)

**Engine** — a sweep in the `snmp` package mirroring `scan.Sweep`:
- A per-host result carrying `ip`, `responded`, and (when it answered) `name`,
  `descr`, `uptimeTicks`.
- `Sweep(ctx, cidr, community, timeout, concurrency, onResult)`:
  parse + mask the CIDR, reject ranges over the host cap, enumerate hosts
  (skipping network/broadcast for IPv4 < /31), and probe each with bounded
  concurrency: dial + `System()`; a successful `System` marks the host a
  responder with its sysName/sysDescr/uptime. `onResult` fires per probed host
  (responder or not — so the UI can show "scanned N"); the return value is the
  responders, sorted by IP. Cancellation via ctx stops launching promptly.
- A `dial` seam (package var defaulting to `Dial`) so the sweep is testable
  offline with a fake client.

**App** — a thin bound method reusing the single-scan infra:
- `SnmpSweep(cidr, community) ([]snmp.SweepHost, error)` — runs the sweep under
  `scanContext()` (so `ScanStop()` cancels it, like the other scans), streams
  each probed host as a `snmp:host` event via `scanEmit`, and returns the
  responders. Any returned error is scrubbed of the community.

**Frontend** — an `SnmpSweep` tool:
- CIDR + masked community inputs, Scan / Stop. Subscribes to `snmp:host` for the
  live progress (count scanned, count responded) BEFORE calling `SnmpSweep`;
  renders the responders table (from the method's return, authoritative + sorted)
  with a progress bar and the "scanned N · M responded" line. Uptime ticks
  formatted human-readably (TimeTicks are centiseconds).
- Joins the "SNMP & monitoring" group.

## Left to the builder

- Exact Go types + JSON tags for the per-host result; whether total host count
  for the progress bar is computed in the frontend from the CIDR or carried in
  an event.
- Uptime formatting helper placement; the empty/scanning/empty-result states.
- Whether `Sweep` dials with `Retries 0` for speed on dead hosts (a dead host
  shouldn't cost retransmits) — a builder call within the engine.
