<script lang="ts">
	import { onMount } from 'svelte';
	import { travelGrpc } from '$lib/grpc';
	import SharedWorkModal from '$lib/components/SharedWorkModal.svelte';

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

	interface Passenger {
		name: string;
		passportNo: string;
		passportExpiry: string;
		dob: string;
		type: 'ADULT' | 'CHILD' | 'INFANT';
		mealPreference: 'VEG' | 'JAIN' | 'NON_VEG';
		wheelchair: boolean;
	}

	interface ActiveBooking {
		id: string;
		customerName: string;
		customerPhone: string;
		customerEmail: string;
		destination: string;
		packageTitle: string;
		travelDates: string;
		paxCount: number;
		airlinePnr: string;
		pnrDeadline: string;
		hotelDmcRef: string;
		hotelName: string;
		visaStatus: 'NOT_REQUIRED' | 'SUBMITTED' | 'BIOMETRICS_DONE' | 'APPROVED' | 'STAMPED';
		totalAmount: number;
		advancePaid: number;
		balanceDue: number;
		invoiceNumber: string;
		status: 'CONFIRMED' | 'OPERATIONS_HOLD' | 'IN_TRANSIT' | 'COMPLETED';
		passengers: Passenger[];
		readiness: {
			hotelReconfirmed: boolean;
			insuranceIssued: boolean;
			webCheckInDone: boolean;
			travelKitSent: boolean;
		};
	}

	let activeTab = $state<'operations' | 'packages' | 'visas' | 'manifest' | 'calculator'>('operations');
	let packages = $state<TourPackage[]>([]);
	let applications = $state<VisaApplication[]>([]);
	let loading = $state(true);
	let error = $state('');
	let statusMsg = $state('');

	// Active Bookings in Agency Operations Pipeline
	let activeBookings = $state<ActiveBooking[]>([
		{
			id: 'TRIP-2026-904',
			customerName: 'Rahul Sharma & Family',
			customerPhone: '+91 98401 23456',
			customerEmail: 'rahul.sharma@techcorp.com',
			destination: 'Dubai & Abu Dhabi, UAE',
			packageTitle: 'Exotic Dubai Sky & Desert Safari 5D/4N',
			travelDates: '14 Nov 2026 - 19 Nov 2026',
			paxCount: 4,
			airlinePnr: 'EK-521 (Emirates)',
			pnrDeadline: 'Ticketing Done ✓',
			hotelDmcRef: 'DMC-DXB-98421',
			hotelName: 'JW Marriott Marquis Dubai (2 Deluxe Rooms)',
			visaStatus: 'STAMPED',
			totalAmount: 185000,
			advancePaid: 95000,
			balanceDue: 90000,
			invoiceNumber: 'INV-TRA-202604',
			status: 'CONFIRMED',
			passengers: [
				{ name: 'Rahul Sharma', passportNo: 'Z5829104', passportExpiry: '2031-08-14', dob: '1984-04-12', type: 'ADULT', mealPreference: 'VEG', wheelchair: false },
				{ name: 'Pooja Sharma', passportNo: 'Z5829105', passportExpiry: '2031-08-14', dob: '1988-11-23', type: 'ADULT', mealPreference: 'JAIN', wheelchair: false },
				{ name: 'Aarav Sharma', passportNo: 'Z9102834', passportExpiry: '2029-01-10', dob: '2016-06-05', type: 'CHILD', mealPreference: 'NON_VEG', wheelchair: false },
				{ name: 'Ananya Sharma', passportNo: 'Z9102835', passportExpiry: '2029-01-10', dob: '2019-09-18', type: 'CHILD', mealPreference: 'VEG', wheelchair: false }
			],
			readiness: {
				hotelReconfirmed: true,
				insuranceIssued: true,
				webCheckInDone: false, // 24h prior
				travelKitSent: true
			}
		},
		{
			id: 'TRIP-2026-908',
			customerName: 'Dr. V. Rajesh & Deepa',
			customerPhone: '+91 98410 88990',
			customerEmail: 'dr.rajesh@apollo.org',
			destination: 'Bali & Nusa Penida, Indonesia',
			packageTitle: 'Tropical Bali Luxury Villas 6D/5N',
			travelDates: '22 Nov 2026 - 28 Nov 2026',
			paxCount: 2,
			airlinePnr: 'SQ-529 (Singapore Airlines)',
			pnrDeadline: 'Hold Expires in 18 Hours ⚠️',
			hotelDmcRef: 'DMC-DPS-1092',
			hotelName: 'The Kayon Jungle Resort Ubud',
			visaStatus: 'NOT_REQUIRED',
			totalAmount: 142000,
			advancePaid: 50000,
			balanceDue: 92000,
			invoiceNumber: 'INV-TRA-202608',
			status: 'OPERATIONS_HOLD',
			passengers: [
				{ name: 'Dr. V. Rajesh', passportNo: 'K4920194', passportExpiry: '2030-05-19', dob: '1979-02-14', type: 'ADULT', mealPreference: 'VEG', wheelchair: false },
				{ name: 'Deepa Rajesh', passportNo: 'K4920195', passportExpiry: '2030-05-19', dob: '1982-08-30', type: 'ADULT', mealPreference: 'VEG', wheelchair: false }
			],
			readiness: {
				hotelReconfirmed: true,
				insuranceIssued: false,
				webCheckInDone: false,
				travelKitSent: false
			}
		},
		{
			id: 'TRIP-2026-912',
			customerName: 'K. Senthil Kumar & Friends',
			customerPhone: '+91 94440 12345',
			customerEmail: 'senthil@builders.in',
			destination: 'Singapore & Sentosa Island',
			packageTitle: 'Singapore Highlights & Universal Studios 4D/3N',
			travelDates: '05 Dec 2026 - 09 Dec 2026',
			paxCount: 6,
			airlinePnr: '6E-1004 (IndiGo)',
			pnrDeadline: 'Ticketing Done ✓',
			hotelDmcRef: 'DMC-SIN-4091',
			hotelName: 'Hotel Boss Singapore (3 Twin Rooms)',
			visaStatus: 'BIOMETRICS_DONE',
			totalAmount: 265000,
			advancePaid: 150000,
			balanceDue: 115000,
			invoiceNumber: 'INV-TRA-202612',
			status: 'CONFIRMED',
			passengers: [
				{ name: 'K. Senthil Kumar', passportNo: 'M1029384', passportExpiry: '2032-11-01', dob: '1975-03-20', type: 'ADULT', mealPreference: 'NON_VEG', wheelchair: false },
				{ name: 'N. Murugan', passportNo: 'M1029385', passportExpiry: '2032-11-01', dob: '1976-07-15', type: 'ADULT', mealPreference: 'NON_VEG', wheelchair: false }
			],
			readiness: {
				hotelReconfirmed: false,
				insuranceIssued: true,
				webCheckInDone: false,
				travelKitSent: false
			}
		}
	]);

	// Shared Work Modal State
	let showSharedWorkModal = $state(false);
	let workBooking = $state<ActiveBooking | null>(null);

	// Travel Kit Modal State
	let showTravelKitModal = $state(false);
	let kitBooking = $state<ActiveBooking | null>(null);

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
				selectedPackage = packages[0];
				if (packages[0].itineraryJson) {
					try {
						parsedItinerary = JSON.parse(packages[0].itineraryJson);
					} catch {
						// ignore parse error
					}
				}
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

	function openSharedWorkHub(booking: ActiveBooking) {
		workBooking = booking;
		showSharedWorkModal = true;
	}

	function openTravelKit(booking: ActiveBooking) {
		kitBooking = booking;
		showTravelKitModal = true;
	}

	function toggleReadiness(bookingId: string, key: keyof ActiveBooking['readiness']) {
		activeBookings = activeBookings.map(b => {
			if (b.id === bookingId) {
				const updated = { ...b.readiness, [key]: !b.readiness[key] };
				return { ...b, readiness: updated };
			}
			return b;
		});
		statusMsg = `Pre-departure readiness updated.`;
		setTimeout(() => statusMsg = '', 2000);
	}

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
	<title>Travel, Tours &amp; Visa Management OS | MrBiz-OS</title>
</svelte:head>

<div class="travel-app">
	<!-- Top Bar -->
	<header class="app-header">
		<div>
			<div class="badge-row">
				<span class="pill-badge">✈️ Global Travel &amp; Visa OS</span>
				<span class="status-indicator">● ConnectRPC v1 Connected</span>
			</div>
			<h1 class="main-title">Tour Operator &amp; Immigration Operations Hub</h1>
			<p class="subtitle">Multi-role shared works deck, airline PNR holds, DMC hotel voucher registry, pre-departure readiness, and Section 206C(1G) Indian TCS engine.</p>
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
		<div class="kpi-card kpi-card-emerald">
			<div class="kpi-label">Active Trip Bookings</div>
			<div class="kpi-value kpi-val-emerald">{activeBookings.length} Bookings</div>
			<div class="kpi-sub">Total ₹{(activeBookings.reduce((sum, b) => sum + b.totalAmount, 0)).toLocaleString('en-IN')} Turn</div>
		</div>
		<div class="kpi-card kpi-card-amber">
			<div class="kpi-label">Accounts Balance Due</div>
			<div class="kpi-value kpi-val-amber">₹{(activeBookings.reduce((sum, b) => sum + b.balanceDue, 0)).toLocaleString('en-IN')}</div>
			<div class="kpi-sub">GST &amp; TCS Invoices Synced</div>
		</div>
		<div class="kpi-card kpi-card-sky">
			<div class="kpi-label">PNR / Voucher Holds</div>
			<div class="kpi-value kpi-val-sky">1 Expiring &lt;24h</div>
			<div class="kpi-sub">GDS Ticketing Deadlines</div>
		</div>
		<div class="kpi-card kpi-card-purple">
			<div class="kpi-label">Visa Cases Pipeline</div>
			<div class="kpi-value kpi-val-purple">{applications.length} Applicants</div>
			<div class="kpi-sub">VFS &amp; Consulate Biometrics</div>
		</div>
	</div>

	<!-- Navigation Tabs -->
	<div class="nav-tabs">
		<button 
			class="tab-btn" 
			class:active={activeTab === 'operations'} 
			onclick={() => activeTab = 'operations'}>
			⚡ Live Operations Deck &amp; Active Bookings ({activeBookings.length})
		</button>
		<button 
			class="tab-btn" 
			class:active={activeTab === 'packages'} 
			onclick={() => activeTab = 'packages'}>
			✈️ Holiday Packages &amp; Itineraries ({packages.length})
		</button>
		<button 
			class="tab-btn" 
			class:active={activeTab === 'visas'} 
			onclick={() => activeTab = 'visas'}>
			🛂 Visa &amp; Immigration Pipeline ({applications.length})
		</button>
		<button 
			class="tab-btn" 
			class:active={activeTab === 'manifest'} 
			onclick={() => activeTab = 'manifest'}>
			👥 Passenger Manifest &amp; Rooming Lists
		</button>
		<button 
			class="tab-btn" 
			class:active={activeTab === 'calculator'} 
			onclick={() => activeTab = 'calculator'}>
			🧮 B2B Secret Margin &amp; TCS 206C Calculator
		</button>
	</div>

	<!-- TAB 0: LIVE OPERATIONS DECK & ACTIVE BOOKINGS (NEW HEARTBEAT) -->
	{#if activeTab === 'operations'}
		<div class="section-container">
			<div class="section-header">
				<div>
					<h2 class="section-title">Active Travel Bookings &amp; Shared Team Cockpit</h2>
					<p class="section-desc">Unified cross-team visibility connecting Sales, Ticketing, DMC Operations, Visa, and Accounts.</p>
				</div>
			</div>

			<div class="bookings-grid">
				{#each activeBookings as b}
					<div class="booking-card">
						<!-- Top Ribbon -->
						<div class="booking-ribbon">
							<div class="flex items-center gap-2">
								<span class="booking-ref">{b.id}</span>
								<span class="status-pill status-{b.status.toLowerCase()}">{b.status}</span>
							</div>
							<div class="booking-dates">📅 {b.travelDates}</div>
						</div>

						<!-- Core Info -->
						<div class="booking-main">
							<div class="customer-block">
								<h3 class="cust-name">{b.customerName}</h3>
								<div class="cust-meta">
									<span>📱 {b.customerPhone}</span>
									<span>👥 {b.paxCount} Travelers</span>
								</div>
								<div class="trip-title">🌴 {b.packageTitle}</div>
							</div>

							<div class="finance-block">
								<div class="finance-total">₹{b.totalAmount.toLocaleString('en-IN')}</div>
								<div class="finance-meta">
									<span class="text-emerald-400">Paid: ₹{b.advancePaid.toLocaleString('en-IN')}</span> · 
									<span class="text-rose-400 font-bold">Due: ₹{b.balanceDue.toLocaleString('en-IN')}</span>
								</div>
								<div class="invoice-ref">Invoice: #{b.invoiceNumber}</div>
							</div>
						</div>

						<!-- Supplier PNR & Voucher Tracking -->
						<div class="suppliers-grid">
							<div class="supplier-cell">
								<span class="sup-label">✈️ Airline PNR:</span>
								<strong class="font-mono text-white">{b.airlinePnr}</strong>
								<span class="sup-status" class:warn={b.pnrDeadline.includes('Expires')}>
									{b.pnrDeadline}
								</span>
							</div>

							<div class="supplier-cell">
								<span class="sup-label">🏨 DMC Hotel Voucher:</span>
								<strong class="font-mono text-sky-400">{b.hotelDmcRef}</strong>
								<span class="sup-sub">{b.hotelName}</span>
							</div>

							<div class="supplier-cell">
								<span class="sup-label">🛂 Visa Status:</span>
								<strong class="font-semibold text-purple-400">{b.visaStatus}</strong>
								<span class="sup-sub">Biometrics &amp; Passport Clear</span>
							</div>
						</div>

						<!-- 48h/24h Pre-Departure Readiness Checklist -->
						<div class="readiness-bar">
							<div class="readiness-label">Pre-Departure Readiness Checklist:</div>
							<div class="readiness-items">
								<button 
									type="button"
									class="check-pill-interactive" 
									class:done={b.readiness.hotelReconfirmed}
									onclick={() => toggleReadiness(b.id, 'hotelReconfirmed')}>
									{b.readiness.hotelReconfirmed ? '✓' : '○'} Hotel Reconfirmed (48h)
								</button>

								<button 
									type="button"
									class="check-pill-interactive" 
									class:done={b.readiness.insuranceIssued}
									onclick={() => toggleReadiness(b.id, 'insuranceIssued')}>
									{b.readiness.insuranceIssued ? '✓' : '○'} Travel Insurance Issued
								</button>

								<button 
									type="button"
									class="check-pill-interactive" 
									class:done={b.readiness.webCheckInDone}
									onclick={() => toggleReadiness(b.id, 'webCheckInDone')}>
									{b.readiness.webCheckInDone ? '✓' : '○'} Web Check-in (24h)
								</button>

								<button 
									type="button"
									class="check-pill-interactive" 
									class:done={b.readiness.travelKitSent}
									onclick={() => toggleReadiness(b.id, 'travelKitSent')}>
									{b.readiness.travelKitSent ? '✓' : '○'} WhatsApp Kit Sent
								</button>
							</div>
						</div>

						<!-- Action Footer -->
						<div class="booking-footer">
							<div class="btn-group-footer">
								<button class="btn btn-primary btn-sm" onclick={() => openSharedWorkHub(b)}>
									⚡ Shared Works Hub (Billing &amp; Tasks)
								</button>
								<button class="btn btn-outline btn-sm" onclick={() => openTravelKit(b)}>
									📜 Travel Kit &amp; Vouchers PDF
								</button>
								<a class="btn btn-whatsapp btn-sm" href="https://wa.me/{b.customerPhone.replace(/[^0-9]/g, '')}?text=Dear%20{encodeURIComponent(b.customerName)},%20update%20regarding%20your%20booking%20{b.id}" target="_blank">
									💬 WhatsApp
								</a>
							</div>
						</div>
					</div>
				{/each}
			</div>
		</div>
	{/if}

	<!-- TAB 1: TOUR PACKAGES -->
	{#if activeTab === 'packages'}
		<div class="section-container">
			<div class="section-header">
				<div>
					<h2 class="section-title">Featured Itineraries &amp; Tour Packages</h2>
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
									<button class="btn btn-primary btn-sm ml-2" onclick={() => {
										workBooking = {
											id: `TRIP-${Date.now().toString().slice(-4)}`,
											customerName: 'Direct Walk-in / Inquirer',
											customerPhone: '+91 98401 23456',
											customerEmail: '',
											destination: pkg.destination,
											packageTitle: pkg.title,
											travelDates: 'Upcoming Month',
											paxCount: 2,
											airlinePnr: 'PENDING_HOLD',
											pnrDeadline: 'Hold 24h',
											hotelDmcRef: 'PENDING_VOUCHER',
											hotelName: 'Standard 4★ Hotel',
											visaStatus: 'SUBMITTED',
											totalAmount: Number(pkg.price) * 2,
											advancePaid: Number(pkg.price),
											balanceDue: Number(pkg.price),
											invoiceNumber: `INV-${Date.now().toString().slice(-4)}`,
											status: 'OPERATIONS_HOLD',
											passengers: [],
											readiness: { hotelReconfirmed: false, insuranceIssued: false, webCheckInDone: false, travelKitSent: false }
										};
										showSharedWorkModal = true;
									}}>
										⚡ Convert to Booking
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
					<h2 class="section-title">Visa &amp; Immigration Case Pipeline</h2>
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

	<!-- TAB 3: PASSENGER MANIFEST & ROOMING LISTS -->
	{#if activeTab === 'manifest'}
		<div class="section-container">
			<div class="section-header">
				<div>
					<h2 class="section-title">Passenger Manifest &amp; Rooming Allocation Lists</h2>
					<p class="section-desc">Critical passport bio details, birth dates, dietary requirements, and wheelchair assistance for airlines &amp; DMCs.</p>
				</div>
			</div>

			<div class="space-y-6">
				{#each activeBookings as b}
					<div class="manifest-card">
						<div class="manifest-header">
							<div>
								<div class="text-xs text-sky-400 font-bold uppercase">{b.id} · {b.packageTitle}</div>
								<h3 class="text-lg font-bold text-white">Lead Pax: {b.customerName} ({b.paxCount} Travelers)</h3>
							</div>
							<div class="rooming-pill">
								🏨 Hotel: {b.hotelName}
							</div>
						</div>

						<table class="data-table mt-3">
							<thead>
								<tr>
									<th>Traveler Name</th>
									<th>Passenger Type</th>
									<th>Passport No</th>
									<th>Expiry Date</th>
									<th>Date of Birth</th>
									<th>Meal Preference</th>
									<th>Special Requests</th>
								</tr>
							</thead>
							<tbody>
								{#each b.passengers as p}
									<tr>
										<td class="font-bold text-white">{p.name}</td>
										<td>
											<span class="badge status-submitted">{p.type}</span>
										</td>
										<td><span class="mono-badge">{p.passportNo}</span></td>
										<td>{p.passportExpiry}</td>
										<td>{p.dob}</td>
										<td>
											<span class="meal-tag meal-{p.mealPreference.toLowerCase()}">{p.mealPreference}</span>
										</td>
										<td>
											{#if p.wheelchair}
												<span class="text-amber-400 text-xs font-bold">♿ Wheelchair Requested</span>
											{:else}
												<span class="text-slate-500 text-xs">Standard</span>
											{/if}
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				{/each}
			</div>
		</div>
	{/if}

	<!-- TAB 4: SECRET MARGIN & INDIAN TCS CALCULATOR -->
	{#if activeTab === 'calculator'}
		<div class="section-container">
			<div class="section-header">
				<div>
					<h2 class="section-title">Secret B2B Margin &amp; Section 206C(1G) TCS Engine</h2>
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
						<label for="vendor-sightseeing">Sightseeing Entrances &amp; Visa Fee:</label>
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
						<div class="text-xs text-sky-400 font-bold uppercase">Official Travel Voucher &amp; Itinerary</div>
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

	<!-- MODAL: TRAVEL KIT & VOUCHER VIEWER -->
	{#if showTravelKitModal && kitBooking}
		<div class="modal-backdrop">
			<div class="modal-container max-w-2xl">
				<div class="modal-header">
					<div>
						<div class="text-xs text-emerald-400 font-bold uppercase">Final Pre-Departure Travel Kit</div>
						<h3 class="text-xl font-bold text-white">{kitBooking.packageTitle}</h3>
						<div class="text-xs text-slate-400">Booking Ref: {kitBooking.id} · Lead: {kitBooking.customerName}</div>
					</div>
					<div class="flex gap-2">
						<button class="btn btn-outline btn-sm" onclick={() => window.print()}>🖨️ Print Kit</button>
						<button class="btn btn-secondary btn-sm" onclick={() => showTravelKitModal = false}>✕</button>
					</div>
				</div>

				<div class="modal-body space-y-4">
					<!-- Flight Ticket Card -->
					<div class="p-4 bg-slate-900/80 rounded-lg border border-slate-700/80">
						<div class="flex justify-between items-center mb-2">
							<div class="font-bold text-white">✈️ E-Ticket &amp; Flight Confirmation</div>
							<span class="font-mono text-emerald-400 font-bold">{kitBooking.airlinePnr}</span>
						</div>
						<div class="text-xs text-slate-300">Routing: Chennai (MAA) ➔ Dubai (DXB) · Emirates EK-521</div>
						<div class="text-2xs text-slate-400 mt-1">Baggage: 30kg Check-in + 7kg Cabin. Terminal 3.</div>
					</div>

					<!-- Hotel Voucher Card -->
					<div class="p-4 bg-slate-900/80 rounded-lg border border-slate-700/80">
						<div class="flex justify-between items-center mb-2">
							<div class="font-bold text-white">🏨 DMC Hotel Check-in Voucher</div>
							<span class="font-mono text-sky-400 font-bold">{kitBooking.hotelDmcRef}</span>
						</div>
						<div class="text-xs text-slate-300">{kitBooking.hotelName}</div>
						<div class="text-2xs text-slate-400 mt-1">Meal Plan: Daily Buffet Breakfast Included. Complimentary Wi-Fi.</div>
					</div>

					<!-- Emergency Concierge -->
					<div class="p-3 bg-emerald-500/10 rounded border border-emerald-500/30 text-xs text-emerald-300 flex justify-between items-center">
						<div>
							<strong>24/7 On-Trip Concierge Desk:</strong>
							<div>Local Airport Meet &amp; Greet: +971 50 123 4567 (Ahmed)</div>
						</div>
						<span class="text-xl">📞</span>
					</div>
				</div>
			</div>
		</div>
	{/if}

	<!-- MODAL: SHARED WORK MODAL (DYNAMIC CROSS-MODULE HUB) -->
	{#if showSharedWorkModal && workBooking}
		<SharedWorkModal
			businessDomain="TRAVEL"
			referenceId={workBooking.id}
			customerName={workBooking.customerName}
			customerPhone={workBooking.customerPhone}
			customerEmail={workBooking.customerEmail}
			title={workBooking.packageTitle}
			amount={workBooking.totalAmount}
			taxRatePercent={5}
			tcsRatePercent={5}
			onClose={() => showSharedWorkModal = false}
			onSuccess={(result) => {
				statusMsg = `✅ Successfully created Invoice #${result.invoiceNumber} and dispatched ${result.tasksDispatched} tasks!`;
				setTimeout(() => statusMsg = '', 5000);
			}}
		/>
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
						<input id="pkg-title" class="text-input" type="text" bind:value={newPkgTitle} placeholder="e.g. Exotic Bali Beach &amp; Kintamani 6D/5N" />
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
					<button class="btn btn-primary" onclick={handleCreatePackage}>Save &amp; Publish Package</button>
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
							<span>Original Passport with &gt; 6 months validity &amp; 2 blank pages</span>
						</label>
						<label class="flex items-center gap-2">
							<input type="checkbox" bind:checked={newVisaChecklist.photographWhiteBg} />
							<span>Recent 35×45mm colour photograph (80% face coverage)</span>
						</label>
						<label class="flex items-center gap-2">
							<input type="checkbox" bind:checked={newVisaChecklist.bankStatementSixMonths} />
							<span>6-month bank statement with branch seal &amp; minimum balance</span>
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
		text-decoration: none;
	}

	.btn-sm {
		font-size: 0.75rem;
		padding: 0.35rem 0.65rem;
		border-radius: 6px;
	}

	.btn-primary { background: #0284c7; color: #ffffff; }
	.btn-primary:hover { background: #0369a1; }
	.btn-secondary { background: #334155; color: #f1f5f9; }
	.btn-secondary:hover { background: #475569; }
	.btn-success { background: #10b981; color: #ffffff; }
	.btn-success:hover { background: #059669; }
	.btn-whatsapp { background: #25d366; color: #000000; font-weight: 700; }
	.btn-whatsapp:hover { background: #20ba59; }

	.btn-outline {
		background: transparent;
		border: 1px solid rgba(255, 255, 255, 0.2);
		color: #cbd5e1;
	}
	.btn-outline:hover { background: rgba(255, 255, 255, 0.05); }

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
		background: rgba(15, 23, 42, 0.75);
		backdrop-filter: blur(12px);
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 14px;
		padding: 1.25rem;
		position: relative;
		overflow: hidden;
		transition: all 0.2s cubic-bezier(0.16, 1, 0.3, 1);
	}

	.kpi-card:hover {
		transform: translateY(-2px);
		box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.5);
		border-color: rgba(255, 255, 255, 0.16);
	}

	.kpi-card::before {
		content: '';
		position: absolute;
		top: 0;
		left: 0;
		right: 0;
		height: 3px;
	}

	.kpi-card-emerald::before { background: linear-gradient(90deg, #10b981, #059669); }
	.kpi-card-amber::before { background: linear-gradient(90deg, #f59e0b, #d97706); }
	.kpi-card-sky::before { background: linear-gradient(90deg, #0ea5e9, #0284c7); }
	.kpi-card-purple::before { background: linear-gradient(90deg, #a855f7, #7c3aed); }

	.kpi-label {
		font-size: 0.75rem;
		font-weight: 700;
		color: #94a3b8;
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}

	.kpi-value {
		font-size: 1.75rem;
		font-weight: 800;
		margin: 0.35rem 0 0.25rem;
		font-family: var(--font-sans, sans-serif);
		letter-spacing: -0.02em;
	}

	.kpi-val-emerald { color: #34d399 !important; text-shadow: 0 0 20px rgba(16, 185, 129, 0.25); }
	.kpi-val-amber { color: #fbbf24 !important; text-shadow: 0 0 20px rgba(245, 158, 11, 0.25); }
	.kpi-val-sky { color: #38bdf8 !important; text-shadow: 0 0 20px rgba(56, 189, 248, 0.25); }
	.kpi-val-purple { color: #c084fc !important; text-shadow: 0 0 20px rgba(168, 85, 247, 0.25); }

	.kpi-sub {
		font-size: 0.75rem;
		color: #94a3b8;
		font-weight: 500;
	}

	/* NAVIGATION TABS */
	.nav-tabs {
		display: flex;
		gap: 0.5rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.1);
		margin-bottom: 1.5rem;
		overflow-x: auto;
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
		white-space: nowrap;
		transition: all 0.15s ease;
	}

	.tab-btn.active {
		color: #38bdf8;
		border-bottom-color: #38bdf8;
	}

	.tab-btn:hover:not(.active) { color: #cbd5e1; }

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

	/* BOOKINGS GRID */
	.bookings-grid {
		display: flex;
		flex-direction: column;
		gap: 1.25rem;
	}

	.booking-card {
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 12px;
		padding: 1.5rem;
		display: flex;
		flex-direction: column;
		gap: 1.25rem;
		transition: border-color 0.15s;
	}

	.booking-card:hover {
		border-color: rgba(56, 189, 248, 0.3);
	}

	.booking-ribbon {
		display: flex;
		justify-content: space-between;
		align-items: center;
		border-bottom: 1px solid rgba(255, 255, 255, 0.06);
		padding-bottom: 0.75rem;
	}

	.booking-ref {
		font-family: monospace;
		font-weight: 800;
		color: #38bdf8;
		font-size: 0.875rem;
		background: rgba(56, 189, 248, 0.1);
		padding: 0.2rem 0.5rem;
		border-radius: 4px;
	}

	.booking-dates {
		font-size: 0.8125rem;
		color: #cbd5e1;
		font-weight: 600;
	}

	.booking-main {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		flex-wrap: wrap;
		gap: 1rem;
	}

	.cust-name {
		font-size: 1.25rem;
		font-weight: 800;
		color: #ffffff;
	}

	.cust-meta {
		display: flex;
		gap: 1rem;
		font-size: 0.75rem;
		color: #94a3b8;
		margin: 0.25rem 0;
	}

	.trip-title {
		font-size: 0.875rem;
		color: #38bdf8;
		font-weight: 600;
	}

	.finance-block {
		text-align: right;
	}

	.finance-total {
		font-size: 1.5rem;
		font-weight: 800;
		color: #ffffff;
	}

	.finance-meta {
		font-size: 0.75rem;
		margin: 0.2rem 0;
	}

	.invoice-ref {
		font-size: 0.6875rem;
		color: #94a3b8;
		font-family: monospace;
	}

	/* SUPPLIERS GRID */
	.suppliers-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
		gap: 1rem;
		background: rgba(255, 255, 255, 0.02);
		border: 1px solid rgba(255, 255, 255, 0.06);
		border-radius: 8px;
		padding: 1rem;
	}

	.supplier-cell {
		display: flex;
		flex-direction: column;
		gap: 0.2rem;
		font-size: 0.75rem;
	}

	.sup-label {
		color: #94a3b8;
		font-weight: 600;
	}

	.sup-status {
		font-size: 0.6875rem;
		color: #34d399;
	}

	.sup-status.warn {
		color: #f87171;
		font-weight: 700;
	}

	.sup-sub {
		color: #cbd5e1;
		font-size: 0.6875rem;
	}

	/* READINESS BAR */
	.readiness-bar {
		background: #1e293b;
		border-radius: 8px;
		padding: 0.875rem 1rem;
		display: flex;
		align-items: center;
		justify-content: space-between;
		flex-wrap: wrap;
		gap: 0.75rem;
	}

	.readiness-label {
		font-size: 0.75rem;
		font-weight: 700;
		color: #cbd5e1;
	}

	.readiness-items {
		display: flex;
		gap: 0.5rem;
		flex-wrap: wrap;
	}

	.check-pill-interactive {
		background: rgba(255, 255, 255, 0.05);
		border: 1px solid rgba(255, 255, 255, 0.1);
		color: #94a3b8;
		font-size: 0.6875rem;
		font-weight: 600;
		padding: 0.3rem 0.6rem;
		border-radius: 6px;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.check-pill-interactive.done {
		background: rgba(16, 185, 129, 0.15);
		border-color: rgba(16, 185, 129, 0.3);
		color: #34d399;
	}

	.booking-footer {
		display: flex;
		justify-content: flex-end;
		border-top: 1px solid rgba(255, 255, 255, 0.06);
		padding-top: 0.75rem;
	}

	.btn-group-footer {
		display: flex;
		gap: 0.5rem;
		flex-wrap: wrap;
	}

	/* MANIFEST CARD */
	.manifest-card {
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 12px;
		padding: 1.25rem;
	}

	.manifest-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		border-bottom: 1px solid rgba(255, 255, 255, 0.06);
		padding-bottom: 0.75rem;
	}

	.rooming-pill {
		font-size: 0.75rem;
		color: #fbbf24;
		background: rgba(245, 158, 11, 0.1);
		padding: 0.25rem 0.6rem;
		border-radius: 6px;
	}

	.meal-tag {
		font-size: 0.6875rem;
		font-weight: 700;
		padding: 0.15rem 0.4rem;
		border-radius: 4px;
	}
	.meal-veg { background: rgba(16, 185, 129, 0.15); color: #34d399; }
	.meal-jain { background: rgba(245, 158, 11, 0.15); color: #fbbf24; }
	.meal-non_veg { background: rgba(239, 68, 68, 0.15); color: #f87171; }

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

	.status-pill {
		font-size: 0.6875rem;
		font-weight: 700;
		padding: 0.2rem 0.5rem;
		border-radius: 6px;
		text-transform: uppercase;
	}
	.status-confirmed { background: rgba(16, 185, 129, 0.15); color: #34d399; }
	.status-operations_hold { background: rgba(245, 158, 11, 0.15); color: #fbbf24; }
	.status-in_transit { background: rgba(56, 189, 248, 0.15); color: #38bdf8; }
	.status-completed { background: rgba(168, 85, 247, 0.15); color: #c084fc; }

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

	.form-row { margin-bottom: 1rem; }
	.form-row label { display: block; font-size: 0.8125rem; color: #94a3b8; margin-bottom: 0.35rem; }

	.input-prefix {
		display: flex;
		align-items: center;
		background: #1e293b;
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: 6px;
		padding: 0 0.75rem;
	}

	.input-prefix span { color: #94a3b8; font-weight: 600; }
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

	.mono-num { font-family: monospace; font-size: 1.125rem; font-weight: 700; color: #ffffff; }

	.quotation-card { border-color: rgba(56, 189, 248, 0.3); }
	.quotation-body { display: flex; flex-direction: column; gap: 1.25rem; }

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

	.label-bold { font-size: 0.8125rem; font-weight: 700; color: #f1f5f9; margin-bottom: 0.6rem; }
	.radio-group { display: flex; flex-direction: column; gap: 0.5rem; }
	.radio-label { display: flex; align-items: center; gap: 0.6rem; font-size: 0.8125rem; color: #cbd5e1; cursor: pointer; }

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
		background: rgba(0, 0, 0, 0.85);
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

	.input-label { display: block; font-size: 0.75rem; font-weight: 600; color: #94a3b8; margin-bottom: 0.35rem; }
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
	.itinerary-timeline { display: flex; flex-direction: column; gap: 1rem; margin-bottom: 1.5rem; }
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

	.day-content { flex: 1; }
	.day-title { font-size: 1rem; font-weight: 700; color: #ffffff; margin-bottom: 0.25rem; }
	.day-activities { font-size: 0.8125rem; color: #94a3b8; margin-bottom: 0.5rem; }
	.day-meta { display: flex; gap: 1rem; font-size: 0.75rem; color: #38bdf8; }

	.voucher-footer {
		display: flex;
		justify-content: space-between;
		align-items: center;
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: 10px;
		padding: 1rem;
	}

	.voucher-notes { font-size: 0.75rem; color: #94a3b8; line-height: 1.4; }

	/* PRINT STYLES */
	@media print {
		.no-print, .nav-tabs, .kpi-grid, .app-header { display: none !important; }
		body { background: white !important; color: black !important; }
		.printable-voucher { background: white !important; color: black !important; border: 1px solid #ccc !important; box-shadow: none !important; }
		.timeline-day, .voucher-footer { background: white !important; border: 1px solid #eee !important; color: black !important; }
		.day-title, .day-activities { color: black !important; }
	}
</style>
