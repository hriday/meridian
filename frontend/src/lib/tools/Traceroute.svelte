<script lang="ts">
  import { onDestroy } from "svelte";
  import { Traceroute, StopTrace } from "../../../wailsjs/go/app/App.js";
  import { EventsOn, EventsOff } from "../../../wailsjs/runtime/runtime.js";

  let target = "cloudflare.com";
  let useV6 = false;
  let running = false;
  let error = "";
  let hops: any[] = [];
  let result: any = null;

  function ms(ns: number): string {
    if (!ns || ns <= 0) return "—";
    return (ns / 1e6).toFixed(1);
  }
  function stats(rtts: number[]): { min: string; avg: string; max: string } {
    const v = (rtts ?? []).filter((x) => x > 0);
    if (!v.length) return { min: "*", avg: "*", max: "*" };
    const sum = v.reduce((a, b) => a + b, 0);
    return { min: ms(Math.min(...v)), avg: ms(sum / v.length), max: ms(Math.max(...v)) };
  }
  function spark(rtts: number[]): string {
    const v = (rtts ?? []).filter((x) => x > 0);
    if (v.length < 2) return "";
    const w = 100, h = 20, mx = Math.max(...v), mn = Math.min(...v), r = mx - mn || 1, step = w / (v.length - 1);
    const d = v.map((x, i) => `${(i * step).toFixed(1)},${(h - 3 - ((x - mn) / r) * (h - 6)).toFixed(1)}`).join(" ");
    return `<svg width="${w}" height="${h}" viewBox="0 0 ${w} ${h}" preserveAspectRatio="none"><polyline points="${d}" fill="none" stroke="var(--accent)" stroke-width="1.5"/></svg>`;
  }

  // Distinct ASNs in path order, for the AS-path strip.
  $: asPath = (() => {
    const seen = new Set<string>();
    const out: any[] = [];
    for (const h of hops) {
      if (!h.asn || seen.has(h.asn)) continue;
      seen.add(h.asn);
      out.push(h);
    }
    return out;
  })();

  function boundary(i: number): boolean {
    if (!hops[i]?.asn) return false;
    for (let j = i - 1; j >= 0; j--) {
      if (hops[j].asn) return hops[j].asn !== hops[i].asn;
    }
    return true;
  }

  async function start() {
    if (running) return;
    hops = [];
    result = null;
    error = "";
    running = true;
    const handler = (h: any) => (hops = [...hops, h]);
    EventsOn("trace:hop", handler);
    try {
      result = await Traceroute(target.trim(), useV6);
      if (result?.hops) hops = result.hops; // authoritative, fully annotated
    } catch (e) {
      error = String(e);
    } finally {
      EventsOff("trace:hop");
      running = false;
    }
  }

  function stop() {
    StopTrace();
  }

  // If the tool is closed mid-trace, stop probing and drop the listener so the
  // backend doesn't keep emitting events to a gone component.
  onDestroy(() => {
    EventsOff("trace:hop");
    if (running) StopTrace();
  });
</script>

<div class="tool">
  <h2>Traceroute (AS-aware)</h2>
  <p class="sub">Each hop labelled with its network (AS) and owner. No admin rights needed — uses ICMP echo.</p>

  <form on:submit|preventDefault={start} class="row">
    <input class="field mono" bind:value={target} placeholder="cloudflare.com  or  104.16.123.45" autocomplete="off" spellcheck="false" aria-label="Target host" />
    <select class="sel" bind:value={useV6} aria-label="IP version">
      <option value={false}>IPv4</option>
      <option value={true}>IPv6</option>
    </select>
    {#if running}
      <button type="button" class="go stop" on:click={stop}>Stop</button>
    {:else}
      <button class="go">Trace</button>
    {/if}
  </form>

  {#if error}<div class="hint">{error}</div>{/if}
  {#if result?.verdict}<div class="summary">{result.verdict}</div>{/if}

  {#if asPath.length}
    <div class="aspath">
      <span class="aslabel">AS path</span>
      {#each asPath as a, i}
        {#if i > 0}<span class="arrow">→</span>{/if}
        <span class="asnode"><span class="n mono">AS{a.asn}</span><span class="o">{a.asOwner || ""}</span></span>
      {/each}
    </div>
  {/if}

  {#if hops.length}
    <table class="tbl">
      <thead><tr><th>#</th><th>Host</th><th>AS · owner</th><th>RTT (min / avg / max)</th><th style="width:110px">trend</th></tr></thead>
      <tbody>
        {#each hops as h, i}
          {@const s = stats(h.rtts)}
          <tr>
            <td class="mono">{h.ttl}</td>
            <td class="host mono" class:boundary={boundary(i)} class:star={h.timeout}>
              {#if h.timeout}*{:else if h.host}{h.host} <span class="dim">({h.addr})</span>{:else}{h.addr}{/if}
            </td>
            <td>
              {#if h.asn}<span class="mono asn">AS{h.asn}</span> <span class="dim">{h.asOwner || ""}</span>{:else}<span class="dim">{h.timeout ? "" : "private / unknown"}</span>{/if}
            </td>
            <td class="mono" class:star={h.timeout}>{s.min} / {s.avg} / {s.max}</td>
            <td>{@html spark(h.rtts) || '<span class="star">—</span>'}</td>
          </tr>
        {/each}
      </tbody>
    </table>
    {#if running}<p class="muted">Tracing… (<button class="linkbtn" on:click={stop}>stop</button>)</p>{/if}
  {/if}

  <p class="foot">BGP AS-path comparison and MTR live view — <span class="soon">soon</span>.</p>
</div>

<style>
  .tool { padding: 16px 20px 40px; max-width: 1000px; }
  h2 { margin: 0 0 2px; font-size: 1.15rem; font-weight: 600; color: var(--text-strong); }
  .sub { margin: 0 0 15px; color: var(--text-dim); font-size: 0.86rem; }
  .row { display: flex; gap: 8px; margin-bottom: 14px; flex-wrap: wrap; }
  .field { flex: 1; min-width: 200px; background: var(--well); border: 1px solid var(--field-border); border-radius: 7px; padding: 9px 12px; color: var(--text); font-size: 0.9rem; outline: none; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .sel { background: var(--well); border: 1px solid var(--field-border); border-radius: 7px; padding: 9px 10px; color: var(--text); font: inherit; font-size: 0.86rem; cursor: pointer; }
  .go { background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px; padding: 0 16px; font: inherit; font-size: 0.9rem; cursor: pointer; }
  .go.stop { background: var(--warn-bg); border-color: var(--warn-border); color: var(--warn-text); }
  .hint { background: var(--warn-bg); border: 1px solid var(--warn-border); color: var(--warn-text); border-radius: 7px; padding: 9px 12px; font-size: 0.85rem; margin-bottom: 12px; }
  .summary { background: var(--card); border: 1px solid var(--accent); border-left-width: 3px; box-shadow: 0 0 0 1px var(--accent-ring); border-radius: 9px; padding: 11px 14px; margin-bottom: 14px; font-size: 0.9rem; color: var(--text-strong); }
  .aspath { display: flex; align-items: center; gap: 7px; flex-wrap: wrap; background: var(--card); border: 1px solid var(--border-soft); border-radius: 9px; padding: 11px 13px; margin-bottom: 14px; }
  .aslabel { font-size: 0.6rem; text-transform: uppercase; letter-spacing: .06em; color: var(--text-dim); margin-right: 4px; }
  .asnode { display: flex; flex-direction: column; gap: 1px; padding: 3px 10px; border: 1px solid var(--border); border-radius: 7px; }
  .asnode .n { font-size: 0.78rem; color: var(--accent); }
  .asnode .o { font-size: 0.64rem; color: var(--text-dim); }
  .arrow { color: var(--text-faint); }
  table.tbl { width: 100%; border-collapse: collapse; font-size: 0.82rem; }
  .tbl th { text-align: left; color: var(--text-dim); font-weight: 500; font-size: 0.68rem; text-transform: uppercase; letter-spacing: .04em; padding: 7px 10px; border-bottom: 1px solid var(--border); }
  .tbl td { padding: 7px 10px; border-bottom: 1px solid var(--border-soft); vertical-align: middle; }
  .tbl td.boundary { box-shadow: inset 3px 0 0 var(--accent); }
  .asn { color: var(--accent); }
  .dim { color: var(--text-dim); } .star { color: var(--text-faint); }
  .muted { color: var(--text-dim); font-size: 0.8rem; margin: 10px 0 0; }
  .linkbtn { background: none; border: none; color: var(--accent); cursor: pointer; font: inherit; padding: 0; text-decoration: underline; }
  .foot { color: var(--text-faint); font-size: 0.74rem; margin: 16px 0 0; }
  .soon { font-size: 0.6rem; color: var(--text-faint); border: 1px solid var(--pill-border); border-radius: 8px; padding: 0 6px; }
  .mono { font-family: var(--font-mono); }
</style>
