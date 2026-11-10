<script lang="ts">
  import { onDestroy } from "svelte";
  import { NetcatConnect, NetcatListen, NetcatSend, NetcatClose } from "../../../wailsjs/go/app/App.js";
  import { EventsOn, EventsOff } from "../../../wailsjs/runtime/runtime.js";

  let host = "127.0.0.1";
  let port = "9000";
  let proto: "tcp" | "udp" = "tcp";
  let useTLS = false;
  let insecure = false;

  let state = "idle"; // idle | connecting | connected | listening | closed | error
  let peer = "";
  let errMsg = "";
  let cert: any = null;

  type Line = { dir: "recv" | "sent"; text: string; hex: string };
  let lines: Line[] = [];
  let view: "text" | "hex" = "text";
  let bytesRecv = 0;
  let bytesSent = 0;

  let sendText = "";
  let ending: "none" | "lf" | "crlf" = "lf";

  const ENDINGS = { none: "", lf: "\n", crlf: "\r\n" };

  EventsOn("netcat:recv", (d: any) => {
    lines = [...lines, { dir: "recv", text: d.text, hex: d.hex }];
    bytesRecv += d.n;
  });
  EventsOn("netcat:status", (d: any) => {
    state = d.state;
    if (d.peer) peer = d.peer;
    errMsg = d.err || "";
  });
  onDestroy(() => {
    EventsOff("netcat:recv");
    EventsOff("netcat:status");
    NetcatClose();
  });

  function reset() {
    lines = []; bytesRecv = 0; bytesSent = 0; cert = null; errMsg = "";
  }

  async function connect() {
    reset();
    state = "connecting";
    try {
      cert = await NetcatConnect(proto, `${host}:${port}`, proto === "tcp" && useTLS, insecure);
      if (!cert || !cert.subject) cert = null;
      state = "connected";
    } catch (e) { state = "error"; errMsg = String(e); }
  }

  async function listen() {
    reset();
    try {
      const bound = await NetcatListen(proto, `${host}:${port}`);
      state = "listening"; peer = bound;
    } catch (e) { state = "error"; errMsg = String(e); }
  }

  async function send() {
    const data = sendText + ENDINGS[ending];
    if (!data) return;
    try {
      await NetcatSend(data);
      lines = [...lines, { dir: "sent", text: data, hex: "" }];
      bytesSent += data.length;
      sendText = "";
    } catch (e) { errMsg = String(e); }
  }

  function close() {
    NetcatClose();
    state = "closed";
  }

  $: live = state === "connected" || state === "listening";
</script>

<div class="tool">
  <!-- Target -->
  <div class="card">
    <div class="line">
      <input class="field mono" bind:value={host} placeholder="host" aria-label="host" />
      <input class="field mono port" bind:value={port} placeholder="port" aria-label="port" />
      <select class="field sm" bind:value={proto}>
        <option value="tcp">TCP</option>
        <option value="udp">UDP</option>
      </select>
      {#if proto === "tcp"}
        <label class="chk"><input type="checkbox" bind:checked={useTLS} /> TLS</label>
        {#if useTLS}<label class="chk"><input type="checkbox" bind:checked={insecure} /> allow unverified</label>{/if}
      {/if}
    </div>
    <div class="line">
      {#if !live}
        <button class="go" on:click={connect}>Connect</button>
        <button class="go ghost" on:click={listen}>Listen</button>
      {:else}
        <button class="go ghost" on:click={close}>Close</button>
      {/if}
      <span class="status {state}">{state}{#if peer} · {peer}{/if}</span>
    </div>
    {#if errMsg}<div class="hint">{errMsg}</div>{/if}
    {#if cert}
      <div class="cap">
        <b>TLS</b> · subject {cert.subject} · issuer {cert.issuer} · expires {cert.notAfter}
        {#if cert.verified}<span class="pill good">verified</span>{:else}<span class="pill bad">unverified</span>{/if}
      </div>
    {/if}
  </div>

  <!-- Transcript -->
  <div class="card span">
    <div class="line between">
      <div class="k">Transcript</div>
      <div class="line">
        <span class="dim small">↓ {bytesRecv}b · ↑ {bytesSent}b</span>
        <div class="toggle">
          <button class:on={view === "text"} on:click={() => (view = "text")}>text</button>
          <button class:on={view === "hex"} on:click={() => (view = "hex")}>hex</button>
        </div>
      </div>
    </div>
    <div class="transcript mono">
      {#each lines as l}
        <div class="row {l.dir}">
          <span class="arrow">{l.dir === "recv" ? "←" : "→"}</span><!--
       --><span class="data">{view === "hex" ? (l.hex || l.text) : l.text}</span>
        </div>
      {/each}
      {#if lines.length === 0}<div class="dim">no data yet</div>{/if}
    </div>
  </div>

  <!-- Send -->
  <div class="card span">
    <div class="k">Send</div>
    <div class="line">
      <input class="field mono grow" bind:value={sendText} placeholder="data to send" disabled={!live}
        on:keydown={(e) => e.key === "Enter" && send()} />
      <select class="field sm" bind:value={ending} aria-label="line ending">
        <option value="none">no newline</option>
        <option value="lf">LF</option>
        <option value="crlf">CRLF</option>
      </select>
      <button class="go" on:click={send} disabled={!live}>Send</button>
    </div>
  </div>
</div>

<style>
  .tool { padding: 14px 16px 24px; }
  .card { background: var(--card); border: 1px solid var(--border-soft); border-radius: 10px; padding: 13px 15px; margin-bottom: 13px; }
  .card.span { grid-column: 1 / -1; }
  .k { font-size: 0.72rem; text-transform: uppercase; letter-spacing: .04em; color: var(--text-dim); }
  .line { display: flex; align-items: center; gap: 8px; margin: 6px 0; flex-wrap: wrap; }
  .line.between { justify-content: space-between; }
  .field { background: var(--well); border: 1px solid var(--field-border); border-radius: 7px; padding: 8px 11px; color: var(--text); font-size: 0.86rem; outline: none; }
  .field.sm { padding: 6px 9px; font-size: 0.82rem; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .port { width: 90px; }
  .grow { flex: 1; min-width: 160px; }
  .mono { font-family: var(--font-mono); }
  .go { background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px; padding: 7px 14px; font: inherit; font-size: 0.85rem; cursor: pointer; }
  .go.ghost { background: transparent; color: var(--accent); }
  .go:disabled { opacity: .5; cursor: default; }
  .chk { font-size: 0.8rem; color: var(--text-dim); display: flex; align-items: center; gap: 4px; }
  .status { font-size: 0.78rem; color: var(--text-dim); }
  .status.connected, .status.listening { color: var(--good, #5ad17a); }
  .status.error { color: var(--bad, #ff6b6b); }
  .cap { font-size: 0.74rem; color: var(--text-dim); margin-top: 7px; display: flex; gap: 6px; align-items: center; flex-wrap: wrap; }
  .dim { color: var(--text-faint); }
  .small { font-size: 0.72rem; }
  .hint { background: var(--warn-bg); border: 1px solid var(--warn-border); color: var(--warn-text); border-radius: 7px; padding: 8px 11px; font-size: 0.82rem; margin-top: 8px; }
  .pill { font-size: 0.64rem; padding: 1px 7px; border-radius: 20px; }
  .pill.good { color: var(--good, #5ad17a); background: var(--good-bg, #102516); }
  .pill.bad { color: var(--bad, #ff6b6b); border: 1px solid var(--bad, #ff6b6b); }
  .toggle { display: inline-flex; border: 1px solid var(--field-border); border-radius: 7px; overflow: hidden; }
  .toggle button { background: var(--well); color: var(--text-dim); border: none; padding: 4px 11px; font-size: 0.76rem; cursor: pointer; }
  .toggle button.on { background: var(--accent); color: var(--on-accent); }
  .transcript { background: var(--well); border: 1px solid var(--border-soft); border-radius: 8px; padding: 10px; margin-top: 6px; max-height: 320px; overflow-y: auto; font-size: 0.8rem; white-space: pre-wrap; word-break: break-all; }
  .row { display: flex; gap: 7px; padding: 1px 0; }
  .row.sent .data { color: var(--accent); }
  .row.recv .data { color: var(--text); }
  .arrow { color: var(--text-faint); user-select: none; }
  .data { flex: 1; }
</style>
