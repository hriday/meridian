<script lang="ts">
  import SubnetCalc from "./tools/SubnetCalc.svelte";
  import Lookup from "./tools/Lookup.svelte";
  import Runbook from "./tools/Runbook.svelte";
  import Traceroute from "./tools/Traceroute.svelte";
  import Bgp from "./tools/Bgp.svelte";
  import AtlasPing from "./tools/AtlasPing.svelte";
  import AtlasTrace from "./tools/AtlasTrace.svelte";
  import FirewallAnalyzer from "./tools/FirewallAnalyzer.svelte";
  import FczoneAnalyzer from "./tools/FczoneAnalyzer.svelte";
  import WebServerAnalyzer from "./tools/WebServerAnalyzer.svelte";
  import TerraformExposure from "./tools/TerraformExposure.svelte";
  import ConfigDiff from "./tools/ConfigDiff.svelte";
  import Tls from "./tools/Tls.svelte";
  import CertInfo from "./tools/CertInfo.svelte";
  import HttpCheck from "./tools/HttpCheck.svelte";
  import Http3 from "./tools/Http3.svelte";
  import IkeProbe from "./tools/IkeProbe.svelte";
  import WgProbe from "./tools/WgProbe.svelte";
  import Whois from "./tools/Whois.svelte";
  import Blocklist from "./tools/Blocklist.svelte";
  import DnsAudit from "./tools/DnsAudit.svelte";
  import AxfrTest from "./tools/AxfrTest.svelte";
  import OpenResolver from "./tools/OpenResolver.svelte";
  import DnsPropagation from "./tools/DnsPropagation.svelte";
  import Netcat from "./tools/Netcat.svelte";
  import AdAudit from "./tools/AdAudit.svelte";
  import CiscoPasswords from "./tools/CiscoPasswords.svelte";
  import SnmpBrute from "./tools/SnmpBrute.svelte";
  import BreachLookup from "./tools/BreachLookup.svelte";
  import CveLookup from "./tools/CveLookup.svelte";
  import HashId from "./tools/HashId.svelte";
  import WwnDecoder from "./tools/WwnDecoder.svelte";
  import JwtInspect from "./tools/JwtInspect.svelte";
  import HtpasswdTool from "./tools/HtpasswdTool.svelte";
  import TotpTool from "./tools/TotpTool.svelte";
  import Transcode from "./tools/Transcode.svelte";
  import ThreatIntel from "./tools/ThreatIntel.svelte";
  import CertExpiry from "./tools/CertExpiry.svelte";
  import EmailAuth from "./tools/EmailAuth.svelte";
  import SmtpProbe from "./tools/SmtpProbe.svelte";
  import SipProbe from "./tools/SipProbe.svelte";
  import StunTest from "./tools/StunTest.svelte";
  import CipherScan from "./tools/CipherScan.svelte";
  import SshAudit from "./tools/SshAudit.svelte";
  import CertIssue from "./tools/CertIssue.svelte";
  import ServiceStatus from "./tools/ServiceStatus.svelte";
  import PtrSweep from "./tools/PtrSweep.svelte";
  import CommandLine from "./tools/CommandLine.svelte";
  import SubdomainFinder from "./tools/SubdomainFinder.svelte";
  import SecurityHeaders from "./tools/SecurityHeaders.svelte";
  import Ntp from "./tools/Ntp.svelte";
  import Dns from "./tools/Dns.svelte";
  import Interfaces from "./tools/Interfaces.svelte";
  import Ping from "./tools/Ping.svelte";
  import PingMonitor from "./tools/PingMonitor.svelte";
  import TcpPing from "./tools/TcpPing.svelte";
  import Discover from "./tools/Discover.svelte";
  import QualityMonitor from "./tools/QualityMonitor.svelte";
  import VoipCalc from "./tools/VoipCalc.svelte";
  import MtrMonitor from "./tools/MtrMonitor.svelte";
  import PathMtu from "./tools/PathMtu.svelte";
  import DhcpDiscover from "./tools/DhcpDiscover.svelte";
  import LldpDiscover from "./tools/LldpDiscover.svelte";
  import ArpScan from "./tools/ArpScan.svelte";
  import Ndp from "./tools/Ndp.svelte";
  import Reconcile from "./tools/Reconcile.svelte";
  import RtpAnalyzer from "./tools/RtpAnalyzer.svelte";
  import PktScan from "./tools/PktScan.svelte";
  import Wifi from "./tools/Wifi.svelte";
  import SnmpMonitor from "./tools/SnmpMonitor.svelte";
  import DeviceHealth from "./tools/DeviceHealth.svelte";
  import SnmpSweep from "./tools/SnmpSweep.svelte";
  import MibBrowser from "./tools/MibBrowser.svelte";
  import SwitchPortMapper from "./tools/SwitchPortMapper.svelte";
  import SyslogReceiver from "./tools/SyslogReceiver.svelte";
  import TrapReceiver from "./tools/TrapReceiver.svelte";
  import NetflowCollector from "./tools/NetflowCollector.svelte";
  import SflowCollector from "./tools/SflowCollector.svelte";
  import TftpReceiver from "./tools/TftpReceiver.svelte";
  import PingSweep from "./tools/PingSweep.svelte";
  import SpeedTest from "./tools/SpeedTest.svelte";
  import PortScan from "./tools/PortScan.svelte";
  import WakeOnLan from "./tools/WakeOnLan.svelte";
  import MacVendor from "./tools/MacVendor.svelte";
  import PrinterStatus from "./tools/PrinterStatus.svelte";
  import Meshtastic from "./tools/Meshtastic.svelte";
  import NetDiagnostic from "./tools/NetDiagnostic.svelte";
  import IpSpace from "./tools/IpSpace.svelte";
  import { themePref, cycleTheme, themeIcon, themeLabel } from "./theme";
  import Settings from "./Settings.svelte";
  import IpChip from "./IpChip.svelte";
  import { requestedTool } from "./nav";
  import CommandPalette from "./CommandPalette.svelte";
  import { AppVersion, CheckForUpdate, CLIInfo, InstallCLI } from "../../wailsjs/go/app/App.js";
  import { BrowserOpenURL } from "../../wailsjs/runtime/runtime.js";

  type Tool = { id: string; name: string; group: string; ready: boolean; kw?: string };
  const tools: Tool[] = [
    { id: "lookup", name: "Lookup", group: "Internet", ready: true, kw: "ip asn rdap whois geoip bgp origin prefix cidr organization abuse peeringdb ripestat intelligence autonomous system" },
    { id: "runbook", name: "Runbook (audit bundles)", group: "Internet", ready: true, kw: "audit bundle report pdf org-recon recon profile recipe workflow" },
    { id: "trace", name: "Traceroute (AS)", group: "Internet", ready: true, kw: "traceroute tracert path hops as-path autonomous system" },
    { id: "bgp", name: "BGP & looking glass", group: "Internet", ready: true, kw: "rpki irr roa looking glass route origin as-path prefix hijack ripestat routeviews validation announced" },
    { id: "atlasping", name: "Remote ping", group: "Internet", ready: true, kw: "ripe atlas remote ping probe vantage point" },
    { id: "atlastrace", name: "Remote traceroute", group: "Internet", ready: true, kw: "ripe atlas remote traceroute probe vantage point" },
    { id: "tls", name: "TLS certificate", group: "Internet", ready: true, kw: "ssl certificate x509 cert chain expiry https" },
    { id: "certinfo", name: "Cert / CSR decoder", group: "Internet", ready: true, kw: "csr x509 decoder pem der certificate signing request parse" },
    { id: "cipherscan", name: "TLS cipher scan", group: "Internet", ready: true, kw: "ssl tls ciphers protocols suites pqc post-quantum harvest-now grade sslyze" },
    { id: "sshaudit", name: "SSH audit (quantum-safe)", group: "Internet", ready: true, kw: "ssh kex mac ciphers host-key post-quantum pqc quantum harvest-now sntrup mlkem openssh grade" },
    { id: "certissue", name: "Let's Encrypt cert", group: "Internet", ready: true, kw: "lets encrypt acme dns-01 certificate issue free" },
    { id: "http", name: "HTTP check", group: "Internet", ready: true, kw: "http headers status redirect url web response" },
    { id: "http3", name: "HTTP/3 (QUIC) reachability", group: "Internet", ready: true, kw: "quic h3 alt-svc udp 443 reachability https svcb" },
    { id: "ike", name: "IKE / IPsec VPN probe", group: "Internet", ready: true, kw: "ipsec vpn ikev2 isakmp phase1 nat-t" },
    { id: "wg", name: "WireGuard probe", group: "Internet", ready: true, kw: "wireguard vpn handshake udp" },
    { id: "svcstatus", name: "Service status", group: "Internet", ready: true, kw: "statuspage outage downdetector incident uptime apple azure google cloudflare github service health" },
    { id: "sechdr", name: "Security headers", group: "Internet", ready: true, kw: "security headers csp hsts x-frame-options referrer owasp" },
    { id: "whois", name: "Whois", group: "Internet", ready: true, kw: "rdap domain registrar registration ownership" },
    { id: "subdomains", name: "Subdomain finder", group: "Internet", ready: true, kw: "crt.sh certificate transparency ct subdomain enumeration recon" },
    { id: "blocklist", name: "Blocklist check", group: "Internet", ready: true, kw: "dnsbl rbl spamhaus spamcop reputation blacklist spam" },
    { id: "dnsaudit", name: "DNS audit", group: "Internet", ready: true, kw: "spf dmarc caa dnssec mx ns zone hygiene dns health" },
    { id: "axfr", name: "Zone transfer (AXFR)", group: "Internet", ready: true, kw: "zone transfer axfr nameserver dns exposure" },
    { id: "openresolver", name: "Open resolver test", group: "Internet", ready: true, kw: "open resolver dns amplification recursion" },
    { id: "dnsprop", name: "DNS propagation", group: "Internet", ready: true, kw: "dns propagation resolvers global anycast" },
    { id: "emailauth", name: "Email auth (SPF/DKIM/DMARC)", group: "Internet", ready: true, kw: "spf dkim dmarc email spoofing deliverability dns txt" },
    { id: "smtp", name: "SMTP probe", group: "Internet", ready: true, kw: "mail mx starttls email server port 25 587" },
    { id: "sip", name: "SIP OPTIONS ping", group: "Internet", ready: true, kw: "voip sip options ping pbx telephony" },
    { id: "stun", name: "STUN (public IP / NAT)", group: "Internet", ready: true, kw: "nat public ip webrtc stun binding" },
    { id: "subnet", name: "Subnet calculator", group: "Addressing", ready: true, kw: "cidr netmask subnet calculator vlsm supernet ipv4 ipv6 wildcard" },
    { id: "ipspace", name: "IP space", group: "Addressing", ready: true, kw: "ipam netbox inventory address plan cidr allocation" },
    { id: "ntp", name: "NTP", group: "Local / environment", ready: true, kw: "time clock sntp server offset" },
    { id: "dns", name: "DNS", group: "Local / environment", ready: true, kw: "dig nslookup resolver query a aaaa mx txt doh dot doq" },
    { id: "ifaces", name: "Interfaces", group: "Local / environment", ready: true, kw: "interfaces nic adapters ip address local ifconfig" },
    { id: "cli", name: "Command line", group: "Local / environment", ready: true, kw: "command line shell terminal commands install cli" },
    { id: "ping1", name: "Ping", group: "Local network", ready: true, kw: "icmp ping latency rtt reachability" },
    { id: "pingmon", name: "Ping monitor", group: "Local network", ready: true, kw: "ping monitor latency uptime chart" },
    { id: "tcpping", name: "TCP ping (port)", group: "Local network", ready: true, kw: "tcp port ping connect syn latency" },
    { id: "quality", name: "Connection quality", group: "Local network", ready: true, kw: "jitter loss latency connection quality mos" },
    { id: "codecbw", name: "VoIP bandwidth", group: "Local network", ready: true, kw: "voip codec bandwidth g711 opus calculator" },
    { id: "mtr", name: "MTR (live path)", group: "Local network", ready: true, kw: "mtr traceroute path loss live hops" },
    { id: "pathmtu", name: "Path MTU", group: "Local network", ready: true, kw: "mtu path pmtud fragmentation discovery" },
    { id: "dhcp", name: "DHCP discover", group: "Local network", ready: true, kw: "dhcp discover lease bootp offer" },
    { id: "lldp", name: "LLDP / CDP neighbor", group: "Local network", ready: true, kw: "lldp cdp neighbor switch port discovery" },
    { id: "arpscan", name: "ARP scan", group: "Local network", ready: true, kw: "arp scan mac neighbor lan sweep" },
    { id: "ndp", name: "IPv6 neighbors (NDP)", group: "Local network", ready: true, kw: "ipv6 neighbor ndp router advertisement ra slaac" },
    { id: "reconcile", name: "LAN reconcile (→ IP space)", group: "Local network", ready: true, kw: "arp ipam reconcile inventory drift" },
    { id: "discover", name: "Service discovery (mDNS/SSDP)", group: "Local network", ready: true, kw: "mdns bonjour ssdp upnp zeroconf service discovery avahi" },
    { id: "rtp", name: "RTP analyzer", group: "Local network", ready: true, kw: "rtp voip jitter media stream analyze" },
    { id: "pktscan", name: "Packet summary", group: "Local network", ready: true, kw: "packet capture pcap sniff summary" },
    { id: "wifi", name: "Wi-Fi analyzer", group: "Local network", ready: true, kw: "wifi wireless ssid rssi channel signal 802.11" },
    { id: "ifmon", name: "SNMP interface monitor", group: "SNMP & monitoring", ready: true, kw: "snmp interface monitor bandwidth throughput utilization" },
    { id: "devhealth", name: "Device health", group: "SNMP & monitoring", ready: true, kw: "snmp cpu memory uptime device health" },
    { id: "snmpsweep", name: "SNMP sweep", group: "SNMP & monitoring", ready: true, kw: "snmp sweep discovery sysdescr" },
    { id: "mib", name: "MIB browser", group: "SNMP & monitoring", ready: true, kw: "mib oid snmp browser walk" },
    { id: "swport", name: "Switch port mapper", group: "SNMP & monitoring", ready: true, kw: "switch port mac fdb cam mapper find" },
    { id: "syslog", name: "Syslog receiver", group: "Collectors", ready: true, kw: "syslog receiver log 514 collector" },
    { id: "traps", name: "Trap receiver", group: "Collectors", ready: true, kw: "snmp trap receiver 162 collector" },
    { id: "netflow", name: "NetFlow", group: "Collectors", ready: true, kw: "netflow ipfix flow top talkers collector" },
    { id: "sflow", name: "sFlow", group: "Collectors", ready: true, kw: "sflow sampled flow top talkers collector" },
    { id: "tftp", name: "TFTP grab", group: "Collectors", ready: true, kw: "tftp config grab backup receiver" },
    { id: "ping", name: "Ping sweep", group: "Local network", ready: true, kw: "ping sweep host discovery alive" },
    { id: "ptrsweep", name: "Reverse-DNS sweep", group: "Local network", ready: true, kw: "reverse dns ptr sweep rdns" },
    { id: "ports", name: "Port scanner", group: "Local network", ready: true, kw: "port scan tcp open services nmap" },
    { id: "wol", name: "Wake-on-LAN", group: "Local network", ready: true, kw: "wake on lan magic packet wol power" },
    { id: "macvendor", name: "MAC vendor", group: "Local network", ready: true, kw: "mac oui vendor manufacturer lookup" },
    { id: "printer", name: "Printer status", group: "Local network", ready: true, kw: "printer ipp cups jetdirect 9100 631 lpd toner ink supplies paper jam print queue page out of paper offline" },
    { id: "mesh", name: "Meshtastic mesh", group: "Local network", ready: true, kw: "meshtastic lora mesh radio node nodes snr rssi hops channel utilization airtime neighbors neighbours serial usb tcp 4403 longfast off-grid heltec t-beam rak" },
    { id: "netdiag", name: "Local network diagnostic", group: "Local network", ready: true, kw: "diagnostic troubleshoot reachability isolation hotspot guest wifi firewall local network privacy gateway dhcp captive portal no route" },
    { id: "speedtest", name: "LAN speed test", group: "Local network", ready: true, kw: "iperf lan speed throughput bandwidth test" },
    { id: "nc", name: "Netcat", group: "Security", ready: true, kw: "netcat nc tcp udp connect listen banner" },
    { id: "adaudit", name: "AD password audit", group: "Security", ready: true, kw: "active directory ntds hash ntlm password audit dcsync crack" },
    { id: "ciscopw", name: "Cisco passwords", group: "Security", ready: true, kw: "cisco type7 type5 enable password decrypt ios" },
    { id: "hashid", name: "Hash identifier", group: "Security", ready: true, kw: "hash identify type md5 sha bcrypt ntlm" },
    { id: "wwn", name: "WWN decoder", group: "Security", ready: true, kw: "wwn wwpn wwnn fibre channel fiber channel fc san storage naa oui vendor hba world wide name worldwide zoning" },
    { id: "jwt", name: "JWT decoder", group: "Security", ready: true, kw: "jwt json web token decode claims jose" },
    { id: "htpasswd", name: "bcrypt / htpasswd", group: "Security", ready: true, kw: "bcrypt htpasswd password hash apache nginx basic auth" },
    { id: "totp", name: "TOTP (2FA) code", group: "Security", ready: true, kw: "totp 2fa otp authenticator mfa code" },
    { id: "transcode", name: "Encode / decode", group: "Security", ready: true, kw: "base64 hex url encode decode rot13 ascii" },
    { id: "snmpbrute", name: "SNMP community audit", group: "Security", ready: true, kw: "snmp community brute default public private audit" },
    { id: "breaches", name: "Breach lookup", group: "Security", ready: true, kw: "breach haveibeenpwned hibp password leak pwned" },
    { id: "cve", name: "CVE lookup", group: "Security", ready: true, kw: "cve vulnerability nvd advisory cpe" },
    { id: "threatintel", name: "Threat intel (C2)", group: "Security", ready: true, kw: "c2 command control ioc malware threat intel abuse" },
    { id: "certexpiry", name: "Cert expiry tracker", group: "Internet", ready: true, kw: "certificate expiry expiration tls ssl monitor renewal tracker" },
    { id: "fwanalyze", name: "Firewall analyzer", group: "Config insight", ready: true, kw: "firewall acl rules analyze cisco asa paloalto fortinet config" },
    { id: "fczone", name: "FC zoning analyzer", group: "Config insight", ready: true, kw: "fibre channel fiber channel fc san storage zoning zone zoneset cfgshow show zoneset wwn wwpn brocade cisco mds vsan alias fabric" },
    { id: "webanalyze", name: "Web server analyzer", group: "Config insight", ready: true, kw: "nginx apache web server config analyze tls headers" },
    { id: "tfexposure", name: "Terraform exposure", group: "Config insight", ready: true, kw: "terraform iac cloud exposure security group s3 aws azure gcp hcl" },
    { id: "cfgdiff", name: "Config diff", group: "Config insight", ready: true, kw: "config diff compare changes before after" },
  ];
  const groups = [...new Set(tools.map((t) => t.group))];

  // Per-viewer UI state, persisted best-effort (storage may be blocked).
  function load<T>(key: string, fallback: T): T {
    try {
      const v = localStorage.getItem(key);
      return v == null ? fallback : (JSON.parse(v) as T);
    } catch {
      return fallback;
    }
  }
  function save(key: string, val: unknown) {
    try {
      localStorage.setItem(key, JSON.stringify(val));
    } catch {
      /* ignore */
    }
  }

  let active = load<string>("meridian.activeTool", "lookup");
  if (!tools.find((t) => t.id === active && t.ready)) active = "lookup";
  let sideCollapsed = load<boolean>("meridian.sideCollapsed", false);
  let closed = load<Record<string, boolean>>("meridian.closedGroups", {});

  $: save("meridian.activeTool", active);
  $: save("meridian.sideCollapsed", sideCollapsed);
  $: save("meridian.closedGroups", closed);

  let searchEl: HTMLInputElement;
  let filter = "";
  let settingsOpen = false;

  // Version + an explicitly user-initiated update check (no background polling).
  let appVersion = "";
  AppVersion().then((v) => (appVersion = v)).catch(() => {});

  // Command-line tool: only surfaces when this build bundles meridian-cli
  // (i.e. a packaged .app), so it stays invisible in dev and unsupported OSes.
  let cli: any = { available: false, installed: false, onPath: true, supported: false, linkPath: "", path: "" };
  let cliBusy = false;
  let cliMsg = "";
  CLIInfo().then((c) => (cli = c)).catch(() => {});
  async function installCLI() {
    cliBusy = true;
    cliMsg = "";
    try {
      cli = await InstallCLI();
      cliMsg = cli.onPath
        ? `Installed. Run “meridian-cli” in any terminal.`
        : `Installed to ${cli.linkPath}, but that folder isn't on your PATH — add it to use “meridian-cli”.`;
    } catch (e: any) {
      cliMsg = String(e?.message ?? e);
    } finally {
      cliBusy = false;
    }
  }

  let upState: "idle" | "checking" | "latest" | "available" | "error" = "idle";
  let upLatest = "", upURL = "", upMsg = "", upNotes = "", upDownload = "", upOS = "", upAsset = "";
  let showUpdate = false;
  async function checkUpdates() {
    if (upState === "checking") return;
    upState = "checking";
    upMsg = "";
    try {
      const r = await CheckForUpdate();
      upLatest = r.latest; upURL = r.url; upNotes = r.notes || "";
      upDownload = r.downloadUrl || ""; upOS = r.osLabel || ""; upAsset = r.assetName || "";
      upState = r.newer ? "available" : "latest";
      if (r.newer) showUpdate = true; // surface it in-app rather than bouncing to GitHub
    } catch (e) {
      upState = "error";
      upMsg = String(e);
    }
  }

  function toggleGroup(g: string) {
    closed = { ...closed, [g]: !closed[g] };
  }

  // A tool pivot (e.g. from IP space → Subnet calc) switches the active tool.
  $: if ($requestedTool) active = $requestedTool.tool;

  const haystack = (t: Tool) => `${t.name} ${t.group} ${t.id} ${t.kw ?? ""}`.toLowerCase();
  // Every query term must appear somewhere in the tool's name/group/id/keywords,
  // so multi-word searches ("post quantum", "open port") narrow correctly.
  $: visible = (g: string) => {
    const terms = filter.trim().toLowerCase().split(/\s+/).filter(Boolean);
    return tools.filter((t) => {
      if (t.group !== g) return false;
      if (terms.length === 0) return true;
      const hay = haystack(t);
      return terms.every((term) => hay.includes(term));
    });
  };
  $: activeTool = tools.find((t) => t.id === active);
</script>

<CommandPalette {tools} />

<div class="wb">
  <header class="bar">
    <button class="icon" on:click={() => (sideCollapsed = !sideCollapsed)} title="Toggle sidebar" aria-label="Toggle sidebar">☰</button>
    <span class="brand">
      <svg class="mark" viewBox="0 0 64 64" width="19" height="19" fill="none"
        stroke="currentColor" stroke-width="3.5" stroke-linecap="round" aria-hidden="true">
        <circle cx="32" cy="32" r="22" />
        <ellipse cx="32" cy="32" rx="9" ry="22" />
        <line x1="10" y1="32" x2="54" y2="32" />
        <circle cx="32" cy="10" r="4.5" fill="currentColor" stroke="none" />
      </svg>
      Meridian
    </span>
    <IpChip />
    <input
      bind:this={searchEl}
      bind:value={filter}
      class="search mono"
      placeholder="Search tools…"
      aria-label="Search tools"
    />
    <kbd>⌘K</kbd>
    <button
      class="theme"
      on:click={cycleTheme}
      title={`Theme: ${themeLabel[$themePref]} — click to change`}
      aria-label={`Theme: ${themeLabel[$themePref]}`}
    >
      <span class="ticon">{themeIcon[$themePref]}</span>
      <span class="tlabel">{themeLabel[$themePref]}</span>
    </button>
    <button class="theme gear" on:click={() => (settingsOpen = true)} title="Settings" aria-label="Settings">
      <span class="ticon">⚙</span>
    </button>
  </header>

  <Settings open={settingsOpen} onClose={() => (settingsOpen = false)} />

  <div class="body">
    {#if !sideCollapsed}
      <nav class="side">
        <div class="navscroll">
          {#each groups as g}
            {#if visible(g).length}
              <button class="grp" on:click={() => toggleGroup(g)}>
                <span class="caret">{closed[g] && !filter ? "▸" : "▾"}</span>{g}
              </button>
              {#if !(closed[g] && !filter)}
                {#each visible(g) as t}
                  <button
                    class="item"
                    class:on={active === t.id}
                    class:disabled={!t.ready}
                    data-tool={t.id}
                    on:click={() => t.ready && (active = t.id)}
                    title={t.ready ? t.name : `${t.name} — coming soon`}
                  >
                    {t.name}{#if !t.ready}<span class="soon">soon</span>{/if}
                  </button>
                {/each}
              {/if}
            {/if}
          {/each}
        </div>
        <div class="sidefoot">
          <span class="ver mono">Meridian {appVersion || "…"}</span>
          <button class="upd link" on:click={() => BrowserOpenURL("https://github.com/hriday/meridian/blob/master/docs/guide.md")} title="Open the Meridian user guide in your browser">
            How to use Meridian ↗
          </button>
          {#if upState === "available"}
            <button class="upd avail" on:click={() => (showUpdate = true)} title="See what's new">
              {upLatest} available
            </button>
          {:else if upState === "latest"}
            <span class="upd ok">✓ Up to date</span>
          {:else if upState === "error"}
            <button class="upd link" on:click={checkUpdates} title={upMsg}>Check failed — retry</button>
          {:else}
            <button class="upd link" on:click={checkUpdates} disabled={upState === "checking"}>
              {upState === "checking" ? "Checking…" : "Check for updates"}
            </button>
          {/if}
          {#if cli.available}
            {#if cli.installed}
              <span class="upd ok" title={cli.path}>✓ CLI installed</span>
            {:else if cli.supported}
              <button class="upd link" on:click={installCLI} disabled={cliBusy} title="Add meridian-cli to your PATH">
                {cliBusy ? "Installing…" : "Install command-line tool"}
              </button>
            {/if}
            {#if cliMsg}<span class="climsg" title={cliMsg}>{cliMsg}</span>{/if}
          {/if}
        </div>
      </nav>
    {/if}

    <main class="main">
      <div class="tabs"><div class="tab on">{activeTool?.name ?? ""}</div></div>
      {#if active === "lookup"}<Lookup />{:else if active === "runbook"}<Runbook />{:else if active === "trace"}<Traceroute />{:else if active === "bgp"}<Bgp />{:else if active === "atlasping"}<AtlasPing />{:else if active === "atlastrace"}<AtlasTrace />{:else if active === "tls"}<Tls />{:else if active === "certinfo"}<CertInfo />{:else if active === "cipherscan"}<CipherScan />{:else if active === "sshaudit"}<SshAudit />{:else if active === "certissue"}<CertIssue />{:else if active === "http"}<HttpCheck />{:else if active === "http3"}<Http3 />{:else if active === "ike"}<IkeProbe />{:else if active === "wg"}<WgProbe />{:else if active === "svcstatus"}<ServiceStatus />{:else if active === "sechdr"}<SecurityHeaders />{:else if active === "whois"}<Whois />{:else if active === "subdomains"}<SubdomainFinder />{:else if active === "blocklist"}<Blocklist />{:else if active === "dnsaudit"}<DnsAudit />{:else if active === "axfr"}<AxfrTest />{:else if active === "openresolver"}<OpenResolver />{:else if active === "dnsprop"}<DnsPropagation />{:else if active === "emailauth"}<EmailAuth />{:else if active === "smtp"}<SmtpProbe />{:else if active === "sip"}<SipProbe />{:else if active === "stun"}<StunTest />{:else if active === "subnet"}<SubnetCalc />{:else if active === "ipspace"}<IpSpace />{:else if active === "ntp"}<Ntp />{:else if active === "dns"}<Dns />{:else if active === "ifaces"}<Interfaces />{:else if active === "cli"}<CommandLine />{:else if active === "ping1"}<Ping />{:else if active === "pingmon"}<PingMonitor />{:else if active === "tcpping"}<TcpPing />{:else if active === "quality"}<QualityMonitor />{:else if active === "codecbw"}<VoipCalc />{:else if active === "mtr"}<MtrMonitor />{:else if active === "pathmtu"}<PathMtu />{:else if active === "dhcp"}<DhcpDiscover />{:else if active === "lldp"}<LldpDiscover />{:else if active === "arpscan"}<ArpScan />{:else if active === "ndp"}<Ndp />{:else if active === "reconcile"}<Reconcile />{:else if active === "discover"}<Discover />{:else if active === "rtp"}<RtpAnalyzer />{:else if active === "pktscan"}<PktScan />{:else if active === "wifi"}<Wifi />{:else if active === "ifmon"}<SnmpMonitor />{:else if active === "devhealth"}<DeviceHealth />{:else if active === "snmpsweep"}<SnmpSweep />{:else if active === "mib"}<MibBrowser />{:else if active === "swport"}<SwitchPortMapper />{:else if active === "syslog"}<SyslogReceiver />{:else if active === "traps"}<TrapReceiver />{:else if active === "netflow"}<NetflowCollector />{:else if active === "sflow"}<SflowCollector />{:else if active === "tftp"}<TftpReceiver />{:else if active === "ping"}<PingSweep />{:else if active === "ptrsweep"}<PtrSweep />{:else if active === "ports"}<PortScan />{:else if active === "wol"}<WakeOnLan />{:else if active === "macvendor"}<MacVendor />{:else if active === "printer"}<PrinterStatus />{:else if active === "mesh"}<Meshtastic />{:else if active === "netdiag"}<NetDiagnostic />{:else if active === "speedtest"}<SpeedTest />{:else if active === "nc"}<Netcat />{:else if active === "adaudit"}<AdAudit />{:else if active === "ciscopw"}<CiscoPasswords />{:else if active === "hashid"}<HashId />{:else if active === "wwn"}<WwnDecoder />{:else if active === "jwt"}<JwtInspect />{:else if active === "htpasswd"}<HtpasswdTool />{:else if active === "totp"}<TotpTool />{:else if active === "transcode"}<Transcode />{:else if active === "snmpbrute"}<SnmpBrute />{:else if active === "breaches"}<BreachLookup />{:else if active === "cve"}<CveLookup />{:else if active === "threatintel"}<ThreatIntel />{:else if active === "certexpiry"}<CertExpiry />{:else if active === "fwanalyze"}<FirewallAnalyzer />{:else if active === "fczone"}<FczoneAnalyzer />{:else if active === "webanalyze"}<WebServerAnalyzer />{:else if active === "tfexposure"}<TerraformExposure />{:else if active === "cfgdiff"}<ConfigDiff />{/if}
    </main>
  </div>
</div>

{#if showUpdate}
  <div class="updov" on:click={(e) => { if (e.currentTarget === e.target) showUpdate = false; }}>
    <div class="updmodal">
      <div class="updhead">
        <div>
          <div class="updtitle">Meridian {upLatest} is available</div>
          <div class="updsub">You're on {appVersion}</div>
        </div>
        <button class="updx" on:click={() => (showUpdate = false)} aria-label="Close">✕</button>
      </div>
      {#if upNotes}
        <div class="updnotes"><div class="updlbl">What's new</div><pre>{upNotes}</pre></div>
      {/if}
      <div class="updactions">
        {#if upDownload}
          <button class="updbtn primary" on:click={() => BrowserOpenURL(upDownload)}>Download for {upOS}</button>
        {/if}
        <button class="updbtn" on:click={() => BrowserOpenURL(upURL)}>All releases ↗</button>
      </div>
      {#if upDownload}
        <p class="updnote">Downloads <span class="mono">{upAsset}</span>.{#if upOS === "Windows"} Windows isn't code-signed yet, so the first launch needs a one-time step: <b>More info → Run anyway</b> at the SmartScreen prompt.{/if}</p>
      {/if}
    </div>
  </div>
{/if}

<style>
  .updov { position: fixed; inset: 0; background: rgba(5,8,12,.55); display: flex; align-items: center; justify-content: center; padding: 24px; z-index: 200; }
  .updmodal { background: var(--panel, var(--card)); border: 1px solid var(--border); border-radius: 14px; width: 100%; max-width: 520px; box-shadow: 0 24px 60px rgba(0,0,0,.4); padding: 18px 20px 20px; }
  .updhead { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; margin-bottom: 12px; }
  .updtitle { font-size: 1.05rem; font-weight: 650; color: var(--text-strong, var(--text)); }
  .updsub { font-size: 0.8rem; color: var(--text-dim); margin-top: 2px; }
  .updx { background: none; border: 0; color: var(--text-dim); cursor: pointer; font-size: 1.05rem; line-height: 1; }
  .updnotes { background: var(--well); border: 1px solid var(--border-soft, var(--border)); border-radius: 9px; padding: 10px 12px; max-height: 300px; overflow: auto; margin-bottom: 14px; }
  .updlbl { font-size: 0.66rem; text-transform: uppercase; letter-spacing: .05em; color: var(--text-dim); margin-bottom: 6px; }
  .updnotes pre { margin: 0; white-space: pre-wrap; overflow-wrap: anywhere; font: inherit; font-size: 0.84rem; color: var(--text); line-height: 1.5; }
  .updactions { display: flex; gap: 9px; flex-wrap: wrap; }
  .updbtn { background: var(--card); border: 1px solid var(--border); color: var(--text); border-radius: 8px; padding: 8px 15px; font: inherit; font-size: 0.86rem; cursor: pointer; }
  .updbtn.primary { background: var(--accent); border-color: transparent; color: var(--on-accent, #fff); font-weight: 550; }
  .updbtn:hover { filter: brightness(1.05); }
  .updnote { font-size: 0.74rem; color: var(--text-faint, var(--text-dim)); margin: 12px 0 0; line-height: 1.5; }
  .updnote .mono { font-family: var(--font-mono, monospace); }
  .wb { display: flex; flex-direction: column; height: 100vh; }
  .bar {
    display: flex; align-items: center; gap: 10px; padding: 8px 12px;
    background: var(--bar); border-bottom: 1px solid var(--border);
  }
  .icon {
    background: none; border: none; color: var(--text-dim); font-size: 1rem; cursor: pointer;
    padding: 2px 6px; border-radius: 6px; line-height: 1;
  }
  .icon:hover { color: var(--text-strong); background: var(--hover); }
  .brand { display: flex; align-items: center; gap: 8px; font-weight: 600; letter-spacing: .02em; color: var(--text-strong); }
  .brand .mark { color: var(--accent); flex: none; }
  .search {
    flex: 1; background: var(--well); border: 1px solid var(--field-border); border-radius: 7px;
    padding: 6px 10px; color: var(--text); font-size: 0.82rem; outline: none;
  }
  .search:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  kbd {
    font: 11px ui-monospace, monospace; color: var(--text-dim);
    border: 1px solid var(--kbd-border); border-radius: 4px; padding: 1px 6px;
  }
  .theme {
    display: flex; align-items: center; gap: 6px; background: none;
    border: 1px solid var(--kbd-border); border-radius: 6px; padding: 3px 8px;
    color: var(--text-dim); font-size: 0.72rem; cursor: pointer; line-height: 1;
  }
  .theme:hover { color: var(--text-strong); border-color: var(--field-border); background: var(--hover); }
  .ticon { font-size: 0.82rem; }
  @media (max-width: 560px) { .tlabel { display: none; } }
  .body { flex: 1; display: flex; min-height: 0; }
  .side {
    width: 190px; flex: none; background: var(--panel); border-right: 1px solid var(--border);
    display: flex; flex-direction: column; overflow: hidden;
  }
  .navscroll { flex: 1; overflow-y: auto; padding: 8px; }
  .sidefoot {
    flex: none; border-top: 1px solid var(--border); padding: 9px 10px;
    display: flex; flex-direction: column; gap: 4px;
  }
  .ver { font-size: 0.68rem; color: var(--text-faint); }
  .upd {
    font-size: 0.72rem; text-align: left; background: none; border: none; padding: 0;
    cursor: pointer; line-height: 1.3;
  }
  .upd.link { color: var(--accent); }
  .upd.link:hover { text-decoration: underline; }
  .upd.avail { color: var(--accent); font-weight: 600; }
  .upd.avail:hover { text-decoration: underline; }
  .upd.ok { color: var(--good, #3fb950); cursor: default; }
  .upd:disabled { color: var(--text-faint); cursor: default; }
  .climsg { font-size: 0.66rem; color: var(--text-faint); line-height: 1.3; }
  .grp {
    display: flex; align-items: center; gap: 5px; width: 100%; text-align: left;
    background: none; border: none; cursor: pointer;
    color: var(--text-faint2); font-size: 0.62rem; text-transform: uppercase; letter-spacing: .07em;
    margin: 12px 0 4px; padding: 2px 8px;
  }
  .grp:hover { color: var(--text-dim); }
  .caret { font-size: 0.6rem; width: 9px; }
  .item {
    display: flex; align-items: center; justify-content: space-between; width: 100%;
    text-align: left; background: none; border: none; color: var(--item-text);
    padding: 5px 9px; border-radius: 6px; font-size: 0.84rem; cursor: pointer;
  }
  .item:hover:not(.disabled) { background: var(--hover); color: var(--text-strong); }
  .item.on { background: var(--active); color: var(--text-strong); }
  .item.disabled { color: var(--text-faint); cursor: default; }
  .soon { font-size: 0.6rem; color: var(--text-faint); border: 1px solid var(--pill-border); border-radius: 8px; padding: 0 6px; }
  .main { flex: 1; min-width: 0; overflow-y: auto; }
  .tabs { display: flex; gap: 2px; padding: 0 10px; background: var(--panel); border-bottom: 1px solid var(--border); }
  .tab { padding: 8px 12px; font-size: 0.82rem; color: var(--tab-inactive); border-bottom: 2px solid transparent; }
  .tab.on { color: var(--text-strong); border-color: var(--accent); }
  .mono { font-family: var(--font-mono); }
</style>
