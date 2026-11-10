# Meridian — MAC / OUI vendor finder

**Date:** 2026-11-10
**Status:** design, ready to plan
**Foundation:** `docs/superpowers/specs/2026-11-08-meridian-design.md` §4.6 ("MAC finder").

## Intent

Given a MAC address (or just its OUI prefix), tell the operator which vendor made
the network interface — the first three bytes of a MAC are the IEEE-assigned
Organizationally Unique Identifier. It's the companion to Wake-on-LAN and the
future ARP/neighbour tools: "whose device is `fc:fb:fb:…`?" Meridian should
normalize the input, look up the vendor, and report it plainly — or say clearly
when the OUI isn't registered (a locally-administered / randomized MAC, or an
unassigned block).

## Data source — my call

A **keyless public MAC-vendor API** (`api.macvendors.com`), used exactly like the
other outbound lookups in Meridian (RDAP, RIPEstat, Team Cymru, PeeringDB): a
generic `User-Agent` carrying no user identity, and only the queried OUI leaving
the machine — never who asked. This keeps the binary small and the data current,
and it is fully offline-testable through the injected `Doer` seam (an httptest
server), the same way the RDAP/PeeringDB code is tested.

Chosen over bundling the full IEEE OUI registry (~a megabyte of data to embed and
keep fresh) because the API matches the project's established on-action-query
pattern and adds no blob. The one caveat is the free tier's rate limit (~1
lookup/second), which the tool surfaces plainly rather than hiding.

*Alternatives considered and rejected for this slice: embedding the IEEE OUI
database (binary bloat + staleness), and a port-43/Wireshark-manuf parse (same
blob concern). A later slice may add an offline cache.*

## Why it's safe and in scope

Read-only, user-initiated. The query contains only a 3-byte OUI prefix (a vendor
identifier, not a device or a person). No user identity is sent. It is the same
lookup every "MAC vendor" web tool performs.

## Behaviour

- **Input:** a MAC or OUI, in any common spelling — `fc:fb:fb:dd:ee:ff`,
  `fc-fb-fb-…`, `fcfb.fbdd.eeff`, bare `fcfbfbddeeff`, or just the OUI
  `fc:fb:fb` / `fcfbfb`. At least three bytes (six hex digits) are required.
- **Found:** the vendor/organization name, plus the normalized OUI (`FC:FB:FB`).
- **Not found:** a clear "this OUI isn't registered" — common for
  locally-administered or randomized MACs (the 2nd-least-significant bit of the
  first byte set), which the tool also points out when it sees it.
- **Rate limited:** a plain message that the free API allows about one lookup a
  second; try again shortly — not a generic error.

## Scope

**In scope:** normalize + validate the MAC/OUI (pure); look up the vendor via the
keyless API (injected `Doer`); engine (`internal/macvendor`), CLI (`macvendor`),
Wails method, and a Workbench tool under "Local network".

**Non-goals (future / out):** an offline/embedded OUI database; bulk lookups;
MAC randomization analysis beyond the locally-administered flag; resolving a MAC
to a device (that's the IP-space records / ARP features). *My calls.*

## UX (Workbench tool, Nielsen-compliant)

- A MAC/OUI input and a Look up button, with a one-line note on what an OUI is
  and that a locally-administered/random MAC has no vendor.
- On a hit: the vendor name prominently, with the normalized OUI beside it.
- On a miss: a plain "not a registered OUI" with the locally-administered note
  when applicable — not an error state.
- On rate-limit or network failure: a clear, specific message.

## Testing posture

- `normalize` is pure: table-tested across the accepted spellings (all yielding
  the same OUI), the too-short / non-hex rejections, and the
  locally-administered-bit detection.
- `Lookup` is tested against an **httptest server** via the injected `Doer` and a
  base-URL seam: a 200 hit, a 404 miss (Found=false, no error), a 429
  rate-limit (a distinct error), and a network failure — all offline.
