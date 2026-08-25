import { getKeyboard, type KeyboardDevice } from './razer/devices';
import { Transport, WebHidTransport, webhidSupported } from './razer/transport';
import { REPORT_LEN, ARG_BASE, crc, type CommandReport } from './razer/report';
import * as C from './razer/commands';
import { LED } from './razer/constants';
import { hwGrid, ELITE_CELLS } from './keyboard/matrix';

export type ConnectionMode = 'detect';

export interface DeviceInfo {
	pid: number;
	name: string;
	serial?: string;
	firmware?: string;
	kbd?: KeyboardDevice;
}

export type EffectKind = 'off' | 'static' | 'wave' | 'wheel' | 'spectrum' | 'reactive' | 'breathing' | 'starlight' | 'custom';

export interface EffectParams {
	kind: EffectKind;
	color?: string; // '#rrggbb'
	color2?: string;
	speed?: number; // 1..4
	direction?: 'left' | 'right';
	mode?: 'single' | 'dual' | 'random';
	/** per-key colours for the custom effect (key code → '#rrggbb'). */
	custom?: Record<string, string>;
}

export class KeyboardController {
	private transport: Transport | null = null;
	private infoValue: DeviceInfo | null = null;

	get connected(): boolean {
		return !!this.transport?.connected;
	}
	get info(): DeviceInfo | null {
		return this.infoValue;
	}

	async connect(granted?: Parameters<WebHidTransport['open']>[0]): Promise<DeviceInfo> {
		if (this.transport) await this.disconnect();
		const t = new WebHidTransport();
		await t.open(granted);
		this.transport = t;
		const h = t.handle;
		if (!h) throw new Error('No device handle.');
		const spec = h.pid ? getKeyboard(h.pid) : undefined;
		this.infoValue = {
			pid: h.pid,
			name: spec?.name ?? h.name,
			kbd: spec,
			serial: await this.readSerial().catch(() => undefined),
			firmware: await this.readFirmware().catch(() => undefined)
		};
		return this.infoValue;
	}

	async disconnect(): Promise<void> {
		if (this.transport) await this.transport.close();
		this.transport = null;
		this.infoValue = null;
	}

	get device(): KeyboardDevice | undefined {
		return this.infoValue?.kbd;
	}

	private get dtoId(): number {
		return this.infoValue?.kbd?.transactionId ?? 0xff;
	}

	private get isExt(): boolean {
		return this.infoValue?.kbd?.style === 'extended';
	}

	private async send(report: CommandReport, tid?: number): Promise<Uint8Array | null> {
		if (!this.transport) throw new Error('Not connected.');
		report.transactionId = tid ?? this.dtoId;
		const resp = await this.transport.send(report);
		if (resp && resp.length >= REPORT_LEN && resp[88] !== crc(resp)) {
			// warn once via logger
		}
		return resp;
	}

	private async readSerial(): Promise<string> {
		const r = await this.send(C.getSerial());
		return r ? ascii(r, 8, 22) : '';
	}
	private async readFirmware(): Promise<string> {
		const r = await this.send(C.getFirmware());
		return r ? ascii(r, 8, 2) : '';
	}

	private parseColor(hex?: string): C.RGB {
		return hexToRgb(hex ?? '#ffffff') ?? { r: 0xff, g: 0xff, b: 0xff };
	}

	async apply(params: EffectParams): Promise<void> {
		if (params.kind === 'custom') {
			await this.applyCustomFrame(params.custom ?? {});
			return;
		}
		const vs = 1; // VARSTORE - the effect is written to the keyboard's on-device storage,
		// so it persists after the app closes and across replug/reboot.
		const led = LED.BACKLIGHT;
		const report = this.buildEffect(params, vs, led);
		report.transactionId = this.dtoId;
		await this.send(report);
	}
	/** Write a per-key colour frame and switch the keyboard to custom mode.
	 *  Custom frames are host-rendered (NOSTORE), so they don't survive a
	 *  power cycle - the browser must re-send them, exactly like OpenRazer. */
	private async applyCustomFrame(colors: Record<string, string>): Promise<void> {
		if (!this.transport) throw new Error('Not connected.');
		const kbd = this.device;
		const rows = kbd?.matrixRows ?? 6;
		const cols = kbd?.matrixCols ?? 22;
		// The Huntsman Elite's 9-row matrix adds media keys and a lightbar; use
		// its cell map so those zones light up too.
		const grid = hwGrid(rows, cols, rows === 9 && cols === 22 ? ELITE_CELLS : undefined);
		let pinned = false;
		for (let r = 0; r < rows; r++) {
			const rowGrid = grid[r];
			const bytes: number[] = [];
			for (let c = 0; c < cols; c++) {
				const code = rowGrid[c];
				const rgb = code ? hexToRgb(colors[code] ?? '') : null;
				bytes.push(rgb?.r ?? 0, rgb?.g ?? 0, rgb?.b ?? 0);
			}
			// A frame packet holds up to 22 columns; wider rows go in chunks.
			for (let start = 0; start < cols; start += 22) {
				const stop = Math.min(start + 21, cols - 1);
				const slice = bytes.slice(start * 3, (stop + 1) * 3);
				const rep = this.isExt ? C.extendedMatrixSetCustomFrame(r, start, stop, slice) : C.classicMatrixSetCustomFrame(r, start, stop, slice);
				rep.transactionId = this.dtoId;
				// Row frames are write-only. The first one goes through the probing
				// send path (which pins the working HID interface); the rest skip
				// the feature-read round trip so painting stays responsive.
				if (!pinned) {
					pinned = true;
					await this.send(rep);
				} else {
					this.transport.sendBurst(rep);
				}
			}
		}
		const activate = this.isExt ? C.extendedEffectCustomFrame() : C.classicEffectCustomFrame(0);
		activate.transactionId = this.dtoId;
		await this.send(activate);
	}
	private buildEffect(p: EffectParams, vs: number, led: number): CommandReport {
		switch (p.kind) {
			case 'off':
				return this.isExt ? C.extendedEffectNone(vs, led) : C.classicEffectNone();
			case 'static':
				return this.isExt ? C.extendedEffectStatic(vs, led, this.parseColor(p.color)) : C.classicEffectStatic(this.parseColor(p.color));
			case 'spectrum':
				return this.isExt ? C.extendedEffectSpectrum(vs, led) : C.classicEffectSpectrum();
			case 'wave': {
				// WAVE_RIGHT is byte 1 on every board. For "left", classic and modern
				// extended matrices expect 2. OpenRazer registers some old extended
				// boards (Huntsman family, BlackWidow Elite) as WAVE_DIRS=(0,1), but
				// those devices clamp 2 down to 1 and ignore 0, so left can never
				// work there; the firmware actually wants 2 as well. Override with
				// waveLeft when a board genuinely needs 0.
				const dir = p.direction === 'left' ? (this.device?.waveLeft ?? 2) : 1;
				return this.isExt ? C.extendedEffectWave(vs, led, dir) : C.classicEffectWave(dir);
			}
			case 'wheel':
				// Hardware colour-wheel effect, exposed only by the BlackWidow V4
				// family (extended matrix effect 0x0a). Every other board gets the
				// closest equivalent (spectrum) rather than an unsupported command.
				if (!this.device?.wheel) {
					return this.isExt ? C.extendedEffectSpectrum(vs, led) : C.classicEffectSpectrum();
				}
				return C.extendedEffectWheel(vs, led, p.direction === 'left' ? 2 : 1);
			case 'reactive':
				return this.isExt ? C.extendedEffectReactive(vs, led, p.speed ?? 2, this.parseColor(p.color)) : C.classicEffectReactive(p.speed ?? 2, this.parseColor(p.color));
			case 'breathing':
				return this.buildBreathing(p, vs, led);
			case 'starlight':
				return this.buildStarlight(p, vs, led);
			default:
				throw new Error(`Unsupported effect: ${p.kind}`);
		}
	}

	private buildBreathing(p: EffectParams, vs: number, led: number): CommandReport {
		if (p.mode === 'dual') {
			return this.isExt
				? C.extendedEffectBreathingDual(vs, led, this.parseColor(p.color), this.parseColor(p.color2))
				: C.classicEffectBreathingDual(this.parseColor(p.color), this.parseColor(p.color2));
		}
		if (p.mode === 'random') {
			return this.isExt ? C.extendedEffectBreathingRandom(vs, led) : C.classicEffectBreathingRandom();
		}
		return this.isExt ? C.extendedEffectBreathingSingle(vs, led, this.parseColor(p.color)) : C.classicEffectBreathingSingle(this.parseColor(p.color));
	}

	private buildStarlight(p: EffectParams, vs: number, led: number): CommandReport {
		if (p.mode === 'random') {
			return this.isExt ? C.extendedEffectStarlightRandom(vs, led, p.speed ?? 3) : C.classicEffectStarlightRandom(p.speed ?? 3);
		}
		if (p.mode === 'dual') {
			return this.isExt
				? C.extendedEffectStarlightDual(vs, led, p.speed ?? 3, this.parseColor(p.color), this.parseColor(p.color2))
				: C.classicEffectStarlightDual(p.speed ?? 3, this.parseColor(p.color), this.parseColor(p.color2));
		}
		return this.isExt ? C.extendedEffectStarlightSingle(vs, led, p.speed ?? 3, this.parseColor(p.color)) : C.classicEffectStarlightSingle(p.speed ?? 3, this.parseColor(p.color));
	}

	async setBrightness(value: number): Promise<void> {
		const vs = 1;
		const led = LED.BACKLIGHT;
		const rep = this.isExt ? C.extendedBrightness(vs, led, value) : C.setLedBrightness(vs, led, value);
		rep.transactionId = this.dtoId;
		await this.send(rep);
	}

	async getBrightness(): Promise<number | null> {
		const vs = 1;
		const led = LED.BACKLIGHT;
		const rep = this.isExt ? C.getExtendedBrightness(vs, led) : C.getLedBrightness(vs, led);
		rep.transactionId = this.dtoId;
		const resp = await this.send(rep);
		return resp && resp.length >= REPORT_LEN ? resp[ARG_BASE + 2] : null;
	}

	/** Game mode disables the Windows/Super key; state is stored on the device. */
	async setGameMode(enabled: boolean): Promise<void> {
		const rep = C.setLedState(1, LED.GAME, enabled ? 1 : 0);
		rep.transactionId = this.dtoId;
		await this.send(rep);
	}

	async getGameMode(): Promise<boolean | null> {
		const rep = C.getLedState(1, LED.GAME);
		rep.transactionId = this.dtoId;
		const resp = await this.send(rep);
		return resp && resp.length >= REPORT_LEN ? resp[ARG_BASE + 2] === 1 : null;
	}

	/** Backlight for the dedicated macro keys (M1-M5); state is stored on the device. */
	async setMacroLeds(enabled: boolean): Promise<void> {
		const rep = C.setLedState(1, LED.MACRO, enabled ? 1 : 0);
		rep.transactionId = this.dtoId;
		await this.send(rep);
	}

	async getMacroLeds(): Promise<boolean | null> {
		const rep = C.getLedState(1, LED.MACRO);
		rep.transactionId = this.dtoId;
		const resp = await this.send(rep);
		return resp && resp.length >= REPORT_LEN ? resp[ARG_BASE + 2] === 1 : null;
	}

	/** Battery level (0-100) and charge state for wireless boards. */
	async getBattery(): Promise<{ level: number; charging: boolean } | null> {
		const kbd = this.infoValue?.kbd;
		if (!kbd?.battery) return null;
		const tid = kbd.batteryTid ?? this.dtoId;
		const level = await this.send(C.getBatteryLevel(), tid);
		if (!level || level.length < REPORT_LEN) return null;
		const status = await this.send(C.getChargingStatus(), tid);
		const raw = level[ARG_BASE + 1];
		const charging = !!(status && status.length >= REPORT_LEN && status[ARG_BASE + 1] === 1);
		return { level: Math.round((raw / 255) * 100), charging };
	}
}

function hexToRgb(hex: string): C.RGB | null {
	const m = /^#?([0-9a-f]{2})([0-9a-f]{2})([0-9a-f]{2})$/i.exec(hex.trim());
	if (!m) return null;
	return { r: parseInt(m[1], 16), g: parseInt(m[2], 16), b: parseInt(m[3], 16) };
}

function ascii(bytes: Uint8Array, offset: number, len: number): string {
	let s = '';
	const end = Math.min(offset + len, bytes.length);
	for (let i = offset; i < end; i++) {
		const c = bytes[i];
		if (c >= 32 && c <= 126) s += String.fromCharCode(c);
	}
	return s.trim();
}

export { webhidSupported };
