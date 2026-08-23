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

<div class="flex max-h-72 flex-col gap-1 overflow-y-auto pr-0.5">
	{#each options as option (option.value)}
		{@const pk = previewKind(option.value)}
		<button
			type="button"
			disabled={disabled}
			aria-pressed={selected === option.value}
			onclick={() => onSelect(option.value)}
			class={cn(
				'flex w-full items-center gap-3 rounded-xl border px-2 py-1.5 text-left transition-colors',
				selected === option.value
					? 'border-primary/60 bg-primary/10'
					: 'border-transparent bg-secondary/50 hover:bg-secondary'
			)}
		>
			<span class="effect-preview effect-{pk}" aria-hidden="true"></span>
			<span class="flex min-w-0 flex-col">
				<span class="text-sm font-medium">{option.label}</span>
				<span class="truncate text-xs text-muted-foreground">{descriptions[pk]}</span>
			</span>
			{#if selected === option.value}
				<span class="ml-auto size-2 shrink-0 rounded-full bg-primary"></span>
			{/if}
		</button>
	{/each}
</div>

<style>
	.effect-preview {
		position: relative;
		flex: none;
		width: 44px;
		height: 30px;
		border-radius: 8px;
		background: #0b0d0f;
		box-shadow: inset 0 0 0 1px rgb(255 255 255 / 0.12);
		overflow: hidden;
	}
	.effect-preview::after {
		content: '';
		position: absolute;
		inset: 5px;
		border-radius: 5px;
		background: var(--primary);
	}
	.effect-off::after {
		display: none;
	}
	.effect-wave::after {
		inset: 0;
		border-radius: 0;
		background: linear-gradient(115deg, #ff0040, #ff8800, #ffee00, #33ff33, #00ccff, #8866ff, #ff44cc, #ff0040);
		background-size: 220% 100%;
		animation: ep-slide 2.4s linear infinite;
	}
	.effect-spectrum::after,
	.effect-wheel::after {
		background: #ff2244;
		animation: ep-hue 2.4s linear infinite;
	}
	.effect-breathing::after {
		animation: ep-breathe 1.8s ease-in-out infinite;
	}
	.effect-reactive::after {
		inset: 9px;
		border-radius: 999px;
		animation: ep-react 1.6s ease-out infinite;
	}
	.effect-starlight::after {
		background:
			radial-gradient(circle at 25% 35%, var(--primary) 0 2px, transparent 3px),
			radial-gradient(circle at 70% 65%, var(--primary) 0 2px, transparent 3px),
			radial-gradient(circle at 55% 20%, var(--primary) 0 1.5px, transparent 2.5px);
		animation: ep-twinkle 1.6s ease-in-out infinite;
	}
	.effect-custom::after {
		background: conic-gradient(var(--primary) 0 25%, #00ccff 0 50%, #ffee00 0 75%, #ff44cc 0);
	}
	@keyframes ep-slide {
		to {
			background-position: 220% 0;
		}
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
	@keyframes ep-react {
		0% {
			box-shadow: 0 0 0 0 var(--primary);
			opacity: 1;
		}
		100% {
			box-shadow: 0 0 0 10px transparent;
			opacity: 0.2;
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
