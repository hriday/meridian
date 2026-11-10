# Meridian — Raw Whois (port 43)

**Date:** 2026-11-10
**Status:** design, ready to plan
**Foundation:** `docs/superpowers/specs/2026-11-08-meridian-design.md` §4.6
("DNS & Whois").

## Intent

RDAP (which the Lookup entity pages already use) is the structured successor to
Whois, but the classic **port-43 Whois** still carries the authoritative
free-text registry record — and covers objects and TLDs where RDAP is thin or
absent. This tool queries Whois for a domain, IP, or ASN and shows the raw
record, having discovered the right server to ask. It complements the
RDAP-backed registration data, not duplicates it.

## Why it's safe and in scope

Read-only, user-initiated TCP queries to public Whois servers (the IANA root and
the authoritative registry/RIR), carrying only the object being looked up — no
user identity. It is exactly what the `whois` command does. §4.6 lists "DNS &
Whois" explicitly.

## How it works (server discovery)

Whois has no single server, so the tool discovers the authoritative one the way
`whois` does:

1. Ask **whois.iana.org** for the object. IANA's response carries a referral
   line — `refer:` (IPs/ASNs → the RIR) or `whois:` (a TLD → its registry) —
   naming the authoritative server.
2. Query that server for the object → the authoritative record.
3. If that record names a further server (a domain's `Registrar WHOIS Server:`
   or an RIR's `ReferralServer:`), follow it **once** more for the most specific
   record.

Each step is a plain port-43 exchange: connect, send the object + CRLF, read the
response, close. The chain of servers consulted is reported so the result is
transparent.

## Behaviour

- Input: a domain, IP, or ASN (as typed). The object is sent verbatim; the TLD is
  used for the IANA lookup when it's a domain.
- Output: the raw Whois text of the most authoritative server reached, the server
  that produced it, and the chain of servers consulted.
- A server that returns no referral is terminal — its response is the record.
- Referrals are followed at most twice total (IANA → authoritative → registrar)
  so a referral loop can't run away.
- A connect/read failure, or no server discoverable, is a clear error.

## Scope

**In scope:** server discovery via IANA referral, the raw record, and the server
chain; engine (`internal/whois`), CLI (`whois`), Wails method, Workbench tool
under "Internet".

**Non-goals (future / out):** parsing the free-text record into fields (RDAP on
the entity pages is the structured view; raw text is the point here); rate-limit
handling beyond surfacing the server's own message; privacy/abuse redaction.
*My calls.*

## UX (Workbench tool, Nielsen-compliant)

- An input (domain/IP/ASN) and a Look up button, with a one-line note that this
  is the raw registry record and that Lookup's Registry tab has the structured
  view.
- The server chain shown as a small breadcrumb (`whois.iana.org →
  whois.verisign-grs.com → …`), then the raw record in a monospace block.
- A clear error on failure. The screen explains itself.

## Testing posture

- Server discovery and the referral chain are tested offline behind a **dial
  seam**: local TCP listeners stand in for IANA, the registry, and the registrar,
  each returning a canned response (with/without a referral line); the engine
  must walk the chain, stop at a terminal server, cap the hops, and report the
  chain. Referral parsing (`refer:`/`whois:`/`Registrar WHOIS Server:`/
  `ReferralServer:`, case-insensitive) is unit-tested directly. A dial failure is
  asserted to error.
