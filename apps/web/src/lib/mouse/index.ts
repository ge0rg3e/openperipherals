/**
 * Mouse driver registry.
 *
 * Detects which vendored protocol family a granted HID device speaks and
 * instantiates the matching driver. Add a brand by implementing `MouseDriver`
 * and appending an entry to `MOUSE_DRIVERS`.
 */

import type { MouseDriver } from './types';
import type { MouseHidDevice, MouseHidFilter } from './webhid';
import { PulsarMouseClient } from './pulsar-device';
import { RazerMouseClient } from './razer-device';
import { RAZER_PRODUCT_IDS } from './razer-devices';

interface MouseDriverEntry {
	brand: string;
	supports(device: MouseHidDevice): boolean;
	create(device: MouseHidDevice): MouseDriver;
}

const MOUSE_DRIVERS: readonly MouseDriverEntry[] = [
	{
		brand: 'Razer',
		supports: (device) => RazerMouseClient.isSupported(device),
		create: (device) => new RazerMouseClient(device)
	},
	{
		brand: 'Pulsar',
		supports: (device) => PulsarMouseClient.isSupported(device),
		create: (device) => new PulsarMouseClient(device)
	}
];

/**
 * Filters for the WebHID chooser covering every registered mouse. Razer mice
 * surface through the shared Razer vendor filter the app already requests;
 * these cover the mouse-only vendor ids.
 */
export const MOUSE_HID_FILTERS: MouseHidFilter[] = [{ vendorId: 0x3710 }, { vendorId: 0x3554 }];

export function mouseSupportScore(device: MouseHidDevice): number {
	return MOUSE_DRIVERS.some((driver) => driver.supports(device)) ? 1 : 0;
}

export function createMouseClient(device: MouseHidDevice): MouseDriver | null {
	const entry = MOUSE_DRIVERS.find((driver) => driver.supports(device));
	return entry ? entry.create(device) : null;
}

export function deviceBrandOf(client: MouseDriver): string {
	for (const driver of MOUSE_DRIVERS) {
		if (driver.supports(client.device)) return driver.brand;
	}
	return 'Unknown';
}

/** Product ids the Razer registry knows, for diagnostics. */
export const supportedRazerProductIds = RAZER_PRODUCT_IDS;

export type { MouseDriver, MouseStatus, MouseLighting, MouseLightingMode, LiftOffDistance, MouseUiHints } from './types';
export type { MouseHidDevice, MouseHidFilter } from './webhid';
