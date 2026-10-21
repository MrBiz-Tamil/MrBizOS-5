<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import InvoiceTemplates from '$lib/components/InvoiceTemplates.svelte';

	let inv = $state<any>(null);
	let company = $state<any>(null);
	let error = $state('');
	let loading = $state(true);
	let extra = $state<any>({});

	onMount(async () => {
		try {
			const res = await fetch(`/api/public/invoices/${encodeURIComponent($page.params.token!)}`, { referrerPolicy: 'no-referrer' });
			const data = await res.json().catch(() => ({}));
			if (!res.ok) throw new Error(data.error || 'This link is invalid or has expired');
			inv = data.invoice;
			company = data.company;
			extra = {
				poNumber: inv.po_number || '',
				vehicleNumber: inv.vehicle_number || '',
				ewayBill: inv.eway_bill || '',
				transporterName: inv.transporter_name || '',
				lrNumber: inv.lr_number || '',
				deliveryPlace: inv.delivery_place || inv.supply_state || '',
				terms: inv.payment_mode || '',
				jurisdictionCity: inv.jurisdiction_city || company.city || '',
				interestRate: inv.interest_rate || '',
				returnPolicy: inv.return_policy || '',
				reverseCharge: inv.reverse_charge || false,
				subscriptionPeriodStart: inv.subscription_period_start || '',
				subscriptionPeriodEnd: inv.subscription_period_end || '',
				grossWeight: inv.gross_weight || '',
				netWeight: inv.net_weight || '',
				cashierName: inv.cashier_name || '',
				counterNumber: inv.counter_number || '',
				projectName: inv.project_name || '',
				signatoryTitle: company.signatory_title || 'Authorized Signatory',
				signatureUrl: company.signature_url || '',
				showBankDetails: inv.template_options?.show_bank_details ?? company.template_options?.show_bank_details ?? true,
				showUPIQR: inv.template_options?.show_upi_qr ?? company.template_options?.show_upi_qr ?? true,
				showTaxSummary: inv.template_options?.show_tax_summary ?? company.template_options?.show_tax_summary ?? true
			};
		} catch (e: any) {
			error = e.message;
		} finally {
			loading = false;
		}
	});
</script>

<svelte:head>
	<title>{inv ? `Invoice ${inv.invoice_number}` : 'Invoice'}</title>
	<meta name="robots" content="noindex, nofollow" />
	<meta name="referrer" content="no-referrer" />
	<style>
		@page { size: A4 portrait; margin: 8mm 10mm; }
		@media print { .no-print { display: none !important; } body { background: #fff !important; } }
	</style>
</svelte:head>

<div style="min-height: 100vh; background: #0b0f19; padding: 1rem;">
	{#if loading}
		<p style="color: var(--text-secondary); text-align: center; padding: 3rem;">Loading invoice...</p>
	{:else if error}
		<div class="card" style="max-width: 460px; margin: 3rem auto; padding: 1.5rem; text-align: center;">
			<h2 style="margin-bottom: 0.5rem;">Invoice unavailable</h2>
			<p style="color: var(--text-secondary);">{error}. Please ask the sender for a new link.</p>
		</div>
	{:else if inv && company}
		<div class="no-print" style="max-width: 860px; margin: 0 auto 1rem; display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 0.5rem;">
			<div style="color: var(--text-secondary); font-size: 0.875rem;">From <strong style="color: #fff;">{company.legal_name || company.name}</strong></div>
			<button class="btn btn-primary" onclick={() => window.print()}>Print / Save as PDF</button>
		</div>
		<div style="max-width: 900px; margin: 0 auto;">
			<InvoiceTemplates invoice={inv} {company} templateId={company.template_id || '1'} copyType="ORIGINAL" taxMode="WITH_GST" paperSize="A4" extraDetails={extra} />
		</div>
	{/if}
</div>
