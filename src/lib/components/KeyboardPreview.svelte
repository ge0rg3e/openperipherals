<script lang="ts">
	import { onMount } from 'svelte';
	import { browser } from '$app/environment';
	import { keyboardMatrixKeys, type MatrixKey } from '$lib/keyboard/layout';
	import { effectColorOf, type PreviewState } from '$lib/preview';
	import type { EffectKind, EffectParams } from '$lib/controller';
	import type { LayoutKind } from '$lib/razer/devices';
	import qmkCodes from '$lib/keyboard-3d/config/qmk_codes.json';
	import LAYOUTS from '$lib/keyboard-3d/config/layouts/layouts';
	import KEYMAPS from '$lib/keyboard-3d/config/keymaps/keymaps';

	let {
		preview = null as EffectKind | null,
		layout = 'full' as LayoutKind,
		effectParams = {} as EffectParams,
		brightness = 255 as number,
		custom = {} as Record<string, string>,
		onKeyClick = null as ((code: string) => void) | null
	}: {
		preview?: EffectKind | null;
		layout?: LayoutKind;
		effectParams?: EffectParams;
		brightness?: number;
		custom?: Record<string, string>;
		onKeyClick?: ((code: string) => void) | null;
	} = $props();

	const allKeys = keyboardMatrixKeys();
	const keyByCode = new Map(allKeys.map((k) => [k.code, k]));
	const clock: PreviewState = $state({
		minCol: Math.min(...allKeys.map((k) => k.col)),
		maxCol: Math.max(...allKeys.map((k) => k.col)),
		time: 0
	});

	// keys physically pressed on the real board, shown in real time
	const pressed: Record<string, number> = $state({});

	// app LayoutKind -> keyboard-3d layout id (100% / TKL / 65% / 60%)
	const LAYOUT_IDS = { full: 100, tkl: 80, small: 65, mini: 60 } as const;

	// map browser KeyboardEvent.code -> matrix key code
	const CODE_MAP: Record<string, string> = {
		Escape: 'ESC',
		Backquote: 'GRAVE',
		Minus: 'MINUS',
		Equal: 'EQUAL',
		Backspace: 'BACKSPACE',
		Tab: 'TAB',
		BracketLeft: 'LBRACKET',
		BracketRight: 'RBRACKET',
		Backslash: 'BACKSLASH',
		Semicolon: 'SEMICOLON',
		Quote: 'APOSTROPHE',
		Enter: 'ENTER',
		ShiftLeft: 'LSHIFT',
		ShiftRight: 'RSHIFT',
		Comma: 'COMMA',
		Period: 'PERIOD',
		Slash: 'SLASH',
		ControlLeft: 'LCTRL',
		ControlRight: 'RCTRL',
		AltLeft: 'LALT',
		AltRight: 'RALT',
		MetaLeft: 'LSUPER',
		Space: 'SPACE',
		CapsLock: 'CAPS',
		NumLock: 'NUMLOCK',
		ArrowUp: 'UP',
		ArrowDown: 'DOWN',
		ArrowLeft: 'LEFT',
		ArrowRight: 'RIGHT',
		Home: 'HOME',
		End: 'END',
		PageUp: 'PGUP',
		PageDown: 'PGDN',
		Delete: 'DEL',
		Insert: 'INS',
		PrintScreen: 'PSCR',
		ScrollLock: 'SLCK',
		Pause: 'PAUSE',
		ContextMenu: 'MENU',
		Numpad0: 'NP0',
		Numpad1: 'NP1',
		Numpad2: 'NP2',
		Numpad3: 'NP3',
		Numpad4: 'NP4',
		Numpad5: 'NP5',
		Numpad6: 'NP6',
		Numpad7: 'NP7',
		Numpad8: 'NP8',
		Numpad9: 'NP9',
		NumpadAdd: 'NPPLUS',
		NumpadSubtract: 'NPMINUS',
		NumpadMultiply: 'NPASTERISK',
		NumpadDivide: 'NPSLASH',
		NumpadDecimal: 'NPDOT',
		NumpadEnter: 'NPENTER'
	};

	function codeToMatrix(code: string): string | null {
		if (/^Key[A-Z]$/.test(code)) return code.slice(3);
		if (/^Digit[0-9]$/.test(code)) return code.slice(5);
		if (/^F([1-9]|1[0-2])$/.test(code)) return code;
		return CODE_MAP[code] ?? null;
	}

	// qmk cap code ("KC_ESC") -> app matrix code ("ESC")
	const qmkToMatrix: Record<string, string> = {};
	for (const [browserCode, qmk] of Object.entries(qmkCodes)) {
		const m = codeToMatrix(browserCode);
		if (m) qmkToMatrix[qmk] = m;
	}

	// Caps with no matrix entry of their own (right super, Fn, media keys...)
	// still light up: derive a pseudo key position from their spot in the
	// layout so spatial effects (wave/wheel) sweep across them too.
	const syntheticKeys = $derived.by(() => {
		const id = LAYOUT_IDS[layout] ?? 100;
		const map = new Map<string, MatrixKey>();
		const dims = LAYOUTS[id].layouts['LAYOUT'].layout;
		KEYMAPS[id].layers[0].forEach((qmk, i) => {
			if (!qmk || qmkToMatrix[qmk]) return;
			const d = dims[i];
			if (!d || map.has(qmk)) return;
			map.set(qmk, { code: '', label: '', row: Math.floor(d.y), col: Math.round(d.x) } as MatrixKey);
		});
		return map;
	});

	// scale a css color by the current brightness (0..255)
	function dim(css: string): string {
		const k = Math.max(0, Math.min(255, brightness)) / 255;
		if (k >= 1) return css;
		const [r, g, b] = rgbOf(css);
		const h = (v: number) => (Math.round(v * k) & 255).toString(16).padStart(2, '0');
		return `#${h(r)}${h(g)}${h(b)}`;
	}
	// extract rgb triple from a css color (hex, rgb(r g b), or rgb(r,g,b))
	function rgbOf(css: string): [number, number, number] {
		let m = /^#?([0-9a-f]{2})([0-9a-f]{2})([0-9a-f]{2})$/i.exec(css.trim());
		if (m) return [parseInt(m[1], 16), parseInt(m[2], 16), parseInt(m[3], 16)];
		m = /^rgba?\(\s*([\d.]+)[,\s]+([\d.]+)[,\s]+([\d.]+)/i.exec(css.trim());
		if (m) return [+m[1], +m[2], +m[3]];
		return [255, 255, 255];
	}

	// per-frame backlight colour for one cap, called by the render loop
	function rgbFor(qmkCode: string): string {
		const matrixCode = qmkToMatrix[qmkCode];
		if (preview === 'custom') {
			return dim(custom?.[matrixCode] ?? '#0a0a0c');
		}
		const k = (matrixCode ? keyByCode.get(matrixCode) : null) ?? syntheticKeys.get(qmkCode);
		if (k && preview) return dim(effectColorOf(preview, effectParams, clock, k, pressed));
		return dim('#14181d');
	}

	let container: HTMLDivElement;
	interface PreviewHandle {
		setRgbProvider(fn: (qmk: string) => string): void;
		setLayout(id: number): void;
		onKeyClick(fn: (qmk: string) => void): void;
		dispose(): void;
	}
	let handle = $state<PreviewHandle | null>(null);

	onMount(() => {
		if (!browser || !container) return;
		let disposed = false;
		let down: ((e: KeyboardEvent) => void) | null = null;
		let up: ((e: KeyboardEvent) => void) | null = null;

		void (async () => {
			const { createKeyboardPreview } = await import('$lib/keyboard-3d');
			if (disposed) return;
			handle = (await createKeyboardPreview(container, {
				layoutId: LAYOUT_IDS[layout] ?? 100
			})) as PreviewHandle;
			handle.setRgbProvider((qmk: string) => {
				clock.time = performance.now();
				return rgbFor(qmk);
			});
			handle.onKeyClick((qmk: string) => {
				const m = qmkToMatrix[qmk];
				if (m && onKeyClick) onKeyClick(m);
			});
			down = (e: KeyboardEvent) => {
				const m = codeToMatrix(e.code);
				if (m) pressed[m] = performance.now();
			};
			up = (e: KeyboardEvent) => {
				const m = codeToMatrix(e.code);
				if (m) delete pressed[m];
			};
			window.addEventListener('keydown', down);
			window.addEventListener('keyup', up);
		})();

		return () => {
			disposed = true;
			if (down) window.removeEventListener('keydown', down);
			if (up) window.removeEventListener('keyup', up);
			handle?.dispose();
			handle = null;
		};
	});

	$effect(() => {
		// follow device layout changes without rebuilding the scene
		handle?.setLayout(LAYOUT_IDS[layout] ?? 100);
	});
</script>

<div bind:this={container} class="h-full w-full" role="region" aria-label="3d preview of keyboard"></div>
