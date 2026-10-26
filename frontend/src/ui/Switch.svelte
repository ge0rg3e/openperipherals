<script lang="ts">
  let {checked, label, disabled = false, onchange}:
    {checked: boolean; label: string; disabled?: boolean; onchange?: (next: boolean) => void} = $props();
</script>

<button class="switch" class:on={checked} role="switch" aria-checked={checked} aria-label={label} {disabled}
        onclick={(e) => { e.stopPropagation(); onchange?.(!checked); }}>
  <span class="switch-knob"></span>
</button>
