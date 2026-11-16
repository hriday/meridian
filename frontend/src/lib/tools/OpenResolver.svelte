<script lang="ts">
  import { OpenResolverTest } from "../../../wailsjs/go/app/App.js";
  import ToolHelp from "../ToolHelp.svelte";

  let resolver = "";
  let rep: any = null;
  let loading = false;
  let error = "";

  async function run() {
    const r = resolver.trim();
    if (!r) return;
    loading = true;
    error = "";
    try {
      rep = await OpenResolverTest(r);
    } catch (e: any) {
      error = String(e?.message ?? e);
      rep = null;
    } finally {
      loading = false;
    }
  }
</script>

<div class="tool">
  <p class="lede">
    Tests whether a DNS server will <strong>recurse for anyone</strong> — an "open resolver." Point it at a device that
    shouldn't be a public resolver (a router, a server, a printer): if it resolves a name it isn't authoritative for, it's
    an accidental open resolver, abusable for <strong>DNS amplification</strong> attacks. Your own configured resolver
    answering for you is normal.
  </p>

  <ToolHelp>
    You want <code>not open</code> (green) for any device that isn't meant to be a public resolver. A red
    <strong><code>OPEN</code></strong> verdict means it recursed for a name it isn't authoritative for — anyone on the
    internet can abuse it for <strong>DNS amplification</strong>, so restrict it to your own network. The facts below
    show the proof: <code>recursion available: yes</code> and <code>answered: yes</code> together are what make it open.
    Testing your own ISP or configured resolver answering you is expected and not a finding.
  </ToolHelp>

  <div class="row">
    <input class="field mono" bind:value={resolver} placeholder="192.168.1.1" spellcheck="false" autocomplete="off"
      on:keydown={(e) => e.key === "Enter" && run()} />
    <button class="btn" on:click={run} disabled={loading || !resolver.trim()}>{loading ? "Probing…" : "Test resolver"}</button>
  </div>

  {#if error}<p class="err">{error}</p>{/if}

  {#if rep}
    <div class="card {rep.open ? 'bad' : 'ok'}">
      <div class="chead">
        <span class="mono">{rep.resolver}</span>
        <span class="badge {rep.open ? 'bad' : 'ok'}">{rep.open ? "OPEN" : "not open"}</span>
      </div>
      <p class="note">{rep.summary}</p>
      <div class="facts">
        <span>probe <code>{rep.probe}</code></span>
        <span>rcode <code>{rep.rcode || "—"}</code></span>
        <span>recursion available <code>{rep.recursionAvailable ? "yes" : "no"}</code></span>
        <span>answered <code>{rep.answered ? "yes" : "no"}</code></span>
      </div>
    </div>
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 30px; }
  .lede { max-width: 820px; margin: 0 0 14px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .mono { font-family: var(--font-mono, ui-monospace, monospace); }
  .row { display: flex; gap: 10px; max-width: 620px; }
  .field { background: var(--well); border: 1px solid var(--field-border, var(--border)); border-radius: 8px; padding: 9px 12px; color: var(--text); font-size: 0.86rem; outline: none; flex: 1 1 auto; min-width: 0; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .btn { background: var(--accent); color: var(--on-accent, #052e2b); border: 1px solid transparent; border-radius: 8px; padding: 9px 16px; font: inherit; font-size: 0.86rem; font-weight: 550; cursor: pointer; white-space: nowrap; }
  .btn:disabled { opacity: .5; cursor: default; }
  .err { color: var(--danger, #e5484d); font-size: 0.82rem; margin: 12px 0 0; }
  .card { max-width: 820px; margin-top: 16px; border: 1px solid var(--border); border-left-width: 3px; border-radius: 10px; padding: 11px 14px; background: var(--card); }
  .card.ok { border-left-color: var(--good, #3fb950); }
  .card.bad { border-left-color: var(--danger, #e5484d); }
  .chead { display: flex; align-items: center; justify-content: space-between; margin-bottom: 6px; gap: 10px; }
  .badge { font-size: 0.68rem; font-weight: 700; text-transform: uppercase; letter-spacing: .03em; padding: 2px 9px; border-radius: 6px; }
  .badge.ok { background: var(--good, #3fb950); color: #04260f; }
  .badge.bad { background: var(--danger, #e5484d); color: #fff; }
  .note { margin: 0 0 9px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .facts { display: flex; flex-wrap: wrap; gap: 6px 16px; font-size: 0.76rem; color: var(--text-faint, var(--text-dim)); }
  .facts code { font-family: var(--font-mono, ui-monospace, monospace); color: var(--text-dim); }
</style>
