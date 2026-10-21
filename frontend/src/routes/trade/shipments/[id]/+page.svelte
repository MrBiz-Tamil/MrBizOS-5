<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { api } from '$lib/api';

	const id = $derived($page.params.id!);
	let data = $state<any>(null);
	let form = $state<any>({});
	let packages = $state<any[]>([]);
	let landed = $state<any>(null);
	let invoiceTrade = $state<any>(null);
	let error = $state('');
	let notice = $state('');

	// cost form
	let costType = $state('FREIGHT');
	let costAmount = $state('');
	let costDesc = $state('');

	// receipt form
	let rcvFc = $state('');
	let rcvInr = $state('');
	let rcvRef = $state('');

	const statuses = ['PLANNED', 'BOOKED', 'IN_TRANSIT', 'ARRIVED', 'CLEARED', 'COMPLETED', 'CANCELLED'];
	const docStatuses = ['PENDING', 'READY', 'SUBMITTED', 'RECEIVED', 'NA'];

	async function load() {
		try {
			data = await api.getShipment(id);
			const s = data.shipment;
			form = { ...s };
			packages = data.packages.map((p: any) => ({ ...p }));
			if (s.direction === 'IMPORT') landed = await api.shipmentLandedCost(id).catch(() => null);
			if (s.invoice_id) invoiceTrade = await api.getExportInvoice(s.invoice_id).catch(() => null);
		} catch (e: any) {
			error = e.message;
		}
	}
	onMount(load);

	function flash(msg: string) {
		notice = msg;
		error = '';
	}

	async function saveShipment(e: Event) {
		e.preventDefault();
		try {
			await api.updateShipment(id, {
				status: form.status,
				incoterm: form.incoterm,
				mode: form.mode,
				port_loading: form.port_loading,
				port_discharge: form.port_discharge,
				vessel_flight: form.vessel_flight,
				container_no: form.container_no,
				bl_awb_no: form.bl_awb_no,
				bl_awb_date: form.bl_awb_date || '',
				etd: form.etd || '',
				eta: form.eta || '',
				customs_ref_no: form.customs_ref_no,
				customs_ref_date: form.customs_ref_date || '',
				port_code: form.port_code,
				notes: form.notes
			});
			await load();
			flash('Shipment saved.');
		} catch (err: any) {
			error = err.message;
		}
	}

	async function saveDoc(d: any) {
		try {
			await api.updateShipmentDoc(id, d.id, { status: d.status, reference_no: d.reference_no, doc_date: d.doc_date || '', notes: d.notes });
			await load();
		} catch (err: any) {
			error = err.message;
		}
	}

	function addPackage() {
		const last = packages[packages.length - 1];
		const from = last ? last.package_to + 1 : 1;
		packages.push({ description: '', package_from: from, package_to: from, package_type: 'CARTON', quantity: '0', unit: '', net_weight_kg: '0', gross_weight_kg: '0', dimensions: '', marks: '' });
	}

	async function savePackages() {
		try {
			const r = await api.setShipmentPackages(id, packages);
			packages = r.packages.map((p: any) => ({ ...p }));
			data.packing_totals = r.packing_totals;
			flash('Packing list saved.');
		} catch (err: any) {
			error = err.message;
		}
	}

	async function addCost(e: Event) {
		e.preventDefault();
		try {
			await api.addShipmentCost(id, { cost_type: costType, amount_inr: costAmount, description: costDesc });
			costAmount = costDesc = '';
			await load();
		} catch (err: any) {
			error = err.message;
		}
	}

	async function removeCost(cid: string) {
		await api.deleteShipmentCost(id, cid);
		await load();
	}

	async function receive(e: Event) {
		e.preventDefault();
		try {
			const p = await api.tradePayment({ invoice_id: data.shipment.invoice_id, payment_method: 'BANK_TRANSFER', reference_number: rcvRef, fc_amount: rcvFc, inr_received: rcvInr });
			rcvFc = rcvInr = rcvRef = '';
			await load();
			flash(`Receipt recorded. Settled Rs. ${p.amount}; forex ${Number(p.forex_gain_loss) >= 0 ? 'gain' : 'loss'} Rs. ${Math.abs(Number(p.forex_gain_loss)).toFixed(2)}.`);
		} catch (err: any) {
			error = err.message;
		}
	}
</script>

<svelte:head><title>{data ? data.shipment.shipment_number : 'Shipment'}</title></svelte:head>

<div style="max-width: 1000px;">
	<a href="/trade/shipments" style="font-size: 0.8125rem;">← All shipments</a>
	{#if notice}<div class="card" style="padding: 0.6rem 1rem; margin: 0.75rem 0; color: #34d399;">{notice}</div>{/if}
	{#if error}<div class="card" style="padding: 0.6rem 1rem; margin: 0.75rem 0; color: #fb7185;">{error}</div>{/if}

	{#if data}
		{@const s = data.shipment}
		<h1 style="margin: 0.5rem 0 0.25rem;">{s.shipment_number} <span style="font-size: 1rem; color: var(--text-secondary);">{s.direction} · {s.counterparty_name}</span></h1>

		<form class="card" style="padding: 1.25rem; margin: 1rem 0;" onsubmit={saveShipment}>
			<h3 style="margin-bottom: 0.75rem;">Shipment details</h3>
			<div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(190px, 1fr)); gap: 0.75rem;">
				<div class="form-group"><label class="form-label" for="st">Status</label><select id="st" class="form-input" bind:value={form.status}>{#each statuses as x}<option value={x}>{x.replace('_', ' ')}</option>{/each}</select></div>
				<div class="form-group"><label class="form-label" for="inc">Incoterm</label><input id="inc" class="form-input" bind:value={form.incoterm} maxlength="5" /></div>
				<div class="form-group"><label class="form-label" for="mo">Mode</label><select id="mo" class="form-input" bind:value={form.mode}>{#each ['SEA', 'AIR', 'ROAD', 'RAIL', 'COURIER'] as x}<option>{x}</option>{/each}</select></div>
				<div class="form-group"><label class="form-label" for="pl">Port of loading</label><input id="pl" class="form-input" bind:value={form.port_loading} /></div>
				<div class="form-group"><label class="form-label" for="pd">Port of discharge</label><input id="pd" class="form-input" bind:value={form.port_discharge} /></div>
				<div class="form-group"><label class="form-label" for="vf">Vessel / flight</label><input id="vf" class="form-input" bind:value={form.vessel_flight} /></div>
				<div class="form-group"><label class="form-label" for="cn">Container no.</label><input id="cn" class="form-input" bind:value={form.container_no} /></div>
				<div class="form-group"><label class="form-label" for="bl">BL / AWB no.</label><input id="bl" class="form-input" bind:value={form.bl_awb_no} /></div>
				<div class="form-group"><label class="form-label" for="bld">BL / AWB date</label><input id="bld" type="date" class="form-input" bind:value={form.bl_awb_date} /></div>
				<div class="form-group"><label class="form-label" for="etd">ETD</label><input id="etd" type="date" class="form-input" bind:value={form.etd} /></div>
				<div class="form-group"><label class="form-label" for="eta">ETA</label><input id="eta" type="date" class="form-input" bind:value={form.eta} /></div>
				<div class="form-group"><label class="form-label" for="cr">{s.direction === 'EXPORT' ? 'Shipping bill no.' : 'Bill of entry no.'}</label><input id="cr" class="form-input" bind:value={form.customs_ref_no} /></div>
				<div class="form-group"><label class="form-label" for="crd">Date</label><input id="crd" type="date" class="form-input" bind:value={form.customs_ref_date} /></div>
				<div class="form-group"><label class="form-label" for="pc">Port code</label><input id="pc" class="form-input" bind:value={form.port_code} /></div>
			</div>
			<div class="form-group"><label class="form-label" for="nt">Notes</label><textarea id="nt" class="form-input" rows="2" bind:value={form.notes}></textarea></div>
			<button class="btn btn-primary">Save</button>
		</form>

		{#if s.direction === 'EXPORT'}
			<div class="card" style="padding: 1.25rem; margin-bottom: 1rem;">
				<h3 style="margin-bottom: 0.5rem;">Export invoice and receipts</h3>
				{#if !s.invoice_id}
					<p style="color: var(--text-secondary);">No invoice yet.</p>
					<a class="btn btn-primary" href="/trade/export-invoice/new?shipment={s.id}{s.customer_id ? '&customer=' + s.customer_id : ''}">Create export invoice for this shipment</a>
				{:else if invoiceTrade}
					<p>Invoice <a href="/invoices/{s.invoice_id}">open</a> · {invoiceTrade.currency} {invoiceTrade.fc_total} at {invoiceTrade.exchange_rate} · received {invoiceTrade.fc_received} · <strong>to receive {invoiceTrade.fc_balance}</strong></p>
					{#if Number(invoiceTrade.fc_balance) > 0}
						<form onsubmit={receive} style="display: flex; gap: 0.75rem; align-items: end; flex-wrap: wrap; margin-top: 0.5rem;">
							<div class="form-group" style="margin: 0;"><label class="form-label" for="rf">Received ({invoiceTrade.currency})</label><input id="rf" class="form-input" bind:value={rcvFc} required /></div>
							<div class="form-group" style="margin: 0;"><label class="form-label" for="ri">Credited by bank (INR)</label><input id="ri" class="form-input" bind:value={rcvInr} required /></div>
							<div class="form-group" style="margin: 0;"><label class="form-label" for="rr">Bank reference</label><input id="rr" class="form-input" bind:value={rcvRef} /></div>
							<button class="btn btn-primary">Record receipt</button>
						</form>
					{/if}
				{/if}
			</div>
		{/if}

		<div class="card" style="padding: 1.25rem; margin-bottom: 1rem;">
			<h3 style="margin-bottom: 0.25rem;">Print documents</h3>
			<p style="color: var(--text-secondary); font-size: 0.8125rem; margin-bottom: 0.75rem;">Built from this shipment, its export invoice, packing list and your trade profile. Open one to fill in the extra details, then print or download the PDF.</p>
			<div style="display: flex; flex-wrap: wrap; gap: 0.5rem;">
				{#each (s.direction === 'EXPORT'
					? [['commercial-invoice', 'Commercial invoice'], ['proforma-invoice', 'Proforma invoice'], ['packing-list', 'Packing list'], ['invoice-cum-packing-list', 'Invoice cum packing list'], ['bill-of-exchange', 'Bill of exchange'], ['shipping-instructions', "Shipper's letter of instruction"], ['bank-cover-letter', 'Bank cover letter'], ['coo-application', 'Certificate of origin application']]
					: [['purchase-order', 'Purchase order'], ['remittance-request', 'Remittance request'], ['grn', 'Goods received note'], ['landed-cost-sheet', 'Landed cost sheet']]) as [t, label]}
					<a class="btn btn-secondary" href="/trade/docs/{t}/{s.id}">{label}</a>
				{/each}
			</div>
		</div>

		<div class="card" style="padding: 1.25rem; margin-bottom: 1rem; overflow-x: auto;">
			<h3 style="margin-bottom: 0.25rem;">Document checklist <span style="font-size: 0.875rem; color: var(--text-secondary);">{s.docs_done} of {s.docs_total} done</span></h3>
			<p style="color: var(--text-secondary); font-size: 0.8125rem; margin-bottom: 0.75rem;">"We prepare" documents are ours; the others are issued by customs, the carrier, a chamber or your bank. Track their reference numbers here.</p>
			<table style="width: 100%; font-size: 0.8125rem;">
				<thead><tr style="text-align: left; color: var(--text-secondary);"><th>Document</th><th>Source</th><th>Status</th><th>Reference</th><th>Date</th><th></th></tr></thead>
				<tbody>
					{#each data.documents as d}
						<tr>
							<td>{d.label}</td>
							<td>{d.issued_by === 'US' ? 'We prepare' : 'Issued by others'}</td>
							<td><select class="form-input" bind:value={d.status} onchange={() => saveDoc(d)}>{#each docStatuses as x}<option value={x}>{x}</option>{/each}</select></td>
							<td><input class="form-input" bind:value={d.reference_no} onchange={() => saveDoc(d)} /></td>
							<td><input type="date" class="form-input" bind:value={d.doc_date} onchange={() => saveDoc(d)} /></td>
							<td></td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>

		<div class="card" style="padding: 1.25rem; margin-bottom: 1rem; overflow-x: auto;">
			<h3 style="margin-bottom: 0.75rem;">Packing list</h3>
			<table style="width: 100%; font-size: 0.8125rem;">
				<thead><tr style="text-align: left; color: var(--text-secondary);"><th>From</th><th>To</th><th>Type</th><th>Description</th><th>Qty</th><th>Unit</th><th>Net kg</th><th>Gross kg</th><th>Size</th><th></th></tr></thead>
				<tbody>
					{#each packages as p, i}
						<tr>
							<td><input class="form-input" style="width: 60px;" type="number" bind:value={p.package_from} /></td>
							<td><input class="form-input" style="width: 60px;" type="number" bind:value={p.package_to} /></td>
							<td><input class="form-input" style="width: 90px;" bind:value={p.package_type} /></td>
							<td><input class="form-input" bind:value={p.description} /></td>
							<td><input class="form-input" style="width: 70px;" bind:value={p.quantity} /></td>
							<td><input class="form-input" style="width: 60px;" bind:value={p.unit} /></td>
							<td><input class="form-input" style="width: 80px;" bind:value={p.net_weight_kg} /></td>
							<td><input class="form-input" style="width: 80px;" bind:value={p.gross_weight_kg} /></td>
							<td><input class="form-input" style="width: 90px;" bind:value={p.dimensions} /></td>
							<td><button type="button" class="btn btn-secondary" onclick={() => packages.splice(i, 1)}>✕</button></td>
						</tr>
					{/each}
				</tbody>
			</table>
			<div style="display: flex; gap: 0.5rem; margin-top: 0.75rem; align-items: center; flex-wrap: wrap;">
				<button type="button" class="btn btn-secondary" onclick={addPackage}>+ Add line</button>
				<button type="button" class="btn btn-primary" onclick={savePackages}>Save packing list</button>
				{#if data.packing_totals}<span style="font-size: 0.8125rem; color: var(--text-secondary);">{data.packing_totals.packages} packages · net {data.packing_totals.net_weight_kg} kg · gross {data.packing_totals.gross_weight_kg} kg</span>{/if}
			</div>
		</div>

		<div class="card" style="padding: 1.25rem; margin-bottom: 1rem; overflow-x: auto;">
			<h3 style="margin-bottom: 0.75rem;">Costs</h3>
			<table style="width: 100%; font-size: 0.8125rem; margin-bottom: 0.75rem;">
				<thead><tr style="text-align: left; color: var(--text-secondary);"><th>Type</th><th>Description</th><th>Amount (Rs.)</th><th>Claimable</th><th></th></tr></thead>
				<tbody>
					{#each data.costs as c}
						<tr><td>{c.cost_type}</td><td>{c.description}</td><td>{c.amount_inr}</td><td>{c.recoverable ? 'Yes (credit)' : 'No'}</td><td><button class="btn btn-secondary" onclick={() => removeCost(c.id)}>✕</button></td></tr>
					{:else}
						<tr><td colspan="5" style="color: var(--text-secondary);">No costs yet.</td></tr>
					{/each}
				</tbody>
			</table>
			<form onsubmit={addCost} style="display: flex; gap: 0.75rem; align-items: end; flex-wrap: wrap;">
				<div class="form-group" style="margin: 0;"><label class="form-label" for="ct">Type</label>
					<select id="ct" class="form-input" bind:value={costType}>{#each ['FREIGHT', 'INSURANCE', 'BCD', 'SWS', 'IGST', 'CLEARING', 'PORT', 'DEMURRAGE', 'BANK', 'OTHER'] as x}<option>{x}</option>{/each}</select></div>
				<div class="form-group" style="margin: 0;"><label class="form-label" for="cd">Description</label><input id="cd" class="form-input" bind:value={costDesc} /></div>
				<div class="form-group" style="margin: 0;"><label class="form-label" for="ca">Amount (Rs.)</label><input id="ca" class="form-input" bind:value={costAmount} required /></div>
				<button class="btn btn-primary">Add cost</button>
			</form>
		</div>

		{#if s.direction === 'IMPORT' && landed}
			<div class="card" style="padding: 1.25rem; margin-bottom: 1rem; overflow-x: auto;">
				<h3 style="margin-bottom: 0.5rem;">Landed cost</h3>
				{#if landed.note}<p style="color: var(--text-secondary); font-size: 0.8125rem;">{landed.note}</p>{/if}
				<p style="font-size: 0.875rem;">Goods Rs. {landed.goods_value_inr} + added costs Rs. {landed.added_costs_inr} = <strong>Rs. {landed.total_landed_inr}</strong>. Claimable duty (IGST) Rs. {landed.recoverable_inr} is not part of the cost.</p>
				{#if landed.items.length}
					<table style="width: 100%; font-size: 0.8125rem; margin-top: 0.5rem;">
						<thead><tr style="text-align: left; color: var(--text-secondary);"><th>Item</th><th>Qty</th><th>Value</th><th>Added cost</th><th>Landed</th><th>Per unit</th></tr></thead>
						<tbody>{#each landed.items as it}<tr><td>{it.description}</td><td>{it.quantity} {it.unit_code}</td><td>{it.value_inr}</td><td>{it.allocated_cost_inr}</td><td>{it.landed_cost_inr}</td><td>{it.landed_unit_cost_inr}</td></tr>{/each}</tbody>
					</table>
				{/if}
			</div>
		{/if}
	{/if}
</div>
