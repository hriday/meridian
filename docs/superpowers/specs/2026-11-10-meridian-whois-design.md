# Meridian — Whois-grade domain registration

**Date:** 2026-11-10
**Status:** design, ready to plan
**Foundation:** `docs/superpowers/specs/2026-11-08-meridian-design.md` §4.6 ("DNS & Whois"),
§4.1 lookups (domain entity).

## Intent

The domain entity already shows registrar, registration/expiry dates, DNSSEC,
and nameservers (via `LookupDomain` → `rdapDomain`). Classic Whois shows more of
what an operator actually needs when a domain is in trouble: the **status codes**
(EPP states like `clientTransferProhibited`, `clientHold`, `pendingDelete`) that
say whether the domain is locked, on hold, or expiring; the **last-updated date**;
and the **registrar abuse contact** to report abuse to. RDAP already carries all
three — we just don't parse them yet. This slice deepens the existing domain
registration to Whois-grade, in the lookup core and its entity page, rather than
adding a redundant standalone tool.

## Why no new tool

A standalone "Whois" tool would duplicate `LookupDomain`, which already resolves
a domain's DNS, hosting, and RDAP registration and is reached by the unified
Lookup / ⌘K search. The right move is to enrich that existing path. *My call,
from finding the domain lookup already present.*

## What's added

Parsed from the domain's RDAP record (all optional — RDAP shape varies by TLD,
and missing pieces stay empty, never error):

- **Status codes** — the RDAP `status[]` array (EPP states). Shown raw (they're
  standard and meaningful), with a short plain-language gloss in the UI for the
  common ones (e.g. `clientHold` → "registrar has stopped it resolving",
  `pendingDelete` → "scheduled for deletion").
- **Last updated** — the RDAP `last changed` event date.
- **Registrar abuse contact** — the abuse entity's email (via the existing
  `findRole(entities, "abuse", "email")` walk).

## Non-goals

- Legacy port-43 WHOIS text scraping (RDAP is the structured successor; we stay
  RDAP-only, consistent with the rest of the project).
- Registrant/admin/tech personal contacts (almost always redacted post-GDPR;
  not worth surfacing empty rows).
- A separate Workbench tool or CLI command (this enriches the existing domain
  lookup). *My calls.*

## Surfaced in

- **Engine:** `DomainReg` and `DomainReport` gain `Statuses []string`,
  `Updated string`, `AbuseEmail string`.
- **CLI:** `lookup <domain>`'s `printDomainReport` prints Status / Updated /
  Abuse rows when present.
- **UI:** the domain entity's registration card shows a status chip row (with
  gloss tooltips), the updated date beside created/expires, and the abuse
  contact — consistent with the existing card's style.

## Testing posture

- `rdapDomain` is tested with an extended RDAP fixture carrying `status`, a
  `last changed` event, and a nested registrar→abuse entity; asserts the three
  new fields parse, and that a record lacking them still yields empty values
  (no error). The existing domain/report tests continue to pass unchanged.
