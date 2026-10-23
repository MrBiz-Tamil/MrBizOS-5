<script lang="ts">
	import { onMount } from 'svelte';

	let currentTheme = $state<'light' | 'dark'>('light');

	onMount(() => {
		const stored = (localStorage.getItem('mrbiz_theme') as 'light' | 'dark') || 'light';
		currentTheme = stored;
		applyTheme(stored);
	});

	function applyTheme(theme: 'light' | 'dark') {
		currentTheme = theme;
		if (typeof document !== 'undefined') {
			document.documentElement.setAttribute('data-theme', theme);
			document.body.className = 'theme-' + theme;
			try {
				localStorage.setItem('mrbiz_theme', theme);
			} catch {}
		}
	}

	function toggle() {
		const next = currentTheme === 'light' ? 'dark' : 'light';
		applyTheme(next);
	}
</script>

<button
	type="button"
	class="theme-toggle-btn"
	onclick={toggle}
	title={currentTheme === 'light' ? 'Switch to Dark Theme' : 'Switch to Light Theme'}
	aria-label="Toggle visual theme"
>
	{#if currentTheme === 'light'}
		<span class="theme-icon">🌙</span>
		<span class="theme-text">Dark</span>
	{:else}
		<span class="theme-icon">☀️</span>
		<span class="theme-text">Light</span>
	{/if}
</button>

<style>
	.theme-toggle-btn {
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
		background: var(--surface-secondary);
		border: 1px solid var(--border-default);
		border-radius: var(--radius-md);
		padding: 0.35rem 0.65rem;
		font-size: 0.75rem;
		font-weight: 600;
		color: var(--text-secondary);
		cursor: pointer;
		transition: all var(--transition-fast);
		user-select: none;
	}

	.theme-toggle-btn:hover {
		background: var(--border-default);
		color: var(--text-primary);
		border-color: var(--border-strong);
	}

	.theme-icon {
		font-size: 0.85rem;
		line-height: 1;
	}

	.theme-text {
		display: inline-block;
	}
</style>
