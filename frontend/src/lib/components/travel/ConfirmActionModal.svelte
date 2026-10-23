<script lang="ts">
	interface Props {
		open: boolean;
		title?: string;
		message?: string;
		confirmText?: string;
		onclose: () => void;
		onconfirm?: (reason: string) => void;
	}

	let { 
		open = false, 
		title = 'Confirm Booking Cancellation', 
		message = 'Are you sure you want to cancel this booking? This action will release inventory and may trigger refund process.',
		confirmText = 'Cancel Booking',
		onclose, 
		onconfirm 
	}: Props = $props();

	let reason = $state('Customer requested cancellation due to personal reasons.');

	function handleConfirm() {
		onconfirm?.(reason);
		onclose();
	}
</script>

{#if open}
	<div class="modal-backdrop" onclick={(e) => { if (e.target === e.currentTarget) onclose(); }}>
		<div class="modal-card animate-in" role="dialog" aria-modal="true">
			<div class="modal-header">
				<div class="header-content">
					<div class="warn-icon-box">⚠️</div>
					<div>
						<h3 class="modal-title">{title}</h3>
						<span class="sub-text">Confirm Action &amp; Audit Trail</span>
					</div>
				</div>
				<button type="button" class="close-btn" onclick={onclose}>✕</button>
			</div>

			<div class="modal-body">
				<p class="warn-message">{message}</p>

				<div class="form-group mt-3">
					<label class="form-label" for="cancel-reason">Reason for Cancellation (Required for Audit)</label>
					<textarea 
						id="cancel-reason" 
						class="form-textarea" 
						rows="3" 
						placeholder="Please specify detailed cancellation reason..."
						bind:value={reason}
					></textarea>
				</div>
			</div>

			<div class="modal-footer">
				<button type="button" class="btn btn-secondary" onclick={onclose}>Cancel</button>
				<button type="button" class="btn btn-danger" onclick={handleConfirm}>
					{confirmText}
				</button>
			</div>
		</div>
	</div>
{/if}

<style>
	.modal-backdrop {
		position: fixed;
		inset: 0;
		background: rgba(15, 23, 42, 0.6);
		backdrop-filter: blur(4px);
		z-index: 9999;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 1.5rem;
	}

	.modal-card {
		background: #ffffff;
		border: 1px solid #e2e8f0;
		border-radius: 12px;
		width: 100%;
		max-width: 480px;
		display: flex;
		flex-direction: column;
		box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1);
		overflow: hidden;
		color: #0f172a;
	}

	.animate-in { animation: modalFadeIn 0.18s ease-out; }
	@keyframes modalFadeIn {
		from { opacity: 0; transform: scale(0.98); }
		to { opacity: 1; transform: scale(1); }
	}

	.modal-header {
		padding: 1.25rem 1.5rem;
		border-bottom: 1px solid #e2e8f0;
		display: flex;
		justify-content: space-between;
		align-items: center;
	}

	.header-content {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}

	.warn-icon-box {
		width: 36px;
		height: 36px;
		border-radius: 8px;
		background: #fef2f2;
		border: 1px solid #fecaca;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 1.1rem;
	}

	.modal-title { font-size: 1.1rem; font-weight: 800; margin: 0; color: #991b1b; }
	.sub-text { font-size: 0.75rem; color: #64748b; }
	.close-btn { background: none; border: none; font-size: 1.25rem; color: #64748b; cursor: pointer; }

	.modal-body { padding: 1.25rem 1.5rem; }

	.warn-message {
		font-size: 0.875rem;
		color: #334155;
		line-height: 1.5;
		margin: 0;
	}

	.form-group { display: flex; flex-direction: column; gap: 0.35rem; }
	.form-label { font-size: 0.75rem; font-weight: 700; color: #475569; text-transform: uppercase; }

	.form-textarea {
		width: 100%;
		padding: 0.55rem 0.75rem;
		border: 1px solid #cbd5e1;
		border-radius: 6px;
		font-size: 0.8125rem;
		color: #0f172a;
		background: #ffffff;
	}
	.form-textarea:focus { outline: none; border-color: #ef4444; box-shadow: 0 0 0 3px rgba(239, 68, 68, 0.1); }

	.modal-footer {
		padding: 1rem 1.5rem;
		border-top: 1px solid #e2e8f0;
		display: flex;
		justify-content: space-between;
		align-items: center;
		background: #ffffff;
	}

	.btn {
		font-size: 0.8125rem;
		font-weight: 600;
		padding: 0.5rem 1rem;
		border-radius: 8px;
		cursor: pointer;
		border: 1px solid transparent;
	}
	.btn-secondary { background: #f1f5f9; color: #334155; }
	.btn-danger { background: #dc2626; color: #ffffff; }
	.btn-danger:hover { background: #b91c1c; }

	.mt-3 { margin-top: 0.75rem; }
</style>
