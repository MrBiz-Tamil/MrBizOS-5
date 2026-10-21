<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';

	let invoices = $state<any[]>([]);
	let filter = $state('OPEN');
	let error = $state('');
	let notice = $state('');
	let reference = $state<Record<string, string>>({});

	async function load() {
		error = '';
		try {
			invoices = await api.adminBillingInvoices(filter);
		} catch (e: any) {
			error = e.message;
		}
	}
	onMount(load);

	async function markPaid(id: string) {
		error = notice = '';
		try {
			await api.adminMarkPaid(id, 'BANK_TRANSFER', reference[id] || '');
			notice = 'Payment recorded and plan activated.';
			await load();
		} catch (e: any) {
			error = e.message;
		}
	}

	async function runNow() {
		error = notice = '';
		try {
			const r = await api.adminRunBilling();
			notice = `Lifecycle run: ${JSON.stringify(r)}`;
			await load();
		} catch (e: any) {
			error = e.message;
		}
	}
</script>

<svelte:head><title>Plan Invoices</title></svelte:head>

<div>
	<h1 style="margin-bottom: 0.25rem;">Plan invoices</h1>
	<p style="color: var(--text-secondary); margin-bottom: 1rem;">Invoices issued to companies for their subscription. Record a payment when the transfer arrives; the plan activates immediately.</p>
	{#if notice}<div class="card" style="padding: 0.6rem 1rem; margin-bottom: 1rem; color: #34d399;">{notice}</div>{/if}
	{#if error}<div class="card" style="padding: 0.6rem 1rem; margin-bottom: 1rem; color: #fb7185;">{error}</div>{/if}
	<div style="display: flex; gap: 0.75rem; margin-bottom: 1rem;">
		<select class="form-input" style="max-width: 180px;" bind:value={filter} onchange={load}>
			<option value="OPEN">Open</option><option value="PAID">Paid</option><option value="">All</option>
		</select>
		<button class="btn btn-secondary" onclick={runNow}>Run billing cycle now</button>
	</div>
	<div class="card" style="padding: 1rem; overflow-x: auto;">
		<table style="width: 100%; font-size: 0.875rem;">
			<thead><tr style="text-align: left; color: var(--text-secondary);"><th>Number</th><th>Company</th><th>Plan</th><th>Total</th><th>Due</th><th>Status</th><th></th></tr></thead>
			<tbody>
				{#each invoices as i}
					<tr>
						<td>{i.invoice_number}</td><td>{i.tenant_name}</td><td>{i.plan_code} ({i.billing_cycle})</td><td>Rs. {i.total_amount}</td><td>{i.due_date}</td><td>{i.status}</td>
						<td>
							{#if i.status === 'OPEN'}
								<input class="form-input" style="width: 140px; display: inline-block;" placeholder="UTR / reference" bind:value={reference[i.id]} />
								<button class="btn btn-primary" onclick={() => markPaid(i.id)}>Mark paid</button>
							{:else}{i.payment_reference}{/if}
						</td>
					</tr>
				{:else}
					<tr><td colspan="7" style="color: var(--text-secondary);">Nothing here.</td></tr>
				{/each}
			</tbody>
		</table>
	</div>
</div>
