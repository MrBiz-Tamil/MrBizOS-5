<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';

	let stockItems = $state<any[]>([]);
	let loading = $state(true);
	let error = $state('');

	let showAdjustModal = $state(false);
	let selectedItem = $state<any | null>(null);
	let newStockValue = $state('');
	let adjustmentReason = $state('');
	let adjustSubmitting = $state(false);
	let adjustError = $state('');

	onMount(async () => {
		await loadStock();
	});

	async function loadStock() {
		loading = true;
		try {
			stockItems = await api.listInventory();
		} catch (err: any) {
			error = err.message || 'Failed to load stock';
		} finally {
			loading = false;
		}
	}

	function openAdjustModal(item: any) {
		selectedItem = item;
		newStockValue = item.current_stock;
		adjustmentReason = '';
		adjustError = '';
		showAdjustModal = true;
	}

	async function handleAdjust(e: Event) {
		e.preventDefault();
		if (!selectedItem || !newStockValue || !adjustmentReason) {
			adjustError = 'New stock level and adjustment reason are required';
			return;
		}

		adjustSubmitting = true;
		try {
			await api.adjustInventory(selectedItem.product_id, newStockValue, adjustmentReason);
			showAdjustModal = false;
			await loadStock();
		} catch (err: any) {
			adjustError = err.message || 'Failed to adjust stock';
		} finally {
			adjustSubmitting = false;
		}
	}

	function formatCurrency(val: string | number): string {
		const num = typeof val === 'string' ? parseFloat(val) : val;
		return new Intl.NumberFormat('en-IN', { style: 'currency', currency: 'INR' }).format(num);
	}
</script>

<svelte:head>
	<title>Inventory Stock & Valuation - SaaS Billing</title>
</svelte:head>

<div style="display: flex; flex-direction: column; gap: 1.5rem;">
	<div style="display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 1rem;">
		<div>
			<h1 style="font-size: 1.75rem; margin-bottom: 0.25rem;">Inventory Stock & Valuation</h1>
			<p style="font-size: 0.875rem; color: var(--text-secondary);">Real-time inventory levels, low-stock threshold monitoring, and audited adjustments.</p>
		</div>
	</div>

	<div class="table-container card" style="padding: 0;">
		<table class="data-table">
			<thead>
				<tr>
					<th>Product Name</th>
					<th>SKU</th>
					<th>Category</th>
					<th class="text-right">Current Stock</th>
					<th class="text-right">Min Stock Threshold</th>
					<th class="text-right">Stock Valuation</th>
					<th>Status</th>
					<th class="text-right">Actions</th>
				</tr>
			</thead>
			<tbody>
				{#if loading}
					<tr>
						<td colspan="8" class="text-center" style="padding: 3rem; color: var(--text-muted);">
							Loading inventory stock...
						</td>
					</tr>
				{:else if error}
					<tr>
						<td colspan="8" class="text-center" style="padding: 2rem; color: var(--rose);">
							⚠️ {error}
						</td>
					</tr>
				{:else if stockItems.length === 0}
					<tr>
						<td colspan="8" class="text-center" style="padding: 3rem; color: var(--text-muted);">
							No products in inventory.
						</td>
					</tr>
				{:else}
					{#each stockItems as item}
						<tr>
							<td style="font-weight: 600;">{item.product_name}</td>
							<td style="font-family: var(--font-mono); color: #818cf8;">{item.sku}</td>
							<td>{item.category}</td>
							<td class="text-right font-mono" style="font-weight: 600;">{parseFloat(item.current_stock).toFixed(2)}</td>
							<td class="text-right font-mono" style="color: var(--text-secondary);">{parseFloat(item.min_stock).toFixed(2)}</td>
							<td class="text-right font-mono text-emerald" style="font-weight: 600;">{formatCurrency(item.stock_value)}</td>
							<td>
								{#if item.is_low_stock}
									<span class="badge" style="background: rgba(244, 63, 94, 0.15); color: #fb7185; border: 1px solid rgba(244, 63, 94, 0.3);">
										⚠️ Low Stock
									</span>
								{:else}
									<span class="badge" style="background: rgba(16, 185, 129, 0.15); color: #34d399; border: 1px solid rgba(16, 185, 129, 0.3);">
										Optimal
									</span>
								{/if}
							</td>
							<td class="text-right">
								<button onclick={() => openAdjustModal(item)} class="btn btn-secondary btn-sm">
									Adjust Stock
								</button>
							</td>
						</tr>
					{/each}
				{/if}
			</tbody>
		</table>
	</div>
</div>

<!-- Adjust Stock Modal -->
{#if showAdjustModal && selectedItem}
	<div class="modal-backdrop">
		<div class="modal-card animate-fade-in">
			<div class="modal-header">
				<h3 style="font-size: 1.125rem;">Adjust Stock: {selectedItem.product_name}</h3>
				<button onclick={() => (showAdjustModal = false)} style="background: none; border: none; color: var(--text-muted); cursor: pointer; font-size: 1.25rem;">✕</button>
			</div>
			<form onsubmit={handleAdjust}>
				<div class="modal-body">
					{#if adjustError}
						<div style="background: rgba(244, 63, 94, 0.15); border: 1px solid rgba(244, 63, 94, 0.3); border-radius: var(--radius-md); padding: 0.75rem; color: #fb7185; font-size: 0.8125rem; margin-bottom: 1rem;">
							⚠️ {adjustError}
						</div>
					{/if}

					<div style="background: rgba(255,255,255,0.04); padding: 0.75rem 1rem; border-radius: var(--radius-md); margin-bottom: 1.25rem; display: flex; justify-content: space-between;">
						<span style="font-size: 0.8125rem; color: var(--text-secondary);">Current Recorded Stock:</span>
						<span class="font-mono" style="font-weight: 700;">{parseFloat(selectedItem.current_stock).toFixed(2)}</span>
					</div>

					<div class="form-group">
						<label class="form-label" for="adj-stock">New Actual Stock Quantity *</label>
						<input id="adj-stock" type="number" step="any" class="form-input font-mono" bind:value={newStockValue} required />
					</div>

					<div class="form-group" style="margin-bottom: 0;">
						<label class="form-label" for="adj-reason">Audit Reason for Adjustment *</label>
						<input id="adj-reason" type="text" class="form-input" bind:value={adjustmentReason} placeholder="e.g. Physical inventory count discrepancy correction" required />
						<span style="font-size: 0.7rem; color: var(--text-muted);">This transaction will be immutably recorded in the security audit logs.</span>
					</div>
				</div>
				<div class="modal-footer">
					<button type="button" onclick={() => (showAdjustModal = false)} class="btn btn-secondary">Cancel</button>
					<button type="submit" class="btn btn-primary" disabled={adjustSubmitting}>
						{adjustSubmitting ? 'Recording Adjustment...' : 'Confirm Adjustment'}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}
