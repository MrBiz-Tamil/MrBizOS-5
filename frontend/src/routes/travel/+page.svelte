<script lang="ts">
	import { onMount } from 'svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { travelGrpc } from '$lib/grpc';

	import { page } from '$app/stores';
	import { goto } from '$app/navigation';

	// Modal Components (Screens 2 - 12)
	import CreateBookingModal from '$lib/components/travel/CreateBookingModal.svelte';
	import ItineraryBuilderModal from '$lib/components/travel/ItineraryBuilderModal.svelte';
	import PassengerModal from '$lib/components/travel/PassengerModal.svelte';
	import VisaModal from '$lib/components/travel/VisaModal.svelte';
	import PaymentInvoiceModal from '$lib/components/travel/PaymentInvoiceModal.svelte';
	import FlightBookingModal from '$lib/components/travel/FlightBookingModal.svelte';
	import TourCalendarModal from '$lib/components/travel/TourCalendarModal.svelte';
	import TripMapModal from '$lib/components/travel/TripMapModal.svelte';
	import WhatsAppLeadModal from '$lib/components/travel/WhatsAppLeadModal.svelte';
	import ConfirmActionModal from '$lib/components/travel/ConfirmActionModal.svelte';
	import AppLauncherModal from '$lib/components/travel/AppLauncherModal.svelte';
	import OnGroundDispatchModal from '$lib/components/travel/OnGroundDispatchModal.svelte';
	import WhatsAppTravelKitModal from '$lib/components/travel/WhatsAppTravelKitModal.svelte';

	// URL View Query Effect (Screens 2 - 12)
	$effect(() => {
		const view = $page.url.searchParams.get('view');
		if (!view) return;
		if (view === 'crm') goto('/travel/crm');
		else if (view === 'bookings') showCreateBooking = true;
		else if (view === 'customers') showPassengerModal = true;
		else if (view === 'itineraries') showItineraryBuilder = true;
		else if (view === 'visa') showVisaModal = true;
		else if (view === 'tickets') showFlightModal = true;
		else if (view === 'calendar') showCalendarModal = true;
		else if (view === 'finance') showPaymentModal = true;
		else if (view === 'operations') {
			selectedBooking = bookings[0];
			showDispatchModal = true;
		} else if (view === 'packages' || view === 'departures') {
			const el = document.getElementById('departures-section');
			if (el) el.scrollIntoView({ behavior: 'smooth' });
		}
	});

	// Modals Open/Close State
	let showCreateBooking = $state(false);
	let showItineraryBuilder = $state(false);
	let showPassengerModal = $state(false);
	let showVisaModal = $state(false);
	let showPaymentModal = $state(false);
	let showFlightModal = $state(false);
	let showCalendarModal = $state(false);
	let showTripMapModal = $state(false);
	let showWhatsAppLeadModal = $state(false);
	let showConfirmModal = $state(false);
	let showAppLauncherModal = $state(false);
	let showDispatchModal = $state(false);
	let showTravelKitModal = $state(false);

	let selectedBooking = $state<any>(null);
	let selectedLead = $state<any>(null);

	// Departures Dataset (Screen 1)
	let departures = $state([
		{
			id: 'dep-1',
			date: '22 Oct 2026',
			package: 'Kashmir Valley Paradise',
			destination: 'Srinagar - Gulmarg',
			booked: 24,
			capacity: 30,
			status: 'Confirmed',
			statusClass: 'status-confirmed'
		},
		{
			id: 'dep-2',
			date: '05 Nov 2026',
			package: 'Dubai Special Desert & City',
			destination: 'Dubai - Abu Dhabi',
			booked: 18,
			capacity: 25,
			status: 'Filling',
			statusClass: 'status-filling'
		},
		{
			id: 'dep-3',
			date: '14 Nov 2026',
			package: 'Europe Classic Express',
			destination: 'Paris - Rome - Zurich',
			booked: 30,
			capacity: 30,
			status: 'Fully Booked',
			statusClass: 'status-full'
		},
		{
			id: 'dep-4',
			date: '02 Dec 2026',
			package: 'Thailand Leisure Explorer',
			destination: 'Bangkok - Phuket',
			booked: 16,
			capacity: 20,
			status: 'Open',
			statusClass: 'status-open'
		},
		{
			id: 'dep-5',
			date: '20 Dec 2026',
			package: 'Singapore Sentosa Explorer',
			destination: 'Singapore',
			booked: 12,
			capacity: 25,
			status: 'Open',
			statusClass: 'status-open'
		}
	]);

	// Recent Leads Dataset (Screen 1)
	let recentLeads = $state([
		{
			id: 'lead-1',
			name: 'Rahul Sharma',
			tour: 'Kashmir Family Tour',
			channel: 'whatsapp',
			time: '10 mins ago',
			status: 'New',
			statusClass: 'lead-new'
		},
		{
			id: 'lead-2',
			name: 'Priya Menon',
			tour: 'Dubai Special',
			channel: 'whatsapp',
			time: '25 mins ago',
			status: 'In Progress',
			statusClass: 'lead-progress'
		},
		{
			id: 'lead-3',
			name: 'Arun Kumar',
			tour: 'Europe Group Tour',
			channel: 'instagram',
			time: '1 hour ago',
			status: 'Qualified',
			statusClass: 'lead-qualified'
		},
		{
			id: 'lead-4',
			name: 'Meena Iyer',
			tour: 'Thailand Tour',
			channel: 'facebook',
			time: '2 hours ago',
			status: 'New',
			statusClass: 'lead-new'
		},
		{
			id: 'lead-5',
			name: 'Vikram Reddy',
			tour: 'Singapore Business',
			channel: 'whatsapp',
			time: '3 hours ago',
			status: 'Follow Up',
			statusClass: 'lead-followup'
		}
	]);

	// Real-World Linked Bookings (Chennai Kashmir Family & Madurai Malaysia Business)
	let bookings = $state([
		{
			id: 'TRV-2026-001',
			customerName: 'Rahul Sharma & Family',
			customerCity: 'Chennai',
			customerPhone: '+91 98401 22334',
			customerEmail: 'rahul.sharma@example.com',
			tripMode: 'FAMILY',
			destination: 'Kashmir Valley (Srinagar - Gulmarg - Pahalgam)',
			packageTitle: 'Kashmir Autumn Paradise (4D/5N)',
			travelDates: '14 Nov 2026 - 19 Nov 2026',
			paxCount: 4,
			status: 'CONFIRMED',
			totalAmount: '₹ 1,65,000',
			hotelName: 'Hotel Grand Kashmir & Betaab Luxury Resort',
			hotelDmcRef: 'DMC-SXR-8819',
			airlinePnr: '6E-502 / AI-991',
			visaStatus: 'Domestic Circuit (Permit Valid)',
			assignedCoordinator: {
				name: 'Kavitha R.',
				phone: '+91 98840 11223',
				department: 'North India Operations Desk'
			},
			logisticsDispatch: {
				driverName: 'Ghulam Nabi',
				driverPhone: '+91 94190 12345',
				vehicleModel: 'Toyota Innova Crysta AC (White)',
				vehiclePlate: 'JK-01-AB-4491',
				pickupLocation: 'Srinagar Airport Exit Gate 1 (Placard: "SHARMA FAMILY")',
				flightMonitor: 'IndiGo 6E-502 (Tracking on-time)',
				guideName: 'Bashir Ahmed (Govt. Licensed #2094)',
				guidePhone: '+91 97970 88990',
				guideLanguages: 'Tamil (conversational), English, Hindi'
			},
			localEmergency: {
				stationLead: 'Khurshid (Srinagar Station Chief)',
				emergencyPhone: '+91 94190 99881',
				hospitalPartner: 'SMHS Hospital Srinagar (15 mins away)'
			}
		},
		{
			id: 'TRV-2026-002',
			customerName: 'Karthik Subramanian',
			customerCity: 'Madurai',
			customerPhone: '+91 94432 55678',
			customerEmail: 'karthik.sub@maduraitech.in',
			tripMode: 'BUSINESS',
			destination: 'Kuala Lumpur & Penang (Malaysia)',
			packageTitle: 'Malaysia Corporate Summit & Networking (3D/2N)',
			travelDates: '28 Nov 2026 - 01 Dec 2026',
			paxCount: 1,
			status: 'IN_PROGRESS',
			totalAmount: '₹ 72,000',
			hotelName: 'Hilton Kuala Lumpur City Centre',
			hotelDmcRef: 'DMC-KUL-4402',
			airlinePnr: 'MH-183 (Malaysia Airlines)',
			visaStatus: 'eVisa Approved (Ref: MY-88219)',
			assignedCoordinator: {
				name: 'Muthuvel S.',
				phone: '+91 98402 33445',
				department: 'SE Asia Corporate Desk'
			},
			logisticsDispatch: {
				driverName: 'Tan Wei Meng',
				driverPhone: '+60 12 345 6789',
				vehicleModel: 'Mercedes-Benz E-Class Executive',
				vehiclePlate: 'WXY 8892',
				pickupLocation: 'KLIA Terminal 1 International Arrivals Gate 3',
				flightMonitor: 'MH-183 Arriving 06:45 AM',
				guideName: 'Govindasamy (Licensed Tourist Guide)',
				guidePhone: '+60 16 987 6543',
				guideLanguages: 'Tamil, English, Malay'
			},
			localEmergency: {
				stationLead: 'KL Embassy Station Liaison',
				emergencyPhone: '+60 3 2147 8000',
				hospitalPartner: 'Gleneagles Hospital Kuala Lumpur'
			}
		}
	]);

	onMount(async () => {
		try {
			const res = await travelGrpc.listTourPackages();
			if (res && res.packages && res.packages.length > 0) {
				console.log('Loaded backend tour packages:', res.packages.length);
			}
		} catch (e) {
			console.log('ConnectRPC connected to local state fallback');
		}
	});

	function openWhatsAppKit(b: any) {
		selectedBooking = b;
		showTravelKitModal = true;
	}

	function openDispatchEditor(b: any) {
		selectedBooking = b;
		showDispatchModal = true;
	}

	function openLeadChat(lead: any) {
		selectedLead = lead;
		showWhatsAppLeadModal = true;
	}

</script>

<svelte:head>
	<title>Travel &amp; Tours OS | MRBiz OS — Integrated Travel Operator System</title>
</svelte:head>

<div class="travel-full-canvas">
	<!-- Top Masthead Banner -->
	<header class="travel-top-masthead">
		<div class="masthead-left">
			<div class="icon-title-wrap">
					<div class="airplane-badge">✈️</div>
					<div>
						<h1 class="screen-title">Travel &amp; Tours OS</h1>
						<div class="sub-nav-links">
							<a href="/travel/book" class="sub-link-btn" style="color: #16a34a; font-weight: 800;">⚡ Book a Trip (1-Page)</a> • 
							<a href="/travel/crm" class="sub-link-btn" style="color: #7c3aed; font-weight: 700;">👥 Odoo CRM &amp; UTMs</a> • 
							<button type="button" class="sub-link-btn" onclick={() => (showCreateBooking = true)}>Bookings</button> • 
							<button type="button" class="sub-link-btn" onclick={() => (showVisaModal = true)}>Visa</button> • 
							<button type="button" class="sub-link-btn" onclick={() => (showCalendarModal = true)}>Tours</button> • 
							<button type="button" class="sub-link-btn" onclick={() => (showItineraryBuilder = true)}>Itineraries</button> • 
							<button type="button" class="sub-link-btn" onclick={() => openDispatchEditor(bookings[0])}>Operations</button> • 
							<button type="button" class="sub-link-btn" onclick={() => (showPaymentModal = true)}>Finance</button>
						</div>
					</div>
				</div>
			</div>

			<div class="masthead-right">
				<div class="explore-banner">
					<span class="explore-text">Explore the World ✈️</span>
				</div>
				<div class="date-dropdown-box" onclick={() => (showCalendarModal = true)} role="button" tabindex="0" onkeydown={(e) => { if (e.key === 'Enter') showCalendarModal = true; }}>
					<span class="date-icon">📅</span>
					<span class="date-text">Today • 22 Oct 2026</span>
					<span class="arrow-down">▾</span>
				</div>
				<button type="button" class="btn btn-sm btn-outline launcher-btn" onclick={() => (showAppLauncherModal = true)} title="Open App Launcher">
					Apps ⊞
				</button>
			</div>
		</header>

		<!-- 6 Actionable KPI Cards -->
		<section class="kpi-banner-grid">
			<!-- 1. Total Bookings -->
			<div class="kpi-tile kpi-blue">
				<div class="kpi-icon-circle blue">✈️</div>
				<div class="kpi-details">
					<span class="kpi-tag">Total Bookings</span>
					<div class="kpi-num-row">
						<span class="kpi-num">186</span>
						<span class="kpi-growth up">▲ 12%</span>
					</div>
					<span class="kpi-sub">This Month</span>
				</div>
			</div>

			<!-- 2. Revenue -->
			<div class="kpi-tile kpi-green">
				<div class="kpi-icon-circle green">₹</div>
				<div class="kpi-details">
					<span class="kpi-tag">Revenue</span>
					<div class="kpi-num-row">
						<span class="kpi-num">₹ 24,60,000</span>
						<span class="kpi-growth up">▲ 18%</span>
					</div>
					<span class="kpi-sub">This Month</span>
				</div>
			</div>

			<!-- 3. Active Customers -->
			<div class="kpi-tile kpi-purple">
				<div class="kpi-icon-circle purple">👥</div>
				<div class="kpi-details">
					<span class="kpi-tag">Active Customers</span>
					<div class="kpi-num-row">
						<span class="kpi-num">324</span>
						<span class="kpi-growth up">▲ 9%</span>
					</div>
					<span class="kpi-sub">Across Circuits</span>
				</div>
			</div>

			<!-- 4. Pending Visa -->
			<div class="kpi-tile kpi-red">
				<div class="kpi-icon-circle red">🛂</div>
				<div class="kpi-details">
					<span class="kpi-tag">Pending Visa</span>
					<div class="kpi-num-row">
						<span class="kpi-num">48</span>
						<span class="kpi-growth down">▼ 6%</span>
					</div>
					<span class="kpi-sub">In Process</span>
				</div>
			</div>

			<!-- 5. Upcoming Tours -->
			<div class="kpi-tile kpi-amber">
				<div class="kpi-icon-circle amber">🧳</div>
				<div class="kpi-details">
					<span class="kpi-tag">Upcoming Tours</span>
					<div class="kpi-num-row">
						<span class="kpi-num">12</span>
						<span class="kpi-growth up">▲ 5%</span>
					</div>
					<span class="kpi-sub">Next 30 Days</span>
				</div>
			</div>

			<!-- 6. Payments Due -->
			<div class="kpi-tile kpi-sky">
				<div class="kpi-icon-circle sky">💳</div>
				<div class="kpi-details">
					<span class="kpi-tag">Payments Due</span>
					<div class="kpi-num-row">
						<span class="kpi-num">₹ 7,85,000</span>
					</div>
					<span class="kpi-sub">14 Invoices</span>
				</div>
			</div>
		</section>

		<!-- Middle Operational Tri-Panel Section -->
		<section class="middle-tri-grid">
			<!-- Panel 1: Departures This Month (Table) -->
			<div class="cockpit-card departures-panel">
				<div class="panel-header">
					<div>
						<h3 class="panel-title">Departures This Month</h3>
						<span class="panel-sub">Capacity &amp; Scheduled departures</span>
					</div>
					<button type="button" class="link-btn" onclick={() => (showCalendarModal = true)}>View All ›</button>
				</div>

				<div class="table-wrapper">
					<table class="departures-table">
						<thead>
							<tr>
								<th>Date</th>
								<th>Tour Package</th>
								<th>Destination</th>
								<th class="text-center">Booked/Capacity</th>
								<th>Status</th>
								<th class="text-center">Action</th>
							</tr>
						</thead>
						<tbody>
							{#each departures as dep}
								<tr>
									<td class="font-mono font-semibold">{dep.date}</td>
									<td class="font-bold text-slate-900">{dep.package}</td>
									<td class="text-slate-600">{dep.destination}</td>
									<td class="font-mono text-center">
										<strong>{dep.booked}</strong>/{dep.capacity}
									</td>
									<td>
										<span class="status-pill {dep.statusClass}">{dep.status}</span>
									</td>
									<td class="text-center">
										<button 
											type="button" 
											class="icon-menu-btn" 
											onclick={() => { toast.success(`Viewing departure manifest for ${dep.package}`); showPassengerModal = true; }}
											title="View Passenger Manifest"
										>
											•••
										</button>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</div>

			<!-- Panel 2: Booking Status (Donut Chart) -->
			<div class="cockpit-card status-donut-panel">
				<div class="panel-header">
					<h3 class="panel-title">Booking Status</h3>
					<span class="period-pill">This Month ▾</span>
				</div>

				<div class="donut-chart-container">
					<!-- Interactive SVG Donut Chart -->
					<div class="donut-graphic">
						<svg viewBox="0 0 160 160" class="donut-svg">
							<!-- Confirmed 52% (Green) -->
							<circle cx="80" cy="80" r="60" fill="transparent" stroke="#10b981" stroke-width="20" stroke-dasharray="196 377" stroke-dashoffset="0" />
							<!-- Pending 17% (Sky) -->
							<circle cx="80" cy="80" r="60" fill="transparent" stroke="#0ea5e9" stroke-width="20" stroke-dasharray="64 377" stroke-dashoffset="-196" />
							<!-- Quotation 15% (Amber) -->
							<circle cx="80" cy="80" r="60" fill="transparent" stroke="#f59e0b" stroke-width="20" stroke-dasharray="56 377" stroke-dashoffset="-260" />
							<!-- Cancelled 6% (Rose) -->
							<circle cx="80" cy="80" r="60" fill="transparent" stroke="#f43f5e" stroke-width="20" stroke-dasharray="23 377" stroke-dashoffset="-316" />
							<!-- On Hold 9% (Purple) -->
							<circle cx="80" cy="80" r="60" fill="transparent" stroke="#8b5cf6" stroke-width="20" stroke-dasharray="34 377" stroke-dashoffset="-339" />
						</svg>
						<div class="donut-center-label">
							<span class="donut-total font-mono">186</span>
							<span class="donut-sub">Total</span>
						</div>
					</div>

					<div class="donut-legend">
						<div class="legend-row"><span class="legend-dot" style="background: #10b981;"></span><span class="leg-name">Confirmed</span><span class="leg-val">96 (52%)</span></div>
						<div class="legend-row"><span class="legend-dot" style="background: #0ea5e9;"></span><span class="leg-name">Pending</span><span class="leg-val">32 (17%)</span></div>
						<div class="legend-row"><span class="legend-dot" style="background: #f59e0b;"></span><span class="leg-name">Quotation</span><span class="leg-val">28 (15%)</span></div>
						<div class="legend-row"><span class="legend-dot" style="background: #f43f5e;"></span><span class="leg-name">Cancelled</span><span class="leg-val">12 (6%)</span></div>
						<div class="legend-row"><span class="legend-dot" style="background: #8b5cf6;"></span><span class="leg-name">On Hold</span><span class="leg-val">16 (9%)</span></div>
					</div>
				</div>
			</div>

			<!-- Panel 3: Recent Leads (Omnichannel Feed) -->
			<div class="cockpit-card leads-panel">
				<div class="panel-header">
					<div>
						<h3 class="panel-title">Recent Leads</h3>
						<span class="panel-sub">Social &amp; Inbound Enquiries</span>
					</div>
					<a href="/travel/crm" class="link-btn">Odoo Pipeline ›</a>
				</div>

				<div class="leads-list">
					{#each recentLeads as lead}
						<button type="button" class="lead-item" onclick={() => openLeadChat(lead)}>
							<div class="lead-channel-icon {lead.channel}">
								{#if lead.channel === 'whatsapp'}📱{:else if lead.channel === 'instagram'}📸{:else}👥{/if}
							</div>
							<div class="lead-info">
								<div class="lead-title-row">
									<span class="lead-name">{lead.name}</span>
									<span class="lead-time font-mono">{lead.time}</span>
								</div>
								<div class="lead-tour">{lead.tour}</div>
							</div>
							<span class="lead-status-pill {lead.statusClass}">{lead.status}</span>
						</button>
					{/each}
				</div>
			</div>
		</section>

		<!-- Quick Actions Bar (Bottom Ribbon with 10 Action Launchers) -->
		<section class="quick-actions-ribbon">
			<button type="button" class="action-tile" onclick={() => (showCreateBooking = true)}>
				<span class="act-icon">✈️</span>
				<span class="act-text">Create Booking</span>
			</button>

			<button type="button" class="action-tile" onclick={() => (showWhatsAppLeadModal = true)}>
				<span class="act-icon">👤</span>
				<span class="act-text">Add Lead</span>
			</button>

			<button type="button" class="action-tile" onclick={() => (showItineraryBuilder = true)}>
				<span class="act-icon">🗺️</span>
				<span class="act-text">Plan Itinerary</span>
			</button>

			<button type="button" class="action-tile" onclick={() => (showCalendarModal = true)}>
				<span class="act-icon">🔍</span>
				<span class="act-text">Search Tours</span>
			</button>

			<button type="button" class="action-tile" onclick={() => (showVisaModal = true)}>
				<span class="act-icon">🛂</span>
				<span class="act-text">Process Visa</span>
			</button>

			<button type="button" class="action-tile" onclick={() => (showFlightModal = true)}>
				<span class="act-icon">🛫</span>
				<span class="act-text">Book Flight</span>
			</button>

			<button type="button" class="action-tile" onclick={() => openDispatchEditor(bookings[0])}>
				<span class="act-icon">🚗</span>
				<span class="act-text">Ground Dispatch</span>
			</button>

			<button type="button" class="action-tile" onclick={() => openWhatsAppKit(bookings[0])}>
				<span class="act-icon">📱</span>
				<span class="act-text">Send Travel Kit</span>
			</button>

			<button type="button" class="action-tile" onclick={() => (showPaymentModal = true)}>
				<span class="act-icon">💳</span>
				<span class="act-text">Create Invoice</span>
			</button>

			<button type="button" class="action-tile action-more" onclick={() => (showTripMapModal = true)}>
				<span class="act-icon">📍</span>
				<span class="act-text">Trip Map</span>
			</button>
		</section>

		<!-- Operations Booking List Section (Single Linked Booking Execution) -->
		<section class="booking-ops-deck">
			<div class="deck-header">
				<div>
					<h3 class="deck-title">Active Booking Operations</h3>
					<span class="deck-sub">Chennai Kashmir Family, Madurai Malaysia Corporate &amp; Kanyakumari Kerala Tours</span>
				</div>
				<div class="deck-actions">
					<button type="button" class="btn btn-sm btn-outline" onclick={() => (showPassengerModal = true)}>
						+ Add Passenger
					</button>
					<button type="button" class="btn btn-sm btn-primary" onclick={() => (showCreateBooking = true)}>
						+ New Booking
					</button>
				</div>
			</div>

			<div class="booking-cards-list">
				{#each bookings as b}
					<div class="ops-booking-card">
						<div class="card-head-strip">
							<div class="cust-meta">
								<div class="cust-avatar">{b.customerName.slice(0, 2).toUpperCase()}</div>
								<div>
									<span class="b-cust-name">{b.customerName}</span>
									<span class="b-ref font-mono">{b.id} • {b.customerCity}</span>
								</div>
							</div>
							<div class="mode-badges">
								<span class="mode-tag mode-{b.tripMode.toLowerCase()}">{b.tripMode}</span>
								<span class="status-badge status-confirmed">{b.status}</span>
							</div>
						</div>

						<!-- 4 Dispatch Pillars -->
						<div class="pillars-deck">
							<div class="pillar-box">
								<span class="p-title">🚗 CHAUFFEUR &amp; CAB</span>
								<span class="p-main">{b.logisticsDispatch.driverName}</span>
								<span class="p-sub font-mono">{b.logisticsDispatch.vehicleModel} ({b.logisticsDispatch.vehiclePlate})</span>
								<span class="p-spec">{b.logisticsDispatch.pickupLocation}</span>
							</div>
							<div class="pillar-box">
								<span class="p-title">🏨 RESORT &amp; STAY</span>
								<span class="p-main">{b.hotelName}</span>
								<span class="p-sub">Ref: <strong class="font-mono text-primary">{b.hotelDmcRef}</strong></span>
								<span class="p-spec">Guaranteed Check-in</span>
							</div>
							<div class="pillar-box">
								<span class="p-title">👨‍💼 LOCAL GUIDE</span>
								<span class="p-main">{b.logisticsDispatch.guideName}</span>
								<span class="p-sub">Languages: {b.logisticsDispatch.guideLanguages}</span>
								<span class="p-spec font-mono">📞 {b.logisticsDispatch.guidePhone}</span>
							</div>
							<div class="pillar-box">
								<span class="p-title">🆘 24/7 COORDINATOR</span>
								<span class="p-main">{b.assignedCoordinator.name}</span>
								<span class="p-sub">{b.assignedCoordinator.department}</span>
								<span class="p-spec font-mono">📞 {b.assignedCoordinator.phone}</span>
							</div>
						</div>

						<!-- Card Actions -->
						<div class="card-footer-bar">
							<div class="price-chip">
								<span class="pr-label">Total Booking Value:</span>
								<span class="pr-val font-mono">{b.totalAmount}</span>
							</div>
							<div class="btns-group">
								<button type="button" class="btn btn-sm btn-outline" onclick={() => openDispatchEditor(b)}>
									🚗 Edit Ground Dispatch
								</button>
								<button type="button" class="btn btn-sm btn-whatsapp" onclick={() => openWhatsAppKit(b)}>
									📱 WhatsApp Travel Kit
								</button>
								<button type="button" class="btn btn-sm btn-secondary" onclick={() => (showConfirmModal = true)}>
									Cancel
								</button>
							</div>
						</div>
					</div>
				{/each}
			</div>
		</section>
</div>

<!-- Modals (Screens 2 - 12) -->
<CreateBookingModal 
	open={showCreateBooking} 
	onclose={() => (showCreateBooking = false)} 
	onsave={(newBooking) => bookings = [newBooking, ...bookings]}
/>

<ItineraryBuilderModal 
	open={showItineraryBuilder} 
	onclose={() => (showItineraryBuilder = false)} 
/>

<PassengerModal 
	open={showPassengerModal} 
	onclose={() => (showPassengerModal = false)} 
/>

<VisaModal 
	open={showVisaModal} 
	onclose={() => (showVisaModal = false)} 
/>

<PaymentInvoiceModal 
	open={showPaymentModal} 
	booking={selectedBooking || bookings[0]}
	onclose={() => (showPaymentModal = false)} 
/>

<FlightBookingModal 
	open={showFlightModal} 
	onclose={() => (showFlightModal = false)} 
/>

<TourCalendarModal 
	open={showCalendarModal} 
	onclose={() => (showCalendarModal = false)} 
/>

<TripMapModal 
	open={showTripMapModal} 
	onclose={() => (showTripMapModal = false)} 
/>

<WhatsAppLeadModal 
	open={showWhatsAppLeadModal} 
	lead={selectedLead || recentLeads[0]}
	onclose={() => (showWhatsAppLeadModal = false)} 
/>

<ConfirmActionModal 
	open={showConfirmModal} 
	onclose={() => (showConfirmModal = false)} 
	onconfirm={(reason) => toast.error(`Booking cancelled. Reason: ${reason}`)}
/>

<AppLauncherModal 
	open={showAppLauncherModal} 
	onclose={() => (showAppLauncherModal = false)} 
/>

<OnGroundDispatchModal 
	open={showDispatchModal} 
	booking={selectedBooking || bookings[0]} 
	onclose={() => (showDispatchModal = false)} 
	onsave={(updated) => {
		const idx = bookings.findIndex(b => b.id === updated.id);
		if (idx !== -1) bookings[idx] = updated;
		toast.success('On-ground driver and local guide dispatch updated!');
	}}
/>

<WhatsAppTravelKitModal 
	open={showTravelKitModal} 
	booking={selectedBooking || bookings[0]} 
	onclose={() => (showTravelKitModal = false)} 
/>

<style>
	/* ==========================================================================
	   SCREEN 1: TRAVEL DASHBOARD ENTERPRISE LIGHT STYLING
	   ========================================================================== */

	.travel-full-canvas {
		min-height: calc(100vh - 64px);
		background: #f8fafc;
		color: #0f172a;
		font-family: var(--font-sans, system-ui, -apple-system, sans-serif);
		padding: 1.5rem 2rem;
		display: flex;
		flex-direction: column;
		gap: 1.5rem;
		width: 100%;
		box-sizing: border-box;
		overflow-x: hidden;
	}

	/* Top Masthead */
	.travel-top-masthead {
		display: flex;
		justify-content: space-between;
		align-items: center;
		flex-wrap: wrap;
		gap: 1rem;
		padding-bottom: 1rem;
		border-bottom: 1px solid #e2e8f0;
	}

	.masthead-left {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}

	.icon-title-wrap {
		display: flex;
		align-items: center;
		gap: 0.85rem;
	}

	.airplane-badge {
		width: 44px;
		height: 44px;
		border-radius: 10px;
		background: #eff6ff;
		color: #2563eb;
		border: 1px solid #bfdbfe;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 1.4rem;
	}

	.screen-title {
		font-size: 1.5rem;
		font-weight: 800;
		color: #0f172a;
		margin: 0;
		letter-spacing: -0.02em;
	}

	.sub-nav-links {
		font-size: 0.8125rem;
		color: #64748b;
		margin-top: 0.2rem;
	}

	.sub-link-btn {
		background: transparent;
		border: none;
		padding: 0;
		font-size: 0.8125rem;
		color: #64748b;
		cursor: pointer;
		font-family: inherit;
	}
	.sub-link-btn:hover { color: #2563eb; text-decoration: underline; }

	.masthead-right {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}

	.explore-banner {
		background: linear-gradient(135deg, #1e40af, #3b82f6);
		color: #ffffff;
		padding: 0.4rem 0.85rem;
		border-radius: 9999px;
		font-size: 0.75rem;
		font-weight: 700;
	}

	.date-dropdown-box {
		display: flex;
		align-items: center;
		gap: 0.4rem;
		background: #ffffff;
		border: 1px solid #cbd5e1;
		border-radius: 8px;
		padding: 0.45rem 0.75rem;
		font-size: 0.8125rem;
		font-weight: 600;
		color: #0f172a;
		cursor: pointer;
	}
	.date-dropdown-box:hover { border-color: #94a3b8; }

	/* 6 KPI Cards */
	.kpi-banner-grid {
		display: grid;
		grid-template-columns: repeat(6, 1fr);
		gap: 0.75rem;
	}

	@media (max-width: 1400px) {
		.kpi-banner-grid { grid-template-columns: repeat(3, 1fr); }
	}
	@media (max-width: 768px) {
		.kpi-banner-grid { grid-template-columns: repeat(2, 1fr); }
	}

	.kpi-tile {
		background: #ffffff;
		border: 1px solid #e2e8f0;
		border-radius: 12px;
		padding: 1rem;
		display: flex;
		align-items: center;
		gap: 0.75rem;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.03);
		transition: transform 0.15s ease;
	}
	.kpi-tile:hover {
		transform: translateY(-2px);
		box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.06);
	}

	.kpi-icon-circle {
		width: 40px;
		height: 40px;
		border-radius: 10px;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 1.25rem;
		flex-shrink: 0;
	}
	.kpi-icon-circle.blue { background: #eff6ff; color: #2563eb; }
	.kpi-icon-circle.green { background: #ecfdf5; color: #059669; }
	.kpi-icon-circle.purple { background: #f5f3ff; color: #7c3aed; }
	.kpi-icon-circle.red { background: #fff1f2; color: #e11d48; }
	.kpi-icon-circle.amber { background: #fef3c7; color: #d97706; }
	.kpi-icon-circle.sky { background: #f0f9ff; color: #0284c7; }

	.kpi-details { display: flex; flex-direction: column; overflow: hidden; }
	.kpi-tag { font-size: 0.7rem; font-weight: 700; color: #64748b; text-transform: uppercase; white-space: nowrap; }
	.kpi-num-row { display: flex; align-items: baseline; gap: 0.35rem; }
	.kpi-num { font-size: 1.25rem; font-weight: 800; color: #0f172a; white-space: nowrap; }
	.kpi-growth { font-size: 0.7rem; font-weight: 700; }
	.kpi-growth.up { color: #059669; }
	.kpi-growth.down { color: #dc2626; }
	.kpi-sub { font-size: 0.65rem; color: #94a3b8; }

	/* Middle Tri-Grid */
	.middle-tri-grid {
		display: grid;
		grid-template-columns: 2fr 1fr 1fr;
		gap: 1rem;
	}

	@media (max-width: 1200px) {
		.middle-tri-grid { grid-template-columns: 1fr; }
	}

	.cockpit-card {
		background: #ffffff;
		border: 1px solid #e2e8f0;
		border-radius: 12px;
		padding: 1.25rem;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.03);
		display: flex;
		flex-direction: column;
	}

	.panel-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 1rem;
		padding-bottom: 0.5rem;
		border-bottom: 1px solid #f1f5f9;
	}

	.panel-title { font-size: 0.95rem; font-weight: 800; color: #0f172a; margin: 0; }
	.panel-sub { font-size: 0.7rem; color: #64748b; }

	.link-btn {
		background: transparent;
		border: none;
		font-size: 0.75rem;
		font-weight: 700;
		color: #2563eb;
		cursor: pointer;
	}

	.period-pill {
		font-size: 0.7rem;
		font-weight: 600;
		background: #f1f5f9;
		padding: 0.2rem 0.55rem;
		border-radius: 6px;
		color: #475569;
		cursor: pointer;
	}

	/* Departures Table */
	.table-wrapper { overflow-x: auto; }
	.departures-table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.8125rem;
	}
	.departures-table th {
		text-align: left;
		padding: 0.55rem 0.75rem;
		color: #64748b;
		font-size: 0.7rem;
		text-transform: uppercase;
		font-weight: 700;
		border-bottom: 1px solid #e2e8f0;
		background: #f8fafc;
	}
	.departures-table td {
		padding: 0.65rem 0.75rem;
		border-bottom: 1px solid #f1f5f9;
	}
	.departures-table tr:hover td { background: #f8fafc; }

	.status-pill {
		font-size: 0.7rem;
		font-weight: 700;
		padding: 0.2rem 0.55rem;
		border-radius: 9999px;
	}
	.status-confirmed { background: #ecfdf5; color: #059669; }
	.status-filling { background: #fef3c7; color: #d97706; }
	.status-full { background: #fee2e2; color: #dc2626; }
	.status-open { background: #eff6ff; color: #2563eb; }

	.icon-menu-btn {
		background: transparent;
		border: none;
		color: #94a3b8;
		font-weight: 800;
		cursor: pointer;
		padding: 0.2rem 0.4rem;
	}
	.icon-menu-btn:hover { color: #0f172a; }

	/* Donut Chart */
	.donut-chart-container {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 1rem;
	}

	.donut-graphic {
		position: relative;
		width: 140px;
		height: 140px;
	}

	.donut-svg {
		width: 100%;
		height: 100%;
		transform: rotate(-90deg);
	}

	.donut-center-label {
		position: absolute;
		inset: 0;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
	}
	.donut-total { font-size: 1.35rem; font-weight: 800; color: #0f172a; }
	.donut-sub { font-size: 0.7rem; color: #64748b; text-transform: uppercase; }

	.donut-legend {
		width: 100%;
		display: flex;
		flex-direction: column;
		gap: 0.35rem;
		font-size: 0.75rem;
	}
	.legend-row { display: flex; align-items: center; justify-content: space-between; }
	.legend-dot { width: 8px; height: 8px; border-radius: 50%; display: inline-block; margin-right: 0.35rem; }
	.leg-name { color: #475569; }
	.leg-val { font-weight: 700; color: #0f172a; font-family: monospace; }

	/* Leads Panel */
	.leads-list {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.lead-item {
		display: flex;
		align-items: center;
		gap: 0.65rem;
		padding: 0.6rem 0.75rem;
		background: #f8fafc;
		border: 1px solid #e2e8f0;
		border-radius: 8px;
		cursor: pointer;
		transition: all 0.12s ease;
		text-align: left;
		font-family: inherit;
		width: 100%;
	}
	.lead-item:hover { border-color: #2563eb; background: #eff6ff; }

	.lead-channel-icon {
		width: 28px;
		height: 28px;
		border-radius: 6px;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 0.9rem;
	}
	.lead-channel-icon.whatsapp { background: #dcfce7; }
	.lead-channel-icon.instagram { background: #fce7f3; }
	.lead-channel-icon.facebook { background: #dbeafe; }

	.lead-info { flex: 1; display: flex; flex-direction: column; }
	.lead-title-row { display: flex; justify-content: space-between; align-items: baseline; }
	.lead-name { font-size: 0.8125rem; font-weight: 700; color: #0f172a; }
	.lead-time { font-size: 0.65rem; color: #94a3b8; }
	.lead-tour { font-size: 0.7rem; color: #64748b; }

	.lead-status-pill {
		font-size: 0.65rem;
		font-weight: 700;
		padding: 0.15rem 0.45rem;
		border-radius: 4px;
	}
	.lead-new { background: #fee2e2; color: #dc2626; }
	.lead-progress { background: #eff6ff; color: #2563eb; }
	.lead-qualified { background: #ecfdf5; color: #059669; }
	.lead-followup { background: #fef3c7; color: #d97706; }

	/* Quick Actions Ribbon */
	.quick-actions-ribbon {
		display: grid;
		grid-template-columns: repeat(10, 1fr);
		gap: 0.5rem;
		background: #ffffff;
		border: 1px solid #e2e8f0;
		border-radius: 12px;
		padding: 0.85rem;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.03);
	}

	@media (max-width: 1200px) {
		.quick-actions-ribbon { grid-template-columns: repeat(5, 1fr); }
	}
	@media (max-width: 640px) {
		.quick-actions-ribbon { grid-template-columns: repeat(2, 1fr); }
	}

	.action-tile {
		background: #f8fafc;
		border: 1px solid #e2e8f0;
		border-radius: 8px;
		padding: 0.65rem 0.35rem;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 0.35rem;
		cursor: pointer;
		transition: all 0.15s ease;
		font-family: inherit;
	}
	.action-tile:hover {
		background: #eff6ff;
		border-color: #bfdbfe;
		transform: translateY(-2px);
	}

	.act-icon { font-size: 1.25rem; }
	.act-text { font-size: 0.7rem; font-weight: 700; color: #0f172a; text-align: center; }

	/* Booking Operations Deck */
	.booking-ops-deck {
		background: #ffffff;
		border: 1px solid #e2e8f0;
		border-radius: 12px;
		padding: 1.5rem;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.03);
	}

	.deck-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 1.25rem;
		padding-bottom: 0.75rem;
		border-bottom: 1px solid #f1f5f9;
	}

	.deck-title { font-size: 1.05rem; font-weight: 800; color: #0f172a; margin: 0; }
	.deck-sub { font-size: 0.75rem; color: #64748b; }
	.deck-actions { display: flex; gap: 0.5rem; }

	.booking-cards-list {
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}

	.ops-booking-card {
		background: #ffffff;
		border: 1px solid #e2e8f0;
		border-radius: 10px;
		overflow: hidden;
		box-shadow: 0 1px 2px rgba(0, 0, 0, 0.02);
	}

	.card-head-strip {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 0.85rem 1.25rem;
		background: #f8fafc;
		border-bottom: 1px solid #e2e8f0;
	}

	.cust-meta { display: flex; align-items: center; gap: 0.75rem; }
	.cust-avatar {
		width: 38px;
		height: 38px;
		border-radius: 8px;
		background: #eff6ff;
		color: #2563eb;
		border: 1px solid #bfdbfe;
		display: flex;
		align-items: center;
		justify-content: center;
		font-weight: 800;
		font-size: 0.85rem;
	}

	.b-cust-name { font-size: 0.95rem; font-weight: 700; color: #0f172a; display: block; }
	.b-ref { font-size: 0.75rem; color: #64748b; }

	.mode-badges { display: flex; gap: 0.5rem; }
	.mode-tag {
		font-size: 0.7rem;
		font-weight: 700;
		padding: 0.2rem 0.55rem;
		border-radius: 6px;
	}
	.mode-family { background: #fef3c7; color: #92400e; }
	.mode-business { background: #eff6ff; color: #1e40af; }
	.status-badge {
		font-size: 0.7rem;
		font-weight: 700;
		padding: 0.2rem 0.55rem;
		border-radius: 6px;
	}

	/* Pillars Deck */
	.pillars-deck {
		display: grid;
		grid-template-columns: repeat(4, 1fr);
		gap: 0.75rem;
		padding: 1rem 1.25rem;
		background: #ffffff;
		border-bottom: 1px solid #f1f5f9;
	}

	@media (max-width: 1024px) {
		.pillars-deck { grid-template-columns: repeat(2, 1fr); }
	}

	.pillar-box {
		background: #f8fafc;
		border: 1px solid #e2e8f0;
		border-radius: 8px;
		padding: 0.75rem;
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}

	.p-title { font-size: 0.65rem; font-weight: 800; color: #64748b; letter-spacing: 0.05em; }
	.p-main { font-size: 0.85rem; font-weight: 700; color: #0f172a; }
	.p-sub { font-size: 0.7rem; color: #475569; }
	.p-spec { font-size: 0.7rem; color: #2563eb; }

	.card-footer-bar {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 0.75rem 1.25rem;
		background: #ffffff;
	}

	.price-chip { display: flex; align-items: baseline; gap: 0.4rem; }
	.pr-label { font-size: 0.75rem; color: #64748b; }
	.pr-val { font-size: 1rem; font-weight: 800; color: #0f172a; }

	.btns-group { display: flex; gap: 0.5rem; }

	.btn {
		font-size: 0.8125rem;
		font-weight: 600;
		padding: 0.5rem 1rem;
		border-radius: 8px;
		cursor: pointer;
		border: 1px solid transparent;
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
		font-family: inherit;
	}
	.btn-sm { padding: 0.35rem 0.75rem; font-size: 0.75rem; }
	.btn-secondary { background: #f1f5f9; color: #334155; }
	.btn-outline { background: #ffffff; border-color: #cbd5e1; color: #475569; }
	.btn-outline:hover { background: #f8fafc; color: #0f172a; }
	.btn-primary { background: #2563eb; color: #ffffff; }
	.btn-primary:hover { background: #1d4ed8; }
	.btn-whatsapp { background: #22c55e; color: #ffffff; font-weight: 700; }
	.btn-whatsapp:hover { background: #16a34a; }

	.text-center { text-align: center; }
</style>
