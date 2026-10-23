<script lang="ts">
	import { toast } from '$lib/stores/toast.svelte';

	interface Props {
		open: boolean;
		booking?: any;
		onclose: () => void;
	}

	let { open = false, booking, onclose }: Props = $props();

	let activeTab = $state<'payment' | 'preview'>('payment');
	let invoiceNo = $state('INV-TRA-202601');
	let customerName = $state('Rahul Sharma');
	let grandTotal = $state('₹ 1,65,000');

	let paymentType = $state<'full' | 'advance' | 'installment'>('advance');
	let amountReceived = $state('90000');
	let paymentDate = $state('15 Oct 2026');
	let paymentMethod = $state('Bank Transfer');
	let referenceNo = $state('UTR12345678');
	let notes = $state('Token advance confirmed via HDFC bank transfer');

	function handleRecordPayment() {
		toast.success(`Payment of ₹${amountReceived} recorded for invoice ${invoiceNo}. Official tax receipt generated!`);
		onclose();
	}
</script>

{#if open}
	<div class="modal-backdrop" onclick={(e) => { if (e.target === e.currentTarget) onclose(); }}>
		<div class="modal-card animate-in" role="dialog" aria-modal="true">
			<!-- Header -->
			<div class="modal-header">
				<div>
					<h3 class="modal-title">Payment &amp; Invoice Settlement</h3>
					<span class="sub-text">Record advance, installments, bank UTR reference and tax invoices</span>
				</div>
				<button type="button" class="close-btn" onclick={onclose}>✕</button>
			</div>

			<!-- Tabs -->
			<div class="modal-tabs">
				<button 
					type="button" 
					class="tab-btn" 
					class:active={activeTab === 'payment'}
					onclick={() => (activeTab = 'payment')}
				>
					Payment Details
				</button>
				<button 
					type="button" 
					class="tab-btn" 
					class:active={activeTab === 'preview'}
					onclick={() => (activeTab = 'preview')}
				>
					Invoice Preview
				</button>
			</div>

			<!-- Body -->
			<div class="modal-body">
				{#if activeTab === 'payment'}
					<div class="invoice-summary-bar">
						<div class="sum-col">
							<span class="sum-label">Invoice No</span>
							<span class="sum-val font-mono">{invoiceNo}</span>
						</div>
						<div class="sum-col">
							<span class="sum-label">Customer</span>
							<span class="sum-val">{customerName}</span>
						</div>
						<div class="sum-col text-right">
							<span class="sum-label">Total Amount</span>
							<span class="sum-val text-primary font-bold">{grandTotal}</span>
						</div>
					</div>

					<div class="form-section mt-4">
						<label class="form-label">Payment Type</label>
						<div class="pill-group">
							<button 
								type="button" 
								class="pill-opt" 
								class:active={paymentType === 'full'}
								onclick={() => (paymentType = 'full')}
							>
								Full Payment
							</button>
							<button 
								type="button" 
								class="pill-opt" 
								class:active={paymentType === 'advance'}
								onclick={() => (paymentType = 'advance')}
							>
								Advance Token
							</button>
							<button 
								type="button" 
								class="pill-opt" 
								class:active={paymentType === 'installment'}
								onclick={() => (paymentType = 'installment')}
							>
								Installment / Part
							</button>
						</div>

						<div class="grid-2-cols mt-3">
							<div class="form-group">
								<label class="form-label" for="amt-rec">Amount Received (₹)</label>
								<input id="amt-rec" type="number" class="form-input font-mono font-bold" bind:value={amountReceived} />
							</div>
							<div class="form-group">
								<label class="form-label" for="pay-date">Payment Date</label>
								<input id="pay-date" type="text" class="form-input" bind:value={paymentDate} />
							</div>
						</div>

						<div class="grid-2-cols mt-3">
							<div class="form-group">
								<label class="form-label" for="pay-method">Payment Method</label>
								<select id="pay-method" class="form-select" bind:value={paymentMethod}>
									<option value="Bank Transfer">NEFT / RTGS / IMPS Bank Transfer</option>
									<option value="UPI">UPI / GPay / PhonePe</option>
									<option value="Credit Card">Corporate / Personal Credit Card</option>
									<option value="Cheque">Cheque</option>
									<option value="Cash">Cash Receipt</option>
								</select>
							</div>
							<div class="form-group">
								<label class="form-label" for="ref-no">Reference / UTR No</label>
								<input id="ref-no" type="text" class="form-input font-mono" placeholder="UTR or Transaction ID" bind:value={referenceNo} />
							</div>
						</div>

						<div class="form-group mt-3">
							<label class="form-label" for="pay-notes">Internal Accounting Notes</label>
							<input id="pay-notes" type="text" class="form-input" placeholder="e.g. Cleared by HDFC current account" bind:value={notes} />
						</div>
					</div>

				{:else if activeTab === 'preview'}
					<div class="preview-receipt-card">
						<div class="receipt-header">
							<div>
								<div class="receipt-brand">MRBIZ GLOBAL TRAVEL &amp; TOURS</div>
								<div class="receipt-sub">GSTIN: 33AAACM1234F1Z8 • Chennai Central Hub</div>
							</div>
							<div class="text-right">
								<span class="badge-receipt">PAYMENT RECEIPT</span>
								<div class="font-mono text-xs text-muted mt-1">NO: RCP-2026-0042</div>
							</div>
						</div>

						<div class="receipt-table mt-3">
							<div class="r-table-row r-head">
								<span>Description</span>
								<span class="text-right">Amount</span>
							</div>
							<div class="r-table-row">
								<span>Kashmir Valley Paradise (4D/5N) Tour Package</span>
								<span class="text-right font-mono">₹ 1,65,000</span>
							</div>
							<div class="r-table-row highlight">
								<span>Advance Payment Received (Bank Transfer - {referenceNo})</span>
								<span class="text-right font-mono font-bold text-emerald">₹ 90,000</span>
							</div>
							<div class="r-table-row due">
								<span>Balance Outstanding Payable Before Departure</span>
								<span class="text-right font-mono font-bold text-amber">₹ 75,000</span>
							</div>
						</div>
					</div>
				{/if}
			</div>

			<!-- Footer -->
			<div class="modal-footer">
				<button type="button" class="btn btn-secondary" onclick={onclose}>Cancel</button>
				<button type="button" class="btn btn-primary" onclick={handleRecordPayment}>
					💳 Record Payment &amp; Issue Receipt
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
		max-width: 640px;
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

	.modal-title { font-size: 1.25rem; font-weight: 800; margin: 0; }
	.sub-text { font-size: 0.8125rem; color: #64748b; }
	.close-btn { background: none; border: none; font-size: 1.25rem; color: #64748b; cursor: pointer; }

	.modal-tabs {
		display: flex;
		background: #f8fafc;
		border-bottom: 1px solid #e2e8f0;
		padding: 0 1rem;
	}

	.tab-btn {
		padding: 0.65rem 1rem;
		background: transparent;
		border: none;
		border-bottom: 2px solid transparent;
		font-size: 0.8125rem;
		font-weight: 600;
		color: #64748b;
		cursor: pointer;
	}
	.tab-btn.active {
		color: #2563eb;
		border-bottom-color: #2563eb;
	}

	.modal-body { padding: 1.5rem; }

	.invoice-summary-bar {
		display: flex;
		justify-content: space-between;
		background: #f8fafc;
		border: 1px solid #e2e8f0;
		border-radius: 8px;
		padding: 0.85rem 1.25rem;
	}

	.sum-col { display: flex; flex-direction: column; }
	.sum-label { font-size: 0.7rem; font-weight: 700; color: #64748b; text-transform: uppercase; }
	.sum-val { font-size: 0.9375rem; font-weight: 700; color: #0f172a; }

	.pill-group {
		display: flex;
		gap: 0.5rem;
		margin-top: 0.35rem;
	}

	.pill-opt {
		padding: 0.4rem 0.85rem;
		border-radius: 6px;
		border: 1px solid #cbd5e1;
		background: #ffffff;
		font-size: 0.8125rem;
		font-weight: 600;
		color: #475569;
		cursor: pointer;
	}
	.pill-opt.active {
		background: #eff6ff;
		color: #2563eb;
		border-color: #bfdbfe;
	}

	.grid-2-cols {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 0.75rem;
	}

	.form-group {
		display: flex;
		flex-direction: column;
		gap: 0.35rem;
	}

	.form-label {
		font-size: 0.75rem;
		font-weight: 700;
		color: #475569;
		text-transform: uppercase;
		letter-spacing: 0.04em;
	}

	.form-input, .form-select {
		width: 100%;
		padding: 0.55rem 0.75rem;
		border: 1px solid #cbd5e1;
		border-radius: 6px;
		font-size: 0.875rem;
		color: #0f172a;
		background: #ffffff;
	}

	.preview-receipt-card {
		background: #ffffff;
		border: 1px solid #e2e8f0;
		border-radius: 8px;
		padding: 1.25rem;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.03);
	}

	.receipt-header {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		border-bottom: 2px solid #e2e8f0;
		padding-bottom: 0.75rem;
	}

	.receipt-brand { font-size: 1rem; font-weight: 800; color: #0f172a; }
	.receipt-sub { font-size: 0.75rem; color: #64748b; }

	.badge-receipt {
		font-size: 0.7rem;
		font-weight: 800;
		background: #ecfdf5;
		color: #059669;
		padding: 0.2rem 0.5rem;
		border-radius: 4px;
	}

	.receipt-table {
		display: flex;
		flex-direction: column;
		font-size: 0.8125rem;
	}

	.r-table-row {
		display: flex;
		justify-content: space-between;
		padding: 0.6rem 0.75rem;
		border-bottom: 1px solid #f1f5f9;
	}
	.r-table-row.r-head { font-weight: 700; color: #64748b; background: #f8fafc; }
	.r-table-row.highlight { background: #ecfdf5; color: #065f46; font-weight: 600; }
	.r-table-row.due { background: #fffbeb; color: #92400e; font-weight: 600; }

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
	.btn-primary { background: #2563eb; color: #ffffff; }
	.btn-primary:hover { background: #1d4ed8; }

	.mt-3 { margin-top: 0.75rem; }
	.mt-4 { margin-top: 1rem; }
	.text-right { text-align: right; }
	.text-primary { color: #2563eb; }
	.text-emerald { color: #059669; }
	.text-amber { color: #d97706; }
	.font-bold { font-weight: 700; }
</style>
