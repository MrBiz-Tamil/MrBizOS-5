<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { api } from '$lib/api';

	let q = $state<any>(null);
	let company = $state<any>(null);
	let error = $state('');
	let busy = $state(false);

	async function load() {
		try {
			[q, company] = await Promise.all([api.getQuotation($page.params.id!), api.getCompany()]);
		} catch (e: any) {
			error = e.message;
		}
	}
	onMount(load);

	async function setStatus(s: string) {
		busy = true;
		error = '';
		try {
			q = await api.setQuotationStatus(q.id, s);
		} catch (e: any) {
			error = e.message;
		} finally {
			busy = false;
		}
	}

	async function convert() {
		if (!confirm('Create a real tax invoice from this quotation?')) return;
		busy = true;
		error = '';
		try {
			const inv = await api.convertQuotation(q.id);
			window.location.href = `/invoices/${inv.id}`;
		} catch (e: any) {
			error = e.message;
			busy = false;
		}
	}

	async function remove() {
		if (!confirm('Delete this quotation?')) return;
		try {
			await api.deleteQuotation(q.id);
			window.location.href = '/quotations';
		} catch (e: any) {
			error = e.message;
		}
	}

	const title = $derived(q?.kind === 'PROFORMA' ? 'PROFORMA INVOICE' : 'QUOTATION');
</script>

<svelte:head>
	<title>{q ? q.quote_number : 'Quotation'}</title>
	<style>
		@media print {
			.no-print, .sidebar, .topbar, header, nav { display: none !important; }
			.quote-sheet { box-shadow: none !important; border: none !important; }
			body { background: #fff !important; }
		}
	</style>
</svelte:head>

<div>
	<div class="no-print" style="display: flex; gap: 0.5rem; flex-wrap: wrap; margin-bottom: 1rem; align-items: center;">
		<a href="/quotations" style="font-size: 0.8125rem;">← All quotations</a>
		{#if q}
			<span style="margin-left: auto;"></span>
			<button class="btn btn-secondary" onclick={() => window.print()}>Print / Save PDF</button>
			{#if q.status === 'DRAFT'}<button class="btn btn-secondary" disabled={busy} onclick={() => setStatus('SENT')}>Mark sent</button>{/if}
			{#if q.status === 'SENT' || q.status === 'DRAFT'}
				<button class="btn btn-secondary" disabled={busy} onclick={() => setStatus('ACCEPTED')}>Mark accepted</button>
				<button class="btn btn-secondary" disabled={busy} onclick={() => setStatus('REJECTED')}>Mark rejected</button>
			{/if}
			{#if q.status !== 'CONVERTED' && q.status !== 'REJECTED'}
				<button class="btn btn-primary" disabled={busy} onclick={convert}>Convert to invoice</button>
			{/if}
			{#if q.status === 'CONVERTED' && q.converted_invoice_id}
				<a class="btn btn-primary" href="/invoices/{q.converted_invoice_id}">View invoice</a>
			{:else if q.status !== 'CONVERTED'}
				<button class="btn btn-secondary" onclick={remove}>Delete</button>
			{/if}
		{/if}
	</div>
	{#if error}<div class="card no-print" style="padding: 0.6rem 1rem; margin-bottom: 1rem; color: #fb7185;">{error}</div>{/if}

	{#if q && company}
		<div class="card quote-sheet" style="padding: 2rem; background: #fff; color: #0f172a; max-width: 860px; margin: 0 auto;">
			<div style="display: flex; justify-content: space-between; gap: 1rem; flex-wrap: wrap;">
				<div>
					<div style="font-size: 1.4rem; font-weight: 700;">{company.legal_name || company.name}</div>
					<div style="font-size: 0.8125rem; color: #475569;">{[company.address, company.city, company.state, company.pincode].filter(Boolean).join(', ')}</div>
					{#if company.gstin}<div style="font-size: 0.8125rem; color: #475569;">GSTIN: {company.gstin}</div>{/if}
				</div>
				<div style="text-align: right;">
					<div style="font-size: 1.25rem; font-weight: 700; color: #4338ca;">{title}</div>
					<div style="font-family: monospace;">{q.quote_number}</div>
					<div style="font-size: 0.8125rem; color: #475569;">Date: {q.quote_date}</div>
					<div style="font-size: 0.8125rem; color: #475569;">Valid until: {q.valid_until}{#if q.expired} (expired){/if}</div>
				</div>
			</div>
			<div style="margin: 1.25rem 0; font-size: 0.875rem;">
				<div style="color: #64748b; font-size: 0.75rem; text-transform: uppercase;">Prepared for</div>
				<div style="font-weight: 600;">{q.customer_name}</div>
				{#if q.customer_gstin}<div>GSTIN: {q.customer_gstin}</div>{/if}
				{#if q.billing_address}<div style="color: #475569;">{q.billing_address}</div>{/if}
				<div style="color: #475569;">Place of supply: {q.supply_state}</div>
			</div>
			<table style="width: 100%; font-size: 0.8125rem; border-collapse: collapse;">
				<thead>
					<tr style="background: #eef2ff; text-align: left;"><th style="padding: 6px;">#</th><th>Description</th><th>HSN/SAC</th><th style="text-align: right;">Qty</th><th style="text-align: right;">Rate</th><th style="text-align: right;">GST</th><th style="text-align: right; padding-right: 6px;">Amount</th></tr>
				</thead>
				<tbody>
					{#each q.items as it, i}
						<tr style="border-bottom: 1px solid #e2e8f0;">
							<td style="padding: 6px;">{i + 1}</td><td>{it.item_description}</td><td>{it.hsn_sac}</td>
							<td style="text-align: right;">{it.quantity} {it.unit_code}</td><td style="text-align: right;">{it.unit_price}</td>
							<td style="text-align: right;">{it.tax_rate}%</td><td style="text-align: right; padding-right: 6px;">{it.total_amount}</td>
						</tr>
					{/each}
				</tbody>
			</table>
			<div style="margin-left: auto; width: 280px; margin-top: 1rem; font-size: 0.875rem;">
				<div style="display: flex; justify-content: space-between;"><span>Taxable value</span><span>{q.taxable_amount}</span></div>
				{#if q.is_inter_state}
					<div style="display: flex; justify-content: space-between;"><span>IGST</span><span>{q.igst_total}</span></div>
				{:else}
					<div style="display: flex; justify-content: space-between;"><span>CGST</span><span>{q.cgst_total}</span></div>
					<div style="display: flex; justify-content: space-between;"><span>SGST</span><span>{q.sgst_total}</span></div>
				{/if}
				<div style="display: flex; justify-content: space-between;"><span>Round off</span><span>{q.round_off}</span></div>
				<div style="display: flex; justify-content: space-between; font-weight: 700; font-size: 1.05rem; border-top: 2px solid #4338ca; margin-top: 4px; padding-top: 4px;"><span>Total</span><span>Rs. {q.grand_total}</span></div>
			</div>
			{#if q.terms}<p style="margin-top: 1.25rem; font-size: 0.8125rem;"><strong>Terms:</strong> {q.terms}</p>{/if}
			{#if q.notes}<p style="font-size: 0.8125rem;"><strong>Notes:</strong> {q.notes}</p>{/if}
			<p style="margin-top: 1.25rem; font-size: 0.75rem; color: #64748b;">This is a {title.toLowerCase()}, not a tax invoice. A tax invoice will be issued on acceptance.</p>
		</div>
	{/if}
</div>
