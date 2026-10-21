<script lang="ts">
	import { onMount } from 'svelte';
	import { api, type PurchaseBill, type Supplier } from '$lib/api';

	let bills = $state<PurchaseBill[]>([]);
	let suppliers = $state<Supplier[]>([]);
	let loading = $state(true);
	let error = $state('');

	// Filters
	let searchQuery = $state('');
	let statusFilter = $state<string>('ALL');
	let selectedSupplierFilter = $state<string>('');

	// Payment Modal
	let showPaymentModal = $state(false);
	let selectedBillForPayment = $state<PurchaseBill | null>(null);
	let paymentAmount = $state('');
	let paymentMethod = $state('BANK_TRANSFER');
	let paymentReference = $state('');
	let paymentNotes = $state('');
	let paymentSubmitting = $state(false);
	let paymentError = $state('');

	// Bill Detail Modal
	let showDetailModal = $state(false);
	let activeBill = $state<PurchaseBill | null>(null);
	let detailLoading = $state(false);

	onMount(async () => {
		await loadData();
	});

	async function loadData() {
		loading = true;
		error = '';
		try {
			const [billList, supList] = await Promise.all([
				api.listPurchases({
					status: statusFilter === 'ALL' ? undefined : statusFilter,
					supplier_id: selectedSupplierFilter || undefined
				}),
				api.listSuppliers().catch(() => [])
			]);
			bills = billList;
			suppliers = supList;
		} catch (err: any) {
			error = err.message || 'Failed to load purchase bills';
		} finally {
			loading = false;
		}
	}

	async function handleFilterChange() {
		loading = true;
		try {
			bills = await api.listPurchases({
				status: statusFilter === 'ALL' ? undefined : statusFilter,
				supplier_id: selectedSupplierFilter || undefined
			});
		} catch (err: any) {
			error = err.message || 'Filter error';
		} finally {
			loading = false;
		}
	}

	let filteredBills = $derived(
		bills.filter((b) => {
			const q = searchQuery.toLowerCase().trim();
			if (!q) return true;
			return (
				b.bill_number.toLowerCase().includes(q) ||
				b.supplier_name.toLowerCase().includes(q) ||
				(b.po_number && b.po_number.toLowerCase().includes(q)) ||
				(b.vehicle_number && b.vehicle_number.toLowerCase().includes(q))
			);
		})
	);

	let totalPurchasesSum = $derived(
		bills.reduce((acc, b) => acc + (b.status !== 'CANCELLED' ? parseFloat(b.grand_total || '0') : 0), 0)
	);

	let totalUnpaidSum = $derived(
		bills.reduce((acc, b) => acc + (b.status !== 'CANCELLED' ? parseFloat(b.balance_due || '0') : 0), 0)
	);

	function openPaymentModal(b: PurchaseBill) {
		selectedBillForPayment = b;
		paymentAmount = b.balance_due;
		paymentMethod = 'BANK_TRANSFER';
		paymentReference = '';
		paymentNotes = '';
		paymentError = '';
		showPaymentModal = true;
	}

	async function submitPayment(e: Event) {
		e.preventDefault();
		if (!selectedBillForPayment) return;
		const amt = parseFloat(paymentAmount);
		if (isNaN(amt) || amt <= 0) {
			paymentError = 'Enter a valid payment amount';
			return;
		}
		paymentSubmitting = true;
		paymentError = '';
		try {
			await api.recordSupplierPayment(selectedBillForPayment.id, {
				amount: paymentAmount,
				payment_method: paymentMethod,
				reference_number: paymentReference,
				notes: paymentNotes
			});
			showPaymentModal = false;
			await loadData();
		} catch (err: any) {
			paymentError = err.message || 'Payment recording failed';
		} finally {
			paymentSubmitting = false;
		}
	}

	async function viewBillDetails(billId: string) {
		showDetailModal = true;
		detailLoading = true;
		try {
			activeBill = await api.getPurchase(billId);
		} catch (err: any) {
			error = err.message || 'Failed to fetch bill details';
		} finally {
			detailLoading = false;
		}
	}

	function formatCurrency(val: string | number | undefined) {
		const num = typeof val === 'number' ? val : parseFloat(val || '0');
		return new Intl.NumberFormat('en-IN', {
			style: 'currency',
			currency: 'INR',
			maximumFractionDigits: 2
		}).format(num);
	}
</script>

<div class="purchases-view">
	<!-- Top Bar -->
	<header class="header-section">
		<div>
			<div class="subtitle-badge">INWARD STOCK & TRADE INVENTORY</div>
			<h1 class="page-title">Purchase Invoices (Vendor Bills)</h1>
			<p class="page-desc">Manage supplier purchase bills, inward stock synchronization, and accounts payable.</p>
		</div>
		<div class="action-buttons">
			<a href="/suppliers" class="btn btn-secondary">
				<span class="icon">🏭</span> Suppliers Directory
			</a>
			<a href="/purchases/new" class="btn btn-primary">
				<span class="icon">+</span> New Inward Bill
			</a>
		</div>
	</header>

	<!-- KPI Summary Bar -->
	<div class="kpi-grid">
		<div class="kpi-card">
			<span class="kpi-label">Total Inward Purchases</span>
			<span class="kpi-value text-blue">{formatCurrency(totalPurchasesSum)}</span>
			<span class="kpi-subtext">{bills.length} bills recorded</span>
		</div>
		<div class="kpi-card highlight-card">
			<span class="kpi-label">Pending Supplier Due</span>
			<span class="kpi-value text-red">{formatCurrency(totalUnpaidSum)}</span>
			<span class="kpi-subtext">Accounts payable</span>
		</div>
		<div class="kpi-card">
			<span class="kpi-label">Fully Settled Bills</span>
			<span class="kpi-value text-green">
				{bills.filter((b) => b.status === 'PAID').length}
			</span>
			<span class="kpi-subtext">Complete payments</span>
		</div>
	</div>

	<!-- Controls & Filters -->
	<div class="controls-bar">
		<div class="search-wrap">
			<span class="search-icon">🔍</span>
			<input
				type="text"
				placeholder="Search by Bill #, Supplier name, PO #, or Vehicle..."
				bind:value={searchQuery}
				class="search-input"
			/>
		</div>
		<div class="filters">
			<select bind:value={selectedSupplierFilter} onchange={handleFilterChange} class="filter-select">
				<option value="">All Suppliers</option>
				{#each suppliers as s}
					<option value={s.id}>{s.name}</option>
				{/each}
			</select>
			<select bind:value={statusFilter} onchange={handleFilterChange} class="filter-select">
				<option value="ALL">All Statuses</option>
				<option value="RECEIVED">Received / Unpaid</option>
				<option value="PARTIALLY_PAID">Partially Paid</option>
				<option value="PAID">Fully Paid</option>
				<option value="CANCELLED">Cancelled</option>
			</select>
		</div>
	</div>

	{#if error}
		<div class="alert alert-error">{error}</div>
	{/if}

	<!-- Bills Table -->
	<div class="table-container">
		{#if loading}
			<div class="loading-state">
				<div class="spinner"></div>
				<p>Loading purchase bills...</p>
			</div>
		{:else if filteredBills.length === 0}
			<div class="empty-state">
				<span class="empty-icon">📥</span>
				<h3>No purchase bills recorded</h3>
				<p>Create an inward bill to update your inventory stock and record payables.</p>
				<a href="/purchases/new" class="btn btn-primary">+ Record Inward Bill</a>
			</div>
		{:else}
			<table class="data-table">
				<thead>
					<tr>
						<th>Bill Number</th>
						<th>Supplier / Vendor</th>
						<th>Date</th>
						<th style="text-align: right;">Taxable Amount</th>
						<th style="text-align: right;">Grand Total</th>
						<th style="text-align: right;">Balance Due</th>
						<th style="text-align: center;">Status</th>
						<th style="text-align: center;">Actions</th>
					</tr>
				</thead>
				<tbody>
					{#each filteredBills as b}
						<tr>
							<td>
								<button class="link-btn" onclick={() => viewBillDetails(b.id)}>
									<strong>{b.bill_number}</strong>
								</button>
								{#if b.po_number}
									<span class="sub-label">PO: {b.po_number}</span>
								{/if}
							</td>
							<td>
								<div class="sup-cell">
									<span class="sup-name">{b.supplier_name}</span>
									{#if b.supplier_gstin}
										<span class="sup-gstin">{b.supplier_gstin}</span>
									{/if}
								</div>
							</td>
							<td>
								<div class="date-cell">
									<span>{new Date(b.bill_date).toLocaleDateString('en-IN')}</span>
									<small class="text-muted">Due: {new Date(b.due_date).toLocaleDateString('en-IN')}</small>
								</div>
							</td>
							<td style="text-align: right; color: #94a3b8;">
								{formatCurrency(b.taxable_amount)}
							</td>
							<td style="text-align: right; font-weight: 700; color: #f8fafc;">
								{formatCurrency(b.grand_total)}
							</td>
							<td style="text-align: right;">
								{#if parseFloat(b.balance_due) > 0.01}
									<span class="text-red font-bold">{formatCurrency(b.balance_due)}</span>
								{:else}
									<span class="text-green">Settled</span>
								{/if}
							</td>
							<td style="text-align: center;">
								{#if b.status === 'PAID'}
									<span class="status-badge badge-paid">PAID</span>
								{:else if b.status === 'PARTIALLY_PAID'}
									<span class="status-badge badge-partial">PARTIAL</span>
								{:else if b.status === 'RECEIVED'}
									<span class="status-badge badge-due">RECEIVED</span>
								{:else}
									<span class="status-badge badge-cancel">CANCELLED</span>
								{/if}
							</td>
							<td style="text-align: center;">
								<div class="action-cell">
									<button class="btn btn-xs btn-outline" onclick={() => viewBillDetails(b.id)}>
										View
									</button>
									{#if parseFloat(b.balance_due) > 0.01 && b.status !== 'CANCELLED'}
										<button class="btn btn-xs btn-primary" onclick={() => openPaymentModal(b)}>
											Pay
										</button>
									{/if}
								</div>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
	</div>
</div>

<!-- Record Payment Modal -->
{#if showPaymentModal && selectedBillForPayment}
	<div class="modal-backdrop">
		<div class="modal-card">
			<div class="modal-header">
				<div>
					<h3>Record Supplier Payment</h3>
					<span class="company-sub">Bill #{selectedBillForPayment.bill_number} • {selectedBillForPayment.supplier_name}</span>
				</div>
				<button class="close-btn" onclick={() => (showPaymentModal = false)}>✕</button>
			</div>
			<form onsubmit={submitPayment}>
				<div class="modal-body">
					{#if paymentError}
						<div class="alert alert-error">{paymentError}</div>
					{/if}
					<div class="due-banner">
						<span>Outstanding Bill Due:</span>
						<strong class="text-red">{formatCurrency(selectedBillForPayment.balance_due)}</strong>
					</div>
					<div class="form-group">
						<label for="pay-amount">Payment Amount (₹) *</label>
						<input
							id="pay-amount"
							type="number"
							step="0.01"
							max={selectedBillForPayment.balance_due}
							bind:value={paymentAmount}
							required
						/>
					</div>
					<div class="form-group">
						<label for="pay-mode">Payment Method</label>
						<select id="pay-mode" bind:value={paymentMethod} class="select-input">
							<option value="BANK_TRANSFER">Bank Transfer (NEFT/RTGS/IMPS)</option>
							<option value="UPI">UPI / QR Code</option>
							<option value="CASH">Cash</option>
							<option value="CHEQUE">Cheque</option>
							<option value="OTHER">Other</option>
						</select>
					</div>
					<div class="form-group">
						<label for="pay-ref">Transaction / Reference Number (UTR)</label>
						<input
							id="pay-ref"
							type="text"
							bind:value={paymentReference}
							placeholder="e.g. UTR1293848123 or Cheque No"
						/>
					</div>
					<div class="form-group">
						<label for="pay-notes">Payment Notes</label>
						<input
							id="pay-notes"
							type="text"
							bind:value={paymentNotes}
							placeholder="e.g. Cleared via HDFC Corporate Account"
						/>
					</div>
				</div>
				<div class="modal-footer">
					<button type="button" class="btn btn-secondary" onclick={() => (showPaymentModal = false)}>
						Cancel
					</button>
					<button type="submit" class="btn btn-primary" disabled={paymentSubmitting}>
						{paymentSubmitting ? 'Recording...' : 'Confirm Payment'}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}

<!-- Bill Detail Drawer / Modal -->
{#if showDetailModal}
	<div class="modal-backdrop">
		<div class="modal-card modal-lg">
			<div class="modal-header">
				<div>
					<h3>Purchase Bill Summary</h3>
					{#if activeBill}
						<span class="company-sub">#{activeBill.bill_number} • {activeBill.supplier_name}</span>
					{/if}
				</div>
				<button class="close-btn" onclick={() => (showDetailModal = false)}>✕</button>
			</div>
			<div class="modal-body">
				{#if detailLoading}
					<div class="loading-state">
						<div class="spinner"></div>
						<p>Loading bill details...</p>
					</div>
				{:else if activeBill}
					<div class="bill-meta-grid">
						<div>
							<span class="meta-label">Supplier Name:</span>
							<strong>{activeBill.supplier_name}</strong>
						</div>
						<div>
							<span class="meta-label">Supplier GSTIN:</span>
							<span>{activeBill.supplier_gstin || 'Unregistered'}</span>
						</div>
						<div>
							<span class="meta-label">Bill Date:</span>
							<span>{new Date(activeBill.bill_date).toLocaleDateString('en-IN')}</span>
						</div>
						<div>
							<span class="meta-label">Due Date:</span>
							<span>{new Date(activeBill.due_date).toLocaleDateString('en-IN')}</span>
						</div>
						{#if activeBill.vehicle_number}
							<div>
								<span class="meta-label">Vehicle No:</span>
								<span>{activeBill.vehicle_number}</span>
							</div>
						{/if}
						{#if activeBill.po_number}
							<div>
								<span class="meta-label">PO Reference:</span>
								<span>{activeBill.po_number}</span>
							</div>
						{/if}
					</div>

					<h4 style="margin: 1rem 0 0.5rem 0; color: #cbd5e1;">Inward Line Items & Batches</h4>
					<table class="data-table">
						<thead>
							<tr>
								<th>Item Description</th>
								<th>Batch No</th>
								<th>Expiry</th>
								<th style="text-align: right;">Qty</th>
								<th style="text-align: right;">Unit Rate</th>
								<th style="text-align: right;">Tax Rate</th>
								<th style="text-align: right;">Total Amount</th>
							</tr>
						</thead>
						<tbody>
							{#if activeBill.items}
								{#each activeBill.items as item}
									<tr>
										<td>
											<strong>{item.item_description}</strong>
											{#if item.hsn_sac}<span class="text-muted" style="font-size: 0.75rem;"> (HSN: {item.hsn_sac})</span>{/if}
										</td>
										<td>
											{#if item.batch_number}
												<span class="badge-batch">{item.batch_number}</span>
											{:else}
												<span class="text-muted">—</span>
											{/if}
										</td>
										<td>
											{#if item.expiry_date}
												<span>{new Date(item.expiry_date).toLocaleDateString('en-IN')}</span>
											{:else}
												<span class="text-muted">—</span>
											{/if}
										</td>
										<td style="text-align: right;">
											<strong>{item.quantity}</strong> {item.unit_code}
										</td>
										<td style="text-align: right;">{formatCurrency(item.purchase_price)}</td>
										<td style="text-align: right;">{item.tax_rate}%</td>
										<td style="text-align: right; font-weight: 600;">{formatCurrency(item.total_amount)}</td>
									</tr>
								{/each}
							{/if}
						</tbody>
					</table>

					<!-- Tax & Grand Totals Breakdown -->
					<div class="totals-box">
						<div class="total-line">
							<span>Subtotal:</span>
							<span>{formatCurrency(activeBill.subtotal)}</span>
						</div>
						{#if parseFloat(activeBill.discount_total) > 0}
							<div class="total-line text-green">
								<span>Discount:</span>
								<span>-{formatCurrency(activeBill.discount_total)}</span>
							</div>
						{/if}
						<div class="total-line">
							<span>Taxable Value:</span>
							<span>{formatCurrency(activeBill.taxable_amount)}</span>
						</div>
						{#if activeBill.is_inter_state}
							<div class="total-line">
								<span>IGST:</span>
								<span>{formatCurrency(activeBill.igst_total)}</span>
							</div>
						{:else}
							<div class="total-line">
								<span>CGST:</span>
								<span>{formatCurrency(activeBill.cgst_total)}</span>
							</div>
							<div class="total-line">
								<span>SGST:</span>
								<span>{formatCurrency(activeBill.sgst_total)}</span>
							</div>
						{/if}
						{#if parseFloat(activeBill.round_off) !== 0}
							<div class="total-line text-muted">
								<span>Round Off:</span>
								<span>{formatCurrency(activeBill.round_off)}</span>
							</div>
						{/if}
						<div class="total-line grand-total-line">
							<strong>Grand Total:</strong>
							<strong>{formatCurrency(activeBill.grand_total)}</strong>
						</div>
						<div class="total-line">
							<span>Paid Amount:</span>
							<span class="text-green">{formatCurrency(activeBill.amount_paid)}</span>
						</div>
						<div class="total-line">
							<strong>Balance Due:</strong>
							<strong class="text-red">{formatCurrency(activeBill.balance_due)}</strong>
						</div>
					</div>
				{/if}
			</div>
			<div class="modal-footer">
				<button class="btn btn-secondary" onclick={() => (showDetailModal = false)}>Close</button>
			</div>
		</div>
	</div>
{/if}

<style>
	.purchases-view {
		padding: 2rem;
		max-width: 1400px;
		margin: 0 auto;
		display: flex;
		flex-direction: column;
		gap: 1.5rem;
	}

	.header-section {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		flex-wrap: wrap;
		gap: 1rem;
	}

	.subtitle-badge {
		font-size: 0.75rem;
		font-weight: 700;
		color: #6366f1;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		margin-bottom: 0.25rem;
	}

	.page-title {
		font-size: 1.75rem;
		font-weight: 700;
		color: #f8fafc;
		margin: 0;
	}

	.page-desc {
		font-size: 0.875rem;
		color: #94a3b8;
		margin-top: 0.25rem;
	}

	.action-buttons {
		display: flex;
		gap: 0.75rem;
	}

	.btn {
		display: inline-flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.625rem 1.25rem;
		border-radius: 8px;
		font-size: 0.875rem;
		font-weight: 600;
		cursor: pointer;
		border: none;
		text-decoration: none;
		transition: all 0.15s ease;
	}

	.btn-primary {
		background: #6366f1;
		color: #ffffff;
	}
	.btn-primary:hover {
		background: #4f46e5;
	}

	.btn-secondary {
		background: rgba(255, 255, 255, 0.08);
		color: #e2e8f0;
		border: 1px solid rgba(255, 255, 255, 0.15);
	}
	.btn-secondary:hover {
		background: rgba(255, 255, 255, 0.14);
	}

	.btn-outline {
		background: transparent;
		color: #a5b4fc;
		border: 1px solid rgba(165, 180, 252, 0.3);
	}
	.btn-outline:hover {
		background: rgba(165, 180, 252, 0.1);
	}

	.btn-xs {
		padding: 0.25rem 0.625rem;
		font-size: 0.75rem;
		border-radius: 6px;
	}

	.link-btn {
		background: none;
		border: none;
		color: #60a5fa;
		cursor: pointer;
		padding: 0;
		font-size: 0.875rem;
		text-decoration: underline;
	}

	/* KPI Grid */
	.kpi-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
		gap: 1rem;
	}

	.kpi-card {
		background: rgba(30, 41, 59, 0.6);
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 12px;
		padding: 1.25rem;
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}

	.highlight-card {
		background: linear-gradient(135deg, rgba(99, 102, 241, 0.15), rgba(239, 68, 68, 0.1));
		border-color: rgba(99, 102, 241, 0.3);
	}

	.kpi-label {
		font-size: 0.8125rem;
		color: #94a3b8;
		font-weight: 500;
	}

	.kpi-value {
		font-size: 1.625rem;
		font-weight: 700;
		color: #f8fafc;
	}

	.kpi-subtext {
		font-size: 0.75rem;
		color: #64748b;
	}

	.text-red {
		color: #f87171;
	}
	.text-blue {
		color: #60a5fa;
	}
	.text-green {
		color: #4ade80;
	}
	.text-muted {
		color: #94a3b8;
	}
	.font-bold {
		font-weight: 700;
	}

	/* Controls */
	.controls-bar {
		display: flex;
		justify-content: space-between;
		align-items: center;
		flex-wrap: wrap;
		gap: 1rem;
	}

	.search-wrap {
		position: relative;
		flex: 1;
		min-width: 280px;
	}

	.search-icon {
		position: absolute;
		left: 0.875rem;
		top: 50%;
		transform: translateY(-50%);
		font-size: 0.875rem;
		color: #64748b;
	}

	.search-input {
		width: 100%;
		padding: 0.625rem 0.875rem 0.625rem 2.5rem;
		background: rgba(15, 23, 42, 0.6);
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: 8px;
		color: #f8fafc;
		font-size: 0.875rem;
		outline: none;
	}
	.search-input:focus {
		border-color: #6366f1;
	}

	.filters {
		display: flex;
		gap: 0.5rem;
	}

	.filter-select {
		padding: 0.5rem 0.875rem;
		background: rgba(15, 23, 42, 0.8);
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: 8px;
		color: #f8fafc;
		font-size: 0.875rem;
		outline: none;
	}

	/* Table */
	.table-container {
		background: rgba(30, 41, 59, 0.5);
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 12px;
		overflow-x: auto;
	}

	.data-table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.875rem;
		text-align: left;
	}

	.data-table th {
		background: rgba(15, 23, 42, 0.6);
		padding: 0.875rem 1rem;
		font-weight: 600;
		color: #94a3b8;
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
	}

	.data-table td {
		padding: 0.875rem 1rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.05);
		vertical-align: middle;
	}

	.sub-label {
		display: block;
		font-size: 0.75rem;
		color: #64748b;
	}

	.sup-cell,
	.date-cell {
		display: flex;
		flex-direction: column;
		gap: 0.125rem;
	}

	.sup-name {
		font-weight: 600;
		color: #f1f5f9;
	}

	.sup-gstin {
		font-family: monospace;
		font-size: 0.6875rem;
		color: #94a3b8;
	}

	.status-badge {
		font-size: 0.6875rem;
		font-weight: 700;
		padding: 0.2rem 0.5rem;
		border-radius: 4px;
		letter-spacing: 0.04em;
	}

	.badge-paid {
		background: rgba(34, 197, 94, 0.15);
		color: #4ade80;
	}
	.badge-partial {
		background: rgba(234, 179, 8, 0.15);
		color: #facc15;
	}
	.badge-due {
		background: rgba(59, 130, 246, 0.15);
		color: #60a5fa;
	}
	.badge-cancel {
		background: rgba(239, 68, 68, 0.15);
		color: #f87171;
	}

	.action-cell {
		display: flex;
		gap: 0.375rem;
		justify-content: center;
	}

	.badge-batch {
		background: rgba(99, 102, 241, 0.15);
		color: #a5b4fc;
		padding: 0.125rem 0.375rem;
		border-radius: 4px;
		font-family: monospace;
		font-size: 0.75rem;
	}

	/* Modal */
	.modal-backdrop {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.75);
		backdrop-filter: blur(4px);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 100;
		padding: 1rem;
	}

	.modal-card {
		background: #1e293b;
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: 12px;
		width: 100%;
		max-width: 550px;
		max-height: 90vh;
		overflow-y: auto;
		display: flex;
		flex-direction: column;
		box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
	}

	.modal-lg {
		max-width: 900px;
	}

	.modal-header {
		padding: 1.25rem 1.5rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
		display: flex;
		justify-content: space-between;
		align-items: center;
	}

	.modal-header h3 {
		margin: 0;
		font-size: 1.125rem;
		color: #f8fafc;
	}

	.close-btn {
		background: transparent;
		border: none;
		color: #94a3b8;
		font-size: 1.25rem;
		cursor: pointer;
	}

	.modal-body {
		padding: 1.5rem;
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}

	.modal-footer {
		padding: 1rem 1.5rem;
		border-top: 1px solid rgba(255, 255, 255, 0.08);
		display: flex;
		justify-content: flex-end;
		gap: 0.75rem;
	}

	.due-banner {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 0.75rem 1rem;
		background: rgba(239, 68, 68, 0.1);
		border: 1px solid rgba(239, 68, 68, 0.25);
		border-radius: 8px;
		font-size: 0.875rem;
	}

	.form-group {
		display: flex;
		flex-direction: column;
		gap: 0.375rem;
	}

	.form-group label {
		font-size: 0.8125rem;
		font-weight: 500;
		color: #cbd5e1;
	}

	.form-group input,
	.select-input {
		padding: 0.5rem 0.75rem;
		background: rgba(15, 23, 42, 0.6);
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: 6px;
		color: #f8fafc;
		font-size: 0.875rem;
		outline: none;
	}

	.alert-error {
		background: rgba(239, 68, 68, 0.15);
		border: 1px solid rgba(239, 68, 68, 0.3);
		color: #f87171;
		padding: 0.75rem;
		border-radius: 6px;
		font-size: 0.875rem;
	}

	.bill-meta-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
		gap: 0.75rem;
		padding: 1rem;
		background: rgba(15, 23, 42, 0.5);
		border-radius: 8px;
		font-size: 0.8125rem;
	}

	.meta-label {
		color: #94a3b8;
		display: block;
		font-size: 0.75rem;
	}

	.totals-box {
		margin-top: 1rem;
		margin-left: auto;
		width: 100%;
		max-width: 320px;
		display: flex;
		flex-direction: column;
		gap: 0.375rem;
		padding: 1rem;
		background: rgba(15, 23, 42, 0.5);
		border-radius: 8px;
		font-size: 0.875rem;
	}

	.total-line {
		display: flex;
		justify-content: space-between;
	}

	.grand-total-line {
		font-size: 1rem;
		border-top: 1px solid rgba(255, 255, 255, 0.1);
		border-bottom: 1px solid rgba(255, 255, 255, 0.1);
		padding: 0.375rem 0;
		color: #f8fafc;
	}

	.loading-state,
	.empty-state {
		padding: 3rem;
		text-align: center;
		color: #94a3b8;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.75rem;
	}

	.empty-icon {
		font-size: 2.5rem;
	}

	.spinner {
		width: 32px;
		height: 32px;
		border: 3px solid rgba(99, 102, 241, 0.2);
		border-top-color: #6366f1;
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}
</style>
