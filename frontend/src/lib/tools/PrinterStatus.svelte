<script lang="ts">
  import { PrinterStatus } from "../../../wailsjs/go/app/App.js";
  import ToolHelp from "../ToolHelp.svelte";

  let host = "";
  let res: any = null;
  let error = "";
  let loading = false;

  async function probe() {
    const q = host.trim();
    if (!q) return;
    loading = true;
    error = "";
    res = null;
    try {
      res = await PrinterStatus(q);
    } catch (e: any) {
      error = String(e?.message ?? e);
    } finally {
      loading = false;
    }
  }

  const stateClass = (s: string) => (s === "idle" ? "ok" : s === "stopped" ? "bad" : s === "processing" ? "busy" : "unk");
  // IPP marker levels: 0–100 normally, with marker-high-levels as the max; <0 = unknown.
  function pct(s: any): number | null {
    if (s.level == null || s.level < 0) return null;
    const max = s.max && s.max !== 100 ? s.max : 100;
    return Math.max(0, Math.min(100, Math.round((s.level * 100) / max)));
  }
  const low = (p: number | null) => p != null && p <= 15;
  const prettyReason = (r: string) => r.replace(/-report$|-warning$/,"").replace(/-/g, " ");
</script>

<div class="tool">
  <p class="lede">
    Check a networked <strong>printer</strong> over <strong>IPP</strong> (no driver, no credentials): its state, what's
    wrong (out of paper, toner low, jam, cover open), supply levels, and which print ports are open. Read-only — it never
    sends a job.
  </p>

  <ToolHelp>
    It asks the printer for its own attributes (IPP <code>Get-Printer-Attributes</code> on port 631). <strong>State</strong>
    is idle / printing / stopped; <strong>Attention</strong> lists the IPP state-reasons — the actionable signals.
    <strong>Supplies</strong> are the printer's reported marker levels (ink/toner), shown as a percentage where the
    printer gives one. The <strong>print ports</strong> (631 IPP, 9100 raw/JetDirect, 515 LPD) tell you how it's
    printable. Nothing but the query leaves your machine.
  </ToolHelp>

  <div class="row">
    <input class="field mono" bind:value={host} placeholder="10.0.0.50   ·   printer.local" spellcheck="false"
      on:keydown={(e) => e.key === "Enter" && probe()} aria-label="Printer host" />
    <button class="btn" on:click={probe} disabled={loading || !host.trim()}>{loading ? "Probing…" : "Check"}</button>
  </div>

  {#if error}<p class="err">{error}</p>{/if}

  {#if res}
    {#if res.reachable}
      <div class="card">
        <div class="phead">
          <span class="pname">{res.makeModel || res.name || res.host}</span>
          <span class="state {stateClass(res.state)}">{res.state}</span>
        </div>
        {#if res.location}<div class="sub">{res.location}</div>{/if}

        {#if res.stateReasons && res.stateReasons.length}
          <div class="reasons">
            {#each res.stateReasons as r}<span class="reason">{prettyReason(r)}</span>{/each}
          </div>
        {/if}

        {#if res.supplies && res.supplies.length}
          <div class="supplies">
            {#each res.supplies as s}
              <div class="supply">
                {#if s.color}<span class="swatch" style={`background:${s.color}`} title={s.color}></span>{/if}
                <span class="sname">{s.name}</span>
                {#if pct(s) != null}
                  <span class="bar"><span class="fill" class:low={low(pct(s))} style={`width:${pct(s)}%` + (s.color && !low(pct(s)) ? `;background:${s.color}` : "")}></span></span>
                  <span class="spct" class:low={low(pct(s))}>{pct(s)}%</span>
                {:else}
                  <span class="spct unk">level unknown</span>
                {/if}
              </div>
            {/each}
          </div>
        {/if}

        <div class="meta">Queued jobs: {res.queuedJobs}{#if res.ippVersions && res.ippVersions.length} · IPP {res.ippVersions.join(", ")}{/if}</div>
      </div>
    {:else}
      <p class="note">{res.note || "The printer didn't answer IPP."}</p>
    {/if}

    {#if res.ports && res.ports.length}
      <div class="ports">
        <span class="pk">Print ports</span>
        {#each res.ports as p}
          <span class="port" class:open={p.open}>{p.port} {p.name} · {p.open ? "open" : "closed"}</span>
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
  .card { max-width: 640px; background: var(--card); border: 1px solid var(--border); border-radius: 12px; padding: 14px 16px; }
  .phead { display: flex; align-items: center; gap: 12px; }
  .pname { font-weight: 650; font-size: 0.95rem; color: var(--text-strong, var(--text)); }
  .state { margin-left: auto; font-size: 0.68rem; text-transform: uppercase; letter-spacing: .04em; font-weight: 700; border-radius: 20px; padding: 2px 10px; }
  .state.ok { color: var(--good, #3fb950); border: 1px solid var(--good, #3fb950); }
  .state.busy { color: var(--accent); border: 1px solid var(--accent); }
  .state.bad { color: var(--bad, #ff6b6b); border: 1px solid var(--bad, #ff6b6b); }
  .state.unk { color: var(--text-dim); border: 1px solid var(--border); }
  .sub { font-size: 0.78rem; color: var(--text-dim); margin-top: 3px; }
  .reasons { display: flex; flex-wrap: wrap; gap: 6px; margin-top: 10px; }
  .reason { font-size: 0.74rem; color: var(--warn, #f5a623); border: 1px solid var(--warn, #f5a623); border-radius: 20px; padding: 2px 10px; text-transform: capitalize; }
  .supplies { margin-top: 12px; display: flex; flex-direction: column; gap: 7px; }
  .supply { display: flex; align-items: center; gap: 10px; }
  /* Border + inset ring keeps black and white swatches visible in both themes. */
  .swatch { width: 11px; height: 11px; border-radius: 3px; flex: none; border: 1px solid rgba(128,128,128,.55); box-shadow: inset 0 0 0 1px rgba(255,255,255,.14); }
  .sname { font-size: 0.78rem; color: var(--text); width: 120px; flex: none; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .bar { flex: 1; height: 8px; background: var(--well); border-radius: 5px; overflow: hidden; }
  .fill { display: block; height: 100%; background: var(--accent); }
  .fill.low { background: var(--bad, #ff6b6b); }
  .spct { font-size: 0.74rem; color: var(--text-dim); width: 40px; text-align: right; flex: none; }
  .spct.low { color: var(--bad, #ff6b6b); font-weight: 600; }
  .spct.unk { width: auto; }
  .meta { margin-top: 12px; font-size: 0.74rem; color: var(--text-faint, var(--text-dim)); }
  .note { max-width: 640px; font-size: 0.82rem; color: var(--text-dim); background: var(--card); border: 1px solid var(--border); border-left: 3px solid var(--warn, #f5a623); border-radius: 9px; padding: 9px 13px; }
  .ports { display: flex; flex-wrap: wrap; align-items: center; gap: 7px; max-width: 640px; margin-top: 12px; }
  .pk { font-size: 0.7rem; text-transform: uppercase; letter-spacing: .05em; color: var(--text-dim); font-weight: 600; }
  .port { font-size: 0.73rem; color: var(--text-faint, var(--text-dim)); border: 1px solid var(--border); border-radius: 20px; padding: 2px 10px; }
  .port.open { color: var(--good, #3fb950); border-color: var(--good, #3fb950); }
</style>
