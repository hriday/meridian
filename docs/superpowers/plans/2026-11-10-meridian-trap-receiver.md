# SNMP trap receiver Implementation Plan (slice 2 of 2)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** An SNMP trap receiver reusing the collector framework — listen on a UDP port, decode v1/v2c traps, live-tail them with a filter.

**Architecture:** A `snmp.ParseTrap` parse function (raw datagram → `snmp.TrapMsg` via gosnmp's `SnmpDecodePacket`, reusing `typeName`/`formatMIBValue`), plugged into a second singleton `collector.Collector[snmp.TrapMsg]` on `App` with `TrapStart/Stop/Poll`; a `TrapReceiver` Svelte tool reusing the shared `LiveTail`.

**Spec:** docs/superpowers/specs/2026-11-10-meridian-collectors-design.md (the traps section)

## Global Constraints

- SNMPv2c/v1 traps. The packet's **community is never copied** onto the message or shown. Default port **1162**, editable, bind 0.0.0.0.
- Reuses the shipped framework: ephemeral 10k ring, poll-since-cursor, auto-stop on shutdown, defensive (panic-free) parse of untrusted datagrams.

## Review Focus

- **Community never on the message or in any error.** (Task 1 test marshals a parsed trap and asserts no community.)
- **Malformed/non-SNMP datagram** → `ok=false` (dropped), never a panic. (Task 1 test.)
- **Trap OID extraction**: v2c → the value of the `snmpTrapOID.0` varbind; v1 → from the generic/specific trap; a well-known trap gets a name. (Task 1 tests.)
- **No leak / lifecycle**: reuses the reviewed Collector (Start/Stop/epoch), so only the app wiring + OnShutdown stop are new. (Task 2 test.)

---

### Task 1: `snmp.ParseTrap`

**Files:** Create `internal/snmp/trap.go`, `internal/snmp/trap_test.go`

**Interfaces:**
- Consumes: gosnmp `SnmpDecodePacket`/`SnmpEncodePacket` (tests build a trap), `typeName`, `formatMIBValue`.
- Produces:
  - `type TrapVar struct { OID string \`json:"oid"\`; Type string \`json:"type"\`; Value string \`json:"value"\` }`
  - `type TrapMsg struct { T time.Time \`json:"t"\`; Source string \`json:"source"\`; Version string \`json:"version"\`; TrapOID string \`json:"trapOid"\`; TrapName string \`json:"trapName"\`; Varbinds []TrapVar \`json:"varbinds"\` }`
  - `func ParseTrap(datagram []byte, src netip.Addr) (TrapMsg, bool)`

- [ ] **Step 1: Failing tests.** Build a trap datagram with gosnmp (`(&g.GoSNMP{Version: g.Version2c}).SnmpEncodePacket(g.SNMPv2Trap, pdus, 0, 0)`), where pdus include `1.3.6.1.6.3.1.1.4.1.0` (snmpTrapOID.0) = the linkDown OID `1.3.6.1.6.3.1.1.5.3` and a data varbind.
```go
func TestParseTrapV2c(t)          // decode → Version "v2c", TrapOID == "1.3.6.1.6.3.1.1.5.3", TrapName "linkDown", Varbinds non-empty, Source == src
func TestParseTrapNoCommunity(t)  // encode with Community "s3cr3t"; the parsed TrapMsg JSON has no "s3cr3t"
func TestParseTrapGarbage(t)      // random/empty bytes → ok=false, no panic
```
- [ ] **Step 2: Run, verify fail.**
- [ ] **Step 3: Implement.** `ParseTrap`: `pkt, err := (&g.GoSNMP{}).SnmpDecodePacket(datagram)`; `err != nil || pkt == nil` → ok=false. Version from `pkt.Version`. For each `pkt.Variables` → `TrapVar{v.Name, typeName(v.Type), formatMIBValue(pduValue(v))}` (bound/hex via the existing formatter). TrapOID: the value of the `1.3.6.1.6.3.1.1.4.1.0` varbind (an OID string) for v2c; for a v1 trap PDU, synthesize from `pkt.GenericTrap`/`SpecificTrap`/`Enterprise`. `TrapName` via a small well-known map (coldStart .5.1, warmStart .5.2, linkDown .5.3, linkUp .5.4, authenticationFailure .5.5). **Never read `pkt.Community` onto the message.** Guard every field access (a short varbind list, a missing snmpTrapOID → empty TrapOID, not a panic).
- [ ] **Step 4: Run with race, verify pass.** Full `go test ./...` green.
- [ ] **Step 5: Commit** — `feat(snmp): v1/v2c trap parser`.

---

### Task 2: App trap methods + TrapReceiver tool

**Files:** Create `internal/app/trap_methods.go`, `internal/app/trap_methods_test.go`, `frontend/src/lib/tools/TrapReceiver.svelte`; modify `internal/app/app.go` (a `traps *collector.Collector[snmp.TrapMsg]` field + init + OnShutdown stop), `frontend/src/lib/Workbench.svelte`; regenerate bindings.

**Interfaces:**
- Produces: `func (a *App) TrapStart(port int) error`, `TrapStop()`, `TrapPoll(afterSeq int) collector.Poll[snmp.TrapMsg]`; a tool `{ id: "traps", name: "Trap receiver", group: "Collectors", ready: true }`.

- [ ] **Step 1: Failing app test** `trap_methods_test.go`: `TestTrapStartPollStop` — `TrapStart(0)`; encode a v2c trap (as in Task 1) and send it UDP to the polled port; `Eventually` `TrapPoll(0).Items` has it with the right TrapOID; `TrapStop()` → running false. `TestTrapStartBindInUse` — held port → error.
- [ ] **Step 2: Run, verify fail.**
- [ ] **Step 3: Implement.** In `app.go` add `traps: collector.New[snmp.TrapMsg](collectorBuffer, snmp.ParseTrap)`; `OnShutdown` also `a.traps.Stop()`. The three methods delegate.
- [ ] **Step 4: Run with race, verify pass.** Full `go test ./...` green.
- [ ] **Step 5: Build `TrapReceiver.svelte`** — same structure as `SyslogReceiver` (poll-since-cursor ~700ms, epoch reset, restore on mount, Stop clears) but port default **1162**, no severity floor, and trap-shaped rows via the `LiveTail` slot: `time · source · trap name/OID · varbinds` (join `oid=value` pairs, truncated with a title tooltip). An in-app usage note. Filter matches source + trap name/OID + varbinds text.
- [ ] **Step 6: Wire Workbench** — import, add `{ id: "traps", … group: "Collectors" }`, add the render branch.
- [ ] **Step 7: Regenerate bindings; build** (`npm run build` green). Confirm `TrapStart/Stop/Poll` + `snmp.TrapMsg` in the bindings.
- [ ] **Step 8: Manual check** (dev server + Playwright, real send): `TrapStart` on 1162; send a real v2c trap (a small Go/python sender, or gosnmp `SendTrap`) to 127.0.0.1:1162; confirm the row shows the trap name/OID + varbinds, filter works, restore-across-switch, Stop clears; regression on the Syslog receiver. (Fallback: stub `TrapPoll` if a real send is impractical; note which.)
- [ ] **Step 9: Commit** — `feat(app,ui): SNMP trap receiver tool`.

---

## Self-Review notes
- Spec coverage: Task 1 = the trap parse (the only new engine logic); Task 2 = app singleton + tool reusing the reviewed framework + LiveTail. Lifecycle/concurrency is already reviewed (slice 1).
- Review Focus: community-redaction + garbage-safety + OID extraction on Task 1; lifecycle on Task 2.
- Types: `collector.Poll[snmp.TrapMsg]` consumed by the tool; `TrapMsg`/`TrapVar` JSON tags match the row fields.
