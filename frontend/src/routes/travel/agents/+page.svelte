<script lang="ts">
	import { toast } from '$lib/stores/toast.svelte';

	interface TourAgent {
		id: string;
		agencyName: string;
		contactPerson: string;
		designation: string;
		phone: string;
		whatsapp: string;
		email: string;
		city: string;
		state: string;
		regionCode: 'tamilnadu' | 'kerala' | 'northindia' | 'international';
		agentType: 'Retail Travel Agency' | 'Corporate Travel Desk' | 'B2B Sub-Agent' | 'Inbound DMC Partner';
		commissionRatePercent: number; // e.g. 10%
		creditLimit: number; // in INR
		currentReceivable: number; // outstanding balance
		activeBookings: number;
		totalTripsVolume: number;
		rating: number;
		kycVerified: boolean;
		iataOrGst: string;
		status: 'active' | 'suspended' | 'pending_kyc';
		lastActive: string;
		primaryMarkets: string[]; // e.g. ["Kashmir Honeymoon", "Malaysia Family", "Kerala Ayurvedic"]
	}

	// State & Filters
	let searchQuery = $state('');
	let filterRegion = $state('all');
	let filterType = $state('all');
	let filterStatus = $state('all');

	// Modals & Drawers
	let showAddAgentModal = $state(false);
	let selectedAgentForKhata = $state<TourAgent | null>(null);

	// Dataset of Area-Wise Tour Agents & B2B Partners
	let agents = $state<TourAgent[]>([
		{
			id: 'agent-1',
			agencyName: 'Madurai Pandian Travel World',
			contactPerson: 'S. Alagarsamy',
			designation: 'Managing Partner',
			phone: '+91 94432 88110',
			whatsapp: '+91 94432 88110',
			email: 'sales@maduraipandiantravels.in',
			city: 'Madurai',
			state: 'Tamil Nadu',
			regionCode: 'tamilnadu',
			agentType: 'Retail Travel Agency',
			commissionRatePercent: 10,
			creditLimit: 300000,
			currentReceivable: 65000,
			activeBookings: 6,
			totalTripsVolume: 84,
			rating: 4.9,
			kycVerified: true,
			iataOrGst: '33AABCM4412F1Z8',
			status: 'active',
			lastActive: '2 hours ago',
			primaryMarkets: ['Malaysia Family', 'Kashmir Winter', 'Singapore Group']
		},
		{
			id: 'agent-2',
			agencyName: 'Chennai Marina Holidays & Corporate Corp',
			contactPerson: 'Vigneshwaran Sundaram',
			designation: 'Head of B2B Ticketing',
			phone: '+91 98401 55662',
			whatsapp: '+91 98401 55662',
			email: 'vignesh@marinaholidays.com',
			city: 'Chennai',
			state: 'Tamil Nadu',
			regionCode: 'tamilnadu',
			agentType: 'Corporate Travel Desk',
			commissionRatePercent: 8,
			creditLimit: 800000,
			currentReceivable: 142000,
			activeBookings: 12,
			totalTripsVolume: 190,
			rating: 4.95,
			kycVerified: true,
			iataOrGst: '33AAACM9823P1ZQ',
			status: 'active',
			lastActive: 'Just now',
			primaryMarkets: ['Dubai Corporate MICE', 'Kashmir Deluxe', 'Europe Leisure']
		},
		{
			id: 'agent-3',
			agencyName: 'Malabar Spice Routes Travel Desk',
			contactPerson: 'Muhammed Shameer',
			designation: 'Operations Director',
			phone: '+91 98472 99001',
			whatsapp: '+91 98472 99001',
			email: 'shameer@malabarroutes.in',
			city: 'Calicut (Kozhikode)',
			state: 'Kerala',
			regionCode: 'kerala',
			agentType: 'B2B Sub-Agent',
			commissionRatePercent: 12,
			creditLimit: 400000,
			currentReceivable: 85000,
			activeBookings: 4,
			totalTripsVolume: 62,
			rating: 4.85,
			kycVerified: true,
			iataOrGst: '32AADCM2291Q1ZM',
			status: 'active',
			lastActive: 'Yesterday',
			primaryMarkets: ['Kerala Backwater Houseboat', 'Dubai Shopping', 'Malaysia Tour']
		},
		{
			id: 'agent-4',
			agencyName: 'Kanyakumari Ocean Breeze Tours',
			contactPerson: 'C. Paulraj',
			designation: 'Proprietor',
			phone: '+91 94862 33441',
			whatsapp: '+91 94862 33441',
			email: 'kanyakumari.oceanbreeze@gmail.com',
			city: 'Kanyakumari / Nagercoil',
			state: 'Tamil Nadu',
			regionCode: 'tamilnadu',
			agentType: 'Retail Travel Agency',
			commissionRatePercent: 10,
			creditLimit: 250000,
			currentReceivable: 32000,
			activeBookings: 3,
			totalTripsVolume: 48,
			rating: 4.8,
			kycVerified: true,
			iataOrGst: '33BBXPK1092D1Z3',
			status: 'active',
			lastActive: 'Today at 11:30 AM',
			primaryMarkets: ['Kerala Munnar/Alleppey', 'Kashmir Snow Trails']
		},
		{
			id: 'agent-5',
			agencyName: 'KL South-East Asia B2B Linkages Sdn Bhd',
			contactPerson: 'Dato’ Kenneth Tan & Farah',
			designation: 'Regional Partnership VP',
			phone: '+60 12 998 7766',
			whatsapp: '+60 12 998 7766',
			email: 'partnerships@sealinkages.my',
			city: 'Kuala Lumpur',
			state: 'Wilayah Persekutuan',
			regionCode: 'international',
			agentType: 'Inbound DMC Partner',
			commissionRatePercent: 15,
			creditLimit: 1500000,
			currentReceivable: 310000,
			activeBookings: 18,
			totalTripsVolume: 340,
			rating: 4.98,
			kycVerified: true,
			iataOrGst: 'MOTAC-MY-99201',
			status: 'active',
			lastActive: '5 mins ago',
			primaryMarkets: ['Malaysia Inbound', 'Singapore Twin-City', 'Thailand Add-ons']
		}
	]);

	// Filtered list
	let filteredAgents = $derived(
		agents.filter((a) => {
			const matchesQuery =
				searchQuery === '' ||
				a.agencyName.toLowerCase().includes(searchQuery.toLowerCase()) ||
				a.contactPerson.toLowerCase().includes(searchQuery.toLowerCase()) ||
				a.city.toLowerCase().includes(searchQuery.toLowerCase()) ||
				a.iataOrGst.toLowerCase().includes(searchQuery.toLowerCase()) ||
				a.primaryMarkets.some((m) => m.toLowerCase().includes(searchQuery.toLowerCase()));

			const matchesRegion = filterRegion === 'all' || a.regionCode === filterRegion;
			const matchesType = filterType === 'all' || a.agentType === filterType;
			const matchesStatus = filterStatus === 'all' || a.status === filterStatus;

			return matchesQuery && matchesRegion && matchesType && matchesStatus;
		})
	);

	// Summary KPI calculations
	let totalPartners = $derived(agents.length);
	let activePartners = $derived(agents.filter((a) => a.status === 'active').length);
	let totalActiveBookings = $derived(agents.reduce((acc, a) => acc + a.activeBookings, 0));
	let totalReceivables = $derived(agents.reduce((acc, a) => acc + a.currentReceivable, 0));

	// Add Agent Form
	let newAgent = $state({
		agencyName: '',
		contactPerson: '',
		designation: 'Managing Partner',
		phone: '',
		whatsapp: '',
		email: '',
		city: '',
		state: 'Tamil Nadu',
		regionCode: 'tamilnadu' as TourAgent['regionCode'],
		agentType: 'Retail Travel Agency' as TourAgent['agentType'],
		commissionRatePercent: 10,
		creditLimit: 300000,
		iataOrGst: '',
		primaryMarketsInput: 'Malaysia, Kashmir, Kerala'
	});

	function handleCreateAgent() {
		if (!newAgent.agencyName || !newAgent.contactPerson || !newAgent.phone || !newAgent.city) {
			toast.error('Please enter Agency Name, Contact Person, Phone, and City.');
			return;
		}

		const agent: TourAgent = {
			id: `agent-${Date.now()}`,
			agencyName: newAgent.agencyName,
			contactPerson: newAgent.contactPerson,
			designation: newAgent.designation,
			phone: newAgent.phone,
			whatsapp: newAgent.whatsapp || newAgent.phone,
			email: newAgent.email,
			city: newAgent.city,
			state: newAgent.state,
			regionCode: newAgent.regionCode,
			agentType: newAgent.agentType,
			commissionRatePercent: Number(newAgent.commissionRatePercent),
			creditLimit: Number(newAgent.creditLimit),
			currentReceivable: 0,
			activeBookings: 0,
			totalTripsVolume: 0,
			rating: 5.0,
			kycVerified: true,
			iataOrGst: newAgent.iataOrGst || `GST-TN-${Math.floor(1000 + Math.random() * 9000)}`,
			status: 'active',
			lastActive: 'Just registered',
			primaryMarkets: newAgent.primaryMarketsInput.split(',').map((s) => s.trim()).filter(Boolean)
		};

		agents = [agent, ...agents];
		showAddAgentModal = false;
		toast.success(`Onboarded B2B Partner "${agent.agencyName}" from ${agent.city}`);

		// reset
		newAgent = {
			agencyName: '',
			contactPerson: '',
			designation: 'Managing Partner',
			phone: '',
			whatsapp: '',
			email: '',
			city: '',
			state: 'Tamil Nadu',
			regionCode: 'tamilnadu',
			agentType: 'Retail Travel Agency',
			commissionRatePercent: 10,
			creditLimit: 300000,
			iataOrGst: '',
			primaryMarketsInput: 'Malaysia, Kashmir, Kerala'
		};
	}

	function getWhatsAppUrl(phone: string, person: string, agency: string) {
		const cleanPhone = phone.replace(/[^0-9]/g, '');
		const text = encodeURIComponent(
			`Vanakkam ${person} (${agency}), greetings from MrBiz OS B2B Desk! How can we assist with your upcoming group tours, visa quotes, or agent commission ledger today?`
		);
		return `https://wa.me/${cleanPhone}?text=${text}`;
	}
</script>

<div class="agents-root">
	<!-- Top Header -->
	<div class="header-card">
		<div class="title-cluster">
			<div class="title-badge">
				<span class="badge-icon">🤝</span>
				<span class="badge-label">B2B Network & Sub-Agent Portal</span>
			</div>
			<h1 class="main-title">Tour Agents & B2B Partner Management</h1>
			<p class="subtitle">
				Manage regional travel agencies, retail distributors, corporate booking desks, and overseas DMCs across <strong>Tamil Nadu (Chennai, Madurai, Kanyakumari)</strong>, <strong>Kerala</strong>, and <strong>Malaysia/Overseas</strong>. Monitor credit limits, commission slabs, and active pipeline bookings.
			</p>
		</div>

		<div class="header-actions">
			<a href="/travel/crm" class="action-btn secondary">
				<span>👥</span>
				<span>Odoo CRM Leads</span>
			</a>
			<button class="action-btn primary" onclick={() => (showAddAgentModal = true)}>
				<span>+</span>
				<span>Onboard Tour Agent</span>
			</button>
		</div>
	</div>

	<!-- KPI Metrics -->
	<div class="kpi-grid">
		<div class="kpi-card">
			<div class="kpi-header">
				<span class="kpi-icon">🏢</span>
				<span class="kpi-label">B2B Partners</span>
			</div>
			<div class="kpi-value">{totalPartners}</div>
			<div class="kpi-sub">Across TN, Kerala & SEA</div>
		</div>

		<div class="kpi-card highlight-green">
			<div class="kpi-header">
				<span class="kpi-icon">✈️</span>
				<span class="kpi-label">Active Bookings</span>
			</div>
			<div class="kpi-value">{totalActiveBookings}</div>
			<div class="kpi-sub">Trips currently in transit / confirmed</div>
		</div>

		<div class="kpi-card highlight-amber">
			<div class="kpi-header">
				<span class="kpi-icon">💳</span>
				<span class="kpi-label">Partner Khata Outstanding</span>
			</div>
			<div class="kpi-value">₹{(totalReceivables / 100000).toFixed(2)}L</div>
			<div class="kpi-sub">Total credit line utilized</div>
		</div>

		<div class="kpi-card highlight-blue">
			<div class="kpi-header">
				<span class="kpi-icon">📈</span>
				<span class="kpi-label">Avg Commission Slab</span>
			</div>
			<div class="kpi-value">10.5%</div>
			<div class="kpi-sub">Performance-tiered payouts</div>
		</div>
	</div>

	<!-- Toolbar & Filters -->
	<div class="filter-card">
		<div class="search-input-wrap">
			<span class="search-icon">🔍</span>
			<input
				type="text"
				placeholder="Search agency by name, owner, city, GST, or tour market..."
				bind:value={searchQuery}
				class="search-input"
			/>
		</div>

		<div class="filter-controls">
			<div class="select-wrap">
				<label for="filter-region">Area / Hub:</label>
				<select id="filter-region" bind:value={filterRegion} class="filter-select">
					<option value="all">All Territories</option>
					<option value="tamilnadu">Tamil Nadu (Chennai, Madurai, Kanyakumari)</option>
					<option value="kerala">Kerala (Cochin, Calicut)</option>
					<option value="international">Malaysia & Overseas DMCs</option>
				</select>
			</div>

			<div class="select-wrap">
				<label for="filter-type">Partner Type:</label>
				<select id="filter-type" bind:value={filterType} class="filter-select">
					<option value="all">All Types</option>
					<option value="Retail Travel Agency">Retail Travel Agency</option>
					<option value="Corporate Travel Desk">Corporate Travel Desk</option>
					<option value="B2B Sub-Agent">B2B Sub-Agent</option>
					<option value="Inbound DMC Partner">Inbound DMC Partner</option>
				</select>
			</div>
		</div>
	</div>

	<!-- Agents Grid -->
	<div class="agents-grid">
		{#if filteredAgents.length === 0}
			<div class="empty-state">
				<span class="empty-icon">🤝</span>
				<h3>No tour agents found matching filters</h3>
				<p>Try clearing your search query or switching territory.</p>
				<button class="action-btn secondary" onclick={() => { searchQuery = ''; filterRegion = 'all'; filterType = 'all'; }}>
					Reset Filters
				</button>
			</div>
		{:else}
			{#each filteredAgents as agent (agent.id)}
				<div class="agent-card">
					<!-- Top Card Header -->
					<div class="card-top">
						<div class="agency-avatar">
							<span class="avatar-sym">🏢</span>
						</div>
						<div class="agency-details">
							<div class="name-status-row">
								<h2 class="agency-name">{agent.agencyName}</h2>
								<span class="status-badge {agent.status}">{agent.status}</span>
							</div>
							<div class="contact-person-row">
								<span class="person-name">👤 {agent.contactPerson}</span>
								<span class="person-role">({agent.designation})</span>
							</div>
						</div>
					</div>

					<!-- Location and Partner Type -->
					<div class="location-banner">
						<span class="loc-text">📍 <strong>{agent.city}</strong>, {agent.state}</span>
						<span class="type-pill">{agent.agentType}</span>
					</div>

					<!-- Commission & Financial Khata Matrix -->
					<div class="khata-box">
						<div class="khata-col">
							<span class="khata-label">Commission Slab</span>
							<span class="khata-val text-green">{agent.commissionRatePercent}%</span>
						</div>
						<div class="khata-col">
							<span class="khata-label">Credit Limit</span>
							<span class="khata-val">₹{(agent.creditLimit / 1000).toFixed(0)}k</span>
						</div>
						<div class="khata-col">
							<span class="khata-label">Outstanding Khata</span>
							<span class="khata-val text-amber">₹{agent.currentReceivable.toLocaleString()}</span>
						</div>
						<div class="khata-col">
							<span class="khata-label">Live Trips</span>
							<span class="khata-val text-blue">{agent.activeBookings} active</span>
						</div>
					</div>

					<!-- Primary Tour Markets -->
					<div class="markets-section">
						<span class="market-label">🎯 Focus Markets:</span>
						<div class="market-pills">
							{#each agent.primaryMarkets as market}
								<span class="market-pill">{market}</span>
							{/each}
						</div>
					</div>

					<!-- Compliance & Tax ID -->
					<div class="compliance-row">
						<span class="comp-id">GST/IATA: <code>{agent.iataOrGst}</code></span>
						<span class="comp-active">Active: {agent.lastActive}</span>
					</div>

					<!-- Action Buttons -->
					<div class="card-footer">
						<a
							href={getWhatsAppUrl(agent.whatsapp, agent.contactPerson, agent.agencyName)}
							target="_blank"
							rel="noreferrer"
							class="action-btn-sm whatsapp"
						>
							<span>💬</span>
							<span>WhatsApp</span>
						</a>

						<a href="tel:{agent.phone}" class="action-btn-sm call">
							<span>📞</span>
							<span>Call</span>
						</a>

						<button class="action-btn-sm khata" onclick={() => (selectedAgentForKhata = agent)}>
							<span>📑</span>
							<span>Khata Ledger</span>
						</button>

						<a href="/travel/book" class="action-btn-sm book">
							<span>⚡</span>
							<span>Book Tour</span>
						</a>
					</div>
				</div>
			{/each}
		{/if}
	</div>
</div>

<!-- Modal: Onboard Tour Agent -->
{#if showAddAgentModal}
	<div class="modal-backdrop">
		<div class="modal-box">
			<div class="modal-header">
				<h2>🤝 Onboard Tour Agent / B2B Partner</h2>
				<button class="close-btn" onclick={() => (showAddAgentModal = false)}>✕</button>
			</div>

			<div class="modal-body">
				<div class="form-row">
					<div class="form-group flex-2">
						<label for="agent-name">Agency / Company Name *</label>
						<input id="agent-name" type="text" placeholder="e.g. Coimbatore Royal Tours" bind:value={newAgent.agencyName} class="form-input" />
					</div>
					<div class="form-group flex-1">
						<label for="agent-type">Partner Category *</label>
						<select id="agent-type" bind:value={newAgent.agentType} class="form-input">
							<option value="Retail Travel Agency">Retail Travel Agency</option>
							<option value="Corporate Travel Desk">Corporate Travel Desk</option>
							<option value="B2B Sub-Agent">B2B Sub-Agent</option>
							<option value="Inbound DMC Partner">Inbound DMC Partner</option>
						</select>
					</div>
				</div>

				<div class="form-row">
					<div class="form-group flex-1">
						<label for="agent-person">Key Contact Person *</label>
						<input id="agent-person" type="text" placeholder="e.g. K. Senthil Kumar" bind:value={newAgent.contactPerson} class="form-input" />
					</div>
					<div class="form-group flex-1">
						<label for="agent-desig">Designation</label>
						<input id="agent-desig" type="text" placeholder="e.g. Managing Director" bind:value={newAgent.designation} class="form-input" />
					</div>
					<div class="form-group flex-1">
						<label for="agent-phone">Mobile / Phone *</label>
						<input id="agent-phone" type="text" placeholder="+91 94433 11223" bind:value={newAgent.phone} class="form-input" />
					</div>
				</div>

				<div class="form-row">
					<div class="form-group flex-1">
						<label for="agent-wa">WhatsApp Direct</label>
						<input id="agent-wa" type="text" placeholder="+91 94433 11223" bind:value={newAgent.whatsapp} class="form-input" />
					</div>
					<div class="form-group flex-1">
						<label for="agent-email">Email Address</label>
						<input id="agent-email" type="email" placeholder="b2b@cbetours.in" bind:value={newAgent.email} class="form-input" />
					</div>
					<div class="form-group flex-1">
						<label for="agent-city">Base City *</label>
						<input id="agent-city" type="text" placeholder="e.g. Coimbatore" bind:value={newAgent.city} class="form-input" />
					</div>
				</div>

				<div class="form-row">
					<div class="form-group flex-1">
						<label for="agent-region">Hub / Territory</label>
						<select id="agent-region" bind:value={newAgent.regionCode} class="form-input">
							<option value="tamilnadu">Tamil Nadu</option>
							<option value="kerala">Kerala</option>
							<option value="northindia">North India</option>
							<option value="international">Malaysia & Overseas</option>
						</select>
					</div>
					<div class="form-group flex-1">
						<label for="agent-comm">Commission Slab (%)</label>
						<input id="agent-comm" type="number" bind:value={newAgent.commissionRatePercent} class="form-input" />
					</div>
					<div class="form-group flex-1">
						<label for="agent-credit">Credit Limit (₹)</label>
						<input id="agent-credit" type="number" bind:value={newAgent.creditLimit} class="form-input" />
					</div>
				</div>

				<div class="form-row">
					<div class="form-group flex-1">
						<label for="agent-gst">GSTIN / IATA Reg Number</label>
						<input id="agent-gst" type="text" placeholder="33AABCC1234F1Z5" bind:value={newAgent.iataOrGst} class="form-input" />
					</div>
					<div class="form-group flex-2">
						<label for="agent-markets">Tour Markets (comma separated)</label>
						<input id="agent-markets" type="text" placeholder="Kashmir 4D/5N, Malaysia Tours, Kerala Houseboat" bind:value={newAgent.primaryMarketsInput} class="form-input" />
					</div>
				</div>
			</div>

			<div class="modal-footer">
				<button class="action-btn secondary" onclick={() => (showAddAgentModal = false)}>Cancel</button>
				<button class="action-btn primary" onclick={handleCreateAgent}>Onboard Partner</button>
			</div>
		</div>
	</div>
{/if}

<!-- Drawer: Agent Khata Ledger -->
{#if selectedAgentForKhata}
	<div class="modal-backdrop">
		<div class="modal-box modal-sm">
			<div class="modal-header">
				<h2>📑 Khata Ledger: {selectedAgentForKhata.agencyName}</h2>
				<button class="close-btn" onclick={() => (selectedAgentForKhata = null)}>✕</button>
			</div>

			<div class="modal-body">
				<div class="khata-banner">
					<div class="khata-metric">
						<span class="km-label">Credit Limit</span>
						<span class="km-val">₹{selectedAgentForKhata.creditLimit.toLocaleString()}</span>
					</div>
					<div class="khata-metric">
						<span class="km-label">Current Outstanding</span>
						<span class="km-val text-amber">₹{selectedAgentForKhata.currentReceivable.toLocaleString()}</span>
					</div>
					<div class="khata-metric">
						<span class="km-label">Available Credit</span>
						<span class="km-val text-green">₹{(selectedAgentForKhata.creditLimit - selectedAgentForKhata.currentReceivable).toLocaleString()}</span>
					</div>
				</div>

				<h3 class="section-title">Recent Trips & Khata Transactions</h3>
				<div class="ledger-list">
					<div class="ledger-item">
						<div class="li-top">
							<span class="li-title">Booking #TRIP-8842 - Kashmir Family (4 Pax)</span>
							<span class="li-amount text-red">+₹1,40,000</span>
						</div>
						<div class="li-meta">Agent Margin: 10% (₹14,000) • Net Payable: ₹1,26,000</div>
					</div>

					<div class="ledger-item">
						<div class="li-top">
							<span class="li-title">Bank NEFT Credit / Payment Received</span>
							<span class="li-amount text-green">-₹1,00,000</span>
						</div>
						<div class="li-meta">HDFC Bank Ref: #NEFT9920148812 • Verified</div>
					</div>

					<div class="ledger-item">
						<div class="li-top">
							<span class="li-title">Booking #TRIP-7719 - Malaysia 5D/4N Package</span>
							<span class="li-amount text-red">+₹75,000</span>
						</div>
						<div class="li-meta">Commission Paid to Agent: ₹7,500</div>
					</div>
				</div>
			</div>

			<div class="modal-footer">
				<button class="action-btn secondary" onclick={() => (selectedAgentForKhata = null)}>Close Ledger</button>
				<button
					class="action-btn primary"
					onclick={() => {
						toast.success(`Khata Statement PDF dispatched to ${selectedAgentForKhata?.email || 'registered agent email'}`);
						selectedAgentForKhata = null;
					}}
				>
					Send Statement via WhatsApp
				</button>
			</div>
		</div>
	</div>
{/if}

<style>
	.agents-root {
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
		background: rgba(16, 185, 129, 0.15);
		border: 1px solid rgba(16, 185, 129, 0.3);
		color: #34d399;
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
		background: #10b981;
		color: #ffffff;
	}
	.action-btn.primary:hover {
		background: #059669;
	}
	.action-btn.secondary {
		background: #1e293b;
		color: #e2e8f0;
		border: 1px solid #334155;
	}
	.action-btn.secondary:hover {
		background: #334155;
	}

	/* KPI Grid */
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

	/* Filter Card */
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

	/* Agents Grid */
	.agents-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(360px, 1fr));
		gap: 1.25rem;
	}

	.agent-card {
		background: #0f172a;
		border: 1px solid #1e293b;
		border-radius: 0.875rem;
		padding: 1.25rem;
		display: flex;
		flex-direction: column;
		gap: 0.875rem;
		transition: all 0.2s ease;
	}
	.agent-card:hover {
		border-color: #10b981;
		box-shadow: 0 4px 20px rgba(16, 185, 129, 0.12);
	}

	.card-top {
		display: flex;
		gap: 0.875rem;
		align-items: flex-start;
	}

	.agency-avatar {
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

	.agency-details {
		flex: 1;
		min-width: 0;
	}

	.name-status-row {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 0.5rem;
	}
	.agency-name {
		font-size: 1.05rem;
		font-weight: 700;
		color: #f8fafc;
		margin: 0;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.status-badge {
		font-size: 0.7rem;
		font-weight: 700;
		padding: 0.2rem 0.5rem;
		border-radius: 9999px;
		text-transform: uppercase;
	}
	.status-badge.active {
		background: rgba(34, 197, 94, 0.15);
		color: #4ade80;
		border: 1px solid rgba(34, 197, 94, 0.3);
	}

	.contact-person-row {
		display: flex;
		gap: 0.35rem;
		font-size: 0.8125rem;
		margin-top: 0.2rem;
	}
	.person-name {
		color: #e2e8f0;
		font-weight: 500;
	}
	.person-role {
		color: #64748b;
	}

	.location-banner {
		background: #1e293b;
		border-left: 3px solid #10b981;
		padding: 0.45rem 0.65rem;
		border-radius: 0.375rem;
		font-size: 0.8125rem;
		color: #e2e8f0;
		display: flex;
		justify-content: space-between;
		align-items: center;
	}

	.type-pill {
		background: rgba(14, 165, 233, 0.15);
		color: #38bdf8;
		font-size: 0.7rem;
		padding: 0.15rem 0.45rem;
		border-radius: 0.25rem;
		font-weight: 600;
	}

	.khata-box {
		display: grid;
		grid-template-columns: 1fr 1fr 1.2fr 1fr;
		gap: 0.5rem;
		background: #090d16;
		border: 1px solid #1e293b;
		padding: 0.6rem 0.75rem;
		border-radius: 0.5rem;
	}

	.khata-col {
		display: flex;
		flex-direction: column;
	}
	.khata-label {
		font-size: 0.6875rem;
		color: #64748b;
	}
	.khata-val {
		font-size: 0.875rem;
		font-weight: 700;
		color: #f8fafc;
	}
	.text-green {
		color: #4ade80;
	}
	.text-amber {
		color: #fbbf24;
	}
	.text-blue {
		color: #38bdf8;
	}
	.text-red {
		color: #f87171;
	}

	.markets-section {
		display: flex;
		flex-direction: column;
		gap: 0.35rem;
	}
	.market-label {
		font-size: 0.75rem;
		color: #94a3b8;
		font-weight: 600;
	}
	.market-pills {
		display: flex;
		flex-wrap: wrap;
		gap: 0.35rem;
	}
	.market-pill {
		background: #1e293b;
		color: #cbd5e1;
		font-size: 0.75rem;
		padding: 0.15rem 0.5rem;
		border-radius: 0.25rem;
	}

	.compliance-row {
		display: flex;
		justify-content: space-between;
		align-items: center;
		font-size: 0.75rem;
		color: #64748b;
	}
	.comp-id code {
		color: #cbd5e1;
		font-family: monospace;
	}

	.card-footer {
		display: flex;
		gap: 0.4rem;
		margin-top: auto;
		padding-top: 0.5rem;
		border-top: 1px solid #1e293b;
	}

	.action-btn-sm {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 0.3rem;
		padding: 0.45rem 0.6rem;
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
	.action-btn-sm.call {
		background: #1e293b;
		color: #e2e8f0;
		border-color: #334155;
	}
	.action-btn-sm.khata {
		background: #1e293b;
		color: #fbbf24;
		border-color: #334155;
	}
	.action-btn-sm.book {
		background: #0284c7;
		color: #ffffff;
	}

	/* Empty State */
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

	/* Modals & Ledgers */
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
		max-width: 520px;
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

	.khata-banner {
		display: grid;
		grid-template-columns: 1fr 1fr 1fr;
		gap: 0.75rem;
		background: #1e293b;
		border: 1px solid #334155;
		padding: 0.75rem 1rem;
		border-radius: 0.5rem;
	}
	.khata-metric {
		display: flex;
		flex-direction: column;
	}
	.km-label {
		font-size: 0.6875rem;
		color: #94a3b8;
	}
	.km-val {
		font-size: 1rem;
		font-weight: 700;
		color: #f8fafc;
	}

	.section-title {
		font-size: 0.875rem;
		color: #e2e8f0;
		margin: 0.5rem 0 0.25rem 0;
	}

	.ledger-list {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}
	.ledger-item {
		background: #1e293b;
		border: 1px solid #334155;
		padding: 0.6rem 0.75rem;
		border-radius: 0.375rem;
		display: flex;
		flex-direction: column;
		gap: 0.2rem;
	}
	.li-top {
		display: flex;
		justify-content: space-between;
		font-size: 0.8125rem;
		font-weight: 600;
		color: #f8fafc;
	}
	.li-meta {
		font-size: 0.75rem;
		color: #94a3b8;
	}
</style>
