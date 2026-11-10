# Device health (multi-device CPU/memory monitor) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A NOC-style tool that watches several devices at once over SNMPv2c and shows each one's CPU and memory, flagging the box that's hot or unreachable.

**Architecture:** A per-device backend poller (one goroutine per device, reusing the shipped `monitor` lifecycle) emits a `DeviceSnapshot` on `metrics:<id>`; a `DeviceHealth` Svelte tool owns a client-side list of devices, each with its own subscription + sparkline ring buffer, and renders a row grid with threshold colouring.

**Tech Stack:** Go 1.23 + Wails v2, Svelte/TypeScript. Reuses `internal/snmp` (`System`, `ResourceUsage`), the app `monitor` lifecycle + `dial`/`scrub`/`snmpTimeout` seams, and `frontend/src/lib/metrics/metricStream.ts`.

**Spec:** docs/superpowers/specs/2026-11-10-meridian-device-health-design.md

## Global Constraints

- SNMPv2c only. Community string masked on input, never displayed back, never in a snapshot/error/event/log — scrub every error path with `scrub(err.Error(), community)`.
- The UI never blocks on polling; one device's failure never stalls the others.
- Offline — no CDN, reuse bundled deps. Pure engine stays under `internal/`; the app layer only binds.
- Reachability probe is `snmp.System(c)` (it returns the transport error and the sysName); `snmp.ResourceUsage(c)` is best-effort and returns nil error even for a dead device, so it is NOT a reachability signal.
- Fixed thresholds (frontend): CPU warn 75 / crit 90; memory warn 80 / crit 90. A metric ≥ warn → amber, ≥ crit → red; the worse of the two sets the row tint + state word.
- Device poll interval 5 s (`deviceInterval` seam).

## Review Focus

- **Community redaction:** community never in any snapshot or error. (Task 1 tests.)
- **Unreachable vs no-OID:** `System` error → `unreachable`/`error` snapshot; `System` ok but `ResourceUsage` returns `HasCPU/HasMem=false` → an `ok` snapshot with n/a metrics, NOT an error. (Task 1 tests, both cases.)
- **First-fail vs later-fail:** first dial/probe failure → one `error` snapshot then the goroutine exits; a later probe failure → `unreachable` and keep polling. (Task 1 tests.)
- **Stop/no-leak:** `StopDeviceMonitor` blocks until the goroutine exits; removing one device doesn't touch another's stream. (Task 1 test + Task 2 manual check.)
- **Per-metric n/a + unreachable rendering:** a row greys + flags on `unreachable`/`error`; a metric shows n/a when its `has*` is false. (Task 2 manual check.)

---

### Task 1: Backend device-health monitor

**Files:**
- Create: `internal/app/devicemon_methods.go`
- Test: `internal/app/devicemon_methods_test.go`

**Interfaces:**
- Consumes: `snmp.System(c) (SysInfo, error)` (SysInfo.Name), `snmp.ResourceUsage(c) (Resources, error)` (CPUPct, MemPct, HasCPU, HasMem); the existing seams `dial`, `scrub`, `snmpTimeout`, `emitEvent`; the `monitor{cancel,done,baseline}` struct + `registerMonitor`/`stopMonitor`. All already in package `app`.
- Produces:
  - `type DeviceSnapshot struct { Name string \`json:"name"\`; CPUPct float64 \`json:"cpuPct"\`; MemPct float64 \`json:"memPct"\`; HasCPU bool \`json:"hasCpu"\`; HasMem bool \`json:"hasMem"\`; State string \`json:"state"\`; Note string \`json:"note"\` }` — `State` ∈ `"ok" | "unreachable" | "error"`.
  - `var deviceInterval = 5 * time.Second`
  - `func (a *App) StartDeviceMonitor(streamID, device, community string) error`
  - `func (a *App) StopDeviceMonitor(streamID string)`

- [ ] **Step 1: Write the failing tests** in `internal/app/devicemon_methods_test.go`. Reuse the sibling file's `captureSnaps(t)`, `setDial(t, client, err)` helpers (same package). Add a `fakeDev` implementing `snmp.Client` whose `Get` answers the System OIDs (sysDescr/sysName/sysUpTime `.0`) with a name and answers `hrStorage*` for memory, and whose `Walk` answers `hrProcessorLoad` (CPU) and `hrStorageType`; add a `failSystem bool` and a `failAfter int` to script transport failures. Set `deviceInterval = time.Millisecond` in each test (save/restore).

```go
// ok stream: name + cpu + mem present
func TestDeviceMonitorStreams(t *testing.T) // start → eventually a DeviceSnapshot{State:"ok", CPUPct>0, Name:"core-sw-1"}; stop; got()[0].name == "metrics:s1"
func TestDeviceMonitorScrubsCommunity(t *testing.T) // dial err contains secret → ListInterfaces-style: StartDeviceMonitor emits an "error" snapshot whose Note has no secret
func TestDeviceMonitorSnapshotNoCommunity(t *testing.T) // every emitted snapshot JSON-marshals without the secret
func TestDeviceMonitorMissingResourcesNA(t *testing.T) // System ok, ResourceUsage Has*=false → snapshot State "ok", HasCPU==false, HasMem==false (NOT an error)
func TestDeviceMonitorFirstFailError(t *testing.T) // failSystem from start → one snapshot State "error"; goroutine exits (no further snapshots after Stop)
func TestDeviceMonitorLaterUnreachable(t *testing.T) // ok once then failAfter → a snapshot State "unreachable"; stream continues
func TestDeviceMonitorStopNoLeak(t *testing.T) // after Stop, snapshot count stops growing
```

- [ ] **Step 2: Run the tests, verify they fail** — `go test ./internal/app/ -run TestDeviceMonitor` → FAIL (undefined StartDeviceMonitor).

- [ ] **Step 3: Implement** `DeviceSnapshot`, `deviceInterval`, `StartDeviceMonitor`, `StopDeviceMonitor` in `internal/app/devicemon_methods.go`. Pattern mirrors `snmpmon_methods.go`'s `StartIfMonitor`: trim device (empty → error); `registerMonitor`; goroutine `defer close(m.done)` → `dial(device, community, snmpTimeout)` (err → emit `DeviceSnapshot{State:"error", Note:"can't reach the device: "+scrub(err.Error(), community)}`, return); then loop — `snmp.System(c)` is the reachability probe (first err → `error` snapshot + return; later err → `unreachable` snapshot + continue), on success `snmp.ResourceUsage(c)` (best-effort) and emit `DeviceSnapshot{Name: sys.Name, CPUPct, MemPct, HasCPU, HasMem, State:"ok"}`; `select` on `ctx.Done()` / `time.After(deviceInterval)`. `monitor.baseline` = a no-op `func(){}` (no baseline this slice). `StopDeviceMonitor` delegates to `a.stopMonitor`.

- [ ] **Step 4: Run the tests, verify they pass** — `go test -race ./internal/app/ -run TestDeviceMonitor` → PASS. Then full `go test ./...` green.

- [ ] **Step 5: Commit** — `feat(app): device-health SNMP monitor (CPU/memory, multi-device)`.

---

### Task 2: Device health frontend tool

**Files:**
- Create: `frontend/src/lib/tools/DeviceHealth.svelte`
- Create (if not kept inline): `frontend/src/lib/metrics/BarMeter.svelte`, `frontend/src/lib/metrics/Sparkline.svelte`
- Modify: `frontend/src/lib/Workbench.svelte` (import + tools[] entry + render branch)
- Regenerate: `frontend/wailsjs` bindings

**Interfaces:**
- Consumes: `StartDeviceMonitor(streamID, device, community)`, `StopDeviceMonitor(streamID)` (regenerated bindings); `DeviceSnapshot` JSON fields (`name, cpuPct, memPct, hasCpu, hasMem, state, note`); `subscribe`, `newStreamID` from `metricStream.ts`. Models the structure of `SnmpMonitor.svelte` (generate id + subscribe BEFORE Start; `onDestroy` teardown; error-state teardown).
- Produces: a tool registered in Workbench as `{ id: "devhealth", name: "Device health", group: "SNMP & monitoring", ready: true }`.

- [ ] **Step 1: Build the pieces.** A dumb `BarMeter` (props `pct`, `color`; track + fill) and a dumb `Sparkline` (props `values:number[]`, `color`; inline SVG polyline), or inline equivalents in the tool. Threshold helpers in the tool: `cpuState(v)` → good/warn/bad at 75/90, `memState(v)` → 80/90; a row's overall state = the worse metric, or `unreachable`/`error`/`nocpu` from the snapshot.

- [ ] **Step 2: Build `DeviceHealth.svelte`.** State: `let devices: {id, addr, community, streamID, unsub, snap, hist:number[]}[] = []`. Add-device form (device address + masked community + Add): push an entry, `newStreamID()`, `subscribe(id, s => onSnap(entry, s))`, then `await StartDeviceMonitor(streamID, addr, community)`; on throw, mark the entry errored and tear its subscription down. `onSnap` stores the snapshot and appends the busier of cpu/mem to `hist` (ring buffer ~12 samples ≈ 60 s). Remove-device control: `StopDeviceMonitor` + `unsub` + splice. A "worst-first" toggle sorts a derived view by max(cpu,mem)/severity (never mutates the source list). `onDestroy`: stop every device. Render the row grid from the mockup: name (snapshot `name` || addr) + addr, CPU `BarMeter` + % (n/a when `!hasCpu`), memory `BarMeter` + % (n/a when `!hasMem`), `Sparkline`, state dot + word; row tinted on warn/crit, greyed + flagged on `unreachable`/`error`. Community is held only in the entry object, never persisted.

- [ ] **Step 3: Wire Workbench** — import `DeviceHealth`, add the tools[] entry (above or below the SNMP interface monitor in the "SNMP & monitoring" group), add the `{:else if active === "devhealth"}<DeviceHealth />` branch.

- [ ] **Step 4: Regenerate bindings** — `$(go env GOPATH)/bin/wails generate module`, then `git checkout -- frontend/wailsjs/runtime`. Confirm `StartDeviceMonitor`/`StopDeviceMonitor` appear in `frontend/wailsjs/go/app/App.d.ts`.

- [ ] **Step 5: Build** — `cd frontend && npm run build` → green.

- [ ] **Step 6: Manual check** (dev server + Playwright, same technique as the interface monitor): stub `window.go.app.App.StartDeviceMonitor`/`StopDeviceMonitor` to capture the streamID, add two devices, inject a steady snapshot and a `cpuPct:93` snapshot and an `state:"unreachable"` snapshot through the live event stream; confirm the steady row is green, the hot row goes red + tinted, a `hasCpu:false` snapshot shows "n/a", the unreachable row greys + flags, and Remove tears a row down without touching the other. Confirm the SNMP interface monitor still renders (regression).

- [ ] **Step 7: Commit** — `feat(app,ui): Device health multi-device monitor tool`.

---

## Self-Review notes

- Spec coverage: Task 1 covers the backend poller + reachability + redaction + n/a-vs-error; Task 2 covers the grid, thresholds, add/remove, worst-first, n/a + unreachable rendering. Non-goals (baseline, persistence, per-core, v3) are out.
- Review Focus: every line has an owning test (Task 1) or manual check (Task 2) above.
- Types: `DeviceSnapshot` JSON tags (Task 1 Produces) match the fields Task 2 reads (`name/cpuPct/memPct/hasCpu/hasMem/state/note`).
