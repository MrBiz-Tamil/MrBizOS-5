import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { devAuthEnabled, demoDisabledResponse } from '$lib/server/devmode';

export const GET: RequestHandler = async () => {
	if (!devAuthEnabled()) return demoDisabledResponse();
	return json({
		total_sales: '1428500.00',
		total_received: '1198500.00',
		outstanding_balance: '230000.00',
		total_invoices_count: 48,
		paid_invoices_count: 41,
		unpaid_invoices_count: 7,
		total_inventory_value: '890000.00',
		low_stock_count: 2,
		recent_invoices: [],
		recent_payments: []
	});
};
