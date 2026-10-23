<script lang="ts">
	interface Props {
		variant?: 'success' | 'warning' | 'error' | 'info' | 'neutral' | 'locked' | 'approved' | 'pending' | 'rejected' | 'draft' | 'active';
		label: string;
		icon?: string;
		size?: 'sm' | 'md';
		showDot?: boolean;
	}

	let {
		variant = 'neutral',
		label,
		icon,
		size = 'md',
		showDot = true
	}: Props = $props();

	// Fallback icons for semantic accessibility when none provided
	let defaultIcon = $derived.by(() => {
		if (icon) return icon;
		switch (variant) {
			case 'success':
			case 'approved':
				return '✓';
			case 'warning':
			case 'pending':
				return '⏳';
			case 'error':
			case 'rejected':
				return '✕';
			case 'locked':
				return '🔒';
			case 'info':
			case 'active':
				return 'ℹ';
			default:
				return '';
		}
	});
</script>

<span class="badge badge-{variant} size-{size}">
	{#if defaultIcon}
		<span class="badge-icon" aria-hidden="true">{defaultIcon}</span>
	{:else if showDot}
		<span class="badge-dot" aria-hidden="true"></span>
	{/if}
	<span class="badge-label">{label}</span>
</span>

<style>
	.badge-icon {
		font-size: 0.75rem;
		line-height: 1;
		font-weight: 700;
	}
	.size-sm {
		font-size: 0.7rem;
		padding: 0.15rem 0.45rem;
	}
	.size-md {
		font-size: 0.78125rem;
		padding: 0.2rem 0.6rem;
	}
</style>
