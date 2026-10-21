<script lang="ts">
	import { onMount } from 'svelte';
	import { bankingGrpc } from '$lib/grpc';

	let transactions = $state<any[]>([]);
	let summary = $state<any>(null);
	let loading = $state(true);
	let error = $state('');
	let reconcileRunning = $state(false);
	let reconcileResult = $state<any>(null);

	// CSV file upload content test
	let sampleCsv = `Date,Description,Reference,Debit,Credit
2026-10-21,NEFT/CREDIT/INVOICE_PAYMENT_RELIANCE,INV-2026-0099,0.00,85000.00
2026-10-21,RTGS/DEBIT/OFFICE_SUPPLIES,BILL-2026-0045,12000.00,0.00`;

	onMount(async () => {
		try {
			const [txnRes, sumRes]: [any, any] = await Promise.all([
				bankingGrpc.listBankTransactions({ bankAccountId: 'bank-1' }),
				bankingGrpc.getReconciliationSummary({ bankAccountId: 'bank-1' })
			]);
			transactions = txnRes?.transactions || [];
			summary = sumRes || null;
		} catch (err: any) {
			error = err.message || 'Failed to connect to Banking Reconciliation service';
		} finally {
			loading = false;
		}
	});

	async function runAutoReconcile() {
		reconcileRunning = true;
		try {
			const res: any = await bankingGrpc.runAutoReconciliation({ bankAccountId: 'bank-1' });
			reconcileResult = res;
		} catch (err: any) {
			alert('Auto-reconciliation error: ' + err.message);
		} finally {
			reconcileRunning = false;
		}
	}

	async function importCsvStatement() {
		try {
			const res: any = await bankingGrpc.importBankStatement({
				bankAccountId: 'bank-1',
				statementCsvContent: sampleCsv
			});
			alert(`Successfully parsed & imported ${res.successfullyImported} bank transactions!`);
			const txnRes: any = await bankingGrpc.listBankTransactions({ bankAccountId: 'bank-1' });
			transactions = txnRes?.transactions || [];
		} catch (err: any) {
			alert('CSV import error: ' + err.message);
		}
	}
</script>

<div class="reconciliation-dashboard">
	<div class="header">
		<div>
			<h1 class="title">Bank Feeds & Automated Reconciliation</h1>
			<p class="subtitle">Match bank credits and debits against General Ledger receivables and payables automatically.</p>
		</div>
		<div class="header-actions">
			<button class="btn-secondary" onclick={importCsvStatement}>📥 Import Bank Statement (CSV)</button>
			<button class="btn-primary" disabled={reconcileRunning} onclick={runAutoReconcile}>
				{reconcileRunning ? 'Reconciling...' : '⚡ Run Auto-Reconciliation'}
			</button>
		</div>
	</div>

	{#if error}
		<div class="error-banner">⚠️ {error}</div>
	{/if}

	{#if reconcileResult}
		<div class="success-banner">
			✨ Auto-Reconciliation Complete: <strong>{reconcileResult.autoMatchedCount} transactions matched</strong> (Total: ₹{reconcileResult.totalReconciledVolume}).
		</div>
	{/if}

	{#if summary}
		<div class="kpi-grid">
			<div class="kpi-card">
				<div class="kpi-label">Bank Statement Balance</div>
				<div class="kpi-value">₹{summary.bankStatementBalance}</div>
			</div>
			<div class="kpi-card">
				<div class="kpi-label">General Ledger Balance</div>
				<div class="kpi-value">₹{summary.generalLedgerBalance}</div>
			</div>
			<div class="kpi-card">
				<div class="kpi-label">Unreconciled Variance</div>
				<div class="kpi-value variance">₹{summary.unreconciledVariance}</div>
			</div>
		</div>
	{/if}

	{#if loading}
		<div class="loading-state">Fetching bank transactions...</div>
	{:else}
		<div class="card">
			<h2 class="section-title">🏦 Bank Account Statement Entries</h2>
			<div class="table-container">
				<table class="data-table">
					<thead>
						<tr>
							<th>Date</th>
							<th>Description & Reference</th>
							<th>Debit (Paid)</th>
							<th>Credit (Received)</th>
							<th>Status</th>
							<th>Matched Record</th>
						</tr>
					</thead>
					<tbody>
						{#each transactions as txn}
							<tr>
								<td>{txn.transactionDate}</td>
								<td>
									<div class="txn-desc">{txn.description}</div>
									<div class="txn-ref">Ref: {txn.referenceNumber}</div>
								</td>
								<td class="text-danger">{txn.debitAmount !== '0.00' ? `₹${txn.debitAmount}` : '-'}</td>
								<td class="text-success">{txn.creditAmount !== '0.00' ? `₹${txn.creditAmount}` : '-'}</td>
								<td>
									<span class="badge status-{txn.status.toLowerCase()}">{txn.status}</span>
								</td>
								<td>
									<span class="matched-id">{txn.matchedInvoiceId || txn.matchedBillId || 'Pending Match'}</span>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</div>
	{/if}
</div>

<style>
	.reconciliation-dashboard {
		padding: 1.5rem;
		max-width: 1400px;
		margin: 0 auto;
	}
	.header {
		display: flex;
		justify-content: space-between;
		align-items: center;
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
	.header-actions {
		display: flex;
		gap: 0.75rem;
	}
	.btn-primary {
		background: #6366f1;
		color: white;
		border: none;
		padding: 0.6rem 1.25rem;
		border-radius: 8px;
		font-weight: 600;
		cursor: pointer;
	}
	.btn-secondary {
		background: #1e293b;
		border: 1px solid rgba(255, 255, 255, 0.15);
		color: #cbd5e1;
		padding: 0.6rem 1.25rem;
		border-radius: 8px;
		font-weight: 600;
		cursor: pointer;
	}
	.kpi-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
		gap: 1rem;
		margin-bottom: 1.5rem;
	}
	.kpi-card {
		background: #1e293b;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 10px;
		padding: 1.25rem;
	}
	.kpi-label {
		font-size: 0.75rem;
		color: #94a3b8;
		text-transform: uppercase;
	}
	.kpi-value {
		font-size: 1.5rem;
		font-weight: 700;
		color: #f8fafc;
		margin-top: 0.25rem;
	}
	.kpi-value.variance {
		color: #f59e0b;
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
		margin-bottom: 1rem;
	}
	.data-table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.8125rem;
	}
	.data-table th, .data-table td {
		padding: 0.75rem;
		text-align: left;
		border-bottom: 1px solid rgba(255, 255, 255, 0.06);
	}
	.data-table th {
		color: #94a3b8;
		font-weight: 600;
	}
	.txn-desc {
		font-weight: 600;
		color: #f1f5f9;
	}
	.txn-ref {
		font-size: 0.6875rem;
		color: #94a3b8;
	}
	.text-success { color: #34d399; font-weight: 600; }
	.text-danger { color: #f87171; font-weight: 600; }
	.badge {
		font-size: 0.6875rem;
		padding: 0.15rem 0.4rem;
		border-radius: 4px;
	}
	.status-auto_matched {
		background: rgba(16, 185, 129, 0.15);
		color: #34d399;
	}
	.status-unreconciled {
		background: rgba(245, 158, 11, 0.15);
		color: #fbbf24;
	}
	.matched-id {
		color: #38bdf8;
		font-weight: 500;
	}
	.success-banner {
		background: rgba(16, 185, 129, 0.15);
		border: 1px solid #10b981;
		color: #34d399;
		padding: 0.75rem 1rem;
		border-radius: 8px;
		margin-bottom: 1.5rem;
	}
	.error-banner {
		background: rgba(239, 68, 68, 0.1);
		border: 1px solid #ef4444;
		color: #fca5a5;
		padding: 0.75rem 1rem;
		border-radius: 8px;
		margin-bottom: 1.5rem;
	}
	.loading-state {
		text-align: center;
		padding: 3rem;
		color: #94a3b8;
	}
</style>
