<script lang="ts">
	import { cn } from '$lib/utils';

	type Option = { value: string; label: string };

	let {
		options,
		selected,
		onSelect,
		disabled = false
	}: {
		options: Array<Option>;
		selected: string | null;
		onSelect: (value: string) => void;
		disabled?: boolean;
	} = $props();

	const descriptions: Record<string, string> = {
		off: 'Turns all lighting off.',
		static: 'One steady colour across the device.',
		wave: 'A rainbow wave sweeps along the device.',
		wheel: 'Colours chase around like a spinning wheel.',
		spectrum: 'Cycles smoothly through every colour.',
		reactive: 'Lights up on each keypress, then fades out.',
		breathing: 'Gently fades in and out of a colour.',
		starlight: 'Random keys twinkle like a night sky.',
		custom: 'Paint each key exactly how you want.'
	};

	function previewKind(value: string): keyof typeof descriptions {
		const v = value.toLowerCase();
		if (v === 'off' || v.startsWith('off')) return 'off';
		if (v.includes('custom')) return 'custom';
		if (v.includes('star')) return 'starlight';
		if (v.includes('breath')) return 'breathing';
		if (v.includes('react') || v.includes('ripple')) return 'reactive';
		if (v.includes('wheel')) return 'wheel';
		if (v.includes('spectrum') || v.includes('rainbow')) return 'spectrum';
		if (v.includes('wave')) return 'wave';
		return 'static';
	}
</script>

<div class="flex flex-wrap gap-1.5">
	{#each options as option (option.value)}
		{@const pk = previewKind(option.value)}
		<button
			type="button"
			disabled={disabled}
			aria-pressed={selected === option.value}
			title={descriptions[pk]}
			onclick={() => onSelect(option.value)}
			class={cn(
				'flex items-center gap-2 rounded-lg border py-1 pl-1 pr-2.5 transition-colors',
				selected === option.value
					? 'border-primary/60 bg-primary/10'
					: 'border-transparent bg-secondary/50 hover:bg-secondary'
			)}
		>
			<span class="effect-preview effect-{pk}" aria-hidden="true">
				{#each Array(12) as _, i}<span class="pk"></span>{/each}
			</span>
			<span class="text-xs font-medium">{option.label}</span>
		</button>
	{/each}
</div>

<style>
	/* mini keyboard: a 4x3 grid of keycap dots, lit per effect */
	.effect-preview {
		display: grid;
		grid-template-columns: repeat(4, 1fr);
		grid-template-rows: repeat(3, 1fr);
		gap: 1.5px;
		flex: none;
		width: 34px;
		height: 24px;
		padding: 3px 4px;
		border-radius: 6px;
		background: #0b0d0f;
		box-shadow: inset 0 0 0 1px rgb(255 255 255 / 0.12);
		overflow: hidden;
	}
	.pk {
		border-radius: 2px;
		background: #1d2127;
	}
	.effect-off .pk {
		background: #23272e;
	}
	.effect-static .pk {
		background: var(--primary);
	}
	.effect-breathing .pk {
		background: var(--primary);
		animation: ep-breathe 1.8s ease-in-out infinite;
	}
	/* wave: rainbow columns drifting through the hue wheel */
	.effect-wave .pk {
		animation: ep-hue 2.4s linear infinite;
	}
	.effect-wave .pk:nth-child(4n + 1) {
		background: hsl(0 85% 55%);
	}
	.effect-wave .pk:nth-child(4n + 2) {
		background: hsl(35 85% 55%);
	}
	.effect-wave .pk:nth-child(4n + 3) {
		background: hsl(90 85% 55%);
	}
	.effect-wave .pk:nth-child(4n + 4) {
		background: hsl(200 85% 55%);
	}
	.effect-wheel .pk {
		background: hsl(0 85% 55%);
		animation: ep-hue 1.6s linear infinite;
	}
	.effect-spectrum .pk {
		background: hsl(0 85% 55%);
		animation: ep-hue 2.8s linear infinite;
	}
	/* reactive: a single key flares and fades like a keypress echo */
	.effect-reactive .pk:nth-child(7) {
		background: var(--primary);
		animation: ep-reactive 1.6s ease-out infinite;
	}
	/* starlight: scattered keys twinkle out of phase */
	.effect-starlight .pk:nth-child(2),
	.effect-starlight .pk:nth-child(5),
	.effect-starlight .pk:nth-child(9),
	.effect-starlight .pk:nth-child(12) {
		background: var(--primary);
		animation: ep-twinkle 1.6s ease-in-out infinite;
	}
	.effect-starlight .pk:nth-child(5) {
		animation-delay: 0.4s;
	}
	.effect-starlight .pk:nth-child(9) {
		animation-delay: 0.8s;
	}
	.effect-starlight .pk:nth-child(12) {
		animation-delay: 1.2s;
	}
	/* custom: every key painted its own colour */
	.effect-custom .pk:nth-child(1) {
		background: #ff4444;
	}
	.effect-custom .pk:nth-child(2) {
		background: #ff8800;
	}
	.effect-custom .pk:nth-child(3) {
		background: #ffdd00;
	}
	.effect-custom .pk:nth-child(4) {
		background: #44dd66;
	}
	.effect-custom .pk:nth-child(5) {
		background: #00ccff;
	}
	.effect-custom .pk:nth-child(6) {
		background: #5577ff;
	}
	.effect-custom .pk:nth-child(7) {
		background: #aa66ff;
	}
	.effect-custom .pk:nth-child(8) {
		background: #ff55cc;
	}
	.effect-custom .pk:nth-child(9) {
		background: #ff6b6b;
	}
	.effect-custom .pk:nth-child(10) {
		background: #ffaa00;
	}
	.effect-custom .pk:nth-child(11) {
		background: #22cc88;
	}
	.effect-custom .pk:nth-child(12) {
		background: #4499ff;
	}
	@keyframes ep-hue {
		to {
			filter: hue-rotate(360deg);
		}
	}
	@keyframes ep-breathe {
		0%,
		100% {
			opacity: 1;
		}
		50% {
			opacity: 0.15;
		}
	}
	@keyframes ep-reactive {
		0% {
			opacity: 1;
			box-shadow: 0 0 6px 2px var(--primary);
		}
		70%,
		100% {
			opacity: 0.25;
			box-shadow: 0 0 0 0 transparent;
		}
	}
	@keyframes ep-twinkle {
		0%,
		100% {
			opacity: 1;
		}
		50% {
			opacity: 0.25;
		}
	}
</style>
