<script lang="ts">
	import PortalNavbar from '$lib/components/portal/PortalNavbar.svelte';

	interface ServiceCard {
		id: string;
		title: string;
		category: string;
		icon: string;
		badge: string;
		badgeColor: string;
		description: string;
		features: string[];
		leadTime: string;
		startingPrice: string;
	}

	const services: ServiceCard[] = [
		{
			id: 'airport-chauffeur',
			title: 'Airport VIP Meet & Chauffeur Transfer',
			category: 'GROUND MOBILITY',
			icon: '🚗',
			badge: 'Zero Wait',
			badgeColor: '#0284c7',
			description: 'Premium on-ground private airport arrival and departure transfers with personalized name placard greeting and sanitized luxury vehicles.',
			features: ['Flight delay auto-tracking', 'Toyota Innova Crysta / Luxury Sedans', 'Uniformed, background-verified chauffeurs', 'Complimentary bottled water & luggage assist'],
			leadTime: 'Instant / 2hr Notice',
			startingPrice: '₹1,850 per transfer'
		},
		{
			id: 'bespoke-holidays',
			title: 'Customized Holiday & Tour Packages',
			category: 'TOUR PACKAGES',
			icon: '🌴',
			badge: 'Handcrafted',
			badgeColor: '#10b981',
			description: 'Bespoke end-to-end itineraries covering Kashmir, Kerala, Dubai, Bali, and Golden Triangle with guaranteed hotel vouchers and private sightseeing.',
			features: ['Flexible day-by-day customization', '3★, 4★ & 5★ luxury heritage resort options', 'Daily breakfast & dinner included (MAP)', 'Dedicated 24/7 on-trip trip manager'],
			leadTime: 'Instant Itinerary PDF',
			startingPrice: '₹14,999 per person'
		},
		{
			id: 'visa-expedite',
			title: 'Express Visa & Document Expediting',
			category: 'DOCUMENTATION',
			icon: '🛂',
			badge: '99.4% Approval',
			badgeColor: '#8b5cf6',
			description: 'Hassle-free e-Visa and sticker visa processing for Dubai (UAE), Malaysia, Singapore, Thailand, Schengen Europe, and UK with document verification.',
			features: ['24-hour express UAE Tourist Visa', 'Biometrics & appointment slot assistance', 'Embassy-compliant cover letter drafting', 'Instant WhatsApp status notifications'],
			leadTime: '24 to 72 Hours',
			startingPrice: '₹2,400 + Govt Fees'
		},
		{
			id: 'luxury-stays',
			title: 'Curated Hotels, Villas & Houseboats',
			category: 'HOSPITALITY',
			icon: '🏨',
			badge: 'DMC Direct Rates',
			badgeColor: '#f59e0b',
			description: 'Direct DMC contracted rates at handpicked luxury riverfront resorts, Dal Lake heritage houseboats, private pool villas, and mountain chalets.',
			features: ['Guaranteed early check-in & late checkout', 'Free room upgrade subject to availability', 'Verified hygiene and central heating standards', 'Direct hotel confirmation voucher in 60s'],
			leadTime: 'Immediate Voucher',
			startingPrice: '₹3,800 per night'
		},
		{
			id: 'activity-passes',
			title: 'Sightseeing & Adventure Passes',
			category: 'EXPERIENCES',
			icon: '🚠',
			badge: 'Skip The Line',
			badgeColor: '#ec4899',
			description: 'Pre-booked skip-the-line passes for high-demand tourist experiences including Gulmarg Gondola Phase 1 & 2, Shikara rides, Desert Safaris, and Scuba Diving.',
			features: ['Guaranteed confirmed cable car time-slots', 'Private 1-hour sunset Dal Lake Shikara ride', 'Certified instructors & equipment safety gear', 'Instant mobile QR entry voucher'],
			leadTime: 'Instant QR Voucher',
			startingPrice: '₹950 per pass'
		},
		{
			id: 'flight-rail',
			title: 'Flight Bookings & Group Fare Blocks',
			category: 'TICKETING',
			icon: '✈️',
			badge: 'Special Group Fare',
			badgeColor: '#06b6d4',
			description: 'Competitive domestic and international air ticketing with special corporate baggage allowances and dedicated date change flex-protection.',
			features: ['20kg extra checked baggage allowance options', 'Web check-in & seat selection assistance', 'Direct airline PNR generation', 'Full refund protection on medical emergencies'],
			leadTime: 'Instant PNR',
			startingPrice: 'Live Airline Net Fare'
		},
		{
			id: 'licensed-guides',
			title: 'Multilingual Licensed Tour Guides',
			category: 'GUIDED TOURS',
			icon: '🗣️',
			badge: 'Govt. Licensed',
			badgeColor: '#6366f1',
			description: 'Department of Tourism certified local guides speaking English, Hindi, Tamil, Telugu, Arabic, and French for cultural and historical immersion.',
			features: ['Certified historical & architectural expertise', 'Family-friendly and safety-first escort', 'Photography assistance at scenic spots', 'Customized pace for seniors and children'],
			leadTime: '12hr Prior Confirmation',
			startingPrice: '₹1,500 per day'
		},
		{
			id: 'emergency-sos',
			title: '24/7 On-Trip Emergency & SOS Dispatch',
			category: 'TRAVEL CARE',
			icon: '🛡️',
			badge: 'Active 24/7',
			badgeColor: '#ef4444',
			description: 'Round-the-clock emergency desk providing rapid response for flight cancellations, sudden medical doctor-on-call, vehicle breakdown replacements.',
			features: ['Sub-15 minute emergency response time', 'Immediate replacement vehicle dispatch', 'Network of affiliated hospitals & tele-consult', 'Coordinated consular & police liaison'],
			leadTime: 'Instant SOS Hotline',
			startingPrice: 'Complimentary with Trips'
		}
	];

	// PNR Tracking State
	let pnrInput = $state('MB-TRV-8849');
	let pnrLoading = $state(false);
	let pnrResult = $state<any>(null);
	let pnrSearched = $state(false);

	// Service Request Modal State
	let showServiceModal = $state(false);
	let selectedService = $state<ServiceCard | null>(null);
	let clientName = $state('');
	let clientPhone = $state('');
	let clientEmail = $state('');
	let clientTravelDate = $state('2026-11-20');
	let clientGuestCount = $state(2);
	let clientDestination = $state('Kashmir Valley');
	let clientNotes = $state('');
	let requestSubmitted = $state(false);
	let generatedTicketId = $state('');

	function openServiceRequest(service: ServiceCard) {
		selectedService = service;
		requestSubmitted = false;
		showServiceModal = true;
	}

	function handleServiceSubmit(e: Event) {
		e.preventDefault();
		generatedTicketId = 'SRV-' + Math.floor(100000 + Math.random() * 900000);
		requestSubmitted = true;
	}

	function lookupPnr() {
		if (!pnrInput.trim()) return;
		pnrLoading = true;
		pnrSearched = true;

		setTimeout(() => {
			const query = pnrInput.trim().toUpperCase();
			if (query.includes('8849') || query.includes('TRV')) {
				pnrResult = {
					found: true,
					pnr: query,
					guestName: 'Rahul Sharma & Family (2 Adults, 2 Kids)',
					packageTitle: 'Kashmir Valley Paradise & Gulmarg Snow (4D/5N)',
					travelDates: '14 Nov 2026 – 19 Nov 2026',
					status: 'CONFIRMED & DISPATCH READY',
					statusColor: '#10b981',
					flight: 'IndiGo 6E-2144 (MAA ➔ SXR 12:30)',
					chauffeur: 'Ghulam Nabi (+91 94190 22334) | Toyota Innova Crysta JK-01-AB-4491',
					hotel: 'Hotel Grand Kashmir (Pahalgam) & Boulevard Palace (Srinagar)',
					mealPlan: 'MAP Plan (Daily Breakfast + Dinner)',
					gondolaPasses: 'Phase 1 & Phase 2 Confirmed (Slot: 10:30 AM)',
					totalPaid: '₹68,500 (100% Cleared)',
					coordinator: 'Bashir Ahmed (+91 94190 55667)'
				};
			} else {
				pnrResult = {
					found: true,
					pnr: query,
					guestName: 'Ananya Verma (Couple)',
					packageTitle: 'Kerala Backwaters & Munnar Tea Mist (5D/6N)',
					travelDates: '22 Nov 2026 – 28 Nov 2026',
					status: 'VOUCHERS ISSUED',
					statusColor: '#0284c7',
					flight: 'Air India Express IX-441 (DEL ➔ COK 11:15)',
					chauffeur: 'K. Sasi Kumar (+91 98470 12345) | Toyota Etios AC KL-07-CD-8910',
					hotel: 'Blanket Luxury Munnar & Kumarakom Lake Houseboat',
					mealPlan: 'CPAI + Houseboat All Meals',
					gondolaPasses: 'Eravikulam Park Safari Passes Confirmed',
					totalPaid: '₹54,200 (100% Cleared)',
					coordinator: 'Deepak Nair (+91 98470 66789)'
				};
			}
			pnrLoading = false;
		}, 400);
	}

	// FAQ Accordion
	let openFaq = $state<number | null>(0);
	function toggleFaq(index: number) {
		openFaq = openFaq === index ? null : index;
	}

	const faqs = [
		{
			q: 'How do I receive my hotel vouchers, flight tickets, and cab details?',
			a: 'Immediately upon booking confirmation, all digital travel vouchers, hotel booking codes, flight e-tickets, and chauffeur details are sent directly to your WhatsApp and registered email. You can also view and download them anytime on your Customer Portal using your PNR or mobile number.'
		},
		{
			q: 'What if my flight arrives late at the airport?',
			a: 'Our dispatch desk automatically monitors live flight arrival radars. Even if your flight is delayed by several hours, your assigned private chauffeur will adjust pickup timing without any waiting penalties or extra surcharges.'
		},
		{
			q: 'Are Gulmarg Gondola and special monument tickets guaranteed?',
			a: 'Yes. We secure confirmed Department of Tourism cable car time-slots in advance so you can skip long physical queues and bypass tourist congestion.'
		},
		{
			q: 'Can I customize the day-wise sightseeing plan while on the trip?',
			a: 'Absolutely. You have a dedicated 24/7 personal tour coordinator assigned to your booking. If you wish to spend more time at a scenic viewpoint or swap a day’s activity, simply let your coordinator know.'
		},
		{
			q: 'What are the emergency support options if medical or logistics issues arise?',
			a: 'Our emergency SOS helpline is monitored 24/7. In the rare event of vehicle breakdown, a replacement car is dispatched immediately. We also maintain verified on-call doctor tie-ups at every major tourist hub.'
		}
	];
</script>

<svelte:head>
	<title>Customer Travel & Concierge Services | MrBiz OS</title>
	<meta name="description" content="Explore premium customer-side travel services, chauffeur airport transfers, custom holiday packages, visa assistance, and live PNR trip tracking." />
</svelte:head>

<div class="customer-portal-page">
	<!-- Shared Navigation Header -->
	<PortalNavbar activeTab="services" />

	<!-- Hero Section -->
	<section class="portal-hero">
		<div class="hero-glow-orbit"></div>
		<div class="hero-container">
			<div class="hero-badge">
				<span class="badge-dot"></span>
				<span>24/7 TRAVELER CONCIERGE & ON-GROUND DISPATCH</span>
			</div>
			<h1 class="hero-title">
				Effortless Holiday Planning & <br />
				<span class="gradient-text">World-Class Travel Services</span>
			</h1>
			<p class="hero-desc">
				From VIP airport chauffeurs and private luxury itineraries to fast-track cable car passes and 24/7 emergency care — everything you need for an unforgettable journey.
			</p>

			<!-- PNR Quick Lookup Tracker Card -->
			<div class="pnr-tracker-card">
				<div class="tracker-header">
					<div class="tracker-title">
						<span class="icon">🔍</span>
						<span>Track Your Trip, Vouchers & Assigned Chauffeur</span>
					</div>
					<div class="demo-pnr-hints">
						<span>Try Demo:</span>
						<button type="button" class="hint-btn" onclick={() => { pnrInput = 'MB-TRV-8849'; lookupPnr(); }}>Kashmir Trip (MB-TRV-8849)</button>
						<button type="button" class="hint-btn" onclick={() => { pnrInput = 'KL-KER-3310'; lookupPnr(); }}>Kerala Trip (KL-KER-3310)</button>
					</div>
				</div>

				<div class="tracker-input-row">
					<div class="input-with-icon">
						<span class="input-icon">🎫</span>
						<input
							type="text"
							bind:value={pnrInput}
							placeholder="Enter Booking ID / PNR / Mobile Number"
							class="pnr-input"
							onkeydown={(e) => e.key === 'Enter' && lookupPnr()}
						/>
					</div>
					<button type="button" class="lookup-btn" onclick={lookupPnr} disabled={pnrLoading}>
						{#if pnrLoading}
							<span class="spinner"></span>
							<span>Searching Radar...</span>
						{:else}
							<span>Find My Trip</span>
							<span class="arrow">➔</span>
						{/if}
					</button>
				</div>

				<!-- Live PNR Result Card -->
				{#if pnrSearched && pnrResult}
					<div class="pnr-result-box">
						<div class="result-top">
							<div>
								<div class="result-pnr-badge">BOOKING REF: {pnrResult.pnr}</div>
								<h3 class="result-trip-title">{pnrResult.packageTitle}</h3>
								<div class="result-guest">{pnrResult.guestName} • 📅 {pnrResult.travelDates}</div>
							</div>
							<div class="status-pill" style="border-color: {pnrResult.statusColor}; color: {pnrResult.statusColor};">
								<span class="status-dot" style="background: {pnrResult.statusColor};"></span>
								<span>{pnrResult.status}</span>
							</div>
						</div>

						<div class="result-details-grid">
							<div class="detail-cell">
								<div class="detail-label">🚗 Assigned Chauffeur & Vehicle</div>
								<div class="detail-value highlight">{pnrResult.chauffeur}</div>
							</div>
							<div class="detail-cell">
								<div class="detail-label">🏨 Confirmed Stays</div>
								<div class="detail-value">{pnrResult.hotel} ({pnrResult.mealPlan})</div>
							</div>
							<div class="detail-cell">
								<div class="detail-label">✈️ Inbound Flight / Arrival</div>
								<div class="detail-value">{pnrResult.flight}</div>
							</div>
							<div class="detail-cell">
								<div class="detail-label">🚠 Fast-Track Passes</div>
								<div class="detail-value">{pnrResult.gondolaPasses}</div>
							</div>
						</div>

						<div class="result-actions">
							<a href="/login/customer" class="action-btn-primary">
								<span>Open Traveler Dashboard</span>
								<span>➔</span>
							</a>
							<a
								href="https://wa.me/919419022334?text=Hi%20Ghulam%20Nabi%2C%20I%20am%20guest%20{encodeURIComponent(pnrResult.guestName)}%20arriving%20with%20PNR%20{pnrResult.pnr}"
								target="_blank"
								rel="noreferrer"
								class="action-btn-wa"
							>
								<span>WhatsApp Chauffeur Directly</span>
							</a>
						</div>
					</div>
				{/if}
			</div>

			<!-- Quick Navigation Action Bar -->
			<div class="hero-action-pills">
				<a href="/travel/booking" class="hero-cta-btn primary">
					<span class="btn-icon">🗺️</span>
					<span>Custom Tour Booking Wizard</span>
				</a>
				<a href="/travel/packages" class="hero-cta-btn secondary">
					<span class="btn-icon">🌴</span>
					<span>Explore 15+ Holiday Circuits</span>
				</a>
				<a href="/travel/partners" class="hero-cta-btn partner">
					<span class="btn-icon">🤝</span>
					<span>B2B Partner & Agency Portal</span>
				</a>
			</div>
		</div>
	</section>

	<!-- Services Catalogue Section -->
	<section class="services-section">
		<div class="section-container">
			<div class="section-header">
				<div class="section-badge">SERVICES DIRECTORY</div>
				<h2 class="section-title">Comprehensive Travel & Hospitality Services</h2>
				<p class="section-subtitle">
					Choose from our à la carte customer offerings or combine them into a complete bespoke holiday package.
				</p>
			</div>

			<div class="services-grid">
				{#each services as s}
					<div class="service-card">
						<div class="card-header">
							<div class="service-icon-box">{s.icon}</div>
							<span class="service-badge" style="background: {s.badgeColor}20; color: {s.badgeColor}; border: 1px solid {s.badgeColor}50;">
								{s.badge}
							</span>
						</div>

						<div class="service-category">{s.category}</div>
						<h3 class="service-title">{s.title}</h3>
						<p class="service-desc">{s.description}</p>

						<div class="feature-list">
							{#each s.features as feat}
								<div class="feature-item">
									<span class="check-icon">✓</span>
									<span>{feat}</span>
								</div>
							{/each}
						</div>

						<div class="card-footer">
							<div class="pricing-info">
								<span class="price-label">Starting at</span>
								<span class="price-val">{s.startingPrice}</span>
								<span class="lead-time">⚡ {s.leadTime}</span>
							</div>
							<button
								type="button"
								class="request-service-btn"
								onclick={() => openServiceRequest(s)}
							>
								<span>Request Service</span>
								<span>➔</span>
							</button>
						</div>
					</div>
				{/each}
			</div>
		</div>
	</section>

	<!-- Why Choose Us / Trust Badges -->
	<section class="trust-section">
		<div class="trust-container">
			<div class="trust-grid">
				<div class="trust-item">
					<div class="trust-icon">🛡️</div>
					<div class="trust-title">100% Guaranteed Vouchers</div>
					<div class="trust-desc">All hotels, flights, and passes are secured via direct DMC tie-ups with zero walkouts.</div>
				</div>
				<div class="trust-item">
					<div class="trust-icon">🚗</div>
					<div class="trust-title">Verified Luxury Fleet</div>
					<div class="trust-desc">GPS-tracked vehicles with vetted, polite chauffeurs and prompt replacements.</div>
				</div>
				<div class="trust-item">
					<div class="trust-icon">⚡</div>
					<div class="trust-title">Sub-15m Emergency SOS</div>
					<div class="trust-desc">Dedicated round-the-clock trip coordinators ready for any itinerary adjustment.</div>
				</div>
				<div class="trust-item">
					<div class="trust-icon">💳</div>
					<div class="trust-title">Transparent Pricing & GST</div>
					<div class="trust-desc">No hidden driver allowances or night surcharges. Transparent tax invoices.</div>
				</div>
			</div>
		</div>
	</section>

	<!-- FAQ Section -->
	<section class="faq-section">
		<div class="faq-container">
			<div class="section-header">
				<div class="section-badge">FREQUENTLY ASKED QUESTIONS</div>
				<h2 class="section-title">Everything You Need To Know</h2>
			</div>

			<div class="faq-list">
				{#each faqs as faq, i}
					<div class="faq-card {openFaq === i ? 'open' : ''}">
						<button type="button" class="faq-question" onclick={() => toggleFaq(i)}>
							<span>{faq.q}</span>
							<span class="faq-icon">{openFaq === i ? '−' : '+'}</span>
						</button>
						{#if openFaq === i}
							<div class="faq-answer">
								{faq.a}
							</div>
						{/if}
					</div>
				{/each}
			</div>
		</div>
	</section>

	<!-- Footer CTA Banner -->
	<section class="portal-cta-banner">
		<div class="cta-inner">
			<h2>Ready to Plan Your Dream Journey?</h2>
			<p>Customize your flights, hotels, luxury cabs, and passes with our interactive tour booking tool.</p>
			<div class="cta-buttons">
				<a href="/travel/booking" class="banner-btn primary">
					<span>Book Your Tour Package Now</span>
					<span>➔</span>
				</a>
				<a href="/login/customer" class="banner-btn secondary">
					<span>Sign In To Traveler Account</span>
				</a>
				<a href="/travel/partners" class="banner-btn partner">
					<span>Become a B2B Travel Partner</span>
				</a>
			</div>
		</div>
	</section>
</div>

<!-- Service Request Modal -->
{#if showServiceModal && selectedService}
	<div class="modal-backdrop" onclick={() => (showServiceModal = false)}>
		<div class="service-modal-box" onclick={(e) => e.stopPropagation()}>
			<button type="button" class="modal-close-btn" onclick={() => (showServiceModal = false)}>✕</button>

			{#if !requestSubmitted}
				<div class="modal-header">
					<div class="modal-service-badge">{selectedService.icon} {selectedService.category}</div>
					<h3 class="modal-title">Request: {selectedService.title}</h3>
					<p class="modal-subtitle">Our 24/7 travel desk will confirm availability and dispatch options within 15 minutes.</p>
				</div>

				<form onsubmit={handleServiceSubmit} class="modal-form">
					<div class="form-row">
						<div class="form-group">
							<label for="c-name">Full Name *</label>
							<input id="c-name" type="text" bind:value={clientName} required placeholder="e.g. Ananya Verma" class="form-input" />
						</div>
						<div class="form-group">
							<label for="c-phone">Phone / WhatsApp Number *</label>
							<input id="c-phone" type="tel" bind:value={clientPhone} required placeholder="+91 98840 12345" class="form-input" />
						</div>
					</div>

					<div class="form-row">
						<div class="form-group">
							<label for="c-email">Email Address</label>
							<input id="c-email" type="email" bind:value={clientEmail} placeholder="ananya@example.com" class="form-input" />
						</div>
						<div class="form-group">
							<label for="c-dest">Destination / City</label>
							<input id="c-dest" type="text" bind:value={clientDestination} placeholder="e.g. Kashmir / Srinagar" class="form-input" />
						</div>
					</div>

					<div class="form-row">
						<div class="form-group">
							<label for="c-date">Preferred Date</label>
							<input id="c-date" type="date" bind:value={clientTravelDate} class="form-input" />
						</div>
						<div class="form-group">
							<label for="c-guests">Number of Guests</label>
							<input id="c-guests" type="number" min="1" max="50" bind:value={clientGuestCount} class="form-input" />
						</div>
					</div>

					<div class="form-group">
						<label for="c-notes">Special Requests / Flight Timing / Cab Preference</label>
						<textarea id="c-notes" bind:value={clientNotes} rows="3" placeholder="e.g. Arriving on 6E-2144 at 12:30pm, require child booster seat and heated room..." class="form-textarea"></textarea>
					</div>

					<div class="price-estimate-box">
						<div class="est-label">Estimated Service Starting Tariff</div>
						<div class="est-amount">{selectedService.startingPrice}</div>
						<div class="est-note">Includes vehicle fuel, tolls, chauffeur fee, and applicable GST.</div>
					</div>

					<div class="modal-actions">
						<button type="button" class="btn-cancel" onclick={() => (showServiceModal = false)}>Cancel</button>
						<button type="submit" class="btn-confirm-submit">
							<span>Submit Service Request</span>
							<span>➔</span>
						</button>
					</div>
				</form>
			{:else}
				<div class="success-box">
					<div class="success-icon">🎉</div>
					<h3>Service Request Confirmed!</h3>
					<div class="ticket-badge">REQUEST ID: {generatedTicketId}</div>
					<p>
						Thank you, <strong>{clientName || 'Guest'}</strong>! We have received your request for <strong>{selectedService.title}</strong>.
					</p>
					<div class="success-alert">
						<span>⚡ A senior travel concierge has been alerted and will send you the confirmed quote and driver/voucher details on WhatsApp at <strong>{clientPhone || '+91 98840 11223'}</strong> within 15 minutes.</span>
					</div>
					<div class="success-actions">
						<a
							href="https://wa.me/919884011223?text=Hi%2C%20I%20just%20submitted%20service%20request%20{generatedTicketId}%20for%20{encodeURIComponent(selectedService.title)}"
							target="_blank"
							rel="noreferrer"
							class="wa-connect-btn"
						>
							<span>Open WhatsApp Chat Directly</span>
						</a>
						<button type="button" class="btn-close-success" onclick={() => (showServiceModal = false)}>Close</button>
					</div>
				</div>
			{/if}
		</div>
	</div>
{/if}

<style>
	:global(body) {
		margin: 0;
		background: #090d16;
		color: #f8fafc;
		font-family: var(--font-sans, system-ui, -apple-system, sans-serif);
	}

	.customer-portal-page {
		min-height: 100vh;
		background: #090d16;
		overflow-x: hidden;
	}

	/* Hero Section */
	.portal-hero {
		position: relative;
		padding: 4.5rem 1.5rem 3.5rem;
		background: radial-gradient(circle at 50% 20%, rgba(2, 132, 199, 0.18) 0%, rgba(15, 23, 42, 0.95) 75%, #090d16 100%);
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
	}

	.hero-glow-orbit {
		position: absolute;
		top: -120px;
		left: 50%;
		transform: translateX(-50%);
		width: 700px;
		height: 350px;
		background: radial-gradient(ellipse at center, rgba(56, 189, 248, 0.25) 0%, rgba(79, 70, 229, 0.15) 50%, transparent 80%);
		filter: blur(80px);
		pointer-events: none;
	}

	.hero-container {
		max-width: 1080px;
		margin: 0 auto;
		text-align: center;
		position: relative;
		z-index: 10;
	}

	.hero-badge {
		display: inline-flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.35rem 0.9rem;
		border-radius: 999px;
		background: rgba(2, 132, 199, 0.15);
		border: 1px solid rgba(56, 189, 248, 0.35);
		color: #38bdf8;
		font-size: 0.76rem;
		font-weight: 700;
		letter-spacing: 0.06em;
		margin-bottom: 1.25rem;
	}

	.badge-dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		background: #38bdf8;
		box-shadow: 0 0 10px #38bdf8;
	}

	.hero-title {
		font-size: clamp(2.2rem, 5vw, 3.4rem);
		font-weight: 900;
		line-height: 1.15;
		letter-spacing: -0.03em;
		margin: 0 0 1.25rem;
		color: #ffffff;
	}

	.gradient-text {
		background: linear-gradient(135deg, #38bdf8 0%, #818cf8 50%, #c084fc 100%);
		-webkit-background-clip: text;
		-webkit-text-fill-color: transparent;
	}

	.hero-desc {
		font-size: 1.1rem;
		color: #94a3b8;
		max-width: 720px;
		margin: 0 auto 2.5rem;
		line-height: 1.6;
	}

	/* PNR Tracker Card */
	.pnr-tracker-card {
		background: rgba(17, 24, 39, 0.85);
		border: 1px solid rgba(255, 255, 255, 0.12);
		border-radius: 18px;
		padding: 1.5rem 1.75rem;
		box-shadow: 0 20px 45px rgba(0, 0, 0, 0.55);
		backdrop-filter: blur(14px);
		margin-bottom: 2.5rem;
		text-align: left;
	}

	.tracker-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		flex-wrap: wrap;
		gap: 0.75rem;
		margin-bottom: 1rem;
	}

	.tracker-title {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 0.95rem;
		font-weight: 700;
		color: #f1f5f9;
	}

	.demo-pnr-hints {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 0.75rem;
		color: #64748b;
	}

	.hint-btn {
		background: rgba(255, 255, 255, 0.06);
		border: 1px solid rgba(255, 255, 255, 0.12);
		color: #93c5fd;
		padding: 0.2rem 0.55rem;
		border-radius: 6px;
		font-size: 0.75rem;
		cursor: pointer;
		transition: all 0.15s;
	}

	.hint-btn:hover {
		background: rgba(2, 132, 199, 0.25);
		border-color: #38bdf8;
		color: #ffffff;
	}

	.tracker-input-row {
		display: flex;
		gap: 0.75rem;
	}

	.input-with-icon {
		position: relative;
		flex: 1;
	}

	.input-icon {
		position: absolute;
		left: 1rem;
		top: 50%;
		transform: translateY(-50%);
		font-size: 1.1rem;
		opacity: 0.7;
	}

	.pnr-input {
		width: 100%;
		box-sizing: border-box;
		padding: 0.85rem 1rem 0.85rem 2.75rem;
		background: rgba(15, 23, 42, 0.9);
		border: 1px solid rgba(255, 255, 255, 0.15);
		border-radius: 12px;
		color: #ffffff;
		font-size: 1rem;
		font-weight: 600;
		outline: none;
		transition: border-color 0.2s;
	}

	.pnr-input:focus {
		border-color: #38bdf8;
		box-shadow: 0 0 0 3px rgba(56, 189, 248, 0.25);
	}

	.lookup-btn {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0 1.6rem;
		background: linear-gradient(135deg, #0284c7 0%, #2563eb 100%);
		border: none;
		border-radius: 12px;
		color: #ffffff;
		font-size: 0.95rem;
		font-weight: 700;
		cursor: pointer;
		white-space: nowrap;
		transition: all 0.2s ease;
		box-shadow: 0 4px 14px rgba(2, 132, 199, 0.4);
	}

	.lookup-btn:hover:not(:disabled) {
		background: linear-gradient(135deg, #0369a1 0%, #1d4ed8 100%);
		transform: translateY(-1px);
	}

	.lookup-btn:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}

	.spinner {
		width: 14px;
		height: 14px;
		border: 2px solid rgba(255, 255, 255, 0.3);
		border-top-color: #ffffff;
		border-radius: 50%;
		animation: spin 0.7s linear infinite;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	/* PNR Result Box */
	.pnr-result-box {
		margin-top: 1.25rem;
		padding: 1.25rem;
		border-radius: 14px;
		background: rgba(15, 23, 42, 0.95);
		border: 1px solid rgba(56, 189, 248, 0.35);
		animation: slide-down 0.3s cubic-bezier(0.16, 1, 0.3, 1);
	}

	@keyframes slide-down {
		from { opacity: 0; transform: translateY(-8px); }
		to { opacity: 1; transform: translateY(0); }
	}

	.result-top {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		flex-wrap: wrap;
		gap: 0.75rem;
		margin-bottom: 1.25rem;
		padding-bottom: 1rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
	}

	.result-pnr-badge {
		font-size: 0.72rem;
		font-weight: 800;
		letter-spacing: 0.06em;
		color: #38bdf8;
		background: rgba(56, 189, 248, 0.15);
		padding: 0.2rem 0.55rem;
		border-radius: 6px;
		display: inline-block;
		margin-bottom: 0.35rem;
	}

	.result-trip-title {
		font-size: 1.18rem;
		font-weight: 800;
		color: #ffffff;
		margin: 0 0 0.25rem;
	}

	.result-guest {
		font-size: 0.85rem;
		color: #94a3b8;
	}

	.status-pill {
		display: flex;
		align-items: center;
		gap: 0.45rem;
		padding: 0.35rem 0.75rem;
		border-radius: 999px;
		border: 1px solid;
		font-size: 0.78rem;
		font-weight: 700;
		background: rgba(0, 0, 0, 0.3);
	}

	.status-dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
	}

	.result-details-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
		gap: 0.85rem;
		margin-bottom: 1.25rem;
	}

	.detail-cell {
		background: rgba(255, 255, 255, 0.03);
		padding: 0.75rem;
		border-radius: 10px;
		border: 1px solid rgba(255, 255, 255, 0.06);
	}

	.detail-label {
		font-size: 0.72rem;
		color: #64748b;
		margin-bottom: 0.35rem;
		font-weight: 600;
	}

	.detail-value {
		font-size: 0.88rem;
		color: #e2e8f0;
		font-weight: 600;
	}

	.detail-value.highlight {
		color: #38bdf8;
	}

	.result-actions {
		display: flex;
		gap: 0.75rem;
		flex-wrap: wrap;
	}

	.action-btn-primary {
		display: flex;
		align-items: center;
		gap: 0.45rem;
		padding: 0.6rem 1.1rem;
		background: #0284c7;
		color: #ffffff;
		border-radius: 10px;
		text-decoration: none;
		font-size: 0.86rem;
		font-weight: 700;
		transition: background 0.15s;
	}

	.action-btn-primary:hover {
		background: #0369a1;
	}

	.action-btn-wa {
		display: flex;
		align-items: center;
		gap: 0.45rem;
		padding: 0.6rem 1.1rem;
		background: rgba(37, 211, 102, 0.15);
		color: #25d366;
		border: 1px solid rgba(37, 211, 102, 0.35);
		border-radius: 10px;
		text-decoration: none;
		font-size: 0.86rem;
		font-weight: 700;
	}

	/* Hero Action Pills */
	.hero-action-pills {
		display: flex;
		justify-content: center;
		gap: 1rem;
		flex-wrap: wrap;
	}

	.hero-cta-btn {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.75rem 1.4rem;
		border-radius: 12px;
		font-size: 0.94rem;
		font-weight: 700;
		text-decoration: none;
		transition: all 0.2s ease;
	}

	.hero-cta-btn.primary {
		background: linear-gradient(135deg, #0284c7 0%, #4f46e5 100%);
		color: #ffffff;
		box-shadow: 0 4px 16px rgba(2, 132, 199, 0.35);
	}

	.hero-cta-btn.primary:hover {
		transform: translateY(-2px);
		box-shadow: 0 6px 20px rgba(2, 132, 199, 0.5);
	}

	.hero-cta-btn.secondary {
		background: rgba(255, 255, 255, 0.08);
		border: 1px solid rgba(255, 255, 255, 0.16);
		color: #ffffff;
	}

	.hero-cta-btn.secondary:hover {
		background: rgba(255, 255, 255, 0.14);
		transform: translateY(-1px);
	}

	.hero-cta-btn.partner {
		background: rgba(16, 185, 129, 0.15);
		border: 1px solid rgba(16, 185, 129, 0.35);
		color: #34d399;
	}

	.hero-cta-btn.partner:hover {
		background: rgba(16, 185, 129, 0.25);
		transform: translateY(-1px);
	}

	/* Services Section */
	.services-section {
		padding: 4.5rem 1.5rem;
		background: #090d16;
	}

	.section-container {
		max-width: 1280px;
		margin: 0 auto;
	}

	.section-header {
		text-align: center;
		max-width: 700px;
		margin: 0 auto 3rem;
	}

	.section-badge {
		font-size: 0.75rem;
		font-weight: 800;
		letter-spacing: 0.1em;
		color: #38bdf8;
		margin-bottom: 0.5rem;
	}

	.section-title {
		font-size: clamp(1.8rem, 3.5vw, 2.4rem);
		font-weight: 800;
		color: #ffffff;
		margin: 0 0 0.85rem;
		letter-spacing: -0.02em;
	}

	.section-subtitle {
		font-size: 1rem;
		color: #94a3b8;
		line-height: 1.55;
		margin: 0;
	}

	.services-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(290px, 1fr));
		gap: 1.5rem;
	}

	.service-card {
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 16px;
		padding: 1.5rem;
		display: flex;
		flex-direction: column;
		transition: all 0.25s ease;
	}

	.service-card:hover {
		border-color: rgba(56, 189, 248, 0.4);
		transform: translateY(-4px);
		box-shadow: 0 16px 32px rgba(0, 0, 0, 0.4);
	}

	.card-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 1rem;
	}

	.service-icon-box {
		width: 48px;
		height: 48px;
		border-radius: 12px;
		background: rgba(255, 255, 255, 0.05);
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 1.5rem;
	}

	.service-badge {
		font-size: 0.72rem;
		font-weight: 700;
		padding: 0.25rem 0.6rem;
		border-radius: 999px;
	}

	.service-category {
		font-size: 0.68rem;
		font-weight: 800;
		letter-spacing: 0.06em;
		color: #64748b;
		margin-bottom: 0.35rem;
	}

	.service-title {
		font-size: 1.15rem;
		font-weight: 700;
		color: #ffffff;
		margin: 0 0 0.65rem;
		line-height: 1.3;
	}

	.service-desc {
		font-size: 0.86rem;
		color: #94a3b8;
		line-height: 1.5;
		margin: 0 0 1.25rem;
	}

	.feature-list {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		margin-bottom: 1.5rem;
		flex: 1;
	}

	.feature-item {
		display: flex;
		align-items: flex-start;
		gap: 0.5rem;
		font-size: 0.82rem;
		color: #cbd5e1;
		line-height: 1.35;
	}

	.check-icon {
		color: #10b981;
		font-weight: 800;
	}

	.card-footer {
		padding-top: 1rem;
		border-top: 1px solid rgba(255, 255, 255, 0.06);
		display: flex;
		align-items: flex-end;
		justify-content: space-between;
		gap: 0.75rem;
	}

	.pricing-info {
		display: flex;
		flex-direction: column;
	}

	.price-label {
		font-size: 0.68rem;
		color: #64748b;
		text-transform: uppercase;
		letter-spacing: 0.04em;
	}

	.price-val {
		font-size: 0.98rem;
		font-weight: 800;
		color: #38bdf8;
	}

	.lead-time {
		font-size: 0.7rem;
		color: #94a3b8;
		margin-top: 0.15rem;
	}

	.request-service-btn {
		display: flex;
		align-items: center;
		gap: 0.35rem;
		padding: 0.55rem 0.9rem;
		border-radius: 10px;
		background: rgba(2, 132, 199, 0.18);
		border: 1px solid rgba(56, 189, 248, 0.35);
		color: #38bdf8;
		font-size: 0.84rem;
		font-weight: 700;
		cursor: pointer;
		transition: all 0.15s;
		white-space: nowrap;
	}

	.request-service-btn:hover {
		background: #0284c7;
		color: #ffffff;
		border-color: #0284c7;
	}

	/* Trust Section */
	.trust-section {
		padding: 3rem 1.5rem;
		background: #0b1120;
		border-top: 1px solid rgba(255, 255, 255, 0.06);
		border-bottom: 1px solid rgba(255, 255, 255, 0.06);
	}

	.trust-container {
		max-width: 1280px;
		margin: 0 auto;
	}

	.trust-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
		gap: 2rem;
	}

	.trust-item {
		display: flex;
		flex-direction: column;
		gap: 0.45rem;
	}

	.trust-icon {
		font-size: 2rem;
		margin-bottom: 0.25rem;
	}

	.trust-title {
		font-size: 1.05rem;
		font-weight: 700;
		color: #ffffff;
	}

	.trust-desc {
		font-size: 0.85rem;
		color: #94a3b8;
		line-height: 1.5;
	}

	/* FAQ Section */
	.faq-section {
		padding: 4.5rem 1.5rem;
		background: #090d16;
	}

	.faq-container {
		max-width: 800px;
		margin: 0 auto;
	}

	.faq-list {
		display: flex;
		flex-direction: column;
		gap: 0.85rem;
	}

	.faq-card {
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 12px;
		overflow: hidden;
		transition: border-color 0.2s;
	}

	.faq-card.open {
		border-color: rgba(56, 189, 248, 0.4);
	}

	.faq-question {
		width: 100%;
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 1.1rem 1.25rem;
		background: none;
		border: none;
		color: #f1f5f9;
		font-size: 0.96rem;
		font-weight: 700;
		text-align: left;
		cursor: pointer;
	}

	.faq-icon {
		font-size: 1.3rem;
		color: #38bdf8;
		margin-left: 0.75rem;
	}

	.faq-answer {
		padding: 0 1.25rem 1.25rem;
		color: #94a3b8;
		font-size: 0.88rem;
		line-height: 1.6;
		border-top: 1px solid rgba(255, 255, 255, 0.04);
		padding-top: 0.85rem;
	}

	/* CTA Banner */
	.portal-cta-banner {
		padding: 4rem 1.5rem;
		background: linear-gradient(135deg, #0369a1 0%, #4338ca 100%);
		text-align: center;
	}

	.cta-inner {
		max-width: 800px;
		margin: 0 auto;
	}

	.cta-inner h2 {
		font-size: clamp(2rem, 4vw, 2.6rem);
		font-weight: 900;
		color: #ffffff;
		margin: 0 0 0.85rem;
		letter-spacing: -0.02em;
	}

	.cta-inner p {
		font-size: 1.1rem;
		color: #e0e7ff;
		margin: 0 0 2rem;
		line-height: 1.6;
	}

	.cta-buttons {
		display: flex;
		justify-content: center;
		gap: 1rem;
		flex-wrap: wrap;
	}

	.banner-btn {
		padding: 0.85rem 1.6rem;
		border-radius: 12px;
		font-size: 0.95rem;
		font-weight: 700;
		text-decoration: none;
		transition: all 0.2s ease;
	}

	.banner-btn.primary {
		background: #ffffff;
		color: #1e1b4b;
		box-shadow: 0 4px 16px rgba(0, 0, 0, 0.25);
	}

	.banner-btn.primary:hover {
		background: #f1f5f9;
		transform: translateY(-2px);
	}

	.banner-btn.secondary {
		background: rgba(255, 255, 255, 0.18);
		border: 1px solid rgba(255, 255, 255, 0.3);
		color: #ffffff;
	}

	.banner-btn.secondary:hover {
		background: rgba(255, 255, 255, 0.28);
	}

	.banner-btn.partner {
		background: rgba(16, 185, 129, 0.3);
		border: 1px solid rgba(52, 211, 153, 0.4);
		color: #ffffff;
	}

	/* Modal Styles */
	.modal-backdrop {
		position: fixed;
		inset: 0;
		z-index: 2000;
		background: rgba(0, 0, 0, 0.75);
		backdrop-filter: blur(8px);
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 1.5rem;
	}

	.service-modal-box {
		position: relative;
		width: 100%;
		max-width: 620px;
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.15);
		border-radius: 20px;
		padding: 2rem;
		box-shadow: 0 25px 60px rgba(0, 0, 0, 0.7);
		max-height: 90vh;
		overflow-y: auto;
	}

	.modal-close-btn {
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
		font-size: 1rem;
	}

	.modal-header {
		margin-bottom: 1.5rem;
	}

	.modal-service-badge {
		font-size: 0.75rem;
		font-weight: 800;
		color: #38bdf8;
		margin-bottom: 0.35rem;
	}

	.modal-title {
		font-size: 1.35rem;
		font-weight: 800;
		color: #ffffff;
		margin: 0 0 0.35rem;
	}

	.modal-subtitle {
		font-size: 0.85rem;
		color: #94a3b8;
		margin: 0;
	}

	.modal-form {
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}

	.form-row {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 1rem;
	}

	.form-group {
		display: flex;
		flex-direction: column;
		gap: 0.35rem;
	}

	.form-group label {
		font-size: 0.78rem;
		font-weight: 700;
		color: #cbd5e1;
	}

	.form-input, .form-textarea {
		width: 100%;
		box-sizing: border-box;
		padding: 0.7rem 0.9rem;
		background: rgba(15, 23, 42, 0.8);
		border: 1px solid rgba(255, 255, 255, 0.12);
		border-radius: 10px;
		color: #ffffff;
		font-size: 0.9rem;
		outline: none;
	}

	.form-input:focus, .form-textarea:focus {
		border-color: #38bdf8;
	}

	.price-estimate-box {
		padding: 0.85rem 1rem;
		border-radius: 10px;
		background: rgba(2, 132, 199, 0.12);
		border: 1px solid rgba(56, 189, 248, 0.25);
	}

	.est-label {
		font-size: 0.72rem;
		color: #94a3b8;
		text-transform: uppercase;
		font-weight: 700;
	}

	.est-amount {
		font-size: 1.25rem;
		font-weight: 800;
		color: #38bdf8;
		margin: 0.15rem 0;
	}

	.est-note {
		font-size: 0.74rem;
		color: #cbd5e1;
	}

	.modal-actions {
		display: flex;
		justify-content: flex-end;
		gap: 0.75rem;
		margin-top: 0.5rem;
	}

	.btn-cancel {
		padding: 0.7rem 1.25rem;
		border-radius: 10px;
		background: rgba(255, 255, 255, 0.08);
		border: none;
		color: #ffffff;
		font-size: 0.88rem;
		font-weight: 600;
		cursor: pointer;
	}

	.btn-confirm-submit {
		display: flex;
		align-items: center;
		gap: 0.45rem;
		padding: 0.7rem 1.4rem;
		border-radius: 10px;
		background: linear-gradient(135deg, #0284c7 0%, #2563eb 100%);
		border: none;
		color: #ffffff;
		font-size: 0.9rem;
		font-weight: 700;
		cursor: pointer;
	}

	/* Success Box */
	.success-box {
		text-align: center;
		padding: 1.5rem 0;
	}

	.success-icon {
		font-size: 3rem;
		margin-bottom: 0.5rem;
	}

	.ticket-badge {
		display: inline-block;
		padding: 0.35rem 0.85rem;
		border-radius: 999px;
		background: rgba(16, 185, 129, 0.2);
		color: #34d399;
		border: 1px solid rgba(16, 185, 129, 0.4);
		font-size: 0.82rem;
		font-weight: 800;
		margin: 0.5rem 0 1rem;
	}

	.success-alert {
		background: rgba(2, 132, 199, 0.15);
		border: 1px solid rgba(56, 189, 248, 0.3);
		border-radius: 12px;
		padding: 1rem;
		color: #e0f2fe;
		font-size: 0.88rem;
		line-height: 1.5;
		margin: 1.25rem 0;
		text-align: left;
	}

	.success-actions {
		display: flex;
		flex-direction: column;
		gap: 0.65rem;
	}

	.wa-connect-btn {
		padding: 0.8rem;
		border-radius: 10px;
		background: #25d366;
		color: #ffffff;
		font-weight: 700;
		font-size: 0.92rem;
		text-decoration: none;
	}

	.btn-close-success {
		padding: 0.7rem;
		border-radius: 10px;
		background: rgba(255, 255, 255, 0.08);
		border: none;
		color: #ffffff;
		cursor: pointer;
		font-weight: 600;
	}

	@media (max-width: 768px) {
		.tracker-input-row {
			flex-direction: column;
		}
		.form-row {
			grid-template-columns: 1fr;
		}
		.hero-action-pills {
			flex-direction: column;
		}
	}
</style>
