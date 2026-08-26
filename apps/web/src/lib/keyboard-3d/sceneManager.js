// @ts-nocheck
import * as THREE from 'three';
import Collection from './collection';

export default class SceneManager extends Collection {
	constructor(options) {
		super();
		this.options = options || {};
		this.el = options.el;
		this.init();
	}
	init() {
		this.scene = new THREE.Scene();
		//main renderer
		this.renderer = new THREE.WebGLRenderer({
			alpha: true,
			logarithmicDepthBuffer: true,
			antialias: true
		});
		this.renderer.localClippingEnabled = true;
		this.renderer.setPixelRatio(window.devicePixelRatio);
		this.el.appendChild(this.renderer.domElement);

		//main setup
		this.setupCamera();
		this.setupLights();
		this.resize();

		//mouse and raycaster
		this.mouse = new THREE.Vector2(-1000, -1000);
		this.raycaster = new THREE.Raycaster();
		this.raycaster.layers.set(1);
		this.onKeyClick = null;

		//bind global events
		this.onResize = () => this.resize();
		window.addEventListener('resize', this.onResize, false);
		if (typeof ResizeObserver !== 'undefined') {
			this.resizeObserver = new ResizeObserver(() => this.resize());
			this.resizeObserver.observe(this.el);
		}
		// browser zoom / monitor switches change devicePixelRatio; re-rasterise
		// at the new density or the canvas stays low-res and looks blurry
		this.onDprChange = () => {
			this.watchDpr();
			this.resize();
		};
		this.watchDpr();
		this.el.addEventListener('mousemove', this, false);
		this.el.addEventListener('click', this, false);
		this.el.addEventListener('touchstart', this, false);
	}
	// arm a media query that fires when devicePixelRatio changes again
	watchDpr() {
		if (this.dprQuery) this.dprQuery.removeEventListener('change', this.onDprChange);
		this.dprQuery = window.matchMedia(`(resolution: ${window.devicePixelRatio || 1}dppx)`);
		this.dprQuery.addEventListener('change', this.onDprChange);
	}
	handleEvent(e) {
		switch (e.type) {
			case 'mousemove':
			case 'touchstart':
				this.move(e);
				if (e.type === 'touchstart') this.mouseClick(e);
				break;
			case 'click':
				this.mouseClick(e);
				break;
		}
	}
	get w() {
		return this.el.offsetWidth;
	}
	get h() {
		return this.el.offsetHeight;
	}
	resize() {
		const w = this.w || 1;
		const h = this.h || 1;
		this.renderer.setPixelRatio(window.devicePixelRatio || 1);
		this.camera.aspect = w / h;
		this.camera.updateProjectionMatrix();
		this.renderer.setSize(w, h);
		this.updateCamera();
	}
	setupCamera() {
		this.camera = new THREE.PerspectiveCamera(60, this.w / this.h, 1, 1000);
		this._target = new THREE.Vector3(0, 1.4, 3.2);
		// near top-down "flat" product shot, framed on the board itself
		this.camera.position.y = 9.6;
		this.camera.position.z = 3.6;
		this.camera.position.x = 0;
		// aim at the board's centre so it sits centred/high in frame
		this.camera.lookAt(this._target);
		this._boardWidth = 23.5;
		this._boardDepth = 7.25;
		this.updateCamera();
	}
	setBoardSize(width, depth) {
		this._boardWidth = width;
		this._boardDepth = depth;
		this.updateCamera();
	}
	updateCamera() {
		if (!this.camera || !this._target) return;
		const w = this._boardWidth || 23.5;
		const d = this._boardDepth || 7.25;
		const padding = 1.06;
		const aspect = this.camera.aspect || 1;
		const vFOV = THREE.MathUtils.degToRad(this.camera.fov);
		const hFOV = 2 * Math.atan(Math.tan(vFOV / 2) * aspect);
		const neededW = (w / 2 * padding) / Math.tan(hFOV / 2);
		const neededD = (d / 2 * padding) / Math.tan(vFOV / 2);
		const needed = Math.max(neededW, neededD, 6);
		const targetY = this._target.y;
		const targetZ = this._target.z;
		const dz = this.camera.position.z - targetZ;
		const dy = Math.sqrt(Math.max(0, needed * needed - dz * dz));
		this.camera.position.y = targetY + Math.max(dy, needed * 0.98);
		this.camera.lookAt(this._target);
	}
	setupLights() {
		let ambiant = new THREE.AmbientLight('#ffffff', 0.5);
		this.scene.add(ambiant);

		//main
		let primaryLight = new THREE.DirectionalLight('#dddddd', 0.7);
		primaryLight.position.set(5, 10, 10);
		primaryLight.target.position.set(0, -10, -10);
		primaryLight.target.updateMatrixWorld();
		this.scene.add(primaryLight, primaryLight.target);

		//secondary shadows
		let shadowLight = new THREE.DirectionalLight('#FFFFFF', 0.2);
		shadowLight.position.set(-4, 3, -10);
		shadowLight.target.position.set(0, 0, 0);
		shadowLight.target.updateMatrixWorld();
		this.scene.add(shadowLight, shadowLight.target);
	}
	move(e) {
		let rect = this.el.getBoundingClientRect();
		let isTouch = e.type === 'touchstart' && e.touches?.length;
		let l = (isTouch ? e.touches[0].clientX : e.clientX) - rect.left;
		let t = (isTouch ? e.touches[0].clientY : e.clientY) - rect.top;
		this.mouse.x = (l / this.w) * 2 - 1;
		this.mouse.y = -(t / this.h) * 2 + 1;
	}
	mouseClick(e) {
		// re-raycast at the click position so touch/click without prior move works
		if (e.clientX != null) {
			const prev = { x: this.mouse.x, y: this.mouse.y };
			this.move(e);
			var hit = this.pickKey();
			this.mouse.x = prev.x;
			this.mouse.y = prev.y;
			if (hit && this.onKeyClick) this.onKeyClick(hit.name);
		} else if (this.intersectedObj && this.onKeyClick) {
			this.onKeyClick(this.intersectedObj.name);
		}
	}
	pickKey() {
		this.raycaster.setFromCamera(this.mouse, this.camera);
		let intersects = this.raycaster.intersectObjects(this.scene.children, true);
		if (!intersects.length) return null;
		let obj = intersects[0].object;
		if (!obj || obj.name === 'IGNORE' || obj.name === 'CASE') return null;
		return obj;
	}
	render() {
		this.update();
		this.onFrame?.();
		this.renderer.render(this.scene, this.camera);
	}
	tick() {
		if (!this.alive) return;
		this.render();
		requestAnimationFrame(this.tick.bind(this));
	}
	start() {
		if (!this.alive) {
			this.alive = true;
			requestAnimationFrame(this.tick.bind(this));
		}
	}
	destroy() {
		this.alive = false;
		window.removeEventListener('resize', this.onResize, false);
		this.resizeObserver?.disconnect();
		this.dprQuery?.removeEventListener('change', this.onDprChange);
		this.el.removeEventListener('mousemove', this, false);
		this.el.removeEventListener('click', this, false);
		this.el.removeEventListener('touchstart', this, false);
		this.components.forEach((c) => c.destroy?.());
		this.removeAll();
		this.renderer.dispose();
		this.renderer.domElement.remove();
	}
}
