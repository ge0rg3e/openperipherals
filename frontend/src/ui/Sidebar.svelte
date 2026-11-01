<script lang="ts">
  import {Button} from "$lib/components/ui/button";
  import EffectPreview from "./EffectPreview.svelte";
  import SensorCard, { type Kind } from "./SensorCard.svelte";
  import Switch from "./Switch.svelte";
  import ColorPicker from "./ColorPicker.svelte";
  import Slider from "./Slider.svelte";
  import {
    app,
    applyBrightness,
    applyMouseBrightness,
    applySync,
    SYNC_COLORS,
    SYNC_EFFECTS,
  } from "../lib/store.svelte";

  interface Reading {
    id: string;
    name: string;
    source: string;
    value: string;
    unit: string;
    kind: Kind;
    max?: number;
  }

  const gb = (b: number) => (b / 1024 ** 3).toFixed(1);

  const readings = $derived.by(() => {
    const out: Reading[] = [];
    const s = app.stats;
    if (s) {
      out.push({
        id: "cpu",
        name: "CPU load",
        source: "Processor",
        value: s.cpu >= 0 ? `${s.cpu}` : "–",
        unit: "%",
        kind: "load",
        max: 100,
      });
      out.push({
        id: "ram",
        name: "Memory",
        source: `${gb(s.ramTotal)} GB installed`,
        value: gb(s.ramUsed),
        unit: "GB",
        kind: "memory",
      });
      s.gpus?.forEach((g, i) => {
        out.push({
          id: `gpu${i}`,
          name: "GPU load",
          source: g.name,
          value: `${g.usage}`,
          unit: "%",
          kind: "load",
          max: 100,
        });
        out.push({
          id: `vram${i}`,
          name: "VRAM",
          source: `${gb(g.vramTotal)} GB`,
          value: gb(g.vramUsed),
          unit: "GB",
          kind: "memory",
        });
      });
    }
    for (const t of app.fans?.sensors ?? []) {
      if (t.temp < 0) continue;
      out.push({
        id: `t:${t.id}`,
        name: t.id.startsWith("gpu") ? "GPU temp" : `${t.name} temp`,
        source: t.id.startsWith("gpu") ? t.name : "Motherboard",
        value: `${t.temp}`,
        unit: "°C",
        kind: "temp",
      });
    }
    for (const f of app.fans?.fans ?? []) {
      if (f.rpm >= 0)
        out.push({
          id: `rpm:${f.id}`,
          name: f.name,
          source: f.source,
          value: `${f.rpm}`,
          unit: "RPM",
          kind: "fan",
        });
      else if (f.duty >= 0)
        out.push({
          id: `duty:${f.id}`,
          name: f.name,
          source: f.source,
          value: `${f.duty}`,
          unit: "%",
          kind: "fan",
          max: 100,
        });
    }
    return out;
  });

  const devices = $derived(
    [app.kb && "Keyboard", app.mouse && "Mouse"].filter(Boolean) as string[],
  );
  const usesColor = $derived(
    app.sync.effect === "static" || app.sync.effect === "breathing",
  );
  const isPreset = $derived(SYNC_COLORS.includes(app.sync.color.toLowerCase()));
</script>

<aside class="row-start-2 min-w-0 min-h-0 overflow-x-hidden overflow-y-auto border-r border-border bg-sidebar px-3 py-3 [scrollbar-color:color-mix(in_oklch,var(--muted-foreground)_35%,transparent)_transparent] [scrollbar-width:thin] [&::-webkit-scrollbar]:w-1.5 [&::-webkit-scrollbar-thumb]:rounded-full [&::-webkit-scrollbar-thumb]:bg-muted-foreground/30 [&::-webkit-scrollbar-track]:bg-transparent max-[680px]:hidden">
  <section class="min-w-0 border-b border-border pb-3">
    <div class="flex items-center justify-between gap-3">
      <div>
        <h3 class="text-xs font-semibold tracking-tight">Lighting</h3>
      </div>
      <Switch
        label="Sync lighting across devices"
        checked={app.sync.enabled}
        disabled={!devices.length}
        onchange={(on) => applySync({ enabled: on })}
      />
    </div>

    <div class="mt-3 flex items-center justify-between gap-3 text-[10px]">
      <span class="text-muted-foreground">Effect</span>
      <span class="font-medium"
        >{SYNC_EFFECTS.find(([id]) => id === app.sync.effect)?.[1]}</span
      >
    </div>
    <div class="mt-2 grid grid-cols-2 gap-1.5" role="radiogroup" aria-label="Sync effect">
      {#each SYNC_EFFECTS as [id, label]}
        <Button
          variant="outline"
          class={app.sync.effect === id
            ? "flex h-14 min-w-0 flex-col justify-center gap-1 rounded-md border-primary bg-[#171717] px-1.5 hover:border-primary/50"
            : "flex h-14 min-w-0 flex-col justify-center gap-1 rounded-md border-border bg-[#0c0c0c] px-1.5 transition-colors hover:border-primary/50"}
          disabled={!devices.length}
          role="radio"
          aria-checked={app.sync.effect === id}
          aria-label={label}
          title={label}
          onclick={() => applySync({ effect: id })}
        >
          <EffectPreview effect={id} color={app.sync.color} />
          <span class="max-w-full truncate text-[10px] text-muted-foreground">{label}</span>
        </Button>
      {/each}
    </div>

    {#if usesColor}
      <div class="mt-3 flex items-center justify-between gap-3 text-[10px]">
        <span class="text-muted-foreground">Colour</span><span class="font-medium"
          >{app.sync.color.toUpperCase()}</span
        >
      </div>
      <div class="mt-2 flex items-center gap-1.5">
        {#each SYNC_COLORS as c}
          <Button
            variant="outline"
            class={app.sync.enabled && app.sync.color.toLowerCase() === c
              ? "grid size-5 shrink-0 place-items-center rounded-full border-primary bg-background p-0 ring-1 ring-primary/40 hover:border-primary/50"
              : "grid size-5 shrink-0 place-items-center rounded-full border-border bg-background p-0 transition-colors hover:border-primary/50"}
            style="background: {c}"
            aria-label="Colour {c}"
            disabled={!devices.length}
            onclick={() => applySync({ color: c })}
          ></Button>
        {/each}
        <span class={app.sync.enabled && !isPreset ? "rounded-full ring-1 ring-primary/40" : ""}>
          <ColorPicker
            compact
            label="Custom colour"
            bind:value={app.sync.color}
            onchange={() => applySync()}
          />
        </span>
      </div>
    {/if}

    {#if app.kb || app.mouse}
      <div class="mt-3 flex items-center justify-between gap-3 text-[10px]"><span class="text-muted-foreground">Brightness</span></div>
      <div class="mt-2 grid grid-cols-[auto_minmax(0,1fr)_2rem] items-center gap-x-2 gap-y-1.5">
        {#if app.kb}
          <span class="text-[9px] text-muted-foreground">Keyboard</span>
          <Slider
            label="Keyboard brightness"
            bind:value={app.kb.brightness}
            onchange={applyBrightness}
          />
          <span class="text-[9px] text-muted-foreground">{app.kb.brightness}%</span>
        {/if}
        {#if app.mouse}
          <span class="text-[9px] text-muted-foreground">Mouse</span>
          <Slider
            label="Mouse brightness"
            bind:value={app.mouse.brightness}
            onchange={applyMouseBrightness}
          />
          <span class="text-[9px] text-muted-foreground">{app.mouse.brightness}%</span>
        {/if}
      </div>
    {/if}
  </section>

  <section class="min-w-0 pt-3">
    <div class="mb-2 flex items-center justify-between"><h3 class="text-xs font-semibold tracking-tight">Sensors</h3></div>
    <div class="grid grid-cols-2 gap-2">
      {#each readings as r (r.id)}
        <SensorCard
          name={r.name}
          source={r.source}
          value={r.value}
          unit={r.unit}
          kind={r.kind}
          max={r.max}
          history={app.history[r.id]}
        />
      {/each}
    </div>
  </section>
</aside>
