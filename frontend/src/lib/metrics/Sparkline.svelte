<script lang="ts">
  // A tiny inline-SVG sparkline. Dumb: caller supplies the 0..100 values and a
  // colour. Renders nothing meaningful with fewer than two points.
  export let values: number[] = [];
  export let color = "var(--good)";
  const w = 92, h = 26, max = 100;

  $: pts =
    values.length < 2
      ? ""
      : values
          .map((v, i) => `${((i / (values.length - 1)) * w).toFixed(1)},${(h - 2 - (Math.max(0, Math.min(max, v)) / max) * (h - 4)).toFixed(1)}`)
          .join(" ");
</script>

<svg class="spark" width={w} height={h} viewBox="0 0 {w} {h}" role="presentation">
  {#if pts}
    <polyline points={pts} fill="none" stroke={color} stroke-width="1.6" />
  {/if}
</svg>

<style>
  .spark { display: block; height: 26px; }
</style>
