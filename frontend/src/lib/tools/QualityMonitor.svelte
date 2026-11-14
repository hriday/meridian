<script lang="ts">
  import { onDestroy } from "svelte";
  import StreamChart from "../metrics/StreamChart.svelte";
  import { newStreamID } from "../metrics/metricStream";
  import { EventsOn, EventsOff } from "../../../wailsjs/runtime/runtime.js";
  import { StartQualityMonitor, StopQualityMonitor } from "../../../wailsjs/go/app/App.js";

  let target = "1.1.1.1";
  let running = false;
  let error = "";
  let streamID = "";
  let snap: any = null;

  let xs: number[] = [];
  let rtts: (number | null)[] = [];
  let counter = 0;
  const WINDOW = 120;

  async function start() {
    const t = target.trim();
    if (!t || running) return;
    error = ""; snap = null; xs = []; rtts = []; counter = 0;
    streamID = newStreamID();
    EventsOn("quality:" + streamID, onSnap);
    running = true;
    try { await StartQualityMonitor(streamID, t); }
    catch (e: any) { error = String(e?.message ?? e); await stop(); }
  }

  function onSnap(s: any) {
    if (s.state === "error") { error = s.note || "monitor error"; stop(); return; }
    snap = s;
    xs = [...xs, counter++].slice(-WINDOW);
    rtts = [...rtts, s.hasRtt ? s.rttMs : null].slice(-WINDOW);
  }

  async function stop() {
    EventsOff("quality:" + streamID);
    const id = streamID;
    running = false;
    if (id) { try { await StopQualityMonitor(id); } catch {} }
  }

  const ratingClass = (r: string) =>
    ({ Excellent: "ex", Good: "gd", Fair: "fa", Poor: "po", Bad: "bd" } as any)[r] || "";

  onDestroy(() => { EventsOff("quality:" + streamID); if (running) StopQualityMonitor(streamID); });
</script>

<div class="tool">
  <h2>Connection quality</h2>
  <p class="sub">
    Continuously measures <strong>latency, jitter and packet loss</strong> and estimates a <strong>MOS</strong> (Mean
    Opinion Score, 1–5) — the standard VoIP quality metric — to answer "is this link good enough for a voice/video call?"
    No admin rights needed. Live only; nothing stored.
  </p>

  <form on:submit|preventDefault={start} class="row">
    <input class="field mono" bind:value={target} placeholder="1.1.1.1  or  teams.microsoft.com" spellcheck="false" aria-label="Target host" />
    {#if running}<button type="button" class="go stop" on:click={stop}>Stop</button>{:else}<button class="go">Measure</button>{/if}
  </form>

  {#if error}<div class="hint">{error}</div>{/if}

  {#if snap}
    <div class="grid">
      <div class="moscard {ratingClass(snap.rating)}">
        <div class="mosval">{(snap.mos ?? 0).toFixed(1)}<span class="of">/5</span></div>
        <div class="rating">{snap.rating}</div>
        <div class="verdict">{snap.summary}</div>
      </div>
      <div class="stats">
        <div class="stat"><span class="sl">Latency</span><span class="sv mono">{snap.hasRtt ? (snap.avgMs ?? 0).toFixed(1) : "—"} <small>ms avg</small></span></div>
        <div class="stat" class:warn={snap.limiting === "jitter"}><span class="sl">Jitter</span><span class="sv mono">{(snap.jitterMs ?? 0).toFixed(1)} <small>ms</small></span></div>
        <div class="stat" class:warn={snap.limiting === "loss"}><span class="sl">Packet loss</span><span class="sv mono">{(snap.lossPct ?? 0).toFixed(0)}<small>%</small></span></div>
      </div>
    </div>

    <div class="chart"><StreamChart {xs} {rtts} bandLo={snap?.bandLo ?? 0} bandHi={snap?.bandHi ?? 0} /></div>
    {#if snap.limiting && snap.limiting !== "none"}
      <p class="limit">Limiting factor: <strong>{snap.limiting}</strong></p>
    {/if}
    <p class="disc">MOS is an estimate from the ITU-T E-model (latency + jitter + loss) — a guide, not a guarantee.</p>
  {/if}
</div>

<style>
  .tool { padding: 16px 20px 40px; max-width: 900px; }
  h2 { margin: 0 0 2px; font-size: 1.15rem; font-weight: 600; color: var(--text-strong); }
  .sub { margin: 0 0 15px; color: var(--text-dim); font-size: 0.86rem; line-height: 1.5; }
  .row { display: flex; gap: 8px; margin-bottom: 14px; flex-wrap: wrap; }
  .field { flex: 1; min-width: 200px; background: var(--well); border: 1px solid var(--field-border); border-radius: 7px; padding: 9px 12px; color: var(--text); font-size: 0.9rem; outline: none; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .go { background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px; padding: 0 18px; font: inherit; font-size: 0.9rem; cursor: pointer; }
  .go.stop { background: var(--warn-bg); border-color: var(--warn-border); color: var(--warn-text); }
  .hint { background: var(--warn-bg); border: 1px solid var(--warn-border); color: var(--warn-text); border-radius: 7px; padding: 9px 12px; font-size: 0.85rem; margin-bottom: 12px; }
  .grid { display: grid; grid-template-columns: minmax(200px, 1fr) 1.2fr; gap: 14px; margin-bottom: 14px; }
  @media (max-width: 640px) { .grid { grid-template-columns: 1fr; } }
  .moscard { border: 1px solid var(--border); border-left-width: 4px; border-radius: 12px; padding: 16px 18px; background: var(--card); }
  .moscard.ex { border-left-color: var(--good, #3fb950); }
  .moscard.gd { border-left-color: #74c365; }
  .moscard.fa { border-left-color: var(--warn-text, #f5a623); }
  .moscard.po { border-left-color: #e8833a; }
  .moscard.bd { border-left-color: var(--danger, #e5484d); }
  .mosval { font-size: 2.6rem; font-weight: 700; color: var(--text-strong); line-height: 1; }
  .of { font-size: 1rem; color: var(--text-dim); font-weight: 400; }
  .rating { font-size: 0.95rem; font-weight: 650; margin-top: 4px; color: var(--text-strong); }
  .verdict { font-size: 0.82rem; color: var(--text-dim); margin-top: 6px; line-height: 1.4; }
  .stats { display: flex; flex-direction: column; gap: 10px; justify-content: center; }
  .stat { display: flex; align-items: baseline; justify-content: space-between; border: 1px solid var(--border-soft, var(--border)); border-radius: 8px; padding: 9px 13px; background: var(--well); }
  .stat.warn { border-color: var(--warn-text, #f5a623); }
  .sl { font-size: 0.78rem; color: var(--text-dim); }
  .sv { font-size: 1.05rem; color: var(--text-strong); font-weight: 600; }
  .sv small { font-size: 0.68rem; color: var(--text-dim); font-weight: 400; }
  .chart { margin-bottom: 8px; }
  .limit { font-size: 0.82rem; color: var(--text-dim); margin: 6px 0 0; }
  .disc { font-size: 0.72rem; color: var(--text-faint, var(--text-dim)); margin: 10px 0 0; }
  .mono { font-family: var(--font-mono); }
</style>
