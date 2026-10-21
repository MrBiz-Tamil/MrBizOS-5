<script lang="ts">
	import { onMount } from 'svelte';

	interface Props {
		open: boolean;
		onclose: () => void;
		onapply: (preset: string, customConfig?: CustomWorkspaceConfig) => void;
	}

	let { open = false, onclose, onapply }: Props = $props();

	export interface WorkspaceModule {
		id: string;
		label: string;
		icon: string;
		href: string;
		category: 'CORE' | 'VERTICAL' | 'AI_GROWTH' | 'OPERATIONS';
		description: string;
		enabled: boolean;
	}

	export interface CustomWorkspaceConfig {
		name: string;
		icon: string;
		badgeText: string;
		homeRoute: string;
		primaryAction: {
			label: string;
			icon: string;
			href: string;
		};
		modules: string[]; // List of enabled module IDs
	}

	let activeModalTab = $state<'presets' | 'customizer'>('presets');

	// All Available Industry Modules
	const allModules: WorkspaceModule[] = [
		{ id: 'billing', label: 'Express Billing & Invoices', icon: '⚡', href: '/billing', category: 'CORE', description: '3-second quick billing, UPI QR, and 17 templates', enabled: true },
		{ id: 'pos', label: 'Retail POS Terminal', icon: '🛒', href: '/pos', category: 'CORE', description: 'Barcode checkout and daily cash drawer tally', enabled: true },
		{ id: 'healthcare', label: 'Smart Healthcare & EMR', icon: '🏥', href: '/healthcare', category: 'VERTICAL', description: 'Doctor Rx pad, pharmacy dispenser, nurse triage & owner stats', enabled: true },
		{ id: 'travel', label: 'Travel & Tour Agency OS', icon: '✈️', href: '/travel', category: 'VERTICAL', description: 'Live tour deck, PNR holds, DMC vouchers & VFS tracking', enabled: true },
		{ id: 'realestate', label: 'Real Estate & Plot Matrix', icon: '🏢', href: '/realestate', category: 'VERTICAL', description: '24-plot visual layout, Indian land converter & cab visits', enabled: true },
		{ id: 'ecommerce', label: 'E-Commerce & Print-on-Demand', icon: '🛍️', href: '/ecommerce', category: 'VERTICAL', description: 'Blank inventory matrix, print lab queue & courier NDR', enabled: true },
		{ id: 'projects', label: 'Projects & UTMS Tasks', icon: '📁', href: '/projects', category: 'OPERATIONS', description: 'Universal Task Management with SLAs & cross-BU delegation', enabled: true },
		{ id: 'inbox', label: 'Omnichannel Social Inbox', icon: '💬', href: '/inbox', category: 'AI_GROWTH', description: 'Unified WhatsApp, Instagram DM, Facebook & Email', enabled: true },
		{ id: 'bpo-voice', label: 'AI Voice BPO Call Center', icon: '🎙️', href: '/bpo-voice', category: 'AI_GROWTH', description: 'Autonomous voice bot with live audio warm transfer', enabled: true },
		{ id: 'accounting', label: 'Financial Accounting & Ledger', icon: '📖', href: '/accounting', category: 'CORE', description: 'General ledger, bank reconciliation & trial balance', enabled: true },
		{ id: 'procurement', label: '3-Way Procurement', icon: '📦', href: '/procurement', category: 'OPERATIONS', description: 'PO matching, GRN receipt and vendor invoice settlement', enabled: true },
		{ id: 'hrms', label: 'HRMS & Automated Payroll', icon: '👥', href: '/hrms', category: 'OPERATIONS', description: 'Employee biometric attendance, salary slips & PF/ESI', enabled: true },
		{ id: 'trade', label: 'Export & Import Trade OS', icon: '🚢', href: '/trade/shipments', category: 'OPERATIONS', description: 'Customs paperwork, container tracking & foreign exchange', enabled: true },
		{ id: 'support', label: 'Support Helpdesk & SLAs', icon: '🎫', href: '/support', category: 'AI_GROWTH', description: 'Customer ticket dispatch with automated SLA breach alarms', enabled: true },
		{ id: 'journeys', label: 'Client Journey Engine', icon: '🛤️', href: '/journeys', category: 'AI_GROWTH', description: 'Visual pipeline tracking client lifecycle deliverables', enabled: true },
		{ id: 'settings', label: 'Company Profile & Settings', icon: '⚙️', href: '/settings', category: 'CORE', description: 'GSTIN, banking setup, signatures and legal terms', enabled: true }
	];

	// Pre-configured Industry Works Presets
	const presets = [
		{
			id: 'HEALTHCARE',
			name: 'Smart Healthcare & Hospital OS',
			icon: '🏥',
			badge: 'Clinical Workstations',
			badgeColor: '#38bdf8',
			description: 'Optimized for clinics, nursing homes, and hospitals. Features 4 personas: Doctor Rx Pad, Pharmacy POS, Nurse Triage, and Director P&L.',
			modules: ['healthcare', 'pos', 'billing', 'accounting', 'inbox', 'hrms', 'settings'],
			homeRoute: '/healthcare',
			primaryAction: { label: '+ New Clinical Rx', icon: '📝', href: '/healthcare' }
		},
		{
			id: 'TRAVEL',
			name: 'Travel & Tour Agency Operations',
			icon: '✈️',
			badge: 'GDS & DMC Hub',
			badgeColor: '#06b6d4',
			description: 'Dedicated to holiday operators, DMC agents, and flight desks. Tracks PNR hold clocks, hotel vouchers, and WhatsApp client packs.',
			modules: ['travel', 'inbox', 'billing', 'bpo-voice', 'accounting', 'journeys', 'settings'],
			homeRoute: '/travel',
			primaryAction: { label: '+ Hold Airline PNR', icon: '🎫', href: '/travel' }
		},
		{
			id: 'REALESTATE',
			name: 'Real Estate & Layout Development',
			icon: '🏢',
			badge: 'Plots & RERA',
			badgeColor: '#10b981',
			description: 'Built for plot promoters, builders, and field closers. Includes 24-plot grid, Indian land unit converter, and cab site visit scheduler.',
			modules: ['realestate', 'bpo-voice', 'inbox', 'billing', 'accounting', 'projects', 'settings'],
			homeRoute: '/realestate',
			primaryAction: { label: '+ Book Site Visit', icon: '🚗', href: '/realestate' }
		},
		{
			id: 'COUNTER',
			name: 'Express Retail Counter & POS',
			icon: '⚡',
			badge: '3-Sec Speed',
			badgeColor: '#fbbf24',
			description: 'Ultra-lean, high-speed retail checkout for supermarkets, stores, and counters. Barcode scanning, dynamic UPI QR, and thermal printing.',
			modules: ['billing', 'pos', 'accounting', 'settings'],
			homeRoute: '/billing',
			primaryAction: { label: '+ New Express Bill', icon: '⚡', href: '/billing' }
		},
		{
			id: 'PROJECTS',
			name: 'Projects & UTMS Delivery OS',
			icon: '📁',
			badge: 'SLA Engine',
			badgeColor: '#818cf8',
			description: 'Geared for software agencies, engineering teams, and consultancies. Kanban boards, cross-BU task delegation, and automated billing triggers.',
			modules: ['projects', 'hrms', 'accounting', 'inbox', 'support', 'billing', 'settings'],
			homeRoute: '/projects',
			primaryAction: { label: '+ New UTMS Task', icon: '⚡', href: '/projects' }
		},
		{
			id: 'COMMERCE',
			name: 'E-Commerce & Print-on-Demand',
			icon: '🛍️',
			badge: 'Apparel & Labs',
			badgeColor: '#ec4899',
			description: 'For apparel merchants and lab operators. Matrix inventory (Color/Size), print queue manager, and courier cash reconciliation.',
			modules: ['ecommerce', 'procurement', 'pos', 'billing', 'inbox', 'accounting', 'settings'],
			homeRoute: '/ecommerce',
			primaryAction: { label: '+ New Print Order', icon: '🖨️', href: '/ecommerce' }
		},
		{
			id: 'FULL',
			name: 'Universal Enterprise ERP (All Domains)',
			icon: '🌐',
			badge: 'Unrestricted',
			badgeColor: '#a855f7',
			description: 'Full unconstrained platform with all 16 business modules, cross-department workflows, and multi-tenant ledger.',
			modules: allModules.map(m => m.id),
			homeRoute: '/',
			primaryAction: { label: '+ Universal Action', icon: '⚡', href: '/services' }
		}
	];

	// Customizer Form State
	let customName = $state('My Custom Operations');
	let customIcon = $state('💼');
	let customBadge = $state('Custom Desk');
	let customHome = $state('/services');
	let customEnabledModules = $state<string[]>(['billing', 'pos', 'projects', 'inbox', 'settings']);
	let customPrimaryActionLabel = $state('+ Quick Action');
	let customPrimaryActionHref = $state('/billing');

	onMount(() => {
		if (typeof localStorage !== 'undefined') {
			const savedCustom = localStorage.getItem('mrbiz_custom_workspace_v2');
			if (savedCustom) {
				try {
					const cfg: CustomWorkspaceConfig = JSON.parse(savedCustom);
					customName = cfg.name || customName;
					customIcon = cfg.icon || customIcon;
					customBadge = cfg.badgeText || customBadge;
					customHome = cfg.homeRoute || customHome;
					customEnabledModules = cfg.modules || customEnabledModules;
					if (cfg.primaryAction) {
						customPrimaryActionLabel = cfg.primaryAction.label || customPrimaryActionLabel;
						customPrimaryActionHref = cfg.primaryAction.href || customPrimaryActionHref;
					}
				} catch (e) {}
			}
		}
	});

	function toggleModule(id: string) {
		if (customEnabledModules.includes(id)) {
			if (customEnabledModules.length > 1) {
				customEnabledModules = customEnabledModules.filter(m => m !== id);
			}
		} else {
			customEnabledModules = [...customEnabledModules, id];
		}
	}

	function handleApplyPreset(preset: typeof presets[0]) {
		const cfg: CustomWorkspaceConfig = {
			name: preset.name,
			icon: preset.icon,
			badgeText: preset.badge,
			homeRoute: preset.homeRoute,
			primaryAction: preset.primaryAction,
			modules: preset.modules
		};
		if (typeof localStorage !== 'undefined') {
			localStorage.setItem('mrbiz_workspace_preset', preset.id);
			localStorage.setItem('mrbiz_custom_workspace_v2', JSON.stringify(cfg));
		}
		onapply(preset.id, cfg);
		onclose();
	}

	function handleSaveCustom() {
		const cfg: CustomWorkspaceConfig = {
			name: customName.trim() || 'My Custom Desk',
			icon: customIcon,
			badgeText: customBadge.trim() || 'Custom',
			homeRoute: customHome,
			primaryAction: {
				label: customPrimaryActionLabel.trim() || '+ Quick Action',
				icon: '⚡',
				href: customPrimaryActionHref
			},
			modules: customEnabledModules
		};
		if (typeof localStorage !== 'undefined') {
			localStorage.setItem('mrbiz_workspace_preset', 'CUSTOM');
			localStorage.setItem('mrbiz_custom_workspace_v2', JSON.stringify(cfg));
		}
		onapply('CUSTOM', cfg);
		onclose();
	}
</script>

{#if open}
	<div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-md">
		<div class="bg-slate-900 border border-slate-700/80 rounded-2xl w-full max-w-4xl max-h-[90vh] flex flex-col shadow-2xl overflow-hidden animate-scale-up">
			
			<!-- Modal Header -->
			<div class="p-5 border-b border-slate-800 flex justify-between items-center bg-slate-950/60">
				<div>
					<div class="flex items-center gap-2">
						<span class="text-2xl">🛠️</span>
						<h2 class="text-xl font-extrabold text-white">Dynamic Workspace Customizer</h2>
						<span class="px-2.5 py-0.5 rounded-full text-xs font-bold bg-indigo-500/10 text-indigo-400 border border-indigo-500/20">
							Works-Based Engine
						</span>
					</div>
					<p class="text-xs text-slate-400 mt-1">
						Tailor MrBiz-OS specifically for your day-to-day operations. Hide unnecessary modules and pin your critical business tools.
					</p>
				</div>
				<button class="w-8 h-8 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 flex items-center justify-center cursor-pointer transition-colors" onclick={onclose}>
					✕
				</button>
			</div>

			<!-- Tab Navigation -->
			<div class="px-5 pt-3 border-b border-slate-800 bg-slate-950/30 flex gap-3">
				<button 
					type="button"
					class="px-4 py-2 text-sm font-bold border-b-2 transition-all cursor-pointer flex items-center gap-2"
					style={activeModalTab === 'presets' ? 'border-color: #6366f1; color: #818cf8;' : 'border-color: transparent; color: #94a3b8;'}
					onclick={() => activeModalTab = 'presets'}>
					<span>🏢 Industry Works Presets</span>
					<span class="px-2 py-0.2 rounded-full text-[10px] bg-slate-800 text-slate-300 font-mono">7 Ready</span>
				</button>
				<button 
					type="button"
					class="px-4 py-2 text-sm font-bold border-b-2 transition-all cursor-pointer flex items-center gap-2"
					style={activeModalTab === 'customizer' ? 'border-color: #6366f1; color: #818cf8;' : 'border-color: transparent; color: #94a3b8;'}
					onclick={() => activeModalTab = 'customizer'}>
					<span>🎨 Custom Workspace Builder</span>
					<span class="px-2 py-0.2 rounded-full text-[10px] bg-indigo-500/20 text-indigo-300 font-mono">{customEnabledModules.length} Active</span>
				</button>
			</div>

			<!-- Modal Body (Scrollable) -->
			<div class="flex-1 overflow-y-auto p-6 space-y-6">
				
				<!-- TAB 1: INDUSTRY PRESETS -->
				{#if activeModalTab === 'presets'}
					<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
						{#each presets as p}
							<div class="p-4 rounded-xl border border-slate-800 hover:border-indigo-500/50 bg-slate-950/60 transition-all flex flex-col justify-between group shadow-sm">
								<div>
									<div class="flex justify-between items-start mb-2">
										<div class="flex items-center gap-2">
											<span class="text-2xl p-2 rounded-lg bg-slate-900 border border-slate-800">{p.icon}</span>
											<div>
												<h3 class="text-sm font-bold text-white group-hover:text-indigo-400 transition-colors">{p.name}</h3>
												<span class="text-[10px] font-bold px-2 py-0.5 rounded-full uppercase tracking-wider" style={`background: ${p.badgeColor}15; color: ${p.badgeColor}; border: 1px solid ${p.badgeColor}30;`}>
													{p.badge}
												</span>
											</div>
										</div>
									</div>
									<p class="text-xs text-slate-400 leading-relaxed mt-2 mb-3">
										{p.description}
									</p>
									<div class="flex flex-wrap gap-1.5 mb-4">
										{#each p.modules as mid}
											{@const mod = allModules.find(m => m.id === mid)}
											{#if mod}
												<span class="text-[10px] px-2 py-0.5 rounded bg-slate-900 border border-slate-800 text-slate-300 font-mono">
													{mod.icon} {mod.label.split(' ')[0]}
												</span>
											{/if}
										{/each}
									</div>
								</div>

								<button 
									class="w-full py-2 px-3 rounded-lg bg-indigo-600/20 hover:bg-indigo-600 text-indigo-300 hover:text-white border border-indigo-500/40 text-xs font-bold transition-all cursor-pointer flex items-center justify-center gap-1.5"
									onclick={() => handleApplyPreset(p)}>
									<span>⚡ Switch to {p.name.split(' ')[0]} Workspace</span>
								</button>
							</div>
						{/each}
					</div>

				<!-- TAB 2: CUSTOM WORKSPACE BUILDER -->
				{:else if activeModalTab === 'customizer'}
					<div class="space-y-6">
						<!-- Workspace Metadata -->
						<div class="grid grid-cols-1 md:grid-cols-3 gap-4 p-4 rounded-xl bg-slate-950/60 border border-slate-800">
							<div>
								<label class="block text-xs font-bold text-slate-300 mb-1" for="cw-name">Workspace Name</label>
								<input id="cw-name" type="text" class="w-full px-3 py-1.5 rounded-lg bg-slate-900 border border-slate-700 text-sm text-white focus:outline-none focus:border-indigo-500" bind:value={customName} />
							</div>
							<div>
								<label class="block text-xs font-bold text-slate-300 mb-1" for="cw-icon">Icon / Emoji</label>
								<div class="flex gap-2">
									<input id="cw-icon" type="text" class="w-16 px-3 py-1.5 rounded-lg bg-slate-900 border border-slate-700 text-sm text-center text-white" bind:value={customIcon} />
									<div class="flex gap-1 items-center">
										{#each ['🏥', '✈️', '🏢', '⚡', '📁', '🛍️', '💼', '🚢'] as emoji}
											<button type="button" class="px-1.5 py-1 text-sm hover:scale-125 transition-transform" onclick={() => customIcon = emoji}>{emoji}</button>
										{/each}
									</div>
								</div>
							</div>
							<div>
								<label class="block text-xs font-bold text-slate-300 mb-1" for="cw-home">Default Home Route</label>
								<select id="cw-home" class="w-full px-3 py-1.5 rounded-lg bg-slate-900 border border-slate-700 text-sm text-white focus:outline-none focus:border-indigo-500" bind:value={customHome}>
									<option value="/">Command Center Dashboard (/)</option>
									<option value="/healthcare">Healthcare Hub (/healthcare)</option>
									<option value="/travel">Travel &amp; Tours OS (/travel)</option>
									<option value="/realestate">Real Estate Deals (/realestate)</option>
									<option value="/billing">Express Billing Counter (/billing)</option>
									<option value="/projects">Projects &amp; UTMS (/projects)</option>
									<option value="/inbox">Omnichannel Inbox (/inbox)</option>
									<option value="/ecommerce">E-Commerce POD (/ecommerce)</option>
								</select>
							</div>
						</div>

						<!-- Module Visibility Checklist -->
						<div>
							<div class="flex justify-between items-center mb-3">
								<div>
									<h3 class="text-sm font-bold text-white">Select Visible Modules ({customEnabledModules.length} of {allModules.length} Enabled)</h3>
									<p class="text-xs text-slate-400">Checked modules will appear directly in your sidebar navigation.</p>
								</div>
								<div class="flex gap-2">
									<button type="button" class="text-xs px-2.5 py-1 rounded bg-slate-800 text-slate-300 hover:text-white" onclick={() => customEnabledModules = allModules.map(m => m.id)}>
										Select All
									</button>
									<button type="button" class="text-xs px-2.5 py-1 rounded bg-slate-800 text-slate-300 hover:text-white" onclick={() => customEnabledModules = ['billing', 'pos', 'settings']}>
										Minimal (Core Only)
									</button>
								</div>
							</div>

							<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
								{#each allModules as mod}
									{@const isSelected = customEnabledModules.includes(mod.id)}
									<button 
										type="button"
										class="p-3 rounded-xl border text-left transition-all cursor-pointer flex items-start gap-3"
										style={isSelected ? 'background: rgba(99, 102, 241, 0.12); border-color: rgba(99, 102, 241, 0.5);' : 'background: rgba(15, 23, 42, 0.5); border-color: rgba(255, 255, 255, 0.08); opacity: 0.6;'}
										onclick={() => toggleModule(mod.id)}>
										<input type="checkbox" checked={isSelected} class="mt-1 rounded accent-indigo-500 cursor-pointer" />
										<div class="flex-1 min-w-0">
											<div class="flex items-center gap-1.5">
												<span class="text-base">{mod.icon}</span>
												<span class="text-xs font-bold text-white truncate">{mod.label}</span>
											</div>
											<p class="text-[10px] text-slate-400 mt-0.5 line-clamp-1">{mod.description}</p>
										</div>
									</button>
								{/each}
							</div>
						</div>

						<!-- Primary Quick Action -->
						<div class="p-4 rounded-xl bg-slate-950/60 border border-slate-800 space-y-3">
							<h4 class="text-xs font-bold text-slate-300 uppercase">Top Navigation Quick-Action Button</h4>
							<div class="grid grid-cols-1 md:grid-cols-2 gap-3">
								<div>
									<label class="block text-xs text-slate-400 mb-1" for="btn-lbl">Button Label</label>
									<input id="btn-lbl" type="text" class="w-full px-3 py-1.5 rounded-lg bg-slate-900 border border-slate-700 text-xs text-white" bind:value={customPrimaryActionLabel} />
								</div>
								<div>
									<label class="block text-xs text-slate-400 mb-1" for="btn-href">Destination Link</label>
									<select id="btn-href" class="w-full px-3 py-1.5 rounded-lg bg-slate-900 border border-slate-700 text-xs text-white" bind:value={customPrimaryActionHref}>
										<option value="/billing">+ Express Counter Bill (/billing)</option>
										<option value="/invoices/new">+ Full Tax Invoice (/invoices/new)</option>
										<option value="/healthcare">+ Clinical Rx Pad (/healthcare)</option>
										<option value="/travel">+ Travel PNR Hold (/travel)</option>
										<option value="/realestate">+ Site Visit Cab (/realestate)</option>
										<option value="/projects">+ UTMS Task (/projects)</option>
										<option value="/inbox">+ Omnichannel Chat (/inbox)</option>
									</select>
								</div>
							</div>
						</div>
					</div>
				{/if}
			</div>

			<!-- Modal Footer -->
			<div class="p-4 border-t border-slate-800 bg-slate-950/80 flex justify-between items-center">
				<div class="text-xs text-slate-400">
					Active Configuration: <strong class="text-white">{customEnabledModules.length} Modules Enabled</strong>
				</div>
				<div class="flex gap-2">
					<button class="px-4 py-2 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs font-bold cursor-pointer transition-colors" onclick={onclose}>
						Cancel
					</button>
					{#if activeModalTab === 'customizer'}
						<button class="px-5 py-2 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-bold shadow-lg shadow-indigo-500/25 cursor-pointer transition-all flex items-center gap-1.5" onclick={handleSaveCustom}>
							<span>💾 Save &amp; Activate Custom Workspace</span>
						</button>
					{/if}
				</div>
			</div>

		</div>
	</div>
{/if}

<style>
	@keyframes scaleUp {
		from { transform: scale(0.96); opacity: 0; }
		to { transform: scale(1); opacity: 1; }
	}
	.animate-scale-up {
		animation: scaleUp 0.18s cubic-bezier(0.16, 1, 0.3, 1) forwards;
	}
</style>
