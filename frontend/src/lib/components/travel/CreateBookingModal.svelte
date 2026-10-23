<script lang="ts">
	import { toast } from '$lib/stores/toast.svelte';

	interface Props {
		open: boolean;
		onclose: () => void;
		onsave?: (booking: any) => void;
		initialCustomerName?: string;
		initialDestination?: string;
		initialAmount?: string;
	}

	let { open = false, onclose, onsave, initialCustomerName, initialDestination, initialAmount }: Props = $props();

	let currentStep = $state<number>(1);
	let customerType = $state<'existing' | 'new'>('existing');
	let searchCustomer = $state('Rahul Sharma');
	let travelType = $state<'international' | 'domestic' | 'both'>('domestic');
	let dateRange = $state('14 Nov 2026 - 19 Nov 2026');
	let adults = $state(2);
	let children = $state(0);
	let infants = $state(0);

	let selectedPackage = $state({
		id: 'pkg-kashmir',
		name: 'Kashmir Valley Paradise (4D/5N)',
		destination: 'Srinagar - Gulmarg - Pahalgam',
		basePrice: '₹ 1,65,000'
	});

	$effect(() => {
		if (initialCustomerName) searchCustomer = initialCustomerName;
		if (initialDestination) {
			selectedPackage = {
				id: 'pkg-custom',
				name: initialDestination + ' Package',
				destination: initialDestination,
				basePrice: initialAmount || '₹ 1,65,000'
			};
		}
	});

	function handleNext() {
		if (currentStep < 4) {
			currentStep += 1;
		} else {
			handleCompleteBooking();
		}
	}

	function handlePrev() {
		if (currentStep > 1) {
			currentStep -= 1;
		}
	}

	function handleCompleteBooking() {
		const newBooking = {
			id: `TRV-${Math.floor(100000 + Math.random() * 900000)}`,
			customerName: searchCustomer || 'New Customer',
			customerCity: 'Chennai',
			customerPhone: '+91 98401 22334',
			destination: selectedPackage.destination,
			packageTitle: selectedPackage.name,
			travelDates: dateRange,
			paxCount: adults + children + infants,
			adults,
			children,
			infants,
			totalAmount: selectedPackage.basePrice,
			status: 'CONFIRMED',
			travelType
		};

		toast.success(`Booking ${newBooking.id} created successfully!`);
		onsave?.(newBooking);
		onclose();
		currentStep = 1;
	}
</script>

{#if open}
	<div class="modal-backdrop" onclick={(e) => { if (e.target === e.currentTarget) onclose(); }}>
		<div class="modal-card animate-in" role="dialog" aria-modal="true">
			<!-- Header -->
			<div class="modal-header">
				<div class="header-left">
					<h3 class="modal-title">Create New Booking</h3>
					<span class="sub-text">End-to-end reservation and inventory allocation</span>
				</div>
				<button type="button" class="close-btn" onclick={onclose}>✕</button>
			</div>

			<!-- Stepper -->
			<div class="stepper-bar">
				<div class="step-item" class:active={currentStep >= 1} class:current={currentStep === 1}>
					<span class="step-num">1</span>
					<span class="step-name">Customer</span>
				</div>
				<div class="step-line" class:filled={currentStep >= 2}></div>
				<div class="step-item" class:active={currentStep >= 2} class:current={currentStep === 2}>
					<span class="step-num">2</span>
					<span class="step-name">Package</span>
				</div>
				<div class="step-line" class:filled={currentStep >= 3}></div>
				<div class="step-item" class:active={currentStep >= 3} class:current={currentStep === 3}>
					<span class="step-num">3</span>
					<span class="step-name">Travellers</span>
				</div>
				<div class="step-line" class:filled={currentStep >= 4}></div>
				<div class="step-item" class:active={currentStep >= 4} class:current={currentStep === 4}>
					<span class="step-num">4</span>
					<span class="step-name">Review</span>
				</div>
			</div>

			<!-- Modal Body -->
			<div class="modal-body">
				{#if currentStep === 1}
					<div class="form-section">
						<label class="section-label">Customer Type</label>
						<div class="radio-row">
							<label class="radio-label">
								<input type="radio" name="custType" value="existing" bind:group={customerType} />
								<span>Existing Customer</span>
							</label>
							<label class="radio-label">
								<input type="radio" name="custType" value="new" bind:group={customerType} />
								<span>New Customer</span>
							</label>
						</div>

						<div class="input-group">
							<label for="cust-search" class="input-label">Search Customer</label>
							<div class="search-input-wrapper">
								<span class="input-icon">🔍</span>
								<input 
									id="cust-search"
									type="text" 
									class="form-input with-icon" 
									placeholder="Search by name, phone or email..."
									bind:value={searchCustomer} 
								/>
							</div>
						</div>

						<label class="section-label mt-4">Travel Type</label>
						<div class="pill-selector">
							<button 
								type="button" 
								class="pill-btn" 
								class:active={travelType === 'international'}
								onclick={() => (travelType = 'international')}
							>
								🌍 International
							</button>
							<button 
								type="button" 
								class="pill-btn" 
								class:active={travelType === 'domestic'}
								onclick={() => (travelType = 'domestic')}
							>
								🇮🇳 Domestic
							</button>
							<button 
								type="button" 
								class="pill-btn" 
								class:active={travelType === 'both'}
								onclick={() => (travelType = 'both')}
							>
								✨ Both / Multi-City
							</button>
						</div>

						<div class="input-group mt-4">
							<label for="date-range" class="input-label">Preferred Dates</label>
							<input 
								id="date-range"
								type="text" 
								class="form-input font-mono" 
								bind:value={dateRange} 
								placeholder="e.g. 14 Nov 2026 - 19 Nov 2026"
							/>
						</div>

						<div class="travellers-grid mt-4">
							<div class="pax-counter-card">
								<span class="pax-type">Adults</span>
								<span class="pax-sub">12+ years</span>
								<div class="counter-box">
									<button type="button" class="cnt-btn" onclick={() => { if (adults > 1) adults -= 1; }}>-</button>
									<span class="cnt-val">{adults}</span>
									<button type="button" class="cnt-btn" onclick={() => (adults += 1)}>+</button>
								</div>
							</div>
							<div class="pax-counter-card">
								<span class="pax-type">Children</span>
								<span class="pax-sub">2-11 years</span>
								<div class="counter-box">
									<button type="button" class="cnt-btn" onclick={() => { if (children > 0) children -= 1; }}>-</button>
									<span class="cnt-val">{children}</span>
									<button type="button" class="cnt-btn" onclick={() => (children += 1)}>+</button>
								</div>
							</div>
							<div class="pax-counter-card">
								<span class="pax-type">Infants</span>
								<span class="pax-sub">Below 2 years</span>
								<div class="counter-box">
									<button type="button" class="cnt-btn" onclick={() => { if (infants > 0) infants -= 1; }}>-</button>
									<span class="cnt-val">{infants}</span>
									<button type="button" class="cnt-btn" onclick={() => (infants += 1)}>+</button>
								</div>
							</div>
						</div>
					</div>
				{:else if currentStep === 2}
					<div class="form-section">
						<label class="section-label">Select Tour Package & Departure</label>
						<div class="package-list">
							<div 
								class="pkg-choice-card" 
								class:selected={selectedPackage.id === 'pkg-kashmir'}
								onclick={() => (selectedPackage = { id: 'pkg-kashmir', name: 'Kashmir Valley Paradise (4D/5N)', destination: 'Srinagar - Gulmarg - Pahalgam', basePrice: '₹ 1,65,000' })}
							>
								<div class="pkg-header">
									<span class="pkg-name">Kashmir Valley Paradise (4D/5N)</span>
									<span class="pkg-price">₹ 1,65,000</span>
								</div>
								<div class="pkg-meta">Srinagar - Gulmarg - Pahalgam • Domestic Circuit • Luxury Innova Included</div>
							</div>

							<div 
								class="pkg-choice-card" 
								class:selected={selectedPackage.id === 'pkg-dubai'}
								onclick={() => (selectedPackage = { id: 'pkg-dubai', name: 'Dubai Special Desert & City (5D/4N)', destination: 'Dubai - Abu Dhabi', basePrice: '₹ 1,95,000' })}
							>
								<div class="pkg-header">
									<span class="pkg-name">Dubai Special Desert & City (5D/4N)</span>
									<span class="pkg-price">₹ 1,95,000</span>
								</div>
								<div class="pkg-meta">Dubai - Abu Dhabi • Visa Included • 4-Star Stay</div>
							</div>

							<div 
								class="pkg-choice-card" 
								class:selected={selectedPackage.id === 'pkg-europe'}
								onclick={() => (selectedPackage = { id: 'pkg-europe', name: 'Europe Classic Express (7D/6N)', destination: 'Paris - Rome - Zurich', basePrice: '₹ 2,40,000' })}
							>
								<div class="pkg-header">
									<span class="pkg-name">Europe Classic Express (7D/6N)</span>
									<span class="pkg-price">₹ 2,40,000</span>
								</div>
								<div class="pkg-meta">Paris - Rome - Zurich • Schengen Visa • Flight Excluded</div>
							</div>
						</div>
					</div>
				{:else if currentStep === 3}
					<div class="form-section">
						<label class="section-label">Traveller Manifest ({adults + children + infants} Passengers)</label>
						<div class="traveller-manifest-preview">
							<div class="traveller-row">
								<span class="t-badge">Lead Pax</span>
								<span class="t-name">{searchCustomer || 'Primary Traveller'}</span>
								<span class="t-age">Adult (34 yrs)</span>
								<span class="t-status ok">Passport Valid ✓</span>
							</div>
							{#if adults > 1}
								<div class="traveller-row">
									<span class="t-badge">Pax 2</span>
									<span class="t-name">Companion Traveller</span>
									<span class="t-age">Adult (31 yrs)</span>
									<span class="t-status ok">Passport Valid ✓</span>
								</div>
							{/if}
						</div>
					</div>
				{:else if currentStep === 4}
					<div class="review-box">
						<div class="review-header">
							<h4>Booking Summary & Quotation</h4>
							<span class="pill-green">Draft Ready</span>
						</div>
						<div class="review-grid">
							<div>
								<span class="r-label">Customer</span>
								<span class="r-val">{searchCustomer}</span>
							</div>
							<div>
								<span class="r-label">Package</span>
								<span class="r-val">{selectedPackage.name}</span>
							</div>
							<div>
								<span class="r-label">Travel Dates</span>
								<span class="r-val">{dateRange}</span>
							</div>
							<div>
								<span class="r-label">Total Amount</span>
								<span class="r-val text-primary font-bold">{selectedPackage.basePrice}</span>
							</div>
						</div>
					</div>
				{/if}
			</div>

			<!-- Footer Actions -->
			<div class="modal-footer">
				<button type="button" class="btn btn-secondary" onclick={onclose}>Cancel</button>
				<div class="footer-right">
					{#if currentStep > 1}
						<button type="button" class="btn btn-outline" onclick={handlePrev}>← Previous</button>
					{/if}
					<button type="button" class="btn btn-primary" onclick={handleNext}>
						{currentStep === 4 ? 'Confirm & Create Booking' : 'Next Step →'}
					</button>
				</div>
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
		max-width: 640px;
		max-height: 90vh;
		display: flex;
		flex-direction: column;
		box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1);
		overflow: hidden;
		color: #0f172a;
	}

	.animate-in {
		animation: modalFadeIn 0.18s ease-out;
	}

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

	.modal-title {
		font-size: 1.25rem;
		font-weight: 800;
		color: #0f172a;
		margin: 0;
	}

	.sub-text {
		font-size: 0.8125rem;
		color: #64748b;
	}

	.close-btn {
		background: transparent;
		border: none;
		font-size: 1.25rem;
		color: #64748b;
		cursor: pointer;
		padding: 0.25rem;
	}
	.close-btn:hover { color: #0f172a; }

	.stepper-bar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 1rem 1.5rem;
		background: #f8fafc;
		border-bottom: 1px solid #e2e8f0;
	}

	.step-item {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 0.8125rem;
		font-weight: 600;
		color: #94a3b8;
	}
	.step-item.active { color: #0f172a; }
	.step-item.current { color: #2563eb; }

	.step-num {
		width: 24px;
		height: 24px;
		border-radius: 50%;
		background: #e2e8f0;
		color: #64748b;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 0.75rem;
		font-weight: 700;
	}
	.step-item.active .step-num { background: #bfdbfe; color: #1e40af; }
	.step-item.current .step-num { background: #2563eb; color: #ffffff; }

	.step-line {
		flex: 1;
		height: 2px;
		background: #e2e8f0;
		margin: 0 0.5rem;
	}
	.step-line.filled { background: #2563eb; }

	.modal-body {
		padding: 1.5rem;
		overflow-y: auto;
		flex: 1;
	}

	.section-label {
		font-size: 0.8125rem;
		font-weight: 700;
		color: #475569;
		display: block;
		margin-bottom: 0.5rem;
		text-transform: uppercase;
		letter-spacing: 0.04em;
	}

	.radio-row {
		display: flex;
		gap: 1.5rem;
		margin-bottom: 1rem;
	}

	.radio-label {
		display: flex;
		align-items: center;
		gap: 0.4rem;
		font-size: 0.875rem;
		font-weight: 600;
		color: #0f172a;
		cursor: pointer;
	}

	.input-group {
		display: flex;
		flex-direction: column;
		gap: 0.35rem;
	}

	.input-label {
		font-size: 0.8125rem;
		font-weight: 600;
		color: #475569;
	}

	.search-input-wrapper {
		position: relative;
	}

	.input-icon {
		position: absolute;
		left: 0.75rem;
		top: 50%;
		transform: translateY(-50%);
		font-size: 0.9rem;
	}

	.form-input {
		width: 100%;
		padding: 0.6rem 0.85rem;
		border: 1px solid #cbd5e1;
		border-radius: 8px;
		font-size: 0.875rem;
		color: #0f172a;
		background: #ffffff;
	}
	.form-input.with-icon { padding-left: 2.2rem; }
	.form-input:focus {
		outline: none;
		border-color: #2563eb;
		box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.1);
	}

	.pill-selector {
		display: flex;
		gap: 0.5rem;
		flex-wrap: wrap;
	}

	.pill-btn {
		padding: 0.45rem 0.85rem;
		border-radius: 8px;
		border: 1px solid #e2e8f0;
		background: #ffffff;
		font-size: 0.8125rem;
		font-weight: 600;
		color: #475569;
		cursor: pointer;
	}
	.pill-btn.active {
		background: #eff6ff;
		color: #2563eb;
		border-color: #bfdbfe;
	}

	.travellers-grid {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 0.75rem;
	}

	.pax-counter-card {
		background: #f8fafc;
		border: 1px solid #e2e8f0;
		border-radius: 8px;
		padding: 0.75rem;
		text-align: center;
		display: flex;
		flex-direction: column;
		align-items: center;
	}

	.pax-type {
		font-size: 0.8125rem;
		font-weight: 700;
		color: #0f172a;
	}
	.pax-sub {
		font-size: 0.7rem;
		color: #94a3b8;
		margin-bottom: 0.5rem;
	}

	.counter-box {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}

	.cnt-btn {
		width: 28px;
		height: 28px;
		border-radius: 6px;
		border: 1px solid #cbd5e1;
		background: #ffffff;
		font-weight: 700;
		cursor: pointer;
	}

	.cnt-val {
		font-size: 0.9375rem;
		font-weight: 800;
		color: #0f172a;
		min-width: 16px;
	}

	.package-list {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	.pkg-choice-card {
		padding: 1rem;
		border: 1px solid #e2e8f0;
		border-radius: 8px;
		cursor: pointer;
		transition: all 0.15s ease;
	}
	.pkg-choice-card:hover { border-color: #94a3b8; }
	.pkg-choice-card.selected {
		border-color: #2563eb;
		background: #eff6ff;
	}

	.pkg-header {
		display: flex;
		justify-content: space-between;
		font-weight: 700;
		font-size: 0.9375rem;
		margin-bottom: 0.25rem;
	}
	.pkg-price { color: #2563eb; }
	.pkg-meta { font-size: 0.75rem; color: #64748b; }

	.traveller-manifest-preview {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.traveller-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0.75rem 1rem;
		background: #f8fafc;
		border: 1px solid #e2e8f0;
		border-radius: 6px;
		font-size: 0.8125rem;
	}

	.t-badge {
		font-weight: 700;
		font-size: 0.7rem;
		background: #eff6ff;
		color: #2563eb;
		padding: 0.15rem 0.4rem;
		border-radius: 4px;
	}

	.t-name { font-weight: 600; color: #0f172a; }
	.t-age { color: #64748b; }
	.t-status.ok { color: #059669; font-weight: 600; }

	.review-box {
		background: #f8fafc;
		border: 1px solid #e2e8f0;
		border-radius: 8px;
		padding: 1.25rem;
	}

	.review-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 1rem;
		padding-bottom: 0.5rem;
		border-bottom: 1px solid #e2e8f0;
	}

	.review-grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 1rem;
	}

	.r-label {
		display: block;
		font-size: 0.75rem;
		color: #64748b;
		text-transform: uppercase;
		font-weight: 600;
	}
	.r-val {
		font-size: 0.9375rem;
		color: #0f172a;
	}

	.pill-green {
		font-size: 0.75rem;
		font-weight: 700;
		color: #059669;
		background: #ecfdf5;
		padding: 0.2rem 0.5rem;
		border-radius: 9999px;
	}

	.modal-footer {
		padding: 1rem 1.5rem;
		border-top: 1px solid #e2e8f0;
		display: flex;
		justify-content: space-between;
		align-items: center;
		background: #ffffff;
	}

	.footer-right {
		display: flex;
		gap: 0.5rem;
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
	.btn-outline { background: #ffffff; border-color: #cbd5e1; color: #475569; }
	.btn-primary { background: #2563eb; color: #ffffff; }
	.btn-primary:hover { background: #1d4ed8; }

	.mt-4 { margin-top: 1rem; }
	.text-primary { color: #2563eb; }
	.font-bold { font-weight: 700; }
</style>
