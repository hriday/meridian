<script lang="ts">
  import SubnetCalc from "./tools/SubnetCalc.svelte";

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
    background: #151821; border-bottom: 1px solid #262a33;
  }
  .brand { font-weight: 600; letter-spacing: .02em; }
  .search {
    flex: 1; background: #0b0d12; border: 1px solid #2f3542; border-radius: 7px;
    padding: 6px 10px; color: #e7ebf3; font-size: 0.82rem; outline: none;
  }
  .search:focus { border-color: #5b8cff; box-shadow: 0 0 0 3px rgba(91,140,255,.15); }
  kbd {
    font: 11px ui-monospace, monospace; color: #7d8596;
    border: 1px solid #333a47; border-radius: 4px; padding: 1px 6px;
  }
  .body { flex: 1; display: flex; min-height: 0; }
  .side {
    width: 190px; flex: none; background: #12151c; border-right: 1px solid #262a33;
    padding: 8px; overflow-y: auto;
  }
  .grp {
    color: #6b7385; font-size: 0.62rem; text-transform: uppercase; letter-spacing: .07em;
    margin: 12px 8px 4px;
  }
  .item {
    display: flex; align-items: center; justify-content: space-between; width: 100%;
    text-align: left; background: none; border: none; color: #aeb5c3;
    padding: 5px 9px; border-radius: 6px; font-size: 0.84rem; cursor: pointer;
  }
  .item:hover:not(.disabled) { background: #192030; color: #fff; }
  .item.on { background: #1e2533; color: #fff; }
  .item.disabled { color: #596072; cursor: default; }
  .soon { font-size: 0.6rem; color: #596072; border: 1px solid #2c333f; border-radius: 8px; padding: 0 6px; }
  .main { flex: 1; min-width: 0; overflow-y: auto; }
  .tabs { display: flex; gap: 2px; padding: 0 10px; background: #12151c; border-bottom: 1px solid #262a33; }
  .tab { padding: 8px 12px; font-size: 0.82rem; color: #8b93a4; border-bottom: 2px solid transparent; }
  .tab.on { color: #fff; border-color: #5b8cff; }
  .mono { font-family: "JetBrains Mono", ui-monospace, "SF Mono", Menlo, monospace; }
</style>
