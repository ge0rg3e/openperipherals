import type { EffectKind, EffectParams } from './controller';
import type { MatrixKey } from './keyboard/layout';

export type RGB3 = [number, number, number];

export function parseHex(hex?: string): RGB3 {
	const m = /^#?([0-9a-f]{2})([0-9a-f]{2})([0-9a-f]{2})$/i.exec((hex ?? '#ffffff').trim());
	if (!m) return [255, 255, 255];
	return [parseInt(m[1], 16), parseInt(m[2], 16), parseInt(m[3], 16)];
}

function mix(a: RGB3, b: RGB3, t: number): RGB3 {
	const k = Math.max(0, Math.min(1, t));
	return [a[0] + (b[0] - a[0]) * k, a[1] + (b[1] - a[1]) * k, a[2] + (b[2] - a[2]) * k];
}

function hex(a: RGB3): string {
	return '#' + (Math.round(a[0]) & 255).toString(16).padStart(2, '0') + (Math.round(a[1]) & 255).toString(16).padStart(2, '0') + (Math.round(a[2]) & 255).toString(16).padStart(2, '0');
}

/** HSV → RGB (0..255 channels), h in [0,1). Used for the spectrum-cycle preview. */
function hsv2rgb(h: number, s: number, v: number): RGB3 {
	const i = Math.floor(h * 6) % 6;
	const f = h * 6 - Math.floor(h * 6);
	const vv = v * 255;
	const p = vv * (1 - s);
	const q = vv * (1 - f * s);
	const t = vv * (1 - (1 - f) * s);
	switch (i) {
		case 0:
			return [vv, t, p];
		case 1:
			return [q, vv, p];
		case 2:
			return [p, vv, t];
		case 3:
			return [p, q, vv];
		case 4:
			return [t, p, vv];
		default:
			return [vv, p, q];
	}
}

/** Colour state used to scatter starlight/ripple deterministically per key. */
export interface PreviewState {
	minCol: number;
	maxCol: number;
	/** ms timestamp (advancing) used for animation. */
	time: number;
}

/**
 * Preview colour for keys that are "off" (Off effect or unpainted in custom
 * mode). Not pure black: the tint lights the legend, so a dim grey keeps the
 * key text faintly readable instead of leaving caps indistinguishable.
 */
export const OFF_KEY_COLOR = '#262b33';

/**
 * Return the live colour for one key at time `st.time` under the effect `p`.
 * `press` maps a key code to the ms timestamp of its last press, used by
 * `reactive` so the preview flashes the actual pressed key (mirroring the
 * hardware) instead of animating a synthetic wave.
 */
export function effectColorOf(kind: EffectKind, p: EffectParams, st: PreviewState, key: MatrixKey, press?: Record<string, number>): string {
	const t = st.time / 1000;
	const color1 = parseHex(p.color);
	const color2 = parseHex(p.color2);
	const speed = p.speed ?? 3; // 1 fast … 4 slowest (hardware: only starlight + reactive)
	// Wave, spectrum and breathing have no speed argument on the extended-matrix
	// command (the firmware drives a fixed rate), so the preview mirrors that
	// constant instead of scaling with the slider.
	const wavePeriod = 3.4; // seconds per full sweep
	const rel = (key.col - st.minCol) / Math.max(1, st.maxCol - st.minCol);
	const dir = p.direction === 'left' ? -1 : 1;

	switch (kind) {
		case 'off':
			return OFF_KEY_COLOR;

		case 'static':
			return p.color ?? '#ffffff';

		case 'wave': {
			// phase advances in +rel for dir=+1 (right) - i.e. a colour feature
			// at a fixed angle drifts toward higher rel as time passes, which
			// reads as the sweep moving right. The double modulo keeps the
			// negative side wrapping into [0,1) instead of going negative.
			const pos = (((rel - (t / wavePeriod) * dir) % 1) + 1) % 1;
			const angle = 0 + pos * Math.PI * 2;
			return hex([Math.abs(Math.sin(angle)) * 255, Math.abs(Math.sin(angle + 2)) * 255, Math.abs(Math.sin(angle + 4)) * 255]);
		}

		case 'spectrum': {
			// Hardware "Spectrum Cycling" drives every LED through the colour
			// spectrum in unison (fixed firmware rate, no speed argument) - the
			// moving rainbow is the separate "Wave" effect.
			const hue = (t / 8) % 1;
			return hex(hsv2rgb(hue, 1, 1));
		}

		case 'wheel': {
			// "Wheel" spins the full colour wheel across the board - visually a
			// hue that sweeps along the keys, like wave but built from HSV.
			const dir = p.direction === 'left' ? -1 : 1;
			const pos = (((rel - (t / 2.6) * dir) % 1) + 1) % 1;
			return hex(hsv2rgb(pos, 1, 1));
		}

		case 'reactive': {
			const base: RGB3 = [26, 30, 34];
			const last = press?.[key.code] ?? 0;
			const age = st.time - last;
			const fade = 900 + speed * 250; // speed 1 (fast) → ~1.2s, 4 (slowest) → ~1.9s
			const k = age < 0 || age >= fade ? 0 : 1 - age / fade;
			return hex(mix(base, color1, k));
		}

		case 'breathing': {
			const o = (Math.sin(t * Math.PI * 0.7) + 1) / 2; // ~2.9s in/out cycle
			const col = p.mode === 'dual' ? mix(color1, color2, Math.sin(t * 1.1) * 0.5 + 0.5) : color1;
			return hex([col[0] * o, col[1] * o, col[2] * o]);
		}

		case 'starlight': {
			const base: RGB3 = [14, 14, 18];
			if (p.mode === 'random') {
				const tw = Math.max(0, (Math.sin(t * 2.2 + (key.row + key.col) * 7.3) + 1) / 2);
				return hex(mix(base, color1, tw));
			}
			if (p.mode === 'dual') {
				const pick = Math.sin(t * 1.1 + key.row * 3 + key.col) > 0 ? color1 : color2;
				const tw = Math.max(0, (Math.sin(t * 1.4 + key.col) + 1) / 2);
				return hex(mix(base, pick, tw));
			}
			const tw = Math.max(0, (Math.sin(t * 1.4 + key.col) + 1) / 2);
			return hex(mix(base, color1, tw));
		}

		case 'custom':
		default:
			return OFF_KEY_COLOR;
	}
}
