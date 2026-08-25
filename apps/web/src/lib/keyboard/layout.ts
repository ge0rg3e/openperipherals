export interface MatrixKey {
	row: number;
	col: number;
	code: string; // unique key code
	label: string;
	/** column span (width). 1 standard. */
	width?: number;
}

interface RowKey {
	code: string;
	label?: string;
	span?: number;
}

interface RowsSpec {
	label: string;
	keys: RowKey[];
}

const baseLayout: readonly RowsSpec[] = [
	{
		label: 'Function',
		keys: [
			{ code: 'ESC', label: 'Esc' },
			{ code: 'F1' },
			{ code: 'F2' },
			{ code: 'F3' },
			{ code: 'F4' },
			{ code: 'F5' },
			{ code: 'F6' },
			{ code: 'F7' },
			{ code: 'F8' },
			{ code: 'F9' },
			{ code: 'F10' },
			{ code: 'F11' },
			{ code: 'F12' },
			{ code: 'PSCR', label: 'Prt', span: 2 },
			{ code: 'SLCK', label: 'Scr' },
			{ code: 'PAUSE', label: 'Pau', span: 2 }
		]
	},
	{
		label: 'Row 1',
		keys: [
			{ code: 'GRAVE', label: '~' },
			{ code: '1' },
			{ code: '2' },
			{ code: '3' },
			{ code: '4' },
			{ code: '5' },
			{ code: '6' },
			{ code: '7' },
			{ code: '8' },
			{ code: '9' },
			{ code: '0' },
			{ code: 'MINUS', label: '-' },
			{ code: 'EQUAL', label: '=' },
			{ code: 'BACKSPACE', label: 'BkSp', span: 2 },
			{ code: 'INS', label: 'Ins' },
			{ code: 'HOME', label: 'Home' },
			{ code: 'PGUP', label: 'PgUp' },
			{ code: 'NUMLOCK', label: 'Num' },
			{ code: 'NPSLASH', label: '/' },
			{ code: 'NPASTERISK', label: '*' },
			{ code: 'NPMINUS', label: '-' }
		]
	},
	{
		label: 'QWERTY',
		keys: [
			{ code: 'TAB', label: 'Tab', span: 2 },
			{ code: 'Q' },
			{ code: 'W' },
			{ code: 'E' },
			{ code: 'R' },
			{ code: 'T' },
			{ code: 'Y' },
			{ code: 'U' },
			{ code: 'I' },
			{ code: 'O' },
			{ code: 'P' },
			{ code: 'LBRACKET', label: '[' },
			{ code: 'RBRACKET', label: ']' },
			{ code: 'BACKSLASH', label: '\\', span: 2 },
			{ code: 'DEL', label: 'Del' },
			{ code: 'END', label: 'End' },
			{ code: 'PGDN', label: 'PgDn' },
			{ code: 'NP7', label: '7' },
			{ code: 'NP8', label: '8' },
			{ code: 'NP9', label: '9' },
			{ code: 'NPPLUS', label: '+' }
		]
	},
	{
		label: 'Caps',
		keys: [
			{ code: 'CAPS', label: 'Caps', span: 2 },
			{ code: 'A' },
			{ code: 'S' },
			{ code: 'D' },
			{ code: 'F' },
			{ code: 'G' },
			{ code: 'H' },
			{ code: 'J' },
			{ code: 'K' },
			{ code: 'L' },
			{ code: 'SEMICOLON', label: ';' },
			{ code: 'APOSTROPHE', label: "'" },
			{ code: 'ENTER', label: 'Enter', span: 2 },
			{ code: 'NP4', label: '4' },
			{ code: 'NP5', label: '5' },
			{ code: 'NP6', label: '6' }
		]
	},
	{
		label: 'Shift',
		keys: [
			{ code: 'LSHIFT', label: 'Shift', span: 3 },
			{ code: 'Z' },
			{ code: 'X' },
			{ code: 'C' },
			{ code: 'V' },
			{ code: 'B' },
			{ code: 'N' },
			{ code: 'M' },
			{ code: 'COMMA', label: ',' },
			{ code: 'PERIOD', label: '.' },
			{ code: 'SLASH', label: '/' },
			{ code: 'RSHIFT', label: 'Shift', span: 3 },
			{ code: 'UP', label: '↑' },
			{ code: 'NP1', label: '1' },
			{ code: 'NP2', label: '2' },
			{ code: 'NP3', label: '3' },
			{ code: 'NPENTER', label: 'Enter', span: 2 }
		]
	},
	{
		label: 'Space',
		keys: [
			{ code: 'LCTRL', label: 'Ctrl', span: 2 },
			{ code: 'LSUPER', label: 'Win', span: 2 },
			{ code: 'LALT', label: 'Alt', span: 2 },
			{ code: 'SPACE', label: '', span: 6 },
			{ code: 'RALT', label: 'Alt' },
			{ code: 'MENU', label: 'Menu' },
			{ code: 'RCTRL', label: 'Ctrl', span: 2 },
			{ code: 'LEFT', label: '←' },
			{ code: 'DOWN', label: '↓' },
			{ code: 'RIGHT', label: '→' },
			{ code: 'NP0', label: '0', span: 2 },
			{ code: 'NPDOT', label: '.' }
		]
	}
];

function expand(): MatrixKey[] {
	const out: MatrixKey[] = [];
	baseLayout.forEach((row, rowIndex) => {
		let col = 0;
		for (const k of row.keys) {
			const width = k.span ?? 1;
			out.push({ row: rowIndex, col, code: k.code, label: k.label ?? k.code, width });
			col += width;
		}
	});
	return out;
}

const MATRIX = expand();
export function keyboardMatrixKeys(): MatrixKey[] {
	return MATRIX;
}

/** Caps the 3D preview renders but the hardware matrix has no LED cell for. */
export const previewOnlyCodes = ['FN'] as const;

export const keyByCode = new Map<string, MatrixKey>();
for (const k of MATRIX) keyByCode.set(k.code, k);
