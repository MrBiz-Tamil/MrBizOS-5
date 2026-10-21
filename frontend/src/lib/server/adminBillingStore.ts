export interface PlatformInvoice {
	id: string;
	tenant_id: string;
	tenant_name: string;
	invoice_number: string;
	plan_code: string;
	billing_cycle: string;
	period_start: string;
	period_end: string;
	amount: number;
	gst_amount: number;
	total_amount: number;
	status: 'OPEN' | 'PAID' | 'VOID';
	due_date: string;
	paid_at: string | null;
	payment_method: string;
	payment_reference: string;
	created_at: string;
}

export let platformInvoicesStore: PlatformInvoice[] = [
	{
		id: 'pinv-101',
		tenant_id: 'tenant-1',
		tenant_name: 'Apex Global Enterprises Ltd',
		invoice_number: 'PINV-2026-001',
		plan_code: 'ENTERPRISE',
		billing_cycle: 'ANNUAL',
		period_start: '2026-10-01',
		period_end: '2027-09-30',
		amount: 99990.00,
		gst_amount: 17998.20,
		total_amount: 117988.20,
		status: 'PAID',
		due_date: '2026-10-15',
		paid_at: '2026-10-05T14:32:00Z',
		payment_method: 'BANK_TRANSFER',
		payment_reference: 'UTR/HDFC/2026100599182',
		created_at: '2026-10-01T00:00:00Z'
	},
	{
		id: 'pinv-102',
		tenant_id: 'tenant-2',
		tenant_name: 'CarePlus Multi-Specialty Clinic',
		invoice_number: 'PINV-2026-002',
		plan_code: 'PRO',
		billing_cycle: 'ANNUAL',
		period_start: '2026-10-01',
		period_end: '2027-09-30',
		amount: 29990.00,
		gst_amount: 5398.20,
		total_amount: 35388.20,
		status: 'OPEN',
		due_date: '2026-10-31',
		paid_at: null,
		payment_method: '',
		payment_reference: '',
		created_at: '2026-10-01T00:00:00Z'
	},
	{
		id: 'pinv-103',
		tenant_id: 'tenant-3',
		tenant_name: 'Horizon Travels & Visa Services',
		invoice_number: 'PINV-2026-003',
		plan_code: 'GROWTH',
		billing_cycle: 'MONTHLY',
		period_start: '2026-10-01',
		period_end: '2026-10-31',
		amount: 4999.00,
		gst_amount: 899.82,
		total_amount: 5898.82,
		status: 'OPEN',
		due_date: '2026-10-25',
		paid_at: null,
		payment_method: '',
		payment_reference: '',
		created_at: '2026-10-01T00:00:00Z'
	},
	{
		id: 'pinv-104',
		tenant_id: 'tenant-4',
		tenant_name: 'Prime Estate Realty Group',
		invoice_number: 'PINV-2026-004',
		plan_code: 'ENTERPRISE',
		billing_cycle: 'ANNUAL',
		period_start: '2026-09-01',
		period_end: '2027-08-31',
		amount: 99990.00,
		gst_amount: 17998.20,
		total_amount: 117988.20,
		status: 'PAID',
		due_date: '2026-09-15',
		paid_at: '2026-09-08T11:20:00Z',
		payment_method: 'NEFT_RTGS',
		payment_reference: 'UTR/ICIC/202609088129',
		created_at: '2026-09-01T00:00:00Z'
	},
	{
		id: 'pinv-105',
		tenant_id: 'tenant-5',
		tenant_name: 'SwiftLogistics Express Cargo',
		invoice_number: 'PINV-2026-005',
		plan_code: 'STARTER',
		billing_cycle: 'MONTHLY',
		period_start: '2026-10-01',
		period_end: '2026-10-31',
		amount: 1999.00,
		gst_amount: 359.82,
		total_amount: 2358.82,
		status: 'OPEN',
		due_date: '2026-10-28',
		paid_at: null,
		payment_method: '',
		payment_reference: '',
		created_at: '2026-10-01T00:00:00Z'
	}
];
