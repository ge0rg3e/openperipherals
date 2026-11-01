<script lang="ts">
  import DeviceTile from "../ui/DeviceTile.svelte";
  import {app, EFFECT_NAMES} from "../lib/store.svelte";

  const effectName = (e: string) => (e === "off" ? "Lighting off" : (EFFECT_NAMES[e] ?? e));
  const stage = $derived(app.mouse ? app.mouse.stages[app.mouse.current] : null);
  const ms = $derived(app.status?.mouse);
  type DeviceKind = "keyboard" | "mouse" | "fan" | "gpu";
  function deviceImage(name: string, kind: DeviceKind): string | undefined {
    const model = name.toLowerCase();
    if (kind === "keyboard" && model.includes("razer") && model.includes("huntsman")) {
      return "/devices/razer-hustman-elite.png";
    }
    if (kind === "mouse" && model.includes("corsair") && model.includes("sabre")) {
      return "/devices/corsair-sabre-rgb.png";
    }
    if (kind === "gpu" && model.includes("1050") && model.includes("ti")) {
      return "/devices/gtx-1050-ti.png";
    }
    if (kind === "fan") return "/devices/fan.png";
    return undefined;
  }
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

<div class="device-grid">
  {#if app.kb && app.status?.keyboard}
    <DeviceTile title={app.status.keyboard.name} onopen={() => (app.view = "keyboard")}
                facts={[...(app.sync.enabled ? ["Synced"] : []), effectName(app.kb.lighting.effect),
                        ...(app.status.keyboard.hasBrightness ? [`Brightness ${app.kb.brightness}%`] : [])]}>
      {#if deviceImage(app.status.keyboard.name, "keyboard")}
        <img class="max-h-28 w-full object-contain" src={deviceImage(app.status.keyboard.name, "keyboard")} alt={app.status.keyboard.name} />
      {/if}
    </DeviceTile>
  {:else}
    <DeviceTile title="Keyboard" offline facts={["Not connected"]}>
      <p class="pointer-events-none relative z-0 mx-auto max-w-xs py-8 text-center text-xs leading-relaxed text-muted-foreground">Plug the keyboard in; it's detected within a few seconds.</p>
    </DeviceTile>
  {/if}

  {#if app.mouse && ms}
    <DeviceTile title={ms.name} onopen={() => (app.view = "mouse")} facts={mouseFacts}>
      {#if deviceImage(ms.name, "mouse")}
        <img class="max-h-28 w-full object-contain" src={deviceImage(ms.name, "mouse")} alt={ms.name} />
      {/if}
    </DeviceTile>
  {:else}
    <DeviceTile title="Mouse" offline facts={["Not connected"]}>
      <p class="pointer-events-none relative z-0 mx-auto max-w-xs py-8 text-center text-xs leading-relaxed text-muted-foreground">Plug the mouse in; it's detected within a few seconds.</p>
    </DeviceTile>
  {/if}

  {#each fans as f (f.id)}
    <DeviceTile title={f.id.startsWith("gpu") ? f.source : f.name} onopen={() => (app.view = "cooling")}
                facts={[
                  f.rpm >= 0 ? `${f.rpm} RPM` : f.duty >= 0 ? `${f.duty}%` : "Auto",
                  f.id.startsWith("gpu") && gpuTemp(f.id) >= 0 ? `${gpuTemp(f.id)} °C` : mode(f.config.mode),
                ]}>
      {#if deviceImage(f.id.startsWith("gpu") ? f.source : f.name, f.id.startsWith("gpu") ? "gpu" : "fan")}
        <img class="max-h-28 w-full object-contain" src={deviceImage(f.id.startsWith("gpu") ? f.source : f.name, f.id.startsWith("gpu") ? "gpu" : "fan")} alt={f.id.startsWith("gpu") ? f.source : f.name} />
      {/if}
    </DeviceTile>
  {/each}

  {#if app.fans?.notice}
    <DeviceTile title="Motherboard fans" offline onopen={() => (app.view = "cooling")} facts={["Unavailable"]}>
      <p class="pointer-events-none relative z-0 mx-auto max-w-xs py-8 text-center text-xs leading-relaxed text-muted-foreground">{app.fans.notice}</p>
    </DeviceTile>
  {/if}
</div>
