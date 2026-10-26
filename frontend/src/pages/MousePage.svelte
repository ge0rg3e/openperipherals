<script lang="ts">
  import PageHeader from "../ui/PageHeader.svelte";
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
    <div class="sync-banner">
      <span><b>Lighting is synced</b> across devices. Changing colours here turns sync off.</span>
      <button class="btn" onclick={() => applySync({enabled: false})}>Unsync</button>
    </div>
  {/if}

  {#if ms.kind === "corsair"}
    <div class="mouse-page">
      <div class="stage-panel mouse-stage">
        <MouseArt colors={mouseZones()} sync={app.sync} brightness={m.brightness} interactive />
        <div class="dpi-readout">
          <b>{m.stages[m.current]?.dpi.toLocaleString()}</b>
          <span>DPI · {m.current === 0 ? "Sniper" : `Stage ${m.current}`}</span>
        </div>
      </div>

      <div class="panels-col">
        <section class="panel">
          <h3 class="panel-title">Lighting</h3>
          <div class="zones">
            {#each ms.zones ?? [] as name, i}
              {#if i !== DPI_ZONE}
                <div class="zone">
                  <span class="zone-name">{name}</span>
                  <ColorPicker label={name} bind:value={m.colors[i]} oninput={previewMouse} onchange={saveMouseColors} />
                </div>
              {/if}
            {/each}
          </div>
          <div class="field">
            <span class="label">Brightness <b>{m.brightness}%</b></span>
            <Slider label="Mouse brightness" bind:value={m.brightness} onchange={applyMouseBrightness} />
          </div>
          <p class="hint">The DPI indicator shows the colour of the active stage, at full brightness.</p>
        </section>

        <section class="panel">
          <h3 class="panel-title">DPI stages</h3>
          <div class="stages">
            {#each m.stages as s, i}
              <div class="stage" class:off={!s.enabled} class:current={i === m.current}>
                <button class="radio" title="Make active" aria-label="Make stage {i} active" onclick={() => activate(i)}></button>
                <Switch label="Enable stage {i}" checked={s.enabled} disabled={i === m.current} onchange={() => toggle(i)} />
                <span class="idx">{i === 0 ? "Sniper" : `Stage ${i}`}</span>
                <Slider label="Stage {i} DPI" min={100} max={ms.maxDpi} step={50} bind:value={s.dpi} onchange={() => applyDpi()} dim={!s.enabled} />
                <input class="num" type="number" min="100" max={ms.maxDpi} step="50" bind:value={s.dpi} onchange={() => applyDpi()} aria-label="Stage {i} DPI value" />
                <ColorPicker compact label="Stage {i} indicator colour" bind:value={s.color} onchange={() => applyDpi(0)} />
              </div>
            {/each}
          </div>
          <p class="hint">Saved to the mouse's onboard memory. The DPI button cycles through enabled stages.</p>
        </section>
      </div>
    </div>
  {:else}
    <div class="mouse-page">
      <div class="stage-panel mouse-stage">
        <MouseArt colors={mouseArtColors()} sync={mouseArtSync()} interactive />
        <div class="dpi-readout">
          <b>{ms.dpi.toLocaleString()}</b>
          <span>DPI{ms.polling ? ` · ${ms.polling.toLocaleString()} Hz` : ""}{ms.battery >= 0 ? ` · ${ms.battery}%${ms.charging ? " charging" : ""}` : ""}</span>
        </div>
      </div>

      <div class="panels-col">
        <section class="panel">
          <div class="panel-head">
            <h3 class="panel-title">Sensor</h3>
            <span class="panel-note">Saved on the mouse</span>
          </div>
          <div class="settings-grid">
            <div class="setting">
              <span class="label">DPI <b>{m.dpi.toLocaleString()}</b></span>
              <div class="row">
                <Slider label="DPI" min={ms.minDpi} max={ms.maxDpi} step={ms.dpiStep} bind:value={m.dpi} onchange={() => applySensorDpi()} />
                <input class="num" type="number" min={ms.minDpi} max={ms.maxDpi} step={ms.dpiStep} bind:value={m.dpi}
                       onchange={() => applySensorDpi(0)} aria-label="DPI value" />
              </div>
            </div>
            {#if ms.pollingRates?.length}
              <div class="setting">
                <span class="label">Polling rate</span>
                <div class="seg">
                  {#each ms.pollingRates as hz}
                    <button class:active={ms.polling === hz} onclick={() => setMouseOption("polling", hz)}>
                      {hz >= 1000 ? `${hz / 1000}K` : hz}
                    </button>
                  {/each}
                </div>
              </div>
            {/if}
            {#if ms.liftOff >= 0 && ms.liftOffs?.length}
              <div class="setting">
                <span class="label">Lift-off distance</span>
                <div class="seg">
                  {#each ms.liftOffs as name, i}
                    <button class:active={ms.liftOff === i} onclick={() => setMouseOption("liftOff", i)}>{name}</button>
                  {/each}
                </div>
              </div>
            {/if}
          </div>
        </section>

        {#if ms.effects?.length}
          <section class="panel">
            <div class="panel-head">
              <h3 class="panel-title">Lighting</h3>
              <span class="panel-note">Saved on the mouse</span>
            </div>
            <div class="effect-row" role="radiogroup" aria-label="Mouse effect">
              {#each ms.effects as id}
                <button class="effect" class:active={l.effect === id} role="radio" aria-checked={l.effect === id} onclick={() => setLight("effect", id)}>
                  <span class="effect-swatch fx-{id}" style="--fx: {l.color}"><i></i><i></i><i></i><i></i></span>
                  <span class="effect-name">{EFFECT_NAMES[id] ?? id}</span>
                </button>
              {/each}
            </div>
            <div class="settings-grid">
              {#if COLOR_EFFECTS.includes(l.effect) && lightMode !== "random"}
                <div class="setting">
                  <span class="label">{lightMode === "dual" ? "Colours" : "Colour"}</span>
                  <div class="row">
                    <ColorPicker label="Mouse colour" bind:value={l.color} onchange={applyMouseLighting} />
                    {#if lightMode === "dual"}
                      <ColorPicker label="Second mouse colour" bind:value={l.color2} onchange={applyMouseLighting} />
                    {/if}
                  </div>
                </div>
              {/if}
              {#if l.effect === "breathing"}
                <div class="setting">
                  <span class="label">Mode</span>
                  <div class="seg">
                    {#each ["single", "dual", "random"] as md}
                      <button class:active={l.mode === md} onclick={() => setLight("mode", md)}>{md}</button>
                    {/each}
                  </div>
                </div>
              {/if}
              {#if l.effect === "reactive"}
                <div class="setting">
                  <span class="label">Speed <b>{Math.min(l.speed, 4)}</b></span>
                  <Slider label="Mouse effect speed" min={1} max={4} bind:value={l.speed} onchange={applyMouseLighting} />
                </div>
              {/if}
            </div>
            <p class="hint">The mouse can't report its lighting, so this shows the last effect set here.</p>
          </section>
        {/if}

        {#if (ms.sleep >= 0 && ms.sleepOptions?.length) || ms.debounce >= 0 || processing.length}
          <section class="panel">
            <h3 class="panel-title">Advanced</h3>
            <div class="settings-grid">
              {#if ms.sleep >= 0 && ms.sleepOptions?.length}
                <div class="setting">
                  <span class="label">Auto sleep after <b>{sleepLabel(ms.sleepOptions[sleepIdx])}</b></span>
                  <Slider label="Auto sleep" min={0} max={ms.sleepOptions.length - 1} bind:value={sleepIdx}
                          onchange={() => setMouseOption("sleep", ms.sleepOptions![sleepIdx])} />
                </div>
              {/if}
              {#if ms.debounce >= 0}
                <div class="setting">
                  <span class="label">Debounce <b>{debounce} ms</b></span>
                  <Slider label="Debounce time" min={0} max={ms.debounceMax} bind:value={debounce}
                          onchange={() => setMouseOption("debounce", debounce)} />
                </div>
              {/if}
              {#each processing as [id, label, on] (id)}
                <div class="setting">
                  <span class="label">{label}</span>
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
