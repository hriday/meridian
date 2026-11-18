<script lang="ts">
  import { AnalyzeFCZoning } from "../../../wailsjs/go/app/App.js";
  import ToolHelp from "../ToolHelp.svelte";

  let configText = "";
  let rep: any = null;
  let busy = false;
  let error = "";

  async function analyze() {
    const v = configText.trim();
    if (!v) return;
    busy = true;
    error = "";
    rep = null;
    try {
      rep = await AnalyzeFCZoning(v);
    } catch (e: any) {
      error = String(e?.message ?? e);
    } finally {
      busy = false;
    }
  }

  const sevClass = (s: string) => (s === "warning" ? "warn" : "info");

  // A bipartite zone <-> member map. It makes over-shared members (a WWN in
  // many zones) and fat zones (not single-initiator-single-target) visible at a
  // glance. It is NOT a traffic/flow map: we can't tell an initiator from a
  // target by its WWN, so there's no direction. Past a threshold it's a
  // hairball, so we fall back to "too large — use the findings".
  const ROW = 26,
    PADY = 18,
    LX = 165,
    RX = 585,
    W = 760;

  $: diagram = rep ? buildDiagram(rep) : null;

  function buildDiagram(r: any) {
    const zones = (Array.isArray(r.zones) ? r.zones : []).filter((z: any) => !(z.name || "").includes("$default_zone$"));
    const memMap = new Map<string, any>();
    for (const z of zones) {
      for (const m of Array.isArray(z.members) ? z.members : []) {
        if (!memMap.has(m.value)) memMap.set(m.value, { value: m.value, kind: m.kind, zones: [] });
        memMap.get(m.value).zones.push(z.name);
      }
    }
    const members = [...memMap.values()];
    if (zones.length === 0) return null;
    if (zones.length > 30 || members.length > 60) {
      return { tooLarge: true, nz: zones.length, nm: members.length };
    }
    const mi = new Map(members.map((m, i) => [m.value, i]));
    const zNodes = zones.map((z: any, i: number) => ({
      name: z.name,
      y: PADY + i * ROW + ROW / 2,
      count: (z.members || []).length,
      fat: (z.members || []).length > 2,
    }));
    const mNodes = members.map((m, i) => ({ ...m, y: PADY + i * ROW + ROW / 2, shared: m.zones.length > 1 }));
    const edges: any[] = [];
    zones.forEach((z: any, zi: number) => {
      for (const m of z.members || []) {
        const idx = mi.get(m.value)!;
        edges.push({ y1: zNodes[zi].y, y2: mNodes[idx].y, shared: mNodes[idx].shared });
      }
    });
    const height = Math.max(zones.length, members.length) * ROW + PADY * 2;
    return { tooLarge: false, zNodes, mNodes, edges, height };
  }

  const trunc = (s: string, n: number) => (s.length > n ? s.slice(0, n - 1) + "…" : s);
  const memLabel = (m: any) =>
    m.kind === "wwn" || m.kind === "pwwn" ? "…" + String(m.value).slice(-8) : trunc(String(m.value), 16);
</script>

<div class="tool">
  <p class="lede">
    Analyze <strong>Fibre Channel switch zoning</strong> for SAN hygiene. Paste <strong>Cisco MDS</strong>
    <span class="mono">show zoneset [active]</span> / running-config, or <strong>Brocade</strong>
    <span class="mono">cfgshow</span> — the vendor is auto-detected. Offline; the config never leaves your machine.
  </p>

  <ToolHelp>
    It maps your <strong>zonesets, zones and aliases</strong> and flags the usual problems: zones that aren't
    <strong>single-initiator-single-target</strong> (the big best practice), empty or single-member zones, zones not in
    any zoneset or not in the active set, and unused aliases. It also reports how many lines it <em>didn't</em> recognize,
    so a coverage gap is visible rather than silently dropped. It reads config you already have — it doesn't talk to a
    live fabric.
  </ToolHelp>

  <textarea
    class="cfg mono"
    bind:value={configText}
    placeholder="Paste `show zoneset active` (Cisco MDS) or `cfgshow` (Brocade) output here…"
    spellcheck="false"
  ></textarea>
  <div class="row">
    <button class="btn" on:click={analyze} disabled={busy || !configText.trim()}>{busy ? "Analyzing…" : "Analyze"}</button>
  </div>

  {#if error}<p class="err">{error}</p>{/if}

  {#if rep}
    <div class="head">
      <span class="vendor">{rep.vendor}</span>
      <span class="counts">{rep.counts.zonesets} zoneset(s) · {rep.counts.zones} zone(s) · {rep.counts.aliases} alias(es)</span>
    </div>

    {#if rep.zonesets && rep.zonesets.length}
      <div class="sets">
        {#each rep.zonesets as zs}
          <span class="set" class:active={zs.active}>{zs.name} · {zs.zones?.length ?? 0} zones{#if zs.active} <span class="ab">active</span>{/if}</span>
        {/each}
      </div>
    {/if}

    {#if diagram}
      <div class="mapwrap">
        <div class="mk">Zone map</div>
        {#if diagram.tooLarge}
          <p class="foot">Too large to diagram clearly — {diagram.nz} zones, {diagram.nm} members. At this scale the findings above are the better view.</p>
        {:else}
          <svg class="map" viewBox={`0 0 ${W} ${diagram.height}`} role="img" aria-label="Zone membership map">
            <text x={LX} y="11" text-anchor="middle" class="colh">zones</text>
            <text x={RX} y="11" text-anchor="middle" class="colh">members</text>
            {#each diagram.edges as e}
              <line x1={LX} y1={e.y1} x2={RX} y2={e.y2} class="edge" class:shared={e.shared} />
            {/each}
            {#each diagram.zNodes as z}
              <circle cx={LX} cy={z.y} r="4" class="node z" class:fat={z.fat} />
              <text x={LX - 10} y={z.y + 3} text-anchor="end" class="zlabel" class:fat={z.fat}>
                <title>{z.name} — {z.count} members{z.fat ? " (not single-initiator-single-target)" : ""}</title>
                {trunc(z.name, 20)}{#if z.fat} ({z.count}){/if}
              </text>
            {/each}
            {#each diagram.mNodes as m}
              <circle cx={RX} cy={m.y} r="4" class="node m" class:shared={m.shared} />
              <text x={RX + 10} y={m.y + 3} class="mlabel" class:shared={m.shared}>
                <title>{m.value}{m.shared ? ` — in ${m.zones.length} zones` : ""}</title>
                {memLabel(m)}
              </text>
            {/each}
          </svg>
          <p class="legend">
            <span class="lg fat">● fat zone (not single-initiator-single-target)</span>
            <span class="lg shared">● member in multiple zones</span>
          </p>
        {/if}
      </div>
    {/if}

    {#if rep.findings && rep.findings.length}
      <div class="findings">
        {#each rep.findings as f}
          <div class="finding {sevClass(f.severity)}">
            <div class="fline"><span class="sev {sevClass(f.severity)}">{f.severity}</span><span class="ftitle">{f.title}</span>{#if f.object}<span class="fobj mono">{f.object}</span>{/if}</div>
            <div class="fdetail">{f.detail}</div>
          </div>
        {/each}
      </div>
    {:else}
      <p class="good">No hygiene findings — zones look clean.</p>
    {/if}

    {#if rep.counts.unparsed > 0}
      <p class="foot">⚠ {rep.counts.unparsed} line(s) weren't recognized, so the parse may be incomplete. If this is a real export, share a sample and I can widen coverage.</p>
    {/if}
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 30px; }
  .lede { max-width: 820px; margin: 0 0 12px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .mono { font-family: var(--font-mono, ui-monospace, monospace); }
  .cfg { width: 100%; max-width: 820px; height: 190px; resize: vertical; background: var(--well); border: 1px solid var(--field-border, var(--border)); border-radius: 10px; padding: 11px 13px; color: var(--text); font-size: 0.8rem; line-height: 1.5; outline: none; }
  .cfg:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .row { margin: 10px 0 0; max-width: 820px; }
  .btn { background: var(--accent); color: var(--on-accent, #052e2b); border: 1px solid transparent; border-radius: 8px; padding: 9px 18px; font: inherit; font-size: 0.86rem; font-weight: 550; cursor: pointer; }
  .btn:disabled { opacity: .5; cursor: default; }
  .err { color: var(--bad, #ff6b6b); font-size: 0.84rem; max-width: 820px; }
  .head { display: flex; align-items: baseline; gap: 12px; max-width: 820px; margin-top: 16px; }
  .vendor { font-weight: 650; color: var(--text-strong, var(--text)); }
  .counts { font-size: 0.8rem; color: var(--text-dim); }
  .sets { display: flex; flex-wrap: wrap; gap: 7px; max-width: 820px; margin-top: 8px; }
  .set { font-size: 0.76rem; background: var(--well); border: 1px solid var(--border); border-radius: 20px; padding: 3px 11px; color: var(--text-dim); }
  .set.active { border-color: var(--accent); color: var(--text); }
  .ab { font-size: 0.64rem; text-transform: uppercase; letter-spacing: .04em; color: var(--on-accent, #052e2b); background: var(--accent); border-radius: 10px; padding: 0 6px; margin-left: 4px; }
  .mapwrap { max-width: 820px; margin-top: 16px; border: 1px solid var(--border); border-radius: 11px; padding: 10px 14px 14px; background: var(--card); }
  .mk { font-size: 0.7rem; text-transform: uppercase; letter-spacing: .05em; color: var(--text-dim); font-weight: 600; margin-bottom: 6px; }
  .map { width: 100%; height: auto; display: block; }
  .map .colh { font-size: 9px; fill: var(--text-dim); text-transform: uppercase; letter-spacing: .05em; }
  .map .edge { stroke: var(--border); stroke-width: 1; opacity: .7; }
  .map .edge.shared { stroke: var(--accent); opacity: .9; }
  .map .node { fill: var(--text-dim); }
  .map .node.z { fill: var(--accent); }
  .map .node.z.fat { fill: var(--warn, #f5a623); }
  .map .node.m.shared { fill: var(--accent); }
  .map .zlabel, .map .mlabel { font-family: var(--font-mono, ui-monospace, monospace); font-size: 10px; fill: var(--text); }
  .map .zlabel.fat { fill: var(--warn, #f5a623); font-weight: 600; }
  .map .mlabel.shared { fill: var(--accent); font-weight: 600; }
  .legend { display: flex; flex-wrap: wrap; gap: 14px; margin: 8px 0 0; font-size: 0.7rem; color: var(--text-dim); }
  .legend .lg.fat { color: var(--warn, #f5a623); }
  .legend .lg.shared { color: var(--accent); }
  .findings { max-width: 820px; margin-top: 14px; display: flex; flex-direction: column; gap: 8px; }
  .finding { border: 1px solid var(--border); border-left: 3px solid var(--warn, #f5a623); border-radius: 9px; padding: 9px 13px; background: var(--card); }
  .finding.info { border-left-color: var(--text-dim); }
  .fline { display: flex; align-items: baseline; gap: 10px; flex-wrap: wrap; }
  .sev { font-size: 0.64rem; text-transform: uppercase; letter-spacing: .05em; font-weight: 700; }
  .sev.warn { color: var(--warn, #f5a623); }
  .sev.info { color: var(--text-dim); }
  .ftitle { font-weight: 600; color: var(--text); font-size: 0.84rem; }
  .fobj { margin-left: auto; font-size: 0.74rem; color: var(--text-faint, var(--text-dim)); }
  .fdetail { font-size: 0.79rem; color: var(--text-dim); margin-top: 3px; line-height: 1.5; }
  .good { max-width: 820px; margin-top: 14px; font-size: 0.84rem; color: var(--good, #3fb950); }
  .foot { max-width: 820px; font-size: 0.74rem; color: var(--text-faint, var(--text-dim)); margin: 12px 0 0; line-height: 1.5; }
</style>
