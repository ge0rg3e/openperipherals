import type { EffectParams } from '../controller';
import { getRedragonKeyboard, type RedragonDevice } from './devices';
import { RedragonTransport, type RedragonHandle } from './transport';
import { buildEffectReports } from './commands';
import type { EvReport } from './report';

export interface RedragonDeviceInfo {
	vendor: 'redragon';
	pid: number;
	name: string;
	serial?: string;
	firmware?: string;
	rdk?: RedragonDevice;
}

async function sendAll(transport: RedragonTransport, reports: EvReport[]): Promise<void> {
	for (const report of reports) {
		try {
			await transport.send(report);
		} catch {
			// The device can briefly reject a write right after a reconnect or
			// while it's busy; retry once after a short pause before giving up.
			await new Promise((r) => setTimeout(r, 25));
			await transport.send(report);
		}
		await new Promise((r) => setTimeout(r, 15));
	}
}

export class RedragonKeyboardController {
	private transport: RedragonTransport | null = null;
	private infoValue: RedragonDeviceInfo | null = null;
	private lastEffect: EffectParams | null = null;
	private currentBrightness = 128;

	get connected(): boolean {
		return !!this.transport?.connected;
	}

	get info(): RedragonDeviceInfo | null {
		return this.infoValue;
	}

	get device(): RedragonDevice | undefined {
		return this.infoValue?.rdk;
	}

	async connect(devices?: Parameters<RedragonTransport['open']>[0]): Promise<RedragonDeviceInfo> {
		await this.disconnect();
		const t = new RedragonTransport();
		const handle: RedragonHandle = await t.open(devices);
		this.transport = t;
		this.infoValue = {
			vendor: 'redragon',
			pid: handle.pid,
			name: handle.name,
			rdk: getRedragonKeyboard(handle.pid)
		};
		return this.infoValue;
	}

	async disconnect(): Promise<void> {
		if (this.transport) {
			await this.transport.close();
			this.transport = null;
			this.infoValue = null;
			this.lastEffect = null;
		}
	}

	async apply(params: EffectParams): Promise<void> {
		if (!this.transport) throw new Error('Redragon transport is not open.');
		this.lastEffect = params;
		const reports = buildEffectReports(params, this.currentBrightness);
		await sendAll(this.transport, reports.reports);
	}

	async setBrightness(value: number): Promise<void> {
		// Brightness lives inside the mode packet; re-send the current effect
		// (if any) with the updated value.
		this.currentBrightness = value;
		if (this.lastEffect && this.transport) {
			const reports = buildEffectReports(this.lastEffect, value);
			await sendAll(this.transport, reports.reports);
		}
	}

	async getBrightness(): Promise<number | null> {
		return this.currentBrightness;
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