<script lang="ts">
  import { PwHashGenerate, PwHashCheck } from "../../../wailsjs/go/app/App.js";
  import ToolHelp from "../ToolHelp.svelte";

  let mode: "gen" | "verify" = "gen";

  // generate
  let password = "";
  let user = "";
  let cost = 10;
  let genRes: any = null;
  let genErr = "";

  // verify
  let vpass = "";
  let vhash = "";
  let vRes: any = null;
  let vErr = "";

  let copied = "";

  async function generate() {
    genErr = ""; genRes = null;
    try {
      genRes = await PwHashGenerate(password, user.trim(), cost);
    } catch (e: any) { genErr = String(e?.message ?? e); }
  }
  async function verify() {
    vErr = ""; vRes = null;
    try {
      vRes = await PwHashCheck(vpass, vhash.trim());
    } catch (e: any) { vErr = String(e?.message ?? e); }
  }
  async function copy(text: string, what: string) {
    try { await navigator.clipboard.writeText(text); copied = what; setTimeout(() => (copied = ""), 1200); } catch {}
  }
</script>

<div class="tool">
  <h2>bcrypt / htpasswd</h2>
  <p class="sub">
    Make a bcrypt password hash (for nginx/Apache basic-auth, an <span class="mono">htpasswd</span> file, app config) or
    verify a password against one — <strong>entirely locally</strong>. Web bcrypt generators send your password to their
    server; this never does. Nothing is stored.
  </p>

  <ToolHelp>
    <strong>Generate</strong> gives you the <code>bcrypt</code> hash and, with a username, the full
    <code>htpasswd</code> line to paste into your auth file. A higher <strong>cost</strong> is slower to compute and
    slower to crack — <code>10</code>–<code>12</code> is a sensible range. <strong>Verify</strong> shows a green check
    when a password matches a hash and a red cross when it doesn't; use it to confirm a credential before you rely on it.
  </ToolHelp>

  <div class="tabs">
    <button class="tab" class:on={mode === "gen"} on:click={() => (mode = "gen")}>Generate</button>
    <button class="tab" class:on={mode === "verify"} on:click={() => (mode = "verify")}>Verify</button>
  </div>

  {#if mode === "gen"}
    <div class="form">
      <label>Password<input class="field mono" type="password" bind:value={password} spellcheck="false" autocomplete="off" /></label>
      <div class="row2">
        <label>Username <span class="opt">(optional, for the htpasswd line)</span><input class="field mono" bind:value={user} spellcheck="false" autocomplete="off" placeholder="admin" /></label>
        <label class="costlbl">Cost <span class="opt">({cost})</span><input class="range" type="range" min="4" max="15" bind:value={cost} /></label>
      </div>
      <button class="go" on:click={generate} disabled={!password}>Generate</button>
    </div>
    {#if genErr}<div class="hint">{genErr}</div>{/if}
    {#if genRes}
      <div class="out">
        <div class="ol"><span class="lbl">bcrypt hash</span><button class="cp" on:click={() => copy(genRes.hash, "hash")}>{copied === "hash" ? "copied" : "copy"}</button></div>
        <pre class="box mono">{genRes.hash}</pre>
        {#if genRes.htpasswd}
          <div class="ol"><span class="lbl">htpasswd line</span><button class="cp" on:click={() => copy(genRes.htpasswd, "line")}>{copied === "line" ? "copied" : "copy"}</button></div>
          <pre class="box mono">{genRes.htpasswd}</pre>
        {/if}
      </div>
    {/if}
  {:else}
    <div class="form">
      <label>Password<input class="field mono" type="password" bind:value={vpass} spellcheck="false" autocomplete="off" /></label>
      <label>bcrypt hash (or <span class="mono">user:hash</span> line)<input class="field mono" bind:value={vhash} spellcheck="false" autocomplete="off" placeholder="$2y$10$…" /></label>
      <button class="go" on:click={verify} disabled={!vpass || !vhash.trim()}>Verify</button>
    </div>
    {#if vErr}<div class="hint">{vErr}</div>{/if}
    {#if vRes}
      <div class="vres {vRes.match ? 'ok' : 'bad'}">
        <span class="vmark">{vRes.match ? "✓" : "✗"}</span>
        <span>{vRes.match ? "Password matches the hash" : "Password does NOT match"}{#if vRes.cost} <span class="dim">(cost {vRes.cost})</span>{/if}</span>
      </div>
    {/if}
  {/if}
</div>

<style>
  .tool { padding: 16px 20px 40px; max-width: 760px; }
  h2 { margin: 0 0 2px; font-size: 1.15rem; font-weight: 600; color: var(--text-strong); }
  .sub { margin: 0 0 14px; color: var(--text-dim); font-size: 0.86rem; line-height: 1.5; }
  .tabs { display: flex; gap: 6px; margin-bottom: 14px; }
  .tab { background: var(--well); border: 1px solid var(--border); border-radius: 7px; padding: 6px 14px; font: inherit; font-size: 0.84rem; color: var(--text-dim); cursor: pointer; }
  .tab.on { background: var(--accent); border-color: var(--accent); color: var(--on-accent, #052e2b); }
  .form { display: flex; flex-direction: column; gap: 11px; max-width: 560px; }
  label { display: flex; flex-direction: column; gap: 4px; font-size: 0.8rem; color: var(--text-dim); }
  .opt { color: var(--text-faint, var(--text-dim)); }
  .field { background: var(--well); border: 1px solid var(--field-border); border-radius: 7px; padding: 8px 11px; color: var(--text); font-size: 0.88rem; outline: none; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .row2 { display: grid; grid-template-columns: 1fr max-content; gap: 11px; }
  .costlbl { min-width: 160px; }
  .range { accent-color: var(--accent); }
  .go { background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px; padding: 8px 18px; font: inherit; font-size: 0.86rem; cursor: pointer; align-self: flex-start; }
  .go:disabled { opacity: .5; cursor: default; }
  .hint { background: var(--warn-bg); border: 1px solid var(--warn-border); color: var(--warn-text); border-radius: 7px; padding: 9px 12px; font-size: 0.82rem; margin: 12px 0; }
  .out { margin-top: 16px; }
  .ol { display: flex; align-items: center; justify-content: space-between; margin: 10px 0 4px; }
  .lbl { font-size: 0.68rem; text-transform: uppercase; letter-spacing: .04em; color: var(--text-faint, var(--text-dim)); font-weight: 650; }
  .cp { background: none; border: 1px solid var(--border); border-radius: 5px; padding: 1px 9px; font-size: 0.7rem; color: var(--text-dim); cursor: pointer; }
  .box { background: var(--well); border: 1px solid var(--border); border-radius: 8px; padding: 10px 12px; font-size: 0.8rem; color: var(--text); margin: 0; overflow-x: auto; white-space: pre-wrap; word-break: break-all; }
  .vres { display: flex; align-items: center; gap: 9px; margin-top: 16px; padding: 11px 14px; border-radius: 8px; font-size: 0.88rem; border: 1px solid var(--border); border-left-width: 3px; color: var(--text); }
  .vres.ok { border-left-color: var(--good, #3fb950); }
  .vres.bad { border-left-color: var(--danger, #e5484d); }
  .vmark { font-size: 1.15rem; font-weight: 700; }
  .vres.ok .vmark { color: var(--good, #3fb950); } .vres.bad .vmark { color: var(--danger, #e5484d); }
  .dim { color: var(--text-dim); }
  .mono { font-family: var(--font-mono); }
</style>
