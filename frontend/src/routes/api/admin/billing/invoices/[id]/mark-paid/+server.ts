import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { platformInvoicesStore, type PlatformInvoice } from '$lib/server/adminBillingStore';

export const POST: RequestHandler = async ({ params, request }) => {
	const { id } = params;
	const body = await request.json().catch(() => ({}));
	const { method = 'BANK_TRANSFER', reference = '' } = body;

	const invoiceIndex = platformInvoicesStore.findIndex((inv: PlatformInvoice) => inv.id === id);
	if (invoiceIndex === -1) {
		return json({ error: `Plan invoice ${id} not found` }, { status: 404 });
	}

	const updated: PlatformInvoice = {
		...platformInvoicesStore[invoiceIndex],
		status: 'PAID' as const,
		payment_method: method || 'BANK_TRANSFER',
		payment_reference: reference || `UTR/RECORDED/${Date.now().toString().slice(-6)}`,
		paid_at: new Date().toISOString()
	};

	platformInvoicesStore[invoiceIndex] = updated;

	return json({
		success: true,
		message: `Invoice ${updated.invoice_number} marked as PAID. Tenant plan activated.`,
		invoice: updated
	});
};
