<script lang="ts">
  // Mode, speed and curve controls for one fan. Used by the fan cards and by
  // the preset editor, so a preset can store the same values a fan can run.
  import {Button} from "$lib/components/ui/button";
  import * as NativeSelect from "$lib/components/ui/native-select";
  import Slider from "./ui/Slider.svelte";
  import type {FanConfig, TempSensor} from "../bindings/openperipherals";

  let {config = $bindable(), sensors, minDuty, gpu = false, onchange}:
    {config: FanConfig; sensors: TempSensor[]; minDuty: number; gpu?: boolean; onchange?: (immediate: boolean) => void} = $props();

  // Curve editor geometry: x = 20-100 °C, y = 0-100 %.
  const W = 300, H = 150, PAD = 22;
  const T0 = 20, T1 = 100;
  const x = (t: number) => PAD + ((t - T0) / (T1 - T0)) * (W - PAD * 2);
  const y = (p: number) => H - PAD - (p / 100) * (H - PAD * 2);
  const points = $derived(config.points ?? []);
  const path = $derived(points.map((p, i) => `${i ? "L" : "M"}${x(p[0])},${y(p[1])}`).join(" "));
  const sensorTemp = $derived(sensors.find((s) => s.id === config.sensor)?.temp ?? -1);

  function mode(m: string) {
    config.mode = m;
    onchange?.(true);
  }

  let svg = $state<SVGSVGElement>();
  let dragging = -1;
  function drag(e: PointerEvent) {
    if (dragging < 0 || !svg) return;
    const r = svg.getBoundingClientRect();
    const px = ((e.clientX - r.left) / r.width) * W;
    const py = ((e.clientY - r.top) / r.height) * H;
    const pts = config.points!;
    // Keep points ordered by temperature.
    const lo = dragging > 0 ? pts[dragging - 1][0] + 1 : T0;
    const hi = dragging < pts.length - 1 ? pts[dragging + 1][0] - 1 : T1;
    const t = Math.round(T0 + ((px - PAD) / (W - PAD * 2)) * (T1 - T0));
    const p = Math.round(((H - PAD - py) / (H - PAD * 2)) * 100);
    pts[dragging] = [Math.min(Math.max(t, lo), hi), Math.min(Math.max(p, minDuty), 100)];
  }
  function endDrag() {
    if (dragging < 0) return;
    dragging = -1;
    onchange?.(true);
  }
</script>

<svelte:window onpointermove={drag} onpointerup={endDrag} />

<div class={config.mode === "curve" ? "fan-editor has-curve" : "fan-editor"}>
  <div class="grid content-start gap-2.5">
    <div class="inline-flex w-fit max-w-full flex-wrap gap-0.5 rounded-md border border-border bg-muted/60 p-0.5">
      {#each [["auto", "Auto"], ["fixed", "Fixed"], ["curve", "Curve"]] as [id, label]}
        <Button variant="ghost" size="sm" class={config.mode === id
          ? "min-h-6 rounded bg-background px-2.5 text-[11px] capitalize text-foreground shadow-sm"
          : "min-h-6 rounded bg-transparent px-2.5 text-[11px] capitalize text-muted-foreground hover:bg-background/70 hover:text-foreground"} onclick={() => mode(id)}>{label}</Button>
      {/each}
    </div>

    {#if config.mode === "auto"}
      <p class="text-[11px] leading-relaxed text-muted-foreground">Controlled by the {gpu ? "graphics driver" : "BIOS"}.</p>
    {:else if config.mode === "fixed"}
      <div class="grid gap-1.5">
        <span class="flex items-center justify-between gap-3 text-xs font-medium text-foreground">Speed <b>{Math.max(config.fixed, minDuty)}%</b></span>
        <Slider label="Fixed speed" min={minDuty} max={100} bind:value={config.fixed} oninput={() => onchange?.(false)} />
      </div>
    {:else}
      <div class="grid gap-1.5">
        <span class="text-xs font-medium text-foreground">Follow</span>
        <NativeSelect.Root bind:value={config.sensor} onchange={() => onchange?.(true)}>
          {#each sensors as s}
            <NativeSelect.Option value={s.id}>{s.name}{s.temp >= 0 ? ` (${s.temp} °C)` : ""}</NativeSelect.Option>
          {/each}
        </NativeSelect.Root>
      </div>
    {/if}
  </div>

  {#if config.mode === "curve"}
    <svg bind:this={svg} class="h-full min-h-36 w-full overflow-visible" viewBox="0 0 {W} {H}" role="img" aria-label="Fan curve">
    {#each [0, 50, 100] as p}
      <line class="stroke-border" style="stroke-width:1" x1={PAD} x2={W - PAD} y1={y(p)} y2={y(p)} />
      <text class="fill-muted-foreground" style="font-size:9px" x={PAD - 4} y={y(p) + 3} text-anchor="end">{p}</text>
    {/each}
    {#each [20, 40, 60, 80, 100] as t}
      <text class="fill-muted-foreground" style="font-size:9px" x={x(t)} y={H - 6} text-anchor="middle">{t}°</text>
    {/each}
    {#if sensorTemp >= T0}
      <line class="stroke-amber-400" style="stroke-dasharray:3 3" x1={x(sensorTemp)} x2={x(sensorTemp)} y1={PAD} y2={H - PAD} />
    {/if}
    <path d={path} class="fill-none stroke-primary" style="stroke-width:2" />
    {#each points as p, i}
      <circle cx={x(p[0])} cy={y(p[1])} r="6" class="fill-primary stroke-background" style="stroke-width:2" role="slider" tabindex="-1"
              aria-valuenow={p[1]} aria-label="Point {i + 1}"
              onpointerdown={(e) => { dragging = i; e.preventDefault(); }}>
        <title>{p[0]} °C → {p[1]}%</title>
      </circle>
    {/each}
    </svg>
  {/if}
</div>
