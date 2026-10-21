<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';

	let overview = $state<any>(null);
	let invoices = $state<any[]>([]);
	let error = $state('');
	let notice = $state('');
	let cycle = $state('MONTHLY');

	async function load() {
		try {
			overview = await api.getBilling();
			invoices = await api.getBillingInvoices().catch(() => []);
		} catch (e: any) {
			error = e.message;
		}
	}
	onMount(load);

	async function choose(code: string) {
		error = notice = '';
		try {
			const inv = await api.chooseBillingPlan(code, cycle);
			notice = `Invoice ${inv.invoice_number} created for Rs. ${inv.total_amount}. Pay it using the details below; your plan activates once we confirm payment.`;
			await load();
		} catch (e: any) {
			error = e.message;
		}
	}

	const statusColor: Record<string, string> = { ACTIVE: '#34d399', TRIAL: '#38bdf8', PAST_DUE: '#fbbf24', SUSPENDED: '#fb7185' };
</script>

<svelte:head><title>Plan & Billing</title></svelte:head>

<div style="max-width: 820px;">
	<h1 style="margin-bottom: 0.25rem;">Plan & Billing</h1>
	<p style="color: var(--text-secondary); margin-bottom: 1.25rem;">Your subscription, plan invoices and how to pay.</p>

	{#if notice}<div class="card" style="padding: 0.75rem 1rem; margin-bottom: 1rem; color: #34d399;">{notice}</div>{/if}
	{#if error}<div class="card" style="padding: 0.75rem 1rem; margin-bottom: 1rem; color: #fb7185;">{error}</div>{/if}

	{#if overview}
		<div class="card" style="padding: 1.25rem; margin-bottom: 1.25rem;">
			<div style="display: flex; justify-content: space-between; flex-wrap: wrap; gap: 0.5rem;">
				<div>
					<div style="font-size: 0.8125rem; color: var(--text-secondary);">Current plan</div>
					<div style="font-size: 1.25rem; font-weight: 600;">{overview.plan_name}</div>
				</div>
				<div style="text-align: right;">
					<span style="color: {statusColor[overview.status] || '#fff'}; font-weight: 600;">{overview.status.replace('_', ' ')}</span>
					{#if overview.trial_days_left !== undefined && overview.trial_days_left !== null}
						<div style="font-size: 0.8125rem; color: var(--text-secondary);">{overview.trial_days_left} day(s) left in your trial</div>
					{:else if overview.renews_at}
						<div style="font-size: 0.8125rem; color: var(--text-secondary);">Renews {new Date(overview.renews_at).toLocaleDateString()}</div>
					{/if}
				</div>
			</div>
			{#if overview.status === 'SUSPENDED'}
				<p style="color: #fb7185; margin-top: 0.75rem; font-size: 0.875rem;">Creating new invoices is paused. You can still sign in, view and export all your data. Pay an open invoice or choose a plan to resume.</p>
			{/if}
		</div>

		<div class="card" style="padding: 1.25rem; margin-bottom: 1.25rem;">
			<h3 style="margin-bottom: 0.75rem;">Choose a plan</h3>
			<div style="margin-bottom: 0.75rem;">
				<label><input type="radio" bind:group={cycle} value="MONTHLY" /> Monthly</label>
				<label style="margin-left: 1rem;"><input type="radio" bind:group={cycle} value="YEARLY" /> Yearly (2 months free)</label>
			</div>
			<div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(210px, 1fr)); gap: 0.75rem;">
				{#each overview.plans as p}
					<div class="card" style="padding: 1rem; border: 1px solid {p.code === overview.plan_code ? '#6366f1' : 'rgba(255,255,255,0.08)'};">
						<div style="font-weight: 600;">{p.name}</div>
						<div style="font-size: 1.25rem; margin: 0.35rem 0;">Rs. {cycle === 'YEARLY' ? p.yearly_price : p.monthly_price}<span style="font-size: 0.75rem; color: var(--text-secondary);"> / {cycle === 'YEARLY' ? 'year' : 'month'} + GST</span></div>
						<div style="font-size: 0.8125rem; color: var(--text-secondary);">{p.max_users} users, {p.max_invoices_per_month} invoices/month</div>
						<button class="btn btn-primary" style="margin-top: 0.75rem; width: 100%;" onclick={() => choose(p.code)}>Choose {p.name}</button>
					</div>
				{/each}
			</div>
			{#if overview.pay_instructions}
				<p style="margin-top: 1rem; font-size: 0.8125rem; color: var(--text-secondary); white-space: pre-line;"><strong>How to pay:</strong> {overview.pay_instructions}</p>
			{/if}
		</div>

		<div class="card" style="padding: 1.25rem;">
			<h3 style="margin-bottom: 0.75rem;">Plan invoices</h3>
			{#if invoices.length === 0}
				<p style="color: var(--text-secondary);">No invoices yet.</p>
			{:else}
				<table style="width: 100%; font-size: 0.875rem;">
					<thead><tr style="text-align: left; color: var(--text-secondary);"><th>Number</th><th>Plan</th><th>Period</th><th>Total</th><th>Due</th><th>Status</th></tr></thead>
					<tbody>
						{#each invoices as i}
							<tr><td>{i.invoice_number}</td><td>{i.plan_code}</td><td>{i.period_start} to {i.period_end}</td><td>Rs. {i.total_amount}</td><td>{i.due_date}</td><td>{i.status}</td></tr>
						{/each}
					</tbody>
				</table>
			{/if}
		</div>
	{/if}
</div>
