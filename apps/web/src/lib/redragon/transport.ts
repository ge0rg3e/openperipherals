import { REDRAGON_USAGE_PAGE, REDRAGON_VID } from './constants';
import { getRedragonKeyboard, REDRAGON_SUPPORTED_PIDS, type RedragonDevice } from './devices';
import { logger } from '../razer/logger';
import type { EvReport } from './report';

export interface RedragonHandle {
	name: string;
	pid: number;
	productName: string;
	layout: RedragonDevice | undefined;
}

interface HidCollection {
	usagePage?: number;
	usage?: number;
	type?: string;
	outputReports?: Array<{ reportId?: number }>;
	children?: unknown[];
}

export interface HidDevice {
	vendorId: number;
	productId: number;
	productName: string;
	opened: boolean;
	open: () => Promise<void>;
	close: () => Promise<void>;
	sendReport: (reportId: number, data: BufferSource) => Promise<void>;
	addEventListener?: (type: string, callback: EventListenerOrEventListenerObject) => void;
	collections?: HidCollection[];
}

type HidRequest = (opts: { filters?: Array<{ vendorId?: number; productId?: number }> }) => Promise<HidDevice[]>;

function webhidSupported(): boolean {
	return typeof window !== 'undefined' && (window.isSecureContext || location.hostname === 'localhost' || location.hostname === '127.0.0.1');
}

export class RedragonTransport {
	private devices: HidDevice[] = [];
	private active: HidDevice | null = null;
	private handleValue: RedragonHandle | null = null;

	get handle(): RedragonHandle | undefined {
		return this.handleValue ?? undefined;
	}

	get connected(): boolean {
		return !!this.active?.opened;
	}

	async open(granted?: HidDevice[]): Promise<RedragonHandle> {
		if (!webhidSupported() || !navigator.hid) {
			throw new Error('WebHID is not available in this browser context.');
		}
		const request = navigator.hid.requestDevice as unknown as HidRequest;
		const requested = granted ?? (await request({ filters: [{ vendorId: REDRAGON_VID }] }));
		// The JESS/Sonix VID is shared by many non-keyboard devices; only keep
		// product ids we know are RGB keyboards.
		const candidates = requested.filter((d) => d.vendorId === REDRAGON_VID && REDRAGON_SUPPORTED_PIDS.includes(d.productId));
		if (candidates.length === 0) {
			throw new Error('No supported Redragon device was granted. Choose a Redragon RGB device from the picker.');
		}
		for (const device of candidates) {
			try {
				if (!device.opened) await device.open();
				this.devices.push(device);
				this.attachInputListener(device);
			} catch (err) {
				logger.warn(`Redragon connect: could not open ${device.productName}: ${err}`);
			}
		}
		if (this.devices.length === 0) {
			throw new Error('Could not open the granted Redragon device. Close any other app using it and retry.');
		}
		// Prefer the interface that declares the 0xFF1C vendor collection (the
		// lighting control); fall back to the first granted handle.
		this.active = this.devices.find((d) => d.collections?.some((c) => c.usagePage === REDRAGON_USAGE_PAGE)) ?? this.devices[0];
		const spec = getRedragonKeyboard(this.active.productId);
		// productName reported by the device is more specific than a shared PID
		// entry (e.g. "Redragon K587", "Redragon K556"), so prefer it.
		const name = this.active.productName?.trim() || spec?.name || `Redragon Device 0x${this.active.productId.toString(16).padStart(4, '0')}`;
		this.handleValue = { name, pid: this.active.productId, productName: name, layout: spec };
		logger.info(`Redragon transport opened: ${name} (0x${this.active.productId.toString(16).padStart(4, '0')})`);
		return this.handleValue;
	}

	async send(report: EvReport): Promise<void> {
		if (!this.active) throw new Error('Redragon transport is not open.');
		await this.active.sendReport(report.reportId, report.data);
		const full = new Uint8Array(report.data.length + 1);
		full[0] = report.reportId;
		full.set(report.data, 1);
		logger.tx(`redragon report 0x${report.reportId.toString(16)}`, full);
	}

	async sendBurst(report: EvReport): Promise<void> {
		return this.send(report);
	}

	private attachInputListener(device: HidDevice): void {
		const handler = (event: Event) => {
			const e = event as Event & { reportId?: number; data?: DataView };
			if (e.data) {
				const bytes = new Uint8Array(e.data.buffer, e.data.byteOffset, e.data.byteLength);
				logger.rx(`redragon input 0x${(e.reportId ?? 0).toString(16)}`, bytes);
			}
		};
		device.addEventListener?.('inputreport', handler);
	}

	async close(): Promise<void> {
		for (const device of this.devices) {
			if (device.opened) {
				try {
					await device.close();
				} catch {
					/* ignore */
				}
			}
		}
		this.devices = [];
		this.active = null;
		this.handleValue = null;
	}
}