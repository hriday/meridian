# Meridian NTP & DNS Tools Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Two "Local / environment" tools — **NTP** (query any time server for stratum, offset, and delay; show the machine's configured time server) and **DNS** (query any resolver for any record type with DNSSEC flags; show the machine's configured resolvers).

**Architecture:** Pure engines `internal/ntp` (a from-scratch SNTP client over `net`) and `internal/dnsx` (a dig-like client over `github.com/miekg/dns`), plus `internal/sysconf` which best-effort-detects the OS's configured time server and DNS resolvers via small OS commands with pure, tested parsers. Wails controller + CLI are thin adapters; two Svelte tools render the results. Every engine isolates a pure decode/summarize function (unit-tested with canned bytes/messages) from the thin network send.

**Tech Stack:** Go 1.23; stdlib `net`/`encoding/binary` for NTP; **new dependency `github.com/miekg/dns`** for DNS; Wails v2.10.1 + Svelte/TS.

**Spec:** docs/superpowers/specs/2026-11-08-meridian-design.md (§4.6 tool list: "DNS & Whois", time/NTP). Scope + layout approved by owner 2026-11-09 against the clickable mockup: two tools under a **Local / environment** sidebar group; each shows a "This machine" card (best-effort, graceful fallback) plus a query form and result cards.

## Global Constraints

- Go **1.23**. One new runtime dependency only: **`github.com/miekg/dns`** (latest stable, pin the exact version `go get` resolves). No others.
- Network egress happens only on a user action (a query), to the server the user chose or the system default. NTP = UDP/123, DNS = UDP/53 (TCP fallback via miekg/dns).
- Per-query timeout **5 s**, applied via context/deadline; a timeout or unreachable server returns an error that the adapter surfaces — never a hang.
- **"This machine" detection is best-effort.** Each detector returns `(values, sourceLabel, err)`; on any failure the tools show "couldn't detect — enter one manually", never a crash or blank. Detection shells out to OS tools (`scutil`, `systemsetup`, `timedatectl`, `w32tm`, `ipconfig`) or reads `/etc/resolv.conf`; the **parsers are pure and tested**, the exec is thin and GOOS-gated at runtime.
- First-class IPv4 **and** IPv6 (DNS servers given as IPv6, AAAA records).
- UI stays non-blocking (async + busy state); themes (Auto/Light/Dark/Terminal), Geist fonts, collapsible sidebar intact. New tools join the existing sidebar group machinery.

## Review Focus

- **NTP server unreachable / timeout** (bad host, blocked UDP/123): the query returns an error and the UI shows it; nothing hangs. *Test in Task 1 (thin Query path via a closed port / deadline).*
- **Malformed or kiss-o'-death NTP reply** (<48 bytes, stratum 0 / LI=3): decode returns an error or a flagged result, never a panic or garbage offset. *Test in Task 1.*
- **DNS NXDOMAIN / SERVFAIL / empty answer**: surfaced as the rcode with an empty answer set and timing still shown — not an error dialog. *Test in Task 2.*
- **System config not detectable** (no `/etc/resolv.conf`, sudo-gated `systemsetup`, locked-down host): detectors return a clean "not detected" with no crash. *Test in Task 3 (parsers on empty/garbage input).*
- **IPv6 DNS server + unusual record types** (query `@2606:4700:4700::1111`, types `CAA`/`SOA`/`TXT` with quotes/`AAAA`): the client targets the v6 server and renders each record's text form correctly. *Test in Task 2.*

---

## Task 1: NTP SNTP client (`internal/ntp`)

**Files:**
- Create: `internal/ntp/ntp.go`
- Test: `internal/ntp/ntp_test.go`

**Interfaces:**
- Produces: `type Reply struct { Server string; Stratum int; RefID string; LeapIndicator string; Precision float64; RootDelay, RootDispersion time.Duration; ServerTime time.Time; Offset, RTT time.Duration; KissOfDeath bool }`; `func decode(resp []byte, t1, t4 time.Time) (Reply, error)` (pure: parses a 48-byte NTP v4 reply, computes offset/RTT from the four timestamps); `func Query(ctx context.Context, server string) (Reply, error)` (dials `udp` to `server:123` — add default port if absent — sends a mode-3 client packet, reads the reply with the ctx deadline, calls `decode`).
- Consumes: nothing.

- [ ] **Step 1: Write the failing test** `TestDecodeStratum1` — build a 48-byte NTP reply (LI=0, version=4, mode=4, stratum=1, refID "GPS", plausible transmit/receive timestamps) and assert `decode` returns `Stratum==1`, `RefID=="GPS"`, `LeapIndicator=="none"`, and a finite `Offset`/`RTT`.

```go
func TestDecodeStratum1(t *testing.T) {
    t1 := time.Unix(1762700000, 0).UTC()
    t4 := t1.Add(18 * time.Millisecond)
    resp := buildNTP(t, 0 /*li*/, 1 /*stratum*/, "GPS", t1.Add(8*time.Millisecond))
    r, err := decode(resp, t1, t4)
    require.NoError(t, err)
    require.Equal(t, 1, r.Stratum)
    require.Equal(t, "GPS", r.RefID)
    require.Equal(t, "none", r.LeapIndicator)
}
```

- [ ] **Step 2: Add** `TestDecodeShortPacketErrors` (a 20-byte slice → error, no panic) and `TestDecodeKissOfDeath` (stratum 0 → `KissOfDeath==true`, no bogus offset).
- [ ] **Step 3: Run the tests, verify they fail.** Run: `go test ./internal/ntp/ -run TestDecode -v` — Expected: FAIL, `decode`/`buildNTP` undefined.
- [ ] **Step 4: Implement** `Reply`, `decode` (NTP timestamp = seconds since 1900; convert to `time.Time`; offset = ((t2−t1)+(t3−t4))/2, RTT = (t4−t1)−(t3−t2); refID text per stratum: ASCII for stratum ≤1, IPv4 for ≥2), and the `buildNTP` test helper, then `Query`.
- [ ] **Step 5: Run the tests, verify they pass.** Expected: PASS.
- [ ] **Step 6: Commit** (`feat(ntp): SNTP client with stratum/offset/delay decode`).

## Task 2: DNS query client (`internal/dnsx`, miekg/dns)

**Files:**
- Create: `internal/dnsx/dnsx.go`
- Modify: `go.mod`, `go.sum` (add `github.com/miekg/dns`)
- Test: `internal/dnsx/dnsx_test.go`

**Interfaces:**
- Consumes: `github.com/miekg/dns` (imported as `mdns`).
- Produces: `type Record struct { Name, Type string; TTL uint32; Value string }`; `type Result struct { Server, Rcode string; Flags []string; Authenticated bool; RTTMillis int; Answers []Record }`; `func summarize(msg *mdns.Msg, server string, rtt time.Duration) Result` (pure: maps rcode, qr/rd/ra/ad flags, and each RR's text form into `Record`s); `func Query(ctx context.Context, name, rtype, server string) (Result, error)` (builds the question for `rtype` (A/AAAA/MX/NS/TXT/CNAME/SOA/PTR/CAA/ANY), sets EDNS0 + DNSSEC OK, exchanges with `server` (default `:53`, IPv6 bracketed), returns `summarize`).

- [ ] **Step 1:** `go get github.com/miekg/dns` and confirm it is the only module added (`go mod tidy`).
- [ ] **Step 2: Write the failing test** `TestSummarizeAAndFlags` — construct an `*mdns.Msg` with Rcode=NOERROR, flags qr/rd/ra/ad, and two A RRs → `summarize` returns `Rcode=="NOERROR"`, `Authenticated==true`, `len(Answers)==2`, and the records' `Type=="A"` with the IP as `Value`.
- [ ] **Step 3: Add** `TestSummarizeNXDOMAIN` (Rcode=NXDOMAIN, no answers → `Rcode=="NXDOMAIN"`, empty `Answers`, no error) and `TestSummarizeRecordTypes` (one each of MX, TXT (quoted), SOA, CAA, AAAA → each `Value` is the expected text form).
- [ ] **Step 4: Run the tests, verify they fail.** Expected: FAIL, `summarize`/types undefined.
- [ ] **Step 5: Implement** `Record`, `Result`, `summarize`, and `Query`. For `Query`, add an **integration test** `TestQueryAgainstLocalServer` that starts a `mdns.Server` on `127.0.0.1:0` (UDP) answering a canned A record, then asserts `Query` returns it — watch it fail first, then make it pass.
- [ ] **Step 6: Run** `go test ./internal/dnsx/` — verify all pass.
- [ ] **Step 7: Commit** (`feat(dnsx): dig-like DNS client over miekg/dns`).

## Task 3: System config detection (`internal/sysconf`)

**Files:**
- Create: `internal/sysconf/sysconf.go`
- Test: `internal/sysconf/sysconf_test.go`

**Interfaces:**
- Produces: `type Found struct { Values []string; Source string; Err string }`; `func Resolvers() Found`; `func TimeServers() Found`; and the pure parsers `parseResolvConf(string) []string`, `parseScutilDNS(string) []string`, `parseSystemsetupNTP(string) string`, `parseTimedatectl(string) string`, `parseW32tmSource(string) string`. `Resolvers`/`TimeServers` pick the detector by `runtime.GOOS`, shell out with a short timeout, run the parser, and always return a `Found` (empty `Values` + a `Source`/`Err` when nothing is detectable — never panic).
- Consumes: nothing (used by adapters in Task 4).

- [ ] **Step 1: Write the failing test** `TestParseResolvConf` — feed a sample `/etc/resolv.conf` (comments, two `nameserver` lines incl. an IPv6, a `search` line) → `parseResolvConf` returns exactly the two nameserver IPs in order.
- [ ] **Step 2: Add** `TestParseScutilDNS` (sample `scutil --dns` text → the `nameserver[n]` IPs, de-duplicated), `TestParseSystemsetupNTP` (`Network Time Server: time.apple.com` → `time.apple.com`), `TestParseTimedatectl`/`TestParseW32tm` (sample outputs → server), and `TestParsersOnGarbage` (empty string and unrelated text → empty slice / "" for every parser, no panic).
- [ ] **Step 3: Run the tests, verify they fail.** Expected: FAIL, parsers undefined.
- [ ] **Step 4: Implement** the pure parsers and the GOOS-gated `Resolvers`/`TimeServers` (exec with a 2 s timeout; set `Source` to the tool used; set `Err` and leave `Values` empty when the tool is missing, errors, or parses to nothing).
- [ ] **Step 5: Run** `go test ./internal/sysconf/` — verify the parser tests pass. (The live `Resolvers()`/`TimeServers()` are environment-dependent and not asserted on specific values.)
- [ ] **Step 6: Commit** (`feat(sysconf): best-effort OS time-server and resolver detection`).

## Task 4: CLI commands + bound app methods

**Files:**
- Modify: `internal/cli/cli.go` (add `ntp` and `dns` commands + usage)
- Create: `internal/app/env_methods.go` (bound `NTPQuery`, `NTPSystem`, `DNSQuery`, `DNSSystem`)
- Test: `internal/cli/cli_env_test.go`, `internal/app/env_methods_test.go`

**Interfaces:**
- Consumes: `ntp.Query`, `dnsx.Query`, `sysconf.Resolvers`/`TimeServers`.
- Produces: CLI `meridian-cli ntp <server> [--json]` and `meridian-cli dns <name> [--type A] [--server <ip>] [--json]`; App methods `func (*App) NTPQuery(server string) (ntp.Reply, error)`, `func (*App) DNSQuery(name, rtype, server string) (dnsx.Result, error)`, `func (*App) NTPSystem() sysconf.Found`, `func (*App) DNSSystem() sysconf.Found`. The query functions are reachable behind package vars so tests inject fakes (as `newLookupClient` does today).

- [ ] **Step 1: Write the failing test** `TestRunDNSJSON` (cli): override the injected dns query func with a fake returning a known `Result` → `cli.Run(["dns","example.com","--type","A","--json"], …)` exits 0 and the JSON contains the fake record.
- [ ] **Step 2: Add** `TestRunNTPJSON` (fake ntp query → JSON contains the stratum) and an app-method test `TestAppDNSQuery` (fake → populated `Result`).
- [ ] **Step 3: Run the tests, verify they fail.** Expected: FAIL.
- [ ] **Step 4: Implement** the commands (flag parsing via the existing `parsePositionals`; text + `--json` output) and the bound methods.
- [ ] **Step 5: Run** `go test ./...` — verify all pass.
- [ ] **Step 6: Commit** (`feat(cli,app): ntp and dns commands + bound methods`).

## Task 5: Frontend — NTP and DNS tools

**Files:**
- Create: `frontend/src/lib/tools/Ntp.svelte`, `frontend/src/lib/tools/Dns.svelte`
- Modify: `frontend/src/lib/Workbench.svelte` (add a "Local / environment" group with the two ready tools; render them)
- (Wails regenerates `frontend/wailsjs/go/**`.)

**Interfaces:**
- Consumes: bound `NTPQuery`/`NTPSystem`/`DNSQuery`/`DNSSystem` and the `ntp.Reply` / `dnsx.Result` / `sysconf.Found` models.

- [ ] **Step 1:** `Ntp.svelte` matching the mockup: a "This machine" card from `NTPSystem()` (with a "couldn't detect" fallback), a server input (+ "Use system server"), and result cards — the stratum chain, stratum/refID/leap/precision/server-time, and the timing row (offset/RTT/root delay/root dispersion). Busy + error states.
- [ ] **Step 2:** `Dns.svelte` matching the mockup: a "This machine" card from `DNSSystem()`, a query row (name + record-type select + server defaulting to system), a flags/response card, and the answer table. Busy + error states; NXDOMAIN/SERVFAIL shown as the rcode, not an error box.
- [ ] **Step 3:** In `Workbench.svelte`, add the `Local / environment` group with `ntp` and `dns` (ready: true) and render `Ntp`/`Dns` for those ids.
- [ ] **Step 4: Build** (`~/go/bin/wails build`) and launch; confirm a real NTP query (e.g. `time.apple.com`) and DNS query render (manual visual check — UI has no unit harness, per the project pattern).
- [ ] **Step 5: Run** `go test ./...` to confirm the engine suite is green.
- [ ] **Step 6: Commit** (`feat(ui): NTP and DNS tools`).

---

## Self-review notes
- Spec coverage: NTP engine = Task 1; DNS engine = Task 2; "this machine" detection = Task 3; CLI/app parity = Task 4; UI = Task 5.
- Type consistency: `ntp.Reply`, `dnsx.Result`/`Record`, `sysconf.Found` are defined in Tasks 1–3 and consumed verbatim by Tasks 4–5.
- Review Focus: unreachable/timeout + malformed NTP (Task 1), NXDOMAIN/SERVFAIL + IPv6/record-types (Task 2), undetectable config (Task 3) each have named tests.
- Proportion: signatures + test names + the NTP math and the one new dependency are specified; implementations are left to the engineer.
