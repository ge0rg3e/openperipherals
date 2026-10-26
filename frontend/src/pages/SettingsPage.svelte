<script lang="ts">
  import {onMount} from "svelte";
  import {PeripheralService} from "../../bindings/openperipherals";
  import PageHeader from "../ui/PageHeader.svelte";
  import Switch from "../ui/Switch.svelte";
  import {app, notify} from "../lib/store.svelte";

  let autostart = $state(false);

  onMount(() => {
    PeripheralService.Autostart().then((a) => (autostart = a)).catch(notify);
  });

  async function toggleAutostart() {
    try {
      await PeripheralService.SetAutostart(!autostart);
      autostart = !autostart;
    } catch (e) {
      notify(e);
    }
  }
</script>

<PageHeader title="Settings" subtitle="Startup and app behaviour" />

<div class="panels">
  <section class="panel">
    <h3 class="panel-title">Startup</h3>
    <div class="key">
      <span>Start with Windows (in tray)</span>
      <Switch label="Start with Windows" checked={autostart} onchange={toggleAutostart} />
    </div>
    <p class="hint">When the app runs as administrator this uses a scheduled task, so fan control works at login without a prompt.</p>
  </section>

  <section class="panel">
    <h3 class="panel-title">Key blocking</h3>
    <p class="meta">Disable keys from the keyboard's page: click a key or the dial on the 3D keyboard.</p>
    <div class="field">
      <button class="key" onclick={() => (app.view = "keyboard")} disabled={!app.kb}>
        <span>Open keyboard</span><span class="state">{Object.keys(app.disabled).length} disabled</span>
      </button>
    </div>
  </section>
</div>
