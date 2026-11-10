# Meridian — MIB browser / walk — Design

**Status:** approved mockup (all three themes), ready to plan.
**Author:** assistant, owner-approved (mockup + defaults).
**Depends on:** the SNMP engine (`snmp.Dial`, `snmp.Client.Walk`/BulkWalkAll) shipped in #35.

## Intent / why

The monitoring tools show curated, interpreted views (throughput, CPU, health).
The MIB browser is the raw truth underneath: point it at a device, give it an
OID (or pick a starting point), and walk the subtree to see *every* value the
device exposes there, with its SNMP type. It's the tool for "what does this
device actually have?", for finding the OID behind a number, for confirming a
device speaks a MIB you expect, and for debugging why a monitor reads what it
reads.

## Goals

- Walk any OID subtree of a device over SNMPv2c and list each OID with its
  **type** (OctetString, Integer, Counter32, Counter64, Gauge32, TimeTicks,
  IpAddress, OID, …) and its **value**.
- **Preset starting points** (chips) for common roots — System, Interfaces
  (ifTable), Interface stats (ifXTable), Processors, Storage, IP addresses — so
  the OID field can be filled without memorizing numbers.
- A walk of a leaf OID returns just that value (covers single-OID lookups; no
  separate Get mode).
- Bounded: results are **capped** (e.g. 5000 OIDs) so an over-broad walk can't
  flood the UI; the cap is surfaced.
- Offline, cross-platform. Read-only. The community is masked on input and never
  shown back or written into an error.

## Non-goals (this slice)

- **No SET / writes** — read-only.
- SNMPv3 (v2c only).
- Full **MIB-name resolution** (numeric OID → symbolic name). Rows show numeric
  OIDs; the preset chips carry the human labels. Name translation needs a MIB
  dictionary and is deferred.
- Persisting / exporting results.
- A tree/expandable view — a flat table of the walked subtree this slice.

## Anti-goals

- The community leaking into a row, response, or error.
- An over-broad walk flooding the UI or hanging it.

## Decisions (owner-approved via the mockup)

- **Walk a subtree** is the one action (a leaf walk returns the single value); no
  separate Get.
- **Raw numeric OIDs** in results; preset chips provide labeled roots. Name
  translation deferred.
- Results capped at 5000 OIDs, with a surfaced note.
- Read-only; community masked.

## Shape (what the builder implements)

**Engine** — a typed walk:
- The current `Walk` returns `PDU{OID, Value}` and drops the ASN.1 type. Add a
  `Type` to `PDU` (populated by `conv` from the gosnmp type via a `typeName`
  map; existing callers ignore it) and a `MIBWalk(c, root, max)` that walks,
  formats each value to a display string, caps the result at `max`, and returns
  `[]MIBRow{OID, Type, Value}` (all strings). A fake `Client` drives the tests.

**App** — a thin bound method:
- `MIBWalk(device, community, oid) ([]snmp.MIBRow, error)` — dial, `snmp.MIBWalk`
  with the 5000 cap, close; errors scrubbed of the community. (The frontend reads
  truncation as `rows.length === cap`.)

**Frontend** — a `MibBrowser` tool:
- Device + masked community + OID inputs and a Walk button; a row of preset
  chips that set the OID field; a results table (OID · Type · Value) with the
  type as a small pill and the value truncated with a tooltip; a status line
  ("N OIDs under <root>") and the cap note when hit. A short in-app usage note.
- Joins the "SNMP & monitoring" group.

## Left to the builder

- Exact `MIBRow`/`PDU.Type` types + JSON tags; the `typeName` map's coverage and
  the value formatter (numbers decimal, OctetString as text, nil as empty).
- Whether the cap is enforced post-walk (simplest, given the `Client.Walk` all-at
  -once shape) or mid-walk; post-walk is acceptable for this slice.
- The preset chips' OID roots and labels; empty/scanning/error states.
