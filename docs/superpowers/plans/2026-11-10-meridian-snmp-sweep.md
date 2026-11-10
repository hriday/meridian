# SNMP sweep Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Sweep a CIDR for SNMP responders and list each one's IP, name, description and uptime, filling in live.

**Architecture:** An `snmp.Sweep` engine mirroring `scan.Sweep` (CIDR enumerate + bounded concurrency + per-host probe via `dial`+`System`), a thin app `SnmpSweep` method that reuses the single-scan `scanContext()`/`ScanStop()` infra and streams `snmp:host` events, and an `SnmpSweep` Svelte tool that shows a live-filling responders table.

**Tech Stack:** Go 1.23 + Wails v2, Svelte/TS. Reuses `snmp.Dial`/`snmp.System`, the app `scanContext`/`ScanStop`/`scanEmit`, and the `scan` sweep pattern.

**Spec:** docs/superpowers/specs/2026-11-10-meridian-snmp-sweep-design.md

## Global Constraints

- SNMPv2c, read-only. Community masked on input, never shown back, never in a result row/event/error — scrub the one returned error with `scrub(err.Error(), community)`.
- Bounded: reject a CIDR whose host count exceeds the sweep cap (65536); a non-responder costs one timeout, not a hang; ctx cancel stops launching promptly.
- UI never blocks on the scan. Offline — no CDN; pure engine under `internal/`.
- Fixed defaults: 1 s timeout, 64 concurrency.

## Review Focus

- **Over-broad CIDR** → a "range too large" error, no fan-out. (Task 1 test.)
- **Invalid CIDR** → a clear error, at both engine and app layers. (Task 1 + Task 2 tests.)
- **Community never in a result or error.** (Task 1 asserts no community in SweepHost JSON; app scrubs the returned error.)
- **Non-responders excluded from the table but counted as scanned.** (Task 1: `onResult` fires per probed host; the return is responders only, sorted by IP.)
- **Network/broadcast skipped** for an IPv4 prefix < /31. (Task 1 scanned-count test.)

---

### Task 1: `snmp.Sweep` engine

**Files:**
- Create: `internal/snmp/sweep.go`
- Test: `internal/snmp/sweep_test.go`

**Interfaces:**
- Consumes: `Dial`, `System(c) (SysInfo, error)` (SysInfo.Name/Descr/UptimeTicks), the `Client` interface — all in package `snmp`.
- Produces:
  - `type SweepHost struct { IP string \`json:"ip"\`; Responded bool \`json:"responded"\`; Name string \`json:"name"\`; Descr string \`json:"descr"\`; UptimeTicks uint32 \`json:"uptimeTicks"\` }`
  - `var sweepDial = Dial` (test seam)
  - `func Sweep(ctx context.Context, cidr, community string, timeout time.Duration, concurrency int, onResult func(SweepHost)) ([]SweepHost, error)`

- [ ] **Step 1: Write the failing tests** in `internal/snmp/sweep_test.go`. Define a `sweepFake` implementing `Client` whose `Get` returns the sys OIDs' values when `ok`, else a transport error; a `setSweepDial(t, fn)` helper (save/restore `sweepDial`). Model: dial returns a `sweepFake`, responder IPs get `ok:true` with a name/descr/uptime, others `ok:false`.

```go
func TestSweepFindsResponders(t *testing.T) // /29, responders at .1 and .3; returns exactly those two sorted by IP with Name/Descr/UptimeTicks; onResult fired 6 times (hosts of a /29 minus net+bcast)
func TestSweepRejectsHugeCIDR(t *testing.T)  // "10.0.0.0/8" → error mentioning the cap; no dial
func TestSweepInvalidCIDR(t *testing.T)      // "nope" → error
func TestSweepNoCommunityInResult(t *testing.T) // every SweepHost JSON-marshals without the community
func TestSweepCancel(t *testing.T)           // a cancelled ctx returns promptly (few/no probes)
```

- [ ] **Step 2: Run the tests, verify they fail** — `go test ./internal/snmp/ -run TestSweep` → FAIL (undefined Sweep).

- [ ] **Step 3: Implement** `SweepHost`, `sweepDial`, `Sweep` in `internal/snmp/sweep.go`. Parse + `.Masked()` the CIDR; compute host bits and reject when `> cap` (const `maxSweepHosts = 65536`); enumerate hosts with a local helper (skip network+broadcast for IPv4 `< /31`, like `scan.enumerate`); `sem`/`wg`/`mu` bounded-concurrency loop (break launching on `ctx.Done()`); per host `probe(ip, community, timeout)` → `sweepDial` then `System(c)` (dial may use `Retries 0` for speed — builder call), a successful `System` → `SweepHost{Responded:true, Name, Descr, UptimeTicks}`, else `Responded:false`; `onResult(h)` per probed host; append responders to results; after `wg.Wait()` sort responders by `netip` and return.

- [ ] **Step 4: Run the tests, verify they pass** — `go test -race ./internal/snmp/ -run TestSweep` → PASS. Then full `go test ./...` green.

- [ ] **Step 5: Commit** — `feat(snmp): CIDR SNMP sweep (find responders)`.

---

### Task 2: App method + SNMP sweep tool

**Files:**
- Create: `internal/app/snmpsweep_methods.go`, `internal/app/snmpsweep_methods_test.go`
- Create: `frontend/src/lib/tools/SnmpSweep.svelte`
- Modify: `frontend/src/lib/Workbench.svelte`
- Regenerate: `frontend/wailsjs` bindings

**Interfaces:**
- Consumes: `snmp.Sweep`, `snmp.SweepHost` (Task 1); the existing `scanContext()`, `ScanStop()`, `scanEmit`, `scrub`. Frontend: the Wails runtime `EventsOn`/`EventsOff` for the global `snmp:host` event (not a per-stream id), the regenerated `SnmpSweep`/`ScanStop` bindings.
- Produces: `func (a *App) SnmpSweep(cidr, community string) ([]snmp.SweepHost, error)`; a tool registered as `{ id: "snmpsweep", name: "SNMP sweep", group: "SNMP & monitoring", ready: true }`.

- [ ] **Step 1: Write the failing app test** `internal/app/snmpsweep_methods_test.go`:
```go
func TestSnmpSweepInvalidCIDR(t *testing.T) // New().SnmpSweep("nonsense", "public") → error, and the error contains no "public"
```
- [ ] **Step 2: Run it, verify it fails** — `go test ./internal/app/ -run TestSnmpSweep` → FAIL (undefined SnmpSweep).

- [ ] **Step 3: Implement** `SnmpSweep` in `internal/app/snmpsweep_methods.go`: `ctx := a.scanContext()`; `res, err := snmp.Sweep(ctx, cidr, community, time.Second, 64, func(h snmp.SweepHost){ a.scanEmit("snmp:host", h) })`; on error return `errors.New(scrub(err.Error(), community))`; else return `res`. (`ScanStop` already exists and cancels it.)

- [ ] **Step 4: Run it, verify it passes** — `go test ./internal/app/ -run TestSnmpSweep` → PASS. Full `go test ./...` green.

- [ ] **Step 5: Build `SnmpSweep.svelte`.** CIDR + masked community inputs, Scan / Stop. On Scan: reset state; compute `total` host count from the CIDR (parse mask; skip net+bcast for IPv4 < /31); `EventsOn("snmp:host", h => { scanned++; if (h.responded) responded++ })` BEFORE calling the backend; `running = true`; `rows = await SnmpSweep(cidr, community)` (authoritative sorted responders) then `running = false` and `EventsOff`; on throw show an error banner and stop. Stop button: `ScanStop()` + `EventsOff` + `running = false`. Render a progress bar (`scanned/total`), the "scanned N · M responded" line, and the responders table (IP, Name, Description, Uptime). Uptime helper: TimeTicks are centiseconds → `Nd Nh`/`Nh Nm`. `onDestroy`: `ScanStop()` + `EventsOff`.

- [ ] **Step 6: Wire Workbench** — import `SnmpSweep`, add the tools[] entry in "SNMP & monitoring", add the `{:else if active === "snmpsweep"}<SnmpSweep />` branch.

- [ ] **Step 7: Regenerate bindings** — `$(go env GOPATH)/bin/wails generate module`, then `git checkout -- frontend/wailsjs/runtime`. Confirm `SnmpSweep` in `App.d.ts` and `snmp.SweepHost` in `models.ts`.

- [ ] **Step 8: Build** — `cd frontend && npm run build` → green.

- [ ] **Step 9: Manual check** (dev server + Playwright): stub `window.go.app.App.SnmpSweep` to `EventsEmit('snmp:host', …)` for ~8 hosts (2 responders) then return the 2 responders; stub `ScanStop`. Run a scan; confirm the progress line counts scanned/responded, the table shows the 2 responders with formatted uptime, and Stop/empty states behave. Confirm another SNMP tool still renders (regression).

- [ ] **Step 10: Commit** — `feat(app,ui): SNMP sweep tool`.

---

## Self-Review notes

- Spec coverage: Task 1 = the engine (enumerate, cap, probe, sort, community-safe); Task 2 = app streaming method + the tool (live fill, progress, Stop, uptime). Non-goals (SET, v3, MIB walk, vendor lookup, persistence) are out.
- Review Focus: each line has an owning test (Task 1/2) or the Task 2 manual check.
- Types: `SweepHost` JSON tags (Task 1) match the fields the tool reads (`ip/responded/name/descr/uptimeTicks`); `SnmpSweep` returns `[]snmp.SweepHost` consumed by the tool.
