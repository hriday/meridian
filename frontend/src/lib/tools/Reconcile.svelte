<script lang="ts">
  import { ReconcileLAN, ApplyReconcileLAN } from "../../../wailsjs/go/app/App.js";

  let loading = false;
  let applying = false;
  let error = "";
  let locked = false;
  let report: any = null;
  let applied: any = null;

  let importNew = true;
  let updateMAC = false;
  let deprecateMissing = false;

  async function run() {
    if (loading) return;
    loading = true;
    error = "";
    locked = false;
    applied = null;
    report = null;
    try {
      report = await ReconcileLAN();
    } catch (e: any) {
      const msg = String(e?.message ?? e);
      if (/lock/i.test(msg)) {
        locked = true;
      } else {
        error = msg;
      }
    } finally {
      loading = false;
    }
  }

  async function apply() {
    if (applying || !report) return;
    applying = true;
    error = "";
    try {
      applied = await ApplyReconcileLAN(report.findings || [], importNew, updateMAC, deprecateMissing);
      // Refresh so the applied changes move out of the actionable buckets.
      await run();
    } catch (e: any) {
      error = String(e?.message ?? e);
    } finally {
      applying = false;
    }
  }

  const of = (cat: string) => (report?.findings || []).filter((f: any) => f.category === cat);
  $: newHosts = of("new");
  $: macChanged = of("mac-change");
  $: missing = of("missing");
  $: actionable =
    (importNew && newHosts.length) ||
    (updateMAC && macChanged.length) ||
    (deprecateMissing && missing.length);
</script>

<div class="tool">
  <p class="lede">
    Sweep the local subnet with <strong>ARP</strong> and diff the live hosts against your saved
    <strong>IP-space inventory</strong>: untracked hosts, <strong>MAC changes</strong> (a different device answering
    for a known IP — a possible rogue or spoof), and records that no longer answer. Review, then import or update in
    one step. Needs elevated privileges (raw L2), and an <strong>unlocked</strong> inventory.
  </p>

  <div class="row">
    <button class="btn" on:click={run} disabled={loading || applying}>
      {loading ? "Sweeping…" : "Reconcile with live sweep"}
    </button>
  </div>

  {#if locked}
    <div class="empty">Your IP-space inventory is locked. Open the <strong>IP space</strong> tool and unlock it, then run this again.</div>
  {/if}
  {#if error}<p class="err">{error}</p>{/if}

  {#if report}
    <div class="summary">
      <span class="pill new">{report.new} new</span>
      <span class="pill warn">{report.macChange} MAC change</span>
      <span class="pill miss">{report.missing} missing</span>
      <span class="pill ok">{report.inSync} in sync</span>
      <span class="sub">on {report.subnet}</span>
    </div>

    {#if newHosts.length}
      <div class="sec">
        <div class="sechead">New — untracked on the LAN <span class="count">{newHosts.length}</span></div>
        {#each newHosts as f}
          <div class="card">
            <span class="ip mono">{f.ip}</span>
            <span class="mac mono">{f.mac}</span>
            <span class="vendor">{f.vendor || ""}</span>
          </div>
        {/each}
      </div>
    {/if}

    {#if macChanged.length}
      <div class="sec">
        <div class="sechead warn">MAC changed — possible rogue / spoof <span class="count">{macChanged.length}</span></div>
        {#each macChanged as f}
          <div class="card">
            <span class="ip mono">{f.ip}</span>
            <span class="mac mono">{f.mac}</span>
            <span class="was mono">was {f.knownMac}</span>
            <span class="vendor">{f.name || ""}</span>
          </div>
        {/each}
      </div>
    {/if}

    {#if missing.length}
      <div class="sec">
        <div class="sechead">Missing — in inventory, didn't answer <span class="count">{missing.length}</span></div>
        {#each missing as f}
          <div class="card">
            <span class="ip mono">{f.ip}</span>
            <span class="mac mono">{f.mac || "—"}</span>
            <span class="vendor">{f.name || ""}</span>
          </div>
        {/each}
      </div>
    {/if}

    {#if newHosts.length || macChanged.length || missing.length}
      <div class="apply">
        <label><input type="checkbox" bind:checked={importNew} /> Import {newHosts.length} new host(s)</label>
        <label><input type="checkbox" bind:checked={updateMAC} /> Update {macChanged.length} changed MAC(s)</label>
        <label><input type="checkbox" bind:checked={deprecateMissing} /> Deprecate {missing.length} missing</label>
        <button class="btn apply-btn" on:click={apply} disabled={applying || !actionable}>
          {applying ? "Applying…" : "Apply to inventory"}
        </button>
      </div>
    {:else}
      <div class="empty">Everything on the LAN matches your inventory. Nothing to reconcile.</div>
    {/if}

    {#if applied}
      <p class="done">Applied: {applied.imported} imported, {applied.updated} updated, {applied.deprecated} deprecated.</p>
    {/if}
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 30px; max-width: 880px; }
  .lede { margin: 0 0 14px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .row { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; }
  .btn { background: var(--accent); color: var(--on-accent, #052e2b); border: 1px solid transparent; border-radius: 8px; padding: 8px 16px; font: inherit; font-size: 0.86rem; font-weight: 550; cursor: pointer; }
  .btn:disabled { opacity: .55; cursor: default; }
  .err { color: var(--danger, #e5484d); font-size: 0.82rem; margin: 12px 0 0; }
  .empty { margin-top: 16px; padding: 12px 15px; border: 1px solid var(--border); border-radius: 10px; background: var(--card); font-size: 0.84rem; color: var(--text-dim); }
  .summary { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; margin: 18px 0 6px; }
  .pill { font-size: 0.74rem; font-weight: 650; padding: 3px 10px; border-radius: 20px; border: 1px solid var(--border); }
  .pill.new { background: var(--accent); color: var(--on-accent, #052e2b); border-color: transparent; }
  .pill.warn { background: color-mix(in srgb, var(--danger, #e5484d) 16%, transparent); color: var(--danger, #e5484d); border-color: color-mix(in srgb, var(--danger, #e5484d) 40%, transparent); }
  .pill.miss { background: var(--well); color: var(--text-dim); }
  .pill.ok { background: var(--well); color: var(--text-dim); }
  .sub { font-size: 0.76rem; color: var(--text-dim); margin-left: 4px; }
  .sec { margin-top: 16px; }
  .sechead { font-size: 0.74rem; text-transform: uppercase; letter-spacing: .05em; color: var(--text-dim); font-weight: 700; margin-bottom: 8px; display: flex; align-items: center; gap: 8px; }
  .sechead.warn { color: var(--danger, #e5484d); }
  .count { background: var(--well); border: 1px solid var(--border); border-radius: 20px; padding: 1px 8px; font-size: 0.72rem; color: var(--text); }
  .card { display: flex; align-items: center; gap: 14px; border: 1px solid var(--border); border-radius: 10px; padding: 9px 13px; margin-bottom: 7px; background: var(--card); font-size: 0.84rem; }
  .ip { min-width: 120px; color: var(--text-strong, var(--text)); font-weight: 600; }
  .mac { min-width: 150px; color: var(--text-dim); }
  .was { color: var(--danger, #e5484d); }
  .vendor { color: var(--text-dim); }
  .apply { display: flex; align-items: center; gap: 16px; flex-wrap: wrap; margin-top: 18px; padding-top: 14px; border-top: 1px solid var(--border); }
  .apply label { font-size: 0.82rem; color: var(--text-dim); display: flex; align-items: center; gap: 6px; }
  .apply-btn { margin-left: auto; }
  .done { margin-top: 12px; font-size: 0.84rem; color: var(--text); }
  .mono { font-family: var(--font-mono, ui-monospace, monospace); }
</style>
