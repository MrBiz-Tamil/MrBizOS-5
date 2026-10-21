<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { api, type Customer, type Product, type Tenant, type Unit } from '$lib/api';

	let company = $state<Tenant | null>(null);
	let customers = $state<Customer[]>([]);
	let products = $state<Product[]>([]);
	let units = $state<Unit[]>([]);

	let selectedCustomerId = $state('');
	let invoiceDate = $state(new Date().toISOString().split('T')[0]);
	let dueDate = $state(new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toISOString().split('T')[0]);
	let supplyState = $state('');
	let billingAddress = $state('');
	let notes = $state('Thank you for your business. Please remit payment by the due date.');
	let loading = $state(true);
	let submitting = $state(false);
	let error = $state('');

	// Optional Logistics & Template Details
	let enableLogistics = $state(false);
	let vehicleNumber = $state('');
	let paymentMode = $state('Cash');
	let deliveryPlace = $state('');
	let poNumber = $state('');
	let jurisdictionCity = $state('');
	let interestRate = $state('24%');
	let returnPolicy = $state('Goods once sold will not be taken back.');
	let ewayBill = $state('');
	let transporterName = $state('');
	let reverseCharge = $state(false);
	let subscriptionPeriodStart = $state('');
	let subscriptionPeriodEnd = $state('');
	let grossWeight = $state('');
	let netWeight = $state('');
	let cashierName = $state('');
	let counterNumber = $state('');
	let projectName = $state('');
	let doctorName = $state('');
	let doctorRegNo = $state('');
	let patientName = $state('');
	let patientAge = $state<number | undefined>(undefined);

	interface FormItem {
		product_id: string;
		item_description: string;
		hsn_sac: string;
		quantity: number;
		unit_code: string;
		unit_price: number;
		discount_rate: number;
		tax_rate: number;
	}

	let items = $state<FormItem[]>([
		{
			product_id: '',
			item_description: '',
			hsn_sac: '',
			quantity: 1,
			unit_code: 'PCS',
			unit_price: 0,
			discount_rate: 0,
			tax_rate: 18
		}
	]);

	onMount(async () => {
		try {
			const [comp, custs, prods, uList] = await Promise.all([
				api.getCompany(),
				api.listCustomers(),
				api.listProducts(),
				api.listUnits()
			]);
			company = comp;
			customers = custs;
			products = prods;
			units = uList;
			supplyState = comp.state;
			if (comp.default_terms) {
				paymentMode = comp.default_terms;
			}
			if (comp.default_jurisdiction) {
				jurisdictionCity = comp.default_jurisdiction;
			} else if (comp.city) {
				jurisdictionCity = comp.city;
			}
			if (comp.default_interest_rate) {
				interestRate = comp.default_interest_rate;
			}
			if (comp.default_return_policy) {
				returnPolicy = comp.default_return_policy;
			}
			if (units.length > 0 && items.length > 0) {
				const hasPCS = units.some((u) => u.code === 'PCS');
				if (!hasPCS) {
					items[0].unit_code = units[0].code;
				}
			}
		} catch (err: any) {
			error = err.message || 'Failed to initialize invoice form';
		} finally {
			loading = false;
		}
	});

	function handleCustomerChange() {
		const cust = customers.find((c) => c.id === selectedCustomerId);
		if (cust) {
			billingAddress = cust.billing_address;
			if (cust.billing_state) {
				supplyState = cust.billing_state;
			}
			if (cust.billing_city) {
				deliveryPlace = cust.billing_city;
			}
		}
	}

	function handleProductChange(index: number) {
		const item = items[index];
		const prod = products.find((p) => p.id === item.product_id);
		if (prod) {
			item.item_description = prod.name;
			item.hsn_sac = prod.hsn_sac;
			item.unit_price = parseFloat(prod.selling_price) || 0;
			item.tax_rate = parseFloat(prod.tax_rate) || 18;
			if (prod.discount_rate) {
				item.discount_rate = parseFloat(prod.discount_rate) || 0;
			}
			const u = units.find((un) => un.id === prod.sales_unit_id);
			if (u) {
				item.unit_code = u.code;
			}
		}
	}


	function addItem() {
		const defaultUnit = units.find((u) => u.code === 'PCS')?.code || (units.length > 0 ? units[0].code : 'PCS');
		items.push({
			product_id: '',
			item_description: '',
			hsn_sac: '',
			quantity: 1,
			unit_code: defaultUnit,
			unit_price: 0,
			discount_rate: 0,
			tax_rate: 18
		});
	}

	function removeItem(index: number) {
		if (items.length > 1) {
			items.splice(index, 1);
		}
	}

	let isInterState = $derived(
		Boolean(company?.state && supplyState && company.state.trim().toLowerCase() !== supplyState.trim().toLowerCase())
	);

	let totals = $derived.by(() => {
		let subtotal = 0;
		let discountTotal = 0;
		let taxableAmount = 0;
		let totalTax = 0;

		for (const it of items) {
			const gross = (it.quantity || 0) * (it.unit_price || 0);
			const disc = gross * ((it.discount_rate || 0) / 100);
			const taxable = Math.max(0, gross - disc);
			const tax = taxable * ((it.tax_rate || 0) / 100);

			subtotal += gross;
			discountTotal += disc;
			taxableAmount += taxable;
			totalTax += tax;
		}

		const exactGrand = taxableAmount + totalTax;
		const roundedGrand = Math.round(exactGrand);
		const roundOff = roundedGrand - exactGrand;

		return {
			subtotal,
			discountTotal,
			taxableAmount,
			totalTax,
			cgst: isInterState ? 0 : totalTax / 2,
			sgst: isInterState ? 0 : totalTax / 2,
			igst: isInterState ? totalTax : 0,
			roundOff,
			grandTotal: roundedGrand
		};
	});

	async function handleSubmit(e: Event) {
		e.preventDefault();
		if (!selectedCustomerId) {
			error = 'Please select a customer';
			return;
		}
		if (items.some((it) => !it.item_description || it.quantity <= 0)) {
			error = 'All line items must have a valid description and quantity > 0';
			return;
		}

		error = '';
		submitting = true;
		try {
			let finalNotes = notes;
			if (enableLogistics) {
				const transportLines = [
					vehicleNumber.trim() ? `Vehicle: ${vehicleNumber.trim()}` : '',
					(deliveryPlace.trim() || '') ? `Delivery Place: ${deliveryPlace.trim()}` : '',
					poNumber.trim() ? `PO: ${poNumber.trim()}` : '',
					paymentMode.trim() ? `Terms: ${paymentMode.trim()}` : '',
					(jurisdictionCity.trim() || company?.default_jurisdiction || company?.city || '') ? `Jurisdiction: ${jurisdictionCity.trim() || company?.default_jurisdiction || company?.city}` : '',
					interestRate.trim() ? `Interest: ${interestRate.trim()}` : '',
					returnPolicy.trim() ? `Policy: ${returnPolicy.trim()}` : '',
					ewayBill.trim() ? `EWay: ${ewayBill.trim()}` : '',
					transporterName.trim() ? `Transporter: ${transporterName.trim()}` : ''
				].filter(Boolean);
				if (transportLines.length > 0) {
					finalNotes = (finalNotes ? finalNotes + '\n\n' : '') + ['[Logistics & Terms]', ...transportLines].join('\n');
				}
			}

			const payload = {
				customer_id: selectedCustomerId,
				invoice_date: new Date(invoiceDate).toISOString(),
				due_date: new Date(dueDate).toISOString(),
				billing_address: billingAddress,
				supply_state: supplyState,
				notes: finalNotes,
				vehicle_number: vehicleNumber.trim(),
				delivery_place: deliveryPlace.trim() || supplyState,
				payment_mode: paymentMode.trim(),
				po_number: poNumber.trim(),
				jurisdiction_city: jurisdictionCity.trim() || company?.default_jurisdiction || company?.city || '',
				interest_rate: interestRate.trim() || '24%',
				return_policy: returnPolicy.trim() || 'Goods once sold will not be taken back.',
				eway_bill: ewayBill.trim(),
				transporter_name: transporterName.trim(),
				reverse_charge: reverseCharge,
				subscription_period_start: subscriptionPeriodStart || undefined,
				subscription_period_end: subscriptionPeriodEnd || undefined,
				gross_weight: grossWeight.trim() || undefined,
				net_weight: netWeight.trim() || undefined,
				cashier_name: cashierName.trim() || undefined,
				counter_number: counterNumber.trim() || undefined,
				project_name: projectName.trim() || undefined,
				doctor_name: doctorName.trim() || undefined,
				doctor_reg_no: doctorRegNo.trim() || undefined,
				patient_name: patientName.trim() || undefined,
				patient_age: patientAge ? Number(patientAge) : undefined,
				items: items.map((it) => ({
					product_id: it.product_id || undefined,
					item_description: it.item_description,
					hsn_sac: it.hsn_sac,
					quantity: it.quantity.toString(),
					unit_code: it.unit_code,
					unit_price: it.unit_price.toString(),
					discount_rate: it.discount_rate.toString(),
					tax_rate: it.tax_rate.toString()
				}))
			};

			const created = await api.createInvoice(payload);
			if (typeof window !== 'undefined') {
				localStorage.setItem(`saas_invoice_transport_${created.id}`, JSON.stringify({
					vehicleNumber,
					paymentMode,
					terms: paymentMode,
					deliveryPlace: deliveryPlace || supplyState,
					poNumber,
					jurisdictionCity: jurisdictionCity || company?.default_jurisdiction || company?.city || '',
					interestRate,
					returnPolicy,
					ewayBill,
					transporterName,
					reverseCharge,
					subscriptionPeriodStart,
					subscriptionPeriodEnd,
					grossWeight,
					netWeight,
					cashierName,
					counterNumber,
					projectName
				}));
			}
			goto(`/invoices/${created.id}`);
		} catch (err: any) {
			error = err.message || 'Failed to create invoice';
		} finally {
			submitting = false;
		}
	}

	function formatCurrency(val: number): string {
		return new Intl.NumberFormat('en-IN', { style: 'currency', currency: 'INR' }).format(val);
	}
</script>

<svelte:head>
	<title>New Sales Invoice - SaaS Billing</title>
</svelte:head>

<div style="display: flex; flex-direction: column; gap: 1.5rem;">
	<div style="display: flex; justify-content: space-between; align-items: center;">
		<div>
			<a href="/invoices" style="font-size: 0.8125rem; color: #818cf8; text-decoration: none; margin-bottom: 0.25rem; display: inline-block;">
				← Back to Invoices
			</a>
			<h1 style="font-size: 1.75rem;">Create New Sales Invoice</h1>
		</div>
		<div>
			<span style="font-size: 0.8125rem; color: var(--text-secondary); margin-right: 0.5rem;">Next Number:</span>
			<span style="font-family: var(--font-mono); font-size: 0.875rem; background: rgba(99, 102, 241, 0.15); color: #a5b4fc; padding: 0.25rem 0.6rem; border-radius: var(--radius-sm); border: 1px solid rgba(99, 102, 241, 0.3);">
				{company?.invoice_prefix || 'INV'}-AUTO
			</span>
		</div>
	</div>

	{#if error}
		<div class="card" style="border-color: rgba(244, 63, 94, 0.4); background: rgba(244, 63, 94, 0.1); color: #fb7185; padding: 1rem;">
			⚠️ {error}
		</div>
	{/if}

	<form onsubmit={handleSubmit} style="display: flex; flex-direction: column; gap: 1.5rem;">
		<!-- Customer & State Section -->
		<div class="card" style="display: grid; grid-template-columns: repeat(auto-fit, minmax(280px, 1fr)); gap: 1.25rem;">
			<div class="form-group" style="margin-bottom: 0;">
				<label class="form-label" for="customer">Customer *</label>
				<select id="customer" class="form-select" bind:value={selectedCustomerId} onchange={handleCustomerChange} required>
					<option value="">Select Customer...</option>
					{#each customers as cust}
						<option value={cust.id}>{cust.name} {cust.company_name ? `(${cust.company_name})` : ''}</option>
					{/each}
				</select>
			</div>

			<div class="form-group" style="margin-bottom: 0;">
				<label class="form-label" for="invoice-date">Invoice Date</label>
				<input id="invoice-date" type="date" class="form-input" bind:value={invoiceDate} required />
			</div>

			<div class="form-group" style="margin-bottom: 0;">
				<label class="form-label" for="due-date">Due Date</label>
				<input id="due-date" type="date" class="form-input" bind:value={dueDate} required />
			</div>

			<div class="form-group" style="margin-bottom: 0;">
				<label class="form-label" for="supply-state">
					Place of Supply (State)
					{#if isInterState}
						<span class="badge" style="background: rgba(14, 165, 233, 0.2); color: #38bdf8; font-size: 0.65rem;">Inter-State (IGST)</span>
					{:else}
						<span class="badge" style="background: rgba(16, 185, 129, 0.2); color: #34d399; font-size: 0.65rem;">Intra-State (CGST+SGST)</span>
					{/if}
				</label>
				<input id="supply-state" type="text" class="form-input" bind:value={supplyState} placeholder="e.g. Maharashtra" required />
			</div>

			<div class="form-group" style="grid-column: 1 / -1; margin-bottom: 0;">
				<label class="form-label" for="billing-addr">Billing Address</label>
				<textarea id="billing-addr" class="form-textarea" rows="2" bind:value={billingAddress} placeholder="Customer billing address..."></textarea>
			</div>
		</div>

		<!-- Optional Logistics, Transport & Terms Section (For BlueMetals / Industrial Templates) -->
		<div class="card" style="background: rgba(15, 23, 42, 0.6); border: 1px solid rgba(99, 102, 241, 0.3); padding: 1.25rem;">
			<div
				style="display: flex; justify-content: space-between; align-items: center; cursor: pointer;"
				role="button"
				tabindex="0"
				onclick={() => (enableLogistics = !enableLogistics)}
				onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); enableLogistics = !enableLogistics; } }}
			>
				<div style="display: flex; align-items: center; gap: 0.6rem;">
					<span style="font-size: 1.25rem;">🚚</span>
					<div>
						<div style="font-size: 0.95rem; font-weight: 700; color: #a5b4fc;">
							Transport, Logistics & Statutory Terms (Optional)
						</div>
						<div style="font-size: 0.75rem; color: var(--text-secondary);">
							Configure Vehicle Number, Delivery Place, Payment Terms, PO #, and Jurisdiction for BlueMetals (Template #11) & Industrial bills
						</div>
					</div>
				</div>
				<button type="button" class="btn btn-secondary btn-sm" style="font-size: 0.75rem; padding: 0.25rem 0.6rem;">
					{enableLogistics ? '▲ Collapse' : '▼ Expand Fields'}
				</button>
			</div>

			{#if enableLogistics}
				<div class="animate-fade-in" style="display: flex; flex-direction: column; gap: 1rem; margin-top: 1.25rem; border-top: 1px solid rgba(255,255,255,0.08); padding-top: 1rem;">
					<div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 1rem;">
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="new-veh">Vehicle / Truck Number</label>
							<input id="new-veh" type="text" class="form-input font-mono" bind:value={vehicleNumber} placeholder="e.g. MH-12-AB-9801" />
							<datalist id="vehicle-suggestions">
								<option value="KA-01-XY-5544"></option>
							</datalist>
						</div>

						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="new-terms">Payment Mode / Terms</label>
							<select id="new-terms" class="form-select" bind:value={paymentMode}>
								<option value="Cash">Cash</option>
								<option value="Credit / Due">Credit / Due</option>
								<option value="Bank Transfer">Bank Transfer (NEFT/RTGS)</option>
								<option value="Cheque">Cheque</option>
								<option value="UPI / QR">UPI / QR</option>
								<option value="Net 15 Days">Net 15 Days</option>
								<option value="Net 30 Days">Net 30 Days</option>
							</select>
						</div>

						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="new-place">Delivery Place / Site</label>
							<input id="new-place" type="text" class="form-input" bind:value={deliveryPlace} placeholder="e.g. POTHAYADI or Site Location" />
						</div>

						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="new-po">Customer P.O. / Work Order #</label>
							<input id="new-po" type="text" class="form-input" bind:value={poNumber} placeholder="e.g. 55 or PO-98421" />
						</div>
					</div>

					<div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 1rem;">
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="new-juris">Legal Jurisdiction City</label>
							<input id="new-juris" type="text" class="form-input" bind:value={jurisdictionCity} placeholder="e.g. Pune" />
						</div>

						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="new-interest">Interest Rate on Due Date</label>
							<input id="new-interest" type="text" class="form-input" bind:value={interestRate} placeholder="e.g. 24%" />
						</div>

						<div class="form-group" style="margin-bottom: 0; grid-column: span 2;">
							<label class="form-label" for="new-policy">Goods Return Policy</label>
							<input id="new-policy" type="text" class="form-input" bind:value={returnPolicy} placeholder="Goods once sold will not be taken back." />
						</div>
					</div>

					<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="new-eway">E-Way Bill Number (Optional)</label>
							<input id="new-eway" type="text" class="form-input font-mono" bind:value={ewayBill} placeholder="12-digit E-Way Bill..." />
						</div>
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="new-trans">Transporter Name (Optional)</label>
							<input id="new-trans" type="text" class="form-input" bind:value={transporterName} placeholder="Logistics carrier..." />
						</div>
					</div>

					<div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 1rem;">
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="new-gross-wt">Gross Weight (Dispatch)</label>
							<input id="new-gross-wt" type="text" class="form-input" bind:value={grossWeight} placeholder="e.g. 24.50 MT" />
						</div>
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="new-net-wt">Net Weight</label>
							<input id="new-net-wt" type="text" class="form-input" bind:value={netWeight} placeholder="e.g. 23.80 MT" />
						</div>
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="new-sub-start">Subscription Start (SaaS) / Check-in (Hostel)</label>
							<input id="new-sub-start" type="date" class="form-input" bind:value={subscriptionPeriodStart} />
						</div>
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="new-sub-end">Subscription End (SaaS) / Check-out (Hostel)</label>
							<input id="new-sub-end" type="date" class="form-input" bind:value={subscriptionPeriodEnd} />
						</div>
					</div>

					<div style="display: grid; grid-template-columns: 1.5fr 1fr 1fr; gap: 1rem;">
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="new-project">Project / Scope (Agency) · Booking Ref (Hostel) · KOT / Order No (Restaurant) · Cake Message (Bakery) · Loyalty / Member Ref (Grocery) · Site / Project (Hardware)</label>
							<input id="new-project" type="text" class="form-input" bind:value={projectName} placeholder="e.g. Q3 Architecture Redesign" />
						</div>
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="new-cashier">Cashier (Retail) · Front Desk (Hostel) · Waiter (Restaurant) · Served By (Bakery) · Cashier (Grocery)</label>
							<input id="new-cashier" type="text" class="form-input" bind:value={cashierName} placeholder="Cashier 01" />
						</div>
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="new-counter">Counter (Retail) · Room / Bed (Hostel) · Table (Restaurant) · Order Token (Bakery) · Counter (Grocery)</label>
							<input id="new-counter" type="text" class="form-input" bind:value={counterNumber} placeholder="Counter A" />
						</div>
					</div>

					<div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 1rem;">
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="new-patient">Patient Name (Pharmacy)</label>
							<input id="new-patient" type="text" class="form-input" bind:value={patientName} placeholder="e.g. Sunita Rao" />
						</div>
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="new-patient-age">Patient Age</label>
							<input id="new-patient-age" type="number" min="0" max="130" class="form-input" bind:value={patientAge} placeholder="e.g. 54" />
						</div>
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="new-doctor">Prescribing Doctor (Pharmacy)</label>
							<input id="new-doctor" type="text" class="form-input" bind:value={doctorName} placeholder="e.g. Dr. Anil Kulkarni, MD" />
						</div>
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="new-doctor-reg">Doctor Reg. No</label>
							<input id="new-doctor-reg" type="text" class="form-input" bind:value={doctorRegNo} placeholder="e.g. MMC-2014/08817" />
						</div>
					</div>

					<div style="margin-top: 0.25rem;">
						<label style="display: flex; align-items: center; gap: 0.5rem; font-size: 0.8125rem; cursor: pointer; color: var(--text-secondary);">
							<input type="checkbox" bind:checked={reverseCharge} />
							<span style="font-weight: 600; color: #fbbf24;">Supply is Subject to Reverse Charge (Section 9(3) / 9(4) of CGST Act)</span>
						</label>
					</div>
				</div>
			{/if}
		</div>

		<!-- Line Items Section -->
		<div class="card" style="padding: 1.25rem;">
			<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem;">
				<h3 style="font-size: 1.125rem;">Invoice Line Items</h3>
				<button type="button" onclick={addItem} class="btn btn-secondary btn-sm">
					+ Add Row
				</button>
			</div>

			<div class="table-container" style="border: none;">
				<table class="data-table">
					<thead>
						<tr>
							<th style="width: 25%;">Product / Item Description</th>
							<th style="width: 12%;">HSN/SAC</th>
							<th style="width: 10%;">Qty</th>
							<th style="width: 10%;">Unit</th>
							<th style="width: 12%;">Rate (₹)</th>
							<th style="width: 8%;">Disc %</th>
							<th style="width: 10%;">GST %</th>
							<th class="text-right" style="width: 10%;">Total</th>
							<th style="width: 3%;"></th>
						</tr>
					</thead>
					<tbody>
						{#each items as item, idx}
							<tr>
								<td>
									<div style="display: flex; flex-direction: column; gap: 0.25rem;">
										<select
											class="form-select"
											style="padding: 0.35rem 0.5rem; font-size: 0.8125rem;"
											bind:value={item.product_id}
											onchange={() => handleProductChange(idx)}
										>
											<option value="">-- Choose Product --</option>
											{#each products as prod}
												<option value={prod.id}>
													{prod.product_type === 'DIGITAL' ? '💻 ' : prod.product_type === 'SERVICE' ? '🛠️ ' : '📦 '}
													{prod.name} (₹{prod.selling_price})
												</option>
											{/each}
										</select>
										<input
											type="text"
											class="form-input"
											style="padding: 0.35rem 0.5rem; font-size: 0.8125rem;"
											placeholder="Description..."
											bind:value={item.item_description}
											required
										/>
									</div>
								</td>
								<td>
									<input
										type="text"
										class="form-input"
										style="padding: 0.35rem 0.5rem; font-size: 0.8125rem; font-family: var(--font-mono);"
										placeholder="HSN"
										bind:value={item.hsn_sac}
									/>
								</td>
								<td>
									<input
										type="number"
										step="any"
										min="0.0001"
										class="form-input"
										style="padding: 0.35rem 0.5rem; font-size: 0.8125rem;"
										bind:value={item.quantity}
										required
									/>
								</td>
								<td>
									<select
										class="form-select"
										style="padding: 0.35rem 0.5rem; font-size: 0.8125rem;"
										bind:value={item.unit_code}
									>
										{#each units as u}
											<option value={u.code}>{u.code} - {u.name}</option>
										{/each}
									</select>
								</td>
								<td>
									<input
										type="number"
										step="any"
										min="0"
										class="form-input"
										style="padding: 0.35rem 0.5rem; font-size: 0.8125rem; font-family: var(--font-mono);"
										bind:value={item.unit_price}
										required
									/>
								</td>
								<td>
									<input
										type="number"
										step="any"
										min="0"
										max="100"
										class="form-input"
										style="padding: 0.35rem 0.5rem; font-size: 0.8125rem;"
										bind:value={item.discount_rate}
									/>
								</td>
								<td>
									<select
										class="form-select"
										style="padding: 0.35rem 0.5rem; font-size: 0.8125rem;"
										bind:value={item.tax_rate}
									>
										<option value={0}>0%</option>
										<option value={5}>5%</option>
										<option value={12}>12%</option>
										<option value={18}>18%</option>
										<option value={28}>28%</option>
									</select>
								</td>
								<td class="text-right font-mono" style="font-weight: 600; font-size: 0.875rem;">
									{formatCurrency(
										Math.max(0, ((Number(item.quantity) || 0) * (Number(item.unit_price) || 0)) * (1 - (Number(item.discount_rate) || 0) / 100)) * (1 + (Number(item.tax_rate) || 0) / 100)
									)}
								</td>
								<td>
									{#if items.length > 1}
										<button
											type="button"
											onclick={() => removeItem(idx)}
											class="btn btn-danger btn-sm"
											style="padding: 0.2rem 0.4rem; font-size: 0.75rem;"
											title="Delete row"
										>
											✕
										</button>
									{/if}
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</div>

		<!-- Summary Calculation Box & Actions -->
		<div style="display: grid; grid-template-columns: 1fr 380px; gap: 1.5rem; align-items: start;">
			<div class="card">
				<div class="form-group" style="margin-bottom: 0;">
					<label class="form-label" for="invoice-notes">Terms, Payment Details & Notes</label>
					<textarea
						id="invoice-notes"
						class="form-textarea"
						rows="4"
						bind:value={notes}
						placeholder="Bank details, UPI ID, payment terms..."
					></textarea>
				</div>
			</div>

			<div class="card" style="display: flex; flex-direction: column; gap: 0.75rem;">
				<div style="display: flex; justify-content: space-between; font-size: 0.875rem; color: var(--text-secondary);">
					<span>Subtotal:</span>
					<span class="font-mono">{formatCurrency(totals.subtotal)}</span>
				</div>

				{#if totals.discountTotal > 0}
					<div style="display: flex; justify-content: space-between; font-size: 0.875rem; color: #fb7185;">
						<span>Discount:</span>
						<span class="font-mono">- {formatCurrency(totals.discountTotal)}</span>
					</div>
				{/if}

				<div style="display: flex; justify-content: space-between; font-size: 0.875rem; color: var(--text-secondary);">
					<span>Taxable Amount:</span>
					<span class="font-mono">{formatCurrency(totals.taxableAmount)}</span>
				</div>

				{#if isInterState}
					<div style="display: flex; justify-content: space-between; font-size: 0.875rem; color: #38bdf8;">
						<span>Integrated GST (IGST):</span>
						<span class="font-mono">{formatCurrency(totals.igst)}</span>
					</div>
				{:else}
					<div style="display: flex; justify-content: space-between; font-size: 0.875rem; color: #34d399;">
						<span>Central GST (CGST):</span>
						<span class="font-mono">{formatCurrency(totals.cgst)}</span>
					</div>
					<div style="display: flex; justify-content: space-between; font-size: 0.875rem; color: #34d399;">
						<span>State GST (SGST):</span>
						<span class="font-mono">{formatCurrency(totals.sgst)}</span>
					</div>
				{/if}

				{#if totals.roundOff !== 0}
					<div style="display: flex; justify-content: space-between; font-size: 0.8125rem; color: var(--text-muted);">
						<span>Round Off:</span>
						<span class="font-mono">{totals.roundOff > 0 ? '+' : ''}{totals.roundOff.toFixed(2)}</span>
					</div>
				{/if}

				<div style="border-top: 1px solid var(--border-subtle); padding-top: 0.75rem; display: flex; justify-content: space-between; font-size: 1.25rem; font-weight: 700;">
					<span>Grand Total:</span>
					<span class="font-mono text-indigo">{formatCurrency(totals.grandTotal)}</span>
				</div>

				<button
					type="submit"
					class="btn btn-primary"
					style="margin-top: 1rem; width: 100%; padding: 0.75rem;"
					disabled={submitting}
				>
					{#if submitting}
						Generating Sequential Invoice...
					{:else}
						Generate & Issue Invoice →
					{/if}
				</button>
			</div>
		</div>
	</form>
</div>
