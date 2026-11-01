<script lang="ts">
  import {onMount} from "svelte";
  import {Events} from "@wailsio/runtime";
  import {WallpaperService, type WallpaperStatus} from "../../bindings/openperipherals";
  import type {Config} from "../../bindings/openperipherals/internal/wallpaper/models";
  import WallpaperPanel from "../ui/WallpaperPanel.svelte";
  import {notify} from "../lib/store.svelte";

  let config = $state<Config>({enabled:false, kind:"local", source:"", fps:30, span:false, pauseCovered:true, pauseBattery:true});
  let status = $state<WallpaperStatus | null>(null);
  let busy = $state(false);
  let loaded = $state(false);
  let loadError = $state(false);

  onMount(() => {
    let disposed = false;
    WallpaperService.Status().then(s => {
      if (disposed) return;
      status = s; config = editableConfig(s.config); loaded = true;
    }).catch(e => {if (!disposed) {loadError = true; notify(e);}});
    const off = Events.On("wallpaper", e => {status = e.data;});
    return () => {disposed = true; off();};
  });

  async function retry() {
    try {
      status = await WallpaperService.Status();
      config = editableConfig(status.config); loaded = true; loadError = false;
    } catch(e) {notify(e);}
  }
  async function browse() {
    try {const path = await (config.kind === "video" ? WallpaperService.PickMP4() : WallpaperService.PickHTML()); if (path) {config.source = path; await settingsChanged();}} catch(e) {notify(e);}
  }
  // Existing built-in settings no longer appear as a selectable source.
  function editableConfig(saved: Config): Config {
    return {...saved, kind: saved.kind === "builtin" ? "local" : saved.kind,
      source: saved.kind === "builtin" ? "" : saved.source};
  }
  async function perform(action: () => Promise<WallpaperStatus>) {
    if (busy) return;
    busy = true;
    try {
      status = await action();
      config.enabled = status.config.enabled;
    } catch(e) {notify(e);} finally {busy = false;}
  }
  async function toggle() {
    if (status?.config.enabled) await perform(WallpaperService.Stop);
    else await perform(() => WallpaperService.Apply($state.snapshot(config)));
  }
  async function settingsChanged() {
    if (!status?.config.enabled || !config.source.trim()) return;
    await perform(() => WallpaperService.Apply($state.snapshot(config)));
  }
</script>

<WallpaperPanel bind:config {status} {busy} {loaded} {loadError}
  onretry={retry} onbrowse={browse} ontoggle={toggle} onsettingschange={settingsChanged} />
