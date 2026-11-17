<script lang="ts">
  import { ServiceStatus, ServiceStatusKnown } from "../../../wailsjs/go/app/App.js";
  import ToolHelp from "../ToolHelp.svelte";
  import { BrowserOpenURL } from "../../../wailsjs/runtime/runtime.js";

  let target = "";
  let known: string[] = [];
  let status: any = null;
  let loading = false;
  let error = "";

  ServiceStatusKnown().then((k) => (known = k || [])).catch(() => {});

  // User-pinned services, kept per-machine in localStorage (the backend already
  // accepts any URL, so a custom chip is just a saved check target).
  type Custom = { label: string; url: string };
  const LS_KEY = "meridian.svcstatus.custom";
  let customs: Custom[] = [];
  let adding = false;
  let newLabel = "";
  let newUrl = "";

  try {
    const raw = localStorage.getItem(LS_KEY);
    if (raw) customs = JSON.parse(raw) || [];
  } catch {}

  function saveCustoms() {
    try {
      localStorage.setItem(LS_KEY, JSON.stringify(customs));
    } catch {}
  }

  function labelFromUrl(u: string): string {
    try {
      const h = new URL(u.includes("://") ? u : "https://" + u).hostname;
      return h.replace(/^www\./, "").replace(/^status\./, "");
    } catch {
      return u;
    }
  }

  function addCustom() {
    const url = newUrl.trim();
    if (!url) return;
    const label = newLabel.trim() || labelFromUrl(url);
    customs = [...customs.filter((c) => c.url !== url), { label, url }];
    saveCustoms();
    newLabel = "";
    newUrl = "";
    adding = false;
    check(url);
  }

  function removeCustom(url: string) {
    customs = customs.filter((c) => c.url !== url);
    saveCustoms();
  }

  async function check(t?: string) {
    const q = (t ?? target).trim();
    if (!q) return;
    target = q;
    loading = true;
    error = "";
    status = null;
    try {
      status = await ServiceStatus(q);
    } catch (e: any) {
      error = String(e?.message ?? e);
    } finally {
      loading = false;
    }
  }

  const compLabel = (s: string) => s.replace(/_/g, " ");
</script>

<div class="tool">
  <p class="lede">
    Check whether an online service is <strong>having an outage right now</strong>, from its <strong>official status
    feed</strong> (authoritative status, not crowd reports). Works with any <strong>Statuspage</strong>-backed service
    plus <strong>Apple</strong>, <strong>Azure</strong> and <strong>Google</strong> Cloud/Workspace — pick a chip or
    paste a status-page URL. Free — no account.
  </p>

  <ToolHelp>
    A green dot with "No active incidents" means the provider reports everything operational — so if you're seeing
    trouble, it's likely your side or the network between you. A red dot lists the <strong>affected components</strong>
    and any <strong>active incidents</strong>, graded by impact (<code>minor</code> → <code>major</code>/<code>critical</code>)
    — that's confirmation the outage is on their end, not yours. Open the incident details for the provider's own
    timeline and ETA.
  </ToolHelp>

  <div class="row">
    <input class="field mono" bind:value={target} placeholder="github  ·  or  https://status.example.com" spellcheck="false"
      on:keydown={(e) => e.key === "Enter" && check()} />
    <button class="btn" on:click={() => check()} disabled={loading || !target.trim()}>{loading ? "Checking…" : "Check"}</button>
  </div>

  <div class="chips">
    {#each known as k}
      <button class="chip" on:click={() => check(k)}>{k}</button>
    {/each}
    {#each customs as c (c.url)}
      <span class="chip custom">
        <button class="clabel" title={c.url} on:click={() => check(c.url)}>{c.label}</button>
        <button class="cx" title="Remove" on:click={() => removeCustom(c.url)}>×</button>
      </span>
    {/each}
    <button class="chip add" on:click={() => (adding = !adding)}>+ Add service</button>
  </div>

  {#if adding}
    <div class="addrow">
      <input class="field mono afield" bind:value={newLabel} placeholder="name (optional)" spellcheck="false" />
      <input class="field mono afield" bind:value={newUrl} placeholder="status URL  ·  e.g. status.acme.com" spellcheck="false"
        on:keydown={(e) => e.key === "Enter" && addCustom()} />
      <button class="btn abtn" on:click={addCustom} disabled={!newUrl.trim()}>Save</button>
    </div>
    <p class="ahint">Any Statuspage, Apple, Azure or Google status URL. Saved on this computer and pinned as a chip.</p>
  {/if}

  {#if error}<p class="err">{error}</p>{/if}

  {#if status}
    <div class="card {status.operational ? 'ok' : 'bad'}">
      <div class="shead">
        <span class="dot {status.operational ? 'ok' : 'bad'}"></span>
        <span class="sname">{status.service}</span>
        <span class="sdesc">{status.description}</span>
        {#if status.source}<span class="ssrc">via {status.source}</span>{/if}
      </div>

      {#if status.impacted && status.impacted.length}
        <div class="sect">
          <div class="sk">Affected components</div>
          {#each status.impacted as c}
            <div class="comp"><span class="cstatus {c.status}">{compLabel(c.status)}</span><span class="cname">{c.name}</span></div>
          {/each}
        </div>
      {/if}

      {#if status.incidents && status.incidents.length}
        <div class="sect">
          <div class="sk">Active incidents</div>
          {#each status.incidents as inc}
            <div class="inc">
              <div class="iline"><span class="impact {inc.impact}">{inc.impact}</span><span class="iname">{inc.name}</span><span class="istatus">{inc.status}</span></div>
              {#if inc.shortlink}<button class="ilink" on:click={() => BrowserOpenURL(inc.shortlink)}>details ↗</button>{/if}
            </div>
          {/each}
        </div>
      {/if}

      {#if status.operational}
        <p class="allgood">No active incidents. <button class="link" on:click={() => BrowserOpenURL(status.url)}>status page ↗</button></p>
      {/if}
    </div>
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 30px; max-width: 820px; }
  .lede { margin: 0 0 14px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .mono { font-family: var(--font-mono, ui-monospace, monospace); }
  .row { display: flex; gap: 10px; max-width: 620px; }
  .field { background: var(--well); border: 1px solid var(--field-border, var(--border)); border-radius: 8px; padding: 9px 12px; color: var(--text); font-size: 0.86rem; outline: none; flex: 1 1 auto; min-width: 0; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .btn { background: var(--accent); color: var(--on-accent, #052e2b); border: 1px solid transparent; border-radius: 8px; padding: 9px 16px; font: inherit; font-size: 0.86rem; font-weight: 550; cursor: pointer; white-space: nowrap; }
  .btn:disabled { opacity: .5; cursor: default; }
  .chips { display: flex; flex-wrap: wrap; gap: 6px; margin-top: 10px; max-width: 620px; }
  .chip { font-size: 0.74rem; background: var(--well); border: 1px solid var(--border); border-radius: 20px; padding: 3px 11px; color: var(--text-dim); cursor: pointer; }
  .chip:hover { border-color: var(--accent); color: var(--text); }
  .chip.custom { display: inline-flex; align-items: center; padding: 0; }
  .chip.custom .clabel { background: none; border: 0; font: inherit; font-size: 0.74rem; color: var(--text-dim); cursor: pointer; padding: 3px 4px 3px 11px; border-radius: 20px 0 0 20px; }
  .chip.custom:hover { border-color: var(--accent); }
  .chip.custom .clabel:hover { color: var(--text); }
  .chip.custom .cx { background: none; border: 0; color: var(--text-faint, var(--text-dim)); cursor: pointer; font-size: 0.95rem; line-height: 1; padding: 0 9px 0 3px; }
  .chip.custom .cx:hover { color: var(--bad, #ff6b6b); }
  .chip.add { border-style: dashed; }
  .addrow { display: flex; gap: 8px; margin-top: 10px; max-width: 620px; }
  .afield { padding: 7px 11px; font-size: 0.8rem; }
  .abtn { padding: 7px 15px; font-size: 0.8rem; }
  .ahint { max-width: 620px; margin: 6px 0 0; font-size: 0.72rem; color: var(--text-faint, var(--text-dim)); }
  .err { color: var(--danger, #e5484d); font-size: 0.82rem; margin: 12px 0 0; max-width: 620px; line-height: 1.5; }
  .card { border: 1px solid var(--border); border-left-width: 3px; border-radius: 11px; padding: 13px 16px; margin-top: 16px; background: var(--card); }
  .card.ok { border-left-color: var(--good, #3fb950); }
  .card.bad { border-left-color: var(--danger, #e5484d); }
  .shead { display: flex; align-items: center; gap: 9px; flex-wrap: wrap; }
  .dot { width: 10px; height: 10px; border-radius: 50%; flex: none; }
  .dot.ok { background: var(--good, #3fb950); }
  .dot.bad { background: var(--danger, #e5484d); }
  .sname { font-weight: 650; font-size: 0.92rem; color: var(--text); }
  .sdesc { font-size: 0.82rem; color: var(--text-dim); }
  .ssrc { margin-left: auto; font-size: 0.66rem; text-transform: uppercase; letter-spacing: .04em; color: var(--text-faint, var(--text-dim)); border: 1px solid var(--border); border-radius: 20px; padding: 1px 8px; }
  .sect { margin-top: 12px; }
  .sk { font-size: 0.7rem; text-transform: uppercase; letter-spacing: .04em; color: var(--text-faint, var(--text-dim)); font-weight: 650; margin-bottom: 6px; }
  .comp { display: flex; align-items: center; gap: 10px; padding: 3px 0; font-size: 0.82rem; }
  .cstatus { font-size: 0.68rem; font-weight: 700; text-transform: uppercase; padding: 1px 7px; border-radius: 5px; white-space: nowrap; background: var(--warn, #f5a623); color: #3a2a00; }
  .cstatus.major_outage, .cstatus.partial_outage { background: var(--danger, #e5484d); color: #fff; }
  .cname { color: var(--text); }
  .inc { padding: 5px 0; border-top: 1px solid var(--border-soft, var(--border)); }
  .inc:first-of-type { border-top: 0; }
  .iline { display: flex; align-items: center; gap: 9px; flex-wrap: wrap; }
  .impact { font-size: 0.66rem; font-weight: 700; text-transform: uppercase; padding: 1px 7px; border-radius: 5px; background: var(--warn, #f5a623); color: #3a2a00; }
  .impact.major, .impact.critical { background: var(--danger, #e5484d); color: #fff; }
  .iname { font-size: 0.83rem; color: var(--text); font-weight: 550; }
  .istatus { font-size: 0.74rem; color: var(--text-dim); }
  .ilink, .link { background: none; border: none; color: var(--accent); font: inherit; font-size: 0.76rem; cursor: pointer; padding: 2px 0; }
  .ilink:hover, .link:hover { text-decoration: underline; }
  .allgood { font-size: 0.82rem; color: var(--text-dim); margin: 10px 0 0; }
</style>
