<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { api } from '$lib/api';
	import { toast } from '$lib/stores/toast.svelte';

	interface QuotationItem {
		id: string;
		description: string;
		hsn_sac?: string;
		quantity: number;
		unit_price: number;
		tax_rate: number;
		amount: number;
	}

	interface Quotation {
		id: string;
		quote_number: string;
		customer_name: string;
		customer_email?: string;
		customer_phone?: string;
		customer_gstin?: string;
		quote_date: string;
		valid_until: string;
		subtotal: number;
		tax_amount: number;
		grand_total: number;
		status: 'DRAFT' | 'SENT' | 'ACCEPTED' | 'REJECTED' | 'CONVERTED';
		notes?: string;
		terms?: string;
		expired?: boolean;
		items?: QuotationItem[];
	}

	let list = $state<Quotation[]>([]);
	let loading = $state(true);
	let error = $state('');

	// Search & Filter state
	let searchQuery = $state('');
	let statusFilter = $state<string>('ALL');

	// Slide-Over Preview Drawer
	let showDrawer = $state(false);
	let selectedQuote = $state<Quotation | null>(null);
	let drawerLoading = $state(false);
	let convertingId = $state<string | null>(null);

	onMount(async () => {
		await loadQuotations();
	});

	async function loadQuotations() {
		loading = true;
		error = '';
		try {
			const data = await api.listQuotations();
			list = data || [];
		} catch (e: any) {
			error = e.message || 'Failed to load quotations';
			toast.error('Could not load quotations', error);
		} finally {
			loading = false;
		}
	}

	function formatCurrency(val: string | number | undefined): string {
		if (val === undefined || val === null) return '₹0.00';
		const num = typeof val === 'string' ? parseFloat(val) : val;
		return new Intl.NumberFormat('en-IN', { style: 'currency', currency: 'INR' }).format(isNaN(num) ? 0 : num);
	}

	function getInitials(name: string | undefined): string {
		if (!name) return 'CU';
		const parts = name.trim().split(/\s+/);
		if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
		return (parts[0][0] + parts[1][0]).toUpperCase();
	}

	function isExpiringSoon(validUntil: string): boolean {
		if (!validUntil) return false;
		const target = new Date(validUntil).getTime();
		const now = Date.now();
		const diffDays = (target - now) / (1000 * 60 * 60 * 24);
		return diffDays >= 0 && diffDays <= 7;
	}

	function isExpiredDate(validUntil: string): boolean {
		if (!validUntil) return false;
		return new Date(validUntil).getTime() < Date.now();
	}

	// Filtered Quotes
	let filteredQuotes = $derived(
		list.filter((q) => {
			const query = searchQuery.toLowerCase().trim();
			const matchesQuery =
				!query ||
				q.quote_number.toLowerCase().includes(query) ||
				q.customer_name.toLowerCase().includes(query) ||
				(q.customer_gstin && q.customer_gstin.toLowerCase().includes(query)) ||
				(q.customer_phone && q.customer_phone.includes(query));

			let matchesStatus = true;
			if (statusFilter === 'EXPIRED') {
				matchesStatus = q.expired || isExpiredDate(q.valid_until);
			} else if (statusFilter !== 'ALL') {
				matchesStatus = q.status === statusFilter;
			}
			return matchesQuery && matchesStatus;
		})
	);

	// KPI Metrics
	let totalQuotedAmount = $derived(
		list.reduce((acc, q) => acc + (parseFloat(String(q.grand_total)) || 0), 0)
	);

	let totalAcceptedOrConverted = $derived(
		list
			.filter((q) => q.status === 'ACCEPTED' || q.status === 'CONVERTED')
			.reduce((acc, q) => acc + (parseFloat(String(q.grand_total)) || 0), 0)
	);

	let pendingDecisionCount = $derived(
		list.filter((q) => q.status === 'SENT' || q.status === 'DRAFT').length
	);

	let expiringSoonCount = $derived(
		list.filter((q) => q.status !== 'CONVERTED' && (isExpiringSoon(q.valid_until) || q.expired || isExpiredDate(q.valid_until))).length
	);

	async function openPreviewDrawer(quote: Quotation) {
		selectedQuote = quote;
		showDrawer = true;
		drawerLoading = true;
		try {
			const detailed = await api.getQuotation(quote.id);
			if (detailed) {
				selectedQuote = { ...quote, ...detailed };
			}
		} catch (e) {
			// Keep local data
		} finally {
			drawerLoading = false;
		}
	}

	async function handleConvertToInvoice(quote: Quotation, e?: MouseEvent) {
		if (e) e.stopPropagation();
		if (!confirm(`Convert quotation ${quote.quote_number} into a finalized GST Tax Invoice?`)) {
			return;
		}
		convertingId = quote.id;
		try {
			const res = await api.convertQuotation(quote.id);
			toast.success('Converted to Tax Invoice!', `Invoice created successfully from ${quote.quote_number}`);
			if (selectedQuote && selectedQuote.id === quote.id) {
				selectedQuote.status = 'CONVERTED';
			}
			await loadQuotations();
			if (res?.invoice_id) {
				goto(`/invoices/${res.invoice_id}`);
			}
		} catch (err: any) {
			toast.error('Conversion Failed', err.message || 'Could not convert quotation');
		} finally {
			convertingId = null;
		}
	}

	async function handleStatusChange(quote: Quotation, newStatus: string, e?: Event) {
		if (e) (e.target as HTMLSelectElement).blur();
		try {
			await api.setQuotationStatus(quote.id, newStatus);
			quote.status = newStatus as any;
			toast.info('Status Updated', `${quote.quote_number} is now marked as ${newStatus}`);
		} catch (err: any) {
			toast.error('Update Failed', err.message || 'Could not update status');
		}
	}

	function handleDrawerKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape' && showDrawer) {
			showDrawer = false;
		}
	}

	function copyShareText(quote: Quotation) {
		const text = `Quotation ${quote.quote_number} for ${quote.customer_name}: Amount ${formatCurrency(quote.grand_total)}. Valid until: ${new Date(quote.valid_until).toLocaleDateString('en-IN')}`;
		navigator.clipboard.writeText(text);
		toast.success('Copied to Clipboard!', 'Quotation summary copied to clipboard');
	}
</script>

<svelte:window onkeydown={handleDrawerKeydown} />

<svelte:head>
	<title>Quotations & Proforma Estimates | MrBiz-OS</title>
</svelte:head>

<div class="quotes-page-container animate-fade-in">
	<!-- Top Cockpit Header -->
	<div class="quotes-header">
		<div>
			<div class="quotes-header-badge">
				<span>📝</span> COMMERCIAL PIPELINE
			</div>
			<h1 class="quotes-title">Quotations & Proforma Estimates</h1>
			<p class="quotes-subtitle">
				Create estimates, track client approval decisions, and convert directly into GST Tax Invoices in 1-click.
			</p>
		</div>

		<div class="quotes-header-actions">
			<a href="/quotations/new" class="btn btn-primary" style="box-shadow: 0 4px 15px rgba(99, 102, 241, 0.4);">
				+ New Quotation
			</a>
			<button type="button" class="btn btn-secondary" onclick={loadQuotations} title="Refresh records">
				🔄 Refresh
			</button>
		</div>
	</div>

	<!-- Executive Financial KPI Strip -->
	<div class="kpi-grid">
		<div class="kpi-card">
			<div class="kpi-top">
				<span class="kpi-label">Total Pipeline Quoted</span>
				<span class="kpi-icon indigo">📊</span>
			</div>
			<div class="kpi-value">{formatCurrency(totalQuotedAmount)}</div>
			<div class="kpi-footer">Across {list.length} total client estimates</div>
		</div>

		<div class="kpi-card">
			<div class="kpi-top">
				<span class="kpi-label">Accepted & Converted</span>
				<span class="kpi-icon emerald">✓</span>
			</div>
			<div class="kpi-value text-emerald">{formatCurrency(totalAcceptedOrConverted)}</div>
			<div class="kpi-footer">
				{list.filter(q => q.status === 'ACCEPTED' || q.status === 'CONVERTED').length} deals closed successfully
			</div>
		</div>

		<div class="kpi-card">
			<div class="kpi-top">
				<span class="kpi-label">Pending Client Review</span>
				<span class="kpi-icon sky">⏳</span>
			</div>
			<div class="kpi-value text-sky">{pendingDecisionCount} Active</div>
			<div class="kpi-footer">Drafts & dispatched proformas awaiting sign-off</div>
		</div>

		<div class="kpi-card">
			<div class="kpi-top">
				<span class="kpi-label">Expiring / Expired</span>
				<span class="kpi-icon amber">⚠️</span>
			</div>
			<div class="kpi-value text-amber">{expiringSoonCount} Deals</div>
			<div class="kpi-footer">Requires price re-validation or close followup</div>
		</div>
	</div>

	<!-- Search & Filter Bar -->
	<div class="card controls-card">
		<div class="search-box">
			<span class="search-icon">🔍</span>
			<input
				type="text"
				class="form-input search-input"
				placeholder="Search by quote #, customer name, GSTIN, phone..."
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
				All ({list.length})
			</button>
			<button
				type="button"
				class="pill-btn"
				class:active={statusFilter === 'SENT'}
				onclick={() => (statusFilter = 'SENT')}
			>
				Sent ({list.filter(q => q.status === 'SENT').length})
			</button>
			<button
				type="button"
				class="pill-btn"
				class:active={statusFilter === 'ACCEPTED'}
				onclick={() => (statusFilter = 'ACCEPTED')}
			>
				Accepted ({list.filter(q => q.status === 'ACCEPTED').length})
			</button>
			<button
				type="button"
				class="pill-btn"
				class:active={statusFilter === 'CONVERTED'}
				onclick={() => (statusFilter = 'CONVERTED')}
			>
				Converted ({list.filter(q => q.status === 'CONVERTED').length})
			</button>
			<button
				type="button"
				class="pill-btn"
				class:active={statusFilter === 'DRAFT'}
				onclick={() => (statusFilter = 'DRAFT')}
			>
				Drafts ({list.filter(q => q.status === 'DRAFT').length})
			</button>
			<button
				type="button"
				class="pill-btn"
				class:active={statusFilter === 'EXPIRED'}
				onclick={() => (statusFilter = 'EXPIRED')}
			>
				Expiring / Expired ({expiringSoonCount})
			</button>
		</div>
	</div>

	<!-- Main Table Container -->
	<div class="card table-card">
		<table class="quotes-table">
			<thead>
				<tr>
					<th>Quote Number</th>
					<th>Customer & Prospect</th>
					<th>Issued Date</th>
					<th>Valid Until</th>
					<th class="text-right">Estimate Total</th>
					<th>Status</th>
					<th class="text-right">Actions</th>
				</tr>
			</thead>
			<tbody>
				{#if loading}
					<tr>
						<td colspan="7" class="state-cell">
							<div class="loading-state">
								<div class="spinner"></div>
								<span>Loading client quotations...</span>
							</div>
						</td>
					</tr>
				{:else if error}
					<tr>
						<td colspan="7" class="state-cell">
							<div class="error-state">
								<span>⚠️</span>
								<p>{error}</p>
								<button class="btn btn-secondary btn-sm" onclick={loadQuotations}>Try Again</button>
							</div>
						</td>
					</tr>
				{:else if filteredQuotes.length === 0}
					<tr>
						<td colspan="7" class="state-cell">
							<div class="empty-state">
								<div class="empty-icon">📝</div>
								<h3>No quotations found</h3>
								<p>
									{#if searchQuery || statusFilter !== 'ALL'}
										No quotes match your filter criteria. Try clearing search.
									{:else}
										Create your first proforma estimate or pricing proposal.
									{/if}
								</p>
								<a href="/quotations/new" class="btn btn-primary btn-sm" style="margin-top: 0.5rem;">
									+ Create First Quotation
								</a>
							</div>
						</td>
					</tr>
				{:else}
					{#each filteredQuotes as q (q.id)}
						<tr class="quote-row" onclick={() => openPreviewDrawer(q)}>
							<!-- Quote # -->
							<td>
								<div class="quote-number-cell">
									<span class="quote-num font-mono">{q.quote_number}</span>
									<button
										type="button"
										class="icon-btn"
										title="Copy Quote Summary"
										onclick={(e) => { e.stopPropagation(); copyShareText(q); }}
									>
										📋
									</button>
								</div>
							</td>

							<!-- Customer -->
							<td>
								<div class="customer-cell">
									<div class="avatar">{getInitials(q.customer_name)}</div>
									<div class="customer-info">
										<span class="customer-name">{q.customer_name}</span>
										{#if q.customer_gstin}
											<span class="customer-gstin font-mono">GSTIN: {q.customer_gstin}</span>
										{:else if q.customer_phone}
											<span class="customer-phone font-mono">{q.customer_phone}</span>
										{/if}
									</div>
								</div>
							</td>

							<!-- Date -->
							<td class="date-cell">
								{new Date(q.quote_date).toLocaleDateString('en-IN', { day: '2-digit', month: 'short', year: 'numeric' })}
							</td>

							<!-- Valid Until -->
							<td>
								<div class="validity-cell">
									<span>{new Date(q.valid_until).toLocaleDateString('en-IN', { day: '2-digit', month: 'short', year: 'numeric' })}</span>
									{#if q.status !== 'CONVERTED'}
										{#if q.expired || isExpiredDate(q.valid_until)}
											<span class="urgency-badge expired">Expired</span>
										{:else if isExpiringSoon(q.valid_until)}
											<span class="urgency-badge soon">&lt; 7 Days</span>
										{/if}
									{/if}
								</div>
							</td>

							<!-- Total -->
							<td class="text-right">
								<span class="amount-val font-mono">{formatCurrency(q.grand_total)}</span>
							</td>

							<!-- Status -->
							<td>
								<div class="status-cell" onclick={(e) => e.stopPropagation()}>
									<span class="status-badge {q.status.toLowerCase()}">
										<span class="status-dot"></span>
										{q.status}
									</span>
									<select
										class="status-select"
										value={q.status}
										onchange={(e) => handleStatusChange(q, (e.target as HTMLSelectElement).value, e)}
										title="Change Status"
									>
										<option value="DRAFT">DRAFT</option>
										<option value="SENT">SENT</option>
										<option value="ACCEPTED">ACCEPTED</option>
										<option value="REJECTED">REJECTED</option>
										<option value="CONVERTED">CONVERTED</option>
									</select>
								</div>
							</td>

							<!-- Actions -->
							<td class="text-right" onclick={(e) => e.stopPropagation()}>
								<div class="actions-group">
									{#if q.status !== 'CONVERTED'}
										<button
											type="button"
											class="btn btn-emerald btn-sm"
											onclick={(e) => handleConvertToInvoice(q, e)}
											disabled={convertingId === q.id}
											title="Convert directly to GST Tax Invoice"
										>
											{#if convertingId === q.id}
												⏳ Converting...
											{:else}
												⚡ Convert to Invoice
											{/if}
										</button>
									{:else}
										<span class="converted-tag">✓ Invoiced</span>
									{/if}

									<button
										type="button"
										class="btn btn-secondary btn-sm"
										onclick={() => openPreviewDrawer(q)}
										title="View Quote Details"
									>
										👁️ Preview
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
<!-- SLIDE-OVER QUOTATION PREVIEW DRAWER            -->
<!-- ============================================== -->
{#if showDrawer && selectedQuote}
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div class="drawer-backdrop animate-fade-in" onclick={() => (showDrawer = false)}>
		<div class="drawer-panel animate-slide-left" onclick={(e) => e.stopPropagation()} role="dialog" aria-modal="true">
			<!-- Drawer Header -->
			<div class="drawer-header">
				<div style="display: flex; align-items: center; gap: 0.6rem;">
					<span style="font-size: 1.4rem;">📝</span>
					<div>
						<div class="drawer-title">{selectedQuote.quote_number}</div>
						<div class="drawer-subtitle">
							Proforma Estimate for <strong>{selectedQuote.customer_name}</strong>
						</div>
					</div>
				</div>
				<button type="button" class="drawer-close-btn" onclick={() => (showDrawer = false)}>✕</button>
			</div>

			<!-- Drawer Body -->
			<div class="drawer-body">
				<!-- Status Banner -->
				<div class="drawer-status-banner {selectedQuote.status.toLowerCase()}">
					<div>
						<span style="font-weight: 700; text-transform: uppercase;">Status: {selectedQuote.status}</span>
						<span style="font-size: 0.8125rem; opacity: 0.9; margin-left: 0.5rem;">
							Valid until {new Date(selectedQuote.valid_until).toLocaleDateString('en-IN', { day: '2-digit', month: 'long', year: 'numeric' })}
						</span>
					</div>
					{#if selectedQuote.status !== 'CONVERTED'}
						<button
							type="button"
							class="btn btn-emerald btn-sm"
							onclick={() => handleConvertToInvoice(selectedQuote!)}
							disabled={convertingId === selectedQuote.id}
						>
							⚡ Convert to Tax Invoice
						</button>
					{/if}
				</div>

				<!-- Quick Summary Grid -->
				<div class="drawer-meta-grid">
					<div class="meta-item">
						<span class="meta-label">Customer / Client</span>
						<span class="meta-val">{selectedQuote.customer_name}</span>
					</div>
					<div class="meta-item">
						<span class="meta-label">GSTIN</span>
						<span class="meta-val font-mono">{selectedQuote.customer_gstin || 'Unregistered'}</span>
					</div>
					<div class="meta-item">
						<span class="meta-label">Quote Date</span>
						<span class="meta-val">{new Date(selectedQuote.quote_date).toLocaleDateString('en-IN')}</span>
					</div>
					<div class="meta-item">
						<span class="meta-label">Total Amount</span>
						<span class="meta-val font-mono text-emerald" style="font-size: 1.1rem; font-weight: 800;">
							{formatCurrency(selectedQuote.grand_total)}
						</span>
					</div>
				</div>

				<!-- Items Breakdown Table -->
				<div style="margin-top: 1.25rem;">
					<div style="font-size: 0.8125rem; font-weight: 700; color: #cbd5e1; text-transform: uppercase; margin-bottom: 0.5rem;">
						Items & Service Breakdown
					</div>

					<div class="drawer-table-wrapper">
						<table class="drawer-items-table">
							<thead>
								<tr>
									<th>Description</th>
									<th class="text-right">Qty</th>
									<th class="text-right">Rate</th>
									<th class="text-right">Tax</th>
									<th class="text-right">Amount</th>
								</tr>
							</thead>
							<tbody>
								{#if selectedQuote.items && selectedQuote.items.length > 0}
									{#each selectedQuote.items as it}
										<tr>
											<td>
												<span style="font-weight: 600; color: #f8fafc;">{it.description}</span>
												{#if it.hsn_sac}
													<span class="font-mono text-xs" style="color: #818cf8; display: block;">HSN: {it.hsn_sac}</span>
												{/if}
											</td>
											<td class="text-right font-mono">{it.quantity}</td>
											<td class="text-right font-mono">{formatCurrency(it.unit_price)}</td>
											<td class="text-right font-mono">{it.tax_rate}%</td>
											<td class="text-right font-mono" style="font-weight: 700;">{formatCurrency(it.amount)}</td>
										</tr>
									{/each}
								{:else}
									<tr>
										<td>
											<span style="font-weight: 600; color: #f8fafc;">Standard Professional Deliverable / SOW Scope</span>
										</td>
										<td class="text-right font-mono">1</td>
										<td class="text-right font-mono">{formatCurrency(selectedQuote.grand_total)}</td>
										<td class="text-right font-mono">18%</td>
										<td class="text-right font-mono" style="font-weight: 700;">{formatCurrency(selectedQuote.grand_total)}</td>
									</tr>
								{/if}
							</tbody>
							<tfoot>
								<tr>
									<td colspan="4" class="text-right" style="font-weight: 600; color: #94a3b8;">Grand Total:</td>
									<td class="text-right font-mono text-emerald" style="font-size: 1rem; font-weight: 800;">
										{formatCurrency(selectedQuote.grand_total)}
									</td>
								</tr>
							</tfoot>
						</table>
					</div>
				</div>

				<!-- Notes / Terms -->
				{#if selectedQuote.notes || selectedQuote.terms}
					<div class="drawer-notes-box">
						{#if selectedQuote.notes}
							<div style="font-size: 0.8125rem; color: #cbd5e1;">
								<strong>Notes:</strong> {selectedQuote.notes}
							</div>
						{/if}
						{#if selectedQuote.terms}
							<div style="font-size: 0.75rem; color: #94a3b8; margin-top: 0.35rem;">
								<strong>Terms & Conditions:</strong> {selectedQuote.terms}
							</div>
						{/if}
					</div>
				{/if}
			</div>

			<!-- Drawer Footer Actions -->
			<div class="drawer-footer">
				<button type="button" class="btn btn-secondary btn-sm" onclick={() => copyShareText(selectedQuote!)}>
					📋 Copy Summary
				</button>
				<a href={`/quotations/${selectedQuote.id}`} class="btn btn-secondary btn-sm">
					📄 Full Detail Page
				</a>
				<button type="button" class="btn btn-secondary btn-sm" onclick={() => window.print()}>
					🖨️ Print Quote
				</button>
				<button type="button" class="btn btn-primary btn-sm" onclick={() => (showDrawer = false)}>
					Done
				</button>
			</div>
		</div>
	</div>
{/if}

<style>
	.quotes-page-container {
		display: flex;
		flex-direction: column;
		gap: 1.5rem;
		max-width: 1400px;
		margin: 0 auto;
	}

	.quotes-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		flex-wrap: wrap;
		gap: 1.25rem;
	}

	.quotes-header-badge {
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

	.quotes-title {
		font-size: 1.75rem;
		font-weight: 800;
		margin: 0;
		color: #ffffff;
	}

	.quotes-subtitle {
		font-size: 0.875rem;
		color: #94a3b8;
		margin: 0.25rem 0 0 0;
	}

	.quotes-header-actions {
		display: flex;
		align-items: center;
		gap: 0.75rem;
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
	}
	.kpi-icon.indigo { background: rgba(99, 102, 241, 0.15); color: #818cf8; }
	.kpi-icon.emerald { background: rgba(16, 185, 129, 0.15); color: #34d399; }
	.kpi-icon.sky { background: rgba(14, 165, 233, 0.15); color: #38bdf8; }
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
		gap: 0.4rem;
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

	.quotes-table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.875rem;
		text-align: left;
	}

	.quotes-table th {
		background: rgba(15, 23, 42, 0.85);
		padding: 0.875rem 1.25rem;
		color: #94a3b8;
		font-size: 0.75rem;
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 0.04em;
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
	}

	.quotes-table td {
		padding: 1rem 1.25rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.04);
		color: #f8fafc;
		vertical-align: middle;
	}

	.quote-row {
		cursor: pointer;
		transition: background 0.15s ease;
	}

	.quote-row:hover {
		background: rgba(255, 255, 255, 0.03);
	}

	.quote-number-cell {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.quote-num {
		font-weight: 700;
		color: #818cf8;
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

	.customer-phone {
		font-size: 0.7rem;
		color: #64748b;
	}

	.date-cell {
		color: #94a3b8;
		font-size: 0.8125rem;
	}

	.validity-cell {
		display: flex;
		flex-direction: column;
		gap: 0.2rem;
		font-size: 0.8125rem;
		color: #cbd5e1;
	}

	.urgency-badge {
		font-size: 0.65rem;
		font-weight: 700;
		padding: 0.1rem 0.35rem;
		border-radius: 4px;
		display: inline-block;
		width: fit-content;
	}
	.urgency-badge.expired {
		background: rgba(244, 63, 94, 0.15);
		color: #fb7185;
		border: 1px solid rgba(244, 63, 94, 0.3);
	}
	.urgency-badge.soon {
		background: rgba(245, 158, 11, 0.15);
		color: #fbbf24;
		border: 1px solid rgba(245, 158, 11, 0.3);
	}

	.amount-val {
		font-weight: 700;
		color: #ffffff;
		font-size: 0.9375rem;
	}

	/* Status Pills */
	.status-cell {
		display: flex;
		align-items: center;
		gap: 0.4rem;
		position: relative;
	}

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

	.status-badge.draft { background: rgba(148, 163, 184, 0.15); color: #cbd5e1; border: 1px solid rgba(148, 163, 184, 0.25); }
	.status-badge.draft .status-dot { background: #94a3b8; }

	.status-badge.sent { background: rgba(56, 189, 248, 0.15); color: #38bdf8; border: 1px solid rgba(56, 189, 248, 0.3); }
	.status-badge.sent .status-dot { background: #38bdf8; }

	.status-badge.accepted { background: rgba(16, 185, 129, 0.15); color: #34d399; border: 1px solid rgba(16, 185, 129, 0.3); }
	.status-badge.accepted .status-dot { background: #10b981; }

	.status-badge.converted { background: rgba(168, 85, 247, 0.15); color: #c084fc; border: 1px solid rgba(168, 85, 247, 0.3); }
	.status-badge.converted .status-dot { background: #a855f7; }

	.status-badge.rejected { background: rgba(244, 63, 94, 0.15); color: #fb7185; border: 1px solid rgba(244, 63, 94, 0.3); }
	.status-badge.rejected .status-dot { background: #f43f5e; }

	.status-select {
		background: rgba(255, 255, 255, 0.05);
		border: 1px solid rgba(255, 255, 255, 0.1);
		color: #cbd5e1;
		font-size: 0.65rem;
		padding: 0.2rem 0.3rem;
		border-radius: 4px;
		cursor: pointer;
	}

	.actions-group {
		display: flex;
		align-items: center;
		justify-content: flex-end;
		gap: 0.5rem;
	}

	.btn-emerald {
		background: #10b981;
		color: #ffffff;
		border: 1px solid #059669;
		box-shadow: 0 2px 8px rgba(16, 185, 129, 0.3);
	}
	.btn-emerald:hover {
		background: #059669;
	}

	.converted-tag {
		font-size: 0.725rem;
		color: #a855f7;
		font-weight: 700;
		background: rgba(168, 85, 247, 0.12);
		padding: 0.25rem 0.55rem;
		border-radius: 6px;
		border: 1px solid rgba(168, 85, 247, 0.25);
	}

	/* State Cells */
	.state-cell {
		text-align: center;
		padding: 3.5rem 1rem !important;
	}

	.loading-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.75rem;
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

	.empty-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.5rem;
		color: #94a3b8;
	}

	.empty-icon {
		font-size: 2.5rem;
		opacity: 0.7;
	}

	.empty-state h3 {
		color: #ffffff;
		margin: 0;
		font-size: 1.125rem;
	}

	.empty-state p {
		color: #64748b;
		font-size: 0.8125rem;
		max-width: 380px;
		margin: 0;
	}

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
		max-width: 580px;
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
		color: #ffffff;
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
		transition: all 0.15s ease;
	}
	.drawer-close-btn:hover {
		background: rgba(255, 255, 255, 0.12);
		color: #ffffff;
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
	.drawer-status-banner.sent { background: rgba(56, 189, 248, 0.12); border: 1px solid rgba(56, 189, 248, 0.3); color: #38bdf8; }
	.drawer-status-banner.accepted { background: rgba(16, 185, 129, 0.12); border: 1px solid rgba(16, 185, 129, 0.3); color: #34d399; }
	.drawer-status-banner.converted { background: rgba(168, 85, 247, 0.12); border: 1px solid rgba(168, 85, 247, 0.3); color: #c084fc; }
	.drawer-status-banner.draft { background: rgba(148, 163, 184, 0.12); border: 1px solid rgba(148, 163, 184, 0.25); color: #cbd5e1; }
	.drawer-status-banner.rejected { background: rgba(244, 63, 94, 0.12); border: 1px solid rgba(244, 63, 94, 0.3); color: #fb7185; }

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

	.drawer-notes-box {
		background: rgba(15, 23, 42, 0.4);
		border: 1px dashed rgba(255, 255, 255, 0.12);
		border-radius: 8px;
		padding: 0.875rem;
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
		.quotes-header { flex-direction: column; align-items: flex-start; }
		.quotes-header-actions { width: 100%; justify-content: space-between; }
		.kpi-grid { grid-template-columns: 1fr 1fr; }
		.drawer-panel { max-width: 100%; }
	}
</style>
