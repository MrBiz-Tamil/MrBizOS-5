<script lang="ts">
	import { onMount } from 'svelte';
	import { healthcareGrpc } from '$lib/grpc';

	interface Doctor {
		id: string;
		tenantId: string;
		name: string;
		specialty: string;
		qualification: string;
		registrationNumber: string;
		experienceYears: number;
		consultationFee: string;
		isAvailable: boolean;
	}

	interface Appointment {
		id: string;
		tenantId: string;
		doctorId: string;
		doctorName: string;
		patientId?: string;
		patientName: string;
		patientPhone?: string;
		scheduledAt: string;
		status: string;
		consultationType: string;
		roomToken?: string;
		notes?: string;
	}

	interface RxMedicineItem {
		id: string;
		name: string;
		dosage: '1-0-1' | '1-1-1' | '1-0-0' | '0-0-1' | 'SOS';
		timing: 'After Food' | 'Before Food' | 'With Food' | 'Bedtime';
		durationDays: number;
		qty: number;
		scheduleH: boolean;
	}

	let activeTab = $state<'rx' | 'pharmacy' | 'opd' | 'appointments'>('rx');
	let doctors = $state<Doctor[]>([]);
	let appointments = $state<Appointment[]>([]);
	let loading = $state(true);
	let error = $state('');
	let statusMsg = $state('');

	// Selected Doctor for Rx
	let selectedDoctor = $state<Doctor>({
		id: 'doc-1',
		tenantId: 'tenant-1',
		name: 'Dr. Priya Sharma',
		specialty: 'Cardiologist & General Medicine',
		qualification: 'MBBS, MD (General Medicine), DM (Cardiology)',
		registrationNumber: 'MCI-48291/2012',
		experienceYears: 12,
		consultationFee: '800.00',
		isAvailable: true
	});

	// Rx Patient Demographics & Vitals
	let rxPatientName = $state('Rajesh Kumar');
	let rxPatientAge = $state(42);
	let rxPatientGender = $state<'Male' | 'Female' | 'Other'>('Male');
	let rxPatientMobile = $state('+91 98401 98765');
	let rxDate = $state(new Date().toISOString().split('T')[0]);
	let rxDiagnosis = $state('Acute Bronchitis & Essential Hypertension (Stage 1)');

	// Vitals
	let vitalsBP = $state('130/85');
	let vitalsPulse = $state(74);
	let vitalsTemp = $state('98.6');
	let vitalsSpO2 = $state(99);
	let vitalsWeight = $state(72);

	// Rx Medication Line Items
	let rxItems = $state<RxMedicineItem[]>([
		{ id: '1', name: 'Tab. Augmentin 625 Duo (Amoxicillin + Clavulanic)', dosage: '1-0-1', timing: 'After Food', durationDays: 5, qty: 10, scheduleH: true },
		{ id: '2', name: 'Tab. Dolo 650 (Paracetamol)', dosage: '1-0-1', timing: 'After Food', durationDays: 3, qty: 6, scheduleH: false },
		{ id: '3', name: 'Tab. Telma 40 (Telmisartan)', dosage: '1-0-0', timing: 'Before Food', durationDays: 30, qty: 30, scheduleH: true },
		{ id: '4', name: 'Syr. Ascoril D Plus (100ml)', dosage: '1-0-1', timing: 'After Food', durationDays: 5, qty: 1, scheduleH: false }
	]);

	let rxAdvice = $state('Steam inhalation twice daily. Drink warm fluids. Review in clinic after 5 days if fever persists.');
	let rxLabTests = $state('Complete Blood Count (CBC), Serum Electrolytes, ECG 12-Lead');

	// Pharmacy Strip Dispenser & Loose Medicine Calculator
	let pharmaStripMRP = $state(180);
	let pharmaTabletsPerStrip = $state(10);
	let pharmaRequiredTabs = $state(7);
	let pharmaScheduleHWarning = $state(true);
	let pharmaExpiryDate = $state('2026-12');

	// Computed Pharmacy Calculations
	let pharmaPerTabPrice = $derived(pharmaStripMRP / pharmaTabletsPerStrip);
	let pharmaPayableAmount = $derived(Math.round(pharmaPerTabPrice * pharmaRequiredTabs));
	let pharmaStripsToCut = $derived((pharmaRequiredTabs / pharmaTabletsPerStrip).toFixed(2));

	// OPD Queue State
	let currentServingToken = $state(14);
	let nextQueueToken = $state(15);
	let opdWaitingList = $state([
		{ token: 14, name: 'Rajesh Kumar', doctor: 'Dr. Priya Sharma', status: 'IN_CABIN' },
		{ token: 15, name: 'S. Meenakshi', doctor: 'Dr. Priya Sharma', status: 'WAITING' },
		{ token: 16, name: 'Arunmozhi Varman', doctor: 'Dr. Arun Verma', status: 'WAITING' },
		{ token: 17, name: 'Fathima Banu', doctor: 'Dr. Priya Sharma', status: 'WAITING' }
	]);

	// Book Consultation Modal
	let showBookModal = $state(false);
	let bookDocId = $state('');
	let bookPatientName = $state('');
	let bookPatientPhone = $state('');
	let bookConsultMode = $state('VIDEO');
	let bookDateTime = $state('');

	async function loadHealthcareData() {
		loading = true;
		error = '';
		try {
			const [docsRes, apptsRes]: [any, any] = await Promise.all([
				healthcareGrpc.listDoctors(),
				healthcareGrpc.listAppointments()
			]);
			doctors = docsRes?.doctors || [];
			appointments = apptsRes?.appointments || [];
			if (doctors.length > 0) {
				selectedDoctor = doctors[0];
				bookDocId = doctors[0].id;
			}
		} catch (err: any) {
			error = err.message || 'Failed to connect to Healthcare gRPC backend';
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		loadHealthcareData();
	});

	function addRxItem() {
		const newId = String(rxItems.length + 1);
		rxItems = [...rxItems, {
			id: newId,
			name: 'Tab. Pan 40 (Pantoprazole)',
			dosage: '1-0-0',
			timing: 'Before Food',
			durationDays: 5,
			qty: 5,
			scheduleH: true
		}];
	}

	function removeRxItem(index: number) {
		rxItems = rxItems.filter((_, i) => i !== index);
	}

	async function saveAndPublishRx() {
		try {
			const res: any = await healthcareGrpc.createPrescription({
				appointmentId: 'apt-walkin-' + Date.now(),
				patientId: 'pat-' + rxPatientMobile.replace(/[^0-9]/g, ''),
				diagnosis: rxDiagnosis,
				items: JSON.stringify(rxItems),
				advice: rxAdvice
			});
			statusMsg = `✅ Prescription Rx #${res.prescription.id} generated and signed by ${selectedDoctor.name}!`;
			setTimeout(() => statusMsg = '', 4000);
		} catch (err: any) {
			alert('Failed to save prescription: ' + err.message);
		}
	}

	function callNextPatient() {
		currentServingToken++;
		nextQueueToken++;
		const first = opdWaitingList.shift();
		if (first) {
			first.status = 'COMPLETED';
		}
		statusMsg = `🔔 Token #${currentServingToken} is now being called into Doctor's Cabin!`;
		setTimeout(() => statusMsg = '', 3000);
	}

	async function handleBookConsultation() {
		if (!bookPatientName) {
			alert('Please enter Patient Name');
			return;
		}
		try {
			const res: any = await healthcareGrpc.bookAppointment({
				doctorId: bookDocId || selectedDoctor.id,
				patientName: bookPatientName,
				patientPhone: bookPatientPhone,
				scheduledAt: bookDateTime || new Date().toISOString(),
				consultationType: bookConsultMode,
				notes: 'Booked via Clinic Management Portal'
			});
			if (res?.appointment) {
				appointments = [res.appointment, ...appointments];
				showBookModal = false;
				bookPatientName = '';
				bookPatientPhone = '';
				statusMsg = `✅ Appointment confirmed for ${res.appointment.patientName}`;
				setTimeout(() => statusMsg = '', 4000);
			}
		} catch (err: any) {
			alert('Failed to book appointment: ' + err.message);
		}
	}

	function printPrescription() {
		window.print();
	}
</script>

<svelte:head>
	<title>Smart Healthcare &amp; Clinical Rx Hub | MrBiz-OS</title>
</svelte:head>

<div class="healthcare-app">
	<!-- Top Bar -->
	<header class="app-header">
		<div>
			<div class="badge-row">
				<span class="pill-badge">🏥 Smart Healthcare &amp; Clinic OS</span>
				<span class="status-indicator">● MCI Compliant · ConnectRPC v1</span>
			</div>
			<h1 class="main-title">Ayya Multi-Specialty Clinic &amp; Pharmacy</h1>
			<p class="subtitle">Doctor electronic prescription pad (Rx), pharmacy blister strip calculator, live OPD token calling, and telemedicine desk.</p>
		</div>

		<div class="header-actions">
			<button class="btn btn-secondary" onclick={() => showBookModal = true}>
				📅 + Book Consultation
			</button>
			<button class="btn btn-primary" onclick={printPrescription}>
				🖨️ Print Clinical Rx Slip
			</button>
		</div>
	</header>

	{#if statusMsg}
		<div class="status-alert">{statusMsg}</div>
	{/if}

	{#if error}
		<div class="error-alert">⚠️ {error}</div>
	{/if}

	<!-- Quick KPI Metrics -->
	<div class="kpi-grid">
		<div class="kpi-card">
			<div class="kpi-label">Now Serving Token</div>
			<div class="kpi-value text-emerald-400">#{currentServingToken}</div>
			<div class="kpi-sub">Cabin 1: {selectedDoctor.name}</div>
		</div>
		<div class="kpi-card">
			<div class="kpi-label">OPD Queue Waiting</div>
			<div class="kpi-value text-sky-400">{opdWaitingList.length} Patients</div>
			<div class="kpi-sub">Avg Wait Time: ~8 Mins</div>
		</div>
		<div class="kpi-card">
			<div class="kpi-label">Prescription Safety</div>
			<div class="kpi-value text-amber-400">Schedule H / H1</div>
			<div class="kpi-sub">Restricted Antibiotic Shield</div>
		</div>
		<div class="kpi-card">
			<div class="kpi-label">Verified Specialists</div>
			<div class="kpi-value text-purple-400">{doctors.length || 2} Doctors</div>
			<div class="kpi-sub">In-Clinic &amp; Telemedicine</div>
		</div>
	</div>

	<!-- Navigation Tabs -->
	<div class="nav-tabs">
		<button 
			class="tab-btn" 
			class:active={activeTab === 'rx'} 
			onclick={() => activeTab = 'rx'}>
			📝 Clinical Rx Prescription Pad
		</button>
		<button 
			class="tab-btn" 
			class:active={activeTab === 'pharmacy'} 
			onclick={() => activeTab = 'pharmacy'}>
			💊 Pharmacy Strip Dispenser &amp; Schedule H Calculator
		</button>
		<button 
			class="tab-btn" 
			class:active={activeTab === 'opd'} 
			onclick={() => activeTab = 'opd'}>
			📢 Live OPD Queue Token Calling
		</button>
		<button 
			class="tab-btn" 
			class:active={activeTab === 'appointments'} 
			onclick={() => activeTab = 'appointments'}>
			🩺 Doctor Consultations &amp; Telemedicine ({appointments.length})
		</button>
	</div>

	<!-- TAB 1: CLINICAL RX PRESCRIPTION PAD -->
	{#if activeTab === 'rx'}
		<div class="rx-workspace">
			<!-- Left Doctor Clinical Controls (Hidden on Print) -->
			<div class="rx-controls no-print">
				<h3 class="section-title text-base mb-3">Doctor &amp; Patient Inputs</h3>

				<div class="form-group mb-3">
					<label class="input-label" for="doc-select">Attending Doctor</label>
					<select id="doc-select" class="text-input" bind:value={selectedDoctor.id} onchange={(e) => {
						const doc = doctors.find(d => d.id === (e.target as HTMLSelectElement).value);
						if (doc) selectedDoctor = doc;
					}}>
						{#each doctors as doc}
							<option value={doc.id}>{doc.name} ({doc.specialty})</option>
						{/each}
					</select>
				</div>

				<div class="grid grid-cols-2 gap-2 mb-3">
					<div>
						<label class="input-label" for="rx-pname">Patient Full Name</label>
						<input id="rx-pname" class="text-input" type="text" bind:value={rxPatientName} />
					</div>
					<div>
						<label class="input-label" for="rx-pmobile">Mobile Number</label>
						<input id="rx-pmobile" class="text-input" type="tel" bind:value={rxPatientMobile} />
					</div>
				</div>

				<div class="grid grid-cols-2 gap-2 mb-3">
					<div>
						<label class="input-label" for="rx-page">Age (Years)</label>
						<input id="rx-page" class="text-input" type="number" bind:value={rxPatientAge} />
					</div>
					<div>
						<label class="input-label" for="rx-pgender">Gender</label>
						<select id="rx-pgender" class="text-input" bind:value={rxPatientGender}>
							<option value="Male">Male</option>
							<option value="Female">Female</option>
							<option value="Other">Other</option>
						</select>
					</div>
				</div>

				<!-- Vitals Section -->
				<div class="p-3 bg-slate-900/60 rounded border border-slate-700/60 mb-3 space-y-2">
					<div class="text-xs font-bold text-sky-400">Clinical Vitals</div>
					<div class="grid grid-cols-3 gap-2">
						<div>
							<label class="input-label" for="v-bp">BP (mmHg)</label>
							<input id="v-bp" class="text-input" type="text" bind:value={vitalsBP} />
						</div>
						<div>
							<label class="input-label" for="v-pulse">Pulse (bpm)</label>
							<input id="v-pulse" class="text-input" type="number" bind:value={vitalsPulse} />
						</div>
						<div>
							<label class="input-label" for="v-temp">Temp (°F)</label>
							<input id="v-temp" class="text-input" type="text" bind:value={vitalsTemp} />
						</div>
					</div>
					<div class="grid grid-cols-2 gap-2">
						<div>
							<label class="input-label" for="v-spo2">SpO2 (%)</label>
							<input id="v-spo2" class="text-input" type="number" bind:value={vitalsSpO2} />
						</div>
						<div>
							<label class="input-label" for="v-wt">Weight (kg)</label>
							<input id="v-wt" class="text-input" type="number" bind:value={vitalsWeight} />
						</div>
					</div>
				</div>

				<div class="form-group mb-3">
					<label class="input-label" for="rx-diag">Clinical Diagnosis / Complaints</label>
					<input id="rx-diag" class="text-input" type="text" bind:value={rxDiagnosis} />
				</div>

				<div class="form-group mb-3">
					<label class="input-label" for="rx-lab">Lab Investigations Advised</label>
					<input id="rx-lab" class="text-input" type="text" bind:value={rxLabTests} />
				</div>

				<div class="form-group mb-3">
					<label class="input-label" for="rx-adv">Lifestyle / Dietary Advice</label>
					<textarea id="rx-adv" class="text-input h-20" bind:value={rxAdvice}></textarea>
				</div>

				<div class="flex gap-2">
					<button class="btn btn-secondary flex-1" onclick={addRxItem}>+ Add Medicine</button>
					<button class="btn btn-primary flex-1" onclick={saveAndPublishRx}>Save &amp; Sign Rx</button>
				</div>
			</div>

			<!-- Right Printable Clinical Prescription Pad -->
			<div class="rx-slip-paper printable-rx">
				<!-- Clinic Header -->
				<div class="clinic-header">
					<div class="clinic-info">
						<h2 class="clinic-name">AYYA MULTI-SPECIALTY CLINIC &amp; EMR</h2>
						<p class="clinic-meta">24/7 Primary Healthcare · Lab Diagnostics · Pharmacy</p>
						<p class="clinic-meta">Address: 128 Gandhi Road, T. Nagar, Chennai - 600017 | Tel: 044-2435 9900</p>
					</div>
					<div class="doctor-badge-box">
						<div class="doc-title">{selectedDoctor.name}</div>
						<div class="doc-sub">{selectedDoctor.qualification}</div>
						<div class="doc-sub text-emerald-700 font-bold">{selectedDoctor.specialty}</div>
						<div class="doc-sub">Reg. No: <strong>{selectedDoctor.registrationNumber}</strong></div>
					</div>
				</div>

				<div class="rx-divider"></div>

				<!-- Patient Demographics & Vitals Row -->
				<div class="patient-banner">
					<div class="patient-line">
						<span><strong>Patient:</strong> {rxPatientName}</span>
						<span><strong>Age/Sex:</strong> {rxPatientAge}Y / {rxPatientGender}</span>
						<span><strong>Mobile:</strong> {rxPatientMobile}</span>
						<span><strong>Date:</strong> {rxDate}</span>
					</div>
					<div class="vitals-line">
						<span><strong>Vitals:</strong> BP: {vitalsBP} mmHg</span>
						<span>Pulse: {vitalsPulse} bpm</span>
						<span>Temp: {vitalsTemp}°F</span>
						<span>SpO2: {vitalsSpO2}%</span>
						<span>Wt: {vitalsWeight} kg</span>
					</div>
				</div>

				<!-- Diagnosis -->
				<div class="diagnosis-block">
					<span class="diag-label">Clinical Diagnosis:</span>
					<span class="diag-value">{rxDiagnosis}</span>
				</div>

				<!-- Rx Symbol & Medication Table -->
				<div class="rx-symbol-row">
					<span class="rx-symbol">℞</span>
					<span class="rx-note text-xs text-slate-500">(Take medications strictly as instructed below)</span>
				</div>

				<table class="rx-table">
					<thead>
						<tr>
							<th class="w-10">#</th>
							<th>Medicine Name &amp; Formulation</th>
							<th>Dosage (M-A-N)</th>
							<th>Instruction</th>
							<th>Days</th>
							<th>Total Qty</th>
							<th class="no-print">Action</th>
						</tr>
					</thead>
					<tbody>
						{#each rxItems as item, idx}
							<tr>
								<td>{idx + 1}</td>
								<td>
									<strong>{item.name}</strong>
									{#if item.scheduleH}
										<span class="schedule-pill">Schedule H</span>
									{/if}
								</td>
								<td class="font-bold text-center">{item.dosage}</td>
								<td>{item.timing}</td>
								<td class="text-center">{item.durationDays} Days</td>
								<td class="text-center font-bold">{item.qty}</td>
								<td class="no-print text-center">
									<button class="btn-delete" onclick={() => removeRxItem(idx)}>✕</button>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>

				<!-- Lab Investigations & Advice -->
				<div class="advice-grid">
					{#if rxLabTests}
						<div class="advice-box">
							<div class="advice-heading">🔬 Investigations Recommended:</div>
							<p>{rxLabTests}</p>
						</div>
					{/if}
					{#if rxAdvice}
						<div class="advice-box">
							<div class="advice-heading">📋 Dietary &amp; General Advice:</div>
							<p>{rxAdvice}</p>
						</div>
					{/if}
				</div>

				<!-- Signature Block -->
				<div class="rx-footer">
					<div class="disclaimer">
						* Emergency Contact: 108. Please present this prescription at the pharmacy counter.
					</div>
					<div class="signature-block">
						<div class="sig-space"></div>
						<div class="sig-doctor-name">{selectedDoctor.name}</div>
						<div class="text-xs text-slate-600">Reg: {selectedDoctor.registrationNumber}</div>
					</div>
				</div>
			</div>
		</div>
	{/if}

	<!-- TAB 2: PHARMACY STRIP DISPENSER & SCHEDULE H CALCULATOR -->
	{#if activeTab === 'pharmacy'}
		<div class="section-container">
			<div class="section-header">
				<div>
					<h2 class="section-title">Retail Pharmacy Loose Strip &amp; Blister Pack Dispenser</h2>
					<p class="section-desc">Instantly compute fractional pricing when patients purchase partial tablet counts from full manufacturer blister strips.</p>
				</div>
			</div>

			<div class="pharmacy-grid">
				<div class="pharma-card">
					<h3 class="form-section-title">Medicine Strip Parameters</h3>
					
					<div class="form-group mb-3">
						<label class="input-label" for="ph-name">Medicine / Formulation Name</label>
						<input id="ph-name" class="text-input" type="text" value="Tab. Augmentin 625 Duo" />
					</div>

					<div class="grid grid-cols-2 gap-3 mb-3">
						<div>
							<label class="input-label" for="ph-mrp">Full Blister Strip MRP (₹)</label>
							<div class="input-prefix">
								<span>₹</span>
								<input id="ph-mrp" type="number" bind:value={pharmaStripMRP} />
							</div>
						</div>
						<div>
							<label class="input-label" for="ph-tabs">Tablets per Strip</label>
							<input id="ph-tabs" class="text-input" type="number" bind:value={pharmaTabletsPerStrip} />
						</div>
					</div>

					<div class="form-group mb-3">
						<label class="input-label" for="ph-req">Patient Required Tablets Count</label>
						<input id="ph-req" class="text-input font-bold text-sky-400 text-lg" type="number" bind:value={pharmaRequiredTabs} />
						<div class="text-xs text-slate-400 mt-1">Prescribed count from doctor prescription slip</div>
					</div>

					<div class="p-3 bg-slate-900/60 rounded border border-slate-700/60 mb-3 space-y-2">
						<label class="flex items-center gap-2 text-xs text-amber-400 font-bold">
							<input type="checkbox" bind:checked={pharmaScheduleHWarning} />
							<span>Schedule H / H1 Restricted Drug (Requires Prescribing Doctor Record)</span>
						</label>
						<div class="text-xs text-slate-400">
							Batch Expiry: <strong>{pharmaExpiryDate}</strong> (Shelf Life: OK)
						</div>
					</div>
				</div>

				<div class="pharma-card bill-card">
					<h3 class="form-section-title text-sky-400">Counter Dispensing Breakdown</h3>

					<div class="dispense-breakdown">
						<div class="dispense-row">
							<span>Per Tablet Unit Rate:</span>
							<strong class="font-mono text-white">₹{pharmaPerTabPrice.toFixed(2)}</strong>
						</div>
						<div class="dispense-row">
							<span>Fractional Strip Quantity:</span>
							<strong class="font-mono text-sky-400">{pharmaStripsToCut} Strips ({pharmaRequiredTabs} Tablets)</strong>
						</div>

						{#if pharmaScheduleHWarning}
							<div class="schedule-warning-box">
								⚠️ <strong>Statutory Warning (Schedule H / H1):</strong> To be sold by retail on the prescription of a Registered Medical Practitioner only. Doctor MCI license recorded.
							</div>
						{/if}

						<div class="pharma-total-box">
							<div>
								<div class="text-xs text-slate-300 uppercase">Payable at Pharmacy Counter</div>
								<div class="text-3xl font-extrabold text-emerald-400">
									₹{pharmaPayableAmount.toLocaleString('en-IN')}
								</div>
							</div>
							<button class="btn btn-primary" onclick={() => {
								statusMsg = `✅ Billed ${pharmaRequiredTabs} tablets for ₹${pharmaPayableAmount}! Printed thermal pharmacy slip.`;
								setTimeout(() => statusMsg = '', 3000);
							}}>
								Dispense &amp; Cut Strip
							</button>
						</div>
					</div>
				</div>
			</div>
		</div>
	{/if}

	<!-- TAB 3: LIVE OPD QUEUE TOKEN CALLING -->
	{#if activeTab === 'opd'}
		<div class="section-container">
			<div class="section-header">
				<div>
					<h2 class="section-title">OPD Patient Queue &amp; Digital Calling Display</h2>
					<p class="section-desc">Live waiting room display board with 1-click audio calling into doctor cabins.</p>
				</div>
				<button class="btn btn-primary" onclick={callNextPatient}>
					📢 Call Next Patient (Token #{nextQueueToken})
				</button>
			</div>

			<div class="opd-board-grid">
				<div class="opd-display-hero">
					<div class="hero-token-label">NOW SERVING IN CABIN 1</div>
					<div class="hero-token-number">#{currentServingToken}</div>
					<div class="hero-doctor">{selectedDoctor.name} ({selectedDoctor.specialty})</div>
				</div>

				<div class="opd-queue-card">
					<h3 class="text-base font-bold text-white mb-3">Next in Line Waiting Queue</h3>
					<div class="space-y-2">
						{#each opdWaitingList as item}
							<div class="queue-item">
								<div class="queue-token">Token #{item.token}</div>
								<div class="queue-details">
									<div class="font-bold text-white">{item.name}</div>
									<div class="text-xs text-slate-400">{item.doctor}</div>
								</div>
								<span class="badge" class:status-available={item.status === 'IN_CABIN'}>{item.status}</span>
							</div>
						{/each}
					</div>
				</div>
			</div>
		</div>
	{/if}

	<!-- TAB 4: DOCTOR CONSULTATIONS & TELEMEDICINE -->
	{#if activeTab === 'appointments'}
		<div class="section-container">
			<div class="section-header">
				<div>
					<h2 class="section-title">Consultations &amp; Telemedicine Records</h2>
					<p class="section-desc">WebRTC video room integration, clinic appointment schedule, and patient history.</p>
				</div>
				<button class="btn btn-primary" onclick={() => showBookModal = true}>
					+ Book Patient Consultation
				</button>
			</div>

			<div class="table-card">
				<table class="data-table">
					<thead>
						<tr>
							<th>Patient Name</th>
							<th>Attending Doctor</th>
							<th>Consultation Type</th>
							<th>Scheduled Time</th>
							<th>Status</th>
							<th>Actions</th>
						</tr>
					</thead>
					<tbody>
						{#each appointments as apt}
							<tr>
								<td>
									<div class="font-bold text-white">{apt.patientName}</div>
									<div class="text-xs text-slate-400">ID: {apt.id}</div>
								</td>
								<td>{apt.doctorName || selectedDoctor.name}</td>
								<td>
									<span class="consult-badge">{apt.consultationType}</span>
								</td>
								<td>{new Date(apt.scheduledAt).toLocaleString()}</td>
								<td>
									<span class="status-badge status-available">{apt.status}</span>
								</td>
								<td>
									{#if apt.consultationType === 'VIDEO'}
										<button class="btn-sm btn-outline-sky" onclick={() => alert(`Launching WebRTC Doctor Room token: ${apt.roomToken}`)}>
											🎥 Join Video Call
										</button>
									{:else}
										<button class="btn-sm btn-outline-success" onclick={() => {
											rxPatientName = apt.patientName;
											activeTab = 'rx';
										}}>
											Write Rx
										</button>
									{/if}
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</div>
	{/if}

	<!-- MODAL: BOOK CONSULTATION -->
	{#if showBookModal}
		<div class="modal-backdrop">
			<div class="modal-container max-w-lg">
				<div class="modal-header">
					<h3 class="text-lg font-bold text-white">Book Patient Consultation</h3>
					<button class="btn btn-secondary" onclick={() => showBookModal = false}>✕</button>
				</div>
				<div class="modal-body space-y-4">
					<div>
						<label class="input-label" for="b-doc">Select Doctor</label>
						<select id="b-doc" class="text-input" bind:value={bookDocId}>
							{#each doctors as d}
								<option value={d.id}>{d.name} ({d.specialty}) - Fee: ₹{d.consultationFee}</option>
							{/each}
						</select>
					</div>
					<div>
						<label class="input-label" for="b-pname">Patient Full Name</label>
						<input id="b-pname" class="text-input" type="text" bind:value={bookPatientName} placeholder="e.g. Ramesh Kannan" />
					</div>
					<div>
						<label class="input-label" for="b-phone">Patient Mobile</label>
						<input id="b-phone" class="text-input" type="tel" bind:value={bookPatientPhone} placeholder="+91 98400 12345" />
					</div>
					<div>
						<label class="input-label" for="b-type">Consultation Mode</label>
						<select id="b-type" class="text-input" bind:value={bookConsultMode}>
							<option value="VIDEO">Video Consultation (Telemedicine WebRTC)</option>
							<option value="IN_PERSON">In-Clinic Physical Visit</option>
						</select>
					</div>
					<div>
						<label class="input-label" for="b-time">Appointment Date &amp; Time</label>
						<input id="b-time" class="text-input" type="datetime-local" bind:value={bookDateTime} />
					</div>
				</div>
				<div class="modal-footer">
					<button class="btn btn-secondary" onclick={() => showBookModal = false}>Cancel</button>
					<button class="btn btn-primary" onclick={handleBookConsultation}>Confirm Appointment</button>
				</div>
			</div>
		</div>
	{/if}
</div>

<style>
	.healthcare-app {
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
		background: rgba(16, 185, 129, 0.15);
		color: #34d399;
		border: 1px solid rgba(16, 185, 129, 0.3);
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

	.btn-primary { background: #10b981; color: #ffffff; }
	.btn-primary:hover { background: #059669; }
	.btn-secondary { background: #334155; color: #f1f5f9; }
	.btn-secondary:hover { background: #475569; }

	.btn-outline-sky { background: transparent; border: 1px solid #0284c7; color: #38bdf8; }
	.btn-outline-success { background: transparent; border: 1px solid #10b981; color: #34d399; }

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
		color: #34d399;
		border-bottom-color: #34d399;
	}

	.tab-btn:hover:not(.active) {
		color: #cbd5e1;
	}

	/* RX WORKSPACE */
	.rx-workspace {
		display: grid;
		grid-template-columns: 380px 1fr;
		gap: 1.5rem;
	}

	@media (max-width: 1024px) {
		.rx-workspace {
			grid-template-columns: 1fr;
		}
	}

	.rx-controls {
		background: #1e293b;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 14px;
		padding: 1.5rem;
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
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: 6px;
		color: #ffffff;
		font-size: 0.8125rem;
	}

	/* PRINTABLE RX PAD */
	.rx-slip-paper {
		background: #ffffff;
		color: #0f172a;
		border: 2px solid #cbd5e1;
		border-radius: 8px;
		padding: 2rem;
		box-shadow: 0 10px 25px rgba(0,0,0,0.3);
		min-height: 700px;
		display: flex;
		flex-direction: column;
		justify-content: space-between;
	}

	.clinic-header {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
	}

	.clinic-name {
		font-size: 1.25rem;
		font-weight: 900;
		color: #047857;
		letter-spacing: -0.01em;
	}

	.clinic-meta {
		font-size: 0.75rem;
		color: #475569;
		margin-top: 0.15rem;
	}

	.doctor-badge-box {
		text-align: right;
		font-size: 0.75rem;
	}

	.doc-title {
		font-size: 1rem;
		font-weight: 800;
		color: #0f172a;
	}

	.doc-sub {
		color: #334155;
		margin-top: 0.1rem;
	}

	.rx-divider {
		height: 2px;
		background: #047857;
		margin: 0.75rem 0;
	}

	.patient-banner {
		background: #f1f5f9;
		border: 1px solid #cbd5e1;
		border-radius: 6px;
		padding: 0.6rem 0.8rem;
		font-size: 0.75rem;
		margin-bottom: 0.75rem;
	}

	.patient-line, .vitals-line {
		display: flex;
		justify-content: space-between;
		gap: 1rem;
		flex-wrap: wrap;
	}

	.vitals-line {
		margin-top: 0.35rem;
		padding-top: 0.35rem;
		border-top: 1px dashed #cbd5e1;
		color: #1e293b;
	}

	.diagnosis-block {
		font-size: 0.8125rem;
		margin-bottom: 0.75rem;
	}

	.diag-label {
		font-weight: 700;
		color: #047857;
	}

	.diag-value {
		font-weight: 600;
		color: #0f172a;
		margin-left: 0.5rem;
	}

	.rx-symbol-row {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		margin-bottom: 0.5rem;
	}

	.rx-symbol {
		font-size: 2rem;
		font-weight: 900;
		color: #047857;
		font-family: serif;
	}

	.rx-table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.8125rem;
		margin-bottom: 1.25rem;
	}

	.rx-table th, .rx-table td {
		border-bottom: 1px solid #e2e8f0;
		padding: 0.5rem 0.6rem;
		text-align: left;
	}

	.rx-table th {
		background: #f8fafc;
		color: #475569;
		font-size: 0.75rem;
	}

	.schedule-pill {
		font-size: 0.625rem;
		background: #fee2e2;
		color: #b91c1c;
		padding: 0.1rem 0.35rem;
		border-radius: 4px;
		margin-left: 0.4rem;
	}

	.btn-delete {
		background: transparent;
		border: none;
		color: #ef4444;
		cursor: pointer;
		font-weight: bold;
	}

	.advice-grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 1rem;
		font-size: 0.75rem;
		margin-bottom: 1.5rem;
	}

	.advice-box {
		background: #f8fafc;
		border: 1px solid #e2e8f0;
		border-radius: 6px;
		padding: 0.6rem;
	}

	.advice-heading {
		font-weight: 700;
		color: #334155;
		margin-bottom: 0.25rem;
	}

	.rx-footer {
		display: flex;
		justify-content: space-between;
		align-items: flex-end;
		border-top: 1px solid #cbd5e1;
		padding-top: 1rem;
	}

	.disclaimer {
		font-size: 0.6875rem;
		color: #64748b;
		max-width: 400px;
	}

	.signature-block {
		text-align: center;
		width: 200px;
	}

	.sig-space {
		height: 40px;
		border-bottom: 1px solid #0f172a;
		margin-bottom: 0.35rem;
	}

	.sig-doctor-name {
		font-size: 0.8125rem;
		font-weight: 800;
		color: #0f172a;
	}

	/* PHARMACY TAB */
	.pharmacy-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
		gap: 1.5rem;
	}

	.pharma-card {
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 12px;
		padding: 1.5rem;
	}

	.form-section-title {
		font-size: 1rem;
		font-weight: 700;
		color: #ffffff;
		margin-bottom: 1rem;
	}

	.input-prefix {
		display: flex;
		align-items: center;
		background: #1e293b;
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: 6px;
		padding: 0 0.75rem;
	}

	.input-prefix span {
		color: #94a3b8;
		font-weight: 600;
	}

	.input-prefix input {
		width: 100%;
		background: transparent;
		border: none;
		color: #ffffff;
		padding: 0.5rem;
		font-size: 1rem;
		outline: none;
	}

	.bill-card {
		border-color: rgba(56, 189, 248, 0.3);
	}

	.dispense-breakdown {
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}

	.dispense-row {
		display: flex;
		justify-content: space-between;
		font-size: 0.875rem;
		padding-bottom: 0.6rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.06);
	}

	.schedule-warning-box {
		background: rgba(239, 68, 68, 0.1);
		border: 1px solid rgba(239, 68, 68, 0.3);
		color: #fca5a5;
		padding: 0.75rem;
		border-radius: 8px;
		font-size: 0.75rem;
		line-height: 1.4;
	}

	.pharma-total-box {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 1rem;
		background: rgba(16, 185, 129, 0.1);
		border: 1px solid rgba(16, 185, 129, 0.3);
		border-radius: 10px;
	}

	/* OPD BOARD */
	.opd-board-grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 1.5rem;
	}

	@media (max-width: 768px) {
		.opd-board-grid {
			grid-template-columns: 1fr;
		}
	}

	.opd-display-hero {
		background: #0f172a;
		border: 2px solid #10b981;
		border-radius: 16px;
		padding: 3rem 2rem;
		text-align: center;
		display: flex;
		flex-direction: column;
		justify-content: center;
		box-shadow: 0 0 30px rgba(16, 185, 129, 0.15);
	}

	.hero-token-label {
		font-size: 0.875rem;
		font-weight: 800;
		letter-spacing: 0.1em;
		color: #34d399;
	}

	.hero-token-number {
		font-size: 5rem;
		font-weight: 900;
		color: #ffffff;
		font-family: monospace;
		line-height: 1.1;
		margin: 0.5rem 0;
	}

	.hero-doctor {
		font-size: 1.125rem;
		color: #cbd5e1;
		font-weight: 600;
	}

	.opd-queue-card {
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 14px;
		padding: 1.5rem;
	}

	.queue-item {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 0.75rem;
		background: #1e293b;
		border-radius: 8px;
		border: 1px solid rgba(255, 255, 255, 0.05);
	}

	.queue-token {
		font-family: monospace;
		font-weight: 800;
		color: #38bdf8;
		font-size: 1rem;
	}

	/* DATA TABLE */
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

	.consult-badge {
		font-size: 0.6875rem;
		font-weight: 700;
		background: rgba(56, 189, 248, 0.15);
		color: #38bdf8;
		padding: 0.2rem 0.5rem;
		border-radius: 4px;
	}

	.status-badge {
		font-size: 0.6875rem;
		font-weight: 700;
		padding: 0.25rem 0.6rem;
		border-radius: 6px;
		text-transform: uppercase;
	}

	.status-available { background: rgba(16, 185, 129, 0.15); color: #34d399; }

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

	@media print {
		.no-print, .nav-tabs, .kpi-grid, .app-header {
			display: none !important;
		}
		body {
			background: white !important;
			color: black !important;
		}
		.rx-slip-paper {
			box-shadow: none !important;
			border: 1px solid #000 !important;
			width: 100% !important;
			margin: 0 auto !important;
			padding: 1rem !important;
		}
	}
</style>
