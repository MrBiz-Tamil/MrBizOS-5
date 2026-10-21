<script lang="ts">
	import '../lib/styles/theme.css';
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { api, setSelectedTenantId, type User, type Tenant, type TenantSubscriptionInfo } from '$lib/api';
	import { realtimeBilling } from '$lib/stores/realtime.svelte';
	import UnitSwitcher from '$lib/components/UnitSwitcher.svelte';

	let { children } = $props();
	let currentUser = $state<User | null>(null);
	let currentTenant = $state<Tenant | null>(null);
	let tenantList = $state<Tenant[]>([]);
	let subInfo = $state<TenantSubscriptionInfo | null>(null);
	let loading = $state(true);
	let currentPath = $derived($page.url.pathname);
	// Pages shown without the app shell (signed-out flows)
	// ?print=1 is the server-side PDF renderer: just the page, no app shell, no banners or dialogs
	let isBarePage = $derived(['/login', '/forgot-password', '/reset-password', '/verify-email', '/terms', '/privacy', '/signup'].includes(currentPath) || currentPath.startsWith('/i/') || $page.url.searchParams.get('print') === '1');

	let dynamicTenantTheme = $derived(
		currentTenant?.template_options?.primary_color
			? `--brand-primary: ${currentTenant.template_options.primary_color};`
			: ''
	);

	// Upgrade Modal
	let showUpgradeModal = $state(false);
	let billingStatus = $state('');
	let trialDaysLeft = $state<number | null>(null);
	let legalRequired = $state(false);
	let legalVersion = $state('');
	let legalBusy = $state(false);
	let legalError = $state('');
	let upgradeFeatureTitle = $state('');
	let upgradeFeatureDesc = $state('');

	let isNextPlan = $derived(
		currentUser?.is_super_admin ||
		subInfo?.plan_code === 'BUSINESS' ||
		subInfo?.plan_code === 'PRO' ||
		subInfo?.plan_code === 'ENTERPRISE' ||
		subInfo?.features?.accounting === true
	);

	onMount(async () => {
		try {
			if (isBarePage) {
				loading = false;
				return;
			}
			const data = await api.getMe();
			currentUser = data.user;
			currentTenant = data.tenant || null;

			if (currentUser?.is_super_admin) {
				try {
					tenantList = await api.listTenants();
					if (!currentTenant && tenantList.length > 0) {
						currentTenant = tenantList[0];
						setSelectedTenantId(tenantList[0].id);
					}
				} catch (e) {}
			}

			// Persist active tenant branding for login memory and visual white-label experience
			if (currentTenant) {
				localStorage.setItem('tenant_brand_name', currentTenant.name || '');
				localStorage.setItem('tenant_brand_legal', currentTenant.legal_name || '');
				if (currentTenant.logo_url) {
					localStorage.setItem('tenant_brand_logo', currentTenant.logo_url);
				}
				// Connect real-time SSE stream for tenant
				realtimeBilling.connect(currentTenant.id);
			}

			// Billing banner (trial / past due / suspended)
			try {
				const b = await api.getBilling();
				billingStatus = b.status;
				trialDaysLeft = b.trial_days_left ?? null;
			} catch (e) {}

			// Terms / privacy acceptance gate
			try {
				const ls = await api.getLegalStatus();
				legalVersion = ls.required_version;
				legalRequired = !ls.accepted;
			} catch (e) {}

			// Load subscription telemetry
			try {
				subInfo = await api.getCompanySubscription();
			} catch (e) {}
		} catch (e) {
			if (!isBarePage) {
				goto('/login');
			}
		} finally {
			loading = false;
		}
	});

	function getInitials(name: string | undefined) {
		if (!name) return 'SB';
		const parts = name.trim().split(/\s+/);
		if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
		return (parts[0][0] + parts[1][0]).toUpperCase();
	}

	function handleSwitchTenant(e: Event) {
		const select = e.target as HTMLSelectElement;
		const id = select.value;
		setSelectedTenantId(id);
		window.location.reload();
	}

	async function handleLogout() {
		try {
			await api.logout();
		} catch (e) {}
		currentUser = null;
		currentTenant = null;
		setSelectedTenantId(null);
		goto('/login');
	}

	async function acceptLegal() {
		legalBusy = true;
		legalError = '';
		try {
			await api.acceptLegal(legalVersion);
			legalRequired = false;
		} catch (e: any) {
			legalError = e.message || 'Could not record acceptance';
		} finally {
			legalBusy = false;
		}
	}

	function handleLockedFeature(e: MouseEvent, title: string, desc: string) {
		if (!isNextPlan) {
			e.preventDefault();
			upgradeFeatureTitle = title;
			upgradeFeatureDesc = desc;
			showUpgradeModal = true;
		}
	}
</script>

{#if loading}
	<div style="display: flex; height: 100vh; align-items: center; justify-content: center; background: #0b0f19;">
		<div style="display: flex; flex-direction: column; align-items: center; gap: 1rem;">
			<div style="width: 40px; height: 40px; border: 3px solid rgba(99, 102, 241, 0.2); border-top-color: #6366f1; border-radius: 50%; animation: spin 0.8s linear infinite;"></div>
			<span style="font-size: 0.875rem; color: #94a3b8; font-weight: 500;">Loading secure workspace...</span>
		</div>
	</div>
{:else if isBarePage}
	{@render children?.()}
{:else}
	{#if legalRequired}
		<div style="position: fixed; inset: 0; z-index: 10000; background: rgba(0,0,0,0.75); display: flex; align-items: center; justify-content: center; padding: 1rem;">
			<div class="card" style="max-width: 460px; padding: 1.75rem;">
				<h2 style="margin-bottom: 0.5rem;">Terms and Privacy Policy</h2>
				<p style="color: var(--text-secondary); font-size: 0.875rem; line-height: 1.5;">
					To continue, please review and accept our <a href="/terms" target="_blank" rel="noopener">Terms of Service</a>
					and <a href="/privacy" target="_blank" rel="noopener">Privacy Policy</a>.
				</p>
				{#if legalError}<p style="color: #fb7185; font-size: 0.8125rem;">{legalError}</p>{/if}
				<div style="display: flex; gap: 0.75rem; margin-top: 1.25rem;">
					<button class="btn btn-primary" onclick={acceptLegal} disabled={legalBusy}>I accept</button>
					<button class="btn btn-secondary" onclick={handleLogout}>Sign out</button>
				</div>
			</div>
		</div>
	{/if}
	<div class="layout-container" style={dynamicTenantTheme}>
		<!-- Sidebar Navigation -->
		<aside class="sidebar">
			<!-- Workspace Branding & Plan Status -->
			<div style="padding: 1.25rem; border-bottom: 1px solid var(--border-subtle); display: flex; align-items: center; gap: 0.75rem;">
				{#if currentTenant?.logo_url}
					<img
						src={currentTenant.logo_url}
						alt={currentTenant.name}
						style="width: 40px; height: 40px; border-radius: 10px; object-fit: contain; background: rgba(255, 255, 255, 0.05); border: 1px solid rgba(255, 255, 255, 0.1); padding: 2px;"
					/>
				{:else}
					<div style="width: 40px; height: 40px; border-radius: 10px; background: linear-gradient(135deg, #4f46e5 0%, #06b6d4 100%); display: flex; align-items: center; justify-content: center; font-family: var(--font-heading); font-weight: 800; color: white; font-size: 1.15rem; box-shadow: 0 4px 14px rgba(99, 102, 241, 0.45); flex-shrink: 0;">
						{getInitials(currentTenant?.name || (currentUser?.is_super_admin ? 'SaaS' : 'Workspace'))}
					</div>
				{/if}
				<div style="display: flex; flex-direction: column; min-width: 0; flex: 1;">
					<span style="font-family: var(--font-heading); font-weight: 800; font-size: 0.9375rem; color: #fff; text-overflow: ellipsis; overflow: hidden; white-space: nowrap;">
						{currentTenant?.name || (currentUser?.is_super_admin ? 'SaaS Platform Admin' : 'Private Workspace')}
					</span>
					
					{#if currentUser?.is_super_admin}
						<span style="font-size: 0.7rem; color: #f59e0b; font-weight: 600; display: flex; align-items: center; gap: 0.25rem;">
							🛡️ Super Admin Scope
						</span>
					{:else if currentTenant?.legal_name}
						<span style="font-size: 0.6875rem; color: var(--text-muted); text-overflow: ellipsis; overflow: hidden; white-space: nowrap;">
							{currentTenant.legal_name}
						</span>
					{:else if isNextPlan}
						<span style="font-size: 0.7rem; color: #818cf8; font-weight: 700; display: flex; align-items: center; gap: 0.25rem;">
							💎 Next Plan (Business)
						</span>
					{:else}
						<span style="font-size: 0.7rem; color: #34d399; font-weight: 600; display: flex; align-items: center; gap: 0.25rem;">
							🌱 Basic Plan (Active)
						</span>
					{/if}
				</div>
			</div>

			<!-- Multi-Unit Switcher -->
			<UnitSwitcher />

			<!-- Navigation Links -->
			<nav style="padding: 0.5rem 0.75rem 1.25rem 0.75rem; display: flex; flex-direction: column; gap: 0.25rem; flex: 1; overflow-y: auto;">
				<div style="font-size: 0.6875rem; font-weight: 700; color: #38bdf8; text-transform: uppercase; letter-spacing: 0.05em; padding: 0.25rem 0.5rem 0.25rem 0.5rem;">
					Enterprise ERP Suite (Odoo/SAP Killer)
				</div>

				<a href="/procurement" class="nav-item" class:active={currentPath.startsWith('/procurement')}>
					<span class="nav-icon">📦</span> Procurement & 3-Way Match
				</a>
				<a href="/accounting/reconciliation" class="nav-item" class:active={currentPath.startsWith('/accounting/reconciliation')}>
					<span class="nav-icon">🏦</span> Bank Auto-Reconciliation
				</a>
				<a href="/hrms" class="nav-item" class:active={currentPath.startsWith('/hrms')}>
					<span class="nav-icon">👥</span> HRMS & 1-Click Payroll
				</a>
				<a href="/workflow" class="nav-item" class:active={currentPath.startsWith('/workflow')}>
					<span class="nav-icon">🤖</span> AI Autonomous Missions
				</a>
				<a href="/healthcare" class="nav-item" class:active={currentPath.startsWith('/healthcare')}>
					<span class="nav-icon">🏥</span> Smart Healthcare Hub
				</a>
				<a href="/travel" class="nav-item" class:active={currentPath.startsWith('/travel')}>
					<span class="nav-icon">✈️</span> Travel & Visa OS
				</a>
				<a href="/realestate" class="nav-item" class:active={currentPath.startsWith('/realestate')}>
					<span class="nav-icon">🏢</span> Real Estate Deals
				</a>

				<div style="font-size: 0.6875rem; font-weight: 700; color: var(--text-muted); text-transform: uppercase; letter-spacing: 0.05em; padding: 0.75rem 0.5rem 0.25rem 0.5rem;">
					Core Invoicing & Billing
				</div>

				<a href="/" class="nav-item" class:active={currentPath === '/'}>
					<span class="nav-icon">📊</span> Dashboard
				</a>

				<a href="/pos" class="nav-item" class:active={currentPath.startsWith('/pos')} style="color: #38bdf8; font-weight: 700; background: rgba(56, 189, 248, 0.08); border-left: 3px solid #38bdf8;">
					<span class="nav-icon">⚡</span> Retail POS Terminal
				</a>

				<a href="/invoices" class="nav-item" class:active={currentPath.startsWith('/invoices')}>
					<span class="nav-icon">📄</span> Sales Invoices
				</a>

				<a href="/quotations" class="nav-item" class:active={currentPath.startsWith('/quotations')}>
					<span class="nav-icon">📝</span> Quotations
				</a>

				<a href="/trade/shipments" class="nav-item" class:active={currentPath.startsWith('/trade/shipments')}>
					<span class="nav-icon">🚢</span> Export / Import Shipments
				</a>
				<a href="/trade/export-invoice/new" class="nav-item" class:active={currentPath.startsWith('/trade/export-invoice')}>
					<span class="nav-icon">🌍</span> New Export Invoice
				</a>
				<a href="/trade/settings" class="nav-item" class:active={currentPath === '/trade/settings'}>
					<span class="nav-icon">💱</span> Trade Settings & Rates
				</a>

				<a href="/invoices/recurring" class="nav-item" class:active={currentPath === '/invoices/recurring'}>
					<span class="nav-icon">🔁</span> Recurring Invoices
				</a>

				<a href="/purchases" class="nav-item" class:active={currentPath.startsWith('/purchases')}>
					<span class="nav-icon">📥</span> Purchase Bills
				</a>

				<a href="/credit-notes" class="nav-item" class:active={currentPath.startsWith('/credit-notes')}>
					<span class="nav-icon">🔄</span> Credit & Debit Notes
				</a>

				<a href="/payments" class="nav-item" class:active={currentPath === '/payments'}>
					<span class="nav-icon">💳</span> Payments Received
				</a>

				<a href="/customers" class="nav-item" class:active={currentPath === '/customers'}>
					<span class="nav-icon">👥</span> Customers & Balances
				</a>

				<a href="/suppliers" class="nav-item" class:active={currentPath.startsWith('/suppliers')}>
					<span class="nav-icon">🏭</span> Suppliers & Khata
				</a>

				<a href="/products" class="nav-item" class:active={currentPath === '/products'}>
					<span class="nav-icon">📦</span> Products & Pricing
				</a>

				<a href="/inventory" class="nav-item" class:active={currentPath === '/inventory'}>
					<span class="nav-icon">📋</span> Inventory Stock
				</a>

				<a href="/units" class="nav-item" class:active={currentPath === '/units'}>
					<span class="nav-icon">⚖️</span> Units & Converter
				</a>

				<!-- Next Plan Advanced Features -->
				<div style="font-size: 0.6875rem; font-weight: 700; color: #818cf8; text-transform: uppercase; letter-spacing: 0.05em; padding: 1rem 0.5rem 0.5rem 0.5rem; display: flex; justify-content: space-between; align-items: center;">
					<span>Financial Suite</span>
					{#if !isNextPlan}
						<span style="font-size: 0.625rem; background: rgba(129, 140, 248, 0.15); color: #a5b4fc; padding: 0.1rem 0.4rem; border-radius: 4px; font-weight: 600;">Next Plan</span>
					{/if}
				</div>

				<a
					href="/accounting"
					class="nav-item"
					class:active={currentPath.startsWith('/accounting')}
					onclick={(e) => handleLockedFeature(e, 'Audited Financial Accounting', 'Full Double-Entry Accounting, General Ledger, Chart of Accounts, Trial Balance, and Profit & Loss statement.')}
				>
					<span class="nav-icon">📖</span> Audited Accounting
					{#if !isNextPlan}
						<span style="margin-left: auto; font-size: 0.75rem;">🔒</span>
					{/if}
				</a>

				<a
					href="/expenses"
					class="nav-item"
					class:active={currentPath.startsWith('/expenses')}
					onclick={(e) => handleLockedFeature(e, 'Company Expense Management', 'Track and categorize company expenses, record vendor receipts, manage payment modes, and automate financial expense postings.')}
				>
					<span class="nav-icon">💸</span> Expense Management
					{#if !isNextPlan}
						<span style="margin-left: auto; font-size: 0.75rem;">🔒</span>
					{/if}
				</a>

				<div style="font-size: 0.6875rem; font-weight: 700; color: var(--text-muted); text-transform: uppercase; letter-spacing: 0.05em; padding: 1rem 0.5rem 0.5rem 0.5rem;">
					Administration
				</div>

				<a href="/settings" class="nav-item" class:active={currentPath === '/settings'}>
					<span class="nav-icon">⚙️</span> Company Profile
				</a>

				<a href="/settings/team" class="nav-item" class:active={currentPath === '/settings/team'}>
					<span class="nav-icon">👥</span> Team & Staff Users
				</a>

				<a href="/settings/security" class="nav-item" class:active={currentPath === "/settings/security"}>
					<span class="nav-icon">🔐</span> Account Security
				</a>

				<a href="/settings/billing" class="nav-item" class:active={currentPath === "/settings/billing"}>
					<span class="nav-icon">💳</span> Plan & Billing
				</a>

				<a href="/accounting/exchange" class="nav-item" class:active={currentPath === "/accounting/exchange"}>
					<span class="nav-icon">🔄</span> Returns & Data Exchange
				</a>

				<a href="/settings/templates" class="nav-item" class:active={currentPath === '/settings/templates'}>
					<span class="nav-icon">🎨</span> Billing Templates
				</a>

				<a href="/settings/templates/studio" class="nav-item" class:active={currentPath === '/settings/templates/studio'} style="color: #f472b6; font-weight: 600;">
					<span class="nav-icon">🛠️</span> Bill Template Studio
				</a>

				<a href="/settings/developer" class="nav-item" class:active={currentPath === '/settings/developer'} style="color: #818cf8; font-weight: 600;">
					<span class="nav-icon">⚡</span> Developer & API Keys
				</a>

				{#if currentUser?.is_super_admin}
					<div style="font-size: 0.6875rem; font-weight: 700; color: #f59e0b; text-transform: uppercase; letter-spacing: 0.05em; padding: 1rem 0.5rem 0.5rem 0.5rem;">
						Super Admin Controls
					</div>

					<a href="/admin" class="nav-item" class:active={currentPath === '/admin'}>
						<span class="nav-icon">🌐</span> Platform Health
					</a>

					<a href="/admin/tenants" class="nav-item" class:active={currentPath === '/admin/tenants'}>
						<span class="nav-icon">🏢</span> Companies Master
					</a>

					<a href="/admin/billing" class="nav-item" class:active={currentPath === "/admin/billing"}>
						<span class="nav-icon">🧾</span> Plan Invoices
					</a>

					<a href="/admin/subscriptions" class="nav-item" class:active={currentPath === '/admin/subscriptions'}>
						<span class="nav-icon">💳</span> Subscription Plans
					</a>

					<a href="/admin/audit" class="nav-item" class:active={currentPath === '/admin/audit'}>
						<span class="nav-icon">🛡️</span> Security Audit Logs
					</a>
				{/if}
			</nav>

			<!-- Bottom User Card -->
			<div style="padding: 1rem; border-top: 1px solid var(--border-subtle); display: flex; align-items: center; justify-content: space-between; background: rgba(0,0,0,0.2);">
				<div style="display: flex; flex-direction: column; min-width: 0;">
					<span style="font-size: 0.8125rem; font-weight: 600; color: var(--text-primary); text-overflow: ellipsis; overflow: hidden; white-space: nowrap;">
						{currentUser?.full_name || currentUser?.email}
					</span>
					<span style="font-size: 0.7rem; color: var(--text-muted);">
						{currentUser?.role}
					</span>
				</div>
				<button onclick={handleLogout} title="Sign Out" class="btn btn-secondary btn-sm" style="padding: 0.35rem 0.6rem;">
					🚪
				</button>
			</div>
		</aside>

		<!-- Main Workspace Area -->
		<div class="main-content">
			<!-- Topbar -->
			<header class="topbar">
				<div style="display: flex; align-items: center; gap: 1rem; flex-wrap: wrap;">
					{#if currentUser?.is_super_admin && tenantList.length > 0}
						<div style="display: flex; align-items: center; gap: 0.5rem;">
							<span style="font-size: 0.75rem; font-weight: 600; color: var(--text-muted); text-transform: uppercase;">Company Scope:</span>
							<select
								class="form-select"
								style="width: auto; padding: 0.35rem 0.75rem; font-size: 0.8125rem; border-color: rgba(99, 102, 241, 0.4); background: rgba(17, 24, 39, 0.9); font-weight: 500;"
								value={currentTenant?.id}
								onchange={handleSwitchTenant}
							>
								{#each tenantList as t}
									<option value={t.id}>{t.name} ({t.gstin})</option>
								{/each}
							</select>
						</div>
					{:else}
						<div style="display: flex; align-items: center; gap: 0.75rem; flex-wrap: wrap;">
							<div style="display: flex; align-items: center; gap: 0.4rem; font-weight: 700; color: #fff; font-size: 0.9375rem;">
								<span>🏢</span> {currentTenant?.name || 'Private Enterprise'}
							</div>
							{#if currentTenant?.gstin}
								<span style="font-size: 0.75rem; font-family: var(--font-mono); color: #818cf8; background: rgba(99, 102, 241, 0.15); padding: 0.15rem 0.55rem; border-radius: 4px; border: 1px solid rgba(99, 102, 241, 0.25);">
									GSTIN: {currentTenant.gstin}
								</span>
							{/if}
							<span style="font-size: 0.75rem; font-weight: 700; background: {isNextPlan ? 'rgba(99, 102, 241, 0.15)' : 'rgba(16, 185, 129, 0.15)'}; color: {isNextPlan ? '#a5b4fc' : '#34d399'}; padding: 0.15rem 0.6rem; border-radius: 9999px; border: 1px solid {isNextPlan ? 'rgba(99, 102, 241, 0.3)' : 'rgba(16, 185, 129, 0.3)'};">
								{isNextPlan ? '💎 Next Plan' : '🌱 Basic Plan'}
							</span>
						</div>
					{/if}
				</div>

				<div style="display: flex; align-items: center; gap: 1rem;">
					{#if !isNextPlan && !currentUser?.is_super_admin}
						<button
							onclick={() => {
								upgradeFeatureTitle = 'Next Plan (Business Tier)';
								upgradeFeatureDesc = 'Unlock Audited Double-Entry Accounting, automated Expense Management, higher invoice quotas, and multi-user collaboration.';
								showUpgradeModal = true;
							}}
							class="btn btn-secondary btn-sm"
							style="border-color: rgba(129, 140, 248, 0.4); color: #a5b4fc;"
						>
							✨ Upgrade to Next Plan
						</button>
					{/if}
					<a href="/invoices/new" class="btn btn-primary btn-sm">
						+ Create Invoice
					</a>
				</div>
			</header>

			<!-- Page Body -->
			<main class="page-body animate-fade-in">
				{#if billingStatus === 'TRIAL' || billingStatus === 'PAST_DUE' || billingStatus === 'SUSPENDED'}
					<div style="margin-bottom: 1rem; padding: 0.65rem 1rem; border-radius: 8px; font-size: 0.875rem; background: {billingStatus === 'SUSPENDED' ? 'rgba(244,63,94,0.15)' : 'rgba(251,191,36,0.12)'}; color: {billingStatus === 'SUSPENDED' ? '#fb7185' : '#fbbf24'};">
						{#if billingStatus === 'TRIAL'}Free trial: {trialDaysLeft ?? 0} day(s) left.
						{:else if billingStatus === 'PAST_DUE'}A plan invoice is overdue.
						{:else}Your account is paused: new invoices are blocked.{/if}
						<a href="/settings/billing" style="margin-left: 0.5rem; text-decoration: underline;">Plan & Billing</a>
					</div>
				{/if}
				{@render children?.()}
			</main>
		</div>
	</div>
{/if}

<!-- Upgrade Modal for Gated Next Plan Features -->
{#if showUpgradeModal}
	<div class="modal-backdrop">
		<div class="modal-card animate-fade-in" style="max-width: 520px; border: 1px solid rgba(129, 140, 248, 0.4); box-shadow: 0 20px 40px rgba(0, 0, 0, 0.6);">
			<div class="modal-header" style="border-bottom: 1px solid rgba(255, 255, 255, 0.08);">
				<div style="display: flex; align-items: center; gap: 0.6rem;">
					<span style="font-size: 1.5rem;">🔒</span>
					<div>
						<h3 style="font-size: 1.125rem; margin: 0; color: #fff;">Feature Locked: {upgradeFeatureTitle}</h3>
						<span style="font-size: 0.75rem; color: #a5b4fc; font-weight: 600;">Requires Next Plan (Business Tier)</span>
					</div>
				</div>
				<button onclick={() => (showUpgradeModal = false)} style="background: none; border: none; color: var(--text-muted); cursor: pointer; font-size: 1.25rem;">✕</button>
			</div>

			<div class="modal-body" style="display: flex; flex-direction: column; gap: 1.25rem; padding: 1.5rem;">
				<p style="font-size: 0.875rem; color: var(--text-secondary); line-height: 1.5; margin: 0;">
					{upgradeFeatureDesc}
				</p>

				<div style="background: rgba(15, 23, 42, 0.6); border: 1px solid rgba(129, 140, 248, 0.2); border-radius: var(--radius-md); padding: 1rem; display: flex; flex-direction: column; gap: 0.6rem;">
					<div style="font-size: 0.8125rem; font-weight: 700; color: #38bdf8;">
						⭐ Included in Next Plan:
					</div>
					<div style="font-size: 0.8125rem; color: var(--text-secondary); display: flex; flex-direction: column; gap: 0.4rem;">
						<div style="display: flex; align-items: center; gap: 0.5rem;">
							<span style="color: #34d399;">✓</span> Full Double-Entry Accounting (Chart of Accounts & Journal Entries)
						</div>
						<div style="display: flex; align-items: center; gap: 0.5rem;">
							<span style="color: #34d399;">✓</span> General Ledger, Trial Balance, and Profit & Loss Statement
						</div>
						<div style="display: flex; align-items: center; gap: 0.5rem;">
							<span style="color: #34d399;">✓</span> Expense Management, Category Tracking & Vendor Receipt Uploads
						</div>
						<div style="display: flex; align-items: center; gap: 0.5rem;">
							<span style="color: #34d399;">✓</span> Up to 2,500 invoices/month and 20 team users
						</div>
					</div>
				</div>

				<div style="font-size: 0.8125rem; color: var(--text-muted);">
					💡 <em>Your current Basic Plan gives you full access to Sales Invoicing, Payments, Products, Inventory, and Customer Credit/Debit Ledgers.</em>
				</div>
			</div>

			<div class="modal-footer" style="display: flex; justify-content: space-between; align-items: center; border-top: 1px solid rgba(255, 255, 255, 0.08); padding: 1rem 1.5rem;">
				{#if currentUser?.is_super_admin}
					<a href="/admin/tenants" class="btn btn-primary btn-sm" onclick={() => (showUpgradeModal = false)}>
						Activate in Companies Master →
					</a>
				{:else}
					<span style="font-size: 0.75rem; color: var(--text-muted);">Contact platform administrator to activate</span>
				{/if}
				<button type="button" onclick={() => (showUpgradeModal = false)} class="btn btn-secondary btn-sm">
					Close
				</button>
			</div>
		</div>
	</div>
{/if}

<style>
	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	.nav-item {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		padding: 0.625rem 0.875rem;
		border-radius: var(--radius-md);
		color: var(--text-secondary);
		font-size: 0.875rem;
		font-weight: 500;
		text-decoration: none;
		transition: all 0.15s ease;
	}

	.nav-item:hover {
		background: var(--bg-card-hover);
		color: var(--text-primary);
	}

	.nav-item.active {
		background: var(--primary-light);
		color: #a5b4fc;
		border: 1px solid rgba(99, 102, 241, 0.25);
		font-weight: 600;
	}

	.nav-icon {
		font-size: 1rem;
		display: inline-block;
		width: 1.25rem;
		text-align: center;
	}
</style>
