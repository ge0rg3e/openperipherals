<script lang="ts">
  import "./layout.css";
  import DevLog from "$lib/components/DevLog.svelte";
  import TitleBar from "$lib/components/desktop/TitleBar.svelte";
  import { Toaster } from "$lib/components/ui/sonner/index.js";

  let { children } = $props();

  const devMode =
    typeof window !== "undefined" &&
    new URLSearchParams(window.location.search).has("dev");
</script>

<svelte:head><link rel="icon" href="/favicon.png" /></svelte:head>
<!-- The whole app lives in exactly one viewport: the window never scrolls,
     content is sized to fit what remains below the title bar. -->
<div class="relative flex h-dvh flex-col overflow-hidden">
  <TitleBar />
  {#if devMode}
    <DevLog />
  {/if}
  <Toaster />
  <div class="min-h-0 flex-1 overflow-hidden">
    {@render children()}
  </div>
</div>
