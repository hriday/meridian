# Meridian — Web server config analyzer: nginx (offline)

**Date:** 2026-11-10
**Status:** design, ready to plan (mockup validated by owner)
**Area:** "Config insight" (joins the Firewall analyzer). This is the first
**web server** analyzer; Apache and Caddy are planned siblings, each its own
parser + slice. nginx **and nginx Plus** are one parser (Plus is the same
syntax plus extra directives).

## Intent

An engineer drops a whole box's nginx config and, **entirely offline**,
Meridian tells them two things: **what's going on** (what's served, what
listens where, what routes to which backend) and **whether it's good** (a
findings list — breaking errors, security, and best-practice). It understands
nginx Plus directives, so they aren't mistaken for errors.

## Input — the whole box via `nginx -T` (owner's call)

The input is the **`nginx -T` dump**: that command prints every config file's
text, each section preceded by a `# configuration file /path/...:` marker, so
**all the included files are present** in one blob (it does not literally
inline them — a file included inside `http` appears as a top-level `server {}`
/ `upstream {}` block after the `http { include …; }` block; the analyzer
treats top-level server/upstream blocks as belonging to the http context). The
operator runs `nginx -T > conf.txt` and **pastes or loads** it. This gives us
the complete config without an include-resolution engine and lets us attribute
each finding to its **source file:line**.

- A single raw `nginx.conf` (or any fragment) is also accepted; when its
  `include` directives can't be resolved (no `nginx -T`), Meridian parses what
  is present and raises an **info finding** naming the unresolved includes,
  rather than implying the picture is complete.
- **Non-goal (planned next):** uploading a **zip** of `/etc/nginx` and
  resolving includes across the archived files. Deferred because it needs its
  own include-resolver (relative/absolute paths + globs) — the one thing
  `nginx -T` hands us already done. Owner confirmed `nginx -T` is a reasonable
  ask.

Everything is offline; the config text never leaves the machine.

### In-app instructions (the empty state tells them how)

The tool's **empty state** shows the exact command to produce the input — the
app's users are technically sound, so this is a short copy-pasteable snippet,
not a tutorial:

```
# Dump the full running config (nginx.conf + every include, resolved):
sudo nginx -T > nginx-config.txt
```

with one-liners for the common variants:
- **In a container:** `docker exec <container> nginx -T > nginx-config.txt`
- **Non-default config path:** `nginx -T -c /path/to/nginx.conf > nginx-config.txt`

…then "paste the output below, or **Load file**." A **copy** affordance on the
command. (`nginx -T` runs a config test and prints the full config; it needs
read access to the config, hence `sudo` on most boxes.)

## Output (validated mockup)

**Count tiles:** Server blocks · Listeners · Upstreams · Locations · Errors ·
Warnings.

**1. What's going on — sites.** Per **server block**: the `server_name`s, its
**listeners** (port, `ssl`, `http2`, `default_server`) as badges, the `root`,
the **routes** (each `location`: static `try_files`/`alias`, a `proxy_pass` to
a named upstream, or a `return`/redirect), any `allow`/`deny` access, the
**TLS** (protocols + cert path), and the **source file** it came from.

**What routes to what — upstreams.** A table of each `upstream`: its servers,
balancing policy (`least_conn`, `ip_hash`, …), and **which server/location
uses it** — the nginx analog of the firewall's "what talks to what". An
upstream referenced by `proxy_pass` but never defined is flagged here (and as
an error).

**2. Is it good — findings.** A severity-tagged list, each with a one-line
explanation and `file:line`:
- **Errors (breaking):** `proxy_pass` to an undefined `upstream`; duplicate
  `server_name`; `listen … ssl` with no `ssl_certificate`; (parse-level)
  unclosed/mismatched blocks.
- **Security:** deprecated `ssl_protocols` (TLSv1/1.1); `autoindex on`;
  missing HSTS/security headers on TLS vhosts; overly broad `allow all` on a
  sensitive location.
- **Best-practice:** `server_tokens` not `off`; a listener with no
  `server_name`; `http2` missing on a 443 listener; large/》default
  `client_max_body_size` surprises. *(Starter set — extensible.)*
- **Info:** unresolved `include` globs (when not fed `nginx -T`).

The owner reviewed a mockup carrying all three finding tiers (errors +
security + best-practice) and the sites/upstreams/routes layout, and approved
the direction.

Held to Meridian's Nielsen bar: fast, legible, self-explaining, phone-width
safe, matching the existing tool visual language.

## Scope

**In scope (v1):** offline parsing of an `nginx -T` dump (or a single config
file) for nginx **and nginx Plus**; the server-block / listener / location /
upstream / TLS model with per-file attribution; the routing (upstream-usage)
map; the findings engine across error / security / best-practice / info; and a
"Web server analyzer" Workbench tool (server type = nginx) in Config insight.

**Non-goals / later (my calls):**
- **Zip-of-directory upload + include resolution** — the planned next
  increment for nginx.
- **Apache** and **Caddy** — planned sibling parsers under the same tool/model.
- Running real `nginx -t` (needs the binary + loaded modules); we do offline
  **structural + heuristic** checks and say so plainly — not a validator
  replacement.
- `stream {}` (TCP/UDP proxy), `map`/`geo` evaluation, Lua/njs, and full
  rewrite-rule simulation — parsed only as far as the model needs; deep
  evaluation is out.

## Decided (my calls, from design review)

- **Severity vs category.** Three severities — **error / warning / info** — plus
  a category tag on each finding (**breaking / security / best-practice**).
  Security and best-practice are both *warnings* (the tiles are Errors and
  Warnings; info findings don't get a tile).
- **False-positive posture on the two flagship errors:**
  - *Duplicate `server_name`* is flagged only when the same name appears on the
    **same listen host:port** (the same name on :80 and :443 is legitimate),
    within the same context.
  - *Undefined upstream* is flagged only when a `proxy_pass` target is a **bare
    name** (no `http(s)://` scheme, no `host:port`, no `$variable`) that matches
    no defined `upstream`. Scheme/host/variable targets are never flagged.
- **Inheritance (server_tokens, security headers).** Checks are
  **http→server inheritance-aware**: a directive set at `http` level counts as
  present for every vhost. A `location`-level `add_header` *replaces* inherited
  headers — v1 notes this as a known limitation rather than modelling full
  per-location effective headers.
- **Security-header set (fixed starter list):** `Strict-Transport-Security`
  (HSTS), `X-Content-Type-Options`, `X-Frame-Options`. CSP/Referrer-Policy are
  info-level suggestions. The vague "large `client_max_body_size`" heuristic is
  **dropped from v1** (too noisy to define well).
- **No "unknown directive" finding in v1.** Directives we don't model are
  parsed and ignored, so nginx **Plus** directives are never flagged — the Plus
  list is needed only to assert tests don't flag them.
- **`nginx -T` preamble** (`nginx: the configuration file … test is
  successful`, etc.) is tolerated and skipped. For a single-file/pasted input
  with no `# configuration file` markers, the source is shown as "pasted
  config" with line numbers from the pasted text.
- **Unknown server type.** `AnalyzeWebServer(text, server)` accepts `"nginx"`
  (and `""` → nginx); `apache`/`caddy` return a clear "not yet supported"
  error. The CLI defaults to nginx.

## Architecture (Meridian conventions)

- A pure engine package (e.g. `internal/webconf`) with an **nginx parser**:
  the brace/`;` grammar (directives, nested `http`/`server`/`location`/
  `upstream`/`stream` contexts, quoted strings, comments), consuming the
  `# configuration file` markers for source attribution. It builds a
  structured `Analysis` (servers, listeners, locations, upstreams, TLS,
  findings, counts). No I/O, no Wails — unit-tested against crafted configs.
- A thin `internal/app` method `AnalyzeWebServer(configText, server string)`
  → `Analysis`, plus a CLI adapter (CLI + JSON per §4.9).
- A `WebServerAnalyzer.svelte` tool in the Config insight group (server-type
  aware; nginx in v1), reusing the firewall tool's visual patterns (load/paste,
  tiles, cards, findings list).

## Testing posture

Built and tested against crafted nginx (`nginx -T`-style) fixtures covering:
multiple server blocks across several `# configuration file` sections,
http/https/http2 listeners, `proxy_pass` to defined and **undefined**
upstreams, duplicate `server_name`, `listen ssl` without a cert, deprecated
`ssl_protocols`, `autoindex on`, `server_tokens`, nginx **Plus** directives
(must not be flagged unknown), and a single-file config with unresolved
`include`. Parser, model extraction, routing map, and each finding get direct
tests. No network, no running nginx.

## Left to the builder

- The nginx parser implementation (brace/`;` tokenizer + context stack) and
  the `Analysis` type shape.
- The exact finding catalogue and messages (starter set above); how severity
  maps to the UI.
- How `# configuration file` markers map a node to `file:line`.
- The nginx Plus directive list to recognise (so they aren't "unknown").
- Diagram vs table-only for the routing map (a table is enough for v1; a
  server→upstream diagram is an optional follow-on like the firewall one).
