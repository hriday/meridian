<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import LiveTail from "../collectors/LiveTail.svelte";
  import { SyslogStart, SyslogStop, SyslogPoll } from "../../../wailsjs/go/app/App.js";

  const CAP = 10000;
  const POLL_MS = 700;
  const severities = [
    { label: "All severities", value: 7 },
    { label: "≥ warning", value: 4 },
    { label: "≥ error", value: 3 },
  ];

  let port = 5514;
  let running = false;
  let rows: any[] = []; // oldest→newest internally
  let lastSeq = 0;
  let epoch = 0;
  let total = 0;
  let dropped = 0;
  let filter = "";
  let sevFloor = 7;
  let err = "";
  let timer: any = null;
  let polling = false; // single-flight: a slow poll can't overlap the next tick

  function applyItems(p: any) {
    running = p.running;
    total = p.total;
    dropped = p.dropped;
    if (p.port) port = p.port;
    // Only take items strictly newer than our cursor — idempotent even if two
    // responses overlap, so rows can never hold duplicate seqs (keyed-each).
    const fresh = (p.items || []).filter((it: any) => it.seq > lastSeq);
    if (fresh.length) {
      rows = [...rows, ...fresh.map((it: any) => ({ seq: it.seq, ...it.val }))].slice(-CAP);
      lastSeq = fresh[fresh.length - 1].seq;
    }
  }

  async function poll() {
    if (polling) return;
    polling = true;
    try {
      const p = await SyslogPoll(lastSeq);
      if (epoch !== 0 && p.epoch !== epoch) {
        // a restart happened — reset and refetch the fresh buffer from 0
        epoch = p.epoch;
        rows = []; lastSeq = 0;
        applyItems(await SyslogPoll(0));
        return;
      }
      epoch = p.epoch;
      applyItems(p);
    } catch {
      /* transient; next tick retries */
    } finally {
      polling = false;
    }
  }

  async function start() {
    err = "";
    try {
      await SyslogStart(port);
    } catch (e) {
      err = String(e);
      running = false;
      return;
    }
    await poll();
  }

  async function stop() {
    try { await SyslogStop(); } catch {}
    running = false;
    rows = []; lastSeq = 0; total = 0; dropped = 0;
  }

  onMount(() => {
    poll(); // restore state if a collector is already running (survives tool switch)
    timer = setInterval(poll, POLL_MS);
  });
  onDestroy(() => { if (timer) clearInterval(timer); }); // leave the collector running

  const fmtTime = (t: string) => {
    try { return new Date(t).toLocaleTimeString("en-GB"); } catch { return ""; }
  };
  const sevClass = (s: number) => (s <= 3 ? "sev-bad" : s === 4 ? "sev-warn" : "sev-info");

  $: shown = rows
    .filter((r) => r.severity <= sevFloor)
    .filter((r) => {
      const f = filter.trim().toLowerCase();
      if (!f) return true;
      return (r.source + " " + (r.tag || "") + " " + r.message).toLowerCase().includes(f);
    })
    .slice()
    .reverse(); // newest first
  $: hostCount = new Set(rows.map((r) => r.source)).size;
</script>

<div class="tool">
  <p class="lede">
    Syslog receiver — listens for syslog from your devices and shows it live, newest first, so you can
    watch what a box is saying while you work on it. Filter by host, severity or text to cut to the one
    thing you care about. In-memory and ephemeral: it keeps the most recent messages while running and
    clears when you stop — a quick look on a laptop, not long-term retention.
  </p>
  <p class="help">
    Point a device's syslog at this machine on the <strong>Listen UDP</strong> port, then <strong>Start</strong>.
    The port defaults high (5514) so it binds without admin — set it to <span class="mono">514</span> only
    if you've launched Meridian elevated. While it runs it holds an open UDP port on your network; it's
    read-only, stores nothing, and stops when you hit Stop or close the app.
  </p>

  <LiveTail
    bind:port
    {running}
    {start}
    {stop}
    bind:filter
    {severities}
    bind:sevFloor
    {total}
    {dropped}
    hosts={hostCount}
    shownCount={shown.length}
    {err}
    note="Rolling buffer · last 10,000 messages · in-memory, cleared on stop · high port (no admin); set 514 if elevated"
  >
    {#each shown as r (r.seq)}
      <div class="lrow" class:m-bad={r.severity <= 3}>
        <span class="ts mono">{fmtTime(r.t)}</span>
        <span class="src mono">{r.source}</span>
        <span class="sevp {sevClass(r.severity)}">{r.severityName}</span>
        <span class="msg mono" title={r.message}>{r.tag ? r.tag + ": " : ""}{r.message}</span>
      </div>
    {/each}
  </LiveTail>
</div>

<style>
  .tool { padding: 14px 16px 20px; }
  .lede { max-width: 960px; margin: 0 0 12px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .help { max-width: 960px; margin: 0 0 14px; padding: 9px 13px; border-left: 2px solid var(--accent); background: var(--well); border-radius: 0 7px 7px 0; font-size: 0.78rem; color: var(--text-dim); line-height: 1.5; }
  .help strong { color: var(--text); font-weight: 600; }
  .mono { font-family: var(--mono, ui-monospace, monospace); }

  .lrow { display: grid; grid-template-columns: 74px 94px 76px 1fr; gap: 12px; align-items: baseline; padding: 5px 14px; border-top: 1px solid var(--border-soft); font-size: 0.78rem; }
  .lrow:first-child { border-top: none; }
  .ts { color: var(--text-faint); }
  .src { color: var(--text); }
  .sevp { font-size: 0.62rem; text-transform: uppercase; letter-spacing: .03em; padding: 1px 7px; border-radius: 4px; border: 1px solid var(--pill-border); text-align: center; }
  .msg { color: var(--text-dim); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .sev-bad { color: var(--bad, #ff6b6b); border-color: var(--bad, #ff6b6b); }
  .sev-warn { color: var(--warn, #d9a441); border-color: var(--warn, #d9a441); }
  .sev-info { color: var(--text-faint); }
  .m-bad .msg { color: var(--text); }
</style>
