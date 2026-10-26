<script lang="ts">
  // Spinning fan rotor (SVG group): two-tone curved blades and a hub. Spins
  // faster as `speed` (0..1) rises; stops at 0.
  let {cx, cy, r, blades = 9, speed = 0.4}: {cx: number; cy: number; r: number; blades?: number; speed?: number} = $props();

  const period = $derived(speed <= 0.01 ? 0 : 2.4 - Math.min(speed, 1) * 2.0);
  const angles = $derived(Array.from({length: blades}, (_, i) => (i * 360) / blades));
  // One blade in unit space (hub at 0,0, tip near radius 1), swept forward.
  const BLADE = "M0.18 -0.04 C0.32 -0.42 0.62 -0.78 0.98 -0.84 C0.86 -0.5 0.62 -0.16 0.24 0.12 Z";
  const EDGE = "M0.18 -0.04 C0.32 -0.42 0.62 -0.78 0.98 -0.84 C0.9 -0.74 0.58 -0.48 0.28 -0.02 Z";
</script>

<g class="rotor" style="transform-origin: {cx}px {cy}px; animation-duration: {period || 1}s; animation-play-state: {period ? 'running' : 'paused'}">
  {#each angles as a}
    <g transform="translate({cx} {cy}) rotate({a}) scale({r})">
      <path d={BLADE} fill="#3b404b" />
      <path d={EDGE} fill="#4b515e" />
    </g>
  {/each}
</g>
<circle {cx} {cy} r={r * 0.3} fill="#20232a" stroke="#0b0c0f" stroke-width={r * 0.04} />
<circle {cx} {cy} r={r * 0.2} fill="#2b2f37" />
<circle {cx} {cy} r={r * 0.07} fill="var(--accent)" />
