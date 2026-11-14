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
			{"bgp", "<prefix|ip> [--json]", "BGP looking glass"},
			{"tls", "<host[:port]> [--json]", "inspect a server's certificate"},
			{"cipherscan", "<host[:port]> [--json]", "TLS versions + cipher suites, graded"},
			{"http", "<url> [--json]", "HTTP(S) check"},
			{"sechdr", "<url> [--json]", "security headers (OWASP-mapped)"},
			{"whois", "<domain|ip|asn> [--json]", "whois"},
			{"subdomains", "<domain> [--json]", "subdomains via Certificate Transparency"},
			{"blocklist", "<ip> [--resolver <ip>] [--json]", "DNS blocklist (DNSBL) check"},
			{"dnsaudit", "<domain> [--resolver <ip>] [--json]", "DNS posture audit (SPF/DMARC/DNSSEC)"},
			{"emailauth", "<domain> [--selectors a,b,c] [--json]", "SPF / DKIM / DMARC"},
			{"certexpiry", "<host[:port]>... [--json]", "bulk TLS cert days-to-expiry"},
			{"certissue", "<domain>... [--email x] [--staging] [--out <dir>]", "get a Let's Encrypt cert (DNS-01)"},
			{"status", "<service|status-url> [--json]", "service outage check (Statuspage)"},
		}},
		{Name: "Addressing", Commands: []CommandDoc{
			{"subnet", "<cidr> [--split <prefixlen>] [--json]", "subnet calculator"},
			{"classify", "<value> [--json]", "classify an address or value"},
			{"ipspace", "list|export [--file <path>] [--passphrase <pw>]", "IP-space inventory"},
		}},
		{Name: "Local / environment", Commands: []CommandDoc{
			{"ntp", "<server> [--json]", "query an NTP server"},
			{"dns", "<name> [--type A] [--server <ip>] [--json]", "DNS lookup"},
			{"interfaces", "[--json]", "local network interfaces"},
		}},
		{Name: "Local network", Commands: []CommandDoc{
			{"ping", "<host> [--count 5] [--interval 500ms] [--json]", "ping"},
			{"pingmon", "<host> [--interval 1s] [--json]", "live ping (Ctrl-C to stop)"},
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
			{"netflow", "[--port 2055] [--json]", "NetFlow v5 receiver"},
			{"tftp", "[--port 69] [--out <dir>] [--json]", "TFTP receiver — saves grabbed files"},
		}},
		{Name: "Security", Commands: []CommandDoc{
			{"nc", "<host> <port> [--udp] [--tls] [--listen]", "netcat"},
			{"adaudit", "--dump <file> [--wordlist <path>] [--json]", "AD password audit"},
			{"ciscopw", "[<config-file>] [--decode <type7>] [--json]", "Cisco password analysis"},
			{"hashid", "<hash> [--json]", "identify a hash type"},
			{"snmpbrute", "<device> --authorized [--communities a,b,c]", "SNMP community audit"},
			{"breaches", "<email> [--json]   (or --latest)", "breach lookup"},
			{"cve", "<id|keyword> [--json]", "CVE lookup (NVD)"},
			{"threatintel", "<ip>... [--json]", "botnet C2 check (abuse.ch)"},
		}},
		{Name: "Config insight", Commands: []CommandDoc{
			{"fwanalyze", "[<config-file>] [--json]", "firewall config analysis (stdin ok)"},
			{"webanalyze", "[<file>] [--server nginx] [--json]", "web-server config analysis (stdin ok)"},
			{"tfexposure", "[<file>] [--source hcl|plan] [--json]", "Terraform exposure scan (stdin ok)"},
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
