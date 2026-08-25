import { newReport, type CommandReport } from './report';
import { LED } from './constants';

export interface RGB {
	r: number;
	g: number;
	b: number;
}

export function clamp(v: number, lo: number, hi: number): number {
	return Math.min(hi, Math.max(lo, v));
}

function rgb(rgb?: RGB): [number, number, number] {
	return rgb ? [rgb.r & 0xff, rgb.g & 0xff, rgb.b & 0xff] : [0, 0, 0];
}

/* Standard device functions (class 0x00) */
export function setDeviceMode(mode: number, param: number): CommandReport {
	const r = newReport(0x00, 0x04, 0x02);
	if (mode !== 0x00 && mode !== 0x03) mode = 0x00;
	r.args[0] = mode;
	r.args[1] = 0x00;
	return r;
}

export function getSerial(): CommandReport {
	return newReport(0x00, 0x82, 0x16);
}

export function getFirmware(): CommandReport {
	return newReport(0x00, 0x81, 0x02);
}

export function getDeviceMode(): CommandReport {
	return newReport(0x00, 0x84, 0x02);
}

export function setLedState(vst: number, ledId: number, ledState: number): CommandReport {
	const r = newReport(0x03, 0x00, 0x03);
	r.args[0] = vst;
	r.args[1] = ledId;
	r.args[2] = clamp(ledState, 0x00, 0x01);
	return r;
}

export function getLedState(vst: number, ledId: number): CommandReport {
	const r = newReport(0x03, 0x80, 0x03);
	r.args[0] = vst;
	r.args[1] = ledId;
	return r;
}

export function setLedRgb(vst: number, ledId: number, rgb1: RGB): CommandReport {
	const r = newReport(0x03, 0x01, 0x05);
	r.args[0] = vst;
	r.args[1] = ledId;
	const [rr, gg, bb] = rgb(rgb1);
	r.args[2] = rr;
	r.args[3] = gg;
	r.args[4] = bb;
	return r;
}

export function setLedBrightness(vst: number, ledId: number, brightness: number): CommandReport {
	const r = newReport(0x03, 0x03, 0x03);
	r.args[0] = vst;
	r.args[1] = ledId;
	r.args[2] = clamp(brightness, 0x00, 0xff);
	return r;
}

export function getLedBrightness(vst: number, ledId: number): CommandReport {
	const r = newReport(0x03, 0x83, 0x03);
	r.args[0] = vst;
	r.args[1] = ledId;
	return r;
}

/* Battery (class 0x07, wireless-capable boards). Level is args[1], 0-255. */
export function getBatteryLevel(): CommandReport {
	return newReport(0x07, 0x80, 0x02);
}

/* Charging status is args[1]: 0 = not charging, 1 = charging. */
export function getChargingStatus(): CommandReport {
	return newReport(0x07, 0x84, 0x02);
}

function classicMatrixBase(argSize: number, effectId: number): CommandReport {
	const r = newReport(0x03, 0x0a, argSize);
	r.args[0] = effectId;
	return r;
}

export function classicEffectNone(): CommandReport {
	return classicMatrixBase(0x01, 0x00);
}

export function classicEffectWave(direction: number): CommandReport {
	const r = classicMatrixBase(0x02, 0x01);
	r.args[1] = clamp(direction, 0x01, 0x02);
	return r;
}

export function classicEffectSpectrum(): CommandReport {
	return classicMatrixBase(0x01, 0x04);
}

export function classicEffectReactive(speed: number, rgb1: RGB): CommandReport {
	const r = classicMatrixBase(0x05, 0x02);
	r.args[1] = clamp(speed, 0x01, 0x04);
	const [rr, gg, bb] = rgb(rgb1);
	r.args[2] = rr;
	r.args[3] = gg;
	r.args[4] = bb;
	return r;
}

export function classicEffectStatic(rgb1: RGB): CommandReport {
	const r = classicMatrixBase(0x04, 0x06);
	const [rr, gg, bb] = rgb(rgb1);
	r.args[1] = rr;
	r.args[2] = gg;
	r.args[3] = bb;
	return r;
}

export function classicEffectBreathingRandom(): CommandReport {
	const r = classicMatrixBase(0x08, 0x03);
	r.args[1] = 0x03;
	return r;
}

export function classicEffectBreathingSingle(rgb1: RGB): CommandReport {
	const r = classicMatrixBase(0x08, 0x03);
	r.args[1] = 0x01;
	const [rr, gg, bb] = rgb(rgb1);
	r.args[2] = rr;
	r.args[3] = gg;
	r.args[4] = bb;
	return r;
}

export function classicEffectBreathingDual(rgb1: RGB, rgb2: RGB): CommandReport {
	const r = classicMatrixBase(0x08, 0x03);
	r.args[1] = 0x02;
	const [r1, g1, b1] = rgb(rgb1);
	r.args[2] = r1;
	r.args[3] = g1;
	r.args[4] = b1;
	const [r2, g2, b2] = rgb(rgb2);
	r.args[5] = r2;
	r.args[6] = g2;
	r.args[7] = b2;
	return r;
}

export function classicEffectStarlightSingle(speed: number, rgb1: RGB): CommandReport {
	const r = classicMatrixBase(0x09, 0x19);
	r.args[1] = 0x01;
	r.args[2] = clamp(speed, 0x01, 0x03);
	const [rr, gg, bb] = rgb(rgb1);
	r.args[3] = rr;
	r.args[4] = gg;
	r.args[5] = bb;
	return r;
}

export function classicEffectStarlightDual(speed: number, rgb1: RGB, rgb2: RGB): CommandReport {
	const r = classicMatrixBase(0x0c, 0x19);
	r.args[1] = 0x02;
	r.args[2] = clamp(speed, 0x01, 0x03);
	const [r1, g1, b1] = rgb(rgb1);
	r.args[3] = r1;
	r.args[4] = g1;
	r.args[5] = b1;
	const [r2, g2, b2] = rgb(rgb2);
	r.args[6] = r2;
	r.args[7] = g2;
	r.args[8] = b2;
	return r;
}

export function classicEffectStarlightRandom(speed: number): CommandReport {
	const r = classicMatrixBase(0x02, 0x19);
	r.args[1] = 0x03;
	r.args[2] = clamp(speed, 0x01, 0x03);
	return r;
}

export function classicMatrixSetCustomFrame(row: number, startCol: number, stopCol: number, rgbBytes: number[]): CommandReport {
	// 0x46 data size = 4 header args + 66 RGB bytes (22 columns) - matches the
	// kernel driver's razer_chroma_standard_matrix_set_custom_frame packet.
	const r = newReport(0x03, 0x0b, 0x46);
	r.args[0] = 0xff; // frame id
	r.args[1] = row;
	r.args[2] = startCol;
	r.args[3] = stopCol;
	r.args.set(rgbBytes, 4);
	return r;
}

export function classicEffectCustomFrame(vst: number): CommandReport {
	const r = classicMatrixBase(0x02, 0x05);
	r.args[1] = vst;
	return r;
}

function extendedMatrixBase(argSize: number, vst: number, ledId: number, effectId: number): CommandReport {
	const r = newReport(0x0f, 0x02, argSize);
	r.args[0] = vst;
	r.args[1] = ledId;
	r.args[2] = effectId;
	return r;
}

export function extendedEffectNone(vst: number, ledId: number): CommandReport {
	return extendedMatrixBase(0x06, vst, ledId, 0x00);
}

export function extendedEffectStatic(vst: number, ledId: number, color: RGB): CommandReport {
	const r = extendedMatrixBase(0x09, vst, ledId, 0x01);
	r.args[5] = 0x01;
	const [rr, gg, bb] = rgb(color);
	r.args[6] = rr;
	r.args[7] = gg;
	r.args[8] = bb;
	return r;
}

export function extendedEffectWave(vst: number, ledId: number, direction: number): CommandReport {
	const r = extendedMatrixBase(0x06, vst, ledId, 0x04);
	r.args[3] = clamp(direction, 0x00, 0x02);
	r.args[4] = 0x28;
	return r;
}

export function extendedEffectStarlightRandom(vst: number, ledId: number, speed: number): CommandReport {
	const r = extendedMatrixBase(0x06, vst, ledId, 0x07);
	r.args[4] = clamp(speed, 0x01, 0x03);
	return r;
}

export function extendedEffectStarlightSingle(vst: number, ledId: number, speed: number, rgb1: RGB): CommandReport {
	const r = extendedMatrixBase(0x09, vst, ledId, 0x07);
	r.args[4] = clamp(speed, 0x01, 0x03);
	r.args[5] = 0x01;
	const [rr, gg, bb] = rgb(rgb1);
	r.args[6] = rr;
	r.args[7] = gg;
	r.args[8] = bb;
	return r;
}

export function extendedEffectStarlightDual(vst: number, ledId: number, speed: number, rgb1: RGB, rgb2: RGB): CommandReport {
	const r = extendedMatrixBase(0x0c, vst, ledId, 0x07);
	r.args[4] = clamp(speed, 0x01, 0x03);
	r.args[5] = 0x02;
	const [r1, g1, b1] = rgb(rgb1);
	r.args[6] = r1;
	r.args[7] = g1;
	r.args[8] = b1;
	const [r2, g2, b2] = rgb(rgb2);
	r.args[9] = r2;
	r.args[10] = g2;
	r.args[11] = b2;
	return r;
}

export function extendedEffectSpectrum(vst: number, ledId: number): CommandReport {
	return extendedMatrixBase(0x06, vst, ledId, 0x03);
}

export function extendedEffectWheel(vst: number, ledId: number, direction: number): CommandReport {
	const r = extendedMatrixBase(0x06, vst, ledId, 0x0a);
	r.args[3] = clamp(direction, 0x01, 0x02);
	r.args[4] = 0x28;
	return r;
}

export function extendedEffectReactive(vst: number, ledId: number, speed: number, color: RGB): CommandReport {
	const r = extendedMatrixBase(0x09, vst, ledId, 0x05);
	r.args[4] = clamp(speed, 0x01, 0x04);
	r.args[5] = 0x01;
	const [rr, gg, bb] = rgb(color);
	r.args[6] = rr;
	r.args[7] = gg;
	r.args[8] = bb;
	return r;
}

export function extendedEffectBreathingRandom(vst: number, ledId: number): CommandReport {
	return extendedMatrixBase(0x06, vst, ledId, 0x02);
}

export function extendedEffectBreathingSingle(vst: number, ledId: number, rgb1: RGB): CommandReport {
	const r = extendedMatrixBase(0x09, vst, ledId, 0x02);
	r.args[3] = 0x01;
	r.args[5] = 0x01;
	const [rr, gg, bb] = rgb(rgb1);
	r.args[6] = rr;
	r.args[7] = gg;
	r.args[8] = bb;
	return r;
}

export function extendedEffectBreathingDual(vst: number, ledId: number, rgb1: RGB, rgb2: RGB): CommandReport {
	const r = extendedMatrixBase(0x0c, vst, ledId, 0x02);
	r.args[3] = 0x02;
	r.args[5] = 0x02;
	const [r1, g1, b1] = rgb(rgb1);
	r.args[6] = r1;
	r.args[7] = g1;
	r.args[8] = b1;
	const [r2, g2, b2] = rgb(rgb2);
	r.args[9] = r2;
	r.args[10] = g2;
	r.args[11] = b2;
	return r;
}

export function extendedEffectCustomFrame(): CommandReport {
	return extendedMatrixBase(0x0c, 0x00, 0x00, 0x08);
}

export function extendedMatrixSetCustomFrame(row: number, startCol: number, stopCol: number, rgbBytes: number[], packetLength = 0x47): CommandReport {
	const rowLength = (stopCol + 1 - startCol) * 3;
	const len = Math.max(packetLength, rowLength + 5);
	const r = newReport(0x0f, 0x03, len);
	r.args[2] = row;
	r.args[3] = startCol;
	r.args[4] = stopCol;
	r.args.set(rgbBytes, 5);
	return r;
}

export function extendedBrightness(vst: number, ledId: number, brightness: number): CommandReport {
	const r = newReport(0x0f, 0x04, 0x03);
	r.args[0] = vst;
	r.args[1] = ledId;
	r.args[2] = clamp(brightness, 0x00, 0xff);
	return r;
}

export function getExtendedBrightness(vst: number, ledId: number): CommandReport {
	const r = newReport(0x0f, 0x84, 0x03);
	r.args[0] = vst;
	r.args[1] = ledId;
	return r;
}

export const LED_ID_BACKLIGHT = LED.BACKLIGHT;
export const LED_ID_MACRO = LED.MACRO;
export const LED_ID_GAME = LED.GAME;
