<script lang="ts">
	import { onMount } from 'svelte';
	import { procurementGrpc } from '$lib/grpc';

	let purchaseOrders = $state<any[]>([]);
	let grns = $state<any[]>([]);
	let lowStockAlerts = $state<any[]>([]);
	let matchResult = $state<any>(null);
	let loading = $state(true);
	let error = $state('');
	let matchTesting = $state(false);

	// Match test inputs
	let billedQuantity = $state(50);
	let billedAmount = $state('60000.00');

	onMount(async () => {
		try {
			const [poRes, grnRes, stockRes]: [any, any, any] = await Promise.all([
				procurementGrpc.listPurchaseOrders(),
				procurementGrpc.listGoodsReceived({ poId: 'po-1' }),
				procurementGrpc.checkLowStockAlerts()
			]);
			purchaseOrders = poRes?.purchaseOrders || [];
			grns = grnRes?.grns || [];
			lowStockAlerts = stockRes?.alerts || [];
		} catch (err: any) {
			error = err.message || 'Failed to connect to Procurement gRPC Service';
		} finally {
			loading = false;
		}
	});

	async function runThreeWayMatch() {
		matchTesting = true;
		try {
			const res: any = await procurementGrpc.verifyThreeWayMatch({
				poId: 'po-1',
				grnId: 'grn-1',
				billId: 'BILL-2026-0012',
				billedQuantity: Number(billedQuantity),
				billedAmount: String(billedAmount)
			});
			matchResult = res?.result || null;
		} catch (err: any) {
			alert('Match verification failed: ' + err.message);
		} finally {
			matchTesting = false;
		}
	}
</script>

<div class="procurement-dashboard">
	<div class="header">
		<div>
			<h1 class="title">Procurement & 3-Way Matching Hub</h1>
			<p class="subtitle">Purchase Orders (PO), Goods Received Notes (GRN), Automated 3-Way Reconciliation & Low Stock Reordering.</p>
		</div>
	</div>

	{#if error}
		<div class="error-banner">⚠️ {error}</div>
	{/if}

	{#if loading}
		<div class="loading-state">Loading procurement workflows...</div>
	{:else}
		<!-- Low Stock Reorder Banner -->
		{#if lowStockAlerts.length > 0}
			<div class="alert-box">
				<div class="alert-header">
					<span>🚨 LOW STOCK SAFETY THRESHOLD BREACH DETECTED ({lowStockAlerts.length} items)</span>
				</div>
				<div class="alert-grid">
					{#each lowStockAlerts as alert}
						<div class="alert-card">
							<div class="alert-title">{alert.productName} ({alert.sku})</div>
							<div class="alert-meta">
								Current Stock: <strong>{alert.currentStock}</strong> | Safety Min: {alert.safetyThreshold}
							</div>
							<div class="alert-reorder">
								Recommended Order: <strong>{alert.reorderQuantity} units</strong> from {alert.preferredSupplier} (Est: ₹{alert.estimatedCost})
							</div>
						</div>
					{/each}
				</div>
			</div>
		{/if}

		<div class="grid-2">
			<!-- Purchase Orders Lifecycle -->
			<div class="card">
				<h2 class="section-title">📦 Purchase Orders (PO)</h2>
				<div class="list">
					{#each purchaseOrders as po}
						<div class="item-row">
							<div>
								<div class="item-title">{po.poNumber} · {po.supplierName}</div>
								<div class="item-subtitle">Items: {po.items?.[0]?.itemDescription} ({po.items?.[0]?.quantity} units)</div>
							</div>
							<div class="row-right">
								<div class="item-price">₹{po.totalAmount}</div>
								<span class="badge status-{po.status.toLowerCase()}">{po.status}</span>
							</div>
						</div>
					{/each}
				</div>
			</div>

			<!-- 3-Way Matching Engine Test Bench -->
			<div class="card">
				<h2 class="section-title">⚖️ 3-Way Match Verification (PO vs GRN vs Vendor Bill)</h2>
				<p class="info-text">Audit incoming vendor bills before authorizing payments. Blocks overbilling and missing stock automatically.</p>
				
				<div class="match-form">
					<div class="form-row">
						<label for="billed-qty">Vendor Billed Quantity:</label>
						<input id="billed-qty" type="number" bind:value={billedQuantity} />
					</div>
					<div class="form-row">
						<label for="billed-amt">Vendor Billed Total (₹):</label>
						<input id="billed-amt" type="text" bind:value={billedAmount} />
					</div>
					<button class="btn-primary" disabled={matchTesting} onclick={runThreeWayMatch}>
						{matchTesting ? 'Evaluating Match...' : 'Verify 3-Way Match'}
					</button>
				</div>

				{#if matchResult}
					<div class="match-result" class:success={matchResult.isMatched} class:failed={!matchResult.isMatched}>
						<div class="result-header">
							{#if matchResult.isMatched}
								<span class="status-badge match-ok">✅ 3-WAY MATCH AUTHORIZED</span>
							{:else}
								<span class="status-badge match-err">⛔ VARIANCE DETECTED (PAYMENT HOLD)</span>
							{/if}
						</div>
						<div class="result-details">
							<div>PO Ref: <strong>{matchResult.poNumber}</strong> | GRN Ref: <strong>{matchResult.grnNumber}</strong></div>
							<div>Quantity: <strong>{matchResult.quantityStatus}</strong> | Price: <strong>{matchResult.priceStatus}</strong></div>
							<div class="result-note">{matchResult.notes}</div>
						</div>
					</div>
				{/if}
			</div>
		</div>
	{/if}
</div>

<style>
	.procurement-dashboard {
		padding: 1.5rem;
		max-width: 1400px;
		margin: 0 auto;
	}
	.header {
		margin-bottom: 2rem;
	}
	.title {
		font-size: 1.75rem;
		font-weight: 700;
		color: #f8fafc;
	}
	.subtitle {
		font-size: 0.875rem;
		color: #94a3b8;
		margin-top: 0.25rem;
	}
	.error-banner {
		background: rgba(239, 68, 68, 0.1);
		border: 1px solid #ef4444;
		color: #fca5a5;
		padding: 0.75rem 1rem;
		border-radius: 8px;
		margin-bottom: 1.5rem;
	}
	.alert-box {
		background: rgba(245, 158, 11, 0.1);
		border: 1px solid #f59e0b;
		border-radius: 10px;
		padding: 1rem;
		margin-bottom: 1.5rem;
	}
	.alert-header {
		font-size: 0.8125rem;
		font-weight: 700;
		color: #f59e0b;
		margin-bottom: 0.75rem;
	}
	.alert-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
		gap: 0.75rem;
	}
	.alert-card {
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 6px;
		padding: 0.75rem;
	}
	.alert-title {
		font-weight: 600;
		color: #f1f5f9;
		font-size: 0.875rem;
	}
	.alert-meta, .alert-reorder {
		font-size: 0.75rem;
		color: #94a3b8;
		margin-top: 0.25rem;
	}
	.grid-2 {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
		gap: 1.5rem;
	}
	.card {
		background: #1e293b;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 12px;
		padding: 1.5rem;
	}
	.section-title {
		font-size: 1.125rem;
		font-weight: 600;
		color: #f8fafc;
		margin-bottom: 0.5rem;
	}
	.info-text {
		font-size: 0.75rem;
		color: #94a3b8;
		margin-bottom: 1rem;
	}
	.list {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}
	.item-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0.75rem;
		background: rgba(255, 255, 255, 0.03);
		border: 1px solid rgba(255, 255, 255, 0.06);
		border-radius: 8px;
	}
	.item-title {
		font-weight: 600;
		color: #f1f5f9;
	}
	.item-subtitle {
		font-size: 0.75rem;
		color: #94a3b8;
	}
	.row-right {
		text-align: right;
	}
	.item-price {
		font-weight: 700;
		color: #38bdf8;
		font-size: 0.875rem;
	}
	.badge {
		font-size: 0.6875rem;
		padding: 0.15rem 0.4rem;
		border-radius: 4px;
	}
	.status-fully_received {
		background: rgba(16, 185, 129, 0.15);
		color: #34d399;
	}
	.status-issued {
		background: rgba(59, 130, 246, 0.15);
		color: #60a5fa;
	}
	.match-form {
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 8px;
		padding: 1rem;
		margin-bottom: 1rem;
	}
	.form-row {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 0.75rem;
	}
	.form-row label {
		font-size: 0.75rem;
		color: #94a3b8;
	}
	.form-row input {
		width: 160px;
		padding: 0.35rem 0.6rem;
		background: #1e293b;
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: 6px;
		color: white;
	}
	.btn-primary {
		width: 100%;
		background: #6366f1;
		color: white;
		border: none;
		padding: 0.5rem;
		border-radius: 6px;
		font-weight: 600;
		cursor: pointer;
	}
	.match-result {
		padding: 1rem;
		border-radius: 8px;
		margin-top: 1rem;
	}
	.match-result.success {
		background: rgba(16, 185, 129, 0.1);
		border: 1px solid #10b981;
	}
	.match-result.failed {
		background: rgba(239, 68, 68, 0.1);
		border: 1px solid #ef4444;
	}
	.status-badge {
		font-weight: 700;
		font-size: 0.8125rem;
	}
	.match-ok { color: #34d399; }
	.match-err { color: #f87171; }
	.result-details {
		font-size: 0.75rem;
		color: #cbd5e1;
		margin-top: 0.5rem;
	}
	.result-note {
		margin-top: 0.25rem;
		color: #94a3b8;
	}
</style>
