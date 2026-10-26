<script lang="ts">
  import PageHeader from "../ui/PageHeader.svelte";
  import KeyboardArt from "../art/KeyboardArt.svelte";
  import Slider from "../ui/Slider.svelte";
  import Switch from "../ui/Switch.svelte";
  import ColorPicker from "../ui/ColorPicker.svelte";
  import {app, applyBrightness, applyKeyboard, applySync, COLOR_EFFECTS, EFFECT_NAMES, setKeyboardOption, setKeyDisabled} from "../lib/store.svelte";
  import {CAPS, DIAL_ACTIONS, isMediaKey, keyCode} from "../lib/layout";

  const kb = $derived(app.status?.keyboard);
  const l = $derived(app.kb!.lighting);
  // What this keyboard can do; the page only offers that.
  const effects = $derived(kb?.effects ?? []);
  const hasModes = $derived(!!kb?.modes && (l.effect === "breathing" || l.effect === "starlight"));
  const mode = $derived(hasModes ? l.mode : "single");
  const usesColor = $derived(COLOR_EFFECTS.includes(l.effect) && mode !== "random");
  const maxSpeed = $derived(kb?.speeds?.[l.effect] ?? 0);
  const directional = $derived(!!kb?.directional?.includes(l.effect));
  const subtitle = $derived([
    kb?.firmware && `Firmware ${kb.firmware}`,
    kb?.serial,
    kb && kb.battery >= 0 && `Battery ${kb.battery}%${kb.charging ? " · charging" : ""}`,
  ].filter(Boolean).join(" · "));

  function set<K extends keyof typeof l>(key: K, value: (typeof l)[K]) {
    l[key] = value;
    applyKeyboard();
  }

  // Key picker popover.
  let stage = $state<HTMLDivElement>();
  let pick = $state<{id: string; label: string; x: number; y: number} | null>(null);

  function select(id: string, label: string, anchor: DOMRect) {
    const r = stage!.getBoundingClientRect();
    pick = {id, label, x: Math.min(anchor.left - r.left + anchor.width / 2, r.width - 130), y: anchor.bottom - r.top + 8};
  }

  interface Action {
    id: string;
    label: string;
    detail: string;
    code: number | null;
  }
  const actions = $derived.by((): Action[] => {
    if (!pick) return [];
    if (pick.id === "dial") return DIAL_ACTIONS.map((a) => ({...a}));
    return [{id: pick.id, label: "Key press", detail: pick.label, code: keyCode(pick.id)}];
  });

  const LABELS: Record<string, string> = Object.fromEntries([
    ...CAPS.map((c) => [c.id, c.label || c.id]),
    ...DIAL_ACTIONS.map((a) => [a.id, `Dial: ${a.label.toLowerCase()}`]),
  ]);
  const disabledIds = $derived(Object.keys(app.disabled).sort());
</script>

<svelte:window onkeydown={(e) => e.key === "Escape" && (pick = null)} />

{#if app.kb && kb}
  <PageHeader back title={kb.name} subtitle={subtitle} />

  {#if app.sync.enabled}
    <div class="sync-banner">
      <span><b>Lighting is synced</b> across devices. Changing it here turns sync off.</span>
      <button class="btn" onclick={() => applySync({enabled: false})}>Unsync</button>
    </div>
  {/if}

  <div class="stage-panel kb-stage" bind:this={stage} onclick={() => (pick = null)} role="presentation">
    <KeyboardArt lighting={app.kb.lighting} brightness={app.kb.brightness} layout={kb.layout} media={kb.media} interactive
                 disabled={app.disabled} selected={pick?.id ?? ""} onselect={select} />
    <p class="stage-hint">Click a key{kb.media ? " or the dial" : ""} to disable it</p>

    {#if pick}
      <div class="popover" style="left: {pick.x}px; top: {pick.y}px" tabindex="-1"
           onclick={(e) => e.stopPropagation()} onkeydown={(e) => e.key === "Escape" && (pick = null)}
           role="dialog" aria-label="{pick.label} options">
        <div class="popover-title">{pick.id === "dial" ? "Dial" : pick.label}</div>
        {#each actions as a (a.id)}
          {#if a.code === null}
            <p class="popover-note">Fn is handled inside the keyboard and never reaches Windows, so it can't be disabled.</p>
          {:else}
            {@const off = a.id in app.disabled}
            <button class="popover-row" class:off onclick={() => setKeyDisabled(a.id, a.code!, !off)}>
              <span><b>{a.label}</b>{#if a.detail}<small>{a.detail}</small>{/if}</span>
              <em>{off ? "Disabled" : "Enabled"}</em>
            </button>
          {/if}
        {/each}
      </div>
    {/if}
  </div>

  <div class="split">
    <section class="panel">
      <div class="panel-head">
        <h3 class="panel-title">Lighting</h3>
        <span class="panel-note">{kb.stored ? "Saved on the keyboard" : "Restored when it connects"}</span>
      </div>
      <div class="effect-row" role="radiogroup" aria-label="Effect">
        {#each effects as id}
          <button class="effect" class:active={l.effect === id} role="radio" aria-checked={l.effect === id} onclick={() => set("effect", id)}>
            <span class="effect-swatch fx-{id === "wheel" ? "wave" : id}" style="--fx: {l.color}"><i></i><i></i><i></i><i></i></span>
            <span class="effect-name">{EFFECT_NAMES[id] ?? id}</span>
          </button>
        {/each}
      </div>

      <div class="settings-grid">
        {#if usesColor}
          <div class="setting">
            <span class="label">{mode === "dual" ? "Colours" : "Colour"}</span>
            <div class="row">
              <ColorPicker label="Colour" bind:value={l.color} onchange={applyKeyboard} />
              {#if mode === "dual"}
                <ColorPicker label="Second colour" bind:value={l.color2} onchange={applyKeyboard} />
              {/if}
            </div>
          </div>
        {/if}
        {#if hasModes}
          <div class="setting">
            <span class="label">Mode</span>
            <div class="seg">
              {#each ["single", "dual", "random"] as m}
                <button class:active={l.mode === m} onclick={() => set("mode", m)}>{m}</button>
              {/each}
            </div>
          </div>
        {/if}
        {#if directional}
          <div class="setting">
            <span class="label">Direction</span>
            <div class="seg">
              {#each ["left", "right"] as d}
                <button class:active={l.direction === d} onclick={() => set("direction", d)}>{d}</button>
              {/each}
            </div>
          </div>
        {/if}
        {#if maxSpeed}
          <div class="setting">
            <span class="label">Speed <b>{Math.min(l.speed, maxSpeed)}</b></span>
            <Slider label="Speed" min={1} max={maxSpeed} bind:value={l.speed} onchange={applyKeyboard} />
          </div>
        {/if}
        {#if kb.hasBrightness}
          <div class="setting">
            <span class="label">Brightness <b>{app.kb.brightness}%</b></span>
            <Slider label="Brightness" bind:value={app.kb.brightness} onchange={applyBrightness} />
          </div>
        {/if}
        {#if kb.gameMode != null}
          <div class="setting" title="Disables the Windows key; saved on the keyboard">
            <span class="label">Game mode</span>
            <Switch label="Game mode" checked={kb.gameMode} onchange={(on) => setKeyboardOption("gameMode", on)} />
          </div>
        {/if}
        {#if kb.macroLeds != null}
          <div class="setting">
            <span class="label">Macro key lights</span>
            <Switch label="Macro key lights" checked={kb.macroLeds} onchange={(on) => setKeyboardOption("macroLeds", on)} />
          </div>
        {/if}
      </div>
    </section>

    <section class="panel">
      <div class="panel-head">
        <h3 class="panel-title">Disabled keys</h3>
        <span class="panel-note">{disabledIds.length || "None"}</span>
      </div>
      {#if disabledIds.length}
        <div class="chips">
          {#each disabledIds as id (id)}
            <button class="chip" title="Enable again" onclick={() => setKeyDisabled(id, app.disabled[id], false)}>
              {LABELS[id] ?? id}<span aria-hidden="true">×</span>
            </button>
          {/each}
        </div>
      {:else}
        <p class="meta">Click a key{kb.media ? " or the dial" : ""} on the keyboard above to disable it.</p>
      {/if}
      <p class="hint">
        {kb.media ? "The dial and media keys are" : "Media keys are"} blocked only on this keyboard. {disabledIds.some((id) => !isMediaKey(id)) ? "Regular keys are blocked on every keyboard, since Windows doesn't say which keyboard they came from. " : ""}Works while the app runs, including from the tray.
      </p>
    </section>
  </div>
{:else}
  <PageHeader back title="Keyboard" subtitle="Not connected" />
{/if}
