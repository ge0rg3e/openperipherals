import { keyboardMatrixKeys, type MatrixKey } from './keyboard/layout';

const FONT: Record<string, string[]> = {
	A: ['01110', '10001', '11111', '10001', '10001'],
	B: ['11110', '10001', '11110', '10001', '11110'],
	C: ['01110', '10001', '10000', '10001', '01110'],
	D: ['11100', '10010', '10010', '10010', '11100'],
	E: ['11111', '10000', '11110', '10000', '11111'],
	F: ['11111', '10000', '11110', '10000', '10000'],
	G: ['01110', '10001', '10111', '10001', '01111'],
	H: ['10001', '10001', '11111', '10001', '10001'],
	I: ['11111', '00100', '00100', '00100', '11111'],
	J: ['00111', '00010', '00010', '10010', '01100'],
	K: ['10001', '10010', '11100', '10010', '10001'],
	L: ['10000', '10000', '10000', '10000', '11111'],
	M: ['10001', '11011', '10101', '10001', '10001'],
	N: ['10001', '11001', '10101', '10011', '10001'],
	O: ['01110', '10001', '10001', '10001', '01110'],
	P: ['11110', '10001', '11110', '10000', '10000'],
	Q: ['01110', '10001', '10101', '10010', '01101'],
	R: ['11110', '10001', '11110', '10010', '10001'],
	S: ['01111', '10000', '01110', '00001', '11110'],
	T: ['11111', '00100', '00100', '00100', '00100'],
	U: ['10001', '10001', '10001', '10001', '01110'],
	V: ['10001', '10001', '10001', '01010', '00100'],
	W: ['10001', '10001', '10101', '11011', '10001'],
	X: ['10001', '01010', '00100', '01010', '10001'],
	Y: ['10001', '01010', '00100', '00100', '00100'],
	Z: ['11111', '00010', '00100', '01000', '11111'],
	0: ['01110', '10011', '10101', '11001', '01110'],
	1: ['00100', '01100', '00100', '00100', '01110'],
	2: ['01110', '10001', '00110', '01000', '11111'],
	3: ['11110', '00001', '01110', '00001', '11110'],
	4: ['10010', '10010', '11111', '00010', '00010'],
	5: ['11111', '10000', '11110', '00001', '11110'],
	6: ['01110', '10000', '11110', '10001', '01110'],
	7: ['11111', '00001', '00010', '00100', '00100'],
	8: ['01110', '10001', '01110', '10001', '01110'],
	9: ['01110', '10001', '01111', '00001', '01110'],
	' ': ['00000', '00000', '00000', '00000', '00000'],
	'!': ['00100', '00100', '00100', '00000', '00100'],
	'?': ['01110', '10001', '00110', '00000', '00100'],
	'.': ['00000', '00000', '00000', '00110', '00110'],
	':': ['00000', '00100', '00000', '00100', '00000'],
	'+': ['00000', '00100', '01110', '00100', '00000'],
	'-': ['00000', '00000', '01110', '00000', '00000'],
	'=': ['00000', '01110', '00000', '01110', '00000'],
	'*': ['00000', '10101', '01110', '10101', '00000'],
	'<': ['00010', '00100', '01000', '00100', '00010'],
	'>': ['01000', '00100', '00010', '00100', '01000'],
	'/': ['00001', '00010', '00100', '01000', '10000'],
	'\\': ['10000', '01000', '00100', '00010', '00001']
};

function glyph(ch: string): string[] | undefined {
	if (ch in FONT) return FONT[ch];
	const l = ch.toLowerCase();
	return l in FONT ? FONT[l] : undefined;
}

export interface DrawTextResult {
	/** per-key colour map (only keys touched are set; rest left absent). */
	colors: Record<string, string>;
	/** number of glyphs actually rendered. */
	rendered: number;
}

export function drawTextToKeys(text: string, colorHex: string, keys: MatrixKey[] = keyboardMatrixKeys()): DrawTextResult {
	const colors: Record<string, string> = {};
	let rendered = 0;

	// group keys by row
	const byRow = new Map<number, MatrixKey[]>();
	for (const k of keys) {
		if (!byRow.has(k.row)) byRow.set(k.row, []);
		byRow.get(k.row)!.push(k);
	}
	const ordered = [...byRow.entries()].sort((a, b) => a[0] - b[0]);
	if (!ordered.length) return { colors, rendered: 0 };

	// We'll draw the text as a 5-row-tall strip starting at the first key row
	// that holds letters (usually row 1). Map each of the 5 glyph rows to the
	// first 5 key rows available.
	const usable = ordered;
	const rowFor = (gi: number): number => usable[Math.min(gi, usable.length - 1)][0];

	// single line of text: compute needed col per glyph and track cursor in matrix
	// columns. Because each key has a unique (row, col), placing pixel (gr, gc) at
	// key(row=rowFor(gr), col=cursor+gc) works.
	const GLYPH_W = 5;
	const SPACE = 1;
	let cursor = 0;
	const strip = (ch: string) => {
		const g = glyph(ch);
		if (!g) return;
		if (cursor + GLYPH_W > 22) {
			cursor = 0; // wrap to next line (rarely used for short text)
		}
		for (let gr = 0; gr < 5; gr++) {
			const rnum = rowFor(gr);
			const rowKeys = byRow.get(rnum) ?? [];
			for (let gc = 0; gc < GLYPH_W; gc++) {
				if (g[gr][gc] !== '1') continue;
				const col = cursor + gc;
				const key = nearestKeyAtCol(rowKeys, col);
				if (key) colors[key.code] = colorHex;
			}
		}
		cursor += GLYPH_W + SPACE;
		rendered++;
	};

	for (const ch of Array.from(text)) strip(ch);
	return { colors, rendered };
}

function nearestKeyAtCol(list: MatrixKey[], col: number): MatrixKey | undefined {
	let best: MatrixKey | undefined;
	for (const k of list) {
		if (k.col <= col && col < k.col + (k.width ?? 1)) return k;
		if (!best || Math.abs(k.col - col) < Math.abs(best.col - col)) best = k;
	}
	return best;
}
