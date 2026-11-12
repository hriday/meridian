<script lang="ts">
  import { IdentifyHash } from "../../../wailsjs/go/app/App.js";

  let hash = "";
  let cands: any[] = [];
  let done = false;
  async function run() {
    const h = hash.trim();
    if (!h) { cands = []; done = false; return; }
    cands = (await IdentifyHash(h)) || [];
    done = true;
  }
  $: hash, (done = false); // reset when the input changes
</script>

<div class="tool">
  <p class="lede">
    Paste a password hash and get its <strong>likely type and the hashcat mode</strong> to crack it. It's a guess from
    the format (prefix, length, character set) — several hashes share a shape — so you get ranked candidates.
    Nothing leaves your machine.
  </p>

  <div class="row">
    <input class="field mono" bind:value={hash} placeholder="e.g. 32ed87bdb5fdc5e9cba88547376818d4  or  $6$salt$…" spellcheck="false" autocomplete="off"
      on:keydown={(e) => e.key === "Enter" && run()} />
    <button class="btn" on:click={run} disabled={!hash.trim()}>Identify</button>
  </div>

  {#if done}
    {#if cands.length}
      <div class="card">
        {#each cands as c, i}
          <div class="crow" class:top={i === 0}>
            <span class="name">{c.name}</span>
            {#if c.mode >= 0}
              <span class="mode mono">hashcat -m {c.mode}</span>
            {:else}
              <span class="mode dim">no hashcat mode</span>
            {/if}
            {#if c.note}<span class="note">{c.note}</span>{/if}
          </div>
        {/each}
      </div>
      <p class="foot">Then crack it in the <strong>AD Password Audit</strong> (for NTLM/-m 1000) or with hashcat directly: <span class="mono">hashcat -m &lt;mode&gt; hash.txt wordlist.txt -r OneRuleToRuleThemStill.rule</span></p>
    {/if}
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 30px; }
  .lede { max-width: 820px; margin: 0 0 14px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .mono { font-family: var(--font-mono, ui-monospace, monospace); }
  .dim { color: var(--text-dim); }
  .row { display: flex; gap: 10px; max-width: 760px; margin-bottom: 14px; }
  .field { background: var(--well); border: 1px solid var(--field-border, var(--border)); border-radius: 8px; padding: 9px 12px; color: var(--text); font-size: 0.86rem; outline: none; flex: 1 1 auto; min-width: 0; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .btn { background: var(--accent); color: var(--on-accent, #052e2b); border: 1px solid transparent; border-radius: 8px; padding: 9px 16px; font: inherit; font-size: 0.86rem; font-weight: 550; cursor: pointer; white-space: nowrap; }
  .btn:disabled { opacity: .5; cursor: default; }
  .card { background: var(--card); border: 1px solid var(--border); border-radius: 11px; max-width: 820px; overflow: hidden; }
  .crow { display: flex; align-items: center; gap: 12px; padding: 10px 14px; border-bottom: 1px solid var(--border-soft, var(--border)); flex-wrap: wrap; }
  .crow:last-child { border-bottom: 0; }
  .crow.top { background: color-mix(in srgb, var(--accent) 7%, transparent); }
  .name { font-weight: 600; color: var(--text); font-size: 0.86rem; }
  .mode { font-size: 0.78rem; color: var(--accent); background: var(--well); border-radius: 6px; padding: 2px 9px; }
  .mode.dim { color: var(--text-dim); background: none; }
  .note { font-size: 0.76rem; color: var(--text-dim); margin-left: auto; }
  .foot { max-width: 820px; font-size: 0.76rem; color: var(--text-faint, var(--text-dim)); margin: 12px 0 0; line-height: 1.6; }
</style>
