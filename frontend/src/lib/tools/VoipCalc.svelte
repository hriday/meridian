<script lang="ts">
  import { VoipCodecs, VoipBandwidth } from "../../../wailsjs/go/app/App.js";
  import ToolHelp from "../ToolHelp.svelte";

  let codecs: any[] = [];
  let codec = "G.711 (PCMU/PCMA)";
  let ptime = 20;
  let calls = 10;
  let res: any = null;
  let error = "";

  VoipCodecs().then((c) => (codecs = c || [])).catch(() => {});

  $: compute(codec, ptime, calls);
  async function compute(c: string, p: number, n: number) {
    try {
      res = await VoipBandwidth(c, Number(p) || 0, Number(n) || 1);
      error = "";
    } catch (e: any) {
      res = null;
      error = String(e?.message ?? e);
    }
  }
  const mbps = (k: number) => (k / 1000).toFixed(2);
</script>

<div class="tool">
  <h2>VoIP codec bandwidth</h2>
  <p class="sub">
    How much a call actually costs on the wire once you add IP/UDP/RTP (and Ethernet) overhead — and how much
    <strong>N concurrent calls</strong> need. Capacity planning: "how many calls fit on this link?" Pure math, offline.
  </p>

  <ToolHelp>
    The big number is total bandwidth for <strong>N concurrent calls</strong> including IP/UDP/RTP overhead — size
    the link above it, with headroom. A <strong>smaller packet interval</strong> (ptime) means more packets and
    more overhead per call; a lower-bitrate codec shrinks the payload. Use the <strong>Ethernet</strong> figure
    when the path is plain switched Ethernet rather than a WAN link.
  </ToolHelp>

  <div class="form">
    <label>Codec<select class="field" bind:value={codec}>{#each codecs as c}<option value={c.name}>{c.name} — {c.bitrateKbps} kbps</option>{/each}</select></label>
    <label>Packet interval (ms)<select class="field" bind:value={ptime}><option value={10}>10</option><option value={20}>20</option><option value={30}>30</option><option value={40}>40</option></select></label>
    <label>Concurrent calls<input class="field" type="number" min="1" bind:value={calls} /></label>
  </div>

  {#if error}<div class="hint">{error}</div>{/if}

  {#if res}
    <div class="total">
      <div class="tv mono">{mbps(res.totalKbps)}<span class="u">Mbps</span></div>
      <div class="tl">for {res.calls} concurrent call{res.calls === 1 ? "" : "s"} <span class="dim">(IP overhead; {mbps(res.totalEthKbps)} Mbps with Ethernet)</span></div>
    </div>
    <div class="grid">
      <div class="k">Payload</div><div class="v mono">{res.payloadBytes} B/packet · {res.packetsPerSec} pkt/s</div>
      <div class="k">One-way stream</div><div class="v mono">{res.oneWayKbps} kbps <span class="dim">({res.oneWayEthKbps} with Ethernet)</span></div>
      <div class="k">Per call (2-way)</div><div class="v mono">{res.perCallKbps} kbps</div>
    </div>
  {/if}
</div>

<style>
  .tool { padding: 16px 20px 40px; max-width: 720px; }
  h2 { margin: 0 0 2px; font-size: 1.15rem; font-weight: 600; color: var(--text-strong); }
  .sub { margin: 0 0 15px; color: var(--text-dim); font-size: 0.86rem; line-height: 1.5; }
  .form { display: flex; gap: 12px; flex-wrap: wrap; margin-bottom: 8px; }
  label { display: flex; flex-direction: column; gap: 4px; font-size: 0.78rem; color: var(--text-dim); }
  .field { background: var(--well); border: 1px solid var(--field-border); border-radius: 7px; padding: 8px 10px; color: var(--text); font-size: 0.86rem; outline: none; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  input.field { width: 110px; }
  .hint { background: var(--warn-bg); border: 1px solid var(--warn-border); color: var(--warn-text); border-radius: 7px; padding: 9px 12px; font-size: 0.82rem; margin: 10px 0; }
  .total { margin: 18px 0 14px; padding: 16px 18px; border: 1px solid var(--border); border-left: 4px solid var(--accent); border-radius: 12px; background: var(--card); }
  .tv { font-size: 2.2rem; font-weight: 700; color: var(--text-strong); line-height: 1; }
  .tv .u { font-size: 1rem; color: var(--text-dim); font-weight: 400; margin-left: 6px; }
  .tl { font-size: 0.84rem; color: var(--text); margin-top: 6px; }
  .grid { display: grid; grid-template-columns: max-content 1fr; gap: 5px 16px; font-size: 0.84rem; align-items: baseline; }
  .k { color: var(--text-dim); white-space: nowrap; } .v { color: var(--text); }
  .dim { color: var(--text-dim); }
  .mono { font-family: var(--font-mono); }
</style>
