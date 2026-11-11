<script lang="ts">
  // Shared collector shell: port + Start/Stop, filter (+ optional severity
  // floor), a status line, and a scrollable newest-first list (slotted rows).
  // Presentational — the owning tool does the polling and filtering.
  export let port = 0;
  export let running = false;
  export let start: () => void;
  export let stop: () => void;
  export let filter = "";
  export let severities: { label: string; value: number }[] = [];
  export let sevFloor = 7;
  export let total = 0;
  export let dropped = 0;
  export let hosts = 0;
  export let shownCount = 0;
  export let err = "";
  export let note = "";

  function clear() {
    filter = "";
    if (severities.length) sevFloor = severities[0].value;
  }
</script>

<div class="tail">
  <div class="row">
    <span class="lbl">Listen UDP</span>
    <input class="field port mono" type="number" min="1" max="65535" bind:value={port} disabled={running} aria-label="Listen port" />
    {#if running}
      <button type="button" class="stop" on:click={stop}>Stop</button>
    {:else}
      <button type="button" class="go" on:click={start}>Start</button>
    {/if}
    <span class="spacer"></span>
    <input class="field filter mono" bind:value={filter} placeholder="filter — host · text" aria-label="Filter" spellcheck="false" autocomplete="off" />
    {#if severities.length}
      <select class="field sev mono" bind:value={sevFloor} aria-label="Severity floor">
        {#each severities as s}<option value={s.value}>{s.label}</option>{/each}
      </select>
    {/if}
    <button type="button" class="ghost" on:click={clear}>Clear</button>
  </div>

  {#if err}
    <div class="banner bad"><span class="vlabel">Couldn't start</span><span class="vnote">{err}</span></div>
  {/if}

  <div class="statusbar">
    <span class="dot" class:on={running}></span>
    <span>{running ? `listening on :${port}` : "stopped"}</span>
    {#if total || running}
      <span>·</span>
      <span><span class="count">{total.toLocaleString()}</span> message{total === 1 ? "" : "s"} from <span class="count">{hosts}</span> host{hosts === 1 ? "" : "s"} · showing {shownCount}{dropped ? ` · ${dropped.toLocaleString()} dropped` : ""}</span>
    {/if}
  </div>

  <div class="log"><slot /></div>
  {#if note}<p class="caption">{note}</p>{/if}
</div>

<style>
  .tail { max-width: 960px; }
  .mono { font-family: var(--mono, ui-monospace, monospace); }
  .row { display: flex; gap: 8px; align-items: center; }
  .lbl { font-size: 0.78rem; color: var(--text-dim); }
  .field { background: var(--well); border: 1px solid var(--field-border); border-radius: 7px; padding: 8px 11px; color: var(--text); font-size: 0.86rem; outline: none; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .field:disabled { opacity: .6; }
  .port { width: 92px; }
  .filter { flex: 1; min-width: 120px; }
  .spacer { flex: 1; }
  .go { background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px; padding: 0 15px; height: 36px; font: inherit; font-size: 0.86rem; cursor: pointer; }
  .stop { background: transparent; border: 1px solid var(--border-soft); color: var(--text-dim); border-radius: 7px; padding: 0 14px; height: 36px; font: inherit; font-size: 0.84rem; cursor: pointer; }
  .stop:hover { color: var(--text); border-color: var(--field-border); }
  .ghost { background: transparent; border: 1px solid var(--border-soft); color: var(--text-dim); border-radius: 7px; padding: 7px 11px; font: inherit; font-size: 0.78rem; cursor: pointer; }

  .banner { margin: 12px 0 0; padding: 10px 13px; border-radius: 9px; border: 1px solid var(--bad, #ff6b6b); background: color-mix(in srgb, var(--bad,#ff6b6b) 10%, transparent); display: flex; flex-direction: column; gap: 2px; }
  .vlabel { font-weight: 600; color: var(--bad, #ff6b6b); font-size: 0.82rem; }
  .vnote { font-size: 0.8rem; color: var(--text-dim); word-break: break-word; }

  .statusbar { display: flex; align-items: center; gap: 9px; margin: 13px 0 8px; font-size: 0.8rem; color: var(--text-dim); }
  .dot { width: 8px; height: 8px; border-radius: 50%; background: var(--text-faint); flex: none; }
  .dot.on { background: var(--good); }
  .count { color: var(--text-strong); font-weight: 600; }

  .log { border: 1px solid var(--border-soft); border-radius: 10px; background: var(--card); overflow-y: auto; max-height: 62vh; }
  .caption { margin-top: 12px; font-size: 0.72rem; color: var(--text-faint); }
</style>
