<script lang="ts">
  import { onDestroy } from "svelte";
  import Gauge from "../metrics/Gauge.svelte";
  import StreamChart from "../metrics/StreamChart.svelte";
  import { subscribe, newStreamID } from "../metrics/metricStream";
  import { SpeedtestStart, SpeedtestStop } from "../../../wailsjs/go/app/App.js";

  let host = "";
  let port = 5201;
  let direction = "download";
  let parallel = 4;
  let duration = 10;
  let udp = false;

  let running = false;
  let streamID = "";
  let unsub: (() => void) | null = null;
  let error = "";

  let snap: any = null;
  let result: any = null;
  let xs: number[] = [];
  let dnVals: (number | null)[] = [];
  let upVals: (number | null)[] = [];
  let counter = 0;
  let peak = 1;
  const WINDOW = 240;

  $: showDown = direction === "download" || direction === "bidir";
  $: showUp = direction === "upload" || direction === "bidir";

  async function start() {
    const h = host.trim();
    if (!h || running) return;
    error = ""; snap = null; result = null;
    xs = []; dnVals = []; upVals = []; counter = 0; peak = 1;
    streamID = newStreamID();
    unsub = subscribe(streamID, onSnapshot);
    running = true;
    try {
      await SpeedtestStart(streamID, h, Number(port) || 5201, direction, Number(parallel) || 1, Number(duration) || 10, udp);
    } catch (e) {
      error = String(e);
      await stop();
    }
  }

  function onSnapshot(s: any) {
    if (s.state === "error") {
      error = s.note || "speed test failed";
      stop();
      return;
    }
    if (s.state === "done") {
      result = s;
      stop();
      return;
    }
    // running
    snap = s;
    const d = showDown ? s.downMbps : 0;
    const u = showUp ? s.upMbps : 0;
    peak = Math.max(peak, d, u, 1);
    xs = [...xs, counter++].slice(-WINDOW);
    dnVals = [...dnVals, showDown ? d : null].slice(-WINDOW);
    upVals = [...upVals, showUp ? u : null].slice(-WINDOW);
  }

  async function stop() {
    if (unsub) { unsub(); unsub = null; }
    const id = streamID;
    running = false;
    if (id) { try { await SpeedtestStop(id); } catch {} }
  }

  onDestroy(() => { if (running) stop(); });

  const clamp = (v: number) => Math.max(0, Math.min(1, v));
  function fmtRate(mbps: number | undefined): [string, string] {
    const v = mbps ?? 0;
    if (v >= 1000) return [(v / 1000).toFixed(2), "Gbit/s"];
    return [v.toFixed(v >= 100 ? 0 : 1), "Mbit/s"];
  }
  const fmtBytes = (b: number) => (b >= 1e9 ? (b / 1e9).toFixed(2) + " GB" : (b / 1e6).toFixed(1) + " MB");
  const lossPct = (r: any) => (r && r.packets > 0 ? (100 * r.lost) / r.packets : 0);

  $: curDownR = fmtRate(snap?.downMbps);
  $: curUpR = fmtRate(snap?.upMbps);
  // StreamChart resolves a bare "--token" to its computed value; a full var()
  // string would reach the canvas unresolved and draw in the grid color.
  $: series = [
    ...(showDown ? [{ values: dnVals, color: "--accent" }] : []),
    ...(showUp ? [{ values: upVals, color: "--good" }] : []),
  ];
</script>

<div class="tool">
  <p class="lede">
    LAN speed test — measures throughput to another host on your network running
    <span class="mono">iperf3 -s</span>. Pick a direction and watch the live rate; the result shows the
    average, retransmits and RTT. Pure-Go client — nothing to install here, and Meridian never serves.
  </p>

  <div class="card span">
    <div class="grid">
      <label class="f"><span class="k0">Host (iperf3 -s)</span>
        <input class="field mono" bind:value={host} placeholder="192.168.1.10  ·  host" spellcheck="false" autocomplete="off" disabled={running} /></label>
      <label class="f sm"><span class="k0">Port</span>
        <input class="field mono" type="number" bind:value={port} min="1" max="65535" disabled={running} /></label>
      <label class="f sm"><span class="k0">Direction</span>
        <select class="field" bind:value={direction} disabled={running}>
          <option value="download">Download</option>
          <option value="upload">Upload</option>
          <option value="bidir">Both</option>
        </select></label>
      <label class="f sm"><span class="k0">Streams</span>
        <input class="field mono" type="number" bind:value={parallel} min="1" max="128" disabled={running} /></label>
      <label class="f sm"><span class="k0">Seconds</span>
        <input class="field mono" type="number" bind:value={duration} min="1" max="60" disabled={running} /></label>
      <label class="f chk"><input type="checkbox" bind:checked={udp} disabled={running} /> <span>UDP</span></label>
    </div>
    <div class="row">
      {#if running}
        <button type="button" class="stop" on:click={stop}>Stop</button>
      {:else}
        <button type="button" class="go" on:click={start} disabled={!host.trim()}>Start test</button>
      {/if}
      <span class="hint">The other end needs <span class="mono">iperf3 -s</span> listening.</span>
    </div>
  </div>

  {#if error}
    <div class="banner bad"><span class="vlabel">Couldn't run the test</span><span class="vnote">{error}</span></div>
  {/if}

  {#if running && snap}
    <div class="gauges">
      {#if showDown}
        <Gauge value={curDownR[0]} unit={curDownR[1]} label="download" pct={clamp((snap?.downMbps ?? 0) / peak)} state="good" />
      {/if}
      {#if showUp}
        <Gauge value={curUpR[0]} unit={curUpR[1]} label="upload" pct={clamp((snap?.upMbps ?? 0) / peak)} state="good" />
      {/if}
    </div>
    <div class="card span">
      <div class="chart-top"><span class="chart-k">throughput · Mbit/s</span><span class="chart-sub mono">{(snap?.elapsed ?? 0).toFixed(0)}s elapsed</span></div>
      <StreamChart {xs} {series} height={160} />
    </div>
  {/if}

  {#if result}
    <div class="card span result">
      <div class="k">Result <span class="count mono">· {result.peer || host} · {result.protocol || "tcp"}</span></div>
      <div class="rgrid">
        {#if showDown}<div class="rt"><div class="rn mono">{fmtRate(result.avgDownMbps)[0]}<span class="ru">{fmtRate(result.avgDownMbps)[1]}</span></div><div class="rl">avg download</div></div>{/if}
        {#if showUp}<div class="rt"><div class="rn mono">{fmtRate(result.avgUpMbps)[0]}<span class="ru">{fmtRate(result.avgUpMbps)[1]}</span></div><div class="rl">avg upload</div></div>{/if}
        <div class="rt"><div class="rn mono">{fmtBytes(result.bytes)}</div><div class="rl">transferred</div></div>
        {#if !udp}<div class="rt"><div class="rn mono">{result.retransmits < 0 ? "–" : result.retransmits}</div><div class="rl">retransmits</div></div>{/if}
        {#if result.rttMeanMs > 0}<div class="rt"><div class="rn mono">{result.rttMeanMs.toFixed(2)}<span class="ru">ms</span></div><div class="rl">mean RTT</div></div>{/if}
        {#if udp}<div class="rt"><div class="rn mono">{result.jitterMs.toFixed(2)}<span class="ru">ms</span></div><div class="rl">jitter</div></div>
        <div class="rt"><div class="rn mono">{lossPct(result).toFixed(2)}<span class="ru">%</span></div><div class="rl">loss</div></div>{/if}
      </div>
    </div>
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 20px; }
  .lede { max-width: 820px; margin: 0 0 14px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .mono { font-family: var(--mono, ui-monospace, monospace); }
  .card { background: var(--card); border: 1px solid var(--border-soft); border-radius: 11px; padding: 13px 15px; max-width: 820px; margin-bottom: 14px; }

  .grid { display: flex; flex-wrap: wrap; gap: 10px; }
  .f { display: flex; flex-direction: column; gap: 5px; flex: 1 1 140px; min-width: 0; }
  .f.sm { flex: 0 0 90px; }
  .f.chk { flex: 0 0 auto; flex-direction: row; align-items: center; gap: 6px; align-self: flex-end; height: 36px; color: var(--text-dim); font-size: 0.84rem; }
  .k0 { font-size: 0.68rem; text-transform: uppercase; letter-spacing: .05em; color: var(--text-faint); }
  .field { background: var(--well); border: 1px solid var(--field-border); border-radius: 7px; padding: 8px 11px; color: var(--text); font-size: 0.86rem; outline: none; width: 100%; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .field:disabled { opacity: .6; }

  .row { display: flex; gap: 10px; margin-top: 12px; align-items: center; flex-wrap: wrap; }
  .go { background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px; padding: 0 16px; height: 38px; font: inherit; font-size: 0.9rem; cursor: pointer; }
  .go:disabled { opacity: .6; cursor: default; }
  .stop { background: transparent; border: 1px solid var(--border-soft); color: var(--text-dim); border-radius: 7px; padding: 0 15px; height: 38px; font: inherit; font-size: 0.86rem; cursor: pointer; }
  .hint { font-size: 0.75rem; color: var(--text-faint); }

  .banner { margin: 0 0 14px; padding: 11px 14px; border-radius: 9px; border: 1px solid var(--bad, #ff6b6b); background: color-mix(in srgb, var(--bad,#ff6b6b) 10%, transparent); max-width: 820px; display: flex; flex-direction: column; gap: 3px; }
  .vlabel { font-weight: 600; color: var(--bad, #ff6b6b); }
  .vnote { font-size: 0.8rem; color: var(--text-dim); overflow-wrap: anywhere; }

  .gauges { display: grid; grid-template-columns: repeat(2, 1fr); gap: 12px; margin-bottom: 14px; max-width: 820px; }
  @media (max-width: 560px) { .gauges { grid-template-columns: 1fr; } }

  .chart-top { display: flex; justify-content: space-between; align-items: baseline; margin-bottom: 6px; }
  .chart-k { font-size: 0.7rem; text-transform: uppercase; letter-spacing: .05em; color: var(--text-faint); }
  .chart-sub { font-size: 0.7rem; color: var(--text-faint); }

  .k { font-size: 0.74rem; text-transform: uppercase; letter-spacing: .04em; color: var(--text-faint); margin-bottom: 12px; }
  .k .count { color: var(--text-dim); text-transform: none; letter-spacing: 0; }
  .rgrid { display: grid; grid-template-columns: repeat(auto-fit, minmax(120px, 1fr)); gap: 14px; }
  .rn { font-size: 1.4rem; font-weight: 650; color: var(--text-strong); font-variant-numeric: tabular-nums; }
  .ru { font-size: 0.72rem; color: var(--text-faint); margin-left: 4px; font-weight: 500; }
  .rl { font-size: 0.66rem; text-transform: uppercase; letter-spacing: .04em; color: var(--text-faint); margin-top: 3px; }
</style>
