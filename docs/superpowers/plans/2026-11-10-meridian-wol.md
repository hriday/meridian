# Wake-on-LAN — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build and broadcast a Wake-on-LAN magic packet for a MAC address, with a plain-language result, across engine, CLI, and the Workbench.

**Architecture:** New pure-ish engine `internal/wol`: `MagicPacket` (pure) builds/validates the 102-byte packet; `Send` dials a UDP broadcast with `SO_BROADCAST` set portably. Thin adapters: a Wails method, a `wol` CLI command, and a `WakeOnLan.svelte` tool under "Local network". Mirrors the established slice structure.

**Tech Stack:** Go 1.23 (`net`, `syscall` for SO_BROADCAST via `net.Dialer.Control`), Wails v2 + Svelte. Tests use a local UDP listener and `stretchr/testify`.

**Spec:** `docs/superpowers/specs/2026-11-10-meridian-wol-design.md`

## Global Constraints

- EUI-48 (6-byte) MACs only, via `net.ParseMAC`; accept colon/hyphen/dot/bare forms.
- Magic packet = 6×`0xFF` + MAC×16 = 102 bytes.
- Default target `255.255.255.255:9`; broadcast address and port configurable.
- `SO_BROADCAST` set portably: `setBroadcast(c syscall.RawConn) error` in two build-tagged files (`!windows` / `windows`), using stdlib `syscall` only (no new deps).
- "Sent" ≠ "woke" — the UI/CLI must not claim the machine powered on.
- Commit attribution: `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`. PR body ends with the Generated-with line.

## Review Focus

- **Malformed / wrong-length MAC.** An 8-byte (EUI-64) or garbage MAC must be rejected with a clear error, not sent. (Task 1 `MagicPacket` test.)
- **Packet byte-exactness.** The MAC must appear at offsets 6, 12, …, 96 (all 16 repetitions), not just the first — an off-by-one in the loop would wake nothing. (Task 1 test checks every repetition.)
- **Accepted MAC spellings parse to the same packet.** `aa:bb:…`, `aa-bb-…`, `aabb.ccdd.eeff`, `aabbccddeeff` must all yield the identical packet. (Task 1 test.)
- **Send delivers the exact bytes.** The datagram received must equal `MagicPacket(mac)`. (Task 2 test against a local listener.)
- **Send surfaces a dial/write error** rather than silently succeeding. (Task 2 test with an unroutable/invalid target.)

---

### Task 1: Magic packet + MAC parsing (pure) and the SO_BROADCAST helpers

**Files:**
- Create: `internal/wol/wol.go`, `internal/wol/sockopt_unix.go` (`//go:build !windows`), `internal/wol/sockopt_windows.go` (`//go:build windows`)
- Test: `internal/wol/wol_test.go`

**Interfaces:**
- Produces:
  - `func MagicPacket(mac string) ([]byte, error)` — parses `mac` with `net.ParseMAC`, requires `len==6`, returns the 102-byte packet; error on bad/wrong-length MAC.
  - `func setBroadcast(c syscall.RawConn) error` — sets `SO_BROADCAST` (per-OS file); unexported.

- [ ] **Step 1: Write failing tests** `internal/wol/wol_test.go` (`package wol`):
  - `TestMagicPacketStructure`: `MagicPacket("01:02:03:04:05:06")` → len 102; `pkt[0:6]` all `0xFF`; for `i` in 0..15, `pkt[6+i*6 : 12+i*6] == {1,2,3,4,5,6}`.
  - `TestMagicPacketAcceptsSpellings`: `01:02:03:04:05:06`, `01-02-03-04-05-06`, `0102.0304.0506`, and (if `net.ParseMAC` rejects bare — verify) handle accordingly; assert the colon/hyphen/dot forms yield identical bytes. (For bare `010203040506`, add a normalization only if `net.ParseMAC` doesn't accept it — decide in Step 3 and test whatever the engine supports.)
  - `TestMagicPacketRejects`: `""`, `"nope"`, and an 8-byte `01:02:03:04:05:06:07:08` each return an error.
- [ ] **Step 2: Run, verify fail** (`go test ./internal/wol/ -run MagicPacket -v`).
- [ ] **Step 3: Implement** `MagicPacket` (ParseMAC; if it returns an 8- or 20-byte address, error "need a 6-byte MAC"; build the packet). Decide bare-hex support: if `net.ParseMAC("010203040506")` fails, add a hex fallback that inserts colons, and cover it with a test; otherwise rely on ParseMAC. Implement `setBroadcast` in the two build-tagged files: `c.Control(func(fd uintptr){ _ = syscall.SetsockoptInt(<fd>, syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1) })` — Unix uses `int(fd)`, Windows uses `syscall.Handle(fd)`.
- [ ] **Step 4: Run, verify pass.**
- [ ] **Step 5: Commit** (`feat(wol): magic packet builder + SO_BROADCAST helpers`).

### Task 2: Send over UDP broadcast

**Files:**
- Modify: `internal/wol/wol.go`
- Test: `internal/wol/send_test.go`

**Interfaces:**
- Consumes: Task 1 `MagicPacket`, `setBroadcast`.
- Produces: `func Send(ctx context.Context, mac, bcast string, port int) error` — builds the packet (bad MAC → error), dials `udp` to `net.JoinHostPort(bcast, port)` with a `net.Dialer{Control: ...setBroadcast...}`, writes the packet once, returns any dial/write error. Empty `bcast` → `255.255.255.255`; `port<=0` → 9.

- [ ] **Step 1: Write failing tests** `internal/wol/send_test.go`:
  - `TestSendDeliversMagicPacket`: start a UDP listener on `127.0.0.1:0`; `Send(ctx, "01:02:03:04:05:06", "127.0.0.1", <port>)`; read the datagram; assert it equals `MagicPacket(mac)`.
  - `TestSendRejectsBadMAC`: `Send(ctx, "nope", "127.0.0.1", 9)` returns an error (no packet sent).
  - `TestSendDefaults`: `bcast==""` and `port==0` resolve to 255.255.255.255:9 — assert via building the expected dial target (or that an empty bcast doesn't panic; keep this light since 255.255.255.255 may not be writable in CI — prefer asserting the defaulting logic through a small helper `target(bcast,port) string` that the test can call directly).
- [ ] **Step 2: Run, verify fail.**
- [ ] **Step 3: Implement** `Send` and the `target(bcast, port)` helper (default + `JoinHostPort`).
- [ ] **Step 4: Run, verify pass**, then full `go test ./internal/wol/`.
- [ ] **Step 5: Commit** (`feat(wol): send magic packet over UDP broadcast`).

### Task 3: Wails method + CLI `wol`

**Files:**
- Create: `internal/app/wol_methods.go`, `internal/app/wol_methods_test.go`
- Modify: `internal/cli/cli.go`
- Create: `internal/cli/printwol_test.go` (`package cli`, internal)

**Interfaces:**
- Produces:
  - `func (a *App) WakeOnLAN(mac, bcast string, port int) (string, error)` — calls `wol.Send`; on success returns a plain confirmation string ("Magic packet sent to <mac> via <target>"); maps errors through.
  - CLI: `case "wol": return runWOL(...)`; `runWOL` (flags `--bcast` default `255.255.255.255`, `--port` default 9, one positional MAC); on success prints the confirmation and a one-line caveat ("'sent' isn't 'woke' — WoL must be enabled and broadcasts don't cross subnets").

- [ ] **Step 1 (app): Write failing test** `wol_methods_test.go`: start a local UDP listener; `New().WakeOnLAN("01:02:03:04:05:06","127.0.0.1",<port>)` returns a non-empty confirmation and no error, and the listener receives 102 bytes; a bad MAC returns an error.
- [ ] **Step 2: Run, verify fail.**
- [ ] **Step 3: Implement** `WakeOnLAN`.
- [ ] **Step 4: Run, verify pass.**
- [ ] **Step 5 (cli): Write failing tests** `printwol_test.go`:
  - `TestRunWOLMissingArg`: `runWOL(nil,…)` → exit 2.
  - `TestRunWOLBadMAC`: `runWOL(["nope"],…)` → exit 1 (send/parse error path), nothing panics.
- [ ] **Step 6: Run, verify fail.**
- [ ] **Step 7: Implement** the `wol` case, `runWOL`, usage line; import `wol`.
- [ ] **Step 8: Run** `go test ./internal/app/ ./internal/cli/`, verify pass.
- [ ] **Step 9: Commit** (`feat(cli,app): wol command and Wails method`).

### Task 4: Workbench tool `WakeOnLan.svelte`

**Files:**
- Create: `frontend/src/lib/tools/WakeOnLan.svelte`
- Modify: `frontend/src/lib/Workbench.svelte` (import, tools entry under "Local network", route)
- Regenerate: bindings (`App.js`/`App.d.ts` WakeOnLAN only; no new model type).

- [ ] **Step 1: Regenerate bindings** (`$(go env GOPATH)/bin/wails generate module`; `git checkout -- frontend/wailsjs/runtime`); verify `WakeOnLAN` in `App.d.ts`.
- [ ] **Step 2: Write `WakeOnLan.svelte`** — MAC input, optional broadcast (default `255.255.255.255`) and port (default 9), a Wake button; a self-explaining lede with the two "may not wake" caveats; on success a plain confirmation banner, on error a clear message. Reuse the established tool visual language and theme tokens.
- [ ] **Step 3: Wire into `Workbench.svelte`**: import, add `{ id: "wol", name: "Wake-on-LAN", group: "Local network", ready: true }` after Port scanner, add the `{:else if active === "wol"}<WakeOnLan />` route.
- [ ] **Step 4: Build** `cd frontend && npm run build`; verify green.
- [ ] **Step 5: Manual check** via the running app (localhost:34115): enter a MAC, click Wake, confirm the confirmation + caveats read clearly (a local UDP broadcast won't error even with nothing listening).
- [ ] **Step 6: Commit** (`feat(app,ui): Wake-on-LAN tool in the Workbench`).

---

## Self-Review notes

- Spec coverage: §4.6 Wake-on-LAN → Tasks 1–4; packet + MAC parsing → Task 1; broadcast send → Task 2; resolver/caveats UX → Task 3/4.
- Review Focus → each maps to a named test (bad MAC, every repetition offset, spellings, exact delivery, send error).
- Proportion: signatures + test names; bodies left to the implementer. The only platform-specific code is the two tiny `setBroadcast` files.
