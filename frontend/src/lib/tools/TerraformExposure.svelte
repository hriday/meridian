<script lang="ts">
  import { AnalyzeTerraform } from "../../../wailsjs/go/app/App.js";

  const SOURCES = [
    { id: "", label: "Auto-detect" },
    { id: "hcl", label: "HCL (.tf files)" },
    { id: "json", label: "Plan JSON (terraform show -json)" },
  ];

  let configText = "";
  let source = "";
  let busy = false;
  let error = "";
  let a: any = null;
  let openFinding = -1;
  let fileInput: HTMLInputElement;

  async function analyze() {
    const v = configText.trim();
    if (!v) { a = null; return; }
    error = ""; a = null; openFinding = -1; busy = true;
    try {
      a = await AnalyzeTerraform(v, source);
    } catch (e) {
      error = String(e);
    } finally {
      busy = false;
    }
  }

  function loadFile(e: Event) {
    const f = (e.target as HTMLInputElement).files?.[0];
    if (!f) return;
    const r = new FileReader();
    r.onload = () => { configText = String(r.result ?? ""); };
    r.readAsText(f);
  }
  function clearAll() { configText = ""; a = null; error = ""; openFinding = -1; }

  const sevClass = (s: string) => (s === "critical" ? "crit" : s === "warning" ? "warn" : "info");

  // Exposure diagram: Internet on the left, exposed resources stacked on the right.
  $: diagram = buildDiagram(a);
  function buildDiagram(an: any) {
    const edges = an?.exposure ?? [];
    if (!edges.length) return null;
    if (edges.length > 10) return { tooMany: true, count: edges.length };
    const h = Math.max(140, edges.length * 58 + 20);
    const rowY = (i: number) => 30 + i * ((h - 60) / Math.max(1, edges.length - 1 || 1)) + (edges.length === 1 ? (h - 60) / 2 : 0);
    const nodes = edges.map((e: any, i: number) => ({
      target: e.target, label: e.label, severity: e.severity, y: rowY(i),
    }));
    return { tooMany: false, nodes, h, cy: h / 2 };
  }
</script>

<div class="tool">
  <p class="lede">
    Paste Terraform — <span class="mono">.tf</span> files or a <span class="mono">terraform show -json</span> plan.
    Meridian reads it locally and shows what your infrastructure exposes to the internet: open security-group
    rules, public buckets and databases, unencrypted storage — with a map of what's reachable from outside.
    Offline; nothing leaves your machine.
  </p>

  <div class="card span">
    <div class="row srcrow">
      <span class="k0">Source</span>
      <select class="srcsel" bind:value={source}>
        {#each SOURCES as s}<option value={s.id}>{s.label}</option>{/each}
      </select>
      <span class="hint">AWS resources are analyzed; others are listed.</span>
    </div>
    <textarea class="mono input" bind:value={configText}
      placeholder="Paste Terraform HCL or `terraform show -json` output here, or use Load file…" spellcheck="false"></textarea>
    <div class="row">
      <button class="go" on:click={analyze} disabled={busy || !configText.trim()}>{busy ? "Analyzing…" : "Analyze"}</button>
      <button class="ghost" on:click={() => fileInput.click()}>Load file…</button>
      <button class="ghost" on:click={clearAll} disabled={!configText && !a}>Clear</button>
      <input type="file" bind:this={fileInput} on:change={loadFile} accept=".tf,.json,.txt,text/plain" hidden />
    </div>
  </div>

  {#if error}
    <div class="banner bad"><span class="vlabel">Couldn't parse it</span><span class="vnote">{error}</span></div>
  {/if}

  {#if a}
    <div class="tiles">
      <div class="tile"><div class="n">{a.counts.resources}</div><div class="l">Resources</div></div>
      <div class="tile"><div class="n">{a.counts.securityGroups}</div><div class="l">Security groups</div></div>
      <div class="tile"><div class="n" class:bad={a.counts.exposed > 0}>{a.counts.exposed}</div><div class="l">Internet-exposed</div></div>
      <div class="tile"><div class="n" class:bad={a.counts.critical > 0}>{a.counts.critical}</div><div class="l">Critical</div></div>
      <div class="tile"><div class="n">{a.counts.findings}</div><div class="l">Findings</div></div>
    </div>

    {#if a.notes && a.notes.length}
      <div class="banner warn">{#each a.notes as note}<span class="vnote">{note}</span>{/each}</div>
    {/if}

    {#if diagram}
      <div class="card span">
        <div class="k">Internet exposure <span class="count">· what's reachable from 0.0.0.0/0</span></div>
        {#if diagram.tooMany}
          <p class="empty">{diagram.count} exposed resources — too many to diagram clearly; see the findings below.</p>
        {:else}
          <div class="diagram">
            <svg viewBox="0 0 820 {diagram.h}" width="100%" role="img" aria-label="Internet exposure diagram">
              <defs>
                <marker id="tf-crit" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse"><path d="M0,0 L10,5 L0,10 z" fill="var(--bad,#ff6b6b)"/></marker>
                <marker id="tf-warn" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse"><path d="M0,0 L10,5 L0,10 z" fill="var(--warn,#e0b341)"/></marker>
                <marker id="tf-info" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse"><path d="M0,0 L10,5 L0,10 z" fill="var(--text-dim,#8c99a8)"/></marker>
              </defs>
              <g class="inet"><rect x="20" y={diagram.cy - 28} width="120" height="56" rx="9"/>
                <text class="it" x="80" y={diagram.cy - 4} text-anchor="middle">Internet</text>
                <text class="is" x="80" y={diagram.cy + 13} text-anchor="middle">0.0.0.0/0</text></g>
              {#each diagram.nodes as n}
                <line class="edge {sevClass(n.severity)}" x1="140" y1={diagram.cy} x2="470" y2={n.y + 24} />
                <text class="elabel" x="305" y={(diagram.cy + n.y + 24) / 2 - 4} text-anchor="middle">{n.label}</text>
                <g class="res {sevClass(n.severity)}"><rect x="470" y={n.y} width="330" height="48" rx="9"/>
                  <text class="rt" x="486" y={n.y + 29}>{n.target}</text></g>
              {/each}
            </svg>
          </div>
          <div class="legend"><span class="lg-crit">critical</span><span class="lg-warn">review</span><span class="lg-info">by design — confirm</span></div>
        {/if}
      </div>
    {/if}

    <div class="card span">
      <div class="k">Findings <span class="count">· {(a.findings ?? []).length}</span></div>
      {#if (a.findings ?? []).length}
        {#each a.findings as f, i}
          <div class="finding" class:open={openFinding === i} on:click={() => (openFinding = openFinding === i ? -1 : i)} role="button" tabindex="0">
            <span class="sev {sevClass(f.severity)}">{f.severity}</span>
            <div class="fbody">
              <div class="ftitle">{f.title}</div>
              <div class="fdetail">{f.detail}</div>
              <div class="fres mono">{f.resource}</div>
            </div>
          </div>
        {/each}
      {:else}<p class="empty">No exposure or misconfiguration findings.</p>{/if}
    </div>

    <div class="card span">
      <div class="k">Resources <span class="count">· {(a.resources ?? []).length}</span></div>
      {#if (a.resources ?? []).length}
        <div class="tbl"><table class="mono">
          <thead><tr><th>Type</th><th>Name</th><th>Exposure</th><th></th></tr></thead>
          <tbody>
            {#each a.resources as r}
              <tr>
                <td class="dim">{r.type}</td>
                <td class="strong">{r.name}</td>
                <td>{#if r.public}{#each r.exposure as ex}<span class="pill exposed">{ex}</span> {/each}{:else}<span class="pill internal">internal</span>{/if}</td>
                <td>{#if r.dynamic}<span class="pill dyn">dynamic</span>{/if}</td>
              </tr>
            {/each}
          </tbody>
        </table></div>
      {:else}<p class="empty">No resources parsed.</p>{/if}
    </div>
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 20px; }
  .lede { max-width: 820px; margin: 0 0 14px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .mono { font-family: var(--mono, ui-monospace, monospace); }
  .input { width: 100%; min-height: 150px; resize: vertical; background: var(--well); border: 1px solid var(--field-border); border-radius: 7px; padding: 10px 12px; color: var(--text); font-size: 0.78rem; outline: none; line-height: 1.45; }
  .input:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .row { display: flex; gap: 8px; margin-top: 10px; align-items: center; flex-wrap: wrap; }
  .srcrow { margin-top: 0; margin-bottom: 10px; }
  .k0 { font-size: 0.72rem; text-transform: uppercase; letter-spacing: .05em; color: var(--text-faint); }
  .srcsel { background: var(--well); border: 1px solid var(--field-border); color: var(--text); border-radius: 6px; padding: 6px 10px; font: inherit; font-size: 0.82rem; }
  .srcsel:focus { outline: none; border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .hint { font-size: 0.74rem; color: var(--text-faint); }
  .go { background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px; padding: 8px 16px; font: inherit; font-size: 0.9rem; cursor: pointer; }
  .go:disabled { opacity: .6; cursor: default; }
  .ghost { background: transparent; border: 1px solid var(--border-soft); color: var(--text-dim); border-radius: 7px; padding: 7px 13px; font: inherit; font-size: 0.82rem; cursor: pointer; }
  .ghost:hover:not(:disabled) { color: var(--text); border-color: var(--field-border); }
  .ghost:disabled { opacity: .5; cursor: default; }

  .tiles { display: grid; grid-template-columns: repeat(5,1fr); gap: 10px; margin: 14px 0; max-width: 820px; }
  @media (max-width: 720px){ .tiles{ grid-template-columns: repeat(2,1fr);} }
  .tile { background: var(--card); border: 1px solid var(--border-soft); border-radius: 10px; padding: 12px 13px; }
  .tile .n { font-size: 1.5rem; font-weight: 650; font-family: var(--mono, monospace); color: var(--text-strong); font-variant-numeric: tabular-nums; }
  .tile .n.bad { color: var(--bad, #ff6b6b); }
  .tile .l { font-size: 0.66rem; text-transform: uppercase; letter-spacing: .04em; color: var(--text-faint); margin-top: 3px; }

  .banner { margin: 0 0 14px; padding: 11px 14px; border-radius: 9px; border: 1px solid var(--border-soft); display: flex; flex-direction: column; gap: 3px; max-width: 820px; }
  .banner.warn { border-color: var(--warn, #d9a441); background: color-mix(in srgb, var(--warn, #d9a441) 12%, transparent); }
  .banner.bad { border-color: var(--bad, #ff6b6b); background: color-mix(in srgb, var(--bad, #ff6b6b) 10%, transparent); }
  .vlabel { font-weight: 600; color: var(--bad, #ff6b6b); }
  .vnote { font-size: 0.78rem; color: var(--text-dim); overflow-wrap: anywhere; }

  .card { background: var(--card); border: 1px solid var(--border-soft); border-radius: 11px; padding: 13px 15px; max-width: 820px; margin-bottom: 14px; }
  .k { font-size: 0.74rem; text-transform: uppercase; letter-spacing: .04em; color: var(--text-faint); margin-bottom: 10px; }
  .k .count { color: var(--text-dim); }
  .empty { font-size: 0.82rem; color: var(--text-dim); margin: 2px 0; }

  .diagram { background: var(--well); border: 1px solid var(--border-soft); border-radius: 10px; padding: 6px; overflow-x: auto; }
  .edge { stroke-width: 1.7; fill: none; }
  .edge.crit { stroke: var(--bad, #ff6b6b); marker-end: url(#tf-crit); }
  .edge.warn { stroke: var(--warn, #e0b341); marker-end: url(#tf-warn); }
  .edge.info { stroke: var(--text-dim, #8c99a8); marker-end: url(#tf-info); }
  .elabel { fill: var(--text-dim); font-size: 10px; font-family: var(--mono, monospace); }
  .inet rect { fill: color-mix(in srgb, var(--bad,#ff6b6b) 16%, var(--card)); stroke: color-mix(in srgb, var(--bad,#ff6b6b) 45%, transparent); }
  .inet .it { fill: var(--text-strong); font-size: 11.5px; font-weight: 600; }
  .inet .is { fill: var(--text-faint); font-size: 9.5px; font-family: var(--mono, monospace); }
  .res rect { fill: var(--card); stroke: var(--border-soft); }
  .res .rt { fill: var(--text); font-size: 11px; font-family: var(--mono, monospace); }
  .legend { display: flex; gap: 16px; flex-wrap: wrap; font-size: 0.72rem; color: var(--text-faint); margin-top: 9px; }
  .legend span::before { content:""; display:inline-block; width:14px; height:0; border-top:2px solid; margin-right:6px; vertical-align: middle; }
  .lg-crit::before { border-color: var(--bad,#ff6b6b); } .lg-warn::before { border-color: var(--warn,#e0b341); } .lg-info::before { border-color: var(--text-dim,#8c99a8); }

  .finding { display: grid; grid-template-columns: 74px 1fr; gap: 12px; padding: 10px 0; border-top: 1px solid var(--border-soft); cursor: pointer; }
  .finding:first-of-type { border-top: none; }
  .finding:hover { background: color-mix(in srgb, var(--accent) 5%, transparent); }
  .sev { font-size: 0.58rem; font-weight: 700; text-transform: uppercase; letter-spacing: .06em; padding: 3px 0; height: fit-content; border-radius: 5px; text-align: center; }
  .sev.crit { color: var(--bad,#ff6b6b); background: color-mix(in srgb, var(--bad,#ff6b6b) 14%, transparent); }
  .sev.warn { color: var(--warn,#e0b341); background: color-mix(in srgb, var(--warn,#e0b341) 14%, transparent); }
  .sev.info { color: var(--text-dim); background: color-mix(in srgb, var(--text-dim) 14%, transparent); }
  .ftitle { color: var(--text-strong); font-weight: 600; font-size: 0.86rem; }
  .fdetail { color: var(--text-dim); font-size: 0.79rem; margin-top: 2px; }
  .fres { color: var(--accent); font-size: 0.74rem; margin-top: 2px; }

  .tbl { overflow-x: auto; }
  table { width: 100%; border-collapse: collapse; font-size: 0.8rem; }
  th { text-align: left; color: var(--text-faint); font-weight: 500; padding: 3px 10px 7px 0; border-bottom: 1px solid var(--border-soft); white-space: nowrap; }
  td { padding: 6px 10px 6px 0; border-bottom: 1px solid var(--border-soft); vertical-align: top; }
  tr:last-child td { border-bottom: none; }
  .strong { color: var(--text-strong); font-weight: 600; }
  .dim { color: var(--text-dim); }
  .pill { display: inline-block; font-size: 0.68rem; padding: 1px 7px; border-radius: 5px; font-family: var(--mono, monospace); }
  .pill.exposed { color: var(--bad,#ff6b6b); background: color-mix(in srgb, var(--bad,#ff6b6b) 13%, transparent); }
  .pill.internal { color: var(--text-dim); background: color-mix(in srgb, var(--text-dim) 12%, transparent); }
  .pill.dyn { color: var(--warn,#e0b341); background: color-mix(in srgb, var(--warn,#e0b341) 13%, transparent); }
</style>
