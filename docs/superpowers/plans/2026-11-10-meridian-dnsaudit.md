# DNS / Email Posture Audit — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Audit a domain's DNS/email posture — nameservers, MX, SPF, DMARC, DNSSEC — each graded in plain language with an overall summary.

**Architecture:** New engine `internal/dnsaudit` with pure record classifiers (`spfVerdict`, `dmarcVerdict`) and a networked `Audit` that reuses `internal/dnsx` to resolve NS/MX/TXT and read the AD flag. Thin adapters: a Wails method, a `dnsaudit` CLI command, and a `DnsAudit.svelte` Workbench tool. Mirrors the dnsbl slice's structure and the DNS tool's resolver-default UX.

**Tech Stack:** Go 1.23, `github.com/miekg/dns` (via `internal/dnsx`), Wails v2 + Svelte/TypeScript. Tests use a local `miekg/dns` server and `stretchr/testify`.

**Spec:** `docs/superpowers/specs/2026-11-10-meridian-dnsaudit-design.md`

## Global Constraints

- Read-only DNS; query carries only the audited domain, no user identity.
- Default resolver = system resolver (`sysconf.Resolvers()` first value), as the DNS/blocklist tools do; UI shows the resolver field.
- DNSSEC via the AD flag is a resolver-dependent signal, not proof — the note must say so.
- A DNS-level refusal (SERVFAIL/REFUSED) is `error`, never a false `ok`/`missing` (same lesson as the dnsbl slice: `dnsx.Query` returns no error for a response that arrives, so check `res.Rcode`).
- Engine UI-free; network behind the `dnsx.Query` seam.
- Commit attribution: `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`. PR body ends with the Generated-with line.

## Review Focus

- **SERVFAIL read as missing/ok.** A check whose query SERVFAILs must read `error`, not `missing` (which would wrongly imply "the record is absent"). (Task 2 test with a SERVFAIL server.)
- **`+all` / bare `all` SPF graded lenient.** `v=spf1 +all` and `v=spf1 ... all` (bare) pass any sender and must be `warn`, not `ok`. (Task 1 `spfVerdict` cases.)
- **Multiple SPF records.** Two `v=spf1` TXT records are invalid (RFC 7208) and must `warn`, not silently take the first. (Task 1 case.)
- **DMARC `p=none` graded strong.** `p=none` is monitoring only and must be `warn`, not `ok`. (Task 1 `dmarcVerdict` case.)
- **Null MX.** `0 .` means the domain explicitly accepts no mail — a distinct `warn` note, not read as a normal mail host. (Task 2 MX handling + a `spfVerdict`-style note check in Task 2.)
- **Quoted TXT.** dnsx renders TXT values quoted (`"v=spf1 -all"`); parsing must strip quotes or every SPF/DMARC check silently reads `missing`. (Task 2 normalization; Task 1 parsers take clean strings.)

---

### Task 1: Pure record classifiers + types

**Files:**
- Create: `internal/dnsaudit/dnsaudit.go`
- Test: `internal/dnsaudit/dnsaudit_test.go`

**Interfaces:**
- Produces:
  - `type Status string` with `StatusOK="ok"`, `StatusWarn="warn"`, `StatusMissing="missing"`, `StatusError="error"`.
  - `type Check struct { Name string; Status Status; Note string; Records []string }`
  - `type Report struct { Domain string; Resolver string; Checks []Check; OKs int; Warns int; Missing int; Errors int; Summary string }`
  - `func spfVerdict(apexTXT []string) (Status, string, []string)` — filters the TXT values (quotes already stripped) to those starting `v=spf1` (case-insensitive); returns `StatusMissing` if none; `StatusWarn` if >1 (invalid); else classifies the terminal `all` mechanism: `-all`→ok(strict), `~all`→ok(soft, noted), `?all`→warn(neutral), `+all` or bare `all` or no `all`→warn(permissive/none). Third return is the matched SPF record(s).
  - `func dmarcVerdict(txt []string) (Status, string, []string)` — filters to `v=DMARC1`; `StatusMissing` if none; parses the `p=` tag: `reject`→ok, `quarantine`→ok(noted), `none`→warn, missing/other→warn(malformed). Returns the matched record(s).

- [ ] **Step 1: Write failing tests** `internal/dnsaudit/dnsaudit_test.go` (`package dnsaudit`):
  - `TestSPFVerdict`: table — `["v=spf1 -all"]`→ok; `["v=spf1 include:_spf.example.com ~all"]`→ok (note mentions soft); `["v=spf1 ?all"]`→warn; `["v=spf1 +all"]`→warn; `["v=spf1 include:x"]` (no all)→warn; `["v=spf1 -all","v=spf1 ~all"]`→warn (multiple); `["not spf"]`→missing; `nil`→missing. Assert the returned record slice holds the spf record(s) for the ok/warn cases.
  - `TestDMARCVerdict`: `["v=DMARC1; p=reject; rua=..."]`→ok; `["v=DMARC1; p=quarantine"]`→ok; `["v=DMARC1; p=none"]`→warn; `["v=DMARC1; rua=mailto:x"]` (no p)→warn; `["something"]`→missing; `nil`→missing.
- [ ] **Step 2: Run tests, verify fail** (`go test ./internal/dnsaudit/ -run 'SPFVerdict|DMARCVerdict' -v`).
- [ ] **Step 3: Implement** `dnsaudit.go`: the types and the two classifiers. For SPF, locate the `all` mechanism with a small scan over space-split tokens matching `^[-~?+]?all$` (case-insensitive); default qualifier `+`. For DMARC, split the record on `;`, trim spaces, find the `p=` tag.
- [ ] **Step 4: Run tests, verify pass.**
- [ ] **Step 5: Commit** (`feat(dnsaudit): SPF and DMARC classifiers + types`).

### Task 2: Networked `Audit`

**Files:**
- Modify: `internal/dnsaudit/dnsaudit.go`
- Test: `internal/dnsaudit/audit_test.go`

**Interfaces:**
- Consumes: Task 1 types/classifiers; `dnsx.Query(ctx, name, rtype, server)` via seam `var queryFn = dnsx.Query`.
- Produces:
  - `func Audit(ctx context.Context, domain, resolver string, timeout time.Duration) (Report, error)` — errors if `domain` empty or `resolver` empty. Runs: NS (apex, type NS), MX (apex), SPF (apex TXT → strip quotes → `spfVerdict`), DMARC (`_dmarc.<domain>` TXT → `dmarcVerdict`), DNSSEC (AD flag from the apex NS response). Each query: a transport error or a non-NOERROR/NXDOMAIN rcode → that check `StatusError`; NXDOMAIN/empty where a record is expected → `StatusMissing` (NS/MX/SPF/DMARC). Builds the five `Check`s, the ok/warn/missing/error counts, and a one-line `Summary` naming the first thing to fix (first non-ok check in NS→MX→SPF→DMARC→DNSSEC order; "looks healthy" when all ok).

- [ ] **Step 1: Write failing tests** `internal/dnsaudit/audit_test.go` using a local `miekg/dns` server (dnsbl/dnsx pattern) whose handler answers by qname+qtype and sets `AuthenticatedData` for a "signed" domain:
  - `TestAuditHealthy`: server returns NS (2 records), MX (`10 mail…`), apex TXT `v=spf1 -all`, `_dmarc` TXT `v=DMARC1; p=reject`, AD=true. Assert each Check status ok, `Warns==0`, `Missing==0`, `Errors==0`, summary says healthy.
  - `TestAuditMissingDMARC`: as above but `_dmarc` NXDOMAIN → DMARC check `StatusMissing`, `Missing>=1`, summary names DMARC.
  - `TestAuditServfailIsError`: NS query SERVFAILs → NS check `StatusError` (not missing); `Errors>=1`.
  - `TestAuditQuotedSPFParsed`: ensure the apex TXT `"v=spf1 -all"` (as dnsx renders it, quoted) is parsed → SPF `StatusOK`, not missing (guards the quote-stripping).
  - `TestAuditRejectsEmpty`: empty domain and empty resolver each error.
- [ ] **Step 2: Run, verify fail.**
- [ ] **Step 3: Implement** `Audit`. Helper `cleanTXT`/`querySet` as needed; strip surrounding quotes from TXT values before classifying. DNSSEC: read `res.Authenticated` from the NS response; ok when true, else warn with the resolver-caveat note (unless the NS query errored, then DNSSEC is also error/unknown).
- [ ] **Step 4: Run, verify pass**, then full `go test ./internal/dnsaudit/`.
- [ ] **Step 5: Commit** (`feat(dnsaudit): networked Audit across NS/MX/SPF/DMARC/DNSSEC`).

### Task 3: Wails method + CLI `dnsaudit`

**Files:**
- Create: `internal/app/dnsaudit_methods.go`, `internal/app/dnsaudit_methods_test.go`
- Modify: `internal/cli/cli.go`
- Create: `internal/cli/printdnsaudit_test.go` (`package cli`, internal)

**Interfaces:**
- Produces:
  - `func (a *App) DNSAudit(domain, resolver string) (dnsaudit.Report, error)` — defaults resolver from `sysconf.Resolvers()`; calls `dnsaudit.Audit` with an 8s timeout.
  - CLI: `case "dnsaudit": return runDNSAudit(...)`; `runDNSAudit` (flags `--resolver`, `--timeout` 8s default, `--json`, one positional domain; resolver default from sysconf); `printDNSAuditReport(w io.Writer, r dnsaudit.Report)` printing summary + per-check lines (name, status, note, records).

- [ ] **Step 1 (app): Write failing test** `dnsaudit_methods_test.go`: start a local DNS server answering a healthy domain; assert `DNSAudit("example.test", addr)` returns a populated Report, no error, and `DNSAudit("", addr)` errors.
- [ ] **Step 2: Run, verify fail.**
- [ ] **Step 3: Implement** `DNSAudit`.
- [ ] **Step 4: Run, verify pass.**
- [ ] **Step 5 (cli): Write failing tests** `printdnsaudit_test.go`:
  - `TestPrintDNSAuditReport`: a Report with ok/warn/missing checks → output contains the summary, each check name, status words, and a record value.
  - `TestRunDNSAuditMissingArg`: `runDNSAudit(nil,…)` returns exit 2.
- [ ] **Step 6: Run, verify fail.**
- [ ] **Step 7: Implement** the `dnsaudit` case, `runDNSAudit`, `printDNSAuditReport`, usage line; import `dnsaudit` (sysconf already imported).
- [ ] **Step 8: Run** `go test ./internal/app/ ./internal/cli/`, verify pass.
- [ ] **Step 9: Commit** (`feat(cli,app): dnsaudit command and Wails method`).

### Task 4: Workbench tool `DnsAudit.svelte`

**Files:**
- Create: `frontend/src/lib/tools/DnsAudit.svelte`
- Modify: `frontend/src/lib/Workbench.svelte` (import, tools entry, route)
- Regenerate: bindings (`App.js`/`App.d.ts`/`models.ts` only).

- [ ] **Step 1: Regenerate bindings** headlessly (`$(go env GOPATH)/bin/wails generate module`), then `git checkout -- frontend/wailsjs/runtime`. Verify `DNSAudit` in `App.d.ts` and a `dnsaudit` namespace in `models.ts`.
- [ ] **Step 2: Write `DnsAudit.svelte`** — domain input, resolver field pre-filled from `DNSSystem()`, Audit button; a self-explaining lede; an overall posture banner (green all-ok / amber otherwise); a per-check list with status pills (OK/Warn/Missing/Error), note, and raw records in mono. Reuse the blocklist/TLS visual language and theme tokens.
- [ ] **Step 3: Wire into `Workbench.svelte`**: import, add `{ id: "dnsaudit", name: "DNS audit", group: "Internet", ready: true }` after the blocklist entry, add the `{:else if active === "dnsaudit"}<DnsAudit />` route.
- [ ] **Step 4: Build** `cd frontend && npm run build`; verify green.
- [ ] **Step 5: Manual check** via the running `wails dev` (localhost:34115): audit a real domain; confirm the posture banner + per-check list read clearly.
- [ ] **Step 6: Commit** (`feat(app,ui): DNS audit tool in the Workbench`).

---

## Self-Review notes

- Spec coverage: §4.6 DNS audit → Tasks 1–4; NS/MX/SPF/DMARC/DNSSEC → Task 2; SPF/DMARC grading → Task 1; resolver default → Task 3/4; posture summary → Task 2.
- Review Focus → each maps to a named test (SERVFAIL, `+all`/bare, multiple SPF, `p=none`, null MX, quoted TXT).
- Proportion: signatures + test names; bodies left to the implementer.
