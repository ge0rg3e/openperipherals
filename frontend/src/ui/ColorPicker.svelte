<script lang="ts">
  import {Input} from "$lib/components/ui/input";

  // Swatch button with its hex value; opens the system colour picker.
  let {value = $bindable(), label = "Colour", compact = false, oninput, onchange}: {
    value: string;
    label?: string;
    compact?: boolean;
    oninput?: () => void;
    onchange?: () => void;
  } = $props();
</script>

<label class={compact
  ? "relative inline-flex size-6 min-w-0 items-center justify-center rounded-full border border-border bg-background p-0"
  : "relative inline-flex h-9 min-w-24 items-center gap-2 rounded-md border border-border bg-background px-2 text-xs"} title={label}>
  <span class="size-4 shrink-0 rounded border border-border" style="background: {value}"></span>
  {#if !compact}<span class="font-mono text-[10px] text-muted-foreground">{value.toUpperCase()}</span>{/if}
  <Input class="absolute inset-0 h-full w-full cursor-pointer opacity-0" type="color" bind:value aria-label={label} {oninput} {onchange} />
</label>
