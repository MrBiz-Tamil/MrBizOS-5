<script lang="ts">
	import { toast } from '$lib/stores/toast.svelte';

	interface Props {
		open: boolean;
		onclose: () => void;
	}

	let { open = false, onclose }: Props = $props();

	let currentStep = $state<number>(1);

	let documents = $state([
		{ id: 'doc-pass', name: 'Passport Copy', status: 'uploaded' },
		{ id: 'doc-photo', name: 'Passport Size Photo (White BG)', status: 'uploaded' },
		{ id: 'doc-itin', name: 'Travel Itinerary & Flight PNR', status: 'uploaded' },
		{ id: 'doc-hotel', name: 'Hotel Confirmation Voucher', status: 'pending' },
		{ id: 'doc-bank', name: 'Bank Statement (6 Months)', status: 'pending' },
		{ id: 'doc-cover', name: 'Cover Letter / Leave Approval', status: 'uploaded' },
		{ id: 'doc-ins', name: 'Travel Medical Insurance ($50K)', status: 'pending' }
	]);

	function handleUpload(docId: string) {
		const doc = documents.find(d => d.id === docId);
		if (doc) {
			doc.status = 'uploaded';
			toast.success(`${doc.name} uploaded and verified!`);
		}
	}
</script>

{#if open}
	<div class="modal-backdrop" onclick={(e) => { if (e.target === e.currentTarget) onclose(); }}>
		<div class="modal-card animate-in" role="dialog" aria-modal="true">
			<!-- Header -->
			<div class="modal-header">
				<div>
					<h3 class="modal-title">Visa Processing Desk</h3>
					<span class="sub-text">Document verification, consulate submission and real-time tracking</span>
				</div>
				<button type="button" class="close-btn" onclick={onclose}>✕</button>
			</div>

			<!-- Stepper Bar -->
			<div class="stepper-bar">
				<div class="step-item" class:active={currentStep >= 1} class:current={currentStep === 1}>
					<span class="step-num">1</span>
					<span class="step-name">Documents</span>
				</div>
				<div class="step-line" class:filled={currentStep >= 2}></div>
				<div class="step-item" class:active={currentStep >= 2} class:current={currentStep === 2}>
					<span class="step-num">2</span>
					<span class="step-name">Application</span>
				</div>
				<div class="step-line" class:filled={currentStep >= 3}></div>
				<div class="step-item" class:active={currentStep >= 3} class:current={currentStep === 3}>
					<span class="step-num">3</span>
					<span class="step-name">Submission</span>
				</div>
				<div class="step-line" class:filled={currentStep >= 4}></div>
				<div class="step-item" class:active={currentStep >= 4} class:current={currentStep === 4}>
					<span class="step-num">4</span>
					<span class="step-name">Tracking</span>
				</div>
			</div>

			<!-- Body Layout -->
			<div class="modal-body-grid">
				<!-- Left: Required Documents Checklist -->
				<div class="checklist-col">
					<div class="col-head">
						<span class="head-title">Required Documents</span>
						<span class="head-count">{documents.filter(d => d.status === 'uploaded').length}/{documents.length} Ready</span>
					</div>

					<div class="doc-list">
						{#each documents as doc}
							<div class="doc-item">
								<div class="doc-name-row">
									<span class="doc-icon">{doc.status === 'uploaded' ? '✅' : '📄'}</span>
									<span class="doc-name">{doc.name}</span>
								</div>
								{#if doc.status === 'uploaded'}
									<span class="badge-uploaded">Uploaded ✓</span>
								{:else}
									<button type="button" class="btn-upload" onclick={() => handleUpload(doc.id)}>
										Upload
									</button>
								{/if}
							</div>
						{/each}
					</div>
				</div>

				<!-- Right: Status & Timeline Tracker -->
				<div class="timeline-col">
					<div class="col-head">
						<span class="head-title">Status &amp; Timeline</span>
						<span class="badge-live">In Processing</span>
					</div>

					<div class="visa-timeline">
						<div class="timeline-node done">
							<div class="node-circle">✓</div>
							<div class="node-content">
								<span class="node-title">Documents Received</span>
								<span class="node-date">15 Oct 2026 • Front Desk</span>
							</div>
						</div>

						<div class="timeline-connector filled"></div>

						<div class="timeline-node done">
							<div class="node-circle">✓</div>
							<div class="node-content">
								<span class="node-title">Application Submitted</span>
								<span class="node-date">16 Oct 2026 • Embassy Portal</span>
							</div>
						</div>

						<div class="timeline-connector active"></div>

						<div class="timeline-node active">
							<div class="node-circle pulse">⏳</div>
							<div class="node-content">
								<span class="node-title">In Consulate Processing</span>
								<span class="node-date">Under Review • VFS Desk</span>
							</div>
						</div>

						<div class="timeline-connector"></div>

						<div class="timeline-node pending">
							<div class="node-circle">🕒</div>
							<div class="node-content">
								<span class="node-title">Approval &amp; Stamping Expected</span>
								<span class="node-date">Estimated: 20 Nov 2026</span>
							</div>
						</div>
					</div>

					<!-- Applicant Summary Card -->
					<div class="applicant-box mt-3">
						<div class="app-row">
							<span class="app-label">Applicant</span>
							<span class="app-val">Rahul Sharma</span>
						</div>
						<div class="app-row">
							<span class="app-label">Country</span>
							<span class="app-val">Dubai (UAE Tourist 30-Day)</span>
						</div>
						<div class="app-row">
							<span class="app-label">Application Ref</span>
							<span class="app-val font-mono">DXB-V-2026-9921</span>
						</div>
					</div>
				</div>
			</div>

			<!-- Footer -->
			<div class="modal-footer">
				<button type="button" class="btn btn-secondary" onclick={onclose}>Cancel</button>
				<button type="button" class="btn btn-primary" onclick={() => { toast.success('Visa dossier updated!'); onclose(); }}>
					Next Step →
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
		max-width: 760px;
		max-height: 90vh;
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
		padding: 1.25rem 1.5rem;
		border-bottom: 1px solid #e2e8f0;
		display: flex;
		justify-content: space-between;
		align-items: center;
	}

	.modal-title { font-size: 1.25rem; font-weight: 800; margin: 0; }
	.sub-text { font-size: 0.8125rem; color: #64748b; }
	.close-btn { background: none; border: none; font-size: 1.25rem; color: #64748b; cursor: pointer; }

	.stepper-bar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0.85rem 1.5rem;
		background: #f8fafc;
		border-bottom: 1px solid #e2e8f0;
	}

	.step-item {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 0.8125rem;
		font-weight: 600;
		color: #94a3b8;
	}
	.step-item.active { color: #0f172a; }
	.step-item.current { color: #2563eb; }

	.step-num {
		width: 22px;
		height: 22px;
		border-radius: 50%;
		background: #e2e8f0;
		color: #64748b;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 0.75rem;
		font-weight: 700;
	}
	.step-item.current .step-num { background: #2563eb; color: #ffffff; }

	.step-line {
		flex: 1;
		height: 2px;
		background: #e2e8f0;
		margin: 0 0.5rem;
	}
	.step-line.filled { background: #2563eb; }

	.modal-body-grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		flex: 1;
		overflow-y: auto;
	}

	.checklist-col {
		padding: 1.25rem 1.5rem;
		border-right: 1px solid #e2e8f0;
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	.timeline-col {
		padding: 1.25rem 1.5rem;
		background: #fafafa;
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}

	.col-head {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding-bottom: 0.5rem;
		border-bottom: 1px solid #e2e8f0;
	}

	.head-title {
		font-size: 0.8125rem;
		font-weight: 700;
		color: #475569;
		text-transform: uppercase;
		letter-spacing: 0.04em;
	}

	.head-count {
		font-size: 0.75rem;
		font-weight: 700;
		color: #2563eb;
		background: #eff6ff;
		padding: 0.15rem 0.5rem;
		border-radius: 9999px;
	}

	.doc-list {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.doc-item {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 0.55rem 0.75rem;
		background: #ffffff;
		border: 1px solid #e2e8f0;
		border-radius: 6px;
		font-size: 0.8125rem;
	}

	.doc-name-row {
		display: flex;
		align-items: center;
		gap: 0.4rem;
	}

	.doc-name { font-weight: 600; color: #0f172a; }

	.badge-uploaded {
		font-size: 0.7rem;
		font-weight: 700;
		color: #059669;
		background: #ecfdf5;
		border: 1px solid #a7f3d0;
		padding: 0.15rem 0.5rem;
		border-radius: 9999px;
	}

	.btn-upload {
		font-size: 0.7rem;
		font-weight: 600;
		color: #2563eb;
		background: #eff6ff;
		border: 1px solid #bfdbfe;
		padding: 0.2rem 0.6rem;
		border-radius: 6px;
		cursor: pointer;
	}
	.btn-upload:hover { background: #dbeafe; }

	.badge-live {
		font-size: 0.7rem;
		font-weight: 700;
		color: #2563eb;
		background: #eff6ff;
		padding: 0.2rem 0.5rem;
		border-radius: 9999px;
	}

	.visa-timeline {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
		padding-left: 0.5rem;
	}

	.timeline-node {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}

	.node-circle {
		width: 24px;
		height: 24px;
		border-radius: 50%;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 0.75rem;
		font-weight: 700;
	}
	.timeline-node.done .node-circle { background: #10b981; color: #ffffff; }
	.timeline-node.active .node-circle { background: #2563eb; color: #ffffff; }
	.timeline-node.pending .node-circle { background: #e2e8f0; color: #64748b; }

	.timeline-connector {
		width: 2px;
		height: 16px;
		background: #e2e8f0;
		margin-left: 11px;
	}
	.timeline-connector.filled { background: #10b981; }
	.timeline-connector.active { background: #2563eb; }

	.node-content { display: flex; flex-direction: column; }
	.node-title { font-size: 0.8125rem; font-weight: 700; color: #0f172a; }
	.node-date { font-size: 0.7rem; color: #64748b; }

	.applicant-box {
		background: #ffffff;
		border: 1px solid #e2e8f0;
		border-radius: 8px;
		padding: 0.85rem;
		display: flex;
		flex-direction: column;
		gap: 0.4rem;
	}

	.app-row {
		display: flex;
		justify-content: space-between;
		font-size: 0.75rem;
	}
	.app-label { color: #64748b; font-weight: 600; }
	.app-val { color: #0f172a; font-weight: 700; }

	.modal-footer {
		padding: 1rem 1.5rem;
		border-top: 1px solid #e2e8f0;
		display: flex;
		justify-content: space-between;
		align-items: center;
		background: #ffffff;
	}

	.btn {
		font-size: 0.8125rem;
		font-weight: 600;
		padding: 0.5rem 1rem;
		border-radius: 8px;
		cursor: pointer;
		border: 1px solid transparent;
	}
	.btn-secondary { background: #f1f5f9; color: #334155; }
	.btn-primary { background: #2563eb; color: #ffffff; }
	.btn-primary:hover { background: #1d4ed8; }
	.mt-3 { margin-top: 0.75rem; }
</style>
