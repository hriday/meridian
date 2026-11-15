<script lang="ts">
  import { WifiStatus } from "../../../wailsjs/go/app/App.js";

  let loading = false;
  let error = "";
  let info: any = null;

  async function run() {
    if (loading) return;
    loading = true;
    error = "";
    info = null;
    try {
      info = await WifiStatus();
    } catch (e: any) {
      error = String(e?.message ?? e);
    } finally {
      loading = false;
    }
  }
  run();

  // --- signal helpers ---
  const strengthPct = (rssi: number) => Math.max(0, Math.min(100, ((rssi + 90) / 60) * 100)); // -90..-30 → 0..100
  const sigColor = (rssi: number) => (rssi >= -60 ? "var(--accent)" : rssi >= -72 ? "#d9a400" : "var(--danger, #e5484d)");
  const snrPct = (snr: number) => Math.max(0, Math.min(100, (snr / 50) * 100));

  // --- channel map ---
  const W = 680, H = 150, padL = 26, padR = 26, top = 14, bot = 120;
  function bandNets(band: string) {
    const out: any[] = [];
    if (info?.current && info.current.band === band && info.current.channel) out.push({ ...info.current, current: true });
    for (const n of info?.neighbors || []) if (n.band === band && n.channel) out.push(n);
    return out;
  }
  function chart(nets: any[]) {
    if (!nets.length) return null;
    const chans = nets.map((n) => n.channel);
    let minC = Math.min(...chans) - 6, maxC = Math.max(...chans) + 6;
    if (maxC === minC) maxC += 1;
    const xOf = (c: number) => padL + ((c - minC) / (maxC - minC)) * (W - padL - padR);
    const yOf = (rssi: number) => top + ((Math.min(-30, Math.max(-95, rssi)) + 30) / (-95 + 30)) * (bot - top);
    const unit = band === "5GHz" ? 2 : 1.6;
    const arcs = nets.map((n) => {
      const mhz = parseInt(n.width) || 20;
      const half = (mhz / 20) * unit;
      const cx = xOf(n.channel), x1 = xOf(n.channel - half), x2 = xOf(n.channel + half);
      const py = yOf(n.rssi);
      return { d: `M ${x1} ${bot} Q ${cx} ${2 * py - bot} ${x2} ${bot} Z`, cx, py, n };
    });
    const ticks = [...new Set(chans)].sort((a, b) => a - b).map((c) => ({ c, x: xOf(c) }));
    return { arcs, ticks };
  }
  $: bands = info ? ["2GHz", "5GHz", "6GHz"].map((b) => ({ band: b, nets: bandNets(b) })).filter((x) => x.nets.length) : [];
</script>

<div class="tool">
  <p class="lede">
    Your <strong>Wi-Fi / RF environment</strong>: the connected network's channel, band, <strong>signal &amp; SNR</strong>,
    and the neighboring APs — with a <strong>channel map</strong> that makes <strong>co-channel contention</strong> and
    adjacent-channel overlap obvious at a glance. No admin rights.
  </p>

  <div class="row">
    <button class="btn" on:click={run} disabled={loading}>{loading ? "Reading…" : "Refresh"}</button>
  </div>

  {#if error}<p class="err">{error}</p>{/if}

  {#if info}
    {#if info.note}<div class="note">{info.note}</div>{/if}

    <!-- Connected summary + gauges -->
    <div class="summary">
      <div class="scol">
        <div class="ssid mono">{info.current.ssid || "(not connected)"}</div>
        <div class="meta">
          ch <b>{info.current.channel}</b> · {info.current.band} · {info.current.width} · {info.current.phyMode} · {info.current.security}
          {#if info.current.txRate}· {info.current.txRate} Mbps{/if}
        </div>
      </div>
    </div>
    <div class="gauges">
      <div class="g">
        <div class="glabel">Signal <span class="mono">{info.current.rssi} dBm</span></div>
        <div class="bar"><span class="fill" style={`width:${strengthPct(info.current.rssi)}%;background:${sigColor(info.current.rssi)}`}></span></div>
      </div>
      <div class="g">
        <div class="glabel">SNR <span class="mono">{info.current.snr} dB</span></div>
        <div class="bar"><span class="fill" style={`width:${snrPct(info.current.snr)}%;background:${info.current.snr >= 25 ? "var(--accent)" : info.current.snr >= 15 ? "#d9a400" : "var(--danger, #e5484d)"}`}></span></div>
      </div>
    </div>

    {#if info.coChannel > 0}
      <div class="cochan">⚠ <b>{info.coChannel}</b> neighbor{info.coChannel === 1 ? "" : "s"} share your channel ({info.current.channel}) — co-channel contention can slow you down. A less-crowded channel may help.</div>
    {/if}

    <!-- Channel map -->
    {#each bands as b}
      <div class="chartwrap">
        <div class="bandlabel">{b.band} channels</div>
        <svg viewBox={`0 0 ${W} ${H}`} class="chart" preserveAspectRatio="none">
          <line x1={padL} y1={bot} x2={W - padR} y2={bot} class="axis" />
          {#each chart(b.nets).arcs as a}
            <path d={a.d} class="arc" class:cur={a.n.current} />
          {/each}
          {#each chart(b.nets).arcs as a}
            <text x={a.cx} y={a.py - 4} class="peak" class:curt={a.n.current}>{a.n.channel}</text>
          {/each}
          {#each chart(b.nets).ticks as t}
            <text x={t.x} y={bot + 14} class="tick">{t.c}</text>
          {/each}
        </svg>
      </div>
    {/each}

    <!-- Neighbors -->
    {#if info.neighbors && info.neighbors.length}
      <div class="sec">
        <div class="sechead">Neighbors <span class="count">{info.neighbors.length}</span></div>
        {#each info.neighbors as n}
          <div class="nb">
            <span class="nch">ch {n.channel}</span>
            <span class="nband">{n.band}</span>
            <span class="nbar"><span class="nfill" style={`width:${strengthPct(n.rssi)}%;background:${sigColor(n.rssi)}`}></span></span>
            <span class="nrssi mono">{n.rssi} dBm</span>
            {#if n.coChannel}<span class="cochip">co-channel</span>{/if}
            <span class="nssid mono">{n.ssid}</span>
          </div>
        {/each}
      </div>
    {/if}
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 30px; max-width: 760px; }
  .lede { margin: 0 0 14px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .row { display: flex; gap: 10px; }
  .btn { background: var(--accent); color: var(--on-accent, #052e2b); border: 1px solid transparent; border-radius: 8px; padding: 8px 16px; font: inherit; font-size: 0.86rem; font-weight: 550; cursor: pointer; }
  .btn:disabled { opacity: .55; cursor: default; }
  .err { color: var(--danger, #e5484d); font-size: 0.82rem; margin: 12px 0 0; }
  .note { margin: 14px 0 0; padding: 9px 12px; border: 1px solid var(--border); border-radius: 8px; background: var(--card); font-size: 0.78rem; color: var(--text-dim); }
  .summary { margin: 16px 0 10px; }
  .ssid { font-size: 1rem; font-weight: 650; color: var(--text-strong, var(--text)); }
  .meta { font-size: 0.8rem; color: var(--text-dim); margin-top: 3px; }
  .gauges { display: grid; grid-template-columns: 1fr 1fr; gap: 10px 20px; margin: 8px 0 4px; }
  .glabel { font-size: 0.76rem; color: var(--text-dim); margin-bottom: 4px; display: flex; justify-content: space-between; }
  .bar { height: 9px; background: var(--well); border: 1px solid var(--border); border-radius: 5px; overflow: hidden; }
  .fill { display: block; height: 100%; }
  .cochan { margin: 12px 0 2px; padding: 10px 13px; border-radius: 9px; font-size: 0.84rem; background: color-mix(in srgb, #d9a400 14%, transparent); border: 1px solid color-mix(in srgb, #d9a400 40%, transparent); color: var(--text); }
  .chartwrap { margin-top: 16px; }
  .bandlabel { font-size: 0.72rem; text-transform: uppercase; letter-spacing: .05em; color: var(--text-dim); font-weight: 700; margin-bottom: 4px; }
  .chart { width: 100%; height: 150px; display: block; }
  .axis { stroke: var(--border); stroke-width: 1; }
  .arc { fill: color-mix(in srgb, var(--text-dim) 18%, transparent); stroke: var(--text-dim); stroke-width: 1; }
  .arc.cur { fill: color-mix(in srgb, var(--accent) 28%, transparent); stroke: var(--accent); stroke-width: 1.5; }
  .peak { font-size: 10px; fill: var(--text-dim); text-anchor: middle; }
  .peak.curt { fill: var(--accent); font-weight: 700; }
  .tick { font-size: 9px; fill: var(--text-dim); text-anchor: middle; }
  .sec { margin-top: 18px; }
  .sechead { font-size: 0.74rem; text-transform: uppercase; letter-spacing: .05em; color: var(--text-dim); font-weight: 700; margin-bottom: 8px; display: flex; align-items: center; gap: 8px; }
  .count { background: var(--well); border: 1px solid var(--border); border-radius: 20px; padding: 1px 8px; font-size: 0.72rem; color: var(--text); }
  .nb { display: flex; align-items: center; gap: 10px; padding: 5px 2px; font-size: 0.8rem; }
  .nb + .nb { border-top: 1px solid var(--border); }
  .nch { min-width: 48px; color: var(--text); }
  .nband { min-width: 40px; color: var(--text-dim); font-size: 0.72rem; }
  .nbar { flex: 0 0 90px; height: 7px; background: var(--well); border: 1px solid var(--border); border-radius: 4px; overflow: hidden; }
  .nfill { display: block; height: 100%; }
  .nrssi { min-width: 64px; color: var(--text-dim); }
  .cochip { font-size: 0.64rem; font-weight: 700; padding: 1px 6px; border-radius: 5px; background: color-mix(in srgb, #d9a400 18%, transparent); color: #a07c00; }
  .nssid { color: var(--text-dim); flex: 1; text-align: right; }
  .mono { font-family: var(--font-mono, ui-monospace, monospace); }
</style>
