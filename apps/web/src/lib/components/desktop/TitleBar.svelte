<script lang="ts">
	import { Minus, Square, X, Copy } from '@lucide/svelte';

	interface DesktopApi {
		env: () => Promise<{ isTilingWM: boolean }>;
		minimize: () => void;
		toggleMaximize: () => void;
		close: () => void;
		isMaximized: () => Promise<boolean>;
		onMaximized: (cb: (value: boolean) => void) => void;
	}

	let api = $state<DesktopApi | null>(null);
	let maximized = $state(false);
	let showWindowControls = $state(false);

	$effect(() => {
		const desktop = (window as unknown as { desktop?: DesktopApi }).desktop;
		if (!desktop) return;
		api = desktop;
		void desktop.isMaximized().then((value) => (maximized = value));
		desktop.onMaximized((value) => (maximized = value));
		void desktop.env().then(({ isTilingWM }) => (showWindowControls = !isTilingWM));
	});

	function onDoubleClick() {
		api?.toggleMaximize();
	}
</script>

{#if api}
	<div
		class="titlebar sticky top-0 z-50 flex h-9 w-full shrink-0 items-center justify-between border-b border-border/60 bg-[#0a0c0e] pl-3 select-none"
		ondblclick={onDoubleClick}
	>
		<span class="pointer-events-none text-xs font-semibold tracking-tight text-muted-foreground">OpenPeripherals</span>
		<div class="flex h-full items-stretch">
			{#if showWindowControls}
				<button class="control" aria-label="Minimize window" onclick={() => api?.minimize()}>
					<Minus class="h-3.5 w-3.5" />
				</button>
				<button class="control" aria-label={maximized ? 'Restore window' : 'Maximize window'} onclick={() => api?.toggleMaximize()}>
					{#if maximized}
						<Copy class="h-3 w-3" />
					{:else}
						<Square class="h-3 w-3" />
					{/if}
				</button>
			{/if}
			<button class="control close" aria-label="Close window" onclick={() => api?.close()}>
				<X class="h-4 w-4" />
			</button>
		</div>
	</div>
{/if}

<style>
	.titlebar {
		-webkit-app-region: drag;
		app-region: drag;
	}
	button.control {
		-webkit-app-region: no-drag;
		app-region: no-drag;
		display: flex;
		width: 2.75rem;
		align-items: center;
		justify-content: center;
		color: oklch(0.705 0.015 286.067);
		transition:
			background-color 120ms ease,
			color 120ms ease;
	}
	button.control:hover {
		background-color: oklch(1 0 0 / 8%);
		color: oklch(0.985 0 0);
	}
	button.control.close:hover {
		background-color: #dc2626;
		color: #ffffff;
	}
</style>
