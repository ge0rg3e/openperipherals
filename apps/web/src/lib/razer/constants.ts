export const USB_VENDOR_ID_RAZER = 0x1532;

export const NOSTORE = 0x00;
export const VARSTORE = 0x01;

export const LED = {
	ZERO: 0x00,
	SCROLL_WHEEL: 0x01,
	BATTERY: 0x03,
	LOGO: 0x04,
	BACKLIGHT: 0x05,
	MACRO: 0x07,
	GAME: 0x08,
	RED_PROFILE: 0x0c,
	GREEN_PROFILE: 0x0d,
	BLUE_PROFILE: 0x0e,
	RIGHT_SIDE: 0x10,
	LEFT_SIDE: 0x11,
	ARGB_CH_1: 0x1a,
	ARGB_CH_2: 0x1b,
	ARGB_CH_3: 0x1c,
	ARGB_CH_4: 0x1d,
	ARGB_CH_5: 0x1e,
	ARGB_CH_6: 0x1f,
	CHARGING: 0x20,
	FAST_CHARGING: 0x21,
	FULLY_CHARGED: 0x22
} as const;

export type LedId = (typeof LED)[keyof typeof LED];

export const CMD = {
	BUSY: 0x01,
	SUCCESSFUL: 0x02,
	FAILURE: 0x03,
	TIMEOUT: 0x04,
	NOT_SUPPORTED: 0x05
} as const;

export enum ClassicEffect {
	NONE = 0x00,
	WAVE = 0x01,
	REACTIVE = 0x02,
	BREATHING = 0x03,
	SPECTRUM = 0x04,
	CUSTOM = 0x05,
	STATIC = 0x06,
	STARLIGHT = 0x19
}

export interface DeviceDefinition {
	pid: number;
	name: string;
	classicTransactionId: number;
	extended: boolean;
	reportIndex: number;
	rows: number;
	cols: number;
	rowLabels?: string[];
}