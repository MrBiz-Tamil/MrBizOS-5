<script lang="ts">
	import { page } from '$app/stores';

	interface Props {
		activeTab?: 'services' | 'booking' | 'packages' | 'partners';
	}

	let { activeTab = 'services' }: Props = $props();

	let mobileMenuOpen = $state(false);
	let portalMenuOpen = $state(false);
	let currentPath = $derived($page.url.pathname);

	function toggleMobileMenu() {
		mobileMenuOpen = !mobileMenuOpen;
	}

	function closeDropdowns() {
		portalMenuOpen = false;
		mobileMenuOpen = false;
	}
</script>

<svelte:window onclick={() => { portalMenuOpen = false; }} />

<header class="portal-header">
	<div class="portal-header-container">
		<!-- Brand & Logo -->
		<a href="/customer/services" class="portal-brand" onclick={closeDropdowns}>
			<div class="brand-badge-icon">
				<span class="icon">✈️</span>
				<div class="pulse-indicator"></div>
			</div>
			<div class="brand-text">
				<div class="brand-title">MrBiz <span class="accent-text">Travel & Tours</span></div>
				<div class="brand-subtitle">Customer Concierge & DMC Network</div>
			</div>
		</a>

		<!-- Desktop Navigation Links -->
		<nav class="portal-nav">
			<a
				href="/customer/services"
				class="nav-link {currentPath.startsWith('/customer/services') || activeTab === 'services' ? 'active' : ''}"
			>
				<span class="nav-icon">🛎️</span>
				<span>Customer Services</span>
			</a>
			<a
				href="/travel/booking"
				class="nav-link {currentPath.startsWith('/travel/booking') || activeTab === 'booking' ? 'active' : ''}"
			>
				<span class="nav-icon">🗺️</span>
				<span>Book a Tour</span>
				<span class="nav-pill">Instant PNR</span>
			</a>
			<a
				href="/travel/packages"
				class="nav-link {currentPath.startsWith('/travel/packages') || activeTab === 'packages' ? 'active' : ''}"
			>
				<span class="nav-icon">🌴</span>
				<span>Holiday Packages</span>
			</a>
			<a
				href="/travel/partners"
				class="nav-link {currentPath.startsWith('/travel/partners') || activeTab === 'partners' ? 'active' : ''}"
			>
				<span class="nav-icon">🤝</span>
				<span>Travel Partners (B2B)</span>
				<span class="partner-pill">12.5% Margin</span>
			</a>
		</nav>

		<!-- Right Side CTAs & Portal Switcher -->
		<div class="portal-actions">
			<!-- WhatsApp Quick Support -->
			<a
				href="https://wa.me/919884011223?text=Hi%20MrBiz%20Travel%20Team%2C%20I%20need%20assistance%20with%20a%20tour%20service"
				target="_blank"
				rel="noreferrer"
				class="wa-support-btn"
				title="Chat with 24/7 Travel Concierge"
			>
				<span class="wa-dot"></span>
				<span class="wa-text">24/7 WhatsApp</span>
			</a>

			<!-- Multi-Portal Login Dropdown -->
			<div class="portal-dropdown-wrapper" onclick={(e) => e.stopPropagation()}>
				<button
					type="button"
					class="portal-dropdown-btn {portalMenuOpen ? 'open' : ''}"
					onclick={() => (portalMenuOpen = !portalMenuOpen)}
					aria-label="Switch Portals and Login Pages"
				>
					<span class="btn-avatar">🔐</span>
					<span class="btn-text">Portal Logins</span>
					<span class="chevron">▾</span>
				</button>

				{#if portalMenuOpen}
					<div class="portal-dropdown-menu">
						<div class="menu-heading">SELECT YOUR ACCESS PORTAL</div>

						<a href="/login/customer" class="portal-item" onclick={closeDropdowns}>
							<div class="portal-item-icon traveler">🧳</div>
							<div class="portal-item-info">
								<div class="portal-item-title">Customer / Traveler Login</div>
								<div class="portal-item-desc">Access your itineraries, vouchers & live chauffeur tracker</div>
							</div>
							<span class="portal-badge customer">Guest</span>
						</a>

						<a href="/login/partner" class="portal-item" onclick={closeDropdowns}>
							<div class="portal-item-icon partner">🤝</div>
							<div class="portal-item-info">
								<div class="portal-item-title">B2B Partner & Travel Agent</div>
								<div class="portal-item-desc">Wholesale tariffs, commission khata & agent bookings</div>
							</div>
							<span class="portal-badge partner">B2B Net</span>
						</a>

						<a href="/login/crew" class="portal-item" onclick={closeDropdowns}>
							<div class="portal-item-icon crew">🚗</div>
							<div class="portal-item-info">
								<div class="portal-item-title">Driver & Tour Guide Crew</div>
								<div class="portal-item-desc">Duty slips, airport pickup placards & guest rosters</div>
							</div>
							<span class="portal-badge crew">On-Ground</span>
						</a>

						<div class="menu-divider"></div>

						<a href="/login" class="portal-item admin" onclick={closeDropdowns}>
							<div class="portal-item-icon admin">💼</div>
							<div class="portal-item-info">
								<div class="portal-item-title">MrBiz ERP Staff / Admin</div>
								<div class="portal-item-desc">Operations desk, finance, billing & master controls</div>
							</div>
							<span class="portal-badge admin">HQ</span>
						</a>
					</div>
				{/if}
			</div>

			<!-- Mobile Menu Button -->
			<button
				type="button"
				class="mobile-toggle-btn"
				onclick={toggleMobileMenu}
				aria-label="Toggle navigation menu"
			>
				{mobileMenuOpen ? '✕' : '☰'}
			</button>
		</div>
	</div>

	<!-- Mobile Navigation Drawer -->
	{#if mobileMenuOpen}
		<div class="mobile-drawer">
			<div class="mobile-links">
				<a href="/customer/services" class="mobile-link" onclick={closeDropdowns}>
					🛎️ Customer Services Portal
				</a>
				<a href="/travel/booking" class="mobile-link" onclick={closeDropdowns}>
					🗺️ Book Tour Package (Instant PNR)
				</a>
				<a href="/travel/packages" class="mobile-link" onclick={closeDropdowns}>
					🌴 Browse All Holiday Packages
				</a>
				<a href="/travel/partners" class="mobile-link" onclick={closeDropdowns}>
					🤝 Tour & Travel Partners Hub (B2B)
				</a>
			</div>

			<div class="mobile-portals-title">DEDICATED LOGIN PORTALS</div>
			<div class="mobile-portals">
				<a href="/login/customer" class="mobile-portal-btn customer" onclick={closeDropdowns}>
					🧳 Customer / Traveler Login
				</a>
				<a href="/login/partner" class="mobile-portal-btn partner" onclick={closeDropdowns}>
					🤝 B2B Partner / Agency Login
				</a>
				<a href="/login/crew" class="mobile-portal-btn crew" onclick={closeDropdowns}>
					🚗 Driver & Guide Crew Login
				</a>
				<a href="/login" class="mobile-portal-btn admin" onclick={closeDropdowns}>
					💼 MrBiz OS Staff Login
				</a>
			</div>
		</div>
	{/if}
</header>

<style>
	.portal-header {
		position: sticky;
		top: 0;
		z-index: 1000;
		background: rgba(15, 23, 42, 0.94);
		backdrop-filter: blur(16px);
		-webkit-backdrop-filter: blur(16px);
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
		color: #f8fafc;
		font-family: var(--font-sans, system-ui, -apple-system, sans-serif);
		box-shadow: 0 4px 20px -2px rgba(0, 0, 0, 0.3);
	}

	.portal-header-container {
		max-width: 1360px;
		margin: 0 auto;
		height: 72px;
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0 1.5rem;
		gap: 1rem;
	}

	/* Brand */
	.portal-brand {
		display: flex;
		align-items: center;
		gap: 0.875rem;
		text-decoration: none;
		color: inherit;
		flex-shrink: 0;
	}

	.brand-badge-icon {
		position: relative;
		width: 44px;
		height: 44px;
		border-radius: 12px;
		background: linear-gradient(135deg, #0284c7 0%, #4f46e5 100%);
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 1.35rem;
		box-shadow: 0 4px 14px rgba(2, 132, 199, 0.35);
	}

	.pulse-indicator {
		position: absolute;
		top: -2px;
		right: -2px;
		width: 10px;
		height: 10px;
		border-radius: 50%;
		background: #10b981;
		border: 2px solid #0f172a;
		animation: pulse-ring 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
	}

	@keyframes pulse-ring {
		0%, 100% { opacity: 1; transform: scale(1); }
		50% { opacity: 0.5; transform: scale(1.2); }
	}

	.brand-text {
		display: flex;
		flex-direction: column;
	}

	.brand-title {
		font-size: 1.15rem;
		font-weight: 800;
		letter-spacing: -0.02em;
		color: #ffffff;
	}

	.accent-text {
		background: linear-gradient(90deg, #38bdf8, #818cf8);
		-webkit-background-clip: text;
		-webkit-text-fill-color: transparent;
	}

	.brand-subtitle {
		font-size: 0.72rem;
		color: #94a3b8;
		font-weight: 500;
		letter-spacing: 0.02em;
	}

	/* Navigation */
	.portal-nav {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.nav-link {
		display: flex;
		align-items: center;
		gap: 0.45rem;
		padding: 0.55rem 0.9rem;
		border-radius: 10px;
		font-size: 0.88rem;
		font-weight: 600;
		color: #cbd5e1;
		text-decoration: none;
		transition: all 0.2s ease;
		border: 1px solid transparent;
	}

	.nav-link:hover {
		color: #ffffff;
		background: rgba(255, 255, 255, 0.06);
		border-color: rgba(255, 255, 255, 0.1);
	}

	.nav-link.active {
		color: #ffffff;
		background: rgba(2, 132, 199, 0.18);
		border-color: rgba(56, 189, 248, 0.4);
		box-shadow: 0 0 16px rgba(2, 132, 199, 0.2);
	}

	.nav-icon {
		font-size: 1.05rem;
	}

	.nav-pill {
		font-size: 0.65rem;
		font-weight: 700;
		background: #0284c7;
		color: #ffffff;
		padding: 0.15rem 0.45rem;
		border-radius: 999px;
		text-transform: uppercase;
		letter-spacing: 0.04em;
	}

	.partner-pill {
		font-size: 0.65rem;
		font-weight: 700;
		background: rgba(16, 185, 129, 0.2);
		color: #34d399;
		border: 1px solid rgba(52, 211, 153, 0.3);
		padding: 0.15rem 0.45rem;
		border-radius: 999px;
	}

	/* Actions */
	.portal-actions {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}

	.wa-support-btn {
		display: flex;
		align-items: center;
		gap: 0.4rem;
		padding: 0.5rem 0.85rem;
		border-radius: 10px;
		background: rgba(37, 211, 102, 0.15);
		color: #25d366;
		border: 1px solid rgba(37, 211, 102, 0.35);
		font-size: 0.82rem;
		font-weight: 600;
		text-decoration: none;
		transition: all 0.2s ease;
	}

	.wa-support-btn:hover {
		background: rgba(37, 211, 102, 0.25);
		border-color: #25d366;
		transform: translateY(-1px);
	}

	.wa-dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		background: #25d366;
		box-shadow: 0 0 8px #25d366;
	}

	/* Multi-Portal Dropdown */
	.portal-dropdown-wrapper {
		position: relative;
	}

	.portal-dropdown-btn {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.55rem 1rem;
		border-radius: 10px;
		background: linear-gradient(135deg, #4f46e5 0%, #3730a3 100%);
		color: #ffffff;
		border: 1px solid rgba(255, 255, 255, 0.18);
		font-size: 0.88rem;
		font-weight: 700;
		cursor: pointer;
		transition: all 0.2s ease;
		box-shadow: 0 4px 12px rgba(79, 70, 229, 0.35);
	}

	.portal-dropdown-btn:hover, .portal-dropdown-btn.open {
		background: linear-gradient(135deg, #4338ca 0%, #312e81 100%);
		border-color: #a5b4fc;
		box-shadow: 0 6px 18px rgba(79, 70, 229, 0.45);
	}

	.chevron {
		font-size: 0.75rem;
		opacity: 0.8;
		transition: transform 0.2s ease;
	}

	.portal-dropdown-btn.open .chevron {
		transform: rotate(180deg);
	}

	.portal-dropdown-menu {
		position: absolute;
		top: calc(100% + 8px);
		right: 0;
		width: 320px;
		background: #111827;
		border: 1px solid rgba(255, 255, 255, 0.12);
		border-radius: 14px;
		padding: 0.75rem;
		box-shadow: 0 20px 40px rgba(0, 0, 0, 0.6);
		display: flex;
		flex-direction: column;
		gap: 0.4rem;
		animation: menu-fade-in 0.2s cubic-bezier(0.16, 1, 0.3, 1);
	}

	@keyframes menu-fade-in {
		from { opacity: 0; transform: translateY(-6px); }
		to { opacity: 1; transform: translateY(0); }
	}

	.menu-heading {
		font-size: 0.68rem;
		font-weight: 800;
		letter-spacing: 0.08em;
		color: #64748b;
		padding: 0.4rem 0.6rem 0.2rem;
	}

	.portal-item {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		padding: 0.65rem;
		border-radius: 10px;
		text-decoration: none;
		color: #f1f5f9;
		transition: all 0.15s ease;
		background: rgba(255, 255, 255, 0.02);
		border: 1px solid transparent;
	}

	.portal-item:hover {
		background: rgba(255, 255, 255, 0.07);
		border-color: rgba(255, 255, 255, 0.12);
		transform: translateX(2px);
	}

	.portal-item-icon {
		width: 38px;
		height: 38px;
		border-radius: 10px;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 1.15rem;
		flex-shrink: 0;
	}

	.portal-item-icon.traveler { background: rgba(2, 132, 199, 0.2); }
	.portal-item-icon.partner { background: rgba(16, 185, 129, 0.2); }
	.portal-item-icon.crew { background: rgba(245, 158, 11, 0.2); }
	.portal-item-icon.admin { background: rgba(124, 58, 237, 0.2); }

	.portal-item-info {
		flex: 1;
		min-width: 0;
	}

	.portal-item-title {
		font-size: 0.85rem;
		font-weight: 700;
		color: #ffffff;
		line-height: 1.25;
	}

	.portal-item-desc {
		font-size: 0.72rem;
		color: #94a3b8;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
		line-height: 1.3;
	}

	.portal-badge {
		font-size: 0.62rem;
		font-weight: 700;
		padding: 0.18rem 0.45rem;
		border-radius: 6px;
		text-transform: uppercase;
	}

	.portal-badge.customer { background: #0284c7; color: white; }
	.portal-badge.partner { background: #10b981; color: white; }
	.portal-badge.crew { background: #f59e0b; color: white; }
	.portal-badge.admin { background: #6366f1; color: white; }

	.menu-divider {
		height: 1px;
		background: rgba(255, 255, 255, 0.08);
		margin: 0.25rem 0;
	}

	/* Mobile Toggle */
	.mobile-toggle-btn {
		display: none;
		background: rgba(255, 255, 255, 0.08);
		border: 1px solid rgba(255, 255, 255, 0.15);
		color: #ffffff;
		font-size: 1.25rem;
		padding: 0.4rem 0.75rem;
		border-radius: 8px;
		cursor: pointer;
	}

	.mobile-drawer {
		display: none;
		padding: 1rem 1.5rem 1.5rem;
		background: #0b1120;
		border-top: 1px solid rgba(255, 255, 255, 0.08);
	}

	.mobile-links {
		display: flex;
		flex-direction: column;
		gap: 0.6rem;
		margin-bottom: 1.25rem;
	}

	.mobile-link {
		padding: 0.7rem 0.9rem;
		border-radius: 8px;
		background: rgba(255, 255, 255, 0.04);
		color: #f1f5f9;
		text-decoration: none;
		font-size: 0.92rem;
		font-weight: 600;
	}

	.mobile-portals-title {
		font-size: 0.7rem;
		font-weight: 800;
		letter-spacing: 0.06em;
		color: #64748b;
		margin-bottom: 0.6rem;
	}

	.mobile-portals {
		display: grid;
		grid-template-columns: 1fr;
		gap: 0.5rem;
	}

	.mobile-portal-btn {
		padding: 0.65rem 0.9rem;
		border-radius: 8px;
		font-size: 0.85rem;
		font-weight: 700;
		text-decoration: none;
		color: #ffffff;
	}

	.mobile-portal-btn.customer { background: #0284c7; }
	.mobile-portal-btn.partner { background: #059669; }
	.mobile-portal-btn.crew { background: #d97706; }
	.mobile-portal-btn.admin { background: #4f46e5; }

	@media (max-width: 1024px) {
		.portal-nav {
			display: none;
		}
		.mobile-toggle-btn {
			display: block;
		}
		.mobile-drawer {
			display: block;
		}
	}

	@media (max-width: 640px) {
		.wa-text {
			display: none;
		}
		.btn-text {
			display: none;
		}
		.portal-dropdown-btn {
			padding: 0.55rem;
		}
		.portal-dropdown-menu {
			width: 290px;
		}
	}
</style>
