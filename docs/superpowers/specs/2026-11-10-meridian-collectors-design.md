# Meridian — Collectors (syslog + SNMP trap receivers) — Design

**Status:** approved mockup (all three themes) + decisions, ready to plan.
**Author:** assistant, owner-approved (laptop/ephemeral framing, syslog+traps, live tail + filter, high default ports).
**Depends on:** the SNMP engine (gosnmp, for the trap listener); the existing app event/method patterns.

## Intent / why

A network engineer on a laptop wants to point a device's **syslog** or **SNMP
traps** at Meridian and watch what it says *right now* — while working on that
box — then stop and move on. This is a quick look, not a retention system: if
someone needs days of logs with search and alerting, that's Graylog's job. So
these collectors are **ephemeral and in-memory**: they run while the app is
open, keep the most recent messages in a bounded buffer, and clear on stop.

## Goals

- A **Syslog receiver** and an **SNMP trap receiver**, each a **live tail with a
  filter**: newest first, `time · source · (severity|trap) · message`, with a
  free-text contains filter (host/text) and — for syslog — a severity floor.
- Collecting **survives tool switches**: the listener and buffer live in the
  backend, so leaving the tool and returning shows what has landed meanwhile.
- **In-memory, bounded, ephemeral**: a rolling buffer of ~10,000 messages
  (oldest dropped); **nothing written to disk**; the buffer clears on Stop and
  the listener auto-stops when the app closes (no orphaned socket).
- **High default ports** (syslog 5514, traps 1162), **editable** — so it binds
  instantly on a laptop with no admin; an elevated user can set 514/162.
- Offline, cross-platform. For traps, the v2c community (if any) is never shown
  back or written into an error.

## Non-goals (this slice/project)

- **No persistence** — no disk store, no retention across app restart, no export.
- **No days-long daemon** / no running when the app is closed.
- **No forwarding/relaying** and **no hosting for others** — these receive the
  user's own device output for a local look, nothing more.
- **No NetFlow / TFTP** collectors yet (the framework should make them easy
  later, but they're out of scope here).
- No search beyond the contains-filter; no correlation across collectors; no
  SNMPv3 traps (v2c/v1 via gosnmp).

## Anti-goals (failure even if it "works")

- An orphaned UDP listener outliving the app or a stopped collector.
- Unbounded memory growth under a chatty sender.
- The UI thread blocking on the socket or on a burst of messages.
- A trap's community string surfacing in a row or error.

## Safety / exposure

To receive a device's syslog or traps the listener must bind the machine's
network interfaces (`0.0.0.0`), not loopback — so **while a collector is running
it holds an open UDP port reachable from the LAN**. That is inherent to being a
receiver and is not "hosting for others": it is read-only, accepts no commands,
stores nothing to disk, forwards nothing, and auto-stops when the app closes or
the user hits Stop. Because the input is unauthenticated and attacker-reachable,
the parsers must be **defensive**: bounded reads, truncation, no panic on
malformed input, and never executing or reflecting payload. The owner sees the
listening state and port whenever the tool is open.

## Decisions (owner-approved)

- Two collectors this project: **syslog** and **traps**. (NetFlow/TFTP later.)
- **Live tail, newest-first**, with a contains filter + a syslog severity floor.
- **In-memory rolling buffer (~10k)**, cleared on Stop; **high editable ports**.
- Built in **two slices**: (1) the framework + syslog receiver, (2) the trap
  receiver reusing it.

## Shape (what the builder implements)

**Framework (`internal/collector`)** — a generic bounded, sequenced ring plus a
listener lifecycle:
- A `Ring` holding the last N items, each tagged with a monotonic `Seq`, with
  `Add`, `Since(seq) → items with Seq>seq`, `Len`/`Dropped`. Concurrency-safe.
- A collector owns a UDP listener goroutine that parses datagrams into typed
  messages and `Add`s them; `Stop` closes the socket and blocks until the
  goroutine exits; the buffer is discarded on stop.

**Transport contract (poll-since-cursor, not per-message events)** — chosen so a
burst can't flood the event bus and so "survives tool switches" is trivial. Each
collector is a **singleton per kind**, held on `App` under a mutex. The app
exposes, per kind:
- `Start(port) error` — binds synchronously so a bind failure (port taken →
  EADDRINUSE, privileged → EACCES) returns as friendly text; port must be
  1–65535. **Start while already running = restart**: stop-and-wait, then bind
  the new port, resetting the buffer and bumping the epoch.
- `Stop()` — closes the socket, blocks until the listener goroutine exits, and
  discards the buffer. Never holds the app mutex while waiting.
- `Poll(afterSeq) → { running bool, port int, epoch int, total int, dropped int, messages []Msg }` — returns messages with `Seq > afterSeq`. **The per-poll cap equals the ring capacity**, so one poll can return the whole buffer and a mount-time restore (afterSeq 0) is never truncated; a slow/away client simply gets everything still retained. `epoch` lets the client detect a restart and reset its cursor; `dropped` is the ring's lifetime eviction count (surfaced as "N dropped — buffer full"). Poll never blocks and returns `running=false` with no messages after a Stop (no error).

The frontend polls on a short interval (~700 ms) while the tool is mounted,
passing the last `Seq` it holds (and watching `epoch`); on mount it polls from 0
to restore the current buffer + running/port state (the tool is destroyed on a
Workbench switch, so re-mount is the restore path). `OnShutdown` stops every
collector (waiting on each goroutine without the app mutex).

**Concurrency & resources:** the listener goroutine `Add`s under the ring's lock;
`Poll` copies the matching slice **under the lock and marshals outside it**, so a
slow Wails marshal can't stall the listener. The UDP read buffer is bounded (max
datagram), and `Message`/varbind values are **truncated** (~2–4 KB) with
non-text rendered hex — so a 10k ring stays well bounded regardless of sender.

**Syslog** — a UDP listener; parse the `<PRI>` to facility + severity (0–7) and
best-effort RFC3164 (host/tag/message). `T` is the **receive time** (reliable;
RFC3164 has no year/zone and device clocks drift) — any parsed device timestamp
is kept separately. Unparseable payloads keep the raw (truncated) text as the
message. `SyslogMsg{Seq, T, Source, Facility, Severity, SeverityName, Tag, Message}`.

**Traps** — a **raw UDP read loop sharing the same collector lifecycle** (bind,
read, close, wait), decoding each datagram with gosnmp's packet unmarshal rather
than gosnmp's blocking `TrapListener` — so both kinds share one `Collector` with
a per-kind `parse([]byte, srcAddr) → msg`, and Stop semantics are identical.
`TrapMsg{Seq, T, Source, TrapOID, TrapName (best-effort for well-known OIDs),
Varbinds []{OID, Value}}`; the packet's community is **never copied** onto the
message. (The framework in slice 1 is built around this shared raw-UDP
lifecycle, so slice 2's trap collector only adds a parse function + tool.)

**Frontend** — a shared **LiveTail** component (port field + Start/Stop, filter +
optional severity floor, the row list newest-first, the status line, the buffer
note) parameterized by its columns and severity handling; two thin tools
(`SyslogReceiver`, `TrapReceiver`) in a new **Collectors** group. Rows past a
severity threshold are colored; the filter is client-side over the polled buffer.

## Left to the builder

- Exact Go types/JSON tags; the `Ring` capacity constant; the per-poll message
  cap; the poll interval.
- The syslog PRI/RFC3164 parse details and the hex/printable handling of a
  non-text payload; the trap OID→name table's (small) coverage.
- Whether `Start` on an already-running collector is a no-op, a restart on a new
  port, or an error (builder's call; a clear, non-crashing behavior either way).
- The LiveTail component's prop shape and how the two tools pass columns.
