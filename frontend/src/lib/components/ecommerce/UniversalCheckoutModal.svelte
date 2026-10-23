<script lang="ts">
	import { toast } from '$lib/stores/toast.svelte';

	interface Props {
		isOpen: boolean;
		itemTitle: string;
		itemType: string;
		businessUnit: string;
		price: number;
		taxRate?: number;
		customerName?: string;
		customerPhone?: string;
		customerEmail?: string;
		onSuccess?: (receipt: any) => void;
		onClose: () => void;
	}

	let {
		isOpen = false,
		itemTitle = 'Service Booking',
		itemType = 'SERVICE',
		businessUnit = 'GENERAL',
		price = 1000,
		taxRate = 18,
		customerName = 'Rahul Sharma',
		customerPhone = '+91 98840 11223',
		customerEmail = 'rahul.sharma@example.com',
		onSuccess,
		onClose
	}: Props = $props();

	// Payment State
	let paymentMode = $state<'upi' | 'card' | 'khata' | 'bank'>('upi');
	let isProcessing = $state(false);
	let isCompleted = $state(false);
	let generatedInvoiceId = $state('');
	let qrSecondsLeft = $state(299); // 5 min timer

	// Form details
	let clientName = $state(customerName);
	let clientPhone = $state(customerPhone);
	let clientEmail = $state(customerEmail);
	let clientGstin = $state('');

	// Card details
	let cardNumber = $state('4532 •••• •••• 8821');
	let cardExpiry = $state('08/28');
	let cardCvv = $state('884');

	// Financials
	let taxAmount = $derived(Math.round(price * (taxRate / 100)));
	let grandTotal = $derived(price + taxAmount);

	$effect(() => {
		if (isOpen && paymentMode === 'upi' && !isCompleted) {
			const interval = setInterval(() => {
				if (qrSecondsLeft > 0) qrSecondsLeft--;
			}, 1000);
			return () => clearInterval(interval);
		}
	});

	function formatTimer(s: number) {
		const m = Math.floor(s / 60);
		const sec = s % 60;
		return `${m}:${sec < 10 ? '0' : ''}${sec}`;
	}

	function handleConfirmPayment() {
		isProcessing = true;
		setTimeout(() => {
			generatedInvoiceId = 'INV-' + Math.floor(100000 + Math.random() * 900000);
			isProcessing = false;
			isCompleted = true;
			toast?.success?.('Payment confirmed! Tax invoice generated in 0.04s.');
			if (onSuccess) {
				onSuccess({
					invoiceId: generatedInvoiceId,
					itemTitle,
					amount: grandTotal,
					customerName: clientName,
					customerPhone: clientPhone
				});
			}
		}, 800);
	}

	function simulateAppScan() {
		toast?.info?.('Simulating phone scan from GooglePay / PhonePe...');
		handleConfirmPayment();
	}
</script>

{#if isOpen}
	<div class="checkout-backdrop" onclick={onClose}>
		<div class="checkout-modal" onclick={(e) => e.stopPropagation()}>
			<button type="button" class="btn-close" onclick={onClose}>✕</button>

			{#if !isCompleted}
				<div class="checkout-header">
					<div class="header-unit-badge">{businessUnit} • SECURE CHECKOUT</div>
					<h2 class="header-title">{itemTitle}</h2>
					<div class="item-meta">
						<span class="type-pill">{itemType.replace('_', ' ')}</span>
						<span class="gst-pill">Inclusive of {taxRate}% GST</span>
					</div>
				</div>

				<div class="checkout-grid">
					<!-- LEFT: Payment Mode Selector -->
					<div class="payment-selection-pane">
						<div class="pane-title">Select Payment Method</div>

						<div class="mode-buttons-list">
							<button
								type="button"
								class="mode-btn {paymentMode === 'upi' ? 'active' : ''}"
								onclick={() => (paymentMode = 'upi')}
							>
								<div class="mode-icon upi">⚡</div>
								<div class="mode-text">
									<div class="mode-label">Dynamic UPI QR Code</div>
									<div class="mode-sub">PhonePe, GPay, Paytm, BHIM (Zero Fee)</div>
								</div>
								<span class="rec-badge">Fastest</span>
							</button>

							<button
								type="button"
								class="mode-btn {paymentMode === 'card' ? 'active' : ''}"
								onclick={() => (paymentMode = 'card')}
							>
								<div class="mode-icon card">💳</div>
								<div class="mode-text">
									<div class="mode-label">Credit / Debit Card</div>
									<div class="mode-sub">Visa, Mastercard, RuPay, Amex</div>
								</div>
							</button>

							<button
								type="button"
								class="mode-btn {paymentMode === 'khata' ? 'active' : ''}"
								onclick={() => (paymentMode = 'khata')}
							>
								<div class="mode-icon khata">📜</div>
								<div class="mode-text">
									<div class="mode-label">B2B Agent Khata Credit</div>
									<div class="mode-sub">Net-15 / Net-30 Accredited Partner Credit</div>
								</div>
							</button>

							<button
								type="button"
								class="mode-btn {paymentMode === 'bank' ? 'active' : ''}"
								onclick={() => (paymentMode = 'bank')}
							>
								<div class="mode-icon bank">🏦</div>
								<div class="mode-text">
									<div class="mode-label">Bank RTGS / NEFT / IMPS</div>
									<div class="mode-sub">Direct Corporate Bank Wire Transfer</div>
								</div>
							</button>
						</div>

						<!-- Customer Info Form -->
						<div class="cust-info-form">
							<div class="form-title">Billing Information</div>
							<div class="input-row-2">
								<input type="text" bind:value={clientName} placeholder="Full Name" class="cust-input" />
								<input type="tel" bind:value={clientPhone} placeholder="WhatsApp Phone" class="cust-input" />
							</div>
							<div class="input-row-2 mt-2">
								<input type="email" bind:value={clientEmail} placeholder="Email Address" class="cust-input" />
								<input type="text" bind:value={clientGstin} placeholder="GSTIN (Optional for B2B ITC)" class="cust-input" />
							</div>
						</div>
					</div>

					<!-- RIGHT: Dynamic Pay Interface & Price Summary -->
					<div class="payment-action-pane">
						<!-- UPI VIEW -->
						{#if paymentMode === 'upi'}
							<div class="upi-box">
								<div class="upi-header">
									<span>SCAN WITH ANY UPI APP</span>
									<span class="timer font-mono">⏱️ {formatTimer(qrSecondsLeft)}</span>
								</div>

								<div class="qr-code-wrapper" onclick={simulateAppScan} title="Click to simulate instant mobile scan!">
									<!-- SVG Dynamic QR Graphic -->
									<svg class="qr-svg" viewBox="0 0 100 100" width="160" height="160">
										<rect width="100" height="100" fill="#ffffff" rx="10"/>
										<!-- Corners -->
										<rect x="10" y="10" width="24" height="24" fill="#090d16" rx="4"/>
										<rect x="14" y="14" width="16" height="16" fill="#ffffff" rx="2"/>
										<rect x="18" y="18" width="8" height="8" fill="#0284c7" rx="1"/>

										<rect x="66" y="10" width="24" height="24" fill="#090d16" rx="4"/>
										<rect x="70" y="14" width="16" height="16" fill="#ffffff" rx="2"/>
										<rect x="74" y="18" width="8" height="8" fill="#0284c7" rx="1"/>

										<rect x="10" y="66" width="24" height="24" fill="#090d16" rx="4"/>
										<rect x="14" y="70" width="16" height="16" fill="#ffffff" rx="2"/>
										<rect x="18" y="74" width="8" height="8" fill="#0284c7" rx="1"/>

										<!-- Center Data Blocks -->
										<rect x="42" y="14" width="6" height="18" fill="#090d16"/>
										<rect x="52" y="10" width="8" height="10" fill="#090d16"/>
										<rect x="40" y="40" width="20" height="20" fill="#4f46e5" rx="3"/>
										<rect x="46" y="46" width="8" height="8" fill="#ffffff" rx="1"/>

										<rect x="14" y="42" width="12" height="6" fill="#090d16"/>
										<rect x="70" y="46" width="18" height="8" fill="#090d16"/>
										<rect x="42" y="68" width="14" height="14" fill="#090d16"/>
										<rect x="64" y="68" width="10" height="18" fill="#090d16"/>
										<rect x="78" y="74" width="12" height="12" fill="#090d16"/>
									</svg>
									<span class="qr-tap-hint">⚡ Click QR to simulate instant app scan</span>
								</div>

								<div class="upi-vpa-row">
									<span class="vpa-text font-mono">mrbizpay@hdfcbank</span>
									<button type="button" class="btn-copy-vpa" onclick={() => toast?.info?.('UPI VPA copied to clipboard')}>Copy</button>
								</div>
							</div>
						{:else if paymentMode === 'card'}
							<div class="card-inputs-box">
								<div class="form-label">Card Number</div>
								<input type="text" bind:value={cardNumber} class="card-input font-mono" />
								<div class="card-row-2">
									<div>
										<div class="form-label">Expiry MM/YY</div>
										<input type="text" bind:value={cardExpiry} class="card-input font-mono" />
									</div>
									<div>
										<div class="form-label">CVV</div>
										<input type="password" maxlength="4" bind:value={cardCvv} class="card-input font-mono" />
									</div>
								</div>
								<div class="secure-badge">🔒 256-Bit Encrypted 3D-Secure Transaction</div>
							</div>
						{:else if paymentMode === 'khata'}
							<div class="khata-box">
								<div class="khata-title">B2B Partner Credit Facility</div>
								<div class="khata-metric">
									<span>Accredited Credit Limit:</span>
									<strong style="color: #34d399;">₹3,00,000 (Available: ₹2,35,000)</strong>
								</div>
								<p class="khata-desc">Amount will be debited to your sub-agent ledger. Automated tax invoice will be added to your bi-weekly reconciliation statement.</p>
							</div>
						{:else}
							<div class="bank-box">
								<div class="bank-title">Bank Wire / RTGS Instructions</div>
								<div class="bank-details font-mono">
									<div>Beneficiary: MrBiz Multi-Group OS Pvt Ltd</div>
									<div>Account No: 50200088910291</div>
									<div>IFSC: HDFC0000412 (Current Account)</div>
								</div>
							</div>
						{/if}

						<!-- Fare Summary -->
						<div class="price-summary-card">
							<div class="p-row">
								<span>Item Base Tariff:</span>
								<span>₹{price.toLocaleString('en-IN')}</span>
							</div>
							<div class="p-row">
								<span>Applicable Tax ({taxRate}% GST):</span>
								<span>₹{taxAmount.toLocaleString('en-IN')}</span>
							</div>
							<div class="p-row total">
								<span>Total Payable:</span>
								<span class="total-val">₹{grandTotal.toLocaleString('en-IN')}</span>
							</div>
						</div>

						<button
							type="button"
							class="btn-pay-now"
							onclick={handleConfirmPayment}
							disabled={isProcessing}
						>
							{#if isProcessing}
								<span class="spinner"></span>
								<span>Authorizing Payment...</span>
							{:else}
								<span>Confirm & Pay ₹{grandTotal.toLocaleString('en-IN')}</span>
								<span>➔</span>
							{/if}
						</button>
					</div>
				</div>
			{:else}
				<!-- SUCCESS RECEIPT VIEW -->
				<div class="success-receipt-card">
					<div class="success-badge-icon">🎉</div>
					<div class="success-top-label">PAYMENT AUTHORIZED & INVOICE ISSUED</div>
					<h2 class="success-title">Order Confirmed!</h2>
					<div class="invoice-tag font-mono">TAX INVOICE: {generatedInvoiceId}</div>

					<p class="success-msg">
						Thank you, <strong>{clientName}</strong>! Your transaction for <strong>{itemTitle}</strong> has been cleared.
					</p>

					<div class="receipt-details-grid">
						<div class="r-cell">
							<span class="r-lbl">Amount Paid:</span>
							<strong class="r-val" style="color: #34d399;">₹{grandTotal.toLocaleString('en-IN')}</strong>
						</div>
						<div class="r-cell">
							<span class="r-lbl">Payment Mode:</span>
							<strong class="r-val">{paymentMode.toUpperCase()}</strong>
						</div>
						<div class="r-cell">
							<span class="r-lbl">Business Unit:</span>
							<strong class="r-val">{businessUnit}</strong>
						</div>
						<div class="r-cell">
							<span class="r-lbl">Generation Speed:</span>
							<strong class="r-val" style="color: #38bdf8;">0.04s Instant DC</strong>
						</div>
					</div>

					<div class="success-actions">
						<a
							href="https://wa.me/{clientPhone.replace(/\D/g, '')}?text=Dear%20{encodeURIComponent(clientName)}%2C%20your%20invoice%20{generatedInvoiceId}%20for%20{encodeURIComponent(itemTitle)}%20(Amount%3A%20INR%20{grandTotal})%20is%20CONFIRMED!"
							target="_blank"
							rel="noreferrer"
							class="btn-wa-receipt"
						>
							<span>💬 Send Tax Invoice to WhatsApp</span>
						</a>
						<button type="button" class="btn-print" onclick={() => window.print()}>
							<span>🖨️ Print GST Tax Invoice</span>
						</button>
						<button type="button" class="btn-done" onclick={onClose}>
							<span>Close Checkout</span>
						</button>
					</div>
				</div>
			{/if}
		</div>
	</div>
{/if}

<style>
	.checkout-backdrop {
		position: fixed;
		inset: 0;
		z-index: 3000;
		background: rgba(0, 0, 0, 0.85);
		backdrop-filter: blur(10px);
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 1.5rem;
		font-family: var(--font-sans, system-ui, sans-serif);
	}

	.checkout-modal {
		position: relative;
		width: 100%;
		max-width: 860px;
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.15);
		border-radius: 20px;
		padding: 2rem;
		box-shadow: 0 25px 60px rgba(0, 0, 0, 0.7);
		color: #f8fafc;
		max-height: 90vh;
		overflow-y: auto;
	}

	.btn-close {
		position: absolute;
		top: 1.25rem;
		right: 1.25rem;
		background: rgba(255, 255, 255, 0.08);
		border: none;
		color: #ffffff;
		width: 32px;
		height: 32px;
		border-radius: 50%;
		cursor: pointer;
		font-size: 1rem;
	}

	.checkout-header {
		margin-bottom: 1.5rem;
		padding-bottom: 1rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
	}

	.header-unit-badge {
		font-size: 0.7rem;
		font-weight: 800;
		color: #38bdf8;
		letter-spacing: 0.08em;
		margin-bottom: 0.25rem;
	}

	.header-title {
		font-size: 1.5rem;
		font-weight: 900;
		color: #ffffff;
		margin: 0 0 0.5rem;
	}

	.item-meta {
		display: flex;
		gap: 0.5rem;
	}

	.type-pill {
		font-size: 0.7rem;
		font-weight: 700;
		padding: 0.15rem 0.5rem;
		border-radius: 6px;
		background: rgba(255, 255, 255, 0.08);
		color: #cbd5e1;
	}

	.gst-pill {
		font-size: 0.7rem;
		font-weight: 700;
		padding: 0.15rem 0.5rem;
		border-radius: 6px;
		background: rgba(16, 185, 129, 0.15);
		color: #34d399;
	}

	.checkout-grid {
		display: grid;
		grid-template-columns: 1fr 340px;
		gap: 1.75rem;
	}

	.pane-title {
		font-size: 0.85rem;
		font-weight: 800;
		color: #94a3b8;
		margin-bottom: 0.75rem;
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}

	.mode-buttons-list {
		display: flex;
		flex-direction: column;
		gap: 0.6rem;
		margin-bottom: 1.5rem;
	}

	.mode-btn {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		padding: 0.85rem 1rem;
		border-radius: 12px;
		background: rgba(255, 255, 255, 0.03);
		border: 2px solid rgba(255, 255, 255, 0.08);
		color: #ffffff;
		text-align: left;
		cursor: pointer;
		transition: all 0.2s;
	}

	.mode-btn:hover {
		background: rgba(255, 255, 255, 0.06);
		border-color: rgba(255, 255, 255, 0.15);
	}

	.mode-btn.active {
		border-color: #38bdf8;
		background: rgba(2, 132, 199, 0.12);
	}

	.mode-icon {
		font-size: 1.3rem;
	}

	.mode-text {
		flex: 1;
	}

	.mode-label {
		font-size: 0.88rem;
		font-weight: 700;
		color: #ffffff;
	}

	.mode-sub {
		font-size: 0.72rem;
		color: #94a3b8;
	}

	.rec-badge {
		font-size: 0.65rem;
		font-weight: 800;
		background: #0284c7;
		color: #ffffff;
		padding: 0.2rem 0.45rem;
		border-radius: 999px;
	}

	/* Cust info form */
	.cust-info-form {
		background: rgba(255, 255, 255, 0.02);
		border: 1px solid rgba(255, 255, 255, 0.06);
		border-radius: 12px;
		padding: 1rem;
	}

	.form-title {
		font-size: 0.75rem;
		font-weight: 800;
		color: #64748b;
		margin-bottom: 0.6rem;
		text-transform: uppercase;
	}

	.input-row-2 {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 0.6rem;
	}

	.mt-2 { margin-top: 0.6rem; }

	.cust-input {
		width: 100%;
		box-sizing: border-box;
		padding: 0.6rem 0.8rem;
		background: #090d16;
		border: 1px solid rgba(255, 255, 255, 0.12);
		border-radius: 8px;
		color: #ffffff;
		font-size: 0.82rem;
		outline: none;
	}

	.cust-input:focus { border-color: #38bdf8; }

	/* Payment Action Pane */
	.payment-action-pane {
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}

	.upi-box {
		background: #090d16;
		border: 1px solid rgba(56, 189, 248, 0.3);
		border-radius: 14px;
		padding: 1.25rem;
		text-align: center;
	}

	.upi-header {
		display: flex;
		justify-content: space-between;
		font-size: 0.72rem;
		font-weight: 800;
		color: #38bdf8;
		margin-bottom: 0.85rem;
	}

	.qr-code-wrapper {
		background: #ffffff;
		padding: 0.75rem;
		border-radius: 12px;
		display: inline-block;
		cursor: pointer;
		transition: transform 0.2s;
	}

	.qr-code-wrapper:hover {
		transform: scale(1.03);
	}

	.qr-tap-hint {
		display: block;
		font-size: 0.68rem;
		color: #0284c7;
		font-weight: 700;
		margin-top: 0.35rem;
	}

	.upi-vpa-row {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.5rem;
		margin-top: 0.85rem;
		font-size: 0.78rem;
		color: #cbd5e1;
	}

	.btn-copy-vpa {
		background: rgba(255, 255, 255, 0.08);
		border: none;
		color: #38bdf8;
		padding: 0.2rem 0.5rem;
		border-radius: 4px;
		cursor: pointer;
		font-size: 0.72rem;
	}

	.card-inputs-box, .khata-box, .bank-box {
		background: #090d16;
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: 14px;
		padding: 1.25rem;
	}

	.form-label { font-size: 0.72rem; color: #94a3b8; margin-bottom: 0.25rem; }
	.card-input {
		width: 100%;
		box-sizing: border-box;
		padding: 0.6rem 0.8rem;
		background: rgba(255, 255, 255, 0.04);
		border: 1px solid rgba(255, 255, 255, 0.12);
		border-radius: 8px;
		color: #ffffff;
		font-size: 0.9rem;
		margin-bottom: 0.6rem;
		outline: none;
	}

	.card-row-2 {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 0.6rem;
	}

	.secure-badge {
		font-size: 0.7rem;
		color: #34d399;
		margin-top: 0.4rem;
	}

	.khata-title, .bank-title {
		font-size: 0.95rem;
		font-weight: 800;
		color: #ffffff;
		margin-bottom: 0.5rem;
	}

	.khata-metric {
		font-size: 0.82rem;
		color: #cbd5e1;
		margin-bottom: 0.5rem;
	}

	.khata-desc {
		font-size: 0.75rem;
		color: #94a3b8;
		line-height: 1.4;
		margin: 0;
	}

	.bank-details {
		font-size: 0.76rem;
		color: #38bdf8;
		line-height: 1.6;
	}

	/* Price Summary */
	.price-summary-card {
		background: rgba(255, 255, 255, 0.02);
		border: 1px solid rgba(255, 255, 255, 0.06);
		border-radius: 12px;
		padding: 0.85rem 1rem;
		display: flex;
		flex-direction: column;
		gap: 0.4rem;
	}

	.p-row {
		display: flex;
		justify-content: space-between;
		font-size: 0.8rem;
		color: #94a3b8;
	}

	.p-row.total {
		padding-top: 0.4rem;
		border-top: 1px solid rgba(255, 255, 255, 0.1);
		font-size: 1.1rem;
		font-weight: 900;
		color: #ffffff;
	}

	.total-val { color: #38bdf8; }

	.btn-pay-now {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.5rem;
		padding: 0.9rem;
		border-radius: 12px;
		background: linear-gradient(135deg, #10b981 0%, #059669 100%);
		border: none;
		color: #ffffff;
		font-weight: 800;
		font-size: 0.95rem;
		cursor: pointer;
		box-shadow: 0 4px 16px rgba(16, 185, 129, 0.4);
	}

	.spinner {
		width: 16px;
		height: 16px;
		border: 2px solid rgba(255, 255, 255, 0.3);
		border-top-color: #ffffff;
		border-radius: 50%;
		animation: spin 0.7s linear infinite;
	}

	@keyframes spin { to { transform: rotate(360deg); } }

	/* Success receipt */
	.success-receipt-card {
		text-align: center;
		padding: 1rem 0;
	}

	.success-badge-icon {
		font-size: 3rem;
		margin-bottom: 0.35rem;
	}

	.success-top-label {
		font-size: 0.72rem;
		font-weight: 800;
		color: #34d399;
		letter-spacing: 0.08em;
	}

	.success-title {
		font-size: 1.8rem;
		font-weight: 900;
		color: #ffffff;
		margin: 0.25rem 0 0.5rem;
	}

	.invoice-tag {
		display: inline-block;
		padding: 0.35rem 0.85rem;
		border-radius: 999px;
		background: rgba(56, 189, 248, 0.15);
		border: 1px solid rgba(56, 189, 248, 0.35);
		color: #38bdf8;
		font-size: 0.85rem;
		font-weight: 800;
		margin-bottom: 1rem;
	}

	.success-msg {
		font-size: 0.95rem;
		color: #cbd5e1;
		max-width: 500px;
		margin: 0 auto 1.5rem;
	}

	.receipt-details-grid {
		display: grid;
		grid-template-columns: repeat(4, 1fr);
		gap: 0.75rem;
		max-width: 680px;
		margin: 0 auto 1.5rem;
		text-align: left;
	}

	.r-cell {
		background: #090d16;
		border: 1px solid rgba(255, 255, 255, 0.06);
		border-radius: 10px;
		padding: 0.75rem;
	}

	.r-lbl { font-size: 0.68rem; color: #64748b; display: block; margin-bottom: 0.2rem; }
	.r-val { font-size: 0.85rem; color: #ffffff; }

	.success-actions {
		display: flex;
		flex-direction: column;
		gap: 0.6rem;
		max-width: 440px;
		margin: 0 auto;
	}

	.btn-wa-receipt {
		padding: 0.85rem;
		border-radius: 10px;
		background: #25d366;
		color: #ffffff;
		font-weight: 800;
		text-decoration: none;
		font-size: 0.9rem;
	}

	.btn-print {
		padding: 0.75rem;
		border-radius: 10px;
		background: rgba(255, 255, 255, 0.08);
		border: 1px solid rgba(255, 255, 255, 0.15);
		color: #ffffff;
		font-weight: 700;
		cursor: pointer;
	}

	.btn-done {
		padding: 0.65rem;
		border-radius: 10px;
		background: none;
		border: none;
		color: #94a3b8;
		cursor: pointer;
		font-size: 0.85rem;
	}

	@media (max-width: 768px) {
		.checkout-grid {
			grid-template-columns: 1fr;
		}
		.receipt-details-grid {
			grid-template-columns: 1fr 1fr;
		}
	}
</style>
