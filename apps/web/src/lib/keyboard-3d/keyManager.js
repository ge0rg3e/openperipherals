// @ts-nocheck
import * as THREE from 'three';
import { toRad, getKeyProfile } from './util';
import qmk_codes from './config/qmk_codes.json';
import KEYMAPS from './config/keymaps/keymaps';
import LAYOUTS from './config/layouts/layouts';
import { Key, KEYSTATES } from './key/key';
import Collection from './collection';

export default class KeyManager extends Collection {
	constructor(opts) {
		super(opts);
		this.height = 1.1;
		this.angle = 6;
		this.layoutId = opts.layoutId || 100;
		this.legend = opts.legend || 'cherry';
		this.color = opts.color || '#ffffff'; // legend ink, glows with the tint
		this.background = opts.background || '#1b1e24'; // fixed cap body colour
		this.rgbProvider = null; // (qmkCode) => css hex | null
		this.setup();
	}

	setup() {
		this.group = new THREE.Object3D();
		this.group.name = 'KEYS';
		this.getLayout();
		this.getKeymap();
		this.createKeys();
		this.bindPressedEvents();
		this.position();
		this.scene.add(this.group);
	}

	get width() {
		return this.layoutFull.width;
	}
	get depth() {
		return this.layoutFull.height;
	}
	get angleOffset() {
		return Math.sin(toRad(this.angle)) * this.depth;
	}

	position() {
		this.group.rotation.x = toRad(this.angle);
		this.group.position.x = -this.layoutFull.width / 2;
		this.group.position.y = this.angleOffset + this.height;
	}

	getKeymap(id = this.layoutId) {
		this.keymap = KEYMAPS[id].layers[0];
	}

	getLayout(id = this.layoutId) {
		this.layoutFull = LAYOUTS[id];
		this.layout = LAYOUTS[id].layouts['LAYOUT'].layout;
	}

	bindPressedEvents() {
		this.onKeyDown = (e) => {
			let code = qmk_codes[e.code];
			let key = this.getKey(code);
			if (!key) return;
			key.setState(KEYSTATES.MOVING_DOWN);
			if (this.onKeyPress) this.onKeyPress(e.code);
		};
		this.onKeyUp = (e) => {
			let code = qmk_codes[e.code];
			let key = this.getKey(code);
			if (!key) return;
			key.setState(KEYSTATES.MOVING_UP);
			if (this.onKeyRelease) this.onKeyRelease(e.code);
		};
		document.addEventListener('keydown', this.onKeyDown);
		document.addEventListener('keyup', this.onKeyUp);
	}

	unbindPressedEvents() {
		document.removeEventListener('keydown', this.onKeyDown);
		document.removeEventListener('keyup', this.onKeyUp);
	}

	setRgbProvider(fn) {
		this.rgbProvider = fn;
	}

	// apply the per-key backlight colour for this frame
	applyTints() {
		if (!this.rgbProvider) return;
		for (let i = 0; i < this.components.length; i++) {
			const key = this.components[i];
			key.setTint(this.rgbProvider(key.code));
		}
	}

	removeKey(key) {
		key.destroy();
		this.remove(key);
	}

	removeAllOldKeys() {
		this.components = this.components.filter((x) => {
			let keep = this.keymap.includes(x.code);
			if (!keep) x.destroy();
			return keep;
		});
	}

	createKeys() {
		let seen = []; //for boards with multiple keys of same code
		this.removeAllOldKeys();
		for (let i = 0; i < this.layout.length; i++) {
			let code = this.keymap[i];
			let dimensions = this.layout[i];
			dimensions.row = getKeyProfile(i, this.layout, this.layoutFull.height);
			let existingKey = this.getKey(code);
			if (existingKey && !seen.includes(code)) {
				if (this.matchesSize(existingKey, dimensions)) {
					existingKey.move(dimensions);
					seen.push(code);
					continue;
				}
				this.removeKey(existingKey);
			}
			let K = new Key({
				dimensions: dimensions,
				container: this.group,
				isIso: this.layoutFull?.is_iso,
				code: code,
				legend: this.legend,
				color: this.color,
				background: this.background
			});
			K.setTint(null);
			this.add(K);
			seen.push(code);
		}
	}

	// rebuild the board for another layout id (100 / 80 / 65 / 60)
	setLayout(id) {
		if (!KEYMAPS[id] || !LAYOUTS[id]) return false;
		this.layoutId = id;
		this.getLayout(id);
		this.getKeymap(id);
		this.createKeys();
		this.position();
		return true;
	}

	getKey(code) {
		let k = this.components.find((x) => x.code === code);
		return k;
	}

	matchesSize(k, dimensions) {
		let hmatch = (k.options.dimensions?.h || 1) === (dimensions?.h || 1);
		let wmatch = (k.options.dimensions?.w || 1) === (dimensions?.w || 1);
		return hmatch && wmatch;
	}

	destroy() {
		this.unbindPressedEvents();
		this.components.forEach((key) => key.destroy());
		this.removeAll();
		this.scene.remove(this.group);
	}
}
