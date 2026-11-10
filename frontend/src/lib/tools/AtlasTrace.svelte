<script lang="ts">
  import { onMount } from "svelte";
  import { AtlasKeyStatus, AtlasTrace } from "../../../wailsjs/go/app/App.js";

  let keySet = false;
  let keyErr = "";

  let target = "";
  let probes = 5;
  let busy = false;
  let error = "";
  let r: any = null;

  onMount(async () => {
    try {
      const s = await AtlasKeyStatus();
      keySet = s.set;
    } catch (e) {
      keyErr = String(e);
    }
  });

  async function run() {
    const v = target.trim();
    if (!v) {
      r = null;
      return;
    }
    error = "";
    r = null;
    busy = true;
    try {
      r = await AtlasTrace(v, probes);
    } catch (e) {
      error = String(e);
    } finally {
      busy = false;
    }
  }

  // Atlas reports no rtt for a non-responding hop; show "–".
  const fmt = (v: number) => (v == null || v < 0 ? "–" : v.toFixed(1));
</script>

<div class="tool">
  <p class="lede">
    Traceroute a target from vantage points around the world using the
    <a href="https://atlas.ripe.net/" target="_blank" rel="noreferrer">RIPE Atlas</a>
    network — the remote counterpart to the local traceroute. Reuses the Atlas
    key you set in Remote ping; it stays on this machine and is sent only to RIPE.
  </p>

  {#if keyErr}
    <div class="banner bad"><span class="vlabel">Key problem</span><span class="vnote">{keyErr}</span></div>
  {/if}

  {#if !keySet}
    <div class="card span">
      <div class="k">RIPE Atlas key needed</div>
      <p class="note">
        This tool needs a RIPE Atlas API key. Add it once in the
        <strong>Remote ping</strong> tool (under Internet) — it is shared across
        the Atlas tools and stored only on this machine.
      </p>
    </div>
  {:else}
    <form on:submit|preventDefault={run} class="row">
      <input
        class="field mono"
        bind:value={target}
        placeholder="1.1.1.1  ·  example.com"
        autocomplete="off"
        spellcheck="false"
        aria-label="Target"
      />
      <label class="probes">
        <span>Probes</span>
        <input class="field num mono" type="number" min="1" max="20" bind:value={probes} aria-label="Probe count" />
      </label>
      <button class="go" disabled={busy}>{busy ? "Measuring…" : "Run"}</button>
    </form>

    {#if busy}
      <p class="measuring">Scheduling the traceroute and waiting for probes to report — this takes a little time…</p>
    {/if}

    {#if error}
      <div class="banner bad"><span class="vlabel">Measurement failed</span><span class="vnote">{error}</span></div>
    {/if}

    {#if r}
      {#if r.probes.length > 0}
        <div class="verdict"
          class:good={r.reached > 0}
          class:warn={r.reached === 0}>
          <span class="vlabel">
            {r.responding} / {r.probes.length} probes traced · {r.reached} reached {r.target}
          </span>
          <span class="vnote mono">msm #{r.measurementId}</span>
        </div>

        {#each r.probes as p}
          <div class="card span">
            <div class="phead">
              <span class="pid mono">#{p.id}</span>
              <span class="pfrom mono">{p.from || "—"}</span>
              <span class="reached" class:ok={p.reached}>{p.reached ? "reached " + p.dest : "did not reach"}</span>
            </div>
            <table class="mono hops"><tbody>
              {#each p.hops as h}
                <tr class:timeout={h.timeout}>
                  <td class="hn">{h.number}</td>
                  <td class="haddr">{h.timeout ? "*" : (h.from || "*")}</td>
                  <td class="hrtt">{h.timeout ? "" : fmt(h.rttMs) + " ms"}</td>
                </tr>
              {/each}
            </tbody></table>
          </div>
        {/each}
      {:else}
        <div class="banner warn">
          <span class="vlabel">No probes reported in time</span>
          <span class="vnote">The measurement (msm #{r.measurementId}) was scheduled but no probes returned results before the timeout — try again, or check your Atlas credits.</span>
        </div>
      {/if}
    {/if}
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 20px; }
  .lede { max-width: 820px; margin: 0 0 14px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .lede a { color: var(--accent); }
  .row { display: flex; gap: 8px; max-width: 820px; align-items: center; flex-wrap: wrap; }
  .field { flex: 1; min-width: 180px; background: var(--well); border: 1px solid var(--field-border); border-radius: 7px; padding: 9px 12px; color: var(--text); font-size: 0.9rem; outline: none; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .field.num { width: 64px; min-width: 64px; flex: 0 0 auto; }
  .mono { font-family: var(--mono, ui-monospace, monospace); }
  .go { background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px; padding: 0 16px; height: 38px; font: inherit; font-size: 0.9rem; cursor: pointer; white-space: nowrap; }
  .go:disabled { opacity: .6; cursor: default; }
  .probes { display: flex; align-items: center; gap: 7px; font-size: 0.82rem; color: var(--text-dim); }
  .note { font-size: 0.8rem; color: var(--text-dim); line-height: 1.55; margin: 0; max-width: 760px; }
  .measuring { font-size: 0.8rem; color: var(--text-dim); margin: 12px 0; max-width: 820px; }

  .verdict, .banner { margin: 14px 0; padding: 11px 14px; border-radius: 9px; border: 1px solid var(--border-soft); display: flex; flex-direction: column; gap: 3px; max-width: 820px; }
  .verdict.good { border-color: transparent; background: var(--good-bg, #102516); }
  .verdict.warn, .banner.warn { border-color: var(--warn, #d9a441); background: color-mix(in srgb, var(--warn, #d9a441) 12%, transparent); }
  .banner.bad { border-color: var(--bad, #ff6b6b); background: color-mix(in srgb, var(--bad, #ff6b6b) 10%, transparent); }
  .vlabel { font-weight: 600; color: var(--text-strong); }
  .verdict.good .vlabel { color: var(--good, #5ad17a); }
  .verdict.warn .vlabel, .banner.warn .vlabel { color: var(--warn, #d9a441); }
  .banner.bad .vlabel { color: var(--bad, #ff6b6b); }
  .vnote { font-size: 0.8rem; color: var(--text-dim); overflow-wrap: anywhere; }

  .card { background: var(--card); border: 1px solid var(--border-soft); border-radius: 10px; padding: 11px 15px; max-width: 820px; margin-bottom: 11px; }
  .k { font-size: 0.74rem; text-transform: uppercase; letter-spacing: 0.04em; color: var(--text-faint); margin-bottom: 9px; }
  .phead { display: flex; align-items: baseline; gap: 10px; margin-bottom: 7px; font-size: 0.82rem; flex-wrap: wrap; }
  .pid { color: var(--text-dim); }
  .pfrom { color: var(--text); margin-right: auto; }
  .reached { font-size: 0.74rem; color: var(--warn, #d9a441); }
  .reached.ok { color: var(--good, #5ad17a); }

  table.hops { width: 100%; border-collapse: collapse; font-size: 0.8rem; }
  .hops td { padding: 2px 10px 2px 0; vertical-align: top; overflow-wrap: anywhere; }
  .hn { width: 28px; color: var(--text-faint); text-align: right; }
  .haddr { color: var(--text); }
  .hrtt { text-align: right; color: var(--text-dim); white-space: nowrap; width: 80px; }
  tr.timeout .haddr { color: var(--text-faint); }
</style>
