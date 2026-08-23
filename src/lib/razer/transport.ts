import { packReport, type CommandReport } from './report';
import { logger } from './logger';
import { getKeyboard } from './devices';

export function webhidContextOk(): boolean {
	return typeof window !== 'undefined' && (window.isSecureContext || location.hostname === 'localhost' || location.hostname === '127.0.0.1');
}

export interface TransportHandle {
	name: string;
	pid: number;
	productName: string;
	serial?: string;
	close: () => Promise<void>;
}

export abstract class Transport {
	abstract open(): Promise<void>;
	abstract send(report: CommandReport): Promise<Uint8Array | null>;
	abstract close(): Promise<void>;
	abstract get handle(): TransportHandle | undefined;
	abstract get connected(): boolean;
	/** optional high-rate, write-only send (no feature-read round trip). */
	sendBurst(report: CommandReport): Promise<void> | void {
		void report;
	}
}

interface HidCollection {
	usagePage?: number;
	usage?: number;
	type?: string;
	children?: unknown[];
}

interface HidDevice {
	vendorId: number;
	productId: number;
	productName: string;
	opened: boolean;
	open: () => Promise<void>;
	close: () => Promise<void>;
	sendReport: (reportId: number, data: BufferSource) => Promise<void>;
	sendFeatureReport?: (reportId: number, data: BufferSource) => Promise<void>;
	receiveFeatureReport?: (reportId: number) => Promise<DataView>;
	collections?: HidCollection[];
}

type WebHidRequestParams = {
	filters?: Array<{ vendorId?: number; productId?: number; usagePage?: number; usage?: number }>;
	exclusionFilters?: unknown[];
};

declare global {
	interface Navigator {
		hid?: {
			requestDevice: (opts: WebHidRequestParams) => Promise<HidDevice[]>;
			getDevices: () => Promise<HidDevice[]>;
		};
	}
}

export function webhidSupported(): boolean {
	return typeof navigator !== 'undefined' && !!navigator.hid;
}

export class WebHidTransport extends Transport {
	private devices: HidDevice[] = [];
	private active: HidDevice | null = null;
	private handleValue: TransportHandle | undefined;
	private connectedFlag = false;

	async open(granted?: HidDevice[]): Promise<void> {
		if (!webhidSupported()) {
			throw new Error('WebHID is not available. Use Chrome/Edge on a supported desktop OS.');
		}
		if (!webhidContextOk()) {
			throw new Error('WebHID requires a secure context. Open the page over https:// or http://localhost.');
		}
		const grantedRaw =
			granted ??
			(await navigator.hid!.requestDevice({
				filters: [{ vendorId: 0x1532 }]
			}));
		if (!grantedRaw.length) throw new Error('No Razer device selected.');

		// Only use handles that are known Razer *keyboards*. Peripheral headsets etc.
		// share the same vendor id but don't implement matrix lighting and would
		// swallow/interleave commands, causing random keys to flip on/off.
		const candidates: HidDevice[] = [];
		for (const d of grantedRaw) {
			if (!getKeyboard(d.productId)) {
				logger.info(`Skipping unsupported Razer device: ${d.productName || '0x' + d.productId.toString(16)}`);
				continue;
			}
			try {
				if (!d.opened) await d.open();
				candidates.push(d);
			} catch (err) {
				const detail = err instanceof Error ? err.message : String(err);
				logger.warn(`device.open() failed for ${d.productName || 'device'}: ${detail}`);
			}
		}
		if (!candidates.length) {
			throw new Error('Could not open a supported Razer device. Grant access to your device (e.g. Huntsman, BlackWidow) - headsets/mice are ignored.');
		}
		// Pin a single keyboard; never hop between devices during writes.
		this.devices = candidates;
		this.active = candidates[0];
		this.connectedFlag = true;
		this.handleValue = {
			name: this.active.productName || `Razer 0x${this.active.productId.toString(16).padStart(4, '0')}`,
			pid: this.active.productId,
			productName: this.active.productName,
			close: () => Promise.resolve().then(() => this.close())
		};
		logger.info(`Connected to ${this.handleValue.name} (${candidates.length} HID handle[s])`);
	}

	async send(report: CommandReport): Promise<Uint8Array | null> {
		if (!this.active) throw new Error('Not connected.');
		const bytes = packReport(report);
		// The device may expose several HID interfaces; Chrome binds `active` to
		// whichever was handed out first, which isn't always the lighting control
		// collection. Probe each granted handle and pin the first one that
		// accepts the write so subsequent sends use the working interface.
		const order = this.devices.indexOf(this.active) >= 0 ? [...this.devices] : [...this.devices];
		let firstErr: unknown = null;
		for (const d of order) {
			try {
				if (!d.opened) await d.open();
				logger.tx(`→ ${d.productName} (feature id 0, ${bytes.length} bytes)`, bytes);
				await d.sendFeatureReport!(0, bytes.buffer as ArrayBuffer);
				this.active = d;
				this.handleValue!.name = d.productName || this.handleValue!.name;
				return await this.readIf(d);
			} catch (err) {
				if (firstErr === null) firstErr = err;
				logger.warn(`interface ${d.productName} rejected write: ${err}`);
			}
		}
		throw new Error(
			`Could not write to “${this.handleValue!.name}”. ${firstErr instanceof Error ? firstErr.message : String(firstErr)} ` +
				'Make sure no other app holds the device and the control HID interface is granted.'
		);
	}

	async sendBurst(report: CommandReport): Promise<void> {
		if (!this.active) return;
		const bytes = packReport(report);
		for (const d of this.devices) {
			try {
				if (!d.opened) await d.open();
				await d.sendFeatureReport!(0, bytes.buffer as ArrayBuffer);
				this.active = d;
				this.handleValue!.name = d.productName || this.handleValue!.name;
				await this.drainRead(d);
				return;
			} catch (err) {
				this.warnBurstOnce(d.productName, err);
			}
		}
	}

	private lastBurstWarn = 0;

	private warnBurstOnce(name: string, err: unknown): void {
		// Repeated failures from the render loop would otherwise flood the log.
		const now = Date.now();
		if (now - this.lastBurstWarn < 2000) return;
		this.lastBurstWarn = now;
		logger.warn(`burst write rejected by ${name}: ${err}`);
	}

	private async drainRead(d: HidDevice): Promise<void> {
		if (typeof d.receiveFeatureReport !== 'function') return;
		try {
			// brief pause so the firmware fills the reply buffer
			await new Promise((r) => setTimeout(r, 5));
			await d.receiveFeatureReport(0);
		} catch {
			/* best-effort: a missed read only delays the next write */
		}
	}

	private async readIf(d: HidDevice): Promise<Uint8Array | null> {
		if (typeof d.receiveFeatureReport !== 'function') return null;
		try {
			// small delay so the device fills the response buffer (as in openId reference)
			await new Promise((r) => setTimeout(r, 20));
			const view = await d.receiveFeatureReport(0);
			const out = new Uint8Array(view!.buffer.slice(0));
			logger.rx(`← ${d.productName} ${out.length} bytes`, out);
			return out;
		} catch (err) {
			logger.warn(`No feature response from ${d.productName}: ${err}`);
			return null;
		}
	}

	get handle(): TransportHandle | undefined {
		return this.handleValue;
	}

	get connected(): boolean {
		return this.connectedFlag;
	}

	async close(): Promise<void> {
		for (const d of this.devices) {
			try {
				await d.close();
			} catch {
				/* ignore */
			}
		}
		this.devices = [];
		this.active = null;
		this.connectedFlag = false;
		this.handleValue = undefined;
	}
}

export type ResponseBuilder = (report: CommandReport) => Uint8Array | null;
