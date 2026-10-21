<script lang="ts">
	import { onMount } from 'svelte';
	import { api, type Payment } from '$lib/api';

	let payments = $state<Payment[]>([]);
	let loading = $state(true);
	let error = $state('');

	onMount(async () => {
		try {
			payments = await api.listPayments();
		} catch (err: any) {
			error = err.message || 'Failed to fetch payments';
		} finally {
			loading = false;
		}
	});

	function formatCurrency(val: string | number): string {
		const num = typeof val === 'string' ? parseFloat(val) : val;
		return new Intl.NumberFormat('en-IN', { style: 'currency', currency: 'INR' }).format(num);
	}
</script>

<svelte:head>
	<title>Payments Received - SaaS Billing</title>
</svelte:head>

<div style="display: flex; flex-direction: column; gap: 1.5rem;">
	<div style="display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 1rem;">
		<div>
			<h1 style="font-size: 1.75rem; margin-bottom: 0.25rem;">Payments Received</h1>
			<p style="font-size: 0.875rem; color: var(--text-secondary);">Immutable record of customer payment receipts and allocations.</p>
		</div>
	</div>

	<div class="table-container card" style="padding: 0;">
		<table class="data-table">
			<thead>
				<tr>
					<th>Payment Number</th>
					<th>Date</th>
					<th>Method</th>
					<th>Reference #</th>
					<th class="text-right">Amount Received</th>
					<th>Notes</th>
					<th class="text-right">Invoice</th>
				</tr>
			</thead>
			<tbody>
				{#if loading}
					<tr>
						<td colspan="7" class="text-center" style="padding: 3rem; color: var(--text-muted);">
							Loading payment records...
						</td>
					</tr>
				{:else if error}
					<tr>
						<td colspan="7" class="text-center" style="padding: 2rem; color: var(--rose);">
							⚠️ {error}
						</td>
					</tr>
				{:else if payments.length === 0}
					<tr>
						<td colspan="7" class="text-center" style="padding: 3rem; color: var(--text-muted);">
							No payment transactions recorded yet.
						</td>
					</tr>
				{:else}
					{#each payments as pay}
						<tr>
							<td style="font-family: var(--font-mono); color: #34d399; font-weight: 600;">
								{pay.payment_number}
							</td>
							<td style="color: var(--text-secondary);">{new Date(pay.payment_date).toLocaleDateString('en-IN')}</td>
							<td>
								<span style="font-size: 0.75rem; background: rgba(255,255,255,0.06); padding: 0.25rem 0.6rem; border-radius: var(--radius-sm);">
									{pay.payment_method}
								</span>
							</td>
							<td style="font-family: var(--font-mono); font-size: 0.8125rem; color: var(--text-secondary);">
								{pay.reference_number || '-'}
							</td>
							<td class="text-right font-mono text-emerald" style="font-weight: 700;">
								{formatCurrency(pay.amount)}
							</td>
							<td style="font-size: 0.8125rem; color: var(--text-muted);">
								{pay.notes || '-'}
							</td>
							<td class="text-right">
								<a href={`/invoices/${pay.invoice_id}`} class="btn btn-secondary btn-sm">
									View Invoice
								</a>
							</td>
						</tr>
					{/each}
				{/if}
			</tbody>
		</table>
	</div>
</div>
