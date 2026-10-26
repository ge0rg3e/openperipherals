<script lang="ts">
  // Flat top-down SABRE built from solid facets (no gradients). Zones:
  // underglow, logo, DPI indicator, scroll wheel. With an animated `sync`,
  // the zones play the effect the app streams to the real mouse.
  import {onMount} from "svelte";
  import type {SyncLighting} from "../../bindings/openperipherals";
  import {hexToRgb, ledColor, rgbToHex, scale} from "../lib/effects";

  let {colors, sync = null, brightness = 100, interactive = false}:
    {colors: string[]; sync?: SyncLighting | null; brightness?: number; interactive?: boolean} = $props();

  const DPI = 2;
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

<div class="mouse-art" onpointermove={move} onpointerleave={() => (tilt = {x: 0, y: 0})} role="img" aria-label="Mouse lighting preview">
  <svg viewBox="0 0 220 360" style="transform: rotateX({(interactive ? 8 : 0) + tilt.x}deg) rotateY({tilt.y}deg)" aria-hidden="true">
    <!-- underglow: lit rim around the base and the light bar under the front -->
    <path d={SHELL} transform="translate(110 179) scale(1.07) translate(-110 -179)" fill="none" stroke={zone(0)} stroke-width="4" stroke-opacity="0.55" />
    <path d={FRONT_BAR} transform="translate(0 -10)" fill="none" stroke={zone(0)} stroke-width="5" stroke-linecap="round" />

    <!-- body -->
    <path d={SHELL} fill="#17191e" stroke="#2a2d35" stroke-width="2" />
    <path d={PALM_FACET} fill="#1c1f25" />
    <path d={LEFT} fill="#22252c" />
    <path d={RIGHT} fill="#1f2228" />
    <path d="M42 150 H178" stroke="#0b0c0f" stroke-width="2.5" />
    <path d="M110 20 V150" stroke="#0b0c0f" stroke-width="3" />

    <!-- side buttons -->
    <rect x="31" y="150" width="8" height="34" rx="4" fill="#2c3039" />
    <rect x="31" y="190" width="8" height="34" rx="4" fill="#2c3039" />

    <!-- wheel channel, wheel, DPI buttons -->
    <rect x="98" y="40" width="24" height="74" rx="12" fill="#0b0c0f" />
    <rect x="102" y="46" width="16" height="62" rx="8" fill={zone(3)} />
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
