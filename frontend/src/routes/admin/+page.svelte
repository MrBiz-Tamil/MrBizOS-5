<script lang="ts">
	import { onMount } from 'svelte';
	import { api, type PlatformOverview } from '$lib/api';

	let overview = $state<PlatformOverview | null>(null);
	let loading = $state(true);
	let error = $state('');

	onMount(async () => {
		try {
			overview = await api.getPlatformOverview();
		} catch (err: any) {
			error = err.message || 'Failed to load platform metrics';
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
	<title>Super Admin Platform Overview - SaaS Billing</title>
</svelte:head>

<div style="display: flex; flex-direction: column; gap: 2rem;">
	<div style="display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 1rem;">
		<div>
			<div style="display: flex; align-items: center; gap: 0.5rem; margin-bottom: 0.25rem;">
				<span class="badge" style="background: rgba(245, 158, 11, 0.2); color: #fbbf24;">Super Admin Console</span>
			</div>
			<h1 style="font-size: 1.75rem;">Platform Health & Overview</h1>
			<p style="font-size: 0.875rem; color: var(--text-secondary);">
				System metrics, multi-tenant database status, and aggregated platform transaction volume.
			</p>
		</div>

		<div style="display: flex; gap: 0.75rem;">
			<a href="/admin/tenants" class="btn btn-primary">
				+ Provision New Company
			</a>
			<a href="/admin/audit" class="btn btn-secondary">
				View Security Audit Logs
			</a>
		</div>
	</div>

	{#if loading}
		<div style="padding: 3rem; text-align: center; color: var(--text-muted);">
			Loading platform metrics...
		</div>
	{:else if error}
		<div class="card" style="border-color: rgba(244, 63, 94, 0.4); color: #fb7185;">
			⚠️ {error}
		</div>
	{:else if overview}
		<div class="metrics-grid">
			<div class="metric-card">
				<div class="metric-title">
					<span>Total Tenant Companies</span>
					<span>🏢</span>
				</div>
				<div class="metric-value text-indigo">
					{overview.total_tenants}
				</div>
				<div class="metric-subtitle">
					{overview.active_tenants} active, {overview.suspended_tenants} suspended
				</div>
			</div>

			<div class="metric-card">
				<div class="metric-title">
					<span>Platform Gross Volume</span>
					<span>💎</span>
				</div>
				<div class="metric-value text-emerald">
					{formatCurrency(overview.total_volume)}
				</div>
				<div class="metric-subtitle">
					{overview.total_invoices} invoices processed
				</div>
			</div>

			<div class="metric-card">
				<div class="metric-title">
					<span>Total Platform Users</span>
					<span>👥</span>
				</div>
				<div class="metric-value">
					{overview.total_users}
				</div>
				<div class="metric-subtitle">
					Across all company tenants
				</div>
			</div>

			<div class="metric-card">
				<div class="metric-title">
					<span>Database Health</span>
					<span>🟢</span>
				</div>
				<div class="metric-value" style="color: #34d399; font-size: 1.35rem;">
					{overview.database_status}
				</div>
				<div class="metric-subtitle">
					PostgreSQL 16 Low-RAM Connection Pool
				</div>
			</div>
		</div>

		<!-- Infrastructure Status Card -->
		<div class="card" style="padding: 1.5rem;">
			<h3 style="font-size: 1.125rem; margin-bottom: 1rem;">Target Infrastructure Profile (&le; ₹1,000/year VPS)</h3>
			<div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 1rem;">
				<div style="background: rgba(255,255,255,0.03); padding: 1rem; border-radius: var(--radius-md);">
					<span style="font-size: 0.75rem; color: var(--text-muted); display: block;">Go Modular Monolith</span>
					<strong style="color: #34d399; font-size: 1rem;">&lt; 25 MB RAM</strong>
					<p style="font-size: 0.75rem; color: var(--text-secondary); margin-top: 0.25rem;">Single binary with pgxpool</p>
				</div>

				<div style="background: rgba(255,255,255,0.03); padding: 1rem; border-radius: var(--radius-md);">
					<span style="font-size: 0.75rem; color: var(--text-muted); display: block;">PostgreSQL 16 Engine</span>
					<strong style="color: #34d399; font-size: 1rem;">~60 MB RAM</strong>
					<p style="font-size: 0.75rem; color: var(--text-secondary); margin-top: 0.25rem;">Tuned max 10 connections</p>
				</div>

				<div style="background: rgba(255,255,255,0.03); padding: 1rem; border-radius: var(--radius-md);">
					<span style="font-size: 0.75rem; color: var(--text-muted); display: block;">SvelteKit BFF Server</span>
					<strong style="color: #34d399; font-size: 1rem;">~35 MB RAM</strong>
					<p style="font-size: 0.75rem; color: var(--text-secondary); margin-top: 0.25rem;">Node SSR with HTTP proxy</p>
				</div>

				<div style="background: rgba(255,255,255,0.03); padding: 1rem; border-radius: var(--radius-md);">
					<span style="font-size: 0.75rem; color: var(--text-muted); display: block;">Estimated Total Consumption</span>
					<strong style="color: #818cf8; font-size: 1rem;">~120 MB RAM</strong>
					<p style="font-size: 0.75rem; color: var(--text-secondary); margin-top: 0.25rem;">Leaves 70%+ free on 512MB VPS</p>
				</div>
			</div>
		</div>
	{/if}
</div>
