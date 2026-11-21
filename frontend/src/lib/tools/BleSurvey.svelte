<script lang="ts">
  import { BLESurvey } from "../../../wailsjs/go/app/App.js";
  import ToolHelp from "../ToolHelp.svelte";

  let seconds = 6;
  let survey: any = null;
  let loading = false;
  let err = "";

  async function run() {
    err = "";
    survey = null;
    loading = true;
    try {
      survey = await BLESurvey(seconds);
    } catch (e) {
      err = String(e);
    } finally {
      loading = false;
    }
  }

  // BLE RSSI (dBm) rule of thumb: ≥ -60 strong/near, down to about -80 usable,
  // below that is faint.
  function sig(rssi: number): string {
    if (rssi >= -60) return "good";
    if (rssi >= -80) return "warn";
    return "bad";
  }
  function proximity(rssi: number): string {
    if (rssi >= -55) return "very close";
    if (rssi >= -70) return "near";
    if (rssi >= -85) return "far";
    return "distant";
  }
</script>

<div class="tool">
  <p class="lede">
    Scan for the <strong>Bluetooth Low Energy</strong> devices advertising around you — phones, watches,
    earbuds, beacons, sensors — and list each with its <strong>signal</strong> (RSSI), name and advertised
    company. A point-in-time survey using this machine's own Bluetooth radio; nothing is connected to or paired.
  </p>

  <ToolHelp sudo="">
    Each row is one device, strongest signal first. The coloured dot and <strong>RSSI</strong> (in dBm, closer to
    zero = nearer) show how strong its advertisement is. <strong>Company</strong> comes from the manufacturer data
    in the advertisement (shown as a raw <code>0xNNNN</code> ID when it isn't a well-known one); many devices
    advertise no name at all. On macOS the first scan asks for Bluetooth access — allow it under
    <strong>System Settings › Privacy &amp; Security › Bluetooth</strong>. Addresses on macOS are per-host random
    identifiers, not hardware MACs (Apple privacy).
  </ToolHelp>

  <div class="searchrow">
    <label class="dur">Scan for
      <select bind:value={seconds} class="sel">
        <option value={4}>4s</option>
        <option value={6}>6s</option>
        <option value={10}>10s</option>
        <option value={20}>20s</option>
      </select>
    </label>
    <button class="btn" on:click={run} disabled={loading}>{loading ? "Scanning…" : "Scan"}</button>
  </div>

  {#if err}<div class="banner bad"><span>{err}</span></div>{/if}

  {#if survey}
    <div class="sumrow">
      <div class="sv"><span class="mono">{survey.count}</span> device{survey.count === 1 ? "" : "s"}</div>
      <div class="dim sm">{survey.named} named · scanned {survey.durationSec}s</div>
    </div>

    {#if survey.devices && survey.devices.length}
      <div class="card">
        {#each survey.devices as d}
          <div class="drow">
            <span class="ddot {sig(d.rssi)}" title={`${d.rssi} dBm`}></span>
            <div class="dbody">
              <div class="dhead">
                <span class="dn">{d.name || "(unnamed)"}</span>
                {#if d.company}<span class="co">{d.company}</span>{/if}
                <span class="rssi mono">{d.rssi} dBm · {proximity(d.rssi)}</span>
              </div>
              <div class="dmeta mono">{d.address}{d.seen > 1 ? `  ·  ${d.seen} adverts` : ""}</div>
            </div>
          </div>
        {/each}
      </div>
    {:else}
      <div class="empty">No BLE devices were advertising nearby during the scan. Make sure Bluetooth is on (and, on macOS, that Meridian has Bluetooth access).</div>
    {/if}
    <p class="foot">A point-in-time survey — Meridian keeps no history. RSSI is the strongest advertisement heard from each device over the scan window.</p>
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 30px; }
  .lede { max-width: 820px; margin: 0 0 12px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .mono { font-family: var(--font-mono, ui-monospace, monospace); }
  .searchrow { display: flex; align-items: center; gap: 12px; max-width: 720px; margin-bottom: 14px; }
  .dur { font-size: 0.84rem; color: var(--text-dim); display: inline-flex; align-items: center; gap: 8px; }
  .sel { background: var(--well); border: 1px solid var(--field-border, var(--border)); border-radius: 8px; padding: 7px 10px; color: var(--text); font: inherit; font-size: 0.84rem; outline: none; }
  .sel:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .btn { background: var(--accent); color: var(--on-accent, #052e2b); border: 1px solid transparent; border-radius: 8px; padding: 9px 18px; font: inherit; font-size: 0.86rem; font-weight: 550; cursor: pointer; white-space: nowrap; }
  .btn:disabled { opacity: .5; cursor: default; }
  .banner { display: flex; align-items: center; gap: 10px; max-width: 820px; padding: 9px 13px; border-radius: 9px; font-size: 0.84rem; margin-bottom: 12px; }
  .banner.bad { border: 1px solid var(--bad, #ff6b6b); background: color-mix(in srgb, var(--bad, #ff6b6b) 10%, transparent); color: var(--text); }

  .sumrow { display: flex; align-items: baseline; gap: 14px; max-width: 820px; margin-bottom: 12px; }
  .sv { font-size: 1.05rem; font-weight: 650; color: var(--text-strong, var(--text)); }
  .dim { color: var(--text-dim); } .sm { font-size: 0.78rem; }

  .card { background: var(--card); border: 1px solid var(--border); border-radius: 11px; max-width: 820px; overflow: hidden; }
  .drow { display: flex; gap: 11px; padding: 10px 14px; border-bottom: 1px solid var(--border-soft, var(--border)); }
  .drow:last-child { border-bottom: 0; }
  .ddot { width: 9px; height: 9px; border-radius: 50%; flex: none; margin-top: 5px; }
  .ddot.good { background: var(--good, #3fb950); }
  .ddot.warn { background: var(--warn, #d6a531); }
  .ddot.bad { background: var(--bad, #f0616a); }
  .dbody { min-width: 0; flex: 1; }
  .dhead { display: flex; align-items: baseline; gap: 10px; flex-wrap: wrap; }
  .dn { font-weight: 600; color: var(--text); font-size: 0.86rem; }
  .co { font-size: 0.68rem; color: var(--text-faint, var(--text-dim)); border: 1px solid var(--border); border-radius: 20px; padding: 1px 8px; }
  .rssi { margin-left: auto; font-size: 0.74rem; color: var(--text-dim); }
  .dmeta { font-size: 0.74rem; color: var(--text-dim); margin-top: 3px; overflow-wrap: anywhere; }

  .empty { max-width: 820px; padding: 14px; border: 1px dashed var(--border); border-radius: 11px; font-size: 0.82rem; color: var(--text-dim); }
  .foot { max-width: 820px; font-size: 0.74rem; color: var(--text-faint, var(--text-dim)); margin: 12px 0 0; }
</style>
