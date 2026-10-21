// Typed client for SaaS Billing API and SvelteKit BFF

export interface User {
	id: string;
	tenant_id?: string;
	email: string;
	full_name: string;
	role: string;
	is_super_admin: boolean;
	is_active: boolean;
	last_login_at?: string;
	created_at?: string;
	updated_at?: string;
}

export interface Tenant {
	id: string;
	name: string;
	legal_name: string;
	gstin: string;
	address: string;
	city: string;
	state: string;
	country: string;
	pincode: string;
	email: string;
	phone: string;
	website?: string;
	logo_url?: string;
	invoice_prefix: string;
	financial_year_start: string;
	currency: string;
	subscription_plan: string;
	subscription_status: string;
	subscription_plan_id?: string;
	billing_cycle?: string;
	max_users_override?: number;
	max_invoices_override?: number;
	renews_at?: string;
	trial_ends_at?: string;
	template_id?: string;
	bank_name?: string;
	bank_account_no?: string;
	bank_ifsc?: string;
	bank_branch?: string;
	bank_account_holder?: string;
	upi_id?: string;
	business_subtitle?: string;
	signatory_title?: string;
	signature_url?: string;
	default_terms?: string;
	default_jurisdiction?: string;
	default_interest_rate?: string;
	default_return_policy?: string;
	template_options?: Record<string, any>;
	drug_license_no?: string;
	fssai_no?: string;
	template_studio_config?: TemplateStudioConfig;
	is_active: boolean;
	created_at: string;
	updated_at?: string;
}

export interface SubscriptionPlan {
	id: string;
	code: string;
	name: string;
	description: string;
	monthly_price: number;
	yearly_price: number;
	max_users: number;
	max_invoices_per_month: number;
	features: Record<string, any>;
	is_active: boolean;
	created_at?: string;
	updated_at?: string;
}

export interface TenantSubscriptionInfo {
	tenant_id: string;
	tenant_name: string;
	plan_code: string;
	plan_name: string;
	status: string;
	billing_cycle: string;
	trial_ends_at?: string;
	renews_at?: string;
	max_users: number;
	active_users_count: number;
	max_invoices_per_month: number;
	invoices_this_month: number;
	features: Record<string, any>;
}

export interface CustomFieldDefinition {
	id: string;
	tenant_id: string;
	entity_name: 'INVOICE' | 'CUSTOMER' | 'PRODUCT';
	field_key: string;
	field_label: string;
	data_type: 'STRING' | 'NUMBER' | 'BOOLEAN' | 'DATE' | 'ENUM';
	validation_regex?: string;
	is_required: boolean;
	options?: string[];
}

export interface APIKey {
	id: string;
	tenant_id: string;
	name: string;
	key_prefix: string;
	scopes: string[];
	revoked: boolean;
	last_used_at?: string;
	expires_at?: string;
	created_at: string;
}

export interface MeteringEventInput {
	meter_slug: string;
	customer_id: string;
	value?: number;
	idempotency_key?: string;
	metadata?: Record<string, any>;
}

export interface TemplateStudioConfig {
	paper_size?: 'thermal_80mm' | 'thermal_58mm' | 'a4' | 'a5';
	theme?: string;
	font_family?: 'monospace' | 'sans' | 'serif';
	font_size?: number;
	divider_style?: 'dashed' | 'solid' | 'double' | 'dotted';
	show_logo?: boolean;
	logo_height?: number;
	show_store_name?: boolean;
	store_name_size?: number;
	show_address?: boolean;
	show_dl_no?: boolean;
	show_fssai_no?: boolean;
	show_gstin?: boolean;
	show_phone?: boolean;
	show_cashier?: boolean;
	show_patient_info?: boolean;
	show_savings_banner?: boolean;
	show_upi_qr?: boolean;
	upi_qr_size?: number;
	show_barcode?: boolean;
	custom_footer_message?: string;
	visible_columns?: {
		sno?: boolean;
		description?: boolean;
		generic_name?: boolean;
		batch_no?: boolean;
		expiry?: boolean;
		rack?: boolean;
		mrp?: boolean;
		rate?: boolean;
		qty?: boolean;
		discount?: boolean;
		tax?: boolean;
		total?: boolean;
	};
}

export interface ProductBatch {
	id: string;
	tenant_id: string;
	product_id: string;
	batch_number: string;
	expiry_date: string;
	mrp: string;
	selling_price: string;
	purchase_price: string;
	current_stock: string;
	created_at: string;
	updated_at: string;
}

export interface SplitPayment {
	mode: 'CASH' | 'UPI' | 'CARD' | 'CREDIT' | 'WALLET';
	amount: string;
	reference?: string;
}

export interface Unit {
	id: string;
	code: string;
	name: string;
	symbol: string;
	category: string;
	base_unit_code: string;
	conversion_factor: string;
	decimal_precision: number;
	is_active?: boolean;
	is_system?: boolean;
}

export interface ProductConversion {
	id: string;
	tenant_id: string;
	product_id?: string;
	from_unit_id: string;
	from_unit_code: string;
	to_unit_id: string;
	to_unit_code: string;
	factor: string;
	created_at: string;
}

export interface Product {
	id: string;
	tenant_id: string;
	name: string;
	sku: string;
	hsn_sac: string;
	category: string;
	description: string;
	product_type?: 'PHYSICAL' | 'DIGITAL' | 'SERVICE';
	fulfillment_mode?: 'NONE' | 'UNLIMITED' | 'LICENSE_POOL' | 'EXTERNAL_URL';
	digital_asset_url?: string;
	download_limit?: number;
	link_expiry_hours?: number;
	purchase_unit_id: string;
	sales_unit_id: string;
	stock_unit_id: string;
	base_unit_id: string;
	tax_rate: string;
	purchase_price: string;
	selling_price: string;
	min_stock: string;
	max_stock: string;
	barcode: string;
	current_stock: string;
	generic_name?: string;
	rack_location?: string;
	is_prescription_required?: boolean;
	mrp?: string;
	discount_rate?: string;
	batches?: ProductBatch[];
	is_active: boolean;
}

export interface DigitalLicenseKey {
	id: string;
	tenant_id: string;
	product_id: string;
	license_key: string;
	status: 'AVAILABLE' | 'ALLOCATED' | 'REVOKED' | 'EXPIRED';
	invoice_id?: string;
	customer_id?: string;
	allocated_at?: string;
	expires_at?: string;
	created_at: string;
}

export interface Warehouse {
	id: string;
	tenant_id: string;
	code: string;
	name: string;
	is_default: boolean;
	address?: string;
	is_active: boolean;
	created_at: string;
	updated_at: string;
}

export interface Customer {
	id: string;
	tenant_id: string;
	name: string;
	company_name?: string;
	gstin?: string;
	email?: string;
	phone?: string;
	billing_address: string;
	billing_city: string;
	billing_state: string;
	billing_pincode: string;
	shipping_address?: string;
	shipping_state?: string;
	is_active: boolean;
}

export interface CustomerWithBalance extends Customer {
	total_invoiced: string;
	total_paid: string;
	outstanding_balance: string;
}

export interface Supplier {
	id: string;
	tenant_id: string;
	name: string;
	company_name?: string;
	gstin?: string;
	email?: string;
	phone?: string;
	address?: string;
	city?: string;
	state?: string;
	pincode?: string;
	pan?: string;
	bank_name?: string;
	bank_account_no?: string;
	bank_ifsc?: string;
	opening_balance: string;
	outstanding_balance: string;
	is_active: boolean;
	created_at: string;
	updated_at?: string;
}

export interface PurchaseItem {
	id?: string;
	product_id: string;
	product_name?: string;
	item_description: string;
	hsn_sac: string;
	batch_number?: string;
	expiry_date?: string;
	quantity: number;
	unit_id: string;
	unit_code: string;
	purchase_price: number;
	discount_rate: number;
	discount_amount?: number;
	taxable_value?: number;
	tax_rate: number;
	cgst_rate?: number;
	cgst_amount?: number;
	sgst_rate?: number;
	sgst_amount?: number;
	igst_rate?: number;
	igst_amount?: number;
	total_amount?: number;
	mrp?: number;
	selling_price?: number;
	rack_location?: string;
}

export interface PurchaseBill {
	id: string;
	tenant_id: string;
	supplier_id: string;
	bill_number: string;
	bill_date: string;
	due_date: string;
	supplier_name: string;
	supplier_gstin?: string;
	supplier_address?: string;
	supplier_state: string;
	is_inter_state: boolean;
	vehicle_number?: string;
	po_number?: string;
	subtotal: string;
	discount_total: string;
	taxable_amount: string;
	cgst_total: string;
	sgst_total: string;
	igst_total: string;
	round_off: string;
	grand_total: string;
	amount_paid: string;
	balance_due: string;
	status: 'RECEIVED' | 'PARTIALLY_PAID' | 'PAID' | 'CANCELLED';
	notes?: string;
	created_at: string;
	updated_at?: string;
	items?: PurchaseItem[];
}

export interface SupplierPayment {
	id: string;
	tenant_id: string;
	purchase_bill_id?: string;
	supplier_id: string;
	supplier_name?: string;
	bill_number?: string;
	payment_number: string;
	payment_date: string;
	amount: string;
	payment_method: string;
	reference_number?: string;
	notes?: string;
	created_at: string;
}

export interface SupplierStatementEntry {
	date: string;
	type: 'PURCHASE' | 'PAYMENT' | 'OPENING';
	reference: string;
	description: string;
	debit: string;
	credit: string;
	running_balance: string;
}

export interface SupplierStatement {
	supplier: Supplier;
	from_date: string;
	to_date: string;
	opening_balance: string;
	total_purchases: string;
	total_payments: string;
	closing_balance: string;
	entries: SupplierStatementEntry[];
}

export interface Expense {
	id: string;
	tenant_id: string;
	expense_number: string;
	category: string;
	amount: string;
	tax_amount: string;
	payment_mode: string;
	payee_name: string;
	expense_date: string;
	description?: string;
	receipt_url?: string;
	account_id?: string;
	created_by?: string;
	created_at: string;
	updated_at?: string;
}

export interface ExpenseCategorySummary {
	category: string;
	count: number;
	total_amount: string;
}

export interface ExpenseSummary {
	total_expenses: string;
	expense_count: number;
	categories: ExpenseCategorySummary[];
	by_payment_mode: Record<string, string>;
}

export interface InvoiceItem {
	id?: string;
	product_id?: string;
	item_description: string;
	hsn_sac: string;
	quantity: string;
	unit_code: string;
	unit_price: string;
	discount_rate: string;
	discount_amount?: string;
	taxable_value?: string;
	tax_rate: string;
	cgst_rate?: string;
	cgst_amount?: string;
	sgst_rate?: string;
	sgst_amount?: string;
	igst_rate?: string;
	igst_amount?: string;
	total_amount?: string;
	batch_id?: string;
	batch_number?: string;
	expiry_date?: string;
	mrp?: string;
	rack_location?: string;
}

export interface Invoice {
	id: string;
	tenant_id: string;
	invoice_number: string;
	invoice_date: string;
	due_date: string;
	customer_id: string;
	customer_name: string;
	customer_gstin: string;
	billing_address: string;
	shipping_address: string;
	supply_state: string;
	is_inter_state: boolean;
	subtotal: string;
	discount_total: string;
	taxable_amount: string;
	cgst_total: string;
	sgst_total: string;
	igst_total: string;
	utgst_total: string;
	round_off: string;
	grand_total: string;
	amount_paid: string;
	balance_due: string;
	status: string;
	notes?: string;
	vehicle_number?: string;
	delivery_place?: string;
	payment_mode?: string;
	po_number?: string;
	po_date?: string;
	eway_bill?: string;
	transporter_name?: string;
	lr_number?: string;
	jurisdiction_city?: string;
	interest_rate?: string;
	return_policy?: string;
	reverse_charge?: boolean;
	subscription_period_start?: string;
	subscription_period_end?: string;
	gross_weight?: string;
	net_weight?: string;
	cashier_name?: string;
	counter_number?: string;
	project_name?: string;
	template_options?: Record<string, any>;
	doctor_name?: string;
	doctor_reg_no?: string;
	patient_name?: string;
	patient_age?: number;
	tendered_cash?: string;
	change_amount?: string;
	split_payments?: SplitPayment[];
	created_at: string;
	items?: InvoiceItem[];
}

export interface Payment {
	id: string;
	tenant_id: string;
	invoice_id: string;
	customer_id: string;
	payment_number: string;
	payment_date: string;
	amount: string;
	payment_method: string;
	reference_number: string;
	notes: string;
	created_at: string;
}

export interface DashboardMetrics {
	total_sales: string;
	total_received: string;
	outstanding_balance: string;
	total_invoices_count: number;
	paid_invoices_count: number;
	unpaid_invoices_count: number;
	total_inventory_value: string;
	low_stock_count: number;
	recent_invoices: Invoice[];
	recent_payments: Payment[];
}

export interface PlatformOverview {
	total_tenants: number;
	active_tenants: number;
	suspended_tenants: number;
	total_users: number;
	total_invoices: number;
	total_volume: string;
	database_status: string;
	current_server_time: string;
}

export interface AuditLog {
	id: string;
	tenant_id?: string;
	user_id?: string;
	action: string;
	entity_type: string;
	entity_id: string;
	old_values?: any;
	new_values?: any;
	ip_address: string;
	user_agent: string;
	status: string;
	created_at: string;
}

export interface Account {
	id: string;
	tenant_id: string;
	account_code: string;
	account_name: string;
	account_type: 'ASSET' | 'LIABILITY' | 'EQUITY' | 'REVENUE' | 'EXPENSE';
	sub_category?: string;
	description?: string;
	is_system: boolean;
	is_active: boolean;
	current_balance: string;
	created_at: string;
	updated_at: string;
}

export interface JournalLine {
	id: string;
	tenant_id: string;
	journal_entry_id: string;
	account_id: string;
	account_code?: string;
	account_name?: string;
	customer_id?: string;
	customer_name?: string;
	debit_amount: string;
	credit_amount: string;
	line_narration: string;
	created_at: string;
}

export interface JournalEntry {
	id: string;
	tenant_id: string;
	entry_number: string;
	entry_date: string;
	reference_type: string;
	reference_id?: string;
	narration: string;
	total_debit: string;
	total_credit: string;
	status: string;
	created_by?: string;
	created_at: string;
	lines?: JournalLine[];
}

export interface GeneralLedgerEntry {
	entry_id: string;
	entry_number: string;
	entry_date: string;
	reference_type: string;
	reference_id?: string;
	account_id: string;
	account_code: string;
	account_name: string;
	customer_name?: string;
	debit_amount: string;
	credit_amount: string;
	line_narration: string;
	running_balance: string;
}

export interface CustomerStatementItem {
	date: string;
	voucher_type: string;
	voucher_number: string;
	reference_number: string;
	narration: string;
	debit_amount: string;
	credit_amount: string;
	balance: string;
}

export interface CustomerStatement {
	customer_id: string;
	customer_name: string;
	customer_gstin: string;
	customer_phone: string;
	customer_address: string;
	opening_balance: string;
	closing_balance: string;
	total_invoiced: string;
	total_paid: string;
	items?: CustomerStatementItem[];
}

export interface TrialBalanceRow {
	account_code: string;
	account_name: string;
	account_type: string;
	debit_amount: string;
	credit_amount: string;
}

export interface TrialBalanceReport {
	as_of_date: string;
	total_debit: string;
	total_credit: string;
	is_balanced: boolean;
	difference: string;
	accounts?: TrialBalanceRow[];
}

export interface ProfitAndLossReport {
	total_revenue: string;
	total_expense: string;
	net_profit: string;
	revenues?: TrialBalanceRow[];
	expenses?: TrialBalanceRow[];
}

export interface BalanceSheetAccount {
	account_code: string;
	account_name: string;
	sub_category: string;
	balance: string;
}

export interface BalanceSheetReport {
	as_of_date: string;
	assets: BalanceSheetAccount[];
	liabilities: BalanceSheetAccount[];
	equity: BalanceSheetAccount[];
	total_assets: string;
	total_liabilities: string;
	total_equity: string;
	net_profit_to_date: string;
	total_liabilities_and_equity: string;
	difference: string;
	is_balanced: boolean;
}

export interface GSTR1InvoiceRow {
	invoice_number: string;
	invoice_date: string;
	customer_name: string;
	customer_gstin: string;
	place_of_supply: string;
	taxable_amount: string;
	cgst_amount: string;
	sgst_amount: string;
	igst_amount: string;
	total_invoice: string;
}

export interface GSTR1HSNRow {
	hsn_sac: string;
	description: string;
	unit_code: string;
	total_quantity: string;
	taxable_amount: string;
	cgst_amount: string;
	sgst_amount: string;
	igst_amount: string;
	total_amount: string;
}

export interface GSTR1Summary {
	from_date: string;
	to_date: string;
	b2b_invoices: GSTR1InvoiceRow[];
	b2c_large: GSTR1InvoiceRow[];
	b2c_small: GSTR1InvoiceRow[];
	hsn_summary: GSTR1HSNRow[];
	total_taxable: string;
	total_cgst: string;
	total_sgst: string;
	total_igst: string;
	total_invoice: string;
	total_invoices: number;
}

export interface GSTR3BOutwardSupply {
	nature_of_supplies: string;
	total_taxable: string;
	igst: string;
	cgst: string;
	sgst: string;
	cess: string;
}

export interface GSTR3BITCEligible {
	details: string;
	total_taxable: string;
	igst: string;
	cgst: string;
	sgst: string;
	cess: string;
}

export interface GSTR3BSummary {
	from_date: string;
	to_date: string;
	outward_taxable: GSTR3BOutwardSupply;
	inward_eligible_itc: GSTR3BITCEligible;
	net_tax_payable: GSTR3BOutwardSupply;
}

export interface CreditNoteItem {
	id: string;
	credit_note_id: string;
	product_id?: string;
	item_description: string;
	batch_number?: string;
	hsn_sac?: string;
	quantity: string;
	unit_code: string;
	unit_price: string;
	taxable_amount: string;
	gst_rate: string;
	cgst_amount: string;
	sgst_amount: string;
	igst_amount: string;
	total_amount: string;
	created_at: string;
}

export interface CreditNote {
	id: string;
	tenant_id: string;
	credit_note_number: string;
	credit_note_date: string;
	invoice_id?: string;
	invoice_number: string;
	customer_id?: string;
	customer_name: string;
	customer_gstin?: string;
	reason_code: string;
	reason_description?: string;
	subtotal: string;
	taxable_amount: string;
	cgst_total: string;
	sgst_total: string;
	igst_total: string;
	round_off: string;
	grand_total: string;
	status: string;
	restore_inventory: boolean;
	created_at: string;
	items?: CreditNoteItem[];
}

export interface DebitNoteItem {
	id: string;
	debit_note_id: string;
	product_id?: string;
	item_description: string;
	batch_number?: string;
	hsn_sac?: string;
	quantity: string;
	unit_code: string;
	unit_price: string;
	taxable_amount: string;
	gst_rate: string;
	cgst_amount: string;
	sgst_amount: string;
	igst_amount: string;
	total_amount: string;
	created_at: string;
}

export interface DebitNote {
	id: string;
	tenant_id: string;
	debit_note_number: string;
	debit_note_date: string;
	purchase_bill_id?: string;
	bill_number: string;
	supplier_id?: string;
	supplier_name: string;
	supplier_gstin?: string;
	reason_code: string;
	reason_description?: string;
	subtotal: string;
	taxable_amount: string;
	cgst_total: string;
	sgst_total: string;
	igst_total: string;
	round_off: string;
	grand_total: string;
	status: string;
	deduct_inventory: boolean;
	created_at: string;
	items?: DebitNoteItem[];
}

export interface WhatsAppShareInfo {
	customer_phone: string;
	formatted_phone: string;
	message_text: string;
	whatsapp_web_url: string;
}

export interface EmailDispatchResult {
	success: boolean;
	recipient: string;
	subject: string;
	status: string;
	message: string;
	mailto_url: string;
}

const API_BASE = '/api';

let selectedTenantId: string | null = null;
if (typeof window !== 'undefined') {
	selectedTenantId = localStorage.getItem('saas_selected_tenant_id');
}

export function getSelectedTenantId(): string | null {
	return selectedTenantId;
}

export function setSelectedTenantId(id: string | null) {
	selectedTenantId = id;
	if (typeof window !== 'undefined') {
		if (id) {
			localStorage.setItem('saas_selected_tenant_id', id);
			document.cookie = `saas_tenant_id=${encodeURIComponent(id)}; path=/; max-age=2592000; SameSite=Lax${location.protocol === 'https:' ? '; Secure' : ''}`;
		} else {
			localStorage.removeItem('saas_selected_tenant_id');
			document.cookie = `saas_tenant_id=; path=/; max-age=0`;
		}
	}
}

// The session credential lives only in the HttpOnly cookie set by the server; it is never readable by
// scripts. Remove any token left in localStorage by older versions of the app.
if (typeof window !== 'undefined') {
	try {
		localStorage.removeItem('saas_token');
	} catch {}
}

async function request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
	const headers: Record<string, string> = {
		'Content-Type': 'application/json',
		...((options.headers as Record<string, string>) || {})
	};

	if (selectedTenantId && !headers['X-Tenant-ID']) {
		headers['X-Tenant-ID'] = selectedTenantId;
	}

	const res = await fetch(`${API_BASE}${endpoint}`, {
		...options,
		headers,
		credentials: 'include'
	});

	if (!res.ok) {
		const errData = await res.json().catch(() => ({ error: res.statusText }));
		throw new Error(errData.error || `Request failed with status ${res.status}`);
	}

	return res.json();
}

export const api = {
	// Auth
	login: async (email: string, password: string) => {
		return request<{ user?: User; tenant?: Tenant; requires_2fa?: boolean; challenge_token?: string }>('/auth/login', {
			method: 'POST',
			body: JSON.stringify({ email, password })
		});
	},
	verifyTwoFactor: (challenge_token: string, code: string) =>
		request<{ user: User; tenant?: Tenant }>('/auth/2fa/verify', {
			method: 'POST',
			body: JSON.stringify({ challenge_token, code })
		}),
	forgotPassword: (email: string) =>
		request<{ message: string }>('/auth/forgot-password', { method: 'POST', body: JSON.stringify({ email }) }),
	resetPassword: (token: string, new_password: string) =>
		request<{ message: string }>('/auth/reset-password', { method: 'POST', body: JSON.stringify({ token, new_password }) }),
	verifyEmail: (token: string) =>
		request<{ message: string }>('/auth/verify-email', { method: 'POST', body: JSON.stringify({ token }) }),
	signup: (body: { company_name: string; full_name: string; email: string; phone: string; state: string; password: string; accept_terms: boolean }) =>
		request<{ message: string }>('/auth/signup', { method: 'POST', body: JSON.stringify(body) }),
	getBilling: () => request<any>('/billing'),
	getBillingInvoices: () => request<any[]>('/billing/invoices'),
	chooseBillingPlan: (plan_code: string, billing_cycle: string) =>
		request<any>('/billing/plan', { method: 'POST', body: JSON.stringify({ plan_code, billing_cycle }) }),
	adminBillingInvoices: (status: string) => request<any[]>(`/admin/billing/invoices?status=${encodeURIComponent(status)}`),
	adminMarkPaid: (id: string, method: string, reference: string) =>
		request<any>(`/admin/billing/invoices/${id}/mark-paid`, { method: 'POST', body: JSON.stringify({ method, reference }) }),
	adminRunBilling: () => request<any>('/admin/billing/run', { method: 'POST' }),
	listRecurring: () => request<any[]>('/recurring-invoices'),
	setRecurringStatus: (id: string, status: string) =>
		request<any>(`/recurring-invoices/${id}/status`, { method: 'POST', body: JSON.stringify({ status }) }),
	deleteRecurring: (id: string) => request<any>(`/recurring-invoices/${id}`, { method: 'DELETE' }),
	makeRecurring: (invoiceId: string, body: { name?: string; frequency: string; start_date: string; end_date?: string }) =>
		request<any>(`/invoices/${invoiceId}/make-recurring`, { method: 'POST', body: JSON.stringify(body) }),
	listQuotations: () => request<any[]>('/quotations'),
	getQuotation: (id: string) => request<any>(`/quotations/${id}`),
	createQuotation: (body: any) => request<any>('/quotations', { method: 'POST', body: JSON.stringify(body) }),
	setQuotationStatus: (id: string, status: string) =>
		request<any>(`/quotations/${id}/status`, { method: 'POST', body: JSON.stringify({ status }) }),
	convertQuotation: (id: string) => request<any>(`/quotations/${id}/convert`, { method: 'POST' }),
	deleteQuotation: (id: string) => request<any>(`/quotations/${id}`, { method: 'DELETE' }),
	getEInvoiceJson: (from: string, to: string) => request<any>(`/reports/einvoice/json?from=${from}&to=${to}`),
	getGSTR1Json: (month: string) => request<any>(`/reports/gst/gstr-1/json?month=${encodeURIComponent(month)}`),
	importCsv: async (kind: string, csv: string, dryRun: boolean) => {
		const headers: Record<string, string> = { 'Content-Type': 'text/csv' };
		if (selectedTenantId) headers['X-Tenant-ID'] = selectedTenantId;
		const res = await fetch(`${API_BASE}/import/${kind}${dryRun ? '?dry_run=1' : ''}`, { method: 'POST', headers, body: csv, credentials: 'include' });
		const data = await res.json().catch(() => ({}));
		if (!res.ok) throw new Error(data.error || `Import failed with status ${res.status}`);
		return data;
	},
	createShareLink: (invoiceId: string, days = 30) =>
		request<{ path: string; expires_at: string }>(`/invoices/${invoiceId}/share-link?days=${days}`, { method: 'POST' }),
	revokeShareLinks: (invoiceId: string) => request<{ revoked: number }>(`/invoices/${invoiceId}/share-links`, { method: 'DELETE' }),
	tradeCurrencies: () => request<any[]>('/trade/currencies'),
	tradeRates: (currency = '') => request<any[]>(`/trade/rates${currency ? '?currency=' + currency : ''}`),
	tradeRateOn: (currency: string, date = '') => request<any>(`/trade/rates/on?currency=${currency}${date ? '&date=' + date : ''}`),
	tradeSetRate: (currency: string, rate: string, date: string) =>
		request<any>('/trade/rates', { method: 'POST', body: JSON.stringify({ currency, rate, date }) }),
	tradeProfile: () => request<any>('/trade/profile'),
	tradeSaveProfile: (p: any) => request<any>('/trade/profile', { method: 'PUT', body: JSON.stringify(p) }),
	createExportInvoice: (body: any) => request<any>('/trade/export-invoices', { method: 'POST', body: JSON.stringify(body) }),
	getExportInvoice: (id: string) => request<any>(`/trade/export-invoices/${id}`),
	tradePayment: (body: any) => request<any>('/trade/payments', { method: 'POST', body: JSON.stringify(body) }),
	listShipments: (direction = '') => request<any[]>(`/trade/shipments${direction ? '?direction=' + direction : ''}`),
	createShipment: (body: any) => request<any>('/trade/shipments', { method: 'POST', body: JSON.stringify(body) }),
	getShipment: (id: string) => request<any>(`/trade/shipments/${id}`),
	updateShipment: (id: string, body: any) => request<any>(`/trade/shipments/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
	updateShipmentDoc: (id: string, docId: string, body: any) =>
		request<any>(`/trade/shipments/${id}/documents/${docId}`, { method: 'PUT', body: JSON.stringify(body) }),
	setShipmentPackages: (id: string, packages: any[]) =>
		request<any>(`/trade/shipments/${id}/packages`, { method: 'PUT', body: JSON.stringify({ packages }) }),
	addShipmentCost: (id: string, body: any) => request<any>(`/trade/shipments/${id}/costs`, { method: 'POST', body: JSON.stringify(body) }),
	deleteShipmentCost: (id: string, costId: string) => request<any>(`/trade/shipments/${id}/costs/${costId}`, { method: 'DELETE' }),
	getShipmentDocData: (id: string) => request<any>(`/trade/shipments/${id}/doc-data`),
	shipmentLandedCost: (id: string) => request<any>(`/trade/shipments/${id}/landed-cost`),
	getLegalStatus: () =>
		request<{ required_version: string; accepted: boolean; accepted_at?: string }>('/legal/status'),
	acceptLegal: (version: string) =>
		request<{ message: string }>('/legal/accept', { method: 'POST', body: JSON.stringify({ version }) }),
	resendVerification: () => request<{ message: string }>('/auth/resend-verification', { method: 'POST' }),
	getSecurityStatus: () =>
		request<{ email: string; email_verified: boolean; two_factor_enabled: boolean; recovery_codes_remaining: number }>('/auth/security'),
	changePassword: (current_password: string, new_password: string) =>
		request<{ message: string }>('/auth/change-password', { method: 'POST', body: JSON.stringify({ current_password, new_password }) }),
	twoFactorSetup: (password: string) =>
		request<{ secret: string; otpauth_uri: string }>('/auth/2fa/setup', { method: 'POST', body: JSON.stringify({ password }) }),
	twoFactorEnable: (code: string) =>
		request<{ recovery_codes: string[]; message: string }>('/auth/2fa/enable', { method: 'POST', body: JSON.stringify({ code }) }),
	twoFactorDisable: (password: string, code: string) =>
		request<{ message: string }>('/auth/2fa/disable', { method: 'POST', body: JSON.stringify({ password, code }) }),
	logout: async () => {
		try {
			await request<{ message: string }>('/auth/logout', { method: 'POST' });
		} finally {
			setSelectedTenantId(null);
		}
	},
	getMe: () => request<{ user: User; tenant?: Tenant }>('/auth/me'),

	// Company / Tenant
	getCompany: () => request<Tenant>('/company'),
	updateCompany: (data: Partial<Tenant>) =>
		request<Tenant>('/company', { method: 'PATCH', body: JSON.stringify(data) }),

	// Super Admin
	listTenants: () => request<Tenant[]>('/admin/tenants'),
	createTenant: (data: Partial<Tenant>) =>
		request<Tenant>('/admin/tenants', { method: 'POST', body: JSON.stringify(data) }),
	updateTenant: (id: string, data: Partial<Tenant>) =>
		request<Tenant>(`/admin/tenants/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
	getPlatformOverview: () => request<PlatformOverview>('/admin/overview'),
	listAuditLogs: () => request<AuditLog[]>('/admin/audit'),

	// Subscriptions & Plan Tiers
	getCompanySubscription: () => request<TenantSubscriptionInfo>('/company/subscription'),
	listSubscriptionPlans: () => request<SubscriptionPlan[]>('/admin/subscriptions/plans'),
	updateTenantSubscription: (tenantId: string, data: {
		plan_id?: string;
		plan_code?: string;
		status?: string;
		billing_cycle?: string;
		trial_ends_at?: string;
		renews_at?: string;
		max_users_override?: number | null;
		max_invoices_override?: number | null;
	}) =>
		request<{ message: string }>(`/admin/tenants/${tenantId}/subscription`, {
			method: 'PATCH',
			body: JSON.stringify(data)
		}),

	// Units
	listUnits: () => request<Unit[]>('/units'),
	createUnit: (data: {
		code: string;
		name: string;
		symbol?: string;
		category: string;
		base_unit_code?: string;
		conversion_factor?: number | string;
		decimal_precision?: number;
	}) =>
		request<Unit>('/units', {
			method: 'POST',
			body: JSON.stringify(data)
		}),
	convertUnit: (quantity: string, from_code: string, to_code: string) =>
		request<{ result: string }>('/units/convert', {
			method: 'POST',
			body: JSON.stringify({ quantity, from_code, to_code })
		}),
	listProductConversions: (productId: string) =>
		request<ProductConversion[]>(`/products/${productId}/conversions`),
	createProductConversion: (
		productId: string,
		data: { from_unit_id: string; to_unit_id: string; factor: number | string }
	) =>
		request<ProductConversion>(`/products/${productId}/conversions`, {
			method: 'POST',
			body: JSON.stringify(data)
		}),

	// Products
	listProducts: () => request<Product[]>('/products'),
	createProduct: (data: Partial<Product>) =>
		request<Product>('/products', { method: 'POST', body: JSON.stringify(data) }),

	// Customers (Basic Plan & above)
	listCustomers: () => request<Customer[]>('/customers'),
	listCustomersWithBalances: () => request<CustomerWithBalance[]>('/customers/balances'),
	createCustomer: (data: Partial<Customer>) =>
		request<Customer>('/customers', { method: 'POST', body: JSON.stringify(data) }),
	getCustomerStatement: (id: string, params?: { from_date?: string; to_date?: string }) => {
		const q = new URLSearchParams();
		if (params?.from_date) q.set('from_date', params.from_date);
		if (params?.to_date) q.set('to_date', params.to_date);
		const qs = q.toString();
		return request<CustomerStatement>(`/customers/${id}/statement${qs ? '?' + qs : ''}`);
	},

	// Suppliers (Vendors & Khata Ledger)
	listSuppliers: () => request<Supplier[]>('/suppliers'),
	getSupplier: (id: string) => request<Supplier>(`/suppliers/${id}`),
	createSupplier: (data: Partial<Supplier>) =>
		request<Supplier>('/suppliers', { method: 'POST', body: JSON.stringify(data) }),
	updateSupplier: (id: string, data: Partial<Supplier>) =>
		request<Supplier>(`/suppliers/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
	getSupplierStatement: (id: string, params?: { from_date?: string; to_date?: string }) => {
		const q = new URLSearchParams();
		if (params?.from_date) q.set('from_date', params.from_date);
		if (params?.to_date) q.set('to_date', params.to_date);
		const qs = q.toString();
		return request<SupplierStatement>(`/suppliers/${id}/statement${qs ? '?' + qs : ''}`);
	},

	// Purchase Invoices / Inward Bills
	listPurchases: (params?: { status?: string; supplier_id?: string }) => {
		const q = new URLSearchParams();
		if (params?.status) q.set('status', params.status);
		if (params?.supplier_id) q.set('supplier_id', params.supplier_id);
		const qs = q.toString();
		return request<PurchaseBill[]>(`/purchases${qs ? '?' + qs : ''}`);
	},
	getPurchase: (id: string) => request<PurchaseBill>(`/purchases/${id}`),
	createPurchase: (data: any) =>
		request<PurchaseBill>('/purchases', { method: 'POST', body: JSON.stringify(data) }),
	recordSupplierPayment: (billId: string, data: { amount: string | number; payment_method: string; reference_number?: string; notes?: string }) =>
		request<SupplierPayment>(`/purchases/${billId}/payments`, { method: 'POST', body: JSON.stringify(data) }),

	// Expenses (Next Plan Tier)
	listExpenses: (params?: { category?: string; from_date?: string; to_date?: string }) => {
		const q = new URLSearchParams();
		if (params?.category) q.set('category', params.category);
		if (params?.from_date) q.set('from_date', params.from_date);
		if (params?.to_date) q.set('to_date', params.to_date);
		const qs = q.toString();
		return request<Expense[]>(`/expenses${qs ? '?' + qs : ''}`);
	},
	createExpense: (data: Partial<Expense>) =>
		request<Expense>('/expenses', { method: 'POST', body: JSON.stringify(data) }),
	getExpenseSummary: (params?: { from_date?: string; to_date?: string }) => {
		const q = new URLSearchParams();
		if (params?.from_date) q.set('from_date', params.from_date);
		if (params?.to_date) q.set('to_date', params.to_date);
		const qs = q.toString();
		return request<ExpenseSummary>(`/expenses/summary${qs ? '?' + qs : ''}`);
	},

	// Invoices
	listInvoices: () => request<Invoice[]>('/invoices'),
	getInvoice: (id: string) => request<Invoice>(`/invoices/${id}`),
	createInvoice: (data: any) =>
		request<Invoice>('/invoices', { method: 'POST', body: JSON.stringify(data) }),
	updateInvoiceLogistics: (id: string, data: {
		vehicle_number?: string;
		delivery_place?: string;
		payment_mode?: string;
		po_number?: string;
		po_date?: string;
		eway_bill?: string;
		transporter_name?: string;
		lr_number?: string;
		jurisdiction_city?: string;
		interest_rate?: string;
		return_policy?: string;
		reverse_charge?: boolean;
		subscription_period_start?: string;
		subscription_period_end?: string;
		gross_weight?: string;
		net_weight?: string;
		cashier_name?: string;
		counter_number?: string;
		project_name?: string;
		template_options?: Record<string, any>;
	}) => request<Invoice>(`/invoices/${id}/logistics`, {
		method: 'PUT',
		body: JSON.stringify(data)
	}),
	cancelInvoice: (id: string, reason: string) =>
		request<{ message: string }>(`/invoices/${id}/cancel`, {
			method: 'POST',
			body: JSON.stringify({ reason })
		}),

	// Payments
	listPayments: () => request<Payment[]>('/payments'),
	createPayment: (data: any) =>
		request<Payment>('/payments', { method: 'POST', body: JSON.stringify(data) }),

	// Inventory
	listInventory: () => request<any[]>('/inventory'),
	adjustInventory: (productId: string, newStock: string, reason: string) =>
		request<{ message: string }>('/inventory/adjust', {
			method: 'POST',
			body: JSON.stringify({ product_id: productId, new_stock: newStock, reason })
		}),

	// Dashboard
	getDashboard: () => request<DashboardMetrics>('/dashboard'),

	// Accounting & Financial Ledger
	listAccounts: () => request<Account[]>('/accounting/accounts'),
	getGeneralLedger: (params?: { account_id?: string; from_date?: string; to_date?: string }) => {
		const q = new URLSearchParams();
		if (params?.account_id) q.set('account_id', params.account_id);
		if (params?.from_date) q.set('from_date', params.from_date);
		if (params?.to_date) q.set('to_date', params.to_date);
		const qs = q.toString();
		return request<GeneralLedgerEntry[]>(`/accounting/ledger${qs ? '?' + qs : ''}`);
	},
	getTrialBalance: (asOfDate?: string) => {
		const qs = asOfDate ? `?as_of_date=${asOfDate}` : '';
		return request<TrialBalanceReport>(`/accounting/trial-balance${qs}`);
	},
	getProfitAndLoss: (params?: { from_date?: string; to_date?: string }) => {
		const q = new URLSearchParams();
		if (params?.from_date) q.set('from_date', params.from_date);
		if (params?.to_date) q.set('to_date', params.to_date);
		const qs = q.toString();
		return request<ProfitAndLossReport>(`/accounting/pnl${qs ? '?' + qs : ''}`);
	},
	getBalanceSheet: (asOfDate?: string) => {
		const qs = asOfDate ? `?as_of=${asOfDate}` : '';
		return request<BalanceSheetReport>(`/accounting/balance-sheet${qs}`);
	},
	getGSTR1Summary: (params?: { from_date?: string; to_date?: string }) => {
		const q = new URLSearchParams();
		if (params?.from_date) q.set('from_date', params.from_date);
		if (params?.to_date) q.set('to_date', params.to_date);
		const qs = q.toString();
		return request<GSTR1Summary>(`/reports/gst/gstr-1${qs ? '?' + qs : ''}`);
	},
	getGSTR3BSummary: (params?: { from_date?: string; to_date?: string }) => {
		const q = new URLSearchParams();
		if (params?.from_date) q.set('from_date', params.from_date);
		if (params?.to_date) q.set('to_date', params.to_date);
		const qs = q.toString();
		return request<GSTR3BSummary>(`/reports/gst/gstr-3b${qs ? '?' + qs : ''}`);
	},

	// Credit & Debit Notes
	listCreditNotes: () => request<CreditNote[]>('/credit-notes'),
	createCreditNote: (data: {
		invoice_id?: string;
		customer_id?: string;
		customer_name: string;
		customer_gstin?: string;
		reason_code: string;
		reason_description?: string;
		restore_inventory?: boolean;
		items: Array<{
			product_id?: string;
			item_description: string;
			batch_number?: string;
			hsn_sac?: string;
			quantity: number | string;
			unit_code?: string;
			unit_price: number | string;
			gst_rate: number | string;
		}>;
	}) =>
		request<CreditNote>('/credit-notes', {
			method: 'POST',
			body: JSON.stringify(data)
		}),
	getCreditNote: (id: string) => request<CreditNote>(`/credit-notes/${id}`),

	listDebitNotes: () => request<DebitNote[]>('/debit-notes'),
	createDebitNote: (data: {
		purchase_bill_id?: string;
		supplier_id?: string;
		supplier_name: string;
		supplier_gstin?: string;
		reason_code: string;
		reason_description?: string;
		deduct_inventory?: boolean;
		items: Array<{
			product_id?: string;
			item_description: string;
			batch_number?: string;
			hsn_sac?: string;
			quantity: number | string;
			unit_code?: string;
			unit_price: number | string;
			gst_rate: number | string;
		}>;
	}) =>
		request<DebitNote>('/debit-notes', {
			method: 'POST',
			body: JSON.stringify(data)
		}),
	getDebitNote: (id: string) => request<DebitNote>(`/debit-notes/${id}`),

	getJournalByReference: (refType: string, refId: string) =>
		request<JournalEntry>(`/accounting/journal-entries/${refType}/${refId}`),

	// Communications (WhatsApp Web & Email)
	getWhatsAppShare: (id: string) => request<WhatsAppShareInfo>(`/invoices/${id}/whatsapp-share`),
	sendInvoiceEmail: (id: string, data: { recipient_email?: string; subject?: string; custom_message?: string; attach_pdf?: boolean }) =>
		request<EmailDispatchResult>(`/invoices/${id}/send-email`, {
			method: 'POST',
			body: JSON.stringify(data)
		}),

	// Digital Licenses & Warehouses
	listProductLicenses: (productId: string) => request<DigitalLicenseKey[]>(`/products/${productId}/licenses`),
	addProductLicenses: (productId: string, keys: string[]) =>
		request<{ imported: number }>(`/products/${productId}/licenses`, {
			method: 'POST',
			body: JSON.stringify({ keys })
		}),
	listWarehouses: () => request<Warehouse[]>('/warehouses'),

	// POS & High-Speed Search
	searchPOSProducts: (q: string) => request<Product[]>(`/pos/products?q=${encodeURIComponent(q)}`),

	// Product Batches (DPCO Pharmacy & Retail Expiry Batches)
	listProductBatches: (productId: string) => request<ProductBatch[]>(`/products/${productId}/batches`),
	createProductBatch: (productId: string, data: Partial<ProductBatch>) =>
		request<ProductBatch>(`/products/${productId}/batches`, {
			method: 'POST',
			body: JSON.stringify(data)
		}),

	// Bill Template Studio Customizer
	updateTemplateStudioConfig: (config: TemplateStudioConfig) =>
		request<{ status: string }>('/company/template-studio', {
			method: 'PUT',
			body: JSON.stringify(config)
		}),

	// Authenticated PDF Downloader
	downloadInvoicePDF: async (
		id: string,
		filename?: string,
		options?: { copy?: string; tax_mode?: string; size?: string }
	) => {
		const q = new URLSearchParams();
		if (options?.copy) q.set('copy', options.copy);
		if (options?.tax_mode) q.set('tax_mode', options.tax_mode);
		if (options?.size) q.set('size', options.size);
		const qs = q.toString();

		const headers: Record<string, string> = {};
		if (selectedTenantId) headers['X-Tenant-ID'] = selectedTenantId;

		const res = await fetch(`${API_BASE}/invoices/${id}/pdf${qs ? '?' + qs : ''}`, {
			headers,
			credentials: 'include'
		});
		if (!res.ok) {
			const err = await res.json().catch(() => ({ error: res.statusText }));
			throw new Error(err.error || `Failed to download PDF: status ${res.status}`);
		}
		const blob = await res.blob();
		const url = window.URL.createObjectURL(blob);
		const a = document.createElement('a');
		a.href = url;
		a.download = filename || `invoice_${id.slice(0, 8)}.pdf`;
		document.body.appendChild(a);
		a.click();
		window.URL.revokeObjectURL(url);
		document.body.removeChild(a);
	},

	// Custom Fields (Extensibility Engine)
	getCustomFields: (entity: string = 'INVOICE') =>
		request<CustomFieldDefinition[]>(`/v1/custom-fields?entity=${encodeURIComponent(entity)}`),
	createCustomField: (data: {
		entity_name: string;
		field_key: string;
		field_label: string;
		data_type: string;
		validation_regex?: string;
		is_required?: boolean;
		options?: string[];
	}) =>
		request<{ status: string }>('/v1/custom-fields', {
			method: 'POST',
			body: JSON.stringify(data)
		}),

	// Developer API Keys
	listAPIKeys: () => request<APIKey[]>('/v1/developer/api-keys'),
	createAPIKey: (data: { name: string; scopes?: string[]; live_mode?: boolean }) =>
		request<{ api_key: APIKey; secret: string }>('/v1/developer/api-keys', {
			method: 'POST',
			body: JSON.stringify(data)
		}),

	// High-Throughput Metering Ingestion
	ingestMeteringEvent: (data: MeteringEventInput) =>
		request<{ status: string }>('/v1/metering/events', {
			method: 'POST',
			body: JSON.stringify(data)
		}),

	// Tenant Staff & User Management
	listUsers: () => request<User[]>('/users'),
	createUser: (data: { email: string; full_name: string; role: string; password: string }) =>
		request<User>('/users', {
			method: 'POST',
			body: JSON.stringify(data)
		}),
	updateUser: (id: string, data: { full_name?: string; role?: string; is_active?: boolean }) =>
		request<User>(`/users/${id}`, {
			method: 'PATCH',
			body: JSON.stringify(data)
		}),
	deleteUser: (id: string) =>
		request<{ message: string }>(`/users/${id}`, {
			method: 'DELETE'
		})
};

