<script lang="ts">
	import { cn } from '$lib/utils';
	import { Plus, Minus, ChevronRight, MousePointer2, RotateCcw, SlidersHorizontal, Zap, Keyboard, Layers } from '@lucide/svelte';

	type SettingRow = { label: string; value: string; type?: 'text' | 'toggle' | 'slider' | 'radio' };

	let {
		id,
		title,
		icon,
		expanded = false,
		onToggle,
		onSelect,
		rows = [],
		position = 'right',
		leaderLine = { x1: 0, y1: 0, x2: 0, y2: 0 }
	}: {
		id: string;
		title: string;
		icon: any;
		expanded: boolean;
		onToggle: (id: string) => void;
		onSelect: (id: string) => void;
		rows: SettingRow[];
		position: 'left' | 'right' | 'top' | 'bottom';
		leaderLine: { x1: number; y1: number; x2: number; y2: number };
	} = $props();

	const icons: Record<string, any> = {
		mouse: MousePointer2,
		scroll: RotateCcw,
		thumb: RotateCcw,
		dpi: SlidersHorizontal,
		polling: Zap,
		lighting: Layers,
		keys: Keyboard
	};

	const Icon = icons[icon] || MousePointer2;
</script>

<div class={cn('callout-card', expanded && 'expanded', position === 'left' && 'pos-left', position === 'right' && 'pos-right')} role="region" aria-label={title}>
	<!-- Leader line -->
	<svg class="leader-line" viewBox="0 0 100 100" preserveAspectRatio="none" aria-hidden="true">
		<defs>
			<marker id="arrowhead" markerWidth="8" markerHeight="6" refX="6" refY="3" orient="auto">
				<polygon points="0 0, 8 3, 0 6" fill="#00FF00" />
			</marker>
		</defs>
		<path 
			d="M{leaderLine.x1} {leaderLine.y1} Q{(leaderLine.x1 + leaderLine.x2) / 2} {leaderLine.y1} {(leaderLine.x1 + leaderLine.x2) / 2} {(leaderLine.y1 + leaderLine.y2) / 2} T{leaderLine.x2} {leaderLine.y2}" 
			stroke="#00FF00" 
			stroke-width="1.5" 
			fill="none" 
			stroke-dasharray="4 4"
			marker-end="url(#arrowhead)"
			style="filter: drop-shadow(0 0 4px #00FF00);"
		/>
	</svg>

	<div class="card-content" onclick={(e) => { e.stopPropagation(); onSelect(id); }}>
		<!-- Header -->
		<div class="card-header">
			<div class="icon-badge">
				<Icon class="size-4" aria-hidden="true" />
			</div>
			<h3 class="card-title">{title}</h3>
			<button 
				class="expand-btn" 
				class:rotated={expanded}
				aria-expanded={expanded}
				aria-label={expanded ? 'Collapse' : 'Expand'}
				onclick={(e) => { e.stopPropagation(); onToggle(id); }}
			>
				{#if expanded}
					<Minus class="size-4" />
				{:else}
					<Plus class="size-4" />
				{/if}
			</button>
		</div>

		<!-- Body -->
		{#if expanded}
			<div class="card-body">
				{#each rows as row (row.label)}
					<div class="setting-row" class:interactive={row.type !== 'text'}>
						<span class="setting-label">{row.label}</span>
						<span class="setting-value">{row.value}</span>
						{#if row.type !== 'text'}
							<ChevronRight class="size-3.5 chevron" aria-hidden="true" />
						{/if}
					</div>
				{/each}
			</div>
		{/if}
	</div>
</div>

<style>
	.callout-card {
		position: absolute;
		min-width: 220px;
		max-width: 280px;
		background: #1C1C1F;
		border: 1px solid #2E2E32;
		border-radius: 12px;
		box-shadow: 
			0 8px 32px rgba(0, 0, 0, 0.4),
			0 0 0 1px rgba(255, 255, 255, 0.02) inset,
			0 0 24px rgba(0, 255, 0, 0.08);
		z-index: 10;
		transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
		transform-origin: center center;
	}

	.callout-card.pos-right {
		transform-origin: left center;
	}

	.callout-card.pos-left {
		transform-origin: right center;
	}

	.callout-card:not(.expanded) {
		min-width: 180px;
	}

	.callout-card.expanded {
		box-shadow: 
			0 12px 48px rgba(0, 0, 0, 0.5),
			0 0 0 1px rgba(255, 255, 255, 0.03) inset,
			0 0 32px rgba(0, 255, 0, 0.15),
			0 0 0 2px rgba(0, 255, 0, 0.1);
		border-color: rgba(0, 255, 0, 0.3);
	}

	.leader-line {
		position: absolute;
		pointer-events: none;
		z-index: -1;
	}

	.callout-card.pos-right .leader-line {
		left: -40px;
		top: 50%;
		width: 40px;
		height: 100%;
		transform: translateY(-50%);
	}

	.callout-card.pos-left .leader-line {
		right: -40px;
		top: 50%;
		width: 40px;
		height: 100%;
		transform: translateY(-50%);
	}

	.card-content {
		padding: 0;
	}

	.card-header {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 12px 14px;
		background: linear-gradient(90deg, rgba(0, 255, 0, 0.1) 0%, transparent 100%);
		border-bottom: 1px solid rgba(0, 255, 0, 0.1);
		border-radius: 12px 12px 0 0;
	}

	.icon-badge {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 28px;
		height: 28px;
		border-radius: 8px;
		background: linear-gradient(135deg, #00FF00 0%, #39FF14 100%);
		color: #0A0A0C;
		flex-shrink: 0;
	}

	.card-title {
		flex: 1;
		font-family: 'Inter', 'Poppins', system-ui, sans-serif;
		font-size: 13px;
		font-weight: 600;
		color: #E8E8E8;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.expand-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 24px;
		height: 24px;
		border-radius: 6px;
		border: 1px solid #2E2E32;
		background: rgba(255, 255, 255, 0.03);
		color: #8A8A8E;
		cursor: pointer;
		transition: all 0.2s ease;
		flex-shrink: 0;
	}

	.expand-btn:hover {
		background: rgba(0, 255, 0, 0.1);
		border-color: #00FF00;
		color: #00FF00;
	}

	.expand-btn.rotated {
		transform: rotate(45deg);
	}

	.card-body {
		padding: 10px 12px;
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.setting-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
		padding: 8px 10px;
		border-radius: 8px;
		background: rgba(255, 255, 255, 0.02);
		border: 1px solid transparent;
		transition: all 0.2s ease;
	}

	.setting-row.interactive {
		cursor: pointer;
	}

	.setting-row.interactive:hover {
		background: rgba(0, 255, 0, 0.05);
		border-color: rgba(0, 255, 0, 0.15);
	}

	.setting-label {
		font-family: 'Inter', 'Poppins', system-ui, sans-serif;
		font-size: 12px;
		font-weight: 500;
		color: #E8E8E8;
		flex-shrink: 0;
	}

	.setting-value {
		font-family: 'Inter', 'Poppins', system-ui, sans-serif;
		font-size: 12px;
		font-weight: 500;
		color: #00FF00;
		font-variant-numeric: tabular-nums;
		flex-shrink: 0;
	}

	.chevron {
		color: #5A5A5E;
		transition: transform 0.2s ease;
	}

	.setting-row.interactive:hover .chevron {
		color: #00FF00;
		transform: translateX(2px);
	}

	/* Slide animation */
	@keyframes slideIn {
		from { opacity: 0; transform: translateY(-8px); }
		to { opacity: 1; transform: translateY(0); }
	}

	@keyframes slideOut {
		from { opacity: 1; transform: translateY(0); }
		to { opacity: 0; transform: translateY(-8px); }
	}

	.card-body {
		animation: slideIn 0.2s ease-out;
	}

	.callout-card:not(.expanded) .card-body {
		display: none;
	}
</style>