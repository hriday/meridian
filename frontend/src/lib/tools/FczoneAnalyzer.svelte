<script lang="ts">
  import { AnalyzeFCZoning } from "../../../wailsjs/go/app/App.js";
  import ToolHelp from "../ToolHelp.svelte";

  let configText = "";
  let rep: any = null;
  let busy = false;
  let error = "";

  async function analyze() {
    const v = configText.trim();
    if (!v) return;
    busy = true;
    error = "";
    rep = null;
    try {
      rep = await AnalyzeFCZoning(v);
    } catch (e: any) {
      error = String(e?.message ?? e);
    } finally {
      busy = false;
    }
  }

  const sevClass = (s: string) => (s === "warning" ? "warn" : "info");
</script>

<div class="tool">
  <p class="lede">
    Analyze <strong>Fibre Channel switch zoning</strong> for SAN hygiene. Paste <strong>Cisco MDS</strong>
    <span class="mono">show zoneset [active]</span> / running-config, or <strong>Brocade</strong>
    <span class="mono">cfgshow</span> — the vendor is auto-detected. Offline; the config never leaves your machine.
  </p>

  <ToolHelp>
    It maps your <strong>zonesets, zones and aliases</strong> and flags the usual problems: zones that aren't
    <strong>single-initiator-single-target</strong> (the big best practice), empty or single-member zones, zones not in
    any zoneset or not in the active set, and unused aliases. It also reports how many lines it <em>didn't</em> recognize,
    so a coverage gap is visible rather than silently dropped. It reads config you already have — it doesn't talk to a
    live fabric.
  </ToolHelp>

  <textarea
    class="cfg mono"
    bind:value={configText}
    placeholder="Paste `show zoneset active` (Cisco MDS) or `cfgshow` (Brocade) output here…"
    spellcheck="false"
  ></textarea>
  <div class="row">
    <button class="btn" on:click={analyze} disabled={busy || !configText.trim()}>{busy ? "Analyzing…" : "Analyze"}</button>
  </div>

  {#if error}<p class="err">{error}</p>{/if}

  {#if rep}
    <div class="head">
      <span class="vendor">{rep.vendor}</span>
      <span class="counts">{rep.counts.zonesets} zoneset(s) · {rep.counts.zones} zone(s) · {rep.counts.aliases} alias(es)</span>
    </div>

    {#if rep.zonesets && rep.zonesets.length}
      <div class="sets">
        {#each rep.zonesets as zs}
          <span class="set" class:active={zs.active}>{zs.name} · {zs.zones?.length ?? 0} zones{#if zs.active} <span class="ab">active</span>{/if}</span>
        {/each}
      </div>
    {/if}

    {#if rep.findings && rep.findings.length}
      <div class="findings">
        {#each rep.findings as f}
          <div class="finding {sevClass(f.severity)}">
            <div class="fline"><span class="sev {sevClass(f.severity)}">{f.severity}</span><span class="ftitle">{f.title}</span>{#if f.object}<span class="fobj mono">{f.object}</span>{/if}</div>
            <div class="fdetail">{f.detail}</div>
          </div>
        {/each}
      </div>
    {:else}
      <p class="good">No hygiene findings — zones look clean.</p>
    {/if}

    {#if rep.counts.unparsed > 0}
      <p class="foot">⚠ {rep.counts.unparsed} line(s) weren't recognized, so the parse may be incomplete. If this is a real export, share a sample and I can widen coverage.</p>
    {/if}
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 30px; }
  .lede { max-width: 820px; margin: 0 0 12px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .mono { font-family: var(--font-mono, ui-monospace, monospace); }
  .cfg { width: 100%; max-width: 820px; height: 190px; resize: vertical; background: var(--well); border: 1px solid var(--field-border, var(--border)); border-radius: 10px; padding: 11px 13px; color: var(--text); font-size: 0.8rem; line-height: 1.5; outline: none; }
  .cfg:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .row { margin: 10px 0 0; max-width: 820px; }
  .btn { background: var(--accent); color: var(--on-accent, #052e2b); border: 1px solid transparent; border-radius: 8px; padding: 9px 18px; font: inherit; font-size: 0.86rem; font-weight: 550; cursor: pointer; }
  .btn:disabled { opacity: .5; cursor: default; }
  .err { color: var(--bad, #ff6b6b); font-size: 0.84rem; max-width: 820px; }
  .head { display: flex; align-items: baseline; gap: 12px; max-width: 820px; margin-top: 16px; }
  .vendor { font-weight: 650; color: var(--text-strong, var(--text)); }
  .counts { font-size: 0.8rem; color: var(--text-dim); }
  .sets { display: flex; flex-wrap: wrap; gap: 7px; max-width: 820px; margin-top: 8px; }
  .set { font-size: 0.76rem; background: var(--well); border: 1px solid var(--border); border-radius: 20px; padding: 3px 11px; color: var(--text-dim); }
  .set.active { border-color: var(--accent); color: var(--text); }
  .ab { font-size: 0.64rem; text-transform: uppercase; letter-spacing: .04em; color: var(--on-accent, #052e2b); background: var(--accent); border-radius: 10px; padding: 0 6px; margin-left: 4px; }
  .findings { max-width: 820px; margin-top: 14px; display: flex; flex-direction: column; gap: 8px; }
  .finding { border: 1px solid var(--border); border-left: 3px solid var(--warn, #f5a623); border-radius: 9px; padding: 9px 13px; background: var(--card); }
  .finding.info { border-left-color: var(--text-dim); }
  .fline { display: flex; align-items: baseline; gap: 10px; flex-wrap: wrap; }
  .sev { font-size: 0.64rem; text-transform: uppercase; letter-spacing: .05em; font-weight: 700; }
  .sev.warn { color: var(--warn, #f5a623); }
  .sev.info { color: var(--text-dim); }
  .ftitle { font-weight: 600; color: var(--text); font-size: 0.84rem; }
  .fobj { margin-left: auto; font-size: 0.74rem; color: var(--text-faint, var(--text-dim)); }
  .fdetail { font-size: 0.79rem; color: var(--text-dim); margin-top: 3px; line-height: 1.5; }
  .good { max-width: 820px; margin-top: 14px; font-size: 0.84rem; color: var(--good, #3fb950); }
  .foot { max-width: 820px; font-size: 0.74rem; color: var(--text-faint, var(--text-dim)); margin: 12px 0 0; line-height: 1.5; }
</style>
