# Multi-server web config analyzer — design

**Date:** 2026-11-10
**Status:** design (awaiting builder-check, then plan)

## Intent & why

Meridian's web-server analyzer (`internal/webconf`) reads an **nginx** config
(`nginx -T` dump) offline and produces a structured view — server blocks,
listeners, locations, upstreams, TLS — plus a **findings** list (breaking
errors, security warnings, best-practice notes). The owner asked to extend it to
the other two servers engineers actually run: **Apache httpd** and **Caddy**.

The valuable, already-built core is the **neutral-ish model** (`model.go`:
Server/Listener/Location/Upstream/Finding/Counts) and the **findings engine**
(`findings.go` → `analyzeFindings`), which operate on the filled model. Adding a
server means writing a parser+extractor that fills the same `Analysis`, then
running `analyzeFindings`. Most findings (missing cert on a TLS listener,
deprecated TLS versions, missing HSTS / security headers, directory listing,
undefined proxy target, no server name) are server-agnostic once the model is
filled.

## Goals

- Parse **Apache httpd** and **Caddy (Caddyfile)** configs offline into the
  existing `Analysis` model (servers, listeners, locations, proxy targets, TLS,
  headers), set `ServerType` accordingly, and run the shared findings.
- Make `analyzeFindings` **server-type-aware** where servers genuinely differ
  (below), instead of emitting nginx-only findings for Apache/Caddy.
- Add a **server selector** (nginx / Apache / Caddy) to the UI (currently
  hardcodes `"nginx"`), with per-server input hints.
- Validate each against a **realistic fixture** (owner rule: crafted toy
  fixtures have hidden Criticals twice before) that includes a TLS vhost, a
  reverse proxy, and a header/security directive.

## Non-goals / anti-goals

- Not a config editor/validator substitute (no `apachectl -t` / `caddy validate`
  equivalence); it's a reading, offline, same as today.
- Not every directive. Target what feeds the model + findings: vhosts/sites,
  listen/ports, document root, locations & reverse proxy, TLS (protocols/cert),
  security headers, version disclosure, directory listing.
- No new dependencies — hand-rolled parsers in-package, matching nginx.go.
- No multi-file include resolution for Apache/Caddy in this slice (nginx relies
  on `nginx -T` concatenation; Apache `Include`/Caddy `import` are noted as
  unresolved, like the existing nginx single-file case).

## Config format targeted per server

| Server | Format | Container syntax | Example directives |
|--------|--------|------------------|--------------------|
| nginx | `nginx -T` dump / single file | `{ … }` + `;` | (existing) |
| Apache | httpd.conf / apache2.conf (+ `apachectl -S` style single file) | `<Section>…</Section>` + line directives | `Listen`, `<VirtualHost *:443>`, `ServerName`, `ServerAlias`, `DocumentRoot`, `<Location>`, `ProxyPass`, `SSLEngine`, `SSLProtocol`, `SSLCertificateFile`, `Header`, `ServerTokens`, `Options Indexes` |
| Caddy | Caddyfile | `address { … }` (brace, newline-terminated, no `;`) | site address line, `root`, `reverse_proxy`, `file_server [browse]`, `tls`, `header`, `redir`, `encode` |

## Model mapping

Reuse `Server`/`Listener`/`Location`/`Upstream` as-is; set `Analysis.ServerType`.

- **Apache** `<VirtualHost addr:port>` → one `Server`; the `addr:port` →
  `Listener` (`:443` or `SSLEngine on` → `SSL=true`); `ServerName`+`ServerAlias`
  → `Names`; `DocumentRoot` → `Root`; `<Location>`/`<Directory>` and
  `ProxyPass /x http://host/` → `Location` (`proxy` with Target = backend,
  `static`, or `redirect` from `Redirect`/`RewriteRule [R]`); `SSLProtocol`
  (e.g. `all -SSLv3 -TLSv1`) → `TLSProtocols` (record the enabled ones); 
  `SSLCertificateFile` → `TLSCert`; `Header [always] set <Name>` → `Headers`;
  `ServerTokens`/`ServerSignature` → `ServerTokens` (version disclosure);
  `Options +Indexes`/`Options Indexes` in a location → `Flags: ["autoindex on"]`.
  Apache has no first-class upstream pool; `ProxyPass` targets a URL directly, so
  `Location.Kind="proxy"`, `Target` = the backend URL (no `Upstream` rows unless
  `<Proxy balancer://name>` is present → optional `Upstream`).
- **Caddy** each site block (address line, possibly comma-separated addresses) →
  one `Server`; a bare hostname or `https://` → `Listener{Port:443, SSL:true}`
  (Caddy auto-HTTPS), `http://`/`:80` → `Port:80, SSL:false`; the address
  hostname(s) → `Names`; `root * <path>` → `Root`; `reverse_proxy <backend>` →
  `Location{Kind:"proxy", Target:backend}`; `file_server browse` → autoindex
  flag; `tls <cert> <key>` → `TLSCert` (a `tls internal`/email form means
  automatic — leave cert empty, NOT a finding); `header <Name> <val>` /
  `header { Name val }` → `Headers`; `redir` → redirect location.

## Findings: server-type-aware gates

`analyzeFindings` runs for all three; these gates prevent false findings:

1. **Missing cert on TLS listener** (`listen ssl` with no cert → breaking): keep
   for nginx & Apache; **skip for Caddy** (auto-HTTPS provisions a cert, so an
   empty `TLSCert` with SSL is normal, not broken).
2. **`server_tokens`/version disclosure not off**: nginx `server_tokens`,
   Apache `ServerTokens`/`ServerSignature`; **skip for Caddy** (Caddy sends no
   version in its Server header by default). Generalize the check to read the
   model's `ServerTokens` field + server type; the current nginx-only
   `a.httpTokens` path stays nginx-specific.
3. **add_header all-or-nothing inheritance** (`effectiveHeader`): this is an
   nginx quirk; for Apache/Caddy a server's headers are simply its own
   (plus global for Caddy/Apache, which this slice treats as server-local). Make
   `effectiveHeader` fall back to a plain "is the header present on the server"
   check for non-nginx types.
4. **Deprecated TLS, missing HSTS/security headers, autoindex, undefined proxy
   target, no server name, duplicate names**: server-agnostic — keep for all.

Concretely: thread `a.ServerType` into the gated checks (it's already on the
Analysis). Where a finding is nginx-specific copy ("set server_tokens off;"),
branch the remediation text on server type.

## File structure

- `internal/webconf/apache.go` + `apache_test.go` — `analyzeApache(configText) (Analysis, error)`.
- `internal/webconf/caddy.go` + `caddy_test.go` — `analyzeCaddy(configText) (Analysis, error)`.
- `internal/webconf/analyze.go` — dispatch `server` to nginx (existing
  parse+extract) / apache / caddy; drop the "nginx only" rejection.
- `internal/webconf/findings.go` — add server-type gates (points 1–3 above);
  `extract` (nginx) and the two new extractors all call `analyzeFindings`.
- `internal/webconf/model.go` — `newAnalysis` takes/sets `ServerType` (or the
  callers set it); add nothing structural if avoidable.
- `internal/app/webconf_methods.go` — doc comment lists servers (binding already
  takes `server`).
- `frontend/src/lib/tools/WebServerAnalyzer.svelte` — add server `<select>`
  (nginx | Apache | Caddy); per-server dump hint (`nginx -T` / `apachectl -S`
  or the config path / the Caddyfile); generalize nginx-specific copy; pass the
  selected server to `AnalyzeWebServer`; re-analyze on change.

## Review focus (inputs the happy-path tests won't cover)

- **Apache line continuations** (`\` at EOL) and case-insensitive directives
  (`ServerName` vs `servername`) — Apache directives are case-insensitive.
- **Apache `<VirtualHost>` with multiple addresses** (`<VirtualHost *:80 *:443>`).
- **Caddy**: a site block with multiple comma-separated addresses; a global
  options block `{ … }` at the top (no address) must not become a bogus server;
  `reverse_proxy` with multiple upstreams or a block body; snippets `(name) { }`
  and `import`.
- **Caddy auto-HTTPS**: a bare-hostname site with no `tls` must NOT raise the
  missing-cert finding, and SHOULD still get HSTS/security-header findings (it
  serves TLS).
- **Secret leakage**: Apache `SSLCertificateKeyFile`/password, Caddy `tls` with
  an email or key — never surface key material; cert *paths* are fine.
- **Wrong-format paste**: an nginx config pasted under "Apache", or vice-versa —
  should not panic; ideally yields few/zero servers rather than garbage.

## Builder-check resolutions (settled calls)

1. **Undefined-proxy-target finding is nginx-only.** Apache `ProxyPass` and Caddy
   `reverse_proxy` keep `Target = backend URL` (so the UI shows the destination).
   The breaking "proxy_pass to undefined upstream" check + the `UsedBy`
   population are gated to `ServerType=="nginx"`; Apache/Caddy proxy locations are
   never flagged for a missing upstream pool (they have none). No model change.
2. **Version-disclosure gate is server-type-aware.** Apache extractor stores the
   `ServerTokens` directive value (Prod/Full/OS/Minor/Major/Min/Minimal) in
   `Server.ServerTokens`; the gate treats Apache as leaking **unless the value is
   `Prod`** (only `Prod` suppresses the version — e.g. `Min` still emits
   `Apache/2.4.x`). Caddy never leaks (skip entirely). nginx keeps its existing
   `server_tokens`/`httpTokens` logic. The finding is emitted only when
   `len(Servers) > 0` for apache (so a garbage/wrong-format paste yields nothing),
   and the remediation copy branches on server type (`server_tokens off;` vs
   `ServerTokens Prod` + `ServerSignature Off`).
3. **Include/import note per server.** The `markerPrefix`/`collectIncludes`
   unresolved-include note in `Analyze` runs for **nginx only**. The Apache and
   Caddy extractors emit their own "unresolved Include/import" note internally
   when they encounter `Include`/`IncludeOptional` (Apache) or `import` (Caddy),
   since those formats are always single-file here.
4. **ServerType casing.** Each extractor sets lowercase `a.ServerType`
   ("apache"/"caddy") before calling `analyzeFindings`; `newAnalysis` keeps
   defaulting to "nginx", so existing nginx tests are untouched. The UI select
   may pass any case — `Analyze` lowercases the dispatch key.

Also settled from the builder's lean:
- Apache `<Directory>` blocks are read only for the `Options Indexes` autoindex
  finding; `Location` rows come from `<Location>` + `ProxyPass` (don't flood).
- Caddy global-options/snippet headers are treated as server-local this slice;
  a global block or `import` is noted as unresolved.
