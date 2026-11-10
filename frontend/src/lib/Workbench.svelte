<script lang="ts">
  import SubnetCalc from "./tools/SubnetCalc.svelte";
  import Lookup from "./tools/Lookup.svelte";
  import Traceroute from "./tools/Traceroute.svelte";
  import Bgp from "./tools/Bgp.svelte";
  import AtlasPing from "./tools/AtlasPing.svelte";
  import AtlasTrace from "./tools/AtlasTrace.svelte";
  import FirewallAnalyzer from "./tools/FirewallAnalyzer.svelte";
  import WebServerAnalyzer from "./tools/WebServerAnalyzer.svelte";
  import Tls from "./tools/Tls.svelte";
  import HttpCheck from "./tools/HttpCheck.svelte";
  import Whois from "./tools/Whois.svelte";
  import Blocklist from "./tools/Blocklist.svelte";
  import DnsAudit from "./tools/DnsAudit.svelte";
  import Netcat from "./tools/Netcat.svelte";
  import AdAudit from "./tools/AdAudit.svelte";
  import Ntp from "./tools/Ntp.svelte";
  import Dns from "./tools/Dns.svelte";
  import Interfaces from "./tools/Interfaces.svelte";
  import Ping from "./tools/Ping.svelte";
  import PingMonitor from "./tools/PingMonitor.svelte";
  import SnmpMonitor from "./tools/SnmpMonitor.svelte";
  import PingSweep from "./tools/PingSweep.svelte";
  import PortScan from "./tools/PortScan.svelte";
  import WakeOnLan from "./tools/WakeOnLan.svelte";
  import MacVendor from "./tools/MacVendor.svelte";
  import IpSpace from "./tools/IpSpace.svelte";
  import { themePref, cycleTheme, themeIcon, themeLabel } from "./theme";
  import { requestedTool } from "./nav";

  type Tool = { id: string; name: string; group: string; ready: boolean };
  const tools: Tool[] = [
    { id: "lookup", name: "Lookup", group: "Internet", ready: true },
    { id: "trace", name: "Traceroute (AS)", group: "Internet", ready: true },
    { id: "bgp", name: "BGP & looking glass", group: "Internet", ready: true },
    { id: "atlasping", name: "Remote ping", group: "Internet", ready: true },
    { id: "atlastrace", name: "Remote traceroute", group: "Internet", ready: true },
    { id: "tls", name: "TLS certificate", group: "Internet", ready: true },
    { id: "http", name: "HTTP check", group: "Internet", ready: true },
    { id: "whois", name: "Whois", group: "Internet", ready: true },
    { id: "blocklist", name: "Blocklist check", group: "Internet", ready: true },
    { id: "dnsaudit", name: "DNS audit", group: "Internet", ready: true },
    { id: "subnet", name: "Subnet calculator", group: "Addressing", ready: true },
    { id: "ipspace", name: "IP space", group: "Addressing", ready: true },
    { id: "ntp", name: "NTP", group: "Local / environment", ready: true },
    { id: "dns", name: "DNS", group: "Local / environment", ready: true },
    { id: "ifaces", name: "Interfaces", group: "Local / environment", ready: true },
    { id: "ping1", name: "Ping", group: "Local network", ready: true },
    { id: "pingmon", name: "Ping monitor", group: "Local network", ready: true },
    { id: "ifmon", name: "SNMP interface monitor", group: "SNMP & monitoring", ready: true },
    { id: "ping", name: "Ping sweep", group: "Local network", ready: true },
    { id: "ports", name: "Port scanner", group: "Local network", ready: true },
    { id: "wol", name: "Wake-on-LAN", group: "Local network", ready: true },
    { id: "macvendor", name: "MAC vendor", group: "Local network", ready: true },
    { id: "nc", name: "Netcat", group: "Security", ready: true },
    { id: "adaudit", name: "AD password audit", group: "Security", ready: true },
    { id: "fwanalyze", name: "Firewall analyzer", group: "Config insight", ready: true },
    { id: "webanalyze", name: "Web server analyzer", group: "Config insight", ready: true },
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
      </nav>
    {/if}

    <main class="main">
      <div class="tabs"><div class="tab on">{activeTool?.name ?? ""}</div></div>
      {#if active === "lookup"}<Lookup />{:else if active === "trace"}<Traceroute />{:else if active === "bgp"}<Bgp />{:else if active === "atlasping"}<AtlasPing />{:else if active === "atlastrace"}<AtlasTrace />{:else if active === "tls"}<Tls />{:else if active === "http"}<HttpCheck />{:else if active === "whois"}<Whois />{:else if active === "blocklist"}<Blocklist />{:else if active === "dnsaudit"}<DnsAudit />{:else if active === "subnet"}<SubnetCalc />{:else if active === "ipspace"}<IpSpace />{:else if active === "ntp"}<Ntp />{:else if active === "dns"}<Dns />{:else if active === "ifaces"}<Interfaces />{:else if active === "ping1"}<Ping />{:else if active === "pingmon"}<PingMonitor />{:else if active === "ifmon"}<SnmpMonitor />{:else if active === "ping"}<PingSweep />{:else if active === "ports"}<PortScan />{:else if active === "wol"}<WakeOnLan />{:else if active === "macvendor"}<MacVendor />{:else if active === "nc"}<Netcat />{:else if active === "adaudit"}<AdAudit />{:else if active === "fwanalyze"}<FirewallAnalyzer />{:else if active === "webanalyze"}<WebServerAnalyzer />{/if}
    </main>
  </div>
</div>

<style>
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
    padding: 8px; overflow-y: auto;
  }
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
