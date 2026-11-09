<script lang="ts">
  import { NTPQuery, NTPSystem } from "../../../wailsjs/go/app/App.js";

  let system: any = null;
  let query = "time.apple.com";
  let busy = false;
  let error = "";
  let reply: any = null;

  async function loadSystem() {
    try {
      system = await NTPSystem();
      if (system?.values?.length) query = system.values[0];
    } catch (_) {
      system = null;
    }
  }

  async function run(server?: string) {
    const s = (server ?? query).trim();
    query = s;
    if (!s) return;
    error = "";
    busy = true;
    reply = null;
    try {
      reply = await NTPQuery(s);
    } catch (e) {
      error = String(e);
    } finally {
      busy = false;
    }
  }

  function ms(ns: number): string {
    if (ns == null) return "—";
    return (ns / 1e6).toFixed(1) + " ms";
  }
  function precision(sec: number): string {
    if (!sec) return "—";
    return "±" + (sec * 1e6).toFixed(2) + " µs";
  }
  function fmtTime(ts: string): string {
    if (!ts) return "—";
    const d = new Date(ts);
    return isNaN(d.getTime()) ? ts : d.toISOString().replace("T", " ").replace("Z", " UTC");
  }

  loadSystem();
  run();
</script>

<div class="tool">
  <h2>NTP</h2>
  <p class="sub">Query a time server — stratum, offset, and delay. Meridian never sets your clock.</p>

  <form on:submit|preventDefault={() => run()} class="row">
    <input class="field mono" bind:value={query} placeholder="time.apple.com" autocomplete="off" spellcheck="false" aria-label="NTP server" />
    <button class="go" disabled={busy}>{busy ? "Querying…" : "Query"}</button>
    {#if system?.values?.length}
      <button type="button" class="ghost" on:click={() => run(system.values[0])}>Use system server</button>
    {/if}
  </form>

  {#if error}<div class="hint">{error}</div>{/if}

  <div class="cards">
    <div class="card hero">
      <div class="chead"><h3>This machine</h3><span class="src">{system?.source ?? ""}</span></div>
      {#if system?.values?.length}
        <dl class="mono"><dt>Time server</dt><dd class="big">{system.values.join(", ")}</dd></dl>
      {:else}
        <p class="muted">Couldn't detect a configured time server{#if system?.err} ({system.err}){/if} — enter one above.</p>
      {/if}
    </div>

    {#if reply}
      <div class="card">
        <div class="chead"><h3>Server response</h3><span class="src">SNTP · now</span></div>
        <div class="chain">
          <span class="node">⏱ ref</span><span class="arrow">→</span>
          <span class="node on">Stratum {reply.stratum}</span><span class="arrow">→</span>
          <span class="node">you</span>
        </div>
        <dl class="mono">
          <dt>Stratum</dt><dd>{reply.stratum}{#if reply.kissOfDeath} <span class="pill bad">kiss-o'-death</span>{/if}</dd>
          <dt>Reference</dt><dd>{reply.refId || "—"}</dd>
          <dt>Leap</dt><dd>{reply.leapIndicator}</dd>
          <dt>Precision</dt><dd>{precision(reply.precisionSeconds)}</dd>
          <dt>Server time</dt><dd>{fmtTime(reply.serverTime)}</dd>
        </dl>
      </div>

      <div class="card span">
        <div class="chead"><h3>Timing</h3><span class="src">computed</span></div>
        <div class="timing">
          <div><span class="tl">Offset</span><span class="tv">{ms(reply.offset)}</span></div>
          <div><span class="tl">Round-trip</span><span class="tv">{ms(reply.rtt)}</span></div>
          <div><span class="tl">Root delay</span><span class="tv">{ms(reply.rootDelay)}</span></div>
          <div><span class="tl">Root dispersion</span><span class="tv">{ms(reply.rootDispersion)}</span></div>
        </div>
      </div>
    {/if}
  </div>
</div>

<style>
  .tool { padding: 16px 20px 40px; max-width: 940px; }
  h2 { margin: 0 0 2px; font-size: 1.15rem; font-weight: 600; color: var(--text-strong); }
  .sub { margin: 0 0 15px; color: var(--text-dim); font-size: 0.86rem; }
  .row { display: flex; gap: 8px; margin-bottom: 14px; flex-wrap: wrap; }
  .field { flex: 1; min-width: 180px; background: var(--well); border: 1px solid var(--field-border); border-radius: 7px; padding: 9px 12px; color: var(--text); font-size: 0.9rem; outline: none; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .go { background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px; padding: 0 16px; font: inherit; font-size: 0.9rem; cursor: pointer; }
  .go:disabled { opacity: .6; cursor: default; }
  .ghost { background: var(--ghost-bg); border: 1px solid var(--ghost-border); color: var(--text); border-radius: 7px; padding: 0 13px; font: inherit; font-size: 0.84rem; cursor: pointer; }
  .hint { background: var(--warn-bg); border: 1px solid var(--warn-border); color: var(--warn-text); border-radius: 7px; padding: 9px 12px; font-size: 0.85rem; margin-bottom: 12px; }
  .cards { display: grid; grid-template-columns: 1fr 1fr; gap: 13px; }
  @media (max-width: 720px) { .cards { grid-template-columns: 1fr; } }
  .card { background: var(--card); border: 1px solid var(--border-soft); border-radius: 10px; padding: 13px 15px; }
  .card.span { grid-column: 1 / -1; } .card.hero { border-color: var(--accent); box-shadow: 0 0 0 1px var(--accent-ring); }
  .chead { display: flex; align-items: center; justify-content: space-between; margin-bottom: 10px; gap: 8px; }
  .chead h3 { margin: 0; font-size: 0.68rem; text-transform: uppercase; letter-spacing: .06em; color: var(--text-dim); font-weight: 600; }
  .src { font-size: 0.62rem; color: var(--text-faint); font-family: var(--font-mono); }
  dl { display: grid; grid-template-columns: 120px 1fr; gap: 6px 12px; margin: 0; font-size: 0.86rem; }
  dt { color: var(--text-dim); } dd { margin: 0; color: var(--text); }
  .big { font-size: 1.05rem; color: var(--text-strong); }
  .muted { color: var(--text-dim); font-size: 0.85rem; margin: 2px 0 0; }
  .chain { display: flex; align-items: center; gap: 8px; margin-bottom: 10px; flex-wrap: wrap; }
  .node { font-size: 0.7rem; padding: 3px 10px; border-radius: 7px; border: 1px solid var(--border); color: var(--text-dim); font-family: var(--font-mono); }
  .node.on { color: var(--on-accent); background: var(--btn); border-color: var(--btn-border); }
  .arrow { color: var(--text-faint); }
  .timing { display: grid; grid-template-columns: repeat(4, 1fr); gap: 10px; }
  @media (max-width: 620px) { .timing { grid-template-columns: 1fr 1fr; } }
  .timing .tl { display: block; color: var(--text-dim); font-size: 0.72rem; }
  .timing .tv { display: block; font-family: var(--font-mono); font-size: 1.05rem; color: var(--text-strong); }
  .pill { display: inline-flex; font-size: 0.64rem; padding: 1px 8px; border-radius: 20px; border: 1px solid var(--border); color: var(--text-dim); }
  .pill.bad { color: var(--bad, #ff6b6b); border-color: transparent; }
  .mono { font-family: var(--font-mono); }
</style>
