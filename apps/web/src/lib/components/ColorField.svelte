<script lang="ts">
	import { cn } from '$lib/utils';

	let {
		value = $bindable('#00ff88'),
		label = 'Pick a colour',
		class: className
	}: { value?: string; label?: string; class?: string } = $props();
</script>

<label
	class={cn(
		'flex h-10 w-full cursor-pointer items-center gap-2.5 rounded-lg bg-secondary px-2.5 transition-colors hover:bg-secondary/70',
		className
	)}
>
	<span class="relative size-6 shrink-0 overflow-hidden rounded-md ring-1 ring-white/25" style="background:{value}">
		<input
			type="color"
			bind:value
			aria-label={label}
			class="absolute inset-0 size-full cursor-pointer opacity-0"
		/>
	</span>
	<span class="font-mono text-xs uppercase tracking-wide text-muted-foreground">{value}</span>
</label>
