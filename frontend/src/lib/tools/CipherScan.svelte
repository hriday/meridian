<script lang="ts">
  import { TLSScan } from "../../../wailsjs/go/app/App.js";

  let host = "";
  let rep: any = null;
  let loading = false;
  let error = "";

  async function run() {
    const h = host.trim();
    if (!h) return;
    loading = true;
    error = "";
    rep = null;
    try {
      rep = await TLSScan(h);
    } catch (e: any) {
      error = String(e?.message ?? e);
    } finally {
      loading = false;
    }
  }

  const gradeClass = (g: string) => (["A", "B"].includes(g) ? "ok" : g === "C" || g === "D" ? "warn" : "bad");
</script>

<div class="tool">
  <p class="lede">
    Probe which <strong>TLS versions and cipher suites</strong> a host accepts, and grade the handshake — flagging
    legacy protocols (TLS 1.0/1.1) and weak ciphers (RC4, 3DES, CBC) that are still negotiable. This looks at the
    <em>handshake</em>; the <strong>TLS certificate</strong> and <strong>Cert expiry</strong> tools look at the cert.
  </p>

  <div class="row">
    <input class="field" bind:value={host} placeholder="example.com  or  host:8443" spellcheck="false" autocomplete="off"
      on:keydown={(e) => e.key === "Enter" && run()} />
    <button class="btn" on:click={run} disabled={loading || !host.trim()}>{loading ? "Scanning…" : "Scan"}</button>
  </div>

  {#if error}<p class="err">{error}</p>{/if}

  {#if rep}
    <div class="grade {gradeClass(rep.grade)}">
      <span class="gletter">{rep.grade}</span>
      <span class="gsum">{rep.summary}</span>
    </div>

    <div class="cols">
      <div class="col">
        <div class="h">Protocols</div>
        {#each rep.protocols as p}
          <div class="prow">
            <span class="pname">{p.name}</span>
            {#if p.supported}
              <span class="badge {p.legacy ? 'bad' : 'ok'}">{p.legacy ? "enabled — legacy" : "enabled"}</span>
            {:else}
              <span class="badge off">off</span>
            {/if}
          </div>
        {/each}
      </div>

      <div class="col">
        <div class="h">Cipher suites ({rep.ciphers.length})</div>
        {#each rep.ciphers as c}
          <div class="crow">
            <span class="badge {c.secure ? 'ok' : 'bad'}">{c.secure ? "secure" : "WEAK"}</span>
            <span class="cver">{c.version}</span>
            <code class="cname">{c.name}</code>
          </div>
        {/each}
      </div>
    </div>

    {#if rep.findings && rep.findings.length}
      <div class="findings">
        <div class="h">Findings</div>
        {#each rep.findings as f}<div class="finding">• {f}</div>{/each}
      </div>
    {:else}
      <p class="foot">No legacy protocols or weak ciphers negotiable — a clean handshake.</p>
    {/if}
    <p class="foot">Note: Go can't test SSLv3 or export/DES ciphers (removed from the standard library), so those aren't probed — absence here isn't proof they're disabled.</p>
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 30px; }
  .lede { max-width: 820px; margin: 0 0 14px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .row { display: flex; gap: 10px; max-width: 620px; }
  .field { background: var(--well); border: 1px solid var(--field-border, var(--border)); border-radius: 8px; padding: 9px 12px; color: var(--text); font-size: 0.86rem; outline: none; flex: 1 1 auto; min-width: 0; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .btn { background: var(--accent); color: var(--on-accent, #052e2b); border: 1px solid transparent; border-radius: 8px; padding: 9px 16px; font: inherit; font-size: 0.86rem; font-weight: 550; cursor: pointer; white-space: nowrap; }
  .btn:disabled { opacity: .5; cursor: default; }
  .err { color: var(--danger, #e5484d); font-size: 0.82rem; margin: 12px 0 0; }
  .grade { display: flex; align-items: center; gap: 14px; max-width: 820px; margin-top: 16px; padding: 12px 16px; border: 1px solid var(--border); border-radius: 11px; background: var(--card); }
  .gletter { font-size: 2rem; font-weight: 800; line-height: 1; width: 44px; text-align: center; border-radius: 8px; padding: 6px 0; }
  .grade.ok .gletter { color: #04260f; background: var(--good, #3fb950); }
  .grade.warn .gletter { color: #3a2a00; background: var(--warn, #f5a623); }
  .grade.bad .gletter { color: #fff; background: var(--danger, #e5484d); }
  .gsum { font-size: 0.84rem; color: var(--text-dim); }
  .cols { display: flex; gap: 14px; max-width: 820px; margin-top: 14px; flex-wrap: wrap; }
  .col { flex: 1 1 300px; min-width: 0; border: 1px solid var(--border); border-radius: 11px; padding: 10px 12px; background: var(--card); }
  .h { font-size: 0.72rem; text-transform: uppercase; letter-spacing: .04em; color: var(--text-dim); font-weight: 650; margin-bottom: 8px; }
  .prow { display: flex; align-items: center; justify-content: space-between; padding: 5px 0; }
  .pname { font-size: 0.84rem; color: var(--text); }
  .crow { display: flex; align-items: center; gap: 8px; padding: 4px 0; flex-wrap: wrap; }
  .cver { font-size: 0.72rem; color: var(--text-dim); }
  .cname { font-family: var(--font-mono, ui-monospace, monospace); font-size: 0.72rem; color: var(--text); word-break: break-all; }
  .badge { font-size: 0.68rem; font-weight: 700; text-transform: uppercase; letter-spacing: .03em; padding: 2px 8px; border-radius: 6px; white-space: nowrap; }
  .badge.ok { background: var(--good, #3fb950); color: #04260f; }
  .badge.bad { background: var(--danger, #e5484d); color: #fff; }
  .badge.off { background: var(--well); color: var(--text-dim); border: 1px solid var(--border); }
  .findings { max-width: 820px; margin-top: 14px; border: 1px solid var(--border); border-left: 3px solid var(--danger, #e5484d); border-radius: 10px; padding: 10px 14px; background: var(--card); }
  .finding { font-size: 0.82rem; color: var(--text); line-height: 1.6; }
  .foot { max-width: 820px; font-size: 0.76rem; color: var(--text-faint, var(--text-dim)); margin: 12px 0 0; line-height: 1.6; }
</style>
