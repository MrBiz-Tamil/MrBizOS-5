// src/lib/stores/realtime.svelte.ts
// Real-time Server-Sent Events (SSE) Client using Svelte 5 Runes

export interface BillingStreamEvent {
	type: 'CONNECTED' | 'PAYMENT_RECEIVED' | 'INVOICE_PAID' | 'JOURNAL_ENTRY_POSTED' | 'METER_TICK';
	payload: any;
	timestamp: string;
}

class RealtimeBillingManager {
	#eventSource: EventSource | null = null;
	#reconnectAttempts = 0;
	#maxReconnectDelay = 15000;

	isConnected = $state(false);
	lastEvent = $state<BillingStreamEvent | null>(null);
	eventLog = $state<BillingStreamEvent[]>([]);

	connect(_tenantId?: string) {
		if (typeof window === 'undefined') return;
		if (this.#eventSource) {
			this.disconnect();
		}

		// The server derives the tenant from the authenticated session (and, for super admins, the
		// saas_tenant_id cookie); it never trusts a tenant id supplied in the URL.
		const url = `/api/v1/realtime/stream`;

		try {
			this.#eventSource = new EventSource(url);

			this.#eventSource.onopen = () => {
				this.isConnected = true;
				this.#reconnectAttempts = 0;
			};

			this.#eventSource.addEventListener('billing_update', (e: MessageEvent) => {
				try {
					const data: BillingStreamEvent = JSON.parse(e.data);
					this.lastEvent = data;
					this.eventLog = [data, ...this.eventLog.slice(0, 49)];
				} catch (err) {
					console.error('[RealtimeSSE] Failed to parse message:', err);
				}
			});

			this.#eventSource.onerror = () => {
				this.isConnected = false;
				this.#eventSource?.close();
				this.#eventSource = null;

				const delay = Math.min(1000 * Math.pow(1.5, this.#reconnectAttempts), this.#maxReconnectDelay);
				this.#reconnectAttempts++;
				setTimeout(() => {
					if (!this.#eventSource) {
						this.connect();
					}
				}, delay);
			};
		} catch (err) {
			console.error('[RealtimeSSE] Connection initialization error:', err);
		}
	}

	disconnect() {
		if (this.#eventSource) {
			this.#eventSource.close();
			this.#eventSource = null;
			this.isConnected = false;
		}
	}
}

export const realtimeBilling = new RealtimeBillingManager();
