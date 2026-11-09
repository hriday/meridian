<script lang="ts">
  import { get } from "svelte/store";
  import { LookupBGP } from "../../../wailsjs/go/app/App.js";
  import { requestedTool } from "../nav";

  let query = "1.1.1.1";
  let busy = false;
  let error = "";
  let report: any = null;
  let vfilter = "";
  let showAllUp = false;
  let showAllTransit = false;

  const CAP = 24; // how many upstream/transit chips to show before "+N more"

  async function run(value?: string) {
    const v = (value ?? query).trim();
    query = v;
    if (!v) {
      report = null;
      return;
    }
    error = "";
    busy = true;
    showAllUp = false;
    showAllTransit = false;
    vfilter = "";
    try {
      report = await LookupBGP(v);
    } catch (e) {
      report = null;
      error = String(e);
    } finally {
      busy = false;
    }
  }

  // Pivot an AS or prefix into the Lookup tool's entity page.
  function pivotTo(value: string) {
    requestedTool.set({ tool: "lookup", value });
  }

  function rpkiClass(state: string): string {
    const s = (state || "").toLowerCase();
    if (s === "valid") return "good";
    if (s === "invalid") return "bad";
    return "soon";
  }

  // Plain-language verdict for the top of the screen.
  function verdict(r: any): string {
    if (!r || r.collectors === 0)
      return "Not seen by any route collector — this exact prefix isn’t in the public routing table. Try an IP (it resolves to its routed /24) or a more-specific prefix.";
    const origins = (r.originAsns || []).map((a: string) => "AS" + a).join(", ");
    const reach =
      r.peers >= r.collectors * 10 ? "widely visible" : r.peers >= r.collectors ? "visible" : "sparsely seen";
    const moas = r.moas ? " ⚠ Multiple origins announce it (MOAS) — possible hijack or anycast." : "";
    return `Announced by ${origins}, seen by ${r.peers} peers across ${r.collectors} collectors — ${reach}.${moas}`;
  }

  $: vantages = report
    ? (report.vantages || []).filter((v: any) => {
        const q = vfilter.toLowerCase();
        return !q || v.rrc.toLowerCase().includes(q) || (v.location || "").toLowerCase().includes(q);
      })
    : [];

  $: upstreams = report ? report.upstreams || [] : [];
  $: transit = report ? report.transit || [] : [];

  // Pick up a pivot into this tool, then resolve on mount.
  const req = get(requestedTool);
  if (req && req.tool === "bgp" && req.value) {
    query = req.value;
    requestedTool.set(null);
  }
  run();
</script>

<div class="tool">
  <form on:submit|preventDefault={() => run()} class="row">
    <input
      class="field mono"
      bind:value={query}
      placeholder="1.1.1.1  ·  104.16.0.0/13  ·  8.8.8.0/24"
      autocomplete="off"
      spellcheck="false"
      aria-label="prefix or IP"
    />
    <button class="go" disabled={busy}>{busy ? "Querying…" : "Look up"}</button>
  </form>

  {#if error}<div class="hint">{error}</div>{/if}

  {#if report}
    <!-- Verdict: what this all means, in plain words. -->
    <div class="verdict" class:bad={report.collectors === 0 || report.moas}>{verdict(report)}</div>

    <div class="cards">
      <!-- Origin & health -->
      <div class="card hero">
        <div class="k">Origin &amp; RPKI</div>
        {#if report.prefix}
          <div class="line">
            <button class="chip" on:click={() => pivotTo(report.prefix)} title="Look up this prefix">{report.prefix}</button>
          </div>
        {/if}
        <div class="line">
          {#each report.originAsns || [] as asn}
            <button class="chip" on:click={() => pivotTo("AS" + asn)} title="Look up AS{asn}">AS{asn}</button>
          {/each}
          {#if report.rpkiState}
            <span class="pill {rpkiClass(report.rpkiState)}">RPKI {report.rpkiState}</span>
          {:else}
            <span class="pill soon">RPKI unknown</span>
          {/if}
        </div>
        <div class="cap">
          <b>{report.peers}</b> peers · <b>{report.collectors}</b> collectors · <b>{(report.paths || []).length}</b> distinct AS paths.
          A peer is one router's view; more peers &amp; collectors = more independent confirmation the route is real.
          <span class="pill-exp">RPKI “valid” means the origin is cryptographically authorized to announce this prefix.</span>
        </div>
      </div>

      <!-- Upstreams & transit -->
      <div class="card">
        <div class="k">Who carries it</div>
        <div class="sub">Upstreams — the AS directly in front of the origin</div>
        <div class="line wrap">
          {#each (showAllUp ? upstreams : upstreams.slice(0, CAP)) as asn}
            <button class="chip sm" on:click={() => pivotTo("AS" + asn)}>AS{asn}</button>
          {/each}
          {#if upstreams.length === 0}<span class="dim">none</span>{/if}
          {#if upstreams.length > CAP}
            <button class="more" on:click={() => (showAllUp = !showAllUp)}>
              {showAllUp ? "show fewer" : `+${upstreams.length - CAP} more`}
            </button>
          {/if}
        </div>
        <div class="sub">Transit — ASes further along the paths</div>
        <div class="line wrap">
          {#each (showAllTransit ? transit : transit.slice(0, CAP)) as asn}
            <button class="chip sm" on:click={() => pivotTo("AS" + asn)}>AS{asn}</button>
          {/each}
          {#if transit.length === 0}<span class="dim">none</span>{/if}
          {#if transit.length > CAP}
            <button class="more" on:click={() => (showAllTransit = !showAllTransit)}>
              {showAllTransit ? "show fewer" : `+${transit.length - CAP} more`}
            </button>
          {/if}
        </div>
      </div>

      <!-- AS paths -->
      <div class="card span">
        <div class="k">AS paths to the origin</div>
        <div class="cap">The hops BGP sees, ranked by how many peers observe each. Each is one route from a vantage point to the origin.</div>
        {#if (report.paths || []).length === 0}
          <div class="dim">No paths — not seen in any collector.</div>
        {:else}
          <table class="paths">
            <tbody>
              {#each report.paths.slice(0, 12) as p}
                <tr>
                  <td class="pct"><span class="bar" style="width:{Math.max(p.percent, 3)}%"></span><span class="pctn">{p.percent}%</span></td>
                  <td class="path">
                    {#each p.path as asn, i}
                      <button class="chip xs" on:click={() => pivotTo("AS" + asn)}>AS{asn}</button>{#if i < p.path.length - 1}<span class="arrow">→</span>{/if}
                    {/each}
                  </td>
                  <td class="peers">{p.peers} peers</td>
                </tr>
              {/each}
            </tbody>
          </table>
          {#if report.paths.length > 12}<div class="dim small">…and {report.paths.length - 12} more paths.</div>{/if}
        {/if}
      </div>

      <!-- Vantage picker -->
      <div class="card span">
        <div class="k">Collectors (vantage points)</div>
        <div class="cap">Each RIS route collector peers with many networks. Filter to find a vantage; peer count shows how many of its peers see this prefix.</div>
        <input class="field sm mono" bind:value={vfilter} placeholder="filter by RRC or location…" aria-label="filter collectors" />
        <div class="vlist">
          {#each vantages as v}
            <div class="vrow"><span class="rrc">{v.rrc}</span><span class="loc">{v.location}</span><span class="scope dim">{v.scope}</span><span class="vp">{v.peers} peers</span></div>
          {/each}
          {#if vantages.length === 0}<div class="dim">No collectors match.</div>{/if}
        </div>
      </div>
    </div>

    {#if report.routingSrc?.err}<div class="hint">looking-glass: {report.routingSrc.err}</div>{/if}
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 20px; }
  .row { display: flex; gap: 8px; max-width: 980px; }
  .field {
    flex: 1; background: var(--well); border: 1px solid var(--field-border); border-radius: 7px;
    padding: 9px 12px; color: var(--text); font-size: 0.9rem; outline: none;
  }
  .field.sm { padding: 6px 10px; font-size: 0.82rem; margin: 8px 0; max-width: 360px; }
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

  .verdict {
    margin: 14px 0 4px; padding: 11px 14px; border-radius: 9px; font-size: 0.9rem; line-height: 1.4;
    background: var(--good-bg, #102516); border: 1px solid var(--border-soft); color: var(--text);
  }
  .verdict.bad { background: var(--warn-bg); border-color: var(--warn-border); color: var(--warn-text); }

  .cards { display: grid; grid-template-columns: 1fr 1fr; gap: 13px; margin-top: 13px; }
  @media (max-width: 760px) { .cards { grid-template-columns: 1fr; } }
  .card { background: var(--card); border: 1px solid var(--border-soft); border-radius: 10px; padding: 13px 15px; }
  .card.span { grid-column: 1 / -1; }
  .card.hero { border-color: var(--accent); box-shadow: 0 0 0 1px var(--accent-ring); }
  .k { font-size: 0.72rem; text-transform: uppercase; letter-spacing: .04em; color: var(--text-dim); margin-bottom: 8px; }
  .sub { font-size: 0.74rem; color: var(--text-dim); margin: 8px 0 5px; }
  .line { display: flex; align-items: center; gap: 6px; margin: 5px 0; }
  .line.wrap { flex-wrap: wrap; }
  .cap { font-size: 0.73rem; color: var(--text-dim); line-height: 1.45; margin-top: 8px; }
  .pill-exp { display: block; margin-top: 4px; color: var(--text-faint); }
  .dim { color: var(--text-faint); }
  .small { font-size: 0.72rem; margin-top: 6px; }

  .chip {
    font-family: var(--font-mono); font-size: 0.78rem; background: var(--well); color: var(--accent);
    border: 1px solid var(--field-border); border-radius: 6px; padding: 2px 8px; cursor: pointer;
  }
  .chip:hover { border-color: var(--accent); }
  .chip.sm { font-size: 0.72rem; padding: 1px 6px; }
  .chip.xs { font-size: 0.7rem; padding: 0 5px; }
  .more {
    font-size: 0.72rem; background: none; border: none; color: var(--accent); cursor: pointer; padding: 1px 4px;
  }

  .pill { display: inline-flex; font-size: 0.66rem; padding: 1px 8px; border-radius: 20px; border: 1px solid var(--border); color: var(--text-dim); }
  .pill.soon { color: var(--text-faint); }
  .pill.good { color: var(--good, #5ad17a); border-color: transparent; background: var(--good-bg, #102516); }
  .pill.bad { color: var(--bad, #ff6b6b); border-color: transparent; }

  .paths { width: 100%; border-collapse: collapse; margin-top: 8px; }
  .paths td { padding: 4px 6px; vertical-align: middle; border-bottom: 1px solid var(--border-soft); }
  .pct { width: 64px; position: relative; }
  .bar { display: inline-block; height: 7px; background: var(--accent); border-radius: 4px; vertical-align: middle; opacity: .5; }
  .pctn { font-size: 0.68rem; color: var(--text-dim); margin-left: 5px; }
  .path { display: flex; flex-wrap: wrap; align-items: center; gap: 3px; }
  .arrow { color: var(--text-faint); margin: 0 1px; }
  .peers { white-space: nowrap; font-size: 0.72rem; color: var(--text-dim); text-align: right; width: 72px; }

  .vlist { margin-top: 4px; max-height: 260px; overflow-y: auto; }
  .vrow { display: grid; grid-template-columns: 70px 1fr auto auto; gap: 10px; align-items: center; padding: 4px 2px; border-bottom: 1px solid var(--border-soft); font-size: 0.8rem; }
  .rrc { font-family: var(--font-mono); color: var(--text); }
  .loc { color: var(--text); }
  .scope { font-size: 0.72rem; }
  .vp { font-size: 0.72rem; color: var(--text-dim); white-space: nowrap; }
</style>
