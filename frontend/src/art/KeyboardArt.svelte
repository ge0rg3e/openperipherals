<script lang="ts">
  // A keyboard in CSS 3D: a plate with an underglow strip and raised caps
  // whose backlight follows the simulated effect. Flat colours only. The
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
  const caps: HTMLElement[] = [];
  const edge: HTMLElement[] = [];
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
    const led = (p: number, seed: number, since = Infinity): RGB =>
      current.effect === "off" ? BLACK : scale(ledColor(current, p, seed, t, since), level);
    board.caps.forEach((c, i) => {
      const el = caps[i];
      if (!el) return;
      const col = led(pos[i], seeds[i], (now - (pressed.get(c.id) ?? -1e9)) / 1000);
      el.style.setProperty("--c", rgbToHex(col));
      el.style.setProperty("--l", rgbToHex(legend(col)));
    });
    edge.forEach((el, s) => el?.style.setProperty("--c", rgbToHex(led((s + 0.5) / SEGMENTS, keySeed(s, 99)))));
    dial?.style.setProperty("--c", rgbToHex(led(DIAL.x / WIDTH, keySeed(DIAL.x, DIAL.y))));
  }

  $effect(() => {
    current = $state.snapshot(lighting);
    level = Math.min(Math.max(brightness, 0), 100) / 100;
    paint(performance.now());
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

<div class="kbd" class:pickable={!!onselect} bind:clientWidth={width}
     style="--u: {u}px; height: {u * (HEIGHT + PAD * 2) * 0.86}px"
     onpointermove={move} onpointerleave={() => (tilt = {x: 0, z: 0})} role="img" aria-label="Keyboard lighting preview">
  <div class="kbd-board" style="width: {u * (WIDTH + PAD * 2)}px; height: {u * (HEIGHT + PAD * 2)}px;
       transform: rotateX({38 + tilt.x}deg) rotateZ({tilt.z}deg)">
    <div class="kbd-edge">
      {#each Array(SEGMENTS) as _, s}<i bind:this={edge[s]}></i>{/each}
    </div>
    {#each board.caps as c, i (c.id)}
      <div class="cap" class:round={c.round} class:blocked={c.id in disabled} class:selected={selected === c.id}
           bind:this={caps[i]} onclick={(e) => pick(e, c.id, capLabel(c))} role="presentation"
           style="left: {(c.x + PAD) * u}px; top: {(c.y + PAD) * u}px; width: {c.w * u}px; height: {c.h * u}px">
        <div class="cap-top"><span>{c.label}</span><i class="cap-side"></i></div>
      </div>
    {/each}
    {#if board.dial}
      <div class="dial" class:blocked={isDialDisabled} class:selected={selected === "dial"} bind:this={dial}
           onclick={(e) => pick(e, "dial", "Dial")} role="presentation"
           style="left: {(DIAL.x - DIAL.r + PAD) * u}px; top: {(DIAL.y - DIAL.r + PAD) * u}px; width: {DIAL.r * 2 * u}px; height: {DIAL.r * 2 * u}px">
        <div class="dial-top"></div>
      </div>
    {/if}
  </div>
</div>
