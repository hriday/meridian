<script lang="ts">
  // A compact, consistent help affordance for a tool: an expandable
  // "How to read this" (the interpretation goes in the default slot) and an
  // optional privileged-upgrade chip for tools where sudo unlocks more data.
  export let sudo = ""; // text for the "more with elevation" chip; empty = none
  let open = false;
</script>

<div class="toolhelp">
  <button class="chip" class:on={open} on:click={() => (open = !open)} aria-expanded={open}>
    <span class="i">ⓘ</span> How to read this
  </button>
  {#if sudo}<span class="chip priv" title="This tool shows more when run with elevated privileges">🔒 {sudo}</span>{/if}
  {#if open}
    <div class="body"><slot /></div>
  {/if}
</div>

<style>
  .toolhelp { display: flex; flex-wrap: wrap; align-items: center; gap: 8px; max-width: 820px; margin: 0 0 14px; }
  .chip {
    display: inline-flex; align-items: center; gap: 5px; font: inherit; font-size: 0.74rem;
    color: var(--text-dim); background: var(--well); border: 1px solid var(--border);
    border-radius: 999px; padding: 3px 10px; cursor: pointer;
  }
  .chip:hover { border-color: var(--accent); color: var(--text); }
  .chip.on { border-color: var(--accent); color: var(--text); }
  .chip.priv { cursor: default; color: var(--warn, #d6a531); border-color: color-mix(in srgb, var(--warn, #d6a531) 45%, transparent); }
  .chip.priv:hover { color: var(--warn, #d6a531); }
  .i { font-style: normal; opacity: .85; }
  .body {
    flex-basis: 100%; font-size: 0.8rem; color: var(--text-dim); line-height: 1.55;
    background: var(--well); border: 1px solid var(--border); border-radius: 9px; padding: 10px 13px;
  }
  .body :global(strong) { color: var(--text); font-weight: 600; }
  .body :global(code) { font-family: var(--font-mono, ui-monospace, monospace); font-size: 0.92em; color: var(--text); }
</style>
