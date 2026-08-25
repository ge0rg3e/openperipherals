<script lang="ts">
	import { cn } from '$lib/utils';
	import { onMount } from 'svelte';
	import { MousePointer2, RotateCcw, SlidersHorizontal, Zap, Keyboard, Layers, ChevronLeft, Check, X, Minus, Plus } from '@lucide/svelte';

	type SettingGroup = {
		id: string;
		label: string;
		icon: any;
		type: 'slider' | 'toggle' | 'radio' | 'select' | 'button-group' | 'text';
		value: any;
		min?: number;
		max?: number;
		step?: number;
		unit?: string;
		options?: Array<{ value: any; label: string; description?: string }>;
		description?: string;
		onChange: (value: any) => void;
	};

	type SidebarData = {
		title: string;
		icon: any;
		groups: SettingGroup[];
	} | null;

	let { data, collapsed = false, onCollapseChange }: { data: SidebarData; collapsed: boolean; onCollapseChange: (c: boolean) => void } = $props();

	const icons: Record<string, any> = {
		mouse: MousePointer2,
		scroll: RotateCcw,
		dpi: SlidersHorizontal,
		polling: Zap,
		keys: Keyboard,
		lighting: Layers
	};

	const HeaderIcon = $derived(data?.icon ?? MousePointer2);

	let activeSlider = false;

	function startDrag(e: MouseEvent, group: SettingGroup) {
		e.preventDefault();
		const track = ((e.currentTarget as HTMLElement).closest('.slider-track') as HTMLElement) ?? (e.currentTarget as HTMLElement);
		const rect = track.getBoundingClientRect();
		const min = group.min ?? 0;
		const max = group.max ?? 100;
		const step = group.step || 1;
		const apply = (clientX: number) => {
			const ratio = Math.max(0, Math.min(1, (clientX - rect.left) / rect.width));
			group.onChange(Math.round((min + ratio * (max - min)) / step) * step);
		};
		activeSlider = true;
		apply(e.clientX);
		const move = (ev: MouseEvent) => {
			if (!activeSlider) return;
			apply(ev.clientX);
		};
		const up = () => {
			activeSlider = false;
			window.removeEventListener('mousemove', move);
			window.removeEventListener('mouseup', up);
		};
		window.addEventListener('mousemove', move);
		window.addEventListener('mouseup', up);
	}

	function handleKeyDrag(e: KeyboardEvent, group: SettingGroup) {
		const step = group.step || 1;
		const min = group.min ?? 0;
		const max = group.max ?? 100;
		const current = Number(group.value) || min;
		if (e.key === 'ArrowLeft' || e.key === 'ArrowDown') {
			group.onChange(Math.max(min, current - step));
			e.preventDefault();
		} else if (e.key === 'ArrowRight' || e.key === 'ArrowUp') {
			group.onChange(Math.min(max, current + step));
			e.preventDefault();
		}
	}
</script>

<aside class={cn('sidebar', collapsed && 'collapsed')} role="complementary" aria-label="Device settings">
	<!-- Collapse handle -->
	<button class="collapse-handle" onclick={() => onCollapseChange(!collapsed)} aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'} aria-expanded={!collapsed}>
		<ChevronLeft class="size-4 {collapsed ? 'rotated' : ''}" />
	</button>

	{#if !collapsed && data}
		<div class="sidebar-content">
			<!-- Section Header -->
			<div class="sidebar-header">
				<div class="header-badge">
					<HeaderIcon class="size-5" aria-hidden="true" />
				</div>
				<h2 class="header-title">{data.title}</h2>
			</div>

			<div class="sidebar-body">
				{#each data.groups as group (group.id)}
					<div class="setting-group">
						<div class="group-header">
							<label class="group-label">{group.label}</label>
							<span class="group-value">
								{#if group.type === 'slider'}
									{group.value}{group.unit || ''}
								{:else if group.type === 'toggle'}
									{group.value ? 'ON' : 'OFF'}
								{:else}
									{group.value}
								{/if}
							</span>
						</div>

						{#if group.description}
							<p class="group-description">{group.description}</p>
						{/if}

						<!-- Slider -->
						{#if group.type === 'slider'}
							<div class="slider-container">
								<div
									class="slider-track"
									role="slider"
									aria-valuenow={Number(group.value)}
									aria-valuemin={group.min}
									aria-valuemax={group.max}
									aria-label={group.label}
									tabindex="0"
									onmousedown={(e) => startDrag(e, group)}
									onkeydown={(e) => handleKeyDrag(e, group)}
								>
									<div class="slider-fill" style="width: {((Number(group.value) - (group.min || 0)) / ((group.max || 100) - (group.min || 0))) * 100}%"></div>
									<div class="slider-thumb" style="left: {((Number(group.value) - (group.min || 0)) / ((group.max || 100) - (group.min || 0))) * 100}%"></div>
									<div class="slider-ticks" aria-hidden="true">
										{#each Array.from({ length: 11 }) as _, i}
											<span style="left: {i * 10}%"></span>
										{/each}
									</div>
								</div>
								<div class="slider-labels">
									<span>{group.min}{group.unit || ''}</span>
									<span>{group.max}{group.unit || ''}</span>
								</div>
							</div>
						{/if}

						<!-- Toggle -->
						{#if group.type === 'toggle'}
							<button
								class="toggle-switch"
								class:on={group.value}
								role="switch"
								aria-checked={!!group.value}
								aria-label={group.label}
								onclick={() => group.onChange(!group.value)}
							>
								<div class="toggle-track">
									<div class="toggle-thumb" />
								</div>
							</button>
						{/if}

						<!-- Radio Group -->
						{#if group.type === 'radio' && group.options}
							<div class="radio-group" role="radiogroup" aria-label={group.label}>
								{#each group.options as option (option.value)}
									<button
										type="button"
										class={cn('radio-btn', group.value === option.value && 'selected')}
										role="radio"
										aria-checked={group.value === option.value}
										onclick={() => group.onChange(option.value)}
									>
										<div class="radio-circle">
											{#if group.value === option.value}
												<Check class="size-3" />
											{/if}
										</div>
										<div class="radio-content">
											<span class="radio-label">{option.label}</span>
											{#if option.description}
												<span class="radio-desc">{option.description}</span>
											{/if}
										</div>
									</button>
								{/each}
							</div>
						{/if}

						<!-- Select -->
						{#if group.type === 'select' && group.options}
							<select 
								class="select-input"
								value={group.value}
								onchange={(e) => group.onChange(e.currentTarget.value)}
								aria-label={group.label}
							>
								{#each group.options as option (option.value)}
									<option value={option.value}>{option.label}</option>
								{/each}
							</select>
						{/if}

						<!-- Button Group -->
						{#if group.type === 'button-group' && group.options}
							<div class="button-group" role="group" aria-label={group.label}>
								{#each group.options as option (option.value)}
									<button
										type="button"
										class={cn('group-btn', group.value === option.value && 'active')}
										onclick={() => group.onChange(option.value)}
									>
										{option.label}
									</button>
								{/each}
							</div>
						{/if}
					</div>
				{/each}
			</div>
		</div>
	{:else if !collapsed}
		<div class="sidebar-empty">
			<div class="empty-icon">
				<MousePointer2 class="size-8" />
			</div>
			<p class="empty-text">Select a zone on the device to configure its settings</p>
		</div>
	{/if}
</aside>

<style>
	.sidebar {
		position: fixed;
		top: 60px;
		right: 0;
		bottom: 0;
		width: 320px;
		background: linear-gradient(180deg, #121214 0%, #0A0A0C 100%);
		border-left: 1px solid #2E2E32;
		display: flex;
		flex-direction: column;
		z-index: 50;
		transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1), width 0.3s cubic-bezier(0.4, 0, 0.2, 1);
		overflow: hidden;
	}

	.sidebar.collapsed {
		width: 48px;
	}

	.collapse-handle {
		position: absolute;
		left: -12px;
		top: 50%;
		transform: translateY(-50%);
		display: flex;
		align-items: center;
		justify-content: center;
		width: 24px;
		height: 48px;
		border-radius: 0 8px 8px 0;
		border: 1px solid #2E2E32;
		border-right: none;
		background: #1C1C1F;
		color: #8A8A8E;
		cursor: pointer;
		transition: all 0.2s ease;
		z-index: 10;
	}

	.collapse-handle:hover {
		background: #232326;
		border-color: #00FF00;
		color: #00FF00;
	}

	.collapse-handle .rotated {
		transform: rotate(180deg);
	}

	.sidebar-content {
		flex: 1;
		display: flex;
		flex-direction: column;
		overflow: hidden;
	}

	.sidebar.collapsed .sidebar-content {
		display: none;
	}

	.sidebar-header {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 20px 20px 16px;
		border-bottom: 1px solid #2E2E32;
		flex-shrink: 0;
	}

	.header-badge {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 40px;
		height: 40px;
		border-radius: 10px;
		background: linear-gradient(135deg, #00FF00 0%, #39FF14 100%);
		color: #0A0A0C;
		flex-shrink: 0;
	}

	.header-title {
		font-family: 'Inter', 'Poppins', system-ui, sans-serif;
		font-size: 16px;
		font-weight: 700;
		color: #E8E8E8;
		letter-spacing: -0.02em;
	}

	.sidebar-body {
		flex: 1;
		overflow-y: auto;
		padding: 16px 20px 24px;
		display: flex;
		flex-direction: column;
		gap: 20px;
	}

	.setting-group {
		display: flex;
		flex-direction: column;
		gap: 10px;
	}

	.group-header {
		display: flex;
		align-items: baseline;
		justify-content: space-between;
		gap: 12px;
	}

	.group-label {
		font-family: 'Inter', 'Poppins', system-ui, sans-serif;
		font-size: 13px;
		font-weight: 600;
		color: #E8E8E8;
	}

	.group-value {
		font-family: 'Inter', 'Poppins', system-ui, sans-serif;
		font-size: 13px;
		font-weight: 700;
		color: #00FF00;
		font-variant-numeric: tabular-nums;
		white-space: nowrap;
	}

	.group-description {
		font-family: 'Inter', 'Poppins', system-ui, sans-serif;
		font-size: 11px;
		color: #8A8A8E;
		line-height: 1.5;
		margin: 0;
	}

	/* Slider */
	.slider-container {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.slider-track {
		position: relative;
		height: 32px;
		border-radius: 6px;
		background: #0D0D0F;
		border: 1px solid #2E2E32;
		cursor: pointer;
		touch-action: none;
	}

	.slider-track:focus-visible {
		outline: 2px solid #00FF00;
		outline-offset: 2px;
		border-color: #00FF00;
	}

	.slider-fill {
		position: absolute;
		top: 0;
		left: 0;
		bottom: 0;
		border-radius: 6px 0 0 6px;
		background: linear-gradient(90deg, #00FF00 0%, #39FF14 100%);
		transition: width 0.1s ease-out;
	}

	.slider-thumb {
		position: absolute;
		top: 50%;
		width: 18px;
		height: 18px;
		border-radius: 50%;
		background: #FFFFFF;
		border: 2px solid #00FF00;
		transform: translate(-50%, -50%);
		box-shadow: 
			0 2px 8px rgba(0, 0, 0, 0.4),
			0 0 0 2px #0A0A0C,
			0 0 12px rgba(0, 255, 0, 0.4);
		transition: transform 0.1s ease, box-shadow 0.2s ease;
		z-index: 2;
	}

	.slider-track:hover .slider-thumb,
	.slider-track:focus-visible .slider-thumb {
		transform: translate(-50%, -50%) scale(1.15);
		box-shadow: 
			0 4px 16px rgba(0, 0, 0, 0.5),
			0 0 0 2px #0A0A0C,
			0 0 20px rgba(0, 255, 0, 0.6);
	}

	.slider-ticks {
		position: absolute;
		top: 0;
		left: 0;
		right: 0;
		bottom: 0;
		pointer-events: none;
	}

	.slider-ticks span {
		position: absolute;
		top: 50%;
		width: 1px;
		height: 8px;
		background: rgba(255, 255, 255, 0.06);
		transform: translate(-50%, -50%);
	}

	.slider-ticks span:first-child,
	.slider-ticks span:last-child {
		display: none;
	}

	.slider-labels {
		display: flex;
		justify-content: space-between;
		font-family: 'Inter', 'Poppins', system-ui, sans-serif;
		font-size: 10px;
		font-weight: 500;
		color: #5A5A5E;
		font-variant-numeric: tabular-nums;
	}

	/* Toggle Switch */
	.toggle-switch {
		display: flex;
		align-items: center;
		justify-content: flex-end;
		width: 56px;
		height: 30px;
		border: none;
		background: transparent;
		cursor: pointer;
		padding: 0;
	}

	.toggle-track {
		position: relative;
		width: 56px;
		height: 30px;
		border-radius: 15px;
		background: #2E2E32;
		border: 1px solid #3A3A3E;
		transition: all 0.2s ease;
	}

	.toggle-switch.on .toggle-track {
		background: linear-gradient(90deg, #00FF00 0%, #39FF14 100%);
		border-color: #00FF00;
		box-shadow: 0 0 12px rgba(0, 255, 0, 0.3);
	}

	.toggle-thumb {
		position: absolute;
		top: 2px;
		left: 2px;
		width: 24px;
		height: 24px;
		border-radius: 50%;
		background: #FFFFFF;
		box-shadow: 0 2px 6px rgba(0, 0, 0, 0.3);
		transition: transform 0.2s cubic-bezier(0.4, 0, 0.2, 1);
	}

	.toggle-switch.on .toggle-thumb {
		transform: translateX(26px);
	}

	.toggle-switch:focus-visible .toggle-track {
		outline: 2px solid #00FF00;
		outline-offset: 2px;
	}

	/* Radio Group */
	.radio-group {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.radio-btn {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 10px 12px;
		border: 1px solid #2E2E32;
		border-radius: 8px;
		background: rgba(255, 255, 255, 0.02);
		cursor: pointer;
		transition: all 0.2s ease;
		text-align: left;
		width: 100%;
	}

	.radio-btn:hover {
		background: rgba(0, 255, 0, 0.05);
		border-color: rgba(0, 255, 0, 0.3);
	}

	.radio-btn.selected {
		background: rgba(0, 255, 0, 0.08);
		border-color: #00FF00;
		box-shadow: 0 0 0 1px rgba(0, 255, 0, 0.1) inset;
	}

	.radio-circle {
		position: relative;
		flex-shrink: 0;
		width: 18px;
		height: 18px;
		border-radius: 50%;
		border: 2px solid #5A5A5E;
		background: transparent;
		display: flex;
		align-items: center;
		justify-content: center;
		transition: all 0.2s ease;
	}

	.radio-btn.selected .radio-circle {
		border-color: #00FF00;
		background: rgba(0, 255, 0, 0.1);
	}

	.radio-circle .size-3 {
		color: #00FF00;
	}

	.radio-content {
		display: flex;
		flex-direction: column;
		gap: 2px;
		min-width: 0;
	}

	.radio-label {
		font-family: 'Inter', 'Poppins', system-ui, sans-serif;
		font-size: 13px;
		font-weight: 500;
		color: #E8E8E8;
	}

	.radio-desc {
		font-family: 'Inter', 'Poppins', system-ui, sans-serif;
		font-size: 11px;
		color: #8A8A8E;
	}

	/* Select */
	.select-input {
		width: 100%;
		height: 40px;
		padding: 0 12px;
		border: 1px solid #2E2E32;
		border-radius: 8px;
		background: #0D0D0F;
		color: #E8E8E8;
		font-family: 'Inter', 'Poppins', system-ui, sans-serif;
		font-size: 13px;
		font-weight: 500;
		cursor: pointer;
		appearance: none;
		background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='16' height='16' viewBox='0 0 24 24' fill='none' stroke='%238A8A8E' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='m6 9 6 6 6-6'/%3E%3C/svg%3E");
		background-repeat: no-repeat;
		background-position: right 12px center;
		padding-right: 36px;
		transition: all 0.2s ease;
	}

	.select-input:hover {
		border-color: rgba(0, 255, 0, 0.3);
	}

	.select-input:focus {
		outline: none;
		border-color: #00FF00;
		box-shadow: 0 0 0 3px rgba(0, 255, 0, 0.1);
	}

	/* Button Group */
	.button-group {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
	}

	.group-btn {
		padding: 8px 14px;
		border: 1px solid #2E2E32;
		border-radius: 8px;
		background: rgba(255, 255, 255, 0.02);
		color: #8A8A8E;
		font-family: 'Inter', 'Poppins', system-ui, sans-serif;
		font-size: 12px;
		font-weight: 500;
		cursor: pointer;
		transition: all 0.2s ease;
		white-space: nowrap;
	}

	.group-btn:hover:not(.active) {
		border-color: rgba(0, 255, 0, 0.3);
		color: #E8E8E8;
		background: rgba(0, 255, 0, 0.03);
	}

	.group-btn.active {
		background: linear-gradient(90deg, rgba(0, 255, 0, 0.15) 0%, rgba(57, 255, 20, 0.1) 100%);
		border-color: #00FF00;
		color: #00FF00;
		box-shadow: 0 0 12px rgba(0, 255, 0, 0.15);
	}

	.group-btn:focus-visible {
		outline: 2px solid #00FF00;
		outline-offset: 2px;
	}

	/* Empty State */
	.sidebar-empty {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 24px;
		text-align: center;
		color: #5A5A5E;
	}

	.empty-icon {
		margin-bottom: 12px;
		opacity: 0.4;
	}

	.empty-text {
		font-family: 'Inter', 'Poppins', system-ui, sans-serif;
		font-size: 13px;
		line-height: 1.5;
		margin: 0;
	}

	/* Scrollbar */
	.sidebar-body::-webkit-scrollbar {
		width: 6px;
	}

	.sidebar-body::-webkit-scrollbar-track {
		background: transparent;
	}

	.sidebar-body::-webkit-scrollbar-thumb {
		background: #2E2E32;
		border-radius: 3px;
	}

	.sidebar-body::-webkit-scrollbar-thumb:hover {
		background: #3A3A3E;
	}
</style>