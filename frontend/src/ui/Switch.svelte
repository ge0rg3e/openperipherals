<script lang="ts">
  import {Switch as SwitchPrimitive} from "$lib/components/ui/switch";

  let {checked, label, disabled = false, onchange}:
    {checked: boolean; label: string; disabled?: boolean; onchange?: (next: boolean) => void} = $props();

  function stopClick(event: MouseEvent) {
    event.stopPropagation();
  }
</script>

<SwitchPrimitive
  {checked}
  {disabled}
  aria-label={label}
  onclick={stopClick}
  onCheckedChange={(next) => onchange?.(next)}
/>
