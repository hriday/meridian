<script lang="ts">
  import { WgProbe } from "../../../wailsjs/go/app/App.js";

  let host = "";
  let port = 51820;
  let pubkey = "";
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
      res = await WgProbe(h, Number(port) || 51820, pubkey.trim());
    } catch (e: any) {
      error = String(e?.message ?? e);
    } finally {
      loading = false;
    }
  }

  $: ok = res && res.responds && (res.replyType === "cookie-reply" || res.replyType === "handshake-response");
</script>

<div class="tool">
  <p class="lede">
    Probe a <strong>WireGuard</strong> endpoint (UDP, default 51820). It sends a real Noise
    <span class="mono">handshake initiation</span>; a <strong>cookie reply</strong> or handshake response
    confirms a WireGuard responder. WireGuard is <strong>silent by design</strong> — it drops any initiation
    it can't authenticate — so for a meaningful probe paste the peer's <strong>public key</strong> (base64);
    without it, silence is expected and proves nothing. Read-only; probe endpoints you're authorized to test.
  </p>

  <form class="row" on:submit|preventDefault={run}>
    <input class="field mono" bind:value={host} placeholder="vpn.example.com" autocomplete="off" spellcheck="false" aria-label="WireGuard host" />
    <input class="field mono port" type="number" min="1" max="65535" bind:value={port} aria-label="Port" />
    <button class="btn" disabled={loading}>{loading ? "Probing…" : "Probe"}</button>
  </form>
  <input class="field mono wide" bind:value={pubkey} placeholder="peer public key (base64) — recommended" autocomplete="off" spellcheck="false" aria-label="Peer public key" />

  {#if error}<p class="err">{error}</p>{/if}

  {#if res}
    <div class="verdict {ok ? 'ok' : 'warn'}">{res.verdict}</div>
    <div class="grid">
      <div class="k">Target</div><div class="v mono">{res.target}</div>
      <div class="k">Valid mac1</div><div class="v">{res.validMac1 ? "yes (authenticated)" : "no (no peer key)"}</div>
      <div class="k">Responds</div><div class="v">{res.responds ? "yes" : "no"}</div>
      {#if res.responds}
        <div class="k">Reply</div><div class="v mono">{res.replyType}</div>
        {#if res.rttMs}<div class="k">Round-trip</div><div class="v">{res.rttMs} ms</div>{/if}
      {/if}
    </div>
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 30px; max-width: 760px; }
  .lede { margin: 0 0 14px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .lede strong { color: var(--text); }
  .row { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
  .field { flex: 1; min-width: 160px; background: var(--well); border: 1px solid var(--field-border, var(--border)); border-radius: 7px; padding: 8px 11px; color: var(--text); font-size: 0.88rem; outline: none; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .field.port { flex: 0 0 auto; min-width: 0; width: 92px; }
  .field.wide { width: 100%; box-sizing: border-box; margin-top: 10px; }
  .btn { background: var(--accent); color: var(--on-accent, #052e2b); border: 1px solid transparent; border-radius: 8px; padding: 8px 18px; font: inherit; font-size: 0.86rem; font-weight: 550; cursor: pointer; }
  .btn:disabled { opacity: .55; cursor: default; }
  .err { color: var(--danger, #e5484d); font-size: 0.82rem; margin: 12px 0 0; }
  .verdict { margin: 18px 0 12px; padding: 11px 14px; border-radius: 9px; font-size: 0.9rem; border: 1px solid var(--border); background: var(--card); color: var(--text-strong, var(--text)); border-left-width: 3px; }
  .verdict.ok { border-left-color: var(--accent); }
  .verdict.warn { border-left-color: #d9a400; }
  .grid { display: grid; grid-template-columns: 140px 1fr; gap: 6px 14px; font-size: 0.85rem; }
  .k { color: var(--text-dim); }
  .v { color: var(--text); word-break: break-word; }
  .mono { font-family: var(--font-mono, ui-monospace, monospace); }
</style>
