# Local Network Diagnostic Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A deterministic (no-AI) "Local Network Diagnostic" recipe tool that runs an ordered probe sequence and reads one named verdict (+ cause + fix) off the result pattern, plus a "Run Local Network Diagnostic" affordance on every LAN tool's reachability failure.

**Architecture:** New pure engine package `internal/netdiag/` holds the verdict types, the pure `Classify`/`Describe` decision tree (the TDD core), orchestration (`Run`), a new passive IPv4 neighbor reader, and a per-OS tri-state OS-block check. A thin `internal/app` binding exposes it to the Svelte frontend; a CLI subcommand reuses the same engine; a Svelte tool renders steps + verdict and auto-runs from the existing `requestedTool` nav store that failing tools set.

**Tech Stack:** Go 1.26 (`github.com/hriday/meridian`), Wails v2, Svelte. Per-OS files via build tags; func-var + `Supported()` seams; inline string-literal test fixtures (repo has no golden/testdata convention).

**Spec:** `docs/superpowers/specs/2026-11-19-local-network-diagnostic-design.md`

## Global Constraints

- **No AI / LLM anywhere** — fixed decision tree only.
- **Runs unprivileged.** The core verdict must be reachable with no `sudo`; must not depend on the raw-socket `arpscan` sweep or the `internal/trace` prober (raw socket, needs root on Linux). "Own reachability" uses a plain TCP connect via `scan.ScanPorts` / `scan.NewICMPPinger` (genuinely unprivileged).
- **IPv4-only for v1.** An IPv6-only LAN (no routable IPv4) yields `INCONCLUSIVE`, never a false `DHCP_FAILED`.
- **Honesty:** `available==false` from the ICMP pinger means "couldn't test", never "host down". An errored/thin probe yields `INCONCLUSIVE`, never a confident guess.
- **Nielsen:** show each step's live status; name the concrete fix inline.
- **CI does not build the frontend** — frontend tasks verify with `cd frontend && npm run build` locally.
- Engine packages hold all logic; `internal/app` methods are thin delegates (repo rule).

## Review Focus

- **ICMP unavailable** (`available==false`) must not classify as host-down → **Task 1** test `TestClassify_ICMPUnavailableNotHostDown` (Observations with the gateway/peer "available=false" path still routes to the TCP-fallback-derived state, not `HOST_DOWN`).
- **Thin / stale peer sample** must not trigger isolation/block verdicts → **Task 1** test `TestClassify_UnconfidentPeerSampleIsInconclusive`; stale ARP entries filtered → **Task 2** test `TestNeighbors_DropsIncompleteAndFailed`.
- **Single dead host, healthy LAN** → `HOST_DOWN` not a network verdict → **Task 1** test `TestClassify_TargetDownButPeersReached`.
- **Captive portal vs internet-down** (`gateway ✓ / internet ✗`, intercepted vs clean) → **Task 1** tests `TestClassify_CaptivePortal` and `TestClassify_InternetDownLANFine`.
- **IPv6-only LAN** → `INCONCLUSIVE` not `DHCP_FAILED` → **Task 1** test `TestClassify_IPv6OnlyIsInconclusive`.

---

### Task 1: Verdict types + pure decision tree (`Classify` / `Describe`)

**Files:**
- Create: `internal/netdiag/netdiag.go`
- Test: `internal/netdiag/netdiag_test.go`

**Interfaces:**
- Consumes: nothing (pure).
- Produces:
  ```go
  type Verdict string
  const (
    VerdictDHCPFailed         Verdict = "dhcp_failed"
    VerdictGatewayUnreachable Verdict = "gateway_unreachable"
    VerdictCaptivePortal      Verdict = "captive_portal"
    VerdictInternetDown       Verdict = "internet_down"
    VerdictOSBlockingMeridian Verdict = "os_blocking_meridian"
    VerdictClientIsolation    Verdict = "client_isolation"
    VerdictLocalAccessBlocked Verdict = "local_access_blocked"
    VerdictHostDown           Verdict = "host_down"
    VerdictAllClear           Verdict = "all_clear"
    VerdictInconclusive       Verdict = "inconclusive"
  )
  type OSBlockState int
  const ( OSBlockUnknown OSBlockState = iota; OSBlockBlocked; OSBlockNotBlocked; OSBlockCouldNotDetermine )
  type StepResult struct { // Status: pass|fail|skip|error — JSON-tagged for the frontend
    Name   string `json:"name"`
    Status string `json:"status"`
    Detail string `json:"detail"`
  }
  type Observations struct {
    GOOS                                            string
    HasGateway, IsLinkLocal, IPv6Only               bool
    GatewayReachable                                bool
    InternetReachable, InternetIntercepted          bool
    PeerSampleConfident                             bool
    PeersTried, PeersReached                        int
    TargetGiven, TargetReached                      bool
    OSBlock                                         OSBlockState
    ProbeError                                      string // non-empty ⇒ a required probe errored
    Steps                                           []StepResult
  }
  type Report struct {
    Verdict            Verdict  `json:"verdict"`
    Statement          string   `json:"statement"`
    Causes             []string `json:"causes"`
    Fixes              []string `json:"fixes"`
    Steps              []StepResult `json:"steps"`
  }
  func Classify(o Observations) Verdict
  func Describe(o Observations, v Verdict) Report   // fills Statement/Causes/Fixes (per-OS for OS_BLOCKING/LOCAL_ACCESS_BLOCKED), attaches o.Steps
  ```

- [ ] **Step 1: Write failing tests** covering every verdict row and the Review-Focus cases. Table test over `Observations → Verdict`, one case per row of the spec's verdict table, plus: `TestClassify_IPv6OnlyIsInconclusive`, `TestClassify_ICMPUnavailableNotHostDown`, `TestClassify_UnconfidentPeerSampleIsInconclusive`, `TestClassify_CaptivePortal`, `TestClassify_InternetDownLANFine`, `TestClassify_TargetDownButPeersReached`, `TestClassify_LocalAccessBlockedWhenOSBlockCouldNotDetermine`, `TestClassify_OSBlockingWhenBlocked`, `TestClassify_ClientIsolationWhenProvenNotBlocked`. Plus `TestDescribe_OSBlockingCopyPerOS` asserting macOS text names "Local Network" settings and Linux `LOCAL_ACCESS_BLOCKED` lists both causes.

- [ ] **Step 2: Run tests, verify they fail** — `go test ./internal/netdiag/ -run TestClassify -v` → FAIL (undefined `Classify`).

- [ ] **Step 3: Implement `Classify` and `Describe`** in `internal/netdiag/netdiag.go` following the spec's verdict table ordering (DHCP → gateway → captive/internet-down → peers[block/isolation/local-blocked] → host-down → all-clear → inconclusive), the IPv6-only guard, and the peer-confidence gate. `Describe` holds the per-verdict/per-OS copy strings.

- [ ] **Step 4: Run tests, verify pass** — `go test ./internal/netdiag/ -v` → PASS.

- [ ] **Step 5: Commit** — `feat(netdiag): verdict types and pure decision tree`.

---

### Task 2: Passive IPv4 neighbor-table reader (per-OS)

**Files:**
- Create: `internal/netdiag/neigh.go` (OS-agnostic wrapper + `Supported()`), `internal/netdiag/neigh_darwin.go`, `internal/netdiag/neigh_linux.go`, `internal/netdiag/neigh_windows.go`, `internal/netdiag/neigh_other.go`
- Test: `internal/netdiag/neigh_test.go`

**Interfaces:**
- Consumes: Task 1 package.
- Produces:
  ```go
  type Neighbor struct { IP string; MAC string; State string } // State normalized: "reachable"|"stale"|"incomplete"|"failed"
  func Neighbors(ctx context.Context) ([]Neighbor, error) // excludes incomplete/failed; passive, no raw socket
  func NeighborsSupported() bool
  // seam for tests:
  var neighborsRaw func(ctx context.Context) (string, error) // per-OS sets this in init(); parse is OS-agnostic where formats allow, else per-OS parse fn var
  ```
  Per-OS source: darwin `arp -an`, linux `/proc/net/arp` (or `ip neigh`), windows `arp -a`. Parsing filters to reachable/stale states.

- [ ] **Step 1: Write failing test** `TestNeighbors_DropsIncompleteAndFailed`: feed an inline fixture string (sample `arp -an` / `/proc/net/arp` output incl. an `(incomplete)` / `0x0` FAILED row) through the parse seam; assert the incomplete/failed rows are dropped and good rows return with normalized state. Add `TestNeighbors_ParsesIPAndMAC`.

- [ ] **Step 2: Run test, verify fail** — `go test ./internal/netdiag/ -run TestNeighbors -v` → FAIL.

- [ ] **Step 3: Implement** the OS-agnostic parse + wrapper and the per-OS `neighborsRaw` providers (func-var set in each file's `init()`), with `NeighborsSupported()` false on `neigh_other.go`.

- [ ] **Step 4: Run test, verify pass** — `go test ./internal/netdiag/ -run TestNeighbors -v` → PASS.

- [ ] **Step 5: Commit** — `feat(netdiag): passive IPv4 neighbor-table reader`.

---

### Task 3: OS-block check (per-OS, tri-state)

**Files:**
- Create: `internal/netdiag/osblock.go` (wrapper + seam), `internal/netdiag/osblock_darwin.go`, `_linux.go`, `_windows.go`, `_other.go`
- Test: `internal/netdiag/osblock_test.go`

**Interfaces:**
- Consumes: Task 1 (`OSBlockState`).
- Produces:
  ```go
  // ownReach/baselineReach are injected so the comparison logic is testable
  // without touching the host or shelling out.
  type reachFn func(ctx context.Context, peer string, port int) (ok bool, ran bool)
  func osBlockCheck(ctx context.Context, peers []string, ownReach, baselineReach reachFn) (OSBlockState, string)
  func OSBlockSupported() bool
  ```
  macOS baseline = Apple `/sbin/ping`/`nc` (TCC-exempt); own = plain TCP connect. Windows adds network-category/firewall corroboration. Linux `baselineReach` ran=false (no per-app gating) ⇒ contributes `CouldNotDetermine`; best-effort `ufw`/`firewalld` status fills the detail string.

- [ ] **Step 1: Write failing tests** for the pure comparison `osBlockCheck` with injected `reachFn`s: `TestOSBlock_BaselineReachesOwnCannot_Blocked` (baseline ok, own fail ⇒ `OSBlockBlocked`), `TestOSBlock_BothFail_NotBlocked` (both fail, both ran ⇒ `OSBlockNotBlocked`), `TestOSBlock_BaselineCouldNotRun_CouldNotDetermine` (baseline ran=false ⇒ `OSBlockCouldNotDetermine`).

- [ ] **Step 2: Run tests, verify fail** — `go test ./internal/netdiag/ -run TestOSBlock -v` → FAIL.

- [ ] **Step 3: Implement** `osBlockCheck` (pure comparison) and the per-OS real `ownReach`/`baselineReach` providers + detail enrichment behind build tags; `OSBlockSupported()` false on `_other.go`. Note in a code comment: the real per-OS probes are environment-dependent and are verified manually, not by unit tests.

- [ ] **Step 4: Run tests, verify pass** — `go test ./internal/netdiag/ -run TestOSBlock -v` → PASS.

- [ ] **Step 5: Commit** — `feat(netdiag): tri-state per-OS OS-block check`.

---

### Task 4: Probe orchestration (`Run`)

**Files:**
- Create: `internal/netdiag/run.go`
- Test: `internal/netdiag/run_test.go`

**Interfaces:**
- Consumes: Tasks 1–3; `internal/myip` (`myip.Local()`), `internal/netif` (`netif.List()`), `internal/netroute` (`netroute.DefaultGateway()`), `internal/scan` (`scan.NewICMPPinger()`, `scan.ScanPorts`).
- Produces:
  ```go
  type Options struct { Target string }
  func Run(ctx context.Context, opts Options, onStep func(StepResult)) (Report, error)
  // internal probe seams (package-level func vars) so run_test.go fakes them:
  var probeIdentity   func() (ip, gw, iface, prefix string, linkLocal, ipv6Only, hasGW bool, err error)
  var probeReach      func(ctx context.Context, host string) (reached, available bool)
  var sampleInternet  func(ctx context.Context) (reached, intercepted bool)
  ```
  `Run` executes steps in order, emits each `StepResult` via `onStep`, builds `Observations` (seeds peers: target → `Neighbors()` minus gateway → bounded in-subnet last resort; applies the confidence threshold; sets `ProbeError` on a required-probe failure), then `return Describe(obs, Classify(obs)), nil`.

- [ ] **Step 1: Write failing tests** overriding the probe seams: `TestRun_HotspotIsolationPath` (identity ok, gateway+internet reachable, all sampled peers fail, OS-block NotBlocked ⇒ `CLIENT_ISOLATION`), `TestRun_MacOSBlockPath` (same but OS-block Blocked ⇒ `OS_BLOCKING_MERIDIAN`), `TestRun_GatewayExcludedFromPeers` (only reachable "peer" is the gateway ⇒ peer sample unconfident ⇒ not a false ALL_CLEAR), `TestRun_StreamsStepsInOrder` (asserts `onStep` names/order).

- [ ] **Step 2: Run tests, verify fail** — `go test ./internal/netdiag/ -run TestRun -v` → FAIL.

- [ ] **Step 3: Implement `Run`** wiring the real probes through the seams (identity via `myip`/`netif`, gateway/peer reach via `scan` ICMP+TCP fallback per spec step 2/4, internet via fixed anycast TCP, neighbors via Task 2, OS-block via Task 3), peer seeding with gateway excluded and confidence threshold applied.

- [ ] **Step 4: Run tests, verify pass** — `go test ./internal/netdiag/ -v` (whole package) → PASS.

- [ ] **Step 5: Commit** — `feat(netdiag): probe orchestration`.

---

### Task 5: Wails binding

**Files:**
- Create: `internal/app/netdiag_methods.go`, `internal/app/netdiag_methods_test.go`
- Modify: regenerate `frontend/wailsjs/go/app/App.js` + `App.d.ts` (via `wails generate module` or the repo's binding-gen step)

**Interfaces:**
- Consumes: Task 4 (`netdiag.Run`).
- Produces: `func (a *App) NetworkDiagnostic(target string) (netdiag.Report, error)` — clamps input, builds a bounded timeout ctx, delegates to `netdiag.Run` (onStep nil for v1; full `Report.Steps` returned), returns the engine struct.

- [ ] **Step 1: Write failing test** `TestNetworkDiagnostic_DelegatesToEngine` — override the package's `netdiag.Run` seam (or inject) to return a known Report; assert the method returns it and applies the timeout/clamp. (Mirror `internal/app/ping_methods_test.go`.)

- [ ] **Step 2: Run test, verify fail** — `go test ./internal/app/ -run TestNetworkDiagnostic -v` → FAIL.

- [ ] **Step 3: Implement** the method; regenerate the TS bindings so `NetworkDiagnostic` appears in `frontend/wailsjs/go/app/App.d.ts`.

- [ ] **Step 4: Run test, verify pass** — `go test ./internal/app/ -run TestNetworkDiagnostic -v` → PASS; confirm `App.d.ts` contains `NetworkDiagnostic`.

- [ ] **Step 5: Commit** — `feat(app): bind NetworkDiagnostic`.

---

### Task 6: CLI subcommand `netdiag`

**Files:**
- Create: `internal/cli/netdiag.go`, `internal/cli/netdiag_test.go`
- Modify: `internal/cli/cli.go` (switch `case "netdiag"`), `internal/cli/catalog.go` (help entry under "Local network")

**Interfaces:**
- Consumes: Task 4 (`netdiag.Run`).
- Produces: `func runNetDiag(args []string, stdout, stderr io.Writer) int` — `flag.NewFlagSet`, `--json`, optional positional host; runs `netdiag.Run`; `writeJSON` or a human `printNetDiag` using the shared `reportRow` helper; exit 0 (all_clear/host_down informational) / 1 (problem verdict) / 2 (usage).

- [ ] **Step 1: Write failing tests** `TestRunNetDiag_JSON` (feeds a faked `netdiag.Run` via seam, `--json`, asserts the verdict appears in JSON) and `TestRunNetDiag_HumanLists Fix` (human output contains the fix line). Mirror `internal/cli/printer_test.go`.

- [ ] **Step 2: Run tests, verify fail** — `go test ./internal/cli/ -run TestRunNetDiag -v` → FAIL.

- [ ] **Step 3: Implement** `runNetDiag` + `printNetDiag`; add the `cli.go` switch case and the `catalog.go` entry (`{name:"netdiag", args:"[host]", desc:"Diagnose why local-network resources are unreachable"}` under "Local network").

- [ ] **Step 4: Run tests, verify pass** — `go test ./internal/cli/ -v` → PASS; `go run ./cmd/meridian-cli netdiag --help` shows the command.

- [ ] **Step 5: Commit** — `feat(cli): netdiag subcommand`.

---

### Task 7: Frontend tool component + registry

**Files:**
- Create: `frontend/src/lib/tools/NetDiagnostic.svelte`
- Modify: `frontend/src/lib/Workbench.svelte` (3 edits: import, `tools[]` registry entry, render branch)

**Interfaces:**
- Consumes: Task 5 binding `NetworkDiagnostic` from `../../../wailsjs/go/app/App.js`; the existing `requestedTool` store in `frontend/src/lib/nav.ts`.
- Produces: tool id `"netdiag"`, group `"Local network"`. Component: `host` input + "Run" button; `async run()` calls `NetworkDiagnostic(host)` in try/catch/finally; renders the ordered `res.steps` (pass/fail/skip badges) and a prominent verdict card (`res.verdict` → statement · causes · fixes). On mount, if `$requestedTool.tool === "netdiag"`, seed `host = $requestedTool.value` and auto-run. Inline `<style>` using shared CSS vars.

- [ ] **Step 1: Implement** the component and the three `Workbench.svelte` edits (registry `{ id:"netdiag", name:"Local network diagnostic", group:"Local network", ready:true, kw:"diagnostic troubleshoot reachability isolation hotspot guest wifi firewall local network privacy gateway dhcp captive portal" }`; render `{:else if active==="netdiag"}<NetDiagnostic />`). *(No Svelte unit-test harness in repo — verification is build + manual, per the frontend constraint.)*

- [ ] **Step 2: Build-check** — `cd frontend && npm run build` → succeeds with no errors; grep the build output/registry to confirm the tool compiles.

- [ ] **Step 3: Manual smoke** — `wails dev` (or run the built app), open the tool, run it against a host, confirm steps + verdict render and that the sidebar entry/search finds it. Record the result.

- [ ] **Step 4: Commit** — `feat(frontend): Local Network Diagnostic tool`.

---

### Task 8: Failure-integration buttons on LAN tools

**Files:**
- Modify: `frontend/src/lib/tools/PrinterStatus.svelte`, `Ping.svelte`, `PortScan.svelte`, `PingSweep.svelte`, `Discover.svelte`, `ArpScan.svelte`, and the SNMP tool component (names per the registry).

**Interfaces:**
- Consumes: `requestedTool` from `nav.ts`.
- Produces: in each tool, on a **reachability/timeout** failure (not auth/parse errors), a "Run Local Network Diagnostic" button that calls `requestedTool.set({ tool: "netdiag", value: <the failed host> })`. Each tool defines which of its result/error shapes counts as a reachability failure.

- [ ] **Step 1: Implement** the button in each listed component, gated on that tool's reachability-failure condition (e.g. PrinterStatus: `res && !res.reachable` or the error path; SNMP: timeout only, never a wrong-community/auth error).

- [ ] **Step 2: Build-check** — `cd frontend && npm run build` → succeeds.

- [ ] **Step 3: Manual smoke** — trigger a reachability failure in at least the printer tool against an unreachable host; click the button; confirm it switches to `netdiag` pre-seeded and auto-runs. Record the result.

- [ ] **Step 4: Commit** — `feat(frontend): offer the diagnostic on LAN-tool failures`.

---

### Task 9: README + screenshots (new-tool checklist)

**Files:**
- Modify: `README.md` (tool write-up in the tools list + a mention in the Local-network section), and the screenshots asset per the repo's convention.

**Interfaces:** none.

- [ ] **Step 1: Write** the README entry describing the diagnostic (what it concludes, the failure-button hand-off) consistent with the existing tool write-ups; add/refresh the screenshot.

- [ ] **Step 2: Verify** — `go build ./...` and `cd frontend && npm run build` still pass; README renders (no broken links).

- [ ] **Step 3: Commit** — `docs: document the Local Network Diagnostic`.

---

## Notes for the executor

- Run the **whole** `go test ./...` before finishing — not just `./internal/netdiag/...` — to catch any cross-package breakage from the binding/CLI edits.
- The per-OS `osblock`/`neigh` real providers can only be unit-tested at the pure-logic seam; their live behavior is verified manually on each OS (macOS already corroborated by the printer case). Say so in the final report; do not claim the OS branches are proven by unit tests.
- This whole feature is one branch (`local-network-diagnostic`, already cut from `master` with the spec committed) → finish via PR, no local merge ([[integration-always-pr]]).
