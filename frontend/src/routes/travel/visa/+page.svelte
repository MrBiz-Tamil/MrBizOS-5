<script lang="ts">
	import { toast } from '$lib/stores/toast.svelte';

	interface VisaTariff {
		id: string;
		country: string;
		flagEmoji: string;
		visaType: string; // e.g. "30-Day Single Entry Tourist eVisa"
		entryType: 'Single Entry' | 'Multiple Entry';
		validityDays: number;
		stayPeriodDays: number;
		processingTime: string; // e.g. "24 to 48 Hours Express"
		embassyFee: number; // in INR
		serviceFee: number; // in INR (Agency Margin)
		totalSellingPrice: number; // in INR
		expressFeeAddon: number;
		requiredDocuments: string[];
		status: 'active' | 'suspended';
	}

	interface VisaApplication {
		id: string;
		applicantName: string;
		passportNumber: string;
		passportExpiry: string;
		country: string;
		visaType: string;
		travelDate: string;
		customerPhone: string;
		customerEmail: string;
		customerCity: string;
		status: 'Docs Received' | 'Under Review' | 'Submitted to Embassy' | 'Approved' | 'Rejected';
		totalPaid: number;
		appliedDate: string;
		evisaDownloadUrl?: string;
		remarks: string;
	}

	// Active Tab: 'pricing' | 'applications'
	let activeTab = $state<'pricing' | 'applications'>('pricing');

	// Filters
	let searchQuery = $state('');
	let filterCountry = $state('all');
	let filterStatus = $state('all');

	// Modals
	let showAddTariffModal = $state(false);
	let showNewApplicationModal = $state(false);
	let selectedTariffForChecklist = $state<VisaTariff | null>(null);

	// Dataset: Visa Tariffs & Price List
	let tariffs = $state<VisaTariff[]>([
		{
			id: 'visa-dxb-30',
			country: 'United Arab Emirates (Dubai)',
			flagEmoji: '🇦🇪',
			visaType: '30-Day Tourist eVisa (Express)',
			entryType: 'Single Entry',
			validityDays: 60,
			stayPeriodDays: 30,
			processingTime: '24 to 48 Hours',
			embassyFee: 5800,
			serviceFee: 1700,
			totalSellingPrice: 7500,
			expressFeeAddon: 1500,
			requiredDocuments: [
				'Passport front & back color scan (min 6 months validity)',
				'White background digital photo (35x45mm, 80% face view)',
				'Confirmed return flight tickets',
				'Hotel booking voucher / host invitation letter'
			],
			status: 'active'
		},
		{
			id: 'visa-dxb-60',
			country: 'United Arab Emirates (Dubai)',
			flagEmoji: '🇦🇪',
			visaType: '60-Day Multiple Entry Tourist Visa',
			entryType: 'Multiple Entry',
			validityDays: 60,
			stayPeriodDays: 60,
			processingTime: '3 to 4 Working Days',
			embassyFee: 11500,
			serviceFee: 2500,
			totalSellingPrice: 14000,
			expressFeeAddon: 2000,
			requiredDocuments: [
				'Passport color scan with min 6 months validity',
				'Digital photo 35x45mm white background',
				'6-month bank statement with min ₹1.5L balance',
				'Return air ticket & accommodation voucher'
			],
			status: 'active'
		},
		{
			id: 'visa-mys-30',
			country: 'Malaysia',
			flagEmoji: '🇲🇾',
			visaType: 'Malaysia MDAC / Tourist eVisa',
			entryType: 'Single Entry',
			validityDays: 90,
			stayPeriodDays: 30,
			processingTime: '48 to 72 Hours',
			embassyFee: 2200,
			serviceFee: 1300,
			totalSellingPrice: 3500,
			expressFeeAddon: 1000,
			requiredDocuments: [
				'Passport front page color scan',
				'Studio photograph with white background (35x50mm)',
				'Confirmed return air tickets',
				'Hotel stay reservation confirmation in Malaysia'
			],
			status: 'active'
		},
		{
			id: 'visa-sg-30',
			country: 'Singapore',
			flagEmoji: '🇸🇬',
			visaType: 'Singapore Electronic Tourist Visa (ETA)',
			entryType: 'Multiple Entry',
			validityDays: 730,
			stayPeriodDays: 30,
			processingTime: '3 to 5 Working Days',
			embassyFee: 2100,
			serviceFee: 1400,
			totalSellingPrice: 3500,
			expressFeeAddon: 1200,
			requiredDocuments: [
				'Passport color scan (min 6 months validity)',
				'Matte finish photo with 80% face coverage (35x45mm)',
				'Form 14A duly signed by applicant',
				'Confirmed return flight ticket & hotel booking'
			],
			status: 'active'
		},
		{
			id: 'visa-sch-90',
			country: 'Schengen (France / Switzerland)',
			flagEmoji: '🇪🇺',
			visaType: 'Short-Stay Tourist Visa (Type C)',
			entryType: 'Multiple Entry',
			validityDays: 90,
			stayPeriodDays: 90,
			processingTime: '12 to 15 Working Days',
			embassyFee: 7800,
			serviceFee: 4700,
			totalSellingPrice: 12500,
			expressFeeAddon: 3000,
			requiredDocuments: [
				'Original Passport with 2 blank pages',
				'VFS appointment biometrics confirmation',
				'Covering letter with detailed day-wise itinerary',
				'6 months bank statement sealed by bank branch',
				'Last 3 years ITR acknowledgement forms',
				'Travel health insurance covering min €30,000'
			],
			status: 'active'
		}
	]);

	// Dataset: Live Applications Tracker
	let applications = $state<VisaApplication[]>([
		{
			id: 'APP-9912',
			applicantName: 'Muthukumar Ramasamy',
			passportNumber: 'Z6549812',
			passportExpiry: '14 May 2031',
			country: 'United Arab Emirates (Dubai)',
			visaType: '30-Day Tourist eVisa (Express)',
			travelDate: '28 Oct 2026',
			customerPhone: '+91 94431 88992',
			customerEmail: 'muthu.ramasamy@gmail.com',
			customerCity: 'Chennai',
			status: 'Approved',
			totalPaid: 7500,
			appliedDate: '21 Oct 2026',
			evisaDownloadUrl: 'https://cdn.mrbiz.os/visas/uae-9912.pdf',
			remarks: 'Approved by GDRFA Dubai. PDF generated & delivered via WhatsApp.'
		},
		{
			id: 'APP-9913',
			applicantName: 'Kavitha Muthukumar',
			passportNumber: 'Z6549813',
			passportExpiry: '14 May 2031',
			country: 'United Arab Emirates (Dubai)',
			visaType: '30-Day Tourist eVisa (Express)',
			travelDate: '28 Oct 2026',
			customerPhone: '+91 94431 88992',
			customerEmail: 'muthu.ramasamy@gmail.com',
			customerCity: 'Chennai',
			status: 'Approved',
			totalPaid: 7500,
			appliedDate: '21 Oct 2026',
			evisaDownloadUrl: 'https://cdn.mrbiz.os/visas/uae-9913.pdf',
			remarks: 'Approved by GDRFA Dubai.'
		},
		{
			id: 'APP-9914',
			applicantName: 'Selvaganapathy Subramanian',
			passportNumber: 'P8842109',
			passportExpiry: '08 Nov 2029',
			country: 'Malaysia',
			visaType: 'Malaysia MDAC / Tourist eVisa',
			travelDate: '02 Nov 2026',
			customerPhone: '+91 98402 77114',
			customerEmail: 'selva.ganapathy@yahoo.com',
			customerCity: 'Madurai',
			status: 'Submitted to Embassy',
			totalPaid: 3500,
			appliedDate: '22 Oct 2026',
			remarks: 'Submitted to Malaysia Immigration portal. Expected approval in 24 hrs.'
		},
		{
			id: 'APP-9915',
			applicantName: 'Dr. Anandhakrishnan Iyer',
			passportNumber: 'R1098231',
			passportExpiry: '22 Aug 2033',
			country: 'Singapore',
			visaType: 'Singapore Electronic Tourist Visa (ETA)',
			travelDate: '10 Nov 2026',
			customerPhone: '+91 98410 44552',
			customerEmail: 'dr.anand@apollohospitals.org',
			customerCity: 'Chennai',
			status: 'Docs Received',
			totalPaid: 3500,
			appliedDate: '23 Oct 2026',
			remarks: 'Form 14A pending signature verification.'
		}
	]);

	// Filtered lists
	let filteredTariffs = $derived(
		tariffs.filter((t) => {
			const matchesQuery =
				searchQuery === '' ||
				t.country.toLowerCase().includes(searchQuery.toLowerCase()) ||
				t.visaType.toLowerCase().includes(searchQuery.toLowerCase());
			const matchesCountry = filterCountry === 'all' || t.country.includes(filterCountry);
			return matchesQuery && matchesCountry;
		})
	);

	let filteredApplications = $derived(
		applications.filter((a) => {
			const matchesQuery =
				searchQuery === '' ||
				a.applicantName.toLowerCase().includes(searchQuery.toLowerCase()) ||
				a.passportNumber.toLowerCase().includes(searchQuery.toLowerCase()) ||
				a.country.toLowerCase().includes(searchQuery.toLowerCase()) ||
				a.id.toLowerCase().includes(searchQuery.toLowerCase());
			const matchesStatus = filterStatus === 'all' || a.status === filterStatus;
			return matchesQuery && matchesStatus;
		})
	);

	// Summary stats
	let totalActiveTariffs = $derived(tariffs.length);
	let liveApplicationsCount = $derived(applications.length);
	let approvedCount = $derived(applications.filter((a) => a.status === 'Approved').length);
	let processingRevenue = $derived(applications.reduce((acc, a) => acc + a.totalPaid, 0));

	// Add Tariff Form
	let newTariff = $state({
		country: '',
		flagEmoji: '🌐',
		visaType: '',
		entryType: 'Single Entry' as VisaTariff['entryType'],
		validityDays: 60,
		stayPeriodDays: 30,
		processingTime: '2 to 3 Working Days',
		embassyFee: 4000,
		serviceFee: 1500,
		expressFeeAddon: 1000,
		requiredDocsInput: 'Passport Scan, Passport Photo, Return Air Tickets, Hotel Voucher'
	});

	let computedTotalSellingPrice = $derived(
		Number(newTariff.embassyFee) + Number(newTariff.serviceFee)
	);

	function handleCreateTariff() {
		if (!newTariff.country || !newTariff.visaType) {
			toast.error('Please enter Country and Visa Type');
			return;
		}

		const tariff: VisaTariff = {
			id: `visa-${Date.now()}`,
			country: newTariff.country,
			flagEmoji: newTariff.flagEmoji || '🛂',
			visaType: newTariff.visaType,
			entryType: newTariff.entryType,
			validityDays: Number(newTariff.validityDays),
			stayPeriodDays: Number(newTariff.stayPeriodDays),
			processingTime: newTariff.processingTime,
			embassyFee: Number(newTariff.embassyFee),
			serviceFee: Number(newTariff.serviceFee),
			totalSellingPrice: computedTotalSellingPrice,
			expressFeeAddon: Number(newTariff.expressFeeAddon),
			requiredDocuments: newTariff.requiredDocsInput.split(',').map((s) => s.trim()).filter(Boolean),
			status: 'active'
		};

		tariffs = [tariff, ...tariffs];
		showAddTariffModal = false;
		toast.success(`Published visa tariff for ${tariff.country} at ₹${computedTotalSellingPrice.toLocaleString()}`);

		// reset
		newTariff = {
			country: '',
			flagEmoji: '🌐',
			visaType: '',
			entryType: 'Single Entry',
			validityDays: 60,
			stayPeriodDays: 30,
			processingTime: '2 to 3 Working Days',
			embassyFee: 4000,
			serviceFee: 1500,
			expressFeeAddon: 1000,
			requiredDocsInput: 'Passport Scan, Passport Photo, Return Air Tickets, Hotel Voucher'
		};
	}

	// New Application Form
	let newApp = $state({
		applicantName: '',
		passportNumber: '',
		passportExpiry: '2030-12-31',
		country: 'United Arab Emirates (Dubai)',
		visaType: '30-Day Tourist eVisa (Express)',
		travelDate: '2026-11-05',
		customerPhone: '',
		customerEmail: '',
		customerCity: 'Chennai',
		totalPaid: 7500,
		remarks: 'Standard application submission'
	});

	function handleCreateApplication() {
		if (!newApp.applicantName || !newApp.passportNumber || !newApp.customerPhone) {
			toast.error('Please enter Applicant Name, Passport Number, and Phone');
			return;
		}

		const app: VisaApplication = {
			id: `APP-${Math.floor(1000 + Math.random() * 9000)}`,
			applicantName: newApp.applicantName,
			passportNumber: newApp.passportNumber.toUpperCase(),
			passportExpiry: newApp.passportExpiry,
			country: newApp.country,
			visaType: newApp.visaType,
			travelDate: newApp.travelDate,
			customerPhone: newApp.customerPhone,
			customerEmail: newApp.customerEmail,
			customerCity: newApp.customerCity,
			status: 'Submitted to Embassy',
			totalPaid: Number(newApp.totalPaid),
			appliedDate: 'Just now',
			remarks: newApp.remarks
		};

		applications = [app, ...applications];
		showNewApplicationModal = false;
		activeTab = 'applications';
		toast.success(`Registered visa file #${app.id} for ${app.applicantName} (${app.country})`);

		newApp = {
			applicantName: '',
			passportNumber: '',
			passportExpiry: '2030-12-31',
			country: 'United Arab Emirates (Dubai)',
			visaType: '30-Day Tourist eVisa (Express)',
			travelDate: '2026-11-05',
			customerPhone: '',
			customerEmail: '',
			customerCity: 'Chennai',
			totalPaid: 7500,
			remarks: 'Standard application submission'
		};
	}

	function getWhatsAppChecklistUrl(tariff: VisaTariff) {
		const text = encodeURIComponent(
			`*MRBIZ OS VISA CHECKLIST: ${tariff.country}*\n\n` +
			`🛂 *Category:* ${tariff.visaType}\n` +
			`💰 *Embassy Fee + Service:* ₹${tariff.totalSellingPrice.toLocaleString()} (Express +₹${tariff.expressFeeAddon})\n` +
			`⏳ *Processing Time:* ${tariff.processingTime}\n\n` +
			`📋 *Required Documents:*\n` +
			tariff.requiredDocuments.map((doc, idx) => `${idx + 1}. ${doc}`).join('\n') +
			`\n\nPlease send clear photos or PDF scans of these documents to initiate submission!`
		);
		return `https://wa.me/?text=${text}`;
	}
</script>

<div class="visa-root">
	<!-- Top Header -->
	<div class="header-card">
		<div class="title-cluster">
			<div class="title-badge">
				<span class="badge-icon">🛂</span>
				<span class="badge-label">Visa Processing & Price Management</span>
			</div>
			<h1 class="main-title">Visa Tariffs, Country Checklists & Filing Desk</h1>
			<p class="subtitle">
				Manage country visa processing prices (Embassy fee + Agency markup), express turnaround times, mandatory document checklists, and track live applicant filing statuses for <strong>Dubai</strong>, <strong>Malaysia</strong>, <strong>Singapore</strong>, and <strong>Schengen</strong>.
			</p>
		</div>

		<div class="header-actions">
			<button class="action-btn secondary" onclick={() => (showAddTariffModal = true)}>
				<span>🏷️</span>
				<span>Add Country Tariff</span>
			</button>
			<button class="action-btn primary" onclick={() => (showNewApplicationModal = true)}>
				<span>+</span>
				<span>New Visa Application</span>
			</button>
		</div>
	</div>

	<!-- KPI Summary Metrics -->
	<div class="kpi-grid">
		<div class="kpi-card">
			<div class="kpi-header">
				<span class="kpi-icon">🌍</span>
				<span class="kpi-label">Active Visa Tariffs</span>
			</div>
			<div class="kpi-value">{totalActiveTariffs}</div>
			<div class="kpi-sub">Countries & Entry Types</div>
		</div>

		<div class="kpi-card highlight-blue">
			<div class="kpi-header">
				<span class="kpi-icon">📋</span>
				<span class="kpi-label">Live Applications</span>
			</div>
			<div class="kpi-value">{liveApplicationsCount}</div>
			<div class="kpi-sub">Active passport filings</div>
		</div>

		<div class="kpi-card highlight-green">
			<div class="kpi-header">
				<span class="kpi-icon">✅</span>
				<span class="kpi-label">eVisas Approved</span>
			</div>
			<div class="kpi-value">{approvedCount}</div>
			<div class="kpi-sub">Delivered to travelers</div>
		</div>

		<div class="kpi-card highlight-amber">
			<div class="kpi-header">
				<span class="kpi-icon">💳</span>
				<span class="kpi-label">Total Visa Billing</span>
			</div>
			<div class="kpi-value">₹{processingRevenue.toLocaleString()}</div>
			<div class="kpi-sub">Collected Embassy & Service fees</div>
		</div>
	</div>

	<!-- Navigation Tabs & Toolbar -->
	<div class="tab-toolbar-card">
		<div class="tab-buttons">
			<button
				class="tab-btn {activeTab === 'pricing' ? 'active' : ''}"
				onclick={() => (activeTab = 'pricing')}
			>
				<span>🏷️</span>
				<span>Country Visa Pricing & Tariffs ({filteredTariffs.length})</span>
			</button>
			<button
				class="tab-btn {activeTab === 'applications' ? 'active' : ''}"
				onclick={() => (activeTab = 'applications')}
			>
				<span>📂</span>
				<span>Live Applications & Passport Tracker ({filteredApplications.length})</span>
			</button>
		</div>

		<div class="search-input-wrap">
			<span class="search-icon">🔍</span>
			<input
				type="text"
				placeholder="Search by country, visa type, applicant name, or passport..."
				bind:value={searchQuery}
				class="search-input"
			/>
		</div>
	</div>

	<!-- TAB 1: Visa Pricing & Tariffs -->
	{#if activeTab === 'pricing'}
		<div class="tariffs-grid">
			{#each filteredTariffs as tariff (tariff.id)}
				<div class="tariff-card">
					<div class="card-top">
						<div class="country-cluster">
							<span class="flag-icon">{tariff.flagEmoji}</span>
							<div>
								<h2 class="country-name">{tariff.country}</h2>
								<div class="visa-type">{tariff.visaType}</div>
							</div>
						</div>
						<span class="entry-tag">{tariff.entryType}</span>
					</div>

					<!-- Pricing Breakdown Box -->
					<div class="pricing-box">
						<div class="price-split">
							<div class="ps-item">
								<span class="ps-label">Embassy Fee</span>
								<span class="ps-val">₹{tariff.embassyFee.toLocaleString()}</span>
							</div>
							<span class="plus-sym">+</span>
							<div class="ps-item">
								<span class="ps-label">Service Charge</span>
								<span class="ps-val text-green">₹{tariff.serviceFee.toLocaleString()}</span>
							</div>
							<span class="equal-sym">=</span>
							<div class="ps-item highlight">
								<span class="ps-label">Total Client Price</span>
								<span class="ps-total">₹{tariff.totalSellingPrice.toLocaleString()}</span>
							</div>
						</div>
						<div class="express-note">
							⚡ Express processing available: +₹{tariff.expressFeeAddon.toLocaleString()}
						</div>
					</div>

					<!-- Timeline & Validity -->
					<div class="validity-row">
						<span>⏳ Turnaround: <strong>{tariff.processingTime}</strong></span>
						<span>📅 Stay: <strong>{tariff.stayPeriodDays} Days</strong></span>
					</div>

					<!-- Required Docs Preview -->
					<div class="docs-section">
						<span class="docs-title">Required Documents ({tariff.requiredDocuments.length}):</span>
						<ul class="docs-list">
							{#each tariff.requiredDocuments.slice(0, 3) as doc}
								<li>• {doc}</li>
							{/each}
						</ul>
					</div>

					<!-- Footer Actions -->
					<div class="card-footer">
						<button class="action-btn-sm checklist" onclick={() => (selectedTariffForChecklist = tariff)}>
							<span>📋</span>
							<span>Full Checklist</span>
						</button>

						<a
							href={getWhatsAppChecklistUrl(tariff)}
							target="_blank"
							rel="noreferrer"
							class="action-btn-sm whatsapp"
						>
							<span>💬</span>
							<span>Send Checklist</span>
						</a>

						<button
							class="action-btn-sm apply"
							onclick={() => {
								newApp.country = tariff.country;
								newApp.visaType = tariff.visaType;
								newApp.totalPaid = tariff.totalSellingPrice;
								showNewApplicationModal = true;
							}}
						>
							<span>+</span>
							<span>Apply File</span>
						</button>
					</div>
				</div>
			{/each}
		</div>
	{/if}

	<!-- TAB 2: Live Applications Tracker -->
	{#if activeTab === 'applications'}
		<div class="table-container">
			<table class="visa-table">
				<thead>
					<tr>
						<th>File Ref</th>
						<th>Applicant & Passport</th>
						<th>Country & Category</th>
						<th>Travel Date</th>
						<th>City & Contact</th>
						<th>Fee Paid</th>
						<th>Filing Status</th>
						<th>Actions</th>
					</tr>
				</thead>
				<tbody>
					{#if filteredApplications.length === 0}
						<tr>
							<td colspan="8" class="empty-td">No visa applications matching your criteria</td>
						</tr>
					{:else}
						{#each filteredApplications as app (app.id)}
							<tr>
								<td>
									<span class="ref-badge">{app.id}</span>
									<span class="applied-sub">{app.appliedDate}</span>
								</td>
								<td>
									<div class="applicant-col">
										<span class="app-name">{app.applicantName}</span>
										<span class="app-pass">Passport: <code>{app.passportNumber}</code></span>
									</div>
								</td>
								<td>
									<div class="country-col">
										<span class="dest-country">{app.country}</span>
										<span class="dest-category">{app.visaType}</span>
									</div>
								</td>
								<td>
									<span class="travel-date">📅 {app.travelDate}</span>
								</td>
								<td>
									<div class="contact-col">
										<span>{app.customerCity}</span>
										<span class="phone-sub">{app.customerPhone}</span>
									</div>
								</td>
								<td>
									<span class="price-val">₹{app.totalPaid.toLocaleString()}</span>
								</td>
								<td>
									<span class="status-pill {app.status.toLowerCase().replace(/\s+/g, '-')}">
										{app.status}
									</span>
								</td>
								<td>
									<div class="table-actions">
										{#if app.status === 'Approved'}
											<button
												class="mini-btn success"
												onclick={() => toast.success(`eVisa PDF for ${app.applicantName} dispatched via WhatsApp!`)}
												title="Download & WhatsApp eVisa"
											>
												📥 eVisa
											</button>
										{:else}
											<button
												class="mini-btn"
												onclick={() => {
													app.status = 'Approved';
													applications = [...applications];
													toast.success(`Marked visa application #${app.id} as Approved!`);
												}}
												title="Approve Visa"
											>
												✓ Approve
											</button>
										{/if}
									</div>
								</td>
							</tr>
						{/each}
					{/if}
				</tbody>
			</table>
		</div>
	{/if}
</div>

<!-- Modal: Add Country Visa Tariff -->
{#if showAddTariffModal}
	<div class="modal-backdrop">
		<div class="modal-box">
			<div class="modal-header">
				<h2>🏷️ Add Country Visa Tariff & Price</h2>
				<button class="close-btn" onclick={() => (showAddTariffModal = false)}>✕</button>
			</div>

			<div class="modal-body">
				<div class="form-row">
					<div class="form-group flex-2">
						<label for="tf-country">Destination Country *</label>
						<input id="tf-country" type="text" placeholder="e.g. Thailand, Vietnam, UK" bind:value={newTariff.country} class="form-input" />
					</div>
					<div class="form-group flex-1">
						<label for="tf-flag">Flag Emoji</label>
						<input id="tf-flag" type="text" placeholder="🇹🇭" bind:value={newTariff.flagEmoji} class="form-input" />
					</div>
				</div>

				<div class="form-row">
					<div class="form-group flex-2">
						<label for="tf-type">Visa Category / Type *</label>
						<input id="tf-type" type="text" placeholder="e.g. 30-Day Single Entry Tourist eVisa" bind:value={newTariff.visaType} class="form-input" />
					</div>
					<div class="form-group flex-1">
						<label for="tf-entry">Entry Type</label>
						<select id="tf-entry" bind:value={newTariff.entryType} class="form-input">
							<option value="Single Entry">Single Entry</option>
							<option value="Multiple Entry">Multiple Entry</option>
						</select>
					</div>
				</div>

				<div class="form-row">
					<div class="form-group flex-1">
						<label for="tf-turnaround">Processing Turnaround</label>
						<input id="tf-turnaround" type="text" placeholder="e.g. 2 to 3 Working Days" bind:value={newTariff.processingTime} class="form-input" />
					</div>
					<div class="form-group flex-1">
						<label for="tf-stay">Stay Period (Days)</label>
						<input id="tf-stay" type="number" bind:value={newTariff.stayPeriodDays} class="form-input" />
					</div>
				</div>

				<!-- Pricing Split -->
				<div class="costing-engine-panel">
					<h3 class="engine-title">💰 Price Split Structure</h3>
					<div class="form-row">
						<div class="form-group flex-1">
							<label for="tf-embassy">Embassy Fee (₹)</label>
							<input id="tf-embassy" type="number" bind:value={newTariff.embassyFee} class="form-input" />
						</div>
						<div class="form-group flex-1">
							<label for="tf-service">Service Fee / Margin (₹)</label>
							<input id="tf-service" type="number" bind:value={newTariff.serviceFee} class="form-input" />
						</div>
						<div class="form-group flex-1">
							<label for="tf-addon">Express Addon (₹)</label>
							<input id="tf-addon" type="number" bind:value={newTariff.expressFeeAddon} class="form-input" />
						</div>
					</div>
					<div class="computed-total">
						Total Selling Price: <strong>₹{computedTotalSellingPrice.toLocaleString()}</strong>
					</div>
				</div>

				<div class="form-group">
					<label for="tf-docs">Required Documents (comma separated)</label>
					<textarea id="tf-docs" rows="3" bind:value={newTariff.requiredDocsInput} class="form-input"></textarea>
				</div>
			</div>

			<div class="modal-footer">
				<button class="action-btn secondary" onclick={() => (showAddTariffModal = false)}>Cancel</button>
				<button class="action-btn primary" onclick={handleCreateTariff}>Publish Tariff</button>
			</div>
		</div>
	</div>
{/if}

<!-- Modal: New Visa Application -->
{#if showNewApplicationModal}
	<div class="modal-backdrop">
		<div class="modal-box">
			<div class="modal-header">
				<h2>🛂 File New Visa Application</h2>
				<button class="close-btn" onclick={() => (showNewApplicationModal = false)}>✕</button>
			</div>

			<div class="modal-body">
				<div class="form-row">
					<div class="form-group flex-2">
						<label for="app-name">Applicant Full Name (As in Passport) *</label>
						<input id="app-name" type="text" placeholder="e.g. Ramesh Kumar" bind:value={newApp.applicantName} class="form-input" />
					</div>
					<div class="form-group flex-1">
						<label for="app-pass">Passport Number *</label>
						<input id="app-pass" type="text" placeholder="e.g. Z5549012" bind:value={newApp.passportNumber} class="form-input" />
					</div>
				</div>

				<div class="form-row">
					<div class="form-group flex-1">
						<label for="app-country">Target Destination *</label>
						<select id="app-country" bind:value={newApp.country} class="form-input">
							{#each tariffs as t}
								<option value={t.country}>{t.flagEmoji} {t.country}</option>
							{/each}
						</select>
					</div>
					<div class="form-group flex-1">
						<label for="app-travel">Target Travel Date</label>
						<input id="app-travel" type="date" bind:value={newApp.travelDate} class="form-input" />
					</div>
				</div>

				<div class="form-row">
					<div class="form-group flex-1">
						<label for="app-phone">Customer Mobile / WhatsApp *</label>
						<input id="app-phone" type="text" placeholder="+91 94431 88992" bind:value={newApp.customerPhone} class="form-input" />
					</div>
					<div class="form-group flex-1">
						<label for="app-city">City / Origin</label>
						<input id="app-city" type="text" placeholder="Chennai / Madurai" bind:value={newApp.customerCity} class="form-input" />
					</div>
					<div class="form-group flex-1">
						<label for="app-fee">Fee Collected (₹)</label>
						<input id="app-fee" type="number" bind:value={newApp.totalPaid} class="form-input" />
					</div>
				</div>
			</div>

			<div class="modal-footer">
				<button class="action-btn secondary" onclick={() => (showNewApplicationModal = false)}>Cancel</button>
				<button class="action-btn primary" onclick={handleCreateApplication}>Create Visa File</button>
			</div>
		</div>
	</div>
{/if}

<!-- Drawer: Full Document Checklist -->
{#if selectedTariffForChecklist}
	<div class="modal-backdrop">
		<div class="modal-box modal-sm">
			<div class="modal-header">
				<h2>📋 {selectedTariffForChecklist.country} Visa Checklist</h2>
				<button class="close-btn" onclick={() => (selectedTariffForChecklist = null)}>✕</button>
			</div>

			<div class="modal-body">
				<div class="checklist-summary">
					<div class="cs-country">{selectedTariffForChecklist.flagEmoji} {selectedTariffForChecklist.country}</div>
					<div class="cs-sub">{selectedTariffForChecklist.visaType} • ₹{selectedTariffForChecklist.totalSellingPrice.toLocaleString()}</div>
				</div>

				<h3 class="checklist-header">Mandatory Documents Required:</h3>
				<div class="checklist-items">
					{#each selectedTariffForChecklist.requiredDocuments as doc, idx}
						<div class="checklist-item">
							<span class="chk-num">{idx + 1}</span>
							<span class="chk-text">{doc}</span>
						</div>
					{/each}
				</div>
			</div>

			<div class="modal-footer">
				<button class="action-btn secondary" onclick={() => (selectedTariffForChecklist = null)}>Close</button>
				<a
					href={getWhatsAppChecklistUrl(selectedTariffForChecklist)}
					target="_blank"
					rel="noreferrer"
					class="action-btn primary"
				>
					<span>💬</span>
					<span>Share via WhatsApp</span>
				</a>
			</div>
		</div>
	</div>
{/if}

<style>
	.visa-root {
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

	.kpi-card.highlight-blue {
		border-color: rgba(14, 165, 233, 0.4);
		background: linear-gradient(180deg, rgba(14, 165, 233, 0.08) 0%, #0f172a 100%);
	}
	.kpi-card.highlight-green {
		border-color: rgba(34, 197, 94, 0.4);
		background: linear-gradient(180deg, rgba(34, 197, 94, 0.08) 0%, #0f172a 100%);
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

	/* Tab toolbar */
	.tab-toolbar-card {
		background: #0f172a;
		border: 1px solid #1e293b;
		border-radius: 0.75rem;
		padding: 0.75rem 1.25rem;
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
	}

	.tab-buttons {
		display: flex;
		gap: 0.5rem;
	}
	.tab-btn {
		background: #1e293b;
		border: 1px solid #334155;
		color: #94a3b8;
		padding: 0.5rem 1rem;
		border-radius: 0.5rem;
		font-size: 0.8125rem;
		font-weight: 600;
		cursor: pointer;
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		transition: all 0.15s ease;
	}
	.tab-btn.active {
		background: #0284c7;
		color: #ffffff;
		border-color: #0284c7;
	}

	.search-input-wrap {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		background: #1e293b;
		border: 1px solid #334155;
		border-radius: 0.5rem;
		padding: 0.45rem 0.75rem;
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
		font-size: 0.8125rem;
		width: 100%;
	}

	/* Tariffs Grid */
	.tariffs-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(380px, 1fr));
		gap: 1.25rem;
	}

	.tariff-card {
		background: #0f172a;
		border: 1px solid #1e293b;
		border-radius: 0.875rem;
		padding: 1.25rem;
		display: flex;
		flex-direction: column;
		gap: 0.875rem;
		transition: all 0.2s ease;
	}
	.tariff-card:hover {
		border-color: #10b981;
		box-shadow: 0 4px 20px rgba(16, 185, 129, 0.12);
	}

	.card-top {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		gap: 0.5rem;
	}

	.country-cluster {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}
	.flag-icon {
		font-size: 2rem;
	}
	.country-name {
		font-size: 1.05rem;
		font-weight: 700;
		color: #f8fafc;
		margin: 0;
	}
	.visa-type {
		font-size: 0.775rem;
		color: #94a3b8;
	}

	.entry-tag {
		background: rgba(14, 165, 233, 0.15);
		color: #38bdf8;
		font-size: 0.7rem;
		font-weight: 700;
		padding: 0.2rem 0.5rem;
		border-radius: 0.25rem;
	}

	.pricing-box {
		background: #090d16;
		border: 1px solid #1e293b;
		border-radius: 0.5rem;
		padding: 0.75rem;
		display: flex;
		flex-direction: column;
		gap: 0.4rem;
	}

	.price-split {
		display: flex;
		align-items: center;
		justify-content: space-between;
		font-size: 0.75rem;
	}
	.ps-item {
		display: flex;
		flex-direction: column;
	}
	.ps-label {
		font-size: 0.6875rem;
		color: #64748b;
	}
	.ps-val {
		font-weight: 600;
		color: #f8fafc;
	}
	.plus-sym, .equal-sym {
		color: #64748b;
		font-weight: 700;
	}
	.ps-total {
		font-size: 1.15rem;
		font-weight: 800;
		color: #34d399;
	}
	.text-green {
		color: #34d399;
	}

	.express-note {
		font-size: 0.7rem;
		color: #fbbf24;
		border-top: 1px dashed #1e293b;
		padding-top: 0.35rem;
	}

	.validity-row {
		display: flex;
		justify-content: space-between;
		font-size: 0.75rem;
		color: #cbd5e1;
		background: #1e293b;
		padding: 0.45rem 0.65rem;
		border-radius: 0.375rem;
	}

	.docs-section {
		font-size: 0.75rem;
	}
	.docs-title {
		color: #94a3b8;
		font-weight: 600;
		display: block;
		margin-bottom: 0.25rem;
	}
	.docs-list {
		list-style: none;
		padding-left: 0;
		margin: 0;
		color: #cbd5e1;
		display: flex;
		flex-direction: column;
		gap: 0.15rem;
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
	.action-btn-sm.checklist {
		background: #1e293b;
		color: #e2e8f0;
		border-color: #334155;
	}
	.action-btn-sm.whatsapp {
		background: #059669;
		color: #ffffff;
	}
	.action-btn-sm.apply {
		background: #0284c7;
		color: #ffffff;
	}

	/* Table for Live Applications */
	.table-container {
		background: #0f172a;
		border: 1px solid #1e293b;
		border-radius: 0.875rem;
		overflow-x: auto;
	}
	.visa-table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.8125rem;
		text-align: left;
	}
	.visa-table th {
		background: #1e293b;
		color: #94a3b8;
		padding: 0.75rem 1rem;
		font-weight: 600;
		border-bottom: 1px solid #334155;
	}
	.visa-table td {
		padding: 0.85rem 1rem;
		border-bottom: 1px solid #1e293b;
		color: #cbd5e1;
	}
	.empty-td {
		text-align: center;
		padding: 2.5rem !important;
		color: #64748b;
	}

	.ref-badge {
		background: #1e293b;
		color: #38bdf8;
		padding: 0.15rem 0.45rem;
		border-radius: 0.25rem;
		font-family: monospace;
		font-weight: 700;
		font-size: 0.75rem;
		display: inline-block;
	}
	.applied-sub {
		display: block;
		font-size: 0.6875rem;
		color: #64748b;
		margin-top: 0.2rem;
	}

	.applicant-col {
		display: flex;
		flex-direction: column;
	}
	.app-name {
		font-weight: 700;
		color: #f8fafc;
	}
	.app-pass {
		font-size: 0.75rem;
		color: #94a3b8;
	}
	.app-pass code {
		color: #cbd5e1;
		font-family: monospace;
	}

	.country-col {
		display: flex;
		flex-direction: column;
	}
	.dest-country {
		font-weight: 600;
		color: #f8fafc;
	}
	.dest-category {
		font-size: 0.7rem;
		color: #94a3b8;
	}

	.status-pill {
		padding: 0.25rem 0.55rem;
		border-radius: 9999px;
		font-size: 0.7rem;
		font-weight: 700;
		text-transform: uppercase;
		display: inline-block;
	}
	.status-pill.approved {
		background: rgba(34, 197, 94, 0.15);
		color: #4ade80;
		border: 1px solid rgba(34, 197, 94, 0.3);
	}
	.status-pill.submitted-to-embassy {
		background: rgba(14, 165, 233, 0.15);
		color: #38bdf8;
		border: 1px solid rgba(14, 165, 233, 0.3);
	}
	.status-pill.docs-received {
		background: rgba(245, 158, 11, 0.15);
		color: #fbbf24;
		border: 1px solid rgba(245, 158, 11, 0.3);
	}

	.table-actions {
		display: flex;
		gap: 0.35rem;
	}
	.mini-btn {
		background: #1e293b;
		border: 1px solid #334155;
		color: #e2e8f0;
		padding: 0.3rem 0.55rem;
		border-radius: 0.25rem;
		font-size: 0.75rem;
		cursor: pointer;
	}
	.mini-btn.success {
		background: #059669;
		color: #ffffff;
		border: none;
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

	.costing-engine-panel {
		background: #090d16;
		border: 1px solid #334155;
		border-radius: 0.5rem;
		padding: 0.85rem;
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}
	.engine-title {
		margin: 0;
		font-size: 0.8125rem;
		color: #34d399;
		font-weight: 700;
	}
	.computed-total {
		font-size: 0.875rem;
		color: #f8fafc;
		text-align: right;
		border-top: 1px solid #1e293b;
		padding-top: 0.4rem;
	}

	.checklist-summary {
		background: #1e293b;
		padding: 0.75rem 1rem;
		border-radius: 0.5rem;
	}
	.cs-country {
		font-size: 1.1rem;
		font-weight: 700;
		color: #f8fafc;
	}
	.cs-sub {
		font-size: 0.75rem;
		color: #94a3b8;
	}
	.checklist-header {
		font-size: 0.875rem;
		color: #e2e8f0;
		margin: 0.5rem 0 0.25rem 0;
	}
	.checklist-items {
		display: flex;
		flex-direction: column;
		gap: 0.4rem;
	}
	.checklist-item {
		display: flex;
		gap: 0.5rem;
		align-items: center;
		background: #1e293b;
		padding: 0.5rem 0.75rem;
		border-radius: 0.375rem;
		font-size: 0.8125rem;
		color: #cbd5e1;
	}
	.chk-num {
		background: #0ea5e9;
		color: #ffffff;
		border-radius: 9999px;
		width: 18px;
		height: 18px;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 0.65rem;
		font-weight: 800;
		flex-shrink: 0;
	}
</style>
