<script lang="ts">
  import { SwitchPortMap } from "../../../wailsjs/go/app/App.js";

  let device = "";
  let community = "";
  let rows: any[] = [];
  let loading = false;
  let ran = false;
  let err = "";

  async function run() {
    const d = device.trim();
    if (!d || loading) return;
    err = "";
    ran = false;
    loading = true;
    try {
      rows = (await SwitchPortMap(d, community)) ?? [];
      ran = true;
    } catch (e) {
      err = String(e);
    } finally {
      loading = false;
    }
  }

  $: ports = new Set(rows.map((r) => r.port)).size;
</script>

<div class="tool">
  <p class="lede">
    Map which MAC (and IP) lives on which physical switch port. Meridian walks the
    switch's SNMP forwarding database and joins it to the interface names and ARP
    table. SNMPv2c read-only; the community string is used only to connect and is
    never logged.
  </p>

  <div class="card span">
    <div class="grid">
      <label class="f"><span class="k0">Switch IP / host</span>
        <input class="field mono" bind:value={device} placeholder="192.168.1.1" spellcheck="false" autocomplete="off" disabled={loading} /></label>
      <label class="f sm"><span class="k0">Community</span>
        <input class="field mono" type="password" bind:value={community} placeholder="public" autocomplete="off" disabled={loading} /></label>
    </div>
    <div class="row">
      <button type="button" class="go" on:click={run} disabled={!device.trim() || loading}>{loading ? "Walking…" : "Map ports"}</button>
      <span class="hint">Read-only SNMP walk of the BRIDGE-MIB, IF-MIB and ARP table.</span>
    </div>
  </div>

  {#if err}
    <div class="banner bad"><span class="vlabel">Couldn't map the switch</span><span class="vnote">{err}</span></div>
  {/if}

  {#if ran}
    {#if rows.length === 0}
      <div class="nochg">No forwarding entries returned — check SNMP access and that this is a switch.</div>
    {:else}
      <div class="summary"><span class="count mono">{rows.length}</span> MAC{rows.length === 1 ? "" : "s"} across <span class="count mono">{ports}</span> port{ports === 1 ? "" : "s"}</div>
      <div class="card span">
        <div class="tablewrap">
          <table>
            <thead><tr><th>Port</th><th>VLAN</th><th>MAC</th><th>IP</th></tr></thead>
            <tbody>
              {#each rows as r}
                <tr>
                  <td class="mono port">{r.port}</td>
                  <td class="mono vlan">{r.vlan || "—"}</td>
                  <td class="mono mac">{r.mac}</td>
                  <td class="mono ip">{r.ip || "—"}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </div>
    {/if}
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 20px; }
  .lede { max-width: 820px; margin: 0 0 14px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .mono { font-family: var(--mono, ui-monospace, monospace); }
  .card { background: var(--card); border: 1px solid var(--border-soft); border-radius: 11px; padding: 13px 15px; max-width: 820px; margin-bottom: 14px; }
  .grid { display: flex; flex-wrap: wrap; gap: 10px; }
  .f { display: flex; flex-direction: column; gap: 5px; flex: 1 1 200px; min-width: 0; }
  .f.sm { flex: 0 0 160px; }
  .k0 { font-size: 0.68rem; text-transform: uppercase; letter-spacing: .05em; color: var(--text-faint); }
  .field { background: var(--well); border: 1px solid var(--field-border); border-radius: 7px; padding: 8px 11px; color: var(--text); font-size: 0.86rem; outline: none; width: 100%; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .field:disabled { opacity: .6; }
  .row { display: flex; gap: 10px; margin-top: 12px; align-items: center; flex-wrap: wrap; }
  .go { background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px; padding: 0 16px; height: 38px; font: inherit; font-size: 0.9rem; cursor: pointer; }
  .go:disabled { opacity: .6; cursor: default; }
  .hint { font-size: 0.75rem; color: var(--text-faint); }

  .banner { margin: 0 0 14px; padding: 11px 14px; border-radius: 9px; border: 1px solid var(--bad, #ff6b6b); background: color-mix(in srgb, var(--bad,#ff6b6b) 10%, transparent); max-width: 820px; display: flex; flex-direction: column; gap: 3px; }
  .vlabel { font-weight: 600; color: var(--bad, #ff6b6b); }
  .vnote { font-size: 0.8rem; color: var(--text-dim); overflow-wrap: anywhere; }

  .summary { margin: 2px 0 10px; font-size: 0.84rem; color: var(--text-dim); }
  .summary .count { color: var(--text-strong); font-weight: 650; }
  .nochg { color: var(--text-dim); font-size: 0.85rem; padding: 8px 0; }

  .tablewrap { overflow-x: auto; }
  table { width: 100%; border-collapse: collapse; font-size: 0.82rem; min-width: 480px; }
  thead th { text-align: left; font-size: 0.66rem; text-transform: uppercase; letter-spacing: .05em; color: var(--text-faint); font-weight: 600; padding: 8px 12px; border-bottom: 1px solid var(--border); }
  tbody td { padding: 7px 12px; border-bottom: 1px solid var(--border-soft); }
  tbody tr:last-child td { border-bottom: 0; }
  tbody tr:hover { background: var(--card2, var(--well)); }
  .port { color: var(--text-strong); }
  .vlan { color: var(--text-dim); }
  .mac { color: var(--text); }
  .ip { color: var(--text-dim); }
</style>
