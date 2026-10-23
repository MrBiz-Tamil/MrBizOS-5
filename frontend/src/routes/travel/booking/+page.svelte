<script lang="ts">
	import PortalNavbar from '$lib/components/portal/PortalNavbar.svelte';

	interface TourPackageOption {
		id: string;
		title: string;
		circuit: string;
		nights: number;
		days: number;
		basePricePerPax: number;
		imageUrl: string;
		badge: string;
		highlights: string[];
		bestFor: string;
		defaultVehicle: string;
	}

	const packages: TourPackageOption[] = [
		{
			id: 'kashmir-paradise',
			title: 'Kashmir Valley Paradise & Gulmarg Snow',
			circuit: 'Srinagar ➔ Gulmarg ➔ Pahalgam ➔ Dal Lake',
			nights: 5,
			days: 6,
			basePricePerPax: 18500,
			imageUrl: 'https://images.unsplash.com/photo-1595815771614-ade9d652a65d?auto=format&fit=crop&w=800&q=80',
			badge: 'Top Seller',
			highlights: ['Gondola Phase 1 & 2 Cable Car', 'Heritage Dal Lake Houseboat Night', 'Betaab Valley & Aru Valley Chauffeur', 'Shikara 1-Hour Sunset Cruise'],
			bestFor: 'Family & Honeymoon',
			defaultVehicle: 'Toyota Innova Crysta AC'
		},
		{
			id: 'kerala-backwaters',
			title: 'Kerala Mist & Emerald Backwaters',
			circuit: 'Cochin ➔ Munnar ➔ Thekkady ➔ Alleppey Houseboat',
			nights: 5,
			days: 6,
			basePricePerPax: 16900,
			imageUrl: 'https://images.unsplash.com/photo-1602216056096-3b40cc0c9944?auto=format&fit=crop&w=800&q=80',
			badge: 'Trending',
			highlights: ['Private Luxury Houseboat with all meals', 'Munnar Tea Plantation Safari', 'Periyar Wildlife Sanctuary boat cruise', 'Spice Plantation Tour'],
			bestFor: 'Nature Lovers & Relaxation',
			defaultVehicle: 'Toyota Etios / Crysta AC'
		},
		{
			id: 'dubai-marvels',
			title: 'Dubai Extravaganza & Desert Safari',
			circuit: 'Dubai Marina ➔ Downtown ➔ Desert Dunes ➔ Abu Dhabi',
			nights: 4,
			days: 5,
			basePricePerPax: 32000,
			imageUrl: 'https://images.unsplash.com/photo-1512453979798-5ea266f8880c?auto=format&fit=crop&w=800&q=80',
			badge: 'International',
			highlights: ['Burj Khalifa 124th Floor At The Top', 'Red Dune Desert Safari with BBQ Dinner', 'Dubai Marina Luxury Yacht Cruise', 'Abu Dhabi Sheikh Zayed Mosque'],
			bestFor: 'Luxury & Shopping',
			defaultVehicle: 'Luxury GMC / Lexus Chauffeur'
		},
		{
			id: 'bali-escape',
			title: 'Exotic Bali Tropical Island & Temples',
			circuit: 'Denpasar ➔ Ubud ➔ Kuta ➔ Nusa Penida Island',
			nights: 5,
			days: 6,
			basePricePerPax: 28500,
			imageUrl: 'https://images.unsplash.com/photo-1537996194471-e657df975ab4?auto=format&fit=crop&w=800&q=80',
			badge: 'Honeymoon Choice',
			highlights: ['Private Pool Villa in Ubud', 'Nusa Penida Kelingking Beach speed boat', 'Uluwatu Sunset Temple & Kecak Dance', 'Tegenungan Waterfall & Jungle Swing'],
			bestFor: 'Couples & Adventure',
			defaultVehicle: 'Toyota Avanza AC Chauffeur'
		},
		{
			id: 'rajasthan-royals',
			title: 'Royal Rajasthan Heritage & Palaces',
			circuit: 'Jaipur Pink City ➔ Jodhpur Blue City ➔ Udaipur Lakes',
			nights: 6,
			days: 7,
			basePricePerPax: 21500,
			imageUrl: 'https://images.unsplash.com/photo-1477587458883-47145ed94245?auto=format&fit=crop&w=800&q=80',
			badge: 'Cultural Gold',
			highlights: ['Amber Fort Elephant / Jeep Ascend', 'Lake Pichola Royal Boat Ride', 'Mehrangarh Fort Guided Tour', 'Chokhi Dhani Rajasthani Cultural Dinner'],
			bestFor: 'Heritage & Architecture',
			defaultVehicle: 'Toyota Innova Crysta AC'
		}
	];

	// Wizard Step State
	let currentStep = $state<1 | 2 | 3 | 4>(1);

	// Step 1: Selected Package
	let selectedPackageId = $state<string>('kashmir-paradise');
	let activePackage = $derived(packages.find(p => p.id === selectedPackageId) || packages[0]);

	// Step 2: Customization
	let travelStartDate = $state('2026-11-15');
	let travelEndDate = $state('2026-11-20');
	let adultsCount = $state(2);
	let childrenCount = $state(0);
	let infantsCount = $state(0);

	let hotelTier = $state<'3star' | '4star' | '5star'>('4star');
	let vehicleOption = $state<'sedan' | 'crysta' | 'tempo' | 'luxury'>('crysta');

	// Add-on checkboxes
	let addFlights = $state(false);
	let addGondolaPasses = $state(true);
	let addShikaraCruise = $state(true);
	let addMealPlan = $state(true); // MAP Plan
	let addTravelInsurance = $state(true);

	// Step 3: Traveler info
	let guestName = $state('Rahul Sharma');
	let guestPhone = $state('+91 98840 11223');
	let guestEmail = $state('rahul.sharma@example.com');
	let pickupLocation = $state('Srinagar Airport Exit (SXR)');
	let flightNumber = $state('6E-2144');
	let mealPreference = $state('North Indian (Veg & Non-Veg)');
	let specialNotes = $state('Traveling with family, request child booster seat and room with valley view.');

	// Step 4: Payment plan
	let paymentPlan = $state<'token' | 'full'>('token');

	// Financial Calculations
	let hotelTierMultiplier = $derived(hotelTier === '3star' ? 1.0 : hotelTier === '4star' ? 1.25 : 1.6);
	let vehicleCostPerTrip = $derived(
		vehicleOption === 'sedan' ? 9000 :
		vehicleOption === 'crysta' ? 14500 :
		vehicleOption === 'tempo' ? 24000 : 36000
	);

	let totalPassengers = $derived(adultsCount + childrenCount);
	let basePackageCost = $derived(Math.round(activePackage.basePricePerPax * hotelTierMultiplier * adultsCount + (activePackage.basePricePerPax * 0.7 * hotelTierMultiplier * childrenCount)));

	let addonsCost = $derived(
		(addFlights ? 7500 * totalPassengers : 0) +
		(addGondolaPasses ? 1950 * totalPassengers : 0) +
		(addShikaraCruise ? 1200 : 0) +
		(addMealPlan ? 800 * activePackage.nights * totalPassengers : 0) +
		(addTravelInsurance ? 350 * totalPassengers : 0)
	);

	let subtotal = $derived(basePackageCost + vehicleCostPerTrip + addonsCost);
	let gstTax = $derived(Math.round(subtotal * 0.05));
	let grandTotal = $derived(subtotal + gstTax);
	let tokenDeposit = $derived(Math.round(grandTotal * 0.25));
	let payableNow = $derived(paymentPlan === 'token' ? tokenDeposit : grandTotal);

	// Confirmation Modal
	let showConfirmationModal = $state(false);
	let generatedPnr = $state('');

	function goToStep(step: 1 | 2 | 3 | 4) {
		currentStep = step;
		window.scrollTo({ top: 180, behavior: 'smooth' });
	}

	function confirmBooking() {
		generatedPnr = 'MB-TRV-' + Math.floor(10000 + Math.random() * 90000);
		showConfirmationModal = true;
	}
</script>

<svelte:head>
	<title>Book a Tour Package | MrBiz OS Travel & Tours</title>
	<meta name="description" content="Custom tour package booking with instant PNR generation, hotel voucher issuance, chauffeur assignment, and price calculator." />
</svelte:head>

<div class="booking-portal-page">
	<PortalNavbar activeTab="booking" />

	<!-- Header Banner -->
	<div class="booking-header">
		<div class="booking-header-inner">
			<div class="header-badge">
				<span class="pulse-dot"></span>
				<span>INSTANT DMC CONFIRMATION • ZERO WALKOUT GUARANTEE</span>
			</div>
			<h1 class="header-title">Tailor & Book Your Ideal Holiday</h1>
			<p class="header-sub">
				Select your circuit, customize private chauffeurs, pick luxury hotel tiers, and receive verified digital vouchers instantly.
			</p>

			<!-- Step Progress Indicator -->
			<div class="wizard-steps-nav">
				<button type="button" class="wizard-step {currentStep >= 1 ? 'active' : ''} {currentStep === 1 ? 'current' : ''}" onclick={() => goToStep(1)}>
					<span class="step-num">1</span>
					<span class="step-label">Select Circuit</span>
				</button>
				<div class="step-line {currentStep >= 2 ? 'filled' : ''}"></div>
				<button type="button" class="wizard-step {currentStep >= 2 ? 'active' : ''} {currentStep === 2 ? 'current' : ''}" onclick={() => goToStep(2)}>
					<span class="step-num">2</span>
					<span class="step-label">Customize Stays & Fleet</span>
				</button>
				<div class="step-line {currentStep >= 3 ? 'filled' : ''}"></div>
				<button type="button" class="wizard-step {currentStep >= 3 ? 'active' : ''} {currentStep === 3 ? 'current' : ''}" onclick={() => goToStep(3)}>
					<span class="step-num">3</span>
					<span class="step-label">Traveler Details</span>
				</button>
				<div class="step-line {currentStep >= 4 ? 'filled' : ''}"></div>
				<button type="button" class="wizard-step {currentStep >= 4 ? 'active' : ''} {currentStep === 4 ? 'current' : ''}" onclick={() => goToStep(4)}>
					<span class="step-num">4</span>
					<span class="step-label">Review & Confirm</span>
				</button>
			</div>
		</div>
	</div>

	<!-- Main Wizard Workspace -->
	<div class="booking-content-wrap">
		<div class="booking-grid">
			<!-- LEFT COLUMN: WIZARD STEPS -->
			<div class="wizard-pane">
				<!-- STEP 1: SELECT PACKAGE -->
				{#if currentStep === 1}
					<div class="step-card">
						<div class="card-step-title">
							<span class="step-indicator">STEP 1 OF 4</span>
							<h2>Choose Your Holiday Destination</h2>
							<p>All packages include verified hotel vouchers, dedicated chauffeur, airport transfers, and daily breakfasts.</p>
						</div>

						<div class="package-cards-list">
							{#each packages as p}
								<div
									class="pkg-select-item {selectedPackageId === p.id ? 'selected' : ''}"
									onclick={() => (selectedPackageId = p.id)}
								>
									<div class="pkg-img-wrap">
										<img src={p.imageUrl} alt={p.title} class="pkg-thumb" />
										<span class="pkg-badge">{p.badge}</span>
									</div>

									<div class="pkg-body">
										<div class="pkg-top-meta">
											<span class="circuit-route">📍 {p.circuit}</span>
											<span class="duration-tag">⏱️ {p.days}D / {p.nights}N</span>
										</div>
										<h3 class="pkg-title">{p.title}</h3>

										<div class="highlights-row">
											{#each p.highlights as h}
												<span class="highlight-chip">✓ {h}</span>
											{/each}
										</div>

										<div class="pkg-footer">
											<div class="price-box">
												<span class="from-text">From</span>
												<span class="rate-text">₹{p.basePricePerPax.toLocaleString('en-IN')}</span>
												<span class="pax-text">/ person</span>
											</div>
											<button
												type="button"
												class="btn-select-pkg {selectedPackageId === p.id ? 'selected' : ''}"
											>
												{selectedPackageId === p.id ? 'Selected ✓' : 'Select Package'}
											</button>
										</div>
									</div>
								</div>
							{/each}
						</div>

						<div class="step-next-bar">
							<button type="button" class="btn-step-next" onclick={() => goToStep(2)}>
								<span>Continue to Customization</span>
								<span>➔</span>
							</button>
						</div>
					</div>
				{/if}

				<!-- STEP 2: CUSTOMIZE STAYS & FLEET -->
				{#if currentStep === 2}
					<div class="step-card">
						<div class="card-step-title">
							<span class="step-indicator">STEP 2 OF 4</span>
							<h2>Customize Stays, Fleet & Inclusions</h2>
							<p>Tailor your group size, hotel standards, chauffeur vehicle, and sightseeing passes.</p>
						</div>

						<!-- Dates & Group -->
						<div class="section-box">
							<h3 class="box-title">📅 Travel Dates & Passenger Count</h3>
							<div class="form-row-3">
								<div class="form-field">
									<label for="b-start">Departure Date</label>
									<input id="b-start" type="date" bind:value={travelStartDate} class="form-ctrl" />
								</div>
								<div class="form-field">
									<label for="b-end">Return Date</label>
									<input id="b-end" type="date" bind:value={travelEndDate} class="form-ctrl" />
								</div>
								<div class="form-field">
									<label for="b-dur">Duration</label>
									<input id="b-dur" type="text" readonly value="{activePackage.nights} Nights / {activePackage.days} Days" class="form-ctrl readonly" />
								</div>
							</div>

							<div class="form-row-3 mt-3">
								<div class="form-field">
									<label for="b-adults">Adults (12+ yrs)</label>
									<input id="b-adults" type="number" min="1" max="20" bind:value={adultsCount} class="form-ctrl" />
								</div>
								<div class="form-field">
									<label for="b-kids">Children (5 - 11 yrs)</label>
									<input id="b-kids" type="number" min="0" max="10" bind:value={childrenCount} class="form-ctrl" />
								</div>
								<div class="form-field">
									<label for="b-infants">Infants (0 - 4 yrs)</label>
									<input id="b-infants" type="number" min="0" max="5" bind:value={infantsCount} class="form-ctrl" />
								</div>
							</div>
						</div>

						<!-- Hotel Category -->
						<div class="section-box">
							<h3 class="box-title">🏨 Hotel Accommodation Standard</h3>
							<div class="tier-options-grid">
								<div
									class="tier-option {hotelTier === '3star' ? 'selected' : ''}"
									onclick={() => (hotelTier = '3star')}
								>
									<div class="tier-badge">3-Star Deluxe</div>
									<div class="tier-name">Comfort & Cozy</div>
									<div class="tier-desc">Centrally heated, attached private baths, complimentary Wi-Fi and buffet breakfast.</div>
									<div class="tier-rate">Standard DMC Rate</div>
								</div>

								<div
									class="tier-option {hotelTier === '4star' ? 'selected' : ''}"
									onclick={() => (hotelTier = '4star')}
								>
									<div class="tier-badge recommend">Recommended</div>
									<div class="tier-name">4-Star Premium Boutique</div>
									<div class="tier-desc">Mountain view / Riverfront balconies, premium hospitality, buffet dinner included.</div>
									<div class="tier-rate">+25% Upgrade</div>
								</div>

								<div
									class="tier-option {hotelTier === '5star' ? 'selected' : ''}"
									onclick={() => (hotelTier = '5star')}
								>
									<div class="tier-badge luxury">Luxury VIP</div>
									<div class="tier-name">5-Star Luxury & Heritage</div>
									<div class="tier-desc">World-class luxury resorts & ultra-deluxe Royal Shikara houseboats with gourmet dining.</div>
									<div class="tier-rate">+60% Upgrade</div>
								</div>
							</div>
						</div>

						<!-- Vehicle Selection -->
						<div class="section-box">
							<h3 class="box-title">🚗 Dedicated Private Chauffeur & Cab</h3>
							<div class="vehicle-options-grid">
								<div
									class="vehicle-card {vehicleOption === 'sedan' ? 'selected' : ''}"
									onclick={() => (vehicleOption = 'sedan')}
								>
									<div class="vehicle-icon">🚕</div>
									<div class="vehicle-name">Sedan AC</div>
									<div class="vehicle-model">Swift Dzire / Toyota Etios</div>
									<div class="vehicle-cap">Up to 3 Guests • 2 Bags</div>
									<div class="vehicle-price">₹9,000 all-inclusive</div>
								</div>

								<div
									class="vehicle-card {vehicleOption === 'crysta' ? 'selected' : ''}"
									onclick={() => (vehicleOption = 'crysta')}
								>
									<div class="vehicle-tag">Most Popular</div>
									<div class="vehicle-icon">🚙</div>
									<div class="vehicle-name">Toyota Innova Crysta</div>
									<div class="vehicle-model">Luxury Captain Seats AC</div>
									<div class="vehicle-cap">Up to 6 Guests • 5 Bags</div>
									<div class="vehicle-price">₹14,500 all-inclusive</div>
								</div>

								<div
									class="vehicle-card {vehicleOption === 'tempo' ? 'selected' : ''}"
									onclick={() => (vehicleOption = 'tempo')}
								>
									<div class="vehicle-icon">🚐</div>
									<div class="vehicle-name">Tempo Traveller 12-Seater</div>
									<div class="vehicle-model">Pushback Luxury Coach AC</div>
									<div class="vehicle-cap">7 to 12 Guests • Group Bags</div>
									<div class="vehicle-price">₹24,000 all-inclusive</div>
								</div>
							</div>
						</div>

						<!-- Add-ons & Fast Passes -->
						<div class="section-box">
							<h3 class="box-title">✨ Curated Experience Add-ons</h3>
							<div class="addons-list">
								<label class="addon-item">
									<input type="checkbox" bind:checked={addGondolaPasses} />
									<div class="addon-info">
										<div class="addon-title">🚠 Gulmarg Gondola Cable Car Phase 1 & 2 Passes</div>
										<div class="addon-desc">Confirmed queue-bypass time-slots up to Kongdoori & Apharwat Peak.</div>
									</div>
									<div class="addon-price">+₹1,950 / pax</div>
								</label>

								<label class="addon-item">
									<input type="checkbox" bind:checked={addShikaraCruise} />
									<div class="addon-info">
										<div class="addon-title">🚣 1-Hour Private Sunset Shikara Ride on Dal Lake</div>
										<div class="addon-desc">Floating gardens, lotus flowers & evening floating market exploration.</div>
									</div>
									<div class="addon-price">+₹1,200 total</div>
								</label>

								<label class="addon-item">
									<input type="checkbox" bind:checked={addMealPlan} />
									<div class="addon-info">
										<div class="addon-title">🍽️ MAP Meal Plan (Daily Gourmet Dinner Included)</div>
										<div class="addon-desc">Chefs special multi-course dinner at all hotels & houseboat stays.</div>
									</div>
									<div class="addon-price">+₹800 / pax / night</div>
								</label>

								<label class="addon-item">
									<input type="checkbox" bind:checked={addFlights} />
									<div class="addon-info">
										<div class="addon-title">✈️ Domestic Flight Ticketing Assistance</div>
										<div class="addon-desc">Round-trip scheduled flights with 20kg check-in baggage allowance.</div>
									</div>
									<div class="addon-price">+₹7,500 / pax est.</div>
								</label>

								<label class="addon-item">
									<input type="checkbox" bind:checked={addTravelInsurance} />
									<div class="addon-info">
										<div class="addon-title">🛡️ Travel Protection & On-Ground Medical Cover</div>
										<div class="addon-desc">Flight delay protection, luggage loss cover & emergency doctor on call.</div>
									</div>
									<div class="addon-price">+₹350 / pax</div>
								</label>
							</div>
						</div>

						<div class="step-nav-bar">
							<button type="button" class="btn-step-prev" onclick={() => goToStep(1)}>
								<span>← Back to Circuits</span>
							</button>
							<button type="button" class="btn-step-next" onclick={() => goToStep(3)}>
								<span>Proceed to Guest Details</span>
								<span>➔</span>
							</button>
						</div>
					</div>
				{/if}

				<!-- STEP 3: TRAVELER DETAILS -->
				{#if currentStep === 3}
					<div class="step-card">
						<div class="card-step-title">
							<span class="step-indicator">STEP 3 OF 4</span>
							<h2>Lead Passenger & Arrival Information</h2>
							<p>Provide contact information for instant digital vouchers and chauffeur placard setup.</p>
						</div>

						<div class="section-box">
							<h3 class="box-title">👤 Primary Contact</h3>
							<div class="form-row-2">
								<div class="form-field">
									<label for="g-name">Primary Traveler Full Name *</label>
									<input id="g-name" type="text" bind:value={guestName} placeholder="e.g. Rahul Sharma" class="form-ctrl" />
								</div>
								<div class="form-field">
									<label for="g-phone">WhatsApp Number (For Instant Vouchers) *</label>
									<input id="g-phone" type="tel" bind:value={guestPhone} placeholder="+91 98840 11223" class="form-ctrl" />
								</div>
							</div>

							<div class="form-row-2 mt-3">
								<div class="form-field">
									<label for="g-email">Email Address *</label>
									<input id="g-email" type="email" bind:value={guestEmail} placeholder="rahul@example.com" class="form-ctrl" />
								</div>
								<div class="form-field">
									<label for="g-pickup">Arrival Airport / Pickup Point</label>
									<input id="g-pickup" type="text" bind:value={pickupLocation} placeholder="e.g. Srinagar Airport Exit (SXR)" class="form-ctrl" />
								</div>
							</div>
						</div>

						<div class="section-box">
							<h3 class="box-title">✈️ Flight & Dietary Preferences</h3>
							<div class="form-row-2">
								<div class="form-field">
									<label for="g-flight">Arrival Flight Number (If Available)</label>
									<input id="g-flight" type="text" bind:value={flightNumber} placeholder="e.g. IndiGo 6E-2144" class="form-ctrl" />
								</div>
								<div class="form-field">
									<label for="g-diet">Meal & Dietary Preference</label>
									<select id="g-diet" bind:value={mealPreference} class="form-ctrl">
										<option value="North Indian (Veg & Non-Veg)">North Indian (Veg & Non-Veg)</option>
										<option value="Pure Vegetarian (Jain Available)">Pure Vegetarian (Jain Available)</option>
										<option value="South Indian Specialty">South Indian Specialty</option>
										<option value="Halal Certified">Halal Certified</option>
										<option value="Continental / Western">Continental / Western</option>
									</select>
								</div>
							</div>

							<div class="form-field mt-3">
								<label for="g-notes">Special Requests / Room Layout / Elderly Assistance</label>
								<textarea id="g-notes" bind:value={specialNotes} rows="3" class="form-ctrl textarea" placeholder="e.g. Ground floor room requested, anniversary cake arrangement..."></textarea>
							</div>
						</div>

						<div class="step-nav-bar">
							<button type="button" class="btn-step-prev" onclick={() => goToStep(2)}>
								<span>← Back to Customization</span>
							</button>
							<button type="button" class="btn-step-next" onclick={() => goToStep(4)}>
								<span>Review & Generate Booking</span>
								<span>➔</span>
							</button>
						</div>
					</div>
				{/if}

				<!-- STEP 4: REVIEW & CONFIRM -->
				{#if currentStep === 4}
					<div class="step-card">
						<div class="card-step-title">
							<span class="step-indicator">STEP 4 OF 4</span>
							<h2>Final Review & Booking Guarantee</h2>
							<p>Review the comprehensive itinerary breakdown and secure your booking with our DMC guarantee.</p>
						</div>

						<div class="review-summary-card">
							<div class="summary-header">
								<div class="summary-pkg-name">📍 {activePackage.title}</div>
								<div class="summary-dates">📅 {travelStartDate} to {travelEndDate} ({activePackage.nights}N/{activePackage.days}D)</div>
							</div>

							<div class="summary-specs-grid">
								<div class="spec-box">
									<div class="spec-label">Group Size</div>
									<div class="spec-val">{adultsCount} Adults{#if childrenCount > 0}, {childrenCount} Children{/if}</div>
								</div>
								<div class="spec-box">
									<div class="spec-label">Hotel Tier</div>
									<div class="spec-val">{hotelTier === '3star' ? '3-Star Deluxe' : hotelTier === '4star' ? '4-Star Premium' : '5-Star Luxury'}</div>
								</div>
								<div class="spec-box">
									<div class="spec-label">Private Fleet</div>
									<div class="spec-val">{vehicleOption === 'crysta' ? 'Toyota Innova Crysta' : vehicleOption === 'sedan' ? 'Sedan AC' : 'Tempo Traveller'}</div>
								</div>
								<div class="spec-box">
									<div class="spec-label">Lead Traveler</div>
									<div class="spec-val">{guestName} ({guestPhone})</div>
								</div>
							</div>

							<!-- Fare Breakdown Table -->
							<div class="fare-breakdown">
								<div class="fare-row">
									<span>Base Tour Package ({adultsCount} Adults)</span>
									<span>₹{basePackageCost.toLocaleString('en-IN')}</span>
								</div>
								<div class="fare-row">
									<span>Dedicated Chauffeur & Vehicle ({activePackage.days} Days)</span>
									<span>₹{vehicleCostPerTrip.toLocaleString('en-IN')}</span>
								</div>
								{#if addonsCost > 0}
									<div class="fare-row">
										<span>Selected Experiences & Passes</span>
										<span>₹{addonsCost.toLocaleString('en-IN')}</span>
									</div>
								{/if}
								<div class="fare-row subtotal">
									<span>Subtotal</span>
									<span>₹{subtotal.toLocaleString('en-IN')}</span>
								</div>
								<div class="fare-row tax">
									<span>Applicable GST (5%)</span>
									<span>₹{gstTax.toLocaleString('en-IN')}</span>
								</div>
								<div class="fare-row total">
									<span>Grand Total Fares</span>
									<span>₹{grandTotal.toLocaleString('en-IN')}</span>
								</div>
							</div>
						</div>

						<!-- Payment Options -->
						<div class="section-box">
							<h3 class="box-title">💳 Select Booking Guarantee Option</h3>
							<div class="payment-options">
								<label class="pay-option-card {paymentPlan === 'token' ? 'selected' : ''}">
									<input type="radio" name="payplan" value="token" bind:group={paymentPlan} />
									<div class="pay-option-content">
										<div class="pay-title">Pay 25% Advance Token Guarantee</div>
										<div class="pay-desc">Pay only <strong>₹{tokenDeposit.toLocaleString('en-IN')}</strong> today to lock hotel rooms & flight seats. Balance payable on arrival.</div>
									</div>
									<div class="pay-amount">₹{tokenDeposit.toLocaleString('en-IN')}</div>
								</label>

								<label class="pay-option-card {paymentPlan === 'full' ? 'selected' : ''}">
									<input type="radio" name="payplan" value="full" bind:group={paymentPlan} />
									<div class="pay-option-content">
										<div class="pay-title">Pay 100% Full Advance (Instant 5% Off)</div>
										<div class="pay-desc">Get an instant 5% invoice rebate and VIP complimentary room upgrade vouchers.</div>
									</div>
									<div class="pay-amount">₹{grandTotal.toLocaleString('en-IN')}</div>
								</label>
							</div>
						</div>

						<div class="step-nav-bar">
							<button type="button" class="btn-step-prev" onclick={() => goToStep(3)}>
								<span>← Edit Details</span>
							</button>
							<button type="button" class="btn-confirm-final" onclick={confirmBooking}>
								<span>Confirm & Generate Instant Vouchers</span>
								<span>➔</span>
							</button>
						</div>
					</div>
				{/if}
			</div>

			<!-- RIGHT COLUMN: LIVE COST CALCULATOR SIDEBAR -->
			<div class="sidebar-pane">
				<div class="live-calc-card">
					<div class="calc-header">
						<span class="calc-badge">LIVE DMC QUOTATION</span>
						<h3 class="calc-title">{activePackage.title}</h3>
						<div class="calc-circuit">📍 {activePackage.circuit}</div>
					</div>

					<div class="calc-preview-thumb">
						<img src={activePackage.imageUrl} alt={activePackage.title} />
						<div class="thumb-overlay">
							<span>{activePackage.days} Days / {activePackage.nights} Nights</span>
						</div>
					</div>

					<div class="calc-details-list">
						<div class="calc-row">
							<span class="row-label">Travelers</span>
							<span class="row-val">{adultsCount} Adults {#if childrenCount > 0}+ {childrenCount} Kids{/if}</span>
						</div>
						<div class="calc-row">
							<span class="row-label">Hotel Category</span>
							<span class="row-val">{hotelTier === '3star' ? '3★ Deluxe' : hotelTier === '4star' ? '4★ Boutique' : '5★ Luxury'}</span>
						</div>
						<div class="calc-row">
							<span class="row-label">Vehicle</span>
							<span class="row-val">{vehicleOption === 'crysta' ? 'Innova Crysta' : vehicleOption === 'sedan' ? 'Sedan AC' : 'Tempo'}</span>
						</div>
						<div class="calc-row">
							<span class="row-label">Meals</span>
							<span class="row-val">{addMealPlan ? 'Breakfast + Dinner (MAP)' : 'Breakfast (CP)'}</span>
						</div>
					</div>

					<div class="calc-price-summary">
						<div class="grand-total-row">
							<span class="label">Total Amount</span>
							<span class="value">₹{grandTotal.toLocaleString('en-IN')}</span>
						</div>
						<div class="token-due-row">
							<span class="label">Payable Today (Token)</span>
							<span class="value">₹{tokenDeposit.toLocaleString('en-IN')}</span>
						</div>
					</div>

					<div class="trust-guarantee-box">
						<div class="guarantee-item">
							<span class="icon">🛡️</span>
							<span>100% Confirmed Vouchers</span>
						</div>
						<div class="guarantee-item">
							<span class="icon">🚗</span>
							<span>Dedicated Vetted Chauffeur</span>
						</div>
						<div class="guarantee-item">
							<span class="icon">⚡</span>
							<span>24/7 Trip Coordinator Support</span>
						</div>
					</div>

					<!-- Direct Assistance Link -->
					<div class="sidebar-help">
						<div class="help-text">Need a customized itinerary or group discount?</div>
						<a
							href="https://wa.me/919884011223?text=Hi%20MrBiz%20Travel%2C%20I%20am%20customizing%20{encodeURIComponent(activePackage.title)}%20for%20{adultsCount}%20guests."
							target="_blank"
							rel="noreferrer"
							class="sidebar-wa-btn"
						>
							<span>💬 WhatsApp Travel Expert</span>
						</a>
					</div>
				</div>
			</div>
		</div>
	</div>
</div>

<!-- Instant Booking Confirmation Modal -->
{#if showConfirmationModal}
	<div class="modal-backdrop">
		<div class="confirm-voucher-modal">
			<div class="voucher-card">
				<!-- Voucher Header -->
				<div class="voucher-top">
					<div class="voucher-brand">
						<div class="brand-logo">✈️</div>
						<div>
							<div class="brand-name">MrBiz Travel & Tours OS</div>
							<div class="brand-sub">Official DMC Confirmation Voucher</div>
						</div>
					</div>
					<div class="pnr-box">
						<div class="pnr-label">BOOKING PNR REF</div>
						<div class="pnr-code">{generatedPnr}</div>
					</div>
				</div>

				<div class="voucher-confirmed-banner">
					<span class="check">✓</span>
					<span>TOUR RESERVATION CONFIRMED & QUEUED FOR DISPATCH</span>
				</div>

				<!-- Voucher Details Grid -->
				<div class="voucher-grid">
					<div class="v-cell">
						<div class="v-label">Package & Circuit</div>
						<div class="v-val bold">{activePackage.title}</div>
						<div class="v-sub">{activePackage.circuit}</div>
					</div>
					<div class="v-cell">
						<div class="v-label">Travel Dates</div>
						<div class="v-val">{travelStartDate} ➔ {travelEndDate}</div>
						<div class="v-sub">{activePackage.nights} Nights / {activePackage.days} Days</div>
					</div>
					<div class="v-cell">
						<div class="v-label">Primary Guest</div>
						<div class="v-val">{guestName}</div>
						<div class="v-sub">{guestPhone} • {totalPassengers} Pax</div>
					</div>
					<div class="v-cell">
						<div class="v-label">Assigned Vehicle</div>
						<div class="v-val">{vehicleOption === 'crysta' ? 'Toyota Innova Crysta AC' : 'Sedan AC'}</div>
						<div class="v-sub">Placard: {guestName.toUpperCase()}</div>
					</div>
				</div>

				<!-- Financial Summary in Voucher -->
				<div class="voucher-finance">
					<div class="f-item">
						<span>Total Tour Fare (incl. GST):</span>
						<strong>₹{grandTotal.toLocaleString('en-IN')}</strong>
					</div>
					<div class="f-item highlight">
						<span>Booking Token Paid:</span>
						<strong>₹{payableNow.toLocaleString('en-IN')}</strong>
					</div>
					<div class="f-item balance">
						<span>Balance Payable on Arrival:</span>
						<span>₹{(grandTotal - payableNow).toLocaleString('en-IN')}</span>
					</div>
				</div>

				<div class="voucher-actions">
					<a href="/login/customer" class="v-btn dashboard">
						<span>Open My Traveler Dashboard</span>
						<span>➔</span>
					</a>
					<a
						href="https://wa.me/{guestPhone.replace(/\D/g, '')}?text=Dear%20{encodeURIComponent(guestName)}%2C%20your%20booking%20for%20{encodeURIComponent(activePackage.title)}%20is%20CONFIRMED!%20Booking%20PNR%3A%20{generatedPnr}."
						target="_blank"
						rel="noreferrer"
						class="v-btn wa"
					>
						<span>Send Voucher to WhatsApp</span>
					</a>
					<button type="button" class="v-btn print" onclick={() => window.print()}>
						<span>🖨️ Print / Save PDF</span>
					</button>
				</div>
			</div>
		</div>
	</div>
{/if}

<style>
	.booking-portal-page {
		min-height: 100vh;
		background: #090d16;
		color: #f8fafc;
		font-family: var(--font-sans, system-ui, -apple-system, sans-serif);
	}

	/* Header */
	.booking-header {
		background: radial-gradient(circle at 50% 20%, rgba(2, 132, 199, 0.2) 0%, rgba(15, 23, 42, 0.95) 75%, #090d16 100%);
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
		padding: 3rem 1.5rem 2rem;
		text-align: center;
	}

	.booking-header-inner {
		max-width: 900px;
		margin: 0 auto;
	}

	.header-badge {
		display: inline-flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.35rem 0.85rem;
		border-radius: 999px;
		background: rgba(16, 185, 129, 0.15);
		border: 1px solid rgba(52, 211, 153, 0.35);
		color: #34d399;
		font-size: 0.74rem;
		font-weight: 800;
		letter-spacing: 0.06em;
		margin-bottom: 1rem;
	}

	.pulse-dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		background: #34d399;
	}

	.header-title {
		font-size: clamp(2rem, 4vw, 3rem);
		font-weight: 900;
		color: #ffffff;
		margin: 0 0 0.75rem;
		letter-spacing: -0.02em;
	}

	.header-sub {
		font-size: 1.05rem;
		color: #94a3b8;
		margin: 0 0 2rem;
	}

	/* Wizard Steps Nav */
	.wizard-steps-nav {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.5rem;
		flex-wrap: wrap;
	}

	.wizard-step {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		background: rgba(255, 255, 255, 0.05);
		border: 1px solid rgba(255, 255, 255, 0.1);
		padding: 0.5rem 1rem;
		border-radius: 999px;
		color: #94a3b8;
		font-size: 0.85rem;
		font-weight: 700;
		cursor: pointer;
		transition: all 0.2s;
	}

	.wizard-step.active {
		color: #ffffff;
		background: rgba(255, 255, 255, 0.09);
	}

	.wizard-step.current {
		background: #0284c7;
		border-color: #38bdf8;
		box-shadow: 0 0 14px rgba(2, 132, 199, 0.4);
		color: #ffffff;
	}

	.step-num {
		width: 20px;
		height: 20px;
		border-radius: 50%;
		background: rgba(255, 255, 255, 0.2);
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 0.75rem;
	}

	.step-line {
		width: 32px;
		height: 2px;
		background: rgba(255, 255, 255, 0.1);
	}

	.step-line.filled {
		background: #0284c7;
	}

	/* Booking Content Layout */
	.booking-content-wrap {
		max-width: 1320px;
		margin: 0 auto;
		padding: 2.5rem 1.5rem 4rem;
	}

	.booking-grid {
		display: grid;
		grid-template-columns: 1fr 380px;
		gap: 2rem;
		align-items: flex-start;
	}

	/* Step Card */
	.step-card {
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 18px;
		padding: 2rem;
		box-shadow: 0 16px 40px rgba(0, 0, 0, 0.4);
	}

	.card-step-title {
		margin-bottom: 2rem;
		padding-bottom: 1rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.06);
	}

	.step-indicator {
		font-size: 0.72rem;
		font-weight: 800;
		color: #38bdf8;
		letter-spacing: 0.08em;
	}

	.card-step-title h2 {
		font-size: 1.6rem;
		font-weight: 800;
		color: #ffffff;
		margin: 0.35rem 0;
	}

	.card-step-title p {
		font-size: 0.92rem;
		color: #94a3b8;
		margin: 0;
	}

	/* Package Cards in Step 1 */
	.package-cards-list {
		display: flex;
		flex-direction: column;
		gap: 1.25rem;
	}

	.pkg-select-item {
		display: grid;
		grid-template-columns: 240px 1fr;
		background: rgba(15, 23, 42, 0.7);
		border: 2px solid rgba(255, 255, 255, 0.08);
		border-radius: 16px;
		overflow: hidden;
		cursor: pointer;
		transition: all 0.2s ease;
	}

	.pkg-select-item:hover {
		border-color: rgba(56, 189, 248, 0.4);
		transform: translateY(-2px);
	}

	.pkg-select-item.selected {
		border-color: #38bdf8;
		background: rgba(2, 132, 199, 0.08);
		box-shadow: 0 0 20px rgba(2, 132, 199, 0.25);
	}

	.pkg-img-wrap {
		position: relative;
		height: 100%;
		min-height: 180px;
	}

	.pkg-thumb {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}

	.pkg-badge {
		position: absolute;
		top: 10px;
		left: 10px;
		background: #0284c7;
		color: #ffffff;
		font-size: 0.7rem;
		font-weight: 800;
		padding: 0.2rem 0.55rem;
		border-radius: 6px;
	}

	.pkg-body {
		padding: 1.25rem;
		display: flex;
		flex-direction: column;
		justify-content: space-between;
	}

	.pkg-top-meta {
		display: flex;
		justify-content: space-between;
		font-size: 0.78rem;
		color: #94a3b8;
		margin-bottom: 0.35rem;
	}

	.pkg-title {
		font-size: 1.22rem;
		font-weight: 800;
		color: #ffffff;
		margin: 0 0 0.65rem;
	}

	.highlights-row {
		display: flex;
		flex-wrap: wrap;
		gap: 0.4rem;
		margin-bottom: 1rem;
	}

	.highlight-chip {
		font-size: 0.74rem;
		background: rgba(255, 255, 255, 0.05);
		color: #cbd5e1;
		padding: 0.2rem 0.55rem;
		border-radius: 6px;
	}

	.pkg-footer {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding-top: 0.75rem;
		border-top: 1px solid rgba(255, 255, 255, 0.06);
	}

	.price-box {
		display: flex;
		align-items: baseline;
		gap: 0.35rem;
	}

	.from-text, .pax-text {
		font-size: 0.78rem;
		color: #94a3b8;
	}

	.rate-text {
		font-size: 1.35rem;
		font-weight: 900;
		color: #38bdf8;
	}

	.btn-select-pkg {
		padding: 0.5rem 1rem;
		border-radius: 8px;
		background: rgba(255, 255, 255, 0.08);
		border: 1px solid rgba(255, 255, 255, 0.15);
		color: #ffffff;
		font-size: 0.84rem;
		font-weight: 700;
		cursor: pointer;
	}

	.btn-select-pkg.selected {
		background: #0284c7;
		border-color: #38bdf8;
	}

	/* Section Boxes */
	.section-box {
		background: rgba(255, 255, 255, 0.02);
		border: 1px solid rgba(255, 255, 255, 0.06);
		border-radius: 14px;
		padding: 1.25rem;
		margin-bottom: 1.5rem;
	}

	.box-title {
		font-size: 1.05rem;
		font-weight: 700;
		color: #ffffff;
		margin: 0 0 1rem;
	}

	.form-row-3 {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 1rem;
	}

	.form-row-2 {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 1rem;
	}

	.mt-3 {
		margin-top: 1rem;
	}

	.form-field {
		display: flex;
		flex-direction: column;
		gap: 0.35rem;
	}

	.form-field label {
		font-size: 0.78rem;
		font-weight: 700;
		color: #94a3b8;
	}

	.form-ctrl {
		width: 100%;
		box-sizing: border-box;
		padding: 0.7rem 0.9rem;
		background: #090d16;
		border: 1px solid rgba(255, 255, 255, 0.12);
		border-radius: 8px;
		color: #ffffff;
		font-size: 0.9rem;
		outline: none;
	}

	.form-ctrl:focus {
		border-color: #38bdf8;
	}

	.form-ctrl.readonly {
		opacity: 0.7;
		cursor: default;
	}

	.form-ctrl.textarea {
		resize: vertical;
	}

	/* Tier Grid */
	.tier-options-grid {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 1rem;
	}

	.tier-option {
		background: #090d16;
		border: 2px solid rgba(255, 255, 255, 0.08);
		border-radius: 12px;
		padding: 1rem;
		cursor: pointer;
		transition: all 0.2s;
	}

	.tier-option.selected {
		border-color: #38bdf8;
		background: rgba(2, 132, 199, 0.1);
	}

	.tier-badge {
		font-size: 0.68rem;
		font-weight: 800;
		color: #94a3b8;
		margin-bottom: 0.25rem;
	}

	.tier-badge.recommend { color: #34d399; }
	.tier-badge.luxury { color: #f59e0b; }

	.tier-name {
		font-size: 0.96rem;
		font-weight: 700;
		color: #ffffff;
		margin-bottom: 0.35rem;
	}

	.tier-desc {
		font-size: 0.78rem;
		color: #94a3b8;
		line-height: 1.4;
		margin-bottom: 0.75rem;
	}

	.tier-rate {
		font-size: 0.8rem;
		font-weight: 700;
		color: #38bdf8;
	}

	/* Vehicle Grid */
	.vehicle-options-grid {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 1rem;
	}

	.vehicle-card {
		position: relative;
		background: #090d16;
		border: 2px solid rgba(255, 255, 255, 0.08);
		border-radius: 12px;
		padding: 1rem;
		cursor: pointer;
		transition: all 0.2s;
	}

	.vehicle-card.selected {
		border-color: #38bdf8;
		background: rgba(2, 132, 199, 0.1);
	}

	.vehicle-tag {
		position: absolute;
		top: -10px;
		right: 10px;
		background: #10b981;
		color: #ffffff;
		font-size: 0.65rem;
		font-weight: 800;
		padding: 0.15rem 0.5rem;
		border-radius: 999px;
	}

	.vehicle-icon {
		font-size: 1.8rem;
		margin-bottom: 0.35rem;
	}

	.vehicle-name {
		font-size: 0.95rem;
		font-weight: 800;
		color: #ffffff;
	}

	.vehicle-model {
		font-size: 0.76rem;
		color: #94a3b8;
		margin-bottom: 0.35rem;
	}

	.vehicle-cap {
		font-size: 0.74rem;
		color: #cbd5e1;
		margin-bottom: 0.5rem;
	}

	.vehicle-price {
		font-size: 0.82rem;
		font-weight: 800;
		color: #38bdf8;
	}

	/* Add-ons */
	.addons-list {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	.addon-item {
		display: flex;
		align-items: center;
		gap: 1rem;
		background: #090d16;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 10px;
		padding: 0.75rem 1rem;
		cursor: pointer;
	}

	.addon-info {
		flex: 1;
	}

	.addon-title {
		font-size: 0.88rem;
		font-weight: 700;
		color: #ffffff;
	}

	.addon-desc {
		font-size: 0.76rem;
		color: #94a3b8;
	}

	.addon-price {
		font-size: 0.85rem;
		font-weight: 700;
		color: #38bdf8;
		white-space: nowrap;
	}

	/* Review Summary */
	.review-summary-card {
		background: rgba(255, 255, 255, 0.02);
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 14px;
		padding: 1.5rem;
		margin-bottom: 1.5rem;
	}

	.summary-header {
		margin-bottom: 1.25rem;
		padding-bottom: 1rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.06);
	}

	.summary-pkg-name {
		font-size: 1.25rem;
		font-weight: 800;
		color: #ffffff;
		margin-bottom: 0.25rem;
	}

	.summary-dates {
		font-size: 0.85rem;
		color: #94a3b8;
	}

	.summary-specs-grid {
		display: grid;
		grid-template-columns: repeat(4, 1fr);
		gap: 0.75rem;
		margin-bottom: 1.5rem;
	}

	.spec-box {
		background: #090d16;
		padding: 0.75rem;
		border-radius: 8px;
	}

	.spec-label {
		font-size: 0.7rem;
		color: #64748b;
		margin-bottom: 0.2rem;
	}

	.spec-val {
		font-size: 0.85rem;
		font-weight: 700;
		color: #e2e8f0;
	}

	/* Fare Table */
	.fare-breakdown {
		display: flex;
		flex-direction: column;
		gap: 0.6rem;
		padding-top: 1rem;
		border-top: 1px solid rgba(255, 255, 255, 0.06);
	}

	.fare-row {
		display: flex;
		justify-content: space-between;
		font-size: 0.88rem;
		color: #cbd5e1;
	}

	.fare-row.subtotal {
		padding-top: 0.5rem;
		border-top: 1px dashed rgba(255, 255, 255, 0.1);
		color: #94a3b8;
	}

	.fare-row.tax {
		color: #94a3b8;
		font-size: 0.82rem;
	}

	.fare-row.total {
		padding-top: 0.5rem;
		border-top: 1px solid rgba(255, 255, 255, 0.15);
		font-size: 1.25rem;
		font-weight: 800;
		color: #38bdf8;
	}

	/* Payment Options */
	.payment-options {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	.pay-option-card {
		display: flex;
		align-items: center;
		gap: 1rem;
		background: #090d16;
		border: 2px solid rgba(255, 255, 255, 0.08);
		border-radius: 12px;
		padding: 1rem;
		cursor: pointer;
	}

	.pay-option-card.selected {
		border-color: #38bdf8;
		background: rgba(2, 132, 199, 0.1);
	}

	.pay-option-content {
		flex: 1;
	}

	.pay-title {
		font-size: 0.95rem;
		font-weight: 700;
		color: #ffffff;
	}

	.pay-desc {
		font-size: 0.8rem;
		color: #94a3b8;
	}

	.pay-amount {
		font-size: 1.2rem;
		font-weight: 900;
		color: #38bdf8;
	}

	/* Step Nav Buttons */
	.step-next-bar, .step-nav-bar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-top: 2rem;
		padding-top: 1.5rem;
		border-top: 1px solid rgba(255, 255, 255, 0.08);
	}

	.step-next-bar {
		justify-content: flex-end;
	}

	.btn-step-prev {
		background: rgba(255, 255, 255, 0.08);
		border: 1px solid rgba(255, 255, 255, 0.15);
		color: #ffffff;
		padding: 0.75rem 1.4rem;
		border-radius: 10px;
		font-weight: 700;
		cursor: pointer;
	}

	.btn-step-next {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		background: linear-gradient(135deg, #0284c7 0%, #2563eb 100%);
		border: none;
		color: #ffffff;
		padding: 0.75rem 1.6rem;
		border-radius: 10px;
		font-weight: 700;
		font-size: 0.92rem;
		cursor: pointer;
		box-shadow: 0 4px 14px rgba(2, 132, 199, 0.4);
	}

	.btn-confirm-final {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		background: linear-gradient(135deg, #10b981 0%, #059669 100%);
		border: none;
		color: #ffffff;
		padding: 0.85rem 1.8rem;
		border-radius: 12px;
		font-weight: 800;
		font-size: 1rem;
		cursor: pointer;
		box-shadow: 0 4px 16px rgba(16, 185, 129, 0.4);
	}

	/* RIGHT SIDEBAR: Live Calculator */
	.live-calc-card {
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: 18px;
		padding: 1.5rem;
		position: sticky;
		top: 90px;
		box-shadow: 0 16px 36px rgba(0, 0, 0, 0.4);
	}

	.calc-badge {
		font-size: 0.68rem;
		font-weight: 800;
		color: #38bdf8;
		letter-spacing: 0.08em;
	}

	.calc-title {
		font-size: 1.15rem;
		font-weight: 800;
		color: #ffffff;
		margin: 0.25rem 0;
	}

	.calc-circuit {
		font-size: 0.78rem;
		color: #94a3b8;
		margin-bottom: 1rem;
	}

	.calc-preview-thumb {
		position: relative;
		border-radius: 10px;
		overflow: hidden;
		height: 120px;
		margin-bottom: 1rem;
	}

	.calc-preview-thumb img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}

	.thumb-overlay {
		position: absolute;
		bottom: 8px;
		right: 8px;
		background: rgba(0, 0, 0, 0.75);
		color: #ffffff;
		font-size: 0.72rem;
		font-weight: 700;
		padding: 0.2rem 0.5rem;
		border-radius: 6px;
	}

	.calc-details-list {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		padding-bottom: 1rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.06);
		margin-bottom: 1rem;
	}

	.calc-row {
		display: flex;
		justify-content: space-between;
		font-size: 0.8rem;
	}

	.row-label { color: #94a3b8; }
	.row-val { color: #f1f5f9; font-weight: 600; }

	.calc-price-summary {
		background: rgba(2, 132, 199, 0.1);
		border: 1px solid rgba(56, 189, 248, 0.25);
		border-radius: 10px;
		padding: 0.85rem 1rem;
		margin-bottom: 1rem;
	}

	.grand-total-row, .token-due-row {
		display: flex;
		justify-content: space-between;
		align-items: baseline;
	}

	.grand-total-row .label { font-size: 0.85rem; color: #cbd5e1; }
	.grand-total-row .value { font-size: 1.35rem; font-weight: 900; color: #ffffff; }

	.token-due-row {
		margin-top: 0.4rem;
		padding-top: 0.4rem;
		border-top: 1px dashed rgba(255, 255, 255, 0.1);
	}

	.token-due-row .label { font-size: 0.78rem; color: #38bdf8; }
	.token-due-row .value { font-size: 1.05rem; font-weight: 800; color: #38bdf8; }

	.trust-guarantee-box {
		display: flex;
		flex-direction: column;
		gap: 0.45rem;
		margin-bottom: 1.25rem;
	}

	.guarantee-item {
		display: flex;
		align-items: center;
		gap: 0.45rem;
		font-size: 0.78rem;
		color: #94a3b8;
	}

	.sidebar-help {
		text-align: center;
		padding-top: 0.75rem;
		border-top: 1px solid rgba(255, 255, 255, 0.06);
	}

	.help-text {
		font-size: 0.76rem;
		color: #64748b;
		margin-bottom: 0.5rem;
	}

	.sidebar-wa-btn {
		display: block;
		padding: 0.55rem;
		border-radius: 8px;
		background: rgba(37, 211, 102, 0.15);
		color: #25d366;
		border: 1px solid rgba(37, 211, 102, 0.35);
		font-size: 0.82rem;
		font-weight: 700;
		text-decoration: none;
	}

	/* Voucher Confirmation Modal */
	.modal-backdrop {
		position: fixed;
		inset: 0;
		z-index: 2000;
		background: rgba(0, 0, 0, 0.85);
		backdrop-filter: blur(10px);
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 1.5rem;
	}

	.confirm-voucher-modal {
		max-width: 640px;
		width: 100%;
		background: #0f172a;
		border: 2px solid #38bdf8;
		border-radius: 20px;
		padding: 2rem;
		box-shadow: 0 25px 60px rgba(0, 0, 0, 0.8);
		max-height: 90vh;
		overflow-y: auto;
	}

	.voucher-top {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 1.25rem;
		padding-bottom: 1rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.1);
	}

	.voucher-brand {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}

	.brand-logo {
		font-size: 1.8rem;
		background: #0284c7;
		width: 44px;
		height: 44px;
		border-radius: 10px;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.brand-name {
		font-size: 1.05rem;
		font-weight: 800;
		color: #ffffff;
	}

	.brand-sub {
		font-size: 0.72rem;
		color: #94a3b8;
	}

	.pnr-box {
		text-align: right;
	}

	.pnr-label {
		font-size: 0.65rem;
		color: #94a3b8;
		font-weight: 700;
	}

	.pnr-code {
		font-size: 1.35rem;
		font-weight: 900;
		color: #38bdf8;
		letter-spacing: 0.04em;
	}

	.voucher-confirmed-banner {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		background: rgba(16, 185, 129, 0.2);
		border: 1px solid #10b981;
		color: #34d399;
		padding: 0.75rem 1rem;
		border-radius: 10px;
		font-size: 0.84rem;
		font-weight: 800;
		margin-bottom: 1.25rem;
	}

	.voucher-grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 0.85rem;
		margin-bottom: 1.25rem;
	}

	.v-cell {
		background: rgba(255, 255, 255, 0.03);
		border: 1px solid rgba(255, 255, 255, 0.06);
		border-radius: 10px;
		padding: 0.75rem;
	}

	.v-label {
		font-size: 0.7rem;
		color: #64748b;
		margin-bottom: 0.2rem;
	}

	.v-val {
		font-size: 0.88rem;
		font-weight: 700;
		color: #f1f5f9;
	}

	.v-val.bold { color: #38bdf8; }

	.v-sub {
		font-size: 0.74rem;
		color: #94a3b8;
		margin-top: 0.15rem;
	}

	.voucher-finance {
		background: rgba(2, 132, 199, 0.1);
		border-radius: 10px;
		padding: 0.85rem 1rem;
		margin-bottom: 1.5rem;
	}

	.f-item {
		display: flex;
		justify-content: space-between;
		font-size: 0.85rem;
		color: #cbd5e1;
		margin-bottom: 0.35rem;
	}

	.f-item.highlight {
		color: #34d399;
	}

	.f-item.balance {
		padding-top: 0.35rem;
		border-top: 1px dashed rgba(255, 255, 255, 0.1);
		color: #94a3b8;
		font-size: 0.8rem;
	}

	.voucher-actions {
		display: flex;
		flex-direction: column;
		gap: 0.65rem;
	}

	.v-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.5rem;
		padding: 0.8rem;
		border-radius: 10px;
		font-weight: 700;
		font-size: 0.92rem;
		text-decoration: none;
		cursor: pointer;
		border: none;
	}

	.v-btn.dashboard {
		background: linear-gradient(135deg, #0284c7 0%, #2563eb 100%);
		color: #ffffff;
	}

	.v-btn.wa {
		background: #25d366;
		color: #ffffff;
	}

	.v-btn.print {
		background: rgba(255, 255, 255, 0.08);
		color: #ffffff;
	}

	@media (max-width: 1024px) {
		.booking-grid {
			grid-template-columns: 1fr;
		}
		.pkg-select-item {
			grid-template-columns: 1fr;
		}
		.form-row-3, .tier-options-grid, .vehicle-options-grid, .summary-specs-grid {
			grid-template-columns: 1fr;
		}
	}
</style>
