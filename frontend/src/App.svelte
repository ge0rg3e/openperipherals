<script lang="ts">
  import {onMount} from "svelte";
  import TopBar from "./ui/TopBar.svelte";
  import Sidebar from "./ui/Sidebar.svelte";
  import Home from "./pages/Home.svelte";
  import KeyboardPage from "./pages/KeyboardPage.svelte";
  import MousePage from "./pages/MousePage.svelte";
  import CoolingPage from "./pages/CoolingPage.svelte";
  import SettingsPage from "./pages/SettingsPage.svelte";
  import WallpaperPage from "./pages/WallpaperPage.svelte";
  import {app, start} from "./lib/store.svelte";

  onMount(start);

  // A device page whose device was unplugged falls back to Home.
  $effect(() => {
    if ((app.view === "keyboard" && !app.kb) || (app.view === "mouse" && !app.mouse)) app.view = "home";
  });
</script>

<div class="app-shell" class:with-sidebar={app.sidebar}>
  <TopBar />
  {#if app.sidebar}<Sidebar />{/if}
  <main class="workspace">
    <div class="page-content">
      {#if app.view === "keyboard"}
        <KeyboardPage />
      {:else if app.view === "mouse"}
        <MousePage />
      {:else if app.view === "cooling"}
        <CoolingPage />
      {:else if app.view === "settings"}
        <SettingsPage />
      {:else if app.view === "wallpaper"}
        <WallpaperPage />
      {:else}
        <Home />
      {/if}
    </div>
  </main>

  {#if app.error}
    <div class="fixed right-5 bottom-5 z-50 max-w-md rounded-lg border border-destructive/40 bg-destructive/15 px-4 py-3 text-sm text-foreground shadow-lg" role="alert">{app.error}</div>
  {/if}
</div>
