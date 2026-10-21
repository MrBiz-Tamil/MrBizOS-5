import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

export const GET: RequestHandler = async () => {
	return json({
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
		bank_account_holder: 'Global Business Group Ltd',
		upi_id: 'mrbiz@upi',
		business_subtitle: 'Multi-Industry Enterprise Holdings',
		signatory_title: 'Executive Director & Authorized Signatory',
		default_terms: 'Payment due upon invoice receipt. Subject to Chennai jurisdiction.',
		default_jurisdiction: 'Chennai',
		default_interest_rate: '18% p.a.',
		default_return_policy: 'Standard terms apply across all industry subsidiaries.',
		drug_license_no: 'DL-TN-2026-9921',
		fssai_no: '11521001000452',
		is_active: true,
		created_at: new Date().toISOString()
	});
};
