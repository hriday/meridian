# MIB browser / walk Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Walk any OID subtree of a device over SNMPv2c and list each OID with its type and value.

**Architecture:** A typed walk in the `snmp` engine (`MIBWalk` — walk, leaf Get-fallback, value formatting incl. hex, result cap), a thin app `MIBWalk` method, and a `MibBrowser` Svelte tool with preset-root chips and a results table.

**Tech Stack:** Go 1.23 + Wails v2, Svelte/TS. Reuses `snmp.Dial`, `snmp.Client.Walk`/`Get`, `conv`/`pduValue`, and the thin app-method + `scrub` pattern.

**Spec:** docs/superpowers/specs/2026-11-10-meridian-mib-browser-design.md

## Global Constraints

- SNMPv2c, **read-only** (no SET). Community masked on input, never shown back, never in a row/response/error — scrub the returned error with `scrub(err.Error(), community)`.
- Bounded: results capped at **5000** OIDs; the cap is surfaced in the UI.
- Offline — no CDN; pure engine under `internal/`. The UI call is async (never blocks the UI thread).

## Review Focus

- **Community never in a row or error.** (Task 1: no community-bearing field on MIBRow; Task 2 app scrubs.)
- **Type preserved**: each value carries its ASN.1 type name (OctetString/Integer/Counter32/Counter64/Gauge32/TimeTicks/IpAddress/OID/…). (Task 1 `typeName` + walk tests.)
- **Leaf OID**: walking a full instance OID that has no subtree returns the single value (Get-fallback), not an empty table. (Task 1 test.)
- **Non-printable OctetString** (e.g. a MAC/phys-address) renders as hex, not mojibake. (Task 1 test.)
- **Cap**: a walk exceeding 5000 rows is truncated to 5000 and the UI says so. (Task 1 test + Task 2 manual check.)

---

### Task 1: Typed MIB walk in the engine

**Files:**
- Modify: `internal/snmp/snmp.go` (add `PDU.Type`, set it in `conv`, add `typeName`)
- Create: `internal/snmp/mibwalk.go`, `internal/snmp/mibwalk_test.go`

**Interfaces:**
- Consumes: `Client.Walk`/`Get`, `conv`/`pduValue`, the gosnmp alias `g`.
- Produces:
  - `PDU` gains `Type string` (set by `conv` via `typeName(v.Type)`; existing callers ignore it).
  - `func typeName(t g.Asn1BER) string` — maps the common BER types to names; default `"—"`.
  - `type MIBRow struct { OID string \`json:"oid"\`; Type string \`json:"type"\`; Value string \`json:"value"\` }`
  - `func MIBWalk(c Client, root string, max int) ([]MIBRow, error)`

- [ ] **Step 1: Write the failing tests** in `internal/snmp/mibwalk_test.go` using the existing `fakeClient{get,walk}` (construct PDUs with `Type` set). Also a `typeName` unit test.

```go
func TestTypeName(t *testing.T) // g.Counter32→"Counter32", g.TimeTicks→"TimeTicks", g.OctetString→"OctetString", unknown→"—"
func TestMIBWalkTypesAndValues(t *testing.T) // walk returns 3 PDUs (OctetString "core-sw-1", TimeTicks uint(1850), Counter32 uint(99)); MIBRows carry oid+type+value strings in order
func TestMIBWalkLeafGetFallback(t *testing.T) // walk(root)=empty, get(root)=a value → exactly one MIBRow with that value
func TestMIBWalkHexNonPrintable(t *testing.T) // an OctetString value containing non-printable bytes (e.g. "\x00\xde\xad") → a hex string, not raw
func TestMIBWalkCaps(t *testing.T) // walk returns 10 PDUs, max=4 → 4 rows
```

- [ ] **Step 2: Run, verify fail** — `go test ./internal/snmp/ -run 'TestMIBWalk|TestTypeName'` → FAIL (undefined).

- [ ] **Step 3: Implement.** Add `Type string` to `PDU`; in `conv` set `Type: typeName(v.Type)`. Write `typeName` (switch over `g.OctetString`, `g.Integer`, `g.Counter32`, `g.Counter64`, `g.Gauge32`, `g.TimeTicks`, `g.IPAddress`, `g.ObjectIdentifier`, `g.Null`, `g.NoSuchObject`/`Instance`/`g.EndOfMibView`; default `"—"`). In `mibwalk.go`: `MIBWalk` calls `c.Walk(root)`; if it returns zero rows, `c.Get([]string{root})` and keep the single PDU if its value is non-nil; map each PDU to `MIBRow{OID, Type, Value: formatMIBValue(pdu.Value)}`; cap the slice at `max`. `formatMIBValue(any) string`: `nil`→`""`; `string`→ printable as-is else `hex.EncodeToString([]byte(s))`; everything else → `fmt.Sprint(v)`. (Printable = no `unicode.IsControl` runes other than none; treat any control byte as non-printable.)

- [ ] **Step 4: Run, verify pass** — `go test -race ./internal/snmp/ -run 'TestMIBWalk|TestTypeName'` → PASS. Full `go test ./...` green (the `PDU.Type` addition must not break existing snmp/app tests).

- [ ] **Step 5: Commit** — `feat(snmp): typed MIB walk (OID, type, value)`.

---

### Task 2: App method + MIB browser tool

**Files:**
- Create: `internal/app/mibwalk_methods.go`, `internal/app/mibwalk_methods_test.go`
- Create: `frontend/src/lib/tools/MibBrowser.svelte`
- Modify: `frontend/src/lib/Workbench.svelte`
- Regenerate: `frontend/wailsjs` bindings

**Interfaces:**
- Consumes: `snmp.MIBWalk`, `snmp.MIBRow` (Task 1), `scrub`, `dial` (the app `dial` seam used by the other SNMP methods). Frontend: the regenerated `MIBWalk` binding, `snmp.MIBRow`.
- Produces: `func (a *App) MIBWalk(device, community, oid string) ([]snmp.MIBRow, error)`; a tool registered as `{ id: "mib", name: "MIB browser", group: "SNMP & monitoring", ready: true }`.

- [ ] **Step 1: Write the failing app test** `internal/app/mibwalk_methods_test.go` using the existing `setDial`/`fakeSNMP` helpers (a fake whose Walk returns a couple of typed PDUs):
```go
func TestMIBWalkReturnsRows(t *testing.T)        // setDial(fake with 2 walk PDUs); New().MIBWalk("10.0.0.1","public","1.3.6.1.2.1.1") → 2 rows
func TestMIBWalkDialErrorScrubbed(t *testing.T)  // setDial(nil, err containing the community) → error without the community
```
(Use `const mibCap = 5000`; the app passes it to `snmp.MIBWalk`.)

- [ ] **Step 2: Run, verify fail** — `go test ./internal/app/ -run TestMIBWalk` → FAIL (undefined / dial seam).

- [ ] **Step 3: Implement** `MIBWalk` in `internal/app/mibwalk_methods.go`: trim `oid` and strip a leading `.`; if empty → error "enter an OID"; `c, err := dial(device, community, snmpTimeout)` (scrub err); `defer c.Close()`; `rows, err := snmp.MIBWalk(c, oid, mibCap)` (scrub err); return rows.

- [ ] **Step 4: Run, verify pass** — `go test ./internal/app/ -run TestMIBWalk` → PASS. Full `go test ./...` green.

- [ ] **Step 5: Build `MibBrowser.svelte`.** Device + masked community + OID inputs (default `1.3.6.1.2.1.1`), a Walk button, and a row of preset chips that set the OID field: System `1.3.6.1.2.1.1`, Interfaces `1.3.6.1.2.1.2.2`, Interface stats `1.3.6.1.2.1.31.1.1`, Processors `1.3.6.1.2.1.25.3.3`, Storage `1.3.6.1.2.1.25.2.3`, IP addresses `1.3.6.1.2.1.4.20`. On Walk: `rows = await MIBWalk(device, community, oid)` (await; show a "walking…" state); on throw show an error banner. Render the results table (OID · Type pill · Value with tooltip + ellipsis), a status line ("N OIDs under <oid>"), and — when `rows.length === 5000` — a "capped at 5000" note. A short in-app usage note. `active` chip reflects the current OID.

- [ ] **Step 6: Wire Workbench** — import `MibBrowser`, add the tools[] entry in "SNMP & monitoring", add the `{:else if active === "mib"}<MibBrowser />` branch.

- [ ] **Step 7: Regenerate bindings** — `$(go env GOPATH)/bin/wails generate module`, then `git checkout -- frontend/wailsjs/runtime`. Confirm `MIBWalk` in `App.d.ts` and `snmp.MIBRow` in `models.ts`.

- [ ] **Step 8: Build** — `cd frontend && npm run build` → green.

- [ ] **Step 9: Manual check** (dev server + Playwright): stub `window.go.app.App.MIBWalk` to return a set of rows (varied types) incl. a 5000-row case; walk; confirm the table shows OID/type/value, a preset chip fills the OID and re-walks, the cap note shows at 5000, and the error state works. Confirm another SNMP tool still renders (regression).

- [ ] **Step 10: Commit** — `feat(app,ui): MIB browser tool`.

---

## Self-Review notes

- Spec coverage: Task 1 = typed walk + leaf fallback + hex + cap; Task 2 = app method + tool (chips, table, cap note, in-app note). Non-goals (SET, v3, name resolution, export, tree view) are out.
- Review Focus: each line maps to a Task 1 test or the Task 2 manual check.
- Types: `MIBRow` JSON tags (oid/type/value) match the tool's reads; `MIBWalk` returns `[]snmp.MIBRow` consumed by the tool.
