# Meridian Lookups Slice 2 — Prefix & Domain Pages Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans (inline) to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Add **Prefix** and **Domain** entity pages to Lookup — on the same pivoting entity-page shell — answering "who announces this prefix / is it RPKI-valid" and "where does this domain resolve / who's it registered to," reusing the existing RDAP, RIPEstat, DNS, and AS-annotation engines.

**Architecture:** Extend `internal/lookup` with Prefix and Domain sources + resolvers. Prefix = RDAP (registry) + RIPEstat (`prefix-overview`, `related-prefixes`, `rpki-validation`). Domain = the `dnsx` engine (records) + RDAP domain (`rdap.org/domain/<name>`) + `asnmap` (who hosts the A records). Also back-fill the IP page's RPKI "soon" using the same RIPEstat RPKI call. The Wails controller + CLI route the new kinds; the Svelte `EntityPage` grows `prefix` and `domain` variants.

**Tech Stack:** Go 1.23; stdlib + existing deps (`dnsx`/miekg, `asnmap`/Cymru, RDAP, RIPEstat). No new dependency.

**Spec:** scratchpad design `2026-11-09-lookups-slice2-design.md` (owner-approved 2026-11-09) + docs/superpowers/specs/2026-11-08-meridian-design.md §4.1. Decisions: RPKI via RIPEstat `rpki-validation`; domain registration via RDAP only (ccTLD-without-RDAP shows "no RDAP data," not an error); Prefix **and** Domain in this slice.

## Global Constraints
- Go 1.23; stdlib-only additions; generic `meridian/0.1` User-Agent; per-source Source{Label,FetchedAt,Err}; partial-failure never fails the whole lookup (slice-1 pattern).
- Network only on user action. First-class IPv4 + IPv6 (v6 prefixes, AAAA).
- UI: plain-language verdict + source/freshness chips (project standard); themes/fonts/collapsible intact.
- Reuse: `EntityPage.svelte`, `internal/lookup` RDAP/RIPEstat client, `internal/dnsx`, `internal/asnmap`. No duplication of those.

## Review Focus
- **Unannounced prefix / RPKI unknown** → "not globally seen" / "unknown", no error. *T2.*
- **Domain with no RDAP (ccTLD)** → DNS still resolves; registration section says "no RDAP data." *T2.*
- **RPKI invalid** (origin ≠ ROA) → surfaced clearly as "invalid," not hidden. *T1/T2.*
- **Domain NXDOMAIN** → clean "doesn't resolve," DNS section shows the rcode. *T2.*
- **IPv6 prefix + AAAA domain** end to end. *T1/T2.*

---

## Task 1: Prefix + Domain source functions
**Files:** add `internal/lookup/prefix.go`, `internal/lookup/domain.go`; tests alongside.
**Interfaces (Produces):**
- `rdapDomain(ctx, name) (DomainReg, error)` — parse registrar, created/expires, nameservers, DNSSEC from `rdapBase+"domain/"+name` (tolerant, like rdapIP).
- `ripePrefixOverview(ctx, prefix) (originASNs []string, holder string, announced bool, err error)`.
- `ripeRelatedPrefixes(ctx, prefix) (lessSpecific, moreSpecific []string, err error)`.
- `ripeRPKI(ctx, prefix, asn string) (state string, roa string, err error)` — RIPEstat `rpki-validation` (state ∈ valid/invalid/unknown).
- Reuse `rdapIP` for a prefix's covering block registry.
- [ ] Failing tests: `TestRdapDomainParses` (canned RDAP domain → registrar/expiry/NS/DNSSEC), `TestRipePrefixOverview`, `TestRipeRPKIStates` (valid + invalid fixtures), `TestRipeRelatedPrefixes`. RED → implement → GREEN → commit.

## Task 2: LookupPrefix + LookupDomain resolvers (+ IP-page RPKI back-fill)
**Files:** `internal/lookup/resolve_prefix.go`, `resolve_domain.go`; a `dnsx` query seam on `Client`; tests.
**Interfaces (Produces):**
- `type PrefixReport struct { Input, OriginASN, OriginName, Registry, RegisteredOrg, AbuseEmail, RPKIState, RPKIRoa string; LessSpecific, MoreSpecific []string; RoutingSrc, RegistrySrc, RPKISrc Source }`.
- `type DomainReport struct { Input string; Records []dnsx.Record; ResolvesTo []string; HostedBy []string /*ASN owners*/; Registrar, Created, Expires, DNSSEC string; Nameservers []string; DNSSrc, RegistrySrc Source }`.
- `func (*Client) LookupPrefix(ctx, prefix string) (PrefixReport, error)`, `func (*Client) LookupDomain(ctx, name string) (DomainReport, error)`.
- Back-fill: `IPReport` gains `RPKIState` filled from `ripeRPKI(coveringPrefix, originASN)`.
- [ ] Failing tests (httptest combined server + injected dnsx query): `TestLookupPrefixCombines` (origin/holder/rpki/registry), `TestLookupPrefixUnannounced`, `TestLookupDomainResolvesAndRegisters`, `TestLookupDomainNoRDAP` (DNS ok, registration empty, no error), `TestLookupDomainNXDOMAIN`, `TestLookupIPHasRPKI`. RED → implement → GREEN → commit.

## Task 3: CLI routing + app methods
**Files:** `internal/cli/cli.go` (route `KindPrefixV4/V6`→LookupPrefix, `KindDomain/Hostname`→LookupDomain in `runLookup`), `internal/app/lookup_methods.go` (add `LookupPrefix`, `LookupDomain`); tests.
- [ ] Failing tests: `TestRunLookupPrefixJSON`, `TestRunLookupDomainJSON` (fake-backed), `TestAppLookupDomain`. RED → implement → GREEN → `go test ./...` → commit.

## Task 4: Frontend — Prefix & Domain entity pages
**Files:** `frontend/src/lib/EntityPage.svelte` (add `type==="prefix"` and `"domain"` layouts), `frontend/src/lib/tools/Lookup.svelte` (route prefix/domain), models regenerate.
- Prefix page: hero "who announces this" (origin AS, RPKI badge valid/invalid/unknown), Registry, more/less-specifics (pivots). Fill the IP page's RPKI chip from `rpkiState`.
- Domain page: hero "where it resolves / who hosts it," DNS records table (each IP a pivot), Registration (or "no RDAP data").
- [ ] Build (`wails build`), launch, verify a real prefix (`104.16.0.0/13`) and domain (`cloudflare.com`) render; `go test ./...`; commit.

## Self-review notes
- Coverage: prefix/domain sources (T1), resolvers + RPKI back-fill (T2), adapters (T3), UI (T4).
- Reuse verified: EntityPage, RDAP parser, RIPEstat client, dnsx, asnmap — no re-implementation.
- Review Focus items each have a named test in T1/T2.
- Proportion: new code is a handful of source funcs + two reports + two Svelte layouts; the engines are done.
