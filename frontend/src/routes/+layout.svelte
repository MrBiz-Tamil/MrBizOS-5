<script lang="ts">
	import '../lib/styles/theme.css';
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { api, setSelectedTenantId, type User, type Tenant, type TenantSubscriptionInfo } from '$lib/api';
	import { realtimeBilling } from '$lib/stores/realtime.svelte';
	import UnitSwitcher from '$lib/components/UnitSwitcher.svelte';
	import WorkspaceManagerModal, { type CustomWorkspaceConfig } from '$lib/components/WorkspaceManagerModal.svelte';
	import ToastContainer from '$lib/components/ToastContainer.svelte';

	let { children } = $props();
	let currentUser = $state<User | null>(null);
	let currentTenant = $state<Tenant | null>(null);
	let tenantList = $state<Tenant[]>([]);
	let subInfo = $state<TenantSubscriptionInfo | null>(null);
	let loading = $state(true);
	let currentPath = $derived($page.url.pathname);
	// Pages shown without the app shell (signed-out flows)
	// ?print=1 is the server-side PDF renderer: just the page, no app shell, no banners or dialogs
	let isBarePage = $derived(['/login', '/forgot-password', '/reset-password', '/verify-email', '/terms', '/privacy', '/signup', '/services'].includes(currentPath) || currentPath.startsWith('/i/') || $page.url.searchParams.get('print') === '1');

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

	// Dynamic Works-Based Workspaces & Customization Engine
	type WorkspacePreset = 'FULL' | 'COUNTER' | 'GROWTH' | 'OPERATIONS' | 'HEALTHCARE' | 'TRAVEL' | 'REALESTATE' | 'PROJECTS' | 'COMMERCE' | 'CUSTOM';
	let activeWorkspacePreset = $state<WorkspacePreset>('FULL');
	let activeCustomConfig = $state<CustomWorkspaceConfig | null>(null);
	let showWorkspaceManagerModal = $state(false);
	let showWorkspaceMenu = $state(false);
	let showProfileMenu = $state(false);
	let billingOnlyMode = $derived(activeWorkspacePreset === 'COUNTER');

	const workspacePresetsList: { id: WorkspacePreset; icon: string; name: string; badge: string; desc: string }[] = [
		{ id: 'FULL', icon: '🌐', name: 'Enterprise ERP Suite', badge: 'All Modules', desc: 'Complete 16-module multi-industry ERP' },
		{ id: 'HEALTHCARE', icon: '🏥', name: 'Smart Healthcare OS', badge: 'Clinical Suite', desc: 'Doctors, Telehealth, EMR Rx Pad, Pharmacy POS' },
		{ id: 'TRAVEL', icon: '✈️', name: 'Travel & Tours OS', badge: 'GDS & DMC Hub', desc: 'Tour packages, PNR hold clocks & DMC vouchers' },
		{ id: 'REALESTATE', icon: '🏢', name: 'Real Estate Matrix', badge: 'Plots & RERA', desc: '24-plot layout matrix, Indian land units & visits' },
		{ id: 'PROJECTS', icon: '📁', name: 'UTMS Delivery OS', badge: 'SLA Engine', desc: 'SLA engine, Kanban pipeline & milestone billing' },
		{ id: 'COUNTER', icon: '⚡', name: 'Express Retail Counter', badge: 'POS Terminal', desc: 'Ultra-fast retail counter with instant UPI QR' },
		{ id: 'COMMERCE', icon: '🛍️', name: 'E-Commerce & POD', badge: 'Apparel & Labs', desc: 'Etsy sync, blank matrix & print lab queue' },
		{ id: 'GROWTH', icon: '💬', name: 'Social Sales & AI Voice', badge: 'Growth Desk', desc: 'Omnichannel WhatsApp/IG, AI voice & deals' },
		{ id: 'OPERATIONS', icon: '📦', name: 'Supply Chain & Ops', badge: '3-Way Match', desc: '3-way procurement, batches & vendor khata' }
	];

	let currentWorkspaceLabel = $derived.by(() => {
		switch (activeWorkspacePreset) {
			case 'HEALTHCARE': return { icon: '🏥', name: 'Smart Healthcare OS', badge: 'Clinical Suite' };
			case 'TRAVEL': return { icon: '✈️', name: 'Travel & Tours OS', badge: 'GDS & DMC Hub' };
			case 'REALESTATE': return { icon: '🏢', name: 'Real Estate Matrix', badge: 'Plots & RERA' };
			case 'PROJECTS': return { icon: '📁', name: 'UTMS Delivery OS', badge: 'SLA Engine' };
			case 'COMMERCE': return { icon: '🛍️', name: 'E-Commerce & POD', badge: 'Apparel & Labs' };
			case 'COUNTER': return { icon: '⚡', name: 'Express Retail Counter', badge: 'POS Terminal' };
			case 'GROWTH': return { icon: '💬', name: 'Social Sales & AI Voice', badge: 'Growth Desk' };
			case 'OPERATIONS': return { icon: '📦', name: 'Supply Chain & Ops', badge: '3-Way Match' };
			case 'CUSTOM': return {
				icon: activeCustomConfig?.icon || '💼',
				name: activeCustomConfig?.name || 'Custom Workspace',
				badge: activeCustomConfig?.badgeText || 'Custom Desk'
			};
			default: return { icon: '🌐', name: 'Enterprise ERP Suite', badge: 'All 16 Modules' };
		}
	});

	function setWorkspacePreset(preset: WorkspacePreset, customConfig?: CustomWorkspaceConfig) {
		activeWorkspacePreset = preset;
		if (customConfig) {
			activeCustomConfig = customConfig;
		}
		if (typeof localStorage !== 'undefined') {
			localStorage.setItem('mrbiz_workspace_preset', preset);
			localStorage.setItem('mrbiz_billing_only_mode', String(preset === 'COUNTER'));
			if (customConfig) {
				localStorage.setItem('mrbiz_custom_workspace_v2', JSON.stringify(customConfig));
			}
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
		{ id: 'act-services', title: 'Services & Verticals Hub', subtitle: 'Explore all 16 business modules, POS, and templates', icon: '🌐', category: 'Actions', href: '/services' },
		{ id: 'act-bill', title: 'New Express Bill', subtitle: 'Open 3-second quick billing counter with UPI QR', icon: '⚡', category: 'Actions', href: '/billing' },
		{ id: 'act-pos', title: 'Retail POS Terminal', subtitle: 'Barcode scanner checkout screen', icon: '🛒', category: 'Actions', href: '/pos' },
		{ id: 'act-inv', title: 'Create Tax Invoice', subtitle: 'Full GST sales invoice with logistics', icon: '📄', category: 'Actions', href: '/invoices/new' },
		{ id: 'act-call', title: 'AI Voice BPO Dial', subtitle: 'Outbound campaign & live audio screen pop', icon: '🎙️', category: 'Actions', href: '/bpo-voice' },
		{ id: 'act-inbox', title: 'Omnichannel Social Inbox', subtitle: 'Unified WhatsApp, LinkedIn, IG, FB messages', icon: '💬', category: 'Actions', href: '/inbox' },
		{ id: 'act-rec', title: 'Bank Auto-Reconciliation', subtitle: 'Drag & drop bank statement CSV match', icon: '🏦', category: 'Actions', href: '/accounting/reconciliation' },
		{ id: 'act-pay', title: 'HRMS 1-Click Payroll', subtitle: 'Process monthly salaries and timesheets', icon: '👥', category: 'Actions', href: '/hrms' },

		// Works-Based Workspaces & Presets
		{ id: 'ws-custom-modal', title: 'Custom Workspace Studio...', subtitle: 'Configure modules, branding & landing route for your team', icon: '🛠️', category: 'Workspaces', action: () => { showWorkspaceManagerModal = true; } },
		{ id: 'ws-health', title: 'Switch Workspace: Smart Healthcare OS', subtitle: 'Doctor Rx Pad, Pharmacy POS, Nurse triage & owner stats', icon: '🏥', category: 'Workspaces', action: () => setWorkspacePreset('HEALTHCARE') },
		{ id: 'ws-travel', title: 'Switch Workspace: Travel & Tours Agency', subtitle: 'Tour packages, PNR hold clocks & DMC vouchers', icon: '✈️', category: 'Workspaces', action: () => setWorkspacePreset('TRAVEL') },
		{ id: 'ws-estate', title: 'Switch Workspace: Real Estate Deals', subtitle: '24-plot layout matrix, Indian land units & cab visits', icon: '🏢', category: 'Workspaces', action: () => setWorkspacePreset('REALESTATE') },
		{ id: 'ws-proj', title: 'Switch Workspace: UTMS Projects & SOW', subtitle: 'SLA engine, Kanban pipeline & milestone invoicing', icon: '📁', category: 'Workspaces', action: () => setWorkspacePreset('PROJECTS') },
		{ id: 'ws-ecom', title: 'Switch Workspace: E-Commerce & Print-on-Demand', subtitle: 'Blank inventory matrix, print lab queue & courier NDR', icon: '🛍️', category: 'Workspaces', action: () => setWorkspacePreset('COMMERCE') },
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
				const savedCustom = localStorage.getItem('mrbiz_custom_workspace_v2');
				if (savedCustom) {
					try {
						activeCustomConfig = JSON.parse(savedCustom);
					} catch (e) {}
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
						<div style="display: flex; align-items: center; gap: 0.35rem;">
							<span style="font-size: 0.85rem;">{currentWorkspaceLabel.icon}</span>
							<span style="font-size: 0.65rem; font-weight: 700; color: #94a3b8; text-transform: uppercase; letter-spacing: 0.05em;">{currentWorkspaceLabel.badge}</span>
						</div>
						<button
							type="button"
							onclick={() => { showCommandPalette = true; commandSearch = ''; selectedCommandIndex = 0; }}
							style="background: none; border: none; font-size: 0.65rem; color: #818cf8; cursor: pointer; font-family: var(--font-mono); display: flex; align-items: center; gap: 0.2rem;"
							title="Command Palette"
						>
							🔍 <span>⌘K</span>
						</button>
					</div>

					<!-- Primary Works Studio Button -->
					<button
						type="button"
						class="ws-studio-btn"
						onclick={() => (showWorkspaceManagerModal = true)}
						title="Manage & Customize Works Workspace"
					>
						<span>🛠️ Works Studio & Presets</span>
						<span style="font-size: 0.65rem; color: #a5b4fc; font-weight: 700;">⚙️ Setup</span>
					</button>

					<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 0.25rem; margin-top: 0.35rem;">
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
							class:active={activeWorkspacePreset === 'HEALTHCARE'}
							onclick={() => setWorkspacePreset('HEALTHCARE')}
							title="Smart Healthcare Hub"
						>
							🏥 Clinic
						</button>
						<button
							type="button"
							class="ws-btn"
							class:active={activeWorkspacePreset === 'TRAVEL'}
							onclick={() => setWorkspacePreset('TRAVEL')}
							title="Travel & Tour Agency OS"
						>
							✈️ Travel
						</button>
						<button
							type="button"
							class="ws-btn"
							class:active={activeWorkspacePreset === 'REALESTATE'}
							onclick={() => setWorkspacePreset('REALESTATE')}
							title="Real Estate Matrix"
						>
							🏢 Real Estate
						</button>
						<button
							type="button"
							class="ws-btn"
							class:active={activeWorkspacePreset === 'PROJECTS'}
							onclick={() => setWorkspacePreset('PROJECTS')}
							title="UTMS Projects & SOW"
						>
							📁 UTMS
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

				{:else if activeWorkspacePreset === 'HEALTHCARE'}
					<!-- ============================================== -->
					<!-- PRESET: SMART HEALTHCARE & HOSPITAL OS         -->
					<!-- ============================================== -->
					<div style="font-size: 0.6875rem; font-weight: 700; color: #38bdf8; text-transform: uppercase; letter-spacing: 0.05em; padding: 0.25rem 0.5rem; display: flex; align-items: center; justify-content: space-between;">
						<span>Smart Healthcare OS</span>
						<span style="font-size: 0.625rem; background: rgba(56, 189, 248, 0.15); color: #38bdf8; padding: 0.1rem 0.4rem; border-radius: 4px;">4 Workstations</span>
					</div>

					<a href="/healthcare" class="nav-item" class:active={currentPath.startsWith('/healthcare')} style="color: #38bdf8; font-weight: 700; background: rgba(56, 189, 248, 0.1); border-left: 3px solid #38bdf8;">
						<span class="nav-icon">🏥</span> Clinical Workstations (Rx/EMR)
					</a>

					<a href="/billing" class="nav-item" class:active={currentPath.startsWith('/billing')}>
						<span class="nav-icon">⚡</span> Quick Patient OPD Billing
					</a>

					<a href="/pos" class="nav-item" class:active={currentPath.startsWith('/pos')}>
						<span class="nav-icon">🛒</span> Pharmacy POS Dispenser
					</a>

					<a href="/invoices" class="nav-item" class:active={currentPath.startsWith('/invoices')}>
						<span class="nav-icon">📄</span> Invoices & Insurance Claims
					</a>

					<a href="/customers" class="nav-item" class:active={currentPath === '/customers'}>
						<span class="nav-icon">👥</span> Patient Records & History
					</a>

					<a href="/products" class="nav-item" class:active={currentPath === '/products'}>
						<span class="nav-icon">📦</span> Pharmacy Formulary / Drugs
					</a>

					<a href="/inventory" class="nav-item" class:active={currentPath === '/inventory'}>
						<span class="nav-icon">📋</span> Drug Batches & Expiry Dates
					</a>

					<a href="/inbox" class="nav-item" class:active={currentPath.startsWith('/inbox')}>
						<span class="nav-icon">💬</span> Patient WhatsApp Helpline
					</a>

					<a href="/accounting" class="nav-item" class:active={currentPath.startsWith('/accounting')}>
						<span class="nav-icon">📖</span> Hospital Accounts & P&L
					</a>

					<a href="/hrms" class="nav-item" class:active={currentPath.startsWith('/hrms')}>
						<span class="nav-icon">👥</span> Doctors & Nursing Roster
					</a>

					<a href="/settings" class="nav-item" class:active={currentPath === '/settings'}>
						<span class="nav-icon">⚙️</span> Clinic Profile & Rx Header
					</a>

				{:else if activeWorkspacePreset === 'TRAVEL'}
					<!-- ============================================== -->
					<!-- PRESET: TRAVEL & TOUR AGENCY OPERATIONS        -->
					<!-- ============================================== -->
					<div style="font-size: 0.6875rem; font-weight: 700; color: #06b6d4; text-transform: uppercase; letter-spacing: 0.05em; padding: 0.25rem 0.5rem; display: flex; align-items: center; justify-content: space-between;">
						<span>Travel & Tour Agency OS</span>
						<span style="font-size: 0.625rem; background: rgba(6, 182, 212, 0.15); color: #06b6d4; padding: 0.1rem 0.4rem; border-radius: 4px;">GDS & DMC</span>
					</div>

					<a href="/travel" class="nav-item" class:active={currentPath.startsWith('/travel')} style="color: #06b6d4; font-weight: 700; background: rgba(6, 182, 212, 0.1); border-left: 3px solid #06b6d4;">
						<span class="nav-icon">✈️</span> Travel Agency Desk (PNR & Visa)
					</a>

					<a href="/inbox" class="nav-item" class:active={currentPath.startsWith('/inbox')}>
						<span class="nav-icon">💬</span> Omnichannel WhatsApp Leads
					</a>

					<a href="/bpo-voice" class="nav-item" class:active={currentPath.startsWith('/bpo-voice')}>
						<span class="nav-icon">🎙️</span> AI Voice Flight / Visa Bot
					</a>

					<a href="/billing" class="nav-item" class:active={currentPath.startsWith('/billing')}>
						<span class="nav-icon">⚡</span> Quick Tour Booking Bill
					</a>

					<a href="/invoices" class="nav-item" class:active={currentPath.startsWith('/invoices')}>
						<span class="nav-icon">📄</span> Tax Invoices & DMC Vouchers
					</a>

					<a href="/quotations" class="nav-item" class:active={currentPath.startsWith('/quotations')}>
						<span class="nav-icon">📝</span> Custom Tour Quotations
					</a>

					<a href="/customers" class="nav-item" class:active={currentPath === '/customers'}>
						<span class="nav-icon">👥</span> Travelers & Passport DB
					</a>

					<a href="/journeys" class="nav-item" class:active={currentPath.startsWith('/journeys')}>
						<span class="nav-icon">🛤️</span> Tour Itinerary Engine
					</a>

					<a href="/accounting" class="nav-item" class:active={currentPath.startsWith('/accounting')}>
						<span class="nav-icon">📖</span> Tour Operator Ledger & P&L
					</a>

					<a href="/settings" class="nav-item" class:active={currentPath === '/settings'}>
						<span class="nav-icon">⚙️</span> Agency Setup & Airlines
					</a>

				{:else if activeWorkspacePreset === 'REALESTATE'}
					<!-- ============================================== -->
					<!-- PRESET: REAL ESTATE & LAYOUT DEVELOPMENT      -->
					<!-- ============================================== -->
					<div style="font-size: 0.6875rem; font-weight: 700; color: #10b981; text-transform: uppercase; letter-spacing: 0.05em; padding: 0.25rem 0.5rem; display: flex; align-items: center; justify-content: space-between;">
						<span>Real Estate OS</span>
						<span style="font-size: 0.625rem; background: rgba(16, 185, 129, 0.15); color: #10b981; padding: 0.1rem 0.4rem; border-radius: 4px;">Plots & RERA</span>
					</div>

					<a href="/realestate" class="nav-item" class:active={currentPath.startsWith('/realestate')} style="color: #10b981; font-weight: 700; background: rgba(16, 185, 129, 0.1); border-left: 3px solid #10b981;">
						<span class="nav-icon">🏢</span> 24-Plot Matrix & Site Visits
					</a>

					<a href="/bpo-voice" class="nav-item" class:active={currentPath.startsWith('/bpo-voice')}>
						<span class="nav-icon">🎙️</span> AI Voice Site Visit Dialer
					</a>

					<a href="/inbox" class="nav-item" class:active={currentPath.startsWith('/inbox')}>
						<span class="nav-icon">💬</span> Buyer WhatsApp & Inquiries
					</a>

					<a href="/projects" class="nav-item" class:active={currentPath.startsWith('/projects')}>
						<span class="nav-icon">📁</span> Layout Development Tasks
					</a>

					<a href="/billing" class="nav-item" class:active={currentPath.startsWith('/billing')}>
						<span class="nav-icon">⚡</span> Plot Advance Token Receipt
					</a>

					<a href="/invoices" class="nav-item" class:active={currentPath.startsWith('/invoices')}>
						<span class="nav-icon">📄</span> Sales Invoices & Agreements
					</a>

					<a href="/quotations" class="nav-item" class:active={currentPath.startsWith('/quotations')}>
						<span class="nav-icon">📝</span> Plot Cost Sheets & Quotations
					</a>

					<a href="/customers" class="nav-item" class:active={currentPath === '/customers'}>
						<span class="nav-icon">👥</span> Property Buyers & Leads
					</a>

					<a href="/accounting" class="nav-item" class:active={currentPath.startsWith('/accounting')}>
						<span class="nav-icon">📖</span> Project Escrow & Land Ledger
					</a>

					<a href="/settings" class="nav-item" class:active={currentPath === '/settings'}>
						<span class="nav-icon">⚙️</span> Builder & RERA Setup
					</a>

				{:else if activeWorkspacePreset === 'PROJECTS'}
					<!-- ============================================== -->
					<!-- PRESET: UTMS PROJECTS & SOW DELIVERY           -->
					<!-- ============================================== -->
					<div style="font-size: 0.6875rem; font-weight: 700; color: #818cf8; text-transform: uppercase; letter-spacing: 0.05em; padding: 0.25rem 0.5rem; display: flex; align-items: center; justify-content: space-between;">
						<span>UTMS Project OS</span>
						<span style="font-size: 0.625rem; background: rgba(129, 140, 248, 0.15); color: #818cf8; padding: 0.1rem 0.4rem; border-radius: 4px;">SLA Engine</span>
					</div>

					<a href="/projects" class="nav-item" class:active={currentPath.startsWith('/projects')} style="color: #818cf8; font-weight: 700; background: rgba(129, 140, 248, 0.1); border-left: 3px solid #818cf8;">
						<span class="nav-icon">📁</span> UTMS Tasks & SLA Command Center
					</a>

					<a href="/billing" class="nav-item" class:active={currentPath.startsWith('/billing')}>
						<span class="nav-icon">⚡</span> Milestone Billing Triggers
					</a>

					<a href="/invoices" class="nav-item" class:active={currentPath.startsWith('/invoices')}>
						<span class="nav-icon">📄</span> Client Milestone Invoices
					</a>

					<a href="/customers" class="nav-item" class:active={currentPath === '/customers'}>
						<span class="nav-icon">👥</span> Client SOW Accounts
					</a>

					<a href="/hrms" class="nav-item" class:active={currentPath.startsWith('/hrms')}>
						<span class="nav-icon">👥</span> Timesheets & Team Payroll
					</a>

					<a href="/support" class="nav-item" class:active={currentPath.startsWith('/support')}>
						<span class="nav-icon">🎫</span> Client SLA Helpdesk
					</a>

					<a href="/inbox" class="nav-item" class:active={currentPath.startsWith('/inbox')}>
						<span class="nav-icon">💬</span> Omnichannel Client Inbox
					</a>

					<a href="/workflow" class="nav-item" class:active={currentPath.startsWith('/workflow')}>
						<span class="nav-icon">🤖</span> Autonomous AI Workflows
					</a>

					<a href="/accounting" class="nav-item" class:active={currentPath.startsWith('/accounting')}>
						<span class="nav-icon">📖</span> Consulting SOW Ledger & P&L
					</a>

					<a href="/settings" class="nav-item" class:active={currentPath === '/settings'}>
						<span class="nav-icon">⚙️</span> Agency Delivery Settings
					</a>

				{:else if activeWorkspacePreset === 'COMMERCE'}
					<!-- ============================================== -->
					<!-- PRESET: E-COMMERCE & PRINT-ON-DEMAND           -->
					<!-- ============================================== -->
					<div style="font-size: 0.6875rem; font-weight: 700; color: #ec4899; text-transform: uppercase; letter-spacing: 0.05em; padding: 0.25rem 0.5rem; display: flex; align-items: center; justify-content: space-between;">
						<span>E-Commerce & POD OS</span>
						<span style="font-size: 0.625rem; background: rgba(236, 72, 153, 0.15); color: #ec4899; padding: 0.1rem 0.4rem; border-radius: 4px;">Apparel & Labs</span>
					</div>

					<a href="/ecommerce" class="nav-item" class:active={currentPath.startsWith('/ecommerce')} style="color: #ec4899; font-weight: 700; background: rgba(236, 72, 153, 0.1); border-left: 3px solid #ec4899;">
						<span class="nav-icon">🛍️</span> Blank Matrix & Print Lab Queue
					</a>

					<a href="/pos" class="nav-item" class:active={currentPath.startsWith('/pos')}>
						<span class="nav-icon">🛒</span> Retail Counter POS
					</a>

					<a href="/billing" class="nav-item" class:active={currentPath.startsWith('/billing')}>
						<span class="nav-icon">⚡</span> Quick Order Invoicing
					</a>

					<a href="/procurement" class="nav-item" class:active={currentPath.startsWith('/procurement')}>
						<span class="nav-icon">📦</span> 3-Way Blank Procurement
					</a>

					<a href="/inventory" class="nav-item" class:active={currentPath === '/inventory'}>
						<span class="nav-icon">📋</span> Garment Stock & Color/Size
					</a>

					<a href="/trade/shipments" class="nav-item" class:active={currentPath.startsWith('/trade/shipments')}>
						<span class="nav-icon">🚢</span> Courier Logistics & NDR
					</a>

					<a href="/inbox" class="nav-item" class:active={currentPath.startsWith('/inbox')}>
						<span class="nav-icon">💬</span> Omnichannel Buyer Support
					</a>

					<a href="/accounting" class="nav-item" class:active={currentPath.startsWith('/accounting')}>
						<span class="nav-icon">📖</span> Store Financial Accounting
					</a>

					<a href="/settings" class="nav-item" class:active={currentPath === '/settings'}>
						<span class="nav-icon">⚙️</span> Print Lab & Store Settings
					</a>

				{:else if activeWorkspacePreset === 'CUSTOM'}
					<!-- ============================================== -->
					<!-- PRESET: DYNAMIC CUSTOM WORKSPACE STUDIO        -->
					<!-- ============================================== -->
					<div style="font-size: 0.6875rem; font-weight: 700; color: #a855f7; text-transform: uppercase; letter-spacing: 0.05em; padding: 0.25rem 0.5rem; display: flex; align-items: center; justify-content: space-between;">
						<span>{activeCustomConfig?.name || 'Custom Workspace'}</span>
						<button
							type="button"
							onclick={() => (showWorkspaceManagerModal = true)}
							style="background: none; border: none; font-size: 0.65rem; color: #c084fc; cursor: pointer; text-decoration: underline;"
						>
							Edit
						</button>
					</div>

					{#if activeCustomConfig?.primaryAction}
						<a href={activeCustomConfig.primaryAction.href} class="nav-item" style="color: #c084fc; font-weight: 700; background: rgba(168, 85, 247, 0.12); border-left: 3px solid #c084fc;">
							<span class="nav-icon">{activeCustomConfig.primaryAction.icon || '⚡'}</span> {activeCustomConfig.primaryAction.label}
						</a>
					{/if}

					{#if !activeCustomConfig?.modules || activeCustomConfig.modules.includes('billing')}
						<a href="/billing" class="nav-item" class:active={currentPath.startsWith('/billing')}>
							<span class="nav-icon">⚡</span> Express Billing
						</a>
					{/if}
					{#if activeCustomConfig?.modules?.includes('pos')}
						<a href="/pos" class="nav-item" class:active={currentPath.startsWith('/pos')}>
							<span class="nav-icon">🛒</span> Retail POS Terminal
						</a>
					{/if}
					{#if activeCustomConfig?.modules?.includes('healthcare')}
						<a href="/healthcare" class="nav-item" class:active={currentPath.startsWith('/healthcare')}>
							<span class="nav-icon">🏥</span> Smart Healthcare Hub
						</a>
					{/if}
					{#if activeCustomConfig?.modules?.includes('travel')}
						<a href="/travel" class="nav-item" class:active={currentPath.startsWith('/travel')}>
							<span class="nav-icon">✈️</span> Travel Agency OS
						</a>
					{/if}
					{#if activeCustomConfig?.modules?.includes('realestate')}
						<a href="/realestate" class="nav-item" class:active={currentPath.startsWith('/realestate')}>
							<span class="nav-icon">🏢</span> Real Estate Deals
						</a>
					{/if}
					{#if activeCustomConfig?.modules?.includes('ecommerce')}
						<a href="/ecommerce" class="nav-item" class:active={currentPath.startsWith('/ecommerce')}>
							<span class="nav-icon">🛍️</span> E-Commerce & Print-on-Demand
						</a>
					{/if}
					{#if activeCustomConfig?.modules?.includes('projects')}
						<a href="/projects" class="nav-item" class:active={currentPath.startsWith('/projects')}>
							<span class="nav-icon">📁</span> Projects & UTMS Tasks
						</a>
					{/if}
					{#if activeCustomConfig?.modules?.includes('inbox')}
						<a href="/inbox" class="nav-item" class:active={currentPath.startsWith('/inbox')}>
							<span class="nav-icon">💬</span> Omnichannel Social Inbox
						</a>
					{/if}
					{#if activeCustomConfig?.modules?.includes('bpo-voice')}
						<a href="/bpo-voice" class="nav-item" class:active={currentPath.startsWith('/bpo-voice')}>
							<span class="nav-icon">🎙️</span> AI Voice Agent BPO Desk
						</a>
					{/if}
					{#if activeCustomConfig?.modules?.includes('procurement')}
						<a href="/procurement" class="nav-item" class:active={currentPath.startsWith('/procurement')}>
							<span class="nav-icon">📦</span> 3-Way Procurement
						</a>
					{/if}
					{#if activeCustomConfig?.modules?.includes('hrms')}
						<a href="/hrms" class="nav-item" class:active={currentPath.startsWith('/hrms')}>
							<span class="nav-icon">👥</span> HRMS & Automated Payroll
						</a>
					{/if}
					{#if activeCustomConfig?.modules?.includes('trade')}
						<a href="/trade/shipments" class="nav-item" class:active={currentPath.startsWith('/trade/shipments')}>
							<span class="nav-icon">🚢</span> Export & Import Trade OS
						</a>
					{/if}
					{#if activeCustomConfig?.modules?.includes('support')}
						<a href="/support" class="nav-item" class:active={currentPath.startsWith('/support')}>
							<span class="nav-icon">🎫</span> Support Helpdesk & SLAs
						</a>
					{/if}
					{#if activeCustomConfig?.modules?.includes('journeys')}
						<a href="/journeys" class="nav-item" class:active={currentPath.startsWith('/journeys')}>
							<span class="nav-icon">🛤️</span> Journey Lifecycle Engine
						</a>
					{/if}
					{#if !activeCustomConfig?.modules || activeCustomConfig.modules.includes('accounting')}
						<a href="/accounting" class="nav-item" class:active={currentPath.startsWith('/accounting')}>
							<span class="nav-icon">📖</span> Financial Accounting
						</a>
					{/if}
					{#if !activeCustomConfig?.modules || activeCustomConfig.modules.includes('settings')}
						<a href="/settings" class="nav-item" class:active={currentPath === '/settings'}>
							<span class="nav-icon">⚙️</span> Company Profile & Settings
						</a>
					{/if}

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

					<a href="/services" class="nav-item" class:active={currentPath === '/services'} style="color: #38bdf8; font-weight: 700; background: rgba(56, 189, 248, 0.08); border-left: 3px solid #38bdf8;">
						<span class="nav-icon">🌐</span> All 16 Services Hub
					</a>

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
			<!-- Clean Organized Topbar -->
			<header class="topbar">
				<!-- Left: Company Scope & Identity -->
				<div class="topbar-left">
					{#if currentUser?.is_super_admin && tenantList.length > 0}
						<div class="company-scope-chip" title="Active Enterprise Tenant Scope">
							<span class="company-icon">🏢</span>
							<select
								class="company-scope-select"
								value={currentTenant?.id}
								onchange={handleSwitchTenant}
								title="Switch Operating Company"
							>
								{#each tenantList as t}
									<option value={t.id}>{t.name}</option>
								{/each}
							</select>
						</div>
					{:else}
						<div class="company-brand-chip" title="GSTIN: {currentTenant?.gstin || 'Unregistered'}">
							<span class="company-icon">🏢</span>
							<span class="company-name">{currentTenant?.name || 'Private Enterprise'}</span>
						</div>
					{/if}

					<span class="plan-pill {isNextPlan ? 'plan-next' : 'plan-basic'}">
						{isNextPlan ? '💎 Next Plan' : '🌱 Basic'}
					</span>
				</div>

				<!-- Center: Compact Universal Search + Single Unified Workspace Switcher -->
				<div class="topbar-center">
					<button
						type="button"
						onclick={() => { showCommandPalette = true; commandSearch = ''; selectedCommandIndex = 0; }}
						class="cmd-search-trigger"
						title="Quick Search & Navigation (Ctrl+K)"
					>
						<span class="search-lens">🔍</span>
						<span class="cmd-search-text">Search anything or jump to...</span>
						<kbd class="cmd-kbd">Ctrl K</kbd>
					</button>

					<!-- Unified Workspace Dropdown Button -->
					<div class="workspace-dropdown-container">
						<button
							type="button"
							class="workspace-trigger-btn"
							class:active={showWorkspaceMenu}
							onclick={() => (showWorkspaceMenu = !showWorkspaceMenu)}
							title="Switch Workspace Preset or Launch Studio"
						>
							<span class="ws-icon">{currentWorkspaceLabel.icon}</span>
							<span class="ws-title">{currentWorkspaceLabel.name}</span>
							<span class="ws-tag">{currentWorkspaceLabel.badge}</span>
							<svg class="ws-chevron" class:rotate={showWorkspaceMenu} viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<polyline points="6 9 12 15 18 9"/>
							</svg>
						</button>

						{#if showWorkspaceMenu}
							<button
								class="menu-backdrop"
								type="button"
								aria-label="Close workspace menu"
								onclick={() => (showWorkspaceMenu = false)}
							></button>
							<div class="workspace-popover animate-fade-in">
								<div class="popover-header">
									<span class="popover-title">SWITCH BUSINESS WORKSPACE</span>
									<span class="popover-count">{workspacePresetsList.length} presets</span>
								</div>
								<div class="popover-list">
									{#each workspacePresetsList as p}
										<button
											type="button"
											class="workspace-item"
											class:selected={activeWorkspacePreset === p.id}
											onclick={() => {
												setWorkspacePreset(p.id);
												showWorkspaceMenu = false;
											}}
										>
											<span class="item-icon">{p.icon}</span>
											<div class="item-info">
												<div class="item-head">
													<span class="item-name">{p.name}</span>
													<span class="item-badge">{p.badge}</span>
												</div>
												<div class="item-desc">{p.desc}</div>
											</div>
											{#if activeWorkspacePreset === p.id}
												<span class="active-check">✓</span>
											{/if}
										</button>
									{/each}
								</div>
								<div class="popover-footer">
									<button
										type="button"
										class="studio-launch-btn"
										onclick={() => {
											showWorkspaceMenu = false;
											showWorkspaceManagerModal = true;
										}}
									>
										🛠️ Custom Workspace Studio & Presets...
									</button>
								</div>
							</div>
						{/if}
					</div>
				</div>

				<!-- Right: Live Telemetry, Quick Create, Profile Dropdown -->
				<div class="topbar-right">
					<div class="telemetry-chip" title="Live Server-Sent Events Connected">
						<span class="pulse-dot"></span>
						<span class="telemetry-label">Live</span>
					</div>

					<a href="/invoices/new" class="btn btn-primary btn-sm glow-btn action-create-btn">
						<svg class="btn-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<line x1="12" y1="5" x2="12" y2="19"/>
							<line x1="5" y1="12" x2="19" y2="12"/>
						</svg>
						Create Invoice
					</a>

					<!-- Profile Menu Trigger -->
					<div class="profile-dropdown-container">
						<button
							type="button"
							class="profile-chip-btn"
							class:active={showProfileMenu}
							onclick={() => (showProfileMenu = !showProfileMenu)}
							title="{currentUser?.full_name || 'Admin'} ({currentUser?.role || 'User'})"
						>
							<div class="avatar-badge">
								{getInitials(currentUser?.full_name)}
							</div>
							<span class="user-role-text">{currentUser?.role || 'Admin'}</span>
							<svg class="chevron-mini" class:rotate={showProfileMenu} viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<polyline points="6 9 12 15 18 9"/>
							</svg>
						</button>

						{#if showProfileMenu}
							<button
								class="menu-backdrop"
								type="button"
								aria-label="Close profile menu"
								onclick={() => (showProfileMenu = false)}
							></button>
							<div class="profile-popover animate-fade-in">
								<div class="profile-meta-card">
									<div class="meta-fullname">{currentUser?.full_name || 'Executive Administrator'}</div>
									<div class="meta-email font-mono">{currentUser?.email || 'admin@mrbiz.local'}</div>
									<div class="meta-tenant">{currentTenant?.name || 'Global Business Group'}</div>
								</div>
								<div class="profile-links">
									<a href="/settings" class="profile-link" onclick={() => (showProfileMenu = false)}>
										⚙️ Organization Settings
									</a>
									<a href="/admin/billing" class="profile-link" onclick={() => (showProfileMenu = false)}>
										💳 Plan Invoices & ARR Cockpit
									</a>
									<a href="/hrms" class="profile-link" onclick={() => (showProfileMenu = false)}>
										👥 Human Resources & Payroll
									</a>
									<a href="/services" class="profile-link" onclick={() => (showProfileMenu = false)}>
										🌐 16 Industry Modules Hub
									</a>
									<div class="menu-divider"></div>
									<button type="button" class="profile-link text-danger" onclick={() => { showProfileMenu = false; handleLogout(); }}>
										🚪 Sign Out of Workspace
									</button>
								</div>
							</div>
						{/if}
					</div>
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

<!-- Dynamic Works Workspace Manager & Customizer Modal -->
<WorkspaceManagerModal
	open={showWorkspaceManagerModal}
	onclose={() => (showWorkspaceManagerModal = false)}
	onapply={(preset, customConfig) => {
		setWorkspacePreset(preset as WorkspacePreset, customConfig);
		if (customConfig?.homeRoute && customConfig.homeRoute !== currentPath) {
			goto(customConfig.homeRoute);
		}
	}}
/>

<!-- Global Floating Toast Feedback Viewport -->
<ToastContainer />

<style>
	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	/* Workspace Studio Sidebar Button */
	.ws-studio-btn {
		width: 100%;
		background: linear-gradient(135deg, rgba(99, 102, 241, 0.18) 0%, rgba(168, 85, 247, 0.18) 100%);
		border: 1px solid rgba(99, 102, 241, 0.35);
		border-radius: 6px;
		color: #e0e7ff;
		font-size: 0.6875rem;
		font-weight: 700;
		padding: 0.4rem 0.5rem;
		display: flex;
		align-items: center;
		justify-content: space-between;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.ws-studio-btn:hover {
		background: linear-gradient(135deg, rgba(99, 102, 241, 0.3) 0%, rgba(168, 85, 247, 0.3) 100%);
		border-color: #818cf8;
		box-shadow: 0 0 12px rgba(99, 102, 241, 0.25);
		color: #ffffff;
	}

	/* Topbar Works Selector Pill */
	.topbar-works-selector-pill {
		display: flex;
		align-items: center;
		gap: 0.35rem;
		background: rgba(30, 41, 59, 0.7);
		border: 1px solid rgba(255, 255, 255, 0.12);
		padding: 0.2rem 0.6rem;
		border-radius: 6px;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.topbar-works-selector-pill:hover {
		background: rgba(51, 65, 85, 0.9);
		border-color: #818cf8;
	}

	.topbar-works-tag {
		font-size: 0.6rem;
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 0.04em;
		background: rgba(99, 102, 241, 0.2);
		color: #c7d2fe;
		padding: 0.1rem 0.35rem;
		border-radius: 4px;
		border: 1px solid rgba(99, 102, 241, 0.3);
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

	/* ========================================================
	   CLEAN ORGANIZED TOPBAR DESIGN SYSTEM
	   ======================================================== */
	.topbar {
		height: 60px;
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0 1.5rem;
		background: rgba(15, 23, 42, 0.85);
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
		backdrop-filter: blur(12px);
		position: sticky;
		top: 0;
		z-index: 50;
		gap: 1rem;
	}

	.topbar-left {
		display: flex;
		align-items: center;
		gap: 0.65rem;
		flex-shrink: 0;
	}

	.company-scope-chip,
	.company-brand-chip {
		display: flex;
		align-items: center;
		gap: 0.45rem;
		background: rgba(255, 255, 255, 0.04);
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 8px;
		padding: 0.35rem 0.65rem;
		font-size: 0.85rem;
		font-weight: 600;
		color: #f1f5f9;
	}

	.company-icon {
		font-size: 0.95rem;
	}

	.company-name {
		white-space: nowrap;
	}

	.company-scope-select {
		background: transparent;
		border: none;
		color: #f1f5f9;
		font-size: 0.85rem;
		font-weight: 600;
		cursor: pointer;
		outline: none;
		padding-right: 0.25rem;
	}

	.company-scope-select option {
		background: #0f172a;
		color: #f1f5f9;
	}

	.plan-pill {
		font-size: 0.7rem;
		font-weight: 700;
		padding: 0.2rem 0.55rem;
		border-radius: 9999px;
		letter-spacing: 0.03em;
		white-space: nowrap;
	}

	.plan-next {
		background: rgba(99, 102, 241, 0.15);
		color: #a5b4fc;
		border: 1px solid rgba(99, 102, 241, 0.3);
	}

	.plan-basic {
		background: rgba(16, 185, 129, 0.15);
		color: #34d399;
		border: 1px solid rgba(16, 185, 129, 0.3);
	}

	/* Center: Search & Unified Workspace Switcher */
	.topbar-center {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		flex: 1;
		justify-content: center;
		max-width: 680px;
	}

	.cmd-search-trigger {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		background: rgba(15, 23, 42, 0.6);
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: 8px;
		padding: 0.4rem 0.75rem;
		color: #94a3b8;
		font-size: 0.82rem;
		cursor: pointer;
		transition: all 0.15s ease;
		flex: 1;
		max-width: 280px;
	}

	.cmd-search-trigger:hover {
		background: rgba(30, 41, 59, 0.8);
		border-color: rgba(99, 102, 241, 0.4);
		color: #f1f5f9;
	}

	.search-lens {
		font-size: 0.85rem;
		opacity: 0.7;
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

	/* Unified Workspace Dropdown Button */
	.workspace-dropdown-container {
		position: relative;
	}

	.workspace-trigger-btn {
		display: flex;
		align-items: center;
		gap: 0.45rem;
		background: rgba(255, 255, 255, 0.04);
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: 8px;
		padding: 0.4rem 0.75rem;
		color: #f1f5f9;
		cursor: pointer;
		transition: all 0.15s ease;
		font-size: 0.82rem;
		font-weight: 600;
	}

	.workspace-trigger-btn:hover,
	.workspace-trigger-btn.active {
		background: rgba(99, 102, 241, 0.15);
		border-color: rgba(99, 102, 241, 0.4);
		color: #ffffff;
	}

	.ws-icon {
		font-size: 0.95rem;
	}

	.ws-title {
		white-space: nowrap;
	}

	.ws-tag {
		font-size: 0.68rem;
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 0.04em;
		padding: 0.1rem 0.4rem;
		border-radius: 4px;
		background: rgba(99, 102, 241, 0.2);
		color: #a5b4fc;
		border: 1px solid rgba(99, 102, 241, 0.3);
		white-space: nowrap;
	}

	.ws-chevron {
		width: 14px;
		height: 14px;
		color: #94a3b8;
		transition: transform 0.2s ease;
	}

	.ws-chevron.rotate {
		transform: rotate(180deg);
	}

	/* Menu Backdrop */
	.menu-backdrop {
		position: fixed;
		inset: 0;
		z-index: 99;
		background: transparent;
		border: none;
		cursor: default;
		padding: 0;
		margin: 0;
	}

	/* Workspace Popover */
	.workspace-popover {
		position: absolute;
		top: calc(100% + 8px);
		left: 50%;
		transform: translateX(-50%);
		width: 380px;
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.12);
		border-radius: 12px;
		box-shadow: 0 20px 40px rgba(0, 0, 0, 0.6), 0 0 20px rgba(99, 102, 241, 0.15);
		z-index: 100;
		overflow: hidden;
		animation: popoverFadeIn 0.15s ease-out;
	}

	@keyframes popoverFadeIn {
		from { opacity: 0; transform: translate(-50%, -6px); }
		to { opacity: 1; transform: translate(-50%, 0); }
	}

	.popover-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 0.75rem 1rem 0.5rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.06);
	}

	.popover-title {
		font-size: 0.68rem;
		font-weight: 700;
		color: #94a3b8;
		letter-spacing: 0.05em;
	}

	.popover-count {
		font-size: 0.68rem;
		color: #64748b;
	}

	.popover-list {
		max-height: 340px;
		overflow-y: auto;
		padding: 0.4rem;
		display: flex;
		flex-direction: column;
		gap: 0.2rem;
	}

	.workspace-item {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		padding: 0.55rem 0.75rem;
		background: transparent;
		border: 1px solid transparent;
		border-radius: 8px;
		color: #cbd5e1;
		cursor: pointer;
		text-align: left;
		transition: all 0.12s ease;
		width: 100%;
	}

	.workspace-item:hover {
		background: rgba(255, 255, 255, 0.05);
		color: #ffffff;
	}

	.workspace-item.selected {
		background: rgba(99, 102, 241, 0.15);
		border-color: rgba(99, 102, 241, 0.35);
		color: #ffffff;
	}

	.item-icon {
		font-size: 1.25rem;
		width: 2rem;
		height: 2rem;
		display: flex;
		align-items: center;
		justify-content: center;
		background: rgba(255, 255, 255, 0.05);
		border-radius: 6px;
		flex-shrink: 0;
	}

	.item-info {
		flex: 1;
		min-width: 0;
	}

	.item-head {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		margin-bottom: 0.15rem;
	}

	.item-name {
		font-size: 0.82rem;
		font-weight: 600;
		color: #f1f5f9;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.item-badge {
		font-size: 0.65rem;
		padding: 0.1rem 0.35rem;
		border-radius: 4px;
		background: rgba(255, 255, 255, 0.08);
		color: #94a3b8;
	}

	.item-desc {
		font-size: 0.7rem;
		color: #64748b;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.active-check {
		font-weight: 700;
		color: #818cf8;
		font-size: 0.95rem;
	}

	.popover-footer {
		padding: 0.5rem;
		border-top: 1px solid rgba(255, 255, 255, 0.06);
		background: rgba(0, 0, 0, 0.2);
	}

	.studio-launch-btn {
		width: 100%;
		padding: 0.45rem 0.75rem;
		background: transparent;
		border: 1px dashed rgba(99, 102, 241, 0.4);
		border-radius: 6px;
		color: #a5b4fc;
		font-size: 0.75rem;
		font-weight: 600;
		cursor: pointer;
		transition: all 0.15s ease;
		text-align: center;
	}

	.studio-launch-btn:hover {
		background: rgba(99, 102, 241, 0.15);
		color: #ffffff;
	}

	/* Right Utility Strip */
	.topbar-right {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		flex-shrink: 0;
	}

	.telemetry-chip {
		display: flex;
		align-items: center;
		gap: 0.35rem;
		font-size: 0.72rem;
		font-weight: 600;
		color: #34d399;
		background: rgba(16, 185, 129, 0.1);
		border: 1px solid rgba(16, 185, 129, 0.2);
		padding: 0.25rem 0.55rem;
		border-radius: 9999px;
	}

	.pulse-dot {
		width: 6px;
		height: 6px;
		border-radius: 50%;
		background: #34d399;
		animation: pulse-dot 1.8s infinite;
	}

	@keyframes pulse-dot {
		0%, 100% { opacity: 1; transform: scale(1); }
		50% { opacity: 0.4; transform: scale(0.85); }
	}

	.action-create-btn {
		font-size: 0.8rem;
		padding: 0.4rem 0.85rem;
		white-space: nowrap;
	}

	/* Profile Chip */
	.profile-dropdown-container {
		position: relative;
	}

	.profile-chip-btn {
		display: flex;
		align-items: center;
		gap: 0.45rem;
		background: rgba(255, 255, 255, 0.04);
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: 9999px;
		padding: 0.2rem 0.55rem 0.2rem 0.2rem;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.profile-chip-btn:hover,
	.profile-chip-btn.active {
		background: rgba(255, 255, 255, 0.08);
		border-color: rgba(99, 102, 241, 0.4);
	}

	.avatar-badge {
		width: 26px;
		height: 26px;
		border-radius: 50%;
		background: linear-gradient(135deg, #6366f1, #8b5cf6);
		color: #ffffff;
		font-size: 0.72rem;
		font-weight: 700;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.user-role-text {
		font-size: 0.75rem;
		font-weight: 600;
		color: #cbd5e1;
		text-transform: capitalize;
	}

	.chevron-mini {
		width: 12px;
		height: 12px;
		color: #94a3b8;
		transition: transform 0.2s ease;
	}

	.chevron-mini.rotate {
		transform: rotate(180deg);
	}

	/* Profile Popover Menu */
	.profile-popover {
		position: absolute;
		top: calc(100% + 8px);
		right: 0;
		width: 260px;
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.12);
		border-radius: 12px;
		box-shadow: 0 20px 40px rgba(0, 0, 0, 0.6);
		z-index: 100;
		overflow: hidden;
		animation: popoverFadeInRight 0.15s ease-out;
	}

	@keyframes popoverFadeInRight {
		from { opacity: 0; transform: translateY(-6px); }
		to { opacity: 1; transform: translateY(0); }
	}

	.profile-meta-card {
		padding: 0.85rem 1rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.06);
		background: rgba(255, 255, 255, 0.02);
	}

	.meta-fullname {
		font-size: 0.85rem;
		font-weight: 700;
		color: #f1f5f9;
	}

	.meta-email {
		font-size: 0.72rem;
		color: #94a3b8;
		margin: 0.1rem 0 0.35rem;
	}

	.meta-tenant {
		font-size: 0.7rem;
		color: #818cf8;
	}

	.profile-links {
		padding: 0.4rem;
		display: flex;
		flex-direction: column;
		gap: 0.15rem;
	}

	.profile-link {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.5rem 0.75rem;
		background: transparent;
		border: none;
		border-radius: 6px;
		color: #cbd5e1;
		font-size: 0.8rem;
		font-weight: 500;
		text-align: left;
		text-decoration: none;
		cursor: pointer;
		width: 100%;
		transition: all 0.12s ease;
	}

	.profile-link:hover {
		background: rgba(255, 255, 255, 0.06);
		color: #ffffff;
	}

	.menu-divider {
		height: 1px;
		background: rgba(255, 255, 255, 0.06);
		margin: 0.25rem 0;
	}

	.text-danger {
		color: #fb7185;
	}

	.text-danger:hover {
		background: rgba(239, 68, 68, 0.15);
		color: #fca5a5;
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
