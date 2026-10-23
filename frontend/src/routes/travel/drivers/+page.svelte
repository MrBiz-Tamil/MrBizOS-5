<script lang="ts">
	import { toast } from '$lib/stores/toast.svelte';

	interface Driver {
		id: string;
		name: string;
		agencyName: string;
		agencyId: string;
		phone: string;
		whatsapp: string;
		area: string; // e.g. "Kashmir (Srinagar, Gulmarg, Pahalgam)"
		regionCode: 'kashmir' | 'kerala' | 'malaysia' | 'dubai' | 'tamilnadu';
		vehicleModel: string;
		vehiclePlate: string;
		vehicleType: 'Innova Crysta' | 'Prime Sedan' | 'Tempo Traveller' | 'Luxury Coach' | 'SUV 4x4';
		capacityPax: number;
		dailyRate: number; // in INR / local converted
		currency: string;
		rating: number;
		verifiedLicense: boolean;
		status: 'available' | 'on_trip' | 'off_duty';
		currentTripNote?: string;
		languages: string[];
	}

	interface CarAgency {
		id: string;
		agencyName: string;
		ownerName: string;
		phone: string;
		whatsapp: string;
		email: string;
		baseCity: string;
		regionCode: 'kashmir' | 'kerala' | 'malaysia' | 'dubai' | 'tamilnadu';
		totalVehicles: number;
		fleetBreakdown: string;
		paymentTerms: string;
		contractStatus: 'Verified Partner' | 'Direct Fleet' | 'On Probation';
		gstOrTaxId: string;
	}

	// Active Tab: 'drivers' | 'agencies'
	let activeTab = $state<'drivers' | 'agencies'>('drivers');

	// Filters
	let searchQuery = $state('');
	let filterRegion = $state('all');
	let filterVehicleType = $state('all');
	let filterStatus = $state('all');

	// Modals
	let showAddDriverModal = $state(false);
	let showAddAgencyModal = $state(false);
	let selectedDriver = $state<Driver | null>(null);

	// Dataset: Car Agencies
	let agencies = $state<CarAgency[]>([
		{
			id: 'agency-1',
			agencyName: 'Himalayan Wheels Transport Consortium',
			ownerName: 'Fayaz Ahmed Mir',
			phone: '+91 94190 88221',
			whatsapp: '+91 94190 88221',
			email: 'himalayan.wheels@dmc.in',
			baseCity: 'Srinagar, Jammu & Kashmir',
			regionCode: 'kashmir',
			totalVehicles: 24,
			fleetBreakdown: '14 Innova Crysta, 6 Swift Dzire, 4 Tempo Travellers',
			paymentTerms: '50% on Airport Pickup, Balance on Pahalgam checkout',
			contractStatus: 'Verified Partner',
			gstOrTaxId: '01AAACH8849J1Z4'
		},
		{
			id: 'agency-2',
			agencyName: 'Royal Kerala Holidays Fleet Logistics',
			ownerName: 'V. K. Sasidharan',
			phone: '+91 98470 22339',
			whatsapp: '+91 98470 22339',
			email: 'fleet@royalkerala.com',
			baseCity: 'Cochin & Alleppey, Kerala',
			regionCode: 'kerala',
			totalVehicles: 32,
			fleetBreakdown: '18 Innova Crysta, 10 Swift Dzire, 4 Urbania Vans',
			paymentTerms: 'Weekly reconciliation (Net 7 days)',
			contractStatus: 'Verified Partner',
			gstOrTaxId: '32AABCR9912K1Z9'
		},
		{
			id: 'agency-3',
			agencyName: 'KL Elite Executive Transfers Sdn Bhd',
			ownerName: 'Raymond Tan',
			phone: '+60 12 345 6789',
			whatsapp: '+60 12 345 6789',
			email: 'dispatch@klelite.com.my',
			baseCity: 'Kuala Lumpur, Malaysia',
			regionCode: 'malaysia',
			totalVehicles: 16,
			fleetBreakdown: '8 Mercedes E-Class, 6 Sprinter Coaches, 2 Alphard VIP',
			paymentTerms: '100% Pre-funded B2B DMC credit line',
			contractStatus: 'Verified Partner',
			gstOrTaxId: 'MY-SST-8849102'
		},
		{
			id: 'agency-4',
			agencyName: 'Dubai Desert Star Limousine LLC',
			ownerName: 'Rashid Al-Nuaimi',
			phone: '+971 4 223 9988',
			whatsapp: '+971 50 882 1144',
			email: 'dispatch@desertstarlimo.ae',
			baseCity: 'Deira & Business Bay, Dubai',
			regionCode: 'dubai',
			totalVehicles: 40,
			fleetBreakdown: '20 Land Cruiser 4x4, 12 HiAce Vans, 8 Mercedes Coaches',
			paymentTerms: 'Pre-voucher booking voucher credit',
			contractStatus: 'Verified Partner',
			gstOrTaxId: 'TRN-10029481900003'
		},
		{
			id: 'agency-5',
			agencyName: 'South Tamil Nadu Tourist Taxi Network',
			ownerName: 'M. Senthil Kumar',
			phone: '+91 94431 88992',
			whatsapp: '+91 94431 88992',
			email: 'senthil.cabs@maduraitravels.in',
			baseCity: 'Madurai & Kanniyakumari, Tamil Nadu',
			regionCode: 'tamilnadu',
			totalVehicles: 28,
			fleetBreakdown: '16 Innova Crysta, 8 Dzire, 4 Force Tempo Travellers',
			paymentTerms: 'Trip-wise cash / UPI token',
			contractStatus: 'Direct Fleet',
			gstOrTaxId: '33AABCS4412B1Z8'
		}
	]);

	// Dataset: Drivers (Area, Tour & Trip-Wise)
	let drivers = $state<Driver[]>([
		{
			id: 'drv-1',
			name: 'Ghulam Nabi',
			agencyName: 'Himalayan Wheels Transport Consortium',
			agencyId: 'agency-1',
			phone: '+91 94190 22334',
			whatsapp: '+91 94190 22334',
			area: 'Srinagar - Gulmarg - Pahalgam - Sonamarg Circuit',
			regionCode: 'kashmir',
			vehicleModel: 'Toyota Innova Crysta AC (White)',
			vehiclePlate: 'JK-01-AB-4491',
			vehicleType: 'Innova Crysta',
			capacityPax: 7,
			dailyRate: 3800,
			currency: '₹',
			rating: 4.9,
			verifiedLicense: true,
			status: 'on_trip',
			currentTripNote: 'Active: Rahul Sharma & Family (4N/5D)',
			languages: ['Tamil (Conversational)', 'English', 'Hindi', 'Kashmiri']
		},
		{
			id: 'drv-2',
			name: 'Mukhtar Ahmed Wani',
			agencyName: 'Himalayan Wheels Transport Consortium',
			agencyId: 'agency-1',
			phone: '+91 94191 66778',
			whatsapp: '+91 94191 66778',
			area: 'Srinagar City - Airport Transfers - Doodhpathri',
			regionCode: 'kashmir',
			vehicleModel: 'Maruti Suzuki Dzire AC Prime',
			vehiclePlate: 'JK-01-AC-8890',
			vehicleType: 'Prime Sedan',
			capacityPax: 4,
			dailyRate: 2400,
			currency: '₹',
			rating: 4.8,
			verifiedLicense: true,
			status: 'available',
			languages: ['English', 'Hindi', 'Urdu']
		},
		{
			id: 'drv-3',
			name: 'Santhosh Kumar',
			agencyName: 'Royal Kerala Holidays Fleet Logistics',
			agencyId: 'agency-2',
			phone: '+91 98470 11992',
			whatsapp: '+91 98470 11992',
			area: 'Cochin - Munnar - Thekkady - Alleppey Houseboat Circuit',
			regionCode: 'kerala',
			vehicleModel: 'Maruti Dzire Tour AC',
			vehiclePlate: 'KL-07-CD-9981',
			vehicleType: 'Prime Sedan',
			capacityPax: 4,
			dailyRate: 2600,
			currency: '₹',
			rating: 4.9,
			verifiedLicense: true,
			status: 'available',
			languages: ['Tamil', 'Malayalam', 'English', 'Hindi']
		},
		{
			id: 'drv-4',
			name: 'Biju Varghese',
			agencyName: 'Royal Kerala Holidays Fleet Logistics',
			agencyId: 'agency-2',
			phone: '+91 98472 44556',
			whatsapp: '+91 98472 44556',
			area: 'Kanniyakumari - Kovalam - Trivandrum - Varkala',
			regionCode: 'kerala',
			vehicleModel: 'Toyota Innova Crysta AC 2.4 Z',
			vehiclePlate: 'KL-01-BK-3342',
			vehicleType: 'Innova Crysta',
			capacityPax: 7,
			dailyRate: 3600,
			currency: '₹',
			rating: 5.0,
			verifiedLicense: true,
			status: 'on_trip',
			currentTripNote: 'Active: Ananya & Vignesh (Kerala Restmode)',
			languages: ['Tamil', 'Malayalam', 'English']
		},
		{
			id: 'drv-5',
			name: 'Tan Wei Meng',
			agencyName: 'KL Elite Executive Transfers Sdn Bhd',
			agencyId: 'agency-3',
			phone: '+60 16 987 6543',
			whatsapp: '+60 16 987 6543',
			area: 'KLIA Terminal 1/2 ➔ KLCC ➔ Genting Highlands ➔ Malacca',
			regionCode: 'malaysia',
			vehicleModel: 'Mercedes-Benz Sprinter Executive Coach',
			vehiclePlate: 'WXY 8892',
			vehicleType: 'Luxury Coach',
			capacityPax: 14,
			dailyRate: 7500,
			currency: '₹',
			rating: 4.9,
			verifiedLicense: true,
			status: 'available',
			languages: ['English', 'Malay', 'Tamil (Basic)']
		},
		{
			id: 'drv-6',
			name: 'Tariq Al-Mansoor',
			agencyName: 'Dubai Desert Star Limousine LLC',
			agencyId: 'agency-4',
			phone: '+971 50 123 4567',
			whatsapp: '+971 50 123 4567',
			area: 'Dubai Airport (DXB) ➔ Marina ➔ Desert Safari ➔ Abu Dhabi',
			regionCode: 'dubai',
			vehicleModel: 'Toyota Land Cruiser 4x4 V8 (Safari Ready)',
			vehiclePlate: 'DXB-55912',
			vehicleType: 'SUV 4x4',
			capacityPax: 6,
			dailyRate: 6500,
			currency: '₹',
			rating: 4.8,
			verifiedLicense: true,
			status: 'available',
			languages: ['Arabic', 'English', 'Hindi', 'Tamil (Basic)']
		},
		{
			id: 'drv-7',
			name: 'R. Murugan',
			agencyName: 'South Tamil Nadu Tourist Taxi Network',
			agencyId: 'agency-5',
			phone: '+91 94432 11002',
			whatsapp: '+91 94432 11002',
			area: 'Madurai Meenakshi ➔ Rameswaram ➔ Kanniyakumari Cape',
			regionCode: 'tamilnadu',
			vehicleModel: 'Force Urbania AC (17 Seater)',
			vehiclePlate: 'TN-58-AX-9910',
			vehicleType: 'Tempo Traveller',
			capacityPax: 17,
			dailyRate: 5500,
			currency: '₹',
			rating: 4.7,
			verifiedLicense: true,
			status: 'available',
			languages: ['Tamil', 'English']
		}
	]);

	// Filtered Drivers
	let filteredDrivers = $derived.by(() => {
		return drivers.filter((d) => {
			if (searchQuery.trim()) {
				const q = searchQuery.toLowerCase();
				const matchName = d.name.toLowerCase().includes(q);
				const matchAgency = d.agencyName.toLowerCase().includes(q);
				const matchPlate = d.vehiclePlate.toLowerCase().includes(q);
				const matchArea = d.area.toLowerCase().includes(q);
				const matchPhone = d.phone.includes(q);
				if (!matchName && !matchAgency && !matchPlate && !matchArea && !matchPhone) return false;
			}
			if (filterRegion !== 'all' && d.regionCode !== filterRegion) return false;
			if (filterVehicleType !== 'all' && d.vehicleType !== filterVehicleType) return false;
			if (filterStatus !== 'all' && d.status !== filterStatus) return false;
			return true;
		});
	});

	// Filtered Agencies
	let filteredAgencies = $derived.by(() => {
		return agencies.filter((a) => {
			if (searchQuery.trim()) {
				const q = searchQuery.toLowerCase();
				const matchName = a.agencyName.toLowerCase().includes(q);
				const matchOwner = a.ownerName.toLowerCase().includes(q);
				const matchCity = a.baseCity.toLowerCase().includes(q);
				if (!matchName && !matchOwner && !matchCity) return false;
			}
			if (filterRegion !== 'all' && a.regionCode !== filterRegion) return false;
			return true;
		});
	});

	function handleAddDriver(e: SubmitEvent) {
		e.preventDefault();
		const form = e.target as HTMLFormElement;
		const formData = new FormData(form);
		const newD: Driver = {
			id: 'drv-' + Date.now(),
			name: formData.get('name') as string,
			agencyName: formData.get('agencyName') as string,
			agencyId: 'agency-custom',
			phone: formData.get('phone') as string,
			whatsapp: formData.get('phone') as string,
			area: formData.get('area') as string,
			regionCode: formData.get('regionCode') as any,
			vehicleModel: formData.get('vehicleModel') as string,
			vehiclePlate: formData.get('vehiclePlate') as string,
			vehicleType: formData.get('vehicleType') as any,
			capacityPax: Number(formData.get('capacityPax')) || 4,
			dailyRate: Number(formData.get('dailyRate')) || 3000,
			currency: '₹',
			rating: 5.0,
			verifiedLicense: true,
			status: 'available',
			languages: (formData.get('languages') as string).split(',').map(s => s.trim())
		};

		drivers.unshift(newD);
		showAddDriverModal = false;
		toast.success(`Driver ${newD.name} registered under ${newD.agencyName}!`);
	}

	function handleAddAgency(e: SubmitEvent) {
		e.preventDefault();
		const form = e.target as HTMLFormElement;
		const formData = new FormData(form);
		const newA: CarAgency = {
			id: 'agency-' + Date.now(),
			agencyName: formData.get('agencyName') as string,
			ownerName: formData.get('ownerName') as string,
			phone: formData.get('phone') as string,
			whatsapp: formData.get('phone') as string,
			email: formData.get('email') as string,
			baseCity: formData.get('baseCity') as string,
			regionCode: formData.get('regionCode') as any,
			totalVehicles: Number(formData.get('totalVehicles')) || 10,
			fleetBreakdown: formData.get('fleetBreakdown') as string,
			paymentTerms: formData.get('paymentTerms') as string,
			contractStatus: 'Verified Partner',
			gstOrTaxId: formData.get('gstOrTaxId') as string
		};

		agencies.unshift(newA);
		showAddAgencyModal = false;
		toast.success(`Car Fleet Agency ${newA.agencyName} onboarded successfully!`);
	}
</script>

<svelte:head>
	<title>Driver &amp; Car Agency Contact Management | Travel &amp; Tours OS</title>
</svelte:head>

<div class="drivers-mgmt-canvas">
	<!-- Top Masthead Control Panel -->
	<header class="top-masthead">
		<div class="masthead-main">
			<div>
				<div class="breadcrumbs-row">
					<a href="/travel" class="crumb-link">✈️ Travel &amp; Tours OS</a>
					<span class="crumb-sep">/</span>
					<span class="crumb-active">Driver &amp; Car Agency Operations</span>
				</div>
				<h1 class="page-title">🚗 Drivers &amp; Fleet Agency Management</h1>
				<p class="page-sub">
					Organize chauffeurs, vehicle models, and transport fleet agencies area-wise, tour-wise, and trip-wise.
				</p>
			</div>

			<!-- Quick Actions -->
			<div class="top-actions">
				<a href="/travel/book" class="btn btn-emerald btn-sm">
					⚡ 1-Page Book Trip
				</a>
				<button type="button" class="btn btn-outline btn-sm" onclick={() => (showAddAgencyModal = true)}>
					🏢 + Add Fleet Agency
				</button>
				<button type="button" class="btn btn-primary btn-sm" onclick={() => (showAddDriverModal = true)}>
					🚗 + Register Driver
				</button>
			</div>
		</div>

		<!-- Nav Tabs & Search Filter Strip -->
		<div class="sub-controls-bar">
			<!-- Tab Switcher -->
			<div class="tab-toggle-group">
				<button 
					type="button" 
					class="tab-btn {activeTab === 'drivers' ? 'active' : ''}" 
					onclick={() => (activeTab = 'drivers')}
				>
					🚗 Drivers Directory ({filteredDrivers.length})
				</button>
				<button 
					type="button" 
					class="tab-btn {activeTab === 'agencies' ? 'active' : ''}" 
					onclick={() => (activeTab = 'agencies')}
				>
					🏢 Car Agencies &amp; Fleet Owners ({filteredAgencies.length})
				</button>
			</div>

			<!-- Filter Selectors -->
			<div class="filters-wrap">
				<div class="search-box">
					<span class="search-icon">🔍</span>
					<input 
						type="text" 
						bind:value={searchQuery} 
						placeholder="Search by driver, agency, phone, area, or plate number..."
						class="search-input"
					/>
				</div>

				<!-- Region Selector -->
				<select bind:value={filterRegion} class="filter-select">
					<option value="all">📍 All Operating Areas</option>
					<option value="kashmir">Kashmir (Srinagar / Gulmarg / Pahalgam)</option>
					<option value="kerala">Kerala (Cochin / Munnar / Alleppey)</option>
					<option value="malaysia">Malaysia (Kuala Lumpur / Genting)</option>
					<option value="dubai">Dubai &amp; Abu Dhabi (UAE)</option>
					<option value="tamilnadu">Tamil Nadu (Chennai / Madurai / Kanniyakumari)</option>
				</select>

				{#if activeTab === 'drivers'}
					<!-- Vehicle Type -->
					<select bind:value={filterVehicleType} class="filter-select">
						<option value="all">🚙 All Vehicle Types</option>
						<option value="Innova Crysta">Innova Crysta (7-Seater)</option>
						<option value="Prime Sedan">Prime Sedan (4-Seater)</option>
						<option value="Tempo Traveller">Tempo Traveller (17-Seater)</option>
						<option value="Luxury Coach">Luxury Coach / Bus</option>
						<option value="SUV 4x4">SUV 4x4 (Desert / Snow)</option>
					</select>

					<!-- Status -->
					<select bind:value={filterStatus} class="filter-select">
						<option value="all">Status: All</option>
						<option value="available">🟢 Available</option>
						<option value="on_trip">🟡 On Active Trip</option>
						<option value="off_duty">🔴 Off Duty</option>
					</select>
				{/if}
			</div>
		</div>
	</header>

	<!-- Main Body: Drivers Grid or Agency Table -->
	{#if activeTab === 'drivers'}
		<div class="drivers-cards-grid">
			{#each filteredDrivers as drv}
				<div class="driver-card">
					<!-- Top Row: Avatar, Name, Rating -->
					<div class="card-top-row">
						<div class="driver-profile-info">
							<div class="driver-avatar-circle">
								{drv.name.split(' ').map(n => n[0]).join('')}
							</div>
							<div>
								<div class="driver-name-title">
									<h3 class="d-name">{drv.name}</h3>
									{#if drv.verifiedLicense}
										<span class="verified-badge" title="Police & Transport License Verified">✓ Verified</span>
									{/if}
								</div>
								<span class="agency-affiliation">🏢 {drv.agencyName}</span>
							</div>
						</div>
						<div class="status-rating-col">
							<span class="status-pill {drv.status}">
								{#if drv.status === 'available'}🟢 Available
								{:else if drv.status === 'on_trip'}🟡 On Trip
								{:else}🔴 Off Duty{/if}
							</span>
							<span class="rating-badge">★ {drv.rating}</span>
						</div>
					</div>

					<!-- Operating Area & Circuit -->
					<div class="driver-area-box">
						<span class="area-label">📍 OPERATING CIRCUIT / AREA:</span>
						<span class="area-text">{drv.area}</span>
					</div>

					<!-- Active Trip Banner if on trip -->
					{#if drv.status === 'on_trip' && drv.currentTripNote}
						<div class="active-trip-banner">
							<span>🚗 {drv.currentTripNote}</span>
						</div>
					{/if}

					<!-- Vehicle Specs -->
					<div class="vehicle-specs-grid">
						<div class="spec-cell">
							<span class="spec-lbl">Vehicle Model</span>
							<span class="spec-val font-semibold">{drv.vehicleModel}</span>
						</div>
						<div class="spec-cell">
							<span class="spec-lbl">Plate Registration</span>
							<span class="spec-val font-mono font-bold">{drv.vehiclePlate}</span>
						</div>
						<div class="spec-cell">
							<span class="spec-lbl">Capacity</span>
							<span class="spec-val">{drv.capacityPax} Pax + Luggage</span>
						</div>
						<div class="spec-cell">
							<span class="spec-lbl">Per-Day Rate</span>
							<span class="spec-val font-mono font-bold text-emerald">{drv.currency} {drv.dailyRate.toLocaleString('en-IN')} / day</span>
						</div>
					</div>

					<!-- Spoken Languages -->
					<div class="languages-row">
						<span class="lang-lbl">🗣️ Languages:</span>
						<div class="lang-chips">
							{#each drv.languages as l}
								<span class="lang-chip">{l}</span>
							{/each}
						</div>
					</div>

					<!-- Action Buttons -->
					<div class="card-footer-actions">
						<a 
							href={`https://wa.me/${drv.whatsapp.replace(/[^0-9]/g, '')}?text=${encodeURIComponent(`Hello ${drv.name}, regarding tour trip pickup dispatch...`)}`} 
							target="_blank" 
							class="btn btn-whatsapp btn-xs"
						>
							📱 WhatsApp Driver
						</a>
						<a href={`tel:${drv.phone}`} class="btn btn-secondary btn-xs font-mono">
							📞 {drv.phone}
						</a>
						<a href={`/travel/book`} class="btn btn-primary btn-xs" title="Assign this driver in booking desk">
							⚡ Assign Trip
						</a>
					</div>
				</div>
			{/each}
		</div>
	{:else}
		<!-- Car Agencies Table View -->
		<div class="agencies-table-card">
			<div class="table-responsive">
				<table class="agencies-table">
					<thead>
						<tr>
							<th>Car Agency / Fleet Supplier</th>
							<th>Owner / Dispatch Manager</th>
							<th>Operating City / Base</th>
							<th>Fleet Size &amp; Models</th>
							<th>Payment Terms &amp; Settlement</th>
							<th>GST / Tax ID</th>
							<th>Contract Status</th>
							<th>Contact</th>
						</tr>
					</thead>
					<tbody>
						{#each filteredAgencies as ag}
							<tr>
								<td>
									<div class="agency-title-cell">
										<span class="ag-name font-bold">{ag.agencyName}</span>
										<span class="ag-email text-xs">{ag.email}</span>
									</div>
								</td>
								<td>
									<span class="owner-name font-semibold">{ag.ownerName}</span>
									<span class="owner-phone font-mono text-xs block">{ag.phone}</span>
								</td>
								<td>📍 {ag.baseCity}</td>
								<td>
									<span class="fleet-count font-bold font-mono text-blue">{ag.totalVehicles} Vehicles</span>
									<span class="fleet-desc text-xs block">{ag.fleetBreakdown}</span>
								</td>
								<td>
									<span class="terms-text text-xs">{ag.paymentTerms}</span>
								</td>
								<td class="font-mono text-xs font-semibold">{ag.gstOrTaxId}</td>
								<td>
									<span class="contract-badge {ag.contractStatus.toLowerCase().replace(/ /g, '-')}">
										{ag.contractStatus}
									</span>
								</td>
								<td>
									<a 
										href={`https://wa.me/${ag.whatsapp.replace(/[^0-9]/g, '')}?text=${encodeURIComponent(`Hello ${ag.ownerName}, regarding car fleet requirements from Travel & Tours OS...`)}`}
										target="_blank" 
										class="btn btn-whatsapp btn-xs"
									>
										📱 Contact
									</a>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</div>
	{/if}
</div>

<!-- =========================================================================
     MODAL: REGISTER DRIVER
     ========================================================================= -->
{#if showAddDriverModal}
	<div class="modal-backdrop" onclick={() => (showAddDriverModal = false)}>
		<div class="modal-card" onclick={(e) => e.stopPropagation()} role="dialog">
			<div class="modal-header">
				<h3>🚗 Register New Chauffeur / Driver</h3>
				<button type="button" class="close-x" onclick={() => (showAddDriverModal = false)}>✕</button>
			</div>
			<form onsubmit={handleAddDriver}>
				<div class="modal-body-grid">
					<div class="form-group">
						<label class="form-label" for="drv-name">Driver Full Name *</label>
						<input id="drv-name" name="name" type="text" placeholder="e.g. Ghulam Nabi" required class="form-input" />
					</div>
					<div class="form-group">
						<label class="form-label" for="drv-agency">Associated Car Agency *</label>
						<select id="drv-agency" name="agencyName" class="form-select" required>
							{#each agencies as ag}
								<option value={ag.agencyName}>{ag.agencyName}</option>
							{/each}
						</select>
					</div>
					<div class="form-group">
						<label class="form-label" for="drv-phone">Mobile / WhatsApp Number *</label>
						<input id="drv-phone" name="phone" type="text" placeholder="+91 94190 22334" required class="form-input font-mono" />
					</div>
					<div class="form-group">
						<label class="form-label" for="drv-reg">Area / Operating Region *</label>
						<select id="drv-reg" name="regionCode" class="form-select" required>
							<option value="kashmir">Kashmir (Srinagar/Gulmarg/Pahalgam)</option>
							<option value="kerala">Kerala (Cochin/Munnar/Alleppey)</option>
							<option value="malaysia">Malaysia (Kuala Lumpur/Genting)</option>
							<option value="dubai">Dubai &amp; Abu Dhabi (UAE)</option>
							<option value="tamilnadu">Tamil Nadu (Chennai/Madurai)</option>
						</select>
					</div>
					<div class="form-group full-width">
						<label class="form-label" for="drv-area">Operating Circuit / Destination Routes *</label>
						<input id="drv-area" name="area" type="text" placeholder="e.g. Srinagar Airport ➔ Gulmarg ➔ Pahalgam Circuit" required class="form-input" />
					</div>
					<div class="form-group">
						<label class="form-label" for="drv-veh">Vehicle Model *</label>
						<input id="drv-veh" name="vehicleModel" type="text" placeholder="Toyota Innova Crysta AC (White)" required class="form-input" />
					</div>
					<div class="form-group">
						<label class="form-label" for="drv-plate">Registration Plate Number *</label>
						<input id="drv-plate" name="vehiclePlate" type="text" placeholder="JK-01-AB-4491" required class="form-input font-mono" />
					</div>
					<div class="form-group">
						<label class="form-label" for="drv-type">Vehicle Category</label>
						<select id="drv-type" name="vehicleType" class="form-select">
							<option value="Innova Crysta">Innova Crysta (7-Seater)</option>
							<option value="Prime Sedan">Prime Sedan (4-Seater)</option>
							<option value="Tempo Traveller">Tempo Traveller (17-Seater)</option>
							<option value="Luxury Coach">Luxury Coach</option>
							<option value="SUV 4x4">SUV 4x4</option>
						</select>
					</div>
					<div class="form-group">
						<label class="form-label" for="drv-rate">Per-Day Rate (₹) *</label>
						<input id="drv-rate" name="dailyRate" type="number" placeholder="3500" required class="form-input font-mono" />
					</div>
					<div class="form-group full-width">
						<label class="form-label" for="drv-lang">Languages Spoken (comma separated)</label>
						<input id="drv-lang" name="languages" type="text" placeholder="Tamil, English, Hindi, Kashmiri" class="form-input" />
					</div>
				</div>
				<div class="modal-footer">
					<button type="button" class="btn btn-secondary" onclick={() => (showAddDriverModal = false)}>Cancel</button>
					<button type="submit" class="btn btn-primary">🚗 Register Driver</button>
				</div>
			</form>
		</div>
	</div>
{/if}

<!-- =========================================================================
     MODAL: ONBOARD CAR FLEET AGENCY
     ========================================================================= -->
{#if showAddAgencyModal}
	<div class="modal-backdrop" onclick={() => (showAddAgencyModal = false)}>
		<div class="modal-card" onclick={(e) => e.stopPropagation()} role="dialog">
			<div class="modal-header">
				<h3>🏢 Onboard Car Agency / Fleet Supplier</h3>
				<button type="button" class="close-x" onclick={() => (showAddAgencyModal = false)}>✕</button>
			</div>
			<form onsubmit={handleAddAgency}>
				<div class="modal-body-grid">
					<div class="form-group">
						<label class="form-label" for="ag-name">Agency / Fleet Name *</label>
						<input id="ag-name" name="agencyName" type="text" placeholder="e.g. Royal Kashmir Transport Consortium" required class="form-input" />
					</div>
					<div class="form-group">
						<label class="form-label" for="ag-owner">Owner / Contact Person *</label>
						<input id="ag-owner" name="ownerName" type="text" placeholder="e.g. Fayaz Ahmed Mir" required class="form-input" />
					</div>
					<div class="form-group">
						<label class="form-label" for="ag-phone">Mobile / WhatsApp Number *</label>
						<input id="ag-phone" name="phone" type="text" placeholder="+91 94190 88221" required class="form-input font-mono" />
					</div>
					<div class="form-group">
						<label class="form-label" for="ag-email">Email Address</label>
						<input id="ag-email" name="email" type="email" placeholder="contact@agency.com" class="form-input" />
					</div>
					<div class="form-group">
						<label class="form-label" for="ag-city">Base City &amp; State *</label>
						<input id="ag-city" name="baseCity" type="text" placeholder="Srinagar, J&K" required class="form-input" />
					</div>
					<div class="form-group">
						<label class="form-label" for="ag-region">Operating Region *</label>
						<select id="ag-region" name="regionCode" class="form-select" required>
							<option value="kashmir">Kashmir Circuit</option>
							<option value="kerala">Kerala Circuit</option>
							<option value="malaysia">Malaysia Circuit</option>
							<option value="dubai">Dubai &amp; UAE</option>
							<option value="tamilnadu">Tamil Nadu Circuit</option>
						</select>
					</div>
					<div class="form-group">
						<label class="form-label" for="ag-fleet">Total Fleet Size</label>
						<input id="ag-fleet" name="totalVehicles" type="number" placeholder="20" class="form-input font-mono" />
					</div>
					<div class="form-group">
						<label class="form-label" for="ag-gst">GSTIN / Commercial Tax ID</label>
						<input id="ag-gst" name="gstOrTaxId" type="text" placeholder="01AAACH8849J1Z4" class="form-input font-mono" />
					</div>
					<div class="form-group full-width">
						<label class="form-label" for="ag-break">Fleet Breakdown &amp; Models</label>
						<input id="ag-break" name="fleetBreakdown" type="text" placeholder="12 Innovas, 6 Sedans, 2 Tempo Travellers" class="form-input" />
					</div>
					<div class="form-group full-width">
						<label class="form-label" for="ag-pay">Payment Terms &amp; Settlement</label>
						<input id="ag-pay" name="paymentTerms" type="text" placeholder="50% on Airport Pickup, Balance on Pahalgam checkout" class="form-input" />
					</div>
				</div>
				<div class="modal-footer">
					<button type="button" class="btn btn-secondary" onclick={() => (showAddAgencyModal = false)}>Cancel</button>
					<button type="submit" class="btn btn-primary">🏢 Onboard Agency</button>
				</div>
			</form>
		</div>
	</div>
{/if}

<style>
	/* ==========================================================================
	   DRIVERS & FLEET AGENCY STYLING
	   ========================================================================== */

	.drivers-mgmt-canvas {
		display: flex;
		flex-direction: column;
		min-height: calc(100vh - 64px);
		background: #f8fafc;
		color: #0f172a;
		font-family: var(--font-sans, system-ui, -apple-system, sans-serif);
		padding: 1.25rem 2rem;
		gap: 1.25rem;
		box-sizing: border-box;
	}

	.top-masthead {
		background: #ffffff;
		border: 1px solid #e2e8f0;
		border-radius: 12px;
		padding: 1.25rem 1.5rem;
		box-shadow: 0 1px 3px rgba(0,0,0,0.04);
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}

	.masthead-main {
		display: flex;
		justify-content: space-between;
		align-items: center;
		flex-wrap: wrap;
		gap: 1rem;
	}

	.breadcrumbs-row {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 0.8125rem;
		margin-bottom: 4px;
	}

	.crumb-link {
		color: #64748b;
		text-decoration: none;
		font-weight: 600;
	}
	.crumb-link:hover { color: #2563eb; }
	.crumb-sep { color: #cbd5e1; }
	.crumb-active { color: #0f172a; font-weight: 700; }

	.page-title {
		font-size: 1.4rem;
		font-weight: 800;
		margin: 0;
		color: #0f172a;
		letter-spacing: -0.02em;
	}

	.page-sub {
		font-size: 0.8125rem;
		color: #64748b;
		margin: 4px 0 0 0;
	}

	.top-actions {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		flex-wrap: wrap;
	}

	/* Sub-Controls & Filters */
	.sub-controls-bar {
		display: flex;
		justify-content: space-between;
		align-items: center;
		flex-wrap: wrap;
		gap: 0.75rem;
		padding-top: 0.75rem;
		border-top: 1px solid #f1f5f9;
	}

	.tab-toggle-group {
		display: flex;
		background: #f1f5f9;
		border-radius: 8px;
		padding: 3px;
		border: 1px solid #e2e8f0;
	}

	.tab-btn {
		background: transparent;
		border: none;
		padding: 6px 14px;
		border-radius: 6px;
		font-size: 0.8125rem;
		font-weight: 700;
		color: #64748b;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.tab-btn.active {
		background: #ffffff;
		color: #0f172a;
		box-shadow: 0 1px 3px rgba(0,0,0,0.06);
	}

	.filters-wrap {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		flex-wrap: wrap;
	}

	.search-box {
		position: relative;
		display: flex;
		align-items: center;
		min-width: 280px;
	}

	.search-icon {
		position: absolute;
		left: 10px;
		font-size: 0.8rem;
		color: #94a3b8;
	}

	.search-input {
		width: 100%;
		padding: 6px 12px 6px 30px;
		border-radius: 6px;
		border: 1px solid #cbd5e1;
		font-size: 0.8125rem;
		outline: none;
	}

	.filter-select {
		padding: 6px 10px;
		border-radius: 6px;
		border: 1px solid #cbd5e1;
		font-size: 0.8125rem;
		background: #ffffff;
		color: #334155;
		font-weight: 600;
	}

	/* Drivers Grid */
	.drivers-cards-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(380px, 1fr));
		gap: 1.25rem;
	}

	.driver-card {
		background: #ffffff;
		border: 1px solid #e2e8f0;
		border-radius: 12px;
		padding: 1.25rem;
		box-shadow: 0 1px 3px rgba(0,0,0,0.04);
		display: flex;
		flex-direction: column;
		gap: 0.85rem;
		transition: transform 0.12s ease, box-shadow 0.12s ease;
	}

	.driver-card:hover {
		transform: translateY(-2px);
		box-shadow: 0 4px 12px rgba(0,0,0,0.08);
	}

	.card-top-row {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
	}

	.driver-profile-info {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}

	.driver-avatar-circle {
		width: 44px;
		height: 44px;
		border-radius: 50%;
		background: #eff6ff;
		color: #2563eb;
		border: 1px solid #bfdbfe;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 1rem;
		font-weight: 800;
	}

	.driver-name-title {
		display: flex;
		align-items: center;
		gap: 0.4rem;
	}

	.d-name {
		font-size: 1.05rem;
		font-weight: 800;
		color: #0f172a;
		margin: 0;
	}

	.verified-badge {
		background: #ecfdf5;
		color: #059669;
		font-size: 0.65rem;
		font-weight: 800;
		padding: 1px 6px;
		border-radius: 4px;
		border: 1px solid #a7f3d0;
	}

	.agency-affiliation {
		font-size: 0.75rem;
		color: #64748b;
		display: block;
		margin-top: 2px;
	}

	.status-rating-col {
		display: flex;
		flex-direction: column;
		align-items: flex-end;
		gap: 0.25rem;
	}

	.status-pill {
		font-size: 0.7rem;
		font-weight: 700;
		padding: 2px 7px;
		border-radius: 12px;
	}

	.status-pill.available { background: #dcfce7; color: #166534; }
	.status-pill.on_trip { background: #fef3c7; color: #92400e; }
	.status-pill.off_duty { background: #fee2e2; color: #991b1b; }

	.rating-badge {
		font-size: 0.75rem;
		font-weight: 700;
		color: #eab308;
	}

	.driver-area-box {
		background: #f8fafc;
		border-radius: 6px;
		padding: 6px 10px;
		font-size: 0.75rem;
	}

	.area-label {
		font-weight: 700;
		color: #64748b;
		margin-right: 4px;
	}

	.area-text {
		color: #1e40af;
		font-weight: 600;
	}

	.active-trip-banner {
		background: #fffbeb;
		border: 1px solid #fde68a;
		border-radius: 6px;
		padding: 6px 10px;
		font-size: 0.75rem;
		color: #92400e;
		font-weight: 700;
	}

	.vehicle-specs-grid {
		display: grid;
		grid-template-columns: repeat(2, 1fr);
		gap: 0.5rem;
		background: #f8fafc;
		padding: 0.75rem;
		border-radius: 8px;
		border: 1px solid #f1f5f9;
	}

	.spec-cell {
		display: flex;
		flex-direction: column;
	}

	.spec-lbl {
		font-size: 0.65rem;
		color: #64748b;
		text-transform: uppercase;
		font-weight: 600;
	}

	.spec-val {
		font-size: 0.8125rem;
		color: #0f172a;
	}

	.languages-row {
		display: flex;
		align-items: center;
		gap: 0.4rem;
		font-size: 0.75rem;
	}

	.lang-lbl {
		color: #64748b;
		font-weight: 600;
	}

	.lang-chips {
		display: flex;
		gap: 0.25rem;
		flex-wrap: wrap;
	}

	.lang-chip {
		background: #f1f5f9;
		color: #334155;
		padding: 1px 6px;
		border-radius: 4px;
		font-size: 0.7rem;
	}

	.card-footer-actions {
		display: flex;
		gap: 0.5rem;
		border-top: 1px solid #f1f5f9;
		padding-top: 0.75rem;
		margin-top: auto;
	}

	/* Agencies Table */
	.agencies-table-card {
		background: #ffffff;
		border-radius: 12px;
		border: 1px solid #e2e8f0;
		padding: 1rem;
		box-shadow: 0 1px 3px rgba(0,0,0,0.03);
	}

	.table-responsive {
		overflow-x: auto;
	}

	.agencies-table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.8125rem;
	}

	.agencies-table th {
		background: #f8fafc;
		padding: 0.65rem 0.75rem;
		text-align: left;
		font-weight: 700;
		color: #475569;
		border-bottom: 1px solid #cbd5e1;
	}

	.agencies-table td {
		padding: 0.65rem 0.75rem;
		border-bottom: 1px solid #f1f5f9;
		vertical-align: middle;
	}

	.contract-badge {
		font-size: 0.65rem;
		font-weight: 700;
		padding: 2px 7px;
		border-radius: 4px;
	}
	.contract-badge.verified-partner { background: #dcfce7; color: #166534; }
	.contract-badge.direct-fleet { background: #dbeafe; color: #1e40af; }

	/* Modals */
	.modal-backdrop {
		position: fixed;
		inset: 0;
		background: rgba(15, 23, 42, 0.5);
		backdrop-filter: blur(3px);
		z-index: 9999;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 1rem;
	}

	.modal-card {
		background: #ffffff;
		border-radius: 12px;
		width: 640px;
		max-width: 95vw;
		box-shadow: 0 20px 40px rgba(0,0,0,0.2);
		overflow: hidden;
	}

	.modal-header {
		padding: 1rem 1.25rem;
		display: flex;
		justify-content: space-between;
		align-items: center;
		border-bottom: 1px solid #e2e8f0;
	}
	.modal-header h3 { margin: 0; font-size: 1.1rem; font-weight: 800; }
	.close-x { background: transparent; border: none; font-size: 1.1rem; cursor: pointer; }

	.modal-body-grid {
		padding: 1.25rem;
		display: grid;
		grid-template-columns: repeat(2, 1fr);
		gap: 0.85rem;
	}

	.form-group {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}
	.form-group.full-width { grid-column: span 2; }

	.form-label { font-size: 0.75rem; font-weight: 700; color: #475569; }
	.form-input, .form-select {
		padding: 7px 10px;
		border-radius: 6px;
		border: 1px solid #cbd5e1;
		font-size: 0.8125rem;
	}

	.modal-footer {
		padding: 0.85rem 1.25rem;
		background: #f8fafc;
		border-top: 1px solid #e2e8f0;
		display: flex;
		justify-content: flex-end;
		gap: 0.5rem;
	}

	/* General Buttons & Badges */
	.btn {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		padding: 0.45rem 0.85rem;
		border-radius: 6px;
		font-weight: 600;
		font-size: 0.8125rem;
		cursor: pointer;
		border: 1px solid transparent;
		transition: all 0.15s ease;
		text-decoration: none;
	}

	.btn-primary { background: #2563eb; color: #ffffff; }
	.btn-primary:hover { background: #1d4ed8; }
	.btn-secondary { background: #ffffff; border-color: #cbd5e1; color: #334155; }
	.btn-secondary:hover { background: #f8fafc; }
	.btn-outline { background: transparent; border-color: #cbd5e1; color: #475569; }
	.btn-outline:hover { background: #f1f5f9; }
	.btn-emerald { background: #10b981; color: #ffffff; }
	.btn-emerald:hover { background: #059669; }
	.btn-whatsapp { background: #22c55e; color: #ffffff; }
	.btn-whatsapp:hover { background: #16a34a; }
	.btn-sm { padding: 0.35rem 0.7rem; font-size: 0.75rem; }
	.btn-xs { padding: 0.25rem 0.55rem; font-size: 0.7rem; }

	.font-mono { font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace; }
	.font-bold { font-weight: 700; }
	.font-semibold { font-weight: 600; }
	.text-blue { color: #2563eb; }
	.text-emerald { color: #059669; }
	.text-xs { font-size: 0.7rem; }
</style>
