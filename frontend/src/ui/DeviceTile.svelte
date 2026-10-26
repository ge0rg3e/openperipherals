<script lang="ts">
  import type {Snippet} from "svelte";
  import Icon from "./Icon.svelte";

  let {title, facts = [], wide = false, offline = false, onopen, children}:
    {title: string; facts?: string[]; wide?: boolean; offline?: boolean; onopen?: () => void; children: Snippet} = $props();
</script>

<div class="tile" class:wide class:offline>
  <button class="tile-hit" onclick={onopen} disabled={!onopen} aria-label="Open {title}"></button>
  <div class="tile-head">
    <h3><span class="tile-dot" class:on={!offline}></span>{title}</h3>
    {#if onopen}
      <span class="tile-gear" aria-hidden="true"><Icon name="gear" size={15} /></span>
    {/if}
  </div>
  {#if facts.length}
    <div class="tile-facts">
      {#each facts as f}<span>{f}</span>{/each}
    </div>
  {/if}
  <div class="tile-art">{@render children()}</div>
</div>
