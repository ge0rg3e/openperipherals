<script lang="ts">
  // Minimal stroke icons, 24×24.
  let {name, size = 16}: {name: keyof typeof PATHS; size?: number} = $props();

  const PATHS = {
    gear: "M12 15.5a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7Z M19.4 13a7.6 7.6 0 0 0 0-2l2-1.6-2-3.4-2.4 1a7.4 7.4 0 0 0-1.7-1L15 3.5h-4l-.4 2.5a7.4 7.4 0 0 0-1.7 1l-2.4-1-2 3.4L6.6 11a7.6 7.6 0 0 0 0 2l-2 1.6 2 3.4 2.4-1a7.4 7.4 0 0 0 1.7 1l.4 2.5h4l.4-2.5a7.4 7.4 0 0 0 1.7-1l2.4 1 2-3.4Z",
    refresh: "M20 11a8 8 0 0 0-14.6-4.5L4 8 M4 4v4h4 M4 13a8 8 0 0 0 14.6 4.5L20 16 M20 20v-4h-4",
    back: "M15 5l-7 7 7 7",
    panel: "M4 5h16v14H4z M9 5v14",
    keyboard: "M3 7h18v10H3z M6 10h1 M9 10h1 M12 10h1 M15 10h1 M18 10h0 M7 14h10",
    mouse: "M12 3a6 6 0 0 0-6 6v6a6 6 0 0 0 12 0V9a6 6 0 0 0-6-6Z M12 3v6 M6 9h12",
    fan: "M12 12m-2 0a2 2 0 1 0 4 0 2 2 0 1 0-4 0 M12 10c0-4 1-7 4-7 2 0 2 3-1 5 M14 12c4 0 7 1 7 4 0 2-3 2-5-1 M12 14c0 4-1 7-4 7-2 0-2-3 1-5 M10 12c-4 0-7-1-7-4 0-2 3-2 5 1",
    sliders: "M4 6h10 M18 6h2 M4 12h4 M12 12h8 M4 18h12 M20 18h0 M16 4v4 M10 10v4 M18 16v4",
    temp: "M10 13.5V5a2 2 0 1 1 4 0v8.5a4 4 0 1 1-4 0Z",
    bolt: "M13 3L5 14h6l-1 7 8-11h-6l1-7Z",
    chip: "M7 7h10v10H7z M10 3v4 M14 3v4 M10 17v4 M14 17v4 M3 10h4 M3 14h4 M17 10h4 M17 14h4",
    memory: "M3 8h18v8H3z M7 8v8 M11 8v8 M15 8v8 M19 8v8",
    plus: "M12 5v14 M5 12h14",
    shield: "M12 3l7 3v5c0 4.5-3 8.3-7 10-4-1.7-7-5.5-7-10V6l7-3Z M12 8v5 M12 16v.5",
    minimize: "M6 12h12",
    maximize: "M6.5 6.5h11v11h-11z",
    restore: "M6.5 9.5h8v8h-8z M9.5 9.5V6.5h8v8h-3",
    close: "M7 7l10 10 M17 7L7 17",
  } as const;
</script>

<svg class="icon" width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor"
     stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
  <path d={PATHS[name]} />
</svg>
