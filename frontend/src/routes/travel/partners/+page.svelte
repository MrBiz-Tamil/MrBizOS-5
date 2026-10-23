<script lang="ts">
	import PortalNavbar from '$lib/components/portal/PortalNavbar.svelte';

	// B2B Earnings Calculator State
	let monthlyTrips = $state(15);
	let avgTripValue = $state(45000); // INR per trip
	let agentTier = $state<'silver' | 'gold' | 'platinum'>('gold');

	let commissionRate = $derived(
		agentTier === 'silver' ? 0.10 :
		agentTier === 'gold' ? 0.125 : 0.15
	);

	let monthlyGrossVolume = $derived(monthlyTrips * avgTripValue);
	let monthlyCommissionEarnings = $derived(Math.round(monthlyGrossVolume * commissionRate));
	let annualCommissionEarnings = $derived(monthlyCommissionEarnings * 12);

	// Onboarding Application Form State
	let agencyName = $state('');
	let contactPerson = $state('');
	let email = $state('');
	let phone = $state('');
	let city = $state('Madurai');
	let state = $state('Tamil Nadu');
	let iataOrGst = $state('');
	let partnerType = $state('Retail Travel Agency');
	let primaryCircuits = $state('Kashmir & Kerala Honeymoons');
	let isSubmitting = $state(false);
	let applicationSuccess = $state(false);
	let generatedPartnerId = $state('');

	function handleRegister(e: Event) {
		e.preventDefault();
		isSubmitting = true;
		setTimeout(() => {
			generatedPartnerId = 'B2B-AGY-' + Math.floor(1000 + Math.random() * 9000);
			isSubmitting = false;
			applicationSuccess = true;
		}, 600);
	}
</script>

<svelte:head>
	<title>Tour & Travel Partners Portal (B2B) | MrBiz OS</title>
	<meta name="description" content="Join 1,200+ travel agencies, corporate desks, and DMCs with wholesale net tariffs, agent khata credit, and up to 15% override commission." />
</svelte:head>

<div class="partners-portal-page">
	<PortalNavbar activeTab="partners" />

	<!-- Hero Section -->
	<section class="partner-hero">
		<div class="hero-glow"></div>
		<div class="hero-content">
			<div class="partner-badge">
				<span class="badge-dot"></span>
				<span>OFFICIAL B2B TRAVEL PARTNER & DMC PROGRAM</span>
			</div>
			<h1 class="hero-title">
				Wholesale Net Tariffs, Instant Vouchers & <br />
				<span class="gradient-text">Sub-Agent Khata Credit</span>
			</h1>
			<p class="hero-desc">
				Powering over 1,200+ travel agents, retail agencies, and corporate travel desks with direct DMC inventory, white-label client itineraries, and up to 15% guaranteed commission payouts.
			</p>

			<div class="hero-stats-row">
				<div class="hero-stat-box">
					<div class="stat-number">1,240+</div>
					<div class="stat-label">Active Travel Partners</div>
				</div>
				<div class="hero-stat-box">
					<div class="stat-number">15%</div>
					<div class="stat-label">Max Override Margin</div>
				</div>
				<div class="hero-stat-box">
					<div class="stat-number">₹10 Lakh</div>
					<div class="stat-label">Instant Khata Credit Line</div>
				</div>
				<div class="hero-stat-box">
					<div class="stat-number">0%</div>
					<div class="stat-label">Walkout Record (DMC Confirmed)</div>
				</div>
			</div>

			<div class="hero-cta-group">
				<a href="#register" class="cta-primary">
					<span>Register as B2B Travel Partner</span>
					<span>➔</span>
				</a>
				<a href="/login/partner" class="cta-secondary">
					<span>Already a Partner? Sign In</span>
				</a>
			</div>
		</div>
	</section>

	<!-- Partner Tiers Section -->
	<section class="tiers-section">
		<div class="section-container">
			<div class="section-header">
				<div class="sec-badge">PARTNERSHIP TIERS</div>
				<h2 class="sec-title">Tailored For Every Stage of Growth</h2>
				<p class="sec-desc">Whether you are an independent travel consultant or an established enterprise DMC, our tier benefits scale with your volume.</p>
			</div>

			<div class="tiers-grid">
				<!-- SILVER TIER -->
				<div class="tier-card">
					<div class="tier-header">
						<div class="tier-icon">🥈</div>
						<div class="tier-badge">Silver Affiliate</div>
					</div>
					<h3 class="tier-title">Independent Agent</h3>
					<div class="commission-display">
						<span class="rate">10%</span>
						<span class="label">Commission Margin</span>
					</div>
					<p class="tier-summary">Ideal for freelance consultants, tour planners, and boutique travel desks.</p>

					<div class="tier-perks">
						<div class="perk-item">✓ Access to live B2B wholesale tariffs</div>
						<div class="perk-item">✓ Instant PDF itinerary generator with your logo</div>
						<div class="perk-item">✓ 24/7 dedicated WhatsApp agent support desk</div>
						<div class="perk-item">✓ Bi-weekly commission settlements</div>
						<div class="perk-item">✓ Zero minimum monthly booking quotas</div>
					</div>

					<a href="#register" class="tier-btn">Join as Silver Partner</a>
				</div>

				<!-- GOLD TIER -->
				<div class="tier-card featured">
					<div class="featured-ribbon">Most Popular</div>
					<div class="tier-header">
						<div class="tier-icon">🥇</div>
						<div class="tier-badge gold">Gold Accredited Agency</div>
					</div>
					<h3 class="tier-title">Retail Travel Agency</h3>
					<div class="commission-display">
						<span class="rate">12.5%</span>
						<span class="label">Commission Margin</span>
					</div>
					<p class="tier-summary">Designed for registered travel agencies with physical offices and regular group volume.</p>

					<div class="tier-perks">
						<div class="perk-item">✓ <strong>₹3,00,000 15-day rolling credit khata</strong></div>
						<div class="perk-item">✓ Priority confirmation for peak season hotel blocks</div>
						<div class="perk-item">✓ Custom placard branding for your guests at airport</div>
						<div class="perk-item">✓ Dedicated Senior Key Account Manager</div>
						<div class="perk-item">✓ Co-branded promotional posters & flyers</div>
					</div>

					<a href="#register" class="tier-btn gold">Apply for Gold Tier</a>
				</div>

				<!-- PLATINUM TIER -->
				<div class="tier-card">
					<div class="tier-header">
						<div class="tier-icon">💎</div>
						<div class="tier-badge platinum">Platinum Master DMC</div>
					</div>
					<h3 class="tier-title">Enterprise & Inbound DMC</h3>
					<div class="commission-display">
						<span class="rate">15%</span>
						<span class="label">Commission Override</span>
					</div>
					<p class="tier-summary">For high-volume operators, franchise networks, and corporate travel management firms.</p>

					<div class="tier-perks">
						<div class="perk-item">✓ <strong>₹10,00,000 monthly credit facility</strong></div>
						<div class="perk-item">✓ Deepest wholesale ground contract tariffs</div>
						<div class="perk-item">✓ Full REST API integration for automated ticketing</div>
						<div class="perk-item">✓ White-labeled customer mobile app & web portal</div>
						<div class="perk-item">✓ VIP luxury vehicle dispatch prioritization</div>
					</div>

					<a href="#register" class="tier-btn">Apply for Platinum DMC</a>
				</div>
			</div>
		</div>
	</section>

	<!-- Interactive B2B Profit Calculator -->
	<section class="calculator-section">
		<div class="section-container">
			<div class="calc-wrapper">
				<div class="calc-left">
					<div class="sec-badge">EARNINGS SIMULATOR</div>
					<h2 class="sec-title">Calculate Your Monthly Agency Revenue</h2>
					<p class="sec-desc">See how much additional profit your agency can generate each month by partnering with MrBiz OS.</p>

					<div class="controls-group">
						<div class="slider-box">
							<div class="slider-top">
								<span class="label">Monthly Booked Trips:</span>
								<span class="val highlight">{monthlyTrips} Trips</span>
							</div>
							<input type="range" min="1" max="100" step="1" bind:value={monthlyTrips} class="slider-ctrl" />
							<div class="slider-labels">
								<span>1 Trip</span>
								<span>50 Trips</span>
								<span>100 Trips</span>
							</div>
						</div>

						<div class="slider-box">
							<div class="slider-top">
								<span class="label">Average Tour Booking Value:</span>
								<span class="val highlight">₹{avgTripValue.toLocaleString('en-IN')}</span>
							</div>
							<input type="range" min="15000" max="150000" step="5000" bind:value={avgTripValue} class="slider-ctrl" />
							<div class="slider-labels">
								<span>₹15,000</span>
								<span>₹75,000</span>
								<span>₹1,50,000</span>
							</div>
						</div>

						<div class="tier-select-row">
							<span class="label">Select Your Agency Tier:</span>
							<div class="tier-btns">
								<button type="button" class="t-btn {agentTier === 'silver' ? 'active' : ''}" onclick={() => (agentTier = 'silver')}>Silver (10%)</button>
								<button type="button" class="t-btn {agentTier === 'gold' ? 'active' : ''}" onclick={() => (agentTier = 'gold')}>Gold (12.5%)</button>
								<button type="button" class="t-btn {agentTier === 'platinum' ? 'active' : ''}" onclick={() => (agentTier = 'platinum')}>Platinum (15%)</button>
							</div>
						</div>
					</div>
				</div>

				<div class="calc-right">
					<div class="earnings-card">
						<div class="card-tag">ESTIMATED PARTNER RETURN</div>
						<div class="est-monthly-title">Monthly Agency Profit</div>
						<div class="est-monthly-val">₹{monthlyCommissionEarnings.toLocaleString('en-IN')}</div>

						<div class="annual-row">
							<span>Projected Annual Earnings:</span>
							<strong>₹{annualCommissionEarnings.toLocaleString('en-IN')}</strong>
						</div>

						<div class="card-divider"></div>

						<div class="volume-metrics">
							<div class="v-metric">
								<span class="m-label">Gross Trip Volume</span>
								<span class="m-val">₹{monthlyGrossVolume.toLocaleString('en-IN')}</span>
							</div>
							<div class="v-metric">
								<span class="m-label">Commission Override</span>
								<span class="m-val highlight">{(commissionRate * 100).toFixed(1)}%</span>
							</div>
						</div>

						<a href="#register" class="claim-earnings-btn">
							<span>Claim Your B2B Account</span>
							<span>➔</span>
						</a>
					</div>
				</div>
			</div>
		</div>
	</section>

	<!-- Partner Network & Supplier Ecosystem -->
	<section class="ecosystem-section">
		<div class="section-container">
			<div class="section-header">
				<div class="sec-badge">OUR NETWORK INFRASTRUCTURE</div>
				<h2 class="sec-title">Direct Contracts Across Key Travel Pillars</h2>
			</div>

			<div class="ecosystem-grid">
				<div class="eco-card">
					<div class="eco-icon">🏨</div>
					<div class="eco-title">450+ Contracted Hotels</div>
					<div class="eco-desc">Direct DMC inventory allocations with Taj, Radisson, Grand Palace, and luxury mountain chalets.</div>
				</div>
				<div class="eco-card">
					<div class="eco-icon">🚗</div>
					<div class="eco-title">180+ Luxury Fleet Units</div>
					<div class="eco-desc">Sanitized Toyota Innova Crystas, Tempo Travellers, and VIP Mercedes with verified chauffeurs.</div>
				</div>
				<div class="eco-card">
					<div class="eco-icon">🚠</div>
					<div class="eco-title">Guaranteed Pass Blocks</div>
					<div class="eco-desc">Pre-reserved slots for Gulmarg Gondola, Burj Khalifa, Dal Lake Shikaras, and desert camps.</div>
				</div>
				<div class="eco-card">
					<div class="eco-icon">📜</div>
					<div class="eco-title">Automated Khata & Invoicing</div>
					<div class="eco-desc">Instant GST compliance, sub-agent credit debits, and one-click reconciliation statements.</div>
				</div>
			</div>
		</div>
	</section>

	<!-- Partner Registration Application Section -->
	<section id="register" class="register-section">
		<div class="section-container">
			<div class="register-card">
				{#if !applicationSuccess}
					<div class="register-header">
						<div class="sec-badge">B2B APPLICATION</div>
						<h2 class="sec-title">Register Your Agency in Under 2 Minutes</h2>
						<p class="sec-desc">Get instant access to wholesale tariffs and launch your agency travel desk today.</p>
					</div>

					<form onsubmit={handleRegister} class="register-form">
						<div class="form-grid-2">
							<div class="f-group">
								<label for="p-agency">Agency / Travel Company Name *</label>
								<input id="p-agency" type="text" required bind:value={agencyName} placeholder="e.g. Madurai Pandian Travel World" class="f-input" />
							</div>
							<div class="f-group">
								<label for="p-contact">Primary Contact Person & Designation *</label>
								<input id="p-contact" type="text" required bind:value={contactPerson} placeholder="e.g. S. Alagarsamy (Managing Partner)" class="f-input" />
							</div>
						</div>

						<div class="form-grid-2">
							<div class="f-group">
								<label for="p-email">Official Agency Email Address *</label>
								<input id="p-email" type="email" required bind:value={email} placeholder="sales@pandiantravels.in" class="f-input" />
							</div>
							<div class="f-group">
								<label for="p-phone">Mobile / WhatsApp Number *</label>
								<input id="p-phone" type="tel" required bind:value={phone} placeholder="+91 94432 88110" class="f-input" />
							</div>
						</div>

						<div class="form-grid-3">
							<div class="f-group">
								<label for="p-city">City *</label>
								<input id="p-city" type="text" required bind:value={city} class="f-input" />
							</div>
							<div class="f-group">
								<label for="p-state">State / Region *</label>
								<input id="p-state" type="text" required bind:value={state} class="f-input" />
							</div>
							<div class="f-group">
								<label for="p-gst">GSTIN / IATA / Trade License</label>
								<input id="p-gst" type="text" bind:value={iataOrGst} placeholder="33AABCM4412F1Z8" class="f-input" />
							</div>
						</div>

						<div class="form-grid-2">
							<div class="f-group">
								<label for="p-type">Business Model</label>
								<select id="p-type" bind:value={partnerType} class="f-input">
									<option value="Retail Travel Agency">Retail Travel Agency</option>
									<option value="Corporate Travel Desk">Corporate Travel Desk</option>
									<option value="B2B Sub-Agent">B2B Sub-Agent</option>
									<option value="Inbound DMC Partner">Inbound DMC Partner</option>
								</select>
							</div>
							<div class="f-group">
								<label for="p-circuits">Primary Markets / Circuits</label>
								<input id="p-circuits" type="text" bind:value={primaryCircuits} placeholder="e.g. Kashmir, Kerala, Dubai, Malaysia" class="f-input" />
							</div>
						</div>

						<div class="form-submit-row">
							<button type="submit" class="submit-partner-btn" disabled={isSubmitting}>
								{#if isSubmitting}
									<span class="spinner"></span>
									<span>Verifying Credentials...</span>
								{:else}
									<span>Submit Partnership Application</span>
									<span>➔</span>
								{/if}
							</button>
						</div>
					</form>
				{:else}
					<div class="app-success-box">
						<div class="success-icon">🎉</div>
						<h2 class="success-title">Partnership Application Approved!</h2>
						<div class="partner-code-badge">PARTNER ID: {generatedPartnerId}</div>
						<p class="success-msg">
							Welcome aboard, <strong>{agencyName || 'Partner'}</strong>! Your B2B partner account has been activated with <strong>Gold Tier (12.5% Commission Margin)</strong>.
						</p>

						<div class="credentials-card">
							<div class="cred-row">
								<span>Assigned Partner Code:</span>
								<strong>{generatedPartnerId}</strong>
							</div>
							<div class="cred-row">
								<span>Approved Credit Line:</span>
								<strong style="color: #34d399;">₹3,00,000 (15-Day Rolling Khata)</strong>
							</div>
							<div class="cred-row">
								<span>Wholesale Portal Status:</span>
								<strong style="color: #38bdf8;">ACTIVE & READY</strong>
							</div>
						</div>

						<div class="success-actions">
							<a href="/login/partner?code={generatedPartnerId}" class="login-partner-btn">
								<span>Proceed to B2B Partner Login</span>
								<span>➔</span>
							</a>
							<a
								href="https://wa.me/919884011223?text=Hi%2C%20we%20have%20registered%20as%20B2B%20partner%20{generatedPartnerId}%20({encodeURIComponent(agencyName)})."
								target="_blank"
								rel="noreferrer"
								class="contact-mgr-btn"
							>
								<span>Connect with Key Account Manager</span>
							</a>
						</div>
					</div>
				{/if}
			</div>
		</div>
	</section>
</div>

<style>
	.partners-portal-page {
		min-height: 100vh;
		background: #090d16;
		color: #f8fafc;
		font-family: var(--font-sans, system-ui, -apple-system, sans-serif);
	}

	/* Hero */
	.partner-hero {
		position: relative;
		padding: 5rem 1.5rem 4rem;
		background: radial-gradient(circle at 50% 15%, rgba(16, 185, 129, 0.15) 0%, rgba(15, 23, 42, 0.95) 75%, #090d16 100%);
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
		text-align: center;
	}

	.hero-glow {
		position: absolute;
		top: -100px;
		left: 50%;
		transform: translateX(-50%);
		width: 750px;
		height: 380px;
		background: radial-gradient(ellipse at center, rgba(16, 185, 129, 0.2) 0%, rgba(2, 132, 199, 0.1) 60%, transparent 80%);
		filter: blur(90px);
		pointer-events: none;
	}

	.hero-content {
		max-width: 1080px;
		margin: 0 auto;
		position: relative;
		z-index: 10;
	}

	.partner-badge {
		display: inline-flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.35rem 0.9rem;
		border-radius: 999px;
		background: rgba(16, 185, 129, 0.15);
		border: 1px solid rgba(52, 211, 153, 0.35);
		color: #34d399;
		font-size: 0.74rem;
		font-weight: 800;
		letter-spacing: 0.08em;
		margin-bottom: 1.25rem;
	}

	.badge-dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		background: #34d399;
		box-shadow: 0 0 10px #34d399;
	}

	.hero-title {
		font-size: clamp(2.2rem, 5vw, 3.4rem);
		font-weight: 900;
		line-height: 1.15;
		margin: 0 0 1.25rem;
		color: #ffffff;
		letter-spacing: -0.02em;
	}

	.gradient-text {
		background: linear-gradient(135deg, #34d399 0%, #38bdf8 60%, #818cf8 100%);
		-webkit-background-clip: text;
		-webkit-text-fill-color: transparent;
	}

	.hero-desc {
		font-size: 1.1rem;
		color: #94a3b8;
		max-width: 750px;
		margin: 0 auto 2.5rem;
		line-height: 1.6;
	}

	.hero-stats-row {
		display: grid;
		grid-template-columns: repeat(4, 1fr);
		gap: 1.25rem;
		max-width: 900px;
		margin: 0 auto 2.5rem;
	}

	.hero-stat-box {
		background: rgba(17, 24, 39, 0.8);
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: 14px;
		padding: 1.25rem 0.75rem;
	}

	.stat-number {
		font-size: 1.8rem;
		font-weight: 900;
		color: #ffffff;
		margin-bottom: 0.25rem;
	}

	.stat-label {
		font-size: 0.76rem;
		color: #94a3b8;
		font-weight: 600;
	}

	.hero-cta-group {
		display: flex;
		justify-content: center;
		gap: 1rem;
		flex-wrap: wrap;
	}

	.cta-primary {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.85rem 1.8rem;
		border-radius: 12px;
		background: linear-gradient(135deg, #10b981 0%, #059669 100%);
		color: #ffffff;
		font-size: 0.96rem;
		font-weight: 800;
		text-decoration: none;
		box-shadow: 0 4px 18px rgba(16, 185, 129, 0.4);
		transition: transform 0.2s;
	}

	.cta-primary:hover {
		transform: translateY(-2px);
	}

	.cta-secondary {
		padding: 0.85rem 1.8rem;
		border-radius: 12px;
		background: rgba(255, 255, 255, 0.08);
		border: 1px solid rgba(255, 255, 255, 0.2);
		color: #ffffff;
		font-size: 0.96rem;
		font-weight: 700;
		text-decoration: none;
	}

	.cta-secondary:hover {
		background: rgba(255, 255, 255, 0.14);
	}

	/* Section Base */
	.section-container {
		max-width: 1240px;
		margin: 0 auto;
		padding: 4.5rem 1.5rem;
	}

	.section-header {
		text-align: center;
		max-width: 700px;
		margin: 0 auto 3rem;
	}

	.sec-badge {
		font-size: 0.72rem;
		font-weight: 800;
		letter-spacing: 0.1em;
		color: #34d399;
		margin-bottom: 0.4rem;
	}

	.sec-title {
		font-size: clamp(1.8rem, 3vw, 2.3rem);
		font-weight: 800;
		color: #ffffff;
		margin: 0 0 0.75rem;
		letter-spacing: -0.02em;
	}

	.sec-desc {
		font-size: 0.96rem;
		color: #94a3b8;
		line-height: 1.55;
		margin: 0;
	}

	/* Tiers Grid */
	.tiers-grid {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 1.5rem;
		align-items: stretch;
	}

	.tier-card {
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 18px;
		padding: 2rem;
		display: flex;
		flex-direction: column;
		position: relative;
		transition: all 0.2s ease;
	}

	.tier-card.featured {
		border-color: #34d399;
		background: linear-gradient(180deg, rgba(16, 185, 129, 0.08) 0%, #0f172a 100%);
		box-shadow: 0 16px 40px rgba(16, 185, 129, 0.18);
		transform: scale(1.03);
	}

	.featured-ribbon {
		position: absolute;
		top: -12px;
		right: 24px;
		background: #10b981;
		color: #ffffff;
		font-size: 0.7rem;
		font-weight: 800;
		padding: 0.2rem 0.65rem;
		border-radius: 999px;
		text-transform: uppercase;
		letter-spacing: 0.04em;
	}

	.tier-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 1rem;
	}

	.tier-icon {
		font-size: 2.2rem;
	}

	.tier-badge {
		font-size: 0.72rem;
		font-weight: 800;
		padding: 0.25rem 0.6rem;
		border-radius: 999px;
		background: rgba(255, 255, 255, 0.08);
		color: #cbd5e1;
	}

	.tier-badge.gold {
		background: rgba(245, 158, 11, 0.2);
		color: #f59e0b;
		border: 1px solid rgba(245, 158, 11, 0.4);
	}

	.tier-badge.platinum {
		background: rgba(139, 92, 246, 0.2);
		color: #a78bfa;
		border: 1px solid rgba(139, 92, 246, 0.4);
	}

	.tier-title {
		font-size: 1.35rem;
		font-weight: 800;
		color: #ffffff;
		margin: 0 0 0.85rem;
	}

	.commission-display {
		background: rgba(255, 255, 255, 0.03);
		border-radius: 10px;
		padding: 0.75rem 1rem;
		display: flex;
		align-items: baseline;
		gap: 0.5rem;
		margin-bottom: 1rem;
	}

	.commission-display .rate {
		font-size: 2rem;
		font-weight: 900;
		color: #34d399;
	}

	.commission-display .label {
		font-size: 0.78rem;
		color: #94a3b8;
		font-weight: 600;
	}

	.tier-summary {
		font-size: 0.85rem;
		color: #94a3b8;
		line-height: 1.5;
		margin: 0 0 1.5rem;
	}

	.tier-perks {
		display: flex;
		flex-direction: column;
		gap: 0.65rem;
		margin-bottom: 2rem;
		flex: 1;
	}

	.perk-item {
		font-size: 0.82rem;
		color: #cbd5e1;
		line-height: 1.4;
	}

	.tier-btn {
		display: block;
		text-align: center;
		padding: 0.75rem;
		border-radius: 10px;
		background: rgba(255, 255, 255, 0.08);
		border: 1px solid rgba(255, 255, 255, 0.15);
		color: #ffffff;
		font-weight: 700;
		font-size: 0.88rem;
		text-decoration: none;
		transition: all 0.2s;
	}

	.tier-btn:hover {
		background: rgba(255, 255, 255, 0.15);
	}

	.tier-btn.gold {
		background: linear-gradient(135deg, #10b981 0%, #059669 100%);
		border: none;
		box-shadow: 0 4px 14px rgba(16, 185, 129, 0.4);
	}

	.tier-btn.gold:hover {
		background: linear-gradient(135deg, #059669 0%, #047857 100%);
	}

	/* Calculator Section */
	.calculator-section {
		background: #0b1120;
		border-top: 1px solid rgba(255, 255, 255, 0.06);
		border-bottom: 1px solid rgba(255, 255, 255, 0.06);
	}

	.calc-wrapper {
		display: grid;
		grid-template-columns: 1fr 420px;
		gap: 2.5rem;
		align-items: center;
	}

	.controls-group {
		display: flex;
		flex-direction: column;
		gap: 1.5rem;
		margin-top: 1.5rem;
	}

	.slider-box {
		background: rgba(255, 255, 255, 0.02);
		border: 1px solid rgba(255, 255, 255, 0.06);
		border-radius: 12px;
		padding: 1.25rem;
	}

	.slider-top {
		display: flex;
		justify-content: space-between;
		margin-bottom: 0.75rem;
		font-size: 0.9rem;
	}

	.slider-top .label { color: #cbd5e1; font-weight: 600; }
	.slider-top .highlight { color: #34d399; font-weight: 800; font-size: 1.05rem; }

	.slider-ctrl {
		width: 100%;
		accent-color: #10b981;
		cursor: pointer;
	}

	.slider-labels {
		display: flex;
		justify-content: space-between;
		font-size: 0.72rem;
		color: #64748b;
		margin-top: 0.4rem;
	}

	.tier-select-row {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.tier-select-row .label {
		font-size: 0.85rem;
		color: #94a3b8;
		font-weight: 600;
	}

	.tier-btns {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 0.5rem;
	}

	.t-btn {
		padding: 0.65rem;
		border-radius: 8px;
		background: rgba(255, 255, 255, 0.05);
		border: 1px solid rgba(255, 255, 255, 0.1);
		color: #cbd5e1;
		font-weight: 700;
		font-size: 0.82rem;
		cursor: pointer;
	}

	.t-btn.active {
		background: rgba(16, 185, 129, 0.2);
		border-color: #34d399;
		color: #ffffff;
	}

	.earnings-card {
		background: #0f172a;
		border: 2px solid #34d399;
		border-radius: 20px;
		padding: 2rem;
		box-shadow: 0 20px 50px rgba(16, 185, 129, 0.2);
	}

	.card-tag {
		font-size: 0.7rem;
		font-weight: 800;
		color: #34d399;
		letter-spacing: 0.06em;
		margin-bottom: 0.35rem;
	}

	.est-monthly-title {
		font-size: 0.95rem;
		color: #cbd5e1;
	}

	.est-monthly-val {
		font-size: 2.6rem;
		font-weight: 900;
		color: #ffffff;
		margin: 0.25rem 0 1rem;
	}

	.annual-row {
		display: flex;
		justify-content: space-between;
		font-size: 0.85rem;
		color: #94a3b8;
	}

	.annual-row strong {
		color: #34d399;
		font-size: 1rem;
	}

	.card-divider {
		height: 1px;
		background: rgba(255, 255, 255, 0.08);
		margin: 1.25rem 0;
	}

	.volume-metrics {
		display: flex;
		justify-content: space-between;
		margin-bottom: 1.5rem;
	}

	.v-metric {
		display: flex;
		flex-direction: column;
		gap: 0.2rem;
	}

	.m-label { font-size: 0.72rem; color: #64748b; }
	.m-val { font-size: 0.95rem; font-weight: 700; color: #ffffff; }
	.m-val.highlight { color: #34d399; }

	.claim-earnings-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.5rem;
		padding: 0.85rem;
		border-radius: 12px;
		background: linear-gradient(135deg, #10b981 0%, #059669 100%);
		color: #ffffff;
		font-weight: 800;
		font-size: 0.95rem;
		text-decoration: none;
	}

	/* Ecosystem Grid */
	.ecosystem-grid {
		display: grid;
		grid-template-columns: repeat(4, 1fr);
		gap: 1.5rem;
	}

	.eco-card {
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 16px;
		padding: 1.5rem;
	}

	.eco-icon {
		font-size: 2rem;
		margin-bottom: 0.75rem;
	}

	.eco-title {
		font-size: 1.05rem;
		font-weight: 700;
		color: #ffffff;
		margin-bottom: 0.45rem;
	}

	.eco-desc {
		font-size: 0.82rem;
		color: #94a3b8;
		line-height: 1.45;
	}

	/* Registration Section */
	.register-card {
		max-width: 840px;
		margin: 0 auto;
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.12);
		border-radius: 20px;
		padding: 2.5rem;
		box-shadow: 0 20px 50px rgba(0, 0, 0, 0.5);
	}

	.register-header {
		text-align: center;
		margin-bottom: 2rem;
	}

	.register-form {
		display: flex;
		flex-direction: column;
		gap: 1.25rem;
	}

	.form-grid-2 {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 1.25rem;
	}

	.form-grid-3 {
		display: grid;
		grid-template-columns: 1fr 1fr 1.2fr;
		gap: 1.25rem;
	}

	.f-group {
		display: flex;
		flex-direction: column;
		gap: 0.4rem;
	}

	.f-group label {
		font-size: 0.78rem;
		font-weight: 700;
		color: #cbd5e1;
	}

	.f-input {
		width: 100%;
		box-sizing: border-box;
		padding: 0.75rem 0.9rem;
		background: #090d16;
		border: 1px solid rgba(255, 255, 255, 0.14);
		border-radius: 10px;
		color: #ffffff;
		font-size: 0.9rem;
		outline: none;
	}

	.f-input:focus {
		border-color: #34d399;
	}

	.form-submit-row {
		margin-top: 1rem;
	}

	.submit-partner-btn {
		width: 100%;
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.5rem;
		padding: 0.95rem;
		border-radius: 12px;
		background: linear-gradient(135deg, #10b981 0%, #059669 100%);
		border: none;
		color: #ffffff;
		font-size: 1rem;
		font-weight: 800;
		cursor: pointer;
		box-shadow: 0 4px 18px rgba(16, 185, 129, 0.4);
	}

	.spinner {
		width: 16px;
		height: 16px;
		border: 2px solid rgba(255, 255, 255, 0.3);
		border-top-color: #ffffff;
		border-radius: 50%;
		animation: spin 0.7s linear infinite;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	/* Success Box */
	.app-success-box {
		text-align: center;
		padding: 1.5rem 0;
	}

	.success-icon {
		font-size: 3.5rem;
		margin-bottom: 0.5rem;
	}

	.success-title {
		font-size: 1.8rem;
		font-weight: 800;
		color: #ffffff;
		margin: 0 0 0.5rem;
	}

	.partner-code-badge {
		display: inline-block;
		padding: 0.4rem 1rem;
		border-radius: 999px;
		background: rgba(16, 185, 129, 0.2);
		border: 1px solid #10b981;
		color: #34d399;
		font-size: 0.9rem;
		font-weight: 800;
		margin-bottom: 1rem;
	}

	.success-msg {
		font-size: 0.95rem;
		color: #cbd5e1;
		max-width: 550px;
		margin: 0 auto 1.5rem;
		line-height: 1.5;
	}

	.credentials-card {
		background: #090d16;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 12px;
		padding: 1.25rem;
		max-width: 480px;
		margin: 0 auto 2rem;
		display: flex;
		flex-direction: column;
		gap: 0.65rem;
		text-align: left;
	}

	.cred-row {
		display: flex;
		justify-content: space-between;
		font-size: 0.86rem;
		color: #94a3b8;
	}

	.success-actions {
		display: flex;
		justify-content: center;
		gap: 1rem;
		flex-wrap: wrap;
	}

	.login-partner-btn {
		padding: 0.85rem 1.6rem;
		border-radius: 10px;
		background: linear-gradient(135deg, #10b981 0%, #059669 100%);
		color: #ffffff;
		font-weight: 700;
		text-decoration: none;
	}

	.contact-mgr-btn {
		padding: 0.85rem 1.6rem;
		border-radius: 10px;
		background: rgba(255, 255, 255, 0.08);
		color: #ffffff;
		font-weight: 600;
		text-decoration: none;
	}

	@media (max-width: 1024px) {
		.hero-stats-row {
			grid-template-columns: repeat(2, 1fr);
		}
		.tiers-grid {
			grid-template-columns: 1fr;
		}
		.calc-wrapper {
			grid-template-columns: 1fr;
		}
		.ecosystem-grid {
			grid-template-columns: repeat(2, 1fr);
		}
		.form-grid-2, .form-grid-3 {
			grid-template-columns: 1fr;
		}
	}
</style>
