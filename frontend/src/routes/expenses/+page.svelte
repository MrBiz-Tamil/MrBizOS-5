<script lang="ts">
	import { onMount } from 'svelte';
	import { api, type Expense, type ExpenseSummary, type TenantSubscriptionInfo, type User } from '$lib/api';

	let currentUser = $state<User | null>(null);
	let subInfo = $state<TenantSubscriptionInfo | null>(null);
	let expenses = $state<Expense[]>([]);
	let summary = $state<ExpenseSummary | null>(null);
	let loading = $state(true);
	let error = $state('');

	// Feature Gating flag
	let isFeatureUnlocked = $derived(
		currentUser?.is_super_admin ||
		subInfo?.plan_code === 'BUSINESS' ||
		subInfo?.plan_code === 'PRO' ||
		subInfo?.plan_code === 'ENTERPRISE' ||
		subInfo?.features?.expenses === true
	);

	// Filters
	let filterCategory = $state('');
	let filterFromDate = $state('');
	let filterToDate = $state('');

	// Record Expense Modal
	let showCreateModal = $state(false);
	let category = $state('Office Supplies');
	let amount = $state('');
	let taxAmount = $state('0.00');
	let paymentMode = $state('BANK_TRANSFER');
	let payeeName = $state('');
	let expenseDate = $state(new Date().toISOString().split('T')[0]);
	let description = $state('');
	let receiptUrl = $state('');
	let modalError = $state('');
	let submitting = $state(false);

	const categories = [
		'Office Supplies',
		'Rent & Utilities',
		'Salaries & Wages',
		'Travel & Transportation',
		'Software & Subscriptions',
		'Marketing & Ads',
		'Repairs & Maintenance',
		'Vendor Services',
		'Taxes & Legal Fees',
		'Miscellaneous'
	];

	const paymentModes = [
		{ id: 'BANK_TRANSFER', name: 'Bank Transfer (NEFT/RTGS/IMPS)' },
		{ id: 'UPI', name: 'UPI / QR Code' },
		{ id: 'CASH', name: 'Cash on Hand' },
		{ id: 'CARD', name: 'Corporate Credit/Debit Card' },
		{ id: 'CHEQUE', name: 'Cheque' }
	];

	onMount(async () => {
		try {
			const me = await api.getMe();
			currentUser = me.user;
			try {
				subInfo = await api.getCompanySubscription();
			} catch (e) {}

			if (isFeatureUnlocked) {
				await loadExpenses();
			}
		} catch (err: any) {
			error = err.message || 'Failed to initialize expense dashboard';
		} finally {
			loading = false;
		}
	});

	async function loadExpenses() {
		loading = true;
		error = '';
		try {
			const [expList, expSum] = await Promise.all([
				api.listExpenses({
					category: filterCategory || undefined,
					from_date: filterFromDate || undefined,
					to_date: filterToDate || undefined
				}).catch(() => []),
				api.getExpenseSummary({
					from_date: filterFromDate || undefined,
					to_date: filterToDate || undefined
				}).catch(() => null)
			]);
			expenses = expList || [];
			summary = expSum;
		} catch (err: any) {
			expenses = [];
			if (err.message?.includes('PLAN_UPGRADE_REQUIRED') || err.message?.includes('Forbidden')) {
				// Feature is locked
				if (subInfo && subInfo.features) {
					subInfo.features.expenses = false;
				}
			} else {
				error = err.message || 'Failed to load expenses';
			}
		} finally {
			loading = false;
		}
	}

	function formatCurrency(val: string | number) {
		const num = typeof val === 'string' ? parseFloat(val) : val;
		if (isNaN(num)) return '₹0.00';
		return '₹' + num.toLocaleString('en-IN', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
	}

	async function handleCreateExpense(e: Event) {
		e.preventDefault();
		modalError = '';
		const trimmedPayee = payeeName.trim();
		const parsedAmt = parseFloat(amount);

		if (!trimmedPayee) {
			modalError = 'Payee / Vendor name is required';
			return;
		}
		if (isNaN(parsedAmt) || parsedAmt <= 0) {
			modalError = 'Expense amount must be greater than zero';
			return;
		}

		submitting = true;
		try {
			await api.createExpense({
				category,
				amount: amount,
				tax_amount: taxAmount || '0.00',
				payment_mode: paymentMode,
				payee_name: trimmedPayee,
				expense_date: expenseDate,
				description: description.trim(),
				receipt_url: receiptUrl.trim()
			});

			showCreateModal = false;
			amount = '';
			payeeName = '';
			description = '';
			receiptUrl = '';
			await loadExpenses();
		} catch (err: any) {
			modalError = err.message || 'Failed to record expense';
		} finally {
			submitting = false;
		}
	}
</script>

<svelte:head>
	<title>Expense Management - SaaS Billing</title>
</svelte:head>

<div style="display: flex; flex-direction: column; gap: 1.75rem;">
	<!-- Header Bar -->
	<div style="display: flex; justify-content: space-between; align-items: flex-start; flex-wrap: wrap; gap: 1rem;">
		<div>
			<div style="display: flex; align-items: center; gap: 0.5rem; margin-bottom: 0.25rem;">
				<h1 style="font-size: 1.75rem; margin: 0;">Expense Management & Vouchers</h1>
				<span style="font-size: 0.75rem; font-weight: 700; background: rgba(129, 140, 248, 0.15); color: #a5b4fc; padding: 0.2rem 0.6rem; border-radius: 9999px; border: 1px solid rgba(129, 140, 248, 0.3);">
					💎 Next Plan Tier
				</span>
			</div>
			<p style="font-size: 0.875rem; color: var(--text-secondary); margin: 0;">
				Track company operational expenditure, vendor receipts, and payment method disbursements.
			</p>
		</div>

		{#if isFeatureUnlocked}
			<div style="display: flex; gap: 0.75rem;">
				<button onclick={loadExpenses} class="btn btn-secondary btn-sm" disabled={loading}>
					{loading ? 'Refreshing...' : '🔄 Refresh'}
				</button>
				<button onclick={() => (showCreateModal = true)} class="btn btn-primary btn-sm">
					+ Record Expense
				</button>
			</div>
		{/if}
	</div>

	<!-- If Feature is Locked under Basic Plan -->
	{#if !isFeatureUnlocked}
		<div class="card" style="padding: 3rem 2rem; text-align: center; display: flex; flex-direction: column; align-items: center; gap: 1.25rem; border: 1px solid rgba(129, 140, 248, 0.3); background: linear-gradient(135deg, rgba(30, 41, 59, 0.7) 0%, rgba(15, 23, 42, 0.9) 100%);">
			<div style="width: 64px; height: 64px; border-radius: 50%; background: rgba(129, 140, 248, 0.15); display: flex; align-items: center; justify-content: center; font-size: 2rem; border: 1px solid rgba(129, 140, 248, 0.3);">
				🔒
			</div>

			<div style="max-width: 580px;">
				<h2 style="font-size: 1.5rem; margin-bottom: 0.5rem; color: #fff;">
					Expense Management is Locked on Basic Plan
				</h2>
				<p style="font-size: 0.9375rem; color: var(--text-secondary); line-height: 1.6;">
					Your current subscription tier is the <strong>Basic Plan</strong>. Expense tracking, vendor vouchers, receipt auditing, and automated double-entry postings are available in the <strong>Next Plan (Business Tier)</strong>.
				</p>
			</div>

			<!-- Next Plan Benefits List -->
			<div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 1rem; width: 100%; max-width: 680px; text-align: left; margin: 1rem 0;">
				<div style="background: rgba(15, 23, 42, 0.6); padding: 1rem; border-radius: var(--radius-md); border: 1px solid var(--border);">
					<div style="font-weight: 700; font-size: 0.875rem; color: #38bdf8; margin-bottom: 0.25rem;">💸 Expense Tracking</div>
					<div style="font-size: 0.75rem; color: var(--text-muted);">Categorize rent, utilities, salaries, and vendor bills.</div>
				</div>
				<div style="background: rgba(15, 23, 42, 0.6); padding: 1rem; border-radius: var(--radius-md); border: 1px solid var(--border);">
					<div style="font-weight: 700; font-size: 0.875rem; color: #34d399; margin-bottom: 0.25rem;">🧾 Receipt Auditing</div>
					<div style="font-size: 0.75rem; color: var(--text-muted);">Attach invoices, payment proofs, and digital receipts.</div>
				</div>
				<div style="background: rgba(15, 23, 42, 0.6); padding: 1rem; border-radius: var(--radius-md); border: 1px solid var(--border);">
					<div style="font-weight: 700; font-size: 0.875rem; color: #818cf8; margin-bottom: 0.25rem;">📖 Auto-Accounting</div>
					<div style="font-size: 0.75rem; color: var(--text-muted);">Expenses automatically flow into P&L and Trial Balance.</div>
				</div>
			</div>

			{#if currentUser?.is_super_admin}
				<a href="/admin/tenants" class="btn btn-primary">
					Activate Next Plan in Companies Master →
				</a>
			{:else}
				<div style="font-size: 0.8125rem; color: var(--text-muted);">
					Please contact your Super Administrator to upgrade your company account to the <strong>Next Plan (Business Tier)</strong>.
				</div>
			{/if}
		</div>
	{:else}
		<!-- Unlocked Dashboard View -->
		{#if error}
			<div style="background: rgba(244, 63, 94, 0.15); border: 1px solid rgba(244, 63, 94, 0.3); border-radius: var(--radius-md); padding: 1rem; color: #fb7185;">
				⚠️ {error}
			</div>
		{/if}

		<!-- KPI Cards -->
		<div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(240px, 1fr)); gap: 1.25rem;">
			<div class="card" style="padding: 1.25rem; display: flex; flex-direction: column; gap: 0.4rem; border-left: 4px solid #f43f5e;">
				<span style="font-size: 0.75rem; font-weight: 700; color: var(--text-muted); text-transform: uppercase;">
					Total Expenses Recorded
				</span>
				<span style="font-size: 1.5rem; font-weight: 800; color: #fb7185; font-family: var(--font-heading);">
					{formatCurrency(summary?.total_expenses || 0)}
				</span>
				<span style="font-size: 0.75rem; color: var(--text-secondary);">
					Total expenditure across all categories
				</span>
			</div>

			<div class="card" style="padding: 1.25rem; display: flex; flex-direction: column; gap: 0.4rem; border-left: 4px solid #818cf8;">
				<span style="font-size: 0.75rem; font-weight: 700; color: var(--text-muted); text-transform: uppercase;">
					Expense Vouchers Count
				</span>
				<span style="font-size: 1.5rem; font-weight: 800; color: #fff; font-family: var(--font-heading);">
					{summary?.expense_count || 0} Entries
				</span>
				<span style="font-size: 0.75rem; color: var(--text-secondary);">
					Individual expenditure claims posted
				</span>
			</div>

			<div class="card" style="padding: 1.25rem; display: flex; flex-direction: column; gap: 0.4rem; border-left: 4px solid #38bdf8;">
				<span style="font-size: 0.75rem; font-weight: 700; color: var(--text-muted); text-transform: uppercase;">
					Top Spending Category
				</span>
				<span style="font-size: 1.25rem; font-weight: 700; color: #38bdf8; font-family: var(--font-heading); text-overflow: ellipsis; overflow: hidden; white-space: nowrap;">
					{summary?.categories && summary.categories.length > 0 ? summary.categories[0].category : 'No Data'}
				</span>
				<span style="font-size: 0.75rem; color: var(--text-secondary);">
					{summary?.categories && summary.categories.length > 0 ? formatCurrency(summary.categories[0].total_amount) : '0.00'}
				</span>
			</div>
		</div>

		<!-- Filter Bar -->
		<div class="card" style="padding: 0.75rem 1rem; display: flex; gap: 1rem; align-items: flex-end; flex-wrap: wrap;">
			<div style="display: flex; flex-direction: column; gap: 0.25rem; min-width: 180px;">
				<label for="f-cat" style="font-size: 0.75rem; color: var(--text-muted);">Category Filter</label>
				<select id="f-cat" class="form-select" style="padding: 0.3rem 0.6rem; font-size: 0.8125rem;" bind:value={filterCategory} onchange={loadExpenses}>
					<option value="">All Categories</option>
					{#each categories as c}
						<option value={c}>{c}</option>
					{/each}
				</select>
			</div>

			<div style="display: flex; flex-direction: column; gap: 0.25rem;">
				<label for="f-from" style="font-size: 0.75rem; color: var(--text-muted);">From Date</label>
				<input id="f-from" type="date" class="form-input" style="padding: 0.3rem 0.6rem; font-size: 0.8125rem;" bind:value={filterFromDate} onchange={loadExpenses} />
			</div>

			<div style="display: flex; flex-direction: column; gap: 0.25rem;">
				<label for="f-to" style="font-size: 0.75rem; color: var(--text-muted);">To Date</label>
				<input id="f-to" type="date" class="form-input" style="padding: 0.3rem 0.6rem; font-size: 0.8125rem;" bind:value={filterToDate} onchange={loadExpenses} />
			</div>

			<button
				onclick={() => { filterCategory = ''; filterFromDate = ''; filterToDate = ''; loadExpenses(); }}
				class="btn btn-secondary btn-sm"
				style="height: 34px;"
			>
				Reset Filters
			</button>
		</div>

		<!-- Expenses List Table -->
		<div class="table-container card" style="padding: 0; overflow-x: auto;">
			<table class="data-table">
				<thead>
					<tr>
						<th>Expense #</th>
						<th>Date</th>
						<th>Category</th>
						<th>Payee / Vendor</th>
						<th>Payment Mode</th>
						<th>Description</th>
						<th class="text-right">Amount (₹)</th>
						<th>Receipt</th>
					</tr>
				</thead>
				<tbody>
					{#if loading}
						<tr>
							<td colspan="8" class="text-center" style="padding: 3rem; color: var(--text-muted);">
								Loading expenses...
							</td>
						</tr>
					{:else if expenses.length === 0}
						<tr>
							<td colspan="8" class="text-center" style="padding: 3rem; color: var(--text-muted);">
								No expenses recorded yet. Click "+ Record Expense" to post your first expense voucher.
							</td>
						</tr>
					{:else}
						{#each expenses as exp}
							<tr>
								<td style="font-family: var(--font-mono); font-weight: 700; color: #818cf8; font-size: 0.8125rem;">
									{exp.expense_number}
								</td>
								<td style="font-family: var(--font-mono); font-size: 0.8125rem; color: var(--text-secondary);">
									{new Date(exp.expense_date).toLocaleDateString('en-IN')}
								</td>
								<td>
									<span class="badge" style="background: rgba(99, 102, 241, 0.15); color: #a5b4fc;">
										{exp.category}
									</span>
								</td>
								<td style="font-weight: 600; color: #fff;">
									{exp.payee_name}
								</td>
								<td style="font-size: 0.75rem; color: var(--text-muted);">
									{exp.payment_mode}
								</td>
								<td style="font-size: 0.8125rem; color: var(--text-secondary); max-width: 250px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">
									{exp.description || '-'}
								</td>
								<td class="text-right font-mono" style="font-weight: 700; color: #fb7185;">
									{formatCurrency(exp.amount)}
								</td>
								<td>
									{#if exp.receipt_url}
										<a href={exp.receipt_url} target="_blank" rel="noopener noreferrer" class="btn btn-secondary btn-sm" style="font-size: 0.7rem; padding: 0.2rem 0.5rem;">
											📎 View
										</a>
									{:else}
										<span style="font-size: 0.75rem; color: var(--text-muted);">-</span>
									{/if}
								</td>
							</tr>
						{/each}
					{/if}
				</tbody>
			</table>
		</div>
	{/if}
</div>

<!-- Record Expense Modal -->
{#if showCreateModal}
	<div class="modal-backdrop">
		<div class="modal-card animate-fade-in" style="max-width: 600px;">
			<div class="modal-header">
				<h3 style="font-size: 1.125rem;">Record Expense Voucher</h3>
				<button onclick={() => (showCreateModal = false)} style="background: none; border: none; color: var(--text-muted); cursor: pointer; font-size: 1.25rem;">✕</button>
			</div>

			<form onsubmit={handleCreateExpense}>
				<div class="modal-body" style="display: flex; flex-direction: column; gap: 1rem;">
					{#if modalError}
						<div style="background: rgba(244, 63, 94, 0.15); border: 1px solid rgba(244, 63, 94, 0.3); border-radius: var(--radius-md); padding: 0.75rem; color: #fb7185; font-size: 0.8125rem;">
							⚠️ {modalError}
						</div>
					{/if}

					<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="exp-cat">Expense Category *</label>
							<select id="exp-cat" class="form-select" bind:value={category} required>
								{#each categories as c}
									<option value={c}>{c}</option>
								{/each}
							</select>
						</div>

						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="exp-payee">Payee / Vendor Name *</label>
							<input id="exp-payee" type="text" class="form-input" bind:value={payeeName} placeholder="e.g. AWS Cloud, Landlord, Office Mart" required />
						</div>
					</div>

					<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="exp-amt">Total Amount (₹) *</label>
							<input id="exp-amt" type="number" step="0.01" min="0.01" class="form-input font-mono" bind:value={amount} placeholder="0.00" required />
						</div>

						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="exp-tax">Included GST / Tax (₹)</label>
							<input id="exp-tax" type="number" step="0.01" min="0" class="form-input font-mono" bind:value={taxAmount} placeholder="0.00" />
						</div>
					</div>

					<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="exp-mode">Disbursement Method *</label>
							<select id="exp-mode" class="form-select" bind:value={paymentMode} required>
								{#each paymentModes as pm}
									<option value={pm.id}>{pm.name}</option>
								{/each}
							</select>
						</div>

						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="exp-date">Expense Date *</label>
							<input id="exp-date" type="date" class="form-input" bind:value={expenseDate} required />
						</div>
					</div>

					<div class="form-group" style="margin-bottom: 0;">
						<label class="form-label" for="exp-desc">Narration / Business Purpose</label>
						<input id="exp-desc" type="text" class="form-input" bind:value={description} placeholder="e.g. Monthly cloud server hosting bill for primary database" />
					</div>

					<div class="form-group" style="margin-bottom: 0;">
						<label class="form-label" for="exp-receipt">Receipt / Invoice Document URL</label>
						<input id="exp-receipt" type="url" class="form-input" bind:value={receiptUrl} placeholder="https://drive.google.com/... or scan link" />
					</div>
				</div>

				<div class="modal-footer">
					<button type="button" onclick={() => (showCreateModal = false)} class="btn btn-secondary">Cancel</button>
					<button type="submit" class="btn btn-primary" disabled={submitting}>
						{submitting ? 'Recording Expense...' : 'Record Expense Voucher'}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}
