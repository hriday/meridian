# Meridian — Firewall config analyzer (offline)

**Date:** 2026-11-10
**Status:** design, ready to plan (mockup validated by owner)
**Foundation:** `docs/superpowers/specs/2026-11-08-meridian-design.md` §4.7
(Multi-vendor configs). This reframes part of §4.7 from *live device
management* toward **offline config comprehension**: drop an exported config,
understand it — no connection to the device.

## Intent

A network engineer exports a ruleset from their firewall and often can't
easily answer basic questions about it: how many VLANs are there, what can
reach what, on which ports. Meridian takes that exported config as **text the
engineer already has** and, **entirely offline**, tells them what's inside:
the VLANs, a "what talks to what" view derived from the policy rules, headline
counts, and a flow diagram.

This is the first tool in a **"Config insight"** area. It ships for one vendor
first (**FortiGate**); **Cisco ASA** is the planned next vendor, and an
**nginx config check** (validity + structure) is a planned sibling tool in the
same area — both get their own design/slice.

## Why offline matters

Meridian never connects to the firewall. The engineer pastes or drops the
exported config file; parsing happens locally in Go; nothing is sent anywhere
(consistent with Meridian's privacy stance). This makes it safe to analyse
production configs, testable without real gear, and usable on a config pulled
from anywhere (a backup, a ticket, a colleague).

## What it extracts (FortiGate, v1)

From a FortiGate (`config …/edit …/set …/next/end`) export:

- **VLANs** — every interface with a `vlanid`: id, name, parent interface,
  subnet (from the interface IP), and the zone it belongs to.
- **Address objects / groups** — subnets and hosts referenced by policies.
- **Services** — custom + built-in services referenced (port/protocol).
- **Zones** — `config system zone` membership (interfaces → zone).
- **Policies** — `config firewall policy`: srcintf, dstintf, srcaddr,
  dstaddr, service, action (accept/deny), plus enough to drill down.

From those it derives:

- **"What talks to what"** — for each policy, map srcintf/dstintf (interface
  or zone, possibly `any`) to the VLAN(s) behind them, and produce flows:
  **source VLAN → destination VLAN, service(s), action**. Aggregated at the
  **VLAN-pair + action** level (services collected). The exact source/
  destination **address objects and host IPs** behind each flow are shown on
  **drill-down**, but the flow is derived from the *interface* mapping, not by
  matching addresses against VLAN subnets — so the view is an honest reading
  of the ruleset at the VLAN level, which the UI states plainly, not a
  host-level effective-reachability computation (owner's call: VLAN-level v1).
  Non-VLAN interfaces (physical/tunnel) appear as nodes named by the
  interface; a WAN-role interface (`set role wan`) or `any` renders as
  "Internet (any)". Accept and deny for the same pair are separate rows
  (green / red); FortiGate's implicit deny is not drawn; first-match ordering
  and shadowing are out of v1 (an inventory, not a path evaluation).
- **Counts** — VLANs, policies, address objects, distinct subnets/IPs,
  services, zones. "Subnets/IPs" = the distinct set of resolved subnet/host
  values across address objects plus VLAN interface subnets, excluding `all`
  and unresolvable FQDN objects.
- **Excluded from flows:** policies with `set status disable` are skipped;
  policies using source/dest/service negation or `internet-service` are
  flagged "not fully analysed" rather than drawn as a normal flow (so the
  tool never shows a misleading reach).

## UX (validated mockup)

The results screen the owner approved:

1. **Load bar** — the parsed file name, line count, parse time; a vendor chip
   (FortiGate). Input is paste-or-drop of the exported config. **If the export
   is multi-VDOM, a VDOM picker appears here** (`[VDOM: root ▾]`) and the rest
   of the screen reflects the selected VDOM; a single-VDOM export skips it.
2. **Count tiles** — VLANs · Policies · Address objects · Subnets/IPs ·
   Services · Zones.
3. **VLANs table** — id, name, interface, subnet, zone.
4. **What talks to what** — the flow table: *source → destination · service
   pills (port/proto) · direction · allow/deny*, VLAN-aggregated, with a note
   that clicking a flow reveals the exact policies/IPs behind it. Denies are
   shown in red.
5. **Flow diagram** — VLANs as nodes, permitted flows as green directed edges,
   explicit denies as red dashed edges, and an "Internet (any)" cloud for
   external-bound rules. Edge labels name the **service unambiguously**
   (`tcp/22` or the service name `SSH`, never a bare `22`); the arrowhead
   points at the destination VLAN.

Held to Meridian's Nielsen bar: fast, legible, self-explaining, works at phone
width, matches the existing tool visual language (cards, mono, good/warn/bad).

## Scope

**In scope (v1):** offline FortiGate config parsing, **including multi-VDOM
exports** (detect every VDOM, a picker, one analysis per selected VDOM); the
VLAN/address/service/zone/policy model; the VLAN-level "what talks to what"
derivation; the counts; and the Workbench tool showing load (+ VDOM picker) →
tiles → VLAN table → flow table → diagram, with flow drill-down to the
underlying policies, address objects and host IPs.

**Non-goals / later (my calls):**
- Host-address-matched reachability (matching srcaddr/dstaddr against VLAN
  subnets to refine or filter flows) — the owner chose VLAN/interface-level
  for v1; address detail lives in drill-down.
- First-match ordering, shadowed/unused-rule analysis, policy optimisation,
  risk scoring — a follow-on once the model exists.
- Cisco ASA and other vendors — planned next, separate parser + slice.
- nginx config check — planned sibling tool, separate design.
- Live device connection, config download/compare/edit, config history
  (the original §4.7 live-management vision) — out of this tool.
- NAT, VIPs, routing, SD-WAN, and non-policy config sections — parsed only as
  far as needed for VLANs + flows in v1.

## Decided (my calls, for the builder to honour)

- **VLAN-level aggregation is the default** for "what talks to what", with
  per-policy / per-IP drill-down — not a raw rule dump front-and-centre.
- **The diagram is in scope for v1** (owner reacted to it as core), but it is
  the last thing built, so it can slip to a fast-follow slice if it balloons
  without blocking the table-based value.
- **Input is paste-or-drop** of the exported text (a large paste area plus a
  "load file" affordance), not a live pull.
- Pure-Go **block parser** for the FortiGate format (tokenise the
  `config/edit/set/next/end` structure) — not regex-scraping. It tolerates the
  `#config-version=` header, `config vdom` blocks, quoted/escaped string values,
  and encrypted secrets (`ENC …`), which are **never displayed**.
- **Multi-VDOM:** the parser detects every VDOM and produces one `Analysis`
  per VDOM; the UI picks which to show (owner uses VDOMs).
- **Tool placement:** a new **"Config insight"** Workbench group; tool id
  `fwanalyze`; the CLI adapter reads a config from a file path or stdin and
  emits JSON.
- **Counts / flow calls** as in "What it extracts": Subnets/IPs excludes `all`/
  unresolvable FQDNs; disabled policies skipped; negated/`internet-service`
  policies flagged not-fully-analysed; accept/deny per pair as separate rows;
  implicit deny and rule ordering out of v1.

## Architecture (Meridian conventions)

- Pure engine package (e.g. `internal/fwconfig`) with a FortiGate parser that
  turns config text into a structured `Analysis` (VLANs, objects, services,
  zones, policies, derived flows, counts). No I/O, no Wails — unit-testable
  against crafted sample configs.
- Thin `internal/app` method binding `Analyze(configText, vendor)` →
  one-or-more per-VDOM `Analysis` results, plus a CLI adapter (CLI + JSON per
  §4.9).
- A `FirewallAnalyzer.svelte` Workbench tool in a new "Config insight" group,
  with the VDOM picker when the export is multi-VDOM.

## Testing posture

Built and tested against the documented FortiGate config format (the owner has
no sample, so correctness is pinned by crafted representative fixtures covering:
VLAN interfaces, zones grouping VLANs, address objects/groups, custom +
built-in services, accept and deny policies, `any`/zone srcintf-dstintf,
and multi-service policies). Flow derivation, VLAN mapping, and counts each get
direct tests. No network, no real gear.

## Left to the builder

- Exact package layout and the `Analysis` type shape; the block-parser
  implementation.
- How zones and `any` expand into VLAN flows (and how to present a flow that
  spans multiple VLANs).
- Diagram rendering approach (hand-rolled SVG vs a small graph layout helper)
  and node placement for readability.
- Drill-down interaction specifics (flow → policies → address objects/IPs).
- The exact FortiGate keys to read for each section (verify against current
  FortiOS config syntax during implementation).
