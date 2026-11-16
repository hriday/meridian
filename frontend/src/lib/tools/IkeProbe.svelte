<script lang="ts">
  import { IkeProbe } from "../../../wailsjs/go/app/App.js";
  import ToolHelp from "../ToolHelp.svelte";

  let host = "";
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
      res = await IkeProbe(h);
    } catch (e: any) {
      error = String(e?.message ?? e);
    } finally {
      loading = false;
    }
  }

  $: ok = res && res.responds && res.accepted && res.accepted.length;
  $: verdict = !res
    ? ""
    : !res.responds
    ? "No IKE response — UDP/500 filtered, or no VPN responder here"
    : ok
    ? `IKEv2 responder up — accepts a proposal${res.natTraversal ? ", NAT-T supported" : ""}`
    : res.version === "IKEv1"
    ? "IKEv1 responder (legacy) — detailed proposal probing is IKEv2 only"
    : "IKE responder answered, but chose no proposal";
</script>

<div class="tool">
  <p class="lede">
    Probe an <strong>IKE / IPsec VPN gateway</strong> (IKEv2, UDP/500). It sends a real
    <span class="mono">IKE_SA_INIT</span> and reads back whether the gateway answers, the
    <strong>cryptographic proposal it accepts</strong> (encryption, PRF, integrity, DH group), and whether
    <strong>NAT traversal</strong> is supported. Read-only — it never completes the handshake or authenticates.
    Probe gateways you're authorized to test.
  </p>

  <ToolHelp>
    <strong>Responds: yes</strong> with an accepted proposal means a live IKEv2 responder — the
    <strong>encryption, PRF, integrity and DH group</strong> rows are exactly what it agreed to, so weak picks
    (a small DH group, a legacy cipher) are where to harden. <strong>No response</strong> means UDP/500 is
    filtered or nothing's listening — not proof the VPN is down. <strong>NAT traversal: yes</strong> means
    clients behind NAT can connect.
  </ToolHelp>

  <form class="row" on:submit|preventDefault={run}>
    <input class="field mono" bind:value={host} placeholder="vpn.example.com" autocomplete="off" spellcheck="false" aria-label="VPN gateway host" />
    <button class="btn" disabled={loading}>{loading ? "Probing…" : "Probe"}</button>
  </form>

  {#if error}<p class="err">{error}</p>{/if}

  {#if res}
    <div class="verdict {ok ? 'ok' : res.responds ? 'warn' : 'bad'}">{verdict}</div>
    <div class="grid">
      <div class="k">Target</div><div class="v mono">{res.target}</div>
      <div class="k">Responds</div><div class="v">{res.responds ? "yes" : "no"}</div>
      {#if res.responds}
        {#if res.version}<div class="k">Version</div><div class="v">{res.version}</div>{/if}
        <div class="k">NAT traversal</div><div class="v">{res.natTraversal ? "yes" : "no"}</div>
        {#if res.rttMs}<div class="k">Round-trip</div><div class="v">{res.rttMs} ms</div>{/if}
        {#each res.accepted || [] as t}
          <div class="k">{t.kind}</div><div class="v mono">{t.name}</div>
        {/each}
        {#if res.notify && res.notify.length}
          <div class="k">Notify</div><div class="v mono small">{res.notify.join(", ")}</div>
        {/if}
      {/if}
    </div>
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 30px; max-width: 760px; }
  .lede { margin: 0 0 14px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .lede strong { color: var(--text); }
  .row { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
  .field { flex: 1; min-width: 200px; background: var(--well); border: 1px solid var(--field-border, var(--border)); border-radius: 7px; padding: 8px 11px; color: var(--text); font-size: 0.88rem; outline: none; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
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
