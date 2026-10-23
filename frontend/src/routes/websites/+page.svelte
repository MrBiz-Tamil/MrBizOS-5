<script lang="ts">
	import { toast } from '$lib/stores/toast.svelte';

	interface StorefrontConfig {
		id: string;
		businessUnit: string;
		businessName: string;
		domain: string;
		routePath: string;
		tagline: string;
		primaryColor: string;
		isPublished: boolean;
		sslActive: boolean;
		monthlyVisits: string;
		blocks: { id: string; name: string; type: string; enabled: boolean }[];
	}

	let sites = $state<StorefrontConfig[]>([
		{
			id: 'SITE-TRV',
			businessUnit: 'TRAVEL',
			businessName: 'MrBiz Travel Concierge & Tour OS',
			domain: 'travel.mrbiz.group',
			routePath: '/customer/services',
			tagline: 'Handcrafted Holiday Circuits, Verified Hotels & Chauffeur Airport Meet',
			primaryColor: '#0284c7',
			isPublished: true,
			sslActive: true,
			monthlyVisits: '42,800',
			blocks: [
				{ id: 'b1', name: 'Hero Banner & PNR Search Radar', type: 'hero', enabled: true },
				{ id: 'b2', name: 'Curated Services Catalogue Grid', type: 'services', enabled: true },
				{ id: 'b3', name: '4-Step Interactive Tour Booking Wizard', type: 'booking', enabled: true },
				{ id: 'b4', name: 'Verified Guest Reviews & Video Stories', type: 'testimonials', enabled: true },
				{ id: 'b5', name: 'FAQ & Emergency Care Desk', type: 'faq', enabled: true },
				{ id: 'b6', name: '24/7 WhatsApp Floating Support Button', type: 'whatsapp_widget', enabled: true }
			]
		},
		{
			id: 'SITE-MED',
			businessUnit: 'HEALTHCARE',
			businessName: 'DrAyya Telehealth & Clinic Hub',
			domain: 'clinic.mrbiz.group',
			routePath: '/healthcare',
			tagline: '24/7 Specialist Doctor Consultations, E-Prescriptions & Lab Reports',
			primaryColor: '#ef4444',
			isPublished: true,
			sslActive: true,
			monthlyVisits: '18,200',
			blocks: [
				{ id: 'b1', name: 'Telemedicine Hero & Symptom Checker', type: 'hero', enabled: true },
				{ id: 'b2', name: 'Specialist Doctors & Time Slots', type: 'services', enabled: true },
				{ id: 'b3', name: 'Instant Video Appointment Booking', type: 'booking', enabled: true },
				{ id: 'b4', name: 'Patient CSAT & Hospital Accreditations', type: 'testimonials', enabled: true }
			]
		},
		{
			id: 'SITE-EST',
			businessUnit: 'REALESTATE',
			businessName: 'Prime Realty & Commercial Estates',
			domain: 'realty.mrbiz.group',
			routePath: '/realestate',
			tagline: 'Exclusive Office Complexes, Luxury Villas & Escrow Token Booking',
			primaryColor: '#f59e0b',
			isPublished: true,
			sslActive: true,
			monthlyVisits: '9,400',
			blocks: [
				{ id: 'b1', name: 'Featured Commercial Units & Floorplans', type: 'hero', enabled: true },
				{ id: 'b2', name: 'Virtual 3D Site Tour Booking', type: 'booking', enabled: true },
				{ id: 'b3', name: 'Escrow Token Reservation Gate', type: 'services', enabled: true }
			]
		},
		{
			id: 'SITE-RET',
			businessUnit: 'ECOMMERCE',
			businessName: 'MrBiz D2C Print-on-Demand & Retail',
			domain: 'shop.mrbiz.group',
			routePath: '/ecommerce',
			tagline: 'Premium Organic Apparel, Gallery Stretched Canvas & Custom Merchandise',
			primaryColor: '#8b5cf6',
			isPublished: true,
			sslActive: true,
			monthlyVisits: '31,500',
			blocks: [
				{ id: 'b1', name: 'Seasonal Apparel Drop & Promo Codes', type: 'hero', enabled: true },
				{ id: 'b2', name: 'Product Catalog Grid with Instant Cart', type: 'services', enabled: true },
				{ id: 'b3', name: 'Global Courier Tracking Radar', type: 'faq', enabled: true }
			]
		},
		{
			id: 'SITE-CSL',
			businessUnit: 'CONSULTING',
			businessName: 'MrBiz Software Consulting & SOW Agency',
			domain: 'agency.mrbiz.group',
			routePath: '/consulting',
			tagline: 'Enterprise ConnectRPC Backends, SvelteKit Frontends & Full Architecture Audits',
			primaryColor: '#6366f1',
			isPublished: true,
			sslActive: true,
			monthlyVisits: '12,100',
			blocks: [
				{ id: 'b1', name: 'Case Studies & High-Scale Systems', type: 'hero', enabled: true },
				{ id: 'b2', name: 'SOW Retainer Packages & Rate Cards', type: 'services', enabled: true },
				{ id: 'b3', name: 'Schedule 30-Min Technical Audit Call', type: 'booking', enabled: true }
			]
		}
	]);

	// Selected Site for Live CMS Customizer
	let selectedSiteId = $state<string>('SITE-TRV');
	let activeSite = $derived(sites.find(s => s.id === selectedSiteId) || sites[0]);

	// Device Preview State
	let previewDevice = $state<'desktop' | 'tablet' | 'mobile'>('desktop');

	// Domain Customizer Modal
	let showDomainModal = $state(false);
	let customDomainInput = $state('');
	let dnsStatus = $state<'unchecked' | 'checking' | 'verified'>('unchecked');

	function toggleBlock(blockId: string) {
		const b = activeSite.blocks.find(x => x.id === blockId);
		if (b) {
			b.enabled = !b.enabled;
			toast?.success?.(`Updated layout block: ${b.name}`);
		}
	}

	function handleSaveSite() {
		toast?.success?.(`Published changes for ${activeSite.businessName} to production edge servers!`);
	}

	function checkDnsRecord() {
		dnsStatus = 'checking';
		setTimeout(() => {
			dnsStatus = 'verified';
			toast?.success?.('CNAME record successfully verified. SSL certificate auto-provisioned!');
		}, 800);
	}
</script>

<svelte:head>
	<title>Multi-Business Websites & Headless CMS | MrBiz OS</title>
	<meta name="description" content="Manage public website storefronts, custom domains, visual page blocks, and themes for all group subsidiaries." />
</svelte:head>

<div class="websites-manager-page">
	<!-- Page Header -->
	<header class="page-top-bar">
		<div>
			<div class="sub-pill">HEADLESS MULTI-SITE STOREFRONT ENGINE</div>
			<h1 class="page-title">Multi-Business Websites & CMS</h1>
			<p class="page-desc">Centralized publishing, custom domains, visual page blocks, and branding across all group companies.</p>
		</div>

		<div class="top-actions">
			<button type="button" class="btn-domain-modal" onclick={() => (showDomainModal = true)}>
				<span>🌐 Custom Domain Manager</span>
			</button>
			<button type="button" class="btn-save-all" onclick={handleSaveSite}>
				<span>⚡ Publish Changes Live</span>
			</button>
		</div>
	</header>

	<!-- Main Workspace Split -->
	<div class="workspace-grid">
		<!-- LEFT PANE: Site Selector & Visual Block Configurator -->
		<div class="config-sidebar">
			<!-- Storefront Selector -->
			<div class="sidebar-box">
				<div class="box-label">SELECT BUSINESS STOREFRONT</div>
				<div class="sites-list">
					{#each sites as s}
						<button
							type="button"
							class="site-select-btn {selectedSiteId === s.id ? 'active' : ''}"
							onclick={() => (selectedSiteId = s.id)}
						>
							<div class="btn-color-bar" style="background: {s.primaryColor};"></div>
							<div class="btn-info">
								<div class="s-name">{s.businessName}</div>
								<div class="s-domain font-mono">{s.domain}</div>
							</div>
							<span class="status-dot-active"></span>
						</button>
					{/each}
				</div>
			</div>

			<!-- Visual Page Blocks Manager -->
			<div class="sidebar-box">
				<div class="box-label">MODULAR STOREFRONT PAGE BLOCKS</div>
				<p class="box-sub">Enable or reorder blocks rendered on <strong>{activeSite.domain}</strong>:</p>

				<div class="blocks-stack">
					{#each activeSite.blocks as blk}
						<div class="block-row {blk.enabled ? '' : 'disabled'}">
							<div class="block-drag-icon">⋮⋮</div>
							<div class="block-info">
								<div class="block-title">{blk.name}</div>
								<div class="block-type font-mono">{blk.type}</div>
							</div>
							<label class="toggle-switch">
								<input
									type="checkbox"
									checked={blk.enabled}
									onchange={() => toggleBlock(blk.id)}
								/>
								<span class="slider"></span>
							</label>
						</div>
					{/each}
				</div>

				<button type="button" class="btn-add-block" onclick={() => toast?.info?.('Block library: Hero, Services, Booking Wizard, Pricing, Reviews, FAQ')}>
					+ Add Custom Block from Library
				</button>
			</div>

			<!-- Branding & Colors -->
			<div class="sidebar-box">
				<div class="box-label">BRANDING & THEME PALETTE</div>
				<div class="branding-group">
					<div class="brand-field">
						<label for="w-title">Storefront Title</label>
						<input id="w-title" type="text" bind:value={activeSite.businessName} class="b-input" />
					</div>
					<div class="brand-field mt-2">
						<label for="w-tagline">Marketing Tagline</label>
						<textarea id="w-tagline" bind:value={activeSite.tagline} rows="2" class="b-input textarea"></textarea>
					</div>
					<div class="brand-field mt-2">
						<label for="w-color">Primary Brand Accent</label>
						<div class="color-picker-row">
							<input id="w-color" type="color" bind:value={activeSite.primaryColor} class="color-picker" />
							<span class="font-mono" style="color: {activeSite.primaryColor}; font-weight: 700;">{activeSite.primaryColor}</span>
						</div>
					</div>
				</div>
			</div>
		</div>

		<!-- RIGHT PANE: Live Visual Storefront Preview -->
		<div class="preview-workspace">
			<div class="preview-toolbar">
				<div class="device-buttons">
					<button
						type="button"
						class="dev-btn {previewDevice === 'desktop' ? 'active' : ''}"
						onclick={() => (previewDevice = 'desktop')}
					>
						🖥️ Desktop (100%)
					</button>
					<button
						type="button"
						class="dev-btn {previewDevice === 'tablet' ? 'active' : ''}"
						onclick={() => (previewDevice = 'tablet')}
					>
						📱 Tablet (768px)
					</button>
					<button
						type="button"
						class="dev-btn {previewDevice === 'mobile' ? 'active' : ''}"
						onclick={() => (previewDevice = 'mobile')}
					>
						📱 Mobile (375px)
					</button>
				</div>

				<div class="preview-url-bar font-mono">
					<span class="lock-icon">🔒</span>
					<span>https://{activeSite.domain}</span>
					<a href={activeSite.routePath} target="_blank" rel="noreferrer" class="open-tab-link" title="Open in New Window">↗</a>
				</div>
			</div>

			<!-- Interactive Live Storefront Canvas -->
			<div class="preview-viewport-container {previewDevice}">
				<div class="storefront-canvas" style="--site-brand: {activeSite.primaryColor};">
					<!-- Storefront Top Header -->
					<div class="canvas-header">
						<div class="canvas-logo">
							<span class="logo-mark" style="background: {activeSite.primaryColor};">★</span>
							<span class="logo-text">{activeSite.businessName}</span>
						</div>
						<div class="canvas-nav-links">
							<span>Services</span>
							<span>Booking</span>
							<span>Vouchers</span>
							<span class="canvas-cta-pill" style="background: {activeSite.primaryColor};">Customer Portal</span>
						</div>
					</div>

					<!-- Block 1: Hero -->
					{#if activeSite.blocks.find(b => b.type === 'hero')?.enabled}
						<div class="canvas-hero" style="background: radial-gradient(circle at 50% 10%, {activeSite.primaryColor}25 0%, #090d16 80%);">
							<div class="hero-chip" style="color: {activeSite.primaryColor}; border-color: {activeSite.primaryColor}40;">
								OFFICIAL DIGITAL STOREFRONT • {activeSite.businessUnit}
							</div>
							<h2 class="hero-h1">{activeSite.businessName}</h2>
							<p class="hero-tagline">{activeSite.tagline}</p>
							<div class="hero-btns">
								<a href={activeSite.routePath} class="btn-canvas-primary" style="background: {activeSite.primaryColor};">
									Explore Services & Bookings ➔
								</a>
								<a href="/login/customer" class="btn-canvas-secondary">
									Customer Sign In
								</a>
							</div>
						</div>
					{/if}

					<!-- Block 2: Services Directory Grid -->
					{#if activeSite.blocks.find(b => b.type === 'services')?.enabled}
						<div class="canvas-section">
							<div class="sec-label" style="color: {activeSite.primaryColor};">FEATURED COMMERCIAL OFFERINGS</div>
							<div class="canvas-services-cards">
								<div class="c-card">
									<div class="c-icon" style="background: {activeSite.primaryColor}20; color: {activeSite.primaryColor};">⭐</div>
									<h4>Signature {activeSite.businessUnit} Package</h4>
									<p>Instant digital confirmation with direct DMC net rates & zero walkout guarantee.</p>
									<div class="c-price" style="color: {activeSite.primaryColor};">From ₹14,999</div>
								</div>
								<div class="c-card">
									<div class="c-icon" style="background: {activeSite.primaryColor}20; color: {activeSite.primaryColor};">⚡</div>
									<h4>Express Consultation & Fast-Track</h4>
									<p>Sub-15 minute response time with vetted specialists and personal coordinators.</p>
									<div class="c-price" style="color: {activeSite.primaryColor};">From ₹850</div>
								</div>
								<div class="c-card">
									<div class="c-icon" style="background: {activeSite.primaryColor}20; color: {activeSite.primaryColor};">🛡️</div>
									<h4>24/7 Care & Emergency Guarantee</h4>
									<p>Continuous on-trip and post-service care desk with immediate human warm handoff.</p>
									<div class="c-price" style="color: {activeSite.primaryColor};">Complimentary</div>
								</div>
							</div>
						</div>
					{/if}

					<!-- Block 3: Booking Wizard Preview -->
					{#if activeSite.blocks.find(b => b.type === 'booking')?.enabled}
						<div class="canvas-booking-banner" style="border-color: {activeSite.primaryColor}40;">
							<div>
								<div class="b-badge" style="background: {activeSite.primaryColor}20; color: {activeSite.primaryColor};">LIVE RESERVATION ENGINE</div>
								<h3>Instant Slot & Ticket Allocation</h3>
								<p>Dynamic UPI QR checkout, instant tax invoice generation, and WhatsApp voucher dispatch.</p>
							</div>
							<a href={activeSite.routePath} class="btn-launch-wizard" style="background: {activeSite.primaryColor};">
								Open Interactive Wizard ➔
							</a>
						</div>
					{/if}

					<!-- Floating WhatsApp Widget -->
					{#if activeSite.blocks.find(b => b.type === 'whatsapp_widget')?.enabled}
						<div class="canvas-floating-wa">
							<span class="wa-dot"></span>
							<span>24/7 WhatsApp Concierge</span>
						</div>
					{/if}
				</div>
			</div>
		</div>
	</div>
</div>

<!-- Custom Domain DNS Modal -->
{#if showDomainModal}
	<div class="modal-backdrop" onclick={() => (showDomainModal = false)}>
		<div class="domain-modal-box" onclick={(e) => e.stopPropagation()}>
			<button type="button" class="btn-modal-close" onclick={() => (showDomainModal = false)}>✕</button>

			<div class="modal-header">
				<div class="m-badge">EDGE ROUTING & DNS</div>
				<h3 class="m-title">Custom Domain Setup</h3>
				<p class="m-sub">Point your company custom domain to MrBiz-OS with automatic Let's Encrypt SSL.</p>
			</div>

			<div class="domain-form">
				<div class="d-field">
					<label for="d-domain">Your Custom Domain / Subdomain</label>
					<input id="d-domain" type="text" bind:value={customDomainInput} placeholder="e.g. tours.pandiantravels.in" class="d-input font-mono" />
				</div>

				<div class="dns-records-box">
					<div class="dns-title">Required DNS CNAME Record:</div>
					<div class="dns-row font-mono">
						<span class="col-type">CNAME</span>
						<span class="col-host">{customDomainInput || 'tours'}</span>
						<span class="col-target">ingress.mrbiz.group</span>
					</div>
				</div>

				{#if dnsStatus === 'verified'}
					<div class="dns-success-banner">
						<span>✅ Domain validated! SSL certificate generated and active on global CDN edge.</span>
					</div>
				{/if}

				<div class="modal-actions">
					<button type="button" class="btn-cancel" onclick={() => (showDomainModal = false)}>Cancel</button>
					<button type="button" class="btn-verify-dns" onclick={checkDnsRecord} disabled={dnsStatus === 'checking'}>
						{#if dnsStatus === 'checking'}
							<span>Verifying DNS Records...</span>
						{:else}
							<span>Verify DNS & Issue SSL</span>
						{/if}
					</button>
				</div>
			</div>
		</div>
	</div>
{/if}

<style>
	.websites-manager-page {
		min-height: 100vh;
		background: #090d16;
		color: #f8fafc;
		font-family: var(--font-sans, system-ui, sans-serif);
		padding: 2rem;
		box-sizing: border-box;
	}

	.page-top-bar {
		display: flex;
		justify-content: space-between;
		align-items: center;
		flex-wrap: wrap;
		gap: 1rem;
		padding-bottom: 1.5rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
		margin-bottom: 2rem;
	}

	.sub-pill {
		font-size: 0.72rem;
		font-weight: 800;
		color: #38bdf8;
		letter-spacing: 0.08em;
	}

	.page-title {
		font-size: 1.8rem;
		font-weight: 900;
		color: #ffffff;
		margin: 0.2rem 0;
	}

	.page-desc {
		font-size: 0.9rem;
		color: #94a3b8;
		margin: 0;
	}

	.top-actions {
		display: flex;
		gap: 0.75rem;
	}

	.btn-domain-modal {
		padding: 0.7rem 1.25rem;
		border-radius: 10px;
		background: rgba(255, 255, 255, 0.08);
		border: 1px solid rgba(255, 255, 255, 0.15);
		color: #ffffff;
		font-size: 0.85rem;
		font-weight: 700;
		cursor: pointer;
	}

	.btn-save-all {
		padding: 0.7rem 1.4rem;
		border-radius: 10px;
		background: linear-gradient(135deg, #10b981 0%, #059669 100%);
		border: none;
		color: #ffffff;
		font-size: 0.88rem;
		font-weight: 800;
		cursor: pointer;
		box-shadow: 0 4px 14px rgba(16, 185, 129, 0.4);
	}

	/* Workspace Grid */
	.workspace-grid {
		display: grid;
		grid-template-columns: 360px 1fr;
		gap: 2rem;
		align-items: flex-start;
	}

	.config-sidebar {
		display: flex;
		flex-direction: column;
		gap: 1.5rem;
	}

	.sidebar-box {
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 16px;
		padding: 1.25rem;
	}

	.box-label {
		font-size: 0.72rem;
		font-weight: 800;
		color: #64748b;
		letter-spacing: 0.08em;
		margin-bottom: 0.65rem;
	}

	.box-sub {
		font-size: 0.78rem;
		color: #94a3b8;
		margin: 0 0 0.85rem;
	}

	.sites-list {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.site-select-btn {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		padding: 0.7rem 0.85rem;
		border-radius: 10px;
		background: rgba(255, 255, 255, 0.02);
		border: 1px solid rgba(255, 255, 255, 0.06);
		color: #ffffff;
		text-align: left;
		cursor: pointer;
		transition: all 0.15s;
	}

	.site-select-btn:hover {
		background: rgba(255, 255, 255, 0.06);
	}

	.site-select-btn.active {
		border-color: #38bdf8;
		background: rgba(2, 132, 199, 0.12);
	}

	.btn-color-bar {
		width: 4px;
		height: 32px;
		border-radius: 999px;
	}

	.btn-info {
		flex: 1;
		min-width: 0;
	}

	.s-name {
		font-size: 0.84rem;
		font-weight: 700;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.s-domain {
		font-size: 0.72rem;
		color: #94a3b8;
	}

	.status-dot-active {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		background: #34d399;
	}

	/* Blocks Stack */
	.blocks-stack {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		margin-bottom: 1rem;
	}

	.block-row {
		display: flex;
		align-items: center;
		gap: 0.65rem;
		padding: 0.65rem 0.8rem;
		background: #090d16;
		border: 1px solid rgba(255, 255, 255, 0.06);
		border-radius: 8px;
	}

	.block-row.disabled {
		opacity: 0.45;
	}

	.block-drag-icon {
		color: #64748b;
		cursor: grab;
	}

	.block-info {
		flex: 1;
	}

	.block-title {
		font-size: 0.8rem;
		font-weight: 700;
		color: #ffffff;
	}

	.block-type {
		font-size: 0.68rem;
		color: #38bdf8;
	}

	/* Toggle switch */
	.toggle-switch {
		position: relative;
		display: inline-block;
		width: 34px;
		height: 20px;
	}

	.toggle-switch input { opacity: 0; width: 0; height: 0; }
	.slider {
		position: absolute; cursor: pointer; inset: 0;
		background-color: #334155; border-radius: 20px; transition: 0.2s;
	}
	.slider:before {
		position: absolute; content: ""; height: 14px; width: 14px; left: 3px; bottom: 3px;
		background-color: white; border-radius: 50%; transition: 0.2s;
	}
	input:checked + .slider { background-color: #10b981; }
	input:checked + .slider:before { transform: translateX(14px); }

	.btn-add-block {
		width: 100%;
		padding: 0.6rem;
		border-radius: 8px;
		background: rgba(255, 255, 255, 0.05);
		border: 1px dashed rgba(255, 255, 255, 0.2);
		color: #cbd5e1;
		font-size: 0.78rem;
		font-weight: 700;
		cursor: pointer;
	}

	.brand-field label {
		display: block;
		font-size: 0.72rem;
		font-weight: 700;
		color: #94a3b8;
		margin-bottom: 0.3rem;
	}

	.b-input {
		width: 100%;
		box-sizing: border-box;
		padding: 0.6rem 0.8rem;
		background: #090d16;
		border: 1px solid rgba(255, 255, 255, 0.12);
		border-radius: 8px;
		color: #ffffff;
		font-size: 0.82rem;
	}

	.b-input.textarea { resize: vertical; }

	.color-picker-row {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}

	.color-picker {
		width: 36px;
		height: 36px;
		border: none;
		border-radius: 8px;
		background: none;
		cursor: pointer;
	}

	/* PREVIEW WORKSPACE */
	.preview-workspace {
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 16px;
		overflow: hidden;
	}

	.preview-toolbar {
		background: #090d16;
		padding: 0.75rem 1.25rem;
		display: flex;
		justify-content: space-between;
		align-items: center;
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
		flex-wrap: wrap;
		gap: 0.75rem;
	}

	.device-buttons {
		display: flex;
		gap: 0.35rem;
	}

	.dev-btn {
		padding: 0.45rem 0.75rem;
		border-radius: 6px;
		background: rgba(255, 255, 255, 0.05);
		border: 1px solid rgba(255, 255, 255, 0.08);
		color: #94a3b8;
		font-size: 0.76rem;
		font-weight: 700;
		cursor: pointer;
	}

	.dev-btn.active {
		background: rgba(2, 132, 199, 0.2);
		border-color: #38bdf8;
		color: #ffffff;
	}

	.preview-url-bar {
		display: flex;
		align-items: center;
		gap: 0.45rem;
		background: rgba(255, 255, 255, 0.05);
		padding: 0.35rem 0.85rem;
		border-radius: 6px;
		font-size: 0.75rem;
		color: #cbd5e1;
	}

	.open-tab-link {
		color: #38bdf8;
		text-decoration: none;
		font-weight: 800;
	}

	/* Viewport container */
	.preview-viewport-container {
		display: flex;
		justify-content: center;
		background: #060911;
		padding: 1.5rem;
		min-height: 600px;
		overflow-x: auto;
	}

	.preview-viewport-container.desktop .storefront-canvas { width: 100%; max-width: 900px; }
	.preview-viewport-container.tablet .storefront-canvas { width: 720px; }
	.preview-viewport-container.mobile .storefront-canvas { width: 360px; }

	.storefront-canvas {
		background: #090d16;
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: 14px;
		overflow: hidden;
		box-shadow: 0 20px 50px rgba(0, 0, 0, 0.7);
		position: relative;
	}

	.canvas-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 0.85rem 1.25rem;
		background: rgba(15, 23, 42, 0.95);
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
	}

	.canvas-logo {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.logo-mark {
		width: 24px;
		height: 24px;
		border-radius: 6px;
		display: flex;
		align-items: center;
		justify-content: center;
		color: white;
		font-size: 0.8rem;
		font-weight: 900;
	}

	.logo-text { font-size: 0.85rem; font-weight: 800; color: #ffffff; }

	.canvas-nav-links {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		font-size: 0.74rem;
		color: #94a3b8;
	}

	.canvas-cta-pill {
		padding: 0.25rem 0.6rem;
		border-radius: 999px;
		color: #ffffff;
		font-weight: 700;
	}

	.canvas-hero {
		padding: 3rem 1.5rem;
		text-align: center;
		border-bottom: 1px solid rgba(255, 255, 255, 0.06);
	}

	.hero-chip {
		display: inline-block;
		font-size: 0.68rem;
		font-weight: 800;
		padding: 0.2rem 0.6rem;
		border-radius: 999px;
		background: rgba(255, 255, 255, 0.04);
		border: 1px solid;
		margin-bottom: 1rem;
	}

	.hero-h1 {
		font-size: 1.8rem;
		font-weight: 900;
		color: #ffffff;
		margin: 0 0 0.5rem;
	}

	.hero-tagline {
		font-size: 0.92rem;
		color: #94a3b8;
		max-width: 500px;
		margin: 0 auto 1.5rem;
		line-height: 1.5;
	}

	.hero-btns {
		display: flex;
		justify-content: center;
		gap: 0.75rem;
	}

	.btn-canvas-primary {
		padding: 0.65rem 1.25rem;
		border-radius: 8px;
		color: #ffffff;
		font-size: 0.82rem;
		font-weight: 800;
		text-decoration: none;
	}

	.btn-canvas-secondary {
		padding: 0.65rem 1.25rem;
		border-radius: 8px;
		background: rgba(255, 255, 255, 0.08);
		color: #ffffff;
		font-size: 0.82rem;
		font-weight: 700;
		text-decoration: none;
	}

	.canvas-section {
		padding: 2rem 1.5rem;
	}

	.sec-label {
		font-size: 0.7rem;
		font-weight: 800;
		letter-spacing: 0.08em;
		margin-bottom: 1rem;
		text-align: center;
	}

	.canvas-services-cards {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
		gap: 1rem;
	}

	.c-card {
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.06);
		border-radius: 10px;
		padding: 1rem;
	}

	.c-icon {
		width: 32px;
		height: 32px;
		border-radius: 8px;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 1.1rem;
		margin-bottom: 0.5rem;
	}

	.c-card h4 { font-size: 0.88rem; font-weight: 800; color: #ffffff; margin: 0 0 0.35rem; }
	.c-card p { font-size: 0.74rem; color: #94a3b8; line-height: 1.4; margin: 0 0 0.75rem; }
	.c-price { font-size: 0.82rem; font-weight: 800; }

	.canvas-booking-banner {
		margin: 0 1.5rem 2rem;
		padding: 1.25rem;
		border-radius: 12px;
		background: rgba(255, 255, 255, 0.02);
		border: 1px solid;
		display: flex;
		justify-content: space-between;
		align-items: center;
		flex-wrap: wrap;
		gap: 1rem;
	}

	.b-badge {
		font-size: 0.65rem;
		font-weight: 800;
		padding: 0.15rem 0.5rem;
		border-radius: 4px;
		display: inline-block;
		margin-bottom: 0.25rem;
	}

	.canvas-booking-banner h3 { font-size: 1.05rem; font-weight: 800; color: #ffffff; margin: 0 0 0.25rem; }
	.canvas-booking-banner p { font-size: 0.75rem; color: #94a3b8; margin: 0; }

	.btn-launch-wizard {
		padding: 0.6rem 1.1rem;
		border-radius: 8px;
		color: #ffffff;
		font-size: 0.8rem;
		font-weight: 800;
		text-decoration: none;
	}

	.canvas-floating-wa {
		position: absolute;
		bottom: 16px;
		right: 16px;
		display: flex;
		align-items: center;
		gap: 0.4rem;
		background: #25d366;
		color: #ffffff;
		padding: 0.45rem 0.85rem;
		border-radius: 999px;
		font-size: 0.72rem;
		font-weight: 800;
		box-shadow: 0 4px 14px rgba(37, 211, 102, 0.4);
	}

	.wa-dot {
		width: 6px;
		height: 6px;
		border-radius: 50%;
		background: #ffffff;
	}

	/* Modal */
	.modal-backdrop {
		position: fixed;
		inset: 0;
		z-index: 3000;
		background: rgba(0, 0, 0, 0.8);
		backdrop-filter: blur(8px);
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 1.5rem;
	}

	.domain-modal-box {
		position: relative;
		width: 100%;
		max-width: 580px;
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.15);
		border-radius: 18px;
		padding: 2rem;
		box-shadow: 0 25px 60px rgba(0, 0, 0, 0.8);
	}

	.btn-modal-close {
		position: absolute;
		top: 1.25rem;
		right: 1.25rem;
		background: rgba(255, 255, 255, 0.08);
		border: none;
		color: #ffffff;
		width: 32px;
		height: 32px;
		border-radius: 50%;
		cursor: pointer;
	}

	.m-badge { font-size: 0.7rem; font-weight: 800; color: #38bdf8; margin-bottom: 0.25rem; }
	.m-title { font-size: 1.4rem; font-weight: 900; color: #ffffff; margin: 0 0 0.35rem; }
	.m-sub { font-size: 0.85rem; color: #94a3b8; margin: 0 0 1.5rem; }

	.d-field label { font-size: 0.78rem; font-weight: 700; color: #cbd5e1; display: block; margin-bottom: 0.35rem; }
	.d-input {
		width: 100%;
		box-sizing: border-box;
		padding: 0.75rem 0.9rem;
		background: #090d16;
		border: 1px solid rgba(255, 255, 255, 0.14);
		border-radius: 10px;
		color: #ffffff;
		font-size: 0.92rem;
		margin-bottom: 1.25rem;
	}

	.dns-records-box {
		background: #090d16;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 10px;
		padding: 1rem;
		margin-bottom: 1.25rem;
	}

	.dns-title { font-size: 0.75rem; color: #64748b; font-weight: 700; margin-bottom: 0.5rem; }
	.dns-row {
		display: flex;
		justify-content: space-between;
		font-size: 0.82rem;
		color: #38bdf8;
	}

	.dns-success-banner {
		background: rgba(16, 185, 129, 0.15);
		border: 1px solid rgba(16, 185, 129, 0.35);
		color: #34d399;
		padding: 0.75rem;
		border-radius: 8px;
		font-size: 0.82rem;
		margin-bottom: 1.25rem;
	}

	.modal-actions {
		display: flex;
		justify-content: flex-end;
		gap: 0.75rem;
	}

	.btn-cancel {
		padding: 0.7rem 1.25rem;
		border-radius: 8px;
		background: rgba(255, 255, 255, 0.08);
		border: none;
		color: #ffffff;
		font-size: 0.85rem;
		cursor: pointer;
	}

	.btn-verify-dns {
		padding: 0.7rem 1.4rem;
		border-radius: 8px;
		background: linear-gradient(135deg, #0284c7 0%, #2563eb 100%);
		border: none;
		color: #ffffff;
		font-size: 0.85rem;
		font-weight: 800;
		cursor: pointer;
	}

	@media (max-width: 1024px) {
		.workspace-grid {
			grid-template-columns: 1fr;
		}
	}
</style>
