<script lang="ts">
	import { realtimeBilling } from '$lib/stores/realtime.svelte';

	interface LiveEntry {
		id: string;
		time: string;
		type: string;
		entryNumber: string;
		narration: string;
		amount: string;
		status: 'POSTED' | 'PENDING' | 'REVERSED';
	}

	let entries = $state<LiveEntry[]>([
		{
			id: 'seed-1',
			time: 'Just now',
			type: 'INVOICE_FINALIZED',
			entryNumber: 'JV-2026-0881',
			narration: 'Subscription Renewal - Enterprise Plan (Monthly)',
			amount: '$5,999.00',
			status: 'POSTED'
		},
		{
			id: 'seed-2',
			time: '2m ago',
			type: 'PAYMENT_RECEIVED',
			entryNumber: 'JV-2026-0880',
			narration: 'Payment via Stripe Checkout (Card ending 4242)',
			amount: '$2,499.00',
			status: 'POSTED'
		}
	]);

	// Reactively prepend live events from SSE
	$effect(() => {
		const evt = realtimeBilling.lastEvent;
		if (evt && evt.type !== 'CONNECTED') {
			const newEntry: LiveEntry = {
				id: `live-${Date.now()}-${Math.random().toString(36).substring(2, 5)}`,
				time: new Date(evt.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' }),
				type: evt.type,
				entryNumber: evt.payload?.entry_number || `EVT-${Date.now().toString().slice(-4)}`,
				narration: evt.payload?.narration || evt.payload?.message || `${evt.type.replace(/_/g, ' ')}`,
				amount: evt.payload?.amount || (evt.payload?.units ? `${evt.payload.units} units` : '$0.00'),
				status: 'POSTED'
			};
			entries = [newEntry, ...entries.slice(0, 19)];
		}
	});
</script>

<div class="rounded-xl border border-zinc-800 bg-zinc-950/80 backdrop-blur-md p-5 shadow-2xl transition-all">
	<div class="flex items-center justify-between border-b border-zinc-800/80 pb-4 mb-4">
		<div class="flex items-center gap-3">
			<div class="h-8 w-8 rounded-lg bg-indigo-500/10 border border-indigo-500/20 flex items-center justify-center text-indigo-400">
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
				</svg>
			</div>
			<div>
				<h3 class="text-sm font-semibold tracking-wide text-zinc-100 flex items-center gap-2">
					Double-Entry Financial Stream
					<span class="text-[10px] px-2 py-0.5 rounded-full font-mono bg-zinc-800 text-zinc-300">Live SSE</span>
				</h3>
				<p class="text-xs text-zinc-500">Immutable append-only journal entries & metering events</p>
			</div>
		</div>

		<div class="flex items-center gap-2 px-3 py-1.5 rounded-full bg-zinc-900 border border-zinc-800 text-xs">
			<span class="relative flex h-2 w-2">
				<span class="absolute inline-flex h-full w-full animate-ping rounded-full {realtimeBilling.isConnected ? 'bg-emerald-400' : 'bg-amber-400'} opacity-75"></span>
				<span class="relative inline-flex h-2 w-2 rounded-full {realtimeBilling.isConnected ? 'bg-emerald-500' : 'bg-amber-500'}"></span>
			</span>
			<span class="font-mono text-[11px] {realtimeBilling.isConnected ? 'text-emerald-400' : 'text-amber-400'}">
				{realtimeBilling.isConnected ? 'STREAM CONNECTED' : 'RECONNECTING'}
			</span>
		</div>
	</div>

	<div class="overflow-x-auto">
		<table class="w-full text-left font-mono text-xs">
			<thead class="text-zinc-500 border-b border-zinc-800 text-[11px]">
				<tr>
					<th class="py-2.5 font-medium">TIME</th>
					<th class="py-2.5 font-medium">ENTRY NUMBER</th>
					<th class="py-2.5 font-medium">TRANSACTION / NARRATION</th>
					<th class="py-2.5 font-medium text-right">TOTAL</th>
					<th class="py-2.5 font-medium text-right">STATUS</th>
				</tr>
			</thead>
			<tbody class="divide-y divide-zinc-900">
				{#each entries as item (item.id)}
					<tr class="transition-colors hover:bg-zinc-900/40 group">
						<td class="py-3 text-zinc-500">{item.time}</td>
						<td class="py-3 font-semibold text-indigo-400 group-hover:text-indigo-300">{item.entryNumber}</td>
						<td class="py-3 text-zinc-300 font-sans text-xs">
							<span class="block">{item.narration}</span>
							<span class="text-[10px] font-mono text-zinc-500">{item.type}</span>
						</td>
						<td class="py-3 text-right text-zinc-200 font-semibold">{item.amount}</td>
						<td class="py-3 text-right">
							<span class="inline-flex items-center px-2 py-0.5 rounded text-[10px] font-bold {item.status === 'POSTED' ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20' : 'bg-amber-500/10 text-amber-400 border border-amber-500/20'}">
								{item.status}
							</span>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
</div>
