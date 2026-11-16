<script lang="ts">
  import { SMTPProbe } from "../../../wailsjs/go/app/App.js";
  import ToolHelp from "../ToolHelp.svelte";

  let host = "";
  let mode = "auto"; // auto(25/starttls) | submission(587) | implicit(465) | mx
  let loading = false;
  let error = "";
  let res: any = null;

  async function run() {
    const h = host.trim();
    if (!h || loading) return;
    loading = true;
    error = "";
    res = null;
    let port = "";
    let implicit = false;
    let useMX = false;
    if (mode === "submission") port = "587";
    else if (mode === "implicit") { implicit = true; port = "465"; }
    else if (mode === "mx") useMX = true;
    try {
      res = await SMTPProbe(h, port, implicit, useMX);
    } catch (e: any) {
      error = String(e?.message ?? e);
    } finally {
      loading = false;
    }
  }

  function tlsLabel(r: any): string {
    if (r.implicitTLS) return "Implicit TLS (connected encrypted)";
    if (r.startTLS) return "STARTTLS offered";
    return "No TLS offered";
  }
</script>

<div class="tool">
  <h2>SMTP probe</h2>
  <p class="sub">
    Actually talk to a mail server: read its <strong>banner</strong>, list its <strong>EHLO capabilities</strong>,
    check <strong>STARTTLS</strong>, and inspect the <strong>TLS certificate</strong>. Complements the email-auth
    tool (which only reads DNS). Never authenticates or sends mail. Point-in-time; nothing stored.
  </p>

  <ToolHelp>
    Read the <strong>TLS</strong> row first: <strong>STARTTLS offered</strong> or <strong>implicit TLS</strong> is
    what you want — <strong>No TLS</strong> means mail to this host can cross in the clear. Then check
    <strong>Valid for host</strong> is <code>yes</code> and <strong>Expires</strong> isn't near zero: a mismatch
    or a soon-to-expire cert warns or breaks delivery. The <strong>EHLO capabilities</strong> and
    <strong>auth mechanisms</strong> list what the server supports.
  </ToolHelp>

  <form on:submit|preventDefault={run} class="row">
    <input class="field mono" bind:value={host} placeholder="mail.example.com  or  example.com (with MX)" autocomplete="off" spellcheck="false" aria-label="Host or domain" />
    <select class="sel" bind:value={mode} aria-label="Connection mode">
      <option value="auto">Port 25 / STARTTLS</option>
      <option value="submission">Submission (587)</option>
      <option value="implicit">Implicit TLS (465)</option>
      <option value="mx">Domain → MX (25)</option>
    </select>
    <button class="go" disabled={loading || !host.trim()}>{loading ? "Probing…" : "Probe"}</button>
  </form>

  {#if error}<div class="hint">{error}</div>{/if}

  {#if res}
    <div class="card">
      <div class="grid">
        <div class="k">Server</div><div class="v mono">{res.host}:{res.port}</div>
        <div class="k">Banner</div><div class="v mono">{res.banner}</div>
        {#if res.greeting}<div class="k">Greeting</div><div class="v">{res.greeting}</div>{/if}
        <div class="k">TLS</div>
        <div class="v"><span class="tls {res.implicitTLS || res.startTLS ? 'ok' : 'no'}">{tlsLabel(res)}</span></div>
      </div>

      {#if res.tls}
        <div class="sect">
          <div class="sk">Certificate</div>
          <div class="grid">
            <div class="k">Protocol</div><div class="v mono">{res.tls.version} · {res.tls.cipher}</div>
            <div class="k">Subject</div><div class="v mono">{res.tls.subject}</div>
            <div class="k">Issuer</div><div class="v mono">{res.tls.issuer}</div>
            <div class="k">Valid for host</div>
            <div class="v"><span class="badge {res.tls.hostMatch ? 'ok' : 'bad'}">{res.tls.hostMatch ? "yes" : "NO — mismatch"}</span></div>
            <div class="k">Expires</div>
            <div class="v"><span class={res.tls.daysLeft < 15 ? "soon" : ""}>{res.tls.daysLeft} days</span> <span class="dim">({(res.tls.notAfter || "").slice(0,10)})</span></div>
          </div>
        </div>
      {/if}

      {#if res.authMechs && res.authMechs.length}
        <div class="sect"><div class="sk">Auth mechanisms</div><div class="chips">{#each res.authMechs as m}<span class="chip mono">{m}</span>{/each}</div></div>
      {/if}

      {#if res.capabilities && res.capabilities.length}
        <div class="sect"><div class="sk">EHLO capabilities</div><div class="chips">{#each res.capabilities as c}<span class="chip mono">{c}</span>{/each}</div></div>
      {/if}
    </div>
  {/if}
</div>

<style>
  .tool { padding: 16px 20px 40px; max-width: 900px; }
  h2 { margin: 0 0 2px; font-size: 1.15rem; font-weight: 600; color: var(--text-strong); }
  .sub { margin: 0 0 15px; color: var(--text-dim); font-size: 0.86rem; line-height: 1.5; }
  .row { display: flex; gap: 8px; margin-bottom: 14px; flex-wrap: wrap; }
  .field { flex: 1; min-width: 200px; background: var(--well); border: 1px solid var(--field-border); border-radius: 7px; padding: 9px 12px; color: var(--text); font-size: 0.9rem; outline: none; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .sel { background: var(--well); border: 1px solid var(--field-border); border-radius: 7px; padding: 9px 10px; color: var(--text); font: inherit; font-size: 0.84rem; cursor: pointer; }
  .go { background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px; padding: 0 16px; font: inherit; font-size: 0.9rem; cursor: pointer; }
  .go:disabled { opacity: .5; cursor: default; }
  .hint { background: var(--warn-bg); border: 1px solid var(--warn-border); color: var(--warn-text); border-radius: 7px; padding: 9px 12px; font-size: 0.85rem; margin-bottom: 12px; }
  .card { border: 1px solid var(--border); border-radius: 11px; padding: 14px 16px; margin-top: 8px; background: var(--card); }
  .grid { display: grid; grid-template-columns: max-content 1fr; gap: 5px 16px; font-size: 0.84rem; align-items: baseline; }
  .k { color: var(--text-dim); white-space: nowrap; } .v { color: var(--text); word-break: break-word; }
  .sect { margin-top: 14px; }
  .sk { font-size: 0.68rem; text-transform: uppercase; letter-spacing: .04em; color: var(--text-faint, var(--text-dim)); font-weight: 650; margin-bottom: 7px; }
  .tls.ok { color: var(--good, #3fb950); } .tls.no { color: var(--danger, #e5484d); }
  .badge { font-size: 0.72rem; font-weight: 650; padding: 1px 8px; border-radius: 6px; }
  .badge.ok { background: color-mix(in srgb, var(--good, #3fb950) 18%, transparent); color: var(--good, #3fb950); }
  .badge.bad { background: var(--danger, #e5484d); color: #fff; }
  .soon { color: var(--danger, #e5484d); font-weight: 600; }
  .chips { display: flex; flex-wrap: wrap; gap: 6px; }
  .chip { font-size: 0.72rem; background: var(--well); border: 1px solid var(--border); border-radius: 6px; padding: 2px 8px; color: var(--text-dim); }
  .dim { color: var(--text-dim); }
  .mono { font-family: var(--font-mono); }
</style>
