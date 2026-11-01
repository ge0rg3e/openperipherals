<script lang="ts">
  import {Button} from "$lib/components/ui/button";
  import Icon from "./Icon.svelte";
  import {app} from "../lib/store.svelte";

  let {title, subtitle = "", back = false}: {title: string; subtitle?: string; back?: boolean} = $props();
</script>

<div class="page-heading">
  {#if back}
    <Button variant="ghost" size="icon" class="inline-flex size-7 items-center justify-center rounded-md border border-transparent bg-transparent text-muted-foreground transition-colors hover:bg-muted hover:text-foreground" aria-label="Back to Home" title="Back" onclick={() => (app.view = "home")}>
      <Icon name="back" size={16} />
    </Button>
  {/if}
  <div>
    <h2 class="text-base font-semibold tracking-tight">{title}</h2>
    {#if subtitle}<p class="mt-1 text-xs leading-relaxed text-muted-foreground">{subtitle}</p>{/if}
  </div>
</div>
