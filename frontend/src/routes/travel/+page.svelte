<script lang="ts">
	import { onMount } from 'svelte';
	import { travelGrpc } from '$lib/grpc';

	interface ItineraryDay {
		day: number;
		title: string;
		activities: string;
		hotel: string;
		meals: string; // e.g., 'Breakfast & Dinner'
	}

	interface TourPackage {
		id: string;
		tenantId: string;
		title: string;
		destination: string;
		durationDays: number;
		price: string;
		itineraryJson?: string;
		isActive?: boolean;
		hotelRating?: string;
		inclusions?: string[];
	}

	interface VisaApplication {
		id: string;
		tenantId: string;
		applicantName: string;
		passportNumber: string;
		country: string;
		visaType: string;
		status: string;
		checklistJson?: string;
		submittedAt: string;
		appointmentDate?: string;
		phone?: string;
	}

	let activeTab = $state<'packages' | 'visas' | 'calculator'>('packages');
	let packages = $state<TourPackage[]>([]);
	let applications = $state<VisaApplication[]>([]);
	let loading = $state(true);
	let error = $state('');
	let statusMsg = $state('');

	// Selected package for itinerary preview & print
	let selectedPackage = $state<TourPackage | null>(null);
	let parsedItinerary = $state<ItineraryDay[]>([]);
	let showItineraryModal = $state(false);

	// Modals
	let showNewPackageModal = $state(false);
	let showNewVisaModal = $state(false);

	// New Package Form State
	let newPkgTitle = $state('');
	let newPkgDestination = $state('');
	let newPkgDuration = $state(5);
	let newPkgPrice = $state('35000');
	let newPkgDays = $state<ItineraryDay[]>([
		{ day: 1, title: 'Arrival & Hotel Check-in', activities: 'Airport pickup, transfer to 4-star hotel, welcome dinner cruise.', hotel: 'Radisson Blu / Equivalent', meals: 'Dinner' },
		{ day: 2, title: 'Guided City Tour & Landmark Sightseeing', activities: 'Half-day city tour with English-speaking guide, entry tickets to main museum.', hotel: 'Radisson Blu / Equivalent', meals: 'Breakfast' },
		{ day: 3, title: 'Leisure & Adventure Excursions', activities: 'Desert safari / Island hopping / Theme park access with transfers.', hotel: 'Radisson Blu / Equivalent', meals: 'Breakfast & Dinner' },
		{ day: 4, title: 'Local Markets & Shopping Tour', activities: 'Free day for gold souk, artisan market, local cuisine walk.', hotel: 'Radisson Blu / Equivalent', meals: 'Breakfast' },
		{ day: 5, title: 'Departure & Airport Transfer', activities: 'Breakfast, check-out, souvenir shopping, private transfer to airport.', hotel: 'Checkout', meals: 'Breakfast' }
	]);

	// New Visa Form State
	let newVisaApplicant = $state('');
	let newVisaPassport = $state('');
	let newVisaPhone = $state('');
	let newVisaCountry = $state('United Arab Emirates');
	let newVisaType = $state('30 Days Tourist');
	let newVisaChecklist = $state({
		passportSixMonthsValid: true,
		photographWhiteBg: true,
		bankStatementSixMonths: false,
		itrThreeYears: false,
		airTicketConfirmed: false,
		hotelVoucherConfirmed: false,
		travelInsurance: false
	});

	// Secret Margin & Indian TCS Calculator State
	let calcVendorFlight = $state(28000);
	let calcVendorHotel = $state(22000);
	let calcVendorLandTransport = $state(12000);
	let calcVendorSightseeing = $state(8000);
	let calcAgentMarkupPercent = $state(15);
	let calcTcsTier = $state<'NONE' | 'DOMESTIC' | 'OVERSEAS_5' | 'OVERSEAS_20'>('OVERSEAS_5');

	// Computed Margin Calculator values
	let totalVendorNet = $derived(calcVendorFlight + calcVendorHotel + calcVendorLandTransport + calcVendorSightseeing);
	let agentProfitMargin = $derived(Math.round(totalVendorNet * (calcAgentMarkupPercent / 100)));
	let baseSellingPrice = $derived(totalVendorNet + agentProfitMargin);
	let tcsRatePercent = $derived(calcTcsTier === 'OVERSEAS_5' ? 5 : calcTcsTier === 'OVERSEAS_20' ? 20 : 0);
	let tcsAmount = $derived(Math.round(baseSellingPrice * (tcsRatePercent / 100)));
	let finalCustomerQuotation = $derived(baseSellingPrice + tcsAmount);

	async function loadData() {
		loading = true;
		error = '';
		try {
			const [pkgRes, appRes]: [any, any] = await Promise.all([
				travelGrpc.listTourPackages(),
				travelGrpc.listVisaApplications()
			]);
			packages = pkgRes?.packages || [];
			applications = appRes?.applications || [];
			if (packages.length > 0 && !selectedPackage) {
				openItineraryPreview(packages[0]);
			}
		} catch (err: any) {
			error = err.message || 'Failed to connect to Travel & Visa gRPC backend';
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		loadData();
	});

	function openItineraryPreview(pkg: TourPackage) {
		selectedPackage = pkg;
		if (pkg.itineraryJson) {
			try {
				parsedItinerary = JSON.parse(pkg.itineraryJson);
			} catch {
				parsedItinerary = [
					{ day: 1, title: 'Arrival & Welcome', activities: 'Airport pickup and hotel check-in', hotel: 'City Hotel', meals: 'Dinner' },
					{ day: 2, title: 'Full Day Sightseeing', activities: 'Major landmarks and cultural tour', hotel: 'City Hotel', meals: 'Breakfast' },
					{ day: 3, title: 'Return Transfer', activities: 'Hotel check-out and drop to airport', hotel: 'N/A', meals: 'Breakfast' }
				];
			}
		} else {
			parsedItinerary = [
				{ day: 1, title: 'Arrival & Transfer', activities: 'Welcome reception and hotel check-in', hotel: 'Standard 4-Star', meals: 'Dinner' },
				{ day: 2, title: 'Local Attractions', activities: 'Guided city landmarks tour', hotel: 'Standard 4-Star', meals: 'Breakfast' }
			];
		}
		showItineraryModal = true;
	}

	async function handleCreatePackage() {
		if (!newPkgTitle || !newPkgDestination) {
			alert('Please enter package title and destination');
			return;
		}
		try {
			const res: any = await travelGrpc.createTourPackage({
				title: newPkgTitle,
				destination: newPkgDestination,
				durationDays: Number(newPkgDuration),
				price: String(newPkgPrice),
				itineraryJson: JSON.stringify(newPkgDays)
			});
			if (res?.package) {
				packages = [res.package, ...packages];
				showNewPackageModal = false;
				statusMsg = `✅ Tour package "${res.package.title}" published!`;
				setTimeout(() => statusMsg = '', 4000);
			}
		} catch (err: any) {
			alert('Failed to create package: ' + err.message);
		}
	}

	async function handleSubmitVisa() {
		if (!newVisaApplicant || !newVisaPassport) {
			alert('Please provide applicant name and passport number');
			return;
		}
		try {
			const res: any = await travelGrpc.submitVisaApplication({
				applicantName: newVisaApplicant,
				passportNumber: newVisaPassport,
				country: newVisaCountry,
				visaType: newVisaType,
				checklistJson: JSON.stringify(newVisaChecklist)
			});
			if (res?.application) {
				applications = [res.application, ...applications];
				showNewVisaModal = false;
				newVisaApplicant = '';
				newVisaPassport = '';
				statusMsg = `✅ Visa file submitted for ${res.application.applicantName} (${res.application.country})`;
				setTimeout(() => statusMsg = '', 4000);
			}
		} catch (err: any) {
			alert('Failed to submit visa: ' + err.message);
		}
	}

	async function updateVisaStatus(id: string, newStatus: string) {
		try {
			await travelGrpc.updateVisaStatus({ id, status: newStatus });
			applications = applications.map(a => a.id === id ? { ...a, status: newStatus } : a);
			statusMsg = `Status updated to ${newStatus}`;
			setTimeout(() => statusMsg = '', 3000);
		} catch (err: any) {
			alert('Failed to update status: ' + err.message);
		}
	}

	function printItinerary() {
		window.print();
	}
</script>

<svelte:head>
	<title>Travel, Tours & Visa Management OS | MrBiz-OS</title>
</svelte:head>

<div class="travel-app">
	<!-- Top Bar -->
	<header class="app-header">
		<div>
			<div class="badge-row">
				<span class="pill-badge">✈️ Global Travel & Visa OS</span>
				<span class="status-indicator">● ConnectRPC v1 Connected</span>
			</div>
			<h1 class="main-title">Tour Operator & Immigration Desk</h1>
			<p class="subtitle">Day-by-day itinerary designer, secret B2B margin calculator, Indian 206C(1G) TCS engine, and Embassy visa tracking.</p>
		</div>

		<div class="header-actions">
			<button class="btn btn-secondary" onclick={() => showNewVisaModal = true}>
				🛂 + New Visa File
			</button>
			<button class="btn btn-primary" onclick={() => showNewPackageModal = true}>
				✈️ + New Tour Package
			</button>
		</div>
	</header>

	{#if statusMsg}
		<div class="status-alert">{statusMsg}</div>
	{/if}

	{#if error}
		<div class="error-alert">⚠️ {error}</div>
	{/if}

	<!-- Quick KPI Metric Cards -->
	<div class="kpi-grid">
		<div class="kpi-card">
			<div class="kpi-label">Active Holiday Packages</div>
			<div class="kpi-value">{packages.length}</div>
			<div class="kpi-sub text-emerald-400">Domestic & International</div>
		</div>
		<div class="kpi-card">
			<div class="kpi-label">Visa Applications Pipeline</div>
			<div class="kpi-value text-sky-400">{applications.length}</div>
			<div class="kpi-sub">Embassy & VFS Tracking</div>
		</div>
		<div class="kpi-card">
			<div class="kpi-label">TCS Compliance Engine</div>
			<div class="kpi-value text-amber-400">Sec 206C(1G)</div>
			<div class="kpi-sub">5% / 20% Auto-TCS Slabs</div>
		</div>
		<div class="kpi-card">
			<div class="kpi-label">B2B Margin Guard</div>
			<div class="kpi-value text-purple-400">{calcAgentMarkupPercent}% Markup</div>
			<div class="kpi-sub">Confidential Net Costing</div>
		</div>
	</div>

	<!-- Navigation Tabs -->
	<div class="nav-tabs">
		<button 
			class="tab-btn" 
			class:active={activeTab === 'packages'} 
			onclick={() => activeTab = 'packages'}>
			✈️ Holiday Packages & Itineraries ({packages.length})
		</button>
		<button 
			class="tab-btn" 
			class:active={activeTab === 'visas'} 
			onclick={() => activeTab = 'visas'}>
			🛂 Visa & Immigration Tracker ({applications.length})
		</button>
		<button 
			class="tab-btn" 
			class:active={activeTab === 'calculator'} 
			onclick={() => activeTab = 'calculator'}>
			🧮 B2B Secret Margin & TCS 206C Calculator
		</button>
	</div>

	<!-- TAB 1: TOUR PACKAGES -->
	{#if activeTab === 'packages'}
		<div class="section-container">
			<div class="section-header">
				<div>
					<h2 class="section-title">Featured Itineraries & Tour Packages</h2>
					<p class="section-desc">Client proposals ready for 1-click WhatsApp share and PDF voucher printing.</p>
				</div>
				<button class="btn btn-outline" onclick={loadData}>🔄 Refresh</button>
			</div>

			{#if loading}
				<div class="loading-state">Fetching packages from TravelConnect service...</div>
			{:else}
				<div class="packages-grid">
					{#each packages as pkg}
						<div class="pkg-card">
							<div class="pkg-header">
								<span class="destination-pill">📍 {pkg.destination}</span>
								<span class="duration-badge">⏱️ {pkg.durationDays} Days / {pkg.durationDays - 1} Nights</span>
							</div>
							<h3 class="pkg-title">{pkg.title}</h3>
							<div class="pkg-highlights">
								<span>🏨 4★ Accommodation</span>
								<span>🚗 Airport Transfers</span>
								<span>🍽️ Daily Breakfast</span>
								<span>🎟️ Entry Sightseeing</span>
							</div>
							<div class="pkg-footer">
								<div>
									<div class="price-label">Starting from</div>
									<div class="price-val">₹{Number(pkg.price).toLocaleString('en-IN')} <span class="per-person">/ pax</span></div>
								</div>
								<div class="btn-group">
									<button class="btn-view" onclick={() => openItineraryPreview(pkg)}>
										📜 View Itinerary
									</button>
								</div>
							</div>
						</div>
					{/each}
				</div>
			{/if}
		</div>
	{/if}

	<!-- TAB 2: VISA APPLICATIONS -->
	{#if activeTab === 'visas'}
		<div class="section-container">
			<div class="section-header">
				<div>
					<h2 class="section-title">Visa & Immigration Case Pipeline</h2>
					<p class="section-desc">Real-time status tracking, document completeness check, and biometrics scheduling.</p>
				</div>
				<button class="btn btn-primary" onclick={() => showNewVisaModal = true}>+ Log New Visa File</button>
			</div>

			<div class="table-card">
				<table class="data-table">
					<thead>
						<tr>
							<th>Applicant Name</th>
							<th>Passport Number</th>
							<th>Destination Country</th>
							<th>Visa Category</th>
							<th>Checklist Verification</th>
							<th>Current Embassy Status</th>
							<th>Actions</th>
						</tr>
					</thead>
					<tbody>
						{#each applications as app}
							<tr>
								<td>
									<div class="font-semibold text-white">{app.applicantName}</div>
									<div class="text-xs text-slate-400">Ref: {app.id}</div>
								</td>
								<td>
									<span class="mono-badge">{app.passportNumber}</span>
								</td>
								<td>
									<span class="country-cell">🌍 {app.country}</span>
								</td>
								<td>{app.visaType}</td>
								<td>
									<div class="checklist-pills">
										<span class="check-pill done">✓ Passport Valid</span>
										<span class="check-pill done">✓ Photo 35x45</span>
										<span class="check-pill" class:done={app.status !== 'SUBMITTED'}>6M Bank Stmt</span>
									</div>
								</td>
								<td>
									<span class="status-badge status-{app.status.toLowerCase()}">
										{app.status.replace('_', ' ')}
									</span>
								</td>
								<td>
									<select 
										class="status-select" 
										value={app.status} 
										onchange={(e) => updateVisaStatus(app.id, (e.target as HTMLSelectElement).value)}>
										<option value="SUBMITTED">SUBMITTED</option>
										<option value="UNDER_REVIEW">UNDER REVIEW</option>
										<option value="BIOMETRICS_DONE">BIOMETRICS DONE</option>
										<option value="VISA_APPROVED">VISA APPROVED</option>
										<option value="REJECTED">REJECTED</option>
									</select>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</div>
	{/if}

	<!-- TAB 3: SECRET MARGIN & INDIAN TCS CALCULATOR -->
	{#if activeTab === 'calculator'}
		<div class="section-container">
			<div class="section-header">
				<div>
					<h2 class="section-title">Secret B2B Margin & Section 206C(1G) TCS Engine</h2>
					<p class="section-desc">Strictly keep your vendor net procurement rates confidential while quoting full customer invoices with mandatory Indian TCS.</p>
				</div>
			</div>

			<div class="calc-grid">
				<!-- Left: Net B2B Vendor Costs (Secret) -->
				<div class="calc-card">
					<div class="card-head-secret">
						🔒 Confidential B2B Procurement Costs (Never Disclosed to Client)
					</div>

					<div class="form-row">
						<label for="vendor-flight">Vendor Airfare Net Cost (Flight PNR):</label>
						<div class="input-prefix">
							<span>₹</span>
							<input id="vendor-flight" type="number" bind:value={calcVendorFlight} />
						</div>
					</div>

					<div class="form-row">
						<label for="vendor-hotel">DMC Hotel Contract Rate (Nights × Rooms):</label>
						<div class="input-prefix">
							<span>₹</span>
							<input id="vendor-hotel" type="number" bind:value={calcVendorHotel} />
						</div>
					</div>

					<div class="form-row">
						<label for="vendor-transport">Local Cab / Coach Transfers:</label>
						<div class="input-prefix">
							<span>₹</span>
							<input id="vendor-transport" type="number" bind:value={calcVendorLandTransport} />
						</div>
					</div>

					<div class="form-row">
						<label for="vendor-sightseeing">Sightseeing Entrances & Visa Fee:</label>
						<div class="input-prefix">
							<span>₹</span>
							<input id="vendor-sightseeing" type="number" bind:value={calcVendorSightseeing} />
						</div>
					</div>

					<div class="vendor-total-bar">
						<span>Net Vendor Procurement Cost:</span>
						<span class="mono-num">₹{totalVendorNet.toLocaleString('en-IN')}</span>
					</div>

					<div class="form-row mt-4">
						<label for="agent-markup">Agency Profit Margin Markup (%):</label>
						<input id="agent-markup" type="range" min="5" max="40" bind:value={calcAgentMarkupPercent} />
						<div class="flex justify-between text-xs text-slate-400">
							<span>5% (Competitive B2B)</span>
							<span class="text-sky-400 font-bold">{calcAgentMarkupPercent}% (+₹{agentProfitMargin.toLocaleString('en-IN')})</span>
							<span>40% (Luxury Private)</span>
						</div>
					</div>
				</div>

				<!-- Right: Indian TCS & Client Quotation -->
				<div class="calc-card quotation-card">
					<div class="card-head-quote">
						📄 Formal Client Quotation Breakdown
					</div>

					<div class="quotation-body">
						<div class="quote-line">
							<span>Base Tour Package Selling Price:</span>
							<span class="font-bold text-white">₹{baseSellingPrice.toLocaleString('en-IN')}</span>
						</div>

						<div class="tcs-selector">
							<div class="label-bold">Overseas Tour Program (OTP) TCS Slab:</div>
							<div class="radio-group">
								<label class="radio-label">
									<input type="radio" bind:group={calcTcsTier} value="DOMESTIC" />
									<span>Domestic Tour (0% TCS)</span>
								</label>
								<label class="radio-label">
									<input type="radio" bind:group={calcTcsTier} value="OVERSEAS_5" />
									<span>Overseas Package ≤ ₹7 Lakhs (5% TCS)</span>
								</label>
								<label class="radio-label">
									<input type="radio" bind:group={calcTcsTier} value="OVERSEAS_20" />
									<span>Overseas Package &gt; ₹7 Lakhs (20% TCS)</span>
								</label>
							</div>
						</div>

						{#if tcsRatePercent > 0}
							<div class="tcs-breakdown-box">
								<div class="flex justify-between">
									<span>TCS Collected (Sec 206C(1G) @ {tcsRatePercent}%):</span>
									<span class="text-amber-400 font-bold">+ ₹{tcsAmount.toLocaleString('en-IN')}</span>
								</div>
								<p class="text-xs text-slate-400 mt-1">
									* Client will claim this TCS back as credit on Form 26AS / AIS during Income Tax filing with PAN.
								</p>
							</div>
						{/if}

						<div class="grand-quote-total">
							<div>
								<div class="text-xs uppercase tracking-wider text-slate-300">Total Client Quoted Invoice</div>
								<div class="text-3xl font-extrabold text-emerald-400">
									₹{finalCustomerQuotation.toLocaleString('en-IN')}
								</div>
							</div>
							<button class="btn btn-success" onclick={() => {
								newPkgTitle = `Custom Itinerary (${calcTcsTier})`;
								newPkgPrice = String(finalCustomerQuotation);
								showNewPackageModal = true;
							}}>
								Convert to Package
							</button>
						</div>

						<div class="mt-4 p-3 bg-slate-900/60 rounded border border-slate-700/50 text-xs text-slate-300 space-y-1">
							<div>💰 <strong>Your Net Profit:</strong> ₹{agentProfitMargin.toLocaleString('en-IN')}</div>
							<div>🛡️ <strong>TCS Liability:</strong> ₹{tcsAmount.toLocaleString('en-IN')} (to remit to Govt)</div>
						</div>
					</div>
				</div>
			</div>
		</div>
	{/if}

	<!-- MODAL: ITINERARY PREVIEW & PRINTABLE VOUCHER -->
	{#if showItineraryModal && selectedPackage}
		<div class="modal-backdrop">
			<div class="modal-container printable-voucher">
				<div class="modal-header">
					<div>
						<div class="text-xs text-sky-400 font-bold uppercase">Official Travel Voucher & Itinerary</div>
						<h2 class="text-xl font-bold text-white">{selectedPackage.title}</h2>
						<div class="text-xs text-slate-400">Destination: {selectedPackage.destination} · {selectedPackage.durationDays} Days</div>
					</div>
					<div class="no-print flex gap-2">
						<button class="btn btn-outline" onclick={printItinerary}>🖨️ Print Voucher</button>
						<button class="btn btn-secondary" onclick={() => showItineraryModal = false}>✕ Close</button>
					</div>
				</div>

				<div class="itinerary-timeline">
					{#each parsedItinerary as day}
						<div class="timeline-day">
							<div class="day-badge">Day {day.day}</div>
							<div class="day-content">
								<h4 class="day-title">{day.title}</h4>
								<p class="day-activities">{day.activities}</p>
								<div class="day-meta">
									<span>🏨 Hotel: {day.hotel}</span>
									<span>🍽️ Meal: {day.meals}</span>
								</div>
							</div>
						</div>
					{/each}
				</div>

				<div class="voucher-footer">
					<div class="voucher-notes">
						<div>✓ Valid for bookings through MrBiz-OS Travel Partners.</div>
						<div>✓ Standard cancellation and baggage terms apply.</div>
						<div>✓ 24/7 Emergency Local Concierge: +91 98200 12345</div>
					</div>
					<div class="voucher-price">
						<div class="text-xs text-slate-400">Total Payable</div>
						<div class="text-2xl font-bold text-sky-400">₹{Number(selectedPackage.price).toLocaleString('en-IN')}</div>
					</div>
				</div>
			</div>
		</div>
	{/if}

	<!-- MODAL: NEW TOUR PACKAGE -->
	{#if showNewPackageModal}
		<div class="modal-backdrop">
			<div class="modal-container max-w-xl">
				<div class="modal-header">
					<h3 class="text-lg font-bold text-white">Create New Tour Package</h3>
					<button class="btn btn-secondary" onclick={() => showNewPackageModal = false}>✕</button>
				</div>
				<div class="modal-body space-y-4">
					<div>
						<label class="input-label" for="pkg-title">Package Name / Headline</label>
						<input id="pkg-title" class="text-input" type="text" bind:value={newPkgTitle} placeholder="e.g. Exotic Bali Beach & Kintamani 6D/5N" />
					</div>
					<div class="grid grid-cols-2 gap-3">
						<div>
							<label class="input-label" for="pkg-dest">Destination</label>
							<input id="pkg-dest" class="text-input" type="text" bind:value={newPkgDestination} placeholder="e.g. Bali, Indonesia" />
						</div>
						<div>
							<label class="input-label" for="pkg-dur">Duration (Days)</label>
							<input id="pkg-dur" class="text-input" type="number" bind:value={newPkgDuration} />
						</div>
					</div>
					<div>
						<label class="input-label" for="pkg-price">Client Price (₹ per person)</label>
						<input id="pkg-price" class="text-input" type="number" bind:value={newPkgPrice} />
					</div>
				</div>
				<div class="modal-footer">
					<button class="btn btn-secondary" onclick={() => showNewPackageModal = false}>Cancel</button>
					<button class="btn btn-primary" onclick={handleCreatePackage}>Save & Publish Package</button>
				</div>
			</div>
		</div>
	{/if}

	<!-- MODAL: NEW VISA APPLICATION -->
	{#if showNewVisaModal}
		<div class="modal-backdrop">
			<div class="modal-container max-w-xl">
				<div class="modal-header">
					<h3 class="text-lg font-bold text-white">Submit New Visa Application File</h3>
					<button class="btn btn-secondary" onclick={() => showNewVisaModal = false}>✕</button>
				</div>
				<div class="modal-body space-y-4">
					<div>
						<label class="input-label" for="visa-name">Applicant Full Name (As per Passport)</label>
						<input id="visa-name" class="text-input" type="text" bind:value={newVisaApplicant} placeholder="e.g. Rahul Ramesh" />
					</div>
					<div class="grid grid-cols-2 gap-3">
						<div>
							<label class="input-label" for="visa-pass">Passport Number</label>
							<input id="visa-pass" class="text-input font-mono" type="text" bind:value={newVisaPassport} placeholder="e.g. Z5829104" />
						</div>
						<div>
							<label class="input-label" for="visa-phone">Mobile / WhatsApp</label>
							<input id="visa-phone" class="text-input" type="tel" bind:value={newVisaPhone} placeholder="+91 98765 43210" />
						</div>
					</div>
					<div class="grid grid-cols-2 gap-3">
						<div>
							<label class="input-label" for="visa-country">Target Country</label>
							<select id="visa-country" class="text-input" bind:value={newVisaCountry}>
								<option value="United Arab Emirates">United Arab Emirates (UAE)</option>
								<option value="Schengen Area (France/Germany/Italy)">Schengen Area</option>
								<option value="United Kingdom">United Kingdom (UK)</option>
								<option value="Singapore">Singapore</option>
								<option value="Thailand">Thailand</option>
								<option value="United States of America">USA B1/B2</option>
							</select>
						</div>
						<div>
							<label class="input-label" for="visa-type">Visa Category</label>
							<select id="visa-type" class="text-input" bind:value={newVisaType}>
								<option value="Tourist 30 Days Single Entry">Tourist 30 Days Single Entry</option>
								<option value="Tourist 60 Days Multiple Entry">Tourist 60 Days Multiple Entry</option>
								<option value="Business Visitor (C Type)">Business Visitor</option>
								<option value="Transit Visa 96 Hours">Transit Visa 96 Hours</option>
							</select>
						</div>
					</div>

					<div class="p-3 bg-slate-900/60 rounded border border-slate-700/60 text-xs space-y-2">
						<div class="font-bold text-sky-400">Embassies Mandatory Document Verification</div>
						<label class="flex items-center gap-2">
							<input type="checkbox" bind:checked={newVisaChecklist.passportSixMonthsValid} />
							<span>Original Passport with &gt; 6 months validity & 2 blank pages</span>
						</label>
						<label class="flex items-center gap-2">
							<input type="checkbox" bind:checked={newVisaChecklist.photographWhiteBg} />
							<span>Recent 35×45mm colour photograph (80% face coverage)</span>
						</label>
						<label class="flex items-center gap-2">
							<input type="checkbox" bind:checked={newVisaChecklist.bankStatementSixMonths} />
							<span>6-month bank statement with branch seal & minimum balance</span>
						</label>
						<label class="flex items-center gap-2">
							<input type="checkbox" bind:checked={newVisaChecklist.travelInsurance} />
							<span>Overseas travel medical insurance (Min $50,000 cover)</span>
						</label>
					</div>
				</div>
				<div class="modal-footer">
					<button class="btn btn-secondary" onclick={() => showNewVisaModal = false}>Cancel</button>
					<button class="btn btn-primary" onclick={handleSubmitVisa}>Submit File to Pipeline</button>
				</div>
			</div>
		</div>
	{/if}
</div>

<style>
	.travel-app {
		padding: 1.5rem;
		max-width: 1440px;
		margin: 0 auto;
		color: #e2e8f0;
	}

	.app-header {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		flex-wrap: wrap;
		gap: 1rem;
		margin-bottom: 1.5rem;
	}

	.badge-row {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		margin-bottom: 0.5rem;
	}

	.pill-badge {
		font-size: 0.75rem;
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		background: rgba(14, 165, 233, 0.15);
		color: #38bdf8;
		border: 1px solid rgba(14, 165, 233, 0.3);
		padding: 0.2rem 0.6rem;
		border-radius: 9999px;
	}

	.status-indicator {
		font-size: 0.75rem;
		color: #10b981;
	}

	.main-title {
		font-size: 1.875rem;
		font-weight: 800;
		color: #ffffff;
		letter-spacing: -0.02em;
	}

	.subtitle {
		font-size: 0.875rem;
		color: #94a3b8;
		margin-top: 0.25rem;
		max-width: 800px;
	}

	.header-actions {
		display: flex;
		gap: 0.75rem;
	}

	.btn {
		font-size: 0.875rem;
		font-weight: 600;
		padding: 0.625rem 1.25rem;
		border-radius: 8px;
		cursor: pointer;
		display: inline-flex;
		align-items: center;
		gap: 0.5rem;
		transition: all 0.15s ease;
		border: none;
	}

	.btn-primary {
		background: #0284c7;
		color: #ffffff;
	}
	.btn-primary:hover {
		background: #0369a1;
	}

	.btn-secondary {
		background: #334155;
		color: #f1f5f9;
	}
	.btn-secondary:hover {
		background: #475569;
	}

	.btn-outline {
		background: transparent;
		border: 1px solid rgba(255, 255, 255, 0.2);
		color: #cbd5e1;
	}
	.btn-outline:hover {
		background: rgba(255, 255, 255, 0.05);
	}

	.btn-success {
		background: #10b981;
		color: #ffffff;
	}
	.btn-success:hover {
		background: #059669;
	}

	.status-alert {
		background: rgba(16, 185, 129, 0.15);
		border: 1px solid #10b981;
		color: #6ee7b7;
		padding: 0.75rem 1rem;
		border-radius: 8px;
		margin-bottom: 1.25rem;
	}

	.error-alert {
		background: rgba(239, 68, 68, 0.15);
		border: 1px solid #ef4444;
		color: #fca5a5;
		padding: 0.75rem 1rem;
		border-radius: 8px;
		margin-bottom: 1.25rem;
	}

	/* KPI GRID */
	.kpi-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
		gap: 1rem;
		margin-bottom: 1.75rem;
	}

	.kpi-card {
		background: #1e293b;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 12px;
		padding: 1.25rem;
	}

	.kpi-label {
		font-size: 0.75rem;
		font-weight: 600;
		color: #94a3b8;
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}

	.kpi-value {
		font-size: 1.75rem;
		font-weight: 800;
		color: #ffffff;
		margin: 0.25rem 0;
	}

	.kpi-sub {
		font-size: 0.75rem;
		color: #64748b;
	}

	/* NAVIGATION TABS */
	.nav-tabs {
		display: flex;
		gap: 0.5rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.1);
		margin-bottom: 1.5rem;
	}

	.tab-btn {
		background: transparent;
		border: none;
		color: #94a3b8;
		font-size: 0.875rem;
		font-weight: 600;
		padding: 0.75rem 1.25rem;
		border-bottom: 2px solid transparent;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.tab-btn.active {
		color: #38bdf8;
		border-bottom-color: #38bdf8;
	}

	.tab-btn:hover:not(.active) {
		color: #cbd5e1;
	}

	/* SECTION CONTAINER */
	.section-container {
		background: #1e293b;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 14px;
		padding: 1.5rem;
	}

	.section-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 1.5rem;
	}

	.section-title {
		font-size: 1.25rem;
		font-weight: 700;
		color: #ffffff;
	}

	.section-desc {
		font-size: 0.8125rem;
		color: #94a3b8;
	}

	/* PACKAGES GRID */
	.packages-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(340px, 1fr));
		gap: 1.25rem;
	}

	.pkg-card {
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 12px;
		padding: 1.25rem;
		display: flex;
		flex-direction: column;
		justify-content: space-between;
		transition: transform 0.15s, border-color 0.15s;
	}

	.pkg-card:hover {
		transform: translateY(-2px);
		border-color: rgba(56, 189, 248, 0.4);
	}

	.pkg-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 0.75rem;
	}

	.destination-pill {
		font-size: 0.75rem;
		font-weight: 600;
		color: #38bdf8;
		background: rgba(56, 189, 248, 0.1);
		padding: 0.2rem 0.5rem;
		border-radius: 6px;
	}

	.duration-badge {
		font-size: 0.75rem;
		color: #94a3b8;
	}

	.pkg-title {
		font-size: 1.125rem;
		font-weight: 700;
		color: #ffffff;
		margin-bottom: 0.75rem;
	}

	.pkg-highlights {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
		font-size: 0.75rem;
		color: #cbd5e1;
		margin-bottom: 1.25rem;
	}

	.pkg-highlights span {
		background: rgba(255, 255, 255, 0.04);
		padding: 0.2rem 0.45rem;
		border-radius: 4px;
	}

	.pkg-footer {
		display: flex;
		justify-content: space-between;
		align-items: flex-end;
		border-top: 1px solid rgba(255, 255, 255, 0.06);
		padding-top: 0.75rem;
	}

	.price-label {
		font-size: 0.6875rem;
		color: #94a3b8;
		text-transform: uppercase;
	}

	.price-val {
		font-size: 1.25rem;
		font-weight: 800;
		color: #38bdf8;
	}

	.per-person {
		font-size: 0.6875rem;
		font-weight: 400;
		color: #94a3b8;
	}

	.btn-view {
		background: rgba(56, 189, 248, 0.15);
		color: #38bdf8;
		border: 1px solid rgba(56, 189, 248, 0.3);
		padding: 0.4rem 0.75rem;
		border-radius: 6px;
		font-size: 0.75rem;
		font-weight: 600;
		cursor: pointer;
	}

	.btn-view:hover {
		background: rgba(56, 189, 248, 0.25);
	}

	/* DATA TABLE */
	.table-card {
		overflow-x: auto;
	}

	.data-table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.8125rem;
		text-align: left;
	}

	.data-table th {
		background: rgba(0, 0, 0, 0.25);
		color: #94a3b8;
		font-weight: 600;
		padding: 0.75rem 1rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
	}

	.data-table td {
		padding: 0.875rem 1rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.04);
		color: #cbd5e1;
	}

	.mono-badge {
		font-family: monospace;
		background: #0f172a;
		padding: 0.2rem 0.4rem;
		border-radius: 4px;
		border: 1px solid rgba(255, 255, 255, 0.1);
		color: #f8fafc;
	}

	.country-cell {
		font-weight: 600;
		color: #f1f5f9;
	}

	.checklist-pills {
		display: flex;
		gap: 0.35rem;
		flex-wrap: wrap;
	}

	.check-pill {
		font-size: 0.6875rem;
		padding: 0.15rem 0.4rem;
		border-radius: 4px;
		background: rgba(255, 255, 255, 0.05);
		color: #94a3b8;
	}

	.check-pill.done {
		background: rgba(16, 185, 129, 0.15);
		color: #34d399;
	}

	.status-badge {
		font-size: 0.6875rem;
		font-weight: 700;
		padding: 0.25rem 0.6rem;
		border-radius: 6px;
		text-transform: uppercase;
	}

	.status-submitted { background: rgba(59, 130, 246, 0.15); color: #60a5fa; }
	.status-under_review { background: rgba(245, 158, 11, 0.15); color: #fbbf24; }
	.status-biometrics_done { background: rgba(168, 85, 247, 0.15); color: #c084fc; }
	.status-visa_approved { background: rgba(16, 185, 129, 0.15); color: #34d399; }
	.status-rejected { background: rgba(239, 68, 68, 0.15); color: #f87171; }

	.status-select {
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.15);
		color: #f1f5f9;
		padding: 0.3rem 0.5rem;
		border-radius: 6px;
		font-size: 0.75rem;
	}

	/* CALCULATOR GRID */
	.calc-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(420px, 1fr));
		gap: 1.5rem;
	}

	.calc-card {
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 12px;
		padding: 1.5rem;
	}

	.card-head-secret {
		font-size: 0.875rem;
		font-weight: 700;
		color: #f43f5e;
		background: rgba(244, 63, 94, 0.1);
		border: 1px dashed rgba(244, 63, 94, 0.3);
		padding: 0.6rem 0.8rem;
		border-radius: 8px;
		margin-bottom: 1.25rem;
	}

	.card-head-quote {
		font-size: 0.875rem;
		font-weight: 700;
		color: #38bdf8;
		background: rgba(56, 189, 248, 0.1);
		border: 1px solid rgba(56, 189, 248, 0.3);
		padding: 0.6rem 0.8rem;
		border-radius: 8px;
		margin-bottom: 1.25rem;
	}

	.form-row {
		margin-bottom: 1rem;
	}

	.form-row label {
		display: block;
		font-size: 0.8125rem;
		color: #94a3b8;
		margin-bottom: 0.35rem;
	}

	.input-prefix {
		display: flex;
		align-items: center;
		background: #1e293b;
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: 6px;
		padding: 0 0.75rem;
	}

	.input-prefix span {
		color: #94a3b8;
		font-weight: 600;
	}

	.input-prefix input {
		width: 100%;
		background: transparent;
		border: none;
		color: #ffffff;
		padding: 0.5rem;
		font-family: monospace;
		font-size: 1rem;
		outline: none;
	}

	.vendor-total-bar {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 0.75rem 1rem;
		background: rgba(255, 255, 255, 0.03);
		border-radius: 8px;
		border: 1px solid rgba(255, 255, 255, 0.08);
		font-size: 0.875rem;
		color: #cbd5e1;
	}

	.mono-num {
		font-family: monospace;
		font-size: 1.125rem;
		font-weight: 700;
		color: #ffffff;
	}

	.quotation-card {
		border-color: rgba(56, 189, 248, 0.3);
	}

	.quotation-body {
		display: flex;
		flex-direction: column;
		gap: 1.25rem;
	}

	.quote-line {
		display: flex;
		justify-content: space-between;
		font-size: 1rem;
		padding-bottom: 0.75rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
	}

	.tcs-selector {
		background: rgba(255, 255, 255, 0.02);
		padding: 1rem;
		border-radius: 8px;
		border: 1px solid rgba(255, 255, 255, 0.06);
	}

	.label-bold {
		font-size: 0.8125rem;
		font-weight: 700;
		color: #f1f5f9;
		margin-bottom: 0.6rem;
	}

	.radio-group {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.radio-label {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		font-size: 0.8125rem;
		color: #cbd5e1;
		cursor: pointer;
	}

	.tcs-breakdown-box {
		background: rgba(245, 158, 11, 0.1);
		border: 1px solid rgba(245, 158, 11, 0.3);
		border-radius: 8px;
		padding: 0.75rem;
	}

	.grand-quote-total {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 1rem;
		background: rgba(16, 185, 129, 0.1);
		border: 1px solid rgba(16, 185, 129, 0.3);
		border-radius: 10px;
	}

	/* MODAL STYLES */
	.modal-backdrop {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.8);
		backdrop-filter: blur(4px);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 1000;
		padding: 1rem;
	}

	.modal-container {
		background: #1e293b;
		border: 1px solid rgba(255, 255, 255, 0.15);
		border-radius: 14px;
		padding: 1.75rem;
		width: 100%;
		max-height: 90vh;
		overflow-y: auto;
	}

	.modal-header {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
		padding-bottom: 1rem;
		margin-bottom: 1.25rem;
	}

	.modal-footer {
		display: flex;
		justify-content: flex-end;
		gap: 0.75rem;
		border-top: 1px solid rgba(255, 255, 255, 0.08);
		padding-top: 1rem;
		margin-top: 1.5rem;
	}

	.input-label {
		display: block;
		font-size: 0.75rem;
		font-weight: 600;
		color: #94a3b8;
		margin-bottom: 0.35rem;
	}

	.text-input {
		width: 100%;
		padding: 0.6rem 0.75rem;
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: 6px;
		color: #ffffff;
		font-size: 0.875rem;
	}

	/* ITINERARY TIMELINE */
	.itinerary-timeline {
		display: flex;
		flex-direction: column;
		gap: 1rem;
		margin-bottom: 1.5rem;
	}

	.timeline-day {
		display: flex;
		gap: 1rem;
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.06);
		border-radius: 10px;
		padding: 1rem;
	}

	.day-badge {
		background: #0284c7;
		color: #ffffff;
		font-weight: 800;
		font-size: 0.75rem;
		padding: 0.35rem 0.6rem;
		border-radius: 6px;
		height: fit-content;
		white-space: nowrap;
	}

	.day-content {
		flex: 1;
	}

	.day-title {
		font-size: 1rem;
		font-weight: 700;
		color: #ffffff;
		margin-bottom: 0.25rem;
	}

	.day-activities {
		font-size: 0.8125rem;
		color: #94a3b8;
		margin-bottom: 0.5rem;
	}

	.day-meta {
		display: flex;
		gap: 1rem;
		font-size: 0.75rem;
		color: #38bdf8;
	}

	.voucher-footer {
		display: flex;
		justify-content: space-between;
		align-items: center;
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: 10px;
		padding: 1rem;
	}

	.voucher-notes {
		font-size: 0.75rem;
		color: #94a3b8;
		line-height: 1.4;
	}

	/* PRINT STYLES */
	@media print {
		.no-print, .nav-tabs, .kpi-grid, .app-header {
			display: none !important;
		}
		body {
			background: white !important;
			color: black !important;
		}
		.printable-voucher {
			background: white !important;
			color: black !important;
			border: 1px solid #ccc !important;
			box-shadow: none !important;
		}
		.timeline-day, .voucher-footer {
			background: white !important;
			border: 1px solid #eee !important;
			color: black !important;
		}
		.day-title, .day-activities {
			color: black !important;
		}
	}
</style>
