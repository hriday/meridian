<script lang="ts">
  import { SIPProbe } from "../../../wailsjs/go/app/App.js";

  let host = "";
  let transport = "udp";
  let port = "";
  let loading = false;
  let error = "";
  let res: any = null;

  async function run() {
    const h = host.trim();
    if (!h || loading) return;
    loading = true; error = ""; res = null;
    try {
      res = await SIPProbe(h, port.trim(), transport);
    } catch (e: any) {
      error = String(e?.message ?? e);
    } finally {
      loading = false;
    }
  }

  const ok = (r: any) => r && r.statusCode >= 200 && r.statusCode < 300;
  const alive = (r: any) => r && r.statusCode > 0;
</script>

<div class="tool">
  <h2>SIP OPTIONS ping</h2>
  <p class="sub">
    Send a SIP <span class="mono">OPTIONS</span> request — the canonical "ping" for VoIP signaling — to a PBX, SIP trunk,
    or endpoint, and see whether it answers and what it supports. The first thing to check when calls won't set up.
    Nothing is stored.
  </p>

  <form on:submit|preventDefault={run} class="row">
    <input class="field mono" bind:value={host} placeholder="pbx.example.com  or  sip.provider.net" spellcheck="false" aria-label="SIP host" />
    <select class="sel" bind:value={transport} aria-label="Transport">
      <option value="udp">UDP</option>
      <option value="tcp">TCP</option>
      <option value="tls">TLS</option>
    </select>
    <input class="field port mono" bind:value={port} placeholder="port" spellcheck="false" aria-label="Port" />
    <button class="go" disabled={loading || !host.trim()}>{loading ? "Probing…" : "Probe"}</button>
  </form>

  {#if error}<div class="hint">{error}</div>{/if}

  {#if res}
    <div class="card {ok(res) ? 'ok' : alive(res) ? 'warn' : 'bad'}">
      <div class="chead">
        <span class="dot {ok(res) ? 'ok' : alive(res) ? 'warn' : 'bad'}"></span>
        <span class="status mono">{res.statusCode} {res.reason}</span>
        <span class="verdict">{ok(res) ? "reachable & healthy" : alive(res) ? "reachable — SIP is answering" : "no SIP response"}</span>
      </div>
      <div class="grid">
        <div class="k">Server</div><div class="v mono">{res.target} <span class="dim">({res.transport.toUpperCase()}, {res.rttMillis} ms)</span></div>
        {#if res.server}<div class="k">Server header</div><div class="v mono">{res.server}</div>{/if}
        {#if res.userAgent}<div class="k">User-Agent</div><div class="v mono">{res.userAgent}</div>{/if}
      </div>
      {#if res.allow && res.allow.length}
        <div class="sect"><div class="sk">Methods</div><div class="chips">{#each res.allow as m}<span class="chip mono">{m}</span>{/each}</div></div>
      {/if}
      {#if res.supported && res.supported.length}
        <div class="sect"><div class="sk">Supported extensions</div><div class="chips">{#each res.supported as s}<span class="chip mono">{s}</span>{/each}</div></div>
      {/if}
    </div>
  {/if}
</div>

<style>
  .tool { padding: 16px 20px 40px; max-width: 880px; }
  h2 { margin: 0 0 2px; font-size: 1.15rem; font-weight: 600; color: var(--text-strong); }
  .sub { margin: 0 0 15px; color: var(--text-dim); font-size: 0.86rem; line-height: 1.5; }
  .row { display: flex; gap: 8px; margin-bottom: 14px; flex-wrap: wrap; }
  .field { flex: 1; min-width: 180px; background: var(--well); border: 1px solid var(--field-border); border-radius: 7px; padding: 9px 12px; color: var(--text); font-size: 0.9rem; outline: none; }
  .field.port { flex: 0 0 72px; min-width: 0; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .sel { background: var(--well); border: 1px solid var(--field-border); border-radius: 7px; padding: 9px 10px; color: var(--text); font: inherit; font-size: 0.86rem; cursor: pointer; }
  .go { background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px; padding: 0 16px; font: inherit; font-size: 0.9rem; cursor: pointer; }
  .go:disabled { opacity: .5; cursor: default; }
  .hint { background: var(--warn-bg); border: 1px solid var(--warn-border); color: var(--warn-text); border-radius: 7px; padding: 9px 12px; font-size: 0.85rem; margin-bottom: 12px; }
  .card { border: 1px solid var(--border); border-left-width: 3px; border-radius: 11px; padding: 13px 16px; margin-top: 8px; background: var(--card); }
  .card.ok { border-left-color: var(--good, #3fb950); } .card.warn { border-left-color: var(--warn-text, #f5a623); } .card.bad { border-left-color: var(--danger, #e5484d); }
  .chead { display: flex; align-items: center; gap: 9px; flex-wrap: wrap; margin-bottom: 11px; }
  .dot { width: 10px; height: 10px; border-radius: 50%; flex: none; }
  .dot.ok { background: var(--good, #3fb950); } .dot.warn { background: var(--warn-text, #f5a623); } .dot.bad { background: var(--danger, #e5484d); }
  .status { font-weight: 650; color: var(--text-strong); }
  .verdict { font-size: 0.82rem; color: var(--text-dim); }
  .grid { display: grid; grid-template-columns: max-content 1fr; gap: 5px 16px; font-size: 0.84rem; align-items: baseline; }
  .k { color: var(--text-dim); white-space: nowrap; } .v { color: var(--text); }
  .sect { margin-top: 12px; }
  .sk { font-size: 0.68rem; text-transform: uppercase; letter-spacing: .04em; color: var(--text-faint, var(--text-dim)); font-weight: 650; margin-bottom: 6px; }
  .chips { display: flex; flex-wrap: wrap; gap: 5px; }
  .chip { font-size: 0.72rem; background: var(--well); border: 1px solid var(--border); border-radius: 6px; padding: 2px 8px; color: var(--text-dim); }
  .dim { color: var(--text-dim); }
  .mono { font-family: var(--font-mono); }
</style>
