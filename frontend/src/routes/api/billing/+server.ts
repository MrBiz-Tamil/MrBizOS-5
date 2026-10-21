import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { devAuthEnabled, demoDisabledResponse } from '$lib/server/devmode';

export const GET: RequestHandler = async () => {
	if (!devAuthEnabled()) return demoDisabledResponse();
	return json({
		status: 'ACTIVE',
		plan_code: 'ENTERPRISE',
		plan_name: 'Enterprise Unlimited',
		billing_cycle: 'ANNUAL',
		trial_days_left: null,
		subscription_id: 'sub-ent-01'
	});
};
