<script lang="ts">
  import Workbench from "./lib/Workbench.svelte";
  import "./lib/theme"; // applies the stored theme at startup
</script>

<Workbench />

<style>
  /* Design tokens. :root is the dark palette (the product's default identity);
     [data-theme="light"] and [data-theme="terminal"] swap in alternates.
     theme.ts sets data-theme to light/dark/terminal on <html> (resolving
     "auto" against the OS). Font tokens live on :root and apply to every
     theme. */
  :global(:root),
  :global(:root[data-theme="dark"]) {
    --font-sans: "Geist", -apple-system, BlinkMacSystemFont, "Segoe UI",
      "Helvetica Neue", Arial, sans-serif;
    --font-mono: "Geist Mono", ui-monospace, "SF Mono", Menlo, Consolas, monospace;

    --bg: #0f1115;
    --bar: #151821;
    --panel: #12151c;
    --card: #141820;
    --well: #0b0d12;
    --border: #262a33;
    --border-soft: #232834;
    --field-border: #2f3542;
    --kbd-border: #333a47;
    --pill-border: #2c333f;
    --hover: #192030;
    --active: #1e2533;
    --text: #d7dbe3;
    --text-strong: #ffffff;
    --text-dim: #7d8596;
    --text-faint: #596072;
    --text-faint2: #6b7385;
    --item-text: #aeb5c3;
    --tab-inactive: #8b93a4;
    --accent: #5b8cff;
    --accent-ring: rgba(91, 140, 255, 0.15);
    --btn: #2b4c9a;
    --btn-border: #3d63bd;
    --on-accent: #ffffff;
    --ghost-bg: #1b2130;
    --ghost-border: #2f3a52;
    --warn-bg: #1b1a14;
    --warn-border: #4a401f;
    --warn-text: #f2c46d;
    --good-bg: #102516;
    --good: #5ad17a;
  }

  :global(:root[data-theme="light"]) {
    --bg: #f5f7fa;
    --bar: #ffffff;
    --panel: #f0f2f6;
    --card: #ffffff;
    --well: #ffffff;
    --border: #e3e6ec;
    --border-soft: #e8ebf1;
    --field-border: #cfd5df;
    --kbd-border: #d2d8e2;
    --pill-border: #d8dde6;
    --hover: #eef1f7;
    --active: #e7edff;
    --text: #2a2f3a;
    --text-strong: #0f1218;
    --text-dim: #6a7183;
    --text-faint: #99a0b0;
    --text-faint2: #8a91a2;
    --item-text: #3a4150;
    --tab-inactive: #6a7183;
    --accent: #2f6bff;
    --accent-ring: rgba(47, 107, 255, 0.18);
    --btn: #2f6bff;
    --btn-border: #2f6bff;
    --on-accent: #ffffff;
    --ghost-bg: #eef1f7;
    --ghost-border: #cfd5df;
    --warn-bg: #fff7e6;
    --warn-border: #f0d79a;
    --warn-text: #8a6a16;
    --good-bg: #e7f6ec;
    --good: #15803d;
  }

  /* Terminal: green phosphor on CRT black. Paired with monospace everywhere
     and a faint glow (below) for the old-terminal feel. */
  :global(:root[data-theme="terminal"]) {
    --bg: #000000;
    --bar: #040804;
    --panel: #030703;
    --card: #050a05;
    --well: #000000;
    --border: #0f3d1f;
    --border-soft: #0c3019;
    --field-border: #155a2c;
    --kbd-border: #155a2c;
    --pill-border: #0f3d1f;
    --hover: #0a2110;
    --active: #0d2e16;
    --text: #33ff66;
    --text-strong: #c6ffd6;
    --text-dim: #19a349;
    --text-faint: #0e7a34;
    --text-faint2: #11863a;
    --item-text: #2fe55f;
    --tab-inactive: #178f40;
    --accent: #4dff88;
    --accent-ring: rgba(51, 255, 102, 0.22);
    --btn: #0d2e16;
    --btn-border: #1f7a3c;
    --on-accent: #8affb0;
    --ghost-bg: #0a2110;
    --ghost-border: #155a2c;
    --warn-bg: #1a1400;
    --warn-border: #5a4a0f;
    --warn-text: #ffcf4d;
    --good-bg: #04160a;
    --good: #4dff88;
  }

  :global(body) {
    margin: 0;
    background: var(--bg);
    color: var(--text);
    font-family: var(--font-sans);
    -webkit-font-smoothing: antialiased;
    transition: background-color 0.15s ease, color 0.15s ease;
  }
  :global(*) { box-sizing: border-box; }

  /* Terminal theme: monospace UI + phosphor glow on prominent text. */
  :global(:root[data-theme="terminal"] body) { font-family: var(--font-mono); }
  :global(:root[data-theme="terminal"] .mono),
  :global(:root[data-theme="terminal"] h2),
  :global(:root[data-theme="terminal"] h3),
  :global(:root[data-theme="terminal"] .brand) {
    text-shadow: 0 0 6px rgba(57, 255, 120, 0.35);
  }
</style>
