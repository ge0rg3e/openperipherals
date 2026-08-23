/**
 * Demo devices - in-memory stand-ins that satisfy the same contracts the real
 * hardware drivers implement, so the full workspace UI can be explored without
 * a WebHID-capable browser or any physical device attached.
 */
import type { EffectParams, DeviceInfo } from './controller';
import { getKeyboard } from './razer/devices';
import { logger } from './razer/logger';
import type { MouseDriver, MouseLighting, MouseStatus } from './mouse/types';
import type { MouseHidDevice } from './mouse/webhid';

// --- Demo keyboard (Razer-shaped) ---

/** Public API mirrors KeyboardController closely enough for the shared UI. */
export class DemoKeyboardController {
	readonly demo = true;
	private infoValue: DeviceInfo | null = null;
	private brightness = 180;
	private gameMode = false;
	private macroLeds = false;

	get connected(): boolean {
		return this.infoValue !== null;
	}

	get info(): DeviceInfo | null {
		return this.infoValue;
	}

	get device() {
		return this.infoValue?.kbd;
	}

	async connect(): Promise<DeviceInfo> {
		const kbd = getKeyboard(0x0226); // Huntsman Elite spec: extended 9x22 matrix + custom frames
		if (!kbd) throw new Error('Demo keyboard spec is missing.');
		this.infoValue = { pid: kbd.pid, name: kbd.name, serial: 'DX0000DEMO', firmware: 'v1.0.0', kbd };
		logger.info('Demo keyboard connected (simulated device).');
		return this.infoValue;
	}

	async disconnect(): Promise<void> {
		this.infoValue = null;
		logger.info('Demo keyboard disconnected.');
	}

	async apply(params: EffectParams): Promise<void> {
		this.assertConnected();
		logger.tx(`demo effect: ${describeEffect(params)}`);
	}

	async setBrightness(value: number): Promise<void> {
		this.assertConnected();
		this.brightness = value;
		logger.tx(`demo brightness: ${Math.round((value / 255) * 100)}%`);
	}

	async getBrightness(): Promise<number | null> {
		this.assertConnected();
		return this.brightness;
	}

	async setGameMode(enabled: boolean): Promise<void> {
		this.assertConnected();
		this.gameMode = enabled;
		logger.tx(`demo game mode: ${enabled ? 'on' : 'off'}`);
	}

	async getGameMode(): Promise<boolean | null> {
		this.assertConnected();
		return this.gameMode;
	}

	async setMacroLeds(enabled: boolean): Promise<void> {
		this.assertConnected();
		this.macroLeds = enabled;
		logger.tx(`demo macro leds: ${enabled ? 'on' : 'off'}`);
	}

	async getMacroLeds(): Promise<boolean | null> {
		this.assertConnected();
		return this.macroLeds;
	}

	async getBattery(): Promise<{ level: number; charging: boolean }> {
		this.assertConnected();
		return { level: 84, charging: false };
	}

	private assertConnected(): void {
		if (!this.infoValue) throw new Error('Demo keyboard is not connected.');
	}
}

function describeEffect(p: EffectParams): string {
	const parts: string[] = [p.kind];
	if (p.mode) parts.push(p.mode);
	if (p.color) parts.push(p.color);
	if (p.color2) parts.push(p.color2);
	if (p.speed) parts.push(`speed ${p.speed}`);
	if (p.direction) parts.push(p.direction);
	return parts.join(' ');
}

// --- Demo mouse (Razer Cobra Pro-shaped) ---

const DEMO_MOUSE_VID = 0x1532;
const DEMO_MOUSE_PID = 0x00b0;

function fakeMouseDevice(): MouseHidDevice {
	return {
		vendorId: DEMO_MOUSE_VID,
		productId: DEMO_MOUSE_PID,
		productName: 'Razer Cobra Pro',
		serialNumber: 'DM0000DEMO',
		opened: true,
		open: async () => undefined,
		close: async () => undefined,
		sendReport: async () => undefined,
		sendFeatureReport: async () => undefined,
		receiveFeatureReport: async () => new DataView(new ArrayBuffer(90)),
		collections: [],
		addEventListener: () => undefined,
		removeEventListener: () => undefined
	};
}

export class DemoMouseDriver implements MouseDriver {
	readonly device: MouseHidDevice = fakeMouseDevice();

	private state: MouseStatus = {
		brand: 'Razer',
		name: 'Razer Cobra Pro',
		ui: {
			family: 'demo',
			dpiStageEditor: { maxStages: 5, countEditable: true, minDpi: 100, maxDpi: 30000, stepDpi: 50 }
		},
		batteryPercent: 87,
		batteryState: 'Discharging',
		dpi: 1600,
		dpiStages: [400, 800, 1600, 3200],
		activeDpiStage: 2,
		pollingRateHz: 1000,
		supportedPollingRates: [125, 500, 1000, 2000, 4000, 8000],
		activeProfile: null,
		connectionType: 'Wireless',
		signalStrength: 92,
		motionSync: true,
		sleepTimeout: 900,
		angleSnapping: false,
		rippleControl: false,
		debounceMs: 10,
		liftOffDistance: 'Low',
		supportedLiftOffDistances: ['Low', 'Medium', 'High'],
		gamingSurfaceMode: 'On',
		lightingZones: [
			{
				zone: 'Scroll wheel',
				modes: ['Off', 'Static', 'Spectrum', 'Breathing single', 'Reactive'],
				mode: 'Static',
				color: '#00ff88',
				color2: '#0088ff',
				colorModes: ['Static', 'Breathing single', 'Reactive'],
				dualColorModes: [],
				reactiveModes: ['Reactive'],
				speeds: [1, 2, 3],
				speed: 2,
				brightness: 100,
				brightnessLevels: [0, 25, 50, 75, 100]
			},
			{
				zone: 'Logo',
				modes: ['Off', 'Static', 'Spectrum', 'Wave'],
				mode: 'Static',
				color: '#0088ff',
				color2: '#ff00ff',
				colorModes: ['Static'],
				dualColorModes: ['Wave'],
				reactiveModes: [],
				speeds: [1, 2, 3],
				speed: 2
			}
		] satisfies MouseLighting[],
		firmware: ['v1.04']
	};

	async open(): Promise<void> {
		logger.info('Demo mouse connected (simulated device).');
	}

	async close(): Promise<void> {
		logger.info('Demo mouse disconnected.');
	}

	async readStatus(): Promise<MouseStatus> {
		return structuredClone(this.state);
	}

	getDpiOptions(): number[] {
		return [400, 800, 1600, 3200, 6400, 16000, 30000];
	}

	async setDpi(dpi: number): Promise<number> {
		const stages = [...(this.state.dpiStages ?? [])];
		stages[this.state.activeDpiStage ?? 0] = dpi;
		this.state = { ...this.state, dpi, dpiStages: stages };
		logger.tx(`demo dpi: ${dpi}`);
		return dpi;
	}

	async setActiveDpiStage(stage: number): Promise<number> {
		const stages = this.state.dpiStages ?? [];
		const index = Math.max(0, Math.min(stage, stages.length - 1));
		this.state = { ...this.state, activeDpiStage: index, dpi: stages[index] ?? this.state.dpi };
		logger.tx(`demo dpi stage: ${index + 1}`);
		return index;
	}

	async setDpiStageCount(count: number): Promise<number[]> {
		const stages = [...(this.state.dpiStages ?? [])].slice(0, count);
		while (stages.length < count) stages.push(stages[stages.length - 1] ?? 800);
		this.state = { ...this.state, dpiStages: stages };
		return stages;
	}

	async setPollingRate(pollingRateHz: number): Promise<number> {
		this.state = { ...this.state, pollingRateHz };
		logger.tx(`demo polling rate: ${pollingRateHz} Hz`);
		return pollingRateHz;
	}

	async setLiftOffDistance(value: 'Low' | 'Medium' | 'High'): Promise<'Low' | 'Medium' | 'High'> {
		this.state = { ...this.state, liftOffDistance: value };
		logger.tx(`demo lift-off distance: ${value}`);
		return value;
	}

	async setMotionSync(enabled: boolean): Promise<boolean> {
		this.state = { ...this.state, motionSync: enabled };
		logger.tx(`demo motion sync: ${enabled ? 'on' : 'off'}`);
		return enabled;
	}

	async setAngleSnapping(enabled: boolean): Promise<boolean> {
		this.state = { ...this.state, angleSnapping: enabled };
		logger.tx(`demo angle snapping: ${enabled ? 'on' : 'off'}`);
		return enabled;
	}

	async setRippleControl(enabled: boolean): Promise<boolean> {
		this.state = { ...this.state, rippleControl: enabled };
		logger.tx(`demo ripple control: ${enabled ? 'on' : 'off'}`);
		return enabled;
	}

	async setSleepTimeout(seconds: number): Promise<number> {
		this.state = { ...this.state, sleepTimeout: seconds };
		logger.tx(`demo sleep timeout: ${seconds}s`);
		return seconds;
	}

	async setDebounceTime(milliseconds: number): Promise<number> {
		this.state = { ...this.state, debounceMs: milliseconds };
		logger.tx(`demo debounce: ${milliseconds} ms`);
		return milliseconds;
	}

	getSleepOptions(): readonly number[] {
		return [0, 60, 180, 300, 600, 900];
	}

	getDebounceMaxMs(): number {
		return 40;
	}

	async setLighting(lighting: MouseLighting): Promise<MouseLighting> {
		const zones = (this.state.lightingZones ?? []).map((z) => (z.zone === lighting.zone ? lighting : z));
		this.state = { ...this.state, lightingZones: zones };
		logger.tx(`demo lighting: ${lighting.zone} -> ${lighting.mode ?? 'off'}`);
		return lighting;
	}
}
