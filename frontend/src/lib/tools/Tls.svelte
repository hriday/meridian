<script lang="ts">
  import { TLSCheck } from "../../../wailsjs/go/app/App.js";

  let query = "cloudflare.com";
  let busy = false;
  let error = "";
  let report: any = null;

  async function run() {
    const v = query.trim();
    if (!v) {
      report = null;
      return;
    }
    error = "";
    busy = true;
    try {
      report = await TLSCheck(v);
    } catch (e) {
      report = null;
      error = String(e);
    } finally {
      busy = false;
    }
  }

  const ok = (r: any) => r && r.verdict === "Valid";

  function fmtDate(s: string): string {
    if (!s) return "—";
    const d = new Date(s);
    return isNaN(d.getTime()) ? "—" : d.toISOString().slice(0, 10);
  }

  // Plain-language gloss for the verdict banner.
  function verdictNote(r: any): string {
    switch (r?.verdict) {
      case "Valid":
        return "This certificate is trusted, matches the host, is in date, and uses modern crypto.";
      case "Expired":
        return "The certificate's validity period has passed — browsers will refuse it.";
      case "Not yet valid":
        return "The certificate's start date is in the future.";
      case "Self-signed":
        return "Signed by itself, not a recognised authority — browsers won't trust it without manual import.";
      case "Untrusted chain":
        return "The chain doesn't lead to a trusted root authority (missing intermediate, or a private CA).";
      case "Hostname mismatch":
        return "The certificate is valid but not issued for the name you connected to.";
      case "Weak key":
        return "The public key is too small (RSA < 2048) to be considered safe.";
      case "Weak signature":
        return "Signed with an obsolete algorithm (SHA-1/MD5) that is no longer safe.";
      default:
        return r?.verdict?.startsWith("Expiring")
          ? "Still valid, but renew soon — it expires within three weeks."
          : "";
    }
  }
</script>

<div class="tool">
  <form on:submit|preventDefault={run} class="row">
    <input
      class="field mono"
      bind:value={query}
      placeholder="cloudflare.com  ·  example.com:8443"
      autocomplete="off"
      spellcheck="false"
      aria-label="host or host:port"
    />
    <button class="go" disabled={busy}>{busy ? "Checking…" : "Check"}</button>
  </form>

  {#if error}<div class="hint">Couldn't connect: {error}</div>{/if}

  {#if report}
    <div class="verdict" class:good={ok(report)} class:bad={!ok(report)}>
      <span class="vlabel">{report.verdict}</span>
      <span class="vnote">{verdictNote(report)}</span>
    </div>

    <div class="cards">
      <div class="card span">
        <div class="k">Certificate</div>
        <dl>
          <dt>Subject</dt><dd class="mono">{report.subject || "—"}</dd>
          <dt>Issuer</dt><dd class="mono">{report.issuer || "—"}</dd>
          {#if report.sans?.length}
            <dt>Names (SAN)</dt>
            <dd class="chips">{#each report.sans as s}<span class="chip">{s}</span>{/each}</dd>
          {/if}
          {#if report.serial}<dt>Serial</dt><dd class="mono dim">{report.serial}</dd>{/if}
        </dl>
      </div>

      <div class="card">
        <div class="k">Validity &amp; keys</div>
        <dl>
          <dt>Valid from</dt><dd>{fmtDate(report.notBefore)}</dd>
          <dt>Valid until</dt>
          <dd>{fmtDate(report.notAfter)} <span class="dim">({report.daysToExpiry} days)</span></dd>
          <dt>Public key</dt><dd>{report.keyType} {report.keyBits}-bit</dd>
          <dt>Signature</dt><dd class="mono">{report.sigAlg}</dd>
          <dt>Connection</dt><dd>{report.tlsVersion}{#if report.cipherSuite} · <span class="dim">{report.cipherSuite}</span>{/if}</dd>
        </dl>
      </div>

      <div class="card">
        <div class="k">Trust signals</div>
        <div class="line wrap">
          <span class="pill {report.chainTrusted ? 'good' : 'bad'}">{report.chainTrusted ? "Chain trusted" : "Chain not trusted"}</span>
          <span class="pill {report.hostnameMatch ? 'good' : 'bad'}">{report.hostnameMatch ? "Hostname matches" : "Hostname mismatch"}</span>
          {#if report.selfSigned}<span class="pill bad">Self-signed</span>{/if}
        </div>
        <div class="cap">
          <b>Chain trusted</b> — a recognised authority vouches for it.
          <b>Hostname matches</b> — it was issued for the name you connected to.
        </div>
        {#if report.warnings?.length}
          <ul class="warns">
            {#each report.warnings as w}<li>{w}</li>{/each}
          </ul>
        {/if}
      </div>
    </div>
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 20px; }
  .row { display: flex; gap: 8px; max-width: 760px; }
  .field {
    flex: 1; background: var(--well); border: 1px solid var(--field-border); border-radius: 7px;
    padding: 9px 12px; color: var(--text); font-size: 0.9rem; outline: none;
  }
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
  .vlabel { font-weight: 600; color: var(--text-strong); }
  .verdict.good .vlabel { color: var(--good, #5ad17a); }
  .verdict.bad .vlabel { color: var(--bad, #ff6b6b); }
  .vnote { font-size: 0.82rem; color: var(--text-dim); }

  .cards { display: grid; grid-template-columns: 1fr 1fr; gap: 13px; }
  @media (max-width: 760px) { .cards { grid-template-columns: 1fr; } }
  .card { background: var(--card); border: 1px solid var(--border-soft); border-radius: 10px; padding: 13px 15px; }
  .card.span { grid-column: 1 / -1; }
  .k { font-size: 0.74rem; text-transform: uppercase; letter-spacing: 0.04em; color: var(--text-faint); margin-bottom: 9px; }
  dl { display: grid; grid-template-columns: max-content 1fr; gap: 5px 14px; margin: 0; }
  dt { color: var(--text-dim); font-size: 0.82rem; }
  dd { margin: 0; font-size: 0.86rem; color: var(--text); overflow-wrap: anywhere; }
  .dim { color: var(--text-dim); }
  .line { display: flex; gap: 6px; }
  .line.wrap { flex-wrap: wrap; }
  .chips { display: flex; flex-wrap: wrap; gap: 6px; }
  .chip {
    display: inline-flex; font-size: 0.76rem; padding: 2px 9px; border-radius: 7px;
    border: 1px solid var(--border-soft); background: var(--well);
  }
  .pill { display: inline-flex; font-size: 0.72rem; padding: 2px 9px; border-radius: 20px; border: 1px solid var(--border); color: var(--text-dim); }
  .pill.good { color: var(--good, #5ad17a); border-color: transparent; background: var(--good-bg, #102516); }
  .pill.bad { color: var(--bad, #ff6b6b); border-color: transparent; background: color-mix(in srgb, var(--bad, #ff6b6b) 12%, transparent); }
  .cap { margin-top: 9px; font-size: 0.76rem; color: var(--text-dim); line-height: 1.45; }
  .warns { margin: 9px 0 0; padding-left: 18px; color: var(--warn-text); font-size: 0.8rem; }
  .warns li { margin: 2px 0; }
</style>
