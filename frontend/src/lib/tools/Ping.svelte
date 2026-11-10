<script lang="ts">
  import { PingHost } from "../../../wailsjs/go/app/App.js";

  let host = "1.1.1.1";
  let count = 5;
  let busy = false;
  let error = "";
  let stats: any = null;

  async function run() {
    const h = host.trim();
    if (!h) {
      stats = null;
      return;
    }
    error = "";
    stats = null;
    busy = true;
    try {
      stats = await PingHost(h, Number(count) || 5);
    } catch (e) {
      error = String(e);
    } finally {
      busy = false;
    }
  }

  // Bar heights for the RTT sparkline, scaled to the max RTT; timeouts render
  // as a zero-height marked bar.
  function bars(s: any): { h: number; ok: boolean }[] {
    const replies = s?.replies ?? [];
    const max = Math.max(1, ...replies.filter((r: any) => r.ok).map((r: any) => r.rttMillis));
    return replies.map((r: any) => ({ h: r.ok ? Math.max(6, (r.rttMillis / max) * 100) : 0, ok: r.ok }));
  }

  // One-line plain-language read of the result.
  function verdict(s: any): string {
    if (!s) return "";
    if (!s.available) return "Unprivileged ICMP isn't available on this machine, so latency can't be measured.";
    if (s.received === 0) return "No replies — the host didn't answer (it may be down or blocking ICMP).";
    const parts: string[] = [];
    if (s.lossPct > 0) parts.push(`${s.lossPct.toFixed(0)}% packet loss`);
    const jumpy = s.jitterMs > 30 || (s.avgMs > 0 && s.jitterMs > s.avgMs * 0.5);
    if (jumpy) parts.push("high jitter — the connection is uneven");
    if (parts.length === 0) {
      if (s.avgMs < 30) return "Low, steady latency — a healthy connection.";
      if (s.avgMs < 100) return "Moderate latency, steady.";
      return "High but steady latency.";
    }
    return parts.join("; ") + ".";
  }

  const fmt = (v: number) => (v == null ? "—" : v.toFixed(2));
</script>

<div class="tool">
  <p class="lede">
    Measures how well a host responds — not just up/down. Sends a few ICMP echoes and
    reports round-trip latency, how much it varies (<b>jitter</b>), and any packet
    loss. For many hosts at once, use <span class="mono">Ping sweep</span>.
  </p>

  <form on:submit|preventDefault={run} class="grid">
    <label>Host
      <input class="field mono" bind:value={host} placeholder="1.1.1.1  ·  example.com" autocomplete="off" spellcheck="false" aria-label="host" />
    </label>
    <label>Count
      <input class="field mono num" type="number" bind:value={count} min="1" max="50" aria-label="count" />
    </label>
    <button class="go" disabled={busy}>{busy ? "Pinging…" : "Ping"}</button>
  </form>

  {#if error}<div class="banner bad"><span class="vlabel">Couldn't ping</span><span class="vnote">{error}</span></div>{/if}

  {#if stats}
    <div class="verdict" class:good={stats.available && stats.received > 0 && stats.lossPct === 0} class:warn={stats.available && (stats.received === 0 || stats.lossPct > 0)} class:bad={!stats.available}>
      <span class="vlabel">{stats.host}{#if stats.ip && stats.ip !== stats.host} · {stats.ip}{/if}</span>
      <span class="vnote">{verdict(stats)}</span>
    </div>

    {#if stats.available}
      <div class="cards">
        <div class="card span">
          <div class="k">Round trips</div>
          <div class="spark">
            {#each bars(stats) as b}
              <div class="bar" class:lost={!b.ok} style={`height:${b.h}%`} title={b.ok ? "" : "timeout"}></div>
            {/each}
          </div>
          <div class="seqs">
            {#each stats.replies as r}
              <span class="seq" class:lost={!r.ok}>{r.ok ? `${fmt(r.rttMillis)}` : "×"}</span>
            {/each}
          </div>
        </div>

        <div class="card">
          <div class="k">Packets</div>
          <dl>
            <dt>Sent</dt><dd>{stats.sent}</dd>
            <dt>Received</dt><dd>{stats.received}</dd>
            <dt>Loss</dt><dd>{stats.lossPct.toFixed(0)}%</dd>
          </dl>
        </div>

        <div class="card">
          <div class="k">Latency (ms)</div>
          <dl>
            <dt>min / avg / max</dt><dd class="mono">{fmt(stats.minMs)} / {fmt(stats.avgMs)} / {fmt(stats.maxMs)}</dd>
            <dt>Std dev</dt><dd class="mono">{fmt(stats.stddevMs)}</dd>
            <dt>Jitter</dt><dd class="mono">{fmt(stats.jitterMs)}</dd>
          </dl>
        </div>
      </div>
    {/if}
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 20px; }
  .lede { max-width: 760px; margin: 0 0 14px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .grid { display: flex; gap: 10px; align-items: flex-end; flex-wrap: wrap; max-width: 760px; }
  label { display: flex; flex-direction: column; gap: 5px; font-size: 0.76rem; color: var(--text-dim); }
  .field {
    background: var(--well); border: 1px solid var(--field-border); border-radius: 7px;
    padding: 9px 12px; color: var(--text); font-size: 0.9rem; outline: none; min-width: 260px;
  }
  .field.num { min-width: 80px; width: 80px; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .mono { font-family: var(--mono, ui-monospace, monospace); }
  .go {
    background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px;
    padding: 0 16px; height: 37px; font: inherit; font-size: 0.9rem; cursor: pointer; white-space: nowrap;
  }
  .go:disabled { opacity: .6; cursor: default; }

  .verdict, .banner {
    margin: 14px 0; padding: 11px 14px; border-radius: 9px; border: 1px solid var(--border-soft);
    display: flex; flex-direction: column; gap: 3px; max-width: 760px;
  }
  .verdict.good { border-color: transparent; background: var(--good-bg, #102516); }
  .verdict.warn { border-color: var(--warn, #d9a441); background: color-mix(in srgb, var(--warn, #d9a441) 12%, transparent); }
  .verdict.bad, .banner.bad { border-color: var(--bad, #ff6b6b); background: color-mix(in srgb, var(--bad, #ff6b6b) 10%, transparent); }
  .vlabel { font-weight: 600; color: var(--text-strong); }
  .verdict.good .vlabel { color: var(--good, #5ad17a); }
  .verdict.warn .vlabel { color: var(--warn, #d9a441); }
  .verdict.bad .vlabel, .banner.bad .vlabel { color: var(--bad, #ff6b6b); }
  .vnote { font-size: 0.82rem; color: var(--text-dim); }

  .cards { display: grid; grid-template-columns: 1fr 1fr; gap: 13px; }
  @media (max-width: 760px) { .cards { grid-template-columns: 1fr; } }
  .card { background: var(--card); border: 1px solid var(--border-soft); border-radius: 10px; padding: 13px 15px; }
  .card.span { grid-column: 1 / -1; }
  .k { font-size: 0.74rem; text-transform: uppercase; letter-spacing: 0.04em; color: var(--text-faint); margin-bottom: 9px; }
  .spark { display: flex; align-items: flex-end; gap: 4px; height: 72px; padding: 4px 0; }
  .bar { flex: 1; min-width: 6px; background: var(--accent); border-radius: 3px 3px 0 0; opacity: 0.85; }
  .bar.lost { background: var(--bad, #ff6b6b); height: 100% !important; opacity: 0.18; }
  .seqs { display: flex; gap: 4px; margin-top: 6px; }
  .seq { flex: 1; min-width: 6px; text-align: center; font-size: 0.68rem; color: var(--text-dim); font-family: var(--mono, ui-monospace, monospace); }
  .seq.lost { color: var(--bad, #ff6b6b); }
  dl { display: grid; grid-template-columns: max-content 1fr; gap: 5px 14px; margin: 0; }
  dt { color: var(--text-dim); font-size: 0.82rem; }
  dd { margin: 0; font-size: 0.86rem; color: var(--text); }
</style>
