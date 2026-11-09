# Meridian Lookups (Slice 1) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Any-value lookup of an IP address or ASN that answers "who owns this" from live RDAP + RIPEstat data, shown on a pivoting, source-labelled entity page (and via the CLI).

**Architecture:** A pure `internal/lookup` engine with typed reports whose every section carries its own source + fetch time + error, fed by two data-source files (RDAP via rdap.org, routing/holder via RIPEstat) behind an injectable HTTP layer with a timeout and an in-memory TTL cache. The Wails controller and CLI are thin adapters, exactly as the foundation slice established. The Svelte Lookup tool renders the entity page and makes the Workbench sidebar collapsible.

**Tech Stack:** Go 1.23 (stdlib `net/http`, `encoding/json`, `net/netip`); Wails v2.10.1 + Svelte/TS; no new Go dependencies.

**Spec:** docs/superpowers/specs/2026-11-08-meridian-design.md (§4.1 Lookups, §3 decisions 48–49). Slice-1 scope approved by owner 2026-11-09 against the clickable mockup: entity types **IP + ASN**; tabs **Overview + Registry** (Routing folded into Overview; DNS/Reachability/RPKI/PeeringDB shown as "soon"); hero = ownership; data = RDAP + RIPEstat; pivot header + per-card source/freshness + Copy as JSON.

## Global Constraints

- Go **1.23**; no new third-party Go modules — stdlib only for the engine.
- Network egress happens **only** on a user-initiated lookup. Outbound HTTP uses a generic `User-Agent: meridian/0.1 (+https://github.com/hriday/meridian)` — **never** the user's email or any identity.
- Every report section carries **Source** (human label) + **FetchedAt** (UTC) + **Err** (string, empty if ok). A failed source yields a partial report, never a failed lookup.
- HTTP calls use a **5 s** timeout and go through the injectable `Doer` so tests never touch the network.
- First-class **IPv4 and IPv6** throughout.
- UI stays non-blocking (async calls, busy state); interactions < 100 ms; themes (Auto/Light/Dark/Terminal), Geist fonts, and the globe logo remain intact.
- **Collapsible menus:** the Workbench sidebar collapses to an icon rail; tool groups collapse individually. State persisted in localStorage (per-viewer, wrapped in try/catch).
- Counts that can be large (announced-prefix totals) cross to JS as plain integers (they fit in float64) — no big-int needed this slice.

## Review Focus

- **Private / bogon / unannounced IP** (10.0.0.0/8, 192.0.2.0/24, an unrouted public IP): no origin AS and often no RDAP → the page must say "private/unannounced, no routing" cleanly, not error. *Test in Task 4.*
- **RDAP shape variance across RIRs** (ARIN vs RIPE vs APNIC/LACNIC/AFRINIC differ in entity nesting and vCard layout): org name + abuse email extraction must tolerate missing/rearranged fields and never panic. *Test in Task 3.*
- **Offline / timeout / non-200**: each source returns its error into the section; `Lookup*` still returns a report; the UI shows the error chip and never hangs. *Test in Tasks 1 & 4.*
- **IPv6 end to end** (2606:4700::/32, an AAAA target): RDAP + RIPEstat + netip parsing all handle v6. *Test in Tasks 2–4.*
- **Malformed / truncated JSON or huge arrays** (announced-prefixes with tens of thousands of entries): parse defensively, count without retaining everything, surface a parse error rather than OOM/panic. *Test in Tasks 1 & 2.*

---

## Task 1: Report types + HTTP fetch layer

**Files:**
- Create: `internal/lookup/types.go`
- Create: `internal/lookup/client.go`
- Test: `internal/lookup/client_test.go`

**Interfaces:**
- Produces: `type Source struct { Label string; FetchedAt time.Time; Err string }`; `type Doer interface { Do(*http.Request) (*http.Response, error) }`; `type Client struct { ... }`; `func NewClient(d Doer, ttl time.Duration) *Client`; `func (*Client) GetJSON(ctx context.Context, url string, v any) error` (applies User-Agent, timeout via ctx, TTL cache keyed by url, returns error on non-200 / decode failure).
- Consumes: nothing (first task).

- [ ] **Step 1: Write the failing test** `TestGetJSONCachesAndSetsUserAgent`

```go
// A Doer records requests and returns a canned 200 JSON body on first call,
// then fails if called twice for the same URL (proving the cache).
func TestGetJSONCachesAndSetsUserAgent(t *testing.T) {
    var got *http.Request
    d := doerFunc(func(r *http.Request) (*http.Response, error) {
        if got != nil { t.Fatal("second network call for cached url") }
        got = r
        return jsonResp(200, `{"n":7}`), nil
    })
    c := lookup.NewClient(d, time.Minute)
    var out struct{ N int }
    require.NoError(t, c.GetJSON(context.Background(), "https://x/y", &out))
    require.Equal(t, 7, out.N)
    require.Contains(t, got.Header.Get("User-Agent"), "meridian/")
    require.NoError(t, c.GetJSON(context.Background(), "https://x/y", &out)) // served from cache
}
```

- [ ] **Step 2: Add `TestGetJSONErrorsOnNon200` and `TestGetJSONErrorsOnBadBody`** (non-200 → error containing the status; invalid JSON → error), each asserting the error is returned (not panicked).
- [ ] **Step 3: Run the tests, verify they fail** (`go test ./internal/lookup/ -run TestGetJSON -v`) — Expected: FAIL, `lookup.NewClient` undefined.
- [ ] **Step 4: Implement** `Source`, `Doer`, `Client`, `NewClient`, `GetJSON` in `types.go`/`client.go`. Minimal in-memory `map[string]cached` with mutex + TTL. User-Agent constant.
- [ ] **Step 5: Run the tests, verify they pass.** Expected: PASS.
- [ ] **Step 6: Commit** (`feat(lookup): report source metadata + cached JSON HTTP client`).

## Task 2: RIPEstat source (routing + holder)

**Files:**
- Create: `internal/lookup/ripestat.go`
- Test: `internal/lookup/ripestat_test.go`

**Interfaces:**
- Consumes: `Client.GetJSON` (Task 1).
- Produces: `func (*Client) ripeNetworkInfo(ctx, ip string) (prefix string, originASNs []string, err error)` (RIPEstat `network-info`); `func (*Client) ripeASOverview(ctx, asn string) (holder string, err error)` (`as-overview`); `func (*Client) ripeRoutingStatus(ctx, asn string) (v4Count, v6Count, neighbours int, err error)` (`routing-status`). Base URL is a package var so tests point it at httptest.

- [ ] **Step 1: Write the failing test** `TestRipeNetworkInfoParsesPrefixAndOrigin` using `httptest.NewServer` returning a recorded `network-info` payload for `104.16.123.45` → asserts prefix `104.16.0.0/13`, origin `["13335"]`.
- [ ] **Step 2: Add `TestRipeRoutingStatusCounts`** (recorded `routing-status` for AS13335 → v4Count>0, v6Count>0, neighbours>0) and `TestRipeNetworkInfoNoOrigin` (unannounced IP payload → empty origin slice, no error).
- [ ] **Step 3: Run the tests, verify they fail.** Expected: FAIL, functions undefined.
- [ ] **Step 4: Implement** the three functions: build the RIPEstat URL from the base var, decode into minimal structs matching the `data` envelope, extract fields. Tolerate missing `data` keys (return zero values + nil error for "no data").
- [ ] **Step 5: Run the tests, verify they pass.**
- [ ] **Step 6: Commit** (`feat(lookup): RIPEstat routing and holder source`).

## Task 3: RDAP source (ownership + abuse)

**Files:**
- Create: `internal/lookup/rdap.go`
- Test: `internal/lookup/rdap_test.go`

**Interfaces:**
- Consumes: `Client.GetJSON` (Task 1).
- Produces: `type RDAPEntity struct { Handle, Org, AbuseEmail, Registry string }`; `func (*Client) rdapIP(ctx, ip string) (RDAPEntity, error)`; `func (*Client) rdapASN(ctx, asn string) (RDAPEntity, error)`. Base URL (`https://rdap.org/`) is a package var; `GetJSON`/client follows redirects to the RIR.

- [ ] **Step 1: Write the failing test** `TestRdapIPExtractsOrgAndAbuse` with a recorded ARIN-style RDAP body (nested `entities` with roles `registrant` and `abuse`, vCard `fn` + `email`) → asserts `Org` and `AbuseEmail` and `Registry`.
- [ ] **Step 2: Add `TestRdapToleratesMissingVCard`** (RIPE-style body with a differently nested entity and no abuse vCard → returns what it can, empty `AbuseEmail`, **no panic/error**) and `TestRdapASN` (autnum body → handle + org).
- [ ] **Step 3: Run the tests, verify they fail.** Expected: FAIL, functions undefined.
- [ ] **Step 4: Implement** tolerant RDAP parsing: decode into `map`/loosely-typed structs; walk `entities` recursively for roles; pull `fn`/`email` from the jCard array defensively (guard every index/type assertion). Derive `Registry` from the response `port43`/`links` or the `handle` suffix.
- [ ] **Step 5: Run the tests, verify they pass.**
- [ ] **Step 6: Commit** (`feat(lookup): tolerant RDAP ownership source`).

## Task 4: Resolver — LookupIP / LookupASN

**Files:**
- Create: `internal/lookup/resolve.go`
- Test: `internal/lookup/resolve_test.go`

**Interfaces:**
- Consumes: Tasks 2 & 3 functions; `netinput.Classify` (foundation) is used by the adapters, not here.
- Produces: `type IPReport struct { Input, Version, CoveringPrefix, OriginASN, OriginName, RegisteredOrg, AbuseEmail, Registry string; RPKINote string; Routing, Registry Source; ... }`; `type ASNReport struct { ASN, Holder, Registry string; V4Prefixes, V6Prefixes, Neighbours int; RoutingSrc, RegistrySrc Source }`; `func (*Client) LookupIP(ctx, ip string) (IPReport, error)`; `func (*Client) LookupASN(ctx, asn string) (ASNReport, error)`. Each combines sources, stamps `Source.FetchedAt`, and records per-source errors instead of aborting.

- [ ] **Step 1: Write the failing test** `TestLookupIPCombinesSources` with an `httptest` mux serving both RDAP and RIPEstat canned bodies → asserts `CoveringPrefix`, `OriginASN`, `RegisteredOrg`, `AbuseEmail` all populated and `Routing.Err == "" && Registry.Err == ""`.
- [ ] **Step 2: Add** `TestLookupIPPrivateAddress` (`10.1.2.3` → no network call needed for routing; `Version=="IPv4"`, empty `OriginASN`, a human `RPKINote`/routing note like "private (RFC1918), not globally routed", no error) and `TestLookupPartialOnSourceError` (RIPEstat server returns 500 → report still returns, `Routing.Err` non-empty, Registry section still populated) and `TestLookupIPv6` (v6 target end to end).
- [ ] **Step 3: Run the tests, verify they fail.** Expected: FAIL, types/functions undefined.
- [ ] **Step 4: Implement** `LookupIP`/`LookupASN`: classify private/bogon via `netip` (`IsPrivate`/`IsLoopback`/`IsLinkLocalUnicast`/`IsMulticast`) to short-circuit routing; otherwise call RIPEstat then RDAP; fill reports; set sources + errors; never return a non-nil error unless the input itself is invalid.
- [ ] **Step 5: Run the tests, verify they pass.** Then run the whole package: `go test ./internal/lookup/`.
- [ ] **Step 6: Commit** (`feat(lookup): IP and ASN resolvers with partial-failure reports`).

## Task 5: CLI `lookup` + bound App methods

**Files:**
- Modify: `internal/cli/cli.go` (add `lookup` command + usage)
- Create: `internal/app/lookup_methods.go` (bound `LookupIP`, `LookupASN`)
- Test: `internal/cli/cli_lookup_test.go`, `internal/app/lookup_methods_test.go`

**Interfaces:**
- Consumes: `lookup.NewClient`, `LookupIP`, `LookupASN`; `netinput.Classify` to route a bare value to the right resolver.
- Produces: CLI `meridian-cli lookup <ip|asn> [--json]`; App methods `func (*App) LookupIP(string) (lookup.IPReport, error)` and `func (*App) LookupASN(string) (lookup.ASNReport, error)` (constructing a real `http.Client` Doer with the 5 s timeout).

- [ ] **Step 1: Write the failing test** `TestRunLookupJSON` (in cli): inject a fake resolver/base-URL pointing at an httptest server (reuse Task 4 fixtures) → `cli.Run(["lookup","104.16.123.45","--json"], …)` exits 0 and the JSON contains `"OriginASN":"13335"`.
- [ ] **Step 2: Add** `TestRunLookupRoutesASN` (`lookup AS13335` → ASN report path) and an app-method test `TestAppLookupIP` asserting the bound method returns a populated report (httptest-backed).
- [ ] **Step 3: Run the tests, verify they fail.** Expected: FAIL.
- [ ] **Step 4: Implement** the `lookup` command (classify → `LookupIP`/`LookupASN`; text and `--json` output via the existing `writeJSON`) and the bound methods. Keep the `Doer` injectable for tests (package var or constructor arg).
- [ ] **Step 5: Run the tests, verify they pass.** Then `go test ./...`.
- [ ] **Step 6: Commit** (`feat(lookup): CLI command and bound app methods`).

## Task 6: Frontend — Lookup tool, entity page, collapsible sidebar

**Files:**
- Create: `frontend/src/lib/tools/Lookup.svelte`
- Create: `frontend/src/lib/EntityPage.svelte` (Overview + Registry cards, pivot header, source/freshness chips, Copy JSON)
- Modify: `frontend/src/lib/Workbench.svelte` (activate Lookup; collapsible sidebar + collapsible groups)
- Modify: `frontend/src/lib/kinds.ts` if needed for entity routing
- (Wails regenerates `frontend/wailsjs/go/**` bindings on build.)

**Interfaces:**
- Consumes: bound `LookupIP`, `LookupASN`, `Classify` (generated bindings); `lookup.IPReport`/`ASNReport` models.

- [ ] **Step 1:** Build `EntityPage.svelte` matching the approved mockup: pivot breadcrumb (IP → prefix → AS → RIR, pivots call back into Lookup), hero "Who owns this" card, Overview (Routing + This-address) and Registry tab, each card with a source+freshness chip from the report's `Source`, and a working **Copy as JSON** button. "soon" tabs for Routing/DNS/Reachability.
- [ ] **Step 2:** Build `Lookup.svelte`: any-value input → `Classify` → call `LookupIP`/`LookupASN` → render `EntityPage`; busy + error states; opens the default value on mount (like SubnetCalc).
- [ ] **Step 3:** Make the Workbench sidebar collapsible — a toggle collapses it to an icon rail; each group header toggles its items; persist both in localStorage (try/catch). Activate the Lookup tool (ready: true) and set it as the default tab.
- [ ] **Step 4: Build** (`~/go/bin/wails build`) and verify it compiles; launch and confirm a real lookup renders (manual visual check — UI has no unit harness, per the foundation pattern).
- [ ] **Step 5: Run** `go test ./...` to confirm the engine suite is green.
- [ ] **Step 6: Commit** (`feat(lookup): entity-page UI, Lookup tool, collapsible sidebar`).

---

## Self-review notes
- Spec coverage: ownership (RDAP+RIPEstat) = Tasks 2–4; entity page + pivot + source labels = Task 6; CLI parity = Task 5. RPKI/PeeringDB/DNS/Reachability are explicitly out of slice 1 (shown as "soon").
- Type consistency: `Source` (Task 1) is reused by all reports; resolver field names (Task 4) are consumed verbatim by CLI (Task 5) and UI (Task 6).
- Review Focus: each of the five input classes has a named test in Tasks 1–4.
- Proportion: plan specifies signatures, test names, and the data-source URLs/decisions; implementations are left to the engineer.
