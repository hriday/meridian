<script lang="ts">
  import { MeshtasticSnapshot, MeshtasticSerialPorts } from "../../../wailsjs/go/app/App.js";
  import ToolHelp from "../ToolHelp.svelte";
  import { onMount, onDestroy, afterUpdate } from "svelte";

  let target = "";
  let snap: any = null;
  let loading = false;
  let err = "";

  // Detected USB serial devices, so a node can be picked instead of typing its
  // device path. A background convenience: if the scan fails we stay silent and
  // the field still works — we never claim "no devices" on a failed scan.
  let ports: { path: string; label: string }[] = [];
  let portsScanned = false;

  async function loadPorts() {
    try {
      const r = await MeshtasticSerialPorts();
      ports = Array.isArray(r) ? r : [];
    } catch (e) {
      ports = [];
    } finally {
      portsScanned = true;
    }
  }
  onMount(loadPorts);

  // Picking a detected device fills the field and reads it straight away — the
  // common case is one click. The field stays editable for TCP hosts or a path
  // the scan didn't surface.
  function pickPort(p: { path: string }) {
    target = p.path;
    run();
  }

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

  // --- Node map (Leaflet + OpenStreetMap tiles) ---------------------------
  // The map is opt-in and Leaflet is imported ONLY when the user asks for it, so
  // nothing from Leaflet runs on the tool-open path. (Auto-initialising Leaflet
  // on open froze the whole window inside the macOS WKWebView in v1.21.0; keeping
  // it off the open path keeps the tool usable regardless of that.)
  let mapEl: HTMLDivElement;
  let map: any = null;
  let layer: any = null;
  let L: any = null;
  let showMap = false;
  let mapLoading = false;
  let mapErr = "";

  async function enableMap() {
    mapErr = "";
    if (!L) {
      mapLoading = true;
      try {
        L = (await import("leaflet")).default;
        await import("leaflet/dist/leaflet.css");
      } catch (e) {
        mapErr = "Couldn't load the map component.";
        mapLoading = false;
        return;
      }
      mapLoading = false;
    }
    showMap = true;
  }

  function positioned(s: any): any[] {
    // Strict checks: a real position has hasPosition === true AND finite numeric
    // coordinates. This also keeps a marker off the map for any malformed node.
    const ok = (n: any) => n?.hasPosition === true && Number.isFinite(n?.lat) && Number.isFinite(n?.lon);
    const out: any[] = [];
    if (ok(s?.me)) out.push({ ...s.me, _me: true });
    for (const n of s?.nodes || []) if (ok(n)) out.push(n);
    return out;
  }
  function esc(s: string): string {
    return String(s).replace(/[&<>"]/g, (c) => (({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" } as any)[c]));
  }

  $: pts = snap ? positioned(snap) : [];

  // Sync the map after the DOM updates, so the <div bind:this={mapEl}> exists
  // when we hand it to Leaflet. Keyed so it only re-syncs when the positioned
  // set actually changes (not on every unrelated render, e.g. the loading flag).
  // Sync the map after the DOM updates, so the <div bind:this={mapEl}> exists
  // when we hand it to Leaflet. Only runs once the user has opened the map and
  // Leaflet has loaded. Keyed (on every field a marker renders) so it re-syncs
  // only when the positioned set actually changes.
  let mapKey = "";
  afterUpdate(() => {
    if (!showMap || !L) return;
    const key = pts.map((p) => JSON.stringify([p._me ? 1 : 0, p.num, p.lat, p.lon, p.snr, p.longName, p.shortName, p.id, p.hopsAway, p.viaMqtt])).join("|");
    if (key !== mapKey) {
      mapKey = key;
      syncMap(mapEl, pts);
    }
  });

  function syncMap(el: HTMLDivElement | undefined, points: any[]) {
    if (!L) return;
    // A map failure must never wedge the component (and with it the app's whole
    // update cycle): contain every Leaflet call.
    try {
      if (!el || !points.length) {
        if (map) { map.remove(); map = null; layer = null; }
        return;
      }
      if (!map) {
        map = L.map(el, { zoomControl: true, attributionControl: true });
        // OSM's current tile endpoint (the a/b/c subdomains are deprecated).
        L.tileLayer("https://tile.openstreetmap.org/{z}/{x}/{y}.png", {
          maxZoom: 19,
          attribution: "&copy; OpenStreetMap contributors",
        }).addTo(map);
      }
      if (layer) layer.remove();
      layer = L.layerGroup().addTo(map);
      const ll: [number, number][] = [];
      for (const p of points) {
        const cls = p._me ? "me" : snrClass(p.snr);
        const m = L.marker([p.lat, p.lon], {
          icon: L.divIcon({ className: "mkwrap", html: `<span class="mk ${cls}"></span>`, iconSize: [16, 16], iconAnchor: [8, 8] }),
        }).addTo(layer);
        const detail = p._me ? "this node" : `SNR ${Number(p.snr).toFixed(1)} dB &middot; ${esc(hops(p))}`;
        m.bindPopup(`<b>${esc(nodeName(p))}</b><br>${detail}`);
        ll.push([p.lat, p.lon]);
      }
      if (ll.length === 1) map.setView(ll[0], 14);
      else map.fitBounds(ll, { padding: [28, 28] });
      // The container may have been sized after creation; let Leaflet re-measure.
      setTimeout(() => { try { map && map.invalidateSize(); } catch {} }, 0);
    } catch (e) {
      if (map) { try { map.remove(); } catch {} }
      map = null;
      layer = null;
      mapErr = "The map failed to render.";
    }
  }

  onDestroy(() => { if (map) { try { map.remove(); } catch {} map = null; } });
</script>

<div class="tool">
  <p class="lede">
    Connect to a <strong>Meshtastic</strong> LoRa mesh node over USB serial or TCP and read its
    <strong>node database</strong> — every node it currently knows, with per-neighbour signal, hop distance,
    channel airtime and positions. A point-in-time snapshot of the mesh; Meridian keeps no history.
  </p>

  <ToolHelp>
    A node plugged in <strong>over USB</strong> shows up as a button under the box — click it to read that
    device, no path to type. Otherwise enter the node's <strong>serial device</strong> (e.g.
    <code>/dev/cu.usbserial-10</code> on macOS, <code>/dev/ttyUSB0</code> on Linux, <code>COM3</code> on Windows)
    or a <strong>host</strong> if the node exposes its TCP API (<code>192.168.1.50</code>, default port
    <code>4403</code>). The USB port is exclusive — close any other app connected to the node first. Each row is one node: the coloured dot and
    <strong>SNR</strong> are the last-heard signal (green is strong, amber usable, red marginal), <strong>hops</strong>
    is how many relays away it sits (<code>direct</code> is a one-hop RF neighbour), and <code>via MQTT</code> means it
    was seen through an internet gateway rather than over the air. Nodes that report a position can be
    shown on a <strong>map</strong> (press <strong>Show map</strong>; your node is a square), which loads
    OpenStreetMap tiles — the only time this tool reaches the internet.
  </ToolHelp>

  <div class="searchrow">
    <input class="field mono" bind:value={target} placeholder="/dev/cu.usbserial-10   or   192.168.1.50" spellcheck="false" autocomplete="off"
      on:keydown={(e) => e.key === "Enter" && run()} />
    <button class="btn" on:click={run} disabled={loading || !target.trim()}>{loading ? "Reading node…" : "Read node"}</button>
  </div>

  {#if ports.length}
    <div class="picker">
      <span class="pklbl">Detected over USB:</span>
      {#each ports as p}
        <button class="chip" class:on={target.trim() === p.path} on:click={() => pickPort(p)} disabled={loading} title={`Read ${p.path}`}>
          <span class="cname">{p.label}</span>
          <span class="cpath mono">{p.path}</span>
        </button>
      {/each}
      <button class="rescan" on:click={loadPorts} disabled={loading} title="Scan for attached USB serial devices again" aria-label="Rescan">↻</button>
    </div>
  {:else if portsScanned}
    <div class="picker">
      <span class="pklbl dim">No USB serial devices detected — plug the node in over USB, or enter a host for its TCP API.</span>
      <button class="rescan" on:click={loadPorts} disabled={loading} title="Scan again" aria-label="Rescan">↻</button>
    </div>
  {/if}

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

    {#if pts.length}
      <div class="mapwrap">
        <div class="maphead">Node map · {pts.length} positioned {pts.length === 1 ? "node" : "nodes"}</div>
        {#if showMap}
          <div class="mapbox" bind:this={mapEl}></div>
          <div class="maphint">Map tiles load from OpenStreetMap — the only outbound request this tool makes.</div>
        {:else}
          <button class="mapbtn" on:click={enableMap} disabled={mapLoading}>{mapLoading ? "Loading map…" : "Show map"}</button>
          <div class="maphint">Plots the positioned nodes on an OpenStreetMap map. Loads map tiles (an outbound request) only when you open it.</div>
        {/if}
        {#if mapErr}<div class="maphint err">{mapErr}</div>{/if}
      </div>
    {/if}

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

  .picker { display: flex; flex-wrap: wrap; align-items: center; gap: 8px; max-width: 720px; margin: -4px 0 14px; }
  .pklbl { font-size: 0.74rem; color: var(--text-dim); font-weight: 550; }
  .pklbl.dim { color: var(--text-faint, var(--text-dim)); font-weight: 400; }
  .chip { display: inline-flex; align-items: baseline; gap: 8px; background: var(--well); border: 1px solid var(--border); border-radius: 20px; padding: 5px 13px; font: inherit; cursor: pointer; max-width: 100%; }
  .chip:hover:not(:disabled) { border-color: var(--accent); }
  .chip.on { border-color: var(--accent); box-shadow: 0 0 0 2px var(--accent-ring); }
  .chip:disabled { opacity: .55; cursor: default; }
  .cname { font-size: 0.8rem; color: var(--text); font-weight: 500; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 320px; }
  .cpath { font-size: 0.72rem; color: var(--text-dim); white-space: nowrap; }
  .rescan { background: transparent; border: 1px solid var(--border); border-radius: 50%; width: 26px; height: 26px; line-height: 1; color: var(--text-dim); cursor: pointer; font-size: 0.9rem; flex: none; }
  .rescan:hover:not(:disabled) { border-color: var(--accent); color: var(--text); }
  .rescan:disabled { opacity: .5; cursor: default; }

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

  .mapwrap { max-width: 820px; margin: 0 0 14px; }
  .maphead { font-size: 0.7rem; text-transform: uppercase; letter-spacing: .05em; color: var(--text-dim); font-weight: 600; margin-bottom: 6px; }
  .mapbox { height: 320px; border: 1px solid var(--border); border-radius: 11px; overflow: hidden; z-index: 0; background: var(--well); }
  .maphint { font-size: 0.7rem; color: var(--text-faint, var(--text-dim)); margin-top: 5px; }
  .maphint.err { color: var(--bad, #f0616a); }
  .mapbtn { background: var(--well); color: var(--text); border: 1px solid var(--border); border-radius: 8px; padding: 8px 16px; font: inherit; font-size: 0.82rem; font-weight: 500; cursor: pointer; }
  .mapbtn:hover { border-color: var(--accent); }
  .mapbtn:disabled { opacity: .5; cursor: default; }
  /* Leaflet injects marker HTML outside Svelte's style scope, so these are global. */
  :global(.mk) { display: block; width: 13px; height: 13px; border-radius: 50%; border: 2px solid #fff; box-shadow: 0 0 0 1px rgba(0,0,0,.35); box-sizing: border-box; }
  :global(.mk.good) { background: #3fb950; }
  :global(.mk.warn) { background: #d6a531; }
  :global(.mk.bad) { background: #f0616a; }
  :global(.mk.me) { background: var(--accent, #3b82f6); border-radius: 3px; }
  :global(.leaflet-container) { font: inherit; }

  .empty { max-width: 820px; padding: 14px; border: 1px dashed var(--border); border-radius: 11px; font-size: 0.82rem; color: var(--text-dim); }
  .foot { max-width: 820px; font-size: 0.74rem; color: var(--text-faint, var(--text-dim)); margin: 12px 0 0; }
</style>
