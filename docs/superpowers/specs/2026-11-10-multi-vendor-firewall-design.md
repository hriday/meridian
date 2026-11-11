# Multi-vendor firewall config analyzer — design

**Date:** 2026-11-10
**Status:** design (awaiting builder-check, then plan)

## Intent & why

Meridian's firewall analyzer (`internal/fwconfig`) today reads **FortiGate** exports
offline and derives a structured, vendor-neutral view: L3 segments, address/service/
zone/policy objects, and a "what talks to what" flow graph. Network engineers work
across vendors; the owner asked to extend it to the other big firewalls — **Palo Alto
PAN-OS**, **Cisco ASA**, and **Juniper SRX** — "just like Fortinet."

The valuable, already-built core is the **vendor-neutral model** (`model.go`) and the
**flow deriver** (`flows.go` → `derive(&a)`), which turn `[]Policy` + `[]Zone` +
interfaces into an aggregated flow view. Adding a vendor means writing a parser +
interpreter that fills the same `Analysis` and calls `derive`. Everything downstream
(flows, counts, the Svelte UI's tables and flow drill-in) then works unchanged.

## Goals

- Parse **PAN-OS**, **ASA**, and **SRX** configs offline into the existing `Analysis`
  model, producing segments, zones, addresses, services, policies, flows, and counts —
  the same shape the UI already renders for FortiGate.
- Add a **vendor selector** to the analyzer UI (currently hardcodes `"fortigate"`), and
  let the backend dispatch on it.
- Each vendor validated against **realistic, real-export-structured fixtures** (per the
  owner's standing rule: crafted toy fixtures have hidden Criticals twice before).
- Secrets (pre-shared keys, password hashes, certificates) are **never surfaced**, same
  as the FortiGate path.

## Non-goals / anti-goals

- **Not** a config editor, pusher, or live-device connection — this stays offline,
  read-only, paste/load-a-file, exactly like today.
- **Not** full fidelity of every vendor knob. We target the sections that feed the
  model (interfaces/segments, zones, address & service objects + groups, security
  rules). NAT, VPN, routing, QoS, and app-ID signatures are out of scope.
- **Not** every config dialect in this slice. Each vendor targets the one line-oriented
  format engineers actually paste/diff (below). Hierarchical/XML variants are a noted
  follow-on, not this slice.
- No new external dependencies — all parsing is hand-rolled line/token work in-package,
  matching the existing FortiGate parser.

## Config format targeted per vendor (the key decision)

Each vendor has more than one on-disk form. This slice targets the **flat, line-oriented
"set"/native format** for each, because it is what engineers paste into a tool, what
diffs cleanly in change tickets, and what parses uniformly:

| Vendor | Targeted format | Example first tokens | Deferred (noted) |
|--------|-----------------|----------------------|------------------|
| PAN-OS | `set`-format (`set cli config-output-format set`) | `set network interface…`, `set zone…`, `set address…`, `set rulebase security rules…` | XML running-config |
| Cisco ASA | native running-config (the only format) | `interface…`, `nameif…`, `object network…`, `access-list…`, `access-group…` | — |
| Juniper SRX | `set`-format (`show configuration \| display set`) | `set interfaces…`, `set security zones…`, `set security policies…`, `set applications…` | curly-brace hierarchical |

If a user pastes a deferred format, the vendor interpreter should detect it (e.g. PAN-OS
XML starts with `<`, SRX curly-brace has `{`) and return a clear error naming the
expected format, rather than silently producing an empty analysis.

## Model mapping

The existing `Analysis` model is vendor-neutral enough to reuse as-is; the mapping per
vendor:

- **Segment** (reuse `VLAN` struct — it is really an L3 interface/segment: `ID, Name,
  Interface, Subnet, Zone`; `ID=0` when there is no VLAN tag). PAN-OS layer3 interfaces
  & subinterfaces; ASA `interface` + `nameif` + `ip address`; SRX `interfaces` units
  with `family inet address`.
- **Zone**: PAN-OS zones (interfaces are members); SRX `security-zone` (interfaces are
  members); ASA has no zones — synthesize one "zone" per `nameif` name so `derive`'s
  zone/interface resolution still works (an ASA ACL is applied per-nameif via
  `access-group`).
- **Address** (`Name, Kind, Value`): PAN-OS `address` (`ip-netmask`/`ip-range`/`fqdn`)
  + `address-group`; ASA `object network` (`host`/`subnet`/`range`) + `object-group
  network` + `name` aliases; SRX `address-book` address + address-set.
- **Service** (`Name, Proto, Ports`): PAN-OS `service` + `service-group`; ASA `object
  service` + `object-group service` + protocol/port literals in ACLs; SRX `applications
  application` + `application-set` (plus junos- predefined names passed through).
- **Policy** (`ID, Name, SrcIntf, DstIntf, SrcAddr, DstAddr, Services, Action,
  Disabled, Negated`): PAN-OS `rulebase security rules` (from/to zones, source/dest/
  application/service, action allow|deny|drop|reset); ASA extended `access-list` lines
  grouped by ACL name, with `access-group` binding ACL→nameif+direction to populate
  Src/DstIntf; SRX `security policies from-zone X to-zone Y policy P` (match
  source-/destination-address/application, then permit|deny|reject).
  - `SrcIntf/DstIntf` carry **zone names** for PAN-OS/SRX (as FortiGate does) so
    `derive`'s zone-member resolution maps them to segments; for ASA they carry the
    `nameif` interface name.
  - `Action`: normalize to `accept` (permit/allow) or `deny` (deny/drop/reject/block),
    matching what `flows.go` expects (`"accept"` → allow).
  - `Negated`: set when a rule uses negation or application/URL-category match the flow
    graph can't represent, so `derive` lists it as a Note rather than a misleading flow.

`derive(&a)` and `Counts` are reused unchanged. The `wan` map (interfaces that face the
Internet) is populated per vendor where determinable (PAN-OS zone named/typed external,
ASA `nameif outside` / security-level 0, SRX untrust zone) and otherwise left empty
(flows then show the real zone name rather than "Internet (any)" — acceptable).

## File structure

Per vendor, mirroring `fortigate.go` + `fortigate_test.go`:

- `internal/fwconfig/panos.go` + `panos_test.go`
- `internal/fwconfig/asa.go` + `asa_test.go`
- `internal/fwconfig/srx.go` + `srx_test.go`
- A small shared `setcmd.go` helper if the PAN-OS/SRX `set`-line tokenization is worth
  sharing (both are "set " + space-separated tokens with quoted values); otherwise each
  keeps its own. Decide during implementation — do not over-abstract across three
  genuinely different grammars.
- `analyze.go`: replace the `vendor != "fortigate"` rejection with a dispatch to
  `interpretFortigate` / `interpretPanos` / `interpretAsa` / `interpretSrx`. (Rename the
  FortiGate `interpret`/`parse` to vendor-scoped names, or dispatch by wrapping.)
- `internal/app/fwconfig_methods.go`: doc comment only (the binding already takes
  `vendor`); no signature change.
- `frontend/src/lib/tools/FirewallAnalyzer.svelte`: add a vendor `<select>` (fortigate |
  panos | asa | srx), pass it to `AnalyzeFirewall`, re-analyze on change; generalize the
  "VDOM" selector label to "context" (PAN-OS vsys / ASA context / SRX root), and the
  "VLANs" count label to "Segments".

## Review focus (inputs the happy-path tests won't cover)

- **Wrong-format paste**: PAN-OS XML or SRX curly-brace into the set-format parser →
  clear error, not an empty/garbage analysis.
- **Secret leakage**: `set … psk …`, ASA `… password …`/`key …`, SRX
  `… pre-shared-key …` must never appear in any Address/Service/Policy/Note value.
- **ASA `access-group` with no matching `access-list`** (and vice-versa) → the ACL is
  shown with unknown interface binding, not dropped or crashed.
- **Quoted names with spaces / special chars** (PAN-OS allows `"My Rule"`), and object
  names colliding with keywords.
- **Empty or single-section configs** (only addresses, no policies) → valid analysis
  with empty flows, not an error.

## Builder-check resolutions (settled calls)

A fresh builder read this design against the code and raised six questions. Resolved
here — all settleable from the existing model, the feature's purpose, and the owner's
"no ugly" (Nielsen) standard; the owner's standing "go ahead, adjust later if it's
wrong" covers these. The three most output-shaping (★) are surfaced to the owner on
handoff so they can veto.

1. **Policy identity for named-rule vendors.** Assign a synthetic sequential `int`
   `Policy.ID` (1,2,3… within each Analysis) for keying/drill-in, and put the vendor's
   **real rule name** in `Policy.Name` (the model already carries it). The UI leads with
   `Name`; `#N` is a secondary handle. No model change.
2. ★ **PAN-OS App-ID does NOT force a Note.** A rule with `application` other than `any`
   is still drawn as a flow; the App-ID value is carried as a service-style label
   (merged into `Services`, e.g. `app:ssl`). `Negated` is set only for genuinely
   ungraphable matches (negated source/dest addresses, URL-category-only rules). Without
   this, real PAN-OS configs would produce ~zero flows — defeating the feature.
3. **Single Analysis per vendor this slice.** PAN-OS vsys / ASA context / SRX LSYS
   splitting is a noted follow-on. Each new vendor returns exactly one `Analysis`
   (`VDOM` = "" or the single vsys/context name if trivially present). No per-context
   scoping logic. (FortiGate multi-VDOM is unchanged.)
4. ★ **ASA flow rendering.** `access-group ACL in interface X` → that ACL's policies get
   `SrcIntf=[X]`; `... out interface X` → `DstIntf=[X]`. The unbound side is left empty,
   so `derive` renders `X → Internet (any)` (or `Internet (any) → X` for `out`). This is
   the honest per-interface reading ASA supports — ASA has no destination-zone concept.
   Documented as a Note on ASA analyses so the limitation is explicit. `nameif outside`
   / security-level 0 populates `a.wan` so outside↔Internet collapses correctly.
5. **UI generalized beyond the three named changes.** The segments table card is
   FortiGate-shaped (header "VLANs", a numeric "VLAN" column bound to `v.id`). For the
   new vendors `ID=0`, which would show "0" on every row (ugly). So: rename the card to
   "Segments", and **hide the VLAN-id column when every segment in the view has ID 0**
   (FortiGate still shows it). Generalize the hardcoded "FortiGate/VLANs" lede to be
   vendor-neutral. These join the vendor `<select>` + context/label relabels.
6. **No rename; dispatch in `analyze.go`.** `analyze.go` already calls `parse`+
   `interpret` for FortiGate — keep that case untouched (so `fortigate_test.go` is not
   touched) and add `case "panos"/"asa"/"srx"` each calling a single
   `analyzePanos/analyzeAsa/analyzeSrx(configText string) ([]Analysis, error)` that owns
   its own parse → fill → `derive`. The FortiGate `parse`/`interpret`/`*Block` tree is
   not shared with the new vendors.

Also settled from the builder's minor flags:
- `countSubnetsIPs` excludes the catch-all by name `"all"` only; generalize to also skip
  `"any"` (PAN-OS/SRX catch-all) — value `0.0.0.0/0` is already skipped.
- ASA: the synthesized zone name, its member interface name, and the `a.wan` key must be
  the same string (the `nameif`), or the Internet collapse won't fire.

## Decided during implementation (builder's discretion)

- Separate parsers per vendor; share at most a quoted-token splitter (PAN-OS and SRX
  `set`-lines diverge enough that a shared grammar would obscure more than it saves).
- ASA `object-group` kept as a group object (`Kind: "group"`), matching FortiGate
  addrgrp — not flattened.
