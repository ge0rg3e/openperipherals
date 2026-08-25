<script lang="ts">
	import { onMount } from 'svelte';
	import { logger, type LogEntry } from '$lib/razer/logger';
	import { describeReport } from '$lib/razer/describe';
	import { logVersion } from '$lib/store';

	let entries = $state(logger.all);
	let collapsed = $state(false);

	let x = $state(0);
	let y = $state(0);
	let positioned = false;

	let dragging = false;
	let offsetX = 0;
	let offsetY = 0;
	let startX = 0;
	let startY = 0;

	let listEl = $state<HTMLDivElement | undefined>();

	// reflector box used to measure the panel once it is laid out
	let measured = $state<HTMLDivElement | undefined>();

	function fmtHex(hex?: Uint8Array): string {
		if (!hex || hex.length === 0) return '';
		const parts: string[] = [];
		for (let i = 0; i < hex.length; i++) {
			parts.push(hex[i].toString(16).padStart(2, '0'));
		}
		return parts.join(' ');
	}

	function time(ts: number): string {
		return new Date(ts).toLocaleTimeString(undefined, {
			hour12: false,
			fractionalSecondDigits: 3
		});
	}

	// keep in sync with the shared log buffer, always following the newest line
	$effect(() => {
		return logVersion.subscribe(() => {
			entries = logger.all;
			requestAnimationFrame(() => {
				if (listEl) listEl.scrollTop = listEl.scrollHeight;
			});
		});
	});

	// default position: top-right, just inside the viewport
	$effect(() => {
		if (positioned || !measured) return;
		positioned = true;
		x = window.innerWidth - measured.offsetWidth - 16;
		y = 16;
	});

	// drag by the header
	function onHeaderDown(e: PointerEvent) {
		dragging = true;
		startX = e.clientX;
		startY = e.clientY;
		offsetX = x;
		offsetY = y;
		window.addEventListener('pointermove', onMove);
		window.addEventListener('pointerup', onUp);
	}

	function onMove(e: PointerEvent) {
		if (!dragging) return;
		const nx = offsetX + e.clientX - startX;
		const ny = offsetY + e.clientY - startY;
		x = Math.max(0, Math.min(nx, window.innerWidth - measured!.offsetWidth));
		y = Math.max(0, Math.min(ny, window.innerHeight - 40));
	}

	function onUp() {
		dragging = false;
		window.removeEventListener('pointermove', onMove);
		window.removeEventListener('pointerup', onUp);
	}
</script>

<div
	class="devlog"
	role="dialog"
	aria-label="Live HID communication log"
	style:left="{x}px"
	style:top="{y}px"
>
	<div
		bind:this={measured}
		class="hide-measure"
	></div>
	<div class="dl-head" role="button" tabindex="-1" aria-label="Drag to move log panel" onpointerdown={onHeaderDown} onpointerup={onUp}>
		<span class="dl-title">Live HID log</span>
		<button
			type="button"
			class="dl-toggle"
			title={collapsed ? 'Expand' : 'Collapse'}
			onclick={() => (collapsed = !collapsed)}
		>
			{collapsed ? '+' : '×'}
		</button>
	</div>

	{#if !collapsed}
		<div class="dl-list" bind:this={listEl}>
			{#each entries as e (e.id)}
				<div class="dl-row dl-{e.level}">
					<span class="dl-ts">{time(e.ts)}</span>
					<span class="dl-dir">{e.direction === 'tx' ? '→' : e.direction === 'rx' ? '←' : '·'}</span>
					<span class="dl-msg">{e.message}</span>
					{#if e.hex && e.hex.length}
						<span class="dl-hint">{describeReport(e.hex, e.direction ?? 'tx')}</span>
						<span class="dl-hex">{fmtHex(e.hex)}</span>
					{/if}
				</div>
			{/each}
		</div>
	{/if}
</div>

<style>
	.devlog {
		position: fixed;
		z-index: 9999;
		width: min(480px, calc(100vw - 32px));
		font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
		font-size: 11px;
		line-height: 1.5;
		color: #d4d4d4;
		background: rgba(8, 10, 14, 0.92);
		border: 1px solid rgba(255, 255, 255, 0.12);
		border-radius: 8px;
		box-shadow: 0 8px 30px rgba(0, 0, 0, 0.5);
		overflow: hidden;
		user-select: none;
	}

	.hide-measure {
		position: absolute;
		visibility: hidden;
		pointer-events: none;
		width: inherit;
		height: 0;
	}

	.dl-head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 8px;
		padding: 6px 8px;
		background: rgba(255, 255, 255, 0.06);
		border-bottom: 1px solid rgba(255, 255, 255, 0.1);
		cursor: move;
	}

	.dl-title {
		font-weight: 600;
		font-size: 12px;
		letter-spacing: 0.04em;
	}

	.dl-toggle {
		background: none;
		border: 0;
		color: inherit;
		cursor: pointer;
		font-size: 14px;
		line-height: 1;
		padding: 0 3px;
		border-radius: 4px;
	}
	.dl-toggle:hover {
		background: rgba(255, 255, 255, 0.1);
	}

	.dl-list {
		overflow-y: auto;
		max-height: 40vh;
	}

	.dl-row {
		display: block;
		padding: 2px 8px;
		border-bottom: 1px solid rgba(255, 255, 255, 0.04);
		white-space: pre-wrap;
		word-break: break-all;
	}

	.dl-send {
		color: #7ee787;
	}
	.dl-recv {
		color: #79c0ff;
	}
	.dl-warn {
		color: #d29922;
	}
	.dl-error {
		color: #ff7b72;
	}
	.dl-info {
		color: #d4d4d4;
	}

	.dl-ts {
		color: #6e7681;
		margin-right: 6px;
	}

	.dl-dir {
		margin-right: 6px;
		font-weight: 700;
	}

	.dl-msg {
		margin-right: 6px;
	}

	.dl-hint {
		display: block;
		color: #ffd28a;
		margin: 1px 0 1px 0;
		font-weight: 500;
	}

	.dl-hex {
		display: block;
		color: #6e7681;
		font-size: 10px;
		line-height: 1.4;
	}
</style>