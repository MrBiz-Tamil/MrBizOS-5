<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';

	interface BusinessUnit {
		id: string;
		name: string;
		code: string;
		tagline: string;
		color: string;
		icon: string;
		route: string;
		preset: string;
	}

	const units: BusinessUnit[] = [
		{ id: 'travel', name: 'Travel & Visa OS', code: 'TRAVEL', tagline: 'Tours, PNR & DMC Vouchers', color: '#06b6d4', icon: '✈️', route: '/travel', preset: 'TRAVEL' },
		{ id: 'healthcare', name: 'Smart Healthcare', code: 'HEALTH', tagline: 'Doctors, Telehealth & EMR', color: '#10b981', icon: '🏥', route: '/healthcare', preset: 'HEALTHCARE' },
		{ id: 'billing', name: 'Finance & Invoicing', code: 'FINANCE', tagline: 'GST Billing & Ledger', color: '#6366f1', icon: '💳', route: '/invoices', preset: 'FULL' },
		{ id: 'realestate', name: 'Real Estate Deals', code: 'REALESTATE', tagline: 'Properties & Site Visits', color: '#f59e0b', icon: '🏢', route: '/realestate', preset: 'REALESTATE' },
		{ id: 'ecommerce', name: 'E-Commerce & POD', code: 'ECOM', tagline: 'Print-on-Demand & Orders', color: '#8b5cf6', icon: '🛍️', route: '/ecommerce', preset: 'COMMERCE' },
		{ id: 'consulting', name: 'Software Consulting', code: 'CONSULTING', tagline: 'SOWs, Sprints & Timesheets', color: '#ec4899', icon: '💻', route: '/consulting', preset: 'PROJECTS' },
		{ id: 'workflow', name: 'AI Missions & Approvals', code: 'APPROVALS', tagline: 'Autonomous Agents & Tasks', color: '#38bdf8', icon: '🤖', route: '/workflow', preset: 'OPERATIONS' },
	];

	let isOpen = $state(false);
	let currentUnitId = $state('travel');
	let currentUnit = $derived(units.find(u => u.id === currentUnitId) || units[0]);

	// Route-aware synchronization
	$effect(() => {
		const path = $page.url.pathname;
		if (path.startsWith('/travel') || path.startsWith('/journeys') || path.startsWith('/bpo-voice')) {
			currentUnitId = 'travel';
		} else if (path.startsWith('/healthcare')) {
			currentUnitId = 'healthcare';
		} else if (path.startsWith('/realestate')) {
			currentUnitId = 'realestate';
		} else if (path.startsWith('/ecommerce')) {
			currentUnitId = 'ecommerce';
		} else if (path.startsWith('/consulting')) {
			currentUnitId = 'consulting';
		} else if (path.startsWith('/workflow')) {
			currentUnitId = 'workflow';
		} else if (path.startsWith('/billing') || path.startsWith('/pos')) {
			// Retain current unit or default to travel/billing
			const saved = typeof localStorage !== 'undefined' ? localStorage.getItem('mrbiz_active_unit') : null;
			if (saved && units.some(u => u.id === saved)) {
				currentUnitId = saved;
			}
		}
	});

	onMount(() => {
		const saved = localStorage.getItem('mrbiz_active_unit');
		if (saved && units.some(u => u.id === saved)) {
			currentUnitId = saved;
		}

		const handleUnitChange = (e: any) => {
			if (e.detail?.unit && units.some(u => u.id === e.detail.unit)) {
				currentUnitId = e.detail.unit;
			}
		};

		window.addEventListener('mrbiz_unit_changed', handleUnitChange);
		return () => {
			window.removeEventListener('mrbiz_unit_changed', handleUnitChange);
		};
	});

	function selectUnit(u: BusinessUnit) {
		currentUnitId = u.id;
		if (typeof localStorage !== 'undefined') {
			localStorage.setItem('mrbiz_active_unit', u.id);
			localStorage.setItem('mrbiz_workspace_preset', u.preset);
			window.dispatchEvent(new CustomEvent('mrbiz_unit_changed', { detail: { unit: u.id, preset: u.preset } }));
		}
		isOpen = false;
		goto(u.route);
	}
</script>

<div class="unit-switcher-container">
	<button
		type="button"
		class="unit-selector-btn"
		onclick={() => isOpen = !isOpen}
		aria-haspopup="true"
		aria-expanded={isOpen}
	>
		<span class="unit-badge" style="background-color: {currentUnit.color}20; color: {currentUnit.color}; border: 1px solid {currentUnit.color}40;">
			<span class="unit-icon">{currentUnit.icon}</span>
			<span class="unit-name">{currentUnit.name}</span>
		</span>
		<span class="chevron" class:rotate={isOpen}>▼</span>
	</button>

	{#if isOpen}
		<div class="dropdown-menu">
			<div class="dropdown-header">
				SWITCH BUSINESS UNIT
			</div>
			{#each units as u}
				<button
					type="button"
					class="unit-option"
					class:selected={u.id === currentUnitId}
					onclick={() => selectUnit(u)}
				>
					<span class="option-icon" style="background-color: {u.color}15; color: {u.color};">{u.icon}</span>
					<div class="option-info">
						<div class="option-title">{u.name}</div>
						<div class="option-subtitle">{u.tagline}</div>
					</div>
					{#if u.id === currentUnitId}
						<span class="active-indicator" style="background-color: {u.color};"></span>
					{/if}
				</button>
			{/each}
		</div>
	{/if}
</div>

<style>
	.unit-switcher-container {
		position: relative;
		display: inline-block;
		width: 100%;
		padding: 0.25rem 0.5rem 0.75rem 0.5rem;
	}

	.unit-selector-btn {
		width: 100%;
		display: flex;
		align-items: center;
		justify-content: space-between;
		background: rgba(255, 255, 255, 0.04);
		border: 1px solid rgba(255, 255, 255, 0.1);
		padding: 0.5rem 0.75rem;
		border-radius: 8px;
		cursor: pointer;
		transition: all 0.2s ease;
	}

	.unit-selector-btn:hover {
		background: rgba(255, 255, 255, 0.08);
		border-color: rgba(255, 255, 255, 0.2);
	}

	.unit-badge {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 0.8125rem;
		font-weight: 600;
		padding: 0.2rem 0.5rem;
		border-radius: 6px;
	}

	.unit-name {
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.chevron {
		font-size: 0.625rem;
		color: #94a3b8;
		transition: transform 0.2s ease;
	}

	.chevron.rotate {
		transform: rotate(180deg);
	}

	.dropdown-menu {
		position: absolute;
		top: 100%;
		left: 0.5rem;
		right: 0.5rem;
		background: #1e293b;
		border: 1px solid rgba(255, 255, 255, 0.15);
		border-radius: 10px;
		box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.5);
		z-index: 1000;
		padding: 0.5rem 0;
		animation: dropdownFadeIn 0.15s ease-out;
	}

	.dropdown-header {
		font-size: 0.625rem;
		font-weight: 700;
		color: #94a3b8;
		letter-spacing: 0.05em;
		padding: 0.25rem 0.75rem 0.5rem 0.75rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.06);
	}

	.unit-option {
		width: 100%;
		display: flex;
		align-items: center;
		gap: 0.75rem;
		padding: 0.5rem 0.75rem;
		background: transparent;
		border: none;
		text-align: left;
		cursor: pointer;
		transition: background 0.15s ease;
	}

	.unit-option:hover {
		background: rgba(255, 255, 255, 0.06);
	}

	.unit-option.selected {
		background: rgba(255, 255, 255, 0.1);
	}

	.option-icon {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 2rem;
		height: 2rem;
		border-radius: 6px;
		font-size: 1rem;
	}

	.option-title {
		font-size: 0.8125rem;
		font-weight: 600;
		color: #f8fafc;
	}

	.option-subtitle {
		font-size: 0.6875rem;
		color: #94a3b8;
	}

	.active-indicator {
		margin-left: auto;
		width: 6px;
		height: 6px;
		border-radius: 50%;
	}

	@keyframes dropdownFadeIn {
		from { opacity: 0; transform: translateY(-4px); }
		to { opacity: 1; transform: translateY(0); }
	}
</style>
