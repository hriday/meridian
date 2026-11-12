<script lang="ts">
  import { CVELookup, CVESearch, CVEIsID } from "../../../wailsjs/go/app/App.js";
  import { BrowserOpenURL } from "../../../wailsjs/runtime/runtime.js";

  let query = "";
  let results: any[] = [];
  let loading = false;
  let err = "";
  let searched = "";
  let mode: "id" | "search" = "search";

  async function run() {
    err = ""; results = [];
    const q = query.trim();
    if (!q) { err = "Enter a CVE id (CVE-2021-44228) or a keyword (log4j, openssl, …)."; return; }
    loading = true;
    try {
      if (await CVEIsID(q)) { mode = "id"; results = [await CVELookup(q)]; }
      else { mode = "search"; results = (await CVESearch(q)) || []; }
      searched = q;
    } catch (e) { err = String(e); }
    finally { loading = false; }
  }
  function sevColor(sev: string, score: number): string {
    const s = (sev || "").toUpperCase();
    if (s === "CRITICAL" || score >= 9) return "var(--bad,#f0616a)";
    if (s === "HIGH" || score >= 7) return "var(--warn,#d6a531)";
    if (s === "MEDIUM" || score >= 4) return "var(--warn,#d6a531)";
    return "var(--good,#3fb950)";
  }
</script>

<div class="tool">
  <p class="lede">
    Look up a <strong>CVE</strong> by id, or search by keyword (a product or technology). Shows the CVSS severity,
    the weakness type (CWE), and — importantly — whether <strong>CISA lists it as actively exploited</strong>.
    Data from the <a href="https://nvd.nist.gov" target="_blank" rel="noreferrer">NVD</a>; free, no key.
  </p>

  <div class="searchrow">
    <input class="field mono" bind:value={query} placeholder="CVE-2021-44228   or   log4j, openssl 3.0, fortinet…" spellcheck="false" autocomplete="off"
      on:keydown={(e) => e.key === "Enter" && run()} />
    <button class="btn" on:click={run} disabled={loading || !query.trim()}>{loading ? "Searching NVD…" : "Look up"}</button>
  </div>
  <div class="note">The NVD is keyless but rate-limited (a few requests per 30s) — if it throttles, wait a moment and retry.</div>

  {#if err}<div class="banner bad"><span>{err}</span></div>{/if}

  {#if searched && !loading && !err}
    {#if mode === "search"}
      <div class="summary"><span class="exp mono">{results.length}</span> result{results.length === 1 ? "" : "s"} for <span class="mono">{searched}</span>{#if results.length >= 30} (showing the first 30){/if}.</div>
    {/if}
    {#if results.length === 0}
      <p class="muted">No CVEs found.</p>
    {/if}
    <div class="list">
      {#each results as c}
        <div class="cve">
          <div class="chead">
            <span class="cid mono">{c.id}</span>
            {#if c.cvss > 0}<span class="sev" style={`background:${sevColor(c.severity, c.cvss)}`}>{c.severity || "—"} {c.cvss.toFixed(1)}</span>{/if}
            {#if c.kev}<span class="kev" title="CISA Known Exploited Vulnerabilities catalogue">⚠ Exploited in the wild</span>{/if}
            <span class="cspace"></span>
            {#if c.published}<span class="dim sm mono">{c.published}</span>{/if}
          </div>
          {#if c.description}<p class="cdesc">{c.description}</p>{/if}
          <div class="cmeta">
            {#each c.cwe || [] as w}<span class="cwe">{w}</span>{/each}
            {#if c.vector}<span class="vec mono sm dim" title="CVSS vector">{c.vector}</span>{/if}
          </div>
          {#if c.references?.length}
            <div class="refs">{#each c.references.slice(0, 5) as r}<button class="reflink" on:click={() => BrowserOpenURL(r)} title={r}>{(() => { try { return new URL(r).hostname; } catch (e) { return r; } })()} ↗</button>{/each}</div>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 30px; }
  .lede { max-width: 820px; margin: 0 0 12px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .mono { font-family: var(--font-mono, ui-monospace, monospace); }
  .muted { color: var(--text-dim); font-size: 0.85rem; }
  .searchrow { display: flex; gap: 10px; max-width: 760px; margin-bottom: 8px; }
  .field { background: var(--well); border: 1px solid var(--field-border, var(--border)); border-radius: 8px; padding: 9px 12px; color: var(--text); font-size: 0.88rem; outline: none; flex: 1 1 auto; min-width: 0; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .btn { background: var(--accent); color: var(--on-accent, #052e2b); border: 1px solid transparent; border-radius: 8px; padding: 9px 16px; font: inherit; font-size: 0.86rem; font-weight: 550; cursor: pointer; white-space: nowrap; }
  .btn:disabled { opacity: .5; cursor: default; }
  .note { max-width: 820px; font-size: 0.76rem; color: var(--text-faint, var(--text-dim)); margin-bottom: 14px; }
  .banner { display: flex; align-items: center; gap: 10px; max-width: 820px; padding: 9px 13px; border-radius: 9px; font-size: 0.84rem; margin-bottom: 12px; }
  .banner.bad { border: 1px solid var(--bad, #ff6b6b); background: color-mix(in srgb, var(--bad, #ff6b6b) 10%, transparent); color: var(--text); }
  .summary { max-width: 820px; font-size: 0.86rem; color: var(--text-dim); margin-bottom: 10px; }
  .summary .exp { color: var(--accent); font-weight: 650; }
  .list { display: flex; flex-direction: column; gap: 10px; max-width: 820px; }
  .cve { background: var(--card); border: 1px solid var(--border); border-radius: 11px; padding: 12px 14px; }
  .chead { display: flex; align-items: center; gap: 9px; flex-wrap: wrap; }
  .cid { font-weight: 650; color: var(--text-strong, var(--text)); font-size: 0.9rem; }
  .sev { color: #fff; font-size: 0.68rem; font-weight: 650; padding: 1px 8px; border-radius: 20px; white-space: nowrap; }
  .kev { font-size: 0.68rem; font-weight: 650; color: var(--bad, #f0616a); border: 1px solid color-mix(in srgb, var(--bad,#f0616a) 45%, var(--border)); border-radius: 20px; padding: 1px 8px; }
  .cspace { flex: 1 1 auto; }
  .dim { color: var(--text-dim); } .sm { font-size: 0.74rem; }
  .cdesc { font-size: 0.82rem; color: var(--text-dim); line-height: 1.55; margin: 8px 0 0; overflow-wrap: anywhere; }
  .cmeta { display: flex; align-items: center; gap: 7px; flex-wrap: wrap; margin-top: 8px; }
  .cwe { font-size: 0.68rem; color: var(--accent); border: 1px solid color-mix(in srgb, var(--accent) 40%, var(--border)); border-radius: 6px; padding: 1px 7px; }
  .vec { margin-left: auto; overflow-wrap: anywhere; }
  .refs { display: flex; flex-wrap: wrap; gap: 6px; margin-top: 9px; }
  .reflink { background: var(--well); border: 1px solid var(--border-soft, var(--border)); border-radius: 6px; padding: 2px 8px; font-size: 0.72rem; color: var(--accent); cursor: pointer; }
  .reflink:hover { border-color: var(--accent); }
</style>
