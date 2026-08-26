// @ts-nocheck
import SceneManager from './sceneManager';
import CaseManager from './caseManager';
import KeyManager from './keyManager';
import legendsFontUrl from './assets/fonts/legends.woff';
import LAYOUTS from './config/layouts/layouts';

let legendsFontRequested = false;

async function ensureLegendsFont() {
	if (legendsFontRequested) return;
	legendsFontRequested = true;
	try {
		const font = new FontFace('legends', `url(${legendsFontUrl})`);
		await font.load();
		document.fonts.add(font);
	} catch {
		/* fall back to system font for legends */
	}
}

export async function createKeyboardPreview(element, options = {}) {
	//ensure fonts loaded for canvas textures
	await ensureLegendsFont();

	const ThreeApp = new SceneManager({ el: element });

	const KEYS = new KeyManager({
		scene: ThreeApp.scene,
		layoutId: options.layoutId || 100
	});

	const caseManager = new CaseManager({
		scene: ThreeApp.scene,
		layoutId: options.layoutId || 100,
		color: options.caseColor || '#1c1c1f'
	});

	function boardSize(id) {
		const l = LAYOUTS[id];
		if (!l) return { w: 23.5, d: 7.25 };
		return { w: l.width + 1, d: l.height + 1 };
	}
	const initial = boardSize(options.layoutId || 100);
	ThreeApp.setBoardSize(initial.w, initial.d);

	//start render loop; tints are refreshed every frame before drawing
	ThreeApp.onFrame = () => KEYS.applyTints();
	ThreeApp.add(KEYS);
	ThreeApp.start();

	return {
		setRgbProvider(fn) {
			KEYS.setRgbProvider(fn);
		},
		setLayout(id) {
			KEYS.setLayout(id);
			caseManager.setLayout(id);
			const s = boardSize(id);
			ThreeApp.setBoardSize(s.w, s.d);
		},
		onKeyClick(fn) {
			ThreeApp.onKeyClick = fn;
		},
		dispose() {
			ThreeApp.destroy();
			caseManager.destroy();
		}
	};
}
