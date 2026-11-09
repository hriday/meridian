<script lang="ts">
  import { DNSQuery, DNSSystem } from "../../../wailsjs/go/app/App.js";

  const TYPES = ["A", "AAAA", "MX", "NS", "TXT", "CNAME", "SOA", "PTR", "CAA", "ANY"];

  let system: any = null;
  let name = "cloudflare.com";
  let rtype = "A";
  let server = "";
  let busy = false;
  let error = "";
  let result: any = null;

  async function loadSystem() {
    try {
      system = await DNSSystem();
      if (!server && system?.values?.length) server = system.values[0];
    } catch (_) {
      system = null;
    }
  }

  async function run() {
    const n = name.trim();
    if (!n) return;
    error = "";
    busy = true;
    result = null;
    try {
      let srv = server.trim();
      if (!srv && system?.values?.length) srv = system.values[0];
      result = await DNSQuery(n, rtype, srv);
    } catch (e) {
      error = String(e);
    } finally {
      busy = false;
    }
  }

  // Plain-language bottom line for the result.
  function verdict(r: any): string {
    if (!r) return "";
    const n = (r.answers ?? []).length;
    switch (r.rcode) {
      case "NOERROR":
        if (n > 0) {
          const s = n === 1 ? "" : "s";
          return `${name} has ${n} ${rtype} record${s}, answered by ${r.server} in ${r.rttMillis} ms${r.authenticated ? " — DNSSEC-validated" : ""}.`;
        }
        return `${name} exists, but has no ${rtype} records.`;
      case "NXDOMAIN":
        return `${name} doesn't exist (NXDOMAIN) — there's no such name.`;
      case "SERVFAIL":
        return `${r.server} couldn't answer (SERVFAIL) — the resolver failed or the zone is broken.`;
      case "REFUSED":
        return `${r.server} refused the query (REFUSED).`;
      default:
        return `${r.server} returned ${r.rcode}.`;
    }
  }

  const FLAG_MEANING: Record<string, string> = {
    qr: "response",
    aa: "authoritative",
    tc: "truncated",
    rd: "recursion requested",
    ra: "recursion available",
    ad: "DNSSEC-validated",
    cd: "DNSSEC checking off",
  };
  function flagHelp(flags: string[]): string {
    return (flags ?? []).map((f) => `${f} = ${FLAG_MEANING[f] ?? f}`).join(" · ");
  }

  loadSystem();
  run();
</script>

<div class="tool">
  <h2>DNS</h2>
  <p class="sub">Query any resolver for any record type. Shows the resolver(s) this machine is configured to use.</p>

  <form on:submit|preventDefault={run} class="row">
    <input class="field mono" bind:value={name} placeholder="example.com" autocomplete="off" spellcheck="false" aria-label="Name" style="flex:2" />
    <select class="sel" bind:value={rtype} aria-label="Record type">
      {#each TYPES as t}<option>{t}</option>{/each}
    </select>
    <input class="field mono" bind:value={server} placeholder="resolver (default: system)" autocomplete="off" spellcheck="false" aria-label="Resolver" style="flex:1.2" />
    <button class="go" disabled={busy}>{busy ? "Querying…" : "Query"}</button>
  </form>

  {#if error}<div class="hint">{error}</div>{/if}

  {#if result}<div class="summary">{verdict(result)}</div>{/if}

  <div class="cards">
    <div class="card hero">
      <div class="chead"><h3>This machine</h3><span class="src">{system?.source ?? ""}</span></div>
      {#if system?.values?.length}
        <dl class="mono"><dt>Resolver(s)</dt><dd>{#each system.values as v}{v}<br />{/each}</dd></dl>
        <p class="cap">The resolver(s) your computer uses by default — leave the server box blank to query these.</p>
      {:else}
        <p class="muted">Couldn't detect configured resolvers{#if system?.err} ({system.err}){/if} — enter one above.</p>
      {/if}
    </div>

    {#if result}
      <div class="card">
        <div class="chead"><h3>Response</h3><span class="src">{result.server} · {result.rttMillis} ms</span></div>
        <dl class="mono">
          <dt>Status</dt>
          <dd>{result.rcode}<span class="cap">response code — NOERROR means the query succeeded</span></dd>
          <dt>Flags</dt>
          <dd>{(result.flags ?? []).join(" ") || "—"}<span class="cap">{flagHelp(result.flags)}</span></dd>
          <dt>DNSSEC</dt>
          <dd>{result.authenticated ? "authenticated (ad)" : "not authenticated"}<span class="cap">whether the resolver cryptographically validated the answer</span></dd>
          <dt>Answers</dt>
          <dd>{(result.answers ?? []).length}</dd>
        </dl>
      </div>

      <div class="card span">
        <div class="chead"><h3>Answer — {rtype} records for {name}</h3><span class="src">{result.server}</span></div>
        {#if (result.answers ?? []).length}
          <table class="tbl mono">
            <thead><tr><th>Name</th><th>Type</th><th>TTL</th><th>Value</th></tr></thead>
            <tbody>
              {#each result.answers as a}
                <tr><td>{a.name}</td><td>{a.type}</td><td>{a.ttl}</td><td class="val">{a.value}</td></tr>
              {/each}
            </tbody>
          </table>
        {:else}
          <p class="muted">No answer records ({result.rcode}).</p>
        {/if}
      </div>
    {/if}
  </div>
</div>

<style>
  .tool { padding: 16px 20px 40px; max-width: 940px; }
  h2 { margin: 0 0 2px; font-size: 1.15rem; font-weight: 600; color: var(--text-strong); }
  .sub { margin: 0 0 15px; color: var(--text-dim); font-size: 0.86rem; }
  .row { display: flex; gap: 8px; margin-bottom: 14px; flex-wrap: wrap; }
  .field { flex: 1; min-width: 150px; background: var(--well); border: 1px solid var(--field-border); border-radius: 7px; padding: 9px 12px; color: var(--text); font-size: 0.9rem; outline: none; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .sel { background: var(--well); border: 1px solid var(--field-border); border-radius: 7px; padding: 9px 10px; color: var(--text); font: inherit; font-size: 0.86rem; cursor: pointer; }
  .go { background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px; padding: 0 16px; font: inherit; font-size: 0.9rem; cursor: pointer; white-space: nowrap; }
  .go:disabled { opacity: .6; cursor: default; }
  .hint { background: var(--warn-bg); border: 1px solid var(--warn-border); color: var(--warn-text); border-radius: 7px; padding: 9px 12px; font-size: 0.85rem; margin-bottom: 12px; }
  .cards { display: grid; grid-template-columns: 1fr 1fr; gap: 13px; }
  @media (max-width: 720px) { .cards { grid-template-columns: 1fr; } }
  .card { background: var(--card); border: 1px solid var(--border-soft); border-radius: 10px; padding: 13px 15px; }
  .card.span { grid-column: 1 / -1; } .card.hero { border-color: var(--accent); box-shadow: 0 0 0 1px var(--accent-ring); }
  .chead { display: flex; align-items: center; justify-content: space-between; margin-bottom: 10px; gap: 8px; }
  .chead h3 { margin: 0; font-size: 0.68rem; text-transform: uppercase; letter-spacing: .06em; color: var(--text-dim); font-weight: 600; }
  .src { font-size: 0.62rem; color: var(--text-faint); font-family: var(--font-mono); }
  dl { display: grid; grid-template-columns: 100px 1fr; gap: 6px 12px; margin: 0; font-size: 0.86rem; }
  dt { color: var(--text-dim); } dd { margin: 0; color: var(--text); }
  .muted { color: var(--text-dim); font-size: 0.85rem; margin: 2px 0 0; }
  .summary { background: var(--card); border: 1px solid var(--accent); border-left-width: 3px; box-shadow: 0 0 0 1px var(--accent-ring); border-radius: 9px; padding: 11px 14px; margin-bottom: 14px; font-size: 0.9rem; color: var(--text-strong); max-width: 940px; }
  .cap { display: block; font-family: var(--font-sans); color: var(--text-faint); font-size: 0.72rem; margin-top: 2px; }
  table.tbl { width: 100%; border-collapse: collapse; font-size: 0.82rem; }
  .tbl th { text-align: left; color: var(--text-dim); font-weight: 500; font-size: 0.68rem; text-transform: uppercase; letter-spacing: .04em; padding: 6px 10px; border-bottom: 1px solid var(--border); }
  .tbl td { padding: 6px 10px; border-bottom: 1px solid var(--border-soft); }
  .tbl .val { word-break: break-all; }
  .mono { font-family: var(--font-mono); }
</style>
