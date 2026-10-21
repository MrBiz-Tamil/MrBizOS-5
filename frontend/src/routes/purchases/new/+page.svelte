<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { api, type Supplier, type Product, type Unit, type Tenant } from '$lib/api';

	interface InwardItemRow {
		product_id: string;
		product_name: string;
		item_description: string;
		hsn_sac: string;
		batch_number: string;
		expiry_date: string;
		quantity: number;
		unit_id: string;
		unit_code: string;
		purchase_price: number;
		discount_rate: number;
		tax_rate: number;
		mrp: number;
		selling_price: number;
		rack_location: string;
	}

	// State
	let company = $state<Tenant | null>(null);
	let suppliers = $state<Supplier[]>([]);
	let products = $state<Product[]>([]);
	let units = $state<Unit[]>([]);
	let loading = $state(true);
	let submitting = $state(false);
	let error = $state('');

	// Header Bill details
	let selectedSupplierId = $state('');
	let billNumber = $state('');
	let billDate = $state(new Date().toISOString().slice(0, 10));
	let dueDate = $state('');
	let poNumber = $state('');
	let vehicleNumber = $state('');
	let notes = $state('');

	// Initial Payment
	let initialPayment = $state(0);
	let paymentMethod = $state('BANK_TRANSFER');
	let paymentRef = $state('');

	// Items Table
	let items = $state<InwardItemRow[]>([
		{
			product_id: '',
			product_name: '',
			item_description: '',
			hsn_sac: '',
			batch_number: '',
			expiry_date: '',
			quantity: 1,
			unit_id: '',
			unit_code: 'PCS',
			purchase_price: 0,
			discount_rate: 0,
			tax_rate: 18,
			mrp: 0,
			selling_price: 0,
			rack_location: ''
		}
	]);

	onMount(async () => {
		try {
			const [comp, supList, prodList, unitList] = await Promise.all([
				api.getCompany().catch(() => null),
				api.listSuppliers().catch(() => []),
				api.listProducts().catch(() => []),
				api.listUnits().catch(() => [])
			]);
			company = comp;
			suppliers = supList;
			products = prodList;
			units = unitList;

			// Set default due date to 30 days from today
			const d = new Date();
			d.setDate(d.getDate() + 30);
			dueDate = d.toISOString().slice(0, 10);
		} catch (err: any) {
			error = err.message || 'Failed to initialize inward bill';
		} finally {
			loading = false;
		}
	});

	let selectedSupplier = $derived(
		suppliers.find((s) => s.id === selectedSupplierId) || null
	);

	let isInterState = $derived(
		!!(
			company?.state &&
			selectedSupplier?.state &&
			company.state.trim().toUpperCase() !== selectedSupplier.state.trim().toUpperCase()
		)
	);

	function handleProductSelect(index: number, e: Event) {
		const select = e.target as HTMLSelectElement;
		const prodId = select.value;
		const prod = products.find((p) => p.id === prodId);
		if (!prod) return;

		items[index].product_id = prod.id;
		items[index].product_name = prod.name;
		items[index].item_description = prod.name;
		items[index].hsn_sac = prod.hsn_sac || '';
		items[index].purchase_price = parseFloat(prod.purchase_price || '0');
		items[index].tax_rate = parseFloat(prod.tax_rate || '18');
		items[index].mrp = parseFloat(prod.mrp || '0');
		items[index].selling_price = parseFloat(prod.selling_price || '0');
		items[index].rack_location = prod.rack_location || '';

		if (prod.purchase_unit_id) {
			items[index].unit_id = prod.purchase_unit_id;
			const u = units.find((x) => x.id === prod.purchase_unit_id);
			if (u) items[index].unit_code = u.code;
		}
	}

	function addItemRow() {
		const defaultUnit = units[0];
		items = [
			...items,
			{
				product_id: '',
				product_name: '',
				item_description: '',
				hsn_sac: '',
				batch_number: '',
				expiry_date: '',
				quantity: 1,
				unit_id: defaultUnit?.id || '',
				unit_code: defaultUnit?.code || 'PCS',
				purchase_price: 0,
				discount_rate: 0,
				tax_rate: 18,
				mrp: 0,
				selling_price: 0,
				rack_location: ''
			}
		];
	}

	function removeItemRow(index: number) {
		if (items.length <= 1) return;
		items = items.filter((_, i) => i !== index);
	}

	// Line calculations
	function calculateLineMath(item: InwardItemRow) {
		const gross = item.purchase_price * item.quantity;
		const discount = gross * (item.discount_rate / 100);
		const taxable = gross - discount;
		const tax = taxable * (item.tax_rate / 100);
		const total = taxable + tax;
		return { gross, discount, taxable, tax, total };
	}

	// Grand totals
	let computedTotals = $derived.by(() => {
		let subtotal = 0;
		let discountTotal = 0;
		let taxableTotal = 0;
		let taxTotal = 0;

		for (const it of items) {
			const m = calculateLineMath(it);
			subtotal += m.gross;
			discountTotal += m.discount;
			taxableTotal += m.taxable;
			taxTotal += m.tax;
		}

		const rawGrandTotal = taxableTotal + taxTotal;
		const grandTotal = Math.round(rawGrandTotal);
		const roundOff = grandTotal - rawGrandTotal;
		const balanceDue = Math.max(0, grandTotal - initialPayment);

		return {
			subtotal,
			discountTotal,
			taxableTotal,
			taxTotal,
			roundOff,
			grandTotal,
			balanceDue
		};
	});

	async function handleSubmit(e: Event) {
		e.preventDefault();
		if (!selectedSupplierId) {
			error = 'Please select a supplier';
			return;
		}
		if (!billNumber.trim()) {
			error = 'Supplier Bill Number is required';
			return;
		}

		// Validate items
		for (let i = 0; i < items.length; i++) {
			const it = items[i];
			if (!it.product_id) {
				error = `Row #${i + 1}: Please choose a product`;
				return;
			}
			if (it.quantity <= 0) {
				error = `Row #${i + 1}: Quantity must be greater than zero`;
				return;
			}
			if (it.purchase_price < 0) {
				error = `Row #${i + 1}: Purchase price cannot be negative`;
				return;
			}
		}

		submitting = true;
		error = '';

		try {
			await api.createPurchase({
				supplier_id: selectedSupplierId,
				bill_number: billNumber.trim(),
				bill_date: billDate,
				due_date: dueDate,
				vehicle_number: vehicleNumber.trim(),
				po_number: poNumber.trim(),
				notes: notes.trim(),
				initial_payment: initialPayment.toString(),
				payment_method: paymentMethod,
				payment_ref: paymentRef.trim(),
				items: items.map((it) => ({
					product_id: it.product_id,
					item_description: it.item_description || it.product_name,
					hsn_sac: it.hsn_sac,
					batch_number: it.batch_number,
					expiry_date: it.expiry_date || undefined,
					quantity: it.quantity,
					unit_id: it.unit_id || units[0]?.id,
					unit_code: it.unit_code,
					purchase_price: it.purchase_price,
					discount_rate: it.discount_rate,
					tax_rate: it.tax_rate,
					mrp: it.mrp,
					selling_price: it.selling_price,
					rack_location: it.rack_location
				}))
			});

			goto('/purchases');
		} catch (err: any) {
			error = err.message || 'Failed to save inward purchase bill';
		} finally {
			submitting = false;
		}
	}

	function formatCurrency(val: number) {
		return new Intl.NumberFormat('en-IN', {
			style: 'currency',
			currency: 'INR',
			maximumFractionDigits: 2
		}).format(val);
	}
</script>

<div class="new-purchase-view">
	<header class="header-section">
		<div>
			<a href="/purchases" class="back-link">← Back to Purchase Bills</a>
			<h1 class="page-title">New Inward Purchase Bill</h1>
			<p class="page-desc">Record vendor bill, auto-increment inventory stock, sync batches, and update payables.</p>
		</div>
		<div class="supplier-pill">
			{#if selectedSupplier}
				<span>📍 Supplier State: <strong>{selectedSupplier.state || 'Unspecified'}</strong></span>
				<span class="tax-tag" class:inter={isInterState}>
					{isInterState ? 'Inter-State (IGST)' : 'Intra-State (CGST + SGST)'}
				</span>
			{:else}
				<span class="text-muted">Select a supplier to determine GST split</span>
			{/if}
		</div>
	</header>

	{#if error}
		<div class="alert alert-error">{error}</div>
	{/if}

	<form onsubmit={handleSubmit} class="bill-form">
		<!-- Section 1: Vendor & Document Details -->
		<div class="panel">
			<h3 class="panel-title">1. Supplier & Invoice Details</h3>
			<div class="grid-3">
				<div class="form-group">
					<label for="supplier-select">Supplier / Vendor *</label>
					<div class="select-with-link">
						<select id="supplier-select" bind:value={selectedSupplierId} required class="input-ctrl">
							<option value="">-- Select Vendor --</option>
							{#each suppliers as s}
								<option value={s.id}>{s.name} {s.company_name ? `(${s.company_name})` : ''}</option>
							{/each}
						</select>
						<a href="/suppliers" target="_blank" class="add-sup-btn" title="Add New Supplier">+ New</a>
					</div>
				</div>
				<div class="form-group">
					<label for="bill-no">Vendor Bill / Invoice # *</label>
					<input
						id="bill-no"
						type="text"
						bind:value={billNumber}
						placeholder="e.g. INV-9842"
						required
						class="input-ctrl"
					/>
				</div>
				<div class="form-group">
					<label for="bill-date">Bill Date *</label>
					<input id="bill-date" type="date" bind:value={billDate} required class="input-ctrl" />
				</div>
				<div class="form-group">
					<label for="due-date">Payment Due Date</label>
					<input id="due-date" type="date" bind:value={dueDate} class="input-ctrl" />
				</div>
				<div class="form-group">
					<label for="po-no">Purchase Order (PO) #</label>
					<input id="po-no" type="text" bind:value={poNumber} placeholder="Optional PO Ref" class="input-ctrl" />
				</div>
				<div class="form-group">
					<label for="veh-no">Vehicle / Dispatch #</label>
					<input id="veh-no" type="text" bind:value={vehicleNumber} placeholder="e.g. MH-12-AB-1234" class="input-ctrl" />
				</div>
			</div>
		</div>

		<!-- Section 2: Inward Items Grid -->
		<div class="panel">
			<div class="panel-header-row">
				<h3 class="panel-title">2. Inward Stock Items & Batches</h3>
				<span class="stock-sync-notice">⚡ Inventory & batches auto-increment upon save</span>
			</div>

			<div class="items-table-wrapper">
				<table class="items-table">
					<thead>
						<tr>
							<th style="width: 250px;">Product Name *</th>
							<th style="width: 130px;">Batch # (Opt)</th>
							<th style="width: 130px;">Expiry Date</th>
							<th style="width: 90px;">Quantity</th>
							<th style="width: 100px;">Unit</th>
							<th style="width: 110px;">Purchase Rate (₹)</th>
							<th style="width: 90px;">Disc %</th>
							<th style="width: 90px;">GST %</th>
							<th style="width: 110px;">MRP (₹)</th>
							<th style="width: 110px;">Selling Rate</th>
							<th style="width: 120px; text-align: right;">Total (₹)</th>
							<th style="width: 50px;"></th>
						</tr>
					</thead>
					<tbody>
						{#each items as item, index}
							{@const math = calculateLineMath(item)}
							<tr>
								<td>
									<select
										value={item.product_id}
										onchange={(e) => handleProductSelect(index, e)}
										required
										class="table-input"
									>
										<option value="">-- Choose Product --</option>
										{#each products as p}
											<option value={p.id}>{p.name} ({p.sku})</option>
										{/each}
									</select>
								</td>
								<td>
									<input
										type="text"
										bind:value={item.batch_number}
										placeholder="e.g. BATCH-01"
										class="table-input"
									/>
								</td>
								<td>
									<input
										type="date"
										bind:value={item.expiry_date}
										class="table-input"
									/>
								</td>
								<td>
									<input
										type="number"
										step="0.001"
										min="0.001"
										bind:value={item.quantity}
										required
										class="table-input"
										style="text-align: right;"
									/>
								</td>
								<td>
									<select bind:value={item.unit_code} class="table-input">
										{#each units as u}
											<option value={u.code}>{u.code}</option>
										{/each}
									</select>
								</td>
								<td>
									<input
										type="number"
										step="0.01"
										min="0"
										bind:value={item.purchase_price}
										required
										class="table-input"
										style="text-align: right;"
									/>
								</td>
								<td>
									<input
										type="number"
										step="0.1"
										min="0"
										max="100"
										bind:value={item.discount_rate}
										class="table-input"
										style="text-align: right;"
									/>
								</td>
								<td>
									<select bind:value={item.tax_rate} class="table-input" style="text-align: right;">
										<option value={0}>0%</option>
										<option value={5}>5%</option>
										<option value={12}>12%</option>
										<option value={18}>18%</option>
										<option value={28}>28%</option>
									</select>
								</td>
								<td>
									<input
										type="number"
										step="0.01"
										min="0"
										bind:value={item.mrp}
										placeholder="MRP"
										class="table-input"
										style="text-align: right;"
									/>
								</td>
								<td>
									<input
										type="number"
										step="0.01"
										min="0"
										bind:value={item.selling_price}
										placeholder="Sales Rate"
										class="table-input"
										style="text-align: right;"
									/>
								</td>
								<td style="text-align: right; font-weight: 700; color: #f8fafc;">
									{formatCurrency(math.total)}
								</td>
								<td style="text-align: center;">
									<button
										type="button"
										class="del-btn"
										disabled={items.length <= 1}
										onclick={() => removeItemRow(index)}
										title="Delete row"
									>
										✕
									</button>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>

			<div style="margin-top: 1rem;">
				<button type="button" class="btn btn-secondary btn-sm" onclick={addItemRow}>
					+ Add Another Item
				</button>
			</div>
		</div>

		<!-- Section 3: Summary & Settlement -->
		<div class="grid-2">
			<!-- Payment on Entry -->
			<div class="panel">
				<h3 class="panel-title">3. Immediate Payment (Settlement)</h3>
				<p class="text-muted" style="font-size: 0.8125rem; margin-bottom: 1rem;">
					If you made an instant payment to the vendor upon receiving this bill, enter details here:
				</p>
				<div class="form-group">
					<label for="pay-init">Initial Amount Paid (₹)</label>
					<input
						id="pay-init"
						type="number"
						step="0.01"
						min="0"
						max={computedTotals.grandTotal}
						bind:value={initialPayment}
						class="input-ctrl"
						placeholder="0.00"
					/>
				</div>
				{#if initialPayment > 0}
					<div class="grid-2" style="margin-top: 0.75rem;">
						<div class="form-group">
							<label for="pay-meth">Payment Mode</label>
							<select id="pay-meth" bind:value={paymentMethod} class="input-ctrl">
								<option value="BANK_TRANSFER">Bank Transfer (NEFT/RTGS)</option>
								<option value="UPI">UPI / QR Code</option>
								<option value="CASH">Cash</option>
								<option value="CHEQUE">Cheque</option>
							</select>
						</div>
						<div class="form-group">
							<label for="pay-reference">Reference / UTR #</label>
							<input id="pay-reference" type="text" bind:value={paymentRef} placeholder="UTR / Transaction Ref" class="input-ctrl" />
						</div>
					</div>
				{/if}
				<div class="form-group" style="margin-top: 0.75rem;">
					<label for="bill-notes">Notes / Remarks</label>
					<textarea id="bill-notes" bind:value={notes} rows="2" placeholder="Optional internal notes on stock inward" class="input-ctrl"></textarea>
				</div>
			</div>

			<!-- Live Totals Panel -->
			<div class="panel totals-panel">
				<h3 class="panel-title">Financial Summary</h3>
				<div class="totals-breakdown">
					<div class="tot-row">
						<span>Subtotal (Gross):</span>
						<span>{formatCurrency(computedTotals.subtotal)}</span>
					</div>
					{#if computedTotals.discountTotal > 0}
						<div class="tot-row text-green">
							<span>Discount:</span>
							<span>-{formatCurrency(computedTotals.discountTotal)}</span>
						</div>
					{/if}
					<div class="tot-row">
						<span>Taxable Value:</span>
						<span>{formatCurrency(computedTotals.taxableTotal)}</span>
					</div>
					{#if isInterState}
						<div class="tot-row">
							<span>IGST (Integrated Tax):</span>
							<span>{formatCurrency(computedTotals.taxTotal)}</span>
						</div>
					{:else}
						<div class="tot-row">
							<span>CGST (Central Tax):</span>
							<span>{formatCurrency(computedTotals.taxTotal / 2)}</span>
						</div>
						<div class="tot-row">
							<span>SGST (State Tax):</span>
							<span>{formatCurrency(computedTotals.taxTotal / 2)}</span>
						</div>
					{/if}
					{#if computedTotals.roundOff !== 0}
						<div class="tot-row text-muted">
							<span>Round Off:</span>
							<span>{formatCurrency(computedTotals.roundOff)}</span>
						</div>
					{/if}
					<div class="tot-row grand-row">
						<strong>Grand Total:</strong>
						<strong class="text-blue" style="font-size: 1.25rem;">
							{formatCurrency(computedTotals.grandTotal)}
						</strong>
					</div>
					<div class="tot-row">
						<span>Paid Now:</span>
						<span class="text-green">{formatCurrency(initialPayment)}</span>
					</div>
					<div class="tot-row due-row">
						<strong>Net Balance Due:</strong>
						<strong class="text-red" style="font-size: 1.125rem;">
							{formatCurrency(computedTotals.balanceDue)}
						</strong>
					</div>
				</div>

				<div class="submit-wrap">
					<button type="submit" class="btn btn-primary btn-block" disabled={submitting}>
						{submitting ? 'Saving Inward Bill...' : '💾 Save & Update Inventory'}
					</button>
				</div>
			</div>
		</div>
	</form>
</div>

<style>
	.new-purchase-view {
		padding: 2rem;
		max-width: 1440px;
		margin: 0 auto;
		display: flex;
		flex-direction: column;
		gap: 1.5rem;
	}

	.header-section {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		flex-wrap: wrap;
		gap: 1rem;
	}

	.back-link {
		font-size: 0.8125rem;
		color: #818cf8;
		text-decoration: none;
		display: inline-block;
		margin-bottom: 0.25rem;
	}
	.back-link:hover {
		text-decoration: underline;
	}

	.page-title {
		font-size: 1.75rem;
		font-weight: 700;
		color: #f8fafc;
		margin: 0;
	}

	.page-desc {
		font-size: 0.875rem;
		color: #94a3b8;
		margin-top: 0.25rem;
	}

	.supplier-pill {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		background: rgba(30, 41, 59, 0.6);
		padding: 0.5rem 1rem;
		border-radius: 8px;
		border: 1px solid rgba(255, 255, 255, 0.08);
		font-size: 0.8125rem;
		color: #cbd5e1;
	}

	.tax-tag {
		background: rgba(34, 197, 94, 0.15);
		color: #4ade80;
		padding: 0.2rem 0.5rem;
		border-radius: 4px;
		font-weight: 700;
		font-size: 0.75rem;
	}
	.tax-tag.inter {
		background: rgba(59, 130, 246, 0.15);
		color: #60a5fa;
	}

	.panel {
		background: rgba(30, 41, 59, 0.6);
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 12px;
		padding: 1.5rem;
		margin-bottom: 1.5rem;
	}

	.panel-title {
		font-size: 1.125rem;
		font-weight: 600;
		color: #f1f5f9;
		margin: 0 0 1rem 0;
	}

	.panel-header-row {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 1rem;
	}

	.stock-sync-notice {
		font-size: 0.75rem;
		font-weight: 600;
		color: #fbbf24;
		background: rgba(245, 158, 11, 0.1);
		padding: 0.25rem 0.625rem;
		border-radius: 20px;
	}

	.grid-3 {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
		gap: 1rem;
	}

	.grid-2 {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
		gap: 1.5rem;
	}

	.form-group {
		display: flex;
		flex-direction: column;
		gap: 0.375rem;
	}

	.form-group label {
		font-size: 0.8125rem;
		font-weight: 500;
		color: #cbd5e1;
	}

	.input-ctrl {
		padding: 0.625rem 0.875rem;
		background: rgba(15, 23, 42, 0.6);
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: 6px;
		color: #f8fafc;
		font-size: 0.875rem;
		outline: none;
	}
	.input-ctrl:focus {
		border-color: #6366f1;
	}

	.select-with-link {
		display: flex;
		gap: 0.5rem;
	}
	.select-with-link select {
		flex: 1;
	}

	.add-sup-btn {
		display: inline-flex;
		align-items: center;
		padding: 0 0.75rem;
		background: rgba(99, 102, 241, 0.15);
		color: #a5b4fc;
		border: 1px solid rgba(99, 102, 241, 0.3);
		border-radius: 6px;
		text-decoration: none;
		font-size: 0.8125rem;
		font-weight: 600;
	}

	/* Items Table */
	.items-table-wrapper {
		overflow-x: auto;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 8px;
	}

	.items-table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.8125rem;
	}

	.items-table th {
		background: rgba(15, 23, 42, 0.7);
		padding: 0.625rem 0.75rem;
		color: #94a3b8;
		font-weight: 600;
		text-align: left;
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
		white-space: nowrap;
	}

	.items-table td {
		padding: 0.5rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.05);
	}

	.table-input {
		width: 100%;
		padding: 0.375rem 0.5rem;
		background: rgba(15, 23, 42, 0.5);
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 4px;
		color: #f8fafc;
		font-size: 0.8125rem;
		outline: none;
	}
	.table-input:focus {
		border-color: #6366f1;
		background: #0f172a;
	}

	.del-btn {
		background: transparent;
		border: none;
		color: #f87171;
		cursor: pointer;
		font-size: 1rem;
		padding: 0.25rem;
		border-radius: 4px;
	}
	.del-btn:disabled {
		opacity: 0.3;
		cursor: not-allowed;
	}

	/* Totals Breakdown */
	.totals-panel {
		display: flex;
		flex-direction: column;
		justify-content: space-between;
	}

	.totals-breakdown {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		font-size: 0.875rem;
	}

	.tot-row {
		display: flex;
		justify-content: space-between;
		color: #cbd5e1;
	}

	.grand-row {
		border-top: 1px solid rgba(255, 255, 255, 0.1);
		border-bottom: 1px solid rgba(255, 255, 255, 0.1);
		padding: 0.5rem 0;
		font-size: 1.125rem;
		color: #f8fafc;
	}

	.due-row {
		padding-top: 0.25rem;
	}

	.submit-wrap {
		margin-top: 1.5rem;
	}

	.btn {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 0.5rem;
		padding: 0.75rem 1.5rem;
		border-radius: 8px;
		font-size: 0.875rem;
		font-weight: 600;
		cursor: pointer;
		border: none;
		text-decoration: none;
		transition: all 0.15s ease;
	}

	.btn-primary {
		background: #6366f1;
		color: #ffffff;
	}
	.btn-primary:hover {
		background: #4f46e5;
	}

	.btn-secondary {
		background: rgba(255, 255, 255, 0.08);
		color: #e2e8f0;
		border: 1px solid rgba(255, 255, 255, 0.15);
	}

	.btn-sm {
		padding: 0.375rem 0.75rem;
		font-size: 0.75rem;
	}

	.btn-block {
		width: 100%;
	}

	.text-red {
		color: #f87171;
	}
	.text-blue {
		color: #60a5fa;
	}
	.text-green {
		color: #4ade80;
	}
	.text-muted {
		color: #94a3b8;
	}

	.alert-error {
		background: rgba(239, 68, 68, 0.15);
		border: 1px solid rgba(239, 68, 68, 0.3);
		color: #f87171;
		padding: 0.75rem 1rem;
		border-radius: 8px;
		font-size: 0.875rem;
	}
</style>
