<script lang="ts">
  import { STUNDiscover } from "../../../wailsjs/go/app/App.js";
  import ToolHelp from "../ToolHelp.svelte";

  let server = "";
  let loading = false;
  let error = "";
  let res: any = null;

  async function run() {
    if (loading) return;
    loading = true; error = ""; res = null;
    try {
      res = await STUNDiscover(server.trim());
    } catch (e: any) {
      error = String(e?.message ?? e);
    } finally {
      loading = false;
    }
  }

  const symmetric = (r: any) => r && /symmetric/i.test(r.natType);
</script>

<div class="tool">
  <h2>STUN — public IP &amp; NAT type</h2>
  <p class="sub">
    Ask a STUN server what your <strong>public (reflexive) address</strong> is, and reason about the NAT in front of you —
    the key to diagnosing one-way audio and NAT-traversal problems in SIP &amp; WebRTC. Querying two servers flags a
    <strong>symmetric NAT</strong>, which breaks media without a relay. Nothing is stored.
  </p>

  <ToolHelp>
    The big value is your <strong>public (reflexive) address</strong> as the internet sees it. The verdict reads
    the NAT: a <strong>symmetric NAT</strong> (red) is the problem case — it breaks SIP/WebRTC media unless you
    add a TURN relay; a cone or no-NAT result (green/amber) traverses fine. Compare the public IP to your
    <strong>local IP</strong> to confirm whether you're behind NAT at all.
  </ToolHelp>

  <form on:submit|preventDefault={run} class="row">
    <input class="field mono" bind:value={server} placeholder="stun.l.google.com:19302  (blank = public defaults)" spellcheck="false" aria-label="STUN server" />
    <button class="go" disabled={loading}>{loading ? "Querying…" : "Discover"}</button>
  </form>

  {#if error}<div class="hint">{error}</div>{/if}

  {#if res}
    <div class="card {symmetric(res) ? 'bad' : res.nat ? 'warn' : 'ok'}">
      <div class="big mono">{res.publicIP}<span class="port">:{res.publicPort}</span></div>
      <div class="biglbl">your public address</div>
      <div class="grid">
        <div class="k">Local IP</div><div class="v mono">{res.localIP}</div>
        <div class="k">Behind NAT</div><div class="v">{res.nat ? "yes" : "no"}</div>
        <div class="k">STUN server</div><div class="v mono">{res.server} <span class="dim">({res.rttMillis} ms)</span></div>
      </div>
      <div class="assess {symmetric(res) ? 'bad' : res.nat ? 'warn' : 'ok'}">{res.natType}</div>
    </div>
  {/if}
</div>

<style>
  .tool { padding: 16px 20px 40px; max-width: 760px; }
  h2 { margin: 0 0 2px; font-size: 1.15rem; font-weight: 600; color: var(--text-strong); }
  .sub { margin: 0 0 15px; color: var(--text-dim); font-size: 0.86rem; line-height: 1.5; }
  .row { display: flex; gap: 8px; margin-bottom: 14px; flex-wrap: wrap; }
  .field { flex: 1; min-width: 220px; background: var(--well); border: 1px solid var(--field-border); border-radius: 7px; padding: 9px 12px; color: var(--text); font-size: 0.88rem; outline: none; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .go { background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px; padding: 0 18px; font: inherit; font-size: 0.9rem; cursor: pointer; }
  .go:disabled { opacity: .5; cursor: default; }
  .hint { background: var(--warn-bg); border: 1px solid var(--warn-border); color: var(--warn-text); border-radius: 7px; padding: 9px 12px; font-size: 0.85rem; margin-bottom: 12px; }
  .card { border: 1px solid var(--border); border-left-width: 4px; border-radius: 12px; padding: 16px 18px; margin-top: 8px; background: var(--card); }
  .card.ok { border-left-color: var(--good, #3fb950); } .card.warn { border-left-color: var(--warn-text, #f5a623); } .card.bad { border-left-color: var(--danger, #e5484d); }
  .big { font-size: 1.7rem; font-weight: 700; color: var(--text-strong); line-height: 1.1; }
  .port { color: var(--text-dim); font-weight: 400; }
  .biglbl { font-size: 0.74rem; color: var(--text-dim); margin: 2px 0 12px; }
  .grid { display: grid; grid-template-columns: max-content 1fr; gap: 5px 16px; font-size: 0.84rem; align-items: baseline; }
  .k { color: var(--text-dim); white-space: nowrap; } .v { color: var(--text); }
  .assess { margin-top: 12px; font-size: 0.85rem; padding: 9px 12px; border-radius: 8px; background: var(--well); color: var(--text); }
  .assess.ok { color: var(--good, #3fb950); } .assess.warn { color: var(--text); } .assess.bad { color: var(--danger, #e5484d); font-weight: 600; }
  .dim { color: var(--text-dim); }
  .mono { font-family: var(--font-mono); }
</style>
