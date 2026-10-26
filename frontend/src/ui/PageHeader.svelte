<script lang="ts">
  import Icon from "./Icon.svelte";
  import {app} from "../lib/store.svelte";

  let {title, subtitle = "", back = false}: {title: string; subtitle?: string; back?: boolean} = $props();
</script>

<div class="page-head">
  {#if back}
    <button class="icon-btn" aria-label="Back to Home" title="Back" onclick={() => (app.view = "home")}>
      <Icon name="back" size={18} />
    </button>
  {/if}
  <div>
    <h2>{title}</h2>
    {#if subtitle}<p class="meta">{subtitle}</p>{/if}
  </div>
</div>
