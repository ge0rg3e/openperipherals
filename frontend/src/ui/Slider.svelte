<script lang="ts">
  import {Slider as SliderPrimitive} from "$lib/components/ui/slider";

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

  function update(next: number) {
    if (next === value) return;
    value = next;
    oninput?.();
  }
</script>

<SliderPrimitive
  type="single"
  value={Math.min(Math.max(value, min), max)}
  {min}
  {max}
  {step}
  aria-label={label}
  onValueChange={update}
  onValueCommit={() => onchange?.()}
  class={dim ? "opacity-45" : ""}
/>
