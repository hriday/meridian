<script lang="ts">
  import { LLDPDiscover, LLDPSupported } from "../../../wailsjs/go/app/App.js";

  let loading = false;
  let error = "";
  let neighbors: any[] | null = null;
  let supported = true;

  LLDPSupported().then((s) => (supported = s)).catch(() => {});

  const needsPriv = () => error.toLowerCase().includes("elevated privileges");
  const needsNpcap = () => error.toLowerCase().includes("npcap");

  async function run() {
    if (loading) return;
    loading = true;
    error = "";
    neighbors = null;
    try {
      neighbors = await LLDPDiscover();
    } catch (e: any) {
      error = String(e?.message ?? e);
    } finally {
      loading = false;
    }
  }
</script>

<div class="tool">
  <h2>LLDP / CDP neighbor</h2>
  <p class="sub">
    Listen for the switch's own advertisements to answer <strong>"what switch and port am I plugged into?"</strong>
    Reads LLDP (and Cisco CDP) frames off the wire. Only works on a <strong>managed switch</strong> that advertises them
    — a home router won't. Point-in-time; nothing stored.
  </p>

  {#if !supported}
    <div class="hint">LLDP/CDP capture isn't supported on this operating system yet.</div>
  {:else}
    <button class="go" on:click={run} disabled={loading}>{loading ? "Listening… (up to ~35s)" : "Listen for neighbors"}</button>
    <p class="muted">Switches announce every ~30s, so this listens up to ~35 seconds. Needs raw capture — on macOS, run elevated or with ChmodBPF; on Linux, <span class="mono">sudo</span>; on Windows, install <span class="mono">Npcap</span> (free).</p>
  {/if}

  {#if loading}
    <div class="card listening"><span class="spinner"></span> Listening for LLDP and CDP advertisements…</div>
  {/if}

  {#if error}
    {#if needsNpcap()}
      <div class="card warn">
        <div class="ctitle">Needs Npcap</div>
        <p class="cbody">Windows has no built-in raw-capture API, so LLDP/CDP needs <strong>Npcap</strong> — a free driver from <span class="mono">npcap.com</span> (it's what Wireshark installs). Install it, then try again.</p>
      </div>
    {:else if needsPriv()}
      <div class="card warn">
        <div class="ctitle">Needs elevated privileges</div>
        <p class="cbody">Raw packet capture requires root. Run <span class="mono">sudo meridian-cli lldp</span>, relaunch the app elevated, or install ChmodBPF (Wireshark) to grant BPF access.</p>
      </div>
    {:else}
      <div class="hint">{error}</div>
    {/if}
  {/if}

  {#if neighbors}
    {#if neighbors.length === 0}
      <div class="card"><p class="cbody">No LLDP/CDP neighbors heard. The upstream device may not advertise them (common for consumer routers), or none was sent during the listen window — try again.</p></div>
    {:else}
      {#each neighbors as n}
        <div class="card ok">
          <div class="chead"><span class="proto">{n.protocol}</span><span class="sw">{n.systemName || n.chassisID}</span></div>
          <div class="grid">
            {#if n.portID}<div class="k">Port</div><div class="v mono">{n.portID}{#if n.portDesc && n.portDesc !== n.portID} <span class="dim">({n.portDesc})</span>{/if}</div>{/if}
            {#if n.vlan}<div class="k">VLAN</div><div class="v mono">{n.vlan}</div>{/if}
            {#if n.platform}<div class="k">Platform</div><div class="v">{n.platform}</div>{/if}
            {#if n.mgmtIP}<div class="k">Mgmt IP</div><div class="v mono">{n.mgmtIP}</div>{/if}
            {#if n.chassisID}<div class="k">Chassis</div><div class="v mono">{n.chassisID}</div>{/if}
            {#if n.capabilities && n.capabilities.length}<div class="k">Capabilities</div><div class="v">{n.capabilities.join(", ")}</div>{/if}
            {#if n.systemDesc}<div class="k">System</div><div class="v dim">{n.systemDesc}</div>{/if}
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
  .go:disabled { opacity: .6; cursor: default; }
  .muted { color: var(--text-dim); font-size: 0.78rem; margin: 8px 0 0; }
  .hint { background: var(--warn-bg); border: 1px solid var(--warn-border); color: var(--warn-text); border-radius: 7px; padding: 9px 12px; font-size: 0.85rem; margin: 12px 0; }
  .card { border: 1px solid var(--border); border-radius: 11px; padding: 13px 16px; margin-top: 12px; background: var(--card); }
  .card.ok { border-left: 3px solid var(--good, #3fb950); }
  .card.warn { border-left: 3px solid var(--warn-text, #f5a623); }
  .card.listening { display: flex; align-items: center; gap: 10px; color: var(--text-dim); font-size: 0.86rem; }
  .ctitle { font-size: 0.95rem; font-weight: 600; color: var(--text-strong); margin-bottom: 6px; }
  .cbody { font-size: 0.84rem; color: var(--text); line-height: 1.5; margin: 0; }
  .chead { display: flex; align-items: baseline; gap: 10px; margin-bottom: 10px; }
  .proto { font-size: 0.66rem; font-weight: 700; text-transform: uppercase; letter-spacing: .04em; padding: 1px 7px; border-radius: 5px; background: color-mix(in srgb, var(--accent) 18%, transparent); color: var(--accent); }
  .sw { font-size: 0.98rem; font-weight: 650; color: var(--text-strong); }
  .grid { display: grid; grid-template-columns: max-content 1fr; gap: 4px 16px; font-size: 0.84rem; align-items: baseline; }
  .k { color: var(--text-dim); white-space: nowrap; } .v { color: var(--text); }
  .dim { color: var(--text-dim); }
  .spinner { width: 13px; height: 13px; border: 2px solid var(--border); border-top-color: var(--accent); border-radius: 50%; display: inline-block; animation: spin 0.8s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }
  .mono { font-family: var(--font-mono); }
</style>
