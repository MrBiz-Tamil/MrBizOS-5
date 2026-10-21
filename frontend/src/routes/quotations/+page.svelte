<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';

	let list = $state<any[]>([]);
	let error = $state('');
	let loading = $state(true);

	onMount(async () => {
		try {
			list = await api.listQuotations();
		} catch (e: any) {
			error = e.message;
		} finally {
			loading = false;
		}
	});

	const color: Record<string, string> = { DRAFT: '#94a3b8', SENT: '#38bdf8', ACCEPTED: '#34d399', REJECTED: '#fb7185', CONVERTED: '#a78bfa' };
</script>

<svelte:head><title>Quotations</title></svelte:head>

<div>
	<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem; flex-wrap: wrap; gap: 0.5rem;">
		<div>
			<h1 style="margin-bottom: 0.25rem;">Quotations & Proforma</h1>
			<p style="color: var(--text-secondary);">Price estimates for customers. They are not tax invoices; convert an accepted one into an invoice in one click.</p>
		</div>
		<a href="/quotations/new" class="btn btn-primary">+ New quotation</a>
	</div>
	{#if error}<div class="card" style="padding: 0.6rem 1rem; color: #fb7185;">{error}</div>{/if}
	<div class="card" style="padding: 1rem; overflow-x: auto;">
		<table style="width: 100%; font-size: 0.875rem;">
			<thead><tr style="text-align: left; color: var(--text-secondary);"><th>Number</th><th>Customer</th><th>Date</th><th>Valid until</th><th>Total</th><th>Status</th></tr></thead>
			<tbody>
				{#each list as q}
					<tr>
						<td><a href="/quotations/{q.id}">{q.quote_number}</a></td>
						<td>{q.customer_name}</td>
						<td>{q.quote_date}</td>
						<td>{q.valid_until}{#if q.expired} <span style="color: #fbbf24;">(expired)</span>{/if}</td>
						<td>Rs. {q.grand_total}</td>
						<td style="color: {color[q.status]}; font-weight: 600;">{q.status}</td>
					</tr>
				{:else}
					<tr><td colspan="6" style="color: var(--text-secondary);">{loading ? 'Loading...' : 'No quotations yet.'}</td></tr>
				{/each}
			</tbody>
		</table>
	</div>
</div>
