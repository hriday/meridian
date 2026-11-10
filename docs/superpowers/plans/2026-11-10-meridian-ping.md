# Enhanced Ping — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ping one host N times and report min/avg/max/stddev/jitter latency and packet loss, with per-ping streaming, across engine, CLI, and the Workbench.

**Architecture:** New engine `internal/ping`: a pure `summarize` computes the stats over a slice of replies; `Run` drives the existing `scan.Pinger` (reused unprivileged ICMP prober), streaming each reply. Thin adapters: a Wails method, a `ping` CLI command, and a `Ping.svelte` tool under "Local network". Mirrors the established slice structure.

**Tech Stack:** Go 1.23 (`net/netip`, `math`), reusing `internal/scan` (`Pinger`, `NewICMPPinger`), Wails v2 + Svelte. Tests use a fake `scan.Pinger` and `stretchr/testify`.

**Spec:** `docs/superpowers/specs/2026-11-10-meridian-ping-design.md`

## Global Constraints

- Reuse `scan.Pinger` / `scan.NewICMPPinger`; no new privilege. ICMP unavailability is reported plainly, never as 100% loss.
- Stats over the *received* replies: min/avg/max/stddev (population) + jitter (mean |rttᵢ − rttᵢ₋₁| over consecutive replies); loss% = (sent−received)/sent×100.
- Intervals stay humane (default 500 ms); bounded `count` (default 5); per-ping timeout default 1 s.
- Engine resolves the host to an IP behind a seam so tests stay offline.
- Commit attribution: `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`. PR body ends with the Generated-with line.

## Review Focus

- **All-loss ≠ divide-by-zero.** With zero received replies, loss is 100% and min/avg/max/stddev/jitter are 0 (or clearly "n/a"), no NaN/panic. (Task 1 test.)
- **Single reply.** One received reply → stddev 0 and jitter 0 (no consecutive pair), min==avg==max==that RTT. (Task 1 test.)
- **ICMP unavailable** → `Available=false`, reported distinctly from loss; `Run` doesn't spin `count` times pointlessly claiming loss. (Task 2 test with a fake pinger returning available=false.)
- **Context cancellation** stops the run early without panicking, returning the partial stats gathered. (Task 2 test.)
- **Jitter is consecutive, not range.** RTTs `[10, 40, 10]` → jitter = mean(|40−10|,|10−40|) = 30, not max−min. (Task 1 test pins the exact value.)

---

### Task 1: Pure stats (`summarize`) + types

**Files:**
- Create: `internal/ping/ping.go`
- Test: `internal/ping/ping_test.go`

**Interfaces:**
- Produces:
  - `type Reply struct { Seq int; RTTMillis float64; OK bool }`
  - `type Stats struct { Host string; IP string; Sent int; Received int; LossPct float64; MinMs float64; AvgMs float64; MaxMs float64; StddevMs float64; JitterMs float64; Available bool; Replies []Reply }`
  - `func summarize(host, ip string, replies []Reply, available bool) Stats` — Sent = len(replies); Received = count of OK; LossPct from those; min/avg/max/stddev over OK RTTs; jitter = mean abs diff of consecutive OK RTTs; carries Replies/Available through.

- [ ] **Step 1: Write failing tests** `internal/ping/ping_test.go` (`package ping`):
  - `TestSummarizeBasic`: replies with OK RTTs `[10,20,30]` → Sent 3, Received 3, Loss 0, Min 10, Avg 20, Max 30, Stddev ≈ 8.165 (population), and assert to a tolerance (`require.InDelta`).
  - `TestSummarizeJitterConsecutive`: OK RTTs `[10,40,10]` → JitterMs == 30 (mean of |30|,|30|), not 30-from-range coincidence — add a `[10,40,20]` case → mean(|30|,|20|) = 25 to disambiguate.
  - `TestSummarizeLoss`: 4 replies, 1 OK (rtt 10), 3 not OK → Sent 4, Received 1, Loss 75, Min==Avg==Max==10, Stddev 0, Jitter 0.
  - `TestSummarizeAllLoss`: 3 replies all not OK → Received 0, Loss 100, Min/Avg/Max/Stddev/Jitter all 0, no NaN (`require.False(math.IsNaN(...))`).
  - `TestSummarizeSingle`: 1 OK reply rtt 15 → Stddev 0, Jitter 0, Min==Avg==Max==15.
- [ ] **Step 2: Run, verify fail** (`go test ./internal/ping/ -run Summarize -v`).
- [ ] **Step 3: Implement** `summarize` (+ the types). Guard the zero-received and single-reply cases so no division by zero and no NaN.
- [ ] **Step 4: Run, verify pass.**
- [ ] **Step 5: Commit** (`feat(ping): latency statistics (min/avg/max/stddev/jitter/loss)`).

### Task 2: `Run` — drive the pinger, stream replies

**Files:**
- Modify: `internal/ping/ping.go`
- Test: `internal/ping/run_test.go`

**Interfaces:**
- Consumes: `scan.Pinger` (`Ping(ctx, netip.Addr, timeout) (time.Duration, bool, bool)`), Task 1 `summarize`.
- Produces:
  - `var resolveIP = func(ctx context.Context, host string) (netip.Addr, error)` — package-var seam; default parses an IP literal or resolves via `net.DefaultResolver.LookupNetIP` (first address).
  - `func Run(ctx context.Context, p scan.Pinger, host string, count int, interval, timeout time.Duration, onReply func(Reply)) (Stats, error)` — resolves host→IP; if the first ping reports ICMP unavailable, returns `summarize(host, ip, nil, false)` (Available=false) without looping; otherwise pings `count` times `interval` apart (respecting ctx between pings), building `[]Reply`, calling `onReply` per reply; returns `summarize(host, ip.String(), replies, true)`. Cancellation returns the partial stats gathered (nil error on ctx.Done, or ctx.Err — pick one and test it).

- [ ] **Step 1: Write failing tests** `internal/ping/run_test.go` using a fake pinger:
  - a `fakePinger` whose `Ping` returns scripted `(rtt, ok, available)` per call (from a slice, or a func).
  - `TestRunAggregates`: fake returns 5 OK pings with varied RTTs; `Run(ctx, fake, "127.0.0.1", 5, 0, time.Second, nil)` → Stats.Sent 5, Received 5, IP "127.0.0.1", and stats match `summarize`.
  - `TestRunStreamsReplies`: collect `onReply` calls; assert one per ping, in sequence order.
  - `TestRunICMPUnavailable`: fake returns `available=false` on the first ping; assert `Stats.Available==false`, `Sent==0` (didn't loop), no panic.
  - `TestRunCancel`: a ctx cancelled after the 2nd ping (fake cancels via its func, or cancel before call with interval>0) → Run returns partial Stats (Received ≤ count), no panic.
- [ ] **Step 2: Run, verify fail.**
- [ ] **Step 3: Implement** `Run` + `resolveIP`. Use a `select` on `ctx.Done()` and a timer for the interval; parse IP literals before DNS.
- [ ] **Step 4: Run, verify pass**, then full `go test ./internal/ping/`.
- [ ] **Step 5: Commit** (`feat(ping): Run drives the pinger and streams replies`).

### Task 3: Wails method + CLI `ping`

**Files:**
- Create: `internal/app/ping_methods.go`, `internal/app/ping_methods_test.go`
- Modify: `internal/cli/cli.go`
- Create: `internal/cli/printping_test.go` (`package cli`, internal)

**Interfaces:**
- Produces:
  - `func (a *App) PingHost(host string, count int) (ping.Stats, error)` — builds `scan.NewICMPPinger()`, calls `ping.Run` with interval 500 ms, per-ping timeout 1 s, `count` (default 5 when ≤0), no streaming callback (synchronous; returns full Stats).
  - CLI: `case "ping": return runPing(...)`; `runPing` (flags `--count` 5, `--interval` 500ms, `--timeout` 1s, `--json`, one positional host; streams each reply via `onReply` printing `seq · rtt`/`timeout`); `printPingStats(w, Stats)` printing the summary (sent/received/loss, min/avg/max/stddev/jitter) and an ICMP-unavailable note when applicable.

- [ ] **Step 1 (app): Write failing test** `ping_methods_test.go`: override `ping`'s `resolveIP`? No — instead inject a fake pinger. Simplest: `PingHost` builds a real ICMPPinger, which may be unavailable in CI; to keep the test deterministic, **add a package-var seam `newPinger = scan.NewICMPPinger` in `ping_methods.go`** and have the test swap it for a fake that returns scripted OK pings, then assert `PingHost("127.0.0.1", 3)` returns Stats with Received 3. (Resolve of "127.0.0.1" is a literal, no DNS.)
- [ ] **Step 2: Run, verify fail.**
- [ ] **Step 3: Implement** `PingHost` (+ the `newPinger` seam).
- [ ] **Step 4: Run, verify pass.**
- [ ] **Step 5 (cli): Write failing tests** `printping_test.go`:
  - `TestPrintPingStats`: a `ping.Stats` (received<sent, known min/avg/max/jitter) → output contains sent/received, the loss %, and the min/avg/max/jitter figures.
  - `TestPrintPingStatsUnavailable`: Stats with `Available=false` → output explains ICMP isn't available, not "100% loss".
  - `TestRunPingMissingArg`: `runPing(nil,…)` → exit 2.
- [ ] **Step 6: Run, verify fail.**
- [ ] **Step 7: Implement** the `ping` case, `runPing`, `printPingStats`, usage line; import `ping` and `scan`.
- [ ] **Step 8: Run** `go test ./internal/app/ ./internal/cli/`, verify pass.
- [ ] **Step 9: Commit** (`feat(cli,app): ping command and Wails method`).

### Task 4: Workbench tool `Ping.svelte`

**Files:**
- Create: `frontend/src/lib/tools/Ping.svelte`
- Modify: `frontend/src/lib/Workbench.svelte` (import, tools entry under "Local network", route)
- Regenerate: bindings (`App.js`/`App.d.ts`/`models.ts` — `PingHost` + `ping.Stats`).

- [ ] **Step 1: Regenerate bindings** (`$(go env GOPATH)/bin/wails generate module`; `git checkout -- frontend/wailsjs/runtime`); verify `PingHost` in `App.d.ts` and a `ping` namespace in `models.ts`.
- [ ] **Step 2: Write `Ping.svelte`** — host input, count (default 5), Ping button; a self-explaining lede; on result a per-ping list (seq · RTT / timeout) with a compact RTT sparkline (inline SVG bars), a stats row (sent/received, loss %, min/avg/max/stddev/jitter) and a one-line plain-language verdict; an ICMP-unavailable note when `!available`. (Synchronous call — show a "Pinging…" state while `PingHost` runs.) Reuse the established tool visual language and theme tokens.
- [ ] **Step 3: Wire into `Workbench.svelte`**: import, add `{ id: "ping1", name: "Ping", group: "Local network", ready: true }` before "Ping sweep", add the `{:else if active === "ping1"}<Ping />` route. (Use a distinct id from the existing `ping` sweep id.)
- [ ] **Step 4: Build** `cd frontend && npm run build`; verify green.
- [ ] **Step 5: Manual check** via the running app (localhost:34115): ping a reachable host (e.g. 1.1.1.1), confirm the per-ping list, sparkline, stats, and verdict render and read clearly.
- [ ] **Step 6: Commit** (`feat(app,ui): enhanced ping tool in the Workbench`).

---

## Self-Review notes

- Spec coverage: §4.6 enhanced ping → Tasks 1–4; stats → Task 1; run/stream/availability → Task 2; CLI/app → Task 3; UI sparkline+verdict → Task 4.
- Review Focus → each maps to a named test (all-loss, single, unavailable, cancel, jitter-consecutive).
- Proportion: signatures + test names; bodies left to the implementer. The engine reuses `scan.Pinger`, so no new ICMP code.
