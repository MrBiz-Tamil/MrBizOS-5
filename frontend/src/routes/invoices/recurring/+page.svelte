<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';

	let schedules = $state<any[]>([]);
	let error = $state('');
	let notice = $state('');
	let loading = $state(true);

	async function load() {
		try {
			schedules = await api.listRecurring();
		} catch (e: any) {
			error = e.message;
		} finally {
			loading = false;
		}
	}
	onMount(load);

	async function setStatus(id: string, status: string) {
		error = notice = '';
		try {
			await api.setRecurringStatus(id, status);
			await load();
		} catch (e: any) {
			error = e.message;
		}
	}

	async function remove(id: string) {
		if (!confirm('Delete this schedule? Invoices already issued are not affected.')) return;
		error = notice = '';
		try {
			await api.deleteRecurring(id);
			notice = 'Schedule deleted.';
			await load();
		} catch (e: any) {
			error = e.message;
		}
	}

	const color: Record<string, string> = { ACTIVE: '#34d399', PAUSED: '#fbbf24', ENDED: '#94a3b8' };
</script>

<svelte:head><title>Recurring Invoices</title></svelte:head>

<div>
	<h1 style="margin-bottom: 0.25rem;">Recurring invoices</h1>
	<p style="color: var(--text-secondary); margin-bottom: 1rem;">
		Invoices issued automatically on a schedule. To add one, open any invoice and choose <strong>Repeat this invoice</strong>.
	</p>
	{#if notice}<div class="card" style="padding: 0.6rem 1rem; margin-bottom: 1rem; color: #34d399;">{notice}</div>{/if}
	{#if error}<div class="card" style="padding: 0.6rem 1rem; margin-bottom: 1rem; color: #fb7185;">{error}</div>{/if}

	<div class="card" style="padding: 1rem; overflow-x: auto;">
		<table style="width: 100%; font-size: 0.875rem;">
			<thead>
				<tr style="text-align: left; color: var(--text-secondary);">
					<th>Name</th><th>Customer</th><th>Repeats</th><th>Next invoice</th><th>Issued</th><th>Status</th><th></th>
				</tr>
			</thead>
			<tbody>
				{#each schedules as s}
					<tr>
						<td>{s.name}{#if s.last_error}<div style="color: #fb7185; font-size: 0.75rem;">{s.last_error}</div>{/if}</td>
						<td>{s.customer_name}</td>
						<td>{s.frequency.toLowerCase()}</td>
						<td>{s.status === 'ENDED' ? '-' : s.next_run_date}</td>
						<td>{s.runs_count}{#if s.last_invoice_id} · <a href="/invoices/{s.last_invoice_id}">latest</a>{/if}</td>
						<td style="color: {color[s.status]}; font-weight: 600;">{s.status}</td>
						<td style="white-space: nowrap;">
							{#if s.status === 'ACTIVE'}
								<button class="btn btn-secondary" onclick={() => setStatus(s.id, 'PAUSED')}>Pause</button>
							{:else if s.status === 'PAUSED'}
								<button class="btn btn-primary" onclick={() => setStatus(s.id, 'ACTIVE')}>Resume</button>
							{/if}
							<button class="btn btn-secondary" onclick={() => remove(s.id)}>Delete</button>
						</td>
					</tr>
				{:else}
					<tr><td colspan="7" style="color: var(--text-secondary);">{loading ? 'Loading...' : 'No recurring invoices yet.'}</td></tr>
				{/each}
			</tbody>
		</table>
	</div>
</div>
