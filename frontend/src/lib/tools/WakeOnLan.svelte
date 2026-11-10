<script lang="ts">
  import { WakeOnLAN } from "../../../wailsjs/go/app/App.js";

  let mac = "";
  let bcast = "255.255.255.255";
  let port = 9;
  let busy = false;
  let error = "";
  let sent = "";

  async function run() {
    const m = mac.trim();
    if (!m) {
      sent = "";
      return;
    }
    error = "";
    sent = "";
    busy = true;
    try {
      sent = await WakeOnLAN(m, bcast.trim(), Number(port) || 9);
    } catch (e) {
      error = String(e);
    } finally {
      busy = false;
    }
  }
</script>

<div class="tool">
  <p class="lede">
    Powers on a machine by its MAC address with a Wake-on-LAN "magic packet" — a UDP
    broadcast carrying the MAC. Two reasons it may not wake the target:
    <b>Wake-on-LAN must be enabled</b> in its BIOS/firmware and network card, and a
    broadcast <b>doesn't cross subnets</b> — send from the same LAN, or set the
    broadcast address of the target's subnet (e.g. <span class="mono">192.168.1.255</span>).
  </p>

  <form on:submit|preventDefault={run} class="grid">
    <label>MAC address
      <input class="field mono" bind:value={mac} placeholder="aa:bb:cc:dd:ee:ff" autocomplete="off" spellcheck="false" aria-label="MAC address" />
    </label>
    <label>Broadcast
      <input class="field mono" bind:value={bcast} placeholder="255.255.255.255" autocomplete="off" spellcheck="false" aria-label="broadcast address" />
    </label>
    <label>Port
      <input class="field mono port" type="number" bind:value={port} min="1" max="65535" aria-label="port" />
    </label>
    <button class="go" disabled={busy}>{busy ? "Sending…" : "Wake"}</button>
  </form>

  {#if error}<div class="banner bad"><span class="vlabel">Couldn't send</span><span class="vnote">{error}</span></div>{/if}

  {#if sent}
    <div class="banner good">
      <span class="vlabel">{sent}</span>
      <span class="vnote">"Sent" isn't "woke" — Meridian can't tell whether the machine powered on. Give it a few seconds, then check if it's up.</span>
    </div>
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 20px; }
  .lede { max-width: 760px; margin: 0 0 14px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .grid { display: flex; gap: 10px; align-items: flex-end; flex-wrap: wrap; max-width: 760px; }
  label { display: flex; flex-direction: column; gap: 5px; font-size: 0.76rem; color: var(--text-dim); }
  .field {
    background: var(--well); border: 1px solid var(--field-border); border-radius: 7px;
    padding: 9px 12px; color: var(--text); font-size: 0.9rem; outline: none; min-width: 220px;
  }
  .field.port { min-width: 90px; width: 90px; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .mono { font-family: var(--mono, ui-monospace, monospace); }
  .go {
    background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px;
    padding: 0 16px; height: 37px; font: inherit; font-size: 0.9rem; cursor: pointer; white-space: nowrap;
  }
  .go:disabled { opacity: .6; cursor: default; }

  .banner {
    margin: 16px 0 0; padding: 11px 14px; border-radius: 9px; border: 1px solid var(--border-soft);
    display: flex; flex-direction: column; gap: 3px; max-width: 760px;
  }
  .banner.good { border-color: transparent; background: var(--good-bg, #102516); }
  .banner.bad { border-color: var(--bad, #ff6b6b); background: color-mix(in srgb, var(--bad, #ff6b6b) 10%, transparent); }
  .vlabel { font-weight: 600; color: var(--text-strong); }
  .banner.good .vlabel { color: var(--good, #5ad17a); }
  .banner.bad .vlabel { color: var(--bad, #ff6b6b); }
  .vnote { font-size: 0.82rem; color: var(--text-dim); }
</style>
