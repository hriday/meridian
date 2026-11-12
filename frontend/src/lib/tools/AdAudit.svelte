<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import {
    ADAuditDetect, ADAuditParseText, ADAuditParseFile, ADAuditRun, ADAuditSave, ADAuditCheckPwned,
    WordlistCatalog, WordlistInstalled, WordlistDir, WordlistDownload, PickFile,
    RulesCatalog, RulesInstalled, RulesDir, RulesDownload,
  } from "../../../wailsjs/go/app/App.js";
  import { EventsOn, EventsOff } from "../../../wailsjs/runtime/runtime.js";

  let cracker: any = { available: false };
  let dir = "";
  let catalog: any[] = [];
  let installed: string[] = [];
  let selectedWordlist = "";
  let customWordlist = ""; // a wordlist you picked from disk
  let rulesPath = "";

  let pasteText = "";
  let dumpPath = "";
  let accounts: any[] = [];
  let parseErr = "";

  let running = false;
  let progress = { recovered: 0, total: 0, percent: 0 };
  let report: any = null;
  let runErr = "";
  let reveal = false;

  let savePath = "";
  let savePass = "";
  let saveMsg = "";

  // Have I Been Pwned (Pwned Passwords, NTLM, k-anonymity) — user-initiated.
  let pwned: Record<string, number> = {};
  let pwnedChecked = false, pwnedBusy = false, pwnedErr = "";
  async function checkPwned() {
    pwnedErr = ""; pwnedBusy = true;
    try { pwned = (await ADAuditCheckPwned()) || {}; pwnedChecked = true; }
    catch (e) { pwnedErr = String(e); }
    finally { pwnedBusy = false; }
  }
  function pwnedOf(r: any): number { return pwned[r.account?.ntHash] || 0; }
  $: pwnedRows = report
    ? report.results.filter((r: any) => pwnedOf(r) > 0).sort((a: any, b: any) => pwnedOf(b) - pwnedOf(a))
    : [];

  let downloading = "";
  let rulesCatalog: any[] = [], rulesInstalled: string[] = [], rulesDir = "", downloadingRule = "";

  async function refresh() {
    cracker = await ADAuditDetect();
    dir = await WordlistDir();
    catalog = await WordlistCatalog();
    installed = (await WordlistInstalled()) || [];
    if (!selectedWordlist && installed.length) selectedWordlist = installed[0];
    rulesDir = await RulesDir();
    rulesCatalog = await RulesCatalog();
    rulesInstalled = (await RulesInstalled()) || [];
  }

  async function downloadRule(name: string) {
    downloadingRule = name;
    try {
      const p = await RulesDownload(name);
      rulesInstalled = (await RulesInstalled()) || [];
      rulesPath = p; // use it for the next run
    } catch (e) { parseErr = String(e); }
    finally { downloadingRule = ""; }
  }
  function ruleInstalled(name: string): boolean { return rulesInstalled.includes(name); }
  function ruleUse(name: string) { rulesPath = rulesDir ? `${rulesDir}/${name}` : name; }

  onMount(() => {
    refresh();
    EventsOn("adaudit:progress", (p: any) => (progress = p));
  });
  onDestroy(() => EventsOff("adaudit:progress"));

  async function parsePaste() {
    parseErr = "";
    try {
      accounts = (await ADAuditParseText(pasteText)) || [];
    } catch (e) { parseErr = String(e); }
  }
  async function loadFile() {
    parseErr = "";
    try {
      accounts = (await ADAuditParseFile(dumpPath)) || [];
    } catch (e) { parseErr = String(e); }
  }

  async function download(name: string) {
    downloading = name;
    try {
      await WordlistDownload(name);
      installed = (await WordlistInstalled()) || [];
      selectedWordlist = name;
    } catch (e) { parseErr = String(e); }
    finally { downloading = ""; }
  }

  async function pickWordlist() {
    try {
      const p = await PickFile("Choose a wordlist");
      if (p) customWordlist = p;
    } catch (e) { parseErr = String(e); }
  }
  async function pickRules() {
    try {
      const p = await PickFile("Choose a hashcat rules file");
      if (p) rulesPath = p;
    } catch (e) { parseErr = String(e); }
  }

  $: humanAccounts = accounts.filter((a) => !a.machine).length;
  // The wordlist actually used: your picked file wins, else the selected installed one.
  $: wordlistPath = customWordlist || (selectedWordlist ? (dir ? `${dir}/${selectedWordlist}` : selectedWordlist) : "");

  async function run() {
    runErr = ""; report = null; pwned = {}; pwnedChecked = false; pwnedErr = "";
    if (!wordlistPath) { runErr = "Pick a wordlist first (download one, or Browse to your own)."; return; }
    running = true;
    progress = { recovered: 0, total: 0, percent: 0 };
    try {
      report = await ADAuditRun(accounts, wordlistPath, rulesPath ? [rulesPath] : []);
    } catch (e) { runErr = String(e); }
    finally { running = false; }
  }

  async function save() {
    saveMsg = "";
    try {
      await ADAuditSave(savePath, savePass);
      saveMsg = "Saved (encrypted).";
    } catch (e) { saveMsg = String(e); }
  }

  function pillClass(s: string) {
    return s === "weak" ? "bad" : s === "medium" ? "soon" : "good";
  }
</script>

<div class="tool">
  <div class="note">
    🔒 Authorized self-audit of <b>your own</b> directory. Import hashes you
    administer, find weak passwords, fix policy. Everything stays on this
    machine — nothing is uploaded; recovered passwords are only shown here and,
    if you choose, saved encrypted.
  </div>

  <!-- Cracker + wordlist status -->
  <div class="cards">
    <div class="card">
      <div class="k">Cracker</div>
      {#if cracker.available}
        <div><span class="pill good">hashcat</span> <span class="dim mono">{cracker.version}</span></div>
      {:else}
        <div><span class="pill bad">not found</span></div>
        <div class="cap">Install <span class="mono">hashcat</span> and ensure it's on your PATH, then reopen this tool.</div>
      {/if}
    </div>

    <div class="card">
      <div class="k">Wordlist</div>
      <select class="field sm" bind:value={selectedWordlist}>
        {#each installed as w}<option value={w}>{w}</option>{/each}
        {#if installed.length === 0}<option value="" disabled>none installed — download one below</option>{/if}
      </select>
      <div class="cap">Downloaded lists live in <span class="mono">{dir}</span>. Catalog entries are checksum-verified:</div>
      <div class="line wrap">
        {#each catalog as e}
          {#if !installed.includes(e.name)}
            <button class="chip" on:click={() => download(e.name)} disabled={downloading === e.name} title={e.desc}>
              {downloading === e.name ? "downloading…" : `↓ ${e.name}`}
            </button>
          {/if}
        {/each}
      </div>
      <div class="sub">…or use your own list (rockyou, a custom one)</div>
      <div class="line">
        <input class="field sm mono grow" bind:value={customWordlist} placeholder="/path/to/your/wordlist.txt" />
        <button class="go ghost" on:click={pickWordlist}>Browse…</button>
        {#if customWordlist}<button class="link" on:click={() => (customWordlist = "")}>clear</button>{/if}
      </div>
    </div>
  </div>

  <!-- Import -->
  <div class="card span">
    <div class="k">Import hashes</div>
    <div class="cap">Paste NT hashes (one per line, or <span class="mono">user:hash</span>) or load a secretsdump/pwdump file.</div>
    <textarea class="field mono ta" bind:value={pasteText} rows="4" placeholder="8846f7eaee8fb117ad06bdd830b7586c&#10;alice:31d6cfe0d16ae931b73c59d7e0c089c0"></textarea>
    <div class="line">
      <button class="go" on:click={parsePaste}>Parse pasted</button>
      <input class="field sm mono" bind:value={dumpPath} placeholder="/path/to/secretsdump.txt" />
      <button class="go ghost" on:click={loadFile}>Load file</button>
    </div>
    {#if parseErr}<div class="hint">{parseErr}</div>{/if}
    {#if accounts.length}<div class="cap">Loaded <b>{accounts.length}</b> accounts ({humanAccounts} user, {accounts.length - humanAccounts} machine).</div>{/if}
  </div>

  <!-- Run -->
  <div class="card span">
    <div class="k">Run audit</div>
    <div class="sub">Rules (optional)</div>
    <div class="line wrap">
      {#each rulesCatalog as r}
        {#if ruleInstalled(r.name)}
          <button class="chip" class:on={rulesPath.endsWith(r.name)} on:click={() => ruleUse(r.name)} title={r.desc}>✓ {r.name}{rulesPath.endsWith(r.name) ? " — in use" : ""}</button>
        {:else}
          <button class="chip" on:click={() => downloadRule(r.name)} disabled={downloadingRule === r.name} title={r.desc}>
            {downloadingRule === r.name ? "downloading…" : `↓ ${r.name}`}
          </button>
        {/if}
      {/each}
    </div>
    <div class="line">
      <input class="field sm mono grow" bind:value={rulesPath} placeholder="hashcat rules file — e.g. best64.rule" />
      <button class="go ghost" on:click={pickRules}>Browse…</button>
      {#if rulesPath}<button class="go ghost sm" on:click={() => (rulesPath = '')}>Clear</button>{/if}
    </div>
    <div class="cap">
      Rules mutate each word into many variations (add a year, capitalize, a→@ …), so <span class="mono">password</span>
      also tries <span class="mono">Password1</span>, <span class="mono">P@ssw0rd!</span>, etc. Download
      <span class="mono">OneRuleToRuleThemStill</span> above for a strong default (by <a href="https://github.com/stealthsploit/OneRuleToRuleThemStill" target="_blank" rel="noreferrer">stealthsploit</a>, checksum-verified), or Browse to hashcat's own <span class="mono">best64.rule</span>. Leave empty for a plain wordlist run.
    </div>
    <div class="line">
      <button class="go" on:click={run} disabled={running || !cracker.available || accounts.length === 0}>
        {running ? "Cracking…" : "Run audit"}
      </button>
    </div>
    {#if running || progress.total > 0}
      <div class="bar-wrap"><div class="bar" style="width:{progress.percent}%"></div></div>
      <div class="cap">Recovered {progress.recovered}/{progress.total} · {progress.percent.toFixed(0)}% of keyspace</div>
    {/if}
    {#if runErr}<div class="hint">{runErr}</div>{/if}
  </div>

  <!-- Results -->
  {#if report}
    <div class="verdict" class:bad={report.percentCracked > 20}>
      {report.cracked} of {report.total} passwords cracked ({report.percentCracked.toFixed(0)}%)
      {#if report.percentCracked > 20}— that's a lot of weak passwords.{:else}— not bad, keep an eye on the weak ones.{/if}
    </div>

    <div class="cards">
      <div class="card">
        <div class="k">Patterns</div>
        {#each report.patterns as p}<div class="line"><span class="pill soon">{p.pattern}</span> <span class="dim">× {p.count}</span></div>{/each}
        {#if (report.patterns || []).length === 0}<div class="dim">none</div>{/if}
      </div>
      <div class="card">
        <div class="k">Password reuse</div>
        {#each report.reuse as g}
          <div class="line"><span class="mono">{reveal ? g.plaintext : "••••••"}</span> <span class="dim">— {g.accounts.join(", ")}</span></div>
        {/each}
        {#if (report.reuse || []).length === 0}<div class="dim">no reuse found</div>{/if}
      </div>
    </div>

    <div class="card span">
      <div class="line between">
        <div class="k">Cracked accounts</div>
        <label class="reveal"><input type="checkbox" bind:checked={reveal} /> reveal passwords</label>
      </div>
      <table class="results">
        <tbody>
          {#each report.results.filter((r) => r.cracked) as r}
            <tr>
              <td class="mono">{r.account.name || "(unknown)"}</td>
              <td><span class="pill {pillClass(r.strength)}">{r.strength}</span></td>
              <td class="dim">{r.pattern}</td>
              <td class="mono pw">{reveal ? r.plaintext : "••••••"}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>

    <div class="card span">
      <div class="line between">
        <div class="k">Breached passwords (Have I Been Pwned)</div>
        <button class="go ghost sm" on:click={checkPwned} disabled={pwnedBusy}>{pwnedBusy ? "Checking…" : pwnedChecked ? "Re-check" : "Check against Have I Been Pwned"}</button>
      </div>
      <div class="cap">Checks every account's NT hash — <b>including ones that weren't cracked</b> — against the Pwned Passwords breach corpus. Only the first 5 characters of each hash leave your machine (k-anonymity); the password is never sent.</div>
      {#if pwnedErr}<div class="cap err">{pwnedErr}</div>{/if}
      {#if pwnedChecked}
        {#if pwnedRows.length}
          <div class="verdict bad" style="margin-top:8px">{pwnedRows.length} of {report.total} account{report.total === 1 ? "" : "s"} use a password found in known breaches — change these now.</div>
          <table class="results">
            <tbody>
              {#each pwnedRows as r}
                <tr>
                  <td class="mono">{r.account.name || "(unknown)"}</td>
                  <td>{#if r.cracked}<span class="pill {pillClass(r.strength)}">{r.strength}</span>{:else}<span class="pill soon">not cracked</span>{/if}</td>
                  <td class="mono pw">{reveal && r.cracked ? r.plaintext : "••••••"}</td>
                  <td class="dim" style="text-align:right;white-space:nowrap">seen <b style="color:var(--bad,#ff6b6b)">{pwnedOf(r).toLocaleString()}</b>×</td>
                </tr>
              {/each}
            </tbody>
          </table>
        {:else}
          <div class="verdict" style="margin-top:8px">None of the audited passwords appear in Pwned Passwords. Good — but that only means they're not in a known breach, not that they're strong.</div>
        {/if}
      {/if}
    </div>

    <div class="card span">
      <div class="k">Save (encrypted)</div>
      <div class="cap">The report contains plaintext passwords — saving encrypts it with a passphrase (no recovery if lost).</div>
      <div class="line">
        <input class="field sm mono" bind:value={savePath} placeholder="/path/to/audit.enc" />
        <input class="field sm" type="password" bind:value={savePass} placeholder="passphrase" />
        <button class="go ghost" on:click={save} disabled={!savePath || !savePass}>Save</button>
      </div>
      {#if saveMsg}<div class="cap">{saveMsg}</div>{/if}
    </div>
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 24px; }
  .note { background: var(--warn-bg); border: 1px solid var(--warn-border); color: var(--warn-text); border-radius: 9px; padding: 10px 13px; font-size: 0.82rem; line-height: 1.4; margin-bottom: 13px; max-width: 980px; }
  .cards { display: grid; grid-template-columns: 1fr 1fr; gap: 13px; margin-bottom: 13px; }
  @media (max-width: 760px) { .cards { grid-template-columns: 1fr; } }
  .card { background: var(--card); border: 1px solid var(--border-soft); border-radius: 10px; padding: 13px 15px; margin-bottom: 13px; }
  .card.span { grid-column: 1 / -1; }
  .k { font-size: 0.72rem; text-transform: uppercase; letter-spacing: .04em; color: var(--text-dim); margin-bottom: 8px; }
  .cap { font-size: 0.73rem; color: var(--text-dim); line-height: 1.45; margin-top: 7px; }
  .sub { font-size: 0.74rem; color: var(--text-dim); margin: 8px 0 4px; }
  .grow { flex: 1; min-width: 180px; }
  .link { background: none; border: none; color: var(--accent); cursor: pointer; font-size: 0.73rem; padding: 0 2px; }
  .line { display: flex; align-items: center; gap: 8px; margin: 7px 0; flex-wrap: wrap; }
  .line.wrap { flex-wrap: wrap; }
  .line.between { justify-content: space-between; }
  .dim { color: var(--text-faint); }
  .mono { font-family: var(--font-mono); }
  .field { background: var(--well); border: 1px solid var(--field-border); border-radius: 7px; padding: 8px 11px; color: var(--text); font-size: 0.86rem; outline: none; }
  .field.sm { padding: 6px 10px; font-size: 0.82rem; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .ta { width: 100%; resize: vertical; margin: 6px 0; box-sizing: border-box; }
  select.field { min-width: 240px; }
  .go { background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px; padding: 7px 14px; font: inherit; font-size: 0.85rem; cursor: pointer; white-space: nowrap; }
  .go.ghost { background: transparent; color: var(--accent); }
  .go.sm { padding: 4px 11px; font-size: 0.78rem; }
  .cap.err { color: var(--bad, #ff6b6b); }
  .go:disabled { opacity: .5; cursor: default; }
  .chip { font-family: var(--font-mono); font-size: 0.76rem; background: var(--well); color: var(--accent); border: 1px solid var(--field-border); border-radius: 6px; padding: 3px 9px; cursor: pointer; }
  .chip:disabled { opacity: .6; }
  .hint { background: var(--warn-bg); border: 1px solid var(--warn-border); color: var(--warn-text); border-radius: 7px; padding: 8px 11px; font-size: 0.82rem; margin-top: 8px; }
  .verdict { margin: 4px 0 13px; padding: 11px 14px; border-radius: 9px; font-size: 0.9rem; background: var(--good-bg, #102516); border: 1px solid var(--border-soft); color: var(--text); }
  .verdict.bad { background: var(--warn-bg); border-color: var(--warn-border); color: var(--warn-text); }
  .pill { display: inline-flex; font-size: 0.66rem; padding: 1px 8px; border-radius: 20px; border: 1px solid var(--border); color: var(--text-dim); }
  .pill.good { color: var(--good, #5ad17a); border-color: transparent; background: var(--good-bg, #102516); }
  .pill.bad { color: var(--bad, #ff6b6b); border-color: transparent; }
  .pill.soon { color: var(--text-faint); }
  .bar-wrap { height: 8px; background: var(--well); border-radius: 5px; overflow: hidden; margin: 8px 0 2px; }
  .bar { height: 100%; background: var(--accent); border-radius: 5px; transition: width .2s; }
  .results { width: 100%; border-collapse: collapse; margin-top: 6px; }
  .results td { padding: 5px 8px; border-bottom: 1px solid var(--border-soft); font-size: 0.82rem; vertical-align: middle; }
  .pw { color: var(--text); }
  .reveal { font-size: 0.76rem; color: var(--text-dim); display: flex; align-items: center; gap: 5px; cursor: pointer; }
</style>
