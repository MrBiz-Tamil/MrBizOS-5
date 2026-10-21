<script lang="ts">
	import { onMount } from 'svelte';
	import { travelGrpc } from '$lib/grpc';

	let packages = $state<any[]>([]);
	let applications = $state<any[]>([]);
	let loading = $state(true);
	let error = $state('');

	onMount(async () => {
		try {
			const [pkgRes, appRes]: [any, any] = await Promise.all([
				travelGrpc.listTourPackages(),
				travelGrpc.listVisaApplications()
			]);
			packages = pkgRes?.packages || [];
			applications = appRes?.applications || [];
		} catch (err: any) {
			error = err.message || 'Failed to load Travel & Visa data';
		} finally {
			loading = false;
		}
	});
</script>

<div class="travel-dashboard">
	<div class="header">
		<div>
			<h1 class="title">Travel, Tours & Visa Management OS</h1>
			<p class="subtitle">Global itinerary planning, immigration checklists, and visa lifecycle tracking.</p>
		</div>
	</div>

	{#if error}
		<div class="error-banner">⚠️ {error}</div>
	{/if}

	{#if loading}
		<div class="loading-state">Loading packages & visa records...</div>
	{:else}
		<div class="grid-2">
			<!-- Tour Packages -->
			<div class="card">
				<h2 class="section-title">✈️ Featured Tour Packages</h2>
				<div class="list">
					{#each packages as pkg}
						<div class="item-row">
							<div>
								<div class="item-title">{pkg.title}</div>
								<div class="item-subtitle">📍 {pkg.destination} · {pkg.durationDays} Days</div>
							</div>
							<div class="item-price">₹{pkg.price}</div>
						</div>
					{/each}
				</div>
			</div>

			<!-- Visa Applications Pipeline -->
			<div class="card">
				<h2 class="section-title">🛂 Immigration & Visa Pipeline</h2>
				<div class="list">
					{#each applications as app}
						<div class="item-row">
							<div>
								<div class="item-title">{app.applicantName}</div>
								<div class="item-subtitle">Passport: {app.passportNumber} · {app.country} ({app.visaType})</div>
							</div>
							<span class="badge status-{app.status.toLowerCase()}">{app.status}</span>
						</div>
					{/each}
				</div>
			</div>
		</div>
	{/if}
</div>

<style>
	.travel-dashboard {
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
		color: #38bdf8;
	}
	.badge {
		font-size: 0.75rem;
		padding: 0.2rem 0.5rem;
		border-radius: 4px;
	}
	.status-under_review {
		background: rgba(245, 158, 11, 0.15);
		color: #fbbf24;
	}
	.status-submitted {
		background: rgba(59, 130, 246, 0.15);
		color: #60a5fa;
	}
</style>
