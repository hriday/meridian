# MAC / OUI Vendor Finder — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Look up the vendor of a MAC address / OUI prefix via the keyless public MAC-vendor API, across engine, CLI, and the Workbench.

**Architecture:** New self-contained engine `internal/macvendor`: `normalize` (pure) validates and extracts the OUI; `Lookup` fetches the vendor over an injected `Doer` against a base-URL seam (mirrors the lookup/peeringdb HTTP pattern, generic User-Agent, no user identity). Thin adapters: a Wails method, a `macvendor` CLI command, and a `MacVendor.svelte` tool under "Local network".

**Tech Stack:** Go 1.23 (`net/http`), Wails v2 + Svelte. Tests use `httptest` + `stretchr/testify`.

**Spec:** `docs/superpowers/specs/2026-11-10-meridian-macvendor-design.md`

## Global Constraints

- Keyless public API; generic `User-Agent` (`meridian/0.1 (+https://github.com/hriday/meridian)`), no user identity — only the OUI is sent.
- Input accepts colon/hyphen/dot/bare spellings; ≥3 bytes (6 hex) required.
- 404 is "not registered" (Found=false), **not** an error; 429 is a distinct rate-limit error; other non-2xx and transport failures are errors.
- Locally-administered MACs (first byte's bit 0x02 set) are flagged — their OUI has no real vendor.
- Base URL behind a package var so httptest can swap it; `Doer` injected so tests stay offline.
- Commit attribution: `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`. PR body ends with the Generated-with line.

## Review Focus

- **404 ≠ error.** An unregistered OUI must return `Found=false` with a nil error, not a failure. (Task 1 Lookup test.)
- **429 is distinct.** A rate-limit must surface as a specific "≈1/sec, try again" error, not a generic status error. (Task 1 test.)
- **Spellings normalize identically.** `fc:fb:fb`, `fc-fb-fb`, `fcfb.fb…`, `fcfbfb`, and a full MAC all yield OUI `FC:FB:FB`. (Task 1 normalize test.)
- **Too-short / non-hex rejected before any request.** `fc:fb`, `zz:fb:fb`, `""` → error, no network. (Task 1 test.)
- **Locally-administered flag.** `02:…`, `x2/x6/xA/xE` first byte → `LocallyAdministered=true`; a global OUI (e.g. `fc:…`) → false. (Task 1 normalize test, exact bit.)

---

### Task 1: Engine — normalize + Lookup

**Files:**
- Create: `internal/macvendor/macvendor.go`
- Test: `internal/macvendor/macvendor_test.go`

**Interfaces:**
- Produces:
  - `type Doer interface { Do(*http.Request) (*http.Response, error) }`
  - `type Result struct { Input string; OUI string; Vendor string; Found bool; LocallyAdministered bool }`
  - `var baseURL = "https://api.macvendors.com/"` (package var, swappable in tests)
  - `func normalize(input string) (oui string, query string, localAdmin bool, err error)` — strips separators, requires ≥6 hex, returns the OUI as `AA:BB:CC` (upper), the query (the 6-hex OUI), and whether bit `0x02` of the first byte is set.
  - `func Lookup(ctx context.Context, d Doer, input string) (Result, error)` — normalizes (bad input → error, no request); GETs `baseURL+query` with the generic User-Agent; 200 → `Vendor = trimmed body, Found=true`; 404 → `Found=false, nil error`; 429 → rate-limit error; other non-2xx / transport error → error. Fills `Input/OUI/LocallyAdministered`.

- [ ] **Step 1: Write failing tests** `internal/macvendor/macvendor_test.go` (`package macvendor`):
  - `TestNormalizeSpellings`: `fc:fb:fb`, `fc-fb-fb`, `fcfb.fbdd.eeff`, `fcfbfbddeeff`, `FC:FB:FB:DD:EE:FF` → oui `FC:FB:FB`, query `FCFBFB` (or lowercase — assert consistent), no error.
  - `TestNormalizeRejects`: `""`, `fc:fb`, `fcfb`, `zz:fb:fb` → error.
  - `TestNormalizeLocalAdmin`: `02:00:00`→true; `06:..`→true; `0a:..`→true; `0e:..`→true; `fc:fb:fb`→false; `00:..`→false.
  - `TestLookupFound`: httptest server returning 200 `"Apple, Inc."`; swap `baseURL` to it; `Lookup(ctx, srv.Client(), "fc:fb:fb:dd:ee:ff")` → `Vendor=="Apple, Inc."`, `Found==true`, `OUI=="FC:FB:FB"`, nil error.
  - `TestLookupNotFound`: server returns 404; `Found==false`, nil error.
  - `TestLookupRateLimited`: server returns 429; error, and its message mentions the rate limit (e.g. contains "second" or "rate").
  - `TestLookupNetworkError`: a `Doer` returning an error → Lookup returns an error.
  - (Restore `baseURL` with `t.Cleanup` in each server test.)
- [ ] **Step 2: Run, verify fail** (`go test ./internal/macvendor/ -v`).
- [ ] **Step 3: Implement** `macvendor.go`: `normalize` (strip `:-.`, validate hex+len, uppercase OUI, `localAdmin = firstByte & 0x02 != 0`), `Lookup` (http.NewRequestWithContext, set User-Agent, read body with a small LimitReader, switch on status). Keep the `userAgent` constant local to the package.
- [ ] **Step 4: Run, verify pass.**
- [ ] **Step 5: Commit** (`feat(macvendor): OUI normalize + keyless vendor lookup`).

### Task 2: Wails method + CLI `macvendor`

**Files:**
- Create: `internal/app/macvendor_methods.go`, `internal/app/macvendor_methods_test.go`
- Modify: `internal/cli/cli.go`
- Create: `internal/cli/printmacvendor_test.go` (`package cli`, internal)

**Interfaces:**
- Produces:
  - `var macLookup = macvendor.Lookup` (app package var, swappable in the app test)
  - `func (a *App) MACVendor(mac string) (macvendor.Result, error)` — calls `macLookup` with an 8s-timeout `*http.Client` and `a.lookupCtx()`.
  - CLI: `case "macvendor": return runMACVendor(...)`; `runMACVendor` (one positional MAC/OUI, `--json`); builds an `*http.Client{Timeout: 8s}`, calls `macvendor.Lookup`; `printMACVendorReport(w, Result)` printing the OUI and vendor, or a "not a registered OUI" line (with the locally-administered note when set).

- [ ] **Step 1 (app): Write failing test** `macvendor_methods_test.go`: swap `macLookup` to a fake returning a known `Result`; assert `New().MACVendor("fc:fb:fb")` returns it; restore in `t.Cleanup`.
- [ ] **Step 2: Run, verify fail.**
- [ ] **Step 3: Implement** `MACVendor` (+ the `macLookup` seam).
- [ ] **Step 4: Run, verify pass.**
- [ ] **Step 5 (cli): Write failing tests** `printmacvendor_test.go`:
  - `TestPrintMACVendorFound`: a `Result{OUI:"FC:FB:FB", Vendor:"Apple, Inc.", Found:true}` → output contains both.
  - `TestPrintMACVendorNotFound`: `Result{OUI:"02:00:00", Found:false, LocallyAdministered:true}` → output says it's not registered AND mentions locally-administered/random.
  - `TestRunMACVendorMissingArg`: `runMACVendor(nil,…)` → exit 2.
- [ ] **Step 6: Run, verify fail.**
- [ ] **Step 7: Implement** the `macvendor` case, `runMACVendor`, `printMACVendorReport`, usage line; import `macvendor` and `net/http`.
- [ ] **Step 8: Run** `go test ./internal/app/ ./internal/cli/`, verify pass.
- [ ] **Step 9: Commit** (`feat(cli,app): macvendor command and Wails method`).

### Task 3: Workbench tool `MacVendor.svelte`

**Files:**
- Create: `frontend/src/lib/tools/MacVendor.svelte`
- Modify: `frontend/src/lib/Workbench.svelte` (import, tools entry under "Local network", route)
- Regenerate: bindings (`App.js`/`App.d.ts`/`models.ts` — `MACVendor` + `macvendor.Result`).

- [ ] **Step 1: Regenerate bindings** (`$(go env GOPATH)/bin/wails generate module`; `git checkout -- frontend/wailsjs/runtime`); verify `MACVendor` in `App.d.ts` and a `macvendor` namespace in `models.ts`.
- [ ] **Step 2: Write `MacVendor.svelte`** — MAC/OUI input, Look up button; a self-explaining lede; on a hit a result card (vendor + OUI), on a miss a plain "not a registered OUI" note (with the locally-administered hint when set), on error a clear message. Reuse the established tool visual language and theme tokens.
- [ ] **Step 3: Wire into `Workbench.svelte`**: import, add `{ id: "macvendor", name: "MAC vendor", group: "Local network", ready: true }` after Wake-on-LAN, add the `{:else if active === "macvendor"}<MacVendor />` route.
- [ ] **Step 4: Build** `cd frontend && npm run build`; verify green.
- [ ] **Step 5: Manual check** via the running app (localhost:34115): look up a known OUI (e.g. `fc:fb:fb`) and a locally-administered one (`02:00:00`); confirm the hit, miss, and notes read clearly.
- [ ] **Step 6: Commit** (`feat(app,ui): MAC vendor tool in the Workbench`).

---

## Self-Review notes

- Spec coverage: §4.6 MAC finder → Tasks 1–3; normalize/local-admin → Task 1; keyless lookup with 404/429 handling → Task 1; CLI/app → Task 2; UI → Task 3.
- Review Focus → each maps to a named test (404-not-error, 429-distinct, spellings, rejection, local-admin bit).
- Proportion: signatures + test names; bodies left to the implementer. Self-contained HTTP mirrors the lookup client's shape.
