<script lang="ts">
  import { onDestroy } from "svelte";
  import { TOTPCode } from "../../../wailsjs/go/app/App.js";

  let input = "";
  let code: any = null;
  let error = "";
  let copied = false;
  let timer: any = null;

  // Recompute whenever the secret changes, then tick every second.
  $: start(input);

  function start(_: string) {
    if (timer) { clearInterval(timer); timer = null; }
    tick();
    if (input.trim()) timer = setInterval(tick, 1000);
  }

  async function tick() {
    const s = input.trim();
    if (!s) { code = null; error = ""; return; }
    try {
      code = await TOTPCode(s);
      error = "";
    } catch (e: any) {
      code = null;
      error = String(e?.message ?? e);
    }
  }

  async function copy() {
    if (!code) return;
    try { await navigator.clipboard.writeText(code.code); copied = true; setTimeout(() => (copied = false), 1200); } catch {}
  }

  const grouped = (c: string) => (c && c.length === 6 ? c.slice(0, 3) + " " + c.slice(3) : c);

  onDestroy(() => timer && clearInterval(timer));
</script>

<div class="tool">
  <h2>TOTP (2FA) code</h2>
  <p class="sub">
    Generate the current time-based 2FA code from a secret — <strong>locally</strong>. Web TOTP generators mean your 2FA
    seed is sent to someone else's server, which defeats the whole point; this never leaves your machine. Great for
    testing a 2FA setup. Nothing is stored.
  </p>

  <input class="field mono" bind:value={input} spellcheck="false" autocomplete="off"
    placeholder="base32 secret (JBSWY3DP…)  or  otpauth://totp/…" />

  {#if error}<div class="hint">{error}</div>{/if}

  {#if code}
    <div class="codecard">
      {#if code.issuer || code.account}
        <div class="who">{code.issuer}{#if code.issuer && code.account} · {/if}{code.account}</div>
      {/if}
      <button class="code mono" on:click={copy} title="Click to copy">{grouped(code.code)}</button>
      <div class="meta">
        <div class="bar"><div class="fill" style={`width:${(code.secondsLeft / code.period) * 100}%`}></div></div>
        <span class="left">{code.secondsLeft}s</span>
      </div>
      <div class="cphint">{copied ? "copied!" : "click the code to copy"}</div>
    </div>
  {/if}
</div>

<style>
  .tool { padding: 16px 20px 40px; max-width: 620px; }
  h2 { margin: 0 0 2px; font-size: 1.15rem; font-weight: 600; color: var(--text-strong); }
  .sub { margin: 0 0 14px; color: var(--text-dim); font-size: 0.86rem; line-height: 1.5; }
  .field { background: var(--well); border: 1px solid var(--field-border); border-radius: 8px; padding: 10px 12px; color: var(--text); font-size: 0.86rem; outline: none; width: 100%; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .hint { background: var(--warn-bg); border: 1px solid var(--warn-border); color: var(--warn-text); border-radius: 7px; padding: 9px 12px; font-size: 0.82rem; margin: 12px 0; }
  .codecard { margin-top: 18px; border: 1px solid var(--border); border-radius: 12px; padding: 20px; background: var(--card); text-align: center; }
  .who { font-size: 0.82rem; color: var(--text-dim); margin-bottom: 10px; }
  .code { background: none; border: none; cursor: pointer; font-size: 2.6rem; font-weight: 700; letter-spacing: 0.1em; color: var(--text-strong); padding: 0; line-height: 1; }
  .code:hover { color: var(--accent); }
  .meta { display: flex; align-items: center; gap: 10px; margin: 16px auto 0; max-width: 320px; }
  .bar { flex: 1; height: 6px; background: var(--well); border-radius: 4px; overflow: hidden; }
  .fill { height: 100%; background: var(--accent); border-radius: 4px; transition: width 0.9s linear; }
  .left { font-size: 0.78rem; color: var(--text-dim); font-family: var(--font-mono); min-width: 28px; text-align: right; }
  .cphint { font-size: 0.72rem; color: var(--text-faint, var(--text-dim)); margin-top: 10px; }
  .mono { font-family: var(--font-mono); }
</style>
