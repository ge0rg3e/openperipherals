<script lang="ts">
  import PageHeader from "../ui/PageHeader.svelte";
  import FanCard from "../FanCard.svelte";
  import {app, notify} from "../lib/store.svelte";
  import {FanService} from "../../bindings/openperipherals";
</script>

<PageHeader title="Cooling" subtitle="Each fan can be on Auto, Fixed or a temperature curve" />

{#if app.fans}
  {#if app.fans.notice}
    <div class="notice">
      <span>{app.fans.notice}</span>
      {#if !app.fans.elevated}
        <button onclick={() => FanService.RestartAsAdmin().catch(notify)}>Restart as administrator</button>
      {/if}
    </div>
  {/if}

  <div class="fan-grid">
    {#each app.fans.fans ?? [] as fan (fan.id)}
      <FanCard {fan} sensors={app.fans.sensors ?? []} onerror={notify} />
    {:else}
      <p class="meta">No controllable fans found.</p>
    {/each}
  </div>

  <p class="hint">Motherboard fans never go below 20%. If the CPU reaches 90 °C or a curve's sensor stops reporting, every controlled fan goes to 100%. Quitting the app from the tray hands every fan back to Auto.</p>
{:else}
  <p class="meta">Detecting fans…</p>
{/if}
