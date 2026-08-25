import { EVISION_CMD, EVISION_PARAMETER, EVISION_REPORT_ID, EVISION_REPORT_SIZE } from './constants';

export interface EvReport {
	reportId: number;
	data: Uint8Array<ArrayBuffer>;
}

/**
 * Build a final WebHID report from a raw 64-byte EVision buffer (OpenRGB's
 * `usb_buf` layout). Byte 0 holds the output-report id (0x04); the remaining 63
 * bytes are the payload handed to `sendReport`.
 */
function toReport(buf: number[]): EvReport {
	const full = new Uint8Array(EVISION_REPORT_SIZE);
	for (let i = 0; i < buf.length && i < EVISION_REPORT_SIZE; i++) full[i] = buf[i];
	const data = full.slice(1);
	return { reportId: full[0] === 0 ? EVISION_REPORT_ID : full[0], data };
}

/** OpenRGB's ComputeChecksum(): sum of bytes 0x03..0x3F -> usb_buf[1], usb_buf[2]. */
function checksumBytes(buf: number[]): [number, number] {
	let sum = 0;
	for (let i = 0x03; i < EVISION_REPORT_SIZE; i++) sum += buf[i] & 0xff;
	return [sum & 0xff, (sum >> 8) & 0xff];
}

function base(): number[] {
	const buf: number[] = new Array(EVISION_REPORT_SIZE).fill(0);
	buf[0] = EVISION_REPORT_ID;
	return buf;
}

export function beginReport(): EvReport {
	const buf = base();
	buf[1] = EVISION_CMD.BEGIN;
	buf[2] = 0x00;
	buf[3] = EVISION_CMD.BEGIN;
	return toReport(buf);
}

export function endReport(): EvReport {
	const buf = base();
	buf[1] = EVISION_CMD.END;
	buf[2] = 0x00;
	buf[3] = EVISION_CMD.END;
	return toReport(buf);
}

/** 0x06 "set parameter" packet used for every hardware effect. */
export function setParameterReport(parameter: number, data: number[]): EvReport {
	const buf = base();
	buf[3] = EVISION_CMD.SET_PARAMETER;
	buf[4] = data.length;
	buf[5] = parameter;
	for (let i = 0; i < data.length && i + 0x08 < EVISION_REPORT_SIZE; i++) buf[0x08 + i] = data[i];
	const [lo, hi] = checksumBytes(buf);
	buf[1] = lo;
	buf[2] = hi;
	return toReport(buf);
}

/**
 * Full 8-byte mode block: [mode, brightness, speed, direction, random,
 * red, green, blue] sent as one parameter (mode + flags). This mirrors OpenRGB's
 * SendKeyboardModeEx.
 */
export function modeExReport(opts: {
	mode: number;
	brightness: number;
	speed: number;
	direction: number;
	random: boolean;
	color: [number, number, number];
}): EvReport {
	return setParameterReport(EVISION_PARAMETER.MODE, [
		opts.mode,
		opts.brightness,
		opts.speed,
		opts.direction,
		opts.random ? 1 : 0,
		opts.color[0],
		opts.color[1],
		opts.color[2]
	]);
}

/** 0x11 "write color data" packet used for per-key (custom) frames. */
export function colorDataReport(data: number[], offset: number): EvReport {
	const buf = base();
	buf[3] = EVISION_CMD.WRITE_COLOR_DATA;
	buf[4] = data.length;
	buf[5] = offset & 0xff;
	buf[6] = offset >> 8;
	for (let i = 0; i < data.length && i + 0x08 < EVISION_REPORT_SIZE; i++) buf[0x08 + i] = data[i];
	const [lo, hi] = checksumBytes(buf);
	buf[1] = lo;
	buf[2] = hi;
	return toReport(buf);
}