<script lang="ts">
  import { onDestroy } from "svelte";
  import Gauge from "../metrics/Gauge.svelte";
  import StreamChart from "../metrics/StreamChart.svelte";
  import { subscribe, newStreamID } from "../metrics/metricStream";
  import { StartPingMonitor, StopPingMonitor, SetPingBaseline } from "../../../wailsjs/go/app/App.js";

  let target = "1.1.1.1";
  let running = false;
  let streamID = "";
  let unsub: (() => void) | null = null;
  let error = "";

  let snap: any = null;
  let xs: number[] = [];
  let rtts: (number | null)[] = [];
  let counter = 0;
  const WINDOW = 120;

  async function start() {
    const t = target.trim();
    if (!t || running) return;
    error = "";
    snap = null;
    xs = [];
    rtts = [];
    counter = 0;
    streamID = newStreamID();
    unsub = subscribe(streamID, onSnapshot);
    running = true;
    try {
      await StartPingMonitor(streamID, t);
    } catch (e) {
      error = String(e);
      await stop();
    }
  }

  function onSnapshot(s: any) {
    if (s.state === "error") {
      error = s.note || "monitor error";
      stop(); // the backend monitor has already exited — return to a clean Start state
      return;
    }
    snap = s;
    xs = [...xs, counter++].slice(-WINDOW);
    rtts = [...rtts, s.hasRtt ? s.rttMs : null].slice(-WINDOW);
  }

  async function stop() {
    if (unsub) { unsub(); unsub = null; }
    const id = streamID;
    running = false;
    if (id) { try { await StopPingMonitor(id); } catch {} }
  }

  function baseline() {
    if (running && streamID) SetPingBaseline(streamID);
  }

  onDestroy(() => { if (running) stop(); });

  const fmt = (v: number | undefined) => (v == null ? "–" : v.toFixed(1));
  const clamp = (v: number) => Math.max(0, Math.min(1, v));

  // per-gauge state against the snapshot's known-good band
  $: warming = !snap || snap.warming;
  $: rttState = warming ? "idle" : snap.avgMs > snap.bandHi ? "bad" : "good";
  $: lossState = warming ? "idle" : snap.lossPct >= 10 ? "bad" : snap.lossPct > 0 ? "warn" : "good";
  $: bandW = snap ? Math.max(5, snap.bandHi - snap.bandLo) : 5;
  $: jitState = warming ? "idle" : snap.jitterMs > 2 * bandW ? "bad" : snap.jitterMs > bandW ? "warn" : "good";

  const words: Record<string, [string, string]> = {
    warming: ["WARMING UP", "idle"],
    steady: ["STEADY", "good"],
    degraded: ["DEGRADED", "warn"],
    loss: ["PACKET LOSS", "bad"],
    anomaly: ["ANOMALY", "bad"],
  };
  $: stateWord = snap ? (words[snap.state] ?? ["", "idle"]) : ["IDLE", "idle"];
  $: stateColor = stateWord[1] === "bad" ? "var(--bad)" : stateWord[1] === "warn" ? "var(--warn)" : stateWord[1] === "good" ? "var(--good)" : "var(--text-faint)";
</script>

<div class="tool">
  <p class="lede">
    Live ping — streams the round-trip time and loss from this machine and shows, at a glance,
    whether it's <strong>known-good</strong> or something's off. The chart's shaded band is the
    normal range; spikes and loss are flagged against it. The UI never blocks on the polling.
  </p>

  <form class="row" on:submit|preventDefault={start}>
    <input class="field mono" bind:value={target} placeholder="1.1.1.1  ·  host" aria-label="Target" spellcheck="false" autocomplete="off" />
    {#if running}
      <button type="button" class="go" on:click={stop}>Pause</button>
      <button type="button" class="ghost" on:click={baseline}>Set baseline</button>
    {:else}
      <button type="submit" class="go" disabled={!target.trim()}>Start</button>
    {/if}
  </form>

  {#if error}
    <div class="banner bad"><span class="vlabel">Couldn't monitor</span><span class="vnote">{error}</span></div>
  {/if}

  {#if running || snap}
    <div class="statusbar">
      <span class="dot" style="background:{stateColor}"></span>
      <span class="state-word" style="color:{stateColor}">{stateWord[0]}</span>
      {#if snap && !warming}<span class="state-sub">known-good {fmt(snap.bandLo)}–{fmt(snap.bandHi)} ms</span>{/if}
    </div>

    <div class="gauges">
      <Gauge value={fmt(snap?.avgMs)} unit="ms" label="avg rtt" pct={clamp((snap?.avgMs ?? 0) / 100)} state={rttState} />
      <Gauge value={fmt(snap?.lossPct)} unit="%" label="loss" pct={clamp((snap?.lossPct ?? 0) / 100)} state={lossState} />
      <Gauge value={fmt(snap?.jitterMs)} unit="ms" label="jitter" pct={clamp((snap?.jitterMs ?? 0) / 30)} state={jitState} />
    </div>

    <div class="card">
      <div class="chart-top"><span class="chart-k">rtt · ms</span><span class="chart-sub mono">1s interval</span></div>
      <StreamChart {xs} {rtts} bandLo={snap?.bandLo ?? 0} bandHi={snap?.bandHi ?? 0} />
    </div>
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 20px; }
  .lede { max-width: 820px; margin: 0 0 14px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .mono { font-family: var(--mono, ui-monospace, monospace); }
  .row { display: flex; gap: 8px; max-width: 820px; align-items: center; }
  .field { flex: 1; background: var(--well); border: 1px solid var(--field-border); border-radius: 7px; padding: 9px 12px; color: var(--text); font-size: 0.9rem; outline: none; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .go { background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px; padding: 0 16px; height: 38px; font: inherit; font-size: 0.9rem; cursor: pointer; }
  .go:disabled { opacity: .6; cursor: default; }
  .ghost { background: transparent; border: 1px solid var(--border-soft); color: var(--text-dim); border-radius: 7px; padding: 8px 13px; font: inherit; font-size: 0.82rem; cursor: pointer; }
  .ghost:hover { color: var(--text); border-color: var(--field-border); }

  .banner { margin: 14px 0; padding: 11px 14px; border-radius: 9px; border: 1px solid var(--bad, #ff6b6b); background: color-mix(in srgb, var(--bad,#ff6b6b) 10%, transparent); max-width: 820px; display: flex; flex-direction: column; gap: 3px; }
  .vlabel { font-weight: 600; color: var(--bad, #ff6b6b); }
  .vnote { font-size: 0.8rem; color: var(--text-dim); }

  .statusbar { display: flex; align-items: center; gap: 9px; margin: 16px 0 12px; font-size: 0.82rem; max-width: 820px; }
  .dot { width: 9px; height: 9px; border-radius: 50%; flex: 0 0 auto; }
  .state-word { font-weight: 650; letter-spacing: .02em; }
  .state-sub { color: var(--text-dim); font-size: 0.78rem; }

  .gauges { display: grid; grid-template-columns: repeat(3, 1fr); gap: 12px; margin-bottom: 14px; max-width: 820px; }
  @media (max-width: 560px) { .gauges { grid-template-columns: 1fr; } }

  .card { background: var(--card); border: 1px solid var(--border-soft); border-radius: 11px; padding: 11px 14px; max-width: 820px; }
  .chart-top { display: flex; justify-content: space-between; align-items: baseline; margin-bottom: 6px; }
  .chart-k { font-size: 0.7rem; text-transform: uppercase; letter-spacing: .05em; color: var(--text-faint); }
  .chart-sub { font-size: 0.7rem; color: var(--text-faint); }
</style>
