<script lang="ts">
  import { onDestroy } from "svelte";
  import { PortScan, ScanCommonPorts, ScanStop } from "../../../wailsjs/go/app/App.js";
  import { EventsOn, EventsOff } from "../../../wailsjs/runtime/runtime.js";

  let host = "127.0.0.1";
  let portSpec = "22,80,443,3389,8080";
  let common = false;
  let showAll = false;

  let scanning = false;
  let scanned = 0;
  let total = 0;
  let results: any[] = [];
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

  EventsOn("scan:port", () => { scanned += 1; });
  onDestroy(() => { EventsOff("scan:port"); ScanStop(); });

  async function run() {
    err = ""; results = []; scanned = 0;
    let ports: number[];
    try {
      ports = common ? await ScanCommonPorts() : parsePorts(portSpec);
    } catch (e) { err = String(e); return; }
    if (!ports.length) { err = "No valid ports."; return; }
    total = ports.length;
    scanning = true;
    try {
      results = await PortScan(host, ports);
    } catch (e) { err = String(e); }
    finally { scanning = false; }
  }

  function stop() { ScanStop(); }

  function stateClass(s: string) { return s === "open" ? "good" : s === "closed" ? "dim" : "soon"; }
  $: shown = results.filter((r) => showAll || r.state === "open");
  $: openCount = results.filter((r) => r.state === "open").length;
</script>

<div class="tool">
  <div class="note">🔎 Scan only hosts and networks you're authorized to test.</div>

  <div class="card">
    <div class="line">
      <input class="field mono" bind:value={host} placeholder="host or IP" aria-label="host" />
      {#if !common}
        <input class="field mono grow" bind:value={portSpec} placeholder="22,80,443,8000-8100" aria-label="ports" />
      {/if}
      <label class="chk"><input type="checkbox" bind:checked={common} /> common ports</label>
    </div>
    <div class="line">
      {#if !scanning}<button class="go" on:click={run}>Scan</button>
      {:else}<button class="go ghost" on:click={stop}>Stop</button>{/if}
      {#if scanning || total > 0}
        <span class="dim small">{scanned}/{total} scanned · {openCount} open</span>
      {/if}
    </div>
    {#if err}<div class="hint">{err}</div>{/if}
  </div>

  {#if results.length}
    <div class="card span">
      <div class="line between">
        <div class="k">{openCount} open port{openCount === 1 ? "" : "s"} on {host}</div>
        <label class="chk"><input type="checkbox" bind:checked={showAll} /> show closed/filtered</label>
      </div>
      <table class="results">
        <tbody>
          {#each shown as r}
            <tr>
              <td class="mono">{r.port}/tcp</td>
              <td><span class="pill {stateClass(r.state)}">{r.state}</span></td>
              <td class="dim">{r.service}</td>
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
  .line.between { justify-content: space-between; }
  .field { background: var(--well); border: 1px solid var(--field-border); border-radius: 7px; padding: 8px 11px; color: var(--text); font-size: 0.86rem; outline: none; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .grow { flex: 1; min-width: 200px; }
  .mono { font-family: var(--font-mono); }
  .go { background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px; padding: 7px 14px; font: inherit; font-size: 0.85rem; cursor: pointer; }
  .go.ghost { background: transparent; color: var(--accent); }
  .chk { font-size: 0.8rem; color: var(--text-dim); display: flex; align-items: center; gap: 4px; }
  .dim { color: var(--text-faint); }
  .small { font-size: 0.72rem; }
  .hint { background: var(--warn-bg); border: 1px solid var(--warn-border); color: var(--warn-text); border-radius: 7px; padding: 8px 11px; font-size: 0.82rem; margin-top: 8px; }
  .results { width: 100%; border-collapse: collapse; margin-top: 6px; }
  .results td { padding: 5px 8px; border-bottom: 1px solid var(--border-soft); font-size: 0.83rem; }
  .pill { font-size: 0.66rem; padding: 1px 8px; border-radius: 20px; }
  .pill.good { color: var(--good, #5ad17a); background: var(--good-bg, #102516); }
  .pill.soon { color: var(--text-faint); border: 1px solid var(--pill-border); }
  .pill.dim { color: var(--text-faint); }
</style>
