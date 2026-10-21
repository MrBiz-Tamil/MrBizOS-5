import type { Handle } from '@sveltejs/kit';
import { redirect } from '@sveltejs/kit';

// Server-side guard: pages other than /login require a session cookie. This is only a presence check
// (the API validates the session on every call), but it stops protected app shells from being served
// to anonymous visitors and adds baseline security headers to every page.
const PUBLIC_PATHS = ['/login', '/forgot-password', '/reset-password', '/verify-email', '/terms', '/privacy', '/signup', '/robots.txt', '/favicon.svg'];

export const handle: Handle = async ({ event, resolve }) => {
	const { pathname } = event.url;
	const isPublic = PUBLIC_PATHS.some((p) => pathname === p) || pathname.startsWith('/_app/') || pathname.startsWith('/i/');
	if (!isPublic && !event.cookies.get('saas_session')) {
		throw redirect(303, '/login');
	}

	const response = await resolve(event);
	response.headers.set('X-Content-Type-Options', 'nosniff');
	response.headers.set('X-Frame-Options', 'DENY');
	response.headers.set('Referrer-Policy', 'strict-origin-when-cross-origin');
	response.headers.set('Permissions-Policy', 'camera=(), microphone=(), geolocation=()');
	return response;
};
