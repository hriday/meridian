<script lang="ts">
  import { JWTDecode, JWTVerify } from "../../../wailsjs/go/app/App.js";
  import { get } from "svelte/store";
  import { requestedTool } from "../nav";

  let token = "";
  const _req = get(requestedTool);
  if (_req && _req.tool === "jwt" && _req.value) { token = _req.value; requestedTool.set(null); }
  let decoded: any = null;
  let decodeErr = "";

  let key = "";
  let verifyRes: any = null;
  let verifyErr = "";
  let showVerify = false;

  // Decode live as the token changes.
  $: decode(token);
  let decodeSeq = 0;
  async function decode(t: string) {
    const seq = ++decodeSeq;
    verifyRes = null;
    verifyErr = "";
    const s = t.trim();
    if (!s) { decoded = null; decodeErr = ""; return; }
    try {
      const d = await JWTDecode(s);
      if (seq === decodeSeq) { decoded = d; decodeErr = ""; }
    } catch (e: any) {
      if (seq === decodeSeq) { decoded = null; decodeErr = String(e?.message ?? e); }
    }
  }

  async function verify() {
    verifyErr = "";
    verifyRes = null;
    try {
      verifyRes = await JWTVerify(token.trim(), key);
    } catch (e: any) {
      verifyErr = String(e?.message ?? e);
    }
  }

  const fmt = (iso: string) => (iso ? new Date(iso).toLocaleString() : "");
</script>

<div class="tool">
  <h2>JWT decoder &amp; verifier</h2>
  <p class="sub">
    Decode and verify a JSON Web Token <strong>entirely on your machine</strong> — unlike pasting it into a website, the
    token (a bearer credential) never leaves this computer. See the header, claims, and expiry, and optionally check the
    signature. Nothing is stored.
  </p>

  <textarea class="field mono tok" bind:value={token} rows="4" spellcheck="false"
    placeholder="Paste a JWT (eyJhbGciOi…)"></textarea>

  {#if decodeErr}<div class="hint">{decodeErr}</div>{/if}

  {#if decoded}
    {#if decoded.warnings && decoded.warnings.length}
      <div class="warns">
        {#each decoded.warnings as w}<div class="warn">⚠ {w}</div>{/each}
      </div>
    {/if}

    <div class="claims">
      <div class="k">Algorithm</div><div class="v mono">{decoded.alg}{#if decoded.typ} · {decoded.typ}{/if}</div>
      {#if decoded.kid}<div class="k">Key ID</div><div class="v mono">{decoded.kid}</div>{/if}
      {#if decoded.issuer}<div class="k">Issuer</div><div class="v mono">{decoded.issuer}</div>{/if}
      {#if decoded.subject}<div class="k">Subject</div><div class="v mono">{decoded.subject}</div>{/if}
      {#if decoded.audience && decoded.audience.length}<div class="k">Audience</div><div class="v mono">{decoded.audience.join(", ")}</div>{/if}
      {#if decoded.issuedAt}<div class="k">Issued</div><div class="v">{fmt(decoded.issuedAt)}</div>{/if}
      {#if decoded.notBefore}<div class="k">Not before</div><div class="v">{fmt(decoded.notBefore)} {#if decoded.notYetValid}<span class="badge bad">not yet valid</span>{/if}</div>{/if}
      {#if decoded.expiresAt}<div class="k">Expires</div><div class="v">{fmt(decoded.expiresAt)} {#if decoded.expired}<span class="badge bad">expired</span>{:else}<span class="badge ok">valid</span>{/if}</div>{/if}
    </div>

    <div class="cols">
      <div class="seg"><div class="segh">Header</div><pre class="mono box">{decoded.headerJSON}</pre></div>
      <div class="seg"><div class="segh">Payload</div><pre class="mono box">{decoded.payloadJSON}</pre></div>
    </div>

    <div class="verifybox">
      <button class="disclose" on:click={() => (showVerify = !showVerify)}>{showVerify ? "▾" : "▸"} Verify signature</button>
      {#if showVerify}
        <p class="vhint">For <span class="mono">HS*</span> paste the shared secret. For <span class="mono">RS* / ES* / PS* / EdDSA</span> paste a PEM public key (<span class="mono">-----BEGIN PUBLIC KEY-----</span>) or certificate.</p>
        <textarea class="field mono key" bind:value={key} rows="3" spellcheck="false" placeholder="HMAC secret, or -----BEGIN PUBLIC KEY-----"></textarea>
        <button class="go" on:click={verify} disabled={!token.trim() || !key.trim()}>Verify</button>
        {#if verifyErr}<div class="hint">{verifyErr}</div>{/if}
        {#if verifyRes}
          <div class="vres {verifyRes.valid ? 'ok' : 'bad'}">
            <span class="vmark">{verifyRes.valid ? "✓" : "✗"}</span>
            <span>{verifyRes.valid ? "Signature valid" : "Signature invalid"} <span class="dim">({verifyRes.algorithm})</span> — {verifyRes.reason}</span>
          </div>
        {/if}
      {/if}
    </div>
  {/if}
</div>

<style>
  .tool { padding: 16px 20px 40px; max-width: 900px; }
  h2 { margin: 0 0 2px; font-size: 1.15rem; font-weight: 600; color: var(--text-strong); }
  .sub { margin: 0 0 14px; color: var(--text-dim); font-size: 0.86rem; line-height: 1.5; }
  .field { background: var(--well); border: 1px solid var(--field-border); border-radius: 8px; padding: 9px 12px; color: var(--text); font-size: 0.85rem; outline: none; width: 100%; resize: vertical; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .tok { word-break: break-all; }
  .hint { background: var(--warn-bg); border: 1px solid var(--warn-border); color: var(--warn-text); border-radius: 7px; padding: 9px 12px; font-size: 0.82rem; margin: 10px 0; }
  .warns { margin: 12px 0; display: flex; flex-direction: column; gap: 6px; }
  .warn { background: color-mix(in srgb, var(--danger, #e5484d) 12%, transparent); border: 1px solid color-mix(in srgb, var(--danger, #e5484d) 40%, transparent); color: var(--danger, #e5484d); border-radius: 7px; padding: 7px 11px; font-size: 0.82rem; }
  .claims { display: grid; grid-template-columns: max-content 1fr; gap: 5px 16px; margin: 14px 0; font-size: 0.84rem; align-items: baseline; }
  .claims .k { color: var(--text-dim); white-space: nowrap; } .claims .v { color: var(--text); }
  .badge { font-size: 0.68rem; font-weight: 650; padding: 1px 7px; border-radius: 6px; margin-left: 6px; }
  .badge.ok { background: color-mix(in srgb, var(--good, #3fb950) 18%, transparent); color: var(--good, #3fb950); }
  .badge.bad { background: var(--danger, #e5484d); color: #fff; }
  .cols { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; margin-bottom: 14px; }
  @media (max-width: 680px) { .cols { grid-template-columns: 1fr; } }
  .segh { font-size: 0.68rem; text-transform: uppercase; letter-spacing: .04em; color: var(--text-faint, var(--text-dim)); font-weight: 650; margin-bottom: 5px; }
  .box { background: var(--well); border: 1px solid var(--border); border-radius: 8px; padding: 11px 13px; font-size: 0.8rem; color: var(--text); margin: 0; overflow-x: auto; white-space: pre; }
  .verifybox { border-top: 1px solid var(--border); padding-top: 12px; }
  .disclose { background: none; border: none; color: var(--accent); font: inherit; font-size: 0.86rem; cursor: pointer; padding: 0; }
  .vhint { font-size: 0.78rem; color: var(--text-dim); margin: 8px 0; line-height: 1.5; }
  .key { margin-bottom: 8px; }
  .go { background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px; padding: 8px 16px; font: inherit; font-size: 0.86rem; cursor: pointer; }
  .go:disabled { opacity: .5; cursor: default; }
  .vres { display: flex; align-items: center; gap: 9px; margin-top: 11px; padding: 10px 13px; border-radius: 8px; font-size: 0.85rem; border: 1px solid var(--border); border-left-width: 3px; }
  .vres.ok { border-left-color: var(--good, #3fb950); color: var(--text); }
  .vres.bad { border-left-color: var(--danger, #e5484d); color: var(--text); }
  .vmark { font-size: 1.1rem; font-weight: 700; }
  .vres.ok .vmark { color: var(--good, #3fb950); } .vres.bad .vmark { color: var(--danger, #e5484d); }
  .dim { color: var(--text-dim); }
  .mono { font-family: var(--font-mono); }
</style>
