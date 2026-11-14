<script lang="ts">
  import { ServiceStatus, ServiceStatusKnown } from "../../../wailsjs/go/app/App.js";
  import { BrowserOpenURL } from "../../../wailsjs/runtime/runtime.js";

  let target = "";
  let known: string[] = [];
  let status: any = null;
  let loading = false;
  let error = "";

  ServiceStatusKnown().then((k) => (known = k || [])).catch(() => {});

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
    Check whether an online service is <strong>having an outage right now</strong>, from its official
    <strong>Statuspage</strong> feed (authoritative status, not crowd reports). Pick a known service or paste any
    status-page URL (e.g. <span class="mono">status.example.com</span>). Free — no account.
  </p>

  <div class="row">
    <input class="field mono" bind:value={target} placeholder="github  ·  or  https://status.example.com" spellcheck="false"
      on:keydown={(e) => e.key === "Enter" && check()} />
    <button class="btn" on:click={() => check()} disabled={loading || !target.trim()}>{loading ? "Checking…" : "Check"}</button>
  </div>

  <div class="chips">
    {#each known as k}
      <button class="chip" on:click={() => check(k)}>{k}</button>
    {/each}
  </div>

  {#if error}<p class="err">{error}</p>{/if}

  {#if status}
    <div class="card {status.operational ? 'ok' : 'bad'}">
      <div class="shead">
        <span class="dot {status.operational ? 'ok' : 'bad'}"></span>
        <span class="sname">{status.service}</span>
        <span class="sdesc">{status.description}</span>
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
