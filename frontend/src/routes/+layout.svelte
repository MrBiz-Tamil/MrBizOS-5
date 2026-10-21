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

	// Adaptive Workspaces (Role-Based Progressive Disclosure)
	type WorkspacePreset = 'FULL' | 'COUNTER' | 'GROWTH' | 'OPERATIONS';
	let activeWorkspacePreset = $state<WorkspacePreset>('FULL');
	let billingOnlyMode = $derived(activeWorkspacePreset === 'COUNTER');

	function setWorkspacePreset(preset: WorkspacePreset) {
		activeWorkspacePreset = preset;
		if (typeof localStorage !== 'undefined') {
			localStorage.setItem('mrbiz_workspace_preset', preset);
			localStorage.setItem('mrbiz_billing_only_mode', String(preset === 'COUNTER'));
		}
	}

	function toggleBillingMode() {
		setWorkspacePreset(activeWorkspacePreset === 'COUNTER' ? 'FULL' : 'COUNTER');
	}

	// Universal Command Palette State (Ctrl+K / Cmd+K)
	let showCommandPalette = $state(false);
	let commandSearch = $state('');
	let selectedCommandIndex = $state(0);

	interface CommandItem {
		id: string;
		title: string;
		subtitle: string;
		icon: string;
		category: string;
		href?: string;
		action?: () => void;
	}

	const commandItems: CommandItem[] = [
		// Quick Actions
		{ id: 'act-bill', title: 'New Express Bill', subtitle: 'Open 3-second quick billing counter with UPI QR', icon: '⚡', category: 'Actions', href: '/billing' },
		{ id: 'act-pos', title: 'Retail POS Terminal', subtitle: 'Barcode scanner checkout screen', icon: '🛒', category: 'Actions', href: '/pos' },
		{ id: 'act-inv', title: 'Create Tax Invoice', subtitle: 'Full GST sales invoice with logistics', icon: '📄', category: 'Actions', href: '/invoices/new' },
		{ id: 'act-call', title: 'AI Voice BPO Dial', subtitle: 'Outbound campaign & live audio screen pop', icon: '🎙️', category: 'Actions', href: '/bpo-voice' },
		{ id: 'act-inbox', title: 'Omnichannel Social Inbox', subtitle: 'Unified WhatsApp, LinkedIn, IG, FB messages', icon: '💬', category: 'Actions', href: '/inbox' },
		{ id: 'act-rec', title: 'Bank Auto-Reconciliation', subtitle: 'Drag & drop bank statement CSV match', icon: '🏦', category: 'Actions', href: '/accounting/reconciliation' },
		{ id: 'act-pay', title: 'HRMS 1-Click Payroll', subtitle: 'Process monthly salaries and timesheets', icon: '👥', category: 'Actions', href: '/hrms' },

		// Workspace Presets
		{ id: 'ws-counter', title: 'Switch Workspace: Express Counter & POS', subtitle: 'Clean, distraction-free retail counter', icon: '⚡', category: 'Workspaces', action: () => setWorkspacePreset('COUNTER') },
		{ id: 'ws-growth', title: 'Switch Workspace: Growth & Social Sales', subtitle: 'Focused on WhatsApp/IG inbox, voice, and deals', icon: '💬', category: 'Workspaces', action: () => setWorkspacePreset('GROWTH') },
		{ id: 'ws-ops', title: 'Switch Workspace: Supply Chain & Ops', subtitle: '3-way procurement, inventory, and trade shipments', icon: '📦', category: 'Workspaces', action: () => setWorkspacePreset('OPERATIONS') },
		{ id: 'ws-full', title: 'Switch Workspace: Full Enterprise ERP', subtitle: 'All 16 business departments and multi-units', icon: '🌐', category: 'Workspaces', action: () => setWorkspacePreset('FULL') },

		// Industry Verticals
		{ id: 'dom-health', title: 'Smart Healthcare Hub', subtitle: 'Patients, appointments, and EMR prescriptions', icon: '🏥', category: 'Verticals', href: '/healthcare' },
		{ id: 'dom-travel', title: 'Travel & Visa OS', subtitle: 'Tour packages, flight locks, and itinerary PDF', icon: '✈️', category: 'Verticals', href: '/travel' },
		{ id: 'dom-estate', title: 'Real Estate Deals', subtitle: 'Property listings, site visits, and escrow', icon: '🏢', category: 'Verticals', href: '/realestate' },
		{ id: 'dom-ecom', title: 'E-Commerce & Print-on-Demand', subtitle: 'Etsy orders, blank reservations, and lab routing', icon: '🛍️', category: 'Verticals', href: '/ecommerce' },
		{ id: 'dom-consult', title: 'Software Consulting & SOW', subtitle: 'Client contracts, timesheets, and milestone billing', icon: '💻', category: 'Verticals', href: '/consulting' },
		{ id: 'dom-trade', title: 'Export & Import Trade OS', subtitle: 'Commercial invoices, packing lists, and customs', icon: '🚢', category: 'Verticals', href: '/trade/shipments' },

		// Financials & Core
		{ id: 'fin-acc', title: 'Audited Financial Accounting', subtitle: 'General ledger, trial balance, and P&L', icon: '📖', category: 'Financials', href: '/accounting' },
		{ id: 'fin-quote', title: 'Quotations & Estimates', subtitle: 'Draft proposals and convert to bills', icon: '📝', category: 'Financials', href: '/quotations' },
		{ id: 'fin-cust', title: 'Customers & Balances', subtitle: 'Client ledger, credit limits, and aging khata', icon: '👥', category: 'Financials', href: '/customers' },
		{ id: 'fin-supp', title: 'Suppliers & Vendors', subtitle: 'Purchase accounts and vendor khata', icon: '🏭', category: 'Financials', href: '/suppliers' },
		{ id: 'fin-prod', title: 'Products & Pricing Catalog', subtitle: 'HSN/SAC codes, tax slabs, and price lists', icon: '📦', category: 'Financials', href: '/products' },
		{ id: 'fin-inv', title: 'Inventory Stock & Batches', subtitle: 'Warehouse bins, low-stock alerts, and transfers', icon: '📋', category: 'Financials', href: '/inventory' },
		{ id: 'fin-tmpl', title: '17 Billing Templates Gallery', subtitle: 'Configure print layouts, colors, and UPI QR', icon: '🎨', category: 'Settings', href: '/settings/templates' },
		{ id: 'fin-set', title: 'Company Profile & Settings', subtitle: 'GSTIN, bank details, and legal agreements', icon: '⚙️', category: 'Settings', href: '/settings' },
	];

	let filteredCommands = $derived(
		commandItems.filter((i) =>
			i.title.toLowerCase().includes(commandSearch.toLowerCase()) ||
			i.subtitle.toLowerCase().includes(commandSearch.toLowerCase()) ||
			i.category.toLowerCase().includes(commandSearch.toLowerCase())
		)
	);

	function handleGlobalKeydown(e: KeyboardEvent) {
		if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
			e.preventDefault();
			showCommandPalette = !showCommandPalette;
			if (showCommandPalette) {
				commandSearch = '';
				selectedCommandIndex = 0;
			}
		} else if (e.key === 'Escape' && showCommandPalette) {
			showCommandPalette = false;
		} else if (showCommandPalette) {
			if (e.key === 'ArrowDown') {
				e.preventDefault();
				selectedCommandIndex = (selectedCommandIndex + 1) % (filteredCommands.length || 1);
			} else if (e.key === 'ArrowUp') {
				e.preventDefault();
				selectedCommandIndex = (selectedCommandIndex - 1 + (filteredCommands.length || 1)) % (filteredCommands.length || 1);
			} else if (e.key === 'Enter') {
				e.preventDefault();
				if (filteredCommands[selectedCommandIndex]) {
					executeCommand(filteredCommands[selectedCommandIndex]);
				}
			}
		}
	}

	function executeCommand(cmd: CommandItem) {
		showCommandPalette = false;
		if (cmd.action) {
			cmd.action();
		} else if (cmd.href) {
			goto(cmd.href);
		}
	}

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
			if (typeof localStorage !== 'undefined') {
				const savedPreset = localStorage.getItem('mrbiz_workspace_preset') as WorkspacePreset;
				if (savedPreset) {
					activeWorkspacePreset = savedPreset;
				} else if (localStorage.getItem('mrbiz_billing_only_mode') === 'true') {
					activeWorkspacePreset = 'COUNTER';
				}
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

<svelte:window onkeydown={handleGlobalKeydown} />

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
				<!-- Workspace Mode Selector Card -->
				<div style="background: rgba(15, 23, 42, 0.7); border: 1px solid rgba(255, 255, 255, 0.08); border-radius: 8px; padding: 0.5rem; margin-bottom: 0.75rem;">
					<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.35rem;">
						<span style="font-size: 0.65rem; font-weight: 700; color: #94a3b8; text-transform: uppercase; letter-spacing: 0.05em;">Workspace View</span>
						<button
							type="button"
							onclick={() => { showCommandPalette = true; commandSearch = ''; selectedCommandIndex = 0; }}
							style="background: none; border: none; font-size: 0.65rem; color: #818cf8; cursor: pointer; font-family: var(--font-mono); display: flex; align-items: center; gap: 0.2rem;"
							title="Command Palette"
						>
							🔍 <span>⌘K</span>
						</button>
					</div>
					<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 0.25rem;">
						<button
							type="button"
							class="ws-btn"
							class:active={activeWorkspacePreset === 'COUNTER'}
							onclick={() => setWorkspacePreset('COUNTER')}
							title="Express Counter & Retail POS"
						>
							⚡ Counter
						</button>
						<button
							type="button"
							class="ws-btn"
							class:active={activeWorkspacePreset === 'GROWTH'}
							onclick={() => setWorkspacePreset('GROWTH')}
							title="Growth & Social Sales"
						>
							💬 Growth
						</button>
						<button
							type="button"
							class="ws-btn"
							class:active={activeWorkspacePreset === 'OPERATIONS'}
							onclick={() => setWorkspacePreset('OPERATIONS')}
							title="Supply Chain & Ops"
						>
							📦 Ops
						</button>
						<button
							type="button"
							class="ws-btn"
							class:active={activeWorkspacePreset === 'FULL'}
							onclick={() => setWorkspacePreset('FULL')}
							title="Full Enterprise ERP"
						>
							🌐 Full ERP
						</button>
					</div>
				</div>

				{#if activeWorkspacePreset === 'COUNTER'}
					<!-- ============================================== -->
					<!-- PRESET 1: EXPRESS COUNTER & RETAIL POS         -->
					<!-- ============================================== -->
					<div style="font-size: 0.6875rem; font-weight: 700; color: #34d399; text-transform: uppercase; letter-spacing: 0.05em; padding: 0.25rem 0.5rem;">
						Express Retail Counter
					</div>

					<a href="/billing" class="nav-item" class:active={currentPath.startsWith('/billing')} style="color: #38bdf8; font-weight: 700; background: rgba(56, 189, 248, 0.12); border-left: 3px solid #38bdf8;">
						<span class="nav-icon">⚡</span> Express Billing Counter
					</a>

					<a href="/" class="nav-item" class:active={currentPath === '/'}>
						<span class="nav-icon">📊</span> Quick Dashboard
					</a>

					<a href="/pos" class="nav-item" class:active={currentPath.startsWith('/pos')}>
						<span class="nav-icon">🛒</span> Retail POS Terminal
					</a>

					<a href="/invoices" class="nav-item" class:active={currentPath.startsWith('/invoices')}>
						<span class="nav-icon">📄</span> Sales Invoices
					</a>

					<a href="/quotations" class="nav-item" class:active={currentPath.startsWith('/quotations')}>
						<span class="nav-icon">📝</span> Quotations & Estimates
					</a>

					<a href="/payments" class="nav-item" class:active={currentPath === '/payments'}>
						<span class="nav-icon">💳</span> Payments Received
					</a>

					<a href="/customers" class="nav-item" class:active={currentPath === '/customers'}>
						<span class="nav-icon">👥</span> Customers & Balances
					</a>

					<a href="/products" class="nav-item" class:active={currentPath === '/products'}>
						<span class="nav-icon">📦</span> Products & Pricing
					</a>

					<a href="/inventory" class="nav-item" class:active={currentPath === '/inventory'}>
						<span class="nav-icon">📋</span> Inventory Stock
					</a>

					<a href="/settings/templates" class="nav-item" class:active={currentPath === '/settings/templates'}>
						<span class="nav-icon">🎨</span> 17 Billing Templates
					</a>

					<a href="/settings" class="nav-item" class:active={currentPath === '/settings'}>
						<span class="nav-icon">⚙️</span> Company Profile & QR
					</a>

				{:else if activeWorkspacePreset === 'GROWTH'}
					<!-- ============================================== -->
					<!-- PRESET 2: GROWTH & OMNICHANNEL SOCIAL SALES     -->
					<!-- ============================================== -->
					<div style="font-size: 0.6875rem; font-weight: 700; color: #f43f5e; text-transform: uppercase; letter-spacing: 0.05em; padding: 0.25rem 0.5rem;">
						Growth & Omnichannel Sales
					</div>

					<a href="/inbox" class="nav-item" class:active={currentPath.startsWith('/inbox')} style="color: #f43f5e; font-weight: 700; background: rgba(244, 63, 94, 0.08); border-left: 3px solid #f43f5e;">
						<span class="nav-icon">💬</span> Omnichannel Social Inbox
					</a>

					<a href="/bpo-voice" class="nav-item" class:active={currentPath.startsWith('/bpo-voice')} style="color: #06b6d4; font-weight: 700; background: rgba(6, 182, 212, 0.08); border-left: 3px solid #06b6d4;">
						<span class="nav-icon">🎙️</span> AI Voice Agent BPO Desk
					</a>

					<a href="/support" class="nav-item" class:active={currentPath.startsWith('/support')}>
						<span class="nav-icon">🎫</span> Support Helpdesk & SLAs
					</a>

					<a href="/journeys" class="nav-item" class:active={currentPath.startsWith('/journeys')}>
						<span class="nav-icon">🛤️</span> Journey Lifecycle Engine
					</a>

					<a href="/projects" class="nav-item" class:active={currentPath.startsWith('/projects')}>
						<span class="nav-icon">📁</span> Projects & UTMS Tasks
					</a>

					<a href="/realestate" class="nav-item" class:active={currentPath.startsWith('/realestate')}>
						<span class="nav-icon">🏢</span> Real Estate Deals
					</a>

					<a href="/travel" class="nav-item" class:active={currentPath.startsWith('/travel')}>
						<span class="nav-icon">✈️</span> Travel & Visa OS
					</a>

					<a href="/healthcare" class="nav-item" class:active={currentPath.startsWith('/healthcare')}>
						<span class="nav-icon">🏥</span> Smart Healthcare Hub
					</a>

					<a href="/billing" class="nav-item" class:active={currentPath.startsWith('/billing')}>
						<span class="nav-icon">⚡</span> Quick Billing Counter
					</a>

					<a href="/customers" class="nav-item" class:active={currentPath === '/customers'}>
						<span class="nav-icon">👥</span> Customers & CRM
					</a>

				{:else if activeWorkspacePreset === 'OPERATIONS'}
					<!-- ============================================== -->
					<!-- PRESET 3: SUPPLY CHAIN, OPERATIONS & TRADE     -->
					<!-- ============================================== -->
					<div style="font-size: 0.6875rem; font-weight: 700; color: #a855f7; text-transform: uppercase; letter-spacing: 0.05em; padding: 0.25rem 0.5rem;">
						Supply Chain & Operations
					</div>

					<a href="/procurement" class="nav-item" class:active={currentPath.startsWith('/procurement')} style="color: #a855f7; font-weight: 700; background: rgba(168, 85, 247, 0.08); border-left: 3px solid #a855f7;">
						<span class="nav-icon">📦</span> Procurement & 3-Way Match
					</a>

					<a href="/inventory" class="nav-item" class:active={currentPath === '/inventory'}>
						<span class="nav-icon">📋</span> Inventory Stock & Batches
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

					<a href="/purchases" class="nav-item" class:active={currentPath.startsWith('/purchases')}>
						<span class="nav-icon">📥</span> Purchase Bills
					</a>

					<a href="/products" class="nav-item" class:active={currentPath === '/products'}>
						<span class="nav-icon">📦</span> Products & Pricing
					</a>

					<a href="/units" class="nav-item" class:active={currentPath === '/units'}>
						<span class="nav-icon">⚖️</span> Units & Converter
					</a>

					<a href="/accounting/reconciliation" class="nav-item" class:active={currentPath.startsWith('/accounting/reconciliation')}>
						<span class="nav-icon">🏦</span> Bank Auto-Reconciliation
					</a>

					<a href="/accounting" class="nav-item" class:active={currentPath.startsWith('/accounting')}>
						<span class="nav-icon">📖</span> Audited Accounting
					</a>

					<a href="/suppliers" class="nav-item" class:active={currentPath.startsWith('/suppliers')}>
						<span class="nav-icon">🏭</span> Suppliers & Khata
					</a>

				{:else}
					<!-- ============================================== -->
					<!-- PRESET 4: FULL UNIFIED ENTERPRISE ERP          -->
					<!-- ============================================== -->
					<div style="font-size: 0.6875rem; font-weight: 700; color: #38bdf8; text-transform: uppercase; letter-spacing: 0.05em; padding: 0.25rem 0.5rem;">
						Enterprise ERP Suite
					</div>

					<a href="/inbox" class="nav-item" class:active={currentPath.startsWith('/inbox')} style="color: #f43f5e; font-weight: 700; background: rgba(244, 63, 94, 0.08); border-left: 3px solid #f43f5e;">
						<span class="nav-icon">💬</span> Omnichannel Social Inbox
					</a>
					<a href="/bpo-voice" class="nav-item" class:active={currentPath.startsWith('/bpo-voice')} style="color: #06b6d4; font-weight: 700; background: rgba(6, 182, 212, 0.08); border-left: 3px solid #06b6d4;">
						<span class="nav-icon">🎙️</span> AI Voice Agent BPO Desk
					</a>
					<a href="/support" class="nav-item" class:active={currentPath.startsWith('/support')}>
						<span class="nav-icon">🎫</span> Support Helpdesk & SLAs
					</a>
					<a href="/projects" class="nav-item" class:active={currentPath.startsWith('/projects')}>
						<span class="nav-icon">📁</span> Projects & UTMS Tasks
					</a>
					<a href="/journeys" class="nav-item" class:active={currentPath.startsWith('/journeys')}>
						<span class="nav-icon">🛤️</span> Journey Lifecycle Engine
					</a>
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

					<a href="/billing" class="nav-item" class:active={currentPath.startsWith('/billing')} style="color: #38bdf8; font-weight: 700; background: rgba(56, 189, 248, 0.08); border-left: 3px solid #38bdf8;">
						<span class="nav-icon">⚡</span> Express Billing Counter
					</a>

					<a href="/" class="nav-item" class:active={currentPath === '/'}>
						<span class="nav-icon">📊</span> Dashboard
					</a>

					<a href="/pos" class="nav-item" class:active={currentPath.startsWith('/pos')}>
						<span class="nav-icon">🛒</span> Retail POS Terminal
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
				{/if}

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

				<!-- Center: Universal Command Palette (Ctrl+K) Trigger & Quick Workspace Switcher -->
				<div style="display: flex; align-items: center; gap: 0.75rem; flex: 1; justify-content: center; max-width: 620px;">
					<button
						type="button"
						onclick={() => { showCommandPalette = true; commandSearch = ''; selectedCommandIndex = 0; }}
						class="cmd-search-trigger"
						title="Quick Search & Navigation (Ctrl+K)"
					>
						<span style="opacity: 0.7; font-size: 0.85rem;">🔍</span>
						<span class="cmd-search-text">Search anything or run action...</span>
						<kbd class="cmd-kbd">Ctrl K</kbd>
					</button>

					<div class="topbar-workspace-pills">
						<button
							type="button"
							class="topbar-pill"
							class:active={activeWorkspacePreset === 'COUNTER'}
							onclick={() => setWorkspacePreset('COUNTER')}
							title="Express Retail Counter & POS"
						>
							⚡ Counter
						</button>
						<button
							type="button"
							class="topbar-pill"
							class:active={activeWorkspacePreset === 'GROWTH'}
							onclick={() => setWorkspacePreset('GROWTH')}
							title="Omnichannel Social Sales & AI Voice"
						>
							💬 Growth
						</button>
						<button
							type="button"
							class="topbar-pill"
							class:active={activeWorkspacePreset === 'OPERATIONS'}
							onclick={() => setWorkspacePreset('OPERATIONS')}
							title="Supply Chain, 3-Way Match & Trade"
						>
							📦 Ops
						</button>
						<button
							type="button"
							class="topbar-pill"
							class:active={activeWorkspacePreset === 'FULL'}
							onclick={() => setWorkspacePreset('FULL')}
							title="Full Enterprise ERP Suite"
						>
							🌐 Full
						</button>
					</div>
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

<!-- ============================================== -->
<!-- UNIVERSAL COMMAND PALETTE MODAL (Ctrl+K / Cmd+K)-->
<!-- ============================================== -->
{#if showCommandPalette}
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div class="cmd-palette-backdrop" onclick={(e) => { if (e.target === e.currentTarget) showCommandPalette = false; }}>
		<div class="cmd-palette-card animate-scale-in" role="dialog" aria-modal="true" aria-label="Universal Command Palette">
			<div class="cmd-palette-input-box">
				<span class="cmd-palette-icon">🔍</span>
				<!-- svelte-ignore a11y_autofocus -->
				<input
					type="text"
					class="cmd-palette-input"
					placeholder="Search modules, run workflows, switch presets, or jump to page..."
					bind:value={commandSearch}
					autofocus
				/>
				<button
					type="button"
					onclick={() => (showCommandPalette = false)}
					class="cmd-palette-close-btn"
					title="Close (Esc)"
				>
					ESC
				</button>
			</div>

			<div class="cmd-palette-list">
				{#if filteredCommands.length === 0}
					<div class="cmd-palette-empty">
						<span style="font-size: 1.5rem; margin-bottom: 0.25rem;">🔎</span>
						<span style="font-weight: 600; color: #fff;">No matching commands found</span>
						<span style="font-size: 0.75rem; color: #64748b;">Try searching for "billing", "whatsapp", "pos", "trade", "gstin", or "preset"</span>
					</div>
				{:else}
					{#each filteredCommands as cmd, idx}
						<button
							type="button"
							class="cmd-palette-row"
							class:active={idx === selectedCommandIndex}
							onclick={() => executeCommand(cmd)}
							onmouseenter={() => (selectedCommandIndex = idx)}
						>
							<span class="cmd-row-icon">{cmd.icon}</span>
							<div class="cmd-row-content">
								<div class="cmd-row-title">{cmd.title}</div>
								<div class="cmd-row-subtitle">{cmd.subtitle}</div>
							</div>
							<span class="cmd-row-category">{cmd.category}</span>
							<span class="cmd-row-enter" title="Press Enter to execute">↵</span>
						</button>
					{/each}
				{/if}
			</div>

			<div class="cmd-palette-footer">
				<div class="cmd-footer-keys">
					<span class="cmd-key">↑</span>
					<span class="cmd-key">↓</span>
					<span class="cmd-hint">Navigate</span>
				</div>
				<div class="cmd-footer-keys">
					<span class="cmd-key">↵ Enter</span>
					<span class="cmd-hint">Execute</span>
				</div>
				<div class="cmd-footer-keys">
					<span class="cmd-key">Esc</span>
					<span class="cmd-hint">Dismiss</span>
				</div>
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

	/* Workspace Switcher Buttons in Sidebar */
	.ws-btn {
		background: rgba(30, 41, 59, 0.7);
		border: 1px solid rgba(255, 255, 255, 0.08);
		color: #94a3b8;
		font-size: 0.6875rem;
		font-weight: 600;
		padding: 0.35rem 0.4rem;
		border-radius: 6px;
		cursor: pointer;
		transition: all 0.15s ease;
		text-align: center;
	}

	.ws-btn:hover {
		background: rgba(51, 65, 85, 0.8);
		color: #f1f5f9;
		border-color: rgba(99, 102, 241, 0.3);
	}

	.ws-btn.active {
		background: rgba(99, 102, 241, 0.2);
		color: #a5b4fc;
		border-color: #6366f1;
		font-weight: 700;
		box-shadow: 0 0 10px rgba(99, 102, 241, 0.25);
	}

	/* Topbar Command Search Trigger */
	.cmd-search-trigger {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		background: rgba(15, 23, 42, 0.7);
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: 8px;
		padding: 0.35rem 0.75rem;
		color: #94a3b8;
		font-size: 0.8125rem;
		cursor: pointer;
		transition: all 0.15s ease;
		flex: 1;
		max-width: 300px;
	}

	.cmd-search-trigger:hover {
		background: rgba(30, 41, 59, 0.9);
		border-color: rgba(99, 102, 241, 0.4);
		color: #f1f5f9;
	}

	.cmd-search-text {
		flex: 1;
		text-align: left;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.cmd-kbd {
		background: rgba(0, 0, 0, 0.4);
		border: 1px solid rgba(255, 255, 255, 0.15);
		border-radius: 4px;
		padding: 0.1rem 0.35rem;
		font-size: 0.65rem;
		font-family: var(--font-mono);
		color: #cbd5e1;
	}

	/* Topbar Workspace Pills */
	.topbar-workspace-pills {
		display: flex;
		align-items: center;
		gap: 0.25rem;
		background: rgba(15, 23, 42, 0.8);
		padding: 0.2rem;
		border-radius: 8px;
		border: 1px solid rgba(255, 255, 255, 0.08);
	}

	.topbar-pill {
		background: transparent;
		border: none;
		color: #94a3b8;
		font-size: 0.7rem;
		font-weight: 600;
		padding: 0.25rem 0.55rem;
		border-radius: 6px;
		cursor: pointer;
		transition: all 0.15s ease;
		white-space: nowrap;
	}

	.topbar-pill:hover {
		color: #f8fafc;
		background: rgba(255, 255, 255, 0.05);
	}

	.topbar-pill.active {
		background: #6366f1;
		color: #ffffff;
		font-weight: 700;
		box-shadow: 0 2px 8px rgba(99, 102, 241, 0.4);
	}

	/* Command Palette Modal */
	.cmd-palette-backdrop {
		position: fixed;
		inset: 0;
		z-index: 10000;
		background: rgba(11, 15, 25, 0.8);
		backdrop-filter: blur(8px);
		display: flex;
		align-items: flex-start;
		justify-content: center;
		padding-top: 10vh;
		padding-left: 1rem;
		padding-right: 1rem;
	}

	.cmd-palette-card {
		background: #111827;
		border: 1px solid rgba(255, 255, 255, 0.15);
		border-radius: 12px;
		width: 100%;
		max-width: 640px;
		box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.75), 0 0 20px rgba(99, 102, 241, 0.2);
		overflow: hidden;
		display: flex;
		flex-direction: column;
	}

	.cmd-palette-input-box {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		padding: 0.875rem 1.25rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
		background: rgba(255, 255, 255, 0.02);
	}

	.cmd-palette-icon {
		font-size: 1.125rem;
		opacity: 0.8;
	}

	.cmd-palette-input {
		flex: 1;
		background: transparent;
		border: none;
		outline: none;
		color: #f8fafc;
		font-size: 1rem;
		font-family: inherit;
	}

	.cmd-palette-input::placeholder {
		color: #64748b;
	}

	.cmd-palette-close-btn {
		background: rgba(255, 255, 255, 0.06);
		border: 1px solid rgba(255, 255, 255, 0.1);
		color: #94a3b8;
		border-radius: 4px;
		padding: 0.2rem 0.45rem;
		font-size: 0.6875rem;
		font-family: var(--font-mono);
		cursor: pointer;
	}

	.cmd-palette-close-btn:hover {
		background: rgba(255, 255, 255, 0.1);
		color: #fff;
	}

	.cmd-palette-list {
		max-height: 380px;
		overflow-y: auto;
		padding: 0.5rem;
		display: flex;
		flex-direction: column;
		gap: 0.2rem;
	}

	.cmd-palette-row {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		padding: 0.65rem 0.875rem;
		border-radius: 8px;
		background: transparent;
		border: 1px solid transparent;
		color: #cbd5e1;
		text-align: left;
		cursor: pointer;
		transition: all 0.12s ease;
		width: 100%;
	}

	.cmd-palette-row:hover,
	.cmd-palette-row.active {
		background: rgba(99, 102, 241, 0.15);
		border-color: rgba(99, 102, 241, 0.35);
		color: #fff;
	}

	.cmd-row-icon {
		font-size: 1.25rem;
		display: flex;
		align-items: center;
		justify-content: center;
		width: 2rem;
		height: 2rem;
		border-radius: 6px;
		background: rgba(255, 255, 255, 0.05);
		flex-shrink: 0;
	}

	.cmd-row-content {
		flex: 1;
		min-width: 0;
	}

	.cmd-row-title {
		font-size: 0.875rem;
		font-weight: 600;
		color: #f1f5f9;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.cmd-row-subtitle {
		font-size: 0.75rem;
		color: #94a3b8;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.cmd-row-category {
		font-size: 0.65rem;
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		padding: 0.15rem 0.45rem;
		border-radius: 4px;
		background: rgba(255, 255, 255, 0.06);
		color: #94a3b8;
		white-space: nowrap;
	}

	.cmd-row-enter {
		font-family: var(--font-mono);
		font-size: 0.75rem;
		color: #6366f1;
		opacity: 0;
		transition: opacity 0.12s ease;
	}

	.cmd-palette-row.active .cmd-row-enter {
		opacity: 1;
	}

	.cmd-palette-empty {
		padding: 2.5rem 1rem;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 0.5rem;
		text-align: center;
	}

	.cmd-palette-footer {
		display: flex;
		align-items: center;
		gap: 1.25rem;
		padding: 0.6rem 1.25rem;
		border-top: 1px solid rgba(255, 255, 255, 0.08);
		background: rgba(0, 0, 0, 0.2);
	}

	.cmd-footer-keys {
		display: flex;
		align-items: center;
		gap: 0.35rem;
	}

	.cmd-key {
		background: rgba(255, 255, 255, 0.08);
		border: 1px solid rgba(255, 255, 255, 0.12);
		border-radius: 4px;
		padding: 0.1rem 0.35rem;
		font-size: 0.65rem;
		font-family: var(--font-mono);
		color: #cbd5e1;
	}

	.cmd-hint {
		font-size: 0.7rem;
		color: #64748b;
	}
</style>
