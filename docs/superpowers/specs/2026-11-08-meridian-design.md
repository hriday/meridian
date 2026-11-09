# Meridian — Design

**Status:** approved scope, pre-plan
**Date:** 2026-11-08
**Repo:** `github.com/hriday/meridian` (private until release)
**Licence:** Apache 2.0

---

## 1. Intent and why

Meridian is an open-source desktop app for **getting information about networks, addresses and routes** — and recording your own address space. It is cross-platform (Windows, Linux, macOS).

It grew out of wanting an open-source answer to SolarWinds Engineer's Toolset, but the goal is **not** a SolarWinds clone. SolarWinds' 61-tool list is a checklist of what people expect, not the target. The target is a solid, fast, beautiful tool an engineer opens to *find something out*: who owns an IP, what AS a route passes through, how to carve a subnet, what's recorded against a block.

The owner's own stated needs drove the core: "see the AS hops", "what the IP ranges are", "calculate subnets", "record IP space". Those are the everyday centre; the broader tool set surrounds them.

**Why these choices:** the owner reaches for ipduh.com and Hurricane Electric's BGP toolkit/looking glass and wants that quality — any-value search, pivot-everywhere linking, consistent per-entity pages — with their weak spots fixed (no source/freshness labels; unbounded unsorted lists; walls of text).

## 2. Goals, non-goals, anti-goals

**Goals**
- A fast, polished, genuinely nice-to-use desktop app. UI/UX is a first-order requirement, held to **Nielsen's 10 usability heuristics**.
- Cover the everyday core (AS traceroute, lookups, subnet calc, IP-space records) excellently.
- Cover the ~60 SolarWinds-style tools (minus non-network ones).
- First-class IPv4 **and** IPv6 everywhere; BGP, AS numbers, RPKI, RDAP, PeeringDB.
- Multi-vendor from day one: Cisco, Fortinet, Palo Alto, Juniper, Arista.
- Every result labelled with **where the data came from and how fresh it is**.
- A CLI mode for every tool with JSON output, for scripting.

**Non-goals**
- Meridian never hosts anything for anyone. Sync uses storage the user already controls (git, S3-compatible, or a small self-run sync binary).
- No accounts, sign-up, or telemetry. The app reaches the internet only for lookups the user asks for.
- Not a continuous NMS/alerting platform (LibreNMS/PRTG territory) — it's an on-demand toolkit. The listener/collector tools *do* run persistently (see below), but Meridian is not an alerting/paging backend.

**Anti-goals (failure even if it technically works)**
- Slow or clunky to use.
- Ugly. The owner called this "unpardonable".
- Copying SolarWinds' names, branding, or UI.
- Dumping unsorted/unpaged lists or unlabelled data (the HE/ipduh failure mode).

## 3. Constraints and decisions made

- **Stack:** Go networking engine + **Wails** desktop UI (web UI in a native window; one small binary per OS, no bundled browser). *My call; owner deferred stack choice.*
- **Licence:** Apache 2.0. *My call.*
- **Release:** **core first, then open up.** Repo stays private until the everyday core (lookups + entity pages, AS traceroute, v4/v6 subnet calc, local IP-space with change log) is polished; it's published at that point, and the ~60 tools, five vendors and security suite arrive as later public updates. *Owner.* (Supersedes the earlier "all-or-nothing" reading.)
- **Primary layout:** "Workbench" — tool sidebar + tabs + a ⌘K any-value search bar (chosen over an ask-bar-first and a launchpad-grid alternative). *Owner picked from mockups.*
- **Lookup design:** one page per entity (IP / prefix / ASN / domain), opening in tabs; a one-line pivot header showing the whole chain (IP → prefix → origin AS → RPKI/registry), every item clickable; consistent sub-tabs (Overview, Routing, Registry, DNS, Reachability); per-card source + freshness. *Owner approved mockup.*
- **Looking glass:** we have no routers of our own. Use free public route collectors (**RIPE RIS, RouteViews**) for route/BGP views and **RIPE Atlas** for remote ping/traceroute (needs the user's own free Atlas key + credits). Link out to HE rather than scrape it. *My call, owner aware.*
- **Windows-only SolarWinds tools:** WMI Browser kept but shown only on Windows; Permissions Analyzer (NTFS/share perms) dropped as non-network. *My call.*
- **SNMP brute-force/dictionary tools:** kept as audit tools for your own gear, off by default, with a warning. *My call.*

## 4. Feature areas

### 4.1 Lookups (core)
Any-value search recognises IP/prefix/ASN/domain/hostname. One entity page each, tabbed, pivoting, source-labelled. Actions per entity: trace, ping, scan ports, check blocklists, TLS cert, **Save to IP space**, copy as JSON. Data: RDAP (RIR), RIPEstat/collectors for routing, RPKI validator, PeeringDB for IX/peering. A lookup of one of *your own* addresses shows your IP-space notes beside the public data.

### 4.2 AS-aware traceroute (core)
Traceroute where each hop shows AS number, owner, latency, and AS boundaries highlighted; show the BGP AS path alongside (may differ from the packet path). MTR-style live view.

### 4.3 Subnet calculator (core)
IPv4 and IPv6. Split, merge, summarise ranges; plan ("12 × /24 from this block"). Embedded into IP-space (carve a recorded block) and into lookups.

### 4.4 IP-space records (core)
- Nested blocks → subnets → addresses, starting at **any** prefix size (/24, /29, IPv6 /48→/64, etc.); nesting optional.
- Fields per block/subnet: site, VLAN, owner, purpose, location, notes; sites/VLANs chosen from a list for consistency; user-defined custom fields.
- Per address: DNS name, device, MAC, notes. Button to fill names from reverse DNS across a subnet; flag mismatches vs live DNS.
- Guards: reject overlapping subnets and duplicate addresses; flag real clashes with an explanation.
- Planning: per-subnet usage bars; "next free subnet/address" button.
- CSV import/export.
- **Team-aware from the start.** Shared/team editing is in scope (owner: "if the db is shared, we should guard against concurrency and have logging and rollbacks"), without Meridian running any service or accounts. So the **milestone-1 local schema already carries actor identity and conflict metadata** — adding them later would force a data migration. *Actor identity source (my call):* a name set in config, else the OS username, else the git commit author when syncing via git. No login.
- **Storage:** local-first, encrypted SQLite by default. Every change is an append-only **log entry** (who/what/when/why) → that one design gives history, per-change **undo/rollback**, and point-in-time restore. Optional **sync** to a store the user controls: a git repo, an S3-compatible bucket (AWS/MinIO/Backblaze/R2), or a small self-run sync binary we ship. Encrypted client-side before upload. Concurrency: both edits logged; real conflicts (two people assign the same /24) flagged for a human, never silently overwritten. A raw DB file on a shared drive is explicitly **not** supported.

### 4.5 Looking glass
Run ping/traceroute/BGP-route from public vantage points, not just the local machine. Collectors (RIPE RIS/RouteViews) for routes; RIPE Atlas for remote probes (user's own key). Searchable vantage-point picker (fixing HE's wall-of-text list).

### 4.6 SolarWinds-style tool set (~60)
SNMP monitors (CPU/memory/bandwidth/interface/temperature, realtime graphs), SNMP sweep/MIB browser/walk/trap receiver, ping sweep, enhanced ping, port scanner, DNS audit, DNS & Whois, DHCP scope monitor, route viewer, NetFlow realtime collector + generator, WAN traffic generator, syslog server, TFTP server, SFTP/SCP server, Telnet/SSH terminal, Wake-on-LAN, MAC finder, switch port mapper, neighbour map (CDP/LLDP), etc. Windows-only: WMI Browser.

**Persistent collectors (owner decision):** the "always listening" tools — syslog server, SNMP trap receiver, NetFlow collector, TFTP/SFTP servers — are installable as **OS background services/daemons**, so they keep capturing when the app window is closed. This pulls in per-OS service registration, persistent privileged-port binding, and service lifecycle management (install/start/stop/status from the UI). The GUI attaches to the running service to show live data. (Meridian still isn't an alerting backend — it captures and displays; it doesn't page.)

### 4.7 Multi-vendor configs & monitoring
Per-vendor **plug-in** architecture; day-one vendors: **Cisco, Fortinet, Palo Alto, Juniper, Arista**. Use each vendor's best transport: Cisco SSH/RESTCONF; FortiOS REST; PAN-OS XML API (candidate vs running); Junos NETCONF (built-in compare/rollback); Arista eAPI (JSON). Config download/compare/history/edit across all. Firewalls get a **policy viewer** (search rules; find what permits given traffic; flag unused/shadowed rules) in place of Cisco's ACL editor. Monitoring includes firewall stats (sessions, HA, VPN tunnels, CPU). Cisco-only extras (type-7 decrypt, NetFlow setup wizard) stay Cisco-only.

### 4.8 Security tools
- SNMP brute-force / dictionary (audit your own gear; off by default; warned).
- **AD password-strength audit:** confirm-authorisation gate (logged) on first use. Input: an NTLM hash dump (from `ntds.dit`) the user already extracted — Meridian does not pull hashes from a DC. Engine: wrap **hashcat** (GPU) with **John the Ripper** fallback; show progress. Wordlist manager downloads common public lists (rockyou etc.) on demand (not shipped in-repo). Report shows **plaintext per cracked account** plus verdict (weak/reused/in-known-list/fails-policy) and a **pattern view** (shared base word, reuse, season/year suffixes) that points at a policy fix. Result is sensitive: kept **local + encrypted**, never synced by default, one-click delete. *Engine binaries (my call):* assume user-installed hashcat/John; Meridian detects them and guides install if missing, rather than shipping or auto-downloading them (avoids GPU-driver/licensing/distribution burden under a public Apache-2.0 repo). Only wordlists are downloaded on demand.
- **Netcat utility:** connect/listen on TCP/UDP, type or pipe data, see responses and banners. Modern UI over classic `nc`.

### 4.9 Cross-cutting
- CLI mode + JSON output for every tool.
- Source + freshness label on every result card.
- Long lists always have search, sort, paging.
- UI held to Nielsen's heuristics; speed and visual polish are acceptance criteria, not later polish. **Measurable bar (my call):** UI interactions respond in <100 ms; no network/scan/crack operation ever blocks the UI thread (results stream/progress); app cold-start to usable in <2 s; large lists virtualised. The **subjective visual bar is the owner's** — the owner approves the look per milestone (they picked the layout and entity-page mockups).

## 5. Left to the builder
- Detailed module/package layout and the plug-in interface shape (vendor plug-ins; sync-backend plug-ins; data-source adapters).
- Which Go libraries (e.g. gosnmp, miekg/dns, x/net/icmp, x/crypto/ssh, NETCONF/eAPI clients) and how raw-socket/privilege handling works per OS (ICMP needs elevation).
- Encryption scheme and key handling for local DB and sync.
- Build order / milestones (proposed first milestone below, to confirm in planning).
- Testing approach without real gear (GNS3/EVE-NG virtual devices; snmpsim; containerlab for multi-vendor).

**First milestone = the public v1 (owner: "core first, then open up").** The everyday core — any-value lookups + entity pages, AS-aware traceroute, v4/v6 subnet calculator, and local IP-space records with the change log — in the Workbench shell, polished to the acceptance bar in §3. Its schema already carries actor identity + conflict metadata (§4.4) even though sync ships later. When this is ready, the repo goes public. Later public updates add: sync backends, the full SolarWinds tool set (incl. persistent collector services), multi-vendor configs/monitoring, and the security suite.
