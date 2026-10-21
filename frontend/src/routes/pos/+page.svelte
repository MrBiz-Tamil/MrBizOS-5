<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { api, type Product, type ProductBatch, type Customer, type SplitPayment, type Tenant } from '$lib/api';

	interface CartItem {
		product_id: string;
		name: string;
		sku: string;
		generic_name?: string;
		rack_location?: string;
		batch_id?: string;
		batch_number?: string;
		expiry_date?: string;
		available_batches?: ProductBatch[];
		quantity: number;
		unit_code: string;
		mrp: number;
		selling_price: number;
		discount_rate: number;
		tax_rate: number;
		is_prescription_required?: boolean;
	}

	interface HeldCart {
		id: string;
		customer_id?: string;
		customer_name: string;
		items: CartItem[];
		timestamp: string;
		total: number;
	}

	// State
	let company = $state<Tenant | null>(null);
	let customers = $state<Customer[]>([]);
	let selectedCustomerId = $state<string>('');
	let customerSearch = $state('');

	// Pharmacy Doctor Metadata
	let doctorName = $state('');
	let doctorRegNo = $state('');
	let patientName = $state('');
	let patientAge = $state<number | undefined>(undefined);
	let showPharmacyFields = $state(false);
	let isHealthcareMode = $derived(typeof localStorage !== 'undefined' && localStorage.getItem('mrbiz_active_unit') === 'healthcare');
	let isTravelMode = $derived(typeof localStorage !== 'undefined' && localStorage.getItem('mrbiz_active_unit') === 'travel');

	// Cart & Search
	let searchQuery = $state('');
	let searchResults = $state<Product[]>([]);
	let isSearching = $state(false);
	let cart = $state<CartItem[]>([]);
	let searchInputRef: HTMLInputElement | null = null;

	// Held Carts
	let heldCarts = $state<HeldCart[]>([]);
	let showHeldModal = $state(false);

	// Payment & Checkout
	let showPaymentModal = $state(false);
	let tenderedCash = $state<number>(0);
	let paymentMode = $state<'CASH' | 'SPLIT'>('CASH');
	let splitPayments = $state<SplitPayment[]>([
		{ mode: 'CASH', amount: '0' },
		{ mode: 'UPI', amount: '0' }
	]);
	let cashierName = $state('Counter #01');
	let counterNumber = $state('01');
	let isSubmitting = $state(false);

	// Last successful bill for receipt modal
	let lastCompletedBill = $state<any>(null);
	let showReceiptModal = $state(false);

	// High-speed barcode scanner buffer
	let barcodeBuffer = '';
	let lastKeyTime = 0;

	// Computed Totals
	let subtotalMRP = $derived(
		cart.reduce((acc, item) => acc + (item.mrp || item.selling_price) * item.quantity, 0)
	);

	let taxableAmount = $derived(
		cart.reduce((acc, item) => {
			const lineBase = item.selling_price * item.quantity * (1 - item.discount_rate / 100);
			const taxVal = lineBase / (1 + item.tax_rate / 100);
			return acc + taxVal;
		}, 0)
	);

	let grandTotal = $derived(
		Math.round(
			cart.reduce((acc, item) => {
				const lineTotal = item.selling_price * item.quantity * (1 - item.discount_rate / 100);
				return acc + lineTotal;
			}, 0)
		)
	);

	let totalSavings = $derived(
		Math.max(0, subtotalMRP - grandTotal)
	);

	let changeToReturn = $derived(
		Math.max(0, (tenderedCash || 0) - grandTotal)
	);

	let splitTotal = $derived(
		splitPayments.reduce((acc, p) => acc + (parseFloat(p.amount) || 0), 0)
	);

	let splitBalanceRemaining = $derived(
		grandTotal - splitTotal
	);

	// Touch Quick Tiles
	const quickPicks = [
		{ name: 'Fresh Milk 500ml', sku: 'MILK-500', price: 32, mrp: 32, tax: 0, unit: 'PKT' },
		{ name: 'Whole Wheat Bread', sku: 'BREAD-W', price: 45, mrp: 50, tax: 0, unit: 'PKT' },
		{ name: 'Farm Eggs (6 pcs)', sku: 'EGGS-6', price: 48, mrp: 55, tax: 0, unit: 'BOX' },
		{ name: 'Mineral Water 1L', sku: 'WATER-1L', price: 20, mrp: 20, tax: 18, unit: 'BTL' },
		{ name: 'Eco Carry Bag Large', sku: 'BAG-ECO', price: 10, mrp: 15, tax: 18, unit: 'PCS' },
		{ name: 'Paracetamol 650mg (Strip)', sku: 'PARA-650', generic: 'Paracetamol 650mg', price: 34, mrp: 38, tax: 12, unit: 'STRP', rx: false, rack: 'A-02' }
	];

	onMount(async () => {
		try {
			const companyData = await api.getCompany();
			company = companyData;
			customers = await api.listCustomers();
			if (customers.length > 0) {
				selectedCustomerId = customers[0].id;
			}
		} catch (e) {
			console.error('Failed loading POS dependencies', e);
		}

		// Load held carts from localStorage
		try {
			const saved = localStorage.getItem('pos_held_carts');
			if (saved) {
				heldCarts = JSON.parse(saved);
			}
		} catch (e) {}

		// Keyboard Hotkeys and Barcode Scan listener
		window.addEventListener('keydown', handleGlobalKeydown);
		if (searchInputRef) {
			searchInputRef.focus();
		}
	});

	onDestroy(() => {
		if (typeof window !== 'undefined') {
			window.removeEventListener('keydown', handleGlobalKeydown);
		}
	});

	function handleGlobalKeydown(e: KeyboardEvent) {
		const target = e.target as HTMLElement;
		const isInputActive = target && (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.tagName === 'SELECT');

		// Function Key Hotkeys
		if (e.key === 'F1') {
			e.preventDefault();
			searchInputRef?.focus();
			return;
		}
		if (e.key === 'F2') {
			e.preventDefault();
			holdCurrentCart();
			return;
		}
		if (e.key === 'F4') {
			e.preventDefault();
			showHeldModal = true;
			return;
		}
		if (e.key === 'F8') {
			e.preventDefault();
			if (cart.length > 0) {
				paymentMode = 'CASH';
				tenderedCash = grandTotal;
				showPaymentModal = true;
			}
			return;
		}
		if (e.key === 'F9') {
			e.preventDefault();
			clearCart();
			return;
		}
		if (e.key === 'F10') {
			e.preventDefault();
			if (cart.length > 0) {
				paymentMode = 'SPLIT';
				splitPayments = [
					{ mode: 'CASH', amount: (grandTotal / 2).toFixed(2) },
					{ mode: 'UPI', amount: (grandTotal - grandTotal / 2).toFixed(2) }
				];
				showPaymentModal = true;
			}
			return;
		}

		// Global Barcode Hardware Scanner Handling
		// Hardware barcode scanners send keystrokes in rapid succession (< 40ms)
		const now = Date.now();
		if (now - lastKeyTime > 60) {
			barcodeBuffer = '';
		}
		lastKeyTime = now;

		if (e.key === 'Enter') {
			if (barcodeBuffer.length >= 3 && !isInputActive) {
				e.preventDefault();
				processScannedBarcode(barcodeBuffer.trim());
				barcodeBuffer = '';
			}
		} else if (e.key.length === 1 && !isInputActive) {
			barcodeBuffer += e.key;
		}
	}

	async function processScannedBarcode(barcode: string) {
		try {
			const products = await api.searchPOSProducts(barcode);
			if (products && products.length > 0) {
				addToCart(products[0]);
			}
		} catch (e) {
			console.error('Barcode lookup failed', e);
		}
	}

	async function handleSearchInput(e: Event) {
		const q = (e.target as HTMLInputElement).value;
		searchQuery = q;
		if (!q.trim()) {
			searchResults = [];
			return;
		}
		isSearching = true;
		try {
			const results = await api.searchPOSProducts(q.trim());
			searchResults = results || [];
		} catch (e) {
			searchResults = [];
		} finally {
			isSearching = false;
		}
	}

	function handleSearchKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			e.preventDefault();
			if (searchResults.length > 0) {
				addToCart(searchResults[0]);
				searchQuery = '';
				searchResults = [];
			} else if (searchQuery.trim()) {
				// Try direct barcode match
				processScannedBarcode(searchQuery.trim());
				searchQuery = '';
				searchResults = [];
			}
		}
	}

	function addToCart(prod: Product) {
		// If product has prescription requirement, open pharmacy section
		if (prod.is_prescription_required) {
			showPharmacyFields = true;
		}

		// Check if product already exists with same batch
		const bestBatch = prod.batches && prod.batches.length > 0 ? prod.batches[0] : null;
		const existingIdx = cart.findIndex(item => 
			item.product_id === prod.id && (!bestBatch || item.batch_id === bestBatch.id)
		);

		if (existingIdx >= 0) {
			cart[existingIdx].quantity += 1;
			cart = [...cart];
		} else {
			const mrpVal = bestBatch ? parseFloat(bestBatch.mrp) : parseFloat(prod.mrp || prod.selling_price);
			const sellVal = bestBatch ? parseFloat(bestBatch.selling_price) : parseFloat(prod.selling_price);

			cart = [
				...cart,
				{
					product_id: prod.id,
					name: prod.name,
					sku: prod.sku,
					generic_name: prod.generic_name,
					rack_location: prod.rack_location,
					batch_id: bestBatch?.id,
					batch_number: bestBatch?.batch_number,
					expiry_date: bestBatch?.expiry_date,
					available_batches: prod.batches || [],
					quantity: 1,
					unit_code: 'PCS',
					mrp: mrpVal,
					selling_price: sellVal,
					discount_rate: parseFloat(prod.discount_rate || '0'),
					tax_rate: parseFloat(prod.tax_rate || '0'),
					is_prescription_required: prod.is_prescription_required
				}
			];
		}

		searchQuery = '';
		searchResults = [];
		searchInputRef?.focus();
	}

	function addQuickPick(pick: typeof quickPicks[0]) {
		const existingIdx = cart.findIndex(i => i.sku === pick.sku);
		if (existingIdx >= 0) {
			cart[existingIdx].quantity += 1;
			cart = [...cart];
		} else {
			cart = [
				...cart,
				{
					product_id: 'quick-' + pick.sku,
					name: pick.name,
					sku: pick.sku,
					generic_name: pick.generic,
					rack_location: pick.rack,
					quantity: 1,
					unit_code: pick.unit,
					mrp: pick.mrp,
					selling_price: pick.price,
					discount_rate: 0,
					tax_rate: pick.tax,
					is_prescription_required: pick.rx
				}
			];
		}
	}

	function updateQuantity(index: number, delta: number) {
		cart[index].quantity += delta;
		if (cart[index].quantity <= 0) {
			cart.splice(index, 1);
		}
		cart = [...cart];
	}

	function removeItem(index: number) {
		cart.splice(index, 1);
		cart = [...cart];
	}

	function handleBatchChange(index: number, e: Event) {
		const batchId = (e.target as HTMLSelectElement).value;
		const item = cart[index];
		const selected = item.available_batches?.find(b => b.id === batchId);
		if (selected) {
			item.batch_id = selected.id;
			item.batch_number = selected.batch_number;
			item.expiry_date = selected.expiry_date;
			item.mrp = parseFloat(selected.mrp);
			item.selling_price = parseFloat(selected.selling_price);
			cart = [...cart];
		}
	}

	function holdCurrentCart() {
		if (cart.length === 0) return;
		const cust = customers.find(c => c.id === selectedCustomerId);
		const held: HeldCart = {
			id: 'HOLD-' + Date.now().toString().slice(-4),
			customer_id: selectedCustomerId,
			customer_name: cust ? cust.name : 'Walk-in Shopper',
			items: [...cart],
			timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
			total: grandTotal
		};
		heldCarts = [held, ...heldCarts];
		localStorage.setItem('pos_held_carts', JSON.stringify(heldCarts));
		clearCart();
	}

	function resumeCart(held: HeldCart) {
		cart = [...held.items];
		if (held.customer_id) {
			selectedCustomerId = held.customer_id;
		}
		heldCarts = heldCarts.filter(h => h.id !== held.id);
		localStorage.setItem('pos_held_carts', JSON.stringify(heldCarts));
		showHeldModal = false;
	}

	function deleteHeldCart(id: string) {
		heldCarts = heldCarts.filter(h => h.id !== id);
		localStorage.setItem('pos_held_carts', JSON.stringify(heldCarts));
	}

	function clearCart() {
		cart = [];
		doctorName = '';
		doctorRegNo = '';
		patientName = '';
		patientAge = undefined;
		showPharmacyFields = false;
		tenderedCash = 0;
		searchInputRef?.focus();
	}

	async function checkoutBill() {
		if (cart.length === 0) return;
		isSubmitting = true;

		try {
			const invoiceItems = cart.map(item => ({
				product_id: item.product_id.startsWith('quick-') ? undefined : item.product_id,
				item_description: item.name,
				hsn_sac: '9988',
				quantity: item.quantity.toString(),
				unit_code: item.unit_code || 'PCS',
				unit_price: item.selling_price.toString(),
				discount_rate: item.discount_rate.toString(),
				tax_rate: item.tax_rate.toString(),
				batch_id: item.batch_id,
				batch_number: item.batch_number || '',
				expiry_date: item.expiry_date || '',
				mrp: item.mrp.toString(),
				rack_location: item.rack_location || ''
			}));

			const invoicePayload: any = {
				customer_id: selectedCustomerId || customers[0]?.id,
				invoice_date: new Date().toISOString(),
				due_date: new Date().toISOString(),
				billing_address: 'Walk-in Store Customer',
				shipping_address: 'Walk-in Store Customer',
				supply_state: company?.state || 'Maharashtra',
				payment_mode: paymentMode === 'CASH' ? 'CASH' : 'SPLIT',
				cashier_name: cashierName,
				counter_number: counterNumber,
				items: invoiceItems,
				tendered_cash: tenderedCash.toString(),
				change_amount: changeToReturn.toString(),
				doctor_name: doctorName,
				doctor_reg_no: doctorRegNo,
				patient_name: patientName,
				patient_age: patientAge
			};

			if (paymentMode === 'SPLIT') {
				invoicePayload.split_payments = splitPayments.filter(p => parseFloat(p.amount) > 0);
			}

			const created = await api.createInvoice(invoicePayload);
			lastCompletedBill = {
				invoice: created,
				cart: [...cart],
				company: company,
				change: changeToReturn,
				tendered: tenderedCash,
				savings: totalSavings
			};

			showPaymentModal = false;
			showReceiptModal = true;
			clearCart();
		} catch (e: any) {
			alert('Failed to process retail sale: ' + (e?.message || 'Unknown error'));
		} finally {
			isSubmitting = false;
		}
	}

	function printReceipt() {
		window.print();
	}
</script>

<svelte:head>
	<title>Express Retail & Pharmacy POS Terminal | Supermarket Billing</title>
</svelte:head>

<div class="pos-container" id="pos-terminal-app">
	<!-- TOP STATUS & CONTROLS BAR -->
	<header class="pos-topbar">
		<div class="brand-badge">
			<span class="pos-pulse-dot"></span>
			<strong>{company?.name || 'Supermarket POS'}</strong>
			<span class="counter-chip">Counter {counterNumber} • {cashierName}</span>
		</div>

		<!-- Pharmacy Regulatory Alert Pill (Only in Healthcare Workspace) -->
		{#if isHealthcareMode}
			<div class="pharmacy-toggle-wrap">
				<button
					type="button"
					class="rx-btn"
					class:active={showPharmacyFields}
					onclick={() => (showPharmacyFields = !showPharmacyFields)}
					id="btn-toggle-rx"
				>
					<span>🩺</span>
					{showPharmacyFields ? 'Pharmacy Rx Mode (Active)' : 'Pharmacy Rx Metadata'}
				</button>
			</div>
		{/if}

		<!-- Held Carts Pill -->
		<div class="topbar-actions">
			<button
				type="button"
				class="hold-queue-btn"
				onclick={() => (showHeldModal = true)}
				id="btn-open-held"
			>
				<span>⏸️ Parked Bills</span>
				{#if heldCarts.length > 0}
					<span class="queue-badge">{heldCarts.length}</span>
				{/if}
				<kbd>F4</kbd>
			</button>

			<button
				type="button"
				class="hold-current-btn"
				disabled={cart.length === 0}
				onclick={holdCurrentCart}
				id="btn-hold-cart"
			>
				<span>Hold Current</span>
				<kbd>F2</kbd>
			</button>

			<button
				type="button"
				class="clear-cart-btn"
				disabled={cart.length === 0}
				onclick={clearCart}
				id="btn-clear-cart"
			>
				<span>Clear</span>
				<kbd>F9</kbd>
			</button>
		</div>
	</header>

	<!-- PHARMACY METADATA COLLAPSIBLE SECTION -->
	{#if showPharmacyFields}
		<div class="rx-metadata-drawer" id="rx-compliance-box">
			<div class="rx-header">
				<span class="rx-icon">💊</span>
				<div>
					<strong>Schedule H / H1 / X DPCO Pharmacy Prescription Records</strong>
					<p>Drug Controller & State Pharmacy Council mandatory sale compliance</p>
				</div>
			</div>
			<div class="rx-grid">
				<div class="input-group">
					<label for="pos-doctor-name">Doctor Name</label>
					<input
						id="pos-doctor-name"
						type="text"
						placeholder="Dr. R. Sharma, MBBS"
						bind:value={doctorName}
					/>
				</div>
				<div class="input-group">
					<label for="pos-doctor-reg">Doctor Reg. No.</label>
					<input
						id="pos-doctor-reg"
						type="text"
						placeholder="MCI / PMC-98421"
						bind:value={doctorRegNo}
					/>
				</div>
				<div class="input-group">
					<label for="pos-patient-name">Patient Name</label>
					<input
						id="pos-patient-name"
						type="text"
						placeholder="Patient Name"
						bind:value={patientName}
					/>
				</div>
				<div class="input-group">
					<label for="pos-patient-age">Patient Age</label>
					<input
						id="pos-patient-age"
						type="number"
						placeholder="Years"
						bind:value={patientAge}
					/>
				</div>
			</div>
		</div>
	{/if}

	<!-- MAIN POS TERMINAL LAYOUT -->
	<div class="pos-layout">
		<!-- LEFT SIDE: FAST SEARCH & CATALOG -->
		<section class="pos-left-pane">
			<!-- BARCODE & FAST INPUT -->
			<div class="search-hero">
				<div class="search-input-wrapper">
					<span class="scanner-icon">⚡</span>
					<input
						bind:this={searchInputRef}
						type="text"
						id="pos-barcode-search"
						class="pos-search-box"
						placeholder="Scan barcode, generic composition, brand name, or SKU (Press F1)..."
						value={searchQuery}
						oninput={handleSearchInput}
						onkeydown={handleSearchKeydown}
						autocomplete="off"
					/>
					{#if isSearching}
						<span class="search-spinner">⏳</span>
					{/if}
				</div>

				<!-- AUTOCOMPLETE DROPDOWN -->
				{#if searchResults.length > 0}
					<div class="search-dropdown-results" id="pos-search-results">
						{#each searchResults as prod}
							<button
								type="button"
								class="search-result-item"
								onclick={() => addToCart(prod)}
							>
								<div class="res-info">
									<div class="res-title">
										<strong>{prod.name}</strong>
										{#if prod.is_prescription_required}
											<span class="rx-pill">Rx Only</span>
										{/if}
										{#if prod.rack_location}
											<span class="rack-pill">📍 {prod.rack_location}</span>
										{/if}
									</div>
									{#if prod.generic_name}
										<div class="res-generic">Salt: {prod.generic_name}</div>
									{/if}
									<div class="res-meta">
										<span>SKU: {prod.sku}</span>
										<span>Stock: {prod.current_stock || 0}</span>
										{#if prod.batches && prod.batches.length > 0}
											<span class="batch-count-pill">⚡ {prod.batches.length} Active Batches</span>
										{/if}
									</div>
								</div>
								<div class="res-pricing">
									<div class="res-sell-price">₹{prod.selling_price}</div>
									{#if prod.mrp && parseFloat(prod.mrp) > parseFloat(prod.selling_price)}
										<div class="res-mrp">MRP ₹{prod.mrp}</div>
									{/if}
								</div>
							</button>
						{/each}
					</div>
				{/if}
			</div>

			<!-- TOUCH SPEED CATEGORY TILES -->
			<div class="quick-picks-section">
				<div class="section-title">
					<span>⚡ Express 1-Touch Fast Picks</span>
					<span class="subtitle">High velocity items</span>
				</div>
				<div class="quick-picks-grid">
					{#each quickPicks as pick}
						<button
							type="button"
							class="quick-tile"
							onclick={() => addQuickPick(pick)}
							id={`btn-quick-${pick.sku}`}
						>
							<div class="quick-tile-header">
								<span class="quick-name">{pick.name}</span>
								{#if pick.rx}
									<span class="rx-badge">Rx</span>
								{/if}
							</div>
							{#if pick.generic}
								<span class="quick-salt">{pick.generic}</span>
							{/if}
							<div class="quick-tile-footer">
								<span class="quick-price">₹{pick.price}</span>
								{#if pick.mrp > pick.price}
									<span class="quick-mrp">₹{pick.mrp}</span>
								{/if}
								{#if pick.rack}
									<span class="quick-rack">📍 {pick.rack}</span>
								{/if}
							</div>
						</button>
					{/each}
				</div>
			</div>

			<!-- CUSTOMER SELECTION BAR -->
			<div class="customer-strip">
				<label for="pos-customer-select">
					{#if isTravelMode}
						✈️ Lead Passenger / Traveler:
					{:else if isHealthcareMode}
						🏥 Patient / Hospital Record:
					{:else}
						Customer / Loyalty:
					{/if}
				</label>
				<select id="pos-customer-select" bind:value={selectedCustomerId}>
					{#each customers as c}
						<option value={c.id}>{c.name} {c.phone ? `(${c.phone})` : ''}</option>
					{/each}
				</select>
			</div>
		</section>

		<!-- RIGHT SIDE: ACTIVE CART & CHECKOUT -->
		<section class="pos-right-pane">
			<!-- CART TABLE -->
			<div class="cart-table-wrapper">
				<table class="cart-table" id="pos-cart-table">
					<thead>
						<tr>
							<th style="width: 40%;">Item Description</th>
							<th style="width: 25%;">Batch & Expiry</th>
							<th style="width: 15%; text-align: center;">Qty</th>
							<th style="width: 10%; text-align: right;">Rate</th>
							<th style="width: 10%; text-align: right;">Total</th>
						</tr>
					</thead>
					<tbody>
						{#if cart.length === 0}
							<tr class="empty-cart-row">
								<td colspan="5">
									<div class="empty-cart-state">
										<div class="empty-icon">🛒</div>
										<h4>Cart is Empty</h4>
										<p>Scan barcode with hardware gun or use quick pick tiles</p>
									</div>
								</td>
							</tr>
						{:else}
							{#each cart as item, idx}
								<tr class="cart-row">
									<td>
										<div class="cart-item-name">
											<strong>{item.name}</strong>
											{#if item.is_prescription_required}
												<span class="rx-tiny-pill">Rx</span>
											{/if}
										</div>
										{#if item.generic_name}
											<div class="cart-item-salt">{item.generic_name}</div>
										{/if}
										{#if item.rack_location}
											<div class="cart-item-rack">📍 {item.rack_location}</div>
										{/if}
									</td>
									<td>
										{#if item.available_batches && item.available_batches.length > 0}
											<select
												class="batch-dropdown"
												value={item.batch_id}
												onchange={(e) => handleBatchChange(idx, e)}
											>
												{#each item.available_batches as b}
													<option value={b.id}>
														B:{b.batch_number} (Exp:{b.expiry_date}) MRP ₹{b.mrp}
													</option>
												{/each}
											</select>
										{:else if item.batch_number}
											<span class="single-batch-pill">B:{item.batch_number} Exp:{item.expiry_date}</span>
										{:else}
											<span class="no-batch-muted">Standard Stock</span>
										{/if}
									</td>
									<td>
										<div class="qty-control">
											<button type="button" onclick={() => updateQuantity(idx, -1)} class="qty-btn">-</button>
											<span class="qty-display">{item.quantity}</span>
											<button type="button" onclick={() => updateQuantity(idx, 1)} class="qty-btn">+</button>
										</div>
									</td>
									<td style="text-align: right;">
										<div class="item-rate">₹{item.selling_price}</div>
										{#if item.mrp > item.selling_price}
											<div class="item-mrp-struck">₹{item.mrp}</div>
										{/if}
									</td>
									<td style="text-align: right;">
										<div class="item-line-total">
											₹{(item.selling_price * item.quantity).toFixed(2)}
										</div>
										<button
											type="button"
											class="item-delete-btn"
											onclick={() => removeItem(idx)}
											title="Remove item"
										>
											×
										</button>
									</td>
								</tr>
							{/each}
						{/if}
					</tbody>
				</table>
			</div>

			<!-- RUNNING SUMMARY & SAVINGS BANNER -->
			<div class="cart-checkout-card">
				{#if totalSavings > 0}
					<div class="savings-banner" id="pos-savings-banner">
						<span class="sparkle">🎉</span>
						<div>
							<strong>Customer Savings: ₹{totalSavings.toFixed(2)}</strong>
							<span>Retail discount on MRP</span>
						</div>
					</div>
				{/if}

				<div class="summary-breakdown">
					<div class="summary-line">
						<span>Items Count ({cart.reduce((a, b) => a + b.quantity, 0)} units)</span>
						<span>₹{subtotalMRP.toFixed(2)}</span>
					</div>
					<div class="summary-line">
						<span>GST Tax Breakdown</span>
						<span>Included in MRP</span>
					</div>
					<div class="summary-line grand-total-line">
						<span>Grand Total</span>
						<span class="grand-total-amount">₹{grandTotal}</span>
					</div>
				</div>

				<!-- FAST CHECKOUT BUTTONS -->
				<div class="checkout-actions">
					<button
						type="button"
						class="cash-pay-btn"
						disabled={cart.length === 0}
						onclick={() => {
							paymentMode = 'CASH';
							tenderedCash = grandTotal;
							showPaymentModal = true;
						}}
						id="btn-quick-cash"
					>
						<span class="btn-icon">💵</span>
						<div class="btn-text">
							<strong>Quick Cash Pay</strong>
							<span>Exact / Change calc</span>
						</div>
						<kbd>F8</kbd>
					</button>

					<button
						type="button"
						class="split-pay-btn"
						disabled={cart.length === 0}
						onclick={() => {
							paymentMode = 'SPLIT';
							splitPayments = [
								{ mode: 'CASH', amount: (grandTotal / 2).toFixed(2) },
								{ mode: 'UPI', amount: (grandTotal - grandTotal / 2).toFixed(2) }
							];
							showPaymentModal = true;
						}}
						id="btn-split-tender"
					>
						<span class="btn-icon">💳</span>
						<div class="btn-text">
							<strong>Split Tender</strong>
							<span>Cash + UPI + Card</span>
						</div>
						<kbd>F10</kbd>
					</button>
				</div>
			</div>
		</section>
	</div>
</div>

<!-- PAYMENT MODAL -->
{#if showPaymentModal}
	<div class="pos-modal-overlay">
		<div class="pos-modal-card" id="pos-checkout-modal">
			<div class="pos-modal-header">
				<h3>Express Payment Checkout</h3>
				<button type="button" class="close-modal-btn" onclick={() => (showPaymentModal = false)}>✕</button>
			</div>

			<div class="pos-modal-body">
				<div class="modal-due-card">
					<span class="due-label">Total Amount Payable</span>
					<span class="due-val">₹{grandTotal}</span>
				</div>

				<!-- PAYMENT MODE SELECTOR -->
				<div class="mode-pills">
					<button
						type="button"
						class="mode-pill"
						class:active={paymentMode === 'CASH'}
						onclick={() => (paymentMode = 'CASH')}
					>
						💵 Cash Payment
					</button>
					<button
						type="button"
						class="mode-pill"
						class:active={paymentMode === 'SPLIT'}
						onclick={() => (paymentMode = 'SPLIT')}
					>
						💳 Multi-Tender Split
					</button>
				</div>

				{#if paymentMode === 'CASH'}
					<!-- CASH QUICK TENDER BUTTONS -->
					<div class="cash-input-wrap">
						<label for="pos-tendered-cash">Tendered Cash Amount (₹)</label>
						<input
							id="pos-tendered-cash"
							type="number"
							class="big-cash-input"
							bind:value={tenderedCash}
							min="0"
						/>
						<div class="quick-cash-chips">
							<button type="button" onclick={() => (tenderedCash = grandTotal)}>Exact</button>
							<button type="button" onclick={() => (tenderedCash = Math.ceil(grandTotal / 100) * 100)}>₹{Math.ceil(grandTotal / 100) * 100}</button>
							<button type="button" onclick={() => (tenderedCash = 500)}>₹500</button>
							<button type="button" onclick={() => (tenderedCash = 1000)}>₹1000</button>
							<button type="button" onclick={() => (tenderedCash = 2000)}>₹2000</button>
						</div>

						<!-- CHANGE RETURN DISPLAY -->
						<div class="change-return-box" class:alert={tenderedCash < grandTotal}>
							{#if tenderedCash >= grandTotal}
								<span class="change-lbl">Change to Return:</span>
								<span class="change-val">₹{changeToReturn.toFixed(2)}</span>
							{:else}
								<span class="change-lbl" style="color: #f87171;">Short by:</span>
								<span class="change-val" style="color: #f87171;">₹{(grandTotal - tenderedCash).toFixed(2)}</span>
							{/if}
						</div>
					</div>
				{:else}
					<!-- SPLIT TENDER ROWS -->
					<div class="split-tender-rows">
						{#each splitPayments as split, sIdx}
							<div class="split-row">
								<select bind:value={split.mode}>
									<option value="CASH">Cash</option>
									<option value="UPI">UPI / QR</option>
									<option value="CARD">Debit / Credit Card</option>
									<option value="WALLET">Mobile Wallet</option>
									<option value="CREDIT">Customer Credit Ledger</option>
								</select>
								<input
									type="number"
									placeholder="Amount"
									bind:value={split.amount}
								/>
								<button
									type="button"
									class="split-del"
									onclick={() => {
										splitPayments.splice(sIdx, 1);
										splitPayments = [...splitPayments];
									}}
								>
									✕
								</button>
							</div>
						{/each}

						<button
							type="button"
							class="add-split-btn"
							onclick={() => {
								splitPayments = [
									...splitPayments,
									{ mode: 'UPI', amount: Math.max(0, splitBalanceRemaining).toFixed(2) }
								];
							}}
						>
							+ Add Another Payment Method
						</button>

						<div class="split-balance-bar">
							<span>Split Total: ₹{splitTotal.toFixed(2)}</span>
							{#if splitBalanceRemaining === 0}
								<span style="color: #34d399; font-weight: 700;">✓ Fully Balanced</span>
							{:else if splitBalanceRemaining > 0}
								<span style="color: #f59e0b;">Remaining: ₹{splitBalanceRemaining.toFixed(2)}</span>
							{:else}
								<span style="color: #ef4444;">Exceeds by: ₹{Math.abs(splitBalanceRemaining).toFixed(2)}</span>
							{/if}
						</div>
					</div>
				{/if}
			</div>

			<div class="pos-modal-footer">
				<button type="button" class="cancel-btn" onclick={() => (showPaymentModal = false)}>Cancel</button>
				<button
					type="button"
					class="complete-btn"
					disabled={isSubmitting || (paymentMode === 'CASH' && tenderedCash < grandTotal) || (paymentMode === 'SPLIT' && splitBalanceRemaining !== 0)}
					onclick={checkoutBill}
					id="btn-confirm-checkout"
				>
					{#if isSubmitting}
						Processing Sale...
					{:else}
						Complete Sale & Print (Enter)
					{/if}
				</button>
			</div>
		</div>
	</div>
{/if}

<!-- HELD CARTS MODAL -->
{#if showHeldModal}
	<div class="pos-modal-overlay">
		<div class="pos-modal-card" style="max-width: 550px;">
			<div class="pos-modal-header">
				<h3>Parked / Held Customer Bills</h3>
				<button type="button" class="close-modal-btn" onclick={() => (showHeldModal = false)}>✕</button>
			</div>
			<div class="pos-modal-body">
				{#if heldCarts.length === 0}
					<div class="empty-held">
						<p>No bills are currently parked in the queue.</p>
					</div>
				{:else}
					<div class="held-list">
						{#each heldCarts as held}
							<div class="held-item">
								<div class="held-info">
									<strong>{held.customer_name} ({held.id})</strong>
									<span class="held-meta">{held.items.length} items • ₹{held.total} • Parked at {held.timestamp}</span>
								</div>
								<div class="held-actions">
									<button type="button" class="resume-btn" onclick={() => resumeCart(held)}>
										Resume
									</button>
									<button type="button" class="del-held-btn" onclick={() => deleteHeldCart(held.id)}>
										✕
									</button>
								</div>
							</div>
						{/each}
					</div>
				{/if}
			</div>
		</div>
	</div>
{/if}

<!-- PRINT RECEIPT MODAL (Thermal 80mm Preview) -->
{#if showReceiptModal && lastCompletedBill}
	<div class="pos-modal-overlay">
		<div class="pos-modal-card receipt-modal">
			<div class="pos-modal-header">
				<h3>Bill Generated Successfully!</h3>
				<button type="button" class="close-modal-btn" onclick={() => (showReceiptModal = false)}>✕</button>
			</div>
			<div class="pos-modal-body">
				<div class="thermal-receipt-view" id="printable-pos-receipt">
					<div class="th-header">
						<h4>{company?.name || 'SUPERMARKET & PHARMACY'}</h4>
						<p>{company?.address || 'Main Street Road'}, {company?.city || 'Pune'}</p>
						{#if company?.drug_license_no}
							<p>D.L. No: {company.drug_license_no}</p>
						{/if}
						{#if company?.gstin}
							<p>GSTIN: {company.gstin}</p>
						{/if}
						<div class="th-divider">--------------------------------</div>
						<div class="th-meta-row">
							<span>Inv: {lastCompletedBill.invoice.invoice_number}</span>
							<span>Date: {new Date().toLocaleDateString()}</span>
						</div>
						<div class="th-meta-row">
							<span>Cashier: {cashierName}</span>
							<span>Counter: {counterNumber}</span>
						</div>
						<div class="th-divider">================================</div>
					</div>

					<div class="th-items-table">
						<div class="th-head-row">
							<span>Item</span>
							<span style="text-align: center;">Qty</span>
							<span style="text-align: right;">MRP</span>
							<span style="text-align: right;">Total</span>
						</div>
						<div class="th-divider">--------------------------------</div>
						{#each lastCompletedBill.cart as item}
							<div class="th-item-line">
								<div class="th-item-name">{item.name}</div>
								{#if item.batch_number}
									<div class="th-item-batch">B:{item.batch_number} Exp:{item.expiry_date}</div>
								{/if}
								<div class="th-item-calc">
									<span>{item.quantity} x ₹{item.selling_price}</span>
									<span>₹{(item.quantity * item.selling_price).toFixed(2)}</span>
								</div>
							</div>
						{/each}
						<div class="th-divider">--------------------------------</div>
					</div>

					<div class="th-totals">
						<div class="th-tot-row grand">
							<strong>NET PAYABLE:</strong>
							<strong>₹{lastCompletedBill.invoice.grand_total}</strong>
						</div>
						{#if lastCompletedBill.savings > 0}
							<div class="th-savings">
								*** YOU SAVED ₹{lastCompletedBill.savings.toFixed(2)} ON MRP ***
							</div>
						{/if}
						{#if lastCompletedBill.change > 0}
							<div class="th-tot-row">
								<span>Cash Tendered:</span>
								<span>₹{lastCompletedBill.tendered}</span>
							</div>
							<div class="th-tot-row">
								<span>Change Returned:</span>
								<span>₹{lastCompletedBill.change.toFixed(2)}</span>
							</div>
						{/if}
					</div>

					<div class="th-footer">
						<div class="th-divider">================================</div>
						<p>THANK YOU! VISIT AGAIN</p>
						<p>Goods once sold will not be returned</p>
					</div>
				</div>
			</div>
			<div class="pos-modal-footer">
				<button type="button" class="cancel-btn" onclick={() => (showReceiptModal = false)}>Close</button>
				<button type="button" class="complete-btn" onclick={printReceipt}>🖨️ Print Receipt (Ctrl+P)</button>
			</div>
		</div>
	</div>
{/if}

<style>
	.pos-container {
		display: flex;
		flex-direction: column;
		height: calc(100vh - 4.5rem);
		background: #0f172a;
		color: #f8fafc;
		font-family: var(--font-sans, system-ui, sans-serif);
		overflow: hidden;
	}

	/* TOPBAR */
	.pos-topbar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0.625rem 1.25rem;
		background: #1e293b;
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
	}
	.brand-badge {
		display: flex;
		align-items: center;
		gap: 0.625rem;
	}
	.pos-pulse-dot {
		width: 10px;
		height: 10px;
		background: #10b981;
		border-radius: 50%;
		box-shadow: 0 0 8px #10b981;
	}
	.counter-chip {
		font-size: 0.75rem;
		background: rgba(255, 255, 255, 0.08);
		padding: 0.2rem 0.5rem;
		border-radius: 4px;
		color: #94a3b8;
	}
	.rx-btn {
		display: flex;
		align-items: center;
		gap: 0.4rem;
		background: rgba(168, 85, 247, 0.15);
		color: #c084fc;
		border: 1px solid rgba(168, 85, 247, 0.3);
		padding: 0.35rem 0.75rem;
		border-radius: 6px;
		font-size: 0.8125rem;
		cursor: pointer;
		transition: all 0.2s;
	}
	.rx-btn.active {
		background: #a855f7;
		color: #fff;
		border-color: #a855f7;
		box-shadow: 0 0 12px rgba(168, 85, 247, 0.4);
	}
	.topbar-actions {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}
	.topbar-actions button {
		display: flex;
		align-items: center;
		gap: 0.4rem;
		padding: 0.35rem 0.65rem;
		font-size: 0.8125rem;
		border-radius: 6px;
		cursor: pointer;
		border: 1px solid rgba(255, 255, 255, 0.1);
		background: rgba(255, 255, 255, 0.05);
		color: #f1f5f9;
	}
	.queue-badge {
		background: #f59e0b;
		color: #000;
		font-size: 0.6875rem;
		font-weight: 800;
		padding: 0.1rem 0.35rem;
		border-radius: 10px;
	}
	kbd {
		background: rgba(0, 0, 0, 0.4);
		border: 1px solid rgba(255, 255, 255, 0.2);
		border-radius: 3px;
		font-size: 0.625rem;
		padding: 0.1rem 0.3rem;
		color: #94a3b8;
	}

	/* PHARMACY DRAWER */
	.rx-metadata-drawer {
		background: #1e1b4b;
		border-bottom: 1px solid #4338ca;
		padding: 0.75rem 1.25rem;
	}
	.rx-header {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		margin-bottom: 0.5rem;
	}
	.rx-header strong {
		color: #c7d2fe;
		font-size: 0.875rem;
	}
	.rx-header p {
		font-size: 0.6875rem;
		color: #818cf8;
		margin: 0;
	}
	.rx-grid {
		display: grid;
		grid-template-columns: repeat(4, 1fr);
		gap: 0.75rem;
	}
	.input-group label {
		font-size: 0.6875rem;
		color: #a5b4fc;
		margin-bottom: 0.2rem;
		display: block;
	}
	.input-group input {
		width: 100%;
		padding: 0.35rem 0.5rem;
		background: rgba(0, 0, 0, 0.3);
		border: 1px solid rgba(255, 255, 255, 0.15);
		border-radius: 4px;
		color: #fff;
		font-size: 0.8125rem;
	}

	/* MAIN 2-PANE POS LAYOUT */
	.pos-layout {
		display: grid;
		grid-template-columns: 1.1fr 1fr;
		flex: 1;
		min-height: 0;
		overflow: hidden;
	}

	/* LEFT PANE */
	.pos-left-pane {
		display: flex;
		flex-direction: column;
		border-right: 1px solid rgba(255, 255, 255, 0.08);
		background: #0f172a;
		overflow-y: auto;
		padding: 1rem;
		gap: 1rem;
	}

	.search-hero {
		position: relative;
	}
	.search-input-wrapper {
		position: relative;
		display: flex;
		align-items: center;
	}
	.scanner-icon {
		position: absolute;
		left: 0.875rem;
		font-size: 1.25rem;
		color: #38bdf8;
	}
	.pos-search-box {
		width: 100%;
		padding: 0.875rem 1rem 0.875rem 2.75rem;
		background: #1e293b;
		border: 2px solid #38bdf8;
		border-radius: 8px;
		color: #fff;
		font-size: 0.9375rem;
		box-shadow: 0 0 15px rgba(56, 189, 248, 0.15);
		outline: none;
	}
	.pos-search-box:focus {
		border-color: #60a5fa;
		box-shadow: 0 0 20px rgba(56, 189, 248, 0.3);
	}
	.search-spinner {
		position: absolute;
		right: 0.875rem;
	}

	.search-dropdown-results {
		position: absolute;
		top: calc(100% + 4px);
		left: 0;
		right: 0;
		background: #1e293b;
		border: 1px solid #38bdf8;
		border-radius: 8px;
		max-height: 350px;
		overflow-y: auto;
		z-index: 50;
		box-shadow: 0 10px 25px rgba(0, 0, 0, 0.5);
	}
	.search-result-item {
		width: 100%;
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 0.625rem 0.875rem;
		border: none;
		background: none;
		border-bottom: 1px solid rgba(255, 255, 255, 0.05);
		color: #f1f5f9;
		cursor: pointer;
		text-align: left;
	}
	.search-result-item:hover {
		background: rgba(56, 189, 248, 0.15);
	}
	.res-title {
		display: flex;
		align-items: center;
		gap: 0.4rem;
		font-size: 0.875rem;
	}
	.rx-pill {
		background: #dc2626;
		color: #fff;
		font-size: 0.625rem;
		padding: 0.1rem 0.3rem;
		border-radius: 3px;
		font-weight: 700;
	}
	.rack-pill {
		background: #0284c7;
		color: #fff;
		font-size: 0.625rem;
		padding: 0.1rem 0.3rem;
		border-radius: 3px;
	}
	.res-generic {
		font-size: 0.75rem;
		color: #a5b4fc;
	}
	.res-meta {
		font-size: 0.6875rem;
		color: #94a3b8;
		display: flex;
		gap: 0.5rem;
		margin-top: 0.15rem;
	}
	.batch-count-pill {
		color: #34d399;
		font-weight: 600;
	}
	.res-sell-price {
		font-size: 1rem;
		font-weight: 800;
		color: #38bdf8;
	}
	.res-mrp {
		font-size: 0.6875rem;
		text-decoration: line-through;
		color: #94a3b8;
	}

	/* QUICK PICKS */
	.quick-picks-section {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}
	.section-title {
		display: flex;
		align-items: baseline;
		gap: 0.5rem;
		font-size: 0.8125rem;
		font-weight: 700;
		color: #cbd5e1;
	}
	.section-title .subtitle {
		font-size: 0.6875rem;
		color: #64748b;
		font-weight: normal;
	}
	.quick-picks-grid {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 0.5rem;
	}
	.quick-tile {
		background: #1e293b;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 8px;
		padding: 0.625rem;
		display: flex;
		flex-direction: column;
		justify-content: space-between;
		text-align: left;
		cursor: pointer;
		transition: transform 0.1s, border-color 0.1s;
		min-height: 80px;
	}
	.quick-tile:hover {
		border-color: #38bdf8;
		transform: translateY(-2px);
		background: #243248;
	}
	.quick-tile-header {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		gap: 0.25rem;
	}
	.quick-name {
		font-size: 0.8125rem;
		font-weight: 600;
		color: #f1f5f9;
		line-height: 1.2;
	}
	.rx-badge {
		background: #ef4444;
		color: #fff;
		font-size: 0.5625rem;
		padding: 0.05rem 0.25rem;
		border-radius: 2px;
	}
	.quick-salt {
		font-size: 0.6875rem;
		color: #a5b4fc;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.quick-tile-footer {
		display: flex;
		align-items: baseline;
		gap: 0.35rem;
		margin-top: 0.25rem;
	}
	.quick-price {
		font-size: 0.9375rem;
		font-weight: 800;
		color: #38bdf8;
	}
	.quick-mrp {
		font-size: 0.6875rem;
		text-decoration: line-through;
		color: #94a3b8;
	}
	.quick-rack {
		font-size: 0.625rem;
		color: #0284c7;
		margin-left: auto;
	}

	.customer-strip {
		margin-top: auto;
		display: flex;
		align-items: center;
		gap: 0.625rem;
		padding: 0.625rem;
		background: #1e293b;
		border-radius: 6px;
	}
	.customer-strip label {
		font-size: 0.75rem;
		color: #94a3b8;
		white-space: nowrap;
	}
	.customer-strip select {
		flex: 1;
		padding: 0.35rem;
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: 4px;
		color: #fff;
		font-size: 0.8125rem;
	}

	/* RIGHT PANE: CART TABLE & TOTALS */
	.pos-right-pane {
		display: flex;
		flex-direction: column;
		background: #090e17;
		height: 100%;
		overflow: hidden;
	}
	.cart-table-wrapper {
		flex: 1;
		overflow-y: auto;
		padding: 0.5rem;
	}
	.cart-table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.8125rem;
	}
	.cart-table th {
		padding: 0.5rem;
		background: #1e293b;
		color: #94a3b8;
		font-size: 0.6875rem;
		text-transform: uppercase;
		font-weight: 700;
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
	}
	.cart-row td {
		padding: 0.5rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.04);
		vertical-align: middle;
	}
	.cart-item-name {
		display: flex;
		align-items: center;
		gap: 0.3rem;
		font-weight: 600;
	}
	.rx-tiny-pill {
		background: #ef4444;
		color: #fff;
		font-size: 0.5625rem;
		padding: 0.05rem 0.2rem;
		border-radius: 2px;
	}
	.cart-item-salt {
		font-size: 0.6875rem;
		color: #a5b4fc;
	}
	.cart-item-rack {
		font-size: 0.625rem;
		color: #38bdf8;
	}
	.batch-dropdown {
		background: #1e293b;
		border: 1px solid rgba(255, 255, 255, 0.15);
		color: #f1f5f9;
		font-size: 0.6875rem;
		border-radius: 4px;
		padding: 0.2rem;
		max-width: 180px;
	}
	.single-batch-pill {
		font-size: 0.6875rem;
		background: rgba(56, 189, 248, 0.1);
		color: #38bdf8;
		padding: 0.15rem 0.35rem;
		border-radius: 3px;
	}
	.no-batch-muted {
		font-size: 0.6875rem;
		color: #64748b;
	}
	.qty-control {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.25rem;
	}
	.qty-btn {
		width: 24px;
		height: 24px;
		background: #1e293b;
		border: 1px solid rgba(255, 255, 255, 0.15);
		color: #fff;
		border-radius: 3px;
		cursor: pointer;
		font-weight: 700;
	}
	.qty-display {
		min-width: 20px;
		text-align: center;
		font-weight: 700;
	}
	.item-rate {
		font-weight: 700;
		color: #f8fafc;
	}
	.item-mrp-struck {
		font-size: 0.6875rem;
		text-decoration: line-through;
		color: #64748b;
	}
	.item-line-total {
		font-weight: 800;
		color: #34d399;
	}
	.item-delete-btn {
		background: none;
		border: none;
		color: #ef4444;
		cursor: pointer;
		font-size: 1rem;
		margin-left: 0.25rem;
	}
	.empty-cart-state {
		padding: 3rem 1rem;
		text-align: center;
		color: #64748b;
	}
	.empty-icon {
		font-size: 2.5rem;
		margin-bottom: 0.5rem;
	}

	/* CHECKOUT CARD */
	.cart-checkout-card {
		background: #1e293b;
		border-top: 2px solid rgba(255, 255, 255, 0.08);
		padding: 0.875rem 1rem;
		display: flex;
		flex-direction: column;
		gap: 0.625rem;
	}
	.savings-banner {
		background: rgba(16, 185, 129, 0.15);
		border: 1px solid rgba(16, 185, 129, 0.3);
		border-radius: 6px;
		padding: 0.4rem 0.75rem;
		display: flex;
		align-items: center;
		gap: 0.5rem;
		color: #34d399;
		font-size: 0.75rem;
	}
	.summary-breakdown {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}
	.summary-line {
		display: flex;
		justify-content: space-between;
		font-size: 0.8125rem;
		color: #94a3b8;
	}
	.grand-total-line {
		font-size: 1.125rem;
		font-weight: 800;
		color: #fff;
		border-top: 1px dashed rgba(255, 255, 255, 0.15);
		padding-top: 0.4rem;
		margin-top: 0.25rem;
	}
	.grand-total-amount {
		color: #38bdf8;
		font-size: 1.5rem;
	}

	.checkout-actions {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 0.625rem;
	}
	.cash-pay-btn, .split-pay-btn {
		display: flex;
		align-items: center;
		gap: 0.625rem;
		padding: 0.75rem;
		border-radius: 8px;
		border: none;
		cursor: pointer;
		transition: transform 0.1s;
		text-align: left;
	}
	.cash-pay-btn {
		background: #059669;
		color: #fff;
	}
	.cash-pay-btn:hover:not(:disabled) {
		background: #10b981;
		transform: translateY(-1px);
	}
	.split-pay-btn {
		background: #2563eb;
		color: #fff;
	}
	.split-pay-btn:hover:not(:disabled) {
		background: #3b82f6;
		transform: translateY(-1px);
	}
	.btn-icon {
		font-size: 1.5rem;
	}
	.btn-text strong {
		display: block;
		font-size: 0.875rem;
	}
	.btn-text span {
		font-size: 0.6875rem;
		opacity: 0.85;
	}
	.cash-pay-btn:disabled, .split-pay-btn:disabled {
		opacity: 0.4;
		cursor: not-allowed;
	}

	/* MODALS */
	.pos-modal-overlay {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.75);
		backdrop-filter: blur(4px);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 100;
	}
	.pos-modal-card {
		background: #1e293b;
		border: 1px solid rgba(255, 255, 255, 0.15);
		border-radius: 12px;
		width: 100%;
		max-width: 480px;
		box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.7);
		overflow: hidden;
	}
	.pos-modal-header {
		padding: 1rem;
		background: #0f172a;
		display: flex;
		justify-content: space-between;
		align-items: center;
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
	}
	.pos-modal-header h3 {
		margin: 0;
		font-size: 1rem;
	}
	.close-modal-btn {
		background: none;
		border: none;
		color: #94a3b8;
		font-size: 1.25rem;
		cursor: pointer;
	}
	.pos-modal-body {
		padding: 1.25rem;
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}
	.modal-due-card {
		background: #0f172a;
		padding: 1rem;
		border-radius: 8px;
		text-align: center;
		border: 1px solid rgba(56, 189, 248, 0.2);
	}
	.due-label {
		display: block;
		font-size: 0.75rem;
		color: #94a3b8;
	}
	.due-val {
		font-size: 2rem;
		font-weight: 800;
		color: #38bdf8;
	}
	.mode-pills {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 0.5rem;
	}
	.mode-pill {
		padding: 0.5rem;
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.1);
		color: #cbd5e1;
		border-radius: 6px;
		cursor: pointer;
		font-size: 0.8125rem;
		font-weight: 600;
	}
	.mode-pill.active {
		background: #0284c7;
		border-color: #38bdf8;
		color: #fff;
	}
	.big-cash-input {
		width: 100%;
		padding: 0.75rem;
		font-size: 1.5rem;
		font-weight: 800;
		background: #0f172a;
		border: 2px solid #34d399;
		color: #fff;
		border-radius: 8px;
		margin-top: 0.25rem;
		text-align: center;
	}
	.quick-cash-chips {
		display: flex;
		gap: 0.5rem;
		margin-top: 0.5rem;
	}
	.quick-cash-chips button {
		flex: 1;
		padding: 0.4rem;
		background: #334155;
		border: 1px solid rgba(255, 255, 255, 0.1);
		color: #f1f5f9;
		border-radius: 4px;
		font-size: 0.75rem;
		font-weight: 600;
		cursor: pointer;
	}
	.change-return-box {
		margin-top: 0.75rem;
		padding: 0.75rem;
		background: rgba(16, 185, 129, 0.1);
		border-radius: 6px;
		display: flex;
		justify-content: space-between;
		align-items: center;
	}
	.change-lbl {
		font-size: 0.875rem;
		font-weight: 600;
		color: #34d399;
	}
	.change-val {
		font-size: 1.25rem;
		font-weight: 800;
		color: #34d399;
	}
	.split-tender-rows {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}
	.split-row {
		display: flex;
		gap: 0.5rem;
	}
	.split-row select {
		flex: 1;
		background: #0f172a;
		color: #fff;
		border: 1px solid rgba(255, 255, 255, 0.15);
		border-radius: 4px;
		padding: 0.4rem;
	}
	.split-row input {
		width: 120px;
		background: #0f172a;
		color: #fff;
		border: 1px solid rgba(255, 255, 255, 0.15);
		border-radius: 4px;
		padding: 0.4rem;
		font-weight: 700;
	}
	.split-del {
		background: none;
		border: none;
		color: #ef4444;
		cursor: pointer;
	}
	.add-split-btn {
		background: none;
		border: 1px dashed rgba(255, 255, 255, 0.2);
		color: #38bdf8;
		padding: 0.4rem;
		border-radius: 4px;
		cursor: pointer;
		font-size: 0.75rem;
	}
	.split-balance-bar {
		display: flex;
		justify-content: space-between;
		font-size: 0.8125rem;
		background: #0f172a;
		padding: 0.5rem;
		border-radius: 4px;
	}
	.pos-modal-footer {
		padding: 1rem;
		background: #0f172a;
		display: flex;
		justify-content: flex-end;
		gap: 0.75rem;
		border-top: 1px solid rgba(255, 255, 255, 0.08);
	}
	.cancel-btn {
		padding: 0.5rem 1rem;
		background: #334155;
		color: #cbd5e1;
		border: none;
		border-radius: 6px;
		cursor: pointer;
	}
	.complete-btn {
		padding: 0.5rem 1.25rem;
		background: #10b981;
		color: #fff;
		font-weight: 700;
		border: none;
		border-radius: 6px;
		cursor: pointer;
	}
	.complete-btn:disabled {
		opacity: 0.4;
		cursor: not-allowed;
	}

	/* HELD LIST */
	.held-list {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}
	.held-item {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 0.75rem;
		background: #0f172a;
		border-radius: 6px;
		border: 1px solid rgba(255, 255, 255, 0.08);
	}
	.held-meta {
		display: block;
		font-size: 0.6875rem;
		color: #94a3b8;
		margin-top: 0.2rem;
	}
	.resume-btn {
		background: #0284c7;
		color: #fff;
		border: none;
		padding: 0.35rem 0.75rem;
		border-radius: 4px;
		cursor: pointer;
		font-size: 0.8125rem;
	}
	.del-held-btn {
		background: none;
		border: none;
		color: #ef4444;
		cursor: pointer;
		font-size: 1rem;
	}

	/* THERMAL RECEIPT PREVIEW */
	.receipt-modal {
		max-width: 380px;
	}
	.thermal-receipt-view {
		background: #fff;
		color: #000;
		padding: 1rem;
		font-family: 'Courier New', Courier, monospace;
		font-size: 0.75rem;
		line-height: 1.3;
		max-height: 450px;
		overflow-y: auto;
	}
	.th-header {
		text-align: center;
	}
	.th-header h4 {
		margin: 0;
		font-size: 0.875rem;
	}
	.th-header p {
		margin: 0.1rem 0;
	}
	.th-divider {
		overflow: hidden;
		white-space: nowrap;
		margin: 0.25rem 0;
	}
	.th-meta-row {
		display: flex;
		justify-content: space-between;
		font-size: 0.6875rem;
	}
	.th-head-row {
		display: grid;
		grid-template-columns: 2fr 1fr 1fr 1fr;
		font-weight: 700;
	}
	.th-item-line {
		margin: 0.3rem 0;
	}
	.th-item-batch {
		font-size: 0.625rem;
		color: #444;
	}
	.th-item-calc {
		display: flex;
		justify-content: space-between;
	}
	.th-tot-row {
		display: flex;
		justify-content: space-between;
		margin: 0.2rem 0;
	}
	.th-tot-row.grand {
		font-size: 0.875rem;
		border-top: 1px dashed #000;
		border-bottom: 1px dashed #000;
		padding: 0.25rem 0;
	}
	.th-savings {
		text-align: center;
		font-weight: 700;
		margin: 0.3rem 0;
	}
	.th-footer {
		text-align: center;
		margin-top: 0.5rem;
	}

	@media print {
		:global(body *) {
			visibility: hidden;
		}
		#printable-pos-receipt, #printable-pos-receipt * {
			visibility: visible;
		}
		#printable-pos-receipt {
			position: absolute;
			left: 0;
			top: 0;
			width: 80mm;
		}
	}
</style>
