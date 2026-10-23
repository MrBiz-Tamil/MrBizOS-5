<script lang="ts">
	interface Props {
		open: boolean;
		onclose: () => void;
	}

	let { open = false, onclose }: Props = $props();

	let activeTab = $state<'map' | 'route'>('map');

	let waypoints = $state([
		{ id: 'wp-1', name: 'Srinagar (Sheikh ul-Alam Airport)', distance: 'Start', time: '0 min', stopType: 'start' },
		{ id: 'wp-2', name: 'Gulmarg Gondola & Highlands', distance: '55 km', time: '2.0 hrs', stopType: 'waypoint' },
		{ id: 'wp-3', name: 'Pahalgam & Betaab Valley', distance: '95 km', time: '3.0 hrs', stopType: 'waypoint' },
		{ id: 'wp-4', name: 'Sonmarg Golden Meadow Pass', distance: '80 km', time: '2.5 hrs', stopType: 'waypoint' },
		{ id: 'wp-5', name: 'Srinagar Dal Lake Circuit', distance: 'Return', time: 'End', stopType: 'end' }
	]);
</script>

{#if open}
	<div class="modal-backdrop" onclick={(e) => { if (e.target === e.currentTarget) onclose(); }}>
		<div class="modal-card animate-in" role="dialog" aria-modal="true">
			<!-- Header -->
			<div class="modal-header">
				<div>
					<h3 class="modal-title">Trip Map &amp; Route Planner</h3>
					<span class="sub-text">Interactive waypoint routing, pickup stations and driving distances</span>
				</div>
				<button type="button" class="close-btn" onclick={onclose}>✕</button>
			</div>

			<!-- Tabs -->
			<div class="modal-tabs">
				<button 
					type="button" 
					class="tab-btn" 
					class:active={activeTab === 'map'}
					onclick={() => (activeTab = 'map')}
				>
					🗺️ Map View
				</button>
				<button 
					type="button" 
					class="tab-btn" 
					class:active={activeTab === 'route'}
					onclick={() => (activeTab = 'route')}
				>
					📋 Route Details
				</button>
			</div>

			<!-- Body Layout -->
			<div class="modal-body-grid">
				<!-- Left: Waypoints Stop List -->
				<div class="waypoints-col">
					<div class="col-head">
						<span class="head-title">Circuit Stops</span>
						<button type="button" class="btn btn-xs btn-outline">+ Add Stop</button>
					</div>

					<div class="waypoints-list">
						{#each waypoints as wp, idx}
							<div class="waypoint-item">
								<div class="wp-num-col">
									<span class="wp-num" class:start={wp.stopType === 'start'} class:end={wp.stopType === 'end'}>
										{idx + 1}
									</span>
									{#if idx < waypoints.length - 1}
										<div class="wp-connector"></div>
									{/if}
								</div>
								<div class="wp-info">
									<span class="wp-name">{wp.name}</span>
									<span class="wp-meta">{wp.distance} • {wp.time}</span>
								</div>
							</div>
						{/each}
					</div>

					<!-- Route Summary Metric -->
					<div class="route-metric-box mt-3">
						<div class="metric-item">
							<span class="m-label">Total Distance</span>
							<span class="m-val font-mono">230 km</span>
						</div>
						<div class="metric-item">
							<span class="m-label">Est. Driving Time</span>
							<span class="m-val font-mono">7.5 hrs</span>
						</div>
					</div>
				</div>

				<!-- Right: Visual Route Map -->
				<div class="map-view-col">
					<div class="map-canvas-container">
						<!-- Stylized Map Mockup with SVGs & Destination Pins -->
						<svg class="map-svg" viewBox="0 0 500 360" fill="none" xmlns="http://www.w3.org/2000/svg">
							<!-- Mountain & Lake Shapes -->
							<rect width="500" height="360" fill="#E8F4F8" />
							<!-- Terrain features -->
							<path d="M 50 80 Q 150 20 280 60 T 450 40 L 500 360 L 0 360 Z" fill="#D7EBE4" opacity="0.6" />
							<path d="M 120 180 Q 200 130 320 170 T 420 220" stroke="#CBD5E1" stroke-width="8" fill="none" opacity="0.4" />
							
							<!-- River / Water body -->
							<path d="M 80 20 Q 180 140 220 200 T 360 340" stroke="#93C5FD" stroke-width="6" fill="none" stroke-linecap="round" />

							<!-- Curving Route Line -->
							<path d="M 100 240 Q 160 140 230 110 T 340 180 T 260 270" stroke="#2563EB" stroke-width="4" stroke-dasharray="6 4" fill="none" />

							<!-- Pin 1: Srinagar (Start) -->
							<g transform="translate(100, 240)">
								<circle r="12" fill="#2563EB" />
								<circle r="5" fill="#FFFFFF" />
								<text x="18" y="5" font-size="12" font-weight="bold" fill="#0F172A">Srinagar (Gate 1)</text>
							</g>

							<!-- Pin 2: Gulmarg -->
							<g transform="translate(200, 120)">
								<circle r="12" fill="#10B981" />
								<circle r="5" fill="#FFFFFF" />
								<text x="-60" y="-12" font-size="12" font-weight="bold" fill="#0F172A">Gulmarg (55 km)</text>
							</g>

							<!-- Pin 3: Sonmarg -->
							<g transform="translate(350, 110)">
								<circle r="12" fill="#F59E0B" />
								<circle r="5" fill="#FFFFFF" />
								<text x="18" y="5" font-size="12" font-weight="bold" fill="#0F172A">Sonmarg (Pass)</text>
							</g>

							<!-- Pin 4: Pahalgam -->
							<g transform="translate(340, 240)">
								<circle r="12" fill="#8B5CF6" />
								<circle r="5" fill="#FFFFFF" />
								<text x="-90" y="20" font-size="12" font-weight="bold" fill="#0F172A">Pahalgam (Betaab)</text>
							</g>
						</svg>

						<!-- Map Overlay Tag -->
						<div class="map-overlay-card">
							<span class="live-gps-dot"></span>
							<span class="gps-text">Chauffeur GPS Active • Innova JK-01-AB-4491</span>
						</div>
					</div>
				</div>
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
		max-width: 860px;
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
	.tab-btn.active {
		color: #2563eb;
		border-bottom-color: #2563eb;
	}

	.modal-body-grid {
		display: grid;
		grid-template-columns: 320px 1fr;
		flex: 1;
		min-height: 420px;
	}

	.waypoints-col {
		padding: 1.25rem;
		border-right: 1px solid #e2e8f0;
		background: #ffffff;
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	.col-head {
		display: flex;
		justify-content: space-between;
		align-items: center;
	}

	.head-title {
		font-size: 0.8125rem;
		font-weight: 700;
		color: #475569;
		text-transform: uppercase;
	}

	.waypoints-list {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		overflow-y: auto;
	}

	.waypoint-item {
		display: flex;
		gap: 0.75rem;
		font-size: 0.8125rem;
	}

	.wp-num-col {
		display: flex;
		flex-direction: column;
		align-items: center;
		position: relative;
	}

	.wp-num {
		width: 22px;
		height: 22px;
		border-radius: 50%;
		background: #eff6ff;
		color: #2563eb;
		border: 1px solid #bfdbfe;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 0.7rem;
		font-weight: 800;
		z-index: 1;
	}
	.wp-num.start { background: #2563eb; color: #ffffff; }
	.wp-num.end { background: #0f172a; color: #ffffff; }

	.wp-connector {
		width: 2px;
		flex: 1;
		background: #e2e8f0;
		margin: 2px 0;
	}

	.wp-info { display: flex; flex-direction: column; }
	.wp-name { font-weight: 600; color: #0f172a; }
	.wp-meta { font-size: 0.7rem; color: #64748b; }

	.route-metric-box {
		background: #f8fafc;
		border: 1px solid #e2e8f0;
		border-radius: 8px;
		padding: 0.75rem;
		display: flex;
		justify-content: space-between;
	}

	.metric-item { display: flex; flex-direction: column; }
	.m-label { font-size: 0.7rem; color: #64748b; text-transform: uppercase; }
	.m-val { font-size: 0.9375rem; font-weight: 800; color: #0f172a; }

	.map-view-col {
		position: relative;
		background: #e8f4f8;
		overflow: hidden;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.map-canvas-container {
		width: 100%;
		height: 100%;
		position: relative;
	}

	.map-svg {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}

	.map-overlay-card {
		position: absolute;
		bottom: 1rem;
		left: 1rem;
		background: rgba(255, 255, 255, 0.95);
		backdrop-filter: blur(4px);
		border: 1px solid #cbd5e1;
		border-radius: 9999px;
		padding: 0.35rem 0.85rem;
		display: flex;
		align-items: center;
		gap: 0.5rem;
		box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
	}

	.live-gps-dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		background: #10b981;
		box-shadow: 0 0 0 3px rgba(16, 185, 129, 0.3);
	}

	.gps-text { font-size: 0.75rem; font-weight: 600; color: #0f172a; }

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
	.btn-xs { padding: 0.2rem 0.5rem; font-size: 0.7rem; }
	.btn-secondary { background: #f1f5f9; color: #334155; }
	.btn-outline { background: #ffffff; border-color: #cbd5e1; color: #2563eb; }

	.mt-3 { margin-top: 0.75rem; }
</style>
