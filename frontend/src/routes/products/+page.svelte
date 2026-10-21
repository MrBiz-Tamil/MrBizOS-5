<script lang="ts">
	import { onMount } from 'svelte';
	import { api, type Product, type ProductBatch, type Unit, type DigitalLicenseKey } from '$lib/api';

	let products = $state<Product[]>([]);
	let units = $state<Unit[]>([]);
	let loading = $state(true);
	let error = $state('');

	// Add Product Modal state
	let showAddModal = $state(false);
	let newProductType = $state<'PHYSICAL' | 'DIGITAL' | 'SERVICE'>('PHYSICAL');
	let newFulfillmentMode = $state<'NONE' | 'UNLIMITED' | 'LICENSE_POOL' | 'EXTERNAL_URL'>('UNLIMITED');
	let newDigitalUrl = $state('');
	let newName = $state('');
	let newSKU = $state('');
	let newHSN = $state('');
	let newCategory = $state('General');
	let newTaxRate = $state('18');
	let newSellingPrice = $state('');
	let newPurchasePrice = $state('');
	let newMRP = $state('');
	let newDiscountRate = $state('0');
	let newBarcode = $state('');
	let newGenericName = $state('');
	let newRackLocation = $state('');
	let newIsRxRequired = $state(false);
	let newUnitId = $state('');
	let newInitialStock = $state('0');
	let newMinStock = $state('5');
	let modalError = $state('');
	let modalSubmitting = $state(false);

	// License Pool Modal state
	let showLicenseModal = $state(false);
	let selectedProduct = $state<Product | null>(null);
	let licenseKeys = $state<DigitalLicenseKey[]>([]);
	let licenseLoading = $state(false);
	let newKeysInput = $state('');
	let licenseImporting = $state(false);
	let licenseMsg = $state('');

	// Product Batch Management Modal state (DPCO Pharmacy & Retail Expiry)
	let showBatchModal = $state(false);
	let batchProduct = $state<Product | null>(null);
	let productBatches = $state<ProductBatch[]>([]);
	let batchLoading = $state(false);
	let newBatchNo = $state('');
	let newExpiryDate = $state('');
	let newBatchMRP = $state('');
	let newBatchSellingPrice = $state('');
	let newBatchPurchasePrice = $state('');
	let newBatchStock = $state('0');
	let batchSubmitting = $state(false);
	let batchMsg = $state('');
	let batchError = $state('');

	onMount(async () => {
		await loadData();
	});

	async function loadData() {
		loading = true;
		try {
			const [prods, uList] = await Promise.all([api.listProducts(), api.listUnits()]);
			products = prods;
			units = uList;
			if (units.length > 0) {
				newUnitId = units[0].id;
			}
		} catch (err: any) {
			error = err.message || 'Failed to load products';
		} finally {
			loading = false;
		}
	}

	async function handleCreateProduct(e: Event) {
		e.preventDefault();
		modalError = '';
		if (!newName || !newSKU || !newSellingPrice) {
			modalError = 'Please provide product name, SKU, and selling price';
			return;
		}

		// Pick or default unit
		let selUnit = units.find((u) => u.id === newUnitId);
		if (!selUnit && units.length > 0) {
			selUnit = units[0];
			newUnitId = selUnit.id;
		}
		const baseUnit = units.find((u) => u.code === selUnit?.base_unit_code) || selUnit;
		const baseUnitId = baseUnit?.id || newUnitId;

		modalSubmitting = true;
		try {
			await api.createProduct({
				name: newName.trim(),
				sku: newSKU.trim().toUpperCase(),
				hsn_sac: newHSN.trim(),
				category: newCategory.trim(),
				product_type: newProductType,
				fulfillment_mode: newProductType === 'DIGITAL' ? newFulfillmentMode : 'NONE',
				digital_asset_url: newDigitalUrl.trim(),
				tax_rate: newTaxRate,
				selling_price: newSellingPrice,
				purchase_price: newPurchasePrice || '0',
				mrp: newProductType === 'PHYSICAL' && newMRP ? newMRP : (newProductType === 'PHYSICAL' ? newSellingPrice : undefined),
				discount_rate: newProductType === 'PHYSICAL' ? (newDiscountRate || '0') : '0',
				barcode: newProductType === 'PHYSICAL' ? newBarcode.trim() : undefined,
				generic_name: newProductType === 'PHYSICAL' ? newGenericName.trim() : undefined,
				rack_location: newProductType === 'PHYSICAL' ? newRackLocation.trim() : undefined,
				is_prescription_required: newProductType === 'PHYSICAL' ? newIsRxRequired : false,
				purchase_unit_id: newUnitId || (units[0]?.id ?? ''),
				sales_unit_id: newUnitId || (units[0]?.id ?? ''),
				stock_unit_id: newUnitId || (units[0]?.id ?? ''),
				base_unit_id: baseUnitId || (units[0]?.id ?? ''),
				current_stock: newProductType === 'PHYSICAL' ? (newInitialStock || '0') : '0',
				min_stock: newProductType === 'PHYSICAL' ? (newMinStock || '0') : '0'
			});

			showAddModal = false;
			resetForm();
			await loadData();
		} catch (err: any) {
			modalError = err.message || 'Failed to create product';
		} finally {
			modalSubmitting = false;
		}
	}

	function resetForm() {
		newName = '';
		newSKU = '';
		newHSN = '';
		newSellingPrice = '';
		newPurchasePrice = '';
		newMRP = '';
		newDiscountRate = '0';
		newBarcode = '';
		newGenericName = '';
		newRackLocation = '';
		newIsRxRequired = false;
		newDigitalUrl = '';
		newProductType = 'PHYSICAL';
		newFulfillmentMode = 'UNLIMITED';
	}

	async function openLicenseModal(prod: Product) {
		selectedProduct = prod;
		showLicenseModal = true;
		licenseMsg = '';
		newKeysInput = '';
		await loadLicenseKeys(prod.id);
	}

	async function loadLicenseKeys(productId: string) {
		licenseLoading = true;
		try {
			licenseKeys = await api.listProductLicenses(productId);
		} catch (err: any) {
			licenseMsg = 'Failed to load keys: ' + (err.message || '');
		} finally {
			licenseLoading = false;
		}
	}

	async function handleImportKeys() {
		if (!selectedProduct || !newKeysInput.trim()) return;
		const lines = newKeysInput
			.split('\n')
			.map((l) => l.trim())
			.filter((l) => l.length > 0);

		if (lines.length === 0) return;

		licenseImporting = true;
		licenseMsg = '';
		try {
			const res = await api.addProductLicenses(selectedProduct.id, lines);
			licenseMsg = `Successfully imported ${res.imported} new license key(s)!`;
			newKeysInput = '';
			await loadLicenseKeys(selectedProduct.id);
		} catch (err: any) {
			licenseMsg = 'Import error: ' + (err.message || '');
		} finally {
			licenseImporting = false;
		}
	}

	// Batch Modal Handlers
	async function openBatchModal(prod: Product) {
		batchProduct = prod;
		showBatchModal = true;
		batchMsg = '';
		batchError = '';
		newBatchNo = '';
		newExpiryDate = '';
		newBatchMRP = prod.mrp || prod.selling_price || '';
		newBatchSellingPrice = prod.selling_price || '';
		newBatchPurchasePrice = prod.purchase_price || '0';
		newBatchStock = '0';
		await loadBatches(prod.id);
	}

	async function loadBatches(productId: string) {
		batchLoading = true;
		try {
			productBatches = await api.listProductBatches(productId);
		} catch (err: any) {
			batchError = 'Failed to load batches: ' + (err.message || '');
		} finally {
			batchLoading = false;
		}
	}

	async function handleCreateBatch(e: Event) {
		e.preventDefault();
		if (!batchProduct) return;
		if (!newBatchNo.trim() || !newExpiryDate) {
			batchError = 'Please provide Batch Number and Expiry Date';
			return;
		}

		batchSubmitting = true;
		batchError = '';
		batchMsg = '';
		try {
			await api.createProductBatch(batchProduct.id, {
				batch_number: newBatchNo.trim().toUpperCase(),
				expiry_date: newExpiryDate,
				mrp: newBatchMRP || newBatchSellingPrice,
				selling_price: newBatchSellingPrice,
				purchase_price: newBatchPurchasePrice || '0',
				current_stock: newBatchStock || '0'
			});
			batchMsg = `Batch ${newBatchNo.trim().toUpperCase()} registered successfully!`;
			newBatchNo = '';
			newExpiryDate = '';
			newBatchStock = '0';
			await loadBatches(batchProduct.id);
			await loadData();
		} catch (err: any) {
			batchError = err.message || 'Failed to create batch';
		} finally {
			batchSubmitting = false;
		}
	}

	function formatCurrency(val: string | number): string {
		const num = typeof val === 'string' ? parseFloat(val) : val;
		return new Intl.NumberFormat('en-IN', { style: 'currency', currency: 'INR' }).format(num);
	}
</script>

<svelte:head>
	<title>Products & Digital Master - SaaS Platform</title>
</svelte:head>

<div style="display: flex; flex-direction: column; gap: 1.5rem;">
	<!-- Header -->
	<div style="display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 1rem;">
		<div>
			<h1 style="font-size: 1.75rem; margin-bottom: 0.25rem;">Products & Pricing Master</h1>
			<p style="font-size: 0.875rem; color: var(--text-secondary);">
				Manage physical items, digital licenses, software keys, and billable services.
			</p>
		</div>
		<button onclick={() => { resetForm(); showAddModal = true; }} class="btn btn-primary">
			+ Add Product / Service
		</button>
	</div>

	<!-- Products Table -->
	<div class="table-container card" style="padding: 0;">
		<table class="data-table">
			<thead>
				<tr>
					<th>Product Name</th>
					<th>Type</th>
					<th>SKU</th>
					<th>HSN/SAC</th>
					<th>Category</th>
					<th class="text-right">Price</th>
					<th class="text-right">Tax</th>
					<th class="text-right">Stock / Capacity</th>
					<th>Status / Actions</th>
				</tr>
			</thead>
			<tbody>
				{#if loading}
					<tr>
						<td colspan="9" class="text-center" style="padding: 3rem; color: var(--text-muted);">
							Loading products catalog...
						</td>
					</tr>
				{:else if error}
					<tr>
						<td colspan="9" class="text-center" style="padding: 2rem; color: var(--rose);">
							⚠️ {error}
						</td>
					</tr>
				{:else if products.length === 0}
					<tr>
						<td colspan="9" class="text-center" style="padding: 3rem; color: var(--text-muted);">
							No products registered yet. Click "+ Add Product / Service" to start.
						</td>
					</tr>
				{:else}
					{#each products as prod}
						{@const isDigital = prod.product_type === 'DIGITAL'}
						{@const isService = prod.product_type === 'SERVICE'}
						{@const isLowStock = !isDigital && !isService && parseFloat(prod.current_stock) <= parseFloat(prod.min_stock) && parseFloat(prod.min_stock) > 0}
						<tr>
							<td style="font-weight: 600;">
								<div style="display: flex; align-items: flex-start; gap: 0.5rem;">
									<span style="font-size: 1.1rem; margin-top: 0.1rem;">
										{#if isDigital}
											💻
										{:else if isService}
											🛠️
										{:else}
											📦
										{/if}
									</span>
									<div>
										<div style="display: flex; align-items: center; gap: 0.4rem;">
											<span>{prod.name}</span>
											{#if prod.is_prescription_required}
												<span class="badge" style="background: rgba(244, 63, 94, 0.15); color: #fb7185; font-size: 0.68rem; padding: 0.1rem 0.35rem; border: 1px solid rgba(244, 63, 94, 0.3);">Rx</span>
											{/if}
										</div>
										{#if prod.generic_name || prod.rack_location || prod.barcode}
											<div style="font-size: 0.72rem; color: var(--text-muted); font-weight: 400; display: flex; flex-wrap: wrap; gap: 0.4rem; align-items: center; margin-top: 0.2rem;">
												{#if prod.generic_name}
													<span style="color: #93c5fd;">🧬 {prod.generic_name}</span>
												{/if}
												{#if prod.rack_location}
													<span style="background: rgba(255,255,255,0.06); padding: 0.1rem 0.35rem; border-radius: 3px;">📍 Shelf: {prod.rack_location}</span>
												{/if}
												{#if prod.barcode}
													<span style="font-family: var(--font-mono); color: #a5b4fc;">||| {prod.barcode}</span>
												{/if}
											</div>
										{/if}
									</div>
								</div>
							</td>
							<td>
								{#if isDigital}
									<span class="badge" style="background: rgba(168, 85, 247, 0.15); color: #c084fc; border: 1px solid rgba(168, 85, 247, 0.3);">
										Digital
									</span>
								{:else if isService}
									<span class="badge" style="background: rgba(16, 185, 129, 0.15); color: #34d399; border: 1px solid rgba(16, 185, 129, 0.3);">
										Service
									</span>
								{:else}
									<span class="badge" style="background: rgba(59, 130, 246, 0.15); color: #60a5fa; border: 1px solid rgba(59, 130, 246, 0.3);">
										Physical
									</span>
								{/if}
							</td>
							<td style="font-family: var(--font-mono); color: #818cf8;">{prod.sku}</td>
							<td style="font-family: var(--font-mono);">{prod.hsn_sac || '-'}</td>
							<td>
								<span style="font-size: 0.75rem; background: rgba(255,255,255,0.06); padding: 0.2rem 0.5rem; border-radius: var(--radius-sm);">
									{prod.category}
								</span>
							</td>
							<td class="text-right font-mono" style="font-weight: 600;">
								<div>{formatCurrency(prod.selling_price)}</div>
								{#if prod.mrp && parseFloat(prod.mrp) > parseFloat(prod.selling_price)}
									<div style="font-size: 0.7rem; color: var(--text-muted); text-decoration: line-through;">
										MRP: {formatCurrency(prod.mrp)}
									</div>
								{/if}
							</td>
							<td class="text-right font-mono">{parseFloat(prod.tax_rate).toFixed(0)}% GST</td>
							<td class="text-right font-mono">
								{#if isDigital}
									<span style="color: #c084fc; font-size: 0.8125rem;">
										{prod.fulfillment_mode === 'LICENSE_POOL' ? 'Key Pool' : 'Unlimited'}
									</span>
								{:else if isService}
									<span style="color: #34d399; font-size: 0.8125rem;">On Demand</span>
								{:else}
									{parseFloat(prod.current_stock).toFixed(2)}
								{/if}
							</td>
							<td>
								<div style="display: flex; gap: 0.35rem; align-items: center; flex-wrap: wrap;">
									{#if isDigital && prod.fulfillment_mode === 'LICENSE_POOL'}
										<button onclick={() => openLicenseModal(prod)} class="btn btn-secondary" style="padding: 0.25rem 0.6rem; font-size: 0.75rem;">
											🔑 Key Vault
										</button>
									{:else if isDigital}
										<span class="badge" style="background: rgba(168, 85, 247, 0.1); color: #c084fc;">Instant</span>
									{:else if isService}
										<span class="badge" style="background: rgba(16, 185, 129, 0.1); color: #34d399;">Billable</span>
									{:else}
										<button onclick={() => openBatchModal(prod)} class="btn btn-secondary" style="padding: 0.25rem 0.6rem; font-size: 0.75rem; border-color: rgba(99, 102, 241, 0.35); color: #a5b4fc;" title="Manage DPCO Batches, Expiry Dates & Stock">
											🏷️ Batches
										</button>
										{#if isLowStock}
											<span class="badge" style="background: rgba(244, 63, 94, 0.15); color: #fb7185; border: 1px solid rgba(244, 63, 94, 0.3);">
												Low
											</span>
										{:else}
											<span class="badge" style="background: rgba(16, 185, 129, 0.15); color: #34d399;">
												In Stock
											</span>
										{/if}
									{/if}
								</div>
							</td>
						</tr>
					{/each}
				{/if}
			</tbody>
		</table>
	</div>
</div>

<!-- Add Product Modal -->
{#if showAddModal}
	<div class="modal-backdrop">
		<div class="modal-card animate-fade-in" style="max-width: 620px;">
			<div class="modal-header">
				<h3 style="font-size: 1.125rem;">Register New Product or Service</h3>
				<button onclick={() => (showAddModal = false)} style="background: none; border: none; color: var(--text-muted); cursor: pointer; font-size: 1.25rem;">✕</button>
			</div>
			<form onsubmit={handleCreateProduct}>
				<div class="modal-body" style="display: flex; flex-direction: column; gap: 1.25rem;">
					{#if modalError}
						<div style="background: rgba(244, 63, 94, 0.15); border: 1px solid rgba(244, 63, 94, 0.3); border-radius: var(--radius-md); padding: 0.75rem; color: #fb7185; font-size: 0.8125rem;">
							⚠️ {modalError}
						</div>
					{/if}

					<!-- Classification Segmented Picker -->
					<div>
						<div class="form-label" style="margin-bottom: 0.35rem;">Product Classification *</div>
						<div style="display: grid; grid-template-columns: repeat(3, 1fr); gap: 0.5rem;">
							<button
								type="button"
								onclick={() => (newProductType = 'PHYSICAL')}
								style="padding: 0.75rem; border-radius: var(--radius-md); border: 1px solid {newProductType === 'PHYSICAL' ? '#3b82f6' : 'rgba(255,255,255,0.1)'}; background: {newProductType === 'PHYSICAL' ? 'rgba(59, 130, 246, 0.15)' : 'rgba(255,255,255,0.02)'}; color: {newProductType === 'PHYSICAL' ? '#60a5fa' : 'var(--text-secondary)'}; cursor: pointer; display: flex; flex-direction: column; align-items: center; gap: 0.25rem;"
							>
								<span style="font-size: 1.25rem;">📦</span>
								<strong style="font-size: 0.8125rem;">Physical Good</strong>
								<span style="font-size: 0.7rem; color: var(--text-muted);">Warehouse stock</span>
							</button>

							<button
								type="button"
								onclick={() => { newProductType = 'DIGITAL'; newFulfillmentMode = 'LICENSE_POOL'; }}
								style="padding: 0.75rem; border-radius: var(--radius-md); border: 1px solid {newProductType === 'DIGITAL' ? '#a855f7' : 'rgba(255,255,255,0.1)'}; background: {newProductType === 'DIGITAL' ? 'rgba(168, 85, 247, 0.15)' : 'rgba(255,255,255,0.02)'}; color: {newProductType === 'DIGITAL' ? '#c084fc' : 'var(--text-secondary)'}; cursor: pointer; display: flex; flex-direction: column; align-items: center; gap: 0.25rem;"
							>
								<span style="font-size: 1.25rem;">💻</span>
								<strong style="font-size: 0.8125rem;">Digital Product</strong>
								<span style="font-size: 0.7rem; color: var(--text-muted);">License keys / download</span>
							</button>

							<button
								type="button"
								onclick={() => (newProductType = 'SERVICE')}
								style="padding: 0.75rem; border-radius: var(--radius-md); border: 1px solid {newProductType === 'SERVICE' ? '#10b981' : 'rgba(255,255,255,0.1)'}; background: {newProductType === 'SERVICE' ? 'rgba(16, 185, 129, 0.15)' : 'rgba(255,255,255,0.02)'}; color: {newProductType === 'SERVICE' ? '#34d399' : 'var(--text-secondary)'}; cursor: pointer; display: flex; flex-direction: column; align-items: center; gap: 0.25rem;"
							>
								<span style="font-size: 1.25rem;">🛠️</span>
								<strong style="font-size: 0.8125rem;">Service</strong>
								<span style="font-size: 0.7rem; color: var(--text-muted);">Billable contract / hours</span>
							</button>
						</div>
					</div>

					<!-- Digital Fulfillment Options -->
					{#if newProductType === 'DIGITAL'}
						<div style="background: rgba(168, 85, 247, 0.08); border: 1px solid rgba(168, 85, 247, 0.2); border-radius: var(--radius-md); padding: 1rem; display: flex; flex-direction: column; gap: 0.75rem;">
							<div class="form-group" style="margin-bottom: 0;">
								<label class="form-label" for="ful-mode">Digital Fulfillment Mode</label>
								<select id="ful-mode" class="form-select" bind:value={newFulfillmentMode}>
									<option value="LICENSE_POOL">🔑 License Key Pool (Pre-loaded serial codes issued on sale)</option>
									<option value="UNLIMITED">⚡ Instant Delivery / Unlimited Access (No key pool required)</option>
									<option value="EXTERNAL_URL">🌐 Download / Access Link (URL provided to buyer)</option>
								</select>
							</div>

							{#if newFulfillmentMode === 'EXTERNAL_URL'}
								<div class="form-group" style="margin-bottom: 0;">
									<label class="form-label" for="dig-url">Secure Asset Download URL</label>
									<input id="dig-url" type="url" class="form-input font-mono" bind:value={newDigitalUrl} placeholder="https://cdn.yourbrand.com/release/v1.0.zip" />
								</div>
							{/if}
						</div>
					{/if}

					<!-- Core Details -->
					<div class="form-group" style="margin-bottom: 0;">
						<label class="form-label" for="prod-name">Item / Service Name *</label>
						<input id="prod-name" type="text" class="form-input" bind:value={newName} placeholder="e.g. SaaS Annual License, Basmati Rice, or IT Consulting" required />
					</div>

					<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="prod-sku">SKU / Item Code *</label>
							<input id="prod-sku" type="text" class="form-input font-mono" bind:value={newSKU} placeholder="e.g. DIG-LIC-PRO or RICE-01" required />
						</div>

						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="prod-hsn">HSN / SAC Code</label>
							<input id="prod-hsn" type="text" class="form-input font-mono" bind:value={newHSN} placeholder={newProductType === 'DIGITAL' ? '998434 (Software)' : '10063010'} />
						</div>
					</div>

					<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="prod-unit">Billing Unit *</label>
							<select id="prod-unit" class="form-select" bind:value={newUnitId} required>
								{#each units as u}
									<option value={u.id}>{u.name} ({u.code})</option>
								{/each}
							</select>
						</div>

						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="prod-tax">GST Tax Rate</label>
							<select id="prod-tax" class="form-select" bind:value={newTaxRate}>
								<option value="0">0% (Exempt)</option>
								<option value="5">5% GST</option>
								<option value="12">12% GST</option>
								<option value="18">18% GST (Standard)</option>
								<option value="28">28% GST</option>
							</select>
						</div>
					</div>

					<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="prod-sell">Selling Price (₹) *</label>
							<input id="prod-sell" type="number" step="0.01" class="form-input font-mono" bind:value={newSellingPrice} placeholder="499.00" required />
						</div>

						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="prod-purch">Purchase / Base Cost (₹)</label>
							<input id="prod-purch" type="number" step="0.01" class="form-input font-mono" bind:value={newPurchasePrice} placeholder="250.00" />
						</div>
					</div>

					<!-- Physical Retail & Pharmacy Fields -->
					{#if newProductType === 'PHYSICAL'}
						<div style="display: flex; flex-direction: column; gap: 0.85rem; background: rgba(59, 130, 246, 0.05); padding: 0.85rem; border-radius: var(--radius-md); border: 1px solid rgba(59, 130, 246, 0.18);">
							<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">
								<div class="form-group" style="margin-bottom: 0;">
									<label class="form-label" for="prod-mrp">Maximum Retail Price (MRP ₹)</label>
									<input id="prod-mrp" type="number" step="0.01" class="form-input font-mono" bind:value={newMRP} placeholder="e.g. 550.00" />
								</div>
								<div class="form-group" style="margin-bottom: 0;">
									<label class="form-label" for="prod-disc">Default Discount (%)</label>
									<input id="prod-disc" type="number" step="0.1" class="form-input font-mono" bind:value={newDiscountRate} placeholder="0" />
								</div>
							</div>

							<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">
								<div class="form-group" style="margin-bottom: 0;">
									<label class="form-label" for="prod-barcode">Barcode / EAN-13 (POS Scanner)</label>
									<input id="prod-barcode" type="text" class="form-input font-mono" bind:value={newBarcode} placeholder="8901234567890" />
								</div>
								<div class="form-group" style="margin-bottom: 0;">
									<label class="form-label" for="prod-rack">Rack / Shelf Location</label>
									<input id="prod-rack" type="text" class="form-input" bind:value={newRackLocation} placeholder="Rack A-02, Shelf 3" />
								</div>
							</div>

							<div class="form-group" style="margin-bottom: 0;">
								<label class="form-label" for="prod-generic">Generic Salt / Molecule Name (Pharma / FMCG)</label>
								<input id="prod-generic" type="text" class="form-input" bind:value={newGenericName} placeholder="e.g. Paracetamol 650mg, Ibuprofen IP" />
							</div>

							<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">
								<div class="form-group" style="margin-bottom: 0;">
									<label class="form-label" for="prod-stock">Initial Physical Stock</label>
									<input id="prod-stock" type="number" step="any" class="form-input font-mono" bind:value={newInitialStock} placeholder="100" />
								</div>

								<div class="form-group" style="margin-bottom: 0;">
									<label class="form-label" for="prod-min">Reorder Alert Threshold</label>
									<input id="prod-min" type="number" step="any" class="form-input font-mono" bind:value={newMinStock} placeholder="10" />
								</div>
							</div>

							<div style="display: flex; align-items: center; gap: 0.5rem; padding-top: 0.25rem;">
								<input id="prod-rx" type="checkbox" bind:checked={newIsRxRequired} style="cursor: pointer; width: 16px; height: 16px;" />
								<label for="prod-rx" style="font-size: 0.8125rem; font-weight: 500; cursor: pointer; color: #fb7185;">
									⚕️ Prescription Required (Schedule H / H1 Drug Regulatory Flag)
								</label>
							</div>
						</div>
					{:else}
						<div style="font-size: 0.8125rem; color: var(--text-muted); background: rgba(255,255,255,0.03); padding: 0.75rem; border-radius: var(--radius-md);">
							ℹ️ <strong>Zero Warehouse Overhead</strong>: {newProductType === 'DIGITAL' ? 'Digital goods' : 'Services'} bypass physical stock depletion and will not trigger false out-of-stock alerts.
						</div>
					{/if}
				</div>
				<div class="modal-footer">
					<button type="button" onclick={() => (showAddModal = false)} class="btn btn-secondary">Cancel</button>
					<button type="submit" class="btn btn-primary" disabled={modalSubmitting}>
						{modalSubmitting ? 'Saving...' : 'Save Product'}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}

<!-- License Key Pool Modal -->
{#if showLicenseModal && selectedProduct}
	<div class="modal-backdrop">
		<div class="modal-card animate-fade-in" style="max-width: 650px;">
			<div class="modal-header">
				<div>
					<h3 style="font-size: 1.125rem; margin-bottom: 0.25rem;">
						🔑 License Vault: {selectedProduct.name}
					</h3>
					<p style="font-size: 0.75rem; color: var(--text-muted);">
						Manage activation keys and serial numbers for automatic issuance on checkout.
					</p>
				</div>
				<button onclick={() => (showLicenseModal = false)} style="background: none; border: none; color: var(--text-muted); cursor: pointer; font-size: 1.25rem;">✕</button>
			</div>

			<div class="modal-body" style="display: flex; flex-direction: column; gap: 1.25rem;">
				{#if licenseMsg}
					<div style="background: rgba(59, 130, 246, 0.15); border: 1px solid rgba(59, 130, 246, 0.3); border-radius: var(--radius-md); padding: 0.75rem; color: #93c5fd; font-size: 0.8125rem;">
						{licenseMsg}
					</div>
				{/if}

				<!-- Bulk Import Section -->
				<div style="background: rgba(255,255,255,0.02); border: 1px solid rgba(255,255,255,0.08); border-radius: var(--radius-md); padding: 1rem; display: flex; flex-direction: column; gap: 0.75rem;">
					<label class="form-label" for="add-keys" style="font-weight: 600;">
						Add New License Keys (Paste 1 per line)
					</label>
					<textarea
						id="add-keys"
						class="form-input font-mono"
						rows="3"
						bind:value={newKeysInput}
						placeholder="XXXXX-XXXXX-XXXXX-XXXX1&#10;XXXXX-XXXXX-XXXXX-XXXX2"
					></textarea>
					<div style="display: flex; justify-content: flex-end;">
						<button
							onclick={handleImportKeys}
							class="btn btn-primary"
							disabled={licenseImporting || !newKeysInput.trim()}
							style="font-size: 0.8125rem; padding: 0.4rem 1rem;"
						>
							{licenseImporting ? 'Importing...' : '📥 Import Keys into Vault'}
						</button>
					</div>
				</div>

				<!-- Existing Keys Table -->
				<div>
					<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.5rem;">
						<strong style="font-size: 0.875rem;">Keys in Vault ({licenseKeys.length})</strong>
						<span style="font-size: 0.75rem; color: #34d399;">
							{licenseKeys.filter((k) => k.status === 'AVAILABLE').length} Available
						</span>
					</div>

					<div style="max-height: 220px; overflow-y: auto; border: 1px solid rgba(255,255,255,0.08); border-radius: var(--radius-sm);">
						{#if licenseLoading}
							<div style="text-align: center; padding: 2rem; color: var(--text-muted); font-size: 0.8125rem;">
								Loading license keys...
							</div>
						{:else if licenseKeys.length === 0}
							<div style="text-align: center; padding: 2rem; color: var(--text-muted); font-size: 0.8125rem;">
								No keys in vault. Paste license codes above to initialize inventory.
							</div>
						{:else}
							<table class="data-table" style="margin: 0; font-size: 0.8125rem;">
								<thead>
									<tr>
										<th>License Code</th>
										<th>Status</th>
										<th>Allocated At</th>
									</tr>
								</thead>
								<tbody>
									{#each licenseKeys as k}
										<tr>
											<td class="font-mono" style="color: #818cf8;">{k.license_key}</td>
											<td>
												{#if k.status === 'AVAILABLE'}
													<span class="badge" style="background: rgba(16, 185, 129, 0.15); color: #34d399;">
														AVAILABLE
													</span>
												{:else if k.status === 'ALLOCATED'}
													<span class="badge" style="background: rgba(59, 130, 246, 0.15); color: #60a5fa;">
														ALLOCATED
													</span>
												{:else}
													<span class="badge" style="background: rgba(244, 63, 94, 0.15); color: #fb7185;">
														{k.status}
													</span>
												{/if}
											</td>
											<td style="color: var(--text-muted); font-size: 0.75rem;">
												{k.allocated_at ? new Date(k.allocated_at).toLocaleString() : '-'}
											</td>
										</tr>
									{/each}
								</tbody>
							</table>
						{/if}
					</div>
				</div>
			</div>

			<div class="modal-footer">
				<button type="button" onclick={() => (showLicenseModal = false)} class="btn btn-secondary">Close</button>
			</div>
		</div>
	</div>
{/if}

<!-- Product Batch & DPCO Price Modal -->
{#if showBatchModal && batchProduct}
	<div class="modal-backdrop">
		<div class="modal-card animate-fade-in" style="max-width: 700px;">
			<div class="modal-header">
				<div>
					<h3 style="font-size: 1.125rem; margin-bottom: 0.25rem;">
						🏷️ Batch & Price Vault: {batchProduct.name}
					</h3>
					<p style="font-size: 0.75rem; color: var(--text-muted);">
						Manage pharmaceutical & FMCG expiry batches, DPCO regulated MRPs, and unit stocks.
					</p>
				</div>
				<button onclick={() => (showBatchModal = false)} style="background: none; border: none; color: var(--text-muted); cursor: pointer; font-size: 1.25rem;">✕</button>
			</div>

			<div class="modal-body" style="display: flex; flex-direction: column; gap: 1.25rem;">
				{#if batchMsg}
					<div style="background: rgba(16, 185, 129, 0.15); border: 1px solid rgba(16, 185, 129, 0.3); border-radius: var(--radius-md); padding: 0.75rem; color: #34d399; font-size: 0.8125rem;">
						✓ {batchMsg}
					</div>
				{/if}
				{#if batchError}
					<div style="background: rgba(244, 63, 94, 0.15); border: 1px solid rgba(244, 63, 94, 0.3); border-radius: var(--radius-md); padding: 0.75rem; color: #fb7185; font-size: 0.8125rem;">
						⚠️ {batchError}
					</div>
				{/if}

				<!-- Register New Batch Form -->
				<form onsubmit={handleCreateBatch} style="background: rgba(255,255,255,0.02); border: 1px solid rgba(255,255,255,0.08); border-radius: var(--radius-md); padding: 1rem; display: flex; flex-direction: column; gap: 0.75rem;">
					<strong style="font-size: 0.875rem;">+ Register New Batch / Consignment</strong>
					<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem;">
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="batch-no">Batch Number *</label>
							<input id="batch-no" type="text" class="form-input font-mono" bind:value={newBatchNo} placeholder="e.g. B-2026-X1" required />
						</div>
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="batch-exp">Expiry Date (MM/YY or YYYY-MM-DD) *</label>
							<input id="batch-exp" type="date" class="form-input font-mono" bind:value={newExpiryDate} required />
						</div>
					</div>

					<div style="display: grid; grid-template-columns: 1fr 1fr 1fr; gap: 0.75rem;">
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="batch-mrp">Batch MRP (₹)</label>
							<input id="batch-mrp" type="number" step="0.01" class="form-input font-mono" bind:value={newBatchMRP} placeholder="e.g. 60.00" required />
						</div>
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="batch-sell">Selling Price (₹)</label>
							<input id="batch-sell" type="number" step="0.01" class="form-input font-mono" bind:value={newBatchSellingPrice} placeholder="e.g. 54.00" required />
						</div>
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="batch-stock">Batch Stock</label>
							<input id="batch-stock" type="number" step="any" class="form-input font-mono" bind:value={newBatchStock} placeholder="100" required />
						</div>
					</div>

					<div style="display: flex; justify-content: flex-end; margin-top: 0.25rem;">
						<button type="submit" class="btn btn-primary" disabled={batchSubmitting} style="font-size: 0.8125rem; padding: 0.4rem 1rem;">
							{batchSubmitting ? 'Saving...' : 'Add Batch'}
						</button>
					</div>
				</form>

				<!-- Existing Batches Table -->
				<div>
					<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.5rem;">
						<strong style="font-size: 0.875rem;">Active Batches ({productBatches.length})</strong>
						<span style="font-size: 0.75rem; color: var(--text-muted);">FIFO selection will consume nearest expiry first</span>
					</div>

					<div style="max-height: 220px; overflow-y: auto; border: 1px solid rgba(255,255,255,0.08); border-radius: var(--radius-sm);">
						{#if batchLoading}
							<div style="text-align: center; padding: 2rem; color: var(--text-muted); font-size: 0.8125rem;">
								Loading batches...
							</div>
						{:else if productBatches.length === 0}
							<div style="text-align: center; padding: 2rem; color: var(--text-muted); font-size: 0.8125rem;">
								No specific batches recorded. System is using master product rate.
							</div>
						{:else}
							<table class="data-table" style="margin: 0; font-size: 0.8125rem;">
								<thead>
									<tr>
										<th>Batch No</th>
										<th>Expiry Date</th>
										<th class="text-right">MRP</th>
										<th class="text-right">Selling Rate</th>
										<th class="text-right">Stock</th>
									</tr>
								</thead>
								<tbody>
									{#each productBatches as b}
										<tr>
											<td class="font-mono" style="color: #818cf8; font-weight: 600;">{b.batch_number}</td>
											<td class="font-mono">
												{b.expiry_date ? new Date(b.expiry_date).toLocaleDateString('en-GB', { month: 'short', year: 'numeric' }) : '-'}
											</td>
											<td class="text-right font-mono">{formatCurrency(b.mrp)}</td>
											<td class="text-right font-mono" style="color: #34d399;">{formatCurrency(b.selling_price)}</td>
											<td class="text-right font-mono" style="font-weight: 600;">{parseFloat(b.current_stock).toFixed(2)}</td>
										</tr>
									{/each}
								</tbody>
							</table>
						{/if}
					</div>
				</div>
			</div>

			<div class="modal-footer">
				<button type="button" onclick={() => (showBatchModal = false)} class="btn btn-secondary">Close</button>
			</div>
		</div>
	</div>
{/if}

