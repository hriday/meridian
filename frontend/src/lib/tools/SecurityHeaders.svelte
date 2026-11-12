<script lang="ts">
  import { SecurityHeaders } from "../../../wailsjs/go/app/App.js";
  import { requestedTool } from "../nav";

  let url = "";
  let report: any = null;
  let loading = false;
  let err = "";

  // Allow a pivot from another tool (e.g. the Subdomain Finder) to prefill + run.
  const unsub = requestedTool.subscribe((r) => {
    if (r && r.tool === "sechdr" && r.value) { url = r.value; run(); requestedTool.set(null); }
  });
  import { onDestroy } from "svelte";
  onDestroy(unsub);

  async function run() {
    err = ""; report = null;
    if (!url.trim()) { err = "Enter a URL or host."; return; }
    loading = true;
    try { report = await SecurityHeaders(url.trim()); }
    catch (e) { err = String(e); }
    finally { loading = false; }
  }
  const gradeColor: Record<string, string> = { A: "var(--good,#3fb950)", B: "var(--good,#3fb950)", C: "var(--warn,#d6a531)", D: "var(--warn,#d6a531)", F: "var(--bad,#f0616a)" };
  function dot(grade: string): string {
    return grade === "ok" ? "var(--good,#3fb950)" : grade === "warn" ? "var(--warn,#d6a531)" : grade === "info" ? "var(--accent)" : "var(--bad,#f0616a)";
  }
  function label(grade: string): string {
    return grade === "ok" ? "OK" : grade === "warn" ? "weak" : grade === "info" ? "disclosure" : "missing";
  }
</script>

<div class="tool">
  <p class="lede">
    Fetch a URL and grade its live <strong>HTTP security headers</strong> — HSTS, CSP, clickjacking, MIME-sniffing,
    referrer and permissions policy — each mapped to its <strong>OWASP Top 10</strong> category. This checks the
    running server's response (not a config file).
  </p>

  <div class="searchrow">
    <input class="field mono" bind:value={url} placeholder="example.com  or  https://example.com/login" spellcheck="false" autocomplete="off"
      on:keydown={(e) => e.key === "Enter" && run()} />
    <button class="btn" on:click={run} disabled={loading || !url.trim()}>{loading ? "Scanning…" : "Scan headers"}</button>
  </div>

  {#if err}<div class="banner bad"><span>{err}</span></div>{/if}

  {#if report}
    <div class="scorecard">
      <div class="grade" style={`color:${gradeColor[report.grade] || "var(--text)"}`}>{report.grade}</div>
      <div class="scoremeta">
        <div class="stitle">Security headers · <span class="mono">{report.score}/100</span></div>
        <div class="dim sm mono">{report.finalUrl} · HTTP {report.status}{report.https ? " · HTTPS" : " · not HTTPS"}</div>
      </div>
    </div>

    <div class="card">
      {#each report.headers as h}
        <div class="hrow">
          <span class="hdot" style={`background:${dot(h.grade)}`}></span>
          <div class="hbody">
            <div class="hhead">
              <span class="hname mono">{h.name}</span>
              <span class="hstate" style={`color:${dot(h.grade)}`}>{label(h.grade)}</span>
              <span class="howasp">{h.owasp}</span>
            </div>
            {#if h.value}<div class="hval mono">{h.value}</div>{/if}
            <div class="hnote">{h.note}</div>
          </div>
        </div>
      {/each}
    </div>
    <p class="foot">OWASP tags reference the <a href="https://owasp.org/Top10/" target="_blank" rel="noreferrer">OWASP Top 10 (2021)</a>. Good headers reduce risk but aren't a full assessment.</p>
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 30px; }
  .lede { max-width: 820px; margin: 0 0 12px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .mono { font-family: var(--font-mono, ui-monospace, monospace); }
  .searchrow { display: flex; gap: 10px; max-width: 720px; margin-bottom: 14px; }
  .field { background: var(--well); border: 1px solid var(--field-border, var(--border)); border-radius: 8px; padding: 9px 12px; color: var(--text); font-size: 0.88rem; outline: none; flex: 1 1 auto; min-width: 0; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .btn { background: var(--accent); color: var(--on-accent, #052e2b); border: 1px solid transparent; border-radius: 8px; padding: 9px 16px; font: inherit; font-size: 0.86rem; font-weight: 550; cursor: pointer; white-space: nowrap; }
  .btn:disabled { opacity: .5; cursor: default; }
  .banner { display: flex; align-items: center; gap: 10px; max-width: 820px; padding: 9px 13px; border-radius: 9px; font-size: 0.84rem; margin-bottom: 12px; }
  .banner.bad { border: 1px solid var(--bad, #ff6b6b); background: color-mix(in srgb, var(--bad, #ff6b6b) 10%, transparent); color: var(--text); }
  .scorecard { display: flex; align-items: center; gap: 16px; max-width: 820px; margin-bottom: 14px; }
  .grade { font-size: 2.6rem; font-weight: 750; line-height: 1; font-family: var(--font-mono, monospace); width: 56px; text-align: center; }
  .stitle { font-weight: 650; color: var(--text-strong, var(--text)); font-size: 0.92rem; }
  .dim { color: var(--text-dim); } .sm { font-size: 0.76rem; }
  .card { background: var(--card); border: 1px solid var(--border); border-radius: 11px; max-width: 820px; overflow: hidden; }
  .hrow { display: flex; gap: 11px; padding: 11px 14px; border-bottom: 1px solid var(--border-soft, var(--border)); }
  .hrow:last-child { border-bottom: 0; }
  .hdot { width: 9px; height: 9px; border-radius: 50%; flex: none; margin-top: 5px; }
  .hbody { min-width: 0; flex: 1; }
  .hhead { display: flex; align-items: baseline; gap: 10px; flex-wrap: wrap; }
  .hname { font-weight: 600; color: var(--text); font-size: 0.84rem; }
  .hstate { font-size: 0.68rem; text-transform: uppercase; letter-spacing: .05em; font-weight: 650; }
  .howasp { margin-left: auto; font-size: 0.68rem; color: var(--text-faint, var(--text-dim)); border: 1px solid var(--border); border-radius: 20px; padding: 1px 8px; }
  .hval { font-size: 0.76rem; color: var(--text-dim); margin-top: 4px; overflow-wrap: anywhere; }
  .hnote { font-size: 0.8rem; color: var(--text-dim); margin-top: 3px; line-height: 1.5; }
  .foot { max-width: 820px; font-size: 0.74rem; color: var(--text-faint, var(--text-dim)); margin: 12px 0 0; }
</style>
