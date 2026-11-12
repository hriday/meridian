<script lang="ts">
  import { onMount } from "svelte";
  import { HIBPBreaches, HIBPLatestBreach, EmailBreachCheck } from "../../../wailsjs/go/app/App.js";

  let breaches: any[] = [];
  let latest: any = null;
  let loading = true;
  let err = "";
  let query = "";

  onMount(load);
  async function load() {
    loading = true; err = "";
    try {
      breaches = (await HIBPBreaches()) || [];
      try { latest = await HIBPLatestBreach(); } catch (e) { latest = null; }
    } catch (e) { err = String(e); }
    finally { loading = false; }
  }

  // Email breach check (XposedOrNot — the email IS sent to the service).
  let email = "", emailChecked = false, emailBusy = false, emailErr = "", emailHits: string[] = [];
  async function checkEmail() {
    emailErr = ""; emailChecked = false; emailHits = [];
    if (!email.trim() || !email.includes("@")) { emailErr = "Enter an email address."; return; }
    emailBusy = true;
    try { emailHits = (await EmailBreachCheck(email.trim())) || []; emailChecked = true; }
    catch (e) { emailErr = String(e); }
    finally { emailBusy = false; }
  }
  // Enrich a breach name from the loaded catalogue (date, data classes) when it matches.
  function detail(name: string): any {
    const lc = name.toLowerCase();
    return breaches.find((b) => b.name?.toLowerCase() === lc || b.title?.toLowerCase() === lc) || null;
  }

  function fmt(n: number): string { return (n || 0).toLocaleString(); }
  $: q = query.trim().toLowerCase();
  $: matches = q
    ? breaches
        .filter((b) => b.title?.toLowerCase().includes(q) || b.name?.toLowerCase().includes(q) || b.domain?.toLowerCase().includes(q))
        .sort((a, b) => (b.pwnCount || 0) - (a.pwnCount || 0))
        .slice(0, 100)
    : breaches.slice().sort((a, b) => (b.pwnCount || 0) - (a.pwnCount || 0)).slice(0, 20);
</script>

<div class="tool">
  <p class="lede">
    Browse the <strong>Have I Been Pwned</strong> breach catalogue — which sites were breached, when, how many
    accounts, and what data was exposed. Search by name or <strong>domain</strong> (e.g. <span class="mono">adobe.com</span>)
    to see the breaches that hit a company. This is free and keyless.
  </p>
  <!-- Check a specific email (XposedOrNot, free) -->
  <div class="emailbox">
    <div class="eb-title">Check an email address</div>
    <div class="eb-row">
      <input class="field mono" bind:value={email} placeholder="name@example.com" spellcheck="false" autocomplete="off" type="email" on:keydown={(e) => e.key === "Enter" && checkEmail()} />
      <button class="btn" on:click={checkEmail} disabled={emailBusy || !email.trim()}>{emailBusy ? "Checking…" : "Check"}</button>
    </div>
    <div class="eb-note">Uses <a href="https://xposedornot.com" target="_blank" rel="noreferrer">XposedOrNot</a>'s free breach database. Unlike the password check, <strong>the address is sent to that service</strong> — there's no way to check an email privately. HIBP's own email search needs a paid key and isn't used here.</div>
    {#if emailErr}<div class="banner bad" style="margin-top:8px"><span>{emailErr}</span></div>{/if}
    {#if emailChecked}
      {#if emailHits.length}
        <div class="ebverdict bad">Found in <b>{emailHits.length}</b> breach{emailHits.length === 1 ? "" : "es"} — change any reused passwords, and turn on 2FA where you can.</div>
        <div class="ebhits">
          {#each emailHits as h}
            {@const d = detail(h)}
            <div class="ebhit">
              <span class="ebname">{d?.title || h}</span>
              {#if d}<span class="dim sm mono">{d.domain || ""} · {d.breachDate || ""}</span>{/if}
              {#if d?.dataClasses?.length}<span class="dim sm"> · {d.dataClasses.slice(0, 4).join(", ")}{d.dataClasses.length > 4 ? "…" : ""}</span>{/if}
            </div>
          {/each}
        </div>
      {:else}
        <div class="ebverdict ok">Good news — <b>{email.trim()}</b> isn't in any breach XposedOrNot knows about. (That's not a guarantee it's never been exposed.)</div>
      {/if}
    {/if}
  </div>

  <div class="note">
    Below is the breach <em>catalogue</em> — which sites were breached and what leaked. For <em>passwords</em>, the
    <strong>AD Password Audit</strong> checks hashes against breaches for free.
  </div>

  <div class="searchrow">
    <input class="field mono" bind:value={query} placeholder="Search a site or domain — adobe.com, linkedin, …" spellcheck="false" autocomplete="off" />
    {#if !loading}<span class="count">{breaches.length.toLocaleString()} breaches tracked</span>{/if}
  </div>

  {#if loading}
    <p class="muted">Loading the breach catalogue…</p>
  {:else if err}
    <div class="banner bad"><span>{err}</span><button class="btn ghost sm" on:click={load}>Retry</button></div>
  {:else}
    {#if latest && !q}
      <div class="latest">
        <span class="tag">Most recent</span>
        <span class="lt mono">{latest.title}</span>
        <span class="dim">{latest.breachDate} · {fmt(latest.pwnCount)} accounts</span>
      </div>
    {/if}

    {#if !q}<p class="lbl">The 20 largest breaches — search above for a specific site or domain.</p>{/if}
    {#if q && matches.length === 0}<p class="muted">No breach matches “{query}”.</p>{/if}

    <div class="list">
      {#each matches as b}
        <div class="bcard">
          <div class="bhead">
            <span class="btitle">{b.title}</span>
            {#if b.domain}<span class="bdomain mono">{b.domain}</span>{/if}
            <span class="bspacer"></span>
            <span class="bcount mono">{fmt(b.pwnCount)}</span>
            <span class="dim sm">accounts</span>
          </div>
          <div class="bmeta">
            <span class="dim mono">{b.breachDate}</span>
            {#if b.verified}<span class="flag ok">verified</span>{:else}<span class="flag warn">unverified</span>{/if}
            {#if b.sensitive}<span class="flag sens">sensitive</span>{/if}
            {#if b.malware}<span class="flag sens">malware</span>{/if}
          </div>
          {#if b.dataClasses?.length}
            <div class="classes">{#each b.dataClasses as c}<span class="cls">{c}</span>{/each}</div>
          {/if}
          {#if b.description}<p class="bdesc">{b.description}</p>{/if}
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 30px; }
  .lede { max-width: 820px; margin: 0 0 10px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .note { max-width: 820px; font-size: 0.78rem; color: var(--text-dim); background: var(--well); border: 1px solid var(--border-soft, var(--border)); border-radius: 9px; padding: 9px 12px; margin-bottom: 14px; line-height: 1.5; }
  .mono { font-family: var(--font-mono, ui-monospace, monospace); }
  .muted { color: var(--text-dim); font-size: 0.85rem; }
  .searchrow { display: flex; align-items: center; gap: 12px; max-width: 820px; margin-bottom: 14px; }
  .field { flex: 1 1 auto; min-width: 0; background: var(--well); border: 1px solid var(--field-border, var(--border)); border-radius: 8px; padding: 9px 12px; color: var(--text); font-size: 0.88rem; outline: none; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .count { font-size: 0.74rem; color: var(--text-faint, var(--text-dim)); white-space: nowrap; }
  .lbl { font-size: 0.74rem; color: var(--text-dim); margin: 0 0 8px; }
  .latest { display: flex; align-items: center; gap: 10px; max-width: 820px; margin-bottom: 14px; padding: 9px 12px; border: 1px solid color-mix(in srgb, var(--accent) 40%, var(--border)); border-radius: 9px; background: color-mix(in srgb, var(--accent) 7%, transparent); font-size: 0.84rem; flex-wrap: wrap; }
  .latest .tag { font-size: 0.64rem; text-transform: uppercase; letter-spacing: .05em; color: var(--accent); }
  .latest .lt { font-weight: 650; color: var(--text-strong, var(--text)); }
  .dim { color: var(--text-dim); } .sm { font-size: 0.74rem; }
  .banner { display: flex; align-items: center; gap: 10px; max-width: 820px; padding: 9px 13px; border-radius: 9px; font-size: 0.84rem; margin-bottom: 12px; }
  .banner.bad { border: 1px solid var(--bad, #ff6b6b); background: color-mix(in srgb, var(--bad, #ff6b6b) 10%, transparent); color: var(--text); }
  .btn.ghost { background: var(--card); border: 1px solid var(--border); color: var(--text); border-radius: 7px; padding: 4px 11px; font: inherit; font-size: 0.78rem; cursor: pointer; margin-left: auto; }
  .list { display: flex; flex-direction: column; gap: 10px; max-width: 820px; }
  .bcard { background: var(--card); border: 1px solid var(--border-soft, var(--border)); border-radius: 10px; padding: 11px 13px; }
  .bhead { display: flex; align-items: baseline; gap: 9px; flex-wrap: wrap; }
  .btitle { font-weight: 650; color: var(--text-strong, var(--text)); font-size: 0.92rem; }
  .bdomain { font-size: 0.78rem; color: var(--text-dim); }
  .bspacer { flex: 1 1 auto; }
  .bcount { font-weight: 650; color: var(--accent); font-size: 0.95rem; }
  .bmeta { display: flex; align-items: center; gap: 8px; margin-top: 5px; flex-wrap: wrap; }
  .flag { font-size: 0.64rem; padding: 1px 7px; border-radius: 20px; border: 1px solid var(--border); color: var(--text-dim); }
  .flag.ok { color: var(--good, #3fb950); border-color: transparent; background: color-mix(in srgb, var(--good, #3fb950) 14%, transparent); }
  .flag.warn { color: var(--warn, #d6a531); border-color: color-mix(in srgb, var(--warn, #d6a531) 45%, var(--border)); }
  .flag.sens { color: var(--bad, #f0616a); border-color: color-mix(in srgb, var(--bad, #f0616a) 45%, var(--border)); }
  .classes { display: flex; flex-wrap: wrap; gap: 5px; margin-top: 8px; }
  .cls { font-size: 0.68rem; color: var(--text-dim); background: var(--well); border-radius: 6px; padding: 2px 7px; }
  .bdesc { font-size: 0.79rem; color: var(--text-dim); line-height: 1.5; margin: 8px 0 0; overflow-wrap: anywhere; }

  .emailbox { max-width: 820px; background: var(--card); border: 1px solid var(--border); border-radius: 11px; padding: 13px 15px; margin-bottom: 14px; }
  .eb-title { font-weight: 650; color: var(--text-strong, var(--text)); font-size: 0.92rem; margin-bottom: 9px; }
  .eb-row { display: flex; gap: 9px; align-items: center; }
  .eb-row .field { flex: 1 1 auto; }
  .btn { background: var(--accent); color: var(--on-accent, #052e2b); border: 1px solid transparent; border-radius: 8px; padding: 9px 16px; font: inherit; font-size: 0.86rem; font-weight: 550; cursor: pointer; white-space: nowrap; }
  .btn:disabled { opacity: .5; cursor: default; }
  .eb-note { font-size: 0.74rem; color: var(--text-faint, var(--text-dim)); margin-top: 8px; line-height: 1.5; }
  .ebverdict { margin-top: 10px; padding: 9px 12px; border-radius: 8px; font-size: 0.85rem; }
  .ebverdict.bad { background: color-mix(in srgb, var(--bad, #f0616a) 12%, transparent); border: 1px solid color-mix(in srgb, var(--bad, #f0616a) 40%, var(--border)); color: var(--text); }
  .ebverdict.ok { background: color-mix(in srgb, var(--good, #3fb950) 12%, transparent); border: 1px solid color-mix(in srgb, var(--good, #3fb950) 40%, var(--border)); color: var(--text); }
  .ebhits { display: flex; flex-direction: column; gap: 4px; margin-top: 10px; }
  .ebhit { display: flex; align-items: baseline; gap: 8px; flex-wrap: wrap; font-size: 0.82rem; padding: 3px 0; border-bottom: 1px solid var(--border-soft, var(--border)); }
  .ebname { font-weight: 600; color: var(--text); }
</style>
