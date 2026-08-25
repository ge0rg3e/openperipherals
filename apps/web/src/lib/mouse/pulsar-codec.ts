/**
 * Pulsar report-8 configuration protocol.
 *
 * 16-byte output/input reports on HID report id 8. Every packet is
 * `[command][status][addrHi][addrLo][count][params…10][pad][checksum]` and the
 * checksum makes the sum of all bytes including the report id ≡ 0 mod 255
 * offset by 0x55. Settings live in a flat flash address space written through
 * command 0x07 / read back through 0x08; every write is verified by read-back.
 */

export const PULSAR_VENDOR_ID = 0x3710;
/** The Pulsar 4K receiver shares the Teevolution/VGN vendor id. */
export const PULSAR_VGN_RECEIVER_VENDOR_ID = 0x3554;

export const PULSAR_CONFIG_REPORT_ID = 0x08;
export const PULSAR_CONFIG_PACKET_LENGTH = 16;

export const PULSAR_COMMAND = {
	encryptionData: 0x01,
	deviceOnline: 0x03,
	batteryLevel: 0x04,
	writeFlashData: 0x07,
	readFlashData: 0x08,
	getCurrentConfig: 0x0e,
	readVersionId: 0x12,
	getDongleVersion: 0x1d,
	getRssi: 0x2b
} as const;

export const PULSAR_FLASH = {
	reportRate: 0,
	currentDpi: 4,
	liftOffDistance: 10,
	dpiValues: 12,
	debounceTime: 169,
	motionSync: 171,
	sleepTime: 173,
	angleSnapping: 175,
	rippleControl: 177,
	performanceState: 181,
	performanceTime: 183
} as const;

export const PULSAR_POLLING_RATES: readonly number[] = [125, 250, 500, 1000, 2000, 4000, 8000];

/** Sum of every byte plus the report id, folded so the total lands on 0x55. */
export function pulsarPacketChecksum(packet: Uint8Array): number {
	let sum = PULSAR_CONFIG_REPORT_ID;
	for (let index = 0; index < packet.length - 1; index += 1) sum += packet[index] ?? 0;
	return (0x55 - (sum & 0xff)) & 0xff;
}

/** The paired form used inside flash records: sum of the data bytes → 0x55. */
export function pulsarDataChecksum(data: Uint8Array): number {
	let sum = 0;
	for (const value of data) sum += value;
	return (0x55 - (sum & 0xff)) & 0xff;
}

/** Rates up to 1000 Hz encode as a divisor of 1000; above as rate/125. */
export function pulsarDecodePollingRate(encoded: number): number {
	return encoded >= 16 ? (encoded / 16) * 2000 : 1000 / encoded;
}

export function pulsarEncodePollingRate(rate: number): number {
	const encoded = rate <= 1000 ? 1000 / rate : (rate / 2000) * 16;
	if (!Number.isInteger(encoded) || !PULSAR_POLLING_RATES.includes(rate)) {
		throw new Error('Unsupported Pulsar polling rate.');
	}
	return encoded;
}

/**
 * Real Pulsar-vendor mice use a 10-step low range plus dpiEx-flagged high
 * ranges. A record is `[raw, raw, flags, checksum]` with the checksum making
 * the four bytes sum to 0x55.
 */
export function pulsarDecodeDpi(data: Uint8Array): number | null {
	if (data.length < 4) return null;
	const low = data[0] ?? 0;
	const duplicate = data[1] ?? 0;
	const flags = data[2] ?? 0;
	const checksum = data[3] ?? 0;
	if (low !== duplicate || ((low + duplicate + flags + checksum) & 0xff) !== 0x55) return null;
	const raw = low + (((flags & 0x0c) >> 2) << 8);
	let dpi = (raw + 1) * 10;
	if ((flags & 0x02) !== 0) dpi = dpi * 5 + 10000;
	if ((flags & 0x01) !== 0) dpi *= 2;
	return dpi;
}

export function pulsarEncodeDpi(dpi: number): Uint8Array {
	let raw: number;
	let dpiEx: number;
	if (dpi >= 30100) {
		raw = (dpi / 2 - 10050) / 50;
		dpiEx = 0x33;
	} else if (dpi >= 10050) {
		raw = (dpi - 10050) / 50;
		dpiEx = 0x22;
	} else {
		raw = dpi / 10 - 1;
		dpiEx = 0;
	}
	const high = raw >> 8;
	const result = new Uint8Array(4);
	result[0] = raw;
	result[1] = raw;
	result[2] = (high << 2) | (high << 6) | dpiEx | (dpiEx << 4);
	result[3] = pulsarDataChecksum(result.slice(0, 3));
	return result;
}

export function pulsarDpiOptions(): number[] {
	const options: number[] = [];
	for (let dpi = 10; dpi <= 10000; dpi += 10) options.push(dpi);
	for (let dpi = 10050; dpi <= 30000; dpi += 50) options.push(dpi);
	for (let dpi = 30100; dpi <= 32000; dpi += 100) options.push(dpi);
	return options;
}

// The Pulsar 4K Wireless Receiver enumerates under the shared VGN vendor id and
// uses a different, flat 50-step DPI encoding than real Pulsar mice. Branch on
// vendor id, never on CID/MID.

export function pulsarVgnDecodeDpi(data: Uint8Array): number | null {
	if (data.length < 4) return null;
	const low = data[0] ?? 0;
	const duplicate = data[1] ?? 0;
	const flags = data[2] ?? 0;
	const checksum = data[3] ?? 0;
	if (low !== duplicate || ((low + duplicate + flags + checksum) & 0xff) !== 0x55) return null;
	return ((((flags >> 2) & 0x03) << 8) + low + 1) * 50;
}

export function pulsarVgnEncodeDpi(dpi: number): Uint8Array {
	if (!Number.isInteger(dpi) || dpi < 50 || dpi > 26000 || dpi % 50 !== 0) {
		throw new Error('Pulsar DPI must be 50–26,000 in 50 DPI steps.');
	}
	const encoded = dpi / 50 - 1;
	const low = encoded & 0xff;
	const high = (encoded >> 8) & 0x03;
	const flags = (high << 2) | (high << 6);
	return new Uint8Array([low, low, flags, (0x55 - low - low - flags) & 0xff]);
}

export function pulsarVgnDpiOptions(): number[] {
	const options: number[] = [];
	for (let dpi = 50; dpi <= 26000; dpi += 50) options.push(dpi);
	return options;
}
