<script lang="ts">
  import { DNSPropagation } from "../../../wailsjs/go/app/App.js";

  let name = "";
  let rtype = "A";
  let loading = false;
  let error = "";
  let rep: any = null;

  const TYPES = ["A", "AAAA", "MX", "NS", "TXT", "CNAME", "SOA", "CAA"];

  async function run() {
    const n = name.trim();
    if (!n || loading) return;
    loading = true;
    error = "";
    rep = null;
    try {
      rep = await DNSPropagation(n, rtype);
    } catch (e: any) {
      error = String(e?.message ?? e);
    } finally {
      loading = false;
    }
  }

  // Majority group is 0; give each other group a distinct "differs" accent.
  function groupClass(g: number): string {
    if (g < 0) return "err";
    return g === 0 ? "maj" : "diff";
  }
  function valueOf(a: any): string {
    if (a.err) return "—";
    if (!a.values || !a.values.length) return a.rcode || "(no records)";
    return a.values.join(", ");
  }
</script>

<div class="tool">
  <h2>DNS propagation</h2>
  <p class="sub">
    Query one record from <strong>a dozen public resolvers worldwide at once</strong> and see whether they agree.
    After you change a record, a split answer means some resolvers still serve the old value (or the zone is
    inconsistent). Keyless; nothing is stored.
  </p>

  <form on:submit|preventDefault={run} class="row">
    <input class="field mono" bind:value={name} placeholder="example.com" autocomplete="off" spellcheck="false" aria-label="DNS name" />
    <select class="sel" bind:value={rtype} aria-label="Record type">
      {#each TYPES as t}<option value={t}>{t}</option>{/each}
    </select>
    <button class="go" disabled={loading || !name.trim()}>{loading ? "Querying…" : "Check"}</button>
  </form>

  {#if error}<div class="hint">{error}</div>{/if}

  {#if rep}
    <div class="banner {rep.consensus ? 'ok' : 'warn'}">
      {#if rep.consensus}
        <strong>Consensus</strong> — all {rep.total} responding resolvers return the same answer.
      {:else}
        <strong>Diverged</strong> — {rep.propagated} of {rep.total} agree; the rest still differ (propagation in progress or inconsistent).
      {/if}
    </div>

    {#if rep.groups && rep.groups.length > 1}
      <div class="groups">
        {#each rep.groups as g, i}
          <div class="grp {i === 0 ? 'maj' : 'diff'}">
            <div class="gval mono">{g.values.length ? g.values.join(", ") : g.rcode}</div>
            <div class="gres">{g.count} resolver{g.count === 1 ? "" : "s"}: {g.resolvers.join(", ")}</div>
          </div>
        {/each}
      </div>
    {/if}

    <table class="tbl">
      <thead><tr><th>Resolver</th><th>Location</th><th>Answer</th><th class="num">RTT</th></tr></thead>
      <tbody>
        {#each rep.answers as a}
          <tr class={groupClass(a.group)}>
            <td class="rname">{a.resolver.name} <span class="dim mono">{a.resolver.ip}</span></td>
            <td class="dim">{a.resolver.loc || ""}</td>
            <td class="mono aval">{valueOf(a)}{#if a.err}<span class="errtag">{a.err}</span>{/if}</td>
            <td class="num dim">{a.err ? "—" : a.rttMillis + "ms"}</td>
          </tr>
        {/each}
      </tbody>
    </table>
  {/if}
</div>

<style>
  .tool { padding: 16px 20px 40px; max-width: 1000px; }
  h2 { margin: 0 0 2px; font-size: 1.15rem; font-weight: 600; color: var(--text-strong); }
  .sub { margin: 0 0 15px; color: var(--text-dim); font-size: 0.86rem; line-height: 1.5; }
  .row { display: flex; gap: 8px; margin-bottom: 14px; flex-wrap: wrap; }
  .field { flex: 1; min-width: 200px; background: var(--well); border: 1px solid var(--field-border); border-radius: 7px; padding: 9px 12px; color: var(--text); font-size: 0.9rem; outline: none; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .sel { background: var(--well); border: 1px solid var(--field-border); border-radius: 7px; padding: 9px 10px; color: var(--text); font: inherit; font-size: 0.86rem; cursor: pointer; }
  .go { background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px; padding: 0 16px; font: inherit; font-size: 0.9rem; cursor: pointer; }
  .go:disabled { opacity: .5; cursor: default; }
  .hint { background: var(--warn-bg); border: 1px solid var(--warn-border); color: var(--warn-text); border-radius: 7px; padding: 9px 12px; font-size: 0.85rem; margin-bottom: 12px; }
  .banner { border: 1px solid var(--border); border-left-width: 3px; border-radius: 9px; padding: 11px 14px; margin-bottom: 14px; font-size: 0.88rem; color: var(--text); background: var(--card); }
  .banner.ok { border-left-color: var(--good, #3fb950); }
  .banner.warn { border-left-color: var(--warn-text, #f5a623); }
  .groups { display: flex; flex-direction: column; gap: 8px; margin-bottom: 14px; }
  .grp { border: 1px solid var(--border); border-left-width: 3px; border-radius: 8px; padding: 9px 12px; background: var(--well); }
  .grp.maj { border-left-color: var(--good, #3fb950); }
  .grp.diff { border-left-color: var(--danger, #e5484d); }
  .gval { font-size: 0.85rem; color: var(--text-strong); }
  .gres { font-size: 0.74rem; color: var(--text-dim); margin-top: 3px; }
  table.tbl { width: 100%; border-collapse: collapse; font-size: 0.82rem; }
  .tbl th { text-align: left; color: var(--text-dim); font-weight: 500; font-size: 0.68rem; text-transform: uppercase; letter-spacing: .04em; padding: 7px 10px; border-bottom: 1px solid var(--border); }
  .tbl th.num, .tbl td.num { text-align: right; }
  .tbl td { padding: 7px 10px; border-bottom: 1px solid var(--border-soft); vertical-align: top; }
  .tbl tr.diff td.aval { color: var(--danger, #e5484d); }
  .tbl tr.err td.aval { color: var(--text-faint); }
  .rname { color: var(--text); white-space: nowrap; }
  .aval { word-break: break-all; }
  .errtag { color: var(--text-faint); font-size: 0.72rem; margin-left: 6px; }
  .dim { color: var(--text-dim); }
  .mono { font-family: var(--font-mono); }
</style>
