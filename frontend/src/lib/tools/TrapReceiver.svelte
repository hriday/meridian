<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import LiveTail from "../collectors/LiveTail.svelte";
  import { TrapStart, TrapStop, TrapPoll } from "../../../wailsjs/go/app/App.js";

  const CAP = 10000;
  const POLL_MS = 700;

  let port = 1162;
  let running = false;
  let rows: any[] = []; // oldest→newest
  let lastSeq = 0;
  let epoch = 0;
  let total = 0;
  let dropped = 0;
  let filter = "";
  let err = "";
  let timer: any = null;
  let polling = false;

  function applyItems(p: any) {
    running = p.running;
    total = p.total;
    dropped = p.dropped;
    if (p.port) port = p.port;
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
      const p = await TrapPoll(lastSeq);
      if (epoch !== 0 && p.epoch !== epoch) {
        epoch = p.epoch;
        rows = []; lastSeq = 0;
        applyItems(await TrapPoll(0));
        return;
      }
      epoch = p.epoch;
      applyItems(p);
    } catch {
      /* transient */
    } finally {
      polling = false;
    }
  }

  async function start() {
    err = "";
    try {
      await TrapStart(port);
    } catch (e) {
      err = String(e);
      running = false;
      return;
    }
    await poll();
  }

  async function stop() {
    try { await TrapStop(); } catch {}
    running = false;
    rows = []; lastSeq = 0; total = 0; dropped = 0;
  }

  onMount(() => { poll(); timer = setInterval(poll, POLL_MS); });
  onDestroy(() => { if (timer) clearInterval(timer); });

  const fmtTime = (t: string) => {
    try { return new Date(t).toLocaleTimeString("en-GB"); } catch { return ""; }
  };
  const vbText = (r: any) => (r.varbinds || []).map((v: any) => `${v.oid}=${v.value}`).join("  ·  ");
  const trapLabel = (r: any) => r.trapName || r.trapOid || "(trap)";

  $: shown = rows
    .filter((r) => {
      const f = filter.trim().toLowerCase();
      if (!f) return true;
      return (r.source + " " + trapLabel(r) + " " + (r.trapOid || "") + " " + vbText(r)).toLowerCase().includes(f);
    })
    .slice()
    .reverse();
  $: hostCount = new Set(rows.map((r) => r.source)).size;
</script>

<div class="tool">
  <p class="lede">
    SNMP trap receiver — listens for traps your devices send and shows them live, newest first, so you
    catch the event the moment it fires (a link going down, a reboot, an auth failure). Filter by source,
    trap or varbind to find the one you want. In-memory and ephemeral: it keeps the most recent traps
    while running and clears on stop — a quick catch on a laptop, not long-term retention.
  </p>
  <p class="help">
    Point your devices' trap destination at this machine on the <strong>Listen UDP</strong> port, then
    <strong>Start</strong>. The port defaults high (1162) so it binds without admin — set it to
    <span class="mono">162</span> only if you've launched Meridian elevated. Each row is one trap:
    its <strong>source</strong>, the <strong>trap</strong> (name or OID), and its <strong>varbinds</strong>.
    v1 and v2c; the community is never shown. It holds an open UDP port while running and stops on Stop.
  </p>

  <LiveTail
    bind:port
    {running}
    {start}
    {stop}
    bind:filter
    {total}
    {dropped}
    hosts={hostCount}
    shownCount={shown.length}
    {err}
    note="Rolling buffer · last 10,000 traps · in-memory, cleared on stop · high port (no admin); set 162 if elevated"
  >
    {#each shown as r (r.seq)}
      <div class="trow">
        <span class="ts mono">{fmtTime(r.t)}</span>
        <span class="src mono">{r.source}</span>
        <span class="trap mono" title={r.trapOid}>{trapLabel(r)}</span>
        <span class="vb mono" title={vbText(r)}>{vbText(r) || "—"}</span>
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

  .trow { display: grid; grid-template-columns: 74px 94px 1.4fr 2.6fr; gap: 12px; align-items: baseline; padding: 5px 14px; border-top: 1px solid var(--border-soft); font-size: 0.78rem; }
  .trow:first-child { border-top: none; }
  .ts { color: var(--text-faint); }
  .src { color: var(--text); }
  .trap { color: var(--text-strong); font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .vb { color: var(--text-dim); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
</style>
