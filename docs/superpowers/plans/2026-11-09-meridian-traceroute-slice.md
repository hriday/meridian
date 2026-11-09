# Meridian AS-aware Traceroute Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Trace the path to a host and label each hop with its network (AS number + owner), highlighting AS boundaries and the networks crossed — with a plain-language verdict — using unprivileged ICMP so no admin rights are needed.

**Architecture:** A pure `internal/trace` engine whose probing sits behind a `Prober` interface (so the TTL/aggregation loop is tested with a fake, while the real ICMP prober over `golang.org/x/net/icmp` is a thin, GOOS-agnostic datagram implementation). Hops stream out via a callback. `internal/asnmap` annotates hop IPs with AS/owner via Team Cymru's IP-to-ASN DNS service (reusing `internal/dnsx`). The Wails controller streams hops to the frontend as events and the CLI prints them; the Svelte tool renders the live hop table, the AS-path strip, and the verdict.

**Tech Stack:** Go 1.23; **`golang.org/x/net`** (icmp, ipv4, ipv6 — already in the module graph via miekg/dns, promoted to direct); `internal/dnsx` for the Cymru lookups; Wails v2.10.1 events for streaming; Svelte/TS.

**Spec:** docs/superpowers/specs/2026-11-08-meridian-design.md (§4.2 AS-aware traceroute, §3 decision 49). Scope + layout approved by owner 2026-11-09 against the clickable mockup: single run (not MTR-live), AS annotation via Cymru, AS-path strip + boundary highlighting + plain-language verdict; MTR-live, the BGP-path comparison, UDP/TCP modes, and RIPE Atlas remote vantage are deferred ("soon").

## Global Constraints

- Go **1.23**. New direct dependency: **`golang.org/x/net`** (already present indirectly; promote, pin the resolved version). No others.
- **Unprivileged only** — ICMP Echo with a per-probe TTL over datagram sockets (`icmp.ListenPacket("udp4"/"udp6", …)`), never raw sockets or root. If the OS won't grant an ICMP datagram socket (e.g. Linux `net.ipv4.ping_group_range` locked down), surface a clear one-line message, never a crash or silent empty result.
- Defaults: **max 30 hops**, **3 probes/hop**, **1.5 s** per-probe timeout. The whole trace is cancelable.
- Network egress only on a user action. First-class IPv4 **and** IPv6 (ICMP + ICMPv6).
- Hops **stream** to the UI as they resolve; the UI never blocks (a 30-hop × 3-probe × 1.5 s trace can take tens of seconds). Cancel stops it.
- Plain-language **verdict** line + a short caption on protocol fields, per the project standard ([[ui-principles-nielsen]] — matches system to the real world).
- Themes (Auto/Light/Dark/Terminal), Geist fonts, collapsible sidebar intact.

## Review Focus

- **Target never reached** (all `*`, or destination silent past 30 hops): the trace ends cleanly with a "did not reach" verdict — no infinite loop or hang. *Test in Task 2 (fake prober that never reaches).*
- **ICMP socket unavailable** (permission/ping_group_range): construction returns an error the CLI/GUI shows plainly. *Test in Task 5 via an injected prober factory that errors.*
- **Unresolvable or wrong-family target** (bad hostname; A-only name with IPv6 requested): clean error, not a panic. *Test in Task 2 (resolve seam).*
- **Private / bogon / unannounced mid-path hops**: AS annotation skips them (no bogus AS), and boundary detection treats "no AS" correctly. *Test in Task 1 (boundaries) + Task 4 (skip private).*
- **Cymru lookup fails/times out for a hop**: that hop keeps its IP + RTT with no AS; the rest of the trace and the AS-path strip are unaffected. *Test in Task 4.*

---

## Task 1: Trace types + pure aggregation (`internal/trace`)

**Files:**
- Create: `internal/trace/types.go`
- Test: `internal/trace/types_test.go`

**Interfaces:**
- Produces: `type Hop struct { TTL int; Addr string; Host string; RTTs []time.Duration; ASN, ASOwner string; Timeout bool }`; `type Result struct { Target, Addr string; Hops []Hop; Reached bool; Elapsed time.Duration }`; `func rttStats(d []time.Duration) (min, avg, max time.Duration)` (ignoring zero/timeout probes); `func asPath(hops []Hop) []Hop` (the ordered distinct non-empty ASNs, as {ASN,ASOwner}); `func isBoundary(hops []Hop, i int) bool` (hop i starts a new AS vs the previous annotated hop); `func Verdict(r Result) string` (plain-language summary, incl. a "did not reach" form).
- Consumes: nothing.

- [ ] **Step 1: Write the failing test** `TestRTTStatsIgnoresTimeouts` (a mix of real + zero durations → min/avg/max over the real ones) and `TestASPathDistinctInOrder` (hops with ASNs 7922,7922,6939,—,13335 → [7922,6939,13335]).
- [ ] **Step 2: Add** `TestIsBoundary` (boundary true where ASN changes, false within the same AS, handling a `—`/empty hop between) and `TestVerdictReachedAndNotReached` (a reached Result → "N hops to <addr> … crossing K networks …"; an unreached Result → "did not reach …").
- [ ] **Step 3: Run the tests, verify they fail.** Run: `go test ./internal/trace/ -run 'TestRTT|TestASPath|TestIsBoundary|TestVerdict' -v` — Expected: FAIL, undefined.
- [ ] **Step 4: Implement** the types and the four pure helpers.
- [ ] **Step 5: Run the tests, verify they pass.**
- [ ] **Step 6: Commit** (`feat(trace): hop/result types and pure aggregation`).

## Task 2: Trace loop over a Prober seam

**Files:**
- Create: `internal/trace/trace.go`
- Test: `internal/trace/trace_test.go`

**Interfaces:**
- Consumes: Task 1 types.
- Produces: `type Prober interface { Probe(ctx context.Context, dst netip.Addr, ttl, seq int, timeout time.Duration) (from netip.Addr, rtt time.Duration, reached bool, err error); Close() error }`; `type Options struct { MaxHops, Probes int; Timeout time.Duration }`; `func Run(ctx context.Context, dst netip.Addr, p Prober, opts Options, onHop func(Hop)) (Result, error)` (TTL 1..MaxHops; `Probes` probes per hop; a probe with an invalid `from` becomes a `*`; emit each Hop via `onHop`; stop after the hop whose probe `reached` the destination; set `Result.Reached`).

- [ ] **Step 1: Write the failing test** `TestRunReachesDestination` — a fake Prober scripted with a 3-hop path (hop 3 = dst, reached=true) → `Run` returns `Reached==true`, 3 hops, and `onHop` fired 3 times.
- [ ] **Step 2: Add** `TestRunNeverReaches` (fake prober always returns an invalid `from` / not reached → stops at MaxHops, `Reached==false`, every hop `Timeout`), `TestRunMidPathTimeouts` (one hop all-`*`, trace continues), and `TestRunCancel` (ctx canceled mid-trace → returns promptly, no panic).
- [ ] **Step 3: Run the tests, verify they fail.** Expected: FAIL, `Run`/`Prober` undefined.
- [ ] **Step 4: Implement** `Run`, `Prober`, `Options` (default MaxHops 30, Probes 3, Timeout 1.5s when zero).
- [ ] **Step 5: Run the tests, verify they pass.**
- [ ] **Step 6: Commit** (`feat(trace): TTL probe loop over a Prober seam`).

## Task 3: Real ICMP prober (`golang.org/x/net/icmp`)

**Files:**
- Create: `internal/trace/icmp.go`
- Test: `internal/trace/icmp_test.go`

**Interfaces:**
- Consumes: Task 2 `Prober`.
- Produces: `func NewICMPProber(network string) (Prober, error)` (network "ip4"/"ip6"; opens an unprivileged ICMP datagram socket via `icmp.ListenPacket("udp4"/"udp6", "0.0.0.0"/"::")`, returns a clear error if unavailable); the prober sets per-probe TTL (`ipv4.PacketConn.SetTTL` / `ipv6.PacketConn.SetHopLimit`), sends an Echo carrying a known ID/seq, and reads until a matching `TimeExceeded` (hop) or `EchoReply` from the destination (`reached`) or the timeout; plus `func parseICMPReply(proto int, b []byte, wantSeq int) (kind string, matched bool, err error)` (pure: classifies a reply payload as time-exceeded/echo-reply and confirms it carries our seq).

- [ ] **Step 1: Write the failing test** `TestParseICMPReplyEchoReply` and `TestParseICMPReplyTimeExceeded` — build ICMP message bytes with `icmp.Message` for an EchoReply (our seq) and a TimeExceeded (wrapping our Echo) → `parseICMPReply` returns the right `kind` and `matched==true`; a reply for a different seq → `matched==false`.
- [ ] **Step 2: Add** `TestNewICMPProberErrorIsClear` — assert that when the socket can't be opened the error mentions permission/ICMP (exercise the error path by requesting an invalid network, e.g. `NewICMPProber("nope")`).
- [ ] **Step 3: Run the tests, verify they fail.** Expected: FAIL, undefined.
- [ ] **Step 4: Implement** `NewICMPProber`, the prober's `Probe`/`Close`, and `parseICMPReply`. (The live send/recv is exercised end-to-end manually in Task 6, not unit-tested.)
- [ ] **Step 5: Run the tests, verify they pass.** Then `go test ./internal/trace/`.
- [ ] **Step 6: Commit** (`feat(trace): unprivileged ICMP prober (v4/v6)`).

## Task 4: AS annotation via Team Cymru (`internal/asnmap`)

**Files:**
- Create: `internal/asnmap/asnmap.go`
- Test: `internal/asnmap/asnmap_test.go`

**Interfaces:**
- Consumes: `internal/dnsx` (TXT queries); `net/netip` (skip non-global IPs).
- Produces: `type ASInfo struct { ASN, Owner string }`; `func parseOriginTXT(txt string) (asn, prefix string)` and `func parseASNameTXT(txt string) string` (pure parsers for Cymru's `"13335 | 104.16.0.0/13 | US | arin | …"` and `"13335 | US | arin | … | CLOUDFLARENET, US"` formats); `func Lookup(ctx context.Context, q dnsx.QueryFunc, ip string) (ASInfo, error)` (reverses the IP into `…origin.asn.cymru.com`, parses, then resolves the AS name; a non-global IP returns an empty ASInfo and no error without any query). `dnsx.QueryFunc` is a small function type so tests inject a fake.

- [ ] **Step 1: Write the failing test** `TestParseOriginTXT` and `TestParseASNameTXT` with real Cymru-format strings → extract `13335`, prefix, and `CLOUDFLARENET, US`.
- [ ] **Step 2: Add** `TestLookupSkipsPrivate` (ip `10.1.2.3` → empty ASInfo, **no query made** — inject a fake that fails the test if called) and `TestLookupPartialOnError` (origin lookup ok but AS-name query errors → ASInfo has ASN, empty Owner, no error).
- [ ] **Step 3: Run the tests, verify they fail.** Expected: FAIL, undefined.
- [ ] **Step 4: Implement** the parsers and `Lookup` (reverse IPv4/IPv6 into the Cymru name; use `netip` `IsPrivate`/loopback/link-local/multicast to short-circuit).
- [ ] **Step 5: Run the tests, verify they pass.**
- [ ] **Step 6: Commit** (`feat(asnmap): Team Cymru IP-to-ASN annotation`).

## Task 5: CLI `trace` + bound app method (streaming)

**Files:**
- Modify: `internal/cli/cli.go` (add `trace` command)
- Create: `internal/app/trace_methods.go` (bound `Traceroute`, `StopTrace`)
- Test: `internal/cli/cli_trace_test.go`, `internal/app/trace_methods_test.go`

**Interfaces:**
- Consumes: `trace.Run`, `trace.NewICMPProber`, `asnmap.Lookup`, `dnsx.Query`.
- Produces: CLI `meridian-cli trace <host> [--6] [--json]` (resolves the host, runs the trace, annotates hops, prints a table + verdict; `--json` emits the `Result`); App `func (*App) Traceroute(target string, useV6 bool) (trace.Result, error)` (emits a `trace:hop` Wails event per hop and returns the final annotated `Result`) and `func (*App) StopTrace()` (cancels the active trace). The prober factory and resolver are package vars so tests inject fakes.

- [ ] **Step 1: Write the failing test** `TestRunTraceJSON` (cli): inject a fake prober factory (3-hop reach) and a fake AS lookup → `cli.Run(["trace","1.1.1.1","--json"], …)` exits 0 and the JSON Result shows `reached:true` with 3 hops.
- [ ] **Step 2: Add** `TestRunTraceSocketErrorSurfaced` (prober factory returns an error → non-zero exit, message shown) and an app test `TestAppTraceroute` (fake factory → `Result.Reached` true, hops annotated).
- [ ] **Step 3: Run the tests, verify they fail.** Expected: FAIL.
- [ ] **Step 4: Implement** the command and bound methods (resolve host via a seam; build the prober via the factory var; run `trace.Run`; annotate each hop with `asnmap.Lookup` as it streams; CLI text table + `Verdict`; app emits events + stores the cancel for `StopTrace`).
- [ ] **Step 5: Run** `go test ./...`.
- [ ] **Step 6: Commit** (`feat(cli,app): trace command, streaming Traceroute method`).

## Task 6: Frontend — Traceroute tool

**Files:**
- Create: `frontend/src/lib/tools/Traceroute.svelte`
- Modify: `frontend/src/lib/Workbench.svelte` (activate the `trace` tool; render it)
- (Wails regenerates `frontend/wailsjs/go/**`.)

**Interfaces:**
- Consumes: bound `Traceroute`/`StopTrace`, the `EventsOn("trace:hop", …)` Wails runtime, and the `trace.Result`/`Hop` models.

- [ ] **Step 1:** Build `Traceroute.svelte` matching the mockup: target input + IPv4/IPv6 + Trace/Stop; subscribe to `trace:hop` events to append rows live; the hop table (#, host, AS+owner clickable, RTT min/avg/max, a latency sparkline), AS-boundary accent bar, `*` for timeouts; the AS-path strip (from the final/so-far hops); and the plain-language **verdict** line. BGP-path and MTR-live shown as "soon".
- [ ] **Step 2:** Wire `StopTrace` to the Stop button; handle the busy/error/"couldn't open ICMP socket" states; clean up the event listener on unmount.
- [ ] **Step 3:** In `Workbench.svelte`, set the `trace` tool `ready: true` and render `Traceroute`.
- [ ] **Step 4: Build** (`~/go/bin/wails build`) and launch; run a real trace (e.g. `cloudflare.com`) — confirm hops stream, AS labels appear, boundaries highlight (manual visual check; UI has no unit harness).
- [ ] **Step 5: Run** `go test ./...`.
- [ ] **Step 6: Commit** (`feat(ui): AS-aware traceroute tool`).

---

## Self-review notes
- Spec coverage: probing = Tasks 2–3; AS annotation = Task 4; verdict/AS-path/boundaries = Task 1; CLI/app streaming = Task 5; UI = Task 6.
- Type consistency: `trace.Hop`/`Result`, `trace.Prober`/`Options`, `asnmap.ASInfo` defined in 1–4 and consumed verbatim by 5–6.
- Review Focus: never-reached + cancel (Task 2), socket error (Task 5), unresolvable/family (Task 2), private hops (Tasks 1 & 4), Cymru failure (Task 4) each have named tests.
- Proportion: the plan fixes the Prober seam, the Cymru formats, the dependency, and defaults; implementations (incl. the live ICMP socket code) are left to the engineer.
