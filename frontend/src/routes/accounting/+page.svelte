<script lang="ts">
	import { onMount } from 'svelte';
	import {
		api,
		type Account,
		type GeneralLedgerEntry,
		type CustomerStatement,
		type TrialBalanceReport,
		type ProfitAndLossReport,
		type BalanceSheetReport,
		type GSTR1Summary,
		type GSTR3BSummary,
		type Customer
	} from '$lib/api';

	let activeTab = $state<'overview' | 'coa' | 'ledger' | 'statement' | 'trial_balance' | 'pnl' | 'balance_sheet' | 'gst_returns'>('overview');
	let loading = $state(true);
	let error = $state('');

	// Data states
	let accounts = $state<Account[]>([]);
	let customers = $state<Customer[]>([]);
	let selectedCustomerId = $state('');
	let customerStatement = $state<CustomerStatement | null>(null);
	let statementLoading = $state(false);

	// General Ledger filter
	let selectedAccountId = $state('');
	let fromDate = $state('');
	let toDate = $state('');
	let ledgerEntries = $state<GeneralLedgerEntry[]>([]);
	let ledgerLoading = $state(false);

	// Trial Balance & P&L
	let trialBalance = $state<TrialBalanceReport | null>(null);
	let pnlReport = $state<ProfitAndLossReport | null>(null);

	// Balance Sheet State
	let balanceSheet = $state<BalanceSheetReport | null>(null);
	let balanceSheetLoading = $state(false);
	let balanceSheetAsOf = $state('');

	// GST Returns State
	let gstSubTab = $state<'gstr1' | 'gstr3b'>('gstr1');
	let gstr1 = $state<GSTR1Summary | null>(null);
	let gstr3b = $state<GSTR3BSummary | null>(null);
	let gstLoading = $state(false);
	let gstFromDate = $state('');
	let gstToDate = $state('');

	async function loadBalanceSheet() {
		balanceSheetLoading = true;
		try {
			balanceSheet = await api.getBalanceSheet(balanceSheetAsOf || undefined);
		} catch (err: any) {
			console.error(err);
		} finally {
			balanceSheetLoading = false;
		}
	}

	async function loadGSTReturns() {
		gstLoading = true;
		try {
			const [r1, r3b] = await Promise.all([
				api.getGSTR1Summary({ from_date: gstFromDate || undefined, to_date: gstToDate || undefined }),
				api.getGSTR3BSummary({ from_date: gstFromDate || undefined, to_date: gstToDate || undefined })
			]);
			gstr1 = r1;
			gstr3b = r3b;
		} catch (err: any) {
			console.error(err);
		} finally {
			gstLoading = false;
		}
	}

	onMount(async () => {
		await loadOverviewData();
	});

	async function loadOverviewData() {
		loading = true;
		error = '';
		try {
			const [accData, custList, tbData, pnlData] = await Promise.all([
				api.listAccounts(),
				api.listCustomers(),
				api.getTrialBalance(),
				api.getProfitAndLoss()
			]);
			accounts = accData;
			customers = custList;
			trialBalance = tbData;
			pnlReport = pnlData;
			if (custList.length > 0) {
				selectedCustomerId = custList[0].id;
				await loadCustomerStatement(custList[0].id);
			}
			await loadLedger();
		} catch (err: any) {
			error = err.message || 'Failed to load accounting data';
		} finally {
			loading = false;
		}
	}

	async function loadLedger() {
		ledgerLoading = true;
		try {
			ledgerEntries = await api.getGeneralLedger({
				account_id: selectedAccountId || undefined,
				from_date: fromDate || undefined,
				to_date: toDate || undefined
			});
		} catch (err: any) {
			console.error(err);
		} finally {
			ledgerLoading = false;
		}
	}

	async function loadCustomerStatement(custId: string) {
		if (!custId) return;
		statementLoading = true;
		try {
			customerStatement = await api.getCustomerStatement(custId);
		} catch (err: any) {
			console.error(err);
		} finally {
			statementLoading = false;
		}
	}

	function formatCurrency(val: string | number | undefined): string {
		if (!val && val !== 0) return '₹0.00';
		const num = typeof val === 'string' ? parseFloat(val) : val;
		if (isNaN(num)) return '₹0.00';
		return new Intl.NumberFormat('en-IN', { style: 'currency', currency: 'INR' }).format(num);
	}

	function formatDate(val: string | undefined): string {
		if (!val) return '';
		return new Date(val).toLocaleDateString('en-IN', { day: '2-digit', month: 'short', year: 'numeric' });
	}

	// Calculate overview metrics from accounts
	let totalReceivables = $derived(
		accounts.find((a) => a.account_code === '1300')?.current_balance || '0.00'
	);
	let bankBalance = $derived(
		accounts.find((a) => a.account_code === '1200')?.current_balance || '0.00'
	);
	let cashBalance = $derived(
		accounts.find((a) => a.account_code === '1100')?.current_balance || '0.00'
	);
	let totalSalesRevenue = $derived(
		pnlReport?.total_revenue || '0.00'
	);
	let netProfit = $derived(
		pnlReport?.net_profit || '0.00'
	);
	let cgstPayable = $derived(
		accounts.find((a) => a.account_code === '2210')?.current_balance || '0.00'
	);
	let sgstPayable = $derived(
		accounts.find((a) => a.account_code === '2220')?.current_balance || '0.00'
	);
	let igstPayable = $derived(
		accounts.find((a) => a.account_code === '2230')?.current_balance || '0.00'
	);
	let totalTaxPayable = $derived(
		(parseFloat(cgstPayable) + parseFloat(sgstPayable) + parseFloat(igstPayable)).toFixed(2)
	);
	let isLocked = $derived(
		error.includes('PLAN_UPGRADE_REQUIRED') ||
		error.includes('Forbidden') ||
		error.includes('This feature requires an upgrade')
	);
</script>

<svelte:head>
	<title>Audited Accounting & Financial Ledger - SaaS Billing</title>
</svelte:head>

<div style="display: flex; flex-direction: column; gap: 1.5rem;">
	<!-- Header -->
	<div style="display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 1rem;">
		<div>
			<h1 style="font-size: 1.75rem; display: flex; align-items: center; gap: 0.5rem;">
				<span>📖</span> Audited Accounting System
			</h1>
			<p style="font-size: 0.875rem; color: var(--text-secondary);">
				Automated double-entry General Ledger connecting invoices, payments, and tax liabilities with mathematical balance verification.
			</p>
		</div>

		<div style="display: flex; gap: 0.5rem; align-items: center;">
			{#if trialBalance?.is_balanced}
				<div style="display: flex; align-items: center; gap: 0.4rem; background: rgba(16, 185, 129, 0.15); border: 1px solid rgba(16, 185, 129, 0.35); padding: 0.4rem 0.75rem; border-radius: var(--radius-md); font-size: 0.75rem; color: #34d399; font-weight: 600;">
					<span>✓</span> Double-Entry Balanced: ₹{formatCurrency(trialBalance.total_debit).replace('₹', '')}
				</div>
			{/if}
			<button onclick={() => window.print()} class="btn btn-secondary btn-sm">
				🖨️ Print Report
			</button>
		</div>
	</div>

	<!-- Navigation Tabs -->
	<div class="card" style="padding: 0.4rem; display: flex; gap: 0.35rem; flex-wrap: wrap; background: rgba(17, 24, 39, 0.8);">
		<button
			onclick={() => (activeTab = 'overview')}
			class="btn btn-sm"
			style="border-radius: var(--radius-md); background: {activeTab === 'overview' ? 'var(--primary)' : 'transparent'}; color: {activeTab === 'overview' ? '#fff' : 'var(--text-secondary)'}; font-weight: 600;"
		>
			📊 Executive Summary
		</button>
		<button
			onclick={() => (activeTab = 'coa')}
			class="btn btn-sm"
			style="border-radius: var(--radius-md); background: {activeTab === 'coa' ? 'var(--primary)' : 'transparent'}; color: {activeTab === 'coa' ? '#fff' : 'var(--text-secondary)'}; font-weight: 600;"
		>
			📑 Chart of Accounts ({accounts.length})
		</button>
		<button
			onclick={() => (activeTab = 'ledger')}
			class="btn btn-sm"
			style="border-radius: var(--radius-md); background: {activeTab === 'ledger' ? 'var(--primary)' : 'transparent'}; color: {activeTab === 'ledger' ? '#fff' : 'var(--text-secondary)'}; font-weight: 600;"
		>
			📜 General Ledger (Daybook)
		</button>
		<button
			onclick={() => (activeTab = 'statement')}
			class="btn btn-sm"
			style="border-radius: var(--radius-md); background: {activeTab === 'statement' ? 'var(--primary)' : 'transparent'}; color: {activeTab === 'statement' ? '#fff' : 'var(--text-secondary)'}; font-weight: 600;"
		>
			👤 Customer Statements
		</button>
		<button
			onclick={() => (activeTab = 'trial_balance')}
			class="btn btn-sm"
			style="border-radius: var(--radius-md); background: {activeTab === 'trial_balance' ? 'var(--primary)' : 'transparent'}; color: {activeTab === 'trial_balance' ? '#fff' : 'var(--text-secondary)'}; font-weight: 600;"
		>
			⚖️ Trial Balance
		</button>
		<button
			onclick={() => (activeTab = 'pnl')}
			class="btn btn-sm"
			style="border-radius: var(--radius-md); background: {activeTab === 'pnl' ? 'var(--primary)' : 'transparent'}; color: {activeTab === 'pnl' ? '#fff' : 'var(--text-secondary)'}; font-weight: 600;"
		>
			📈 Profit & Loss
		</button>
		<button
			onclick={() => { activeTab = 'balance_sheet'; loadBalanceSheet(); }}
			class="btn btn-sm"
			style="border-radius: var(--radius-md); background: {activeTab === 'balance_sheet' ? 'var(--primary)' : 'transparent'}; color: {activeTab === 'balance_sheet' ? '#fff' : 'var(--text-secondary)'}; font-weight: 600;"
		>
			🏛️ Balance Sheet
		</button>
		<button
			onclick={() => { activeTab = 'gst_returns'; loadGSTReturns(); }}
			class="btn btn-sm"
			style="border-radius: var(--radius-md); background: {activeTab === 'gst_returns' ? 'var(--primary)' : 'transparent'}; color: {activeTab === 'gst_returns' ? '#fff' : 'var(--text-secondary)'}; font-weight: 600;"
		>
			🇮🇳 GST Returns
		</button>
	</div>

	{#if loading}
		<div style="padding: 3rem; text-align: center; color: var(--text-muted);">
			Loading audited accounting engine...
		</div>
	{:else if isLocked}
		<div class="card" style="padding: 3rem 2rem; text-align: center; display: flex; flex-direction: column; align-items: center; gap: 1.25rem; border: 1px solid rgba(129, 140, 248, 0.3); background: linear-gradient(135deg, rgba(30, 41, 59, 0.7) 0%, rgba(15, 23, 42, 0.9) 100%);">
			<div style="width: 64px; height: 64px; border-radius: 50%; background: rgba(129, 140, 248, 0.15); display: flex; align-items: center; justify-content: center; font-size: 2rem; border: 1px solid rgba(129, 140, 248, 0.3);">
				🔒
			</div>

			<div style="max-width: 580px;">
				<h2 style="font-size: 1.5rem; margin-bottom: 0.5rem; color: #fff;">
					Audited Accounting is Locked on Basic Plan
				</h2>
				<p style="font-size: 0.9375rem; color: var(--text-secondary); line-height: 1.6;">
					Your current subscription tier is the <strong>Basic Plan</strong>. Full double-entry financial accounting, Chart of Accounts, General Ledger, Trial Balance, and Profit & Loss statements are enabled in the <strong>Next Plan (Business Tier)</strong>.
				</p>
			</div>

			<div style="background: rgba(15, 23, 42, 0.6); padding: 1rem 1.5rem; border-radius: var(--radius-md); border: 1px solid var(--border); max-width: 540px; text-align: left;">
				<div style="font-size: 0.8125rem; font-weight: 700; color: #34d399; margin-bottom: 0.35rem;">
					💡 Included in Basic Plan:
				</div>
				<div style="font-size: 0.8125rem; color: var(--text-secondary); line-height: 1.5;">
					You have full access to customer billing, payments, and individual <strong>Customer Credit & Debit Statements</strong> in the <a href="/customers" style="color: #38bdf8; font-weight: 600;">Customers & Balances</a> module.
				</div>
			</div>

			<div style="display: flex; gap: 1rem; margin-top: 0.5rem; flex-wrap: wrap; justify-content: center;">
				<a href="/customers" class="btn btn-secondary">
					View Customer Credit & Debit Balances →
				</a>
				<a href="/admin/tenants" class="btn btn-primary">
					Activate Next Plan in Companies Master →
				</a>
			</div>
		</div>
	{:else if error}
		<div class="card" style="padding: 2rem; border-color: rgba(244, 63, 94, 0.4); color: #fb7185; text-align: center;">
			⚠️ {error}
		</div>
	{:else}

		<!-- ====================================================================
		     TAB 1: EXECUTIVE OVERVIEW
		     ==================================================================== -->
		{#if activeTab === 'overview'}
			<!-- Financial KPI Grid -->
			<div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 1rem;">
				<div class="card" style="border-left: 4px solid var(--indigo);">
					<div style="font-size: 0.75rem; color: var(--text-muted); font-weight: 600; text-transform: uppercase;">Accounts Receivable</div>
					<div style="font-size: 1.5rem; font-weight: 800; font-family: var(--font-mono); color: #818cf8; margin-top: 0.25rem;">
						{formatCurrency(totalReceivables)}
					</div>
					<div style="font-size: 0.7rem; color: var(--text-muted); margin-top: 0.35rem;">Customer Outstanding Debts</div>
				</div>

				<div class="card" style="border-left: 4px solid var(--emerald);">
					<div style="font-size: 0.75rem; color: var(--text-muted); font-weight: 600; text-transform: uppercase;">Bank & Cash Balances</div>
					<div style="font-size: 1.5rem; font-weight: 800; font-family: var(--font-mono); color: #34d399; margin-top: 0.25rem;">
						{formatCurrency(parseFloat(bankBalance) + parseFloat(cashBalance))}
					</div>
					<div style="font-size: 0.7rem; color: var(--text-muted); margin-top: 0.35rem;">Bank: {formatCurrency(bankBalance)} | Cash: {formatCurrency(cashBalance)}</div>
				</div>

				<div class="card" style="border-left: 4px solid #38bdf8;">
					<div style="font-size: 0.75rem; color: var(--text-muted); font-weight: 600; text-transform: uppercase;">Net Sales Revenue</div>
					<div style="font-size: 1.5rem; font-weight: 800; font-family: var(--font-mono); color: #38bdf8; margin-top: 0.25rem;">
						{formatCurrency(totalSalesRevenue)}
					</div>
					<div style="font-size: 0.7rem; color: var(--text-muted); margin-top: 0.35rem;">Net Operating Earnings: {formatCurrency(netProfit)}</div>
				</div>

				<div class="card" style="border-left: 4px solid var(--amber);">
					<div style="font-size: 0.75rem; color: var(--text-muted); font-weight: 600; text-transform: uppercase;">Output GST Payable</div>
					<div style="font-size: 1.5rem; font-weight: 800; font-family: var(--font-mono); color: #fbbf24; margin-top: 0.25rem;">
						{formatCurrency(totalTaxPayable)}
					</div>
					<div style="font-size: 0.7rem; color: var(--text-muted); margin-top: 0.35rem;">CGST: {formatCurrency(cgstPayable)} | SGST: {formatCurrency(sgstPayable)} | IGST: {formatCurrency(igstPayable)}</div>
				</div>
			</div>

			<!-- Quick Visual Integrity Card -->
			<div class="card" style="background: rgba(17, 24, 39, 0.6); border-color: rgba(99, 102, 241, 0.25);">
				<h3 style="font-size: 1rem; margin-bottom: 0.75rem; display: flex; align-items: center; gap: 0.5rem;">
					<span>🛡️</span> Accounting Audit & Double-Entry Verification
				</h3>
				<div style="display: grid; grid-template-columns: 1fr 1fr 1fr; gap: 1rem;">
					<div style="background: rgba(255,255,255,0.03); padding: 0.85rem; border-radius: var(--radius-md);">
						<div style="font-size: 0.75rem; color: var(--text-muted);">Trial Balance Equilibrium</div>
						<div style="font-size: 1.125rem; font-weight: 700; color: {trialBalance?.is_balanced ? '#34d399' : '#fb7185'}; margin-top: 0.25rem;">
							{trialBalance?.is_balanced ? '✓ 100% Balanced' : '⚠️ Out of Balance'}
						</div>
						<div style="font-size: 0.7rem; color: var(--text-muted); margin-top: 0.2rem;">Difference: ₹{trialBalance?.difference || '0.00'}</div>
					</div>

					<div style="background: rgba(255,255,255,0.03); padding: 0.85rem; border-radius: var(--radius-md);">
						<div style="font-size: 0.75rem; color: var(--text-muted);">Active Ledger Accounts</div>
						<div style="font-size: 1.125rem; font-weight: 700; color: #a5b4fc; margin-top: 0.25rem;">
							{accounts.length} Standard Indian CoA Accounts
						</div>
						<div style="font-size: 0.7rem; color: var(--text-muted); margin-top: 0.2rem;">Fully categorized with GL sequence codes</div>
					</div>

					<div style="background: rgba(255,255,255,0.03); padding: 0.85rem; border-radius: var(--radius-md);">
						<div style="font-size: 0.75rem; color: var(--text-muted);">Transaction Posting Status</div>
						<div style="font-size: 1.125rem; font-weight: 700; color: #34d399; margin-top: 0.25rem;">
							Real-Time Sync Active
						</div>
						<div style="font-size: 0.7rem; color: var(--text-muted); margin-top: 0.2rem;">Invoices, Payments, & Cancellations auto-posted</div>
					</div>
				</div>
			</div>
		{/if}

		<!-- ====================================================================
		     TAB 2: CHART OF ACCOUNTS
		     ==================================================================== -->
		{#if activeTab === 'coa'}
			<div class="table-container card" style="padding: 0;">
				<table class="data-table">
					<thead>
						<tr>
							<th>Code</th>
							<th>Account Name</th>
							<th>Type</th>
							<th>Sub Category</th>
							<th>Description</th>
							<th class="text-right">Current Balance</th>
							<th>System</th>
						</tr>
					</thead>
					<tbody>
						{#each accounts as acc}
							<tr>
								<td style="font-family: var(--font-mono); font-weight: 700; color: #818cf8;">{acc.account_code}</td>
								<td style="font-weight: 600;">{acc.account_name}</td>
								<td>
									<span class="badge" style="background: {acc.account_type === 'ASSET' ? 'rgba(56, 189, 248, 0.15)' : acc.account_type === 'LIABILITY' ? 'rgba(251, 191, 36, 0.15)' : acc.account_type === 'REVENUE' ? 'rgba(52, 211, 153, 0.15)' : 'rgba(244, 63, 94, 0.15)'}; color: {acc.account_type === 'ASSET' ? '#38bdf8' : acc.account_type === 'LIABILITY' ? '#fbbf24' : acc.account_type === 'REVENUE' ? '#34d399' : '#fb7185'}; font-size: 0.7rem;">
										{acc.account_type}
									</span>
								</td>
								<td style="font-size: 0.8125rem; color: var(--text-secondary);">{acc.sub_category || '—'}</td>
								<td style="font-size: 0.75rem; color: var(--text-muted); max-width: 250px;">{acc.description}</td>
								<td class="text-right font-mono" style="font-weight: 700; color: {parseFloat(acc.current_balance) >= 0 ? 'var(--text-primary)' : '#fb7185'};">
									{formatCurrency(acc.current_balance)}
								</td>
								<td>
									{#if acc.is_system}
										<span style="font-size: 0.65rem; color: var(--text-muted); background: rgba(255,255,255,0.06); padding: 0.15rem 0.4rem; border-radius: 3px;">Protected</span>
									{/if}
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}

		<!-- ====================================================================
		     TAB 3: GENERAL LEDGER (DAYBOOK)
		     ==================================================================== -->
		{#if activeTab === 'ledger'}
			<!-- Ledger Filters -->
			<div class="card" style="padding: 1rem; display: flex; gap: 1rem; align-items: flex-end; flex-wrap: wrap; background: rgba(17, 24, 39, 0.6);">
				<div class="form-group" style="margin-bottom: 0; min-width: 200px;">
					<label class="form-label" for="filter-acc">Filter by Account</label>
					<select id="filter-acc" class="form-select" bind:value={selectedAccountId} onchange={loadLedger}>
						<option value="">All Accounts (General Journal)</option>
						{#each accounts as acc}
							<option value={acc.id}>{acc.account_code} - {acc.account_name}</option>
						{/each}
					</select>
				</div>

				<div class="form-group" style="margin-bottom: 0;">
					<label class="form-label" for="from-dt">From Date</label>
					<input id="from-dt" type="date" class="form-input" bind:value={fromDate} onchange={loadLedger} />
				</div>

				<div class="form-group" style="margin-bottom: 0;">
					<label class="form-label" for="to-dt">To Date</label>
					<input id="to-dt" type="date" class="form-input" bind:value={toDate} onchange={loadLedger} />
				</div>

				<button onclick={loadLedger} class="btn btn-secondary btn-sm" style="height: 38px;">
					↻ Refresh Ledger
				</button>
			</div>

			<div class="table-container card" style="padding: 0;">
				<table class="data-table">
					<thead>
						<tr>
							<th>Date</th>
							<th>Voucher #</th>
							<th>Ref Type</th>
							<th>Account</th>
							<th>Narration</th>
							<th class="text-right">Debit (Dr)</th>
							<th class="text-right">Credit (Cr)</th>
							<th class="text-right">Running Balance</th>
						</tr>
					</thead>
					<tbody>
						{#if ledgerLoading}
							<tr>
								<td colspan="8" class="text-center" style="padding: 2rem; color: var(--text-muted);">
									Loading general ledger...
								</td>
							</tr>
						{:else if ledgerEntries.length === 0}
							<tr>
								<td colspan="8" class="text-center" style="padding: 3rem; color: var(--text-muted);">
									No general ledger transactions found for the selected filter.
								</td>
							</tr>
						{:else}
							{#each ledgerEntries as le}
								<tr>
									<td style="font-size: 0.8125rem;">{formatDate(le.entry_date)}</td>
									<td style="font-family: var(--font-mono); font-weight: 700; color: #818cf8;">{le.entry_number}</td>
									<td>
										<span class="badge" style="background: {le.reference_type === 'INVOICE' ? 'rgba(99, 102, 241, 0.15)' : le.reference_type === 'PAYMENT' ? 'rgba(52, 211, 153, 0.15)' : 'rgba(244, 63, 94, 0.15)'}; font-size: 0.65rem;">
											{le.reference_type}
										</span>
									</td>
									<td style="font-size: 0.8125rem;">
										<span style="font-family: var(--font-mono); font-weight: 600; color: #a5b4fc;">{le.account_code}</span> - {le.account_name}
									</td>
									<td style="font-size: 0.75rem; color: var(--text-secondary); max-width: 250px;">{le.line_narration}</td>
									<td class="text-right font-mono" style="font-weight: 600; color: {parseFloat(le.debit_amount) > 0 ? '#38bdf8' : 'var(--text-muted)'};">
										{parseFloat(le.debit_amount) > 0 ? formatCurrency(le.debit_amount) : '—'}
									</td>
									<td class="text-right font-mono" style="font-weight: 600; color: {parseFloat(le.credit_amount) > 0 ? '#34d399' : 'var(--text-muted)'};">
										{parseFloat(le.credit_amount) > 0 ? formatCurrency(le.credit_amount) : '—'}
									</td>
									<td class="text-right font-mono" style="font-weight: 700;">
										{formatCurrency(le.running_balance)}
									</td>
								</tr>
							{/each}
						{/if}
					</tbody>
				</table>
			</div>
		{/if}

		<!-- ====================================================================
		     TAB 4: CUSTOMER STATEMENTS
		     ==================================================================== -->
		{#if activeTab === 'statement'}
			<div class="card" style="padding: 1rem; display: flex; gap: 1rem; align-items: flex-end; flex-wrap: wrap; background: rgba(17, 24, 39, 0.6);">
				<div class="form-group" style="margin-bottom: 0; min-width: 260px;">
					<label class="form-label" for="sel-cust">Select Customer</label>
					<select id="sel-cust" class="form-select" bind:value={selectedCustomerId} onchange={() => loadCustomerStatement(selectedCustomerId)}>
						{#each customers as c}
							<option value={c.id}>{c.name} {c.phone ? `(${c.phone})` : ''}</option>
						{/each}
					</select>
				</div>

				<button onclick={() => loadCustomerStatement(selectedCustomerId)} class="btn btn-secondary btn-sm" style="height: 38px;">
					↻ Generate Statement
				</button>
			</div>

			{#if statementLoading}
				<div style="padding: 3rem; text-align: center; color: var(--text-muted);">
					Compiling customer statement...
				</div>
			{:else if customerStatement}
				<div class="card" style="padding: 1.5rem; background: #0f172a; border-color: rgba(99, 102, 241, 0.3);">
					<!-- Customer Statement Header -->
					<div style="display: flex; justify-content: space-between; align-items: flex-start; border-bottom: 1px solid var(--border-subtle); padding-bottom: 1.25rem; margin-bottom: 1.25rem;">
						<div>
							<div style="font-size: 0.75rem; color: #818cf8; font-weight: 700; text-transform: uppercase;">Customer Statement of Account</div>
							<h2 style="font-size: 1.35rem; margin-top: 0.2rem;">{customerStatement.customer_name}</h2>
							<p style="font-size: 0.8125rem; color: var(--text-secondary); margin-top: 0.25rem;">
								{#if customerStatement.customer_gstin}<strong>GSTIN:</strong> {customerStatement.customer_gstin} | {/if}
								<strong>Phone:</strong> {customerStatement.customer_phone || '—'}<br />
								{customerStatement.customer_address || ''}
							</p>
						</div>
						<div style="text-align: right;">
							<div style="font-size: 0.75rem; color: var(--text-muted);">Closing Balance Due:</div>
							<div style="font-size: 1.5rem; font-weight: 800; font-family: var(--font-mono); color: {parseFloat(customerStatement.closing_balance) > 0 ? '#fb7185' : '#34d399'};">
								{formatCurrency(customerStatement.closing_balance)}
							</div>
							<div style="font-size: 0.75rem; color: var(--text-muted); margin-top: 0.25rem;">
								Total Invoiced: <strong>{formatCurrency(customerStatement.total_invoiced)}</strong><br />
								Total Payments Received: <strong>{formatCurrency(customerStatement.total_paid)}</strong>
							</div>
						</div>
					</div>

					<!-- Statement Table -->
					<table class="data-table">
						<thead>
							<tr>
								<th>Date</th>
								<th>Voucher</th>
								<th>Ref #</th>
								<th>Narration</th>
								<th class="text-right">Billed (Dr)</th>
								<th class="text-right">Paid (Cr)</th>
								<th class="text-right">Balance Due</th>
							</tr>
						</thead>
						<tbody>
							{#if !customerStatement.items || customerStatement.items.length === 0}
								<tr>
									<td colspan="7" class="text-center" style="padding: 2rem; color: var(--text-muted);">
										No accounting activity recorded for this customer yet.
									</td>
								</tr>
							{:else}
								{#each customerStatement.items as it}
									<tr>
										<td style="font-size: 0.8125rem;">{formatDate(it.date)}</td>
										<td style="font-family: var(--font-mono); font-weight: 600; color: #818cf8;">{it.voucher_number}</td>
										<td style="font-family: var(--font-mono); font-weight: 700;">{it.reference_number || '—'}</td>
										<td style="font-size: 0.75rem; color: var(--text-secondary);">{it.narration}</td>
										<td class="text-right font-mono" style="color: {parseFloat(it.debit_amount) > 0 ? '#38bdf8' : 'var(--text-muted)'};">
											{parseFloat(it.debit_amount) > 0 ? formatCurrency(it.debit_amount) : '—'}
										</td>
										<td class="text-right font-mono" style="color: {parseFloat(it.credit_amount) > 0 ? '#34d399' : 'var(--text-muted)'}; font-weight: 600;">
											{parseFloat(it.credit_amount) > 0 ? formatCurrency(it.credit_amount) : '—'}
										</td>
										<td class="text-right font-mono" style="font-weight: 700;">
											{formatCurrency(it.balance)}
										</td>
									</tr>
								{/each}
							{/if}
						</tbody>
					</table>
				</div>
			{/if}
		{/if}

		<!-- ====================================================================
		     TAB 5: TRIAL BALANCE
		     ==================================================================== -->
		{#if activeTab === 'trial_balance' && trialBalance}
			<div class="card" style="padding: 1.5rem;">
				<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem; flex-wrap: wrap; gap: 0.5rem;">
					<div>
						<h2 style="font-size: 1.25rem;">Audited Trial Balance</h2>
						<p style="font-size: 0.8125rem; color: var(--text-muted);">As of {formatDate(trialBalance.as_of_date)}</p>
					</div>
					<div style="display: flex; gap: 1rem; align-items: center;">
						<div style="font-size: 0.8125rem; color: {trialBalance.is_balanced ? '#34d399' : '#fb7185'}; font-weight: 700; background: rgba(255,255,255,0.04); padding: 0.4rem 0.85rem; border-radius: var(--radius-md);">
							{trialBalance.is_balanced ? '✓ MATHEMATICAL INTEGRITY VERIFIED (DEBIT = CREDIT)' : '⚠️ DISCREPANCY DETECTED'}
						</div>
					</div>
				</div>

				<table class="data-table">
					<thead>
						<tr>
							<th>Account Code</th>
							<th>Account Name</th>
							<th>Account Category</th>
							<th class="text-right">Debit Balance (Dr)</th>
							<th class="text-right">Credit Balance (Cr)</th>
						</tr>
					</thead>
					<tbody>
						{#each trialBalance.accounts || [] as row}
							<tr>
								<td style="font-family: var(--font-mono); font-weight: 700; color: #818cf8;">{row.account_code}</td>
								<td style="font-weight: 600;">{row.account_name}</td>
								<td>
									<span class="badge" style="font-size: 0.65rem;">{row.account_type}</span>
								</td>
								<td class="text-right font-mono" style="font-weight: 600; color: {parseFloat(row.debit_amount) > 0 ? '#38bdf8' : 'var(--text-muted)'};">
									{parseFloat(row.debit_amount) > 0 ? formatCurrency(row.debit_amount) : '—'}
								</td>
								<td class="text-right font-mono" style="font-weight: 600; color: {parseFloat(row.credit_amount) > 0 ? '#34d399' : 'var(--text-muted)'};">
									{parseFloat(row.credit_amount) > 0 ? formatCurrency(row.credit_amount) : '—'}
								</td>
							</tr>
						{/each}
						<tr style="background: rgba(255,255,255,0.05); font-weight: 800; border-top: 2px solid var(--border-subtle);">
							<td colspan="3" style="text-transform: uppercase;">Total Trial Balance Sum</td>
							<td class="text-right font-mono" style="color: #38bdf8; font-size: 1rem;">{formatCurrency(trialBalance.total_debit)}</td>
							<td class="text-right font-mono" style="color: #34d399; font-size: 1rem;">{formatCurrency(trialBalance.total_credit)}</td>
						</tr>
					</tbody>
				</table>
			</div>
		{/if}

		<!-- ====================================================================
		     TAB 6: PROFIT AND LOSS
		     ==================================================================== -->
		{#if activeTab === 'pnl' && pnlReport}
			<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1.5rem;">
				<!-- Revenues -->
				<div class="card" style="padding: 1.25rem;">
					<h3 style="font-size: 1.125rem; color: #34d399; margin-bottom: 0.75rem;">Revenues (Income)</h3>
					<table class="data-table">
						<thead>
							<tr>
								<th>Code</th>
								<th>Account</th>
								<th class="text-right">Amount</th>
							</tr>
						</thead>
						<tbody>
							{#if !pnlReport.revenues || pnlReport.revenues.length === 0}
								<tr>
									<td colspan="3" class="text-center" style="color: var(--text-muted); padding: 1.5rem;">No revenue recorded</td>
								</tr>
							{:else}
								{#each pnlReport.revenues as rev}
									<tr>
										<td style="font-family: var(--font-mono); font-weight: 600;">{rev.account_code}</td>
										<td>{rev.account_name}</td>
										<td class="text-right font-mono" style="color: #34d399; font-weight: 700;">{formatCurrency(rev.credit_amount)}</td>
									</tr>
								{/each}
							{/if}
							<tr style="font-weight: 800; border-top: 2px solid var(--border-subtle);">
								<td colspan="2">Total Revenue</td>
								<td class="text-right font-mono" style="color: #34d399;">{formatCurrency(pnlReport.total_revenue)}</td>
							</tr>
						</tbody>
					</table>
				</div>

				<!-- Expenses & Net Profit -->
				<div class="card" style="padding: 1.25rem;">
					<h3 style="font-size: 1.125rem; color: #fb7185; margin-bottom: 0.75rem;">Expenses</h3>
					<table class="data-table">
						<thead>
							<tr>
								<th>Code</th>
								<th>Account</th>
								<th class="text-right">Amount</th>
							</tr>
						</thead>
						<tbody>
							{#if !pnlReport.expenses || pnlReport.expenses.length === 0}
								<tr>
									<td colspan="3" class="text-center" style="color: var(--text-muted); padding: 1.5rem;">No operational expenses recorded</td>
								</tr>
							{:else}
								{#each pnlReport.expenses as exp}
									<tr>
										<td style="font-family: var(--font-mono); font-weight: 600;">{exp.account_code}</td>
										<td>{exp.account_name}</td>
										<td class="text-right font-mono" style="color: #fb7185; font-weight: 700;">{formatCurrency(exp.debit_amount)}</td>
									</tr>
								{/each}
							{/if}
							<tr style="font-weight: 800; border-top: 2px solid var(--border-subtle);">
								<td colspan="2">Total Expenses</td>
								<td class="text-right font-mono" style="color: #fb7185;">{formatCurrency(pnlReport.total_expense)}</td>
							</tr>
						</tbody>
					</table>

					<div style="background: rgba(16, 185, 129, 0.12); border: 1px solid rgba(16, 185, 129, 0.3); border-radius: var(--radius-md); padding: 1rem; margin-top: 1.5rem; display: flex; justify-content: space-between; align-items: center;">
						<div>
							<div style="font-size: 0.75rem; color: #34d399; font-weight: 700; text-transform: uppercase;">Net Operating Profit / (Loss)</div>
							<div style="font-size: 0.75rem; color: var(--text-muted); margin-top: 0.15rem;">Revenue minus Direct & Indirect Expenses</div>
						</div>
						<div style="font-size: 1.5rem; font-weight: 800; font-family: var(--font-mono); color: {parseFloat(pnlReport.net_profit) >= 0 ? '#34d399' : '#fb7185'};">
							{formatCurrency(pnlReport.net_profit)}
						</div>
					</div>
				</div>
			</div>
		{:else if activeTab === 'balance_sheet'}
			<div style="display: flex; flex-direction: column; gap: 1.5rem;">
				<div class="card" style="padding: 1rem; display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 1rem;">
					<div>
						<h2 style="font-size: 1.25rem; font-weight: 700;">Audited Balance Sheet</h2>
						<p style="font-size: 0.8125rem; color: var(--text-secondary);">
							Statement of Financial Position: Assets = Liabilities + Equity
						</p>
					</div>
					<div style="display: flex; align-items: center; gap: 0.75rem;">
						<label class="form-label" style="margin-bottom: 0;" for="asof-date">As of Date:</label>
						<input id="asof-date" type="date" class="form-input" style="width: auto;" bind:value={balanceSheetAsOf} onchange={() => loadBalanceSheet()} />
						<button class="btn btn-primary btn-sm" onclick={() => loadBalanceSheet()} disabled={balanceSheetLoading}>
							{balanceSheetLoading ? 'Refreshing...' : 'Calculate'}
						</button>
					</div>
				</div>

				{#if balanceSheet}
					<div style="padding: 1rem 1.25rem; border-radius: var(--radius-md); display: flex; justify-content: space-between; align-items: center; background: {balanceSheet.is_balanced ? 'rgba(16, 185, 129, 0.15)' : 'rgba(244, 63, 94, 0.15)'}; border: 1px solid {balanceSheet.is_balanced ? 'rgba(16, 185, 129, 0.35)' : 'rgba(244, 63, 94, 0.35)'};">
						<div style="display: flex; align-items: center; gap: 0.75rem;">
							<span style="font-size: 1.5rem;">{balanceSheet.is_balanced ? '⚖️' : '⚠️'}</span>
							<div>
								<div style="font-weight: 700; color: {balanceSheet.is_balanced ? '#34d399' : '#fb7185'};">
									{balanceSheet.is_balanced ? 'Accounting Equation Balanced: Assets = Liabilities + Equity' : 'Imbalance Detected in Ledger'}
								</div>
								<div style="font-size: 0.8125rem; color: var(--text-secondary);">
									Total Assets: {formatCurrency(balanceSheet.total_assets)} | Total Liabilities & Equity: {formatCurrency(balanceSheet.total_liabilities_and_equity)}
								</div>
							</div>
						</div>
						<div class="font-mono" style="font-size: 1.125rem; font-weight: 700; color: {balanceSheet.is_balanced ? '#34d399' : '#fb7185'};">
							Delta: {formatCurrency(balanceSheet.difference)}
						</div>
					</div>

					<div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(320px, 1fr)); gap: 1.5rem;">
						<div class="card" style="padding: 1.25rem;">
							<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.75rem;">
								<h3 style="font-size: 1.125rem; color: #818cf8;">Assets</h3>
								<span class="font-mono" style="font-weight: 700; color: #818cf8;">{formatCurrency(balanceSheet.total_assets)}</span>
							</div>
							<table class="data-table">
								<thead>
									<tr>
										<th>Code</th>
										<th>Asset Account</th>
										<th class="text-right">Balance</th>
									</tr>
								</thead>
								<tbody>
									{#if !balanceSheet.assets || balanceSheet.assets.length === 0}
										<tr><td colspan="3" class="text-center" style="color: var(--text-muted);">No assets found</td></tr>
									{:else}
										{#each balanceSheet.assets as a}
											<tr>
												<td style="font-family: var(--font-mono); font-weight: 600;">{a.account_code}</td>
												<td>{a.account_name}</td>
												<td class="text-right font-mono" style="font-weight: 600;">{formatCurrency(a.balance)}</td>
											</tr>
										{/each}
									{/if}
								</tbody>
							</table>
						</div>

						<div class="card" style="padding: 1.25rem;">
							<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.75rem;">
								<h3 style="font-size: 1.125rem; color: #fb7185;">Liabilities</h3>
								<span class="font-mono" style="font-weight: 700; color: #fb7185;">{formatCurrency(balanceSheet.total_liabilities)}</span>
							</div>
							<table class="data-table">
								<thead>
									<tr>
										<th>Code</th>
										<th>Liability Account</th>
										<th class="text-right">Balance</th>
									</tr>
								</thead>
								<tbody>
									{#if !balanceSheet.liabilities || balanceSheet.liabilities.length === 0}
										<tr><td colspan="3" class="text-center" style="color: var(--text-muted);">No liabilities found</td></tr>
									{:else}
										{#each balanceSheet.liabilities as l}
											<tr>
												<td style="font-family: var(--font-mono); font-weight: 600;">{l.account_code}</td>
												<td>{l.account_name}</td>
												<td class="text-right font-mono" style="font-weight: 600;">{formatCurrency(l.balance)}</td>
											</tr>
										{/each}
									{/if}
								</tbody>
							</table>
						</div>

						<div class="card" style="padding: 1.25rem;">
							<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.75rem;">
								<h3 style="font-size: 1.125rem; color: #34d399;">Equity & Net Profit</h3>
								<span class="font-mono" style="font-weight: 700; color: #34d399;">{formatCurrency(parseFloat(balanceSheet.total_equity) + parseFloat(balanceSheet.net_profit_to_date))}</span>
							</div>
							<table class="data-table">
								<thead>
									<tr>
										<th>Code</th>
										<th>Equity Component</th>
										<th class="text-right">Balance</th>
									</tr>
								</thead>
								<tbody>
									{#each balanceSheet.equity as eq}
										<tr>
											<td style="font-family: var(--font-mono); font-weight: 600;">{eq.account_code}</td>
											<td>{eq.account_name}</td>
											<td class="text-right font-mono" style="font-weight: 600;">{formatCurrency(eq.balance)}</td>
										</tr>
									{/each}
									<tr style="background: rgba(16, 185, 129, 0.08);">
										<td style="font-family: var(--font-mono); font-weight: 700;">P&L</td>
										<td style="font-weight: 600;">Net Profit / Earnings to Date</td>
										<td class="text-right font-mono" style="font-weight: 700; color: #34d399;">{formatCurrency(balanceSheet.net_profit_to_date)}</td>
									</tr>
								</tbody>
							</table>
						</div>
					</div>
				{/if}
			</div>

		{:else if activeTab === 'gst_returns'}
			<div style="display: flex; flex-direction: column; gap: 1.5rem;">
				<div class="card" style="padding: 1.25rem; display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 1rem;">
					<div>
						<h2 style="font-size: 1.25rem; font-weight: 700; display: flex; align-items: center; gap: 0.5rem;">
							<span>🇮🇳</span> Statutory GST Outward Return Filing Summaries
						</h2>
						<p style="font-size: 0.8125rem; color: var(--text-secondary);">
							Audit-ready summary tables for filing GSTR-1 and GSTR-3B on the GST Common Portal
						</p>
					</div>

					<div style="display: flex; align-items: center; gap: 0.5rem; flex-wrap: wrap;">
						<input type="date" class="form-input" style="width: auto;" bind:value={gstFromDate} />
						<span style="color: var(--text-muted);">to</span>
						<input type="date" class="form-input" style="width: auto;" bind:value={gstToDate} />
						<button class="btn btn-primary btn-sm" onclick={() => loadGSTReturns()} disabled={gstLoading}>
							{gstLoading ? 'Loading...' : 'Generate Returns'}
						</button>
					</div>
				</div>

				<div style="display: flex; gap: 0.5rem;">
					<button
						class="btn btn-sm"
						style="border-radius: var(--radius-md); background: {gstSubTab === 'gstr1' ? 'var(--primary)' : 'rgba(255,255,255,0.06)'}; color: {gstSubTab === 'gstr1' ? '#fff' : 'var(--text-secondary)'}; font-weight: 600;"
						onclick={() => (gstSubTab = 'gstr1')}
					>
						📑 GSTR-1 Outward Supplies
					</button>
					<button
						class="btn btn-sm"
						style="border-radius: var(--radius-md); background: {gstSubTab === 'gstr3b' ? 'var(--primary)' : 'rgba(255,255,255,0.06)'}; color: {gstSubTab === 'gstr3b' ? '#fff' : 'var(--text-secondary)'}; font-weight: 600;"
						onclick={() => (gstSubTab = 'gstr3b')}
					>
						⚖️ GSTR-3B Monthly Tax & ITC Summary
					</button>
				</div>

				{#if gstSubTab === 'gstr1' && gstr1}
					<div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 1rem;">
						<div class="card" style="padding: 1rem;">
							<span style="font-size: 0.75rem; color: var(--text-muted); text-transform: uppercase;">Total Invoices</span>
							<div class="font-mono" style="font-size: 1.5rem; font-weight: 700; margin-top: 0.25rem;">{gstr1.total_invoices}</div>
						</div>
						<div class="card" style="padding: 1rem;">
							<span style="font-size: 0.75rem; color: var(--text-muted); text-transform: uppercase;">Total Taxable Value</span>
							<div class="font-mono" style="font-size: 1.5rem; font-weight: 700; color: #818cf8; margin-top: 0.25rem;">{formatCurrency(gstr1.total_taxable)}</div>
						</div>
						<div class="card" style="padding: 1rem;">
							<span style="font-size: 0.75rem; color: var(--text-muted); text-transform: uppercase;">Total CGST</span>
							<div class="font-mono" style="font-size: 1.5rem; font-weight: 700; color: #34d399; margin-top: 0.25rem;">{formatCurrency(gstr1.total_cgst)}</div>
						</div>
						<div class="card" style="padding: 1rem;">
							<span style="font-size: 0.75rem; color: var(--text-muted); text-transform: uppercase;">Total SGST</span>
							<div class="font-mono" style="font-size: 1.5rem; font-weight: 700; color: #34d399; margin-top: 0.25rem;">{formatCurrency(gstr1.total_sgst)}</div>
						</div>
						<div class="card" style="padding: 1rem;">
							<span style="font-size: 0.75rem; color: var(--text-muted); text-transform: uppercase;">Total IGST</span>
							<div class="font-mono" style="font-size: 1.5rem; font-weight: 700; color: #60a5fa; margin-top: 0.25rem;">{formatCurrency(gstr1.total_igst)}</div>
						</div>
					</div>

					<div class="card" style="padding: 1.25rem;">
						<h3 style="font-size: 1rem; font-weight: 700; margin-bottom: 0.75rem;">Table 4: B2B Invoices (Registered Buyers)</h3>
						<table class="data-table">
							<thead>
								<tr>
									<th>Invoice #</th>
									<th>Date</th>
									<th>Customer</th>
									<th>GSTIN</th>
									<th class="text-right">Taxable</th>
									<th class="text-right">CGST</th>
									<th class="text-right">SGST</th>
									<th class="text-right">IGST</th>
									<th class="text-right">Invoice Total</th>
								</tr>
							</thead>
							<tbody>
								{#if !gstr1.b2b_invoices || gstr1.b2b_invoices.length === 0}
									<tr><td colspan="9" class="text-center" style="color: var(--text-muted); padding: 1rem;">No B2B registered invoices in this period</td></tr>
								{:else}
									{#each gstr1.b2b_invoices as inv}
										<tr>
											<td class="font-mono" style="color: #818cf8; font-weight: 600;">{inv.invoice_number}</td>
											<td>{formatDate(inv.invoice_date)}</td>
											<td>{inv.customer_name}</td>
											<td class="font-mono" style="color: #34d399;">{inv.customer_gstin}</td>
											<td class="text-right font-mono">{formatCurrency(inv.taxable_amount)}</td>
											<td class="text-right font-mono">{formatCurrency(inv.cgst_amount)}</td>
											<td class="text-right font-mono">{formatCurrency(inv.sgst_amount)}</td>
											<td class="text-right font-mono">{formatCurrency(inv.igst_amount)}</td>
											<td class="text-right font-mono" style="font-weight: 700;">{formatCurrency(inv.total_invoice)}</td>
										</tr>
									{/each}
								{/if}
							</tbody>
						</table>
					</div>

					<div class="card" style="padding: 1.25rem;">
						<h3 style="font-size: 1rem; font-weight: 700; margin-bottom: 0.75rem;">Table 12: HSN Summary of Outward Supplies</h3>
						<table class="data-table">
							<thead>
								<tr>
									<th>HSN/SAC</th>
									<th>Description</th>
									<th>UQC</th>
									<th class="text-right">Total Qty</th>
									<th class="text-right">Taxable Value</th>
									<th class="text-right">CGST</th>
									<th class="text-right">SGST</th>
									<th class="text-right">IGST</th>
									<th class="text-right">Total</th>
								</tr>
							</thead>
							<tbody>
								{#if !gstr1.hsn_summary || gstr1.hsn_summary.length === 0}
									<tr><td colspan="9" class="text-center" style="color: var(--text-muted); padding: 1rem;">No HSN items recorded in this period</td></tr>
								{:else}
									{#each gstr1.hsn_summary as h}
										<tr>
											<td class="font-mono" style="font-weight: 600; color: #818cf8;">{h.hsn_sac}</td>
											<td>{h.description}</td>
											<td class="font-mono">{h.unit_code}</td>
											<td class="text-right font-mono">{h.total_quantity}</td>
											<td class="text-right font-mono">{formatCurrency(h.taxable_amount)}</td>
											<td class="text-right font-mono">{formatCurrency(h.cgst_amount)}</td>
											<td class="text-right font-mono">{formatCurrency(h.sgst_amount)}</td>
											<td class="text-right font-mono">{formatCurrency(h.igst_amount)}</td>
											<td class="text-right font-mono" style="font-weight: 700;">{formatCurrency(h.total_amount)}</td>
										</tr>
									{/each}
								{/if}
							</tbody>
						</table>
					</div>

				{:else if gstSubTab === 'gstr3b' && gstr3b}
					<div style="display: flex; flex-direction: column; gap: 1.25rem;">
						<div class="card" style="padding: 1.25rem;">
							<h3 style="font-size: 1.125rem; font-weight: 700; margin-bottom: 0.5rem;">3.1 Details of Outward Supplies & Inward Supplies liable to Reverse Charge</h3>
							<table class="data-table">
								<thead>
									<tr>
										<th>Nature of Supplies</th>
										<th class="text-right">Total Taxable Value</th>
										<th class="text-right">Integrated Tax (IGST)</th>
										<th class="text-right">Central Tax (CGST)</th>
										<th class="text-right">State/UT Tax (SGST)</th>
									</tr>
								</thead>
								<tbody>
									<tr>
										<td style="font-weight: 600;">{gstr3b.outward_taxable.nature_of_supplies}</td>
										<td class="text-right font-mono" style="font-weight: 700;">{formatCurrency(gstr3b.outward_taxable.total_taxable)}</td>
										<td class="text-right font-mono" style="color: #60a5fa;">{formatCurrency(gstr3b.outward_taxable.igst)}</td>
										<td class="text-right font-mono" style="color: #34d399;">{formatCurrency(gstr3b.outward_taxable.cgst)}</td>
										<td class="text-right font-mono" style="color: #34d399;">{formatCurrency(gstr3b.outward_taxable.sgst)}</td>
									</tr>
								</tbody>
							</table>
						</div>

						<div class="card" style="padding: 1.25rem;">
							<h3 style="font-size: 1.125rem; font-weight: 700; margin-bottom: 0.5rem;">4. Eligible Input Tax Credit (ITC) from Vendor Inward Bills</h3>
							<table class="data-table">
								<thead>
									<tr>
										<th>Details</th>
										<th class="text-right">Total Taxable Value</th>
										<th class="text-right">Integrated Tax (IGST)</th>
										<th class="text-right">Central Tax (CGST)</th>
										<th class="text-right">State/UT Tax (SGST)</th>
									</tr>
								</thead>
								<tbody>
									<tr>
										<td style="font-weight: 600;">{gstr3b.inward_eligible_itc.details}</td>
										<td class="text-right font-mono">{formatCurrency(gstr3b.inward_eligible_itc.total_taxable)}</td>
										<td class="text-right font-mono" style="color: #60a5fa;">{formatCurrency(gstr3b.inward_eligible_itc.igst)}</td>
										<td class="text-right font-mono" style="color: #34d399;">{formatCurrency(gstr3b.inward_eligible_itc.cgst)}</td>
										<td class="text-right font-mono" style="color: #34d399;">{formatCurrency(gstr3b.inward_eligible_itc.sgst)}</td>
									</tr>
								</tbody>
							</table>
						</div>

						<div class="card" style="padding: 1.25rem; border-color: var(--border-accent); background: radial-gradient(circle at top right, rgba(99, 102, 241, 0.12), rgba(17, 24, 39, 0.9));">
							<h3 style="font-size: 1.125rem; font-weight: 700; margin-bottom: 0.5rem;">6.1 Net Tax Payable in Cash (Outward Liabilities less Eligible ITC)</h3>
							<table class="data-table">
								<thead>
									<tr>
										<th>Description</th>
										<th class="text-right">IGST Cash Payable</th>
										<th class="text-right">CGST Cash Payable</th>
										<th class="text-right">SGST Cash Payable</th>
									</tr>
								</thead>
								<tbody>
									<tr style="font-weight: 700;">
										<td>Net Outflow to Government</td>
										<td class="text-right font-mono" style="color: #60a5fa; font-size: 1.125rem;">{formatCurrency(gstr3b.net_tax_payable.igst)}</td>
										<td class="text-right font-mono" style="color: #34d399; font-size: 1.125rem;">{formatCurrency(gstr3b.net_tax_payable.cgst)}</td>
										<td class="text-right font-mono" style="color: #34d399; font-size: 1.125rem;">{formatCurrency(gstr3b.net_tax_payable.sgst)}</td>
									</tr>
								</tbody>
							</table>
						</div>
					</div>
				{/if}
			</div>
		{/if}

	{/if}
</div>
