<script lang="ts">
  import { AnalyzeFirewall } from "../../../wailsjs/go/app/App.js";

  let configText = "";
  let busy = false;
  let error = "";
  let analyses: any[] = [];
  let vIdx = 0;
  let openFlow = -1;
  let fileInput: HTMLInputElement;

  $: a = analyses.length ? analyses[vIdx] : null;

  async function analyze() {
    const v = configText.trim();
    if (!v) {
      analyses = [];
      return;
    }
    error = "";
    analyses = [];
    openFlow = -1;
    busy = true;
    try {
      analyses = (await AnalyzeFirewall(v, "fortigate")) ?? [];
      vIdx = 0;
    } catch (e) {
      error = String(e);
    } finally {
      busy = false;
    }
  }

  function loadFile(e: Event) {
    const f = (e.target as HTMLInputElement).files?.[0];
    if (!f) return;
    const r = new FileReader();
    r.onload = () => {
      configText = String(r.result ?? "");
    };
    r.readAsText(f);
  }

  function clearAll() {
    configText = "";
    analyses = [];
    error = "";
    openFlow = -1;
  }

  const policyById = (an: any, id: number) =>
    (an?.policies ?? []).find((p: any) => p.id === id);

  function toggleFlow(i: number) {
    openFlow = openFlow === i ? -1 : i;
  }
</script>

<div class="tool">
  <p class="lede">
    Drop or paste an exported <strong>FortiGate</strong> config. Meridian parses it
    locally — nothing leaves your machine — and shows the VLANs, what talks to what,
    and the counts. It's a reading of the ruleset at the VLAN level, not a live device.
  </p>

  <div class="card span">
    <textarea
      class="mono input"
      bind:value={configText}
      placeholder="Paste a FortiGate config export here, or use Load file…"
      spellcheck="false"
    ></textarea>
    <div class="row">
      <button class="go" on:click={analyze} disabled={busy || !configText.trim()}>
        {busy ? "Analyzing…" : "Analyze"}
      </button>
      <button class="ghost" on:click={() => fileInput.click()}>Load file…</button>
      <button class="ghost" on:click={clearAll} disabled={!configText && !analyses.length}>Clear</button>
      <input type="file" bind:this={fileInput} on:change={loadFile} accept=".conf,.txt,.cfg,text/plain" hidden />
    </div>
  </div>

  {#if error}
    <div class="banner bad"><span class="vlabel">Couldn't parse it</span><span class="vnote">{error}</span></div>
  {/if}

  {#if a}
    {#if analyses.length > 1}
      <div class="vdombar">
        <span class="k0">VDOM</span>
        <select bind:value={vIdx} on:change={() => (openFlow = -1)}>
          {#each analyses as an, i}
            <option value={i}>{an.vdom || "(root)"}</option>
          {/each}
        </select>
        <span class="vnote">{analyses.length} virtual domains in this config</span>
      </div>
    {/if}

    <div class="tiles">
      <div class="tile"><div class="n">{a.counts.vlans}</div><div class="l">VLANs</div></div>
      <div class="tile"><div class="n">{a.counts.policies}</div><div class="l">Policies</div></div>
      <div class="tile"><div class="n">{a.counts.addresses}</div><div class="l">Addr objects</div></div>
      <div class="tile"><div class="n">{a.counts.subnetsIps}</div><div class="l">Subnets / IPs</div></div>
      <div class="tile"><div class="n">{a.counts.services}</div><div class="l">Services</div></div>
      <div class="tile"><div class="n">{a.counts.zones}</div><div class="l">Zones</div></div>
    </div>

    {#if a.notes && a.notes.length}
      <div class="banner warn">
        <span class="vlabel">{a.notes.length} policies not fully analysed</span>
        {#each a.notes as note}<span class="vnote">{note}</span>{/each}
      </div>
    {/if}

    <div class="card span">
      <div class="k">VLANs <span class="count">· {(a.vlans ?? []).length}</span></div>
      {#if (a.vlans ?? []).length}
        <table class="mono"><thead>
          <tr><th>VLAN</th><th>Name</th><th>Interface</th><th>Subnet</th><th>Zone</th></tr>
        </thead><tbody>
          {#each a.vlans as v}
            <tr><td>{v.id}</td><td class="strong">{v.name}</td><td class="dim">{v.interface || "—"}</td><td class="dim">{v.subnet || "—"}</td><td class="dim">{v.zone || "—"}</td></tr>
          {/each}
        </tbody></table>
      {:else}<p class="empty">No VLAN interfaces found in this config.</p>{/if}
    </div>

    <div class="card span">
      <div class="k">What talks to what <span class="count">· {(a.flows ?? []).length} flows</span></div>
      {#if (a.flows ?? []).length}
        <table class="mono flows"><thead>
          <tr><th>Source</th><th>Destination</th><th>Service</th><th>Action</th></tr>
        </thead><tbody>
          {#each a.flows as f, i}
            <tr class="flowrow" class:open={openFlow === i} on:click={() => toggleFlow(i)}>
              <td class="strong">{f.src}</td>
              <td class="strong">{f.dst}</td>
              <td>{#each f.services ?? [] as s}<span class="pill">{s}</span> {/each}{#if !(f.services ?? []).length}<span class="dim">any</span>{/if}</td>
              <td class:allow={f.action === "allow"} class:deny={f.action === "deny"}>{f.action}</td>
            </tr>
            {#if openFlow === i}
              <tr class="drill"><td colspan="4">
                <div class="k0">Policies behind this flow</div>
                {#each f.policyIds ?? [] as pid}
                  {@const p = policyById(a, pid)}
                  {#if p}
                    <div class="pol">
                      <span class="polid">#{p.id}</span>
                      <span class="polname">{p.name || "(unnamed)"}</span>
                      <span class="dim">{(p.srcAddr ?? []).join(", ") || "all"} → {(p.dstAddr ?? []).join(", ") || "all"}</span>
                      <span class="dim">{(p.services ?? []).join(", ") || "any"}</span>
                    </div>
                  {/if}
                {/each}
              </td></tr>
            {/if}
          {/each}
        </tbody></table>
        <div class="note">Click a flow to see the exact policies, address objects and IPs behind it.</div>
      {:else}<p class="empty">No permitted or denied flows derived (no enabled policies, or all use negation).</p>{/if}
    </div>
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 20px; }
  .lede { max-width: 820px; margin: 0 0 14px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .mono { font-family: var(--mono, ui-monospace, monospace); }
  .input { width: 100%; min-height: 150px; resize: vertical; background: var(--well); border: 1px solid var(--field-border); border-radius: 7px; padding: 10px 12px; color: var(--text); font-size: 0.8rem; outline: none; line-height: 1.45; }
  .input:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .row { display: flex; gap: 8px; margin-top: 10px; align-items: center; }
  .go { background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px; padding: 8px 16px; font: inherit; font-size: 0.9rem; cursor: pointer; }
  .go:disabled { opacity: .6; cursor: default; }
  .ghost { background: transparent; border: 1px solid var(--border-soft); color: var(--text-dim); border-radius: 7px; padding: 7px 13px; font: inherit; font-size: 0.82rem; cursor: pointer; }
  .ghost:hover:not(:disabled) { color: var(--text); border-color: var(--field-border); }
  .ghost:disabled { opacity: .5; cursor: default; }

  .vdombar { display: flex; align-items: center; gap: 10px; margin: 14px 0 4px; font-size: 0.82rem; }
  .vdombar select { background: var(--well); border: 1px solid var(--field-border); color: var(--text); border-radius: 6px; padding: 5px 9px; font: inherit; font-size: 0.82rem; }
  .k0 { font-size: 0.72rem; text-transform: uppercase; letter-spacing: .05em; color: var(--text-faint); }

  .tiles { display: grid; grid-template-columns: repeat(6,1fr); gap: 10px; margin: 14px 0; max-width: 820px; }
  @media (max-width: 720px){ .tiles{ grid-template-columns: repeat(3,1fr);} }
  .tile { background: var(--card); border: 1px solid var(--border-soft); border-radius: 10px; padding: 12px 13px; }
  .tile .n { font-size: 1.5rem; font-weight: 650; font-family: var(--mono, monospace); color: var(--text-strong); }
  .tile .l { font-size: 0.68rem; text-transform: uppercase; letter-spacing: .04em; color: var(--text-faint); margin-top: 3px; }

  .banner { margin: 14px 0; padding: 11px 14px; border-radius: 9px; border: 1px solid var(--border-soft); display: flex; flex-direction: column; gap: 3px; max-width: 820px; }
  .banner.warn { border-color: var(--warn, #d9a441); background: color-mix(in srgb, var(--warn, #d9a441) 12%, transparent); }
  .banner.bad { border-color: var(--bad, #ff6b6b); background: color-mix(in srgb, var(--bad, #ff6b6b) 10%, transparent); }
  .vlabel { font-weight: 600; color: var(--text-strong); }
  .banner.warn .vlabel { color: var(--warn, #d9a441); }
  .banner.bad .vlabel { color: var(--bad, #ff6b6b); }
  .vnote { font-size: 0.78rem; color: var(--text-dim); overflow-wrap: anywhere; }

  .card { background: var(--card); border: 1px solid var(--border-soft); border-radius: 11px; padding: 13px 15px; max-width: 820px; margin-bottom: 14px; }
  .k { font-size: 0.74rem; text-transform: uppercase; letter-spacing: .04em; color: var(--text-faint); margin-bottom: 10px; }
  .k .count { color: var(--text-dim); }
  .empty { font-size: 0.82rem; color: var(--text-dim); margin: 2px 0; }

  table { width: 100%; border-collapse: collapse; font-size: 0.8rem; }
  th { text-align: left; color: var(--text-faint); font-weight: 500; padding: 3px 10px 7px 0; border-bottom: 1px solid var(--border-soft); white-space: nowrap; }
  td { padding: 6px 10px 6px 0; border-bottom: 1px solid var(--border-soft); vertical-align: top; overflow-wrap: anywhere; }
  tr:last-child td { border-bottom: none; }
  .strong { color: var(--text-strong); font-weight: 600; }
  .dim { color: var(--text-dim); }
  .pill { display: inline-block; font-family: var(--mono, monospace); font-size: 0.74rem; padding: 1px 7px; border-radius: 5px; background: var(--well); color: var(--text-dim); border: 1px solid var(--border-soft); }
  .allow { color: var(--good, #5ad17a); }
  .deny { color: var(--bad, #ff6b6b); }

  .flowrow { cursor: pointer; }
  .flowrow:hover td { background: color-mix(in srgb, var(--accent) 7%, transparent); }
  .flowrow.open td { background: color-mix(in srgb, var(--accent) 10%, transparent); }
  .drill td { background: var(--well); }
  .pol { display: flex; flex-wrap: wrap; gap: 10px; font-size: 0.78rem; padding: 3px 0; }
  .polid { color: var(--text-faint); font-family: var(--mono, monospace); }
  .polname { color: var(--text); font-weight: 600; }
  .note { font-size: 0.74rem; color: var(--text-faint); margin-top: 9px; }
</style>
