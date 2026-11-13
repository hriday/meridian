<script lang="ts">
  import { PTRSweep } from "../../../wailsjs/go/app/App.js";

  let cidr = "";
  let entries: any[] = [];
  let loading = false;
  let error = "";
  let ran = false;

  async function run() {
    const c = cidr.trim();
    if (!c) return;
    loading = true;
    error = "";
    try {
      entries = (await PTRSweep(c)) || [];
      ran = true;
    } catch (e: any) {
      error = String(e?.message ?? e);
      entries = [];
      ran = true;
    } finally {
      loading = false;
    }
  }
</script>

<div class="tool">
  <p class="lede">
    Reverse-DNS (<strong>PTR</strong>) sweep across a subnet — maps which addresses in a range resolve to a hostname,
    a quick read of what's <em>named</em> in a network. DNS only; no packets are sent to the hosts. Pairs with
    <strong>Ping sweep</strong> / <strong>Port scanner</strong>, which probe reachability.
  </p>

  <div class="row">
    <input class="field mono" bind:value={cidr} placeholder="10.0.0.0/24" spellcheck="false" autocomplete="off"
      on:keydown={(e) => e.key === "Enter" && run()} />
    <button class="btn" on:click={run} disabled={loading || !cidr.trim()}>{loading ? "Sweeping…" : "Sweep"}</button>
  </div>

  {#if error}<p class="err">{error}</p>{/if}

  {#if ran && !error}
    {#if entries.length}
      <div class="card">
        <table>
          <thead><tr><th>Address</th><th>Hostname(s)</th></tr></thead>
          <tbody>
            {#each entries as e}
              <tr>
                <td class="mono ip">{e.ip}</td>
                <td class="names">{e.names.join(", ")}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
      <p class="foot">{entries.length} address{entries.length === 1 ? "" : "es"} with a PTR record. Unnamed addresses aren't shown — no PTR doesn't mean no host.</p>
    {:else}
      <p class="foot">No PTR records in that range. Many networks don't publish reverse DNS for internal hosts.</p>
    {/if}
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 30px; }
  .lede { max-width: 820px; margin: 0 0 14px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .mono { font-family: var(--font-mono, ui-monospace, monospace); }
  .row { display: flex; gap: 10px; max-width: 520px; }
  .field { background: var(--well); border: 1px solid var(--field-border, var(--border)); border-radius: 8px; padding: 9px 12px; color: var(--text); font-size: 0.86rem; outline: none; flex: 1 1 auto; min-width: 0; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .btn { background: var(--accent); color: var(--on-accent, #052e2b); border: 1px solid transparent; border-radius: 8px; padding: 9px 16px; font: inherit; font-size: 0.86rem; font-weight: 550; cursor: pointer; white-space: nowrap; }
  .btn:disabled { opacity: .5; cursor: default; }
  .err { color: var(--danger, #e5484d); font-size: 0.82rem; margin: 12px 0 0; }
  .card { background: var(--card); border: 1px solid var(--border); border-radius: 11px; max-width: 760px; overflow: hidden; margin-top: 16px; }
  table { width: 100%; border-collapse: collapse; font-size: 0.83rem; }
  th { text-align: left; font-weight: 600; color: var(--text-dim); padding: 9px 12px; border-bottom: 1px solid var(--border); background: var(--well); font-size: 0.74rem; text-transform: uppercase; letter-spacing: .03em; }
  td { padding: 8px 12px; border-bottom: 1px solid var(--border-soft, var(--border)); }
  tr:last-child td { border-bottom: 0; }
  .ip { font-weight: 600; color: var(--text); width: 180px; }
  .names { color: var(--text); }
  .foot { max-width: 760px; font-size: 0.76rem; color: var(--text-faint, var(--text-dim)); margin: 12px 0 0; line-height: 1.6; }
</style>
