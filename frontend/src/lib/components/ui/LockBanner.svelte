<script lang="ts">
	interface Props {
		title?: string;
		reason?: string;
		scope?: string;
		lockedBy?: string;
		canUnlock?: boolean;
		onunlock?: () => void;
	}

	let {
		title = 'Resource Locked & Protected',
		reason = 'Financial period closed or operational baseline milestone locked.',
		scope = 'Read-Only presentation active across all tasks and budgets.',
		lockedBy = 'Project Controller',
		canUnlock = true,
		onunlock
	}: Props = $props();
</script>

<div class="project-lock-banner" role="alert">
	<div class="project-lock-info">
		<span class="lock-glyph" aria-hidden="true">🔒</span>
		<div class="lock-text">
			<div class="lock-heading">
				<strong class="lock-status-tag">LOCKED</strong>
				<span class="lock-title">{title}</span>
			</div>
			<p class="lock-details">
				{reason} <span class="lock-scope">Scope: <strong>{scope}</strong></span> (Locked by {lockedBy})
			</p>
		</div>
	</div>

	{#if canUnlock}
		<button
			type="button"
			class="btn btn-outline btn-sm unlock-btn"
			onclick={onunlock}
			title="Authorized project unlock action"
		>
			🔑 Authorized Unlock
		</button>
	{/if}
</div>

<style>
	.lock-glyph {
		font-size: 1.35rem;
		flex-shrink: 0;
	}
	.lock-text {
		display: flex;
		flex-direction: column;
		gap: 0.15rem;
	}
	.lock-heading {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 0.84375rem;
		font-weight: 600;
	}
	.lock-status-tag {
		background: var(--status-warning);
		color: #ffffff;
		padding: 0.1rem 0.4rem;
		border-radius: var(--radius-xs);
		font-size: 0.6875rem;
		letter-spacing: 0.04em;
	}
	.lock-details {
		font-size: 0.78125rem;
		margin: 0;
		opacity: 0.95;
	}
	.lock-scope {
		margin-left: 0.35rem;
	}
	.unlock-btn {
		background: var(--surface-main);
		border-color: var(--status-warning-border);
		color: var(--status-warning);
		font-weight: 600;
		flex-shrink: 0;
	}
	.unlock-btn:hover {
		background: var(--status-warning);
		color: #ffffff;
	}
</style>
