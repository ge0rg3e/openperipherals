<script lang="ts">
  import DeviceTile from "../ui/DeviceTile.svelte";
  import KeyboardArt from "../art/KeyboardArt.svelte";
  import MouseArt from "../art/MouseArt.svelte";
  import FanArt from "../art/FanArt.svelte";
  import GpuArt from "../art/GpuArt.svelte";
  import {app, EFFECT_NAMES, mouseArtColors, mouseArtSync, mouseZones} from "../lib/store.svelte";

  const effectName = (e: string) => (e === "off" ? "Lighting off" : (EFFECT_NAMES[e] ?? e));
  const stage = $derived(app.mouse ? app.mouse.stages[app.mouse.current] : null);
  const ms = $derived(app.status?.mouse);
  const mouseFacts = $derived.by(() => {
    if (!ms || !app.mouse) return [];
    const synced = app.sync.enabled && (ms.kind === "corsair" || !!ms.effects?.length) ? ["Synced"] : [];
    if (ms.kind === "corsair") return [...synced, `DPI ${stage?.dpi ?? "–"}`, app.mouse.current === 0 ? "Sniper" : `Stage ${app.mouse.current}`];
    return [...synced, `DPI ${ms.dpi}`, ...(ms.polling ? [`${ms.polling} Hz`] : []), ...(ms.battery >= 0 ? [`${ms.battery}%`] : [])];
  });
  const fans = $derived(app.fans?.fans ?? []);
  const gpuTemp = (fanId: string) => app.fans?.sensors?.find((s) => s.id === fanId.split(":")[0])?.temp ?? -1;
  const mode = (m: string) => (m === "auto" ? "Auto" : m === "fixed" ? "Fixed" : "Curve");
</script>

<div class="tiles">
  {#if app.kb && app.status?.keyboard}
    <DeviceTile title={app.status.keyboard.name} wide onopen={() => (app.view = "keyboard")}
                facts={[...(app.sync.enabled ? ["Synced"] : []), effectName(app.kb.lighting.effect),
                        ...(app.status.keyboard.hasBrightness ? [`Brightness ${app.kb.brightness}%`] : [])]}>
      <KeyboardArt lighting={app.kb.lighting} brightness={app.kb.brightness} layout={app.status.keyboard.layout} media={app.status.keyboard.media} />
    </DeviceTile>
  {:else}
    <DeviceTile title="Keyboard" wide offline facts={["Not connected"]}>
      <p class="tile-empty">Plug the keyboard in; it's detected within a few seconds.</p>
    </DeviceTile>
  {/if}

  {#if app.mouse && ms}
    <DeviceTile title={ms.name} onopen={() => (app.view = "mouse")} facts={mouseFacts}>
      {#if ms.kind === "corsair"}
        <MouseArt colors={mouseZones()} sync={app.sync} brightness={app.mouse.brightness} />
      {:else}
        <MouseArt colors={mouseArtColors()} sync={mouseArtSync()} />
      {/if}
    </DeviceTile>
  {:else}
    <DeviceTile title="Mouse" offline facts={["Not connected"]}>
      <p class="tile-empty">Plug the mouse in; it's detected within a few seconds.</p>
    </DeviceTile>
  {/if}

  {#each fans as f (f.id)}
    <DeviceTile title={f.id.startsWith("gpu") ? f.source : f.name} onopen={() => (app.view = "cooling")}
                facts={[
                  f.rpm >= 0 ? `${f.rpm} RPM` : f.duty >= 0 ? `${f.duty}%` : "Auto",
                  f.id.startsWith("gpu") && gpuTemp(f.id) >= 0 ? `${gpuTemp(f.id)} °C` : mode(f.config.mode),
                ]}>
      {#if f.id.startsWith("gpu")}
        <GpuArt duty={f.duty} />
      {:else}
        <FanArt duty={f.duty} rpm={f.rpm} />
      {/if}
    </DeviceTile>
  {/each}

  {#if app.fans?.notice}
    <DeviceTile title="Motherboard fans" offline onopen={() => (app.view = "cooling")} facts={["Unavailable"]}>
      <p class="tile-empty">{app.fans.notice}</p>
    </DeviceTile>
  {/if}
</div>
