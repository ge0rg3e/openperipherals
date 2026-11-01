<script lang="ts">
  // 120 mm case fan, front view: chamfered frame, mounting holes, rim and a
  // rotor that spins with the fan's duty (or RPM).
  import Rotor from "./Rotor.svelte";

  let {duty = -1, rpm = -1}: {duty?: number; rpm?: number} = $props();

  const speed = $derived(duty >= 0 ? duty / 100 : rpm >= 0 ? Math.min(rpm / 2200, 1) : 0.4);
  const FRAME = "M18 4 H122 L136 18 V122 L122 136 H18 L4 122 V18 Z";
  const id = $props.id();
</script>

<svg class="block h-auto max-h-40 w-[150px] max-w-full" viewBox="0 0 140 140" role="img" aria-label="Fan">
  <defs>
    <linearGradient id="{id}-frame" x2=".8" y2="1">
      <stop stop-color="#535d6b" /><stop offset=".35" stop-color="#282f39" /><stop offset="1" stop-color="#141920" />
    </linearGradient>
    <linearGradient id="{id}-rim" x2=".6" y2="1">
      <stop stop-color="#657181" /><stop offset=".45" stop-color="#1c242e" /><stop offset="1" stop-color="#3c4654" />
    </linearGradient>
  </defs>
  <path d={FRAME} transform="translate(0 2)" fill="#080b10" stroke="#080b10" stroke-width="3" />
  <path d={FRAME} fill="url(#{id}-frame)" stroke="#667181" stroke-opacity=".45" stroke-width="1" />
  <path d="M18 12 H122 L128 18 V122 L122 128 H18 L12 122 V18 Z" fill="#1b222c" stroke="#96a5b8" stroke-opacity=".12" />
  <circle cx="70" cy="70" r="59" fill="url(#{id}-rim)" />
  <circle cx="70" cy="70" r="55.5" fill="#080c12" stroke="#080c12" stroke-width="1.5" />
  {#each [0, 90, 180, 270] as angle}
    <path d="M66 70 L56 18 H64 L74 70 Z" transform="rotate({angle} 70 70)" fill="#29323d" />
  {/each}
  {#each [[20, 20], [120, 20], [20, 120], [120, 120]] as [x, y]}
    <rect x={x - 9} y={y - 9} width="18" height="18" rx="5" fill="#111720" stroke="#576271" stroke-opacity=".4" />
    <circle cx={x} cy={y} r="5" fill="url(#{id}-rim)" />
    <circle cx={x} cy={y} r="3.1" fill="#080b11" />
  {/each}
  <Rotor cx={70} cy={70} r={53} blades={9} {speed} />
</svg>
