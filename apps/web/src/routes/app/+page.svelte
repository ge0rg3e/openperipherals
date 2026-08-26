<script lang="ts">
	import { sessions, activeSessionId, error, addDevice, removeSession, detectConnectedDevices, bindDeviceHotplug, isDesktopApp } from '$lib/store';
	import KeyboardWorkspace from '$lib/components/KeyboardWorkspace.svelte';
	import MousePanel from '$lib/components/MousePanel.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Badge } from '$lib/components/ui/badge';
	import { toast } from 'svelte-sonner';
	import { Plus, X, Keyboard as KeyboardIcon, Mouse as MouseIcon } from '@lucide/svelte';

	const active = $derived($sessions.find((s) => s.id === $activeSessionId) ?? null);

	function selectSession(id: string) {
		activeSessionId.set(id);
	}

	// Reconnect anything this origin may already access (in the desktop shell
	// that is every plugged-in device) and follow hotplug events. Browsers
	// additionally need the one-time chooser gesture below.
	$effect(() => {
		bindDeviceHotplug();
		void detectConnectedDevices();
	});

	$effect(() => {
		if ($error && !$error.includes('user gesture')) toast.error($error);
	});
</script>

<svelte:head>
	<title>OpenPeripherals</title>
	<meta
		name="description"
		content="Control your Razer Chroma, Logitech G-series or Redragon RGB device from the browser with OpenPeripherals. Apply effects, set brightness, and manage per-key lighting over WebHID."
	/>
	<meta name="robots" content="noindex, nofollow" />
	<meta name="theme-color" content="#0a0c0e" />
</svelte:head>

<div class="min-h-full w-full py-6 px-2 sm:px-3 lg:px-4">
	<div class="mx-auto flex max-w-7xl flex-col gap-6">
		<!-- Header (web only - the desktop shell has its own title bar) -->
		{#if !isDesktopApp()}
			<header class="flex items-center justify-between gap-3">
				<h1 class="flex items-center gap-2 text-lg font-bold tracking-tight">
					OpenPeripherals
					<Badge variant="secondary" class="rounded-full px-1.5 py-0 text-[10px]">beta</Badge>
				</h1>
			</header>
		{/if}

		{#if $sessions.length > 0}
			<!-- Device rail -->
			<nav class="flex flex-wrap items-center gap-2" aria-label="Connected devices">
				{#each $sessions as session (session.id)}
					<div
						class={`group flex items-center gap-2 rounded-full border py-1 pl-2 pr-1 transition-colors ${
							session.id === $activeSessionId
								? 'border-primary/60 bg-primary/10 text-foreground'
								: 'border-border/60 bg-secondary/40 text-muted-foreground hover:bg-secondary/70'
						}`}
					>
						<button type="button" class="flex items-center gap-2" onclick={() => selectSession(session.id)}>
							<span class="flex h-5 w-5 items-center justify-center rounded-full bg-background/80 ring-1 ring-border/60">
								{#if session.kind === 'keyboard'}
									<KeyboardIcon class="h-3 w-3" />
								{:else}
									<MouseIcon class="h-3 w-3" />
								{/if}
							</span>
							<span class="max-w-[14rem] truncate text-xs font-medium {session.id === $activeSessionId ? 'text-foreground' : ''}">
								{session.name}
							</span>
						</button>
						<button
							type="button"
							aria-label={`Disconnect ${session.name}`}
							class="flex h-5 w-5 items-center justify-center rounded-full text-muted-foreground/70 transition-colors hover:bg-destructive/15 hover:text-destructive"
							onclick={() => removeSession(session.id)}
						>
							<X class="h-3 w-3" />
						</button>
					</div>
				{/each}
			</nav>
		{:else}
			<!-- Empty state -->
			<div class="flex flex-col items-center gap-4 rounded-2xl border border-border/60 bg-gradient-to-b from-zinc-950 via-[#0a0c0e] to-black px-6 py-16 text-center shadow-2xl shadow-black/50">
				<div class="flex items-center gap-3 text-muted-foreground">
					<KeyboardIcon class="h-8 w-8" />
					<Plus class="h-4 w-4" />
					<MouseIcon class="h-8 w-8" />
				</div>
				<div class="flex flex-col gap-1">
					<p class="text-base font-semibold">Your workspace is empty</p>
					<p class="max-w-md text-sm text-muted-foreground">
						{#if isDesktopApp()}
							Plug in a supported device and it will connect automatically.
						{:else}
							Grant access once - your devices reconnect automatically from then on. Everything runs locally over WebHID.
						{/if}
					</p>
					{#if !isDesktopApp()}
						<Button size="sm" onclick={() => addDevice()}>
							<Plus class="mr-1 h-3.5 w-3.5" /> Connect device
						</Button>
					{/if}
				</div>
			</div>
		{/if}

		<!-- Workspace for the selected device -->
		{#if active}
			{#if active.kind === 'keyboard'}
				<KeyboardWorkspace sessionId={active.id} />
			{:else}
				<MousePanel sessionId={active.id} />
			{/if}
		{/if}
	</div>
</div>

<style>
	:global(:root) {
		color-scheme: dark;
	}
</style>
