<script lang="ts">
  import { requestedTool } from "./nav";

  type Tool = { id: string; name: string; group: string; ready: boolean };
  export let tools: Tool[] = [];

  let open = false;
  let query = "";
  let idx = 0;
  let inputEl: HTMLInputElement;

  function show() {
    open = true;
    query = "";
    idx = 0;
    // focus after the element mounts
    setTimeout(() => inputEl?.focus(), 0);
  }
  function hide() {
    open = false;
  }

  // Smart-paste: recognise what was typed/pasted and point at the right tool,
  // carrying the value so tools that support it pre-fill (via requestedTool).
  function detect(s: string): { tool: string; value: string; type: string }[] {
    const q = s.trim();
    if (!q) return [];
    const out: { tool: string; value: string; type: string }[] = [];
    const isV4 = /^(\d{1,3}\.){3}\d{1,3}$/.test(q);
    const isMac = /^([0-9a-f]{2}[:-]){5}[0-9a-f]{2}$/i.test(q);
    const isJwt = /^ey[A-Za-z0-9_-]+\.ey[A-Za-z0-9_-]+\./.test(q);
    if (/^(\d{1,3}\.){3}\d{1,3}\/\d{1,2}$/.test(q)) out.push({ tool: "subnet", value: q, type: "CIDR" });
    else if (isV4) out.push({ tool: "lookup", value: q, type: "IPv4 address" });
    else if (!isMac && /^([0-9a-f]{1,4}:){2,}[0-9a-f:]*$/i.test(q)) out.push({ tool: "lookup", value: q, type: "IPv6 address" });
    if (isMac) out.push({ tool: "macvendor", value: q, type: "MAC address" });
    if (isJwt) out.push({ tool: "jwt", value: q, type: "JWT" });
    if (q.includes("-----BEGIN")) out.push({ tool: "certinfo", value: q, type: "PEM / certificate" });
    if (/^https?:\/\//i.test(q)) out.push({ tool: "sechdr", value: q, type: "URL" });
    if (/^[^@\s]+@[^@\s]+\.[a-z]{2,}$/i.test(q)) out.push({ tool: "emailauth", value: q, type: "email" });
    if (/^(0x)?[0-9a-f]{32}$|^[0-9a-f]{40}$|^[0-9a-f]{64}$/i.test(q)) out.push({ tool: "hashid", value: q, type: "hash" });
    if (!isV4 && !isJwt && /^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9-]+)+$/i.test(q) && !q.includes("/"))
      out.push({ tool: "lookup", value: q, type: "domain" });
    // de-dupe by tool, keep first (most specific)
    const seen = new Set<string>();
    return out.filter((d) => (seen.has(d.tool) ? false : (seen.add(d.tool), true)));
  }

  // Fuzzy: subsequence match on name+group, ranked startsWith > word-start > contains.
  function score(tool: Tool, q: string): number {
    const name = tool.name.toLowerCase();
    const hay = (tool.name + " " + tool.group + " " + tool.id).toLowerCase();
    if (!q) return 1;
    if (name.startsWith(q)) return 100 - name.length;
    if (name.includes(q)) return 60 - name.length;
    if (hay.includes(q)) return 30 - hay.length;
    // subsequence
    let i = 0;
    for (const ch of hay) if (ch === q[i]) i++;
    return i === q.length ? 10 - hay.length : -1;
  }

  type Row = { kind: "smart" | "tool"; id: string; label: string; sub: string; value?: string };

  $: q = query.trim().toLowerCase();
  $: detections = detect(query);
  $: toolRows = tools
    .filter((t) => t.ready)
    .map((t) => ({ t, s: score(t, q) }))
    .filter((x) => x.s > -1)
    .sort((a, b) => b.s - a.s)
    .slice(0, 40)
    .map(({ t }) => ({ kind: "tool", id: t.id, label: t.name, sub: t.group } as Row));
  $: rows = [
    ...detections.map(
      (d) =>
        ({
          kind: "smart",
          id: d.tool,
          value: d.value,
          label: toolName(d.tool),
          sub: `detected ${d.type}`,
        } as Row),
    ),
    ...toolRows.filter((r) => !detections.some((d) => d.tool === r.id)),
  ];
  $: if (idx >= rows.length) idx = Math.max(0, rows.length - 1);

  function toolName(id: string): string {
    return tools.find((t) => t.id === id)?.name ?? id;
  }

  function choose(r: Row) {
    if (!r) return;
    if (r.value) requestedTool.set({ tool: r.id, value: r.value });
    else requestedTool.set({ tool: r.id });
    hide();
  }

  function onKey(e: KeyboardEvent) {
    if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") {
      e.preventDefault();
      open ? hide() : show();
      return;
    }
    if (!open) return;
    if (e.key === "Escape") {
      e.preventDefault();
      hide();
    } else if (e.key === "ArrowDown") {
      e.preventDefault();
      idx = Math.min(idx + 1, rows.length - 1);
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      idx = Math.max(idx - 1, 0);
    } else if (e.key === "Enter") {
      e.preventDefault();
      choose(rows[idx]);
    }
  }
</script>

<svelte:window on:keydown={onKey} />

{#if open}
  <div class="scrim" on:mousedown={hide} role="presentation">
    <div class="palette" on:mousedown|stopPropagation role="dialog" aria-label="Command palette">
      <input
        bind:this={inputEl}
        bind:value={query}
        class="q mono"
        placeholder="Jump to a tool, or paste an IP / CIDR / domain / JWT / PEM / MAC / hash…"
        autocomplete="off"
        spellcheck="false"
        aria-label="Command palette search"
      />
      <div class="rows">
        {#each rows as r, i}
          <button
            class="row"
            class:on={i === idx}
            on:mouseenter={() => (idx = i)}
            on:click={() => choose(r)}
          >
            <span class="label">{r.label}</span>
            <span class="sub" class:smart={r.kind === "smart"}>{r.sub}</span>
          </button>
        {/each}
        {#if rows.length === 0}
          <div class="empty">No matching tool.</div>
        {/if}
      </div>
      <div class="hint"><kbd>↑</kbd><kbd>↓</kbd> move · <kbd>↵</kbd> open · <kbd>esc</kbd> close</div>
    </div>
  </div>
{/if}

<style>
  .scrim { position: fixed; inset: 0; background: color-mix(in srgb, #000 42%, transparent); display: flex; justify-content: center; align-items: flex-start; padding-top: 12vh; z-index: 1000; }
  .palette { width: min(620px, 92vw); background: var(--card, #1a1d24); border: 1px solid var(--border); border-radius: 12px; box-shadow: 0 24px 60px rgba(0,0,0,.45); overflow: hidden; }
  .q { width: 100%; box-sizing: border-box; background: transparent; border: none; border-bottom: 1px solid var(--border); color: var(--text); font-size: 0.95rem; padding: 14px 16px; outline: none; }
  .rows { max-height: 46vh; overflow: auto; padding: 6px; }
  .row { display: flex; align-items: baseline; justify-content: space-between; gap: 12px; width: 100%; text-align: left; background: transparent; border: none; border-radius: 8px; padding: 9px 12px; color: var(--text); font: inherit; font-size: 0.86rem; cursor: pointer; }
  .row.on { background: var(--active, color-mix(in srgb, var(--accent) 18%, transparent)); }
  .label { color: var(--text-strong, var(--text)); }
  .sub { color: var(--text-faint); font-size: 0.72rem; white-space: nowrap; }
  .sub.smart { color: var(--accent); font-weight: 600; }
  .empty { padding: 16px; color: var(--text-dim); font-size: 0.84rem; }
  .hint { border-top: 1px solid var(--border); padding: 7px 12px; font-size: 0.68rem; color: var(--text-faint); display: flex; gap: 7px; align-items: center; }
  kbd { background: var(--well); border: 1px solid var(--border); border-radius: 4px; padding: 0 5px; font-family: var(--font-mono, ui-monospace, monospace); font-size: 0.66rem; }
  .mono { font-family: var(--font-mono, ui-monospace, monospace); }
</style>
