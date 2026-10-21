<script lang="ts">
	import { onMount } from 'svelte';
	import { api, type AuditLog } from '$lib/api';

	let logs = $state<AuditLog[]>([]);
	let loading = $state(true);
	let error = $state('');
	let selectedLog = $state<AuditLog | null>(null);

	onMount(async () => {
		try {
			logs = await api.listAuditLogs();
		} catch (err: any) {
			error = err.message || 'Failed to load audit logs';
		} finally {
			loading = false;
		}
	});
</script>

<svelte:head>
	<title>Security & Audit Trail - Super Admin</title>
</svelte:head>

<div style="display: flex; flex-direction: column; gap: 1.5rem;">
	<div>
		<a href="/admin" style="font-size: 0.8125rem; color: #818cf8; text-decoration: none; margin-bottom: 0.25rem; display: inline-block;">
			← Back to Admin Console
		</a>
		<h1 style="font-size: 1.75rem; margin-bottom: 0.25rem;">Immutable System Audit Logs</h1>
		<p style="font-size: 0.875rem; color: var(--text-secondary);">
			Tamper-proof record of logins, transactions, financial cancellations, and forbidden access attempts.
		</p>
	</div>

	<div class="table-container card" style="padding: 0;">
		<table class="data-table">
			<thead>
				<tr>
					<th>Timestamp</th>
					<th>Action</th>
					<th>Entity</th>
					<th>Entity ID</th>
					<th>Status</th>
					<th>Client IP</th>
					<th class="text-right">Payload</th>
				</tr>
			</thead>
			<tbody>
				{#if loading}
					<tr>
						<td colspan="7" class="text-center" style="padding: 3rem; color: var(--text-muted);">
							Loading audit logs...
						</td>
					</tr>
				{:else if error}
					<tr>
						<td colspan="7" class="text-center" style="padding: 2rem; color: var(--rose);">
							⚠️ {error}
						</td>
					</tr>
				{:else if logs.length === 0}
					<tr>
						<td colspan="7" class="text-center" style="padding: 3rem; color: var(--text-muted);">
							No audit logs recorded yet.
						</td>
					</tr>
				{:else}
					{#each logs as log}
						<tr>
							<td style="color: var(--text-secondary); font-size: 0.8125rem; font-family: var(--font-mono);">
								{new Date(log.created_at).toLocaleString('en-IN')}
							</td>
							<td>
								<span style="font-family: var(--font-mono); font-weight: 600; font-size: 0.8125rem; color: #a5b4fc;">
									{log.action}
								</span>
							</td>
							<td style="font-size: 0.8125rem;">{log.entity_type}</td>
							<td style="font-family: var(--font-mono); font-size: 0.75rem; color: var(--text-muted);">
								{log.entity_id ? log.entity_id.slice(0, 12) + '...' : '-'}
							</td>
							<td>
								{#if log.status === 'SUCCESS'}
									<span class="badge" style="background: rgba(16, 185, 129, 0.15); color: #34d399;">SUCCESS</span>
								{:else if log.status === 'FORBIDDEN'}
									<span class="badge" style="background: rgba(244, 63, 94, 0.2); color: #fb7185; border: 1px solid rgba(244, 63, 94, 0.4);">
										FORBIDDEN
									</span>
								{:else}
									<span class="badge" style="background: rgba(245, 158, 11, 0.15); color: #fbbf24;">{log.status}</span>
								{/if}
							</td>
							<td style="font-family: var(--font-mono); font-size: 0.75rem; color: var(--text-secondary);">
								{log.ip_address || '-'}
							</td>
							<td class="text-right">
								<button onclick={() => (selectedLog = log)} class="btn btn-secondary btn-sm">
									Inspect JSON
								</button>
							</td>
						</tr>
					{/each}
				{/if}
			</tbody>
		</table>
	</div>
</div>

<!-- Inspect Payload Modal -->
{#if selectedLog}
	<div class="modal-backdrop">
		<div class="modal-card animate-fade-in" style="max-width: 700px;">
			<div class="modal-header">
				<h3 style="font-size: 1.125rem;">Audit Event: {selectedLog.action}</h3>
				<button onclick={() => (selectedLog = null)} style="background: none; border: none; color: var(--text-muted); cursor: pointer; font-size: 1.25rem;">✕</button>
			</div>
			<div class="modal-body" style="display: flex; flex-direction: column; gap: 1rem;">
				<div style="font-size: 0.8125rem; color: var(--text-secondary); display: flex; flex-direction: column; gap: 0.25rem;">
					<div><strong>Timestamp:</strong> {new Date(selectedLog.created_at).toLocaleString('en-IN')}</div>
					<div><strong>Status:</strong> {selectedLog.status}</div>
					<div><strong>Client IP:</strong> {selectedLog.ip_address}</div>
					<div><strong>User Agent:</strong> {selectedLog.user_agent}</div>
				</div>

				{#if selectedLog.old_values}
					<div>
						<span style="font-size: 0.75rem; font-weight: 700; color: var(--text-muted); text-transform: uppercase;">Previous Values:</span>
						<pre style="background: rgba(0,0,0,0.4); padding: 0.75rem; border-radius: var(--radius-md); font-size: 0.75rem; overflow-x: auto; color: #f43f5e; margin-top: 0.25rem;">{JSON.stringify(selectedLog.old_values, null, 2)}</pre>
					</div>
				{/if}

				{#if selectedLog.new_values}
					<div>
						<span style="font-size: 0.75rem; font-weight: 700; color: var(--text-muted); text-transform: uppercase;">New Values / Submitted State:</span>
						<pre style="background: rgba(0,0,0,0.4); padding: 0.75rem; border-radius: var(--radius-md); font-size: 0.75rem; overflow-x: auto; color: #34d399; margin-top: 0.25rem;">{JSON.stringify(selectedLog.new_values, null, 2)}</pre>
					</div>
				{/if}
			</div>
			<div class="modal-footer">
				<button type="button" onclick={() => (selectedLog = null)} class="btn btn-secondary">Close</button>
			</div>
		</div>
	</div>
{/if}
