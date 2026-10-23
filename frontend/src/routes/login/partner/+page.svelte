<script lang="ts">
	import PortalNavbar from '$lib/components/portal/PortalNavbar.svelte';

	let partnerIdOrEmail = $state('sales@maduraipandiantravels.in');
	let partnerPassword = $state('••••••••••••');
	let rememberPartner = $state(true);
	let isLoading = $state(false);
	let isLoggedIn = $state(false);
	let errorMessage = $state('');

	// Active B2B Partner Data
	let partnerData = $state({
		agencyName: 'Madurai Pandian Travel World',
		partnerId: 'B2B-TN-4412',
		contactPerson: 'S. Alagarsamy (Managing Partner)',
		tier: 'Gold Accredited Agency',
		commissionRate: '12.5%',
		creditLimit: 300000,
		availableCredit: 235000,
		currentReceivable: 65000,
		monthlyTripsVolume: 14,
		monthlyCommissionEarned: 68500,
		activeBookings: [
			{ id: 'BK-KSH-8849', guest: 'Rahul Sharma & Family (4 Pax)', circuit: 'Kashmir Valley Paradise (5N/6D)', netFare: 54000, commission: 6750, status: 'CONFIRMED' },
			{ id: 'BK-KER-2291', guest: 'Dr. Jayaprakash (2 Pax)', circuit: 'Kerala Backwaters & Munnar (4N/5D)', netFare: 38000, commission: 4750, status: 'VOUCHER ISSUED' },
			{ id: 'BK-DXB-9931', guest: 'TVS Sundaram Corporate Group (8 Pax)', circuit: 'Dubai MICE & Desert Safari (4N/5D)', netFare: 220000, commission: 27500, status: 'DISPATCH READY' }
		]
	});

	function handlePartnerLogin(e: Event) {
		e.preventDefault();
		isLoading = true;
		errorMessage = '';

		setTimeout(() => {
			isLoading = false;
			isLoggedIn = true;
		}, 450);
	}

	function loadPresetPartner(agency: string, id: string, contact: string, tier: string, comm: string, limit: number, avail: number, pending: number, earned: number) {
		partnerData.agencyName = agency;
		partnerData.partnerId = id;
		partnerData.contactPerson = contact;
		partnerData.tier = tier;
		partnerData.commissionRate = comm;
		partnerData.creditLimit = limit;
		partnerData.availableCredit = avail;
		partnerData.currentReceivable = pending;
		partnerData.monthlyCommissionEarned = earned;
		partnerIdOrEmail = id;
		isLoggedIn = true;
	}

	function logout() {
		isLoggedIn = false;
	}
</script>

<svelte:head>
	<title>B2B Partner & Travel Agency Login | MrBiz OS</title>
	<meta name="description" content="Dedicated B2B travel partner portal for travel agencies, DMCs, and corporate desks to manage wholesale bookings, credit khata, and commission statements." />
</svelte:head>

<div class="partner-login-page">
	<PortalNavbar activeTab="partners" />

	<div class="page-container">
		{#if !isLoggedIn}
			<!-- B2B LOGIN CARD -->
			<div class="login-wrapper">
				<div class="login-card">
					<div class="login-header">
						<div class="header-icon-badge">🤝</div>
						<div class="portal-badge">B2B TRAVEL PARTNER & DMC HUB</div>
						<h1 class="login-title">Travel Agency Partner Login</h1>
						<p class="login-subtitle">Access your wholesale DMC tariffs, sub-agent credit khata, and client booking rosters.</p>
					</div>

					{#if errorMessage}
						<div class="error-banner">
							<span>⚠️ {errorMessage}</span>
						</div>
					{/if}

					<form onsubmit={handlePartnerLogin} class="login-form">
						<div class="form-group">
							<label for="p-id">Partner Code / Agency IATA / Email Address *</label>
							<div class="input-wrap">
								<span class="icon">🏢</span>
								<input id="p-id" type="text" required bind:value={partnerIdOrEmail} placeholder="e.g. B2B-TN-4412 or sales@agency.com" class="input-field" />
							</div>
						</div>

						<div class="form-group">
							<label for="p-pwd">Partner Secret Password *</label>
							<div class="input-wrap">
								<span class="icon">🔒</span>
								<input id="p-pwd" type="password" required bind:value={partnerPassword} class="input-field" />
							</div>
						</div>

						<div class="remember-row">
							<label class="remember-label">
								<input type="checkbox" bind:checked={rememberPartner} />
								<span>Remember agency session on this terminal</span>
							</label>
							<a href="#forgot" class="forgot-link">Reset Agency Key?</a>
						</div>

						<button type="submit" class="btn-partner-login" disabled={isLoading}>
							{#if isLoading}
								<span class="spinner"></span>
								<span>Authenticating B2B Credentials...</span>
							{:else}
								<span>Sign In to B2B Partner Desk</span>
								<span>➔</span>
							{/if}
						</button>
					</form>

					<!-- 1-Click Demo Partner Accounts -->
					<div class="demo-partners-box">
						<div class="demo-title">⚡ 1-CLICK DEMO PARTNER LOGINS:</div>
						<div class="demo-buttons">
							<button
								type="button"
								class="demo-btn"
								onclick={() => loadPresetPartner('Madurai Pandian Travel World', 'B2B-TN-4412', 'S. Alagarsamy', 'Gold Accredited Agency', '12.5%', 300000, 235000, 65000, 68500)}
							>
								<span>🥇 Madurai Pandian Travel World (Gold Tier - ₹3L Khata)</span>
							</button>
							<button
								type="button"
								class="demo-btn"
								onclick={() => loadPresetPartner('Falcon Inbound DMC Dubai', 'B2B-DXB-9901', 'Rashid Al Nuaimi', 'Platinum Master DMC', '15.0%', 1000000, 840000, 160000, 194000)}
							>
								<span>💎 Falcon Inbound DMC Dubai (Platinum - ₹10L Khata)</span>
							</button>
						</div>
					</div>

					<div class="partner-footer-links">
						<span>Not a registered travel partner yet?</span>
						<a href="/travel/partners#register" class="register-link">Apply for B2B Partnership</a>
					</div>
				</div>
			</div>
		{:else}
			<!-- B2B PARTNER DASHBOARD & KHATA VIEW -->
			<div class="partner-dash-wrap">
				<!-- Header Bar -->
				<div class="dash-top">
					<div>
						<div class="partner-tier-badge">{partnerData.tier}</div>
						<h2 class="agency-title">{partnerData.agencyName}</h2>
						<div class="agency-meta">PARTNER CODE: {partnerData.partnerId} • CONTACT: {partnerData.contactPerson}</div>
					</div>
					<div class="dash-actions">
						<a href="/travel/booking" class="btn-new-booking">
							<span>+ Create B2B Net Booking</span>
						</a>
						<button type="button" class="btn-logout" onclick={logout}>
							<span>Sign Out</span>
						</button>
					</div>
				</div>

				<!-- Financial Khata Ledger Cards -->
				<div class="khata-cards-grid">
					<div class="khata-card">
						<div class="khata-label">APPROVED CREDIT LINE</div>
						<div class="khata-val">₹{partnerData.creditLimit.toLocaleString('en-IN')}</div>
						<div class="khata-sub">15-Day Rolling Sub-Agent Facility</div>
					</div>

					<div class="khata-card highlight">
						<div class="khata-label">AVAILABLE NET BOOKING CREDIT</div>
						<div class="khata-val green">₹{partnerData.availableCredit.toLocaleString('en-IN')}</div>
						<div class="khata-sub">Instant zero-deposit ticket issuance</div>
					</div>

					<div class="khata-card">
						<div class="khata-label">CURRENT OUTSTANDING (RECEIVABLE)</div>
						<div class="khata-val warning">₹{partnerData.currentReceivable.toLocaleString('en-IN')}</div>
						<div class="khata-sub">Next reconciliation settlement: 1st Nov</div>
					</div>

					<div class="khata-card commission">
						<div class="khata-label">MONTHLY EARNED OVERRIDE</div>
						<div class="khata-val purple">₹{partnerData.monthlyCommissionEarned.toLocaleString('en-IN')}</div>
						<div class="khata-sub">{partnerData.commissionRate} Margin on {partnerData.monthlyTripsVolume} Booked Trips</div>
					</div>
				</div>

				<!-- Active Bookings & Tariffs -->
				<div class="bookings-section-card">
					<div class="bookings-header">
						<div>
							<h3 class="b-title">Active Agency Bookings & Roster</h3>
							<p class="b-sub">Live status of your clients currently traveling or departing this week.</p>
						</div>
						<a href="/travel/packages" class="btn-tariffs">
							<span>Browse B2B Wholesale Tariffs</span>
							<span>➔</span>
						</a>
					</div>

					<div class="bookings-table-wrap">
						<table class="bookings-table">
							<thead>
								<tr>
									<th>BOOKING ID</th>
									<th>CLIENT / GUEST NAME</th>
									<th>CIRCUIT</th>
									<th>B2B NET FARE</th>
									<th>AGENCY OVERRIDE</th>
									<th>DISPATCH STATUS</th>
									<th>ACTION</th>
								</tr>
							</thead>
							<tbody>
								{#each partnerData.activeBookings as b}
									<tr>
										<td class="bold-cell">{b.id}</td>
										<td>{b.guest}</td>
										<td>{b.circuit}</td>
										<td class="rate-cell">₹{b.netFare.toLocaleString('en-IN')}</td>
										<td class="comm-cell">+₹{b.commission.toLocaleString('en-IN')}</td>
										<td>
											<span class="status-chip">{b.status}</span>
										</td>
										<td>
											<a href="/travel/booking" class="view-btn">View Voucher</a>
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				</div>
			</div>
		{/if}
	</div>
</div>

<style>
	.partner-login-page {
		min-height: 100vh;
		background: #090d16;
		color: #f8fafc;
		font-family: var(--font-sans, system-ui, -apple-system, sans-serif);
	}

	.page-container {
		max-width: 1240px;
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
		background: rgba(16, 185, 129, 0.15);
		border: 1px solid rgba(52, 211, 153, 0.35);
		display: flex;
		align-items: center;
		justify-content: center;
		margin: 0 auto 0.75rem;
	}

	.portal-badge {
		font-size: 0.72rem;
		font-weight: 800;
		color: #34d399;
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

	.error-banner {
		background: rgba(239, 68, 68, 0.15);
		border: 1px solid rgba(239, 68, 68, 0.35);
		color: #fca5a5;
		padding: 0.65rem 0.85rem;
		border-radius: 8px;
		font-size: 0.82rem;
		margin-bottom: 1rem;
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
		border-color: #34d399;
	}

	.remember-row {
		display: flex;
		justify-content: space-between;
		align-items: center;
		font-size: 0.78rem;
	}

	.remember-label {
		display: flex;
		align-items: center;
		gap: 0.45rem;
		color: #94a3b8;
		cursor: pointer;
	}

	.forgot-link {
		color: #34d399;
		text-decoration: none;
		font-weight: 600;
	}

	.btn-partner-login {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.5rem;
		padding: 0.85rem;
		border-radius: 12px;
		background: linear-gradient(135deg, #10b981 0%, #059669 100%);
		border: none;
		color: #ffffff;
		font-weight: 800;
		font-size: 0.95rem;
		cursor: pointer;
		margin-top: 0.25rem;
		box-shadow: 0 4px 14px rgba(16, 185, 129, 0.4);
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

	/* Demo box */
	.demo-partners-box {
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
		background: rgba(16, 185, 129, 0.15);
		border-color: #34d399;
		color: #ffffff;
	}

	.partner-footer-links {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.5rem;
		font-size: 0.78rem;
		color: #94a3b8;
		margin-top: 1.5rem;
	}

	.register-link {
		color: #34d399;
		font-weight: 700;
		text-decoration: none;
	}

	/* DASHBOARD VIEW */
	.partner-dash-wrap {
		display: flex;
		flex-direction: column;
		gap: 1.5rem;
	}

	.dash-top {
		display: flex;
		justify-content: space-between;
		align-items: center;
		flex-wrap: wrap;
		gap: 1rem;
		padding-bottom: 1.5rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
	}

	.partner-tier-badge {
		font-size: 0.72rem;
		font-weight: 800;
		color: #f59e0b;
		letter-spacing: 0.08em;
	}

	.agency-title {
		font-size: 1.8rem;
		font-weight: 900;
		color: #ffffff;
		margin: 0.2rem 0;
	}

	.agency-meta {
		font-size: 0.82rem;
		color: #94a3b8;
	}

	.dash-actions {
		display: flex;
		gap: 0.75rem;
	}

	.btn-new-booking {
		padding: 0.7rem 1.4rem;
		border-radius: 10px;
		background: linear-gradient(135deg, #10b981 0%, #059669 100%);
		color: #ffffff;
		font-weight: 800;
		font-size: 0.88rem;
		text-decoration: none;
		box-shadow: 0 4px 14px rgba(16, 185, 129, 0.35);
	}

	.btn-logout {
		padding: 0.7rem 1.25rem;
		border-radius: 10px;
		background: rgba(239, 68, 68, 0.15);
		border: 1px solid rgba(239, 68, 68, 0.35);
		color: #fca5a5;
		font-weight: 700;
		font-size: 0.85rem;
		cursor: pointer;
	}

	/* Khata Cards */
	.khata-cards-grid {
		display: grid;
		grid-template-columns: repeat(4, 1fr);
		gap: 1.25rem;
	}

	.khata-card {
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 16px;
		padding: 1.25rem;
		display: flex;
		flex-direction: column;
	}

	.khata-card.highlight {
		border-color: rgba(16, 185, 129, 0.35);
		background: rgba(16, 185, 129, 0.05);
	}

	.khata-card.commission {
		border-color: rgba(168, 85, 247, 0.35);
		background: rgba(168, 85, 247, 0.05);
	}

	.khata-label {
		font-size: 0.7rem;
		font-weight: 800;
		color: #64748b;
		margin-bottom: 0.35rem;
	}

	.khata-val {
		font-size: 1.6rem;
		font-weight: 900;
		color: #ffffff;
		margin-bottom: 0.25rem;
	}

	.khata-val.green { color: #34d399; }
	.khata-val.warning { color: #f59e0b; }
	.khata-val.purple { color: #c084fc; }

	.khata-sub {
		font-size: 0.75rem;
		color: #94a3b8;
	}

	/* Bookings Table */
	.bookings-section-card {
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 16px;
		padding: 1.5rem;
	}

	.bookings-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 1.25rem;
		flex-wrap: wrap;
		gap: 0.75rem;
	}

	.b-title {
		font-size: 1.2rem;
		font-weight: 800;
		color: #ffffff;
		margin: 0 0 0.25rem;
	}

	.b-sub {
		font-size: 0.82rem;
		color: #94a3b8;
		margin: 0;
	}

	.btn-tariffs {
		display: flex;
		align-items: center;
		gap: 0.45rem;
		padding: 0.55rem 1rem;
		border-radius: 8px;
		background: rgba(255, 255, 255, 0.06);
		border: 1px solid rgba(255, 255, 255, 0.12);
		color: #ffffff;
		font-size: 0.82rem;
		font-weight: 700;
		text-decoration: none;
	}

	.bookings-table-wrap {
		overflow-x: auto;
	}

	.bookings-table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.85rem;
		text-align: left;
	}

	.bookings-table th {
		padding: 0.75rem;
		font-size: 0.72rem;
		font-weight: 800;
		color: #64748b;
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
	}

	.bookings-table td {
		padding: 0.9rem 0.75rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.04);
		color: #cbd5e1;
	}

	.bold-cell { font-weight: 700; color: #ffffff; }
	.rate-cell { font-weight: 700; color: #f1f5f9; }
	.comm-cell { font-weight: 800; color: #34d399; }

	.status-chip {
		font-size: 0.72rem;
		font-weight: 800;
		padding: 0.2rem 0.55rem;
		border-radius: 999px;
		background: rgba(16, 185, 129, 0.15);
		color: #34d399;
	}

	.view-btn {
		font-size: 0.78rem;
		font-weight: 700;
		color: #38bdf8;
		text-decoration: none;
	}

	@media (max-width: 1024px) {
		.khata-cards-grid {
			grid-template-columns: repeat(2, 1fr);
		}
	}

	@media (max-width: 640px) {
		.khata-cards-grid {
			grid-template-columns: 1fr;
		}
	}
</style>
