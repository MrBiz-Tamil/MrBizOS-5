<script lang="ts">
	import { toast } from '$lib/stores/toast.svelte';

	interface Props {
		open: boolean;
		onclose: () => void;
	}

	let { open = false, onclose }: Props = $props();

	let activeDay = $state(4);
	let itineraryDate = $state('17 Nov 2026');

	let days = $state([
		{ day: 1, title: 'Arrival & Dal Lake Shikara', city: 'Srinagar' },
		{ day: 2, title: 'Mughal Gardens & Local Heritage', city: 'Srinagar' },
		{ day: 3, title: 'Gulmarg Gondola & Snow Activities', city: 'Gulmarg' },
		{ day: 4, title: 'Pahalgam & Betaab Valley Expedition', city: 'Pahalgam' },
		{ day: 5, title: 'Departure & Souvenir Shopping', city: 'Srinagar' }
	]);

	let activitiesDay4 = $state([
		{
			id: 'act-1',
			time: '08:00 AM',
			title: 'Breakfast at Hotel Grand Kashmir',
			location: 'Pahalgam Valley',
			category: 'MEAL',
			icon: '🍳',
			notes: 'Buffet spread with authentic Kashmiri Kahwa & bread'
		},
		{
			id: 'act-2',
			time: '09:00 AM',
			title: 'Local Sightseeing Betaab Valley',
			location: 'Betaab Valley Wildlife Zone',
			category: 'SIGHTSEEING',
			icon: '🏔️',
			notes: 'Private Innova pickup, entry permits pre-cleared'
		},
		{
			id: 'act-3',
			time: '01:00 PM',
			title: 'Lunch at Riverside Pine Restaurant',
			location: 'Lidder River Bank',
			category: 'MEAL',
			icon: '🍲',
			notes: 'River-view seating, Kashmiri Wazwan & Vegetarian options'
		},
		{
			id: 'act-4',
			time: '02:00 PM',
			title: 'Aru Valley & Chandanwari Nature Trail',
			location: 'Aru Valley Pass',
			category: 'ACTIVITY',
			icon: '🌲',
			notes: 'Guided scenic walk with local guide Bashir'
		},
		{
			id: 'act-5',
			time: '06:00 PM',
			title: 'Return to Hotel & Evening Bonfire',
			location: 'Resort Lawn',
			category: 'LEISURE',
			icon: '🔥',
			notes: 'Warm fleece blankets & hot tea service'
		}
	]);

	function handleAddDay() {
		const newDayNum = days.length + 1;
		days.push({
			day: newDayNum,
			title: `Day ${newDayNum} - Custom Leisure`,
			city: 'Srinagar'
		});
		activeDay = newDayNum;
		toast.success(`Day ${newDayNum} added to itinerary.`);
	}

	function handleSaveItinerary() {
		toast.success('Day-wise tour itinerary saved and published!');
		onclose();
	}
</script>

{#if open}
	<div class="modal-backdrop" onclick={(e) => { if (e.target === e.currentTarget) onclose(); }}>
		<div class="modal-card animate-in" role="dialog" aria-modal="true">
			<!-- Header -->
			<div class="modal-header">
				<div>
					<h3 class="modal-title">Tour Itinerary Builder</h3>
					<span class="sub-text">Day-wise activities, timings, transfers and meal plans</span>
				</div>
				<button type="button" class="close-btn" onclick={onclose}>✕</button>
			</div>

			<!-- Main Layout: Day Sidebar + Activity Timeline -->
			<div class="modal-body-layout">
				<!-- Day Selector Tabs -->
				<div class="day-sidebar">
					<div class="day-sidebar-title">Days & Routing</div>
					<div class="day-pills-list">
						{#each days as d}
							<button 
								type="button" 
								class="day-btn" 
								class:active={activeDay === d.day}
								onclick={() => (activeDay = d.day)}
							>
								<span class="day-badge">Day {d.day}</span>
								<div class="day-btn-content">
									<span class="day-city">{d.city}</span>
									<span class="day-title-sub">{d.title}</span>
								</div>
							</button>
						{/each}
						<button type="button" class="add-day-btn" onclick={handleAddDay}>
							+ Add Day
						</button>
					</div>
				</div>

				<!-- Day Timeline Editor -->
				<div class="timeline-pane">
					<div class="timeline-header">
						<div>
							<h4 class="day-heading">Day {activeDay} — Pahalgam Expedition</h4>
							<span class="date-chip">📅 {itineraryDate}</span>
						</div>
						<button type="button" class="btn btn-sm btn-outline">
							+ Add Activity
						</button>
					</div>

					<div class="timeline-stream">
						{#each activitiesDay4 as act}
							<div class="timeline-item">
								<div class="time-col">
									<span class="time-text font-mono">{act.time}</span>
									<div class="timeline-dot"></div>
									<div class="timeline-line"></div>
								</div>
								<div class="activity-card">
									<div class="activity-top">
										<div class="icon-title">
											<span class="act-icon">{act.icon}</span>
											<div>
												<span class="act-title">{act.title}</span>
												<span class="act-loc">📍 {act.location}</span>
											</div>
										</div>
										<span class="cat-badge">{act.category}</span>
									</div>
									<p class="act-notes">{act.notes}</p>
								</div>
							</div>
						{/each}
					</div>
				</div>
			</div>

			<!-- Footer Actions -->
			<div class="modal-footer">
				<button type="button" class="btn btn-secondary" onclick={onclose}>Cancel</button>
				<div class="footer-right">
					<button type="button" class="btn btn-outline" onclick={() => toast.success('Draft saved.')}>
						Save as Draft
					</button>
					<button type="button" class="btn btn-primary" onclick={handleSaveItinerary}>
						Save Itinerary
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
		max-width: 860px;
		max-height: 90vh;
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
	.close-btn:hover { color: #0f172a; }

	.modal-body-layout {
		display: grid;
		grid-template-columns: 240px 1fr;
		flex: 1;
		overflow: hidden;
	}

	.day-sidebar {
		background: #f8fafc;
		border-right: 1px solid #e2e8f0;
		padding: 1rem;
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
		overflow-y: auto;
	}

	.day-sidebar-title {
		font-size: 0.75rem;
		font-weight: 700;
		color: #64748b;
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}

	.day-pills-list {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.day-btn {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.65rem 0.75rem;
		border-radius: 8px;
		background: #ffffff;
		border: 1px solid #e2e8f0;
		cursor: pointer;
		text-align: left;
		transition: all 0.12s ease;
	}
	.day-btn:hover { border-color: #cbd5e1; }
	.day-btn.active {
		background: #eff6ff;
		border-color: #bfdbfe;
	}

	.day-badge {
		font-size: 0.7rem;
		font-weight: 800;
		background: #2563eb;
		color: #ffffff;
		padding: 0.2rem 0.45rem;
		border-radius: 4px;
		white-space: nowrap;
	}

	.day-btn-content {
		display: flex;
		flex-direction: column;
		overflow: hidden;
	}

	.day-city { font-size: 0.8125rem; font-weight: 700; color: #0f172a; }
	.day-title-sub { font-size: 0.7rem; color: #64748b; white-space: nowrap; text-overflow: ellipsis; overflow: hidden; }

	.add-day-btn {
		padding: 0.5rem;
		border: 1px dashed #cbd5e1;
		border-radius: 8px;
		background: transparent;
		font-size: 0.8125rem;
		font-weight: 600;
		color: #2563eb;
		cursor: pointer;
	}
	.add-day-btn:hover { background: #f1f5f9; }

	.timeline-pane {
		padding: 1.25rem 1.5rem;
		overflow-y: auto;
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}

	.timeline-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding-bottom: 0.75rem;
		border-bottom: 1px solid #f1f5f9;
	}

	.day-heading {
		font-size: 1.05rem;
		font-weight: 800;
		color: #0f172a;
		margin: 0;
	}

	.date-chip {
		font-size: 0.75rem;
		color: #64748b;
		margin-top: 0.2rem;
		display: inline-block;
	}

	.timeline-stream {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	.timeline-item {
		display: flex;
		gap: 1rem;
		position: relative;
	}

	.time-col {
		width: 75px;
		display: flex;
		flex-direction: column;
		align-items: center;
		position: relative;
	}

	.time-text {
		font-size: 0.75rem;
		font-weight: 700;
		color: #475569;
		margin-bottom: 0.35rem;
	}

	.timeline-dot {
		width: 10px;
		height: 10px;
		border-radius: 50%;
		background: #2563eb;
		border: 2px solid #ffffff;
		box-shadow: 0 0 0 2px #bfdbfe;
		z-index: 1;
	}

	.timeline-line {
		position: absolute;
		top: 25px;
		bottom: -15px;
		width: 2px;
		background: #e2e8f0;
	}

	.activity-card {
		flex: 1;
		background: #f8fafc;
		border: 1px solid #e2e8f0;
		border-radius: 8px;
		padding: 0.85rem 1rem;
		display: flex;
		flex-direction: column;
		gap: 0.35rem;
	}

	.activity-top {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
	}

	.icon-title {
		display: flex;
		gap: 0.5rem;
		align-items: center;
	}

	.act-icon { font-size: 1.25rem; }
	.act-title { font-size: 0.875rem; font-weight: 700; color: #0f172a; display: block; }
	.act-loc { font-size: 0.75rem; color: #64748b; }

	.cat-badge {
		font-size: 0.65rem;
		font-weight: 700;
		background: #eff6ff;
		color: #2563eb;
		padding: 0.15rem 0.45rem;
		border-radius: 4px;
	}

	.act-notes {
		font-size: 0.75rem;
		color: #475569;
		margin: 0;
	}

	.modal-footer {
		padding: 1rem 1.5rem;
		border-top: 1px solid #e2e8f0;
		display: flex;
		justify-content: space-between;
		align-items: center;
		background: #ffffff;
	}

	.footer-right { display: flex; gap: 0.5rem; }

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
	.btn-outline { background: #ffffff; border-color: #cbd5e1; color: #475569; }
	.btn-primary { background: #2563eb; color: #ffffff; }
	.btn-primary:hover { background: #1d4ed8; }
</style>
