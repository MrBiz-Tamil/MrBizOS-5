<script lang="ts">
	import { onMount } from 'svelte';
	import { workflowGrpc } from '$lib/grpc';

	let tasks = $state<any[]>([]);
	let approvals = $state<any[]>([]);
	let loading = $state(true);
	let error = $state('');
	let actionFeedback = $state('');

	onMount(async () => {
		try {
			const [tRes, aRes]: [any, any] = await Promise.all([
				workflowGrpc.listTasks({ businessUnitId: 'healthcare' }),
				workflowGrpc.listApprovalQueue({ businessUnitId: 'all' })
			]);
			tasks = tRes?.tasks || [];
			approvals = aRes?.items || [];
		} catch (err: any) {
			error = err.message || 'Failed to connect to Workflow gRPC service';
		} finally {
			loading = false;
		}
	});

	async function handleApprove(id: string) {
		try {
			await workflowGrpc.approveAction({ id, reason: 'Approved via Executive Console' });
			approvals = approvals.filter(a => a.id !== id);
			actionFeedback = `Proposal ${id} approved successfully!`;
			setTimeout(() => actionFeedback = '', 3000);
		} catch (err: any) {
			alert('Approval failed: ' + err.message);
		}
	}

	async function handleReject(id: string) {
		try {
			await workflowGrpc.rejectAction({ id, reason: 'Rejected via Executive Console' });
			approvals = approvals.filter(a => a.id !== id);
			actionFeedback = `Proposal ${id} rejected.`;
			setTimeout(() => actionFeedback = '', 3000);
		} catch (err: any) {
			alert('Rejection failed: ' + err.message);
		}
	}
</script>

<div class="workflow-dashboard">
	<div class="header">
		<div>
			<h1 class="title">AI Missions & Workflow Approval Queue</h1>
			<p class="subtitle">Human-in-the-loop governance for autonomous agent proposals and team tasks.</p>
		</div>
	</div>

	{#if actionFeedback}
		<div class="success-banner">✨ {actionFeedback}</div>
	{/if}

	{#if error}
		<div class="error-banner">⚠️ {error}</div>
	{/if}

	{#if loading}
		<div class="loading-state">Fetching workflow queue...</div>
	{:else}
		<div class="grid-2">
			<!-- AI Autonomous Approvals -->
			<div class="card">
				<h2 class="section-title">🤖 AI Agent Proposals (Pending Approval)</h2>
				<div class="list">
					{#if approvals.length === 0}
						<div class="empty-state">No pending actions in the approval queue.</div>
					{/if}
					{#each approvals as item}
						<div class="approval-card">
							<div class="card-header">
								<span class="source-tag">{item.source} · {item.requestedBy}</span>
								<span class="time-tag">{new Date(item.createdAt).toLocaleTimeString()}</span>
							</div>
							<div class="card-body">
								<div class="item-title">{item.title}</div>
								<pre class="json-preview">{item.payloadJson}</pre>
							</div>
							<div class="card-actions">
								<button class="btn-reject" onclick={() => handleReject(item.id)}>Reject</button>
								<button class="btn-approve" onclick={() => handleApprove(item.id)}>Approve Execution</button>
							</div>
						</div>
					{/each}
				</div>
			</div>

			<!-- Operational Tasks -->
			<div class="card">
				<h2 class="section-title">📋 Active Group Tasks</h2>
				<div class="list">
					{#each tasks as task}
						<div class="task-card">
							<div class="task-header">
								<span class="badge priority-{task.priority.toLowerCase()}">{task.priority}</span>
								<span class="task-status">{task.status}</span>
							</div>
							<div class="task-title">{task.title}</div>
							<div class="task-desc">{task.description}</div>
						</div>
					{/each}
				</div>
			</div>
		</div>
	{/if}
</div>

<style>
	.workflow-dashboard {
		padding: 1.5rem;
		max-width: 1400px;
		margin: 0 auto;
	}
	.header {
		margin-bottom: 2rem;
	}
	.title {
		font-size: 1.75rem;
		font-weight: 700;
		color: #f8fafc;
	}
	.subtitle {
		font-size: 0.875rem;
		color: #94a3b8;
		margin-top: 0.25rem;
	}
	.success-banner {
		background: rgba(16, 185, 129, 0.15);
		border: 1px solid #10b981;
		color: #34d399;
		padding: 0.75rem 1rem;
		border-radius: 8px;
		margin-bottom: 1.5rem;
	}
	.error-banner {
		background: rgba(239, 68, 68, 0.1);
		border: 1px solid #ef4444;
		color: #fca5a5;
		padding: 0.75rem 1rem;
		border-radius: 8px;
		margin-bottom: 1.5rem;
	}
	.loading-state, .empty-state {
		text-align: center;
		padding: 2rem;
		color: #94a3b8;
	}
	.grid-2 {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
		gap: 1.5rem;
	}
	.card {
		background: #1e293b;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 12px;
		padding: 1.5rem;
	}
	.section-title {
		font-size: 1.125rem;
		font-weight: 600;
		color: #f8fafc;
		margin-bottom: 1rem;
	}
	.list {
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}
	.approval-card, .task-card {
		background: rgba(255, 255, 255, 0.03);
		border: 1px solid rgba(255, 255, 255, 0.06);
		border-radius: 8px;
		padding: 1rem;
	}
	.card-header, .task-header {
		display: flex;
		justify-content: space-between;
		font-size: 0.75rem;
		color: #94a3b8;
		margin-bottom: 0.5rem;
	}
	.source-tag {
		color: #06b6d4;
		font-weight: 600;
	}
	.item-title, .task-title {
		font-weight: 600;
		color: #f1f5f9;
		margin-bottom: 0.5rem;
	}
	.json-preview {
		background: #0f172a;
		padding: 0.5rem;
		border-radius: 4px;
		font-size: 0.75rem;
		color: #cbd5e1;
		overflow-x: auto;
		margin-bottom: 0.75rem;
	}
	.card-actions {
		display: flex;
		justify-content: flex-end;
		gap: 0.5rem;
	}
	.btn-approve {
		background: #10b981;
		color: white;
		border: none;
		padding: 0.4rem 0.8rem;
		border-radius: 6px;
		font-size: 0.75rem;
		cursor: pointer;
	}
	.btn-reject {
		background: transparent;
		border: 1px solid rgba(239, 68, 68, 0.5);
		color: #f87171;
		padding: 0.4rem 0.8rem;
		border-radius: 6px;
		font-size: 0.75rem;
		cursor: pointer;
	}
	.badge {
		font-size: 0.6875rem;
		padding: 0.15rem 0.4rem;
		border-radius: 4px;
		font-weight: 600;
	}
	.priority-high {
		background: rgba(239, 68, 68, 0.15);
		color: #f87171;
	}
	.priority-medium {
		background: rgba(245, 158, 11, 0.15);
		color: #fbbf24;
	}
	.task-desc {
		font-size: 0.8125rem;
		color: #94a3b8;
	}
</style>
