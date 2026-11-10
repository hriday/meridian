# Meridian — PeeringDB enrichment (design)

**Status:** approved by owner (2026-11-09, "PeeringDB enrichment").

## Intent and why
Enrich the **ASN entity page** (and, via its origin AS, the IP page) with
**PeeringDB** data: how a network peers — its type, policy, the IXPs it's present
at, and the facilities (data centers) it occupies. This turns the ASN page from
"who is this" into "how does this network connect," which is exactly what a
network/peering engineer wants next. PeeringDB's public API is **keyless**.

## Goals
- For an ASN, fetch its PeeringDB **net** record and show:
  - name, **network type** (Content/NSP/…), **scope** (Global/Regional/…),
    **traffic ratio** (Mostly Outbound/Balanced/…), **peering policy**
    (Open/Selective/Restrictive), **IRR AS-SET**, website, and announced-prefix
    counts (v4/v6);
  - **IX presence** — the Internet Exchanges it's at, with port speed, deduped by
    IX name (a big network has hundreds — the UI caps the list with a "+N more");
  - **facilities** — the data centers it's in (name, city, country), likewise capped.
- Surface it as a **Peering** card on the ASN entity page, each metric with a
  one-line caption (the plain-language standard). The IP page keeps a short
  "peers at N IXPs / N facilities" line linking through its origin AS.
- **CLI:** `meridian-cli lookup AS13335` prints a peering summary.
- Partial failure degrades the card, never the whole lookup (the existing
  `Source{Label, FetchedAt, Err}` pattern).

## Decisions
- **Source:** PeeringDB public API (`peeringdb.com/api/net?asn=`, `/netixlan?net_id=`,
  `/netfac?net_id=`), keyless, reusing the `lookup.Client` (Doer + URL cache).
- The net summary is the primary fetch; the IX/facility lists are best-effort
  (so rate-limiting or an error on them still leaves the summary).
- IX entries are **deduped by IX name** (a net has multiple netixlan rows per IX
  for v4/v6/ports); keep the max observed speed per IX.

## Non-goals / deferred
- Writing to PeeringDB; authenticated endpoints; POCs/contacts; org/IX detail
  pages; "who else is at this IXP" cross-reference; PeeringDB as a standalone
  tool (this is enrichment of the existing Lookup).
- No new dependency; no API key.

## Constraints (global)
- Go 1.23; reuse `lookup.Client`; `peeringDBBase` a package var so tests point it
  at an httptest server. Generic `meridian/*` User-Agent; no user identity.
- Shapes copied from the **verified-live** API (net/netixlan/netfac above) — no
  invented fields.
- The ASN/IP report JSON stays backward-compatible (new optional fields).

## What the builder decides
- Go types/field names, the dedupe/sort/cap specifics (suggest: unique IX names
  sorted by speed desc; facilities sorted by name; store all, cap in the UI like
  BGP's upstream/transit), and the Svelte card layout within the entity-page kit.
