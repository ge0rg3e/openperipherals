<script lang="ts">
	import { sessions, type KeyboardSession } from '$lib/store';
	import type { EffectKind } from '$lib/index';
	import type { EffectParams } from '$lib/controller';
	import KeyboardPreview from '$lib/components/KeyboardPreview.svelte';
	import { Card, CardContent } from '$lib/components/ui/card';
	import { Button } from '$lib/components/ui/button';
	import { Badge } from '$lib/components/ui/badge';
	import { Label } from '$lib/components/ui/label';
	import { Slider } from '$lib/components/ui/slider';
	import { Separator } from '$lib/components/ui/separator';
	import { Switch } from '$lib/components/ui/switch';
	import { Palette } from '@lucide/svelte';
	import ColorField from '$lib/components/ColorField.svelte';
	import EffectPicker from '$lib/components/EffectPicker.svelte';
	import { customSupported, ELITE_CELLS } from '$lib/keyboard/matrix';
	import { keyboardMatrixKeys, previewOnlyCodes } from '$lib/keyboard/layout';

	let { sessionId }: { sessionId: string } = $props();

	// Resolve the live session so workspace state follows the store.
	const kbSession = $derived.by(
		() => $sessions.find((s) => s.id === sessionId && s.kind === 'keyboard') as KeyboardSession | null
	);
	const controller = $derived(kbSession?.controller ?? null);
	const info = $derived(kbSession?.info ?? null);

	const vendor = $derived(info?.vendor ?? 'razer');
	const kbd = $derived(info?.vendor === 'razer' ? info.kbd : undefined);
	const lkb = $derived(info?.vendor === 'logitech' ? info.lkb : undefined);
	const rkb = $derived(info?.vendor === 'redragon' ? info.rdk : undefined);

	// --- Effect state (persisted per browser, applied to this session's device) ---
	let kind = $state<EffectKind>('static');
	let color1 = $state('#00ff88');
	let color2 = $state('#0088ff');
	let speed = $state(3);
	let breathingMode = $state<'single' | 'dual' | 'random'>('single');
	let starlightMode = $state<'single' | 'dual' | 'random'>('single');
	let direction = $state<'left' | 'right'>('right');
	let brightness = $state(127);
	let paintColor = $state('#ffffff');
	let customColors = $state<Record<string, string>>({});
	let battery = $state<{ level: number; charging: boolean } | null>(null);
	let gameMode = $state<boolean>(false);
	let macroLeds = $state<boolean>(false);

	const STORE_KEY = 'openperipherals:effect';

	function loadProfile() {
		try {
			const raw = localStorage.getItem(STORE_KEY);
			if (!raw) return;
			const s = JSON.parse(raw);
			if (typeof s.kind !== 'string') return;
			kind = s.kind;
			if (typeof s.color === 'string') color1 = s.color;
			if (typeof s.color2 === 'string') color2 = s.color2;
			if (typeof s.speed === 'number') speed = s.speed;
			if (s.breathingMode === 'single' || s.breathingMode === 'dual' || s.breathingMode === 'random') breathingMode = s.breathingMode;
			if (s.starlightMode === 'single' || s.starlightMode === 'dual' || s.starlightMode === 'random') starlightMode = s.starlightMode;
			if (s.direction === 'left' || s.direction === 'right') direction = s.direction;
			if (typeof s.custom === 'object' && s.custom !== null) customColors = s.custom;
		} catch {
			/* ignore corrupt storage */
		}
	}

	function saveProfile() {
		try {
			localStorage.setItem(
				STORE_KEY,
				JSON.stringify({ kind, color: color1, color2, speed, breathingMode, starlightMode, direction, custom: customColors })
			);
		} catch {
			/* storage unavailable */
		}
	}

	function paintKey(code: string) {
		customColors = { ...customColors, [code]: paintColor };
	}

	function fillCustom() {
		const next: Record<string, string> = {};
		for (const k of keyboardMatrixKeys()) next[k.code] = paintColor;
		if (isElite) for (const c of ELITE_CELLS) next[c.code] = paintColor;
		for (const c of previewOnlyCodes) next[c] = paintColor;
		customColors = next;
	}

	// Load the saved profile and read back the device's current state once.
	if (typeof window !== 'undefined') {
		loadProfile();
		$effect(() => saveProfile());
	}

	// Flips true once the initial state read-back has settled. The HID transport
	// has no internal queue - parallel feature-report exchanges steal each
	// other's replies - so every device exchange must stay serialized.
	let deviceStateLoaded = $state(false);
	$effect(() => {
		const c = controller;
		if (!c) return;
		void (async () => {
			try {
				const b = await c.getBrightness().catch(() => null);
				if (b != null) brightness = b;
				const gm = await c.getGameMode().catch(() => null);
				if (gm != null) gameMode = gm;
				const ml = await c.getMacroLeds().catch(() => null);
				if (ml != null) macroLeds = ml;
				const bat = await c.getBattery().catch(() => null);
				if (bat) battery = bat;
			} finally {
				deviceStateLoaded = true;
			}
		})();
	});

	// Bumped on every effect-tile click so the auto-apply re-sends even when the
	// user re-picks the tile that's already active.
	let applyNonce = $state(0);

	const effectParams = $derived.by<EffectParams>(() => {
		void applyNonce;
		const p: EffectParams = { kind };
		if (kind === 'static' || kind === 'reactive') p.color = color1;
		if (kind === 'breathing') {
			p.mode = breathingMode;
			p.color = color1;
			p.color2 = color2;
		}
		if (kind === 'starlight') {
			p.mode = starlightMode;
			p.color = color1;
			p.color2 = color2;
			p.speed = speed;
		}
		if (kind === 'reactive') p.speed = speed;
		if (kind === 'wave' || kind === 'wheel') p.direction = direction;
		if (kind === 'custom') p.custom = customColors;
		return p;
	});

	// Auto-apply on any input change, debounced. Also pushes the current effect
	// to the board when the session appears - but only after the initial state
	// read-back has finished, so it can never interleave with those exchanges.
	let applyTimer: ReturnType<typeof setTimeout> | undefined;
	$effect(() => {
		const p = effectParams;
		const c = controller;
		if (!c || !deviceStateLoaded) return;
		clearTimeout(applyTimer);
		applyTimer = setTimeout(() => c.apply(p).catch(() => {}), 120);
	});

	const showColor1 = $derived(kind === 'static' || kind === 'reactive' || kind === 'breathing' || kind === 'starlight');
	const showColor2 = $derived((kind === 'breathing' && breathingMode === 'dual') || (kind === 'starlight' && starlightMode === 'dual'));
	const showMode = $derived(kind === 'breathing' || kind === 'starlight');
	const showSpeed = $derived(kind === 'reactive' || kind === 'starlight');
	const showDirection = $derived(kind === 'wave' || kind === 'wheel');
	const hasOptions = $derived(showMode || showColor1 || showSpeed || showDirection);

	// Per-key "custom" lighting needs a matrix + a driver exposing a custom-frame
	// interface; Redragon EVision firmware gets solid colour "custom" instead.
	const canCustom = $derived(vendor === 'razer' ? !!kbd?.custom && customSupported(kbd.matrixRows, kbd.matrixCols) : vendor === 'logitech' ? !!lkb?.custom : false);
	// The Huntsman Elite (9-row matrix) exposes media keys + a wrist-rest lightbar.
	const isElite = $derived(!!kbd && kbd.matrixRows === 9 && kbd.matrixCols === 22);
	// The "Wheel" hardware effect only exists on the BlackWidow V4 family.
	const canWheel = $derived(!!kbd?.wheel);
	const logiEffects = $derived(new Set(lkb?.effects ?? []));
	const redragonEffects = $derived(new Set(rkb?.effects ?? []));
	const effectList = $derived.by<Array<[EffectKind, string]>>(() => {
		const list: Array<[EffectKind, string]> = [
			['off', 'Off'],
			['static', 'Static']
		];
		if (vendor === 'razer') {
			list.push(['wave', 'Wave']);
			if (canWheel) list.push(['wheel', 'Wheel']);
			list.push(['spectrum', 'Spectrum'], ['reactive', 'Reactive'], ['breathing', 'Breath'], ['starlight', 'Star']);
		} else if (vendor === 'redragon') {
			if (redragonEffects.has('wave')) list.push(['wave', 'Wave']);
			if (redragonEffects.has('spectrum')) list.push(['spectrum', 'Spectrum']);
			if (redragonEffects.has('reactive')) list.push(['reactive', 'Reactive']);
			if (redragonEffects.has('breathing')) list.push(['breathing', 'Breath']);
			if (redragonEffects.has('starlight')) list.push(['starlight', 'Star']);
		} else {
			if (logiEffects.has('wave')) list.push(['wave', 'Wave']);
			if (logiEffects.has('spectrum')) list.push(['spectrum', 'Spectrum']);
			if (logiEffects.has('breathing')) list.push(['breathing', 'Breath']);
			if (logiEffects.has('reactive')) list.push(['reactive', 'Ripple']);
		}
		if (canCustom) list.push(['custom', 'Custom']);
		return list;
	});

	// Fall back to a hardware effect if the connected board can't do the selected one.
	$effect(() => {
		if (!controller) return;
		if (!effectList.some(([k]) => k === kind)) kind = 'static';
	});

	function brandLogo(v: string): string | null {
		return v === 'razer' ? '/razer.png' : v === 'logitech' ? '/logitech.png' : v === 'redragon' ? '/redragon.png' : null;
	}
</script>

{#if kbSession && controller}
	<div class="flex min-h-0 flex-1 flex-col gap-3">
		<!-- Device identity -->
		<div class="flex shrink-0 flex-wrap items-center justify-between gap-2">
			<div class="flex flex-wrap items-center gap-2">
				{#if brandLogo(vendor)}
					<img src={brandLogo(vendor)} alt={vendor} class="h-5 w-auto shrink-0 object-contain" />
				{/if}
				<span class="text-sm font-semibold">{kbSession.name}</span>
				{#if kbSession.serial}
					<span class="hidden text-xs text-muted-foreground sm:inline">Serial {kbSession.serial}</span>
				{/if}
				{#if battery}
					<Badge variant="outline" class="gap-1.5 rounded-full text-[11px]">
						<span class={`h-1.5 w-1.5 rounded-full ${battery.charging ? 'bg-emerald-500' : 'bg-current'}`}></span>
						{`${battery.level}%`}
						{#if battery.charging}
							<span class="text-muted-foreground">charging</span>
						{/if}
					</Badge>
				{/if}
			</div>
			{#if kbSession.firmware}
				<span class="text-xs text-muted-foreground">Firmware {kbSession.firmware}</span>
			{/if}
		</div>

		<div class="flex min-h-0 flex-1 flex-col gap-4">
			<!-- Keyboard stage -->
			<section class="flex min-h-0 flex-1 flex-col">
				<div class="relative min-h-[320px] flex-1 overflow-hidden rounded-2xl bg-transparent p-4 sm:p-6">
					<KeyboardPreview
						preview={kind}
						layout={kbd?.layout ?? lkb?.layout ?? rkb?.layout ?? 'full'}
						{effectParams}
						{brightness}
						custom={customColors}
						onKeyClick={kind === 'custom' && canCustom ? paintKey : null}
					/>
				</div>
			</section>

			<!-- Controls -->
			<Card size="sm" class="shrink-0 border-transparent">
				<CardContent class="flex flex-col gap-3">
					<div class="flex items-center gap-2">
						<span class="flex size-6 items-center justify-center rounded-full bg-primary">
							<Palette class="size-3.5 text-primary-foreground" />
						</span>
						<span class="text-sm font-semibold">Effect</span>
					</div>

					<EffectPicker
						options={effectList.map(([value, label]) => ({ value, label }))}
						selected={kind}
						onSelect={(value) => {
							kind = value as EffectKind;
							applyNonce++;
						}}
					/>

					{#if hasOptions}
						<div class="flex flex-wrap items-center gap-x-4 gap-y-2">
							{#if showMode}
								<div class="flex items-center gap-1.5">
									<span class="text-xs text-muted-foreground">Mode</span>
									{#each ['single', 'dual', 'random'] as m}
										<Button variant={breathingMode === m ? 'default' : 'outline'} size="sm" class="h-7 px-2.5 text-xs capitalize" onclick={() => (breathingMode = m as typeof breathingMode)}>
											{m}
										</Button>
									{/each}
								</div>
							{/if}

							{#if showColor1}
								<div class="flex items-center gap-2">
									<Label class="shrink-0 text-xs text-muted-foreground">Color</Label>
									<ColorField bind:value={color1} label="Primary colour" class="h-8 w-36" />
								</div>
								{#if showColor2}
									<div class="flex items-center gap-2">
										<Label class="shrink-0 text-xs text-muted-foreground">Color 2</Label>
										<ColorField bind:value={color2} label="Secondary colour" class="h-8 w-36" />
									</div>
								{/if}
							{/if}

							{#if showSpeed}
								<div class="flex min-w-48 flex-1 items-center gap-2">
									<Label class="shrink-0 text-xs text-muted-foreground">Speed</Label>
									<Slider type="single" value={speed} onValueChange={(v) => (speed = v as number)} min={1} max={4} step={1} />
									<span class="w-12 shrink-0 text-right text-xs text-muted-foreground">{['', 'Fast', 'Medium', 'Slow', 'Slowest'][speed]}</span>
								</div>
							{/if}

							{#if showDirection}
								<div class="flex items-center gap-1.5">
									<Button variant={direction === 'left' ? 'default' : 'outline'} size="sm" class="h-7 px-2.5 text-xs" onclick={() => (direction = 'left')}>← Left</Button>
									<Button variant={direction === 'right' ? 'default' : 'outline'} size="sm" class="h-7 px-2.5 text-xs" onclick={() => (direction = 'right')}>Right →</Button>
								</div>
							{/if}
						</div>
					{/if}

					{#if kind === 'custom' && canCustom}
						<div class="flex flex-wrap items-center gap-x-3 gap-y-2">
							<Label class="shrink-0 text-xs text-muted-foreground">Paint</Label>
							<ColorField bind:value={paintColor} label="Paint colour" class="h-8 w-36" />
							<Button variant="outline" size="sm" class="h-7 px-2.5 text-xs" onclick={() => (customColors = {})}>Clear</Button>
							<Button variant="outline" size="sm" class="h-7 px-2.5 text-xs" onclick={fillCustom}>Fill all</Button>
							<span class="text-[10px] text-muted-foreground" title="Per-key lighting is host-rendered and resets when the device powers off.">
								Click keys on the device to paint them
							</span>
						</div>
					{/if}

					<Separator />

					<div class="flex items-center gap-3">
						<Label class="shrink-0 text-sm">Brightness</Label>
						<Slider
							type="single"
							value={brightness}
							onValueChange={(v) => {
								brightness = v as number;
								controller.setBrightness(v as number).catch(() => {});
							}}
							min={0}
							max={255}
							step={1}
						/>
						<span class="w-10 shrink-0 text-right font-mono text-xs tabular-nums">{Math.round((brightness / 255) * 100)}%</span>
					</div>

					{#if vendor === 'razer'}
						<div class="flex flex-wrap items-center gap-x-5 gap-y-2">
							<span class="text-xs font-medium tracking-wider text-muted-foreground uppercase">Device</span>
							<div class="flex items-center gap-2" title="Disables the Windows key">
								<Label class="text-sm">Game mode</Label>
								<Switch
									checked={gameMode}
									onCheckedChange={(c) => {
										gameMode = c;
										controller.setGameMode(c).catch(() => {});
									}}
								/>
							</div>
							<div class="flex items-center gap-2" title="M1-M5 backlight">
								<Label class="text-sm">Macro key lights</Label>
								<Switch
									checked={macroLeds}
									onCheckedChange={(c) => {
										macroLeds = c;
										controller.setMacroLeds(c).catch(() => {});
									}}
								/>
							</div>
						</div>
					{/if}
				</CardContent>
			</Card>
		</div>
	</div>
{/if}
