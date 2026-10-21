<script lang="ts">
	import { onMount } from 'svelte';
	import {
		api,
		type CustomerWithBalance,
		type CustomerStatement,
		type Tenant,
		type Invoice,
		type Payment
	} from '$lib/api';

	// Primary View State
	let activeTab = $state<'customers' | 'all-history'>('customers');
	let customers = $state<CustomerWithBalance[]>([]);
	let allInvoices = $state<Invoice[]>([]);
	let allPayments = $state<Payment[]>([]);
	let company = $state<Tenant | null>(null);
	let loading = $state(true);
	let error = $state('');

	// Add Customer Modal State
	let showAddModal = $state(false);
	let newName = $state('');
	let newCompany = $state('');
	let newGSTIN = $state('');
	let newEmail = $state('');
	let newPhone = $state('');
	let newBillingAddress = $state('');
	let newBillingCity = $state('');
	let newBillingState = $state('Maharashtra');
	let newBillingPincode = $state('');
	let modalError = $state('');
	let modalSubmitting = $state(false);

	// Customer Search & Filter (Tab 1)
	let customerSearch = $state('');
	let balanceFilter = $state<'ALL' | 'DUE' | 'SETTLED'>('ALL');

	// Domain Vertical State & Persistence
	type CustomerDomainCategory = 'ALL' | 'TRAVELERS' | 'PATIENTS' | 'CORPORATE';
	let domainCategory = $state<CustomerDomainCategory>('ALL');

	function syncDomainFromStorage() {
		if (typeof window === 'undefined') return;
		const unit = (localStorage.getItem('mrbiz_active_unit') || '').toLowerCase();
		if (unit === 'travel') {
			domainCategory = 'TRAVELERS';
		} else if (unit === 'healthcare') {
			domainCategory = 'PATIENTS';
		} else if (unit === 'consulting' || unit === 'quarry' || unit === 'ecommerce' || unit === 'realestate') {
			domainCategory = 'CORPORATE';
		}
	}

	// All Transactions History Filters (Tab 2)
	let historySearch = $state('');
	let historyType = $state<'ALL' | 'INVOICE' | 'PAYMENT'>('ALL');
	let historyCustomerId = $state('');
	let historyFromDate = $state('');
	let historyToDate = $state('');

	// Customer Statement Modal State
	let showStatementModal = $state(false);
	let selectedCustomer = $state<CustomerWithBalance | null>(null);
	let statementData = $state<CustomerStatement | null>(null);
	let statementLoading = $state(false);
	let statementError = $state('');
	let stmtFromDate = $state('');
	let stmtToDate = $state('');

	// Print Consolidated Ledger Modal State
	let showPrintAllModal = $state(false);

	// Transaction Record definition for Consolidated Feed
	interface ConsolidatedTx {
		id: string;
		date: string;
		rawDate: number;
		customerId: string;
		customerName: string;
		customerGSTIN: string;
		type: 'INVOICE' | 'PAYMENT';
		voucherNumber: string;
		statusOrMethod: string;
		narration: string;
		debit: number;
		credit: number;
	}

	// Load Data
	onMount(() => {
		syncDomainFromStorage();
		const handleUnitChange = (e: any) => {
			const unit = (e?.detail?.unitId || localStorage.getItem('mrbiz_active_unit') || '').toLowerCase();
			if (unit === 'travel') domainCategory = 'TRAVELERS';
			else if (unit === 'healthcare') domainCategory = 'PATIENTS';
			else if (unit === 'consulting' || unit === 'quarry' || unit === 'ecommerce' || unit === 'realestate') domainCategory = 'CORPORATE';
		};
		window.addEventListener('mrbiz_unit_changed', handleUnitChange);
		loadAllData();
		return () => {
			window.removeEventListener('mrbiz_unit_changed', handleUnitChange);
		};
	});

	// Domain Attribute Enrichment Helpers
	function getTravelerDetails(cust: CustomerWithBalance) {
		const seed = (cust.id || cust.name).split('').reduce((acc, c) => acc + c.charCodeAt(0), 0);
		const passportNo = 'P' + (1000000 + (seed * 97) % 8999999);
		const months = ['Mar', 'Jun', 'Aug', 'Oct', 'Dec'];
		const expiry = `${months[seed % months.length]} 202${7 + (seed % 4)}`;
		const visas = [
			{ label: 'Schengen Multi-Entry', status: 'VALID', color: '#34d399' },
			{ label: 'Dubai E-Visa 30D', status: 'APPROVED', color: '#38bdf8' },
			{ label: 'US B1/B2 10Y', status: 'VALID', color: '#a855f7' },
			{ label: 'UK Standard Visitor', status: 'ISSUED', color: '#34d399' },
			{ label: 'Bali VoA Cleared', status: 'ISSUED', color: '#34d399' }
		];
		const pnrCodes = ['EK-7241', 'LH-4910', 'AI-802', 'QR-918', 'LX-208'];
		const destinations = ['Zurich & Swiss Alps', 'Dubai & Abu Dhabi', 'Bali & Ubud', 'London & Paris', 'Tokyo & Kyoto'];
		const visa = visas[seed % visas.length];
		const pnr = pnrCodes[seed % pnrCodes.length];
		const destination = destinations[seed % destinations.length];
		return { passportNo, expiry, visa, pnr, destination };
	}

	function getPatientDetails(cust: CustomerWithBalance) {
		const seed = (cust.id || cust.name).split('').reduce((acc, c) => acc + c.charCodeAt(0), 0);
		const uhid = `UHID-MED-${10000 + (seed % 89999)}`;
		const age = 22 + (seed % 56);
		const gender = seed % 2 === 0 ? 'M' : 'F';
		const doctors = [
			{ name: 'Dr. Sneha Roy', dept: 'Cardiology OPD' },
			{ name: 'Dr. V. Mehta', dept: 'General Medicine' },
			{ name: 'Dr. Ananya Sen', dept: 'Endocrinology' },
			{ name: 'Dr. R. Kulkarni', dept: 'Orthopedics' },
			{ name: 'Dr. K. Sharma', dept: 'Pulmonology' }
		];
		const doc = doctors[seed % doctors.length];
		const categories = ['Outpatient (OPD)', 'Day Care Infusion', 'Chronic Care Plan', 'Pharmacy Rx Only'];
		const careCategory = categories[seed % categories.length];
		return { uhid, age, gender, doctor: doc.name, dept: doc.dept, careCategory };
	}

	async function loadAllData() {
		loading = true;
		error = '';
		try {
			const [custData, invData, payData, compData] = await Promise.all([
				api.listCustomersWithBalances().catch(() => []),
				api.listInvoices().catch(() => []),
				api.listPayments().catch(() => []),
				api.getCompany().catch(() => null)
			]);
			customers = custData;
			allInvoices = invData;
			allPayments = payData;
			company = compData;
		} catch (err: any) {
			error = err.message || 'Failed to load customer credit/debit records';
		} finally {
			loading = false;
		}
	}

	function formatCurrency(val: string | number) {
		const num = typeof val === 'string' ? parseFloat(val) : val;
		if (isNaN(num)) return '₹0.00';
		return '₹' + num.toLocaleString('en-IN', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
	}

	function formatDateStr(dateStr: string) {
		if (!dateStr) return '-';
		try {
			const d = new Date(dateStr);
			if (isNaN(d.getTime())) return dateStr;
			return d.toLocaleDateString('en-IN', { day: '2-digit', month: 'short', year: 'numeric' });
		} catch {
			return dateStr;
		}
	}

	// Overall KPIs
	let totalInvoicedDebit = $derived(
		customers.reduce((sum, c) => sum + (parseFloat(c.total_invoiced) || 0), 0)
	);
	let totalPaidCredit = $derived(
		customers.reduce((sum, c) => sum + (parseFloat(c.total_paid) || 0), 0)
	);
	let totalOutstandingDue = $derived(
		customers.reduce((sum, c) => sum + (parseFloat(c.outstanding_balance) || 0), 0)
	);

	// Customer Map for fast lookup
	let customerMap = $derived(
		new Map(customers.map((c) => [c.id, c]))
	);

	// Filtered Customers (Tab 1)
	let filteredCustomers = $derived(
		customers.filter((c) => {
			const q = customerSearch.trim().toLowerCase();
			let matchesQuery =
				!q ||
				c.name.toLowerCase().includes(q) ||
				(c.company_name && c.company_name.toLowerCase().includes(q)) ||
				(c.gstin && c.gstin.toLowerCase().includes(q)) ||
				(c.phone && c.phone.includes(q)) ||
				(c.email && c.email.toLowerCase().includes(q));

			if (!matchesQuery && q) {
				if (domainCategory === 'TRAVELERS') {
					const t = getTravelerDetails(c);
					matchesQuery =
						t.passportNo.toLowerCase().includes(q) ||
						t.pnr.toLowerCase().includes(q) ||
						t.destination.toLowerCase().includes(q) ||
						t.visa.label.toLowerCase().includes(q);
				} else if (domainCategory === 'PATIENTS') {
					const p = getPatientDetails(c);
					matchesQuery =
						p.uhid.toLowerCase().includes(q) ||
						p.doctor.toLowerCase().includes(q) ||
						p.dept.toLowerCase().includes(q);
				}
			}

			const bal = parseFloat(c.outstanding_balance) || 0;
			const matchesBal =
				balanceFilter === 'ALL' ||
				(balanceFilter === 'DUE' && bal > 0.01) ||
				(balanceFilter === 'SETTLED' && bal <= 0.01);

			return matchesQuery && matchesBal;
		})
	);

	// Consolidated Unified Transaction Feed (Tab 2)
	let allTransactions = $derived.by(() => {
		const list: ConsolidatedTx[] = [];

		// Map Invoices (Debits billed to customer)
		for (const inv of allInvoices) {
			if (inv.status === 'CANCELLED') continue;
			const d = new Date(inv.invoice_date);
			list.push({
				id: inv.id,
				date: inv.invoice_date,
				rawDate: isNaN(d.getTime()) ? 0 : d.getTime(),
				customerId: inv.customer_id,
				customerName: inv.customer_name || customerMap.get(inv.customer_id)?.name || 'Client',
				customerGSTIN: inv.customer_gstin || customerMap.get(inv.customer_id)?.gstin || '',
				type: 'INVOICE',
				voucherNumber: inv.invoice_number,
				statusOrMethod: inv.status,
				narration: `Tax Invoice #${inv.invoice_number}${inv.notes ? ' • ' + inv.notes : ''}`,
				debit: parseFloat(inv.grand_total) || 0,
				credit: 0
			});
		}

		// Map Payments (Credits received from customer)
		for (const pay of allPayments) {
			const d = new Date(pay.payment_date);
			const cust = customerMap.get(pay.customer_id);
			list.push({
				id: pay.id,
				date: pay.payment_date,
				rawDate: isNaN(d.getTime()) ? 0 : d.getTime(),
				customerId: pay.customer_id,
				customerName: cust ? cust.name : 'Client',
				customerGSTIN: (cust && cust.gstin) || '',
				type: 'PAYMENT',
				voucherNumber: pay.payment_number,
				statusOrMethod: pay.payment_method,
				narration: `Payment via ${pay.payment_method}${pay.reference_number ? ' (Ref: ' + pay.reference_number + ')' : ''}${pay.notes ? ' • ' + pay.notes : ''}`,
				debit: 0,
				credit: parseFloat(pay.amount) || 0
			});
		}

		// Sort chronologically descending (newest first)
		return list.sort((a, b) => b.rawDate - a.rawDate);
	});

	// Filtered Transactions Feed (Tab 2)
	let filteredTransactions = $derived(
		allTransactions.filter((tx) => {
			const q = historySearch.trim().toLowerCase();
			const matchesQuery =
				!q ||
				tx.customerName.toLowerCase().includes(q) ||
				tx.voucherNumber.toLowerCase().includes(q) ||
				tx.narration.toLowerCase().includes(q) ||
				tx.customerGSTIN.toLowerCase().includes(q);

			const matchesType = historyType === 'ALL' || tx.type === historyType;
			const matchesCustomer = !historyCustomerId || tx.customerId === historyCustomerId;

			let matchesDate = true;
			if (historyFromDate) {
				matchesDate = matchesDate && tx.date >= historyFromDate;
			}
			if (historyToDate) {
				matchesDate = matchesDate && tx.date <= historyToDate;
			}

			return matchesQuery && matchesType && matchesCustomer && matchesDate;
		})
	);

	let filteredDebitTotal = $derived(
		filteredTransactions.reduce((sum, tx) => sum + tx.debit, 0)
	);
	let filteredCreditTotal = $derived(
		filteredTransactions.reduce((sum, tx) => sum + tx.credit, 0)
	);
	let filteredNetBalance = $derived(filteredDebitTotal - filteredCreditTotal);

	// Customer Statement Modal Actions
	async function openCustomerLedger(c: CustomerWithBalance) {
		selectedCustomer = c;
		statementData = null;
		statementError = '';
		stmtFromDate = '';
		stmtToDate = '';
		showStatementModal = true;
		await loadStatement();
	}

	async function loadStatement() {
		if (!selectedCustomer) return;
		statementLoading = true;
		statementError = '';
		try {
			statementData = await api.getCustomerStatement(selectedCustomer.id, {
				from_date: stmtFromDate || undefined,
				to_date: stmtToDate || undefined
			});
		} catch (err: any) {
			statementError = err.message || 'Failed to load customer statement';
		} finally {
			statementLoading = false;
		}
	}

	function setQuickDatePreset(days: number) {
		const now = new Date();
		stmtToDate = now.toISOString().split('T')[0];
		if (days > 0) {
			const past = new Date();
			past.setDate(past.getDate() - days);
			stmtFromDate = past.toISOString().split('T')[0];
		} else {
			stmtFromDate = '';
		}
		loadStatement();
	}

	function setHistoryDatePreset(days: number) {
		const now = new Date();
		historyToDate = now.toISOString().split('T')[0];
		if (days > 0) {
			const past = new Date();
			past.setDate(past.getDate() - days);
			historyFromDate = past.toISOString().split('T')[0];
		} else {
			historyFromDate = '';
		}
	}

	// CSV Export helpers
	function exportStatementCSV() {
		if (!statementData || !selectedCustomer) return;
		const rows = [
			['Customer Statement of Account'],
			['Customer Name', selectedCustomer.name],
			['Company', selectedCustomer.company_name || ''],
			['GSTIN', selectedCustomer.gstin || ''],
			['Period', `${stmtFromDate || 'Beginning'} to ${stmtToDate || 'Current'}`],
			['Opening Balance', statementData.opening_balance],
			['Total Invoiced', statementData.total_invoiced],
			['Total Received', statementData.total_paid],
			['Closing Balance', statementData.closing_balance],
			[],
			['Date', 'Type', 'Voucher Number', 'Reference', 'Narration', 'Debit (Rs)', 'Credit (Rs)', 'Balance (Rs)']
		];

		for (const item of (statementData.items || [])) {
			rows.push([
				item.date,
				item.voucher_type,
				item.voucher_number,
				item.reference_number,
				`"${item.narration.replace(/"/g, '""')}"`,
				item.debit_amount,
				item.credit_amount,
				item.balance
			]);
		}

		const csvContent = 'data:text/csv;charset=utf-8,' + rows.map((e) => e.join(',')).join('\n');
		const encodedUri = encodeURI(csvContent);
		const link = document.createElement('a');
		link.setAttribute('href', encodedUri);
		link.setAttribute('download', `Statement_${selectedCustomer.name.replace(/\s+/g, '_')}.csv`);
		document.body.appendChild(link);
		link.click();
		document.body.removeChild(link);
	}

	function exportAllTransactionsCSV() {
		const rows = [
			['All Customer Credit & Debit Transaction History'],
			['Generated Date', new Date().toLocaleString('en-IN')],
			['Total Invoiced (Debits)', filteredDebitTotal.toFixed(2)],
			['Total Collected (Credits)', filteredCreditTotal.toFixed(2)],
			['Net Balance', filteredNetBalance.toFixed(2)],
			[],
			['Date', 'Customer Name', 'GSTIN', 'Type', 'Voucher Number', 'Status/Mode', 'Narration', 'Debit (Rs)', 'Credit (Rs)']
		];

		for (const tx of filteredTransactions) {
			rows.push([
				tx.date,
				`"${tx.customerName.replace(/"/g, '""')}"`,
				tx.customerGSTIN,
				tx.type,
				tx.voucherNumber,
				tx.statusOrMethod,
				`"${tx.narration.replace(/"/g, '""')}"`,
				tx.debit.toFixed(2),
				tx.credit.toFixed(2)
			]);
		}

		const csvContent = 'data:text/csv;charset=utf-8,' + rows.map((e) => e.join(',')).join('\n');
		const encodedUri = encodeURI(csvContent);
		const link = document.createElement('a');
		link.setAttribute('href', encodedUri);
		link.setAttribute('download', `All_Customer_Credit_Debit_History.csv`);
		document.body.appendChild(link);
		link.click();
		document.body.removeChild(link);
	}

	// Trigger Print
	function triggerPrint() {
		window.print();
	}

	async function handleCreateCustomer(e: Event) {
		e.preventDefault();
		modalError = '';
		const trimmedName = newName.trim();
		const trimmedState = newBillingState.trim();
		const trimmedGSTIN = newGSTIN.trim().toUpperCase();

		if (!trimmedName || !trimmedState) {
			modalError = 'Customer name and billing state are required';
			return;
		}
		if (trimmedGSTIN && trimmedGSTIN.length !== 15) {
			modalError = 'Indian GSTIN must be exactly 15 characters';
			return;
		}

		modalSubmitting = true;
		try {
			await api.createCustomer({
				name: trimmedName,
				company_name: newCompany.trim(),
				gstin: trimmedGSTIN,
				email: newEmail.trim(),
				phone: newPhone.trim(),
				billing_address: newBillingAddress.trim(),
				billing_city: newBillingCity.trim(),
				billing_state: trimmedState,
				billing_pincode: newBillingPincode.trim()
			});
			showAddModal = false;
			newName = '';
			newCompany = '';
			newGSTIN = '';
			newEmail = '';
			newPhone = '';
			newBillingAddress = '';
			await loadAllData();
		} catch (err: any) {
			modalError = err.message || 'Failed to create customer';
		} finally {
			modalSubmitting = false;
		}
	}
</script>

<svelte:head>
	<title>Customers & Credit/Debit Ledger - SaaS Billing</title>
</svelte:head>

<div class="customer-page-container {showStatementModal || showPrintAllModal ? 'print-hidden' : ''}" style="display: flex; flex-direction: column; gap: 1.5rem;">
	<!-- Top Navigation & Page Header -->
	<div style="display: flex; justify-content: space-between; align-items: flex-start; flex-wrap: wrap; gap: 1rem;">
		<div>
			<div style="display: flex; align-items: center; gap: 0.6rem; margin-bottom: 0.25rem;">
				<h1 style="font-size: 1.75rem; margin: 0; font-family: var(--font-heading);">
					Customers & Credit/Debit Ledger
				</h1>
				<span style="font-size: 0.75rem; font-weight: 700; background: rgba(16, 185, 129, 0.15); color: #34d399; padding: 0.2rem 0.6rem; border-radius: 9999px; border: 1px solid rgba(16, 185, 129, 0.3);">
					🌱 Multi-Tenant Ledger
				</span>
			</div>
			<p style="font-size: 0.875rem; color: var(--text-secondary); margin: 0;">
				Track live customer balances, credit/debit history, tax invoices, payment receipts, and print official statements.
			</p>
		</div>

		<div style="display: flex; align-items: center; gap: 0.75rem; flex-wrap: wrap;">
			<button onclick={loadAllData} class="btn btn-secondary btn-sm" disabled={loading}>
				{loading ? 'Refreshing...' : '🔄 Refresh Data'}
			</button>
			<button onclick={() => (showPrintAllModal = true)} class="btn btn-secondary btn-sm" title="Print full customer balance ledger">
				🖨️ Print Balances Summary
			</button>
			<button onclick={() => (showAddModal = true)} class="btn btn-primary btn-sm">
				+ Add Customer
			</button>
		</div>
	</div>

	<!-- High-Level Financial KPI Overview Cards -->
	<div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(240px, 1fr)); gap: 1.25rem;">
		<div class="card" style="padding: 1.25rem; display: flex; flex-direction: column; gap: 0.4rem; border-left: 4px solid #38bdf8;">
			<div style="display: flex; justify-content: space-between; align-items: center;">
				<span style="font-size: 0.75rem; font-weight: 700; color: var(--text-muted); text-transform: uppercase; letter-spacing: 0.05em;">
					Total Billed (Debits)
				</span>
				<span style="font-size: 0.9rem;">📈</span>
			</div>
			<span style="font-size: 1.6rem; font-weight: 800; color: #fff; font-family: var(--font-heading);">
				{formatCurrency(totalInvoicedDebit)}
			</span>
			<span style="font-size: 0.75rem; color: var(--text-secondary);">
				Total value of tax invoices billed to customers
			</span>
		</div>

		<div class="card" style="padding: 1.25rem; display: flex; flex-direction: column; gap: 0.4rem; border-left: 4px solid #34d399;">
			<div style="display: flex; justify-content: space-between; align-items: center;">
				<span style="font-size: 0.75rem; font-weight: 700; color: var(--text-muted); text-transform: uppercase; letter-spacing: 0.05em;">
					Total Collected (Credits)
				</span>
				<span style="font-size: 0.9rem;">💰</span>
			</div>
			<span style="font-size: 1.6rem; font-weight: 800; color: #34d399; font-family: var(--font-heading);">
				{formatCurrency(totalPaidCredit)}
			</span>
			<span style="font-size: 0.75rem; color: var(--text-secondary);">
				Total payments received across client accounts
			</span>
		</div>

		<div class="card" style="padding: 1.25rem; display: flex; flex-direction: column; gap: 0.4rem; border-left: 4px solid {totalOutstandingDue > 0 ? '#f43f5e' : '#34d399'};">
			<div style="display: flex; justify-content: space-between; align-items: center;">
				<span style="font-size: 0.75rem; font-weight: 700; color: var(--text-muted); text-transform: uppercase; letter-spacing: 0.05em;">
					Net Receivables Due
				</span>
				<span style="font-size: 0.9rem;">⏳</span>
			</div>
			<span style="font-size: 1.6rem; font-weight: 800; color: {totalOutstandingDue > 0 ? '#fb7185' : '#34d399'}; font-family: var(--font-heading);">
				{formatCurrency(totalOutstandingDue)}
			</span>
			<span style="font-size: 0.75rem; color: var(--text-secondary);">
				Total active unpaid balance pending recovery
			</span>
		</div>

		<div class="card" style="padding: 1.25rem; display: flex; flex-direction: column; gap: 0.4rem; border-left: 4px solid #a855f7;">
			<div style="display: flex; justify-content: space-between; align-items: center;">
				<span style="font-size: 0.75rem; font-weight: 700; color: var(--text-muted); text-transform: uppercase; letter-spacing: 0.05em;">
					Active Clients & Volume
				</span>
				<span style="font-size: 0.9rem;">👥</span>
			</div>
			<span style="font-size: 1.6rem; font-weight: 800; color: #c084fc; font-family: var(--font-heading);">
				{customers.length} Clients
			</span>
			<span style="font-size: 0.75rem; color: var(--text-secondary);">
				{allTransactions.length} recorded credit & debit vouchers
			</span>
		</div>
	</div>

	<!-- Navigation Tabs Switcher -->
	<div style="display: flex; align-items: center; justify-content: space-between; border-bottom: 1px solid var(--border); padding-bottom: 0.25rem; flex-wrap: wrap; gap: 1rem;">
		<div style="display: flex; gap: 0.5rem;">
			<button
				onclick={() => (activeTab = 'customers')}
				class="tab-btn {activeTab === 'customers' ? 'active' : ''}"
			>
				👥 Customer Balances Directory ({customers.length})
			</button>
			<button
				onclick={() => (activeTab = 'all-history')}
				class="tab-btn {activeTab === 'all-history' ? 'active' : ''}"
			>
				📜 All Credit & Debit History ({allTransactions.length})
			</button>
		</div>

		{#if activeTab === 'all-history'}
			<button onclick={exportAllTransactionsCSV} class="btn btn-secondary btn-sm" style="font-size: 0.8rem;">
				📥 Export History to CSV
			</button>
		{/if}
	</div>

	<!-- TAB 1: CUSTOMER BALANCES DIRECTORY -->
	{#if activeTab === 'customers'}
		<div style="display: flex; flex-direction: column; gap: 1rem;">
			<!-- Domain Vertical Tab Strip -->
			<div style="display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 0.75rem; background: rgba(15, 23, 42, 0.4); padding: 0.5rem 0.75rem; border-radius: var(--radius-md); border: 1px solid var(--border);">
				<div style="display: flex; gap: 0.4rem; align-items: center; flex-wrap: wrap;">
					<button
						type="button"
						class="domain-tab-pill {domainCategory === 'ALL' ? 'active' : ''}"
						onclick={() => (domainCategory = 'ALL')}
					>
						🌐 All Directory ({customers.length})
					</button>
					<button
						type="button"
						class="domain-tab-pill {domainCategory === 'TRAVELERS' ? 'active' : ''}"
						onclick={() => (domainCategory = 'TRAVELERS')}
					>
						✈️ Travelers & Passport DB
					</button>
					<button
						type="button"
						class="domain-tab-pill {domainCategory === 'PATIENTS' ? 'active' : ''}"
						onclick={() => (domainCategory = 'PATIENTS')}
					>
						🏥 Clinical Patients
					</button>
					<button
						type="button"
						class="domain-tab-pill {domainCategory === 'CORPORATE' ? 'active' : ''}"
						onclick={() => (domainCategory = 'CORPORATE')}
					>
						💼 Corporate & B2B Clients
					</button>
				</div>

				<div style="font-size: 0.75rem; color: var(--text-muted); display: flex; align-items: center; gap: 0.4rem;">
					{#if domainCategory === 'TRAVELERS'}
						<span style="color: #38bdf8;">✈️ Travel Agency Mode: Passport, PNR & Tour Itinerary Manifest</span>
					{:else if domainCategory === 'PATIENTS'}
						<span style="color: #34d399;">🏥 Healthcare Mode: Clinical UHID & OPD Department Roster</span>
					{:else if domainCategory === 'CORPORATE'}
						<span style="color: #c084fc;">💼 Enterprise Mode: B2B Accounts & GSTIN Ledger</span>
					{:else}
						<span>🌐 Global Multi-Tenant Master Directory</span>
					{/if}
				</div>
			</div>

			<!-- Filter Bar -->
			<div style="display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 1rem; background: var(--bg-card); padding: 0.85rem 1.25rem; border-radius: var(--radius-md); border: 1px solid var(--border);">
				<div style="display: flex; align-items: center; gap: 0.75rem; flex: 1; max-width: 480px;">
					<span style="color: var(--text-muted);">🔍</span>
					<input
						type="text"
						class="form-input"
						placeholder={domainCategory === 'TRAVELERS'
							? 'Search passenger, passport #, PNR, destination...'
							: domainCategory === 'PATIENTS'
								? 'Search patient name, UHID, doctor, department...'
								: 'Search by customer, company, GSTIN, phone...'}
						bind:value={customerSearch}
						style="padding: 0.4rem 0.75rem; font-size: 0.875rem;"
					/>
				</div>

				<div style="display: flex; align-items: center; gap: 0.75rem;">
					<label for="filter-status" style="font-size: 0.8125rem; color: var(--text-muted);">Balance Status:</label>
					<select id="filter-status" class="form-select" bind:value={balanceFilter} style="padding: 0.35rem 0.75rem; font-size: 0.8125rem;">
						<option value="ALL">All Records</option>
						<option value="DUE">Outstanding Due (> ₹0)</option>
						<option value="SETTLED">Settled / Clear (₹0)</option>
					</select>
				</div>
			</div>

			<!-- Customer Directory Table -->
			<div class="table-container card" style="padding: 0; overflow-x: auto;">
				<table class="data-table">
					<thead>
						<tr>
							{#if domainCategory === 'TRAVELERS'}
								<th>Passenger & Traveler</th>
								<th>Passport & Expiry</th>
								<th>Visa Clearance</th>
								<th>Tour Destination & PNR</th>
								<th class="text-right">Travel Billed</th>
								<th class="text-right">Paid (Credit)</th>
								<th class="text-right">Balance Due</th>
								<th class="text-center">Travel Ledger</th>
							{:else if domainCategory === 'PATIENTS'}
								<th>Patient Name & UHID</th>
								<th>Attending Doctor</th>
								<th>Care Category & Dept</th>
								<th>Contact & City</th>
								<th class="text-right">Medical Billed</th>
								<th class="text-right">Paid (Credit)</th>
								<th class="text-right">Balance Due</th>
								<th class="text-center">Clinical Ledger</th>
							{:else}
								<th>Customer & Business</th>
								<th>GSTIN & State</th>
								<th>Contact Info</th>
								<th class="text-right">Total Debit (Billed)</th>
								<th class="text-right">Total Credit (Paid)</th>
								<th class="text-right">Net Balance Due</th>
								<th class="text-center">Ledger Actions</th>
							{/if}
						</tr>
					</thead>
					<tbody>
						{#if loading}
							<tr>
								<td colspan="8" class="text-center" style="padding: 3rem; color: var(--text-muted);">
									Loading directory credit & debit balances...
								</td>
							</tr>
						{:else if error}
							<tr>
								<td colspan="8" class="text-center" style="padding: 2.5rem; color: var(--rose);">
									⚠️ {error}
								</td>
							</tr>
						{:else if filteredCustomers.length === 0}
							<tr>
								<td colspan="8" class="text-center" style="padding: 3rem; color: var(--text-muted);">
									No matching records found in this category.
								</td>
							</tr>
						{:else}
							{#each filteredCustomers as cust}
								{@const bal = parseFloat(cust.outstanding_balance) || 0}
								{@const traveler = getTravelerDetails(cust)}
								{@const patient = getPatientDetails(cust)}
								<tr>
									{#if domainCategory === 'TRAVELERS'}
										<td style="font-weight: 600;">
											<div style="color: #fff; font-size: 0.9375rem; display: flex; align-items: center; gap: 0.4rem;">
												<span>✈️</span> {cust.name}
											</div>
											<div style="font-size: 0.75rem; color: var(--text-secondary); margin-top: 0.15rem;">
												{cust.phone || cust.email || 'Lead Passenger'}
											</div>
										</td>
										<td>
											<div style="font-family: var(--font-mono); color: #38bdf8; font-size: 0.8125rem; font-weight: 700;">
												{traveler.passportNo}
											</div>
											<div style="font-size: 0.75rem; color: var(--text-muted); margin-top: 0.1rem;">
												Exp: {traveler.expiry}
											</div>
										</td>
										<td>
											<span style="display: inline-block; font-size: 0.75rem; font-weight: 700; padding: 0.2rem 0.5rem; border-radius: 4px; background: rgba(52, 211, 153, 0.15); color: {traveler.visa.color}; border: 1px solid rgba(52, 211, 153, 0.3);">
												{traveler.visa.label}
											</span>
										</td>
										<td>
											<div style="color: #e2e8f0; font-size: 0.8125rem; font-weight: 600;">
												{traveler.destination}
											</div>
											<div style="font-family: var(--font-mono); font-size: 0.72rem; color: #a855f7; margin-top: 0.1rem;">
												PNR: {traveler.pnr}
											</div>
										</td>
									{:else if domainCategory === 'PATIENTS'}
										<td style="font-weight: 600;">
											<div style="color: #fff; font-size: 0.9375rem; display: flex; align-items: center; gap: 0.4rem;">
												<span>🏥</span> {cust.name}
											</div>
											<div style="font-family: var(--font-mono); font-size: 0.75rem; color: #34d399; margin-top: 0.15rem;">
												{patient.uhid}
											</div>
										</td>
										<td>
											<div style="color: #e2e8f0; font-size: 0.8125rem; font-weight: 600;">
												{patient.doctor}
											</div>
											<div style="font-size: 0.75rem; color: var(--text-muted); margin-top: 0.1rem;">
												{patient.dept}
											</div>
										</td>
										<td>
											<div style="font-size: 0.8125rem; color: #cbd5e1;">
												{patient.careCategory}
											</div>
											<div style="font-size: 0.75rem; color: var(--text-muted); margin-top: 0.1rem;">
												Age: {patient.age}y • {patient.gender}
											</div>
										</td>
										<td style="font-size: 0.8125rem; color: var(--text-secondary);">
											<div>{cust.phone || '-'}</div>
											<div style="font-size: 0.75rem; color: var(--text-muted);">{cust.billing_city || 'Registered'}</div>
										</td>
									{:else}
										<td style="font-weight: 600;">
											<div style="color: #fff; font-size: 0.9375rem;">{cust.name}</div>
											{#if cust.company_name}
												<div style="font-size: 0.75rem; color: var(--text-secondary); font-weight: normal; margin-top: 0.1rem;">
													{cust.company_name}
												</div>
											{/if}
										</td>
										<td>
											<div style="font-family: var(--font-mono); color: #818cf8; font-size: 0.8125rem;">
												{cust.gstin || 'Unregistered'}
											</div>
											<div style="font-size: 0.75rem; color: var(--text-muted); margin-top: 0.1rem;">
												{cust.billing_state}
											</div>
										</td>
										<td style="font-size: 0.8125rem; color: var(--text-secondary);">
											<div>{cust.phone || '-'}</div>
											{#if cust.email}
												<div style="font-size: 0.75rem; color: var(--text-muted);">{cust.email}</div>
											{/if}
										</td>
									{/if}

									<td class="text-right font-mono" style="font-weight: 600; color: #fff;">
										{formatCurrency(cust.total_invoiced)}
									</td>
									<td class="text-right font-mono" style="color: #34d399; font-weight: 600;">
										{formatCurrency(cust.total_paid)}
									</td>
									<td class="text-right font-mono" style="font-weight: 700; color: {bal > 0 ? '#fb7185' : '#34d399'};">
										{formatCurrency(cust.outstanding_balance)}
									</td>
									<td class="text-center">
										<div style="display: flex; gap: 0.5rem; justify-content: center;">
											<button
												onclick={() => openCustomerLedger(cust)}
												class="btn btn-secondary btn-sm"
												style="font-size: 0.75rem; padding: 0.35rem 0.65rem;"
												title="View complete customer credit and debit statement"
											>
												{#if domainCategory === 'TRAVELERS'}
													✈️ Travel Ledger
												{:else if domainCategory === 'PATIENTS'}
													🏥 Patient Ledger
												{:else}
													📜 Statement & Ledger
												{/if}
											</button>
										</div>
									</td>
								</tr>
							{/each}
						{/if}
					</tbody>
				</table>
			</div>
		</div>
	{/if}

	<!-- TAB 2: ALL CREDIT & DEBIT HISTORY (TRANSACTION STREAM) -->
	{#if activeTab === 'all-history'}
		<div style="display: flex; flex-direction: column; gap: 1rem;">
			<!-- Filter Controls for Transactions -->
			<div style="display: flex; flex-direction: column; gap: 1rem; background: var(--bg-card); padding: 1rem 1.25rem; border-radius: var(--radius-md); border: 1px solid var(--border);">
				<div style="display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 1rem;">
					<!-- Search -->
					<div style="display: flex; align-items: center; gap: 0.6rem; flex: 1; min-width: 260px;">
						<span style="color: var(--text-muted);">🔍</span>
						<input
							type="text"
							class="form-input"
							placeholder="Search customer, invoice #, payment #, ref..."
							bind:value={historySearch}
							style="padding: 0.4rem 0.75rem; font-size: 0.875rem;"
						/>
					</div>

					<!-- Voucher Type Filter -->
					<div style="display: flex; align-items: center; gap: 0.5rem;">
						<span style="font-size: 0.8125rem; color: var(--text-muted);">Type:</span>
						<div style="display: flex; border: 1px solid var(--border); border-radius: var(--radius-md); overflow: hidden;">
							<button
								type="button"
								onclick={() => (historyType = 'ALL')}
								class="filter-pill {historyType === 'ALL' ? 'active' : ''}"
							>
								All ({allTransactions.length})
							</button>
							<button
								type="button"
								onclick={() => (historyType = 'INVOICE')}
								class="filter-pill {historyType === 'INVOICE' ? 'active' : ''}"
							>
								Debits / Invoices
							</button>
							<button
								type="button"
								onclick={() => (historyType = 'PAYMENT')}
								class="filter-pill {historyType === 'PAYMENT' ? 'active' : ''}"
							>
								Credits / Payments
							</button>
						</div>
					</div>

					<!-- Customer Dropdown Filter -->
					<div style="display: flex; align-items: center; gap: 0.5rem;">
						<span style="font-size: 0.8125rem; color: var(--text-muted);">Client:</span>
						<select class="form-select" bind:value={historyCustomerId} style="padding: 0.35rem 0.75rem; font-size: 0.8125rem; max-width: 200px;">
							<option value="">All Customers</option>
							{#each customers as c}
								<option value={c.id}>{c.name}</option>
							{/each}
						</select>
					</div>
				</div>

				<!-- Date Range and Presets -->
				<div style="display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 1rem; border-top: 1px solid rgba(255,255,255,0.05); padding-top: 0.75rem;">
					<div style="display: flex; align-items: center; gap: 0.75rem; flex-wrap: wrap;">
						<div style="display: flex; align-items: center; gap: 0.4rem;">
							<span style="font-size: 0.75rem; color: var(--text-muted);">From:</span>
							<input type="date" class="form-input" style="padding: 0.25rem 0.5rem; font-size: 0.8125rem;" bind:value={historyFromDate} />
						</div>
						<div style="display: flex; align-items: center; gap: 0.4rem;">
							<span style="font-size: 0.75rem; color: var(--text-muted);">To:</span>
							<input type="date" class="form-input" style="padding: 0.25rem 0.5rem; font-size: 0.8125rem;" bind:value={historyToDate} />
						</div>
						<button onclick={() => { historyFromDate = ''; historyToDate = ''; }} class="btn btn-secondary btn-sm" style="font-size: 0.75rem; padding: 0.25rem 0.5rem;">
							Clear Dates
						</button>
					</div>

					<div style="display: flex; gap: 0.4rem; align-items: center;">
						<span style="font-size: 0.75rem; color: var(--text-muted);">Quick:</span>
						<button onclick={() => setHistoryDatePreset(0)} class="btn btn-secondary btn-sm" style="font-size: 0.7rem; padding: 0.2rem 0.5rem;">All</button>
						<button onclick={() => setHistoryDatePreset(30)} class="btn btn-secondary btn-sm" style="font-size: 0.7rem; padding: 0.2rem 0.5rem;">Last 30D</button>
						<button onclick={() => setHistoryDatePreset(90)} class="btn btn-secondary btn-sm" style="font-size: 0.7rem; padding: 0.2rem 0.5rem;">Last 90D</button>
					</div>
				</div>

				<!-- Summary of filtered results -->
				<div style="display: flex; justify-content: space-between; align-items: center; background: rgba(15, 23, 42, 0.6); padding: 0.6rem 1rem; border-radius: var(--radius-sm); border: 1px solid rgba(255,255,255,0.06); font-size: 0.8125rem;">
					<span style="color: var(--text-secondary);">
						Showing <strong>{filteredTransactions.length}</strong> transactions
					</span>
					<div style="display: flex; gap: 1.5rem; font-family: var(--font-mono);">
						<span>Debits: <strong style="color: #fff;">{formatCurrency(filteredDebitTotal)}</strong></span>
						<span>Credits: <strong style="color: #34d399;">{formatCurrency(filteredCreditTotal)}</strong></span>
						<span>Net Flow: <strong style="color: {filteredNetBalance > 0 ? '#fb7185' : '#34d399'};">{formatCurrency(filteredNetBalance)}</strong></span>
					</div>
				</div>
			</div>

			<!-- Full History Table -->
			<div class="table-container card" style="padding: 0; overflow-x: auto;">
				<table class="data-table">
					<thead>
						<tr>
							<th>Date</th>
							<th>Customer Details</th>
							<th>Type</th>
							<th>Voucher #</th>
							<th>Status / Mode</th>
							<th>Narration / Details</th>
							<th class="text-right">Debit Amount (₹)</th>
							<th class="text-right">Credit Amount (₹)</th>
							<th class="text-center">Action</th>
						</tr>
					</thead>
					<tbody>
						{#if loading}
							<tr>
								<td colspan="9" class="text-center" style="padding: 3rem; color: var(--text-muted);">
									Loading full transaction history...
								</td>
							</tr>
						{:else if filteredTransactions.length === 0}
							<tr>
								<td colspan="9" class="text-center" style="padding: 3rem; color: var(--text-muted);">
									No credit or debit transactions match the selected filters.
								</td>
							</tr>
						{:else}
							{#each filteredTransactions as tx}
								<tr>
									<td style="font-family: var(--font-mono); font-size: 0.8125rem; color: var(--text-secondary); white-space: nowrap;">
										{formatDateStr(tx.date)}
									</td>
									<td>
										<div style="font-weight: 600; color: #fff;">{tx.customerName}</div>
										{#if tx.customerGSTIN}
											<div style="font-family: var(--font-mono); font-size: 0.75rem; color: #818cf8;">{tx.customerGSTIN}</div>
										{/if}
									</td>
									<td>
										{#if tx.type === 'INVOICE'}
											<span class="badge" style="background: rgba(56, 189, 248, 0.15); color: #38bdf8; font-weight: 700;">
												DEBIT (Invoice)
											</span>
										{:else}
											<span class="badge" style="background: rgba(16, 185, 129, 0.15); color: #34d399; font-weight: 700;">
												CREDIT (Payment)
											</span>
										{/if}
									</td>
									<td style="font-family: var(--font-mono); font-size: 0.8125rem; font-weight: 600;">
										{#if tx.type === 'INVOICE'}
											<a href="/invoices/{tx.id}" style="color: #38bdf8; text-decoration: underline;">
												{tx.voucherNumber}
											</a>
										{:else}
											<span style="color: #34d399;">{tx.voucherNumber}</span>
										{/if}
									</td>
									<td>
										<span style="font-size: 0.75rem; padding: 0.15rem 0.45rem; border-radius: 4px; background: rgba(255,255,255,0.06); color: var(--text-secondary); font-family: var(--font-mono);">
											{tx.statusOrMethod}
										</span>
									</td>
									<td style="font-size: 0.8125rem; color: var(--text-secondary); max-width: 260px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;" title={tx.narration}>
										{tx.narration}
									</td>
									<td class="text-right font-mono" style="font-weight: 600; color: {tx.debit > 0 ? '#fff' : 'var(--text-muted)'};">
										{tx.debit > 0 ? formatCurrency(tx.debit) : '-'}
									</td>
									<td class="text-right font-mono" style="font-weight: 600; color: {tx.credit > 0 ? '#34d399' : 'var(--text-muted)'};">
										{tx.credit > 0 ? formatCurrency(tx.credit) : '-'}
									</td>
									<td class="text-center">
										{#if customerMap.get(tx.customerId)}
											<button
												onclick={() => openCustomerLedger(customerMap.get(tx.customerId)!)}
												class="btn btn-secondary btn-sm"
												style="font-size: 0.7rem; padding: 0.2rem 0.5rem;"
												title="View customer statement"
											>
												📜 Statement
											</button>
										{/if}
									</td>
								</tr>
							{/each}
						{/if}
					</tbody>
				</table>
			</div>
		</div>
	{/if}
</div>

<!-- ========================================================================= -->
<!-- INDIVIDUAL CUSTOMER STATEMENT & PRINT MODAL                              -->
<!-- ========================================================================= -->
{#if showStatementModal && selectedCustomer}
	<div class="modal-backdrop">
		<div class="modal-card animate-fade-in" style="max-width: 900px; width: 95vw;">
			<!-- Modal Header with Print button -->
			<div class="modal-header no-print" style="border-bottom: 1px solid var(--border); display: flex; justify-content: space-between; align-items: center;">
				<div>
					<div style="display: flex; align-items: center; gap: 0.5rem;">
						<h3 style="font-size: 1.25rem; margin: 0; font-family: var(--font-heading);">
							Statement of Account
						</h3>
						<span style="font-size: 0.7rem; background: rgba(56, 189, 248, 0.15); color: #38bdf8; padding: 0.15rem 0.5rem; border-radius: 4px; font-weight: 700;">
							Credit & Debit Ledger
						</span>
					</div>
					<div style="font-size: 0.8125rem; color: var(--text-secondary); margin-top: 0.2rem;">
						Client: <strong style="color: #fff;">{selectedCustomer.name}</strong>
						{#if selectedCustomer.company_name} • {selectedCustomer.company_name}{/if}
						• GSTIN: <span style="font-family: var(--font-mono); color: #818cf8;">{selectedCustomer.gstin || 'Unregistered'}</span>
					</div>
				</div>

				<div style="display: flex; align-items: center; gap: 0.5rem;">
					<button onclick={triggerPrint} class="btn btn-primary btn-sm" style="display: flex; align-items: center; gap: 0.35rem;">
						🖨️ Print Statement
					</button>
					<button onclick={exportStatementCSV} class="btn btn-secondary btn-sm" style="display: flex; align-items: center; gap: 0.35rem;" title="Download CSV">
						📥 Export CSV
					</button>
					<button onclick={() => (showStatementModal = false)} class="btn btn-secondary btn-sm" style="padding: 0.35rem 0.6rem;">
						✕
					</button>
				</div>
			</div>

			<div class="modal-body" style="display: flex; flex-direction: column; gap: 1.25rem; padding: 1.25rem;">
				<!-- Date Filter Controls (Hidden on print) -->
				<div class="no-print" style="display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 0.75rem; background: rgba(15, 23, 42, 0.6); padding: 0.75rem 1rem; border-radius: var(--radius-md); border: 1px solid var(--border);">
					<div style="display: flex; align-items: center; gap: 0.75rem; flex-wrap: wrap;">
						<div style="display: flex; align-items: center; gap: 0.4rem;">
							<label for="stmt-from-date" style="font-size: 0.75rem; color: var(--text-muted);">From:</label>
							<input id="stmt-from-date" type="date" class="form-input" style="padding: 0.3rem 0.6rem; font-size: 0.8125rem;" bind:value={stmtFromDate} onchange={loadStatement} />
						</div>
						<div style="display: flex; align-items: center; gap: 0.4rem;">
							<label for="stmt-to-date" style="font-size: 0.75rem; color: var(--text-muted);">To:</label>
							<input id="stmt-to-date" type="date" class="form-input" style="padding: 0.3rem 0.6rem; font-size: 0.8125rem;" bind:value={stmtToDate} onchange={loadStatement} />
						</div>
						<button onclick={() => { stmtFromDate = ''; stmtToDate = ''; loadStatement(); }} class="btn btn-secondary btn-sm" style="font-size: 0.75rem;">
							Reset Dates
						</button>
					</div>

					<div style="display: flex; gap: 0.4rem; align-items: center;">
						<span style="font-size: 0.75rem; color: var(--text-muted);">Presets:</span>
						<button onclick={() => setQuickDatePreset(0)} class="btn btn-secondary btn-sm" style="font-size: 0.7rem; padding: 0.2rem 0.5rem;">All</button>
						<button onclick={() => setQuickDatePreset(30)} class="btn btn-secondary btn-sm" style="font-size: 0.7rem; padding: 0.2rem 0.5rem;">30 Days</button>
						<button onclick={() => setQuickDatePreset(90)} class="btn btn-secondary btn-sm" style="font-size: 0.7rem; padding: 0.2rem 0.5rem;">90 Days</button>
						<button onclick={() => setQuickDatePreset(365)} class="btn btn-secondary btn-sm" style="font-size: 0.7rem; padding: 0.2rem 0.5rem;">1 Year</button>
					</div>
				</div>

				<!-- ============================================================== -->
				<!-- PRINTABLE STATEMENT CANVAS (Target of window.print)           -->
				<!-- ============================================================== -->
				<div id="printable-customer-statement" class="statement-canvas">
					<!-- Official Business Letterhead Header -->
					<div class="statement-header">
						<div class="statement-company-info">
							<h2 style="font-size: 1.35rem; font-weight: 800; margin: 0; color: inherit;">
								{company?.name || 'SaaS Billing Enterprise'}
							</h2>
							{#if company?.legal_name && company.legal_name !== company.name}
								<div style="font-size: 0.8125rem; font-weight: 600; opacity: 0.85;">
									{company.legal_name}
								</div>
							{/if}
							<div style="font-size: 0.75rem; margin-top: 0.25rem; line-height: 1.4; opacity: 0.85;">
								{#if company?.address}{company.address}, {/if}
								{#if company?.city}{company.city}, {/if}
								{company?.state || ''} {company?.pincode || ''}
								<br />
								{#if company?.gstin}<strong>GSTIN:</strong> {company.gstin} • {/if}
								{#if company?.phone}<strong>Phone:</strong> {company.phone} • {/if}
								{#if company?.email}<strong>Email:</strong> {company.email}{/if}
							</div>
						</div>

						<div class="statement-meta-box">
							<div class="statement-title">STATEMENT OF ACCOUNT</div>
							<div style="font-size: 0.75rem; margin-top: 0.35rem; line-height: 1.4;">
								<strong>Statement Date:</strong> {new Date().toLocaleDateString('en-IN', { day: '2-digit', month: 'short', year: 'numeric' })}
								<br />
								<strong>Period:</strong>
								{#if stmtFromDate || stmtToDate}
									{stmtFromDate ? formatDateStr(stmtFromDate) : 'Start'} to {stmtToDate ? formatDateStr(stmtToDate) : 'Current'}
								{:else}
									Complete All-Time History
								{/if}
							</div>
						</div>
					</div>

					<hr class="statement-divider" />

					<!-- Bill To / Customer Information Box -->
					<div class="statement-customer-box">
						<div>
							<span style="font-size: 0.7rem; font-weight: 700; text-transform: uppercase; letter-spacing: 0.05em; opacity: 0.7;">
								Statement Issued To (Customer):
							</span>
							<div style="font-size: 1.05rem; font-weight: 800; margin-top: 0.15rem;">
								{selectedCustomer.name}
							</div>
							{#if selectedCustomer.company_name}
								<div style="font-size: 0.85rem; font-weight: 600; opacity: 0.9;">
									{selectedCustomer.company_name}
								</div>
							{/if}
							<div style="font-size: 0.75rem; margin-top: 0.2rem; opacity: 0.85; line-height: 1.4;">
								{#if selectedCustomer.billing_address}{selectedCustomer.billing_address}, {/if}
								{#if selectedCustomer.billing_city}{selectedCustomer.billing_city}, {/if}
								{selectedCustomer.billing_state} {selectedCustomer.billing_pincode || ''}
							</div>
						</div>

						<div style="text-align: right; font-size: 0.75rem; line-height: 1.4;">
							<div><strong>GSTIN:</strong> {selectedCustomer.gstin || 'Unregistered'}</div>
							<div><strong>Phone:</strong> {selectedCustomer.phone || '-'}</div>
							<div><strong>Email:</strong> {selectedCustomer.email || '-'}</div>
							<div><strong>Place of Supply:</strong> {selectedCustomer.billing_state}</div>
						</div>
					</div>

					<!-- Financial Summary KPI Boxes -->
					{#if statementData}
						<div class="statement-kpi-grid">
							<div class="statement-kpi-card">
								<div class="kpi-label">Opening Balance</div>
								<div class="kpi-value font-mono">
									{formatCurrency(statementData.opening_balance)}
								</div>
								<div class="kpi-sub">Prior to period</div>
							</div>

							<div class="statement-kpi-card">
								<div class="kpi-label">Total Invoiced (Debits)</div>
								<div class="kpi-value font-mono" style="color: #38bdf8;">
									{formatCurrency(statementData.total_invoiced)}
								</div>
								<div class="kpi-sub">+ Billed to client</div>
							</div>

							<div class="statement-kpi-card">
								<div class="kpi-label">Total Received (Credits)</div>
								<div class="kpi-value font-mono" style="color: #34d399;">
									{formatCurrency(statementData.total_paid)}
								</div>
								<div class="kpi-sub">- Payments collected</div>
							</div>

							<div class="statement-kpi-card" style="border-left: 3px solid #fb7185;">
								<div class="kpi-label">Closing Balance Due</div>
								<div class="kpi-value font-mono" style="color: {parseFloat(statementData.closing_balance) > 0 ? '#fb7185' : '#34d399'};">
									{formatCurrency(statementData.closing_balance)}
								</div>
								<div class="kpi-sub">= Net payable</div>
							</div>
						</div>
					{/if}

					<!-- Detailed Itemized Ledger Table -->
					<div class="statement-table-wrapper">
						<table class="statement-table">
							<colgroup>
								<col style="width: 12%;" />
								<col style="width: 11%;" />
								<col style="width: 18%;" />
								<col style="width: 25%;" />
								<col style="width: 11%;" />
								<col style="width: 11%;" />
								<col style="width: 12%;" />
							</colgroup>
							<thead>
								<tr>
									<th>Date</th>
									<th>Voucher Type</th>
									<th>Voucher #</th>
									<th>Narration / Transaction Description</th>
									<th class="text-right">Debit (₹)</th>
									<th class="text-right">Credit (₹)</th>
									<th class="text-right">Balance (₹)</th>
								</tr>
							</thead>
							<tbody>
								{#if statementLoading}
									<tr>
										<td colspan="7" class="text-center" style="padding: 2rem; color: var(--text-muted);">
											Loading ledger transactions...
										</td>
									</tr>
								{:else if statementError}
									<tr>
										<td colspan="7" class="text-center" style="padding: 2rem; color: var(--rose);">
											⚠️ {statementError}
										</td>
									</tr>
								{:else if !statementData?.items || statementData.items.length === 0}
									<tr>
										<td colspan="7" class="text-center" style="padding: 2rem; color: var(--text-muted);">
											No transactions recorded for this customer in the selected date range.
										</td>
									</tr>
								{:else}
									<!-- Opening balance entry row -->
									{#if statementData && parseFloat(statementData.opening_balance) !== 0}
										<tr style="background: rgba(255, 255, 255, 0.02); font-style: italic;">
											<td style="font-family: var(--font-mono); font-size: 0.8rem; white-space: nowrap;">
												{stmtFromDate ? formatDateStr(stmtFromDate) : 'Opening'}
											</td>
											<td><span class="badge" style="background: rgba(148, 163, 184, 0.15); color: #94a3b8;">OPENING</span></td>
											<td style="font-family: var(--font-mono); font-size: 0.8rem; white-space: nowrap;">-</td>
											<td style="font-size: 0.8rem;">Balance Carried Forward</td>
											<td class="text-right font-mono" style="font-size: 0.8rem; white-space: nowrap;">-</td>
											<td class="text-right font-mono" style="font-size: 0.8rem; white-space: nowrap;">-</td>
											<td class="text-right font-mono" style="font-size: 0.8rem; font-weight: 700; white-space: nowrap;">
												{formatCurrency(statementData.opening_balance)}
											</td>
										</tr>
									{/if}

									{#each statementData.items as item}
										<tr>
											<td style="font-size: 0.8125rem; font-family: var(--font-mono); white-space: nowrap;">
												{formatDateStr(item.date)}
											</td>
											<td>
												{#if item.voucher_type === 'INVOICE'}
													<span class="badge" style="background: rgba(56, 189, 248, 0.15); color: #38bdf8;">INVOICE</span>
												{:else}
													<span class="badge" style="background: rgba(16, 185, 129, 0.15); color: #34d399;">PAYMENT</span>
												{/if}
											</td>
											<td style="font-family: var(--font-mono); font-size: 0.8125rem; font-weight: 600; white-space: nowrap;">
												{item.voucher_number}
											</td>
											<td style="font-size: 0.8125rem; line-height: 1.35; word-break: break-word;">
												{item.narration}
											</td>
											<td class="text-right font-mono" style="font-size: 0.8125rem; font-weight: 600; white-space: nowrap;">
												{parseFloat(item.debit_amount) > 0 ? formatCurrency(item.debit_amount) : '-'}
											</td>
											<td class="text-right font-mono" style="font-size: 0.8125rem; font-weight: 600; color: #34d399; white-space: nowrap;">
												{parseFloat(item.credit_amount) > 0 ? formatCurrency(item.credit_amount) : '-'}
											</td>
											<td class="text-right font-mono" style="font-size: 0.8125rem; font-weight: 700; white-space: nowrap; color: {parseFloat(item.balance) > 0 ? '#fb7185' : '#34d399'};">
												{formatCurrency(item.balance)}
											</td>
										</tr>
									{/each}
								{/if}
							</tbody>
							{#if statementData && statementData.items && statementData.items.length > 0}
								<tfoot>
									<tr style="border-top: 2px solid var(--border); font-weight: 700;">
										<td colspan="4" style="text-align: right; text-transform: uppercase; font-size: 0.8rem;">
											Statement Totals:
										</td>
										<td class="text-right font-mono" style="font-size: 0.875rem; white-space: nowrap;">
											{formatCurrency(statementData.total_invoiced)}
										</td>
										<td class="text-right font-mono" style="font-size: 0.875rem; color: #34d399; white-space: nowrap;">
											{formatCurrency(statementData.total_paid)}
										</td>
										<td class="text-right font-mono" style="font-size: 0.875rem; white-space: nowrap; color: {parseFloat(statementData.closing_balance) > 0 ? '#fb7185' : '#34d399'};">
											{formatCurrency(statementData.closing_balance)}
										</td>
									</tr>
								</tfoot>
							{/if}
						</table>
					</div>

					<!-- Statement Footer: Signatory & Remittance -->
					<div class="statement-footer">
						<div style="font-size: 0.75rem; line-height: 1.5; max-width: 450px; opacity: 0.85;">
							<strong>Terms of Settlement:</strong>
							<br />
							Please verify all debits and credits listed above. If you notice any discrepancy, please notify accounts within 7 days of statement issuance.
							<br />
							<span style="font-size: 0.7rem; opacity: 0.7;">This is an authentic computer-generated statement of account.</span>
						</div>

						<div style="text-align: right; min-width: 200px;">
							<div style="height: 48px;"></div>
							<div style="border-top: 1px solid currentColor; padding-top: 0.25rem; font-size: 0.75rem; font-weight: 700;">
								For {company?.name || 'Company'}
							</div>
							<div style="font-size: 0.7rem; opacity: 0.75;">Authorized Signatory</div>
						</div>
					</div>
				</div>
			</div>

			<div class="modal-footer no-print" style="display: flex; justify-content: space-between; border-top: 1px solid var(--border); padding: 0.85rem 1.25rem;">
				<button type="button" onclick={triggerPrint} class="btn btn-primary btn-sm">
					🖨️ Print / Save as PDF
				</button>
				<button type="button" onclick={() => (showStatementModal = false)} class="btn btn-secondary btn-sm">
					Close Ledger
				</button>
			</div>
		</div>
	</div>
{/if}

<!-- ========================================================================= -->
<!-- PRINT ALL CUSTOMER BALANCES SUMMARY MODAL                                -->
<!-- ========================================================================= -->
{#if showPrintAllModal}
	<div class="modal-backdrop">
		<div class="modal-card animate-fade-in" style="max-width: 900px; width: 95vw;">
			<div class="modal-header no-print" style="border-bottom: 1px solid var(--border); display: flex; justify-content: space-between; align-items: center;">
				<div>
					<h3 style="font-size: 1.15rem; margin: 0; font-family: var(--font-heading);">
						Customer Balances Summary Report
					</h3>
					<div style="font-size: 0.8rem; color: var(--text-secondary);">
						Comprehensive balance ledger across all {customers.length} clients
					</div>
				</div>
				<div style="display: flex; gap: 0.5rem;">
					<button onclick={triggerPrint} class="btn btn-primary btn-sm">
						🖨️ Print Report
					</button>
					<button onclick={() => (showPrintAllModal = false)} class="btn btn-secondary btn-sm">
						✕
					</button>
				</div>
			</div>

			<div class="modal-body" style="padding: 1.25rem;">
				<div id="printable-all-balances" class="statement-canvas">
					<!-- Header -->
					<div class="statement-header">
						<div>
							<h2 style="font-size: 1.3rem; font-weight: 800; margin: 0;">
								{company?.name || 'SaaS Billing Enterprise'}
							</h2>
							<div style="font-size: 0.75rem; opacity: 0.85; margin-top: 0.2rem;">
								{#if company?.gstin}<strong>GSTIN:</strong> {company.gstin} • {/if}
								{#if company?.phone}<strong>Phone:</strong> {company.phone} • {/if}
								{#if company?.email}<strong>Email:</strong> {company.email}{/if}
							</div>
						</div>
						<div class="statement-meta-box">
							<div class="statement-title">CUSTOMER BALANCES LEDGER</div>
							<div style="font-size: 0.75rem; margin-top: 0.25rem;">
								Date: {new Date().toLocaleDateString('en-IN', { day: '2-digit', month: 'short', year: 'numeric' })}
							</div>
						</div>
					</div>

					<hr class="statement-divider" />

					<!-- Summary KPI -->
					<div class="statement-kpi-grid" style="margin-bottom: 1.25rem;">
						<div class="statement-kpi-card">
							<div class="kpi-label">Total Customers</div>
							<div class="kpi-value font-mono">{customers.length}</div>
						</div>
						<div class="statement-kpi-card">
							<div class="kpi-label">Total Billed (Debits)</div>
							<div class="kpi-value font-mono" style="color: #38bdf8;">{formatCurrency(totalInvoicedDebit)}</div>
						</div>
						<div class="statement-kpi-card">
							<div class="kpi-label">Total Collected (Credits)</div>
							<div class="kpi-value font-mono" style="color: #34d399;">{formatCurrency(totalPaidCredit)}</div>
						</div>
						<div class="statement-kpi-card">
							<div class="kpi-label">Net Outstanding Receivables</div>
							<div class="kpi-value font-mono" style="color: {totalOutstandingDue > 0 ? '#fb7185' : '#34d399'};">{formatCurrency(totalOutstandingDue)}</div>
						</div>
					</div>

					<!-- Table -->
					<div class="statement-table-wrapper">
						<table class="statement-table">
							<colgroup>
								<col style="width: 5%;" />
								<col style="width: 25%;" />
								<col style="width: 18%;" />
								<col style="width: 14%;" />
								<col style="width: 12%;" />
								<col style="width: 12%;" />
								<col style="width: 14%;" />
							</colgroup>
							<thead>
								<tr>
									<th>#</th>
									<th>Customer & Company</th>
									<th>GSTIN</th>
									<th>Phone / Contact</th>
									<th class="text-right">Total Debit (₹)</th>
									<th class="text-right">Total Credit (₹)</th>
									<th class="text-right">Outstanding Due (₹)</th>
								</tr>
							</thead>
							<tbody>
								{#each customers as c, idx}
									{@const bal = parseFloat(c.outstanding_balance) || 0}
									<tr>
										<td style="font-size: 0.75rem; opacity: 0.7; white-space: nowrap;">{idx + 1}</td>
										<td style="font-weight: 600;">
											<div>{c.name}</div>
											{#if c.company_name}
												<div style="font-size: 0.7rem; opacity: 0.75;">{c.company_name}</div>
											{/if}
										</td>
										<td style="font-family: var(--font-mono); font-size: 0.75rem; white-space: nowrap;">{c.gstin || '-'}</td>
										<td style="font-size: 0.75rem; white-space: nowrap;">{c.phone || '-'}</td>
										<td class="text-right font-mono" style="font-size: 0.8rem; white-space: nowrap;">{formatCurrency(c.total_invoiced)}</td>
										<td class="text-right font-mono" style="font-size: 0.8rem; color: #34d399; white-space: nowrap;">{formatCurrency(c.total_paid)}</td>
										<td class="text-right font-mono" style="font-size: 0.8rem; font-weight: 700; white-space: nowrap; color: {bal > 0 ? '#fb7185' : '#34d399'};">
											{formatCurrency(c.outstanding_balance)}
										</td>
									</tr>
								{/each}
							</tbody>
							<tfoot>
								<tr style="border-top: 2px solid var(--border); font-weight: 700;">
									<td colspan="4" style="text-align: right; text-transform: uppercase;">Total:</td>
									<td class="text-right font-mono" style="white-space: nowrap;">{formatCurrency(totalInvoicedDebit)}</td>
									<td class="text-right font-mono" style="color: #34d399; white-space: nowrap;">{formatCurrency(totalPaidCredit)}</td>
									<td class="text-right font-mono" style="white-space: nowrap; color: {totalOutstandingDue > 0 ? '#fb7185' : '#34d399'};">{formatCurrency(totalOutstandingDue)}</td>
								</tr>
							</tfoot>
						</table>
					</div>

					<div class="statement-footer">
						<div style="font-size: 0.75rem; opacity: 0.75;">
							Report generated on {new Date().toLocaleString('en-IN')}
						</div>
						<div style="text-align: right; min-width: 180px;">
							<div style="border-top: 1px solid currentColor; padding-top: 0.25rem; font-size: 0.75rem; font-weight: 700;">
								For {company?.name || 'Company'}
							</div>
							<div style="font-size: 0.7rem; opacity: 0.75;">Authorized Signatory</div>
						</div>
					</div>
				</div>
			</div>

			<div class="modal-footer no-print" style="display: flex; justify-content: space-between; border-top: 1px solid var(--border); padding: 0.85rem 1.25rem;">
				<button type="button" onclick={triggerPrint} class="btn btn-primary btn-sm">
					🖨️ Print Report
				</button>
				<button type="button" onclick={() => (showPrintAllModal = false)} class="btn btn-secondary btn-sm">
					Close
				</button>
			</div>
		</div>
	</div>
{/if}

<!-- ========================================================================= -->
<!-- ADD CUSTOMER MODAL                                                        -->
<!-- ========================================================================= -->
{#if showAddModal}
	<div class="modal-backdrop">
		<div class="modal-card animate-fade-in">
			<div class="modal-header">
				<h3 style="font-size: 1.125rem;">
					{#if domainCategory === 'TRAVELERS'}
						✈️ Add New Passenger / Traveler
					{:else if domainCategory === 'PATIENTS'}
						🏥 Register Clinical Patient
					{:else}
						Add New Customer / B2B Client
					{/if}
				</h3>
				<button onclick={() => (showAddModal = false)} style="background: none; border: none; color: var(--text-muted); cursor: pointer; font-size: 1.25rem;">✕</button>
			</div>
			<form onsubmit={handleCreateCustomer}>
				<div class="modal-body" style="display: flex; flex-direction: column; gap: 1rem;">
					{#if modalError}
						<div style="background: rgba(244, 63, 94, 0.15); border: 1px solid rgba(244, 63, 94, 0.3); border-radius: var(--radius-md); padding: 0.75rem; color: #fb7185; font-size: 0.8125rem;">
							⚠️ {modalError}
						</div>
					{/if}

					<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="cust-name">
								{domainCategory === 'TRAVELERS' ? 'Lead Traveler / Passenger Name *' : domainCategory === 'PATIENTS' ? 'Patient Full Name *' : 'Customer / Contact Name *'}
							</label>
							<input id="cust-name" type="text" class="form-input" bind:value={newName} placeholder={domainCategory === 'TRAVELERS' ? 'e.g. Rajesh Khurana' : domainCategory === 'PATIENTS' ? 'e.g. Amit Verma' : 'e.g. Rahul Sharma'} required />
						</div>

						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="cust-comp">
								{domainCategory === 'TRAVELERS' ? 'Tour Package / Reference' : domainCategory === 'PATIENTS' ? 'Attending Doctor / Dept' : 'Company / Business Name'}
							</label>
							<input id="cust-comp" type="text" class="form-input" bind:value={newCompany} placeholder={domainCategory === 'TRAVELERS' ? 'e.g. Swiss Alps 7D/6N' : domainCategory === 'PATIENTS' ? 'e.g. Dr. Sneha Roy (Cardiology)' : 'e.g. Apex Enterprises LLP'} />
						</div>
					</div>

					<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="cust-gstin">
								{domainCategory === 'TRAVELERS' ? 'Passport / PNR Ref' : domainCategory === 'PATIENTS' ? 'UHID / Clinical ID' : 'GSTIN (15 characters)'}
							</label>
							<input id="cust-gstin" type="text" maxlength="15" class="form-input font-mono" bind:value={newGSTIN} placeholder={domainCategory === 'TRAVELERS' ? 'e.g. Z9182341' : domainCategory === 'PATIENTS' ? 'e.g. UHID-MED-49102' : 'e.g. 27AABBC1234F1Z0'} />
						</div>

						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="cust-state">
								{domainCategory === 'TRAVELERS' ? 'Traveler State / Origin *' : domainCategory === 'PATIENTS' ? 'State / Region *' : 'Billing State *'}
							</label>
							<input id="cust-state" type="text" class="form-input" bind:value={newBillingState} placeholder="e.g. Maharashtra" required />
						</div>
					</div>

					<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="cust-email">Email</label>
							<input id="cust-email" type="email" class="form-input" bind:value={newEmail} placeholder="traveler@domain.com" />
						</div>

						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="cust-phone">Phone / Emergency Contact</label>
							<input id="cust-phone" type="tel" class="form-input" bind:value={newPhone} placeholder="9876543210" />
						</div>
					</div>

					<div class="form-group" style="margin-bottom: 0;">
						<label class="form-label" for="cust-addr">
							{domainCategory === 'TRAVELERS' ? 'Residential / Passport Address' : domainCategory === 'PATIENTS' ? 'Patient Address' : 'Billing Address'}
						</label>
						<textarea id="cust-addr" class="form-textarea" rows="2" bind:value={newBillingAddress} placeholder="Street, city, postal code..."></textarea>
					</div>
				</div>
				<div class="modal-footer">
					<button type="button" onclick={() => (showAddModal = false)} class="btn btn-secondary">Cancel</button>
					<button type="submit" class="btn btn-primary" disabled={modalSubmitting}>
						{modalSubmitting ? 'Saving...' : domainCategory === 'TRAVELERS' ? '✈️ Add Traveler' : domainCategory === 'PATIENTS' ? '🏥 Register Patient' : 'Add Customer'}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}

<style>
	/* Tabs Styling */
	.tab-btn {
		background: none;
		border: none;
		padding: 0.6rem 1rem;
		font-size: 0.875rem;
		font-weight: 600;
		color: var(--text-muted);
		cursor: pointer;
		border-bottom: 2px solid transparent;
		transition: all 0.2s ease;
	}
	.tab-btn:hover {
		color: var(--text-primary);
	}
	.tab-btn.active {
		color: var(--primary);
		border-bottom-color: var(--primary);
	}

	/* Filter Pills */
	.filter-pill {
		background: rgba(255, 255, 255, 0.04);
		border: none;
		padding: 0.35rem 0.75rem;
		font-size: 0.75rem;
		font-weight: 600;
		color: var(--text-secondary);
		cursor: pointer;
		transition: background 0.15s;
	}
	.filter-pill:hover {
		background: rgba(255, 255, 255, 0.08);
	}
	.filter-pill.active {
		background: var(--primary);
		color: #fff;
	}

	/* Domain Tab Pills */
	.domain-tab-pill {
		background: rgba(255, 255, 255, 0.03);
		border: 1px solid rgba(255, 255, 255, 0.07);
		border-radius: var(--radius-sm);
		padding: 0.35rem 0.75rem;
		font-size: 0.8125rem;
		font-weight: 600;
		color: var(--text-secondary);
		cursor: pointer;
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
		transition: all 0.2s ease;
	}
	.domain-tab-pill:hover {
		background: rgba(255, 255, 255, 0.07);
		color: #fff;
	}
	.domain-tab-pill.active {
		background: rgba(56, 189, 248, 0.12);
		color: #38bdf8;
		border-color: rgba(56, 189, 248, 0.35);
		box-shadow: 0 0 10px rgba(56, 189, 248, 0.1);
	}

	/* Statement Canvas & Print Layout */
	.statement-canvas {
		background: var(--bg-card);
		border: 1px solid var(--border);
		border-radius: var(--radius-md);
		padding: 1.5rem;
		display: flex;
		flex-direction: column;
		gap: 1.25rem;
	}

	.statement-header {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		gap: 1.5rem;
	}

	.statement-meta-box {
		text-align: right;
	}

	.statement-title {
		font-size: 1.15rem;
		font-weight: 800;
		letter-spacing: 0.06em;
		color: #38bdf8;
		font-family: var(--font-heading);
	}

	.statement-divider {
		border: 0;
		border-top: 1px solid var(--border);
		margin: 0;
	}

	.statement-customer-box {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		background: rgba(255, 255, 255, 0.02);
		padding: 0.85rem 1rem;
		border-radius: var(--radius-sm);
		border: 1px solid var(--border);
		gap: 1.5rem;
	}

	.statement-kpi-grid {
		display: grid;
		grid-template-columns: repeat(4, 1fr);
		gap: 0.75rem;
	}

	.statement-kpi-card {
		background: rgba(255, 255, 255, 0.02);
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		padding: 0.65rem 0.85rem;
	}

	.kpi-label {
		font-size: 0.6875rem;
		text-transform: uppercase;
		font-weight: 700;
		letter-spacing: 0.04em;
		color: var(--text-muted);
	}

	.kpi-value {
		font-size: 1.125rem;
		font-weight: 800;
		margin-top: 0.15rem;
		color: #fff;
	}

	.kpi-sub {
		font-size: 0.65rem;
		color: var(--text-muted);
		margin-top: 0.1rem;
	}

	.statement-table-wrapper {
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		overflow-x: auto;
	}

	.statement-table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.8125rem;
	}

	.statement-table th {
		background: rgba(255, 255, 255, 0.04);
		padding: 0.65rem 0.85rem;
		text-align: left;
		font-weight: 700;
		color: var(--text-secondary);
		border-bottom: 1px solid var(--border);
	}

	.statement-table td {
		padding: 0.6rem 0.85rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.04);
	}

	.statement-table tfoot td {
		padding: 0.75rem 0.85rem;
		background: rgba(255, 255, 255, 0.02);
	}

	.statement-footer {
		display: flex;
		justify-content: space-between;
		align-items: flex-end;
		margin-top: 0.5rem;
		padding-top: 0.75rem;
		border-top: 1px solid var(--border);
		gap: 1.5rem;
	}

	/* ======================================================== */
	/* PROFESSIONAL PRINT MEDIA STYLESHEET                     */
	/* ======================================================== */
	@page {
		size: A4 portrait;
		margin: 10mm 12mm 12mm 12mm;
	}

	@media print {
		/* Eliminate browser scrollbar UI completely */
		* {
			-webkit-print-color-adjust: exact !important;
			print-color-adjust: exact !important;
			scrollbar-width: none !important;
		}
		::-webkit-scrollbar {
			display: none !important;
			width: 0 !important;
			height: 0 !important;
		}

		/* Reset page root styles */
		:global(html),
		:global(body) {
			background: #ffffff !important;
			color: #000000 !important;
			margin: 0 !important;
			padding: 0 !important;
			width: 100% !important;
			height: auto !important;
			overflow: visible !important;
			font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif !important;
		}

		/* Hide application shell, headers, sidebars, navigation */
		:global(aside),
		:global(header),
		:global(nav),
		:global(.sidebar),
		:global(.topbar),
		:global(.app-layout aside),
		:global(.app-layout header),
		:global(.app-layout > *:not(.main-content)),
		.customer-page-container.print-hidden,
		.print-hidden,
		.no-print,
		:global(.no-print) {
			display: none !important;
		}

		:global(.main-content) {
			margin: 0 !important;
			padding: 0 !important;
			width: 100% !important;
			max-width: 100% !important;
			overflow: visible !important;
			background: #ffffff !important;
		}

		/* Flatten modal into normal static page document flow */
		.modal-backdrop {
			position: static !important;
			background: transparent !important;
			padding: 0 !important;
			margin: 0 !important;
			display: block !important;
			overflow: visible !important;
			inset: auto !important;
			width: 100% !important;
			height: auto !important;
			min-height: 0 !important;
			border: none !important;
			box-shadow: none !important;
			backdrop-filter: none !important;
			-webkit-backdrop-filter: none !important;
		}

		.modal-card {
			position: static !important;
			width: 100% !important;
			max-width: 100% !important;
			height: auto !important;
			max-height: none !important;
			overflow: visible !important;
			box-shadow: none !important;
			border: none !important;
			background: #ffffff !important;
			color: #000000 !important;
			padding: 0 !important;
			margin: 0 !important;
		}

		.modal-body {
			padding: 0 !important;
			margin: 0 !important;
			overflow: visible !important;
			max-height: none !important;
		}

		/* Statement canvas formatting */
		.statement-canvas {
			background: #ffffff !important;
			color: #0f172a !important;
			border: none !important;
			padding: 0 !important;
			box-shadow: none !important;
			display: block !important;
			width: 100% !important;
		}

		.statement-header {
			page-break-inside: avoid !important;
			margin-bottom: 8px !important;
		}

		.statement-title {
			color: #0f172a !important;
			border: 1.5px solid #0f172a !important;
			padding: 4px 10px !important;
			font-size: 13px !important;
			font-weight: 800 !important;
			letter-spacing: 0.08em !important;
			text-align: center !important;
			display: inline-block !important;
		}

		.statement-divider {
			border-top: 1.5px solid #0f172a !important;
			margin: 6px 0 10px 0 !important;
		}

		/* Customer info card */
		.statement-customer-box {
			background: #f8fafc !important;
			border: 1px solid #cbd5e1 !important;
			border-radius: 4px !important;
			color: #0f172a !important;
			padding: 8px 12px !important;
			page-break-inside: avoid !important;
			margin-bottom: 10px !important;
		}

		/* 4 Financial KPI Cards */
		.statement-kpi-grid {
			display: grid !important;
			grid-template-columns: repeat(4, 1fr) !important;
			gap: 8px !important;
			page-break-inside: avoid !important;
			margin: 10px 0 !important;
		}

		.statement-kpi-card {
			background: #f8fafc !important;
			border: 1px solid #cbd5e1 !important;
			border-radius: 4px !important;
			padding: 6px 10px !important;
			box-shadow: none !important;
			page-break-inside: avoid !important;
		}

		.kpi-label {
			color: #475569 !important;
			font-size: 9px !important;
			font-weight: 700 !important;
			text-transform: uppercase !important;
			letter-spacing: 0.04em !important;
		}

		.kpi-value {
			color: #0f172a !important;
			font-size: 13px !important;
			font-weight: 800 !important;
			margin-top: 2px !important;
			font-family: 'Courier New', Courier, monospace !important;
		}

		.kpi-sub {
			font-size: 8px !important;
			color: #64748b !important;
		}

		/* Ledger Table */
		.statement-table-wrapper {
			border: 1px solid #0f172a !important;
			border-radius: 0 !important;
			overflow: visible !important;
			box-shadow: none !important;
			margin: 10px 0 !important;
		}

		.statement-table {
			width: 100% !important;
			border-collapse: collapse !important;
			font-size: 10.5px !important;
			line-height: 1.35 !important;
		}

		.statement-table th {
			background: #f1f5f9 !important;
			color: #0f172a !important;
			border-bottom: 1.5px solid #0f172a !important;
			border-right: 1px solid #cbd5e1 !important;
			padding: 5px 7px !important;
			font-weight: 700 !important;
			text-transform: uppercase !important;
			font-size: 9.5px !important;
			letter-spacing: 0.03em !important;
		}
		.statement-table th:last-child {
			border-right: none !important;
		}

		.statement-table tr {
			page-break-inside: avoid !important;
			page-break-after: auto !important;
		}

		.statement-table td {
			color: #0f172a !important;
			border-bottom: 1px solid #e2e8f0 !important;
			border-right: 1px solid #cbd5e1 !important;
			padding: 5px 7px !important;
			font-size: 10.5px !important;
		}
		.statement-table td:last-child {
			border-right: none !important;
		}

		.statement-table tfoot td {
			background: #f8fafc !important;
			color: #0f172a !important;
			border-top: 2px solid #0f172a !important;
			border-bottom: 3px double #0f172a !important;
			font-weight: 800 !important;
			padding: 6px 7px !important;
			page-break-inside: avoid !important;
		}

		.badge {
			border: 1px solid #64748b !important;
			color: #0f172a !important;
			background: transparent !important;
			font-size: 8.5px !important;
			padding: 1px 4px !important;
			font-weight: 700 !important;
		}

		/* Statement Footer */
		.statement-footer {
			border-top: 1px solid #0f172a !important;
			color: #0f172a !important;
			margin-top: 12px !important;
			padding-top: 8px !important;
			page-break-inside: avoid !important;
		}
	}
</style>
