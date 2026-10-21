import { env } from '$env/dynamic/private';
import { json } from '@sveltejs/kit';

/**
 * devAuthEnabled reports whether the insecure local-development demo mode is
 * explicitly enabled via MRBIZ_DEV_AUTH. It is OFF by default so deployed
 * environments fail closed: the demo/compatibility BFF routes that return
 * hardcoded sample payloads (super-admin identity, fixed financials) must not
 * serve fabricated data as if it were real. This mirrors the backend flag of
 * the same name in internal/grpcserver.
 */
export function devAuthEnabled(): boolean {
	const v = (env.MRBIZ_DEV_AUTH ?? '').trim().toLowerCase();
	return v === '1' || v === 'true' || v === 'yes' || v === 'on';
}

/** Standard fail-closed response for a demo endpoint with no real implementation. */
export function demoDisabledResponse() {
	return json(
		{
			error:
				'This endpoint has no real-data implementation. Demo payloads are disabled; set MRBIZ_DEV_AUTH=1 for local demo mode.'
		},
		{ status: 501 }
	);
}
