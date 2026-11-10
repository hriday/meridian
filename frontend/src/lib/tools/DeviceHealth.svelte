<script lang="ts">
  import { onDestroy } from "svelte";
  import BarMeter from "../metrics/BarMeter.svelte";
  import Sparkline from "../metrics/Sparkline.svelte";
  import { subscribe, newStreamID } from "../metrics/metricStream";
  import { StartDeviceMonitor, StopDeviceMonitor } from "../../../wailsjs/go/app/App.js";

  type Entry = {
    id: number;
    addr: string;
    community: string; // held only here, never persisted
    streamID: string;
    unsub: (() => void) | null;
    snap: any | null;
    hist: number[];
  };

  let addr = "";
  let community = "";
  let worstFirst = false;
  let devices: Entry[] = [];
  let seq = 0;
  let addrEl: HTMLInputElement;
  const HIST = 12; // ~60 s at the 5 s poll

  async function add() {
    const a = addr.trim();
    if (!a) return;
    const streamID = newStreamID();
    const e: Entry = { id: ++seq, addr: a, community, streamID, unsub: null, snap: null, hist: [] };
    e.unsub = subscribe(streamID, (s) => onSnap(e, s));
    devices = [...devices, e];
    addr = "";
    addrEl?.focus();
    try {
      await StartDeviceMonitor(streamID, a, community);
    } catch (err) {
      e.snap = { state: "error", note: String(err) };
      devices = [...devices];
    }
  }

  function onSnap(e: Entry, s: any) {
    e.snap = s;
    if (s.state === "ok") {
      const busier = metricVal(s);
      if (busier != null) e.hist = [...e.hist, busier].slice(-HIST);
    }
    devices = [...devices]; // nudge Svelte (mutated an entry in place)
  }

  function remove(e: Entry) {
    if (e.unsub) e.unsub();
    StopDeviceMonitor(e.streamID).catch(() => {});
    devices = devices.filter((d) => d.id !== e.id);
  }

  onDestroy(() => {
    for (const e of devices) {
      if (e.unsub) e.unsub();
      StopDeviceMonitor(e.streamID).catch(() => {});
    }
  });

  // ---- thresholds (fixed): CPU 75/90, MEM 80/90 ----
  const good = "var(--good)", warn = "var(--warn, #d9a441)", bad = "var(--bad, #ff6b6b)", idle = "var(--text-faint)";
  const cpuSev = (v: number) => (v >= 90 ? 3 : v >= 75 ? 2 : 1);
  const memSev = (v: number) => (v >= 90 ? 3 : v >= 80 ? 2 : 1);
  const sevCol = (sev: number) => (sev >= 3 ? bad : sev >= 2 ? warn : good);

  // the busier metric's value (for the sparkline), preferring whichever is present
  function metricVal(s: any): number | null {
    const c = s.hasCpu ? s.cpuPct : null;
    const m = s.hasMem ? s.memPct : null;
    if (c == null && m == null) return null;
    return Math.max(c ?? 0, m ?? 0);
  }

  // per-row verdict: colour, state word, row class
  function verdict(s: any): { col: string; word: string; cls: string } {
    if (!s) return { col: idle, word: "…", cls: "" };
    if (s.state === "error") return { col: bad, word: "ERROR", cls: "down" };
    if (s.state === "unreachable") return { col: warn, word: "UNREACHABLE", cls: "down" };
    if (!s.hasCpu && !s.hasMem) return { col: idle, word: "NO HOST-RES", cls: "" };
    const cSev = s.hasCpu ? cpuSev(s.cpuPct) : 0;
    const mSev = s.hasMem ? memSev(s.memPct) : 0;
    const sev = Math.max(cSev, mSev);
    if (sev >= 3) return { col: bad, word: cSev >= mSev ? "CPU HIGH" : "MEM HIGH", cls: "alert" };
    if (sev >= 2) return { col: warn, word: cSev >= mSev ? "CPU WARN" : "MEM WARN", cls: "warnrow" };
    return { col: good, word: "STEADY", cls: "" };
  }

  // severity rank for worst-first sorting
  function rank(e: Entry): number {
    const s = e.snap;
    if (!s) return 0;
    if (s.state === "error" || s.state === "unreachable") return 100;
    const v = metricVal(s);
    return v ?? -1;
  }

  const pct = (v: number) => v.toFixed(0);
  $: shown = worstFirst ? [...devices].sort((a, b) => rank(b) - rank(a)) : devices;
  $: attention = devices.filter((e) => ["alert", "warnrow", "down"].includes(verdict(e.snap).cls)).length;
</script>

<div class="tool">
  <p class="lede">
    Device health — watches a set of switches, routers and firewalls over
    <strong>SNMPv2c</strong> at once and keeps each one's CPU and memory in front of you, so the box
    that spikes stands out. Add the devices you care about; the grid colours anything past its
    threshold and flags anything that stops answering. The community string is read-only and never
    shown back. The UI never blocks on polling.
  </p>

  <form class="row" on:submit|preventDefault={add}>
    <input class="field addr mono" bind:this={addrEl} bind:value={addr} placeholder="10.0.0.1  ·  device address" aria-label="Device address" spellcheck="false" autocomplete="off" />
    <input class="field comm mono" type="password" bind:value={community} placeholder="community" aria-label="SNMP community" autocomplete="off" />
    <button type="submit" class="go" disabled={!addr.trim()}>Add device</button>
    {#if devices.length > 1}
      <button type="button" class="ghost" class:on={worstFirst} on:click={() => (worstFirst = !worstFirst)}>Sort: worst first</button>
    {/if}
  </form>

  {#if devices.length}
    <div class="grid">
      <div class="head">
        <span>Device</span><span>CPU</span><span>Memory</span><span>60s</span><span>State</span><span></span>
      </div>
      {#each shown as e (e.id)}
        {@const v = verdict(e.snap)}
        <div class="drow {v.cls}">
          <div class="dev">
            <div class="nm mono">{e.snap?.name || e.addr}</div>
            <div class="ad mono">{e.addr}</div>
          </div>

          {#if e.snap && e.snap.state !== "ok"}
            <div class="meter"><span class="na">—</span></div>
            <div class="meter"><span class="na">—</span></div>
          {:else}
            <div class="meter">
              {#if e.snap?.hasCpu}
                <BarMeter pct={e.snap.cpuPct} color={sevCol(cpuSev(e.snap.cpuPct))} /><span class="pct mono">{pct(e.snap.cpuPct)}%</span>
              {:else}<span class="na">n/a</span>{/if}
            </div>
            <div class="meter">
              {#if e.snap?.hasMem}
                <BarMeter pct={e.snap.memPct} color={sevCol(memSev(e.snap.memPct))} /><span class="pct mono">{pct(e.snap.memPct)}%</span>
              {:else}<span class="na">n/a</span>{/if}
            </div>
          {/if}

          <div>
            {#if e.snap && e.snap.state === "ok"}<Sparkline values={e.hist} color={v.col} />{:else}<span class="na">—</span>{/if}
          </div>

          <div class="status"><span class="dot" style="background:{v.col}"></span><span class="sw" style="color:{v.col}">{v.word}</span></div>

          <button class="rm" title="Remove {e.addr}" aria-label="Remove {e.addr}" on:click={() => remove(e)}>×</button>
        </div>
      {/each}
    </div>
    <p class="caption">{devices.length} device{devices.length === 1 ? "" : "s"}{attention ? ` · ${attention} need attention` : ""} · polling every 5s · thresholds CPU 75/90 · MEM 80/90</p>
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 20px; }
  .lede { max-width: 880px; margin: 0 0 14px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .mono { font-family: var(--mono, ui-monospace, monospace); }
  .row { display: flex; gap: 8px; max-width: 880px; align-items: center; }
  .field { background: var(--well); border: 1px solid var(--field-border); border-radius: 7px; padding: 9px 12px; color: var(--text); font-size: 0.9rem; outline: none; }
  .field.addr { flex: 1; }
  .field.comm { flex: 0 0 150px; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .go { background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px; padding: 0 16px; height: 38px; font: inherit; font-size: 0.9rem; cursor: pointer; }
  .go:disabled { opacity: .6; cursor: default; }
  .ghost { background: transparent; border: 1px solid var(--border-soft); color: var(--text-dim); border-radius: 7px; padding: 8px 13px; font: inherit; font-size: 0.78rem; cursor: pointer; }
  .ghost:hover { color: var(--text); border-color: var(--field-border); }
  .ghost.on { color: var(--text-strong); border-color: var(--accent); }

  .grid { max-width: 880px; margin-top: 14px; border: 1px solid var(--border-soft); border-radius: 11px; overflow: hidden; background: var(--card); }
  .head, .drow { display: grid; grid-template-columns: 1.6fr 2fr 2fr 1.2fr 1fr 28px; align-items: center; gap: 14px; padding: 9px 16px; }
  .head { font-size: 0.6rem; text-transform: uppercase; letter-spacing: .08em; color: var(--text-faint); border-bottom: 1px solid var(--border-soft); background: var(--panel); }
  .drow { border-top: 1px solid var(--border-soft); font-size: 0.82rem; }
  .drow:first-of-type { border-top: none; }
  .drow.alert { background: color-mix(in srgb, var(--bad, #ff6b6b) 8%, transparent); }
  .drow.warnrow { background: color-mix(in srgb, var(--warn, #d9a441) 7%, transparent); }
  .drow.down { opacity: .62; }

  .dev .nm { color: var(--text-strong); font-weight: 600; }
  .dev .ad { color: var(--text-faint); font-size: 0.72rem; }

  .meter { display: flex; align-items: center; gap: 9px; }
  .pct { width: 40px; text-align: right; font-size: 0.8rem; }
  .na { color: var(--text-faint); font-size: 0.76rem; }

  .status { display: flex; align-items: center; gap: 7px; }
  .dot { width: 8px; height: 8px; border-radius: 50%; flex: none; }
  .sw { font-size: 0.64rem; letter-spacing: .02em; font-weight: 650; }

  .rm { background: none; border: none; color: var(--text-faint); font-size: 1.05rem; line-height: 1; cursor: pointer; padding: 0; }
  .rm:hover { color: var(--bad, #ff6b6b); }

  .caption { max-width: 880px; margin-top: 12px; font-size: 0.72rem; color: var(--text-faint); }
</style>
