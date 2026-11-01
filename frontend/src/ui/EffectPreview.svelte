<script lang="ts">
  import {onMount} from "svelte";
  import {isAnimated, keySeed, ledColor, rgbToHex} from "../lib/effects";
  import {animatePreviews, previewClock} from "../lib/preview-clock.svelte";

  let {effect: effectId, color = "#33e19a", color2 = "#1f7bff", speed = 2, direction = "right", mode = "single"}: {
    effect: string; color?: string; color2?: string; speed?: number; direction?: string; mode?: string;
  } = $props();
  const lighting = $derived({effect: effectId, color, color2, speed, direction, mode});
  let board: SVGSVGElement;
  let visible = $state(false);
  let reducedMotion = $state(true);
  const keys = [
    ...Array.from({length: 36}, (_, i) => ({x: 5 + (i % 12) * 8.25, y: 5 + Math.floor(i / 12) * 7, w: 6.2})),
    ...[{x: 5, w: 10}, {x: 17, w: 10}, {x: 29, w: 45}, {x: 76, w: 10}, {x: 88, w: 13}].map(k => ({...k, y: 26})),
  ];
  const time = $derived(visible && !reducedMotion && isAnimated(lighting) ? previewClock.time : 1.4);
  const colors = $derived(keys.map(k => {
    const pos = (k.x + k.w / 2) / 108;
    const y = k.y / 31;
    // Auto-demo a key press for reactive previews; the real keyboard remains untouched.
    const pressAge = (time + keySeed(k.x, k.y) * 2) % 2.5;
    const rgb = ledColor(lighting, pos, keySeed(k.x, k.y), time, pressAge, y);
    // Retain the key silhouette when a travelling effect leaves it unlit.
    return rgbToHex(rgb.map(v => 25 + v * 0.9) as [number, number, number]);
  }));

  onMount(() => {
    const media = matchMedia("(prefers-reduced-motion: reduce)");
    const syncMotion = () => reducedMotion = media.matches;
    syncMotion();
    media.addEventListener("change", syncMotion);
    const observer = new IntersectionObserver(entries => visible = entries[0].isIntersecting);
    observer.observe(board);
    return () => {observer.disconnect(); media.removeEventListener("change", syncMotion);};
  });
  $effect(() => {
    if (visible && !reducedMotion && isAnimated(lighting)) return animatePreviews();
  });
</script>

<svg bind:this={board} class="effect-preview" viewBox="0 0 108 36" aria-hidden="true">
  <rect x="1" y="1" width="106" height="34" rx="4" fill="#101216" stroke="#343740"/>
  {#each keys as key, i}
    <rect x={key.x} y={key.y} width={key.w} height="4.6" rx="1" fill={colors[i]} />
  {/each}
  <path d="M7 33 H101" stroke={effectId === "off" ? "#343740" : colors[18]} stroke-width="1" opacity=".65"/>
</svg>

<style>
  .effect-preview { display: block; width: 100%; max-width: 104px; height: 30px; flex-shrink: 0; }
</style>
