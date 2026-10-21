<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';

	let list = $state<any[]>([]);
	let customers = $state<any[]>([]);
	let suppliers = $state<any[]>([]);
	let filter = $state('');
	let error = $state('');
	let showNew = $state(false);
	let busy = $state(false);
	let form = $state<any>({ direction: 'EXPORT', customer_id: '', supplier_id: '', counterparty_name: '', incoterm: '', mode: 'SEA', port_loading: '', port_discharge: '' });

	async function load() {
		try {
			list = await api.listShipments(filter);
		} catch (e: any) {
			error = e.message;
		}
	}
	onMount(async () => {
		await load();
		customers = await api.listCustomers().catch(() => []);
		suppliers = await api.listSuppliers().catch(() => []);
	});

	async function create(e: Event) {
		e.preventDefault();
		error = '';
		busy = true;
		try {
			const body = { ...form, customer_id: form.customer_id || null, supplier_id: form.supplier_id || null };
			const sh = await api.createShipment(body);
			window.location.href = `/trade/shipments/${sh.id}`;
		} catch (err: any) {
			error = err.message;
			busy = false;
		}
	}

	const color: Record<string, string> = { PLANNED: '#94a3b8', BOOKED: '#38bdf8', IN_TRANSIT: '#fbbf24', ARRIVED: '#a78bfa', CLEARED: '#34d399', COMPLETED: '#34d399', CANCELLED: '#fb7185' };
</script>

<svelte:head><title>Export / Import Shipments</title></svelte:head>

<div>
	<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem; flex-wrap: wrap; gap: 0.5rem;">
		<div>
			<h1 style="margin-bottom: 0.25rem;">Export / import shipments</h1>
			<p style="color: var(--text-secondary);">One file per shipment: documents checklist, packing list and costs.</p>
		</div>
		<button class="btn btn-primary" onclick={() => (showNew = !showNew)}>+ New shipment</button>
	</div>
	{#if error}<div class="card" style="padding: 0.6rem 1rem; margin-bottom: 1rem; color: #fb7185;">{error}</div>{/if}

	{#if showNew}
		<form class="card" style="padding: 1.25rem; margin-bottom: 1rem;" onsubmit={create}>
			<div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 0.75rem;">
				<div class="form-group"><label class="form-label" for="d">Direction</label>
					<select id="d" class="form-input" bind:value={form.direction}><option value="EXPORT">Export (we sell abroad)</option><option value="IMPORT">Import (we buy from abroad)</option></select></div>
				{#if form.direction === 'EXPORT'}
					<div class="form-group"><label class="form-label" for="c">Buyer</label>
						<select id="c" class="form-input" bind:value={form.customer_id}><option value="">- choose -</option>{#each customers as c}<option value={c.id}>{c.name}</option>{/each}</select></div>
				{:else}
					<div class="form-group"><label class="form-label" for="s">Supplier</label>
						<select id="s" class="form-input" bind:value={form.supplier_id}><option value="">- choose or type below -</option>{#each suppliers as s}<option value={s.id}>{s.name}</option>{/each}</select></div>
					<div class="form-group"><label class="form-label" for="cn">Or supplier name</label><input id="cn" class="form-input" bind:value={form.counterparty_name} /></div>
				{/if}
				<div class="form-group"><label class="form-label" for="i">Incoterm</label>
					<select id="i" class="form-input" bind:value={form.incoterm}>
						<option value="">-</option>{#each ['EXW', 'FCA', 'FAS', 'FOB', 'CFR', 'CIF', 'CPT', 'CIP', 'DAP', 'DPU', 'DDP'] as t}<option>{t}</option>{/each}
					</select></div>
				<div class="form-group"><label class="form-label" for="m">Mode</label>
					<select id="m" class="form-input" bind:value={form.mode}>{#each ['SEA', 'AIR', 'ROAD', 'RAIL', 'COURIER'] as t}<option>{t}</option>{/each}</select></div>
				<div class="form-group"><label class="form-label" for="pl">Port of loading</label><input id="pl" class="form-input" bind:value={form.port_loading} /></div>
				<div class="form-group"><label class="form-label" for="pd">Port of discharge</label><input id="pd" class="form-input" bind:value={form.port_discharge} /></div>
			</div>
			<button class="btn btn-primary" disabled={busy}>{busy ? 'Creating...' : 'Create shipment file'}</button>
		</form>
	{/if}

	<div style="display: flex; gap: 0.5rem; margin-bottom: 0.75rem;">
		{#each [['', 'All'], ['EXPORT', 'Exports'], ['IMPORT', 'Imports']] as [v, label]}
			<button class="btn {filter === v ? 'btn-primary' : 'btn-secondary'}" onclick={() => { filter = v; load(); }}>{label}</button>
		{/each}
	</div>
	<div class="card" style="padding: 1rem; overflow-x: auto;">
		<table style="width: 100%; font-size: 0.875rem;">
			<thead><tr style="text-align: left; color: var(--text-secondary);"><th>Number</th><th>Type</th><th>Party</th><th>Route</th><th>Status</th><th>Documents</th></tr></thead>
			<tbody>
				{#each list as s}
					<tr>
						<td><a href="/trade/shipments/{s.id}">{s.shipment_number}</a></td>
						<td>{s.direction}</td>
						<td>{s.counterparty_name}</td>
						<td>{s.port_loading || '?'} → {s.port_discharge || '?'} ({s.mode})</td>
						<td style="color: {color[s.status]}; font-weight: 600;">{s.status.replace('_', ' ')}</td>
						<td>{s.docs_done} / {s.docs_total}</td>
					</tr>
				{:else}
					<tr><td colspan="6" style="color: var(--text-secondary);">No shipments yet.</td></tr>
				{/each}
			</tbody>
		</table>
	</div>
</div>
