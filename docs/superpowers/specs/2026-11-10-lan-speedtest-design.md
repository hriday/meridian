# LAN speed test (iperf3) — design

**Date:** 2026-11-10
**Status:** design (awaiting builder-check, then plan)

## Intent & why

The owner wants a **LAN throughput test** — a bandwidth measurement between two points
on their own network — reusing Meridian's realtime gauge/stream foundation (a live
needle). A LAN throughput test needs a cooperating endpoint; the ubiquitous one is
**iperf3** (`iperf3 -s` on a server/host). The owner chose a **self-contained pure-Go
iperf3 client** so nothing has to be installed on Meridian's side: the library
`github.com/appshubcc/go-iperf3` speaks the iperf3 wire protocol and runs against any
standard `iperf3 -s`. Meridian stays a **client only** — it never serves, honoring the
project rule that Meridian never hosts anything for others.

## Goals

- A **LAN speed test** tool: point it at a host running `iperf3 -s`, pick direction
  (download / upload / both) and options (port, parallel streams, duration, TCP/UDP),
  and watch a **live gauge + throughput chart**, then a result summary (avg bitrate,
  retransmits, RTT; jitter/loss for UDP).
- Reuse the realtime foundation: `Gauge.svelte`, `StreamChart.svelte`,
  `metricStream.ts`, and the `monitor` streaming infra (`registerMonitor`/`stopMonitor`
  + `emitEvent("metrics:<streamID>", …)`), exactly as the ping/SNMP monitors do.
- Clear, friendly errors: no server listening, server busy, bad host/port, cancelled.

## Non-goals / anti-goals

- **Never run a server.** The shipped app only calls the library's client (`RunClient`);
  the library's `Server`/`Listen` is used ONLY in tests (loopback), never in app code.
- Not an internet speed test (Cloudflare) — that was explicitly deferred.
- Not UDP-tuning depth (packet-size sweeps, pacing). Offer TCP (default) and a basic UDP
  mode; don't expose every knob.
- No bundled iperf3 binary — the whole point of the pure-Go client is self-containment.

## The library (evaluated)

`github.com/appshubcc/go-iperf3` v0.1.0 (pure Go, no cgo, dep: `golang.org/x/sys`, MIT).
API used:
- `RunClient(ctx, ClientConfig) State` — synchronous; blocks until done; `ctx` cancels.
- `ClientConfig{ Hosts []string; Port int; Protocol (TCP|UDP); Direction (Upload|Download|Bidir); Parallel int (1..128); Duration time.Duration; Bandwidth float64; OnUpdate func(State) }`.
- `OnUpdate` fires on each phase change and ~every 250 ms while running, with
  `State{ Phase (connecting|running|done); Elapsed; Upload; Download (live bits/s); UploadResult, DownloadResult *Flow; Error }`.
- `Flow{ Bytes; Bitrate; Retransmits; Jitter; Lost; Packets; RTTMin/Mean/Max }`.
- Defaults: Port 5201, Duration 10s, MaxStreams 128, MaxDuration 86400.

**Risk acknowledgement:** v0.1.0 is new (~3 weeks old). Mitigations: an engine wrapper
(`internal/speedtest`) isolates the library behind a small Meridian-owned API so a swap
is local; tests exercise the real client over a loopback `Listen` server (the library's
own example pattern) so our integration is verified end-to-end offline; the app wraps
`RunClient` in a recover-guarded goroutine so a library panic can't crash Meridian.

## File structure

- `internal/speedtest/speedtest.go` + `speedtest_test.go` — the engine wrapper:
  - `Options{ Host string; Port int; Direction string ("download"|"upload"|"bidir"); Parallel int; Duration time.Duration; UDP bool }`
  - `Sample{ Elapsed float64; DownMbps, UpMbps float64; Phase string }`
  - `Result{ DownMbps, UpMbps float64; Bytes float64; Retransmits int64; RTTMeanMs float64; JitterMs float64; Lost, Packets int64; Peer, Protocol string }`
  - `Run(ctx context.Context, o Options, onSample func(Sample)) (Result, error)` — maps
    Options → `ClientConfig`, forwards `OnUpdate`→`onSample` (Mbps = bits/1e6), returns
    `Result` from the final `State`, maps `State.Error` to a friendly Go error.
  - Tests use `iperf3.Listen("127.0.0.1:0")` + `Run` over loopback: assert samples arrive
    and `Result.*Mbps > 0` for download, upload, and bidir; assert a connection-refused
    host yields a clear error; assert ctx-cancel stops promptly.
- `internal/app/speedtest_methods.go` + test — `SpeedtestStart(streamID, host string, port int, direction string, parallel, durationSec int, udp bool) error` and `SpeedtestStop(streamID)`:
  - validates host; registers a `monitor` (ctx cancel + done); runs `speedtest.Run` in a
    recover-guarded goroutine; on each `Sample` emits `metrics:<streamID>` with a
    `SpeedSnapshot`; on completion emits a final snapshot (`State:"done"` + the Result),
    or `State:"error"`/`"unreachable"` with a `Note`.
  - `SpeedSnapshot{ Elapsed, DownMbps, UpMbps float64; State string; Note string; Peer string; plus the Result fields on done }`.
  - `OnShutdown`'s `stopAllMonitors()` already stops a running test on app close.
- `frontend/src/lib/tools/SpeedTest.svelte` — host + port(5201) + direction select +
  parallel(4) + duration(10s) + TCP/UDP + Start/Stop; a live `Gauge` (current Mbps,
  pct against an auto-scaling max) + `StreamChart` (down/up series over elapsed) + a
  result card (avg down/up, retransmits, RTT; jitter/loss for UDP). Subscribes to
  `metrics:<streamID>` via `metricStream.ts` BEFORE calling Start (so no sample is
  missed), like the ping monitor. A help line explains the other end needs `iperf3 -s`.
- `frontend/src/lib/Workbench.svelte` — register under **Local network**
  (`{ id:"speedtest", name:"LAN speed test", group:"Local network", ready:true }`).
- Bindings regenerated.

## Error mapping (friendly)

- connection refused / no route → "No iperf3 server answered at <host>:<port> — start `iperf3 -s` on that host."
- `ErrorBusy` → "That iperf3 server is busy running another test — try again shortly."
- bad host → "Couldn't resolve <host>."
- ctx cancelled (Stop) → no error banner; just stop.

## Review focus (inputs the happy-path tests won't cover)

- **No server listening**: a clear, non-panicking error (not a hang) within a short
  connect timeout.
- **Stop mid-test**: `SpeedtestStop` cancels promptly; the goroutine exits (`<-done`);
  no leak; no late event after stop.
- **Library panic / unexpected State**: the recover guard turns it into an error, never
  a crash.
- **Direction semantics**: download = server→client (iperf3 -R), upload = client→server;
  the gauge/labels match; bidir shows both.
- **Parallel/duration bounds**: 0 or huge values clamp (library caps at 128 / 86400);
  the UI shouldn't allow a 24-hour test by accident (cap duration in the UI, e.g. 60s).
- **streamID reuse / double Start**: `registerMonitor` rejects a duplicate id cleanly.

## Builder-check resolutions (settled calls)

1. **`onSample` fires only for `connecting`/`running`.** The library calls `OnUpdate`
   with `Phase=done` (and on failure) carrying the SAME state `RunClient` then returns.
   To avoid a double terminal event, `Run`'s forwarding rule is: call `onSample` only
   when `s.Phase` is connecting or running; the terminal outcome is surfaced **solely**
   via the returned `Result`/error. The app layer emits exactly one terminal snapshot.
2. **Error mapping + pre-resolve.** `RunClient` never returns a Go error — failures land
   in `State.Error` as a code: `busy`, `refused`, `timeout`, `unreachable`, `udp`,
   `protocol`, `interrupted`, `cancelled`, or `server:<n>`. Mapping: `refused`/`timeout`/
   `unreachable` → "No iperf3 server answered at <host>:<port> — start `iperf3 -s` there";
   `busy` → "That iperf3 server is busy — try again shortly"; `cancelled`/`interrupted`
   → no error (Stop). There is **no DNS code** (an unresolvable host classifies as
   `unreachable`), so `Run` **pre-resolves** the host (`net.DefaultResolver.LookupHost`
   with a short context) before `RunClient`; a resolve failure returns "Couldn't resolve
   <host>" directly. On success the terminal `State` sets BOTH `UploadResult` and
   `DownloadResult` for every direction (download-only leaves a ~0 UploadResult), so build
   `Result` only after checking `State.Error == ""` — no nil-deref.
3. **Stop emits nothing.** On Stop the ctx is cancelled → `State.Error="cancelled"`.
   The app goroutine, after `Run` returns, checks `ctx.Err() != nil` (mirroring
   PingMonitor's `if ctx.Err() != nil { return }`) and emits no terminal event.
4. **Loopback tests, whole-second durations.** `Duration` rounds **up to whole seconds**
   (no sub-second test); a loopback `Listen` + `Run` test runs ~1s per direction. The
   library's connect path already has a ~6s timeout and refused is immediate, so no extra
   ctx timeout is needed. Loopback `Listen` is test-only (127.0.0.1), never in app code —
   the "never serve" rule holds.
5. **Gauge auto-scales to the running peak** so a 50 Mbit/s link and a 10 Gbit/s link both
   read well; the result card shows exact numbers.
