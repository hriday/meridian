# BGP & looking glass (slice 1) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A "BGP & looking glass" tool that shows the control-plane view of a prefix/IP — origin AS(es), AS-path analysis, upstreams/transit, collector/vantage list, and RPKI — from public RIS route collectors via RIPEstat.

**Architecture:** A new source layer `ripeLookingGlass` in `internal/lookup` parses the real `looking-glass` JSON into collectors+peers; `LookupBGP` aggregates those into a `BGPReport` (ranked paths, origin/upstream/transit sets, visibility counts, RPKI via the existing `ripeRPKI`). A CLI `bgp` command and an `App.LookupBGP` binding expose it; a `Bgp.svelte` tool renders it with plain-language captions and AS/prefix pivots, wired into the Workbench's existing `bgp` stub.

**Tech Stack:** Go 1.23, `net/netip`, stdlib `encoding/json`/`strings`; Wails v2 + Svelte/TS frontend. No new dependencies — reuse `lookup.Client`, `ripeBase`, `ripeRPKI`, the entity-page/`requestedTool` nav store.

**Spec:** `docs/superpowers/specs/2026-11-09-meridian-bgp-slice1-design.md`

## Global Constraints
- Go directive stays **1.23**; **no new dependencies**.
- Reuse `lookup.Client` (injected `Doer`, URL cache), `ripeBase`, and `ripeRPKI`.
- Outbound requests carry the generic `meridian/0.1` User-Agent (already set in `Client.GetJSON`); never the user's identity.
- Partial failure degrades a card, never the whole report: per-section `Source{Label, FetchedAt, Err}`.
- Plain-language standard (NTP screen): every metric gets a one-line caption; the screen leads with a verdict sentence.
- **ASN tokens in `BGPReport` are stored as plain decimal strings** (e.g. `"2914"`, no `"AS"` prefix) everywhere — paths, origins, upstreams, transit — and the UI prepends `AS` for display and pivots. (This differs deliberately from `PrefixReport.OriginASN`, which is `"AS"`-prefixed; a path is a uniform list of tokens.)
- AS paths read **left→right as seen→origin**: the last element is the origin, the second-to-last is that peer's view of the origin's upstream.

## Review Focus
- **Resource not seen by any collector** (bogon, unannounced, typo'd prefix): RIPEstat returns `rrcs: []` or collectors with no peers → `LookupBGP` must return a valid report with `Collectors: 0`, empty paths, and no divide-by-zero on percent. (Task 2, `TestLookupBGPNotSeen`.)
- **MOAS** (more than one distinct `asn_origin`): `OriginASNs` has >1 entry and `MOAS` is true; RPKI uses the first origin only. (Task 2, `TestLookupBGPMOAS`.)
- **Short / origin-only paths** (`as_path` of length 1, or a 2-element `[upstream, origin]`): no upstream for length 1; empty transit for length 2; no index-out-of-range. (Task 2, `TestLookupBGPPathBuckets`.)
- **RPKI source fails** while routing succeeds: `RPKISrc.Err` set, `RPKIState` empty, routing data still present. (Task 2, `TestLookupBGPRPKIDegrades`.)
- **Duplicate paths across peers**: identical `as_path` from many peers collapses to one `BGPPath` with the summed peer count and correct percent. (Task 2, `TestLookupBGPAggregatesPaths`.)

---

### Task 1: looking-glass source parse

**Files:**
- Create: `internal/lookup/bgp.go`
- Test: `internal/lookup/bgp_test.go`

**Interfaces:**
- Consumes: `Client.GetJSON`, `ripeBase` (both in `internal/lookup`).
- Produces:
  ```go
  type lgPeer struct {
      OriginASN   string   // asn_origin, plain decimal
      ASPath      []string // as_path split on spaces, seen→origin
      NextHop     string
      Origin      string   // IGP/EGP/…
      Peer        string
      Prefix      string
      LastUpdated string
  }
  type lgCollector struct {
      RRC      string
      Location string
      Scope    string
      Peers    []lgPeer
  }
  func (c *Client) ripeLookingGlass(ctx context.Context, resource string) ([]lgCollector, error)
  ```
  URL: `fmt.Sprintf("%slooking-glass/data.json?resource=%s", ripeBase, url.QueryEscape(resource))`.

- [ ] **Step 1: Write the failing test**

In `bgp_test.go`, serve a trimmed real fixture (two collectors; the second with two peers sharing an origin) at path `/looking-glass/` via the existing `lookupServer(t, map[string]srvResp{...})` helper. Fixture peers use real field names (`asn_origin`, `as_path`, `rrc`, `location`, `scope`, `next_hop`, `origin`, `peer`, `prefix`, `last_updated`).
```go
func TestRipeLookingGlass(t *testing.T) {
    c := lookupServer(t, map[string]srvResp{"/looking-glass/": {body: lgBody}})
    cols, err := c.ripeLookingGlass(context.Background(), "140.78.0.0/16")
    require.NoError(t, err)
    require.Len(t, cols, 2)
    require.Equal(t, "RRC01", cols[0].RRC)
    require.Equal(t, "London, GB", cols[0].Location)
    require.Equal(t, []string{"15692", "2914", "1764", "1853", "1205"}, cols[1].Peers[0].ASPath)
    require.Equal(t, "1205", cols[1].Peers[0].OriginASN)
}
```
(`lgBody` is a const in the test file holding the trimmed JSON.)

- [ ] **Step 2: Run test to verify it fails** — `go test ./internal/lookup/ -run TestRipeLookingGlass -v`. Expected: FAIL (undefined `ripeLookingGlass`).

- [ ] **Step 3: Implement `ripeLookingGlass` in `bgp.go`** — decode `data.rrcs[]` into an internal JSON struct, split `as_path` on whitespace (`strings.Fields`), and map into `[]lgCollector`. A peer with an empty `as_path` yields an empty `ASPath` slice.

- [ ] **Step 4: Run test to verify it passes** — same command. Expected: PASS.

- [ ] **Step 5: Commit**
```bash
git add internal/lookup/bgp.go internal/lookup/bgp_test.go
git commit -m "feat(lookup): parse RIPEstat looking-glass collectors and peers"
```

---

### Task 2: LookupBGP aggregation

**Files:**
- Create: `internal/lookup/resolve_bgp.go`
- Test: `internal/lookup/resolve_bgp_test.go`

**Interfaces:**
- Consumes: `ripeLookingGlass` (Task 1), `ripeRPKI` (existing), `Source`, `now()`.
- Produces:
  ```go
  type BGPPath struct {
      Path    []string `json:"path"`    // plain-decimal ASNs, seen→origin
      Peers   int      `json:"peers"`
      Percent int      `json:"percent"` // round(Peers/totalPeers*100)
  }
  type BGPCollector struct {
      RRC      string `json:"rrc"`
      Location string `json:"location"`
      Scope    string `json:"scope"`
      Peers    int    `json:"peers"`
  }
  type BGPReport struct {
      Input      string         `json:"input"`
      Prefix     string         `json:"prefix"`      // most common peer.Prefix
      OriginASNs []string       `json:"originAsns"`  // distinct, plain decimal, sorted
      MOAS       bool           `json:"moas"`
      Collectors int            `json:"collectors"`  // RRCs with >=1 peer
      Peers      int            `json:"peers"`       // total peers seeing it
      Paths      []BGPPath      `json:"paths"`       // ranked desc
      Upstreams  []string       `json:"upstreams"`   // distinct, plain decimal, sorted
      Transit    []string       `json:"transit"`     // distinct, plain decimal, sorted
      RPKIState  string         `json:"rpkiState"`
      Vantages   []BGPCollector `json:"vantages"`
      RoutingSrc Source         `json:"routingSrc"`
      RPKISrc    Source         `json:"rpkiSrc"`
  }
  func (c *Client) LookupBGP(ctx context.Context, input string) (BGPReport, error)
  ```

**Aggregation rules (pin exactly):**
- 20 s overall deadline (`context.WithTimeout`), like `LookupPrefix`.
- `RoutingSrc = Source{Label: "RIPEstat looking-glass", FetchedAt: now()}`; on `ripeLookingGlass` error set `RoutingSrc.Err` and return the report (no RPKI call).
- `totalPeers` = sum of `len(col.Peers)`. `Collectors` = count of collectors with ≥1 peer. If `totalPeers == 0`: return with empty paths/sets, `Percent` never computed (no division).
- `OriginASNs` = distinct non-empty `peer.OriginASN`, sorted. `MOAS = len(OriginASNs) > 1`.
- `Prefix` = the most frequently occurring non-empty `peer.Prefix`.
- `Paths`: group peers by `strings.Join(ASPath, " ")`, count peers per group; one `BGPPath{Path, Peers, Percent}` each; `Percent = int(math.Round(float64(Peers)/float64(totalPeers)*100))`; sort desc by `Peers`, tie-break by the joined path string.
- Bucketing (disjoint), over `originSet = set(OriginASNs)`:
  - `Upstreams` = distinct `path[len-2]` for every path with `len>=2` whose `path[len-2]` ∉ `originSet`.
  - `Transit` = distinct `path[i]` for `i` in `0..len-3` (the positions before the upstream) across all paths, excluding any AS in `originSet` or in `Upstreams`.
  - both sorted (numeric-aware or lexical string sort is fine — pick one and be consistent).
- RPKI: if `len(OriginASNs) >= 1` and `Prefix != ""`, set `RPKISrc = Source{Label:"RIPEstat RPKI", FetchedAt: now()}` and call `ripeRPKI(ctx, Prefix, OriginASNs[0])`; on error set `RPKISrc.Err`, else set `RPKIState` (the `state`).
- `Vantages`: one `BGPCollector{RRC, Location, Scope, Peers: len(col.Peers)}` per collector with ≥1 peer, sorted desc by `Peers`, tie-break by `RRC`.

- [ ] **Step 1: Write the failing tests** — in `resolve_bgp_test.go`, drive `LookupBGP` through `lookupServer` serving `/looking-glass/` and `/rpki-validation/`:
  - `TestLookupBGPAggregatesPaths`: two collectors, a path shared by 3 peers and another by 1 → `Paths[0].Peers==3`, `Percent==75`, `Collectors==2`, `Peers==4`.
  - `TestLookupBGPPathBuckets`: a path `[3356,2914,1764,1205]` with origin `1205` → `Upstreams` contains `1764`, `Transit` contains `3356` and `2914`, none contains `1205`; a `[6939,1205]` path adds `6939` to upstreams; an origin-only `[1205]` path adds nothing and does not panic.
  - `TestLookupBGPMOAS`: two distinct origins → `len(OriginASNs)==2`, `MOAS` true, RPKI queried for the first.
  - `TestLookupBGPNotSeen`: `rrcs: []` → `Collectors==0`, `Peers==0`, empty `Paths`, no error, no RPKI call.
  - `TestLookupBGPRPKIDegrades`: looking-glass ok, `/rpki-validation/` returns 500 → `RPKISrc.Err` non-empty, `RPKIState==""`, `Paths` populated.

- [ ] **Step 2: Run to verify they fail** — `go test ./internal/lookup/ -run TestLookupBGP -v`. Expected: FAIL (undefined `LookupBGP`).

- [ ] **Step 3: Implement `BGPReport`, `BGPPath`, `BGPCollector`, and `LookupBGP`** in `resolve_bgp.go` per the rules above.

- [ ] **Step 4: Run to verify they pass** — same command. Expected: PASS.

- [ ] **Step 5: Commit**
```bash
git add internal/lookup/resolve_bgp.go internal/lookup/resolve_bgp_test.go
git commit -m "feat(lookup): LookupBGP aggregates paths, origins, upstreams, transit, RPKI"
```

---

### Task 3: CLI `bgp` command + App binding

**Files:**
- Modify: `internal/cli/cli.go` (add `bgp` subcommand + `printBGPReport`)
- Create: `internal/app/bgp_methods.go`
- Test: `internal/cli/cli_bgp_test.go`, `internal/app/bgp_methods_test.go`

**Interfaces:**
- Consumes: `lookup.BGPReport`, `(*lookup.Client).LookupBGP`; `App.lookup`, `App.lookupCtx` (existing).
- Produces: `App.LookupBGP(input string) (lookup.BGPReport, error)`; CLI route for `bgp <target>`.

- [ ] **Step 1: Write failing tests**
  - `cli_bgp_test.go` — `TestRunBGPPrintsSummary`: run the `bgp` command against a `lookupServer`-backed client, capture stdout, assert it contains the origin AS, the collector/peer visibility line, and the top path. Follow the existing cli test harness used by `cli_lookup2_test.go` (same server/injection pattern).
  - `bgp_methods_test.go` — `TestAppLookupBGP`: construct an `App` with a looking-glass-backed `lookup.Client` (mirror `lookup2_methods_test.go`), call `LookupBGP`, assert `Collectors>0` and a non-empty origin.

- [ ] **Step 2: Run to verify they fail** — `go test ./internal/cli/ ./internal/app/ -run BGP -v`. Expected: FAIL.

- [ ] **Step 3: Implement**
  - In `cli.go`: add a `bgp` case to the command switch calling `c.LookupBGP(ctx, target)` and `printBGPReport(w, r)`. `printBGPReport` prints: a header line (`BGP view of <Prefix>`), origin(s) with a `MOAS` note when set, a visibility line (`seen by <Peers> peers across <Collectors> collectors · <len(Paths)> distinct paths`), RPKI state when present, the top N (≤5) paths as `ASx → ASy → … (peers, pct%)`, and upstream/transit summaries. Mirror the formatting helpers used by `printPrefixReport`.
  - In `bgp_methods.go`: `func (a *App) LookupBGP(input string) (lookup.BGPReport, error) { return a.lookup.LookupBGP(a.lookupCtx(), input) }`.

- [ ] **Step 4: Run to verify they pass** — same command, then `go test ./...`. Expected: PASS.

- [ ] **Step 5: Commit**
```bash
git add internal/cli/cli.go internal/cli/cli_bgp_test.go internal/app/bgp_methods.go internal/app/bgp_methods_test.go
git commit -m "feat(cli,app): bgp command and LookupBGP binding"
```

---

### Task 4: Bgp.svelte tool + Workbench wiring

**Files:**
- Create: `frontend/src/lib/tools/Bgp.svelte`
- Modify: `frontend/src/lib/Workbench.svelte` (import, route branch, flip `bgp` `ready: true`)
- Modify (regenerated): `frontend/wailsjs/go/app/App.js`, `frontend/wailsjs/go/app/App.d.ts` (add `LookupBGP` binding — produced by `wails build`/`wails generate module`)

**Interfaces:**
- Consumes: `App.LookupBGP` (Task 3) via the generated binding; the `requestedTool` nav store for pivots (as `Lookup.svelte`/`EntityPage.svelte` use it).

> **Note on method:** no frontend unit-test harness exists (consistent with every prior slice); this task is verified by a clean `wails build` (frontend compiles, bindings regenerate) plus a manual launch. TDD's "write the failing test first" is waived here per its config/generated-code exception — stated in the ledger at task start.

- [ ] **Step 1: Regenerate bindings** — run `~/go/bin/wails generate module` (or a `wails build`) so `App.js`/`App.d.ts` gain `LookupBGP`. Verify: `grep LookupBGP frontend/wailsjs/go/app/App.js`.

- [ ] **Step 2: Build `Bgp.svelte`** — an input (prefix or IP; placeholder `104.16.0.0/13 · 1.1.1.1`), a lookup button, and on result:
  - a **verdict line** in plain words (e.g. "Announced by AS13335, seen by 42 peers across 20 collectors — widely visible." / "Not seen by any collector." / a MOAS warning when `moas`).
  - per-metric **captions** explaining peers/collectors/paths and the RPKI state (reuse the RPKI pill classes: valid→good, invalid→bad, unknown→soon).
  - an **AS-path table** (top paths; each AS a pivot chip to its ASN page via `requestedTool`), **upstream** and **transit** chip rows (AS pivots), and a **searchable vantage list** (filter box over `vantages` by RRC/location; each row `RRC · location · scope · N peers`).
  - the prefix pivots to the prefix page; include the existing "Traceroute" / "Save to IP space" actions where they fit.
  - follow the theme tokens and layout conventions of `EntityPage.svelte`; no horizontal overflow at phone width.

- [ ] **Step 3: Wire the Workbench** — import `Bgp`, add `{:else if active === "bgp"}<Bgp />`, and set the `bgp` tool entry `ready: true`.

- [ ] **Step 4: Verify** — `~/go/bin/wails build` succeeds; launch and confirm a real lookup (`104.16.0.0/13`) renders paths, a vantage list, and the RPKI pill; filtering the vantage list works; an AS chip pivots to its ASN page. Capture nothing to commit beyond source + regenerated bindings.

- [ ] **Step 5: Commit**
```bash
git add frontend/src/lib/tools/Bgp.svelte frontend/src/lib/Workbench.svelte frontend/wailsjs/go/app/App.js frontend/wailsjs/go/app/App.d.ts
git commit -m "feat(ui): BGP & looking glass tool"
```

---

## Self-review notes
- **Spec coverage:** origin/visibility/RPKI (T2 report + T4 header), AS-path analysis incl. upstreams/transit (T2 buckets + T4 table/chips), searchable collector picker (T4 vantage list), CLI (T3), own-tool shape (T4 Workbench). Atlas + history are explicit non-goals.
- **Type consistency:** `ripeLookingGlass → []lgCollector` (T1) is the sole input to `LookupBGP` (T2); `BGPReport` is consumed unchanged by CLI (T3) and UI (T4). ASN tokens are plain-decimal throughout the report, `AS`-prefixed only at display — stated in Global Constraints and repeated on each field.
- **Review Focus:** all five rows map to named Task 2 tests.
- **Proportion:** four tasks; engine logic is the bulk, UI is wiring + presentation. Plan pins types, URLs, and aggregation math (the lookups-2 Critical came from unpinned/invented shapes — here the shape is verified live and copied verbatim).
