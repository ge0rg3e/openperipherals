import { describe, expect, it } from 'vitest';
import {
	pulsarDataChecksum,
	pulsarDecodeDpi,
	pulsarDecodePollingRate,
	pulsarEncodeDpi,
	pulsarEncodePollingRate,
	pulsarPacketChecksum,
	pulsarVgnDecodeDpi,
	pulsarVgnEncodeDpi
} from './pulsar-codec';

// All byte vectors below are hardware captures from the OpenMouse protocol
// project's own test suite — they pin the vendored codec to the reference.

describe('pulsar vgn-receiver DPI codec', () => {
	it('decodes captured flash stages as 50-step values', () => {
		expect(pulsarVgnDecodeDpi(new Uint8Array([0x0f, 0x0f, 0x00, 0x37]))).toBe(800);
		expect(pulsarVgnDecodeDpi(new Uint8Array([0x1f, 0x1f, 0x00, 0x17]))).toBe(1600);
		expect(pulsarVgnDecodeDpi(new Uint8Array([0x3f, 0x3f, 0x00, 0xd7]))).toBe(3200);
		expect(pulsarVgnDecodeDpi(new Uint8Array([0x13, 0x13, 0x00, 0x2f]))).toBe(1000);
		expect(pulsarVgnDecodeDpi(new Uint8Array([0x07, 0x07, 0x88, 0xbf]))).toBe(26000);
	});

	it('rejects corrupt or mismatched stages', () => {
		expect(pulsarVgnDecodeDpi(new Uint8Array([0x0f, 0x0f, 0x00, 0x00]))).toBeNull();
		expect(pulsarVgnDecodeDpi(new Uint8Array([0x0f, 0x1f, 0x00, 0x37]))).toBeNull();
		expect(pulsarVgnDecodeDpi(new Uint8Array([0x0f, 0x0f, 0x00]))).toBeNull();
	});

	it('round-trips supported UI values', () => {
		for (const dpi of [50, 400, 800, 1600, 26000]) {
			expect(pulsarVgnDecodeDpi(pulsarVgnEncodeDpi(dpi))).toBe(dpi);
		}
	});

	it('matches the captured write pattern', () => {
		expect([...pulsarVgnEncodeDpi(800)]).toEqual([0x0f, 0x0f, 0x00, 0x37]);
		expect([...pulsarVgnEncodeDpi(1600)]).toEqual([0x1f, 0x1f, 0x00, 0x17]);
		expect([...pulsarVgnEncodeDpi(3200)]).toEqual([0x3f, 0x3f, 0x00, 0xd7]);
	});
});

describe('pulsar-vendor DPI codec', () => {
	it('decodes captured CrazyLight stages as 10-step values', () => {
		expect(pulsarDecodeDpi(new Uint8Array([79, 79, 0, 183]))).toBe(800);
		expect(pulsarDecodeDpi(new Uint8Array([159, 159, 0, 23]))).toBe(1600);
		expect(pulsarDecodeDpi(new Uint8Array([39, 39, 0, 7]))).toBe(400);
	});

	it('rejects corrupt or mismatched stages', () => {
		expect(pulsarDecodeDpi(new Uint8Array([79, 79, 0, 0]))).toBeNull();
		expect(pulsarDecodeDpi(new Uint8Array([79, 159, 0, 183]))).toBeNull();
		expect(pulsarDecodeDpi(new Uint8Array([79, 79, 0]))).toBeNull();
	});

	it('round-trips the full catalogue', () => {
		for (const dpi of [10, 400, 800, 1600, 10000, 10050, 30000, 30100, 32000]) {
			expect(pulsarDecodeDpi(pulsarEncodeDpi(dpi))).toBe(dpi);
		}
	});

	it('matches the captured write pattern', () => {
		expect([...pulsarEncodeDpi(800)]).toEqual([79, 79, 0, 183]);
		expect([...pulsarEncodeDpi(1600)]).toEqual([159, 159, 0, 23]);
		expect([...pulsarEncodeDpi(400)]).toEqual([39, 39, 0, 7]);
	});
});

describe('pulsar checksums and polling rate', () => {
	it('matches the shared 0x55 scheme', () => {
		expect(pulsarDataChecksum(new Uint8Array([0x0f, 0x0f, 0x00]))).toBe(0x37);
		expect(pulsarPacketChecksum(new Uint8Array(16))).toBe((0x55 - 0x08) & 0xff);
	});

	it('round-trips every supported polling rate', () => {
		for (const rate of [125, 250, 500, 1000, 2000, 4000, 8000]) {
			expect(pulsarDecodePollingRate(pulsarEncodePollingRate(rate))).toBe(rate);
		}
	});
});
