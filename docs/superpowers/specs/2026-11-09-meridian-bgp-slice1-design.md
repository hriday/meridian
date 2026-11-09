# Meridian — BGP & looking glass (slice 1) design

**Status:** approved by owner (picked Option A + "yes to the 3" on 2026-11-09).

## Intent and why
Show the **routes and AS-paths** to a prefix/IP as seen from public BGP route
collectors — the *control plane* view that complements traceroute's *data
plane* view. Meridian runs no routers and peers with no one (design decision
49): it reads public RIS route-collector data via RIPEstat. This rounds out
the "who routes this / what are the AS hops" story that lookups and traceroute
began, and fixes the wall-of-text vantage list the owner disliked on bgp.he.net
by making collectors searchable.

## Decisions (confirmed)
1. **Primary source: RIPEstat `looking-glass`** (aggregated RIS peers; no API
   key). Not BGP.tools, not LG-site scraping.
2. **RIPE Atlas live measurements are deferred** to their own later slice (they
   need the user's Atlas key + credit handling).
3. **Shape: its own "BGP & looking glass" tool** in the Internet group (the
   Workbench already stubs `id: "bgp"`, `ready: false`). Not a tab on the
   prefix page.

## Goals
- Given a prefix or IP, fetch the looking-glass view and present:
  - **Origin**: the origin AS(es) seen (flag MOAS — more than one origin).
  - **Visibility/health**: how many collectors and peers see it; how many
    distinct AS paths; RPKI validity of (origin, prefix) reusing the lookups-2
    `ripeRPKI` call.
  - **AS-path analysis**: the distinct AS paths ranked by how many peers
    observe each, with a percentage; the set of **upstreams** (the AS adjacent
    to the origin on each path) and **transit** ASes (all intermediate ASes).
    Every AS is a pivot to its ASN entity page.
  - **Collector / vantage picker**: the list of RRCs that see the prefix, each
    with its id, location, scope, and peer count — **searchable/filterable** by
    location or ASN, and selectable to show that collector's own paths.
- CLI: `meridian-cli bgp <prefix|ip>` prints the same summary.
- Pivots: origin/upstream/transit AS → ASN page; the prefix → prefix page;
  the existing "Traceroute" / "Save to IP space" actions where they fit.

## Non-goals / deferred ("soon")
- RIPE Atlas remote measurements (ping/traceroute *from* a vantage).
- BGP update history / time-travel; community *decoding* beyond passing through
  RIPEstat's `community_info` descriptions.
- Executing looking-glass *commands* on real routers.
- Per-collector deep links out to bgp.he.net (we show our own view).

## Constraints (global, from the project)
- Go 1.23 directive held; **no new dependencies** — reuse the `lookup.Client`
  (injected `Doer`, URL cache), the RIPEstat base, the entity-page shell, the
  `requestedTool` nav store, and `ripeRPKI`.
- Every protocol screen explains itself in plain language: a verdict line plus
  per-metric captions (the standard set on the NTP screen). The BGP screen must
  say, in plain words, what "seen by N peers / M paths" and an RPKI state mean.
- Partial failure degrades a card, never the whole report (per-section
  `Source{Label, FetchedAt, Err}`), matching the other lookups.
- Outbound requests carry the generic `meridian/0.1` User-Agent; no user
  identity.

## The real RIPEstat `looking-glass` shape (verified live 2026-11-09)
`GET https://stat.ripe.net/data/looking-glass/data.json?resource=<prefix|ip>`
```
data.rrcs[]            // one per route collector that sees the resource
  .rrc                 // "RRC01"
  .location            // "London, GB"
  .scope               // "LINX / LONAP"
  .peers[]
    .asn_origin        // "1205"  (string)
    .as_path           // "15692 2914 1764 1853 1205"  (space-joined string)
    .community         // "174:70 1764:10 ..."  (space-joined string)
    .community_info    // { regular: { "1764:100": {description, original} }, large, extended }
    .next_hop          // "5.57.80.113"
    .origin            // "IGP" | "EGP" | ...
    .peer              // peer IP
    .prefix            // the matched prefix, e.g. "140.78.0.0/16"
    .last_updated      // ISO8601
```
AS paths read **left→right as seen→origin**: the last element is the origin AS,
the element before it is that peer's view of the origin's upstream.

## What the builder decides
- Go idioms, file organization within the stated files, exact helper names
  not pinned by the plan, and the Svelte layout details (within the
  plain-language standard). The plan pins the engine types, signatures, the
  aggregation rules, and the tests.
