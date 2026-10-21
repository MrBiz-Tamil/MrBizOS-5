<script lang="ts">
	import { toast } from '$lib/stores/toast.svelte';

	const icons = {
		success: '✓',
		error: '✕',
		warning: '⚠️',
		info: 'ℹ️'
	};
</script>

{#if toast.toasts.length > 0}
	<div class="toast-viewport" aria-live="polite">
		{#each toast.toasts as item (item.id)}
			<div class="toast-card {item.type} animate-slide-in" role="alert">
				<div class="toast-icon">
					{icons[item.type]}
				</div>
				<div class="toast-content">
					<div class="toast-title">{item.title}</div>
					{#if item.description}
						<div class="toast-desc">{item.description}</div>
					{/if}
				</div>
				<button
					type="button"
					class="toast-close"
					onclick={() => toast.dismiss(item.id)}
					aria-label="Dismiss notification"
				>
					✕
				</button>
			</div>
		{/each}
	</div>
{/if}

<style>
	.toast-viewport {
		position: fixed;
		bottom: 1.5rem;
		right: 1.5rem;
		z-index: 99999;
		display: flex;
		flex-direction: column;
		gap: 0.6rem;
		max-width: 420px;
		width: calc(100vw - 3rem);
		pointer-events: none;
	}

	.toast-card {
		pointer-events: auto;
		display: flex;
		align-items: flex-start;
		gap: 0.75rem;
		padding: 0.875rem 1rem;
		background: rgba(17, 24, 39, 0.95);
		backdrop-filter: blur(16px);
		-webkit-backdrop-filter: blur(16px);
		border-radius: 10px;
		border: 1px solid rgba(255, 255, 255, 0.12);
		box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.6), 0 0 15px rgba(0, 0, 0, 0.3);
		color: #f8fafc;
		transition: all 0.2s cubic-bezier(0.16, 1, 0.3, 1);
	}

	.toast-card.success {
		border-left: 4px solid #10b981;
		box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.6), 0 0 15px rgba(16, 185, 129, 0.15);
	}
	.toast-card.success .toast-icon {
		background: rgba(16, 185, 129, 0.2);
		color: #34d399;
	}

	.toast-card.error {
		border-left: 4px solid #f43f5e;
		box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.6), 0 0 15px rgba(244, 63, 94, 0.15);
	}
	.toast-card.error .toast-icon {
		background: rgba(244, 63, 94, 0.2);
		color: #fb7185;
	}

	.toast-card.warning {
		border-left: 4px solid #f59e0b;
		box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.6), 0 0 15px rgba(245, 158, 11, 0.15);
	}
	.toast-card.warning .toast-icon {
		background: rgba(245, 158, 11, 0.2);
		color: #fbbf24;
	}

	.toast-card.info {
		border-left: 4px solid #6366f1;
		box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.6), 0 0 15px rgba(99, 102, 241, 0.15);
	}
	.toast-card.info .toast-icon {
		background: rgba(99, 102, 241, 0.2);
		color: #818cf8;
	}

	.toast-icon {
		width: 24px;
		height: 24px;
		border-radius: 6px;
		display: flex;
		align-items: center;
		justify-content: center;
		font-weight: 800;
		font-size: 0.8125rem;
		flex-shrink: 0;
	}

	.toast-content {
		flex: 1;
		min-width: 0;
	}

	.toast-title {
		font-weight: 700;
		font-size: 0.875rem;
		color: #ffffff;
		line-height: 1.3;
	}

	.toast-desc {
		font-size: 0.8125rem;
		color: #94a3b8;
		margin-top: 0.2rem;
		line-height: 1.4;
	}

	.toast-close {
		background: none;
		border: none;
		color: #64748b;
		cursor: pointer;
		font-size: 0.875rem;
		padding: 0.1rem 0.3rem;
		border-radius: 4px;
		transition: all 0.15s ease;
		flex-shrink: 0;
	}

	.toast-close:hover {
		color: #ffffff;
		background: rgba(255, 255, 255, 0.1);
	}

	@keyframes slideIn {
		from {
			opacity: 0;
			transform: translateY(16px) scale(0.96);
		}
		to {
			opacity: 1;
			transform: translateY(0) scale(1);
		}
	}

	.animate-slide-in {
		animation: slideIn 0.25s cubic-bezier(0.16, 1, 0.3, 1) forwards;
	}
</style>
