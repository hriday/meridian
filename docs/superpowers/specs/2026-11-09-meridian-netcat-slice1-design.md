# Meridian — Security suite slice 2: Netcat (design)

**Status:** approved by owner (2026-11-09) with the two decisions below.

## Intent and why
A raw connection tool — the `nc` in the toolbox. Open a TCP/UDP connection to a
host:port and exchange bytes, or bind a local port and receive them. For the
hundred little network tasks where you need to *speak the protocol by hand*:
poke a service, test a firewall rule, confirm a listener, probe a TLS endpoint.
Data streams both ways, shown as text or hex.

## Decisions (confirmed)
1. **Both modes:** **Connect** (client) and **Listen** (bind a local port),
   over **TCP and UDP**.
2. **TLS connect included:** a "use TLS" option wraps a TCP connection in TLS so
   you can probe HTTPS/SMTPS/IMAPS-style services (type a request, read the
   response; surface the peer cert's subject/issuer/expiry).

## Goals
- **Connect:** dial `host:port` over TCP or UDP; optional TLS (TCP only). Send
  typed/pasted data (with a choice of line ending: none / LF / CRLF); receive
  streamed bytes live. For TLS, show the negotiated cert summary (subject,
  issuer, notAfter) and whether verification passed (allow connecting to a
  self-signed/invalid cert with a clear "unverified" flag — probing is the point).
- **Listen:** bind a local `host:port` (default all-interfaces or localhost) over
  TCP or UDP; show incoming connection(s)/datagrams and the bytes received; allow
  sending a reply back. One active peer at a time is enough for slice 1.
- **Both directions stream live** (Wails events), with a **text / hex** view
  toggle and a byte counter. A clear connection status (connecting / connected /
  closed / error with the peer address).
- **CLI** counterpart: `meridian-cli nc <host> <port> [--udp] [--tls] [--listen]`
  piping stdin ↔ connection (scriptable, like real nc).
- UI follows the approved "listener/terminal" template: a target row
  (host/port/TCP|UDP/TLS), a connect/listen button, a transcript pane (text|hex),
  and a send box with the line-ending choice. Plain-language status.

## Non-goals / deferred
- Port scanning / sweeps (that's the separate Local-network tranche), proxying,
  port-forwarding, `-e`/exec (deliberately **never** — no command execution on
  connect), SOCKS, multiple simultaneous peers, file transfer UI, mTLS client
  certs, DTLS. These are later or out of scope.
- No listening as a public service for others; binds are for the operator's own
  diagnostics on their own machine.

## Constraints (global)
- Go directive **1.23**; stdlib only (`net`, `crypto/tls`, `crypto/x509`,
  `bufio`, `encoding/hex`). No new deps.
- Additive only — new `internal/netcat` package + CLI/app additions + a new
  frontend tool; nothing existing is modified beyond wiring (`Workbench.svelte`,
  the CLI command switch, the App struct).
- The read loop and dialing go through the context so a connection can be
  cancelled/closed cleanly; no goroutine leaks on close.
- **Never** execute a command on connect or data. TLS "unverified" connections
  are allowed but clearly labelled; the default verifies.
- Outbound traffic is whatever the user types — Meridian adds no identity.

## What the builder decides
- Package layout within `internal/netcat`, the exact `Session`/`Server` API
  (suggest a `Session` wrapping a `net.Conn` with Send/registered-recv-callback/
  Close, and a Listen that yields a `Session` on the first accepted peer), the
  hexdump format, and the Svelte layout within the template + plain-language
  standard.
- How the App manages the single active session and streams events
  (`netcat:recv`, `netcat:status`) — builder's call, mirroring the trace tool's
  streaming pattern.
