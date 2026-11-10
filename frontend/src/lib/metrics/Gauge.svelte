<script lang="ts">
  // A 270° radial gauge. Dumb: the caller supplies the display value, the fill
  // fraction, and a state that colors the arc.
  export let value: string | number = "–";
  export let unit = "";
  export let label = "";
  export let pct = 0; // 0..1 fill
  export let state: "good" | "warn" | "bad" | "idle" = "idle";

  const cx = 60, cy = 58, r = 42, start = 135, sweep = 270;
  function polar(a: number) {
    const rad = (a * Math.PI) / 180;
    return [cx + r * Math.cos(rad), cy + r * Math.sin(rad)];
  }
  function arc(a0: number, a1: number) {
    const [x0, y0] = polar(a0), [x1, y1] = polar(a1);
    const large = a1 - a0 > 180 ? 1 : 0;
    return `M${x0.toFixed(1)} ${y0.toFixed(1)} A${r} ${r} 0 ${large} 1 ${x1.toFixed(1)} ${y1.toFixed(1)}`;
  }
  $: clamped = Math.max(0.001, Math.min(1, pct || 0));
  $: col = state === "bad" ? "var(--bad, #ff6b6b)" : state === "warn" ? "var(--warn, #d9a441)" : state === "good" ? "var(--good, #5ad17a)" : "var(--text-faint, #6b7488)";
</script>

<div class="gauge">
  <svg viewBox="0 0 120 98" width="100%" role="img" aria-label="{label} {value}{unit}">
    <path d={arc(start, start + sweep)} fill="none" stroke="var(--track, #2a3142)" stroke-width="9" stroke-linecap="round" />
    <path d={arc(start, start + sweep * clamped)} fill="none" stroke={col} stroke-width="9" stroke-linecap="round" />
    <text x="60" y="56" text-anchor="middle" font-size="19" font-weight="650" fill="var(--text)" class="mono">{value}</text>
    <text x="60" y="72" text-anchor="middle" font-size="9" fill="var(--text-faint)" class="mono">{unit}</text>
  </svg>
  <div class="glabel">{label}</div>
</div>

<style>
  .gauge { background: var(--card); border: 1px solid var(--border-soft); border-radius: 11px; padding: 10px 8px 8px; text-align: center; }
  .mono { font-family: var(--mono, ui-monospace, monospace); }
  .glabel { font-size: 0.64rem; text-transform: uppercase; letter-spacing: 0.05em; color: var(--text-faint); margin-top: 2px; }
</style>
