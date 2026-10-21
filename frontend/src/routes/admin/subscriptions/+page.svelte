<script lang="ts">
	import { onMount } from 'svelte';
	import { api, type SubscriptionPlan } from '$lib/api';

	let plans = $state<SubscriptionPlan[]>([]);
	let loading = $state(true);
	let error = $state('');

	onMount(async () => {
		await loadPlans();
	});

	async function loadPlans() {
		loading = true;
		error = '';
		try {
			plans = await api.listSubscriptionPlans();
		} catch (err: any) {
			error = err.message || 'Failed to load subscription plans';
		} finally {
			loading = false;
		}
	}

	function formatCurrency(val: string | number) {
		const n = typeof val === 'string' ? parseFloat(val) : val;
		if (isNaN(n) || n === 0) return 'Free';
		return '₹' + n.toLocaleString('en-IN', { maximumFractionDigits: 0 });
	}

	function getPlanGradient(code: string) {
		switch ((code || '').toUpperCase()) {
			case 'FREE':
				return 'linear-gradient(135deg, rgba(71, 85, 105, 0.2) 0%, rgba(30, 41, 59, 0.4) 100%)';
			case 'STARTER':
				return 'linear-gradient(135deg, rgba(14, 165, 233, 0.15) 0%, rgba(30, 41, 59, 0.4) 100%)';
			case 'BUSINESS':
				return 'linear-gradient(135deg, rgba(99, 102, 241, 0.2) 0%, rgba(30, 41, 59, 0.5) 100%)';
			case 'ENTERPRISE':
				return 'linear-gradient(135deg, rgba(245, 158, 11, 0.2) 0%, rgba(30, 41, 59, 0.6) 100%)';
			default:
				return 'linear-gradient(135deg, rgba(30, 41, 59, 0.4) 0%, rgba(15, 23, 42, 0.6) 100%)';
		}
	}

	function getPlanAccent(code: string) {
		switch ((code || '').toUpperCase()) {
			case 'FREE':
				return '#94a3b8';
			case 'STARTER':
				return '#38bdf8';
			case 'BUSINESS':
				return '#818cf8';
			case 'ENTERPRISE':
				return '#fbbf24';
			default:
				return '#38bdf8';
		}
	}
</script>

<svelte:head>
	<title>Subscription Plans & Tiers - Super Admin</title>
</svelte:head>

<div style="display: flex; flex-direction: column; gap: 2rem;">
	<!-- Page Header -->
	<div style="display: flex; justify-content: space-between; align-items: flex-start; flex-wrap: wrap; gap: 1rem;">
		<div>
			<div style="display: flex; gap: 1rem; align-items: center; margin-bottom: 0.25rem;">
				<a href="/admin" style="font-size: 0.8125rem; color: #818cf8; text-decoration: none;">
					← Back to Admin Console
				</a>
				<span style="color: var(--border);">|</span>
				<a href="/admin/tenants" style="font-size: 0.8125rem; color: #38bdf8; text-decoration: none;">
					Company Tenants Master →
				</a>
			</div>
			<h1 style="font-size: 1.75rem;">Subscription Plans & Tier Catalog</h1>
			<p style="font-size: 0.875rem; color: var(--text-secondary);">
				Configure global platform subscription tiers, user thresholds, invoice generation quotas, and feature entitlements.
			</p>
		</div>
		<div style="display: flex; gap: 0.75rem;">
			<a href="/admin/tenants" class="btn btn-secondary">
				Manage Tenant Subscriptions
			</a>
			<button onclick={loadPlans} class="btn btn-primary" disabled={loading}>
				{loading ? 'Refreshing...' : '🔄 Refresh Plans'}
			</button>
		</div>
	</div>

	{#if error}
		<div style="background: rgba(244, 63, 94, 0.15); border: 1px solid rgba(244, 63, 94, 0.3); border-radius: var(--radius-md); padding: 1rem; color: #fb7185;">
			⚠️ {error}
		</div>
	{/if}

	<!-- Tier Cards Grid -->
	<div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(260px, 1fr)); gap: 1.5rem;">
		{#if loading}
			<div class="card text-center" style="grid-column: 1 / -1; padding: 3rem; color: var(--text-muted);">
				Loading subscription tiers catalog...
			</div>
		{:else}
			{#each plans as p}
				{@const accent = getPlanAccent(p.code)}
				<div
					class="card"
					style="background: {getPlanGradient(p.code)}; border: 1px solid {accent}40; border-radius: var(--radius-lg); padding: 1.75rem; display: flex; flex-direction: column; justify-content: space-between; position: relative; overflow: hidden; box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.3);"
				>
					{#if p.code === 'BUSINESS'}
						<div style="position: absolute; top: 12px; right: 12px; background: #6366f1; color: #fff; font-size: 0.65rem; font-weight: 700; text-transform: uppercase; letter-spacing: 0.05em; padding: 0.2rem 0.6rem; border-radius: 9999px;">
							Most Popular
						</div>
					{/if}

					<div>
						<div style="display: flex; align-items: center; gap: 0.5rem; margin-bottom: 0.5rem;">
							<span style="font-size: 1.25rem;">
								{p.code === 'FREE' ? '🌱' : p.code === 'STARTER' ? '🚀' : p.code === 'BUSINESS' ? '💼' : '👑'}
							</span>
							<h3 style="font-size: 1.25rem; font-weight: 700; color: {accent}; margin: 0;">{p.name}</h3>
						</div>
						<p style="font-size: 0.8125rem; color: var(--text-secondary); min-height: 2.5rem; margin-bottom: 1.25rem;">
							{p.description}
						</p>

						<div style="margin-bottom: 1.5rem; padding-bottom: 1.25rem; border-bottom: 1px solid rgba(255, 255, 255, 0.08);">
							<div style="display: flex; align-items: baseline; gap: 0.35rem;">
								<span style="font-size: 2rem; font-weight: 800; color: #fff;">{formatCurrency(p.monthly_price)}</span>
								<span style="font-size: 0.8125rem; color: var(--text-muted);">/ month</span>
							</div>
							<div style="font-size: 0.75rem; color: var(--text-muted); margin-top: 0.25rem;">
								or {formatCurrency(p.yearly_price)} billed annually (Save 15%)
							</div>
						</div>

						<div style="display: flex; flex-direction: column; gap: 0.75rem; font-size: 0.8125rem;">
							<div style="display: flex; align-items: center; gap: 0.5rem;">
								<span style="color: #34d399;">✓</span>
								<span>
									<strong>{p.max_users > 1000 ? 'Unlimited' : p.max_users}</strong> Team User{p.max_users === 1 ? '' : 's'}
								</span>
							</div>

							<div style="display: flex; align-items: center; gap: 0.5rem;">
								<span style="color: #34d399;">✓</span>
								<span>
									<strong>{p.max_invoices_per_month > 100000 ? 'Unlimited' : p.max_invoices_per_month.toLocaleString('en-IN')}</strong> Invoices / month
								</span>
							</div>

							<div style="display: flex; align-items: center; gap: 0.5rem;">
								<span style="color: #34d399;">✓</span>
								<span>Customer Credit & Debit Ledger</span>
							</div>

							<div style="display: flex; align-items: center; gap: 0.5rem;">
								<span style="color: #34d399;">✓</span>
								<span>Direct WhatsApp Web Sharing & Email</span>
							</div>

							{#if p.code === 'BUSINESS' || p.code === 'ENTERPRISE' || p.code === 'PRO'}
								<div style="display: flex; align-items: center; gap: 0.5rem;">
									<span style="color: #818cf8;">✓</span>
									<span style="font-weight: 600; color: #a5b4fc;">Audited Double-Entry Accounting</span>
								</div>

								<div style="display: flex; align-items: center; gap: 0.5rem;">
									<span style="color: #818cf8;">✓</span>
									<span style="font-weight: 600; color: #a5b4fc;">Expense Management & Receipt Auditing</span>
								</div>
							{:else}
								<div style="display: flex; align-items: center; gap: 0.5rem; color: var(--text-muted); font-size: 0.75rem;">
									<span>🔒</span>
									<span>Double-Entry Accounting (Next Plan only)</span>
								</div>
								<div style="display: flex; align-items: center; gap: 0.5rem; color: var(--text-muted); font-size: 0.75rem;">
									<span>🔒</span>
									<span>Expense Management (Next Plan only)</span>
								</div>
							{/if}

							{#if p.code !== 'FREE' && p.code !== 'BASIC'}
								<div style="display: flex; align-items: center; gap: 0.5rem;">
									<span style="color: #34d399;">✓</span>
									<span>10 Locked Master Invoice Print Templates</span>
								</div>
							{/if}

							{#if p.code === 'ENTERPRISE'}
								<div style="display: flex; align-items: center; gap: 0.5rem;">
									<span style="color: #fbbf24;">★</span>
									<span style="color: #fbbf24; font-weight: 600;">Dedicated Account Rep & Custom Quotas</span>
								</div>
							{/if}
						</div>
					</div>

					<div style="margin-top: 1.75rem; padding-top: 1rem; border-top: 1px solid rgba(255, 255, 255, 0.08);">
						<a
							href="/admin/tenants"
							class="btn btn-secondary"
							style="width: 100%; text-align: center; justify-content: center; font-size: 0.8125rem;"
						>
							Assign to Tenants →
						</a>
					</div>
				</div>
			{/each}
		{/if}
	</div>

	<!-- Master Catalog Specifications Table -->
	<div class="card" style="padding: 1.5rem; display: flex; flex-direction: column; gap: 1rem;">
		<div style="display: flex; justify-content: space-between; align-items: center;">
			<div>
				<h3 style="font-size: 1.125rem; margin: 0;">Catalog Quota Specifications</h3>
				<p style="font-size: 0.8125rem; color: var(--text-secondary); margin-top: 0.25rem;">
					System-enforced maximum thresholds per billing period across active tenant instances.
				</p>
			</div>
		</div>

		<div class="table-container" style="overflow-x: auto;">
			<table class="data-table">
				<thead>
					<tr>
						<th>Plan Code</th>
						<th>Plan Name</th>
						<th>Monthly Rate</th>
						<th>Annual Rate</th>
						<th>Included Users</th>
						<th>Monthly Invoices Quota</th>
						<th>Status</th>
					</tr>
				</thead>
				<tbody>
					{#each plans as p}
						<tr>
							<td style="font-family: var(--font-mono); font-weight: 700; color: #818cf8;">{p.code}</td>
							<td style="font-weight: 600;">{p.name}</td>
							<td>{formatCurrency(p.monthly_price)}</td>
							<td>{formatCurrency(p.yearly_price)}</td>
							<td>{p.max_users > 1000 ? 'Unlimited' : p.max_users}</td>
							<td>{p.max_invoices_per_month > 100000 ? 'Unlimited' : p.max_invoices_per_month.toLocaleString('en-IN')}</td>
							<td>
								<span class="badge" style="background: rgba(16, 185, 129, 0.15); color: #34d399;">
									ACTIVE
								</span>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	</div>
</div>
