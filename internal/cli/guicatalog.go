package cli

// GUITool documents one tool as it appears in the desktop app: the friendly
// name shown in the sidebar and a plain-English line about what it does and
// when you'd reach for it — GUI-framed, with none of the CLI synopsis or flags
// that the command reference (Catalog) carries.
type GUITool struct {
	Name string `json:"name"`
	Desc string `json:"desc"`
}

// GUIGroup is a sidebar group of tools, in the same order the app shows them.
type GUIGroup struct {
	Name  string    `json:"name"`
	Tools []GUITool `json:"tools"`
}

// GUICatalog is the single source of truth for the in-app Tools overview page
// and the matching GUI reference PDF — the GUI counterpart to Catalog(), which
// serves the command line. The group names and order mirror the workbench
// sidebar; add a tool to the app and add its row here so both the overview and
// the PDF stay current.
func GUICatalog() []GUIGroup {
	return []GUIGroup{
		{Name: "Internet", Tools: []GUITool{
			{"Lookup", "Who owns an IP or ASN — network, organisation, country, abuse contacts and the prefixes they announce."},
			{"Runbook (audit bundles)", "Run a curated sequence of tools against one target and get a single combined report (also exportable as PDF)."},
			{"Traceroute (AS)", "Trace the path to a host, with the autonomous system behind each hop so you can see whose network it crosses."},
			{"BGP & looking glass", "Check how a prefix is routed on the global internet — origin AS, RPKI/IRR validity and whether it looks hijacked."},
			{"Remote ping", "Ping a target from RIPE Atlas probes around the world to see reachability and latency from somewhere that isn't your desk."},
			{"Remote traceroute", "Traceroute to a target from remote RIPE Atlas probes, to compare the path the internet takes from other vantage points."},
			{"TLS certificate", "Inspect a server's certificate — who issued it, when it expires, the chain, and whether anything is wrong."},
			{"Cert / CSR decoder", "Paste a PEM certificate or signing request and read its fields in plain text, no OpenSSL incantation required."},
			{"TLS cipher scan", "See which TLS versions and cipher suites a server accepts, graded, including post-quantum readiness."},
			{"SSH audit (quantum-safe)", "List an SSH server's key exchanges, ciphers and host keys, graded, and flag whether it's ready for quantum-safe algorithms."},
			{"Let's Encrypt cert", "Issue a free Let's Encrypt certificate over the ACME DNS-01 challenge."},
			{"HTTP check", "Fetch a URL and read the status, redirects and response headers — a quick 'is this site up and what does it return'."},
			{"HTTP/3 (QUIC) reachability", "Check whether a site is reachable over HTTP/3 (QUIC), and how it advertises it."},
			{"IKE / IPsec VPN probe", "Probe an IPsec VPN endpoint — which IKEv2 proposal it accepts and whether NAT traversal is offered."},
			{"WireGuard probe", "Send a WireGuard handshake to confirm a responder is listening behind a UDP port."},
			{"Service status", "Check the published status pages of the big cloud and SaaS providers for current incidents and outages."},
			{"Security headers", "Grade a website's security response headers (CSP, HSTS and friends) against what OWASP recommends."},
			{"Whois", "Look up domain registration and ownership details over RDAP."},
			{"Subdomain finder", "Enumerate a domain's subdomains from certificate-transparency logs — useful recon for what a domain exposes."},
			{"Blocklist check", "Check whether an IP or domain is listed on the major spam and reputation blocklists."},
			{"DNS audit", "Health-check a domain's DNS: SPF, DMARC, CAA, DNSSEC, MX and NS hygiene in one pass."},
			{"Zone transfer (AXFR)", "Test whether a nameserver will hand over its whole zone to anyone who asks — a classic misconfiguration."},
			{"Open resolver test", "Check whether a DNS server answers recursive queries for strangers, which makes it an amplification risk."},
			{"DNS propagation", "Query a record from resolvers around the world to watch a DNS change propagate."},
			{"Email auth (SPF/DKIM/DMARC)", "Check a domain's email-authentication records so you know whether its mail can be spoofed."},
			{"SMTP probe", "Connect to a mail server, read its banner and STARTTLS support, and confirm it's reachable."},
			{"SIP OPTIONS ping", "Send a SIP OPTIONS ping to confirm a VoIP PBX or endpoint is alive and responding."},
			{"STUN (public IP / NAT)", "Discover your public IP and what kind of NAT you're behind, the way WebRTC does."},
			{"Cert expiry tracker", "Track certificate expiry dates across a list of hosts so a renewal never surprises you."},
		}},
		{Name: "Addressing", Tools: []GUITool{
			{"Subnet calculator", "Work out network, broadcast, mask, host range and VLSM splits for IPv4 and IPv6 — the everyday subnetting maths."},
			{"IP space", "Keep a point-in-time inventory of your address plan — which CIDRs are allocated and what lives in them."},
		}},
		{Name: "Local / environment", Tools: []GUITool{
			{"NTP", "Query a time server and see its offset, so you can tell whether a clock you depend on is accurate."},
			{"DNS", "Resolve a name the way dig does — any record type, over plain DNS or the encrypted transports (DoH/DoT/DoQ)."},
			{"Interfaces", "List this machine's network interfaces with their addresses — your own local networking at a glance."},
			{"Command line", "Reference for every tool as a meridian-cli subcommand, so you can script, pipe or run it over SSH; install the CLI from here."},
			{"Tools overview", "This page — every tool in Meridian, grouped as in the sidebar, with a plain-English note on what each one does. Export it as a PDF."},
		}},
		{Name: "Local network", Tools: []GUITool{
			{"Ping", "Ping one host and watch the round-trip time — the simplest 'can I reach it and how fast'."},
			{"Ping monitor", "Ping a host continuously and chart the latency and loss over time while you watch."},
			{"TCP ping (port)", "'Ping' a specific TCP port by timing the connect — works where ICMP is blocked, and tells you the service is actually listening."},
			{"Connection quality", "Measure jitter, loss and latency to a target and score the line's fitness for real-time traffic like voice."},
			{"VoIP bandwidth", "Calculate the bandwidth a voice call needs for a given codec and settings."},
			{"MTR (live path)", "A live traceroute that keeps probing every hop, so you can see exactly where on the path the loss or latency appears."},
			{"Path MTU", "Discover the largest packet that reaches a destination without fragmenting — the usual suspect behind 'it works but big transfers hang'."},
			{"DHCP discover", "Send a DHCP discover on the local network and read the offer, to see what a client would be handed and by whom."},
			{"LLDP / CDP neighbor", "Listen for LLDP/CDP and learn which switch and port this machine is plugged into."},
			{"ARP scan", "Sweep the local subnet with ARP to list every device that's actually on the wire, with its MAC."},
			{"IPv6 neighbors (NDP)", "Discover IPv6 neighbours and router advertisements on the local link."},
			{"LAN reconcile (→ IP space)", "Compare what's live on the LAN (from ARP) against your IP-space inventory and surface the drift."},
			{"Service discovery (mDNS/SSDP)", "Find the devices and services advertising themselves on the LAN — printers, media boxes, smart-home gear — via mDNS/Bonjour and SSDP/UPnP."},
			{"RTP analyzer", "Analyse an RTP media stream for jitter and loss to diagnose choppy audio or video."},
			{"Packet summary", "Capture briefly on an interface and summarise what's flowing, without opening a full packet analyser."},
			{"Wi-Fi analyzer", "Survey the Wi-Fi around you — SSIDs, channels and signal strength — to spot congestion and weak coverage."},
			{"Ping sweep", "Sweep a range of addresses to find which hosts are alive."},
			{"Reverse-DNS sweep", "Resolve a range of addresses back to names, to put labels on what a scan found."},
			{"Port scanner", "Scan a host for open TCP ports and the services behind them."},
			{"Wake-on-LAN", "Send a magic packet to power a sleeping machine back on by its MAC address."},
			{"MAC vendor", "Look up the manufacturer behind a MAC address from its OUI."},
			{"Printer status", "Query a network printer for toner, paper, jams and queue state over IPP/SNMP, so you know why it isn't printing."},
			{"Meshtastic mesh", "Connect to a Meshtastic radio (USB or TCP) and see the mesh — nodes, signal, hops, battery and positions on a map."},
			{"Bluetooth LE survey", "Scan for nearby Bluetooth LE devices and beacons with their signal strength — what's broadcasting around you."},
			{"Connected peripherals", "Inventory what's plugged into USB — mice, keyboards, dongles, drives — with honest security call-outs for anything that looks off."},
			{"Link inspector", "Work out whether there's a hub or switch between this machine and the router, from link duplex, LLDP and a gateway probe — honest about its blind spots."},
			{"Local network diagnostic", "Diagnose why the LAN isn't working — gateway, DHCP, DNS, captive portals, host isolation and macOS Local Network Privacy."},
			{"LAN speed test", "Measure raw throughput to another machine on the LAN with an iperf-style test."},
		}},
		{Name: "SNMP & monitoring", Tools: []GUITool{
			{"SNMP interface monitor", "Poll a device's interfaces over SNMP and watch bandwidth and utilisation live."},
			{"Device health", "Read CPU, memory and uptime from a device over SNMP for a quick health check."},
			{"SNMP sweep", "Sweep a range for SNMP-speaking devices and read their system description."},
			{"MIB browser", "Walk a device's SNMP tree by OID to explore what it exposes."},
			{"Switch port mapper", "Ask a switch which port a given MAC or IP is on, from its forwarding table — find where a device is plugged in."},
		}},
		{Name: "Collectors", Tools: []GUITool{
			{"Syslog receiver", "Listen as a syslog server and watch incoming log messages from your devices in real time."},
			{"Trap receiver", "Receive SNMP traps so you can see the asynchronous alerts devices send."},
			{"NetFlow", "Collect NetFlow/IPFIX records and see the top talkers on your network."},
			{"sFlow", "Collect sampled sFlow records and see the top talkers on your network."},
			{"TFTP grab", "Run a brief TFTP server to catch a config a device pushes to you — handy for backups."},
		}},
		{Name: "Security", Tools: []GUITool{
			{"Netcat", "Make or listen for a raw TCP/UDP connection and grab a banner — the Swiss-army knife for poking at a port."},
			{"AD password audit", "Audit Active Directory password hashes (from an NTDS extract) for weak and cracked credentials."},
			{"Cisco passwords", "Decode Cisco type-7 and crackable type-5 passwords out of an IOS config."},
			{"Hash identifier", "Identify what kind of hash a string is, so you know what you're dealing with."},
			{"WWN decoder", "Decode a Fibre Channel World Wide Name into its vendor and format — storage/SAN work."},
			{"JWT decoder", "Decode a JSON Web Token and read its claims without trusting a website with it."},
			{"bcrypt / htpasswd", "Generate bcrypt or htpasswd entries for Apache/nginx basic auth."},
			{"TOTP (2FA) code", "Generate a TOTP two-factor code from a shared secret."},
			{"Encode / decode", "Convert between base64, hex, URL-encoding, ROT13 and plain text."},
			{"SNMP community audit", "Test a device for default and weak SNMP community strings."},
			{"Breach lookup", "Check whether an address or password appears in known breaches (Have I Been Pwned)."},
			{"CVE lookup", "Look up a CVE for its details and severity."},
			{"Threat intel (C2)", "Check an indicator against threat-intel sources for known command-and-control and malicious reputation."},
		}},
		{Name: "Config insight", Tools: []GUITool{
			{"Firewall analyzer", "Read a firewall config (ASA, Palo Alto, Fortinet and more) and surface risky or redundant rules."},
			{"FC zoning analyzer", "Analyse a Fibre Channel zoning export (Brocade/Cisco MDS) for the zones, aliases and single-initiator hygiene."},
			{"Web server analyzer", "Review an nginx or Apache config for TLS, headers and common misconfigurations."},
			{"Terraform exposure", "Scan Terraform/HCL for things exposed to the internet — open security groups, public buckets and the like."},
			{"Config diff", "Compare two configs and see exactly what changed."},
		}},
	}
}

// guiMeta returns the title, tool count and generation line for the GUI
// reference, mirroring docsMeta so the overview page and its PDF agree.
func guiMeta(groups []GUIGroup) (title string, tools int) {
	for _, g := range groups {
		tools += len(g.Tools)
	}
	return "Meridian — Tools Overview", tools
}
