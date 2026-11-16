<script lang="ts">
  import { NDPScan } from "../../../wailsjs/go/app/App.js";
  import ToolHelp from "../ToolHelp.svelte";

  let rep: any = null;
  let loading = false;
  let error = "";
  let withRA = false;

  // macOS returns a masked placeholder MAC to unprivileged callers; show that
  // honestly rather than as a real-looking address.
  const fmtMac = (m: string) => (!m ? "(incomplete)" : m === "2:0:0:0:0:0" ? "(hidden)" : m);
  $: masked = (rep?.neighbors || []).filter((n: any) => n.mac === "2:0:0:0:0:0").length;

  async function run() {
    loading = true;
    error = "";
    try {
      rep = await NDPScan(withRA);
    } catch (e: any) {
      error = String(e?.message ?? e);
      rep = null;
    } finally {
      loading = false;
    }
  }
</script>

<div class="tool">
  <p class="lede">
    Explores the <strong>IPv6 link</strong>: the neighbor cache (the v6 counterpart to ARP — every on-link host the OS has
    seen, with its MAC and reachability) and the <strong>IPv6 routers</strong> the OS has learned — both read
    unprivileged, no sudo. Enable <strong>live capture</strong> (needs sudo) to additionally solicit Router Advertisements
    off the wire and read the full detail: managed/other-config flags, prefixes with SLAAC/on-link flags, RDNSS, MTU and
    exact lifetimes.
  </p>

  <ToolHelp sudo="sudo for real MACs + live RA">
    Two sections. <strong>IPv6 neighbors</strong> is every device on your local IPv6 link — the v6 equivalent of an ARP
    table: address, MAC, interface and reachability. <strong>IPv6 routers</strong> is your gateway(s), learned from Router
    Advertisements; <code>expire</code> is how long that advertisement stays valid. A neighbor flagged <code>router</code>
    that shouldn't be one is a red flag (a rogue Router Advertisement). On macOS, neighbor MACs read <code>(hidden)</code>
    unless you run with <code>sudo</code>.
  </ToolHelp>

  <div class="row">
    <button class="btn" on:click={run} disabled={loading}>{loading ? "Scanning…" : "Scan link"}</button>
    <label class="toggle"><input type="checkbox" bind:checked={withRA} /> live RA capture (sudo)</label>
  </div>

  {#if error}<p class="err">{error}</p>{/if}

  {#if rep}
    <h3 class="sec">IPv6 neighbors <span class="count">{rep.neighbors?.length ?? 0}</span></h3>
    {#if rep.neighbors?.length}
      <div class="card">
        <table class="tbl">
          <thead><tr><th>Address</th><th>MAC</th><th>Interface</th><th>State</th><th></th></tr></thead>
          <tbody>
            {#each rep.neighbors as n}
              <tr>
                <td class="mono">{n.ip}</td>
                <td class="mono dim">{fmtMac(n.mac)}</td>
                <td class="dim">{n.interface}</td>
                <td class="dim">{n.state}</td>
                <td>{#if n.router}<span class="tag">router</span>{/if}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {:else}
      <p class="empty">No IPv6 neighbors cached.</p>
    {/if}
    {#if masked}
      <p class="masknote">🔒 {masked} MAC{masked === 1 ? "" : "s"} hidden by macOS — run the app with <code>sudo</code> for the real hardware addresses.</p>
    {/if}

    <h3 class="sec">IPv6 routers <span class="count">{rep.defaultRouters?.length ?? 0}</span></h3>
    {#if rep.defaultRouters?.length}
      <div class="cards">
        {#each rep.defaultRouters as rt}
          <div class="card ra">
            <span class="mono">{rt.address}{#if rt.interface}%{rt.interface}{/if}</span>
            <span class="facts">
              {#if rt.preference}<span>pref <code>{rt.preference}</code></span>{/if}
              {#if rt.expire}<span>expires <code>{rt.expire}</code></span>{/if}
            </span>
          </div>
        {/each}
      </div>
    {:else}
      <p class="empty">No IPv6 default router learned (your network may be IPv4-only).</p>
    {/if}

    {#if withRA}
      <h3 class="sec">Router Advertisements (live) {#if rep.routers?.length}<span class="count">{rep.routers.length}</span>{/if}</h3>
      {#if rep.raNote}
        <p class="note neu">{rep.raNote}</p>
      {:else if rep.routers?.length}
        <div class="cards">
          {#each rep.routers as ra}
            <div class="card ra">
              <div class="chead"><span class="mono">{ra.router}</span><span class="dim">{ra.interface}</span></div>
              <div class="facts">
                <span>lifetime <code>{ra.routerLifetime}s</code></span>
                <span>M <code>{ra.managed ? 1 : 0}</code></span>
                <span>O <code>{ra.otherConfig ? 1 : 0}</code></span>
                {#if ra.mtu}<span>MTU <code>{ra.mtu}</code></span>{/if}
                {#if ra.sourceMAC}<span>src <code>{ra.sourceMAC}</code></span>{/if}
              </div>
              {#each ra.prefixes || [] as p}
                <div class="prefix mono">{p.prefix}
                  <span class="pflags">{[p.onLink ? "on-link" : "", p.autonomous ? "SLAAC" : ""].filter(Boolean).join(" · ") || "no flags"}</span>
                  <span class="dim">valid {p.validLifetime}s</span>
                </div>
              {/each}
              {#if ra.dns?.length}<div class="extra">RDNSS: <span class="mono">{ra.dns.join(", ")}</span></div>{/if}
              {#if ra.dnsSearch?.length}<div class="extra">search: <span class="mono">{ra.dnsSearch.join(", ")}</span></div>{/if}
            </div>
          {/each}
        </div>
      {:else}
        <p class="empty">No Router Advertisements received. Routers send them periodically — try again, or a router may not be present.</p>
      {/if}
    {/if}
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 30px; }
  .lede { max-width: 820px; margin: 0 0 14px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .mono { font-family: var(--font-mono, ui-monospace, monospace); }
  .row { display: flex; gap: 14px; align-items: center; }
  .btn { background: var(--accent); color: var(--on-accent, #052e2b); border: 1px solid transparent; border-radius: 8px; padding: 9px 16px; font: inherit; font-size: 0.86rem; font-weight: 550; cursor: pointer; white-space: nowrap; }
  .btn:disabled { opacity: .5; cursor: default; }
  .toggle { display: inline-flex; align-items: center; gap: 6px; font-size: 0.8rem; color: var(--text-dim); cursor: pointer; }
  .err { color: var(--danger, #e5484d); font-size: 0.82rem; margin: 12px 0 0; }
  .sec { font-size: 0.78rem; text-transform: uppercase; letter-spacing: .05em; color: var(--text-dim); margin: 20px 0 8px; font-weight: 600; }
  .count { color: var(--accent); margin-left: 4px; }
  .card { background: var(--card); border: 1px solid var(--border); border-radius: 10px; max-width: 820px; overflow: auto; }
  .tbl { width: 100%; border-collapse: collapse; font-size: 0.82rem; }
  .tbl th { position: sticky; top: 0; background: var(--card); text-align: left; font-size: 0.66rem; text-transform: uppercase; letter-spacing: .05em; color: var(--text-dim); font-weight: 600; padding: 8px 10px; border-bottom: 1px solid var(--border); }
  .tbl td { padding: 6px 10px; border-bottom: 1px solid var(--border-soft, var(--border)); }
  .tbl tr:hover td { background: var(--hover, color-mix(in srgb, var(--accent) 5%, transparent)); }
  .dim { color: var(--text-dim); }
  .tag { font-size: 0.64rem; font-weight: 700; text-transform: uppercase; letter-spacing: .04em; color: var(--accent); border: 1px solid color-mix(in srgb, var(--accent) 55%, transparent); border-radius: 5px; padding: 0 5px; }
  .empty { font-size: 0.82rem; color: var(--text-faint, var(--text-dim)); max-width: 820px; }
  .masknote { font-size: 0.76rem; color: var(--warn, #d6a531); max-width: 820px; margin: 8px 0 0; }
  .masknote :global(code) { font-family: var(--font-mono, ui-monospace, monospace); }
  .note { font-size: 0.82rem; max-width: 820px; padding: 9px 13px; border-radius: 9px; border: 1px solid var(--border); border-left-width: 3px; }
  .note.neu { border-left-color: var(--text-faint, var(--text-dim)); color: var(--text-dim); }
  .cards { display: flex; flex-direction: column; gap: 10px; max-width: 820px; }
  .card.ra { padding: 11px 14px; }
  .chead { display: flex; justify-content: space-between; gap: 10px; margin-bottom: 7px; font-size: 0.85rem; }
  .facts { display: flex; flex-wrap: wrap; gap: 5px 14px; font-size: 0.76rem; color: var(--text-dim); margin-bottom: 6px; }
  .facts code { color: var(--text); }
  .prefix { font-size: 0.8rem; padding: 4px 0; display: flex; flex-wrap: wrap; gap: 8px; align-items: baseline; }
  .pflags { font-size: 0.72rem; color: var(--accent); }
  .extra { font-size: 0.76rem; color: var(--text-dim); margin-top: 3px; }
</style>
