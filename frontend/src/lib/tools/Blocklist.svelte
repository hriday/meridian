<script lang="ts">
  import { Blocklist, DNSSystem } from "../../../wailsjs/go/app/App.js";

  let ip = "";
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
    const v = ip.trim();
    if (!v) {
      report = null;
      return;
    }
    error = "";
    busy = true;
    report = null;
    try {
      report = await Blocklist(v, resolver.trim());
    } catch (e) {
      report = null;
      error = String(e);
    } finally {
      busy = false;
    }
  }

  // Verdict banner state: red if listed, amber if the check was incomplete
  // (some lists unreachable) but nothing listed, green if a clean full pass.
  const listed = (r: any) => r && r.listed > 0;
  const incomplete = (r: any) => r && r.listed === 0 && r.unreachable > 0;
  const clean = (r: any) => r && r.listed === 0 && r.unreachable === 0;

  const STATUS_LABEL: Record<string, string> = {
    listed: "Listed",
    clean: "Clean",
    blocked: "Refused",
    error: "Error",
  };
  const pillClass = (s: string) =>
    s === "listed" ? "bad" : s === "clean" ? "good" : "warn";
</script>

<div class="tool">
  <p class="lede">
    Checks whether an IP is on public DNS blocklists (DNSBLs) — the same lists mail
    servers consult. Queries carry only the reversed IP, nothing about you.
    <b>Use your own resolver:</b> big public resolvers (8.8.8.8 / 1.1.1.1) get these
    queries refused, which shows as <i>Refused</i>, not a clean result.
  </p>

  <form on:submit|preventDefault={run} class="row">
    <input
      class="field mono"
      bind:value={ip}
      placeholder="198.51.100.23"
      autocomplete="off"
      spellcheck="false"
      aria-label="IP address to check"
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
    <button class="go" disabled={busy}>{busy ? "Checking…" : "Check"}</button>
  </form>

  {#if error}<div class="hint">Couldn't run the check: {error}</div>{/if}

  {#if report}
    <div
      class="verdict"
      class:good={clean(report)}
      class:warn={incomplete(report)}
      class:bad={listed(report)}
    >
      <span class="vlabel">{report.verdict}</span>
      <span class="vnote">{report.summary}</span>
    </div>

    <div class="card">
      <div class="k">Lists checked ({report.checked})</div>
      <table>
        <thead>
          <tr><th>List</th><th>Status</th><th>Codes</th><th>Detail</th></tr>
        </thead>
        <tbody>
          {#each report.listings as l}
            <tr>
              <td>
                <div class="lname">{l.name}</div>
                <div class="lzone mono dim">{l.zone}</div>
              </td>
              <td><span class="pill {pillClass(l.status)}">{STATUS_LABEL[l.status] ?? l.status}</span></td>
              <td class="mono dim">{(l.codes ?? []).join(", ") || "—"}</td>
              <td class="detail">{l.reason || l.note || ""}</td>
            </tr>
          {/each}
        </tbody>
      </table>
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
  .verdict.bad { border-color: var(--bad, #ff6b6b); background: color-mix(in srgb, var(--bad, #ff6b6b) 10%, transparent); }
  .verdict.warn { border-color: var(--warn, #d9a441); background: color-mix(in srgb, var(--warn, #d9a441) 12%, transparent); }
  .vlabel { font-weight: 600; color: var(--text-strong); }
  .verdict.good .vlabel { color: var(--good, #5ad17a); }
  .verdict.bad .vlabel { color: var(--bad, #ff6b6b); }
  .verdict.warn .vlabel { color: var(--warn, #d9a441); }
  .vnote { font-size: 0.82rem; color: var(--text-dim); }

  .card { background: var(--card); border: 1px solid var(--border-soft); border-radius: 10px; padding: 13px 15px; }
  .k { font-size: 0.74rem; text-transform: uppercase; letter-spacing: 0.04em; color: var(--text-faint); margin-bottom: 9px; }
  table { width: 100%; border-collapse: collapse; font-size: 0.86rem; }
  th {
    text-align: left; font-weight: 500; color: var(--text-dim); font-size: 0.74rem;
    text-transform: uppercase; letter-spacing: 0.04em; padding: 0 10px 6px 0; border-bottom: 1px solid var(--border-soft);
  }
  td { padding: 8px 10px 8px 0; vertical-align: top; border-bottom: 1px solid var(--border-soft); color: var(--text); }
  tr:last-child td { border-bottom: none; }
  .lname { color: var(--text); }
  .lzone { font-size: 0.74rem; }
  .dim { color: var(--text-dim); }
  .detail { font-size: 0.8rem; color: var(--text-dim); overflow-wrap: anywhere; }
  .pill { display: inline-flex; font-size: 0.72rem; padding: 2px 9px; border-radius: 20px; border: 1px solid var(--border); color: var(--text-dim); white-space: nowrap; }
  .pill.good { color: var(--good, #5ad17a); border-color: transparent; background: var(--good-bg, #102516); }
  .pill.bad { color: var(--bad, #ff6b6b); border-color: transparent; background: color-mix(in srgb, var(--bad, #ff6b6b) 12%, transparent); }
  .pill.warn { color: var(--warn, #d9a441); border-color: transparent; background: color-mix(in srgb, var(--warn, #d9a441) 14%, transparent); }
</style>
