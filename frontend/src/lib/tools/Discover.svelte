<script lang="ts">
  import { DiscoverLAN } from "../../../wailsjs/go/app/App.js";

  let seconds = 4;
  let loading = false;
  let error = "";
  let res: any = null;

  async function run() {
    if (loading) return;
    loading = true;
    error = "";
    res = null;
    try {
      res = await DiscoverLAN(Number(seconds) || 4);
    } catch (e: any) {
      error = String(e?.message ?? e);
    } finally {
      loading = false;
    }
  }

  const prettyType = (t: string) => (t || "").replace(/^_/, "").replace(/\._tcp$|\._udp$/, "");
  const nonEmpty = (r: any) => r && ((r.services && r.services.length) || (r.devices && r.devices.length));
</script>

<div class="tool">
  <p class="lede">
    Find devices on your LAN by <strong>what they advertise</strong>: <strong>mDNS / Bonjour</strong> (printers,
    Chromecasts, AirPlay, Sonos, HomeKit, NAS, <span class="mono">_ssh</span>/<span class="mono">_smb</span>…) and
    <strong>SSDP / UPnP</strong> (routers, smart TVs, media servers, IoT). This is the "what is this device, and what
    does it offer" layer on top of a ping or ARP sweep. Multicast, no privileges, point-in-time.
  </p>

  <div class="row">
    <label>Listen for <input class="field" type="number" min="1" max="15" bind:value={seconds} /> s</label>
    <button class="btn" on:click={run} disabled={loading}>{loading ? `Discovering ${seconds}s…` : "Discover"}</button>
  </div>
  <p class="muted">You only hear devices on your own link — run it on the same LAN segment, not over a VPN or a routed/guest network that blocks multicast.</p>

  {#if error}<p class="err">{error}</p>{/if}

  {#if res}
    {#if !nonEmpty(res)}
      <div class="empty">Nothing answered. If you expected devices, check you're on the LAN (not a VPN/guest network) and that mDNS/SSDP isn't filtered.</div>
    {:else}
      {#if res.services && res.services.length}
        <div class="sec">
          <div class="sechead">mDNS / Bonjour <span class="count">{res.services.length}</span></div>
          {#each res.services as s}
            <div class="card">
              <div class="chead">
                <span class="name">{s.instance || s.host || "(unnamed)"}</span>
                <span class="badge">{prettyType(s.type)}</span>
              </div>
              <div class="meta mono">
                {#if s.addrs && s.addrs.length}{s.addrs.join(", ")}{:else}{s.host}{/if}{#if s.port}:{s.port}{/if}
              </div>
              {#if s.info && s.info.length}
                <div class="txt">{#each s.info.slice(0, 5) as t}<span class="chip mono">{t}</span>{/each}</div>
              {/if}
            </div>
          {/each}
        </div>
      {/if}

      {#if res.devices && res.devices.length}
        <div class="sec">
          <div class="sechead">SSDP / UPnP <span class="count">{res.devices.length}</span></div>
          {#each res.devices as d}
            <div class="card">
              <div class="chead">
                <span class="name mono">{d.address}</span>
                <span class="badge alt">{d.type}</span>
              </div>
              {#if d.server}<div class="meta">{d.server}</div>{/if}
              {#if d.location}<div class="meta mono small">{d.location}</div>{/if}
            </div>
          {/each}
        </div>
      {/if}
    {/if}
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 30px; max-width: 880px; }
  .lede { margin: 0 0 14px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .row { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; }
  label { font-size: 0.84rem; color: var(--text-dim); display: flex; align-items: center; gap: 7px; }
  .field { background: var(--well); border: 1px solid var(--field-border, var(--border)); border-radius: 7px; padding: 7px 9px; color: var(--text); font-size: 0.86rem; outline: none; width: 58px; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .btn { background: var(--accent); color: var(--on-accent, #052e2b); border: 1px solid transparent; border-radius: 8px; padding: 8px 16px; font: inherit; font-size: 0.86rem; font-weight: 550; cursor: pointer; }
  .btn:disabled { opacity: .55; cursor: default; }
  .muted { color: var(--text-dim); font-size: 0.76rem; margin: 8px 0 0; line-height: 1.5; }
  .err { color: var(--danger, #e5484d); font-size: 0.82rem; margin: 12px 0 0; }
  .empty { margin-top: 16px; padding: 12px 15px; border: 1px solid var(--border); border-radius: 10px; background: var(--card); font-size: 0.84rem; color: var(--text-dim); }
  .sec { margin-top: 18px; }
  .sechead { font-size: 0.74rem; text-transform: uppercase; letter-spacing: .05em; color: var(--text-dim); font-weight: 700; margin-bottom: 8px; display: flex; align-items: center; gap: 8px; }
  .count { background: var(--well); border: 1px solid var(--border); border-radius: 20px; padding: 1px 8px; font-size: 0.72rem; color: var(--text); }
  .card { border: 1px solid var(--border); border-radius: 11px; padding: 11px 14px; margin-bottom: 9px; background: var(--card); }
  .chead { display: flex; align-items: center; justify-content: space-between; gap: 10px; }
  .name { font-size: 0.9rem; font-weight: 600; color: var(--text-strong, var(--text)); word-break: break-word; }
  .badge { font-size: 0.68rem; font-weight: 700; padding: 2px 8px; border-radius: 6px; background: var(--accent); color: var(--on-accent, #052e2b); white-space: nowrap; }
  .badge.alt { background: var(--well); color: var(--text-dim); border: 1px solid var(--border); font-weight: 600; }
  .meta { font-size: 0.8rem; color: var(--text-dim); margin-top: 4px; word-break: break-word; }
  .meta.small { font-size: 0.72rem; }
  .txt { margin-top: 7px; display: flex; gap: 6px; flex-wrap: wrap; }
  .chip { font-size: 0.68rem; background: var(--well); border: 1px solid var(--border); border-radius: 5px; padding: 1px 6px; color: var(--text-dim); }
  .mono { font-family: var(--font-mono, ui-monospace, monospace); }
</style>
