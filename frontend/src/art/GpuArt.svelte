<script lang="ts">
  // Single-fan graphics card, front view: angular faceted shroud, heatsink
  // fins, I/O bracket with ports, PCB and gold contacts.
  import Rotor from "./Rotor.svelte";

  let {duty = -1}: {duty?: number} = $props();

  const speed = $derived(duty >= 0 ? duty / 100 : 0.4);
  const id = $props.id();
  const SHROUD = "M30 22 H268 L282 36 V116 L268 130 H44 L30 116 Z";
  const FACET = "M30 22 H150 L118 130 H44 L30 116 Z";
</script>

<svg class="block h-auto max-h-40 w-[96%] max-w-full" viewBox="0 0 300 168" role="img" aria-label="Graphics card">
  <defs>
    <linearGradient id="{id}-metal" x2=".4" y2="1">
      <stop stop-color="#8a96a5" /><stop offset=".25" stop-color="#4c5868" /><stop offset=".7" stop-color="#242d39" /><stop offset="1" stop-color="#131a24" />
    </linearGradient>
    <linearGradient id="{id}-shroud" x2=".4" y2="1">
      <stop stop-color="#495463" /><stop offset=".45" stop-color="#262f3b" /><stop offset="1" stop-color="#131b25" />
    </linearGradient>
    <linearGradient id="{id}-gold" x2="0" y2="1">
      <stop stop-color="#f6d485" /><stop offset=".45" stop-color="#c49a43" /><stop offset="1" stop-color="#70501d" />
    </linearGradient>
  </defs>
  <!-- heatsink fins peeking above the shroud -->
  {#each Array(30) as _, i}
    <rect x={44 + i * 7.6} y="14" width="3" height="10" fill="url(#{id}-metal)" />
  {/each}

  <!-- I/O bracket with ports -->
  <rect x="8" y="10" width="20" height="132" rx="2" fill="url(#{id}-metal)" stroke="#8996a7" stroke-opacity=".25" />
  <rect x="12" y="24" width="12" height="8" rx="1.5" fill="#0b0c0f" />
  <rect x="12" y="38" width="12" height="8" rx="1.5" fill="#0b0c0f" />
  <path d="M12 54 H24 L22 62 H14 Z" fill="#0b0c0f" />
  {#each [74, 82, 90, 98, 106] as y}
    <rect x="13" y={y} width="10" height="3" rx="1.5" fill="#262930" />
  {/each}

  <!-- PCB and contacts -->
  <rect x="28" y="128" width="262" height="12" fill="#163222" />
  {#each Array(22) as _, i}
    <rect x={92 + i * 6.5} y="140" width="4.5" height="12" fill="url(#{id}-gold)" />
  {/each}
  <rect x="236" y="140" width="22" height="12" fill="url(#{id}-gold)" />

  <!-- shroud -->
  <path d={SHROUD} transform="translate(0 5)" fill="#080d14" stroke="#080d14" stroke-width="3" />
  <path d={SHROUD} fill="url(#{id}-shroud)" stroke="#657285" stroke-opacity=".5" stroke-width="1.5" />
  <path d={FACET} fill="#76859a" opacity=".08" />
  <path d="M34 23 H267 L279 36 M34 25 V114 L46 126" fill="none" stroke="#b0bdd0" stroke-opacity=".22" />
  <path d="M46 36 H132" stroke="var(--accent)" stroke-width="3" />
  {#each [52, 62, 72, 82, 92, 102] as y, i}
    <path d="M{48 + i * 3} {y} H{100 - i * 6}" stroke="#2f333c" stroke-width="3" stroke-linecap="round" />
  {/each}

  <!-- fan -->
  <circle cx="210" cy="76" r="51" fill="url(#{id}-metal)" />
  <circle cx="210" cy="76" r="47.5" fill="#090e15" stroke="#05080d" stroke-width="2" />
  {#each [[152, 36], [268, 36], [152, 116], [268, 116]] as [x, y]}
    <circle cx={x} cy={y} r="2.2" fill="#111721" stroke="#5c6b7e" stroke-width=".7" />
    <path d="M{x - 1} {y} h2" stroke="#738195" stroke-width=".6" />
  {/each}
  <Rotor cx={210} cy={76} r={45} blades={11} {speed} />
</svg>
