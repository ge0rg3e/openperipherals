/**
 * Shared mouse data model, modelled on the OpenMouse protocol project's
 * driver contract so per-brand drivers stay interchangeable.
 */

import type { MouseHidDevice } from './webhid';

export type LiftOffDistance = 'Low' | 'Medium' | 'High';

/** Optional UI policy a driver attaches to its status. */
export interface MouseUiHints {
	/** Stable driver id, e.g. "pulsar" or "razer". */
	family?: string;
	/** Hide poll rates not listed in supportedPollingRates. */
	hideUnsupportedPollingRates?: boolean;
	/** Show the polling rate but refuse to stage a change. */
	pollingReadOnly?: boolean;
	/** Overrides the polling-rate footnote. */
	pollingNote?: string;
	/** Hide the Low LOD option. */
	hideLodLow?: boolean;
	/** Disable LOD controls while gamingSurfaceMode is "Off". */
	lodRequiresSurface?: boolean;
	/** Hide Motion Sync / angle snap / ripple controls entirely. */
	hideProcessingCard?: boolean;
	hideMotionSync?: boolean;
	hideAngleSnapping?: boolean;
	hideRippleControl?: boolean;
	/**
	 * Multi-stage DPI editor. Set together with `dpiStages` on the status so
	 * the shared stage list renders.
	 */
	dpiStageEditor?: {
		maxStages: number;
		countEditable?: boolean;
		minDpi: number;
		maxDpi: number;
		stepDpi: number;
	};
}

/**
 * A lighting zone a driver can write effects to. `mode` may come from the
 * driver's own last-write cache when the hardware cannot report the effect
 * back (`writeOnly`).
 */
export interface MouseLighting {
	zone: string;
	modes: readonly MouseLightingMode[];
	mode: MouseLightingMode | null;
	color: string | null;
	color2: string | null;
	colorModes: readonly MouseLightingMode[];
	dualColorModes: readonly MouseLightingMode[];
	reactiveModes: readonly MouseLightingMode[];
	speeds: readonly number[];
	speed: number | null;
	/** Optional brightness percentage for zones that expose it. */
	brightness?: number | null;
	/** Brightness percentages the device accepts. */
	brightnessLevels?: readonly number[];
	writeOnly?: boolean;
}

export type MouseLightingMode =
	| 'Off'
	| 'Static'
	| 'Cycling'
	| 'Wave'
	| 'Spectrum'
	| 'Reactive'
	| 'Breathing random'
	| 'Breathing single'
	| 'Breathing dual';

export interface MouseStatus {
	brand: string;
	name: string;
	ui?: MouseUiHints;
	batteryPercent: number | null;
	batteryState: 'Charging' | 'Charging slowly' | 'Almost full' | 'Full' | 'Discharging' | 'Unknown';
	dpi: number;
	dpiY?: number;
	/** On-device DPI stages, where a driver exposes them. */
	dpiStages?: number[];
	/** Active DPI stage index into `dpiStages` (0-based). */
	activeDpiStage?: number;
	pollingRateHz: number;
	supportedPollingRates?: number[];
	activeProfile: number | null;
	connectionType?: 'Wired' | 'Wireless';
	connectionDetail?: string;
	signalStrength?: number | null;
	motionSync?: boolean | null;
	sleepTimeout?: number | null;
	angleSnapping?: boolean | null;
	rippleControl?: boolean | null;
	debounceMs?: number | null;
	liftOffDistance: LiftOffDistance | null;
	supportedLiftOffDistances?: LiftOffDistance[];
	gamingSurfaceMode?: 'On' | 'Off' | 'Auto' | null;
	lighting?: MouseLighting;
	lightingZones?: MouseLighting[];
	firmware: string[];
}

/**
 * The contract every mouse driver implements. Setters beyond the core five are
 * optional — the UI probes for them at runtime so one panel serves every brand.
 */
export interface MouseDriver {
	readonly device: MouseHidDevice;
	open(): Promise<void>;
	close(): Promise<void>;
	readStatus(): Promise<MouseStatus>;
	getDpiOptions(): number[];
	setDpi(dpi: number, dpiY?: number): Promise<number>;
	setPollingRate(pollingRateHz: number): Promise<number>;
	setLiftOffDistance(value: LiftOffDistance): Promise<LiftOffDistance>;
	setMotionSync?(enabled: boolean): Promise<boolean>;
	setAngleSnapping?(enabled: boolean): Promise<boolean>;
	setRippleControl?(enabled: boolean): Promise<boolean>;
	setSleepTimeout?(seconds: number): Promise<number>;
	setDebounceTime?(milliseconds: number): Promise<number>;
	getSleepOptions?(): readonly number[];
	getDebounceMaxMs?(): number;
	setLighting?(lighting: MouseLighting): Promise<MouseLighting>;
}
