<script lang="ts">
  // Contoured SABRE with soft shell highlights and live lighting zones:
  // underglow, logo, DPI indicator, scroll wheel. With an animated `sync`,
  // the zones play the effect the app streams to the real mouse.
  import {onMount} from "svelte";
  import type {SyncLighting} from "../../bindings/openperipherals";
  import {hexToRgb, ledColor, rgbToHex, scale} from "../lib/effects";

  let {colors, sync = null, brightness = 100, interactive = false}:
    {colors: string[]; sync?: SyncLighting | null; brightness?: number; interactive?: boolean} = $props();

  const DPI = 2;
  const id = $props.id();
  const POS = [0.45, 0.7, 0.2, 0.1]; // zone positions front-to-back, as in the backend
  const live = $derived(!!sync?.enabled && ["spectrum", "wave", "breathing"].includes(sync.effect));
  let frame = $state<string[]>([]);

  onMount(() => {
    let raf = 0, last = 0;
    const tick = (now: number) => {
      raf = requestAnimationFrame(tick);
      if (!live || now - last < 1000 / 30) return;
      last = now;
      const l = {effect: sync!.effect, color: sync!.color, color2: sync!.color, mode: "single", speed: 2, direction: "right"};
      frame = POS.map((p, i) => (i === DPI ? colors[DPI] : rgbToHex(ledColor(l, p, 0, now / 1000))));
    };
    raf = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(raf);
  });

  // Brightness dims the lighting zones like the real mouse; the DPI LED stays full.
  const zone = (i: number) => {
    const c = (live && frame.length ? frame[i] : colors[i]) ?? "#000000";
    return i === DPI ? c : rgbToHex(scale(hexToRgb(c), brightness / 100));
  };

  let tilt = $state({x: 0, y: 0});
  function move(e: PointerEvent) {
    if (!interactive) return;
    const r = (e.currentTarget as HTMLElement).getBoundingClientRect();
    tilt = {x: ((e.clientY - r.top) / r.height - 0.5) * -14, y: ((e.clientX - r.left) / r.width - 0.5) * 18};
  }

  const SHELL = "M110 16 C152 16 178 40 184 94 C190 152 190 214 178 264 C166 314 140 342 110 342 C80 342 54 314 42 266 C30 214 30 152 36 96 C42 40 68 16 110 16 Z";
  const LEFT = "M106 20 C76 22 50 40 42 94 C39 116 38 134 39 150 L106 150 Z";
  const RIGHT = "M114 20 C144 22 170 40 178 94 C181 116 182 134 181 150 L114 150 Z";
  const PALM_FACET = "M114 156 L181 156 C182 200 180 236 172 268 C150 252 126 236 114 220 Z";
  const FRONT_BAR = "M52 46 C70 26 150 26 168 46";
</script>

<div class="grid h-full aspect-[220/360] place-items-center [perspective:800px]" onpointermove={move} onpointerleave={() => (tilt = {x: 0, y: 0})} role="img" aria-label="Mouse lighting preview">
  <svg class="h-full w-auto overflow-visible transition-transform duration-300" viewBox="0 0 220 360" style="transform: rotateX({(interactive ? 8 : 0) + tilt.x}deg) rotateY({tilt.y}deg)" aria-hidden="true">
    <defs>
      <radialGradient id="{id}-shell" cx=".35" cy=".3" r=".78">
        <stop stop-color="#49515e" /><stop offset=".4" stop-color="#2b323e" /><stop offset=".8" stop-color="#181e28" /><stop offset="1" stop-color="#0c1119" />
      </radialGradient>
      <linearGradient id="{id}-button" x1="0" y1="0" x2=".85" y2="1">
        <stop stop-color="#626d7d" /><stop offset=".2" stop-color="#3b4554" /><stop offset=".7" stop-color="#252d39" /><stop offset="1" stop-color="#1a212b" />
      </linearGradient>
      <linearGradient id="{id}-edge" x2="1" y2="1">
        <stop stop-color="#a5b3c4" stop-opacity=".65" /><stop offset=".5" stop-color="#657286" stop-opacity=".12" /><stop offset="1" stop-color="#8393a8" stop-opacity=".25" />
      </linearGradient>
      <filter id="{id}-glow" x="-60%" y="-30%" width="220%" height="160%"><feGaussianBlur stdDeviation="5" /></filter>
    </defs>
    <path d={SHELL} transform="translate(0 7)" fill="#05080d" />
    <path d={SHELL} fill="none" stroke={zone(0)} stroke-width="7" opacity=".24" filter="url(#{id}-glow)" />
    <!-- underglow: lit rim around the base and the light bar under the front -->
    <path d={SHELL} transform="translate(110 179) scale(1.07) translate(-110 -179)" fill="none" stroke={zone(0)} stroke-width="4" stroke-opacity="0.55" />
    <path d={FRONT_BAR} transform="translate(0 -10)" fill="none" stroke={zone(0)} stroke-width="5" stroke-linecap="round" />

    <!-- body -->
    <path d={SHELL} fill="url(#{id}-shell)" stroke="url(#{id}-edge)" stroke-width="2" />
    <path d={PALM_FACET} fill="#090f19" opacity=".16" />
    <path d={LEFT} fill="url(#{id}-button)" stroke="#a2b0c3" stroke-opacity=".15" />
    <path d={RIGHT} fill="url(#{id}-button)" stroke="#a2b0c3" stroke-opacity=".1" />
    <path d="M48 139 C47 89 57 51 84 37 M48 179 C48 238 61 283 83 303" fill="none" stroke="#b1bfd0" stroke-opacity=".18" stroke-width="2" stroke-linecap="round" />
    {#each Array(12) as _, i}
      <path d="M{41 + i * .25} {191 + i * 6} l9 3" stroke="#060b12" stroke-opacity=".6" stroke-width="2" />
    {/each}
    <path d="M42 150 H178" stroke="#0b0c0f" stroke-width="2.5" />
    <path d="M110 20 V150" stroke="#0b0c0f" stroke-width="3" />

    <!-- side buttons -->
    <rect x="31" y="150" width="8" height="34" rx="4" fill="#2c3039" />
    <rect x="31" y="190" width="8" height="34" rx="4" fill="#2c3039" />

    <!-- wheel channel, wheel, DPI buttons -->
    <rect x="98" y="40" width="24" height="74" rx="12" fill="#0b0c0f" />
    <rect x="100" y="44" width="20" height="66" rx="10" fill="#566170" />
    <rect x="103" y="47" width="14" height="60" rx="7" fill={zone(3)} />
    <path d="M106 51 V102" stroke="#fff" stroke-opacity=".25" stroke-width="2" />
    {#each [56, 64, 72, 80, 88, 96] as y}
      <rect x="102" y={y} width="16" height="2.5" fill="#0b0c0f" opacity="0.4" />
    {/each}
    <rect x="103" y="120" width="14" height="9" rx="3" fill="#2c3039" />
    <rect x="103" y="133" width="14" height="9" rx="3" fill="#2c3039" />

    <!-- DPI indicator -->
    <rect x="66" y="66" width="22" height="7" rx="3.5" fill={zone(2)} />

    <!-- logo: generic emblem -->
    <g transform="translate(110 262)" fill={zone(1)}>
      <path d="M-20 -8 L0 -22 L20 -8 L20 -1 L0 -15 L-20 -1 Z" />
      <path d="M-20 6 L0 -8 L20 6 L20 13 L0 -1 L-20 13 Z" />
    </g>
  </svg>
</div>
