<script lang="ts">
	import { cn } from '$lib/utils';
	import { onMount } from 'svelte';
	import { MousePointer2, RotateCcw, Layers, Zap, Keyboard, Palette, SlidersHorizontal, Minus, Plus, Copy, Check, X, ChevronDown, ChevronUp, Sparkles, Droplet, Move } from '@lucide/svelte';

	type HSV = { h: number; s: number; v: number };
	type RGB = { r: number; g: number; b: number };

	let {
		selectedColor = '#00FF00',
		secondaryColor = '#0088FF',
		brightness = 100,
		speed = 50,
		activeEffect = 'static',
		onColorChange,
		onSecondaryColorChange,
		onBrightnessChange,
		onSpeedChange,
		onEffectChange
	}: {
		selectedColor: string;
		secondaryColor: string;
		brightness: number;
		speed: number;
		activeEffect: string;
		onColorChange: (color: string) => void;
		onSecondaryColorChange: (color: string) => void;
		onBrightnessChange: (value: number) => void;
		onSpeedChange: (value: number) => void;
		onEffectChange: (effect: string) => void;
	} = $props();

	let colorWheelRef: HTMLCanvasElement;
	let hueRef: HTMLCanvasElement;
	let isDraggingWheel = false;
	let isDraggingHue = false;
	let currentHSV = hexToHSV(selectedColor);
	let hueValue = 0;
	let dragType: 'brightness' | 'speed' | null = null;

	function startSliderDrag(e: MouseEvent, type: 'brightness' | 'speed') {
		e.preventDefault();
		dragType = type;
		const track = ((e.currentTarget as HTMLElement).closest('.gradient-track') as HTMLElement) ?? null;
		if (!track) return;
		const rect = track.getBoundingClientRect();
		const apply = (clientX: number) => {
			const x = Math.max(0, Math.min(rect.width, clientX - rect.left));
			const value = Math.round((x / rect.width) * 100);
			if (dragType === 'brightness') onBrightnessChange(value);
			else if (dragType === 'speed') onSpeedChange(value);
		};
		apply(e.clientX);
		const move = (ev: MouseEvent) => {
			if (!dragType) return;
			apply(ev.clientX);
		};
		const up = () => {
			dragType = null;
			window.removeEventListener('mousemove', move);
			window.removeEventListener('mouseup', up);
		};
		window.addEventListener('mousemove', move);
		window.addEventListener('mouseup', up);
	}

	const effects = [
		{ id: 'static', label: 'Static', icon: '🎨', desc: 'One steady color' },
		{ id: 'breathing', label: 'Breathing', icon: '💨', desc: 'Fade in and out' },
		{ id: 'wave', label: 'Wave', icon: '🌊', desc: 'Color wave across keys' },
		{ id: 'spectrum', label: 'Spectrum', icon: '🌈', desc: 'Cycle all colors' },
		{ id: 'reactive', label: 'Reactive', icon: '⚡', desc: 'Light on keypress' },
		{ id: 'ripple', label: 'Ripple', icon: '💧', desc: 'Ripple from center' },
		{ id: 'starlight', label: 'Starlight', icon: '✨', desc: 'Twinkling stars' },
		{ id: 'wheel', label: 'Color Wheel', icon: '🎡', desc: 'Spinning color wheel' },
		{ id: 'rainbow', label: 'Rainbow', icon: '🌈', desc: 'Flowing rainbow' },
		{ id: 'tidal', label: 'Tidal Wave', icon: '🌊', desc: 'Tidal color flow' },
		{ id: 'storm', label: 'Spinning Storm', icon: '🌀', desc: 'Spiraling colors' },
		{ id: 'lucky', label: 'Lucky Rainbow', icon: '🍀', desc: 'Random color bursts' }
	];

	const presetColors = [
		'#00FF00', '#FF0000', '#0088FF', '#FF2ECC', '#B026FF', '#FFD700',
		'#FF4400', '#00FFFF', '#FF00FF', '#88FF00', '#FF8800', '#FFFFFF'
	];

	function hexToHSV(hex: string): HSV {
		const r = parseInt(hex.slice(1, 3), 16) / 255;
		const g = parseInt(hex.slice(3, 5), 16) / 255;
		const b = parseInt(hex.slice(5, 7), 16) / 255;
		const max = Math.max(r, g, b);
		const min = Math.min(r, g, b);
		const delta = max - min;
		let h = 0;
		if (delta !== 0) {
			if (max === r) h = ((g - b) / delta) % 6;
			else if (max === g) h = (b - r) / delta + 2;
			else h = (r - g) / delta + 4;
			h *= 60;
			if (h < 0) h += 360;
		}
		const s = max === 0 ? 0 : delta / max;
		const v = max;
		return { h, s, v };
	}

	function hsvToHex(hsv: HSV): string {
		const { h, s, v } = hsv;
		const c = v * s;
		const x = c * (1 - Math.abs((h / 60) % 2 - 1));
		const m = v - c;
		let r = 0, g = 0, b = 0;
		if (h < 60) { r = c; g = x; b = 0; }
		else if (h < 120) { r = x; g = c; b = 0; }
		else if (h < 180) { r = 0; g = c; b = x; }
		else if (h < 240) { r = 0; g = x; b = c; }
		else if (h < 300) { r = x; g = 0; b = c; }
		else { r = c; g = 0; b = x; }
		const toHex = (v: number) => Math.round((v + m) * 255).toString(16).padStart(2, '0');
		return `#${toHex(r)}${toHex(g)}${toHex(b)}`;
	}

	function hsvToRGB(hsv: HSV): RGB {
		const hex = hsvToHex(hsv);
		return {
			r: parseInt(hex.slice(1, 3), 16),
			g: parseInt(hex.slice(3, 5), 16),
			b: parseInt(hex.slice(5, 7), 16)
		};
	}

	function updateColorFromHSV() {
		const hex = hsvToHex({ h: hueValue, s: currentHSV.s, v: currentHSV.v });
		selectedColor = hex;
		onColorChange(hex);
	}

	function drawColorWheel() {
		const canvas = colorWheelRef;
		if (!canvas) return;
		const ctx = canvas.getContext('2d')!;
		const size = canvas.width = canvas.height = 280;
		const center = size / 2;
		const radius = center - 4;

		// Clear
		ctx.clearRect(0, 0, size, size);

		// Draw saturation/value wheel
		for (let angle = 0; angle < 360; angle += 1) {
			const rad = (angle * Math.PI) / 180;
			ctx.beginPath();
			ctx.moveTo(center, center);
			ctx.arc(center, center, radius, rad, rad + (Math.PI / 180));
			ctx.strokeStyle = `hsl(${angle}, 100%, 50%)`;
			ctx.lineWidth = radius;
			ctx.stroke();
		}

		// Draw saturation gradient overlay
		const grad = ctx.createRadialGradient(center, center, 0, center, center, radius);
		grad.addColorStop(0, 'white');
		grad.addColorStop(1, 'transparent');
		ctx.fillStyle = grad;
		ctx.beginPath();
		ctx.arc(center, center, radius, 0, Math.PI * 2);
		ctx.fill();

		// Draw value gradient overlay
		const grad2 = ctx.createRadialGradient(center, center, radius, center, center, 0);
		grad2.addColorStop(0, 'black');
		grad2.addColorStop(1, 'transparent');
		ctx.fillStyle = grad2;
		ctx.beginPath();
		ctx.arc(center, center, radius, 0, Math.PI * 2);
		ctx.fill();

		// Draw selector
		const selectorRadius = currentHSV.s * radius;
		const selectorAngle = (hueValue * Math.PI) / 180;
		const selectorX = center + Math.cos(selectorAngle) * selectorRadius;
		const selectorY = center + Math.sin(selectorAngle) * selectorRadius;

		ctx.beginPath();
		ctx.arc(selectorX, selectorY, 10, 0, Math.PI * 2);
		ctx.fillStyle = '#0A0A0C';
		ctx.fill();
		ctx.strokeStyle = '#FFFFFF';
		ctx.lineWidth = 3;
		ctx.stroke();
		ctx.strokeStyle = '#00FF00';
		ctx.lineWidth = 1;
		ctx.stroke();
	}

	function drawHueSlider() {
		const canvas = hueRef;
		if (!canvas) return;
		const ctx = canvas.getContext('2d')!;
		const width = canvas.width = 280;
		const height = canvas.height = 24;

		const grad = ctx.createLinearGradient(0, 0, width, 0);
		for (let i = 0; i <= 360; i += 10) {
			grad.addColorStop(i / 360, `hsl(${i}, 100%, 50%)`);
		}
		ctx.fillStyle = grad;
		ctx.fillRect(0, 0, width, height);
		ctx.fillStyle = '#2E2E32';
		ctx.fillRect(0, 0, width, 2);
		ctx.fillRect(0, height - 2, width, 2);
		ctx.fillRect(0, 0, 2, height);
		ctx.fillRect(width - 2, 0, 2, height);

		// Selector
		const x = (hueValue / 360) * width;
		ctx.beginPath();
		ctx.moveTo(x, -4);
		ctx.lineTo(x - 6, 4);
		ctx.lineTo(x + 6, 4);
		ctx.closePath();
		ctx.fillStyle = '#00FF00';
		ctx.fill();
		ctx.strokeStyle = '#0A0A0C';
		ctx.lineWidth = 2;
		ctx.stroke();
	}

	function handleWheelMouseDown(e: MouseEvent) {
		isDraggingWheel = true;
		updateWheelFromMouse(e);
	}

	function handleHueMouseDown(e: MouseEvent) {
		isDraggingHue = true;
		updateHueFromMouse(e);
	}

	function handleMouseMove(e: MouseEvent) {
		if (isDraggingWheel) updateWheelFromMouse(e);
		if (isDraggingHue) updateHueFromMouse(e);
	}

	function handleMouseUp() {
		isDraggingWheel = false;
		isDraggingHue = false;
	}

	function updateWheelFromMouse(e: MouseEvent) {
		const canvas = colorWheelRef;
		if (!canvas) return;
		const rect = canvas.getBoundingClientRect();
		const center = rect.width / 2;
		const x = e.clientX - rect.left - center;
		const y = e.clientY - rect.top - center;
		const dist = Math.sqrt(x * x + y * y);
		const maxRadius = center - 4;
		let angle = Math.atan2(y, x) * 180 / Math.PI;
		if (angle < 0) angle += 360;
		hueValue = angle;
		currentHSV = { ...currentHSV, s: Math.min(1, dist / maxRadius) };
		updateColorFromHSV();
		drawColorWheel();
		drawHueSlider();
	}

	function updateHueFromMouse(e: MouseEvent) {
		const canvas = hueRef;
		if (!canvas) return;
		const rect = canvas.getBoundingClientRect();
		const x = Math.max(0, Math.min(rect.width, e.clientX - rect.left));
		hueValue = (x / rect.width) * 360;
		updateColorFromHSV();
		drawColorWheel();
		drawHueSlider();
	}

	function handleHexChange(e: Event) {
		const value = (e.target as HTMLInputElement).value;
		if (/^#[0-9A-Fa-f]{6}$/.test(value)) {
			selectedColor = value;
			currentHSV = hexToHSV(value);
			hueValue = currentHSV.h;
			onColorChange(value);
			drawColorWheel();
			drawHueSlider();
		}
	}

	function handleRGBChange(channel: 'r' | 'g' | 'b', e: Event) {
		const value = Math.max(0, Math.min(255, parseInt((e.target as HTMLInputElement).value) || 0));
		const rgb = hsvToRGB(currentHSV);
		rgb[channel] = value;
		const hex = '#' + Object.values(rgb).map(v => v.toString(16).padStart(2, '0')).join('');
		selectedColor = hex;
		currentHSV = hexToHSV(hex);
		hueValue = currentHSV.h;
		onColorChange(hex);
		drawColorWheel();
		drawHueSlider();
	}

	function selectPreset(color: string) {
		selectedColor = color;
		currentHSV = hexToHSV(color);
		hueValue = currentHSV.h;
		onColorChange(color);
		drawColorWheel();
		drawHueSlider();
	}

	function copyHex() {
		navigator.clipboard.writeText(selectedColor);
	}

	$effect(() => {
		currentHSV = hexToHSV(selectedColor);
		hueValue = currentHSV.h;
		if (colorWheelRef) drawColorWheel();
		if (hueRef) drawHueSlider();
	});

	onMount(() => {
		drawColorWheel();
		drawHueSlider();
		window.addEventListener('mousemove', handleMouseMove);
		window.addEventListener('mouseup', handleMouseUp);
		return () => {
			window.removeEventListener('mousemove', handleMouseMove);
			window.removeEventListener('mouseup', handleMouseUp);
		};
	});
</script>

<div class="lighting-panel" role="region" aria-label="RGB Lighting Configuration">
	<!-- Color Picker Section -->
	<section class="panel-section">
		<div class="section-header">
			<div class="section-badge">
				<Palette class="size-4" />
			</div>
			<h3 class="section-title">Color Picker</h3>
		</div>

		<div class="color-picker-grid">
			<!-- HSV Color Wheel -->
			<div class="wheel-container">
				<div class="wheel-wrapper">
					<canvas 
						class="color-wheel" 
						bind:this={colorWheelRef}
						width="280" height="280"
						onmousedown={handleWheelMouseDown}
						role="slider"
						aria-label="Color wheel - hue and saturation"
						aria-valuetext={`Hue ${Math.round(hueValue)}°, Saturation ${Math.round(currentHSV.s * 100)}%`}
						tabindex="0"
					/>
					<canvas 
						class="hue-slider" 
						bind:this={hueRef}
						width="280" height="24"
						onmousedown={handleHueMouseDown}
						role="slider"
						aria-label="Hue slider"
						aria-valuemin="0"
						aria-valuemax="360"
						aria-valuenow={Math.round(hueValue)}
						tabindex="0"
					/>
				</div>
			</div>

			<!-- Color Inputs -->
			<div class="color-inputs">
				<div class="input-group">
					<label class="input-label">HEX</label>
					<div class="hex-input-wrapper">
						<input 
							type="text" 
							class="hex-input" 
							value={selectedColor} 
							oninput={handleHexChange}
							onblur={handleHexChange}
							placeholder="#00FF00"
							spellcheck="false"
						/>
						<button class="copy-btn" onclick={copyHex} aria-label="Copy HEX">
							<Copy class="size-3.5" />
						</button>
					</div>
				</div>

				<div class="input-group">
					<label class="input-label">RGB</label>
					<div class="rgb-inputs">
						<div class="rgb-field">
							<span class="rgb-label">R</span>
							<input 
								type="number" 
								class="rgb-input" 
								min="0" max="255" 
								value={hsvToRGB(currentHSV).r}
								oninput={(e) => handleRGBChange('r', e)}
							/>
						</div>
						<div class="rgb-field">
							<span class="rgb-label">G</span>
							<input 
								type="number" 
								class="rgb-input" 
								min="0" max="255" 
								value={hsvToRGB(currentHSV).g}
								oninput={(e) => handleRGBChange('g', e)}
							/>
						</div>
						<div class="rgb-field">
							<span class="rgb-label">B</span>
							<input 
								type="number" 
								class="rgb-input" 
								min="0" max="255" 
								value={hsvToRGB(currentHSV).b}
								oninput={(e) => handleRGBChange('b', e)}
							/>
						</div>
					</div>
				</div>

				<!-- Preset Swatches -->
				<div class="preset-swatches">
					<label class="input-label">Presets</label>
					<div class="swatch-grid">
						{#each presetColors as color (color)}
							<button
								type="button"
								class={cn('swatch', selectedColor === color && 'active')}
								style="background: {color}"
								onclick={() => selectPreset(color)}
								aria-label={color}
								aria-pressed={selectedColor === color}
							>
								{#if selectedColor === color}
									<Check class="size-3.5" />
								{/if}
							</button>
						{/each}
					</div>
				</div>
			</div>
		</div>
	</section>

	<!-- Effect Presets -->
	<section class="panel-section">
		<div class="section-header">
			<div class="section-badge">
				<Sparkles class="size-4" />
			</div>
			<h3 class="section-title">Effect Presets</h3>
		</div>

		<div class="effect-grid">
			{#each effects as effect (effect.id)}
				<button
					type="button"
					class={cn('effect-btn', activeEffect === effect.id && 'active')}
					onclick={() => onEffectChange(effect.id)}
					aria-pressed={activeEffect === effect.id}
				>
					<span class="effect-icon">{effect.icon}</span>
					<div class="effect-info">
						<span class="effect-name">{effect.label}</span>
						<span class="effect-desc">{effect.desc}</span>
					</div>
					{#if activeEffect === effect.id}
						<div class="effect-active-indicator" />
					{/if}
				</button>
			{/each}
		</div>
	</section>

	<!-- Luminance & Speed Sliders -->
	<section class="panel-section">
		<div class="section-header">
			<div class="section-badge">
				<SlidersHorizontal class="size-4" />
			</div>
			<h3 class="section-title">Luminance & Speed</h3>
		</div>

		<div class="slider-group">
			<div class="gradient-slider">
				<div class="slider-header">
					<label class="slider-label">Luminance</label>
					<span class="slider-value">{brightness}%</span>
				</div>
				<div class="slider-track-container">
					<div 
						class="gradient-track" 
						style="background: linear-gradient(90deg, #0A0A0C 0%, {selectedColor} 100%)"
					>
						<div 
							class="slider-fill" 
							style="width: {brightness}%"
						/>
						<button
							class="slider-handle"
							style="left: {brightness}%"
							role="slider"
							aria-valuenow={brightness}
							aria-valuemin="0"
							aria-valuemax="100"
							aria-label="Luminance"
							onmousedown={(e) => startSliderDrag(e, 'brightness')}
							tabindex="0"
						>
							<Move class="size-3.5" />
						</button>
					</div>
				</div>
			</div>

			<div class="gradient-slider">
				<div class="slider-header">
					<label class="slider-label">Speed</label>
					<span class="slider-value">{speed}%</span>
				</div>
				<div class="slider-track-container">
					<div 
						class="gradient-track rainbow-track"
						style="background: linear-gradient(90deg, #B026FF 0%, #00FF00 25%, #FF2ECC 50%, #00CCFF 75%, #B026FF 100%)"
					>
						<div 
							class="slider-fill" 
							style="width: {speed}%"
						/>
						<button
							class="slider-handle"
							style="left: {speed}%"
							role="slider"
							aria-valuenow={speed}
							aria-valuemin="0"
							aria-valuemax="100"
							aria-label="Effect Speed"
							onmousedown={(e) => startSliderDrag(e, 'speed')}
							tabindex="0"
						>
							<Move class="size-3.5" />
						</button>
					</div>
				</div>
			</div>
		</div>
	</section>

	<!-- Secondary Color (for dual-color effects) -->
	{#if ['breathing', 'wave', 'starlight', 'ripple', 'tidal', 'storm'].includes(activeEffect)}
		<section class="panel-section">
			<div class="section-header">
				<div class="section-badge">
					<Palette class="size-4" />
				</div>
				<h3 class="section-title">Secondary Color</h3>
			</div>
			<div class="secondary-color-picker">
				<input 
					type="color" 
					class="color-input-hidden" 
					value={secondaryColor}
					oninput={(e) => onSecondaryColorChange((e.target as HTMLInputElement).value)}
				/>
				<button 
					class="color-swatch-large" 
					style="background: {secondaryColor}"
					onclick={(e) => { e.preventDefault(); (e.currentTarget.previousElementSibling as HTMLElement | null)?.click(); }}
					aria-label="Secondary color picker"
				/>
				<div class="secondary-presets">
					{#each presetColors as color (color)}
						<button
							type="button"
							class={cn('swatch-sm', secondaryColor === color && 'active')}
							style="background: {color}"
							onclick={() => onSecondaryColorChange(color)}
							aria-pressed={secondaryColor === color}
						/>
					{/each}
				</div>
			</div>
		</section>
	{/if}
</div>

<!-- Accent Lighting Panel (Bottom) -->
<div class="accent-panel" role="region" aria-label="Accent Lighting">
	<div class="accent-header">
		<div class="section-badge accent-badge">
			<Droplet class="size-4" />
		</div>
		<h3 class="section-title">Accent Lighting</h3>
	</div>

	<div class="accent-gradient-preview" style="background: linear-gradient(90deg, #B026FF 0%, #FF2ECC 25%, #00FF00 50%, #00CCFF 75%, #FF2ECC 100%)">
		<div class="accent-brightness-overlay" style="opacity: {1 - brightness / 100}" />
	</div>

	<div class="accent-controls">
		<div class="gradient-slider">
			<div class="slider-header">
				<label class="slider-label">Accent Brightness</label>
				<span class="slider-value">{brightness}%</span>
			</div>
			<div class="slider-track-container">
				<div 
					class="gradient-track" 
					style="background: linear-gradient(90deg, #0A0A0C 0%, #B026FF 100%)"
				>
					<div class="slider-fill" style="width: {brightness}%" />
					<button
						class="slider-handle"
						style="left: {brightness}%"
						role="slider"
						aria-valuenow={brightness}
						aria-valuemin="0"
						aria-valuemax="100"
						aria-label="Accent Brightness"
						onmousedown={(e) => startSliderDrag(e, 'brightness')}
						tabindex="0"
					>
						<Move class="size-3.5" />
					</button>
				</div>
			</div>
		</div>

		<div class="gradient-slider">
			<div class="slider-header">
				<label class="slider-label">Accent Speed</label>
				<span class="slider-value">{speed}%</span>
			</div>
			<div class="slider-track-container">
				<div class="gradient-track rainbow-track">
					<div class="slider-fill" style="width: {speed}%" />
					<button
						class="slider-handle"
						style="left: {speed}%"
						role="slider"
						aria-valuenow={speed}
						aria-valuemin="0"
						aria-valuemax="100"
						aria-label="Accent Speed"
						onmousedown={(e) => startSliderDrag(e, 'speed')}
						tabindex="0"
					>
						<Move class="size-3.5" />
					</button>
				</div>
			</div>
		</div>
	</div>
</div>



<style>
	.lighting-panel {
		display: flex;
		flex-direction: column;
		gap: 24px;
		padding: 20px 20px 24px;
	}

	.panel-section {
		background: rgba(28, 28, 31, 0.6);
		border: 1px solid #2E2E32;
		border-radius: 12px;
		padding: 16px;
		backdrop-filter: blur(10px);
	}

	.section-header {
		display: flex;
		align-items: center;
		gap: 10px;
		margin-bottom: 16px;
	}

	.section-badge {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 32px;
		height: 32px;
		border-radius: 8px;
		background: linear-gradient(135deg, #00FF00 0%, #39FF14 100%);
		color: #0A0A0C;
		flex-shrink: 0;
	}

	.section-title {
		font-family: 'Inter', 'Poppins', system-ui, sans-serif;
		font-size: 14px;
		font-weight: 600;
		color: #E8E8E8;
	}

	/* Color Picker Grid */
	.color-picker-grid {
		display: grid;
		grid-template-columns: 280px 1fr;
		gap: 20px;
		align-items: start;
	}

	.wheel-container {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.wheel-wrapper {
		position: relative;
	}

	.color-wheel {
		border-radius: 50%;
		cursor: crosshair;
		touch-action: none;
		border: 1px solid #2E2E32;
	}

	.color-wheel:focus-visible {
		outline: 2px solid #00FF00;
		outline-offset: 2px;
	}

	.hue-slider {
		border-radius: 12px;
		cursor: pointer;
		touch-action: none;
		border: 1px solid #2E2E32;
	}

	.hue-slider:focus-visible {
		outline: 2px solid #00FF00;
		outline-offset: 2px;
	}

	.color-inputs {
		display: flex;
		flex-direction: column;
		gap: 20px;
	}

	.input-group {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.input-label {
		font-family: 'Inter', 'Poppins', system-ui, sans-serif;
		font-size: 11px;
		font-weight: 600;
		color: #8A8A8E;
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}

	.hex-input-wrapper {
		display: flex;
		gap: 8px;
	}

	.hex-input {
		flex: 1;
		height: 40px;
		padding: 0 12px 0 36px;
		border: 1px solid #2E2E32;
		border-radius: 8px;
		background: #0D0D0F;
		color: #E8E8E8;
		font-family: 'JetBrains Mono', 'Fira Code', monospace;
		font-size: 13px;
		font-weight: 500;
		text-transform: uppercase;
		transition: all 0.2s ease;
	}

	.hex-input::placeholder {
		color: #5A5A5E;
	}

	.hex-input:hover {
		border-color: rgba(0, 255, 0, 0.3);
	}

	.hex-input:focus {
		outline: none;
		border-color: #00FF00;
		box-shadow: 0 0 0 3px rgba(0, 255, 0, 0.1);
	}

	.copy-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 40px;
		height: 40px;
		border: 1px solid #2E2E32;
		border-radius: 8px;
		background: #1C1C1F;
		color: #8A8A8E;
		cursor: pointer;
		transition: all 0.2s ease;
	}

	.copy-btn:hover {
		border-color: #00FF00;
		color: #00FF00;
		background: #232326;
	}

	.rgb-inputs {
		display: flex;
		gap: 8px;
	}

	.rgb-field {
		display: flex;
		flex-direction: column;
		gap: 4px;
		flex: 1;
	}

	.rgb-label {
		font-family: 'Inter', 'Poppins', system-ui, sans-serif;
		font-size: 11px;
		font-weight: 600;
		color: #8A8A8E;
	}

	.rgb-input {
		height: 36px;
		padding: 0 10px;
		border: 1px solid #2E2E32;
		border-radius: 6px;
		background: #0D0D0F;
		color: #E8E8E8;
		font-family: 'JetBrains Mono', 'Fira Code', monospace;
		font-size: 13px;
		font-weight: 500;
		text-align: center;
		transition: all 0.2s ease;
		-moz-appearance: textfield;
	}

	.rgb-input::-webkit-outer-spin-button,
	.rgb-input::-webkit-inner-spin-button {
		-webkit-appearance: none;
		margin: 0;
	}

	.rgb-input:hover {
		border-color: rgba(0, 255, 0, 0.3);
	}

	.rgb-input:focus {
		outline: none;
		border-color: #00FF00;
		box-shadow: 0 0 0 3px rgba(0, 255, 0, 0.1);
	}

	/* Preset Swatches */
	.preset-swatches {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.swatch-grid {
		display: grid;
		grid-template-columns: repeat(6, 1fr);
		gap: 8px;
	}

	.swatch {
		position: relative;
		aspect-ratio: 1;
		border-radius: 8px;
		border: 2px solid transparent;
		cursor: pointer;
		transition: all 0.2s ease;
		overflow: hidden;
	}

	.swatch:hover {
		transform: scale(1.08);
		border-color: rgba(255, 255, 255, 0.3);
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
	}

	.swatch.active {
		border-color: #00FF00;
		box-shadow: 0 0 0 2px #0A0A0C, 0 0 12px rgba(0, 255, 0, 0.4);
	}

	.swatch :global(svg) {
		color: #0A0A0C;
		filter: drop-shadow(0 0 2px #0A0A0C);
	}

	/* Effect Grid */
	.effect-grid {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 10px;
	}

	.effect-btn {
		position: relative;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 6px;
		padding: 14px 10px;
		border: 1px solid #2E2E32;
		border-radius: 10px;
		background: rgba(255, 255, 255, 0.02);
		color: #E8E8E8;
		cursor: pointer;
		transition: all 0.2s ease;
		text-align: center;
		overflow: hidden;
	}

	.effect-btn::before {
		content: '';
		position: absolute;
		inset: 0;
		background: linear-gradient(135deg, rgba(0, 255, 0, 0.05) 0%, rgba(176, 38, 255, 0.05) 100%);
		opacity: 0;
		transition: opacity 0.2s ease;
	}

	.effect-btn:hover {
		border-color: rgba(0, 255, 0, 0.4);
		background: rgba(0, 255, 0, 0.05);
		transform: translateY(-2px);
		box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3);
	}

	.effect-btn:hover::before {
		opacity: 1;
	}

	.effect-btn.active {
		border-color: #00FF00;
		background: rgba(0, 255, 0, 0.1);
		box-shadow: 
			0 0 0 1px rgba(0, 255, 0, 0.1) inset,
			0 4px 16px rgba(0, 255, 0, 0.1);
	}

	.effect-btn.active::before {
		opacity: 1;
	}

	.effect-btn:focus-visible {
		outline: 2px solid #00FF00;
		outline-offset: 2px;
	}

	.effect-icon {
		font-size: 20px;
		line-height: 1;
	}

	.effect-info {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.effect-name {
		font-family: 'Inter', 'Poppins', system-ui, sans-serif;
		font-size: 12px;
		font-weight: 600;
		color: #E8E8E8;
	}

	.effect-desc {
		font-family: 'Inter', 'Poppins', system-ui, sans-serif;
		font-size: 10px;
		color: #8A8A8E;
	}

	.effect-active-indicator {
		position: absolute;
		bottom: 0;
		left: 0;
		right: 0;
		height: 3px;
		background: linear-gradient(90deg, #00FF00 0%, #39FF14 100%);
		border-radius: 0 0 10px 10px;
	}

	/* Gradient Sliders */
	.slider-group {
		display: flex;
		flex-direction: column;
		gap: 20px;
	}

	.gradient-slider {
		display: flex;
		flex-direction: column;
		gap: 10px;
	}

	.slider-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.slider-label {
		font-family: 'Inter', 'Poppins', system-ui, sans-serif;
		font-size: 13px;
		font-weight: 600;
		color: #E8E8E8;
	}

	.slider-value {
		font-family: 'JetBrains Mono', 'Fira Code', monospace;
		font-size: 12px;
		font-weight: 600;
		color: #00FF00;
		min-width: 40px;
		text-align: right;
	}

	.slider-track-container {
		position: relative;
		height: 44px;
	}

	.gradient-track {
		position: relative;
		height: 100%;
		border-radius: 8px;
		border: 1px solid #2E2E32;
		overflow: hidden;
		cursor: pointer;
	}

	.rainbow-track {
		background: linear-gradient(90deg, #B026FF 0%, #00FF00 25%, #FF2ECC 50%, #00CCFF 75%, #B026FF 100%) !important;
	}

	.slider-fill {
		position: absolute;
		top: 0;
		left: 0;
		bottom: 0;
		border-radius: 8px 0 0 8px;
		background: linear-gradient(90deg, rgba(255, 255, 255, 0.2) 0%, rgba(255, 255, 255, 0.05) 100%);
		pointer-events: none;
		transition: width 0.1s ease-out;
	}

	.slider-handle {
		position: absolute;
		top: 50%;
		width: 28px;
		height: 28px;
		border-radius: 50%;
		background: #FFFFFF;
		border: 2px solid #00FF00;
		display: flex;
		align-items: center;
		justify-content: center;
		transform: translate(-50%, -50%);
		box-shadow: 
			0 2px 8px rgba(0, 0, 0, 0.4),
			0 0 0 2px #0A0A0C,
			0 0 12px rgba(0, 255, 0, 0.4);
		transition: transform 0.1s ease, box-shadow 0.2s ease;
		z-index: 2;
		cursor: grab;
		color: #00FF00;
	}

	.slider-handle:hover,
	.slider-handle:focus-visible {
		transform: translate(-50%, -50%) scale(1.15);
		box-shadow: 
			0 4px 16px rgba(0, 0, 0, 0.5),
			0 0 0 2px #0A0A0C,
			0 0 20px rgba(0, 255, 0, 0.6);
	}

	.slider-handle:active {
		cursor: grabbing;
	}

	.slider-handle:focus-visible {
		outline: 2px solid #00FF00;
		outline-offset: 2px;
	}

	/* Secondary Color Picker */
	.secondary-color-picker {
		display: flex;
		align-items: center;
		gap: 12px;
		flex-wrap: wrap;
	}

	.color-input-hidden {
		position: absolute;
		width: 0;
		height: 0;
		opacity: 0;
		pointer-events: none;
	}

	.color-swatch-large {
		width: 48px;
		height: 48px;
		border-radius: 10px;
		border: 2px solid #2E2E32;
		cursor: pointer;
		transition: all 0.2s ease;
		box-shadow: 
			inset 0 1px 0 rgba(255, 255, 255, 0.1),
			0 2px 8px rgba(0, 0, 0, 0.3);
	}

	.color-swatch-large:hover {
		border-color: #00FF00;
		transform: scale(1.05);
		box-shadow: 
			inset 0 1px 0 rgba(255, 255, 255, 0.1),
			0 4px 16px rgba(0, 0, 0, 0.4),
			0 0 12px rgba(0, 255, 0, 0.2);
	}

	.secondary-presets {
		display: flex;
		gap: 6px;
		flex-wrap: wrap;
	}

	.swatch-sm {
		width: 28px;
		height: 28px;
		border-radius: 6px;
		border: 2px solid transparent;
		cursor: pointer;
		transition: all 0.2s ease;
	}

	.swatch-sm:hover {
		transform: scale(1.1);
		border-color: rgba(255, 255, 255, 0.3);
	}

	.swatch-sm.active {
		border-color: #00FF00;
		box-shadow: 0 0 0 2px #0A0A0C, 0 0 8px rgba(0, 255, 0, 0.4);
	}

	/* Accent Panel */
	.accent-panel {
		margin-top: auto;
		padding-top: 16px;
		border-top: 1px solid #2E2E32;
	}

	.accent-header {
		display: flex;
		align-items: center;
		gap: 10px;
		margin-bottom: 16px;
	}

	.accent-badge {
		background: linear-gradient(135deg, #B026FF 0%, #FF2ECC 100%);
	}

	.accent-gradient-preview {
		height: 32px;
		border-radius: 8px;
		border: 1px solid #2E2E32;
		margin-bottom: 16px;
		position: relative;
		overflow: hidden;
		box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.05);
	}

	.accent-brightness-overlay {
		position: absolute;
		inset: 0;
		background: #0A0A0C;
		transition: opacity 0.1s ease;
	}

	.accent-controls {
		display: flex;
		flex-direction: column;
		gap: 16px;
	}

	/* Responsive */
	@media (max-width: 340px) {
		.color-picker-grid {
			grid-template-columns: 1fr;
		}
		
		.effect-grid {
			grid-template-columns: repeat(2, 1fr);
		}
	}
</style>