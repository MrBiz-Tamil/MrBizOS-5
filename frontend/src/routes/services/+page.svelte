<script lang="ts">
	type Category = 'ALL' | 'FINANCIAL' | 'AI_GROWTH' | 'SUPPLY_CHAIN' | 'VERTICALS';

	let activeCategory = $state<Category>('ALL');
	let searchQuery = $state('');

	interface ServiceItem {
		id: string;
		title: string;
		category: Category;
		categoryLabel: string;
		icon: string;
		badge: string;
		badgeColor: string;
		href: string;
		summary: string;
		features: string[];
		stats: string;
	}

	const services: ServiceItem[] = [
		// 1. Standalone Fast Billing
		{
			id: 'billing',
			title: 'Express Billing & 17 Templates',
			category: 'FINANCIAL',
			categoryLabel: 'Financial & POS',
			icon: '⚡',
			badge: 'Ultra Fast',
			badgeColor: '#38bdf8',
			href: '/billing',
			summary: '3-second express retail counter, live dynamic UPI QR code, 80mm thermal receipt printing, and 17 specialized industry templates.',
			features: ['3-Second Counter Sale', 'Dynamic UPI QR Generation', '17 Industry Invoice Styles', 'WhatsApp 1-Click Bill Send'],
			stats: '0.04s Generation'
		},
		// 2. Retail POS Terminal
		{
			id: 'pos',
			title: 'Retail POS Terminal',
			category: 'FINANCIAL',
			categoryLabel: 'Financial & POS',
			icon: '🛒',
			badge: 'Offline-Ready',
			badgeColor: '#10b981',
			href: '/pos',
			summary: 'High-speed touch and barcode scanner point-of-sale for retail counters, supermarkets, pharmacies, and walk-in trade.',
			features: ['Split Payments (Cash/Card/UPI)', 'Barcode Scanner Integration', 'Thermal Receipt Auto-Cut', 'Daily Cash Drawer Tally'],
			stats: 'Instant Barcode Scan'
		},
		// 3. Omnichannel Social Inbox
		{
			id: 'inbox',
			title: 'Omnichannel Social Inbox',
			category: 'AI_GROWTH',
			categoryLabel: 'AI & Growth',
			icon: '💬',
			badge: 'Meta & WhatsApp',
			badgeColor: '#f43f5e',
			href: '/inbox',
			summary: 'Unified communication hub integrating WhatsApp Business, Instagram DM, Facebook Messenger, and Email into a single inbox.',
			features: ['Multi-Channel Unification', 'AI Suggestion & Auto-Reply', '1-Click Convert to Lead/Invoice', 'Customer Journey Tracking'],
			stats: 'Real-time WebSocket'
		},
		// 4. AI Voice Agent BPO Desk
		{
			id: 'bpo-voice',
			title: 'AI Voice Agent BPO Call Center',
			category: 'AI_GROWTH',
			categoryLabel: 'AI & Growth',
			icon: '🎙️',
			badge: 'Sub-600ms Voice',
			badgeColor: '#06b6d4',
			href: '/bpo-voice',
			summary: 'Autonomous AI phone agent for 24/7 customer support, inbound inquiries, order placement, and seamless human warm handoff.',
			features: ['Sub-600ms Voice Latency', 'Live Transcript & Sentiment', 'Warm Human Handoff Desk', 'Automated Outbound Campaigns'],
			stats: '24/7 Zero Wait Time'
		},
		// 5. Support Helpdesk & SLAs
		{
			id: 'support',
			title: 'Support Helpdesk & SLAs',
			category: 'AI_GROWTH',
			categoryLabel: 'AI & Growth',
			icon: '🎫',
			badge: 'Enterprise SLA',
			badgeColor: '#8b5cf6',
			href: '/support',
			summary: 'Enterprise ticket routing, priority escalations, customer satisfaction CSAT ratings, and strict SLA breach timers.',
			features: ['Automated Ticket Assignment', 'SLA Response Tracking', 'Staff Performance Metrics', 'Customer CSAT Feedback'],
			stats: 'Zero Breached Tickets'
		},
		// 6. Projects & UTMS Tasks
		{
			id: 'projects',
			title: 'Projects & UTMS Tasks',
			category: 'AI_GROWTH',
			categoryLabel: 'AI & Growth',
			icon: '📁',
			badge: 'Milestone Sync',
			badgeColor: '#ec4899',
			href: '/projects',
			summary: 'Universal Task Management System (UTMS) with Kanban boards, milestones, timesheets, and auto-conversion to billing.',
			features: ['Kanban Task Lifecycle', 'Employee Timesheet Logging', 'Milestone SOW Tracking', 'Auto-Invoice from Time Spent'],
			stats: '100% Billable Accuracy'
		},
		// 7. Journey Lifecycle Engine
		{
			id: 'journeys',
			title: 'Journey Lifecycle Engine',
			category: 'AI_GROWTH',
			categoryLabel: 'AI & Growth',
			icon: '🛤️',
			badge: 'Automated Funnel',
			badgeColor: '#f59e0b',
			href: '/journeys',
			summary: 'Step-by-step pipeline engine for customer onboarding, loan applications, patient checkups, and post-sale nurturing.',
			features: ['Configurable Visual Stages', 'Stage-Gate Action Triggers', 'Automated Reminders', 'Multi-Vertical Workflows'],
			stats: 'End-to-End Tracking'
		},
		// 8. Procurement & 3-Way Match
		{
			id: 'procurement',
			title: 'Procurement & 3-Way Match',
			category: 'SUPPLY_CHAIN',
			categoryLabel: 'Supply Chain & Trade',
			icon: '📦',
			badge: 'Audit Compliant',
			badgeColor: '#a855f7',
			href: '/procurement',
			summary: 'Strict fraud prevention matching Purchase Orders (PO), Goods Received Notes (GRN), and Vendor Invoices before release of payment.',
			features: ['3-Way Discrepancy Detection', 'Purchase Order Approval Chain', 'Supplier Performance Ledger', 'Inventory Low-Stock Alerts'],
			stats: 'Zero Payment Leakage'
		},
		// 9. Bank Auto-Reconciliation
		{
			id: 'reconciliation',
			title: 'Bank Auto-Reconciliation',
			category: 'FINANCIAL',
			categoryLabel: 'Financial & POS',
			icon: '🏦',
			badge: 'MT940 / CSV',
			badgeColor: '#3b82f6',
			href: '/accounting/reconciliation',
			summary: 'Automated bank feed parsing, statement reconciliation, algorithmic transaction matching, and ledger journal generation.',
			features: ['Instant CSV & Statement Parser', 'Fuzzy Rule-Based Matcher', 'One-Click Manual Match', 'Realtime Ledger Sync'],
			stats: '98% Auto-Match Rate'
		},
		// 10. HRMS & Automated Payroll
		{
			id: 'hrms',
			title: 'HRMS & 1-Click Payroll',
			category: 'FINANCIAL',
			categoryLabel: 'Financial & POS',
			icon: '👥',
			badge: 'Statutory Compliant',
			badgeColor: '#14b8a6',
			href: '/hrms',
			summary: 'Staff records, department assignments, salary structures (Basic, HRA, PF, TDS), and 1-click batch payroll execution.',
			features: ['1-Click Batch Salary Run', 'Automated Payslip PDF', 'Tax & Deduction Calculation', 'Employee Attendance Ledger'],
			stats: 'Instant Salary Run'
		},
		// 11. Smart Healthcare Hub
		{
			id: 'healthcare',
			title: 'Smart Healthcare & EMR Hub',
			category: 'VERTICALS',
			categoryLabel: 'Industry Verticals',
			icon: '🏥',
			badge: 'Healthcare OS',
			badgeColor: '#ef4444',
			href: '/healthcare',
			summary: 'Clinical appointments, doctor scheduling, electronic medical records (EMR), and digital Rx prescriptions.',
			features: ['Doctor Consultation Queues', 'Prescription Writer', 'Patient Medical History', 'Appointment Slots'],
			stats: 'Complete EMR Flow'
		},
		// 12. Travel & Visa OS
		{
			id: 'travel',
			title: 'Travel, Tours & Visa OS',
			category: 'VERTICALS',
			categoryLabel: 'Industry Verticals',
			icon: '✈️',
			badge: 'Tourism Hub',
			badgeColor: '#0ea5e9',
			href: '/travel',
			summary: 'Tour package catalogs, flight & hotel itinerary PDF generation, and multi-country visa stage tracking.',
			features: ['Holiday Package Quotes', 'Itinerary Voucher Builder', 'Visa Embassy Pipeline', 'Multi-Currency Payments'],
			stats: 'Instant Itinerary PDF'
		},
		// 13. Real Estate Deal Hub
		{
			id: 'realestate',
			title: 'Real Estate Deals & Escrow',
			category: 'VERTICALS',
			categoryLabel: 'Industry Verticals',
			icon: '🏢',
			badge: 'Property OS',
			badgeColor: '#d97706',
			href: '/realestate',
			summary: 'Property listings, agent commissions, customer site visits, escrow ledger, and sales agreement contracts.',
			features: ['Unit Inventory & Floor Plans', 'Agent Commission Tracking', 'Site Visit Scheduling', 'Deals & Escrow Khata'],
			stats: 'Full Deal Pipeline'
		},
		// 14. E-Commerce & Print-on-Demand
		{
			id: 'ecommerce',
			title: 'E-Commerce & Print-on-Demand',
			category: 'VERTICALS',
			categoryLabel: 'Industry Verticals',
			icon: '🛍️',
			badge: 'Etsy & Shopify Sync',
			badgeColor: '#84cc16',
			href: '/ecommerce',
			summary: 'Multi-store order routing, blank stock reservations, automated print lab dispatch, and fulfillment tracking.',
			features: ['Etsy & Shopify Webhooks', 'Automated Lab Order Routing', 'Blank Inventory Reservations', 'Carrier Tracking Sync'],
			stats: 'Sub-Minute Routing'
		},
		// 15. Software Consulting & SOW
		{
			id: 'consulting',
			title: 'Software Consulting & SOW',
			category: 'VERTICALS',
			categoryLabel: 'Industry Verticals',
			icon: '💻',
			badge: 'Professional Services',
			badgeColor: '#6366f1',
			href: '/consulting',
			summary: 'Client engagement contracts, Statement of Work (SOW) milestone billing, developer timesheets, and rate cards.',
			features: ['Client Project Agreements', 'Milestone Invoicing Triggers', 'Developer Billable Hours', 'Margin & Cost Analysis'],
			stats: 'Accurate Retainers'
		},
		// 16. Export / Import Trade OS
		{
			id: 'trade',
			title: 'Export & Import Trade OS',
			category: 'SUPPLY_CHAIN',
			categoryLabel: 'Supply Chain & Trade',
			icon: '🚢',
			badge: 'Customs & FX',
			badgeColor: '#e11d48',
			href: '/trade',
			summary: 'Commercial invoices, packing lists, sea/air cargo manifests, multi-currency conversions, and customs filing records.',
			features: ['Multi-Currency Invoicing', 'Container & B/L Tracking', 'Live Foreign Exchange (FX)', 'Packing List Generation'],
			stats: 'Customs Compliant'
		},
		// 17. Blue Metals & Quarry Weighbridge Billing
		{
			id: 'quarry-billing',
			title: 'Blue Metals & Quarry Weighbridge',
			category: 'VERTICALS',
			categoryLabel: 'Industry Verticals',
			icon: '🏗️',
			badge: 'Template 11 Active',
			badgeColor: '#0284c7',
			href: '/billing?mode=quarry',
			summary: 'Automated Gross-Tare-Net weighbridge tonnage billing, Lorry trip pass, mineral transit challan, and heavy boxed Indian quarry GST.',
			features: ['Gross & Tare Digital Weighbridge', '20mm/40mm/M-Sand Quick Palette', 'Lorry Trip Pass & Mineral Challan', 'Heavy Grid Template 11 Print'],
			stats: 'Weighbridge Instant DC'
		},
		// 18. Customer Travel & Concierge Services
		{
			id: 'travel-portal',
			title: 'Customer Travel & Concierge Services',
			category: 'VERTICALS',
			categoryLabel: 'Industry Verticals',
			icon: '🛎️',
			badge: 'Customer-Facing',
			badgeColor: '#0284c7',
			href: '/customer/services',
			summary: 'Public customer-side travel desk with live PNR tracking, VIP chauffeur airport meet, e-Visa expediting, and 24/7 WhatsApp concierge.',
			features: ['Live PNR Trip Tracker', 'Airport VIP Chauffeur Transfer', 'Express e-Visa Filing', '24/7 Travel Care SOS'],
			stats: 'Sub-15m Emergency SOS'
		},
		// 19. Customer Tour Booking Wizard
		{
			id: 'tour-booking',
			title: 'Customer Tour Booking & DMC Engine',
			category: 'VERTICALS',
			categoryLabel: 'Industry Verticals',
			icon: '🗺️',
			badge: 'Instant PNR',
			badgeColor: '#10b981',
			href: '/travel/booking',
			summary: 'Interactive consumer tour booking wizard with customizable circuits (Kashmir, Kerala, Dubai, Bali), hotel tiers, fleet selection, and instant vouchers.',
			features: ['Circuit & Package Customizer', '3★/4★/5★ Hotel Upgrades', 'Chauffeur Cab Selection', 'Instant Digital Voucher PDF'],
			stats: 'Zero Walkout Record'
		},
		// 20. Tour & Travel Partners Hub (B2B)
		{
			id: 'travel-partners',
			title: 'Tour & Travel Partners Hub (B2B)',
			category: 'VERTICALS',
			categoryLabel: 'Industry Verticals',
			icon: '🤝',
			badge: '12.5% Margin',
			badgeColor: '#34d399',
			href: '/travel/partners',
			summary: 'B2B affiliate network for 1,200+ travel agencies with wholesale DMC net tariffs, sub-agent credit khata, and instant partner registration.',
			features: ['B2B Wholesale Net Rates', '₹10L Khata Credit Facility', 'White-label Client Itineraries', 'Interactive Profit Calculator'],
			stats: '1,200+ Active Partners'
		}
	];

	let filteredServices = $derived(
		services.filter((s) => {
			const matchesCategory = activeCategory === 'ALL' || s.category === activeCategory;
			const matchesSearch =
				s.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
				s.summary.toLowerCase().includes(searchQuery.toLowerCase()) ||
				s.categoryLabel.toLowerCase().includes(searchQuery.toLowerCase()) ||
				s.features.some((f) => f.toLowerCase().includes(searchQuery.toLowerCase()));
			return matchesCategory && matchesSearch;
		})
	);
</script>

<svelte:head>
	<title>Enterprise Services Directory & Solutions - MrBiz-OS</title>
</svelte:head>

<div class="services-page">
	
	<!-- Hero Banner -->
	<div class="hero-section">
		<div class="hero-badge">
			<span>⚡ 100% Pure ConnectRPC Modular Architecture</span>
		</div>
		<h1 class="hero-title">
			MrBiz-OS Business Services & Verticals
		</h1>
		<p class="hero-subtitle">
			Explore all 19 integrated enterprise modules and customer-facing travel portals. Built for multi-business groups, tour operators, retail counters, and digital agencies.
		</p>

		<!-- Dedicated Customer & Partner Portals Banner -->
		<div class="portals-highlight-banner">
			<span class="highlight-tag">PUBLIC & PARTNER PORTALS:</span>
			<div class="portals-links">
				<a href="/customer/services" class="portal-chip customer">
					<span>🛎️ Customer Travel Services</span>
				</a>
				<a href="/travel/booking" class="portal-chip booking">
					<span>🗺️ Book a Tour (Instant PNR)</span>
				</a>
				<a href="/travel/partners" class="portal-chip partner">
					<span>🤝 Tour & Travel Partners (B2B)</span>
				</a>
				<a href="/login/customer" class="portal-chip login">
					<span>🔐 Traveler Login</span>
				</a>
			</div>
		</div>

		<!-- Search Bar -->
		<div class="search-bar-wrap">
			<span style="font-size: 1.1rem; opacity: 0.7;">🔍</span>
			<input
				type="text"
				class="search-input"
				placeholder="Search any service, vertical, invoice template, or workflow (e.g. 'whatsapp', 'pos', 'trade', 'payroll')..."
				bind:value={searchQuery}
			/>
			{#if searchQuery}
				<button class="clear-btn" onclick={() => (searchQuery = '')}>✕</button>
			{/if}
		</div>

		<!-- Category Filter Pills -->
		<div class="filter-pills-row">
			<button
				class="pill-btn"
				class:active={activeCategory === 'ALL'}
				onclick={() => (activeCategory = 'ALL')}
			>
				🌟 All Services ({services.length})
			</button>
			<button
				class="pill-btn"
				class:active={activeCategory === 'FINANCIAL'}
				onclick={() => (activeCategory = 'FINANCIAL')}
			>
				⚡ Financials & POS
			</button>
			<button
				class="pill-btn"
				class:active={activeCategory === 'AI_GROWTH'}
				onclick={() => (activeCategory = 'AI_GROWTH')}
			>
				💬 AI & Growth Desk
			</button>
			<button
				class="pill-btn"
				class:active={activeCategory === 'SUPPLY_CHAIN'}
				onclick={() => (activeCategory = 'SUPPLY_CHAIN')}
			>
				📦 Supply Chain & Trade
			</button>
			<button
				class="pill-btn"
				class:active={activeCategory === 'VERTICALS'}
				onclick={() => (activeCategory = 'VERTICALS')}
			>
				🏢 Industry Verticals
			</button>
		</div>
	</div>

	<!-- Services Grid -->
	<div class="services-grid">
		{#if filteredServices.length === 0}
			<div class="no-results-card">
				<span style="font-size: 2.5rem; margin-bottom: 0.5rem;">🔎</span>
				<h3 style="font-size: 1.25rem; color: #fff; margin: 0 0 0.5rem 0;">No services match "{searchQuery}"</h3>
				<p style="color: #94a3b8; font-size: 0.875rem; margin: 0 0 1rem 0;">
					Try searching for keywords like "billing", "whatsapp", "inbox", "healthcare", or "shipment".
				</p>
				<button class="btn btn-secondary" onclick={() => { searchQuery = ''; activeCategory = 'ALL'; }}>
					Reset Search
				</button>
			</div>
		{:else}
			{#each filteredServices as s}
				<div class="service-card animate-fade-in">
					<div class="service-card-top">
						<div class="service-icon-box">
							{s.icon}
						</div>
						<div style="flex: 1; min-width: 0;">
							<div style="display: flex; justify-content: space-between; align-items: flex-start; gap: 0.5rem;">
								<span class="service-category">{s.categoryLabel}</span>
								<span
									class="service-badge"
									style="background: {s.badgeColor}18; color: {s.badgeColor}; border: 1px solid {s.badgeColor}40;"
								>
									{s.badge}
								</span>
							</div>
							<h3 class="service-title">{s.title}</h3>
						</div>
					</div>

					<p class="service-summary">
						{s.summary}
					</p>

					<!-- Key Feature Tags -->
					<div class="feature-tags-wrap">
						{#each s.features as feat}
							<span class="feature-tag">✓ {feat}</span>
						{/each}
					</div>

					<div class="service-card-footer">
						<span class="service-stat">⚡ {s.stats}</span>
						<a href={s.href} class="launch-btn">
							Launch Service →
						</a>
					</div>
				</div>
			{/each}
		{/if}
	</div>

	<!-- Bottom Executive CTA -->
	<div class="cta-banner">
		<div>
			<h2 style="font-size: 1.35rem; color: #fff; margin: 0 0 0.35rem 0; font-weight: 800;">
				Ready to run your multi-business group seamlessly?
			</h2>
			<p style="font-size: 0.875rem; color: #cbd5e1; margin: 0;">
				Start instantly with the express billing counter or launch the full unified ERP workspace.
			</p>
		</div>
		<div style="display: flex; gap: 0.75rem; flex-wrap: wrap;">
			<a href="/login" class="btn btn-primary" style="padding: 0.65rem 1.25rem; font-weight: 700;">
				Sign In to Workspace →
			</a>
			<a href="/billing" class="btn btn-secondary" style="padding: 0.65rem 1.25rem;">
				⚡ Express Billing Counter
			</a>
		</div>
	</div>

</div>

<style>
	.services-page {
		display: flex;
		flex-direction: column;
		gap: 2rem;
		max-width: 1280px;
		margin: 0 auto;
		padding-bottom: 3rem;
	}

	.hero-section {
		text-align: center;
		padding: 2.5rem 1.5rem;
		background: radial-gradient(circle at 50% 0%, rgba(99, 102, 241, 0.18) 0%, transparent 65%),
		            rgba(15, 23, 42, 0.6);
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: var(--radius-lg);
		display: flex;
		flex-direction: column;
		align-items: center;
	}

	.hero-badge {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		background: rgba(99, 102, 241, 0.15);
		border: 1px solid rgba(99, 102, 241, 0.3);
		color: #a5b4fc;
		padding: 0.25rem 0.85rem;
		border-radius: 9999px;
		font-size: 0.75rem;
		font-weight: 700;
		margin-bottom: 1rem;
	}

	.hero-title {
		font-size: 2.25rem;
		font-weight: 900;
		color: #fff;
		margin: 0 0 0.75rem 0;
		letter-spacing: -0.02em;
	}

	.hero-subtitle {
		font-size: 1rem;
		color: #94a3b8;
		max-width: 680px;
		line-height: 1.6;
		margin: 0 0 1.25rem 0;
	}

	.portals-highlight-banner {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.6rem;
		background: rgba(15, 23, 42, 0.7);
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: 14px;
		padding: 0.85rem 1.25rem;
		margin-bottom: 2rem;
		max-width: 820px;
		width: 100%;
		box-sizing: border-box;
	}

	.highlight-tag {
		font-size: 0.7rem;
		font-weight: 800;
		color: #38bdf8;
		letter-spacing: 0.08em;
	}

	.portals-links {
		display: flex;
		gap: 0.65rem;
		flex-wrap: wrap;
		justify-content: center;
	}

	.portal-chip {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		padding: 0.45rem 0.85rem;
		border-radius: 8px;
		font-size: 0.82rem;
		font-weight: 700;
		text-decoration: none;
		transition: all 0.15s ease;
	}

	.portal-chip.customer {
		background: rgba(2, 132, 199, 0.2);
		border: 1px solid rgba(56, 189, 248, 0.4);
		color: #38bdf8;
	}

	.portal-chip.booking {
		background: rgba(16, 185, 129, 0.2);
		border: 1px solid rgba(52, 211, 153, 0.4);
		color: #34d399;
	}

	.portal-chip.partner {
		background: rgba(245, 158, 11, 0.2);
		border: 1px solid rgba(245, 158, 11, 0.4);
		color: #fbbf24;
	}

	.portal-chip.login {
		background: rgba(124, 58, 237, 0.2);
		border: 1px solid rgba(167, 139, 250, 0.4);
		color: #c084fc;
	}

	.portal-chip:hover {
		transform: translateY(-1px);
		filter: brightness(1.2);
	}

	.search-bar-wrap {
		position: relative;
		display: flex;
		align-items: center;
		width: 100%;
		max-width: 640px;
		background: rgba(11, 15, 25, 0.85);
		border: 1px solid rgba(99, 102, 241, 0.35);
		border-radius: 10px;
		padding: 0.6rem 1rem;
		box-shadow: 0 10px 25px rgba(0, 0, 0, 0.4);
		margin-bottom: 1.5rem;
	}

	.search-input {
		flex: 1;
		background: transparent;
		border: none;
		outline: none;
		color: #fff;
		font-size: 0.9375rem;
		margin-left: 0.6rem;
	}

	.search-input::placeholder {
		color: #64748b;
	}

	.clear-btn {
		background: none;
		border: none;
		color: #94a3b8;
		cursor: pointer;
		font-size: 0.85rem;
		padding: 0.2rem 0.4rem;
	}

	.clear-btn:hover {
		color: #fff;
	}

	.filter-pills-row {
		display: flex;
		align-items: center;
		justify-content: center;
		flex-wrap: wrap;
		gap: 0.5rem;
	}

	.pill-btn {
		background: rgba(30, 41, 59, 0.7);
		border: 1px solid rgba(255, 255, 255, 0.08);
		color: #cbd5e1;
		font-size: 0.8125rem;
		font-weight: 600;
		padding: 0.4rem 0.85rem;
		border-radius: 8px;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.pill-btn:hover {
		background: rgba(51, 65, 85, 0.8);
		color: #fff;
		border-color: rgba(99, 102, 241, 0.3);
	}

	.pill-btn.active {
		background: #6366f1;
		color: #fff;
		border-color: #818cf8;
		box-shadow: 0 4px 12px rgba(99, 102, 241, 0.4);
	}

	.services-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
		gap: 1.25rem;
	}

	.service-card {
		background: rgba(17, 24, 39, 0.8);
		backdrop-filter: blur(8px);
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: var(--radius-lg);
		padding: 1.5rem;
		display: flex;
		flex-direction: column;
		transition: all 0.2s ease;
	}

	.service-card:hover {
		border-color: rgba(99, 102, 241, 0.4);
		transform: translateY(-2px);
		box-shadow: 0 15px 30px rgba(0, 0, 0, 0.4), 0 0 15px rgba(99, 102, 241, 0.15);
	}

	.service-card-top {
		display: flex;
		align-items: flex-start;
		gap: 0.85rem;
		margin-bottom: 0.85rem;
	}

	.service-icon-box {
		width: 48px;
		height: 48px;
		border-radius: 12px;
		background: rgba(255, 255, 255, 0.05);
		border: 1px solid rgba(255, 255, 255, 0.1);
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 1.5rem;
		flex-shrink: 0;
	}

	.service-category {
		font-size: 0.65rem;
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		color: #64748b;
	}

	.service-badge {
		font-size: 0.65rem;
		font-weight: 700;
		padding: 0.15rem 0.5rem;
		border-radius: 9999px;
		white-space: nowrap;
	}

	.service-title {
		font-size: 1.1rem;
		font-weight: 800;
		color: #fff;
		margin: 0.25rem 0 0 0;
	}

	.service-summary {
		font-size: 0.8125rem;
		color: #94a3b8;
		line-height: 1.5;
		margin: 0 0 1rem 0;
		flex: 1;
	}

	.feature-tags-wrap {
		display: flex;
		flex-wrap: wrap;
		gap: 0.35rem;
		margin-bottom: 1.25rem;
	}

	.feature-tag {
		background: rgba(255, 255, 255, 0.04);
		border: 1px solid rgba(255, 255, 255, 0.06);
		color: #cbd5e1;
		font-size: 0.7rem;
		padding: 0.2rem 0.5rem;
		border-radius: 4px;
	}

	.service-card-footer {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding-top: 1rem;
		border-top: 1px solid rgba(255, 255, 255, 0.06);
	}

	.service-stat {
		font-size: 0.75rem;
		font-family: var(--font-mono);
		color: #34d399;
		font-weight: 600;
	}

	.launch-btn {
		background: rgba(99, 102, 241, 0.15);
		border: 1px solid rgba(99, 102, 241, 0.3);
		color: #a5b4fc;
		padding: 0.35rem 0.75rem;
		border-radius: 6px;
		font-size: 0.75rem;
		font-weight: 700;
		text-decoration: none;
		transition: all 0.15s ease;
	}

	.launch-btn:hover {
		background: #6366f1;
		color: #fff;
	}

	.no-results-card {
		grid-column: 1 / -1;
		text-align: center;
		padding: 3.5rem 1rem;
		background: rgba(17, 24, 39, 0.6);
		border: 1px dashed rgba(255, 255, 255, 0.1);
		border-radius: var(--radius-lg);
	}

	.cta-banner {
		background: linear-gradient(135deg, rgba(30, 41, 59, 0.9) 0%, rgba(15, 23, 42, 0.95) 100%);
		border: 1px solid rgba(99, 102, 241, 0.3);
		border-radius: var(--radius-lg);
		padding: 1.75rem 2rem;
		display: flex;
		justify-content: space-between;
		align-items: center;
		flex-wrap: wrap;
		gap: 1.5rem;
		box-shadow: 0 10px 25px rgba(0, 0, 0, 0.5);
	}
</style>
