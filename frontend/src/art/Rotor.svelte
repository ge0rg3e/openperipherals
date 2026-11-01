<script lang="ts">
  // Sculpted fan rotor with a stationary light source and machined hub. Spins
  // faster as `speed` (0..1) rises; stops at 0.
  let {cx, cy, r, blades = 9, speed = 0.4}: {cx: number; cy: number; r: number; blades?: number; speed?: number} = $props();

  const period = $derived(speed <= 0.01 ? 0 : 2.4 - Math.min(speed, 1) * 2.0);
  const angles = $derived(Array.from({length: blades}, (_, i) => (i * 360) / blades));
  const id = $props.id();
  // One blade in unit space (hub at 0,0, tip near radius 1), swept forward.
  const BLADE = "M.19 -.1 C.28 -.49 .58 -.79 .84 -.52 C.93 -.42 .97 -.26 .96 -.14 C.7 -.3 .44 .08 .2 .17 Z";
  const EDGE = "M.19 -.1 C.28 -.49 .58 -.79 .84 -.52 C.59 -.61 .4 -.23 .25 .04 Z";
</script>

<defs>
  <linearGradient id="{id}-blade" x1="0" y1="0" x2="1" y2="1">
    <stop stop-color="#7c8798" /><stop offset=".35" stop-color="#485362" /><stop offset=".72" stop-color="#252d39" /><stop offset="1" stop-color="#141a23" />
  </linearGradient>
  <radialGradient id="{id}-hub" cx=".35" cy=".25" r=".8">
    <stop stop-color="#596371" /><stop offset=".5" stop-color="#2e3641" /><stop offset="1" stop-color="#141a22" />
  </radialGradient>
</defs>
<g class="origin-center animate-spin motion-reduce:animate-none" style="transform-box: view-box; transform-origin: {cx}px {cy}px; animation-duration: {period || 1}s; animation-play-state: {period ? 'running' : 'paused'}">
  {#each angles as a}
    <g transform="translate({cx} {cy}) rotate({a}) scale({r})">
      <path d={BLADE} transform="translate(.015 .025)" fill="#05070a" opacity=".65" />
      <path d={BLADE} fill="url(#{id}-blade)" stroke="#8993a3" stroke-opacity=".16" stroke-width=".008" />
      <path d={EDGE} fill="#b2bfce" opacity=".16" />
    </g>
  {/each}
</g>
<circle {cx} cy={cy + r * .035} r={r * .32} fill="#05070b" opacity=".8" />
<circle {cx} {cy} r={r * 0.3} fill="url(#{id}-hub)" stroke="#697483" stroke-opacity=".4" stroke-width={r * 0.02} />
<circle {cx} {cy} r={r * 0.22} fill="none" stroke="#99a7b7" stroke-opacity=".16" stroke-width={r * .01} />
<circle {cx} {cy} r={r * .12} fill="#151b23" />
<circle {cx} {cy} r={r * 0.07} fill="var(--primary)" />
