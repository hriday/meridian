<script lang="ts">
  import { CertDecode } from "../../../wailsjs/go/app/App.js";
  import { get } from "svelte/store";
  import { requestedTool } from "../nav";

  let pem = "";
  const _req = get(requestedTool);
  if (_req && _req.tool === "certinfo" && _req.value) { pem = _req.value; requestedTool.set(null); }
  let result: any = null;
  let error = "";

  $: decode(pem);
  let seq = 0;
  async function decode(t: string) {
    const n = ++seq;
    const s = t.trim();
    if (!s) { result = null; error = ""; return; }
    try {
      const r = await CertDecode(s);
      if (n === seq) { result = r; error = ""; }
    } catch (e: any) {
      if (n === seq) { result = null; error = String(e?.message ?? e); }
    }
  }

  const fmt = (iso: string) => (iso ? new Date(iso).toLocaleDateString() : "");
</script>

<div class="tool">
  <h2>Certificate / CSR decoder</h2>
  <p class="sub">
    Paste a PEM <strong>certificate</strong> or <strong>CSR</strong> to read what's inside — <strong>locally</strong>, so an
    internal cert or a CSR never gets pasted into a web decoder. Handles a full chain; private-key blocks are ignored.
    Nothing is stored.
  </p>

  <textarea class="field mono" bind:value={pem} rows="6" spellcheck="false"
    placeholder="-----BEGIN CERTIFICATE-----&#10;…&#10;-----END CERTIFICATE-----"></textarea>

  {#if error}<div class="hint">{error}</div>{/if}

  {#if result}
    {#each result.items as it}
      <div class="card {it.expired ? 'bad' : ''}">
        <div class="chead">
          <span class="kind">{it.kind === "csr" ? "CSR" : "Certificate"}</span>
          {#if it.selfSigned}<span class="tag">self-signed</span>{/if}
          {#if it.isCA}<span class="tag ca">CA</span>{/if}
          {#if it.expired}<span class="tag bad">expired</span>{:else if it.notAfter}<span class="tag ok">valid</span>{/if}
        </div>
        <div class="grid">
          <div class="k">Subject</div><div class="v mono">{it.subject}</div>
          {#if it.issuer}<div class="k">Issuer</div><div class="v mono">{it.issuer}</div>{/if}
          <div class="k">Key</div><div class="v mono">{it.keyType}</div>
          <div class="k">Signature</div><div class="v mono">{it.sigAlg}</div>
          {#if it.serialHex}<div class="k">Serial</div><div class="v mono sm">{it.serialHex}</div>{/if}
          {#if it.notBefore}<div class="k">Valid from</div><div class="v">{fmt(it.notBefore)}</div>{/if}
          {#if it.notAfter}<div class="k">Valid until</div><div class="v">{fmt(it.notAfter)} <span class="dim">({it.daysLeft} days)</span></div>{/if}
          {#if it.dnsNames && it.dnsNames.length}<div class="k">DNS names</div><div class="v mono">{it.dnsNames.join(", ")}</div>{/if}
          {#if it.ips && it.ips.length}<div class="k">IP SANs</div><div class="v mono">{it.ips.join(", ")}</div>{/if}
          {#if it.emails && it.emails.length}<div class="k">Emails</div><div class="v mono">{it.emails.join(", ")}</div>{/if}
          {#if it.uris && it.uris.length}<div class="k">URIs</div><div class="v mono">{it.uris.join(", ")}</div>{/if}
          {#if it.keyUsage && it.keyUsage.length}<div class="k">Key usage</div><div class="v">{it.keyUsage.join(", ")}</div>{/if}
          {#if it.extKeyUsage && it.extKeyUsage.length}<div class="k">Ext key usage</div><div class="v">{it.extKeyUsage.join(", ")}</div>{/if}
        </div>
      </div>
    {/each}
  {/if}
</div>

<style>
  .tool { padding: 16px 20px 40px; max-width: 900px; }
  h2 { margin: 0 0 2px; font-size: 1.15rem; font-weight: 600; color: var(--text-strong); }
  .sub { margin: 0 0 14px; color: var(--text-dim); font-size: 0.86rem; line-height: 1.5; }
  .field { background: var(--well); border: 1px solid var(--field-border); border-radius: 8px; padding: 9px 12px; color: var(--text); font-size: 0.82rem; outline: none; width: 100%; resize: vertical; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .hint { background: var(--warn-bg); border: 1px solid var(--warn-border); color: var(--warn-text); border-radius: 7px; padding: 9px 12px; font-size: 0.82rem; margin: 10px 0; }
  .card { border: 1px solid var(--border); border-radius: 11px; padding: 13px 16px; margin-top: 14px; background: var(--card); border-left-width: 3px; border-left-color: var(--good, #3fb950); }
  .card.bad { border-left-color: var(--danger, #e5484d); }
  .chead { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; margin-bottom: 11px; }
  .kind { font-size: 0.92rem; font-weight: 650; color: var(--text-strong); }
  .tag { font-size: 0.66rem; font-weight: 650; padding: 1px 8px; border-radius: 6px; background: var(--well); border: 1px solid var(--border); color: var(--text-dim); }
  .tag.ca { color: var(--accent); border-color: var(--accent); }
  .tag.ok { background: color-mix(in srgb, var(--good, #3fb950) 18%, transparent); color: var(--good, #3fb950); border: none; }
  .tag.bad { background: var(--danger, #e5484d); color: #fff; border: none; }
  .grid { display: grid; grid-template-columns: max-content 1fr; gap: 5px 16px; font-size: 0.84rem; align-items: baseline; }
  .k { color: var(--text-dim); white-space: nowrap; } .v { color: var(--text); word-break: break-word; }
  .v.sm { font-size: 0.74rem; }
  .dim { color: var(--text-dim); }
  .mono { font-family: var(--font-mono); }
</style>
