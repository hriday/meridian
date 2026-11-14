<script lang="ts">
  import { CertBegin, CertFinalize, CertCancel } from "../../../wailsjs/go/app/App.js";

  let domains = "";
  let email = "";
  let staging = true; // default to the test CA so a fumbled record doesn't burn prod rate limits

  type Step = "form" | "challenge" | "issuing" | "done";
  let step: Step = "form";
  let error = "";
  let orderId = "";
  let records: any[] = [];
  let result: any = null;
  let wasStaging = false;
  let flash = "";

  async function begin() {
    if (!domains.trim()) return;
    error = "";
    step = "issuing";
    try {
      const r = await CertBegin(email.trim(), domains, staging);
      orderId = r.orderId;
      records = r.records || [];
      wasStaging = r.staging;
      step = records.length ? "challenge" : "form";
      if (!records.length) error = "Nothing to validate — the domains may already be authorized.";
    } catch (e: any) {
      error = String(e?.message ?? e);
      step = "form";
    }
  }

  async function issue() {
    error = "";
    step = "issuing";
    try {
      result = await CertFinalize(orderId);
      step = "done";
    } catch (e: any) {
      error = String(e?.message ?? e);
      step = "challenge"; // let them fix the record and retry
    }
  }

  function cancel() {
    if (orderId) CertCancel(orderId).catch(() => {});
    orderId = "";
    records = [];
    result = null;
    error = "";
    step = "form";
  }

  async function copy(text: string, which: string) {
    try {
      await navigator.clipboard.writeText(text);
      flash = which;
      setTimeout(() => (flash = flash === which ? "" : flash), 1200);
    } catch {}
  }

  function download(filename: string, text: string) {
    try {
      const blob = new Blob([text], { type: "application/x-pem-file" });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(url);
    } catch {}
  }
</script>

<div class="tool">
  <p class="lede">
    Get a <strong>Let's Encrypt</strong> certificate via the <strong>DNS-01</strong> challenge — the firewall-friendly
    method. The CA reads a TXT record from public DNS and never connects back to your host, so it works for
    <strong>internal / firewalled</strong> servers (and wildcards). One-shot: you get the cert and key here, then
    deploy and renew them yourself — Meridian doesn't store or auto-renew.
  </p>

  {#if step === "form" || step === "issuing"}
    <div class="form">
      <label class="fl">Domain(s)
        <input class="field mono" bind:value={domains} placeholder="example.com, www.example.com  (or *.internal.lan)" spellcheck="false" />
      </label>
      <label class="fl">Email <span class="opt">(optional — expiry notices)</span>
        <input class="field" bind:value={email} placeholder="you@example.com" spellcheck="false" />
      </label>
      <label class="check">
        <input type="checkbox" bind:checked={staging} />
        <span>Use the <strong>staging</strong> CA (test cert, not trusted by browsers) — uncheck for a real, trusted certificate</span>
      </label>
      <button class="btn" on:click={begin} disabled={step === "issuing" || !domains.trim()}>
        {step === "issuing" ? "Contacting Let's Encrypt…" : "Request certificate"}
      </button>
      {#if !staging}<p class="warn">Production has rate limits (5 failures/hour per host). Test with staging first if you're unsure about the DNS record.</p>{/if}
    </div>
  {/if}

  {#if error}<p class="err">{error}</p>{/if}

  {#if step === "challenge"}
    <div class="card">
      <div class="ch-head">Publish {records.length === 1 ? "this TXT record" : "these TXT records"} at your DNS</div>
      {#each records as r}
        <div class="rec">
          <div class="rrow"><span class="rk">Type</span><code class="rv">TXT</code></div>
          <div class="rrow"><span class="rk">Name</span><code class="rv">{r.name}</code><button class="mini" on:click={() => copy(r.name, "n" + r.domain)}>{flash === "n" + r.domain ? "✓" : "copy"}</button></div>
          <div class="rrow"><span class="rk">Value</span><code class="rv">{r.value}</code><button class="mini" on:click={() => copy(r.value, "v" + r.domain)}>{flash === "v" + r.domain ? "✓" : "copy"}</button></div>
        </div>
      {/each}
      <p class="hint">Add them, then give DNS a minute to propagate (check with <span class="mono">Meridian → DNS</span> or <span class="mono">dig TXT {records[0]?.name}</span>) before verifying.</p>
      <div class="actions">
        <button class="btn" on:click={issue}>Verify &amp; issue</button>
        <button class="btn ghost" on:click={cancel}>Cancel</button>
      </div>
    </div>
  {/if}

  {#if step === "issuing" && orderId}
    <p class="hint">Waiting for Let's Encrypt to validate the record…</p>
  {/if}

  {#if step === "done" && result}
    <div class="card done">
      <div class="ch-head">
        {wasStaging ? "Test certificate issued (staging — not trusted)" : "Certificate issued"} for {result.domains.join(", ")}
      </div>

      <div class="pem">
        <div class="pem-head"><span>fullchain.pem</span>
          <span class="pem-btns">
            <button class="mini" on:click={() => copy(result.fullchain, "fc")}>{flash === "fc" ? "✓ copied" : "copy"}</button>
            <button class="mini" on:click={() => download("fullchain.pem", result.fullchain)}>download</button>
          </span>
        </div>
        <pre class="mono">{result.fullchain}</pre>
      </div>

      <div class="pem key">
        <div class="pem-head"><span>privkey.pem <span class="secret">keep secret</span></span>
          <span class="pem-btns">
            <button class="mini" on:click={() => copy(result.key, "pk")}>{flash === "pk" ? "✓ copied" : "copy"}</button>
            <button class="mini" on:click={() => download("privkey.pem", result.key)}>download</button>
          </span>
        </div>
        <pre class="mono">{result.key}</pre>
      </div>

      <p class="hint">
        Install <span class="mono">fullchain.pem</span> + <span class="mono">privkey.pem</span> on your server. The cert
        is valid ~90 days — set a reminder to re-run this before it expires (check any time with the
        <strong>Cert expiry</strong> tool). The private key is shown only here and never leaves your machine or hits a log.
      </p>
      <button class="btn ghost" on:click={cancel}>Get another</button>
    </div>
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 30px; max-width: 860px; }
  .lede { margin: 0 0 14px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .mono { font-family: var(--font-mono, ui-monospace, monospace); }
  .form { display: flex; flex-direction: column; gap: 12px; max-width: 620px; }
  .fl { display: flex; flex-direction: column; gap: 5px; font-size: 0.8rem; color: var(--text-dim); }
  .opt { color: var(--text-faint, var(--text-dim)); font-weight: 400; }
  .field { background: var(--well); border: 1px solid var(--field-border, var(--border)); border-radius: 8px; padding: 9px 12px; color: var(--text); font-size: 0.86rem; outline: none; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .check { display: flex; gap: 8px; align-items: flex-start; font-size: 0.8rem; color: var(--text-dim); line-height: 1.4; cursor: pointer; }
  .check input { margin-top: 2px; }
  .btn { background: var(--accent); color: var(--on-accent, #052e2b); border: 1px solid transparent; border-radius: 8px; padding: 9px 16px; font: inherit; font-size: 0.86rem; font-weight: 550; cursor: pointer; align-self: flex-start; }
  .btn:disabled { opacity: .5; cursor: default; }
  .btn.ghost { background: var(--ghost-bg, var(--well)); color: var(--text); border-color: var(--border); }
  .warn { font-size: 0.76rem; color: var(--warn-text, #b5820f); margin: 2px 0 0; }
  .err { color: var(--danger, #e5484d); font-size: 0.82rem; margin: 12px 0 0; max-width: 620px; line-height: 1.5; }
  .card { background: var(--card); border: 1px solid var(--border); border-radius: 11px; padding: 14px 16px; margin-top: 16px; }
  .ch-head { font-weight: 600; font-size: 0.9rem; color: var(--text); margin-bottom: 10px; }
  .rec { border: 1px solid var(--border-soft, var(--border)); border-radius: 9px; padding: 8px 10px; margin-bottom: 8px; }
  .rrow { display: flex; align-items: center; gap: 10px; padding: 3px 0; }
  .rk { width: 48px; font-size: 0.72rem; text-transform: uppercase; letter-spacing: .04em; color: var(--text-faint, var(--text-dim)); flex: none; }
  .rv { font-family: var(--font-mono, ui-monospace, monospace); font-size: 0.78rem; color: var(--text); word-break: break-all; flex: 1 1 auto; }
  .mini { font-size: 0.7rem; background: var(--well); border: 1px solid var(--border); border-radius: 6px; padding: 2px 8px; color: var(--text-dim); cursor: pointer; white-space: nowrap; flex: none; }
  .mini:hover { border-color: var(--accent); color: var(--text); }
  .hint { font-size: 0.78rem; color: var(--text-dim); line-height: 1.6; margin: 10px 0 0; }
  .actions { display: flex; gap: 10px; margin-top: 12px; }
  .pem { margin-top: 12px; }
  .pem-head { display: flex; justify-content: space-between; align-items: center; font-size: 0.76rem; color: var(--text-dim); margin-bottom: 4px; font-family: var(--font-mono, ui-monospace, monospace); }
  .pem-btns { display: flex; gap: 6px; }
  .secret { color: var(--danger, #e5484d); font-weight: 700; text-transform: uppercase; font-size: 0.64rem; letter-spacing: .05em; margin-left: 6px; }
  pre { background: var(--well); border: 1px solid var(--border); border-radius: 8px; padding: 9px 11px; font-size: 0.72rem; color: var(--text); overflow-x: auto; margin: 0; white-space: pre; }
  .pem.key pre { border-color: color-mix(in srgb, var(--danger, #e5484d) 40%, var(--border)); }
</style>
