// Shared application state - a multi-device workspace where keyboards and
// mice are managed side by side as independent sessions.
import { get, writable } from 'svelte/store';
import { KeyboardController, type EffectParams } from './controller';
import { LogitechKeyboardController } from './logitech/controller';
import { LOGITECH_VID } from './logitech/constants';
import { LOGITECH_SUPPORTED_PIDS } from './logitech/devices';
import { RedragonKeyboardController } from './redragon/controller';
import { REDRAGON_VID } from './redragon/constants';
import { REDRAGON_SUPPORTED_PIDS } from './redragon/devices';
import { getKeyboard } from './razer/devices';
import { logger } from './razer/logger';
import { WebHidTransport } from './razer/transport';
import {
	MOUSE_HID_FILTERS,
	createMouseClient,
	deviceBrandOf,
	type MouseDriver,
	type MouseLighting,
	type MouseStatus
} from './mouse';
import type { MouseHidDevice } from './mouse/webhid';
import { DemoKeyboardController, DemoMouseDriver } from './demo';

export type { MouseDriver, MouseLighting, MouseStatus } from './mouse';

// --- Workspace sessions ---

export interface KeyboardSession {
	id: string;
	kind: 'keyboard';
	vendor: 'razer' | 'logitech' | 'redragon';
	pid: number;
	name: string;
	serial?: string;
	firmware?: string;
	/** Stable identity used to avoid connecting the same physical device twice. */
	deviceKey: string;
	controller: KeyboardController | LogitechKeyboardController | RedragonKeyboardController | DemoKeyboardController;
	/** Vendor-tagged device info, including the per-brand capability spec. */
	info:
		| (import('./controller').DeviceInfo & { vendor: 'razer' })
		| import('./logitech/controller').LogitechDeviceInfo
		| import('./redragon/controller').RedragonDeviceInfo;
}

export interface MouseSession {
	id: string;
	kind: 'mouse';
	brand: string;
	pid: number;
	name: string;
	serial?: string;
	deviceKey: string;
	client: MouseDriver;
	status: MouseStatus;
	dpiOptions: number[];
	busy: boolean;
}

export type DeviceSession = KeyboardSession | MouseSession;

/** One picker request lists every brand this app can drive. */
const REQUEST_FILTERS = [
	...MOUSE_HID_FILTERS,
	{ vendorId: 0x1532 },
	{ vendorId: LOGITECH_VID },
	{ vendorId: REDRAGON_VID }
];

let nextSessionId = 1;

export const sessions = writable<DeviceSession[]>([]);
export const activeSessionId = writable<string | null>(null);
export const error = writable<string | null>(null);
export const logVersion = writable(0);

// bump logVersion whenever the logger grows so the debug view re-renders
const origLog = logger.log.bind(logger);
logger.log = (level, message, hex, direction) => {
	origLog(level, message, hex, direction);
	logVersion.update((v) => v + 1);
};

type RawDevice = NonNullable<Parameters<WebHidTransport['open']>[0]>[number];

function deviceKeyOf(device: { vendorId: number; productId: number; serialNumber?: string; productName: string }): string {
	return [device.vendorId, device.productId, device.serialNumber || device.productName].join(':');
}

/**
 * Opens the browser's device chooser and adds every supported device the user
 * grants as a new workspace session, without disturbing existing ones.
 */
export async function addDevice(): Promise<void> {
	error.set(null);
	try {
		if (!navigator.hid) throw new Error('WebHID is not available in this browser.');
		const granted = (await navigator.hid.requestDevice({ filters: REQUEST_FILTERS })) as unknown as MouseHidDevice[];
		if (granted.length === 0) return; // chooser dismissed - not an error

		await ingestDevices(granted, { silent: false });
	} catch (err) {
		error.set(err instanceof Error ? err.message : String(err));
	}
}

/** True when the app runs inside the Electron shell. */
export function isDesktopApp(): boolean {
	return typeof navigator !== 'undefined' && /Electron/i.test(navigator.userAgent);
}

/**
 * Connects every HID device this origin may access without any click.
 * In the desktop shell Electron's device-permission handler exposes all
 * plugged-in devices; in browsers getDevices() returns everything the user
 * granted before, so returning visitors reconnect with zero clicks (the
 * very first grant still needs one gesture - enforced by Chromium).
 */
export async function detectConnectedDevices(): Promise<void> {
	if (!navigator.hid?.getDevices) return;
	error.set(null);
	try {
		const devices = (await navigator.hid.getDevices()) as unknown as MouseHidDevice[];
		if (devices.length > 0) await ingestDevices(devices, { silent: true });
	} catch (err) {
		logger.warn(`auto-detect failed: ${err instanceof Error ? err.message : String(err)}`);
	}
}

let hotplugBound = false;

/**
 * Reacts to devices being plugged/unplugged while running - supported
 * hardware is added automatically, removed hardware drops its session
 * (exactly like unplugging it from the workspace rail).
 */
export function bindDeviceHotplug(): void {
	if (hotplugBound || !navigator.hid?.addEventListener) return;
	hotplugBound = true;
	navigator.hid.addEventListener('connect', (event) => {
		void ingestDevices([event.device as unknown as MouseHidDevice], { silent: true });
	});
	navigator.hid.addEventListener('disconnect', (event) => {
		const key = deviceKeyOf(event.device as unknown as { vendorId: number; productId: number; serialNumber?: string; productName: string });
		for (const session of get(sessions)) {
			if (session.deviceKey === key && session.kind === 'keyboard') {
				void removeSession(session.id);
			} else if (
				session.kind === 'mouse' &&
				session.client.device.vendorId === event.device.vendorId &&
				session.client.device.productId === event.device.productId
			) {
				void removeSession(session.id);
			}
		}
	});
}

/**
 * Turns raw HID handles into workspace sessions for every brand this app can
 * drive, skipping anything already connected.
 */
async function ingestDevices(
	granted: MouseHidDevice[],
	opts: { silent: boolean }
): Promise<void> {
	const created: DeviceSession[] = [];
	const existingKeys = new Set(get(sessions).map((s) => s.deviceKey));

	const logiDevices = granted.filter((d) => d.vendorId === LOGITECH_VID && LOGITECH_SUPPORTED_PIDS.includes(d.productId));
	const redragonDevices = granted.filter((d) => d.vendorId === REDRAGON_VID && REDRAGON_SUPPORTED_PIDS.includes(d.productId));
	const razerDevices = granted.filter((d) => d.vendorId === 0x1532 && getKeyboard(d.productId) !== undefined);

	if (logiDevices.length > 0 && !existingKeys.has(deviceKeyOf(logiDevices[0]))) {
		const controller = new LogitechKeyboardController();
		const info = await controller.connect(logiDevices as never);
		created.push({
			id: `kb-${nextSessionId++}`,
			kind: 'keyboard',
			vendor: 'logitech',
			pid: info.pid,
			name: info.name,
			serial: info.serial,
			firmware: info.firmware,
			deviceKey: deviceKeyOf(logiDevices[0]),
			controller,
			info: { ...info, vendor: 'logitech' }
		});
	}
	if (redragonDevices.length > 0 && !existingKeys.has(deviceKeyOf(redragonDevices[0]))) {
		const controller = new RedragonKeyboardController();
		const info = await controller.connect(redragonDevices as never);
		created.push({
			id: `kb-${nextSessionId++}`,
			kind: 'keyboard',
			vendor: 'redragon',
			pid: info.pid,
			name: info.name,
			serial: info.serial,
			firmware: info.firmware,
			deviceKey: deviceKeyOf(redragonDevices[0]),
			controller,
			info: { ...info, vendor: 'redragon' }
		});
	}
	if (razerDevices.length > 0 && !existingKeys.has(deviceKeyOf(razerDevices[0]))) {
		const controller = new KeyboardController();
		const info = await controller.connect(razerDevices as unknown as Parameters<WebHidTransport['open']>[0]);
		created.push({
			id: `kb-${nextSessionId++}`,
			kind: 'keyboard',
			vendor: 'razer',
			pid: info.pid,
			name: info.name,
			serial: info.serial,
			firmware: info.firmware,
			deviceKey: deviceKeyOf(razerDevices[0]),
			controller,
			info: { ...info, vendor: 'razer' }
		});
	}
	if (created.length === 0) {
		// No keyboard matched - try the mouse drivers.
		const mouseCandidates = granted.filter((d) => !existingKeys.has(deviceKeyOf(d)));
		const device = mouseCandidates.find((d) => createMouseClient(d));
		if (device) {
			created.push(await activateMouse(device));
		} else if (!opts.silent) {
			throw new Error('No supported device was selected. Grant access to a Razer Chroma, Logitech G-series, Redragon RGB or supported gaming mouse.');
		}
	}

	if (created.length === 0) return;
	sessions.update((list) => [...list, ...created]);
	activeSessionId.set(created[created.length - 1].id);
}

/**
 * Adds simulated keyboard + mouse sessions so the workspace can be explored
 * without any physical hardware. Demo devices are marked "(Demo)".
 */
export async function addDemoDevices(): Promise<void> {
	error.set(null);
	try {
		const existingKeys = new Set(get(sessions).map((s) => s.deviceKey));

		const created: DeviceSession[] = [];
		if (!existingKeys.has(DEMO_KEYBOARD_KEY)) {
			const controller = new DemoKeyboardController();
			const info = await controller.connect();
			created.push({
				id: `kb-${nextSessionId++}`,
				kind: 'keyboard',
				vendor: 'razer',
				pid: info.pid,
				name: `${info.name} (Demo)`,
				serial: info.serial,
				firmware: info.firmware,
				deviceKey: DEMO_KEYBOARD_KEY,
				controller,
				info: { ...info, vendor: 'razer' }
			});
		}
		if (!existingKeys.has(DEMO_MOUSE_KEY)) {
			const client = new DemoMouseDriver();
			await client.open();
			const status = await client.readStatus();
			created.push({
				id: `ms-${nextSessionId++}`,
				kind: 'mouse',
				brand: 'Demo',
				pid: client.device.productId,
				name: `${status.name} (Demo)`,
				serial: client.device.serialNumber,
				deviceKey: DEMO_MOUSE_KEY,
				client,
				status,
				dpiOptions: client.getDpiOptions(),
				busy: false
			});
		}

		if (created.length === 0) return;
		sessions.update((list) => [...list, ...created]);
		activeSessionId.set(created[0].id);
		logger.info('Demo mode enabled - devices are simulated, no hardware is touched.');
	} catch (err) {
		error.set(err instanceof Error ? err.message : String(err));
	}
}

const DEMO_KEYBOARD_KEY = 'demo:keyboard';
const DEMO_MOUSE_KEY = 'demo:mouse';

async function activateMouse(device: MouseHidDevice): Promise<MouseSession> {
	const client = createMouseClient(device);
	if (!client) throw new Error('The selected device has no supported mouse driver.');
	// Every driver opens idempotently; some protocols also need an input-report
	// listener registered before the first exchange.
	await client.open();
	let status: MouseStatus;
	try {
		status = await client.readStatus();
	} catch (err) {
		await client.close().catch(() => undefined);
		throw err;
	}
	return {
		id: `ms-${nextSessionId++}`,
		kind: 'mouse',
		brand: deviceBrandOf(client),
		pid: device.productId,
		name: status.name || device.productName || 'Gaming mouse',
		serial: device.serialNumber || undefined,
		deviceKey: deviceKeyOf(device),
		client,
		status,
		dpiOptions: client.getDpiOptions?.() ?? [],
		busy: false
	};
}

/** Disconnects one device and removes it from the workspace. */
export async function removeSession(id: string): Promise<void> {
	const session = get(sessions).find((s) => s.id === id);
	if (!session) return;
	if (session.kind === 'keyboard') await session.controller.disconnect().catch(() => undefined);
	else await session.client.close().catch(() => undefined);
	sessions.update((list) => list.filter((s) => s.id !== id));
	if (get(activeSessionId) === id) {
		const remaining = get(sessions);
		activeSessionId.set(remaining[remaining.length - 1]?.id ?? null);
	}
}

/** Re-read the live status of one connected mouse. */
export async function refreshMouseStatus(sessionId: string): Promise<MouseStatus | null> {
	const session = get(sessions).find((s): s is MouseSession => s.id === sessionId && s.kind === 'mouse');
	if (!session) return null;
	const status = await session.client.readStatus();
	updateMouseSession(sessionId, { status });
	return status;
}

/**
 * Run a driver setter against one connected mouse, then re-read its status so
 * the UI reflects what the hardware actually accepted.
 */
export async function applyMouseSetting<T>(sessionId: string, fn: (client: MouseDriver) => Promise<T>): Promise<T | null> {
	const session = get(sessions).find((s): s is MouseSession => s.id === sessionId && s.kind === 'mouse');
	if (!session) return null;
	updateMouseSession(sessionId, { busy: true });
	error.set(null);
	try {
		const result = await fn(session.client);
		await refreshMouseStatus(sessionId).catch(() => undefined);
		return result;
	} catch (err) {
		error.set(err instanceof Error ? err.message : String(err));
		return null;
	} finally {
		updateMouseSession(sessionId, { busy: false });
	}
}

function updateMouseSession(sessionId: string, patch: Partial<MouseSession>): void {
	sessions.update((list) => list.map((s) => (s.id === sessionId && s.kind === 'mouse' ? { ...s, ...patch } : s)));
}
