<script lang="ts">
  import { HTTP3Probe } from "../../../wailsjs/go/app/App.js";
  import ToolHelp from "../ToolHelp.svelte";

  let host = "cloudflare.com";
  let port = 443;
  let loading = false;
  let error = "";
  let res: any = null;

  async function run() {
    const h = host.trim();
    if (!h || loading) return;
    loading = true;
    error = "";
    res = null;
    try {
      res = await HTTP3Probe(h, Number(port) || 443);
    } catch (e: any) {
      error = String(e?.message ?? e);
    } finally {
      loading = false;
    }
  }

  $: works = res && res.quicReachable && res.alpn === "h3";
  $: blocked = res && !res.quicReachable && res.advertisesH3;
</script>

<div class="tool">
  <p class="lede">
    Test <strong>HTTP/3 (QUIC)</strong> reachability: it attempts the real QUIC handshake on UDP/443 <em>and</em> checks
    whether the server advertises HTTP/3 via <span class="mono">Alt-Svc</span>. Together they tell apart
    <strong>"the server doesn't support h3"</strong> from <strong>"it does, but UDP is blocked on your path"</strong> —
    the usual restrictive-network case. No admin rights.
  </p>

  <ToolHelp>
    Read the two signals together. <strong>QUIC reachable: yes</strong> with <code>ALPN h3</code> means HTTP/3 works
    end to end from where you are. <strong>Advertises h3: yes</strong> but <strong>QUIC reachable: no</strong> is the
    telling case — the server offers HTTP/3 but UDP/443 is blocked on your path (firewall or network), not the server's
    fault. Both <code>no</code> means the server simply doesn't do HTTP/3. The amber verdict flags that blocked-path case
    specifically.
  </ToolHelp>

  <form class="row" on:submit|preventDefault={run}>
    <input class="field mono" bind:value={host} placeholder="cloudflare.com" autocomplete="off" spellcheck="false" aria-label="Host" />
    <input class="field mono port" type="number" min="1" max="65535" bind:value={port} aria-label="Port" />
    <button class="btn" disabled={loading}>{loading ? "Probing…" : "Probe"}</button>
  </form>

  {#if error}<p class="err">{error}</p>{/if}

  {#if res}
    <div class="verdict {works ? 'ok' : blocked ? 'warn' : 'bad'}">{res.verdict}</div>
    <div class="grid">
      <div class="k">QUIC reachable</div><div class="v">{res.quicReachable ? "yes" : "no"}</div>
      {#if res.quicReachable}
        <div class="k">ALPN</div><div class="v mono">{res.alpn || "—"}</div>
        <div class="k">TLS</div><div class="v">{res.tlsVersion}</div>
        <div class="k">Handshake</div><div class="v">{res.handshakeMillis} ms</div>
      {:else if res.quicError}
        <div class="k">QUIC error</div><div class="v mono small">{res.quicError}</div>
      {/if}
      <div class="k">Advertises h3</div><div class="v">{res.advertisesH3 ? "yes" : "no"}</div>
      {#if res.altSvc}<div class="k">Alt-Svc</div><div class="v mono small">{res.altSvc}</div>{/if}
    </div>
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 30px; max-width: 760px; }
  .lede { margin: 0 0 14px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .row { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
  .field { flex: 1; min-width: 160px; background: var(--well); border: 1px solid var(--field-border, var(--border)); border-radius: 7px; padding: 8px 11px; color: var(--text); font-size: 0.88rem; outline: none; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .field.port { flex: 0 0 auto; min-width: 0; width: 92px; }
  .btn { background: var(--accent); color: var(--on-accent, #052e2b); border: 1px solid transparent; border-radius: 8px; padding: 8px 18px; font: inherit; font-size: 0.86rem; font-weight: 550; cursor: pointer; }
  .btn:disabled { opacity: .55; cursor: default; }
  .err { color: var(--danger, #e5484d); font-size: 0.82rem; margin: 12px 0 0; }
  .verdict { margin: 18px 0 12px; padding: 11px 14px; border-radius: 9px; font-size: 0.9rem; border: 1px solid var(--border); background: var(--card); color: var(--text-strong, var(--text)); border-left-width: 3px; }
  .verdict.ok { border-left-color: var(--accent); }
  .verdict.warn { border-left-color: #d9a400; }
  .verdict.bad { border-left-color: var(--danger, #e5484d); }
  .grid { display: grid; grid-template-columns: 140px 1fr; gap: 6px 14px; font-size: 0.85rem; }
  .k { color: var(--text-dim); }
  .v { color: var(--text); word-break: break-word; }
  .v.small { font-size: 0.76rem; }
  .mono { font-family: var(--font-mono, ui-monospace, monospace); }
</style>
