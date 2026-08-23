// @ts-nocheck
import * as THREE from 'three';
import holes from './holes';
import { toRad } from './util';
import LAYOUTS from './config/layouts/layouts';

export default class CaseManager {
	constructor(opts) {
		this.scene = opts.scene;
		this.layoutId = opts.layoutId || 100;
		this.color = opts.color || '#1c1c1f';
		this.bezel = 0.5;
		this.height = 1;
		this.angle = 6;
		this.layout = LAYOUTS[this.layoutId];
		this.setup();
	}

	get width() {
		return this.layout.width + this.bezel * 2;
	}
	get depth() {
		return this.layout.height + this.bezel * 2;
	}
	get angleOffset() {
		return Math.sin(toRad(this.angle)) * this.depth;
	}

	setup() {
		this.group = new THREE.Group();
		this.group.name = 'CASE';
		this.createPlate();
		this.createCase();
		this.position();
		this.scene.add(this.group);
	}

	position() {
		this.group.rotation.x = toRad(this.angle);
		this.group.position.x = -this.layout.width / 2;
		this.group.position.y = this.angleOffset + this.height;
	}

	createPlate() {
		let geometry_plate = new THREE.PlaneGeometry(
			this.width - this.bezel * 2,
			this.depth - this.bezel * 2
		);
		let material_plate = new THREE.MeshLambertMaterial({
			color: 'black'
		});
		this.plate = new THREE.Mesh(geometry_plate, material_plate);
		this.plate.rotateX(-Math.PI / 2);
		this.plate.name = 'IGNORE';
		this.plate.layers.enable(1);
		this.plate.position.set(
			this.width / 2 - this.bezel,
			-0.5,
			this.depth / 2 - this.bezel
		);
		this.group.add(this.plate);
	}

	createCase() {
		let cornerRadius = 0.5;
		let bevel = 0.05;
		let width = this.width;
		let depth = this.depth;

		//create geometry
		let shape = new THREE.Shape();

		//basic outline
		shape.moveTo(0, cornerRadius);
		shape.quadraticCurveTo(0, 0, cornerRadius, 0);
		shape.lineTo(width - cornerRadius, 0);
		shape.quadraticCurveTo(width, 0, width, cornerRadius);
		shape.lineTo(width, depth - cornerRadius);
		shape.quadraticCurveTo(width, depth, width - cornerRadius, depth);
		shape.lineTo(cornerRadius, depth);
		shape.quadraticCurveTo(0, depth, 0, depth - cornerRadius);
		shape.lineTo(0, cornerRadius);

		shape.holes = holes(String(this.layoutId), this.layout, this.bezel);

		let extrudeOptions = {
			depth: this.height,
			steps: 1,
			bevelSegments: 1,
			bevelEnabled: true,
			bevelSize: bevel,
			bevelThickness: bevel
		};

		const geometry = new THREE.ExtrudeGeometry(shape, extrudeOptions);

		this.case = new THREE.Mesh(
			geometry,
			new THREE.MeshPhysicalMaterial({
				color: this.color,
				metalness: 0,
				roughness: 1,
				clearcoat: 0,
				clearcoatRoughness: 1
			})
		);
		this.case.name = 'CASE';
		this.case.rotation.x = Math.PI / 2;
		this.case.position.set(-this.bezel, 0, -this.bezel);
		this.group.add(this.case);
	}

	setColor(color) {
		this.color = color;
		if (this.case) this.case.material.color.set(color);
	}

	setLayout(id) {
		if (!LAYOUTS[id]) return false;
		this.layoutId = id;
		this.layout = LAYOUTS[id];
		this.group.remove(this.case);
		this.case.geometry.dispose();
		this.group.remove(this.plate);
		this.plate.geometry.dispose();
		this.createCase();
		this.createPlate();
		this.position();
		return true;
	}

	destroy() {
		this.case?.geometry.dispose();
		this.plate?.geometry.dispose();
		this.scene.remove(this.group);
	}
}
