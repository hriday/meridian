# Firewall Config Analyzer (FortiGate) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let an engineer drop/paste a FortiGate config export and, entirely offline, see its VLANs, "what talks to what" (VLAN→VLAN · port/proto · allow/deny), headline counts, and a flow diagram — per VDOM.

**Architecture:** A pure engine package `internal/fwconfig`: a generic FortiGate **block parser** (config/edit/set/next/end, multi-VDOM) → a neutral tree; a FortiGate **interpreter** that turns the tree into a per-VDOM `Analysis` (VLANs, zones, addresses, services, policies); a **flow deriver** that turns policies into VLAN-pair flows + counts. A thin `internal/app` method `AnalyzeFirewall` and a CLI adapter. A `FirewallAnalyzer.svelte` Workbench tool in a new "Config insight" group (load/paste + VDOM picker + tiles + VLAN table + flow table + drill-down), with the diagram last.

**Tech Stack:** Go 1.23 (`strings`, `bufio`, `net/netip`), Wails v2 + Svelte. Tests use `stretchr/testify` against crafted config fixtures. No network, no live device.

**Spec:** `docs/superpowers/specs/2026-11-10-meridian-firewall-analyzer-design.md`

## Global Constraints

- **Offline & pure:** `internal/fwconfig` does no I/O and never connects to a device. Config text comes in as a string.
- **Secrets:** encrypted values (`ENC …`, `set passwd/psksecret …`) are parsed structurally but **never** placed in any `Analysis` field returned to the UI/CLI.
- **Multi-VDOM:** a config with `config vdom` yields one `Analysis` per VDOM; a single-VDOM config yields exactly one (VDOM name `""`).
- **Flow fidelity:** flows are a VLAN-level reading of the ruleset — derived from policy src/dst **interface**, not by matching addresses to subnets. Disabled policies are skipped; policies with address/service **negation** or `internet-service` are recorded as a Note, not drawn as a normal flow. Accept and deny for a pair are separate flows. Implicit deny and rule ordering are out of scope.
- CLI + JSON output for the tool (per design §4.9). Commit attribution: `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`; PR body ends with the Generated-with line.

## Review Focus

- **Multi-VDOM split** — `config global` sections and each `config vdom`/`edit <name>` scope are attributed to the right VDOM; a single-VDOM config still yields one Analysis. (Task 2 test.)
- **Interface/zone/`any` → VLAN mapping** — a policy whose srcintf/dstintf is a zone expands to that zone's VLANs; `any` or a WAN-role interface becomes "Internet (any)"; a non-VLAN interface becomes a node named by the interface. (Task 3 test.)
- **Disabled / negated policies** — a `set status disable` policy produces no flow; a negated/`internet-service` policy produces a Note, not a silent allow. (Task 3 test.)
- **Quoted / spaced names** — object names with quotes, spaces, and escaped characters parse intact. (Task 1 test.)
- **Empty / minimal config** — a config with no VLANs or no policies yields an empty-but-valid Analysis (zeroed counts), never a crash or nil deref. (Task 2 & 3 tests.)
- **Secrets never surface** — an `ENC`/password line never appears in any Analysis field. (Task 2 test.)

---

### Task 1: FortiGate block parser

**Files:**
- Create: `internal/fwconfig/parser.go`
- Test: `internal/fwconfig/parser_test.go`

**Interfaces:**
- Produces:
  - `type Block struct { Kind, Name string; Settings map[string][]string; Children []*Block }` — `Kind` is `"config"` or `"edit"`; `Name` is the section path (`"system interface"`) or edit key (`"port1.10"`); `Settings` maps a `set` key to its value tokens; `Children` are nested `config`/`edit` blocks.
  - `func parse(text string) (*Block, error)` — returns a synthetic root whose `Children` are the top-level `config` blocks. Tokenises lines: `config X` / `edit Y` push a block, `next` closes an edit, `end` closes a config, `set k v…` adds a setting (respecting double-quoted values with escaped quotes), `unset`/comments (`#…`) and blank lines ignored.
  - `func (b *Block) configs(path string) []*Block` — child `config` blocks matching a section path (searched recursively within the block).
  - `func (b *Block) edits() []*Block` — the `edit` children of a config block.
  - `func (b *Block) get(key string) (string, bool)` / `func (b *Block) getAll(key string) []string` — a setting's joined value / raw tokens.

- [ ] **Step 1: Write failing tests** `parser_test.go` (`package fwconfig`):
  - `TestParseNestedConfigEdit`: a `config system interface` with two `edit` blocks each with `set` lines → root has one config child with two edits, settings read back.
  - `TestParseQuotedValues`: `set comment "hi there \"x\""` and an edit name with spaces → value and name intact.
  - `TestParseMultiVdom`: a `config global` section plus `config vdom`/`edit root`/…/`edit dmz`/… → the vdom block holds two edits (`root`, `dmz`) each with their own config children.
  - `TestParseIgnoresCommentsAndHeader`: `#config-version=…` and `#comment` lines ignored; a trailing `end` imbalance returns an error (or is tolerated — pick one and assert).
- [ ] **Step 2: Run, verify fail.**
- [ ] **Step 3: Implement** `parser.go` (line scanner + block stack).
- [ ] **Step 4: Run, verify pass.**
- [ ] **Step 5: Commit** (`feat(fwconfig): FortiGate config block parser`).

### Task 2: Model + FortiGate interpretation (per VDOM)

**Files:**
- Create: `internal/fwconfig/model.go`, `internal/fwconfig/fortigate.go`
- Test: `internal/fwconfig/fortigate_test.go`

**Interfaces:**
- Consumes: Task 1's `Block` tree.
- Produces:
  - `type VLAN struct { ID int; Name, Interface, Subnet, Zone string }`
  - `type Zone struct { Name string; Interfaces []string }`
  - `type Address struct { Name, Kind, Value string }` (Kind: subnet/iprange/fqdn/geo; Value human-readable; secrets excluded)
  - `type Service struct { Name, Proto, Ports string }`
  - `type Policy struct { ID int; Name string; SrcIntf, DstIntf, SrcAddr, DstAddr, Services []string; Action string; Disabled, Negated bool }`
  - `type Analysis struct { VDOM string; VLANs []VLAN; Zones []Zone; Addresses []Address; Services []Service; Policies []Policy; Flows []Flow; Counts Counts; Notes []string }` (`Flows`/`Counts` filled in Task 3; init all slices non-nil)
  - `func interpret(root *Block) ([]Analysis, error)` — split into VDOM scopes (`config vdom` edits, else a single unnamed scope merged with `config global`), and for each scope read: `config system interface` (VLANs = interfaces with a `vlanid`; Subnet from `set ip`), `config system zone`, `config firewall address`/`addrgrp`, `config firewall service custom`/`group`, `config firewall policy`. Skip secret settings.

- [ ] **Step 1: Write failing tests** `fortigate_test.go`:
  - `TestInterpretVLANsAndZones`: interfaces with vlanids + a zone grouping two → VLAN fields + Zone membership.
  - `TestInterpretAddressesServicesPolicies`: address objects (subnet + range + fqdn), a custom service (tcp/443), and a policy (srcintf/dstintf/srcaddr/dstaddr/service/action) read correctly.
  - `TestInterpretMultiVdom`: two VDOMs → two Analyses, each with only its own objects; a single-VDOM config → one Analysis, VDOM `""`.
  - `TestInterpretSecretsExcluded`: a config with `set passwd ENC …` → no Analysis field contains the secret.
  - `TestInterpretEmptyConfig`: minimal/empty config → one Analysis, empty slices, no error.
- [ ] **Step 2–4:** fail → implement → pass.
- [ ] **Step 5: Commit** (`feat(fwconfig): model + FortiGate interpreter`).

### Task 3: Flow derivation + counts

**Files:**
- Create: `internal/fwconfig/flows.go`
- Test: `internal/fwconfig/flows_test.go`

**Interfaces:**
- Produces:
  - `type Flow struct { Src, Dst string; Services []string; Action string }` (Src/Dst display names: VLAN name, interface name, or "Internet (any)"; Action "allow"/"deny")
  - `type Counts struct { VLANs, Policies, Addresses, SubnetsIPs, Services, Zones int }`
  - `func derive(a *Analysis)` — mutates `a`: builds `Flows` from `Policies` (map each srcintf/dstintf to VLAN(s) via interface→VLAN and zone→interfaces; `any`/WAN-role → "Internet (any)"; non-VLAN interface → its own name), one flow per (Src, Dst, Action) with Services unioned; skips `Disabled`; records a Note for each `Negated` policy; fills `Counts` (SubnetsIPs = distinct resolved subnet/host values across `Addresses` + VLAN subnets, excluding `all`/unresolvable FQDN).
  - `interpret` (Task 2) calls `derive` on each Analysis before returning.

- [ ] **Step 1: Write failing tests** `flows_test.go` (build Analyses via `interpret` on fixtures, then assert):
  - `TestDeriveAllowAndDeny`: an accept and a deny policy between the same VLAN pair → two flows, actions "allow"/"deny".
  - `TestDeriveZoneExpansion`: a policy srcintf = a zone of two VLANs → flows from both.
  - `TestDeriveAnyAndWanToInternet`: dstintf `any` and a `set role wan` interface → Dst "Internet (any)".
  - `TestDeriveSkipsDisabled`: a `status disable` policy → no flow.
  - `TestDeriveNegatedPolicyNote`: a `srcaddr-negate enable` policy → a Note, no normal flow.
  - `TestDeriveCounts`: a fixture with known VLANs/policies/addresses/services/zones → exact Counts, SubnetsIPs excludes `all`.
- [ ] **Step 2–4:** fail → implement → pass.
- [ ] **Step 5: Commit** (`feat(fwconfig): VLAN flow derivation + counts`).

### Task 4: App method + CLI adapter

**Files:**
- Create: `internal/app/fwconfig_methods.go`, `internal/app/fwconfig_methods_test.go`
- Modify: the CLI command set (follow the existing `internal/cli` pattern); Test: a CLI test in that package.

**Interfaces:**
- Produces:
  - `func (a *App) AnalyzeFirewall(configText, vendor string) ([]fwconfig.Analysis, error)` — trims input; empty → clear error; `vendor` defaults to "fortigate" (only supported value in v1, others → error); returns the per-VDOM analyses.
  - A CLI subcommand reading a config from a file arg or stdin and printing `AnalyzeFirewall` as JSON.

- [ ] **Step 1: Write failing tests**:
  - `TestAnalyzeFirewallEmpty`: `""` → error mentioning no config.
  - `TestAnalyzeFirewallFortigate`: a small config → analyses with expected VLAN/flow counts.
  - `TestAnalyzeFirewallUnknownVendor`: `vendor="cisco"` → a clear "not yet supported" error.
  - CLI test: feed a config via the adapter, assert JSON contains the VLANs/flows (using a `bytes.Buffer`, matching the existing CLI test pattern).
- [ ] **Step 2–4:** fail → implement → pass.
- [ ] **Step 5: Commit** (`feat(app,cli): AnalyzeFirewall method + CLI`).

### Task 5: Workbench tool `FirewallAnalyzer.svelte` (no diagram yet)

**Files:**
- Create: `frontend/src/lib/tools/FirewallAnalyzer.svelte`
- Modify: `frontend/src/lib/Workbench.svelte` (import; a new "Config insight" group entry `{ id: "fwanalyze", name: "Firewall analyzer", group: "Config insight", ready: true }`; route)
- Regenerate: bindings (`AnalyzeFirewall` + the `fwconfig` models).

- [ ] **Step 1: Regenerate bindings** (`$(go env GOPATH)/bin/wails generate module`; `git checkout -- frontend/wailsjs/runtime`); verify `AnalyzeFirewall` in `App.d.ts` and the `fwconfig` types in `models.ts`.
- [ ] **Step 2: Write `FirewallAnalyzer.svelte`**:
  - A large paste area + a "Load file" affordance + Analyze button, calling `AnalyzeFirewall(text, "fortigate")`.
  - On result: if more than one VDOM, a **VDOM picker** (select) controlling which Analysis is shown.
  - For the selected Analysis: the **count tiles**, the **VLANs table** (id/name/interface/subnet/zone), the **"what talks to what" flow table** (Src → Dst · service pills · allow/deny, deny in red), any **Notes** (e.g. negated policies), and a line explaining the VLAN-level reading. Clicking a flow expands the underlying policies/addresses (drill-down).
  - Reuse the established visual language (cards, tiles, mono, good/warn/bad); phone-width safe; guard against null/empty arrays.
- [ ] **Step 3: Wire into `Workbench.svelte`** (import, group entry, route).
- [ ] **Step 4: Build** `cd frontend && npm run build`; verify green.
- [ ] **Step 5: Manual check** in the running app: paste a sample FortiGate config (a fixture), confirm tiles/VLANs/flows render, VDOM picker appears for a multi-VDOM sample, no console errors.
- [ ] **Step 6: Commit** (`feat(app,ui): Firewall analyzer tool (FortiGate)`).

### Task 6: Flow diagram

**Files:**
- Modify: `frontend/src/lib/tools/FirewallAnalyzer.svelte` (add the diagram section)

- [ ] **Step 1: Add an SVG flow diagram** below the flow table for the selected Analysis: VLANs (and "Internet (any)") as nodes, flows as directed edges — green for allow, red dashed for deny — with edge labels naming the service unambiguously (`tcp/22` / `SSH`, never bare `22`), arrowhead at the destination. Compute a simple readable layout (e.g. columns by zone or a ring); keep it legible in both themes and at phone width; degrade gracefully when there are many nodes (cap/scroll).
- [ ] **Step 2: Build** `npm run build`; verify green.
- [ ] **Step 3: Manual check** the diagram renders for the sample config and matches the flow table.
- [ ] **Step 4: Commit** (`feat(ui): firewall flow diagram`).

---

## Self-Review notes

- Spec coverage: offline parse → T1; model/VLANs/zones/addresses/services/policies + multi-VDOM + secrets-excluded → T2; what-talks-to-what + counts + any/WAN/zone/disabled/negation → T3; app+CLI → T4; UI (load, VDOM picker, tiles, tables, drill-down) → T5; diagram → T6.
- Review Focus → each maps to a named test (multi-VDOM split, interface/zone/any mapping, disabled/negated, quoted names, empty config, secrets-excluded).
- Proportion: signatures + test names; parser/interpreter/deriver bodies left to the implementer. No real device, no secrets in fixtures.
- Diagram is the last task so it can slip without blocking the table-based value (per design).
