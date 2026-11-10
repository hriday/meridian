<script lang="ts">
  import { onDestroy } from "svelte";
  import { PingSweep, ScanStop } from "../../../wailsjs/go/app/App.js";
  import { EventsOn, EventsOff } from "../../../wailsjs/runtime/runtime.js";

  let cidr = "192.168.1.0/24";
  let portSpec = "80,443,22";

  let scanning = false;
  let probed = 0;
  let hosts: any[] = [];
  let err = "";

  function parsePorts(spec: string): number[] {
    const out: number[] = [];
    for (const part of spec.split(",")) {
      const p = part.trim();
      if (!p) continue;
      if (p.includes("-")) {
        const [a, b] = p.split("-").map((x) => parseInt(x, 10));
        for (let i = a; i <= b; i++) out.push(i);
      } else out.push(parseInt(p, 10));
    }
    return out.filter((n) => n >= 1 && n <= 65535);
  }

  EventsOn("scan:host", (h: any) => {
    probed += 1;
    if (h.alive) hosts = [...hosts, h];
  });
  onDestroy(() => { EventsOff("scan:host"); ScanStop(); });

  async function run() {
    err = ""; hosts = []; probed = 0; scanning = true;
    try {
      const all = await PingSweep(cidr, parsePorts(portSpec));
      // final authoritative list (events already streamed live hosts)
      hosts = all.filter((h: any) => h.alive);
    } catch (e) { err = String(e); }
    finally { scanning = false; }
  }

  function stop() { ScanStop(); }
  function rttMs(ns: number) { return ns > 0 ? (ns / 1e6).toFixed(1) + " ms" : ""; }
</script>

<div class="tool">
  <div class="note">🔎 Sweep only networks you're authorized to scan.</div>

  <div class="card">
    <div class="line">
      <input class="field mono" bind:value={cidr} placeholder="192.168.1.0/24" aria-label="CIDR" />
      <input class="field mono" bind:value={portSpec} placeholder="fallback ports: 80,443,22" aria-label="fallback ports" />
    </div>
    <div class="line">
      {#if !scanning}<button class="go" on:click={run}>Sweep</button>
      {:else}<button class="go ghost" on:click={stop}>Stop</button>{/if}
      {#if scanning || probed > 0}<span class="dim small">{probed} probed · {hosts.length} up</span>{/if}
    </div>
    <div class="cap">ICMP echo first, then a TCP connect to the fallback ports for hosts that filter ICMP.</div>
    {#if err}<div class="hint">{err}</div>{/if}
  </div>

  {#if hosts.length}
    <div class="card span">
      <div class="k">{hosts.length} host{hosts.length === 1 ? "" : "s"} up</div>
      <table class="results">
        <tbody>
          {#each hosts as h}
            <tr>
              <td class="mono">{h.ip}</td>
              <td><span class="pill {h.method === 'icmp' ? 'good' : 'soon'}">{h.method}</span></td>
              <td class="dim mono">{rttMs(h.rtt)}</td>
              <td class="dim">{h.name}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 24px; }
  .note { background: var(--warn-bg); border: 1px solid var(--warn-border); color: var(--warn-text); border-radius: 9px; padding: 9px 13px; font-size: 0.8rem; margin-bottom: 13px; max-width: 980px; }
  .card { background: var(--card); border: 1px solid var(--border-soft); border-radius: 10px; padding: 13px 15px; margin-bottom: 13px; }
  .card.span { grid-column: 1 / -1; }
  .k { font-size: 0.72rem; text-transform: uppercase; letter-spacing: .04em; color: var(--text-dim); }
  .line { display: flex; align-items: center; gap: 8px; margin: 6px 0; flex-wrap: wrap; }
  .field { background: var(--well); border: 1px solid var(--field-border); border-radius: 7px; padding: 8px 11px; color: var(--text); font-size: 0.86rem; outline: none; min-width: 180px; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .mono { font-family: var(--font-mono); }
  .go { background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px; padding: 7px 14px; font: inherit; font-size: 0.85rem; cursor: pointer; }
  .go.ghost { background: transparent; color: var(--accent); }
  .cap { font-size: 0.73rem; color: var(--text-dim); margin-top: 7px; }
  .dim { color: var(--text-faint); }
  .small { font-size: 0.72rem; }
  .hint { background: var(--warn-bg); border: 1px solid var(--warn-border); color: var(--warn-text); border-radius: 7px; padding: 8px 11px; font-size: 0.82rem; margin-top: 8px; }
  .results { width: 100%; border-collapse: collapse; margin-top: 6px; }
  .results td { padding: 5px 8px; border-bottom: 1px solid var(--border-soft); font-size: 0.83rem; }
  .pill { font-size: 0.66rem; padding: 1px 8px; border-radius: 20px; }
  .pill.good { color: var(--good, #5ad17a); background: var(--good-bg, #102516); }
  .pill.soon { color: var(--text-faint); border: 1px solid var(--pill-border); }
</style>
