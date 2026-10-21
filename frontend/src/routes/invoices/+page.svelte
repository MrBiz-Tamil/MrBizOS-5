<script lang="ts">
	import { onMount } from 'svelte';
	import { api, type Invoice } from '$lib/api';

	let invoices = $state<Invoice[]>([]);
	let loading = $state(true);
	let error = $state('');
	let searchQuery = $state('');
	let statusFilter = $state('ALL');

	onMount(async () => {
		try {
			invoices = await api.listInvoices();
		} catch (err: any) {
			error = err.message || 'Failed to fetch invoices';
		} finally {
			loading = false;
		}
	});

	let filteredInvoices = $derived(
		invoices.filter((inv) => {
			const matchesSearch =
				inv.invoice_number.toLowerCase().includes(searchQuery.toLowerCase()) ||
				inv.customer_name.toLowerCase().includes(searchQuery.toLowerCase()) ||
				(inv.customer_gstin && inv.customer_gstin.toLowerCase().includes(searchQuery.toLowerCase()));
			const matchesStatus = statusFilter === 'ALL' || inv.status === statusFilter;
			return matchesSearch && matchesStatus;
		})
	);

	function formatCurrency(val: string | number): string {
		const num = typeof val === 'string' ? parseFloat(val) : val;
		return new Intl.NumberFormat('en-IN', { style: 'currency', currency: 'INR' }).format(num);
	}
</script>

<svelte:head>
	<title>Sales Invoices - SaaS Billing</title>
</svelte:head>

<div style="display: flex; flex-direction: column; gap: 1.5rem;">
	<!-- Page Header -->
	<div style="display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 1rem;">
		<div>
			<h1 style="font-size: 1.75rem; margin-bottom: 0.25rem;">Sales Invoices</h1>
			<p style="font-size: 0.875rem; color: var(--text-secondary);">Manage GST tax invoices, payment statuses, and customer billing records.</p>
		</div>
		<a href="/invoices/new" class="btn btn-primary">
			+ Create New Invoice
		</a>
	</div>

	<!-- Controls & Search Filter Bar -->
	<div class="card" style="padding: 1rem 1.25rem; display: flex; flex-wrap: wrap; gap: 1rem; align-items: center; justify-content: space-between;">
		<div style="display: flex; gap: 0.75rem; flex: 1; max-width: 400px;">
			<input
				type="text"
				class="form-input"
				placeholder="Search by invoice #, customer name, GSTIN..."
				bind:value={searchQuery}
			/>
		</div>

		<div style="display: flex; gap: 0.5rem; align-items: center;">
			<span style="font-size: 0.8125rem; color: var(--text-secondary);">Status:</span>
			<select class="form-select" style="width: auto; padding: 0.4rem 0.75rem;" bind:value={statusFilter}>
				<option value="ALL">All Statuses</option>
				<option value="ISSUED">Issued (Unpaid)</option>
				<option value="PARTIALLY_PAID">Partially Paid</option>
				<option value="PAID">Paid</option>
				<option value="CANCELLED">Cancelled</option>
			</select>
		</div>
	</div>

	<!-- Invoices Table -->
	<div class="table-container card" style="padding: 0;">
		<table class="data-table">
			<thead>
				<tr>
					<th>Invoice Number</th>
					<th>Customer</th>
					<th>Invoice Date</th>
					<th>Due Date</th>
					<th class="text-right">Taxable</th>
					<th class="text-right">Grand Total</th>
					<th class="text-right">Balance Due</th>
					<th>Status</th>
					<th class="text-right">Actions</th>
				</tr>
			</thead>
			<tbody>
				{#if loading}
					<tr>
						<td colspan="9" class="text-center" style="padding: 3rem; color: var(--text-muted);">
							Loading invoices...
						</td>
					</tr>
				{:else if error}
					<tr>
						<td colspan="9" class="text-center" style="padding: 2rem; color: var(--rose);">
							⚠️ {error}
						</td>
					</tr>
				{:else if filteredInvoices.length === 0}
					<tr>
						<td colspan="9" class="text-center" style="padding: 3rem; color: var(--text-muted);">
							No invoices match the selected filter.
						</td>
					</tr>
				{:else}
					{#each filteredInvoices as inv}
						<tr>
							<td>
								<a href={`/invoices/${inv.id}`} style="color: #818cf8; font-family: var(--font-mono); font-weight: 600; text-decoration: none;">
									{inv.invoice_number}
								</a>
							</td>
							<td>
								<div style="font-weight: 500;">{inv.customer_name}</div>
								{#if inv.customer_gstin}
									<div style="font-size: 0.7rem; color: var(--text-muted); font-family: var(--font-mono);">{inv.customer_gstin}</div>
								{/if}
							</td>
							<td style="color: var(--text-secondary);">{new Date(inv.invoice_date).toLocaleDateString('en-IN')}</td>
							<td style="color: var(--text-secondary);">{new Date(inv.due_date).toLocaleDateString('en-IN')}</td>
							<td class="text-right font-mono" style="color: var(--text-secondary);">{formatCurrency(inv.taxable_amount)}</td>
							<td class="text-right font-mono" style="font-weight: 600;">{formatCurrency(inv.grand_total)}</td>
							<td class="text-right font-mono" style="font-weight: 600;" class:text-amber={parseFloat(inv.balance_due) > 0}>
								{formatCurrency(inv.balance_due)}
							</td>
							<td>
								<span class={`badge badge-${inv.status.toLowerCase().replace('_', '-')}`}>
									<span class="badge-dot"></span>
									{inv.status}
								</span>
							</td>
							<td class="text-right">
								<div style="display: inline-flex; gap: 0.35rem; align-items: center;">
									<a href={`/invoices/${inv.id}`} class="btn btn-secondary btn-sm" title="View details & print">
										View
									</a>
									<div style="display: inline-flex; border: 1px solid rgba(99, 102, 241, 0.4); border-radius: var(--radius-md); overflow: hidden;">
										<a href={`/api/invoices/${inv.id}/pdf?size=a4`} target="_blank" class="btn btn-secondary btn-sm" title="Download standard A4 PDF" style="border: none; border-radius: 0; padding: 0.25rem 0.5rem; font-size: 0.75rem; color: #a5b4fc;">
											A4 ⬇
										</a>
										<a href={`/api/invoices/${inv.id}/pdf?size=a5`} target="_blank" class="btn btn-secondary btn-sm" title="Download compact A5 (half-sheet) PDF" style="border: none; border-radius: 0; border-left: 1px solid rgba(99, 102, 241, 0.3); padding: 0.25rem 0.5rem; font-size: 0.75rem; color: #34d399;">
											A5 ⬇
										</a>
									</div>
								</div>
							</td>
						</tr>
					{/each}
				{/if}
			</tbody>
		</table>
	</div>
</div>
