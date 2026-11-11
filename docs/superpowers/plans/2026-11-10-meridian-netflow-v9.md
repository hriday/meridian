# NetFlow v9/IPFIX (slice 2 of 2) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans.

**Goal:** Decode NetFlow v9 and IPFIX (template-based) into the existing `FlowRecord`, behind the version dispatch, so the top-talkers view covers modern exporters.

**Architecture:** Add `github.com/netsampler/goflow2/v2` (owner-approved) as the vetted template decoder. A stateful `netflow.Parser` holds a per-exporter template cache; its `Parse(datagram, exporter, now) []FlowRecord` dispatches v5 → `ParseV5`, v9/IPFIX → goflow2 `DecodeMessageVersion`, then maps decoded `DataField`s (by IANA IE type) into `FlowRecord`s. The app's collector uses `parser.Parse` instead of the v5-only func. No view/aggregation changes.

**Spec:** docs/superpowers/specs/2026-11-10-meridian-netflow-design.md (slice-2 note)

## Global Constraints

- Defensive against untrusted UDP: the whole v9/IPFIX decode is `recover()`-guarded (→ no records), on top of goflow2's own bounds. A datagram whose template hasn't been seen yet yields no records (goflow2 skips that flowset), not an error.
- The template cache is accessed only from the collector's single read-loop goroutine (no lock needed); keyed per exporter.
- Only the top-talker fields are extracted (v4/v6 5-tuple + octet/packet counts); unknown fields ignored. No behavioural change to the aggregation/view.
- goflow2 is imported ONLY via `.../v2/decoders/netflow` (+ `.../decoders/utils`) to avoid the producer/transport dependency weight.

## Review Focus

- **Field extraction correctness**: IE types → FlowRecord (8/27 src, 12/28 dst, 7 sport, 11 dport, 4 proto, 1/85 bytes, 2/86 packets); big-endian variable-length counters; a record missing src/dst is dropped. (Task 1 unit tests.)
- **Decode safety**: malformed/garbage/truncated v9/IPFIX and a data-set-before-template → no panic, no records (not a crash). (Task 1 tests + recover.)
- **No v5 regression**: `Parse` still returns v5 records identically. (Task 1 test reuses the v5 path.)
- **Stateful cache**: a template packet followed by a data packet for the same exporter decodes the data; the cache persists across datagrams. (Task 2 live check.)

---

### Task 1: v9/IPFIX extraction + the Parser

**Files:** Create `internal/netflow/v9.go`, `internal/netflow/v9_test.go`. Modify `go.mod`/`go.sum` (already fetched).

**Interfaces (produced):**
- `type Parser struct { … }` with `func NewParser() *Parser` and `func (p *Parser) Parse(datagram []byte, exporter netip.Addr, now time.Time) []FlowRecord` — the collector's parse func (replaces the v5-only one). Dispatches by `binary.BigEndian.Uint16(datagram[0:2])`: 5 → `ParseV5`; 9/10 → `p.parseTemplated`; else nil.
- `flowFromFields(values []nf.DataField, exporter netip.Addr, now time.Time) (FlowRecord, bool)` — maps a decoded record's fields to a FlowRecord; ok=false if neither src nor dst present.

- [ ] **Step 1: Failing tests.**
```go
func TestFlowFromFields(t)        // build []nf.DataField{ {Type:8,Value:[10,0,0,9]}, {12,[10,0,0,1]}, {7,be16(50000)}, {11,be16(443)}, {4,[6]}, {1,be32(120000)}, {2,be32(100)} } → FlowRecord src/dst/ports/proto/bytes/packets, ok
func TestFlowFromFieldsV6(t)      // Type 27/28 (16-byte) → IPv6 Src/Dst; 8-byte octet/packet counts decode big-endian
func TestFlowFromFieldsDropsNoAddr(t) // no src/dst field → ok=false
func TestParserV5StillWorks(t)    // Parse() on a v5 datagram (buildV5) returns the v5 records (dispatch regression)
func TestParseTemplatedThenData(t)// feed a hand-built v9 template flowset packet (fields 8,12,7,11,4,1,2 / template 256) → 0 records; then a v9 data packet for template 256 → 1 FlowRecord with the expected 5-tuple+counts
func TestParseGarbageTemplated(t) // version 9 header + random body → nil, no panic
```
(A small `v9pkt`/`v9template`/`v9data` byte builder in the test assembles the header + flowsets by hand.)
- [ ] **Step 2–4: fail → implement → pass (race).** `v9.go`: `Parser{templates map[netip.Addr]nf.NetFlowTemplateSystem}`; `Parse` dispatches; `parseTemplated` wraps the body in `recover()` → nil, gets/creates the exporter's `nf.CreateTemplateSystem()`, `nf.DecodeMessageVersion(bytes.NewBuffer(datagram), ts, &nfv9, &ipfix)`, picks the filled packet by version, walks `DataFlowSet` records → `flowFromFields`. `flowFromFields`: switch on `df.Type`, `df.Value.([]byte)`, `netip.AddrFrom4/16`, `beUint` for ports/proto/counts; require src||dst. Full `go test ./...` green; `go build ./...` clean (confirm no heavy transitive dep pulled — only decoders/netflow+utils imported).
- [ ] **Step 5: Commit** — `feat(netflow): v9/IPFIX decode via goflow2`.

---

### Task 2: Wire the Parser into the app

**Files:** Modify `internal/app/app.go` (hold a `*netflow.Parser`, pass `parser.Parse` to `NewMulti`; drop the v5-only `netflowParse`).

- [ ] **Step 1:** Replace `netflowParse` with `netflow.NewParser()` stored on `App`; `netflow: collector.NewMulti[...](netflowBuffer, a.nfParser.Parse)` (construct the parser in `New`, before the collector). Existing `internal/app` netflow tests (v5 via UDP) still pass — regression.
- [ ] **Step 2:** `go test -race ./internal/app/` green; full `go test ./...` green.
- [ ] **Step 3: Manual check** (dev server + Playwright): send a real **v9** packet pair with a python sender (a template flowset defining the 5-tuple+counts, then a data flowset) to :2055; confirm the conversation appears in top talkers (proving the template cache + goflow2 path end-to-end). Also re-send a v5 packet (regression). Status still shows flows/exporters.
- [ ] **Step 4: Commit** — `feat(app): NetFlow collector decodes v9/IPFIX`.

---

## Self-Review notes
- Spec coverage: Task 1 = the only new parsing (v9/IPFIX extraction + dispatch); Task 2 = swap the collector's parse. Aggregation/view/UI unchanged (slice 1).
- Review Focus: extraction + decode-safety + v5-regression on Task 1; stateful cache end-to-end on Task 2.
- Dependency: goflow2 v2 decoder only; go.mod/go.sum committed; build stays buildable cross-platform (verify GOOS=windows/linux).
