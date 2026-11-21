<script lang="ts">
  import { GUIReference, SaveGUIReferencePDF } from "../../../wailsjs/go/app/App.js";
  import ToolHelp from "../ToolHelp.svelte";

  let groups: any[] = [];
  let filter = "";

  GUIReference().then((g) => (groups = g || [])).catch(() => {});

  let pdfBusy = false;
  let pdfMsg = "";
  async function exportPDF() {
    pdfBusy = true;
    pdfMsg = "";
    try {
      const path = await SaveGUIReferencePDF();
      pdfMsg = path ? `Saved to ${path}` : "";
    } catch (e: any) {
      pdfMsg = String(e?.message ?? e);
    } finally {
      pdfBusy = false;
    }
  }

  // Case-insensitive filter over tool name + description.
  $: f = filter.trim().toLowerCase();
  $: shown = groups
    .map((g) => ({
      ...g,
      tools: (g.tools || []).filter((t: any) => !f || t.name.toLowerCase().includes(f) || t.desc.toLowerCase().includes(f)),
    }))
    .filter((g) => g.tools.length > 0);
  $: total = groups.reduce((n, g) => n + (g.tools ? g.tools.length : 0), 0);
</script>

<div class="tool">
  <p class="lede">
    Everything Meridian does, grouped the same way as the sidebar. Use this to get the lay of the land, or to
    point someone at the right tool. Each one is also a <strong>command line</strong> subcommand — see the
    Command line tool for the exact syntax.
  </p>

  <ToolHelp>
    This is a map of the app, not a result. Every tool in the sidebar is listed here with a plain-English note on
    what it does and when you'd reach for it. Filter to find one by name or by what it's for (try
    <code>switch</code>, <code>certificate</code>, <code>voip</code>), then open it from the sidebar. Export the
    whole thing as a <strong>PDF</strong> to share or print — it's generated from the live tool list, so it's never
    out of date.
  </ToolHelp>

  <div class="exportrow">
    <button class="pill btn" on:click={exportPDF} disabled={pdfBusy}>{pdfBusy ? "Generating…" : "Export tools overview (PDF)"}</button>
    <span class="sdim">every tool, grouped, as a PDF — current as of this build</span>
    {#if pdfMsg}<span class="sdim">{pdfMsg}</span>{/if}
  </div>

  <input class="field" bind:value={filter} placeholder="Filter tools… (e.g. switch, certificate, voip)" spellcheck="false" />

  {#each shown as g}
    <div class="group">
      <div class="gh">{g.name}</div>
      {#each g.tools as t}
        <div class="trow">
          <span class="name">{t.name}</span>
          <span class="desc">{t.desc}</span>
        </div>
      {/each}
    </div>
  {/each}

  {#if f && shown.length === 0}
    <p class="empty">No tool matches “{filter}”.</p>
  {:else if !f}
    <p class="count">{total} tools across {groups.length} groups.</p>
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 30px; max-width: 920px; }
  .lede { margin: 0 0 12px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .exportrow { display: flex; align-items: center; flex-wrap: wrap; gap: 8px 12px; margin-bottom: 12px; }
  .pill { font-size: 0.74rem; font-weight: 600; padding: 3px 10px; border-radius: 20px; }
  .pill.btn { background: var(--accent); color: var(--on-accent, #052e2b); border: none; cursor: pointer; }
  .pill.btn:disabled { opacity: .6; cursor: default; }
  .sdim { font-size: 0.74rem; color: var(--text-faint, var(--text-dim)); }
  .field { width: 100%; box-sizing: border-box; background: var(--well); border: 1px solid var(--field-border, var(--border)); border-radius: 8px; padding: 9px 12px; color: var(--text); font-size: 0.86rem; outline: none; margin-bottom: 14px; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .group { margin-bottom: 16px; }
  .gh { font-size: 0.72rem; text-transform: uppercase; letter-spacing: .05em; color: var(--text-dim); font-weight: 700; margin-bottom: 6px; }
  .trow { display: grid; grid-template-columns: minmax(130px, 0.4fr) 1fr; align-items: baseline; gap: 2px 14px; background: var(--card); border: 1px solid var(--border); border-radius: 9px; padding: 8px 12px; margin-bottom: 6px; }
  .name { color: var(--accent); font-weight: 600; font-size: 0.82rem; }
  .desc { font-size: 0.8rem; color: var(--text-dim); line-height: 1.45; }
  .count { font-size: 0.76rem; color: var(--text-faint, var(--text-dim)); margin-top: 4px; }
  .empty { font-size: 0.82rem; color: var(--text-dim); }
  @media (max-width: 560px) {
    .trow { grid-template-columns: 1fr; gap: 3px; }
  }
</style>
