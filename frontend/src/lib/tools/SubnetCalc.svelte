<script lang="ts">
  import { Classify, SubnetDescribe, SubnetSplit } from "../../../wailsjs/go/app/App.js";
  import { subnet } from "../../../wailsjs/go/models";
  import { isPrefix, kindLabel } from "../kinds";

  let query = "10.0.0.0/24";
  let info: subnet.Info | null = null;
  let hint = "";
  let busy = false;
  let splits: string[] = [];
  let splitLen: number | null = null;

  async function run() {
    hint = "";
    splits = [];
    const value = query.trim();
    if (!value) {
      info = null;
      return;
    }
    busy = true;
    try {
      const r = await Classify(value);
      if (!isPrefix(r.Kind)) {
        info = null;
        hint = `That looks like a ${kindLabel(r.Kind)}. Enter a CIDR block such as 10.0.0.0/24 or 2001:db8::/48.`;
        return;
      }
      info = await SubnetDescribe(value);
      splitLen = info.Version === 4 ? Math.min(info.PrefixLen + 1, 32) : Math.min(info.PrefixLen + 1, 128);
    } catch (e) {
      info = null;
      hint = String(e);
    } finally {
      busy = false;
    }
  }

  async function doSplit() {
    if (!info || splitLen == null) return;
    try {
      splits = await SubnetSplit(info.Prefix, splitLen);
    } catch (e) {
      hint = String(e);
    }
  }

  // Describe the default on first mount so the panel is never empty.
  run();
</script>

<div class="tool">
  <h2>Subnet calculator</h2>
  <p class="sub">IPv4 and IPv6. Enter a CIDR block to see its range, mask, and usable addresses.</p>

  <form on:submit|preventDefault={run} class="row">
    <input
      class="field mono"
      bind:value={query}
      placeholder="10.0.0.0/24  or  2001:db8::/48"
      autocomplete="off"
      spellcheck="false"
      aria-label="CIDR block"
    />
    <button class="go" disabled={busy}>Describe</button>
  </form>

  {#if hint}
    <div class="hint">{hint}</div>
  {/if}

  {#if info}
    <div class="cards">
      <div class="card">
        <h3>Block</h3>
        <dl class="mono">
          <dt>Prefix</dt><dd>{info.Prefix}</dd>
          <dt>Network</dt><dd>{info.Network}</dd>
          {#if info.Broadcast}<dt>Broadcast</dt><dd>{info.Broadcast}</dd>{/if}
          {#if info.Netmask}<dt>Netmask</dt><dd>{info.Netmask}</dd>{/if}
          {#if info.Wildcard}<dt>Wildcard</dt><dd>{info.Wildcard}</dd>{/if}
          <dt>Addresses</dt><dd>{info.TotalCount}</dd>
        </dl>
      </div>
      <div class="card">
        <h3>Usable</h3>
        <dl class="mono">
          <dt>First</dt><dd>{info.UsableFirst}</dd>
          <dt>Last</dt><dd>{info.UsableLast}</dd>
          <dt>Count</dt><dd>{info.UsableCount}</dd>
        </dl>
        <div class="split">
          <label>Split into /<input class="len mono" type="number" bind:value={splitLen} min={info.PrefixLen + 1} max={info.Version === 4 ? 32 : 128} /></label>
          <button class="ghost" on:click={doSplit}>Split</button>
        </div>
      </div>
    </div>

    {#if splits.length}
      <div class="card">
        <h3>{splits.length} subnets</h3>
        <ul class="subnets mono">
          {#each splits as s}<li>{s}</li>{/each}
        </ul>
      </div>
    {/if}
  {/if}
</div>

<style>
  .tool { padding: 18px 20px; max-width: 760px; }
  h2 { margin: 0 0 2px; font-size: 1.1rem; font-weight: 600; color: var(--text-strong); }
  .sub { margin: 0 0 14px; color: var(--text-dim); font-size: 0.85rem; }
  .row { display: flex; gap: 8px; margin-bottom: 14px; }
  .field {
    flex: 1; background: var(--well); border: 1px solid var(--field-border); border-radius: 7px;
    padding: 9px 12px; color: var(--text); font-size: 0.9rem; outline: none;
  }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .go {
    background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px;
    padding: 0 16px; font-size: 0.9rem; cursor: pointer;
  }
  .go:disabled { opacity: .6; cursor: default; }
  .hint {
    background: var(--warn-bg); border: 1px solid var(--warn-border); color: var(--warn-text);
    border-radius: 7px; padding: 9px 12px; font-size: 0.85rem; margin-bottom: 12px;
  }
  .cards { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
  @media (max-width: 620px) { .cards { grid-template-columns: 1fr; } }
  .card {
    background: var(--card); border: 1px solid var(--border-soft); border-radius: 9px;
    padding: 12px 14px; margin-bottom: 12px;
  }
  .card h3 {
    margin: 0 0 8px; font-size: 0.7rem; text-transform: uppercase;
    letter-spacing: .06em; color: var(--text-dim); font-weight: 600;
  }
  dl { display: grid; grid-template-columns: 92px 1fr; gap: 4px 10px; margin: 0; font-size: 0.85rem; }
  dt { color: var(--text-dim); }
  dd { margin: 0; color: var(--text); }
  .mono { font-family: var(--font-mono); }
  .split { display: flex; align-items: center; gap: 10px; margin-top: 12px; }
  .split label { color: var(--text-dim); font-size: 0.85rem; }
  .len { width: 56px; background: var(--well); border: 1px solid var(--field-border); border-radius: 5px; color: var(--text); padding: 4px 6px; }
  .ghost { background: var(--ghost-bg); border: 1px solid var(--ghost-border); color: var(--text); border-radius: 6px; padding: 4px 12px; cursor: pointer; }
  .subnets { list-style: none; margin: 0; padding: 0; columns: 3; font-size: 0.82rem; color: var(--text); }
  @media (max-width: 620px) { .subnets { columns: 2; } }
  .subnets li { padding: 1px 0; }
</style>
