<script lang="ts">
  // A keyboard in CSS 3D: a plate with an underglow strip and raised caps
  // whose backlight follows the simulated effect. The
  // layout is the Huntsman Elite's, cut to the board size. With `onselect`,
  // caps and the dial are clickable.
  import {onMount} from "svelte";
  import type {KeyboardLighting} from "../../bindings/openperipherals";
  import {DIAL, layoutFor, type Cap} from "../lib/layout";
  import {BLACK, isAnimated, keySeed, ledColor, rgbToHex, scale, type RGB} from "../lib/effects";

  let {lighting, brightness, layout = "full", media = false, interactive = false, disabled = {}, selected = "", onselect}: {
    lighting: KeyboardLighting;
    brightness: number;
    layout?: string;
    media?: boolean; // Huntsman Elite media keys and dial
    interactive?: boolean;
    disabled?: Record<string, number>;
    selected?: string;
    onselect?: (id: string, label: string, anchor: DOMRect) => void;
  } = $props();

  const PAD = 0.6; // plate margin around the keys, in units
  const SEGMENTS = 30; // underglow strip resolution
  const LEGEND_FLOOR: RGB = [70, 76, 88]; // unlit legends stay readable

  const board = $derived(layoutFor(layout, media));
  const WIDTH = $derived(board.width);
  const HEIGHT = $derived(board.height);

  let width = $state(0);
  const u = $derived(width / (WIDTH + PAD * 2 + 0.6));

  const seeds = $derived(board.caps.map((c) => keySeed(c.x, c.y)));
  const pos = $derived(board.caps.map((c) => (c.x + c.w / 2) / WIDTH));
  const yPos = $derived(board.caps.map((c) => (c.y + c.h / 2) / HEIGHT));
  const caps = $state<HTMLElement[]>([]);
  const edge = $state<HTMLElement[]>([]);
  let dial = $state<HTMLElement>();
  const pressed = new Map<string, number>();

  // Plain copies read by the render loop; the effect below keeps them current.
  // svelte-ignore state_referenced_locally
  let current = $state.snapshot(lighting);
  // svelte-ignore state_referenced_locally
  let level = brightness / 100;

  // Legends glow: the LED colour lifted toward white so text stays crisp.
  const legend = (c: RGB): RGB => {
    const peak = Math.max(...c);
    if (peak < 40) return LEGEND_FLOOR;
    const k = Math.min(255 / peak, 2.2);
    return [0, 1, 2].map((j) => Math.max(Math.min(c[j] * k + 40, 255), LEGEND_FLOOR[j])) as RGB;
  };

  function paint(now: number) {
    const t = now / 1000;
    const led = (x: number, y: number, seed: number, since = Infinity): RGB =>
      current.effect === "off" ? BLACK : scale(ledColor(current, x, seed, t, since, y), level);
    board.caps.forEach((c, i) => {
      const el = caps[i];
      if (!el) return;
      const col = led(pos[i], yPos[i], seeds[i], (now - (pressed.get(c.id) ?? -1e9)) / 1000);
      el.style.setProperty("--c", rgbToHex(col));
      el.style.setProperty("--l", rgbToHex(legend(col)));
    });
    edge.forEach((el, s) => el?.style.setProperty("--c", rgbToHex(led((s + 0.5) / SEGMENTS, 1, keySeed(s, 99)))));
    dial?.style.setProperty("--c", rgbToHex(led(DIAL.x / WIDTH, DIAL.y / HEIGHT, keySeed(DIAL.x, DIAL.y))));
  }

  $effect(() => {
    current = $state.snapshot(lighting);
    level = Math.min(Math.max(brightness, 0), 100) / 100;
    // Resizing rewrites each key's inline geometry. Paint after that DOM update
    // so static lighting also survives the initial measured layout and resizes.
    if (!width || !board.caps.length) return;
    const frame = requestAnimationFrame(paint);
    return () => cancelAnimationFrame(frame);
  });

  onMount(() => {
    let raf = 0;
    let last = 0;
    const tick = (now: number) => {
      raf = requestAnimationFrame(tick);
      if (!isAnimated(current) || now - last < 1000 / 30) return;
      last = now;
      paint(now);
    };
    raf = requestAnimationFrame(tick);
    const down = (e: KeyboardEvent) => pressed.set(e.code, performance.now());
    window.addEventListener("keydown", down);
    return () => {
      cancelAnimationFrame(raf);
      window.removeEventListener("keydown", down);
    };
  });

  let tilt = $state({x: 0, z: 0});
  function move(e: PointerEvent) {
    if (!interactive) return;
    const r = (e.currentTarget as HTMLElement).getBoundingClientRect();
    tilt = {x: ((e.clientY - r.top) / r.height - 0.5) * -6, z: ((e.clientX - r.left) / r.width - 0.5) * 4};
  }

  function pick(e: MouseEvent, id: string, label: string) {
    if (!onselect) return;
    e.stopPropagation();
    onselect(id, label, (e.currentTarget as HTMLElement).getBoundingClientRect());
  }

  const isDialDisabled = $derived(Object.keys(disabled).some((k) => k.startsWith("dial:")));
  const capLabel = (c: Cap) => c.label || (c.id === "Space" ? "Space" : c.id);
</script>

<div class="flex w-full items-center justify-center [perspective:1500px]" bind:clientWidth={width}
     style="--u: {u}px; height: {u * (HEIGHT + PAD * 2) * 0.86}px"
     onpointermove={move} onpointerleave={() => (tilt = {x: 0, z: 0})} role="img" aria-label="Keyboard lighting preview">
  <div class="relative flex-none rounded-[calc(var(--u)*.35)] border border-[#24262c] bg-[#0d0e10] [transform-style:preserve-3d] transition-transform duration-300 ease-out"
       style="width: {u * (WIDTH + PAD * 2)}px; height: {u * (HEIGHT + PAD * 2)}px;
       transform: rotateX({38 + tilt.x}deg) rotateZ({tilt.z}deg)">
    <div class="absolute top-full right-[calc(var(--u)*.35)] left-[calc(var(--u)*.35)] flex h-[calc(var(--u)*.3)] origin-top [transform:rotateX(-90deg)]">
      {#each Array(SEGMENTS) as _, s}<i class="flex-1 bg-[var(--c,#000)]" bind:this={edge[s]}></i>{/each}
    </div>
    {#each board.caps as c, i (c.id)}
      {@const blocked = c.id in disabled}
      {@const active = selected === c.id}
      <div class={`absolute [transform-style:preserve-3d] ${c.round ? "rounded-full" : "rounded-[calc(var(--u)*.1)]"} ${onselect ? "group cursor-pointer" : ""}`}
           bind:this={caps[i]} onclick={(e) => pick(e, c.id, capLabel(c))} role="presentation"
           style="left: {(c.x + PAD) * u}px; top: {(c.y + PAD) * u}px; width: {c.w * u}px; height: {c.h * u}px">
        <div class={`absolute inset-[calc(var(--u)*.13)] grid place-items-center [transform:translateZ(calc(var(--u)*.22))] ${c.round ? "rounded-full" : "rounded-[calc(var(--u)*.08)]"} ${blocked ? "bg-destructive/15 outline outline-1 outline-destructive" : "bg-[color-mix(in_srgb,var(--c,#000)_10%,#222429)]"} ${active ? "outline outline-2 outline-foreground" : ""} ${onselect ? "group-hover:bg-[color-mix(in_srgb,var(--c,#000)_20%,#2a2d34)]" : ""} transition-colors duration-150`}>
          <span class={blocked ? "font-bold tracking-[.02em] whitespace-nowrap text-destructive" : "font-bold tracking-[.02em] whitespace-nowrap text-[var(--l,#464c58)]"} style="font-size: calc(var(--u)*.25)">{c.label}</span>
          <i class="absolute top-full right-0 left-0 h-[calc(var(--u)*.22)] origin-top [transform:rotateX(-90deg)] bg-[color-mix(in_srgb,var(--c,#000)_16%,#131418)]"></i>
        </div>
      </div>
    {/each}
    {#if board.dial}
      <div class={`absolute rounded-full [transform-style:preserve-3d] ${isDialDisabled ? "bg-destructive/15" : "bg-[#050506]"} ${onselect ? "cursor-pointer" : ""}`} bind:this={dial}
           onclick={(e) => pick(e, "dial", "Dial")} role="presentation"
           style="left: {(DIAL.x - DIAL.r + PAD) * u}px; top: {(DIAL.y - DIAL.r + PAD) * u}px; width: {DIAL.r * 2 * u}px; height: {DIAL.r * 2 * u}px">
        <div class={`absolute inset-0 rounded-full [transform:translateZ(calc(var(--u)*.4))] border-[calc(var(--u)*.08)] ${isDialDisabled ? "border-destructive bg-destructive/15" : "border-[var(--c,#000)] bg-[#26282e]"} ${selected === "dial" ? "outline outline-2 outline-foreground" : ""}`}></div>
      </div>
    {/if}
  </div>
</div>
