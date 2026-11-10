<script lang="ts">
  import { Interfaces } from "../../../wailsjs/go/app/App.js";

  let busy = false;
  let error = "";
  let ifaces: any[] = [];

  async function load() {
    error = "";
    busy = true;
    try {
      ifaces = await Interfaces();
    } catch (e) {
      error = String(e);
    } finally {
      busy = false;
    }
  }
  load();

  const addrs = (i: any) => [...(i.ipv4 ?? []), ...(i.ipv6 ?? [])];
</script>

<div class="tool">
  <div class="head">
    <p class="lede">
      This machine's network interfaces, straight from the OS — name, MAC, MTU, flags, and
      addresses. Entirely local; nothing is sent anywhere.
    </p>
    <button class="go" on:click={load} disabled={busy}>{busy ? "…" : "Refresh"}</button>
  </div>

  {#if error}<div class="hint">Couldn't read interfaces: {error}</div>{/if}

  <div class="cards">
    {#each ifaces as i}
      <div class="card" class:muted={!i.up || i.loopback}>
        <div class="chead">
          <span class="name mono">{i.name}</span>
          <span class="pill {i.up ? 'good' : 'down'}">{i.up ? "up" : "down"}</span>
          {#if i.loopback}<span class="pill lo">loopback</span>{/if}
        </div>
        <dl>
          {#if i.mac}<dt>MAC</dt><dd class="mono">{i.mac}</dd>{/if}
          <dt>MTU</dt><dd>{i.mtu}</dd>
          {#if i.flags?.length}
            <dt>Flags</dt>
            <dd class="chips">{#each i.flags as f}<span class="chip">{f}</span>{/each}</dd>
          {/if}
          {#if addrs(i).length}
            <dt>Addresses</dt>
            <dd class="addrs mono">{#each addrs(i) as a}<div>{a}</div>{/each}</dd>
          {/if}
        </dl>
      </div>
    {/each}
  </div>
</div>

<style>
  .tool { padding: 14px 16px 20px; }
  .head { display: flex; align-items: flex-start; gap: 12px; max-width: 900px; }
  .lede { flex: 1; margin: 0 0 14px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .go {
    background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px;
    padding: 6px 14px; font: inherit; font-size: 0.84rem; cursor: pointer; white-space: nowrap; flex: none;
  }
  .go:disabled { opacity: .6; cursor: default; }
  .hint { margin: 8px 0; color: var(--warn-text); font-size: 0.82rem; }
  .mono { font-family: var(--mono, ui-monospace, monospace); }

  .cards { display: grid; grid-template-columns: repeat(auto-fill, minmax(320px, 1fr)); gap: 13px; }
  .card { background: var(--card); border: 1px solid var(--border-soft); border-radius: 10px; padding: 13px 15px; }
  .card.muted { opacity: 0.62; }
  .chead { display: flex; align-items: center; gap: 8px; margin-bottom: 10px; }
  .name { font-size: 0.95rem; font-weight: 600; color: var(--text-strong); }
  .pill { display: inline-flex; font-size: 0.68rem; padding: 1px 8px; border-radius: 20px; border: 1px solid var(--border); color: var(--text-dim); }
  .pill.good { color: var(--good, #5ad17a); border-color: transparent; background: var(--good-bg, #102516); }
  .pill.down { color: var(--text-dim); }
  .pill.lo { color: var(--text-dim); }
  dl { display: grid; grid-template-columns: max-content 1fr; gap: 5px 14px; margin: 0; }
  dt { color: var(--text-dim); font-size: 0.8rem; }
  dd { margin: 0; font-size: 0.84rem; color: var(--text); overflow-wrap: anywhere; }
  .chips { display: flex; flex-wrap: wrap; gap: 5px; }
  .chip { display: inline-flex; font-size: 0.7rem; padding: 1px 7px; border-radius: 6px; border: 1px solid var(--border-soft); background: var(--well); color: var(--text-dim); }
  .addrs div { line-height: 1.5; }
</style>
