// @ts-nocheck
import * as THREE from 'three';
import { keyTexture } from './texture';
import ambiantOcclusionPath from '../assets/shadow-key-noise.png';
import lightMapPath from '../assets/white.png';

let ambiantOcclusionMap = null;
let lightMap = null;

export const ensureTextures = () => {
	if (ambiantOcclusionMap && lightMap) return;
	const loader = new THREE.TextureLoader();
	ambiantOcclusionMap = loader.load(ambiantOcclusionPath);
	ambiantOcclusionMap.wrapS = THREE.RepeatWrapping;
	ambiantOcclusionMap.wrapT = THREE.RepeatWrapping;

	lightMap = loader.load(lightMapPath);
	lightMap.wrapS = THREE.RepeatWrapping;
	lightMap.wrapT = THREE.RepeatWrapping;
};

var computed_materials = {};

export const KEY_MATERIAL_STATES = {
	DEFAULT: 0,
	ACTIVE: 1,
	HIGHLIGHTED: 2
};

export const setKeyMaterialState = (mesh, state, isoent) => {
	if (state === KEY_MATERIAL_STATES.DEFAULT) {
		setMaterialIndexes(mesh, 2, 3, isoent);
	}
	if (state === KEY_MATERIAL_STATES.ACTIVE) {
		setMaterialIndexes(mesh, 0, 1, isoent);
	}
	if (state === KEY_MATERIAL_STATES.HIGHLIGHTED) {
		setMaterialIndexes(mesh, 0, 1, isoent);
	}
};

const setMaterialIndexes = (mesh, side, top, isoent) => {
	let threshold = isoent ? 10 : 6;
	mesh.geometry.faces.forEach((f, i) => {
		let isTop = i < threshold || i === 8;
		f.materialIndex = isTop ? top : side;
	});
	mesh.geometry.groupsNeedUpdate = true;
};

const getMaterialSet = (opts) => {
	let key = `mat${opts.background}`;

	let legendTexture = keyTexture(opts);
	let top = new THREE.MeshLambertMaterial({
		map: legendTexture,
		lightMap: lightMap,
		lightMapIntensity: 0
	});
	top.map.minFilter = THREE.LinearFilter;

	if (computed_materials[key]) {
		return [computed_materials[key].clone(), top];
	}
	let side = new THREE.MeshStandardMaterial({
		aoMap: ambiantOcclusionMap,
		color: opts.background,
		aoMapIntensity: 0.4,
		lightMap: lightMap,
		lightMapIntensity: 0
	});
	computed_materials[key] = side;
	return [side, top];
};

export const keyMaterials = (opts) => {
	ensureTextures();
	let base = getMaterialSet(opts);
	let active = getMaterialSet(opts);
	let materials = [...active, ...base];
	return materials;
};

export const updateMaterials = (mesh, opts) => {
	let base = getMaterialSet(opts);
	mesh.material[2] = base[0];
	mesh.material[3] = base[1];
	setKeyMaterialState(mesh, KEY_MATERIAL_STATES.DEFAULT, opts.isIsoEnt);
};

export const enableHighlight = (key_mesh, layer) => {
	key_mesh.material.forEach((m) => (m.lightMapIntensity = 0.2));
};
export const disableHighlight = (key_mesh, layer) => {
	key_mesh.material.forEach((m) => (m.lightMapIntensity = 0));
};
