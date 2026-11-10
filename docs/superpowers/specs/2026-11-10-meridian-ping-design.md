# Meridian — Enhanced ping

**Date:** 2026-11-10
**Status:** design, ready to plan
**Foundation:** `docs/superpowers/specs/2026-11-08-meridian-design.md` §4.6 ("enhanced ping").

## Intent

Beyond "is it up?", an operator wants to know *how well* a host responds: the
round-trip latency, how much it varies (jitter), and whether any replies are
lost. Meridian should ping one host a configurable number of times and report
min / average / max / standard-deviation / jitter and packet loss, plus the
per-ping sequence — the classic enhanced `ping` an engineer reaches for, with a
plain-language read of what the numbers mean. This is the §4.6 "enhanced ping"
tool, distinct from the existing Ping **sweep** (many hosts, up/down): this is
one host, measured.

## Why it's safe and in scope

ICMP echo to a user-specified host is the most ordinary diagnostic there is —
read-only, user-initiated, no payload beyond the echo. It reuses Meridian's
existing unprivileged ICMP prober (the one the ping sweep already uses), so no
new privilege or capability is introduced.

## What it measures

For `count` echoes to one host, `interval` apart, each with a per-ping `timeout`:

- **Per ping:** sequence number, round-trip time (ms), and whether it replied.
- **Aggregate:** packets sent / received, **loss %**, and over the replies that
  came back — **min / avg / max / stddev** RTT and **jitter** (the mean absolute
  difference between consecutive replies' RTTs, the usual "how jumpy is it"
  figure).
- **Availability:** whether an unprivileged ICMP socket could be opened at all;
  if not, that is reported plainly (some hosts/OSes restrict it) rather than
  shown as "100% loss".

## Scope

**In scope:**
- Resolve the host to an IP (IPv4/IPv6), ping it `count` times, compute the
  stats, and stream each reply as it arrives (for a live CLI/UI view).
- Engine (`internal/ping`), CLI (`ping`), Wails method, Workbench tool under
  "Local network", following the established seams and self-explaining UI.

**Non-goals (future / out):**
- A TCP/UDP "ping" fallback for ICMP-restricted hosts — that semantics belongs
  to the sweep's reachability probe; this tool is ICMP latency, and says so when
  ICMP isn't available. *My call.*
- Continuous/forever ping — a bounded `count` keeps it a measurement, not a
  monitor; a live graph over a long run is a later enhancement.
- Flood/rapid ping — intervals stay humane (default 500 ms).

## UX (Workbench tool, Nielsen-compliant)

- A host input and a count (default 5), with a Ping button. A one-line note says
  what the tool measures and that latency/jitter matter as much as loss.
- While running, each reply appends to a small list (seq · RTT, or "timeout"),
  with a compact RTT sparkline so variance is visible at a glance.
- When done, a stats row: sent/received, loss %, and min/avg/max/stddev/jitter,
  each labelled, with a one-line plain-language verdict (e.g. "Low, steady
  latency" / "High jitter — the connection is uneven" / "N% loss").
- If ICMP isn't available, a clear note explaining why, not a false "all lost".

## Testing posture

- The statistics are a pure function over a slice of replies: unit-tested with
  scripted RTTs (known min/avg/max/stddev/jitter) and loss cases (some lost, all
  lost), with the math asserted to a tolerance.
- `Run` is tested with a **fake pinger** (scripted RTTs/losses) at zero interval:
  it must ping exactly `count` times, aggregate correctly, stream each reply via
  the callback, and stop early on context cancellation. No real ICMP in tests.
