# Meridian — TLS certificate inspector (design)

**Status:** builder-selected next slice (2026-11-10). Implements the "TLS cert"
per-entity action named in the foundation design §4.1, as a standalone tool first.

## Intent and why
Answer "what certificate is this host serving, and is it healthy?" — the everyday
question before a TLS outage or during a cert-rotation. Connect to a host, show
the presented certificate (and chain), and give a **plain-language verdict**
(valid / expired / self-signed / untrusted / hostname mismatch / expiring soon /
weak crypto), each signal captioned. Reuses Meridian's TLS dialing; no new deps.

## Goals
- Engine `internal/tlscheck`: connect to `host[:port]` (default **443**), retrieve
  the **leaf + presented chain even when untrusted or expired** (so a broken cert
  is still inspectable), and report:
  - Subject CN, Issuer CN, **SANs** (DNS + IP), serial;
  - validity window (NotBefore/NotAfter) and **days to expiry**;
  - public-key type + size (RSA/ECDSA/Ed25519), **signature algorithm**;
  - negotiated **TLS version** and cipher suite;
  - booleans: **SelfSigned**, **ChainTrusted** (verified against system roots +
    presented intermediates), **HostnameMatch** (requested host vs SANs);
  - a **Verdict** string + an ordered **Warnings** list (all issues found).
- **Verdict rules** (worst-first): Expired / Not yet valid / Untrusted chain /
  Self-signed / Hostname mismatch / Expiring soon (< 21 days) / Weak key
  (RSA < 2048) / Weak signature (SHA-1 or MD5) → else **Valid**.
- **CLI:** `meridian-cli tls <host[:port]>` (+ `--json`).
- **UI:** a **TLS** tool in the Workbench — input a host, show a result card with
  the verdict, the fields above, and the SAN list; each metric captioned.
- Degrade, don't crash: a connect/handshake failure returns a clear error; a
  reachable host with a bad cert returns a full report with a non-Valid verdict.

## Decisions (builder calls — no owner sign-off sought this slice)
- **Standalone tool + CLI + engine now; wiring it as a button on the IP/domain
  entity pages is a deferred follow-up** (keeps this slice bounded and matches the
  one-tool-per-Workbench-entry pattern of netcat/bgp/scan).
- **One network handshake**, `InsecureSkipVerify: true`, to always capture the
  presented certificates; trust is then computed locally via `x509` verification
  against system roots + the presented intermediates (deterministic, testable).
- Input accepts `host`, `host:port`, or a `scheme://host...` URL (scheme/path
  stripped). IDN/punycode left as-is.
- Expiry warning threshold **21 days** (builder call).

## Non-goals / deferred
- Entity-page action button (follow-up); OCSP/CRL revocation checks; CT-log
  lookups; cipher/protocol enumeration (testssl-style); STARTTLS for
  SMTP/IMAP/etc.; client-cert auth; editing/exporting the cert.

## Constraints (global)
- Go 1.23; stdlib `crypto/tls` + `crypto/x509` only; no new dependency.
- Testable fully offline: a `dialTLS` seam (package var) lets tests point at a
  local `tls.Listener`, and a `systemRoots` seam lets tests inject a test CA pool
  for the trusted-chain case. Generic behavior; no user identity on the wire.
- CLI + JSON for the tool (§4.9); source/verdict legible (Nielsen bar).

## What the builder decides
- Go type/field names, the exact verdict/warn ordering within the rules above,
  the seam shapes, and the Svelte card layout within the existing tool kit.
