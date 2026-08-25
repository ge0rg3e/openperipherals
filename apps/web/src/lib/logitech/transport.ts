import { LOGITECH_VID } from './constants';
import { getLogitechKeyboard, LOGITECH_SUPPORTED_PIDS, type LogitechDevice } from './devices';
import { logger } from '../razer/logger';
import type { LogiReport } from './report';

export interface LogitechHandle {
	name: string;
	pid: number;
	productName: string;
	serial?: string;
	layout: LogitechDevice | undefined;
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

export class LogitechTransport {
	private devices: HidDevice[] = [];
	private handleValue: LogitechHandle | null = null;

	get handle(): LogitechHandle | undefined {
		return this.handleValue ?? undefined;
	}

	get connected(): boolean {
		return this.devices.some((d) => d.opened);
	}

	async open(granted?: HidDevice[]): Promise<LogitechHandle> {
		if (!webhidSupported() || !navigator.hid) {
			throw new Error('WebHID is not available in this browser context.');
		}
		const request = navigator.hid.requestDevice as unknown as HidRequest;
		const requested = granted ?? (await request({ filters: [{ vendorId: LOGITECH_VID }] }));
		const candidates = requested.filter((d) => LOGITECH_SUPPORTED_PIDS.includes(d.productId));
		if (candidates.length === 0) {
			throw new Error('No supported Logitech device was granted. Choose a Logitech RGB device from the picker.');
		}
		// Open every granted interface. A board like the G512/G610/G810 exposes
		// the 20-byte and 64-byte control reports as separate top-level
		// collections, and each report must be written to the interface that
		// owns it, so we keep them all and route per report id.
		for (const device of candidates) {
			await device.open();
			this.devices.push(device);
			this.attachInputListener(device);
		}
		const device = candidates[0];
		const spec = getLogitechKeyboard(device.productId);
		this.handleValue = {
			name: spec?.name ?? device.productName,
			pid: device.productId,
			productName: device.productName,
			serial: undefined,
			layout: spec
		};
		logger.info(`Logitech transport opened: ${this.handleValue.name} (0x${device.productId.toString(16).padStart(4, '0')})`);
		return this.handleValue;
	}

	/** The opened interface that declares this output report id (falls back to the first one). */
	private pick(reportId: number): HidDevice {
		const byId = this.devices.find((d) => d.collections?.some((c) => c.outputReports?.some((r) => r.reportId === reportId)));
		return byId ?? this.devices[0];
	}

	async send(report: LogiReport): Promise<void> {
		if (this.devices.length === 0) throw new Error('Logitech transport is not open.');
		const device = this.pick(report.reportId);
		await device.sendReport(report.reportId, report.data);
		const full = new Uint8Array(report.data.length + 1);
		full[0] = report.reportId;
		full.set(report.data, 1);
		logger.tx(`logi report 0x${report.reportId.toString(16)}`, full);
	}

	async sendBurst(report: LogiReport): Promise<void> {
		return this.send(report);
	}

	private attachInputListener(device: HidDevice): void {
		const handler = (event: Event) => {
			const e = event as Event & { reportId?: number; data?: DataView };
			if (e.data) {
				const bytes = new Uint8Array(e.data.buffer, e.data.byteOffset, e.data.byteLength);
				logger.rx(`logi input 0x${(e.reportId ?? 0).toString(16)}`, bytes);
			}
		};
		device.addEventListener?.('inputreport', handler);
	}

	async close(): Promise<void> {
		for (const device of this.devices) {
			if (device.opened) await device.close();
		}
		this.devices = [];
		this.handleValue = null;
	}
}
