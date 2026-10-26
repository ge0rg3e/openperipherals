<script lang="ts">
  // Swatch button with its hex value; opens the system colour picker.
  let {value = $bindable(), label = "Colour", compact = false, oninput, onchange}: {
    value: string;
    label?: string;
    compact?: boolean;
    oninput?: () => void;
    onchange?: () => void;
  } = $props();
</script>

<label class="color-picker" class:compact title={label}>
  <span class="color-chip" style="background: {value}"></span>
  {#if !compact}<span class="color-hex">{value.toUpperCase()}</span>{/if}
  <input type="color" bind:value aria-label={label} {oninput} {onchange} />
</label>
