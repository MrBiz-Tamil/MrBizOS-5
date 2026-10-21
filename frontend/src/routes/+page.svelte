<script lang="ts">
	import { onMount } from 'svelte';
	import { api, type DashboardMetrics, type Tenant, type User } from '$lib/api';
	import LiveLedgerStream from '$lib/components/LiveLedgerStream.svelte';

	let metrics = $state<DashboardMetrics | null>(null);
	let company = $state<Tenant | null>(null);
	let user = $state<User | null>(null);
	let loading = $state(true);
	let error = $state('');

	onMount(async () => {
		try {
			const me = await api.getMe();
			user = me.user;
			company = me.tenant || null;
			if (company?.id || user.tenant_id) {
				metrics = await api.getDashboard();
			} else if (user.is_super_admin) {
				try {
					const tList = await api.listTenants();
					if (tList.length > 0) {
						if (!company) company = tList[0];
						metrics = await api.getDashboard();
					}
				} catch (e) {}
			}
		} catch (err: any) {
			error = err.message || 'Failed to load dashboard';
		} finally {
			loading = false;
		}
	});

	function formatCurrency(val: string | number | undefined): string {
		if (!val) return '₹0.00';
		const num = typeof val === 'string' ? parseFloat(val) : val;
		return new Intl.NumberFormat('en-IN', { style: 'currency', currency: 'INR' }).format(num);
	}
</script>

<svelte:head>
	<title>{company?.name ? `${company.name} - Command Center` : 'Dashboard - SaaS Billing & Accounting'}</title>
</svelte:head>

<div style="display: flex; flex-direction: column; gap: 2rem;">
	
	<!-- Personalized Tenant Enterprise Command Center -->
	<div class="card" style="padding: 1.5rem 1.75rem; background: linear-gradient(135deg, rgba(30, 41, 59, 0.7) 0%, rgba(17, 24, 39, 0.9) 100%); border: 1px solid rgba(99, 102, 241, 0.25); display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 1.25rem;">
		<div style="display: flex; align-items: center; gap: 1rem;">
			{#if company?.logo_url}
				<img src={company.logo_url} alt={company.name} style="width: 52px; height: 52px; border-radius: 12px; object-fit: contain; background: rgba(255,255,255,0.05); padding: 4px; border: 1px solid rgba(255,255,255,0.1);" />
			{:else}
				<div style="width: 52px; height: 52px; border-radius: 12px; background: linear-gradient(135deg, #4f46e5 0%, #06b6d4 100%); display: flex; align-items: center; justify-content: center; font-size: 1.35rem; font-weight: 800; color: #fff; box-shadow: 0 4px 15px rgba(99, 102, 241, 0.45); flex-shrink: 0;">
					{company ? company.name.slice(0, 2).toUpperCase() : 'SB'}
				</div>
			{/if}

			<div>
				<div style="display: flex; align-items: center; gap: 0.5rem; flex-wrap: wrap;">
					<h1 style="font-size: 1.5rem; font-weight: 800; margin: 0; color: #fff;">
						{company?.name || 'SaaS Platform Operations'}
					</h1>
					<span style="font-size: 0.75rem; font-weight: 700; background: rgba(16, 185, 129, 0.15); color: #34d399; padding: 0.15rem 0.6rem; border-radius: 9999px; border: 1px solid rgba(16, 185, 129, 0.3);">
						🏢 Active Organization
					</span>
				</div>
				<p style="font-size: 0.8125rem; color: var(--text-secondary); margin: 0.25rem 0 0 0;">
					{#if company}
						<strong style="color: #cbd5e1;">{company.legal_name || company.name}</strong> • GSTIN: <span style="font-family: var(--font-mono); color: #818cf8;">{company.gstin}</span> • Invoice Series: <span style="font-family: var(--font-mono); color: #fbbf24;">{company.invoice_prefix}</span>
					{:else}
						Platform Super Administrator Command Overview
					{/if}
				</p>
			</div>
		</div>

		<div style="display: flex; gap: 0.75rem; flex-wrap: wrap;">
			<a href="/invoices/new" class="btn btn-primary btn-sm" style="padding: 0.5rem 1rem; font-weight: 600;">
				+ New Tax Invoice
			</a>
			<a href="/customers" class="btn btn-secondary btn-sm" style="padding: 0.5rem 1rem;">
				👥 Customer Balances
			</a>
			<a href="/settings" class="btn btn-secondary btn-sm" style="padding: 0.5rem 0.75rem;" title="Manage Company Profile & Branding">
				⚙️ Profile
			</a>
		</div>
	</div>

	{#if loading}
		<div style="padding: 3rem; text-align: center; color: var(--text-muted);">
			Loading financial metrics...
		</div>
	{:else if error}
		<div class="card" style="border-color: rgba(244, 63, 94, 0.3); color: #fb7185;">
			⚠️ {error}
		</div>
	{:else if metrics}
		<!-- Financial Indicator Cards -->
		<div class="metrics-grid">
			<div class="metric-card">
				<div class="metric-title">
					<span>Total Invoiced Sales</span>
					<span style="font-size: 1.1rem;">📈</span>
				</div>
				<div class="metric-value text-indigo">
					{formatCurrency(metrics.total_sales)}
				</div>
				<div class="metric-subtitle">
					{metrics.total_invoices_count} total generated invoices
				</div>
			</div>

			<div class="metric-card">
				<div class="metric-title">
					<span>Outstanding Debtors</span>
					<span style="font-size: 1.1rem;">⏳</span>
				</div>
				<div class="metric-value text-amber">
					{formatCurrency(metrics.outstanding_balance)}
				</div>
				<div class="metric-subtitle">
					{metrics.unpaid_invoices_count} unpaid or partial invoices
				</div>
			</div>

			<div class="metric-card">
				<div class="metric-title">
					<span>Received Payments</span>
					<span style="font-size: 1.1rem;">💰</span>
				</div>
				<div class="metric-value text-emerald">
					{formatCurrency(metrics.total_received)}
				</div>
				<div class="metric-subtitle">
					{metrics.paid_invoices_count} fully settled invoices
				</div>
			</div>

			<div class="metric-card">
				<div class="metric-title">
					<span>Inventory Valuation</span>
					<span style="font-size: 1.1rem;">📦</span>
				</div>
				<div class="metric-value">
					{formatCurrency(metrics.total_inventory_value)}
				</div>
				<div class="metric-subtitle" class:text-rose={metrics.low_stock_count > 0}>
					{metrics.low_stock_count} low-stock alerts
				</div>
			</div>
		</div>

		<!-- Tables Section -->
		<div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(500px, 1fr)); gap: 1.5rem;">
			<!-- Recent Invoices -->
			<div class="card" style="padding: 1.25rem;">
				<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem;">
					<h3 style="font-size: 1.125rem;">Recent Invoices</h3>
					<a href="/invoices" style="font-size: 0.8125rem; color: #818cf8; text-decoration: none;">View All →</a>
				</div>

				<div class="table-container">
					<table class="data-table">
						<thead>
							<tr>
								<th>Invoice</th>
								<th>Customer</th>
								<th>Date</th>
								<th class="text-right">Total</th>
								<th>Status</th>
							</tr>
						</thead>
						<tbody>
							{#if !metrics.recent_invoices || metrics.recent_invoices.length === 0}
								<tr>
									<td colspan="5" class="text-center" style="padding: 2rem; color: var(--text-muted);">
										No invoices recorded yet. Click "+ New Invoice" to create one.
									</td>
								</tr>
							{:else}
								{#each metrics.recent_invoices as inv}
									<tr>
										<td>
											<a href={`/invoices/${inv.id}`} style="color: #a5b4fc; font-weight: 600; text-decoration: none; font-family: var(--font-mono);">
												{inv.invoice_number}
											</a>
										</td>
										<td style="font-weight: 500;">{inv.customer_name}</td>
										<td style="color: var(--text-secondary);">{new Date(inv.invoice_date).toLocaleDateString('en-IN')}</td>
										<td class="text-right font-mono" style="font-weight: 600;">{formatCurrency(inv.grand_total)}</td>
										<td>
											<span class={`badge badge-${inv.status.toLowerCase().replace('_', '-')}`}>
												<span class="badge-dot"></span>
												{inv.status}
											</span>
										</td>
									</tr>
								{/each}
							{/if}
						</tbody>
					</table>
				</div>
			</div>

			<!-- Recent Payments -->
			<div class="card" style="padding: 1.25rem;">
				<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem;">
					<h3 style="font-size: 1.125rem;">Recent Receipts</h3>
					<a href="/payments" style="font-size: 0.8125rem; color: #818cf8; text-decoration: none;">View All →</a>
				</div>

				<div class="table-container">
					<table class="data-table">
						<thead>
							<tr>
								<th>Receipt #</th>
								<th>Date</th>
								<th>Method</th>
								<th class="text-right">Amount</th>
							</tr>
						</thead>
						<tbody>
							{#if !metrics.recent_payments || metrics.recent_payments.length === 0}
								<tr>
									<td colspan="4" class="text-center" style="padding: 2rem; color: var(--text-muted);">
										No payments recorded yet.
									</td>
								</tr>
							{:else}
								{#each metrics.recent_payments as pay}
									<tr>
										<td style="font-family: var(--font-mono); color: #34d399; font-weight: 600;">
											{pay.payment_number}
										</td>
										<td style="color: var(--text-secondary);">{new Date(pay.payment_date).toLocaleDateString('en-IN')}</td>
										<td>
											<span style="font-size: 0.75rem; background: rgba(255,255,255,0.06); padding: 0.2rem 0.5rem; border-radius: var(--radius-sm);">
												{pay.payment_method}
											</span>
										</td>
										<td class="text-right font-mono text-emerald" style="font-weight: 600;">
											{formatCurrency(pay.amount)}
										</td>
									</tr>
								{/each}
							{/if}
						</tbody>
					</table>
				</div>
			</div>
		</div>

		<!-- Real-time Double-Entry Financial Stream (SSE) -->
		<LiveLedgerStream />
	{:else if user?.is_super_admin}
		<div class="card" style="padding: 2.5rem; text-align: center; background: radial-gradient(circle, rgba(99, 102, 241, 0.15) 0%, rgba(17, 24, 39, 0.9) 70%);">
			<div style="font-size: 2.5rem; margin-bottom: 1rem;">🌐</div>
			<h2 style="font-size: 1.5rem; margin-bottom: 0.75rem;">Super Admin Management Portal</h2>
			<p style="color: var(--text-secondary); max-width: 600px; margin: 0 auto 1.5rem auto; font-size: 0.9375rem;">
				You are logged in with system-level administrator privileges. Provision a company tenant to begin issuing invoices and recording payments, or monitor platform health and security audit trails.
			</p>
			<div style="display: flex; justify-content: center; gap: 1rem; flex-wrap: wrap;">
				<a href="/admin/tenants" class="btn btn-primary">+ Provision First Company</a>
				<a href="/admin" class="btn btn-secondary">Platform Health & Metrics</a>
				<a href="/admin/audit" class="btn btn-secondary">System Audit Logs</a>
			</div>
		</div>
	{/if}
</div>
