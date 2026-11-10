# Raw Whois (port 43) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Query port-43 Whois for a domain/IP/ASN, discovering the authoritative server via IANA referral, and return the raw record plus the server chain, across engine, CLI, and the Workbench.

**Architecture:** New engine `internal/whois`: `Query` walks IANA → authoritative registry → optional registrar, each a plain port-43 exchange behind a dial seam; `parseReferral` is a pure referral-line extractor. Thin adapters: a Wails method, a `whois` CLI command, and a `Whois.svelte` tool under "Internet".

**Tech Stack:** Go 1.23 (`net`), Wails v2 + Svelte. Tests use local `net.Listener` TCP servers via the dial seam + `stretchr/testify`.

**Spec:** `docs/superpowers/specs/2026-11-10-meridian-whois-raw-design.md`

## Global Constraints

- Read-only TCP; the object is the only thing sent, no user identity.
- Server discovery via IANA referral (`refer:`/`whois:`); follow at most twice total (IANA → authoritative → registrar), so no referral loop.
- A terminal server (no referral) is the record; a dial/read failure or no discoverable server is an error.
- Dial behind a seam (`var dialWhois`) so tests use local listeners offline.
- Commit attribution: `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`. PR body ends with the Generated-with line.

## Review Focus

- **Referral chain walked and capped.** IANA → registry → registrar yields a 3-server chain with the registrar's record; a server that keeps referring is stopped at the 2-hop cap (no infinite loop). (Task 1 tests with chained listeners.)
- **Terminal server.** A server whose response has no referral line is the final record; the chain ends there. (Task 1 test.)
- **Referral parsing.** `refer:`, `whois:`, `Registrar WHOIS Server:`, `ReferralServer:` (case-insensitive, with a `whois://` or bare host value) all extract the host; a line without one yields "". (Task 1 pure `parseReferral` test.)
- **Query framing.** The object is sent followed by CRLF; the full response is read until the server closes. (Task 1 test asserts the listener received `<object>\r\n`.)
- **Dial failure → error.** (Task 1 test with the seam returning an error.)

---

### Task 1: Engine — `Query` + `parseReferral`

**Files:**
- Create: `internal/whois/whois.go`
- Test: `internal/whois/whois_test.go`

**Interfaces:**
- Produces:
  - `type Result struct { Object string; Server string; Chain []string; Raw string }`
  - `var dialWhois = func(ctx context.Context, addr string) (net.Conn, error)` — default dials `addr` (host:43) with a dialer; a seam for tests.
  - `func parseReferral(body string) string` — returns the referred host from the first `refer:`/`whois:`/`registrar whois server:`/`referralserver:` line (case-insensitive), stripping a `whois://` scheme and any port; "" if none.
  - `func queryServer(ctx context.Context, server, object string) (string, error)` — dials `server:43` (via seam), writes `object + "\r\n"`, reads all, returns the text.
  - `func Query(ctx context.Context, object string, timeout time.Duration) (Result, error)` — starts at `whois.iana.org`; for a domain uses the TLD for the IANA step and the full object thereafter; follows referrals up to 2 hops; records each server in `Chain`; returns the last record. Errors on no server / dial failure.

- [ ] **Step 1: Write failing tests** `internal/whois/whois_test.go` (`package whois`): a helper that starts N local TCP whois listeners (each: read the query line, write a canned response, close) and routes `dialWhois` to them by host via a map.
  - `TestParseReferral`: `"refer: whois.arin.net\n"`→`whois.arin.net`; `"whois:  whois.verisign-grs.com"`→host; `"Registrar WHOIS Server: whois.example-registrar.com"`→host; `"ReferralServer: whois://whois.ripe.net:43"`→`whois.ripe.net`; `"no referral here"`→"".
  - `TestQueryFollowsChain`: iana listener refers to "registry.test"; registry listener (record for the object) refers to "registrar.test" via `Registrar WHOIS Server:`; registrar listener returns the final record. Assert `Result.Raw` is the registrar record, `Server=="registrar.test"`, `Chain==["whois.iana.org","registry.test","registrar.test"]`.
  - `TestQueryTerminal`: iana refers to "registry.test"; registry returns a record with no referral. Assert `Raw` is the registry record, `Server=="registry.test"`, `len(Chain)==2`.
  - `TestQueryCapsHops`: every listener refers onward; assert the walk stops after the 2-hop cap (Chain length bounded, no hang) and returns the last record reached.
  - `TestQuerySendsObjectCRLF`: the listener captures what it received; assert it equals `"<object>\r\n"`.
  - `TestQueryDialError`: `dialWhois` returns an error for the first server → `Query` errors.
- [ ] **Step 2: Run, verify fail** (`go test ./internal/whois/ -v`).
- [ ] **Step 3: Implement** `whois.go`: `parseReferral` (scan lines, prefix-match the four labels case-insensitively, strip `whois://` and `:port`); `queryServer` (dial `net.JoinHostPort(server,"43")`, set deadline from ctx, write, `io.ReadAll`); `Query` (TLD extraction for domains via last `.`; loop with a hop cap of 2 after IANA; dedupe a self-referral; build Chain).
- [ ] **Step 4: Run, verify pass**, then full `go test ./internal/whois/` incl `-race`.
- [ ] **Step 5: Commit** (`feat(whois): port-43 query with IANA referral discovery`).

### Task 2: Wails method + CLI `whois`

**Files:**
- Create: `internal/app/whois_methods.go`, `internal/app/whois_methods_test.go`
- Modify: `internal/cli/cli.go`
- Create: `internal/cli/printwhois_test.go` (`package cli`, internal)

**Interfaces:**
- Produces:
  - `func (a *App) Whois(object string) (whois.Result, error)` — `whois.Query(a.lookupCtx(), object, 15*time.Second)`.
  - CLI: `case "whois": return runWhois(...)`; `runWhois` (one positional object, `--json`); `printWhoisResult(w, Result)` printing the server chain then the raw record.

- [ ] **Step 1 (app): Write failing test** `whois_methods_test.go`: swap `whois.dialWhois` to route to a local listener (single terminal server by overriding so IANA refers to it, or point IANA directly at a canned record); assert `New().Whois("example.com")` returns a populated Result. *(Keep it simple: a helper that makes IANA a terminal server returning a record; assert Raw non-empty.)*
- [ ] **Step 2: Run, verify fail.**
- [ ] **Step 3: Implement** `Whois`.
- [ ] **Step 4: Run, verify pass.**
- [ ] **Step 5 (cli): Write failing tests** `printwhois_test.go`:
  - `TestPrintWhoisResult`: a Result with a chain + raw text → output contains the chain servers and the raw record.
  - `TestRunWhoisMissingArg`: `runWhois(nil,…)` → exit 2.
- [ ] **Step 6: Run, verify fail.**
- [ ] **Step 7: Implement** the `whois` case, `runWhois`, `printWhoisResult`, usage line; import `whois`.
- [ ] **Step 8: Run** `go test ./internal/app/ ./internal/cli/`, verify pass.
- [ ] **Step 9: Commit** (`feat(cli,app): whois command and Wails method`).

### Task 3: Workbench tool `Whois.svelte`

**Files:**
- Create: `frontend/src/lib/tools/Whois.svelte`
- Modify: `frontend/src/lib/Workbench.svelte` (import, tools entry under "Internet", route)
- Regenerate: bindings (`App.js`/`App.d.ts`/`models.ts` — `Whois` + `whois.Result`).

- [ ] **Step 1: Regenerate bindings** (`$(go env GOPATH)/bin/wails generate module`; `git checkout -- frontend/wailsjs/runtime`); verify `Whois` in `App.d.ts` and a `whois` namespace in `models.ts`.
- [ ] **Step 2: Write `Whois.svelte`** — object input + Look up; a self-explaining lede pointing to Lookup's Registry tab for the structured view; the server chain as a breadcrumb; the raw record in a scrollable mono `<pre>`; a clear error. Reuse the established tool visual language.
- [ ] **Step 3: Wire into `Workbench.svelte`**: import, add `{ id: "whois", name: "Whois", group: "Internet", ready: true }` after HTTP check, add the `{:else if active === "whois"}<Whois />` route.
- [ ] **Step 4: Build** `cd frontend && npm run build`; verify green.
- [ ] **Step 5: Manual check** via the running app: look up a domain and an IP, confirm the chain + raw record render.
- [ ] **Step 6: Commit** (`feat(app,ui): whois tool in the Workbench`).

---

## Self-Review notes

- Spec coverage: IANA discovery + chain + raw record → Task 1; CLI/app → Task 2; UI → Task 3.
- Review Focus → each maps to a Task 1 test (chain walk, terminal, parse, CRLF framing, dial error, hop cap).
- Proportion: signatures + test names; bodies left to the implementer.
