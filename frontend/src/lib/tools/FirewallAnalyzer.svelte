<script lang="ts">
  import { AnalyzeFirewall } from "../../../wailsjs/go/app/App.js";

  const VENDORS = [
    { id: "fortigate", label: "FortiGate" },
    { id: "panos", label: "Palo Alto (PAN-OS)" },
    { id: "asa", label: "Cisco ASA" },
    { id: "srx", label: "Juniper SRX" },
  ];
  const FORMAT_HINT: Record<string, string> = {
    fortigate: "a FortiGate config export",
    panos: "a PAN-OS config in set-format (set cli config-output-format set)",
    asa: "a Cisco ASA running-config",
    srx: "a Juniper SRX config in set-format (show configuration | display set)",
  };

  let configText = "";
  let vendor = "fortigate";
  let busy = false;
  let error = "";
  let analyses: any[] = [];
  let vIdx = 0;
  let openFlow = -1;
  let fileInput: HTMLInputElement;

  $: a = analyses.length ? analyses[vIdx] : null;
  // FortiGate segments carry a VLAN id; the other vendors' segments don't, so
  // the id column is hidden when every segment in view is untagged.
  $: showSegId = (a?.vlans ?? []).some((v: any) => v.id > 0);

  async function analyze() {
    const v = configText.trim();
    if (!v) {
      analyses = [];
      return;
    }
    error = "";
    analyses = [];
    openFlow = -1;
    busy = true;
    try {
      analyses = (await AnalyzeFirewall(v, vendor)) ?? [];
      vIdx = 0;
    } catch (e) {
      error = String(e);
    } finally {
      busy = false;
    }
  }

  function onVendorChange() {
    if (configText.trim()) analyze();
  }

  function loadFile(e: Event) {
    const f = (e.target as HTMLInputElement).files?.[0];
    if (!f) return;
    const r = new FileReader();
    r.onload = () => {
      configText = String(r.result ?? "");
    };
    r.readAsText(f);
  }

  function clearAll() {
    configText = "";
    analyses = [];
    error = "";
    openFlow = -1;
  }

  const policyById = (an: any, id: number) =>
    (an?.policies ?? []).find((p: any) => p.id === id);

  function toggleFlow(i: number) {
    openFlow = openFlow === i ? -1 : i;
  }

  // Diagram: lay the flow nodes out on an ellipse; edges are chords.
  $: diagram = buildDiagram(a);
  function buildDiagram(an: any) {
    const flows = (an?.flows ?? []).filter((f: any) => f.src !== f.dst);
    if (!flows.length) return null;
    const names: string[] = [...new Set(flows.flatMap((f: any) => [f.src, f.dst]))];
    if (names.length > 12) return { tooMany: true, count: names.length };
    const cx = 410, cy = 180, rx = 320, ry = 140;
    const pos: Record<string, any> = {};
    names.forEach((name, i) => {
      const ang = -Math.PI / 2 + (2 * Math.PI * i) / names.length;
      pos[name] = { name, x: cx + rx * Math.cos(ang), y: cy + ry * Math.sin(ang) };
    });
    const edges = flows.map((f: any) => {
      const a0 = pos[f.src], b0 = pos[f.dst];
      const dx = b0.x - a0.x, dy = b0.y - a0.y, len = Math.hypot(dx, dy) || 1;
      const ux = dx / len, uy = dy / len, pad = 48;
      const x1 = a0.x + ux * pad, y1 = a0.y + uy * pad;
      const x2 = b0.x - ux * pad, y2 = b0.y - uy * pad;
      const label = (f.services ?? []).slice(0, 2).join(", ") || (f.action === "deny" ? "deny" : "any");
      return { x1, y1, x2, y2, mx: (x1 + x2) / 2, my: (y1 + y2) / 2, action: f.action, label };
    });
    return { nodes: Object.values(pos), edges, tooMany: false };
  }
</script>

<div class="tool">
  <p class="lede">
    Drop or paste an exported firewall config. Meridian parses it locally — nothing
    leaves your machine — and shows the network segments, what talks to what, and the
    counts. It's a reading of the ruleset at the segment level, not a live device.
  </p>

  <div class="card span">
    <div class="row vendorrow">
      <span class="k0">Vendor</span>
      <select class="vendorsel" bind:value={vendor} on:change={onVendorChange}>
        {#each VENDORS as vn}
          <option value={vn.id}>{vn.label}</option>
        {/each}
      </select>
      <span class="vnote">Expecting {FORMAT_HINT[vendor]}.</span>
    </div>
    <textarea
      class="mono input"
      bind:value={configText}
      placeholder="Paste {FORMAT_HINT[vendor]} here, or use Load file…"
      spellcheck="false"
    ></textarea>
    <div class="row">
      <button class="go" on:click={analyze} disabled={busy || !configText.trim()}>
        {busy ? "Analyzing…" : "Analyze"}
      </button>
      <button class="ghost" on:click={() => fileInput.click()}>Load file…</button>
      <button class="ghost" on:click={clearAll} disabled={!configText && !analyses.length}>Clear</button>
      <input type="file" bind:this={fileInput} on:change={loadFile} accept=".conf,.txt,.cfg,text/plain" hidden />
    </div>
  </div>

  {#if error}
    <div class="banner bad"><span class="vlabel">Couldn't parse it</span><span class="vnote">{error}</span></div>
  {/if}

  {#if a}
    {#if analyses.length > 1}
      <div class="vdombar">
        <span class="k0">Context</span>
        <select bind:value={vIdx} on:change={() => (openFlow = -1)}>
          {#each analyses as an, i}
            <option value={i}>{an.vdom || "(root)"}</option>
          {/each}
        </select>
        <span class="vnote">{analyses.length} contexts in this config</span>
      </div>
    {/if}

    <div class="tiles">
      <div class="tile"><div class="n">{a.counts.vlans}</div><div class="l">Segments</div></div>
      <div class="tile"><div class="n">{a.counts.policies}</div><div class="l">Policies</div></div>
      <div class="tile"><div class="n">{a.counts.addresses}</div><div class="l">Addr objects</div></div>
      <div class="tile"><div class="n">{a.counts.subnetsIps}</div><div class="l">Subnets / IPs</div></div>
      <div class="tile"><div class="n">{a.counts.services}</div><div class="l">Services</div></div>
      <div class="tile"><div class="n">{a.counts.zones}</div><div class="l">Zones</div></div>
    </div>

    {#if a.notes && a.notes.length}
      <div class="banner warn">
        <span class="vlabel">{a.notes.length} policies not fully analysed</span>
        {#each a.notes as note}<span class="vnote">{note}</span>{/each}
      </div>
    {/if}

    <div class="card span">
      <div class="k">Segments <span class="count">· {(a.vlans ?? []).length}</span></div>
      {#if (a.vlans ?? []).length}
        <table class="mono"><thead>
          <tr>{#if showSegId}<th>VLAN</th>{/if}<th>Name</th><th>Interface</th><th>Subnet</th><th>Zone</th></tr>
        </thead><tbody>
          {#each a.vlans as v}
            <tr>{#if showSegId}<td>{v.id || "—"}</td>{/if}<td class="strong">{v.name}</td><td class="dim">{v.interface || "—"}</td><td class="dim">{v.subnet || "—"}</td><td class="dim">{v.zone || "—"}</td></tr>
          {/each}
        </tbody></table>
      {:else}<p class="empty">No network segments found in this config.</p>{/if}
    </div>

    <div class="card span">
      <div class="k">What talks to what <span class="count">· {(a.flows ?? []).length} flows</span></div>
      {#if (a.flows ?? []).length}
        <table class="mono flows"><thead>
          <tr><th>Source</th><th>Destination</th><th>Service</th><th>Action</th></tr>
        </thead><tbody>
          {#each a.flows as f, i}
            <tr class="flowrow" class:open={openFlow === i} on:click={() => toggleFlow(i)}>
              <td class="strong">{f.src}</td>
              <td class="strong">{f.dst}</td>
              <td>{#each f.services ?? [] as s}<span class="pill">{s}</span> {/each}{#if !(f.services ?? []).length}<span class="dim">any</span>{/if}</td>
              <td class:allow={f.action === "allow"} class:deny={f.action === "deny"}>{f.action}</td>
            </tr>
            {#if openFlow === i}
              <tr class="drill"><td colspan="4">
                <div class="k0">Policies behind this flow</div>
                {#each f.policyIds ?? [] as pid}
                  {@const p = policyById(a, pid)}
                  {#if p}
                    <div class="pol">
                      <span class="polid">#{p.id}</span>
                      <span class="polname">{p.name || "(unnamed)"}</span>
                      <span class="dim">{(p.srcAddr ?? []).join(", ") || "all"} → {(p.dstAddr ?? []).join(", ") || "all"}</span>
                      <span class="dim">{(p.services ?? []).join(", ") || "any"}</span>
                    </div>
                  {/if}
                {/each}
              </td></tr>
            {/if}
          {/each}
        </tbody></table>
        <div class="note">Click a flow to see the exact policies, address objects and IPs behind it.</div>
      {:else}<p class="empty">No permitted or denied flows derived (no enabled policies, or all use negation).</p>{/if}
    </div>

    {#if diagram}
      <div class="card span">
        <div class="k">Flow diagram</div>
        {#if diagram.tooMany}
          <p class="empty">{diagram.count} nodes — too many to diagram clearly; use the flow table above.</p>
        {:else}
          <div class="diagram">
            <svg viewBox="0 0 820 360" width="100%" role="img" aria-label="VLAN flow diagram">
              <defs>
                <marker id="fw-arrow-allow" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse">
                  <path d="M0,0 L10,5 L0,10 z" fill="var(--good, #5ad17a)"/>
                </marker>
                <marker id="fw-arrow-deny" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse">
                  <path d="M0,0 L10,5 L0,10 z" fill="var(--bad, #ff6b6b)"/>
                </marker>
              </defs>
              {#each diagram.edges as e}
                <line x1={e.x1} y1={e.y1} x2={e.x2} y2={e.y2}
                  class="edge" class:allow={e.action === "allow"} class:deny={e.action === "deny"} />
                <text class="elabel" x={e.mx} y={e.my - 3} text-anchor="middle">{e.label}</text>
              {/each}
              {#each diagram.nodes as n}
                <g class="dnode">
                  <rect x={n.x - 44} y={n.y - 17} width="88" height="34" rx="7" />
                  <text x={n.x} y={n.y + 4} text-anchor="middle">{n.name}</text>
                </g>
              {/each}
            </svg>
          </div>
          <div class="note">Green = permitted, red dashed = explicit deny. Arrow points at the destination; labels name the service.</div>
        {/if}
      </div>
    {/if}
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 20px; }
  .lede { max-width: 820px; margin: 0 0 14px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .mono { font-family: var(--mono, ui-monospace, monospace); }
  .input { width: 100%; min-height: 150px; resize: vertical; background: var(--well); border: 1px solid var(--field-border); border-radius: 7px; padding: 10px 12px; color: var(--text); font-size: 0.8rem; outline: none; line-height: 1.45; }
  .input:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .row { display: flex; gap: 8px; margin-top: 10px; align-items: center; }
  .vendorrow { margin-top: 0; margin-bottom: 10px; flex-wrap: wrap; }
  .vendorsel { background: var(--well); border: 1px solid var(--field-border); color: var(--text); border-radius: 6px; padding: 6px 10px; font: inherit; font-size: 0.82rem; }
  .vendorsel:focus { outline: none; border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .go { background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px; padding: 8px 16px; font: inherit; font-size: 0.9rem; cursor: pointer; }
  .go:disabled { opacity: .6; cursor: default; }
  .ghost { background: transparent; border: 1px solid var(--border-soft); color: var(--text-dim); border-radius: 7px; padding: 7px 13px; font: inherit; font-size: 0.82rem; cursor: pointer; }
  .ghost:hover:not(:disabled) { color: var(--text); border-color: var(--field-border); }
  .ghost:disabled { opacity: .5; cursor: default; }

  .vdombar { display: flex; align-items: center; gap: 10px; margin: 14px 0 4px; font-size: 0.82rem; }
  .vdombar select { background: var(--well); border: 1px solid var(--field-border); color: var(--text); border-radius: 6px; padding: 5px 9px; font: inherit; font-size: 0.82rem; }
  .k0 { font-size: 0.72rem; text-transform: uppercase; letter-spacing: .05em; color: var(--text-faint); }

  .tiles { display: grid; grid-template-columns: repeat(6,1fr); gap: 10px; margin: 14px 0; max-width: 820px; }
  @media (max-width: 720px){ .tiles{ grid-template-columns: repeat(3,1fr);} }
  .tile { background: var(--card); border: 1px solid var(--border-soft); border-radius: 10px; padding: 12px 13px; }
  .tile .n { font-size: 1.5rem; font-weight: 650; font-family: var(--mono, monospace); color: var(--text-strong); }
  .tile .l { font-size: 0.68rem; text-transform: uppercase; letter-spacing: .04em; color: var(--text-faint); margin-top: 3px; }

  .banner { margin: 14px 0; padding: 11px 14px; border-radius: 9px; border: 1px solid var(--border-soft); display: flex; flex-direction: column; gap: 3px; max-width: 820px; }
  .banner.warn { border-color: var(--warn, #d9a441); background: color-mix(in srgb, var(--warn, #d9a441) 12%, transparent); }
  .banner.bad { border-color: var(--bad, #ff6b6b); background: color-mix(in srgb, var(--bad, #ff6b6b) 10%, transparent); }
  .vlabel { font-weight: 600; color: var(--text-strong); }
  .banner.warn .vlabel { color: var(--warn, #d9a441); }
  .banner.bad .vlabel { color: var(--bad, #ff6b6b); }
  .vnote { font-size: 0.78rem; color: var(--text-dim); overflow-wrap: anywhere; }

  .card { background: var(--card); border: 1px solid var(--border-soft); border-radius: 11px; padding: 13px 15px; max-width: 820px; margin-bottom: 14px; }
  .k { font-size: 0.74rem; text-transform: uppercase; letter-spacing: .04em; color: var(--text-faint); margin-bottom: 10px; }
  .k .count { color: var(--text-dim); }
  .empty { font-size: 0.82rem; color: var(--text-dim); margin: 2px 0; }

  table { width: 100%; border-collapse: collapse; font-size: 0.8rem; }
  th { text-align: left; color: var(--text-faint); font-weight: 500; padding: 3px 10px 7px 0; border-bottom: 1px solid var(--border-soft); white-space: nowrap; }
  td { padding: 6px 10px 6px 0; border-bottom: 1px solid var(--border-soft); vertical-align: top; overflow-wrap: anywhere; }
  tr:last-child td { border-bottom: none; }
  .strong { color: var(--text-strong); font-weight: 600; }
  .dim { color: var(--text-dim); }
  .pill { display: inline-block; font-family: var(--mono, monospace); font-size: 0.74rem; padding: 1px 7px; border-radius: 5px; background: var(--well); color: var(--text-dim); border: 1px solid var(--border-soft); }
  .allow { color: var(--good, #5ad17a); }
  .deny { color: var(--bad, #ff6b6b); }

  .flowrow { cursor: pointer; }
  .flowrow:hover td { background: color-mix(in srgb, var(--accent) 7%, transparent); }
  .flowrow.open td { background: color-mix(in srgb, var(--accent) 10%, transparent); }
  .drill td { background: var(--well); }
  .pol { display: flex; flex-wrap: wrap; gap: 10px; font-size: 0.78rem; padding: 3px 0; }
  .polid { color: var(--text-faint); font-family: var(--mono, monospace); }
  .polname { color: var(--text); font-weight: 600; }
  .note { font-size: 0.74rem; color: var(--text-faint); margin-top: 9px; }

  .diagram { background: var(--well); border: 1px solid var(--border-soft); border-radius: 10px; padding: 6px; }
  .edge { stroke-width: 1.6; fill: none; opacity: .85; }
  .edge.allow { stroke: var(--good, #5ad17a); marker-end: url(#fw-arrow-allow); }
  .edge.deny { stroke: var(--bad, #ff6b6b); stroke-dasharray: 4 3; marker-end: url(#fw-arrow-deny); }
  .elabel { fill: var(--text-dim); font-size: 9.5px; font-family: var(--mono, monospace); }
  .dnode rect { fill: var(--card); stroke: var(--border-soft); }
  .dnode text { fill: var(--text); font-size: 11px; font-family: var(--mono, monospace); }
</style>
