<script lang="ts">
  import { PathMTU, MTUSupported } from "../../../wailsjs/go/app/App.js";

  let target = "";
  let loading = false;
  let error = "";
  let res: any = null;
  let supported = true;

  MTUSupported().then((s) => (supported = s)).catch(() => {});

  async function run() {
    const t = target.trim();
    if (!t || loading) return;
    loading = true;
    error = "";
    res = null;
    try {
      res = await PathMTU(t);
    } catch (e: any) {
      error = String(e?.message ?? e);
    } finally {
      loading = false;
    }
  }
</script>

<div class="tool">
  <h2>Path MTU discovery</h2>
  <p class="sub">
    Find the largest packet that reaches a host <strong>without being fragmented</strong> — i.e. the real MTU of the
    path. A value below your link MTU means a smaller-MTU hop (often a VPN or tunnel) that will silently drop full-size
    packets with Don't-Fragment set. No admin rights needed. Point-in-time; nothing stored.
  </p>

  {#if !supported}
    <div class="hint">Path MTU discovery isn't supported on this operating system yet.</div>
  {:else}
    <form on:submit|preventDefault={run} class="row">
      <input class="field mono" bind:value={target} placeholder="cloudflare.com  or  104.16.123.45" autocomplete="off" spellcheck="false" aria-label="Target host" />
      <button class="go" disabled={loading || !target.trim()}>{loading ? "Probing…" : "Discover"}</button>
    </form>
    <p class="muted">IPv4 only. Binary-searches packet sizes with the Don't-Fragment bit set — takes a few seconds.</p>
  {/if}

  {#if error}<div class="hint">{error}</div>{/if}

  {#if res}
    {#if !res.dfEnforced}
      <div class="card warn">
        <div class="chead"><span class="dot warn"></span><span class="ctitle">Can't measure on this system</span></div>
        <p class="cbody">{res.note}</p>
        <p class="cbody dim">The tool deliberately won't report a number it can't trust. On a system that enforces Don't-Fragment it measures the real path MTU.</p>
      </div>
    {:else}
      <div class="card {res.bottleneck ? 'bad' : 'ok'}">
        <div class="chead">
          <span class="dot {res.bottleneck ? 'bad' : 'ok'}"></span>
          <span class="mtu mono">{res.pathMTU}</span><span class="unit">bytes path MTU</span>
        </div>
        <div class="grid">
          <div class="k">Target</div><div class="v mono">{res.target} <span class="dim">({res.addr})</span></div>
          {#if res.localMTU > 0}<div class="k">Your link MTU</div><div class="v mono">{res.localMTU} bytes</div>{/if}
          {#if res.nextHopMTU > 0}<div class="k">Next-hop MTU</div><div class="v mono">{res.nextHopMTU} bytes <span class="dim">(router-reported)</span></div>{/if}
        </div>
        {#if res.bottleneck}
          <p class="cbody bad-text">Bottleneck — the path only carries {res.pathMTU} bytes, below your {res.localMTU}-byte link. Full-size packets with Don't-Fragment set will be dropped by a smaller-MTU hop (often a VPN/tunnel). Lower the MTU/MSS on that path to fix black-holing.</p>
        {:else}
          <p class="cbody">No bottleneck — the path carries your full {res.pathMTU}-byte link MTU end to end.</p>
        {/if}
      </div>
    {/if}
  {/if}
</div>

<style>
  .tool { padding: 16px 20px 40px; max-width: 820px; }
  h2 { margin: 0 0 2px; font-size: 1.15rem; font-weight: 600; color: var(--text-strong); }
  .sub { margin: 0 0 15px; color: var(--text-dim); font-size: 0.86rem; line-height: 1.5; }
  .row { display: flex; gap: 8px; margin-bottom: 8px; flex-wrap: wrap; }
  .field { flex: 1; min-width: 200px; background: var(--well); border: 1px solid var(--field-border); border-radius: 7px; padding: 9px 12px; color: var(--text); font-size: 0.9rem; outline: none; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .go { background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px; padding: 0 16px; font: inherit; font-size: 0.9rem; cursor: pointer; }
  .go:disabled { opacity: .5; cursor: default; }
  .muted { color: var(--text-dim); font-size: 0.78rem; margin: 0 0 6px; }
  .hint { background: var(--warn-bg); border: 1px solid var(--warn-border); color: var(--warn-text); border-radius: 7px; padding: 9px 12px; font-size: 0.85rem; margin: 10px 0; }
  .card { border: 1px solid var(--border); border-left-width: 3px; border-radius: 11px; padding: 14px 16px; margin-top: 16px; background: var(--card); }
  .card.ok { border-left-color: var(--good, #3fb950); }
  .card.bad { border-left-color: var(--danger, #e5484d); }
  .card.warn { border-left-color: var(--warn-text, #f5a623); }
  .chead { display: flex; align-items: baseline; gap: 10px; flex-wrap: wrap; }
  .dot { width: 10px; height: 10px; border-radius: 50%; align-self: center; flex: none; }
  .dot.ok { background: var(--good, #3fb950); } .dot.bad { background: var(--danger, #e5484d); } .dot.warn { background: var(--warn-text, #f5a623); }
  .mtu { font-size: 1.5rem; font-weight: 700; color: var(--text-strong); }
  .unit { font-size: 0.8rem; color: var(--text-dim); }
  .ctitle { font-size: 0.95rem; font-weight: 600; color: var(--text-strong); }
  .grid { display: grid; grid-template-columns: max-content 1fr; gap: 4px 16px; margin: 12px 0 0; font-size: 0.84rem; }
  .k { color: var(--text-dim); } .v { color: var(--text); }
  .cbody { font-size: 0.84rem; color: var(--text); line-height: 1.5; margin: 12px 0 0; }
  .cbody.dim, .dim { color: var(--text-dim); }
  .bad-text { color: var(--danger, #e5484d); }
  .mono { font-family: var(--font-mono); }
</style>
