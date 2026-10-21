import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

export const GET: RequestHandler = async () => {
	return json({
		status: 'ACTIVE',
		plan_code: 'ENTERPRISE',
		plan_name: 'Enterprise Unlimited',
		billing_cycle: 'ANNUAL',
		trial_days_left: null,
		subscription_id: 'sub-ent-01'
	});
};
