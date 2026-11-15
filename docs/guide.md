# How to use Meridian

Meridian is a local-first network workbench: dozens of network, DNS, TLS, BGP,
SNMP and security tools in one desktop app (and one CLI), each **point-in-time** —
go in, get actionable data, move on. Nothing is retained, and anything that would
normally mean pasting a secret into a website is done **on your machine** instead.

This guide covers how the UI works. Each tool also explains itself in-app, and the
full catalog is listed at the end.

---

## Quickstart — 3 real workflows

New here? These show how the pieces fit together. Each works in the app (pick the
named tools from the sidebar) or from the CLI in one line.

**1. Triage a slow or flaky host.** Is it reachable, how's the latency and loss,
where does the path break, and is the TLS endpoint healthy? Run **Ping**,
**MTR**, **TCP ping** and **Traceroute** — or let the built-in runbook do all of
it into one report:

```
meridian-cli runbook triage-host 10.0.0.5
```

**2. Audit a domain's security posture (TLS + DNS + email).** Check the
certificate and cipher grade, DNS/DNSSEC hygiene, SPF/DKIM/DMARC, and HTTP
security headers in one pass with **Cipher scan**, **DNS audit**, **Email auth**
and **Security headers** — or the runbook:

```
meridian-cli runbook audit-domain example.com
```

**3. Find rogue devices on your LAN and reconcile your IP inventory.** Sweep the
LAN with **ARP scan** / **Service discovery**, then **LAN reconcile** diffs what's
actually on the wire against your encrypted **IP space** inventory — new hosts,
missing hosts, conflicts — and you import the changes. Share that inventory across
a team with its built-in encrypted **sync** (your own S3/SFTP/git/local backend —
Meridian never hosts it).

```
meridian-cli arpscan --json            # who's on the LAN right now
meridian-cli ipspace reconcile         # diff the live sweep vs your inventory
```

Once a workflow is routine, write your own **runbook profile** (YAML/TOML/JSON) so
`meridian-cli runbook ./my-profile.yaml <target>` runs exactly your checks.

---

## The workbench

The window is a **sidebar** of tools on the left and the **active tool** on the
right.

- **Pick a tool** — tools are grouped in the sidebar (Internet, Addressing, Local
  network, SNMP & monitoring, Collectors, Security, Config insight). Click one and
  it opens on the right. The header shows the active tool's name.
- **Command palette (⌘K / Ctrl-K)** — a keyboard launcher for the whole toolset:
  start typing a tool name to fuzzy-jump to it, or **paste a value** (an IP, CIDR,
  domain, URL, JWT, PEM, MAC or hash) and it recognises the type and jumps to the
  right tool with the value pre-filled. ↑/↓ to move, Enter to open, Esc to close.
- **Collapse the sidebar** — the **☰** button at the top gives the active tool more
  room.
- **Run a tool** — type your input (an IP, host, prefix, file, token…) and press
  the tool's button or Enter. Results appear in place. Live tools (monitors,
  receivers) stream until you stop them.
- **Footer** — the bottom of the sidebar shows your version, an **update check**
  (manual — Meridian never phones home on its own), **Install command-line tool**
  (adds `meridian-cli` to your `PATH`), and a link to **this guide**.

## Themes

A toggle in the top-right switches between three themes — **dark**, **light**, and
a green-on-black **terminal** mode. Pick whichever suits your environment; it's
purely cosmetic and remembered between launches.

## Point-in-time & privacy

Meridian is built around two ideas:

- **Point-in-time, not historical.** Each tool answers "what's true right now" and
  moves on. There's no database of past results, no trending, no retention. (The
  one deliberate exception is **IP space**, your own encrypted inventory, which you
  explicitly save.)
- **Local-first, outbound only when you ask.** Tools that work with secrets — JWT,
  TOTP, bcrypt/htpasswd, encode/decode, AD password audit, Cisco password decode —
  run **entirely on your machine**, so tokens and passwords never hit a website.
  Tools that must reach the network (lookups, DNS, TLS, BGP…) only do so when you
  run them, and send only what the query needs — never who you are. Meridian also
  **never hosts anything for others**: receivers listen locally, and inventory sync
  stores only an encrypted blob on a backend you control.

## What needs elevated privileges

Most tools need no special rights. A few do, because they use raw sockets or
low ports:

| Tool | Why | Note |
|---|---|---|
| ARP scan, **LAN reconcile** | raw layer-2 | `sudo` / admin — no workaround |
| LLDP/CDP, RTP analyzer, **Packet summary** | packet capture | `sudo` / admin |
| **TCP traceroute** | raw ICMP for hop replies | **no sudo on macOS**; `sudo` on Linux |
| Syslog (514), SNMP traps (162), TFTP (69) | privileged ports | `sudo`, or use a high `--port` |

Everything else — including ICMP ping/traceroute/MTR, `tcpping`, and service
discovery — runs unprivileged. In the **desktop app**, launched normally, the
raw-socket tools above will report "needs elevated privileges" (the app runs as
your user, like Wireshark); the CLI with `sudo` is the clean path for those.

## The command line

Every tool is also a `meridian-cli` subcommand sharing the exact same engine, so
anything in the GUI can be scripted, piped, or run over SSH. Install it from the
sidebar footer (macOS) or build it from source, then:

```bash
meridian-cli help          # every command, grouped
meridian-cli <command>     # run with no args to see its usage
```

Most commands take `--json`. Live receivers/monitors run until **Ctrl-C**.

**Bulk runs** — pipe a newline-delimited target list into `batch` to run any
single-target command over many targets at once (NDJSON with `--json`):

```bash
cat hosts.txt | meridian-cli batch cipherscan --json
```

---

## Every tool at a glance

> In the GUI these appear in the sidebar; the CLI column is the `meridian-cli`
> subcommand. This is a one-line index — open a tool (or run it with no args) for
> the full story.

### Internet

| Tool | What it does | CLI |
|---|---|---|
| Lookup | IP / ASN / prefix / domain intelligence (routing, registration, AS ownership, RPKI) | `lookup` |
| Traceroute | AS-aware path; ICMP or TCP-SYN; each network's PeeringDB peering/IX | `trace`, `tcptrace` |
| BGP & looking glass | origin AS, AS paths, upstreams/transit, RPKI + IRR | `bgp` |
| TLS certificate | inspect a live server's certificate | `tls` |
| Cert & CSR decoder | decode a pasted PEM cert/CSR locally | `certinfo` |
| Cipher scan | TLS versions + ciphers, graded, with post-quantum readiness | `cipherscan` |
| SSH audit | SSH algorithms, host keys, post-quantum readiness | `sshaudit` |
| HTTP check | HTTP(S) request/response check | `http` |
| HTTP/3 reachability | QUIC handshake + Alt-Svc — is HTTP/3 reachable, or blocked on your path? | `http3` |
| IKE / IPsec probe | IKEv2 VPN responder: accepted proposal (enc/PRF/integ/DH) + NAT-T | `ike` |
| Security headers | OWASP-mapped security header review | `sechdr` |
| WHOIS | domain / IP / ASN whois | `whois` |
| Subdomains | subdomains via Certificate Transparency | `subdomains` |
| DNS blocklist | DNSBL check for an IP | `blocklist` |
| DNS audit | SPF / DMARC / DNSSEC posture | `dnsaudit` |
| DNS propagation | one record across global resolvers | `dnspropagation` |
| Email auth | SPF / DKIM / DMARC | `emailauth` |
| SMTP probe | banner / STARTTLS / cert | `smtp` |
| SIP OPTIONS ping | is VoIP signaling up? | `sip` |
| STUN | your public IP + NAT type | `stun` |
| Cert expiry | bulk TLS cert days-to-expiry | `certexpiry` |
| Let's Encrypt | one-shot DNS-01 issuance | `certissue` |
| Service status | Statuspage outage check | `status` |

### Addressing

| Tool | What it does | CLI |
|---|---|---|
| Subnet calculator | network/broadcast/host math + prefix splitting | `subnet` |
| Classify | classify an address or value | `classify` |
| IP space | encrypted, append-only IP inventory — nested blocks, carve into subnets, utilization %, optional sync | `ipspace` |
| LAN reconcile | diff a live ARP sweep against your inventory; import/update | `ipspace reconcile` |

### Local / environment

| Tool | What it does | CLI |
|---|---|---|
| NTP | query an NTP server (stratum/offset/verdict) | `ntp` |
| DNS | any record, any resolver, DNSSEC signals; UDP/TCP or DoT/DoH/DoQ | `dns` |
| Interfaces | local network interfaces | `interfaces` |

### Local network

| Tool | What it does | CLI |
|---|---|---|
| Ping / Ping monitor | ping, and live ping | `ping`, `pingmon` |
| TCP ping | connect latency + loss to a port (works when ICMP is blocked) | `tcpping` |
| Connection quality | latency, jitter, loss, MOS verdict | `quality` |
| VoIP bandwidth | codec call capacity planner | `codecbw` |
| MTR | live path + per-hop loss | `mtr` |
| Path MTU | find MTU black holes | `pathmtu` |
| DHCP discover | find DHCP servers / rogues | `dhcpdiscover` |
| LLDP/CDP | what switch & port am I on (sudo) | `lldp` |
| ARP scan | live MAC↔IP + vendor (sudo) | `arpscan` |
| Service discovery | mDNS/Bonjour + SSDP/UPnP — what's on the LAN | `discover` |
| RTP analyzer | per-stream jitter/loss/codec (sudo) | `rtp` |
| Packet summary | tcpdump-lite: top talkers, TCP resets/retransmits, DNS errors, TLS SNIs (sudo) | `pktscan` |
| Wi-Fi analyzer | channel, signal/SNR, neighbors + channel map / co-channel contention (macOS) | `wifi` |
| Ping sweep | ping / TCP sweep a range | `pingsweep` |
| Port scan | scan ports on a host | `portscan` |
| PTR sweep | reverse-DNS a range | `ptrsweep` |
| Wake-on-LAN | send a magic packet | `wol` |
| MAC vendor | OUI → vendor lookup | `macvendor` |
| Speed test | iperf3 throughput client | `speedtest` |

### SNMP & monitoring

| Tool | What it does | CLI |
|---|---|---|
| SNMP sweep | find SNMP responders | `snmpsweep` |
| Device health | device health snapshot | `devhealth` |
| Switch-port mapper | MAC → switch port table | `swport` |
| MIB browser | SNMP walk | `mib` |
| SNMP community audit | authorization-gated community check | `snmpbrute` |

### Collectors (local receivers — Ctrl-C to stop; low ports need sudo)

| Tool | What it does | CLI |
|---|---|---|
| Syslog | syslog receiver | `syslog` |
| SNMP traps | trap receiver | `traps` |
| NetFlow | NetFlow v5 / v9 / IPFIX receiver | `netflow` |
| sFlow | sFlow v5 receiver — sampled flows → top talkers | `sflow` |
| TFTP | TFTP receiver (saves grabbed files) | `tftp` |

### Security

| Tool | What it does | CLI |
|---|---|---|
| Netcat | connect / listen, UDP / TLS | `nc` |
| JWT decoder + verifier | header/claims/expiry, verified locally | `jwt` |
| TOTP | current 2FA code from a secret | `totp` |
| bcrypt / htpasswd | hash / verify, locally | `htpasswd` |
| Encode / decode | base64 / hex / URL / JSON | `transcode` |
| AD password audit | authorized self-audit, HIBP k-anonymity | `adaudit` |
| Cisco password decode | Cisco type-7 / secret analysis | `ciscopw` |
| Hash identifier | identify a hash type | `hashid` |
| Breach lookup | HIBP breaches | `breaches` |
| CVE lookup | NVD CVE lookup | `cve` |
| Threat intel | abuse.ch botnet C2 check | `threatintel` |

### Scripting & workflows

| Tool | What it does | CLI |
|---|---|---|
| Batch | run a single-target command over a stdin list of targets (NDJSON with `--json`) | `batch` (CLI) |
| Runbook | composite audit — run a profile of tools into one report: built-in (audit-domain, triage-host) or your own YAML/TOML/JSON profile; md/json/html | `runbook` + GUI |

### Config insight

| Tool | What it does | CLI |
|---|---|---|
| Firewall analyzer | FortiGate/PAN-OS/ASA/IOS/NX-OS/SRX/Junos/Arista/Check Point/VyOS/pfSense (auto-detected) | `fwanalyze` |
| Web-server analyzer | nginx / Apache / Caddy config review | `webanalyze` |
| Terraform exposure | internet-exposure scan from HCL / plan — AWS, GCP, Azure | `tfexposure` |
| Config diff | section-aware config compare | `cfgdiff` |

---

*Meridian is open source. Found a rough edge or want a tool? Open an issue on
[GitHub](https://github.com/hriday/meridian).*
