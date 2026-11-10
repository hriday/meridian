# Meridian — RIPE Atlas remote traceroute (bring your own key)

**Date:** 2026-11-10
**Status:** design, ready to plan
**Foundation:** `docs/superpowers/specs/2026-11-08-meridian-design.md` §4.5
(Looking glass — remote ping/traceroute from Atlas probes).
**Builds on:** `docs/superpowers/specs/2026-11-10-meridian-atlas-ping-design.md`
(same key storage, same schedule/poll engine). Remote traceroute was named
there as "the natural follow-up on the same key + engine."

## Intent

Traceroute a target **from vantage points around the world** using the RIPE
Atlas network, showing the per-probe path (each hop's responding address and
RTT) — the remote counterpart to the local AS-aware traceroute. Reuses the
operator's already-stored Atlas key; no new key handling.

## Key handling

Unchanged from the ping slice: the key is read from the local
`settings.json` (0600), sent only to `atlas.ripe.net` in `Authorization:
Key …`, never returned to the frontend in full. This slice adds **no** new
key storage or entry UI — if no key is set, the tool points the operator at
the Remote ping tool's setup (or shows the same short "add a key" message).

## How it works

1. **Schedule** — `POST atlas.ripe.net/api/v2/measurements/` with a one-off
   `traceroute` definition (`is_oneoff: true`, `af` 4/6 from the target,
   `protocol: "ICMP"`, a small `firstHop`/`maxHops` left at Atlas defaults)
   and a worldwide probe selection (`{requested: N, type: "area", value:
   "WW"}`). Returns a measurement id. The scheduling POST is **shared** with
   the ping slice (same endpoint, headers, error mapping) — only the
   definition differs.
2. **Poll results** — `GET /api/v2/measurements/<id>/results/` until results
   arrive or an overall timeout.
3. **Report** — per probe: probe id, source address, destination, and the
   ordered hops. Each hop has a number and up to three replies; a reply is
   either a responding address + RTT or a timeout (`*`). Per hop we show the
   address(es) that answered and the best/representative RTT; a fully
   timed-out hop shows `*`. We also surface whether the probe **reached** the
   destination.

Atlas errors (invalid key, insufficient credits) map to the same clear
messages as the ping slice (shared `scheduleError`).

## Scope

**In scope:** a one-off remote **traceroute** — shared schedule/poll engine,
traceroute definition + result parsing in `internal/atlas`, an `AtlasTrace`
app method, and a "Remote traceroute" Workbench tool under Internet that
reuses the stored key and renders per-probe paths.

**Non-goals (future / out):** UDP/TCP traceroute protocol choice (ICMP only
for now); per-hop ASN/geo enrichment (show hop IP + RTT); choosing specific
probes/countries beyond a worldwide count; reusing past measurements; DNS
measurements. *My calls.*

## UX (Workbench tool, Nielsen-compliant)

- **No key set:** a short panel saying a RIPE Atlas key is needed and to add
  it in the **Remote ping** tool (one place owns key entry); nothing else.
- **Key set:** the ping-style form — a target input, a probe count (default
  5, since traceroute output is larger per probe; sensible cap), a Run
  button. A "measuring…" state while polling.
- **Results:** a global line (responding probes, how many reached the
  target), then **per probe** a compact hop list — `hop · address · rtt`,
  `*` for a timeout, the destination row marked when reached. Reuses the
  established visual language (cards, mono, the good/warn/bad banners).
- The key never leaves the machine except to RIPE.

## Testing posture

- **Engine** tested against `httptest` via the injected `Doer` + base-URL
  seam: the shared schedule parses the id and sends the key header with a
  `traceroute` definition; traceroute results parse per-probe hops (including
  timeout replies `{"x":"*"}` and a reached-destination case); `Trace` polls
  (empty first, then results) and aggregates reached/responding; the same 403
  and insufficient-credits mapping is exercised (shared with ping, so a
  focused test suffices). No live Atlas in tests.
- **App** `AtlasTrace` is tested through the settings seam + an `atlasTrace`
  function seam: errors clearly when no key is set; passes the key/target
  through; clamps the probe count.
- The shared refactor of the scheduling POST keeps the existing ping tests
  green (regression safety).
