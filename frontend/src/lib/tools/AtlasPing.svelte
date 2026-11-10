<script lang="ts">
  import { onMount } from "svelte";
  import {
    AtlasKeyStatus,
    SetAtlasKey,
    ClearAtlasKey,
    AtlasPing,
  } from "../../../wailsjs/go/app/App.js";

  // Key state
  let keySet = false;
  let keyHint = "";
  let keyInput = "";
  let editing = false; // force the setup panel even when a key is set (Change)
  let saving = false;
  let keyError = "";

  // Ping state
  let target = "";
  let probes = 10;
  let busy = false;
  let error = "";
  let r: any = null;

  onMount(refreshStatus);

  async function refreshStatus() {
    try {
      const s = await AtlasKeyStatus();
      keySet = s.set;
      keyHint = s.hint;
    } catch (e) {
      keyError = String(e);
    }
  }

  async function saveKey() {
    const v = keyInput.trim();
    if (!v) return;
    keyError = "";
    saving = true;
    try {
      await SetAtlasKey(v);
      keyInput = "";
      editing = false;
      await refreshStatus();
    } catch (e) {
      keyError = String(e);
    } finally {
      saving = false;
    }
  }

  async function removeKey() {
    keyError = "";
    try {
      await ClearAtlasKey();
      r = null;
      error = "";
      await refreshStatus();
    } catch (e) {
      keyError = String(e);
    }
  }

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
      r = await AtlasPing(v, probes);
    } catch (e) {
      error = String(e);
    } finally {
      busy = false;
    }
  }

  const fmt = (v: number) => (v == null ? "–" : v.toFixed(1));
  const loss = (p: any) =>
    p.sent > 0 ? Math.round(((p.sent - p.received) / p.sent) * 100) : 0;
</script>

<div class="tool">
  <p class="lede">
    Ping a target from vantage points around the world using the
    <a href="https://atlas.ripe.net/" target="_blank" rel="noreferrer">RIPE Atlas</a>
    measurement network — a remote looking glass, not just a ping from this
    machine. You bring your own free Atlas API key; it stays on this computer
    and is sent only to RIPE Atlas.
  </p>

  {#if keyError}
    <div class="banner bad"><span class="vlabel">Key problem</span><span class="vnote">{keyError}</span></div>
  {/if}

  {#if !keySet || editing}
    <!-- Setup panel -->
    <div class="card span setup">
      <div class="k">RIPE Atlas API key</div>
      <p class="note">
        Create a free key at
        <a href="https://atlas.ripe.net/keys/" target="_blank" rel="noreferrer">atlas.ripe.net/keys/</a>
        with the “Schedule a new measurement” permission. Running measurements
        spends Atlas <strong>credits</strong> — you earn these by hosting a probe,
        or request a starter allotment from RIPE. Your key is stored only on this
        machine (<span class="mono">settings.json</span>, owner-only) and never leaves
        it except to RIPE Atlas.
      </p>
      <form on:submit|preventDefault={saveKey} class="row">
        <input
          class="field mono"
          type="password"
          bind:value={keyInput}
          placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
          autocomplete="off"
          spellcheck="false"
          aria-label="RIPE Atlas API key"
        />
        <button class="go" disabled={saving || !keyInput.trim()}>{saving ? "Saving…" : "Save key"}</button>
        {#if editing}
          <button type="button" class="ghost" on:click={() => { editing = false; keyInput = ""; }}>Cancel</button>
        {/if}
      </form>
    </div>
  {:else}
    <!-- Key indicator -->
    <div class="keyline">
      <span class="dot"></span>
      <span class="ktext">Atlas key set · <span class="mono">{keyHint}</span></span>
      <button type="button" class="ghost" on:click={() => { editing = true; }}>Change</button>
      <button type="button" class="ghost danger" on:click={removeKey}>Remove</button>
    </div>

    <!-- Ping form -->
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
        <input class="field num mono" type="number" min="1" max="50" bind:value={probes} aria-label="Probe count" />
      </label>
      <button class="go" disabled={busy}>{busy ? "Measuring…" : "Run"}</button>
    </form>

    {#if busy}
      <p class="measuring">Scheduling the measurement and waiting for probes to report — this takes a little time…</p>
    {/if}

    {#if error}
      <div class="banner bad"><span class="vlabel">Measurement failed</span><span class="vnote">{error}</span></div>
    {/if}

    {#if r}
      {#if r.responding > 0}
        <div class="verdict good">
          <span class="vlabel">{r.responding} / {r.probes.length} probes responding · avg {fmt(r.avgMs)} ms</span>
          <span class="vnote">
            min {fmt(r.minMs)} · avg {fmt(r.avgMs)} · max {fmt(r.maxMs)} ms · mean loss {Math.round(r.lossPct)}%
            <span class="mono">· msm #{r.measurementId}</span>
          </span>
        </div>

        <div class="card span">
          <div class="k">Per probe</div>
          <table class="mono"><thead>
            <tr><th>Probe</th><th>From</th><th class="r">min</th><th class="r">avg</th><th class="r">max</th><th class="r">loss</th></tr>
          </thead><tbody>
            {#each r.probes as p}
              <tr>
                <td class="pid">#{p.id}</td>
                <td class="from">{p.from || "—"}</td>
                <td class="r">{fmt(p.minMs)}</td>
                <td class="r strong">{fmt(p.avgMs)}</td>
                <td class="r">{fmt(p.maxMs)}</td>
                <td class="r" class:lossy={loss(p) > 0}>{loss(p)}%</td>
              </tr>
            {/each}
          </tbody></table>
        </div>
      {:else}
        <div class="banner warn">
          <span class="vlabel">No probes reported in time</span>
          <span class="vnote">The measurement (msm #{r.measurementId}) was scheduled but no probes returned results before the timeout. Try again, or check your Atlas credits.</span>
        </div>
      {/if}
    {/if}
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 20px; }
  .lede { max-width: 820px; margin: 0 0 14px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .lede a, .note a { color: var(--accent); }
  .row { display: flex; gap: 8px; max-width: 820px; align-items: center; flex-wrap: wrap; }
  .field { flex: 1; min-width: 180px; background: var(--well); border: 1px solid var(--field-border); border-radius: 7px; padding: 9px 12px; color: var(--text); font-size: 0.9rem; outline: none; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .field.num { width: 64px; min-width: 64px; flex: 0 0 auto; }
  .mono { font-family: var(--mono, ui-monospace, monospace); }
  .go { background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px; padding: 0 16px; height: 38px; font: inherit; font-size: 0.9rem; cursor: pointer; white-space: nowrap; }
  .go:disabled { opacity: .6; cursor: default; }
  .ghost { background: transparent; border: 1px solid var(--border-soft); color: var(--text-dim); border-radius: 7px; padding: 5px 11px; font: inherit; font-size: 0.8rem; cursor: pointer; }
  .ghost:hover { color: var(--text); border-color: var(--field-border); }
  .ghost.danger:hover { color: var(--bad, #ff6b6b); border-color: var(--bad, #ff6b6b); }

  .probes { display: flex; align-items: center; gap: 7px; font-size: 0.82rem; color: var(--text-dim); }

  .setup { margin-top: 2px; }
  .note { font-size: 0.8rem; color: var(--text-dim); line-height: 1.55; margin: 0 0 12px; max-width: 760px; }

  .keyline { display: flex; align-items: center; gap: 10px; margin: 2px 0 14px; font-size: 0.84rem; color: var(--text); }
  .dot { width: 8px; height: 8px; border-radius: 50%; background: var(--good, #5ad17a); flex: 0 0 auto; }
  .ktext { margin-right: auto; }

  .measuring { font-size: 0.8rem; color: var(--text-dim); margin: 12px 0; max-width: 820px; }

  .verdict, .banner { margin: 14px 0; padding: 11px 14px; border-radius: 9px; border: 1px solid var(--border-soft); display: flex; flex-direction: column; gap: 3px; max-width: 820px; }
  .verdict.good { border-color: transparent; background: var(--good-bg, #102516); }
  .banner.warn { border-color: var(--warn, #d9a441); background: color-mix(in srgb, var(--warn, #d9a441) 12%, transparent); }
  .banner.bad { border-color: var(--bad, #ff6b6b); background: color-mix(in srgb, var(--bad, #ff6b6b) 10%, transparent); }
  .vlabel { font-weight: 600; color: var(--text-strong); }
  .verdict.good .vlabel { color: var(--good, #5ad17a); }
  .banner.warn .vlabel { color: var(--warn, #d9a441); }
  .banner.bad .vlabel { color: var(--bad, #ff6b6b); }
  .vnote { font-size: 0.8rem; color: var(--text-dim); overflow-wrap: anywhere; }

  .card { background: var(--card); border: 1px solid var(--border-soft); border-radius: 10px; padding: 13px 15px; max-width: 820px; margin-bottom: 13px; }
  .k { font-size: 0.74rem; text-transform: uppercase; letter-spacing: 0.04em; color: var(--text-faint); margin-bottom: 9px; }

  table { width: 100%; border-collapse: collapse; font-size: 0.8rem; }
  th { text-align: left; color: var(--text-faint); font-weight: 500; padding: 3px 10px 6px 0; border-bottom: 1px solid var(--border-soft); }
  th.r { text-align: right; }
  td { padding: 4px 10px 4px 0; vertical-align: top; border-bottom: 1px solid var(--border-soft); overflow-wrap: anywhere; }
  tr:last-child td { border-bottom: none; }
  td.r, th.r { text-align: right; white-space: nowrap; }
  td.strong { color: var(--text-strong); font-weight: 600; }
  .pid { color: var(--text-dim); white-space: nowrap; }
  .from { color: var(--text); }
  .lossy { color: var(--warn, #d9a441); }
</style>
