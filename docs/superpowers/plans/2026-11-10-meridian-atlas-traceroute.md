# RIPE Atlas Remote Traceroute — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Run a one-off traceroute from worldwide RIPE Atlas probes against a target, reusing the operator's stored key, and show the per-probe path (hops + RTT) with a global summary.

**Architecture:** Reuse `internal/settings` (key) and the scheduling/polling engine in `internal/atlas`. Extract the shared measurement POST (`scheduleMeasurement`) and raw results GET (`fetchResults`) so both ping and traceroute use them; add traceroute definition building + hop parsing + `Trace`. An `AtlasTrace` app method mirrors `AtlasPing`. A "Remote traceroute" Workbench tool reuses the stored key (no new key entry).

**Tech Stack:** Go 1.23 (`net/http`, `encoding/json`, `netip`), Wails v2 + Svelte. Tests use `httptest` + the base-URL seam + `stretchr/testify`.

**Spec:** `docs/superpowers/specs/2026-11-10-meridian-atlas-traceroute-design.md`

## Global Constraints

- The key is a secret: never committed/hardcoded, read only from the local settings file, sent only to `atlas.ripe.net` in `Authorization: Key …`, never returned to the frontend. This slice adds no new key handling.
- Atlas base URL behind the existing package var; `Doer` injected; config dir behind the existing seam. No live Atlas in tests.
- The shared refactor MUST keep existing ping tests green.
- Commit attribution: `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`. PR body ends with the Generated-with line.

## Review Focus

- **Shared schedule refactor preserves ping behavior.** Existing ping tests still pass; `scheduleMeasurement` posts the exact definition it is given. (Task 1 — existing suite + a new assertion.)
- **Timeout hops parse as timeouts, not 0ms responders.** An Atlas hop whose replies are all `{"x":"*"}` becomes a timeout hop, never `from:"" rtt:0`. (Task 1 test.)
- **Reached detection.** A probe whose path includes a reply from the destination address is marked reached; one that never reaches it is not. (Task 1 test.)
- **No key → AtlasTrace errors clearly** telling the user to add a key in Remote ping. (Task 2 test.)
- **Poll-until-results + graceful timeout** reused from the ping engine; a timeout with nothing reported yields an empty report + no error. (Task 1 test.)

---

### Task 1: `internal/atlas` — shared schedule/fetch + traceroute engine

**Files:**
- Modify: `internal/atlas/atlas.go` (extract `scheduleMeasurement`, `fetchResults`; have ping use them)
- Create: `internal/atlas/traceroute.go`
- Test: `internal/atlas/traceroute_test.go` (`package atlas`)

**Interfaces:**
- Consumes: `Doer`, `baseURL`, `scheduleError`, `userAgent` (existing).
- Produces:
  - `func scheduleMeasurement(ctx context.Context, d Doer, key string, def map[string]any, probes int) (int, error)` — POST `measurements/` with `{definitions:[def], probes:[{requested,type:"area",value:"WW"}]}`; key header; parse `{"measurements":[id]}`; map errors via `scheduleError`.
  - `func fetchResults(ctx context.Context, d Doer, key string, id int) ([]byte, error)` — GET `measurements/<id>/results/`; return the raw body or a mapped error.
  - (Refactor) `schedule` builds the ping def and calls `scheduleMeasurement`; `results` calls `fetchResults` then parses.
  - `type Hop struct { Number int; From string; RttMs float64; Timeout bool }`
  - `type TraceProbe struct { ID int; From string; Dest string; Hops []Hop; Reached bool }`
  - `type TraceReport struct { Target string; MeasurementID int; Probes []TraceProbe; Responding int; Reached int }`
  - `func scheduleTrace(ctx, d Doer, key, target string, probes int) (int, error)` — def `{type:"traceroute", af, target, description, protocol:"ICMP", is_oneoff:true}`.
  - `func traceResults(ctx, d Doer, key string, id int) ([]TraceProbe, error)` — parse `[{prb_id, from, dst_addr, result:[{hop, result:[{from,rtt}|{x:"*"}]}]}]`; per hop: `Timeout` when no reply has a `from`, else `From` = first responder and `RttMs` = min rtt among responders; `Reached` when any hop responder's `from` == `dst_addr`.
  - `func Trace(ctx, d Doer, key, target string, probes int, poll, timeout time.Duration) (TraceReport, error)` — `scheduleTrace`, then poll `traceResults` until non-empty or timeout (same graceful loop as `Ping`); aggregate `Responding` (probes with ≥1 hop) and `Reached`.

- [ ] **Step 1: Extract shared helpers** in `atlas.go` (`scheduleMeasurement`, `fetchResults`); rewrite `schedule`/`results` to use them. Run `go test ./internal/atlas/` — existing ping tests must stay green (regression gate).
- [ ] **Step 2: Write failing tests** `traceroute_test.go`:
  - `TestScheduleTraceSendsTracerouteDef`: handler asserts the POST body has `"type":"traceroute"` and `Authorization: Key testkey`; returns `{"measurements":[55]}`; `scheduleTrace(...)` → 55.
  - `TestTraceResultsParsesHops`: handler returns one probe with hops incl. a timeout hop (`[{"x":"*"},{"x":"*"},{"x":"*"}]`) and a final hop whose `from` == `dst_addr`; assert hop count, the timeout hop `Timeout==true`/`From==""`, a responder hop `From`+`RttMs` (min of its replies), and `Reached==true`.
  - `TestTraceResultsNotReached`: no hop reaches `dst_addr` → `Reached==false`.
  - `TestTracePollsUntilResults`: results `[]` then populated; `Trace(..., poll=5ms, timeout=2s)` → report with 1 probe, `Responding==1`.
  - `TestTraceTimeoutEmpty`: results always `[]`; `Trace(..., timeout=40ms)` → empty report, no error.
- [ ] **Step 3: Run, verify fail.**
- [ ] **Step 4: Implement** `traceroute.go`.
- [ ] **Step 5: Run, verify pass**, then `-race` on the package.
- [ ] **Step 6: Commit** (`feat(atlas): one-off RIPE Atlas traceroute (shared schedule engine)`).

### Task 2: App method — AtlasTrace

**Files:**
- Create: `internal/app/atlastrace_methods.go`, `internal/app/atlastrace_methods_test.go`

**Interfaces:**
- Produces:
  - `var atlasTrace = atlas.Trace` (seam).
  - `func (a *App) AtlasTrace(target string, probes int) (atlas.TraceReport, error)` — load the key (error "No RIPE Atlas key set — add it in the Remote ping tool." when empty); clamp probes to [1, 20] (default 5); build an `*http.Client`; call `atlasTrace` with a poll + timeout (reuse the ping constants) under a context with headroom over the poll timeout.

- [ ] **Step 1: Write failing tests** (`package app`, reuse `useTempSettings`):
  - `TestAtlasTraceNoKey`: no key → error mentions a key.
  - `TestAtlasTraceUsesKey`: set a key; swap `atlasTrace` to a fake asserting key+target → returns a report; assert pass-through.
  - `TestAtlasTraceClampsProbes`: 0 → default ≥1; 9999 → ≤20.
- [ ] **Step 2: Run, verify fail.**
- [ ] **Step 3: Implement.**
- [ ] **Step 4: Run, verify pass.**
- [ ] **Step 5: Commit** (`feat(app): AtlasTrace remote traceroute method`).

### Task 3: Workbench tool `AtlasTrace.svelte`

**Files:**
- Create: `frontend/src/lib/tools/AtlasTrace.svelte`
- Modify: `frontend/src/lib/Workbench.svelte` (import, entry under Internet after Remote ping, route)
- Regenerate: bindings (`App.js`/`App.d.ts`/`models.ts` — `AtlasTrace` + `atlas.TraceReport`/`atlas.TraceProbe`/`atlas.Hop`).

- [ ] **Step 1: Regenerate bindings** (`$(go env GOPATH)/bin/wails generate module`; `git checkout -- frontend/wailsjs/runtime`); verify `AtlasTrace` in `App.d.ts` and the trace types in `models.ts`.
- [ ] **Step 2: Write `AtlasTrace.svelte`** — on mount calls `AtlasKeyStatus()`:
  - no key → a short panel: "A RIPE Atlas key is needed — add it in the Remote ping tool." (no key entry here).
  - key set → the trace form (target input, probe count default 5, Run) calling `AtlasTrace`.
  - results → a summary line (responding / reached) + per-probe hop lists (`hop · address · rtt`, `*` for timeout, destination marked when reached); a "measuring…" busy state; clear key/credits errors. Reuse the established visual language; `fmt` treats `null`/`<0` as `–`.
- [ ] **Step 3: Wire into `Workbench.svelte`**: import, add `{ id: "atlastrace", name: "Remote traceroute", group: "Internet", ready: true }` after `atlasping`, add the `{:else if active === "atlastrace"}<AtlasTrace />` route.
- [ ] **Step 4: Build** `cd frontend && npm run build`; verify green.
- [ ] **Step 5: Manual check** via the running app: the tool renders (key-set form, since a key is stored) with no console errors.
- [ ] **Step 6: Commit** (`feat(app,ui): Remote traceroute (RIPE Atlas) tool`).

---

## Self-Review notes

- Spec coverage: shared engine + traceroute parse → Task 1; app method → Task 2; UI reusing stored key → Task 3.
- Review Focus → each maps to a named test (ping-regression, timeout-hop, reached, no-key, poll/timeout).
- Proportion: signatures + test names; bodies left to the implementer. No key in any fixture.
- DRY: the scheduling POST and results GET are shared with ping rather than duplicated.
