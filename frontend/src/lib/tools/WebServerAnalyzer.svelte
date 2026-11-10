<script lang="ts">
  import { AnalyzeWebServer } from "../../../wailsjs/go/app/App.js";

  let configText = "";
  let busy = false;
  let error = "";
  let a: any = null;
  let fileInput: HTMLInputElement;
  let copied = false;

  const cmd = "sudo nginx -T > nginx-config.txt";

  async function analyze() {
    const v = configText.trim();
    if (!v) { a = null; return; }
    error = "";
    a = null;
    busy = true;
    try {
      a = await AnalyzeWebServer(v, "nginx");
    } catch (e) {
      error = String(e);
    } finally {
      busy = false;
    }
  }

  function loadFile(e: Event) {
    const f = (e.target as HTMLInputElement).files?.[0];
    if (!f) return;
    const r = new FileReader();
    r.onload = () => { configText = String(r.result ?? ""); };
    r.readAsText(f);
  }

  function clearAll() { configText = ""; a = null; error = ""; }

  async function copyCmd() {
    try { await navigator.clipboard.writeText(cmd); copied = true; setTimeout(() => (copied = false), 1200); } catch {}
  }

  const sev = (s: string) => (s === "error" ? "bad" : s === "warning" ? "warn" : "info");
</script>

<div class="tool">
  <p class="lede">
    Analyze an <strong>nginx</strong> (and nginx Plus) config — offline. Shows what's served,
    what routes to which backend, and what looks wrong. It's structure + sanity checks, not a
    substitute for <span class="mono">nginx -t</span>. Nothing leaves your machine.
  </p>

  {#if !a}
    <div class="card span">
      <div class="k">Get the whole config</div>
      <p class="note">On the box, dump the full running config (nginx.conf + every include, resolved):</p>
      <div class="cmdrow">
        <code class="cmd mono">{cmd}</code>
        <button class="ghost" on:click={copyCmd}>{copied ? "Copied" : "Copy"}</button>
      </div>
      <p class="note dim">
        In a container: <span class="mono">docker exec &lt;container&gt; nginx -T</span> ·
        custom path: <span class="mono">nginx -T -c /path/to/nginx.conf</span> · then paste it below or Load file.
      </p>
    </div>
  {/if}

  <div class="card span">
    <textarea class="mono input" bind:value={configText}
      placeholder="Paste `nginx -T` output here, or use Load file…" spellcheck="false"></textarea>
    <div class="row">
      <button class="go" on:click={analyze} disabled={busy || !configText.trim()}>{busy ? "Analyzing…" : "Analyze"}</button>
      <button class="ghost" on:click={() => fileInput.click()}>Load file…</button>
      <button class="ghost" on:click={clearAll} disabled={!configText && !a}>Clear</button>
      <input type="file" bind:this={fileInput} on:change={loadFile} accept=".conf,.txt,.cfg,text/plain" hidden />
    </div>
  </div>

  {#if error}
    <div class="banner bad"><span class="vlabel">Couldn't parse it</span><span class="vnote">{error}</span></div>
  {/if}

  {#if a}
    <div class="tiles">
      <div class="tile"><div class="n">{a.counts.servers}</div><div class="l">Server blocks</div></div>
      <div class="tile"><div class="n">{a.counts.listeners}</div><div class="l">Listeners</div></div>
      <div class="tile"><div class="n">{a.counts.upstreams}</div><div class="l">Upstreams</div></div>
      <div class="tile"><div class="n">{a.counts.locations}</div><div class="l">Locations</div></div>
      <div class="tile" class:bad={a.counts.errors > 0}><div class="n">{a.counts.errors}</div><div class="l">Errors</div></div>
      <div class="tile" class:warn={a.counts.warnings > 0}><div class="n">{a.counts.warnings}</div><div class="l">Warnings</div></div>
    </div>

    {#if (a.findings ?? []).length}
      <div class="card span">
        <div class="k">Is it good — findings <span class="count">· {a.findings.length}</span></div>
        {#each a.findings as f}
          <div class="finding">
            <span class="dot {sev(f.severity)}"></span>
            <div class="f-main">
              <div class="f-title">{f.title}</div>
              <div class="f-where mono">{f.detail}{#if f.file} · {f.file}{#if f.line}:{f.line}{/if}{/if}</div>
            </div>
            <span class="f-tag {sev(f.severity)}">{f.category}</span>
          </div>
        {/each}
      </div>
    {/if}

    <div class="k sec">What's going on — sites</div>
    {#each a.servers ?? [] as s}
      <div class="card span vhost">
        <div class="vhost-head">
          <span class="vhost-name mono">{(s.names ?? []).join(" ") || "(no server_name)"}</span>
          {#each s.listeners ?? [] as l}
            <span class="badge" class:ssl={l.ssl} class:plain={!l.ssl}>{l.port}{l.ssl ? " ssl" : ""}</span>
            {#if l.http2}<span class="badge http2">http2</span>{/if}
          {/each}
          {#if s.file}<span class="vfile mono">{s.file}</span>{/if}
        </div>
        <div class="vbody">
          {#if s.root}<dt>root</dt><dd class="mono">{s.root}</dd>{/if}
          {#if (s.locations ?? []).length}
            <dt>routes</dt><dd>
              {#each s.locations as loc}
                <div class="route"><b class="mono">{loc.path}</b> <span class="arrow">→</span>
                  {#if loc.kind === "proxy"}proxy_pass <b class="mono">{loc.target}</b>
                  {:else if loc.kind === "redirect"}return {loc.target}
                  {:else if loc.kind === "alias"}alias {loc.target}
                  {:else}static{/if}
                  {#each loc.flags ?? [] as fl}<span class="warnflag"> · {fl}</span>{/each}
                </div>
              {/each}
            </dd>
          {/if}
          {#if (s.access ?? []).length}<dt>access</dt><dd class="mono">{s.access.join(" · ")}</dd>{/if}
          {#if (s.tlsProtocols ?? []).length || s.tlsCert}
            <dt>tls</dt><dd class="mono">{(s.tlsProtocols ?? []).join(" ")}{#if s.tlsCert} · {s.tlsCert}{/if}</dd>
          {/if}
        </div>
      </div>
    {/each}

    {#if (a.upstreams ?? []).length}
      <div class="k sec">What routes to what — upstreams</div>
      <div class="card span">
        <table class="mono"><thead>
          <tr><th>Upstream</th><th>Servers</th><th>Policy</th><th>Used by</th></tr>
        </thead><tbody>
          {#each a.upstreams as u}
            <tr>
              <td class="strong">{u.name}</td>
              <td class="dim">{(u.servers ?? []).join(" · ")}</td>
              <td class="dim">{u.policy || "—"}</td>
              <td class="dim">{(u.usedBy ?? []).join(", ") || "—"}</td>
            </tr>
          {/each}
        </tbody></table>
      </div>
    {/if}
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 20px; }
  .lede { max-width: 820px; margin: 0 0 14px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .mono { font-family: var(--mono, ui-monospace, monospace); }
  .input { width: 100%; min-height: 150px; resize: vertical; background: var(--well); border: 1px solid var(--field-border); border-radius: 7px; padding: 10px 12px; color: var(--text); font-size: 0.8rem; outline: none; line-height: 1.45; }
  .input:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .row { display: flex; gap: 8px; margin-top: 10px; align-items: center; }
  .go { background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px; padding: 8px 16px; font: inherit; font-size: 0.9rem; cursor: pointer; }
  .go:disabled { opacity: .6; cursor: default; }
  .ghost { background: transparent; border: 1px solid var(--border-soft); color: var(--text-dim); border-radius: 7px; padding: 7px 13px; font: inherit; font-size: 0.82rem; cursor: pointer; }
  .ghost:hover:not(:disabled) { color: var(--text); border-color: var(--field-border); }
  .ghost:disabled { opacity: .5; cursor: default; }

  .card { background: var(--card); border: 1px solid var(--border-soft); border-radius: 11px; padding: 13px 15px; max-width: 820px; margin-bottom: 13px; }
  .k { font-size: 0.74rem; text-transform: uppercase; letter-spacing: .04em; color: var(--text-faint); margin-bottom: 9px; }
  .k.sec { margin: 20px 0 10px; font-weight: 600; }
  .k .count { color: var(--text-dim); }
  .note { font-size: 0.8rem; color: var(--text-dim); margin: 0 0 8px; line-height: 1.5; }
  .note.dim { color: var(--text-faint); font-size: 0.75rem; margin-top: 10px; }
  .cmdrow { display: flex; gap: 8px; align-items: center; }
  .cmd { background: var(--well); border: 1px solid var(--border-soft); border-radius: 6px; padding: 7px 11px; font-size: 0.8rem; color: var(--text); flex: 1; }

  .tiles { display: grid; grid-template-columns: repeat(6,1fr); gap: 10px; margin: 14px 0; max-width: 820px; }
  @media (max-width: 720px){ .tiles{ grid-template-columns: repeat(3,1fr);} }
  .tile { background: var(--card); border: 1px solid var(--border-soft); border-radius: 10px; padding: 12px 13px; }
  .tile .n { font-size: 1.5rem; font-weight: 650; font-family: var(--mono, monospace); color: var(--text-strong); }
  .tile .l { font-size: 0.68rem; text-transform: uppercase; letter-spacing: .04em; color: var(--text-faint); margin-top: 3px; }
  .tile.bad .n { color: var(--bad, #ff6b6b); }
  .tile.warn .n { color: var(--warn, #d9a441); }

  .banner { margin: 14px 0; padding: 11px 14px; border-radius: 9px; border: 1px solid var(--bad, #ff6b6b); background: color-mix(in srgb, var(--bad, #ff6b6b) 10%, transparent); max-width: 820px; display: flex; flex-direction: column; gap: 3px; }
  .vlabel { font-weight: 600; color: var(--bad, #ff6b6b); }
  .vnote { font-size: 0.8rem; color: var(--text-dim); overflow-wrap: anywhere; }

  .finding { display: flex; gap: 11px; padding: 9px 0; border-bottom: 1px solid var(--border-soft); align-items: flex-start; }
  .finding:last-child { border-bottom: none; }
  .dot { width: 8px; height: 8px; border-radius: 50%; margin-top: 5px; flex: 0 0 auto; }
  .dot.bad { background: var(--bad, #ff6b6b); } .dot.warn { background: var(--warn, #d9a441); } .dot.info { background: var(--text-faint); }
  .f-main { flex: 1; }
  .f-title { font-size: 0.84rem; color: var(--text); }
  .f-where { font-size: 0.72rem; color: var(--text-faint); margin-top: 2px; overflow-wrap: anywhere; }
  .f-tag { font-size: 0.64rem; text-transform: uppercase; letter-spacing: .04em; padding: 1px 7px; border-radius: 4px; align-self: center; white-space: nowrap; }
  .f-tag.bad { color: var(--bad, #ff6b6b); background: color-mix(in srgb, var(--bad,#ff6b6b) 14%, transparent); }
  .f-tag.warn { color: var(--warn, #d9a441); background: color-mix(in srgb, var(--warn,#d9a441) 16%, transparent); }
  .f-tag.info { color: var(--text-dim); background: var(--well); }

  .vhost { padding: 0; overflow: hidden; }
  .vhost-head { display: flex; align-items: center; gap: 9px; padding: 11px 15px; border-bottom: 1px solid var(--border-soft); flex-wrap: wrap; }
  .vhost-name { font-weight: 650; font-size: 0.88rem; color: var(--text-strong); }
  .badge { font-size: 0.68rem; padding: 1px 7px; border-radius: 5px; border: 1px solid var(--border-soft); color: var(--text-dim); font-family: var(--mono, monospace); }
  .badge.ssl { color: var(--good, #5ad17a); border-color: color-mix(in srgb, var(--good,#5ad17a) 40%, transparent); }
  .badge.plain { color: var(--warn, #d9a441); }
  .badge.http2 { color: var(--accent); }
  .vfile { margin-left: auto; color: var(--text-faint); font-size: 0.72rem; }
  .vbody { padding: 10px 15px; display: grid; grid-template-columns: max-content 1fr; gap: 4px 14px; font-size: 0.8rem; }
  .vbody dt { color: var(--text-faint); }
  .vbody dd { margin: 0; color: var(--text); overflow-wrap: anywhere; }
  .route { color: var(--text-dim); padding: 1px 0; } .route b { color: var(--text); } .arrow { color: var(--accent); }
  .warnflag { color: var(--warn, #d9a441); }

  table { width: 100%; border-collapse: collapse; font-size: 0.8rem; }
  th { text-align: left; color: var(--text-faint); font-weight: 500; padding: 3px 10px 7px 0; border-bottom: 1px solid var(--border-soft); }
  td { padding: 6px 10px 6px 0; border-bottom: 1px solid var(--border-soft); vertical-align: top; overflow-wrap: anywhere; }
  tr:last-child td { border-bottom: none; }
  .strong { color: var(--text-strong); font-weight: 600; }
  .dim { color: var(--text-dim); }
</style>
