<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { TftpStart, TftpStop, TftpList, TftpPreview, TftpSave } from "../../../wailsjs/go/app/App.js";

  const POLL_MS = 1000;
  let port = 6969;
  let running = false;
  let files: any[] = [];
  let selectedId: number | null = null;
  let preview = "";
  let err = "";
  let saved = "";
  let timer: any = null;
  let polling = false;

  async function poll() {
    if (polling) return;
    polling = true;
    try {
      const v = await TftpList();
      running = v.running;
      if (v.port) port = v.port;
      files = v.files || [];
      if (selectedId != null && !files.some((f) => f.id === selectedId)) {
        selectedId = null; preview = "";
      }
    } catch {
      /* transient */
    } finally {
      polling = false;
    }
  }

  async function start() {
    err = ""; saved = "";
    try {
      await TftpStart(port);
    } catch (e) {
      err = String(e); running = false; return;
    }
    await poll();
  }

  async function stop() {
    try { await TftpStop(); } catch {}
    running = false; files = []; selectedId = null; preview = ""; saved = "";
  }

  async function select(f: any) {
    selectedId = f.id;
    saved = "";
    try { preview = await TftpPreview(f.id); } catch { preview = ""; }
  }

  async function save(f: any) {
    try {
      const path = await TftpSave(f.id);
      if (path) saved = `Saved to ${path}`;
    } catch (e) { err = String(e); }
  }

  onMount(() => { poll(); timer = setInterval(poll, POLL_MS); });
  onDestroy(() => { if (timer) clearInterval(timer); });

  const human = (b: number) => (b >= 1e6 ? (b / 1e6).toFixed(1) + " MB" : b >= 1e3 ? (b / 1e3).toFixed(1) + " KB" : b + " B");
  const fmtTime = (t: string) => { try { return new Date(t).toLocaleTimeString("en-GB"); } catch { return ""; } };
</script>

<div class="tool">
  <p class="lede">
    TFTP grab — a tiny TFTP server that catches files your devices push to it: point a switch/router at
    this machine and <span class="mono">copy running-config tftp://…</span> drops the config straight here,
    where you can read it and save it. In-memory and ephemeral — no TFTP server to set up, cleared on Stop.
  </p>
  <p class="help">
    On the device: <span class="mono">copy running-config tftp://&lt;this-host&gt;:&lt;port&gt;/&lt;name&gt;</span>
    (port 69 needs admin, so this defaults to <strong>6969</strong> — set it on the device too). Received files
    are held in memory; click one to preview, Save to write it to disk. <strong>Receive-only</strong>: it accepts
    uploads and refuses read requests, so it can't serve files out.
  </p>

  <div class="row">
    <span class="lbl">Listen UDP</span>
    <input class="field port mono" type="number" min="1" max="65535" bind:value={port} disabled={running} aria-label="Listen port" />
    {#if running}
      <button type="button" class="stop" on:click={stop}>Stop</button>
    {:else}
      <button type="button" class="go" on:click={start}>Start</button>
    {/if}
    <span class="spacer"></span>
    <span class="status"><span class="dot" class:on={running}></span>{running ? `listening on :${port}` : "stopped"}{files.length ? ` · ${files.length} file${files.length === 1 ? "" : "s"}` : ""}</span>
  </div>

  {#if err}<div class="banner bad"><span class="vnote">{err}</span></div>{/if}

  {#if files.length}
    <div class="grid">
      <div class="head"><span>File</span><span>Source</span><span class="r">Size</span><span class="r">Time</span><span></span></div>
      {#each files as f (f.id)}
        <div class="drow" class:on={selectedId === f.id} on:click={() => select(f)} role="button" tabindex="0">
          <div class="nm mono">{f.name}</div>
          <div class="src mono">{f.source}</div>
          <div class="sz mono r">{human(f.size)}</div>
          <div class="ts mono r">{fmtTime(f.t)}</div>
          <button class="savebtn" on:click|stopPropagation={() => save(f)}>Save</button>
        </div>
      {/each}
    </div>
    {#if saved}<p class="caption ok">{saved}</p>{/if}
    {#if selectedId != null}
      <div class="preview"><pre class="mono">{preview}</pre></div>
    {/if}
  {:else if running}
    <p class="caption">Waiting for an upload on :{port} — push a file from a device and it'll appear here.</p>
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 20px; }
  .lede { max-width: 960px; margin: 0 0 12px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .help { max-width: 960px; margin: 0 0 14px; padding: 9px 13px; border-left: 2px solid var(--accent); background: var(--well); border-radius: 0 7px 7px 0; font-size: 0.78rem; color: var(--text-dim); line-height: 1.5; }
  .help strong { color: var(--text); font-weight: 600; }
  .mono { font-family: var(--mono, ui-monospace, monospace); }

  .row { display: flex; gap: 8px; align-items: center; max-width: 960px; }
  .lbl { font-size: 0.78rem; color: var(--text-dim); }
  .field { background: var(--well); border: 1px solid var(--field-border); border-radius: 7px; padding: 8px 11px; color: var(--text); font-size: 0.86rem; outline: none; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .field:disabled { opacity: .6; }
  .port { width: 92px; }
  .spacer { flex: 1; }
  .go { background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px; padding: 0 15px; height: 36px; font: inherit; font-size: 0.86rem; cursor: pointer; }
  .stop { background: transparent; border: 1px solid var(--border-soft); color: var(--text-dim); border-radius: 7px; padding: 0 14px; height: 36px; font: inherit; font-size: 0.84rem; cursor: pointer; }
  .status { display: flex; align-items: center; gap: 8px; font-size: 0.8rem; color: var(--text-dim); }
  .dot { width: 8px; height: 8px; border-radius: 50%; background: var(--text-faint); flex: none; }
  .dot.on { background: var(--good); }

  .banner { margin: 12px 0 0; padding: 10px 13px; border-radius: 9px; border: 1px solid var(--bad, #ff6b6b); background: color-mix(in srgb, var(--bad,#ff6b6b) 10%, transparent); max-width: 960px; }
  .vnote { font-size: 0.8rem; color: var(--text-dim); }

  .grid { max-width: 960px; margin-top: 14px; border: 1px solid var(--border-soft); border-radius: 11px; overflow: hidden; background: var(--card); }
  .head, .drow { display: grid; grid-template-columns: 2.2fr 1.4fr 0.8fr 0.9fr 60px; align-items: center; gap: 14px; padding: 8px 16px; }
  .head { font-size: 0.6rem; text-transform: uppercase; letter-spacing: .08em; color: var(--text-faint); border-bottom: 1px solid var(--border-soft); background: var(--panel); }
  .drow { border-top: 1px solid var(--border-soft); font-size: 0.82rem; cursor: pointer; }
  .drow:first-of-type { border-top: none; }
  .drow:hover { background: var(--hover); }
  .drow.on { background: var(--active); }
  .nm { color: var(--text-strong); font-weight: 600; }
  .src { color: var(--text-dim); }
  .sz, .ts { color: var(--text-dim); }
  .r { text-align: right; }
  .savebtn { background: transparent; border: 1px solid var(--border-soft); color: var(--text-dim); border-radius: 6px; padding: 3px 9px; font: inherit; font-size: 0.72rem; cursor: pointer; }
  .savebtn:hover { color: var(--text); border-color: var(--field-border); }

  .preview { max-width: 960px; margin-top: 10px; border: 1px solid var(--border-soft); border-radius: 10px; background: var(--well); overflow: auto; max-height: 48vh; }
  .preview pre { margin: 0; padding: 12px 14px; font-size: 0.76rem; color: var(--text); white-space: pre-wrap; word-break: break-word; }
  .caption { max-width: 960px; margin-top: 12px; font-size: 0.72rem; color: var(--text-faint); }
  .caption.ok { color: var(--good); }
</style>
