<script lang="ts">
	import { toast } from '$lib/stores/toast.svelte';

	interface Props {
		open: boolean;
		onclose: () => void;
	}

	let { open = false, onclose }: Props = $props();

	let tripType = $state<'oneway' | 'round' | 'multi'>('round');
	let fromCity = $state('DEL - New Delhi, Indira Gandhi Intl (DEL)');
	let toCity = $state('DXB - Dubai International Airport (DXB)');
	let depDate = $state('22 Nov 2026');
	let retDate = $state('26 Nov 2026');
	let adults = $state(2);
	let children = $state(0);
	let cabinClass = $state('Economy');

	let searchDone = $state(false);

	let flightResults = $state([
		{
			id: 'fl-1',
			airline: 'Emirates',
			flightNo: 'EK-511',
			depart: '10:30 AM (DEL)',
			arrive: '01:10 PM (DXB)',
			duration: '4h 10m • Non-stop',
			price: '₹ 28,450 / pax',
			seats: '9 seats left'
		},
		{
			id: 'fl-2',
			airline: 'IndiGo 6E',
			flightNo: '6E-1455',
			depart: '02:15 PM (DEL)',
			arrive: '04:55 PM (DXB)',
			duration: '4h 10m • Non-stop',
			price: '₹ 19,890 / pax',
			seats: '4 seats left'
		},
		{
			id: 'fl-3',
			airline: 'Air India',
			flightNo: 'AI-995',
			depart: '08:00 PM (DEL)',
			arrive: '10:35 PM (DXB)',
			duration: '4h 05m • Non-stop',
			price: '₹ 22,100 / pax',
			seats: '6 seats left'
		}
	]);

	function handleSearch() {
		searchDone = true;
		toast.success('Live flight availability loaded across GDS & aggregators!');
	}

	function handleSelectFlight(fl: any) {
		toast.success(`${fl.airline} ${fl.flightNo} selected! PNR hold created.`);
		onclose();
	}
</script>

{#if open}
	<div class="modal-backdrop" onclick={(e) => { if (e.target === e.currentTarget) onclose(); }}>
		<div class="modal-card animate-in" role="dialog" aria-modal="true">
			<!-- Header -->
			<div class="modal-header">
				<div>
					<h3 class="modal-title">Flight &amp; Ticket Booking Desk</h3>
					<span class="sub-text">GDS flight availability, live fare search and instant ticket issuance</span>
				</div>
				<button type="button" class="close-btn" onclick={onclose}>✕</button>
			</div>

			<!-- Search Form -->
			<div class="modal-body">
				<div class="trip-type-selector">
					<label class="radio-label">
						<input type="radio" name="tt" value="oneway" bind:group={tripType} />
						<span>One Way</span>
					</label>
					<label class="radio-label">
						<input type="radio" name="tt" value="round" bind:group={tripType} />
						<span>Round Trip</span>
					</label>
					<label class="radio-label">
						<input type="radio" name="tt" value="multi" bind:group={tripType} />
						<span>Multi City</span>
					</label>
				</div>

				<div class="grid-2-cols mt-3">
					<div class="form-group">
						<label class="form-label" for="fl-from">From (Origin)</label>
						<input id="fl-from" type="text" class="form-input" bind:value={fromCity} />
					</div>
					<div class="form-group">
						<label class="form-label" for="fl-to">To (Destination)</label>
						<input id="fl-to" type="text" class="form-input" bind:value={toCity} />
					</div>
				</div>

				<div class="grid-3-cols mt-3">
					<div class="form-group">
						<label class="form-label" for="fl-dep">Departure Date</label>
						<input id="fl-dep" type="text" class="form-input font-mono" bind:value={depDate} />
					</div>
					<div class="form-group">
						<label class="form-label" for="fl-ret">Return Date</label>
						<input id="fl-ret" type="text" class="form-input font-mono" bind:value={retDate} disabled={tripType === 'oneway'} />
					</div>
					<div class="form-group">
						<label class="form-label" for="fl-class">Cabin Class</label>
						<select id="fl-class" class="form-select" bind:value={cabinClass}>
							<option value="Economy">Economy</option>
							<option value="Premium Economy">Premium Economy</option>
							<option value="Business">Business Class</option>
							<option value="First">First Class</option>
						</select>
					</div>
				</div>

				<div class="search-action-row mt-3">
					<span class="pax-indicator">Passengers: <strong>{adults} Adults</strong>, {children} Child</span>
					<button type="button" class="btn btn-primary" onclick={handleSearch}>
						✈️ Search Available Flights
					</button>
				</div>

				<!-- Live Search Results -->
				{#if searchDone}
					<div class="flight-results mt-4">
						<span class="results-title">Available Direct &amp; Connecting Flights</span>
						<div class="results-list">
							{#each flightResults as fl}
								<div class="flight-card">
									<div class="fl-airline-col">
										<span class="fl-name">{fl.airline}</span>
										<span class="fl-no font-mono">{fl.flightNo}</span>
									</div>
									<div class="fl-timing-col">
										<span class="fl-time">{fl.depart} → {fl.arrive}</span>
										<span class="fl-dur">{fl.duration}</span>
									</div>
									<div class="fl-price-col">
										<span class="fl-price font-bold">{fl.price}</span>
										<span class="fl-seats">{fl.seats}</span>
									</div>
									<button type="button" class="btn btn-sm btn-outline" onclick={() => handleSelectFlight(fl)}>
										Select &amp; Hold
									</button>
								</div>
							{/each}
						</div>
					</div>
				{/if}
			</div>

			<!-- Footer -->
			<div class="modal-footer">
				<button type="button" class="btn btn-secondary" onclick={onclose}>Close</button>
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
		max-width: 720px;
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

	.modal-body { padding: 1.5rem; }

	.trip-type-selector {
		display: flex;
		gap: 1.5rem;
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

	.grid-2-cols { display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem; }
	.grid-3-cols { display: grid; grid-template-columns: repeat(3, 1fr); gap: 0.75rem; }

	.form-group { display: flex; flex-direction: column; gap: 0.35rem; }
	.form-label { font-size: 0.75rem; font-weight: 700; color: #475569; text-transform: uppercase; }

	.form-input, .form-select {
		width: 100%;
		padding: 0.55rem 0.75rem;
		border: 1px solid #cbd5e1;
		border-radius: 6px;
		font-size: 0.875rem;
		color: #0f172a;
		background: #ffffff;
	}

	.search-action-row {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding-top: 0.5rem;
	}

	.pax-indicator { font-size: 0.8125rem; color: #64748b; }
	.pax-indicator strong { color: #0f172a; }

	.flight-results {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
		border-top: 1px solid #e2e8f0;
		padding-top: 1rem;
	}

	.results-title {
		font-size: 0.8125rem;
		font-weight: 700;
		color: #475569;
		text-transform: uppercase;
		letter-spacing: 0.04em;
	}

	.results-list {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.flight-card {
		display: flex;
		justify-content: space-between;
		align-items: center;
		background: #f8fafc;
		border: 1px solid #e2e8f0;
		border-radius: 8px;
		padding: 0.85rem 1rem;
		font-size: 0.8125rem;
	}

	.fl-airline-col { display: flex; flex-direction: column; }
	.fl-name { font-weight: 700; color: #0f172a; }
	.fl-no { font-size: 0.7rem; color: #64748b; }

	.fl-timing-col { display: flex; flex-direction: column; text-align: center; }
	.fl-time { font-weight: 600; color: #0f172a; }
	.fl-dur { font-size: 0.7rem; color: #64748b; }

	.fl-price-col { display: flex; flex-direction: column; text-align: right; }
	.fl-price { color: #2563eb; }
	.fl-seats { font-size: 0.7rem; color: #059669; }

	.modal-footer {
		padding: 1rem 1.5rem;
		border-top: 1px solid #e2e8f0;
		display: flex;
		justify-content: flex-end;
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
	.btn-sm { padding: 0.35rem 0.75rem; font-size: 0.75rem; }
	.btn-secondary { background: #f1f5f9; color: #334155; }
	.btn-outline { background: #ffffff; border-color: #cbd5e1; color: #2563eb; }
	.btn-outline:hover { background: #eff6ff; }
	.btn-primary { background: #2563eb; color: #ffffff; }
	.btn-primary:hover { background: #1d4ed8; }

	.mt-3 { margin-top: 0.75rem; }
	.mt-4 { margin-top: 1rem; }
	.font-bold { font-weight: 700; }
</style>
