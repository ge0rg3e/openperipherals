<script lang="ts">
  import SensorCard, {type Kind} from "./SensorCard.svelte";
  import Switch from "./Switch.svelte";
  import ColorPicker from "./ColorPicker.svelte";
  import Slider from "./Slider.svelte";
  import {app, applyBrightness, applyMouseBrightness, applySync, SYNC_COLORS, SYNC_EFFECTS} from "../lib/store.svelte";

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
      out.push({id: "cpu", name: "CPU load", source: "Processor", value: s.cpu >= 0 ? `${s.cpu}` : "–", unit: "%", kind: "load", max: 100});
      out.push({id: "ram", name: "Memory", source: `${gb(s.ramTotal)} GB installed`, value: gb(s.ramUsed), unit: "GB", kind: "memory"});
      s.gpus?.forEach((g, i) => {
        out.push({id: `gpu${i}`, name: "GPU load", source: g.name, value: `${g.usage}`, unit: "%", kind: "load", max: 100});
        out.push({id: `vram${i}`, name: "VRAM", source: `${gb(g.vramTotal)} GB`, value: gb(g.vramUsed), unit: "GB", kind: "memory"});
      });
    }
    for (const t of app.fans?.sensors ?? []) {
      if (t.temp < 0) continue;
      out.push({id: `t:${t.id}`, name: t.id.startsWith("gpu") ? "GPU temp" : `${t.name} temp`, source: t.id.startsWith("gpu") ? t.name : "Motherboard",
        value: `${t.temp}`, unit: "°C", kind: "temp"});
    }
    for (const f of app.fans?.fans ?? []) {
      if (f.rpm >= 0) out.push({id: `rpm:${f.id}`, name: f.name, source: f.source, value: `${f.rpm}`, unit: "RPM", kind: "fan"});
      else if (f.duty >= 0) out.push({id: `duty:${f.id}`, name: f.name, source: f.source, value: `${f.duty}`, unit: "%", kind: "fan", max: 100});
    }
    return out;
  });

  const devices = $derived([app.kb && "Keyboard", app.mouse && "Mouse"].filter(Boolean) as string[]);
  const usesColor = $derived(app.sync.effect === "static" || app.sync.effect === "breathing");
  const isPreset = $derived(SYNC_COLORS.includes(app.sync.color.toLowerCase()));
</script>

<aside class="sidebar">
  <section class="sync" class:on={app.sync.enabled}>
    <div class="sync-head">
      <div>
        <h3 class="panel-title">Lighting sync</h3>
        <p class="sync-sub">{app.sync.enabled ? `Synced · ${devices.join(" + ") || "no devices"}` : "Each device uses its own lighting"}</p>
      </div>
      <Switch label="Sync lighting across devices" checked={app.sync.enabled} disabled={!devices.length}
              onchange={(on) => applySync({enabled: on})} />
    </div>

    <div class="sync-row">
      <span class="sync-label">Effect</span>
      <span class="sync-value">{SYNC_EFFECTS.find(([id]) => id === app.sync.effect)?.[1]}</span>
    </div>
    <div class="sync-effects" role="radiogroup" aria-label="Sync effect">
      {#each SYNC_EFFECTS as [id, label]}
        <button class="sync-effect" class:active={app.sync.enabled && app.sync.effect === id} disabled={!devices.length}
                role="radio" aria-checked={app.sync.effect === id} aria-label={label} title={label}
                onclick={() => applySync({effect: id})}>
          <span class="effect-swatch fx-{id}" style="--fx: {app.sync.color}"><i></i><i></i><i></i><i></i></span>
        </button>
      {/each}
    </div>

    {#if usesColor}
      <div class="sync-row"><span class="sync-label">Colour</span><span class="sync-value">{app.sync.color.toUpperCase()}</span></div>
      <div class="sync-colors">
        {#each SYNC_COLORS as c}
          <button class="sync-color" class:active={app.sync.enabled && app.sync.color.toLowerCase() === c} style="background: {c}"
                  aria-label="Colour {c}" disabled={!devices.length} onclick={() => applySync({color: c})}></button>
        {/each}
        <span class="sync-custom" class:active={app.sync.enabled && !isPreset}>
          <ColorPicker compact label="Custom colour" bind:value={app.sync.color} onchange={() => applySync()} />
        </span>
      </div>
    {/if}

    {#if app.kb || app.mouse}
      <div class="sync-row"><span class="sync-label">Brightness</span></div>
      <div class="sync-brightness">
        {#if app.kb}
          <span class="sb-name">Keyboard</span>
          <Slider label="Keyboard brightness" bind:value={app.kb.brightness} onchange={applyBrightness} />
          <span class="sb-val">{app.kb.brightness}%</span>
        {/if}
        {#if app.mouse}
          <span class="sb-name">Mouse</span>
          <Slider label="Mouse brightness" bind:value={app.mouse.brightness} onchange={applyMouseBrightness} />
          <span class="sb-val">{app.mouse.brightness}%</span>
        {/if}
      </div>
    {/if}
  </section>

  <section>
    <h3 class="panel-title">Sensors</h3>
    <div class="sensors-grid">
      {#each readings as r (r.id)}
        <SensorCard name={r.name} source={r.source} value={r.value} unit={r.unit} kind={r.kind} max={r.max} history={app.history[r.id]} />
      {/each}
    </div>
  </section>
</aside>
