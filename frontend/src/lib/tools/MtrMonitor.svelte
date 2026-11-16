<script lang="ts">
  import { onDestroy } from "svelte";
  import { newStreamID } from "../metrics/metricStream";
  import { EventsOn, EventsOff } from "../../../wailsjs/runtime/runtime.js";
  import { StartMTR, StopMTR } from "../../../wailsjs/go/app/App.js";
  import ToolHelp from "../ToolHelp.svelte";

  let target = "cloudflare.com";
  let useV6 = false;
  let running = false;
  let error = "";
  let streamID = "";
  let round = 0;
  let destAddr = "";
  let reached = false;
  let hops: any[] = [];

  function ms(ns: number): string {
    if (!ns || ns <= 0) return "—";
    return (ns / 1e6).toFixed(1);
  }

  // Loss drives a red tint: 0% = calm, 100% = alarm.
  function lossColor(pct: number): string {
    if (pct <= 0) return "var(--text-dim)";
    if (pct < 20) return "var(--warn-text, #b8860b)";
    return "var(--danger, #e5484d)";
  }

  async function start() {
    const t = target.trim();
    if (!t || running) return;
    error = "";
    hops = [];
    round = 0;
    destAddr = "";
    reached = false;
    streamID = newStreamID();
    const chan = "mtr:" + streamID;
    EventsOn(chan, onUpdate);
    running = true;
    try {
      await StartMTR(streamID, t, useV6);
    } catch (e) {
      error = String(e);
      await stop();
    }
  }

  function onUpdate(u: any) {
    // The backend signals a fatal error by putting "error: …" in addr with no hops.
    if (u.addr && String(u.addr).startsWith("error:") && (!u.hops || !u.hops.length)) {
      error = String(u.addr).slice(6).trim();
      stop();
      return;
    }
    round = u.round;
    destAddr = u.addr || "";
    reached = !!u.reached;
    hops = u.hops || [];
  }

  async function stop() {
    const chan = "mtr:" + streamID;
    EventsOff(chan);
    const id = streamID;
    running = false;
    if (id) { try { await StopMTR(id); } catch {} }
  }

  onDestroy(() => {
    if (streamID) EventsOff("mtr:" + streamID);
    if (running) StopMTR(streamID);
  });
</script>

<div class="tool">
  <h2>MTR — live path quality</h2>
  <p class="sub">
    Traceroute that keeps running: each round re-probes every hop and accumulates
    <strong>loss and latency per hop</strong>, so you can see <em>where</em> a path degrades, not just that it does.
    No admin rights needed — uses ICMP echo. Live only; nothing is stored.
  </p>

  <ToolHelp>
    Read the <strong>Loss%</strong> column down the path. The hop where loss <strong>first appears and stays high through
    every later hop</strong> is the real trouble spot. Loss at one middle hop that <strong>clears on the hops below it</strong>
    is harmless — that router just deprioritizes replying to pings, it isn't dropping your traffic. Compare
    <strong>Avg</strong> against <strong>Worst</strong>: a big gap means jitter (unstable latency). Let it run a while —
    the stats are cumulative, so more rounds make the numbers trustworthy.
  </ToolHelp>

  <form on:submit|preventDefault={start} class="row">
    <input class="field mono" bind:value={target} placeholder="cloudflare.com  or  104.16.123.45" autocomplete="off" spellcheck="false" aria-label="Target host" />
    <select class="sel" bind:value={useV6} aria-label="IP version" disabled={running}>
      <option value={false}>IPv4</option>
      <option value={true}>IPv6</option>
    </select>
    {#if running}
      <button type="button" class="go stop" on:click={stop}>Stop</button>
    {:else}
      <button class="go">Start</button>
    {/if}
  </form>

  {#if error}<div class="hint">{error}</div>{/if}

  {#if hops.length}
    <div class="meta">
      <span class="dest mono">{target}{#if destAddr && destAddr !== target} <span class="dim">({destAddr})</span>{/if}</span>
      <span class="counter">round {round}</span>
      {#if running}<span class="live">● live</span>{:else}<span class="dim">stopped</span>{/if}
      {#if reached}<span class="reached">reached</span>{/if}
    </div>

    <table class="tbl">
      <thead>
        <tr>
          <th>#</th><th>Host</th><th class="num">Loss%</th><th class="num">Snt</th>
          <th class="num">Last</th><th class="num">Avg</th><th class="num">Best</th><th class="num">Worst</th>
        </tr>
      </thead>
      <tbody>
        {#each hops as h}
          <tr>
            <td class="mono">{h.ttl}</td>
            <td class="host mono">
              {#if h.addr}
                {#if h.host && h.host !== h.addr}{h.host} <span class="dim">({h.addr})</span>{:else}{h.addr}{/if}
                {#if h.asn}<span class="asn mono">{h.asn}</span>{/if}
              {:else}
                <span class="star">???</span>
              {/if}
            </td>
            <td class="num mono" style="color:{lossColor(h.lossPct)}">{h.lossPct.toFixed(1)}</td>
            <td class="num mono dim">{h.sent}</td>
            <td class="num mono">{h.recv ? ms(h.last) : "—"}</td>
            <td class="num mono">{h.recv ? ms(h.avg) : "—"}</td>
            <td class="num mono dim">{h.recv ? ms(h.best) : "—"}</td>
            <td class="num mono dim">{h.recv ? ms(h.worst) : "—"}</td>
          </tr>
        {/each}
      </tbody>
    </table>
    <p class="muted">Times in ms. Loss is cumulative since you pressed Start.</p>
  {:else if running}
    <p class="muted">Probing the path…</p>
  {/if}
</div>

<style>
  .tool { padding: 16px 20px 40px; max-width: 1000px; }
  h2 { margin: 0 0 2px; font-size: 1.15rem; font-weight: 600; color: var(--text-strong); }
  .sub { margin: 0 0 15px; color: var(--text-dim); font-size: 0.86rem; line-height: 1.5; }
  .row { display: flex; gap: 8px; margin-bottom: 14px; flex-wrap: wrap; }
  .field { flex: 1; min-width: 200px; background: var(--well); border: 1px solid var(--field-border); border-radius: 7px; padding: 9px 12px; color: var(--text); font-size: 0.9rem; outline: none; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .sel { background: var(--well); border: 1px solid var(--field-border); border-radius: 7px; padding: 9px 10px; color: var(--text); font: inherit; font-size: 0.86rem; cursor: pointer; }
  .sel:disabled { opacity: .6; cursor: default; }
  .go { background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px; padding: 0 16px; font: inherit; font-size: 0.9rem; cursor: pointer; }
  .go.stop { background: var(--warn-bg); border-color: var(--warn-border); color: var(--warn-text); }
  .hint { background: var(--warn-bg); border: 1px solid var(--warn-border); color: var(--warn-text); border-radius: 7px; padding: 9px 12px; font-size: 0.85rem; margin-bottom: 12px; }
  .meta { display: flex; align-items: center; gap: 12px; margin-bottom: 10px; font-size: 0.8rem; flex-wrap: wrap; }
  .dest { color: var(--text-strong); }
  .counter { color: var(--text-dim); }
  .live { color: var(--good, #3fb950); font-size: 0.74rem; }
  .reached { color: var(--accent); font-size: 0.72rem; border: 1px solid var(--pill-border); border-radius: 8px; padding: 0 7px; }
  table.tbl { width: 100%; border-collapse: collapse; font-size: 0.82rem; }
  .tbl th { text-align: left; color: var(--text-dim); font-weight: 500; font-size: 0.68rem; text-transform: uppercase; letter-spacing: .04em; padding: 7px 10px; border-bottom: 1px solid var(--border); }
  .tbl th.num, .tbl td.num { text-align: right; }
  .tbl td { padding: 7px 10px; border-bottom: 1px solid var(--border-soft); vertical-align: middle; }
  .asn { color: var(--accent); font-size: 0.72rem; margin-left: 5px; }
  .dim { color: var(--text-dim); } .star { color: var(--text-faint); }
  .muted { color: var(--text-dim); font-size: 0.8rem; margin: 10px 0 0; }
  .mono { font-family: var(--font-mono); }
</style>
