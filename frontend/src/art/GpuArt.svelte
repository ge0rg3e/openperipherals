<script lang="ts">
  // Single-fan graphics card, front view: angular faceted shroud, heatsink
  // fins, I/O bracket with ports, PCB and gold contacts. Flat colours only.
  import Rotor from "./Rotor.svelte";

  let {duty = -1}: {duty?: number} = $props();

  const speed = $derived(duty >= 0 ? duty / 100 : 0.4);
  const SHROUD = "M30 22 H268 L282 36 V116 L268 130 H44 L30 116 Z";
  const FACET = "M30 22 H150 L118 130 H44 L30 116 Z";
</script>

<svg class="gpu-art" viewBox="0 0 300 168" role="img" aria-label="Graphics card">
  <!-- heatsink fins peeking above the shroud -->
  {#each Array(30) as _, i}
    <rect x={44 + i * 7.6} y="14" width="3" height="10" fill="#3a3e47" />
  {/each}

  <!-- I/O bracket with ports -->
  <rect x="8" y="10" width="20" height="132" rx="2" fill="#3a3e47" />
  <rect x="12" y="24" width="12" height="8" rx="1.5" fill="#0b0c0f" />
  <rect x="12" y="38" width="12" height="8" rx="1.5" fill="#0b0c0f" />
  <path d="M12 54 H24 L22 62 H14 Z" fill="#0b0c0f" />
  {#each [74, 82, 90, 98, 106] as y}
    <rect x="13" y={y} width="10" height="3" rx="1.5" fill="#262930" />
  {/each}

  <!-- PCB and contacts -->
  <rect x="28" y="128" width="262" height="12" fill="#163222" />
  {#each Array(22) as _, i}
    <rect x={92 + i * 6.5} y="140" width="4.5" height="12" fill="#c9a227" />
  {/each}
  <rect x="236" y="140" width="22" height="12" fill="#c9a227" />

  <!-- shroud -->
  <path d={SHROUD} fill="#1b1d22" stroke="#2c2f37" stroke-width="2" />
  <path d={FACET} fill="#212429" />
  <path d="M46 36 H132" stroke="var(--accent)" stroke-width="3" />
  {#each [52, 62, 72, 82, 92, 102] as y, i}
    <path d="M{48 + i * 3} {y} H{100 - i * 6}" stroke="#2f333c" stroke-width="3" stroke-linecap="round" />
  {/each}

  <!-- fan -->
  <circle cx="210" cy="76" r="50" fill="#2a2d35" />
  <circle cx="210" cy="76" r="47" fill="#0d0e11" />
  <Rotor cx={210} cy={76} r={45} blades={11} {speed} />
</svg>
