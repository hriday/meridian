<script lang="ts">
  import { DHCPDiscover, DHCPSupported } from "../../../wailsjs/go/app/App.js";

  let loading = false;
  let error = "";
  let res: any = null;
  let supported = true;

  DHCPSupported().then((s) => (supported = s)).catch(() => {});

  const needsPriv = () => error.toLowerCase().includes("elevated privileges");

  async function run() {
    if (loading) return;
    loading = true;
    error = "";
    res = null;
    try {
      res = await DHCPDiscover();
    } catch (e: any) {
      error = String(e?.message ?? e);
    } finally {
      loading = false;
    }
  }

  function lease(secs: number): string {
    if (!secs) return "";
    const h = Math.floor(secs / 3600), m = Math.floor((secs % 3600) / 60);
    if (h >= 24) return `${Math.floor(h / 24)}d ${h % 24}h`;
    return h ? `${h}h ${m}m` : `${m}m`;
  }
</script>

<div class="tool">
  <h2>DHCP discover</h2>
  <p class="sub">
    Broadcast a DHCP <span class="mono">DISCOVER</span> on your LAN and list every server that answers — confirm your
    DHCP server and what it hands out, and <strong>catch a rogue server</strong> (more than one answering is a red
    flag). Point-in-time; nothing stored.
  </p>

  {#if !supported}
    <div class="hint">DHCP discovery isn't supported on this operating system yet (macOS and Linux only).</div>
  {:else}
    <button class="go" on:click={run} disabled={loading}>{loading ? "Listening…" : "Discover DHCP servers"}</button>
    <p class="muted">Broadcasts a DISCOVER and listens ~5s. Works without admin on macOS; on Linux, binding port 68 may need <span class="mono">sudo</span>.</p>
  {/if}

  {#if error}
    {#if needsPriv()}
      <div class="card warn">
        <div class="ctitle">Needs elevated privileges</div>
        <p class="cbody">DHCP discovery binds UDP port 68, which requires root. Run <span class="mono">sudo meridian-cli dhcpdiscover</span>, or relaunch the app elevated.</p>
      </div>
    {:else}
      <div class="hint">{error}</div>
    {/if}
  {/if}

  {#if res}
    {#if !res.servers || res.servers.length === 0}
      <div class="card"><p class="cbody">No DHCP server answered on <span class="mono">{res.interface}</span> within the listen window.</p></div>
    {:else}
      <div class="banner {res.rogue ? 'bad' : 'ok'}">
        {#if res.rogue}
          <strong>⚠ Rogue DHCP</strong> — {res.servers.length} servers answered on {res.interface}. Only one should. Investigate the unexpected one immediately.
        {:else}
          <strong>One DHCP server</strong> answered on {res.interface} — as expected.
        {/if}
      </div>
      {#each res.servers as s, i}
        <div class="card {res.rogue && i > 0 ? 'bad' : ''}">
          <div class="grid">
            <div class="k">Server</div><div class="v mono">{s.serverID}</div>
            <div class="k">Offered address</div><div class="v mono">{s.offered}</div>
            {#if s.subnetMask}<div class="k">Subnet mask</div><div class="v mono">{s.subnetMask}</div>{/if}
            {#if s.gateway}<div class="k">Gateway</div><div class="v mono">{s.gateway}</div>{/if}
            {#if s.dns && s.dns.length}<div class="k">DNS</div><div class="v mono">{s.dns.join(", ")}</div>{/if}
            {#if s.domainName}<div class="k">Domain</div><div class="v mono">{s.domainName}</div>{/if}
            {#if s.leaseSecs}<div class="k">Lease</div><div class="v">{lease(s.leaseSecs)}</div>{/if}
          </div>
        </div>
      {/each}
    {/if}
  {/if}
</div>

<style>
  .tool { padding: 16px 20px 40px; max-width: 820px; }
  h2 { margin: 0 0 2px; font-size: 1.15rem; font-weight: 600; color: var(--text-strong); }
  .sub { margin: 0 0 15px; color: var(--text-dim); font-size: 0.86rem; line-height: 1.5; }
  .go { background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px; padding: 9px 16px; font: inherit; font-size: 0.9rem; cursor: pointer; }
  .go:disabled { opacity: .5; cursor: default; }
  .muted { color: var(--text-dim); font-size: 0.78rem; margin: 8px 0 0; }
  .hint { background: var(--warn-bg); border: 1px solid var(--warn-border); color: var(--warn-text); border-radius: 7px; padding: 9px 12px; font-size: 0.85rem; margin: 12px 0; }
  .banner { border: 1px solid var(--border); border-left-width: 3px; border-radius: 9px; padding: 11px 14px; margin: 16px 0 10px; font-size: 0.88rem; color: var(--text); background: var(--card); }
  .banner.ok { border-left-color: var(--good, #3fb950); }
  .banner.bad { border-left-color: var(--danger, #e5484d); }
  .card { border: 1px solid var(--border); border-radius: 11px; padding: 13px 16px; margin-top: 10px; background: var(--card); }
  .card.bad { border-left: 3px solid var(--danger, #e5484d); }
  .card.warn { border-left: 3px solid var(--warn-text, #f5a623); margin-top: 12px; }
  .ctitle { font-size: 0.95rem; font-weight: 600; color: var(--text-strong); margin-bottom: 6px; }
  .cbody { font-size: 0.84rem; color: var(--text); line-height: 1.5; margin: 0; }
  .grid { display: grid; grid-template-columns: max-content 1fr; gap: 4px 16px; font-size: 0.84rem; align-items: baseline; }
  .k { color: var(--text-dim); white-space: nowrap; } .v { color: var(--text); }
  .mono { font-family: var(--font-mono); }
</style>
