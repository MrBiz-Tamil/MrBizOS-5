import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { devAuthEnabled, demoDisabledResponse } from '$lib/server/devmode';

export const GET: RequestHandler = async () => {
	if (!devAuthEnabled()) return demoDisabledResponse();
	return json({
		required_version: '2026-04-01',
		accepted: true,
		accepted_at: '2026-04-01T00:00:00Z'
	});
};
