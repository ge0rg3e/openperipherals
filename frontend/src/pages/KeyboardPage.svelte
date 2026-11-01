<script lang="ts">
  import {Button} from "$lib/components/ui/button";
  import PageHeader from "../ui/PageHeader.svelte";
  import EffectPreview from "../ui/EffectPreview.svelte";
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
  // Software effects are only offered by Razer boards (which store their
  // lighting); other brands with the same effect ids run them in firmware.
  const appDrawn = $derived(!!kb?.stored && ["fire", "ripple", "rain", "comet"].includes(l.effect));
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
    <div class="sync-notice">
      <span>Lighting synced. Editing here turns sync off.</span>
      <Button variant="outline" onclick={() => applySync({enabled: false})}>Unsync</Button>
    </div>
  {/if}

  <div class="keyboard-preview" bind:this={stage} onclick={() => (pick = null)} role="presentation">
    <div class="keyboard-art">
    <KeyboardArt lighting={app.kb.lighting} brightness={app.kb.brightness} layout={kb.layout} media={kb.media} interactive
                 disabled={app.disabled} selected={pick?.id ?? ""} onselect={select} />
    </div>
    <p class="mt-2 text-center text-[11px] text-muted-foreground">Click a key{kb.media ? " or the dial" : ""} to disable it</p>

    {#if pick}
      <div class="absolute z-30 min-w-56 max-w-64 rounded-lg border border-border bg-popover p-2 text-popover-foreground shadow-xl" style="left: {pick.x}px; top: {pick.y}px" tabindex="-1"
           onclick={(e) => e.stopPropagation()} onkeydown={(e) => e.key === "Escape" && (pick = null)}
           role="dialog" aria-label="{pick.label} options">
        <div class="border-b border-border px-2 py-1.5 text-xs font-semibold">{pick.id === "dial" ? "Dial" : pick.label}</div>
        {#each actions as a (a.id)}
          {#if a.code === null}
            <p class="p-2 text-xs leading-relaxed text-muted-foreground">Fn is handled inside the keyboard and never reaches Windows, so it can't be disabled.</p>
          {:else}
            {@const off = a.id in app.disabled}
            <Button variant="ghost" class="flex w-full items-center justify-between gap-3 rounded-md border-0 bg-transparent px-2 py-2 text-left text-xs hover:bg-muted" onclick={() => setKeyDisabled(a.id, a.code!, !off)}>
              <span><b>{a.label}</b>{#if a.detail}<small class="block text-muted-foreground">{a.detail}</small>{/if}</span>
              <em class={off ? "font-normal not-italic text-primary" : "font-normal not-italic text-muted-foreground"}>{off ? "Disabled" : "Enabled"}</em>
            </Button>
          {/if}
        {/each}
      </div>
    {/if}
  </div>

  <div class="device-controls mt-4">
    <section class="settings-section min-w-0">
      <div class="mb-2 flex items-center justify-between gap-3">
        <h3 class="text-sm font-semibold tracking-tight">Lighting</h3>
        <span class="text-xs text-muted-foreground" title={appDrawn ? "Runs while OpenPeripherals is open; the keyboard falls back to a hardware effect when it quits." : ""}>
          {appDrawn ? "Drawn by the app" : kb.stored ? "Saved on the keyboard" : "Restored when it connects"}
        </span>
      </div>
      <div class="effect-grid" role="radiogroup" aria-label="Effect">
        {#each effects as id}
          <Button variant="outline" size="sm" class={l.effect === id
            ? "h-auto min-h-14 grid min-w-0 justify-items-center gap-1 rounded-md border-primary/60 bg-primary/10 px-1.5 py-1.5 text-[10px] leading-tight text-foreground hover:border-primary/40"
            : "h-auto min-h-14 grid min-w-0 justify-items-center gap-1 rounded-md border-border bg-background/30 px-1.5 py-1.5 text-[10px] leading-tight text-muted-foreground transition-colors hover:border-primary/40 hover:text-foreground"} role="radio" aria-checked={l.effect === id} onclick={() => set("effect", id)}>
            <EffectPreview effect={id} color={l.color} color2={l.color2} speed={l.speed} direction={l.direction} mode={l.mode} />
            <span class="max-w-full truncate">{EFFECT_NAMES[id] ?? id}</span>
          </Button>
        {/each}
      </div>

      <div class="option-grid">
        {#if usesColor}
          <div class="grid min-w-0 content-start gap-2.5">
            <span class="flex items-center justify-between gap-3 text-xs font-medium">{mode === "dual" ? "Colours" : "Colour"}</span>
            <div class="flex min-w-0 items-center gap-3">
              <ColorPicker label="Colour" bind:value={l.color} onchange={applyKeyboard} />
              {#if mode === "dual"}
                <ColorPicker label="Second colour" bind:value={l.color2} onchange={applyKeyboard} />
              {/if}
            </div>
          </div>
        {/if}
        {#if hasModes}
          <div class="grid min-w-0 content-start gap-2.5">
            <span class="flex items-center justify-between gap-3 text-xs font-medium">Mode</span>
            <div class="inline-flex w-fit max-w-full flex-wrap gap-1 rounded-lg border border-border bg-muted/60 p-1">
              {#each ["single", "dual", "random"] as m}
                <Button variant="ghost" size="sm" class={l.mode === m ? "min-h-7 rounded-md bg-background px-3 text-xs capitalize text-foreground shadow-sm" : "min-h-7 rounded-md bg-transparent px-3 text-xs capitalize text-muted-foreground hover:bg-background/70 hover:text-foreground"} onclick={() => set("mode", m)}>{m}</Button>
              {/each}
            </div>
          </div>
        {/if}
        {#if directional}
          <div class="grid min-w-0 content-start gap-2.5">
            <span class="flex items-center justify-between gap-3 text-xs font-medium">Direction</span>
            <div class="inline-flex w-fit max-w-full flex-wrap gap-1 rounded-lg border border-border bg-muted/60 p-1">
              {#each ["left", "right"] as d}
                <Button variant="ghost" size="sm" class={l.direction === d ? "min-h-7 rounded-md bg-background px-3 text-xs capitalize text-foreground shadow-sm" : "min-h-7 rounded-md bg-transparent px-3 text-xs capitalize text-muted-foreground hover:bg-background/70 hover:text-foreground"} onclick={() => set("direction", d)}>{d}</Button>
              {/each}
            </div>
          </div>
        {/if}
        {#if maxSpeed}
          <div class="grid min-w-0 content-start gap-2.5">
            <span class="flex items-center justify-between gap-3 text-xs font-medium">Speed <b>{Math.min(l.speed, maxSpeed)}</b></span>
            <Slider label="Speed" min={1} max={maxSpeed} bind:value={l.speed} onchange={applyKeyboard} />
          </div>
        {/if}
        {#if kb.hasBrightness}
          <div class="grid min-w-0 content-start gap-2.5">
            <span class="flex items-center justify-between gap-3 text-xs font-medium">Brightness <b>{app.kb.brightness}%</b></span>
            <Slider label="Brightness" bind:value={app.kb.brightness} onchange={applyBrightness} />
          </div>
        {/if}
        {#if kb.gameMode != null}
          <div class="flex items-center justify-between gap-3" title="Disables the Windows key; saved on the keyboard">
            <span class="text-xs font-medium">Game mode</span>
            <Switch label="Game mode" checked={kb.gameMode} onchange={(on) => setKeyboardOption("gameMode", on)} />
          </div>
        {/if}
        {#if kb.macroLeds != null}
          <div class="flex items-center justify-between gap-3">
            <span class="text-xs font-medium">Macro key lights</span>
            <Switch label="Macro key lights" checked={kb.macroLeds} onchange={(on) => setKeyboardOption("macroLeds", on)} />
          </div>
        {/if}
      </div>
    </section>

    <section class="settings-section min-w-0">
      <div class="mb-2 flex items-center justify-between gap-3">
        <h3 class="text-sm font-semibold tracking-tight">Disabled keys</h3>
        <span class="text-xs text-muted-foreground">{disabledIds.length || "None"}</span>
      </div>
      {#if disabledIds.length}
        <div class="flex flex-wrap gap-2">
          {#each disabledIds as id (id)}
            <Button variant="outline" class="inline-flex items-center gap-2 rounded-full border-border bg-muted px-3 py-1.5 text-xs hover:border-primary/50" title="Enable again" onclick={() => setKeyDisabled(id, app.disabled[id], false)}>
              {LABELS[id] ?? id}<span class="text-muted-foreground" aria-hidden="true">×</span>
            </Button>
          {/each}
        </div>
      {:else}
        <p class="text-xs leading-relaxed text-muted-foreground">Click a key{kb.media ? " or the dial" : ""} on the keyboard above to disable it.</p>
      {/if}
      <p class="mt-3 text-xs leading-relaxed text-muted-foreground">
        {kb.media ? "The dial and media keys are" : "Media keys are"} blocked only on this keyboard. {disabledIds.some((id) => !isMediaKey(id)) ? "Regular keys are blocked on every keyboard, since Windows doesn't say which keyboard they came from. " : ""}Works while the app runs, including from the tray.
      </p>
    </section>
  </div>
{:else}
  <PageHeader back title="Keyboard" subtitle="Not connected" />
{/if}
