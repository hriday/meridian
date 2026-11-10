# Meridian — HTTP(S) endpoint check

**Date:** 2026-11-10
**Status:** design, ready to plan
**Foundation:** `docs/superpowers/specs/2026-11-08-meridian-design.md` §4.6 (the
~60 SolarWinds-style tools — HTTP/HTTPS monitors are part of that family).

## Intent

Point Meridian at a URL and get the operator's-eye view of what the endpoint
actually does: the final status, the redirect chain it went through, the key
response headers (server, content type/length), whether it's HTTP/1.1 or HTTP/2,
and where the time went — DNS, TCP connect, TLS handshake, time-to-first-byte,
and total. The "is this endpoint healthy and how does it respond?" check you'd
otherwise do with `curl -v` / `curl -w`.

## Why it's safe and in scope

A single user-initiated GET to a user-specified URL, with the project's generic
`User-Agent` and no user identity. It reads the response metadata (status,
headers, a little of the body) and reports it. The SolarWinds tool set explicitly
includes HTTP/HTTPS monitors; this is that, as a one-shot check.

## What it reports

- **Final URL + status** (code + text, e.g. `200 OK`), and the **protocol**
  (`HTTP/1.1` or `HTTP/2`).
- **Redirect chain**: each hop as `{status, url}` (e.g. `301 http://…` →
  `200 https://…`), capped so a redirect loop can't run forever.
- **Headers**: Server, Content-Type, Content-Length called out, plus the full
  final response header set.
- **TLS**: for an `https://` endpoint, the negotiated TLS version (empty for
  plain HTTP).
- **Timing** (milliseconds, via `net/http/httptrace`): DNS, connect, TLS
  handshake, time-to-first-byte, and total.

## Behaviour

- Redirects are followed manually (the client does not auto-follow) so each hop
  is captured with its status and target, up to a cap (then reported as "too
  many redirects").
- A non-2xx final status is reported plainly — it is a result, not a tool error.
- A connect/DNS/TLS/timeout failure is an error with a clear message.
- The body is read only minimally (enough to measure TTFB and close cleanly);
  this is a metadata check, not a downloader.
- A bare host (`example.com`) defaults to `https://`.

## Scope

**In scope:** one-shot GET with redirect-chain + header + timing capture;
engine (`internal/httpcheck`), CLI (`http`), Wails method, Workbench tool under
"Internet".

**Non-goals (future / out):** content/body assertions or keyword matching;
POST/other methods; auth; repeated polling / uptime monitoring; downloading the
body. *My calls.*

## UX (Workbench tool, Nielsen-compliant)

- A URL input and a Check button, with a one-line note on what it measures.
- A status banner (green 2xx, amber 3xx/4xx, red 5xx/error) with the final URL.
- A redirect-chain list when there was one; a timing row (DNS/connect/TLS/TTFB/
  total) ideally as a small stacked bar; a headers card (Server/Content-Type/
  Content-Length highlighted, full headers below).
- The screen explains itself.

## Testing posture

- `Check` is tested against `httptest` servers (offline): a 200 with headers
  (asserts status/text/server/content-type and non-negative timings); a redirect
  chain (301 → 200, asserts the hops and the final URL); a non-2xx final (still a
  result, no error); a redirect loop (capped, reported as too many); an invalid
  URL and a dial failure (errors). TLS version is asserted via
  `httptest.NewTLSServer` with its trusting client.
