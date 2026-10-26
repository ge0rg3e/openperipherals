<script lang="ts">
  // Flat slider: a native range input (keyboard and screen-reader support)
  // over a drawn track with a solid fill.
  let {value = $bindable(), min = 0, max = 100, step = 1, label = "", oninput, onchange, dim = false}: {
    value: number;
    min?: number;
    max?: number;
    step?: number;
    label?: string;
    oninput?: () => void;
    onchange?: () => void;
    dim?: boolean;
  } = $props();

  const pct = $derived(((Math.min(Math.max(value, min), max) - min) / (max - min || 1)) * 100);
</script>

<div class="slider" class:dim style="--p: {pct}%">
  <span class="slider-track"><span class="slider-fill"></span></span>
  <input type="range" {min} {max} {step} bind:value aria-label={label} {oninput} {onchange} />
</div>
