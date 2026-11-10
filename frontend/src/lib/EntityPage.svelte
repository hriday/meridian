<script lang="ts">
  export let type: "ip" | "asn" | "prefix" | "domain";
  export let data: any;
  export let onpivot: (value: string) => void = () => {};

  let tab: "overview" | "registry" = "overview";
  // Reset to Overview whenever a new entity loads.
  $: if (data) tab = "overview";

  function ago(ts: string): string {
    if (!ts) return "";
    const d = (Date.now() - new Date(ts).getTime()) / 1000;
    if (isNaN(d)) return "";
    if (d < 60) return `${Math.max(0, Math.round(d))}s ago`;
    if (d < 3600) return `${Math.round(d / 60)}m ago`;
    return `${Math.round(d / 3600)}h ago`;
  }
  function chip(src: any): string {
    if (!src) return "";
    if (src.err) return `${src.label}: unavailable`;
    return `${src.label} · ${ago(src.fetchedAt)}`;
  }
  function rpkiClass(state: string): string {
    return state === "valid" ? "good" : state === "invalid" ? "bad" : "";
  }
  // Port speed comes from PeeringDB in Mbps.
  function gbps(mbps: number): string {
    if (!mbps) return "";
    return mbps >= 1000 ? `${+(mbps / 1000).toFixed(mbps % 1000 ? 1 : 0)} Gbps` : `${mbps} Mbps`;
  }
  // Plain-language gloss for a PeeringDB general peering policy.
  function policyNote(policy: string): string {
    switch ((policy || "").toLowerCase()) {
      case "open":
        return "Open — will peer with most networks, usually with no paperwork.";
      case "selective":
        return "Selective — peers case by case, often with requirements.";
      case "restrictive":
        return "Restrictive — rarely adds new peers.";
      default:
        return "";
    }
  }
  async function copyJSON() {
    try {
      await navigator.clipboard.writeText(JSON.stringify(data, null, 2));
    } catch (_) {
      /* clipboard blocked; ignore */
    }
  }
</script>

<div class="entity">
  {#if type === "ip"}
    <div class="crumbs">
      <span class="cur mono">{data.input}</span>
      {#if data.coveringPrefix}<span class="sep">›</span><button class="piv mono" on:click={() => onpivot(data.coveringPrefix)}>{data.coveringPrefix}</button>{/if}
      {#if data.originAsn}<span class="sep">›</span><button class="piv mono" on:click={() => onpivot(data.originAsn)}>{data.originAsn}</button>{/if}
      {#if data.registry}<span class="sep">·</span><span>{data.registry}</span>{/if}
    </div>
    <div class="hhead"><span class="htitle mono">{data.input}</span><span class="htag">{data.version}</span></div>
    <p class="hsub">
      {#if data.originName}Announced by {data.originAsn} · {data.originName}{:else if data.routingNote}{data.routingNote}{:else}No global routing seen{/if}
    </p>
  {:else if type === "asn"}
    <div class="crumbs">
      <span class="cur mono">{data.asn}</span>
      {#if data.holder}<span class="sep">·</span><span>{data.holder}</span>{/if}
      {#if data.registry}<span class="sep">›</span><span>{data.registry}</span>{/if}
    </div>
    <div class="hhead"><span class="htitle mono">{data.asn}</span><span class="htag">Autonomous System</span></div>
    <p class="hsub">{data.holder || "Autonomous system"}{#if data.registry} · registered via {data.registry}{/if}</p>
  {:else if type === "prefix"}
    <div class="crumbs">
      <span class="cur mono">{data.input}</span>
      {#if data.originAsn}<span class="sep">›</span><button class="piv mono" on:click={() => onpivot(data.originAsn)}>{data.originAsn}</button>{/if}
      {#if data.registry}<span class="sep">·</span><span>{data.registry}</span>{/if}
    </div>
    <div class="hhead"><span class="htitle mono">{data.input}</span><span class="htag">Prefix</span></div>
    <p class="hsub">{#if data.originAsn}Announced by {data.originAsn}{#if data.originName} · {data.originName}{/if}{:else}Not globally announced{/if}</p>
  {:else}
    <div class="crumbs">
      <span class="cur mono">{data.input}</span>
      {#if data.registrar}<span class="sep">·</span><span>{data.registrar}</span>{/if}
    </div>
    <div class="hhead"><span class="htitle">{data.input}</span><span class="htag">Domain</span></div>
    <p class="hsub">{#if data.resolvesTo?.length}Resolves to {data.resolvesTo.length} address{data.resolvesTo.length > 1 ? "es" : ""}{#if data.hostedBy?.length} · hosted by {data.hostedBy.join(", ")}{/if}{:else if data.rcode === "NXDOMAIN"}This domain doesn't resolve (NXDOMAIN){:else}No A/AAAA records{#if data.rcode && data.rcode !== "NOERROR"} ({data.rcode}){/if}{/if}</p>
  {/if}

  <div class="actions"><button class="btn" on:click={copyJSON}>Copy as JSON</button></div>

  <div class="tabbar">
    <button class="tab" class:on={tab === "overview"} on:click={() => (tab = "overview")}>Overview</button>
    <button class="tab" class:on={tab === "registry"} on:click={() => (tab = "registry")}>Registry</button>
    {#if type === "ip"}
      <span class="tab dis">Routing <span class="soon">soon</span></span>
      <span class="tab dis">DNS <span class="soon">soon</span></span>
      <span class="tab dis">Reachability <span class="soon">soon</span></span>
    {/if}
  </div>

  {#if type === "ip"}
    {#if tab === "overview"}
      <div class="cards">
        <div class="card hero span">
          <div class="chead">
            <h3>Who owns this</h3>
            <span class="srcs"><span class="src">{chip(data.routingSrc)}</span><span class="src">{chip(data.registrySrc)}</span></span>
          </div>
          <dl class="mono">
            {#if data.coveringPrefix}<dt>Covering prefix</dt><dd class="big"><button class="piv" on:click={() => onpivot(data.coveringPrefix)}>{data.coveringPrefix}</button></dd>{/if}
            {#if data.originAsn}<dt>Origin AS</dt><dd class="big"><button class="piv" on:click={() => onpivot(data.originAsn)}>{data.originAsn}</button> <span class="dim">{data.originName}</span></dd>{/if}
            {#if data.registeredOrg}<dt>Registered org</dt><dd>{data.registeredOrg}{#if data.registry} <span class="pill">{data.registry}</span>{/if}</dd>{/if}
            {#if data.abuseEmail}<dt>Abuse</dt><dd>{data.abuseEmail}</dd>{/if}
            {#if data.routingNote}<dt>Note</dt><dd class="dim">{data.routingNote}</dd>{/if}
          </dl>
          {#if data.routingSrc?.err}<p class="err">Routing data unavailable: {data.routingSrc.err}</p>{/if}
          {#if data.registrySrc?.err}<p class="err">Registry data unavailable: {data.registrySrc.err}</p>{/if}
        </div>
        <div class="card">
          <div class="chead"><h3>Routing</h3><span class="src">{chip(data.routingSrc)}</span></div>
          <dl class="mono">
            <dt>Origin AS</dt><dd>{data.originAsn || "—"}</dd>
            <dt>Prefix</dt><dd>{data.coveringPrefix || "—"}</dd>
            <dt>RPKI</dt><dd>{#if data.rpkiState}<span class="pill {rpkiClass(data.rpkiState)}">{data.rpkiState}</span>{:else}<span class="pill soon">unknown</span>{/if}</dd>
          </dl>
          {#if data.routingSrc?.err}<p class="err">Routing data unavailable: {data.routingSrc.err}</p>{/if}
        </div>
      </div>
    {:else}
      <div class="cards"><div class="card span">
        <div class="chead"><h3>Registry (RDAP)</h3><span class="src">{chip(data.registrySrc)}</span></div>
        <dl class="mono"><dt>Organization</dt><dd>{data.registeredOrg || "—"}</dd><dt>Registry</dt><dd>{data.registry || "—"}</dd><dt>Abuse</dt><dd>{data.abuseEmail || "—"}</dd></dl>
      </div></div>
    {/if}

  {:else if type === "asn"}
    {#if tab === "overview"}
      <div class="cards">
        <div class="card hero span">
          <div class="chead"><h3>Autonomous System</h3><span class="src">{chip(data.routingSrc)}</span></div>
          <dl class="mono">
            <dt>Holder</dt><dd class="big">{data.holder || "—"}</dd>
            <dt>Prefixes (v4)</dt><dd>{data.v4Prefixes} announced</dd>
            <dt>Prefixes (v6)</dt><dd>{data.v6Prefixes} announced</dd>
            <dt>Neighbours</dt><dd>{data.neighbours} observed</dd>
          </dl>
        </div>

        {#if data.peering}
          <div class="card span">
            <div class="chead"><h3>Peering</h3><span class="src">{chip(data.peeringSrc)}</span></div>
            <dl class="mono">
              {#if data.peering.type}<dt>Type</dt><dd>{data.peering.type}</dd>{/if}
              {#if data.peering.scope}<dt>Scope</dt><dd>{data.peering.scope}</dd>{/if}
              {#if data.peering.ratio}<dt>Traffic</dt><dd>{data.peering.ratio}</dd>{/if}
              {#if data.peering.policy}<dt>Policy</dt><dd><span class="pill">{data.peering.policy}</span></dd>{/if}
              {#if data.peering.irrAsSet}<dt>IRR AS-SET</dt><dd>{data.peering.irrAsSet}</dd>{/if}
              {#if data.peering.v4Prefixes || data.peering.v6Prefixes}<dt>Max prefixes</dt><dd>{data.peering.v4Prefixes} v4 · {data.peering.v6Prefixes} v6 <span class="dim">suggested filter limits</span></dd>{/if}
              {#if data.peering.website}<dt>Website</dt><dd><a class="piv" href={data.peering.website} target="_blank" rel="noopener noreferrer">{data.peering.website}</a></dd>{/if}
            </dl>
            {#if policyNote(data.peering.policy)}<p class="cap">{policyNote(data.peering.policy)}</p>{/if}

            {#if data.peering.ixps?.length}
              <div class="pblock">
                <div class="phead"><span>Internet exchanges</span><span class="dim">{data.peering.ixps.length} present</span></div>
                <div class="chips">
                  {#each data.peering.ixps.slice(0, 12) as ix}
                    <span class="pchip">{ix.name}{#if ix.speed} <span class="dim">{gbps(ix.speed)}</span>{/if}</span>
                  {/each}
                  {#if data.peering.ixps.length > 12}<span class="pchip more">+{data.peering.ixps.length - 12} more</span>{/if}
                </div>
              </div>
            {/if}

            {#if data.peering.facilities?.length}
              <div class="pblock">
                <div class="phead"><span>Facilities</span><span class="dim">{data.peering.facilities.length} present</span></div>
                <div class="chips">
                  {#each data.peering.facilities.slice(0, 12) as f}
                    <span class="pchip">{f.name}{#if f.city} <span class="dim">{f.city}{#if f.country}, {f.country}{/if}</span>{/if}</span>
                  {/each}
                  {#if data.peering.facilities.length > 12}<span class="pchip more">+{data.peering.facilities.length - 12} more</span>{/if}
                </div>
              </div>
            {/if}
          </div>
        {:else if data.peeringSrc?.err}
          <div class="card span">
            <div class="chead"><h3>Peering</h3><span class="src">{chip(data.peeringSrc)}</span></div>
            <p class="err">PeeringDB was unavailable: {data.peeringSrc.err}</p>
          </div>
        {/if}
      </div>
    {:else}
      <div class="cards"><div class="card span">
        <div class="chead"><h3>Registry (RDAP)</h3><span class="src">{chip(data.registrySrc)}</span></div>
        <dl class="mono"><dt>Handle</dt><dd>{data.asn}</dd><dt>Holder</dt><dd>{data.holder || "—"}</dd><dt>Registry</dt><dd>{data.registry || "—"}</dd></dl>
      </div></div>
    {/if}

  {:else if type === "prefix"}
    {#if tab === "overview"}
      <div class="cards">
        <div class="card hero span">
          <div class="chead"><h3>Who announces this</h3><span class="srcs"><span class="src">{chip(data.routingSrc)}</span><span class="src">{chip(data.rpkiSrc)}</span></span></div>
          <dl class="mono">
            {#if data.originAsn}<dt>Origin AS</dt><dd class="big"><button class="piv" on:click={() => onpivot(data.originAsn)}>{data.originAsn}</button> <span class="dim">{data.originName}</span></dd>{/if}
            <dt>Announced</dt><dd>{data.announced ? "yes" : "no"}</dd>
            <dt>RPKI</dt><dd>{#if data.rpkiState}<span class="pill {rpkiClass(data.rpkiState)}">{data.rpkiState}</span> {#if data.rpkiRoa}<span class="dim">{data.rpkiRoa}</span>{/if}{:else}<span class="pill soon">unknown</span>{/if}</dd>
            {#if data.registeredOrg}<dt>Registered org</dt><dd>{data.registeredOrg}{#if data.registry} <span class="pill">{data.registry}</span>{/if}</dd>{/if}
            {#if data.abuseEmail}<dt>Abuse</dt><dd>{data.abuseEmail}</dd>{/if}
          </dl>
          {#if data.routingSrc?.err}<p class="err">Routing data unavailable: {data.routingSrc.err}</p>{/if}
        </div>
        {#if data.lessSpecific?.length || data.moreSpecific?.length}
          <div class="card span">
            <div class="chead"><h3>Related prefixes</h3><span class="src">{chip(data.routingSrc)}</span></div>
            {#if data.lessSpecific?.length}<p class="lbl">Less specific (enclosing)</p><div class="lst">{#each data.lessSpecific as p}<button class="piv mono" on:click={() => onpivot(p)}>{p}</button>{/each}</div>{/if}
            {#if data.moreSpecific?.length}<p class="lbl">More specific (sub-prefixes)</p><div class="lst">{#each data.moreSpecific as p}<button class="piv mono" on:click={() => onpivot(p)}>{p}</button>{/each}</div>{/if}
          </div>
        {/if}
      </div>
    {:else}
      <div class="cards"><div class="card span">
        <div class="chead"><h3>Registry (RDAP)</h3><span class="src">{chip(data.registrySrc)}</span></div>
        <dl class="mono"><dt>Organization</dt><dd>{data.registeredOrg || "—"}</dd><dt>Registry</dt><dd>{data.registry || "—"}</dd><dt>Abuse</dt><dd>{data.abuseEmail || "—"}</dd></dl>
      </div></div>
    {/if}

  {:else if tab === "overview"}
    <div class="cards">
      <div class="card hero span">
        <div class="chead"><h3>Where it resolves</h3><span class="src">{chip(data.dnsSrc)}</span></div>
        <dl class="mono">
          <dt>Resolves to</dt><dd>{#if data.resolvesTo?.length}{#each data.resolvesTo as ip, i}{#if i > 0}, {/if}<button class="piv" on:click={() => onpivot(ip)}>{ip}</button>{/each}{:else}—{/if}</dd>
          <dt>Hosted by</dt><dd>{data.hostedBy?.join(", ") || "—"}</dd>
        </dl>
        {#if data.dnsSrc?.err}<p class="err">DNS unavailable: {data.dnsSrc.err}</p>{/if}
      </div>
      <div class="card span">
        <div class="chead"><h3>DNS records</h3><span class="src">{chip(data.dnsSrc)}</span></div>
        {#if data.records?.length}
          <table class="rec mono"><tbody>{#each data.records as r}<tr><td class="rt">{r.type}</td><td>{r.name}</td><td class="rv">{r.value}</td></tr>{/each}</tbody></table>
        {:else}<p class="dim">No records{#if data.rcode && data.rcode !== "NOERROR"} · {data.rcode}{/if}.</p>{/if}
      </div>
    </div>
  {:else}
    <div class="cards"><div class="card span">
      <div class="chead"><h3>Registration (RDAP)</h3><span class="src">{chip(data.registrySrc)}</span></div>
      {#if data.registrySrc?.err}<p class="dim">No RDAP registration data for this domain (some TLDs don't serve RDAP).</p>{/if}
      <dl class="mono">
        <dt>Registrar</dt><dd>{data.registrar || "—"}</dd>
        <dt>Created</dt><dd>{data.created || "—"}</dd>
        <dt>Expires</dt><dd>{data.expires || "—"}</dd>
        <dt>DNSSEC</dt><dd>{data.dnssec || "—"}</dd>
        <dt>Nameservers</dt><dd>{data.nameservers?.join(", ") || "—"}</dd>
      </dl>
    </div></div>
  {/if}
</div>

<style>
  .entity { padding: 16px 20px 40px; max-width: 980px; }
  .crumbs { display: flex; align-items: center; flex-wrap: wrap; gap: 7px; font-size: 0.8rem; margin-bottom: 13px; }
  .crumbs .sep { color: var(--text-faint); }
  .crumbs .cur { color: var(--text-strong); font-weight: 500; }
  .piv { background: none; border: none; color: var(--accent); cursor: pointer; font: inherit; padding: 0; }
  .piv:hover { text-decoration: underline; }
  .hhead { display: flex; align-items: baseline; gap: 12px; flex-wrap: wrap; }
  .htitle { font-size: 1.5rem; font-weight: 500; color: var(--text-strong); word-break: break-all; }
  .htag { font-size: 0.75rem; color: var(--text-dim); border: 1px solid var(--border); border-radius: 20px; padding: 2px 10px; }
  .hsub { color: var(--text-dim); font-size: 0.86rem; margin: 2px 0 16px; }
  .actions { display: flex; gap: 8px; margin: 0 0 16px; }
  .btn { background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px; padding: 6px 13px; font: inherit; font-size: 0.82rem; cursor: pointer; }
  .tabbar { display: flex; gap: 4px; border-bottom: 1px solid var(--border); margin-bottom: 16px; flex-wrap: wrap; }
  .tab { padding: 8px 13px; font: inherit; font-size: 0.84rem; color: var(--tab-inactive); background: none; border: none; border-bottom: 2px solid transparent; cursor: pointer; display: flex; align-items: center; gap: 6px; }
  .tab.on { color: var(--text-strong); border-color: var(--accent); }
  .tab.dis { color: var(--text-faint); cursor: default; }
  .soon { font-size: 0.6rem; color: var(--text-faint); border: 1px solid var(--pill-border); border-radius: 8px; padding: 0 6px; }
  .cards { display: grid; grid-template-columns: 1fr 1fr; gap: 13px; }
  @media (max-width: 760px) { .cards { grid-template-columns: 1fr; } }
  .card { background: var(--card); border: 1px solid var(--border-soft); border-radius: 10px; padding: 13px 15px; }
  .card.span { grid-column: 1 / -1; }
  .card.hero { border-color: var(--accent); box-shadow: 0 0 0 1px var(--accent-ring); }
  .chead { display: flex; align-items: center; justify-content: space-between; margin-bottom: 9px; gap: 8px; }
  .chead h3 { margin: 0; font-size: 0.68rem; text-transform: uppercase; letter-spacing: .06em; color: var(--text-dim); font-weight: 600; }
  .src { font-size: 0.62rem; color: var(--text-faint); font-family: var(--font-mono); }
  .srcs { display: flex; flex-wrap: wrap; gap: 4px 10px; justify-content: flex-end; }
  dl { display: grid; grid-template-columns: 140px 1fr; gap: 6px 12px; margin: 0; font-size: 0.85rem; }
  dt { color: var(--text-dim); }
  dd { margin: 0; color: var(--text); }
  .big { font-size: 1rem; color: var(--text-strong); }
  .dim { color: var(--text-dim); font-size: 0.82rem; }
  .pill { display: inline-flex; font-size: 0.66rem; padding: 1px 8px; border-radius: 20px; border: 1px solid var(--border); color: var(--text-dim); }
  .pill.soon { color: var(--text-faint); }
  .pill.good { color: var(--good, #5ad17a); border-color: transparent; background: var(--good-bg, #102516); }
  .pill.bad { color: var(--bad, #ff6b6b); border-color: transparent; }
  .lbl { color: var(--text-dim); font-size: 0.7rem; text-transform: uppercase; letter-spacing: .04em; margin: 8px 0 4px; }
  .lst { display: flex; flex-wrap: wrap; gap: 6px 14px; }
  table.rec { width: 100%; border-collapse: collapse; font-size: 0.82rem; }
  .rec td { padding: 4px 8px; border-bottom: 1px solid var(--border-soft); vertical-align: top; }
  .rec .rt { color: var(--accent); width: 60px; }
  .rec .rv { word-break: break-all; }
  .err { color: var(--warn-text); font-size: 0.78rem; margin: 8px 0 0; }
  .cap { color: var(--text-dim); font-size: 0.78rem; margin: 6px 0 0; }
  .pblock { margin-top: 12px; }
  .phead { display: flex; justify-content: space-between; align-items: baseline; font-size: 0.74rem; text-transform: uppercase; letter-spacing: 0.04em; color: var(--text-faint); margin-bottom: 6px; }
  .chips { display: flex; flex-wrap: wrap; gap: 6px; }
  .pchip { display: inline-flex; gap: 5px; align-items: baseline; font-size: 0.76rem; padding: 2px 9px; border-radius: 7px; border: 1px solid var(--border-soft); background: var(--card); }
  .pchip.more { color: var(--text-dim); border-style: dashed; }
  .mono { font-family: var(--font-mono); }
</style>
