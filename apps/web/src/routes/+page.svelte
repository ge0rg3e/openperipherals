<script lang="ts">
	import { Tabs, TabsList, TabsTrigger, TabsContent } from '$lib/components/ui/tabs';
	import { Select, SelectContent, SelectItem, SelectTrigger } from '$lib/components/ui/select';
	import { Download } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import pkg from '@root/package.json' with { type: 'json' };

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

	const ARCH_INSTALL_CMD = `sudo pacman -U ./OpenPeripherals-${APP_VERSION}-x64-arch.pacman`;

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
			showArchInstructions = true;
		}
	}

	function downloadDesktop(os: string) {
		downloadOs = '';
		const target = DESKTOP_DOWNLOADS[os];
		if (!target) return;
		if (os === 'arch') {
			window.open(target.href, '_blank', 'noopener');
			showArchInstructions = true;
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

	import { KEYBOARD_DEVICES } from '$lib/razer/devices';
	import { LOGITECH_DEVICES } from '$lib/logitech/devices';
	import { REDRAGON_DEVICES } from '$lib/redragon/devices';

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

	const categories: Array<{ id: string; label: string; groups: Array<{ label: string; items: Array<{ name: string; image?: string }> }> }> = [
		{ id: 'razer', label: 'Razer', groups: razerGroups },
		{ id: 'logitech', label: 'Logitech', groups: logiGroups },
		{ id: 'redragon', label: 'Redragon', groups: redragonGroups }
	];

	let activeCategory = $state(categories[0].id);
	const activeCat = $derived(categories.find((c) => c.id === activeCategory)!);
	const deviceCount = $derived(activeCat.groups.reduce((n, g) => n + g.items.length, 0));

	let mobileMenuOpen = $state(false);

	const features = [
		{
			title: 'Hardware effects',
			description: 'Static, wave, spectrum, reactive, breathing and starlight rendered by the device itself.',
			tag: '6 effects',
			accent: 'from-[#ff3b2f]/20 via-[#ff6b2f]/10 to-transparent'
		},
		{
			title: 'Web or desktop',
			description: 'Use it in your browser or install the app. Same features, always local over WebHID. No drivers, no Synapse.',
			tag: 'WebHID',
			accent: 'from-[#737373]/20 via-white/5 to-transparent'
		},
		{
			title: 'Privacy first',
			description: 'Everything stays on your machine and your device. No telemetry, no tracking, no account.',
			tag: '100% local',
			accent: 'from-[#9dff00]/15 via-white/5 to-transparent'
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
	<meta name="theme-color" content="#000000" />
	<link rel="canonical" href="https://openperipherals.vercel.app/" />
	<meta property="og:type" content="website" />
	<meta property="og:site_name" content="OpenPeripherals" />
	<meta property="og:title" content="OpenPeripherals control your Razer, Logitech & Redragon RGB devices in the browser" />
	<meta property="og:description" content="Free, open-source WebHID app for Razer Chroma, Logitech G-series and Redragon RGB devices. No drivers, no Synapse, nothing to install." />
	<meta property="og:url" content="https://openperipherals.vercel.app/" />
	<meta property="og:image" content="https://openperipherals.vercel.app/og.png" />
	<meta property="og:image:width" content="1200" />
	<meta property="og:image:height" content="630" />
	<meta name="twitter:card" content="summary_large_image" />
	<script type="application/ld+json">
		{
			"@context": "https://schema.org",
			"@type": "WebApplication",
			"name": "OpenPeripherals",
			"url": "https://openperipherals.vercel.app/",
			"applicationCategory": "UtilityApplication",
			"operatingSystem": "Any (Chromium browsers)",
			"isAccessibleForFree": true,
			"offers": { "@type": "Offer", "price": "0", "priceCurrency": "USD" }
		}
	</script>
</svelte:head>

<div class="relative flex min-h-screen flex-col bg-black text-white selection:bg-white selection:text-black">
	<!-- subtle ambient gradients like Raycast -->
	<div class="pointer-events-none absolute inset-0 overflow-hidden">
		<div class="absolute left-1/2 top-[-180px] h-[520px] w-[980px] -translate-x-1/2 rounded-full bg-[radial-gradient(ellipse_at_center,_rgba(255,255,255,0.07),transparent_60%)] blur-[1px]"></div>
		<div class="absolute left-1/2 top-[220px] h-[420px] w-[720px] -translate-x-1/2 rounded-full bg-[radial-gradient(ellipse_at_center,_rgba(255,59,47,0.06),transparent_65%)]"></div>
	</div>

	<!-- Header - floating pill, geometrically symmetric -->
	<header class="sticky top-3.5 z-20 mx-auto w-[calc(100%-24px)] max-w-[1160px]">
		<div
			class="flex items-center justify-between gap-2 rounded-[14px] border border-white/[0.07] bg-[#0f0f10]/75 px-1.5 py-1.5 shadow-[0_8px_32px_rgba(0,0,0,0.6),0_0_0_1px_rgba(255,255,255,0.04)_inset] backdrop-blur-2xl sm:gap-4 sm:rounded-2xl sm:px-2.5"
		>
			<a href="/" class="flex min-w-0 shrink-0 items-center gap-2 pl-1.5 sm:pl-2 pr-1">
				<span class="text-[13px] font-[600] tracking-tight sm:text-[13.5px]">OpenPeripherals</span>
				<span class="hidden rounded-full border border-white/10 bg-white/5 px-1.5 py-0.5 text-[10px] font-medium leading-none text-white/60 sm:inline-flex">beta</span>
			</a>

			<nav class="hidden flex-1 items-center justify-center gap-0.5 lg:flex">
				<a href="#supported" class="rounded-full px-3 py-1.5 text-[13px] font-[450] text-white/45 transition hover:bg-white/[0.06] hover:text-white/90">Devices</a>
				<a href="#troubleshooting" class="rounded-full px-3 py-1.5 text-[13px] font-[450] text-white/45 transition hover:bg-white/[0.06] hover:text-white/90">Troubleshooting</a>
				<a
					href="https://github.com/ge0rg3e/openperipherals"
					target="_blank"
					rel="noopener"
					class="rounded-full px-3 py-1.5 text-[13px] font-[450] text-white/45 transition hover:bg-white/[0.06] hover:text-white/90">GitHub</a
				>
			</nav>

			<div class="flex shrink-0 items-center justify-end gap-1.5">
				<div class="hidden lg:flex">
					<Select type="single" bind:value={downloadOs} onValueChange={downloadDesktop}>
						<SelectTrigger
							size="default"
							class="h-8 gap-1.5 rounded-full border-white/10 bg-white/[0.06] px-3 text-[13px] font-medium text-white/80 hover:bg-white/10 hover:text-white data-[placeholder]:text-white/60 [&_svg]:opacity-60"
						>
							<Download class="h-3.5 w-3.5" />
							<span>Download</span>
						</SelectTrigger>
						<SelectContent class="rounded-xl border-white/10 bg-[#121212]">
							{#each Object.entries(DESKTOP_DOWNLOADS) as [os, d] (os)}
								<SelectItem value={os} label={d.label} class="text-sm" />
							{/each}
						</SelectContent>
					</Select>
				</div>
				<button
					onclick={openWebApp}
					class="group inline-flex h-8 items-center gap-1 rounded-full bg-white px-2.5 py-0 text-[13px] font-[550] tracking-tight text-black shadow-[0_1px_2px_rgba(0,0,0,0.35)] transition hover:bg-white/90 sm:px-3"
				>
					<span class="pl-0.5">Open<span class="hidden sm:inline"> Web App</span></span>
					<span class="flex h-5 w-5 items-center justify-center rounded-full bg-black/[0.07] text-black/40 transition group-hover:bg-black/10">
						<svg width="10" height="10" viewBox="0 0 12 12" fill="none"
							><path d="M4 3l4 3-4 3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" /></svg
						>
					</span>
				</button>
				<button
					class="inline-flex h-8 w-8 items-center justify-center rounded-full border border-white/10 bg-white/[0.06] text-white/70 hover:bg-white/10 hover:text-white lg:hidden"
					aria-label="Toggle menu"
					aria-expanded={mobileMenuOpen}
					onclick={() => (mobileMenuOpen = !mobileMenuOpen)}
				>
					<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M4 7h16M4 12h16M4 17h16" /></svg>
				</button>
			</div>
		</div>
		{#if mobileMenuOpen}
			<div class="mt-2 rounded-2xl border border-white/[0.07] bg-[#0f0f10]/95 p-1.5 shadow-[0_16px_40px_rgba(0,0,0,0.6)] backdrop-blur-2xl lg:hidden">
				<a
					href="#supported"
					onclick={() => (mobileMenuOpen = false)}
					class="flex items-center rounded-xl px-3 py-2.5 text-[13px] font-medium text-white/70 hover:bg-white/[0.06] hover:text-white">Devices</a
				>
				<a
					href="#troubleshooting"
					onclick={() => (mobileMenuOpen = false)}
					class="flex items-center rounded-xl px-3 py-2.5 text-[13px] font-medium text-white/70 hover:bg-white/[0.06] hover:text-white">Troubleshooting</a
				>
				<a
					href="https://github.com/ge0rg3e/openperipherals"
					target="_blank"
					rel="noopener"
					onclick={() => (mobileMenuOpen = false)}
					class="flex items-center rounded-xl px-3 py-2.5 text-[13px] font-medium text-white/70 hover:bg-white/[0.06] hover:text-white">GitHub</a
				>
				<div class="my-1 h-px bg-white/[0.06]"></div>
				<div class="px-1 pb-1 pt-1">
					<Select
						type="single"
						bind:value={downloadOs}
						onValueChange={(v) => {
							downloadDesktop(v);
							mobileMenuOpen = false;
						}}
					>
						<SelectTrigger size="default" class="w-full justify-between rounded-xl border-white/10 bg-white/[0.06] px-3 py-2 text-[13px] font-medium">
							<span class="flex items-center gap-1.5"><Download class="h-3.5 w-3.5" />Download</span>
						</SelectTrigger>
						<SelectContent class="rounded-xl border-white/10 bg-[#121212]">
							{#each Object.entries(DESKTOP_DOWNLOADS) as [os, d] (os)}
								<SelectItem value={os} label={d.label} class="text-sm" />
							{/each}
						</SelectContent>
					</Select>
				</div>
			</div>
		{/if}
	</header>

	<main class="relative flex-1">
		<!-- Hero -->
		<section class="relative mx-auto flex w-full max-w-[720px] flex-col items-center px-4 pb-12 pt-14 text-center sm:pt-[80px]">
			<h1 class="mt-2 max-w-[640px] text-balance text-[34px] font-[680] leading-[0.92] tracking-[-0.045em] sm:text-[50px] md:text-[54px]">
				Control your RGB
				<span class="bg-gradient-to-b from-white to-white/65 bg-clip-text text-transparent">without the bloat.</span>
			</h1>

			<p class="mx-auto mt-5 max-w-[560px] text-pretty text-[14px] leading-[1.6] text-white/[0.42] sm:text-[15px]">
				A free, open-source WebHID app that controls your devices without any proprietary software. Connect, change settings, close the tab. Done.
			</p>

			<div class="mt-8 flex flex-wrap items-center justify-center gap-2.5">
				<button
					onclick={openWebApp}
					class="group inline-flex h-[37px] items-center gap-0 overflow-hidden rounded-full bg-white p-1 pr-1 text-sm font-medium text-black shadow-[0_4px_24px_rgba(255,255,255,0.12),0_1px_2px_rgba(0,0,0,0.35)] transition hover:bg-white/90 active:scale-[0.98]"
				>
					<span class="px-3.5 text-[13.5px] font-[600] tracking-tight">Open Web App</span>
					<span class="flex h-6 w-6 items-center justify-center rounded-full bg-black/[0.07] text-black/45 transition group-hover:bg-black/10">
						<svg width="12" height="12" viewBox="0 0 12 12" fill="none"
							><path d="M4 3l4 3-4 3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" /></svg
						>
					</span>
				</button>

				<a
					href="#supported"
					class="inline-flex h-[37px] items-center gap-1.5 rounded-full border border-white/[0.08] bg-white/[0.05] px-4.5 text-[13.5px] font-[500] text-white/85 shadow-[0_1px_12px_rgba(0,0,0,0.4)] backdrop-blur transition hover:bg-white/[0.08] hover:text-white"
				>
					<svg width="13" height="13" viewBox="0 0 24 24" fill="none" class="opacity-50"
						><circle cx="12" cy="12" r="8.5" stroke="currentColor" stroke-width="1.5" /><path
							d="M10 8.5l4 3.5-4 3.5"
							stroke="currentColor"
							stroke-width="1.4"
							stroke-linecap="round"
							stroke-linejoin="round"
						/></svg
					>
					Browse devices
				</a>
			</div>

			<div class="mt-4 flex items-center justify-center">
				<Select type="single" bind:value={downloadOs} onValueChange={downloadDesktop}>
					<SelectTrigger size="default" class="h-7 rounded-full border-white/[0.07] bg-white/[0.02] px-3.5 text-xs font-[450] text-white/40 hover:bg-white/[0.06] hover:text-white/70">
						<span class="flex items-center gap-1.5"><Download class="h-3 w-3 opacity-60" />Download desktop app</span>
					</SelectTrigger>
					<SelectContent class="rounded-xl border-white/10 bg-[#121212]">
						{#each Object.entries(DESKTOP_DOWNLOADS) as [os, d] (os)}
							<SelectItem value={os} label={d.label} />
						{/each}
					</SelectContent>
				</Select>
			</div>
			<p class="mt-2.5 text-[11px] tracking-wide text-white/25">
				Desktop builds served from <a href={RELEASES_URL} target="_blank" rel="noopener" class="underline decoration-white/10 underline-offset-2 hover:text-white/40">GitHub Releases</a>
				<span class="text-white/20"> · </span><span class="text-white/30">v{APP_VERSION}</span>
			</p>

			{#if showArchInstructions}
				<div class="mt-6 w-full max-w-xl rounded-2xl border border-white/[0.07] bg-[#0A0A0A] p-4 text-left">
					<div class="flex items-center justify-between">
						<h3 class="text-sm font-semibold tracking-tight">Arch Linux quick install</h3>
						<button onclick={() => (showArchInstructions = false)} class="rounded-full p-1 text-white/30 hover:bg-white/5 hover:text-white">✕</button>
					</div>
					<div class="relative mt-3">
						<button
							type="button"
							class="absolute right-1.5 top-1.5 z-10 rounded-full bg-white px-2.5 py-1 text-[11px] font-semibold text-black transition hover:bg-white/90"
							onclick={copyArchCommand}
						>
							{archCopied ? 'Copied!' : 'Copy'}
						</button>
						<pre class="overflow-x-auto rounded-xl border border-white/[0.06] bg-black px-4 py-3 pr-16 font-mono text-xs leading-relaxed text-white/80">{ARCH_INSTALL_CMD}</pre>
					</div>
				</div>
			{/if}
		</section>

		<!-- Features -->
		<section class="mx-auto w-full max-w-[1160px] px-4 py-12 sm:px-6 sm:py-16">
			<h2 class="text-[22px] font-[650] leading-[1.1] tracking-[-0.025em] sm:text-[28px]">
				<span class="text-white">Remarkably simple.</span><br />
				<span class="text-white/35">Big on control.</span>
			</h2>

			<div class="mt-7 grid gap-3.5 sm:grid-cols-3">
				{#each features as f}
					<div
						class="group relative flex flex-col overflow-hidden rounded-[16px] border border-white/[0.06] bg-[#0A0A0A] shadow-[0_8px_24px_rgba(0,0,0,0.4)] transition hover:border-white/[0.09] hover:shadow-[0_12px_32px_rgba(0,0,0,0.5)]"
					>
						<div class="absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-white/[0.07] to-transparent"></div>
						<!-- visual - refined, Raycast-subtle -->
						<div class="relative h-[156px] shrink-0 overflow-hidden border-b border-white/[0.05] bg-[#0A0A0A]">
							<div class="absolute inset-0 bg-[radial-gradient(ellipse_420px_180px_at_50%_0%,rgba(255,255,255,0.035),transparent_68%)]"></div>
							{#if f.title === 'Hardware effects'}
								<div class="absolute inset-0 bg-gradient-to-b {f.accent} opacity-[0.18]"></div>
							{/if}
							{#if f.title === 'Hardware effects'}
								<div class="absolute inset-0 flex flex-col items-center justify-center gap-3 p-6">
									<div class="flex items-center gap-1.5">
										{#each Array(5) as _, i}
											<span class="h-1.5 w-1.5 rounded-full {i === 2 ? 'bg-white shadow-[0_0_8px_rgba(255,255,255,0.6)]' : 'bg-white/20'}"></span>
										{/each}
									</div>
									<div class="h-9 w-full max-w-[220px] rounded-full border border-white/[0.06] bg-white/[0.04] p-1">
										<div class="h-full w-full rounded-full bg-gradient-to-r from-white/80 via-white/30 to-white/5"></div>
									</div>
									<div class="flex gap-1.5">
										{#each ['S', 'W', 'P', 'R', 'B', 'T'] as l}
											<span class="flex h-5 w-5 items-center justify-center rounded-full border border-white/10 bg-white/[0.04] text-[9px] font-medium text-white/35">{l}</span>
										{/each}
									</div>
								</div>
							{:else if f.title === 'Web or desktop'}
								<div class="absolute inset-0 flex items-center justify-center gap-4 p-6">
									<div class="flex flex-col items-center gap-2">
										<div class="flex h-[52px] w-[68px] items-center justify-center rounded-xl border border-white/10 bg-white text-black shadow-lg">
											<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"
												><circle cx="12" cy="12" r="9" /><path d="M3 12h18M12 3a15 15 0 0 1 0 18M12 3a15 15 0 0 0 0 18" /></svg
											>
										</div>
										<span class="text-[10px] tracking-wide text-white/30">Browser</span>
									</div>
									<div class="flex items-center gap-1 text-white/15">
										<span class="h-px w-4 bg-white/15"></span>
										<span class="flex h-5 w-5 items-center justify-center rounded-full border border-white/10 text-[10px] text-white/25">/</span>
										<span class="h-px w-4 bg-white/15"></span>
									</div>
									<div class="flex flex-col items-center gap-2">
										<div class="flex h-[52px] w-[68px] items-center justify-center rounded-xl border border-white/10 bg-[#141416] text-white">
											<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"
												><rect x="3" y="3" width="18" height="18" rx="3" /><path d="M8 3v18M3 8h18" /></svg
											>
										</div>
										<span class="text-[10px] tracking-wide text-white/30">Desktop</span>
									</div>
								</div>
							{:else}
								<div class="absolute inset-0 flex items-center justify-center">
									<div class="relative flex h-[84px] w-[84px] items-center justify-center rounded-full border border-white/[0.07] bg-white/[0.03]">
										<div class="absolute inset-0 rounded-full border border-dashed border-white/10"></div>
										<svg width="26" height="26" viewBox="0 0 24 24" fill="none" class="text-white/85"
											><path d="M12 3l7 4v5c0 4.2-2.9 8.1-7 9-4.1-.9-7-4.8-7-9V7l7-4z" stroke="currentColor" stroke-width="1.6" stroke-linejoin="round" /><path
												d="M9 12l2 2 4-4"
												stroke="currentColor"
												stroke-width="1.6"
												stroke-linecap="round"
												stroke-linejoin="round"
											/></svg
										>
									</div>
								</div>
							{/if}
							<span class="absolute bottom-2.5 left-2.5 rounded-full border border-white/[0.08] bg-black/60 px-2.5 py-1 text-[10.5px] font-[500] leading-none text-white/55 backdrop-blur"
								>{f.tag}</span
							>
						</div>
						<div class="relative flex flex-1 flex-col bg-[#0A0A0A] p-4 sm:p-[18px]">
							<h3 class="text-[13px] font-[600] tracking-tight text-white">{f.title}</h3>
							<p class="mt-1.5 text-[12.5px] leading-[1.6] text-white/40">{f.description}</p>
						</div>
					</div>
				{/each}
			</div>
		</section>

		<!-- Supported devices -->
		<section id="supported" class="border-t border-white/[0.05] bg-[#050507]">
			<div class="mx-auto w-full max-w-[1160px] px-4 py-12 sm:px-6 sm:py-16">
				<div class="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
					<div>
						<h2 class="text-[22px] font-[620] tracking-[-0.02em] sm:text-[26px]">Supported devices</h2>
						<p class="mt-1.5 text-sm text-white/40">
							{deviceCount} <span class="text-white/60">{activeCat.label}</span> devices
						</p>
					</div>
					<Tabs bind:value={activeCategory} class="w-auto">
						<TabsList class="h-8 rounded-full border border-white/10 bg-white/[0.04] p-1">
							{#each categories as cat}
								<TabsTrigger
									value={cat.id}
									class="rounded-full px-3.5 py-1 text-[13px] font-medium data-[state=active]:bg-white data-[state=active]:text-black data-[state=inactive]:text-white/50"
									>{cat.label}</TabsTrigger
								>
							{/each}
						</TabsList>
					</Tabs>
				</div>

				<Tabs bind:value={activeCategory} class="mt-8">
					{#each categories as cat}
						<TabsContent value={cat.id} class="mt-0">
							{#each cat.groups as group}
								<div class="mt-8 first:mt-6">
									<div class="flex items-center gap-2">
										<h3 class="text-[11px] font-semibold tracking-[0.08em] text-white/30 uppercase">{group.label}</h3>
										<span class="rounded-full border border-white/10 bg-white/[0.04] px-2 py-0.5 text-[10px] font-medium text-white/40">{group.items.length}</span>
										<span class="h-px flex-1 bg-white/[0.06]"></span>
									</div>
									<div class="mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
										{#each group.items as d}
											<div class="group flex h-[168px] flex-col overflow-hidden rounded-2xl border border-white/[0.07] bg-[#0A0A0A] transition hover:border-white/15">
												<div class="relative min-h-0 flex-1 bg-[#08080A]">
													{#if d.image}
														<img src={d.image} alt={d.name} loading="lazy" class="h-full w-full object-contain p-3 opacity-90 transition group-hover:opacity-100" />
													{:else}
														<div class="flex h-full w-full items-center justify-center text-white/10">
															<span class="text-2xl">⌨</span>
														</div>
													{/if}
													<!-- subtle top inner light -->
													<div class="pointer-events-none absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-white/10 to-transparent opacity-50"></div>
													<span
														class="absolute bottom-2 left-2 rounded-full border border-white/10 bg-black/60 px-2 py-0.5 text-[10px] font-medium text-white/60 backdrop-blur"
														>{cat.label}</span
													>
												</div>
												<div class="flex shrink-0 items-center justify-between border-t border-white/[0.06] bg-[#0F0F10] px-3.5 py-2.5">
													<span class="truncate text-[13px] font-[550] tracking-tight">{d.name}</span>
													<span class="ml-2 h-1.5 w-1.5 shrink-0 rounded-full bg-[#00ff88] shadow-[0_0_6px_rgba(0,255,136,0.5)]"></span>
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

		<!-- Troubleshooting - Raycast muted dark card -->
		<section id="troubleshooting" class="border-t border-white/[0.06] bg-black">
			<div class="mx-auto w-full max-w-[820px] px-4 py-12 sm:px-6 sm:py-16">
				<h2 class="text-[22px] font-[620] tracking-[-0.02em]">Troubleshooting</h2>
				<p class="mt-1 text-sm text-white/35">Why WebHID may not control your lighting on Linux.</p>

				<div class="mt-8 overflow-hidden rounded-2xl border border-white/[0.07] bg-[#0A0A0A]">
					<div class="h-px bg-gradient-to-r from-transparent via-white/10 to-transparent"></div>
					<div class="p-5 sm:p-6">
						<div class="inline-flex items-center gap-2 rounded-full border border-amber-500/20 bg-amber-500/10 px-2.5 py-1 text-[11px] font-medium text-amber-200/80">
							<span class="h-1.5 w-1.5 rounded-full bg-amber-400"></span>
							Linux needs a one-time permission fix
						</div>
						<p class="mt-3 max-w-2xl text-sm leading-relaxed text-white/50">
							Browsers talk to your device through WebHID, which on Linux is guarded by the OS. By default your device is only readable, not writable. The app connects but lighting
							controls do nothing until a small udev rule grants access.
						</p>
						<p class="mt-3 text-sm leading-relaxed text-white/50">
							If “Connect Device” looks like it worked but nothing changes, run these three lines, then <span class="text-white/70">unplug and replug your device</span> (or reboot):
						</p>

						<div class="relative mt-4 overflow-hidden rounded-xl border border-white/[0.07] bg-black">
							<button
								type="button"
								class="absolute right-1.5 top-1.5 z-10 rounded-full bg-white px-3 py-1 text-[11px] font-semibold text-black transition hover:bg-white/90"
								onclick={copyLinuxCommands}
							>
								{copied ? 'Copied!' : 'Copy'}
							</button>
							<pre class="overflow-x-auto px-4 py-3.5 pr-20 font-mono text-[11.5px] leading-relaxed text-white/75">{linuxSetupCommands}</pre>
						</div>

						<a
							href="https://developer.chrome.com/docs/capabilities/hid#:~:text=On%20most%20Linux%20systems"
							target="_blank"
							rel="noopener noreferrer"
							class="mt-4 inline-flex items-center gap-1.5 rounded-full border border-white/10 bg-white/[0.04] px-3.5 py-1.5 text-xs font-medium text-white/60 transition hover:bg-white/[0.08] hover:text-white"
						>
							Read on Chrome Docs
							<svg width="12" height="12" viewBox="0 0 12 12" fill="none"><path d="M4 3l4 3-4 3" stroke="currentColor" stroke-width="1.3" /></svg>
						</a>
					</div>
				</div>
			</div>
		</section>
	</main>

	<footer class="border-t border-white/[0.06] bg-black">
		<div class="mx-auto flex w-full max-w-[1160px] flex-col gap-3 px-4 py-6 sm:flex-row sm:items-center sm:justify-between sm:px-6">
			<span class="text-xs tracking-tight text-white/30">OpenPeripherals {new Date().getFullYear()} · Free & open source</span>
			<div class="flex items-center gap-1">
				<a
					href="https://github.com/ge0rg3e/openperipherals"
					target="_blank"
					rel="noopener"
					class="rounded-full px-3 py-1.5 text-xs font-medium text-white/40 hover:bg-white/5 hover:text-white transition">GitHub</a
				>
				<span class="text-white/10">·</span>
				<a href="https://x.com/ge0rg3e_dev" target="_blank" rel="noopener" class="rounded-full px-3 py-1.5 text-xs font-medium text-white/40 hover:bg-white/5 hover:text-white transition"
					>Follow @ge0rg3e_dev</a
				>
			</div>
		</div>
	</footer>
</div>
