import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { platformInvoicesStore, type PlatformInvoice } from '$lib/server/adminBillingStore';
import { devAuthEnabled, demoDisabledResponse } from '$lib/server/devmode';

export const GET: RequestHandler = async ({ url }) => {
	if (!devAuthEnabled()) return demoDisabledResponse();
	const statusFilter = url.searchParams.get('status');

	let results: PlatformInvoice[] = platformInvoicesStore;
	if (statusFilter && statusFilter.trim() !== '') {
		results = results.filter((inv) => inv.status.toUpperCase() === statusFilter.toUpperCase());
	}

	return json(results);
};
