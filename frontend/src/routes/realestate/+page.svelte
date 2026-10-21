<script lang="ts">
	import { onMount } from 'svelte';
	import { realestateGrpc } from '$lib/grpc';

	let properties = $state<any[]>([]);
	let siteVisits = $state<any[]>([]);
	let loading = $state(true);
	let error = $state('');

	onMount(async () => {
		try {
			const [propRes, visitRes]: [any, any] = await Promise.all([
				realestateGrpc.listProperties(),
				realestateGrpc.listSiteVisits()
			]);
			properties = propRes?.properties || [];
			siteVisits = visitRes?.siteVisits || [];
		} catch (err: any) {
			error = err.message || 'Failed to load Real Estate records';
		} finally {
			loading = false;
		}
	});
</script>

<div class="realestate-dashboard">
	<div class="header">
		<div>
			<h1 class="title">Real Estate & Land Operations Hub</h1>
			<p class="subtitle">Property inventory, site visit logistics, and deal room management.</p>
		</div>
	</div>

	{#if error}
		<div class="error-banner">⚠️ {error}</div>
	{/if}

	{#if loading}
		<div class="loading-state">Fetching property portfolio...</div>
	{:else}
		<div class="grid-2">
			<!-- Available Properties -->
			<div class="card">
				<h2 class="section-title">🏢 Active Property Inventory</h2>
				<div class="list">
					{#each properties as prop}
						<div class="item-row">
							<div>
								<div class="item-title">{prop.title}</div>
								<div class="item-subtitle">📍 {prop.location} · {prop.areaSqft} Sqft ({prop.propertyType})</div>
							</div>
							<div class="item-price">₹{prop.price}</div>
						</div>
					{/each}
				</div>
			</div>

			<!-- Scheduled Site Visits -->
			<div class="card">
				<h2 class="section-title">🚗 Site Visits Scheduled</h2>
				<div class="list">
					{#each siteVisits as visit}
						<div class="item-row">
							<div>
								<div class="item-title">{visit.clientName} ({visit.clientPhone})</div>
								<div class="item-subtitle">Time: {new Date(visit.scheduledTime).toLocaleString()}</div>
							</div>
							<span class="badge status-scheduled">{visit.status}</span>
						</div>
					{/each}
				</div>
			</div>
		</div>
	{/if}
</div>

<style>
	.realestate-dashboard {
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
	.error-banner {
		background: rgba(239, 68, 68, 0.1);
		border: 1px solid #ef4444;
		color: #fca5a5;
		padding: 0.75rem 1rem;
		border-radius: 8px;
		margin-bottom: 1.5rem;
	}
	.loading-state {
		text-align: center;
		padding: 3rem;
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
		gap: 0.75rem;
	}
	.item-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0.75rem;
		background: rgba(255, 255, 255, 0.03);
		border: 1px solid rgba(255, 255, 255, 0.06);
		border-radius: 8px;
	}
	.item-title {
		font-weight: 600;
		color: #f1f5f9;
	}
	.item-subtitle {
		font-size: 0.75rem;
		color: #94a3b8;
	}
	.item-price {
		font-weight: 700;
		color: #f59e0b;
	}
	.badge {
		font-size: 0.75rem;
		padding: 0.2rem 0.5rem;
		border-radius: 4px;
	}
	.status-scheduled {
		background: rgba(245, 158, 11, 0.15);
		color: #fbbf24;
	}
</style>
