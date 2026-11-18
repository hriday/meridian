<script lang="ts">
  import { themePref, themeLabel, themeIcon, type ThemePref } from "./theme";

  export let open = false;
  export let onClose: () => void = () => {};

  const themes: ThemePref[] = ["auto", "light", "dark", "terminal"];
  const pick = (t: ThemePref) => themePref.set(t);
  const onKey = (e: KeyboardEvent) => {
    if (e.key === "Escape") onClose();
  };
</script>

<svelte:window on:keydown={onKey} />

{#if open}
  <div class="overlay" on:click={onClose} role="presentation">
    <div class="panel" on:click|stopPropagation role="dialog" aria-modal="true" aria-label="Settings">
      <div class="phead">
        <h2>Settings</h2>
        <button class="x" on:click={onClose} aria-label="Close settings">×</button>
      </div>

      <section>
        <h3>Appearance</h3>
        <p class="hint">How Meridian looks. Saved on this computer and applied every launch.</p>
        <div class="seg" role="radiogroup" aria-label="Theme">
          {#each themes as t}
            <button
              class="opt"
              class:on={$themePref === t}
              role="radio"
              aria-checked={$themePref === t}
              on:click={() => pick(t)}
            >
              <span class="oi">{themeIcon[t]}</span>{themeLabel[t]}
            </button>
          {/each}
        </div>
        <p class="sub"><strong>Auto</strong> follows your operating system's light/dark setting and tracks it live.</p>
      </section>

      <p class="foot">Preferences are stored on this machine — more will land here over time.</p>
    </div>
  </div>
{/if}

<style>
  .overlay {
    position: fixed;
    inset: 0;
    background: color-mix(in srgb, #000 45%, transparent);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 50;
    padding: 16px;
  }
  .panel {
    width: 440px;
    max-width: 100%;
    background: var(--card);
    border: 1px solid var(--border);
    border-radius: 14px;
    box-shadow: 0 20px 60px rgba(0, 0, 0, 0.35);
    padding: 18px 20px 16px;
  }
  .phead {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 14px;
  }
  .phead h2 {
    margin: 0;
    font-size: 1.05rem;
    color: var(--text-strong, var(--text));
  }
  .x {
    background: none;
    border: 0;
    color: var(--text-dim);
    font-size: 1.4rem;
    line-height: 1;
    cursor: pointer;
    padding: 0 4px;
  }
  .x:hover {
    color: var(--text);
  }
  section h3 {
    margin: 0 0 4px;
    font-size: 0.72rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-dim);
  }
  .hint {
    margin: 0 0 10px;
    font-size: 0.78rem;
    color: var(--text-dim);
    line-height: 1.5;
  }
  .seg {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 8px;
  }
  .opt {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 9px 12px;
    font: inherit;
    font-size: 0.86rem;
    color: var(--text);
    background: var(--well);
    border: 1px solid var(--border);
    border-radius: 9px;
    cursor: pointer;
  }
  .opt:hover {
    border-color: var(--accent);
  }
  .opt.on {
    border-color: var(--accent);
    background: var(--accent-ring, color-mix(in srgb, var(--accent) 14%, transparent));
    color: var(--text-strong, var(--text));
    font-weight: 550;
  }
  .oi {
    font-size: 1rem;
  }
  .sub {
    margin: 10px 0 0;
    font-size: 0.74rem;
    color: var(--text-faint, var(--text-dim));
    line-height: 1.5;
  }
  .foot {
    margin: 16px 0 0;
    padding-top: 12px;
    border-top: 1px solid var(--border);
    font-size: 0.72rem;
    color: var(--text-faint, var(--text-dim));
  }
</style>
