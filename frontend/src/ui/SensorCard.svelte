<script lang="ts" module>
  export type Kind = "load" | "temp" | "fan" | "memory";
</script>

<script lang="ts">
  import * as Card from "$lib/components/ui/card";
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

<Card.Root class="flex h-[112px] gap-0 shadow-none ring-0 min-w-0 flex-col overflow-hidden sensor-reading rounded-none border-0 border-b border-border bg-transparent px-1 py-2.5" title="{name} · {source}">
  <div class="flex h-7 items-start justify-between gap-2">
    <div class="grid min-w-0 flex-1">
      <span class="truncate text-[11px] font-medium leading-tight">{name}</span>
      <span class="truncate text-[10px] leading-tight text-muted-foreground">{source}</span>
    </div>
    <span class="shrink-0 text-muted-foreground"><Icon name={ICONS[kind]} size={13} /></span>
  </div>
  <div class="mt-2 text-lg font-semibold leading-none tabular-nums">{value}<span class="ml-1 text-[9px] font-normal text-muted-foreground">{unit}</span></div>
  <svg class="mt-auto h-7 w-full shrink-0 overflow-visible" viewBox="0 0 {W} {H}" preserveAspectRatio="none" aria-hidden="true">
    {#if path}<path class="fill-primary/10" d="{path} L{W},{H} L0,{H} Z" />{/if}
    <path class={kind === "temp" ? "fill-none stroke-orange-400" : kind === "fan" ? "fill-none stroke-sky-400" : kind === "memory" ? "fill-none stroke-violet-400" : "fill-none stroke-primary"} style="stroke-width:1.5" d={path} />
  </svg>
</Card.Root>
