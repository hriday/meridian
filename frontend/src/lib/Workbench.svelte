<script lang="ts">
  import SubnetCalc from "./tools/SubnetCalc.svelte";
  import { themePref, cycleTheme, themeIcon, themeLabel } from "./theme";

  type Tool = { id: string; name: string; group: string; ready: boolean };
  const tools: Tool[] = [
    { id: "lookup", name: "Lookup", group: "Internet", ready: false },
    { id: "trace", name: "Traceroute (AS)", group: "Internet", ready: false },
    { id: "bgp", name: "BGP & looking glass", group: "Internet", ready: false },
    { id: "subnet", name: "Subnet calculator", group: "Addressing", ready: true },
    { id: "ipspace", name: "IP space", group: "Addressing", ready: false },
    { id: "ping", name: "Ping sweep", group: "Local network", ready: false },
    { id: "ports", name: "Port scanner", group: "Local network", ready: false },
  ];
  const groups = [...new Set(tools.map((t) => t.group))];

  let active = "subnet";
  let searchEl: HTMLInputElement;
  let filter = "";

  function onKey(e: KeyboardEvent) {
    if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") {
      e.preventDefault();
      searchEl?.focus();
    }
  }

  $: visible = (g: string) =>
    tools.filter((t) => t.group === g && t.name.toLowerCase().includes(filter.toLowerCase()));
</script>

<svelte:window on:keydown={onKey} />

<div class="wb">
  <header class="bar">
    <span class="brand">Meridian</span>
    <input
      bind:this={searchEl}
      bind:value={filter}
      class="search mono"
      placeholder="Search tools, IPs, ASNs…"
      aria-label="Search"
    />
    <kbd>⌘K</kbd>
    <button
      class="theme"
      on:click={cycleTheme}
      title={`Theme: ${themeLabel[$themePref]} — click to change`}
      aria-label={`Theme: ${themeLabel[$themePref]}`}
    >
      <span class="ticon">{themeIcon[$themePref]}</span>
      <span class="tlabel">{themeLabel[$themePref]}</span>
    </button>
  </header>

  <div class="body">
    <nav class="side">
      {#each groups as g}
        {#if visible(g).length}
          <div class="grp">{g}</div>
          {#each visible(g) as t}
            <button
              class="item"
              class:on={active === t.id}
              class:disabled={!t.ready}
              on:click={() => t.ready && (active = t.id)}
              title={t.ready ? t.name : `${t.name} — coming soon`}
            >
              {t.name}{#if !t.ready}<span class="soon">soon</span>{/if}
            </button>
          {/each}
        {/if}
      {/each}
    </nav>

    <main class="main">
      <div class="tabs"><div class="tab on">Subnet calculator</div></div>
      {#if active === "subnet"}<SubnetCalc />{/if}
    </main>
  </div>
</div>

<style>
  .wb { display: flex; flex-direction: column; height: 100vh; }
  .bar {
    display: flex; align-items: center; gap: 10px; padding: 8px 12px;
    background: var(--bar); border-bottom: 1px solid var(--border);
  }
  .brand { font-weight: 600; letter-spacing: .02em; color: var(--text-strong); }
  .search {
    flex: 1; background: var(--well); border: 1px solid var(--field-border); border-radius: 7px;
    padding: 6px 10px; color: var(--text); font-size: 0.82rem; outline: none;
  }
  .search:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  kbd {
    font: 11px ui-monospace, monospace; color: var(--text-dim);
    border: 1px solid var(--kbd-border); border-radius: 4px; padding: 1px 6px;
  }
  .theme {
    display: flex; align-items: center; gap: 6px; background: none;
    border: 1px solid var(--kbd-border); border-radius: 6px; padding: 3px 8px;
    color: var(--text-dim); font-size: 0.72rem; cursor: pointer; line-height: 1;
  }
  .theme:hover { color: var(--text-strong); border-color: var(--field-border); background: var(--hover); }
  .ticon { font-size: 0.82rem; }
  @media (max-width: 560px) { .tlabel { display: none; } }
  .body { flex: 1; display: flex; min-height: 0; }
  .side {
    width: 190px; flex: none; background: var(--panel); border-right: 1px solid var(--border);
    padding: 8px; overflow-y: auto;
  }
  .grp {
    color: var(--text-faint2); font-size: 0.62rem; text-transform: uppercase; letter-spacing: .07em;
    margin: 12px 8px 4px;
  }
  .item {
    display: flex; align-items: center; justify-content: space-between; width: 100%;
    text-align: left; background: none; border: none; color: var(--item-text);
    padding: 5px 9px; border-radius: 6px; font-size: 0.84rem; cursor: pointer;
  }
  .item:hover:not(.disabled) { background: var(--hover); color: var(--text-strong); }
  .item.on { background: var(--active); color: var(--text-strong); }
  .item.disabled { color: var(--text-faint); cursor: default; }
  .soon { font-size: 0.6rem; color: var(--text-faint); border: 1px solid var(--pill-border); border-radius: 8px; padding: 0 6px; }
  .main { flex: 1; min-width: 0; overflow-y: auto; }
  .tabs { display: flex; gap: 2px; padding: 0 10px; background: var(--panel); border-bottom: 1px solid var(--border); }
  .tab { padding: 8px 12px; font-size: 0.82rem; color: var(--tab-inactive); border-bottom: 2px solid transparent; }
  .tab.on { color: var(--text-strong); border-color: var(--accent); }
  .mono { font-family: var(--font-mono); }
</style>
