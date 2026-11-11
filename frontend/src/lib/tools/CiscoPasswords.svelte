<script lang="ts">
  import { AnalyzeCiscoPasswords } from "../../../wailsjs/go/app/App.js";

  let cfg = "";
  let secrets: any[] = [];
  let analyzed = false;
  let err = "";

  async function analyze() {
    err = "";
    analyzed = false;
    try {
      secrets = (await AnalyzeCiscoPasswords(cfg)) ?? [];
      analyzed = true;
    } catch (e) {
      err = String(e);
    }
  }
  function clear() {
    cfg = "";
    secrets = [];
    analyzed = false;
    err = "";
  }
  async function loadFile(ev: Event) {
    const f = (ev.target as HTMLInputElement).files?.[0];
    if (!f) return;
    cfg = await f.text();
    (ev.target as HTMLInputElement).value = "";
  }

  const verdictClass = (v: string) =>
    v === "cleartext" || v === "reversible"
      ? "bad"
      : v === "strong-hash"
        ? "good"
        : "warn";
  const verdictLabel: Record<string, string> = {
    cleartext: "cleartext",
    reversible: "reversible (type 7)",
    "reversible-with-key": "reversible with key",
    "weak-hash": "weak hash",
    "deprecated-hash": "deprecated hash",
    "strong-hash": "strong hash",
    unknown: "unknown",
  };

  $: exposed = secrets.filter((s) => s.decoded).length;
</script>

<div class="tool">
  <p class="lede">
    Paste a Cisco IOS/ASA config. Meridian recovers the <strong>reversible</strong>
    passwords (type 7 is trivially decodable; type 0 is cleartext) and rates the
    one-way hashes (type 5 MD5 is weak; type 8/9 are strong). Everything runs
    locally — the config never leaves your machine. Recovered passwords are
    sensitive; clear the view when you're done.
  </p>

  <div class="card span">
    <div class="colhdr">
      <span class="k0">Cisco config</span>
      <label class="file">Load…<input type="file" on:change={loadFile} /></label>
    </div>
    <textarea class="field mono" bind:value={cfg} placeholder="paste `show running-config` (or any snippet with password/secret lines)…" spellcheck="false"></textarea>
    <div class="row">
      <button type="button" class="go" on:click={analyze} disabled={!cfg.trim()}>Decode</button>
      <button type="button" class="ghost" on:click={clear}>Clear</button>
    </div>
  </div>

  {#if err}
    <div class="banner bad"><span class="vlabel">Couldn't analyze</span><span class="vnote">{err}</span></div>
  {/if}

  {#if analyzed}
    {#if secrets.length === 0}
      <div class="nochg">No password or secret lines found.</div>
    {:else}
      <div class="summary">
        <span class="count mono">{secrets.length}</span> secret{secrets.length === 1 ? "" : "s"} ·
        <span class="exp mono">{exposed}</span> recovered in plaintext
      </div>
      <div class="card span">
        <div class="tablewrap">
          <table>
            <thead><tr><th>Where</th><th>Type</th><th>Strength</th><th>Recovered password</th></tr></thead>
            <tbody>
              {#each secrets as s}
                <tr>
                  <td class="mono ctx">{s.context} <span class="kw">{s.keyword}</span></td>
                  <td class="mono ty">type {s.type}</td>
                  <td><span class="badge {verdictClass(s.verdict)}">{verdictLabel[s.verdict] || s.verdict}</span></td>
                  <td class="pw">
                    {#if s.decoded}<span class="mono plain">{s.plaintext}</span>{:else}<span class="hash">— not reversible</span>{/if}
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </div>
    {/if}
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 20px; }
  .lede { max-width: 820px; margin: 0 0 14px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .mono { font-family: var(--mono, ui-monospace, monospace); }
  .card { background: var(--card); border: 1px solid var(--border-soft); border-radius: 11px; padding: 13px 15px; max-width: 900px; margin-bottom: 14px; }
  .colhdr { display: flex; align-items: center; justify-content: space-between; margin-bottom: 6px; }
  .k0 { font-size: 0.68rem; text-transform: uppercase; letter-spacing: .05em; color: var(--text-faint); }
  .file { font-size: 0.72rem; color: var(--accent); cursor: pointer; }
  .file input { display: none; }
  textarea.field { background: var(--well); border: 1px solid var(--field-border); border-radius: 8px; padding: 9px 11px; color: var(--text); font-size: 0.8rem; outline: none; width: 100%; min-height: 160px; resize: vertical; line-height: 1.45; }
  textarea.field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .row { display: flex; gap: 10px; margin-top: 10px; align-items: center; }
  .go { background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px; padding: 0 16px; height: 36px; font: inherit; font-size: 0.88rem; cursor: pointer; }
  .go:disabled { opacity: .55; cursor: default; }
  .ghost { background: transparent; border: 1px solid var(--border-soft); color: var(--text-dim); border-radius: 7px; padding: 0 14px; height: 36px; font: inherit; font-size: 0.84rem; cursor: pointer; }

  .banner { margin: 0 0 14px; padding: 11px 14px; border-radius: 9px; border: 1px solid var(--bad, #ff6b6b); background: color-mix(in srgb, var(--bad,#ff6b6b) 10%, transparent); max-width: 820px; display: flex; flex-direction: column; gap: 3px; }
  .vlabel { font-weight: 600; color: var(--bad, #ff6b6b); }
  .vnote { font-size: 0.8rem; color: var(--text-dim); overflow-wrap: anywhere; }

  .summary { margin: 2px 0 10px; font-size: 0.84rem; color: var(--text-dim); }
  .summary .count { color: var(--text-strong); font-weight: 650; }
  .summary .exp { color: var(--bad, #f0616a); font-weight: 650; }
  .nochg { color: var(--text-dim); font-size: 0.85rem; padding: 8px 0; }

  .tablewrap { overflow-x: auto; }
  table { width: 100%; border-collapse: collapse; font-size: 0.82rem; min-width: 560px; }
  thead th { text-align: left; font-size: 0.66rem; text-transform: uppercase; letter-spacing: .05em; color: var(--text-faint); font-weight: 600; padding: 8px 12px; border-bottom: 1px solid var(--border); }
  tbody td { padding: 8px 12px; border-bottom: 1px solid var(--border-soft); vertical-align: middle; }
  tbody tr:last-child td { border-bottom: 0; }
  .ctx { color: var(--text); } .ctx .kw { color: var(--text-faint); }
  .ty { color: var(--text-dim); white-space: nowrap; }
  .badge { display: inline-block; font-size: 0.68rem; font-weight: 600; border-radius: 999px; padding: 2px 9px; white-space: nowrap; }
  .badge.bad { color: var(--bad, #f0616a); background: color-mix(in srgb, var(--bad,#f0616a) 14%, transparent); }
  .badge.warn { color: var(--warn, #d6a531); background: color-mix(in srgb, var(--warn,#d6a531) 15%, transparent); }
  .badge.good { color: var(--good, #3fb950); background: color-mix(in srgb, var(--good,#3fb950) 14%, transparent); }
  .pw .plain { color: var(--bad, #f0616a); font-weight: 600; overflow-wrap: anywhere; }
  .pw .hash { color: var(--text-faint); font-size: 0.78rem; }
</style>
