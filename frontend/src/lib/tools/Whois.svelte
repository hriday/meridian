<script lang="ts">
  import { Whois } from "../../../wailsjs/go/app/App.js";

  let query = "";
  let busy = false;
  let error = "";
  let r: any = null;

  async function run() {
    const v = query.trim();
    if (!v) {
      r = null;
      return;
    }
    error = "";
    r = null;
    busy = true;
    try {
      r = await Whois(v);
    } catch (e) {
      error = String(e);
    } finally {
      busy = false;
    }
  }
</script>

<div class="tool">
  <p class="lede">
    The raw port-43 Whois record for a domain, IP, or ASN — the authoritative registry
    text. Meridian discovers the right server via IANA and follows its referrals. For the
    structured view of a domain's registration, use <span class="mono">Lookup</span> →
    Registry. The query carries only what you type.
  </p>

  <form on:submit|preventDefault={run} class="row">
    <input class="field mono" bind:value={query} placeholder="example.com  ·  1.1.1.1  ·  AS13335"
      autocomplete="off" spellcheck="false" aria-label="domain, IP, or ASN" />
    <button class="go" disabled={busy}>{busy ? "Looking up…" : "Look up"}</button>
  </form>

  {#if error}<div class="banner bad"><span class="vlabel">Couldn't query Whois</span><span class="vnote">{error}</span></div>{/if}

  {#if r}
    {#if r.chain?.length}
      <div class="chain">
        {#each r.chain as s, i}
          {#if i > 0}<span class="arrow">→</span>{/if}
          <span class="server mono" class:final={i === r.chain.length - 1}>{s}</span>
        {/each}
      </div>
    {/if}
    <pre class="record mono">{r.raw}</pre>
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 20px; }
  .lede { max-width: 820px; margin: 0 0 14px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .row { display: flex; gap: 8px; max-width: 760px; }
  .field { flex: 1; background: var(--well); border: 1px solid var(--field-border); border-radius: 7px; padding: 9px 12px; color: var(--text); font-size: 0.9rem; outline: none; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .mono { font-family: var(--mono, ui-monospace, monospace); }
  .go { background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px; padding: 0 16px; font: inherit; font-size: 0.9rem; cursor: pointer; white-space: nowrap; }
  .go:disabled { opacity: .6; cursor: default; }

  .banner { margin: 14px 0; padding: 11px 14px; border-radius: 9px; border: 1px solid var(--border-soft); display: flex; flex-direction: column; gap: 3px; max-width: 820px; }
  .banner.bad { border-color: var(--bad, #ff6b6b); background: color-mix(in srgb, var(--bad, #ff6b6b) 10%, transparent); }
  .vlabel { font-weight: 600; color: var(--bad, #ff6b6b); }
  .vnote { font-size: 0.82rem; color: var(--text-dim); }

  .chain { display: flex; align-items: center; flex-wrap: wrap; gap: 7px; margin: 14px 0 10px; font-size: 0.8rem; }
  .arrow { color: var(--text-faint); }
  .server { color: var(--text-dim); padding: 2px 8px; border-radius: 6px; border: 1px solid var(--border-soft); background: var(--well); }
  .server.final { color: var(--accent); border-color: transparent; background: var(--active); }
  .record {
    margin: 0; max-width: 100%; background: var(--card); border: 1px solid var(--border-soft);
    border-radius: 10px; padding: 14px 16px; font-size: 0.78rem; line-height: 1.5; color: var(--text);
    white-space: pre-wrap; overflow-wrap: anywhere; max-height: 70vh; overflow-y: auto;
  }
</style>
