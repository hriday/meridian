<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { NetflowStart, NetflowStop, NetflowTop } from "../../../wailsjs/go/app/App.js";

  const POLL_MS = 1500;
  const windows = [
    { label: "30s", v: 30 },
    { label: "1m", v: 60 },
    { label: "5m", v: 300 },
  ];

  let port = 2055;
  let running = false;
  let windowSec = 60;
  let byPackets = false;
  let view: any = { top: [], flows: 0, exporters: 0, windowSecCovered: 0 };
  let err = "";
  let timer: any = null;
  let polling = false;

  async function poll() {
    if (polling) return;
    polling = true;
    try {
      const v = await NetflowTop(windowSec, byPackets);
      view = v;
      running = v.running;
      if (v.port) port = v.port;
    } catch {
      /* transient */
    } finally {
      polling = false;
    }
  }

  async function start() {
    err = "";
    try {
      await NetflowStart(port);
    } catch (e) {
      err = String(e);
      running = false;
      return;
    }
    await poll();
  }

  async function stop() {
    try { await NetflowStop(); } catch {}
    running = false;
    view = { top: [], flows: 0, exporters: 0, windowSecCovered: 0 };
  }

  function setWindow(v: number) { windowSec = v; poll(); }
  function setSort(p: boolean) { byPackets = p; poll(); }

  onMount(() => { poll(); timer = setInterval(poll, POLL_MS); });
  onDestroy(() => { if (timer) clearInterval(timer); });

  const human = (b: number) =>
    b >= 1e9 ? (b / 1e9).toFixed(1) + " GB" : b >= 1e6 ? (b / 1e6).toFixed(1) + " MB" : b >= 1e3 ? (b / 1e3).toFixed(1) + " KB" : b + " B";
  const hpkts = (p: number) => (p >= 1e6 ? (p / 1e6).toFixed(1) + "M" : p >= 1e3 ? (p / 1e3).toFixed(1) + "k" : "" + p);

  $: top = view.top || [];
  $: maxV = Math.max(1, top.length ? (byPackets ? top[0].packets : top[0].bytes) : 1);
  $: truncated = running && view.truncated;
</script>

<div class="tool">
  <p class="lede">
    NetFlow — receives flow exports from your routers and switches and shows the top talkers right now:
    who's moving the most traffic, to where, over what. Point an exporter at this machine, pick a window,
    and the heavy conversations rise to the top. In-memory and ephemeral — a quick look on a laptop, not a
    flow warehouse.
  </p>
  <p class="help">
    Set your device's NetFlow/IPFIX export destination to this machine on the <strong>Listen UDP</strong>
    port (2055 is common; also 9995/9996), then <strong>Start</strong> — these ports need no admin. Each row
    is a <strong>conversation</strong> (source → destination:port) aggregated over the window, ranked by
    bytes or packets. NetFlow <strong>v5</strong> today; v9/IPFIX next. Read-only; stops on Stop.
  </p>

  <div class="row">
    <span class="lbl">Listen UDP</span>
    <input class="field port mono" type="number" min="1" max="65535" bind:value={port} disabled={running} aria-label="Listen port" />
    {#if running}
      <button type="button" class="stop" on:click={stop}>Stop</button>
    {:else}
      <button type="button" class="go" on:click={start}>Start</button>
    {/if}
    <span class="spacer"></span>
    <div class="seg">
      {#each windows as w}<button class:on={windowSec === w.v} on:click={() => setWindow(w.v)}>{w.label}</button>{/each}
    </div>
    <div class="seg">
      <button class:on={!byPackets} on:click={() => setSort(false)}>by bytes</button>
      <button class:on={byPackets} on:click={() => setSort(true)}>by packets</button>
    </div>
  </div>

  {#if err}
    <div class="banner bad"><span class="vlabel">Couldn't start</span><span class="vnote">{err}</span></div>
  {/if}

  <div class="statusbar">
    <span class="dot" class:on={running}></span>
    <span>{running ? `listening on :${port}` : "stopped"}</span>
    {#if running}
      <span>·</span>
      <span><span class="count">{view.flows.toLocaleString()}</span> flows from <span class="count">{view.exporters}</span> exporter{view.exporters === 1 ? "" : "s"}{truncated ? ` · covers last ${view.windowSecCovered}s` : ""}</span>
    {/if}
  </div>

  {#if top.length}
    <div class="grid">
      <div class="head"><span>Conversation (src → dst:port)</span><span>Proto</span><span>{byPackets ? "Packets" : "Bytes"}</span><span class="r">{byPackets ? "Bytes" : "Packets"}</span><span class="r">Flows</span></div>
      {#each top as c}
        <div class="drow">
          <div class="conv mono">{c.srcIp} <span class="arrow">→</span> {c.dstIp}:{c.dstPort}</div>
          <div><span class="proto">{c.proto}</span></div>
          <div class="bytes">
            <div class="bar"><div class="fill" style="width:{Math.max(2, ((byPackets ? c.packets : c.bytes) / maxV) * 100)}%"></div></div>
            <span class="bval mono">{byPackets ? hpkts(c.packets) : human(c.bytes)}</span>
          </div>
          <div class="sec mono r">{byPackets ? human(c.bytes) : hpkts(c.packets)}</div>
          <div class="fl mono r">{c.flows}</div>
        </div>
      {/each}
    </div>
    <p class="caption">Top talkers over the last {windows.find((w) => w.v === windowSec)?.label} · aggregated by src→dst:port/proto · in-memory, cleared on stop</p>
  {:else if running}
    <p class="caption">Waiting for flows on :{port} — point an exporter here and traffic will appear.</p>
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 20px; }
  .lede { max-width: 960px; margin: 0 0 12px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .help { max-width: 960px; margin: 0 0 14px; padding: 9px 13px; border-left: 2px solid var(--accent); background: var(--well); border-radius: 0 7px 7px 0; font-size: 0.78rem; color: var(--text-dim); line-height: 1.5; }
  .help strong { color: var(--text); font-weight: 600; }
  .mono { font-family: var(--mono, ui-monospace, monospace); }

  .row { display: flex; gap: 8px; align-items: center; max-width: 960px; }
  .lbl { font-size: 0.78rem; color: var(--text-dim); }
  .field { background: var(--well); border: 1px solid var(--field-border); border-radius: 7px; padding: 8px 11px; color: var(--text); font-size: 0.86rem; outline: none; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .field:disabled { opacity: .6; }
  .port { width: 92px; }
  .spacer { flex: 1; }
  .go { background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px; padding: 0 15px; height: 36px; font: inherit; font-size: 0.86rem; cursor: pointer; }
  .stop { background: transparent; border: 1px solid var(--border-soft); color: var(--text-dim); border-radius: 7px; padding: 0 14px; height: 36px; font: inherit; font-size: 0.84rem; cursor: pointer; }
  .seg { display: flex; border: 1px solid var(--field-border); border-radius: 7px; overflow: hidden; }
  .seg button { background: transparent; border: none; color: var(--text-dim); padding: 7px 11px; font: inherit; font-size: 0.78rem; cursor: pointer; }
  .seg button.on { background: var(--active); color: var(--text-strong); }

  .banner { margin: 12px 0 0; padding: 10px 13px; border-radius: 9px; border: 1px solid var(--bad, #ff6b6b); background: color-mix(in srgb, var(--bad,#ff6b6b) 10%, transparent); max-width: 960px; display: flex; flex-direction: column; gap: 2px; }
  .vlabel { font-weight: 600; color: var(--bad, #ff6b6b); font-size: 0.82rem; }
  .vnote { font-size: 0.8rem; color: var(--text-dim); }

  .statusbar { display: flex; align-items: center; gap: 9px; margin: 13px 0 8px; max-width: 960px; font-size: 0.8rem; color: var(--text-dim); }
  .dot { width: 8px; height: 8px; border-radius: 50%; background: var(--text-faint); flex: none; }
  .dot.on { background: var(--good); }
  .count { color: var(--text-strong); font-weight: 600; }

  .grid { max-width: 960px; border: 1px solid var(--border-soft); border-radius: 11px; overflow: hidden; background: var(--card); }
  .head, .drow { display: grid; grid-template-columns: 2.6fr 0.7fr 2.2fr 0.9fr 0.7fr; align-items: center; gap: 14px; padding: 9px 16px; }
  .head { font-size: 0.6rem; text-transform: uppercase; letter-spacing: .08em; color: var(--text-faint); border-bottom: 1px solid var(--border-soft); background: var(--panel); }
  .drow { border-top: 1px solid var(--border-soft); font-size: 0.82rem; }
  .drow:first-of-type { border-top: none; }
  .conv { color: var(--text-strong); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .arrow { color: var(--text-faint); }
  .proto { font-size: 0.64rem; padding: 1px 7px; border-radius: 4px; border: 1px solid var(--pill-border); color: var(--text-dim); }
  .bytes { display: flex; align-items: center; gap: 9px; }
  .bar { flex: 1; height: 7px; border-radius: 4px; background: var(--track, #2a3142); overflow: hidden; }
  .fill { height: 100%; border-radius: 4px; background: var(--accent); }
  .bval { width: 64px; text-align: right; }
  .sec, .fl { color: var(--text-dim); }
  .r { text-align: right; }
  .caption { max-width: 960px; margin-top: 12px; font-size: 0.72rem; color: var(--text-faint); }
</style>
