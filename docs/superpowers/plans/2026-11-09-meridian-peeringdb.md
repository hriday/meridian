# PeeringDB enrichment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enrich the ASN entity page (engine, CLI, UI) with PeeringDB data — network type/policy, the IXPs a network is present at, and its facilities.

**Architecture:** A new `internal/lookup/peeringdb.go` fetcher reuses the existing `lookup.Client` (Doer + URL cache) against a `peeringDBBase` package-var seam. `LookupASN` gains a best-effort Peering section carrying its own `Source`. CLI `printASNReport` and the Svelte ASN overview render it.

**Tech Stack:** Go 1.23, existing `lookup.Client`/`GetJSON`, httptest fixtures; Svelte/TS frontend.

**Spec:** `docs/superpowers/specs/2026-11-09-meridian-peeringdb-design.md`

## Global Constraints
- Go 1.23; no new dependency; no API key.
- `peeringDBBase = "https://www.peeringdb.com/api/"` — a package var so tests point it at an httptest server (mirror `ripeBase`).
- Requests go through `Client.GetJSON` (generic `meridian/*` User-Agent, no user identity).
- JSON field names copied from the verified-live API only: net → `data[0]` with `id, name, info_type, info_scope, info_ratio, policy_general, irr_as_set, website, info_prefixes4, info_prefixes6, ix_count, fac_count`; netixlan → `data[]` with `name, speed`; netfac → `data[]` with `name, city, country`.
- `ASNReport` JSON stays backward-compatible (new optional fields only).
- Net summary is primary; IX/facility list fetches are best-effort (an error on a list degrades that list, not the lookup, and never the net summary).

## Review Focus
- **ASN not in PeeringDB** (`net?asn=` returns `{"data":[]}`) — the card is absent, the rest of the ASN report is unaffected, no error surfaced. (Task 1 `TestPeeringDBNotInRegistry`, Task 2 wiring.)
- **List endpoint fails while net succeeds** (netixlan/netfac 404/500) — summary + counts still render, lists empty, no lookup error. (Task 1 `TestPeeringDBListErrorDegrades`.)
- **Net endpoint fails** (timeout/5xx) — `PeeringSrc.Err` set, `Peering` nil, ASN report otherwise intact. (Task 1 `TestPeeringDBNetError`, Task 2.)
- **Duplicate IX rows** (same IX name across v4/v6/multiple ports) — collapsed to one entry at the max observed speed. (Task 1 `TestPeeringDBParsesNetAndLists`.)
- **Huge presence** (hundreds of IXPs/facilities, e.g. Cloudflare's 418/224) — engine stores all; the UI caps the visible list with a "+N more". (Task 4.)

---

### Task 1: PeeringDB engine fetch + parse

**Files:**
- Create: `internal/lookup/peeringdb.go`
- Test: `internal/lookup/peeringdb_test.go`

**Interfaces:**
- Consumes: `Client.GetJSON(ctx, url, v)` (existing).
- Produces:
  - `type PeerIX struct { Name string `json:"name"`; Speed int `json:"speed"` }` (Speed = Mbps, max across the net's ports at that IX)
  - `type PeerFac struct { Name, City, Country string }` (json `name`/`city`/`country`)
  - `type Peering struct { Name, Type, Scope, Ratio, Policy, IRRASSet, Website string; V4Prefixes, V6Prefixes, IXCount, FacCount int; IXPs []PeerIX; Facilities []PeerFac }` (json: `name,type,scope,ratio,policy,irrAsSet,website,v4Prefixes,v6Prefixes,ixCount,facCount,ixps,facilities`)
  - `var peeringDBBase = "https://www.peeringdb.com/api/"`
  - `func (c *Client) peeringDB(ctx context.Context, asn string) (*Peering, error)` — `asn` may be "13335" or "AS13335"; returns `(nil, nil)` when no net record exists, `(nil, err)` on net-fetch failure, else the populated `*Peering` (IX/facility list-fetch errors are swallowed, leaving those slices empty).

- [ ] **Step 1: Write the failing tests**

In `peeringdb_test.go`, a helper that serves fixtures by path fragment and swaps `peeringDBBase` (mirror `ripeServer`), then:

```go
func TestPeeringDBParsesNetAndLists(t *testing.T) {
    c := pdbServer(t, map[string]srvResp{
        "net":      {body: `{"data":[{"id":4224,"name":"Cloudflare","info_type":"Content","info_scope":"Global","info_ratio":"Mostly Outbound","policy_general":"Open","irr_as_set":"ARIN::AS13335:AS-CLOUDFLARE","website":"https://www.cloudflare.com","info_prefixes4":80000,"info_prefixes6":5000,"ix_count":2,"fac_count":1}]}`},
        "netixlan": {body: `{"data":[{"name":"Equinix Paris","speed":100000},{"name":"Equinix Paris","speed":200000},{"name":"AMS-IX","speed":400000}]}`},
        "netfac":   {body: `{"data":[{"name":"Equinix DC1","city":"Ashburn","country":"US"}]}`},
    })
    p, err := c.peeringDB(context.Background(), "AS13335")
    require.NoError(t, err)
    require.NotNil(t, p)
    require.Equal(t, "Cloudflare", p.Name)
    require.Equal(t, "Content", p.Type)
    require.Equal(t, "Open", p.Policy)
    require.Equal(t, 80000, p.V4Prefixes)
    require.Equal(t, 2, p.IXCount)
    // dedupe by IX name, max speed; sorted by speed desc
    require.Equal(t, []PeerIX{{Name: "AMS-IX", Speed: 400000}, {Name: "Equinix Paris", Speed: 200000}}, p.IXPs)
    require.Equal(t, []PeerFac{{Name: "Equinix DC1", City: "Ashburn", Country: "US"}}, p.Facilities)
}

func TestPeeringDBNotInRegistry(t *testing.T) {
    c := pdbServer(t, map[string]srvResp{"net": {body: `{"data":[]}`}})
    p, err := c.peeringDB(context.Background(), "AS65000")
    require.NoError(t, err)
    require.Nil(t, p)
}

func TestPeeringDBListErrorDegrades(t *testing.T) {
    c := pdbServer(t, map[string]srvResp{
        "net":      {body: `{"data":[{"id":1,"name":"N","info_type":"NSP","ix_count":5}]}`},
        "netixlan": {code: 500, body: ``},
        "netfac":   {code: 404, body: ``},
    })
    p, err := c.peeringDB(context.Background(), "1")
    require.NoError(t, err)
    require.NotNil(t, p)
    require.Equal(t, "N", p.Name)
    require.Empty(t, p.IXPs)
    require.Empty(t, p.Facilities)
}

func TestPeeringDBNetError(t *testing.T) {
    c := pdbServer(t, map[string]srvResp{"net": {code: 500, body: ``}})
    p, err := c.peeringDB(context.Background(), "AS13335")
    require.Error(t, err)
    require.Nil(t, p)
}
```

`pdbServer` reuses `srvResp` from `resolve_test.go`; match `netixlan`/`netfac` **before** `net` (they contain "net"), e.g. order checks longest-fragment-first or test the full path segment.

- [ ] **Step 2: Run tests, verify they fail** (`go test ./internal/lookup/ -run TestPeeringDB -v`) — Expected: FAIL, `peeringDB` / types undefined.

- [ ] **Step 3: Implement `internal/lookup/peeringdb.go`**

Define the types and `peeringDBBase`. `peeringDB`: strip `AS` prefix → number. GET `peeringDBBase+"net?asn="+num` into a struct with `Data []netRec`; if `len(Data)==0` return `nil,nil`; on error return `nil,err`. Map `Data[0]` into `*Peering`. Then best-effort GET `netixlan?net_id=<id>` and `netfac?net_id=<id>` (ignore their errors). Dedupe IXPs by `name` keeping max `speed`; sort IXPs by speed desc then name; sort facilities by name then city.

- [ ] **Step 4: Run tests, verify they pass** (`go test ./internal/lookup/ -run TestPeeringDB -v`) — Expected: PASS.

- [ ] **Step 5: Commit** (`internal/lookup/peeringdb.go`, `internal/lookup/peeringdb_test.go`).

---

### Task 2: Wire Peering into LookupASN + ASNReport

**Files:**
- Modify: `internal/lookup/resolve.go` (`ASNReport` struct + `LookupASN`)
- Modify: `internal/lookup/resolve_test.go` (add `peeringDBBase` swap to `lookupServer`)

**Interfaces:**
- Consumes: `(*Client).peeringDB` (Task 1).
- Produces: `ASNReport.Peering *Peering `json:"peering,omitempty"`` and `ASNReport.PeeringSrc Source `json:"peeringSrc"``.

- [ ] **Step 1: Write the failing test** — extend the combine test (new test alongside `TestLookupASNCombines`):

```go
func TestLookupASNIncludesPeering(t *testing.T) {
    c := lookupServer(t, map[string]srvResp{
        "routing-status": {body: `{"data":{"announced_space":{"v4":{"prefixes":1},"v6":{"prefixes":1}},"observed_neighbours":1}}`},
        "as-overview":    {body: `{"data":{"holder":"CLOUDFLARENET, US"}}`},
        "/autnum/":       {body: rdapASNBody}, // existing fixture if present; else minimal
        "netixlan":       {body: `{"data":[{"name":"AMS-IX","speed":400000}]}`},
        "netfac":         {body: `{"data":[{"name":"Equinix DC1","city":"Ashburn","country":"US"}]}`},
        "net":            {body: `{"data":[{"id":4224,"name":"Cloudflare","info_type":"Content","policy_general":"Open","ix_count":1,"fac_count":1}]}`},
    })
    r, err := c.LookupASN(context.Background(), "AS13335")
    require.NoError(t, err)
    require.NotNil(t, r.Peering)
    require.Equal(t, "Open", r.Peering.Policy)
    require.Equal(t, "PeeringDB", r.PeeringSrc.Label)
    require.Empty(t, r.PeeringSrc.Err)
}
```

Add `peeringDBBase` to the bases `lookupServer` swaps and restores. Confirm the `net` fragment doesn't collide with existing fixtures (RDAP autnum path is `/autnum/`, RIPE paths differ — safe).

- [ ] **Step 2: Run test, verify it fails** — Expected: FAIL, `r.Peering`/`PeeringSrc` undefined.

- [ ] **Step 3: Implement** — add the two fields to `ASNReport`; in `LookupASN`, after the RDAP block, set `r.PeeringSrc = Source{Label: "PeeringDB", FetchedAt: now()}` and call `peeringDB`: on error set `r.PeeringSrc.Err`, else `r.Peering = p` (p may be nil when not in registry — leave Err empty).

- [ ] **Step 4: Run tests, verify they pass** (`go test ./internal/lookup/ -v`) — Expected: PASS (including unchanged ASN tests).

- [ ] **Step 5: Commit** (`internal/lookup/resolve.go`, `internal/lookup/resolve_test.go`).

---

### Task 3: CLI peering summary

**Files:**
- Modify: `internal/cli/cli.go` (`printASNReport`)
- Test: `internal/cli/cli_test.go` (add a case)

**Interfaces:**
- Consumes: `lookup.ASNReport.Peering`, `.PeeringSrc`.

- [ ] **Step 1: Write the failing test** — build an `ASNReport` with a populated `Peering` (Type "Content", Policy "Open", IXCount 2, one IXP, one facility), call `printASNReport`, assert the output `Contains` "Content", "Open", and the IX/facility counts (separate `require.Contains` per token, not one padded string — the lookups-2 lesson). Add a second assertion that with `Peering == nil` the output has no "Peering" row.

- [ ] **Step 2: Run test, verify it fails** — Expected: FAIL (no peering rows emitted).

- [ ] **Step 3: Implement** — in `printASNReport`, when `r.Peering != nil`, emit rows: `Peering` = `"<Type> · policy <Policy>"`, `IXPs` = `"<n> (<ixCount> total)"` style summary with the top IX name(s), `Facilities` = count. Keep each `reportRow`. Add `r.PeeringSrc` to the `printSourceErrs` call.

- [ ] **Step 4: Run tests, verify they pass** (`go test ./internal/cli/ -v`) — Expected: PASS.

- [ ] **Step 5: Commit** (`internal/cli/cli.go`, `internal/cli/cli_test.go`).

---

### Task 4: ASN overview Peering card (UI)

**Files:**
- Modify: `frontend/src/lib/EntityPage.svelte` (ASN overview `.cards`)

**Interfaces:**
- Consumes: `data.peering` (`{name,type,scope,ratio,policy,irrAsSet,website,v4Prefixes,v6Prefixes,ixCount,facCount,ixps,facilities}`), `data.peeringSrc`.

- [ ] **Step 1: Add the card** — in the `type === "asn"` overview block, after the hero card and guarded by `{#if data.peering}`, add a `card span` titled **Peering** with `<span class="src">{chip(data.peeringSrc)}</span>`:
  - a `dl.mono`: Type (`{data.peering.type}`), Scope, Traffic ratio, Peering policy (as a `pill`), IRR AS-SET, with a one-line `.cap`/`.dim` caption under the policy explaining Open/Selective/Restrictive in plain words;
  - **IXPs** — `{data.peering.ixCount}` present; list the first ~12 `ixps` as chips showing name + speed (`{(ix.speed/1000)} Gbps`), then `{#if data.peering.ixps.length > 12}+{data.peering.ixps.length - 12} more{/if}`;
  - **Facilities** — `{data.peering.facCount}`; first ~12 `facilities` as `name — city, country`, same "+N more".
  - `{#if data.peeringSrc?.err}<p class="err">Peering data unavailable: {data.peeringSrc.err}</p>{/if}`.
  - Reuse existing card/chip/pill/cap CSS; add a `.chips`/`.cap` rule only if absent.

- [ ] **Step 2: Verify the build** — `cd frontend && npm run build` (or `npx svelte-check`) — Expected: no type/compile errors.

- [ ] **Step 3: Commit** (`frontend/src/lib/EntityPage.svelte`).

---

## Self-review notes
- Spec coverage: net summary (T1/T4), IX presence + facilities with dedupe/cap (T1 engine, T4 UI cap), Peering card with captions (T4), CLI summary (T3), partial-failure degradation (T1/T2), keyless/seam/UA (Global Constraints, T1). IP-page "peers at N" line is spec-"keeps"/minor — deferred to a follow-up, not in these tasks (noted so it isn't mistaken for a gap).
- Type consistency: `Peering`/`PeerIX`/`PeerFac` defined T1, consumed by name T2–T4; `peeringDBBase` defined T1, swapped T1/T2 tests.
- Proportion: four tasks, engine-heavy; UI task is build-verified (no Svelte unit-test harness in this repo, matching existing tool UIs).
