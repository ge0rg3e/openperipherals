// @ts-nocheck
import * as THREE from 'three';
import { keyMaterials, setKeyMaterialState, KEY_MATERIAL_STATES } from './materials';
import { keyGeometry, keyGeometryISOEnter } from './geometry';

export const KEYSTATES = {
	INITIAL: 0, // full height
	PRESSED: 1, // bottomed out
	MOVING_UP: 2, // resetting after press
	MOVING_DOWN: 3 // being pressed down
};

// A single key, all size and  values are in std key units unless otherwise specified
export class Key {
	constructor(options) {
		this.options = options || {};
		this.code = this.options.code;
		this.state = KEYSTATES.INITIAL;
		this.previousState = KEYSTATES.INITIAL;
		this.is_iso_enter = this.code === 'KC_ENT' && this.options.isIso;
		this.direction = -1; // is key moving up or down
		this.gutter = 0.05; // space inbetween keys (this is subtracted from the key width not added after the key)
		this.start_y = -0.05; // initial y position and reset after releasing key
		this.dist_pressed = 0.25; // max vertical distance the key can be pressed down
		this.press_velocity = 0.1; // speed of press, smaller = smoother slower motion
		this.legend = this.options.legend || 'cherry';
		this.tintCache = null;
		this.setup();
	}

	setup() {
		this.geometryOptions = {
			row: this.row,
			w: this.w,
			h: this.h,
			x: 1,
			y: 1
		};

		let geometry = this.is_iso_enter
			? keyGeometryISOEnter(this.geometryOptions)
			: keyGeometry(this.geometryOptions);
		let materials = keyMaterials(this.materialOptions);

		this.cap = new THREE.Mesh(geometry, materials);
		this.cap.name = this.options.code;
		// clickable by the scene raycaster, which only tests layer 1
		this.cap.layers.enable(1);
		setKeyMaterialState(this.cap, KEY_MATERIAL_STATES.DEFAULT, this.is_iso_enter);

		this.cap.castShadow = false;
		this.cap.receiveShadow = false;
		this.cap.position.y = this.start_y;
		this.cap.position.x = this.x;
		this.cap.position.z = this.y;
		this.options.container.add(this.cap);
	}
	get h() {
		return this.options.dimensions.h || 1;
	}
	get w() {
		return this.options.dimensions.w || 1;
	}
	get x() {
		return this.options.dimensions.x || 0;
	}
	get y() {
		return this.options.dimensions.y || 0;
	}
	get row() {
		return this.options.dimensions.row || 1;
	}
	get materialOptions() {
		return {
			w: this.w,
			h: this.h,
			sub: '',
			legend: this.legend,
			code: this.options.code,
			color: this.options.color,
			background: this.options.background,
			isIsoEnt: this.is_iso_enter
		};
	}
	destroy() {
		this.cap.material.forEach((m) => m.dispose());
		this.cap.geometry.dispose();
		this.options.container.remove(this.cap);
	}
	move(dimensions) {
		this.options.dimensions = dimensions;
		this.cap.position.x = this.x;
		this.cap.position.z = this.y;
	}
	// set the state of the key and prevent chaning in wrong order
	setState(state) {
		// if keyup event fires before key is finished animating down, add up animating to the queue
		if (this.state === KEYSTATES.MOVING_DOWN && state === KEYSTATES.MOVING_UP) {
			this.queueRelease = true;
		}
		if (
			(this.state === KEYSTATES.INITIAL &&
				(state === KEYSTATES.PRESSED || state === KEYSTATES.MOVING_UP)) ||
			(this.state === KEYSTATES.PRESSED &&
				(state === KEYSTATES.INITIAL || state === KEYSTATES.MOVING_DOWN))
		) {
			return;
		}
		this.state = state;
	}
	// reset key to stating position and update state
	reset() {
		this.cap.position.y = this.start_y;
		this.setState(KEYSTATES.INITIAL);
		this.direction = -1;
	}
	// set key to fully pressed position and update state
	bottomOut() {
		this.cap.position.y = this.start_y - this.dist_pressed;
		this.setState(KEYSTATES.PRESSED);
		this.direction = 1;
		if (this.queueRelease) {
			this.setState(KEYSTATES.MOVING_UP);
			this.queueRelease = false;
		}
	}
	// per-frame backlight colour applied to the cap top (legend + halo glow).
	// the keycap body itself keeps its fixed colour; only the texture on top
	// is multiplied by the tint, so legends light up instead of whole caps.
	// passing null resets the top to its baked, untinted colours.
	setTint(hex) {
		if (hex === this.tintCache) return;
		this.tintCache = hex;
		const color = hex ? new THREE.Color(hex) : null;
		for (let i = 0; i < 4; i++) {
			const m = this.cap.material[i];
			if (!m) continue;
			const isTop = i === 1 || i === 3;
			if (color && isTop) m.color.copy(color);
			else if (isTop) m.color.set('#ffffff');
			else m.color.set(this.options.background);
		}
	}
	// update key
	update() {
		// check if key needs to be updated
		if (this.state === KEYSTATES.INITIAL || this.state === KEYSTATES.PRESSED) return;
		// animate key up or down
		this.cap.position.y = this.cap.position.y + this.press_velocity * this.direction;

		// key has reached max height
		if (this.cap.position.y >= this.start_y) {
			this.reset();
		}
		// key has bottomed out
		if (this.cap.position.y <= this.start_y - this.dist_pressed) {
			this.bottomOut();
		}
	}
}
