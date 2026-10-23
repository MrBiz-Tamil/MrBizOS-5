<script lang="ts">
	import { toast } from '$lib/stores/toast.svelte';

	interface Props {
		open: boolean;
		booking: any;
		onclose: () => void;
		onsave?: (updatedBooking: any) => void;
	}

	let { open = false, booking, onclose, onsave }: Props = $props();

	// Local mutable state initialized from booking
	let tripMode = $state<string>('FAMILY');
	let customerCity = $state<string>('Chennai');
	let specialInstructions = $state<string>('');

	// Coordinator
	let coordinatorName = $state('Kavitha R.');
	let coordinatorPhone = $state('+91 98840 11223');
	let coordinatorDept = $state('North India Operations');

	// Driver & Transport Dispatch
	let driverName = $state('Ghulam Nabi');
	let driverPhone = $state('+91 94190 12345');
	let vehicleModel = $state('Toyota Innova Crysta AC (White)');
	let vehiclePlate = $state('JK-01-AB-4491');
	let pickupLocation = $state('Srinagar Airport (SXR) Gate 1 (Placard: "SHARMA FAMILY")');
	let flightMonitor = $state('IndiGo 6E-502 (Tracking on-time)');

	// Local Area Guide
	let guideName = $state('Bashir Ahmed (Govt. Licensed #2094)');
	let guidePhone = $state('+91 97970 88990');
	let guideLanguages = $state('Tamil (conversational), English, Hindi');

	// Local Emergency Station Connection
	let stationLead = $state('Khurshid (Srinagar Station Chief)');
	let emergencyPhone = $state('+91 94190 99881');
	let hospitalPartner = $state('SMHS Hospital Srinagar (15 mins away)');

	// Synchronize when booking changes
	$effect(() => {
		if (booking) {
			tripMode = booking.tripMode || 'FAMILY';
			customerCity = booking.customerCity || 'Chennai';
			specialInstructions = booking.specialInstructions || 'Pure Veg Meals · Child-friendly pacing';

			if (booking.assignedCoordinator) {
				coordinatorName = booking.assignedCoordinator.name || 'Kavitha R.';
				coordinatorPhone = booking.assignedCoordinator.phone || '+91 98840 11223';
				coordinatorDept = booking.assignedCoordinator.department || 'Operations Desk';
			}

			if (booking.logisticsDispatch) {
				driverName = booking.logisticsDispatch.driverName || 'Ghulam Nabi';
				driverPhone = booking.logisticsDispatch.driverPhone || '+91 94190 12345';
				vehicleModel = booking.logisticsDispatch.vehicleModel || 'Innova Crysta AC';
				vehiclePlate = booking.logisticsDispatch.vehiclePlate || 'JK-01-AB-4491';
				pickupLocation = booking.logisticsDispatch.pickupLocation || 'Airport Exit Gate';
				flightMonitor = booking.logisticsDispatch.flightMonitor || 'Flight 6E-502';
				guideName = booking.logisticsDispatch.guideName || 'Bashir Ahmed';
				guidePhone = booking.logisticsDispatch.guidePhone || '+91 97970 88990';
				guideLanguages = booking.logisticsDispatch.guideLanguages || 'Tamil, English, Hindi';
			}

			if (booking.localEmergency) {
				stationLead = booking.localEmergency.stationLead || 'Station Chief';
				emergencyPhone = booking.localEmergency.emergencyPhone || '+91 94190 99881';
				hospitalPartner = booking.localEmergency.hospitalPartner || 'Local Civil Hospital';
			}
		}
	});

	function handleSaveDispatch() {
		if (!booking) return;

		const updated = {
			...booking,
			tripMode,
			customerCity,
			specialInstructions,
			assignedCoordinator: {
				name: coordinatorName,
				phone: coordinatorPhone,
				department: coordinatorDept
			},
			logisticsDispatch: {
				driverName,
				driverPhone,
				vehicleModel,
				vehiclePlate,
				pickupLocation,
				flightMonitor,
				guideName,
				guidePhone,
				guideLanguages
			},
			localEmergency: {
				stationLead,
				emergencyPhone,
				hospitalPartner
			}
		};

		onsave?.(updated);
		toast.success(
			'On-Ground Dispatch Saved',
			`Assigned driver ${driverName} (${vehiclePlate}) & coordinator ${coordinatorName} for ${booking.customerName}.`
		);
		onclose();
	}
</script>

{#if open && booking}
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div class="modal-backdrop animate-fade-in" onclick={(e) => { if (e.target === e.currentTarget) onclose(); }}>
		<div class="modal-card dispatch-modal-card" role="dialog" aria-modal="true" aria-labelledby="dispatch-title">
			
			<!-- Modal Header -->
			<div class="modal-header">
				<div style="display: flex; align-items: center; gap: 0.65rem;">
					<div class="header-icon-box">🚗</div>
					<div>
						<div class="text-xs font-mono" style="color: var(--brand-primary); font-weight: 700;">
							BOOKING #{booking.id} · {booking.destination}
						</div>
						<h2 id="dispatch-title" class="header-title">
							On-Ground Logistics &amp; Driver Dispatch Desk
						</h2>
					</div>
				</div>
				<button type="button" class="close-btn" onclick={onclose}>✕</button>
			</div>

			<!-- Modal Body -->
			<div class="modal-body-scroll">
				<!-- Section 1: Trip Archetype & Traveler Origin -->
				<div class="section-box">
					<div class="section-head">
						<span class="section-number">1</span>
						<h4>Trip Archetype &amp; Traveler Origin Details</h4>
					</div>

					<div class="inputs-grid-3">
						<div class="form-group">
							<label class="form-label" for="inp-mode">Trip Mode / Persona</label>
							<select id="inp-mode" class="form-select" bind:value={tripMode}>
								<option value="FAMILY">👨‍👩‍👧‍👦 Family Tour (Pacing &amp; Veg/Jain Food)</option>
								<option value="BUSINESS">💼 Business / Corporate MICE (VIP Transit)</option>
								<option value="REST_LEISURE">🌴 Rest &amp; Leisure Mode (Slow Travel / Spa)</option>
								<option value="GROUP_TOUR">👥 Group Departure (Fixed Coach &amp; TL)</option>
							</select>
						</div>

						<div class="form-group">
							<label class="form-label" for="inp-city">Customer Origin City</label>
							<input id="inp-city" type="text" class="form-input" bind:value={customerCity} placeholder="e.g. Chennai, Madurai, Kanyakumari" />
						</div>

						<div class="form-group">
							<label class="form-label" for="inp-notes">Special Instructions</label>
							<input id="inp-notes" type="text" class="form-input" bind:value={specialInstructions} placeholder="e.g. Pure Veg Meals, Child Crib, Senior Wheelchair" />
						</div>
					</div>
				</div>

				<!-- Section 2: Dedicated Agency Coordinator -->
				<div class="section-box">
					<div class="section-head">
						<span class="section-number">2</span>
						<h4>Assigned Agency Trip Coordinator</h4>
					</div>

					<div class="inputs-grid-3">
						<div class="form-group">
							<label class="form-label" for="inp-coord-name">Coordinator Officer Name</label>
							<input id="inp-coord-name" type="text" class="form-input" bind:value={coordinatorName} />
						</div>

						<div class="form-group">
							<label class="form-label" for="inp-coord-phone">Direct Mobile / WhatsApp</label>
							<input id="inp-coord-phone" type="tel" class="form-input font-mono" bind:value={coordinatorPhone} />
						</div>

						<div class="form-group">
							<label class="form-label" for="inp-coord-dept">Operations Desk</label>
							<input id="inp-coord-dept" type="text" class="form-input" bind:value={coordinatorDept} />
						</div>
					</div>
				</div>

				<!-- Section 3: Car Pickup, Driver & Vehicle Plate Details -->
				<div class="section-box highlight-box">
					<div class="section-head">
						<span class="section-number">3</span>
						<h4>Car Pickup &amp; Chauffeur Details (Shared with Customer)</h4>
					</div>

					<div class="inputs-grid-2">
						<div class="form-group">
							<label class="form-label" for="inp-drv-name">Chauffeur / Driver Name</label>
							<input id="inp-drv-name" type="text" class="form-input font-bold" bind:value={driverName} />
						</div>

						<div class="form-group">
							<label class="form-label" for="inp-drv-phone">Driver Mobile (WhatsApp Available)</label>
							<input id="inp-drv-phone" type="tel" class="form-input font-mono" bind:value={driverPhone} />
						</div>

						<div class="form-group">
							<label class="form-label" for="inp-veh-model">Vehicle Model &amp; Type</label>
							<input id="inp-veh-model" type="text" class="form-input" bind:value={vehicleModel} placeholder="e.g. Toyota Innova Crysta AC (White)" />
						</div>

						<div class="form-group">
							<label class="form-label" for="inp-veh-plate">Vehicle License Plate Number</label>
							<input id="inp-veh-plate" type="text" class="form-input font-mono font-bold" style="color: var(--brand-primary);" bind:value={vehiclePlate} placeholder="e.g. JK-01-AB-4491" />
						</div>
					</div>

					<div class="inputs-grid-2" style="margin-top: 0.75rem;">
						<div class="form-group">
							<label class="form-label" for="inp-pickup-loc">Exact Pickup Point &amp; Placard Text</label>
							<input id="inp-pickup-loc" type="text" class="form-input" bind:value={pickupLocation} placeholder="e.g. Airport Gate 1 with Name Placard" />
						</div>

						<div class="form-group">
							<label class="form-label" for="inp-flight-mon">Live Flight Delay Tracking</label>
							<input id="inp-flight-mon" type="text" class="form-input font-mono" bind:value={flightMonitor} placeholder="e.g. IndiGo 6E-502 (Tracking)" />
						</div>
					</div>
				</div>

				<!-- Section 4: Local Guide & Emergency Station Desk -->
				<div class="section-box">
					<div class="section-head">
						<span class="section-number">4</span>
						<h4>Local Certified Guide &amp; Destination Station Emergency Lead</h4>
					</div>

					<div class="inputs-grid-2">
						<div class="form-group">
							<label class="form-label" for="inp-guide-name">Local Guide Name &amp; License No</label>
							<input id="inp-guide-name" type="text" class="form-input" bind:value={guideName} />
						</div>

						<div class="form-group">
							<label class="form-label" for="inp-guide-lang">Languages Spoken by Guide</label>
							<input id="inp-guide-lang" type="text" class="form-input" bind:value={guideLanguages} placeholder="e.g. Tamil, English, Hindi" />
						</div>

						<div class="form-group">
							<label class="form-label" for="inp-stat-lead">Destination Station Emergency Lead</label>
							<input id="inp-stat-lead" type="text" class="form-input" bind:value={stationLead} />
						</div>

						<div class="form-group">
							<label class="form-label" for="inp-stat-phone">24/7 Ground Emergency Helpline</label>
							<input id="inp-stat-phone" type="tel" class="form-input font-mono" bind:value={emergencyPhone} />
						</div>
					</div>
				</div>
			</div>

			<!-- Modal Footer -->
			<div class="modal-footer">
				<button type="button" class="btn btn-secondary" onclick={onclose}>Cancel</button>
				<button type="button" class="btn btn-primary" onclick={handleSaveDispatch}>
					💾 Save On-Ground Dispatch Details
				</button>
			</div>
		</div>
	</div>
{/if}

<style>
	.modal-backdrop {
		position: fixed;
		inset: 0;
		background: rgba(15, 23, 42, 0.85);
		backdrop-filter: blur(8px);
		z-index: 9999;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 1.5rem;
	}

	.dispatch-modal-card {
		background: var(--surface-main);
		border: 1px solid var(--border-default);
		border-radius: var(--radius-lg);
		width: 100%;
		max-width: 780px;
		max-height: 90vh;
		display: flex;
		flex-direction: column;
		box-shadow: var(--shadow-xl);
		overflow: hidden;
	}

	.modal-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 1.25rem 1.5rem;
		border-bottom: 1px solid var(--border-default);
		background: var(--surface-secondary);
	}

	.header-icon-box {
		width: 40px;
		height: 40px;
		border-radius: var(--radius-md);
		background: rgba(79, 70, 229, 0.12);
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 1.25rem;
		border: 1px solid rgba(79, 70, 229, 0.25);
	}

	.header-title {
		font-size: 1.15rem;
		font-weight: 800;
		margin: 0;
		color: var(--text-primary);
	}

	.close-btn {
		background: none;
		border: none;
		color: var(--text-muted);
		font-size: 1.25rem;
		cursor: pointer;
		padding: 0.25rem;
	}

	.modal-body-scroll {
		padding: 1.5rem;
		overflow-y: auto;
		display: flex;
		flex-direction: column;
		gap: 1.25rem;
	}

	.section-box {
		background: var(--surface-secondary);
		border: 1px solid var(--border-default);
		border-radius: var(--radius-md);
		padding: 1.25rem;
	}

	.section-box.highlight-box {
		border-color: rgba(79, 70, 229, 0.35);
		background: rgba(79, 70, 229, 0.04);
	}

	.section-head {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		margin-bottom: 1rem;
	}

	.section-head h4 {
		font-size: 0.875rem;
		font-weight: 800;
		color: var(--text-primary);
		margin: 0;
		letter-spacing: 0.02em;
	}

	.section-number {
		width: 22px;
		height: 22px;
		border-radius: 50%;
		background: var(--brand-primary);
		color: #ffffff;
		font-size: 0.75rem;
		font-weight: 800;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.inputs-grid-2 {
		display: grid;
		grid-template-columns: repeat(2, 1fr);
		gap: 0.85rem;
	}

	.inputs-grid-3 {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 0.85rem;
	}

	@media (max-width: 700px) {
		.inputs-grid-2, .inputs-grid-3 {
			grid-template-columns: 1fr;
		}
	}

	.modal-footer {
		display: flex;
		justify-content: flex-end;
		gap: 0.75rem;
		padding: 1rem 1.5rem;
		border-top: 1px solid var(--border-default);
		background: var(--surface-secondary);
	}
</style>
