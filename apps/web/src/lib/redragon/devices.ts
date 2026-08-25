import type { LayoutKind } from '../razer/devices';

export type RedragonEffect = 'off' | 'static' | 'spectrum' | 'wave' | 'wheel' | 'breathing' | 'reactive' | 'starlight';

export interface RedragonDevice {
	pid: number;
	name: string;
	layout: LayoutKind;
	effects: RedragonEffect[];
	custom: false;
	image?: string;
}

export const REDRAGON_DEVICES: RedragonDevice[] = [
	{
		pid: 0x5004,
		name: 'Redragon K556 Devarajas',
		layout: 'full',
		effects: ['off', 'static', 'spectrum', 'wave', 'breathing', 'reactive', 'starlight'],
		custom: false,
		image: '/redragon.png'
	},
	{
		pid: 0x5004,
		name: 'Redragon K587 PRO Magic Wand',
		layout: 'full',
		effects: ['off', 'static', 'spectrum', 'wave', 'breathing', 'reactive', 'starlight'],
		custom: false,
		image: '/redragon.png'
	},
	{
		pid: 0x5004,
		name: 'Redragon Surara K582',
		layout: 'full',
		effects: ['off', 'static', 'spectrum', 'wave', 'breathing', 'reactive', 'starlight'],
		custom: false,
		image: '/redragon.png'
	},
	{
		pid: 0x5004,
		name: 'Redragon K589 Shrapnel',
		layout: 'full',
		effects: ['off', 'static', 'spectrum', 'wave', 'breathing', 'reactive', 'starlight'],
		custom: false,
		image: '/redragon.png'
	},
	{
		pid: 0x5104,
		name: 'Redragon K552 Kumara',
		layout: 'tkl',
		effects: ['off', 'static', 'spectrum', 'wave', 'breathing', 'reactive', 'starlight'],
		custom: false,
		image: '/redragon.png'
	},
	{
		pid: 0x5204,
		name: 'Redragon K550 Yama',
		layout: 'full',
		effects: ['off', 'static', 'spectrum', 'wave', 'breathing', 'reactive', 'starlight'],
		custom: false,
		image: '/redragon.png'
	},
	{
		pid: 0x672e,
		name: 'Redragon K512',
		layout: 'full',
		effects: ['off', 'static', 'spectrum', 'wave', 'breathing', 'reactive', 'starlight'],
		custom: false,
		image: '/redragon.png'
	}
];

export function getRedragonKeyboard(pid: number): RedragonDevice | undefined {
	return REDRAGON_DEVICES.find((d) => d.pid === pid);
}

export const REDRAGON_SUPPORTED_PIDS: number[] = [...new Set(REDRAGON_DEVICES.map((d) => d.pid))];
