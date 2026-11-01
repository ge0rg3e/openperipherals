<script lang="ts">
  import {Button} from "$lib/components/ui/button";
  import {Input} from "$lib/components/ui/input";
  import PageHeader from "../ui/PageHeader.svelte";
  import EffectPreview from "../ui/EffectPreview.svelte";
  import MouseArt from "../art/MouseArt.svelte";
  import Slider from "../ui/Slider.svelte";
  import Switch from "../ui/Switch.svelte";
  import ColorPicker from "../ui/ColorPicker.svelte";
  import {
    app, applyDpi, applyMouseBrightness, applyMouseLighting, applySensorDpi, applySync, COLOR_EFFECTS, DPI_ZONE,
    EFFECT_NAMES, mouseArtColors, mouseArtSync, mouseZones, previewMouse, saveMouseColors, setMouseOption,
  } from "../lib/store.svelte";

  const ms = $derived(app.status?.mouse);
  const m = $derived(app.mouse!);

  function activate(i: number) {
    m.current = i;
    m.stages[i].enabled = true;
    applyDpi(0);
  }

  function toggle(i: number) {
    if (i === m.current) return; // the active stage must stay enabled
    m.stages[i].enabled = !m.stages[i].enabled;
    applyDpi(0);
  }

  // Razer and Pulsar mice.
  const l = $derived(m.lighting);
  const lightMode = $derived(l.effect === "breathing" ? l.mode : "single");
  const subtitle = $derived(ms ? [
    ms.firmware && `Firmware ${ms.firmware}`,
    `up to ${ms.maxDpi.toLocaleString()} DPI`,
    ms.kind === "sensor" && (ms.wireless ? "Wireless" : "Wired"),
  ].filter(Boolean).join(" · ") : "");

  function setLight<K extends keyof typeof l>(key: K, value: (typeof l)[K]) {
    l[key] = value;
    applyMouseLighting();
  }

  const sleepLabel = (s: number) => (s < 60 ? `${s} s` : `${s / 60} min`);

  // Slider positions, kept in step with what the mouse reports.
  let sleepIdx = $state(0);
  let debounce = $state(0);
  $effect(() => {
    sleepIdx = Math.max(ms?.sleepOptions?.indexOf(ms.sleep) ?? 0, 0);
    debounce = Math.max(ms?.debounce ?? 0, 0);
  });
  const processing = $derived(ms ? ([
    ["motionSync", "Motion Sync", ms.motionSync],
    ["angleSnap", "Angle snapping", ms.angleSnap],
    ["ripple", "Ripple control", ms.ripple],
  ] as const).filter(([, , v]) => v != null) : []);
</script>

{#if app.mouse && ms}
  <PageHeader back title={ms.name} subtitle={subtitle} />

  {#if app.sync.enabled && (ms.kind === "corsair" || ms.effects?.length)}
    <div class="sync-notice">
      <span>Lighting synced. Editing here turns sync off.</span>
      <Button variant="outline" onclick={() => applySync({enabled: false})}>Unsync</Button>
    </div>
  {/if}

  {#if ms.kind === "corsair"}
    <div class="mouse-layout">
      <div class="mouse-preview">
        <MouseArt colors={mouseZones()} sync={app.sync} brightness={m.brightness} interactive />
        <div class="mouse-reading">
          <b class="text-base tabular-nums">{m.stages[m.current]?.dpi.toLocaleString()}</b>
          <span class="text-[10px] text-muted-foreground">DPI · {m.current === 0 ? "Sniper" : `Stage ${m.current}`}</span>
        </div>
      </div>

      <div class="device-controls">
        <section class="settings-section grid content-start gap-3">
          <h3 class="text-sm font-semibold tracking-tight">Lighting</h3>
          <div class="zone-grid">
            {#each ms.zones ?? [] as name, i}
              {#if i !== DPI_ZONE}
                <div class="zone-control">
                  <span class="min-w-20 text-xs font-medium">{name}</span>
                  <ColorPicker label={name} bind:value={m.colors[i]} oninput={previewMouse} onchange={saveMouseColors} />
                </div>
              {/if}
            {/each}
          </div>
          <div class="grid gap-2">
            <span class="flex items-center justify-between gap-3 text-xs font-medium">Brightness <b>{m.brightness}%</b></span>
            <Slider label="Mouse brightness" bind:value={m.brightness} onchange={applyMouseBrightness} />
          </div>
          <p class="text-xs leading-relaxed text-muted-foreground">The DPI indicator shows the colour of the active stage, at full brightness.</p>
        </section>

        <section class="settings-section grid content-start gap-3">
          <h3 class="text-sm font-semibold tracking-tight">DPI stages</h3>
          <div class="grid gap-2">
            {#each m.stages as s, i}
              <div class={i === m.current
                ? `dpi-stage min-w-0 border-b border-primary/40 bg-primary/5 px-1.5 py-2 ${s.enabled ? "" : "opacity-55"}`
                : `dpi-stage min-w-0 border-b border-border px-1.5 py-2 ${s.enabled ? "" : "opacity-55"}`}>
                <Button variant="outline" size="icon-xs" class="size-4 rounded-full border border-muted-foreground/50 bg-background p-0" title="Make active" aria-pressed={i === m.current} aria-label="Make stage {i} active" onclick={() => activate(i)}></Button>
                <Switch label="Enable stage {i}" checked={s.enabled} disabled={i === m.current} onchange={() => toggle(i)} />
                <span class="min-w-20 text-xs font-medium">{i === 0 ? "Sniper" : `Stage ${i}`}</span>
                <Slider label="Stage {i} DPI" min={100} max={ms.maxDpi} step={50} bind:value={s.dpi} onchange={() => applyDpi()} dim={!s.enabled} />
                <Input class="h-7 w-20 rounded-md border border-input bg-background px-2 text-right text-xs tabular-nums outline-none focus-visible:ring-2 focus-visible:ring-ring" type="number" min="100" max={ms.maxDpi} step="50" bind:value={s.dpi} onchange={() => applyDpi()} aria-label="Stage {i} DPI value" />
                <ColorPicker compact label="Stage {i} indicator colour" bind:value={s.color} onchange={() => applyDpi(0)} />
              </div>
            {/each}
          </div>
          <p class="text-xs leading-relaxed text-muted-foreground">Saved to the mouse's onboard memory. The DPI button cycles through enabled stages.</p>
        </section>
      </div>
    </div>
  {:else}
    <div class="mouse-layout">
      <div class="mouse-preview">
        <MouseArt colors={mouseArtColors()} sync={mouseArtSync()} interactive />
        <div class="mouse-reading">
          <b class="text-base tabular-nums">{ms.dpi.toLocaleString()}</b>
          <span class="text-[10px] text-muted-foreground">DPI{ms.polling ? ` · ${ms.polling.toLocaleString()} Hz` : ""}{ms.battery >= 0 ? ` · ${ms.battery}%${ms.charging ? " charging" : ""}` : ""}</span>
        </div>
      </div>

      <div class="device-controls">
        <section class="settings-section grid content-start gap-3">
          <div class="flex items-center justify-between gap-3">
            <h3 class="text-sm font-semibold tracking-tight">Sensor</h3>
            <span class="text-xs text-muted-foreground">Saved on the mouse</span>
          </div>
          <div class="option-grid">
            <div class="grid min-w-0 content-start gap-2.5">
              <span class="flex items-center justify-between gap-3 text-xs font-medium">DPI <b>{m.dpi.toLocaleString()}</b></span>
              <div class="flex min-w-0 items-center gap-3">
                <Slider label="DPI" min={ms.minDpi} max={ms.maxDpi} step={ms.dpiStep} bind:value={m.dpi} onchange={() => applySensorDpi()} />
                <Input class="h-8 w-24 text-right tabular-nums" type="number" min={ms.minDpi} max={ms.maxDpi} step={ms.dpiStep} bind:value={m.dpi}
                       onchange={() => applySensorDpi(0)} aria-label="DPI value" />
              </div>
            </div>
            {#if ms.pollingRates?.length}
              <div class="grid min-w-0 content-start gap-2.5">
                <span class="flex items-center justify-between gap-3 text-xs font-medium">Polling rate</span>
                <div class="inline-flex w-fit max-w-full flex-wrap gap-1">
                  {#each ms.pollingRates as hz}
                    <Button variant="ghost" size="sm" class={ms.polling === hz ? "min-h-7 rounded-md bg-background px-3 text-xs text-foreground shadow-sm" : "min-h-7 rounded-md bg-transparent px-3 text-xs text-muted-foreground hover:bg-background/70 hover:text-foreground"} onclick={() => setMouseOption("polling", hz)}>
                      {hz >= 1000 ? `${hz / 1000}K` : hz}
                    </Button>
                  {/each}
                </div>
              </div>
            {/if}
            {#if ms.liftOff >= 0 && ms.liftOffs?.length}
              <div class="grid min-w-0 content-start gap-2.5">
                <span class="flex items-center justify-between gap-3 text-xs font-medium">Lift-off distance</span>
                <div class="inline-flex w-fit max-w-full flex-wrap gap-1">
                  {#each ms.liftOffs as name, i}
                    <Button variant="ghost" size="sm" class={ms.liftOff === i ? "min-h-7 rounded-md bg-background px-3 text-xs text-foreground shadow-sm" : "min-h-7 rounded-md bg-transparent px-3 text-xs text-muted-foreground hover:bg-background/70 hover:text-foreground"} onclick={() => setMouseOption("liftOff", i)}>{name}</Button>
                  {/each}
                </div>
              </div>
            {/if}
          </div>
        </section>

        {#if ms.effects?.length}
          <section class="settings-section grid content-start gap-3">
            <div class="flex items-center justify-between gap-3">
              <h3 class="text-sm font-semibold tracking-tight">Lighting</h3>
              <span class="text-xs text-muted-foreground">Saved on the mouse</span>
            </div>
            <div class="effect-grid" role="radiogroup" aria-label="Mouse effect">
              {#each ms.effects as id}
                <Button variant="outline" size="sm" class={l.effect === id
                  ? "h-auto min-h-14 grid min-w-0 justify-items-center gap-1 rounded-md border-primary/60 bg-primary/10 px-1.5 py-1.5 text-[10px] leading-tight text-foreground hover:border-primary/40"
                  : "h-auto min-h-14 grid min-w-0 justify-items-center gap-1 border-border bg-transparent px-1.5 py-1.5 text-[10px] leading-tight text-muted-foreground transition-colors hover:border-primary/40 hover:text-foreground"} role="radio" aria-checked={l.effect === id} onclick={() => setLight("effect", id)}>
                  <EffectPreview effect={id} color={l.color} color2={l.color2} speed={l.speed} direction={l.direction} mode={l.mode} />
                  <span class="max-w-full truncate">{EFFECT_NAMES[id] ?? id}</span>
                </Button>
              {/each}
            </div>
            <div class="option-grid">
              {#if COLOR_EFFECTS.includes(l.effect) && lightMode !== "random"}
                <div class="grid min-w-0 content-start gap-2.5">
                  <span class="flex items-center justify-between gap-3 text-xs font-medium">{lightMode === "dual" ? "Colours" : "Colour"}</span>
                  <div class="flex min-w-0 items-center gap-3">
                    <ColorPicker label="Mouse colour" bind:value={l.color} onchange={applyMouseLighting} />
                    {#if lightMode === "dual"}
                      <ColorPicker label="Second mouse colour" bind:value={l.color2} onchange={applyMouseLighting} />
                    {/if}
                  </div>
                </div>
              {/if}
              {#if l.effect === "breathing"}
                <div class="grid min-w-0 content-start gap-2.5">
                  <span class="flex items-center justify-between gap-3 text-xs font-medium">Mode</span>
                  <div class="inline-flex w-fit max-w-full flex-wrap gap-1">
                    {#each ["single", "dual", "random"] as md}
                      <Button variant="ghost" size="sm" class={l.mode === md ? "min-h-7 rounded-md bg-background px-3 text-xs capitalize text-foreground shadow-sm" : "min-h-7 rounded-md bg-transparent px-3 text-xs capitalize text-muted-foreground hover:bg-background/70 hover:text-foreground"} onclick={() => setLight("mode", md)}>{md}</Button>
                    {/each}
                  </div>
                </div>
              {/if}
              {#if l.effect === "reactive"}
                <div class="grid min-w-0 content-start gap-2.5">
                  <span class="flex items-center justify-between gap-3 text-xs font-medium">Speed <b>{Math.min(l.speed, 4)}</b></span>
                  <Slider label="Mouse effect speed" min={1} max={4} bind:value={l.speed} onchange={applyMouseLighting} />
                </div>
              {/if}
            </div>
            <p class="text-xs leading-relaxed text-muted-foreground">The mouse can't report its lighting, so this shows the last effect set here.</p>
          </section>
        {/if}

        {#if (ms.sleep >= 0 && ms.sleepOptions?.length) || ms.debounce >= 0 || processing.length}
          <section class="settings-section grid content-start gap-3">
            <h3 class="text-sm font-semibold tracking-tight">Advanced</h3>
            <div class="option-grid">
              {#if ms.sleep >= 0 && ms.sleepOptions?.length}
                <div class="grid min-w-0 content-start gap-2.5">
                  <span class="flex items-center justify-between gap-3 text-xs font-medium">Auto sleep after <b>{sleepLabel(ms.sleepOptions[sleepIdx])}</b></span>
                  <Slider label="Auto sleep" min={0} max={ms.sleepOptions.length - 1} bind:value={sleepIdx}
                          onchange={() => setMouseOption("sleep", ms.sleepOptions![sleepIdx])} />
                </div>
              {/if}
              {#if ms.debounce >= 0}
                <div class="grid min-w-0 content-start gap-2.5">
                  <span class="flex items-center justify-between gap-3 text-xs font-medium">Debounce <b>{debounce} ms</b></span>
                  <Slider label="Debounce time" min={0} max={ms.debounceMax} bind:value={debounce}
                          onchange={() => setMouseOption("debounce", debounce)} />
                </div>
              {/if}
              {#each processing as [id, label, on] (id)}
                <div class="flex items-center justify-between gap-3">
                  <span class="text-xs font-medium">{label}</span>
                  <Switch {label} checked={!!on} onchange={(next) => setMouseOption(id, next ? 1 : 0)} />
                </div>
              {/each}
            </div>
          </section>
        {/if}
      </div>
    </div>
  {/if}
{:else}
  <PageHeader back title="Mouse" subtitle="Not connected" />
{/if}
