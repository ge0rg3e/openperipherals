<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import { Badge } from '$lib/components/ui/badge';
	import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';

	import { KEYBOARD_DEVICES } from '$lib/razer/devices';
	import { LOGITECH_DEVICES } from '$lib/logitech/devices';
	import { REDRAGON_DEVICES } from '$lib/redragon/devices';
	import { Tabs, TabsList, TabsTrigger, TabsContent } from '$lib/components/ui/tabs';
	import { Select, SelectContent, SelectItem, SelectTrigger } from '$lib/components/ui/select';
	import { Download } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import pkg from '../../package.json' with { type: 'json' };

	function openWebApp() {
		goto('/app');
	}

	const RELEASES_URL = 'https://github.com/ge0rg3e/openperipherals/releases';
	const APP_VERSION = (pkg as { version: string }).version;

	const DESKTOP_DOWNLOADS: Record<string, { label: string; href: string }> = {
		windows: { label: 'Windows (x64)', href: `${RELEASES_URL}/latest/download/OpenPeripherals-${APP_VERSION}-x64-setup.exe` },
		debian: { label: 'Debian / Ubuntu (.deb)', href: `${RELEASES_URL}/latest/download/OpenPeripherals-${APP_VERSION}-amd64-debian.deb` },
		arch: { label: 'Arch Linux (.pacman)', href: `${RELEASES_URL}/latest/download/OpenPeripherals-${APP_VERSION}-x64-arch.pacman` }
	};

	// pacman default: RemoteFileSigLevel=Required → `pacman -U <url>` 404s on .sig for unsigned packages.
	// Short Arch flow: download (triggered automatically) + local install (LocalFileSigLevel=Optional handles unsigned).
	const ARCH_INSTALL_CMD = `sudo pacman -U ~/Downloads/OpenPeripherals-${APP_VERSION}-x64-arch.pacman`;

	let downloadOs = $state('');
	let archCopied = $state(false);
	let showArchInstructions = $state(false);

	async function copyArchCommand() {
		try {
			await navigator.clipboard.writeText(ARCH_INSTALL_CMD);
			archCopied = true;
			showArchInstructions = true;
			setTimeout(() => (archCopied = false), 2000);
		} catch {
			/* clipboard unavailable - still show instructions */
			showArchInstructions = true;
		}
	}

	function downloadDesktop(os: string) {
		downloadOs = '';
		const target = DESKTOP_DOWNLOADS[os];
		if (!target) return;
		if (os === 'arch') {
			window.open(target.href, '_blank', 'noopener');
			copyArchCommand();
			return;
		}
		window.open(target.href, '_blank', 'noopener');
	}

	const linuxSetupCommands = `sudo tee /etc/udev/rules.d/55-openperipherals.rules <<'EOF'
KERNEL=="hidraw*", SUBSYSTEM=="hidraw", ATTRS{idVendor}=="1532", MODE="0666", TAG+="uaccess"
KERNEL=="hidraw*", SUBSYSTEM=="hidraw", ATTRS{idVendor}=="046d", MODE="0666", TAG+="uaccess"
KERNEL=="hidraw*", SUBSYSTEM=="hidraw", ATTRS{idVendor}=="0c45", MODE="0666", TAG+="uaccess"
KERNEL=="hidraw*", SUBSYSTEM=="hidraw", ATTRS{idVendor}=="3710", MODE="0666", TAG+="uaccess"
KERNEL=="hidraw*", SUBSYSTEM=="hidraw", ATTRS{idVendor}=="3554", MODE="0666", TAG+="uaccess"
EOF
sudo udevadm control --reload-rules
sudo udevadm trigger`;

	let copied = $state(false);

	async function copyLinuxCommands() {
		try {
			await navigator.clipboard.writeText(linuxSetupCommands);
			copied = true;
			setTimeout(() => (copied = false), 2000);
		} catch {
			/* clipboard unavailable */
		}
	}

	const family = (name: string): string => {
		if (name.startsWith('BlackWidow')) return 'BlackWidow';
		if (name.startsWith('Huntsman')) return 'Huntsman';
		if (name.startsWith('DeathStalker')) return 'DeathStalker';
		return 'Other';
	};

	const familyOrder = ['BlackWidow', 'Huntsman', 'DeathStalker', 'Other'];
	const razerGroups = familyOrder
		.map((label) => ({
			label,
			items: KEYBOARD_DEVICES.filter((d) => family(d.name) === label)
		}))
		.filter((g) => g.items.length > 0);

	const logiFamily = (name: string): string => {
		if (name.includes('G213')) return 'G213 Prodigy';
		if (name.includes('G512')) return 'G512';
		if (name.includes('G610')) return 'G610';
		if (name.includes('G810')) return 'G810 Orion Spectrum';
		if (name.includes('G910')) return 'G910 Orion';
		if (name.includes('G813') || name.includes('G815')) return 'G813 / G815';
		if (name.includes('G915')) return 'G915 Lightspeed';
		if (name.includes('G Pro')) return 'G Pro';
		return 'Other';
	};

	const logiOrder = ['G213 Prodigy', 'G512', 'G610', 'G810 Orion Spectrum', 'G910 Orion', 'G Pro', 'G813 / G815', 'G915 Lightspeed', 'Other'];
	const logiGroups = logiOrder
		.map((label) => ({
			label,
			items: LOGITECH_DEVICES.filter((d) => logiFamily(d.name) === label)
		}))
		.filter((g) => g.items.length > 0);

	const redragonFamily = (name: string): string => {
		if (name.includes('Magic Wand') || name.includes('K587')) return 'K587 Magic Wand';
		if (name.includes('K552')) return 'K552 Kumara';
		if (name.includes('K550')) return 'K550 Yama';
		if (name.includes('K512')) return 'K512';
		if (name.includes('Surara') || name.includes('K582')) return 'Surara K582';
		if (name.includes('K589')) return 'K589 Shrapnel';
		if (name.includes('Mitra')) return 'K551 Mitra';
		return 'Other';
	};

	const redragonOrder = ['K587 Magic Wand', 'K556 Devarajas', 'K552 Kumara', 'K550 Yama', 'K589 Shrapnel', 'Surara K582', 'K512', 'Other'];
	const redragonGroups = redragonOrder
		.map((label) => ({
			label,
			items: REDRAGON_DEVICES.filter((d) => redragonFamily(d.name) === label)
		}))
		.filter((g) => g.items.length > 0);

	// Supported brands. Each entry resolves its own device groups; the Connect
	// button lists every supported keyboard from all brands in one picker.
	const categories: Array<{ id: string; label: string; groups: Array<{ label: string; items: Array<{ name: string; image?: string }> }> }> = [
		{
			id: 'razer',
			label: 'Razer',
			groups: razerGroups
		},
		{
			id: 'logitech',
			label: 'Logitech',
			groups: logiGroups
		},
		{
			id: 'redragon',
			label: 'Redragon',
			groups: redragonGroups
		}
	];

	let activeCategory = $state(categories[0].id);
	const activeCat = $derived(categories.find((c) => c.id === activeCategory)!);
	const deviceCount = $derived(activeCat.groups.reduce((n, g) => n + g.items.length, 0));

	const features = [
		{
			title: 'Hardware effects',
			description: 'Static, wave, spectrum, reactive, breathing and starlight, rendered by the device itself.'
		},
		{
			title: 'Web or desktop',
			description: 'Use it right in your browser or install the desktop app - same features either way, always local over WebHID. No drivers, no Synapse.'
		},
		{
			title: 'Privacy first',
			description: 'Everything stays on your machine and your device. No telemetry no tracking.'
		}
	];
</script>

<svelte:head>
	<title>OpenPeripherals: control your Razer, Logitech & Redragon RGB devices in the browser</title>
	<meta
		name="description"
		content="OpenPeripherals is a free, open-source WebHID app that controls your Razer Chroma, Logitech G-series and Redragon device lighting straight from the browser. No drivers, no Synapse, nothing to install."
	/>
	<meta name="robots" content="index, follow" />
	<meta name="theme-color" content="#0a0c0e" />
	<link rel="canonical" href="https://openperipherals.vercel.app/" />

	<meta property="og:type" content="website" />
	<meta property="og:site_name" content="OpenPeripherals" />
	<meta property="og:title" content="OpenPeripherals control your Razer, Logitech & Redragon RGB devices in the browser" />
	<meta property="og:description" content="Free, open-source WebHID app for Razer Chroma, Logitech G-series and Redragon RGB devices. No drivers, no Synapse, nothing to install." />
	<meta property="og:url" content="https://openperipherals.vercel.app/" />
	<meta property="og:image" content="https://openperipherals.vercel.app/og.png" />
	<meta property="og:image:width" content="1200" />
	<meta property="og:image:height" content="630" />
	<meta property="og:image:alt" content="OpenPeripherals - device lighting in your browser" />

	<meta name="twitter:card" content="summary_large_image" />
	<meta name="twitter:title" content="OpenPeripherals control your Razer, Logitech & Redragon RGB devices in the browser" />
	<meta name="twitter:description" content="Free, open-source WebHID app for Razer Chroma, Logitech G-series and Redragon RGB devices. No drivers, no Synapse, nothing to install." />
	<meta name="twitter:image" content="https://openperipherals.vercel.app/og.png" />

	<script type="application/ld+json">
		{
			"@context": "https://schema.org",
			"@type": "WebApplication",
			"name": "OpenPeripherals",
			"alternateName": "OpenPeripherals web app",
			"url": "https://openperipherals.vercel.app/",
			"applicationCategory": "UtilityApplication",
			"operatingSystem": "Any (Chromium browsers)",
			"browserRequirements": "WebHID API in a secure context",
			"description": "Free, open-source browser app that controls Razer Chroma, Logitech G-series and Redragon device lighting over WebHID. No drivers and no proprietary software required.",
			"isAccessibleForFree": true,
			"offers": { "@type": "Offer", "price": "0", "priceCurrency": "USD" },
			"featureList": [
				"Static, wave, spectrum, reactive, breathing and starlight hardware effects",
				"Per-key custom lighting",
				"Works with Razer Chroma, Logitech G-series and Redragon devices",
				"No install, no drivers, no vendor software"
			],
			"author": { "@type": "Person", "name": "Ge0rg3e", "url": "https://github.com/ge0rg3e" },
			"publisher": { "@type": "Organization", "name": "OpenPeripherals", "url": "https://openperipherals.vercel.app/" }
		}
	</script>
</svelte:head>

<div class="flex min-h-screen flex-col bg-background">
	<!-- Header -->
	<header class="sticky top-0 z-10 border-b border-border/60 bg-background/80 backdrop-blur">
		<div class="mx-auto flex w-full max-w-6xl items-center justify-between py-3">
			<a href="/" class="flex items-center gap-2 font-semibold">
				OpenPeripherals
				<Badge variant="secondary" class="rounded-full px-1.5 py-0 text-[10px]">beta</Badge>
			</a>
			<div class="flex items-center gap-3">
				<Button href="#troubleshooting" variant="ghost" size="sm">Troubleshooting</Button>
				<Button variant="outline" size="sm" onclick={openWebApp}>Open Web App</Button>
			</div>
		</div>
	</header>

	<main class="flex-1">
		<!-- Hero -->
		<section class="mx-auto w-full max-w-4xl px-4 pt-16 pb-12 text-center sm:pt-24">
			<h1 class="mt-6 text-4xl font-bold tracking-tight text-balance sm:text-6xl">OpenPeripherals</h1>

			<p class="mx-auto mt-4 max-w-1xl text-base text-muted-foreground text-pretty sm:text-lg">
				A free, open-source WebHID app that controls your devices without any proprietary software.<br />Connect your device, change its settings, and close the web app when you're done.
			</p>

			<div class="mt-8 flex flex-wrap items-center justify-center gap-3">
				<Button size="lg" onclick={openWebApp}>Open Web App</Button>
				<Button href="#supported" variant="outline" size="lg">Browse Devices</Button>
			</div>
			<div class="mt-3 flex justify-center">
				<Select type="single" bind:value={downloadOs} onValueChange={downloadDesktop}>
					<SelectTrigger
						size="default"
						class="h-9 rounded-lg border-border/60 bg-secondary px-4 text-sm font-medium text-secondary-foreground hover:bg-secondary/70 aria-expanded:bg-secondary/70"
					>
						<span class="flex items-center gap-1.5"><Download class="h-4 w-4" />Download desktop app</span>
					</SelectTrigger>
					<SelectContent>
						{#each Object.entries(DESKTOP_DOWNLOADS) as [os, d] (os)}
							<SelectItem value={os} label={d.label} />
						{/each}
					</SelectContent>
				</Select>
			</div>
			<p class="mt-3 text-xs text-muted-foreground">
				Desktop builds are served from
				<a href={RELEASES_URL} target="_blank" rel="noopener noreferrer" class="underline underline-offset-2 hover:text-foreground">GitHub Releases</a>.
			</p>
			{#if showArchInstructions}
				<div class="mx-auto mt-4 max-w-xl rounded-xl border border-border/60 bg-card p-4 text-left">
					<h3 class="text-sm font-semibold">Arch Linux</h3>
					<div class="relative mt-3">
						<button
							type="button"
							class="absolute right-1.5 top-1.5 z-10 rounded bg-sky-500/20 px-2 py-0.5 text-[10px] font-medium text-sky-200 transition hover:bg-sky-500/40 focus:outline-none focus-visible:ring-1 focus-visible:ring-sky-300"
							onclick={copyArchCommand}
						>
							{archCopied ? 'Copied!' : 'Copy'}
						</button>
						<pre class="overflow-x-auto rounded-lg border border-border/60 bg-background px-4 py-3 pr-16 font-mono text-xs leading-relaxed text-sky-100">{ARCH_INSTALL_CMD}</pre>
					</div>
					<p class="mt-2 text-xs text-muted-foreground">download starts automatically → run above in terminal → open <span class="font-mono text-foreground">OpenPeripherals</span> in app launcher</p>
				</div>
			{/if}
		</section>

		<!-- Features -->
		<section class="mx-auto w-full max-w-4xl px-4 pb-16">
			<div class="grid gap-4 sm:grid-cols-3">
				{#each features as f}
					<Card class="border-border/60 bg-card">
						<CardHeader class="pb-2">
							<CardTitle class="text-sm">{f.title}</CardTitle>
						</CardHeader>
						<CardContent>
							<p class="text-xs leading-relaxed text-muted-foreground">
								{f.description}
							</p>
						</CardContent>
					</Card>
				{/each}
			</div>
		</section>

		<!-- Supported devices -->
		<section id="supported" class="border-t border-border/60 bg-secondary/40">
			<div class="mx-auto w-full max-w-5xl px-4 py-16">
				<div class="flex flex-col items-start justify-between gap-3 sm:flex-row sm:items-end">
					<div>
						<h2 class="text-2xl font-bold tracking-tight">Supported devices</h2>
						<p class="mt-2 max-w-xl text-sm text-muted-foreground">
							{deviceCount}
							{activeCat.label} devices supported.
						</p>
					</div>
					<Badge variant="secondary" class="rounded-full">
						{deviceCount} devices
					</Badge>
				</div>

				<Tabs bind:value={activeCategory} class="mt-8">
					<TabsList>
						{#each categories as cat}
							<TabsTrigger value={cat.id}>{cat.label}</TabsTrigger>
						{/each}
					</TabsList>

					{#each categories as cat}
						<TabsContent value={cat.id}>
							{#each cat.groups as group}
								<div class="mt-10">
									<div class="flex items-center gap-2">
										<h3 class="text-xs font-semibold tracking-wider text-muted-foreground uppercase">
											{group.label}
										</h3>
										<span class="rounded-full bg-secondary px-2 py-0.5 text-[10px] font-medium text-muted-foreground">{group.items.length}</span>
									</div>
									<div class="mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
										{#each group.items as d}
											<div class="flex h-[180px] flex-col overflow-hidden rounded-xl border border-border/60 bg-card transition-colors hover:border-border">
												<div class="min-h-0 flex-1 bg-secondary/50">
													{#if d.image}
														<img src={d.image} alt={d.name} loading="lazy" class="h-full w-full object-contain p-1" />
													{:else}
														<div class="flex h-full w-full items-center justify-center text-3xl text-muted-foreground/30"></div>
													{/if}
												</div>
												<div class="shrink-0 truncate border-t border-border/60 px-4 py-2.5 text-sm font-medium">
													{d.name}
												</div>
											</div>
										{/each}
									</div>
								</div>
							{/each}
						</TabsContent>
					{/each}
				</Tabs>
			</div>
		</section>

		<!-- Troubleshooting -->
		<section id="troubleshooting" class="border-t border-border/60 bg-secondary/40">
			<div class="mx-auto w-full max-w-4xl px-4 py-16">
				<h2 class="text-2xl font-bold tracking-tight">Troubleshooting</h2>
				<p class="mt-2 max-w-2xl text-sm text-muted-foreground">Why WebHID may not control your lighting on Linux.</p>

				<div class="mt-8 rounded-xl border border-border/60 bg-card p-6">
					<h3 class="text-sm font-semibold">Linux needs a one-time permission fix</h3>
					<p class="mt-3 max-w-2xl text-sm leading-relaxed text-muted-foreground">
						Browsers talk to your device through the WebHID API, which on Linux is guarded by the OS.<br />By default your device is only readable, not writable, so the app connects but
						the lighting controls do nothing until a small udev rule grants access.
					</p>
					<p class="mt-3 max-w-2xl text-sm leading-relaxed text-muted-foreground">
						If "Connect Device" looks like it worked but nothing changes, run these three lines in a terminal, then unplug and replug your device (or reboot):
					</p>
					<div class="relative mt-4">
						<button
							type="button"
							class="absolute right-1.5 top-1.5 z-10 rounded bg-amber-500/20 px-2 py-0.5 text-[10px] font-medium text-amber-200 transition hover:bg-amber-500/40 focus:outline-none focus-visible:ring-1 focus-visible:ring-amber-300"
							onclick={copyLinuxCommands}
						>
							{copied ? 'Copied!' : 'Copy'}
						</button>
						<pre class="overflow-x-auto rounded-lg border border-border/60 bg-background px-4 py-3 font-mono text-xs leading-relaxed text-amber-100">{linuxSetupCommands}</pre>
					</div>
					<a
						href="https://developer.chrome.com/docs/capabilities/hid#:~:text=On%20most%20Linux%20systems%2C%20HID%20devices%20are%20mapped%20with%20read%2Donly%20permissions%20by%20default.%20To%20allow%20Chrome%20to%20open%20an%20HID%20device%2C%20you%20will%20need%20to%20add%20a%20new%20udev%20rule.%20Create%20a%20file%20at%20/etc/udev/rules.d/50%2Dyourdevicename.rules%20with%20the%20following%20content%3A"
						target="_blank"
						rel="noopener noreferrer"
						class="mt-4 inline-flex items-center gap-1 rounded bg-amber-500/20 px-2.5 py-1.5 text-xs font-medium text-amber-200 transition hover:bg-amber-500/40 focus:outline-none focus-visible:ring-1 focus-visible:ring-amber-300"
					>
						Read on Chrome Docs
					</a>
				</div>
			</div>
		</section>
	</main>

	<footer class="border-t border-border/60">
		<div class="mx-auto flex w-full max-w-6xl items-center justify-between py-6 text-xs text-muted-foreground">
			<span>OpenPeripherals - {new Date().getFullYear()}</span>

			<div class="flex items-center gap-3">
				<Button variant="link" size="sm" class="px-0" target="_blank" href="https://github.com/ge0rg3e/openperipherals">GitHub</Button>

				<Button variant="link" size="sm" class="px-0" target="_blank" href="https://x.com/ge0rg3e_dev">Follow @ge0rg3e_dev on X</Button>
			</div>
		</div>
	</footer>
</div>
