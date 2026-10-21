<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';

	let customers = $state<any[]>([]);
	let units = $state<any[]>([]);
	let customerId = $state('');
	let kind = $state('QUOTATION');
	let validUntil = $state('');
	let terms = $state('');
	let notes = $state('');
	let lines = $state([{ item_description: '', hsn_sac: '', quantity: '1', unit_code: 'PCS', unit_price: '', discount_rate: '0', tax_rate: '18' }]);
	let error = $state('');
	let busy = $state(false);

	onMount(async () => {
		[customers, units] = await Promise.all([api.listCustomers(), api.listUnits()]);
		if (units.length && !units.find((u: any) => u.code === 'PCS')) lines[0].unit_code = units[0].code;
	});

	function addLine() {
		lines.push({ item_description: '', hsn_sac: '', quantity: '1', unit_code: lines[0].unit_code, unit_price: '', discount_rate: '0', tax_rate: '18' });
	}

	async function submit(e: Event) {
		e.preventDefault();
		error = '';
		busy = true;
		try {
			const q = await api.createQuotation({
				kind,
				customer_id: customerId,
				valid_until: validUntil || undefined,
				terms,
				notes,
				items: lines
			});
			window.location.href = `/quotations/${q.id}`;
		} catch (err: any) {
			error = err.message || 'Could not create the quotation';
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head><title>New Quotation</title></svelte:head>

<div style="max-width: 980px;">
	<h1 style="margin-bottom: 1rem;">New quotation</h1>
	{#if error}<div class="card" style="padding: 0.6rem 1rem; margin-bottom: 1rem; color: #fb7185;">{error}</div>{/if}
	<form onsubmit={submit}>
		<div class="card" style="padding: 1.25rem; margin-bottom: 1rem; display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 1rem;">
			<div class="form-group"><label class="form-label" for="k">Document</label>
				<select id="k" class="form-input" bind:value={kind}><option value="QUOTATION">Quotation</option><option value="PROFORMA">Proforma invoice</option></select></div>
			<div class="form-group"><label class="form-label" for="c">Customer</label>
				<select id="c" class="form-input" bind:value={customerId} required>
					<option value="" disabled>Select customer</option>
					{#each customers as c}<option value={c.id}>{c.name}</option>{/each}
				</select></div>
			<div class="form-group"><label class="form-label" for="v">Valid until (default 15 days)</label>
				<input id="v" type="date" class="form-input" bind:value={validUntil} /></div>
		</div>

		<div class="card" style="padding: 1.25rem; margin-bottom: 1rem; overflow-x: auto;">
			<table style="width: 100%; font-size: 0.8125rem;">
				<thead><tr style="text-align: left; color: var(--text-secondary);"><th>Description</th><th>HSN/SAC</th><th>Qty</th><th>Unit</th><th>Price</th><th>Disc %</th><th>GST %</th><th></th></tr></thead>
				<tbody>
					{#each lines as l, i}
						<tr>
							<td><input class="form-input" bind:value={l.item_description} required placeholder="Item or service" /></td>
							<td><input class="form-input" style="width: 90px;" bind:value={l.hsn_sac} /></td>
							<td><input class="form-input" style="width: 70px;" bind:value={l.quantity} required /></td>
							<td><select class="form-input" bind:value={l.unit_code}>{#each units as u}<option value={u.code}>{u.code}</option>{/each}</select></td>
							<td><input class="form-input" style="width: 100px;" bind:value={l.unit_price} required /></td>
							<td><input class="form-input" style="width: 60px;" bind:value={l.discount_rate} /></td>
							<td><input class="form-input" style="width: 60px;" bind:value={l.tax_rate} /></td>
							<td>{#if lines.length > 1}<button type="button" class="btn btn-secondary" onclick={() => lines.splice(i, 1)}>✕</button>{/if}</td>
						</tr>
					{/each}
				</tbody>
			</table>
			<button type="button" class="btn btn-secondary" style="margin-top: 0.75rem;" onclick={addLine}>+ Add line</button>
		</div>

		<div class="card" style="padding: 1.25rem; margin-bottom: 1rem;">
			<div class="form-group"><label class="form-label" for="t">Terms</label><textarea id="t" class="form-input" rows="2" bind:value={terms}></textarea></div>
			<div class="form-group"><label class="form-label" for="n">Notes</label><textarea id="n" class="form-input" rows="2" bind:value={notes}></textarea></div>
		</div>
		<button class="btn btn-primary" disabled={busy}>{busy ? 'Saving...' : 'Create'}</button>
	</form>
</div>
