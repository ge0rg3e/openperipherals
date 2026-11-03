<script lang="ts">
  import {System} from "@wailsio/runtime";
  const windowsDesktop = System.IsWindows();
  import {onMount} from "svelte";
  import {Button} from "$lib/components/ui/button";
  import {PeripheralService} from "../../bindings/openperipherals";
  import PageHeader from "../ui/PageHeader.svelte";
  import Switch from "../ui/Switch.svelte";
  import {app, notify} from "../lib/store.svelte";

  let autostart = $state(false);
  let autostartElevated = $state(false);
  let busy = $state(false);

  onMount(() => {
    PeripheralService.Autostart().then((a) => (autostart = a)).catch(notify);
    PeripheralService.AutostartElevated().then((a) => (autostartElevated = a)).catch(notify);
  });

  async function toggleAutostart() {
    if (busy) return;
    busy = true;
    const next = !autostart;
    try {
      await PeripheralService.SetAutostart(next);
      autostart = next;
      autostartElevated = windowsDesktop && next;
    } catch (e) {
      notify(e);
    } finally {
      busy = false;
    }
  }

  async function fixStartup() {
    if (busy) return;
    busy = true;
    try {
      // The old entry is an unelevated Run-key value. Turning startup on again
      // relaunches elevated, installs the highest-privilege task and removes
      // the Run key only after that succeeds.
      await PeripheralService.SetAutostart(true);
      autostart = true;
      autostartElevated = true;
    } catch (e) {
      notify(e);
    } finally {
      busy = false;
    }
  }
</script>

<PageHeader title="Settings" />

<div class="settings-grid">
  <section class="settings-section">
    <h3 class="mb-2 text-sm font-semibold tracking-tight">Startup</h3>
    <div class="grid gap-2">
      <div class="flex w-full items-center justify-between gap-3 border-b border-border py-2.5 text-left text-sm">
        <span>{windowsDesktop ? "Start with Windows (as administrator)" : "Start at login"}</span>
        <Switch label={windowsDesktop ? "Start with Windows as administrator" : "Start at login"} checked={autostart} disabled={busy} onchange={toggleAutostart} />
      </div>
      {#if !windowsDesktop}
        <p class="text-xs leading-relaxed text-muted-foreground">Opens OpenPeripherals when you sign in.</p>
      {:else if autostart && !autostartElevated}
        <div class="flex flex-wrap items-center justify-between gap-3 border-l-2 border-amber-400 bg-amber-500/5 px-3 py-2.5 text-sm text-amber-100">
          <span>Windows currently starts an unelevated copy, so cooling asks for a restart each time.</span>
          <Button variant="outline" onclick={fixStartup} disabled={busy}>Fix startup</Button>
        </div>
      {:else}
        <p class="text-xs leading-relaxed text-muted-foreground">
          {#if autostart}
            Runs in the tray at login with administrator rights, so fan control works without a prompt.
          {:else}
            Starts in the tray at login with administrator rights, so cooling never asks for a restart. Enabling it may show a UAC prompt once.
          {/if}
        </p>
      {/if}
    </div>
  </section>

  <section class="settings-section">
    <h3 class="mb-2 text-sm font-semibold tracking-tight">Key blocking</h3>
    <div class="grid gap-2">
      <p class="text-xs leading-relaxed text-muted-foreground">Choose keys or dial actions to block on your keyboard.</p>
      <Button variant="outline" class="flex w-full items-center justify-between gap-3 rounded-md border-border bg-transparent px-3 py-2.5 text-left text-sm hover:bg-muted/40" onclick={() => (app.view = "keyboard")} disabled={!app.kb}>
        <span>Open keyboard</span><span class="text-xs text-muted-foreground">{Object.keys(app.disabled).length} disabled</span>
      </Button>
    </div>
  </section>
</div>
