// Hardware RGB-matrix layout used for per-key "custom" lighting.
//
// The grid is the canonical 6-row × 22-column Chroma matrix shared by the
// driver and OpenRazer (macro column M1-M5/M6 at col 0, Razer logo at row 0
// col 20). Key positions come from OpenRazer's KEY_MAPPING in
// daemon/openrazer_daemon/keyboard.py; the codes are the same logical codes
// used by ./layout.ts. Boards with a smaller geometry simply drop the rows /
// columns they don't have (e.g. TKL boards drop the numpad columns 18-21).
//
// The Huntsman Elite is a 9-row matrix (MATRIX_DIMS [9,22]): rows 0-5 hold the
// keys at the same positions as CELLS (it has no macro column and no logo), the
// function row carries media keys at cols 18-21, and rows 6-8 are the
// wrist-rest underglow lightbar. Positions follow RazerGenie's
// razerhuntsmanelite.json (issue z3ntu/RazerGenie#67).

export interface HwCell {
	code: string;
	row: number;
	col: number;
}

const CELLS: HwCell[] = [
	// Row 0 - function row (M6 at col 0, logo at col 20)
	{ code: 'ESC', row: 0, col: 1 },
	{ code: 'F1', row: 0, col: 3 },
	{ code: 'F2', row: 0, col: 4 },
	{ code: 'F3', row: 0, col: 5 },
	{ code: 'F4', row: 0, col: 6 },
	{ code: 'F5', row: 0, col: 7 },
	{ code: 'F6', row: 0, col: 8 },
	{ code: 'F7', row: 0, col: 9 },
	{ code: 'F8', row: 0, col: 10 },
	{ code: 'F9', row: 0, col: 11 },
	{ code: 'F10', row: 0, col: 12 },
	{ code: 'F11', row: 0, col: 13 },
	{ code: 'F12', row: 0, col: 14 },
	{ code: 'PSCR', row: 0, col: 15 },
	{ code: 'SLCK', row: 0, col: 16 },
	{ code: 'PAUSE', row: 0, col: 17 },
	// Row 1
	{ code: 'GRAVE', row: 1, col: 1 },
	{ code: '1', row: 1, col: 2 },
	{ code: '2', row: 1, col: 3 },
	{ code: '3', row: 1, col: 4 },
	{ code: '4', row: 1, col: 5 },
	{ code: '5', row: 1, col: 6 },
	{ code: '6', row: 1, col: 7 },
	{ code: '7', row: 1, col: 8 },
	{ code: '8', row: 1, col: 9 },
	{ code: '9', row: 1, col: 10 },
	{ code: '0', row: 1, col: 11 },
	{ code: 'MINUS', row: 1, col: 12 },
	{ code: 'EQUAL', row: 1, col: 13 },
	{ code: 'BACKSPACE', row: 1, col: 14 },
	{ code: 'INS', row: 1, col: 15 },
	{ code: 'HOME', row: 1, col: 16 },
	{ code: 'PGUP', row: 1, col: 17 },
	{ code: 'NUMLOCK', row: 1, col: 18 },
	{ code: 'NPSLASH', row: 1, col: 19 },
	{ code: 'NPASTERISK', row: 1, col: 20 },
	{ code: 'NPMINUS', row: 1, col: 21 },
	// Row 2
	{ code: 'TAB', row: 2, col: 1 },
	{ code: 'Q', row: 2, col: 2 },
	{ code: 'W', row: 2, col: 3 },
	{ code: 'E', row: 2, col: 4 },
	{ code: 'R', row: 2, col: 5 },
	{ code: 'T', row: 2, col: 6 },
	{ code: 'Y', row: 2, col: 7 },
	{ code: 'U', row: 2, col: 8 },
	{ code: 'I', row: 2, col: 9 },
	{ code: 'O', row: 2, col: 10 },
	{ code: 'P', row: 2, col: 11 },
	{ code: 'LBRACKET', row: 2, col: 12 },
	{ code: 'RBRACKET', row: 2, col: 13 },
	{ code: 'DEL', row: 2, col: 15 },
	{ code: 'END', row: 2, col: 16 },
	{ code: 'PGDN', row: 2, col: 17 },
	{ code: 'NP7', row: 2, col: 18 },
	{ code: 'NP8', row: 2, col: 19 },
	{ code: 'NP9', row: 2, col: 20 },
	{ code: 'NPPLUS', row: 2, col: 21 },
	// Row 3
	{ code: 'CAPS', row: 3, col: 1 },
	{ code: 'A', row: 3, col: 2 },
	{ code: 'S', row: 3, col: 3 },
	{ code: 'D', row: 3, col: 4 },
	{ code: 'F', row: 3, col: 5 },
	{ code: 'G', row: 3, col: 6 },
	{ code: 'H', row: 3, col: 7 },
	{ code: 'J', row: 3, col: 8 },
	{ code: 'K', row: 3, col: 9 },
	{ code: 'L', row: 3, col: 10 },
	{ code: 'SEMICOLON', row: 3, col: 11 },
	{ code: 'APOSTROPHE', row: 3, col: 12 },
	{ code: 'ENTER', row: 3, col: 14 },
	{ code: 'NP4', row: 3, col: 18 },
	{ code: 'NP5', row: 3, col: 19 },
	{ code: 'NP6', row: 3, col: 20 },
	// Row 4
	{ code: 'LSHIFT', row: 4, col: 1 },
	{ code: 'BACKSLASH', row: 4, col: 2 },
	{ code: 'Z', row: 4, col: 3 },
	{ code: 'X', row: 4, col: 4 },
	{ code: 'C', row: 4, col: 5 },
	{ code: 'V', row: 4, col: 6 },
	{ code: 'B', row: 4, col: 7 },
	{ code: 'N', row: 4, col: 8 },
	{ code: 'M', row: 4, col: 9 },
	{ code: 'COMMA', row: 4, col: 10 },
	{ code: 'PERIOD', row: 4, col: 11 },
	{ code: 'SLASH', row: 4, col: 12 },
	{ code: 'RSHIFT', row: 4, col: 14 },
	{ code: 'UP', row: 4, col: 16 },
	{ code: 'NP1', row: 4, col: 18 },
	{ code: 'NP2', row: 4, col: 19 },
	{ code: 'NP3', row: 4, col: 20 },
	{ code: 'NPENTER', row: 4, col: 21 },
	// Row 5
	{ code: 'LCTRL', row: 5, col: 1 },
	{ code: 'LSUPER', row: 5, col: 2 },
	{ code: 'LALT', row: 5, col: 3 },
	{ code: 'SPACE', row: 5, col: 7 },
	{ code: 'RALT', row: 5, col: 11 },
	{ code: 'MENU', row: 5, col: 13 },
	{ code: 'RCTRL', row: 5, col: 14 },
	{ code: 'LEFT', row: 5, col: 15 },
	{ code: 'DOWN', row: 5, col: 16 },
	{ code: 'RIGHT', row: 5, col: 17 },
	{ code: 'NP0', row: 5, col: 19 },
	{ code: 'NPDOT', row: 5, col: 20 }
];

// Media keys on the Huntsman Elite's function row (cols 18-21, where the Razer
// logo sits on other full-size boards).
const ELITE_MEDIA_CELLS: HwCell[] = [
	{ code: 'MEDIA_PREV', row: 0, col: 18 },
	{ code: 'MEDIA_PLAY', row: 0, col: 19 },
	{ code: 'MEDIA_NEXT', row: 0, col: 20 },
	{ code: 'MUTE', row: 0, col: 21 }
];

// Wrist-rest underglow zones (matrix rows 6-8: cols 0-18, 0-18, 0-19).
const ELITE_LIGHTBAR_CELLS: HwCell[] = (() => {
	const cells: HwCell[] = [];
	for (let r = 0; r < 3; r++) {
		const row = 6 + r;
		const cols = r === 2 ? 20 : 19;
		for (let c = 0; c < cols; c++) {
			cells.push({ code: `LIGHTBAR${r}_${c}`, row, col: c });
		}
	}
	return cells;
})();

/** Full cell map for the Huntsman Elite's 9×22 matrix. */
export const ELITE_CELLS: HwCell[] = [...CELLS, ...ELITE_MEDIA_CELLS, ...ELITE_LIGHTBAR_CELLS];

/** Build a rows×cols grid of key codes, null for cells with no LED. */
export function hwGrid(rows: number, cols: number, cells: HwCell[] = CELLS): (string | null)[][] {
	const grid: (string | null)[][] = [];
	for (let r = 0; r < rows; r++) {
		const rowCells: (string | null)[] = [];
		for (let c = 0; c < cols; c++) rowCells.push(null);
		grid.push(rowCells);
	}
	for (const cell of cells) {
		if (cell.row < rows && cell.col < cols) grid[cell.row][cell.col] = cell.code;
	}
	return grid;
}

/** Whether a board can host per-key custom frames. */
export function customSupported(rows?: number, cols?: number): boolean {
	if (cols === undefined) return false;
	// Standard 6-row boards (full/TKL/some compact) with 17-22 columns.
	if (rows === 6 && cols >= 17 && cols <= 22) return true;
	// The Huntsman Elite's 9-row matrix (6 key rows + 3 lightbar rows).
	if (rows === 9 && cols === 22) return true;
	return false;
}
