<script lang="ts">
  import { LinkInspect } from "../../../wailsjs/go/app/App.js";
  import ToolHelp from "../ToolHelp.svelte";

  let report: any = null;
  let loading = false;
  let err = "";

  async function run() {
    err = "";
    report = null;
    loading = true;
    try {
      report = await LinkInspect(0); // 0 = default listen window
    } catch (e) {
      err = String(e);
    } finally {
      loading = false;
    }
  }

  function linkLabel(r: any): string {
    if (r.wireless) return "Wi-Fi (wireless)";
    if (r.linkKnown) {
      const speed = r.speedMbps ? `${r.speedMbps} Mb/s` : "";
      const dup = r.duplex ? `${r.duplex} duplex` : "duplex unknown";
      return [speed, dup].filter(Boolean).join(", ");
    }
    return "speed/duplex unknown on this system";
  }

  // The verdict's tone drives the banner colour: a detected hub/switch is a
  // finding (amber), a clean full-duplex read is reassuring (green), anything
  // hedged/inconclusive is neutral.
  function verdictTone(r: any): string {
    const v = (r.verdict || "").toLowerCase();
    if (v.includes("hub is almost certainly") || v.includes("switch is directly attached")) return "warn";
    if (v.includes("no hub or announced switch") || v.includes("no hub detected")) return "good";
    return "info";
  }
</script>

<div class="tool">
  <p class="lede">
    Is there a <strong>hub or switch between this machine and the router</strong>? A host usually can't tell —
    an unmanaged switch has no IP and announces nothing — so this gathers the signals that <em>do</em> betray one:
    the link's <strong>duplex</strong> (a hub forces half-duplex) and a managed switch's <strong>LLDP/CDP</strong>
    advertisement. It reports an honest, hedged verdict, blind spots and all.
  </p>

  <ToolHelp sudo="LLDP/CDP capture">
    Two signals. <strong>Link duplex</strong> is read locally and needs no privilege: a real hub shares the wire and
    forces <strong>half-duplex</strong>, while switches and routers run full-duplex — so a half-duplex link is a strong
    hub tell. <strong>LLDP/CDP</strong> is how a <em>managed</em> switch names itself (and the port you're on); capturing
    those frames needs elevated privilege (sudo / BPF group), and when it can't run the tool says so rather than implying
    nothing's there. <strong>Limits:</strong> an unmanaged switch is invisible to every check here, this can't count
    switches in the path, and on Wi-Fi a host can't see past the access point. For the full picture run the
    <strong>triage-l2</strong> recipe (it adds ARP neighbours and gateway reachability).
  </ToolHelp>

  <div class="searchrow">
    <button class="btn" on:click={run} disabled={loading}>{loading ? "Inspecting… (listening for switches)" : "Inspect link"}</button>
  </div>

  {#if err}<div class="banner bad"><span>{err}</span></div>{/if}

  {#if report}
    <div class="facts">
      <div class="fact"><span class="k">Egress</span><span class="v mono">{report.iface || "—"}</span></div>
      <div class="fact"><span class="k">Gateway</span><span class="v mono">{report.gateway || "unknown"}</span></div>
      <div class="fact"><span class="k">Link</span><span class="v">{linkLabel(report)}</span></div>
    </div>

    <div class="verdict {verdictTone(report)}">{report.verdict}</div>

    {#if report.findings && report.findings.length}
      <div class="card">
        {#each report.findings as f}
          <div class="findrow {f.level}">
            <span class="fmark {f.level}"></span>
            <span class="ftext">{f.text}</span>
          </div>
        {/each}
      </div>
    {/if}

    <p class="foot">
      Point-in-time; Meridian keeps no history. “No hub or switch detected” is never proof the path is clean — an
      unmanaged switch leaves no trace a host can see.
    </p>
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 30px; }
  .lede { max-width: 820px; margin: 0 0 12px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .mono { font-family: var(--font-mono, ui-monospace, monospace); }
  .searchrow { display: flex; align-items: center; gap: 12px; max-width: 720px; margin-bottom: 14px; }
  .btn { background: var(--accent); color: var(--on-accent, #052e2b); border: 1px solid transparent; border-radius: 8px; padding: 9px 18px; font: inherit; font-size: 0.86rem; font-weight: 550; cursor: pointer; white-space: nowrap; }
  .btn:disabled { opacity: .5; cursor: default; }
  .banner { display: flex; align-items: center; gap: 10px; max-width: 820px; padding: 9px 13px; border-radius: 9px; font-size: 0.84rem; margin-bottom: 12px; }
  .banner.bad { border: 1px solid var(--bad, #ff6b6b); background: color-mix(in srgb, var(--bad, #ff6b6b) 10%, transparent); color: var(--text); }

  .facts { display: flex; flex-wrap: wrap; gap: 10px 26px; max-width: 820px; margin-bottom: 14px; }
  .fact { display: flex; flex-direction: column; gap: 2px; }
  .fact .k { font-size: 0.66rem; text-transform: uppercase; letter-spacing: 0.05em; color: var(--text-faint, var(--text-dim)); }
  .fact .v { font-size: 0.9rem; color: var(--text); }

  .verdict { max-width: 820px; padding: 12px 15px; border-radius: 10px; font-size: 0.92rem; font-weight: 600; line-height: 1.45; margin-bottom: 12px; border: 1px solid var(--border); }
  .verdict.warn { border-color: var(--warn, #d6a531); background: color-mix(in srgb, var(--warn, #d6a531) 12%, transparent); }
  .verdict.good { border-color: var(--good, #3fb950); background: color-mix(in srgb, var(--good, #3fb950) 11%, transparent); }
  .verdict.info { background: var(--card); }

  .card { background: var(--card); border: 1px solid var(--border); border-radius: 11px; max-width: 820px; overflow: hidden; }
  .findrow { display: flex; gap: 10px; padding: 10px 14px; border-bottom: 1px solid var(--border-soft, var(--border)); font-size: 0.82rem; line-height: 1.45; color: var(--text); }
  .findrow:last-child { border-bottom: 0; }
  .fmark { width: 8px; height: 8px; border-radius: 50%; flex: none; margin-top: 6px; background: var(--text-faint, #888); }
  .fmark.found { background: var(--warn, #d6a531); }
  .fmark.good { background: var(--good, #3fb950); }
  .fmark.limit { background: var(--text-faint, #888); }
  .fmark.info { background: var(--accent, #58a6ff); }
  .findrow.limit .ftext { color: var(--text-dim); }

  .foot { max-width: 820px; font-size: 0.74rem; color: var(--text-faint, var(--text-dim)); margin: 12px 0 0; }
</style>
