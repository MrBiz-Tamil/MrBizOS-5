<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';
	import { toast } from '$lib/stores/toast.svelte';

	interface StockItem {
		product_id: string;
		product_name: string;
		sku: string;
		category: string;
		current_stock: string | number;
		min_stock: string | number;
		stock_value: string | number;
		is_low_stock: boolean;
	}

	let stockItems = $state<StockItem[]>([]);
	let loading = $state(true);
	let error = $state('');

	// Search & Filters
	let searchQuery = $state('');
	let categoryFilter = $state('ALL');
	let statusFilter = $state<'ALL' | 'LOW' | 'OPTIMAL' | 'OUT_OF_STOCK'>('ALL');

	// Slide-Over Adjustment Drawer State
	let showAdjustDrawer = $state(false);
	let selectedItem = $state<StockItem | null>(null);
	let adjustMode = $state<'SET_EXACT' | 'ADD' | 'REDUCE'>('SET_EXACT');
	let adjustmentQuantity = $state('');
	let adjustmentReason = $state('');
	let adjustSubmitting = $state(false);
	let adjustError = $state('');

	onMount(async () => {
		await loadStock();
	});

	async function loadStock() {
		loading = true;
		error = '';
		try {
			stockItems = await api.listInventory();
		} catch (err: any) {
			error = err.message || 'Failed to load inventory stock';
			toast.error('Inventory Load Failed', error);
		} finally {
			loading = false;
		}
	}

	function formatCurrency(val: string | number | undefined): string {
		if (val === undefined || val === null) return '₹0.00';
		const num = typeof val === 'string' ? parseFloat(val) : val;
		return new Intl.NumberFormat('en-IN', { style: 'currency', currency: 'INR' }).format(isNaN(num) ? 0 : num);
	}

	// Dynamic Categories
	let categories = $derived.by(() => {
		const set = new Set<string>();
		stockItems.forEach((it) => {
			if (it.category) set.add(it.category);
		});
		return Array.from(set);
	});

	// Filtered Items
	let filteredItems = $derived(
		stockItems.filter((item) => {
			const q = searchQuery.toLowerCase().trim();
			const matchesQuery =
				!q ||
				item.product_name.toLowerCase().includes(q) ||
				item.sku.toLowerCase().includes(q) ||
				(item.category && item.category.toLowerCase().includes(q));

			const matchesCategory = categoryFilter === 'ALL' || item.category === categoryFilter;

			const current = parseFloat(String(item.current_stock)) || 0;
			const isOutOfStock = current <= 0;
			const isLow = item.is_low_stock || (current > 0 && current <= (parseFloat(String(item.min_stock)) || 0));

			let matchesStatus = true;
			if (statusFilter === 'OUT_OF_STOCK') matchesStatus = isOutOfStock;
			else if (statusFilter === 'LOW') matchesStatus = isLow && !isOutOfStock;
			else if (statusFilter === 'OPTIMAL') matchesStatus = !isLow && !isOutOfStock;

			return matchesQuery && matchesCategory && matchesStatus;
		})
	);

	// Cockpit KPIs
	let totalValuation = $derived(
		stockItems.reduce((acc, it) => acc + (parseFloat(String(it.stock_value)) || 0), 0)
	);

	let totalUnitsOnHand = $derived(
		stockItems.reduce((acc, it) => acc + (parseFloat(String(it.current_stock)) || 0), 0)
	);

	let lowStockCount = $derived(
		stockItems.filter((it) => {
			const cur = parseFloat(String(it.current_stock)) || 0;
			const min = parseFloat(String(it.min_stock)) || 0;
			return it.is_low_stock || (cur > 0 && cur <= min);
		}).length
	);

	let outOfStockCount = $derived(
		stockItems.filter((it) => (parseFloat(String(it.current_stock)) || 0) <= 0).length
	);

	function openAdjustDrawer(item: StockItem) {
		selectedItem = item;
		adjustMode = 'SET_EXACT';
		adjustmentQuantity = String(parseFloat(String(item.current_stock)) || 0);
		adjustmentReason = '';
		adjustError = '';
		showAdjustDrawer = true;
	}

	let targetFinalStock = $derived.by(() => {
		if (!selectedItem) return 0;
		const cur = parseFloat(String(selectedItem.current_stock)) || 0;
		const delta = parseFloat(adjustmentQuantity) || 0;
		if (adjustMode === 'SET_EXACT') return delta;
		if (adjustMode === 'ADD') return cur + delta;
		if (adjustMode === 'REDUCE') return Math.max(0, cur - delta);
		return cur;
	});

	async function handleAdjustSubmit(e: Event) {
		e.preventDefault();
		if (!selectedItem) return;

		const finalQty = targetFinalStock;
		if (isNaN(finalQty) || finalQty < 0) {
			adjustError = 'Calculated stock cannot be negative';
			return;
		}

		if (!adjustmentReason.trim()) {
			adjustError = 'Please provide an adjustment reason or audit note';
			return;
		}

		adjustSubmitting = true;
		adjustError = '';
		try {
			await api.adjustInventory(selectedItem.product_id, String(finalQty), adjustmentReason.trim());
			toast.success(
				'Stock Adjusted!',
				`${selectedItem.product_name} updated to ${finalQty} units (${adjustmentReason})`
			);
			showAdjustDrawer = false;
			await loadStock();
		} catch (err: any) {
			adjustError = err.message || 'Failed to adjust inventory stock';
			toast.error('Adjustment Failed', adjustError);
		} finally {
			adjustSubmitting = false;
		}
	}

	function handleDrawerKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape' && showAdjustDrawer) {
			showAdjustDrawer = false;
		}
	}

	function exportInventoryCSV() {
		if (filteredItems.length === 0) {
			toast.warning('No Records', 'No inventory records to export');
			return;
		}
		const headers = ['Product Name', 'SKU', 'Category', 'Current Stock', 'Min Threshold', 'Stock Valuation', 'Status'];
		const rows = filteredItems.map((it) => {
			const cur = parseFloat(String(it.current_stock)) || 0;
			const min = parseFloat(String(it.min_stock)) || 0;
			const status = cur <= 0 ? 'OUT_OF_STOCK' : it.is_low_stock || cur <= min ? 'LOW_STOCK' : 'OPTIMAL';
			return [
				`"${it.product_name.replace(/"/g, '""')}"`,
				it.sku,
				`"${it.category || 'General'}"`,
				it.current_stock,
				it.min_stock,
				it.stock_value,
				status
			];
		});
		const csvContent = 'data:text/csv;charset=utf-8,' + [headers.join(','), ...rows.map((r) => r.join(','))].join('\n');
		const encodedUri = encodeURI(csvContent);
		const link = document.createElement('a');
		link.setAttribute('href', encodedUri);
		link.setAttribute('download', `inventory_valuation_${new Date().toISOString().slice(0, 10)}.csv`);
		document.body.appendChild(link);
		link.click();
		document.body.removeChild(link);
		toast.success('Inventory Exported', 'Inventory valuation CSV report downloaded successfully');
	}
</script>

<svelte:window onkeydown={handleDrawerKeydown} />

<svelte:head>
	<title>Inventory Stock & Valuation Cockpit | MrBiz-OS</title>
</svelte:head>

<div class="inventory-page-container animate-fade-in">
	<!-- Header -->
	<div class="inventory-header">
		<div>
			<div class="header-badge">
				<span>📋</span> WAREHOUSE & ASSET MANAGEMENT
			</div>
			<h1 class="page-title">Inventory Stock & Valuation</h1>
			<p class="page-subtitle">
				Audited stock levels, low-inventory alarms, real-time asset valuation, and physical count reconciliation.
			</p>
		</div>

		<div class="header-actions">
			<button type="button" class="btn btn-secondary" onclick={exportInventoryCSV} title="Export CSV Valuation">
				📥 Export CSV
			</button>
			<button type="button" class="btn btn-secondary" onclick={loadStock} title="Refresh inventory">
				🔄 Refresh
			</button>
			<a href="/products" class="btn btn-primary" style="box-shadow: 0 4px 15px rgba(99, 102, 241, 0.4);">
				+ Manage Products & Batches
			</a>
		</div>
	</div>

	<!-- Executive KPI Cockpit -->
	<div class="kpi-grid">
		<div class="kpi-card">
			<div class="kpi-top">
				<span class="kpi-label">Stock Valuation Asset</span>
				<span class="kpi-icon emerald">₹</span>
			</div>
			<div class="kpi-value text-emerald">{formatCurrency(totalValuation)}</div>
			<div class="kpi-footer">Total inventory holding value at cost</div>
		</div>

		<div class="kpi-card">
			<div class="kpi-top">
				<span class="kpi-label">Total Units on Hand</span>
				<span class="kpi-icon indigo">📦</span>
			</div>
			<div class="kpi-value text-indigo">{totalUnitsOnHand.toLocaleString('en-IN', { maximumFractionDigits: 1 })}</div>
			<div class="kpi-footer">Across {stockItems.length} registered product SKUs</div>
		</div>

		<div class="kpi-card">
			<div class="kpi-top">
				<span class="kpi-label">Low Stock Alarms</span>
				<span class="kpi-icon amber">⚠️</span>
			</div>
			<div class="kpi-value text-amber">{lowStockCount} SKUs</div>
			<div class="kpi-footer">Breached re-order threshold limit</div>
		</div>

		<div class="kpi-card">
			<div class="kpi-top">
				<span class="kpi-label">Out of Stock</span>
				<span class="kpi-icon rose">🚫</span>
			</div>
			<div class="kpi-value text-rose">{outOfStockCount} SKUs</div>
			<div class="kpi-footer">Zero inventory count • Requires PO</div>
		</div>
	</div>

	<!-- Controls & Search Card -->
	<div class="card controls-card">
		<div class="search-box">
			<span class="search-icon">🔍</span>
			<input
				type="text"
				class="form-input search-input"
				placeholder="Search by product name, SKU code, category..."
				bind:value={searchQuery}
			/>
			{#if searchQuery}
				<button class="clear-search-btn" onclick={() => (searchQuery = '')}>✕</button>
			{/if}
		</div>

		<div class="filters-row">
			<!-- Stock Status Pills -->
			<div class="filter-pills">
				<button
					type="button"
					class="pill-btn"
					class:active={statusFilter === 'ALL'}
					onclick={() => (statusFilter = 'ALL')}
				>
					All Stock ({stockItems.length})
				</button>
				<button
					type="button"
					class="pill-btn"
					class:active={statusFilter === 'LOW'}
					onclick={() => (statusFilter = 'LOW')}
				>
					⚠️ Low Stock ({lowStockCount})
				</button>
				<button
					type="button"
					class="pill-btn"
					class:active={statusFilter === 'OPTIMAL'}
					onclick={() => (statusFilter = 'OPTIMAL')}
				>
					✓ Optimal ({stockItems.length - lowStockCount - outOfStockCount})
				</button>
				<button
					type="button"
					class="pill-btn"
					class:active={statusFilter === 'OUT_OF_STOCK'}
					onclick={() => (statusFilter = 'OUT_OF_STOCK')}
				>
					🚫 Out of Stock ({outOfStockCount})
				</button>
			</div>

			<!-- Category Pills -->
			{#if categories.length > 0}
				<div class="filter-pills category-pills">
					<button
						type="button"
						class="pill-btn"
						class:active={categoryFilter === 'ALL'}
						onclick={() => (categoryFilter = 'ALL')}
					>
						All Categories
					</button>
					{#each categories as cat}
						<button
							type="button"
							class="pill-btn"
							class:active={categoryFilter === cat}
							onclick={() => (categoryFilter = cat)}
						>
							{cat}
						</button>
					{/each}
				</div>
			{/if}
		</div>
	</div>

	<!-- Main Inventory Table -->
	<div class="card table-card">
		<table class="inventory-table">
			<thead>
				<tr>
					<th>Product SKU & Details</th>
					<th>Category</th>
					<th class="text-right">Current Stock</th>
					<th>Stock Gauge vs Min</th>
					<th class="text-right">Min Threshold</th>
					<th class="text-right">Stock Valuation</th>
					<th>Health Status</th>
					<th class="text-right">Actions</th>
				</tr>
			</thead>
			<tbody>
				{#if loading}
					<tr>
						<td colspan="8" class="state-cell">
							<div class="loading-state">
								<div class="spinner"></div>
								<span>Loading real-time inventory levels...</span>
							</div>
						</td>
					</tr>
				{:else if error}
					<tr>
						<td colspan="8" class="state-cell">
							<div class="error-state">
								<span>⚠️</span>
								<p>{error}</p>
								<button class="btn btn-secondary btn-sm" onclick={loadStock}>Retry</button>
							</div>
						</td>
					</tr>
				{:else if filteredItems.length === 0}
					<tr>
						<td colspan="8" class="state-cell">
							<div class="empty-state">
								<div class="empty-icon">📦</div>
								<h3>No inventory items found</h3>
								<p>
									{#if searchQuery || categoryFilter !== 'ALL' || statusFilter !== 'ALL'}
										No items match your filter criteria. Try clearing search or category filter.
									{:else}
										Create your first product in the products catalog to start tracking stock.
									{/if}
								</p>
								<a href="/products" class="btn btn-primary btn-sm" style="margin-top: 0.5rem;">
									Go to Products Catalog
								</a>
							</div>
						</td>
					</tr>
				{:else}
					{#each filteredItems as item (item.product_id)}
						{@const cur = parseFloat(String(item.current_stock)) || 0}
						{@const min = parseFloat(String(item.min_stock)) || 1}
						{@const ratio = Math.min(100, Math.round((cur / (min * 2)) * 100))}
						{@const isZero = cur <= 0}
						{@const isLow = item.is_low_stock || (cur > 0 && cur <= min)}
						<tr class="item-row" onclick={() => openAdjustDrawer(item)}>
							<!-- Product SKU -->
							<td>
								<div class="prod-cell">
									<div class="prod-avatar">📦</div>
									<div class="prod-info">
										<span class="prod-name">{item.product_name}</span>
										<span class="prod-sku font-mono">{item.sku}</span>
									</div>
								</div>
							</td>

							<!-- Category -->
							<td>
								<span class="category-tag">{item.category || 'General'}</span>
							</td>

							<!-- Current Stock -->
							<td class="text-right">
								<span class="cur-stock font-mono" class:text-rose={isZero} class:text-amber={isLow && !isZero}>
									{cur.toFixed(2)}
								</span>
							</td>

							<!-- Stock Bar Gauge -->
							<td>
								<div class="gauge-container" title={`${cur} in stock vs ${min} minimum`}>
									<div
										class="gauge-bar"
										class:bar-red={isZero}
										class:bar-amber={isLow && !isZero}
										class:bar-green={!isLow && !isZero}
										style={`width: ${Math.max(5, ratio)}%`}
									></div>
								</div>
							</td>

							<!-- Min Threshold -->
							<td class="text-right">
								<span class="min-stock font-mono">{min.toFixed(2)}</span>
							</td>

							<!-- Valuation -->
							<td class="text-right">
								<span class="val-num font-mono text-emerald">{formatCurrency(item.stock_value)}</span>
							</td>

							<!-- Status -->
							<td>
								{#if isZero}
									<span class="health-pill out-of-stock">
										<span class="dot"></span> Out of Stock
									</span>
								{:else if isLow}
									<span class="health-pill low-stock">
										<span class="dot"></span> Low Stock
									</span>
								{:else}
									<span class="health-pill optimal">
										<span class="dot"></span> Optimal
									</span>
								{/if}
							</td>

							<!-- Actions -->
							<td class="text-right" onclick={(e) => e.stopPropagation()}>
								<button
									type="button"
									class="btn btn-secondary btn-sm"
									onclick={() => openAdjustDrawer(item)}
									title="Audit or Adjust Stock Level"
								>
									⚡ Adjust Stock
								</button>
							</td>
						</tr>
					{/each}
				{/if}
			</tbody>
		</table>
	</div>
</div>

<!-- ============================================== -->
<!-- SLIDE-OVER STOCK ADJUSTMENT DRAWER             -->
<!-- ============================================== -->
{#if showAdjustDrawer && selectedItem}
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div class="drawer-backdrop animate-fade-in" onclick={() => (showAdjustDrawer = false)}>
		<div class="drawer-panel animate-slide-left" onclick={(e) => e.stopPropagation()} role="dialog" aria-modal="true">
			<!-- Drawer Header -->
			<div class="drawer-header">
				<div style="display: flex; align-items: center; gap: 0.6rem;">
					<span style="font-size: 1.4rem;">⚖️</span>
					<div>
						<div class="drawer-title">Adjust Stock Level</div>
						<div class="drawer-subtitle font-mono">{selectedItem.sku} • {selectedItem.product_name}</div>
					</div>
				</div>
				<button type="button" class="drawer-close-btn" onclick={() => (showAdjustDrawer = false)}>✕</button>
			</div>

			<!-- Drawer Body Form -->
			<form onsubmit={handleAdjustSubmit} class="drawer-body">
				<!-- Current Stock Card -->
				<div class="current-stock-card">
					<div class="meta-col">
						<span class="meta-label">Current Stock on Hand</span>
						<span class="meta-value font-mono">{parseFloat(String(selectedItem.current_stock)).toFixed(2)} Units</span>
					</div>
					<div class="meta-col">
						<span class="meta-label">Min Threshold</span>
						<span class="meta-value font-mono">{parseFloat(String(selectedItem.min_stock)).toFixed(2)} Units</span>
					</div>
				</div>

				<!-- Adjustment Mode Selector -->
				<div class="form-group">
					<!-- svelte-ignore a11y_label_has_associated_control -->
					<label class="input-label">Adjustment Mode</label>
					<div class="mode-selector-grid">
						<button
							type="button"
							class="mode-btn"
							class:active={adjustMode === 'SET_EXACT'}
							onclick={() => { adjustMode = 'SET_EXACT'; adjustmentQuantity = String(selectedItem!.current_stock); }}
						>
							<span style="font-size: 1.1rem;">🎯</span>
							<strong>Set Exact Count</strong>
							<small>Physical tally</small>
						</button>
						<button
							type="button"
							class="mode-btn"
							class:active={adjustMode === 'ADD'}
							onclick={() => { adjustMode = 'ADD'; adjustmentQuantity = '10'; }}
						>
							<span style="font-size: 1.1rem;">📥</span>
							<strong>Add Incoming</strong>
							<small>Restock / return</small>
						</button>
						<button
							type="button"
							class="mode-btn"
							class:active={adjustMode === 'REDUCE'}
							onclick={() => { adjustMode = 'REDUCE'; adjustmentQuantity = '5'; }}
						>
							<span style="font-size: 1.1rem;">📤</span>
							<strong>Write Off / Reduce</strong>
							<small>Damage / loss</small>
						</button>
					</div>
				</div>

				<!-- Quantity Input -->
				<div class="form-group">
					<label for="adj-qty" class="input-label">
						{#if adjustMode === 'SET_EXACT'}
							Exact Physical Verified Count
						{:else if adjustMode === 'ADD'}
							Units to Add (+)
						{:else}
							Units to Deduct (-)
						{/if}
					</label>
					<input
						id="adj-qty"
						type="number"
						step="any"
						min="0"
						class="form-input font-mono"
						style="font-size: 1.125rem; font-weight: 700;"
						placeholder="Enter quantity..."
						bind:value={adjustmentQuantity}
						required
					/>
				</div>

				<!-- Calculated Result Banner -->
				<div class="calc-preview-card">
					<span class="calc-label">Resulting Stock Level</span>
					<div class="calc-math font-mono">
						<span>{parseFloat(String(selectedItem.current_stock)).toFixed(2)}</span>
						{#if adjustMode === 'ADD'}
							<span style="color: #34d399;">+ {parseFloat(adjustmentQuantity) || 0}</span>
						{:else if adjustMode === 'REDUCE'}
							<span style="color: #fb7185;">- {parseFloat(adjustmentQuantity) || 0}</span>
						{:else}
							<span style="color: #818cf8;">→</span>
						{/if}
						<span style="font-weight: 900; font-size: 1.25rem; color: #ffffff;">= {targetFinalStock.toFixed(2)} Units</span>
					</div>
				</div>

				<!-- Reason Input -->
				<div class="form-group">
					<label for="adj-reason" class="input-label">Audit Adjustment Reason *</label>
					<input
						id="adj-reason"
						type="text"
						class="form-input"
						placeholder="e.g. Month-end physical verification tally, supplier replacement, damaged goods..."
						bind:value={adjustmentReason}
						required
					/>
					<div class="quick-reasons">
						<button
							type="button"
							class="qr-pill"
							onclick={() => (adjustmentReason = 'Physical audit count reconciliation')}
						>
							Audit Count
						</button>
						<button
							type="button"
							class="qr-pill"
							onclick={() => (adjustmentReason = 'Damaged goods write-off')}
						>
							Damaged Write-off
						</button>
						<button
							type="button"
							class="qr-pill"
							onclick={() => (adjustmentReason = 'Customer sales return restocked')}
						>
							Return Restocked
						</button>
						<button
							type="button"
							class="qr-pill"
							onclick={() => (adjustmentReason = 'Supplier warranty replacement')}
						>
							Warranty Replacement
						</button>
					</div>
				</div>

				{#if adjustError}
					<div class="error-banner">⚠️ {adjustError}</div>
				{/if}

				<!-- Footer Buttons inside form -->
				<div class="drawer-footer-actions">
					<button type="button" class="btn btn-secondary" onclick={() => (showAdjustDrawer = false)}>
						Cancel
					</button>
					<button type="submit" class="btn btn-primary" disabled={adjustSubmitting}>
						{#if adjustSubmitting}
							⏳ Recording...
						{:else}
							✓ Commit Stock Adjustment
						{/if}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}

<style>
	.inventory-page-container {
		display: flex;
		flex-direction: column;
		gap: 1.5rem;
		max-width: 1400px;
		margin: 0 auto;
	}

	.inventory-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		flex-wrap: wrap;
		gap: 1.25rem;
	}

	.header-badge {
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
		font-size: 0.7rem;
		font-weight: 800;
		color: #818cf8;
		background: rgba(99, 102, 241, 0.12);
		padding: 0.2rem 0.55rem;
		border-radius: 6px;
		border: 1px solid rgba(99, 102, 241, 0.25);
		margin-bottom: 0.35rem;
		letter-spacing: 0.05em;
	}

	.page-title {
		font-size: 1.75rem;
		font-weight: 800;
		margin: 0;
		color: #ffffff;
	}

	.page-subtitle {
		font-size: 0.875rem;
		color: #94a3b8;
		margin: 0.25rem 0 0 0;
	}

	.header-actions {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		flex-wrap: wrap;
	}

	/* KPI Cockpit */
	.kpi-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
		gap: 1rem;
	}

	.kpi-card {
		background: rgba(17, 24, 39, 0.75);
		backdrop-filter: blur(16px);
		-webkit-backdrop-filter: blur(16px);
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 12px;
		padding: 1.25rem;
		display: flex;
		flex-direction: column;
		gap: 0.4rem;
		transition: all 0.2s ease;
	}

	.kpi-card:hover {
		border-color: rgba(99, 102, 241, 0.3);
		transform: translateY(-2px);
		box-shadow: 0 8px 20px rgba(0, 0, 0, 0.3);
	}

	.kpi-top {
		display: flex;
		justify-content: space-between;
		align-items: center;
	}

	.kpi-label {
		font-size: 0.8125rem;
		font-weight: 600;
		color: #94a3b8;
		text-transform: uppercase;
		letter-spacing: 0.03em;
	}

	.kpi-icon {
		width: 32px;
		height: 32px;
		border-radius: 8px;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 1rem;
		font-weight: 800;
	}
	.kpi-icon.emerald { background: rgba(16, 185, 129, 0.15); color: #34d399; }
	.kpi-icon.indigo { background: rgba(99, 102, 241, 0.15); color: #818cf8; }
	.kpi-icon.amber { background: rgba(245, 158, 11, 0.15); color: #fbbf24; }
	.kpi-icon.rose { background: rgba(244, 63, 94, 0.15); color: #fb7185; }

	.kpi-value {
		font-size: 1.6rem;
		font-weight: 800;
		color: #ffffff;
		font-family: var(--font-heading);
		line-height: 1.1;
	}

	.kpi-footer {
		font-size: 0.75rem;
		color: #64748b;
	}

	/* Controls Card */
	.controls-card {
		padding: 1rem 1.25rem;
		display: flex;
		flex-direction: column;
		gap: 0.875rem;
	}

	.search-box {
		position: relative;
		display: flex;
		align-items: center;
	}

	.search-icon {
		position: absolute;
		left: 0.875rem;
		color: #64748b;
		font-size: 0.9rem;
	}

	.search-input {
		padding-left: 2.5rem;
		padding-right: 2rem;
		height: 42px;
		border-radius: 8px;
		background: rgba(15, 23, 42, 0.8);
		border: 1px solid rgba(255, 255, 255, 0.1);
		color: #f8fafc;
		width: 100%;
	}

	.search-input:focus {
		border-color: #6366f1;
		box-shadow: 0 0 10px rgba(99, 102, 241, 0.25);
	}

	.clear-search-btn {
		position: absolute;
		right: 0.75rem;
		background: none;
		border: none;
		color: #64748b;
		cursor: pointer;
		font-size: 0.8rem;
	}

	.filters-row {
		display: flex;
		justify-content: space-between;
		align-items: center;
		flex-wrap: wrap;
		gap: 0.75rem;
	}

	.filter-pills {
		display: flex;
		align-items: center;
		gap: 0.35rem;
		flex-wrap: wrap;
	}

	.pill-btn {
		background: rgba(255, 255, 255, 0.04);
		border: 1px solid rgba(255, 255, 255, 0.08);
		color: #94a3b8;
		font-size: 0.775rem;
		font-weight: 600;
		padding: 0.35rem 0.75rem;
		border-radius: 6px;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.pill-btn:hover {
		background: rgba(255, 255, 255, 0.08);
		color: #f8fafc;
	}

	.pill-btn.active {
		background: rgba(99, 102, 241, 0.2);
		border-color: #6366f1;
		color: #a5b4fc;
		font-weight: 700;
	}

	/* Table Card */
	.table-card {
		padding: 0;
		overflow: hidden;
		border: 1px solid rgba(255, 255, 255, 0.08);
	}

	.inventory-table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.875rem;
		text-align: left;
	}

	.inventory-table th {
		background: rgba(15, 23, 42, 0.85);
		padding: 0.875rem 1.25rem;
		color: #94a3b8;
		font-size: 0.75rem;
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 0.04em;
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
	}

	.inventory-table td {
		padding: 1rem 1.25rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.04);
		color: #f8fafc;
		vertical-align: middle;
	}

	.item-row {
		cursor: pointer;
		transition: background 0.15s ease;
	}

	.item-row:hover {
		background: rgba(255, 255, 255, 0.03);
	}

	.prod-cell {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}

	.prod-avatar {
		width: 32px;
		height: 32px;
		border-radius: 8px;
		background: rgba(99, 102, 241, 0.15);
		border: 1px solid rgba(99, 102, 241, 0.25);
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 0.95rem;
		flex-shrink: 0;
	}

	.prod-info {
		display: flex;
		flex-direction: column;
	}

	.prod-name {
		font-weight: 600;
		color: #ffffff;
	}

	.prod-sku {
		font-size: 0.725rem;
		color: #818cf8;
	}

	.category-tag {
		font-size: 0.725rem;
		background: rgba(255, 255, 255, 0.05);
		border: 1px solid rgba(255, 255, 255, 0.1);
		padding: 0.2rem 0.5rem;
		border-radius: 4px;
		color: #cbd5e1;
	}

	.cur-stock {
		font-weight: 700;
		font-size: 0.95rem;
	}

	.min-stock {
		color: #94a3b8;
		font-size: 0.8125rem;
	}

	.val-num {
		font-weight: 700;
	}

	/* Gauge Bar */
	.gauge-container {
		width: 100px;
		height: 6px;
		background: rgba(255, 255, 255, 0.08);
		border-radius: 9999px;
		overflow: hidden;
	}

	.gauge-bar {
		height: 100%;
		border-radius: 9999px;
		transition: width 0.3s ease;
	}
	.gauge-bar.bar-green { background: #10b981; }
	.gauge-bar.bar-amber { background: #f59e0b; }
	.gauge-bar.bar-red { background: #f43f5e; }

	/* Health Pill */
	.health-pill {
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
		font-size: 0.725rem;
		font-weight: 700;
		padding: 0.25rem 0.6rem;
		border-radius: 9999px;
	}

	.health-pill .dot {
		width: 6px;
		height: 6px;
		border-radius: 50%;
	}

	.health-pill.optimal { background: rgba(16, 185, 129, 0.15); color: #34d399; border: 1px solid rgba(16, 185, 129, 0.3); }
	.health-pill.optimal .dot { background: #10b981; }

	.health-pill.low-stock { background: rgba(245, 158, 11, 0.15); color: #fbbf24; border: 1px solid rgba(245, 158, 11, 0.3); }
	.health-pill.low-stock .dot { background: #f59e0b; }

	.health-pill.out-of-stock { background: rgba(244, 63, 94, 0.15); color: #fb7185; border: 1px solid rgba(244, 63, 94, 0.3); }
	.health-pill.out-of-stock .dot { background: #f43f5e; }

	/* State Cells */
	.state-cell {
		text-align: center;
		padding: 3.5rem 1rem !important;
	}

	.loading-state, .error-state, .empty-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.5rem;
		color: #94a3b8;
	}

	.spinner {
		width: 32px;
		height: 32px;
		border: 3px solid rgba(99, 102, 241, 0.2);
		border-top-color: #6366f1;
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	.empty-icon { font-size: 2.5rem; opacity: 0.7; }
	.empty-state h3 { color: #ffffff; margin: 0; font-size: 1.125rem; }
	.empty-state p { color: #64748b; font-size: 0.8125rem; max-width: 380px; margin: 0; }

	/* SLIDE-OVER DRAWER */
	.drawer-backdrop {
		position: fixed;
		inset: 0;
		z-index: 10000;
		background: rgba(11, 15, 25, 0.7);
		backdrop-filter: blur(8px);
		display: flex;
		justify-content: flex-end;
	}

	.drawer-panel {
		width: 100%;
		max-width: 520px;
		height: 100vh;
		background: #111827;
		border-left: 1px solid rgba(255, 255, 255, 0.12);
		box-shadow: -10px 0 30px rgba(0, 0, 0, 0.6);
		display: flex;
		flex-direction: column;
		overflow: hidden;
	}

	.drawer-header {
		padding: 1.25rem 1.5rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
		display: flex;
		justify-content: space-between;
		align-items: center;
		background: rgba(15, 23, 42, 0.6);
	}

	.drawer-title {
		font-size: 1.2rem;
		font-weight: 800;
		color: #ffffff;
	}

	.drawer-subtitle {
		font-size: 0.8125rem;
		color: #94a3b8;
	}

	.drawer-close-btn {
		background: rgba(255, 255, 255, 0.06);
		border: 1px solid rgba(255, 255, 255, 0.1);
		color: #94a3b8;
		width: 32px;
		height: 32px;
		border-radius: 8px;
		display: flex;
		align-items: center;
		justify-content: center;
		cursor: pointer;
	}

	.drawer-body {
		flex: 1;
		overflow-y: auto;
		padding: 1.5rem;
		display: flex;
		flex-direction: column;
		gap: 1.25rem;
	}

	.current-stock-card {
		background: rgba(15, 23, 42, 0.7);
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 8px;
		padding: 1rem;
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 1rem;
	}

	.meta-col {
		display: flex;
		flex-direction: column;
		gap: 0.2rem;
	}

	.meta-label {
		font-size: 0.7rem;
		color: #94a3b8;
		text-transform: uppercase;
		letter-spacing: 0.04em;
	}

	.meta-value {
		font-size: 1.15rem;
		font-weight: 800;
		color: #ffffff;
	}

	.mode-selector-grid {
		display: grid;
		grid-template-columns: 1fr 1fr 1fr;
		gap: 0.5rem;
		margin-top: 0.35rem;
	}

	.mode-btn {
		background: rgba(255, 255, 255, 0.04);
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 8px;
		padding: 0.75rem 0.5rem;
		color: #94a3b8;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.2rem;
		cursor: pointer;
		transition: all 0.15s ease;
		text-align: center;
	}

	.mode-btn strong {
		font-size: 0.75rem;
		color: #ffffff;
	}

	.mode-btn small {
		font-size: 0.65rem;
		color: #64748b;
	}

	.mode-btn.active {
		background: rgba(99, 102, 241, 0.2);
		border-color: #6366f1;
		box-shadow: 0 0 12px rgba(99, 102, 241, 0.25);
	}
	.mode-btn.active strong {
		color: #a5b4fc;
	}

	.calc-preview-card {
		background: rgba(99, 102, 241, 0.08);
		border: 1px dashed rgba(99, 102, 241, 0.3);
		border-radius: 8px;
		padding: 1rem;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.25rem;
	}

	.calc-label {
		font-size: 0.725rem;
		color: #a5b4fc;
		text-transform: uppercase;
		letter-spacing: 0.04em;
	}

	.calc-math {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 1rem;
		color: #cbd5e1;
	}

	.quick-reasons {
		display: flex;
		flex-wrap: wrap;
		gap: 0.35rem;
		margin-top: 0.4rem;
	}

	.qr-pill {
		background: rgba(255, 255, 255, 0.04);
		border: 1px solid rgba(255, 255, 255, 0.08);
		color: #94a3b8;
		font-size: 0.6875rem;
		padding: 0.2rem 0.5rem;
		border-radius: 4px;
		cursor: pointer;
		transition: all 0.15s ease;
	}
	.qr-pill:hover {
		background: rgba(255, 255, 255, 0.08);
		color: #f8fafc;
	}

	.error-banner {
		background: rgba(244, 63, 94, 0.15);
		border: 1px solid rgba(244, 63, 94, 0.3);
		color: #fb7185;
		padding: 0.75rem;
		border-radius: 6px;
		font-size: 0.8125rem;
	}

	.drawer-footer-actions {
		display: flex;
		justify-content: flex-end;
		gap: 0.75rem;
		margin-top: auto;
		padding-top: 1rem;
		border-top: 1px solid rgba(255, 255, 255, 0.08);
	}

	/* ANIMATIONS */
	@keyframes slideLeft {
		from { transform: translateX(100%); }
		to { transform: translateX(0); }
	}

	.animate-slide-left {
		animation: slideLeft 0.25s cubic-bezier(0.16, 1, 0.3, 1) forwards;
	}

	@media (max-width: 768px) {
		.inventory-header { flex-direction: column; align-items: flex-start; }
		.header-actions { width: 100%; justify-content: space-between; }
		.kpi-grid { grid-template-columns: 1fr 1fr; }
		.drawer-panel { max-width: 100%; }
	}
</style>
