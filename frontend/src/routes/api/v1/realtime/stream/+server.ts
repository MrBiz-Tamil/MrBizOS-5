import type { RequestHandler } from './$types';

export const GET: RequestHandler = async () => {
	const stream = new ReadableStream({
		start(controller) {
			const encoder = new TextEncoder();
			const now = new Date().toISOString();
			controller.enqueue(
				encoder.encode(
					`event: billing_update\ndata: {"type":"CONNECTED","payload":{"message":"Subscribed to MrBiz-OS real-time stream"},"timestamp":"${now}"}\n\n`
				)
			);
			const interval = setInterval(() => {
				controller.enqueue(encoder.encode(`: heartbeat ${new Date().toISOString()}\n\n`));
			}, 20000);

			return () => clearInterval(interval);
		}
	});

	return new Response(stream, {
		headers: {
			'Content-Type': 'text/event-stream',
			'Cache-Control': 'no-cache',
			'Connection': 'keep-alive',
			'Access-Control-Allow-Origin': '*'
		}
	});
};
