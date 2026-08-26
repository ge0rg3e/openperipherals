<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { Switch } from '$lib/components/ui/switch';
	import { toast } from 'svelte-sonner';
	import { CircleAlert, CircleCheck, Download, RefreshCw, X } from '@lucide/svelte';

	interface AppInfo {
		version: string;
		electron: string;
		chrome: string;
		platform: string;
		packaged: boolean;
	}

	interface UpdateState {
		upToDate: boolean | null;
		latestVersion: string | null;
		url?: string;
		error?: string;
	}

	interface DesktopApi {
		getAppInfo: () => Promise<AppInfo>;
		checkForUpdate: () => Promise<UpdateState>;
		downloadUpdate: () => void;
		getLaunchOnBoot: () => Promise<boolean>;
		setLaunchOnBoot: (value: boolean) => Promise<{ ok: boolean; enabled: boolean; error?: string }>;
	}

	let { open = $bindable(false) }: { open?: boolean } = $props();

	let api = $state<DesktopApi | null>(null);
	let info = $state<AppInfo | null>(null);
	let update = $state<UpdateState | null>(null);
	let checking = $state(false);
	let launchOnBoot = $state(false);
	let savingLaunch = $state(false);

	$effect(() => {
		api = (window as unknown as { desktop?: DesktopApi }).desktop ?? null;
	});

	// Load everything each time the dialog opens so "Check on open" stays fresh.
	$effect(() => {
		if (open) void load();
	});

	async function load() {
		if (!api) return;
		info = await api.getAppInfo();
		launchOnBoot = await api.getLaunchOnBoot();
		await checkUpdate();
	}

	async function checkUpdate() {
		if (!api) return;
		checking = true;
		update = null;
		try {
			update = await api.checkForUpdate();
		} finally {
			checking = false;
		}
	}

	async function toggleLaunchOnBoot(next: boolean) {
		if (!api) return;
		savingLaunch = true;
		try {
			const result = await api.setLaunchOnBoot(next);
			launchOnBoot = result.enabled;
			if (result.ok) {
				toast.success(result.enabled ? 'OpenPeripherals will start at login (minimized to the tray).' : 'OpenPeripherals will no longer start at login.');
			} else {
				toast.error(result.error ?? 'Could not change the startup setting.');
			}
		} catch (err) {
			toast.error(err instanceof Error ? err.message : String(err));
		} finally {
			savingLaunch = false;
		}
	}

	function onKeydown(event: KeyboardEvent) {
		if (open && event.key === 'Escape') open = false;
	}
</script>

<svelte:window onkeydown={onKeydown} />

{#if open}
	<!-- Anchored below the title bar (h-9) so window controls and dragging stay usable -->
	<div class="absolute inset-x-0 bottom-0 top-9 z-[100] flex items-center justify-center p-4" role="dialog" aria-modal="true" aria-label="Settings">
		<button class="absolute inset-0 cursor-default bg-black/70 backdrop-blur-sm" aria-label="Close settings" onclick={() => (open = false)}></button>

		<div class="relative flex max-h-[85vh] w-full max-w-md flex-col overflow-hidden rounded-2xl border border-border/60 bg-[#0a0c0e] shadow-2xl shadow-black/60">
			<!-- Header -->
			<div class="flex items-center justify-between border-b border-border/60 px-5 py-3.5">
				<h2 class="text-sm font-semibold tracking-tight">Settings</h2>
				<button
					class="flex h-6 w-6 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
					aria-label="Close"
					onclick={() => (open = false)}
				>
					<X class="h-4 w-4" />
				</button>
			</div>

			{#if !api}
				<p class="px-5 py-8 text-center text-sm text-muted-foreground">Settings are only available in the desktop app.</p>
			{:else}
				<div class="flex flex-col gap-5 overflow-y-auto px-5 py-4">
					<!-- Startup -->
					<section class="flex items-center justify-between gap-3">
						<div class="flex flex-col">
							<span class="text-sm font-medium">Start at login</span>
							<span class="text-xs text-muted-foreground">Launches minimized to the system tray when you sign in</span>
						</div>
						<Switch checked={launchOnBoot} disabled={savingLaunch} onCheckedChange={(c) => toggleLaunchOnBoot(c)} />
					</section>

					<!-- Updates -->
					<section class="flex flex-col gap-2.5 border-t border-border/60 pt-4">
						<div class="flex items-center justify-between gap-3">
							<span class="text-sm font-medium">Updates</span>
							<Button variant="outline" size="sm" class="h-7 gap-1.5 px-2.5 text-xs" disabled={checking} onclick={checkUpdate}>
								<RefreshCw class="h-3 w-3 {checking ? 'animate-spin' : ''}" />
								Check now
							</Button>
						</div>

						{#if checking}
							<p class="flex items-center gap-1.5 text-xs text-muted-foreground">Checking for updates…</p>
						{:else if update?.error}
							<p class="flex items-start gap-1.5 text-xs text-destructive">
								<CircleAlert class="mt-px h-3.5 w-3.5 shrink-0" />
								Could not reach GitHub: {update.error}
							</p>
						{:else if update?.upToDate}
							<p class="flex items-center gap-1.5 text-xs text-emerald-400">
								<CircleCheck class="h-3.5 w-3.5" />
								You're on the latest version (v{info?.version})
							</p>
						{:else if update?.latestVersion}
							<div class="flex items-center justify-between gap-3 rounded-lg border border-primary/40 bg-primary/10 px-3 py-2">
								<p class="text-xs">New version available: <span class="font-semibold">v{update.latestVersion}</span></p>
								<Button size="sm" class="h-7 gap-1.5 px-2.5 text-xs" onclick={() => api?.downloadUpdate()}>
									<Download class="h-3 w-3" />
									Download
								</Button>
							</div>
						{/if}
					</section>

					<!-- About -->
					<section class="flex flex-col gap-2 border-t border-border/60 pt-4">
						<div class="flex items-center gap-2">
							<span class="text-sm font-semibold tracking-tight">OpenPeripherals</span>
							<Badge variant="secondary" class="rounded-full px-1.5 py-0 text-[10px]">beta</Badge>
						</div>
						<dl class="grid grid-cols-[7rem_1fr] gap-x-3 gap-y-1 text-xs">
							<dt class="text-muted-foreground">Version</dt>
							<dd>v{info?.version ?? '—'}</dd>
							<dt class="text-muted-foreground">Electron</dt>
							<dd>{info?.electron ?? '—'}</dd>
							<dt class="text-muted-foreground">Chromium</dt>
							<dd>{info?.chrome ?? '—'}</dd>
							<dt class="text-muted-foreground">Platform</dt>
							<dd class="capitalize">{info?.platform ?? '—'}</dd>
						</dl>
						<a
							class="text-xs text-muted-foreground underline-offset-2 transition-colors hover:text-foreground hover:underline"
							href="https://github.com/ge0rg3e/openperipherals"
							target="_blank"
							rel="noreferrer"
						>
							Source &amp; releases on GitHub
						</a>
					</section>
				</div>
			{/if}
		</div>
	</div>
{/if}
