<script lang="ts">
  import { WWNDecode } from "../../../wailsjs/go/app/App.js";
  import ToolHelp from "../ToolHelp.svelte";

  let input = "";
  let res: any = null;
  let error = "";
  let loading = false;

  async function decode() {
    const q = input.trim();
    if (!q) return;
    loading = true;
    error = "";
    res = null;
    try {
      res = await WWNDecode(q);
    } catch (e: any) {
      error = String(e?.message ?? e);
    } finally {
      loading = false;
    }
  }
</script>

<div class="tool">
  <p class="lede">
    Decode a Fibre Channel <strong>World Wide Name</strong> (WWN / WWPN / WWNN): its <strong>NAA format</strong>, the
    embedded <strong>IEEE OUI</strong> and the vendor it belongs to, plus clean re-formats. Paste any spelling —
    colons, Cisco dotted, or bare hex.
  </p>

  <ToolHelp>
    A WWN is a 64-bit Fibre Channel identifier. Its leading nibble is the <strong>NAA</strong> type, which fixes where
    the 24-bit <strong>OUI</strong> sits — and that OUI is the same IEEE vendor registry a MAC address uses, so the
    vendor resolves the same way. Whether a name is a <em>node</em> (WWNN) or a <em>port</em> (WWPN) isn't encoded in a
    standard, derivable way, so it isn't guessed. Only the OUI ever leaves your machine (for the vendor lookup).
  </ToolHelp>

  <div class="row">
    <input
      class="field mono"
      bind:value={input}
      placeholder="20:00:00:25:b5:00:00:01   ·   2000.0025.b500.0001"
      spellcheck="false"
      on:keydown={(e) => e.key === "Enter" && decode()}
      aria-label="WWN / WWPN"
    />
    <button class="btn" on:click={decode} disabled={loading || !input.trim()}>{loading ? "…" : "Decode"}</button>
  </div>

  {#if error}<p class="err">{error}</p>{/if}

  {#if res}
    <div class="card">
      <dl class="mono">
        <dt>WWN</dt>
        <dd class="big">{res.canonical}</dd>
        <dt>Cisco form</dt>
        <dd>{res.dotted}</dd>
        <dt>Bare</dt>
        <dd>{res.bare}</dd>
        <dt>Format</dt>
        <dd>{res.naaDesc}</dd>
        {#if res.oui}
          <dt>OUI</dt>
          <dd>{res.oui}{#if res.vendor}<span class="vendor">{res.vendor}</span>{/if}</dd>
        {/if}
        {#if res.vsid}
          <dt>Vendor-specific</dt>
          <dd>{res.vsid}</dd>
        {/if}
      </dl>
      {#if res.notes && res.notes.length}
        <ul class="notes">
          {#each res.notes as n}<li>{n}</li>{/each}
        </ul>
      {/if}
    </div>
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 30px; }
  .lede { max-width: 820px; margin: 0 0 12px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .mono { font-family: var(--font-mono, ui-monospace, monospace); }
  .row { display: flex; gap: 10px; max-width: 720px; margin-bottom: 14px; }
  .field { flex: 1 1 auto; min-width: 0; background: var(--well); border: 1px solid var(--field-border, var(--border)); border-radius: 8px; padding: 9px 12px; color: var(--text); font-size: 0.88rem; outline: none; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .btn { background: var(--accent); color: var(--on-accent, #052e2b); border: 1px solid transparent; border-radius: 8px; padding: 9px 18px; font: inherit; font-size: 0.86rem; font-weight: 550; cursor: pointer; }
  .btn:disabled { opacity: .5; cursor: default; }
  .err { color: var(--bad, #ff6b6b); font-size: 0.84rem; max-width: 820px; }
  .card { max-width: 820px; background: var(--card); border: 1px solid var(--border); border-radius: 12px; padding: 14px 16px; }
  dl { display: grid; grid-template-columns: max-content 1fr; gap: 6px 18px; margin: 0; font-size: 0.84rem; }
  dt { color: var(--text-dim); }
  dd { margin: 0; color: var(--text); overflow-wrap: anywhere; }
  dd.big { font-size: 1.05rem; font-weight: 600; letter-spacing: .02em; }
  .vendor { margin-left: 10px; font-size: 0.72rem; color: var(--on-accent, #052e2b); background: var(--accent); border-radius: 20px; padding: 1px 9px; vertical-align: 1px; }
  .notes { margin: 12px 0 0; padding-left: 18px; }
  .notes li { font-size: 0.76rem; color: var(--text-faint, var(--text-dim)); line-height: 1.5; margin-top: 3px; }
</style>
