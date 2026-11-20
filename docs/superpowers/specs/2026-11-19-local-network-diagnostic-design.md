# Meridian — Local Network Diagnostic (design)

*Date: 2026-11-19. Status: design, pending approval.*

## Intent & why

When a Meridian LAN tool can't reach its target, the raw signals are
*misleading*: "no route to host", "all print ports closed", "is this even a
printer?". None of them name the real cause. The motivating case: a printer
check that read the printer perfectly under `sudo` but showed every port closed
without it — the actual cause was **macOS Local Network privacy silently blocking
the app**, not anything wrong with the printer or the network. A network engineer
shouldn't have to guess between *client isolation*, *the OS blocking the app*,
*a failed DHCP lease*, and *one dead host* — those are **deterministically
distinguishable**, and Meridian is exactly the tool that should distinguish them.

This builds a **recipe**: a fixed, ordered sequence of deterministic probes whose
*pattern* of results reads off exactly one plain-language verdict, with the likely
cause and the concrete fix. No AI, no heuristics, no interpretation — a decision
tree. The headline can be as blunt as: *"Meridian can reach the internet but
cannot reach any other device on this local network — this is a hotspot / guest
Wi-Fi that isolates clients."*

## Goals

- A standalone **Local Network Diagnostic** tool (sidebar group "Local network")
  that runs the ordered recipe and renders: each step's live pass/fail/skip, then
  one named **verdict** with cause + fix.
- **Cross-platform**, all three OSes, including the per-OS "is the OS itself
  blocking Meridian?" step: macOS (Local Network privacy), Windows (Defender
  Firewall + Public/Private network profile), Linux (`ufw`/`firewalld`).
- Every LAN-reaching tool (printer, ping, port scan, sweep, discovery, ARP scan,
  SNMP) gains a **"Run Local Network Diagnostic"** affordance when it fails to
  reach its target — pre-seeded with that target.
- **CLI parity**: `meridian-cli netdiag [host]` with `--json`.
- **Honest verdicts**: a distinct "the local network is fine — that *specific*
  host isn't responding" result, so the tool blames the network only when the
  network is actually the problem. And an explicit **Inconclusive** when a probe
  itself errors ambiguously — never collapse uncertainty into a confident wrong
  verdict (the [[meridian-failed-lookup-not-verdict]] principle).

## Non-goals

- No historical retention or trending — point-in-time, run-once (re-runnable).
  (Per [[meridian-point-in-time-philosophy]].)
- Not a continuous monitor.
- Not a general network scanner / no aggressive sweeping — the peer sample is
  small and gentle.
- It **names** the fix; it does not apply it (never flips firewall settings or
  TCC for the user).
- No AI / LLM anywhere in the path.
- **IPv6-only LANs are out of scope for v1.** The recipe is IPv4-primary (IPv4
  anycast internet check, IPv4 gateway/link-local logic). On a modern IPv6-only
  network (GUA via SLAAC, no IPv4) the tool must report `INCONCLUSIVE` — a stated
  limitation — **not** a false `DHCP_FAILED`. A full IPv6 path is a later slice.

## Anti-goals (failure even if it "works")

- **Crying wolf** — declaring "network broken" when a single host is down. This
  erodes the trust that makes the tool worth having. The `HOST_DOWN` verdict
  exists precisely to prevent this.
- **A failed/slow probe rendering as a confident verdict.** An errored or
  timed-out probe that we can't classify yields `INCONCLUSIVE`, stating which step
  failed — not a false "isolation" / "all clear".
- **Requiring privilege.** The entire point is diagnosing the *unprivileged*
  app's experience. The core verdict must be reachable with **no `sudo`**. It must
  not depend on the raw-socket `arpscan` sweep; when a privileged capability is
  unavailable that is expected and must not degrade the verdict to a false
  negative.

## Constraints

- Deterministic decision tree, no AI. (User: "I don't want to use AI or anything
  here.")
- Must run unprivileged (see anti-goals).
- Nielsen ([[ui-principles-nielsen]]): **visibility of system status** — show each
  step as it runs; **recognition not recall** — name the concrete fix inline,
  per [[remediation-name-the-fix]].
- CI does not build the frontend ([[meridian-ci-no-frontend-build]]) — the builder
  must `npm run build` the Svelte and pipeline-check it locally.
- New-tool checklist ([[meridian-new-tool-checklist]]): CLI subcommand, README
  write-up, screenshots.
- Secrets hygiene: N/A here (no SNMP community strings in this tool's path), but do
  not surface any system firewall rule content that contains secrets verbatim.

## The recipe (ordered probes)

Each step is one deterministic probe. Steps short-circuit where a terminal verdict
is already certain. All probes use **unprivileged** ICMP (`scan.NewICMPPinger()`,
which threads an `available bool`) with a **TCP-connect fallback** when ICMP is
unavailable — and `available == false` is reported as "couldn't test by ICMP",
never as "host down".

1. **Identity** — `myip.Local()` → `PrimaryIP`, `PrimaryIf`, `Gateway`; plus
   `netif.List()` for the interface prefix/subnet. Derive `hasGateway` and
   `isLinkLocal` (IPv4 `169.254/16`, or only `fe80::` / no routable address).
2. **Gateway reachability** — ICMP to the gateway, with a TCP-connect fallback when
   ICMP is unavailable. The fallback must treat a **refused** connection as
   *reachable* (the host answered) and use a wide port set (e.g. 80/443/53/8080),
   so an alive-but-ICMP-filtered or odd-admin-port router doesn't read as
   unreachable.
3. **Internet reachability** — TCP-connect to a small fixed set of well-known
   anycast IPs **by raw IP** (no DNS dependency), e.g. `1.1.1.1:443`,
   `8.8.8.8:53`, `9.9.9.9:443`. (Not blocked by Local Network privacy — these are
   off-LAN.) Optionally a separate DNS-resolution check to tell "DNS broken" from
   "no internet" (builder's call; stretch).
4. **LAN peer reachability** — probe a bounded **peer sample**, seeded in priority
   order: (a) the user's target if provided, (b) passive neighbor-table entries
   (filtered to reachable/stale-but-present states — drop `INCOMPLETE`/`FAILED`),
   (c) a small gentle set of in-subnet addresses as last resort. **Exclude the
   gateway** from the sample (it's already proven in step 2 and usually stays
   reachable under isolation, so counting it would mask a real block). Record
   `peersTried` / `peersReached` and whether the user's specific target was
   reached.
5. **OS-block check (per-OS, tri-state)** — returns `blocked` / `notBlocked` /
   `couldNotDetermine`. "Meridian's own socket" here is a **plain TCP connect**
   (what the real LAN tools use — `scan.ScanPorts` / `scan.NewICMPPinger`), **not**
   the `internal/trace` prober (which opens a raw socket and needs root on Linux).
   Only meaningful when a trusted baseline can reach at least one peer:
   - **macOS**: an Apple binary (`/sbin/ping`, `nc` — TCC-exempt) reaches a peer
     the app's own connect can't ⇒ `blocked`. (The exact signature from the printer
     case: `nc` worked, the app didn't.) Both fail / can't run baseline ⇒
     `notBlocked` / `couldNotDetermine`.
   - **Windows**: compare baseline-vs-own-connect, and query the active
     connection's **network category** (Public vs Private) + Defender Firewall for a
     per-binary block as corroboration.
   - **Linux**: no per-app LAN gating exists, so this essentially never returns
     `blocked` (see verdict note). Best-effort `ufw`/`firewalld` status query (a
     `Supported()`-gated, root-dependent enrichment — *not* a deterministic driver)
     feeds the `LOCAL_ACCESS_BLOCKED` detail.

## Verdict logic (pure decision tree)

A pure function `Classify(Observations) Report` — the TDD core. The OS-block check
is **tri-state** (`blocked`, `notBlocked`, `couldNotDetermine`), matching the
repo's `Supported()` idiom. `peersReached==0` is only *trusted* when the peer
sample met the **confidence threshold** (below); otherwise the peer dimension is
`INCONCLUSIVE`, never a guessed verdict.

| # | Condition | Verdict |
|---|-----------|---------|
| 1 | `!hasGateway \|\| isLinkLocal` | **DHCP_FAILED** — not actually on a network |
| 2 | gateway unreachable | **GATEWAY_UNREACHABLE** — associated but the router isn't answering (bad link) |
| 3 | gateway ✓, internet ✗, internet probe intercepted (RST/hijack, not clean timeout) | **CAPTIVE_PORTAL** — a login page is intercepting traffic *(detection is stretch; see builder notes)* |
| 4 | gateway ✓, internet ✗ (clean), peers reached | **INTERNET_DOWN** — your LAN is fine, the internet/WAN is down |
| 5 | internet ✓, peer sample confident, `peersReached==0`, OS-block **positive** | **OS_BLOCKING_MERIDIAN** — this machine is blocking Meridian specifically (per-OS copy; see note) |
| 6 | internet ✓, peer sample confident, `peersReached==0`, OS-block **notBlocked** (baseline *also* can't reach any peer — proven not app-specific) | **CLIENT_ISOLATION** — hotspot / guest Wi-Fi isolates clients |
| 7 | internet ✓, peer sample confident, `peersReached==0`, OS-block **couldNotDetermine** | **LOCAL_ACCESS_BLOCKED** — the blunt-fact verdict (below) |
| 8 | `peersReached>0` but the user's target specifically unreachable | **HOST_DOWN** — network fine, that host isn't answering |
| 9 | gateway ✓, internet ✓, peers ✓ (and target ✓ or none given) | **ALL_CLEAR** |
| 10 | a required probe errored, or the peer sample wasn't confident | **INCONCLUSIVE** — names the failed/thin step |

**Row 7 — `LOCAL_ACCESS_BLOCKED` (owner decision, 2026-11-19):** when we can
*see* that no local device is reachable but **can't prove** whether it's the
network isolating everyone or something on this machine blocking Meridian (the
unprivileged-Linux case, or any time the baseline comparison can't run), the tool
states the **observable truth confidently** — "Meridian can't reach any device on
this local network (the gateway and internet are fine)" — then lists **both**
candidate causes with their fixes, and tells the user how to pin it down (re-run
with `sudo`, or grant Local Network access). It does **not** guess a single cause.
This is the chosen behavior over "guess isolation" and over "say inconclusive."

**Ordering note:** rows 5→6→7 separate "*only Meridian* is blocked" (proven
app-specific) from "*nobody* can reach peers" (proven network isolation) from
"can't prove which" (blunt fact). This is the heart of the recipe.

**Per-OS meaning of row 5 (`OS_BLOCKING_MERIDIAN`):** the claim "the OS singled out
Meridian" is only literally true where the OS does per-application LAN gating:
- **macOS** — Local Network privacy (TCC) is genuinely per-app. The positive
  signature (Apple-exempt `nc`/`ping` reaches a peer the app can't) fires here.
  Fix: System Settings › Privacy & Security › Local Network › enable Meridian.
- **Windows** — Defender Firewall can block per-binary, and a Public network
  profile blocks LAN. Positive signature possible. Fix: allow Meridian through the
  firewall, or set the network to Private.
- **Linux** — there is **no per-app LAN blocking** (`ufw`/`firewalld`/`iptables`
  are app-agnostic by port/addr). So the row-5 positive signature effectively
  never fires on Linux; Linux lands on row 6 (proven isolation) or row 7
  (`LOCAL_ACCESS_BLOCKED`), and a best-effort firewall-status query *enriches* the
  row-7 detail ("`ufw` is active and may be dropping local traffic") without being
  the deterministic driver.

**Peer-sample confidence threshold:** `peersReached==0` drives the scariest
verdicts, so it must be earned. Require at least **K distinct, likely-up peers**
(default K≈2) probed-and-failed — where the user's *explicit target* counts as a
strong signal and a single stale ARP entry does **not** — before `peersReached==0`
is trusted. Below the threshold ⇒ `INCONCLUSIVE`. Exact K is builder-tunable; the
principle is fixed (it's the load-bearing half of the "don't cry wolf" promise).

Each verdict carries a plain statement, the cause(s), and the concrete fix. Exact
copy is the builder's, following this intent.

## How it's made (decisions taken)

Follows the repo's 4-touch tool pattern (pure engine package → thin `app` binding
→ CLI → Svelte), confirmed against existing tools (`ipp`/printer is the exemplar).

- **New engine package `internal/netdiag/`:**
  - `netdiag.go` — pure types (`StepResult`, `Observations`, `Report`, a `Verdict`
    enum) and the pure `Classify(Observations) Report`. No host access here →
    exhaustively unit-tested over result patterns (inline string-literal fixtures +
    func-var seams; the repo has **no** golden/testdata convention).
  - `run.go` — `func Run(ctx context.Context, opts Options, onStep func(StepResult)) (Report, error)`
    orchestrates the probes in order, streams each `StepResult`, assembles
    `Observations`, calls `Classify`. Probe calls go through interface/func-var
    seams so tests fake them.
  - Reuses: `myip.Local()`, `netif.List()`, `netroute.DefaultGateway()`,
    `scan.NewICMPPinger()` / `scan.ScanPorts` / `trace.NewTCPProber`,
    `ndp.Neighbors()` (IPv6 passive table).
  - **New passive IPv4 neighbor-table reader** (the map confirmed none exists) —
    per-OS, func-var + `Supported()` idiom: darwin `arp -an`, linux `/proc/net/arp`
    or `ip neigh`, windows `arp -a` / `GetIpNetTable`. Passive (no raw socket) so it
    works unprivileged. Lives in `netdiag` (e.g. `neigh_darwin.go` …) or a small
    `internal/neigh/` package — builder's call.
  - **OS-block check** — `osblock_darwin.go` / `_linux.go` / `_windows.go` /
    `_other.go`, each a build-tagged `osBlockCheck(ctx, peer, port) (blocked bool,
    detail string, supported bool)` wired via func-var + `Supported()` (the
    `netroute`/`ndp` idiom).
- **Binding** `internal/app/netdiag_methods.go` — thin:
  `func (a *App) NetworkDiagnostic(target string) (netdiag.Report, error)`,
  clamp/timeout/delegate. The full `Report` carries every `StepResult`, so a
  synchronous return renders all steps. Live per-step streaming (Wails events, like
  the ping monitor) is optional polish, not required for v1.
- **CLI** `internal/cli/netdiag.go` (`runNetDiag`, `--json`) + a `catalog.go` entry
  under "Local network". Reuses the same `netdiag.Run`.
- **Frontend** `frontend/src/lib/tools/NetDiagnostic.svelte` + 3 edits in
  `Workbench.svelte` (import, registry entry `{ id:"netdiag", name:"Local network
  diagnostic", group:"Local network", ready:true, kw:"diagnostic troubleshoot
  reachability isolation hotspot firewall local network privacy gateway dhcp" }`,
  render branch `{:else if active==="netdiag"}`). Renders the ordered steps
  (pass/fail/skip + detail) and a prominent verdict card (statement · cause · fix).
  Regenerate `frontend/wailsjs/go/app/` bindings. Uses shared CSS vars + `ToolHelp`.
- **Failure integration** — **reuse the existing nav store**, don't invent
  plumbing. `frontend/src/lib/nav.ts` already exports
  `requestedTool = writable<{ tool: string; value?: string }>`, Workbench already
  consumes it, and ~10 tools already pre-fill via it (e.g. `Bgp.svelte` →
  `requestedTool.set({ tool: "lookup", value })`). A failing LAN tool calls
  `requestedTool.set({ tool: "netdiag", value: host })`; `NetDiagnostic.svelte`
  reads it on mount and auto-runs. Each LAN tool (printer, ping, PortScan,
  PingSweep, Discover, ArpScan, SNMP) renders a "Run Local Network Diagnostic"
  button **only on a reachability/timeout failure** — *not* on auth failures (wrong
  SNMP community) or parse errors, where the LAN is clearly fine. Which failure
  shape qualifies is per-tool.

## Review focus (inputs the recipe must not mishandle)

- **ICMP unavailable** (`available==false`) must not read as "host down" — fall
  back to TCP and label honestly.
- **No testable peer** (empty neighbor table, no target) → `INCONCLUSIVE` for the
  peer dimension, never a guessed `CLIENT_ISOLATION`.
- **Single dead host** with a healthy LAN → `HOST_DOWN`, not a network verdict.
- **Runs without `sudo`** and without the `arpscan` raw-socket path.
- **OS-block check is the hardest to test** — `Classify` is fully unit-tested; the
  per-OS `osBlockCheck` probes are seam-faked in tests, and their real behavior
  needs **manual verification on each OS** (macOS path already corroborated by the
  printer case). Call this out; don't claim the OS-block branches are proven by
  unit tests alone.

## Left to the builder

Exact timeouts; peer-sample size and the no-neighbors fallback strategy; the fixed
internet-check IPs; the precise per-OS firewall-query commands and their parsing;
optional captive-portal detection; synchronous-report vs live-step streaming;
exact verdict copy and card layout; the `openDiagnostic(host)` wiring mechanism.

## Decisions log (from the builder-check, 2026-11-19)

- **Owner:** the can't-prove-the-cause case → `LOCAL_ACCESS_BLOCKED` (blunt
  observable fact + both candidate causes + how to disambiguate), not a guessed
  isolation and not a bare "inconclusive".
- **Author calls:** added `INTERNET_DOWN` (+ `CAPTIVE_PORTAL` sub-case) for the
  `gateway ✓ / internet ✗` states the first draft dropped; reuse the existing
  `requestedTool` nav store for failure hand-off; exclude the gateway from
  `peersReached` and filter stale ARP entries; probe "own reachability" with a
  plain TCP connect, not the privileged `trace` prober; scope v1 to IPv4
  (IPv6-only ⇒ `INCONCLUSIVE`, not a misfire); peer-sample confidence threshold
  before trusting `peersReached==0`; per-OS semantics for the OS-block verdict
  (per-app on macOS/Windows; app-agnostic firewall enrichment on Linux); firewall
  queries are best-effort `Supported()`-gated enrichment, not deterministic
  drivers.

---

*Builds on [[engineers-tools-project]]; motivated by [[owner-mac-little-snitch]]
(macOS Local Network privacy as the printer root cause) and the now-shipped
`NSLocalNetworkUsageDescription` (#266). Related specs:
`docs/superpowers/specs/2026-11-09-meridian-localnet-slice1-design.md`.*
