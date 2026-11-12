<script lang="ts">
  import { ThreatIntelCheck } from "../../../wailsjs/go/app/App.js";

  let input = "";
  let hits: any[] = [];
  let loading = false;
  let error = "";
  let ran = false;

  $: listedCount = hits.filter((h) => h.listed).length;

  async function run() {
    if (!input.trim()) return;
    loading = true;
    error = "";
    try {
      hits = (await ThreatIntelCheck(input)) || [];
      ran = true;
    } catch (e: any) {
      error = String(e?.message ?? e);
      hits = [];
      ran = true;
    } finally {
      loading = false;
    }
  }
</script>

<div class="tool">
  <p class="lede">
    Paste IP addresses and check them against <strong>abuse.ch Feodo Tracker</strong> — the command-and-control
    servers of botnets like <em>Emotet, Dridex, QakBot</em> and <em>TrickBot</em>. A hit means the IP has been
    seen acting as malware C2 infrastructure. The blocklist is a free, keyless public feed; only the IPs you paste
    are looked up.
  </p>

  <textarea
    class="field mono"
    bind:value={input}
    rows="5"
    placeholder={"One IP per line, or comma/space separated\n91.240.118.168\n162.243.103.246, 8.8.8.8"}
    spellcheck="false"
  ></textarea>

  <div class="row">
    <button class="btn" on:click={run} disabled={loading || !input.trim()}>
      {loading ? "Checking…" : "Check IPs"}
    </button>
    {#if ran && !error}
      <span class="summary" class:bad={listedCount > 0}>
        {#if listedCount > 0}
          {listedCount} of {hits.length} flagged as known botnet C2
        {:else}
          None of {hits.length} on the C2 blocklist
        {/if}
      </span>
    {/if}
  </div>

  {#if error}
    <p class="err">{error}</p>
  {/if}

  {#if ran && !error && hits.length}
    <div class="card">
      <table>
        <thead>
          <tr><th>IP</th><th>Verdict</th><th>Malware</th><th>Status</th><th>AS / country</th><th>Last online</th></tr>
        </thead>
        <tbody>
          {#each hits as h}
            <tr class:flagged={h.listed}>
              <td class="mono ip">{h.ip}</td>
              <td>
                {#if h.listed}
                  <span class="badge bad">BOTNET C2</span>
                {:else}
                  <span class="badge ok">clean</span>
                {/if}
              </td>
              <td>{h.malware || "—"}</td>
              <td class="dim">{h.status || "—"}{#if h.port}&nbsp;:{h.port}{/if}</td>
              <td class="dim">{h.listed ? `${h.asName || "AS" + (h.asNumber || "?")} · ${h.country || "?"}` : "—"}</td>
              <td class="dim mono">{h.lastOnline || "—"}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
    <p class="foot">
      Data from <strong>abuse.ch Feodo Tracker</strong> (CC0). A "clean" verdict only means the IP isn't a <em>known</em>
      C2 on this feed — not that it's safe. Cross-check a hit with the <strong>Blocklist check</strong> tool for mail/spam
      reputation.
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
  .err { color: var(--danger, #e5484d); font-size: 0.82rem; margin: 12px 0 0; max-width: 760px; }
  .card { background: var(--card); border: 1px solid var(--border); border-radius: 11px; max-width: 900px; overflow: hidden; margin-top: 16px; }
  table { width: 100%; border-collapse: collapse; font-size: 0.82rem; }
  th { text-align: left; font-weight: 600; color: var(--text-dim); padding: 9px 12px; border-bottom: 1px solid var(--border); background: var(--well); font-size: 0.74rem; text-transform: uppercase; letter-spacing: .03em; }
  td { padding: 9px 12px; border-bottom: 1px solid var(--border-soft, var(--border)); vertical-align: middle; }
  tr:last-child td { border-bottom: 0; }
  tr.flagged { background: color-mix(in srgb, var(--danger, #e5484d) 8%, transparent); }
  .ip { font-weight: 600; color: var(--text); }
  .badge { display: inline-block; font-size: 0.7rem; font-weight: 700; letter-spacing: .03em; padding: 2px 8px; border-radius: 6px; }
  .badge.bad { background: var(--danger, #e5484d); color: #fff; }
  .badge.ok { background: var(--well); color: var(--text-dim); border: 1px solid var(--border); }
  .foot { max-width: 900px; font-size: 0.76rem; color: var(--text-faint, var(--text-dim)); margin: 12px 0 0; line-height: 1.6; }
</style>
