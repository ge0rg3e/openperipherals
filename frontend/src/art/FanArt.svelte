<script lang="ts">
  // 120 mm case fan, front view: chamfered frame, mounting holes, rim and a
  // rotor that spins with the fan's duty (or RPM).
  import Rotor from "./Rotor.svelte";

  let {duty = -1, rpm = -1}: {duty?: number; rpm?: number} = $props();

  const speed = $derived(duty >= 0 ? duty / 100 : rpm >= 0 ? Math.min(rpm / 2200, 1) : 0.4);
  const FRAME = "M18 4 H122 L136 18 V122 L122 136 H18 L4 122 V18 Z";
</script>

<svg class="fan-art" viewBox="0 0 140 140" role="img" aria-label="Fan">
  <path d={FRAME} fill="#1b1d22" stroke="#2c2f37" stroke-width="2" />
  <path d="M18 12 H122 L128 18 V122 L122 128 H18 L12 122 V18 Z" fill="#202329" />
  {#each [[20, 20], [120, 20], [20, 120], [120, 120]] as [x, y]}
    <circle cx={x} cy={y} r="6.5" fill="#2c2f37" />
    <circle cx={x} cy={y} r="3.6" fill="#0b0c0f" />
  {/each}
  <circle cx="70" cy="70" r="58" fill="#2a2d35" />
  <circle cx="70" cy="70" r="55" fill="#0d0e11" />
  <Rotor cx={70} cy={70} r={53} blades={9} {speed} />
</svg>
