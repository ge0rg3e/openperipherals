/**
 * Razer per-PID capability registry.
 *
 * Razer's protocol is not self-describing: nothing on the wire says which
 * commands a mouse answers, so capability lives in this table keyed on the
 * exact product id. A wrong transaction id means silence (readStatus throws on
 * the firmware read and nothing is written); a wrong ceiling is caught by the
 * read-back every setter performs; a wrong capability flag suppresses a command
 * rather than inventing one.
 */

import { RAZER_TRANSACTION_ID_1F, RAZER_TRANSACTION_ID_3F, RAZER_TRANSACTION_ID_FF } from './razer-codec';

export interface RazerProduct {
	model: string;
	wireless: boolean;
	pollingRates: readonly number[];
	maxDpi: number;
	transactionId: number;
	hasBattery: boolean;
	vendorControlInterface?: boolean;
	dpiStorageByte?: number;
	highRatePolling: boolean;
	liftOff: boolean;
}

export const RATES_1K: readonly number[] = [125, 500, 1000];
export const RATES_8K: readonly number[] = [125, 500, 1000, 2000, 4000, 8000];

/**
 * Transaction ids audited per product id — they follow neither transport group,
 * connection type nor family name, so they are kept in one flat block.
 */
const TRANSACTION_3F: readonly number[] = [
	0x0050, 0x0059, 0x005a, 0x005c, 0x0060, 0x0064, 0x0065, 0x006f, 0x0070,
	0x0072, 0x0073, 0x007c, 0x007d, 0x0084, 0x008c,
	0x007a, 0x007b, // Viper Ultimate — observed answering 0x3f, not OpenRazer's 0xff
	0x006e, 0x0071, 0x0098 // DeathAdder Essential
];

const TRANSACTION_1F: readonly number[] = [
	0x0062, 0x006c, 0x0077, 0x0080, 0x0085, 0x0086, 0x0088, 0x008d, 0x008f,
	0x0090, 0x0094, 0x0096, 0x0099, 0x009a, 0x009c, 0x009e, 0x009f, 0x00a1,
	0x00a4, 0x00a5, 0x00a6, 0x00a7, 0x00a8, 0x00aa, 0x00ab, 0x00af, 0x00b0, 0x00b2, 0x00b3,
	0x00b4, 0x00b6, 0x00b7, 0x00b8, 0x00b9, 0x00be, 0x00bf, 0x00c0, 0x00c1,
	0x00c2, 0x00c3, 0x00c4, 0x00c5, 0x00c7, 0x00c8, 0x00cb, 0x00cc, 0x00cd,
	0x00d0, 0x00d1, 0x00d3, 0x00d4, 0x00d6, 0x00d7, 0x00de, 0x00df
];

/** 0xff is the fallback for unaudited products, not a safe default. */
function transactionIdFor(productId: number): number {
	if (TRANSACTION_3F.includes(productId)) return RAZER_TRANSACTION_ID_3F;
	if (TRANSACTION_1F.includes(productId)) return RAZER_TRANSACTION_ID_1F;
	return RAZER_TRANSACTION_ID_FF;
}

// Sensor ceilings by generation, from published specifications.
const DPI_CHROMA = 16_000;
const DPI_FOCUS = 20_000;
const DPI_FOCUS_PRO = 30_000;
const DPI_FOCUS_PRO_35K = 35_000;

interface ProductDefaults extends Omit<RazerProduct, 'model' | 'transactionId'> {}

/** Chroma-era wired mice; which interface carries control varies by revision. */
const STANDARD: ProductDefaults = {
	wireless: false,
	pollingRates: RATES_1K,
	maxDpi: DPI_CHROMA,
	hasBattery: false,
	vendorControlInterface: true,
	highRatePolling: false,
	liftOff: false
};

const STANDARD_WIRELESS: ProductDefaults = { ...STANDARD, wireless: true, hasBattery: true };

/** Modern HyperSpeed generation, wired half (wireless mice on a cable). */
const MODERN_WIRED: ProductDefaults = {
	wireless: false,
	pollingRates: RATES_1K,
	maxDpi: DPI_FOCUS,
	hasBattery: true,
	highRatePolling: false,
	liftOff: false
};

/** Stock 1000 Hz HyperSpeed receiver. */
const MODERN_RECEIVER: ProductDefaults = { ...MODERN_WIRED, wireless: true, highRatePolling: true };

/** Older receivers that predate the extended polling command. */
const LEGACY_RECEIVER: ProductDefaults = { ...MODERN_RECEIVER, highRatePolling: false };

/** Receivers shipping as HyperPolling dongles, reaching 8000 Hz. */
const HYPERPOLLING_RECEIVER: ProductDefaults = {
	...MODERN_RECEIVER,
	pollingRates: RATES_8K,
	maxDpi: DPI_FOCUS_PRO_35K
};

type Definition = [number, Omit<RazerProduct, 'transactionId'>];

const PRODUCT_DEFINITIONS: readonly Definition[] = [
	// ---- Viper V2/V3 Pro (verified generation) ----
	[0x00a5, { model: 'Viper V2 Pro', wireless: false, pollingRates: RATES_1K, maxDpi: DPI_FOCUS_PRO, hasBattery: true, highRatePolling: false, liftOff: true }],
	[0x00a6, { model: 'Viper V2 Pro', wireless: true, pollingRates: RATES_1K, maxDpi: DPI_FOCUS_PRO, hasBattery: true, highRatePolling: true, liftOff: true }],
	[0x00c0, { model: 'Viper V3 Pro', wireless: false, pollingRates: RATES_1K, maxDpi: DPI_FOCUS_PRO_35K, hasBattery: true, highRatePolling: false, liftOff: true }],
	[0x00c1, { model: 'Viper V3 Pro', wireless: true, pollingRates: RATES_8K, maxDpi: DPI_FOCUS_PRO_35K, hasBattery: true, highRatePolling: true, liftOff: true }],

	// ---- DeathAdder Essential family ----
	[0x006e, { model: 'DeathAdder Essential', ...STANDARD, maxDpi: 6400 }],
	[0x0071, { model: 'DeathAdder Essential White Edition', ...STANDARD, maxDpi: 6400 }],
	[0x0098, { model: 'DeathAdder Essential (2021)', ...STANDARD, maxDpi: 6400 }],

	// ---- standard wired ----
	[0x0015, { model: 'Naga', ...STANDARD, maxDpi: 8200 }],
	[0x0020, { model: 'Abyssus 1800', ...STANDARD, maxDpi: 1800 }],
	[0x002e, { model: 'Naga 2012', ...STANDARD, maxDpi: 8200 }],
	[0x0034, { model: 'Taipan', ...STANDARD, maxDpi: 8200 }],
	[0x0037, { model: 'DeathAdder 2013', ...STANDARD, maxDpi: 6400 }],
	[0x0040, { model: 'Naga 2014', ...STANDARD, maxDpi: 8200 }],
	[0x0043, { model: 'DeathAdder Chroma', ...STANDARD }],
	[0x0046, { model: 'Mamba Tournament Edition', ...STANDARD }],
	[0x0053, { model: 'Naga Chroma', ...STANDARD }],
	[0x005c, { model: 'DeathAdder Elite', ...STANDARD }],
	[0x0060, { model: 'Lancehead Tournament Edition', ...STANDARD }],
	[0x0064, { model: 'Basilisk', ...STANDARD }],
	[0x0065, { model: 'Basilisk Essential', ...STANDARD, maxDpi: 6400 }],
	[0x0067, { model: 'Naga Trinity', ...STANDARD }],
	[0x006c, { model: 'Mamba Elite', ...STANDARD }],
	[0x0084, { model: 'DeathAdder V2', ...STANDARD, maxDpi: DPI_FOCUS, dpiStorageByte: 0x00 }],
	[0x0085, { model: 'Basilisk V2', ...STANDARD, maxDpi: DPI_FOCUS }],
	[0x008c, { model: 'DeathAdder V2 Mini', ...STANDARD, maxDpi: 8500 }],
	[0x0091, { model: 'Viper 8KHz', ...STANDARD, maxDpi: DPI_FOCUS, pollingRates: RATES_8K, highRatePolling: true }],
	[0x00a1, { model: 'DeathAdder V2 Lite', ...STANDARD, maxDpi: 8500 }],
	[0x00b2, { model: 'DeathAdder V3', ...STANDARD, maxDpi: DPI_FOCUS_PRO, pollingRates: RATES_8K, highRatePolling: true }],

	// ---- index3 / receivers ----
	[0x0096, { model: 'Naga X', ...STANDARD, maxDpi: 18_000 }],
	[0x0099, { model: 'Basilisk V3', ...STANDARD, maxDpi: 26_000 }],
	[0x00cb, { model: 'Basilisk V3 35K', ...STANDARD, maxDpi: DPI_FOCUS_PRO_35K }],
	[0x0062, { model: 'Atheris', ...STANDARD_WIRELESS, maxDpi: 7200 }],
	[0x0094, { model: 'Orochi V2', ...STANDARD_WIRELESS, maxDpi: 18_000 }],
	[0x007a, { model: 'Viper Ultimate (Wired)', ...MODERN_WIRED }],
	[0x007b, { model: 'Viper Ultimate', ...MODERN_RECEIVER }],
	[0x007c, { model: 'DeathAdder V2 Pro (Wired)', ...MODERN_WIRED }],
	[0x007d, { model: 'DeathAdder V2 Pro', ...MODERN_RECEIVER }],
	[0x009e, { model: 'Viper Mini Signature Edition (Wired)', ...MODERN_WIRED, maxDpi: DPI_FOCUS_PRO }],
	[0x009f, { model: 'Viper Mini Signature Edition', ...MODERN_RECEIVER, maxDpi: DPI_FOCUS_PRO, pollingRates: RATES_8K }],
	// Verified: stock HyperSpeed receiver answers only the legacy divisor-of-1000 command.
	[0x00b8, { model: 'Viper V3 HyperSpeed', ...MODERN_RECEIVER, highRatePolling: false, liftOff: true, maxDpi: DPI_FOCUS_PRO }],

	// ---- new-receiver ----
	[0x006f, { model: 'Lancehead Wireless', ...LEGACY_RECEIVER }],
	[0x0070, { model: 'Lancehead Wireless (Wired)', ...MODERN_WIRED }],
	[0x0072, { model: 'Mamba Wireless', ...LEGACY_RECEIVER }],
	[0x0077, { model: 'Pro Click', ...LEGACY_RECEIVER }],
	[0x0080, { model: 'Pro Click (Wired)', ...MODERN_WIRED }],
	[0x0083, { model: 'Basilisk X HyperSpeed', ...LEGACY_RECEIVER }],
	[0x0086, { model: 'Basilisk Ultimate (Wired)', ...MODERN_WIRED }],
	[0x0088, { model: 'Basilisk Ultimate', ...LEGACY_RECEIVER }],
	[0x008f, { model: 'Naga Pro (Wired)', ...MODERN_WIRED }],
	[0x0090, { model: 'Naga Pro', ...LEGACY_RECEIVER }],
	[0x009a, { model: 'Pro Click Mini', ...LEGACY_RECEIVER, maxDpi: 12_000 }],
	[0x009c, { model: 'DeathAdder V2 X HyperSpeed', ...LEGACY_RECEIVER, maxDpi: 14_000 }],
	[0x00a7, { model: 'Naga V2 Pro (Wired)', ...MODERN_WIRED, maxDpi: DPI_FOCUS_PRO }],
	[0x00a8, { model: 'Naga V2 Pro', ...MODERN_RECEIVER, maxDpi: DPI_FOCUS_PRO }],
	[0x00aa, { model: 'Basilisk V3 Pro (Wired)', ...MODERN_WIRED, maxDpi: DPI_FOCUS_PRO }],
	[0x00ab, { model: 'Basilisk V3 Pro', ...MODERN_RECEIVER, maxDpi: DPI_FOCUS_PRO }],
	[0x00af, { model: 'Cobra Pro (Wired)', ...MODERN_WIRED, maxDpi: DPI_FOCUS_PRO }],
	[0x00b0, { model: 'Cobra Pro', ...MODERN_RECEIVER, maxDpi: DPI_FOCUS_PRO }],
	[0x00b4, { model: 'Naga V2 HyperSpeed', ...LEGACY_RECEIVER, maxDpi: DPI_FOCUS_PRO }],
	[0x00b6, { model: 'DeathAdder V3 Pro (Wired)', ...MODERN_WIRED, maxDpi: DPI_FOCUS_PRO }],
	// Measured: extended command reads back but the link stays at 1000 Hz on the
	// stock receiver, so this row keeps the legacy encoding.
	[0x00b7, { model: 'DeathAdder V3 Pro', ...MODERN_RECEIVER, highRatePolling: false, maxDpi: DPI_FOCUS_PRO }],
	[0x00b9, { model: 'Basilisk V3 X HyperSpeed', ...LEGACY_RECEIVER, maxDpi: 18_000 }],
	[0x00c2, { model: 'DeathAdder V3 Pro (Wired)', ...MODERN_WIRED, maxDpi: DPI_FOCUS_PRO }],
	[0x00c3, { model: 'DeathAdder V3 Pro', ...MODERN_RECEIVER, maxDpi: DPI_FOCUS_PRO }],
	[0x00c4, { model: 'DeathAdder V3 HyperSpeed (Wired)', ...MODERN_WIRED, maxDpi: DPI_FOCUS_PRO }],
	[0x00c5, { model: 'DeathAdder V3 HyperSpeed', ...LEGACY_RECEIVER, maxDpi: DPI_FOCUS_PRO }],
	[0x00cc, { model: 'Basilisk V3 Pro 35K (Wired)', ...MODERN_WIRED, maxDpi: DPI_FOCUS_PRO_35K }],
	[0x00cd, { model: 'Basilisk V3 Pro 35K', ...HYPERPOLLING_RECEIVER }]
];

/** Every product, with its audited transaction id attached. */
export const RAZER_PRODUCTS: ReadonlyMap<number, RazerProduct> = new Map(
	PRODUCT_DEFINITIONS.map(([productId, product]) => [productId, { ...product, transactionId: transactionIdFor(productId) }])
);

export const RAZER_PRODUCT_IDS: readonly number[] = [...RAZER_PRODUCTS.keys()];
