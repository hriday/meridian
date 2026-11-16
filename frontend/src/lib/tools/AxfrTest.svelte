<script lang="ts">
  import { AXFRTest } from "../../../wailsjs/go/app/App.js";

  let domain = "";
  let rep: any = null;
  let loading = false;
  let error = "";

  async function run() {
    const d = domain.trim();
    if (!d) return;
    loading = true;
    error = "";
    try {
      rep = await AXFRTest(d, "");
    } catch (e: any) {
      error = String(e?.message ?? e);
      rep = null;
    } finally {
      loading = false;
    }
  }
  const cls = (s: string) => (s === "transferred" ? "bad" : s === "refused" ? "ok" : "neu");
  const label = (s: string) => (s === "transferred" ? "EXPOSED" : s === "refused" ? "refused" : "error");
</script>

<div class="tool">
  <p class="lede">
    Tests whether a domain's authoritative <strong>nameservers allow a zone transfer (AXFR)</strong> to any client. A
    correctly configured server refuses; one that allows it hands over the <strong>entire zone</strong> — every host and
    subdomain — in one query. Read-only: it asks for the transfer and reports whether the server would have given it.
  </p>

  <div class="row">
    <input class="field mono" bind:value={domain} placeholder="example.com" spellcheck="false" autocomplete="off"
      on:keydown={(e) => e.key === "Enter" && run()} />
    <button class="btn" on:click={run} disabled={loading || !domain.trim()}>{loading ? "Testing…" : "Test AXFR"}</button>
  </div>

  {#if error}<p class="err">{error}</p>{/if}

  {#if rep}
    <div class="summary {rep.exposed ? 'bad' : 'ok'}">{rep.summary}</div>
    <div class="cards">
      {#each rep.nameservers as ns}
        <div class="card {cls(ns.status)}">
          <div class="chead">
            <span class="mono nsname">{ns.nameserver}</span>
            <span class="badge {cls(ns.status)}">{label(ns.status)}</span>
          </div>
          <p class="note">{ns.detail}</p>
          {#if ns.sample && ns.sample.length}
            <div class="leak">
              {#each ns.sample as s}<code class="rec">{s}</code>{/each}
              {#if ns.records > ns.sample.length}<span class="more">…and {ns.records - ns.sample.length} more records</span>{/if}
            </div>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 30px; }
  .lede { max-width: 820px; margin: 0 0 14px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .mono { font-family: var(--font-mono, ui-monospace, monospace); }
  .row { display: flex; gap: 10px; max-width: 620px; }
  .field { background: var(--well); border: 1px solid var(--field-border, var(--border)); border-radius: 8px; padding: 9px 12px; color: var(--text); font-size: 0.86rem; outline: none; flex: 1 1 auto; min-width: 0; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .btn { background: var(--accent); color: var(--on-accent, #052e2b); border: 1px solid transparent; border-radius: 8px; padding: 9px 16px; font: inherit; font-size: 0.86rem; font-weight: 550; cursor: pointer; white-space: nowrap; }
  .btn:disabled { opacity: .5; cursor: default; }
  .err { color: var(--danger, #e5484d); font-size: 0.82rem; margin: 12px 0 0; }
  .summary { max-width: 820px; margin: 16px 0 10px; padding: 9px 13px; border-radius: 9px; font-size: 0.85rem; border: 1px solid var(--border); border-left-width: 3px; }
  .summary.ok { border-left-color: var(--good, #3fb950); }
  .summary.bad { border-left-color: var(--danger, #e5484d); color: var(--text); }
  .cards { display: flex; flex-direction: column; gap: 10px; max-width: 820px; }
  .card { border: 1px solid var(--border); border-left-width: 3px; border-radius: 10px; padding: 11px 14px; background: var(--card); }
  .card.ok { border-left-color: var(--good, #3fb950); }
  .card.bad { border-left-color: var(--danger, #e5484d); }
  .card.neu { border-left-color: var(--text-faint, var(--text-dim)); }
  .chead { display: flex; align-items: center; justify-content: space-between; margin-bottom: 6px; gap: 10px; }
  .nsname { font-size: 0.85rem; color: var(--text); word-break: break-all; }
  .badge { font-size: 0.68rem; font-weight: 700; text-transform: uppercase; letter-spacing: .03em; padding: 2px 9px; border-radius: 6px; white-space: nowrap; }
  .badge.ok { background: var(--good, #3fb950); color: #04260f; }
  .badge.bad { background: var(--danger, #e5484d); color: #fff; }
  .badge.neu { background: var(--well); color: var(--text-dim); border: 1px solid var(--border); }
  .note { margin: 0; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .leak { margin-top: 7px; display: flex; flex-direction: column; gap: 4px; }
  .rec { font-family: var(--font-mono, ui-monospace, monospace); font-size: 0.75rem; color: var(--text); background: var(--well); border-radius: 6px; padding: 4px 8px; word-break: break-all; }
  .more { font-size: 0.74rem; color: var(--text-faint, var(--text-dim)); }
</style>
