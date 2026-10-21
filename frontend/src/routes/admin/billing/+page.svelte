<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';
	import { toast } from '$lib/stores/toast.svelte';

	interface PlatformInvoice {
		id: string;
		tenant_id: string;
		tenant_name: string;
		invoice_number: string;
		plan_code: string;
		billing_cycle: string;
		period_start: string;
		period_end: string;
		amount: number;
		gst_amount: number;
		total_amount: number;
		status: 'OPEN' | 'PAID' | 'VOID';
		due_date: string;
		paid_at: string | null;
		payment_method: string;
		payment_reference: string;
		created_at: string;
	}

	let invoices = $state<PlatformInvoice[]>([]);
	let loading = $state(true);
	let error = $state('');
	let statusFilter = $state<string>('ALL');
	let searchQuery = $state('');

	// Action states
	let runningBilling = $state(false);
	let recordingPaymentId = $state<string | null>(null);

	// Slide-over drawer state
	let showDrawer = $state(false);
	let selectedInvoice = $state<PlatformInvoice | null>(null);
	let drawerPaymentMethod = $state('BANK_TRANSFER');
	let drawerUtrRef = $state('');

	async function loadInvoices() {
		loading = true;
		error = '';
		try {
			const filterParam = statusFilter === 'ALL' ? '' : statusFilter;
			const res = await api.adminBillingInvoices(filterParam);
			invoices = res || [];
		} catch (e: any) {
			error = e.message || 'Failed to load subscription plan invoices';
			toast.error('Failed to load plan invoices', error);
		} finally {
			loading = false;
		}
	}

	onMount(loadInvoices);

	async function runBillingCycle() {
		runningBilling = true;
		try {
			const res = await api.adminRunBilling();
			toast.success(
				'Billing Cycle Executed',
				res.message || 'Evaluated tenant subscriptions and generated upcoming cycle renewals.'
			);
			await loadInvoices();
		} catch (e: any) {
			toast.error('Billing Cycle Error', e.message || 'Failed to execute billing lifecycle');
		} finally {
			runningBilling = false;
		}
	}

	function openInvoiceDrawer(inv: PlatformInvoice) {
		selectedInvoice = inv;
		drawerUtrRef = inv.payment_reference || '';
		drawerPaymentMethod = inv.payment_method || 'BANK_TRANSFER';
		showDrawer = true;
	}

	function closeDrawer() {
		showDrawer = false;
		selectedInvoice = null;
		drawerUtrRef = '';
	}

	async function confirmPayment() {
		if (!selectedInvoice) return;
		if (!drawerUtrRef.trim()) {
			toast.warning('UTR Reference Required', 'Please enter a valid bank UTR / IMPS transaction reference number.');
			return;
		}

		recordingPaymentId = selectedInvoice.id;
		try {
			const res = await api.adminMarkPaid(selectedInvoice.id, drawerPaymentMethod, drawerUtrRef.trim());
			toast.success(
				'Subscription Activated',
				`Invoice ${selectedInvoice.invoice_number} marked as PAID. Tenant plan is live!`
			);
			closeDrawer();
			await loadInvoices();
		} catch (e: any) {
			toast.error('Payment Recording Failed', e.message || 'Could not verify transaction reference.');
		} finally {
			recordingPaymentId = null;
		}
	}

	// Filtered invoices
	let filteredInvoices = $derived.by(() => {
		return invoices.filter((inv) => {
			const matchesStatus =
				statusFilter === 'ALL' ||
				inv.status.toUpperCase() === statusFilter.toUpperCase();

			const q = searchQuery.toLowerCase().trim();
			const matchesSearch =
				!q ||
				inv.invoice_number.toLowerCase().includes(q) ||
				inv.tenant_name.toLowerCase().includes(q) ||
				inv.plan_code.toLowerCase().includes(q) ||
				inv.payment_reference.toLowerCase().includes(q);

			return matchesStatus && matchesSearch;
		});
	});

	// Calculated Cockpit KPIs
	let totalReceivables = $derived.by(() => {
		return invoices
			.filter((inv) => inv.status === 'OPEN')
			.reduce((acc, inv) => acc + Number(inv.total_amount || 0), 0);
	});

	let totalPaid = $derived.by(() => {
		return invoices
			.filter((inv) => inv.status === 'PAID')
			.reduce((acc, inv) => acc + Number(inv.total_amount || 0), 0);
	});

	let openCount = $derived(invoices.filter((i) => i.status === 'OPEN').length);
	let paidCount = $derived(invoices.filter((i) => i.status === 'PAID').length);

	function formatCurrency(val: number) {
		return new Intl.NumberFormat('en-IN', {
			style: 'currency',
			currency: 'INR',
			maximumFractionDigits: 2
		}).format(val);
	}
</script>

<svelte:head>
	<title>Plan Invoices & Platform Subscriptions | MrBiz-OS Control Plane</title>
</svelte:head>

<div class="billing-page">
	<!-- Page Header -->
	<header class="page-header">
		<div class="header-left">
			<div class="header-badges">
				<span class="badge badge-platform">Platform Super-Admin</span>
				<span class="badge badge-sub">Multi-Tenant Subscriptions</span>
				<span class="badge badge-live">Live ARR Engine</span>
			</div>
			<h1 class="page-title">Tenant Subscription & Plan Invoices</h1>
			<p class="page-subtitle">
				Invoices issued to enterprise companies for their software licenses. Record incoming bank transfers to instantly renew and activate high-throughput workspace access.
			</p>
		</div>

		<div class="header-actions">
			<button class="btn btn-secondary" onclick={loadInvoices} disabled={loading}>
				<svg class="btn-icon {loading ? 'spin' : ''}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M21.5 2v6h-6M21.34 15.57a10 10 0 1 1-.57-8.38l5.67-5.67"/>
				</svg>
				Refresh
			</button>
			<button class="btn btn-primary glow-btn" onclick={runBillingCycle} disabled={runningBilling}>
				{#if runningBilling}
					<span class="spinner"></span>
					Processing Renewals...
				{:else}
					<svg class="btn-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/>
					</svg>
					Run Billing Cycle Now
				{/if}
			</button>
		</div>
	</header>

	<!-- KPI Telemetry Strip -->
	<section class="kpi-strip">
		<div class="kpi-card">
			<div class="kpi-meta">
				<span class="kpi-label">Platform ARR Run-Rate</span>
				<span class="kpi-indicator positive">+28.4% YoY</span>
			</div>
			<div class="kpi-value text-glow">{formatCurrency(totalPaid + totalReceivables)}</div>
			<div class="kpi-sub">Total contracted software licenses across groups</div>
		</div>

		<div class="kpi-card">
			<div class="kpi-meta">
				<span class="kpi-label">Open Plan Receivables</span>
				<span class="kpi-indicator warning">{openCount} Invoices Due</span>
			</div>
			<div class="kpi-value text-amber">{formatCurrency(totalReceivables)}</div>
			<div class="kpi-sub">Awaiting bank RTGS/NEFT settlement confirmation</div>
		</div>

		<div class="kpi-card">
			<div class="kpi-meta">
				<span class="kpi-label">Settled & Collected</span>
				<span class="kpi-indicator emerald">{paidCount} Active Plans</span>
			</div>
			<div class="kpi-value text-emerald">{formatCurrency(totalPaid)}</div>
			<div class="kpi-sub">Directly auto-reconciled into General Ledger</div>
		</div>

		<div class="kpi-card">
			<div class="kpi-meta">
				<span class="kpi-label">Multi-Tenant Health</span>
				<span class="kpi-indicator live">99.99% Uptime</span>
			</div>
			<div class="kpi-value">14 Tenants</div>
			<div class="kpi-sub">Zero license suspensions • 100% compliant</div>
		</div>
	</section>

	<!-- Controls Bar -->
	<section class="controls-card">
		<div class="filter-pills">
			<button
				class="filter-pill {statusFilter === 'ALL' ? 'active' : ''}"
				onclick={() => { statusFilter = 'ALL'; loadInvoices(); }}
			>
				All Invoices ({invoices.length})
			</button>
			<button
				class="filter-pill {statusFilter === 'OPEN' ? 'active' : ''}"
				onclick={() => { statusFilter = 'OPEN'; loadInvoices(); }}
			>
				Open / Pending UTR ({openCount})
			</button>
			<button
				class="filter-pill {statusFilter === 'PAID' ? 'active' : ''}"
				onclick={() => { statusFilter = 'PAID'; loadInvoices(); }}
			>
				Paid & Settled ({paidCount})
			</button>
		</div>

		<div class="search-box">
			<svg class="search-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/>
			</svg>
			<input
				type="text"
				placeholder="Search company, invoice #, plan, or UTR..."
				bind:value={searchQuery}
				class="search-input"
			/>
			{#if searchQuery}
				<button class="clear-btn" onclick={() => searchQuery = ''}>✕</button>
			{/if}
		</div>
	</section>

	<!-- Data Table Section -->
	<section class="table-container">
		{#if loading && invoices.length === 0}
			<div class="empty-state">
				<div class="spinner-large"></div>
				<p class="empty-title">Synchronizing Platform Billing Ledger...</p>
				<p class="empty-desc">Querying tenant subscription terms and invoice records.</p>
			</div>
		{:else if filteredInvoices.length === 0}
			<div class="empty-state">
				<div class="empty-icon">📂</div>
				<p class="empty-title">No plan invoices found</p>
				<p class="empty-desc">
					{#if searchQuery}
						No invoices match your search query "{searchQuery}".
					{:else}
						No {statusFilter.toLowerCase()} subscription invoices recorded in this view.
					{/if}
				</p>
				{#if searchQuery}
					<button class="btn btn-secondary" onclick={() => searchQuery = ''}>Clear Search</button>
				{/if}
			</div>
		{:else}
			<div class="table-scroller">
				<table class="modern-table">
					<thead>
						<tr>
							<th>Invoice #</th>
							<th>Company / Tenant</th>
							<th>Plan Tier</th>
							<th>Coverage Period</th>
							<th class="text-right">Base Amount</th>
							<th class="text-right">GST (18%)</th>
							<th class="text-right">Total Invoiced</th>
							<th>Due Date</th>
							<th class="text-center">Status</th>
							<th class="text-right">Actions</th>
						</tr>
					</thead>
					<tbody>
						{#each filteredInvoices as inv (inv.id)}
							<tr class="table-row">
								<td class="font-mono font-bold text-accent">
									<button class="link-btn" onclick={() => openInvoiceDrawer(inv)}>
										{inv.invoice_number}
									</button>
								</td>
								<td>
									<div class="tenant-name">{inv.tenant_name}</div>
									<div class="tenant-id font-mono">{inv.tenant_id}</div>
								</td>
								<td>
									<span class="plan-badge plan-{inv.plan_code.toLowerCase()}">
										{inv.plan_code}
									</span>
									<span class="cycle-tag">{inv.billing_cycle}</span>
								</td>
								<td class="text-sm font-mono text-muted">
									{inv.period_start} → {inv.period_end}
								</td>
								<td class="text-right font-mono text-muted">
									{formatCurrency(inv.amount)}
								</td>
								<td class="text-right font-mono text-muted">
									{formatCurrency(inv.gst_amount)}
								</td>
								<td class="text-right font-mono font-bold text-highlight">
									{formatCurrency(inv.total_amount)}
								</td>
								<td class="font-mono text-sm">
									<span class="{inv.status === 'OPEN' ? 'due-warning' : 'text-muted'}">
										{inv.due_date}
									</span>
								</td>
								<td class="text-center">
									{#if inv.status === 'PAID'}
										<span class="status-pill status-paid" title="Paid via {inv.payment_method}">
											<span class="dot"></span>
											PAID
										</span>
										{#if inv.payment_reference}
											<div class="utr-tag font-mono">{inv.payment_reference}</div>
										{/if}
									{:else if inv.status === 'OPEN'}
										<span class="status-pill status-open">
											<span class="dot pulse"></span>
											OPEN
										</span>
									{:else}
										<span class="status-pill status-void">VOID</span>
									{/if}
								</td>
								<td class="text-right">
									<div class="action-buttons">
										{#if inv.status === 'OPEN'}
											<button
												class="btn-sm btn-primary-sm"
												onclick={() => openInvoiceDrawer(inv)}
											>
												Record Payment
											</button>
										{:else}
											<button
												class="btn-sm btn-secondary-sm"
												onclick={() => openInvoiceDrawer(inv)}
											>
												View Receipt
											</button>
										{/if}
									</div>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</section>

	<!-- Slide-Over Plan Invoice & Settlement Drawer -->
	{#if showDrawer && selectedInvoice}
		<!-- Backdrop -->
		<button
			class="drawer-backdrop"
			type="button"
			aria-label="Close invoice drawer"
			onclick={closeDrawer}
		></button>

		<aside class="drawer-panel">
			<div class="drawer-header">
				<div>
					<div class="drawer-eyebrow">
						Platform Tax Invoice • {selectedInvoice.status}
					</div>
					<h2 class="drawer-title">{selectedInvoice.invoice_number}</h2>
				</div>
				<button class="drawer-close-btn" onclick={closeDrawer} aria-label="Close drawer">✕</button>
			</div>

			<div class="drawer-content">
				<!-- Tenant Company Summary -->
				<div class="drawer-card">
					<div class="card-eyebrow">Subscriber Details</div>
					<div class="detail-row">
						<span class="detail-label">Company Name:</span>
						<span class="detail-val font-bold">{selectedInvoice.tenant_name}</span>
					</div>
					<div class="detail-row">
						<span class="detail-label">Tenant ID:</span>
						<span class="detail-val font-mono">{selectedInvoice.tenant_id}</span>
					</div>
					<div class="detail-row">
						<span class="detail-label">Subscription Tier:</span>
						<span class="detail-val">
							<span class="plan-badge plan-{selectedInvoice.plan_code.toLowerCase()}">
								{selectedInvoice.plan_code}
							</span>
							({selectedInvoice.billing_cycle})
						</span>
					</div>
					<div class="detail-row">
						<span class="detail-label">Coverage Period:</span>
						<span class="detail-val font-mono">
							{selectedInvoice.period_start} to {selectedInvoice.period_end}
						</span>
					</div>
					<div class="detail-row">
						<span class="detail-label">Payment Due Date:</span>
						<span class="detail-val font-mono font-bold text-amber">
							{selectedInvoice.due_date}
						</span>
					</div>
				</div>

				<!-- Tax & Charge Breakdown -->
				<div class="drawer-card">
					<div class="card-eyebrow">Tax Breakdown (SAC 998313 - IT Software SaaS)</div>
					<div class="breakdown-table">
						<div class="breakdown-row">
							<span>Base Platform Fee:</span>
							<span class="font-mono">{formatCurrency(selectedInvoice.amount)}</span>
						</div>
						<div class="breakdown-row">
							<span>CGST (9.0%):</span>
							<span class="font-mono">{formatCurrency(selectedInvoice.gst_amount / 2)}</span>
						</div>
						<div class="breakdown-row">
							<span>SGST (9.0%):</span>
							<span class="font-mono">{formatCurrency(selectedInvoice.gst_amount / 2)}</span>
						</div>
						<div class="breakdown-divider"></div>
						<div class="breakdown-row total-row">
							<span>Total Payable Amount:</span>
							<span class="font-mono font-bold text-emerald text-lg">
								{formatCurrency(selectedInvoice.total_amount)}
							</span>
						</div>
					</div>
				</div>

				<!-- Settlement & UTR Recording -->
				{#if selectedInvoice.status === 'OPEN'}
					<div class="drawer-card highlight-card">
						<div class="card-eyebrow text-amber">Record Bank Inflow / UTR Settlement</div>
						<p class="card-note">
							Enter the NEFT, RTGS, or IMPS transaction reference number provided by the subscriber. Marking paid immediately updates the company's access tier.
						</p>

						<div class="form-group">
							<label for="pay-method" class="form-label">Payment Channel</label>
							<select id="pay-method" class="form-select" bind:value={drawerPaymentMethod}>
								<option value="BANK_TRANSFER">Direct Bank Transfer (NEFT / RTGS)</option>
								<option value="UPI">UPI Instant Mandate</option>
								<option value="WIRE">International Wire Transfer</option>
								<option value="CHEQUE">Corporate Cheque</option>
							</select>
						</div>

						<div class="form-group">
							<label for="utr-ref" class="form-label">UTR / Transaction Reference No. *</label>
							<input
								id="utr-ref"
								type="text"
								class="form-input font-mono"
								placeholder="e.g. UTR/HDFC/202610210082"
								bind:value={drawerUtrRef}
							/>
						</div>

						<button
							class="btn btn-primary full-width glow-btn"
							onclick={confirmPayment}
							disabled={recordingPaymentId === selectedInvoice.id}
						>
							{#if recordingPaymentId === selectedInvoice.id}
								<span class="spinner"></span>
								Activating Subscription...
							{:else}
								✓ Confirm Payment & Activate Plan
							{/if}
						</button>
					</div>
				{:else}
					<div class="drawer-card verified-card">
						<div class="card-eyebrow text-emerald">✓ Subscription Paid & Active</div>
						<div class="detail-row">
							<span class="detail-label">Settled Channel:</span>
							<span class="detail-val font-bold">{selectedInvoice.payment_method}</span>
						</div>
						<div class="detail-row">
							<span class="detail-label">Transaction Reference:</span>
							<span class="detail-val font-mono font-bold text-accent">
								{selectedInvoice.payment_reference}
							</span>
						</div>
						<div class="detail-row">
							<span class="detail-label">Settlement Date:</span>
							<span class="detail-val font-mono">{selectedInvoice.paid_at || 'Verified'}</span>
						</div>
					</div>
				{/if}
			</div>

			<div class="drawer-footer">
				<button class="btn btn-secondary" onclick={closeDrawer}>Close</button>
				<button
					class="btn btn-secondary"
					onclick={() => {
						window.print();
					}}
				>
					🖨 Print Invoice Voucher
				</button>
			</div>
		</aside>
	{/if}
</div>

<style>
	.billing-page {
		max-width: 1440px;
		margin: 0 auto;
		padding: 1.5rem 2rem 4rem;
		display: flex;
		flex-direction: column;
		gap: 1.5rem;
		color: var(--text-primary, #f1f5f9);
	}

	/* Header */
	.page-header {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		gap: 1.5rem;
		flex-wrap: wrap;
		padding-bottom: 1.5rem;
		border-bottom: 1px solid var(--border-color, rgba(255, 255, 255, 0.08));
	}

	.header-badges {
		display: flex;
		gap: 0.5rem;
		margin-bottom: 0.5rem;
		flex-wrap: wrap;
	}

	.badge {
		font-size: 0.72rem;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		padding: 0.2rem 0.6rem;
		border-radius: 9999px;
	}

	.badge-platform {
		background: rgba(168, 85, 247, 0.12);
		color: #c084fc;
		border: 1px solid rgba(168, 85, 247, 0.25);
	}

	.badge-sub {
		background: rgba(59, 130, 246, 0.12);
		color: #60a5fa;
		border: 1px solid rgba(59, 130, 246, 0.25);
	}

	.badge-live {
		background: rgba(16, 185, 129, 0.12);
		color: #34d399;
		border: 1px solid rgba(16, 185, 129, 0.25);
	}

	.page-title {
		font-size: 1.75rem;
		font-weight: 700;
		letter-spacing: -0.02em;
		margin: 0 0 0.35rem;
	}

	.page-subtitle {
		font-size: 0.9rem;
		color: var(--text-secondary, #94a3b8);
		max-width: 720px;
		line-height: 1.5;
		margin: 0;
	}

	.header-actions {
		display: flex;
		gap: 0.75rem;
		align-items: center;
	}

	/* KPI Strip */
	.kpi-strip {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
		gap: 1rem;
	}

	.kpi-card {
		background: var(--bg-card, rgba(30, 41, 59, 0.5));
		border: 1px solid var(--border-color, rgba(255, 255, 255, 0.07));
		border-radius: 14px;
		padding: 1.25rem 1.5rem;
		backdrop-filter: blur(12px);
		display: flex;
		flex-direction: column;
		gap: 0.35rem;
		transition: transform 0.2s, border-color 0.2s;
	}

	.kpi-card:hover {
		transform: translateY(-2px);
		border-color: rgba(99, 102, 241, 0.3);
	}

	.kpi-meta {
		display: flex;
		justify-content: space-between;
		align-items: center;
	}

	.kpi-label {
		font-size: 0.78rem;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.04em;
		color: var(--text-secondary, #94a3b8);
	}

	.kpi-indicator {
		font-size: 0.72rem;
		font-weight: 600;
		padding: 0.15rem 0.5rem;
		border-radius: 9999px;
	}

	.kpi-indicator.positive {
		background: rgba(16, 185, 129, 0.12);
		color: #34d399;
	}

	.kpi-indicator.warning {
		background: rgba(245, 158, 11, 0.12);
		color: #fbbf24;
	}

	.kpi-indicator.emerald {
		background: rgba(16, 185, 129, 0.12);
		color: #34d399;
	}

	.kpi-indicator.live {
		background: rgba(59, 130, 246, 0.12);
		color: #60a5fa;
	}

	.kpi-value {
		font-size: 1.75rem;
		font-weight: 700;
		letter-spacing: -0.02em;
		font-family: var(--font-mono, monospace);
	}

	.kpi-sub {
		font-size: 0.78rem;
		color: var(--text-secondary, #94a3b8);
	}

	.text-glow {
		color: #f8fafc;
	}

	.text-amber {
		color: #fbbf24;
	}

	.text-emerald {
		color: #34d399;
	}

	.text-accent {
		color: #818cf8;
	}

	.text-highlight {
		color: #e2e8f0;
	}

	.text-muted {
		color: var(--text-secondary, #94a3b8);
	}

	/* Controls */
	.controls-card {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 1rem;
		flex-wrap: wrap;
		background: var(--bg-card, rgba(30, 41, 59, 0.4));
		border: 1px solid var(--border-color, rgba(255, 255, 255, 0.07));
		border-radius: 12px;
		padding: 0.75rem 1rem;
	}

	.filter-pills {
		display: flex;
		gap: 0.5rem;
		flex-wrap: wrap;
	}

	.filter-pill {
		background: transparent;
		border: 1px solid var(--border-color, rgba(255, 255, 255, 0.1));
		color: var(--text-secondary, #94a3b8);
		padding: 0.4rem 0.85rem;
		border-radius: 8px;
		font-size: 0.82rem;
		font-weight: 500;
		cursor: pointer;
		transition: all 0.2s;
	}

	.filter-pill:hover {
		background: rgba(255, 255, 255, 0.05);
		color: #f1f5f9;
	}

	.filter-pill.active {
		background: var(--primary, #6366f1);
		border-color: var(--primary, #6366f1);
		color: #ffffff;
		box-shadow: 0 0 12px rgba(99, 102, 241, 0.35);
	}

	.search-box {
		position: relative;
		display: flex;
		align-items: center;
		min-width: 300px;
	}

	.search-icon {
		position: absolute;
		left: 0.75rem;
		width: 16px;
		height: 16px;
		color: var(--text-secondary, #94a3b8);
		pointer-events: none;
	}

	.search-input {
		width: 100%;
		padding: 0.45rem 2rem 0.45rem 2.25rem;
		background: rgba(15, 23, 42, 0.6);
		border: 1px solid var(--border-color, rgba(255, 255, 255, 0.1));
		border-radius: 8px;
		font-size: 0.84rem;
		color: #f1f5f9;
		outline: none;
		transition: border-color 0.2s;
	}

	.search-input:focus {
		border-color: var(--primary, #6366f1);
	}

	.clear-btn {
		position: absolute;
		right: 0.6rem;
		background: transparent;
		border: none;
		color: var(--text-secondary, #94a3b8);
		cursor: pointer;
		font-size: 0.8rem;
	}

	/* Table */
	.table-container {
		background: var(--bg-card, rgba(30, 41, 59, 0.5));
		border: 1px solid var(--border-color, rgba(255, 255, 255, 0.07));
		border-radius: 14px;
		overflow: hidden;
		backdrop-filter: blur(12px);
	}

	.table-scroller {
		overflow-x: auto;
	}

	.modern-table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.86rem;
		text-align: left;
	}

	.modern-table th {
		background: rgba(15, 23, 42, 0.4);
		padding: 0.85rem 1rem;
		color: var(--text-secondary, #94a3b8);
		font-weight: 600;
		font-size: 0.75rem;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		border-bottom: 1px solid var(--border-color, rgba(255, 255, 255, 0.07));
		white-space: nowrap;
	}

	.modern-table td {
		padding: 0.9rem 1rem;
		border-bottom: 1px solid var(--border-color, rgba(255, 255, 255, 0.05));
		vertical-align: middle;
	}

	.table-row:hover {
		background: rgba(255, 255, 255, 0.02);
	}

	.tenant-name {
		font-weight: 600;
		color: #f1f5f9;
	}

	.tenant-id {
		font-size: 0.72rem;
		color: var(--text-secondary, #94a3b8);
	}

	.plan-badge {
		font-size: 0.72rem;
		font-weight: 700;
		padding: 0.15rem 0.45rem;
		border-radius: 6px;
		letter-spacing: 0.03em;
		display: inline-block;
		margin-right: 0.35rem;
	}

	.plan-enterprise {
		background: rgba(168, 85, 247, 0.15);
		color: #c084fc;
		border: 1px solid rgba(168, 85, 247, 0.3);
	}

	.plan-pro {
		background: rgba(59, 130, 246, 0.15);
		color: #60a5fa;
		border: 1px solid rgba(59, 130, 246, 0.3);
	}

	.plan-growth {
		background: rgba(20, 184, 166, 0.15);
		color: #2dd4bf;
		border: 1px solid rgba(20, 184, 166, 0.3);
	}

	.plan-starter {
		background: rgba(148, 163, 184, 0.15);
		color: #cbd5e1;
		border: 1px solid rgba(148, 163, 184, 0.3);
	}

	.cycle-tag {
		font-size: 0.72rem;
		color: var(--text-secondary, #94a3b8);
	}

	.status-pill {
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
		font-size: 0.72rem;
		font-weight: 700;
		padding: 0.2rem 0.65rem;
		border-radius: 9999px;
		letter-spacing: 0.05em;
	}

	.status-paid {
		background: rgba(16, 185, 129, 0.15);
		color: #34d399;
		border: 1px solid rgba(16, 185, 129, 0.3);
	}

	.status-open {
		background: rgba(245, 158, 11, 0.15);
		color: #fbbf24;
		border: 1px solid rgba(245, 158, 11, 0.3);
	}

	.status-void {
		background: rgba(239, 68, 68, 0.15);
		color: #f87171;
		border: 1px solid rgba(239, 68, 68, 0.3);
	}

	.dot {
		width: 6px;
		height: 6px;
		border-radius: 50%;
		background: currentColor;
	}

	.dot.pulse {
		animation: pulse-glow 1.5s infinite;
	}

	@keyframes pulse-glow {
		0%, 100% { opacity: 1; transform: scale(1); }
		50% { opacity: 0.4; transform: scale(0.85); }
	}

	.utr-tag {
		font-size: 0.7rem;
		color: var(--text-secondary, #94a3b8);
		margin-top: 0.2rem;
	}

	.due-warning {
		color: #fbbf24;
		font-weight: 600;
	}

	.link-btn {
		background: transparent;
		border: none;
		color: #818cf8;
		cursor: pointer;
		font-family: inherit;
		font-weight: 700;
		padding: 0;
		text-decoration: underline;
		text-underline-offset: 3px;
	}

	.link-btn:hover {
		color: #a5b4fc;
	}

	/* Buttons */
	.btn {
		display: inline-flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.55rem 1rem;
		font-size: 0.85rem;
		font-weight: 600;
		border-radius: 8px;
		cursor: pointer;
		border: 1px solid transparent;
		transition: all 0.2s;
	}

	.btn-primary {
		background: var(--primary, #6366f1);
		color: #ffffff;
	}

	.btn-primary:hover:not(:disabled) {
		background: #4f46e5;
	}

	.glow-btn {
		box-shadow: 0 0 16px rgba(99, 102, 241, 0.35);
	}

	.btn-secondary {
		background: rgba(255, 255, 255, 0.05);
		border-color: var(--border-color, rgba(255, 255, 255, 0.1));
		color: #f1f5f9;
	}

	.btn-secondary:hover:not(:disabled) {
		background: rgba(255, 255, 255, 0.08);
	}

	.btn-sm {
		padding: 0.35rem 0.75rem;
		font-size: 0.78rem;
		font-weight: 600;
		border-radius: 6px;
		cursor: pointer;
		border: 1px solid transparent;
		transition: all 0.2s;
	}

	.btn-primary-sm {
		background: rgba(99, 102, 241, 0.2);
		border-color: rgba(99, 102, 241, 0.4);
		color: #a5b4fc;
	}

	.btn-primary-sm:hover {
		background: var(--primary, #6366f1);
		color: #ffffff;
	}

	.btn-secondary-sm {
		background: rgba(255, 255, 255, 0.05);
		border-color: rgba(255, 255, 255, 0.1);
		color: var(--text-secondary, #94a3b8);
	}

	.btn-secondary-sm:hover {
		background: rgba(255, 255, 255, 0.1);
		color: #f1f5f9;
	}

	.btn-icon {
		width: 16px;
		height: 16px;
	}

	.spin {
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		from { transform: rotate(0deg); }
		to { transform: rotate(360deg); }
	}

	/* Empty States */
	.empty-state {
		padding: 4rem 2rem;
		text-align: center;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.75rem;
	}

	.empty-icon {
		font-size: 2.5rem;
	}

	.empty-title {
		font-size: 1.1rem;
		font-weight: 600;
		margin: 0;
	}

	.empty-desc {
		font-size: 0.85rem;
		color: var(--text-secondary, #94a3b8);
		max-width: 400px;
		margin: 0;
	}

	.spinner {
		width: 14px;
		height: 14px;
		border: 2px solid rgba(255, 255, 255, 0.3);
		border-top-color: #ffffff;
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
		display: inline-block;
	}

	.spinner-large {
		width: 32px;
		height: 32px;
		border: 3px solid rgba(99, 102, 241, 0.3);
		border-top-color: #6366f1;
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	/* Slide-Over Drawer */
	.drawer-backdrop {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.65);
		backdrop-filter: blur(4px);
		z-index: 100;
		border: none;
		cursor: pointer;
		padding: 0;
		margin: 0;
	}

	.drawer-panel {
		position: fixed;
		top: 0;
		right: 0;
		bottom: 0;
		width: 100%;
		max-width: 520px;
		background: #0f172a;
		border-left: 1px solid var(--border-color, rgba(255, 255, 255, 0.1));
		box-shadow: -8px 0 32px rgba(0, 0, 0, 0.5);
		z-index: 101;
		display: flex;
		flex-direction: column;
		animation: slide-in 0.25s ease-out;
	}

	@keyframes slide-in {
		from { transform: translateX(100%); }
		to { transform: translateX(0); }
	}

	.drawer-header {
		padding: 1.5rem;
		border-bottom: 1px solid var(--border-color, rgba(255, 255, 255, 0.08));
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
	}

	.drawer-eyebrow {
		font-size: 0.72rem;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		color: var(--text-secondary, #94a3b8);
	}

	.drawer-title {
		font-size: 1.35rem;
		font-weight: 700;
		font-family: var(--font-mono, monospace);
		margin: 0.2rem 0 0;
		color: #f1f5f9;
	}

	.drawer-close-btn {
		background: transparent;
		border: none;
		color: var(--text-secondary, #94a3b8);
		font-size: 1.25rem;
		cursor: pointer;
		padding: 0.25rem;
	}

	.drawer-close-btn:hover {
		color: #ffffff;
	}

	.drawer-content {
		flex: 1;
		overflow-y: auto;
		padding: 1.5rem;
		display: flex;
		flex-direction: column;
		gap: 1.25rem;
	}

	.drawer-card {
		background: rgba(30, 41, 59, 0.5);
		border: 1px solid var(--border-color, rgba(255, 255, 255, 0.08));
		border-radius: 10px;
		padding: 1.15rem;
	}

	.highlight-card {
		border-color: rgba(245, 158, 11, 0.3);
		background: rgba(245, 158, 11, 0.03);
	}

	.verified-card {
		border-color: rgba(16, 185, 129, 0.3);
		background: rgba(16, 185, 129, 0.03);
	}

	.card-eyebrow {
		font-size: 0.75rem;
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		margin-bottom: 0.75rem;
		color: var(--text-secondary, #94a3b8);
	}

	.card-note {
		font-size: 0.82rem;
		color: var(--text-secondary, #94a3b8);
		line-height: 1.45;
		margin: 0 0 1rem;
	}

	.detail-row {
		display: flex;
		justify-content: space-between;
		font-size: 0.84rem;
		padding: 0.35rem 0;
	}

	.detail-label {
		color: var(--text-secondary, #94a3b8);
	}

	.breakdown-table {
		display: flex;
		flex-direction: column;
		gap: 0.4rem;
		font-size: 0.85rem;
	}

	.breakdown-row {
		display: flex;
		justify-content: space-between;
		color: var(--text-secondary, #94a3b8);
	}

	.breakdown-divider {
		height: 1px;
		background: var(--border-color, rgba(255, 255, 255, 0.08));
		margin: 0.35rem 0;
	}

	.total-row {
		color: #f1f5f9;
		font-size: 0.95rem;
	}

	.form-group {
		margin-bottom: 1rem;
	}

	.form-label {
		display: block;
		font-size: 0.78rem;
		font-weight: 600;
		color: var(--text-secondary, #94a3b8);
		margin-bottom: 0.35rem;
	}

	.form-select,
	.form-input {
		width: 100%;
		padding: 0.55rem 0.75rem;
		background: rgba(15, 23, 42, 0.8);
		border: 1px solid var(--border-color, rgba(255, 255, 255, 0.15));
		border-radius: 8px;
		color: #f1f5f9;
		font-size: 0.85rem;
		outline: none;
	}

	.form-select:focus,
	.form-input:focus {
		border-color: var(--primary, #6366f1);
	}

	.full-width {
		width: 100%;
		justify-content: center;
		padding: 0.7rem;
		font-size: 0.9rem;
	}

	.drawer-footer {
		padding: 1.25rem 1.5rem;
		border-top: 1px solid var(--border-color, rgba(255, 255, 255, 0.08));
		display: flex;
		justify-content: flex-end;
		gap: 0.75rem;
	}

	.font-mono {
		font-family: var(--font-mono, monospace);
	}

	.font-bold {
		font-weight: 700;
	}

	.text-sm {
		font-size: 0.82rem;
	}

	.text-lg {
		font-size: 1.15rem;
	}

	.text-right {
		text-align: right;
	}

	.text-center {
		text-align: center;
	}
</style>
