/**
 * WebHID driver for Razer's 90-byte feature-report mice.
 *
 * The control channel is the interface whose only collection is Generic
 * Desktop Mouse; older models carry it on a vendor-defined collection instead.
 * Exchanges serialise through a queue, retry busy replies, and every setter
 * confirms by read-back.
 */

import type { LiftOffDistance, MouseDriver, MouseLighting, MouseLightingMode, MouseStatus } from './types';
import type { MouseHidDevice } from './webhid';
import {
	RAZER_READ,
	RAZER_REPORT_ID,
	RAZER_STATUS,
	RAZER_STORAGE,
	RAZER_TRACKING_DISTANCES,
	RazerProtocolError,
	decodeBatteryPercent,
	decodeCharging,
	decodeDpi,
	decodeExtendedPollingRate,
	decodeFirmwareVersion,
	decodeLegacyPollingRate,
	decodeLiftOff,
	decodeRazerResponse,
	decodeSleepTimeout,
	encodeRazerRequest,
	isRazerGetter,
	razerReadDpiCommand,
	razerSetDpiCommand,
	razerSetExtendedEffectCommand,
	razerSetExtendedPollingCommand,
	razerSetLegacyPollingCommand,
	razerSetSleepTimeoutCommand,
	razerSetTrackingDistanceCommand,
	type RazerCommand,
	type RazerExtendedEffect,
	type RazerReactiveSpeed,
	type RazerTrackingDistance
} from './razer-codec';
import { RATES_1K, RAZER_PRODUCTS } from './razer-devices';

const DPI_MIN = 100;
const RESPONSE_DELAY_MS = 100;
const RESPONSE_ATTEMPTS = 6;
const POLLING_LINK_SETTLE_MS = 150;

// The vendor software slides from 1 to 15 minutes.
const SLEEP_OPTIONS: readonly number[] = Array.from({ length: 15 }, (_, index) => (index + 1) * 60);

/** Razer exposes its control channel on the plain mouse interface... */
function isMouseControlInterface(device: MouseHidDevice): boolean {
	const [collection, ...rest] = device.collections;
	return rest.length === 0 && collection?.usagePage === 0x01 && collection?.usage === 0x02;
}

/** ...or on a vendor-defined one, varying by hardware revision. */
function hasVendorCollection(device: MouseHidDevice): boolean {
	return device.collections.some((collection) => (collection.usagePage ?? 0) >= 0xff00);
}

export class RazerMouseClient implements MouseDriver {
	private queue: Promise<unknown> = Promise.resolve();
	private readonly staticReads = new Map<string, Promise<Uint8Array | null>>();
	private lightingCache: MouseLighting | null = null;

	readonly device: MouseHidDevice;

	constructor(device: MouseHidDevice) {
		this.device = device;
	}

	static isSupported(device: MouseHidDevice): boolean {
		const product = RAZER_PRODUCTS.get(device.productId);
		if (device.vendorId !== 0x1532 || !product) return false;
		return isMouseControlInterface(device) || (product.vendorControlInterface === true && hasVendorCollection(device));
	}

	private profile() {
		return RAZER_PRODUCTS.get(this.device.productId);
	}

	async open(): Promise<void> {
		if (!this.device.opened) await this.device.open();
	}

	async close(): Promise<void> {
		this.staticReads.clear();
		this.lightingCache = null;
		if (this.device.opened) await this.device.close();
	}

	isWireless(): boolean {
		return this.profile()?.wireless ?? false;
	}

	maxDpi(): number {
		return this.profile()?.maxDpi ?? 30_000;
	}

	getSupportedPollingRates(): number[] {
		return [...(this.profile()?.pollingRates ?? RATES_1K)];
	}

	getSleepOptions(): readonly number[] {
		return SLEEP_OPTIONS;
	}

	getDpiOptions(): number[] {
		// The sensor takes any whole DPI up to the model's ceiling.
		const options: number[] = [];
		for (let dpi = DPI_MIN; dpi <= this.maxDpi(); dpi += 1) options.push(dpi);
		return options;
	}

	async readStatus(): Promise<MouseStatus> {
		await this.open();
		const wireless = this.isWireless();
		const firmware = await this.once('firmware', () => this.request(RAZER_READ.firmware));
		if (!firmware) throw new Error('The mouse did not report a firmware version.');
		// A wired mouse with no cell answers battery and power commands as
		// unsupported, which must not abort the whole status read.
		const battery = await this.readBattery().catch(() => null);
		const sleep = await this.request(RAZER_READ.sleepTimeout).catch(() => null);
		const dpi = decodeDpi(await this.request(razerReadDpiCommand(this.dpiStorageByte())));
		const pollingRateHz = await this.readPollingRateHz();
		// A model without lift-off can still answer class 0x0b with an all-zero
		// payload that decodes as "Low", so the control is gated per product.
		const liftOff = this.profile()?.liftOff === true ? await this.readLiftOff().catch(() => null) : null;
		return {
			brand: 'Razer',
			name: this.displayName(),
			ui: {
				family: 'razer',
				hideUnsupportedPollingRates: true,
				// No sensor-processing command is confirmed for this protocol.
				hideProcessingCard: true
			},
			batteryPercent: battery ? Math.min(battery.percent, 100) : null,
			batteryState: battery?.state ?? 'Unknown',
			dpi: dpi.x,
			dpiY: dpi.y,
			pollingRateHz,
			supportedPollingRates: this.getSupportedPollingRates(),
			activeProfile: null,
			connectionType: wireless ? 'Wireless' : 'Wired',
			sleepTimeout: sleep ? decodeSleepTimeout(sleep) : null,
			liftOffDistance: liftOff?.tracking ?? null,
			supportedLiftOffDistances: liftOff ? [...RAZER_TRACKING_DISTANCES] : [],
			lighting: this.lightingCache ?? this.defaultLighting(),
			firmware: [`Mouse ${decodeFirmwareVersion(firmware)}`]
		};
	}

	async setDpi(dpi: number, dpiY: number = dpi): Promise<number> {
		const ceiling = this.maxDpi();
		for (const value of [dpi, dpiY]) {
			if (!Number.isInteger(value) || value < DPI_MIN || value > ceiling) {
				throw new Error(`DPI must be a whole number between ${DPI_MIN} and ${ceiling.toLocaleString()}.`);
			}
		}
		await this.request(razerSetDpiCommand(dpi, dpiY, this.dpiStorageByte()));
		const confirmed = decodeDpi(await this.request(razerReadDpiCommand(this.dpiStorageByte())));
		if (confirmed.x !== dpi || confirmed.y !== dpiY) {
			throw new Error(`The mouse kept ${confirmed.x.toLocaleString()} DPI instead of ${dpi.toLocaleString()}.`);
		}
		return confirmed.x;
	}

	async setPollingRate(pollingRateHz: number): Promise<number> {
		if (!this.getSupportedPollingRates().includes(pollingRateHz)) {
			throw new Error(`This mouse does not support ${pollingRateHz.toLocaleString()} Hz on this connection.`);
		}
		const highRate = this.profile()?.highRatePolling ?? this.isWireless();
		await this.request(highRate ? razerSetExtendedPollingCommand(pollingRateHz) : razerSetLegacyPollingCommand(pollingRateHz));
		// Changing the rate briefly reconfigures the wireless link; let it settle
		// before the read-back or the exchange comes back corrupt.
		if (this.isWireless()) await delay(POLLING_LINK_SETTLE_MS);
		const confirmed = await this.readPollingRateHz();
		if (confirmed !== pollingRateHz) {
			throw new Error(`The mouse kept ${confirmed.toLocaleString()} Hz instead of ${pollingRateHz.toLocaleString()} Hz.`);
		}
		return confirmed;
	}

	async setLiftOffDistance(distance: LiftOffDistance): Promise<LiftOffDistance> {
		await this.request(razerSetTrackingDistanceCommand(distance));
		const confirmed = decodeLiftOff(await this.request(RAZER_READ.liftOff));
		if (confirmed.tracking !== distance) {
			throw new Error(`The mouse kept ${confirmed.tracking ?? 'an unknown'} tracking distance instead of ${distance}.`);
		}
		return distance;
	}

	async setSleepTimeout(seconds: number): Promise<number> {
		if (!SLEEP_OPTIONS.includes(seconds)) throw new Error('Auto sleep must be between 1 and 15 minutes.');
		await this.request(razerSetSleepTimeoutCommand(seconds));
		const confirmed = decodeSleepTimeout(await this.request(RAZER_READ.sleepTimeout));
		if (confirmed !== seconds) throw new Error(`The mouse kept ${confirmed} seconds instead of ${seconds}.`);
		return confirmed;
	}

	/**
	 * Writes an underglow effect. Razer's effect commands are writes without a
	 * matching read, so the result is cached driver-side (`writeOnly`) and the
	 * panel renders the cache until the next write.
	 */
	async setLighting(lighting: MouseLighting): Promise<MouseLighting> {
		const effect = lightingModeToEffect(lighting.mode);
		if (!effect) throw new Error(`${lighting.mode} is not a lighting mode this driver can send.`);
		await this.request(
			razerSetExtendedEffectCommand(effect, {
				color: lighting.color ?? undefined,
				color2: lighting.color2 ?? undefined,
				speed: (lighting.speed ?? 3) as RazerReactiveSpeed
			})
		);
		this.lightingCache = { ...lighting };
		return this.lightingCache;
	}

	// --- internals ---

	private displayName(): string {
		const known = this.profile();
		return known ? `Razer ${known.model}` : this.device.productName || 'Razer';
	}

	private dpiStorageByte(): number {
		return this.profile()?.dpiStorageByte ?? RAZER_STORAGE;
	}

	private defaultLighting(): MouseLighting {
		return {
			zone: 'Underglow',
			modes: ['Off', 'Static', 'Spectrum', 'Reactive', 'Breathing random', 'Breathing single', 'Breathing dual'],
			mode: null,
			color: null,
			color2: null,
			colorModes: ['Static', 'Reactive', 'Breathing single'],
			dualColorModes: ['Breathing dual'],
			reactiveModes: ['Reactive'],
			speeds: [1, 2, 3, 4],
			speed: null,
			writeOnly: true
		};
	}

	private async readBattery(): Promise<{ percent: number; state: MouseStatus['batteryState'] }> {
		const level = await this.request(RAZER_READ.battery);
		const charging = await this.request(RAZER_READ.charging).catch(() => null);
		return {
			percent: decodeBatteryPercent(level),
			state:
				charging === null ? 'Unknown' : decodeCharging(charging) ? 'Charging' : 'Discharging'
		};
	}

	private async readLiftOff(): Promise<{ tracking: RazerTrackingDistance | null } | null> {
		const reply = await this.request(RAZER_READ.liftOff).catch(() => null);
		return reply ? decodeLiftOff(reply) : null;
	}

	/**
	 * Wired answers only the legacy polling command and receivers only the
	 * extended one, so ask for the expected one first and keep the other as a
	 * fallback.
	 */
	private async readPollingRateHz(): Promise<number> {
		const extended: readonly [RazerCommand, typeof decodeExtendedPollingRate, boolean] = [
			RAZER_READ.pollingRateExtended,
			decodeExtendedPollingRate,
			true
		];
		const legacy: readonly [RazerCommand, typeof decodeLegacyPollingRate, boolean] = [
			RAZER_READ.pollingRate,
			decodeLegacyPollingRate,
			false
		];
		const preferExtended = this.profile()?.highRatePolling ?? this.isWireless();
		for (const [command, decode] of preferExtended ? [extended, legacy] : [legacy, extended]) {
			const reply = await this.request(command).catch(() => null);
			if (!reply) continue;
			return decode(reply);
		}
		throw new Error('The mouse did not report a polling rate.');
	}

	private once(key: string, read: () => Promise<Uint8Array | null>): Promise<Uint8Array | null> {
		const pending = this.staticReads.get(key);
		if (pending) return pending;
		const started = read();
		this.staticReads.set(key, started);
		started.catch(() => this.staticReads.delete(key));
		return started;
	}

	private async request(command: RazerCommand): Promise<Uint8Array> {
		const run = this.queue.then(
			() => this.exchange(command),
			() => this.exchange(command)
		);
		this.queue = run.catch(() => undefined);
		return await run;
	}

	private async exchange(command: RazerCommand): Promise<Uint8Array> {
		await this.open();
		const transactionId = this.profile()?.transactionId;
		const request = encodeRazerRequest(command, transactionId);
		// A corrupt reply means the exchange itself was lost — the receiver can
		// return garbage while it reconfigures the link — so the send retries.
		for (let attempt = 0; attempt < RESPONSE_ATTEMPTS; attempt += 1) {
			await this.device.sendFeatureReport(RAZER_REPORT_ID, request as unknown as BufferSource);
			const reply = await this.awaitReply(command);
			if (reply) return reply;
		}
		throw new Error('The mouse stayed busy — it may be asleep or out of range.');
	}

	private async awaitReply(command: RazerCommand): Promise<Uint8Array | null> {
		for (let read = 0; read < RESPONSE_ATTEMPTS; read += 1) {
			await delay(RESPONSE_DELAY_MS);
			const view = await this.device.receiveFeatureReport(RAZER_REPORT_ID);
			const reply = new Uint8Array(view.buffer.slice(view.byteOffset, view.byteOffset + view.byteLength));
			try {
				return decodeRazerResponse(reply, command);
			} catch (error) {
				if (!(error instanceof RazerProtocolError)) throw error;
				if (error.status === RAZER_STATUS.busy) continue;
				// A bad checksum or wrong length carries no status — transport noise.
				if (error.status === null) return null;
				// Getters may be repeated; setters must not be re-sent when an
				// earlier reply is still buffered.
				if (error.stale && isRazerGetter(command)) return null;
				throw error;
			}
		}
		return null;
	}
}

function lightingModeToEffect(mode: MouseLightingMode | null): RazerExtendedEffect | null {
	switch (mode) {
		case 'Off':
			return 'off';
		case 'Static':
			return 'static';
		case 'Spectrum':
		case 'Cycling':
			return 'spectrum';
		case 'Reactive':
			return 'reactive';
		case 'Breathing random':
			return 'breathing-random';
		case 'Breathing single':
			return 'breathing-single';
		case 'Breathing dual':
			return 'breathing-dual';
		default:
			return null;
	}
}

function delay(milliseconds: number): Promise<void> {
	return new Promise((resolve) => setTimeout(resolve, milliseconds));
}
