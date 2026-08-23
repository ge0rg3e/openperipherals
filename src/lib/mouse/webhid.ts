/**
 * Local WebHID surface for the mouse module.
 *
 * The rest of this app declares only a minimal `Navigator.hid` global (see
 * `$lib/razer/transport`), so the mouse drivers carry their own fuller device
 * typing here and cast at the single entry point that talks to the browser.
 */

export interface HidReportInfo {
	reportId: number;
}

export interface HidCollectionInfo {
	usagePage: number;
	usage: number;
	inputReports: readonly HidReportInfo[];
	outputReports: readonly HidReportInfo[];
	featureReports?: readonly unknown[];
	children?: readonly unknown[];
}

export interface HidInputReportEvent extends Event {
	readonly device: MouseHidDevice;
	readonly reportId: number;
	readonly data: DataView;
}

export interface MouseHidDevice {
	readonly vendorId: number;
	readonly productId: number;
	readonly productName: string;
	readonly serialNumber?: string;
	readonly opened: boolean;
	open(): Promise<void>;
	close(): Promise<void>;
	sendReport(reportId: number, data: BufferSource): Promise<void>;
	sendFeatureReport(reportId: number, data: BufferSource): Promise<void>;
	receiveFeatureReport(reportId: number): Promise<DataView>;
	collections: readonly HidCollectionInfo[];
	addEventListener(type: 'inputreport', listener: (event: HidInputReportEvent) => void): void;
	removeEventListener(type: 'inputreport', listener: (event: HidInputReportEvent) => void): void;
}

export interface MouseHidFilter {
	vendorId?: number;
	productId?: number;
	usagePage?: number;
	usage?: number;
}

/** Requests device access through the browser picker; empty when refused. */
export async function requestMouseDevices(filters: MouseHidFilter[]): Promise<MouseHidDevice[]> {
	const granted = await navigator.hid?.requestDevice({ filters });
	return (granted ?? []) as unknown as MouseHidDevice[];
}
