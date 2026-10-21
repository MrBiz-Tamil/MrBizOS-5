<script lang="ts">
	import { onMount } from 'svelte';
	import {
		api,
		type CreditNote,
		type DebitNote,
		type Customer,
		type Supplier,
		type Product
	} from '$lib/api';

	let activeTab = $state<'credit_notes' | 'debit_notes'>('credit_notes');
	let creditNotes = $state<CreditNote[]>([]);
	let debitNotes = $state<DebitNote[]>([]);
	let customers = $state<Customer[]>([]);
	let suppliers = $state<Supplier[]>([]);
	let products = $state<Product[]>([]);
	let loading = $state(true);
	let error = $state('');

	// Credit Note Modal State
	let showCNModal = $state(false);
	let cnCustomerName = $state('');
	let cnCustomerGSTIN = $state('');
	let cnInvoiceNumber = $state('');
	let cnReasonCode = $state('SALES_RETURN');
	let cnReasonDescription = $state('');
	let cnRestoreInventory = $state(true);
	let cnItems = $state<Array<{
		product_id?: string;
		item_description: string;
		batch_number: string;
		hsn_sac: string;
		quantity: number;
		unit_code: string;
		unit_price: number;
		gst_rate: number;
	}>>([
		{ item_description: '', batch_number: '', hsn_sac: '', quantity: 1, unit_code: 'PCS', unit_price: 0, gst_rate: 18 }
	]);
	let cnSubmitting = $state(false);
	let cnModalError = $state('');

	// Debit Note Modal State
	let showDNModal = $state(false);
	let dnSupplierName = $state('');
	let dnSupplierGSTIN = $state('');
	let dnBillNumber = $state('');
	let dnReasonCode = $state('PURCHASE_RETURN');
	let dnReasonDescription = $state('');
	let dnDeductInventory = $state(true);
	let dnItems = $state<Array<{
		product_id?: string;
		item_description: string;
		batch_number: string;
		hsn_sac: string;
		quantity: number;
		unit_code: string;
		unit_price: number;
		gst_rate: number;
	}>>([
		{ item_description: '', batch_number: '', hsn_sac: '', quantity: 1, unit_code: 'PCS', unit_price: 0, gst_rate: 18 }
	]);
	let dnSubmitting = $state(false);
	let dnModalError = $state('');

	// Details Modal
	let selectedCN = $state<CreditNote | null>(null);
	let selectedDN = $state<DebitNote | null>(null);

	onMount(async () => {
		await loadData();
	});

	async function loadData() {
		loading = true;
		error = '';
		try {
			const [cnList, dnList, custList, suppList, prodList] = await Promise.all([
				api.listCreditNotes(),
				api.listDebitNotes(),
				api.listCustomers(),
				api.listSuppliers(),
				api.listProducts()
			]);
			creditNotes = cnList;
			debitNotes = dnList;
			customers = custList;
			suppliers = suppList;
			products = prodList;
		} catch (err: any) {
			error = err.message || 'Failed to load credit and debit notes';
		} finally {
			loading = false;
		}
	}

	function addCNItem() {
		cnItems.push({ item_description: '', batch_number: '', hsn_sac: '', quantity: 1, unit_code: 'PCS', unit_price: 0, gst_rate: 18 });
	}

	function removeCNItem(idx: number) {
		if (cnItems.length > 1) {
			cnItems.splice(idx, 1);
		}
	}

	function handleProductSelectCN(idx: number, pId: string) {
		const prod = products.find((p) => p.id === pId);
		if (prod) {
			cnItems[idx].product_id = prod.id;
			cnItems[idx].item_description = prod.name;
			cnItems[idx].hsn_sac = prod.hsn_sac || '';
			cnItems[idx].unit_price = parseFloat(prod.selling_price || '0');
			cnItems[idx].gst_rate = parseFloat(prod.tax_rate || '18');
		}
	}

	function addDNItem() {
		dnItems.push({ item_description: '', batch_number: '', hsn_sac: '', quantity: 1, unit_code: 'PCS', unit_price: 0, gst_rate: 18 });
	}

	function removeDNItem(idx: number) {
		if (dnItems.length > 1) {
			dnItems.splice(idx, 1);
		}
	}

	function handleProductSelectDN(idx: number, pId: string) {
		const prod = products.find((p) => p.id === pId);
		if (prod) {
			dnItems[idx].product_id = prod.id;
			dnItems[idx].item_description = prod.name;
			dnItems[idx].hsn_sac = prod.hsn_sac || '';
			dnItems[idx].unit_price = parseFloat(prod.purchase_price || prod.selling_price || '0');
			dnItems[idx].gst_rate = parseFloat(prod.tax_rate || '18');
		}
	}

	async function submitCreditNote() {
		cnSubmitting = true;
		cnModalError = '';
		try {
			await api.createCreditNote({
				customer_name: cnCustomerName.trim() || 'Direct Consumer',
				customer_gstin: cnCustomerGSTIN.trim() || undefined,
				reason_code: cnReasonCode,
				reason_description: cnReasonDescription.trim() || undefined,
				restore_inventory: cnRestoreInventory,
				items: cnItems.map((it) => ({
					product_id: it.product_id || undefined,
					item_description: it.item_description.trim() || 'Returned item',
					batch_number: it.batch_number.trim() || undefined,
					hsn_sac: it.hsn_sac.trim() || undefined,
					quantity: Number(it.quantity) || 1,
					unit_code: it.unit_code || 'PCS',
					unit_price: Number(it.unit_price) || 0,
					gst_rate: Number(it.gst_rate) || 0
				}))
			});
			showCNModal = false;
			await loadData();
		} catch (err: any) {
			cnModalError = err.message || 'Failed to issue credit note';
		} finally {
			cnSubmitting = false;
		}
	}

	async function submitDebitNote() {
		dnSubmitting = true;
		dnModalError = '';
		try {
			await api.createDebitNote({
				supplier_name: dnSupplierName.trim() || 'Direct Vendor',
				supplier_gstin: dnSupplierGSTIN.trim() || undefined,
				reason_code: dnReasonCode,
				reason_description: dnReasonDescription.trim() || undefined,
				deduct_inventory: dnDeductInventory,
				items: dnItems.map((it) => ({
					product_id: it.product_id || undefined,
					item_description: it.item_description.trim() || 'Returned goods',
					batch_number: it.batch_number.trim() || undefined,
					hsn_sac: it.hsn_sac.trim() || undefined,
					quantity: Number(it.quantity) || 1,
					unit_code: it.unit_code || 'PCS',
					unit_price: Number(it.unit_price) || 0,
					gst_rate: Number(it.gst_rate) || 0
				}))
			});
			showDNModal = false;
			await loadData();
		} catch (err: any) {
			dnModalError = err.message || 'Failed to issue debit note';
		} finally {
			dnSubmitting = false;
		}
	}

	function formatCurrency(val: string | number | undefined): string {
		if (!val && val !== 0) return '₹0.00';
		const num = typeof val === 'string' ? parseFloat(val) : val;
		if (isNaN(num)) return '₹0.00';
		return new Intl.NumberFormat('en-IN', { style: 'currency', currency: 'INR' }).format(num);
	}

	function formatDate(val: string | undefined): string {
		if (!val) return '';
		return new Date(val).toLocaleDateString('en-IN', { day: '2-digit', month: 'short', year: 'numeric' });
	}
</script>

<svelte:head>
	<title>Statutory Returns & Credit/Debit Notes - SaaS Billing</title>
</svelte:head>

<div style="display: flex; flex-direction: column; gap: 1.5rem;">
	<!-- Header -->
	<div style="display: flex; justify-content: space-between; align-items: flex-start; flex-wrap: wrap; gap: 1rem;">
		<div>
			<h1 style="font-size: 1.75rem; display: flex; align-items: center; gap: 0.5rem;">
				<span>🔄</span> Statutory Returns, Credit & Debit Notes
			</h1>
			<p style="font-size: 0.875rem; color: var(--text-secondary);">
				GST Section 34 compliant credit and debit notes with automatic inventory restoration and reversing double-entry journal vouchers.
			</p>
		</div>

		<div style="display: flex; gap: 0.75rem;">
			<button class="btn btn-primary" onclick={() => (showCNModal = true)}>
				+ Issue Credit Note (Sales Return)
			</button>
			<button class="btn btn-secondary" onclick={() => (showDNModal = true)}>
				+ Issue Debit Note (Purchase Return)
			</button>
		</div>
	</div>

	<!-- Tab Switcher -->
	<div class="card" style="padding: 0.4rem; display: flex; gap: 0.5rem; background: rgba(17, 24, 39, 0.8);">
		<button
			class="btn btn-sm"
			style="border-radius: var(--radius-md); background: {activeTab === 'credit_notes' ? 'var(--primary)' : 'transparent'}; color: {activeTab === 'credit_notes' ? '#fff' : 'var(--text-secondary)'}; font-weight: 600;"
			onclick={() => (activeTab = 'credit_notes')}
		>
			📥 Credit Notes / Sales Returns ({creditNotes.length})
		</button>
		<button
			class="btn btn-sm"
			style="border-radius: var(--radius-md); background: {activeTab === 'debit_notes' ? 'var(--primary)' : 'transparent'}; color: {activeTab === 'debit_notes' ? '#fff' : 'var(--text-secondary)'}; font-weight: 600;"
			onclick={() => (activeTab = 'debit_notes')}
		>
			📤 Debit Notes / Purchase Returns ({debitNotes.length})
		</button>
	</div>

	{#if error}
		<div style="background: rgba(244, 63, 94, 0.15); border: 1px solid rgba(244, 63, 94, 0.3); color: #fb7185; padding: 1rem; border-radius: var(--radius-md);">
			{error}
		</div>
	{/if}

	{#if loading}
		<div style="padding: 3rem; text-align: center; color: var(--text-muted);">
			Loading credit and debit notes...
		</div>
	{:else if activeTab === 'credit_notes'}
		<!-- Credit Notes Table -->
		<div class="card" style="padding: 1.25rem;">
			<table class="data-table">
				<thead>
					<tr>
						<th>Note Number</th>
						<th>Date</th>
						<th>Customer</th>
						<th>Original Invoice</th>
						<th>Reason</th>
						<th class="text-right">Taxable</th>
						<th class="text-right">GST Total</th>
						<th class="text-right">Grand Total</th>
						<th>Inventory</th>
						<th>Status</th>
						<th class="text-center">Action</th>
					</tr>
				</thead>
				<tbody>
					{#if creditNotes.length === 0}
						<tr>
							<td colspan="11" class="text-center" style="color: var(--text-muted); padding: 2rem;">
								No credit notes issued yet. Click "+ Issue Credit Note" to record a sales return or trade discount.
							</td>
						</tr>
					{:else}
						{#each creditNotes as cn}
							<tr>
								<td class="font-mono" style="color: #818cf8; font-weight: 700;">{cn.credit_note_number}</td>
								<td>{formatDate(cn.credit_note_date)}</td>
								<td style="font-weight: 600;">{cn.customer_name}</td>
								<td>{cn.invoice_number || '—'}</td>
								<td>
									<span style="font-size: 0.75rem; background: rgba(255,255,255,0.06); padding: 0.2rem 0.5rem; border-radius: var(--radius-sm);">
										{cn.reason_code.replace(/_/g, ' ')}
									</span>
								</td>
								<td class="text-right font-mono">{formatCurrency(cn.taxable_amount)}</td>
								<td class="text-right font-mono" style="color: #34d399;">
									{formatCurrency(parseFloat(cn.cgst_total || '0') + parseFloat(cn.sgst_total || '0') + parseFloat(cn.igst_total || '0'))}
								</td>
								<td class="text-right font-mono" style="font-weight: 700; color: #fb7185;">
									{formatCurrency(cn.grand_total)}
								</td>
								<td>
									{#if cn.restore_inventory}
										<span style="font-size: 0.6875rem; background: rgba(16, 185, 129, 0.15); color: #34d399; padding: 0.15rem 0.4rem; border-radius: var(--radius-sm); font-weight: 600;">RESTORED</span>
									{:else}
										<span style="font-size: 0.6875rem; background: rgba(255,255,255,0.06); color: var(--text-muted); padding: 0.15rem 0.4rem; border-radius: var(--radius-sm);">FINANCIAL ONLY</span>
									{/if}
								</td>
								<td>
									<span style="font-size: 0.6875rem; background: rgba(99, 102, 241, 0.15); color: #a5b4fc; padding: 0.15rem 0.4rem; border-radius: var(--radius-sm); font-weight: 600;">
										{cn.status}
									</span>
								</td>
								<td class="text-center">
									<button class="btn btn-ghost btn-sm" onclick={() => (selectedCN = cn)}>
										View Details
									</button>
								</td>
							</tr>
						{/each}
					{/if}
				</tbody>
			</table>
		</div>

	{:else if activeTab === 'debit_notes'}
		<!-- Debit Notes Table -->
		<div class="card" style="padding: 1.25rem;">
			<table class="data-table">
				<thead>
					<tr>
						<th>Note Number</th>
						<th>Date</th>
						<th>Vendor / Supplier</th>
						<th>Bill Number</th>
						<th>Reason</th>
						<th class="text-right">Taxable</th>
						<th class="text-right">GST Reversal</th>
						<th class="text-right">Grand Total</th>
						<th>Inventory</th>
						<th>Status</th>
						<th class="text-center">Action</th>
					</tr>
				</thead>
				<tbody>
					{#if debitNotes.length === 0}
						<tr>
							<td colspan="11" class="text-center" style="color: var(--text-muted); padding: 2rem;">
								No debit notes issued yet. Click "+ Issue Debit Note" to record a purchase return to vendor.
							</td>
						</tr>
					{:else}
						{#each debitNotes as dn}
							<tr>
								<td class="font-mono" style="color: #fb7185; font-weight: 700;">{dn.debit_note_number}</td>
								<td>{formatDate(dn.debit_note_date)}</td>
								<td style="font-weight: 600;">{dn.supplier_name}</td>
								<td>{dn.bill_number || '—'}</td>
								<td>
									<span style="font-size: 0.75rem; background: rgba(255,255,255,0.06); padding: 0.2rem 0.5rem; border-radius: var(--radius-sm);">
										{dn.reason_code.replace(/_/g, ' ')}
									</span>
								</td>
								<td class="text-right font-mono">{formatCurrency(dn.taxable_amount)}</td>
								<td class="text-right font-mono" style="color: #60a5fa;">
									{formatCurrency(parseFloat(dn.cgst_total || '0') + parseFloat(dn.sgst_total || '0') + parseFloat(dn.igst_total || '0'))}
								</td>
								<td class="text-right font-mono" style="font-weight: 700; color: #34d399;">
									{formatCurrency(dn.grand_total)}
								</td>
								<td>
									{#if dn.deduct_inventory}
										<span style="font-size: 0.6875rem; background: rgba(244, 63, 94, 0.15); color: #fb7185; padding: 0.15rem 0.4rem; border-radius: var(--radius-sm); font-weight: 600;">DEDUCTED</span>
									{:else}
										<span style="font-size: 0.6875rem; background: rgba(255,255,255,0.06); color: var(--text-muted); padding: 0.15rem 0.4rem; border-radius: var(--radius-sm);">FINANCIAL ONLY</span>
									{/if}
								</td>
								<td>
									<span style="font-size: 0.6875rem; background: rgba(99, 102, 241, 0.15); color: #a5b4fc; padding: 0.15rem 0.4rem; border-radius: var(--radius-sm); font-weight: 600;">
										{dn.status}
									</span>
								</td>
								<td class="text-center">
									<button class="btn btn-ghost btn-sm" onclick={() => (selectedDN = dn)}>
										View Details
									</button>
								</td>
							</tr>
						{/each}
					{/if}
				</tbody>
			</table>
		</div>
	{/if}

	<!-- Issue Credit Note Modal -->
	{#if showCNModal}
		<div style="position: fixed; inset: 0; background: rgba(0,0,0,0.75); backdrop-filter: blur(4px); display: flex; align-items: center; justify-content: center; z-index: 50; padding: 1.5rem;">
			<div class="card" style="width: 100%; max-width: 820px; max-height: 90vh; overflow-y: auto; padding: 1.75rem; border-color: var(--border-accent);">
				<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1.25rem;">
					<h3 style="font-size: 1.25rem; font-weight: 700;">Issue Statutory Credit Note (Sales Return)</h3>
					<button class="btn btn-ghost" onclick={() => (showCNModal = false)}>✕</button>
				</div>

				{#if cnModalError}
					<div style="background: rgba(244, 63, 94, 0.15); border: 1px solid rgba(244, 63, 94, 0.3); color: #fb7185; padding: 0.75rem; border-radius: var(--radius-md); font-size: 0.8125rem; margin-bottom: 1rem;">
						{cnModalError}
					</div>
				{/if}

				<form onsubmit={(e) => { e.preventDefault(); submitCreditNote(); }} style="display: flex; flex-direction: column; gap: 1rem;">
					<div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 1rem;">
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="cn-cust-name">Customer Name</label>
							<input id="cn-cust-name" type="text" class="form-input" placeholder="Customer or Business Name" bind:value={cnCustomerName} required />
						</div>
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="cn-cust-gstin">Customer GSTIN (Optional)</label>
							<input id="cn-cust-gstin" type="text" class="form-input font-mono" placeholder="29ABCDE1234F1Z5" bind:value={cnCustomerGSTIN} style="text-transform: uppercase;" />
						</div>
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="cn-reason">Statutory Reason Code (Sec 34)</label>
							<select id="cn-reason" class="form-select" bind:value={cnReasonCode}>
								<option value="SALES_RETURN">SALES RETURN (Physical Goods Returned)</option>
								<option value="POST_SALE_DISCOUNT">POST-SALE DISCOUNT / RATE REVISION</option>
								<option value="DEFICIENCY_IN_SERVICE">DEFICIENCY IN SERVICE</option>
								<option value="CORRECTION_IN_INVOICE">CORRECTION IN INVOICE</option>
								<option value="OTHER">OTHER COMMERCIAL REASON</option>
							</select>
						</div>
					</div>

					<div style="display: grid; grid-template-columns: 2fr 1fr; gap: 1rem; align-items: center;">
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="cn-desc">Reason Description / Remarks</label>
							<input id="cn-desc" type="text" class="form-input" placeholder="e.g. Returned 2 damaged pieces from batch B-101" bind:value={cnReasonDescription} />
						</div>
						<label style="display: flex; align-items: center; gap: 0.5rem; font-size: 0.875rem; cursor: pointer; margin-top: 1.25rem;">
							<input type="checkbox" bind:checked={cnRestoreInventory} />
							<span>Restore Stock Quantity</span>
						</label>
					</div>

					<!-- Line Items -->
					<div style="margin-top: 0.5rem;">
						<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.5rem;">
							<span class="form-label" style="margin-bottom: 0;">Return Line Items</span>
							<button type="button" class="btn btn-secondary btn-sm" onclick={() => addCNItem()}>+ Add Line</button>
						</div>

						<table class="data-table" style="font-size: 0.8125rem;">
							<thead>
								<tr>
									<th>Select Product</th>
									<th>Description</th>
									<th>Batch</th>
									<th style="width: 70px;">Qty</th>
									<th style="width: 100px;">Price</th>
									<th style="width: 80px;">GST %</th>
									<th></th>
								</tr>
							</thead>
							<tbody>
								{#each cnItems as item, idx}
									<tr>
										<td>
											<select class="form-select" onchange={(e) => handleProductSelectCN(idx, (e.target as HTMLSelectElement).value)}>
												<option value="">-- Custom Item --</option>
												{#each products as p}
													<option value={p.id}>{p.name}</option>
												{/each}
											</select>
										</td>
										<td>
											<input type="text" class="form-input" bind:value={item.item_description} placeholder="Item Description" required />
										</td>
										<td>
											<input type="text" class="form-input font-mono" bind:value={item.batch_number} placeholder="Batch #" />
										</td>
										<td>
											<input type="number" step="any" min="0.001" class="form-input font-mono" bind:value={item.quantity} required />
										</td>
										<td>
											<input type="number" step="any" min="0" class="form-input font-mono" bind:value={item.unit_price} required />
										</td>
										<td>
											<select class="form-select font-mono" bind:value={item.gst_rate}>
												<option value={0}>0%</option>
												<option value={5}>5%</option>
												<option value={12}>12%</option>
												<option value={18}>18%</option>
												<option value={28}>28%</option>
											</select>
										</td>
										<td class="text-center">
											{#if cnItems.length > 1}
												<button type="button" class="btn btn-ghost btn-sm" style="color: #fb7185;" onclick={() => removeCNItem(idx)}>✕</button>
											{/if}
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>

					<div style="display: flex; justify-content: flex-end; gap: 0.75rem; margin-top: 1rem;">
						<button type="button" class="btn btn-ghost" onclick={() => (showCNModal = false)}>Cancel</button>
						<button type="submit" class="btn btn-primary" disabled={cnSubmitting}>
							{cnSubmitting ? 'Posting Note...' : 'Issue Credit Note'}
						</button>
					</div>
				</form>
			</div>
		</div>
	{/if}

	<!-- Issue Debit Note Modal -->
	{#if showDNModal}
		<div style="position: fixed; inset: 0; background: rgba(0,0,0,0.75); backdrop-filter: blur(4px); display: flex; align-items: center; justify-content: center; z-index: 50; padding: 1.5rem;">
			<div class="card" style="width: 100%; max-width: 820px; max-height: 90vh; overflow-y: auto; padding: 1.75rem; border-color: var(--border-accent);">
				<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1.25rem;">
					<h3 style="font-size: 1.25rem; font-weight: 700;">Issue Statutory Debit Note (Vendor Return)</h3>
					<button class="btn btn-ghost" onclick={() => (showDNModal = false)}>✕</button>
				</div>

				{#if dnModalError}
					<div style="background: rgba(244, 63, 94, 0.15); border: 1px solid rgba(244, 63, 94, 0.3); color: #fb7185; padding: 0.75rem; border-radius: var(--radius-md); font-size: 0.8125rem; margin-bottom: 1rem;">
						{dnModalError}
					</div>
				{/if}

				<form onsubmit={(e) => { e.preventDefault(); submitDebitNote(); }} style="display: flex; flex-direction: column; gap: 1rem;">
					<div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 1rem;">
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="dn-supp-name">Supplier / Vendor Name</label>
							<input id="dn-supp-name" type="text" class="form-input" placeholder="Supplier Name" bind:value={dnSupplierName} required />
						</div>
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="dn-supp-gstin">Supplier GSTIN (Optional)</label>
							<input id="dn-supp-gstin" type="text" class="form-input font-mono" placeholder="29ABCDE1234F1Z5" bind:value={dnSupplierGSTIN} style="text-transform: uppercase;" />
						</div>
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="dn-reason">Reason Code</label>
							<select id="dn-reason" class="form-select" bind:value={dnReasonCode}>
								<option value="PURCHASE_RETURN">PURCHASE RETURN (Defective Goods Returned)</option>
								<option value="POST_PURCHASE_DISCOUNT">POST-PURCHASE DISCOUNT / RATE REDUCTION</option>
								<option value="DEFICIENCY_IN_GOODS">DEFICIENCY IN GOODS</option>
								<option value="OTHER">OTHER COMMERCIAL REASON</option>
							</select>
						</div>
					</div>

					<div style="display: grid; grid-template-columns: 2fr 1fr; gap: 1rem; align-items: center;">
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="dn-desc">Reason Description / Remarks</label>
							<input id="dn-desc" type="text" class="form-input" placeholder="e.g. Returning 5 damaged widgets to supplier" bind:value={dnReasonDescription} />
						</div>
						<label style="display: flex; align-items: center; gap: 0.5rem; font-size: 0.875rem; cursor: pointer; margin-top: 1.25rem;">
							<input type="checkbox" bind:checked={dnDeductInventory} />
							<span>Deduct Stock Quantity</span>
						</label>
					</div>

					<div style="margin-top: 0.5rem;">
						<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.5rem;">
							<span class="form-label" style="margin-bottom: 0;">Return Line Items</span>
							<button type="button" class="btn btn-secondary btn-sm" onclick={() => addDNItem()}>+ Add Line</button>
						</div>

						<table class="data-table" style="font-size: 0.8125rem;">
							<thead>
								<tr>
									<th>Select Product</th>
									<th>Description</th>
									<th>Batch</th>
									<th style="width: 70px;">Qty</th>
									<th style="width: 100px;">Price</th>
									<th style="width: 80px;">GST %</th>
									<th></th>
								</tr>
							</thead>
							<tbody>
								{#each dnItems as item, idx}
									<tr>
										<td>
											<select class="form-select" onchange={(e) => handleProductSelectDN(idx, (e.target as HTMLSelectElement).value)}>
												<option value="">-- Custom Item --</option>
												{#each products as p}
													<option value={p.id}>{p.name}</option>
												{/each}
											</select>
										</td>
										<td>
											<input type="text" class="form-input" bind:value={item.item_description} placeholder="Item Description" required />
										</td>
										<td>
											<input type="text" class="form-input font-mono" bind:value={item.batch_number} placeholder="Batch #" />
										</td>
										<td>
											<input type="number" step="any" min="0.001" class="form-input font-mono" bind:value={item.quantity} required />
										</td>
										<td>
											<input type="number" step="any" min="0" class="form-input font-mono" bind:value={item.unit_price} required />
										</td>
										<td>
											<select class="form-select font-mono" bind:value={item.gst_rate}>
												<option value={0}>0%</option>
												<option value={5}>5%</option>
												<option value={12}>12%</option>
												<option value={18}>18%</option>
												<option value={28}>28%</option>
											</select>
										</td>
										<td class="text-center">
											{#if dnItems.length > 1}
												<button type="button" class="btn btn-ghost btn-sm" style="color: #fb7185;" onclick={() => removeDNItem(idx)}>✕</button>
											{/if}
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>

					<div style="display: flex; justify-content: flex-end; gap: 0.75rem; margin-top: 1rem;">
						<button type="button" class="btn btn-ghost" onclick={() => (showDNModal = false)}>Cancel</button>
						<button type="submit" class="btn btn-primary" disabled={dnSubmitting}>
							{dnSubmitting ? 'Posting Note...' : 'Issue Debit Note'}
						</button>
					</div>
				</form>
			</div>
		</div>
	{/if}

	<!-- View Credit Note Details Modal -->
	{#if selectedCN}
		<div style="position: fixed; inset: 0; background: rgba(0,0,0,0.75); backdrop-filter: blur(4px); display: flex; align-items: center; justify-content: center; z-index: 50; padding: 1.5rem;">
			<div class="card" style="width: 100%; max-width: 650px; padding: 1.75rem; border-color: var(--border-accent);">
				<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1.25rem;">
					<h3 style="font-size: 1.25rem; font-weight: 700;">Credit Note #{selectedCN.credit_note_number}</h3>
					<button class="btn btn-ghost" onclick={() => (selectedCN = null)}>✕</button>
				</div>

				<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem; font-size: 0.875rem; margin-bottom: 1.25rem;">
					<div><span style="color: var(--text-muted);">Customer:</span> <strong>{selectedCN.customer_name}</strong></div>
					<div><span style="color: var(--text-muted);">GSTIN:</span> <strong>{selectedCN.customer_gstin || 'Unregistered'}</strong></div>
					<div><span style="color: var(--text-muted);">Date:</span> <strong>{formatDate(selectedCN.credit_note_date)}</strong></div>
					<div><span style="color: var(--text-muted);">Reason:</span> <strong>{selectedCN.reason_code}</strong></div>
					<div style="grid-column: span 2;"><span style="color: var(--text-muted);">Remarks:</span> {selectedCN.reason_description || 'None'}</div>
				</div>

				<div style="background: rgba(255,255,255,0.04); border-radius: var(--radius-md); padding: 1rem; display: flex; flex-direction: column; gap: 0.5rem; font-size: 0.875rem;">
					<div style="display: flex; justify-content: space-between;"><span>Taxable Amount:</span> <span class="font-mono">{formatCurrency(selectedCN.taxable_amount)}</span></div>
					<div style="display: flex; justify-content: space-between;"><span>CGST:</span> <span class="font-mono">{formatCurrency(selectedCN.cgst_total)}</span></div>
					<div style="display: flex; justify-content: space-between;"><span>SGST:</span> <span class="font-mono">{formatCurrency(selectedCN.sgst_total)}</span></div>
					<div style="display: flex; justify-content: space-between;"><span>IGST:</span> <span class="font-mono">{formatCurrency(selectedCN.igst_total)}</span></div>
					<div style="display: flex; justify-content: space-between; font-size: 1.125rem; font-weight: 700; border-top: 1px solid var(--border-subtle); padding-top: 0.5rem; color: #fb7185;">
						<span>Credit Note Total:</span> <span class="font-mono">{formatCurrency(selectedCN.grand_total)}</span>
					</div>
				</div>

				<div style="display: flex; justify-content: flex-end; margin-top: 1.25rem;">
					<button class="btn btn-secondary" onclick={() => (selectedCN = null)}>Close</button>
				</div>
			</div>
		</div>
	{/if}

	<!-- View Debit Note Details Modal -->
	{#if selectedDN}
		<div style="position: fixed; inset: 0; background: rgba(0,0,0,0.75); backdrop-filter: blur(4px); display: flex; align-items: center; justify-content: center; z-index: 50; padding: 1.5rem;">
			<div class="card" style="width: 100%; max-width: 650px; padding: 1.75rem; border-color: var(--border-accent);">
				<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1.25rem;">
					<h3 style="font-size: 1.25rem; font-weight: 700;">Debit Note #{selectedDN.debit_note_number}</h3>
					<button class="btn btn-ghost" onclick={() => (selectedDN = null)}>✕</button>
				</div>

				<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem; font-size: 0.875rem; margin-bottom: 1.25rem;">
					<div><span style="color: var(--text-muted);">Supplier:</span> <strong>{selectedDN.supplier_name}</strong></div>
					<div><span style="color: var(--text-muted);">GSTIN:</span> <strong>{selectedDN.supplier_gstin || 'Unregistered'}</strong></div>
					<div><span style="color: var(--text-muted);">Date:</span> <strong>{formatDate(selectedDN.debit_note_date)}</strong></div>
					<div><span style="color: var(--text-muted);">Reason:</span> <strong>{selectedDN.reason_code}</strong></div>
					<div style="grid-column: span 2;"><span style="color: var(--text-muted);">Remarks:</span> {selectedDN.reason_description || 'None'}</div>
				</div>

				<div style="background: rgba(255,255,255,0.04); border-radius: var(--radius-md); padding: 1rem; display: flex; flex-direction: column; gap: 0.5rem; font-size: 0.875rem;">
					<div style="display: flex; justify-content: space-between;"><span>Taxable Amount:</span> <span class="font-mono">{formatCurrency(selectedDN.taxable_amount)}</span></div>
					<div style="display: flex; justify-content: space-between;"><span>CGST:</span> <span class="font-mono">{formatCurrency(selectedDN.cgst_total)}</span></div>
					<div style="display: flex; justify-content: space-between;"><span>SGST:</span> <span class="font-mono">{formatCurrency(selectedDN.sgst_total)}</span></div>
					<div style="display: flex; justify-content: space-between;"><span>IGST:</span> <span class="font-mono">{formatCurrency(selectedDN.igst_total)}</span></div>
					<div style="display: flex; justify-content: space-between; font-size: 1.125rem; font-weight: 700; border-top: 1px solid var(--border-subtle); padding-top: 0.5rem; color: #34d399;">
						<span>Debit Note Total:</span> <span class="font-mono">{formatCurrency(selectedDN.grand_total)}</span>
					</div>
				</div>

				<div style="display: flex; justify-content: flex-end; margin-top: 1.25rem;">
					<button class="btn btn-secondary" onclick={() => (selectedDN = null)}>Close</button>
				</div>
			</div>
		</div>
	{/if}
</div>
