# nginx Config Analyzer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let an engineer paste/load an `nginx -T` dump and, offline, see what's served (server blocks, listeners, routes), what routes to which backend (upstreams), and a findings list (errors / security / best-practice / info) — for nginx and nginx Plus.

**Architecture:** A pure engine package `internal/webconf`: an nginx **parser** (brace/`;` grammar + `# configuration file` markers for source attribution) → a directive tree; a **model extractor** (server blocks, listeners, locations, upstreams, TLS, headers); a **findings + routing** pass (upstream-usage map, the finding catalogue, counts). A thin `internal/app` `AnalyzeWebServer` method and a CLI adapter. A `WebServerAnalyzer.svelte` tool in the Config insight group (server type = nginx), reusing the Firewall analyzer's visual patterns.

**Tech Stack:** Go 1.23 (`strings`, `strconv`), Wails v2 + Svelte. Tests use `stretchr/testify` against crafted `nginx -T`-style fixtures. No network, no running nginx.

**Spec:** `docs/superpowers/specs/2026-11-10-meridian-nginx-analyzer-design.md`

## Global Constraints

- **Offline & pure:** `internal/webconf` does no I/O; config text comes in as a string.
- **nginx + Plus:** Plus directives are parsed like any other and **never flagged** (there is no "unknown directive" finding in v1).
- **Severity/category:** every Finding has a severity (**error / warning / info**) and a category (**breaking / security / best-practice**); security and best-practice are warnings.
- **False-positive rules (must hold):** duplicate `server_name` only when the same name is on the **same listen host:port**; undefined-upstream only when a `proxy_pass` target is a **bare name** (no scheme, no `host:port`, no `$var`) matching no `upstream`.
- **Inheritance:** `server_tokens` and security-header checks are **http→server inheritance-aware** (a directive at `http` covers every vhost).
- CLI + JSON per §4.9. Commit attribution: `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`; PR body ends with the Generated-with line.

## Review Focus

- **Duplicate `server_name` false positive:** the same name on `:80` and `:443` is NOT an error; the same name twice on the same host:port IS. (Task 3 test.)
- **Undefined-upstream false positive:** `proxy_pass http://host:8080`, `proxy_pass https://x`, and `proxy_pass http://$backend` are never flagged; a bare `proxy_pass http://api` with no `upstream api` IS. (Task 3 test.)
- **Inheritance:** HSTS/`server_tokens` set once at `http` level satisfies every vhost (no per-vhost false positive). (Task 3 test.)
- **nginx Plus directives** (`health_check`, `api`, `keyval`, `sticky`, `zone_sync`) parse and never produce a finding. (Task 2 test.)
- **`nginx -T` markers + preamble:** file:line attribution follows `# configuration file X:` markers (line resets per file); the `nginx: … test is successful` preamble is skipped. (Task 1 test.)
- **Empty/minimal config:** valid empty Analysis, non-nil slices, no crash. (Task 2/3 tests.)

---

### Task 1: nginx parser (brace/`;` grammar + source markers)

**Files:**
- Create: `internal/webconf/parser.go`
- Test: `internal/webconf/parser_test.go`

**Interfaces:**
- Produces:
  - `type Directive struct { Name string; Args []string; Block []*Directive; File string; Line int }`
  - `func parse(text string) ([]*Directive, error)` — returns top-level directives. Grammar: a directive is `name args… ;` or `name args… { <directives> }`; `#` comments run to end of line; double-quoted/single-quoted args are one token (quotes stripped). A `# configuration file <path>:` comment sets the current source file and resets the per-file line counter; other `#` lines (incl. the `nginx: … ` preamble) are ignored. Each Directive records the File + Line where it starts. An unclosed `{` returns an error.
  - `func (d *Directive) children(name string) []*Directive` — direct child directives with the given name.
  - `func (d *Directive) arg(i int) string` — the i-th arg or "".

- [ ] **Step 1: Write failing tests** `parser_test.go` (`package webconf`):
  - `TestParseNestedBlocks`: `http { server { listen 80; location / { return 200; } } }` → nested tree; `children` navigation works.
  - `TestParseDirectiveArgsAndQuotes`: `add_header X-Frame-Options "SAMEORIGIN";` → name + quoted arg intact; multiple args split.
  - `TestParseFileMarkersAndPreamble`: input with a `nginx: configuration file … test is successful` preamble and two `# configuration file /a.conf:` / `/b.conf:` sections → directives attributed to the right File with per-file Line numbers; preamble ignored.
  - `TestParseUnclosedBlockErrors`: a missing `}` → error.
- [ ] **Step 2: Run, verify fail.**
- [ ] **Step 3: Implement** `parser.go` (rune scanner: tokens, `;`/`{`/`}` terminators, quote handling, comment + marker handling, block stack).
- [ ] **Step 4: Run, verify pass.**
- [ ] **Step 5: Commit** (`feat(webconf): nginx config parser`).

### Task 2: Model extraction (servers, listeners, locations, upstreams, TLS)

**Files:**
- Create: `internal/webconf/model.go`, `internal/webconf/nginx.go`
- Test: `internal/webconf/nginx_test.go`

**Interfaces:**
- Consumes: Task 1's `Directive` tree.
- Produces (model.go):
  - `type Listener struct { Addr string; Port int; SSL, HTTP2, Default bool }`
  - `type Location struct { Path, Kind, Target string; Flags []string }` (Kind: static | proxy | redirect | alias)
  - `type Server struct { Names []string; Listeners []Listener; Root string; Locations []Location; TLSProtocols []string; TLSCert string; Access []string; ServerTokens string; Headers []string; File string; Line int }`
  - `type Upstream struct { Name string; Servers []string; Policy string; UsedBy []string; File string; Line int }`
  - `type Finding struct { Severity, Category, Title, Detail, File string; Line int }` (filled in Task 3)
  - `type Counts struct { Servers, Listeners, Upstreams, Locations, Errors, Warnings int }`
  - `type Analysis struct { ServerType string; Servers []Server; Upstreams []Upstream; Findings []Finding; Counts Counts; Notes []string }` (all slices init non-nil)
  - (nginx.go) `func extract(dirs []*Directive) Analysis` — walk `http { server { … } upstream { … } }`: build Servers (from each `server` block: `server_name`, `listen` → Listener incl. `ssl`/`http2`/`default_server`, `root`, each `location` → Location with proxy_pass/return/alias/try_files, `ssl_protocols`, `ssl_certificate`, `allow`/`deny`, `server_tokens`, `add_header` names) and Upstreams (`upstream` blocks: `server` lines, a policy directive like `least_conn`/`ip_hash`). Plus directives are read through without error.

- [ ] **Step 1: Write failing tests** `nginx_test.go`:
  - `TestExtractServersAndListeners`: two server blocks incl. `listen 443 ssl http2` and `listen 80 default_server` → Names, Listener fields.
  - `TestExtractLocationsAndProxy`: locations with `proxy_pass http://api`, `return 301 …`, `try_files`, `alias` → Kind + Target.
  - `TestExtractUpstreams`: `upstream api { least_conn; server 10.0.0.1:8080; server 10.0.0.2:8080; }` → Servers + Policy.
  - `TestExtractTLSAndHeaders`: `ssl_protocols`, `ssl_certificate`, `server_tokens off`, `add_header` names captured.
  - `TestExtractPlusDirectivesNoError`: a config using `health_check`, `api`, `keyval` extracts normally and yields no Findings from extraction.
  - `TestExtractEmpty`: empty/minimal config → Analysis with non-nil empty slices.
- [ ] **Step 2–4:** fail → implement → pass.
- [ ] **Step 5: Commit** (`feat(webconf): nginx model extraction`).

### Task 3: Routing map + findings + counts

**Files:**
- Create: `internal/webconf/findings.go`
- Test: `internal/webconf/findings_test.go`

**Interfaces:**
- Produces:
  - `func analyzeFindings(a *Analysis)` — mutates `a`: fills `Upstream.UsedBy` (each location whose proxy_pass targets a bare upstream name); appends Findings; fills Counts. `extract` (Task 2) calls it before returning.
  - Findings implemented (each with file:line, severity, category):
    - **error/breaking:** `proxy_pass` to an undefined upstream (bare-name rule); duplicate `server_name` on the same listen host:port; `listen … ssl` with no `ssl_certificate` in that server.
    - **warning/security:** `ssl_protocols` includes TLSv1 or TLSv1.1; a `location` with `autoindex on`; a TLS vhost missing HSTS or `X-Content-Type-Options`/`X-Frame-Options` (inheritance-aware across http→server).
    - **warning/best-practice:** `server_tokens` not `off` (inheritance-aware); a listener with no `server_name`.
    - **info:** (hook for unresolved includes — populated by the app/CLI layer when a raw file is parsed, see Task 4).
  - Counts: Servers, Listeners (sum), Upstreams, Locations (sum), Errors (severity==error), Warnings (severity==warning).

- [ ] **Step 1: Write failing tests** `findings_test.go` (build via `extract` on fixtures):
  - `TestFindingUndefinedUpstream`: bare `proxy_pass http://api` with no upstream → one error; `proxy_pass http://10.0.0.1:8080`, `proxy_pass http://$b`, and a defined upstream → NO finding.
  - `TestFindingDuplicateServerNameSamePort`: same `server_name` on two `:443` servers → error; the same name on `:80` and `:443` → NO finding.
  - `TestFindingSslListenNoCert`: `listen 8443 ssl` with no `ssl_certificate` → error.
  - `TestFindingDeprecatedTLSAndAutoindex`: `ssl_protocols TLSv1 TLSv1.2;` → security; `autoindex on;` → security.
  - `TestFindingServerTokensInheritance`: `server_tokens off;` at `http` level → NO best-practice finding for any vhost; omitted everywhere → a finding.
  - `TestFindingSecurityHeadersInheritance`: HSTS `add_header` at `http` level → NO missing-HSTS finding; absent → finding on the TLS vhost.
  - `TestCounts`: a fixture with known servers/listeners/upstreams/locations/errors/warnings → exact Counts.
  - `TestRoutingUsedBy`: an upstream is listed as UsedBy the location that proxies to it.
- [ ] **Step 2–4:** fail → implement → pass; then `-race`.
- [ ] **Step 5: Commit** (`feat(webconf): nginx routing map + findings + counts`).

### Task 4: App method + CLI adapter

**Files:**
- Create: `internal/webconf/analyze.go`, `internal/app/webconf_methods.go`, `internal/app/webconf_methods_test.go`
- Modify: `internal/cli/cli.go` (dispatch + usage); Create: `internal/cli/webanalyze.go`, `internal/cli/cli_webanalyze_test.go`

**Interfaces:**
- Produces:
  - (webconf) `func Analyze(configText, server string) (Analysis, error)` — empty text → error; `server` defaults to "nginx"; `apache`/`caddy`/other → "not yet supported" error; parses + extracts. When the input has **no `# configuration file` markers** and contains unresolved `include` directives, add an **info** Note/Finding naming them ("source: pasted config").
  - `func (a *App) AnalyzeWebServer(configText, server string) (webconf.Analysis, error)` — thin wrapper.
  - CLI subcommand `webanalyze [<file>] [--server nginx] [--json]` — reads a file arg or stdin, prints a text summary or JSON.

- [ ] **Step 1: Write failing tests**:
  - `TestAnalyzeWebServerEmpty` (error mentions config); `TestAnalyzeWebServerNginx` (a small dump → expected servers/findings); `TestAnalyzeWebServerUnknownServer` (`apache` → "support" error).
  - `TestAnalyzeUnresolvedInclude`: a single `nginx.conf` with `include sites-enabled/*.conf;` and no markers → an info finding/note mentioning the include.
  - CLI test: feed a dump via a `webStdin` seam, assert JSON contains server names + a finding, and the text summary names them.
- [ ] **Step 2–4:** fail → implement → pass.
- [ ] **Step 5: Commit** (`feat(app,cli): AnalyzeWebServer method + webanalyze CLI`).

### Task 5: Workbench tool `WebServerAnalyzer.svelte`

**Files:**
- Create: `frontend/src/lib/tools/WebServerAnalyzer.svelte`
- Modify: `frontend/src/lib/Workbench.svelte` (import; `{ id: "webanalyze", name: "Web server analyzer", group: "Config insight", ready: true }` after `fwanalyze`; route)
- Regenerate: bindings (`AnalyzeWebServer` + `webconf` models).

- [ ] **Step 1: Regenerate bindings** (`$(go env GOPATH)/bin/wails generate module`; `git checkout -- frontend/wailsjs/runtime`); verify `AnalyzeWebServer` in `App.d.ts` and the `webconf` types in `models.ts`.
- [ ] **Step 2: Write `WebServerAnalyzer.svelte`**:
  - **Empty state** shows the instruction snippet: `sudo nginx -T > nginx-config.txt` (+ container `docker exec <c> nginx -T` and `-c /path` variants) with a **copy** button, then a paste area + **Load file** + Analyze (calls `AnalyzeWebServer(text, "nginx")`).
  - On result: count tiles (Servers, Listeners, Upstreams, Locations, Errors, Warnings — Errors/Warnings tinted); a **Sites** section (per server block: name, listen badges incl. ssl/http2, root, routes with upstream links, access, TLS, source file); an **Upstreams** table (servers, policy, used-by); a **Findings** list grouped/sorted by severity with category tag + file:line.
  - Reuse the firewall tool's visual language; guard null/empty arrays; phone-width safe.
- [ ] **Step 3: Wire into `Workbench.svelte`** (import, Config-insight entry, route).
- [ ] **Step 4: Build** `cd frontend && npm run build`; verify green.
- [ ] **Step 5: Manual check** in the running app: paste a sample `nginx -T` dump (a fixture), confirm tiles/sites/upstreams/findings render and the empty-state instructions show; no console errors.
- [ ] **Step 6: Commit** (`feat(app,ui): Web server analyzer tool (nginx)`).

---

## Self-Review notes

- Spec coverage: parse+markers → T1; servers/listeners/locations/upstreams/TLS/Plus → T2; routing map + all finding tiers + counts + inheritance + false-positive rules → T3; app+CLI+unresolved-include → T4; UI with nginx -T instructions → T5.
- Review Focus → each maps to a named test (dup server_name same-port, undefined-upstream bare-name, inheritance, Plus-not-flagged, markers+preamble, empty config).
- Proportion: signatures + test names; parser/extractor/findings bodies left to the implementer. No secrets in fixtures; no running nginx.
- Routing is table-only in v1 (a server→upstream diagram is an optional follow-on, per the design).
