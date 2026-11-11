# Multi-server Web Analyzer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Extend `internal/webconf` to analyze Apache httpd and Caddy configs (in addition to nginx), filling the existing `Analysis` model and running the shared findings engine with server-type gates; add a server selector to the UI.

**Architecture:** `model.go` + `findings.go` (`analyzeFindings`) are reused. Each new server gets `analyzeX(configText string) (Analysis, error)` that parses its config, fills `Analysis`, sets lowercase `ServerType`, and calls `analyzeFindings`. `analyze.go` dispatches on server. `analyzeFindings` gains server-type gates. nginx path untouched.

**Tech Stack:** Go 1.23, hand-rolled parsers (no new deps); Svelte frontend; Wails binding already accepts `server`.

**Spec:** `docs/superpowers/specs/2026-11-10-web-multiserver-design.md`

## Global Constraints

- No new dependencies; parsing in-package, mirroring nginx.go style.
- Secrets (SSL key files' contents, `tls` keys, passwords) never surfaced; cert/key *paths* are fine.
- Each new server returns ONE Analysis; `ServerType` set lowercase before `analyzeFindings`.
- Existing nginx findings behavior must stay byte-for-byte identical (findings_test.go asserts titles/categories): every gate's `ServerType=="nginx"` branch preserves current logic.
- Fixtures must reflect real export structure (owner rule).

## Review Focus

- **Direct-URL proxy not flagged** (Apache/Caddy): a `ProxyPass`/`reverse_proxy` to a URL must NOT raise the nginx "undefined upstream" breaking error. (Task 1 + 2/3.)
- **Caddy auto-HTTPS**: a bare-hostname site with no `tls` must NOT raise missing-cert, but SHOULD get HSTS/security-header findings. (Task 1 + 3.)
- **Apache version disclosure**: `ServerTokens` absent or non-`Prod` → warning; `Prod` → none. (Task 1 + 2.)
- **Wrong-format paste / zero servers**: no panic, no spurious version-leak finding. (Task 1 gate requires servers.)
- **Secret leakage**: `SSLCertificateKeyFile` path ok; no key material surfaced. (Task 2/3 secret test.)

---

### Task 1: Server-type gates in `analyzeFindings`

**Files:**
- Modify: `internal/webconf/findings.go`
- Modify: `internal/webconf/findings_test.go` (add gate tests using hand-built Analysis structs)

**Interfaces:**
- Consumes: `Analysis.ServerType`, model fields.
- Produces: gated `analyzeFindings` — no signature change (reads `a.ServerType`).

Gates (each preserves nginx behavior exactly):
1. Undefined-upstream routing check + `UsedBy`: run only when `a.ServerType == "nginx"`.
2. Missing-cert-on-TLS: skip when `a.ServerType == "caddy"`.
3. `effectiveHeader`: for non-nginx types, return "is the header present on the server" (no nginx all-or-nothing http inheritance).
4. Version-disclosure (`serverTokensLeak` + finding): nginx = existing; caddy = never; apache = leaking unless any server's `ServerTokens` equalsFold `Prod`, and only when `len(Servers) > 0`. Remediation copy branches on server type.

- [ ] **Step 1: Write failing tests** in `findings_test.go` (build `Analysis` structs directly, set `ServerType`, call `analyzeFindings`):
  - `TestFindingsCaddyNoMissingCert`: ServerType caddy, a server with an SSL listener and empty TLSCert → NO "no ssl_certificate" error, but YES missing-HSTS warning.
  - `TestFindingsApacheProxyNotFlagged`: ServerType apache, a proxy Location with Target `http://10.0.0.5:8080` and no Upstreams → NO "undefined upstream" error.
  - `TestFindingsApacheVersionDisclosure`: ServerType apache, server with `ServerTokens=""` → version-disclosure warning; with `ServerTokens="Prod"` → none.
  - `TestFindingsApacheEmptyNoLeak`: ServerType apache, `len(Servers)==0` → no version-disclosure finding.
  - `TestFindingsNginxUnchanged`: an nginx Analysis identical to an existing case still yields the same findings (guard against regressions).
- [ ] **Step 2: Run tests, verify fail** (`go test ./internal/webconf/ -run TestFindings`).
- [ ] **Step 3: Implement** the four gates in `findings.go`.
- [ ] **Step 4: Run tests** — new gate tests pass AND the full existing suite passes (`go test ./internal/webconf/`).
- [ ] **Step 5: Commit** (`refactor(webconf): server-type-aware findings gates`).

---

### Task 2: Apache httpd analyzer

**Files:**
- Create: `internal/webconf/apache.go`, `internal/webconf/apache_test.go`
- Modify: `internal/webconf/analyze.go` (add `case "apache"`)

**Interfaces:**
- Produces: `analyzeApache(configText string) (Analysis, error)`.
- Consumes: model types, `analyzeFindings`.

Parser: line-oriented, directives are case-insensitive, `<Section attrs>…</Section>` containers, line continuation on trailing `\`, `#` comments. Build a light tree of sections + directive lines. Map:
- `<VirtualHost addr:port [addr2:port2]>` → one `Server`; each `addr:port` → `Listener` (`:443` or inner `SSLEngine on` → SSL).
- `ServerName`, `ServerAlias` → `Names`; `DocumentRoot` → `Root`.
- `<Location path>` / `ProxyPass path target` → `Location` (proxy with Target=backend URL; else static); `Redirect`/`RewriteRule ... [R]` → redirect.
- `SSLProtocol` → `TLSProtocols` (enabled set, e.g. parse `all -SSLv3 -TLSv1` → list the still-enabled TLS versions, or record raw tokens for the deprecated-version check); `SSLCertificateFile` → `TLSCert`.
- `Header [always] set <Name> …` → `Headers` (append `<Name>`); `ServerTokens`/`ServerSignature` → `ServerTokens`; `Options … Indexes` in a `<Directory>`/`<Location>` → the owning/representative Location gets `Flags:["autoindex on"]`.
- `Include`/`IncludeOptional` present → emit an info "unresolved Include" note.
- Reject/guard: a paste whose first meaningful line looks like nginx (`http {`) or Caddy → still parse (produce ~0 servers), never panic.

- [ ] **Step 1: Write failing tests** in `apache_test.go` against a realistic httpd fixture (a `:80`→`:443` redirect vhost + a TLS vhost with ServerName/ServerAlias, DocumentRoot, a `ProxyPass` to a backend URL, `SSLProtocol`, `SSLCertificateFile`, a `Header always set Strict-Transport-Security`, `ServerTokens Prod`, and an `SSLCertificateKeyFile` secret-ish path):
  - `TestApacheVHostsListeners`: 2 servers, names + root, SSL listener on 443.
  - `TestApacheProxyLocation`: the ProxyPass location has Kind proxy, Target the backend URL, and is NOT flagged undefined.
  - `TestApacheTLSAndHeaders`: TLSCert set, HSTS header present (no missing-HSTS finding), ServerTokens Prod (no version finding).
  - `TestApacheSecretNotSurfaced`: the key file path/content marker not surfaced as a secret value (cert path ok).
  - `TestApacheDeprecatedTLS`: an `SSLProtocol` enabling TLSv1 → deprecated-TLS warning.
- [ ] **Step 2: Run, verify fail.**
- [ ] **Step 3: Implement** `analyzeApache` in `apache.go`; add `case "apache"` to `analyze.go`.
- [ ] **Step 4: Run, verify pass** + full suite.
- [ ] **Step 5: Commit** (`feat(webconf): Apache httpd analyzer`).

---

### Task 3: Caddy (Caddyfile) analyzer

**Files:**
- Create: `internal/webconf/caddy.go`, `internal/webconf/caddy_test.go`
- Modify: `internal/webconf/analyze.go` (add `case "caddy"`)

**Interfaces:**
- Produces: `analyzeCaddy(configText string) (Analysis, error)`.
- Consumes: model types, `analyzeFindings`.

Parser: brace blocks, newline-terminated directives (no `;`), `#` comments. Distinguish: a leading **global options** block `{ … }` (no address) and snippets `(name) { … }` → not servers (note as unresolved if `import` used). Each site block (address line, possibly `a.com, b.com`) → one `Server`:
- address forms: `https://host` or bare `host` → `Listener{Port:443, SSL:true}`; `http://host` or `:80` → `Port:80`; `host:PORT` → that port.
- addresses → `Names`; `root * <path>` → `Root`; `reverse_proxy <backend…>` → `Location{Kind:proxy, Target: first backend}`; `file_server browse` → a Location flag `autoindex on`; `tls <cert> <key>` (file form) → `TLSCert`; `tls <email>` / `tls internal` → automatic (leave cert empty); `header [always] <Name> <val>` or `header { <Name> <val> … }` → `Headers`; `redir` → redirect Location; `import <x>` → unresolved note.

- [ ] **Step 1: Write failing tests** in `caddy_test.go` against a realistic Caddyfile (a bare-hostname auto-HTTPS site with `reverse_proxy`, a `header` security directive, a second site with explicit `tls cert key` + `file_server browse`, a global `{ }` block, and a `tls you@example.com` email form that must not leak):
  - `TestCaddySitesListeners`: 2 servers (global block not a server), names, 443 SSL listeners.
  - `TestCaddyReverseProxy`: proxy Location Target = backend, not flagged undefined.
  - `TestCaddyAutoHTTPSNoCertFinding`: bare-hostname site, no `tls` → NO missing-cert finding, but missing-HSTS warning present.
  - `TestCaddyBrowseAutoindex`: `file_server browse` → autoindex finding.
  - `TestCaddySecretNotSurfaced`: the `tls` email / key not surfaced.
- [ ] **Step 2: Run, verify fail.**
- [ ] **Step 3: Implement** `analyzeCaddy` in `caddy.go`; add `case "caddy"` to `analyze.go`.
- [ ] **Step 4: Run, verify pass** + full suite.
- [ ] **Step 5: Commit** (`feat(webconf): Caddy Caddyfile analyzer`).

---

### Task 4: UI server selector + generalization

**Files:**
- Modify: `frontend/src/lib/tools/WebServerAnalyzer.svelte`
- Modify: `internal/app/webconf_methods.go` (doc comment: list servers)

**Interfaces:** Consumes `AnalyzeWebServer(configText, server)` (unchanged signature).

Changes:
- Add a server `<select>` (nginx | Apache | Caddy) bound to `server`; pass it to `AnalyzeWebServer` (replace hardcoded `"nginx"`); re-analyze on change.
- Per-server input hint + dump command (`sudo nginx -T` / the Apache config path or `apachectl -S` / the `Caddyfile`); generalize the nginx-specific lede/placeholder.
- Keep the existing server/upstream/findings rendering (model unchanged). Upstreams card only shows for nginx (Apache/Caddy produce none) — hide it when empty (likely already conditional; verify).

- [ ] **Step 1: Implement** Svelte changes.
- [ ] **Step 2: Build** `npm run build` → no errors.
- [ ] **Step 3: Manual check** (build + Playwright if the dev server is up): select each server, paste the task fixture, confirm servers/findings render.
- [ ] **Step 4: Commit** (`feat(app,ui): web analyzer server selector (Apache/Caddy)`).

---

## Self-review notes

- Spec coverage: findings gates (T1), Apache (T2), Caddy (T3), UI+dispatch (T2/T3/T4). All design sections covered.
- Review Focus: direct-URL proxy (T1/T2/T3 tests), Caddy auto-HTTPS (T1/T3), Apache version (T1/T2), wrong-format/zero-servers (T1 gate), secret leakage (T2/T3).
- Types consistent: all extractors fill the same model and call `analyzeFindings`; `ServerType` set before.
- Proportion: plan gives signatures, directive→model mappings, and test names/assertions, not parser bodies.
