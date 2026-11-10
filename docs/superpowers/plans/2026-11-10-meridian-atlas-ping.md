# RIPE Atlas Remote Ping — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let the operator enter their RIPE Atlas API key in-app (stored locally), then run a one-off ping from worldwide Atlas probes and see per-probe + aggregate results.

**Architecture:** `internal/settings` persists the key locally (`os.UserConfigDir()/meridian/settings.json`, 0600). `internal/atlas` schedules a one-off ping over an injected `Doer` (base-URL seam, `Authorization: Key` header), polls results, and aggregates. App methods manage the key and run `AtlasPing`. A "Remote ping" Workbench tool handles key setup + the ping form + results.

**Tech Stack:** Go 1.23 (`net/http`, `encoding/json`, `os`), Wails v2 + Svelte. Tests use `httptest` + a temp config dir + `stretchr/testify`.

**Spec:** `docs/superpowers/specs/2026-11-10-meridian-atlas-ping-design.md`

## Global Constraints

- The key is a secret: never committed, never hardcoded, written only to the local settings file (0600), sent only to `atlas.ripe.net` in `Authorization: Key …`, and never returned to the frontend in full (masked hint only).
- Atlas base URL behind a package var; `Doer` injected so tests stay offline. Config dir behind a seam for tests.
- 403/invalid key and insufficient-credits errors map to clear messages, not raw status codes.
- Commit attribution: `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`. PR body ends with the Generated-with line.

## Review Focus

- **Key never leaks to the frontend.** `AtlasKeyHint` returns only a masked hint + a boolean, never the full key. (Task 3 test.)
- **Key file perms.** The settings file is written `0600`. (Task 1 test asserts the mode.)
- **Missing config → empty key, no error.** First run with no file yields "" and no error. (Task 1 test.)
- **Insufficient-credits error is distinct.** An Atlas 4xx with a credits message surfaces a credits-specific error, not a generic one. (Task 2 test.)
- **Poll-until-results.** `Ping` keeps polling while results are empty and returns once they arrive (or times out), without hanging. (Task 2 test with a server that returns empty then populated.)
- **No key set → AtlasPing errors clearly** telling the user to add a key. (Task 3 test.)

---

### Task 1: `internal/settings` — local key storage

**Files:**
- Create: `internal/settings/settings.go`
- Test: `internal/settings/settings_test.go`

**Interfaces:**
- Produces:
  - `var configDir = os.UserConfigDir` (seam)
  - `func AtlasKey() (string, error)` — reads `<configDir>/meridian/settings.json`; missing file → `("", nil)`.
  - `func SetAtlasKey(key string) error` — creates `<configDir>/meridian/` (0700) and writes `settings.json` (0600) with the key (trimmed); empty key is allowed (acts like clear).
  - `func ClearAtlasKey() error` — sets the key to "" (or removes it from the file).
  - internal `type file struct { AtlasAPIKey string \`json:"atlasApiKey"\` }`.

- [ ] **Step 1: Write failing tests** `internal/settings/settings_test.go` (`package settings`): a helper pointing `configDir` at `t.TempDir()` (restore via `t.Cleanup`).
  - `TestAtlasKeyMissingFileEmpty`: fresh temp dir → `AtlasKey()` returns `"", nil`.
  - `TestSetAndGetAtlasKey`: `SetAtlasKey("abc")` then `AtlasKey()` → `"abc"`.
  - `TestSetAtlasKeyFilePerms`: after `SetAtlasKey`, stat `<dir>/meridian/settings.json`, assert mode `0600`.
  - `TestClearAtlasKey`: set then clear → `AtlasKey()` returns `""`.
  - `TestSetAtlasKeyTrims`: `SetAtlasKey("  k \n")` → stored `"k"`.
- [ ] **Step 2: Run, verify fail** (`go test ./internal/settings/ -v`).
- [ ] **Step 3: Implement** `settings.go` (JSON read/write; `os.MkdirAll(dir, 0700)`; `os.WriteFile(path, data, 0600)`; tolerate `os.IsNotExist`).
- [ ] **Step 4: Run, verify pass.**
- [ ] **Step 5: Commit** (`feat(settings): local Atlas API key storage (0600)`).

### Task 2: `internal/atlas` — schedule + poll + aggregate ping

**Files:**
- Create: `internal/atlas/atlas.go`
- Test: `internal/atlas/atlas_test.go`

**Interfaces:**
- Produces:
  - `type Doer interface { Do(*http.Request) (*http.Response, error) }`
  - `var baseURL = "https://atlas.ripe.net/api/v2/"`
  - `type Probe struct { ID int; From string; MinMs, AvgMs, MaxMs float64; Sent, Received int }`
  - `type Report struct { Target string; MeasurementID int; Probes []Probe; Responding int; MinMs, AvgMs, MaxMs, LossPct float64 }`
  - `func schedule(ctx, d Doer, key, target string, probes int) (int, error)` — POST `measurements/` with the key header; body `{definitions:[{type:"ping",af,target,description,is_oneoff:true,packets:3}],probes:[{requested:probes,type:"area",value:"WW"}]}`; af = 6 if target is an IPv6 literal else 4; parses `{"measurements":[id]}`; an Atlas error body maps to a clear error (credits → distinct).
  - `func results(ctx, d Doer, key string, id int) ([]Probe, error)` — GET `measurements/<id>/results/`; parse `[{prb_id, from, min, avg, max, sent, rcvd}]` (ping; `min/avg/max` may be -1 when lost).
  - `func Ping(ctx, d Doer, key, target string, probes int, poll, timeout time.Duration) (Report, error)` — schedule, then poll `results` until non-empty or timeout; aggregate (responding = Received>0; global min/avg/max over responding; mean LossPct).

- [ ] **Step 1: Write failing tests** `internal/atlas/atlas_test.go` (`package atlas`), httptest server + base-URL seam:
  - `TestScheduleSendsKeyAndParsesID`: handler asserts `Authorization: Key testkey` and returns `{"measurements":[42]}`; `schedule(...)` → 42.
  - `TestScheduleInsufficientCredits`: handler returns 400 `{"error":{"detail":"...credits...","status":400}}`; assert the error mentions credits.
  - `TestScheduleInvalidKey`: handler returns 403; assert an error (mentions key/authorization).
  - `TestResultsParsesPing`: handler returns `[{"prb_id":1,"from":"9.9.9.9","min":1.1,"avg":1.5,"max":2.0,"sent":3,"rcvd":3}]`; assert the Probe fields.
  - `TestPingPollsUntilResults`: schedule returns id; results returns `[]` on the first GET then the populated array on the second; `Ping(..., poll=5ms, timeout=2s)` → Report with 1 probe, Responding 1, aggregates set.
  - `TestPingTimeoutEmpty`: results always `[]`; `Ping(..., timeout=50ms)` returns a Report with 0 probes and no error (nothing reported in time) — or a clear "no probes reported" error; pick one and test it.
- [ ] **Step 2: Run, verify fail.**
- [ ] **Step 3: Implement** `atlas.go` (JSON bodies; `req.Header.Set("Authorization", "Key "+key)`; generic User-Agent; parse Atlas error envelope; af from `netip.ParseAddr`+`Is6`; poll loop with a timer + ctx).
- [ ] **Step 4: Run, verify pass**, then `-race`.
- [ ] **Step 5: Commit** (`feat(atlas): schedule + poll a one-off RIPE Atlas ping`).

### Task 3: App methods — key management + AtlasPing

**Files:**
- Create: `internal/app/atlas_methods.go`, `internal/app/atlas_methods_test.go`

**Interfaces:**
- Produces:
  - `type AtlasKeyStatus struct { Set bool; Hint string }`
  - `func (a *App) AtlasKeyStatus() (AtlasKeyStatus, error)` — `Set` = key non-empty; `Hint` = masked (`"…"+last6`), never the full key.
  - `func (a *App) SetAtlasKey(key string) error` / `func (a *App) ClearAtlasKey() error` — via settings.
  - `func (a *App) AtlasPing(target string, probes int) (atlas.Report, error)` — loads the key (error "No RIPE Atlas key set — add it in the Remote ping tool." when empty), clamps probes (1..50, default 10), calls `atlas.Ping` with a built `*http.Client`, a 5s poll and ~60s timeout. Seam the atlas call (`var atlasPing = atlas.Ping`) and the key read (via `settings`, with its `configDir` seam) so the test stays offline.

- [ ] **Step 1: Write failing tests** `atlas_methods_test.go`: point `settings.configDir` at a temp dir.
  - `TestAtlasKeyStatusMasks`: `SetAtlasKey("abcdef1234567890")`; `AtlasKeyStatus()` → `Set true`, `Hint` contains the last 6 (`567890`) and NOT the full key.
  - `TestAtlasPingNoKey`: no key → `AtlasPing("1.1.1.1", 5)` errors, message mentions adding a key.
  - `TestAtlasPingUsesKey`: set a key; swap `atlasPing` to a fake asserting it received that key and target → returns a Report; assert it passes through.
  - `TestClearAtlasKey`: set, clear, status `Set false`.
- [ ] **Step 2: Run, verify fail.**
- [ ] **Step 3: Implement** the methods + seams.
- [ ] **Step 4: Run, verify pass.**
- [ ] **Step 5: Commit** (`feat(app): Atlas key management + AtlasPing method`).

### Task 4: Workbench tool `AtlasPing.svelte`

**Files:**
- Create: `frontend/src/lib/tools/AtlasPing.svelte`
- Modify: `frontend/src/lib/Workbench.svelte` (import, tools entry under "Internet", route)
- Regenerate: bindings (`App.js`/`App.d.ts`/`models.ts` — `AtlasKeyStatus`/`SetAtlasKey`/`ClearAtlasKey`/`AtlasPing` + `atlas.Report`/`atlas.Probe`).

- [ ] **Step 1: Regenerate bindings** (`$(go env GOPATH)/bin/wails generate module`; `git checkout -- frontend/wailsjs/runtime`); verify the methods in `App.d.ts` and an `atlas` namespace in `models.ts`.
- [ ] **Step 2: Write `AtlasPing.svelte`** — on mount calls `AtlasKeyStatus()`:
  - no key → a setup panel: password input + Save (calls `SetAtlasKey`, refreshes status), a note (what it's for, stored locally only, link to atlas.ripe.net, needs credits).
  - key set → a masked "Atlas key set · {hint}" line with Change/Remove, then the ping form (target input, probe count default 10, Run) calling `AtlasPing`.
  - results → a summary banner (responding / min-avg-max / loss) + a per-probe list (id, from, RTT, loss); a "measuring…" busy state; clear error messages (key/credits). Reuse the established visual language. The key field is `type="password"`; never render the full key.
- [ ] **Step 3: Wire into `Workbench.svelte`**: import, add `{ id: "atlasping", name: "Remote ping", group: "Internet", ready: true }` after BGP, add the `{:else if active === "atlasping"}<AtlasPing />` route.
- [ ] **Step 4: Build** `cd frontend && npm run build`; verify green.
- [ ] **Step 5: Manual check** via the running app: confirm the no-key setup panel renders with the note; (the owner will paste their own key to test live — do not enter a key).
- [ ] **Step 6: Commit** (`feat(app,ui): Remote ping (RIPE Atlas) tool in the Workbench`).

---

## Self-Review notes

- Spec coverage: local key storage + entry → Task 1/3/4; schedule/poll/aggregate → Task 2; UI with setup + results → Task 4.
- Review Focus → each maps to a named test (key-not-leaked, 0600 perms, missing-config, credits error, poll-until-results, no-key error).
- Proportion: signatures + test names; bodies left to the implementer. The key is never written to the repo or a fixture.
