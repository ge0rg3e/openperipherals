<script lang="ts">
  import FanConfigEditor from "./FanConfigEditor.svelte";
  import {FanService, type Fan, type FanConfig, type TempSensor} from "../bindings/openperipherals";

  let {fan, sensors, onerror}: {fan: Fan; sensors: TempSensor[]; onerror: (e: unknown) => void} = $props();

  // svelte-ignore state_referenced_locally
  let cfg = $state<FanConfig>({...fan.config, points: (fan.config.points ?? []).map((p) => [...p])});

  let saveTimer: ReturnType<typeof setTimeout> | undefined;
  function save(immediate = false) {
    clearTimeout(saveTimer);
    const delay = immediate ? 0 : 250;
    saveTimer = setTimeout(() => FanService.SetFanConfig(fan.id, cfg).catch(onerror), delay);
  }
</script>

<section class="control-panel fan-panel">
  <header class="flex items-start justify-between gap-4">
    <div>
      <h2 class="text-sm font-semibold">{fan.name}</h2>
      <p class="text-xs leading-relaxed text-muted-foreground">{fan.source}</p>
    </div>
    <div class="flex items-baseline gap-2 text-right text-xs text-muted-foreground">
      {#if fan.rpm >= 0}<b class="text-base text-foreground tabular-nums">{fan.rpm}</b> RPM{/if}
      {#if fan.duty >= 0}<span class="ml-2 tabular-nums">{fan.duty}%</span>{/if}
    </div>
  </header>

  <FanConfigEditor bind:config={cfg} {sensors} minDuty={fan.minDuty} gpu={fan.id.startsWith("gpu")}
                   onchange={(immediate) => save(immediate)} />
</section>
