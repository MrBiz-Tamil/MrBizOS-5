<script lang="ts">
	import { onMount } from 'svelte';
	import { realestateGrpc } from '$lib/grpc';

	interface Property {
		id: string;
		tenantId: string;
		title: string;
		propertyType: string;
		location: string;
		price: string;
		areaSqft: string;
		bedrooms?: number;
		status: string;
		amenitiesJson?: string;
	}

	interface SiteVisit {
		id: string;
		tenantId: string;
		propertyId: string;
		clientName: string;
		clientPhone: string;
		scheduledTime: string;
		status: string;
		driverName?: string;
		cabRegNumber?: string;
		pickupLocation?: string;
	}

	interface PropertyDeal {
		id: string;
		tenantId: string;
		propertyId: string;
		buyerName: string;
		sellerName: string;
		dealAmount: string;
		commissionAmount: string;
		status: string;
	}

	interface PlotUnit {
		plotNo: number;
		surveyNo: string;
		facing: 'East' | 'North' | 'West' | 'South';
		areaSqft: number;
		cents: number;
		ratePerSqft: number;
		status: 'AVAILABLE' | 'TOKEN_ADVANCE' | 'AGREEMENT_DONE' | 'DEED_REGISTERED';
		buyerName?: string;
		tokenAdvancePaid?: number;
	}

	let activeTab = $state<'matrix' | 'converter' | 'receipt' | 'visits' | 'portfolio'>('matrix');
	let properties = $state<Property[]>([]);
	let siteVisits = $state<SiteVisit[]>([]);
	let deals = $state<PropertyDeal[]>([]);
	let loading = $state(true);
	let error = $state('');
	let statusMsg = $state('');

	// 24-Plot Layout Matrix
	let plots = $state<PlotUnit[]>([
		{ plotNo: 1, surveyNo: '142/2A', facing: 'East', areaSqft: 2400, cents: 5.51, ratePerSqft: 2200, status: 'AVAILABLE' },
		{ plotNo: 2, surveyNo: '142/2A', facing: 'East', areaSqft: 2400, cents: 5.51, ratePerSqft: 2200, status: 'TOKEN_ADVANCE', buyerName: 'K. Senthil Kumar', tokenAdvancePaid: 200000 },
		{ plotNo: 3, surveyNo: '142/2B', facing: 'North', areaSqft: 1800, cents: 4.13, ratePerSqft: 2350, status: 'AVAILABLE' },
		{ plotNo: 4, surveyNo: '142/2B', facing: 'North', areaSqft: 1800, cents: 4.13, ratePerSqft: 2350, status: 'AGREEMENT_DONE', buyerName: 'M. Anand & Deepa', tokenAdvancePaid: 500000 },
		{ plotNo: 5, surveyNo: '142/2C', facing: 'East', areaSqft: 2400, cents: 5.51, ratePerSqft: 2200, status: 'DEED_REGISTERED', buyerName: 'G. Balasubramanian', tokenAdvancePaid: 5280000 },
		{ plotNo: 6, surveyNo: '142/2C', facing: 'East', areaSqft: 2400, cents: 5.51, ratePerSqft: 2200, status: 'AVAILABLE' },
		{ plotNo: 7, surveyNo: '143/1', facing: 'North', areaSqft: 3200, cents: 7.35, ratePerSqft: 2100, status: 'AVAILABLE' },
		{ plotNo: 8, surveyNo: '143/1', facing: 'North', areaSqft: 3200, cents: 7.35, ratePerSqft: 2100, status: 'TOKEN_ADVANCE', buyerName: 'Dr. V. Rajesh', tokenAdvancePaid: 250000 },
		{ plotNo: 9, surveyNo: '143/2', facing: 'South', areaSqft: 1500, cents: 3.44, ratePerSqft: 2150, status: 'AVAILABLE' },
		{ plotNo: 10, surveyNo: '143/2', facing: 'South', areaSqft: 1500, cents: 3.44, ratePerSqft: 2150, status: 'AVAILABLE' },
		{ plotNo: 11, surveyNo: '144/1A', facing: 'West', areaSqft: 2400, cents: 5.51, ratePerSqft: 2100, status: 'AVAILABLE' },
		{ plotNo: 12, surveyNo: '144/1A', facing: 'West', areaSqft: 2400, cents: 5.51, ratePerSqft: 2100, status: 'AVAILABLE' },
		{ plotNo: 13, surveyNo: '144/1B', facing: 'East', areaSqft: 2100, cents: 4.82, ratePerSqft: 2250, status: 'AVAILABLE' },
		{ plotNo: 14, surveyNo: '144/1B', facing: 'East', areaSqft: 2100, cents: 4.82, ratePerSqft: 2250, status: 'TOKEN_ADVANCE', buyerName: 'R. Venkatesan', tokenAdvancePaid: 150000 },
		{ plotNo: 15, surveyNo: '144/2', facing: 'North', areaSqft: 2400, cents: 5.51, ratePerSqft: 2300, status: 'AVAILABLE' },
		{ plotNo: 16, surveyNo: '144/2', facing: 'North', areaSqft: 2400, cents: 5.51, ratePerSqft: 2300, status: 'AVAILABLE' },
		{ plotNo: 17, surveyNo: '145/1', facing: 'East', areaSqft: 4000, cents: 9.18, ratePerSqft: 2000, status: 'AVAILABLE' },
		{ plotNo: 18, surveyNo: '145/1', facing: 'East', areaSqft: 4000, cents: 9.18, ratePerSqft: 2000, status: 'AGREEMENT_DONE', buyerName: 'Sun Promoters B2B', tokenAdvancePaid: 1200000 },
		{ plotNo: 19, surveyNo: '145/2', facing: 'North', areaSqft: 1800, cents: 4.13, ratePerSqft: 2350, status: 'AVAILABLE' },
		{ plotNo: 20, surveyNo: '145/2', facing: 'North', areaSqft: 1800, cents: 4.13, ratePerSqft: 2350, status: 'AVAILABLE' },
		{ plotNo: 21, surveyNo: '146/1A', facing: 'South', areaSqft: 2400, cents: 5.51, ratePerSqft: 2100, status: 'AVAILABLE' },
		{ plotNo: 22, surveyNo: '146/1A', facing: 'South', areaSqft: 2400, cents: 5.51, ratePerSqft: 2100, status: 'AVAILABLE' },
		{ plotNo: 23, surveyNo: '146/1B', facing: 'West', areaSqft: 2200, cents: 5.05, ratePerSqft: 2150, status: 'TOKEN_ADVANCE', buyerName: 'N. Murugesan', tokenAdvancePaid: 200000 },
		{ plotNo: 24, surveyNo: '146/1B', facing: 'West', areaSqft: 2200, cents: 5.05, ratePerSqft: 2150, status: 'AVAILABLE' }
	]);

	let selectedPlot = $state<PlotUnit | null>(plots[0]);

	// Indian Land Area Converter State (Anchor: Square Feet)
	let convSqft = $state(2400);
	let convCents = $derived((convSqft / 435.6).toFixed(3));
	let convGrounds = $derived((convSqft / 2400).toFixed(3));
	let convGunthas = $derived((convSqft / 1089).toFixed(3));
	let convSqYards = $derived((convSqft / 9).toFixed(2));
	let convAcres = $derived((convSqft / 43560).toFixed(4));
	let convHectares = $derived((convSqft / 107639).toFixed(4));

	// Token Advance & Booking Receipt Generator State
	let reraRegNo = $state('TN/RERA/LAYOUT/2026/0942');
	let ventureName = $state('Greenfield Smart Meadows - Phase II');
	let receiptNo = $state('ADV-2026-084');
	let receiptDate = $state(new Date().toISOString().split('T')[0]);
	let buyerFullName = $state('K. Senthil Kumar');
	let buyerMobile = $state('+91 98401 23456');
	let buyerAadhaarPan = $state('AAAPK1234F / 5849 1029 3847');
	let receiptPlotNo = $state(2);
	let receiptSurveyNo = $state('142/2A');
	let receiptAreaSqft = $state(2400);
	let receiptRateSqft = $state(2200);
	let receiptTokenPaid = $state(200000);
	let paymentMode = $state<'UPI' | 'NEFT_RTGS' | 'CHEQUE' | 'CASH'>('UPI');
	let txnRef = $state('UPI/RR/20261021/984210');
	let agreementDaysDeadline = $state(30);

	let receiptTotalConsideration = $derived(receiptAreaSqft * receiptRateSqft);
	let receiptBalancePayable = $derived(receiptTotalConsideration - receiptTokenPaid);

	// Schedule Site Visit Form Modal
	let showSiteVisitModal = $state(false);
	let visitClientName = $state('');
	let visitClientPhone = $state('');
	let visitPickupAddress = $state('Anna Nagar West, Chennai');
	let visitDriverName = $state('Murugan (Innova Crysta - TN 01 BK 4821)');
	let visitDateTime = $state('');

	async function loadRealEstateData() {
		loading = true;
		error = '';
		try {
			const [propRes, visitRes, dealRes]: [any, any, any] = await Promise.all([
				realestateGrpc.listProperties(),
				realestateGrpc.listSiteVisits(),
				realestateGrpc.listDeals()
			]);
			properties = propRes?.properties || [];
			siteVisits = visitRes?.siteVisits || [];
			deals = dealRes?.deals || [];
		} catch (err: any) {
			error = err.message || 'Failed to load Real Estate records from gRPC service';
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		loadRealEstateData();
	});

	function handlePlotClick(plot: PlotUnit) {
		selectedPlot = plot;
		receiptPlotNo = plot.plotNo;
		receiptSurveyNo = plot.surveyNo;
		receiptAreaSqft = plot.areaSqft;
		receiptRateSqft = plot.ratePerSqft;
		if (plot.buyerName) {
			buyerFullName = plot.buyerName;
			receiptTokenPaid = plot.tokenAdvancePaid || 100000;
		}
	}

	function updatePlotStatus(status: 'AVAILABLE' | 'TOKEN_ADVANCE' | 'AGREEMENT_DONE' | 'DEED_REGISTERED') {
		if (!selectedPlot) return;
		selectedPlot.status = status;
		if (status === 'TOKEN_ADVANCE' && !selectedPlot.buyerName) {
			selectedPlot.buyerName = buyerFullName;
			selectedPlot.tokenAdvancePaid = receiptTokenPaid;
		}
		plots = [...plots];
		statusMsg = `✅ Plot #${selectedPlot.plotNo} marked as ${status.replace('_', ' ')}`;
		setTimeout(() => statusMsg = '', 3000);
	}

	async function handleScheduleVisit() {
		if (!visitClientName || !visitClientPhone) {
			alert('Please enter Client Name and Mobile number');
			return;
		}
		try {
			const res: any = await realestateGrpc.scheduleSiteVisit({
				propertyId: selectedPlot ? `plot-${selectedPlot.plotNo}` : 'layout-greenfield',
				clientName: visitClientName,
				clientPhone: visitClientPhone,
				scheduledTime: visitDateTime || new Date(Date.now() + 86400000).toISOString()
			});
			if (res?.siteVisit) {
				siteVisits = [{ ...res.siteVisit, driverName: visitDriverName, pickupLocation: visitPickupAddress }, ...siteVisits];
				showSiteVisitModal = false;
				visitClientName = '';
				visitClientPhone = '';
				statusMsg = `✅ Site visit scheduled for ${res.siteVisit.clientName} with cab pickup!`;
				setTimeout(() => statusMsg = '', 4000);
			}
		} catch (err: any) {
			alert('Failed to schedule visit: ' + err.message);
		}
	}

	function printAdvanceReceipt() {
		window.print();
	}
</script>

<svelte:head>
	<title>Real Estate & Land Layout Operations Hub | MrBiz-OS</title>
</svelte:head>

<div class="realestate-app">
	<!-- Top Bar -->
	<header class="app-header">
		<div>
			<div class="badge-row">
				<span class="pill-badge">🏢 Real Estate & Land Layout OS</span>
				<span class="status-indicator">● RERA Compliant · ConnectRPC v1</span>
			</div>
			<h1 class="main-title">{ventureName}</h1>
			<p class="subtitle">Interactive 24-plot layout matrix, Indian land unit converter, official token booking receipts, and cab site visit tracking.</p>
		</div>

		<div class="header-actions">
			<button class="btn btn-secondary" onclick={() => showSiteVisitModal = true}>
				🚗 Schedule Site Visit Cab
			</button>
			<button class="btn btn-primary" onclick={() => activeTab = 'receipt'}>
				📄 Generate Token Receipt
			</button>
		</div>
	</header>

	{#if statusMsg}
		<div class="status-alert">{statusMsg}</div>
	{/if}

	{#if error}
		<div class="error-alert">⚠️ {error}</div>
	{/if}

	<!-- Venture KPI Dashboard -->
	<div class="kpi-grid">
		<div class="kpi-card">
			<div class="kpi-label">Total Layout Extent</div>
			<div class="kpi-value text-white">24 Units</div>
			<div class="kpi-sub">Total 3.84 Acres Land</div>
		</div>
		<div class="kpi-card">
			<div class="kpi-label">Available Ready Plots</div>
			<div class="kpi-value text-emerald-400">
				{plots.filter(p => p.status === 'AVAILABLE').length} Available
			</div>
			<div class="kpi-sub">Clear Title &amp; DTCP / CMDA</div>
		</div>
		<div class="kpi-card">
			<div class="kpi-label">Token Advance Received</div>
			<div class="kpi-value text-amber-400">
				{plots.filter(p => p.status === 'TOKEN_ADVANCE').length} Plots
			</div>
			<div class="kpi-sub">Under 30-Day Agreement Window</div>
		</div>
		<div class="kpi-card">
			<div class="kpi-label">Sub-Registrar Deeds Done</div>
			<div class="kpi-value text-sky-400">
				{plots.filter(p => p.status === 'DEED_REGISTERED' || p.status === 'AGREEMENT_DONE').length} Registered
			</div>
			<div class="kpi-sub">100% Consideration Cleared</div>
		</div>
	</div>

	<!-- Navigation Tabs -->
	<div class="nav-tabs">
		<button 
			class="tab-btn" 
			class:active={activeTab === 'matrix'} 
			onclick={() => activeTab = 'matrix'}>
			🗺️ 24-Plot Visual Layout Matrix
		</button>
		<button 
			class="tab-btn" 
			class:active={activeTab === 'converter'} 
			onclick={() => activeTab = 'converter'}>
			📐 Indian Land Area Converter (Cent / Ground / Sqft)
		</button>
		<button 
			class="tab-btn" 
			class:active={activeTab === 'receipt'} 
			onclick={() => activeTab = 'receipt'}>
			🧾 Official Token Advance Booking Receipt
		</button>
		<button 
			class="tab-btn" 
			class:active={activeTab === 'visits'} 
			onclick={() => activeTab = 'visits'}>
			🚗 Site Visit Cab &amp; Visitor Log ({siteVisits.length})
		</button>
		<button 
			class="tab-btn" 
			class:active={activeTab === 'portfolio'} 
			onclick={() => activeTab = 'portfolio'}>
			🏢 Commercial &amp; Villa Inventory ({properties.length})
		</button>
	</div>

	<!-- TAB 1: 24-PLOT VISUAL LAYOUT MATRIX -->
	{#if activeTab === 'matrix'}
		<div class="matrix-layout">
			<!-- Visual Grid Board -->
			<div class="matrix-board">
				<div class="board-header">
					<div>
						<h2 class="section-title">Layout Master Grid (Venture Map)</h2>
						<p class="section-desc">Click any plot to view survey details, dimension, facing, or update booking status.</p>
					</div>
					<div class="legend-bar">
						<span class="legend-item"><span class="dot available"></span> Available</span>
						<span class="legend-item"><span class="dot token"></span> Token Paid</span>
						<span class="legend-item"><span class="dot agreement"></span> Agreement Done</span>
						<span class="legend-item"><span class="dot registered"></span> Registered Deed</span>
					</div>
				</div>

				<div class="plots-grid">
					{#each plots as plot}
						<button 
							type="button"
							class="plot-tile status-{plot.status.toLowerCase()}" 
							class:selected={selectedPlot?.plotNo === plot.plotNo}
							onclick={() => handlePlotClick(plot)}>
							<div class="plot-number">Plot #{plot.plotNo}</div>
							<div class="plot-extent">{plot.areaSqft} sqft ({plot.cents} Cents)</div>
							<div class="plot-facing">{plot.facing} Facing</div>
							<div class="plot-status-text">{plot.status.replace('_', ' ')}</div>
						</button>
					{/each}
				</div>
			</div>

			<!-- Plot Details & Control Drawer -->
			{#if selectedPlot}
				<div class="plot-drawer">
					<div class="drawer-header">
						<div>
							<div class="text-xs text-sky-400 font-bold uppercase">Plot Inspection Card</div>
							<h3 class="text-xl font-extrabold text-white">Plot #{selectedPlot.plotNo}</h3>
						</div>
						<span class="status-badge status-{selectedPlot.status.toLowerCase()}">
							{selectedPlot.status.replace('_', ' ')}
						</span>
					</div>

					<div class="drawer-specs">
						<div class="spec-row">
							<span>Survey / Khasra No:</span>
							<strong class="text-white">{selectedPlot.surveyNo}</strong>
						</div>
						<div class="spec-row">
							<span>Total Extent (Sq. Ft):</span>
							<strong class="text-sky-400">{selectedPlot.areaSqft} sq. ft</strong>
						</div>
						<div class="spec-row">
							<span>Cent Equivalent:</span>
							<strong class="text-amber-400">{selectedPlot.cents} Cents</strong>
						</div>
						<div class="spec-row">
							<span>Ground Equivalent:</span>
							<strong class="text-emerald-400">{(selectedPlot.areaSqft / 2400).toFixed(2)} Ground</strong>
						</div>
						<div class="spec-row">
							<span>Facing (Vastu Direction):</span>
							<strong class="text-white">{selectedPlot.facing} Facing</strong>
						</div>
						<div class="spec-row">
							<span>Agreed Rate:</span>
							<strong class="text-white">₹{selectedPlot.ratePerSqft} / sqft</strong>
						</div>
						<div class="spec-row total-row">
							<span>Total Consideration:</span>
							<strong class="text-2xl text-emerald-400">₹{(selectedPlot.areaSqft * selectedPlot.ratePerSqft).toLocaleString('en-IN')}</strong>
						</div>
					</div>

					{#if selectedPlot.buyerName}
						<div class="buyer-info-box">
							<div class="text-xs text-amber-400 font-bold">👤 Current Booking Details</div>
							<div class="text-sm font-bold text-white mt-1">{selectedPlot.buyerName}</div>
							<div class="text-xs text-slate-300">Advance Paid: ₹{(selectedPlot.tokenAdvancePaid || 0).toLocaleString('en-IN')}</div>
						</div>
					{/if}

					<!-- Status Transition Actions -->
					<div class="drawer-actions">
						<div class="text-xs font-bold text-slate-400 mb-2">Change Plot Stage:</div>
						<div class="action-btn-group">
							<button class="btn btn-sm btn-outline-success" onclick={() => updatePlotStatus('AVAILABLE')}>
								Mark Available
							</button>
							<button class="btn btn-sm btn-outline-amber" onclick={() => updatePlotStatus('TOKEN_ADVANCE')}>
								Token Advance
							</button>
							<button class="btn btn-sm btn-outline-sky" onclick={() => updatePlotStatus('AGREEMENT_DONE')}>
								Agreement Done
							</button>
							<button class="btn btn-sm btn-outline-purple" onclick={() => updatePlotStatus('DEED_REGISTERED')}>
								Deed Registered
							</button>
						</div>

						<button class="btn btn-primary w-full mt-4" onclick={() => {
							receiptPlotNo = selectedPlot!.plotNo;
							receiptSurveyNo = selectedPlot!.surveyNo;
							receiptAreaSqft = selectedPlot!.areaSqft;
							receiptRateSqft = selectedPlot!.ratePerSqft;
							activeTab = 'receipt';
						}}>
							🧾 Print Token Receipt for Plot #{selectedPlot.plotNo}
						</button>
					</div>
				</div>
			{/if}
		</div>
	{/if}

	<!-- TAB 2: INDIAN LAND AREA CONVERTER -->
	{#if activeTab === 'converter'}
		<div class="section-container">
			<div class="section-header">
				<div>
					<h2 class="section-title">Indian Multi-Regional Land Measurement Converter</h2>
					<p class="section-desc">Instantly convert land measurements across South, North, and Central Indian real estate standards.</p>
				</div>
			</div>

			<div class="converter-box">
				<div class="converter-input-hero">
					<label class="hero-label" for="conv-sqft">Enter Base Area in Square Feet (Sq. Ft):</label>
					<div class="hero-input-wrapper">
						<input id="conv-sqft" type="number" bind:value={convSqft} class="hero-input font-mono" />
						<span class="unit-tag">Sq. Ft</span>
					</div>
				</div>

				<div class="converted-grid">
					<div class="converted-tile">
						<div class="tile-label">Cents (Tamil Nadu / Kerala / AP)</div>
						<div class="tile-val text-amber-400">{convCents} <span class="unit">Cents</span></div>
						<div class="tile-sub">1 Cent = 435.6 Sq. Ft</div>
					</div>

					<div class="converted-tile">
						<div class="tile-label">Grounds (Chennai / Urban Real Estate)</div>
						<div class="tile-val text-emerald-400">{convGrounds} <span class="unit">Grounds</span></div>
						<div class="tile-sub">1 Ground = 2,400 Sq. Ft</div>
					</div>

					<div class="converted-tile">
						<div class="tile-label">Gunthas (Telangana / Maharashtra / Karnataka)</div>
						<div class="tile-val text-sky-400">{convGunthas} <span class="unit">Gunthas</span></div>
						<div class="tile-sub">1 Guntha = 1,089 Sq. Ft (33ft × 33ft)</div>
					</div>

					<div class="converted-tile">
						<div class="tile-label">Square Yards / Gaj (North India / Hyderabad)</div>
						<div class="tile-val text-purple-400">{convSqYards} <span class="unit">Sq. Yards (Gaj)</span></div>
						<div class="tile-sub">1 Sq. Yard = 9 Sq. Ft</div>
					</div>

					<div class="converted-tile">
						<div class="tile-label">Acres (Standard Agriculture / Farm)</div>
						<div class="tile-val text-rose-400">{convAcres} <span class="unit">Acres</span></div>
						<div class="tile-sub">1 Acre = 43,560 Sq. Ft = 100 Cents</div>
					</div>

					<div class="converted-tile">
						<div class="tile-label">Hectares (Metric Standard)</div>
						<div class="tile-val text-teal-400">{convHectares} <span class="unit">Hectares</span></div>
						<div class="tile-sub">1 Hectare = 2.471 Acres = 107,639 Sq. Ft</div>
					</div>
				</div>
			</div>
		</div>
	{/if}

	<!-- TAB 3: OFFICIAL TOKEN ADVANCE RECEIPT GENERATOR -->
	{#if activeTab === 'receipt'}
		<div class="section-container">
			<div class="section-header no-print">
				<div>
					<h2 class="section-title">Official Plot Token &amp; Advance Booking Slip</h2>
					<p class="section-desc">Issue binding token receipts with promoter legal declaration, balance schedule, and seal.</p>
				</div>
				<button class="btn btn-primary" onclick={printAdvanceReceipt}>🖨️ Print Official Receipt</button>
			</div>

			<div class="receipt-builder-grid">
				<!-- Left Form Controls (Hidden on Print) -->
				<div class="receipt-form no-print">
					<h3 class="form-section-title">Booking Details</h3>
					<div class="space-y-3">
						<div>
							<label class="input-label" for="buyer-name">Buyer Full Name</label>
							<input id="buyer-name" class="text-input" type="text" bind:value={buyerFullName} />
						</div>
						<div class="grid grid-cols-2 gap-2">
							<div>
								<label class="input-label" for="buyer-mobile">Mobile Number</label>
								<input id="buyer-mobile" class="text-input" type="tel" bind:value={buyerMobile} />
							</div>
							<div>
								<label class="input-label" for="buyer-pan">PAN / Aadhaar</label>
								<input id="buyer-pan" class="text-input font-mono" type="text" bind:value={buyerAadhaarPan} />
							</div>
						</div>

						<div class="grid grid-cols-2 gap-2">
							<div>
								<label class="input-label" for="rec-plot">Plot Number</label>
								<input id="rec-plot" class="text-input" type="number" bind:value={receiptPlotNo} />
							</div>
							<div>
								<label class="input-label" for="rec-survey">Survey Number</label>
								<input id="rec-survey" class="text-input" type="text" bind:value={receiptSurveyNo} />
							</div>
						</div>

						<div class="grid grid-cols-2 gap-2">
							<div>
								<label class="input-label" for="rec-area">Plot Area (Sq. Ft)</label>
								<input id="rec-area" class="text-input" type="number" bind:value={receiptAreaSqft} />
							</div>
							<div>
								<label class="input-label" for="rec-rate">Rate per Sq. Ft (₹)</label>
								<input id="rec-rate" class="text-input" type="number" bind:value={receiptRateSqft} />
							</div>
						</div>

						<div class="grid grid-cols-2 gap-2">
							<div>
								<label class="input-label" for="rec-advance">Token Advance (₹)</label>
								<input id="rec-advance" class="text-input" type="number" bind:value={receiptTokenPaid} />
							</div>
							<div>
								<label class="input-label" for="rec-mode">Payment Mode</label>
								<select id="rec-mode" class="text-input" bind:value={paymentMode}>
									<option value="UPI">UPI Transfer</option>
									<option value="NEFT_RTGS">NEFT / RTGS</option>
									<option value="CHEQUE">Cheque / Demand Draft</option>
									<option value="CASH">Cash Deposit</option>
								</select>
							</div>
						</div>

						<div>
							<label class="input-label" for="rec-txn">Transaction / Cheque / UTR Ref #</label>
							<input id="rec-txn" class="text-input font-mono" type="text" bind:value={txnRef} />
						</div>

						<div>
							<label class="input-label" for="rec-deadline">Agreement Execution Window (Days)</label>
							<input id="rec-deadline" class="text-input" type="number" bind:value={agreementDaysDeadline} />
						</div>
					</div>
				</div>

				<!-- Right Printable Advance Receipt -->
				<div class="receipt-paper printable-slip">
					<!-- Slip Header -->
					<div class="slip-header">
						<div class="company-brand">
							<h2>MRBIZ DEVELOPERS &amp; PROMOTERS LLP</h2>
							<p>RERA Registration No: <strong>{reraRegNo}</strong></p>
							<p>Corporate Office: 4th Floor, Skyline Tower, Mount Road, Chennai - 600002</p>
						</div>
						<div class="receipt-meta">
							<div class="doc-badge">TOKEN ADVANCE RECEIPT</div>
							<div class="text-xs mt-1">Receipt #: <strong>{receiptNo}</strong></div>
							<div class="text-xs">Date: <strong>{receiptDate}</strong></div>
						</div>
					</div>

					<div class="slip-divider"></div>

					<!-- Receipt Body -->
					<div class="slip-body">
						<p class="declaration-text">
							Received with thanks from <strong>Mr./Ms. {buyerFullName}</strong> (Mobile: {buyerMobile}, PAN/Aadhaar: {buyerAadhaarPan}) a sum of <strong>₹{receiptTokenPaid.toLocaleString('en-IN')}</strong> ({paymentMode} Ref: {txnRef}) towards non-refundable token booking advance for the following layout plot:
						</p>

						<table class="slip-table">
							<tbody>
								<tr>
									<th>Project / Venture Name</th>
									<td>{ventureName}</td>
									<th>RERA Approved</th>
									<td>Yes (DTCP Approved)</td>
								</tr>
								<tr>
									<th>Plot Number</th>
									<td class="font-bold text-base">Plot #{receiptPlotNo}</td>
									<th>Survey / Khasra No</th>
									<td>{receiptSurveyNo}</td>
								</tr>
								<tr>
									<th>Extent / Plot Area</th>
									<td>{receiptAreaSqft} Sq. Ft ({(receiptAreaSqft / 435.6).toFixed(2)} Cents)</td>
									<th>Agreed Rate / Sq. Ft</th>
									<td>₹{receiptRateSqft.toLocaleString('en-IN')} / sqft</td>
								</tr>
								<tr class="highlight-row">
									<th>Total Consideration</th>
									<td class="font-bold">₹{receiptTotalConsideration.toLocaleString('en-IN')}</td>
									<th>Balance Due at Agreement</th>
									<td class="font-bold text-rose-600">₹{receiptBalancePayable.toLocaleString('en-IN')}</td>
								</tr>
							</tbody>
						</table>

						<!-- Construction / Sale Milestone Terms -->
						<div class="terms-box">
							<div class="font-bold mb-1">Standard Payment Milestone Schedule:</div>
							<ol class="milestone-list">
								<li>1. Token Advance Booking: <strong>₹{receiptTokenPaid.toLocaleString('en-IN')}</strong> (Received today)</li>
								<li>2. Sale Agreement within {agreementDaysDeadline} days: <strong>20%</strong> of Total Consideration</li>
								<li>3. Layout Infrastructure &amp; Compound Plinth: <strong>30%</strong></li>
								<li>4. Final Registration at Sub-Registrar Office &amp; Patta Handover: <strong>Remaining Balance</strong></li>
							</ol>
						</div>

						<div class="legal-disclaimer">
							* Note: This receipt acknowledges receipt of booking advance. Balance payment must be remitted within {agreementDaysDeadline} days from receipt date, failing which the booking stands cancelled subject to promoter terms. Subject to Chennai jurisdiction only.
						</div>
					</div>

					<!-- Signatures -->
					<div class="slip-signatures">
						<div class="sig-block">
							<div class="sig-line"></div>
							<div>Signature of Purchaser / Buyer</div>
						</div>
						<div class="sig-block">
							<div class="sig-seal">MRBIZ PROMOTERS SEAL</div>
							<div class="sig-line"></div>
							<div>Authorized Signatory (Promoter)</div>
						</div>
					</div>
				</div>
			</div>
		</div>
	{/if}

	<!-- TAB 4: SITE VISIT CAB & VISITOR LOG -->
	{#if activeTab === 'visits'}
		<div class="section-container">
			<div class="section-header">
				<div>
					<h2 class="section-title">Site Visit Cab &amp; Visitor Logistics</h2>
					<p class="section-desc">Assign customer pickup cabs, track prospective buyers, and log site feedback.</p>
				</div>
				<button class="btn btn-primary" onclick={() => showSiteVisitModal = true}>
					🚗 + Schedule Customer Cab Visit
				</button>
			</div>

			<div class="table-card">
				<table class="data-table">
					<thead>
						<tr>
							<th>Customer Details</th>
							<th>Pickup Location</th>
							<th>Cab &amp; Driver Assigned</th>
							<th>Scheduled Time</th>
							<th>Status</th>
							<th>Outcome</th>
						</tr>
					</thead>
					<tbody>
						{#each siteVisits as visit}
							<tr>
								<td>
									<div class="font-bold text-white">{visit.clientName}</div>
									<div class="text-xs text-slate-400">{visit.clientPhone}</div>
								</td>
								<td>{visit.pickupLocation || 'Anna Nagar West / City Center'}</td>
								<td>
									<span class="mono-badge">{visit.driverName || 'Cab: TN 01 BK 4821 (Murugan)'}</span>
								</td>
								<td>{new Date(visit.scheduledTime).toLocaleString()}</td>
								<td>
									<span class="status-badge status-token_advance">{visit.status}</span>
								</td>
								<td>
									<button class="btn-sm btn-outline-amber" onclick={() => {
										buyerFullName = visit.clientName;
										buyerMobile = visit.clientPhone;
										activeTab = 'receipt';
									}}>
										Collect Token
									</button>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</div>
	{/if}

	<!-- TAB 5: COMMERCIAL & VILLA INVENTORY -->
	{#if activeTab === 'portfolio'}
		<div class="section-container">
			<div class="section-header">
				<div>
					<h2 class="section-title">Active Real Estate Inventory</h2>
					<p class="section-desc">Commercial office spaces, luxury villas, and multi-tenant assets.</p>
				</div>
			</div>

			<div class="portfolio-grid">
				{#each properties as prop}
					<div class="prop-card">
						<div class="prop-type-badge">{prop.propertyType}</div>
						<h3 class="prop-title">{prop.title}</h3>
						<div class="prop-location">📍 {prop.location}</div>
						<div class="prop-specs">
							<span>📐 {prop.areaSqft} Sq. Ft</span>
							{#if prop.bedrooms}
								<span>🛏️ {prop.bedrooms} BHK</span>
							{/if}
						</div>
						<div class="prop-footer">
							<div>
								<div class="text-xs text-slate-400">Total Valuation</div>
								<div class="text-lg font-bold text-emerald-400">₹{Number(prop.price).toLocaleString('en-IN')}</div>
							</div>
							<span class="badge status-available">{prop.status}</span>
						</div>
					</div>
				{/each}
			</div>
		</div>
	{/if}

	<!-- MODAL: SCHEDULE SITE VISIT CAB -->
	{#if showSiteVisitModal}
		<div class="modal-backdrop">
			<div class="modal-container max-w-lg">
				<div class="modal-header">
					<h3 class="text-lg font-bold text-white">Schedule Customer Site Visit Cab</h3>
					<button class="btn btn-secondary" onclick={() => showSiteVisitModal = false}>✕</button>
				</div>
				<div class="modal-body space-y-4">
					<div>
						<label class="input-label" for="v-name">Customer Full Name</label>
						<input id="v-name" class="text-input" type="text" bind:value={visitClientName} placeholder="e.g. Ramesh Kannan" />
					</div>
					<div>
						<label class="input-label" for="v-phone">Customer Mobile Number</label>
						<input id="v-phone" class="text-input" type="tel" bind:value={visitClientPhone} placeholder="+91 98400 12345" />
					</div>
					<div>
						<label class="input-label" for="v-pickup">Customer Pickup Address</label>
						<input id="v-pickup" class="text-input" type="text" bind:value={visitPickupAddress} placeholder="e.g. Near Shanthi Colony, Anna Nagar" />
					</div>
					<div>
						<label class="input-label" for="v-driver">Driver &amp; Vehicle Allocation</label>
						<select id="v-driver" class="text-input" bind:value={visitDriverName}>
							<option value="Murugan (Innova Crysta - TN 01 BK 4821)">Murugan (Innova Crysta - TN 01 BK 4821)</option>
							<option value="Karthik (Maruti Ertiga - TN 09 AZ 8192)">Karthik (Maruti Ertiga - TN 09 AZ 8192)</option>
							<option value="Saravanan (Toyota Fortuner - TN 07 CL 9900)">Saravanan (Toyota Fortuner - TN 07 CL 9900)</option>
						</select>
					</div>
					<div>
						<label class="input-label" for="v-date">Visit Date &amp; Time</label>
						<input id="v-date" class="text-input" type="datetime-local" bind:value={visitDateTime} />
					</div>
				</div>
				<div class="modal-footer">
					<button class="btn btn-secondary" onclick={() => showSiteVisitModal = false}>Cancel</button>
					<button class="btn btn-primary" onclick={handleScheduleVisit}>Confirm &amp; Dispatch Cab</button>
				</div>
			</div>
		</div>
	{/if}
</div>

<style>
	.realestate-app {
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
		background: rgba(217, 119, 6, 0.15);
		color: #fbbf24;
		border: 1px solid rgba(217, 119, 6, 0.3);
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
		border: none;
		transition: all 0.15s ease;
	}

	.btn-sm {
		font-size: 0.75rem;
		padding: 0.35rem 0.65rem;
		border-radius: 6px;
	}

	.btn-primary { background: #d97706; color: #ffffff; }
	.btn-primary:hover { background: #b45309; }
	.btn-secondary { background: #334155; color: #f1f5f9; }
	.btn-secondary:hover { background: #475569; }

	.btn-outline-success { background: transparent; border: 1px solid #10b981; color: #34d399; }
	.btn-outline-amber { background: transparent; border: 1px solid #f59e0b; color: #fbbf24; }
	.btn-outline-sky { background: transparent; border: 1px solid #0284c7; color: #38bdf8; }
	.btn-outline-purple { background: transparent; border: 1px solid #a855f7; color: #c084fc; }

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
		color: #fbbf24;
		border-bottom-color: #fbbf24;
	}

	.tab-btn:hover:not(.active) {
		color: #cbd5e1;
	}

	/* MATRIX LAYOUT */
	.matrix-layout {
		display: grid;
		grid-template-columns: 1fr 360px;
		gap: 1.5rem;
	}

	@media (max-width: 1024px) {
		.matrix-layout {
			grid-template-columns: 1fr;
		}
	}

	.matrix-board {
		background: #1e293b;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 14px;
		padding: 1.5rem;
	}

	.board-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		flex-wrap: wrap;
		gap: 1rem;
		margin-bottom: 1.25rem;
	}

	.legend-bar {
		display: flex;
		gap: 1rem;
		font-size: 0.75rem;
		color: #cbd5e1;
	}

	.legend-item {
		display: flex;
		align-items: center;
		gap: 0.35rem;
	}

	.dot {
		width: 10px;
		height: 10px;
		border-radius: 50%;
	}

	.dot.available { background: #10b981; }
	.dot.token { background: #f59e0b; }
	.dot.agreement { background: #0284c7; }
	.dot.registered { background: #a855f7; }

	.plots-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(130px, 1fr));
		gap: 0.75rem;
	}

	.plot-tile {
		background: #0f172a;
		border: 2px solid rgba(255, 255, 255, 0.08);
		border-radius: 8px;
		padding: 0.75rem 0.5rem;
		text-align: center;
		cursor: pointer;
		transition: all 0.15s ease;
		display: flex;
		flex-direction: column;
		justify-content: space-between;
		min-height: 110px;
		color: inherit;
		font-family: inherit;
		width: 100%;
	}

	.plot-tile:hover {
		transform: translateY(-2px);
		box-shadow: 0 4px 12px rgba(0,0,0,0.4);
	}

	.plot-tile.selected {
		border-color: #fbbf24 !important;
		outline: 2px solid rgba(251, 191, 36, 0.4);
	}

	.plot-tile.status-available { border-color: rgba(16, 185, 129, 0.4); background: rgba(16, 185, 129, 0.05); }
	.plot-tile.status-token_advance { border-color: rgba(245, 158, 11, 0.5); background: rgba(245, 158, 11, 0.08); }
	.plot-tile.status-agreement_done { border-color: rgba(2, 132, 199, 0.5); background: rgba(2, 132, 199, 0.08); }
	.plot-tile.status-deed_registered { border-color: rgba(168, 85, 247, 0.5); background: rgba(168, 85, 247, 0.08); }

	.plot-number {
		font-size: 0.875rem;
		font-weight: 800;
		color: #ffffff;
	}

	.plot-extent {
		font-size: 0.6875rem;
		color: #94a3b8;
		margin: 0.25rem 0;
	}

	.plot-facing {
		font-size: 0.6875rem;
		color: #38bdf8;
		font-weight: 600;
	}

	.plot-status-text {
		font-size: 0.625rem;
		font-weight: 700;
		text-transform: uppercase;
		margin-top: 0.25rem;
		padding: 0.15rem;
		border-radius: 4px;
	}

	.status-available .plot-status-text { color: #34d399; }
	.status-token_advance .plot-status-text { color: #fbbf24; }
	.status-agreement_done .plot-status-text { color: #38bdf8; }
	.status-deed_registered .plot-status-text { color: #c084fc; }

	/* PLOT DRAWER */
	.plot-drawer {
		background: #1e293b;
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: 14px;
		padding: 1.5rem;
		display: flex;
		flex-direction: column;
		gap: 1.25rem;
	}

	.drawer-header {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
		padding-bottom: 0.75rem;
	}

	.drawer-specs {
		display: flex;
		flex-direction: column;
		gap: 0.6rem;
		font-size: 0.8125rem;
	}

	.spec-row {
		display: flex;
		justify-content: space-between;
		color: #94a3b8;
	}

	.total-row {
		border-top: 1px solid rgba(255, 255, 255, 0.08);
		padding-top: 0.75rem;
		align-items: center;
	}

	.buyer-info-box {
		background: rgba(245, 158, 11, 0.1);
		border: 1px solid rgba(245, 158, 11, 0.25);
		padding: 0.75rem;
		border-radius: 8px;
	}

	.action-btn-group {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 0.5rem;
	}

	/* CONVERTER */
	.converter-box {
		max-width: 900px;
		margin: 0 auto;
	}

	.converter-input-hero {
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: 12px;
		padding: 1.5rem;
		margin-bottom: 1.5rem;
		text-align: center;
	}

	.hero-label {
		font-size: 0.875rem;
		font-weight: 600;
		color: #94a3b8;
		margin-bottom: 0.5rem;
		display: block;
	}

	.hero-input-wrapper {
		display: inline-flex;
		align-items: center;
		background: #1e293b;
		border: 2px solid #fbbf24;
		border-radius: 10px;
		padding: 0.5rem 1rem;
	}

	.hero-input {
		background: transparent;
		border: none;
		color: #ffffff;
		font-size: 2rem;
		font-weight: 800;
		text-align: center;
		outline: none;
		width: 240px;
	}

	.unit-tag {
		font-size: 1rem;
		font-weight: 700;
		color: #fbbf24;
	}

	.converted-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
		gap: 1rem;
	}

	.converted-tile {
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 10px;
		padding: 1.25rem;
	}

	.tile-label {
		font-size: 0.75rem;
		color: #94a3b8;
		font-weight: 600;
		margin-bottom: 0.35rem;
	}

	.tile-val {
		font-size: 1.625rem;
		font-weight: 800;
	}

	.tile-val .unit {
		font-size: 0.8125rem;
		font-weight: 500;
		color: #cbd5e1;
	}

	.tile-sub {
		font-size: 0.6875rem;
		color: #64748b;
		margin-top: 0.25rem;
	}

	/* RECEIPT BUILDER */
	.receipt-builder-grid {
		display: grid;
		grid-template-columns: 380px 1fr;
		gap: 1.5rem;
	}

	@media (max-width: 1024px) {
		.receipt-builder-grid {
			grid-template-columns: 1fr;
		}
	}

	.receipt-form {
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 12px;
		padding: 1.25rem;
	}

	.form-section-title {
		font-size: 1rem;
		font-weight: 700;
		color: #ffffff;
		margin-bottom: 1rem;
	}

	.input-label {
		display: block;
		font-size: 0.75rem;
		color: #94a3b8;
		margin-bottom: 0.25rem;
	}

	.text-input {
		width: 100%;
		padding: 0.5rem 0.75rem;
		background: #1e293b;
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: 6px;
		color: #ffffff;
		font-size: 0.8125rem;
	}

	/* PRINTABLE SLIP */
	.receipt-paper {
		background: #ffffff;
		color: #0f172a;
		border: 2px solid #cbd5e1;
		border-radius: 8px;
		padding: 2rem;
		box-shadow: 0 10px 25px rgba(0,0,0,0.3);
	}

	.slip-header {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
	}

	.company-brand h2 {
		font-size: 1.25rem;
		font-weight: 900;
		color: #0f172a;
		letter-spacing: -0.01em;
	}

	.company-brand p {
		font-size: 0.75rem;
		color: #475569;
		margin-top: 0.15rem;
	}

	.doc-badge {
		background: #d97706;
		color: #ffffff;
		font-weight: 800;
		font-size: 0.75rem;
		padding: 0.25rem 0.6rem;
		border-radius: 4px;
		text-align: right;
	}

	.slip-divider {
		height: 2px;
		background: #0f172a;
		margin: 1rem 0;
	}

	.declaration-text {
		font-size: 0.875rem;
		line-height: 1.6;
		color: #1e293b;
		margin-bottom: 1.25rem;
	}

	.slip-table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.8125rem;
		margin-bottom: 1.25rem;
	}

	.slip-table th, .slip-table td {
		border: 1px solid #94a3b8;
		padding: 0.6rem 0.75rem;
		text-align: left;
	}

	.slip-table th {
		background: #f1f5f9;
		color: #334155;
		width: 25%;
	}

	.highlight-row {
		background: #fffbeb;
	}

	.terms-box {
		background: #f8fafc;
		border: 1px dashed #94a3b8;
		padding: 0.75rem;
		border-radius: 6px;
		font-size: 0.75rem;
		color: #334155;
		margin-bottom: 1rem;
	}

	.milestone-list {
		margin-left: 1rem;
		line-height: 1.5;
	}

	.legal-disclaimer {
		font-size: 0.6875rem;
		color: #64748b;
		line-height: 1.4;
		margin-bottom: 2rem;
	}

	.slip-signatures {
		display: flex;
		justify-content: space-between;
		margin-top: 2rem;
	}

	.sig-block {
		text-align: center;
		font-size: 0.75rem;
		font-weight: 600;
		color: #334155;
		width: 220px;
	}

	.sig-line {
		border-bottom: 1px solid #0f172a;
		margin-bottom: 0.5rem;
		height: 40px;
	}

	.sig-seal {
		font-size: 0.625rem;
		color: #94a3b8;
		border: 1px dashed #cbd5e1;
		padding: 0.25rem;
		border-radius: 4px;
		margin-bottom: 0.25rem;
	}

	/* PORTFOLIO GRID */
	.portfolio-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
		gap: 1.25rem;
	}

	.prop-card {
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 12px;
		padding: 1.25rem;
		display: flex;
		flex-direction: column;
		justify-content: space-between;
	}

	.prop-type-badge {
		font-size: 0.6875rem;
		font-weight: 700;
		color: #fbbf24;
		text-transform: uppercase;
		margin-bottom: 0.5rem;
	}

	.prop-title {
		font-size: 1.125rem;
		font-weight: 700;
		color: #ffffff;
		margin-bottom: 0.25rem;
	}

	.prop-location {
		font-size: 0.75rem;
		color: #94a3b8;
		margin-bottom: 1rem;
	}

	.prop-specs {
		display: flex;
		gap: 1rem;
		font-size: 0.8125rem;
		color: #38bdf8;
		margin-bottom: 1rem;
	}

	.prop-footer {
		display: flex;
		justify-content: space-between;
		align-items: flex-end;
		border-top: 1px solid rgba(255, 255, 255, 0.06);
		padding-top: 0.75rem;
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
	}

	.status-badge {
		font-size: 0.6875rem;
		font-weight: 700;
		padding: 0.25rem 0.6rem;
		border-radius: 6px;
		text-transform: uppercase;
	}

	.status-available { background: rgba(16, 185, 129, 0.15); color: #34d399; }
	.status-token_advance { background: rgba(245, 158, 11, 0.15); color: #fbbf24; }
	.status-agreement_done { background: rgba(2, 132, 199, 0.15); color: #38bdf8; }
	.status-deed_registered { background: rgba(168, 85, 247, 0.15); color: #c084fc; }

	/* MODAL */
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
	}

	.modal-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
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

	@media print {
		.no-print, .nav-tabs, .kpi-grid, .app-header {
			display: none !important;
		}
		body {
			background: white !important;
			color: black !important;
		}
		.receipt-paper {
			box-shadow: none !important;
			border: 1px solid #000 !important;
			width: 100% !important;
			margin: 0 auto !important;
		}
	}
</style>
