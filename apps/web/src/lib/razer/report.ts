export const REPORT_LEN = 90;
export const ARG_BASE = 8;
export const CRC_INDEX = 88;
export const MAX_ARGS = 80;

export interface CommandReport {
	status: number;
	transactionId: number;
	remainingPackets: number;
	protocolType: number;
	commandClass: number;
	commandId: number;
	dataSize: number;
	args: Uint8Array;
}

export function newReport(commandClass: number, commandId: number, dataSize: number): CommandReport {
	return {
		status: 0x00,
		transactionId: 0x00,
		remainingPackets: 0x00,
		protocolType: 0x00,
		commandClass,
		commandId,
		dataSize,
		args: new Uint8Array(MAX_ARGS)
	};
}

function be16(value: number): [number, number] {
	return [(value >> 8) & 0xff, value & 0xff];
}

export function packReport(report: CommandReport): Uint8Array {
	const data = new Uint8Array(REPORT_LEN);
	data[0] = report.status;
	data[1] = report.transactionId;
	const [hi, lo] = be16(report.remainingPackets);
	data[2] = hi;
	data[3] = lo;
	data[4] = report.protocolType;
	data[5] = report.dataSize;
	data[6] = report.commandClass;
	data[7] = report.commandId;
	data[5] = report.dataSize;
	data.set(report.args.subarray(0, report.dataSize), 8);
	data[CRC_INDEX] = crc(data);
	data[89] = 0x00;
	return data;
}

export function crc(data: Uint8Array): number {
	let c = 0;
	for (let i = 2; i < 88; i++) c ^= data[i];
	return c;
}

export function unpackReport(data: Uint8Array | number[]): CommandReport {
	const u = data instanceof Uint8Array ? data : Uint8Array.from(data);
	const args = new Uint8Array(MAX_ARGS);
	const size = Math.min(u[5], MAX_ARGS);
	args.set(u.slice(8, 8 + size));
	return {
		status: u[0],
		transactionId: u[1],
		remainingPackets: (u[2] << 8) | u[3],
		protocolType: u[4],
		commandClass: u[6],
		commandId: u[7],
		dataSize: u[5],
		args
	};
}

export function validCrc(data: Uint8Array): boolean {
	return data[88] === crc(data);
}

export function formatHexSequence(data: Uint8Array | number[], separator = ' '): string {
	const u = data instanceof Uint8Array ? data : Uint8Array.from(data);
	const parts: string[] = [];
	for (let i = 0; i < u.length; i++) {
		if (i > 0 && i % 16 === 0) parts.push('\n');
		parts.push(u[i].toString(16).padStart(2, '0'));
	}
	return parts.join(separator);
}

export function formatHex(data: Uint8Array | number[]): string {
	return formatHexSequence(data);
}
