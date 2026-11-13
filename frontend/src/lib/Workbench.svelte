<script lang="ts">
  import SubnetCalc from "./tools/SubnetCalc.svelte";
  import Lookup from "./tools/Lookup.svelte";
  import Traceroute from "./tools/Traceroute.svelte";
  import Bgp from "./tools/Bgp.svelte";
  import AtlasPing from "./tools/AtlasPing.svelte";
  import AtlasTrace from "./tools/AtlasTrace.svelte";
  import FirewallAnalyzer from "./tools/FirewallAnalyzer.svelte";
  import WebServerAnalyzer from "./tools/WebServerAnalyzer.svelte";
  import TerraformExposure from "./tools/TerraformExposure.svelte";
  import ConfigDiff from "./tools/ConfigDiff.svelte";
  import Tls from "./tools/Tls.svelte";
  import HttpCheck from "./tools/HttpCheck.svelte";
  import Whois from "./tools/Whois.svelte";
  import Blocklist from "./tools/Blocklist.svelte";
  import DnsAudit from "./tools/DnsAudit.svelte";
  import Netcat from "./tools/Netcat.svelte";
  import AdAudit from "./tools/AdAudit.svelte";
  import CiscoPasswords from "./tools/CiscoPasswords.svelte";
  import SnmpBrute from "./tools/SnmpBrute.svelte";
  import BreachLookup from "./tools/BreachLookup.svelte";
  import CveLookup from "./tools/CveLookup.svelte";
  import HashId from "./tools/HashId.svelte";
  import ThreatIntel from "./tools/ThreatIntel.svelte";
  import CertExpiry from "./tools/CertExpiry.svelte";
  import EmailAuth from "./tools/EmailAuth.svelte";
  import SubdomainFinder from "./tools/SubdomainFinder.svelte";
  import SecurityHeaders from "./tools/SecurityHeaders.svelte";
  import Ntp from "./tools/Ntp.svelte";
  import Dns from "./tools/Dns.svelte";
  import Interfaces from "./tools/Interfaces.svelte";
  import Ping from "./tools/Ping.svelte";
  import PingMonitor from "./tools/PingMonitor.svelte";
  import SnmpMonitor from "./tools/SnmpMonitor.svelte";
  import DeviceHealth from "./tools/DeviceHealth.svelte";
  import SnmpSweep from "./tools/SnmpSweep.svelte";
  import MibBrowser from "./tools/MibBrowser.svelte";
  import SwitchPortMapper from "./tools/SwitchPortMapper.svelte";
  import SyslogReceiver from "./tools/SyslogReceiver.svelte";
  import TrapReceiver from "./tools/TrapReceiver.svelte";
  import NetflowCollector from "./tools/NetflowCollector.svelte";
  import TftpReceiver from "./tools/TftpReceiver.svelte";
  import PingSweep from "./tools/PingSweep.svelte";
  import SpeedTest from "./tools/SpeedTest.svelte";
  import PortScan from "./tools/PortScan.svelte";
  import WakeOnLan from "./tools/WakeOnLan.svelte";
  import MacVendor from "./tools/MacVendor.svelte";
  import IpSpace from "./tools/IpSpace.svelte";
  import { themePref, cycleTheme, themeIcon, themeLabel } from "./theme";
  import { requestedTool } from "./nav";
  import { AppVersion, CheckForUpdate, CLIInfo, InstallCLI } from "../../wailsjs/go/app/App.js";
  import { BrowserOpenURL } from "../../wailsjs/runtime/runtime.js";

  type Tool = { id: string; name: string; group: string; ready: boolean };
  const tools: Tool[] = [
    { id: "lookup", name: "Lookup", group: "Internet", ready: true },
    { id: "trace", name: "Traceroute (AS)", group: "Internet", ready: true },
    { id: "bgp", name: "BGP & looking glass", group: "Internet", ready: true },
    { id: "atlasping", name: "Remote ping", group: "Internet", ready: true },
    { id: "atlastrace", name: "Remote traceroute", group: "Internet", ready: true },
    { id: "tls", name: "TLS certificate", group: "Internet", ready: true },
    { id: "http", name: "HTTP check", group: "Internet", ready: true },
    { id: "sechdr", name: "Security headers", group: "Internet", ready: true },
    { id: "whois", name: "Whois", group: "Internet", ready: true },
    { id: "subdomains", name: "Subdomain finder", group: "Internet", ready: true },
    { id: "blocklist", name: "Blocklist check", group: "Internet", ready: true },
    { id: "dnsaudit", name: "DNS audit", group: "Internet", ready: true },
    { id: "emailauth", name: "Email auth (SPF/DKIM/DMARC)", group: "Internet", ready: true },
    { id: "subnet", name: "Subnet calculator", group: "Addressing", ready: true },
    { id: "ipspace", name: "IP space", group: "Addressing", ready: true },
    { id: "ntp", name: "NTP", group: "Local / environment", ready: true },
    { id: "dns", name: "DNS", group: "Local / environment", ready: true },
    { id: "ifaces", name: "Interfaces", group: "Local / environment", ready: true },
    { id: "ping1", name: "Ping", group: "Local network", ready: true },
    { id: "pingmon", name: "Ping monitor", group: "Local network", ready: true },
    { id: "ifmon", name: "SNMP interface monitor", group: "SNMP & monitoring", ready: true },
    { id: "devhealth", name: "Device health", group: "SNMP & monitoring", ready: true },
    { id: "snmpsweep", name: "SNMP sweep", group: "SNMP & monitoring", ready: true },
    { id: "mib", name: "MIB browser", group: "SNMP & monitoring", ready: true },
    { id: "swport", name: "Switch port mapper", group: "SNMP & monitoring", ready: true },
    { id: "syslog", name: "Syslog receiver", group: "Collectors", ready: true },
    { id: "traps", name: "Trap receiver", group: "Collectors", ready: true },
    { id: "netflow", name: "NetFlow", group: "Collectors", ready: true },
    { id: "tftp", name: "TFTP grab", group: "Collectors", ready: true },
    { id: "ping", name: "Ping sweep", group: "Local network", ready: true },
    { id: "ports", name: "Port scanner", group: "Local network", ready: true },
    { id: "wol", name: "Wake-on-LAN", group: "Local network", ready: true },
    { id: "macvendor", name: "MAC vendor", group: "Local network", ready: true },
    { id: "speedtest", name: "LAN speed test", group: "Local network", ready: true },
    { id: "nc", name: "Netcat", group: "Security", ready: true },
    { id: "adaudit", name: "AD password audit", group: "Security", ready: true },
    { id: "ciscopw", name: "Cisco passwords", group: "Security", ready: true },
    { id: "hashid", name: "Hash identifier", group: "Security", ready: true },
    { id: "snmpbrute", name: "SNMP community audit", group: "Security", ready: true },
    { id: "breaches", name: "Breach lookup", group: "Security", ready: true },
    { id: "cve", name: "CVE lookup", group: "Security", ready: true },
    { id: "threatintel", name: "Threat intel (C2)", group: "Security", ready: true },
    { id: "certexpiry", name: "Cert expiry tracker", group: "Internet", ready: true },
    { id: "fwanalyze", name: "Firewall analyzer", group: "Config insight", ready: true },
    { id: "webanalyze", name: "Web server analyzer", group: "Config insight", ready: true },
    { id: "tfexposure", name: "Terraform exposure", group: "Config insight", ready: true },
    { id: "cfgdiff", name: "Config diff", group: "Config insight", ready: true },
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

  function onKey(e: KeyboardEvent) {
    if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") {
      e.preventDefault();
      sideCollapsed = false;
      searchEl?.focus();
    }
  }

  function toggleGroup(g: string) {
    closed = { ...closed, [g]: !closed[g] };
  }

  // A tool pivot (e.g. from IP space → Subnet calc) switches the active tool.
  $: if ($requestedTool) active = $requestedTool.tool;

  $: visible = (g: string) =>
    tools.filter((t) => t.group === g && t.name.toLowerCase().includes(filter.toLowerCase()));
  $: activeTool = tools.find((t) => t.id === active);
</script>

<svelte:window on:keydown={onKey} />

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
  </header>

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
              <button class="upd link" on:click={installCLI} disabled={cliBusy} title="Symlink meridian-cli onto your PATH">
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
      {#if active === "lookup"}<Lookup />{:else if active === "trace"}<Traceroute />{:else if active === "bgp"}<Bgp />{:else if active === "atlasping"}<AtlasPing />{:else if active === "atlastrace"}<AtlasTrace />{:else if active === "tls"}<Tls />{:else if active === "http"}<HttpCheck />{:else if active === "sechdr"}<SecurityHeaders />{:else if active === "whois"}<Whois />{:else if active === "subdomains"}<SubdomainFinder />{:else if active === "blocklist"}<Blocklist />{:else if active === "dnsaudit"}<DnsAudit />{:else if active === "emailauth"}<EmailAuth />{:else if active === "subnet"}<SubnetCalc />{:else if active === "ipspace"}<IpSpace />{:else if active === "ntp"}<Ntp />{:else if active === "dns"}<Dns />{:else if active === "ifaces"}<Interfaces />{:else if active === "ping1"}<Ping />{:else if active === "pingmon"}<PingMonitor />{:else if active === "ifmon"}<SnmpMonitor />{:else if active === "devhealth"}<DeviceHealth />{:else if active === "snmpsweep"}<SnmpSweep />{:else if active === "mib"}<MibBrowser />{:else if active === "swport"}<SwitchPortMapper />{:else if active === "syslog"}<SyslogReceiver />{:else if active === "traps"}<TrapReceiver />{:else if active === "netflow"}<NetflowCollector />{:else if active === "tftp"}<TftpReceiver />{:else if active === "ping"}<PingSweep />{:else if active === "ports"}<PortScan />{:else if active === "wol"}<WakeOnLan />{:else if active === "macvendor"}<MacVendor />{:else if active === "speedtest"}<SpeedTest />{:else if active === "nc"}<Netcat />{:else if active === "adaudit"}<AdAudit />{:else if active === "ciscopw"}<CiscoPasswords />{:else if active === "hashid"}<HashId />{:else if active === "snmpbrute"}<SnmpBrute />{:else if active === "breaches"}<BreachLookup />{:else if active === "cve"}<CveLookup />{:else if active === "threatintel"}<ThreatIntel />{:else if active === "certexpiry"}<CertExpiry />{:else if active === "fwanalyze"}<FirewallAnalyzer />{:else if active === "webanalyze"}<WebServerAnalyzer />{:else if active === "tfexposure"}<TerraformExposure />{:else if active === "cfgdiff"}<ConfigDiff />{/if}
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
        <p class="updnote">Downloads <span class="mono">{upAsset}</span>. Meridian isn't code-signed, so the first launch needs the one-time step in the README (right-click → Open on macOS; "More info → Run anyway" on Windows).</p>
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
