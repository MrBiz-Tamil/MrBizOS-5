import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { devAuthEnabled, demoDisabledResponse } from '$lib/server/devmode';

export const GET: RequestHandler = async () => {
	if (!devAuthEnabled()) return demoDisabledResponse();
	// Demo-only: a hardcoded super-admin session. Real sessions must be resolved
	// from the session cookie by a real backend; this is gated behind MRBIZ_DEV_AUTH.
	return json({
		user: {
			id: 'user-1',
			tenant_id: 'tenant-1',
			email: 'admin@mrbiz.local',
			full_name: 'Executive Administrator',
			role: 'admin',
			is_super_admin: true,
			is_active: true,
			current_business_unit_id: 'travel',
			allowed_business_units: ['travel', 'realestate', 'healthcare', 'trade', 'quarry', 'consulting']
		},
		tenant: {
			id: 'tenant-1',
			name: 'Global Business Group',
			legal_name: 'Global Business Group Ltd',
			gstin: '27AABCA1234F1Z6',
			address: 'Skyline Commercial Towers, Anna Salai',
			city: 'Chennai',
			state: 'Tamil Nadu',
			country: 'India',
			pincode: '600002',
			email: 'admin@mrbiz.local',
			phone: '+91 98400 12345',
			invoice_prefix: 'INV-2026',
			financial_year_start: '2026-04-01',
			currency: 'INR',
			subscription_plan: 'ENTERPRISE',
			subscription_status: 'ACTIVE',
			bank_name: 'HDFC Bank Ltd',
			bank_account_no: '50200012345678',
			bank_ifsc: 'HDFC0000240',
			bank_branch: 'Mount Road',
			upi_id: 'mrbiz@upi'
		}
	});
};
