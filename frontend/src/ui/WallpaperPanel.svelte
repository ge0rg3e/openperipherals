<script lang="ts">
  import {System} from "@wailsio/runtime";
  import type {WallpaperStatus} from "../../bindings/openperipherals";
  import type {Config} from "../../bindings/openperipherals/internal/wallpaper/models";
  import {Button} from "$lib/components/ui/button";
  import {Input} from "$lib/components/ui/input";
  import Icon from "./Icon.svelte";
  import Switch from "./Switch.svelte";

  let {config = $bindable(), status = null, busy = false, loaded = false, loadError = false,
    ontoggle, onsettingschange, onbrowse, onretry}: {
    config: Config; status?: WallpaperStatus | null; busy?: boolean; loaded?: boolean; loadError?: boolean;
    ontoggle: () => void; onsettingschange: () => void; onbrowse: () => void; onretry: () => void;
  } = $props();

  const linux = System.IsLinux();
  const sources = [
    {kind:"local", title:"HTML file", icon:"file"},
    {kind:"video", title:"MP4 video", icon:"play"},
    {kind:"url", title:"Web page", icon:"globe"},
  ] as const;
  let drafts = $state<Record<string,string>>({});
  function choose(kind: string) {
    if (kind === config.kind) return;
    drafts[config.kind] = config.source;
    config.kind = kind;
    config.source = drafts[kind] ?? "";
    onsettingschange();
  }
  const enabled = $derived(!!status?.config.enabled);
  const canEnable = $derived(["local", "url", "video"].includes(config.kind) && !!config.source.trim());
  function updateSetting<K extends keyof Config>(key: K, value: Config[K]) {
    config[key] = value;
    onsettingschange();
  }
</script>


<div class="wallpaper-page">
  <header class="page-heading justify-between">
    <h2 class="text-base font-semibold tracking-tight">Wallpaper</h2>
    <Button variant={enabled ? "outline" : "default"} aria-pressed={enabled}
      disabled={busy || !loaded || (!enabled && !canEnable)} onclick={ontoggle}>
      {busy ? "Working..." : enabled ? "Disable" : "Enable"}
    </Button>
  </header>

  {#if loadError}
    <div class="mb-4 flex items-center justify-between gap-3 rounded-md border border-destructive/40 bg-destructive/10 p-3" role="alert">
      Wallpaper settings could not load.<Button variant="outline" onclick={onretry}>Try again</Button>
    </div>
  {/if}

  <section class="settings-section grid gap-3" aria-label="Wallpaper source">
    <h3 class="text-sm font-semibold">Source</h3>
    <div class="source-options" role="group" aria-label="Wallpaper source">
      {#each sources.filter(source => !linux || source.kind === "video") as source}
        <Button variant="outline" aria-pressed={config.kind === source.kind} disabled={busy || !loaded} onclick={() => choose(source.kind)}
          class="source-option">
          <Icon name={source.icon} size={15}/>{source.title}
        </Button>
      {/each}
    </div>

    <label class="grid gap-2 text-xs font-medium" for="wallpaper-source">
      {config.kind === "video" ? "MP4 file" : config.kind === "local" ? "HTML file" : "Page URL"}
      <div class="flex min-w-0 items-center gap-2">
        <Input id="wallpaper-source" type={config.kind === "url" ? "url" : "text"} bind:value={config.source} disabled={busy || !loaded}
          placeholder={config.kind === "url" ? "https://example.com/wallpaper" : config.kind === "video" ? "Choose an .mp4 file" : "Choose an .html or .htm file"} spellcheck="false" onchange={onsettingschange}
          class="h-9 min-w-0 flex-1 rounded-md border border-input bg-background px-3 text-xs select-text"/>
        {#if config.kind !== "url"}<Button variant="outline" onclick={onbrowse} disabled={busy || !loaded}><Icon name="folder" size={14}/>Browse</Button>{/if}
      </div>
    </label>
  </section>

  {#if status?.reason}<p role="status" class="my-4 rounded-md border border-border bg-muted p-3 text-xs leading-relaxed" class:text-destructive={status.phase === "error"}>{status.reason}</p>{/if}

  <section class="settings-section mt-4" aria-labelledby="playback-title">
    <h3 id="playback-title" class="mb-2 text-sm font-semibold">Playback</h3>
    {#if config.kind === "video"}
      <p class="mb-3 text-xs leading-relaxed text-muted-foreground">Videos loop silently at their original frame rate. Use a 1080p, 30 FPS H.264 MP4 for lower resource use. {#if !linux}Automatic pause settings also apply to videos.{/if}</p>
    {:else}
    <div class="setting-row">
      <span class="text-xs font-medium">Frame limit</span>
      <div class="segmented-control" role="group" aria-label="Frame limit">
        {#each [15,24,30,60] as fps}
          <Button variant="ghost" size="sm" aria-pressed={config.fps === fps} aria-label="{fps} FPS" disabled={busy || !loaded} onclick={() => updateSetting("fps", fps)}>{fps}</Button>
        {/each}
        <span class="pr-2 text-[10px] text-muted-foreground">FPS</span>
      </div>
    </div>
    {/if}
    {#if !linux}
    <div class="setting-row"><span class="text-xs font-medium">Pause when desktop is covered</span><Switch label="Pause when desktop is covered" checked={config.pauseCovered} disabled={busy || !loaded} onchange={v => updateSetting("pauseCovered", v)}/></div>
    <div class="setting-row"><span class="text-xs font-medium">Pause on battery</span><Switch label="Pause wallpaper on battery" checked={config.pauseBattery} disabled={busy || !loaded} onchange={v => updateSetting("pauseBattery", v)}/></div>
    <div class="setting-row"><span class="text-xs font-medium">Span all displays</span><Switch label="Span wallpaper across all displays" checked={config.span} disabled={busy || !loaded} onchange={v => updateSetting("span", v)}/></div>
    {/if}
  </section>
</div>
