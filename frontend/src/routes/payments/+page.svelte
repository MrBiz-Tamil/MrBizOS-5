<script lang="ts">
	import { onMount } from 'svelte';
	import { api, type Payment, type Tenant, type Invoice, type Customer } from '$lib/api';
	import { toast } from '$lib/stores/toast.svelte';

	let payments = $state<Payment[]>([]);
	let company = $state<Tenant | null>(null);
	let invoices = $state<Invoice[]>([]);
	let customers = $state<Customer[]>([]);
	let loading = $state(true);
	let error = $state('');

	// Search & Filters
	let searchQuery = $state('');
	let methodFilter = $state<string>('ALL');
	let dateFilter = $state<'ALL' | 'TODAY' | 'MONTH' | 'LAST30'>('ALL');

	// Slide-Over Voucher Drawer
	let showDrawer = $state(false);
	let selectedPayment = $state<Payment | null>(null);

	onMount(async () => {
		await loadData();
	});

	async function loadData() {
		loading = true;
		error = '';
		try {
			const [payList, comp, invList, custList] = await Promise.all([
				api.listPayments().catch(() => []),
				api.getCompany().catch(() => null),
				api.listInvoices().catch(() => []),
				api.listCustomers().catch(() => [])
			]);
			payments = payList;
			company = comp;
			invoices = invList;
			customers = custList;
		} catch (err: any) {
			error = err.message || 'Failed to fetch payment records';
			toast.error('Could not load payments', error);
		} finally {
			loading = false;
		}
	}

	function formatCurrency(val: string | number | undefined): string {
		if (val === undefined || val === null) return '₹0.00';
		const num = typeof val === 'string' ? parseFloat(val) : val;
		return new Intl.NumberFormat('en-IN', { style: 'currency', currency: 'INR' }).format(isNaN(num) ? 0 : num);
	}

	// Helper to lookup Invoice details
	function getInvoice(invoiceId: string | undefined): Invoice | undefined {
		if (!invoiceId) return undefined;
		return invoices.find((inv) => inv.id === invoiceId);
	}

	function getCustomerName(payment: Payment): string {
		const inv = getInvoice(payment.invoice_id);
		if (inv?.customer_name) return inv.customer_name;
		if (payment.customer_id) {
			const cust = customers.find((c) => c.id === payment.customer_id);
			if (cust) return cust.name;
		}
		return 'Direct Customer';
	}

	// Filtered Payments
	let filteredPayments = $derived(
		payments.filter((p) => {
			const q = searchQuery.toLowerCase().trim();
			const custName = getCustomerName(p).toLowerCase();
			const inv = getInvoice(p.invoice_id);
			const invNumber = inv?.invoice_number?.toLowerCase() || '';

			const matchesQuery =
				!q ||
				p.payment_number.toLowerCase().includes(q) ||
				(p.reference_number && p.reference_number.toLowerCase().includes(q)) ||
				(p.notes && p.notes.toLowerCase().includes(q)) ||
				custName.includes(q) ||
				invNumber.includes(q);

			const matchesMethod = methodFilter === 'ALL' || p.payment_method?.toUpperCase() === methodFilter;

			let matchesDate = true;
			const pDate = new Date(p.payment_date).getTime();
			const now = Date.now();
			if (dateFilter === 'TODAY') {
				const startOfToday = new Date().setHours(0, 0, 0, 0);
				matchesDate = pDate >= startOfToday;
			} else if (dateFilter === 'LAST30') {
				matchesDate = now - pDate <= 30 * 24 * 60 * 60 * 1000;
			} else if (dateFilter === 'MONTH') {
				const d = new Date(p.payment_date);
				const current = new Date();
				matchesDate = d.getMonth() === current.getMonth() && d.getFullYear() === current.getFullYear();
			}

			return matchesQuery && matchesMethod && matchesDate;
		})
	);

	// Inflow Cockpit Analytics
	let totalInflow = $derived(
		payments.reduce((acc, p) => acc + (parseFloat(p.amount) || 0), 0)
	);

	let todayInflow = $derived(
		payments
			.filter((p) => {
				const startOfToday = new Date().setHours(0, 0, 0, 0);
				return new Date(p.payment_date).getTime() >= startOfToday;
			})
			.reduce((acc, p) => acc + (parseFloat(p.amount) || 0), 0)
	);

	let upiVolume = $derived(
		payments
			.filter((p) => p.payment_method?.toUpperCase().includes('UPI'))
			.reduce((acc, p) => acc + (parseFloat(p.amount) || 0), 0)
	);

	let bankVolume = $derived(
		payments
			.filter((p) => p.payment_method?.toUpperCase().includes('BANK') || p.payment_method?.toUpperCase().includes('NEFT'))
			.reduce((acc, p) => acc + (parseFloat(p.amount) || 0), 0)
	);

	function openVoucherDrawer(p: Payment) {
		selectedPayment = p;
		showDrawer = true;
	}

	function handleDrawerKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape' && showDrawer) {
			showDrawer = false;
		}
	}

	function copyReceiptText(p: Payment) {
		const cust = getCustomerName(p);
		const inv = getInvoice(p.invoice_id);
		const text = `Receipt #${p.payment_number} from ${company?.name || 'Company'}\nDate: ${new Date(p.payment_date).toLocaleDateString('en-IN')}\nReceived: ${formatCurrency(p.amount)} via ${p.payment_method}\nRef: ${p.reference_number || 'N/A'}\nInvoice: ${inv?.invoice_number || 'Direct Payment'}\nThank you for your payment!`;
		navigator.clipboard.writeText(text);
		toast.success('Receipt Copied!', 'Receipt summary copied to clipboard for WhatsApp sharing');
	}

	function exportPaymentsCSV() {
		if (filteredPayments.length === 0) {
			toast.warning('No Records', 'No payment records to export');
			return;
		}
		const headers = ['Payment #', 'Date', 'Customer', 'Invoice #', 'Method', 'Reference #', 'Amount', 'Notes'];
		const rows = filteredPayments.map((p) => [
			p.payment_number,
			new Date(p.payment_date).toLocaleDateString('en-IN'),
			`"${getCustomerName(p)}"`,
			getInvoice(p.invoice_id)?.invoice_number || 'N/A',
			p.payment_method,
			`"${p.reference_number || ''}"`,
			p.amount,
			`"${(p.notes || '').replace(/"/g, '""')}"`
		]);
		const csvContent = 'data:text/csv;charset=utf-8,' + [headers.join(','), ...rows.map((r) => r.join(','))].join('\n');
		const encodedUri = encodeURI(csvContent);
		const link = document.createElement('a');
		link.setAttribute('href', encodedUri);
		link.setAttribute('download', `payments_report_${new Date().toISOString().slice(0, 10)}.csv`);
		document.body.appendChild(link);
		link.click();
		document.body.removeChild(link);
		toast.success('Report Exported', 'Payments CSV report downloaded successfully');
	}
</script>

<svelte:window onkeydown={handleDrawerKeydown} />

<svelte:head>
	<title>Payments Received & Bank Inflow | MrBiz-OS</title>
</svelte:head>

<div class="payments-page-container animate-fade-in">
	<!-- Top Cockpit Header -->
	<div class="payments-header">
		<div>
			<div class="header-badge">
				<span>💳</span> CASH FLOW & INFLOW LEDGER
			</div>
			<h1 class="page-title">Payments Received</h1>
			<p class="page-subtitle">
				Audited customer receipts, UPI QR allocations, NEFT reconciliations, and instant payment vouchers.
			</p>
		</div>

		<div class="header-actions">
			<button type="button" class="btn btn-secondary" onclick={exportPaymentsCSV} title="Export CSV Report">
				📥 Export CSV
			</button>
			<button type="button" class="btn btn-secondary" onclick={loadData} title="Refresh records">
				🔄 Refresh
			</button>
			<a href="/invoices" class="btn btn-primary" style="box-shadow: 0 4px 15px rgba(99, 102, 241, 0.4);">
				+ Record Payment on Invoice
			</a>
		</div>
	</div>

	<!-- Executive Financial KPI Strip -->
	<div class="kpi-grid">
		<div class="kpi-card">
			<div class="kpi-top">
				<span class="kpi-label">Total Inflow Collected</span>
				<span class="kpi-icon emerald">₹</span>
			</div>
			<div class="kpi-value text-emerald">{formatCurrency(totalInflow)}</div>
			<div class="kpi-footer">Across {payments.length} verified transactions</div>
		</div>

		<div class="kpi-card">
			<div class="kpi-top">
				<span class="kpi-label">Today's Collections</span>
				<span class="kpi-icon sky">⚡</span>
			</div>
			<div class="kpi-value text-sky">{formatCurrency(todayInflow)}</div>
			<div class="kpi-footer">Real-time daily cash drawer & UPI tally</div>
		</div>

		<div class="kpi-card">
			<div class="kpi-top">
				<span class="kpi-label">UPI & QR Settlements</span>
				<span class="kpi-icon indigo">📱</span>
			</div>
			<div class="kpi-value text-indigo">{formatCurrency(upiVolume)}</div>
			<div class="kpi-footer">
				{payments.filter(p => p.payment_method?.toUpperCase().includes('UPI')).length} digital payments
			</div>
		</div>

		<div class="kpi-card">
			<div class="kpi-top">
				<span class="kpi-label">Bank NEFT / RTGS</span>
				<span class="kpi-icon amber">🏦</span>
			</div>
			<div class="kpi-value text-amber">{formatCurrency(bankVolume)}</div>
			<div class="kpi-footer">Reconciled wire transfers & checks</div>
		</div>
	</div>

	<!-- Search & Filter Controls -->
	<div class="card controls-card">
		<div class="search-box">
			<span class="search-icon">🔍</span>
			<input
				type="text"
				class="form-input search-input"
				placeholder="Search by payment #, customer, UTR reference, invoice #, notes..."
				bind:value={searchQuery}
			/>
			{#if searchQuery}
				<button class="clear-search-btn" onclick={() => (searchQuery = '')}>✕</button>
			{/if}
		</div>

		<div class="filters-row">
			<!-- Payment Method Pills -->
			<div class="filter-pills">
				<button
					type="button"
					class="pill-btn"
					class:active={methodFilter === 'ALL'}
					onclick={() => (methodFilter = 'ALL')}
				>
					All Methods
				</button>
				<button
					type="button"
					class="pill-btn"
					class:active={methodFilter === 'UPI'}
					onclick={() => (methodFilter = 'UPI')}
				>
					📱 UPI / QR
				</button>
				<button
					type="button"
					class="pill-btn"
					class:active={methodFilter === 'BANK_TRANSFER'}
					onclick={() => (methodFilter = 'BANK_TRANSFER')}
				>
					🏦 Bank NEFT
				</button>
				<button
					type="button"
					class="pill-btn"
					class:active={methodFilter === 'CASH'}
					onclick={() => (methodFilter = 'CASH')}
				>
					💵 Cash
				</button>
				<button
					type="button"
					class="pill-btn"
					class:active={methodFilter === 'CARD'}
					onclick={() => (methodFilter = 'CARD')}
				>
					💳 Card
				</button>
			</div>

			<!-- Date Filter Pills -->
			<div class="filter-pills date-pills">
				<button
					type="button"
					class="pill-btn"
					class:active={dateFilter === 'ALL'}
					onclick={() => (dateFilter = 'ALL')}
				>
					All Time
				</button>
				<button
					type="button"
					class="pill-btn"
					class:active={dateFilter === 'TODAY'}
					onclick={() => (dateFilter = 'TODAY')}
				>
					Today
				</button>
				<button
					type="button"
					class="pill-btn"
					class:active={dateFilter === 'MONTH'}
					onclick={() => (dateFilter = 'MONTH')}
				>
					This Month
				</button>
				<button
					type="button"
					class="pill-btn"
					class:active={dateFilter === 'LAST30'}
					onclick={() => (dateFilter = 'LAST30')}
				>
					Last 30 Days
				</button>
			</div>
		</div>
	</div>

	<!-- Main Payments Table -->
	<div class="card table-card">
		<table class="payments-table">
			<thead>
				<tr>
					<th>Payment Number</th>
					<th>Receipt Date</th>
					<th>Customer Name</th>
					<th>Method & Channel</th>
					<th>Reference / UTR #</th>
					<th class="text-right">Amount Received</th>
					<th>Invoice Link</th>
					<th class="text-right">Actions</th>
				</tr>
			</thead>
			<tbody>
				{#if loading}
					<tr>
						<td colspan="8" class="state-cell">
							<div class="loading-state">
								<div class="spinner"></div>
								<span>Loading payment transactions...</span>
							</div>
						</td>
					</tr>
				{:else if error}
					<tr>
						<td colspan="8" class="state-cell">
							<div class="error-state">
								<span>⚠️</span>
								<p>{error}</p>
								<button class="btn btn-secondary btn-sm" onclick={loadData}>Retry</button>
							</div>
						</td>
					</tr>
				{:else if filteredPayments.length === 0}
					<tr>
						<td colspan="8" class="state-cell">
							<div class="empty-state">
								<div class="empty-icon">💳</div>
								<h3>No payment records found</h3>
								<p>
									{#if searchQuery || methodFilter !== 'ALL' || dateFilter !== 'ALL'}
										No receipts match your search filters. Try clearing your filters.
									{:else}
										Record your first customer payment settlement from sales invoices.
									{/if}
								</p>
								<a href="/invoices" class="btn btn-primary btn-sm" style="margin-top: 0.5rem;">
									Go to Invoices to Settle
								</a>
							</div>
						</td>
					</tr>
				{:else}
					{#each filteredPayments as pay (pay.id)}
						{@const inv = getInvoice(pay.invoice_id)}
						{@const custName = getCustomerName(pay)}
						<tr class="payment-row" onclick={() => openVoucherDrawer(pay)}>
							<!-- Payment # -->
							<td>
								<div class="pay-num-cell">
									<span class="pay-num font-mono">{pay.payment_number}</span>
									<button
										type="button"
										class="icon-btn"
										title="Copy Receipt"
										onclick={(e) => { e.stopPropagation(); copyReceiptText(pay); }}
									>
										📋
									</button>
								</div>
							</td>

							<!-- Date -->
							<td class="date-cell">
								{new Date(pay.payment_date).toLocaleDateString('en-IN', { day: '2-digit', month: 'short', year: 'numeric' })}
							</td>

							<!-- Customer -->
							<td>
								<div class="customer-name-val">{custName}</div>
							</td>

							<!-- Method -->
							<td>
								<span class="method-pill {pay.payment_method?.toLowerCase()}">
									{#if pay.payment_method?.toUpperCase().includes('UPI')}
										📱 UPI
									{:else if pay.payment_method?.toUpperCase().includes('BANK') || pay.payment_method?.toUpperCase().includes('NEFT')}
										🏦 Bank Wire
									{:else if pay.payment_method?.toUpperCase().includes('CASH')}
										💵 Cash
									{:else if pay.payment_method?.toUpperCase().includes('CARD')}
										💳 Card
									{:else}
										⚖️ {pay.payment_method}
									{/if}
								</span>
							</td>

							<!-- Reference # -->
							<td>
								{#if pay.reference_number}
									<span class="ref-num font-mono">{pay.reference_number}</span>
								{:else}
									<span style="color: #64748b; font-size: 0.75rem;">—</span>
								{/if}
							</td>

							<!-- Amount -->
							<td class="text-right">
								<span class="amount-val font-mono">{formatCurrency(pay.amount)}</span>
							</td>

							<!-- Invoice Link -->
							<td>
								{#if inv}
									<a
										href={`/invoices/${inv.id}`}
										class="inv-link font-mono"
										onclick={(e) => e.stopPropagation()}
									>
										{inv.invoice_number}
									</a>
								{:else if pay.invoice_id}
									<a
										href={`/invoices/${pay.invoice_id}`}
										class="inv-link font-mono"
										onclick={(e) => e.stopPropagation()}
									>
										Inv #{pay.invoice_id.slice(-6).toUpperCase()}
									</a>
								{:else}
									<span style="color: #64748b; font-size: 0.75rem;">Unallocated</span>
								{/if}
							</td>

							<!-- Actions -->
							<td class="text-right" onclick={(e) => e.stopPropagation()}>
								<div class="actions-group">
									<button
										type="button"
										class="btn btn-secondary btn-sm"
										onclick={() => openVoucherDrawer(pay)}
										title="View Receipt Voucher"
									>
										🧾 Voucher
									</button>
									<button
										type="button"
										class="btn btn-secondary btn-sm"
										onclick={() => copyReceiptText(pay)}
										title="WhatsApp Summary"
									>
										💬 Share
									</button>
								</div>
							</td>
						</tr>
					{/each}
				{/if}
			</tbody>
		</table>
	</div>
</div>

<!-- ============================================== -->
<!-- SLIDE-OVER RECEIPT VOUCHER DRAWER              -->
<!-- ============================================== -->
{#if showDrawer && selectedPayment}
	{@const inv = getInvoice(selectedPayment.invoice_id)}
	{@const custName = getCustomerName(selectedPayment)}
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div class="drawer-backdrop animate-fade-in" onclick={() => (showDrawer = false)}>
		<div class="drawer-panel animate-slide-left" onclick={(e) => e.stopPropagation()} role="dialog" aria-modal="true">
			<!-- Header -->
			<div class="drawer-header">
				<div style="display: flex; align-items: center; gap: 0.6rem;">
					<span style="font-size: 1.4rem;">🧾</span>
					<div>
						<div class="drawer-title">{selectedPayment.payment_number}</div>
						<div class="drawer-subtitle">
							Official Payment Receipt • {new Date(selectedPayment.payment_date).toLocaleDateString('en-IN', { day: '2-digit', month: 'long', year: 'numeric' })}
						</div>
					</div>
				</div>
				<button type="button" class="drawer-close-btn" onclick={() => (showDrawer = false)}>✕</button>
			</div>

			<!-- Body: Printable Voucher Preview -->
			<div class="drawer-body">
				<div class="printable-voucher">
					<!-- Company Header -->
					<div class="voucher-top">
						<div>
							<div class="voucher-company-name">{company?.name || 'Private Enterprise'}</div>
							{#if company?.legal_name && company.legal_name !== company.name}
								<div class="voucher-sub">{company.legal_name}</div>
							{/if}
							{#if company?.gstin}
								<div class="voucher-sub font-mono">GSTIN: {company.gstin}</div>
							{/if}
						</div>
						<div class="voucher-badge">
							PAID RECEIPT
						</div>
					</div>

					<!-- Divider -->
					<div class="voucher-divider"></div>

					<!-- Receipt Amount Hero -->
					<div class="voucher-hero">
						<span class="hero-label">Amount Received</span>
						<span class="hero-amount font-mono">{formatCurrency(selectedPayment.amount)}</span>
						<span class="hero-mode">Settled via {selectedPayment.payment_method}</span>
					</div>

					<!-- Details Grid -->
					<div class="voucher-grid">
						<div class="voucher-field">
							<span class="f-label">Payer / Customer</span>
							<span class="f-val">{custName}</span>
						</div>
						<div class="voucher-field">
							<span class="f-label">Receipt Date</span>
							<span class="f-val">{new Date(selectedPayment.payment_date).toLocaleDateString('en-IN')}</span>
						</div>
						<div class="voucher-field">
							<span class="f-label">Reference / UTR</span>
							<span class="f-val font-mono">{selectedPayment.reference_number || 'N/A'}</span>
						</div>
						<div class="voucher-field">
							<span class="f-label">Allocated Invoice</span>
							<span class="f-val font-mono text-indigo">
								{inv?.invoice_number || (selectedPayment.invoice_id ? 'Inv #' + selectedPayment.invoice_id.slice(-6).toUpperCase() : 'Direct Khata')}
							</span>
						</div>
					</div>

					{#if selectedPayment.notes}
						<div class="voucher-notes">
							<strong>Payment Narration:</strong> {selectedPayment.notes}
						</div>
					{/if}

					<div class="voucher-footer-sign">
						<span>Digitally Verified Receipt</span>
						<span>Authorized Signatory</span>
					</div>
				</div>
			</div>

			<!-- Footer Actions -->
			<div class="drawer-footer">
				<button type="button" class="btn btn-secondary btn-sm" onclick={() => copyReceiptText(selectedPayment!)}>
					💬 Copy for WhatsApp
				</button>
				<button type="button" class="btn btn-secondary btn-sm" onclick={() => window.print()}>
					🖨️ Print Voucher
				</button>
				{#if inv}
					<a href={`/invoices/${inv.id}`} class="btn btn-primary btn-sm">
						View Invoice →
					</a>
				{/if}
			</div>
		</div>
	</div>
{/if}

<style>
	.payments-page-container {
		display: flex;
		flex-direction: column;
		gap: 1.5rem;
		max-width: 1400px;
		margin: 0 auto;
	}

	.payments-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		flex-wrap: wrap;
		gap: 1.25rem;
	}

	.header-badge {
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
		font-size: 0.7rem;
		font-weight: 800;
		color: #34d399;
		background: rgba(16, 185, 129, 0.12);
		padding: 0.2rem 0.55rem;
		border-radius: 6px;
		border: 1px solid rgba(16, 185, 129, 0.25);
		margin-bottom: 0.35rem;
		letter-spacing: 0.05em;
	}

	.page-title {
		font-size: 1.75rem;
		font-weight: 800;
		margin: 0;
		color: #ffffff;
	}

	.page-subtitle {
		font-size: 0.875rem;
		color: #94a3b8;
		margin: 0.25rem 0 0 0;
	}

	.header-actions {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		flex-wrap: wrap;
	}

	/* KPI Cockpit */
	.kpi-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
		gap: 1rem;
	}

	.kpi-card {
		background: rgba(17, 24, 39, 0.75);
		backdrop-filter: blur(16px);
		-webkit-backdrop-filter: blur(16px);
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 12px;
		padding: 1.25rem;
		display: flex;
		flex-direction: column;
		gap: 0.4rem;
		transition: all 0.2s ease;
	}

	.kpi-card:hover {
		border-color: rgba(99, 102, 241, 0.3);
		transform: translateY(-2px);
		box-shadow: 0 8px 20px rgba(0, 0, 0, 0.3);
	}

	.kpi-top {
		display: flex;
		justify-content: space-between;
		align-items: center;
	}

	.kpi-label {
		font-size: 0.8125rem;
		font-weight: 600;
		color: #94a3b8;
		text-transform: uppercase;
		letter-spacing: 0.03em;
	}

	.kpi-icon {
		width: 32px;
		height: 32px;
		border-radius: 8px;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 1rem;
		font-weight: 800;
	}
	.kpi-icon.emerald { background: rgba(16, 185, 129, 0.15); color: #34d399; }
	.kpi-icon.sky { background: rgba(14, 165, 233, 0.15); color: #38bdf8; }
	.kpi-icon.indigo { background: rgba(99, 102, 241, 0.15); color: #818cf8; }
	.kpi-icon.amber { background: rgba(245, 158, 11, 0.15); color: #fbbf24; }

	.kpi-value {
		font-size: 1.6rem;
		font-weight: 800;
		color: #ffffff;
		font-family: var(--font-heading);
		line-height: 1.1;
	}

	.kpi-footer {
		font-size: 0.75rem;
		color: #64748b;
	}

	/* Controls Card */
	.controls-card {
		padding: 1rem 1.25rem;
		display: flex;
		flex-direction: column;
		gap: 0.875rem;
	}

	.search-box {
		position: relative;
		display: flex;
		align-items: center;
	}

	.search-icon {
		position: absolute;
		left: 0.875rem;
		color: #64748b;
		font-size: 0.9rem;
	}

	.search-input {
		padding-left: 2.5rem;
		padding-right: 2rem;
		height: 42px;
		border-radius: 8px;
		background: rgba(15, 23, 42, 0.8);
		border: 1px solid rgba(255, 255, 255, 0.1);
		color: #f8fafc;
		width: 100%;
	}

	.search-input:focus {
		border-color: #10b981;
		box-shadow: 0 0 10px rgba(16, 185, 129, 0.25);
	}

	.clear-search-btn {
		position: absolute;
		right: 0.75rem;
		background: none;
		border: none;
		color: #64748b;
		cursor: pointer;
		font-size: 0.8rem;
	}

	.filters-row {
		display: flex;
		justify-content: space-between;
		align-items: center;
		flex-wrap: wrap;
		gap: 0.75rem;
	}

	.filter-pills {
		display: flex;
		align-items: center;
		gap: 0.35rem;
		flex-wrap: wrap;
	}

	.pill-btn {
		background: rgba(255, 255, 255, 0.04);
		border: 1px solid rgba(255, 255, 255, 0.08);
		color: #94a3b8;
		font-size: 0.775rem;
		font-weight: 600;
		padding: 0.35rem 0.75rem;
		border-radius: 6px;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.pill-btn:hover {
		background: rgba(255, 255, 255, 0.08);
		color: #f8fafc;
	}

	.pill-btn.active {
		background: rgba(16, 185, 129, 0.2);
		border-color: #10b981;
		color: #34d399;
		font-weight: 700;
	}

	/* Table Card */
	.table-card {
		padding: 0;
		overflow: hidden;
		border: 1px solid rgba(255, 255, 255, 0.08);
	}

	.payments-table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.875rem;
		text-align: left;
	}

	.payments-table th {
		background: rgba(15, 23, 42, 0.85);
		padding: 0.875rem 1.25rem;
		color: #94a3b8;
		font-size: 0.75rem;
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 0.04em;
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
	}

	.payments-table td {
		padding: 1rem 1.25rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.04);
		color: #f8fafc;
		vertical-align: middle;
	}

	.payment-row {
		cursor: pointer;
		transition: background 0.15s ease;
	}

	.payment-row:hover {
		background: rgba(255, 255, 255, 0.03);
	}

	.pay-num-cell {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.pay-num {
		font-weight: 700;
		color: #34d399;
	}

	.icon-btn {
		background: none;
		border: none;
		cursor: pointer;
		font-size: 0.8rem;
		opacity: 0.6;
		transition: opacity 0.15s ease;
	}
	.icon-btn:hover { opacity: 1; }

	.customer-name-val {
		font-weight: 600;
		color: #ffffff;
	}

	.method-pill {
		display: inline-flex;
		align-items: center;
		gap: 0.3rem;
		font-size: 0.75rem;
		font-weight: 700;
		padding: 0.2rem 0.55rem;
		border-radius: 6px;
		background: rgba(255, 255, 255, 0.06);
		border: 1px solid rgba(255, 255, 255, 0.1);
	}

	.ref-num {
		font-size: 0.8125rem;
		color: #94a3b8;
	}

	.amount-val {
		font-weight: 800;
		color: #34d399;
		font-size: 0.95rem;
	}

	.inv-link {
		color: #818cf8;
		text-decoration: none;
		font-weight: 600;
		background: rgba(99, 102, 241, 0.1);
		padding: 0.2rem 0.5rem;
		border-radius: 4px;
		border: 1px solid rgba(99, 102, 241, 0.25);
		font-size: 0.775rem;
		transition: all 0.15s ease;
	}
	.inv-link:hover {
		background: rgba(99, 102, 241, 0.2);
		color: #a5b4fc;
	}

	.actions-group {
		display: flex;
		align-items: center;
		justify-content: flex-end;
		gap: 0.4rem;
	}

	/* State Cells */
	.state-cell {
		text-align: center;
		padding: 3.5rem 1rem !important;
	}

	.loading-state, .error-state, .empty-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.5rem;
		color: #94a3b8;
	}

	.spinner {
		width: 32px;
		height: 32px;
		border: 3px solid rgba(16, 185, 129, 0.2);
		border-top-color: #10b981;
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	.empty-icon { font-size: 2.5rem; opacity: 0.7; }
	.empty-state h3 { color: #ffffff; margin: 0; font-size: 1.125rem; }
	.empty-state p { color: #64748b; font-size: 0.8125rem; max-width: 380px; margin: 0; }

	/* SLIDE-OVER DRAWER */
	.drawer-backdrop {
		position: fixed;
		inset: 0;
		z-index: 10000;
		background: rgba(11, 15, 25, 0.7);
		backdrop-filter: blur(8px);
		display: flex;
		justify-content: flex-end;
	}

	.drawer-panel {
		width: 100%;
		max-width: 540px;
		height: 100vh;
		background: #111827;
		border-left: 1px solid rgba(255, 255, 255, 0.12);
		box-shadow: -10px 0 30px rgba(0, 0, 0, 0.6);
		display: flex;
		flex-direction: column;
		overflow: hidden;
	}

	.drawer-header {
		padding: 1.25rem 1.5rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
		display: flex;
		justify-content: space-between;
		align-items: center;
		background: rgba(15, 23, 42, 0.6);
	}

	.drawer-title {
		font-size: 1.2rem;
		font-weight: 800;
		color: #34d399;
		font-family: var(--font-mono);
	}

	.drawer-subtitle {
		font-size: 0.8125rem;
		color: #94a3b8;
	}

	.drawer-close-btn {
		background: rgba(255, 255, 255, 0.06);
		border: 1px solid rgba(255, 255, 255, 0.1);
		color: #94a3b8;
		width: 32px;
		height: 32px;
		border-radius: 8px;
		display: flex;
		align-items: center;
		justify-content: center;
		cursor: pointer;
	}

	.drawer-body {
		flex: 1;
		overflow-y: auto;
		padding: 1.5rem;
	}

	/* Voucher Paper Style */
	.printable-voucher {
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: 12px;
		padding: 1.5rem;
		box-shadow: 0 10px 25px rgba(0, 0, 0, 0.3);
		display: flex;
		flex-direction: column;
		gap: 1.25rem;
	}

	.voucher-top {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
	}

	.voucher-company-name {
		font-size: 1.15rem;
		font-weight: 800;
		color: #ffffff;
	}

	.voucher-sub {
		font-size: 0.775rem;
		color: #94a3b8;
	}

	.voucher-badge {
		background: rgba(16, 185, 129, 0.15);
		color: #34d399;
		border: 1px solid rgba(16, 185, 129, 0.3);
		padding: 0.25rem 0.6rem;
		border-radius: 4px;
		font-size: 0.7rem;
		font-weight: 800;
		letter-spacing: 0.05em;
	}

	.voucher-divider {
		height: 1px;
		background: rgba(255, 255, 255, 0.08);
	}

	.voucher-hero {
		background: rgba(16, 185, 129, 0.08);
		border: 1px dashed rgba(16, 185, 129, 0.3);
		border-radius: 8px;
		padding: 1.25rem;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.25rem;
		text-align: center;
	}

	.hero-label {
		font-size: 0.75rem;
		color: #94a3b8;
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}

	.hero-amount {
		font-size: 2rem;
		font-weight: 900;
		color: #34d399;
	}

	.hero-mode {
		font-size: 0.8125rem;
		color: #a7f3d0;
		font-weight: 600;
	}

	.voucher-grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 1rem;
	}

	.voucher-field {
		display: flex;
		flex-direction: column;
		gap: 0.2rem;
	}

	.f-label {
		font-size: 0.6875rem;
		color: #64748b;
		text-transform: uppercase;
		letter-spacing: 0.04em;
	}

	.f-val {
		font-size: 0.875rem;
		font-weight: 600;
		color: #f8fafc;
	}

	.voucher-notes {
		background: rgba(0, 0, 0, 0.25);
		padding: 0.75rem 1rem;
		border-radius: 6px;
		font-size: 0.8125rem;
		color: #cbd5e1;
		border-left: 3px solid #6366f1;
	}

	.voucher-footer-sign {
		margin-top: 1rem;
		display: flex;
		justify-content: space-between;
		font-size: 0.7rem;
		color: #64748b;
		border-top: 1px solid rgba(255, 255, 255, 0.06);
		padding-top: 0.75rem;
	}

	.drawer-footer {
		padding: 1rem 1.5rem;
		border-top: 1px solid rgba(255, 255, 255, 0.08);
		background: rgba(15, 23, 42, 0.8);
		display: flex;
		justify-content: flex-end;
		align-items: center;
		gap: 0.6rem;
	}

	/* ANIMATIONS */
	@keyframes slideLeft {
		from { transform: translateX(100%); }
		to { transform: translateX(0); }
	}

	.animate-slide-left {
		animation: slideLeft 0.25s cubic-bezier(0.16, 1, 0.3, 1) forwards;
	}

	@media (max-width: 768px) {
		.payments-header { flex-direction: column; align-items: flex-start; }
		.header-actions { width: 100%; justify-content: space-between; }
		.kpi-grid { grid-template-columns: 1fr 1fr; }
		.drawer-panel { max-width: 100%; }
	}
</style>
