<script lang="ts">
  // Custom title bar for the frameless window: drag anywhere empty to move,
  // double-click to maximise. Close hides to the tray like the native button.
  import {onMount} from "svelte";
  import {Window} from "@wailsio/runtime";
  import Icon from "./Icon.svelte";
  import {FanService} from "../../bindings/openperipherals";
  import {app, notify, rescan, type View} from "../lib/store.svelte";

  const tabs: [View, string][] = [["home", "Home"], ["cooling", "Cooling"], ["settings", "Settings"]];
  // Device pages live under Home.
  const section = $derived(app.view === "keyboard" || app.view === "mouse" ? "home" : app.view);

  let scanning = $state(false);
  async function scan() {
    scanning = true;
    await rescan();
    scanning = false;
  }

  // Fan control problems stay visible from every page.
  const fanNotice = $derived(app.fans?.notice ?? "");
  const needsAdmin = $derived(!!fanNotice && app.fans?.elevated === false);
  function fixFans() {
    if (needsAdmin) FanService.RestartAsAdmin().catch(notify);
    else app.view = "cooling";
  }

  let maximised = $state(false);
  const sync = () => Window.IsMaximised().then((m) => (maximised = m)).catch(() => {});
  onMount(() => {
    sync();
    window.addEventListener("resize", sync);
    return () => window.removeEventListener("resize", sync);
  });
</script>

<header class="titlebar">
  <div class="logo" aria-hidden="true"><span></span><span></span><span></span><span></span></div>
  <span class="app-name">OpenPeripherals<span class="beta-tag">Beta</span></span>
  <nav class="no-drag">
    {#each tabs as [id, label]}
      <button class:active={section === id} onclick={() => (app.view = id)}>{label}</button>
    {/each}
  </nav>

  <div class="titlebar-actions no-drag">
    {#if fanNotice}
      <button class="title-warning" title={needsAdmin ? `${fanNotice} Click to restart as administrator.` : fanNotice} onclick={fixFans}>
        <Icon name="shield" size={14} />
        <span>{needsAdmin ? "Fans need admin" : "Fan control unavailable"}</span>
      </button>
    {/if}
    <button class="icon-btn" class:on={app.sidebar} title={app.sidebar ? "Hide side panel" : "Show side panel"}
            aria-label="Toggle side panel" onclick={() => (app.sidebar = !app.sidebar)}>
      <Icon name="panel" />
    </button>
    <button class="icon-btn" class:spinning={scanning} title="Rescan devices" aria-label="Rescan devices" onclick={scan} disabled={scanning}>
      <Icon name="refresh" />
    </button>
  </div>

  <div class="win-controls no-drag">
    <button aria-label="Minimise" title="Minimise" onclick={() => Window.Minimise()}><Icon name="minimize" /></button>
    <button aria-label={maximised ? "Restore" : "Maximise"} title={maximised ? "Restore" : "Maximise"}
            onclick={() => Window.ToggleMaximise().then(sync)}>
      <Icon name={maximised ? "restore" : "maximize"} />
    </button>
    <button class="win-close" aria-label="Close to tray" title="Close to tray" onclick={() => Window.Close()}><Icon name="close" /></button>
  </div>
</header>
