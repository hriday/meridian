# Meridian — Local-network tranche: Ping sweep + Port scanner (design)

**Status:** approved by owner (2026-11-09) with the two decisions below.

## Intent and why
Fill the stubbed **Local network** group with the two everyday recon tools a
network engineer runs on *their own* networks: **Ping sweep** (which hosts on
this subnet are up?) and **Port scanner** (which ports are open on this host?).
Diagnostics, not attack tooling — the operator enters the target; there is no
stealth, evasion, or mass-internet scanning.

## Decisions (confirmed)
1. **Both tools in this slice** (they share the concurrency/target/result machinery).
2. **Ping sweep = unprivileged ICMP echo + TCP-connect fallback:** ICMP (reusing
   the traceroute datagram-socket approach) is primary; hosts that filter ICMP
   are confirmed by a TCP connect to a common port.

## Goals
- **Ping sweep:** given a CIDR (e.g. `192.168.1.0/24`), enumerate host addresses
  and report which are **alive**, with RTT and the detection method (icmp/tcp),
  and best-effort reverse DNS. Bounded concurrency; a per-host timeout; progress
  streamed as results arrive. Guard the range size (refuse absurd sweeps, e.g.
  prefixes larger than a configurable host cap).
- **Port scanner:** given a host and a set of ports (list and/or ranges, plus a
  "common ports" preset), **TCP-connect scan** → each port **open** (connected),
  **closed** (actively refused), or **filtered** (timed out). Map well-known
  ports to service names. Bounded concurrency; per-port timeout; streamed
  results. (SYN/stealth scanning needs raw sockets/root — deferred.)
- **CLI:** `meridian-cli pingsweep <cidr>` and `meridian-cli portscan <host> <ports>`.
- UI: two tools in the Local-network group following the "scan" template — a
  target/options row, a run button with live progress, and a results table
  (host/RTT/method for sweep; port/state/service for scan). Plain-language
  captions; a one-line "scan networks you're authorized to" note.

## Non-goals / deferred
- SYN/stealth/UDP/OS-fingerprint scanning; service/version detection beyond a
  name map; banner grabbing; scanning a host *range* for ports (single host per
  scan this slice); OS detection; scheduling; saved scan history.
- Any form of evasion, rate-limit dodging, or scanning targets the operator
  doesn't control. The UI carries an authorized-use note.

## Constraints (global)
- Go directive **1.23**; reuse `golang.org/x/net/icmp`/`ipv4`/`ipv6` (already a
  dep, used by traceroute) for the ICMP pinger; otherwise stdlib (`net`,
  `net/netip`, `context`). No new deps. Additive only.
- The ICMP prober sits behind an injected **`Pinger` seam** so the sweep engine
  and the TCP paths are unit-tested with a fake pinger + local listeners — no
  raw sockets or real network in tests.
- Bounded worker pools, context-cancellable, per-probe timeouts; no unbounded
  goroutine fan-out even on a /16-sized (capped) range.
- Unprivileged ICMP may be unavailable (platform/policy); the engine degrades to
  the TCP fallback and says so, never crashes.

## What the builder decides
- Package layout (suggest `internal/scan`), the exact `Pinger` interface, the
  common-port/service map contents, the host-count cap and default concurrency/
  timeouts, port-spec parsing (`22,80,443,8000-8100`), and the Svelte layout
  within the scan template.
- Whether to reuse traceroute's ICMP code directly or write a focused echo
  pinger in `internal/scan` — builder's call, as long as ICMP stays unprivileged
  and behind the seam.
