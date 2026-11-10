# Network Interfaces Viewer — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** List the host's network interfaces (name, MAC, MTU, flags, IPv4/IPv6 addresses), across engine, CLI, and the Workbench.

**Architecture:** New engine `internal/netif`: `List` assembles a view model from `net.Interfaces()` + per-interface addresses, both behind package-var seams for offline tests. Thin adapters: a Wails method, an `interfaces` CLI command, and a `Interfaces.svelte` tool under "Local / environment".

**Tech Stack:** Go 1.23 (`net`), Wails v2 + Svelte. Tests use injected fakes + `stretchr/testify`.

**Spec:** `docs/superpowers/specs/2026-11-10-meridian-netif-design.md`

## Global Constraints

- Local, read-only; no network connection, nothing sent anywhere.
- Addresses split into IPv4 / IPv6, in CIDR form; MAC formatted `aa:bb:cc:dd:ee:ff`, empty when absent.
- One interface whose address lookup fails is still listed (empty addresses); never fail the whole list for it.
- Stable order (by index). Seams for the interface list and per-interface addresses so tests stay offline.
- Commit attribution: `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`. PR body ends with the Generated-with line.

## Review Focus

- **Address-lookup error on one interface** → that interface still listed, addresses empty, no overall error. (Task 1 test with a seam that errors for one iface.)
- **IPv4/IPv6 split** — a `*net.IPNet` with a v4 address goes to IPv4, a v6 to IPv6, each as CIDR. (Task 1 test with mixed addrs.)
- **Empty MAC** — a loopback-style interface (nil HardwareAddr) yields `MAC == ""`, not a panic or `"<nil>"`. (Task 1 test.)
- **Flags to plain words** — Up/Loopback/Broadcast/PointToPoint/Multicast map to the expected labels; a down interface has no "up". (Task 1 test.)
- **Non-IPNet addr** — an address that isn't `*net.IPNet` (rare) is skipped without panic. (Task 1 test.)

---

### Task 1: Engine — `List`

**Files:**
- Create: `internal/netif/netif.go`
- Test: `internal/netif/netif_test.go`

**Interfaces:**
- Produces:
  - `type Iface struct { Index int; Name string; MAC string; MTU int; Up bool; Loopback bool; Flags []string; IPv4 []string; IPv6 []string }`
  - `var listInterfaces = net.Interfaces` (seam)
  - `var interfaceAddrs = func(i net.Interface) ([]net.Addr, error) { return i.Addrs() }` (seam)
  - `func List() ([]Iface, error)` — calls `listInterfaces()`; for each, builds `Iface` (MAC from `HardwareAddr.String()` → `""` when empty; `Flags` from the net.Flags bits; Up/Loopback booleans; addresses via `interfaceAddrs`, split v4/v6 by `ip.To4() != nil`, formatted as CIDR). An address-lookup error on one interface leaves its addresses empty and does not fail `List`. Sorted by Index.

- [ ] **Step 1: Write failing tests** `internal/netif/netif_test.go` (`package netif`):
  - a helper to set `listInterfaces` / `interfaceAddrs` to fakes (restore via `t.Cleanup`).
  - `TestListFormatsInterface`: one iface (Index 2, Name "en0", MAC `aa:bb:cc:dd:ee:ff`, MTU 1500, flags Up|Broadcast|Multicast) with addrs `192.168.1.20/24` and `fe80::1/64` → MAC string, MTU, Up true, Loopback false, Flags contains "up"/"broadcast"/"multicast", IPv4 == ["192.168.1.20/24"], IPv6 == ["fe80::1/64"].
  - `TestListEmptyMAC`: a loopback iface (nil HardwareAddr, flags Up|Loopback) → `MAC == ""`, Loopback true, Flags contains "loopback".
  - `TestListAddrErrorStillListed`: `interfaceAddrs` returns an error for the iface → it's still in the result with empty IPv4/IPv6 and `List` returns nil error.
  - `TestListSkipsNonIPNet`: an addr that is a `*net.IPAddr` (not IPNet) is skipped without panic.
  - `TestListSorted`: two ifaces returned out of index order → result sorted by Index.
- [ ] **Step 2: Run, verify fail** (`go test ./internal/netif/ -v`).
- [ ] **Step 3: Implement** `netif.go` (flag mapping helper; v4/v6 split via `net.IP.To4()`; CIDR via the `*net.IPNet` String or `ip.String()+"/"+masklen`).
- [ ] **Step 4: Run, verify pass**, then also run `List()` against the real host once (a tiny `TestListRealHostSmoke` that just asserts no error and ≥1 interface — loopback always exists).
- [ ] **Step 5: Commit** (`feat(netif): list local interfaces (MAC/MTU/flags/addresses)`).

### Task 2: Wails method + CLI `interfaces`

**Files:**
- Create: `internal/app/netif_methods.go`, `internal/app/netif_methods_test.go`
- Modify: `internal/cli/cli.go`
- Create: `internal/cli/printnetif_test.go` (`package cli`, internal)

**Interfaces:**
- Produces:
  - `func (a *App) Interfaces() ([]netif.Iface, error)` — returns `netif.List()`.
  - CLI: `case "interfaces": return runInterfaces(...)`; `runInterfaces` (`--json`, no positional); `printInterfaces(w, []netif.Iface)` printing a block per interface (name, state, MAC, MTU, flags, addresses).

- [ ] **Step 1 (app): Write failing test** `netif_methods_test.go`: `New().Interfaces()` returns no error and ≥1 interface (real host loopback). (The engine seams are already unit-tested; this checks the method wiring.)
- [ ] **Step 2: Run, verify fail.**
- [ ] **Step 3: Implement** `Interfaces`.
- [ ] **Step 4: Run, verify pass.**
- [ ] **Step 5 (cli): Write failing tests** `printnetif_test.go`:
  - `TestPrintInterfaces`: a slice with one up iface (MAC, MTU, an IPv4) → output contains the name, MAC, MTU, "up", and the address.
  - `TestRunInterfacesJSON`: `runInterfaces(["--json"], …)` → exit 0 and output is JSON (starts with `[`). (Uses the real host; assert exit 0.)
- [ ] **Step 6: Run, verify fail.**
- [ ] **Step 7: Implement** the `interfaces` case, `runInterfaces`, `printInterfaces`, usage line; import `netif`.
- [ ] **Step 8: Run** `go test ./internal/app/ ./internal/cli/`, verify pass.
- [ ] **Step 9: Commit** (`feat(cli,app): interfaces command and Wails method`).

### Task 3: Workbench tool `Interfaces.svelte`

**Files:**
- Create: `frontend/src/lib/tools/Interfaces.svelte`
- Modify: `frontend/src/lib/Workbench.svelte` (import, tools entry under "Local / environment", route)
- Regenerate: bindings (`App.js`/`App.d.ts`/`models.ts` — `Interfaces` + `netif.Iface`).

- [ ] **Step 1: Regenerate bindings** (`$(go env GOPATH)/bin/wails generate module`; `git checkout -- frontend/wailsjs/runtime`); verify `Interfaces` in `App.d.ts` and a `netif` namespace in `models.ts`.
- [ ] **Step 2: Write `Interfaces.svelte`** — loads the list on mount (and a Refresh button); a one-line lede; one card per interface with name + up/down, MAC + MTU, flag chips, and IPv4/IPv6 addresses; down/loopback muted. Reuse the established tool visual language and theme tokens.
- [ ] **Step 3: Wire into `Workbench.svelte`**: import, add `{ id: "ifaces", name: "Interfaces", group: "Local / environment", ready: true }` (distinct id) after DNS, add the `{:else if active === "ifaces"}<Interfaces />` route.
- [ ] **Step 4: Build** `cd frontend && npm run build`; verify green.
- [ ] **Step 5: Manual check** via the running app (localhost:34115): confirm the interface cards render with the host's real NICs, addresses, and flags.
- [ ] **Step 6: Commit** (`feat(app,ui): interfaces tool in the Workbench`).

---

## Self-Review notes

- Spec coverage: list local interfaces → Tasks 1–3; formatting/split/error-tolerance → Task 1; CLI/app → Task 2; UI → Task 3.
- Review Focus → each maps to a Task 1 test (addr-error, v4/v6 split, empty MAC, flag words, non-IPNet).
- Proportion: signatures + test names; bodies left to the implementer. Pure stdlib, no external deps.
