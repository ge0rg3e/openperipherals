<script lang="ts" module>
  export type Kind = "load" | "temp" | "fan" | "memory";
</script>

<script lang="ts">
  import Icon from "./Icon.svelte";

  let {name, source, value, unit, kind, history = [], max}:
    {name: string; source: string; value: string; unit: string; kind: Kind; history?: number[]; max?: number} = $props();

  const ICONS = {load: "chip", temp: "temp", fan: "fan", memory: "memory"} as const;
  const W = 100, H = 26;

  // Sparkline over the last minute; a fixed max keeps percentages comparable.
  const path = $derived.by(() => {
    if (history.length < 2) return "";
    const hi = max ?? Math.max(...history) * 1.15;
    const lo = max !== undefined ? 0 : Math.min(...history) * 0.85;
    const span = Math.max(hi - lo, 1e-6);
    return history
      .map((v, i) => `${i ? "L" : "M"}${(i / (history.length - 1)) * W},${H - ((v - lo) / span) * (H - 2) - 1}`)
      .join(" ");
  });
</script>

<div class="sensor-card kind-{kind}" title="{name} · {source}">
  <div class="sensor-top">
    <div class="sensor-names">
      <span class="sensor-name">{name}</span>
      <span class="sensor-source">{source}</span>
    </div>
    <span class="sensor-icon"><Icon name={ICONS[kind]} size={14} /></span>
  </div>
  <div class="sensor-value">{value}<span>{unit}</span></div>
  <svg class="sensor-spark" viewBox="0 0 {W} {H}" preserveAspectRatio="none" aria-hidden="true">
    {#if path}<path class="area" d="{path} L{W},{H} L0,{H} Z" />{/if}
    <path class="line" d={path} />
  </svg>
</div>
