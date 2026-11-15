<script lang="ts">
  import { onMount } from "svelte";
  import { RunbookProfiles, RunRunbook, RunbookRender, SaveReport } from "../../../wailsjs/go/app/App.js";

  let profiles: any[] = [];
  let profile = "";
  let target = "";
  let loading = false;
  let error = "";
  let res: any = null;
  let exportMsg = "";

  // Export the already-run report — no re-running the tools.
  async function exportFile(fmt: string, ext: string) {
    exportMsg = "";
    try {
      const content = await RunbookRender(res, fmt);
      const base = `${res.profile || "runbook"}-${res.target}`.replace(/[^\w.\-]+/g, "_");
      const path = await SaveReport(`${base}.${ext}`, content);
      if (path) exportMsg = "Saved to " + path;
    } catch (e: any) {
      exportMsg = String(e?.message ?? e);
    }
  }

  // PDF: render the report's HTML into an off-screen iframe and print it, so the
  // OS "Save as PDF" captures just the report (not the whole app window).
  async function exportPDF() {
    exportMsg = "";
    try {
      const html = await RunbookRender(res, "html");
      const iframe = document.createElement("iframe");
      iframe.style.cssText = "position:fixed;right:0;bottom:0;width:0;height:0;border:0";
      iframe.onload = () => {
        try {
          iframe.contentWindow?.focus();
          iframe.contentWindow?.print();
        } finally {
          setTimeout(() => iframe.remove(), 1500);
        }
      };
      document.body.appendChild(iframe);
      iframe.srcdoc = html;
    } catch (e: any) {
      exportMsg = String(e?.message ?? e);
    }
  }

  onMount(async () => {
    try {
      profiles = await RunbookProfiles();
      if (profiles.length) profile = profiles[0].id;
    } catch (e: any) {
      error = String(e?.message ?? e);
    }
  });

  $: current = profiles.find((p) => p.id === profile);

  async function run() {
    if (loading || !target.trim() || !profile) return;
    loading = true;
    error = "";
    res = null;
    try {
      res = await RunRunbook(profile, target.trim());
    } catch (e: any) {
      error = String(e?.message ?? e);
    } finally {
      loading = false;
    }
  }
</script>

<div class="tool">
  <p class="lede">
    <strong>Runbooks</strong> run a whole profile of tools against one target and gather the results into a single
    report — one click instead of a dozen. Great for onboarding a domain or triaging a host. Each step is the same
    engine as the individual tool.
  </p>

  <div class="row">
    <select class="sel" bind:value={profile} aria-label="Runbook profile">
      {#each profiles as p}<option value={p.id}>{p.name} ({p.steps} checks)</option>{/each}
    </select>
    <input class="field mono" bind:value={target} placeholder="acme.com  /  10.0.0.5" autocomplete="off" spellcheck="false"
      on:keydown={(e) => e.key === "Enter" && run()} aria-label="Target" />
    <button class="btn" on:click={run} disabled={loading || !target.trim()}>{loading ? "Running…" : "Run"}</button>
  </div>
  {#if current}<p class="muted">{current.desc}</p>{/if}
  {#if loading}<p class="muted">Running {current?.steps} checks — a full path trace is one of them, so give it a moment…</p>{/if}

  {#if error}<p class="err">{error}</p>{/if}

  {#if res}
    <div class="head">
      <span class="h">{res.name} — <span class="mono">{res.target}</span></span>
      <span class="sub">{res.steps?.length} checks · {new Date(res.generated).toLocaleString()}</span>
      <span class="exp">
        <span class="xlbl">Export</span>
        <button class="xb" on:click={() => exportFile("md", "txt")}>Text</button>
        <button class="xb" on:click={() => exportFile("json", "json")}>JSON</button>
        <button class="xb" on:click={() => exportFile("html", "html")}>HTML</button>
        <button class="xb" on:click={exportPDF}>PDF</button>
      </span>
    </div>
    {#if exportMsg}<p class="muted">{exportMsg}</p>{/if}
    {#each res.steps as s}
      <div class="step">
        <div class="shead">
          <span class="cmd mono">{s.command}</span>
          <span class="args mono">{(s.args || []).join(" ")}</span>
          <span class="dot {s.ok ? 'ok' : 'bad'}">{s.ok ? "✓" : "⚠"}</span>
        </div>
        <pre class="out">{s.output || "(no output)"}</pre>
      </div>
    {/each}
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 30px; max-width: 900px; }
  .lede { margin: 0 0 14px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .row { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
  .sel { background: var(--well); border: 1px solid var(--field-border, var(--border)); border-radius: 7px; padding: 8px 10px; color: var(--text); font: inherit; font-size: 0.86rem; cursor: pointer; }
  .field { flex: 1; min-width: 180px; background: var(--well); border: 1px solid var(--field-border, var(--border)); border-radius: 7px; padding: 8px 11px; color: var(--text); font-size: 0.88rem; outline: none; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .btn { background: var(--accent); color: var(--on-accent, #052e2b); border: 1px solid transparent; border-radius: 8px; padding: 8px 18px; font: inherit; font-size: 0.86rem; font-weight: 550; cursor: pointer; }
  .btn:disabled { opacity: .55; cursor: default; }
  .muted { color: var(--text-dim); font-size: 0.78rem; margin: 9px 0 0; }
  .err { color: var(--danger, #e5484d); font-size: 0.82rem; margin: 12px 0 0; }
  .head { margin: 20px 0 8px; display: flex; align-items: baseline; gap: 12px; flex-wrap: wrap; }
  .h { font-size: 1rem; font-weight: 650; color: var(--text-strong, var(--text)); }
  .sub { font-size: 0.74rem; color: var(--text-dim); }
  .exp { margin-left: auto; display: flex; align-items: center; gap: 6px; }
  .xlbl { font-size: 0.7rem; text-transform: uppercase; letter-spacing: .05em; color: var(--text-faint); }
  .xb { background: var(--well); border: 1px solid var(--border); border-radius: 6px; padding: 4px 10px; color: var(--text); font: inherit; font-size: 0.76rem; cursor: pointer; }
  .xb:hover { border-color: var(--accent); color: var(--text-strong, var(--text)); }
  .step { margin-top: 12px; border: 1px solid var(--border); border-radius: 10px; overflow: hidden; background: var(--card); }
  .shead { display: flex; align-items: center; gap: 10px; padding: 8px 12px; background: var(--well); border-bottom: 1px solid var(--border); }
  .cmd { font-weight: 650; color: var(--accent); font-size: 0.86rem; }
  .args { color: var(--text-dim); font-size: 0.76rem; flex: 1; }
  .dot { font-weight: 700; }
  .dot.ok { color: var(--accent); }
  .dot.bad { color: var(--danger, #e5484d); }
  .out { margin: 0; padding: 11px 13px; font-family: var(--font-mono, ui-monospace, monospace); font-size: 0.76rem; line-height: 1.45; color: var(--text); white-space: pre-wrap; word-break: break-word; max-height: 360px; overflow: auto; }
  .mono { font-family: var(--font-mono, ui-monospace, monospace); }
</style>
