import type { EffectParams } from '../controller';
import { getLogitechKeyboard, type LogitechDevice } from './devices';
import { LogitechTransport, type LogitechHandle } from './transport';
import { buildCustomFrames, buildEffectPackets } from './commands';
import type { LogiReport } from './report';

export interface LogitechDeviceInfo {
	vendor: 'logitech';
	pid: number;
	name: string;
	serial?: string;
	firmware?: string;
	lkb?: LogitechDevice;
}

async function sendAll(transport: LogitechTransport, reports: LogiReport[]): Promise<void> {
	for (const report of reports) {
		try {
			await transport.send(report);
		} catch {
			// The device can briefly reject a write right after a reconnect or while
			// it's busy; retry once after a short pause before giving up.
			await new Promise((r) => setTimeout(r, 25));
			await transport.send(report);
		}
		// Some firmware drops reports sent back-to-back; a short gap keeps them apart.
		await new Promise((r) => setTimeout(r, 5));
	}
}

export class LogitechKeyboardController {
	private transport: LogitechTransport | null = null;
	private infoValue: LogitechDeviceInfo | null = null;

	get connected(): boolean {
		return !!this.transport?.connected;
	}

	get info(): LogitechDeviceInfo | null {
		return this.infoValue;
	}

	get device(): LogitechDevice | undefined {
		return this.infoValue?.lkb;
	}

	async connect(devices?: Parameters<LogitechTransport['open']>[0]): Promise<LogitechDeviceInfo> {
		await this.disconnect();
		const t = new LogitechTransport();
		const handle: LogitechHandle = await t.open(devices);
		this.transport = t;
		this.infoValue = {
			vendor: 'logitech',
			pid: handle.pid,
			name: handle.name,
			serial: handle.serial,
			firmware: undefined,
			lkb: getLogitechKeyboard(handle.pid)
		};
		return this.infoValue;
	}

	async disconnect(): Promise<void> {
		if (this.transport) {
			await this.transport.close();
			this.transport = null;
			this.infoValue = null;
		}
	}

	async apply(params: EffectParams): Promise<void> {
		const device = this.device;
		if (!device) throw new Error('No Logitech device connected.');
		if (!this.transport) throw new Error('Logitech transport is not open.');

		if (params.kind === 'custom' && params.custom) {
			const frames = buildCustomFrames(device, params.custom);
			await sendAll(this.transport, frames.init);
			await sendAll(this.transport, frames.packets);
			await sendAll(this.transport, frames.commit);
			return;
		}

		const effect = buildEffectPackets(device, params);
		await sendAll(this.transport, effect.init);
		await sendAll(this.transport, effect.packets);
		await sendAll(this.transport, effect.commit);
	}

	async setBrightness(_value: number): Promise<void> {
		// brightness lives inside the G915 mode frame; standalone writes are a no-op.
		return;
	}

	async getBrightness(): Promise<number | null> {
		return null;
	}

	async setGameMode(_enabled: boolean): Promise<void> {
		return;
	}

	async getGameMode(): Promise<boolean | null> {
		return null;
	}

	async setMacroLeds(_enabled: boolean): Promise<void> {
		return;
	}

	async getMacroLeds(): Promise<boolean | null> {
		return null;
	}

	async getBattery(): Promise<{ level: number; charging: boolean } | null> {
		return null;
	}
}
