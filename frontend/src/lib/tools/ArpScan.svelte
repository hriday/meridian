<script lang="ts">
  import { ARPScan, ARPSupported } from "../../../wailsjs/go/app/App.js";
  import { requestedTool } from "../nav";
  import ToolHelp from "../ToolHelp.svelte";

  let loading = false;
  let error = "";
  let res: any = null;
  let supported = true;

  ARPSupported().then((s) => (supported = s)).catch(() => {});

  const needsPriv = () => error.toLowerCase().includes("elevated privileges");

  async function run() {
    if (loading) return;
    loading = true;
    error = "";
    res = null;
    try {
      res = await ARPScan();
    } catch (e: any) {
      error = String(e?.message ?? e);
    } finally {
      loading = false;
    }
  }
</script>

<div class="tool">
  <h2>ARP scan</h2>
  <p class="sub">
    Sweep your local subnet with ARP requests and list <strong>every device that's actually on the segment</strong> —
    its IP, hardware MAC, and vendor. Because ARP is link-local, it finds devices that ignore ping. Point-in-time;
    nothing stored.
  </p>

  <ToolHelp sudo="needs sudo or BPF-group access">
    Each row is a host that <strong>answered an ARP request</strong>, so it is genuinely live on this segment right now —
    more reliable than ping. The <strong>vendor</strong> comes from the MAC's manufacturer prefix; a blank one just means
    an unknown or randomized prefix, not a problem. Match the list against what you expect on the LAN — an
    <strong>unfamiliar device</strong>, or more hosts than you can account for, is what to chase down.
  </ToolHelp>

  {#if !supported}
    <div class="hint">ARP scanning isn't supported on this operating system yet.</div>
  {:else}
    <button class="go" on:click={run} disabled={loading}>{loading ? "Sweeping…" : "Scan the local subnet"}</button>
    <p class="muted">Broadcasts an ARP request to each address and listens a couple of seconds. On Windows it works without admin; on macOS it needs ChmodBPF or elevated; on Linux, <span class="mono">sudo</span>.</p>
  {/if}

  {#if loading}
    <div class="card listening"><span class="spinner"></span> Sweeping and resolving vendors…</div>
  {/if}

  {#if error}
    {#if needsPriv()}
      <div class="card warn">
        <div class="ctitle">Needs elevated privileges</div>
        <p class="cbody">Raw packet access is required. Run <span class="mono">sudo meridian-cli arpscan</span>, relaunch the app elevated, or install ChmodBPF (Wireshark).</p>
      </div>
    {:else}
      <div class="hint">{error}</div>
    {/if}
  {/if}

  {#if res}
    <div class="head">
      <span class="count">{res.hosts.length}</span> host{res.hosts.length === 1 ? "" : "s"} on
      <span class="mono">{res.subnet}</span> <span class="dim">· {res.interface}</span>
    </div>
    {#if !res.hosts.length}<button class="diag" type="button" on:click={() => requestedTool.set({ tool: "netdiag", value: "" })}>Run Local Network Diagnostic →</button>{/if}
    {#if res.hosts.length}
      <table class="tbl">
        <thead><tr><th>IP</th><th>MAC</th><th>Vendor</th></tr></thead>
        <tbody>
          {#each res.hosts as h}
            <tr>
              <td class="mono">{h.ip}</td>
              <td class="mono dim">{h.mac}</td>
              <td>{h.vendor || "—"}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  {/if}
</div>

<style>
  .tool { padding: 16px 20px 40px; max-width: 760px; }
  h2 { margin: 0 0 2px; font-size: 1.15rem; font-weight: 600; color: var(--text-strong); }
  .sub { margin: 0 0 15px; color: var(--text-dim); font-size: 0.86rem; line-height: 1.5; }
  .go { background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px; padding: 9px 16px; font: inherit; font-size: 0.9rem; cursor: pointer; }
  .go:disabled { opacity: .6; cursor: default; }
  .muted { color: var(--text-dim); font-size: 0.78rem; margin: 8px 0 0; }
  .hint { background: var(--warn-bg); border: 1px solid var(--warn-border); color: var(--warn-text); border-radius: 7px; padding: 9px 12px; font-size: 0.85rem; margin: 12px 0; }
  .card { border: 1px solid var(--border); border-radius: 11px; padding: 13px 16px; margin-top: 12px; background: var(--card); }
  .card.warn { border-left: 3px solid var(--warn-text, #f5a623); }
  .card.listening { display: flex; align-items: center; gap: 10px; color: var(--text-dim); font-size: 0.86rem; }
  .ctitle { font-size: 0.95rem; font-weight: 600; color: var(--text-strong); margin-bottom: 6px; }
  .cbody { font-size: 0.84rem; color: var(--text); line-height: 1.5; margin: 0; }
  .head { margin: 16px 0 8px; font-size: 0.86rem; color: var(--text); }
  .count { font-weight: 700; color: var(--text-strong); }
  table.tbl { width: 100%; border-collapse: collapse; font-size: 0.84rem; }
  .tbl th { text-align: left; color: var(--text-dim); font-weight: 500; font-size: 0.68rem; text-transform: uppercase; letter-spacing: .04em; padding: 7px 10px; border-bottom: 1px solid var(--border); }
  .tbl td { padding: 7px 10px; border-bottom: 1px solid var(--border-soft); }
  .dim { color: var(--text-dim); }
  .spinner { width: 13px; height: 13px; border: 2px solid var(--border); border-top-color: var(--accent); border-radius: 50%; display: inline-block; animation: spin 0.8s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }
  .mono { font-family: var(--font-mono); }
  .diag { margin-top: 8px; align-self: flex-start; background: transparent; color: var(--accent); border: 1px solid var(--border); border-radius: 8px; padding: 6px 12px; font: inherit; font-size: 0.8rem; cursor: pointer; }
  .diag:hover { border-color: var(--accent); }
</style>
