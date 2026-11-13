<script lang="ts">
  import { EmailAuthCheck } from "../../../wailsjs/go/app/App.js";

  let domain = "";
  let selectors = "";
  let rep: any = null;
  let loading = false;
  let error = "";

  async function run() {
    const d = domain.trim();
    if (!d) return;
    loading = true;
    error = "";
    try {
      rep = await EmailAuthCheck(d, selectors.trim());
    } catch (e: any) {
      error = String(e?.message ?? e);
      rep = null;
    } finally {
      loading = false;
    }
  }

  const cls = (s: string) => (s === "pass" ? "ok" : s === "warn" ? "warn" : "bad");
  $: foundSelectors = rep ? (rep.dkim || []).filter((d: any) => d.found) : [];
  $: foundNames = foundSelectors.map((d: any) => d.selector).join(", ");
</script>

<div class="tool">
  <p class="lede">
    Check a domain's <strong>outbound-mail authentication</strong> — can it send authenticated mail, and how spoofable
    is it? Reads <strong>SPF</strong>, <strong>DMARC</strong>, and probes common <strong>DKIM</strong> selectors, all from
    DNS. For the broader DNS posture (nameservers, MX, DNSSEC) use <strong>DNS audit</strong>.
  </p>

  <div class="row">
    <input class="field" bind:value={domain} placeholder="example.com" spellcheck="false" autocomplete="off"
      on:keydown={(e) => e.key === "Enter" && run()} />
    <button class="btn" on:click={run} disabled={loading || !domain.trim()}>{loading ? "Checking…" : "Check"}</button>
  </div>
  <input class="field sub" bind:value={selectors} placeholder="DKIM selectors (optional, comma-separated — default: common set)" spellcheck="false" />

  {#if error}<p class="err">{error}</p>{/if}

  {#if rep}
    <div class="cards">
      <div class="card {cls(rep.spf.status)}">
        <div class="chead"><span class="mech">SPF</span><span class="badge {cls(rep.spf.status)}">{rep.spf.status}</span></div>
        <p class="note">{rep.spf.note}</p>
        {#if rep.spf.record}<code class="rec">{rep.spf.record}</code>{/if}
      </div>

      <div class="card {cls(rep.dmarc.status)}">
        <div class="chead"><span class="mech">DMARC</span><span class="badge {cls(rep.dmarc.status)}">{rep.dmarc.status}</span></div>
        <p class="note">{rep.dmarc.note}</p>
        {#if rep.dmarc.record}<code class="rec">{rep.dmarc.record}</code>{/if}
      </div>

      <div class="card {rep.dkimFound ? 'ok' : 'warn'}">
        <div class="chead"><span class="mech">DKIM</span><span class="badge {rep.dkimFound ? 'ok' : 'warn'}">{rep.dkimFound ? "found" : "not found"}</span></div>
        {#if rep.dkimFound}
          <p class="note">Key published for selector(s): <strong>{foundNames}</strong></p>
        {:else}
          <p class="note">No DKIM key found among the {rep.dkim.length} probed selectors. DKIM uses a secret selector name DNS won't list — if yours isn't in the default set, add it above.</p>
        {/if}
      </div>
    </div>
    <p class="foot">A domain with SPF <strong>-all</strong>, DMARC <strong>p=reject</strong>, and DKIM in place is hard to spoof. Anything at <em>warn</em> or <em>fail</em> leaves room for forged mail.</p>
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 30px; }
  .lede { max-width: 820px; margin: 0 0 14px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .row { display: flex; gap: 10px; max-width: 620px; }
  .field { background: var(--well); border: 1px solid var(--field-border, var(--border)); border-radius: 8px; padding: 9px 12px; color: var(--text); font-size: 0.86rem; outline: none; flex: 1 1 auto; min-width: 0; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .field.sub { max-width: 620px; margin-top: 8px; font-size: 0.8rem; box-sizing: border-box; width: 100%; }
  .btn { background: var(--accent); color: var(--on-accent, #052e2b); border: 1px solid transparent; border-radius: 8px; padding: 9px 16px; font: inherit; font-size: 0.86rem; font-weight: 550; cursor: pointer; white-space: nowrap; }
  .btn:disabled { opacity: .5; cursor: default; }
  .err { color: var(--danger, #e5484d); font-size: 0.82rem; margin: 12px 0 0; }
  .cards { display: flex; flex-direction: column; gap: 10px; max-width: 820px; margin-top: 16px; }
  .card { border: 1px solid var(--border); border-left-width: 3px; border-radius: 10px; padding: 11px 14px; background: var(--card); }
  .card.ok { border-left-color: var(--good, #3fb950); }
  .card.warn { border-left-color: var(--warn, #f5a623); }
  .card.bad { border-left-color: var(--danger, #e5484d); }
  .chead { display: flex; align-items: center; justify-content: space-between; margin-bottom: 6px; }
  .mech { font-weight: 650; font-size: 0.9rem; color: var(--text); letter-spacing: .02em; }
  .badge { font-size: 0.7rem; font-weight: 700; text-transform: uppercase; letter-spacing: .03em; padding: 2px 9px; border-radius: 6px; }
  .badge.ok { background: var(--good, #3fb950); color: #04260f; }
  .badge.warn { background: var(--warn, #f5a623); color: #3a2a00; }
  .badge.bad { background: var(--danger, #e5484d); color: #fff; }
  .note { margin: 0; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .rec { display: block; margin-top: 7px; font-family: var(--font-mono, ui-monospace, monospace); font-size: 0.76rem; color: var(--text); background: var(--well); border-radius: 6px; padding: 6px 9px; word-break: break-all; }
  .foot { max-width: 820px; font-size: 0.76rem; color: var(--text-faint, var(--text-dim)); margin: 12px 0 0; line-height: 1.6; }
</style>
