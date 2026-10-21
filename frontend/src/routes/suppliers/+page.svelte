<script lang="ts">
	import { onMount } from 'svelte';
	import { api, type Supplier, type SupplierStatement, type Tenant } from '$lib/api';

	let suppliers = $state<Supplier[]>([]);
	let company = $state<Tenant | null>(null);
	let loading = $state(true);
	let error = $state('');

	// Search and filter
	let searchQuery = $state('');
	let filterMode = $state<'ALL' | 'DUE' | 'SETTLED'>('ALL');

	// Add Supplier Modal
	let showAddModal = $state(false);
	let newName = $state('');
	let newCompany = $state('');
	let newGSTIN = $state('');
	let newEmail = $state('');
	let newPhone = $state('');
	let newAddress = $state('');
	let newCity = $state('');
	let newState = $state('Maharashtra');
	let newPincode = $state('');
	let newPan = $state('');
	let newBankName = $state('');
	let newBankAccountNo = $state('');
	let newBankIFSC = $state('');
	let newOpeningBalance = $state('0.00');
	let modalError = $state('');
	let modalSubmitting = $state(false);

	// Statement Modal
	let showStatementModal = $state(false);
	let selectedSupplier = $state<Supplier | null>(null);
	let statement = $state<SupplierStatement | null>(null);
	let statementLoading = $state(false);
	let statementError = $state('');
	let fromDate = $state('');
	let toDate = $state('');

	onMount(async () => {
		await loadData();
	});

	async function loadData() {
		loading = true;
		error = '';
		try {
			const [supList, comp] = await Promise.all([
				api.listSuppliers(),
				api.getCompany().catch(() => null)
			]);
			suppliers = supList;
			company = comp;
			if (comp?.state) {
				newState = comp.state;
			}
		} catch (err: any) {
			error = err.message || 'Failed to load suppliers';
		} finally {
			loading = false;
		}
	}

	let filteredSuppliers = $derived(
		suppliers.filter((s) => {
			const q = searchQuery.toLowerCase().trim();
			const matchesQuery =
				!q ||
				s.name.toLowerCase().includes(q) ||
				(s.company_name && s.company_name.toLowerCase().includes(q)) ||
				(s.gstin && s.gstin.toLowerCase().includes(q)) ||
				(s.phone && s.phone.includes(q));

			const bal = parseFloat(s.outstanding_balance || '0');
			if (filterMode === 'DUE') return matchesQuery && bal > 0.01;
			if (filterMode === 'SETTLED') return matchesQuery && bal <= 0.01;
			return matchesQuery;
		})
	);

	let totalPayableSum = $derived(
		suppliers.reduce((acc, s) => acc + parseFloat(s.outstanding_balance || '0'), 0)
	);

	async function handleCreateSupplier(e: Event) {
		e.preventDefault();
		if (!newName.trim()) {
			modalError = 'Supplier name is required';
			return;
		}
		modalSubmitting = true;
		modalError = '';
		try {
			await api.createSupplier({
				name: newName.trim(),
				company_name: newCompany.trim(),
				gstin: newGSTIN.trim().toUpperCase(),
				email: newEmail.trim(),
				phone: newPhone.trim(),
				address: newAddress.trim(),
				city: newCity.trim(),
				state: newState.trim(),
				pincode: newPincode.trim(),
				pan: newPan.trim().toUpperCase(),
				bank_name: newBankName.trim(),
				bank_account_no: newBankAccountNo.trim(),
				bank_ifsc: newBankIFSC.trim().toUpperCase(),
				opening_balance: newOpeningBalance
			});
			showAddModal = false;
			resetForm();
			await loadData();
		} catch (err: any) {
			modalError = err.message || 'Failed to create supplier';
		} finally {
			modalSubmitting = false;
		}
	}

	function resetForm() {
		newName = '';
		newCompany = '';
		newGSTIN = '';
		newEmail = '';
		newPhone = '';
		newAddress = '';
		newCity = '';
		newPincode = '';
		newPan = '';
		newBankName = '';
		newBankAccountNo = '';
		newBankIFSC = '';
		newOpeningBalance = '0.00';
		modalError = '';
	}

	async function openStatement(sup: Supplier) {
		selectedSupplier = sup;
		showStatementModal = true;
		statementLoading = true;
		statementError = '';
		try {
			statement = await api.getSupplierStatement(sup.id, {
				from_date: fromDate || undefined,
				to_date: toDate || undefined
			});
		} catch (err: any) {
			statementError = err.message || 'Failed to load statement';
		} finally {
			statementLoading = false;
		}
	}

	async function reloadStatement() {
		if (!selectedSupplier) return;
		statementLoading = true;
		statementError = '';
		try {
			statement = await api.getSupplierStatement(selectedSupplier.id, {
				from_date: fromDate || undefined,
				to_date: toDate || undefined
			});
		} catch (err: any) {
			statementError = err.message || 'Failed to load statement';
		} finally {
			statementLoading = false;
		}
	}

	function formatCurrency(val: string | number | undefined) {
		const num = typeof val === 'number' ? val : parseFloat(val || '0');
		return new Intl.NumberFormat('en-IN', {
			style: 'currency',
			currency: 'INR',
			maximumFractionDigits: 2
		}).format(num);
	}
</script>

<div class="suppliers-view">
	<!-- Top Bar -->
	<header class="header-section">
		<div>
			<div class="subtitle-badge">VENDOR & ACCOUNTS PAYABLE</div>
			<h1 class="page-title">Suppliers & Khata Ledger</h1>
			<p class="page-desc">Track trade creditors, inward purchase bills, payments made, and live payables.</p>
		</div>
		<div class="action-buttons">
			<a href="/purchases/new" class="btn btn-secondary">
				<span class="icon">📥</span> New Inward Bill
			</a>
			<button class="btn btn-primary" onclick={() => (showAddModal = true)}>
				<span class="icon">+</span> Add Supplier
			</button>
		</div>
	</header>

	<!-- KPI Summary Bar -->
	<div class="kpi-grid">
		<div class="kpi-card">
			<span class="kpi-label">Total Suppliers</span>
			<span class="kpi-value">{suppliers.length}</span>
			<span class="kpi-subtext">Active vendors</span>
		</div>
		<div class="kpi-card highlight-card">
			<span class="kpi-label">Total Accounts Payable</span>
			<span class="kpi-value text-red">{formatCurrency(totalPayableSum)}</span>
			<span class="kpi-subtext">Money owed to suppliers</span>
		</div>
		<div class="kpi-card">
			<span class="kpi-label">Suppliers With Due</span>
			<span class="kpi-value text-yellow">
				{suppliers.filter((s) => parseFloat(s.outstanding_balance || '0') > 0.01).length}
			</span>
			<span class="kpi-subtext">Pending bill settlements</span>
		</div>
	</div>

	<!-- Controls & Search -->
	<div class="controls-bar">
		<div class="search-wrap">
			<span class="search-icon">🔍</span>
			<input
				type="text"
				placeholder="Search supplier by name, company, GSTIN or phone..."
				bind:value={searchQuery}
				class="search-input"
			/>
		</div>
		<div class="filters">
			<button
				class="filter-pill"
				class:active={filterMode === 'ALL'}
				onclick={() => (filterMode = 'ALL')}
			>
				All ({suppliers.length})
			</button>
			<button
				class="filter-pill"
				class:active={filterMode === 'DUE'}
				onclick={() => (filterMode = 'DUE')}
			>
				Balance Due ({suppliers.filter((s) => parseFloat(s.outstanding_balance || '0') > 0.01).length})
			</button>
			<button
				class="filter-pill"
				class:active={filterMode === 'SETTLED'}
				onclick={() => (filterMode = 'SETTLED')}
			>
				Settled ({suppliers.filter((s) => parseFloat(s.outstanding_balance || '0') <= 0.01).length})
			</button>
		</div>
	</div>

	{#if error}
		<div class="alert alert-error">{error}</div>
	{/if}

	<!-- Suppliers Directory Table -->
	<div class="table-container">
		{#if loading}
			<div class="loading-state">
				<div class="spinner"></div>
				<p>Loading suppliers and payables...</p>
			</div>
		{:else if filteredSuppliers.length === 0}
			<div class="empty-state">
				<span class="empty-icon">🏭</span>
				<h3>No suppliers found</h3>
				<p>Add your first vendor or trade distributor to start recording purchase bills.</p>
				<button class="btn btn-primary" onclick={() => (showAddModal = true)}>
					+ Add First Supplier
				</button>
			</div>
		{:else}
			<table class="data-table">
				<thead>
					<tr>
						<th>Supplier / Vendor</th>
						<th>Contact Details</th>
						<th>GSTIN & State</th>
						<th>Bank Info</th>
						<th style="text-align: right;">Opening Balance</th>
						<th style="text-align: right;">Net Payable</th>
						<th style="text-align: center;">Actions</th>
					</tr>
				</thead>
				<tbody>
					{#each filteredSuppliers as sup}
						<tr>
							<td>
								<div class="sup-name-cell">
									<strong>{sup.name}</strong>
									{#if sup.company_name}
										<span class="company-sub">{sup.company_name}</span>
									{/if}
								</div>
							</td>
							<td>
								<div class="contact-cell">
									{#if sup.phone}<span>📞 {sup.phone}</span>{/if}
									{#if sup.email}<span class="text-muted">✉️ {sup.email}</span>{/if}
								</div>
							</td>
							<td>
								<div class="gst-cell">
									{#if sup.gstin}
										<span class="badge-gst">{sup.gstin}</span>
									{:else}
										<span class="badge-unreg">Unregistered</span>
									{/if}
									<span class="state-label">{sup.city || ''} {sup.state || ''}</span>
								</div>
							</td>
							<td>
								<div class="bank-cell">
									{#if sup.bank_name}
										<span>{sup.bank_name}</span>
										<small class="text-muted">A/C: {sup.bank_account_no || 'N/A'}</small>
									{:else}
										<span class="text-muted">—</span>
									{/if}
								</div>
							</td>
							<td style="text-align: right; color: var(--text-muted);">
								{formatCurrency(sup.opening_balance)}
							</td>
							<td style="text-align: right;">
								{#if parseFloat(sup.outstanding_balance || '0') > 0.01}
									<span class="badge-payable">{formatCurrency(sup.outstanding_balance)}</span>
								{:else}
									<span class="badge-settled">Settled</span>
								{/if}
							</td>
							<td style="text-align: center;">
								<button class="btn btn-sm btn-outline" onclick={() => openStatement(sup)}>
									📖 Statement
								</button>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
	</div>
</div>

<!-- Add Supplier Modal -->
{#if showAddModal}
	<div class="modal-backdrop">
		<div class="modal-card">
			<div class="modal-header">
				<h3>Add New Supplier</h3>
				<button class="close-btn" onclick={() => (showAddModal = false)}>✕</button>
			</div>
			<form onsubmit={handleCreateSupplier}>
				<div class="modal-body">
					{#if modalError}
						<div class="alert alert-error">{modalError}</div>
					{/if}
					<div class="form-grid">
						<div class="form-group full-width">
							<label for="sup-name">Supplier / Vendor Name *</label>
							<input
								id="sup-name"
								type="text"
								bind:value={newName}
								placeholder="e.g. Apex Health & Pharma Distributors"
								required
							/>
						</div>
						<div class="form-group">
							<label for="sup-company">Company / Trade Name</label>
							<input
								id="sup-company"
								type="text"
								bind:value={newCompany}
								placeholder="e.g. Apex Enterprises Pvt Ltd"
							/>
						</div>
						<div class="form-group">
							<label for="sup-gstin">GSTIN (15 Digits)</label>
							<input
								id="sup-gstin"
								type="text"
								bind:value={newGSTIN}
								placeholder="e.g. 27AABCA1234F1Z9"
								maxlength="15"
							/>
						</div>
						<div class="form-group">
							<label for="sup-phone">Phone Number</label>
							<input
								id="sup-phone"
								type="text"
								bind:value={newPhone}
								placeholder="e.g. 9876543210"
							/>
						</div>
						<div class="form-group">
							<label for="sup-email">Email Address</label>
							<input
								id="sup-email"
								type="email"
								bind:value={newEmail}
								placeholder="e.g. vendor@supplier.com"
							/>
						</div>
						<div class="form-group full-width">
							<label for="sup-address">Billing Address</label>
							<input
								id="sup-address"
								type="text"
								bind:value={newAddress}
								placeholder="Street address, warehouse location"
							/>
						</div>
						<div class="form-group">
							<label for="sup-city">City</label>
							<input id="sup-city" type="text" bind:value={newCity} placeholder="e.g. Mumbai" />
						</div>
						<div class="form-group">
							<label for="sup-state">State (For GST Tax Rules)</label>
							<input id="sup-state" type="text" bind:value={newState} placeholder="e.g. Maharashtra" />
						</div>
						<div class="form-group">
							<label for="sup-pincode">Pincode</label>
							<input id="sup-pincode" type="text" bind:value={newPincode} placeholder="e.g. 400001" />
						</div>
						<div class="form-group">
							<label for="sup-openbal">Opening Balance (₹ Owed to Vendor)</label>
							<input
								id="sup-openbal"
								type="number"
								step="0.01"
								bind:value={newOpeningBalance}
								placeholder="0.00"
							/>
						</div>
						<div class="form-group">
							<label for="sup-bank">Bank Name</label>
							<input id="sup-bank" type="text" bind:value={newBankName} placeholder="e.g. HDFC Bank" />
						</div>
						<div class="form-group">
							<label for="sup-acct">Bank Account No</label>
							<input id="sup-acct" type="text" bind:value={newBankAccountNo} placeholder="e.g. 501000123456" />
						</div>
						<div class="form-group">
							<label for="sup-ifsc">Bank IFSC</label>
							<input id="sup-ifsc" type="text" bind:value={newBankIFSC} placeholder="e.g. HDFC0001234" />
						</div>
					</div>
				</div>
				<div class="modal-footer">
					<button type="button" class="btn btn-secondary" onclick={() => (showAddModal = false)}>
						Cancel
					</button>
					<button type="submit" class="btn btn-primary" disabled={modalSubmitting}>
						{modalSubmitting ? 'Saving...' : 'Save Supplier'}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}

<!-- Statement Modal -->
{#if showStatementModal && selectedSupplier}
	<div class="modal-backdrop">
		<div class="modal-card modal-lg">
			<div class="modal-header">
				<div>
					<h3>Supplier Khata Statement</h3>
					<span class="company-sub">{selectedSupplier.name} {selectedSupplier.company_name ? `(${selectedSupplier.company_name})` : ''}</span>
				</div>
				<div style="display: flex; gap: 0.5rem; align-items: center;">
					<button class="btn btn-sm btn-outline" onclick={() => window.print()}>🖨️ Print</button>
					<button class="close-btn" onclick={() => (showStatementModal = false)}>✕</button>
				</div>
			</div>
			<div class="modal-body">
				<!-- Date Filter -->
				<div class="stmt-filter-bar">
					<div class="date-group">
						<label for="stmt-from">From:</label>
						<input id="stmt-from" type="date" bind:value={fromDate} onchange={reloadStatement} />
					</div>
					<div class="date-group">
						<label for="stmt-to">To:</label>
						<input id="stmt-to" type="date" bind:value={toDate} onchange={reloadStatement} />
					</div>
					<button class="btn btn-sm btn-secondary" onclick={reloadStatement}>Apply</button>
				</div>

				{#if statementLoading}
					<div class="loading-state">
						<div class="spinner"></div>
						<p>Loading ledger entries...</p>
					</div>
				{:else if statementError}
					<div class="alert alert-error">{statementError}</div>
				{:else if statement}
					<!-- Summary Cards -->
					<div class="stmt-kpi-grid">
						<div class="stmt-kpi">
							<span class="lbl">Opening Balance</span>
							<span class="val">{formatCurrency(statement.opening_balance)}</span>
						</div>
						<div class="stmt-kpi">
							<span class="lbl">Purchases (Credit)</span>
							<span class="val text-blue">+{formatCurrency(statement.total_purchases)}</span>
						</div>
						<div class="stmt-kpi">
							<span class="lbl">Payments Made (Debit)</span>
							<span class="val text-green">-{formatCurrency(statement.total_payments)}</span>
						</div>
						<div class="stmt-kpi highlight">
							<span class="lbl">Closing Outstanding</span>
							<span class="val text-red">{formatCurrency(statement.closing_balance)}</span>
						</div>
					</div>

					<!-- Entries Ledger -->
					{#if statement.entries.length === 0}
						<div class="empty-state" style="padding: 2rem;">
							<p>No purchases or payments recorded in this period.</p>
						</div>
					{:else}
						<table class="data-table stmt-table">
							<thead>
								<tr>
									<th>Date</th>
									<th>Type</th>
									<th>Reference #</th>
									<th>Description</th>
									<th style="text-align: right;">Payment Made (Dr)</th>
									<th style="text-align: right;">Purchase Bill (Cr)</th>
									<th style="text-align: right;">Balance</th>
								</tr>
							</thead>
							<tbody>
								{#each statement.entries as entry}
									<tr>
										<td>{new Date(entry.date).toLocaleDateString('en-IN')}</td>
										<td>
											{#if entry.type === 'PURCHASE'}
												<span class="badge-purchase">PURCHASE</span>
											{:else}
												<span class="badge-payment">PAYMENT</span>
											{/if}
										</td>
										<td><strong>{entry.reference}</strong></td>
										<td class="text-muted">{entry.description}</td>
										<td style="text-align: right; color: #22c55e;">
											{parseFloat(entry.debit) > 0 ? formatCurrency(entry.debit) : '—'}
										</td>
										<td style="text-align: right; color: #3b82f6;">
											{parseFloat(entry.credit) > 0 ? formatCurrency(entry.credit) : '—'}
										</td>
										<td style="text-align: right; font-weight: 600;">
											{formatCurrency(entry.running_balance)}
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					{/if}
				{/if}
			</div>
		</div>
	</div>
{/if}

<style>
	.suppliers-view {
		padding: 2rem;
		max-width: 1400px;
		margin: 0 auto;
		display: flex;
		flex-direction: column;
		gap: 1.5rem;
	}

	.header-section {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		flex-wrap: wrap;
		gap: 1rem;
	}

	.subtitle-badge {
		font-size: 0.75rem;
		font-weight: 700;
		color: #6366f1;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		margin-bottom: 0.25rem;
	}

	.page-title {
		font-size: 1.75rem;
		font-weight: 700;
		color: var(--text-primary, #f8fafc);
		margin: 0;
	}

	.page-desc {
		font-size: 0.875rem;
		color: var(--text-muted, #94a3b8);
		margin-top: 0.25rem;
	}

	.action-buttons {
		display: flex;
		gap: 0.75rem;
	}

	.btn {
		display: inline-flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.625rem 1.25rem;
		border-radius: 8px;
		font-size: 0.875rem;
		font-weight: 600;
		cursor: pointer;
		border: none;
		text-decoration: none;
		transition: all 0.15s ease;
	}

	.btn-primary {
		background: #6366f1;
		color: #ffffff;
	}
	.btn-primary:hover {
		background: #4f46e5;
	}

	.btn-secondary {
		background: rgba(255, 255, 255, 0.08);
		color: #e2e8f0;
		border: 1px solid rgba(255, 255, 255, 0.15);
	}
	.btn-secondary:hover {
		background: rgba(255, 255, 255, 0.14);
	}

	.btn-outline {
		background: transparent;
		color: #a5b4fc;
		border: 1px solid rgba(165, 180, 252, 0.3);
	}
	.btn-outline:hover {
		background: rgba(165, 180, 252, 0.1);
	}

	.btn-sm {
		padding: 0.375rem 0.75rem;
		font-size: 0.75rem;
	}

	/* KPI Grid */
	.kpi-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
		gap: 1rem;
	}

	.kpi-card {
		background: rgba(30, 41, 59, 0.6);
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 12px;
		padding: 1.25rem;
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}

	.highlight-card {
		background: linear-gradient(135deg, rgba(99, 102, 241, 0.15), rgba(239, 68, 68, 0.1));
		border-color: rgba(99, 102, 241, 0.3);
	}

	.kpi-label {
		font-size: 0.8125rem;
		color: #94a3b8;
		font-weight: 500;
	}

	.kpi-value {
		font-size: 1.625rem;
		font-weight: 700;
		color: #f8fafc;
	}

	.kpi-subtext {
		font-size: 0.75rem;
		color: #64748b;
	}

	.text-red {
		color: #f87171;
	}
	.text-yellow {
		color: #fbbf24;
	}
	.text-blue {
		color: #60a5fa;
	}
	.text-green {
		color: #4ade80;
	}
	.text-muted {
		color: #94a3b8;
	}

	/* Controls */
	.controls-bar {
		display: flex;
		justify-content: space-between;
		align-items: center;
		flex-wrap: wrap;
		gap: 1rem;
	}

	.search-wrap {
		position: relative;
		flex: 1;
		min-width: 280px;
	}

	.search-icon {
		position: absolute;
		left: 0.875rem;
		top: 50%;
		transform: translateY(-50%);
		font-size: 0.875rem;
		color: #64748b;
	}

	.search-input {
		width: 100%;
		padding: 0.625rem 0.875rem 0.625rem 2.5rem;
		background: rgba(15, 23, 42, 0.6);
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: 8px;
		color: #f8fafc;
		font-size: 0.875rem;
		outline: none;
	}
	.search-input:focus {
		border-color: #6366f1;
	}

	.filters {
		display: flex;
		gap: 0.5rem;
	}

	.filter-pill {
		padding: 0.375rem 0.875rem;
		border-radius: 20px;
		font-size: 0.8125rem;
		background: rgba(255, 255, 255, 0.05);
		border: 1px solid rgba(255, 255, 255, 0.1);
		color: #94a3b8;
		cursor: pointer;
		transition: all 0.15s;
	}

	.filter-pill.active {
		background: #6366f1;
		color: #ffffff;
		border-color: #6366f1;
	}

	/* Table */
	.table-container {
		background: rgba(30, 41, 59, 0.5);
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 12px;
		overflow-x: auto;
	}

	.data-table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.875rem;
		text-align: left;
	}

	.data-table th {
		background: rgba(15, 23, 42, 0.6);
		padding: 0.875rem 1rem;
		font-weight: 600;
		color: #94a3b8;
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
	}

	.data-table td {
		padding: 1rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.05);
		vertical-align: middle;
	}

	.sup-name-cell {
		display: flex;
		flex-direction: column;
		gap: 0.125rem;
	}

	.company-sub {
		font-size: 0.75rem;
		color: #94a3b8;
	}

	.contact-cell,
	.gst-cell,
	.bank-cell {
		display: flex;
		flex-direction: column;
		gap: 0.125rem;
		font-size: 0.8125rem;
	}

	.badge-gst {
		background: rgba(59, 130, 246, 0.15);
		color: #93c5fd;
		padding: 0.125rem 0.375rem;
		border-radius: 4px;
		font-family: monospace;
		font-size: 0.75rem;
		width: fit-content;
	}

	.badge-unreg {
		color: #64748b;
		font-size: 0.75rem;
	}

	.state-label {
		font-size: 0.75rem;
		color: #64748b;
	}

	.badge-payable {
		background: rgba(239, 68, 68, 0.15);
		color: #f87171;
		font-weight: 700;
		padding: 0.25rem 0.5rem;
		border-radius: 6px;
		font-size: 0.875rem;
	}

	.badge-settled {
		color: #4ade80;
		font-size: 0.8125rem;
		font-weight: 600;
	}

	.badge-purchase {
		background: rgba(59, 130, 246, 0.15);
		color: #60a5fa;
		font-size: 0.6875rem;
		font-weight: 700;
		padding: 0.125rem 0.375rem;
		border-radius: 4px;
	}

	.badge-payment {
		background: rgba(34, 197, 94, 0.15);
		color: #4ade80;
		font-size: 0.6875rem;
		font-weight: 700;
		padding: 0.125rem 0.375rem;
		border-radius: 4px;
	}

	/* Modal */
	.modal-backdrop {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.75);
		backdrop-filter: blur(4px);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 100;
		padding: 1rem;
	}

	.modal-card {
		background: #1e293b;
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: 12px;
		width: 100%;
		max-width: 650px;
		max-height: 90vh;
		overflow-y: auto;
		display: flex;
		flex-direction: column;
		box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
	}

	.modal-lg {
		max-width: 900px;
	}

	.modal-header {
		padding: 1.25rem 1.5rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
		display: flex;
		justify-content: space-between;
		align-items: center;
	}

	.modal-header h3 {
		margin: 0;
		font-size: 1.125rem;
		color: #f8fafc;
	}

	.close-btn {
		background: transparent;
		border: none;
		color: #94a3b8;
		font-size: 1.25rem;
		cursor: pointer;
	}

	.modal-body {
		padding: 1.5rem;
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}

	.modal-footer {
		padding: 1rem 1.5rem;
		border-top: 1px solid rgba(255, 255, 255, 0.08);
		display: flex;
		justify-content: flex-end;
		gap: 0.75rem;
	}

	.form-grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 1rem;
	}

	.full-width {
		grid-column: 1 / -1;
	}

	.form-group {
		display: flex;
		flex-direction: column;
		gap: 0.375rem;
	}

	.form-group label {
		font-size: 0.8125rem;
		font-weight: 500;
		color: #cbd5e1;
	}

	.form-group input {
		padding: 0.5rem 0.75rem;
		background: rgba(15, 23, 42, 0.6);
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: 6px;
		color: #f8fafc;
		font-size: 0.875rem;
		outline: none;
	}
	.form-group input:focus {
		border-color: #6366f1;
	}

	.alert-error {
		background: rgba(239, 68, 68, 0.15);
		border: 1px solid rgba(239, 68, 68, 0.3);
		color: #f87171;
		padding: 0.75rem;
		border-radius: 6px;
		font-size: 0.875rem;
	}

	/* Statement KPI */
	.stmt-filter-bar {
		display: flex;
		gap: 1rem;
		align-items: center;
		padding: 0.75rem;
		background: rgba(15, 23, 42, 0.5);
		border-radius: 8px;
	}

	.date-group {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 0.8125rem;
		color: #cbd5e1;
	}

	.date-group input {
		padding: 0.25rem 0.5rem;
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.1);
		color: #f8fafc;
		border-radius: 4px;
	}

	.stmt-kpi-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
		gap: 0.75rem;
	}

	.stmt-kpi {
		background: rgba(15, 23, 42, 0.6);
		padding: 0.875rem;
		border-radius: 8px;
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}

	.stmt-kpi.highlight {
		border: 1px solid rgba(239, 68, 68, 0.3);
		background: rgba(239, 68, 68, 0.08);
	}

	.stmt-kpi .lbl {
		font-size: 0.75rem;
		color: #94a3b8;
	}

	.stmt-kpi .val {
		font-size: 1.125rem;
		font-weight: 700;
		color: #f8fafc;
	}

	.loading-state,
	.empty-state {
		padding: 3rem;
		text-align: center;
		color: #94a3b8;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.75rem;
	}

	.empty-icon {
		font-size: 2.5rem;
	}

	.spinner {
		width: 32px;
		height: 32px;
		border: 3px solid rgba(99, 102, 241, 0.2);
		border-top-color: #6366f1;
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}
</style>
