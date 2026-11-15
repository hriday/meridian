package cli

import (
	"fmt"
	"strings"
)

// CommandDoc documents one CLI command for both the text help and the GUI
// reference page.
type CommandDoc struct {
	Name string `json:"name"`
	Args string `json:"args"` // the argument/flag synopsis, without the command name
	Desc string `json:"desc"`
}

// CommandGroup is a set of related commands, named to match the desktop app's
// tool groups.
type CommandGroup struct {
	Name     string       `json:"name"`
	Note     string       `json:"note,omitempty"`
	Commands []CommandDoc `json:"commands"`
}

// Catalog is the single source of truth for what commands exist. Both the
// grouped `help` text and the GUI's command-line reference render from it, so
// they can't drift apart.
func Catalog() []CommandGroup {
	return []CommandGroup{
		{Name: "Internet", Commands: []CommandDoc{
			{"lookup", "<ip|asn> [--json]", "IP / ASN intelligence"},
			{"trace", "<host> [--6] [--json]", "traceroute with AS lookups"},
		{"tcptrace", "<host> <port> [--json]", "TCP-SYN traceroute (through firewalls that drop UDP/ICMP; no sudo on macOS)"},
			{"bgp", "<prefix|ip> [--json]", "BGP looking glass"},
			{"tls", "<host[:port]> [--json]", "inspect a server's certificate"},
			{"certinfo", "[<file>] [--json]", "decode a PEM certificate / CSR (stdin ok)"},
			{"cipherscan", "<host[:port]> [--json]", "TLS versions + cipher suites, graded"},
			{"sshaudit", "<host[:port]> [--json]", "SSH algorithms + host keys + post-quantum readiness"},
			{"ike", "<host> [--json]", "IKE/IPsec VPN probe — IKEv2 accepted proposal + NAT-T (UDP/500)"},
			{"wg", "<host> [--port 51820] [--pubkey <b64>] [--json]", "WireGuard probe — handshake initiation; cookie/handshake reply confirms a responder"},
			{"http", "<url> [--json]", "HTTP(S) check"},
			{"http3", "<host> [--port 443] [--json]", "HTTP/3 (QUIC) reachability — handshake + Alt-Svc"},
			{"sechdr", "<url> [--json]", "security headers (OWASP-mapped)"},
			{"whois", "<domain|ip|asn> [--json]", "whois"},
			{"subdomains", "<domain> [--json]", "subdomains via Certificate Transparency"},
			{"blocklist", "<ip> [--resolver <ip>] [--json]", "DNS blocklist (DNSBL) check"},
			{"dnsaudit", "<domain> [--resolver <ip>] [--json]", "DNS posture audit (SPF/DMARC/DNSSEC)"},
			{"dnspropagation", "<name> [--type A] [--json]", "check a record across global resolvers"},
			{"emailauth", "<domain> [--selectors a,b,c] [--json]", "SPF / DKIM / DMARC"},
			{"smtp", "<host|domain> [--mx] [--tls] [--port N] [--json]", "SMTP banner / STARTTLS / cert probe"},
			{"sip", "<host> [--transport udp|tcp|tls] [--port N] [--json]", "SIP OPTIONS ping (is VoIP signaling up?)"},
			{"stun", "[<server>] [--json]", "public IP + NAT type (STUN)"},
			{"certexpiry", "<host[:port]>... [--json]", "bulk TLS cert days-to-expiry"},
			{"certissue", "<domain>... [--email x] [--staging] [--out <dir>]", "get a Let's Encrypt cert (DNS-01)"},
			{"status", "<service|status-url> [--json]", "service outage check (Statuspage)"},
		}},
		{Name: "Addressing", Commands: []CommandDoc{
			{"subnet", "<cidr> [--split <prefixlen>] [--json]", "subnet calculator"},
			{"classify", "<value> [--json]", "classify an address or value"},
			{"ipspace", "list|export|import|reconcile [--format csv|netbox|terraform] [--file <p>] [--passphrase <pw>]", "IP-space inventory — import/export CSV · NetBox YAML · Terraform CIDR; reconcile vs a live ARP sweep"},
		}},
		{Name: "Local / environment", Commands: []CommandDoc{
			{"ntp", "<server> [--json]", "query an NTP server"},
			{"dns", "<name> [--type A] [--server <ip>] [--transport udp|tcp|dot|doh|doq] [--json]", "DNS lookup (incl. DNS-over-TLS / HTTPS / QUIC)"},
			{"interfaces", "[--json]", "local network interfaces"},
		}},
		{Name: "Local network", Commands: []CommandDoc{
			{"ping", "<host> [--count 5] [--interval 500ms] [--json]", "ping"},
			{"pingmon", "<host> [--interval 1s] [--json]", "live ping (Ctrl-C to stop)"},
			{"tcpping", "<host> <port> [--count 5] [--interval 500ms] [--json]", "TCP connect latency + loss (ping a port; works when ICMP is blocked)"},
			{"quality", "<host> [--count 25] [--json]", "connection quality — jitter, loss, MOS"},
			{"codecbw", "<codec> [--ptime 20] [--calls N] [--json]", "VoIP codec call bandwidth"},
			{"mtr", "<host> [--interval 1s] [--6] [--json]", "live path + per-hop loss (Ctrl-C to stop)"},
			{"pathmtu", "<host> [--json]", "discover the path MTU (find MTU bottlenecks)"},
			{"dhcpdiscover", "[--wait 5s] [--json]", "find DHCP servers / rogues on the LAN"},
			{"lldp", "[--wait 35s] [--iface en0] [--json]", "LLDP/CDP neighbor — what switch/port (raw capture: sudo or BPF group)"},
			{"arpscan", "[--wait 2s] [--max 1024] [--json]", "ARP sweep the LAN — live MAC/IP + vendor"},
			{"discover", "[--seconds 8] [--json]", "find LAN devices via mDNS/Bonjour + SSDP/UPnP"},
			{"rtp", "[--seconds 10] [--iface en0] [--json]", "analyze RTP streams — jitter/loss (raw capture: sudo or BPF group)"},
			{"pktscan", "[--iface en0] [--seconds 8] [--top 10] [--json]", "live packet summary — top talkers, TCP resets/retransmits, DNS errors, TLS SNI (raw capture: sudo or BPF group)"},
			{"wifi", "[--json]", "Wi-Fi/RF environment — channel, signal, neighbors, co-channel (macOS / Linux nmcli / Windows netsh)"},
			{"pingsweep", "<cidr> [--ports 22,80,443] [--json]", "ping / TCP sweep"},
			{"portscan", "<host> <ports|--common> [--json]", "port scan"},
			{"ptrsweep", "<cidr> [--concurrency 64] [--json]", "reverse-DNS sweep"},
			{"wol", "<mac> [--bcast 255.255.255.255] [--port 9]", "wake-on-LAN"},
			{"macvendor", "<mac|oui> [--json]", "MAC vendor lookup"},
			{"speedtest", "<iperf3-host> [--dir download|upload|bidir] [--json]", "throughput test"},
		}},
		{Name: "SNMP & monitoring", Commands: []CommandDoc{
			{"snmpsweep", "<cidr> [--community public] [--json]", "find SNMP responders"},
			{"devhealth", "<device> [--community public] [--json]", "device health snapshot"},
			{"swport", "<device> [--community public] [--json]", "MAC-to-port table"},
			{"mib", "<device> [--root <oid>] [--max 500] [--json]", "SNMP walk"},
		}},
		{Name: "Collectors", Note: "receivers; Ctrl-C to stop. Ports <1024 need sudo or a high --port.", Commands: []CommandDoc{
			{"syslog", "[--port 514] [--json]", "syslog receiver"},
			{"traps", "[--port 162] [--json]", "SNMP trap receiver"},
			{"netflow", "[--port 2055] [--json]", "NetFlow v5 / v9 / IPFIX receiver"},
			{"sflow", "[--port 6343] [--json]", "sFlow v5 receiver — sampled flows → top talkers"},
			{"tftp", "[--port 69] [--out <dir>] [--json]", "TFTP receiver — saves grabbed files"},
		}},
		{Name: "Security", Commands: []CommandDoc{
			{"nc", "<host> <port> [--udp] [--tls] [--listen]", "netcat"},
			{"adaudit", "--dump <file> [--wordlist <path>] [--json]", "AD password audit"},
			{"ciscopw", "[<config-file>] [--decode <type7>] [--json]", "Cisco password analysis"},
			{"hashid", "<hash> [--json]", "identify a hash type"},
			{"jwt", "<token> [--secret s | --keyfile f] [--json]", "decode + verify a JWT (local, private)"},
			{"htpasswd", "[<user>] [--cost 10] [--check <hash>]  (password on stdin)", "bcrypt hash / verify (htpasswd)"},
			{"totp", "<base32-secret | otpauth-uri> [--json]", "current 2FA (TOTP) code"},
			{"transcode", "<op> [<text>]   (base64/hex/url/json, stdin ok)", "encode / decode text"},
			{"snmpbrute", "<device> --authorized [--communities a,b,c]", "SNMP community audit"},
			{"breaches", "<email> [--json]   (or --latest)", "breach lookup"},
			{"cve", "<id|keyword> [--json]", "CVE lookup (NVD)"},
			{"threatintel", "<ip>... [--json]", "botnet C2 check (abuse.ch)"},
		}},
		{Name: "Scripting", Note: "pipe a newline-delimited target list on stdin.", Commands: []CommandDoc{
			{"completion", "bash|zsh|fish|powershell", "print a shell tab-completion script"},
			{"batch", "<command> [flags]   (targets on stdin)", "run a single-target command over many targets — e.g. `cat hosts.txt | meridian-cli batch cipherscan --json` (NDJSON)"},
			{"runbook", "<profile|file.yaml> <target> [--format md|json|html] [--out f] | list", "composite audit → one report: a built-in profile (org-recon, audit-domain, triage-host) or your own YAML/TOML/JSON"},
		}},
		{Name: "Config insight", Commands: []CommandDoc{
			{"fwanalyze", "[<config-file>] [--vendor auto] [--json]", "firewall config analysis — FortiGate/ASA/IOS/NX-OS/SRX/Junos/PAN-OS/Arista/VyOS/pfSense (stdin ok)"},
			{"webanalyze", "[<file>] [--server nginx] [--json]", "web-server config analysis (stdin ok)"},
			{"tfexposure", "[<file>] [--source hcl|plan] [--json]", "Terraform exposure scan — AWS/GCP/Azure (stdin ok)"},
			{"cfgdiff", "<old-file> <new-file> [--json]", "section-aware config diff"},
		}},
	}
}

// renderHelp builds the grouped `help` text from the catalog.
func renderHelp() string {
	var b strings.Builder
	b.WriteString("meridian-cli — Meridian's network toolkit on the command line\n\n")
	b.WriteString("Every command is a thin wrapper over the same engine the desktop app uses.\n")
	b.WriteString("Most accept --json. Streaming commands run until Ctrl-C.\n")
	for _, g := range Catalog() {
		b.WriteString("\n")
		b.WriteString(g.Name)
		if g.Note != "" {
			b.WriteString(" (" + g.Note + ")")
		}
		b.WriteString("\n")
		for _, c := range g.Commands {
			left := c.Name + " " + c.Args
			b.WriteString(fmt.Sprintf("  %-52s %s\n", left, c.Desc))
		}
	}
	return b.String()
}
