<script lang="ts">
	interface Props {
		open: boolean;
		onclose: () => void;
	}

	let { open = false, onclose }: Props = $props();

	let viewMode = $state<'month' | 'week' | 'day'>('month');
	let currentMonth = $state('November 2026');

	let calendarDays = $state([
		{ date: 1, dayName: 'Sun', hasTour: false },
		{ date: 2, dayName: 'Mon', hasTour: false },
		{ date: 3, dayName: 'Tue', hasTour: false },
		{ date: 4, dayName: 'Wed', hasTour: false },
		{ date: 5, dayName: 'Thu', tour: { name: 'Dubai Special (5D/4N)', color: 'amber', span: 5 } },
		{ date: 6, dayName: 'Fri', tourCont: 'Dubai Special' },
		{ date: 7, dayName: 'Sat', tourCont: 'Dubai Special' },
		{ date: 8, dayName: 'Sun', tourCont: 'Dubai Special' },
		{ date: 9, dayName: 'Mon', tourCont: 'Dubai Special' },
		{ date: 10, dayName: 'Tue', hasTour: false },
		{ date: 11, dayName: 'Wed', hasTour: false },
		{ date: 12, dayName: 'Thu', hasTour: false },
		{ date: 13, dayName: 'Fri', hasTour: false },
		{ date: 14, dayName: 'Sat', tour: { name: 'Europe Classic (7D/6N)', color: 'purple', span: 7 } },
		{ date: 15, dayName: 'Sun', tourCont: 'Europe Classic' },
		{ date: 16, dayName: 'Mon', tourCont: 'Europe Classic' },
		{ date: 17, dayName: 'Tue', tourCont: 'Europe Classic' },
		{ date: 18, dayName: 'Wed', tourCont: 'Europe Classic' },
		{ date: 19, dayName: 'Thu', tourCont: 'Europe Classic' },
		{ date: 20, dayName: 'Fri', tourCont: 'Europe Classic' },
		{ date: 21, dayName: 'Sat', hasTour: false },
		{ date: 22, dayName: 'Sun', tour: { name: 'Kashmir Valley (4D/5N)', color: 'emerald', span: 4 } },
		{ date: 23, dayName: 'Mon', tourCont: 'Kashmir Valley' },
		{ date: 24, dayName: 'Tue', tourCont: 'Kashmir Valley' },
		{ date: 25, dayName: 'Wed', tourCont: 'Kashmir Valley' },
		{ date: 26, dayName: 'Thu', hasTour: false },
		{ date: 27, dayName: 'Fri', hasTour: false },
		{ date: 28, dayName: 'Sat', tour: { name: 'Singapore Explorer (5D/4N)', color: 'sky', span: 5 } },
		{ date: 29, dayName: 'Sun', tourCont: 'Singapore Explorer' },
		{ date: 30, dayName: 'Mon', tourCont: 'Singapore Explorer' }
	]);
</script>

{#if open}
	<div class="modal-backdrop" onclick={(e) => { if (e.target === e.currentTarget) onclose(); }}>
		<div class="modal-card animate-in" role="dialog" aria-modal="true">
			<!-- Header -->
			<div class="modal-header">
				<div>
					<h3 class="modal-title">Tour Operations &amp; Departures Calendar</h3>
					<span class="sub-text">Capacity tracking, multi-day scheduled departures and staff deadlines</span>
				</div>
				<button type="button" class="close-btn" onclick={onclose}>✕</button>
			</div>

			<!-- Calendar Bar -->
			<div class="cal-controls-bar">
				<div class="month-selector">
					<button type="button" class="nav-arrow">‹</button>
					<span class="month-name">{currentMonth}</span>
					<button type="button" class="nav-arrow">›</button>
					<button type="button" class="btn btn-sm btn-outline">Today</button>
				</div>

				<div class="view-mode-pills">
					<button 
						type="button" 
						class="v-pill" 
						class:active={viewMode === 'month'}
						onclick={() => (viewMode = 'month')}
					>
						Month
					</button>
					<button 
						type="button" 
						class="v-pill" 
						class:active={viewMode === 'week'}
						onclick={() => (viewMode = 'week')}
					>
						Week
					</button>
					<button 
						type="button" 
						class="v-pill" 
						class:active={viewMode === 'day'}
						onclick={() => (viewMode = 'day')}
					>
						Day
					</button>
				</div>
			</div>

			<!-- Calendar Grid -->
			<div class="cal-grid-body">
				<!-- Day Headers -->
				<div class="cal-week-headers">
					<span>Sun</span><span>Mon</span><span>Tue</span><span>Wed</span><span>Thu</span><span>Fri</span><span>Sat</span>
				</div>

				<!-- Days Cells -->
				<div class="cal-days-matrix">
					{#each calendarDays as d}
						<div class="cal-cell" class:has-tour={d.tour || d.tourCont}>
							<div class="cell-date font-mono">{d.date}</div>
							{#if d.tour}
								<div class="tour-strip strip-{d.tour.color}">
									<span class="tour-strip-text">{d.tour.name}</span>
								</div>
							{:else if d.tourCont}
								<div class="tour-strip-cont">
									<span class="tour-strip-text cont-text">{d.tourCont}</span>
								</div>
							{/if}
						</div>
					{/each}
				</div>
			</div>

			<!-- Legend & Footer -->
			<div class="modal-footer">
				<div class="cal-legend">
					<span class="leg-item"><span class="dot emerald"></span> Kashmir Valley</span>
					<span class="leg-item"><span class="dot amber"></span> Dubai Special</span>
					<span class="leg-item"><span class="dot purple"></span> Europe Classic</span>
					<span class="leg-item"><span class="dot sky"></span> Singapore Explorer</span>
				</div>
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
		max-width: 880px;
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

	.cal-controls-bar {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 0.75rem 1.5rem;
		background: #f8fafc;
		border-bottom: 1px solid #e2e8f0;
	}

	.month-selector {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.nav-arrow {
		width: 28px;
		height: 28px;
		border-radius: 6px;
		border: 1px solid #cbd5e1;
		background: #ffffff;
		cursor: pointer;
		font-weight: 700;
	}

	.month-name {
		font-size: 1rem;
		font-weight: 800;
		color: #0f172a;
		min-width: 130px;
		text-align: center;
	}

	.view-mode-pills {
		display: flex;
		background: #e2e8f0;
		border-radius: 6px;
		padding: 0.2rem;
		gap: 0.2rem;
	}

	.v-pill {
		padding: 0.25rem 0.65rem;
		border: none;
		background: transparent;
		font-size: 0.75rem;
		font-weight: 600;
		color: #64748b;
		border-radius: 4px;
		cursor: pointer;
	}
	.v-pill.active { background: #ffffff; color: #0f172a; font-weight: 700; }

	.cal-grid-body {
		padding: 1.25rem 1.5rem;
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.cal-week-headers {
		display: grid;
		grid-template-columns: repeat(7, 1fr);
		text-align: center;
		font-size: 0.75rem;
		font-weight: 700;
		color: #64748b;
		text-transform: uppercase;
	}

	.cal-days-matrix {
		display: grid;
		grid-template-columns: repeat(7, 1fr);
		gap: 4px;
	}

	.cal-cell {
		background: #f8fafc;
		border: 1px solid #e2e8f0;
		border-radius: 6px;
		min-height: 70px;
		padding: 0.35rem;
		display: flex;
		flex-direction: column;
		gap: 0.2rem;
	}

	.cell-date {
		font-size: 0.75rem;
		font-weight: 700;
		color: #475569;
	}

	.tour-strip {
		padding: 0.2rem 0.35rem;
		border-radius: 4px;
		font-size: 0.65rem;
		font-weight: 700;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.strip-emerald { background: #ecfdf5; color: #059669; border: 1px solid #a7f3d0; }
	.strip-amber { background: #fef3c7; color: #d97706; border: 1px solid #fde68a; }
	.strip-purple { background: #f5f3ff; color: #7c3aed; border: 1px solid #ddd6fe; }
	.strip-sky { background: #eff6ff; color: #0284c7; border: 1px solid #bfdbfe; }

	.tour-strip-cont {
		height: 8px;
		background: #e2e8f0;
		border-radius: 2px;
	}
	.cont-text { display: none; }

	.modal-footer {
		padding: 1rem 1.5rem;
		border-top: 1px solid #e2e8f0;
		display: flex;
		justify-content: space-between;
		align-items: center;
		background: #ffffff;
	}

	.cal-legend {
		display: flex;
		gap: 1rem;
		font-size: 0.75rem;
		color: #64748b;
	}

	.leg-item { display: flex; align-items: center; gap: 0.35rem; }
	.dot { width: 8px; height: 8px; border-radius: 50%; }
	.dot.emerald { background: #10b981; }
	.dot.amber { background: #f59e0b; }
	.dot.purple { background: #8b5cf6; }
	.dot.sky { background: #0ea5e9; }

	.btn {
		font-size: 0.8125rem;
		font-weight: 600;
		padding: 0.5rem 1rem;
		border-radius: 8px;
		cursor: pointer;
		border: 1px solid transparent;
	}
	.btn-sm { padding: 0.25rem 0.55rem; font-size: 0.75rem; }
	.btn-secondary { background: #f1f5f9; color: #334155; }
	.btn-outline { background: #ffffff; border-color: #cbd5e1; color: #475569; }
</style>
