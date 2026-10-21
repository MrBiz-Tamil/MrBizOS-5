<script lang="ts">
	import { onMount } from 'svelte';
	import { healthcareGrpc } from '$lib/grpc';

	let doctors = $state<any[]>([]);
	let appointments = $state<any[]>([]);
	let loading = $state(true);
	let error = $state('');

	// Booking modal
	let showBookModal = $state(false);
	let selectedDoctorId = $state('');
	let patientName = $state('');
	let patientPhone = $state('');
	let scheduledAt = $state('');
	let consultationType = $state('VIDEO');
	let notes = $state('');
	let bookingSubmitting = $state(false);

	onMount(async () => {
		try {
			const [docsRes, apptsRes]: [any, any] = await Promise.all([
				healthcareGrpc.listDoctors(),
				healthcareGrpc.listAppointments()
			]);
			doctors = docsRes?.doctors || [];
			appointments = apptsRes?.appointments || [];
		} catch (err: any) {
			error = err.message || 'Failed to connect to Healthcare gRPC Service';
		} finally {
			loading = false;
		}
	});

	async function handleBook() {
		bookingSubmitting = true;
		try {
			const res: any = await healthcareGrpc.bookAppointment({
				doctorId: selectedDoctorId,
				patientName,
				patientPhone,
				scheduledAt: scheduledAt || new Date().toISOString(),
				consultationType,
				notes
			});
			if (res.appointment) {
				appointments = [res.appointment, ...appointments];
				showBookModal = false;
				patientName = '';
				patientPhone = '';
				notes = '';
			}
		} catch (err: any) {
			alert('Booking error: ' + err.message);
		} finally {
			bookingSubmitting = false;
		}
	}
</script>

<div class="healthcare-dashboard">
	<div class="header">
		<div>
			<h1 class="title">Smart Healthcare & Telemedicine Hub</h1>
			<p class="subtitle">Doctor consultations, EMR records, and WebRTC video signaling powered by gRPC.</p>
		</div>
		<button class="btn-primary" onclick={() => showBookModal = true}>
			+ Book Consultation
		</button>
	</div>

	{#if error}
		<div class="error-banner">
			⚠️ {error}
		</div>
	{/if}

	{#if loading}
		<div class="loading-state">Connecting to gRPC Service...</div>
	{:else}
		<div class="grid-2">
			<!-- Available Doctors -->
			<div class="card">
				<h2 class="section-title">🏥 Verified Specialists</h2>
				<div class="doctors-list">
					{#each doctors as doc}
						<div class="doctor-row">
							<div class="avatar">👨‍⚕️</div>
							<div class="doc-info">
								<div class="doc-name">{doc.name}</div>
								<div class="doc-meta">{doc.specialty} · {doc.qualification}</div>
								<div class="doc-fee">Fee: ₹{doc.consultationFee}</div>
							</div>
							<span class="badge active">Available</span>
						</div>
					{/each}
				</div>
			</div>

			<!-- Active Appointments -->
			<div class="card">
				<h2 class="section-title">📅 Consultations & Schedules</h2>
				<div class="appointments-list">
					{#each appointments as apt}
						<div class="appointment-row">
							<div>
								<div class="patient-name">{apt.patientName}</div>
								<div class="apt-meta">{apt.doctorName} · {apt.consultationType}</div>
							</div>
							<div class="apt-actions">
								<span class="badge" class:confirmed={apt.status === 'CONFIRMED'}>{apt.status}</span>
								{#if apt.consultationType === 'VIDEO'}
									<button class="btn-call" onclick={() => alert('Launching WebRTC / LiveKit Video Room: ' + apt.roomToken)}>
										🎥 Join Call
									</button>
								{/if}
							</div>
						</div>
					{/each}
				</div>
			</div>
		</div>
	{/if}

	{#if showBookModal}
		<div class="modal-backdrop">
			<div class="modal-box">
				<h3>Schedule Patient Consultation</h3>
				<div class="form-group">
					<label for="doc-select">Select Doctor</label>
					<select id="doc-select" bind:value={selectedDoctorId}>
						{#each doctors as d}
							<option value={d.id}>{d.name} ({d.specialty})</option>
						{/each}
					</select>
				</div>
				<div class="form-group">
					<label for="patient-name">Patient Full Name</label>
					<input id="patient-name" type="text" bind:value={patientName} placeholder="Enter full name" />
				</div>
				<div class="form-group">
					<label for="patient-phone">Phone Number</label>
					<input id="patient-phone" type="tel" bind:value={patientPhone} placeholder="+91 98765 43210" />
				</div>
				<div class="form-group">
					<label for="consult-mode">Consultation Mode</label>
					<select id="consult-mode" bind:value={consultationType}>
						<option value="VIDEO">Video Consultation (Telemedicine)</option>
						<option value="IN_PERSON">In-Clinic Visit</option>
					</select>
				</div>
				<div class="modal-actions">
					<button class="btn-cancel" onclick={() => showBookModal = false}>Cancel</button>
					<button class="btn-primary" disabled={bookingSubmitting} onclick={handleBook}>
						{bookingSubmitting ? 'Confirming...' : 'Confirm Appointment'}
					</button>
				</div>
			</div>
		</div>
	{/if}
</div>

<style>
	.healthcare-dashboard {
		padding: 1.5rem;
		max-width: 1400px;
		margin: 0 auto;
	}
	.header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 2rem;
	}
	.title {
		font-size: 1.75rem;
		font-weight: 700;
		color: #f8fafc;
	}
	.subtitle {
		font-size: 0.875rem;
		color: #94a3b8;
		margin-top: 0.25rem;
	}
	.btn-primary {
		background: #10b981;
		color: #ffffff;
		border: none;
		padding: 0.625rem 1.25rem;
		border-radius: 8px;
		font-weight: 600;
		cursor: pointer;
		transition: background 0.15s;
	}
	.btn-primary:hover {
		background: #059669;
	}
	.error-banner {
		background: rgba(239, 68, 68, 0.1);
		border: 1px solid #ef4444;
		color: #fca5a5;
		padding: 0.75rem 1rem;
		border-radius: 8px;
		margin-bottom: 1.5rem;
	}
	.loading-state {
		text-align: center;
		padding: 3rem;
		color: #94a3b8;
	}
	.grid-2 {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
		gap: 1.5rem;
	}
	.card {
		background: #1e293b;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 12px;
		padding: 1.5rem;
	}
	.section-title {
		font-size: 1.125rem;
		font-weight: 600;
		color: #f8fafc;
		margin-bottom: 1rem;
	}
	.doctors-list, .appointments-list {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}
	.doctor-row, .appointment-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0.75rem;
		background: rgba(255, 255, 255, 0.03);
		border: 1px solid rgba(255, 255, 255, 0.06);
		border-radius: 8px;
	}
	.avatar {
		font-size: 1.5rem;
		margin-right: 0.75rem;
	}
	.doc-info {
		flex: 1;
	}
	.doc-name, .patient-name {
		font-weight: 600;
		color: #f1f5f9;
	}
	.doc-meta, .apt-meta {
		font-size: 0.75rem;
		color: #94a3b8;
	}
	.doc-fee {
		font-size: 0.75rem;
		color: #10b981;
		font-weight: 600;
	}
	.badge {
		font-size: 0.75rem;
		padding: 0.2rem 0.5rem;
		border-radius: 4px;
		background: rgba(255, 255, 255, 0.1);
		color: #cbd5e1;
	}
	.badge.active, .badge.confirmed {
		background: rgba(16, 185, 129, 0.15);
		color: #34d399;
	}
	.apt-actions {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}
	.btn-call {
		background: #0284c7;
		color: white;
		border: none;
		padding: 0.35rem 0.75rem;
		border-radius: 6px;
		font-size: 0.75rem;
		cursor: pointer;
	}
	.modal-backdrop {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.7);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 1000;
	}
	.modal-box {
		background: #1e293b;
		border: 1px solid rgba(255, 255, 255, 0.15);
		border-radius: 12px;
		padding: 1.5rem;
		width: 100%;
		max-width: 480px;
	}
	.form-group {
		margin-bottom: 1rem;
	}
	.form-group label {
		display: block;
		font-size: 0.75rem;
		color: #94a3b8;
		margin-bottom: 0.25rem;
	}
	.form-group input, .form-group select {
		width: 100%;
		padding: 0.5rem 0.75rem;
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: 6px;
		color: white;
	}
	.modal-actions {
		display: flex;
		justify-content: flex-end;
		gap: 0.75rem;
		margin-top: 1.5rem;
	}
	.btn-cancel {
		background: transparent;
		border: 1px solid rgba(255, 255, 255, 0.2);
		color: #94a3b8;
		padding: 0.5rem 1rem;
		border-radius: 6px;
		cursor: pointer;
	}
</style>
