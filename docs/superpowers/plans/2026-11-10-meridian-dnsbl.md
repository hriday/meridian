# IP Blocklist (DNSBL) Reputation Check — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Check whether an IP is listed on public DNS blocklists, reporting per-list status, return codes, and reasons — and honestly distinguishing a refused query from a clean one.

**Architecture:** New pure engine `internal/dnsbl` builds reversed-IP query names, classifies DNSBL return codes, and aggregates a verdict; its networked `Check` reuses `internal/dnsx` to query a resolver. Thin adapters: a Wails method on `App`, a `blocklist` CLI command, and a `Blocklist.svelte` Workbench tool. Mirrors the TLS-cert slice's shape and the DNS tool's resolver-default UX.

**Tech Stack:** Go 1.23, `github.com/miekg/dns` (via `internal/dnsx`), Wails v2 + Svelte/TypeScript. Tests use a local `miekg/dns` server (as `internal/dnsx` does) and `stretchr/testify`.

**Spec:** `docs/superpowers/specs/2026-11-10-meridian-dnsbl-design.md`

## Global Constraints

- IP-only: non-IP input is rejected with a clear message; no hostname resolution in this tool.
- Query carries no user identity — only the reversed IP + zone. Generic User-Agent rule is DNS-N/A here (pure DNS).
- Return codes in `127.255.255.0/24` are **errors, never listings** (`.252` typo, `.254` public/open resolver, `.255` too-many-queries). Listings are `127.0.0.0/8` outside that band.
- Default resolver = system resolver (`sysconf.Resolvers()` first value), as the DNS tool does; UI shows the public-resolver caveat.
- Engine stays UI-free and pure where possible; network isolated behind the `dnsx.Query` seam. Default zone list is data, user-editable at the edges.
- Commit attribution: `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`. PR body ends with the Generated-with line.

## Review Focus

- **Public-resolver refusal mis-read as a result.** A `127.255.255.254` answer must surface as `blocked`/incomplete, never `clean` and never `listed`. (Task 1 `classify` + Task 2 verdict aggregation tests pin this.)
- **IPv6 reversal correctness.** A compressed IPv6 (`2001:db8::1`) must expand to all 32 reversed nibbles; a wrong expansion silently checks the wrong name and always reads "clean". (Task 1 `reverseName` test with a compressed v6 address.)
- **Partial check read as all-clear.** When some zones error/timeout, the summary must mark the check incomplete, so "not listed" on 2 of 4 reachable lists isn't shown as a full pass. (Task 2 verdict test with a mixed clean+error set.)
- **Invalid input.** A hostname or garbage string must be rejected before any query, with a message telling the user to resolve it first. (Task 1 `reverseName` error case + Task 3 CLI missing/invalid-arg test.)
- **Listed code other than .2.** A list returning `127.0.0.10` (Spamhaus PBL) must read as `listed`, not clean — classification keys on the `127/8`-minus-error-band range, not a single code. (Task 1 `classify` test.)

---

### Task 1: Reversed-name building, zone list, and return-code classification (pure)

**Files:**
- Create: `internal/dnsbl/dnsbl.go`
- Test: `internal/dnsbl/dnsbl_test.go`

**Interfaces:**
- Produces:
  - `type Zone struct { Host string; Name string; Note string }`
  - `func DefaultZones() []Zone`
  - `func reverseName(ip, zone string) (string, error)` — unexported; `("1.2.3.4","z")→"4.3.2.1.z"`, IPv6 → 32 reversed nibbles + `.z`; error on non-IP.
  - `type Status string` with consts `StatusListed="listed"`, `StatusClean="clean"`, `StatusBlocked="blocked"`, `StatusError="error"`.
  - `func classify(codes []string) (Status, string)` — maps A-record answer IPs to a status + a plain-language note. `127.255.255.0/24` → `blocked`; any other `127.0.0.0/8` → `listed`; empty → `clean`; answers present but none in `127/8` → `error`.

- [ ] **Step 1: Write failing tests** in `internal/dnsbl/dnsbl_test.go` (`package dnsbl`):
  - `TestReverseNameIPv4`: `reverseName("1.2.3.4","zen.spamhaus.org")` → `"4.3.2.1.zen.spamhaus.org"`, no error.
  - `TestReverseNameIPv6`: `reverseName("2001:db8::1","z")` → the 32 reversed nibbles of the fully-expanded address followed by `.z` (assert exact string: `"1.0.0.0...0.8.b.d.0.1.0.0.2.z"` — compute the full expansion of `2001:0db8:0000:0000:0000:0000:0000:0001`).
  - `TestReverseNameRejectsNonIP`: `reverseName("example.com","z")` returns an error.
  - `TestClassifyListed`: `classify([]string{"127.0.0.2"})` → `StatusListed`; `classify([]string{"127.0.0.10"})` → `StatusListed`.
  - `TestClassifyClean`: `classify(nil)` → `StatusClean`.
  - `TestClassifyBlocked`: `classify([]string{"127.255.255.254"})` → `StatusBlocked` with a non-empty note mentioning the resolver; also `.252` and `.255` → `StatusBlocked`.
  - `TestClassifyErrorOnOddAnswer`: `classify([]string{"10.0.0.1"})` → `StatusError`.
  - `TestDefaultZonesNonEmpty`: `DefaultZones()` returns ≥3 zones, each with non-empty `Host` and `Name`, and includes `zen.spamhaus.org`.
- [ ] **Step 2: Run tests, verify they fail** (`go test ./internal/dnsbl/ -run 'ReverseName|Classify|DefaultZones' -v`) — expect undefined-symbol failures.
- [ ] **Step 3: Implement** `dnsbl.go`: `reverseName` using `net.ParseIP` (`.To4()` for v4 reversed octets; else `.To16()` expanded to nibbles, reversed, dotted); `classify` keying on parsing each code with `net.ParseIP` and testing membership of `127.255.255.0/24` then `127.0.0.0/8`; `DefaultZones()` returning the spec's four zones; `Status` consts.
- [ ] **Step 4: Run tests, verify they pass.**
- [ ] **Step 5: Commit** (`feat(dnsbl): reversed-name, zone list, return-code classification`).

### Task 2: Networked `Check` and verdict aggregation

**Files:**
- Modify: `internal/dnsbl/dnsbl.go`
- Test: `internal/dnsbl/check_test.go`

**Interfaces:**
- Consumes: Task 1's `Zone`, `reverseName`, `classify`, `Status`; `dnsx.Query(ctx, name, rtype, server) (dnsx.Result, error)` via a package-var seam `var queryFn = dnsx.Query`.
- Produces:
  - `type Listing struct { Zone string; Name string; Status Status; Codes []string; Reason string; Note string }`
  - `type Report struct { IP string; Resolver string; Checked int; Listed int; Unreachable int; Verdict string; Summary string; Listings []Listing }`
  - `func Check(ctx context.Context, ip, resolver string, zones []Zone, timeout time.Duration) (Report, error)` — errors if `ip` is not an IP or `resolver` is empty; per zone does an A query (and a TXT query when listed) via `queryFn`, builds a `Listing`; a per-zone network error → `StatusError`; aggregates counts, verdict, and plain-language summary.

- [ ] **Step 1: Write failing tests** in `internal/dnsbl/check_test.go` (`package dnsbl`), using a local `miekg/dns` server (copy the `internal/dnsx` test's listener pattern) whose handler inspects the queried name and returns chosen A/TXT records:
  - `TestCheckListedAndClean`: server lists the reversed IP on one zone (A `127.0.0.2` + TXT "listed: see https://...") and NXDOMAINs another; point all zones at the local server; assert `Listed==1`, the listed `Listing` has `Status==StatusListed`, `Codes` contains `127.0.0.2`, `Reason` contains the TXT text, and `Verdict` starts with "Listed on 1".
  - `TestCheckAllClean`: server NXDOMAINs every name; assert `Listed==0`, `Unreachable==0`, verdict is the not-listed wording, summary mentions the count of lists checked.
  - `TestCheckBlockedIncomplete`: server returns `127.255.255.254` for one zone, NXDOMAIN for the rest; assert that zone's `Status==StatusBlocked`, `Unreachable>=1`, it is **not** counted in `Listed`, and the summary marks the check incomplete.
  - `TestCheckRejectsNonIP`: `Check(ctx,"example.com","127.0.0.1:53",DefaultZones(),2s)` returns an error.
  - `TestCheckRejectsEmptyResolver`: empty resolver → error.
  - (Inject the local server addr as each `Zone.Host` so `reverseName` builds `…​.<zonehost>`; the handler matches on the trailing labels. Alternatively set `queryFn` to a fake keyed on name — pick whichever keeps the test readable; the local-server route exercises `dnsx` end to end.)
- [ ] **Step 2: Run tests, verify they fail.**
- [ ] **Step 3: Implement** `Check` + aggregation. For each zone: `reverseName(ip, zone.Host)`; `queryFn` A; on network error → `Listing{Status:StatusError, Note: err}`; collect A answer values → `classify`; if `StatusListed`, do a TXT query for the reason (best-effort, ignore its errors). Count `Listed` (StatusListed) and `Unreachable` (StatusBlocked+StatusError). Verdict: listed → `"Listed on N blocklist(s)"`; else not-listed wording. Summary: name listed zones; if `Unreachable>0`, append that K of M lists could not be reached so the result is incomplete.
- [ ] **Step 4: Run tests, verify they pass**, then full `go test ./internal/dnsbl/`.
- [ ] **Step 5: Commit** (`feat(dnsbl): networked Check with verdict aggregation`).

### Task 3: Wails method + CLI `blocklist` command

**Files:**
- Create: `internal/app/dnsbl_methods.go`, `internal/app/dnsbl_methods_test.go`
- Modify: `internal/cli/cli.go`
- Create: `internal/cli/printblocklist_test.go` (`package cli`, internal — to test the unexported printer directly, as `printtls_test.go` does)

**Interfaces:**
- Consumes: `dnsbl.Check`, `dnsbl.DefaultZones`, `dnsbl.Report`; `sysconf.Resolvers()` for the default resolver; `a.lookupCtx()`.
- Produces:
  - `func (a *App) Blocklist(ip, resolver string) (dnsbl.Report, error)` — defaults `resolver` from `sysconf.Resolvers()` first value when empty; calls `dnsbl.Check` with `DefaultZones()` and an 8s timeout.
  - CLI: `case "blocklist": return runBlocklist(...)`; `runBlocklist` (flagset `--resolver`, `--timeout` default 8s, `--json`, one positional IP; default resolver from `sysconf.Resolvers()` when `--resolver` absent); `printBlocklistReport(w io.Writer, r dnsbl.Report)` printing verdict, resolver, and a per-list block (name, status, codes, reason).

- [ ] **Step 1 (app): Write failing test** `internal/app/dnsbl_methods_test.go`: set `dnsbl`'s `queryFn` (exported test seam if needed — otherwise point the app test at a local server) so `Blocklist("1.2.3.4","127.0.0.1:NNN")` returns a report with the expected listed count. Simpler: start a local `miekg/dns` server in the test and pass its addr as the resolver; assert `Blocklist` returns a populated `Report` and no error. *(If `queryFn` must be reachable from the app test, expose it as package var `QueryFn` in dnsbl; prefer the local-server route to avoid widening the API.)*
- [ ] **Step 2: Run, verify fail.**
- [ ] **Step 3: Implement** `Blocklist` in `dnsbl_methods.go` (default resolver via `sysconf.Resolvers()`).
- [ ] **Step 4: Run, verify pass.**
- [ ] **Step 5 (cli): Write failing tests** `internal/cli/printblocklist_test.go`:
  - `TestPrintBlocklistReportListed`: a `Report` with one listed + one clean + one blocked list → output contains the verdict, each list name, the `127.0.0.2` code, the reason, and the "incomplete"/blocked wording.
  - `TestRunBlocklistMissingArg`: `runBlocklist` with no positional returns a usage error.
- [ ] **Step 6: Run, verify fail** (printer/func undefined).
- [ ] **Step 7: Implement** the `blocklist` case, `runBlocklist`, `printBlocklistReport`, and add the usage line; import `dnsbl` and `sysconf`.
- [ ] **Step 8: Run** `go test ./internal/app/ ./internal/cli/`, verify pass.
- [ ] **Step 9: Commit** (`feat(cli,app): blocklist command and Wails method`).

### Task 4: Workbench tool `Blocklist.svelte`

**Files:**
- Create: `frontend/src/lib/tools/Blocklist.svelte`
- Modify: `frontend/src/lib/Workbench.svelte` (import, tools array entry, route)
- Regenerate: Wails bindings for the new `Blocklist` method (`App.js`/`App.d.ts`/`models.ts` only).

**Interfaces:**
- Consumes: generated `Blocklist(ip, resolver)` and `DNSSystem()` bindings.

- [ ] **Step 1: Regenerate bindings** headlessly: `$(go env GOPATH)/bin/wails generate module`; then `git checkout -- frontend/wailsjs/runtime` to keep only the additive `App.js`/`App.d.ts`/`models.ts` deltas. Verify `Blocklist` appears in `App.d.ts`.
- [ ] **Step 2: Write `Blocklist.svelte`** — IP input, resolver field pre-filled from `DNSSystem()` first value, Check button; a self-explaining one-liner about DNSBLs + the public-resolver caveat; a verdict banner (green not-listed / red listed / amber incomplete); a per-list table with status pills, mono zone, codes, and reason. Reuse the TLS tool's verdict/card visual language and theme tokens.
- [ ] **Step 3: Wire into `Workbench.svelte`**: `import Blocklist`; add `{ id: "blocklist", name: "Blocklist check", group: "Internet", ready: true }` after the TLS entry; add `{:else if active === "blocklist"}<Blocklist />` to the route line.
- [ ] **Step 4: Build** `cd frontend && npm run build`; verify green.
- [ ] **Step 5: Manual check** via `wails dev` in the browser at localhost:34115 (macOS 26 beta — native window is the known-broken upstream case): confirm the tool renders, a check runs, and the verdict/caveat read clearly.
- [ ] **Step 6: Commit** (`feat(app,ui): blocklist tool in the Workbench`).

---

## Self-Review notes

- Spec coverage: §4.1 "check blocklists" → Tasks 1–4; IPv4+IPv6 → Task 1; return-code error band → Task 1 `classify` + Task 2 verdict; resolver default/caveat → Task 3 method + Task 4 UI; user-editable zones → `DefaultZones()` is data (full edit UI deferred, noted).
- Review Focus items each map to a named test (public-resolver, IPv6, partial, invalid input, non-.2 listing).
- Proportion: plan is interface signatures + test names, bodies left to the implementer; no transcribed function bodies.
