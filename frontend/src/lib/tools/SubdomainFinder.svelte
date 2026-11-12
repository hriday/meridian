<script lang="ts">
  import { CTSubdomains } from "../../../wailsjs/go/app/App.js";
  import { requestedTool } from "../nav";

  let domain = "";
  let subs: any[] = [];
  let loading = false;
  let err = "";
  let searched = "";
  let filter = "";

  async function find() {
    err = ""; subs = [];
    if (!domain.trim() || !domain.includes(".")) { err = "Enter a domain, e.g. example.com"; return; }
    loading = true;
    try { subs = (await CTSubdomains(domain.trim())) || []; searched = domain.trim(); }
    catch (e) { err = String(e); }
    finally { loading = false; }
  }
  $: f = filter.trim().toLowerCase();
  $: rows = f ? subs.filter((s) => s.name.includes(f)) : subs;
  function pivot(tool: string, value: string) { requestedTool.set({ tool, value }); }
  async function copyAll() {
    try { await navigator.clipboard.writeText(subs.map((s) => s.name).join("\n")); } catch (e) {}
  }
</script>

<div class="tool">
  <p class="lede">
    Discover a domain's <strong>subdomains from Certificate Transparency logs</strong> (via
    <a href="https://crt.sh" target="_blank" rel="noreferrer">crt.sh</a>) — every host that's ever been issued a TLS
    certificate. A fast, free, keyless way to map the attack surface, then pivot into Lookup, TLS, or HTTP.
  </p>

  <div class="searchrow">
    <input class="field mono" bind:value={domain} placeholder="example.com" spellcheck="false" autocomplete="off"
      on:keydown={(e) => e.key === "Enter" && find()} />
    <button class="btn" on:click={find} disabled={loading || !domain.trim()}>{loading ? "Searching CT logs…" : "Find subdomains"}</button>
  </div>
  <div class="note">Finds only names that appear in a certificate — not every host that exists. crt.sh can be slow (or flaky) on large domains; just retry if it errors.</div>

  {#if err}
    <div class="banner bad"><span>{err}</span><button class="link" on:click={find}>Retry</button></div>
  {/if}

  {#if searched && !loading && !err}
    <div class="summary">
      <span class="exp mono">{subs.length}</span> subdomain{subs.length === 1 ? "" : "s"} of <span class="mono">{searched}</span> found in CT logs.
      {#if subs.length}<button class="link" on:click={copyAll}>Copy all</button>{/if}
    </div>
    {#if subs.length}
      <input class="field mono filter" bind:value={filter} placeholder="filter — api, dev, vpn…" spellcheck="false" />
      <div class="card">
        <table class="tbl">
          <thead><tr><th>Host</th><th class="r">Certs</th><th>Last seen</th><th class="r">Pivot</th></tr></thead>
          <tbody>
            {#each rows as s}
              <tr>
                <td class="mono host">{#if s.wildcard}<span class="wc" title="appeared as a wildcard *.{s.name}">*</span>{/if}{s.name}</td>
                <td class="r dim">{s.certs}</td>
                <td class="dim mono sm">{s.lastSeen || ""}</td>
                <td class="r piv">
                  <button class="link" on:click={() => pivot("lookup", s.name)}>Look up</button>
                  <button class="link" on:click={() => pivot("tls", s.name)}>TLS</button>
                  <button class="link" on:click={() => pivot("http", "https://" + s.name)}>HTTP</button>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 30px; }
  .lede { max-width: 820px; margin: 0 0 12px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .mono { font-family: var(--font-mono, ui-monospace, monospace); }
  .searchrow { display: flex; gap: 10px; max-width: 700px; margin-bottom: 8px; }
  .field { background: var(--well); border: 1px solid var(--field-border, var(--border)); border-radius: 8px; padding: 9px 12px; color: var(--text); font-size: 0.88rem; outline: none; }
  .searchrow .field { flex: 1 1 auto; min-width: 0; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .filter { max-width: 320px; margin-bottom: 10px; font-size: 0.82rem; padding: 7px 11px; }
  .btn { background: var(--accent); color: var(--on-accent, #052e2b); border: 1px solid transparent; border-radius: 8px; padding: 9px 16px; font: inherit; font-size: 0.86rem; font-weight: 550; cursor: pointer; white-space: nowrap; }
  .btn:disabled { opacity: .5; cursor: default; }
  .note { max-width: 820px; font-size: 0.76rem; color: var(--text-faint, var(--text-dim)); margin-bottom: 14px; line-height: 1.5; }
  .banner { display: flex; align-items: center; gap: 10px; max-width: 820px; padding: 9px 13px; border-radius: 9px; font-size: 0.84rem; margin-bottom: 12px; }
  .banner.bad { border: 1px solid var(--bad, #ff6b6b); background: color-mix(in srgb, var(--bad, #ff6b6b) 10%, transparent); color: var(--text); }
  .summary { max-width: 820px; font-size: 0.86rem; color: var(--text-dim); margin-bottom: 10px; display: flex; align-items: center; gap: 10px; }
  .summary .exp { color: var(--accent); font-weight: 650; }
  .link { background: none; border: 0; color: var(--accent); cursor: pointer; font: inherit; font-size: 0.78rem; padding: 0 2px; }
  .link:hover { text-decoration: underline; }
  .card { background: var(--card); border: 1px solid var(--border); border-radius: 10px; max-width: 820px; overflow: auto; max-height: 540px; }
  .tbl { width: 100%; border-collapse: collapse; font-size: 0.82rem; }
  .tbl th { position: sticky; top: 0; background: var(--card); text-align: left; font-size: 0.66rem; text-transform: uppercase; letter-spacing: .05em; color: var(--text-dim); font-weight: 600; padding: 8px 10px; border-bottom: 1px solid var(--border); }
  .tbl th.r, .tbl td.r { text-align: right; }
  .tbl td { padding: 6px 10px; border-bottom: 1px solid var(--border-soft, var(--border)); }
  .tbl tr:hover td { background: var(--hover, color-mix(in srgb, var(--accent) 5%, transparent)); }
  .host { color: var(--text); }
  .wc { color: var(--warn, #d6a531); margin-right: 1px; }
  .dim { color: var(--text-dim); } .sm { font-size: 0.76rem; }
  .piv { white-space: nowrap; }
</style>
