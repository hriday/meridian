# Meridian — NetFlow collector (top talkers) — Design

**Status:** approved mockup (all three themes) + decisions, ready to plan.
**Author:** assistant, owner-approved (top-talkers view; v5 + v9/IPFIX; library for templates; two slices).
**Depends on:** the collector framework (`internal/collector` — the bounded `Ring`, the UDP-listener lifecycle) shipped in #40.

## Intent / why

A network engineer points a router/switch's NetFlow export at their laptop to
answer one question fast: **who's moving the most traffic right now, to where,
over what?** Not a flow warehouse — a live top-talkers board for a quick look,
in-memory and ephemeral like the other collectors.

## Goals

- Receive **NetFlow v5 and v9/IPFIX** flow exports on a UDP port (default 2055,
  editable; v9 often 9995/9996). These are non-privileged ports — no admin.
- **Aggregate flows into conversations** (`src → dst:port`, protocol) over a
  **rolling window** (30 s / 1 m / 5 m) and rank them by **bytes** (or packets),
  showing the heavy talkers — bytes bar, packet count, flow count.
- Show **flows received**, **distinct exporters**, and which versions decoded.
- In-memory, ephemeral: a bounded buffer of recent flow records, cleared on
  Stop; auto-stops on app close. Nothing to disk.
- Offline, cross-platform. Defensive against untrusted, attacker-reachable UDP
  (bounded reads, no panic on malformed packets).

## Non-goals

- No persistence / retention / export; no per-flow forensic tail (the raw tail
  was considered and rejected in favour of aggregation).
- No sFlow (packet-sampling, a different protocol) this project.
- No reverse-DNS of IPs by default (latency/noise — a possible later toggle).
- No alerting or baselining.

## Anti-goals

- A hand-rolled, unbounded binary parser on an attacker-reachable port (use a
  maintained decoder for the template-based versions).
- Unbounded memory under a flood.
- The UI blocking on the socket or on aggregation.

## Decisions (owner-approved)

- **Top-talkers aggregated view** (not a raw flow tail).
- **v5 + v9/IPFIX.** v5 is a fixed self-describing format (hand-rolled, trivial);
  v9/IPFIX are template-based and get a **maintained Go decoder library** rather
  than a hand-rolled parser on untrusted input.
- Aggregate by `src → dst:port` + protocol; default window **1 m**, rank by bytes
  (a by-packets toggle).
- **Two slices:** (1) v5 + the aggregation + the view (no new dependency);
  (2) v9/IPFIX via the decoder library, reusing everything else.

## Shape (what the builder implements)

**Why NetFlow doesn't use the live-tail collector as-is:** one datagram carries
*many* flow records (not one message), v9/IPFIX decoding is *stateful* (a
per-exporter template cache), and the view is an *aggregate* (not a cursor of
new rows). So it reuses the framework's bounded `Ring` and UDP-listener lifecycle
but adds a multi-record parse and an aggregating poll.

**Framework tweak:** generalize the collector to a **multi-record parse**
(`parse([]byte, src) → []T`) via a new `NewMulti` constructor; `New` (single)
stays for syslog/traps. The read loop adds each returned record to the ring.

**Engine (`internal/netflow`, slice 1):**
- `FlowRecord{Exporter, SrcIP, DstIP string; SrcPort, DstPort int; Proto int; Bytes, Packets uint64; T time.Time}` (`T` = receive time).
- `ParseV5(datagram []byte, exporter netip.Addr, now time.Time) ([]FlowRecord, bool)` — validate the 24-byte v5 header (version 5, sane count), then `count` × 48-byte records → FlowRecords. Bounded; ok=false on a malformed/short/non-v5 packet.
- `Conversation{SrcIP, DstIP string; DstPort, Proto int; Bytes, Packets uint64; Flows int}` and `TopTalkers(records []FlowRecord, window time.Duration, now time.Time, byPackets bool, topN int) []Conversation` — keep records within the window, group by `(SrcIP,DstIP,DstPort,Proto)`, sum, sort desc by bytes (or packets), return top N.

**App:**
- A singleton `collector.Collector[netflow.FlowRecord]` built with `NewMulti` + a
  parse that dispatches by version (slice 1: v5; slice 2 adds v9/IPFIX), a larger
  buffer (flow records are small; ~50k).
- `NetflowStart(port)`, `NetflowStop()`, and `NetflowTop(windowSec int, byPackets bool) NetflowView` where `NetflowView{Running bool; Port int; Flows, Exporters int; Top []netflow.Conversation}` — reads the collector's current records (via the ring), aggregates, returns. `OnShutdown` stops it.

**Frontend (`NetflowCollector.svelte`):**
- Port + Start/Stop, a window segmented control (30s/1m/5m), a by-bytes/by-packets
  toggle, the status line (flows · exporters · versions), and the ranked
  conversation table (conversation · proto pill · bytes bar+value · packets ·
  flows). Polls `NetflowTop` every ~1.5 s; restores on mount; clears on Stop.
  New **Collectors** group entry. An in-app usage note.

**Slice 2 (v9/IPFIX):** add a decoder (a maintained Go NetFlow/IPFIX library,
verified at build time) behind the version dispatch, with its per-exporter
template cache living in the parse closure (stateful). No view/aggregation
changes. The decoder choice is a builder call, made against the library's real
API, favouring a focused decoder over a full flow pipeline.

## Resolved builder-check decisions

- **Aggregation reads the ring via a focused, in-lock filter, not `PollSince(0)`.**
  Add `Ring.Values(keep func(T) bool) []T` (copies only matching plain values
  under the ring lock) + `Collector.Snapshot(keep) []T` (empty when not running).
  `NetflowTop` passes `keep = f.T ≥ cutoff`, so only in-window records are copied
  and the aggregation runs **outside** the lock (the read loop isn't stalled).
- **IPs are `netip.Addr` inside `FlowRecord`** (comparable → used directly in the
  conversation map key, no per-record string alloc); strings only in the
  `Conversation` output. v5 is IPv4-only.
- **Counts are in-window and bounded**: `Flows` = in-window record count,
  `Exporters` = distinct exporters among in-window records (a spoofed-source
  flood can't grow an unbounded set). `NetflowView` also carries
  `WindowSecCovered` (from the oldest retained in-window record) so a view that
  can't cover the full requested window (ring evicted older flows under load)
  says so, instead of 30s/1m/5m silently converging.
- **Multi-record parse**: `Collector` stores one `parse([]byte, src) []T`; `New`
  wraps a single-record parse into a 0/1-slice form (signature unchanged — existing
  syslog/trap tests untouched); `NewMulti` takes the slice form. A datagram's
  records are added with one lock acquisition via `Ring.AddAll([]T)`.
- **v5 bounds**: require version==5, count 1–30, `len(datagram) ≥ 24 + 48·count`
  (tolerate trailing padding); big-endian `encoding/binary`, no unsafe. Malformed
  → `ok=false`/no records, never a panic.
- Ring ~65536 flow records; `TopN` = 50; no epoch needed for the aggregate view
  (re-poll restores); receive-time windowing confirmed.

## Left to the builder

- Exact Go types/JSON tags; the ring size and top-N; the v5 field offsets; the
  conversation key and byte/packet human formatting (done in the UI).
- Whether the exporter-version counters are surfaced per-version or just "v5/v9/
  IPFIX".
- The specific v9/IPFIX library (slice 2), chosen against its real API.
