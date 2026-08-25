// Constants for the Redragon EVision keyboard protocol.
//
// The vast majority of Redragon RGB keyboards (K550/K552/K556, the K587 Magic
// Wand family, K589 Shrapnel, Surara K582, ...) are built around the EVision /
// Huafenda firmware, which is the same controller driven by OpenRGB's
// EVisionKeyboardController. Redragon keyboards use the JESS VID (0x0c45) and
// expose a vendor "lighting" HID collection on usage page 0xFF1C.

export const REDRAGON_VID = 0x0c45;
export const REDRAGON_USAGE_PAGE = 0xff1c;

/** Output report id used for all EVision lighting packets (64-byte report). */
export const EVISION_REPORT_ID = 0x04;
export const EVISION_REPORT_SIZE = 64;

export const EVISION_CMD = {
	BEGIN: 0x01,
	END: 0x02,
	SET_PARAMETER: 0x06,
	WRITE_COLOR_DATA: 0x11
} as const;

export const EVISION_PARAMETER = {
	MODE: 0x00,
	BRIGHTNESS: 0x01,
	SPEED: 0x02,
	DIRECTION: 0x03,
	RANDOM_COLOR_FLAG: 0x04,
	MODE_COLOR: 0x05
} as const;

export const EVISION_MODE = {
	COLOR_WAVE_SHORT: 0x01,
	COLOR_WAVE_LONG: 0x02,
	COLOR_WHEEL: 0x03,
	SPECTRUM_CYCLE: 0x04,
	BREATHING: 0x05,
	STATIC: 0x06,
	REACTIVE: 0x07,
	REACTIVE_RIPPLE: 0x08,
	REACTIVE_LINE: 0x09,
	STARLIGHT_FAST: 0x0a,
	BLOOMING: 0x0b,
	RAINBOW_WAVE_VERTICAL: 0x0c,
	HURRICANE: 0x0d,
	ACCUMULATE: 0x0e,
	STARLIGHT_SLOW: 0x0f,
	VISOR: 0x10,
	SURMOUNT: 0x11,
	RAINBOW_WAVE_CIRCLE: 0x12,
	CUSTOM: 0x14
} as const;

/** 0x00 = lowest (off) ... 0x04 = brightest. */
export const EVISION_BRIGHTNESS = {
	OFF: 0x00,
	LOWEST: 0x01,
	NORMAL: 0x03,
	HIGHEST: 0x04
} as const;

/** 0x00 = fastest ... 0x05 = slowest. */
export const EVISION_SPEED = {
	FASTEST: 0x00,
	FAST: 0x02,
	NORMAL: 0x03,
	SLOW: 0x04,
	SLOWEST: 0x05
} as const;

/** Direction byte used inside the mode parameter block. */
export const EVISION_DIRECTION = {
	LEFT: 0x00,
	RIGHT: 0x01,
	UP: 0x02,
	DOWN: 0x03
} as const;

/** openkeyboard speed level (1..4) -> EVision speed byte. */
export function appSpeedToEvision(level: number): number {
	switch (level) {
		case 1:
			return EVISION_SPEED.FASTEST;
		case 2:
			return EVISION_SPEED.FAST;
		case 4:
			return EVISION_SPEED.SLOWEST;
		default:
			return EVISION_SPEED.NORMAL;
	}
}

/** openkeyboard brightness (0..255) -> EVision brightness byte (0..4). */
export function appBrightnessToEvision(value: number): number {
	return Math.max(0, Math.min(4, Math.round((value / 255) * 4)));
}