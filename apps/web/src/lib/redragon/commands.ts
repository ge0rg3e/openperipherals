import type { EffectParams } from '../controller';
import {
	appBrightnessToEvision,
	appSpeedToEvision,
	EVISION_DIRECTION,
	EVISION_MODE,
	EVISION_SPEED
} from './constants';
import { beginReport, colorDataReport, endReport, modeExReport, type EvReport } from './report';

const DEFAULT_COLOR: [number, number, number] = [0xff, 0xff, 0xff];

function hexToRgb(hex: string): [number, number, number] {
	const m = /^#?([0-9a-f]{6})$/i.exec(hex.trim());
	if (!m) return DEFAULT_COLOR;
	const v = parseInt(m[1], 16);
	return [(v >> 16) & 0xff, (v >> 8) & 0xff, v & 0xff];
}

export interface EffectReports {
	reports: EvReport[];
}

/** openkeyboard effect kind -> EVision hardware mode byte + mode flags. */
export function evisionModeFor(kind: string): { mode: number; random: boolean } {
	switch (kind) {
		case 'off':
			// The EVision firmware has no dedicated off mode; OpenRGB's lowest
			// brightness is 0 (off), so static at brightness 0 turns the board off.
			return { mode: EVISION_MODE.STATIC, random: false };
		case 'spectrum':
			return { mode: EVISION_MODE.SPECTRUM_CYCLE, random: false };
		case 'wheel':
		case 'wave':
			return { mode: kind === 'wheel' ? EVISION_MODE.COLOR_WHEEL : EVISION_MODE.COLOR_WAVE_LONG, random: true };
		case 'breathing':
			return { mode: EVISION_MODE.BREATHING, random: false };
		case 'reactive':
			return { mode: EVISION_MODE.REACTIVE, random: false };
		case 'starlight':
			return { mode: EVISION_MODE.STARLIGHT_FAST, random: false };
		case 'custom':
			// Per-key frames are unreliable on EVision firmware (see OpenRGB
			// @direct :x:); fall back to a solid colour so a "custom" selection
			// always produces visible output.
			return { mode: EVISION_MODE.STATIC, random: false };
		case 'static':
		default:
			return { mode: EVISION_MODE.STATIC, random: false };
	}
}

/** Build the write sequence for a hardware effect. */
export function buildEffectReports(params: EffectParams, brightness: number): EffectReports {
	const off = params.kind === 'off';
	const { mode, random } = evisionModeFor(params.kind);
	const color = params.color ? hexToRgb(params.color) : DEFAULT_COLOR;
	const speed = params.kind === 'starlight' || params.kind === 'reactive' || params.kind === 'wave' || params.kind === 'wheel' || params.kind === 'breathing' ? appSpeedToEvision(params.speed ?? 3) : EVISION_SPEED.NORMAL;
	const direction = params.direction === 'left' ? EVISION_DIRECTION.LEFT : EVISION_DIRECTION.RIGHT;

	return {
		reports: [modeExReport({ mode, brightness: off ? 0 : appBrightnessToEvision(brightness), speed, direction, random, color })]
	};
}

/**
 * Build the per-key (custom) colour frame sequence. The EVision matrix holds
 * 126 LEDs addressed sequentially (6 rows x ~21 cols of RGB triples); each
 * colour-data packet carries up to 0x36 bytes (18 keys) and every packet is
 * wrapped in begin/end markers like OpenRGB's SetKeyboardColors.
 */
export function buildCustomColorReports(colors: Uint8Array<ArrayBuffer>): EffectReports {
	const reports: EvReport[] = [beginReport()];
	const maxBytes = 0x36;
	for (let offset = 0; offset < colors.length; offset += maxBytes) {
		const slice = colors.slice(offset, Math.min(offset + maxBytes, colors.length));
		reports.push(colorDataReport([...slice], offset));
	}
	reports.push(endReport());
	return { reports };
}