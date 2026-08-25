import { unpackReport } from './report';

function hex2(n: number): string {
	return n.toString(16).padStart(2, '0');
}

function hexc(n: number): string {
	return '0x' + hex2(n);
}

function rgb(a: Uint8Array, off: number): string {
	return '#' + hex2(a[off]) + hex2(a[off + 1]) + hex2(a[off + 2]);
}

function frameDetail(a: Uint8Array, rowOff: number, startOff: number, stopOff: number): string {
	const row = a[rowOff];
	const s = a[startOff];
	const e = a[stopOff];
	const n = e >= s ? e - s + 1 : 0;
	return `row ${row} · cols ${s}–${e} · ${n} keys`;
}

function describeClassicEffect(a: Uint8Array): string {
	switch (a[0]) {
		case 0x00:
			return 'Off';
		case 0x01:
			return `Wave dir=${a[1] === 1 ? 'right' : 'left'}`;
		case 0x02:
			return `Reactive speed=${a[1]} ${rgb(a, 2)}`;
		case 0x03: {
			const mode = a[1];
			const m = mode === 1 ? 'single' : mode === 2 ? 'dual' : 'random';
			let s = `Breathing ${m}`;
			if (mode === 1) s += ` ${rgb(a, 2)}`;
			if (mode === 2) s += ` ${rgb(a, 2)} → ${rgb(a, 5)}`;
			return s;
		}
		case 0x04:
			return 'Spectrum';
		case 0x05:
			return 'Custom frame';
		case 0x06:
			return `Static ${rgb(a, 1)}`;
		case 0x19: {
			const mode = a[1];
			const m = mode === 1 ? 'single' : mode === 2 ? 'dual' : 'random';
			let s = `Starlight ${m} speed=${a[2]}`;
			if (mode === 1) s += ` ${rgb(a, 3)}`;
			if (mode === 2) s += ` ${rgb(a, 3)} → ${rgb(a, 6)}`;
			return s;
		}
		default:
			return `effect ${hexc(a[0])}`;
	}
}

function describeExtendedEffect(a: Uint8Array): string {
	switch (a[2]) {
		case 0x00:
			return 'Off';
		case 0x01:
			return `Static ${rgb(a, 6)}`;
		case 0x02: {
			const mode = a[3] || a[5];
			const m = mode === 1 ? 'single' : mode === 2 ? 'dual' : 'random';
			let s = `Breathing ${m}`;
			if (mode === 1) s += ` ${rgb(a, 6)}`;
			if (mode === 2) s += ` ${rgb(a, 6)} → ${rgb(a, 9)}`;
			return s;
		}
		case 0x03:
			return 'Spectrum';
		case 0x04:
			return `Wave dir=${a[3] === 1 ? 'right' : 'left'}`;
		case 0x05:
			return `Reactive speed=${a[4]} ${rgb(a, 6)}`;
		case 0x07: {
			const mode = a[5];
			const m = mode === 1 ? 'single' : mode === 2 ? 'dual' : 'random';
			let s = `Starlight ${m} speed=${a[4]}`;
			if (mode === 1) s += ` ${rgb(a, 6)}`;
			if (mode === 2) s += ` ${rgb(a, 6)} → ${rgb(a, 9)}`;
			return s;
		}
		case 0x08:
			return 'Custom frame';
		default:
			return `effect ${hexc(a[2])}`;
	}
}

export function describeReport(bytes: Uint8Array, direction: 'tx' | 'rx'): string {
	if (bytes.length < 8) return 'short report';
	const r = unpackReport(bytes);
	const cls = r.commandClass;
	const id = r.commandId;
	const tag = direction === 'rx' ? 'RSP' : 'SET';

	let what = '';
	let detail = '';

	switch (cls) {
		case 0x00:
			switch (id) {
				case 0x04:
					what = 'Set driver mode';
					detail = `mode=${r.args[0]}`;
					break;
				case 0x82:
					what = 'Get serial';
					if (direction === 'rx') {
						let s = '';
						for (let i = 0; i < 16 && i < r.dataSize; i++) {
							const c = r.args[i];
							if (c >= 32 && c <= 126) s += String.fromCharCode(c);
						}
						detail = `"${s.trim()}"`;
					}
					break;
				case 0x81:
					what = 'Get firmware';
					if (direction === 'rx') detail = `v${r.args[0]}.${r.args[1]}`;
					break;
				case 0x84:
					what = 'Get device mode';
					break;
				default:
					what = 'Device cmd';
			}
			break;

		case 0x03:
			switch (id) {
				case 0x0a:
					what = 'Effect';
					detail = describeClassicEffect(r.args);
					break;
				case 0x0b:
					what = 'Custom frame';
					detail = frameDetail(r.args, 1, 2, 3);
					break;
				case 0x00:
					what = 'Set LED state';
					break;
				case 0x01:
					what = 'Set LED RGB';
					detail = rgb(r.args, 2);
					break;
				case 0x03:
					what = 'Set brightness';
					detail = `${r.args[2]}`;
					break;
				case 0x83:
					what = 'Get brightness';
					if (direction === 'rx') detail = `${r.args[2]}`;
					break;
				case 0x80:
					what = 'Get LED state';
					break;
				case 0x81:
					what = 'Get LED RGB';
					if (direction === 'rx') detail = rgb(r.args, 2);
					break;
				default:
					what = 'LED cmd';
			}
			break;

		case 0x0f:
			switch (id) {
				case 0x02:
					what = 'Effect';
					detail = describeExtendedEffect(r.args);
					break;
				case 0x03:
					what = 'Custom frame';
					detail = frameDetail(r.args, 2, 3, 4);
					break;
				case 0x04:
					what = 'Set brightness';
					detail = `${r.args[2]}`;
					break;
				case 0x84:
					what = 'Get brightness';
					if (direction === 'rx') detail = `${r.args[2]}`;
					break;
				default:
					what = 'Ext cmd';
			}
			break;

		case 0x02:
			what = 'Cmd';
			break;

		default:
			what = 'Cmd';
	}

	return `${tag} ${hexc(cls)}/${hexc(id)} ${what}${detail ? ` · ${detail}` : ''}`;
}
