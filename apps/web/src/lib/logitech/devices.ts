import type { LayoutKind } from '../razer/devices';
import { LOGITECH_PIDS, type LogitechFamily } from './constants';

export type LogitechEffect = 'off' | 'static' | 'spectrum' | 'wave' | 'breathing' | 'reactive';

export interface LogitechDevice {
	pid: number;
	name: string;
	family: LogitechFamily;
	layout: LayoutKind;
	/** output report ids the firmware accepts (0x11=20-byte, 0x12=64-byte). */
	reportIds: number[];
	/** true when the board supports per-key (custom) frames. */
	custom: boolean;
	/** true when the firmware exposes the hardware wave effect. */
	wave: boolean;
	/** hardware effects the board can render natively. */
	effects: LogitechEffect[];
	matrixRows: number;
	matrixCols: number;
	/** URL (from /static) of a product photo shown on the landing page. */
	image?: string;
}

export const LOGITECH_DEVICES: LogitechDevice[] = [
	{
		pid: LOGITECH_PIDS.G213,
		name: 'Logitech G213 Prodigy',
		family: 'g213',
		layout: 'full',
		reportIds: [0x11],
		custom: false,
		wave: true,
		effects: ['off', 'static', 'spectrum', 'wave', 'breathing'],
		matrixRows: 6,
		matrixCols: 22,
		image: 'https://www.logitechg.com/content/dam/gaming/en/non-braid/g213-finch/g213-gallery-1-nb.png'
	},
	{
		pid: LOGITECH_PIDS.G512,
		name: 'Logitech G512 Carbon',
		family: 'romerg',
		layout: 'full',
		reportIds: [0x11, 0x12],
		custom: true,
		wave: false,
		effects: ['off', 'static', 'spectrum', 'breathing'],
		matrixRows: 6,
		matrixCols: 22,
		image: 'https://www.logitechg.com/content/dam/gaming/en/non-braid/flying-v-mx-g512/g513-backlit-mechanical-gaming-keyboard.png'
	},
	{
		pid: LOGITECH_PIDS.G512_RGB,
		name: 'Logitech G512 RGB',
		family: 'romerg',
		layout: 'full',
		reportIds: [0x11, 0x12],
		custom: true,
		wave: false,
		effects: ['off', 'static', 'spectrum', 'breathing'],
		matrixRows: 6,
		matrixCols: 22,
		image: 'https://www.logitechg.com/content/dam/gaming/en/non-braid/flying-v-mx-g512/g513-backlit-mechanical-gaming-keyboard.png'
	},
	{
		pid: LOGITECH_PIDS.G610_1,
		name: 'Logitech G610 Orion',
		family: 'romerg',
		layout: 'full',
		reportIds: [0x11, 0x12],
		custom: true,
		wave: false,
		effects: ['off', 'static', 'spectrum', 'breathing'],
		matrixRows: 6,
		matrixCols: 22,
		image: 'https://computerarenakh.com/image/cache/catalog/SEA%20DARA/111.NEW%202021/ASDFASDFA-468x278.png'
	},
	{
		pid: LOGITECH_PIDS.G610_2,
		name: 'Logitech G610 Orion',
		family: 'romerg',
		layout: 'full',
		reportIds: [0x11, 0x12],
		custom: true,
		wave: false,
		effects: ['off', 'static', 'spectrum', 'breathing'],
		matrixRows: 6,
		matrixCols: 22,
		image: 'https://computerarenakh.com/image/cache/catalog/SEA%20DARA/111.NEW%202021/ASDFASDFA-468x278.png'
	},
	{
		pid: LOGITECH_PIDS.G810_1,
		name: 'Logitech G810 Orion Spectrum',
		family: 'romerg',
		layout: 'full',
		reportIds: [0x11, 0x12],
		custom: true,
		wave: false,
		effects: ['off', 'static', 'spectrum', 'breathing'],
		matrixRows: 6,
		matrixCols: 22,
		image: 'https://www.logitechg.com/content/dam/products/gaming/keyboards/g810-orion-spectrum-rgb-gaming-keyboard/920-007757/g810-orion-spectrum-rgb-mechanical-keyboard13.png'
	},
	{
		pid: LOGITECH_PIDS.G810_2,
		name: 'Logitech G810 Orion Spectrum',
		family: 'romerg',
		layout: 'full',
		reportIds: [0x11, 0x12],
		custom: true,
		wave: false,
		effects: ['off', 'static', 'spectrum', 'breathing'],
		matrixRows: 6,
		matrixCols: 22,
		image: 'https://www.logitechg.com/content/dam/products/gaming/keyboards/g810-orion-spectrum-rgb-gaming-keyboard/920-007757/g810-orion-spectrum-rgb-mechanical-keyboard13.png'
	},
	{
		pid: LOGITECH_PIDS.G910_ORION_SPARK,
		name: 'Logitech G910 Orion Spark',
		family: 'g910spark',
		layout: 'full',
		reportIds: [0x11, 0x12],
		custom: true,
		wave: false,
		effects: ['off', 'static', 'spectrum', 'breathing'],
		matrixRows: 6,
		matrixCols: 22,
		image: 'https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcRyXhoto288bBxOLX1YLqUxgizZ9ABS8AgF6wEMfhf3jg&s=10'
	},
	{
		pid: LOGITECH_PIDS.G910,
		name: 'Logitech G910 Orion Spectrum',
		family: 'romerg',
		layout: 'full',
		reportIds: [0x11, 0x12],
		custom: true,
		wave: false,
		effects: ['off', 'static', 'spectrum', 'breathing'],
		matrixRows: 6,
		matrixCols: 22,
		image: 'https://m.media-amazon.com/images/I/81FXE5u1xNL._AC_UF894,1000_QL80_.jpg'
	},
	{
		pid: LOGITECH_PIDS.G_PRO,
		name: 'Logitech G Pro Keyboard',
		family: 'romerg',
		layout: 'full',
		reportIds: [0x11, 0x12],
		custom: true,
		wave: false,
		effects: ['off', 'static', 'spectrum', 'breathing'],
		matrixRows: 6,
		matrixCols: 22,
		image: 'https://www.relaxedtech.com/reviews/logitech/g-pro-mechanical-gaming-keyboard/logitech-g-pro-keyboard-front.png'
	},
	{
		pid: LOGITECH_PIDS.G813,
		name: 'Logitech G813 RGB',
		family: 'g815',
		layout: 'full',
		reportIds: [0x11],
		custom: true,
		wave: false,
		effects: ['off', 'static', 'spectrum', 'breathing'],
		matrixRows: 6,
		matrixCols: 22,
		image: `https://resource.logitechg.com/w_544,h_466,ar_7:6,c_pad,q_auto,f_auto,dpr_1.0/d_transparent.gif/content/dam/gaming/en/products/g815/g815-gallery-3.png`
	},
	{
		pid: LOGITECH_PIDS.G815,
		name: 'Logitech G815 RGB',
		family: 'g815',
		layout: 'full',
		reportIds: [0x11],
		custom: true,
		wave: false,
		effects: ['off', 'static', 'spectrum', 'breathing'],
		matrixRows: 6,
		matrixCols: 22,
		image: 'https://www.logitechg.com/content/dam/gaming/en/non-braid/g815-harpy-corded/ch-g815-wired-gallery-topdown-nb.png'
	},
	{
		pid: LOGITECH_PIDS.G915,
		name: 'Logitech G915 Lightspeed',
		family: 'g915',
		layout: 'full',
		reportIds: [0x11],
		custom: true,
		wave: true,
		effects: ['off', 'static', 'spectrum', 'wave', 'breathing', 'reactive'],
		matrixRows: 6,
		matrixCols: 22,
		image: 'https://www.logitechg.com/content/dam/gaming/en/products/g915/g915-gallery-2.png'
	},
	{
		pid: LOGITECH_PIDS.G915_TKL,
		name: 'Logitech G915 TKL Lightspeed',
		family: 'g915',
		layout: 'tkl',
		reportIds: [0x11],
		custom: true,
		wave: true,
		effects: ['off', 'static', 'spectrum', 'wave', 'breathing', 'reactive'],
		matrixRows: 6,
		matrixCols: 17,
		image: 'https://www.logitechg.com/content/dam/gaming/en/products/g915-tkl/g915-tkl-gallery/deu-g915-tkl-carbon-gallery-topdown.png'
	}
];

export function getLogitechKeyboard(pid: number): LogitechDevice | undefined {
	return LOGITECH_DEVICES.find((d) => d.pid === pid);
}

export const LOGITECH_SUPPORTED_PIDS: number[] = LOGITECH_DEVICES.map((d) => d.pid);

/** key code (as in layout.ts) → Logitech LED index (USB HID usage id). */
export const KEY_TO_IDX: Record<string, number> = {
	ESC: 0x29,
	F1: 0x3a,
	F2: 0x3b,
	F3: 0x3c,
	F4: 0x3d,
	F5: 0x3e,
	F6: 0x3f,
	F7: 0x40,
	F8: 0x41,
	F9: 0x42,
	F10: 0x43,
	F11: 0x44,
	F12: 0x45,
	PSCR: 0x46,
	SLCK: 0x47,
	PAUSE: 0x48,
	GRAVE: 0x35,
	'1': 0x1e,
	'2': 0x1f,
	'3': 0x20,
	'4': 0x21,
	'5': 0x22,
	'6': 0x23,
	'7': 0x24,
	'8': 0x25,
	'9': 0x26,
	'0': 0x27,
	MINUS: 0x2d,
	EQUAL: 0x2e,
	BACKSPACE: 0x2a,
	INS: 0x49,
	HOME: 0x4a,
	PGUP: 0x4b,
	NUMLOCK: 0x53,
	NPSLASH: 0x54,
	NPASTERISK: 0x55,
	NPMINUS: 0x56,
	TAB: 0x2b,
	Q: 0x14,
	W: 0x1a,
	E: 0x08,
	R: 0x15,
	T: 0x17,
	Y: 0x1c,
	U: 0x18,
	I: 0x0c,
	O: 0x12,
	P: 0x13,
	LBRACKET: 0x2f,
	RBRACKET: 0x30,
	BACKSLASH: 0x31,
	DEL: 0x4c,
	END: 0x4d,
	PGDN: 0x4e,
	NP7: 0x5f,
	NP8: 0x60,
	NP9: 0x61,
	NPPLUS: 0x57,
	CAPS: 0x39,
	A: 0x04,
	S: 0x16,
	D: 0x07,
	F: 0x09,
	G: 0x0a,
	H: 0x0b,
	J: 0x0d,
	K: 0x0e,
	L: 0x0f,
	SEMICOLON: 0x33,
	APOSTROPHE: 0x34,
	ENTER: 0x28,
	NP4: 0x5c,
	NP5: 0x5d,
	NP6: 0x5e,
	LSHIFT: 0xe1,
	Z: 0x1d,
	X: 0x1b,
	C: 0x06,
	V: 0x19,
	B: 0x05,
	N: 0x11,
	M: 0x10,
	COMMA: 0x36,
	PERIOD: 0x37,
	SLASH: 0x38,
	RSHIFT: 0xe5,
	UP: 0x52,
	NP1: 0x59,
	NP2: 0x5a,
	NP3: 0x5b,
	NPENTER: 0x58,
	LCTRL: 0xe0,
	LSUPER: 0xe3,
	LALT: 0xe2,
	SPACE: 0x2c,
	RALT: 0xe6,
	MENU: 0xe7,
	RCTRL: 0xe4,
	LEFT: 0x50,
	DOWN: 0x51,
	RIGHT: 0x4f,
	NP0: 0x62,
	NPDOT: 0x63
};

export const NUMPAD_CODES = new Set(['NUMLOCK', 'NPSLASH', 'NPASTERISK', 'NPMINUS', 'NPPLUS', 'NPENTER', 'NP0', 'NP1', 'NP2', 'NP3', 'NP4', 'NP5', 'NP6', 'NP7', 'NP8', 'NP9', 'NPDOT']);
