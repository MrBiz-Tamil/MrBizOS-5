import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

export const GET: RequestHandler = async () => {
	return json({
		required_version: '2026-04-01',
		accepted: true,
		accepted_at: '2026-04-01T00:00:00Z'
	});
};
