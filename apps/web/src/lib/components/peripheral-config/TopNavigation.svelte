<script lang="ts">
	import { cn } from '$lib/utils';
	import { MousePointer2, Keyboard, Settings, Zap, Layers, User, Menu } from '@lucide/svelte';

	type Tab = { id: string; label: string; icon: any };

	let { activeTab = 'lighting', onTabChange }: { activeTab: string; onTabChange: (tab: string) => void } = $props();

	const tabs: Tab[] = [
		{ id: 'lighting', label: 'Lighting', icon: Layers },
		{ id: 'custom-keys', label: 'Custom Keys', icon: Keyboard },
		{ id: 'performance', label: 'Performance', icon: Zap },
		{ id: 'macros', label: 'Macro Management', icon: Settings }
	];
</script>

<nav class="top-nav" role="navigation" aria-label="Main navigation">
	<div class="nav-left">
		<div class="brand">
			<div class="brand-icon">
				<MousePointer2 class="size-5" />
			</div>
			<span class="brand-text">OpenPeripherals</span>
		</div>
	</div>

	<div class="nav-center">
		<div class="tab-container" role="tablist">
			{#each tabs as tab (tab.id)}
				<button
					type="button"
					role="tab"
					aria-selected={activeTab === tab.id}
					aria-controls={`panel-${tab.id}`}
					class={cn(
						'tab-btn',
						activeTab === tab.id && 'active'
					)}
					onclick={() => onTabChange(tab.id)}
				>
					<tab.icon class="size-4" aria-hidden="true" />
					<span>{tab.label}</span>
				</button>
			{/each}
		</div>
	</div>

	<div class="nav-right">
		<button class="profile-btn" aria-label="Profile menu">
			<div class="avatar">
				<User class="size-4" />
			</div>
		</button>
	</div>
</nav>

<style>
	.top-nav {
		position: fixed;
		top: 0;
		left: 0;
		right: 0;
		height: 60px;
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0 24px;
		background: linear-gradient(180deg, rgba(10, 10, 12, 0.95) 0%, rgba(18, 18, 20, 0.9) 100%);
		border-bottom: 1px solid #2E2E32;
		backdrop-filter: blur(20px);
		z-index: 100;
	}

	.nav-left {
		display: flex;
		align-items: center;
	}

	.brand {
		display: flex;
		align-items: center;
		gap: 10px;
	}

	.brand-icon {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 32px;
		height: 32px;
		border-radius: 8px;
		background: linear-gradient(135deg, #00FF00 0%, #39FF14 100%);
		box-shadow: 0 0 20px rgba(0, 255, 0, 0.3);
	}

	.brand-icon :global(svg) {
		color: #0A0A0C;
	}

	.brand-text {
		font-family: 'Inter', 'Poppins', system-ui, sans-serif;
		font-size: 16px;
		font-weight: 700;
		color: #E8E8E8;
		letter-spacing: -0.02em;
	}

	.nav-center {
		flex: 1;
		display: flex;
		justify-content: center;
	}

	.tab-container {
		display: flex;
		align-items: center;
		gap: 4px;
		background: rgba(28, 28, 31, 0.6);
		border: 1px solid #2E2E32;
		border-radius: 12px;
		padding: 4px;
		backdrop-filter: blur(10px);
	}

	.tab-btn {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 8px 16px;
		border: none;
		background: transparent;
		border-radius: 8px;
		color: #8A8A8E;
		font-family: 'Inter', 'Poppins', system-ui, sans-serif;
		font-size: 13px;
		font-weight: 500;
		cursor: pointer;
		transition: all 0.2s ease;
		white-space: nowrap;
	}

	.tab-btn:hover:not(.active) {
		color: #E8E8E8;
		background: rgba(255, 255, 255, 0.03);
	}

	.tab-btn.active {
		background: #1C1C1F;
		color: #00FF00;
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3), inset 0 1px 0 rgba(255, 255, 255, 0.05);
	}

	.tab-btn:focus-visible {
		outline: 2px solid #00FF00;
		outline-offset: -2px;
	}

	.nav-right {
		display: flex;
		align-items: center;
		gap: 12px;
	}

	.profile-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 36px;
		height: 36px;
		border: 1px solid #2E2E32;
		border-radius: 50%;
		background: #1C1C1F;
		color: #8A8A8E;
		cursor: pointer;
		transition: all 0.2s ease;
	}

	.profile-btn:hover {
		border-color: #00FF00;
		color: #00FF00;
		box-shadow: 0 0 12px rgba(0, 255, 0, 0.15);
	}

	.avatar {
		display: flex;
		align-items: center;
		justify-content: center;
	}
</style>