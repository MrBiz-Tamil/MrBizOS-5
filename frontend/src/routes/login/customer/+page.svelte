<script lang="ts">
	import PortalNavbar from '$lib/components/portal/PortalNavbar.svelte';

	// Login State
	let loginMode = $state<'pnr' | 'password'>('pnr');
	let pnrCode = $state('MB-TRV-8849');
	let mobileNumber = $state('+91 98840 11223');
	let email = $state('rahul.sharma@example.com');
	let password = $state('******');
	let otpSent = $state(false);
	let otpValue = $state('4481');
	let isLoading = $state(false);
	let isLoggedIn = $state(false);
	let errorMessage = $state('');

	// Active Traveler Trip State
	let travelerData = $state<any>({
		name: 'Rahul Sharma',
		phone: '+91 98840 11223',
		pnr: 'MB-TRV-8849',
		circuit: 'Kashmir Valley Paradise & Gulmarg Snow (4D/5N)',
		dates: '14 Nov 2026 – 19 Nov 2026',
		passengers: '2 Adults, 2 Children',
		status: 'CONFIRMED & ON-GROUND READY',
		chauffeur: {
			name: 'Ghulam Nabi',
			phone: '+91 94190 22334',
			vehicle: 'Toyota Innova Crysta AC (White)',
			plate: 'JK-01-AB-4491',
			placard: 'SHARMA FAMILY'
		},
		hotels: [
			{ city: 'Pahalgam', name: 'Hotel Grand Kashmir Riverfront', nights: '2 Nights (MAP Plan)', voucher: 'DMC-PHL-8819' },
			{ city: 'Gulmarg', name: 'Pine Palace Resort & Spa', nights: '1 Night (Heated Suite)', voucher: 'DMC-GLM-4421' },
			{ city: 'Srinagar', name: 'Royal Heritage Shikara Houseboat', nights: '2 Nights (Dal Lake)', voucher: 'DMC-SXR-1102' }
		],
		passes: [
			{ name: 'Gulmarg Gondola Phase 1 (Kongdoori)', status: 'CONFIRMED (Time: 10:30 AM)' },
			{ name: 'Gulmarg Gondola Phase 2 (Apharwat Peak)', status: 'CONFIRMED (Time: 12:00 PM)' },
			{ name: '1-Hour Sunset Dal Lake Shikara Ride', status: 'CONFIRMED (Ghat 11)' }
		],
		flight: {
			airline: 'IndiGo 6E-2144',
			route: 'MAA (Chennai) ➔ SXR (Srinagar via DEL)',
			departure: '06:15 AM',
			arrival: '12:30 PM',
			status: 'ON TIME • GATE 4'
		}
	});

	function handleSendOtp() {
		if (!pnrCode.trim() || !mobileNumber.trim()) {
			errorMessage = 'Please enter both Booking PNR and Mobile Number';
			return;
		}
		errorMessage = '';
		isLoading = true;
		setTimeout(() => {
			otpSent = true;
			isLoading = false;
		}, 400);
	}

	function handleVerifyAndLogin() {
		isLoading = true;
		errorMessage = '';
		setTimeout(() => {
			isLoading = false;
			isLoggedIn = true;
		}, 400);
	}

	function handlePasswordLogin(e: Event) {
		e.preventDefault();
		isLoading = true;
		setTimeout(() => {
			isLoading = false;
			isLoggedIn = true;
		}, 400);
	}

	function loadPreset(name: string, phone: string, pnr: string, circuit: string) {
		travelerData.name = name;
		travelerData.phone = phone;
		travelerData.pnr = pnr;
		travelerData.circuit = circuit;
		pnrCode = pnr;
		mobileNumber = phone;
		email = name.toLowerCase().replace(' ', '.') + '@example.com';
		isLoggedIn = true;
	}

	function logout() {
		isLoggedIn = false;
		otpSent = false;
	}
</script>

<svelte:head>
	<title>Customer & Traveler Login | MrBiz OS</title>
	<meta name="description" content="Customer portal login to access your live trip itinerary, chauffeur tracker, digital vouchers, and emergency SOS care." />
</svelte:head>

<div class="traveler-login-page">
	<PortalNavbar />

	<div class="page-container">
		{#if !isLoggedIn}
			<!-- LOGIN CARD -->
			<div class="login-wrapper">
				<div class="login-card">
					<!-- Top Branding -->
					<div class="login-header">
						<div class="header-icon-badge">🧳</div>
						<div class="portal-category">CUSTOMER & TRAVELER PORTAL</div>
						<h1 class="login-title">Access Your Tour & Vouchers</h1>
						<p class="login-subtitle">Log in to view confirmed itineraries, track your private chauffeur, and download e-tickets.</p>
					</div>

					<!-- Login Mode Tabs -->
					<div class="mode-tabs">
						<button
							type="button"
							class="mode-tab {loginMode === 'pnr' ? 'active' : ''}"
							onclick={() => { loginMode = 'pnr'; errorMessage = ''; }}
						>
							<span>🎫 Login via Booking PNR & Phone</span>
						</button>
						<button
							type="button"
							class="mode-tab {loginMode === 'password' ? 'active' : ''}"
							onclick={() => { loginMode = 'password'; errorMessage = ''; }}
						>
							<span>✉️ Email & Password</span>
						</button>
					</div>

					{#if errorMessage}
						<div class="error-banner">
							<span>⚠️ {errorMessage}</span>
						</div>
					{/if}

					<!-- PNR + OTP FORM -->
					{#if loginMode === 'pnr'}
						<div class="login-form">
							<div class="form-group">
								<label for="pnr-ref">Tour Booking ID / PNR Number *</label>
								<div class="input-wrap">
									<span class="icon">🎫</span>
									<input id="pnr-ref" type="text" bind:value={pnrCode} placeholder="e.g. MB-TRV-8849" class="input-field" />
								</div>
							</div>

							<div class="form-group">
								<label for="pnr-phone">Registered Mobile / WhatsApp Number *</label>
								<div class="input-wrap">
									<span class="icon">📱</span>
									<input id="pnr-phone" type="tel" bind:value={mobileNumber} placeholder="+91 98840 11223" class="input-field" />
								</div>
							</div>

							{#if otpSent}
								<div class="otp-box">
									<label for="otp-val">Enter 4-Digit WhatsApp OTP Code (Demo: 4481)</label>
									<input id="otp-val" type="text" maxlength="6" bind:value={otpValue} class="otp-input" />
									<div class="otp-hint">OTP sent to {mobileNumber}. Enter code to access your itinerary.</div>
								</div>

								<button type="button" class="btn-submit-login" onclick={handleVerifyAndLogin} disabled={isLoading}>
									{#if isLoading}
										<span class="spinner"></span>
										<span>Authenticating Trip...</span>
									{:else}
										<span>Verify & Enter Dashboard</span>
										<span>➔</span>
									{/if}
								</button>
							{:else}
								<button type="button" class="btn-submit-login" onclick={handleSendOtp} disabled={isLoading}>
									{#if isLoading}
										<span class="spinner"></span>
										<span>Sending OTP...</span>
									{:else}
										<span>Send OTP & Continue</span>
										<span>➔</span>
									{/if}
								</button>
							{/if}
						</div>
					{:else}
						<!-- EMAIL & PASSWORD FORM -->
						<form onsubmit={handlePasswordLogin} class="login-form">
							<div class="form-group">
								<label for="cust-email">Traveler Email Address *</label>
								<div class="input-wrap">
									<span class="icon">✉️</span>
									<input id="cust-email" type="email" required bind:value={email} class="input-field" />
								</div>
							</div>

							<div class="form-group">
								<label for="cust-pass">Password *</label>
								<div class="input-wrap">
									<span class="icon">🔒</span>
									<input id="cust-pass" type="password" required bind:value={password} class="input-field" />
								</div>
							</div>

							<button type="submit" class="btn-submit-login" disabled={isLoading}>
								{#if isLoading}
									<span class="spinner"></span>
									<span>Signing In...</span>
								{:else}
									<span>Sign In to Dashboard</span>
									<span>➔</span>
								{/if}
							</button>
						</form>
					{/if}

					<!-- Quick 1-Click Demo Accounts -->
					<div class="demo-logins-box">
						<div class="demo-title">⚡ 1-CLICK INSTANT DEMO GUEST LOGINS:</div>
						<div class="demo-buttons">
							<button
								type="button"
								class="demo-btn"
								onclick={() => loadPreset('Rahul Sharma & Family', '+91 98840 11223', 'MB-TRV-8849', 'Kashmir Valley Paradise & Gulmarg Snow')}
							>
								<span>🏔️ Rahul Sharma (Kashmir 4D/5N)</span>
							</button>
							<button
								type="button"
								class="demo-btn"
								onclick={() => loadPreset('Ananya & Vikram Verma', '+91 98470 12345', 'KL-KER-3310', 'Kerala Backwaters & Munnar Honeymoon')}
							>
								<span>🌴 Ananya Verma (Kerala Honeymoon)</span>
							</button>
						</div>
					</div>

					<div class="login-footer-links">
						<span>Need to book a new tour?</span>
						<a href="/travel/booking" class="link-bold">Book a Tour Package</a>
						<span class="divider">•</span>
						<a href="/login/partner" class="link-muted">Travel Partner Login</a>
					</div>
				</div>
			</div>
		{:else}
			<!-- TRAVELER LIVE DASHBOARD VIEW -->
			<div class="dashboard-wrap">
				<!-- Top Bar -->
				<div class="dash-top-bar">
					<div>
						<div class="dash-welcome">WELCOME BACK, TRAVELER</div>
						<h2 class="dash-name">{travelerData.name}</h2>
						<div class="dash-pnr-tag">BOOKING PNR: {travelerData.pnr}</div>
					</div>
					<div class="top-actions">
						<button type="button" class="btn-print-itinerary" onclick={() => window.print()}>
							<span>🖨️ Print Full Voucher Kit</span>
						</button>
						<button type="button" class="btn-logout" onclick={logout}>
							<span>Sign Out</span>
						</button>
					</div>
				</div>

				<!-- Hero Status Banner -->
				<div class="dash-status-card">
					<div class="status-left">
						<div class="trip-badge">LIVE ACTIVE TRIP</div>
						<h3 class="trip-title">{travelerData.circuit}</h3>
						<div class="trip-meta">
							<span>📅 {travelerData.dates}</span>
							<span>👥 {travelerData.passengers}</span>
							<span>📍 Srinagar - Gulmarg - Pahalgam</span>
						</div>
					</div>
					<div class="status-right">
						<div class="status-box">
							<span class="dot"></span>
							<span>{travelerData.status}</span>
						</div>
					</div>
				</div>

				<!-- 3 Column Action Grid -->
				<div class="dash-grid-3">
					<!-- Card 1: Chauffeur & Vehicle -->
					<div class="dash-card">
						<div class="card-head">
							<span class="icon">🚗</span>
							<h4>Your Assigned Chauffeur</h4>
						</div>
						<div class="chauffeur-profile">
							<div class="driver-avatar">GN</div>
							<div>
								<div class="driver-name">{travelerData.chauffeur.name}</div>
								<div class="driver-role">Private VIP Chauffeur</div>
							</div>
						</div>
						<div class="info-list">
							<div class="info-item">
								<span>Vehicle:</span>
								<strong>{travelerData.chauffeur.vehicle}</strong>
							</div>
							<div class="info-item">
								<span>Number Plate:</span>
								<strong class="plate">{travelerData.chauffeur.plate}</strong>
							</div>
							<div class="info-item">
								<span>Airport Placard:</span>
								<strong>"{travelerData.chauffeur.placard}"</strong>
							</div>
						</div>
						<div class="card-btn-row">
							<a
								href="https://wa.me/{travelerData.chauffeur.phone.replace(/\D/g, '')}?text=Hi%20{encodeURIComponent(travelerData.chauffeur.name)}%2C%20I%20am%20guest%20{encodeURIComponent(travelerData.name)}%20arriving%20for%20PNR%20{travelerData.pnr}"
								target="_blank"
								rel="noreferrer"
								class="btn-driver-wa"
							>
								<span>WhatsApp Chauffeur Directly</span>
							</a>
						</div>
					</div>

					<!-- Card 2: Inbound Flight Radar -->
					<div class="dash-card">
						<div class="card-head">
							<span class="icon">✈️</span>
							<h4>Flight & Arrival Tracking</h4>
						</div>
						<div class="flight-route-box">
							<div class="airline-text">{travelerData.flight.airline}</div>
							<div class="route-text">{travelerData.flight.route}</div>
						</div>
						<div class="info-list">
							<div class="info-item">
								<span>Departure:</span>
								<strong>{travelerData.flight.departure}</strong>
							</div>
							<div class="info-item">
								<span>Arrival Time:</span>
								<strong>{travelerData.flight.arrival}</strong>
							</div>
							<div class="info-item">
								<span>Radar Status:</span>
								<strong style="color: #34d399;">{travelerData.flight.status}</strong>
							</div>
						</div>
						<div class="alert-box">
							<span>✓ Chauffeur will be standing at Gate Exit Gate 1 with your name placard.</span>
						</div>
					</div>

					<!-- Card 3: 24/7 SOS Travel Care -->
					<div class="dash-card">
						<div class="card-head">
							<span class="icon">🛡️</span>
							<h4>24/7 Concierge & SOS Desk</h4>
						</div>
						<p class="sos-desc">Your dedicated on-trip coordinator is ready for route changes, doctor-on-call, or hotel adjustments.</p>
						<div class="info-list">
							<div class="info-item">
								<span>Trip Coordinator:</span>
								<strong>Bashir Ahmed</strong>
							</div>
							<div class="info-item">
								<span>Emergency Hotline:</span>
								<strong style="color: #ef4444;">+91 98840 11223</strong>
							</div>
						</div>
						<div class="card-btn-row">
							<a
								href="https://wa.me/919884011223?text=SOS%20Emergency%20Care%20Ticket%20for%20PNR%20{travelerData.pnr}"
								target="_blank"
								rel="noreferrer"
								class="btn-sos-wa"
							>
								<span>Contact 24/7 SOS Desk</span>
							</a>
						</div>
					</div>
				</div>

				<!-- Confirmed Hotel Vouchers & Fast Passes -->
				<div class="dash-split-2">
					<!-- Hotel Vouchers -->
					<div class="dash-card">
						<div class="card-head">
							<span class="icon">🏨</span>
							<h4>Confirmed Hotel & Houseboat Vouchers</h4>
						</div>
						<div class="hotels-list">
							{#each travelerData.hotels as h}
								<div class="hotel-item">
									<div class="h-city">{h.city}</div>
									<div class="h-details">
										<div class="h-name">{h.name}</div>
										<div class="h-plan">{h.nights}</div>
									</div>
									<div class="h-code">{h.voucher}</div>
								</div>
							{/each}
						</div>
					</div>

					<!-- Fast Passes -->
					<div class="dash-card">
						<div class="card-head">
							<span class="icon">🚠</span>
							<h4>Sightseeing & Adventure Passes</h4>
						</div>
						<div class="passes-list">
							{#each travelerData.passes as p}
								<div class="pass-item">
									<div class="pass-name">{p.name}</div>
									<div class="pass-status">{p.status}</div>
								</div>
							{/each}
						</div>
					</div>
				</div>
			</div>
		{/if}
	</div>
</div>

<style>
	.traveler-login-page {
		min-height: 100vh;
		background: #090d16;
		color: #f8fafc;
		font-family: var(--font-sans, system-ui, -apple-system, sans-serif);
	}

	.page-container {
		max-width: 1200px;
		margin: 0 auto;
		padding: 3rem 1.5rem;
	}

	.login-wrapper {
		max-width: 520px;
		margin: 2rem auto;
	}

	.login-card {
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.12);
		border-radius: 20px;
		padding: 2.25rem;
		box-shadow: 0 20px 50px rgba(0, 0, 0, 0.5);
	}

	.login-header {
		text-align: center;
		margin-bottom: 1.5rem;
	}

	.header-icon-badge {
		font-size: 2.5rem;
		width: 60px;
		height: 60px;
		border-radius: 16px;
		background: rgba(2, 132, 199, 0.15);
		border: 1px solid rgba(56, 189, 248, 0.35);
		display: flex;
		align-items: center;
		justify-content: center;
		margin: 0 auto 0.75rem;
	}

	.portal-category {
		font-size: 0.72rem;
		font-weight: 800;
		color: #38bdf8;
		letter-spacing: 0.08em;
		margin-bottom: 0.25rem;
	}

	.login-title {
		font-size: 1.6rem;
		font-weight: 800;
		color: #ffffff;
		margin: 0 0 0.45rem;
	}

	.login-subtitle {
		font-size: 0.86rem;
		color: #94a3b8;
		margin: 0;
		line-height: 1.5;
	}

	/* Mode Tabs */
	.mode-tabs {
		display: flex;
		background: rgba(255, 255, 255, 0.04);
		border-radius: 10px;
		padding: 0.25rem;
		margin-bottom: 1.5rem;
	}

	.mode-tab {
		flex: 1;
		padding: 0.6rem;
		border: none;
		background: none;
		color: #94a3b8;
		font-size: 0.78rem;
		font-weight: 700;
		border-radius: 8px;
		cursor: pointer;
		transition: all 0.15s;
	}

	.mode-tab.active {
		background: #0284c7;
		color: #ffffff;
	}

	.error-banner {
		background: rgba(239, 68, 68, 0.15);
		border: 1px solid rgba(239, 68, 68, 0.35);
		color: #fca5a5;
		padding: 0.65rem 0.85rem;
		border-radius: 8px;
		font-size: 0.82rem;
		margin-bottom: 1rem;
	}

	/* Form */
	.login-form {
		display: flex;
		flex-direction: column;
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

	.input-wrap {
		position: relative;
	}

	.input-wrap .icon {
		position: absolute;
		left: 0.85rem;
		top: 50%;
		transform: translateY(-50%);
		font-size: 1rem;
		opacity: 0.7;
	}

	.input-field {
		width: 100%;
		box-sizing: border-box;
		padding: 0.75rem 0.9rem 0.75rem 2.5rem;
		background: #090d16;
		border: 1px solid rgba(255, 255, 255, 0.14);
		border-radius: 10px;
		color: #ffffff;
		font-size: 0.92rem;
		outline: none;
	}

	.input-field:focus {
		border-color: #38bdf8;
	}

	.otp-box {
		background: rgba(2, 132, 199, 0.1);
		border: 1px solid rgba(56, 189, 248, 0.3);
		border-radius: 12px;
		padding: 1rem;
		text-align: center;
	}

	.otp-box label {
		font-size: 0.8rem;
		font-weight: 700;
		color: #e0f2fe;
		display: block;
		margin-bottom: 0.5rem;
	}

	.otp-input {
		width: 160px;
		text-align: center;
		letter-spacing: 0.3em;
		font-size: 1.4rem;
		font-weight: 800;
		padding: 0.5rem;
		background: #090d16;
		border: 2px solid #38bdf8;
		border-radius: 8px;
		color: #ffffff;
		outline: none;
		margin-bottom: 0.4rem;
	}

	.otp-hint {
		font-size: 0.72rem;
		color: #94a3b8;
	}

	.btn-submit-login {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.5rem;
		padding: 0.85rem;
		border-radius: 12px;
		background: linear-gradient(135deg, #0284c7 0%, #2563eb 100%);
		border: none;
		color: #ffffff;
		font-weight: 800;
		font-size: 0.95rem;
		cursor: pointer;
		margin-top: 0.5rem;
		box-shadow: 0 4px 14px rgba(2, 132, 199, 0.4);
	}

	.spinner {
		width: 16px;
		height: 16px;
		border: 2px solid rgba(255, 255, 255, 0.3);
		border-top-color: #ffffff;
		border-radius: 50%;
		animation: spin 0.7s linear infinite;
	}

	@keyframes spin { to { transform: rotate(360deg); } }

	/* Demo Logins Box */
	.demo-logins-box {
		margin-top: 1.5rem;
		padding-top: 1.25rem;
		border-top: 1px solid rgba(255, 255, 255, 0.08);
	}

	.demo-title {
		font-size: 0.7rem;
		font-weight: 800;
		color: #64748b;
		margin-bottom: 0.65rem;
	}

	.demo-buttons {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.demo-btn {
		text-align: left;
		padding: 0.6rem 0.85rem;
		border-radius: 8px;
		background: rgba(255, 255, 255, 0.04);
		border: 1px solid rgba(255, 255, 255, 0.08);
		color: #cbd5e1;
		font-size: 0.8rem;
		font-weight: 600;
		cursor: pointer;
		transition: all 0.15s;
	}

	.demo-btn:hover {
		background: rgba(2, 132, 199, 0.2);
		border-color: #38bdf8;
		color: #ffffff;
	}

	.login-footer-links {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.5rem;
		font-size: 0.78rem;
		color: #94a3b8;
		margin-top: 1.5rem;
	}

	.link-bold { color: #38bdf8; font-weight: 700; text-decoration: none; }
	.link-muted { color: #94a3b8; text-decoration: none; }
	.divider { color: #475569; }

	/* DASHBOARD VIEW */
	.dashboard-wrap {
		display: flex;
		flex-direction: column;
		gap: 1.5rem;
	}

	.dash-top-bar {
		display: flex;
		justify-content: space-between;
		align-items: center;
		flex-wrap: wrap;
		gap: 1rem;
		padding-bottom: 1.5rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
	}

	.dash-welcome {
		font-size: 0.72rem;
		font-weight: 800;
		color: #38bdf8;
		letter-spacing: 0.08em;
	}

	.dash-name {
		font-size: 1.8rem;
		font-weight: 900;
		color: #ffffff;
		margin: 0.2rem 0;
	}

	.dash-pnr-tag {
		font-size: 0.82rem;
		color: #94a3b8;
	}

	.top-actions {
		display: flex;
		gap: 0.75rem;
	}

	.btn-print-itinerary {
		padding: 0.65rem 1.25rem;
		border-radius: 10px;
		background: rgba(255, 255, 255, 0.08);
		border: 1px solid rgba(255, 255, 255, 0.15);
		color: #ffffff;
		font-weight: 700;
		font-size: 0.85rem;
		cursor: pointer;
	}

	.btn-logout {
		padding: 0.65rem 1.25rem;
		border-radius: 10px;
		background: rgba(239, 68, 68, 0.15);
		border: 1px solid rgba(239, 68, 68, 0.35);
		color: #fca5a5;
		font-weight: 700;
		font-size: 0.85rem;
		cursor: pointer;
	}

	/* Status Card */
	.dash-status-card {
		background: linear-gradient(135deg, rgba(2, 132, 199, 0.2) 0%, rgba(15, 23, 42, 0.95) 100%);
		border: 1px solid rgba(56, 189, 248, 0.35);
		border-radius: 18px;
		padding: 1.5rem 2rem;
		display: flex;
		justify-content: space-between;
		align-items: center;
		flex-wrap: wrap;
		gap: 1rem;
	}

	.trip-badge {
		font-size: 0.7rem;
		font-weight: 800;
		color: #34d399;
		letter-spacing: 0.08em;
		margin-bottom: 0.25rem;
	}

	.trip-title {
		font-size: 1.4rem;
		font-weight: 800;
		color: #ffffff;
		margin: 0 0 0.45rem;
	}

	.trip-meta {
		display: flex;
		gap: 1rem;
		font-size: 0.85rem;
		color: #cbd5e1;
		flex-wrap: wrap;
	}

	.status-box {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		background: rgba(16, 185, 129, 0.2);
		border: 1px solid #10b981;
		color: #34d399;
		padding: 0.5rem 1rem;
		border-radius: 999px;
		font-weight: 800;
		font-size: 0.82rem;
	}

	.dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		background: #34d399;
	}

	/* Dash Grid 3 */
	.dash-grid-3 {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 1.5rem;
	}

	.dash-card {
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 16px;
		padding: 1.5rem;
		display: flex;
		flex-direction: column;
	}

	.card-head {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		margin-bottom: 1.25rem;
	}

	.card-head .icon { font-size: 1.3rem; }
	.card-head h4 { font-size: 1.05rem; font-weight: 800; color: #ffffff; margin: 0; }

	.chauffeur-profile {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		margin-bottom: 1rem;
	}

	.driver-avatar {
		width: 44px;
		height: 44px;
		border-radius: 50%;
		background: #0284c7;
		display: flex;
		align-items: center;
		justify-content: center;
		font-weight: 800;
		color: #ffffff;
	}

	.driver-name { font-size: 0.95rem; font-weight: 800; color: #ffffff; }
	.driver-role { font-size: 0.74rem; color: #94a3b8; }

	.info-list {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		margin-bottom: 1.25rem;
		flex: 1;
	}

	.info-item {
		display: flex;
		justify-content: space-between;
		font-size: 0.82rem;
		color: #94a3b8;
	}

	.info-item strong { color: #f1f5f9; }
	.info-item strong.plate { color: #38bdf8; }

	.btn-driver-wa {
		display: block;
		text-align: center;
		padding: 0.7rem;
		border-radius: 10px;
		background: #25d366;
		color: #ffffff;
		font-weight: 700;
		font-size: 0.85rem;
		text-decoration: none;
	}

	.flight-route-box {
		background: rgba(255, 255, 255, 0.03);
		border-radius: 8px;
		padding: 0.75rem;
		margin-bottom: 1rem;
	}

	.airline-text { font-size: 0.9rem; font-weight: 800; color: #ffffff; }
	.route-text { font-size: 0.76rem; color: #94a3b8; }

	.alert-box {
		background: rgba(16, 185, 129, 0.1);
		border: 1px solid rgba(16, 185, 129, 0.25);
		border-radius: 8px;
		padding: 0.65rem;
		font-size: 0.78rem;
		color: #34d399;
	}

	.sos-desc {
		font-size: 0.82rem;
		color: #94a3b8;
		line-height: 1.4;
		margin: 0 0 1rem;
	}

	.btn-sos-wa {
		display: block;
		text-align: center;
		padding: 0.7rem;
		border-radius: 10px;
		background: rgba(239, 68, 68, 0.2);
		border: 1px solid #ef4444;
		color: #fca5a5;
		font-weight: 700;
		font-size: 0.85rem;
		text-decoration: none;
	}

	/* Split 2 */
	.dash-split-2 {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 1.5rem;
	}

	.hotels-list, .passes-list {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	.hotel-item {
		display: flex;
		align-items: center;
		justify-content: space-between;
		background: rgba(255, 255, 255, 0.02);
		border: 1px solid rgba(255, 255, 255, 0.06);
		border-radius: 10px;
		padding: 0.75rem 1rem;
	}

	.h-city {
		font-size: 0.72rem;
		font-weight: 800;
		color: #38bdf8;
		width: 70px;
	}

	.h-details { flex: 1; }
	.h-name { font-size: 0.86rem; font-weight: 700; color: #ffffff; }
	.h-plan { font-size: 0.74rem; color: #94a3b8; }

	.h-code {
		font-size: 0.75rem;
		background: rgba(255, 255, 255, 0.06);
		padding: 0.2rem 0.5rem;
		border-radius: 6px;
		color: #cbd5e1;
	}

	.pass-item {
		display: flex;
		align-items: center;
		justify-content: space-between;
		background: rgba(255, 255, 255, 0.02);
		border: 1px solid rgba(255, 255, 255, 0.06);
		border-radius: 10px;
		padding: 0.75rem 1rem;
	}

	.pass-name { font-size: 0.85rem; font-weight: 700; color: #ffffff; }
	.pass-status { font-size: 0.75rem; color: #34d399; font-weight: 700; }

	@media (max-width: 1024px) {
		.dash-grid-3, .dash-split-2 {
			grid-template-columns: 1fr;
		}
	}
</style>
