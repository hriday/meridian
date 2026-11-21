<script lang="ts">
  import { PeripheralsSurvey } from "../../../wailsjs/go/app/App.js";
  import ToolHelp from "../ToolHelp.svelte";

  let survey: any = null;
  let loading = false;
  let err = "";

  async function run() {
    err = "";
    survey = null;
    loading = true;
    try {
      survey = await PeripheralsSurvey();
    } catch (e) {
      err = String(e);
    } finally {
      loading = false;
    }
  }
</script>

<div class="tool">
  <p class="lede">
    See the devices <strong>attached to this machine over USB</strong> right now — mice, keyboards,
    <strong>2.4&nbsp;GHz wireless receivers</strong> (the dongles that aren't Bluetooth), flash drives, serial
    adapters, cameras — each with its vendor and a few honest security call-outs. A point-in-time inventory;
    Meridian keeps no history.
  </p>

  <ToolHelp sudo="">
    This lists what's <strong>plugged into this computer</strong> — it can't see wireless devices in the air. A
    2.4&nbsp;GHz dongle uses a proprietary radio this machine has no receiver for, so it only appears here once it's
    physically connected. The <strong>flags are prompts to verify, not verdicts</strong>: a malicious "keyboard"
    (BadUSB) looks like a real keyboard to the OS, so Meridian can only point at things worth a second look — an
    unrecognized vendor, a receiver that can hide paired devices, a device that can type keystrokes from a source
    you may not know. <strong>No flags does not mean safe.</strong>
  </ToolHelp>

  <div class="searchrow">
    <button class="btn" on:click={run} disabled={loading}>{loading ? "Scanning…" : "Scan"}</button>
  </div>

  {#if err}<div class="banner bad"><span>{err}</span></div>{/if}

  {#if survey}
    <div class="sumrow">
      <div class="sv"><span class="mono">{survey.count}</span> device{survey.count === 1 ? "" : "s"}</div>
      <div class="dim sm">
        {#if survey.flagged}<span class="flagcount">{survey.flagged} flagged</span> · {/if}attached over USB
      </div>
    </div>
    {#if survey.note}<div class="banner warn"><span>{survey.note}</span></div>{/if}

    {#if survey.devices && survey.devices.length}
      <div class="card">
        {#each survey.devices as d}
          <div class="prow">
            <div class="phead">
              <span class="kind">{d.kind}</span>
              <span class="pn">{d.name}</span>
              {#if d.vendor}<span class="co">{d.vendor}</span>{:else}<span class="co unk">unknown vendor</span>{/if}
              {#if d.vendorId}<span class="ids mono">{d.vendorId}:{d.productId}</span>{/if}
            </div>
            {#if d.flags && d.flags.length}
              <div class="flags">
                {#each d.flags as f}
                  <div class="flag {f.level}"><span class="fdot"></span><span>{f.text}</span></div>
                {/each}
              </div>
            {/if}
          </div>
        {/each}
      </div>
    {:else}
      <div class="empty">
        No USB peripherals attached. A wireless mouse/keyboard dongle only shows up here once it's plugged into a
        port — this can't see devices over the air.
      </div>
    {/if}
    <p class="foot">
      No flags isn't a clean bill of health — a malicious device can look ordinary. Point-in-time; Meridian keeps no history.
    </p>
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 30px; }
  .lede { max-width: 820px; margin: 0 0 12px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .mono { font-family: var(--font-mono, ui-monospace, monospace); }
  .searchrow { display: flex; align-items: center; gap: 12px; max-width: 720px; margin-bottom: 14px; }
  .btn { background: var(--accent); color: var(--on-accent, #052e2b); border: 1px solid transparent; border-radius: 8px; padding: 9px 18px; font: inherit; font-size: 0.86rem; font-weight: 550; cursor: pointer; white-space: nowrap; }
  .btn:disabled { opacity: .5; cursor: default; }
  .banner { display: flex; align-items: center; gap: 10px; max-width: 820px; padding: 9px 13px; border-radius: 9px; font-size: 0.84rem; margin-bottom: 12px; }
  .banner.bad { border: 1px solid var(--bad, #ff6b6b); background: color-mix(in srgb, var(--bad, #ff6b6b) 10%, transparent); color: var(--text); }
  .banner.warn { border: 1px solid var(--warn, #d6a531); background: color-mix(in srgb, var(--warn, #d6a531) 12%, transparent); color: var(--text); }

  .sumrow { display: flex; align-items: baseline; gap: 14px; max-width: 820px; margin-bottom: 12px; }
  .sv { font-size: 1.05rem; font-weight: 650; color: var(--text-strong, var(--text)); }
  .dim { color: var(--text-dim); } .sm { font-size: 0.78rem; }
  .flagcount { color: var(--warn, #d6a531); font-weight: 600; }

  .card { background: var(--card); border: 1px solid var(--border); border-radius: 11px; max-width: 820px; overflow: hidden; }
  .prow { padding: 11px 14px; border-bottom: 1px solid var(--border-soft, var(--border)); }
  .prow:last-child { border-bottom: 0; }
  .phead { display: flex; align-items: baseline; gap: 10px; flex-wrap: wrap; }
  .kind { font-size: 0.68rem; text-transform: uppercase; letter-spacing: 0.04em; color: var(--text-faint, var(--text-dim)); min-width: 118px; }
  .pn { font-weight: 600; color: var(--text); font-size: 0.86rem; }
  .co { font-size: 0.68rem; color: var(--text-faint, var(--text-dim)); border: 1px solid var(--border); border-radius: 20px; padding: 1px 8px; }
  .co.unk { color: var(--warn, #d6a531); border-color: var(--warn, #d6a531); }
  .ids { margin-left: auto; font-size: 0.72rem; color: var(--text-dim); }

  .flags { margin-top: 7px; display: flex; flex-direction: column; gap: 5px; }
  .flag { display: flex; gap: 8px; align-items: baseline; font-size: 0.78rem; line-height: 1.4; padding: 6px 9px; border-radius: 8px; }
  .flag.watch { background: color-mix(in srgb, var(--bad, #f0616a) 9%, transparent); color: var(--text); }
  .flag.info { background: var(--well, color-mix(in srgb, var(--text) 5%, transparent)); color: var(--text-dim); }
  .fdot { width: 7px; height: 7px; border-radius: 50%; flex: none; margin-top: 5px; }
  .flag.watch .fdot { background: var(--bad, #f0616a); }
  .flag.info .fdot { background: var(--text-faint, #888); }

  .empty { max-width: 820px; padding: 14px; border: 1px dashed var(--border); border-radius: 11px; font-size: 0.82rem; color: var(--text-dim); }
  .foot { max-width: 820px; font-size: 0.74rem; color: var(--text-faint, var(--text-dim)); margin: 12px 0 0; }
</style>
