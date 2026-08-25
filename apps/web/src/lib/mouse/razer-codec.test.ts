import { describe, expect, it } from 'vitest';
import {
	RAZER_PACKET_LENGTH,
	RAZER_STATUS,
	RAZER_TRANSACTION_ID,
	RazerProtocolError,
	decodeBatteryPercent,
	decodeCharging,
	decodeDpi,
	decodeExtendedPollingRate,
	decodeLegacyPollingRate,
	decodeRazerResponse,
	decodeSleepTimeout,
	encodeRazerRequest,
	razerChecksum,
	razerReadDpiCommand,
	razerSetDpiCommand,
	razerSetExtendedEffectCommand,
	razerSetExtendedPollingCommand,
	razerSetLegacyPollingCommand
} from './razer-codec';

/** Builds a well-formed 90-byte reply for a command. */
function reply(
	command: { commandClass: number; commandId: number },
	args: readonly number[],
	status: number = RAZER_STATUS.ok
): Uint8Array {
	const packet = new Uint8Array(RAZER_PACKET_LENGTH);
	packet[0] = status;
	packet[1] = RAZER_TRANSACTION_ID;
	packet[5] = args.length;
	packet[6] = command.commandClass;
	packet[7] = command.commandId;
	packet.set(args, 8);
	packet[88] = razerChecksum(packet);
	return packet;
}

describe('razer framing', () => {
	it('encodes the documented packet layout', () => {
		const command = razerSetDpiCommand(3200, 3200);
		const request = encodeRazerRequest(command);
		expect(request.length).toBe(RAZER_PACKET_LENGTH);
		expect(request[1]).toBe(RAZER_TRANSACTION_ID);
		expect(request[5]).toBe(command.dataSize);
		expect(request[6]).toBe(0x04);
		expect(request[7]).toBe(0x05);
		// storage byte + big-endian x/y pair + two padding zeros
		expect([...request.slice(8, 15)]).toEqual([0x01, 0x0c, 0x80, 0x0c, 0x80, 0x00, 0x00]);
		expect(request[88]).toBe(razerChecksum(request));
	});

	it('decodes a matching reply and rejects a corrupt one', () => {
		const command = razerReadDpiCommand();
		const ok = decodeRazerResponse(reply(command, [0x01, 0x0c, 0x80, 0x0c, 0x80, 0, 0]), command);
		expect(decodeDpi(ok)).toEqual({ x: 3200, y: 3200 });

		const corrupted = reply(command, [1, 2, 3, 4, 5, 6, 7]);
		corrupted[40] ^= 0xff;
		expect(() => decodeRazerResponse(corrupted, command)).toThrow(RazerProtocolError);
	});

	it('distinguishes unsupported and stale replies', () => {
		const command = razerReadDpiCommand();
		expect(() => decodeRazerResponse(reply(command, [], RAZER_STATUS.unsupported), command)).toThrow(RazerProtocolError);
		// A different command echoing back is stale, not an answer.
		try {
			decodeRazerResponse(reply({ commandClass: 0x07, commandId: 0x80 }, []), command);
			throw new Error('should have thrown');
		} catch (error) {
			expect(error).toBeInstanceOf(RazerProtocolError);
			expect((error as RazerProtocolError).stale).toBe(true);
		}
	});
});

describe('razer value codecs', () => {
	it('decodes battery, charging and sleep replies', () => {
		expect(decodeBatteryPercent(new Uint8Array([0, 0xd9]))).toBe(85);
		expect(decodeCharging(new Uint8Array([0, 1]))).toBe(true);
		expect(decodeSleepTimeout(new Uint8Array([0x02, 0x58]))).toBe(600);
	});

	it('encodes polling divisors per transport', () => {
		expect(razerSetLegacyPollingCommand(500).args?.[0]).toBe(2);
		expect(razerSetExtendedPollingCommand(4000).args?.[1]).toBe(2);
		const pollingRead = { commandClass: 0x00, commandId: 0x85, dataSize: 0x01 };
		const hyperRead = { commandClass: 0x00, commandId: 0xc0, dataSize: 0x02 };
		for (const rate of [125, 250, 500, 1000]) {
			const args = decodeRazerResponse(reply(pollingRead, [1000 / rate]), pollingRead);
			expect(decodeLegacyPollingRate(args)).toBe(rate);
		}
		for (const rate of [2000, 4000, 8000]) {
			const args = decodeRazerResponse(reply(hyperRead, [0, 8000 / rate]), hyperRead);
			expect(decodeExtendedPollingRate(args)).toBe(rate);
		}
	});

	it('builds extended-matrix effect payloads', () => {
		const off = razerSetExtendedEffectCommand('off');
		expect(off.commandClass).toBe(0x0f);
		expect(off.args).toEqual([0x01, 0x04, 0x00, 0, 0, 0]);

		const breathingDual = razerSetExtendedEffectCommand('breathing-dual', { color: '#ff0000', color2: '#00ff00' });
		expect(breathingDual.args?.slice(3, 9)).toEqual([0x02, 0x00, 0x02, 0xff, 0x00, 0x00, 0x00, 0xff, 0x00].slice(0, 6));
		expect([...breathingDual.args ?? []].length).toBe(12);

		const reactive = razerSetExtendedEffectCommand('reactive', { color: '#123456', speed: 2 });
		expect(reactive.args?.slice(3)).toEqual([0x00, 2, 0x01, 0x12, 0x34, 0x56]);
	});
});
