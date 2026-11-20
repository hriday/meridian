<script lang="ts">
  import { MeshtasticSnapshot } from "../../../wailsjs/go/app/App.js";
  import ToolHelp from "../ToolHelp.svelte";

  let target = "";
  let snap: any = null;
  let loading = false;
  let err = "";

  async function run() {
    err = "";
    snap = null;
    if (!target.trim()) {
      err = "Enter a serial device (e.g. /dev/cu.usbserial-10) or a host (e.g. 192.168.1.50).";
      return;
    }
    loading = true;
    try {
      snap = await MeshtasticSnapshot(target.trim());
    } catch (e) {
      err = String(e);
    } finally {
      loading = false;
    }
  }

  function nodeName(n: any): string {
    return n.longName || n.shortName || n.id || "!" + ((n.num ?? 0) >>> 0).toString(16);
  }
  function heard(epoch: number): string {
    if (!epoch || epoch <= 0) return "—";
    const d = Date.now() / 1000 - epoch;
    if (d < 60) return "now";
    if (d < 3600) return Math.floor(d / 60) + "m ago";
    if (d < 86400) return Math.floor(d / 3600) + "h ago";
    return Math.floor(d / 86400) + "d ago";
  }
  function hops(n: any): string {
    if (n.viaMqtt) return "via MQTT";
    if (n.hopsAway == null) return "hops ?";
    if (n.hopsAway === 0) return "direct";
    return n.hopsAway + (n.hopsAway === 1 ? " hop" : " hops");
  }
  // Meshtastic SNR rule of thumb: ≥0 dB is a strong link, down to about −10 dB is
  // still usable, below that is marginal.
  function snrClass(snr: number): string {
    if (snr >= 0) return "good";
    if (snr > -10) return "warn";
    return "bad";
  }
  function batt(n: any): string {
    if (n.battery == null) return "";
    if (n.battery > 100) return "powered";
    return n.battery + "%";
  }
</script>

<div class="tool">
  <p class="lede">
    Connect to a <strong>Meshtastic</strong> LoRa mesh node over USB serial or TCP and read its
    <strong>node database</strong> — every node it currently knows, with per-neighbour signal, hop distance,
    channel airtime and positions. A point-in-time snapshot of the mesh; Meridian keeps no history.
  </p>

  <ToolHelp>
    Enter the node's <strong>serial device</strong> (e.g. <code>/dev/cu.usbserial-10</code> on macOS,
    <code>/dev/ttyUSB0</code> on Linux, <code>COM3</code> on Windows) or a <strong>host</strong> if the node exposes
    its TCP API (<code>192.168.1.50</code>, default port <code>4403</code>). Each row is one node: the coloured dot and
    <strong>SNR</strong> are the last-heard signal (green is strong, amber usable, red marginal), <strong>hops</strong>
    is how many relays away it sits (<code>direct</code> is a one-hop RF neighbour), and <code>via MQTT</code> means it
    was seen through an internet gateway rather than over the air.
  </ToolHelp>

  <div class="searchrow">
    <input class="field mono" bind:value={target} placeholder="/dev/cu.usbserial-10   or   192.168.1.50" spellcheck="false" autocomplete="off"
      on:keydown={(e) => e.key === "Enter" && run()} />
    <button class="btn" on:click={run} disabled={loading || !target.trim()}>{loading ? "Reading node…" : "Read node"}</button>
  </div>

  {#if err}<div class="banner bad"><span>{err}</span></div>{/if}

  {#if snap}
    <div class="nodehead">
      <div class="idblock">
        <div class="nname">{snap.me ? nodeName(snap.me) : "Connected node"}</div>
        <div class="dim sm mono">{snap.me?.id ? snap.me.id + " · " : ""}{snap.transport} · {snap.target}</div>
      </div>
      <div class="metrics">
        <div class="metric"><div class="mv mono">{snap.hwModel || "—"}</div><div class="ml">hardware</div></div>
        <div class="metric"><div class="mv mono">{snap.firmware || "—"}</div><div class="ml">firmware</div></div>
        <div class="metric"><div class="mv mono">{snap.channelUtil.toFixed(1)}%</div><div class="ml">channel util</div></div>
        <div class="metric"><div class="mv mono">{snap.airUtilTx.toFixed(1)}%</div><div class="ml">tx airtime</div></div>
        <div class="metric"><div class="mv mono">{snap.nodeCount}</div><div class="ml">nodes seen</div></div>
      </div>
    </div>

    {#if snap.nodes && snap.nodes.length}
      <div class="card">
        {#each snap.nodes as n}
          <div class="noderow">
            <span class="ndot {snrClass(n.snr)}" title={`SNR ${n.snr.toFixed(1)} dB`}></span>
            <div class="nbody">
              <div class="nhead">
                <span class="nn">{nodeName(n)}</span>
                <span class="nid mono">{n.id}</span>
                {#if n.role}<span class="role">{n.role}</span>{/if}
                <span class="hop">{hops(n)}</span>
              </div>
              <div class="nmeta">
                <span class="mono">SNR {n.snr.toFixed(1)} dB</span>
                <span>· heard {heard(n.lastHeard)}</span>
                {#if batt(n)}<span>· {batt(n)}</span>{/if}
                {#if n.hasPosition}<span>· <a href={`https://www.openstreetmap.org/?mlat=${n.lat}&mlon=${n.lon}#map=12/${n.lat}/${n.lon}`} target="_blank" rel="noreferrer">{n.lat.toFixed(4)}, {n.lon.toFixed(4)}</a></span>{/if}
              </div>
            </div>
          </div>
        {/each}
      </div>
    {:else}
      <div class="empty">This node's database has no other nodes right now — it may be freshly powered on, or out of range of the rest of the mesh.</div>
    {/if}
    <p class="foot">Read {snap.transport === "serial" ? "over USB serial" : "over TCP"} from <span class="mono">{snap.target}</span>. SNR is the signal-to-noise of each node's last-heard packet; a point-in-time read, not a live feed.</p>
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 30px; }
  .lede { max-width: 820px; margin: 0 0 12px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .mono { font-family: var(--font-mono, ui-monospace, monospace); }
  .searchrow { display: flex; gap: 10px; max-width: 720px; margin-bottom: 14px; }
  .field { background: var(--well); border: 1px solid var(--field-border, var(--border)); border-radius: 8px; padding: 9px 12px; color: var(--text); font-size: 0.88rem; outline: none; flex: 1 1 auto; min-width: 0; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .btn { background: var(--accent); color: var(--on-accent, #052e2b); border: 1px solid transparent; border-radius: 8px; padding: 9px 16px; font: inherit; font-size: 0.86rem; font-weight: 550; cursor: pointer; white-space: nowrap; }
  .btn:disabled { opacity: .5; cursor: default; }
  .banner { display: flex; align-items: center; gap: 10px; max-width: 820px; padding: 9px 13px; border-radius: 9px; font-size: 0.84rem; margin-bottom: 12px; }
  .banner.bad { border: 1px solid var(--bad, #ff6b6b); background: color-mix(in srgb, var(--bad, #ff6b6b) 10%, transparent); color: var(--text); }

  .nodehead { display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: 14px; max-width: 820px; margin-bottom: 14px; padding: 12px 14px; background: var(--card); border: 1px solid var(--border); border-radius: 11px; }
  .nname { font-weight: 650; color: var(--text-strong, var(--text)); font-size: 1.05rem; }
  .dim { color: var(--text-dim); } .sm { font-size: 0.76rem; }
  .metrics { display: flex; flex-wrap: wrap; gap: 18px; }
  .metric { text-align: right; min-width: 56px; }
  .mv { font-size: 0.92rem; color: var(--text); font-weight: 600; }
  .ml { font-size: 0.66rem; text-transform: uppercase; letter-spacing: .05em; color: var(--text-faint, var(--text-dim)); margin-top: 2px; }

  .card { background: var(--card); border: 1px solid var(--border); border-radius: 11px; max-width: 820px; overflow: hidden; }
  .noderow { display: flex; gap: 11px; padding: 11px 14px; border-bottom: 1px solid var(--border-soft, var(--border)); }
  .noderow:last-child { border-bottom: 0; }
  .ndot { width: 9px; height: 9px; border-radius: 50%; flex: none; margin-top: 5px; }
  .ndot.good { background: var(--good, #3fb950); }
  .ndot.warn { background: var(--warn, #d6a531); }
  .ndot.bad { background: var(--bad, #f0616a); }
  .nbody { min-width: 0; flex: 1; }
  .nhead { display: flex; align-items: baseline; gap: 10px; flex-wrap: wrap; }
  .nn { font-weight: 600; color: var(--text); font-size: 0.86rem; }
  .nid { font-size: 0.74rem; color: var(--text-dim); }
  .role { font-size: 0.68rem; color: var(--text-faint, var(--text-dim)); border: 1px solid var(--border); border-radius: 20px; padding: 1px 8px; }
  .hop { margin-left: auto; font-size: 0.72rem; color: var(--text-dim); }
  .nmeta { font-size: 0.78rem; color: var(--text-dim); margin-top: 3px; display: flex; flex-wrap: wrap; gap: 6px; line-height: 1.5; }
  .nmeta a { color: var(--accent); text-decoration: none; }
  .nmeta a:hover { text-decoration: underline; }

  .empty { max-width: 820px; padding: 14px; border: 1px dashed var(--border); border-radius: 11px; font-size: 0.82rem; color: var(--text-dim); }
  .foot { max-width: 820px; font-size: 0.74rem; color: var(--text-faint, var(--text-dim)); margin: 12px 0 0; }
</style>
