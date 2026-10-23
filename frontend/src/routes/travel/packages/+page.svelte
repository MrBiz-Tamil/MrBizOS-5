<script lang="ts">
	import { toast } from '$lib/stores/toast.svelte';

	interface ItineraryDay {
		dayNumber: number;
		title: string;
		highlights: string;
		stayHotel: string;
		mealsIncluded: string;
		cabType: string;
	}

	interface TourPackage {
		id: string;
		title: string;
		circuit: string;
		regionCode: 'kashmir' | 'kerala' | 'malaysia' | 'dubai' | 'tamilnadu';
		durationDays: number;
		durationNights: number;
		tourStyle: 'Family Tour' | 'Honeymoon Couple' | 'Group Tour' | 'Business / Corporate' | 'Restmode / Leisure';
		thumbnail: string;
		badge: string;
		// Season-wise pricing per pax (twin sharing)
		pricing: {
			peakSeason: number; // e.g. Summer/Dec/Diwali
			regularSeason: number;
			offPeakSeason: number;
			currency: string;
		};
		// Cost Rollup Breakdown (Regular Season)
		costRollup: {
			hotelCost: number;
			cabCost: number;
			guideCost: number;
			sightseeingCost: number;
			visaAndInsurance: number;
			agencyMargin: number;
			totalSellingPrice: number;
		};
		inclusions: string[];
		exclusions: string[];
		dayWiseItinerary: ItineraryDay[];
		departuresCount: number;
		status: 'active' | 'draft' | 'seasonal_closed';
	}

	// Filter state
	let searchQuery = $state('');
	let filterRegion = $state('all');
	let filterStyle = $state('all');
	let selectedSeason = $state<'peak' | 'regular' | 'offpeak'>('regular');

	// Modals
	let showCreatePackageModal = $state(false);
	let selectedPackageForDetail = $state<TourPackage | null>(null);

	// Dataset of Tour Packages
	let packages = $state<TourPackage[]>([
		{
			id: 'pkg-kashmir-1',
			title: 'Kashmir Paradise: Snow Trails & Dal Lake Shikara',
			circuit: 'Srinagar • Gulmarg • Pahalgam • Sonamarg',
			regionCode: 'kashmir',
			durationDays: 4,
			durationNights: 5,
			tourStyle: 'Family Tour',
			thumbnail: '🏔️',
			badge: 'Bestseller in Chennai & TN',
			pricing: {
				peakSeason: 38500,
				regularSeason: 29500,
				offPeakSeason: 24000,
				currency: 'INR'
			},
			costRollup: {
				hotelCost: 12000,
				cabCost: 7500,
				guideCost: 2000,
				sightseeingCost: 2000,
				visaAndInsurance: 1000,
				agencyMargin: 5000,
				totalSellingPrice: 29500
			},
			inclusions: [
				'4 Nights Deluxe Hotel + 1 Night Houseboat Stay on Dal Lake',
				'Dedicated Innova Crysta throughout with heater & fuel',
				'Daily Breakfast & Kashmiri Wazwan Dinner',
				'Shikara Sunrise Ride & Gondola Phase-1 Ticket',
				'Govt Certified Local Guide at Pahalgam & Gulmarg'
			],
			exclusions: ['Airfare Chennai/Madurai to Srinagar', 'Pony / Horse ride at Baisaran', 'Personal shopping'],
			dayWiseItinerary: [
				{
					dayNumber: 1,
					title: 'Arrival in Srinagar & Dal Lake Shikara Ride',
					highlights: 'Airport pickup in Innova, check-in to Luxury Houseboat, evening 2-hour Shikara ride on Dal Lake.',
					stayHotel: 'Mascot Luxury Houseboat, Dal Lake',
					mealsIncluded: 'Dinner',
					cabType: 'Innova Crysta'
				},
				{
					dayNumber: 2,
					title: 'Srinagar to Gulmarg Day Excursion',
					highlights: 'Drive across pine forests, ride the famous Gulmarg Gondola, snow activities at Kongdoori.',
					stayHotel: 'Grand Mumtaz Srinagar',
					mealsIncluded: 'Breakfast, Dinner',
					cabType: 'Innova Crysta'
				},
				{
					dayNumber: 3,
					title: 'Srinagar to Pahalgam Valley of Shepherds',
					highlights: 'Visit Saffron fields of Pampore, Awantipora Ruins, Lidder River, and Betaab Valley.',
					stayHotel: 'Pahalgam Pine Spring Resort',
					mealsIncluded: 'Breakfast, Dinner',
					cabType: 'Innova Crysta'
				},
				{
					dayNumber: 4,
					title: 'Mughal Gardens Heritage Walk & Departure Drop',
					highlights: 'Shalimar Bagh, Nishat Bagh, Shankaracharya Temple darshan, departure drop to Srinagar airport.',
					stayHotel: 'Departure Day',
					mealsIncluded: 'Breakfast',
					cabType: 'Innova Crysta'
				}
			],
			departuresCount: 18,
			status: 'active'
		},
		{
			id: 'pkg-kerala-2',
			title: 'Gods Own Country: Munnar Tea Hills & Alleppey Houseboat',
			circuit: 'Cochin • Munnar • Thekkady • Alleppey Backwaters',
			regionCode: 'kerala',
			durationDays: 5,
			durationNights: 4,
			tourStyle: 'Restmode / Leisure',
			thumbnail: '🌴',
			badge: 'Popular for Madurai & TN Families',
			pricing: {
				peakSeason: 32000,
				regularSeason: 24500,
				offPeakSeason: 19500,
				currency: 'INR'
			},
			costRollup: {
				hotelCost: 10500,
				cabCost: 6500,
				guideCost: 1500,
				sightseeingCost: 1200,
				visaAndInsurance: 800,
				agencyMargin: 4000,
				totalSellingPrice: 24500
			},
			inclusions: [
				'2 Nights Munnar Tea Resort + 1 Night Thekkady + 1 Night Private Houseboat',
				'Private AC Dzire / Ertiga Cab from Madurai or Cochin pickup',
				'All Meals on Houseboat (Authentic Kerala Sadhya & Karimeen fry)',
				'Spice Plantation Tour with English/Tamil speaking guide'
			],
			exclusions: ['Train / Flight to Cochin', 'Elephant safari', 'Kathakali cultural entry tickets'],
			dayWiseItinerary: [
				{
					dayNumber: 1,
					title: 'Pickup from Cochin / Madurai & Drive to Munnar',
					highlights: 'Valara & Cheeyappara waterfalls view, check-in to tea garden hillside resort.',
					stayHotel: 'Amber Dale Luxury Tea Valley Resort',
					mealsIncluded: 'Dinner',
					cabType: 'Ertiga AC'
				},
				{
					dayNumber: 2,
					title: 'Munnar Sightseeing & Eravikulam National Park',
					highlights: 'Nilgiri Tahr sighting, Tea Museum, Mattupetty Dam boating, Echo Point.',
					stayHotel: 'Amber Dale Luxury Tea Valley Resort',
					mealsIncluded: 'Breakfast, Dinner',
					cabType: 'Ertiga AC'
				},
				{
					dayNumber: 3,
					title: 'Munnar to Thekkady Periyar Wildlife',
					highlights: 'Spice plantation guided walk, Periyar lake boat safari, spice shopping.',
					stayHotel: 'Greenwoods Resort Thekkady',
					mealsIncluded: 'Breakfast, Dinner',
					cabType: 'Ertiga AC'
				},
				{
					dayNumber: 4,
					title: 'Thekkady to Alleppey Luxury Houseboat Cruise',
					highlights: 'Boarding cruise at 12 PM, cruising Vembanad lake, backwater village life, sunset dinner.',
					stayHotel: 'Lakes & Lagoons AC Deluxe Houseboat',
					mealsIncluded: 'Lunch, Evening Snacks, Dinner',
					cabType: 'Ertiga AC'
				},
				{
					dayNumber: 5,
					title: 'Alleppey Checkout & Drop to Cochin Airport / Madurai',
					highlights: 'Traditional Kerala breakfast, drive back to Cochin or Madurai drop.',
					stayHotel: 'Departure Day',
					mealsIncluded: 'Breakfast',
					cabType: 'Ertiga AC'
				}
			],
			departuresCount: 24,
			status: 'active'
		},
		{
			id: 'pkg-malaysia-3',
			title: 'Malaysia Wonder: KL Twin Towers & Genting Highlands',
			circuit: 'Kuala Lumpur • Batu Caves • Genting Highlands • Putrajaya',
			regionCode: 'malaysia',
			durationDays: 4,
			durationNights: 5,
			tourStyle: 'Group Tour',
			thumbnail: '🏙️',
			badge: 'Includes eVisa & Cable Car',
			pricing: {
				peakSeason: 58000,
				regularSeason: 46000,
				offPeakSeason: 39500,
				currency: 'INR'
			},
			costRollup: {
				hotelCost: 18000,
				cabCost: 9000,
				guideCost: 4000,
				sightseeingCost: 4500,
				visaAndInsurance: 3500,
				agencyMargin: 7000,
				totalSellingPrice: 46000
			},
			inclusions: [
				'4 Nights at 4-Star Hotel in Bukit Bintang, Kuala Lumpur',
				'Malaysia eVisa processing & travel insurance included',
				'Return Airport transfers & full day Genting tour with Awana SkyWay Cable Car',
				'KL City Tour: Petronas Towers photo stop, King Palace, National Mosque',
				'Batu Caves Lord Murugan Temple tour with Tamil speaking tour manager'
			],
			exclusions: ['Air ticket Chennai to KL (Optional add-on ₹16,500 return)', 'Tourism Tax RM10/room/night'],
			dayWiseItinerary: [
				{
					dayNumber: 1,
					title: 'Arrival KLIA Airport & Bukit Bintang Check-in',
					highlights: 'Meet & greet by tour manager, transfer to hotel, evening Jalan Alor street food walk.',
					stayHotel: 'PARKROYAL Serviced Suites KL',
					mealsIncluded: 'Dinner',
					cabType: 'Executive Coach'
				},
				{
					dayNumber: 2,
					title: 'Batu Caves Pilgrimage & Genting Highlands Cable Car',
					highlights: '140-ft Murugan statue darshan, Awana Skyway glass-floor cable car, Genting SkyWorlds.',
					stayHotel: 'PARKROYAL Serviced Suites KL',
					mealsIncluded: 'Breakfast, Dinner',
					cabType: 'Executive Coach'
				},
				{
					dayNumber: 3,
					title: 'Putrajaya Cruise & KL City Highlights',
					highlights: 'Pink Mosque, Putra Square lake cruise, Petronas Twin Towers skybridge visit.',
					stayHotel: 'PARKROYAL Serviced Suites KL',
					mealsIncluded: 'Breakfast, Lunch',
					cabType: 'Executive Coach'
				},
				{
					dayNumber: 4,
					title: 'Shopping at Pavilion Mall & KLIA Departure Drop',
					highlights: 'Souvenir shopping, duty-free coordination, airport drop for return flight.',
					stayHotel: 'Departure Day',
					mealsIncluded: 'Breakfast',
					cabType: 'Executive Coach'
				}
			],
			departuresCount: 12,
			status: 'active'
		},
		{
			id: 'pkg-dubai-4',
			title: 'Dubai Glamour: Burj Khalifa, Desert Safari & Marina Cruise',
			circuit: 'Downtown Dubai • Marina • Red Dunes Desert • Abu Dhabi',
			regionCode: 'dubai',
			durationDays: 5,
			durationNights: 4,
			tourStyle: 'Family Tour',
			thumbnail: '🏜️',
			badge: 'VIP Desert Camp & 4x4 Dune Bashing',
			pricing: {
				peakSeason: 78000,
				regularSeason: 62000,
				offPeakSeason: 51000,
				currency: 'INR'
			},
			costRollup: {
				hotelCost: 26000,
				cabCost: 11000,
				guideCost: 5000,
				sightseeingCost: 8000,
				visaAndInsurance: 4000,
				agencyMargin: 8000,
				totalSellingPrice: 62000
			},
			inclusions: [
				'4 Nights at 4-Star Bur Dubai / Downtown Hotel with Breakfast',
				'UAE 30-Day Tourist Visa & Medical Insurance included',
				'Burj Khalifa 124th Floor At the Top ticket with Dubai Mall Fountain show',
				'Red Dune Desert Safari with BBQ Dinner, Tanoura dance & Camel ride',
				'Luxury Dhow Cruise Dinner at Dubai Marina with live music'
			],
			exclusions: ['Airfare Chennai/Trichy/Madurai to Dubai', 'Personal expenses & Tourism Dirham fee'],
			dayWiseItinerary: [
				{
					dayNumber: 1,
					title: 'Arrival in Dubai & Marina Dhow Cruise Dinner',
					highlights: 'Airport pickup in private GMC, check-in, 8 PM Marina luxury Dhow dinner cruise.',
					stayHotel: 'Four Points by Sheraton Downtown',
					mealsIncluded: 'Dinner',
					cabType: 'Private GMC / Van'
				},
				{
					dayNumber: 2,
					title: 'Burj Khalifa Skydeck & Dubai Mall Aquarium',
					highlights: '124th floor panoramic observation deck, Dubai Mall shopping, evening musical fountains.',
					stayHotel: 'Four Points by Sheraton Downtown',
					mealsIncluded: 'Breakfast',
					cabType: 'Private GMC / Van'
				},
				{
					dayNumber: 3,
					title: 'Red Dunes 4x4 Desert Safari with BBQ Night',
					highlights: 'Afternoon pickup, thrilling 45-min dune bashing, sandboarding, falcon photo, BBQ buffet.',
					stayHotel: 'Four Points by Sheraton Downtown',
					mealsIncluded: 'Breakfast, BBQ Dinner',
					cabType: 'Land Cruiser 4x4'
				},
				{
					dayNumber: 4,
					title: 'Old Dubai Gold Souk & Departure Drop',
					highlights: 'Abra water taxi crossing, Gold & Spice Souk bargaining, departure drop to DXB airport.',
					stayHotel: 'Departure Day',
					mealsIncluded: 'Breakfast',
					cabType: 'Private GMC / Van'
				}
			],
			departuresCount: 15,
			status: 'active'
		}
	]);

	// Filtered Packages
	let filteredPackages = $derived(
		packages.filter((p) => {
			const matchesQuery =
				searchQuery === '' ||
				p.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
				p.circuit.toLowerCase().includes(searchQuery.toLowerCase()) ||
				p.tourStyle.toLowerCase().includes(searchQuery.toLowerCase());

			const matchesRegion = filterRegion === 'all' || p.regionCode === filterRegion;
			const matchesStyle = filterStyle === 'all' || p.tourStyle === filterStyle;

			return matchesQuery && matchesRegion && matchesStyle;
		})
	);

	// New Package Form State
	let newPkg = $state({
		title: '',
		circuit: '',
		regionCode: 'kashmir' as TourPackage['regionCode'],
		durationDays: 4,
		durationNights: 5,
		tourStyle: 'Family Tour' as TourPackage['tourStyle'],
		peakPrice: 38000,
		regularPrice: 29000,
		offPeakPrice: 23000,
		hotelCost: 12000,
		cabCost: 7000,
		guideCost: 2000,
		sightseeingCost: 2000,
		visaCost: 1000,
		margin: 5000,
		inclusionsInput: 'Deluxe Hotels, Private Cab with Driver, Daily Breakfast & Dinner, Sightseeing',
		exclusionsInput: 'Airfare, Personal shopping, Tips'
	});

	// Automatic Rollup in Form
	let computedRegularSelling = $derived(
		Number(newPkg.hotelCost) +
		Number(newPkg.cabCost) +
		Number(newPkg.guideCost) +
		Number(newPkg.sightseeingCost) +
		Number(newPkg.visaCost) +
		Number(newPkg.margin)
	);

	function handleSavePackage() {
		if (!newPkg.title || !newPkg.circuit) {
			toast.error('Please enter Tour Package Title and Circuit Details');
			return;
		}

		const pkg: TourPackage = {
			id: `pkg-${Date.now()}`,
			title: newPkg.title,
			circuit: newPkg.circuit,
			regionCode: newPkg.regionCode,
			durationDays: Number(newPkg.durationDays),
			durationNights: Number(newPkg.durationNights),
			tourStyle: newPkg.tourStyle,
			thumbnail: '✈️',
			badge: 'Custom Package',
			pricing: {
				peakSeason: Number(newPkg.peakPrice),
				regularSeason: computedRegularSelling,
				offPeakSeason: Number(newPkg.offPeakPrice),
				currency: 'INR'
			},
			costRollup: {
				hotelCost: Number(newPkg.hotelCost),
				cabCost: Number(newPkg.cabCost),
				guideCost: Number(newPkg.guideCost),
				sightseeingCost: Number(newPkg.sightseeingCost),
				visaAndInsurance: Number(newPkg.visaCost),
				agencyMargin: Number(newPkg.margin),
				totalSellingPrice: computedRegularSelling
			},
			inclusions: newPkg.inclusionsInput.split(',').map((s) => s.trim()).filter(Boolean),
			exclusions: newPkg.exclusionsInput.split(',').map((s) => s.trim()).filter(Boolean),
			dayWiseItinerary: [
				{
					dayNumber: 1,
					title: 'Arrival & Destination Orientation',
					highlights: 'Airport pickup in private vehicle, hotel check-in, orientation dinner.',
					stayHotel: 'Grand Deluxe Resort',
					mealsIncluded: 'Dinner',
					cabType: 'Dedicated Cab'
				},
				{
					dayNumber: 2,
					title: 'Key Sightseeing & Guided Heritage Walk',
					highlights: 'Full day sightseeing with certified local tour guide.',
					stayHotel: 'Grand Deluxe Resort',
					mealsIncluded: 'Breakfast, Dinner',
					cabType: 'Dedicated Cab'
				}
			],
			departuresCount: 6,
			status: 'active'
		};

		packages = [pkg, ...packages];
		showCreatePackageModal = false;
		toast.success(`Created Tour Package "${pkg.title}" with Cost Rollup ₹${computedRegularSelling.toLocaleString()}`);

		// Reset form
		newPkg = {
			title: '',
			circuit: '',
			regionCode: 'kashmir',
			durationDays: 4,
			durationNights: 5,
			tourStyle: 'Family Tour',
			peakPrice: 38000,
			regularPrice: 29000,
			offPeakPrice: 23000,
			hotelCost: 12000,
			cabCost: 7000,
			guideCost: 2000,
			sightseeingCost: 2000,
			visaCost: 1000,
			margin: 5000,
			inclusionsInput: 'Deluxe Hotels, Private Cab with Driver, Daily Breakfast & Dinner, Sightseeing',
			exclusionsInput: 'Airfare, Personal shopping, Tips'
		};
	}

	function getBrochureWhatsAppUrl(pkg: TourPackage) {
		const text = encodeURIComponent(
			`*MRBIZ OS TOUR BROCHURE*\n\n` +
			`🌴 *${pkg.title}*\n` +
			`📍 Circuit: ${pkg.circuit}\n` +
			`⏳ Duration: ${pkg.durationDays} Days / ${pkg.durationNights} Nights (${pkg.tourStyle})\n` +
			`💰 Regular Price: ₹${pkg.pricing.regularSeason.toLocaleString()} per pax (Twin Sharing)\n` +
			`🔥 Peak Season: ₹${pkg.pricing.peakSeason.toLocaleString()} | 🌧️ Off-Peak: ₹${pkg.pricing.offPeakSeason.toLocaleString()}\n\n` +
			`✨ *Key Inclusions:*\n` +
			pkg.inclusions.slice(0, 3).map((inc) => `• ${inc}`).join('\n') +
			`\n\nContact our tour desk to book now or customize this itinerary!`
		);
		return `https://wa.me/?text=${text}`;
	}
</script>

<div class="packages-root">
	<!-- Top Header -->
	<div class="header-card">
		<div class="title-cluster">
			<div class="title-badge">
				<span class="badge-icon">📦</span>
				<span class="badge-label">Tour Package Designer & Costing Engine</span>
			</div>
			<h1 class="main-title">Tour Package Inventory & Itinerary Costing</h1>
			<p class="subtitle">
				Design custom tour products for <strong>Kashmir</strong>, <strong>Kerala</strong>, <strong>Malaysia</strong>, <strong>Dubai</strong>, and <strong>Tamil Nadu</strong>. Calculate day-wise cost rollups (Cab + Resort + Guide + Visa + Profit Margin) and toggle season-wise pricing tiers.
			</p>
		</div>

		<div class="header-actions">
			<a href="/travel/book" class="action-btn secondary">
				<span>⚡</span>
				<span>1-Page Booking Desk</span>
			</a>
			<button class="action-btn primary" onclick={() => (showCreatePackageModal = true)}>
				<span>+</span>
				<span>Design Tour Package</span>
			</button>
		</div>
	</div>

	<!-- Toolbar & Season Switcher -->
	<div class="toolbar-card">
		<div class="search-input-wrap">
			<span class="search-icon">🔍</span>
			<input
				type="text"
				placeholder="Search package by title, circuit, or style (e.g. Family, Honeymoon)..."
				bind:value={searchQuery}
				class="search-input"
			/>
		</div>

		<div class="filter-controls">
			<!-- Season Price Selector -->
			<div class="season-toggle-wrap">
				<span class="toggle-label">Active Season Tariff:</span>
				<div class="toggle-buttons">
					<button
						class="season-btn {selectedSeason === 'peak' ? 'active peak' : ''}"
						onclick={() => (selectedSeason = 'peak')}
					>
						🔥 Peak Season
					</button>
					<button
						class="season-btn {selectedSeason === 'regular' ? 'active regular' : ''}"
						onclick={() => (selectedSeason = 'regular')}
					>
						☀️ Regular
					</button>
					<button
						class="season-btn {selectedSeason === 'offpeak' ? 'active offpeak' : ''}"
						onclick={() => (selectedSeason = 'offpeak')}
					>
						🌧️ Off-Peak
					</button>
				</div>
			</div>

			<!-- Region Filter -->
			<div class="select-wrap">
				<label for="filter-circuit">Circuit:</label>
				<select id="filter-circuit" bind:value={filterRegion} class="filter-select">
					<option value="all">All Destinations</option>
					<option value="kashmir">🏔️ Kashmir Paradise</option>
					<option value="kerala">🌴 Kerala Backwaters</option>
					<option value="malaysia">🏙️ Malaysia & Genting</option>
					<option value="dubai">🏜️ Dubai & Emirates</option>
				</select>
			</div>

			<!-- Style Filter -->
			<div class="select-wrap">
				<label for="filter-style">Trip Style:</label>
				<select id="filter-style" bind:value={filterStyle} class="filter-select">
					<option value="all">All Styles</option>
					<option value="Family Tour">Family Tour</option>
					<option value="Honeymoon Couple">Honeymoon Couple</option>
					<option value="Group Tour">Group Tour</option>
					<option value="Restmode / Leisure">Restmode / Leisure</option>
				</select>
			</div>
		</div>
	</div>

	<!-- Tour Packages Grid -->
	<div class="packages-grid">
		{#if filteredPackages.length === 0}
			<div class="empty-state">
				<span class="empty-icon">🗺️</span>
				<h3>No tour packages found</h3>
				<p>Try searching another destination or resetting your filter criteria.</p>
				<button class="action-btn secondary" onclick={() => { searchQuery = ''; filterRegion = 'all'; filterStyle = 'all'; }}>
					Reset Filters
				</button>
			</div>
		{:else}
			{#each filteredPackages as pkg (pkg.id)}
				<div class="package-card">
					<!-- Top Banner -->
					<div class="pkg-header">
						<div class="thumb-badge-wrap">
							<span class="pkg-thumb">{pkg.thumbnail}</span>
							<span class="feature-badge">{pkg.badge}</span>
						</div>
						<div class="pkg-style-tag">{pkg.tourStyle}</div>
					</div>

					<!-- Title & Circuit -->
					<div class="pkg-body">
						<h2 class="pkg-title">{pkg.title}</h2>
						<div class="pkg-circuit">
							<span class="circuit-pin">📍</span>
							<span class="circuit-names">{pkg.circuit}</span>
						</div>
						<div class="pkg-duration">
							<span>⏳ {pkg.durationDays} Days / {pkg.durationNights} Nights</span>
							<span>•</span>
							<span>🛫 {pkg.departuresCount} Departures Left</span>
						</div>

						<!-- Live Pricing Block -->
						<div class="price-display-box {selectedSeason}">
							<div class="price-left">
								<span class="price-season-label">
									{selectedSeason === 'peak' ? 'Peak Season Rate' : selectedSeason === 'regular' ? 'Regular Season Rate' : 'Off-Peak Discount Rate'}
								</span>
								<div class="price-val">
									{#if selectedSeason === 'peak'}
										₹{pkg.pricing.peakSeason.toLocaleString()}
									{:else if selectedSeason === 'regular'}
										₹{pkg.pricing.regularSeason.toLocaleString()}
									{:else}
										₹{pkg.pricing.offPeakSeason.toLocaleString()}
									{/if}
									<span class="pax-tag">/ pax</span>
								</div>
							</div>
							<div class="price-right">
								<span class="margin-label">Agency Margin</span>
								<span class="margin-val">+₹{pkg.costRollup.agencyMargin.toLocaleString()}</span>
							</div>
						</div>

						<!-- Cost Rollup Breakdown -->
						<div class="cost-rollup-list">
							<div class="cr-row">
								<span>🏨 Hotel & Meals: ₹{pkg.costRollup.hotelCost.toLocaleString()}</span>
								<span>🚗 Cab & Driver: ₹{pkg.costRollup.cabCost.toLocaleString()}</span>
							</div>
							<div class="cr-row">
								<span>🧭 Guide Batta: ₹{pkg.costRollup.guideCost.toLocaleString()}</span>
								<span>🎟️ Tickets/Visa: ₹{(pkg.costRollup.sightseeingCost + pkg.costRollup.visaAndInsurance).toLocaleString()}</span>
							</div>
						</div>

						<!-- Inclusions preview -->
						<div class="inclusions-preview">
							<span class="inc-title">Key Inclusions:</span>
							<ul class="inc-list">
								{#each pkg.inclusions.slice(0, 3) as inc}
									<li>✓ {inc}</li>
								{/each}
							</ul>
						</div>
					</div>

					<!-- Footer Actions -->
					<div class="card-footer">
						<button class="action-btn-sm details" onclick={() => (selectedPackageForDetail = pkg)}>
							<span>🗺️</span>
							<span>Day-Wise Plan</span>
						</button>

						<a
							href={getBrochureWhatsAppUrl(pkg)}
							target="_blank"
							rel="noreferrer"
							class="action-btn-sm whatsapp"
						>
							<span>💬</span>
							<span>Brochure</span>
						</a>

						<a href="/travel/book" class="action-btn-sm book">
							<span>⚡</span>
							<span>Book Trip</span>
						</a>
					</div>
				</div>
			{/each}
		{/if}
	</div>
</div>

<!-- Modal: Design Tour Package & Costing Engine -->
{#if showCreatePackageModal}
	<div class="modal-backdrop">
		<div class="modal-box">
			<div class="modal-header">
				<h2>📦 Design Tour Package & Costing Engine</h2>
				<button class="close-btn" onclick={() => (showCreatePackageModal = false)}>✕</button>
			</div>

			<div class="modal-body">
				<div class="form-row">
					<div class="form-group flex-2">
						<label for="pkg-title">Package Name / Headline *</label>
						<input id="pkg-title" type="text" placeholder="e.g. Kashmir 4D/5N Family Escape" bind:value={newPkg.title} class="form-input" />
					</div>
					<div class="form-group flex-1">
						<label for="pkg-region">Circuit Region *</label>
						<select id="pkg-region" bind:value={newPkg.regionCode} class="form-input">
							<option value="kashmir">🏔️ Kashmir Paradise</option>
							<option value="kerala">🌴 Kerala Backwaters</option>
							<option value="malaysia">🏙️ Malaysia & Genting</option>
							<option value="dubai">🏜️ Dubai & Emirates</option>
							<option value="tamilnadu">🛕 Tamil Nadu Heritage</option>
						</select>
					</div>
				</div>

				<div class="form-row">
					<div class="form-group flex-2">
						<label for="pkg-circuit">Cities & Circuit Route *</label>
						<input id="pkg-circuit" type="text" placeholder="e.g. Srinagar • Gulmarg • Pahalgam" bind:value={newPkg.circuit} class="form-input" />
					</div>
					<div class="form-group flex-1">
						<label for="pkg-style">Tour Style</label>
						<select id="pkg-style" bind:value={newPkg.tourStyle} class="form-input">
							<option value="Family Tour">Family Tour</option>
							<option value="Honeymoon Couple">Honeymoon Couple</option>
							<option value="Group Tour">Group Tour</option>
							<option value="Business / Corporate">Business / Corporate</option>
							<option value="Restmode / Leisure">Restmode / Leisure</option>
						</select>
					</div>
				</div>

				<div class="form-row">
					<div class="form-group flex-1">
						<label for="pkg-days">Duration Days</label>
						<input id="pkg-days" type="number" bind:value={newPkg.durationDays} class="form-input" />
					</div>
					<div class="form-group flex-1">
						<label for="pkg-nights">Duration Nights</label>
						<input id="pkg-nights" type="number" bind:value={newPkg.durationNights} class="form-input" />
					</div>
					<div class="form-group flex-1">
						<label for="pkg-peak">Peak Season Tariff (₹)</label>
						<input id="pkg-peak" type="number" bind:value={newPkg.peakPrice} class="form-input" />
					</div>
					<div class="form-group flex-1">
						<label for="pkg-offpeak">Off-Peak Tariff (₹)</label>
						<input id="pkg-offpeak" type="number" bind:value={newPkg.offPeakPrice} class="form-input" />
					</div>
				</div>

				<!-- Cost Rollup Engine -->
				<div class="costing-engine-panel">
					<h3 class="engine-title">⚙️ Day-Wise Cost Rollup (Per Pax Baseline)</h3>
					<div class="engine-grid">
						<div class="engine-item">
							<label for="eng-hotel">Hotel & Stay (₹)</label>
							<input id="eng-hotel" type="number" bind:value={newPkg.hotelCost} class="engine-input" />
						</div>
						<div class="engine-item">
							<label for="eng-cab">Cab & Driver (₹)</label>
							<input id="eng-cab" type="number" bind:value={newPkg.cabCost} class="engine-input" />
						</div>
						<div class="engine-item">
							<label for="eng-guide">Guide Batta (₹)</label>
							<input id="eng-guide" type="number" bind:value={newPkg.guideCost} class="engine-input" />
						</div>
						<div class="engine-item">
							<label for="eng-tickets">Sightseeing/Activities (₹)</label>
							<input id="eng-tickets" type="number" bind:value={newPkg.sightseeingCost} class="engine-input" />
						</div>
						<div class="engine-item">
							<label for="eng-visa">Visa & Insurance (₹)</label>
							<input id="eng-visa" type="number" bind:value={newPkg.visaCost} class="engine-input" />
						</div>
						<div class="engine-item">
							<label for="eng-margin">Agency Markup / Profit (₹)</label>
							<input id="eng-margin" type="number" bind:value={newPkg.margin} class="engine-input highlight" />
						</div>
					</div>

					<div class="computed-price-bar">
						<span class="cp-label">Computed Regular Season Price:</span>
						<span class="cp-amount">₹{computedRegularSelling.toLocaleString()} <small>/ pax</small></span>
					</div>
				</div>

				<div class="form-group">
					<label for="pkg-inclusions">Package Inclusions (comma separated)</label>
					<textarea id="pkg-inclusions" rows="2" bind:value={newPkg.inclusionsInput} class="form-input"></textarea>
				</div>
			</div>

			<div class="modal-footer">
				<button class="action-btn secondary" onclick={() => (showCreatePackageModal = false)}>Cancel</button>
				<button class="action-btn primary" onclick={handleSavePackage}>Save & Publish Package</button>
			</div>
		</div>
	</div>
{/if}

<!-- Drawer: Day-Wise Itinerary Plan View -->
{#if selectedPackageForDetail}
	<div class="modal-backdrop">
		<div class="modal-box">
			<div class="modal-header">
				<h2>🗺️ {selectedPackageForDetail.title} — Day-Wise Itinerary</h2>
				<button class="close-btn" onclick={() => (selectedPackageForDetail = null)}>✕</button>
			</div>

			<div class="modal-body">
				<div class="itinerary-timeline">
					{#each selectedPackageForDetail.dayWiseItinerary as day}
						<div class="timeline-step">
							<div class="step-badge">Day {day.dayNumber}</div>
							<div class="step-card">
								<h3 class="step-title">{day.title}</h3>
								<p class="step-highlights">{day.highlights}</p>
								<div class="step-meta">
									<span>🏨 Stay: <strong>{day.stayHotel}</strong></span>
									<span>🍽️ Meals: <strong>{day.mealsIncluded}</strong></span>
									<span>🚗 Cab: <strong>{day.cabType}</strong></span>
								</div>
							</div>
						</div>
					{/each}
				</div>
			</div>

			<div class="modal-footer">
				<button class="action-btn secondary" onclick={() => (selectedPackageForDetail = null)}>Close Itinerary</button>
				<a href="/travel/book" class="action-btn primary">
					<span>⚡</span>
					<span>Book This Itinerary</span>
				</a>
			</div>
		</div>
	</div>
{/if}

<style>
	.packages-root {
		display: flex;
		flex-direction: column;
		gap: 1.25rem;
		padding: 1.5rem;
		max-width: 1440px;
		margin: 0 auto;
	}

	.header-card {
		background: #0f172a;
		border: 1px solid #1e293b;
		border-radius: 1rem;
		padding: 1.5rem 2rem;
		display: flex;
		justify-content: space-between;
		align-items: center;
		flex-wrap: wrap;
		gap: 1.25rem;
	}

	.title-cluster {
		max-width: 820px;
	}

	.title-badge {
		display: inline-flex;
		align-items: center;
		gap: 0.5rem;
		background: rgba(14, 165, 233, 0.15);
		border: 1px solid rgba(14, 165, 233, 0.3);
		color: #38bdf8;
		padding: 0.25rem 0.75rem;
		border-radius: 9999px;
		font-size: 0.75rem;
		font-weight: 600;
		text-transform: uppercase;
		margin-bottom: 0.5rem;
	}

	.main-title {
		font-size: 1.6rem;
		font-weight: 800;
		color: #f8fafc;
		margin: 0 0 0.5rem 0;
	}

	.subtitle {
		font-size: 0.875rem;
		color: #94a3b8;
		line-height: 1.5;
		margin: 0;
	}

	.header-actions {
		display: flex;
		gap: 0.75rem;
		align-items: center;
	}

	.action-btn {
		display: inline-flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.65rem 1.25rem;
		border-radius: 0.5rem;
		font-weight: 600;
		font-size: 0.875rem;
		cursor: pointer;
		text-decoration: none;
		transition: all 0.2s ease;
		border: none;
	}
	.action-btn.primary {
		background: #0ea5e9;
		color: #ffffff;
	}
	.action-btn.primary:hover {
		background: #0284c7;
	}
	.action-btn.secondary {
		background: #1e293b;
		color: #e2e8f0;
		border: 1px solid #334155;
	}
	.action-btn.secondary:hover {
		background: #334155;
	}

	/* Toolbar & Season selector */
	.toolbar-card {
		background: #0f172a;
		border: 1px solid #1e293b;
		border-radius: 0.75rem;
		padding: 1rem 1.25rem;
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
	}

	.search-input-wrap {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		background: #1e293b;
		border: 1px solid #334155;
		border-radius: 0.5rem;
		padding: 0.5rem 0.75rem;
		flex: 1;
		min-width: 260px;
	}
	.search-icon {
		opacity: 0.7;
	}
	.search-input {
		background: transparent;
		border: none;
		outline: none;
		color: #f8fafc;
		font-size: 0.875rem;
		width: 100%;
	}

	.filter-controls {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 1rem;
	}

	.season-toggle-wrap {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}
	.toggle-label {
		font-size: 0.8125rem;
		color: #94a3b8;
		font-weight: 500;
	}
	.toggle-buttons {
		display: flex;
		background: #1e293b;
		border-radius: 0.5rem;
		padding: 0.2rem;
		border: 1px solid #334155;
	}
	.season-btn {
		background: transparent;
		border: none;
		color: #94a3b8;
		font-size: 0.75rem;
		font-weight: 600;
		padding: 0.35rem 0.65rem;
		border-radius: 0.35rem;
		cursor: pointer;
		transition: all 0.15s ease;
	}
	.season-btn.active.peak {
		background: #ef4444;
		color: #ffffff;
	}
	.season-btn.active.regular {
		background: #0ea5e9;
		color: #ffffff;
	}
	.season-btn.active.offpeak {
		background: #10b981;
		color: #ffffff;
	}

	.select-wrap {
		display: flex;
		align-items: center;
		gap: 0.4rem;
		font-size: 0.8125rem;
		color: #94a3b8;
	}
	.filter-select {
		background: #1e293b;
		border: 1px solid #334155;
		color: #f8fafc;
		font-size: 0.8125rem;
		padding: 0.45rem 0.65rem;
		border-radius: 0.375rem;
		outline: none;
	}

	/* Packages Grid */
	.packages-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(420px, 1fr));
		gap: 1.25rem;
	}

	.package-card {
		background: #0f172a;
		border: 1px solid #1e293b;
		border-radius: 0.875rem;
		overflow: hidden;
		display: flex;
		flex-direction: column;
		transition: all 0.2s ease;
	}
	.package-card:hover {
		border-color: #38bdf8;
		box-shadow: 0 4px 20px rgba(14, 165, 233, 0.15);
	}

	.pkg-header {
		background: #1e293b;
		padding: 0.75rem 1.25rem;
		display: flex;
		justify-content: space-between;
		align-items: center;
		border-bottom: 1px solid #334155;
	}

	.thumb-badge-wrap {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}
	.pkg-thumb {
		font-size: 1.25rem;
	}
	.feature-badge {
		background: rgba(245, 158, 11, 0.15);
		color: #fbbf24;
		border: 1px solid rgba(245, 158, 11, 0.3);
		padding: 0.15rem 0.5rem;
		border-radius: 9999px;
		font-size: 0.7rem;
		font-weight: 700;
	}

	.pkg-style-tag {
		background: rgba(14, 165, 233, 0.15);
		color: #38bdf8;
		padding: 0.15rem 0.5rem;
		border-radius: 0.25rem;
		font-size: 0.7rem;
		font-weight: 600;
	}

	.pkg-body {
		padding: 1.25rem;
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
		flex: 1;
	}

	.pkg-title {
		font-size: 1.15rem;
		font-weight: 700;
		color: #f8fafc;
		margin: 0;
		line-height: 1.35;
	}

	.pkg-circuit {
		display: flex;
		align-items: center;
		gap: 0.35rem;
		font-size: 0.8125rem;
		color: #94a3b8;
	}
	.circuit-names {
		color: #e2e8f0;
	}

	.pkg-duration {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 0.75rem;
		color: #64748b;
	}

	.price-display-box {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 0.75rem 1rem;
		border-radius: 0.5rem;
		background: #090d16;
		border: 1px solid #1e293b;
	}
	.price-display-box.peak {
		border-color: rgba(239, 68, 68, 0.4);
		background: linear-gradient(180deg, rgba(239, 68, 68, 0.08) 0%, #090d16 100%);
	}
	.price-display-box.regular {
		border-color: rgba(14, 165, 233, 0.4);
		background: linear-gradient(180deg, rgba(14, 165, 233, 0.08) 0%, #090d16 100%);
	}
	.price-display-box.offpeak {
		border-color: rgba(34, 197, 94, 0.4);
		background: linear-gradient(180deg, rgba(34, 197, 94, 0.08) 0%, #090d16 100%);
	}

	.price-season-label {
		font-size: 0.6875rem;
		color: #94a3b8;
		text-transform: uppercase;
		font-weight: 600;
	}
	.price-val {
		font-size: 1.4rem;
		font-weight: 800;
		color: #f8fafc;
	}
	.pax-tag {
		font-size: 0.75rem;
		font-weight: 500;
		color: #64748b;
	}
	.price-right {
		text-align: right;
	}
	.margin-label {
		display: block;
		font-size: 0.6875rem;
		color: #64748b;
	}
	.margin-val {
		font-size: 0.875rem;
		font-weight: 700;
		color: #4ade80;
	}

	.cost-rollup-list {
		background: #1e293b;
		padding: 0.6rem 0.75rem;
		border-radius: 0.375rem;
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
		font-size: 0.75rem;
		color: #cbd5e1;
	}
	.cr-row {
		display: flex;
		justify-content: space-between;
	}

	.inclusions-preview {
		font-size: 0.75rem;
	}
	.inc-title {
		font-weight: 600;
		color: #94a3b8;
		margin-bottom: 0.25rem;
		display: block;
	}
	.inc-list {
		margin: 0;
		padding-left: 0;
		list-style: none;
		color: #cbd5e1;
		display: flex;
		flex-direction: column;
		gap: 0.2rem;
	}

	.card-footer {
		display: flex;
		gap: 0.4rem;
		padding: 0.75rem 1.25rem;
		background: #090d16;
		border-top: 1px solid #1e293b;
	}

	.action-btn-sm {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 0.3rem;
		padding: 0.5rem 0.75rem;
		border-radius: 0.375rem;
		font-size: 0.75rem;
		font-weight: 600;
		text-decoration: none;
		cursor: pointer;
		border: 1px solid transparent;
		transition: all 0.15s ease;
		flex: 1;
	}
	.action-btn-sm.details {
		background: #1e293b;
		color: #e2e8f0;
		border-color: #334155;
	}
	.action-btn-sm.whatsapp {
		background: #059669;
		color: #ffffff;
	}
	.action-btn-sm.book {
		background: #0284c7;
		color: #ffffff;
	}

	/* Empty state */
	.empty-state {
		grid-column: 1 / -1;
		background: #0f172a;
		border: 1px dashed #334155;
		border-radius: 1rem;
		padding: 3rem 1.5rem;
		text-align: center;
		color: #94a3b8;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.75rem;
	}
	.empty-icon {
		font-size: 2.5rem;
	}

	/* Modal & Timeline */
	.modal-backdrop {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.75);
		backdrop-filter: blur(4px);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 9999;
		padding: 1rem;
	}

	.modal-box {
		background: #0f172a;
		border: 1px solid #1e293b;
		border-radius: 1rem;
		width: 100%;
		max-width: 780px;
		box-shadow: 0 20px 40px rgba(0, 0, 0, 0.5);
		display: flex;
		flex-direction: column;
		max-height: 90vh;
		overflow-y: auto;
	}

	.modal-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 1.25rem 1.5rem;
		border-bottom: 1px solid #1e293b;
	}
	.modal-header h2 {
		margin: 0;
		font-size: 1.15rem;
		font-weight: 700;
		color: #f8fafc;
	}
	.close-btn {
		background: transparent;
		border: none;
		color: #94a3b8;
		font-size: 1.25rem;
		cursor: pointer;
	}

	.modal-body {
		padding: 1.25rem 1.5rem;
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}
	.modal-footer {
		display: flex;
		justify-content: flex-end;
		gap: 0.75rem;
		padding: 1rem 1.5rem;
		border-top: 1px solid #1e293b;
		background: #090d16;
	}

	.form-row {
		display: flex;
		gap: 0.75rem;
		flex-wrap: wrap;
	}
	.form-group {
		display: flex;
		flex-direction: column;
		gap: 0.35rem;
		flex: 1;
	}
	.flex-1 { flex: 1; }
	.flex-2 { flex: 2; }

	.form-group label {
		font-size: 0.75rem;
		font-weight: 600;
		color: #94a3b8;
	}
	.form-input {
		background: #1e293b;
		border: 1px solid #334155;
		color: #f8fafc;
		border-radius: 0.375rem;
		padding: 0.5rem 0.75rem;
		font-size: 0.8125rem;
		outline: none;
	}

	/* Costing Engine Panel */
	.costing-engine-panel {
		background: #090d16;
		border: 1px solid #334155;
		border-radius: 0.5rem;
		padding: 1rem;
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}
	.engine-title {
		margin: 0;
		font-size: 0.8125rem;
		color: #38bdf8;
		font-weight: 700;
	}
	.engine-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(130px, 1fr));
		gap: 0.5rem;
	}
	.engine-item {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}
	.engine-item label {
		font-size: 0.6875rem;
		color: #94a3b8;
	}
	.engine-input {
		background: #1e293b;
		border: 1px solid #334155;
		color: #f8fafc;
		border-radius: 0.25rem;
		padding: 0.4rem 0.5rem;
		font-size: 0.8125rem;
		outline: none;
	}
	.engine-input.highlight {
		border-color: #10b981;
		color: #34d399;
		font-weight: 700;
	}

	.computed-price-bar {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding-top: 0.5rem;
		border-top: 1px solid #1e293b;
	}
	.cp-label {
		font-size: 0.8125rem;
		color: #cbd5e1;
		font-weight: 600;
	}
	.cp-amount {
		font-size: 1.25rem;
		font-weight: 800;
		color: #f8fafc;
	}
	.cp-amount small {
		font-size: 0.75rem;
		color: #64748b;
	}

	/* Timeline */
	.itinerary-timeline {
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}
	.timeline-step {
		display: flex;
		gap: 1rem;
		align-items: flex-start;
	}
	.step-badge {
		background: #0ea5e9;
		color: #ffffff;
		font-size: 0.75rem;
		font-weight: 800;
		padding: 0.35rem 0.75rem;
		border-radius: 9999px;
		flex-shrink: 0;
	}
	.step-card {
		background: #1e293b;
		border: 1px solid #334155;
		border-radius: 0.5rem;
		padding: 0.85rem 1rem;
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 0.4rem;
	}
	.step-title {
		margin: 0;
		font-size: 0.95rem;
		font-weight: 700;
		color: #f8fafc;
	}
	.step-highlights {
		margin: 0;
		font-size: 0.8125rem;
		color: #cbd5e1;
		line-height: 1.4;
	}
	.step-meta {
		display: flex;
		flex-wrap: wrap;
		gap: 0.75rem;
		font-size: 0.75rem;
		color: #94a3b8;
		padding-top: 0.35rem;
		border-top: 1px solid rgba(255, 255, 255, 0.05);
	}
	.step-meta strong {
		color: #e2e8f0;
	}
</style>
