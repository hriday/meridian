# HTTP(S) Endpoint Check — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fetch a URL and report final status, redirect chain, headers, protocol, TLS version, and per-phase timing, across engine, CLI, and the Workbench.

**Architecture:** New engine `internal/httpcheck`: `Check` issues a GET, following redirects manually (so each hop is captured) with `http.ErrUseLastResponse`, and captures per-phase timings via `net/http/httptrace`. Thin adapters: a Wails method, an `http` CLI command, and a `HttpCheck.svelte` tool under "Internet".

**Tech Stack:** Go 1.23 (`net/http`, `net/http/httptrace`), Wails v2 + Svelte. Tests use `httptest` + `stretchr/testify`.

**Spec:** `docs/superpowers/specs/2026-11-10-meridian-httpcheck-design.md`

## Global Constraints

- One-shot GET; generic `User-Agent` (`meridian/0.1 (+https://github.com/hriday/meridian)`), no user identity.
- Redirects followed manually with a cap (`maxRedirects = 10`); exceeding it is a reported outcome, not a crash.
- Non-2xx final status is a result (no error); DNS/connect/TLS/timeout are errors.
- Bare host → default scheme `https://`.
- Body read minimally then closed; this is a metadata check.
- Commit attribution: `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`. PR body ends with the Generated-with line.

## Review Focus

- **Redirect chain captured, loop capped.** A 301→200 yields two hops with correct statuses + the final URL; a self-redirect loop stops at the cap and reports "too many redirects", no infinite loop. (Task 1 tests.)
- **Non-2xx ≠ error.** A 404/500 final returns a Result with that status and nil error. (Task 1 test.)
- **Scheme defaulting.** `example.com` → `https://example.com`; an explicit scheme is kept. (Task 1 pure `normalizeURL` test.)
- **Timings present & sane.** Against a live `httptest` server, `TotalMillis >= 0` and TTFB <= Total; TLS version empty for plain HTTP, set for HTTPS. (Task 1 tests.)
- **Body/handles closed.** Every response body is closed (no leak) even on the redirect path. (Task 1 — verified by the race/leak-free suite; structurally via `defer`.)

---

### Task 1: Engine — `Check`

**Files:**
- Create: `internal/httpcheck/httpcheck.go`
- Test: `internal/httpcheck/httpcheck_test.go`

**Interfaces:**
- Produces:
  - `type Hop struct { Status int; URL string }`
  - `type Timing struct { DNSMillis, ConnectMillis, TLSMillis, TTFBMillis, TotalMillis float64 }`
  - `type Result struct { URL string; FinalURL string; Status int; StatusText string; Proto string; TLSVersion string; Server string; ContentType string; ContentLength int64; Redirects []Hop; Headers [][2]string; Timing Timing }`
  - `func normalizeURL(raw string) (string, error)` — adds `https://` when no scheme; errors on empty/garbage.
  - `func Check(ctx context.Context, rawURL string, timeout time.Duration) (Result, error)` — normalizes; manual redirect loop (client with `CheckRedirect: func(...) error { return http.ErrUseLastResponse }`, cap 10) recording each hop; on the final response fills status/proto/TLS/headers; one `httptrace.ClientTrace` over the final request fills Timing; `TotalMillis` is wall time over the whole call. Non-2xx → Result, nil error. Dial/timeout → error. Too many redirects → error ("too many redirects (>10)").

- [ ] **Step 1: Write failing tests** `internal/httpcheck/httpcheck_test.go` (`package httpcheck`):
  - `TestNormalizeURL`: `example.com`→`https://example.com`; `http://x`→`http://x`; `""`→error; `"  "`→error.
  - `TestCheckOK`: httptest server 200 with `Server: test`, `Content-Type: text/plain`; assert Status 200, StatusText contains "OK", Server "test", ContentType "text/plain", `Timing.TotalMillis >= 0`, `len(Redirects)==0`.
  - `TestCheckRedirectChain`: server with `/a` → 301 `/b`, `/b` → 200; Check the `/a` URL; assert `Redirects` has the 301 hop, final Status 200, `FinalURL` ends `/b`.
  - `TestCheckNotFound`: server returns 404; assert Status 404, nil error.
  - `TestCheckRedirectLoop`: `/loop` → 302 `/loop`; assert Check returns an error mentioning "redirect".
  - `TestCheckTLSVersion`: `httptest.NewTLSServer`; use its client's transport (seam below); assert `TLSVersion` non-empty and starts "TLS".
  - `TestCheckDialError`: `http://127.0.0.1:1`; assert error.
  - For the TLS test, expose a seam `var newTransport = func() *http.Transport { ... }` (or accept a `*http.Client` seam) so the test can inject `httptest.NewTLSServer`'s trusting transport; keep production using a default transport.
- [ ] **Step 2: Run, verify fail** (`go test ./internal/httpcheck/ -v`).
- [ ] **Step 3: Implement** `httpcheck.go`: `normalizeURL`; a client per-request with `ErrUseLastResponse`; manual loop resolving `Location` against the current URL (`url.Parse` + `ResolveReference`); httptrace hooks (DNSStart/Done, ConnectStart/Done, TLSHandshakeStart/Done, GotFirstResponseByte) recording monotonic deltas; collect final headers into `[][2]string`; set TLS from `resp.TLS.Version` via a `tlsName` helper; close every body.
- [ ] **Step 4: Run, verify pass**, then full `go test ./internal/httpcheck/` incl `-race`.
- [ ] **Step 5: Commit** (`feat(httpcheck): endpoint check with redirect chain + timing`).

### Task 2: Wails method + CLI `http`

**Files:**
- Create: `internal/app/httpcheck_methods.go`, `internal/app/httpcheck_methods_test.go`
- Modify: `internal/cli/cli.go`
- Create: `internal/cli/printhttp_test.go` (`package cli`, internal)

**Interfaces:**
- Produces:
  - `func (a *App) HTTPCheck(rawURL string) (httpcheck.Result, error)` — `httpcheck.Check(a.lookupCtx(), rawURL, 15*time.Second)`.
  - CLI: `case "http": return runHTTP(...)`; `runHTTP` (one positional URL, `--timeout` 15s, `--json`); `printHTTPResult(w, Result)` printing status/proto, redirect hops, key headers, TLS, and the timing line.

- [ ] **Step 1 (app): Write failing test** `httpcheck_methods_test.go`: an `httptest` server (200); `New().HTTPCheck(srv.URL)` returns Status 200, no error. (Since the method hardcodes the production transport, point the test at a plain-HTTP httptest server, which the default transport reaches.)
- [ ] **Step 2: Run, verify fail.**
- [ ] **Step 3: Implement** `HTTPCheck`.
- [ ] **Step 4: Run, verify pass.**
- [ ] **Step 5 (cli): Write failing tests** `printhttp_test.go`:
  - `TestPrintHTTPResult`: a Result (200, a redirect hop, Server/Content-Type, a timing) → output contains the status, the hop, the headers, and a timing figure.
  - `TestRunHTTPMissingArg`: `runHTTP(nil,…)` → exit 2.
- [ ] **Step 6: Run, verify fail.**
- [ ] **Step 7: Implement** the `http` case, `runHTTP`, `printHTTPResult`, usage line; import `httpcheck`.
- [ ] **Step 8: Run** `go test ./internal/app/ ./internal/cli/`, verify pass.
- [ ] **Step 9: Commit** (`feat(cli,app): http command and Wails method`).

### Task 3: Workbench tool `HttpCheck.svelte`

**Files:**
- Create: `frontend/src/lib/tools/HttpCheck.svelte`
- Modify: `frontend/src/lib/Workbench.svelte` (import, tools entry under "Internet", route)
- Regenerate: bindings (`App.js`/`App.d.ts`/`models.ts` — `HTTPCheck` + `httpcheck.Result`).

- [ ] **Step 1: Regenerate bindings** (`$(go env GOPATH)/bin/wails generate module`; `git checkout -- frontend/wailsjs/runtime`); verify `HTTPCheck` in `App.d.ts` and an `httpcheck` namespace in `models.ts`.
- [ ] **Step 2: Write `HttpCheck.svelte`** — URL input + Check; a status banner (green 2xx / amber 3xx-4xx / red 5xx-or-error) with the final URL and protocol; a redirect-chain list when present; a timing row (DNS/connect/TLS/TTFB/total, as a small stacked bar); a headers card (Server/Content-Type/Content-Length highlighted, full headers below). Reuse the established tool visual language.
- [ ] **Step 3: Wire into `Workbench.svelte`**: import, add `{ id: "http", name: "HTTP check", group: "Internet", ready: true }` after TLS certificate, add the `{:else if active === "http"}<HttpCheck />` route.
- [ ] **Step 4: Build** `cd frontend && npm run build`; verify green.
- [ ] **Step 5: Manual check** via the running app: check a real URL (e.g. `http://github.com` to see the →https redirect) and confirm the chain, timing, headers, and TLS render.
- [ ] **Step 6: Commit** (`feat(app,ui): HTTP check tool in the Workbench`).

---

## Self-Review notes

- Spec coverage: status/redirects/headers/TLS/timing → Task 1; CLI/app → Task 2; UI → Task 3.
- Review Focus → each maps to a Task 1 test (redirect chain + loop cap, non-2xx, scheme default, timings/TLS, body close via defer).
- Proportion: signatures + test names; bodies left to the implementer.
