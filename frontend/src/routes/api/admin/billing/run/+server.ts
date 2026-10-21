import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { platformInvoicesStore, type PlatformInvoice } from '$lib/server/adminBillingStore';

export const POST: RequestHandler = async () => {
	// Simulate evaluating subscription tenures and generating upcoming renewal invoice
	const nextIdNum = platformInvoicesStore.length + 101;
	const newInvoice: PlatformInvoice = {
		id: `pinv-${nextIdNum}`,
		tenant_id: `tenant-${(platformInvoicesStore.length % 5) + 1}`,
		tenant_name: 'Metro Diagnostics & Imaging Lab',
		invoice_number: `PINV-2026-${String(platformInvoicesStore.length + 1).padStart(3, '0')}`,
		plan_code: 'PRO',
		billing_cycle: 'MONTHLY',
		period_start: '2026-11-01',
		period_end: '2026-11-30',
		amount: 2999.00,
		gst_amount: 539.82,
		total_amount: 3538.82,
		status: 'OPEN' as const,
		due_date: '2026-11-10',
		paid_at: null,
		payment_method: '',
		payment_reference: '',
		created_at: new Date().toISOString()
	};

	platformInvoicesStore.unshift(newInvoice);

	return json({
		success: true,
		cycle_timestamp: new Date().toISOString(),
		tenants_evaluated: 14,
		renewals_generated: 1,
		new_invoice: newInvoice,
		message: `Platform billing cycle completed. 14 tenants evaluated, generated ${newInvoice.invoice_number}.`
	});
};
