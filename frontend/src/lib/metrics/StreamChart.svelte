<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import uPlot from "uplot";

  // The tool owns the ring buffer and passes the window each update. Either a
  // single rtt series (ping, with anomaly/loss highlighting) OR a `series` set
  // of {values, color} lines (e.g. rx/tx throughput).
  export let xs: number[] = [];
  export let rtts: (number | null)[] = [];
  export let series: { values: (number | null)[]; color: string }[] | null = null;
  export let bandLo = 0;
  export let bandHi = 0;
  export let height = 150;

  let el: HTMLDivElement;
  let u: uPlot | null = null;
  let builtCols = 0;
  let themeObs: MutationObserver | null = null;
  let roObs: ResizeObserver | null = null;

  function cssVar(name: string, fallback: string): string {
    if (!el) return fallback;
    const v = getComputedStyle(el).getPropertyValue(name).trim();
    return v || fallback;
  }

  $: usingSeries = !!(series && series.length);
  // anomaly series (single-series mode only): rtt where it's out of band.
  $: anomalies = usingSeries
    ? []
    : rtts.map((v) => (v != null && (bandHi > 0 || bandLo > 0) && (v < bandLo || v > bandHi) ? v : null));
  $: lossXs = usingSeries ? [] : xs.filter((_, i) => rtts[i] == null);
  $: wantCols = usingSeries ? 1 + series!.length : 3;

  function data(): uPlot.AlignedData {
    return usingSeries ? [xs, ...series!.map((s) => s.values)] as any : [xs, rtts as any, anomalies as any];
  }

  function makeOpts(w: number): uPlot.Options {
    const good = cssVar("--good", "#5ad17a");
    const bad = cssVar("--bad", "#ff6b6b");
    const grid = cssVar("--border-soft", "#262c3a");
    const text = cssVar("--text-faint", "#6b7488");
    const bandFill = cssVar("--good", "#5ad17a");
    const resolve = (c: string) => (c.startsWith("--") ? cssVar(c, "#888") : c);
    const seriesOpts: any[] = usingSeries
      ? [{}, ...series!.map((s) => ({ stroke: resolve(s.color), width: 1.7, spanGaps: false, points: { show: false } }))]
      : [
          {},
          { stroke: good, width: 1.8, spanGaps: false, points: { show: false } },
          { stroke: bad, width: 2, spanGaps: false, points: { show: true, size: 5, stroke: bad, fill: bad } },
        ];
    return {
      width: w,
      height,
      cursor: { show: false },
      legend: { show: false },
      scales: { x: { time: false } },
      axes: [
        { show: false },
        { stroke: text, grid: { stroke: grid, width: 1 }, ticks: { stroke: grid }, size: 40,
          font: "10px ui-monospace, monospace" },
      ],
      series: seriesOpts,
      hooks: {
        drawClear: [
          (self: uPlot) => {
            const ctx = self.ctx;
            const { left, top, width, height } = self.bbox;
            if (bandHi > bandLo) {
              const yTop = self.valToPos(bandHi, "y", true);
              const yBot = self.valToPos(bandLo, "y", true);
              ctx.save();
              ctx.fillStyle = bandFill;
              ctx.globalAlpha = 0.1;
              ctx.fillRect(left, yTop, width, yBot - yTop);
              ctx.restore();
            }
            // loss markers: red dashed verticals
            if (lossXs.length) {
              ctx.save();
              ctx.strokeStyle = bad;
              ctx.globalAlpha = 0.7;
              ctx.setLineDash([3, 3]);
              for (const lx of lossXs) {
                const px = self.valToPos(lx, "x", true);
                ctx.beginPath();
                ctx.moveTo(px, top);
                ctx.lineTo(px, top + height);
                ctx.stroke();
              }
              ctx.restore();
            }
          },
        ],
      },
    };
  }

  function build() {
    if (u) { u.destroy(); u = null; }
    if (!el) return;
    const w = el.clientWidth || 600;
    u = new uPlot(makeOpts(w), data(), el);
    builtCols = wantCols;
  }

  onMount(() => {
    build();
    roObs = new ResizeObserver(() => { if (u && el) u.setSize({ width: el.clientWidth || 600, height }); });
    roObs.observe(el);
    themeObs = new MutationObserver(build); // theme switch → rebuild to re-read CSS vars
    themeObs.observe(document.documentElement, { attributes: true, attributeFilter: ["data-theme"] });
  });

  onDestroy(() => {
    roObs?.disconnect();
    themeObs?.disconnect();
    u?.destroy();
    u = null;
  });

  // the series COUNT is fixed at construction, so a change in column count
  // (single ↔ multi, or a different number of lines) rebuilds the chart.
  $: if (u && wantCols !== builtCols) build();
  // push new data on prop change — reference xs/series/rtts/anomalies directly
  // so Svelte tracks them as dependencies (a data() call would hide them).
  $: if (u && wantCols === builtCols)
    u.setData(usingSeries ? ([xs, ...series!.map((s) => s.values)] as any) : [xs, rtts as any, anomalies as any]);
</script>

<div class="chartwrap" bind:this={el}></div>

<style>
  .chartwrap { width: 100%; }
  :global(.uplot), :global(.uplot *) { font-family: var(--mono, ui-monospace, monospace) !important; }
</style>
