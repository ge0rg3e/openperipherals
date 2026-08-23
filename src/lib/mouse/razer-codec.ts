/**
 * Razer mouse control protocol: 90-byte feature reports on report ID 0.
 *
 * Razer does not declare this report in its HID descriptor, but WebHID does not
 * validate report ids against it, so the exchange still succeeds. Packet layout:
 * `[status][transactionId][0][0][dataSize][commandClass][commandId][args…][…][xor]`
 * with the checksum XORing bytes 2..87 into byte 88. Reads set the high bit of
 * the command id; writes clear it.
 */

export const RAZER_REPORT_ID = 0;
export const RAZER_PACKET_LENGTH = 90;

/**
 * Razer's three transaction ids, named by value. A mismatch is silent — the
 * mouse simply never replies — so every product must state its own id.
 */
export const RAZER_TRANSACTION_ID_FF = 0xff;
export const RAZER_TRANSACTION_ID_3F = 0x3f;
export const RAZER_TRANSACTION_ID_1F = 0x1f;
/** Default for `encodeRazerRequest`; the per-product table overrides it. */
export const RAZER_TRANSACTION_ID = RAZER_TRANSACTION_ID_1F;

const ARGS_OFFSET = 8;
const CHECKSUM_INDEX = 88;
const BATTERY_SCALE = 255;

export const RAZER_STATUS = {
	busy: 0x01,
	ok: 0x02,
	failure: 0x03,
	timeout: 0x04,
	unsupported: 0x05
} as const;

/** Persistent store selector; the V2 generation reads/writes through 0x00. */
export const RAZER_STORAGE = 0x01;

const RAZER_LED_LOGO = 0x04;

export interface RazerCommand {
	commandClass: number;
	commandId: number;
	dataSize: number;
	args?: readonly number[];
}

export const RAZER_READ = {
	firmware: { commandClass: 0x00, commandId: 0x81, dataSize: 0x02 },
	serial: { commandClass: 0x00, commandId: 0x82, dataSize: 0x16 },
	battery: { commandClass: 0x07, commandId: 0x80, dataSize: 0x02 },
	charging: { commandClass: 0x07, commandId: 0x84, dataSize: 0x02 },
	sleepTimeout: { commandClass: 0x07, commandId: 0x83, dataSize: 0x02 },
	dpi: { commandClass: 0x04, commandId: 0x85, dataSize: 0x07, args: [RAZER_STORAGE] },
	pollingRate: { commandClass: 0x00, commandId: 0x85, dataSize: 0x01 },
	pollingRateExtended: { commandClass: 0x00, commandId: 0xc0, dataSize: 0x02, args: [0x00] },
	liftOff: { commandClass: 0x0b, commandId: 0x85, dataSize: 0x05 }
} as const satisfies Record<string, RazerCommand>;

export const RAZER_WRITE = {
	dpi: { commandClass: 0x04, commandId: 0x05, dataSize: 0x07 },
	pollingRate: { commandClass: 0x00, commandId: 0x05, dataSize: 0x01 },
	pollingRateExtended: { commandClass: 0x00, commandId: 0x40, dataSize: 0x02 },
	sensorSetting: { commandClass: 0x0b, commandId: 0x0b, dataSize: 0x04 },
	sleepTimeout: { commandClass: 0x07, commandId: 0x03, dataSize: 0x02 }
} as const satisfies Record<string, RazerCommand>;

export class RazerProtocolError extends Error {
	readonly status: number | null;
	/** The reply belonged to an earlier exchange; re-reading may still find it. */
	readonly stale: boolean;

	constructor(message: string, status: number | null = null, stale = false) {
		super(message);
		this.name = 'RazerProtocolError';
		this.status = status;
		this.stale = stale;
	}
}

/** Repeating a question costs nothing; repeating an instruction is forbidden. */
export function isRazerGetter(command: RazerCommand): boolean {
	return (command.commandId & 0x80) !== 0;
}

export function razerSetDpiCommand(x: number, y: number, storageByte: number = RAZER_STORAGE): RazerCommand {
	return {
		...RAZER_WRITE.dpi,
		args: [storageByte, (x >> 8) & 0xff, x & 0xff, (y >> 8) & 0xff, y & 0xff, 0x00, 0x00]
	};
}

export function razerReadDpiCommand(storageByte: number = RAZER_STORAGE): RazerCommand {
	return { ...RAZER_READ.dpi, args: [storageByte] };
}

export function razerSetSleepTimeoutCommand(seconds: number): RazerCommand {
	return { ...RAZER_WRITE.sleepTimeout, args: [(seconds >> 8) & 0xff, seconds & 0xff] };
}

function pollingDivisor(ceiling: number, pollingRateHz: number): number {
	const divisor = ceiling / pollingRateHz;
	if (!Number.isInteger(divisor) || divisor < 1 || divisor > 0xff) {
		throw new Error(`${pollingRateHz} Hz is not a rate this mouse can encode.`);
	}
	return divisor;
}

export function razerSetLegacyPollingCommand(pollingRateHz: number): RazerCommand {
	return { ...RAZER_WRITE.pollingRate, args: [pollingDivisor(1000, pollingRateHz)] };
}

export function razerSetExtendedPollingCommand(pollingRateHz: number): RazerCommand {
	return { ...RAZER_WRITE.pollingRateExtended, args: [0x00, pollingDivisor(8000, pollingRateHz)] };
}

// --- lighting (extended matrix effect family) ---

export type RazerExtendedEffect =
	| 'off'
	| 'static'
	| 'spectrum'
	| 'reactive'
	| 'breathing-random'
	| 'breathing-single'
	| 'breathing-dual';

/** Effect ids from openrazer's extended-matrix family. */
export const RAZER_EFFECT: Record<RazerExtendedEffect, number> = {
	off: 0x00,
	static: 0x01,
	spectrum: 0x03,
	reactive: 0x05,
	'breathing-random': 0x02,
	'breathing-single': 0x02,
	'breathing-dual': 0x02
};

/** Synapse's reactive speed scale: 1 is fast, 4 is slow. */
export type RazerReactiveSpeed = 1 | 2 | 3 | 4;

export function parseHexColor(hex: string): [number, number, number] {
	const match = /^#?([0-9a-f]{6})$/i.exec(hex.trim());
	if (!match) throw new Error(`${hex} is not a "#rrggbb" colour.`);
	const value = Number.parseInt(match[1], 16);
	return [(value >> 16) & 0xff, (value >> 8) & 0xff, value & 0xff];
}

export function razerSetExtendedEffectCommand(
	effect: RazerExtendedEffect,
	options: { color?: string; color2?: string; speed?: RazerReactiveSpeed } = {}
): RazerCommand {
	const args: number[] = [RAZER_STORAGE, RAZER_LED_LOGO, RAZER_EFFECT[effect]];
	switch (effect) {
		case 'off':
		case 'spectrum':
		case 'breathing-random':
			args.push(0x00, 0x00, 0x00);
			break;
		case 'static':
			args.push(0x00, 0x00, 0x01, ...parseHexColor(options.color ?? '#00ff88'));
			break;
		case 'reactive':
			args.push(0x00, options.speed ?? 3, 0x01, ...parseHexColor(options.color ?? '#00ff88'));
			break;
		case 'breathing-single':
			args.push(0x01, 0x00, 0x01, ...parseHexColor(options.color ?? '#00ff88'));
			break;
		case 'breathing-dual':
			args.push(
				0x02,
				0x00,
				0x02,
				...parseHexColor(options.color ?? '#00ff88'),
				...parseHexColor(options.color2 ?? '#0088ff')
			);
			break;
	}
	return { commandClass: 0x0f, commandId: 0x02, dataSize: args.length, args };
}

// --- framing ---

export function razerChecksum(packet: Uint8Array): number {
	let checksum = 0;
	for (let index = 2; index < CHECKSUM_INDEX; index += 1) checksum ^= packet[index];
	return checksum;
}

export function encodeRazerRequest(command: RazerCommand, transactionId: number = RAZER_TRANSACTION_ID): Uint8Array {
	const packet = new Uint8Array(RAZER_PACKET_LENGTH);
	packet[1] = transactionId;
	packet[5] = command.dataSize;
	packet[6] = command.commandClass;
	packet[7] = command.commandId;
	packet.set(command.args ?? [], ARGS_OFFSET);
	packet[CHECKSUM_INDEX] = razerChecksum(packet);
	return packet;
}

function describe(command: RazerCommand, problem: string): string {
	const hex = (value: number) => `0x${value.toString(16).padStart(2, '0')}`;
	return `Class ${hex(command.commandClass)} command ${hex(command.commandId)} ${problem}.`;
}

/** Returns the reply arguments, or throws with the reported status. */
export function decodeRazerResponse(packet: Uint8Array, command: RazerCommand): Uint8Array {
	if (packet.length !== RAZER_PACKET_LENGTH) {
		throw new RazerProtocolError(describe(command, `returned ${packet.length} bytes instead of ${RAZER_PACKET_LENGTH}`));
	}
	if (packet[CHECKSUM_INDEX] !== razerChecksum(packet)) {
		throw new RazerProtocolError(describe(command, 'returned a reply with a bad checksum'));
	}
	const status = packet[0];
	if (status === RAZER_STATUS.unsupported) {
		throw new RazerProtocolError(describe(command, 'is not supported by this mouse'), status);
	}
	if (status !== RAZER_STATUS.ok) {
		throw new RazerProtocolError(describe(command, `returned status 0x${status.toString(16).padStart(2, '0')}`), status);
	}
	if (packet[6] !== command.commandClass || packet[7] !== command.commandId) {
		throw new RazerProtocolError(describe(command, 'was answered by a different command'), status, true);
	}
	const length = Math.min(packet[5], RAZER_PACKET_LENGTH - ARGS_OFFSET);
	return packet.slice(ARGS_OFFSET, ARGS_OFFSET + length);
}

// --- decoders ---

export function decodeFirmwareVersion(args: Uint8Array): string {
	return `${args[0]}.${args[1]}`;
}

export function decodeSerial(args: Uint8Array): string {
	let text = '';
	for (const byte of args) {
		if (byte === 0) break;
		text += String.fromCharCode(byte);
	}
	return text.trim();
}

export function decodeBatteryPercent(args: Uint8Array): number {
	return Math.round(((args[1] ?? 0) * 100) / BATTERY_SCALE);
}

export function decodeCharging(args: Uint8Array): boolean {
	return args[1] === 1;
}

export function decodeSleepTimeout(args: Uint8Array): number {
	return ((args[0] ?? 0) << 8) | (args[1] ?? 0);
}

export interface RazerDpi {
	x: number;
	y: number;
}

export function decodeDpi(args: Uint8Array): RazerDpi {
	return { x: ((args[1] ?? 0) << 8) | (args[2] ?? 0), y: ((args[3] ?? 0) << 8) | (args[4] ?? 0) };
}

export type RazerTrackingDistance = 'Low' | 'Medium' | 'High';

/** Indexed by the byte the mouse reports, so the order is the encoding. */
export const RAZER_TRACKING_DISTANCES: readonly RazerTrackingDistance[] = ['Low', 'Medium', 'High'];

/** Second argument of every class 0x0b write — a smart-tracking selector. */
const RAZER_SENSOR_SELECTOR = 0x04;

/** Third argument of `0x0b`/`0x0b`, choosing which setting the fourth carries. */
const RAZER_SENSOR_SETTING_TRACKING_DISTANCE = 0x01;

export interface RazerLiftOff {
	tracking: RazerTrackingDistance | null;
}

export function decodeLiftOff(args: Uint8Array): RazerLiftOff {
	return { tracking: RAZER_TRACKING_DISTANCES[args[2] ?? 99] ?? null };
}

/** Writing a tracking level also takes the mouse out of asymmetric mode. */
export function razerSetTrackingDistanceCommand(distance: RazerTrackingDistance): RazerCommand {
	const level = RAZER_TRACKING_DISTANCES.indexOf(distance);
	if (level < 0) throw new Error(`${distance} is not a tracking distance this mouse offers.`);
	return {
		...RAZER_WRITE.sensorSetting,
		args: [0x00, RAZER_SENSOR_SELECTOR, RAZER_SENSOR_SETTING_TRACKING_DISTANCE, level]
	};
}

// --- polling-rate decoders ---

/** Legacy polling encodes the rate as a divisor of 1000. */
export function decodeLegacyPollingRate(args: Uint8Array): number {
	if (!args[0]) throw new RazerProtocolError('The mouse reported an unknown polling rate.');
	return Math.round(1000 / args[0]);
}

/** HyperPolling rates encode as a divisor of 8000; the rate lives in byte 1. */
export function decodeExtendedPollingRate(args: Uint8Array): number {
	if (!args[1]) throw new RazerProtocolError('The mouse reported an unknown polling rate.');
	return Math.round(8000 / args[1]);
}
