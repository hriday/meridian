<script lang="ts">
  import { PacketScan } from "../../../wailsjs/go/app/App.js";

  let seconds = 8;
  let loading = false;
  let error = "";
  let needsPriv = false;
  let res: any = null;

  async function run() {
    if (loading) return;
    loading = true;
    error = "";
    needsPriv = false;
    res = null;
    try {
      res = await PacketScan(Number(seconds) || 8);
    } catch (e: any) {
      const msg = String(e?.message ?? e);
      if (/privile|sudo|ChmodBPF/i.test(msg)) needsPriv = true;
      else error = msg;
    } finally {
      loading = false;
    }
  }

  const bytes = (n: number): string => {
    if (!n) return "0 B";
    const u = ["B", "KB", "MB", "GB", "TB"];
    let i = 0;
    while (n >= 1024 && i < u.length - 1) { n /= 1024; i++; }
    return `${i === 0 ? n : n.toFixed(1)} ${u[i]}`;
  };
</script>

<div class="tool">
  <p class="lede">
    A <strong>tcpdump-lite</strong>: capture on your interface for a few seconds and summarize what's on the wire —
    <strong>top talkers</strong>, protocol mix, <strong>TCP resets &amp; retransmits</strong>, <strong>DNS errors</strong>,
    and the <strong>TLS SNIs</strong> seen. Use it to tell <em>what</em> is happening when ping/MTR/RTP flags an anomaly.
    Needs <strong>raw-capture access</strong>: root/<span class="mono">sudo</span>, or on macOS membership in a BPF-access
    group (e.g. Wireshark's <span class="mono">ChmodBPF</span>) — if a capture just worked, you already have it, no admin
    needed. On a switched LAN you only see traffic you're in the path of.
  </p>

  <div class="row">
    <label>Capture for <input class="field" type="number" min="1" max="60" bind:value={seconds} /> s</label>
    <button class="btn" on:click={run} disabled={loading}>{loading ? `Capturing ${seconds}s…` : "Capture"}</button>
  </div>

  {#if needsPriv}
    <div class="empty">Raw packet capture needs elevated privileges — run the CLI with <span class="mono">sudo meridian-cli pktscan</span>, or launch Meridian elevated (on macOS you can also install ChmodBPF).</div>
  {/if}
  {#if error}<p class="err">{error}</p>{/if}

  {#if res}
    <div class="summary">
      <span class="pill">{res.packets} packets</span>
      <span class="pill">{bytes(res.bytes)}</span>
      <span class="pill">{res.tcpSyns} SYN</span>
      <span class="pill warn">{res.tcpResets} RST</span>
      <span class="pill warn">{res.retransmits} retransmit{res.retransmits === 1 ? "" : "s"}</span>
      <span class="sub">on {res.interface} · {res.seconds?.toFixed(1)}s</span>
    </div>

    {#if res.packets === 0}
      <div class="empty">No packets. On a switched LAN you only see traffic you're in the path of (your own, broadcast/multicast, or a SPAN/mirror port).</div>
    {:else}
      {#if res.protocols && res.protocols.length}
        <div class="protos">
          {#each res.protocols as p}<span class="chip"><b>{p.proto}</b> {p.packets}</span>{/each}
        </div>
      {/if}

      <div class="sec">
        <div class="sechead">Top talkers</div>
        {#each res.topTalkers || [] as t}
          <div class="talker">
            <span class="ip mono">{t.ip}</span>
            <span class="tb">{bytes(t.bytes)}</span>
            <span class="tp">{t.packets} pkts</span>
          </div>
        {/each}
      </div>

      {#if res.dnsErrors && res.dnsErrors.length}
        <div class="sec">
          <div class="sechead warn">DNS errors</div>
          {#each res.dnsErrors as d}<span class="chip warn"><b>{d.name}</b> {d.count}</span>{/each}
        </div>
      {/if}

      {#if res.snis && res.snis.length}
        <div class="sec">
          <div class="sechead">TLS SNIs <span class="count">{res.snis.length}</span></div>
          <div class="protos">{#each res.snis as s}<span class="chip mono">{s}</span>{/each}</div>
        </div>
      {/if}
    {/if}
  {/if}
</div>

<style>
  .tool { padding: 14px 16px 30px; max-width: 880px; }
  .lede { margin: 0 0 14px; font-size: 0.82rem; color: var(--text-dim); line-height: 1.5; }
  .row { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; }
  label { font-size: 0.84rem; color: var(--text-dim); display: flex; align-items: center; gap: 7px; }
  .field { background: var(--well); border: 1px solid var(--field-border, var(--border)); border-radius: 7px; padding: 7px 9px; color: var(--text); font-size: 0.86rem; outline: none; width: 58px; }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .btn { background: var(--accent); color: var(--on-accent, #052e2b); border: 1px solid transparent; border-radius: 8px; padding: 8px 16px; font: inherit; font-size: 0.86rem; font-weight: 550; cursor: pointer; }
  .btn:disabled { opacity: .55; cursor: default; }
  .err { color: var(--danger, #e5484d); font-size: 0.82rem; margin: 12px 0 0; }
  .empty { margin-top: 16px; padding: 12px 15px; border: 1px solid var(--border); border-radius: 10px; background: var(--card); font-size: 0.84rem; color: var(--text-dim); }
  .summary { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; margin: 18px 0 10px; }
  .pill { font-size: 0.74rem; font-weight: 650; padding: 3px 10px; border-radius: 20px; border: 1px solid var(--border); background: var(--well); color: var(--text); }
  .pill.warn { background: color-mix(in srgb, var(--danger, #e5484d) 15%, transparent); color: var(--danger, #e5484d); border-color: color-mix(in srgb, var(--danger, #e5484d) 40%, transparent); }
  .sub { font-size: 0.76rem; color: var(--text-dim); margin-left: 4px; }
  .protos { display: flex; gap: 6px; flex-wrap: wrap; margin: 6px 0 2px; }
  .chip { font-size: 0.72rem; background: var(--well); border: 1px solid var(--border); border-radius: 6px; padding: 2px 8px; color: var(--text-dim); }
  .chip.warn { color: var(--danger, #e5484d); border-color: color-mix(in srgb, var(--danger, #e5484d) 40%, transparent); }
  .sec { margin-top: 16px; }
  .sechead { font-size: 0.74rem; text-transform: uppercase; letter-spacing: .05em; color: var(--text-dim); font-weight: 700; margin-bottom: 8px; display: flex; align-items: center; gap: 8px; }
  .sechead.warn { color: var(--danger, #e5484d); }
  .count { background: var(--well); border: 1px solid var(--border); border-radius: 20px; padding: 1px 8px; font-size: 0.72rem; color: var(--text); }
  .talker { display: flex; align-items: center; gap: 12px; border: 1px solid var(--border); border-radius: 9px; padding: 7px 12px; margin-bottom: 6px; background: var(--card); font-size: 0.84rem; }
  .ip { min-width: 200px; color: var(--text-strong, var(--text)); font-weight: 600; }
  .tb { min-width: 90px; color: var(--text); }
  .tp { color: var(--text-dim); }
  .mono { font-family: var(--font-mono, ui-monospace, monospace); }
</style>
