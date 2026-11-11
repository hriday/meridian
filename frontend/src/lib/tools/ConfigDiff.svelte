<script lang="ts">
  import { DiffConfigs } from "../../../wailsjs/go/app/App.js";

  let a = "";
  let b = "";
  let res: any = null;
  let err = "";

  async function compare() {
    err = "";
    res = null;
    try {
      res = await DiffConfigs(a, b);
    } catch (e) {
      err = String(e);
    }
  }
  function clear() {
    a = "";
    b = "";
    res = null;
    err = "";
  }
  async function loadFile(which: "a" | "b", ev: Event) {
    const f = (ev.target as HTMLInputElement).files?.[0];
    if (!f) return;
    const text = await f.text();
    if (which === "a") a = text;
    else b = text;
    (ev.target as HTMLInputElement).value = "";
  }

  // Group the diff into runs of lines belonging to a section that changed, so the
  // view shows what changed in context and skips untouched sections.
  $: groups = res ? groupChanged(res) : [];
  function groupChanged(r: any) {
    const changed = new Set(r.sections);
    const out: { section: string; lines: any[] }[] = [];
    let cur: { section: string; lines: any[] } | null = null;
    for (const l of r.lines) {
      if (!changed.has(l.section)) {
        cur = null;
        continue;
      }
      if (!cur || cur.section !== l.section) {
        cur = { section: l.section, lines: [] };
        out.push(cur);
      }
      cur.lines.push(l);
    }
    return out;
  }
</script>

<div class="tool">
  <p class="lede">
    Compare two device configs — paste an earlier and a later version (any vendor).
    Meridian lines them up and shows what changed, grouped by the section each change
    falls in. Offline; nothing leaves your machine.
  </p>

  <div class="cols">
    <div class="col">
      <div class="colhdr">
        <span class="k0">Config A · earlier</span>
        <label class="file">Load…<input type="file" on:change={(e) => loadFile("a", e)} /></label>
      </div>
      <textarea class="field mono" bind:value={a} placeholder="paste the earlier config…" spellcheck="false"></textarea>
    </div>
    <div class="col">
      <div class="colhdr">
        <span class="k0">Config B · later</span>
        <label class="file">Load…<input type="file" on:change={(e) => loadFile("b", e)} /></label>
      </div>
      <textarea class="field mono" bind:value={b} placeholder="paste the later config…" spellcheck="false"></textarea>
    </div>
  </div>

  <div class="row">
    <button type="button" class="go" on:click={compare} disabled={!a.trim() && !b.trim()}>Compare</button>
    <button type="button" class="ghost" on:click={clear}>Clear</button>
  </div>

  {#if err}
    <div class="banner bad"><span class="vlabel">Couldn't compare</span><span class="vnote">{err}</span></div>
  {/if}

  {#if res}
    <div class="summary">
      <span class="s-add">+{res.added}</span>
      <span class="s-del">−{res.removed}</span>
      <span class="s-sec">across {res.sections.length} section{res.sections.length === 1 ? "" : "s"}</span>
    </div>
    {#if res.sections.length === 0}
      <div class="nochg">No differences — the two configs are identical.</div>
    {:else}
      {#each groups as g}
        <div class="sec">
          <div class="sechdr mono">{g.section || "(top level)"}</div>
          <div class="lines mono">
            {#each g.lines as l}
              <div class="ln {l.op}"><span class="gutter">{l.op === "add" ? "+" : l.op === "del" ? "−" : " "}</span><span class="txt">{l.text || " "}</span></div>
            {/each}
          </div>
        </div>
      {/each}
    {/if}
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 20px; }
  .lede { max-width: 820px; margin: 0 0 14px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .mono { font-family: var(--mono, ui-monospace, monospace); }

  .cols { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; max-width: 980px; }
  @media (max-width: 680px) { .cols { grid-template-columns: 1fr; } }
  .col { display: flex; flex-direction: column; gap: 6px; min-width: 0; }
  .colhdr { display: flex; align-items: center; justify-content: space-between; }
  .k0 { font-size: 0.68rem; text-transform: uppercase; letter-spacing: .05em; color: var(--text-faint); }
  .file { font-size: 0.72rem; color: var(--accent); cursor: pointer; }
  .file input { display: none; }
  textarea.field {
    background: var(--well); border: 1px solid var(--field-border); border-radius: 8px;
    padding: 9px 11px; color: var(--text); font-size: 0.8rem; outline: none; width: 100%;
    min-height: 180px; resize: vertical; line-height: 1.45;
  }
  textarea.field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }

  .row { display: flex; gap: 10px; margin: 12px 0 6px; align-items: center; }
  .go { background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px; padding: 0 16px; height: 36px; font: inherit; font-size: 0.88rem; cursor: pointer; }
  .go:disabled { opacity: .55; cursor: default; }
  .ghost { background: transparent; border: 1px solid var(--border-soft); color: var(--text-dim); border-radius: 7px; padding: 0 14px; height: 36px; font: inherit; font-size: 0.84rem; cursor: pointer; }

  .banner { margin: 12px 0; padding: 11px 14px; border-radius: 9px; border: 1px solid var(--bad, #ff6b6b); background: color-mix(in srgb, var(--bad,#ff6b6b) 10%, transparent); max-width: 820px; display: flex; flex-direction: column; gap: 3px; }
  .vlabel { font-weight: 600; color: var(--bad, #ff6b6b); }
  .vnote { font-size: 0.8rem; color: var(--text-dim); overflow-wrap: anywhere; }

  .summary { margin: 14px 0 10px; font-size: 0.85rem; display: flex; gap: 12px; align-items: baseline; }
  .s-add { color: var(--good, #3fb950); font-weight: 600; font-variant-numeric: tabular-nums; }
  .s-del { color: var(--bad, #f0616a); font-weight: 600; font-variant-numeric: tabular-nums; }
  .s-sec { color: var(--text-dim); font-size: 0.8rem; }
  .nochg { color: var(--text-dim); font-size: 0.85rem; padding: 10px 0; }

  .sec { max-width: 980px; border: 1px solid var(--border-soft); border-radius: 10px; overflow: hidden; margin-bottom: 10px; }
  .sechdr { background: var(--card2, var(--well)); border-bottom: 1px solid var(--border-soft); padding: 7px 12px; font-size: 0.76rem; color: var(--text-strong); }
  .lines { overflow-x: auto; }
  .ln { display: flex; gap: 8px; padding: 1px 12px; font-size: 0.78rem; line-height: 1.5; white-space: pre; }
  .ln .gutter { width: 10px; flex: none; color: var(--text-faint); }
  .ln.same { color: var(--text-dim); }
  .ln.add { background: color-mix(in srgb, var(--good,#3fb950) 12%, transparent); color: var(--text); }
  .ln.add .gutter { color: var(--good, #3fb950); }
  .ln.del { background: color-mix(in srgb, var(--bad,#f0616a) 12%, transparent); color: var(--text); }
  .ln.del .gutter { color: var(--bad, #f0616a); }
  .txt { white-space: pre; }
</style>
