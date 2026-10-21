<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { page } from '$app/stores';
	import { api, setSelectedTenantId } from '$lib/api';
	import TradeDocument from '$lib/components/TradeDocument.svelte';

	const type = $derived($page.params.type!);
	const id = $derived($page.params.id!);
	const printMode = typeof window !== 'undefined' && new URLSearchParams(window.location.search).get('print') === '1';

	let data = $state<any>(null);
	let extras = $state<Record<string, string>>({});
	let error = $state('');
	let saved = $state('');

	// checklist row that stores each document's editable fields (as JSON in its notes)
	const checklistType: Record<string, string> = {
		'commercial-invoice': 'COMMERCIAL_INVOICE', 'invoice-cum-packing-list': 'COMMERCIAL_INVOICE', 'proforma-invoice': 'PROFORMA_INVOICE',
		'packing-list': 'PACKING_LIST', 'bill-of-exchange': 'BILL_OF_EXCHANGE', 'shipping-instructions': 'SLI', 'bank-cover-letter': 'BANK_COVER',
		'coo-application': 'COO', 'purchase-order': 'PURCHASE_ORDER', 'remittance-request': 'REMITTANCE', grn: 'GRN', 'landed-cost-sheet': 'LANDED_COST'
	};

	const fields: Record<string, [string, string][]> = {
		'commercial-invoice': [['buyer_order_no', "Buyer's order no."], ['lc_no', 'Letter of credit no.']],
		'invoice-cum-packing-list': [['buyer_order_no', "Buyer's order no."], ['lc_no', 'Letter of credit no.']],
		'proforma-invoice': [['buyer_order_no', "Buyer's enquiry / order no."]],
		'packing-list': [['buyer_order_no', "Buyer's order no."]],
		'bill-of-exchange': [['tenor', 'Tenor (e.g. At 30 days from B/L date)'], ['payee', 'Pay to the order of'], ['bill_date', 'Bill date'], ['lc_no', 'LC no.'], ['lc_date', 'LC date'], ['issuing_bank', 'Issuing bank'], ['drawee', 'Drawee (default: buyer)'], ['drawee_address', 'Drawee address']],
		'shipping-instructions': [['forwarder', 'To (forwarder / shipping line)'], ['freight_terms', 'Freight (PREPAID / COLLECT)'], ['bl_type', 'B/L type and originals'], ['special', 'Special instructions']],
		'bank-cover-letter': [['bank_name', 'Bank'], ['bank_branch', 'Branch / address'], ['mode', 'Negotiation or collection'], ['lc_no', 'LC no.'], ['lc_date', 'LC date'], ['originals', 'Originals of each'], ['copies', 'Copies of each'], ['instructions', 'Extra instructions']],
		'coo-application': [['chamber', 'Issuing chamber / authority'], ['criterion', 'Origin criterion / remarks']],
		'purchase-order': [['supplier_address', 'Supplier address'], ['po_no', 'PO number'], ['po_date', 'PO date'], ['currency', 'Currency'], ['delivery_by', 'Delivery by'], ['payment_terms', 'Payment terms'], ['terms', 'Extra terms']],
		'remittance-request': [['bank_name', 'Remitting bank'], ['currency', 'Currency'], ['amount', 'Amount'], ['ben_name', 'Beneficiary name'], ['ben_address', 'Beneficiary address'], ['ben_country', 'Beneficiary country'], ['ben_bank', 'Beneficiary bank'], ['ben_swift', 'SWIFT / BIC'], ['ben_account', 'Account / IBAN'], ['supplier_invoice', 'Supplier invoice no.'], ['supplier_invoice_date', 'Supplier invoice date'], ['purpose_code', 'Purpose code (as your bank requires)'], ['remarks', 'Remarks']],
		grn: [['grn_no', 'GRN no.'], ['received_on', 'Received on'], ['location', 'Received at (warehouse)'], ['received_by', 'Received by'], ['remarks', 'Remarks']]
	};

	const lineFields = $derived(
		type === 'purchase-order' ? (data?.landed_cost?.items || []).map((it: any, i: number) => [`price_${i}`, `Unit price: ${it.description}`])
		: type === 'grn' ? (data?.landed_cost?.items || []).flatMap((it: any, i: number) => [[`rcv_${i}`, `Received qty: ${it.description}`], [`cond_${i}`, `Condition: ${it.description}`]])
		: []
	);

	const docRow = $derived((data?.documents || []).find((d: any) => d.doc_type === checklistType[type]));

	async function load() {
		try {
			const t = new URLSearchParams(window.location.search).get('tenant');
			if (t) setSelectedTenantId(t);
			data = await api.getShipmentDocData(id);
			try {
				const parsed = JSON.parse(docRow?.notes || '{}');
				extras = parsed && typeof parsed === 'object' ? parsed.extras || {} : {};
			} catch {
				extras = {};
			}
			if (printMode) {
				await tick();
				try { await document.fonts.ready; } catch {}
				document.documentElement.dataset.printReady = '1';
			}
		} catch (e: any) {
			error = e.message || 'Could not load the document';
			if (printMode) document.documentElement.dataset.printError = error;
		}
	}
	onMount(load);

	async function saveExtras() {
		saved = '';
		if (!docRow) return;
		try {
			await api.updateShipmentDoc(id, docRow.id, {
				status: docRow.status === 'PENDING' ? 'READY' : docRow.status,
				reference_no: docRow.reference_no,
				doc_date: docRow.doc_date || '',
				notes: JSON.stringify({ extras })
			});
			saved = 'Saved. The PDF download uses these fields.';
			data = await api.getShipmentDocData(id);
		} catch (e: any) {
			error = e.message;
		}
	}
</script>

<svelte:head>
	<title>{type} {data ? data.shipment.shipment_number : ''}</title>
	<style>
		@page { size: A4 portrait; margin: 10mm; }
		@media print { .no-print { display: none !important; } body { background: #fff !important; } }
	</style>
</svelte:head>

<div style="padding: 1rem;">
	{#if !printMode}
		<div class="no-print" style="max-width: 820px; margin: 0 auto 1rem;">
			<a href="/trade/shipments/{id}" style="font-size: 0.8125rem;">← Back to shipment</a>
			{#if error}<div class="card" style="padding: 0.6rem 1rem; margin: 0.5rem 0; color: #fb7185;">{error}</div>{/if}
			{#if data}
				<div style="display: flex; gap: 0.5rem; flex-wrap: wrap; margin: 0.75rem 0;">
					<button class="btn btn-primary" onclick={() => window.print()}>Print / Save as PDF</button>
					<a class="btn btn-secondary" href="/api/trade/shipments/{id}/documents/{type}/pdf" target="_blank" rel="noopener">Download PDF (server)</a>
				</div>
				{#if (fields[type] || []).length || lineFields.length}
					<details class="card" style="padding: 0.75rem 1rem;" open>
						<summary style="cursor: pointer; font-weight: 600;">Fill in details printed on this document</summary>
						<div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(240px, 1fr)); gap: 0.5rem; margin-top: 0.75rem;">
							{#each [...(fields[type] || []), ...lineFields] as [key, label]}
								<div class="form-group" style="margin: 0;"><label class="form-label" for={key}>{label}</label><input id={key} class="form-input" bind:value={extras[key]} /></div>
							{/each}
						</div>
						<button class="btn btn-secondary" style="margin-top: 0.75rem;" onclick={saveExtras}>Save details</button>
						{#if saved}<span style="margin-left: 0.5rem; color: #34d399; font-size: 0.8125rem;">{saved}</span>{/if}
					</details>
				{/if}
			{/if}
		</div>
	{/if}
	{#if data}<TradeDocument {type} {data} {extras} />{/if}
</div>
