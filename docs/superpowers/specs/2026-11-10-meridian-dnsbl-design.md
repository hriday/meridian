# Meridian — IP blocklist (DNSBL) reputation check

**Date:** 2026-11-10
**Status:** design, ready to plan
**Foundation:** `docs/superpowers/specs/2026-11-08-meridian-design.md` §4.1 (per-entity
action "check blocklists"), §4.6 tool set.

## Intent

An engineer wants to know whether an IP address — their own mail server, a
customer's host, a suspicious source — is listed on public DNS-based blocklists
(DNSBLs / RBLs). Meridian should answer that in plain language from the keyless,
public DNSBL system: query a curated set of blocklist zones, say whether the
address is listed and on which lists, show the raw return code and any reason
text the list publishes, and — crucially — tell the truth when a query was
*refused* rather than silently reporting "clean".

This is the §4.1 "check blocklists" per-entity action, built as its own slice
(engine + CLI + Workbench tool), pairing with the TLS cert inspector already
shipped.

## Why this is a safe, defensive tool

A DNSBL check is read-only and user-initiated. It makes outbound **DNS** queries
to public blocklist zones, reversing the queried IP into the zone name (e.g.
checking `1.2.3.4` against `zen.spamhaus.org` queries `4.3.2.1.zen.spamhaus.org`).
The query carries **no user identity** — only the IP being checked and the
blocklist zone. It is the same lookup every mail server does on every inbound
connection. It does not list, submit, or report anything to anyone.

## How DNSBLs work (the facts the engine must encode)

- **Query shape.** For IPv4 `a.b.c.d` against zone `Z`, query the A record of
  `d.c.b.a.Z`. For IPv6, expand to the full 32 nibbles, reverse, dot-separate,
  append `Z`. A TXT record at the same name carries a human-readable reason /
  removal URL when the list publishes one.
- **Listed.** An A answer in `127.0.0.0/8` means listed. The last octet encodes
  which sub-list (Spamhaus: `127.0.0.2`=SBL, `.3`=CSS, `.4–.7`=XBL/CBL,
  `.9`=DROP, `.10/.11`=PBL). Different lists use different codes, so the engine
  reports the raw code(s) rather than hard-coding every list's meaning.
- **Not listed.** `NXDOMAIN` (or NOERROR with no answer) means the address is
  not on that list — this is the clean result.
- **Error / refused (the subtlety that must not be mis-read).** Spamhaus (and by
  convention other lists) return answers in **`127.255.255.0/24`** to signal an
  *error*, not a listing:
  - `127.255.255.252` — typo in the DNSBL name,
  - `127.255.255.254` — query came via a **public/open resolver** (e.g. Google
    `8.8.8.8`, Cloudflare `1.1.1.1`) or from an IP with generic reverse DNS;
    blocked under Spamhaus's Fair Use Policy,
  - `127.255.255.255` — too many queries (rate-limited).
  These **must not** be interpreted as "listed". They mean the check could not be
  performed; the engine reports them as a distinct `blocked` status with a
  plain-language fix (use your own resolver, or reduce query volume).
  Sources: Spamhaus "Check your return codes now" (spamhaus.org/news/article/807);
  Spamhaus DNSBL usage FAQ (check.spamhaus.org).
- **Resolver matters.** Because the big public resolvers are blocked, the default
  resolver for this tool must be the **user's own system resolver**, exactly as
  the existing DNS tool defaults it (`DNSSystem()` → first value). The UI must
  say, in plain words, that using a big public resolver will get queries refused.

## Scope

**In scope (this slice):**
- IPv4 and IPv6 reputation lookups against a curated, **user-editable** set of
  public IP DNSBL zones, each with a human label.
- Per-zone result: `listed` / `clean` / `blocked` / `error`, the raw return
  code(s), and the TXT reason when present.
- An aggregate verdict and plain-language summary: "Listed on N of M lists" /
  "Not listed on any of the M lists checked" / incomplete when some zones errored.
- Resolver selectable; defaults to the system resolver with the public-resolver
  caveat shown.
- Engine (`internal/dnsbl`), CLI command (`blocklist`), Wails method, Workbench
  tool, all following the established seams and the one-page-per-tool,
  explain-itself UI.

**Non-goals (future slices):**
- Domain/URI blocklists (DBL / URIBL / SURBL) — a separate "domain reputation"
  action. This slice is IP-only.
- Delisting/removal automation, or submitting reports to any list.
- Bulk/sweep checking of many IPs at once (the per-entity action is one IP; a
  sweep belongs with the ping-sweep family).
- Hard-coding every list's per-code sub-list meaning. Raw code + TXT is enough
  and stays correct as lists evolve.

## Default zones

A small, well-known starting set; the user can add or remove zones. Each carries
a label and an optional note (some lists require registering your resolver IP or
your own resolver):

- `zen.spamhaus.org` — Spamhaus ZEN (SBL+XBL+PBL combined)
- `bl.spamcop.net` — SpamCop
- `dnsbl.dronebl.org` — DroneBL
- `b.barracudacentral.org` — Barracuda (note: free registration of your resolver
  IP required; unregistered queries are refused)

The engine is agnostic to the list: a defunct or registration-gated zone simply
reports `error`/`blocked` honestly rather than a false "clean".

## Behaviour / verdict

- `listed` on ≥1 zone → verdict "Listed on N blocklist(s)", summary names them.
- all `clean` → "Not listed on any of the M lists checked".
- some zones `blocked`/`error` → the verdict still reflects listed/clean for the
  zones that answered, but the summary notes the check was **incomplete** and how
  many lists could not be reached, so the user never reads a partial pass as a
  full all-clear.
- An input that is not a valid IP is rejected with a clear message (this tool is
  IP-only; a hostname must be resolved first).

## UX (Workbench tool, Nielsen-compliant)

- One input (IP), a resolver field pre-filled with the system resolver, a Check
  button. A one-line note explains what a DNSBL check is and the
  public-resolver caveat — the screen explains itself.
- A verdict banner: green "Not listed", red "Listed on N", amber when the check
  was incomplete (some lists unreachable).
- A per-list table: list name, status pill (Listed / Clean / Blocked / Error),
  return code(s), and reason text when present, with the zone shown in mono.
- Consistent with the TLS tool's visual language (verdict banner + cards/table,
  amber = caution not alarm).

## Testing posture

- Pure functions (reversed-name building for v4/v6, return-code classification,
  verdict aggregation) are unit-tested directly with table cases, including the
  `127.255.255.254` "blocked" path and a mixed listed/clean/blocked set.
- The networked `Check` is tested against a **local `miekg/dns` server** (the
  pattern already used by `internal/dnsx`), which answers the reversed-name
  queries with chosen A/TXT records, so tests are fully offline and deterministic.
