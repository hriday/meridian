<script>
  import { onMount } from "svelte";
  import { LocalNetwork, STUNDiscover } from "../../wailsjs/go/app/App.js";

  let local = null;             // myip.Info
  let publicIP = "";
  let natType = "";
  let publicState = "loading";  // loading | ok | error
  let open = false;
  let copied = "";

  const isStr = (v) => typeof v === "string" && v.length > 0;

  async function loadLocal() {
    try {
      const info = await LocalNetwork();
      // Guard against the e2e proxy stub: only accept a real-looking result.
      if (info && (isStr(info.primaryIP) || Array.isArray(info.interfaces))) local = info;
    } catch (_) { /* leave local null -> renders "…" */ }
  }

  async function loadPublic() {
    try {
      const r = await STUNDiscover("");
      if (r && isStr(r.publicIP)) {
        publicIP = r.publicIP;
        natType = isStr(r.natType) ? r.natType : "";
        publicState = "ok";
      } else {
        publicState = "error";
      }
    } catch (_) {
      publicState = "error"; // a failed lookup is "n/a", never a fake 0
    }
  }

  onMount(() => { loadLocal(); loadPublic(); });

  async function copy(v) {
    if (!isStr(v)) return;
    try {
      await navigator.clipboard.writeText(v);
      copied = v;
      setTimeout(() => (copied = ""), 1200);
    } catch (_) { /* clipboard blocked; ignore */ }
  }

  $: primaryIP = local && isStr(local.primaryIP) ? local.primaryIP : "";
  $: primaryIf = local && isStr(local.primaryIf) ? local.primaryIf : "";
  $: gateway = local && isStr(local.gateway) ? local.gateway : "";
  $: ifaces = local && Array.isArray(local.interfaces) ? local.interfaces : [];
  $: pubLabel = publicState === "ok" ? publicIP : publicState === "error" ? "n/a" : "…";
</script>

<div class="ipchip" class:open>
  <button class="chip mono" on:click={() => (open = !open)} title="Your IP addresses" aria-expanded={open}>
    <span class="glyph">⬡</span><span class="val">{primaryIP || "…"}</span>
    <span class="dot">·</span>
    <span class="glyph up">↗</span><span class="val" class:dim={publicState !== "ok"}>{pubLabel}</span>
  </button>

  {#if open}
    <button class="scrim" on:click={() => (open = false)} aria-label="Close IP panel" tabindex="-1"></button>
    <div class="pop mono" role="dialog" aria-label="IP addresses">
      <div class="phead">This device</div>
      {#each ifaces as f}
        <div class="prow">
          <span class="k">{f.name}{f.name === primaryIf ? " ·" : ""}</span>
          <button class="v" on:click={() => copy((f.ipv4 || [])[0])} title="Copy">{(f.ipv4 || []).join(", ")}</button>
        </div>
      {/each}
      {#if gateway}
        <div class="prow"><span class="k">Gateway</span><button class="v" on:click={() => copy(gateway)} title="Copy">{gateway}</button></div>
      {/if}
      <div class="prow top">
        <span class="k">Public</span>
        <button class="v" class:dim={publicState !== "ok"} on:click={() => copy(publicIP)} title="Copy">{pubLabel}</button>
      </div>
      {#if natType}<div class="prow"><span class="k">NAT</span><span class="v flat">{natType}</span></div>{/if}
      <div class="foot">{copied ? `copied ${copied}` : "click a value to copy"}</div>
    </div>
  {/if}
</div>

<style>
  .ipchip { position: relative; flex: none; }
  .chip {
    display: inline-flex; align-items: center; gap: 5px;
    background: var(--well); border: 1px solid var(--border); border-radius: 8px;
    padding: 5px 10px; color: var(--text); font-size: 0.78rem; cursor: pointer; white-space: nowrap;
  }
  .chip:hover { border-color: var(--accent); }
  .ipchip.open .chip { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .glyph { color: var(--accent); font-size: 0.82rem; }
  .glyph.up { color: var(--text-dim); }
  .val { font-weight: 550; }
  .val.dim { color: var(--text-dim); font-weight: 450; }
  .dot { color: var(--text-dim); margin: 0 2px; }

  .scrim { position: fixed; inset: 0; background: transparent; border: none; cursor: default; z-index: 40; }
  .pop {
    position: absolute; top: calc(100% + 7px); left: 0; z-index: 41; min-width: 240px;
    background: var(--card); border: 1px solid var(--border); border-radius: 10px;
    box-shadow: 0 10px 30px rgba(0,0,0,.28); padding: 8px; font-size: 0.78rem;
  }
  .phead { font-size: 0.64rem; text-transform: uppercase; letter-spacing: .06em; color: var(--text-dim); padding: 2px 6px 6px; }
  .prow { display: flex; align-items: center; gap: 12px; padding: 3px 6px; }
  .prow.top { margin-top: 6px; border-top: 1px solid var(--border); padding-top: 8px; }
  .k { color: var(--text-dim); flex: none; min-width: 74px; }
  .v {
    margin-left: auto; color: var(--text-strong, var(--text)); font-weight: 550; text-align: right;
    background: none; border: none; padding: 1px 4px; border-radius: 5px; cursor: pointer; font: inherit; font-weight: 550;
  }
  .v:hover { background: var(--well); }
  .v.flat { cursor: default; }
  .v.flat:hover { background: none; }
  .v.dim { color: var(--text-dim); font-weight: 450; }
  .foot { font-size: 0.64rem; color: var(--text-dim); padding: 6px 6px 2px; }
</style>
