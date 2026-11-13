<script lang="ts">
  import { CLICommands, CLIInfo, InstallCLI } from "../../../wailsjs/go/app/App.js";

  let groups: any[] = [];
  let cli: any = { available: false, installed: false, supported: false, onPath: true, linkPath: "", path: "" };
  let filter = "";
  let copied = "";
  let cliBusy = false;
  let cliMsg = "";

  CLICommands().then((g) => (groups = g || [])).catch(() => {});
  CLIInfo().then((c) => (cli = c)).catch(() => {});

  async function copy(c: any) {
    const text = `meridian-cli ${c.name}`;
    try {
      await navigator.clipboard.writeText(text);
      copied = c.name;
      setTimeout(() => (copied = copied === c.name ? "" : copied), 1200);
    } catch {
      copied = "";
    }
  }

  async function installCLI() {
    cliBusy = true;
    cliMsg = "";
    try {
      cli = await InstallCLI();
      cliMsg = cli.onPath ? "Installed — run “meridian-cli” in any terminal." : `Installed to ${cli.linkPath}, not on your PATH yet.`;
    } catch (e: any) {
      cliMsg = String(e?.message ?? e);
    } finally {
      cliBusy = false;
    }
  }

  // Case-insensitive filter over name + description.
  $: f = filter.trim().toLowerCase();
  $: shown = groups
    .map((g) => ({
      ...g,
      commands: g.commands.filter((c: any) => !f || c.name.toLowerCase().includes(f) || c.desc.toLowerCase().includes(f) || c.args.toLowerCase().includes(f)),
    }))
    .filter((g) => g.commands.length > 0);
</script>

<div class="tool">
  <p class="lede">
    Every Meridian tool is also a <strong>command-line</strong> subcommand, sharing the same engine — so you can
    script it, pipe it, or run it headless over SSH. Most commands take <span class="mono">--json</span>; streaming
    ones run until <span class="mono">Ctrl-C</span>. Click a command to copy it.
  </p>

  <div class="status">
    {#if cli.available}
      {#if cli.installed}
        <span class="pill ok">✓ Installed</span>
        <span class="sdim mono">{cli.path}</span>
      {:else if cli.supported}
        <button class="pill btn" on:click={installCLI} disabled={cliBusy}>{cliBusy ? "Installing…" : "Install command-line tool"}</button>
        <span class="sdim">adds <span class="mono">meridian-cli</span> to your PATH</span>
      {:else}
        <span class="sdim">Bundled with this app. On {cli.os}, add its folder to your PATH to use <span class="mono">meridian-cli</span>.</span>
      {/if}
      {#if cliMsg}<span class="sdim">{cliMsg}</span>{/if}
    {:else}
      <span class="sdim">Build it from the repo: <span class="mono">go build -o ~/.local/bin/meridian-cli ./cmd/meridian-cli</span></span>
    {/if}
  </div>

  <input class="field" bind:value={filter} placeholder="Filter commands… (e.g. tls, dns, snmp)" spellcheck="false" />

  {#each shown as g}
    <div class="group">
      <div class="gh">{g.name}{#if g.note}<span class="gnote">— {g.note}</span>{/if}</div>
      {#each g.commands as c}
        <button class="crow" class:copied={copied === c.name} on:click={() => copy(c)} title="Copy “meridian-cli {c.name}”">
          <code class="cmd"><span class="prog">meridian-cli</span> <span class="name">{c.name}</span> <span class="args">{c.args}</span></code>
          <span class="desc">{c.desc}</span>
          <span class="copy">{copied === c.name ? "copied ✓" : "copy"}</span>
        </button>
      {/each}
    </div>
  {/each}

  {#if f && shown.length === 0}
    <p class="empty">No command matches “{filter}”.</p>
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 30px; max-width: 920px; }
  .lede { margin: 0 0 12px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .mono { font-family: var(--font-mono, ui-monospace, monospace); }
  .status { display: flex; align-items: center; flex-wrap: wrap; gap: 8px 12px; margin-bottom: 12px; }
  .pill { font-size: 0.74rem; font-weight: 600; padding: 3px 10px; border-radius: 20px; }
  .pill.ok { background: color-mix(in srgb, var(--good, #3fb950) 16%, transparent); color: var(--good, #3fb950); }
  .pill.btn { background: var(--accent); color: var(--on-accent, #052e2b); border: none; cursor: pointer; }
  .pill.btn:disabled { opacity: .6; cursor: default; }
  .sdim { font-size: 0.74rem; color: var(--text-faint, var(--text-dim)); }
  .field { width: 100%; box-sizing: border-box; background: var(--well); border: 1px solid var(--field-border, var(--border)); border-radius: 8px; padding: 9px 12px; color: var(--text); font-size: 0.86rem; outline: none; margin-bottom: 14px; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .group { margin-bottom: 16px; }
  .gh { font-size: 0.72rem; text-transform: uppercase; letter-spacing: .05em; color: var(--text-dim); font-weight: 700; margin-bottom: 6px; }
  .gnote { text-transform: none; letter-spacing: 0; font-weight: 400; color: var(--text-faint, var(--text-dim)); margin-left: 6px; }
  .crow { display: grid; grid-template-columns: 1fr auto; align-items: baseline; gap: 2px 12px; width: 100%; text-align: left; background: var(--card); border: 1px solid var(--border); border-radius: 9px; padding: 8px 12px; margin-bottom: 6px; cursor: pointer; }
  .crow:hover { border-color: var(--accent); }
  .crow.copied { border-color: var(--good, #3fb950); }
  .cmd { font-family: var(--font-mono, ui-monospace, monospace); font-size: 0.8rem; grid-column: 1; min-width: 0; overflow-wrap: anywhere; }
  .prog { color: var(--text-faint, var(--text-dim)); }
  .name { color: var(--accent); font-weight: 600; }
  .args { color: var(--text-dim); }
  .desc { grid-column: 1; font-size: 0.78rem; color: var(--text-dim); }
  .copy { grid-column: 2; grid-row: 1 / span 2; align-self: center; font-size: 0.7rem; color: var(--text-faint, var(--text-dim)); white-space: nowrap; }
  .crow.copied .copy { color: var(--good, #3fb950); }
  .empty { font-size: 0.82rem; color: var(--text-dim); }
</style>
