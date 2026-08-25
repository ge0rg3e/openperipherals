/**
 * WebHID driver for the Pulsar report-8 family: Pulsar flash mice plus the
 * Pulsar 4K Wireless Receiver, which enumerates under the shared
 * Teevolution/VGN vendor id and speaks the same 16-byte protocol.
 *
 * Every write is wrapped in a device-online toggle (command 0x03) and verified
 * by reading the same flash range back.
 */

import type { LiftOffDistance, MouseDriver, MouseStatus } from './types';
import type { HidInputReportEvent, MouseHidDevice } from './webhid';
import {
	PULSAR_COMMAND as COMMAND,
	PULSAR_CONFIG_PACKET_LENGTH,
	PULSAR_CONFIG_REPORT_ID,
	PULSAR_FLASH as FLASH,
	PULSAR_POLLING_RATES,
	PULSAR_VGN_RECEIVER_VENDOR_ID,
	PULSAR_VENDOR_ID,
	pulsarDecodeDpi,
	pulsarDecodePollingRate,
	pulsarDpiOptions,
	pulsarEncodeDpi,
	pulsarEncodePollingRate,
	pulsarPacketChecksum,
	pulsarVgnDecodeDpi,
	pulsarVgnDpiOptions,
	pulsarVgnEncodeDpi
} from './pulsar-codec';

const PULSAR_SLEEP_OPTIONS: readonly number[] = [1, 3, 6, 12, 30, 60, 180];
const DEBOUNCE_MAX_MS = 15;
const RESPONSE_TIMEOUT_MS = 1200;

export class PulsarMouseClient implements MouseDriver {
	private info: {
		cid: number;
		mid: number;
		type: number;
		dongleType: number;
		connection: 'Wired' | 'Wireless';
		maximumPollingRateHz: number;
	} | null = null;

	private responseWaiter: {
		command: number;
		resolve: (bytes: Uint8Array) => void;
		reject: (reason: Error) => void;
	} | null = null;

	private readonly onInputReport = (event: HidInputReportEvent): void => {
		const bytes = new Uint8Array(event.data.buffer.slice(event.data.byteOffset, event.data.byteOffset + event.data.byteLength));
		if (event.reportId === PULSAR_CONFIG_REPORT_ID && bytes[0] === this.responseWaiter?.command) {
			const waiter = this.responseWaiter;
			this.responseWaiter = null;
			waiter.resolve(bytes);
		}
	};

	readonly device: MouseHidDevice;

	constructor(device: MouseHidDevice) {
		this.device = device;
	}

	static isSupported(device: MouseHidDevice): boolean {
		if (device.vendorId !== PULSAR_VENDOR_ID && device.vendorId !== PULSAR_VGN_RECEIVER_VENDOR_ID) return false;
		return device.collections.some(
			(collection) =>
				collection.inputReports.length === 1 &&
				collection.outputReports.length === 1 &&
				collection.inputReports[0]?.reportId === PULSAR_CONFIG_REPORT_ID &&
				collection.outputReports[0]?.reportId === PULSAR_CONFIG_REPORT_ID
		);
	}

	async open(): Promise<void> {
		if (!this.device.opened) await this.device.open();
		this.device.addEventListener('inputreport', this.onInputReport);
	}

	async close(): Promise<void> {
		this.device.removeEventListener('inputreport', this.onInputReport);
		this.responseWaiter?.reject(new Error('The Pulsar device was closed.'));
		this.responseWaiter = null;
		if (this.device.opened) await this.device.close();
	}

	getDpiOptions(): number[] {
		return this.isVgnReceiver() ? pulsarVgnDpiOptions() : pulsarDpiOptions();
	}

	getSleepOptions(): readonly number[] {
		return PULSAR_SLEEP_OPTIONS;
	}

	getDebounceMaxMs(): number {
		return DEBOUNCE_MAX_MS;
	}

	async readStatus(): Promise<MouseStatus> {
		const info = this.info ?? (await this.readDeviceInfo());
		return await this.withDeviceControl(async () => {
			const flash = await this.readFlash(FLASH.reportRate, FLASH.rippleControl + 2);
			const battery = await this.query(COMMAND.batteryLevel);
			const deviceVersion = await this.query(COMMAND.readVersionId);
			const dongleVersion = await this.query(COMMAND.getDongleVersion).catch(() => null);
			const rssi = await this.query(COMMAND.getRssi).catch(() => null);
			const currentStage = Math.min(flash[FLASH.currentDpi] ?? 0, 7);
			const dpi = this.decodeDpi(flash.slice(FLASH.dpiValues + currentStage * 4, FLASH.dpiValues + currentStage * 4 + 4)) ?? 800;
			const lodValue = flash[FLASH.liftOffDistance];
			return {
				brand: 'Pulsar',
				name: this.device.productName || 'Pulsar Mouse',
				ui: { family: 'pulsar', hideUnsupportedPollingRates: true },
				batteryPercent: battery[5] <= 100 ? battery[5] : null,
				batteryState: battery[6] === 1 ? 'Charging' : 'Discharging',
				dpi,
				pollingRateHz: pulsarDecodePollingRate(flash[FLASH.reportRate]),
				supportedPollingRates: PULSAR_POLLING_RATES.filter((rate) => rate <= info.maximumPollingRateHz),
				activeProfile: null,
				connectionType: info.connection,
				signalStrength: rssi && rssi[1] === 0 ? Math.min(rssi[5] ?? 0, 4) : null,
				debounceMs: flash[FLASH.debounceTime] ?? null,
				motionSync: flash[FLASH.motionSync] === 1,
				sleepTimeout: flash[FLASH.sleepTime] ?? null,
				angleSnapping: flash[FLASH.angleSnapping] === 1,
				rippleControl: flash[FLASH.rippleControl] === 1,
				liftOffDistance: lodValue === 3 ? 'Low' : lodValue === 1 ? 'Medium' : lodValue === 2 ? 'High' : null,
				firmware: [
					this.decodeVersionOptional('Mouse', deviceVersion) ?? 'Mouse firmware unavailable',
					this.decodeVersionOptional('Dongle', dongleVersion) ?? 'Dongle firmware unavailable'
				]
			};
		});
	}

	async setPollingRate(pollingRateHz: number): Promise<number> {
		const info = this.info ?? (await this.readDeviceInfo());
		if (pollingRateHz > info.maximumPollingRateHz) {
			throw new Error(`This connection supports at most ${info.maximumPollingRateHz} Hz.`);
		}
		const encoded = pulsarEncodePollingRate(pollingRateHz);
		return await this.withDeviceControl(async () => {
			await this.writeCheckedByte(FLASH.reportRate, encoded);
			const confirmed = pulsarDecodePollingRate((await this.readFlash(FLASH.reportRate, 2))[0]);
			if (confirmed !== pollingRateHz) throw new Error(`The mouse kept ${confirmed} Hz instead of ${pollingRateHz} Hz.`);
			return confirmed;
		});
	}

	async setDpi(dpi: number): Promise<number> {
		if (!this.getDpiOptions().includes(dpi)) throw new Error(`${dpi} DPI is not supported by this Pulsar sensor.`);
		return await this.withDeviceControl(async () => {
			const currentStage = (await this.readFlash(FLASH.currentDpi, 2))[0] ?? 0;
			const address = FLASH.dpiValues + Math.min(currentStage, 7) * 4;
			await this.writeFlash(address, this.encodeDpi(dpi));
			const confirmed = this.decodeDpi(await this.readFlash(address, 4));
			if (confirmed !== dpi) throw new Error(`The mouse kept ${confirmed ?? 'an unknown DPI'} instead of ${dpi} DPI.`);
			return confirmed;
		});
	}

	async setLiftOffDistance(liftOffDistance: LiftOffDistance): Promise<LiftOffDistance> {
		const encoded = ({ Low: 3, Medium: 1, High: 2 } as const)[liftOffDistance];
		return await this.withDeviceControl(async () => {
			await this.writeCheckedByte(FLASH.liftOffDistance, encoded);
			const confirmedValue = (await this.readFlash(FLASH.liftOffDistance, 2))[0];
			const confirmed: LiftOffDistance | null =
				confirmedValue === 3 ? 'Low' : confirmedValue === 1 ? 'Medium' : confirmedValue === 2 ? 'High' : null;
			if (confirmed !== liftOffDistance) throw new Error(`The mouse kept ${confirmed ?? 'an unknown LOD'} instead of ${liftOffDistance}.`);
			return confirmed;
		});
	}

	async setMotionSync(enabled: boolean): Promise<boolean> {
		return await this.setVerifiedBoolean(FLASH.motionSync, enabled, 'Motion Sync');
	}

	async setAngleSnapping(enabled: boolean): Promise<boolean> {
		return await this.setVerifiedBoolean(FLASH.angleSnapping, enabled, 'angle snapping');
	}

	async setRippleControl(enabled: boolean): Promise<boolean> {
		return await this.setVerifiedBoolean(FLASH.rippleControl, enabled, 'ripple control');
	}

	async setDebounceTime(debounceMs: number): Promise<number> {
		if (!Number.isInteger(debounceMs) || debounceMs < 0 || debounceMs > DEBOUNCE_MAX_MS) {
			throw new Error(`This Pulsar model supports a debounce time from 0 to ${DEBOUNCE_MAX_MS} ms.`);
		}
		return await this.setVerifiedByte(FLASH.debounceTime, debounceMs, 'debounce time');
	}

	async setSleepTimeout(timeout: number): Promise<number> {
		if (!PULSAR_SLEEP_OPTIONS.includes(timeout)) throw new Error('Unsupported Pulsar sleep timeout.');
		return await this.withDeviceControl(async () => {
			await this.writeCheckedByte(FLASH.sleepTime, timeout);
			await this.writeCheckedByte(FLASH.performanceTime, timeout);
			const sleepConfirmed = (await this.readFlash(FLASH.sleepTime, 2))[0];
			const performanceConfirmed = (await this.readFlash(FLASH.performanceTime, 2))[0];
			if (sleepConfirmed !== timeout || performanceConfirmed !== timeout) {
				throw new Error('The mouse did not confirm the requested sleep timeout.');
			}
			return sleepConfirmed;
		});
	}

	// --- identification ---

	private isVgnReceiver(): boolean {
		return this.device.vendorId === PULSAR_VGN_RECEIVER_VENDOR_ID;
	}

	private encodeDpi(dpi: number): Uint8Array {
		return this.isVgnReceiver() ? pulsarVgnEncodeDpi(dpi) : pulsarEncodeDpi(dpi);
	}

	private decodeDpi(data: Uint8Array): number | null {
		return this.isVgnReceiver() ? pulsarVgnDecodeDpi(data) : pulsarDecodeDpi(data);
	}

	private async readDeviceInfo(): Promise<{
		cid: number;
		mid: number;
		type: number;
		dongleType: number;
		connection: 'Wired' | 'Wireless';
		maximumPollingRateHz: number;
	}> {
		await this.open();
		const challenge = new Uint8Array(8);
		crypto.getRandomValues(challenge);
		challenge.fill(0, 4);
		const response = await this.query(COMMAND.encryptionData, challenge);
		this.assertAccepted(response, 'identification');
		const type = response[11] ?? 0xff;
		this.info = {
			cid: response[9] ?? 0,
			mid: response[10] ?? 0,
			type,
			dongleType: response[12] ?? 0,
			connection: type === 2 || type === 3 ? 'Wired' : 'Wireless',
			maximumPollingRateHz:
				({ 0: 1000, 1: 4000, 2: 1000, 3: 8000, 4: 2000, 5: 8000 } as Record<number, number>)[type] ?? 1000
		};
		return this.info;
	}

	// --- transport helpers ---

	private async withDeviceControl<T>(operation: () => Promise<T>): Promise<T> {
		await this.setDeviceOnline(true);
		try {
			return await operation();
		} finally {
			await this.setDeviceOnline(false).catch(() => undefined);
		}
	}

	private async setDeviceOnline(enabled: boolean): Promise<void> {
		let response: Uint8Array | null = null;
		for (let attempt = 0; attempt < 20; attempt += 1) {
			const packet = this.createPacket(COMMAND.deviceOnline);
			packet[5] = enabled ? 1 : 0;
			packet[15] = pulsarPacketChecksum(packet);
			response = await this.exchange(packet);
			this.assertAccepted(response, enabled ? 'host-control entry' : 'host-control exit');
			if (response[9] !== 1) break;
			await new Promise<void>((resolve) => window.setTimeout(resolve, 10));
		}
		if (!response || response[9] === 1) throw new Error('The Pulsar receiver stayed busy.');
		if (enabled && response[5] !== 1) throw new Error('The Pulsar mouse is offline. Move it or click a button, then retry.');
	}

	private async readFlash(address: number, length: number): Promise<Uint8Array> {
		const result = new Uint8Array(length);
		for (let offset = 0; offset < length; offset += 10) {
			const count = Math.min(10, length - offset);
			const packet = this.createPacket(COMMAND.readFlashData);
			const currentAddress = address + offset;
			packet[2] = currentAddress >> 8;
			packet[3] = currentAddress & 0xff;
			packet[4] = count;
			packet[15] = pulsarPacketChecksum(packet);
			const response = await this.exchange(packet);
			this.assertAccepted(response, 'configuration read');
			result.set(response.slice(5, 5 + count), offset);
		}
		return result;
	}

	private async writeFlash(address: number, data: Uint8Array): Promise<void> {
		for (let offset = 0; offset < data.length; offset += 10) {
			const chunk = data.slice(offset, offset + 10);
			const packet = this.createPacket(COMMAND.writeFlashData);
			const currentAddress = address + offset;
			packet[2] = currentAddress >> 8;
			packet[3] = currentAddress & 0xff;
			packet[4] = chunk.length;
			packet.set(chunk, 5);
			packet[15] = pulsarPacketChecksum(packet);
			this.assertAccepted(await this.exchange(packet), 'configuration write');
		}
	}

	private async writeCheckedByte(address: number, value: number): Promise<void> {
		await this.writeFlash(address, new Uint8Array([value, (0x55 - value) & 0xff]));
	}

	private async setVerifiedByte(address: number, value: number, label: string): Promise<number> {
		return await this.withDeviceControl(async () => {
			await this.writeCheckedByte(address, value);
			const confirmed = (await this.readFlash(address, 2))[0];
			if (confirmed !== value) throw new Error(`The mouse did not confirm the requested ${label}.`);
			return confirmed;
		});
	}

	private async setVerifiedBoolean(address: number, enabled: boolean, label: string): Promise<boolean> {
		return (await this.setVerifiedByte(address, enabled ? 1 : 0, label)) === 1;
	}

	private async query(command: number, parameters = new Uint8Array()): Promise<Uint8Array> {
		if (parameters.length > 10) throw new Error('Pulsar queries support at most 10 parameter bytes.');
		const packet = this.createPacket(command);
		packet[4] = parameters.length;
		packet.set(parameters, 5);
		packet[15] = pulsarPacketChecksum(packet);
		return await this.exchange(packet);
	}

	private async exchange(packet: Uint8Array): Promise<Uint8Array> {
		if (this.responseWaiter) throw new Error('Another Pulsar request is already in progress.');
		const command = packet[0];
		let timer: ReturnType<typeof setTimeout>;
		const response = new Promise<Uint8Array>((resolve, reject) => {
			timer = setTimeout(() => {
				this.responseWaiter = null;
				reject(new Error(`The Pulsar mouse did not answer command 0x${command.toString(16).padStart(2, '0')}.`));
			}, RESPONSE_TIMEOUT_MS);
			this.responseWaiter = {
				command,
				resolve: (bytes) => {
					clearTimeout(timer);
					resolve(bytes);
				},
				reject: (reason) => {
					clearTimeout(timer);
					reject(reason);
				}
			};
		});
		void response.catch(() => undefined);
		try {
			await this.device.sendReport(PULSAR_CONFIG_REPORT_ID, packet as unknown as BufferSource);
		} catch (error) {
			const detail = error instanceof Error ? error.message : String(error);
			const waiter = this.responseWaiter as { reject: (reason: Error) => void } | null;
			this.responseWaiter = null;
			waiter?.reject(new Error(`Chrome could not write Pulsar report 8. ${detail}`));
		}
		return await response;
	}

	private createPacket(command: number): Uint8Array {
		const packet = new Uint8Array(PULSAR_CONFIG_PACKET_LENGTH);
		packet[0] = command;
		return packet;
	}

	private decodeVersionOptional(label: string, response: Uint8Array | null): string | null {
		if (!response || response[1] !== 0) return null;
		return `${label} v${response[5] ?? 0}.${(response[6] ?? 0).toString(16).padStart(2, '0')}`;
	}

	private assertAccepted(response: Uint8Array, operation: string): void {
		if (response[1] !== 0) throw new Error(`The Pulsar receiver rejected the ${operation} (status ${response[1]}).`);
	}
}
