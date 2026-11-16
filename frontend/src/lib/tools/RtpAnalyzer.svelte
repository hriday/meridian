<script lang="ts">
  import { RTPCapture, RTPSupported } from "../../../wailsjs/go/app/App.js";
  import ToolHelp from "../ToolHelp.svelte";

  let seconds = 10;
  let loading = false;
  let error = "";
  let streams: any[] | null = null;
  let supported = true;

  RTPSupported().then((s) => (supported = s)).catch(() => {});

  const needsNpcap = () => error.toLowerCase().includes("npcap");
  const needsPriv = () => error.toLowerCase().includes("elevated privileges");

  async function run() {
    if (loading) return;
    loading = true; error = ""; streams = null;
    try {
      streams = await RTPCapture(Number(seconds) || 10);
    } catch (e: any) {
      error = String(e?.message ?? e);
    } finally {
      loading = false;
    }
  }
  const hex = (n: number) => "0x" + (n >>> 0).toString(16).toUpperCase().padStart(8, "0");
  const bad = (s: any) => s.lossPct > 1 || s.jitterMs > 30;
</script>

<div class="tool">
  <h2>RTP stream analyzer</h2>
  <p class="sub">
    Capture live <strong>RTP media</strong> off the wire and measure real per-stream <strong>jitter (RFC 3550), packet
    loss and sequence gaps</strong> — the ground truth when a call sounds bad. You only see RTP you're in the path of
    (your own calls, or a SPAN/mirror port). Point-in-time; nothing stored.
  </p>

  <ToolHelp sudo="needs sudo or BPF-group access">
    Each card is one media stream. <strong>Loss</strong> and <strong>jitter</strong> are the numbers that matter —
    under roughly <strong>1% loss</strong> and <strong>30 ms jitter</strong> is healthy (green); above either turns
    the card red and is what a caller hears as choppy or robotic audio. <strong>No streams</strong> just means
    none crossed your capture point — make a call, or use a SPAN/mirror port.
  </ToolHelp>

  {#if !supported}
    <div class="hint">RTP capture isn't supported on this operating system yet.</div>
  {:else}
    <div class="row">
      <label>Capture for <input class="field" type="number" min="3" max="60" bind:value={seconds} /> s</label>
      <button class="go" on:click={run} disabled={loading}>{loading ? `Capturing ${seconds}s…` : "Capture"}</button>
    </div>
    <p class="muted">Needs raw capture — on macOS, ChmodBPF or elevated; on Linux, <span class="mono">sudo</span>; on Windows, Npcap.</p>
  {/if}

  {#if loading}<div class="card listening"><span class="spinner"></span> Sniffing for RTP…</div>{/if}

  {#if error}
    {#if needsNpcap()}
      <div class="card warn"><div class="ctitle">Needs Npcap</div><p class="cbody">Install <strong>Npcap</strong> (free, <span class="mono">npcap.com</span>) for raw capture on Windows, then try again.</p></div>
    {:else if needsPriv()}
      <div class="card warn"><div class="ctitle">Needs elevated privileges</div><p class="cbody">Run <span class="mono">sudo meridian-cli rtp</span>, relaunch elevated, or install ChmodBPF (Wireshark).</p></div>
    {:else}<div class="hint">{error}</div>{/if}
  {/if}

  {#if streams}
    {#if streams.length === 0}
      <div class="card"><p class="cbody">No RTP streams seen. You only capture RTP you're in the path of — make a call, or use a SPAN/mirror port.</p></div>
    {:else}
      {#each streams as s}
        <div class="card {bad(s) ? 'bad' : 'ok'}">
          <div class="chead"><span class="ssrc mono">{hex(s.ssrc)}</span><span class="codec">{s.codec}</span></div>
          <div class="flow mono">{s.src} → {s.dst}</div>
          <div class="stats">
            <div class="stat"><span class="sl">Packets</span><span class="sv mono">{s.packets}</span></div>
            <div class="stat" class:warn={s.lossPct > 1}><span class="sl">Loss</span><span class="sv mono">{s.lossPct}% <small>({s.lost})</small></span></div>
            <div class="stat" class:warn={s.jitterMs > 30}><span class="sl">Jitter</span><span class="sv mono">{s.jitterMs} ms</span></div>
            <div class="stat"><span class="sl">Duration</span><span class="sv mono">{(s.durationMs / 1000).toFixed(1)}s</span></div>
          </div>
        </div>
      {/each}
    {/if}
  {/if}
</div>

<style>
  .tool { padding: 16px 20px 40px; max-width: 880px; }
  h2 { margin: 0 0 2px; font-size: 1.15rem; font-weight: 600; color: var(--text-strong); }
  .sub { margin: 0 0 15px; color: var(--text-dim); font-size: 0.86rem; line-height: 1.5; }
  .row { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; }
  label { font-size: 0.84rem; color: var(--text-dim); display: flex; align-items: center; gap: 7px; }
  .field { background: var(--well); border: 1px solid var(--field-border); border-radius: 7px; padding: 7px 9px; color: var(--text); font-size: 0.86rem; outline: none; width: 64px; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .go { background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px; padding: 8px 16px; font: inherit; font-size: 0.86rem; cursor: pointer; }
  .go:disabled { opacity: .6; cursor: default; }
  .muted { color: var(--text-dim); font-size: 0.78rem; margin: 8px 0 0; }
  .hint { background: var(--warn-bg); border: 1px solid var(--warn-border); color: var(--warn-text); border-radius: 7px; padding: 9px 12px; font-size: 0.85rem; margin: 12px 0; }
  .card { border: 1px solid var(--border); border-radius: 11px; padding: 13px 16px; margin-top: 12px; background: var(--card); }
  .card.ok { border-left: 3px solid var(--good, #3fb950); } .card.bad { border-left: 3px solid var(--danger, #e5484d); }
  .card.warn { border-left: 3px solid var(--warn-text, #f5a623); }
  .card.listening { display: flex; align-items: center; gap: 10px; color: var(--text-dim); font-size: 0.86rem; }
  .ctitle { font-size: 0.95rem; font-weight: 600; color: var(--text-strong); margin-bottom: 6px; }
  .cbody { font-size: 0.84rem; color: var(--text); line-height: 1.5; margin: 0; }
  .chead { display: flex; align-items: baseline; gap: 10px; margin-bottom: 4px; }
  .ssrc { font-size: 0.95rem; font-weight: 650; color: var(--text-strong); }
  .codec { font-size: 0.78rem; color: var(--accent); }
  .flow { font-size: 0.8rem; color: var(--text-dim); margin-bottom: 10px; }
  .stats { display: flex; gap: 10px; flex-wrap: wrap; }
  .stat { border: 1px solid var(--border-soft, var(--border)); border-radius: 8px; padding: 7px 12px; min-width: 90px; }
  .stat.warn { border-color: var(--warn-text, #f5a623); }
  .sl { display: block; font-size: 0.68rem; color: var(--text-dim); }
  .sv { font-size: 0.95rem; color: var(--text-strong); font-weight: 600; }
  .sv small { font-size: 0.68rem; color: var(--text-dim); font-weight: 400; }
  .spinner { width: 13px; height: 13px; border: 2px solid var(--border); border-top-color: var(--accent); border-radius: 50%; display: inline-block; animation: spin 0.8s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }
  .mono { font-family: var(--font-mono); }
</style>
