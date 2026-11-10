# Meridian — RIPE Atlas remote ping (bring your own key)

**Date:** 2026-11-10
**Status:** design, ready to plan
**Foundation:** `docs/superpowers/specs/2026-11-08-meridian-design.md` §4.5
("Looking glass" — RIPE Atlas for remote ping/traceroute, needs the user's own
free Atlas key + credits).

## Intent

Ping a target **from vantage points around the world**, not just the local
machine, using the RIPE Atlas measurement network — the "looking glass" the
foundation design calls for. The operator supplies their own free Atlas API key
(entered in the app, stored locally), Meridian schedules a one-off ping
measurement across Atlas probes, waits for the probes to report, and shows the
per-probe round-trip times with a global summary.

This slice delivers remote **ping**; remote traceroute is the natural follow-up
on the same key + engine.

## Key handling — the core constraint

The Atlas API key is a **secret**, and Meridian is open-source. Therefore:

- The key is **never** committed, never in the source, never hardcoded. Meridian
  ships with no key.
- The operator enters their key **in the app**. It is persisted **locally** to
  `os.UserConfigDir()/meridian/settings.json` with `0600` permissions (readable
  only by them), on their machine only.
- The key is sent **only to RIPE Atlas** (`atlas.ripe.net`), in the
  `Authorization: Key …` header, and nowhere else — not to any other service, not
  in logs, not back to the frontend in full (the UI shows only a masked hint and
  whether a key is set).

The in-app setup also tells the operator where to get a key and that running
measurements needs Atlas **credits** (usually from hosting a probe).

## How it works

1. **Schedule** — `POST atlas.ripe.net/api/v2/measurements/` with the key, a
   one-off `ping` definition (`is_oneoff: true`, `af` 4/6 from the target,
   `packets: 3`) and a probe selection (`{requested: N, type: "area", value:
   "WW"}`). The response gives a measurement id.
2. **Poll results** — `GET /api/v2/measurements/<id>/results/` with the key every
   few seconds until results arrive (probes report over ~tens of seconds) or an
   overall timeout; the one-off measurement stops itself when done.
3. **Report** — per probe: probe id, source address, min/avg/max RTT, packets
   sent/received (loss). Globally: responding probes, overall min/avg/max, and
   mean loss.

Atlas errors are surfaced plainly: an invalid/expired key, and — the common one
— **insufficient credits** ("host a probe or get credits"), each as a clear
message rather than a raw HTTP status.

## Scope

**In scope:** local key storage + in-app entry; schedule + poll + parse a one-off
Atlas **ping**; engine (`internal/settings`, `internal/atlas`), app methods, and
a "Remote ping" Workbench tool under "Internet".

**Non-goals (future / out):** remote traceroute and DNS measurements (same key +
engine, later); choosing specific probes/countries beyond a world-wide count;
reusing or listing past measurements; per-probe geo/ASN enrichment (show probe id
+ source IP for now); any server-side storage of the key. *My calls.*

## UX (Workbench tool, Nielsen-compliant)

- If **no key is set**: a setup panel — a password-type field to paste the key, a
  Save button, and a plain note: what it's for, that it's stored only on this
  machine, a link to get a key (atlas.ripe.net), and that running measurements
  needs credits. Nothing else is shown until a key exists.
- If a **key is set**: a masked indicator (e.g. "Atlas key set · …9fd19d") with a
  Change / Remove control, then the ping form — a target input, a probe count
  (default 10, sensible cap), and a Run button.
- **Results**: a global summary banner (responding probes, min/avg/max, loss) and
  a per-probe list (probe id, source, RTT, loss). A "measuring…" state while
  polling, since it takes a little time. Clear errors for key/credit failures.
- The screen explains itself; the key never leaves the machine except to RIPE.

## Testing posture

- **Settings** are tested behind a config-dir seam (a temp dir): set → persisted
  with `0600` → read back; clear removes it; a missing file yields an empty key,
  no error.
- **Atlas engine** is tested against an `httptest` server via an injected `Doer`
  and base-URL seam: schedule parses the measurement id and sends the key header;
  results parse per-probe ping fields; `Ping` polls (empty first, then results)
  and aggregates; a 403 and an insufficient-credits error map to clear messages.
  No live Atlas in tests.
- **App** key methods are tested through the settings seam; `AtlasPing` errors
  clearly when no key is set.
