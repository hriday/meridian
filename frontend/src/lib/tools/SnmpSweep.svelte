<script lang="ts">
  import { onDestroy } from "svelte";
  import { SnmpSweep, ScanStop } from "../../../wailsjs/go/app/App.js";
  import { EventsOn, EventsOff } from "../../../wailsjs/runtime/runtime.js";

  let cidr = "10.0.0.0/24";
  let community = "";
  let scanning = false;
  let scanned = 0;
  let responded = 0;
  let total = 0;
  let rows: any[] = [];
  let err = "";
  let startedAt = 0;
  let elapsed = 0;

  EventsOn("snmp:host", (h: any) => {
    scanned += 1;
    if (h.responded) responded += 1;
    elapsed = (Date.now() - startedAt) / 1000;
  });
  onDestroy(() => { EventsOff("snmp:host"); ScanStop(); });

  // host count of a CIDR for the progress bar (IPv4; skip net+broadcast for < /31)
  function hostCount(c: string): number {
    const m = c.trim().match(/^(\d{1,3}(?:\.\d{1,3}){3})\/(\d{1,2})$/);
    if (!m) return 0;
    const bits = parseInt(m[2], 10);
    if (bits < 0 || bits > 32) return 0;
    const n = Math.pow(2, 32 - bits);
    return bits < 31 ? Math.max(0, n - 2) : n;
  }

  async function run() {
    const c = cidr.trim();
    if (!c || scanning) return;
    err = ""; rows = []; scanned = 0; responded = 0;
    total = hostCount(c);
    startedAt = Date.now(); elapsed = 0;
    scanning = true;
    try {
      const res = await SnmpSweep(c, community); // authoritative sorted responders
      rows = res || [];
    } catch (e) {
      err = String(e);
    } finally {
      scanning = false;
      elapsed = (Date.now() - startedAt) / 1000;
    }
  }

  function stop() { ScanStop(); }

  // SNMP TimeTicks are centiseconds (1/100 s)
  function uptime(ticks: number): string {
    const s = Math.floor((ticks || 0) / 100);
    const d = Math.floor(s / 86400), h = Math.floor((s % 86400) / 3600), m = Math.floor((s % 3600) / 60);
    if (d > 0) return `${d}d ${h}h`;
    if (h > 0) return `${h}h ${m}m`;
    return `${m}m`;
  }

  $: progress = total > 0 ? Math.min(100, (scanned / total) * 100) : scanning ? 0 : 100;
</script>

<div class="tool">
  <p class="lede">
    SNMP sweep — walks a subnet and finds everything that answers <strong>SNMPv2c</strong>, so you can
    see what's actually out there: the IP, its name, what it is, and how long it's been up. Point it at a
    CIDR with a read community; responders fill in as they're found. Read-only; the community is never
    shown back. The UI never blocks on the scan.
  </p>

  <form class="row" on:submit|preventDefault={run}>
    <input class="field cidr mono" bind:value={cidr} placeholder="10.0.0.0/24  ·  subnet (CIDR)" aria-label="Subnet CIDR" spellcheck="false" autocomplete="off" />
    <input class="field comm mono" type="password" bind:value={community} placeholder="community" aria-label="SNMP community" autocomplete="off" />
    {#if scanning}
      <button type="button" class="go" on:click={stop}>Stop</button>
    {:else}
      <button type="submit" class="go" disabled={!cidr.trim()}>Scan</button>
    {/if}
  </form>

  <p class="help">
    Enter a subnet and your SNMP read community, then <strong>Scan</strong>. Each row is a device that
    answered SNMP — its <strong>address</strong>, the <strong>name</strong> it reports (sysName),
    <strong>what it is</strong> (model/OS, from sysDescr), and its <strong>uptime</strong>. A device
    that doesn't appear simply didn't answer: SNMP off, a different community, or v3-only.
  </p>

  {#if err}
    <div class="banner bad"><span class="vlabel">Couldn't sweep</span><span class="vnote">{err}</span></div>
  {/if}

  {#if scanning || scanned > 0}
    <div class="statusbar">
      <span>scanned <span class="count">{scanned}</span>{total ? ` / ${total}` : ""} · <span class="count">{responded}</span> responded</span>
      <div class="prog"><div class="progfill" style="width:{progress}%"></div></div>
      <span class="mono">{scanning ? "scanning…" : `done · ${elapsed.toFixed(1)}s`}</span>
    </div>
  {/if}

  {#if rows.length}
    <div class="grid">
      <div class="head"><span>Address</span><span>Name</span><span>Description</span><span class="r">Uptime</span></div>
      {#each rows as h}
        <div class="drow">
          <div class="ip mono">{h.ip}</div>
          <div class="nm mono">{h.name || "—"}</div>
          <div class="desc mono" title={h.descr}>{h.descr || "—"}</div>
          <div class="up mono r">{uptime(h.uptimeTicks)}</div>
        </div>
      {/each}
    </div>
    <p class="caption">{rows.length} SNMP responder{rows.length === 1 ? "" : "s"}{total ? ` in ${cidr.trim()}` : ""} · sorted by address · v2c · 1s timeout</p>
  {:else if !scanning && scanned > 0 && !err}
    <p class="caption">No hosts answered SNMP in {cidr.trim()} — check the community and that SNMP is enabled.</p>
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 20px; }
  .lede { max-width: 880px; margin: 0 0 14px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .mono { font-family: var(--mono, ui-monospace, monospace); }
  .row { display: flex; gap: 8px; max-width: 880px; align-items: center; }
  .field { background: var(--well); border: 1px solid var(--field-border); border-radius: 7px; padding: 9px 12px; color: var(--text); font-size: 0.9rem; outline: none; }
  .field.cidr { flex: 1; }
  .field.comm { flex: 0 0 150px; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .go { background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px; padding: 0 16px; height: 38px; font: inherit; font-size: 0.9rem; cursor: pointer; }
  .go:disabled { opacity: .6; cursor: default; }

  .help { max-width: 880px; margin: 12px 0 0; padding: 9px 13px; border-left: 2px solid var(--accent); background: var(--well); border-radius: 0 7px 7px 0; font-size: 0.78rem; color: var(--text-dim); line-height: 1.5; }
  .help strong { color: var(--text); font-weight: 600; }

  .banner { margin: 14px 0; padding: 11px 14px; border-radius: 9px; border: 1px solid var(--bad, #ff6b6b); background: color-mix(in srgb, var(--bad,#ff6b6b) 10%, transparent); max-width: 880px; display: flex; flex-direction: column; gap: 3px; }
  .vlabel { font-weight: 600; color: var(--bad, #ff6b6b); }
  .vnote { font-size: 0.8rem; color: var(--text-dim); }

  .statusbar { display: flex; align-items: center; gap: 12px; margin: 16px 0 8px; max-width: 880px; font-size: 0.8rem; color: var(--text-dim); }
  .count { color: var(--text-strong); font-weight: 600; }
  .prog { flex: 1; height: 5px; border-radius: 3px; background: var(--track, #2a3142); overflow: hidden; }
  .progfill { height: 100%; background: var(--accent); border-radius: 3px; transition: width 0.2s ease; }

  .grid { max-width: 880px; margin-top: 6px; border: 1px solid var(--border-soft); border-radius: 11px; overflow: hidden; background: var(--card); }
  .head, .drow { display: grid; grid-template-columns: 1.1fr 1.3fr 3fr 1fr; align-items: center; gap: 14px; padding: 9px 16px; }
  .head { font-size: 0.6rem; text-transform: uppercase; letter-spacing: .08em; color: var(--text-faint); border-bottom: 1px solid var(--border-soft); background: var(--panel); }
  .drow { border-top: 1px solid var(--border-soft); font-size: 0.82rem; }
  .drow:first-of-type { border-top: none; }
  .ip { color: var(--text-strong); font-weight: 600; }
  .desc { color: var(--text-dim); font-size: 0.76rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .up { color: var(--text-dim); }
  .r { text-align: right; }

  .caption { max-width: 880px; margin-top: 12px; font-size: 0.72rem; color: var(--text-faint); }
</style>
