<script lang="ts">
	import { toast } from '$lib/stores/toast.svelte';

	interface TourGuide {
		id: string;
		name: string;
		avatar: string;
		badgeNumber: string; // Govt / Tourism Dept License Number
		licenseBody: 'Ministry of Tourism (Incredible India)' | 'Kerala Tourism Dept' | 'J&K Tourism Dept' | 'Malaysia MOTAC' | 'Dubai DTCM' | 'Tamil Nadu Tourism';
		area: string; // e.g. "Kashmir (Srinagar, Gulmarg, Pahalgam)"
		regionCode: 'kashmir' | 'kerala' | 'malaysia' | 'dubai' | 'tamilnadu';
		phone: string;
		whatsapp: string;
		email: string;
		rating: number;
		totalTripsCompleted: number;
		languages: string[];
		specializations: string[];
		dailyRate: number;
		halfDayRate: number;
		currency: string;
		status: 'available' | 'on_tour' | 'resting';
		currentAssignment?: {
			tripTitle: string;
			customerName: string;
			customerCity: string;
			dates: string;
			pax: number;
		};
		notes: string;
		verifiedBadge: boolean;
	}

	// Filter and State
	let searchQuery = $state('');
	let filterRegion = $state('all');
	let filterLanguage = $state('all');
	let filterStatus = $state('all');
	let filterSpecialization = $state('all');

	// Modals
	let showAddGuideModal = $state(false);
	let selectedGuideForDetail = $state<TourGuide | null>(null);
	let showAssignModal = $state(false);
	let targetGuide = $state<TourGuide | null>(null);

	// Quick Assign form state
	let assignTripTitle = $state('');
	let assignCustomer = $state('');
	let assignDates = $state('');
	let assignPax = $state(2);

	// Dataset of Area-Wise Local Tour Guides
	let guides = $state<TourGuide[]>([
		{
			id: 'guide-1',
			name: 'Bashir Ahmed Lone',
			avatar: '🏔️',
			badgeNumber: 'JKT-SRINAGAR-2021-089',
			licenseBody: 'J&K Tourism Dept',
			area: 'Kashmir (Srinagar, Gulmarg, Pahalgam, Sonamarg)',
			regionCode: 'kashmir',
			phone: '+91 94191 44552',
			whatsapp: '+91 94191 44552',
			email: 'bashir.kashmirtour@gmail.com',
			rating: 4.95,
			totalTripsCompleted: 142,
			languages: ['English', 'Hindi', 'Urdu', 'Kashmiri'],
			specializations: ['Heritage Architecture', 'Gulmarg Ski & Gondola', 'Mughal Gardens Walk', 'Photography'],
			dailyRate: 3000,
			halfDayRate: 1800,
			currency: 'INR',
			status: 'on_tour',
			currentAssignment: {
				tripTitle: 'Kashmir Paradise 4D/5N Family Vacation',
				customerName: 'Karthik Subramanian',
				customerCity: 'Chennai',
				dates: '24 Oct - 28 Oct',
				pax: 4
			},
			notes: 'Expert in Shankaracharya Temple, Betaab Valley history, and Kashmiri Wazwan food trails.',
			verifiedBadge: true
		},
		{
			id: 'guide-2',
			name: 'K. S. Narayanan Nair',
			avatar: '🌴',
			badgeNumber: 'KTDC-GUIDE-KL-4091',
			licenseBody: 'Kerala Tourism Dept',
			area: 'Kerala (Munnar, Thekkady, Alleppey Backwaters, Kochi)',
			regionCode: 'kerala',
			phone: '+91 98471 33440',
			whatsapp: '+91 98471 33440',
			email: 'narayanan.kerala@godsowncountry.in',
			rating: 4.92,
			totalTripsCompleted: 218,
			languages: ['Tamil', 'Malayalam', 'English', 'Hindi'],
			specializations: ['Tea Plantation Walks', 'Alleppey Backwater Heritage', 'Kathakali & Kalaripayattu', 'Spice Plantation'],
			dailyRate: 2500,
			halfDayRate: 1500,
			currency: 'INR',
			status: 'available',
			notes: 'Fluent in both Tamil & English. Ideal for Tamil Nadu family groups visiting Munnar & Alleppey.',
			verifiedBadge: true
		},
		{
			id: 'guide-3',
			name: 'Suresh Kumar Gounder',
			avatar: '🛕',
			badgeNumber: 'TNT-MADURAI-H-014',
			licenseBody: 'Tamil Nadu Tourism',
			area: 'Tamil Nadu (Madurai, Rameswaram, Kanyakumari, Chettinad)',
			regionCode: 'tamilnadu',
			phone: '+91 94431 77882',
			whatsapp: '+91 94431 77882',
			email: 'suresh.maduraiguide@yahoo.com',
			rating: 4.98,
			totalTripsCompleted: 310,
			languages: ['Tamil', 'English', 'Hindi'],
			specializations: ['Meenakshi Amman Temple Iconography', 'Chettinad Palace Heritage', 'Dhanushkodi Ramayana Trail', 'Kanyakumari Sunsets'],
			dailyRate: 2200,
			halfDayRate: 1300,
			currency: 'INR',
			status: 'available',
			notes: 'Certified senior temple historian. Arranges VIP darshan protocol and authentic Chettinad lunch trails.',
			verifiedBadge: true
		},
		{
			id: 'guide-4',
			name: 'Kenneth Lim',
			avatar: '🏙️',
			badgeNumber: 'MOTAC-KL-BLUE-5520',
			licenseBody: 'Malaysia MOTAC',
			area: 'Malaysia (Kuala Lumpur, Batu Caves, Genting, Malacca)',
			regionCode: 'malaysia',
			phone: '+60 16 887 4321',
			whatsapp: '+60 16 887 4321',
			email: 'kenneth.lim@kltourguide.my',
			rating: 4.91,
			totalTripsCompleted: 175,
			languages: ['English', 'Malay', 'Tamil', 'Cantonese'],
			specializations: ['KL Modern Architecture', 'Batu Caves Pilgrimage', 'Malacca Dutch Heritage', 'Jalan Alor Street Food'],
			dailyRate: 6500,
			halfDayRate: 4000,
			currency: 'INR',
			status: 'available',
			notes: 'Speaks conversant Tamil for South Indian groups. Handles Genting cable car VIP queues.',
			verifiedBadge: true
		},
		{
			id: 'guide-5',
			name: 'Rashid Al-Maktoum / Tariq',
			avatar: '🏜️',
			badgeNumber: 'DTCM-TOURIST-LIC-9902',
			licenseBody: 'Dubai DTCM',
			area: 'Dubai & UAE (Downtown Burj, Old Bastakiya, Desert Safari, Marina)',
			regionCode: 'dubai',
			phone: '+971 50 123 9988',
			whatsapp: '+971 50 123 9988',
			email: 'tariq.dubai@safari-guide.ae',
			rating: 4.97,
			totalTripsCompleted: 260,
			languages: ['English', 'Arabic', 'Hindi', 'Urdu'],
			specializations: ['VIP Dune Bashing', 'Burj Khalifa Sky Deck', 'Gold Souk Shopping Assistance', 'Private Yacht Host'],
			dailyRate: 12000,
			halfDayRate: 7000,
			currency: 'INR',
			status: 'on_tour',
			currentAssignment: {
				tripTitle: 'Dubai Luxury Shopping & Desert Experience',
				customerName: 'A. Rahman & Family',
				customerCity: 'Madurai',
				dates: '23 Oct - 26 Oct',
				pax: 6
			},
			notes: 'Official DTCM Gold Card guide. Excellent for Indian family VIP coordination and Arabic heritage storytelling.',
			verifiedBadge: true
		},
		{
			id: 'guide-6',
			name: 'Ghulam Mohammad Zargar',
			avatar: '🛶',
			badgeNumber: 'JKT-DAL-SHIKARA-044',
			licenseBody: 'J&K Tourism Dept',
			area: 'Kashmir (Dal Lake, Nigeen Lake, Old Srinagar Wooden City)',
			regionCode: 'kashmir',
			phone: '+91 99065 11200',
			whatsapp: '+91 99065 11200',
			email: 'ghulam.dallake@kashmir.org',
			rating: 4.88,
			totalTripsCompleted: 195,
			languages: ['Hindi', 'Urdu', 'English', 'Kashmiri'],
			specializations: ['Floating Vegetable Market', 'Wood Carving Artisans', 'Pashmina Weaving Tour', 'Houseboat Hospitality'],
			dailyRate: 2400,
			halfDayRate: 1400,
			currency: 'INR',
			status: 'available',
			notes: 'Lifelong Dal Lake resident. Expert at arranging authentic private shikara sunrise rides.',
			verifiedBadge: true
		}
	]);

	// Filtered Guides
	let filteredGuides = $derived(
		guides.filter((g) => {
			const matchesQuery =
				searchQuery === '' ||
				g.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
				g.area.toLowerCase().includes(searchQuery.toLowerCase()) ||
				g.badgeNumber.toLowerCase().includes(searchQuery.toLowerCase()) ||
				g.languages.some((l) => l.toLowerCase().includes(searchQuery.toLowerCase())) ||
				g.specializations.some((s) => s.toLowerCase().includes(searchQuery.toLowerCase()));

			const matchesRegion = filterRegion === 'all' || g.regionCode === filterRegion;
			const matchesLang = filterLanguage === 'all' || g.languages.includes(filterLanguage);
			const matchesStatus = filterStatus === 'all' || g.status === filterStatus;
			const matchesSpec = filterSpecialization === 'all' || g.specializations.some(s => s.toLowerCase().includes(filterSpecialization.toLowerCase()));

			return matchesQuery && matchesRegion && matchesLang && matchesStatus && matchesSpec;
		})
	);

	// Quick Stats
	let totalGuides = $derived(guides.length);
	let availableGuides = $derived(guides.filter((g) => g.status === 'available').length);
	let onTourGuides = $derived(guides.filter((g) => g.status === 'on_tour').length);
	let tamilSpeakingGuides = $derived(guides.filter((g) => g.languages.includes('Tamil')).length);

	// Form for New Guide
	let newGuide = $state({
		name: '',
		badgeNumber: '',
		licenseBody: 'Kerala Tourism Dept' as TourGuide['licenseBody'],
		area: '',
		regionCode: 'kerala' as TourGuide['regionCode'],
		phone: '',
		whatsapp: '',
		email: '',
		languagesInput: 'English, Tamil, Hindi',
		specializationsInput: 'Heritage, Family Tours',
		dailyRate: 2500,
		halfDayRate: 1500,
		currency: 'INR',
		notes: ''
	});

	function handleRegisterGuide() {
		if (!newGuide.name || !newGuide.phone || !newGuide.area) {
			toast.error('Please enter Guide Name, Area, and Phone Number');
			return;
		}

		const guide: TourGuide = {
			id: `guide-${Date.now()}`,
			name: newGuide.name,
			avatar: '🧭',
			badgeNumber: newGuide.badgeNumber || `GOVT-LIC-${Math.floor(1000 + Math.random() * 9000)}`,
			licenseBody: newGuide.licenseBody,
			area: newGuide.area,
			regionCode: newGuide.regionCode,
			phone: newGuide.phone,
			whatsapp: newGuide.whatsapp || newGuide.phone,
			email: newGuide.email,
			rating: 5.0,
			totalTripsCompleted: 0,
			languages: newGuide.languagesInput.split(',').map((s) => s.trim()).filter(Boolean),
			specializations: newGuide.specializationsInput.split(',').map((s) => s.trim()).filter(Boolean),
			dailyRate: Number(newGuide.dailyRate),
			halfDayRate: Number(newGuide.halfDayRate),
			currency: newGuide.currency,
			status: 'available',
			notes: newGuide.notes,
			verifiedBadge: true
		};

		guides = [guide, ...guides];
		showAddGuideModal = false;
		toast.success(`Registered licensed guide "${guide.name}" for ${guide.area}`);

		// Reset form
		newGuide = {
			name: '',
			badgeNumber: '',
			licenseBody: 'Kerala Tourism Dept',
			area: '',
			regionCode: 'kerala',
			phone: '',
			whatsapp: '',
			email: '',
			languagesInput: 'English, Tamil, Hindi',
			specializationsInput: 'Heritage, Family Tours',
			dailyRate: 2500,
			halfDayRate: 1500,
			currency: 'INR',
			notes: ''
		};
	}

	function openAssignTripModal(guide: TourGuide) {
		targetGuide = guide;
		assignTripTitle = '';
		assignCustomer = '';
		assignDates = '';
		assignPax = 4;
		showAssignModal = true;
	}

	function handleConfirmAssignment() {
		if (!targetGuide) return;
		if (!assignTripTitle || !assignCustomer) {
			toast.error('Please fill in Trip Title and Customer Name');
			return;
		}

		targetGuide.status = 'on_tour';
		targetGuide.currentAssignment = {
			tripTitle: assignTripTitle,
			customerName: assignCustomer,
			customerCity: 'Chennai',
			dates: assignDates || 'Next 4 Days',
			pax: assignPax
		};

		// trigger reactivity
		guides = [...guides];
		showAssignModal = false;
		toast.success(`Assigned guide ${targetGuide.name} to "${assignTripTitle}"!`);
	}

	function handleMarkAvailable(guide: TourGuide) {
		guide.status = 'available';
		guide.currentAssignment = undefined;
		guides = [...guides];
		toast.info(`Guide ${guide.name} is now marked Available for new assignments.`);
	}

	function getWhatsAppUrl(phone: string, guideName: string, area: string) {
		const cleanPhone = phone.replace(/[^0-9]/g, '');
		const text = encodeURIComponent(
			`Vanakkam / Hello ${guideName}, greetings from MrBiz OS Tour Operations! We have a guest booking for ${area}. Could you confirm your availability and daily batta?`
		);
		return `https://wa.me/${cleanPhone}?text=${text}`;
	}
</script>

<div class="guides-root">
	<!-- Top Header with Actions -->
	<div class="header-card">
		<div class="title-cluster">
			<div class="title-badge">
				<span class="badge-icon">🧭</span>
				<span class="badge-label">Destination Experts & Tour Historians</span>
			</div>
			<h1 class="main-title">Area-Wise Local Tour Guide Management</h1>
			<p class="subtitle">
				Coordinate licensed destination guides for <strong>Kashmir</strong>, <strong>Kerala</strong>, <strong>Malaysia</strong>, <strong>Dubai</strong>, and <strong>Tamil Nadu</strong>. Filter by languages (Tamil, English, Hindi, Arabic), verify Tourism badges, and assign day-wise trip duties.
			</p>
		</div>

		<div class="header-actions">
			<a href="/travel/book" class="action-btn secondary">
				<span>⚡</span>
				<span>1-Page Booking Desk</span>
			</a>
			<button class="action-btn primary" onclick={() => (showAddGuideModal = true)}>
				<span>+</span>
				<span>Register Tour Guide</span>
			</button>
		</div>
	</div>

	<!-- KPI Summary Metrics -->
	<div class="kpi-grid">
		<div class="kpi-card">
			<div class="kpi-header">
				<span class="kpi-icon">📜</span>
				<span class="kpi-label">Active Guides</span>
			</div>
			<div class="kpi-value">{totalGuides}</div>
			<div class="kpi-sub">Govt & Tourism Dept Certified</div>
		</div>

		<div class="kpi-card highlight-green">
			<div class="kpi-header">
				<span class="kpi-icon">🟢</span>
				<span class="kpi-label">Available Today</span>
			</div>
			<div class="kpi-value">{availableGuides}</div>
			<div class="kpi-sub">Ready for instant assignment</div>
		</div>

		<div class="kpi-card highlight-blue">
			<div class="kpi-header">
				<span class="kpi-icon">🎒</span>
				<span class="kpi-label">On Active Tours</span>
			</div>
			<div class="kpi-value">{onTourGuides}</div>
			<div class="kpi-sub">Guiding guest groups currently</div>
		</div>

		<div class="kpi-card highlight-amber">
			<div class="kpi-header">
				<span class="kpi-icon">🗣️</span>
				<span class="kpi-label">Tamil Speaking</span>
			</div>
			<div class="kpi-value">{tamilSpeakingGuides}</div>
			<div class="kpi-sub">Dedicated for TN/South travelers</div>
		</div>
	</div>

	<!-- Filter & Search Toolbar -->
	<div class="filter-card">
		<div class="search-input-wrap">
			<span class="search-icon">🔍</span>
			<input
				type="text"
				placeholder="Search guide by name, area, license badge, or specialty..."
				bind:value={searchQuery}
				class="search-input"
			/>
		</div>

		<div class="filter-controls">
			<!-- Region Filter -->
			<div class="select-wrap">
				<label for="filter-region">Circuit / Area:</label>
				<select id="filter-region" bind:value={filterRegion} class="filter-select">
					<option value="all">All Circuits</option>
					<option value="kashmir">🏔️ Kashmir Circuit</option>
					<option value="kerala">🌴 Kerala Backwaters & Hills</option>
					<option value="malaysia">🏙️ Malaysia & Genting</option>
					<option value="dubai">🏜️ Dubai & Emirates</option>
					<option value="tamilnadu">🛕 Tamil Nadu Temples</option>
				</select>
			</div>

			<!-- Language Filter -->
			<div class="select-wrap">
				<label for="filter-lang">Language:</label>
				<select id="filter-lang" bind:value={filterLanguage} class="filter-select">
					<option value="all">All Languages</option>
					<option value="Tamil">Tamil (தமிழ்)</option>
					<option value="English">English</option>
					<option value="Hindi">Hindi</option>
					<option value="Malayalam">Malayalam</option>
					<option value="Arabic">Arabic</option>
				</select>
			</div>

			<!-- Status Filter -->
			<div class="select-wrap">
				<label for="filter-status">Status:</label>
				<select id="filter-status" bind:value={filterStatus} class="filter-select">
					<option value="all">All Statuses</option>
					<option value="available">🟢 Available Only</option>
					<option value="on_tour">🎒 On Active Tour</option>
				</select>
			</div>
		</div>
	</div>

	<!-- Guide Cards Grid -->
	<div class="guides-grid">
		{#if filteredGuides.length === 0}
			<div class="empty-state">
				<span class="empty-icon">🧭</span>
				<h3>No guides found matching this criteria</h3>
				<p>Try clearing your search terms or changing the circuit/language filter.</p>
				<button class="action-btn secondary" onclick={() => { searchQuery = ''; filterRegion = 'all'; filterLanguage = 'all'; filterStatus = 'all'; }}>
					Reset All Filters
				</button>
			</div>
		{:else}
			{#each filteredGuides as guide (guide.id)}
				<div class="guide-card {guide.status}">
					<!-- Top Card Header -->
					<div class="card-top">
						<div class="avatar-wrap">
							<span class="avatar-icon">{guide.avatar}</span>
							{#if guide.verifiedBadge}
								<span class="verified-dot" title="Govt Tourism Verified License">✓</span>
							{/if}
						</div>
						<div class="guide-info">
							<div class="name-row">
								<h2 class="guide-name">{guide.name}</h2>
								<span class="status-tag {guide.status}">
									{guide.status === 'available' ? 'Available' : 'On Tour'}
								</span>
							</div>
							<div class="badge-row">
								<span class="license-badge">🆔 {guide.badgeNumber}</span>
								<span class="license-body">({guide.licenseBody})</span>
							</div>
						</div>
					</div>

					<!-- Area & Circuit Badge -->
					<div class="circuit-banner">
						<span class="circuit-icon">📍</span>
						<span class="circuit-text">{guide.area}</span>
					</div>

					<!-- Languages & Specializations -->
					<div class="meta-section">
						<div class="meta-row">
							<span class="meta-label">🗣️ Languages:</span>
							<div class="pills-wrap">
								{#each guide.languages as lang}
									<span class="lang-pill {lang === 'Tamil' ? 'special' : ''}">{lang}</span>
								{/each}
							</div>
						</div>

						<div class="meta-row">
							<span class="meta-label">🎯 Specialty:</span>
							<div class="pills-wrap">
								{#each guide.specializations as spec}
									<span class="spec-pill">{spec}</span>
								{/each}
							</div>
						</div>
					</div>

					<!-- Pricing & Performance -->
					<div class="rate-bar">
						<div class="rate-col">
							<span class="rate-label">Full Day (8 hrs)</span>
							<span class="rate-amount">₹{guide.dailyRate.toLocaleString()}</span>
						</div>
						<div class="rate-col">
							<span class="rate-label">Half Day (4 hrs)</span>
							<span class="rate-amount">₹{guide.halfDayRate.toLocaleString()}</span>
						</div>
						<div class="rate-col rating-col">
							<span class="rating-val">⭐ {guide.rating}</span>
							<span class="trips-count">{guide.totalTripsCompleted} trips</span>
						</div>
					</div>

					<!-- Current Assignment if on Tour -->
					{#if guide.status === 'on_tour' && guide.currentAssignment}
						<div class="assignment-box">
							<div class="assignment-header">
								<span class="assign-icon">🚩</span>
								<span class="assign-title">Active Assignment</span>
								<button class="release-btn" onclick={() => handleMarkAvailable(guide)} title="Mark trip as completed">
									Release Guide
								</button>
							</div>
							<div class="assignment-trip">{guide.currentAssignment.tripTitle}</div>
							<div class="assignment-guest">
								Guest: <strong>{guide.currentAssignment.customerName}</strong> ({guide.currentAssignment.customerCity}) • {guide.currentAssignment.pax} Pax
							</div>
							<div class="assignment-dates">Dates: {guide.currentAssignment.dates}</div>
						</div>
					{/if}

					<!-- Notes -->
					{#if guide.notes}
						<div class="guide-notes">
							💡 <em>{guide.notes}</em>
						</div>
					{/if}

					<!-- Action Footer -->
					<div class="card-footer">
						<a
							href={getWhatsAppUrl(guide.whatsapp, guide.name, guide.area)}
							target="_blank"
							rel="noreferrer"
							class="action-btn-sm whatsapp"
						>
							<span>💬</span>
							<span>WhatsApp</span>
						</a>

						<a href="tel:{guide.phone}" class="action-btn-sm call">
							<span>📞</span>
							<span>Call</span>
						</a>

						{#if guide.status === 'available'}
							<button class="action-btn-sm assign" onclick={() => openAssignTripModal(guide)}>
								<span>📌</span>
								<span>Assign to Trip</span>
							</button>
						{:else}
							<button class="action-btn-sm details" onclick={() => (selectedGuideForDetail = guide)}>
								<span>ℹ️</span>
								<span>Details</span>
							</button>
						{/if}
					</div>
				</div>
			{/each}
		{/if}
	</div>
</div>

<!-- Modal: Register Tour Guide -->
{#if showAddGuideModal}
	<div class="modal-backdrop">
		<div class="modal-box">
			<div class="modal-header">
				<h2>🧭 Register Destination Tour Guide</h2>
				<button class="close-btn" onclick={() => (showAddGuideModal = false)}>✕</button>
			</div>

			<div class="modal-body">
				<div class="form-row">
					<div class="form-group flex-2">
						<label for="guide-name">Guide Full Name *</label>
						<input id="guide-name" type="text" placeholder="e.g. M. S. Ramanathan" bind:value={newGuide.name} class="form-input" />
					</div>
					<div class="form-group flex-1">
						<label for="guide-circuit">Circuit / Region *</label>
						<select id="guide-circuit" bind:value={newGuide.regionCode} class="form-input">
							<option value="kerala">🌴 Kerala</option>
							<option value="kashmir">🏔️ Kashmir</option>
							<option value="tamilnadu">🛕 Tamil Nadu</option>
							<option value="malaysia">🏙️ Malaysia</option>
							<option value="dubai">🏜️ Dubai</option>
						</select>
					</div>
				</div>

				<div class="form-row">
					<div class="form-group flex-2">
						<label for="guide-area">Operational Area / Monuments *</label>
						<input id="guide-area" type="text" placeholder="e.g. Kerala (Munnar, Thekkady, Alleppey)" bind:value={newGuide.area} class="form-input" />
					</div>
					<div class="form-group flex-1">
						<label for="guide-badge">Govt License / Badge #</label>
						<input id="guide-badge" type="text" placeholder="e.g. KTDC-GUIDE-0992" bind:value={newGuide.badgeNumber} class="form-input" />
					</div>
				</div>

				<div class="form-row">
					<div class="form-group flex-1">
						<label for="guide-lic-body">Licensing Authority</label>
						<select id="guide-lic-body" bind:value={newGuide.licenseBody} class="form-input">
							<option value="Kerala Tourism Dept">Kerala Tourism Dept</option>
							<option value="J&K Tourism Dept">J&K Tourism Dept</option>
							<option value="Tamil Nadu Tourism">Tamil Nadu Tourism</option>
							<option value="Ministry of Tourism (Incredible India)">Ministry of Tourism (Incredible India)</option>
							<option value="Malaysia MOTAC">Malaysia MOTAC</option>
							<option value="Dubai DTCM">Dubai DTCM</option>
						</select>
					</div>
					<div class="form-group flex-1">
						<label for="guide-phone">Mobile / Phone *</label>
						<input id="guide-phone" type="text" placeholder="+91 98470 11223" bind:value={newGuide.phone} class="form-input" />
					</div>
					<div class="form-group flex-1">
						<label for="guide-wa">WhatsApp Direct</label>
						<input id="guide-wa" type="text" placeholder="+91 98470 11223" bind:value={newGuide.whatsapp} class="form-input" />
					</div>
				</div>

				<div class="form-row">
					<div class="form-group flex-1">
						<label for="guide-langs">Languages Spoken (comma separated)</label>
						<input id="guide-langs" type="text" placeholder="English, Tamil, Hindi, Malayalam" bind:value={newGuide.languagesInput} class="form-input" />
					</div>
					<div class="form-group flex-1">
						<label for="guide-specs">Specializations (comma separated)</label>
						<input id="guide-specs" type="text" placeholder="Tea Estates, Backwaters, Food Trail" bind:value={newGuide.specializationsInput} class="form-input" />
					</div>
				</div>

				<div class="form-row">
					<div class="form-group flex-1">
						<label for="guide-rate-full">Full Day Rate (₹)</label>
						<input id="guide-rate-full" type="number" bind:value={newGuide.dailyRate} class="form-input" />
					</div>
					<div class="form-group flex-1">
						<label for="guide-rate-half">Half Day Rate (₹)</label>
						<input id="guide-rate-half" type="number" bind:value={newGuide.halfDayRate} class="form-input" />
					</div>
					<div class="form-group flex-1">
						<label for="guide-email">Email (Optional)</label>
						<input id="guide-email" type="email" placeholder="guide@dmc.in" bind:value={newGuide.email} class="form-input" />
					</div>
				</div>

				<div class="form-group">
					<label for="guide-notes">Operational Notes & Landmark Knowledge</label>
					<textarea id="guide-notes" rows="2" placeholder="e.g. Excellent for family groups, arranges boat safari tickets in advance" bind:value={newGuide.notes} class="form-input"></textarea>
				</div>
			</div>

			<div class="modal-footer">
				<button class="action-btn secondary" onclick={() => (showAddGuideModal = false)}>Cancel</button>
				<button class="action-btn primary" onclick={handleRegisterGuide}>Save & Register Guide</button>
			</div>
		</div>
	</div>
{/if}

<!-- Modal: Assign Guide to Trip -->
{#if showAssignModal && targetGuide}
	<div class="modal-backdrop">
		<div class="modal-box modal-sm">
			<div class="modal-header">
				<h2>📌 Assign {targetGuide.name} to Tour</h2>
				<button class="close-btn" onclick={() => (showAssignModal = false)}>✕</button>
			</div>

			<div class="modal-body">
				<div class="guide-summary-banner">
					<div class="guide-sub-name">Guide: <strong>{targetGuide.name}</strong></div>
					<div class="guide-sub-area">Area: {targetGuide.area}</div>
					<div class="guide-sub-rate">Daily Rate: ₹{targetGuide.dailyRate.toLocaleString()}</div>
				</div>

				<div class="form-group">
					<label for="assign-title">Tour Trip Title *</label>
					<input id="assign-title" type="text" placeholder="e.g. Kashmir 4D/5N Family Escape" bind:value={assignTripTitle} class="form-input" />
				</div>

				<div class="form-group">
					<label for="assign-guest">Primary Guest / Customer Name *</label>
					<input id="assign-guest" type="text" placeholder="e.g. Ramesh Kumar (Chennai)" bind:value={assignCustomer} class="form-input" />
				</div>

				<div class="form-row">
					<div class="form-group flex-2">
						<label for="assign-dates">Tour Dates</label>
						<input id="assign-dates" type="text" placeholder="e.g. 25 Oct - 29 Oct 2026" bind:value={assignDates} class="form-input" />
					</div>
					<div class="form-group flex-1">
						<label for="assign-pax">Pax Count</label>
						<input id="assign-pax" type="number" bind:value={assignPax} class="form-input" />
					</div>
				</div>
			</div>

			<div class="modal-footer">
				<button class="action-btn secondary" onclick={() => (showAssignModal = false)}>Cancel</button>
				<button class="action-btn primary" onclick={handleConfirmAssignment}>Confirm Assignment</button>
			</div>
		</div>
	</div>
{/if}

<style>
	.guides-root {
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

	/* KPI Cards */
	.kpi-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
		gap: 1rem;
	}

	.kpi-card {
		background: #0f172a;
		border: 1px solid #1e293b;
		border-radius: 0.75rem;
		padding: 1rem 1.25rem;
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}

	.kpi-card.highlight-green {
		border-color: rgba(34, 197, 94, 0.4);
		background: linear-gradient(180deg, rgba(34, 197, 94, 0.08) 0%, #0f172a 100%);
	}

	.kpi-card.highlight-blue {
		border-color: rgba(14, 165, 233, 0.4);
		background: linear-gradient(180deg, rgba(14, 165, 233, 0.08) 0%, #0f172a 100%);
	}

	.kpi-card.highlight-amber {
		border-color: rgba(245, 158, 11, 0.4);
		background: linear-gradient(180deg, rgba(245, 158, 11, 0.08) 0%, #0f172a 100%);
	}

	.kpi-header {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 0.8125rem;
		color: #94a3b8;
		font-weight: 500;
	}

	.kpi-value {
		font-size: 1.75rem;
		font-weight: 800;
		color: #f8fafc;
	}

	.kpi-sub {
		font-size: 0.75rem;
		color: #64748b;
	}

	/* Filters */
	.filter-card {
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
		min-width: 280px;
	}

	.search-icon {
		font-size: 0.875rem;
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
		gap: 0.75rem;
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

	/* Guides Grid */
	.guides-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(360px, 1fr));
		gap: 1.25rem;
	}

	.guide-card {
		background: #0f172a;
		border: 1px solid #1e293b;
		border-radius: 0.875rem;
		padding: 1.25rem;
		display: flex;
		flex-direction: column;
		gap: 0.875rem;
		transition: all 0.2s ease;
	}

	.guide-card:hover {
		border-color: #38bdf8;
		box-shadow: 0 4px 20px rgba(14, 165, 233, 0.12);
	}

	.card-top {
		display: flex;
		gap: 0.875rem;
		align-items: flex-start;
	}

	.avatar-wrap {
		position: relative;
		width: 48px;
		height: 48px;
		background: #1e293b;
		border-radius: 0.75rem;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 1.5rem;
		flex-shrink: 0;
	}

	.verified-dot {
		position: absolute;
		bottom: -4px;
		right: -4px;
		background: #10b981;
		color: #ffffff;
		border-radius: 9999px;
		font-size: 0.625rem;
		font-weight: 800;
		width: 16px;
		height: 16px;
		display: flex;
		align-items: center;
		justify-content: center;
		border: 2px solid #0f172a;
	}

	.guide-info {
		flex: 1;
		min-width: 0;
	}

	.name-row {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 0.5rem;
	}

	.guide-name {
		font-size: 1.05rem;
		font-weight: 700;
		color: #f8fafc;
		margin: 0;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.status-tag {
		font-size: 0.7rem;
		font-weight: 700;
		padding: 0.2rem 0.5rem;
		border-radius: 9999px;
		text-transform: uppercase;
	}
	.status-tag.available {
		background: rgba(34, 197, 94, 0.15);
		color: #4ade80;
		border: 1px solid rgba(34, 197, 94, 0.3);
	}
	.status-tag.on_tour {
		background: rgba(14, 165, 233, 0.15);
		color: #38bdf8;
		border: 1px solid rgba(14, 165, 233, 0.3);
	}

	.badge-row {
		display: flex;
		flex-wrap: wrap;
		gap: 0.35rem;
		margin-top: 0.2rem;
		font-size: 0.75rem;
	}
	.license-badge {
		color: #cbd5e1;
		font-family: monospace;
	}
	.license-body {
		color: #64748b;
	}

	.circuit-banner {
		background: #1e293b;
		border-left: 3px solid #0ea5e9;
		padding: 0.45rem 0.65rem;
		border-radius: 0.375rem;
		font-size: 0.8125rem;
		color: #e2e8f0;
		display: flex;
		align-items: center;
		gap: 0.4rem;
	}

	.meta-section {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.meta-row {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}

	.meta-label {
		font-size: 0.75rem;
		color: #94a3b8;
		font-weight: 600;
	}

	.pills-wrap {
		display: flex;
		flex-wrap: wrap;
		gap: 0.35rem;
	}

	.lang-pill {
		background: #1e293b;
		color: #cbd5e1;
		font-size: 0.75rem;
		padding: 0.15rem 0.5rem;
		border-radius: 0.25rem;
	}
	.lang-pill.special {
		background: rgba(245, 158, 11, 0.2);
		color: #fbbf24;
		border: 1px solid rgba(245, 158, 11, 0.4);
		font-weight: 700;
	}

	.spec-pill {
		background: rgba(99, 102, 241, 0.15);
		color: #a5b4fc;
		font-size: 0.75rem;
		padding: 0.15rem 0.5rem;
		border-radius: 0.25rem;
	}

	.rate-bar {
		display: grid;
		grid-template-columns: 1fr 1fr 1fr;
		gap: 0.5rem;
		background: #090d16;
		border: 1px solid #1e293b;
		padding: 0.6rem 0.75rem;
		border-radius: 0.5rem;
	}

	.rate-col {
		display: flex;
		flex-direction: column;
	}

	.rate-label {
		font-size: 0.6875rem;
		color: #64748b;
	}

	.rate-amount {
		font-size: 0.9375rem;
		font-weight: 700;
		color: #f8fafc;
	}

	.rating-col {
		align-items: flex-end;
		justify-content: center;
	}
	.rating-val {
		font-size: 0.875rem;
		font-weight: 700;
		color: #facc15;
	}
	.trips-count {
		font-size: 0.6875rem;
		color: #64748b;
	}

	.assignment-box {
		background: rgba(14, 165, 233, 0.08);
		border: 1px solid rgba(14, 165, 233, 0.25);
		border-radius: 0.5rem;
		padding: 0.65rem 0.85rem;
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
		font-size: 0.8125rem;
	}

	.assignment-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		font-weight: 700;
		color: #38bdf8;
		font-size: 0.75rem;
	}

	.release-btn {
		background: #1e293b;
		border: 1px solid #334155;
		color: #94a3b8;
		font-size: 0.7rem;
		padding: 0.15rem 0.45rem;
		border-radius: 0.25rem;
		cursor: pointer;
	}
	.release-btn:hover {
		background: #ef4444;
		color: #ffffff;
		border-color: #ef4444;
	}

	.assignment-trip {
		font-weight: 600;
		color: #f8fafc;
	}
	.assignment-guest,
	.assignment-dates {
		color: #94a3b8;
		font-size: 0.75rem;
	}

	.guide-notes {
		font-size: 0.775rem;
		color: #94a3b8;
		background: rgba(30, 41, 59, 0.4);
		padding: 0.4rem 0.6rem;
		border-radius: 0.375rem;
	}

	.card-footer {
		display: flex;
		gap: 0.5rem;
		margin-top: auto;
		padding-top: 0.5rem;
		border-top: 1px solid #1e293b;
	}

	.action-btn-sm {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 0.35rem;
		padding: 0.45rem 0.75rem;
		border-radius: 0.375rem;
		font-size: 0.75rem;
		font-weight: 600;
		text-decoration: none;
		cursor: pointer;
		border: 1px solid transparent;
		transition: all 0.15s ease;
		flex: 1;
	}

	.action-btn-sm.whatsapp {
		background: #059669;
		color: #ffffff;
	}
	.action-btn-sm.whatsapp:hover {
		background: #047857;
	}

	.action-btn-sm.call {
		background: #1e293b;
		color: #e2e8f0;
		border-color: #334155;
	}
	.action-btn-sm.call:hover {
		background: #334155;
	}

	.action-btn-sm.assign {
		background: #0284c7;
		color: #ffffff;
	}
	.action-btn-sm.assign:hover {
		background: #0369a1;
	}

	.action-btn-sm.details {
		background: #1e293b;
		color: #94a3b8;
		border-color: #334155;
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

	/* Modal Styles */
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
		max-width: 680px;
		box-shadow: 0 20px 40px rgba(0, 0, 0, 0.5);
		display: flex;
		flex-direction: column;
		max-height: 90vh;
		overflow-y: auto;
	}
	.modal-box.modal-sm {
		max-width: 480px;
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
	.close-btn:hover {
		color: #f8fafc;
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
	.flex-1 {
		flex: 1;
	}
	.flex-2 {
		flex: 2;
	}

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
	.form-input:focus {
		border-color: #38bdf8;
	}

	.guide-summary-banner {
		background: #1e293b;
		border: 1px solid #334155;
		border-radius: 0.5rem;
		padding: 0.75rem 1rem;
		display: flex;
		flex-direction: column;
		gap: 0.2rem;
		font-size: 0.8125rem;
		color: #cbd5e1;
	}
	.guide-sub-rate {
		color: #38bdf8;
		font-weight: 700;
	}
</style>
