<script lang="ts">
  import { CertExpiryCheck } from "../../../wailsjs/go/app/App.js";

  let input = "";
  let rows: any[] = [];
  let loading = false;
  let ran = false;

  // Urgency buckets for color + sorting emphasis.
  function level(r: any): string {
    if (!r.ok) return "err";
    if (r.notYet) return "notyet";
    if (r.expired) return "expired";
    if (r.daysLeft <= 14) return "critical";
    if (r.daysLeft <= 30) return "warn";
    return "ok";
  }
  function label(r: any): string {
    if (!r.ok) return "unreachable";
    if (r.notYet) return "not valid yet";
    if (r.expired) return `expired ${-r.daysLeft}d ago`;
    if (r.daysLeft === 0) return "expires today";
    return `${r.daysLeft} day${r.daysLeft === 1 ? "" : "s"} left`;
  }

  $: worst = rows.filter((r) => r.ok && !r.notYet && r.daysLeft <= 14).length;

  async function run() {
    if (!input.trim()) return;
    loading = true;
    try {
      rows = (await CertExpiryCheck(input)) || [];
      ran = true;
    } finally {
      loading = false;
    }
  }
</script>

<div class="tool">
  <p class="lede">
    Paste a list of hosts and see <strong>how soon each TLS certificate expires</strong>, soonest first — so the one
    about to lapse is at the top. Each is a direct TLS dial (default port 443; add <span class="mono">:port</span> to
    override). Nothing but the connection itself is involved — no third-party service. For a deep look at one cert,
    use the <strong>TLS certificate</strong> tool.
  </p>

  <textarea
    class="field mono"
    bind:value={input}
    rows="5"
    placeholder={"One host per line, or comma/space separated\nexample.com\ninternal.example:8443\nhttps://api.example.com"}
    spellcheck="false"
  ></textarea>

  <div class="row">
    <button class="btn" on:click={run} disabled={loading || !input.trim()}>
      {loading ? "Checking…" : "Check certificates"}
    </button>
    {#if ran}
      <span class="summary" class:bad={worst > 0}>
        {#if worst > 0}
          {worst} expiring within 14 days
        {:else}
          nothing urgent ({rows.length} checked)
        {/if}
      </span>
    {/if}
  </div>

  {#if ran && rows.length}
    <div class="card">
      <table>
        <thead>
          <tr><th>Host</th><th>Expiry</th><th>Expires</th><th>Issuer</th></tr>
        </thead>
        <tbody>
          {#each rows as r}
            <tr class={level(r)}>
              <td class="mono host">{r.host}</td>
              <td><span class="badge {level(r)}">{label(r)}</span></td>
              <td class="dim mono">{r.ok ? r.notAfter : (r.error || "—")}</td>
              <td class="dim">{r.ok ? (r.issuer || "—") : "—"}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
    <p class="foot">
      An expiring cert means an outage when it lapses — renew well ahead. Unreachable hosts couldn't complete a TLS
      handshake (wrong port, firewall, or not serving TLS).
    </p>
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 30px; }
  .lede { max-width: 820px; margin: 0 0 14px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .mono { font-family: var(--font-mono, ui-monospace, monospace); }
  .dim { color: var(--text-dim); }
  .field { background: var(--well); border: 1px solid var(--field-border, var(--border)); border-radius: 8px; padding: 9px 12px; color: var(--text); font-size: 0.86rem; outline: none; width: 100%; max-width: 760px; box-sizing: border-box; resize: vertical; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .row { display: flex; align-items: center; gap: 14px; max-width: 760px; margin: 12px 0 0; }
  .btn { background: var(--accent); color: var(--on-accent, #052e2b); border: 1px solid transparent; border-radius: 8px; padding: 9px 16px; font: inherit; font-size: 0.86rem; font-weight: 550; cursor: pointer; white-space: nowrap; }
  .btn:disabled { opacity: .5; cursor: default; }
  .summary { font-size: 0.8rem; color: var(--text-dim); }
  .summary.bad { color: var(--danger, #e5484d); font-weight: 600; }
  .card { background: var(--card); border: 1px solid var(--border); border-radius: 11px; max-width: 900px; overflow: hidden; margin-top: 16px; }
  table { width: 100%; border-collapse: collapse; font-size: 0.82rem; }
  th { text-align: left; font-weight: 600; color: var(--text-dim); padding: 9px 12px; border-bottom: 1px solid var(--border); background: var(--well); font-size: 0.74rem; text-transform: uppercase; letter-spacing: .03em; }
  td { padding: 9px 12px; border-bottom: 1px solid var(--border-soft, var(--border)); vertical-align: middle; }
  tr:last-child td { border-bottom: 0; }
  tr.expired, tr.critical, tr.err { background: color-mix(in srgb, var(--danger, #e5484d) 7%, transparent); }
  tr.warn { background: color-mix(in srgb, var(--warn, #f5a623) 8%, transparent); }
  .host { font-weight: 600; color: var(--text); }
  .badge { display: inline-block; font-size: 0.72rem; font-weight: 650; padding: 2px 9px; border-radius: 6px; white-space: nowrap; }
  .badge.ok { background: var(--well); color: var(--text-dim); border: 1px solid var(--border); }
  .badge.warn { background: var(--warn, #f5a623); color: #3a2a00; }
  .badge.critical, .badge.expired { background: var(--danger, #e5484d); color: #fff; }
  .badge.err, .badge.notyet { background: var(--well); color: var(--text-dim); border: 1px dashed var(--border); }
  .foot { max-width: 900px; font-size: 0.76rem; color: var(--text-faint, var(--text-dim)); margin: 12px 0 0; line-height: 1.6; }
</style>
