<script lang="ts">
	import { toast } from '$lib/stores/toast.svelte';
	import { goto } from '$app/navigation';

	// =========================================================================
	// SINGLE 1-PAGE TRIP BOOKING FORM STATE
	// =========================================================================

	// 1. Customer & Origin
	let customerName = $state('Rahul Sharma & Family');
	let customerPhone = $state('+91 98840 11223');
	let customerWhatsapp = $state('+91 98840 11223');
	let customerEmail = $state('rahul.sharma@example.com');
	let originCity = $state('Chennai');
	let originState = $state('Tamil Nadu');

	// 2. Trip Style & Category
	let tripCategory = $state<'family' | 'business' | 'restmode' | 'group' | 'honeymoon'>('family');
	let destinationPackage = $state('Kashmir Valley Paradise (4D/5N)');
	let destinationCity = $state('Srinagar - Gulmarg - Pahalgam');
	let startDate = $state('2026-11-14');
	let endDate = $state('2026-11-19');
	let durationNights = $state(5);
	let adultsCount = $state(2);
	let childrenCount = $state(2);
	let infantsCount = $state(0);

	// 3. On-Ground Car Pickup & Chauffeur
	let vehicleModel = $state('Toyota Innova Crysta AC (White)');
	let vehiclePlate = $state('JK-01-AB-4491');
	let driverName = $state('Ghulam Nabi');
	let driverPhone = $state('+91 94190 22334');
	let pickupLocation = $state('Srinagar International Airport (SXR) - Exit Gate 1');
	let pickupPlacard = $state('SHARMA FAMILY');
	let dropoffLocation = $state('Srinagar International Airport (SXR)');

	// 4. Hotel & Stay Details
	let hotelName = $state('Hotel Grand Kashmir & Betaab Luxury Resort');
	let hotelLocation = $state('Pahalgam Riverfront & Srinagar Boulevard');
	let roomCategory = $state('Executive Suite (Heated) + Extra Bed');
	let roomsCount = $state(2);
	let mealPlan = $state('MAP (Breakfast + Dinner included)');
	let hotelVoucherCode = $state('DMC-SXR-8819');
	let guaranteedCheckIn = $state(true);

	// 5. Flights & Tickets (Domestic / International)
	let includeFlights = $state(true);
	let departureFlight = $state('IndiGo 6E-2144 (MAA 06:15 ➔ SXR 12:30 via DEL)');
	let departurePnr = $state('IND-884920');
	let returnFlight = $state('IndiGo 6E-2145 (SXR 14:00 ➔ MAA 20:15 via DEL)');
	let returnPnr = $state('IND-884921');
	let ticketsIncluded = $state('Gulmarg Gondola Cable Car Phase 1 & 2 (4 Passes) + Shikara Dal Lake Ride');

	// 6. Visa Processing
	let isInternational = $state(false);
	let visaStatus = $state('Not Required (Domestic Circuit)');
	let passportNumbers = $state('A9823411, A9823412');

	// 7. Local Guide & 24/7 Tour Coordinator
	let guideName = $state('Bashir Ahmed (Govt. Licensed #2094)');
	let guidePhone = $state('+91 97970 88990');
	let guideLanguages = $state('Tamil (Conversational), English, Hindi, Kashmiri');
	let coordinatorName = $state('Kavitha R.');
	let coordinatorPhone = $state('+91 98840 11223');
	let coordinatorDepartment = $state('North India Operations Desk');

	// 8. Commercials, Billing & Khata
	let totalPackageCost = $state(165000);
	let advancePaid = $state(65000);
	let balanceDue = $derived(totalPackageCost - advancePaid);
	let paymentMode = $state('Bank IMPS / UPI Transfer');
	let utrReference = $state('UTR-20261022-88492');

	// Form Submission & WhatsApp Kit Modal
	let showSuccessModal = $state(false);
	let generatedBookingId = $state('TRV-2026-084');

	// =========================================================================
	// QUICK TEMPLATE PRESETS (MATCHING USER BUSINESS EXAMPLES)
	// =========================================================================
	function applyPreset(presetType: 'kashmir_chennai' | 'malaysia_madurai' | 'kerala_kanniyakumari' | 'group_departure') {
		if (presetType === 'kashmir_chennai') {
			customerName = 'Rahul Sharma & Family';
			customerPhone = '+91 98840 11223';
			customerWhatsapp = '+91 98840 11223';
			customerEmail = 'rahul.sharma@example.com';
			originCity = 'Chennai';
			originState = 'Tamil Nadu';
			tripCategory = 'family';
			destinationPackage = 'Kashmir Valley Paradise (4D/5N)';
			destinationCity = 'Srinagar - Gulmarg - Pahalgam';
			startDate = '2026-11-14';
			endDate = '2026-11-19';
			durationNights = 5;
			adultsCount = 2;
			childrenCount = 2;
			infantsCount = 0;
			vehicleModel = 'Toyota Innova Crysta AC (White)';
			vehiclePlate = 'JK-01-AB-4491';
			driverName = 'Ghulam Nabi';
			driverPhone = '+91 94190 22334';
			pickupLocation = 'Srinagar Airport Exit Gate 1';
			pickupPlacard = 'SHARMA FAMILY';
			dropoffLocation = 'Srinagar Airport';
			hotelName = 'Hotel Grand Kashmir & Betaab Luxury Resort';
			hotelLocation = 'Pahalgam & Srinagar Boulevard';
			roomCategory = 'Executive Suite (Heated) + Extra Bed';
			roomsCount = 2;
			mealPlan = 'MAP (Breakfast + Dinner included)';
			hotelVoucherCode = 'DMC-SXR-8819';
			includeFlights = true;
			departureFlight = 'IndiGo 6E-2144 (MAA 06:15 ➔ SXR 12:30)';
			departurePnr = 'IND-884920';
			returnFlight = 'IndiGo 6E-2145 (SXR 14:00 ➔ MAA 20:15)';
			returnPnr = 'IND-884921';
			ticketsIncluded = 'Gulmarg Gondola Phase 1 & 2 Passes + Shikara Boat Ride';
			isInternational = false;
			visaStatus = 'Not Required (Domestic Circuit)';
			guideName = 'Bashir Ahmed (Govt. Licensed #2094)';
			guidePhone = '+91 97970 88990';
			guideLanguages = 'Tamil (Conversational), English, Hindi';
			coordinatorName = 'Kavitha R.';
			coordinatorPhone = '+91 98840 11223';
			totalPackageCost = 165000;
			advancePaid = 65000;
			paymentMode = 'UPI / GPay';
			utrReference = 'UPI-20261022-77889';
			toast.success('Loaded: Chennai Customer ➔ Kashmir Family Trip (4N/5D)');
		} else if (presetType === 'malaysia_madurai') {
			customerName = 'Karthik Subramanian';
			customerPhone = '+91 94441 55667';
			customerWhatsapp = '+91 94441 55667';
			customerEmail = 'karthik.sub@techmadurai.in';
			originCity = 'Madurai';
			originState = 'Tamil Nadu';
			tripCategory = 'business';
			destinationPackage = 'Malaysia Business & Twin Towers Expo';
			destinationCity = 'Kuala Lumpur - Putrajaya';
			startDate = '2026-11-20';
			endDate = '2026-11-24';
			durationNights = 4;
			adultsCount = 6;
			childrenCount = 0;
			infantsCount = 0;
			vehicleModel = 'Mercedes-Benz Executive Sprinter Coach';
			vehiclePlate = 'WXY 8892 (KL)';
			driverName = 'Tan Wei Meng';
			driverPhone = '+60 16 987 6543';
			pickupLocation = 'KLIA Terminal 1 International Arrivals Gate 3';
			pickupPlacard = 'TECH MADURAI DELEGATION';
			dropoffLocation = 'KLIA Terminal 1';
			hotelName = 'Hilton Kuala Lumpur City Centre';
			hotelLocation = 'KL Sentral / KLCC';
			roomCategory = 'Grand Executive Twin Rooms';
			roomsCount = 3;
			mealPlan = 'CP (Buffet Breakfast included)';
			hotelVoucherCode = 'DMC-KUL-4402';
			includeFlights = true;
			departureFlight = 'AirAsia AK-24 (TRZ 09:30 ➔ KUL 15:45)';
			departurePnr = 'AK-559124';
			returnFlight = 'AirAsia AK-25 (KUL 17:00 ➔ TRZ 18:30)';
			returnPnr = 'AK-559125';
			ticketsIncluded = 'KLCC Petronas Twin Towers Skybridge + Convention Badges';
			isInternational = true;
			visaStatus = 'Malaysia eNTRI / e-Visa Approved (Stamped)';
			passportNumbers = 'N4489211, N4489212, N4489213, N4489214, N4489215, N4489216';
			guideName = 'Govindasamy (Licensed Tourist Guide #MY881)';
			guidePhone = '+60 16 987 6543';
			guideLanguages = 'Tamil, English, Malay';
			coordinatorName = 'Muthuvel S.';
			coordinatorPhone = '+91 98402 33445';
			totalPackageCost = 320000;
			advancePaid = 150000;
			paymentMode = 'Corporate Wire / RTGS';
			utrReference = 'RTGS-20261021-99881';
			toast.success('Loaded: Madurai Customer ➔ Malaysia Business Trip');
		} else if (presetType === 'kerala_kanniyakumari') {
			customerName = 'Ananya & Vignesh';
			customerPhone = '+91 97890 33445';
			customerWhatsapp = '+91 97890 33445';
			customerEmail = 'vignesh.a@gmail.com';
			originCity = 'Kanniyakumari';
			originState = 'Tamil Nadu';
			tripCategory = 'restmode';
			destinationPackage = 'Kerala Backwaters & Munnar Tea Retreat';
			destinationCity = 'Alleppey - Munnar - Cochin';
			startDate = '2026-11-28';
			endDate = '2026-12-02';
			durationNights = 4;
			adultsCount = 2;
			childrenCount = 0;
			infantsCount = 0;
			vehicleModel = 'Maruti Dzire AC Prime Sedan';
			vehiclePlate = 'KL-07-CD-9981';
			driverName = 'Santhosh Kumar';
			driverPhone = '+91 98470 11992';
			pickupLocation = 'Nagercoil Junction Railway Station / Kanniyakumari Doorstep';
			pickupPlacard = 'VIGNESH & ANANYA';
			dropoffLocation = 'Cochin International Airport (COK)';
			hotelName = 'Lake Palace Luxury Houseboat & Munnar Tea County';
			hotelLocation = 'Vembanad Lake Alleppey & Munnar Hills';
			roomCategory = 'Private 1-Bedroom AC Deluxe Houseboat + Tea View Suite';
			roomsCount = 1;
			mealPlan = 'AP (All Meals: Breakfast + Traditional Kerala Lunch + Dinner)';
			hotelVoucherCode = 'DMC-KER-1104';
			includeFlights = false;
			departureFlight = 'Surface Drive from Kanniyakumari';
			departurePnr = 'N/A (Cab)';
			returnFlight = 'Surface / Drop Cochin';
			returnPnr = 'N/A';
			ticketsIncluded = 'Alleppey Backwater Shikara Cruise + Eravikulam National Park Pass';
			isInternational = false;
			visaStatus = 'Not Required (Domestic)';
			guideName = 'Biju Varghese (Local Kerala Guide)';
			guidePhone = '+91 98470 55443';
			guideLanguages = 'Tamil, Malayalam, English';
			coordinatorName = 'Deepa Lakshmi';
			coordinatorPhone = '+91 97890 11224';
			totalPackageCost = 98000;
			advancePaid = 40000;
			paymentMode = 'UPI / PhonePe';
			utrReference = 'UPI-20261020-44556';
			toast.success('Loaded: Kanniyakumari Customer ➔ Kerala Restmode Trip');
		} else if (presetType === 'group_departure') {
			customerName = 'Dr. S. Rajendran & Medical Club';
			customerPhone = '+91 94220 88990';
			customerWhatsapp = '+91 94220 88990';
			customerEmail = 'dr.rajendran@apollo.org';
			originCity = 'Coimbatore';
			originState = 'Tamil Nadu';
			tripCategory = 'group';
			destinationPackage = 'Dubai Desert & City Spectacular Group Tour';
			destinationCity = 'Dubai - Abu Dhabi';
			startDate = '2026-12-05';
			endDate = '2026-12-10';
			durationNights = 5;
			adultsCount = 8;
			childrenCount = 0;
			infantsCount = 0;
			vehicleModel = 'Force Urbania / Tempo Traveller 17-Seater AC';
			vehiclePlate = 'DXB-55912';
			driverName = 'Tariq Al-Mansoor';
			driverPhone = '+971 50 123 4567';
			pickupLocation = 'Dubai International Airport (DXB) Terminal 3';
			pickupPlacard = 'COIMBATORE DOCTORS GROUP';
			dropoffLocation = 'DXB Terminal 3';
			hotelName = 'JW Marriott Marquis Hotel Dubai';
			hotelLocation = 'Business Bay, Dubai';
			roomCategory = 'Deluxe City View Rooms';
			roomsCount = 4;
			mealPlan = 'MAP (Breakfast + Dinner included)';
			hotelVoucherCode = 'DMC-DXB-9988';
			includeFlights = true;
			departureFlight = 'Air Arabia G9-412 (CJB 04:30 ➔ SHJ 07:15)';
			departurePnr = 'G9-441029';
			returnFlight = 'Air Arabia G9-413 (SHJ 21:00 ➔ CJB 02:45)';
			returnPnr = 'G9-441030';
			ticketsIncluded = 'Burj Khalifa 124th Floor + Desert Safari 4x4 + Marina Dhow Cruise';
			isInternational = true;
			visaStatus = 'UAE Tourist 30-Day Express Visas (8 Pax Approved)';
			passportNumbers = 'K8819201 through K8819208';
			guideName = 'Mustafa Kamal (Govt. Licensed Tour Guide)';
			guidePhone = '+971 55 987 6543';
			guideLanguages = 'Tamil, English, Arabic, Hindi';
			coordinatorName = 'Priya Menon';
			coordinatorPhone = '+91 98840 11223';
			totalPackageCost = 680000;
			advancePaid = 300000;
			paymentMode = 'Bank IMPS Transfer';
			utrReference = 'IMPS-20261018-99001';
			toast.success('Loaded: Coimbatore Doctors Group Tour ➔ Dubai');
		}
	}

	function handleConfirmBooking() {
		if (!customerName || !destinationPackage) {
			toast.error('Please enter customer name and destination');
			return;
		}

		generatedBookingId = `TRV-${Math.floor(100000 + Math.random() * 900000)}`;
		showSuccessModal = true;
		toast.success(`Booking ${generatedBookingId} confirmed and ground logistics assigned!`);
	}

	function copyWhatsAppMessage() {
		const text = `✈️ *TOUR BOOKING CONFIRMATION & GROUND PASS*
━━━━━━━━━━━━━━━━━━━━━━━━━━
📌 *Booking ID:* ${generatedBookingId}
👤 *Lead Guest:* ${customerName} (${originCity})
🧳 *Package:* ${destinationPackage}
📅 *Dates:* ${startDate} to ${endDate} (${durationNights} Nights)
👥 *Guests:* ${adultsCount} Adults, ${childrenCount} Children

🚗 *ON-GROUND CHAUFFEUR & CAB:*
• *Vehicle:* ${vehicleModel} (${vehiclePlate})
• *Chauffeur:* ${driverName} (📞 ${driverPhone})
• *Airport Pickup:* ${pickupLocation}
• *Placard:* "${pickupPlacard}"

🏨 *HOTEL / RESORT VOUCHER:*
• *Hotel:* ${hotelName}
• *Room:* ${roomCategory} (${roomsCount} Rooms)
• *Meal Plan:* ${mealPlan}
• *Voucher Ref:* ${hotelVoucherCode} (Guaranteed Check-in)

🧭 *LOCAL LICENSED GUIDE:*
• *Guide:* ${guideName} (📞 ${guidePhone})
• *Languages:* ${guideLanguages}

🆘 *24/7 TOUR COORDINATOR:*
• *Coordinator:* ${coordinatorName} (📞 ${coordinatorPhone})

💰 *FINANCIAL SUMMARY:*
• *Total Tour Value:* ₹${totalPackageCost.toLocaleString('en-IN')}
• *Advance Paid:* ₹${advancePaid.toLocaleString('en-IN')} (${paymentMode})
• *Balance Due:* ₹${balanceDue.toLocaleString('en-IN')}

Have a wonderful and memorable trip with *Travel & Tours OS*!`;

		if (typeof navigator !== 'undefined' && navigator.clipboard) {
			navigator.clipboard.writeText(text);
			toast.success('WhatsApp Travel Kit text copied to clipboard!');
		}
	}

	function formatCurrency(val: number) {
		return new Intl.NumberFormat('en-IN', { style: 'currency', currency: 'INR', maximumFractionDigits: 0 }).format(val);
	}
</script>

<svelte:head>
	<title>1-Page Trip Booking Desk | Travel &amp; Tours OS</title>
</svelte:head>

<div class="book-trip-container">
	<!-- Top Masthead Bar -->
	<header class="book-top-masthead">
		<div class="masthead-left">
			<a href="/travel" class="back-link">‹ Back to Dashboard</a>
			<div class="title-wrap">
				<span class="page-badge">⚡ UNIFIED 1-PAGE OPERATOR DESK</span>
				<h1 class="page-title">Book a Trip &amp; Ground Dispatch</h1>
				<p class="page-sub">
					Fill customer details, car pickup, resort stay, flights, local guide &amp; tickets — all on one page without popup modals.
				</p>
			</div>
		</div>

		<!-- Quick 1-Click Templates -->
		<div class="preset-templates-bar">
			<span class="preset-label">1-Click Presets:</span>
			<button type="button" class="preset-btn kashmir" onclick={() => applyPreset('kashmir_chennai')}>
				🏔️ Chennai ➔ Kashmir (Family)
			</button>
			<button type="button" class="preset-btn malaysia" onclick={() => applyPreset('malaysia_madurai')}>
				🏙️ Madurai ➔ Malaysia (Business)
			</button>
			<button type="button" class="preset-btn kerala" onclick={() => applyPreset('kerala_kanniyakumari')}>
				🌴 Kanniyakumari ➔ Kerala (Restmode)
			</button>
			<button type="button" class="preset-btn dubai" onclick={() => applyPreset('group_departure')}>
				👥 Group Tour (Dubai)
			</button>
		</div>
	</header>

	<!-- Main 1-Page Form Layout -->
	<form class="book-main-form" onsubmit={(e) => { e.preventDefault(); handleConfirmBooking(); }}>
		
		<!-- =====================================================================
		     SECTION 1: CUSTOMER & TRIP CATEGORY
		     ===================================================================== -->
		<section class="form-section-card">
			<div class="section-header">
				<div class="sec-icon blue">👤</div>
				<div>
					<h3 class="sec-title">1. Customer Information &amp; Trip Category</h3>
					<span class="sec-sub">Lead guest contact and tour classification</span>
				</div>
			</div>

			<!-- Category Buttons -->
			<div class="category-selector-row">
				<button 
					type="button" 
					class="cat-chip {tripCategory === 'family' ? 'active' : ''}" 
					onclick={() => (tripCategory = 'family')}
				>
					👨‍👩‍👧‍👦 Family Tour
				</button>
				<button 
					type="button" 
					class="cat-chip {tripCategory === 'business' ? 'active' : ''}" 
					onclick={() => (tripCategory = 'business')}
				>
					💼 Business Trip
				</button>
				<button 
					type="button" 
					class="cat-chip {tripCategory === 'restmode' ? 'active' : ''}" 
					onclick={() => (tripCategory = 'restmode')}
				>
					🌴 Restmode / Leisure
				</button>
				<button 
					type="button" 
					class="cat-chip {tripCategory === 'group' ? 'active' : ''}" 
					onclick={() => (tripCategory = 'group')}
				>
					👥 Group Tour
				</button>
				<button 
					type="button" 
					class="cat-chip {tripCategory === 'honeymoon' ? 'active' : ''}" 
					onclick={() => (tripCategory = 'honeymoon')}
				>
					💍 Honeymoon
				</button>
			</div>

			<div class="fields-grid-3">
				<div class="input-block">
					<label class="input-label" for="cust-name">Customer / Family Name *</label>
					<input id="cust-name" type="text" bind:value={customerName} class="text-input" placeholder="e.g. Rahul Sharma & Family" required />
				</div>
				<div class="input-block">
					<label class="input-label" for="cust-phone">WhatsApp / Phone Number *</label>
					<input id="cust-phone" type="text" bind:value={customerWhatsapp} class="text-input font-mono" placeholder="+91 98840 11223" required />
				</div>
				<div class="input-block">
					<label class="input-label" for="cust-email">Email Address</label>
					<input id="cust-email" type="email" bind:value={customerEmail} class="text-input" placeholder="guest@example.com" />
				</div>
			</div>

			<div class="fields-grid-3" style="margin-top: 1rem;">
				<div class="input-block">
					<label class="input-label" for="cust-city">Origin City (Pickup Source)</label>
					<input id="cust-city" type="text" bind:value={originCity} class="text-input" placeholder="Chennai / Madurai / Kanniyakumari" />
				</div>
				<div class="input-block">
					<label class="input-label" for="cust-dest">Destination Package *</label>
					<input id="cust-dest" type="text" bind:value={destinationPackage} class="text-input font-bold text-blue" placeholder="e.g. Kashmir Valley Paradise" required />
				</div>
				<div class="input-block">
					<label class="input-label" for="cust-dates">Travel Dates</label>
					<div class="dates-flex-row">
						<input type="date" bind:value={startDate} class="text-input" />
						<span class="to-text">to</span>
						<input type="date" bind:value={endDate} class="text-input" />
					</div>
				</div>
			</div>

			<div class="fields-grid-4" style="margin-top: 1rem;">
				<div class="input-block">
					<label class="input-label" for="pax-adults">Adults (12+ yrs)</label>
					<input id="pax-adults" type="number" bind:value={adultsCount} min="1" class="text-input font-mono" />
				</div>
				<div class="input-block">
					<label class="input-label" for="pax-children">Children (2-11 yrs)</label>
					<input id="pax-children" type="number" bind:value={childrenCount} min="0" class="text-input font-mono" />
				</div>
				<div class="input-block">
					<label class="input-label" for="pax-nights">Duration (Nights)</label>
					<input id="pax-nights" type="number" bind:value={durationNights} min="1" class="text-input font-mono" />
				</div>
				<div class="input-block">
					<label class="input-label">Total Travelling Pax</label>
					<div class="pax-badge-display font-mono">{adultsCount + childrenCount + infantsCount} Passengers</div>
				</div>
			</div>
		</section>

		<!-- =====================================================================
		     SECTION 2: CAR PICKUP, CHAUFFEUR & VEHICLE DISPATCH
		     ===================================================================== -->
		<section class="form-section-card">
			<div class="section-header">
				<div class="sec-icon red">🚗</div>
				<div>
					<h3 class="sec-title">2. Car Pickup, Chauffeur &amp; Ground Fleet Details</h3>
					<span class="sec-sub">Driver phone, vehicle registration plate, and airport pickup placard</span>
				</div>
			</div>

			<div class="fields-grid-3">
				<div class="input-block">
					<label class="input-label" for="veh-model">Assigned Vehicle Model</label>
					<select id="veh-model" bind:value={vehicleModel} class="select-input">
						<option value="Toyota Innova Crysta AC (White)">Toyota Innova Crysta AC (White)</option>
						<option value="Maruti Dzire AC Prime Sedan">Maruti Dzire AC Prime Sedan</option>
						<option value="Force Urbania / Tempo Traveller 17-Seater AC">Force Urbania / Tempo Traveller 17-Seater AC</option>
						<option value="Mercedes-Benz Executive Sprinter Coach">Mercedes-Benz Executive Sprinter Coach</option>
						<option value="Mahindra Scorpio-N 4x4 (Snow terrain)">Mahindra Scorpio-N 4x4 (Snow terrain)</option>
					</select>
				</div>
				<div class="input-block">
					<label class="input-label" for="veh-plate">Vehicle Plate / Registration Number</label>
					<input id="veh-plate" type="text" bind:value={vehiclePlate} class="text-input font-mono" placeholder="JK-01-AB-4491" />
				</div>
				<div class="input-block">
					<label class="input-label" for="drv-name">Assigned Chauffeur / Driver Name</label>
					<input id="drv-name" type="text" bind:value={driverName} class="text-input" placeholder="e.g. Ghulam Nabi" />
				</div>
			</div>

			<div class="fields-grid-3" style="margin-top: 1rem;">
				<div class="input-block">
					<label class="input-label" for="drv-phone">Driver Direct Contact (Mobile / WhatsApp)</label>
					<input id="drv-phone" type="text" bind:value={driverPhone} class="text-input font-mono font-bold" placeholder="+91 94190 22334" />
				</div>
				<div class="input-block">
					<label class="input-label" for="pickup-loc">Airport / Station Pickup Point</label>
					<input id="pickup-loc" type="text" bind:value={pickupLocation} class="text-input" placeholder="Srinagar Airport Exit Gate 1" />
				</div>
				<div class="input-block">
					<label class="input-label" for="pickup-plac">Placard Signboard (Driver will hold this name)</label>
					<input id="pickup-plac" type="text" bind:value={pickupPlacard} class="text-input font-bold text-emerald" placeholder="SHARMA FAMILY" />
				</div>
			</div>
		</section>

		<!-- =====================================================================
		     SECTION 3: HOTEL & RESORT STAY
		     ===================================================================== -->
		<section class="form-section-card">
			<div class="section-header">
				<div class="sec-icon amber">🏨</div>
				<div>
					<h3 class="sec-title">3. Hotel &amp; Resort Stay Logistics</h3>
					<span class="sec-sub">Room allocation, meal plans, voucher confirmation code</span>
				</div>
			</div>

			<div class="fields-grid-3">
				<div class="input-block">
					<label class="input-label" for="htl-name">Confirmed Resort / Hotel Name</label>
					<input id="htl-name" type="text" bind:value={hotelName} class="text-input font-semibold" placeholder="e.g. Hotel Grand Kashmir & Betaab Luxury Resort" />
				</div>
				<div class="input-block">
					<label class="input-label" for="htl-loc">Location / Area</label>
					<input id="htl-loc" type="text" bind:value={hotelLocation} class="text-input" placeholder="Pahalgam & Srinagar" />
				</div>
				<div class="input-block">
					<label class="input-label" for="htl-vouch">Hotel Voucher / DMC Code</label>
					<input id="htl-vouch" type="text" bind:value={hotelVoucherCode} class="text-input font-mono font-bold" placeholder="DMC-SXR-8819" />
				</div>
			</div>

			<div class="fields-grid-3" style="margin-top: 1rem;">
				<div class="input-block">
					<label class="input-label" for="htl-room">Room Category &amp; Configuration</label>
					<input id="htl-room" type="text" bind:value={roomCategory} class="text-input" placeholder="Executive Suite + Extra Bed" />
				</div>
				<div class="input-block">
					<label class="input-label" for="htl-meal">Meal Plan</label>
					<select id="htl-meal" bind:value={mealPlan} class="select-input">
						<option value="MAP (Breakfast + Dinner included)">MAP (Breakfast + Dinner included)</option>
						<option value="CP (Breakfast only)">CP (Breakfast only)</option>
						<option value="AP (All Meals: Breakfast + Lunch + Dinner)">AP (All Meals: Breakfast + Lunch + Dinner)</option>
						<option value="EP (Room Only)">EP (Room Only)</option>
					</select>
				</div>
				<div class="input-block">
					<label class="input-label">Guaranteed Check-in Status</label>
					<label class="checkbox-row">
						<input type="checkbox" bind:checked={guaranteedCheckIn} />
						<span>Guaranteed Check-in (Voucher pre-paid)</span>
					</label>
				</div>
			</div>
		</section>

		<!-- =====================================================================
		     SECTION 4: FLIGHTS, TICKETS & VISA PROCESS
		     ===================================================================== -->
		<section class="form-section-card">
			<div class="section-header">
				<div class="sec-icon purple">✈️</div>
				<div>
					<h3 class="sec-title">4. Flights, Tickets &amp; Visa Processing</h3>
					<span class="sec-sub">Airlines, PNR holds, sightseeing passes, international visa status</span>
				</div>
			</div>

			<div class="fields-grid-2">
				<div class="input-block">
					<label class="input-label" for="flt-dep">Departure Flight &amp; PNR</label>
					<div class="input-with-pnr">
						<input id="flt-dep" type="text" bind:value={departureFlight} class="text-input" placeholder="IndiGo 6E-2144 (MAA ➔ SXR)" />
						<input type="text" bind:value={departurePnr} class="text-input pnr-box font-mono" placeholder="PNR" />
					</div>
				</div>
				<div class="input-block">
					<label class="input-label" for="flt-ret">Return Flight &amp; PNR</label>
					<div class="input-with-pnr">
						<input id="flt-ret" type="text" bind:value={returnFlight} class="text-input" placeholder="IndiGo 6E-2145 (SXR ➔ MAA)" />
						<input type="text" bind:value={returnPnr} class="text-input pnr-box font-mono" placeholder="PNR" />
					</div>
				</div>
			</div>

			<div class="fields-grid-2" style="margin-top: 1rem;">
				<div class="input-block">
					<label class="input-label" for="tkt-inc">Sightseeing &amp; Activity Passes Included</label>
					<input id="tkt-inc" type="text" bind:value={ticketsIncluded} class="text-input" placeholder="Gulmarg Gondola Phase 1 & 2 Passes + Shikara Ride" />
				</div>
				<div class="input-block">
					<label class="input-label" for="visa-st">Visa Processing Status</label>
					<input id="visa-st" type="text" bind:value={visaStatus} class="text-input font-mono" placeholder="Not Required or e-Visa Approved" />
				</div>
			</div>
		</section>

		<!-- =====================================================================
		     SECTION 5: LOCAL GUIDE & 24/7 COORDINATOR
		     ===================================================================== -->
		<section class="form-section-card">
			<div class="section-header">
				<div class="sec-icon green">🧭</div>
				<div>
					<h3 class="sec-title">5. Local Area Guide &amp; 24/7 Emergency Support</h3>
					<span class="sec-sub">On-ground destination guide and dedicated operations manager</span>
				</div>
			</div>

			<div class="fields-grid-3">
				<div class="input-block">
					<label class="input-label" for="gd-name">Local Guide Name</label>
					<input id="gd-name" type="text" bind:value={guideName} class="text-input font-semibold" placeholder="Bashir Ahmed (Govt. Licensed #2094)" />
				</div>
				<div class="input-block">
					<label class="input-label" for="gd-phone">Guide Phone Number</label>
					<input id="gd-phone" type="text" bind:value={guidePhone} class="text-input font-mono font-bold" placeholder="+91 97970 88990" />
				</div>
				<div class="input-block">
					<label class="input-label" for="gd-lang">Guide Languages Spoken</label>
					<input id="gd-lang" type="text" bind:value={guideLanguages} class="text-input" placeholder="Tamil, English, Hindi" />
				</div>
			</div>

			<div class="fields-grid-2" style="margin-top: 1rem;">
				<div class="input-block">
					<label class="input-label" for="co-name">24/7 Agency Operations Coordinator</label>
					<input id="co-name" type="text" bind:value={coordinatorName} class="text-input" placeholder="Kavitha R." />
				</div>
				<div class="input-block">
					<label class="input-label" for="co-phone">Emergency 24/7 Helpline</label>
					<input id="co-phone" type="text" bind:value={coordinatorPhone} class="text-input font-mono font-bold text-blue" placeholder="+91 98840 11223" />
				</div>
			</div>
		</section>

		<!-- =====================================================================
		     SECTION 6: BILLING, ADVANCE PAYMENT & CONFIRMATION
		     ===================================================================== -->
		<section class="form-section-card billing-card">
			<div class="section-header">
				<div class="sec-icon green">💳</div>
				<div>
					<h3 class="sec-title">6. Commercial Billing &amp; Advance Payment Khata</h3>
					<span class="sec-sub">Record package price, advance received, and payment transaction reference</span>
				</div>
			</div>

			<div class="billing-summary-grid">
				<div class="input-block">
					<label class="input-label" for="bill-tot">Total Package Price (₹) *</label>
					<input id="bill-tot" type="number" bind:value={totalPackageCost} class="text-input font-mono text-lg font-bold" required />
				</div>
				<div class="input-block">
					<label class="input-label" for="bill-adv">Advance Amount Received (₹) *</label>
					<input id="bill-adv" type="number" bind:value={advancePaid} class="text-input font-mono text-lg font-bold text-emerald" required />
				</div>
				<div class="input-block">
					<label class="input-label">Balance Outstanding (₹)</label>
					<div class="balance-display font-mono text-lg font-bold">
						{formatCurrency(balanceDue)}
					</div>
				</div>
			</div>

			<div class="fields-grid-2" style="margin-top: 1rem;">
				<div class="input-block">
					<label class="input-label" for="bill-mod">Payment Mode</label>
					<select id="bill-mod" bind:value={paymentMode} class="select-input">
						<option value="Bank IMPS / UPI Transfer">Bank IMPS / UPI Transfer</option>
						<option value="UPI / GPay">UPI / GPay</option>
						<option value="Corporate Wire / RTGS">Corporate Wire / RTGS</option>
						<option value="Cash Counter">Cash Counter</option>
						<option value="Credit Card">Credit Card</option>
					</select>
				</div>
				<div class="input-block">
					<label class="input-label" for="bill-utr">Transaction Reference / UTR Number</label>
					<input id="bill-utr" type="text" bind:value={utrReference} class="text-input font-mono" placeholder="UTR-20261022-88492" />
				</div>
			</div>

			<!-- Big Action Submission Strip -->
			<div class="submit-action-strip">
				<div class="summary-reminder">
					<span>✅ Ready to confirm:</span>
					<strong>{customerName}</strong> ({destinationPackage}) • <strong>{adultsCount + childrenCount} Pax</strong>
				</div>
				<button type="submit" class="btn-confirm-booking">
					🚀 Confirm Trip &amp; Dispatch WhatsApp Travel Kit
				</button>
			</div>
		</section>
	</form>
</div>

<!-- =========================================================================
     SUCCESS & WHATSAPP TRAVEL KIT MODAL
     ========================================================================= -->
{#if showSuccessModal}
	<div class="modal-backdrop" onclick={() => (showSuccessModal = false)}>
		<div class="modal-card" onclick={(e) => e.stopPropagation()} role="dialog">
			<div class="modal-header-green">
				<div class="succ-icon">🎉</div>
				<div>
					<h3 class="succ-title">Trip Booked Successfully!</h3>
					<span class="succ-booking-id font-mono">Booking Ref: {generatedBookingId}</span>
				</div>
				<button type="button" class="close-x-btn" onclick={() => (showSuccessModal = false)}>✕</button>
			</div>

			<div class="modal-body">
				<p class="succ-desc">
					All ground logistics have been assigned. Your customer <strong>{customerName}</strong> is confirmed for <strong>{destinationPackage}</strong>.
				</p>

				<!-- Preview WhatsApp Message -->
				<div class="whatsapp-preview-card">
					<div class="wa-header">
						<span>📱 WhatsApp Travel Kit Ready to Send:</span>
						<button type="button" class="copy-chip-btn" onclick={copyWhatsAppMessage}>
							📋 Copy Message
						</button>
					</div>
					<div class="wa-message-content font-mono">
						✈️ <strong>TOUR BOOKING CONFIRMATION &amp; GROUND PASS</strong><br />
						━━━━━━━━━━━━━━━━━━━━━━━━━━<br />
						📌 <strong>Booking ID:</strong> {generatedBookingId}<br />
						👤 <strong>Lead Guest:</strong> {customerName} ({originCity})<br />
						🧳 <strong>Package:</strong> {destinationPackage}<br />
						📅 <strong>Dates:</strong> {startDate} to {endDate} ({durationNights} Nights)<br />
						👥 <strong>Guests:</strong> {adultsCount} Adults, {childrenCount} Children<br /><br />
						🚗 <strong>CHAUFFEUR &amp; CAB:</strong><br />
						• Vehicle: {vehicleModel} ({vehiclePlate})<br />
						• Chauffeur: {driverName} (📞 {driverPhone})<br />
						• Airport Pickup: {pickupLocation}<br />
						• Placard: "{pickupPlacard}"<br /><br />
						🏨 <strong>RESORT STAY:</strong><br />
						• Hotel: {hotelName}<br />
						• Room: {roomCategory}<br />
						• Voucher: {hotelVoucherCode} (Guaranteed Check-in)<br /><br />
						🧭 <strong>LOCAL GUIDE:</strong><br />
						• Guide: {guideName} (📞 {guidePhone})<br />
						• Languages: {guideLanguages}<br /><br />
						💰 <strong>PAYMENT:</strong><br />
						• Total: {formatCurrency(totalPackageCost)} (Advance: {formatCurrency(advancePaid)})<br />
						• Balance Due: {formatCurrency(balanceDue)}
					</div>
				</div>
			</div>

			<div class="modal-footer">
				<a 
					href={`https://wa.me/${customerWhatsapp.replace(/[^0-9]/g, '')}?text=${encodeURIComponent(`Hello ${customerName}, here is your Travel & Tours OS confirmed digital voucher and chauffeur pass for ${destinationPackage}! Booking ID: ${generatedBookingId}`)}`}
					target="_blank" 
					class="btn-send-whatsapp"
				>
					📱 Open in WhatsApp Web / App
				</a>
				<button type="button" class="btn-go-dashboard" onclick={() => goto('/travel')}>
					Go to Dashboard ➔
				</button>
			</div>
		</div>
	</div>
{/if}

<style>
	/* ==========================================================================
	   1-PAGE TRIP BOOKING DESK STYLING
	   ========================================================================== */

	.book-trip-container {
		max-width: 1300px;
		margin: 0 auto;
		padding: 1.5rem;
		display: flex;
		flex-direction: column;
		gap: 1.5rem;
		font-family: var(--font-sans, system-ui, -apple-system, sans-serif);
		color: #0f172a;
	}

	.book-top-masthead {
		background: #ffffff;
		border: 1px solid #e2e8f0;
		border-radius: 12px;
		padding: 1.25rem 1.5rem;
		box-shadow: 0 1px 3px rgba(0,0,0,0.04);
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}

	.back-link {
		font-size: 0.8125rem;
		font-weight: 700;
		color: #2563eb;
		text-decoration: none;
	}

	.page-badge {
		font-size: 0.65rem;
		font-weight: 800;
		color: #ffffff;
		background: #2563eb;
		padding: 2px 8px;
		border-radius: 4px;
		letter-spacing: 0.05em;
		display: inline-block;
		margin-bottom: 4px;
	}

	.page-title {
		font-size: 1.6rem;
		font-weight: 800;
		margin: 0;
		color: #0f172a;
		letter-spacing: -0.02em;
	}

	.page-sub {
		font-size: 0.85rem;
		color: #64748b;
		margin: 4px 0 0 0;
	}

	/* Preset Buttons Bar */
	.preset-templates-bar {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		flex-wrap: wrap;
		padding-top: 0.75rem;
		border-top: 1px solid #f1f5f9;
	}

	.preset-label {
		font-size: 0.75rem;
		font-weight: 700;
		color: #64748b;
		text-transform: uppercase;
	}

	.preset-btn {
		background: #f8fafc;
		border: 1px solid #cbd5e1;
		padding: 6px 12px;
		border-radius: 6px;
		font-size: 0.8rem;
		font-weight: 700;
		color: #334155;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.preset-btn:hover {
		background: #eff6ff;
		border-color: #93c5fd;
		color: #1e40af;
		transform: translateY(-1px);
	}

	/* Form Section Cards */
	.form-section-card {
		background: #ffffff;
		border: 1px solid #e2e8f0;
		border-radius: 12px;
		padding: 1.25rem 1.5rem;
		box-shadow: 0 1px 3px rgba(0,0,0,0.03);
		display: flex;
		flex-direction: column;
		gap: 0.85rem;
	}

	.section-header {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		border-bottom: 1px solid #f1f5f9;
		padding-bottom: 0.75rem;
	}

	.sec-icon {
		width: 38px;
		height: 38px;
		border-radius: 10px;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 1.25rem;
	}
	.sec-icon.blue { background: #eff6ff; }
	.sec-icon.red { background: #fef2f2; }
	.sec-icon.amber { background: #fffbeb; }
	.sec-icon.purple { background: #faf5ff; }
	.sec-icon.green { background: #f0fdf4; }

	.sec-title {
		font-size: 1.05rem;
		font-weight: 800;
		color: #0f172a;
		margin: 0;
	}

	.sec-sub {
		font-size: 0.75rem;
		color: #64748b;
	}

	/* Category Selector */
	.category-selector-row {
		display: flex;
		gap: 0.5rem;
		flex-wrap: wrap;
		margin-bottom: 0.5rem;
	}

	.cat-chip {
		background: #f8fafc;
		border: 1px solid #e2e8f0;
		padding: 6px 14px;
		border-radius: 20px;
		font-size: 0.8125rem;
		font-weight: 700;
		color: #475569;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.cat-chip.active {
		background: #2563eb;
		color: #ffffff;
		border-color: #2563eb;
		box-shadow: 0 2px 6px rgba(37, 99, 235, 0.25);
	}

	/* Grid Layouts */
	.fields-grid-3 {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 1rem;
	}

	.fields-grid-2 {
		display: grid;
		grid-template-columns: repeat(2, 1fr);
		gap: 1rem;
	}

	.fields-grid-4 {
		display: grid;
		grid-template-columns: repeat(4, 1fr);
		gap: 1rem;
	}

	.input-block {
		display: flex;
		flex-direction: column;
		gap: 0.35rem;
	}

	.input-label {
		font-size: 0.75rem;
		font-weight: 700;
		color: #475569;
	}

	.text-input, .select-input {
		width: 100%;
		padding: 8px 12px;
		border-radius: 8px;
		border: 1px solid #cbd5e1;
		font-size: 0.875rem;
		color: #0f172a;
		background: #ffffff;
		outline: none;
		box-sizing: border-box;
		transition: border-color 0.15s ease;
	}

	.text-input:focus, .select-input:focus {
		border-color: #2563eb;
		box-shadow: 0 0 0 2px rgba(37, 99, 235, 0.15);
	}

	.dates-flex-row {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.to-text {
		font-size: 0.75rem;
		color: #94a3b8;
	}

	.pax-badge-display {
		padding: 8px 12px;
		background: #f1f5f9;
		border-radius: 8px;
		border: 1px solid #cbd5e1;
		font-size: 0.875rem;
		font-weight: 700;
		color: #1e40af;
	}

	.input-with-pnr {
		display: flex;
		gap: 0.5rem;
	}

	.pnr-box {
		width: 110px;
		font-weight: 700;
		text-align: center;
		text-transform: uppercase;
	}

	.checkbox-row {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 0.8125rem;
		font-weight: 600;
		color: #334155;
		margin-top: 6px;
		cursor: pointer;
	}

	/* Billing Card */
	.billing-card {
		background: #f8fafc;
		border: 2px solid #cbd5e1;
	}

	.billing-summary-grid {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 1rem;
	}

	.balance-display {
		padding: 8px 12px;
		background: #fef2f2;
		border-radius: 8px;
		border: 1px solid #fecaca;
		color: #dc2626;
	}

	.submit-action-strip {
		display: flex;
		justify-content: space-between;
		align-items: center;
		flex-wrap: wrap;
		gap: 1rem;
		margin-top: 1.25rem;
		padding-top: 1.25rem;
		border-top: 1px solid #e2e8f0;
	}

	.summary-reminder {
		font-size: 0.9rem;
		color: #334155;
	}

	.btn-confirm-booking {
		background: #16a34a;
		color: #ffffff;
		border: none;
		border-radius: 8px;
		padding: 12px 24px;
		font-size: 1rem;
		font-weight: 800;
		cursor: pointer;
		box-shadow: 0 4px 14px rgba(22, 163, 74, 0.35);
		transition: all 0.15s ease;
	}

	.btn-confirm-booking:hover {
		background: #15803d;
		transform: translateY(-1px);
		box-shadow: 0 6px 18px rgba(22, 163, 74, 0.45);
	}

	/* Modal Backdrop & Card */
	.modal-backdrop {
		position: fixed;
		inset: 0;
		background: rgba(15, 23, 42, 0.55);
		backdrop-filter: blur(4px);
		z-index: 9999;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 1rem;
	}

	.modal-card {
		background: #ffffff;
		border-radius: 14px;
		width: 680px;
		max-width: 95vw;
		box-shadow: 0 20px 40px rgba(0,0,0,0.25);
		overflow: hidden;
	}

	.modal-header-green {
		background: #10b981;
		color: #ffffff;
		padding: 1.25rem 1.5rem;
		display: flex;
		align-items: center;
		gap: 1rem;
		position: relative;
	}

	.succ-icon {
		font-size: 2rem;
	}

	.succ-title {
		font-size: 1.3rem;
		font-weight: 800;
		margin: 0;
	}

	.succ-booking-id {
		font-size: 0.85rem;
		background: rgba(255,255,255,0.25);
		padding: 2px 8px;
		border-radius: 4px;
	}

	.close-x-btn {
		position: absolute;
		right: 1.25rem;
		top: 1.25rem;
		background: transparent;
		border: none;
		color: #ffffff;
		font-size: 1.2rem;
		cursor: pointer;
	}

	.modal-body {
		padding: 1.5rem;
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}

	.succ-desc {
		font-size: 0.95rem;
		color: #334155;
		margin: 0;
	}

	.whatsapp-preview-card {
		background: #f0fdf4;
		border: 1px solid #bbf7d0;
		border-radius: 8px;
		padding: 1rem;
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	.wa-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		font-size: 0.8125rem;
		font-weight: 700;
		color: #166534;
	}

	.copy-chip-btn {
		background: #ffffff;
		border: 1px solid #86efac;
		padding: 3px 8px;
		border-radius: 4px;
		font-size: 0.75rem;
		font-weight: 700;
		color: #15803d;
		cursor: pointer;
	}

	.wa-message-content {
		background: #ffffff;
		border: 1px solid #dcfce7;
		border-radius: 6px;
		padding: 0.85rem;
		font-size: 0.75rem;
		line-height: 1.5;
		color: #14532d;
		max-height: 250px;
		overflow-y: auto;
	}

	.modal-footer {
		padding: 1rem 1.5rem;
		background: #f8fafc;
		border-top: 1px solid #e2e8f0;
		display: flex;
		justify-content: flex-end;
		gap: 0.75rem;
	}

	.btn-send-whatsapp {
		background: #22c55e;
		color: #ffffff;
		padding: 10px 18px;
		border-radius: 8px;
		font-weight: 700;
		font-size: 0.875rem;
		text-decoration: none;
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
	}

	.btn-go-dashboard {
		background: #ffffff;
		border: 1px solid #cbd5e1;
		color: #334155;
		padding: 10px 18px;
		border-radius: 8px;
		font-weight: 700;
		font-size: 0.875rem;
		cursor: pointer;
	}

	/* General Utilities */
	.font-mono { font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace; }
	.font-bold { font-weight: 700; }
	.font-semibold { font-weight: 600; }
	.text-blue { color: #2563eb; }
	.text-emerald { color: #059669; }
	.text-lg { font-size: 1.1rem; }
</style>
