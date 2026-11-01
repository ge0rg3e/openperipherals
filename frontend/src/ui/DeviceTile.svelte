<script lang="ts">
  import type {Snippet} from "svelte";
  import * as Card from "$lib/components/ui/card";
  import {Button} from "$lib/components/ui/button";
  import Icon from "./Icon.svelte";

  let {title, facts = [], wide = false, offline = false, onopen, children}:
    {title: string; facts?: string[]; wide?: boolean; offline?: boolean; onopen?: () => void; children: Snippet} = $props();

  const tileClass = $derived(`group relative min-w-0 overflow-hidden rounded-lg border bg-card p-3 gap-0 shadow-none ring-0 transition-colors ${onopen ? "cursor-pointer hover:border-primary/35 hover:bg-[#151515]" : ""} ${wide ? "sm:col-span-2" : ""} ${offline ? "border-dashed bg-card/50" : "border-border"}`);
</script>

<Card.Root class={tileClass}>
  {#if onopen}
    <Button variant="ghost" class="absolute inset-0 z-10 h-full w-full cursor-pointer rounded-lg border-0 bg-transparent p-0 hover:bg-transparent focus-visible:ring-inset" onclick={onopen} aria-label="Open {title}"></Button>
  {/if}
  <div class="pointer-events-none relative z-20 flex min-w-0 items-center justify-between gap-2">
    <h3 class="flex min-w-0 items-center gap-1.5 truncate text-[13px] font-semibold"><span class={offline ? "size-1.5 shrink-0 rounded-full bg-muted-foreground/45" : "size-1.5 shrink-0 rounded-full bg-foreground"}></span><span class="truncate">{title}</span></h3>
    {#if onopen}
      <span class="text-muted-foreground" aria-hidden="true"><Icon name="gear" size={13} /></span>
    {/if}
  </div>
  {#if facts.length}
    <div class="pointer-events-none relative z-20 mt-2 flex flex-wrap gap-1">
      {#each facts as f}<span class="rounded-md bg-[#242424] px-1.5 py-1 text-[10px] leading-none text-muted-foreground">{f}</span>{/each}
    </div>
  {/if}
  <div class="pointer-events-none relative z-0 mt-2 flex h-32 items-center justify-center overflow-hidden">{@render children()}</div>
</Card.Root>
