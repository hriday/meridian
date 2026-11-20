<script lang="ts">
  import { get } from "svelte/store";
  import { NetworkDiagnostic } from "../../../wailsjs/go/app/App.js";
  import { requestedTool } from "../nav";
  import ToolHelp from "../ToolHelp.svelte";

  let host = "";
  let res: any = null;
  let error = "";
  let loading = false;

  async function run() {
    const q = host.trim();
    if (!q) return;
    loading = true;
    error = "";
    res = null;
    try {
      res = await NetworkDiagnostic(q);
    } catch (e: any) {
      error = String(e?.message ?? e);
    } finally {
      loading = false;
    }
  }

  // Severity of the verdict card: all_clear good, inconclusive neutral, anything else a problem.
  const sev = (v: string) => (v === "all_clear" ? "good" : v === "inconclusive" ? "neutral" : "warn");
  const label = (s: string) => (s === "pass" ? "PASS" : s === "fail" ? "FAIL" : s === "skip" ? "SKIP" : "ERROR");

  // Hand-off from a failing tool: seed the host and run immediately.
  const req = get(requestedTool);
  if (req && req.tool === "netdiag") {
    if (req.value) host = req.value;
    requestedTool.set(null);
    if (host.trim()) run();
  }
</script>

<div class="tool">
  <p class="lede">
    Can't reach something on the local network? This runs a short, ordered set of checks — DHCP, gateway, internet,
    then the target host — and tells you <strong>which layer is broken and what to do about it</strong>.
  </p>

  <ToolHelp>
    Each step is a probe from this machine. <strong>Pass</strong> means that layer works, <strong>fail</strong> means
    it doesn't, <strong>skip</strong> means it wasn't needed (an earlier step already explained the problem), and
    <strong>error</strong> means the probe itself couldn't run. The verdict card names the most likely cause and the
    concrete fix. It only reads state and sends a few probes; it changes nothing.
  </ToolHelp>

  <div class="row">
    <input class="field mono" bind:value={host} placeholder="10.0.0.50   ·   nas.local" spellcheck="false"
      on:keydown={(e) => e.key === "Enter" && run()} aria-label="Target host" />
    <button class="btn" on:click={run} disabled={loading || !host.trim()}>{loading ? "Running…" : "Run"}</button>
  </div>

  {#if error}<p class="err">{error}</p>{/if}

  {#if res}
    <div class="verdict {sev(res.verdict)}" role="status">
      <div class="vlabel">Verdict</div>
      <div class="vstate">{res.statement}</div>
      {#if res.causes && res.causes.length}
        <div class="vh">Likely cause</div>
        <ul>{#each res.causes as c}<li>{c}</li>{/each}</ul>
      {/if}
      {#if res.fixes && res.fixes.length}
        <div class="vh">How to fix</div>
        <ol class="fixes">{#each res.fixes as f}<li>{f}</li>{/each}</ol>
      {/if}
    </div>

    {#if res.steps && res.steps.length}
      <div class="steps">
        {#each res.steps as s}
          <div class="step">
            <span class="badge {s.status}">{label(s.status)}</span>
            <span class="sname">{s.name}</span>
            {#if s.detail}<span class="sdetail">{s.detail}</span>{/if}
          </div>
        {/each}
      </div>
    {/if}
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 30px; }
  .lede { max-width: 820px; margin: 0 0 12px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .mono { font-family: var(--font-mono, ui-monospace, monospace); }
  .row { display: flex; gap: 10px; max-width: 640px; margin-bottom: 14px; }
  .field { flex: 1 1 auto; min-width: 0; background: var(--well); border: 1px solid var(--field-border, var(--border)); border-radius: 8px; padding: 9px 12px; color: var(--text); font-size: 0.88rem; outline: none; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .btn { background: var(--accent); color: var(--on-accent, #052e2b); border: 1px solid transparent; border-radius: 8px; padding: 9px 18px; font: inherit; font-size: 0.86rem; font-weight: 550; cursor: pointer; }
  .btn:disabled { opacity: .5; cursor: default; }
  .err { color: var(--bad, #ff6b6b); font-size: 0.84rem; max-width: 820px; }

  .verdict { max-width: 640px; background: var(--card); border: 1px solid var(--border); border-left: 4px solid var(--border); border-radius: 12px; padding: 14px 16px; margin-bottom: 14px; }
  .verdict.good { border-left-color: var(--good, #3fb950); }
  .verdict.warn { border-left-color: var(--warn, #f5a623); }
  .verdict.neutral { border-left-color: var(--text-dim); }
  .vlabel { font-size: 0.68rem; text-transform: uppercase; letter-spacing: .06em; font-weight: 700; color: var(--text-dim); }
  .vstate { font-size: 1.02rem; font-weight: 650; color: var(--text-strong, var(--text)); margin: 4px 0 8px; line-height: 1.4; }
  .vh { font-size: 0.7rem; text-transform: uppercase; letter-spacing: .05em; font-weight: 700; color: var(--text-dim); margin-top: 10px; }
  .verdict ul, .verdict ol { margin: 4px 0 0; padding-left: 20px; font-size: 0.84rem; color: var(--text); line-height: 1.5; }
  .fixes li { font-weight: 550; }

  .steps { max-width: 640px; display: flex; flex-direction: column; gap: 6px; }
  .step { display: flex; align-items: baseline; gap: 10px; background: var(--card); border: 1px solid var(--border); border-radius: 9px; padding: 8px 12px; }
  .badge { flex: none; width: 52px; text-align: center; font-size: 0.64rem; font-weight: 700; letter-spacing: .04em; border-radius: 20px; padding: 2px 0; border: 1px solid var(--border); color: var(--text-dim); }
  .badge.pass { color: var(--good, #3fb950); border-color: var(--good, #3fb950); }
  .badge.fail { color: var(--bad, #ff6b6b); border-color: var(--bad, #ff6b6b); }
  .badge.error { color: var(--warn, #f5a623); border-color: var(--warn, #f5a623); }
  .sname { font-size: 0.84rem; font-weight: 600; color: var(--text); flex: none; }
  .sdetail { font-size: 0.78rem; color: var(--text-dim); min-width: 0; overflow-wrap: anywhere; }
</style>
