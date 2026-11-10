<script lang="ts">
  import { onDestroy } from "svelte";
  import Gauge from "../metrics/Gauge.svelte";
  import StreamChart from "../metrics/StreamChart.svelte";
  import { subscribe, newStreamID } from "../metrics/metricStream";
  import {
    ListInterfaces,
    StartIfMonitor,
    StopIfMonitor,
    SetIfBaseline,
  } from "../../../wailsjs/go/app/App.js";
  import type { snmp } from "../../../wailsjs/go/models";

  let device = "";
  let community = "";
  let ifaces: snmp.Interface[] = [];
  let selected = -1; // ifIndex
  let listing = false;
  let listErr = "";

  let running = false;
  let streamID = "";
  let unsub: (() => void) | null = null;
  let error = "";
  let baselined = false;

  let snap: any = null;
  let xs: number[] = [];
  let rx: (number | null)[] = []; // Mbps in
  let tx: (number | null)[] = []; // Mbps out
  let counter = 0;
  const WINDOW = 120;

  $: picked = ifaces.find((i) => i.index === selected);
  $: series = [
    { values: rx, color: "--good" },
    { values: tx, color: "--accent" },
  ];

  async function list() {
    const d = device.trim();
    if (!d || listing) return;
    listErr = "";
    listing = true;
    ifaces = [];
    selected = -1;
    try {
      ifaces = await ListInterfaces(d, community);
      if (ifaces.length) {
        const up = ifaces.find((i) => i.operStatus === "up");
        selected = (up ?? ifaces[0]).index;
      } else {
        listErr = "no interfaces returned — check the device, community and that SNMP is enabled";
      }
    } catch (e) {
      listErr = String(e);
    } finally {
      listing = false;
    }
  }

  async function start() {
    const d = device.trim();
    if (!d || selected < 0 || running) return;
    error = "";
    snap = null;
    baselined = false;
    xs = []; rx = []; tx = []; counter = 0;
    streamID = newStreamID();
    unsub = subscribe(streamID, onSnapshot);
    running = true;
    try {
      await StartIfMonitor(streamID, d, community, selected, picked?.speedBps ?? 0);
    } catch (e) {
      error = String(e);
      await stop();
    }
  }

  function onSnapshot(s: any) {
    if (s.state === "error") {
      error = s.note || "monitor error";
      stop(); // backend goroutine has already exited — return to a clean Start state
      return;
    }
    snap = s;
    xs = [...xs, counter++].slice(-WINDOW);
    rx = [...rx, s.inBps / 1e6].slice(-WINDOW);
    tx = [...tx, s.outBps / 1e6].slice(-WINDOW);
  }

  async function stop() {
    if (unsub) { unsub(); unsub = null; }
    const id = streamID;
    running = false;
    if (id) { try { await StopIfMonitor(id); } catch {} }
  }

  function baseline() {
    if (running && streamID) {
      SetIfBaseline(streamID);
      baselined = true;
    }
  }

  onDestroy(() => { if (running) stop(); });

  const mbps = (bps: number | undefined) => (bps == null ? "–" : (bps / 1e6).toFixed(1));
  const pct = (v: number | undefined) => (v == null ? "–" : v.toFixed(0));
  const clamp = (v: number) => Math.max(0, Math.min(1, v));
  const speedLabel = (b: number) => (b >= 1e9 ? `${(b / 1e9).toFixed(0)}G` : b >= 1e6 ? `${(b / 1e6).toFixed(0)}M` : b > 0 ? `${b}` : "?");

  // gauge states
  $: warming = !snap || snap.warming;
  // known-good band is the normal TOTAL (rx+tx) throughput range, in bps — shown
  // as labelled Mbps text (the chart lines are per-direction, so a shared rect
  // would misread). Frozen once the user sets a baseline.
  $: bandMbps = snap && snap.bandHi > snap.bandLo ? [snap.bandLo / 1e6, snap.bandHi / 1e6] : null;
  $: utilState = !snap?.hasUtil ? "idle" : snap.utilPct > 90 ? "bad" : snap.utilPct > 70 ? "warn" : "good";
  $: cpuState = !snap?.hasCpu ? "idle" : snap.cpuPct > 90 ? "bad" : snap.cpuPct > 75 ? "warn" : "good";
  $: memState = !snap?.hasMem ? "idle" : snap.memPct > 90 ? "bad" : snap.memPct > 80 ? "warn" : "good";

  const words: Record<string, [string, string]> = {
    warming: ["WARMING UP", "idle"],
    steady: ["STEADY", "good"],
    busy: ["BUSY", "warn"],
    saturated: ["SATURATED", "bad"],
    down: ["LINK DOWN", "bad"],
    unreachable: ["UNREACHABLE", "warn"],
    error: ["ERROR", "bad"],
  };
  $: stateWord = snap ? (words[snap.state] ?? ["", "idle"]) : ["IDLE", "idle"];
  $: stateColor = stateWord[1] === "bad" ? "var(--bad)" : stateWord[1] === "warn" ? "var(--warn)" : stateWord[1] === "good" ? "var(--good)" : "var(--text-faint)";
  $: errDelta = (snap?.inErrDelta ?? 0) + (snap?.outErrDelta ?? 0);
</script>

<div class="tool">
  <p class="lede">
    Live interface monitor — polls a switch or router over <strong>SNMPv2c</strong> and streams
    the port's receive/transmit throughput, link utilisation and (where the device exposes them)
    CPU and memory. The community string is read-only and never shown back or written into any
    error. Point it at a device, list its interfaces, pick one, and watch for known-good or anomaly.
  </p>

  <form class="row" on:submit|preventDefault={list}>
    <input class="field mono" bind:value={device} placeholder="10.0.0.1  ·  device address" aria-label="Device" spellcheck="false" autocomplete="off" />
    <input class="field mono short" type="password" bind:value={community} placeholder="community" aria-label="SNMP community" autocomplete="off" />
    <button type="submit" class="go" disabled={!device.trim() || listing || running}>{listing ? "Listing…" : "List"}</button>
  </form>

  {#if listErr}
    <div class="banner bad"><span class="vlabel">Couldn't list interfaces</span><span class="vnote">{listErr}</span></div>
  {/if}

  {#if ifaces.length}
    <div class="row pick">
      <select class="field mono" bind:value={selected} disabled={running} aria-label="Interface">
        {#each ifaces as i}
          <option value={i.index}>
            {i.descr || `if ${i.index}`}{i.speedBps ? ` · ${speedLabel(i.speedBps)}bps` : ""}{i.operStatus && i.operStatus !== "up" ? ` · ${i.operStatus}` : ""}
          </option>
        {/each}
      </select>
      {#if running}
        <button type="button" class="go" on:click={stop}>Pause</button>
        <button type="button" class="ghost" on:click={baseline}>Set baseline</button>
      {:else}
        <button type="button" class="go" on:click={start} disabled={selected < 0}>Start</button>
      {/if}
    </div>
  {/if}

  {#if error}
    <div class="banner bad"><span class="vlabel">Couldn't monitor</span><span class="vnote">{error}</span></div>
  {/if}

  {#if running || snap}
    <div class="statusbar">
      <span class="dot" style="background:{stateColor}"></span>
      <span class="state-word" style="color:{stateColor}">{stateWord[0]}</span>
      {#if picked}<span class="state-sub">{picked.descr || `if ${picked.index}`}{picked.speedBps ? ` · ${speedLabel(picked.speedBps)}bps link` : ""}</span>{/if}
      {#if !warming && bandMbps}<span class="state-sub">known-good{baselined ? " (baseline)" : ""} {bandMbps[0].toFixed(1)}–{bandMbps[1].toFixed(1)} Mbps total</span>{/if}
      {#if errDelta > 0}<span class="state-sub err">+{errDelta} errors</span>{/if}
    </div>

    <div class="gauges">
      <Gauge value={snap?.hasUtil ? pct(snap?.utilPct) : "n/a"} unit={snap?.hasUtil ? "%" : ""} label="link util" pct={clamp((snap?.utilPct ?? 0) / 100)} state={utilState} />
      <Gauge value={snap?.hasCpu ? pct(snap?.cpuPct) : "n/a"} unit={snap?.hasCpu ? "%" : ""} label="cpu" pct={clamp((snap?.cpuPct ?? 0) / 100)} state={cpuState} />
      <Gauge value={snap?.hasMem ? pct(snap?.memPct) : "n/a"} unit={snap?.hasMem ? "%" : ""} label="memory" pct={clamp((snap?.memPct ?? 0) / 100)} state={memState} />
    </div>

    <div class="card">
      <div class="chart-top">
        <span class="chart-k">throughput · Mbps</span>
        <span class="legend mono">
          <span class="lg"><span class="sw rx"></span>rx {mbps(snap?.inBps)}</span>
          <span class="lg"><span class="sw tx"></span>tx {mbps(snap?.outBps)}</span>
        </span>
      </div>
      <StreamChart {xs} {series} />
    </div>
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 20px; }
  .lede { max-width: 820px; margin: 0 0 14px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .mono { font-family: var(--mono, ui-monospace, monospace); }
  .row { display: flex; gap: 8px; max-width: 820px; align-items: center; }
  .row.pick { margin-top: 10px; }
  .field { flex: 1; background: var(--well); border: 1px solid var(--field-border); border-radius: 7px; padding: 9px 12px; color: var(--text); font-size: 0.9rem; outline: none; }
  .field.short { flex: 0 0 160px; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  select.field { cursor: pointer; }
  .go { background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px; padding: 0 16px; height: 38px; font: inherit; font-size: 0.9rem; cursor: pointer; }
  .go:disabled { opacity: .6; cursor: default; }
  .ghost { background: transparent; border: 1px solid var(--border-soft); color: var(--text-dim); border-radius: 7px; padding: 8px 13px; font: inherit; font-size: 0.82rem; cursor: pointer; }
  .ghost:hover { color: var(--text); border-color: var(--field-border); }

  .banner { margin: 14px 0; padding: 11px 14px; border-radius: 9px; border: 1px solid var(--bad, #ff6b6b); background: color-mix(in srgb, var(--bad,#ff6b6b) 10%, transparent); max-width: 820px; display: flex; flex-direction: column; gap: 3px; }
  .vlabel { font-weight: 600; color: var(--bad, #ff6b6b); }
  .vnote { font-size: 0.8rem; color: var(--text-dim); word-break: break-word; }

  .statusbar { display: flex; align-items: center; gap: 9px; margin: 16px 0 12px; font-size: 0.82rem; max-width: 820px; flex-wrap: wrap; }
  .dot { width: 9px; height: 9px; border-radius: 50%; flex: 0 0 auto; }
  .state-word { font-weight: 650; letter-spacing: .02em; }
  .state-sub { color: var(--text-dim); font-size: 0.78rem; }
  .state-sub.err { color: var(--bad); }

  .gauges { display: grid; grid-template-columns: repeat(3, 1fr); gap: 12px; margin-bottom: 14px; max-width: 820px; }
  @media (max-width: 560px) { .gauges { grid-template-columns: 1fr; } }

  .card { background: var(--card); border: 1px solid var(--border-soft); border-radius: 11px; padding: 11px 14px; max-width: 820px; }
  .chart-top { display: flex; justify-content: space-between; align-items: baseline; margin-bottom: 6px; }
  .chart-k { font-size: 0.7rem; text-transform: uppercase; letter-spacing: .05em; color: var(--text-faint); }
  .legend { display: flex; gap: 12px; font-size: 0.7rem; color: var(--text-dim); }
  .lg { display: inline-flex; align-items: center; gap: 5px; }
  .sw { width: 9px; height: 3px; border-radius: 2px; display: inline-block; }
  .sw.rx { background: var(--good); }
  .sw.tx { background: var(--accent); }
</style>
