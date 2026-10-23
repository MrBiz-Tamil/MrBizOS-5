<script lang="ts">
	interface Props {
		open: boolean;
		onclose: () => void;
	}

	let { open = false, onclose }: Props = $props();

	let searchQuery = $state('');

	let apps = [
		{ id: 'crm', name: 'CRM & Sales', icon: '👥', color: '#6366f1', route: '/customers' },
		{ id: 'travel', name: 'Travel & Tours', icon: '✈️', color: '#2563eb', route: '/travel', active: true },
		{ id: 'healthcare', name: 'Healthcare Hub', icon: '🩺', color: '#10b981', route: '/healthcare' },
		{ id: 'finance', name: 'Finance & Accounts', icon: '₹', color: '#059669', route: '/accounting' },
		{ id: 'projects', name: 'Project Mgmt', icon: '📁', color: '#8b5cf6', route: '/projects' },
		{ id: 'realestate', name: 'Real Estate OS', icon: '🏠', color: '#f43f5e', route: '/realestate' },
		{ id: 'ecommerce', name: 'E-commerce Store', icon: '🛍️', color: '#f59e0b', route: '/ecommerce' },
		{ id: 'ai', name: 'AI Voice & BPO', icon: '🤖', color: '#a855f7', route: '/bpo-voice' },
		{ id: 'hrms', name: 'HR & Employees', icon: '👔', color: '#3b82f6', route: '/hrms' },
		{ id: 'docs', name: 'Documents & SOW', icon: '📄', color: '#10b981', route: '/quotations' },
		{ id: 'reports', name: 'BI & Reports', icon: '📊', color: '#0ea5e9', route: '/billing' },
		{ id: 'settings', name: 'System Settings', icon: '⚙️', color: '#64748b', route: '/settings' }
	];

	let filteredApps = $derived(
		apps.filter(a => a.name.toLowerCase().includes(searchQuery.toLowerCase()))
	);
</script>

{#if open}
	<div class="modal-backdrop" onclick={(e) => { if (e.target === e.currentTarget) onclose(); }}>
		<div class="modal-card animate-in" role="dialog" aria-modal="true">
			<!-- Header -->
			<div class="modal-header">
				<h3 class="modal-title">Application Launcher</h3>
				<button type="button" class="close-btn" onclick={onclose}>✕</button>
			</div>

			<!-- Search -->
			<div class="search-wrap">
				<span class="search-icon">🔍</span>
				<input 
					type="text" 
					class="search-input" 
					placeholder="Search applications, modules and business units..." 
					bind:value={searchQuery}
				/>
			</div>

			<!-- 3x3 App Grid -->
			<div class="apps-grid">
				{#each filteredApps as app}
					<a 
						href={app.route} 
						class="app-card" 
						class:active-app={app.active}
						onclick={onclose}
					>
						<div class="app-icon-box" style="background: {app.color}15; color: {app.color};">
							{app.icon}
						</div>
						<span class="app-name">{app.name}</span>
						{#if app.active}
							<span class="app-current-pill">Current</span>
						{/if}
					</a>
				{/each}
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
		max-width: 580px;
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
		padding: 1.25rem 1.5rem 0.5rem 1.5rem;
		display: flex;
		justify-content: space-between;
		align-items: center;
	}

	.modal-title { font-size: 1.15rem; font-weight: 800; margin: 0; }
	.close-btn { background: none; border: none; font-size: 1.25rem; color: #64748b; cursor: pointer; }

	.search-wrap {
		position: relative;
		padding: 0.75rem 1.5rem;
	}

	.search-icon {
		position: absolute;
		left: 2.25rem;
		top: 50%;
		transform: translateY(-50%);
		font-size: 0.9rem;
		color: #94a3b8;
	}

	.search-input {
		width: 100%;
		padding: 0.6rem 0.85rem 0.6rem 2.2rem;
		border: 1px solid #cbd5e1;
		border-radius: 8px;
		font-size: 0.875rem;
		color: #0f172a;
		background: #f8fafc;
	}
	.search-input:focus { outline: none; border-color: #2563eb; background: #ffffff; }

	.apps-grid {
		display: grid;
		grid-template-columns: repeat(4, 1fr);
		gap: 0.75rem;
		padding: 0.5rem 1.5rem 1.5rem 1.5rem;
	}

	@media (max-width: 600px) {
		.apps-grid { grid-template-columns: repeat(2, 1fr); }
	}

	.app-card {
		background: #ffffff;
		border: 1px solid #e2e8f0;
		border-radius: 10px;
		padding: 1rem 0.5rem;
		display: flex;
		flex-direction: column;
		align-items: center;
		text-align: center;
		gap: 0.5rem;
		text-decoration: none;
		transition: all 0.15s ease;
		position: relative;
	}
	.app-card:hover {
		border-color: #94a3b8;
		transform: translateY(-2px);
		box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.05);
	}
	.app-card.active-app {
		border-color: #2563eb;
		background: #eff6ff;
	}

	.app-icon-box {
		width: 44px;
		height: 44px;
		border-radius: 10px;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 1.4rem;
	}

	.app-name {
		font-size: 0.75rem;
		font-weight: 700;
		color: #0f172a;
	}

	.app-current-pill {
		position: absolute;
		top: 6px;
		right: 6px;
		font-size: 0.6rem;
		font-weight: 800;
		background: #2563eb;
		color: #ffffff;
		padding: 0.1rem 0.35rem;
		border-radius: 4px;
	}
</style>
