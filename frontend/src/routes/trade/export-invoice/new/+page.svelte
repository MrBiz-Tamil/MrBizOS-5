<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { api } from '$lib/api';

	let customers = $state<any[]>([]);
	let units = $state<any[]>([]);
	let currencies = $state<any[]>([]);
	let profile = $state<any>(null);
	let error = $state('');
	let busy = $state(false);
	let rateNote = $state('');

	let customerId = $state('');
	let exportType = $state('LUT');
	let currency = $state('USD');
	let exchangeRate = $state('');
	let incoterm = $state('FOB');
	let freight = $state('0');
	let insurance = $state('0');
	let paymentTerms = $state('');
	let buyerCountry = $state('');
	let buyerTaxId = $state('');
	let consigneeName = $state('');
	let consigneeAddress = $state('');
	let notifyParty = $state('');
	let countryDestination = $state('');
	let portLoading = $state('');
	let portDischarge = $state('');
	let finalDestination = $state('');
	let vesselFlight = $state('');
	let portCode = $state('');
	let shipmentId = $state('');
	let lines = $state([{ item_description: '', hsn_sac: '', quantity: '1', unit_code: 'PCS', unit_price: '', discount_rate: '0', tax_rate: '0' }]);

	onMount(async () => {
		shipmentId = $page.url.searchParams.get('shipment') || '';
		[customers, units, currencies, profile] = await Promise.all([api.listCustomers(), api.listUnits(), api.tradeCurrencies(), api.tradeProfile().catch(() => null)]);
		if (profile) {
			portLoading = profile.default_port_loading || '';
			portCode = profile.default_port_code || '';
		}
		const qc = $page.url.searchParams.get('customer');
		if (qc) customerId = qc;
		if (units.length && !units.find((u: any) => u.code === 'PCS')) lines[0].unit_code = units[0].code;
		await fetchRate();
	});

	async function fetchRate() {
		rateNote = '';
		try {
			const r = await api.tradeRateOn(currency);
			exchangeRate = String(r.rate);
			rateNote = `Rate from ${r.rate_date} (${r.source}). Change it if needed.`;
		} catch {
			exchangeRate = '';
			rateNote = `No ${currency} rate saved yet; type one or add it in Trade Settings.`;
		}
	}

	function addLine() {
		lines.push({ item_description: '', hsn_sac: '', quantity: '1', unit_code: lines[0].unit_code, unit_price: '', discount_rate: '0', tax_rate: exportType.includes('IGST') ? '18' : '0' });
	}

	const fcTotal = $derived(lines.reduce((s, l) => s + (parseFloat(l.quantity) || 0) * (parseFloat(l.unit_price) || 0) * (1 - (parseFloat(l.discount_rate) || 0) / 100), 0));
	const inrTotal = $derived(fcTotal * (parseFloat(exchangeRate) || 0));

	async function submit(e: Event) {
		e.preventDefault();
		error = '';
		busy = true;
		try {
			const today = new Date().toISOString();
			const res = await api.createExportInvoice({
				customer_id: customerId,
				invoice_date: today,
				due_date: today,
				items: lines.map((l) => ({ ...l, tax_rate: exportType.includes('IGST') ? l.tax_rate : '0' })),
				trade: {
					export_type: exportType,
					currency,
					exchange_rate: exchangeRate,
					incoterm,
					fc_freight: freight || '0',
					fc_insurance: insurance || '0',
					payment_terms: paymentTerms,
					buyer_country: buyerCountry,
					buyer_tax_id: buyerTaxId,
					consignee_name: consigneeName,
					consignee_address: consigneeAddress,
					notify_party: notifyParty,
					country_destination: countryDestination,
					port_loading: portLoading,
					port_discharge: portDischarge,
					final_destination: finalDestination,
					vessel_flight: vesselFlight,
					port_code: portCode,
					shipment_id: shipmentId || undefined
				}
			});
			window.location.href = shipmentId ? `/trade/shipments/${shipmentId}` : `/invoices/${res.invoice.id}`;
		} catch (err: any) {
			error = err.message || 'Could not create the export invoice';
			busy = false;
		}
	}
</script>

<svelte:head><title>New Export Invoice</title></svelte:head>

<div style="max-width: 1000px;">
	<h1 style="margin-bottom: 0.25rem;">New export invoice</h1>
	<p style="color: var(--text-secondary); margin-bottom: 1rem;">Enter prices in the foreign currency. The books are kept in INR at the exchange rate you set here.</p>
	{#if error}<div class="card" style="padding: 0.6rem 1rem; margin-bottom: 1rem; color: #fb7185;">{error}</div>{/if}
	<form onsubmit={submit}>
		<div class="card" style="padding: 1.25rem; margin-bottom: 1rem;">
			<h3 style="margin-bottom: 0.75rem;">Buyer and terms</h3>
			<div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(210px, 1fr)); gap: 0.75rem;">
				<div class="form-group"><label class="form-label" for="c">Buyer (customer)</label>
					<select id="c" class="form-input" bind:value={customerId} required><option value="" disabled>Select</option>{#each customers as c}<option value={c.id}>{c.name}</option>{/each}</select></div>
				<div class="form-group"><label class="form-label" for="et">GST treatment</label>
					<select id="et" class="form-input" bind:value={exportType}>
						<option value="LUT">Under LUT (no IGST)</option><option value="IGST">With IGST payment (refund claim)</option>
						<option value="SEZ_LUT">SEZ under LUT</option><option value="SEZ_IGST">SEZ with IGST payment</option>
					</select></div>
				<div class="form-group"><label class="form-label" for="cu">Currency</label>
					<select id="cu" class="form-input" bind:value={currency} onchange={fetchRate}>{#each currencies.filter((x) => x.code !== 'INR') as x}<option value={x.code}>{x.code}</option>{/each}</select></div>
				<div class="form-group"><label class="form-label" for="er">INR per 1 {currency}</label><input id="er" class="form-input" bind:value={exchangeRate} required /><div style="font-size: 0.75rem; color: var(--text-secondary);">{rateNote}</div></div>
				<div class="form-group"><label class="form-label" for="in">Incoterm</label>
					<select id="in" class="form-input" bind:value={incoterm}>{#each ['EXW', 'FCA', 'FAS', 'FOB', 'CFR', 'CIF', 'CPT', 'CIP', 'DAP', 'DPU', 'DDP'] as t}<option>{t}</option>{/each}</select></div>
				<div class="form-group"><label class="form-label" for="pt">Payment terms</label><input id="pt" class="form-input" bind:value={paymentTerms} placeholder="30% advance, balance against BL copy" /></div>
				<div class="form-group"><label class="form-label" for="bc">Buyer country</label><input id="bc" class="form-input" bind:value={buyerCountry} /></div>
				<div class="form-group"><label class="form-label" for="bt">Buyer tax ID (VAT / EORI)</label><input id="bt" class="form-input" bind:value={buyerTaxId} /></div>
				<div class="form-group"><label class="form-label" for="cn">Consignee (if different)</label><input id="cn" class="form-input" bind:value={consigneeName} /></div>
				<div class="form-group"><label class="form-label" for="ca">Consignee address</label><input id="ca" class="form-input" bind:value={consigneeAddress} /></div>
				<div class="form-group"><label class="form-label" for="np">Notify party</label><input id="np" class="form-input" bind:value={notifyParty} /></div>
			</div>
		</div>

		<div class="card" style="padding: 1.25rem; margin-bottom: 1rem;">
			<h3 style="margin-bottom: 0.75rem;">Shipment</h3>
			<div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(210px, 1fr)); gap: 0.75rem;">
				<div class="form-group"><label class="form-label" for="cd">Country of destination</label><input id="cd" class="form-input" bind:value={countryDestination} /></div>
				<div class="form-group"><label class="form-label" for="pl">Port of loading</label><input id="pl" class="form-input" bind:value={portLoading} /></div>
				<div class="form-group"><label class="form-label" for="pc">Port code</label><input id="pc" class="form-input" bind:value={portCode} /></div>
				<div class="form-group"><label class="form-label" for="pd">Port of discharge</label><input id="pd" class="form-input" bind:value={portDischarge} /></div>
				<div class="form-group"><label class="form-label" for="fd">Final destination</label><input id="fd" class="form-input" bind:value={finalDestination} /></div>
				<div class="form-group"><label class="form-label" for="vf">Vessel / flight</label><input id="vf" class="form-input" bind:value={vesselFlight} /></div>
				<div class="form-group"><label class="form-label" for="fr">Freight ({currency}, included in prices)</label><input id="fr" class="form-input" bind:value={freight} /></div>
				<div class="form-group"><label class="form-label" for="ins">Insurance ({currency}, included in prices)</label><input id="ins" class="form-input" bind:value={insurance} /></div>
			</div>
		</div>

		<div class="card" style="padding: 1.25rem; margin-bottom: 1rem; overflow-x: auto;">
			<h3 style="margin-bottom: 0.75rem;">Goods (prices in {currency})</h3>
			<table style="width: 100%; font-size: 0.8125rem;">
				<thead><tr style="text-align: left; color: var(--text-secondary);"><th>Description</th><th>HS code</th><th>Qty</th><th>Unit</th><th>Price</th><th>Disc %</th>{#if exportType.includes('IGST')}<th>IGST %</th>{/if}<th></th></tr></thead>
				<tbody>
					{#each lines as l, i}
						<tr>
							<td><input class="form-input" bind:value={l.item_description} required /></td>
							<td><input class="form-input" style="width: 110px;" bind:value={l.hsn_sac} /></td>
							<td><input class="form-input" style="width: 70px;" bind:value={l.quantity} required /></td>
							<td><select class="form-input" bind:value={l.unit_code}>{#each units as u}<option value={u.code}>{u.code}</option>{/each}</select></td>
							<td><input class="form-input" style="width: 100px;" bind:value={l.unit_price} required /></td>
							<td><input class="form-input" style="width: 60px;" bind:value={l.discount_rate} /></td>
							{#if exportType.includes('IGST')}<td><input class="form-input" style="width: 60px;" bind:value={l.tax_rate} /></td>{/if}
							<td>{#if lines.length > 1}<button type="button" class="btn btn-secondary" onclick={() => lines.splice(i, 1)}>✕</button>{/if}</td>
						</tr>
					{/each}
				</tbody>
			</table>
			<button type="button" class="btn btn-secondary" style="margin-top: 0.75rem;" onclick={addLine}>+ Add line</button>
			<p style="margin-top: 0.75rem; font-size: 0.9rem;">Invoice value: <strong>{currency} {fcTotal.toFixed(2)}</strong> ≈ <strong>Rs. {inrTotal.toFixed(2)}</strong> at {exchangeRate || '?'}</p>
		</div>
		<button class="btn btn-primary" disabled={busy}>{busy ? 'Creating...' : 'Create export invoice'}</button>
	</form>
</div>
