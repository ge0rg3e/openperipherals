// Approximates the keyboard's hardware lighting effects so the previews move
// the way the real board does. Every function is pure: colour from effect
// settings, a key's position (0..1 across the board) and the time.
import type {KeyboardLighting} from "../../bindings/openperipherals";

export type RGB = [number, number, number];

export const BLACK: RGB = [0, 0, 0];

export function hexToRgb(hex: string | undefined): RGB {
  const v = parseInt((hex ?? "").replace("#", ""), 16);
  return Number.isNaN(v) ? [255, 255, 255] : [(v >> 16) & 255, (v >> 8) & 255, v & 255];
}

export function rgbToHex(c: RGB): string {
  return "#" + c.map((v) => Math.round(Math.min(255, Math.max(0, v))).toString(16).padStart(2, "0")).join("");
}

export const scale = (c: RGB, k: number): RGB => [c[0] * k, c[1] * k, c[2] * k];

/** Fully saturated colour around the colour wheel, h in turns. */
function hue(h: number): RGB {
  const f = (n: number) => {
    const k = (n + h * 6) % 6;
    return 255 * (1 - Math.max(0, Math.min(k, 4 - k, 1)));
  };
  return [f(5), f(3), f(1)];
}

const frac = (v: number) => v - Math.floor(v);

/** Stable pseudo-random value in [0, 1) for a number. */
const noise = (n: number) => frac(Math.sin(n * 91.345 + 47.853) * 24634.6345);

const BREATH = 3.2; // seconds per breathing cycle

/** Whether the effect changes over time. */
export const isAnimated = (l: KeyboardLighting) => l.effect !== "static" && l.effect !== "off";

/**
 * Colour of one LED. `pos` is its horizontal position across the board
 * (0..1), `seed` a per-key constant, `t` seconds, `sincePress` seconds since
 * the key was last pressed (Infinity if never).
 */
export function ledColor(l: KeyboardLighting, pos: number, seed: number, t: number, sincePress = Infinity): RGB {
  const c1 = hexToRgb(l.color);
  const c2 = hexToRgb(l.color2);
  switch (l.effect) {
    case "static":
      return c1;
    case "spectrum":
      return hue(t / 6);
    case "wave":
    case "wheel": {
      const dir = l.direction === "left" ? 1 : -1;
      return hue(pos + (dir * t) / 4);
    }
    case "breathing": {
      const cycle = Math.floor(t / BREATH);
      const level = (1 - Math.cos((2 * Math.PI * t) / BREATH)) / 2;
      const base = l.mode === "random" ? hue(noise(cycle)) : l.mode === "dual" && cycle % 2 ? c2 : c1;
      return scale(base, level);
    }
    case "reactive": {
      const fade = 0.35 * Math.max(1, l.speed || 2);
      return scale(c1, Math.max(0, 1 - sincePress / fade));
    }
    case "starlight": {
      const period = 3.2 - Math.min(Math.max(l.speed || 2, 1), 3) * 0.6;
      const phase = t / period + seed;
      const p = frac(phase);
      const level = p < 0.3 ? Math.sin((p / 0.3) * Math.PI) : 0;
      const base = l.mode === "random" ? hue(noise(seed * 13 + Math.floor(phase))) : l.mode === "dual" && seed < 0.5 ? c2 : c1;
      return scale(base, level);
    }
    default:
      return BLACK;
  }
}

/** Per-key constant used to scatter starlight twinkles. */
export const keySeed = (x: number, y: number) => noise(x * 7.13 + y * 3.71);
