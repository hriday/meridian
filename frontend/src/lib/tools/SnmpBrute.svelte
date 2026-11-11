<script lang="ts">
  import { SnmpBruteCommunities } from "../../../wailsjs/go/app/App.js";

  const DEFAULTS = [
    "public", "private", "cisco", "admin", "snmp", "read", "write",
    "manager", "monitor", "community", "default", "security", "network",
    "router", "switch", "test", "guest", "secret", "access", "root",
  ].join("\n");

  let device = "";
  let list = DEFAULTS;
  let authorized = false;
  let loading = false;
  let ran = false;
  let err = "";
  let hits: string[] = [];
  let tried = 0;

  $: communityCount = list.split(/\r?\n/).map((s) => s.trim()).filter(Boolean).length;

  async function run() {
    const d = device.trim();
    const communities = list.split(/\r?\n/).map((s) => s.trim()).filter(Boolean);
    if (!d || !authorized || loading || communities.length === 0) return;
    err = "";
    ran = false;
    loading = true;
    tried = new Set(communities).size;
    try {
      hits = (await SnmpBruteCommunities(d, communities, authorized)) ?? [];
      ran = true;
    } catch (e) {
      err = String(e);
    } finally {
      loading = false;
    }
  }
</script>

<div class="tool">
  <p class="lede">
    Audit the SNMP community strings on <strong>your own</strong> gear — find weak or
    default read communities before someone else does. Meridian tries each string with
    one read-only <span class="mono">sysDescr</span> GET (SNMPv2c); it never writes and
    never changes the device, and the strings are used only to connect.
  </p>

  <div class="warn">
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 9v4m0 4h.01M10.3 3.9 1.8 18a2 2 0 0 0 1.7 3h17a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0Z"/></svg>
    <span>Only run this against devices you own or are explicitly authorized to test. Unauthorized scanning may be illegal.</span>
  </div>

  <div class="card span">
    <div class="grid">
      <label class="f"><span class="k0">Device IP / host</span>
        <input class="field mono" bind:value={device} placeholder="192.168.1.1" spellcheck="false" autocomplete="off" disabled={loading} /></label>
      <label class="f wide"><span class="k0">Community strings (one per line)</span>
        <textarea class="field mono small" bind:value={list} spellcheck="false" disabled={loading}></textarea></label>
    </div>
    <label class="auth">
      <input type="checkbox" bind:checked={authorized} disabled={loading} />
      <span>I am authorized to test this device.</span>
    </label>
    <div class="row">
      <button type="button" class="go" on:click={run} disabled={!device.trim() || !authorized || loading || communityCount === 0}>{loading ? "Testing…" : "Audit communities"}</button>
      <span class="hint">Read-only; one sysDescr GET per string · SNMPv2c, UDP 161.</span>
    </div>
  </div>

  {#if err}
    <div class="banner bad"><span class="vlabel">Couldn't run the audit</span><span class="vnote">{err}</span></div>
  {/if}

  {#if ran}
    {#if hits.length === 0}
      <div class="ok">No community string granted read access across {tried} tried. Good — no weak/default read community found (that these strings cover).</div>
    {:else}
      <div class="summary"><span class="exp mono">{hits.length}</span> working community string{hits.length === 1 ? "" : "s"} out of {tried} tried — these grant SNMP read access:</div>
      <div class="card span">
        <ul class="hits">
          {#each hits as h}
            <li class="mono"><span class="dot"></span>{h}</li>
          {/each}
        </ul>
      </div>
    {/if}
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 20px; }
  .lede { max-width: 820px; margin: 0 0 12px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .mono { font-family: var(--mono, ui-monospace, monospace); }
  .warn { display: flex; gap: 9px; align-items: flex-start; max-width: 820px; margin-bottom: 14px; padding: 10px 13px; border-radius: 9px;
    border: 1px solid color-mix(in srgb, var(--warn,#d6a531) 45%, var(--border)); background: color-mix(in srgb, var(--warn,#d6a531) 9%, transparent);
    font-size: 0.8rem; color: var(--text-dim); }
  .warn svg { width: 16px; height: 16px; flex: none; color: var(--warn, #d6a531); margin-top: 1px; }
  .card { background: var(--card); border: 1px solid var(--border-soft); border-radius: 11px; padding: 13px 15px; max-width: 820px; margin-bottom: 14px; }
  .grid { display: flex; flex-wrap: wrap; gap: 10px; }
  .f { display: flex; flex-direction: column; gap: 5px; flex: 1 1 200px; min-width: 0; }
  .f.wide { flex: 2 1 320px; }
  .k0 { font-size: 0.68rem; text-transform: uppercase; letter-spacing: .05em; color: var(--text-faint); }
  .field { background: var(--well); border: 1px solid var(--field-border); border-radius: 7px; padding: 8px 11px; color: var(--text); font-size: 0.86rem; outline: none; width: 100%; }
  .field.small { font-size: 0.78rem; min-height: 120px; resize: vertical; line-height: 1.5; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .field:disabled { opacity: .6; }
  .auth { display: flex; align-items: center; gap: 8px; margin: 12px 0 2px; font-size: 0.84rem; color: var(--text); }
  .row { display: flex; gap: 10px; margin-top: 12px; align-items: center; flex-wrap: wrap; }
  .go { background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px; padding: 0 16px; height: 38px; font: inherit; font-size: 0.9rem; cursor: pointer; }
  .go:disabled { opacity: .55; cursor: default; }
  .hint { font-size: 0.75rem; color: var(--text-faint); }

  .banner { margin: 0 0 14px; padding: 11px 14px; border-radius: 9px; border: 1px solid var(--bad, #ff6b6b); background: color-mix(in srgb, var(--bad,#ff6b6b) 10%, transparent); max-width: 820px; display: flex; flex-direction: column; gap: 3px; }
  .vlabel { font-weight: 600; color: var(--bad, #ff6b6b); }
  .vnote { font-size: 0.8rem; color: var(--text-dim); overflow-wrap: anywhere; }

  .ok { max-width: 820px; color: var(--good, #3fb950); font-size: 0.85rem; padding: 8px 0; }
  .summary { margin: 2px 0 10px; font-size: 0.84rem; color: var(--text-dim); max-width: 820px; }
  .summary .exp { color: var(--bad, #f0616a); font-weight: 650; }
  .hits { list-style: none; margin: 0; padding: 4px 2px; display: flex; flex-direction: column; gap: 4px; }
  .hits li { display: flex; align-items: center; gap: 9px; font-size: 0.86rem; color: var(--bad, #f0616a); font-weight: 600; }
  .hits .dot { width: 7px; height: 7px; border-radius: 50%; background: var(--bad, #f0616a); flex: none; }
</style>
