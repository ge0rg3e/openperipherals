<script lang="ts">
	import { sessions, applyMouseSetting, type MouseLighting, type MouseSession } from '$lib/store';
	import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Button } from '$lib/components/ui/button';
	import { Badge } from '$lib/components/ui/badge';
	import { Label } from '$lib/components/ui/label';
	import { Slider } from '$lib/components/ui/slider';
	import { Input } from '$lib/components/ui/input';
	import { Separator } from '$lib/components/ui/separator';
	import { Switch } from '$lib/components/ui/switch';
	import { Gauge, Zap, Lightbulb, Cpu, SlidersHorizontal } from '@lucide/svelte';
	import ColorField from '$lib/components/ColorField.svelte';
	import EffectPicker from '$lib/components/EffectPicker.svelte';

	let { sessionId }: { sessionId: string } = $props();

	// Resolve the live session from the store so status/busy updates re-render.
	const mouseSession = $derived.by(
		(): MouseSession | null => ($sessions.find((s) => s.id === sessionId && s.kind === 'mouse') as MouseSession | undefined) ?? null
	);
	const client = $derived(mouseSession?.client as unknown as Record<string, unknown> | null);
	const status = $derived(mouseSession?.status ?? null);
	const hints = $derived(status?.ui ?? {});
	const dpiOptions = $derived(mouseSession?.dpiOptions ?? []);

	const caps = $derived.by(() => {
		const has = (name: string) => typeof client?.[name] === 'function';
		return {
			setDpi: has('setDpi'),
			setActiveDpiStage: has('setActiveDpiStage'),
			setDpiStageCount: has('setDpiStageCount'),
			setPollingRate: has('setPollingRate'),
			setLiftOffDistance: has('setLiftOffDistance'),
			setLighting: has('setLighting'),
			setMotionSync: has('setMotionSync'),
			setAngleSnapping: has('setAngleSnapping'),
			setRippleControl: has('setRippleControl'),
			setSleepTimeout: has('setSleepTimeout'),
			setDebounceTime: has('setDebounceTime')
		};
	});

	const sleepOptions = $derived.by(() => {
		const c = client as unknown as { getSleepOptions?: () => readonly number[] };
		return c?.getSleepOptions?.() ?? [];
	});

	const debounceMaxMs = $derived.by(() => {
		const c = client as unknown as { getDebounceMaxMs?: () => number };
		return c?.getDebounceMaxMs?.() ?? 0;
	});

	// --- Sensitivity ---
	const stageEditor = $derived(hints.dpiStageEditor ?? null);
	const dpiStages = $derived(status?.dpiStages ?? []);
	const activeStage = $derived(status?.activeDpiStage ?? 0);

	const dpiMin = $derived(stageEditor ? stageEditor.minDpi : dpiOptions.length > 0 ? Math.min(...dpiOptions) : 100);
	const dpiMax = $derived(stageEditor ? stageEditor.maxDpi : dpiOptions.length > 0 ? Math.max(...dpiOptions) : 26000);
	const dpiStep = $derived(stageEditor ? stageEditor.stepDpi : dpiOptions.length > 0 ? nearestStep(dpiOptions) : 50);

	function nearestStep(options: number[]): number {
		for (const step of [50, 100, 200, 400]) {
			if (options.every((v) => v % step === 0)) return step;
		}
		return 1;
	}

	function snapDpi(value: number): number {
		if (dpiOptions.length === 0) return Math.round(value / dpiStep) * dpiStep;
		let best = dpiOptions[0];
		for (const o of dpiOptions) {
			if (Math.abs(o - value) < Math.abs(best - value)) best = o;
		}
		return best;
	}

	async function setDpi(value: number) {
		await applyMouseSetting(sessionId, (c) =>
			(c as { setDpi: (dpi: number) => Promise<unknown> }).setDpi(snapDpi(value))
		);
	}

	async function setActiveStage(stage: number) {
		await applyMouseSetting(sessionId, (c) =>
			(c as unknown as { setActiveDpiStage: (stage: number) => Promise<unknown> }).setActiveDpiStage(stage)
		);
	}

	async function setStageCount(count: number) {
		await applyMouseSetting(sessionId, (c) =>
			(c as unknown as { setDpiStageCount: (count: number) => Promise<unknown> }).setDpiStageCount(count)
		);
	}

	// --- Polling rate ---
	const COMMON_RATES = [8000, 4000, 2000, 1000, 500, 250, 125];
	const pollingRates = $derived.by(() => {
		const supported = status?.supportedPollingRates;
		if (hints.hideUnsupportedPollingRates && supported?.length) return [...supported].sort((a, b) => b - a);
		const max = supported?.length ? Math.max(...supported) : COMMON_RATES[0];
		return COMMON_RATES.filter((r) => r <= max);
	});
	const pollingReadOnly = $derived(!!hints.pollingReadOnly || !caps.setPollingRate);

	async function setPollingRate(rate: number) {
		if (pollingReadOnly) return;
		await applyMouseSetting(sessionId, (c) =>
			(c as { setPollingRate: (rate: number) => Promise<unknown> }).setPollingRate(rate)
		);
	}

	// --- Lift-off distance ---
	const lodOptions = $derived.by(() => {
		let opts = status?.supportedLiftOffDistances ?? (['Low', 'Medium', 'High'] as Array<'Low' | 'Medium' | 'High'>);
		if (hints.hideLodLow) opts = opts.filter((o) => o !== 'Low');
		return opts;
	});

	async function setLod(value: 'Low' | 'Medium' | 'High') {
		await applyMouseSetting(sessionId, (c) =>
			(c as { setLiftOffDistance: (v: 'Low' | 'Medium' | 'High') => Promise<unknown> }).setLiftOffDistance(value)
		);
	}

	// --- Lighting ---
	const lightingZones = $derived.by(() => {
		if (!status) return [] as MouseLighting[];
		if (status.lightingZones?.length) return status.lightingZones.slice(0, 8);
		return status.lighting ? [status.lighting] : [];
	});

	function applyZone(zone: MouseLighting, patch: Partial<MouseLighting>) {
		return applyMouseSetting(sessionId, (c) =>
			(c as { setLighting: (l: MouseLighting) => Promise<unknown> }).setLighting({ ...zone, ...patch })
		);
	}

	// --- Processing toggles ---
	async function setToggle(name: 'motionSync' | 'angleSnapping' | 'rippleControl', enabled: boolean) {
		const setterName = name === 'motionSync' ? 'setMotionSync' : name === 'angleSnapping' ? 'setAngleSnapping' : 'setRippleControl';
		await applyMouseSetting(sessionId, (c) =>
			(c as unknown as Record<string, (e: boolean) => Promise<unknown>>)[setterName](enabled)
		);
	}

	async function setSleep(seconds: number) {
		await applyMouseSetting(sessionId, (c) =>
			(c as { setSleepTimeout: (s: number) => Promise<unknown> }).setSleepTimeout(seconds)
		);
	}

	async function setDebounce(ms: number) {
		await applyMouseSetting(sessionId, (c) =>
			(c as { setDebounceTime: (ms: number) => Promise<unknown> }).setDebounceTime(ms)
		);
	}

	const batteryLabel = $derived.by(() => {
		const s = status;
		if (!s || s.batteryPercent == null) return null;
		return `${Math.round(s.batteryPercent)}%`;
	});

	const processingVisible = $derived(
		status &&
			!hints.hideProcessingCard &&
			((caps.setMotionSync && !hints.hideMotionSync) ||
				(caps.setAngleSnapping && !hints.hideAngleSnapping) ||
				(caps.setRippleControl && !hints.hideRippleControl))
	);
</script>

{#if status}
	<div class="flex flex-col gap-4">
		<!-- Device identity -->
		<div class="flex flex-wrap items-center gap-2">
			<span class="flex items-center gap-2 text-sm font-semibold">
				{mouseSession?.name ?? status.name}
				{#if mouseSession?.brand}
					<Badge variant="secondary" class="rounded-full px-1.5 py-0 text-[10px] capitalize">{mouseSession.brand}</Badge>
				{/if}
			</span>
		</div>

		<!-- Live status -->
		<div class="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
			{#if batteryLabel}
				<Badge variant="outline" class="gap-1.5 rounded-full text-[11px]">
					<span class={`h-1.5 w-1.5 rounded-full ${status.batteryState === 'Discharging' ? 'bg-current' : 'bg-emerald-500'}`}></span>
					{batteryLabel}
					<span class="text-muted-foreground lowercase">{status.batteryState.toLowerCase()}</span>
				</Badge>
			{/if}
			{#if status.connectionType}
				<Badge variant="outline" class="rounded-full text-[11px]">{status.connectionType}</Badge>
			{/if}
			{#if status.pollingRateHz}
				<Badge variant="outline" class="rounded-full text-[11px]">{status.pollingRateHz} Hz</Badge>
			{/if}
			{#if status.firmware.length > 0}
				<span>Firmware {status.firmware.join(', ')}</span>
			{/if}
		</div>

		<div class="grid items-start gap-4 md:grid-cols-2 xl:grid-cols-3">
			<!-- Sensitivity -->
			<Card class="border-transparent">
				<CardHeader class="pb-2">
					<CardTitle class="flex items-center gap-2.5 text-sm font-semibold">
						<span class="flex size-7 items-center justify-center rounded-full bg-primary">
							<Gauge class="size-4 text-primary-foreground" />
						</span>
						Sensitivity
					</CardTitle>
				</CardHeader>
				<CardContent class="flex flex-col gap-4">
					{#if dpiStages.length > 0 && caps.setActiveDpiStage}
						<div class="flex flex-col gap-2">
							<div class="flex items-center justify-between">
								<Label class="text-sm">DPI stage</Label>
								{#if stageEditor?.countEditable !== false && caps.setDpiStageCount && stageEditor}
									<div class="flex items-center gap-1">
										{#each Array.from({ length: stageEditor.maxStages }, (_, i) => i + 1) as count}
											<Button
												variant={dpiStages.length === count ? 'default' : 'outline'}
												size="sm"
												class="h-6 px-2 text-xs"
												onclick={() => setStageCount(count)}
											>
												{count}
											</Button>
										{/each}
									</div>
								{/if}
							</div>
							<div class="grid grid-cols-4 gap-1.5">
								{#each dpiStages as dpi, i}
									<Button
										variant={activeStage === i ? 'default' : 'outline'}
										size="sm"
										class="px-1 font-mono text-xs"
										onclick={() => setActiveStage(i)}
									>
										{(dpi ?? 0).toLocaleString()}
									</Button>
								{/each}
							</div>
						</div>
					{/if}

					{#if caps.setDpi}
						<div class="flex flex-col gap-2">
							<div class="flex items-center justify-between">
								<Label class="text-sm">Current DPI</Label>
								<span class="w-14 text-right font-mono text-xs tabular-nums">{status.dpi.toLocaleString()}</span>
							</div>
							<Slider
								type="single"
								value={snapDpi(status.dpi)}
								onValueChange={(v) => setDpi(v as number)}
								min={dpiMin}
								max={dpiMax}
								step={dpiStep}
							/>
							<div class="flex justify-between text-[10px] text-muted-foreground tabular-nums">
								<span>{dpiMin}</span>
								<span>{dpiMax.toLocaleString()}</span>
							</div>
							{#if dpiOptions.length > 0}
								<div class="flex flex-wrap gap-1.5">
									{#each [...new Set(dpiOptions)].slice(0, 8) as preset}
										<Button
											variant={status.dpi === preset ? 'default' : 'outline'}
											size="sm"
											class="h-7 px-2.5 font-mono text-xs"
											onclick={() => setDpi(preset)}
										>
											{preset >= 1000 ? `${preset / 1000}k` : preset}
										</Button>
									{/each}
								</div>
							{/if}
						</div>
					{/if}
				</CardContent>
			</Card>

			<!-- Polling + LOD -->
			<Card class="border-transparent">
				<CardHeader class="pb-2">
					<CardTitle class="flex items-center gap-2.5 text-sm font-semibold">
						<span class="flex size-7 items-center justify-center rounded-full bg-primary">
							<Zap class="size-4 text-primary-foreground" />
						</span>
						Performance
					</CardTitle>
				</CardHeader>
				<CardContent class="flex flex-col gap-4">
					<div class="flex flex-col gap-2">
						<div class="flex items-center justify-between">
							<Label class="text-sm">Polling rate</Label>
							{#if hints.pollingNote}<span class="text-xs text-muted-foreground">{hints.pollingNote}</span>{/if}
						</div>
						<div class="grid grid-cols-4 gap-1.5">
							{#each pollingRates as rate}
								<Button
									variant={status.pollingRateHz === rate ? 'default' : 'outline'}
									size="sm"
									class="px-1 font-mono text-xs"
									disabled={pollingReadOnly || (mouseSession?.busy ?? false)}
									onclick={() => setPollingRate(rate)}
								>
									{rate >= 1000 ? `${rate / 1000}k` : rate}
								</Button>
							{/each}
						</div>
					</div>

					{#if lodOptions.length > 0 && caps.setLiftOffDistance}
						<Separator />
						<div class="flex flex-col gap-2">
							<Label class="text-sm">Lift-off distance</Label>
							<div class="grid grid-cols-3 gap-1.5">
								{#each lodOptions as lod}
									<Button
										variant={status.liftOffDistance === lod ? 'default' : 'outline'}
										size="sm"
										class="text-xs capitalize"
										disabled={(mouseSession?.busy ?? false) || (!!hints.lodRequiresSurface && status.gamingSurfaceMode === 'Off')}
										onclick={() => setLod(lod)}
									>
										{lod}
									</Button>
								{/each}
							</div>
						</div>
					{/if}
				</CardContent>
			</Card>

			<!-- Lighting -->
			{#if lightingZones.length > 0 && caps.setLighting}
				<Card class="border-transparent">
					<CardHeader class="pb-2">
						<CardTitle class="flex items-center gap-2.5 text-sm font-semibold">
							<span class="flex size-7 items-center justify-center rounded-full bg-primary">
								<Lightbulb class="size-4 text-primary-foreground" />
							</span>
							Lighting
						</CardTitle>
					</CardHeader>
					<CardContent class="flex flex-col gap-4">
						{#each lightingZones as zone, zi (zone.zone)}
							<div class="flex flex-col gap-2">
								{#if lightingZones.length > 1}
									<Separator />
									<Label class="text-sm">{zone.zone}</Label>
								{/if}
								<EffectPicker
									options={zone.modes.map((mode) => ({ value: mode, label: mode.replace('Breathing ', 'Breath ') }))}
									selected={zone.mode ?? null}
									disabled={(mouseSession?.busy ?? false)}
									onSelect={(mode) => applyZone(zone, { mode: mode as MouseLighting['mode'] })}
								/>
								{#if zone.mode && zone.colorModes.includes(zone.mode)}
									<ColorField
										label="Colour"
										bind:value={() => zone.color ?? '#00ff88', (v) => applyZone(zone, { color: v })}
									/>
								{/if}
								{#if zone.mode && zone.dualColorModes.includes(zone.mode)}
								<div class="grid grid-cols-2 gap-2">
									<ColorField
										label="Primary colour"
										bind:value={() => zone.color ?? '#00ff88', (v) => applyZone(zone, { color: v })}
									/>
									<ColorField
										label="Secondary colour"
										bind:value={() => zone.color2 ?? '#0088ff', (v) => applyZone(zone, { color2: v })}
									/>
								</div>
								{/if}
								{#if zone.mode && zone.reactiveModes.includes(zone.mode)}
									<div class="grid grid-cols-4 gap-1.5">
										{#each zone.speeds as speed}
											<Button
												variant={zone.speed === speed ? 'default' : 'outline'}
												size="sm"
												class="text-xs"
												onclick={() => applyZone(zone, { speed })}
											>
												{speed}
											</Button>
										{/each}
									</div>
								{/if}
								{#if zone.brightnessLevels && zone.brightnessLevels.length > 0 && zi === 0}
									<div class="flex flex-col gap-1.5">
										<Label class="text-sm">Brightness</Label>
										<Slider
											type="single"
											value={zone.brightness ?? zone.brightnessLevels[zone.brightnessLevels.length - 1]}
											onValueChange={(v) => applyZone(zone, { brightness: v as number })}
											min={Math.min(...zone.brightnessLevels)}
											max={Math.max(...zone.brightnessLevels)}
											step={zone.brightnessLevels[1] - zone.brightnessLevels[0] || 25}
										/>
										<div class="flex justify-between text-[10px] text-muted-foreground tabular-nums">
											<span>{Math.min(...zone.brightnessLevels)}</span>
											<span>{Math.max(...zone.brightnessLevels)}</span>
										</div>
									</div>
								{/if}
							</div>
						{/each}
					</CardContent>
				</Card>
			{/if}

			<!-- Processing toggles -->
			{#if processingVisible}
				<Card class="border-transparent">
					<CardHeader class="pb-2">
						<CardTitle class="flex items-center gap-2.5 text-sm font-semibold">
							<span class="flex size-7 items-center justify-center rounded-full bg-primary">
								<Cpu class="size-4 text-primary-foreground" />
							</span>
							Sensor processing
						</CardTitle>
					</CardHeader>
					<CardContent class="flex flex-col gap-3">
						{#if caps.setMotionSync && !hints.hideMotionSync}
							<div class="flex items-center justify-between gap-3">
								<div class="flex flex-col">
									<Label class="text-sm">Motion sync</Label>
									<span class="text-xs text-muted-foreground">Synchronise sensor reads with polling</span>
								</div>
								<Switch
									checked={!!status.motionSync}
									disabled={(mouseSession?.busy ?? false)}
									onCheckedChange={(c) => setToggle('motionSync', c)}
								/>
							</div>
						{/if}
						{#if caps.setAngleSnapping && !hints.hideAngleSnapping}
							<div class="flex items-center justify-between gap-3">
								<div class="flex flex-col">
									<Label class="text-sm">Angle snapping</Label>
									<span class="text-xs text-muted-foreground">Straightens diagonal tracking</span>
								</div>
								<Switch
									checked={!!status.angleSnapping}
									disabled={(mouseSession?.busy ?? false)}
									onCheckedChange={(c) => setToggle('angleSnapping', c)}
								/>
							</div>
						{/if}
						{#if caps.setRippleControl && !hints.hideRippleControl}
							<div class="flex items-center justify-between gap-3">
								<div class="flex flex-col">
									<Label class="text-sm">Ripple control</Label>
									<span class="text-xs text-muted-foreground">Reduces sensor jitter</span>
								</div>
								<Switch
									checked={!!status.rippleControl}
									disabled={(mouseSession?.busy ?? false)}
									onCheckedChange={(c) => setToggle('rippleControl', c)}
								/>
							</div>
						{/if}
					</CardContent>
				</Card>
			{/if}

			<!-- Advanced -->
			{#if (sleepOptions.length > 0 && caps.setSleepTimeout) || (debounceMaxMs > 0 && caps.setDebounceTime)}
				<Card class="border-transparent">
					<CardHeader class="pb-2">
						<CardTitle class="flex items-center gap-2.5 text-sm font-semibold">
							<span class="flex size-7 items-center justify-center rounded-full bg-primary">
								<SlidersHorizontal class="size-4 text-primary-foreground" />
							</span>
							Advanced
						</CardTitle>
					</CardHeader>
					<CardContent class="flex flex-col gap-4">
						{#if sleepOptions.length > 0 && caps.setSleepTimeout}
							<div class="flex items-center justify-between gap-3">
								<Label class="text-sm">Auto sleep after</Label>
								<select
									class="h-8 rounded-lg border-transparent bg-secondary px-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
									value={status.sleepTimeout ?? sleepOptions[0]}
									disabled={(mouseSession?.busy ?? false)}
									onchange={(e) => setSleep(Number((e.currentTarget as HTMLSelectElement).value))}
								>
									{#each sleepOptions as seconds}
										<option value={seconds}>{seconds === 0 ? 'Never' : `${seconds}s`}</option>
									{/each}
								</select>
							</div>
						{/if}
						{#if debounceMaxMs > 0 && caps.setDebounceTime && (status.debounceMs ?? 0) > 0}
							<div class="flex flex-col gap-2">
								<div class="flex items-center justify-between">
									<Label class="text-sm">Debounce time</Label>
									<span class="w-12 text-right font-mono text-xs tabular-nums">{status.debounceMs} ms</span>
								</div>
								<Slider
									type="single"
									value={status.debounceMs ?? 4}
									onValueChange={(v) => setDebounce(v as number)}
									min={2}
									max={debounceMaxMs}
									step={2}
								/>
								<div class="flex justify-between text-[10px] text-muted-foreground tabular-nums">
									<span>2 ms</span>
									<span>{debounceMaxMs} ms</span>
								</div>
							</div>
						{/if}
					</CardContent>
				</Card>
			{/if}
		</div>

		<p class="text-xs text-muted-foreground">
			Settings are written straight to the mouse and verified by read-back. Values the driver cannot confirm are hidden.
		</p>
	</div>
{/if}
