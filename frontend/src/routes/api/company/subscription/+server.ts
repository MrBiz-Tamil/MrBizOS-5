import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { devAuthEnabled, demoDisabledResponse } from '$lib/server/devmode';

export const GET: RequestHandler = async () => {
	if (!devAuthEnabled()) return demoDisabledResponse();
	return json({
		tenant_id: 'tenant-1',
		plan_code: 'ENTERPRISE',
		plan_name: 'Enterprise Multi-OS',
		status: 'ACTIVE',
		billing_cycle: 'ANNUAL',
		max_users: 100,
		max_invoices: 100000,
		features: {
			accounting: true,
			expenses: true,
			reports: true,
			realtime: true,
			multi_currency: true,
			api_access: true,
			unlimited_branches: true
		}
	});
};
