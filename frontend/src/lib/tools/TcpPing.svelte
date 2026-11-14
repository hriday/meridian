<script lang="ts">
  import { TCPPing } from "../../../wailsjs/go/app/App.js";

  let host = "";
  let port: number = 443;
  let count: number = 10;
  let rep: any = null;
  let loading = false;
  let error = "";

  async function run() {
    const h = host.trim();
    if (!h || !port) return;
    loading = true;
    error = "";
    rep = null;
    try {
      rep = await TCPPing(h, Number(port), Number(count) || 10);
    } catch (e: any) {
      error = String(e?.message ?? e);
    } finally {
      loading = false;
    }
  }

  const verdictClass = (r: any) => (!r || r.received === 0 ? "bad" : r.lossPct > 0 ? "warn" : "ok");
  const maxRtt = (r: any) => Math.max(1, ...r.probes.filter((p: any) => p.ok).map((p: any) => p.rttMs));
</script>

<div class="tool">
  <p class="lede">
    A <strong>ping for a TCP port</strong>: it times the connection handshake to <code>host:port</code> instead of
    sending ICMP — so it works through firewalls that drop ping, and needs no elevated privileges. A refused or
    timed-out connection counts as loss. Point-in-time; nothing stored.
  </p>

  <div class="row">
    <input class="field host" bind:value={host} placeholder="host or IP" spellcheck="false" autocomplete="off"
      on:keydown={(e) => e.key === "Enter" && run()} />
    <input class="field port" type="number" min="1" max="65535" bind:value={port} placeholder="port" />
    <label class="cnt">× <input class="field num" type="number" min="1" max="50" bind:value={count} /></label>
    <button class="btn" on:click={run} disabled={loading || !host.trim() || !port}>{loading ? "Pinging…" : "Ping"}</button>
  </div>

  {#if error}<p class="err">{error}</p>{/if}

  {#if rep}
    <div class="verdict {verdictClass(rep)}">
      <span class="vtext">{rep.verdict}</span>
      {#if rep.address}<span class="vaddr">{rep.address}</span>{/if}
    </div>

    <div class="tiles">
      <div class="tile"><span class="tl">Loss</span><span class="tv">{rep.lossPct}%</span><span class="ts">{rep.received}/{rep.sent}</span></div>
      {#if rep.received > 0}
        <div class="tile"><span class="tl">Avg</span><span class="tv">{rep.avgMs}</span><span class="ts">ms</span></div>
        <div class="tile"><span class="tl">Min</span><span class="tv">{rep.minMs}</span><span class="ts">ms</span></div>
        <div class="tile"><span class="tl">Max</span><span class="tv">{rep.maxMs}</span><span class="ts">ms</span></div>
        <div class="tile"><span class="tl">Jitter</span><span class="tv">{rep.jitterMs}</span><span class="ts">ms</span></div>
      {/if}
    </div>

    {#if rep.probes && rep.probes.length}
      <div class="bars">
        {#each rep.probes as p}
          <div class="bar" title={p.ok ? `#${p.seq}: ${p.rttMs} ms` : `#${p.seq}: ${p.err}`}>
            <div class="fill {p.ok ? 'ok' : 'miss'}" style="height:{p.ok ? Math.max(4, (p.rttMs / maxRtt(rep)) * 100) : 100}%"></div>
          </div>
        {/each}
      </div>
    {/if}
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 30px; }
  .lede { max-width: 760px; margin: 0 0 14px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .lede code { font-family: var(--font-mono, ui-monospace, monospace); font-size: 0.78rem; }
  .row { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
  .field { background: var(--well); border: 1px solid var(--field-border, var(--border)); border-radius: 8px; padding: 9px 12px; color: var(--text); font-size: 0.86rem; outline: none; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .host { flex: 1 1 200px; min-width: 0; }
  .port { width: 92px; }
  .cnt { display: flex; align-items: center; gap: 6px; color: var(--text-dim); font-size: 0.82rem; }
  .num { width: 64px; }
  .btn { background: var(--accent); color: var(--on-accent, #052e2b); border: 1px solid transparent; border-radius: 8px; padding: 9px 16px; font: inherit; font-size: 0.86rem; font-weight: 550; cursor: pointer; white-space: nowrap; }
  .btn:disabled { opacity: .5; cursor: default; }
  .err { color: var(--danger, #e5484d); font-size: 0.82rem; margin: 12px 0 0; }
  .verdict { max-width: 760px; margin-top: 16px; padding: 11px 15px; border: 1px solid var(--border); border-left: 3px solid var(--border); border-radius: 10px; background: var(--card); display: flex; align-items: center; justify-content: space-between; gap: 12px; flex-wrap: wrap; }
  .verdict.ok { border-left-color: var(--good, #3fb950); }
  .verdict.warn { border-left-color: var(--warn, #f5a623); }
  .verdict.bad { border-left-color: var(--danger, #e5484d); }
  .vtext { font-size: 0.86rem; color: var(--text); }
  .vaddr { font-family: var(--font-mono, ui-monospace, monospace); font-size: 0.74rem; color: var(--text-dim); }
  .tiles { display: flex; gap: 10px; max-width: 760px; margin-top: 12px; flex-wrap: wrap; }
  .tile { flex: 1 1 90px; border: 1px solid var(--border); border-radius: 10px; padding: 9px 12px; background: var(--card); display: flex; flex-direction: column; gap: 1px; }
  .tl { font-size: 0.66rem; text-transform: uppercase; letter-spacing: .04em; color: var(--text-dim); font-weight: 650; }
  .tv { font-size: 1.15rem; font-weight: 700; color: var(--text-strong, var(--text)); }
  .ts { font-size: 0.68rem; color: var(--text-dim); }
  .bars { display: flex; align-items: flex-end; gap: 3px; height: 72px; max-width: 760px; margin-top: 14px; padding: 8px 10px; border: 1px solid var(--border); border-radius: 10px; background: var(--card); }
  .bar { flex: 1 1 0; height: 100%; display: flex; align-items: flex-end; min-width: 3px; }
  .fill { width: 100%; border-radius: 2px 2px 0 0; }
  .fill.ok { background: var(--good, #3fb950); }
  .fill.miss { background: var(--danger, #e5484d); opacity: .35; }
</style>
