<script lang="ts">
	import PortalNavbar from '$lib/components/portal/PortalNavbar.svelte';

	let mobileOrVehicle = $state('JK-01-AB-4491');
	let crewPin = $state('2234');
	let isLoading = $state(false);
	let isLoggedIn = $state(false);
	let showPlacardModal = $state(false);

	let crewData = $state({
		name: 'Ghulam Nabi',
		role: 'Senior Chauffeur & Fleet Coordinator',
		vehicle: 'Toyota Innova Crysta AC (White)',
		plate: 'JK-01-AB-4491',
		phone: '+91 94190 22334',
		todayDuty: {
			pickupTime: '12:30 PM (Flight Inbound)',
			flight: 'IndiGo 6E-2144 (MAA ➔ SXR)',
			flightStatus: 'ON TIME • EXPECTED 12:28 PM',
			guestName: 'RAHUL SHARMA & FAMILY',
			passengerCount: '2 Adults, 2 Kids (4 Pax)',
			guestPhone: '+91 98840 11223',
			destination: 'SXR Airport ➔ Hotel Grand Kashmir, Pahalgam',
			placardText: 'SHARMA FAMILY',
			dutySlipNo: 'DS-2026-9041'
		}
	});

	function handleLogin(e: Event) {
		e.preventDefault();
		isLoading = true;
		setTimeout(() => {
			isLoading = false;
			isLoggedIn = true;
		}, 400);
	}

	function loadPresetCrew(name: string, role: string, vehicle: string, plate: string, phone: string, guest: string, placard: string) {
		crewData.name = name;
		crewData.role = role;
		crewData.vehicle = vehicle;
		crewData.plate = plate;
		crewData.phone = phone;
		crewData.todayDuty.guestName = guest;
		crewData.todayDuty.placardText = placard;
		mobileOrVehicle = plate;
		isLoggedIn = true;
	}

	function logout() {
		isLoggedIn = false;
	}
</script>

<svelte:head>
	<title>Driver & Guide Crew Portal | MrBiz OS</title>
	<meta name="description" content="Mobile-first duty portal for tour chauffeurs, licensed guides, and airport dispatch staff." />
</svelte:head>

<div class="crew-portal-page">
	<PortalNavbar />

	<div class="page-container">
		{#if !isLoggedIn}
			<!-- CREW LOGIN CARD -->
			<div class="login-wrapper">
				<div class="login-card">
					<div class="login-header">
						<div class="header-icon-badge">🚗</div>
						<div class="portal-badge">ON-GROUND CREW & DISPATCH</div>
						<h1 class="login-title">Driver & Guide Duty Login</h1>
						<p class="login-subtitle">Sign in with your registered Vehicle Number or Mobile PIN to access today's guest placards and duty slips.</p>
					</div>

					<form onsubmit={handleLogin} class="login-form">
						<div class="form-group">
							<label for="c-veh">Vehicle Plate No. or Registered Driver Mobile *</label>
							<div class="input-wrap">
								<span class="icon">🚘</span>
								<input id="c-veh" type="text" required bind:value={mobileOrVehicle} placeholder="e.g. JK-01-AB-4491 or 9419022334" class="input-field" />
							</div>
						</div>

						<div class="form-group">
							<label for="c-pin">4-Digit Duty Slip PIN *</label>
							<div class="input-wrap">
								<span class="icon">🔑</span>
								<input id="c-pin" type="password" maxlength="6" required bind:value={crewPin} class="input-field pin" />
							</div>
						</div>

						<button type="submit" class="btn-crew-login" disabled={isLoading}>
							{#if isLoading}
								<span class="spinner"></span>
								<span>Loading Daily Duty Roster...</span>
							{:else}
								<span>Access Today's Pickups</span>
								<span>➔</span>
							{/if}
						</button>
					</form>

					<!-- Demo Crew accounts -->
					<div class="demo-box">
						<div class="demo-title">⚡ 1-CLICK DEMO CREW LOGINS:</div>
						<div class="demo-buttons">
							<button
								type="button"
								class="demo-btn"
								onclick={() => loadPresetCrew('Ghulam Nabi', 'Senior Chauffeur', 'Toyota Innova Crysta AC (White)', 'JK-01-AB-4491', '+91 94190 22334', 'RAHUL SHARMA & FAMILY', 'SHARMA FAMILY')}
							>
								<span>🚗 Ghulam Nabi (Innova Crysta JK-01-AB-4491)</span>
							</button>
							<button
								type="button"
								class="demo-btn"
								onclick={() => loadPresetCrew('Bashir Ahmed', 'Govt. Licensed Tour Guide', 'Guide Desk / Valley Circuit', 'GUIDE-#2094', '+91 94190 55667', 'DR. JAYAPRAKASH & GUESTS', 'DR. JAYAPRAKASH')}
							>
								<span>🗣️ Bashir Ahmed (Govt. Licensed Guide)</span>
							</button>
						</div>
					</div>

					<div class="login-footer">
						<a href="/login/customer" class="link-muted">Looking for Traveler Login?</a>
					</div>
				</div>
			</div>
		{:else}
			<!-- CREW ACTIVE DUTY VIEW -->
			<div class="crew-dash-wrap">
				<!-- Top Bar -->
				<div class="crew-top">
					<div class="crew-profile-row">
						<div class="crew-avatar">GN</div>
						<div>
							<div class="crew-badge">ACTIVE DUTY ASSIGNED</div>
							<h2 class="crew-name">{crewData.name}</h2>
							<div class="crew-sub">{crewData.role} • 🚘 {crewData.plate}</div>
						</div>
					</div>
					<button type="button" class="btn-logout" onclick={logout}>Sign Out</button>
				</div>

				<!-- Main Duty Card -->
				<div class="duty-card">
					<div class="duty-header">
						<div>
							<div class="duty-pill">DUTY SLIP: {crewData.todayDuty.dutySlipNo}</div>
							<h3 class="duty-title">Today's Airport Arrival Meet & Greet</h3>
						</div>
						<button type="button" class="btn-open-placard" onclick={() => (showPlacardModal = true)}>
							<span>📱 Fullscreen Airport Placard</span>
						</button>
					</div>

					<div class="duty-grid-2">
						<div class="duty-box">
							<div class="d-label">Inbound Flight & Arrival Time</div>
							<div class="d-val bold">{crewData.todayDuty.flight}</div>
							<div class="d-sub green">Radar: {crewData.todayDuty.flightStatus}</div>
						</div>

						<div class="duty-box">
							<div class="d-label">Primary Guest & Pax Count</div>
							<div class="d-val">{crewData.todayDuty.guestName}</div>
							<div class="d-sub">{crewData.todayDuty.passengerCount}</div>
						</div>

						<div class="duty-box">
							<div class="d-label">Pickup Location</div>
							<div class="d-val">Srinagar Airport (SXR) - Exit Gate 1</div>
							<div class="d-sub">Holding Name Board: "{crewData.todayDuty.placardText}"</div>
						</div>

						<div class="duty-box">
							<div class="d-label">Hotel Drop-off Destination</div>
							<div class="d-val">{crewData.todayDuty.destination}</div>
							<div class="d-sub">Voucher Code: DMC-PHL-8819</div>
						</div>
					</div>

					<!-- Direct Passenger Actions -->
					<div class="duty-actions-row">
						<a
							href="https://wa.me/{crewData.todayDuty.guestPhone.replace(/\D/g, '')}?text=Hello%20{encodeURIComponent(crewData.todayDuty.guestName)}%2C%20this%20is%20your%20chauffeur%20{encodeURIComponent(crewData.name)}%20waiting%20at%20Airport%20Exit%20Gate%201%20with%20Innova%20Crysta%20{crewData.plate}."
							target="_blank"
							rel="noreferrer"
							class="btn-guest-wa"
						>
							<span>💬 WhatsApp Guest Directly</span>
						</a>
						<a href="tel:{crewData.todayDuty.guestPhone}" class="btn-guest-call">
							<span>📞 Call Guest</span>
						</a>
						<a
							href="https://maps.google.com/?q=Srinagar+International+Airport+to+Pahalgam"
							target="_blank"
							rel="noreferrer"
							class="btn-maps"
						>
							<span>📍 Start GPS Route</span>
						</a>
					</div>
				</div>
			</div>
		{/if}
	</div>
</div>

<!-- Fullscreen Airport Placard Modal for Drivers -->
{#if showPlacardModal}
	<div class="placard-modal-backdrop" onclick={() => (showPlacardModal = false)}>
		<div class="placard-screen">
			<div class="placard-top-hint">Tap anywhere to close • Hold screen up at Airport Exit Gate</div>
			<div class="placard-brand">MRBIZ TRAVEL & TOURS CONCIERGE</div>
			<div class="placard-huge-name">{crewData.todayDuty.placardText}</div>
			<div class="placard-sub-info">{crewData.todayDuty.passengerCount} • {crewData.todayDuty.flight}</div>
			<div class="placard-footer-plate">Chauffeur: {crewData.name} ({crewData.plate})</div>
		</div>
	</div>
{/if}

<style>
	.crew-portal-page {
		min-height: 100vh;
		background: #090d16;
		color: #f8fafc;
		font-family: var(--font-sans, system-ui, -apple-system, sans-serif);
	}

	.page-container {
		max-width: 900px;
		margin: 0 auto;
		padding: 3rem 1.5rem;
	}

	.login-wrapper {
		max-width: 480px;
		margin: 2rem auto;
	}

	.login-card {
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.12);
		border-radius: 20px;
		padding: 2.25rem;
		box-shadow: 0 20px 50px rgba(0, 0, 0, 0.55);
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
		background: rgba(245, 158, 11, 0.15);
		border: 1px solid rgba(245, 158, 11, 0.35);
		display: flex;
		align-items: center;
		justify-content: center;
		margin: 0 auto 0.75rem;
	}

	.portal-badge {
		font-size: 0.72rem;
		font-weight: 800;
		color: #f59e0b;
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

	.login-form {
		display: flex;
		flex-direction: column;
		gap: 1.1rem;
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
		border-color: #f59e0b;
	}

	.input-field.pin {
		letter-spacing: 0.3em;
		font-weight: 800;
	}

	.btn-crew-login {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.5rem;
		padding: 0.85rem;
		border-radius: 12px;
		background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);
		border: none;
		color: #ffffff;
		font-weight: 800;
		font-size: 0.95rem;
		cursor: pointer;
		box-shadow: 0 4px 14px rgba(245, 158, 11, 0.4);
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

	.demo-box {
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
		padding: 0.65rem 0.85rem;
		border-radius: 8px;
		background: rgba(255, 255, 255, 0.04);
		border: 1px solid rgba(255, 255, 255, 0.08);
		color: #cbd5e1;
		font-size: 0.8rem;
		font-weight: 600;
		cursor: pointer;
	}

	.demo-btn:hover {
		background: rgba(245, 158, 11, 0.15);
		border-color: #f59e0b;
		color: #ffffff;
	}

	.login-footer {
		text-align: center;
		margin-top: 1.5rem;
	}

	.link-muted { color: #94a3b8; font-size: 0.8rem; text-decoration: none; }

	/* ACTIVE DUTY VIEW */
	.crew-dash-wrap {
		display: flex;
		flex-direction: column;
		gap: 1.5rem;
	}

	.crew-top {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding-bottom: 1.25rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
	}

	.crew-profile-row {
		display: flex;
		align-items: center;
		gap: 1rem;
	}

	.crew-avatar {
		width: 50px;
		height: 50px;
		border-radius: 50%;
		background: #f59e0b;
		color: #ffffff;
		font-weight: 800;
		font-size: 1.2rem;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.crew-badge {
		font-size: 0.68rem;
		font-weight: 800;
		color: #34d399;
		letter-spacing: 0.08em;
	}

	.crew-name {
		font-size: 1.5rem;
		font-weight: 800;
		color: #ffffff;
		margin: 0.15rem 0;
	}

	.crew-sub {
		font-size: 0.82rem;
		color: #94a3b8;
	}

	.btn-logout {
		padding: 0.65rem 1.25rem;
		border-radius: 10px;
		background: rgba(239, 68, 68, 0.15);
		border: 1px solid rgba(239, 68, 68, 0.35);
		color: #fca5a5;
		font-weight: 700;
		cursor: pointer;
	}

	/* Duty Card */
	.duty-card {
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: 18px;
		padding: 1.75rem;
		box-shadow: 0 16px 40px rgba(0, 0, 0, 0.4);
	}

	.duty-header {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		margin-bottom: 1.5rem;
		flex-wrap: wrap;
		gap: 1rem;
	}

	.duty-pill {
		font-size: 0.72rem;
		font-weight: 800;
		color: #f59e0b;
		background: rgba(245, 158, 11, 0.15);
		padding: 0.2rem 0.55rem;
		border-radius: 6px;
		display: inline-block;
		margin-bottom: 0.35rem;
	}

	.duty-title {
		font-size: 1.35rem;
		font-weight: 800;
		color: #ffffff;
		margin: 0;
	}

	.btn-open-placard {
		display: flex;
		align-items: center;
		gap: 0.45rem;
		padding: 0.7rem 1.2rem;
		background: linear-gradient(135deg, #0284c7 0%, #2563eb 100%);
		border: none;
		border-radius: 10px;
		color: #ffffff;
		font-size: 0.88rem;
		font-weight: 800;
		cursor: pointer;
	}

	.duty-grid-2 {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 1rem;
		margin-bottom: 1.5rem;
	}

	.duty-box {
		background: rgba(255, 255, 255, 0.02);
		border: 1px solid rgba(255, 255, 255, 0.06);
		border-radius: 12px;
		padding: 1rem;
	}

	.d-label { font-size: 0.72rem; color: #64748b; margin-bottom: 0.25rem; }
	.d-val { font-size: 0.95rem; font-weight: 700; color: #ffffff; }
	.d-val.bold { color: #38bdf8; }
	.d-sub { font-size: 0.78rem; color: #94a3b8; margin-top: 0.2rem; }
	.d-sub.green { color: #34d399; font-weight: 700; }

	.duty-actions-row {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 0.75rem;
	}

	.btn-guest-wa {
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 0.85rem;
		background: #25d366;
		color: #ffffff;
		font-weight: 800;
		font-size: 0.88rem;
		border-radius: 10px;
		text-decoration: none;
	}

	.btn-guest-call {
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 0.85rem;
		background: rgba(255, 255, 255, 0.08);
		color: #ffffff;
		font-weight: 700;
		font-size: 0.88rem;
		border-radius: 10px;
		text-decoration: none;
	}

	.btn-maps {
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 0.85rem;
		background: #0284c7;
		color: #ffffff;
		font-weight: 700;
		font-size: 0.88rem;
		border-radius: 10px;
		text-decoration: none;
	}

	/* Fullscreen Placard */
	.placard-modal-backdrop {
		position: fixed;
		inset: 0;
		z-index: 3000;
		background: #ffffff;
		color: #000000;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 2rem;
		cursor: pointer;
	}

	.placard-screen {
		text-align: center;
		max-width: 900px;
	}

	.placard-top-hint {
		font-size: 0.85rem;
		color: #64748b;
		margin-bottom: 2rem;
	}

	.placard-brand {
		font-size: 1.2rem;
		font-weight: 900;
		letter-spacing: 0.1em;
		color: #0284c7;
		margin-bottom: 1.5rem;
	}

	.placard-huge-name {
		font-size: clamp(3rem, 8vw, 5.5rem);
		font-weight: 900;
		line-height: 1.05;
		color: #000000;
		letter-spacing: 0.04em;
		margin-bottom: 1.5rem;
	}

	.placard-sub-info {
		font-size: 1.5rem;
		font-weight: 700;
		color: #334155;
		margin-bottom: 2rem;
	}

	.placard-footer-plate {
		font-size: 1.2rem;
		font-weight: 800;
		background: #f1f5f9;
		padding: 0.75rem 1.5rem;
		border-radius: 999px;
		display: inline-block;
	}

	@media (max-width: 768px) {
		.duty-grid-2, .duty-actions-row {
			grid-template-columns: 1fr;
		}
	}
</style>
