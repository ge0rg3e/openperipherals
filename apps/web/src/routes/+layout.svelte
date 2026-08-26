<script lang="ts">
  import "./layout.css";
  import DevLog from "$lib/components/DevLog.svelte";
  import TitleBar from "$lib/components/desktop/TitleBar.svelte";
  import { Toaster } from "$lib/components/ui/sonner/index.js";

  let { children } = $props();

  const devMode =
    typeof window !== "undefined" &&
    new URLSearchParams(window.location.search).has("dev");

  // In the desktop shell the window never scrolls as a whole: the title bar
  // stays pinned and the content area below it owns the scrollbar.
  const desktop =
    typeof navigator !== "undefined" && /Electron/i.test(navigator.userAgent);
</script>

<svelte:head><link rel="icon" href="/favicon.png" /></svelte:head>
<div class="relative flex min-h-dvh flex-col {desktop ? 'h-dvh overflow-hidden' : ''}">
  <TitleBar />
  {#if devMode}
    <DevLog />
  {/if}
  <Toaster />
  <div class="min-h-0 flex-1 {desktop ? 'overflow-y-auto' : ''}">
    {@render children()}
  </div>
</div>
