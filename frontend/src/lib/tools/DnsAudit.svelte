<script lang="ts">
  import { DNSAudit, DNSSystem } from "../../../wailsjs/go/app/App.js";

  let domain = "";
  let resolver = "";
  let system: any = null;
  let busy = false;
  let error = "";
  let report: any = null;

  async function loadSystem() {
    try {
      system = await DNSSystem();
      if (!resolver && system?.values?.length) resolver = system.values[0];
    } catch (_) {
      system = null;
    }
  }
  loadSystem();

  async function run() {
    const v = domain.trim();
    if (!v) {
      report = null;
      return;
    }
    error = "";
    busy = true;
    report = null;
    try {
      report = await DNSAudit(v, resolver.trim());
    } catch (e) {
      report = null;
      error = String(e);
    } finally {
      busy = false;
    }
  }

  // Overall banner: green only when nothing is weak, missing, or errored.
  const allGood = (r: any) => r && r.warns === 0 && r.missing === 0 && r.errors === 0;

  const STATUS_LABEL: Record<string, string> = {
    ok: "OK",
    warn: "Warn",
    missing: "Missing",
    error: "Error",
  };
  const pillClass = (s: string) => (s === "ok" ? "good" : s === "error" ? "err" : "warn");
</script>

<div class="tool">
  <p class="lede">
    Grades a domain's DNS and email posture — nameservers, mail (MX), SPF, DMARC and
    DNSSEC — each in plain language. Read-only lookups through your resolver; the
    query carries only the domain.
  </p>

  <form on:submit|preventDefault={run} class="row">
    <input
      class="field mono"
      bind:value={domain}
      placeholder="example.com"
      autocomplete="off"
      spellcheck="false"
      aria-label="domain to audit"
    />
    <input
      class="field mono resolver"
      bind:value={resolver}
      placeholder="resolver (system)"
      autocomplete="off"
      spellcheck="false"
      aria-label="resolver"
      title="DNS resolver to query — defaults to your system resolver"
    />
    <button class="go" disabled={busy}>{busy ? "Auditing…" : "Audit"}</button>
  </form>

  {#if error}<div class="hint">Couldn't run the audit: {error}</div>{/if}

  {#if report}
    <div class="verdict" class:good={allGood(report)} class:warn={!allGood(report)}>
      <span class="vlabel">
        {report.oks}/{report.checks.length} healthy
        {#if report.warns}· {report.warns} weak{/if}
        {#if report.missing}· {report.missing} missing{/if}
        {#if report.errors}· {report.errors} error{/if}
      </span>
      <span class="vnote">{report.summary}</span>
    </div>

    <div class="card">
      <div class="k">Checks</div>
      {#each report.checks as c}
        <div class="check">
          <div class="chead">
            <span class="cname">{c.name}</span>
            <span class="pill {pillClass(c.status)}">{STATUS_LABEL[c.status] ?? c.status}</span>
          </div>
          <div class="cnote">{c.note}</div>
          {#if c.records?.length}
            <div class="recs">
              {#each c.records as rec}<div class="rec mono">{rec}</div>{/each}
            </div>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 20px; }
  .lede { max-width: 760px; margin: 0 0 14px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .row { display: flex; gap: 8px; max-width: 760px; }
  .field {
    background: var(--well); border: 1px solid var(--field-border); border-radius: 7px;
    padding: 9px 12px; color: var(--text); font-size: 0.9rem; outline: none;
  }
  .field:not(.resolver) { flex: 1; }
  .field.resolver { width: 190px; flex: none; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .mono { font-family: var(--mono, ui-monospace, monospace); }
  .go {
    background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px;
    padding: 0 16px; font: inherit; font-size: 0.9rem; cursor: pointer; white-space: nowrap;
  }
  .go:disabled { opacity: .6; cursor: default; }
  .hint { margin-top: 12px; color: var(--warn-text); font-size: 0.82rem; }

  .verdict {
    margin: 14px 0; padding: 11px 14px; border-radius: 9px; border: 1px solid var(--border-soft);
    display: flex; flex-direction: column; gap: 3px;
  }
  .verdict.good { border-color: transparent; background: var(--good-bg, #102516); }
  .verdict.warn { border-color: var(--warn, #d9a441); background: color-mix(in srgb, var(--warn, #d9a441) 12%, transparent); }
  .vlabel { font-weight: 600; color: var(--text-strong); }
  .verdict.good .vlabel { color: var(--good, #5ad17a); }
  .verdict.warn .vlabel { color: var(--warn, #d9a441); }
  .vnote { font-size: 0.82rem; color: var(--text-dim); }

  .card { background: var(--card); border: 1px solid var(--border-soft); border-radius: 10px; padding: 13px 15px; }
  .k { font-size: 0.74rem; text-transform: uppercase; letter-spacing: 0.04em; color: var(--text-faint); margin-bottom: 9px; }
  .check { padding: 10px 0; border-bottom: 1px solid var(--border-soft); }
  .check:last-child { border-bottom: none; padding-bottom: 0; }
  .check:first-of-type { padding-top: 0; }
  .chead { display: flex; align-items: center; gap: 9px; }
  .cname { font-size: 0.9rem; color: var(--text); font-weight: 500; }
  .cnote { font-size: 0.82rem; color: var(--text-dim); margin-top: 3px; line-height: 1.45; }
  .recs { margin-top: 6px; display: flex; flex-direction: column; gap: 2px; }
  .rec { font-size: 0.78rem; color: var(--text-dim); overflow-wrap: anywhere; }
  .pill { display: inline-flex; font-size: 0.72rem; padding: 2px 9px; border-radius: 20px; border: 1px solid var(--border); color: var(--text-dim); white-space: nowrap; }
  .pill.good { color: var(--good, #5ad17a); border-color: transparent; background: var(--good-bg, #102516); }
  .pill.warn { color: var(--warn, #d9a441); border-color: transparent; background: color-mix(in srgb, var(--warn, #d9a441) 14%, transparent); }
  .pill.err { color: var(--text-dim); border-color: var(--border); }
</style>
