import type { EffectParams } from '../controller';
import {
	G213_ZONE,
	G815_FRAME_TYPE,
	G915_FRAME_TYPE,
	G915_LOGO_MODE,
	G915_MODE,
	G915_ZONE_MODE,
	LOGI_MODE,
	ROMERG_ZONE_DIRECT,
	ROMERG_ZONE_MODE,
	frameBytesFor,
	type LogitechFamily
} from './constants';
import { KEY_TO_IDX, NUMPAD_CODES, type LogitechDevice } from './devices';
import {
	appSpeedToValue,
	g213Mode,
	g213SetZone,
	g815Commit,
	g815Frame,
	g815InitDirect,
	g815Mode,
	g915BeginModeSet,
	g915Commit,
	g915Frame,
	g915InitializeDirect,
	g915Mode,
	romergCommit,
	romergDirectFrame,
	romergMode,
	type LedFrame,
	type LogiReport
} from './report';

export interface EffectPackets {
	init: LogiReport[];
	packets: LogiReport[];
	commit: LogiReport[];
}

export interface CustomFrameResult {
	init: LogiReport[];
	packets: LogiReport[];
	commit: LogiReport[];
}

export const LOGITECH_MAX_LEDS_PER_PACKET = 14;
export const LOGITECH_MAX_KEYS_PER_BIG_FRAME = 13;

/** openkeyboard effect kind → Logitech hardware mode byte. */
export function mapKind(family: LogitechFamily, kind: string): number {
	switch (kind) {
		case 'off':
			return LOGI_MODE.OFF;
		case 'static':
			return LOGI_MODE.STATIC;
		case 'spectrum':
			return LOGI_MODE.CYCLE;
		case 'wave':
			return family === 'g213' || family === 'g915' ? LOGI_MODE.WAVE : LOGI_MODE.CYCLE;
		case 'reactive':
			// Ripple (reactive) is a hardware effect on the LIGHTSYNC G915;
			// every other Logitech family has no ripple engine and gets the
			// closest equivalent (colour cycle) instead.
			return family === 'g915' ? LOGI_MODE.RIPPLE : LOGI_MODE.CYCLE;
		case 'wheel':
		case 'starlight':
			return LOGI_MODE.CYCLE;
		case 'breathing':
			return LOGI_MODE.BREATHING;
		case 'custom':
			return LOGI_MODE.STATIC;
		default:
			return LOGI_MODE.STATIC;
	}
}

function hexToRgb(hex: string): [number, number, number] {
	const m = /^#?([0-9a-f]{6})$/i.exec(hex.trim());
	if (!m) return [0, 0, 0];
	const v = parseInt(m[1], 16);
	return [(v >> 16) & 0xff, (v >> 8) & 0xff, v & 0xff];
}

export function buildEffectPackets(device: LogitechDevice, sequence: EffectParams): EffectPackets {
	const speed = appSpeedToValue(sequence.speed ?? 3);
	const rgb: [number, number, number] = sequence.color ? hexToRgb(sequence.color) : [255, 255, 255];
	const mode = mapKind(device.family, sequence.kind);
	const direction = sequence.direction ?? 'right';

	switch (device.family) {
		case 'g213':
			return g213Effect(mode, speed, direction, rgb);
		case 'g915':
			return g915Effect(mode, speed, rgb);
		case 'g815':
			return {
				init: [],
				packets: [g815Mode(0x00, mode, speed, rgb), g815Mode(0x01, mode, speed, rgb)],
				commit: []
			};
		default: {
			const fb = frameBytesFor(device.family);
			return {
				init: [],
				packets: [romergMode(fb, ROMERG_ZONE_MODE.KEYBOARD, mode, speed, rgb), romergMode(fb, ROMERG_ZONE_MODE.LOGO, mode, speed, rgb)],
				commit: [romergCommit(fb)]
			};
		}
	}
}

function g213Effect(mode: number, speed: number, direction: 'left' | 'right', rgb: [number, number, number]): EffectPackets {
	if (mode === LOGI_MODE.OFF) {
		const zones = [G213_ZONE.LEFT_AREA, G213_ZONE.MIDDLE_AREA, G213_ZONE.RIGHT_AREA, G213_ZONE.ARROW_HOME, G213_ZONE.NUMPAD];
		return { init: [], packets: zones.map((z) => g213SetZone(z, [0, 0, 0])), commit: [] };
	}
	return { init: [], packets: [g213Mode(mode, speed, direction, rgb)], commit: [] };
}

function g915Effect(mode: number, speed: number, rgb: [number, number, number]): EffectPackets {
	// The logo zone uses its own (smaller) effect enum. Off/Static map straight
	// through, wave and cycle share the "cycle" logo mode, breathing has a
	// dedicated logo mode, and ripple lights the logo statically (the ripple
	// engine only covers the keyboard zone) - mirroring G HUB / OpenRGB.
	const logoMode =
		mode === G915_MODE.OFF || mode === G915_MODE.STATIC
			? mode
			: mode === G915_MODE.BREATHING
				? G915_LOGO_MODE.BREATHING
				: mode === G915_MODE.CYCLE || mode === G915_MODE.WAVE
					? G915_LOGO_MODE.CYCLE
					: G915_LOGO_MODE.STATIC;
	const packets: LogiReport[] = [];
	if (mode !== G915_MODE.OFF) {
		packets.push(g915Mode(G915_ZONE_MODE.KEYBOARD, mode, logoMode, speed, 0x00, rgb));
	}
	packets.push(g915Mode(G915_ZONE_MODE.LOGO, logoMode, logoMode, speed, 0x00, rgb));
	return { init: g915BeginModeSet(), packets, commit: [] };
}

/** Build the per-key colour frames for a custom lighting profile. */
export function buildCustomFrames(device: LogitechDevice, colors: Record<string, string>): CustomFrameResult {
	const leds: LedFrame[] = [];
	for (const [code, color] of Object.entries(colors)) {
		if (device.layout !== 'full' && NUMPAD_CODES.has(code)) continue;
		const idx = KEY_TO_IDX[code];
		if (idx === undefined) continue;
		const rgb = hexToRgb(color);
		leds.push({ idx, r: rgb[0], g: rgb[1], b: rgb[2] });
	}

	if (device.family === 'g213') return g213Custom(leds);
	if (device.family === 'g815' || device.family === 'g915') return lightsyncCustom(device.family, leds);
	return romergCustom(device.family, leds);
}

function g213Custom(leds: LedFrame[]): CustomFrameResult {
	const color: [number, number, number] = leds.length > 0 ? [leds[0].r, leds[0].g, leds[0].b] : [0, 0, 0];
	const zones = [G213_ZONE.LEFT_AREA, G213_ZONE.MIDDLE_AREA, G213_ZONE.RIGHT_AREA, G213_ZONE.ARROW_HOME, G213_ZONE.NUMPAD];
	return { init: [], packets: zones.map((z) => g213SetZone(z, color)), commit: [] };
}

function romergCustom(family: LogitechFamily, leds: LedFrame[]): CustomFrameResult {
	const fb = frameBytesFor(family);
	const packets: LogiReport[] = [];
	for (let i = 0; i < leds.length; i += LOGITECH_MAX_LEDS_PER_PACKET) {
		const chunk = leds.slice(i, i + LOGITECH_MAX_LEDS_PER_PACKET);
		packets.push(romergDirectFrame(fb, ROMERG_ZONE_DIRECT.KEYBOARD, chunk));
	}
	return { init: [], packets, commit: [romergCommit(fb)] };
}

function lightsyncCustom(family: 'g815' | 'g915', leds: LedFrame[]): CustomFrameResult {
	const init = family === 'g915' ? g915InitializeDirect() : g815InitDirect();
	const commit = family === 'g915' ? [g915Commit()] : [g815Commit()];
	const packets: LogiReport[] = [];
	const little = family === 'g915' ? G915_FRAME_TYPE.LITTLE : G815_FRAME_TYPE.LITTLE;
	const big = family === 'g915' ? G915_FRAME_TYPE.BIG : G815_FRAME_TYPE.BIG;

	const byColor = new Map<string, { rgb: [number, number, number]; keys: number[] }>();
	for (const led of leds) {
		const key = `${led.r},${led.g},${led.b}`;
		const entry = byColor.get(key) ?? { rgb: [led.r, led.g, led.b] as [number, number, number], keys: [] };
		entry.keys.push(transformIdx(family, led.idx));
		byColor.set(key, entry);
	}

	for (const entry of byColor.values()) {
		if (entry.keys.length <= 4) {
			const data: number[] = [];
			for (const key of entry.keys) data.push(key, entry.rgb[0], entry.rgb[1], entry.rgb[2]);
			data.push(0xff);
			packets.push(family === 'g915' ? g915Frame(little, data) : g815Frame(little, data));
		} else {
			for (let i = 0; i < entry.keys.length; i += LOGITECH_MAX_KEYS_PER_BIG_FRAME) {
				const chunk = entry.keys.slice(i, i + LOGITECH_MAX_KEYS_PER_BIG_FRAME);
				const data = [entry.rgb[0], entry.rgb[1], entry.rgb[2], ...chunk, 0xff];
				packets.push(family === 'g915' ? g915Frame(big, data) : g815Frame(big, data));
			}
		}
	}
	return { init, packets, commit };
}

/** Map a USB HID usage id to the LIGHTSYNC key index used in frames. */
function transformIdx(family: 'g815' | 'g915', idx: number): number {
	if (idx >= 0xe0 && idx <= 0xe7) return idx - 0x78;
	return idx - 0x03;
}
