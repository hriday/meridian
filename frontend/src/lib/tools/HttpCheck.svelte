<script lang="ts">
  import { HTTPCheck } from "../../../wailsjs/go/app/App.js";

  let url = "";
  let busy = false;
  let error = "";
  let r: any = null;

  async function run() {
    const v = url.trim();
    if (!v) {
      r = null;
      return;
    }
    error = "";
    r = null;
    busy = true;
    try {
      r = await HTTPCheck(v);
    } catch (e) {
      error = String(e);
    } finally {
      busy = false;
    }
  }

  const klass = (s: number) =>
    s >= 200 && s < 300 ? "good" : s >= 500 ? "bad" : "warn";
  const fmt = (v: number) => (v == null ? "0" : v.toFixed(0));
  // Bar width as a fraction of total, for the timing row.
  const pct = (v: number, total: number) => (total > 0 ? Math.min(100, (v / total) * 100) : 0);

  // Pull out the highlighted headers; the rest go in the full list.
  const special = ["Server", "Content-Type", "Content-Length"];
  const otherHeaders = (hs: any[]) => (hs ?? []).filter((h) => !special.includes(h[0]));
</script>

<div class="tool">
  <p class="lede">
    Fetches a URL and shows what the endpoint actually does — final status, any redirects,
    the key headers, protocol and TLS, and where the time goes (DNS, connect, TLS, first byte).
    One GET; it reads the response, it doesn't download the body.
  </p>

  <form on:submit|preventDefault={run} class="row">
    <input class="field mono" bind:value={url} placeholder="github.com  ·  https://example.com/path"
      autocomplete="off" spellcheck="false" aria-label="URL" />
    <button class="go" disabled={busy}>{busy ? "Checking…" : "Check"}</button>
  </form>

  {#if error}<div class="banner bad"><span class="vlabel">Couldn't reach it</span><span class="vnote">{error}</span></div>{/if}

  {#if r}
    <div class="verdict" class:good={klass(r.status) === "good"} class:warn={klass(r.status) === "warn"} class:bad={klass(r.status) === "bad"}>
      <span class="vlabel">{r.status} {r.statusText} · {r.proto}</span>
      <span class="vnote mono">{r.finalUrl}</span>
    </div>

    {#if r.redirects?.length}
      <div class="card">
        <div class="k">Redirects</div>
        {#each r.redirects as h}<div class="hop mono"><span class="hs">{h.status}</span> {h.url}</div>{/each}
        <div class="hop mono final"><span class="hs ok">{r.status}</span> {r.finalUrl}</div>
      </div>
    {/if}

    <div class="cards">
      <div class="card">
        <div class="k">Timing (ms)</div>
        {#each [["DNS", r.timing.dnsMillis], ["Connect", r.timing.connectMillis], ["TLS", r.timing.tlsMillis], ["First byte", r.timing.ttfbMillis], ["Total", r.timing.totalMillis]] as [label, v]}
          <div class="trow">
            <span class="tlabel">{label}</span>
            <span class="tbar"><span class="tfill" style={`width:${pct(v, r.timing.totalMillis)}%`}></span></span>
            <span class="tval mono">{fmt(v)}</span>
          </div>
        {/each}
      </div>

      <div class="card">
        <div class="k">Response</div>
        <dl>
          {#if r.tlsVersion}<dt>TLS</dt><dd>{r.tlsVersion}</dd>{/if}
          {#if r.server}<dt>Server</dt><dd>{r.server}</dd>{/if}
          {#if r.contentType}<dt>Content-Type</dt><dd class="mono">{r.contentType}</dd>{/if}
          {#if r.contentLength >= 0}<dt>Content-Length</dt><dd>{r.contentLength}</dd>{/if}
        </dl>
      </div>
    </div>

    {#if otherHeaders(r.headers).length}
      <div class="card span">
        <div class="k">All headers</div>
        <table class="mono"><tbody>
          {#each otherHeaders(r.headers) as h}<tr><td class="hk">{h[0]}</td><td class="hv">{h[1]}</td></tr>{/each}
        </tbody></table>
      </div>
    {/if}
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 20px; }
  .lede { max-width: 820px; margin: 0 0 14px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .row { display: flex; gap: 8px; max-width: 820px; }
  .field { flex: 1; background: var(--well); border: 1px solid var(--field-border); border-radius: 7px; padding: 9px 12px; color: var(--text); font-size: 0.9rem; outline: none; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .mono { font-family: var(--mono, ui-monospace, monospace); }
  .go { background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px; padding: 0 16px; font: inherit; font-size: 0.9rem; cursor: pointer; white-space: nowrap; }
  .go:disabled { opacity: .6; cursor: default; }

  .verdict, .banner { margin: 14px 0; padding: 11px 14px; border-radius: 9px; border: 1px solid var(--border-soft); display: flex; flex-direction: column; gap: 3px; max-width: 820px; }
  .verdict.good { border-color: transparent; background: var(--good-bg, #102516); }
  .verdict.warn { border-color: var(--warn, #d9a441); background: color-mix(in srgb, var(--warn, #d9a441) 12%, transparent); }
  .verdict.bad, .banner.bad { border-color: var(--bad, #ff6b6b); background: color-mix(in srgb, var(--bad, #ff6b6b) 10%, transparent); }
  .vlabel { font-weight: 600; color: var(--text-strong); }
  .verdict.good .vlabel { color: var(--good, #5ad17a); }
  .verdict.warn .vlabel { color: var(--warn, #d9a441); }
  .verdict.bad .vlabel, .banner.bad .vlabel { color: var(--bad, #ff6b6b); }
  .vnote { font-size: 0.8rem; color: var(--text-dim); overflow-wrap: anywhere; }

  .cards { display: grid; grid-template-columns: 1fr 1fr; gap: 13px; max-width: 820px; }
  @media (max-width: 720px) { .cards { grid-template-columns: 1fr; } }
  .card { background: var(--card); border: 1px solid var(--border-soft); border-radius: 10px; padding: 13px 15px; max-width: 820px; margin-bottom: 13px; }
  .card.span { }
  .k { font-size: 0.74rem; text-transform: uppercase; letter-spacing: 0.04em; color: var(--text-faint); margin-bottom: 9px; }
  .hop { font-size: 0.8rem; color: var(--text-dim); padding: 2px 0; overflow-wrap: anywhere; }
  .hop.final { color: var(--text); }
  .hs { display: inline-block; min-width: 30px; color: var(--warn, #d9a441); }
  .hs.ok { color: var(--good, #5ad17a); }
  .trow { display: flex; align-items: center; gap: 10px; margin: 3px 0; font-size: 0.82rem; }
  .tlabel { width: 78px; color: var(--text-dim); }
  .tbar { flex: 1; height: 8px; background: var(--well); border-radius: 4px; overflow: hidden; }
  .tfill { display: block; height: 100%; background: var(--accent); border-radius: 4px; }
  .tval { width: 48px; text-align: right; color: var(--text); }
  dl { display: grid; grid-template-columns: max-content 1fr; gap: 5px 14px; margin: 0; }
  dt { color: var(--text-dim); font-size: 0.82rem; }
  dd { margin: 0; font-size: 0.86rem; color: var(--text); overflow-wrap: anywhere; }
  table { width: 100%; border-collapse: collapse; font-size: 0.78rem; }
  td { padding: 3px 10px 3px 0; vertical-align: top; border-bottom: 1px solid var(--border-soft); overflow-wrap: anywhere; }
  tr:last-child td { border-bottom: none; }
  .hk { color: var(--text-dim); white-space: nowrap; }
  .hv { color: var(--text); }
</style>
