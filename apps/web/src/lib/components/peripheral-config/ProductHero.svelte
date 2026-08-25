<script lang="ts">
	import { cn } from '$lib/utils';
	import { onMount } from 'svelte';
	import { MousePointer2, Keyboard, RotateCcw, ZoomIn, ZoomOut, Layers } from '@lucide/svelte';

	type DeviceType = 'mouse' | 'keyboard';
	type ViewMode = '3d' | 'schematic';

	let {
		deviceType = 'mouse',
		viewMode = '3d',
		onViewModeChange,
		calloutMarkers
	}: {
		deviceType: DeviceType;
		viewMode: ViewMode;
		onViewModeChange: (mode: ViewMode) => void;
		calloutMarkers: Array<{ id: string; x: number; y: number; label: string }>;
	} = $props();

	let container: HTMLDivElement;
	let rotation = 0;
	let scale = 1;
	let isDragging = false;
	let lastX = 0;
	let lastY = 0;

	function handleMouseDown(e: MouseEvent) {
		isDragging = true;
		lastX = e.clientX;
		lastY = e.clientY;
		e.preventDefault();
	}

	function handleMouseMove(e: MouseEvent) {
		if (!isDragging) return;
		const dx = e.clientX - lastX;
		const dy = e.clientY - lastY;
		rotation += dx * 0.5;
		scale = Math.max(0.5, Math.min(2, scale - dy * 0.005));
		lastX = e.clientX;
		lastY = e.clientY;
	}

	function handleMouseUp() {
		isDragging = false;
	}

	function handleWheel(e: WheelEvent) {
		e.preventDefault();
		scale = Math.max(0.5, Math.min(2, scale - e.deltaY * 0.001));
	}

	function resetView() {
		rotation = 0;
		scale = 1;
	}

	function createMouseSVG() {
		return `
			<svg viewBox="0 0 400 300" fill="none" xmlns="http://www.w3.org/2000/svg">
				<!-- Mouse body -->
				<path d="M50 150 Q50 80 120 50 Q200 20 280 50 Q350 80 350 150 Q350 220 280 250 Q200 280 120 250 Q50 220 50 150" 
					fill="url(#mouseBody)" stroke="#2E2E32" stroke-width="2"/>
				
				<!-- Left click -->
				<path d="M50 150 Q50 80 120 50 Q180 30 220 60" 
					fill="url(#leftClick)" stroke="#2E2E32" stroke-width="1.5"/>
				
				<!-- Right click -->
				<path d="M350 150 Q350 80 280 50 Q220 30 180 60" 
					fill="url(#rightClick)" stroke="#2E2E32" stroke-width="1.5"/>
				
				<!-- Scroll wheel -->
				<ellipse cx="200" cy="85" rx="28" ry="16" fill="url(#scrollWheel)" stroke="#1A1A1C" stroke-width="2"/>
				<ellipse cx="200" cy="85" rx="18" ry="8" fill="#0D0D0F" stroke="#39FF14" stroke-width="1" opacity="0.5"/>
				
				<!-- Side buttons -->
				<path d="M45 100 Q40 90 35 85 Q40 80 45 75 Q50 70 55 70 Q65 70 70 75 Q75 80 70 85 Q65 90 60 100" 
					fill="#1C1C1F" stroke="#2E2E32" stroke-width="1.5"/>
				<path d="M45 170 Q40 180 35 185 Q40 190 45 195 Q50 200 55 200 Q65 200 70 195 Q75 190 70 185 Q65 180 60 170" 
					fill="#1C1C1F" stroke="#2E2E32" stroke-width="1.5"/>
				
				<!-- Thumb rest -->
				<path d="M45 200 Q35 220 50 260 Q70 280 100 270" 
					fill="url(#thumbRest)" stroke="#2E2E32" stroke-width="1.5"/>
				
				<!-- RGB strips -->
				<path d="M60 140 Q80 110 130 100" stroke="url(#rgbGradient)" stroke-width="3" stroke-linecap="round" fill="none" opacity="0.8"/>
				<path d="M340 140 Q320 110 270 100" stroke="url(#rgbGradient)" stroke-width="3" stroke-linecap="round" fill="none" opacity="0.8"/>
				<ellipse cx="200" cy="150" rx="60" ry="10" fill="url(#rgbGradient)" opacity="0.4"/>
				
				<!-- Sensor -->
				<ellipse cx="200" cy="220" rx="12" ry="8" fill="#050505" stroke="#00FF00" stroke-width="1" opacity="0.6"/>
				
				<defs>
					<linearGradient id="mouseBody" x1="0" y1="0" x2="400" y2="300">
						<stop offset="0%" stop-color="#1C1C1F"/>
						<stop offset="50%" stop-color="#141416"/>
						<stop offset="100%" stop-color="#0D0D0F"/>
					</linearGradient>
					<linearGradient id="leftClick" x1="50" y1="150" x2="220" y2="60">
						<stop offset="0%" stop-color="#1C1C1F"/>
						<stop offset="100%" stop-color="#101012"/>
					</linearGradient>
					<linearGradient id="rightClick" x1="350" y1="150" x2="180" y2="60">
						<stop offset="0%" stop-color="#1C1C1F"/>
						<stop offset="100%" stop-color="#101012"/>
					</linearGradient>
					<linearGradient id="scrollWheel" x1="172" y1="69" x2="228" y2="101">
						<stop offset="0%" stop-color="#2A2A2E"/>
						<stop offset="50%" stop-color="#1C1C1F"/>
						<stop offset="100%" stop-color="#0F0F11"/>
					</linearGradient>
					<linearGradient id="thumbRest" x1="45" y1="200" x2="100" y2="270">
						<stop offset="0%" stop-color="#1C1C1F"/>
						<stop offset="100%" stop-color="#101012"/>
					</linearGradient>
					<linearGradient id="rgbGradient" x1="0" y1="0" x2="400" y2="0">
						<stop offset="0%" stop-color="#B026FF"/>
						<stop offset="25%" stop-color="#00FF00"/>
						<stop offset="50%" stop-color="#FF2ECC"/>
						<stop offset="75%" stop-color="#00CCFF"/>
						<stop offset="100%" stop-color="#B026FF"/>
					</linearGradient>
				</defs>
			</svg>
		`;
	}

	function createKeyboardSVG() {
		return `
			<svg viewBox="0 0 800 300" fill="none" xmlns="http://www.w3.org/2000/svg">
				<!-- Keyboard base -->
				<rect x="20" y="20" width="760" height="260" rx="16" fill="url(#kbBase)"/>
				<rect x="20" y="20" width="760" height="260" rx="16" stroke="#2E2E32" stroke-width="1.5" fill="none"/>
				
				<!-- Key rows -->
				${generateKeyRows()}
				
				<defs>
					<linearGradient id="kbBase" x1="0" y1="0" x2="800" y2="300">
						<stop offset="0%" stop-color="#1C1C1F"/>
						<stop offset="50%" stop-color="#141416"/>
						<stop offset="100%" stop-color="#0D0D0F"/>
					</linearGradient>
					<linearGradient id="keyGrad" x1="0" y1="0" x2="0" y2="50">
						<stop offset="0%" stop-color="#2A2A2E"/>
						<stop offset="100%" stop-color="#141416"/>
					</linearGradient>
					<linearGradient id="keyGradActive" x1="0" y1="0" x2="0" y2="50">
						<stop offset="0%" stop-color="#00FF00"/>
						<stop offset="100%" stop-color="#00CC00"/>
					</linearGradient>
					<linearGradient id="rgbGradient" x1="0" y1="0" x2="800" y2="0">
						<stop offset="0%" stop-color="#B026FF"/>
						<stop offset="25%" stop-color="#00FF00"/>
						<stop offset="50%" stop-color="#FF2ECC"/>
						<stop offset="75%" stop-color="#00CCFF"/>
						<stop offset="100%" stop-color="#B026FF"/>
					</linearGradient>
				</defs>
			</svg>
		`;
	}

	function generateKeyRows() {
		const rows = [
			['Esc', 'F1', 'F2', 'F3', 'F4', 'F5', 'F6', 'F7', 'F8', 'F9', 'F10', 'F11', 'F12', 'PrtSc', 'ScrLk', 'Pause'],
			['`', '1', '2', '3', '4', '5', '6', '7', '8', '9', '0', '-', '=', 'Backspace', 'Ins', 'Home', 'PgUp'],
			['Tab', 'Q', 'W', 'E', 'R', 'T', 'Y', 'U', 'I', 'O', 'P', '[', ']', '\\', 'Del', 'End', 'PgDn'],
			['Caps', 'A', 'S', 'D', 'F', 'G', 'H', 'J', 'K', 'L', ';', "'", 'Enter', ' ', ' ', ' '],
			['Shift', 'Z', 'X', 'C', 'V', 'B', 'N', 'M', ',', '.', '/', 'Shift', '↑', ' ', ' ', ' '],
			['Ctrl', 'Win', 'Alt', 'Space', 'Alt', 'Fn', 'Ctrl', '←', '↓', '→', ' ', ' ', ' ', ' ', ' ', ' ']
		];

		const keyWidth = 44;
		const keyHeight = 44;
		const gap = 4;
		const startX = 40;
		const startY = 40;

		let svg = '';
		
		rows.forEach((row, rowIdx) => {
			let x = startX;
			row.forEach((key, keyIdx) => {
				const isWide = ['Backspace', 'Tab', 'Enter', 'Shift', 'Space', 'Caps'].includes(key);
				const w = isWide ? keyWidth * (key === 'Space' ? 6.5 : 2) : keyWidth;
				const isActive = key === 'G' || key === 'Space'; // Highlighted keys
				
				svg += `
					<rect x="${x}" y="${startY + rowIdx * (keyHeight + gap)}" 
						width="${w - gap}" height="${keyHeight}" rx="6" 
						fill="${isActive ? 'url(#keyGradActive)' : 'url(#keyGrad)'}" 
						stroke="${isActive ? '#00FF00' : '#2E2E32'}" 
						stroke-width="${isActive ? '2' : '1'}"
						filter="${isActive ? 'drop-shadow(0 0 8px #00FF00)' : 'none'}"/>
					<text x="${x + (w - gap) / 2}" y="${startY + rowIdx * (keyHeight + gap) + 28}" 
						text-anchor="middle" font-family="Inter, system-ui" font-size="11" 
						font-weight="${isActive ? '700' : '500'}" fill="${isActive ? '#0A0A0C' : '#E8E8E8'}">${key}</text>
				`;
				x += w;
			});
		});

		// Numpad
		const numpadKeys = [
			['Num', '/', '*', '-'],
			['7', '8', '9', '+'],
			['4', '5', '6', ' '],
			['1', '2', '3', ' '],
			['0', '.', ' ', ' ']
		];
		
		const numpadStartX = 620;
		numpadKeys.forEach((row, rowIdx) => {
			let x = numpadStartX;
			row.forEach((key, keyIdx) => {
				const isWide = key === '+' || key === '0';
				const w = isWide ? keyWidth * 2 : keyWidth;
				if (key !== ' ') {
					svg += `
						<rect x="${x}" y="${startY + (rowIdx + 1) * (keyHeight + gap)}" 
							width="${w - gap}" height="${keyHeight}" rx="6" 
							fill="url(#keyGrad)" stroke="#2E2E32" stroke-width="1"/>
						<text x="${x + (w - gap) / 2}" y="${startY + (rowIdx + 1) * (keyHeight + gap) + 28}" 
							text-anchor="middle" font-family="Inter, system-ui" font-size="11" font-weight="500" fill="#E8E8E8">${key}</text>
					`;
				}
				x += w;
			});
		});

		return svg;
	}
</script>

<div class="product-hero" bind:this={container} onmousedown={handleMouseDown} onmousemove={handleMouseMove} onmouseup={handleMouseUp} onmouseleave={handleMouseUp} onwheel={handleWheel} style:transform="rotateY({rotation}deg) scale({scale})">
	<div class="hero-bg">
		<!-- Vignette overlay -->
		<div class="vignette" />
		
		<!-- Ambient glow -->
		<div class="ambient-glow" />
	</div>

	<div class="device-stage">
		{#if deviceType === 'mouse'}
			<div class="device-render mouse-render">{@html createMouseSVG()}</div>
		{:else}
			<div class="device-render keyboard-render">{@html createKeyboardSVG()}</div>
		{/if}
	</div>

	<!-- Callout markers on device -->
	<div class="callout-markers">
		{#each calloutMarkers as marker (marker.id)}
			<div 
				class="marker" 
				style="left: {marker.x}%; top: {marker.y}%;"
				role="button"
				aria-label={marker.label}
				tabindex="0"
			>
				<div class="marker-dot" />
				<div class="marker-pulse" />
			</div>
		{/each}
	</div>

	<!-- View controls -->
	<div class="view-controls">
		<button class="view-btn" onclick={() => onViewModeChange('3d')} class:active={viewMode === '3d'} aria-label="3D View">
			<RotateCcw class="size-4" />
		</button>
		<button class="view-btn" onclick={() => onViewModeChange('schematic')} class:active={viewMode === 'schematic'} aria-label="Schematic View">
			<Layers class="size-4" />
		</button>
		<button class="view-btn" onclick={resetView} aria-label="Reset View">
			<ZoomIn class="size-4" />
		</button>
	</div>
</div>

<style>
	.product-hero {
		position: relative;
		flex: 1;
		min-height: 500px;
		background: 
			radial-gradient(ellipse 80% 60% at 50% 30%, rgba(0, 255, 0, 0.03) 0%, transparent 70%),
			radial-gradient(ellipse 60% 80% at 50% 70%, rgba(176, 38, 255, 0.02) 0%, transparent 70%),
			#0A0A0C;
		border-radius: 16px;
		border: 1px solid #2E2E32;
		overflow: hidden;
		cursor: grab;
		transform-origin: center center;
		transition: transform 0.1s ease-out;
	}

	.product-hero:active {
		cursor: grabbing;
	}

	.hero-bg {
		position: absolute;
		inset: 0;
		pointer-events: none;
	}

	.vignette {
		position: absolute;
		inset: 0;
		background: radial-gradient(ellipse at center, transparent 40%, rgba(0, 0, 0, 0.6) 100%);
	}

	.ambient-glow {
		position: absolute;
		width: 400px;
		height: 400px;
		border-radius: 50%;
		background: radial-gradient(circle, rgba(0, 255, 0, 0.08) 0%, transparent 70%);
		top: 50%;
		left: 50%;
		transform: translate(-50%, -50%);
		pointer-events: none;
		animation: pulse 4s ease-in-out infinite;
	}

	@keyframes pulse {
		0%, 100% { opacity: 0.3; transform: translate(-50%, -50%) scale(1); }
		50% { opacity: 0.5; transform: translate(-50%, -50%) scale(1.1); }
	}

	.device-stage {
		position: absolute;
		top: 50%;
		left: 50%;
		transform: translate(-50%, -50%);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 1;
	}

	.device-render {
		filter: drop-shadow(0 40px 80px rgba(0, 0, 0, 0.5)) drop-shadow(0 0 60px rgba(0, 255, 0, 0.05));
	}

	.mouse-render {
		width: 500px;
		height: 375px;
	}

	.keyboard-render {
		width: 700px;
		height: 260px;
	}

	.callout-markers {
		position: absolute;
		inset: 0;
		pointer-events: none;
		z-index: 5;
	}

	.marker {
		position: absolute;
		pointer-events: auto;
		transform: translate(-50%, -50%);
	}

	.marker-dot {
		width: 14px;
		height: 14px;
		border-radius: 50%;
		background: #00FF00;
		border: 3px solid #0A0A0C;
		box-shadow: 
			0 0 0 2px #00FF00,
			0 0 12px rgba(0, 255, 0, 0.6),
			0 0 24px rgba(0, 255, 0, 0.3);
		animation: markerPulse 2s ease-in-out infinite;
	}

	.marker-pulse {
		position: absolute;
		top: 50%;
		left: 50%;
		width: 24px;
		height: 24px;
		border-radius: 50%;
		border: 2px solid #00FF00;
		transform: translate(-50%, -50%) scale(0);
		opacity: 0;
		animation: markerRing 2s ease-out infinite;
	}

	@keyframes markerPulse {
		0%, 100% { box-shadow: 0 0 0 2px #00FF00, 0 0 12px rgba(0, 255, 0, 0.6), 0 0 24px rgba(0, 255, 0, 0.3); }
		50% { box-shadow: 0 0 0 2px #00FF00, 0 0 20px rgba(0, 255, 0, 0.8), 0 0 40px rgba(0, 255, 0, 0.4); }
	}

	@keyframes markerRing {
		0% { transform: translate(-50%, -50%) scale(0); opacity: 0.8; }
		100% { transform: translate(-50%, -50%) scale(2.5); opacity: 0; }
	}

	.marker:focus-visible .marker-dot {
		outline: 2px solid #00FF00;
		outline-offset: 4px;
	}

	.view-controls {
		position: absolute;
		bottom: 20px;
		right: 20px;
		display: flex;
		gap: 8px;
		z-index: 10;
	}

	.view-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 40px;
		height: 40px;
		border-radius: 10px;
		border: 1px solid #2E2E32;
		background: rgba(28, 28, 31, 0.9);
		color: #8A8A8E;
		cursor: pointer;
		backdrop-filter: blur(10px);
		transition: all 0.2s ease;
	}

	.view-btn:hover {
		border-color: #00FF00;
		color: #00FF00;
		background: rgba(28, 28, 31, 1);
		box-shadow: 0 0 16px rgba(0, 255, 0, 0.15);
	}

	.view-btn.active {
		background: #1C1C1F;
		border-color: #00FF00;
		color: #00FF00;
		box-shadow: 0 0 16px rgba(0, 255, 0, 0.2), inset 0 1px 0 rgba(255, 255, 255, 0.05);
	}

	.view-btn:focus-visible {
		outline: 2px solid #00FF00;
		outline-offset: 2px;
	}
</style>