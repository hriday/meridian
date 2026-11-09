<script lang="ts">
  import { Classify, LookupIP, LookupASN } from "../../../wailsjs/go/app/App.js";
  import { Kind, kindLabel } from "../kinds";
  import EntityPage from "../EntityPage.svelte";

  let query = "1.1.1.1";
  let busy = false;
  let error = "";
  let kind: "ip" | "asn" | null = null;
  let report: any = null;

  async function run(value?: string) {
    const v = (value ?? query).trim();
    query = v;
    if (!v) {
      report = null;
      return;
    }
    error = "";
    busy = true;
    try {
      const r = await Classify(v);
      if (r.Kind === Kind.IPv4 || r.Kind === Kind.IPv6) {
        report = await LookupIP(r.Normalized);
        kind = "ip";
      } else if (r.Kind === Kind.ASN) {
        report = await LookupASN(r.Normalized);
        kind = "asn";
      } else {
        report = null;
        kind = null;
        error = `Enter an IP address or ASN — “${v}” looks like a ${kindLabel(r.Kind)}.`;
      }
    } catch (e) {
      report = null;
      error = String(e);
    } finally {
      busy = false;
    }
  }

  function pivot(value: string) {
    run(value);
  }

  // Resolve the default value on mount so the panel is never empty.
  run();
</script>

<div class="tool">
  <form on:submit|preventDefault={() => run()} class="row">
    <input
      class="field mono"
      bind:value={query}
      placeholder="1.1.1.1   ·   2606:4700::   ·   AS13335"
      autocomplete="off"
      spellcheck="false"
      aria-label="IP address or ASN"
    />
    <button class="go" disabled={busy}>{busy ? "Looking up…" : "Look up"}</button>
  </form>

  {#if error}<div class="hint">{error}</div>{/if}
  {#if report && kind}
    <EntityPage type={kind} data={report} onpivot={pivot} />
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 4px; }
  .row { display: flex; gap: 8px; max-width: 980px; }
  .field {
    flex: 1; background: var(--well); border: 1px solid var(--field-border); border-radius: 7px;
    padding: 9px 12px; color: var(--text); font-size: 0.9rem; outline: none;
  }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .go {
    background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px;
    padding: 0 16px; font: inherit; font-size: 0.9rem; cursor: pointer; white-space: nowrap;
  }
  .go:disabled { opacity: .6; cursor: default; }
  .hint {
    background: var(--warn-bg); border: 1px solid var(--warn-border); color: var(--warn-text);
    border-radius: 7px; padding: 9px 12px; font-size: 0.85rem; margin: 12px 0 0; max-width: 980px;
  }
  .mono { font-family: var(--font-mono); }
</style>
