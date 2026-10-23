<script lang="ts">
	import { toast } from '$lib/stores/toast.svelte';

	interface Props {
		open: boolean;
		lead?: any;
		onclose: () => void;
	}

	let { open = false, lead, onclose }: Props = $props();

	let leadName = $state('Priya Menon');
	let leadPhone = $state('+91 98401 23456');
	let leadSource = $state('WhatsApp Business Lead');
	let leadStatus = $state('IN_PROGRESS');

	let messageInput = $state('');

	let chatMessages = $state([
		{
			id: 'msg-1',
			sender: 'client',
			senderName: 'Priya Menon',
			text: 'Hi, I am interested in a Dubai family tour with my husband and 2 kids. Can you share package details and inclusions?',
			time: '10:20 AM'
		},
		{
			id: 'msg-2',
			sender: 'agent',
			senderName: 'MrBiz Travel Advisor',
			text: 'Sure Priya! Here are our top-rated Dubai family packages including Desert Safari and Burj Khalifa passes. Which dates are you planning?',
			hasCard: true,
			cardTitle: 'Dubai Family Special (5D/4N)',
			cardPrice: '₹ 1,95,000 for 4 Pax',
			time: '10:22 AM'
		},
		{
			id: 'msg-3',
			sender: 'client',
			senderName: 'Priya Menon',
			text: 'Looks great! We are planning around 15 Nov to 20 Nov. Are flights and UAE visas included in this quotation?',
			time: '10:25 AM'
		}
	]);

	function handleSend() {
		if (!messageInput.trim()) return;
		chatMessages.push({
			id: `msg-${Date.now()}`,
			sender: 'agent',
			senderName: 'MrBiz Travel Advisor',
			text: messageInput.trim(),
			time: 'Just now'
		});
		messageInput = '';
		toast.success('WhatsApp message delivered to client via Business API.');
	}

	function handleMarkQualified() {
		leadStatus = 'QUALIFIED';
		toast.success(`Lead ${leadName} marked as QUALIFIED! Created quotation draft.`);
	}
</script>

{#if open}
	<div class="modal-backdrop" onclick={(e) => { if (e.target === e.currentTarget) onclose(); }}>
		<div class="modal-card animate-in" role="dialog" aria-modal="true">
			<!-- Header -->
			<div class="modal-header">
				<div class="lead-info-left">
					<div class="avatar-circle">PM</div>
					<div>
						<div class="lead-name-row">
							<h3 class="modal-title">{leadName}</h3>
							<span class="badge-source">📱 WhatsApp</span>
							<span class="badge-status {leadStatus.toLowerCase()}">{leadStatus}</span>
						</div>
						<div class="sub-text font-mono">{leadPhone} • {leadSource}</div>
					</div>
				</div>

				<div class="header-actions">
					<button type="button" class="btn btn-sm btn-emerald" onclick={handleMarkQualified}>
						✓ Mark as Qualified
					</button>
					<button type="button" class="close-btn" onclick={onclose}>✕</button>
				</div>
			</div>

			<!-- Chat Conversation Body -->
			<div class="chat-viewport">
				<div class="chat-date-divider">
					<span>Today • 22 Oct 2026</span>
				</div>

				<div class="chat-stream">
					{#each chatMessages as msg}
						<div class="chat-bubble-row" class:agent-row={msg.sender === 'agent'}>
							<div class="chat-bubble" class:agent-bubble={msg.sender === 'agent'}>
								<div class="bubble-sender">{msg.senderName}</div>
								<p class="bubble-text">{msg.text}</p>

								{#if msg.hasCard}
									<div class="package-attachment-card">
										<div class="att-icon">🏖️</div>
										<div class="att-info">
											<span class="att-title">{msg.cardTitle}</span>
											<span class="att-price">{msg.cardPrice}</span>
										</div>
										<button type="button" class="btn btn-xs btn-outline">View Package</button>
									</div>
								{/if}

								<span class="bubble-time font-mono">{msg.time}</span>
							</div>
						</div>
					{/each}
				</div>
			</div>

			<!-- Message Input Bar -->
			<div class="chat-input-bar">
				<input 
					type="text" 
					class="chat-input" 
					placeholder="Type an official WhatsApp reply or select quick response template..." 
					bind:value={messageInput}
					onkeydown={(e) => { if (e.key === 'Enter') handleSend(); }}
				/>
				<button type="button" class="btn btn-whatsapp" onclick={handleSend}>
					Send 🚀
				</button>
			</div>
		</div>
	</div>
{/if}

<style>
	.modal-backdrop {
		position: fixed;
		inset: 0;
		background: rgba(15, 23, 42, 0.6);
		backdrop-filter: blur(4px);
		z-index: 9999;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 1.5rem;
	}

	.modal-card {
		background: #ffffff;
		border: 1px solid #e2e8f0;
		border-radius: 12px;
		width: 100%;
		max-width: 660px;
		display: flex;
		flex-direction: column;
		box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1);
		overflow: hidden;
		color: #0f172a;
	}

	.animate-in { animation: modalFadeIn 0.18s ease-out; }
	@keyframes modalFadeIn {
		from { opacity: 0; transform: scale(0.98); }
		to { opacity: 1; transform: scale(1); }
	}

	.modal-header {
		padding: 1rem 1.25rem;
		border-bottom: 1px solid #e2e8f0;
		display: flex;
		justify-content: space-between;
		align-items: center;
		background: #ffffff;
	}

	.lead-info-left {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}

	.avatar-circle {
		width: 40px;
		height: 40px;
		border-radius: 50%;
		background: #dcfce7;
		color: #166534;
		border: 1px solid #86efac;
		display: flex;
		align-items: center;
		justify-content: center;
		font-weight: 800;
		font-size: 0.9rem;
	}

	.lead-name-row {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.modal-title { font-size: 1.05rem; font-weight: 800; margin: 0; }
	.sub-text { font-size: 0.75rem; color: #64748b; }

	.badge-source {
		font-size: 0.7rem;
		font-weight: 700;
		background: #ecfdf5;
		color: #15803d;
		padding: 0.15rem 0.45rem;
		border-radius: 4px;
	}

	.badge-status {
		font-size: 0.7rem;
		font-weight: 700;
		padding: 0.15rem 0.45rem;
		border-radius: 4px;
	}
	.badge-status.in_progress { background: #eff6ff; color: #2563eb; }
	.badge-status.qualified { background: #fef3c7; color: #d97706; }

	.header-actions {
		display: flex;
		align-items: center;
		gap: 0.65rem;
	}

	.close-btn { background: none; border: none; font-size: 1.25rem; color: #64748b; cursor: pointer; }

	.chat-viewport {
		height: 380px;
		overflow-y: auto;
		background: #efeae2;
		background-image: radial-gradient(#cbd5e1 1px, transparent 1px);
		background-size: 16px 16px;
		padding: 1.25rem;
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	.chat-date-divider {
		text-align: center;
		margin-bottom: 0.5rem;
	}
	.chat-date-divider span {
		font-size: 0.7rem;
		font-weight: 600;
		background: rgba(255, 255, 255, 0.9);
		color: #64748b;
		padding: 0.2rem 0.65rem;
		border-radius: 9999px;
		box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
	}

	.chat-stream {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	.chat-bubble-row {
		display: flex;
		justify-content: flex-start;
	}
	.chat-bubble-row.agent-row {
		justify-content: flex-end;
	}

	.chat-bubble {
		max-width: 80%;
		background: #ffffff;
		border-radius: 8px 8px 8px 2px;
		padding: 0.65rem 0.85rem;
		box-shadow: 0 1px 2px rgba(0, 0, 0, 0.08);
		position: relative;
	}
	.chat-bubble.agent-bubble {
		background: #d9fdd3;
		border-radius: 8px 8px 2px 8px;
	}

	.bubble-sender {
		font-size: 0.65rem;
		font-weight: 700;
		color: #0284c7;
		margin-bottom: 0.2rem;
	}
	.agent-bubble .bubble-sender { color: #15803d; }

	.bubble-text {
		font-size: 0.8125rem;
		color: #0f172a;
		line-height: 1.45;
		margin: 0 0 0.35rem 0;
	}

	.bubble-time {
		font-size: 0.65rem;
		color: #94a3b8;
		display: block;
		text-align: right;
	}

	.package-attachment-card {
		background: #ffffff;
		border: 1px solid #bbf7d0;
		border-radius: 6px;
		padding: 0.5rem;
		display: flex;
		align-items: center;
		gap: 0.5rem;
		margin: 0.35rem 0;
	}
	.att-icon { font-size: 1.25rem; }
	.att-info { display: flex; flex-direction: column; flex: 1; }
	.att-title { font-size: 0.75rem; font-weight: 700; color: #0f172a; }
	.att-price { font-size: 0.7rem; color: #059669; font-weight: 600; }

	.chat-input-bar {
		display: flex;
		gap: 0.5rem;
		padding: 0.85rem 1.25rem;
		background: #f0f2f5;
		border-top: 1px solid #e2e8f0;
	}

	.chat-input {
		flex: 1;
		padding: 0.55rem 0.85rem;
		border: 1px solid #cbd5e1;
		border-radius: 8px;
		font-size: 0.8125rem;
		color: #0f172a;
		background: #ffffff;
	}
	.chat-input:focus { outline: none; border-color: #2563eb; }

	.btn {
		font-size: 0.8125rem;
		font-weight: 600;
		padding: 0.5rem 1rem;
		border-radius: 8px;
		cursor: pointer;
		border: 1px solid transparent;
	}
	.btn-sm { padding: 0.35rem 0.75rem; font-size: 0.75rem; }
	.btn-xs { padding: 0.2rem 0.5rem; font-size: 0.7rem; }
	.btn-outline { background: #ffffff; border-color: #cbd5e1; color: #2563eb; }
	.btn-emerald { background: #10b981; color: #ffffff; }
	.btn-emerald:hover { background: #059669; }
	.btn-whatsapp { background: #22c55e; color: #ffffff; font-weight: 700; }
	.btn-whatsapp:hover { background: #16a34a; }
</style>
