<script lang="ts">
  import { Transcode, TranscodeDetect } from "../../../wailsjs/go/app/App.js";

  let input = "";
  let output = "";
  let error = "";
  let activeOp = "";
  let guesses: any[] = [];
  let copied = false;

  const encodes = [
    ["base64-encode", "Base64"], ["base64url-encode", "Base64 URL"],
    ["hex-encode", "Hex"], ["url-encode", "URL"],
  ];
  const decodes = [
    ["base64-decode", "Base64"], ["base64url-decode", "Base64 URL"],
    ["hex-decode", "Hex"], ["url-decode", "URL"],
  ];
  const jsons = [["json-pretty", "Pretty"], ["json-minify", "Minify"]];

  // Live "what is this?" detection.
  $: detect(input);
  let dseq = 0;
  async function detect(t: string) {
    const n = ++dseq;
    if (!t.trim()) { guesses = []; return; }
    try {
      const g = await TranscodeDetect(t);
      if (n === dseq) guesses = g || [];
    } catch { if (n === dseq) guesses = []; }
  }

  async function run(op: string) {
    activeOp = op;
    error = "";
    try {
      output = await Transcode(op, input);
    } catch (e: any) {
      output = "";
      error = String(e?.message ?? e);
    }
  }

  async function copy() {
    try { await navigator.clipboard.writeText(output); copied = true; setTimeout(() => (copied = false), 1200); } catch {}
  }
</script>

<div class="tool">
  <h2>Encode / decode</h2>
  <p class="sub">
    Base64, Base64-URL, hex, URL percent-encoding, and JSON pretty/minify — both ways, <strong>locally</strong>. No
    pasting (often sensitive) payloads into base64decode.org / jsonlint. Nothing is stored.
  </p>

  <textarea class="field mono" bind:value={input} rows="5" spellcheck="false" placeholder="Paste or type here…"></textarea>

  {#if guesses.length}
    <div class="detect">
      <span class="dl">Looks like:</span>
      {#each guesses as g}
        <button class="gchip" on:click={() => { output = g.result; error = ''; activeOp = ''; }}>{g.label}</button>
      {/each}
    </div>
  {/if}

  <div class="ops">
    <div class="opgroup"><span class="gl">Encode</span>{#each encodes as [op, name]}<button class="op" class:on={activeOp === op} on:click={() => run(op)}>{name}</button>{/each}</div>
    <div class="opgroup"><span class="gl">Decode</span>{#each decodes as [op, name]}<button class="op" class:on={activeOp === op} on:click={() => run(op)}>{name}</button>{/each}</div>
    <div class="opgroup"><span class="gl">JSON</span>{#each jsons as [op, name]}<button class="op" class:on={activeOp === op} on:click={() => run(op)}>{name}</button>{/each}</div>
  </div>

  {#if error}<div class="hint">{error}</div>{/if}

  {#if output}
    <div class="ohead"><span class="lbl">Output</span><button class="cp" on:click={copy}>{copied ? "copied" : "copy"}</button></div>
    <pre class="field mono box">{output}</pre>
  {/if}
</div>

<style>
  .tool { padding: 16px 20px 40px; max-width: 880px; }
  h2 { margin: 0 0 2px; font-size: 1.15rem; font-weight: 600; color: var(--text-strong); }
  .sub { margin: 0 0 14px; color: var(--text-dim); font-size: 0.86rem; line-height: 1.5; }
  .field { background: var(--well); border: 1px solid var(--field-border); border-radius: 8px; padding: 10px 12px; color: var(--text); font-size: 0.84rem; outline: none; width: 100%; resize: vertical; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .detect { display: flex; align-items: center; gap: 7px; flex-wrap: wrap; margin: 10px 0; }
  .dl { font-size: 0.74rem; color: var(--text-dim); }
  .gchip { font-size: 0.74rem; background: color-mix(in srgb, var(--accent) 14%, transparent); border: 1px solid var(--accent); color: var(--accent); border-radius: 20px; padding: 2px 11px; cursor: pointer; }
  .ops { display: flex; flex-wrap: wrap; gap: 16px; margin: 14px 0; }
  .opgroup { display: flex; align-items: center; gap: 6px; }
  .gl { font-size: 0.66rem; text-transform: uppercase; letter-spacing: .04em; color: var(--text-faint, var(--text-dim)); font-weight: 650; margin-right: 2px; }
  .op { font-size: 0.8rem; background: var(--well); border: 1px solid var(--border); border-radius: 7px; padding: 5px 12px; color: var(--text); cursor: pointer; }
  .op:hover { border-color: var(--accent); }
  .op.on { background: var(--accent); border-color: var(--accent); color: var(--on-accent, #052e2b); }
  .hint { background: var(--warn-bg); border: 1px solid var(--warn-border); color: var(--warn-text); border-radius: 7px; padding: 9px 12px; font-size: 0.82rem; margin: 10px 0; }
  .ohead { display: flex; align-items: center; justify-content: space-between; margin: 12px 0 5px; }
  .lbl { font-size: 0.68rem; text-transform: uppercase; letter-spacing: .04em; color: var(--text-faint, var(--text-dim)); font-weight: 650; }
  .cp { background: none; border: 1px solid var(--border); border-radius: 5px; padding: 1px 10px; font-size: 0.72rem; color: var(--text-dim); cursor: pointer; }
  .box { margin: 0; white-space: pre-wrap; word-break: break-all; max-height: 360px; overflow: auto; }
  .mono { font-family: var(--font-mono); }
</style>
