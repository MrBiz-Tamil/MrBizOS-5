<script lang="ts">
	import { toast } from '$lib/stores/toast.svelte';

	interface Props {
		open: boolean;
		passenger?: any;
		onclose: () => void;
		onsave?: (pax: any) => void;
	}

	let { open = false, passenger, onclose, onsave }: Props = $props();

	let activeTab = $state<'basic' | 'passport' | 'contact' | 'special'>('basic');

	// Form State
	let title = $state('Mr.');
	let firstName = $state('Rahul');
	let lastName = $state('Sharma');
	let dob = $state('12 Apr 1985');
	let gender = $state('Male');
	let nationality = $state('Indian');

	let passportNumber = $state('Z1234567');
	let passportExpiry = $state('15 Mar 2030');
	let issueCountry = $state('India');

	let dietary = $state('Vegetarian');
	let specialAssistance = $state('None');
	let notes = $state('');

	function handleSave() {
		const updated = {
			title,
			firstName,
			lastName,
			fullName: `${title} ${firstName} ${lastName}`,
			dob,
			gender,
			nationality,
			passportNumber,
			passportExpiry,
			issueCountry,
			dietary,
			specialAssistance,
			notes
		};

		toast.success(`Passenger details for ${updated.fullName} saved.`);
		onsave?.(updated);
		onclose();
	}
</script>

{#if open}
	<div class="modal-backdrop" onclick={(e) => { if (e.target === e.currentTarget) onclose(); }}>
		<div class="modal-card animate-in" role="dialog" aria-modal="true">
			<!-- Header -->
			<div class="modal-header">
				<div>
					<h3 class="modal-title">Add / Edit Passenger</h3>
					<span class="sub-text">Passenger identification, manifest compliance and dietary requirements</span>
				</div>
				<button type="button" class="close-btn" onclick={onclose}>✕</button>
			</div>

			<!-- Tab Navigation -->
			<div class="modal-tabs">
				<button 
					type="button" 
					class="tab-btn" 
					class:active={activeTab === 'basic'}
					onclick={() => (activeTab = 'basic')}
				>
					Basic Details
				</button>
				<button 
					type="button" 
					class="tab-btn" 
					class:active={activeTab === 'passport'}
					onclick={() => (activeTab = 'passport')}
				>
					Passport &amp; ID
				</button>
				<button 
					type="button" 
					class="tab-btn" 
					class:active={activeTab === 'contact'}
					onclick={() => (activeTab = 'contact')}
				>
					Contact
				</button>
				<button 
					type="button" 
					class="tab-btn" 
					class:active={activeTab === 'special'}
					onclick={() => (activeTab = 'special')}
				>
					Special Request
				</button>
			</div>

			<!-- Form Content -->
			<div class="modal-body">
				{#if activeTab === 'basic'}
					<div class="form-grid-3">
						<div class="form-group">
							<label class="form-label" for="pax-title">Title</label>
							<select id="pax-title" class="form-select" bind:value={title}>
								<option value="Mr.">Mr.</option>
								<option value="Mrs.">Mrs.</option>
								<option value="Ms.">Ms.</option>
								<option value="Master">Master</option>
								<option value="Dr.">Dr.</option>
							</select>
						</div>
						<div class="form-group">
							<label class="form-label" for="pax-fn">First Name</label>
							<input id="pax-fn" type="text" class="form-input" bind:value={firstName} />
						</div>
						<div class="form-group">
							<label class="form-label" for="pax-ln">Last Name</label>
							<input id="pax-ln" type="text" class="form-input" bind:value={lastName} />
						</div>
					</div>

					<div class="form-grid-3 mt-3">
						<div class="form-group">
							<label class="form-label" for="pax-dob">Date of Birth</label>
							<input id="pax-dob" type="text" class="form-input font-mono" bind:value={dob} />
						</div>
						<div class="form-group">
							<label class="form-label" for="pax-gender">Gender</label>
							<select id="pax-gender" class="form-select" bind:value={gender}>
								<option value="Male">Male</option>
								<option value="Female">Female</option>
								<option value="Other">Other</option>
							</select>
						</div>
						<div class="form-group">
							<label class="form-label" for="pax-nat">Nationality</label>
							<input id="pax-nat" type="text" class="form-input" bind:value={nationality} />
						</div>
					</div>

				{:else if activeTab === 'passport'}
					<div class="form-grid-3">
						<div class="form-group">
							<label class="form-label" for="pax-pass-num">Passport Number</label>
							<input id="pax-pass-num" type="text" class="form-input font-mono" bind:value={passportNumber} />
						</div>
						<div class="form-group">
							<label class="form-label" for="pax-pass-exp">Passport Expiry Date</label>
							<input id="pax-pass-exp" type="text" class="form-input font-mono" bind:value={passportExpiry} />
						</div>
						<div class="form-group">
							<label class="form-label" for="pax-pass-iss">Issue Country</label>
							<input id="pax-pass-iss" type="text" class="form-input" bind:value={issueCountry} />
						</div>
					</div>

					<div class="passport-validity-alert mt-3">
						<span class="alert-icon">🛡️</span>
						<span>Passport validity is <strong>greater than 6 months</strong> from scheduled departure date. Cleared for international travel.</span>
					</div>

				{:else if activeTab === 'contact'}
					<div class="form-grid-2">
						<div class="form-group">
							<label class="form-label" for="pax-phone">Emergency Contact Phone</label>
							<input id="pax-phone" type="tel" class="form-input font-mono" value="+91 98401 22334" />
						</div>
						<div class="form-group">
							<label class="form-label" for="pax-email">Email Address</label>
							<input id="pax-email" type="email" class="form-input" value="rahul.sharma@example.com" />
						</div>
					</div>

				{:else if activeTab === 'special'}
					<div class="form-grid-2">
						<div class="form-group">
							<label class="form-label" for="pax-diet">Dietary Preference</label>
							<select id="pax-diet" class="form-select" bind:value={dietary}>
								<option value="Vegetarian">Vegetarian (Pure Veg)</option>
								<option value="Jain">Jain Meal</option>
								<option value="Non-Vegetarian">Non-Vegetarian (Halal)</option>
								<option value="Vegan">Vegan</option>
								<option value="Gluten-Free">Gluten-Free</option>
							</select>
						</div>
						<div class="form-group">
							<label class="form-label" for="pax-assist">Special Assistance</label>
							<select id="pax-assist" class="form-select" bind:value={specialAssistance}>
								<option value="None">None</option>
								<option value="Wheelchair">Wheelchair at Airport</option>
								<option value="Elderly">Elderly Pacing</option>
								<option value="InfantCot">Infant Bassinet</option>
							</select>
						</div>
					</div>

					<div class="form-group mt-3">
						<label class="form-label" for="pax-notes">Additional Notes</label>
						<textarea id="pax-notes" class="form-textarea" rows="2" placeholder="Enter any special requests..." bind:value={notes}></textarea>
					</div>
				{/if}
			</div>

			<!-- Footer -->
			<div class="modal-footer">
				<button type="button" class="btn btn-secondary" onclick={onclose}>Cancel</button>
				<button type="button" class="btn btn-primary" onclick={handleSave}>Save Passenger</button>
			</div>
		</div>
	</div>
{/if}

<style>
	.modal-backdrop {
		position: fixed;
		inset: 0;
		background: rgba(15, 23, 42, 0.6);
		backdrop-filter: blur(4px);
		z-index: 9999;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 1.5rem;
	}

	.modal-card {
		background: #ffffff;
		border: 1px solid #e2e8f0;
		border-radius: 12px;
		width: 100%;
		max-width: 620px;
		display: flex;
		flex-direction: column;
		box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1);
		overflow: hidden;
		color: #0f172a;
	}

	.animate-in { animation: modalFadeIn 0.18s ease-out; }
	@keyframes modalFadeIn {
		from { opacity: 0; transform: scale(0.98); }
		to { opacity: 1; transform: scale(1); }
	}

	.modal-header {
		padding: 1.25rem 1.5rem;
		border-bottom: 1px solid #e2e8f0;
		display: flex;
		justify-content: space-between;
		align-items: center;
	}

	.modal-title { font-size: 1.25rem; font-weight: 800; margin: 0; }
	.sub-text { font-size: 0.8125rem; color: #64748b; }
	.close-btn { background: none; border: none; font-size: 1.25rem; color: #64748b; cursor: pointer; }

	.modal-tabs {
		display: flex;
		background: #f8fafc;
		border-bottom: 1px solid #e2e8f0;
		padding: 0 1rem;
	}

	.tab-btn {
		padding: 0.65rem 1rem;
		background: transparent;
		border: none;
		border-bottom: 2px solid transparent;
		font-size: 0.8125rem;
		font-weight: 600;
		color: #64748b;
		cursor: pointer;
	}
	.tab-btn:hover { color: #0f172a; }
	.tab-btn.active {
		color: #2563eb;
		border-bottom-color: #2563eb;
	}

	.modal-body {
		padding: 1.5rem;
	}

	.form-grid-3 {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 0.75rem;
	}

	.form-grid-2 {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 0.75rem;
	}

	.form-group {
		display: flex;
		flex-direction: column;
		gap: 0.35rem;
	}

	.form-label {
		font-size: 0.75rem;
		font-weight: 700;
		color: #475569;
		text-transform: uppercase;
		letter-spacing: 0.04em;
	}

	.form-input, .form-select, .form-textarea {
		width: 100%;
		padding: 0.55rem 0.75rem;
		border: 1px solid #cbd5e1;
		border-radius: 6px;
		font-size: 0.875rem;
		color: #0f172a;
		background: #ffffff;
	}
	.form-input:focus, .form-select:focus, .form-textarea:focus {
		outline: none;
		border-color: #2563eb;
		box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.1);
	}

	.passport-validity-alert {
		background: #ecfdf5;
		border: 1px solid #a7f3d0;
		border-radius: 6px;
		padding: 0.75rem 1rem;
		font-size: 0.8125rem;
		color: #065f46;
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.modal-footer {
		padding: 1rem 1.5rem;
		border-top: 1px solid #e2e8f0;
		display: flex;
		justify-content: space-between;
		align-items: center;
		background: #ffffff;
	}

	.btn {
		font-size: 0.8125rem;
		font-weight: 600;
		padding: 0.5rem 1rem;
		border-radius: 8px;
		cursor: pointer;
		border: 1px solid transparent;
	}
	.btn-secondary { background: #f1f5f9; color: #334155; }
	.btn-primary { background: #2563eb; color: #ffffff; }
	.btn-primary:hover { background: #1d4ed8; }

	.mt-3 { margin-top: 0.75rem; }
</style>
