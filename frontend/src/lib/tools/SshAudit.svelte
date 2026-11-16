<script lang="ts">
  import { SSHAudit } from "../../../wailsjs/go/app/App.js";
  import ToolHelp from "../ToolHelp.svelte";

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
      rep = await SSHAudit(h);
    } catch (e: any) {
      error = String(e?.message ?? e);
    } finally {
      loading = false;
    }
  }

  const gradeClass = (g: string) => (["A", "B"].includes(g) ? "ok" : g === "C" || g === "D" ? "warn" : "bad");
  const pqClass = (pq: any) => (pq?.preferred ? "ok" : pq?.supported ? "warn" : "bad");
  const pqLabel = (pq: any) => (pq?.preferred ? "Quantum-safe (default)" : pq?.supported ? "Quantum-safe (offered)" : "Not quantum-safe");
</script>

<div class="tool">
  <p class="lede">
    Inspect an SSH server's <strong>offered algorithms and host keys</strong> without logging in, and grade them — with
    a focus on <strong>post-quantum readiness</strong>. The threat is <em>harvest-now-decrypt-later</em>: traffic
    recorded today is decrypted once a quantum computer exists, so what matters is a <strong>post-quantum key
    exchange</strong> (sntrup761x25519 or ML-KEM). Nothing is authenticated and nothing is stored — the algorithm list
    comes from the server's cleartext KEXINIT, and host keys from handshakes aborted the instant the key arrives.
  </p>

  <ToolHelp>
    The letter grade is <strong>weakest-link</strong>: it reflects the weakest algorithm the server still offers across
    its <strong>ciphers, MACs, key exchanges and host-key signatures</strong> — not the key exchange alone.
    <code>A</code> = clean and a post-quantum key exchange; <code>B</code> = clean but no PQ; <code>C</code> = one
    <em>moderate</em> weakness (a legacy/SHA-1 MAC, a SHA-1 key exchange, or <code>ssh-rsa</code> with no
    <code>rsa-sha2</code>); <code>F</code> = something broken. So a server with an excellent key exchange can still land
    at <code>C</code> over a single legacy MAC — the <strong>Findings</strong> list names exactly what capped it.
    The <strong>post-quantum panel</strong> is the separate PQ signal: green <code>Quantum-safe (default)</code> means a
    PQ exchange (<code>sntrup761x25519</code> or ML-KEM) is preferred, amber means offered-but-not-default, red means none
    (open to <strong>harvest-now-decrypt-later</strong>). Classical RSA/ECDSA/Ed25519 host-key signatures are expected and
    not a problem.
  </ToolHelp>

  <div class="row">
    <input class="field" bind:value={host} placeholder="ssh.example.com  or  host:2222" spellcheck="false" autocomplete="off"
      on:keydown={(e) => e.key === "Enter" && run()} />
    <button class="btn" on:click={run} disabled={loading || !host.trim()}>{loading ? "Auditing…" : "Audit"}</button>
  </div>

  {#if error}<p class="err">{error}</p>{/if}

  {#if rep}
    <div class="grade {gradeClass(rep.grade)}">
      <span class="gletter">{rep.grade}</span>
      <span class="gsum">{rep.summary}</span>
    </div>

    <div class="pq {pqClass(rep.pq)}">
      <div class="pqhead"><span class="pqbadge">{pqLabel(rep.pq)}</span><span class="pqtitle">Post-quantum key exchange</span></div>
      <p class="pqbody">{rep.pq.verdict}</p>
      {#if rep.pq.algorithms && rep.pq.algorithms.length}
        <div class="pqalgos">{#each rep.pq.algorithms as a}<code>{a}</code>{/each}</div>
      {/if}
    </div>

    <div class="cols">
      <div class="col">
        <div class="h">Host keys ({rep.hostKeys ? rep.hostKeys.length : 0})</div>
        {#if rep.hostKeys && rep.hostKeys.length}
          {#each rep.hostKeys as k}
            <div class="krow">
              <span class="ktype">{k.type}{#if k.bits} · {k.bits}-bit{/if}</span>
              <code class="kfp">{k.fingerprint}</code>
            </div>
          {/each}
        {:else}
          <p class="foot">No host-key fingerprints captured (the server may use a key type this build can't request).</p>
        {/if}
      </div>

      <div class="col">
        <div class="h">Algorithms</div>
        <div class="alg"><span class="al">Key exchange</span><span class="av">{rep.kex.join(", ")}</span></div>
        <div class="alg"><span class="al">Ciphers</span><span class="av">{rep.ciphers.join(", ")}</span></div>
        <div class="alg"><span class="al">MACs</span><span class="av">{rep.macs.join(", ")}</span></div>
        <div class="alg"><span class="al">Host-key algos</span><span class="av">{rep.hostKeyAlgos.join(", ")}</span></div>
      </div>
    </div>

    {#if rep.findings && rep.findings.length}
      <div class="findings">
        <div class="h">Findings</div>
        {#each rep.findings as f}<div class="finding">• {f}</div>{/each}
      </div>
    {:else}
      <p class="foot">No weak algorithms, and a post-quantum key exchange is in use — a clean, future-proof configuration.</p>
    {/if}
    {#if rep.remediation && rep.remediation.length}
      <div class="improve">
        <div class="h">How to improve — toward grade A</div>
        {#each rep.remediation as s}<pre class="fix">{s}</pre>{/each}
      </div>
    {/if}
    <p class="foot">
      Authentication (host-key signatures) is still classical RSA/ECDSA/Ed25519 across the board — post-quantum
      signature algorithms are standardized but not yet fielded in SSH, so that isn't a finding. The grade is
      <em>weakest-link</em> across the ciphers, MACs, key exchange and host-key signatures, so a single legacy algorithm
      can cap it — the <em>Findings</em> above name what did.
    </p>
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
  .pq { max-width: 820px; margin-top: 12px; padding: 12px 16px; border: 1px solid var(--border); border-left: 3px solid var(--border); border-radius: 11px; background: var(--card); }
  .pq.ok { border-left-color: var(--good, #3fb950); }
  .pq.warn { border-left-color: var(--warn, #f5a623); }
  .pq.bad { border-left-color: var(--danger, #e5484d); }
  .pqhead { display: flex; align-items: center; gap: 10px; margin-bottom: 6px; }
  .pqtitle { font-size: 0.72rem; text-transform: uppercase; letter-spacing: .04em; color: var(--text-dim); font-weight: 650; }
  .pqbadge { font-size: 0.72rem; font-weight: 700; padding: 3px 9px; border-radius: 6px; white-space: nowrap; }
  .pq.ok .pqbadge { background: var(--good, #3fb950); color: #04260f; }
  .pq.warn .pqbadge { background: var(--warn, #f5a623); color: #3a2a00; }
  .pq.bad .pqbadge { background: var(--danger, #e5484d); color: #fff; }
  .pqbody { font-size: 0.84rem; color: var(--text); line-height: 1.55; margin: 0; }
  .pqalgos { margin-top: 8px; display: flex; gap: 8px; flex-wrap: wrap; }
  .pqalgos code { font-family: var(--font-mono, ui-monospace, monospace); font-size: 0.72rem; background: var(--well); border: 1px solid var(--border); border-radius: 6px; padding: 2px 7px; color: var(--text); }
  .cols { display: flex; gap: 14px; max-width: 820px; margin-top: 14px; flex-wrap: wrap; }
  .col { flex: 1 1 320px; min-width: 0; border: 1px solid var(--border); border-radius: 11px; padding: 10px 12px; background: var(--card); }
  .h { font-size: 0.72rem; text-transform: uppercase; letter-spacing: .04em; color: var(--text-dim); font-weight: 650; margin-bottom: 8px; }
  .krow { display: flex; flex-direction: column; gap: 2px; padding: 5px 0; border-top: 1px solid var(--border); }
  .krow:first-of-type { border-top: none; }
  .ktype { font-size: 0.78rem; color: var(--text); }
  .kfp { font-family: var(--font-mono, ui-monospace, monospace); font-size: 0.72rem; color: var(--text-dim); word-break: break-all; }
  .alg { display: flex; flex-direction: column; gap: 2px; padding: 5px 0; border-top: 1px solid var(--border); }
  .alg:first-of-type { border-top: none; }
  .al { font-size: 0.68rem; text-transform: uppercase; letter-spacing: .03em; color: var(--text-dim); font-weight: 650; }
  .av { font-family: var(--font-mono, ui-monospace, monospace); font-size: 0.72rem; color: var(--text); word-break: break-word; line-height: 1.5; }
  .findings { max-width: 820px; margin-top: 14px; border: 1px solid var(--border); border-left: 3px solid var(--warn, #f5a623); border-radius: 10px; padding: 10px 14px; background: var(--card); }
  .finding { font-size: 0.82rem; color: var(--text); line-height: 1.6; }
  .improve { max-width: 820px; margin-top: 12px; border: 1px solid var(--border); border-left: 3px solid var(--accent); border-radius: 10px; padding: 10px 14px; background: var(--card); }
  .improve .h { font-size: 0.7rem; text-transform: uppercase; letter-spacing: .05em; color: var(--text-dim); font-weight: 600; margin-bottom: 6px; }
  .fix { font-family: var(--font-mono, ui-monospace, monospace); font-size: 0.76rem; color: var(--text); background: var(--well); border-radius: 6px; padding: 7px 10px; margin: 6px 0 0; white-space: pre-wrap; word-break: break-word; line-height: 1.5; }
  .foot { max-width: 820px; font-size: 0.76rem; color: var(--text-faint, var(--text-dim)); margin: 12px 0 0; line-height: 1.6; }
</style>
