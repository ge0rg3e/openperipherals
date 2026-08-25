import {
	G213_WAVE,
	G915_FRAME_TYPE,
	G915_LOGO_MODE,
	G915_MODE,
	G915_WIRED,
	G915_ZONE_MODE,
	LOGI_MODE,
	REPORT_20,
	REPORT_64,
	type FamilyFrameBytes
} from './constants';

export interface LogiReport {
	reportId: number;
	data: Uint8Array<ArrayBuffer>;
}

export interface LedFrame {
	idx: number;
	r: number;
	g: number;
	b: number;
}

/** Encode a speed value (in openkeyboard's 1..4 scale) as two speed bytes. */
export function speedBytes(value: number): [number, number] {
	const scaled = Math.round(value) * 100;
	return [(scaled >> 8) & 0xff, scaled & 0xff];
}

/** openkeyboard speed level → Logitech speed value used in the report byte(s). */
export function appSpeedToValue(level: number): number {
	const speeds: Record<number, number> = { 1: 10, 2: 50, 3: 128, 4: 200 };
	return speeds[level] ?? 50;
}

/** Build a 20-byte output report. Unset bytes are zeroed by the Uint8Array. */
export function report20(bytes: number[]): LogiReport {
	const buf = new Uint8Array(20);
	for (let i = 0; i < bytes.length && i < 20; i++) buf[i] = bytes[i];
	return { reportId: buf[0], data: buf.slice(1) };
}

/** Build a 64-byte output report (romerg custom frames). */
export function report64(bytes: number[]): LogiReport {
	const buf = new Uint8Array(64);
	for (let i = 0; i < bytes.length && i < 64; i++) buf[i] = bytes[i];
	return { reportId: buf[0], data: buf.slice(1) };
}

export function g213Mode(mode: number, speed: number, direction: 'left' | 'right', rgb: [number, number, number]): LogiReport {
	const bytes = new Array(20).fill(0);
	bytes[0] = REPORT_20;
	bytes[1] = 0xff;
	bytes[2] = 0x0c;
	bytes[3] = 0x3c;
	bytes[4] = 0x00;
	bytes[5] = mode;
	bytes[6] = rgb[0];
	bytes[7] = rgb[1];
	bytes[8] = rgb[2];
	const val = speedBytes(speed);
	if (mode === LOGI_MODE.BREATHING) {
		bytes[9] = val[0];
		bytes[10] = val[1];
		bytes[12] = 0x64;
	} else if (mode === LOGI_MODE.CYCLE) {
		bytes[11] = val[0];
		bytes[12] = val[1];
		bytes[13] = 0x64;
	} else if (mode === LOGI_MODE.WAVE) {
		bytes[11] = val[0];
		bytes[12] = val[1];
		bytes[13] = direction === 'left' ? G213_WAVE.LEFT : G213_WAVE.RIGHT;
		bytes[15] = val[0];
	}
	return report20(bytes);
}

export function g213SetZone(zone: number, rgb: [number, number, number]): LogiReport {
	const bytes = new Array(20).fill(0);
	bytes[0] = REPORT_20;
	bytes[1] = 0xff;
	bytes[2] = 0x0c;
	bytes[3] = 0x3a;
	bytes[4] = zone;
	bytes[5] = 0x01;
	bytes[6] = rgb[0];
	bytes[7] = rgb[1];
	bytes[8] = rgb[2];
	bytes[9] = 0x02;
	return report20(bytes);
}

export function romergMode(fb: FamilyFrameBytes, zone: number, mode: number, speed: number, rgb: [number, number, number]): LogiReport {
	const bytes = new Array(20).fill(0);
	bytes[0] = REPORT_20;
	bytes[1] = 0xff;
	bytes[2] = fb.modeFeature;
	bytes[3] = fb.modeCmd;
	bytes[4] = zone;
	bytes[5] = mode;
	bytes[6] = rgb[0];
	bytes[7] = rgb[1];
	bytes[8] = rgb[2];
	const val = speedBytes(speed);
	if (mode === LOGI_MODE.CYCLE) {
		bytes[11] = val[0];
		bytes[12] = val[1];
		bytes[13] = 0x64;
	} else if (mode === LOGI_MODE.BREATHING) {
		bytes[9] = val[0];
		bytes[10] = val[1];
		bytes[12] = 0x64;
	}
	return report20(bytes);
}

export function romergCommit(fb: FamilyFrameBytes): LogiReport {
	const bytes = new Array(20).fill(0);
	bytes[0] = REPORT_20;
	bytes[1] = 0xff;
	bytes[2] = fb.commitFeature;
	bytes[3] = fb.commitCmd;
	return report20(bytes);
}

export function romergDirectFrame(fb: FamilyFrameBytes, zone: number, leds: LedFrame[]): LogiReport {
	const bytes = new Array(64).fill(0);
	bytes[0] = REPORT_64;
	bytes[1] = 0xff;
	bytes[2] = fb.frameFeature;
	bytes[3] = fb.frameCmd;
	bytes[5] = zone;
	bytes[7] = leds.length;
	let offset = 8;
	for (const led of leds) {
		bytes[offset++] = led.idx;
		bytes[offset++] = led.r;
		bytes[offset++] = led.g;
		bytes[offset++] = led.b;
	}
	return report64(bytes);
}

export function g815InitDirect(): LogiReport[] {
	return [
		report20([REPORT_20, 0xff, 0x08, 0x3e]),
		report20([REPORT_20, 0xff, 0x08, 0x1e]),
		report20([REPORT_20, 0xff, 0x0f, 0x1e, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x01]),
		report20([REPORT_20, 0xff, 0x0f, 0x1e, 0x01, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x01])
	];
}

export function g815Mode(zone: number, mode: number, speed: number, rgb: [number, number, number]): LogiReport {
	const bytes = new Array(20).fill(0);
	bytes[0] = REPORT_20;
	bytes[1] = 0xff;
	bytes[2] = 0x0d;
	bytes[3] = 0x3d;
	bytes[4] = zone;
	bytes[5] = mode;
	bytes[6] = rgb[0];
	bytes[7] = rgb[1];
	bytes[8] = rgb[2];
	const val = speedBytes(speed);
	if (mode === LOGI_MODE.CYCLE) {
		bytes[11] = val[0];
		bytes[12] = val[1];
		bytes[13] = 0x64;
	} else if (mode === LOGI_MODE.BREATHING) {
		bytes[9] = val[0];
		bytes[10] = val[1];
		bytes[12] = 0x64;
	}
	return report20(bytes);
}

export function g815Frame(frameType: number, data: number[]): LogiReport {
	const bytes = new Array(20).fill(0);
	bytes[0] = REPORT_20;
	bytes[1] = 0xff;
	bytes[2] = 0x10;
	bytes[3] = frameType;
	for (let i = 0; i < Math.min(16, data.length); i++) bytes[4 + i] = data[i];
	return report20(bytes);
}

export function g815Commit(): LogiReport {
	return report20([REPORT_20, 0xff, 0x10, 0x5d]);
}

export function g915BeginModeSet(): LogiReport[] {
	return [
		report20([REPORT_20, G915_WIRED.DEVICE_INDEX, G915_WIRED.FEATURE_4522, 0x3e]),
		report20([REPORT_20, G915_WIRED.DEVICE_INDEX, G915_WIRED.FEATURE_4522, 0x1e])
	];
}

export function g915InitializeDirect(): LogiReport[] {
	return [
		report20([REPORT_20, G915_WIRED.DEVICE_INDEX, G915_WIRED.FEATURE_4522, 0x3e]),
		report20([REPORT_20, G915_WIRED.DEVICE_INDEX, G915_WIRED.FEATURE_4522, 0x1e]),
		report20([REPORT_20, G915_WIRED.DEVICE_INDEX, G915_WIRED.FEATURE_8071, 0x1e, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x01]),
		report20([REPORT_20, G915_WIRED.DEVICE_INDEX, G915_WIRED.FEATURE_8071, 0x1e, 0x01, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x01])
	];
}

export function g915Mode(zone: number, mode: number, logoMode: number, speed: number, brightness: number, rgb: [number, number, number]): LogiReport {
	const bytes = new Array(20).fill(0);
	bytes[0] = REPORT_20;
	bytes[1] = G915_WIRED.DEVICE_INDEX;
	bytes[2] = G915_WIRED.FEATURE_8071;
	bytes[3] = G915_FRAME_TYPE.MODE;
	bytes[4] = zone;
	bytes[5] = mode;
	bytes[6] = rgb[0];
	bytes[7] = rgb[1];
	bytes[8] = rgb[2];
	const speedVal = mode === G915_MODE.RIPPLE ? Math.round(speed) : Math.round(speed) * 100;
	if (mode === G915_MODE.STATIC) {
		bytes[9] = 0x02;
	} else if ((mode === G915_MODE.BREATHING && zone === G915_ZONE_MODE.KEYBOARD) || (logoMode === G915_LOGO_MODE.BREATHING && zone === G915_ZONE_MODE.LOGO)) {
		bytes[9] = (speedVal >> 8) & 0xff;
		bytes[10] = speedVal & 0xff;
		bytes[12] = brightness & 0xff;
	} else if ((mode === G915_MODE.CYCLE && zone === G915_ZONE_MODE.KEYBOARD) || (logoMode === G915_LOGO_MODE.CYCLE && zone === G915_ZONE_MODE.LOGO)) {
		bytes[11] = (speedVal >> 8) & 0xff;
		bytes[12] = speedVal & 0xff;
		bytes[13] = brightness & 0xff;
	} else if (mode === G915_MODE.WAVE) {
		bytes[12] = speedVal & 0xff;
		bytes[13] = 0x01;
		bytes[14] = brightness & 0xff;
		bytes[15] = (speedVal >> 8) & 0xff;
	} else if (mode === G915_MODE.RIPPLE) {
		bytes[11] = speedVal & 0xff;
	}
	bytes[16] = 0x01;
	return report20(bytes);
}

export function g915Frame(frameType: number, data: number[]): LogiReport {
	const bytes = new Array(20).fill(0);
	bytes[0] = REPORT_20;
	bytes[1] = G915_WIRED.DEVICE_INDEX;
	bytes[2] = G915_WIRED.FEATURE_8081;
	bytes[3] = frameType;
	for (let i = 0; i < Math.min(16, data.length); i++) bytes[4 + i] = data[i];
	return report20(bytes);
}

export function g915Commit(): LogiReport {
	return report20([REPORT_20, G915_WIRED.DEVICE_INDEX, G915_WIRED.FEATURE_8081, 0x7f]);
}
