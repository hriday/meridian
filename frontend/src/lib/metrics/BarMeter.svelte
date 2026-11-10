<script lang="ts">
  // A horizontal meter. Dumb: caller supplies the fill fraction and colour.
  export let pct = 0; // 0..100
  export let color = "var(--good)";
  $: w = Math.max(0, Math.min(100, pct || 0));
</script>

<div class="bar" role="presentation">
  <div class="fill" style="width:{w}%;background:{color}"></div>
</div>

<style>
  .bar { flex: 1; height: 7px; border-radius: 4px; background: var(--track, #2a3142); overflow: hidden; }
  .fill { height: 100%; border-radius: 4px; transition: width 0.3s ease; }
</style>
