<script lang="ts">
  import { MACVendor } from "../../../wailsjs/go/app/App.js";
  import { get } from "svelte/store";
  import { requestedTool } from "../nav";
  import ToolHelp from "../ToolHelp.svelte";

  let query = "";
  const _req = get(requestedTool);
  if (_req && _req.tool === "macvendor" && _req.value) { query = _req.value; requestedTool.set(null); }
  let busy = false;
  let error = "";
  let result: any = null;

  async function run() {
    const v = query.trim();
    if (!v) {
      result = null;
      return;
    }
    error = "";
    result = null;
    busy = true;
    try {
      result = await MACVendor(v);
    } catch (e) {
      result = null;
      error = String(e);
    } finally {
      busy = false;
    }
  }
</script>

<div class="tool">
  <p class="lede">
    Identifies the maker of a network card from its MAC address. The first three bytes
    are the <b>OUI</b> — an IEEE-assigned vendor ID. A locally-administered or randomized
    MAC has no vendor. The lookup sends only the OUI, nothing about you.
  </p>

  <ToolHelp>
    A green hit names the <strong>manufacturer</strong> registered to that OUI — useful for telling what a device on your
    network actually is (an Apple, Cisco, or Raspberry Pi NIC, say). A <strong>not registered</strong> result usually
    means the MAC is <strong>locally-administered or randomized</strong> (common on phones doing privacy randomization and
    on virtual interfaces), so there's no real vendor to find — not an error.
  </ToolHelp>

  <form on:submit|preventDefault={run} class="row">
    <input
      class="field mono"
      bind:value={query}
      placeholder="fc:fb:fb:dd:ee:ff  ·  fc:fb:fb  ·  fcfbfb"
      autocomplete="off"
      spellcheck="false"
      aria-label="MAC address or OUI"
    />
    <button class="go" disabled={busy}>{busy ? "Looking up…" : "Look up"}</button>
  </form>

  {#if error}<div class="banner bad"><span class="vlabel">Couldn't look up</span><span class="vnote">{error}</span></div>{/if}

  {#if result}
    {#if result.found}
      <div class="card hit">
        <div class="k">OUI {result.oui}</div>
        <div class="vendor">{result.vendor}</div>
      </div>
    {:else}
      <div class="banner warn">
        <span class="vlabel">OUI {result.oui} — not a registered vendor</span>
        <span class="vnote">
          {#if result.locallyAdministered}
            This is a locally-administered or randomized MAC (the first byte's 0x02 bit is set), so it has no manufacturer — common for privacy MAC randomization and virtual interfaces.
          {:else}
            This OUI isn't in the registry — it may be unassigned.
          {/if}
        </span>
      </div>
    {/if}
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 20px; }
  .lede { max-width: 760px; margin: 0 0 14px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .row { display: flex; gap: 8px; max-width: 760px; }
  .field {
    flex: 1; background: var(--well); border: 1px solid var(--field-border); border-radius: 7px;
    padding: 9px 12px; color: var(--text); font-size: 0.9rem; outline: none;
  }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .mono { font-family: var(--mono, ui-monospace, monospace); }
  .go {
    background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px;
    padding: 0 16px; font: inherit; font-size: 0.9rem; cursor: pointer; white-space: nowrap;
  }
  .go:disabled { opacity: .6; cursor: default; }

  .card { background: var(--card); border: 1px solid var(--border-soft); border-radius: 10px; padding: 14px 16px; margin-top: 16px; max-width: 760px; }
  .card.hit { border-color: transparent; background: var(--good-bg, #102516); }
  .k { font-size: 0.74rem; text-transform: uppercase; letter-spacing: 0.04em; color: var(--text-faint); margin-bottom: 6px; font-family: var(--mono, ui-monospace, monospace); }
  .vendor { font-size: 1.15rem; font-weight: 600; color: var(--good, #5ad17a); }

  .banner { margin-top: 16px; padding: 11px 14px; border-radius: 9px; border: 1px solid var(--border-soft); display: flex; flex-direction: column; gap: 3px; max-width: 760px; }
  .banner.warn { border-color: var(--warn, #d9a441); background: color-mix(in srgb, var(--warn, #d9a441) 12%, transparent); }
  .banner.bad { border-color: var(--bad, #ff6b6b); background: color-mix(in srgb, var(--bad, #ff6b6b) 10%, transparent); }
  .vlabel { font-weight: 600; color: var(--text-strong); }
  .banner.warn .vlabel { color: var(--warn, #d9a441); }
  .banner.bad .vlabel { color: var(--bad, #ff6b6b); }
  .vnote { font-size: 0.82rem; color: var(--text-dim); line-height: 1.45; }
</style>
