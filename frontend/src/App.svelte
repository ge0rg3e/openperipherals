<script lang="ts">
  import {onMount} from "svelte";
  import TopBar from "./ui/TopBar.svelte";
  import Sidebar from "./ui/Sidebar.svelte";
  import Home from "./pages/Home.svelte";
  import KeyboardPage from "./pages/KeyboardPage.svelte";
  import MousePage from "./pages/MousePage.svelte";
  import CoolingPage from "./pages/CoolingPage.svelte";
  import SettingsPage from "./pages/SettingsPage.svelte";
  import {app, start} from "./lib/store.svelte";

  onMount(start);

  // A device page whose device was unplugged falls back to Home.
  $effect(() => {
    if ((app.view === "keyboard" && !app.kb) || (app.view === "mouse" && !app.mouse)) app.view = "home";
  });
</script>

<div class="shell" class:no-sidebar={!app.sidebar}>
  <TopBar />
  {#if app.sidebar}<Sidebar />{/if}
  <main class="content">
    {#if app.view === "keyboard"}
      <KeyboardPage />
    {:else if app.view === "mouse"}
      <MousePage />
    {:else if app.view === "cooling"}
      <CoolingPage />
    {:else if app.view === "settings"}
      <SettingsPage />
    {:else}
      <Home />
    {/if}
  </main>

  {#if app.error}
    <div class="toast" role="alert">{app.error}</div>
  {/if}
</div>
