# Multi-vendor Firewall Analyzer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend `internal/fwconfig` to analyze Palo Alto PAN-OS, Cisco ASA, and Juniper SRX configs (in addition to FortiGate), filling the existing vendor-neutral `Analysis` model, and add a vendor selector to the analyzer UI.

**Architecture:** The `Analysis` model (`model.go`) and flow deriver (`flows.go` → `derive(&a)`) are already vendor-neutral. Each new vendor gets a self-contained `analyzeX(configText string) ([]Analysis, error)` that parses its line-oriented config, fills `Analysis` (segments as `VLAN`, zones, addresses, services, policies, `wan`), and calls `derive`. `analyze.go` dispatches on vendor. FortiGate's `parse`/`interpret` path is untouched.

**Tech Stack:** Go 1.23, hand-rolled line parsers (no new deps); Svelte frontend; Wails binding already accepts `vendor`.

**Spec:** `docs/superpowers/specs/2026-11-10-multi-vendor-firewall-design.md`

## Global Constraints

- No new external dependencies; all parsing in-package, mirroring `parser.go`/`fortigate.go` style.
- Secrets (PSK, password/hash, key, certificate bodies) are NEVER surfaced in any Address/Service/Policy/Note value.
- Each vendor returns exactly ONE `Analysis` this slice (multi-vsys/context/LSYS is follow-on); `VDOM` = "" or the single context name if trivially present.
- `Policy.Action` normalized to `accept` (permit/allow) or `deny` (deny/drop/reject/block) — `flows.go` maps `accept`→allow.
- Named-rule vendors: synthetic sequential `Policy.ID` (1,2,3… per Analysis); real rule name in `Policy.Name`.
- All `Analysis` slices initialized non-nil (use `newAnalysis`), so JSON emits `[]` not `null`.
- Fixtures must reflect REAL export structure (owner rule: crafted toy fixtures have hidden Criticals). Use multi-section, realistically-named configs.

## Review Focus

- **Wrong-format paste**: PAN-OS XML (starts `<`) or SRX curly-brace (`{`) into the set-format parser → clear error naming the expected format, not an empty analysis. (Tested in Tasks 1 & 3.)
- **Secret leakage**: a config line carrying a PSK / password / key must not put the secret into any surfaced value. (Tested per vendor.)
- **ASA access-group with no matching access-list (and vice-versa)**: ACL shown with best-effort/unknown interface binding, never dropped or panicking. (Task 2.)
- **Quoted names with spaces** (PAN-OS `"My Rule"`, address `"DC Servers"`): preserved as one token. (Task 1.)
- **Object/rule referencing an undefined address or service name**: passed through as the literal name, no panic, no silent drop. (Each vendor.)

---

### Task 1: Palo Alto PAN-OS (`set`-format)

**Files:**
- Create: `internal/fwconfig/panos.go`
- Create: `internal/fwconfig/panos_test.go`
- Modify: `internal/fwconfig/analyze.go` (add `case "panos"`)
- Modify: `internal/fwconfig/flows.go` (`countSubnetsIPs`: also skip catch-all named `"any"`)

**Interfaces:**
- Produces: `analyzePanos(configText string) ([]Analysis, error)`; a shared `splitSetTokens(line string) []string` quoted-aware tokenizer (also used by Task 3).
- Consumes: `newAnalysis`, `derive`, model types from `model.go`.

PAN-OS set-lines parsed (prefix → model):
- `set network interface ethernet <if> layer3 ip <A.B.C.D/M>` and `... units <if>.<n> ...` → `VLAN{Name:if, Interface:if, Subnet:cidr}` (ID = subinterface tag if `.N`, else 0).
- `set zone <Z> network layer3 <if>` → `Zone{Name:Z, Interfaces:[...]}`; member subinterfaces resolve to segment zone.
- `set address <N> ip-netmask <v>` / `ip-range <v>` / `fqdn <v>` → `Address{N, subnet|iprange|fqdn, v}`; `set address-group <N> static [ <a> <b> ]` → `Address{N, group, "a, b"}`.
- `set service <N> protocol tcp|udp port <p>` → `Service{N, TCP|UDP, p}`; `set service-group <N> members [ ... ]` → `Service{N, group, "..."}`.
- `set rulebase security rules <N> from <z> to <z> source <a> destination <a> application <app> service <s> action allow|deny|drop|reset` (+ `disabled yes`, `negate-source yes`) → `Policy`. App-ID merged into `Services` as `app:<app>` (unless `any`); `Negated` only for negate-source/destination or url-category-only.
- `nameif outside`-equivalent: a zone whose name is `untrust`/`external` or interface role → `a.wan[member]=true`.
- Non-set input (first non-blank line starts with `<`) → error: `panos: this looks like XML; paste the 'set'-format config (set cli config-output-format set)`.

- [ ] **Step 1: Write failing tests** in `panos_test.go` against a realistic multi-section set-format fixture (interfaces + zones + address/address-group + service + 2 security rules incl. one with App-ID and one `disabled yes`, plus a `set … psk <secret>` line):
  - `TestPanosSegmentsZones`: 2 segments with correct subnets, zone membership populated.
  - `TestPanosAddressesServices`: address (ip-netmask→subnet, ip-range→iprange), address-group (group), service (tcp port), service-group (group).
  - `TestPanosPolicLiesAndFlows`: 2 policies, names preserved, App-ID rule still produces a flow with `app:ssl` in services, disabled rule excluded from flows, action normalized.
  - `TestPanosSecretNotSurfaced`: the `psk` secret string appears in NO address/service/policy/note value.
  - `TestPanosRejectsXML`: input starting `<config>` → error mentioning "set".
- [ ] **Step 2: Run tests, verify they fail** (`go test ./internal/fwconfig/ -run TestPanos` → undefined `analyzePanos`).
- [ ] **Step 3: Implement** `analyzePanos` + `splitSetTokens` in `panos.go`; add `case "panos": return analyzePanos(configText)` to `analyze.go`; generalize `countSubnetsIPs` to also skip `"any"`.
- [ ] **Step 4: Run tests, verify pass** (`go test ./internal/fwconfig/ -run TestPanos -v`).
- [ ] **Step 5: Commit** (`feat(fwconfig): Palo Alto PAN-OS set-format analyzer`).

---

### Task 2: Cisco ASA (native running-config)

**Files:**
- Create: `internal/fwconfig/asa.go`, `internal/fwconfig/asa_test.go`
- Modify: `internal/fwconfig/analyze.go` (add `case "asa"`)

**Interfaces:**
- Produces: `analyzeAsa(configText string) ([]Analysis, error)`.
- Consumes: `newAnalysis`, `derive`, model types.

ASA lines parsed:
- `interface <if>` block with indented `nameif <name>`, `ip address <A.B.C.D> <mask>`, optional `vlan <n>`, `security-level <n>` → `VLAN{Name:nameif, Interface:if, Subnet:cidr, Zone:nameif}` + synthesized `Zone{Name:nameif, Interfaces:[nameif]}`; `security-level 0` or `nameif outside` → `a.wan[nameif]=true`.
- `name <ip> <alias>` → alias map used when resolving ACL/object addresses.
- `object network <N>` block with `host <ip>` / `subnet <ip> <mask>` / `range <a> <b>` → `Address`; `object-group network <N>` with `network-object ...` → `Address{group}`.
- `object service <N>` / `object-group service <N>` → `Service` / `Service{group}`.
- `access-list <ACL> extended permit|deny <proto> <src> <dst> [eq <port>]` grouped by `<ACL>` → policies (SrcAddr/DstAddr/Services from the ACE; Action from permit/deny).
- `access-group <ACL> in|out interface <nameif>` → binds every policy of `<ACL>`: `in` ⇒ `SrcIntf=[nameif]`; `out` ⇒ `DstIntf=[nameif]`. Unbound side empty. An `access-list` with no `access-group` → policies kept with empty intf (derive → Internet(any)); add a Note. Add a Note explaining ASA renders per bound interface → any (no dest-zone concept).

- [ ] **Step 1: Write failing tests** in `asa_test.go` against a realistic fixture (2 `interface`+`nameif` incl. `outside` security-level 0, a `name` alias, an `object network` host + an `object-group network`, an `object service`, a named `access-list` with 2 ACEs, an `access-group ... in interface outside`, and a `... key <secret>`/`password <secret>` line):
  - `TestAsaInterfacesZones`: 2 segments, nameif as zone, outside in `wan`.
  - `TestAsaObjects`: object network host→subnet /32, object-group→group, object service.
  - `TestAsaAclToFlows`: ACEs under one ACL become policies; `access-group in interface outside` sets SrcIntf=[outside]; flows render outside→Internet(any); action normalized.
  - `TestAsaDanglingAccessGroup`: `access-group` naming a nonexistent ACL, and an ACL with no access-group → no panic, Note present.
  - `TestAsaSecretNotSurfaced`: key/password secret not in any value.
- [ ] **Step 2: Run tests, verify fail.**
- [ ] **Step 3: Implement** `analyzeAsa` in `asa.go`; add `case "asa"` to `analyze.go`.
- [ ] **Step 4: Run tests, verify pass.**
- [ ] **Step 5: Commit** (`feat(fwconfig): Cisco ASA analyzer`).

---

### Task 3: Juniper SRX (`set`-format)

**Files:**
- Create: `internal/fwconfig/srx.go`, `internal/fwconfig/srx_test.go`
- Modify: `internal/fwconfig/analyze.go` (add `case "srx"`)

**Interfaces:**
- Produces: `analyzeSrx(configText string) ([]Analysis, error)`.
- Consumes: `splitSetTokens` (Task 1), `newAnalysis`, `derive`, model types.

SRX set-lines parsed:
- `set interfaces <if> unit <n> family inet address <A.B.C.D/M>` → `VLAN{Name:if.n, Interface:if, Subnet:cidr}`.
- `set security zones security-zone <Z> interfaces <if>.<n>` → `Zone` membership; untrust zone → `a.wan`.
- `set security address-book <book> address <N> <cidr|host>` / `address-set <N> address <m>` → `Address` / `Address{group}` (global and per-zone books both accepted).
- `set applications application <N> protocol <tcp|udp> destination-port <p>` / `application-set <N> application <m>` → `Service` / `Service{group}`; predefined `junos-<name>` referenced in a policy passed through as a service literal.
- `set security policies from-zone <X> to-zone <Y> policy <P> match source-address <a> destination-address <a> application <app>` + `then permit|deny|reject` → `Policy{Name:P, SrcIntf:[X], DstIntf:[Y], ...}`.
- Non-set input containing a top-level `{` before any `set ` → error: `srx: this looks like curly-brace format; paste 'show configuration | display set' output`.

- [ ] **Step 1: Write failing tests** in `srx_test.go` against a realistic fixture (2 interfaces, trust+untrust zones, a global address-book with address + address-set, an application + application-set, 2 from-zone/to-zone policies incl. one `then deny`, and a `set … pre-shared-key … <secret>` line):
  - `TestSrxInterfacesZones`: segments + zone membership; untrust in `wan`.
  - `TestSrxAddressBookApplications`: address (cidr→subnet), address-set (group), application (tcp dport), application-set (group).
  - `TestSrxPoliciesFlows`: 2 policies with from/to zones as Src/DstIntf, names preserved, deny normalized, flows derived zone→zone.
  - `TestSrxSecretNotSurfaced`: pre-shared-key secret absent from all values.
  - `TestSrxRejectsCurlyBrace`: curly-brace input → error mentioning "display set".
- [ ] **Step 2: Run tests, verify fail.**
- [ ] **Step 3: Implement** `analyzeSrx` in `srx.go`; add `case "srx"` to `analyze.go`.
- [ ] **Step 4: Run tests, verify pass.**
- [ ] **Step 5: Commit** (`feat(fwconfig): Juniper SRX set-format analyzer`).

---

### Task 4: UI vendor selector + generalization

**Files:**
- Modify: `frontend/src/lib/tools/FirewallAnalyzer.svelte`
- Modify: `internal/app/fwconfig_methods.go` (doc comment: list supported vendors)
- Regenerate: `frontend/wailsjs/go/app/App.*`, `models.ts` (no signature change expected; run generate to be safe)

**Interfaces:**
- Consumes: `AnalyzeFirewall(configText, vendor)` (unchanged signature).

Changes:
- Add a vendor `<select>` (fortigate | panos | asa | srx) bound to a `vendor` var; pass it to `AnalyzeFirewall` (replace hardcoded `"fortigate"` at line 25); re-analyze (or clear results) on vendor change.
- Relabel the context `<select>` from "VDOM" to "Context" (serves VDOM/vsys/context/root); keep `an.vdom || "(root)"`.
- Segments table card: rename header "VLANs" → "Segments"; hide the numeric **VLAN-id column** when every segment in the current view has `id === 0` (FortiGate still shows it). Generalize the count tile label "VLANs" → "Segments".
- Generalize the hardcoded "FortiGate"/"VLANs" lede text to be vendor-neutral (name the selected vendor, or neutral wording).

- [ ] **Step 1: Implement** the Svelte changes (this task is UI; verification is build + manual, not a Go unit test — note in ledger per TDD exception for UI).
- [ ] **Step 2: Build** `npm run build` (in `frontend/`) → succeeds with no errors.
- [ ] **Step 3: Regenerate bindings** (`$(go env GOPATH)/bin/wails generate module`; then `git checkout -- frontend/wailsjs/runtime`).
- [ ] **Step 4: Manual check** via the running app (or Playwright): select each vendor, paste the task fixture, confirm segments/zones/flows render and the VLAN-id column is hidden for non-FortiGate.
- [ ] **Step 5: Commit** (`feat(app,ui): firewall analyzer vendor selector (PAN-OS/ASA/SRX)`).

---

## Self-review notes

- Spec coverage: PAN-OS (T1), ASA (T2), SRX (T3), UI selector + generalization (T4), dispatch + countSubnetsIPs (T1). All design sections covered.
- Review Focus: wrong-format (T1/T3 reject tests), secret leakage (per-vendor secret tests), ASA dangling access-group (T2), quoted names (T1 fixture), undefined-reference passthrough (implicit in each — add an assertion if a fixture references an undefined name).
- Types consistent: every vendor fills the same model via `newAnalysis` + `derive`; `splitSetTokens` shared by T1/T3 defined in T1.
- Proportion: plan specifies signatures, prefixes, and test names/assertions, not bodies — implementer writes the parser logic.
