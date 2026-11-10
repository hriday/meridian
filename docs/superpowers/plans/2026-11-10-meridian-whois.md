# Whois-grade Domain Registration — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deepen the existing domain registration (RDAP) to Whois-grade — status codes, last-updated date, and registrar abuse contact — in the lookup core, CLI, and entity page.

**Architecture:** Extend `rdapDomain`/`DomainReg` (`internal/lookup/domain.go`) to parse the extra RDAP fields, carry them on `DomainReport` (`internal/lookup/resolve_domain.go`), print them in the CLI (`printDomainReport`), and show them on the domain entity card (`EntityPage.svelte`). No new engine, tool, or command.

**Tech Stack:** Go 1.23, existing `lookup.Client` RDAP path, Wails v2 + Svelte. Tests reuse the `rdapServer(t, body)` fixture helper.

**Spec:** `docs/superpowers/specs/2026-11-10-meridian-whois-design.md`

## Global Constraints

- RDAP-only; every new field optional — a record lacking them yields empty values, never an error. Existing domain tests must keep passing.
- No new Workbench tool / CLI command — enrich the existing domain lookup.
- Abuse contact reuses the existing `findRole(entities, "abuse", "email")` walk.
- Commit attribution: `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`. PR body ends with the Generated-with line.

## Review Focus

- **Missing fields don't break the common case.** A TLD whose RDAP has no `status`, no `last changed`, or no abuse entity must still return the existing registrar/dates/NS/DNSSEC with the new fields empty. (Task 1 test with the original fixture body unchanged.)
- **Abuse nested under registrar.** The abuse email is usually on an entity nested inside the registrar entity; the walk must find it. (Task 1 fixture nests abuse under registrar.)
- **Multiple status codes preserved in order.** `status:["clientTransferProhibited","clientDeleteProhibited"]` yields both, in order. (Task 1 test.)
- **`last changed` vs other events.** Only the `last changed` event becomes `Updated`; `registration`/`expiration` still map to Created/Expires. (Task 1 test asserts all three distinctly.)

---

### Task 1: Parse status / updated / abuse in the engine

**Files:**
- Modify: `internal/lookup/domain.go` (DomainReg + rdapDomain), `internal/lookup/resolve_domain.go` (DomainReport + copy)
- Test: `internal/lookup/domain_test.go`

**Interfaces:**
- `DomainReg` gains `Statuses []string`, `Updated string`, `AbuseEmail string`.
- `DomainReport` gains the same three fields, copied from reg in `LookupDomain`.
- `rdapDomain` additionally decodes `status []string`, the `events` entry with action `last changed` → `Updated`, and `findRole(raw.Entities, "abuse", "email")` → `AbuseEmail`.

- [ ] **Step 1: Write failing test** — extend `domain_test.go`:
  - Add to `rdapDomainBody` (or a new `rdapDomainFullBody`): `"status":["clientTransferProhibited","clientDeleteProhibited"]`, an event `{"eventAction":"last changed","eventDate":"2024-05-01T00:00:00Z"}`, and a registrar entity with a nested abuse entity `{"roles":["abuse"],"vcardArray":["vcard",[["email",{},"text","abuse@registrar.example"]]]}`.
  - `TestRdapDomainParsesWhoisFields`: assert `reg.Statuses == ["clientTransferProhibited","clientDeleteProhibited"]`, `reg.Updated` contains `2024-05-01`, `reg.AbuseEmail == "abuse@registrar.example"`, and Created/Expires still parse.
  - `TestRdapDomainMissingWhoisFieldsEmpty`: using the ORIGINAL `rdapDomainBody` (no status/last-changed/abuse), assert `reg.Statuses` empty, `reg.Updated == ""`, `reg.AbuseEmail == ""`, no error.
- [ ] **Step 2: Run, verify fail** (`go test ./internal/lookup/ -run RdapDomain -v`).
- [ ] **Step 3: Implement** the three new fields in `DomainReg`, the parse in `rdapDomain` (add `Status []string` and keep `Events`; walk abuse via `findRole`), the three fields on `DomainReport`, and the copy in `LookupDomain`.
- [ ] **Step 4: Run, verify pass**, then full `go test ./internal/lookup/`.
- [ ] **Step 5: Commit** (`feat(lookup): Whois-grade domain fields — status, updated, abuse`).

### Task 2: Surface in CLI and the domain entity page

**Files:**
- Modify: `internal/cli/cli.go` (`printDomainReport`)
- Test: `internal/cli/printdomain_test.go` (internal `package cli`; create if absent — otherwise add to the existing domain printer test)
- Modify: `frontend/src/lib/EntityPage.svelte` (domain registration card)
- Regenerate: bindings (`models.ts` DomainReport delta only)

- [ ] **Step 1: Write failing CLI test** — `TestPrintDomainReportWhoisFields`: a `DomainReport` with Statuses/Updated/AbuseEmail → `printDomainReport` output contains a status code, the updated date, and the abuse email; a report without them omits those rows (no empty labels).
- [ ] **Step 2: Run, verify fail.**
- [ ] **Step 3: Implement** the new rows in `printDomainReport` (print each only when non-empty; join statuses with ", ").
- [ ] **Step 4: Run, verify pass.**
- [ ] **Step 5: Regenerate bindings** (`$(go env GOPATH)/bin/wails generate module`; `git checkout -- frontend/wailsjs/runtime`); verify the `DomainReport` model in `models.ts` gained `statuses/updated/abuseEmail`.
- [ ] **Step 6: Update `EntityPage.svelte`** — in the domain registration card, add a status chip row (with a `statusGloss()` helper giving a plain-language tooltip for common EPP codes), the Updated date beside Created/Expires, and the abuse contact line. Only render each when present. Match the card's existing style.
- [ ] **Step 7: Build** `cd frontend && npm run build`; verify green.
- [ ] **Step 8: Manual check** via the running app (localhost:34115): look up a domain (e.g. google.com) and confirm the status chips, updated date, and abuse contact render and read clearly.
- [ ] **Step 9: Commit** (`feat(cli,ui): show Whois-grade domain registration fields`).

---

## Self-Review notes

- Spec coverage: status/updated/abuse → Task 1 (engine) + Task 2 (surface). No new tool (enrichment only).
- Review Focus → each maps to a Task 1 test (missing-fields-empty, nested abuse, multiple statuses, last-changed-distinct).
- Proportion: signatures + test names; existing fixture extended rather than rewritten.
