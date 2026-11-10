# Meridian — DNS / email posture audit

**Date:** 2026-11-10
**Status:** design, ready to plan
**Foundation:** `docs/superpowers/specs/2026-11-08-meridian-design.md` §4.6 ("DNS audit"),
§4.1 lookups.

## Intent

An engineer wants a one-glance answer to "is this domain's DNS and email posture
sound?" — the kind of check you'd otherwise run across a handful of `dig`
commands and remember how to read. Meridian should resolve the domain's core
records and grade each in plain language: who serves the zone (NS), whether it
can receive mail (MX), whether senders are constrained (SPF), whether spoofing
is policed (DMARC), and whether answers are DNSSEC-validated — then give an
overall posture line. This is the §4.6 "DNS audit" tool, built as its own slice
(engine + CLI + Workbench tool), and the diagnostic companion to the existing
DNS-query and blocklist tools.

## Why it's safe and in scope

Read-only, user-initiated DNS lookups against the user's resolver. The queries
carry only the domain being audited — no user identity. It is the same handful
of lookups a mail admin runs by hand; it changes nothing.

## Checks (this slice)

Each check resolves records via the user's resolver (default: system resolver,
as the DNS and blocklist tools do) and produces a status + a plain-language note
+ the raw records it looked at.

1. **Nameservers (NS).** The domain's NS records at the apex. `ok` when present;
   `missing` (a problem) when none; `error` on SERVFAIL/timeout.
2. **Mail (MX).** The MX records. `ok` when present; a single `.`/null MX
   (`0 .`) is called out as "explicitly accepts no mail" (`warn`, intentional);
   no MX is `warn` ("this domain can't receive mail"). Errors reported honestly.
3. **SPF.** The apex TXT records starting `v=spf1`. Parse the terminal `all`
   mechanism: `-all` = strict (`ok`), `~all` = soft-fail (`ok`, noted as soft),
   `?all`/no `all` = neutral (`warn`), `+all` = passes anything (`warn`,
   dangerous). More than one `v=spf1` record is invalid per RFC 7208 (`warn`).
   No SPF record = `missing`.
4. **DMARC.** The TXT at `_dmarc.<domain>` starting `v=DMARC1`. Parse `p=`:
   `reject` = strong (`ok`), `quarantine` = medium (`ok`, noted), `none` =
   monitoring only (`warn`). No DMARC record = `missing`.
5. **DNSSEC.** Whether the resolver returned answers as authenticated (the AD
   flag) for the apex. `ok` when authenticated; otherwise `warn` with the honest
   caveat that this reflects *the resolver's* validation — a resolver that
   doesn't validate, or a zone that isn't signed, both show as not-validated, so
   this is a signal, not a proof. (A full DS/DNSKEY chain walk is a later slice.)

**Non-goals (future slices):** DKIM (needs a selector, not generically
discoverable); CAA/TLSA; a DS/DNSKEY chain validator; per-mechanism SPF
expansion / include flattening; checking MX host reachability (that's the TLS /
port tools). *My calls, from a bounded-first-slice reading of §4.6.*

## Overall posture

A short plain-language summary: counts of ok/warn/missing across the five checks
and the single most important thing to fix first (e.g. "No DMARC record — add
one at `_dmarc.<domain>`"). The per-check detail carries the rest.

## UX (Workbench tool, Nielsen-compliant)

- One input (domain), a resolver field pre-filled with the system resolver, an
  Audit button, and a one-line note on what the audit covers — the screen
  explains itself.
- An overall posture banner (green when nothing is missing/warn, amber
  otherwise — this is advisory, never a red "alarm": a weak posture is the
  user's own domain to improve).
- A per-check list: check name, status pill (OK / Warn / Missing / Error), the
  plain-language note, and the raw record(s) in mono.
- Consistent with the TLS and blocklist tools' visual language.

## Testing posture

- Pure parsers (SPF `all`-qualifier classification incl. multiple-record and
  `+all`; DMARC `p=` classification incl. missing tag) are table-tested directly.
- The networked `Audit` is tested against a local `miekg/dns` server (the pattern
  used by `internal/dnsx` and `internal/dnsbl`) that serves chosen NS/MX/TXT
  answers and sets/omits the AD flag, so tests are fully offline and
  deterministic. A SERVFAIL path asserts a check reads as `error`, not `ok`.
