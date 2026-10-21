<script lang="ts">
	import { onMount } from 'svelte';
	import { api, type Invoice } from '$lib/api';
	import { toast } from '$lib/stores/toast.svelte';

	let invoices = $state<Invoice[]>([]);
	let loading = $state(true);
	let error = $state('');

	// Search & Filter state
	let searchQuery = $state('');
	let statusFilter = $state('ALL');

	// Slide-Over Drawer state
	let showDrawer = $state(false);
	let selectedInvoice = $state<Invoice | null>(null);
	let drawerLoading = $state(false);

	onMount(async () => {
		await loadInvoices();
	});

	async function loadInvoices() {
		loading = true;
		error = '';
		try {
			invoices = await api.listInvoices();
		} catch (err: any) {
			error = err.message || 'Failed to fetch invoices';
			toast.error('Could not load invoices', error);
		} finally {
			loading = false;
		}
	}

	function formatCurrency(val: string | number | undefined): string {
		if (val === undefined || val === null) return '₹0.00';
		const num = typeof val === 'string' ? parseFloat(val) : val;
		return new Intl.NumberFormat('en-IN', { style: 'currency', currency: 'INR' }).format(isNaN(num) ? 0 : num);
	}

	function isOverdue(dueDate: string, balanceDue: string | number): boolean {
		const bal = parseFloat(String(balanceDue)) || 0;
		if (bal <= 0.01) return false;
		return new Date(dueDate).getTime() < Date.now();
	}

	function getInitials(name: string | undefined): string {
		if (!name) return 'CU';
		const parts = name.trim().split(/\s+/);
		if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
		return (parts[0][0] + parts[1][0]).toUpperCase();
	}

	// Filtered Invoices
	let filteredInvoices = $derived(
		invoices.filter((inv) => {
			const q = searchQuery.toLowerCase().trim();
			const matchesSearch =
				!q ||
				inv.invoice_number.toLowerCase().includes(q) ||
				inv.customer_name.toLowerCase().includes(q) ||
				(inv.customer_gstin && inv.customer_gstin.toLowerCase().includes(q)) ||
				(inv.po_number && inv.po_number.toLowerCase().includes(q));

			let matchesStatus = true;
			if (statusFilter === 'OVERDUE') {
				matchesStatus = isOverdue(inv.due_date, inv.balance_due);
			} else if (statusFilter !== 'ALL') {
				matchesStatus = inv.status === statusFilter;
			}
			return matchesSearch && matchesStatus;
		})
	);

	// Cockpit Financial KPIs
	let totalInvoiced = $derived(
		invoices.reduce((acc, inv) => acc + (inv.status !== 'CANCELLED' ? parseFloat(inv.grand_total || '0') : 0), 0)
	);

	let totalOutstanding = $derived(
		invoices.reduce((acc, inv) => acc + (inv.status !== 'CANCELLED' ? parseFloat(inv.balance_due || '0') : 0), 0)
	);

	let totalSettled = $derived(
		invoices.reduce((acc, inv) => acc + (inv.status !== 'CANCELLED' ? parseFloat(inv.amount_paid || '0') : 0), 0)
	);

	let overdueCount = $derived(
		invoices.filter((inv) => isOverdue(inv.due_date, inv.balance_due)).length
	);

	async function openInvoiceDrawer(inv: Invoice) {
		selectedInvoice = inv;
		showDrawer = true;
		drawerLoading = true;
		try {
			const detailed = await api.getInvoice(inv.id);
			if (detailed) {
				selectedInvoice = { ...inv, ...detailed };
			}
		} catch (e) {
			// fallback to local data
		} finally {
			drawerLoading = false;
		}
	}

	function handleDrawerKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape' && showDrawer) {
			showDrawer = false;
		}
	}

	function copyShareText(inv: Invoice) {
		const text = `Tax Invoice #${inv.invoice_number} for ${inv.customer_name}: Amount ${formatCurrency(inv.grand_total)}, Balance Due: ${formatCurrency(inv.balance_due)}. Due Date: ${new Date(inv.due_date).toLocaleDateString('en-IN')}`;
		navigator.clipboard.writeText(text);
		toast.success('Copied to Clipboard!', 'Invoice summary copied for WhatsApp / email sharing');
	}

	function exportInvoicesCSV() {
		if (filteredInvoices.length === 0) {
			toast.warning('No Records', 'No invoices to export');
			return;
		}
		const headers = ['Invoice #', 'Customer Name', 'GSTIN', 'Invoice Date', 'Due Date', 'Taxable Amount', 'Grand Total', 'Amount Paid', 'Balance Due', 'Status'];
		const rows = filteredInvoices.map((inv) => [
			inv.invoice_number,
			`"${inv.customer_name.replace(/"/g, '""')}"`,
			inv.customer_gstin || 'Unregistered',
			new Date(inv.invoice_date).toLocaleDateString('en-IN'),
			new Date(inv.due_date).toLocaleDateString('en-IN'),
			inv.taxable_amount,
			inv.grand_total,
			inv.amount_paid,
			inv.balance_due,
			inv.status
		]);
		const csvContent = 'data:text/csv;charset=utf-8,' + [headers.join(','), ...rows.map((r) => r.join(','))].join('\n');
		const encodedUri = encodeURI(csvContent);
		const link = document.createElement('a');
		link.setAttribute('href', encodedUri);
		link.setAttribute('download', `sales_invoices_${new Date().toISOString().slice(0, 10)}.csv`);
		document.body.appendChild(link);
		link.click();
		document.body.removeChild(link);
		toast.success('Invoices Exported', 'Sales invoice register exported to CSV');
	}
</script>

<svelte:window onkeydown={handleDrawerKeydown} />

<svelte:head>
	<title>Sales Invoices & Receivables Cockpit | MrBiz-OS</title>
</svelte:head>

<div class="invoices-page-container animate-fade-in">
	<!-- Top Header -->
	<div class="invoices-header">
		<div>
			<div class="header-badge">
				<span>📄</span> SALES & TAX INVOICING
			</div>
			<h1 class="page-title">Sales Invoices</h1>
			<p class="page-subtitle">
				Audited GST sales invoices, payment allocations, e-way bills, and debtor aging records.
			</p>
		</div>

		<div class="header-actions">
			<button type="button" class="btn btn-secondary" onclick={exportInvoicesCSV} title="Export CSV Register">
				📥 Export CSV
			</button>
			<button type="button" class="btn btn-secondary" onclick={loadInvoices} title="Refresh records">
				🔄 Refresh
			</button>
			<a href="/invoices/new" class="btn btn-primary" style="box-shadow: 0 4px 15px rgba(99, 102, 241, 0.4);">
				+ Create New Invoice
			</a>
		</div>
	</div>

	<!-- Executive Financial KPI Strip -->
	<div class="kpi-grid">
		<div class="kpi-card">
			<div class="kpi-top">
				<span class="kpi-label">Total Invoiced Sales</span>
				<span class="kpi-icon indigo">📈</span>
			</div>
			<div class="kpi-value text-indigo">{formatCurrency(totalInvoiced)}</div>
			<div class="kpi-footer">Across {invoices.length} generated tax invoices</div>
		</div>

		<div class="kpi-card">
			<div class="kpi-top">
				<span class="kpi-label">Outstanding Debtors</span>
				<span class="kpi-icon amber">⏳</span>
			</div>
			<div class="kpi-value text-amber">{formatCurrency(totalOutstanding)}</div>
			<div class="kpi-footer">Pending customer collections</div>
		</div>

		<div class="kpi-card">
			<div class="kpi-top">
				<span class="kpi-label">Settled & Realized</span>
				<span class="kpi-icon emerald">✓</span>
			</div>
			<div class="kpi-value text-emerald">{formatCurrency(totalSettled)}</div>
			<div class="kpi-footer">
				{invoices.filter(i => i.status === 'PAID').length} fully settled invoices
			</div>
		</div>

		<div class="kpi-card">
			<div class="kpi-top">
				<span class="kpi-label">Overdue &gt; Terms</span>
				<span class="kpi-icon rose">⚠️</span>
			</div>
			<div class="kpi-value text-rose">{overdueCount} Invoices</div>
			<div class="kpi-footer">Due date has elapsed • Send reminder</div>
		</div>
	</div>

	<!-- Search & Filter Controls -->
	<div class="card controls-card">
		<div class="search-box">
			<span class="search-icon">🔍</span>
			<input
				type="text"
				class="form-input search-input"
				placeholder="Search by invoice #, customer name, GSTIN, PO #..."
				bind:value={searchQuery}
			/>
			{#if searchQuery}
				<button class="clear-search-btn" onclick={() => (searchQuery = '')}>✕</button>
			{/if}
		</div>

		<!-- Segmented Filter Pills -->
		<div class="filter-pills">
			<button
				type="button"
				class="pill-btn"
				class:active={statusFilter === 'ALL'}
				onclick={() => (statusFilter = 'ALL')}
			>
				All Invoices ({invoices.length})
			</button>
			<button
				type="button"
				class="pill-btn"
				class:active={statusFilter === 'ISSUED'}
				onclick={() => (statusFilter = 'ISSUED')}
			>
				Issued / Unpaid ({invoices.filter(i => i.status === 'ISSUED').length})
			</button>
			<button
				type="button"
				class="pill-btn"
				class:active={statusFilter === 'PARTIALLY_PAID'}
				onclick={() => (statusFilter = 'PARTIALLY_PAID')}
			>
				Partially Paid ({invoices.filter(i => i.status === 'PARTIALLY_PAID').length})
			</button>
			<button
				type="button"
				class="pill-btn"
				class:active={statusFilter === 'PAID'}
				onclick={() => (statusFilter = 'PAID')}
			>
				Paid ({invoices.filter(i => i.status === 'PAID').length})
			</button>
			<button
				type="button"
				class="pill-btn"
				class:active={statusFilter === 'OVERDUE'}
				onclick={() => (statusFilter = 'OVERDUE')}
			>
				Overdue ({overdueCount})
			</button>
			<button
				type="button"
				class="pill-btn"
				class:active={statusFilter === 'CANCELLED'}
				onclick={() => (statusFilter = 'CANCELLED')}
			>
				Cancelled ({invoices.filter(i => i.status === 'CANCELLED').length})
			</button>
		</div>
	</div>

	<!-- Main Invoices Table -->
	<div class="card table-card">
		<table class="invoices-table">
			<thead>
				<tr>
					<th>Invoice Number</th>
					<th>Customer & Client</th>
					<th>Invoice Date</th>
					<th>Due Date</th>
					<th class="text-right">Grand Total</th>
					<th class="text-right">Balance Due</th>
					<th>Status</th>
					<th class="text-right">Actions</th>
				</tr>
			</thead>
			<tbody>
				{#if loading}
					<tr>
						<td colspan="8" class="state-cell">
							<div class="loading-state">
								<div class="spinner"></div>
								<span>Loading sales invoices...</span>
							</div>
						</td>
					</tr>
				{:else if error}
					<tr>
						<td colspan="8" class="state-cell">
							<div class="error-state">
								<span>⚠️</span>
								<p>{error}</p>
								<button class="btn btn-secondary btn-sm" onclick={loadInvoices}>Retry</button>
							</div>
						</td>
					</tr>
				{:else if filteredInvoices.length === 0}
					<tr>
						<td colspan="8" class="state-cell">
							<div class="empty-state">
								<div class="empty-icon">📄</div>
								<h3>No invoices match the selected filter</h3>
								<p>
									{#if searchQuery || statusFilter !== 'ALL'}
										Try clearing your search query or choosing another status tab.
									{:else}
										Generate your first GST sales invoice with logistics and payment terms.
									{/if}
								</p>
								<a href="/invoices/new" class="btn btn-primary btn-sm" style="margin-top: 0.5rem;">
									+ Create First Invoice
								</a>
							</div>
						</td>
					</tr>
				{:else}
					{#each filteredInvoices as inv (inv.id)}
						{@const overdue = isOverdue(inv.due_date, inv.balance_due)}
						<tr class="inv-row" onclick={() => openInvoiceDrawer(inv)}>
							<!-- Invoice # -->
							<td>
								<div class="inv-num-cell">
									<a
										href={`/invoices/${inv.id}`}
										class="inv-num font-mono"
										onclick={(e) => e.stopPropagation()}
									>
										{inv.invoice_number}
									</a>
									<button
										type="button"
										class="icon-btn"
										title="Copy Invoice Summary"
										onclick={(e) => { e.stopPropagation(); copyShareText(inv); }}
									>
										📋
									</button>
								</div>
							</td>

							<!-- Customer -->
							<td>
								<div class="customer-cell">
									<div class="avatar">{getInitials(inv.customer_name)}</div>
									<div class="customer-info">
										<span class="customer-name">{inv.customer_name}</span>
										{#if inv.customer_gstin}
											<span class="customer-gstin font-mono">GSTIN: {inv.customer_gstin}</span>
										{/if}
									</div>
								</div>
							</td>

							<!-- Invoice Date -->
							<td class="date-cell">
								{new Date(inv.invoice_date).toLocaleDateString('en-IN', { day: '2-digit', month: 'short', year: 'numeric' })}
							</td>

							<!-- Due Date -->
							<td>
								<div class="due-cell">
									<span class:text-rose={overdue}>
										{new Date(inv.due_date).toLocaleDateString('en-IN', { day: '2-digit', month: 'short', year: 'numeric' })}
									</span>
									{#if overdue}
										<span class="overdue-tag">Overdue</span>
									{/if}
								</div>
							</td>

							<!-- Grand Total -->
							<td class="text-right">
								<span class="amount-val font-mono">{formatCurrency(inv.grand_total)}</span>
							</td>

							<!-- Balance Due -->
							<td class="text-right">
								<span
									class="balance-val font-mono"
									class:text-emerald={parseFloat(inv.balance_due) <= 0.01}
									class:text-amber={parseFloat(inv.balance_due) > 0.01 && !overdue}
									class:text-rose={overdue}
								>
									{formatCurrency(inv.balance_due)}
								</span>
							</td>

							<!-- Status -->
							<td>
								<span class="status-badge {inv.status.toLowerCase().replace('_', '-')}">
									<span class="status-dot"></span>
									{inv.status}
								</span>
							</td>

							<!-- Actions -->
							<td class="text-right" onclick={(e) => e.stopPropagation()}>
								<div class="actions-group">
									<button
										type="button"
										class="btn btn-secondary btn-sm"
										onclick={() => openInvoiceDrawer(inv)}
										title="Quick Preview Drawer"
									>
										👁️ Preview
									</button>

									<!-- Quick PDF Download Pills -->
									<div class="pdf-pills-group">
										<a
											href={`/api/invoices/${inv.id}/pdf?size=a4`}
											target="_blank"
											class="pdf-pill"
											title="Download Standard A4 PDF"
										>
											A4 ⬇
										</a>
										<a
											href={`/api/invoices/${inv.id}/pdf?size=a5`}
											target="_blank"
											class="pdf-pill"
											title="Download A5 Half-Sheet PDF"
										>
											A5 ⬇
										</a>
									</div>
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
<!-- SLIDE-OVER INVOICE PREVIEW DRAWER              -->
<!-- ============================================== -->
{#if showDrawer && selectedInvoice}
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div class="drawer-backdrop animate-fade-in" onclick={() => (showDrawer = false)}>
		<div class="drawer-panel animate-slide-left" onclick={(e) => e.stopPropagation()} role="dialog" aria-modal="true">
			<!-- Drawer Header -->
			<div class="drawer-header">
				<div style="display: flex; align-items: center; gap: 0.6rem;">
					<span style="font-size: 1.4rem;">📄</span>
					<div>
						<div class="drawer-title">{selectedInvoice.invoice_number}</div>
						<div class="drawer-subtitle">
							Tax Invoice for <strong>{selectedInvoice.customer_name}</strong>
						</div>
					</div>
				</div>
				<button type="button" class="drawer-close-btn" onclick={() => (showDrawer = false)}>✕</button>
			</div>

			<!-- Drawer Body -->
			<div class="drawer-body">
				<!-- Status Banner -->
				<div class="drawer-status-banner {selectedInvoice.status.toLowerCase().replace('_', '-')}">
					<div>
						<span style="font-weight: 700; text-transform: uppercase;">Status: {selectedInvoice.status}</span>
						<span style="font-size: 0.8125rem; opacity: 0.9; margin-left: 0.5rem;">
							Due Date: {new Date(selectedInvoice.due_date).toLocaleDateString('en-IN', { day: '2-digit', month: 'long', year: 'numeric' })}
						</span>
					</div>
					{#if parseFloat(selectedInvoice.balance_due) > 0.01}
						<a href={`/invoices/${selectedInvoice.id}`} class="btn btn-emerald btn-sm">
							💳 Settle Payment
						</a>
					{/if}
				</div>

				<!-- Financial Summary Cards -->
				<div class="drawer-meta-grid">
					<div class="meta-item">
						<span class="meta-label">Customer / Recipient</span>
						<span class="meta-val">{selectedInvoice.customer_name}</span>
					</div>
					<div class="meta-item">
						<span class="meta-label">GSTIN</span>
						<span class="meta-val font-mono">{selectedInvoice.customer_gstin || 'Unregistered'}</span>
					</div>
					<div class="meta-item">
						<span class="meta-label">Taxable Value</span>
						<span class="meta-val font-mono">{formatCurrency(selectedInvoice.taxable_amount)}</span>
					</div>
					<div class="meta-item">
						<span class="meta-label">Grand Total</span>
						<span class="meta-val font-mono" style="font-size: 1.15rem; font-weight: 800; color: #ffffff;">
							{formatCurrency(selectedInvoice.grand_total)}
						</span>
					</div>
					<div class="meta-item">
						<span class="meta-label">Amount Paid</span>
						<span class="meta-val font-mono text-emerald">{formatCurrency(selectedInvoice.amount_paid)}</span>
					</div>
					<div class="meta-item">
						<span class="meta-label">Balance Due</span>
						<span class="meta-val font-mono" class:text-amber={parseFloat(selectedInvoice.balance_due) > 0}>
							{formatCurrency(selectedInvoice.balance_due)}
						</span>
					</div>
				</div>

				<!-- Items Table -->
				<div style="margin-top: 1rem;">
					<div style="font-size: 0.8125rem; font-weight: 700; color: #cbd5e1; text-transform: uppercase; margin-bottom: 0.5rem;">
						Billed Items & Line Details
					</div>

					<div class="drawer-table-wrapper">
						<table class="drawer-items-table">
							<thead>
								<tr>
									<th>Description</th>
									<th class="text-right">Qty</th>
									<th class="text-right">Rate</th>
									<th class="text-right">Tax</th>
									<th class="text-right">Total</th>
								</tr>
							</thead>
							<tbody>
								{#if selectedInvoice.items && selectedInvoice.items.length > 0}
									{#each selectedInvoice.items as it}
										<tr>
											<td>
												<span style="font-weight: 600; color: #f8fafc;">{it.item_description}</span>
												{#if it.hsn_sac}
													<span class="font-mono text-xs" style="color: #818cf8; display: block;">HSN: {it.hsn_sac}</span>
												{/if}
											</td>
											<td class="text-right font-mono">{it.quantity}</td>
											<td class="text-right font-mono">{formatCurrency(it.unit_price)}</td>
											<td class="text-right font-mono">{it.tax_rate}%</td>
											<td class="text-right font-mono" style="font-weight: 700;">{formatCurrency(it.total_amount)}</td>
										</tr>
									{/each}
								{:else}
									<tr>
										<td>
											<span style="font-weight: 600; color: #f8fafc;">Standard Sales Invoice Deliverable</span>
										</td>
										<td class="text-right font-mono">1</td>
										<td class="text-right font-mono">{formatCurrency(selectedInvoice.taxable_amount)}</td>
										<td class="text-right font-mono">18%</td>
										<td class="text-right font-mono" style="font-weight: 700;">{formatCurrency(selectedInvoice.grand_total)}</td>
									</tr>
								{/if}
							</tbody>
							<tfoot>
								<tr>
									<td colspan="4" class="text-right" style="font-weight: 600; color: #94a3b8;">Grand Total:</td>
									<td class="text-right font-mono" style="font-size: 1rem; font-weight: 800; color: #ffffff;">
										{formatCurrency(selectedInvoice.grand_total)}
									</td>
								</tr>
							</tfoot>
						</table>
					</div>
				</div>

				<!-- Quick Share / Delivery Strip -->
				<div class="share-box">
					<div style="font-size: 0.8125rem; font-weight: 700; color: #f8fafc; margin-bottom: 0.25rem;">
						Direct Client Share & Downloads
					</div>
					<div style="display: flex; gap: 0.5rem; flex-wrap: wrap;">
						<button type="button" class="btn btn-secondary btn-sm" onclick={() => copyShareText(selectedInvoice!)}>
							📋 Copy WhatsApp Text
						</button>
						<a href={`/api/invoices/${selectedInvoice.id}/pdf?size=a4`} target="_blank" class="btn btn-secondary btn-sm">
							📄 Download A4 PDF
						</a>
						<a href={`/api/invoices/${selectedInvoice.id}/pdf?size=thermal_80mm`} target="_blank" class="btn btn-secondary btn-sm">
							🧾 80mm Thermal Receipt
						</a>
					</div>
				</div>
			</div>

			<!-- Drawer Footer -->
			<div class="drawer-footer">
				<a href={`/invoices/${selectedInvoice.id}`} class="btn btn-primary btn-sm">
					Open Full Invoice Details →
				</a>
				<button type="button" class="btn btn-secondary btn-sm" onclick={() => (showDrawer = false)}>
					Close
				</button>
			</div>
		</div>
	</div>
{/if}

<style>
	.invoices-page-container {
		display: flex;
		flex-direction: column;
		gap: 1.5rem;
		max-width: 1400px;
		margin: 0 auto;
	}

	.invoices-header {
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
		color: #818cf8;
		background: rgba(99, 102, 241, 0.12);
		padding: 0.2rem 0.55rem;
		border-radius: 6px;
		border: 1px solid rgba(99, 102, 241, 0.25);
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
	.kpi-icon.indigo { background: rgba(99, 102, 241, 0.15); color: #818cf8; }
	.kpi-icon.amber { background: rgba(245, 158, 11, 0.15); color: #fbbf24; }
	.kpi-icon.emerald { background: rgba(16, 185, 129, 0.15); color: #34d399; }
	.kpi-icon.rose { background: rgba(244, 63, 94, 0.15); color: #fb7185; }

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
		border-color: #6366f1;
		box-shadow: 0 0 10px rgba(99, 102, 241, 0.25);
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
		background: rgba(99, 102, 241, 0.2);
		border-color: #6366f1;
		color: #a5b4fc;
		font-weight: 700;
	}

	/* Table Card */
	.table-card {
		padding: 0;
		overflow: hidden;
		border: 1px solid rgba(255, 255, 255, 0.08);
	}

	.invoices-table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.875rem;
		text-align: left;
	}

	.invoices-table th {
		background: rgba(15, 23, 42, 0.85);
		padding: 0.875rem 1.25rem;
		color: #94a3b8;
		font-size: 0.75rem;
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 0.04em;
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
	}

	.invoices-table td {
		padding: 1rem 1.25rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.04);
		color: #f8fafc;
		vertical-align: middle;
	}

	.inv-row {
		cursor: pointer;
		transition: background 0.15s ease;
	}

	.inv-row:hover {
		background: rgba(255, 255, 255, 0.03);
	}

	.inv-num-cell {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.inv-num {
		font-weight: 700;
		color: #818cf8;
		text-decoration: none;
	}
	.inv-num:hover {
		text-decoration: underline;
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

	.customer-cell {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}

	.avatar {
		width: 32px;
		height: 32px;
		border-radius: 8px;
		background: linear-gradient(135deg, rgba(99, 102, 241, 0.3) 0%, rgba(168, 85, 247, 0.3) 100%);
		border: 1px solid rgba(99, 102, 241, 0.3);
		color: #e0e7ff;
		display: flex;
		align-items: center;
		justify-content: center;
		font-weight: 700;
		font-size: 0.75rem;
		flex-shrink: 0;
	}

	.customer-info {
		display: flex;
		flex-direction: column;
		min-width: 0;
	}

	.customer-name {
		font-weight: 600;
		color: #ffffff;
	}

	.customer-gstin {
		font-size: 0.7rem;
		color: #94a3b8;
	}

	.date-cell {
		color: #94a3b8;
		font-size: 0.8125rem;
	}

	.due-cell {
		display: flex;
		flex-direction: column;
		gap: 0.2rem;
		font-size: 0.8125rem;
	}

	.overdue-tag {
		font-size: 0.65rem;
		font-weight: 700;
		color: #fb7185;
		background: rgba(244, 63, 94, 0.15);
		padding: 0.1rem 0.35rem;
		border-radius: 4px;
		width: fit-content;
		border: 1px solid rgba(244, 63, 94, 0.3);
	}

	.amount-val {
		font-weight: 700;
		color: #ffffff;
	}

	.balance-val {
		font-weight: 800;
	}

	/* Status Badge */
	.status-badge {
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
		padding: 0.25rem 0.6rem;
		border-radius: 9999px;
		font-size: 0.725rem;
		font-weight: 700;
		letter-spacing: 0.03em;
		text-transform: uppercase;
	}

	.status-dot {
		width: 6px;
		height: 6px;
		border-radius: 50%;
	}

	.status-badge.paid { background: rgba(16, 185, 129, 0.15); color: #34d399; border: 1px solid rgba(16, 185, 129, 0.3); }
	.status-badge.paid .status-dot { background: #10b981; }

	.status-badge.partially-paid { background: rgba(245, 158, 11, 0.15); color: #fbbf24; border: 1px solid rgba(245, 158, 11, 0.3); }
	.status-badge.partially-paid .status-dot { background: #f59e0b; }

	.status-badge.issued { background: rgba(56, 189, 248, 0.15); color: #38bdf8; border: 1px solid rgba(56, 189, 248, 0.3); }
	.status-badge.issued .status-dot { background: #38bdf8; }

	.status-badge.cancelled { background: rgba(244, 63, 94, 0.15); color: #fb7185; border: 1px solid rgba(244, 63, 94, 0.3); }
	.status-badge.cancelled .status-dot { background: #f43f5e; }

	.actions-group {
		display: flex;
		align-items: center;
		justify-content: flex-end;
		gap: 0.4rem;
	}

	.pdf-pills-group {
		display: inline-flex;
		border: 1px solid rgba(99, 102, 241, 0.4);
		border-radius: 6px;
		overflow: hidden;
	}

	.pdf-pill {
		background: rgba(99, 102, 241, 0.1);
		color: #a5b4fc;
		padding: 0.25rem 0.5rem;
		font-size: 0.725rem;
		font-weight: 600;
		text-decoration: none;
		transition: all 0.15s ease;
	}
	.pdf-pill:hover {
		background: rgba(99, 102, 241, 0.25);
		color: #ffffff;
	}
	.pdf-pill:not(:first-child) {
		border-left: 1px solid rgba(99, 102, 241, 0.3);
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
		border: 3px solid rgba(99, 102, 241, 0.2);
		border-top-color: #6366f1;
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
		max-width: 560px;
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
		color: #818cf8;
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
		display: flex;
		flex-direction: column;
		gap: 1.25rem;
	}

	.drawer-status-banner {
		padding: 0.875rem 1rem;
		border-radius: 8px;
		display: flex;
		justify-content: space-between;
		align-items: center;
		flex-wrap: wrap;
		gap: 0.5rem;
	}
	.drawer-status-banner.paid { background: rgba(16, 185, 129, 0.12); border: 1px solid rgba(16, 185, 129, 0.3); color: #34d399; }
	.drawer-status-banner.partially-paid { background: rgba(245, 158, 11, 0.12); border: 1px solid rgba(245, 158, 11, 0.3); color: #fbbf24; }
	.drawer-status-banner.issued { background: rgba(56, 189, 248, 0.12); border: 1px solid rgba(56, 189, 248, 0.3); color: #38bdf8; }
	.drawer-status-banner.cancelled { background: rgba(244, 63, 94, 0.12); border: 1px solid rgba(244, 63, 94, 0.3); color: #fb7185; }

	.btn-emerald {
		background: #10b981;
		color: #ffffff;
		border: 1px solid #059669;
		text-decoration: none;
	}

	.drawer-meta-grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 0.75rem;
		background: rgba(15, 23, 42, 0.6);
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 8px;
		padding: 1rem;
	}

	.meta-item {
		display: flex;
		flex-direction: column;
		gap: 0.2rem;
	}

	.meta-label {
		font-size: 0.6875rem;
		color: #94a3b8;
		text-transform: uppercase;
		letter-spacing: 0.04em;
	}

	.meta-val {
		font-weight: 600;
		color: #ffffff;
		font-size: 0.875rem;
	}

	.drawer-table-wrapper {
		background: rgba(15, 23, 42, 0.6);
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 8px;
		overflow: hidden;
	}

	.drawer-items-table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.8125rem;
	}

	.drawer-items-table th {
		background: rgba(0, 0, 0, 0.3);
		padding: 0.6rem 0.75rem;
		color: #94a3b8;
		font-size: 0.6875rem;
		text-transform: uppercase;
		border-bottom: 1px solid rgba(255, 255, 255, 0.06);
	}

	.drawer-items-table td {
		padding: 0.6rem 0.75rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.04);
		color: #cbd5e1;
	}

	.drawer-items-table tfoot td {
		background: rgba(0, 0, 0, 0.2);
		border-top: 1px solid rgba(255, 255, 255, 0.08);
		padding: 0.75rem;
	}

	.share-box {
		background: rgba(15, 23, 42, 0.5);
		border: 1px dashed rgba(255, 255, 255, 0.12);
		border-radius: 8px;
		padding: 1rem;
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
		.invoices-header { flex-direction: column; align-items: flex-start; }
		.header-actions { width: 100%; justify-content: space-between; }
		.kpi-grid { grid-template-columns: 1fr 1fr; }
		.drawer-panel { max-width: 100%; }
	}
</style>
