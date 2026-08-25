export const LOGITECH_VID = 0x046d;

export const REPORT_20 = 0x11;
export const REPORT_64 = 0x12;

export type LogitechFamily = 'g213' | 'romerg' | 'g910spark' | 'g815' | 'g915';

export const LOGITECH_PIDS = {
	G213: 0xc336,
	G512: 0xc342,
	G512_RGB: 0xc33c,
	G610_1: 0xc333,
	G610_2: 0xc338,
	G810_1: 0xc331,
	G810_2: 0xc337,
	G813: 0xc232,
	G815: 0xc33f,
	G910_ORION_SPARK: 0xc32b,
	G910: 0xc335,
	G_PRO: 0xc339,
	G915: 0xc33e,
	G915_TKL: 0xc343
} as const;

export const LOGI_MODE = {
	OFF: 0x00,
	STATIC: 0x01,
	BREATHING: 0x02,
	CYCLE: 0x03,
	WAVE: 0x04,
	RIPPLE: 0x05
} as const;

export const G213_ZONE = {
	LEFT_AREA: 0x01,
	MIDDLE_AREA: 0x02,
	RIGHT_AREA: 0x03,
	ARROW_HOME: 0x04,
	NUMPAD: 0x05
} as const;

export const G213_WAVE = {
	LEFT: 0x06,
	RIGHT: 0x01
} as const;

export const ROMERG_ZONE_MODE = {
	KEYBOARD: 0x00,
	LOGO: 0x01
} as const;

export const ROMERG_ZONE_DIRECT = {
	KEYBOARD: 0x01,
	MEDIA: 0x02,
	LOGO: 0x10,
	INDICATORS: 0x40
} as const;

export const G815_FRAME_TYPE = {
	LITTLE: 0x1f,
	BIG: 0x6f
} as const;

export const G915_ZONE_MODE = {
	LOGO: 0x00,
	KEYBOARD: 0x01,
	MULTIMEDIA: 0x02,
	GKEYS: 0x03,
	MODIFIERS: 0x04
} as const;

export const G915_FRAME_TYPE = {
	LITTLE: 0x1f,
	BIG: 0x6f,
	MODE: 0x1e
} as const;

export const G915_MODE = {
	OFF: 0x00,
	STATIC: 0x01,
	BREATHING: 0x02,
	CYCLE: 0x03,
	WAVE: 0x04,
	RIPPLE: 0x05,
	DIRECT: 0xff
} as const;

export const G915_LOGO_MODE = {
	OFF: 0x00,
	STATIC: 0x01,
	CYCLE: 0x02,
	BREATHING: 0x03
} as const;

export const G915_WIRED = {
	DEVICE_INDEX: 0xff,
	FEATURE_4522: 0x0e,
	FEATURE_8040: 0x13,
	FEATURE_8071: 0x09,
	FEATURE_8081: 0x0a
} as const;

export interface FamilyFrameBytes {
	modeFeature: number;
	modeCmd: number;
	frameFeature: number;
	frameCmd: number;
	commitFeature: number;
	commitCmd: number;
}

export const ROMERG_FRAME_BYTES: FamilyFrameBytes = {
	modeFeature: 0x0d,
	modeCmd: 0x3d,
	frameFeature: 0x0c,
	frameCmd: 0x3d,
	commitFeature: 0x0c,
	commitCmd: 0x5d
};

export const G910SPARK_FRAME_BYTES: FamilyFrameBytes = {
	modeFeature: 0x10,
	modeCmd: 0x3b,
	frameFeature: 0x0f,
	frameCmd: 0x3f,
	commitFeature: 0x0f,
	commitCmd: 0x5f
};

export const G815_FRAME_BYTES: FamilyFrameBytes = {
	modeFeature: 0x0d,
	modeCmd: 0x3d,
	frameFeature: 0x10,
	frameCmd: 0x00,
	commitFeature: 0x10,
	commitCmd: 0x5d
};

export function frameBytesFor(family: LogitechFamily): FamilyFrameBytes {
	switch (family) {
		case 'g910spark':
			return G910SPARK_FRAME_BYTES;
		case 'g815':
			return G815_FRAME_BYTES;
		default:
			return ROMERG_FRAME_BYTES;
	}
}

export const LOGITECH_MAX_FRAMES_PER_PACKET = 0x0e;
export const LOGITECH_DATA_SIZE = 16;
export const LOGITECH_MAX_KEY_PER_COLOR = 13;
