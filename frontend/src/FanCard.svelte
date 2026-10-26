<script lang="ts">
  import Slider from "./ui/Slider.svelte";
  import {FanService, type Fan, type FanConfig, type TempSensor} from "../bindings/openperipherals";

  let {fan, sensors, onerror}: {fan: Fan; sensors: TempSensor[]; onerror: (e: unknown) => void} = $props();

  // svelte-ignore state_referenced_locally
  let cfg = $state<FanConfig>({...fan.config, points: (fan.config.points ?? []).map((p) => [...p])});

  // Curve editor geometry: x = 20-100 °C, y = 0-100 %.
  const W = 300, H = 150, PAD = 22;
  const T0 = 20, T1 = 100;
  const x = (t: number) => PAD + ((t - T0) / (T1 - T0)) * (W - PAD * 2);
  const y = (p: number) => H - PAD - (p / 100) * (H - PAD * 2);
  const points = $derived(cfg.points ?? []);
  const path = $derived(points.map((p, i) => `${i ? "L" : "M"}${x(p[0])},${y(p[1])}`).join(" "));
  const sensorTemp = $derived(sensors.find((s) => s.id === cfg.sensor)?.temp ?? -1);

  let saveTimer: ReturnType<typeof setTimeout> | undefined;
  function save(delay = 250) {
    clearTimeout(saveTimer);
    saveTimer = setTimeout(() => FanService.SetFanConfig(fan.id, cfg).catch(onerror), delay);
  }

  function mode(m: string) {
    cfg.mode = m;
    save(0);
  }

  let svg = $state<SVGSVGElement>();
  let dragging = -1;
  function drag(e: PointerEvent) {
    if (dragging < 0 || !svg) return;
    const r = svg.getBoundingClientRect();
    const px = ((e.clientX - r.left) / r.width) * W;
    const py = ((e.clientY - r.top) / r.height) * H;
    const pts = cfg.points!;
    // Keep points ordered by temperature.
    const lo = dragging > 0 ? pts[dragging - 1][0] + 1 : T0;
    const hi = dragging < pts.length - 1 ? pts[dragging + 1][0] - 1 : T1;
    const t = Math.round(T0 + ((px - PAD) / (W - PAD * 2)) * (T1 - T0));
    const p = Math.round(((H - PAD - py) / (H - PAD * 2)) * 100);
    pts[dragging] = [Math.min(Math.max(t, lo), hi), Math.min(Math.max(p, fan.minDuty), 100)];
  }
  function endDrag() {
    if (dragging < 0) return;
    dragging = -1;
    save(0);
  }
</script>

<svelte:window onpointermove={drag} onpointerup={endDrag} />

<section class="card fan">
  <header>
    <div>
      <h2>{fan.name}</h2>
      <p class="meta">{fan.source}</p>
    </div>
    <div class="readout">
      {#if fan.rpm >= 0}<b>{fan.rpm}</b> RPM{/if}
      {#if fan.duty >= 0}<span>{fan.duty}%</span>{/if}
    </div>
  </header>

  <div class="seg">
    {#each [["auto", "Auto"], ["fixed", "Fixed"], ["curve", "Curve"]] as [id, label]}
      <button class:active={cfg.mode === id} onclick={() => mode(id)}>{label}</button>
    {/each}
  </div>

  {#if cfg.mode === "auto"}
    <p class="hint">Controlled by the {fan.id.startsWith("gpu") ? "graphics driver" : "BIOS"}.</p>
  {:else if cfg.mode === "fixed"}
    <div class="field">
      <span class="label">Speed <b>{Math.max(cfg.fixed, fan.minDuty)}%</b></span>
      <Slider label="Fixed speed" min={fan.minDuty} max={100} bind:value={cfg.fixed} oninput={() => save()} />
    </div>
  {:else}
    <div class="field">
      <span class="label">Follow</span>
      <select bind:value={cfg.sensor} onchange={() => save(0)}>
        {#each sensors as s}
          <option value={s.id}>{s.name}{s.temp >= 0 ? ` (${s.temp} °C)` : ""}</option>
        {/each}
      </select>
    </div>
    <svg bind:this={svg} class="curve" viewBox="0 0 {W} {H}" role="img" aria-label="Fan curve">
      {#each [0, 50, 100] as p}
        <line class="grid" x1={PAD} x2={W - PAD} y1={y(p)} y2={y(p)} />
        <text class="axis" x={PAD - 4} y={y(p) + 3} text-anchor="end">{p}</text>
      {/each}
      {#each [20, 40, 60, 80, 100] as t}
        <text class="axis" x={x(t)} y={H - 6} text-anchor="middle">{t}°</text>
      {/each}
      {#if sensorTemp >= T0}
        <line class="now" x1={x(sensorTemp)} x2={x(sensorTemp)} y1={PAD} y2={H - PAD} />
      {/if}
      <path d={path} class="line" />
      {#each points as p, i}
        <circle cx={x(p[0])} cy={y(p[1])} r="6" class="pt" role="slider" tabindex="-1"
                aria-valuenow={p[1]} aria-label="Point {i + 1}"
                onpointerdown={(e) => { dragging = i; e.preventDefault(); }}>
          <title>{p[0]} °C → {p[1]}%</title>
        </circle>
      {/each}
    </svg>
  {/if}
</section>
