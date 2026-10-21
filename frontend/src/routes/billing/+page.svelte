<script lang="ts">
	import { onMount } from 'svelte';
	import { billingGrpc } from '$lib/grpc';

	interface InvoiceItem {
		id?: string;
		productId: string;
		itemDescription: string;
		hsnSac: string;
		quantity: string;
		unitCode: string;
		unitPrice: string;
		discountRate: string;
		taxRate: string;
		taxableAmount?: string;
		cgstAmount?: string;
		sgstAmount?: string;
		igstAmount?: string;
		totalAmount?: string;
	}

	interface Invoice {
		id: string;
		tenantId: string;
		invoiceNumber: string;
		customerId: string;
		customerName: string;
		customerGstin?: string;
		invoiceDate: string;
		dueDate: string;
		status: string;
		subtotal: string;
		discountTotal: string;
		taxTotal: string;
		grandTotal: string;
		amountPaid: string;
		balanceDue: string;
		paymentMode: string;
		supplyState: string;
		isInterState: boolean;
		cgstTotal?: string;
		sgstTotal?: string;
		igstTotal?: string;
		items?: InvoiceItem[];
	}

	interface Receivable {
		customerId: string;
		customerName: string;
		phone: string;
		totalBilled: string;
		totalPaid: string;
		balanceDue: string;
		unpaidInvoicesCount: number;
	}

	interface BillingSummary {
		todaySales: string;
		monthSales: string;
		totalOutstanding: string;
		totalInvoicesCount: number;
		paidInvoicesCount: number;
		unpaidInvoicesCount: number;
		topReceivables: Receivable[];
	}

	let activeTab = $state<'quickbill' | 'invoices' | 'khata'>('quickbill');
	let loading = $state(true);
	let billingOnlyMode = $state(false);

	// Summary KPI metrics
	let summary = $state<BillingSummary>({
		todaySales: '0.00',
		monthSales: '0.00',
		totalOutstanding: '0.00',
		totalInvoicesCount: 0,
		paidInvoicesCount: 0,
		unpaidInvoicesCount: 0,
		topReceivables: []
	});

	let invoices = $state<Invoice[]>([]);
	let statusFilter = $state<string>('ALL');

	// New Quick Bill State
	let customerName = $state('Cash Walk-in Customer');
	let customerPhone = $state('+91 98200 12345');
	let customerGstin = $state('');
	let supplyState = $state('Maharashtra (27)');
	let paymentMode = $state<'CASH' | 'UPI' | 'CARD' | 'CREDIT'>('UPI');
	let tenderedAmount = $state('');
	let billNotes = $state('Thank you for your business!');

	let items = $state<InvoiceItem[]>([
		{
			productId: 'PROD-01',
			itemDescription: 'Organic Cold-Pressed Coconut Oil 1L',
			hsnSac: '1513',
			quantity: '2',
			unitCode: 'BTL',
			unitPrice: '350.00',
			discountRate: '5',
			taxRate: '18'
		},
		{
			productId: 'PROD-02',
			itemDescription: 'Pure Forest Wild Honey 500g',
			hsnSac: '0409',
			quantity: '1',
			unitCode: 'JAR',
			unitPrice: '500.00',
			discountRate: '0',
			taxRate: '18'
		}
	]);

	// Payment Modal State
	let showPayModal = $state(false);
	let selectedInvoiceForPay = $state<Invoice | null>(null);
	let payAmount = $state('');
	let payMode = $state('UPI');
	let payRef = $state('');
	let payMessage = $state<string | null>(null);

	// Action notification banner
	let bannerMessage = $state<string | null>(null);

	// Live calculations for quick bill
	let isInterState = $derived(supplyState !== 'Maharashtra (27)');

	let grossSubtotal = $derived(
		items.reduce((sum, item) => {
			const q = parseFloat(item.quantity) || 0;
			const p = parseFloat(item.unitPrice) || 0;
			return sum + q * p;
		}, 0)
	);

	let totalDiscount = $derived(
		items.reduce((sum, item) => {
			const q = parseFloat(item.quantity) || 0;
			const p = parseFloat(item.unitPrice) || 0;
			const d = parseFloat(item.discountRate) || 0;
			return sum + (q * p * d) / 100.0;
		}, 0)
	);

	let taxableAmount = $derived(grossSubtotal - totalDiscount);

	let totalTax = $derived(
		items.reduce((sum, item) => {
			const q = parseFloat(item.quantity) || 0;
			const p = parseFloat(item.unitPrice) || 0;
			const d = parseFloat(item.discountRate) || 0;
			const lineTaxable = q * p - (q * p * d) / 100.0;
			const t = parseFloat(item.taxRate) || 0;
			return sum + (lineTaxable * t) / 100.0;
		}, 0)
	);

	let grandTotal = $derived(taxableAmount + totalTax);

	let payableAmount = $derived(Math.round(grandTotal));
	let roundOff = $derived((payableAmount - grandTotal).toFixed(2));

	let calculatedChange = $derived.by(() => {
		const tendered = parseFloat(tenderedAmount);
		if (isNaN(tendered) || tendered < payableAmount) return 0;
		return (tendered - payableAmount).toFixed(2);
	});

	let calculatedDue = $derived.by(() => {
		if (paymentMode === 'CREDIT') return payableAmount.toFixed(2);
		const tendered = parseFloat(tenderedAmount);
		if (isNaN(tendered)) return payableAmount.toFixed(2);
		if (tendered >= payableAmount) return '0.00';
		return (payableAmount - tendered).toFixed(2);
	});

	// UPI payment URI
	let upiUri = $derived(
		`upi://pay?pa=mrbiz@upi&pn=MrBizStore&am=${payableAmount}&cu=INR&tn=BillPayment`
	);

	onMount(async () => {
		if (typeof localStorage !== 'undefined') {
			billingOnlyMode = localStorage.getItem('mrbiz_billing_only_mode') === 'true';
		}
		await loadBillingData();
	});

	function toggleBillingOnlyMode() {
		billingOnlyMode = !billingOnlyMode;
		if (typeof localStorage !== 'undefined') {
			localStorage.setItem('mrbiz_billing_only_mode', String(billingOnlyMode));
		}
		bannerMessage = billingOnlyMode
			? '⚡ Focused Billing Mode activated! Non-essential modules hidden.'
			: '🌐 Full Business-OS Suite restored.';
		setTimeout(() => (bannerMessage = null), 4000);
	}

	async function loadBillingData() {
		loading = true;
		try {
			const [invRes, sumRes] = await Promise.all([
				billingGrpc.listInvoices({ tenant_id: 'tenant-1' }),
				billingGrpc.getBillingSummary({ tenant_id: 'tenant-1' })
			]);

			if (invRes && invRes.invoices) {
				invoices = invRes.invoices;
			}
			if (sumRes) {
				summary = {
					todaySales: sumRes.today_sales || '0.00',
					monthSales: sumRes.month_sales || '0.00',
					totalOutstanding: sumRes.total_outstanding || '0.00',
					totalInvoicesCount: sumRes.total_invoices_count || 0,
					paidInvoicesCount: sumRes.paid_invoices_count || 0,
					unpaidInvoicesCount: sumRes.unpaid_invoices_count || 0,
					topReceivables: sumRes.top_receivables || []
				};
			}
		} catch (e: any) {
			console.warn('Billing load error, fallback ready:', e);
		} finally {
			loading = false;
		}
	}

	function addItem() {
		items = [
			...items,
			{
				productId: `PROD-${items.length + 1}`,
				itemDescription: '',
				hsnSac: '9983',
				quantity: '1',
				unitCode: 'PCS',
				unitPrice: '100.00',
				discountRate: '0',
				taxRate: '18'
			}
		];
	}

	function removeItem(index: number) {
		if (items.length > 1) {
			items = items.filter((_, i) => i !== index);
		}
	}

	function setWalkinCustomer() {
		customerName = 'Cash Walk-in Customer';
		customerPhone = '+91 98200 12345';
		customerGstin = '';
		paymentMode = 'CASH';
		tenderedAmount = String(payableAmount);
	}

	async function generateBill() {
		try {
			loading = true;
			const res = await billingGrpc.createInvoice({
				tenant_id: 'tenant-1',
				customer_id: customerName,
				supply_state: supplyState,
				payment_mode: paymentMode,
				tendered_cash: paymentMode === 'CREDIT' ? '0' : tenderedAmount || String(payableAmount),
				notes: billNotes,
				items: items.map((i) => ({
					product_id: i.productId,
					item_description: i.itemDescription || 'General Goods/Service',
					hsn_sac: i.hsnSac,
					quantity: i.quantity,
					unit_code: i.unitCode,
					unit_price: i.unitPrice,
					discount_rate: i.discountRate,
					tax_rate: i.taxRate
				}))
			});

			if (res && res.invoice) {
				bannerMessage = `✅ Bill ${res.invoice.invoice_number} created successfully! Total: ₹${res.invoice.grand_total}`;
				await loadBillingData();
				activeTab = 'invoices';
			}
		} catch (err: any) {
			bannerMessage = `Error creating bill: ${err.message}`;
		} finally {
			loading = false;
			setTimeout(() => (bannerMessage = null), 5000);
		}
	}

	function shareBillOnWhatsApp(inv?: Invoice) {
		const targetInv = inv || {
			invoiceNumber: 'INV-NEW',
			customerName,
			grandTotal: String(payableAmount),
			balanceDue: calculatedDue,
			paymentMode
		};

		const text = `Hello ${targetInv.customerName}, here is your bill *${targetInv.invoiceNumber}* from MrBiz-OS Store.\nTotal Amount: ₹${targetInv.grandTotal}\nBalance Due: ₹${targetInv.balanceDue}\nPayment Link: https://mrbiz.app/pay/${targetInv.invoiceNumber}\nThank you!`;
		const url = `https://wa.me/?text=${encodeURIComponent(text)}`;
		window.open(url, '_blank');
	}

	function printThermalReceipt() {
		window.print();
	}

	function openPaymentModal(inv: Invoice) {
		selectedInvoiceForPay = inv;
		payAmount = inv.balanceDue;
		payMode = 'UPI';
		payRef = 'UPI-' + Date.now().toString().slice(-6);
		payMessage = null;
		showPayModal = true;
	}

	async function submitRecordPayment() {
		if (!selectedInvoiceForPay) return;
		try {
			const res = await billingGrpc.recordPayment({
				tenant_id: 'tenant-1',
				invoice_id: selectedInvoiceForPay.id,
				amount_paid: payAmount,
				payment_mode: payMode,
				reference_number: payRef,
				payment_date: new Date().toISOString().split('T')[0]
			});

			if (res) {
				bannerMessage = res.message || 'Payment recorded successfully!';
				showPayModal = false;
				await loadBillingData();
			}
		} catch (e: any) {
			payMessage = e.message;
		}
	}

	let filteredInvoices = $derived.by(() => {
		if (statusFilter === 'ALL') return invoices;
		return invoices.filter((i) => i.status === statusFilter);
	});
</script>

<svelte:head>
	<title>Express Billing & Invoicing Counter | MrBiz-OS</title>
</svelte:head>

<div class="billing-container">
	<!-- Top Notification Banner -->
	{#if bannerMessage}
		<div class="alert-banner">
			<span>{bannerMessage}</span>
			<button class="banner-close" onclick={() => (bannerMessage = null)}>✕</button>
		</div>
	{/if}

	<!-- Header with Mode Switcher -->
	<header class="billing-header">
		<div class="title-group">
			<div class="title-row">
				<h1>⚡ Express Billing & Quick Invoicing Counter</h1>
				<span class="badge-tag">GST Ready</span>
				{#if billingOnlyMode}
					<span class="badge-mode">Focused Billing Only</span>
				{/if}
			</div>
			<p class="subtitle">
				Ultra-fast 3-second billing. Auto GST calculation, dynamic UPI QR code, 80mm thermal print, and WhatsApp bills.
			</p>
		</div>

		<div class="header-actions">
			<button
				class="mode-toggle-btn"
				class:active={billingOnlyMode}
				onclick={toggleBillingOnlyMode}
				title="Toggle between focused billing-only mode and full enterprise suite"
			>
				{billingOnlyMode ? '⚙️ Switch to Full ERP Suite' : '⚡ Focused Billing-Only Mode'}
			</button>

			<button class="walkin-btn" onclick={setWalkinCustomer}>
				🚶 Cash Walk-in Customer
			</button>
		</div>
	</header>

	<!-- KPI Metric Cards -->
	<div class="metrics-grid">
		<div class="kpi-card sales-today">
			<div class="kpi-icon">💰</div>
			<div class="kpi-info">
				<span class="kpi-label">Today's Sales</span>
				<span class="kpi-value">₹{summary.todaySales}</span>
			</div>
		</div>

		<div class="kpi-card sales-month">
			<div class="kpi-icon">📈</div>
			<div class="kpi-info">
				<span class="kpi-label">This Month's Sales</span>
				<span class="kpi-value">₹{summary.monthSales}</span>
			</div>
		</div>

		<div class="kpi-card outstanding">
			<div class="kpi-icon">⏳</div>
			<div class="kpi-info">
				<span class="kpi-label">Total Outstanding Due</span>
				<span class="kpi-value warning">₹{summary.totalOutstanding}</span>
			</div>
		</div>

		<div class="kpi-card bill-counts">
			<div class="kpi-icon">🧾</div>
			<div class="kpi-info">
				<span class="kpi-label">Invoices Generated</span>
				<span class="kpi-value">
					{summary.paidInvoicesCount} <span class="sub">Paid</span> / {summary.unpaidInvoicesCount} <span class="sub">Pending</span>
				</span>
			</div>
		</div>
	</div>

	<!-- Navigation Tabs -->
	<nav class="billing-tabs" aria-label="Billing Views">
		<button
			class="tab-btn"
			class:active={activeTab === 'quickbill'}
			onclick={() => (activeTab = 'quickbill')}
		>
			⚡ Express Billing Counter
		</button>
		<button
			class="tab-btn"
			class:active={activeTab === 'invoices'}
			onclick={() => (activeTab = 'invoices')}
		>
			📜 All Bills & Invoices ({invoices.length})
		</button>
		<button
			class="tab-btn"
			class:active={activeTab === 'khata'}
			onclick={() => (activeTab = 'khata')}
		>
			👥 Receivables & Customer Khata ({summary.topReceivables.length})
		</button>
	</nav>

	<!-- TAB 1: EXPRESS BILLING COUNTER -->
	{#if activeTab === 'quickbill'}
		<div class="bill-workbench">
			<!-- Left: Customer & Items -->
			<div class="bill-main">
				<!-- Customer Details Card -->
				<div class="card customer-card">
					<div class="card-header">
						<h3>1. Customer Details & Place of Supply</h3>
						<span class="gst-state-badge">
							{isInterState ? 'IGST Applicable (Inter-State)' : 'CGST + SGST (Intra-State)'}
						</span>
					</div>

					<div class="form-row">
						<div class="form-group flex-2">
							<label for="custName">Customer / Company Name</label>
							<input
								id="custName"
								type="text"
								bind:value={customerName}
								placeholder="Enter customer name or select walk-in"
							/>
						</div>

						<div class="form-group flex-1">
							<label for="custPhone">Phone / WhatsApp</label>
							<input
								id="custPhone"
								type="text"
								bind:value={customerPhone}
								placeholder="+91 98..."
							/>
						</div>

						<div class="form-group flex-1">
							<label for="custGstin">GSTIN (Optional)</label>
							<input
								id="custGstin"
								type="text"
								bind:value={customerGstin}
								placeholder="27AABCA..."
							/>
						</div>

						<div class="form-group flex-1">
							<label for="supplyState">State of Supply</label>
							<select id="supplyState" bind:value={supplyState}>
								<option value="Maharashtra (27)">Maharashtra (27)</option>
								<option value="Gujarat (24)">Gujarat (24)</option>
								<option value="Karnataka (29)">Karnataka (29)</option>
								<option value="Delhi (07)">Delhi (07)</option>
								<option value="Tamil Nadu (33)">Tamil Nadu (33)</option>
								<option value="Telangana (36)">Telangana (36)</option>
								<option value="Uttar Pradesh (09)">Uttar Pradesh (09)</option>
							</select>
						</div>
					</div>
				</div>

				<!-- Items Table Card -->
				<div class="card items-card">
					<div class="card-header">
						<h3>2. Items & Services</h3>
						<button class="add-row-btn" onclick={addItem}>+ Add Item Line</button>
					</div>

					<div class="table-responsive">
						<table class="items-table">
							<thead>
								<tr>
									<th style="width: 35%;">Item Description</th>
									<th style="width: 10%;">HSN/SAC</th>
									<th style="width: 10%;">Qty</th>
									<th style="width: 10%;">Unit</th>
									<th style="width: 12%;">Price (₹)</th>
									<th style="width: 8%;">Disc %</th>
									<th style="width: 8%;">GST %</th>
									<th style="width: 12%;">Total (₹)</th>
									<th style="width: 5%;"></th>
								</tr>
							</thead>
							<tbody>
								{#each items as item, index}
									{@const q = parseFloat(item.quantity) || 0}
									{@const p = parseFloat(item.unitPrice) || 0}
									{@const d = parseFloat(item.discountRate) || 0}
									{@const t = parseFloat(item.taxRate) || 0}
									{@const lineTaxable = q * p - (q * p * d) / 100.0}
									{@const lineTotal = lineTaxable + (lineTaxable * t) / 100.0}
									<tr>
										<td>
											<input
												type="text"
												class="table-input"
												bind:value={item.itemDescription}
												placeholder="Item name / barcode scan"
											/>
										</td>
										<td>
											<input
												type="text"
												class="table-input center"
												bind:value={item.hsnSac}
												placeholder="HSN"
											/>
										</td>
										<td>
											<input
												type="number"
												min="1"
												class="table-input center"
												bind:value={item.quantity}
											/>
										</td>
										<td>
											<select class="table-input" bind:value={item.unitCode}>
												<option value="PCS">PCS</option>
												<option value="BOX">BOX</option>
												<option value="KG">KG</option>
												<option value="MTR">MTR</option>
												<option value="BTL">BTL</option>
												<option value="JAR">JAR</option>
											</select>
										</td>
										<td>
											<input
												type="number"
												step="0.01"
												class="table-input right"
												bind:value={item.unitPrice}
											/>
										</td>
										<td>
											<input
												type="number"
												min="0"
												max="100"
												class="table-input center"
												bind:value={item.discountRate}
											/>
										</td>
										<td>
											<select class="table-input center" bind:value={item.taxRate}>
												<option value="0">0%</option>
												<option value="5">5%</option>
												<option value="12">12%</option>
												<option value="18">18%</option>
												<option value="28">28%</option>
											</select>
										</td>
										<td class="line-total-cell right">
											₹{lineTotal.toFixed(2)}
										</td>
										<td class="center">
											<button
												class="del-row-btn"
												disabled={items.length === 1}
												onclick={() => removeItem(index)}
												title="Delete item"
											>
												✕
											</button>
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>

					<div class="table-footer-actions">
						<button class="add-row-btn-ghost" onclick={addItem}>
							+ Add Another Item
						</button>
						<span class="keyboard-hint">Tip: Enter barcode or type to autocomplete</span>
					</div>
				</div>
			</div>

			<!-- Right: Calculation & Instant Checkout -->
			<div class="bill-sidebar">
				<div class="card checkout-card">
					<div class="checkout-header">
						<h3>3. Payment & Settlement</h3>
						<span class="active-mode-badge">{paymentMode}</span>
					</div>

					<!-- Payment Mode Selector -->
					<div class="payment-mode-grid">
						<button
							class="pay-btn"
							class:selected={paymentMode === 'UPI'}
							onclick={() => {
								paymentMode = 'UPI';
								tenderedAmount = String(payableAmount);
							}}
						>
							📱 UPI QR
						</button>
						<button
							class="pay-btn"
							class:selected={paymentMode === 'CASH'}
							onclick={() => {
								paymentMode = 'CASH';
								tenderedAmount = String(payableAmount);
							}}
						>
							💵 Cash
						</button>
						<button
							class="pay-btn"
							class:selected={paymentMode === 'CARD'}
							onclick={() => {
								paymentMode = 'CARD';
								tenderedAmount = String(payableAmount);
							}}
						>
							💳 Card
						</button>
						<button
							class="pay-btn"
							class:selected={paymentMode === 'CREDIT'}
							onclick={() => {
								paymentMode = 'CREDIT';
								tenderedAmount = '0';
							}}
						>
							⏳ Credit / Due
						</button>
					</div>

					<!-- Dynamic UPI QR Scanner Display -->
					{#if paymentMode === 'UPI'}
						<div class="upi-box">
							<div class="upi-qr-wrapper">
								<!-- Interactive QR visual -->
								<div class="qr-placeholder">
									<svg viewBox="0 0 100 100" class="qr-svg">
										<rect x="5" y="5" width="30" height="30" fill="#0f172a" />
										<rect x="10" y="10" width="20" height="20" fill="#ffffff" />
										<rect x="14" y="14" width="12" height="12" fill="#0f172a" />

										<rect x="65" y="5" width="30" height="30" fill="#0f172a" />
										<rect x="70" y="10" width="20" height="20" fill="#ffffff" />
										<rect x="74" y="14" width="12" height="12" fill="#0f172a" />

										<rect x="5" y="65" width="30" height="30" fill="#0f172a" />
										<rect x="10" y="70" width="20" height="20" fill="#ffffff" />
										<rect x="14" y="74" width="12" height="12" fill="#0f172a" />

										<rect x="42" y="10" width="16" height="80" fill="#38bdf8" />
										<rect x="10" y="42" width="80" height="16" fill="#38bdf8" />
									</svg>
								</div>
								<div class="upi-text">
									<strong>Scan to Pay ₹{payableAmount}</strong>
									<span class="upi-id">UPI ID: mrbiz@upi</span>
								</div>
							</div>
						</div>
					{/if}

					<!-- Settlement Input -->
					<div class="tendered-group">
						<label for="tenderedCash">Amount Received (₹)</label>
						<input
							id="tenderedCash"
							type="number"
							step="1"
							bind:value={tenderedAmount}
							placeholder={String(payableAmount)}
						/>
					</div>

					<div class="balance-calc-row">
						{#if paymentMode !== 'CREDIT' && parseFloat(tenderedAmount) > payableAmount}
							<div class="change-tag return">
								<span>Change to Return:</span>
								<strong>₹{calculatedChange}</strong>
							</div>
						{:else if calculatedDue !== '0.00'}
							<div class="change-tag due">
								<span>Balance Remaining:</span>
								<strong>₹{calculatedDue}</strong>
							</div>
						{:else}
							<div class="change-tag full">
								<span>Payment Status:</span>
								<strong>Fully Settled</strong>
							</div>
						{/if}
					</div>

					<!-- Bill Breakdown Summary -->
					<div class="summary-breakdown">
						<div class="sum-row">
							<span>Gross Subtotal</span>
							<span>₹{grossSubtotal.toFixed(2)}</span>
						</div>
						<div class="sum-row">
							<span>Total Discount</span>
							<span class="disc">-₹{totalDiscount.toFixed(2)}</span>
						</div>
						<div class="sum-row">
							<span>Taxable Value</span>
							<span>₹{taxableAmount.toFixed(2)}</span>
						</div>
						{#if isInterState}
							<div class="sum-row">
								<span>IGST (Integrated Tax)</span>
								<span>₹{totalTax.toFixed(2)}</span>
							</div>
						{:else}
							<div class="sum-row">
								<span>CGST (Central Tax)</span>
								<span>₹{(totalTax / 2).toFixed(2)}</span>
							</div>
							<div class="sum-row">
								<span>SGST (State Tax)</span>
								<span>₹{(totalTax / 2).toFixed(2)}</span>
							</div>
						{/if}
						<div class="sum-row">
							<span>Round Off</span>
							<span>₹{roundOff}</span>
						</div>

						<div class="sum-total-row">
							<span>Net Payable</span>
							<span class="big-total">₹{payableAmount}</span>
						</div>
					</div>

					<!-- Fast Execution Action Buttons -->
					<div class="checkout-actions">
						<button class="primary-checkout-btn" disabled={loading} onclick={generateBill}>
							⚡ Generate & Save Bill
						</button>

						<div class="action-btn-row">
							<button class="secondary-btn" onclick={printThermalReceipt}>
								🖨️ 80mm Print
							</button>
							<button class="whatsapp-btn" onclick={() => shareBillOnWhatsApp()}>
								💬 WhatsApp Bill
							</button>
						</div>
					</div>
				</div>
			</div>
		</div>
	{/if}

	<!-- TAB 2: INVOICES & PAYMENT RECEIPTS -->
	{#if activeTab === 'invoices'}
		<div class="invoices-view">
			<div class="invoices-toolbar">
				<div class="filter-group">
					<button
						class="filter-pill"
						class:active={statusFilter === 'ALL'}
						onclick={() => (statusFilter = 'ALL')}
					>
						All ({invoices.length})
					</button>
					<button
						class="filter-pill"
						class:active={statusFilter === 'PAID'}
						onclick={() => (statusFilter = 'PAID')}
					>
						Paid
					</button>
					<button
						class="filter-pill"
						class:active={statusFilter === 'PARTIAL'}
						onclick={() => (statusFilter = 'PARTIAL')}
					>
						Partial
					</button>
					<button
						class="filter-pill"
						class:active={statusFilter === 'UNPAID'}
						onclick={() => (statusFilter = 'UNPAID')}
					>
						Unpaid
					</button>
					<button
						class="filter-pill"
						class:active={statusFilter === 'OVERDUE'}
						onclick={() => (statusFilter = 'OVERDUE')}
					>
						Overdue
					</button>
				</div>

				<button class="new-bill-shortcut" onclick={() => (activeTab = 'quickbill')}>
					+ New Quick Bill
				</button>
			</div>

			<div class="card table-card">
				<div class="table-responsive">
					<table class="data-table">
						<thead>
							<tr>
								<th>Bill #</th>
								<th>Customer</th>
								<th>Date</th>
								<th>Total Amount</th>
								<th>Paid</th>
								<th>Balance Due</th>
								<th>Status</th>
								<th>Mode</th>
								<th>Actions</th>
							</tr>
						</thead>
						<tbody>
							{#each filteredInvoices as inv}
								<tr>
									<td class="bold">{inv.invoiceNumber}</td>
									<td>
										<div class="cust-cell">
											<span class="cust-title">{inv.customerName}</span>
											{#if inv.customerGstin}
												<span class="cust-gstin">{inv.customerGstin}</span>
											{/if}
										</div>
									</td>
									<td>{inv.invoiceDate}</td>
									<td class="bold">₹{inv.grandTotal}</td>
									<td class="green">₹{inv.amountPaid}</td>
									<td class={parseFloat(inv.balanceDue) > 0 ? 'red bold' : 'muted'}>
										₹{inv.balanceDue}
									</td>
									<td>
										<span class="status-badge {inv.status.toLowerCase()}">
											{inv.status}
										</span>
									</td>
									<td>{inv.paymentMode}</td>
									<td>
										<div class="actions-cell">
											{#if parseFloat(inv.balanceDue) > 0}
												<button
													class="record-pay-btn"
													onclick={() => openPaymentModal(inv)}
													title="Record Payment"
												>
													💳 Record Pay
												</button>
											{/if}
											<button
												class="icon-action-btn"
												onclick={() => shareBillOnWhatsApp(inv)}
												title="Share on WhatsApp"
											>
												💬
											</button>
											<button
												class="icon-action-btn"
												onclick={printThermalReceipt}
												title="Print PDF Receipt"
											>
												🖨️
											</button>
										</div>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</div>
		</div>
	{/if}

	<!-- TAB 3: CUSTOMER RECEIVABLES & KHATA -->
	{#if activeTab === 'khata'}
		<div class="khata-view">
			<div class="card table-card">
				<div class="card-header">
					<h3>Customer Outstanding Khata Ledger</h3>
					<span class="sub-text">Real-time balances across customers</span>
				</div>

				<div class="table-responsive">
					<table class="data-table">
						<thead>
							<tr>
								<th>Customer Name</th>
								<th>Phone</th>
								<th>Total Invoiced</th>
								<th>Total Collected</th>
								<th>Outstanding Balance Due</th>
								<th>Unpaid Bills</th>
								<th>Payment Reminder</th>
							</tr>
						</thead>
						<tbody>
							{#each summary.topReceivables as rec}
								<tr>
									<td class="bold">{rec.customerName}</td>
									<td>{rec.phone}</td>
									<td>₹{rec.totalBilled}</td>
									<td class="green">₹{rec.totalPaid}</td>
									<td class="red bold">₹{rec.balanceDue}</td>
									<td>
										<span class="unpaid-badge">{rec.unpaidInvoicesCount} Bills</span>
									</td>
									<td>
										<button
											class="whatsapp-nudge-btn"
											onclick={() => {
												const msg = `Dear ${rec.customerName}, a gentle reminder that your pending balance of ₹${rec.balanceDue} is due. You can pay via UPI: mrbiz@upi. Thank you!`;
												window.open(`https://wa.me/?text=${encodeURIComponent(msg)}`, '_blank');
											}}
										>
											💬 Send WhatsApp Reminder
										</button>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</div>
		</div>
	{/if}

	<!-- Record Payment Modal -->
	{#if showPayModal && selectedInvoiceForPay}
		<div class="modal-backdrop">
			<div class="modal-card">
				<div class="modal-header">
					<h3>Record Payment for {selectedInvoiceForPay.invoiceNumber}</h3>
					<button class="modal-close" onclick={() => (showPayModal = false)}>✕</button>
				</div>

				{#if payMessage}
					<div class="modal-error">{payMessage}</div>
				{/if}

				<div class="modal-body">
					<div class="modal-info-row">
						<span>Customer:</span>
						<strong>{selectedInvoiceForPay.customerName}</strong>
					</div>
					<div class="modal-info-row">
						<span>Total Bill:</span>
						<strong>₹{selectedInvoiceForPay.grandTotal}</strong>
					</div>
					<div class="modal-info-row">
						<span>Balance Due:</span>
						<strong class="red">₹{selectedInvoiceForPay.balanceDue}</strong>
					</div>

					<div class="form-group" style="margin-top: 1rem;">
						<label for="payAmount">Amount Paid (₹)</label>
						<input id="payAmount" type="number" bind:value={payAmount} />
					</div>

					<div class="form-group">
						<label for="payMode">Payment Mode</label>
						<select id="payMode" bind:value={payMode}>
							<option value="UPI">UPI</option>
							<option value="CASH">Cash</option>
							<option value="CARD">Debit / Credit Card</option>
							<option value="NETBANKING">Net Banking</option>
							<option value="CHEQUE">Cheque</option>
						</select>
					</div>

					<div class="form-group">
						<label for="payRef">Reference / Transaction ID</label>
						<input id="payRef" type="text" bind:value={payRef} placeholder="UTR / Ref Number" />
					</div>
				</div>

				<div class="modal-footer">
					<button class="secondary-btn" onclick={() => (showPayModal = false)}>Cancel</button>
					<button class="primary-checkout-btn" onclick={submitRecordPayment}>
						Confirm Payment
					</button>
				</div>
			</div>
		</div>
	{/if}
</div>

<style>
	.billing-container {
		padding: 1.5rem;
		max-width: 1400px;
		margin: 0 auto;
		color: #e2e8f0;
	}

	.alert-banner {
		background: #065f46;
		color: #a7f3d0;
		padding: 0.75rem 1rem;
		border-radius: 8px;
		margin-bottom: 1rem;
		display: flex;
		justify-content: space-between;
		align-items: center;
		font-weight: 600;
	}

	.banner-close {
		background: transparent;
		border: none;
		color: inherit;
		cursor: pointer;
		font-size: 1rem;
	}

	.billing-header {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		margin-bottom: 1.5rem;
		gap: 1rem;
		flex-wrap: wrap;
	}

	.title-row {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}

	h1 {
		font-size: 1.625rem;
		font-weight: 800;
		color: #ffffff;
		margin: 0;
	}

	.badge-tag {
		background: rgba(56, 189, 248, 0.15);
		color: #38bdf8;
		padding: 0.2rem 0.6rem;
		border-radius: 6px;
		font-size: 0.75rem;
		font-weight: 700;
	}

	.badge-mode {
		background: rgba(16, 185, 129, 0.15);
		color: #34d399;
		padding: 0.2rem 0.6rem;
		border-radius: 6px;
		font-size: 0.75rem;
		font-weight: 700;
	}

	.subtitle {
		color: #94a3b8;
		font-size: 0.875rem;
		margin-top: 0.25rem;
	}

	.header-actions {
		display: flex;
		gap: 0.75rem;
		align-items: center;
	}

	.mode-toggle-btn {
		background: #1e293b;
		color: #94a3b8;
		border: 1px solid #334155;
		padding: 0.5rem 1rem;
		border-radius: 8px;
		font-weight: 600;
		cursor: pointer;
		transition: all 0.2s;
	}

	.mode-toggle-btn.active {
		background: rgba(16, 185, 129, 0.15);
		color: #34d399;
		border-color: #059669;
	}

	.walkin-btn {
		background: #0284c7;
		color: #ffffff;
		border: none;
		padding: 0.5rem 1rem;
		border-radius: 8px;
		font-weight: 600;
		cursor: pointer;
	}

	.metrics-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
		gap: 1rem;
		margin-bottom: 1.5rem;
	}

	.kpi-card {
		background: #0f172a;
		border: 1px solid #1e293b;
		padding: 1rem 1.25rem;
		border-radius: 12px;
		display: flex;
		align-items: center;
		gap: 1rem;
	}

	.kpi-icon {
		font-size: 2rem;
	}

	.kpi-info {
		display: flex;
		flex-direction: column;
	}

	.kpi-label {
		font-size: 0.75rem;
		color: #94a3b8;
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}

	.kpi-value {
		font-size: 1.375rem;
		font-weight: 800;
		color: #ffffff;
	}

	.kpi-value.warning {
		color: #f59e0b;
	}

	.kpi-value .sub {
		font-size: 0.875rem;
		color: #94a3b8;
		font-weight: 500;
	}

	.billing-tabs {
		display: flex;
		gap: 0.5rem;
		border-bottom: 1px solid #334155;
		margin-bottom: 1.5rem;
	}

	.tab-btn {
		background: transparent;
		border: none;
		color: #94a3b8;
		padding: 0.75rem 1.25rem;
		font-size: 0.9375rem;
		font-weight: 600;
		cursor: pointer;
		border-bottom: 2px solid transparent;
		transition: all 0.2s;
	}

	.tab-btn.active {
		color: #38bdf8;
		border-bottom-color: #38bdf8;
	}

	.bill-workbench {
		display: grid;
		grid-template-columns: 1fr 380px;
		gap: 1.5rem;
		align-items: start;
	}

	@media (max-width: 1024px) {
		.bill-workbench {
			grid-template-columns: 1fr;
		}
	}

	.card {
		background: #0f172a;
		border: 1px solid #1e293b;
		border-radius: 12px;
		padding: 1.25rem;
		margin-bottom: 1.5rem;
	}

	.card-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 1rem;
	}

	.card-header h3 {
		font-size: 1rem;
		font-weight: 700;
		color: #ffffff;
		margin: 0;
	}

	.gst-state-badge {
		font-size: 0.75rem;
		color: #38bdf8;
		background: rgba(56, 189, 248, 0.1);
		padding: 0.2rem 0.5rem;
		border-radius: 4px;
	}

	.form-row {
		display: flex;
		gap: 1rem;
		flex-wrap: wrap;
	}

	.form-group {
		display: flex;
		flex-direction: column;
		gap: 0.4rem;
	}

	.flex-1 {
		flex: 1;
		min-width: 160px;
	}

	.flex-2 {
		flex: 2;
		min-width: 220px;
	}

	label {
		font-size: 0.75rem;
		color: #94a3b8;
		font-weight: 600;
	}

	input, select {
		background: #1e293b;
		border: 1px solid #334155;
		color: #ffffff;
		padding: 0.5rem 0.75rem;
		border-radius: 6px;
		font-size: 0.875rem;
	}

	.table-responsive {
		overflow-x: auto;
	}

	.items-table, .data-table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.875rem;
	}

	th {
		text-align: left;
		padding: 0.6rem 0.75rem;
		color: #94a3b8;
		font-size: 0.75rem;
		text-transform: uppercase;
		border-bottom: 1px solid #334155;
	}

	td {
		padding: 0.6rem 0.75rem;
		border-bottom: 1px solid #1e293b;
	}

	.table-input {
		width: 100%;
		box-sizing: border-box;
		background: #1e293b;
		border: 1px solid #334155;
		color: #ffffff;
		padding: 0.4rem 0.5rem;
		border-radius: 4px;
	}

	.center {
		text-align: center;
	}

	.right {
		text-align: right;
	}

	.line-total-cell {
		font-weight: 700;
		color: #ffffff;
	}

	.del-row-btn {
		background: transparent;
		border: none;
		color: #ef4444;
		cursor: pointer;
		font-weight: 700;
	}

	.del-row-btn:disabled {
		opacity: 0.3;
		cursor: not-allowed;
	}

	.table-footer-actions {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-top: 1rem;
	}

	.add-row-btn {
		background: #0284c7;
		color: #ffffff;
		border: none;
		padding: 0.4rem 0.75rem;
		border-radius: 6px;
		font-weight: 600;
		cursor: pointer;
		font-size: 0.8125rem;
	}

	.add-row-btn-ghost {
		background: transparent;
		color: #38bdf8;
		border: 1px dashed #38bdf8;
		padding: 0.4rem 0.75rem;
		border-radius: 6px;
		font-weight: 600;
		cursor: pointer;
	}

	.keyboard-hint {
		font-size: 0.75rem;
		color: #64748b;
	}

	.checkout-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 1rem;
	}

	.active-mode-badge {
		background: #0284c7;
		color: #ffffff;
		font-size: 0.75rem;
		padding: 0.2rem 0.5rem;
		border-radius: 4px;
		font-weight: 700;
	}

	.payment-mode-grid {
		display: grid;
		grid-template-columns: repeat(2, 1fr);
		gap: 0.5rem;
		margin-bottom: 1rem;
	}

	.pay-btn {
		background: #1e293b;
		border: 1px solid #334155;
		color: #cbd5e1;
		padding: 0.6rem;
		border-radius: 8px;
		font-weight: 700;
		cursor: pointer;
		transition: all 0.2s;
	}

	.pay-btn.selected {
		background: #0284c7;
		color: #ffffff;
		border-color: #38bdf8;
	}

	.upi-box {
		background: #021a36;
		border: 1px dashed #38bdf8;
		border-radius: 8px;
		padding: 0.75rem;
		margin-bottom: 1rem;
	}

	.upi-qr-wrapper {
		display: flex;
		align-items: center;
		gap: 1rem;
	}

	.qr-placeholder {
		width: 70px;
		height: 70px;
		background: #ffffff;
		border-radius: 6px;
		padding: 4px;
	}

	.qr-svg {
		width: 100%;
		height: 100%;
	}

	.upi-text {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}

	.upi-id {
		font-size: 0.75rem;
		color: #94a3b8;
	}

	.tendered-group {
		margin-bottom: 0.75rem;
	}

	.tendered-group input {
		width: 100%;
		box-sizing: border-box;
		font-size: 1.125rem;
		font-weight: 700;
		text-align: right;
	}

	.balance-calc-row {
		margin-bottom: 1rem;
	}

	.change-tag {
		display: flex;
		justify-content: space-between;
		padding: 0.5rem 0.75rem;
		border-radius: 6px;
		font-size: 0.875rem;
	}

	.change-tag.return {
		background: rgba(16, 185, 129, 0.15);
		color: #34d399;
	}

	.change-tag.due {
		background: rgba(245, 158, 11, 0.15);
		color: #fbbf24;
	}

	.change-tag.full {
		background: rgba(56, 189, 248, 0.15);
		color: #38bdf8;
	}

	.summary-breakdown {
		border-top: 1px solid #1e293b;
		padding-top: 0.75rem;
		margin-bottom: 1.25rem;
	}

	.sum-row {
		display: flex;
		justify-content: space-between;
		font-size: 0.8125rem;
		color: #94a3b8;
		margin-bottom: 0.35rem;
	}

	.sum-row .disc {
		color: #34d399;
	}

	.sum-total-row {
		display: flex;
		justify-content: space-between;
		align-items: center;
		border-top: 1px dashed #334155;
		padding-top: 0.6rem;
		margin-top: 0.5rem;
		color: #ffffff;
		font-weight: 800;
	}

	.big-total {
		font-size: 1.5rem;
		color: #38bdf8;
	}

	.checkout-actions {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.primary-checkout-btn {
		background: #0284c7;
		color: #ffffff;
		border: none;
		padding: 0.75rem;
		border-radius: 8px;
		font-weight: 700;
		font-size: 1rem;
		cursor: pointer;
		transition: background 0.2s;
	}

	.primary-checkout-btn:hover {
		background: #0369a1;
	}

	.action-btn-row {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 0.5rem;
	}

	.secondary-btn {
		background: #1e293b;
		border: 1px solid #334155;
		color: #ffffff;
		padding: 0.6rem;
		border-radius: 8px;
		font-weight: 600;
		cursor: pointer;
	}

	.whatsapp-btn {
		background: #166534;
		border: none;
		color: #ffffff;
		padding: 0.6rem;
		border-radius: 8px;
		font-weight: 600;
		cursor: pointer;
	}

	/* Invoices Tab */
	.invoices-toolbar {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 1rem;
		flex-wrap: wrap;
		gap: 0.75rem;
	}

	.filter-group {
		display: flex;
		gap: 0.4rem;
	}

	.filter-pill {
		background: #1e293b;
		border: 1px solid #334155;
		color: #94a3b8;
		padding: 0.35rem 0.75rem;
		border-radius: 9999px;
		font-size: 0.8125rem;
		cursor: pointer;
	}

	.filter-pill.active {
		background: #0284c7;
		color: #ffffff;
		border-color: #38bdf8;
	}

	.new-bill-shortcut {
		background: #0284c7;
		color: #ffffff;
		border: none;
		padding: 0.4rem 0.8rem;
		border-radius: 6px;
		font-weight: 600;
		cursor: pointer;
	}

	.bold {
		font-weight: 700;
		color: #ffffff;
	}

	.green {
		color: #34d399;
	}

	.red {
		color: #f87171;
	}

	.muted {
		color: #64748b;
	}

	.cust-cell {
		display: flex;
		flex-direction: column;
	}

	.cust-title {
		font-weight: 600;
	}

	.cust-gstin {
		font-size: 0.75rem;
		color: #94a3b8;
	}

	.status-badge {
		padding: 0.2rem 0.5rem;
		border-radius: 4px;
		font-size: 0.75rem;
		font-weight: 700;
	}

	.status-badge.paid {
		background: rgba(16, 185, 129, 0.15);
		color: #34d399;
	}

	.status-badge.partial {
		background: rgba(245, 158, 11, 0.15);
		color: #fbbf24;
	}

	.status-badge.unpaid {
		background: rgba(239, 68, 68, 0.15);
		color: #f87171;
	}

	.status-badge.overdue {
		background: rgba(239, 68, 68, 0.25);
		color: #ef4444;
	}

	.actions-cell {
		display: flex;
		gap: 0.4rem;
	}

	.record-pay-btn {
		background: #0284c7;
		border: none;
		color: #ffffff;
		padding: 0.25rem 0.5rem;
		border-radius: 4px;
		font-size: 0.75rem;
		cursor: pointer;
	}

	.icon-action-btn {
		background: #1e293b;
		border: 1px solid #334155;
		color: #ffffff;
		padding: 0.25rem 0.5rem;
		border-radius: 4px;
		cursor: pointer;
	}

	.unpaid-badge {
		background: rgba(239, 68, 68, 0.15);
		color: #f87171;
		padding: 0.2rem 0.5rem;
		border-radius: 4px;
		font-size: 0.75rem;
	}

	.whatsapp-nudge-btn {
		background: #166534;
		border: none;
		color: #ffffff;
		padding: 0.35rem 0.75rem;
		border-radius: 6px;
		font-size: 0.75rem;
		font-weight: 600;
		cursor: pointer;
	}

	/* Modal */
	.modal-backdrop {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.7);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 1000;
	}

	.modal-card {
		background: #0f172a;
		border: 1px solid #334155;
		border-radius: 12px;
		width: 100%;
		max-width: 440px;
		padding: 1.5rem;
	}

	.modal-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 1rem;
	}

	.modal-header h3 {
		margin: 0;
		font-size: 1.125rem;
		color: #ffffff;
	}

	.modal-close {
		background: transparent;
		border: none;
		color: #94a3b8;
		cursor: pointer;
		font-size: 1.125rem;
	}

	.modal-info-row {
		display: flex;
		justify-content: space-between;
		padding: 0.3rem 0;
		font-size: 0.875rem;
		color: #94a3b8;
	}

	.modal-footer {
		display: flex;
		justify-content: flex-end;
		gap: 0.75rem;
		margin-top: 1.5rem;
	}

	.modal-error {
		background: rgba(239, 68, 68, 0.15);
		color: #f87171;
		padding: 0.5rem;
		border-radius: 6px;
		font-size: 0.8125rem;
		margin-bottom: 1rem;
	}
</style>
