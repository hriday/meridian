<script lang="ts">
  import { MIBWalk } from "../../../wailsjs/go/app/App.js";

  const CAP = 5000;
  const presets: { label: string; oid: string }[] = [
    { label: "System", oid: "1.3.6.1.2.1.1" },
    { label: "Interfaces", oid: "1.3.6.1.2.1.2.2" },
    { label: "Interface stats", oid: "1.3.6.1.2.1.31.1.1" },
    { label: "Processors", oid: "1.3.6.1.2.1.25.3.3" },
    { label: "Storage", oid: "1.3.6.1.2.1.25.2.3" },
    { label: "IP addresses", oid: "1.3.6.1.2.1.4.20" },
  ];

  let device = "";
  let community = "";
  let oid = "1.3.6.1.2.1.1";
  let walking = false;
  let rows: any[] = [];
  let walkedOid = "";
  let err = "";

  async function walk() {
    const d = device.trim();
    const o = oid.trim();
    if (!d || !o || walking) return;
    err = ""; walking = true;
    try {
      rows = (await MIBWalk(d, community, o)) || [];
      walkedOid = o.replace(/^\./, "");
    } catch (e) {
      err = String(e);
      rows = [];
    } finally {
      walking = false;
    }
  }

  function pick(p: { oid: string }) {
    oid = p.oid;
    if (device.trim()) walk();
  }

  $: capped = rows.length >= CAP;
</script>

<div class="tool">
  <p class="lede">
    MIB browser — point it at a device and walk any part of its MIB tree: enter an OID (or pick a
    starting point) and see every value underneath, with its type. For when you need the raw truth below
    the dashboards — what a device actually exposes, the OID behind a number, or why a monitor reads what
    it reads. Read-only; the community is never shown back.
  </p>

  <form class="row" on:submit|preventDefault={walk}>
    <input class="field dev mono" bind:value={device} placeholder="10.0.0.1  ·  device" aria-label="Device" spellcheck="false" autocomplete="off" />
    <input class="field comm mono" type="password" bind:value={community} placeholder="community" aria-label="SNMP community" autocomplete="off" />
    <input class="field oid mono" bind:value={oid} placeholder="1.3.6.1.2.1.1" aria-label="OID" spellcheck="false" autocomplete="off" />
    <button type="submit" class="go" disabled={!device.trim() || !oid.trim() || walking}>{walking ? "Walking…" : "Walk"}</button>
  </form>

  <div class="chips">
    {#each presets as p}
      <button type="button" class="chip" class:on={oid === p.oid} on:click={() => pick(p)}>{p.label}</button>
    {/each}
  </div>

  <p class="help">
    Each row is one OID the device exposes under the point you walked — its numeric <strong>OID</strong>,
    its SNMP <strong>type</strong> (OctetString, Counter32, TimeTicks…), and its current <strong>value</strong>.
    A <strong>Walk</strong> of a branch returns everything beneath it; a full instance OID (ending in a
    number like <span class="mono">…1.5.0</span>) returns that single value. Binary values such as a MAC
    show as hex.
  </p>

  {#if err}
    <div class="banner bad"><span class="vlabel">Couldn't walk</span><span class="vnote">{err}</span></div>
  {/if}

  {#if rows.length}
    <div class="statusbar">
      <span><span class="count">{rows.length}{capped ? "+" : ""}</span> OID{rows.length === 1 ? "" : "s"} under <span class="root mono">{walkedOid}</span></span>
    </div>
    <div class="grid">
      <div class="head"><span>OID</span><span>Type</span><span>Value</span></div>
      {#each rows as r}
        <div class="drow">
          <div class="oidc mono">{r.oid}</div>
          <div><span class="pill mono">{r.type}</span></div>
          <div class="val mono" title={r.value}>{r.value || "—"}</div>
        </div>
      {/each}
    </div>
    {#if capped}
      <p class="caption">Showing the first {CAP} OIDs — the subtree is larger. Walk a deeper OID to narrow it.</p>
    {:else}
      <p class="caption">Walked {walkedOid} · v2c</p>
    {/if}
  {:else if !walking && walkedOid && !err}
    <p class="caption">Nothing under {walkedOid} — the device may not implement that branch.</p>
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 20px; }
  .lede { max-width: 900px; margin: 0 0 14px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .mono { font-family: var(--mono, ui-monospace, monospace); }
  .row { display: flex; gap: 8px; max-width: 900px; align-items: center; }
  .field { background: var(--well); border: 1px solid var(--field-border); border-radius: 7px; padding: 9px 12px; color: var(--text); font-size: 0.88rem; outline: none; }
  .field.dev { flex: 1; }
  .field.comm { flex: 0 0 130px; }
  .field.oid { flex: 1.4; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .go { background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px; padding: 0 16px; height: 38px; font: inherit; font-size: 0.88rem; cursor: pointer; }
  .go:disabled { opacity: .6; cursor: default; }

  .chips { display: flex; gap: 7px; flex-wrap: wrap; max-width: 900px; margin: 10px 0 2px; }
  .chip { font-size: 0.72rem; color: var(--text-dim); background: transparent; border: 1px solid var(--pill-border); border-radius: 20px; padding: 3px 11px; cursor: pointer; font-family: inherit; }
  .chip:hover { color: var(--text); border-color: var(--field-border); }
  .chip.on { color: var(--text-strong); border-color: var(--accent); background: var(--active); }

  .help { max-width: 900px; margin: 12px 0 0; padding: 9px 13px; border-left: 2px solid var(--accent); background: var(--well); border-radius: 0 7px 7px 0; font-size: 0.78rem; color: var(--text-dim); line-height: 1.5; }
  .help strong { color: var(--text); font-weight: 600; }

  .banner { margin: 14px 0; padding: 11px 14px; border-radius: 9px; border: 1px solid var(--bad, #ff6b6b); background: color-mix(in srgb, var(--bad,#ff6b6b) 10%, transparent); max-width: 900px; display: flex; flex-direction: column; gap: 3px; }
  .vlabel { font-weight: 600; color: var(--bad, #ff6b6b); }
  .vnote { font-size: 0.8rem; color: var(--text-dim); word-break: break-word; }

  .statusbar { display: flex; align-items: center; gap: 10px; margin: 16px 0 6px; max-width: 900px; font-size: 0.8rem; color: var(--text-dim); }
  .count { color: var(--text-strong); font-weight: 600; }
  .root { color: var(--text-faint); }

  .grid { max-width: 900px; border: 1px solid var(--border-soft); border-radius: 11px; overflow: hidden; background: var(--card); }
  .head, .drow { display: grid; grid-template-columns: 2.1fr 1fr 3.4fr; align-items: center; gap: 14px; padding: 8px 16px; }
  .head { font-size: 0.6rem; text-transform: uppercase; letter-spacing: .08em; color: var(--text-faint); border-bottom: 1px solid var(--border-soft); background: var(--panel); }
  .drow { border-top: 1px solid var(--border-soft); font-size: 0.8rem; }
  .drow:first-of-type { border-top: none; }
  .oidc { color: var(--text-strong); }
  .pill { font-size: 0.64rem; padding: 1px 8px; border-radius: 20px; border: 1px solid var(--pill-border); color: var(--text-dim); }
  .val { color: var(--text-dim); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .caption { max-width: 900px; margin-top: 12px; font-size: 0.72rem; color: var(--text-faint); }
</style>
