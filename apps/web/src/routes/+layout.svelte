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
<TitleBar />
{#if devMode}
  <DevLog />
{/if}
<Toaster />
{@render children()}
