<script lang="ts">
	import { toast } from '$lib/stores/toast.svelte';

	interface Props {
		open: boolean;
		booking: any;
		onclose: () => void;
	}

	let { open = false, booking, onclose }: Props = $props();

	let activeViewTab = $state<'preview' | 'whatsapp_text'>('preview');
	let copied = $state(false);

	// Generate WhatsApp message text based on booking & logistics
	let whatsAppMessage = $derived.by(() => {
		if (!booking) return '';

		const modeEmoji = 
			booking.tripMode === 'FAMILY' ? '👨‍👩‍👧‍👦 FAMILY HOLIDAY' :
			booking.tripMode === 'BUSINESS' ? '💼 EXECUTIVE BUSINESS TOUR' :
			booking.tripMode === 'REST_LEISURE' ? '🌴 REST & WELLNESS RETREAT' : '👥 GROUP TOUR DEPARTURE';

		const coordinator = booking.assignedCoordinator || { name: 'Operations Team', phone: '+91 98840 11223' };
		const logistics = booking.logisticsDispatch || {
			driverName: 'Assigned Chauffeur',
			driverPhone: '+91 94190 12345',
			vehicleModel: 'AC Tourist Cab',
			vehiclePlate: 'DL-01-XX-0000',
			pickupLocation: 'Arrival Airport / Station Exit Gate',
			guideName: 'Govt. Licensed Tour Guide',
			guidePhone: '+91 97970 88990',
			guideLanguages: 'English, Hindi, Tamil'
		};
		const emergency = booking.localEmergency || {
			stationLead: 'Local Station Desk',
			emergencyPhone: '+91 94190 99881',
			hospitalPartner: 'Nearest City Hospital'
		};

		return `*✈️ MRBIZ TRAVEL OS — YOUR DIGITAL TRAVEL KIT*
*Booking Ref:* ${booking.id}
*Trip Archetype:* ${modeEmoji}
*Client:* ${booking.customerName} (${booking.customerCity || 'Tamil Nadu'})
*Destination:* ${booking.destination}
*Travel Dates:* ${booking.travelDates} (${booking.paxCount} Travelers)

━━━━━━━━━━━━━━━━━━━━━
🚗 *ON-GROUND TRANSPORT & AIRPORT PICKUP*
• *Cab:* ${logistics.vehicleModel}
• *Plate Number:* *${logistics.vehiclePlate}*
• *Chauffeur:* ${logistics.driverName} (${logistics.driverPhone})
• *Pickup Point:* ${logistics.pickupLocation}
• *Flight Tracking:* ${logistics.flightMonitor || 'Live GPS Tracked'}

━━━━━━━━━━━━━━━━━━━━━
🏨 *CONFIRMED STAY & MEAL PLAN*
• *Hotel:* ${booking.hotelName}
• *Voucher Code:* *${booking.hotelDmcRef}*
• *Special Diet/Needs:* ${booking.specialInstructions || 'Standard meal plan'}

━━━━━━━━━━━━━━━━━━━━━
🎟️ *TRAVEL TICKETS & PASSES*
• *Flight / Train PNR:* *${booking.airlinePnr}*
• *Visa Status:* ${booking.visaStatus} (Stamped & Verified)
• *Entry Passes:* Tickets synced to your booking bar-code.

━━━━━━━━━━━━━━━━━━━━━
👨‍💼 *LOCAL AREA CERTIFIED GUIDE*
• *Guide Name:* ${logistics.guideName}
• *Direct Phone:* ${logistics.guidePhone}
• *Languages:* ${logistics.guideLanguages}

━━━━━━━━━━━━━━━━━━━━━
🆘 *24/7 DEDICATED TRIP COORDINATOR & SAFETY*
• *Trip Manager:* ${coordinator.name} (${coordinator.phone})
• *Local Station Lead:* ${emergency.stationLead} (${emergency.emergencyPhone})
• *Emergency Hospital Partner:* ${emergency.hospitalPartner}

_Wish you a safe, memorable, and seamless journey with MrBiz Global Travel!_
_Download your complete voucher packet from our portal anytime._`;
	});

	function copyToClipboard() {
		if (!whatsAppMessage) return;
		navigator.clipboard.writeText(whatsAppMessage);
		copied = true;
		toast.success('WhatsApp travel kit copied to clipboard!');
		setTimeout(() => {
			copied = false;
		}, 3000);
	}

	function printTravelKit() {
		window.print();
	}
</script>

{#if open && booking}
	<div 
		class="modal-backdrop" 
		role="dialog" 
		aria-modal="true" 
		tabindex="-1"
		onclick={(e) => { if (e.target === e.currentTarget) onclose(); }}
		onkeydown={(e) => { if (e.key === 'Escape') onclose(); }}>
		<div class="travel-kit-modal animate-in">
			<!-- Header -->
			<div class="modal-header">
				<div class="header-main">
					<div class="header-badges">
						<span class="kit-badge">📱 Customer Digital Travel Kit</span>
						<span class="booking-id-tag font-mono">{booking.id}</span>
						<span class="mode-tag mode-{booking.tripMode ? booking.tripMode.toLowerCase() : 'family'}">
							{#if booking.tripMode === 'BUSINESS'}💼 Business Tour
							{:else if booking.tripMode === 'REST_LEISURE'}🌴 Rest & Leisure Mode
							{:else if booking.tripMode === 'GROUP_TOUR'}👥 Group Tour
							{:else}👨‍👩‍👧‍👦 Family Tour{/if}
						</span>
					</div>
					<h2 class="modal-title">{booking.customerName} · {booking.destination}</h2>
					<p class="modal-subtitle">
						Origin: <strong class="text-sky-300">{booking.customerCity || 'Tamil Nadu'}</strong> · 
						Travel Dates: <span class="text-slate-300">{booking.travelDates}</span> · 
						Pax: <span class="text-amber-400 font-bold">{booking.paxCount} Travelers</span>
					</p>
				</div>
				<button class="close-btn" onclick={onclose} aria-label="Close modal">✕</button>
			</div>

			<!-- Tab switcher between Mobile Card Preview & WhatsApp Copy -->
			<div class="view-tabs">
				<button 
					type="button" 
					class="view-tab-btn" 
					class:active={activeViewTab === 'preview'} 
					onclick={() => activeViewTab = 'preview'}>
					📱 Interactive Mobile Travel Pass Preview
				</button>
				<button 
					type="button" 
					class="view-tab-btn" 
					class:active={activeViewTab === 'whatsapp_text'} 
					onclick={() => activeViewTab = 'whatsapp_text'}>
					💬 WhatsApp Text Formatted
				</button>
			</div>

			<!-- Body -->
			<div class="modal-body">
				{#if activeViewTab === 'preview'}
					<!-- Mobile Screen Frame Preview -->
					<div class="mobile-mockup-wrapper">
						<div class="mobile-phone-frame print-area">
							<!-- Phone Top Bar -->
							<div class="phone-status-bar">
								<span class="text-xs">09:41</span>
								<div class="flex items-center gap-1.5 text-xs">
									<span>5G</span>
									<span>100% 🔋</span>
								</div>
							</div>

							<!-- Brand Header -->
							<div class="pass-brand-card">
								<div class="pass-brand-flex">
									<div>
										<div class="pass-brand-name">MRBIZ GLOBAL TRAVEL</div>
										<div class="pass-trip-title">{booking.packageTitle}</div>
									</div>
									<div class="pass-qr-box">
										<div class="qr-code-art">QR-PASS</div>
									</div>
								</div>
								<div class="pass-pnr-strip">
									<div>
										<span class="strip-label">CLIENT ORIGIN</span>
										<div class="strip-val">{booking.customerCity || 'Chennai'}</div>
									</div>
									<div>
										<span class="strip-label">FLIGHT / PNR</span>
										<div class="strip-val font-mono text-amber-300">{booking.airlinePnr}</div>
									</div>
									<div>
										<span class="strip-label">VOUCHER REF</span>
										<div class="strip-val font-mono text-emerald-300">{booking.hotelDmcRef}</div>
									</div>
								</div>
							</div>

							<!-- Pass Modules Stack -->
							<div class="pass-sections-stack">
								<!-- 1. Dedicated On-Ground Chauffeur & Cab -->
								<div class="pass-card transport-card">
									<div class="card-title-row">
										<span class="card-icon">🚗</span>
										<span class="card-heading">DEDICATED ON-GROUND CHAUFFEUR</span>
										<span class="card-status-badge">DISPATCHED ✓</span>
									</div>
									<div class="card-highlight-plate">
										<div class="plate-box font-mono">
											{booking.logisticsDispatch?.vehiclePlate || 'JK-01-AB-4491'}
										</div>
										<div class="vehicle-desc">
											{booking.logisticsDispatch?.vehicleModel || 'Toyota Innova Crysta AC'}
										</div>
									</div>
									<div class="grid-2-cols mt-2">
										<div>
											<span class="info-label">Chauffeur</span>
											<div class="info-val font-semibold">{booking.logisticsDispatch?.driverName || 'Ghulam Nabi'}</div>
										</div>
										<div>
											<span class="info-label">Direct Contact</span>
											<a class="phone-link" href="tel:{booking.logisticsDispatch?.driverPhone || '+919419012345'}">
												📞 {booking.logisticsDispatch?.driverPhone || '+91 94190 12345'}
											</a>
										</div>
									</div>
									<div class="pickup-notice mt-2">
										📍 <strong>Pickup Point:</strong> {booking.logisticsDispatch?.pickupLocation || 'Airport Exit Gate (Name Placard)'}
									</div>
								</div>

								<!-- 2. Hotel & Meal Plan -->
								<div class="pass-card hotel-card">
									<div class="card-title-row">
										<span class="card-icon">🏨</span>
										<span class="card-heading">CONFIRMED RESORT & MEALS</span>
										<span class="hotel-star-badge">4★ / 5★ LUXURY</span>
									</div>
									<div class="font-bold text-white text-base mt-1">{booking.hotelName}</div>
									<div class="flex items-center justify-between text-xs text-slate-300 mt-1">
										<span>Voucher: <strong class="font-mono text-sky-400">{booking.hotelDmcRef}</strong></span>
										<span>Status: <strong class="text-emerald-400">Guaranteed Check-in</strong></span>
									</div>
									<div class="instruction-bubble mt-2">
										🍽️ <strong>Meal & Dietary:</strong> {booking.specialInstructions || 'Daily Breakfast & Dinner included'}
									</div>
								</div>

								<!-- 3. Local Certified Tour Guide -->
								<div class="pass-card guide-card">
									<div class="card-title-row">
										<span class="card-icon">👨‍💼</span>
										<span class="card-heading">LOCAL CERTIFIED TOUR GUIDE</span>
										<span class="guide-lang-badge">TAMIL / ENG</span>
									</div>
									<div class="flex items-center justify-between mt-1">
										<div>
											<div class="font-bold text-white">{booking.logisticsDispatch?.guideName || 'Bashir Ahmed'}</div>
											<div class="text-xs text-slate-400">Languages: {booking.logisticsDispatch?.guideLanguages || 'Tamil, English, Hindi'}</div>
										</div>
										<a class="phone-link text-xs" href="tel:{booking.logisticsDispatch?.guidePhone || '+919797088990'}">
											📞 {booking.logisticsDispatch?.guidePhone || '+91 97970 88990'}
										</a>
									</div>
								</div>

								<!-- 4. 24/7 Coordinator & Local Emergency Station -->
								<div class="pass-card emergency-card">
									<div class="card-title-row">
										<span class="card-icon">🆘</span>
										<span class="card-heading">24/7 LOCAL EMERGENCY STATION</span>
										<span class="sos-badge">STATION HOTLINE</span>
									</div>
									<div class="grid-2-cols mt-1">
										<div>
											<span class="info-label">Assigned Coordinator</span>
											<div class="info-val">{booking.assignedCoordinator?.name || 'Kavitha R.'}</div>
											<div class="text-xs text-sky-400">{booking.assignedCoordinator?.phone || '+91 98840 11223'}</div>
										</div>
										<div>
											<span class="info-label">Local Destination Lead</span>
											<div class="info-val">{booking.localEmergency?.stationLead || 'Station Chief Khurshid'}</div>
											<div class="text-xs text-rose-400">{booking.localEmergency?.emergencyPhone || '+91 94190 99881'}</div>
										</div>
									</div>
									<div class="hospital-sub mt-2">
										🏥 Partner Hospital: {booking.localEmergency?.hospitalPartner || 'SMHS Hospital (24h Trauma Center)'}
									</div>
								</div>
							</div>
						</div>
					</div>
				{:else}
					<!-- WhatsApp Raw Formatted Text Tab -->
					<div class="whatsapp-tab-pane">
						<div class="whatsapp-pane-intro">
							<div class="flex items-center gap-2">
								<span class="text-2xl">💬</span>
								<div>
									<h4 class="font-bold text-white">One-Click Client Dispatch via WhatsApp</h4>
									<p class="text-xs text-slate-400">
										Formatted with markdown asterisks and clear section dividers. Directly share with 
										<span class="text-emerald-400 font-bold">{booking.customerPhone}</span>.
									</p>
								</div>
							</div>
							<div class="flex items-center gap-2">
								<button 
									type="button" 
									class="btn btn-secondary btn-sm" 
									onclick={copyToClipboard}>
									{copied ? '✓ Copied!' : '📋 Copy Text'}
								</button>
								<a 
									class="btn btn-whatsapp btn-sm" 
									href="https://wa.me/{booking.customerPhone.replace(/[^0-9]/g, '')}?text={encodeURIComponent(whatsAppMessage)}" 
									target="_blank" 
									rel="noreferrer">
									🚀 Open WhatsApp
								</a>
							</div>
						</div>

						<pre class="whatsapp-pre font-mono">{whatsAppMessage}</pre>
					</div>
				{/if}
			</div>

			<!-- Modal Footer -->
			<div class="modal-footer">
				<div class="footer-left">
					<button type="button" class="btn btn-outline" onclick={printTravelKit}>
						🖨️ Print / Download PDF Voucher
					</button>
					<button type="button" class="btn btn-secondary" onclick={copyToClipboard}>
						{copied ? '✓ Copied to Clipboard!' : '📋 Copy WhatsApp Message'}
					</button>
				</div>
				<div class="footer-right">
					<a 
						class="btn btn-whatsapp" 
						href="https://wa.me/{booking.customerPhone.replace(/[^0-9]/g, '')}?text={encodeURIComponent(whatsAppMessage)}" 
						target="_blank" 
						rel="noreferrer">
						💬 Send Direct to {booking.customerName}
					</a>
					<button type="button" class="btn btn-secondary" onclick={onclose}>
						Close
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
		background: rgba(15, 23, 42, 0.65);
		backdrop-filter: blur(6px);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 9999;
		padding: 1rem;
		overflow-y: auto;
	}

	.travel-kit-modal {
		background: #ffffff;
		border: 1px solid #e2e8f0;
		border-radius: 1rem;
		width: 100%;
		max-width: 860px;
		max-height: 94vh;
		display: flex;
		flex-direction: column;
		box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
		overflow: hidden;
		color: #0f172a;
	}

	.animate-in {
		animation: modalFadeIn 0.2s cubic-bezier(0.16, 1, 0.3, 1);
	}

	@keyframes modalFadeIn {
		from {
			opacity: 0;
			transform: scale(0.97) translateY(8px);
		}
		to {
			opacity: 1;
			transform: scale(1) translateY(0);
		}
	}

	.modal-header {
		padding: 1.25rem 1.5rem;
		border-bottom: 1px solid #e2e8f0;
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		background: #ffffff;
	}

	.header-badges {
		display: flex;
		align-items: center;
		flex-wrap: wrap;
		gap: 0.5rem;
		margin-bottom: 0.35rem;
	}

	.kit-badge {
		font-size: 0.75rem;
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		background: #eff6ff;
		color: #2563eb;
		border: 1px solid #bfdbfe;
		padding: 0.2rem 0.6rem;
		border-radius: 9999px;
	}

	.booking-id-tag {
		font-size: 0.75rem;
		background: #f1f5f9;
		color: #334155;
		font-family: monospace;
		font-weight: 700;
		padding: 0.2rem 0.5rem;
		border-radius: 0.25rem;
		border: 1px solid #e2e8f0;
	}

	.mode-tag {
		font-size: 0.75rem;
		font-weight: 600;
		padding: 0.2rem 0.6rem;
		border-radius: 9999px;
	}
	.mode-family { background: #fef3c7; color: #92400e; border: 1px solid #fde68a; }
	.mode-business { background: #eff6ff; color: #1e40af; border: 1px solid #bfdbfe; }
	.mode-rest_leisure { background: #ecfdf5; color: #065f46; border: 1px solid #a7f3d0; }
	.mode-group_tour { background: #f5f3ff; color: #5b21b6; border: 1px solid #ddd6fe; }

	.modal-title {
		font-size: 1.25rem;
		font-weight: 800;
		color: #0f172a;
		margin: 0;
	}

	.modal-subtitle {
		font-size: 0.8125rem;
		color: #64748b;
		margin: 0.25rem 0 0 0;
	}

	.close-btn {
		background: transparent;
		border: none;
		color: #64748b;
		font-size: 1.25rem;
		cursor: pointer;
		padding: 0.25rem;
		line-height: 1;
		border-radius: 0.25rem;
	}
	.close-btn:hover {
		color: #0f172a;
		background: #f1f5f9;
	}

	/* Tabs */
	.view-tabs {
		display: flex;
		background: #f8fafc;
		border-bottom: 1px solid #e2e8f0;
		padding: 0 1.5rem;
	}

	.view-tab-btn {
		background: transparent;
		border: none;
		border-bottom: 2px solid transparent;
		color: #64748b;
		padding: 0.75rem 1rem;
		font-size: 0.875rem;
		font-weight: 600;
		cursor: pointer;
		transition: all 0.15s ease;
	}
	.view-tab-btn:hover {
		color: #0f172a;
	}
	.view-tab-btn.active {
		color: #2563eb;
		border-bottom-color: #2563eb;
		font-weight: 700;
	}

	.modal-body {
		padding: 1.5rem;
		overflow-y: auto;
		flex: 1;
		background: #f8fafc;
	}

	/* Mobile Mockup Styles */
	.mobile-mockup-wrapper {
		display: flex;
		justify-content: center;
	}

	.mobile-phone-frame {
		width: 100%;
		max-width: 480px;
		background: #ffffff;
		border: 1px solid #cbd5e1;
		border-radius: 2rem;
		padding: 1.25rem;
		box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 8px 10px -6px rgba(0, 0, 0, 0.05);
	}

	.phone-status-bar {
		display: flex;
		justify-content: space-between;
		align-items: center;
		color: #64748b;
		padding: 0 0.5rem 0.75rem 0.5rem;
		border-bottom: 1px solid #e2e8f0;
	}

	.pass-brand-card {
		background: linear-gradient(135deg, #1e40af, #2563eb);
		border-radius: 1rem;
		padding: 1rem;
		margin-top: 0.75rem;
		color: #ffffff;
	}

	.pass-brand-flex {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		gap: 0.5rem;
	}

	.pass-brand-name {
		font-size: 0.6875rem;
		font-weight: 800;
		letter-spacing: 0.08em;
		color: #bfdbfe;
	}

	.pass-trip-title {
		font-size: 0.9375rem;
		font-weight: 700;
		color: #ffffff;
		margin-top: 0.2rem;
	}

	.pass-qr-box {
		background: #ffffff;
		padding: 0.35rem;
		border-radius: 0.5rem;
		text-align: center;
	}

	.qr-code-art {
		font-size: 0.625rem;
		font-weight: 900;
		color: #000000;
		letter-spacing: 0.05em;
	}

	.pass-pnr-strip {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 0.5rem;
		margin-top: 0.75rem;
		padding-top: 0.75rem;
		border-top: 1px dashed rgba(255, 255, 255, 0.3);
	}

	.strip-label {
		display: block;
		font-size: 0.625rem;
		color: #bfdbfe;
		font-weight: 600;
		text-transform: uppercase;
	}

	.strip-val {
		font-size: 0.8125rem;
		font-weight: 700;
		color: #ffffff;
		margin-top: 0.1rem;
	}

	/* Pass Section Cards */
	.pass-sections-stack {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
		margin-top: 0.875rem;
	}

	.pass-card {
		background: #ffffff;
		border: 1px solid #e2e8f0;
		border-radius: 0.875rem;
		padding: 0.875rem;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.03);
	}

	.card-title-row {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		margin-bottom: 0.5rem;
	}

	.card-icon {
		font-size: 1.125rem;
	}

	.card-heading {
		font-size: 0.6875rem;
		font-weight: 800;
		letter-spacing: 0.05em;
		color: #475569;
		flex: 1;
	}

	.card-status-badge {
		font-size: 0.625rem;
		font-weight: 700;
		background: #ecfdf5;
		color: #059669;
		border: 1px solid #a7f3d0;
		padding: 0.15rem 0.4rem;
		border-radius: 9999px;
	}

	.hotel-star-badge {
		font-size: 0.625rem;
		font-weight: 700;
		background: #eff6ff;
		color: #2563eb;
		border: 1px solid #bfdbfe;
		padding: 0.15rem 0.4rem;
		border-radius: 9999px;
	}

	.guide-lang-badge {
		font-size: 0.625rem;
		font-weight: 700;
		background: #f5f3ff;
		color: #6d28d9;
		border: 1px solid #ddd6fe;
		padding: 0.15rem 0.4rem;
		border-radius: 9999px;
	}

	.sos-badge {
		font-size: 0.625rem;
		font-weight: 700;
		background: #fff1f2;
		color: #e11d48;
		border: 1px solid #fecdd3;
		padding: 0.15rem 0.4rem;
		border-radius: 9999px;
	}

	.card-highlight-plate {
		background: #f8fafc;
		border: 1px dashed #cbd5e1;
		border-radius: 0.5rem;
		padding: 0.5rem 0.75rem;
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.plate-box {
		font-size: 1.0625rem;
		font-weight: 900;
		color: #0f172a;
		letter-spacing: 0.08em;
	}

	.vehicle-desc {
		font-size: 0.75rem;
		color: #64748b;
	}

	.grid-2-cols {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 0.75rem;
	}

	.info-label {
		display: block;
		font-size: 0.625rem;
		color: #64748b;
		text-transform: uppercase;
		font-weight: 600;
	}

	.info-val {
		font-size: 0.8125rem;
		color: #0f172a;
		margin-top: 0.1rem;
	}

	.phone-link {
		color: #2563eb;
		text-decoration: none;
		font-weight: 600;
		font-size: 0.8125rem;
		display: inline-block;
		margin-top: 0.1rem;
	}
	.phone-link:hover {
		text-decoration: underline;
	}

	.pickup-notice {
		background: #eff6ff;
		border-left: 3px solid #2563eb;
		padding: 0.35rem 0.5rem;
		border-radius: 0 0.35rem 0.35rem 0;
		font-size: 0.75rem;
		color: #1e40af;
	}

	.instruction-bubble {
		background: #ecfdf5;
		border-left: 3px solid #10b981;
		padding: 0.35rem 0.5rem;
		border-radius: 0 0.35rem 0.35rem 0;
		font-size: 0.75rem;
		color: #065f46;
	}

	.hospital-sub {
		font-size: 0.6875rem;
		color: #e11d48;
		background: #fff1f2;
		border: 1px solid #fecdd3;
		padding: 0.35rem 0.5rem;
		border-radius: 0.35rem;
	}

	/* WhatsApp Formatted Text Pane */
	.whatsapp-tab-pane {
		background: #ffffff;
		border: 1px solid #e2e8f0;
		border-radius: 0.75rem;
		padding: 1.25rem;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.03);
	}

	.whatsapp-pane-intro {
		display: flex;
		justify-content: space-between;
		align-items: center;
		flex-wrap: wrap;
		gap: 0.75rem;
		padding-bottom: 0.75rem;
		border-bottom: 1px solid #f1f5f9;
		margin-bottom: 1rem;
	}

	.whatsapp-pre {
		background: #f8fafc;
		border: 1px solid #cbd5e1;
		border-radius: 0.5rem;
		padding: 1.25rem;
		color: #0f172a;
		font-family: monospace;
		font-size: 0.8125rem;
		line-height: 1.6;
		white-space: pre-wrap;
		word-break: break-word;
		max-height: 480px;
		overflow-y: auto;
	}

	/* Footer */
	.modal-footer {
		padding: 1rem 1.5rem;
		background: #ffffff;
		border-top: 1px solid #e2e8f0;
		display: flex;
		justify-content: space-between;
		align-items: center;
		flex-wrap: wrap;
		gap: 0.75rem;
	}

	.footer-left, .footer-right {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		flex-wrap: wrap;
	}

	.btn {
		font-family: inherit;
		font-size: 0.8125rem;
		font-weight: 600;
		padding: 0.5rem 0.875rem;
		border-radius: 0.5rem;
		cursor: pointer;
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
		text-decoration: none;
		transition: all 0.15s ease;
		border: 1px solid transparent;
	}

	.btn-sm {
		padding: 0.35rem 0.65rem;
		font-size: 0.75rem;
	}

	.btn-secondary {
		background: #f1f5f9;
		color: #334155;
		border-color: #cbd5e1;
	}
	.btn-secondary:hover {
		background: #e2e8f0;
		color: #0f172a;
	}

	.btn-outline {
		background: #ffffff;
		border: 1px solid #cbd5e1;
		color: #475569;
	}
	.btn-outline:hover {
		background: #f8fafc;
		border-color: #94a3b8;
	}

	.btn-whatsapp {
		background: #22c55e;
		color: #ffffff;
		border-color: #16a34a;
		font-weight: 700;
	}
	.btn-whatsapp:hover {
		background: #16a34a;
	}

	@media print {
		.modal-backdrop {
			position: static;
			background: none;
			padding: 0;
		}
		.travel-kit-modal {
			border: none;
			box-shadow: none;
			max-width: 100%;
		}
		.modal-header, .view-tabs, .modal-footer, .close-btn {
			display: none;
		}
		.mobile-phone-frame {
			border: 1px solid #000;
			box-shadow: none;
			color: #000;
			background: #fff;
		}
	}
</style>
