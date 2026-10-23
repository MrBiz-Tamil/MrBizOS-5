<script lang="ts">
	import { moduleStore, MODULE_CATALOG, type ModuleDescriptor } from '$lib/stores/modules.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';

	interface Props {
		open: boolean;
		targetModuleId?: string;
		onclose: () => void;
		onactivated?: (moduleId: string) => void;
	}

	let { open = false, targetModuleId, onclose, onactivated }: Props = $props();

	// Modal State
	let selectedModuleId = $state<string>('healthcare');
	let activeModalView = $state<'focused' | 'catalog'>('focused');
	let selectedBillingCycle = $state<'MONTHLY' | 'ANNUAL'>('MONTHLY');
	let paymentMethod = $state<'UPI' | 'CARD' | 'NETBANKING'>('UPI');
	let isProcessingPayment = $state(false);
	let filterCategory = $state<'ALL' | 'VERTICAL' | 'AI_GROWTH' | 'OPERATIONS' | 'CORE'>('ALL');

	// Synchronize when targetModuleId changes
	$effect(() => {
		if (targetModuleId) {
			selectedModuleId = targetModuleId;
			activeModalView = 'focused';
		} else if (!selectedModuleId) {
			selectedModuleId = 'healthcare';
		}
	});

	let activeModule = $derived(
		moduleStore.getModuleDescriptor(selectedModuleId) || MODULE_CATALOG[0]
	);

	let currentLicense = $derived(moduleStore.getLicense(selectedModuleId));
	let isAlreadyActive = $derived(moduleStore.isModuleActive(selectedModuleId));

	let currentPrice = $derived(
		selectedBillingCycle === 'ANNUAL' ? activeModule.priceAnnual : activeModule.priceMonthly
	);

	let filteredModules = $derived(
		MODULE_CATALOG.filter((m) => filterCategory === 'ALL' || m.category === filterCategory)
	);

	function handlePayAndActivate() {
		isProcessingPayment = true;
		setTimeout(() => {
			moduleStore.activateModule(selectedModuleId, selectedBillingCycle, paymentMethod);
			isProcessingPayment = false;
			onactivated?.(selectedModuleId);
			onclose();
		}, 600);
	}

	function handleStartTrial() {
		moduleStore.startTrial(selectedModuleId);
		onactivated?.(selectedModuleId);
		onclose();
	}

	function handleDeactivate() {
		moduleStore.deactivateModule(selectedModuleId);
	}

	function selectModuleToActivate(mod: ModuleDescriptor) {
		selectedModuleId = mod.id;
		activeModalView = 'focused';
	}
</script>

{#if open}
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div class="modal-backdrop animate-fade-in" onclick={(e) => { if (e.target === e.currentTarget) onclose(); }}>
		<div class="modal-card activation-modal-card" role="dialog" aria-modal="true" aria-labelledby="modal-title">
			
			<!-- Modal Header Bar -->
			<div class="modal-header">
				<div class="modal-header-left">
					<span class="header-icon">🧩</span>
					<div>
						<h2 id="modal-title" class="header-title">
							{activeModalView === 'focused' ? `Activate Module: ${activeModule.name}` : 'Module Store & Custom Add-Ons'}
						</h2>
						<span class="header-subtitle">
							Modular licensing · Subscribe and pay only for the business units you use
						</span>
					</div>
				</div>

				<div class="modal-header-actions">
					<div class="tab-pill-toggle">
						<button
							type="button"
							class="pill-tab"
							class:active={activeModalView === 'focused'}
							onclick={() => (activeModalView = 'focused')}
						>
							Module Detail
						</button>
						<button
							type="button"
							class="pill-tab"
							class:active={activeModalView === 'catalog'}
							onclick={() => (activeModalView = 'catalog')}
						>
							All Modules ({MODULE_CATALOG.length})
						</button>
					</div>

					<button type="button" class="close-btn" onclick={onclose} aria-label="Close modal">
						✕
					</button>
				</div>
			</div>

			<!-- Modal Body: Focused Module Activation View -->
			{#if activeModalView === 'focused'}
				<div class="modal-body focused-body">
					<!-- Hero Banner for Selected Module -->
					<div class="module-hero-banner" style={`border-left: 4px solid ${activeModule.accentVar};`}>
						<div class="hero-icon-box" style={`background: ${activeModule.accentVar}15; color: ${activeModule.accentVar};`}>
							{activeModule.icon}
						</div>
						<div class="hero-text">
							<div class="hero-top-row">
								<h3 class="hero-title">{activeModule.name}</h3>
								<div class="hero-status-tag">
									{#if isAlreadyActive}
										{#if currentLicense?.status === 'TRIAL'}
											<Badge variant="warning" label="14-DAY TRIAL ACTIVE" icon="⏳" size="sm" />
										{:else}
											<Badge variant="success" label="ACTIVE LICENSE" icon="✓" size="sm" />
										{/if}
									{:else}
										<Badge variant="locked" label="NOT ACTIVATED" icon="🔒" size="sm" />
									{/if}
								</div>
							</div>
							<p class="hero-tagline">{activeModule.tagline}</p>
							<p class="hero-desc">{activeModule.description}</p>
						</div>
					</div>

					<!-- Content Grid: Features vs Checkout -->
					<div class="activation-split-grid">
						<!-- Left: Included Business Capabilities -->
						<div class="features-card card">
							<h4 class="section-title">✨ What's Unlocked in {activeModule.name}</h4>
							<ul class="features-list">
								{#each activeModule.features as feat}
									<li class="feature-item">
										<span class="check-glyph">✓</span>
										<span>{feat}</span>
									</li>
								{/each}
							</ul>

							<div class="isolation-notice">
								<span class="notice-icon">🛡️</span>
								<div class="notice-text">
									<strong>Zero Bundling Guarantee:</strong>
									You are subscribing strictly to <em>{activeModule.name}</em>.
									No unselected modules will be billed or altered.
								</div>
							</div>
						</div>

						<!-- Right: Pricing & Payment Checkout -->
						<div class="checkout-card card">
							<h4 class="section-title">💳 Subscription Plan</h4>

							{#if activeModule.isCore}
								<div class="core-free-box">
									<span class="core-icon">🎁</span>
									<div>
										<strong>Core Platform Standard</strong>
										<p class="text-caption text-secondary" style="margin: 0;">
											This module is part of your baseline MRBiz OS and is included at no extra cost.
										</p>
									</div>
								</div>
							{:else}
								<!-- Billing Cycle Toggle -->
								<div class="billing-cycle-switch">
									<button
										type="button"
										class="cycle-btn"
										class:active={selectedBillingCycle === 'MONTHLY'}
										onclick={() => (selectedBillingCycle = 'MONTHLY')}
									>
										<span>Monthly</span>
										<strong>₹{activeModule.priceMonthly.toLocaleString('en-IN')}/mo</strong>
									</button>

									<button
										type="button"
										class="cycle-btn"
										class:active={selectedBillingCycle === 'ANNUAL'}
										onclick={() => (selectedBillingCycle = 'ANNUAL')}
									>
										<div class="save-badge">Save 17%</div>
										<span>Annual Billing</span>
										<strong>₹{activeModule.priceAnnual.toLocaleString('en-IN')}/yr</strong>
									</button>
								</div>

								<!-- Payment Method Selector -->
								<div class="payment-methods-box">
									<span class="sub-label">Payment Method:</span>
									<div class="pay-method-options">
										<label class="pay-option" class:selected={paymentMethod === 'UPI'}>
											<input type="radio" name="pay-method" value="UPI" bind:group={paymentMethod} />
											<span>⚡ UPI Instant QR</span>
										</label>
										<label class="pay-option" class:selected={paymentMethod === 'CARD'}>
											<input type="radio" name="pay-method" value="CARD" bind:group={paymentMethod} />
											<span>💳 Debit / Credit Card</span>
										</label>
										<label class="pay-option" class:selected={paymentMethod === 'NETBANKING'}>
											<input type="radio" name="pay-method" value="NETBANKING" bind:group={paymentMethod} />
											<span>🏦 Net Banking / NEFT</span>
										</label>
									</div>
								</div>

								<!-- Total Summary Box -->
								<div class="checkout-total-box">
									<span class="text-secondary text-caption">Due Today for {activeModule.name}:</span>
									<strong class="total-price font-mono text-brand">
										₹{currentPrice.toLocaleString('en-IN')}
										<span class="cycle-note">{selectedBillingCycle === 'ANNUAL' ? '/year' : '/month'}</span>
									</strong>
								</div>

								<!-- Action Buttons -->
								<div class="checkout-actions">
									{#if !isAlreadyActive}
										<button
											type="button"
											class="btn btn-primary btn-lg pay-btn"
											onclick={handlePayAndActivate}
											disabled={isProcessingPayment}
										>
											{#if isProcessingPayment}
												<span>Processing Payment...</span>
											{:else}
												<span>⚡ Pay ₹{currentPrice.toLocaleString('en-IN')} &amp; Activate Module</span>
											{/if}
										</button>

										<button
											type="button"
											class="btn btn-outline trial-btn"
											onclick={handleStartTrial}
										>
											🎁 Start 14-Day Free Trial (No Card Required)
										</button>
									{:else}
										<div class="active-badge-notice">
											<span style="color: var(--status-success); font-weight: 700;">✓ This module is currently active!</span>
											<span class="text-caption text-secondary">
												Valid until: {currentLicense?.expiresAt ? new Date(currentLicense.expiresAt).toLocaleDateString('en-IN') : 'Ongoing'}
											</span>
										</div>

										<button
											type="button"
											class="btn btn-danger btn-sm"
											onclick={handleDeactivate}
										>
											Pause / Deactivate Module
										</button>
									{/if}
								</div>
							{/if}
						</div>
					</div>
				</div>
			{/if}

			<!-- Modal Body: Catalog of All 16 Modules -->
			{#if activeModalView === 'catalog'}
				<div class="modal-body catalog-body">
					<!-- Category Filter Tabs -->
					<div class="catalog-filter-bar">
						<span class="filter-label">Filter:</span>
						<div class="filter-pills">
							{#each ['ALL', 'VERTICAL', 'OPERATIONS', 'AI_GROWTH', 'CORE'] as cat}
								<button
									type="button"
									class="cat-pill"
									class:active={filterCategory === cat}
									onclick={() => (filterCategory = cat as any)}
								>
									{cat === 'ALL' ? '🌐 All Modules' : cat === 'VERTICAL' ? '🏥 Industry Verticals' : cat === 'AI_GROWTH' ? '🤖 AI & Growth' : cat === 'OPERATIONS' ? '📦 Operations' : '⚙️ Core'}
								</button>
							{/each}
						</div>
					</div>

					<!-- Modules Grid -->
					<div class="modules-catalog-grid">
						{#each filteredModules as mod (mod.id)}
							{@const active = moduleStore.isModuleActive(mod.id)}
							{@const lic = moduleStore.getLicense(mod.id)}
							
							<div class="card module-catalog-card" class:card-active={active}>
								<div class="card-head">
									<div class="module-icon-sm" style={`background: ${mod.accentVar}15; color: ${mod.accentVar};`}>
										{mod.icon}
									</div>
									<div class="card-status">
										{#if active}
											{#if lic?.status === 'TRIAL'}
												<Badge variant="warning" label="TRIAL" icon="⏳" size="sm" />
											{:else}
												<Badge variant="success" label="ACTIVE" icon="✓" size="sm" />
											{/if}
										{:else}
											<Badge variant="locked" label="LOCKED" icon="🔒" size="sm" />
										{/if}
									</div>
								</div>

								<h4 class="card-title">{mod.name}</h4>
								<p class="card-tagline">{mod.tagline}</p>

								<div class="card-footer-strip">
									<div class="price-tag font-mono">
										{#if mod.isCore}
											<span class="text-success font-bold">Included</span>
										{:else}
											<strong>₹{mod.priceMonthly.toLocaleString('en-IN')}</strong><span class="period">/mo</span>
										{/if}
									</div>

									<button
										type="button"
										class="btn btn-sm {active ? 'btn-secondary' : 'btn-primary'}"
										onclick={() => selectModuleToActivate(mod)}
									>
										{active ? 'Manage' : '⚡ Activate'}
									</button>
								</div>
							</div>
						{/each}
					</div>
				</div>
			{/if}

			<!-- Modal Footer -->
			<div class="modal-footer">
				<div class="footer-summary">
					<span class="text-caption text-secondary">
						Active Modules: <strong class="text-brand">{moduleStore.activeCount} / {MODULE_CATALOG.length}</strong>
					</span>
					<span class="footer-dot">·</span>
					<span class="text-caption text-secondary">
						Monthly Commitment: <strong class="font-mono text-brand">₹{moduleStore.totalMonthlyCommitment.toLocaleString('en-IN')}/mo</strong>
					</span>
				</div>

				<button type="button" class="btn btn-secondary" onclick={onclose}>
					Done
				</button>
			</div>
		</div>
	</div>
{/if}

<style>
	.activation-modal-card {
		max-width: 920px;
		width: 95vw;
		background: var(--surface-main);
		border: 1px solid var(--border-default);
		border-radius: var(--radius-xl);
		box-shadow: var(--shadow-lg);
		display: flex;
		flex-direction: column;
		max-height: 90vh;
		overflow: hidden;
	}

	.modal-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 1rem 1.5rem;
		border-bottom: 1px solid var(--border-default);
		background: var(--surface-secondary);
		gap: 1rem;
	}

	.modal-header-left {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}

	.header-icon {
		font-size: 1.75rem;
		line-height: 1;
	}

	.header-title {
		font-size: var(--text-card-title);
		font-weight: 700;
		color: var(--text-primary);
		line-height: 1.25;
	}

	.header-subtitle {
		font-size: var(--text-caption);
		color: var(--text-muted);
		display: block;
	}

	.modal-header-actions {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}

	.tab-pill-toggle {
		display: flex;
		background: var(--surface-main);
		border: 1px solid var(--border-default);
		border-radius: var(--radius-full);
		padding: 0.15rem;
		gap: 0.15rem;
	}

	.pill-tab {
		background: transparent;
		border: none;
		border-radius: var(--radius-full);
		padding: 0.3rem 0.75rem;
		font-size: var(--text-caption);
		font-weight: 600;
		color: var(--text-secondary);
		cursor: pointer;
		transition: all var(--transition-fast);
	}

	.pill-tab.active {
		background: var(--brand-primary);
		color: #ffffff;
	}

	.close-btn {
		background: none;
		border: none;
		color: var(--text-muted);
		font-size: 1.25rem;
		cursor: pointer;
		padding: 0.25rem 0.5rem;
		border-radius: var(--radius-sm);
		transition: color var(--transition-fast);
	}
	.close-btn:hover {
		color: var(--text-primary);
	}

	.modal-body {
		padding: 1.5rem;
		overflow-y: auto;
		display: flex;
		flex-direction: column;
		gap: 1.25rem;
	}

	/* Hero Banner */
	.module-hero-banner {
		display: flex;
		align-items: flex-start;
		gap: 1.15rem;
		background: var(--surface-secondary);
		border: 1px solid var(--border-default);
		border-radius: var(--radius-lg);
		padding: 1.25rem;
	}

	.hero-icon-box {
		width: 52px;
		height: 52px;
		border-radius: var(--radius-md);
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 1.75rem;
		flex-shrink: 0;
	}

	.hero-text {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}

	.hero-top-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.75rem;
	}

	.hero-title {
		font-size: 1.25rem;
		font-weight: 700;
		color: var(--text-primary);
	}

	.hero-tagline {
		font-size: var(--text-table);
		font-weight: 600;
		color: var(--text-primary);
		margin: 0;
	}

	.hero-desc {
		font-size: var(--text-caption);
		color: var(--text-secondary);
		margin: 0;
		line-height: 1.4;
	}

	/* Split Grid */
	.activation-split-grid {
		display: grid;
		grid-template-columns: 1.1fr 1fr;
		gap: 1.25rem;
	}

	@media (max-width: 768px) {
		.activation-split-grid {
			grid-template-columns: 1fr;
		}
	}

	.section-title {
		font-size: var(--text-card-title);
		font-weight: 700;
		color: var(--text-primary);
		margin-bottom: 0.85rem;
	}

	.features-list {
		list-style: none;
		display: flex;
		flex-direction: column;
		gap: 0.6rem;
		margin: 0 0 1.25rem 0;
		padding: 0;
	}

	.feature-item {
		display: flex;
		align-items: flex-start;
		gap: 0.5rem;
		font-size: var(--text-table);
		color: var(--text-primary);
		line-height: 1.4;
	}

	.check-glyph {
		color: var(--status-success);
		font-weight: 700;
		flex-shrink: 0;
	}

	.isolation-notice {
		display: flex;
		align-items: flex-start;
		gap: 0.65rem;
		background: var(--surface-secondary);
		border: 1px solid var(--border-default);
		border-radius: var(--radius-md);
		padding: 0.75rem;
		font-size: var(--text-caption);
		color: var(--text-secondary);
	}

	.notice-icon {
		font-size: 1.15rem;
		flex-shrink: 0;
	}

	/* Checkout Right Column */
	.billing-cycle-switch {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 0.75rem;
		margin-bottom: 1rem;
	}

	.cycle-btn {
		background: var(--surface-secondary);
		border: 1px solid var(--border-default);
		border-radius: var(--radius-md);
		padding: 0.75rem;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.25rem;
		cursor: pointer;
		position: relative;
		transition: all var(--transition-fast);
		color: var(--text-secondary);
	}

	.cycle-btn.active {
		background: var(--brand-primary-soft);
		border-color: var(--brand-primary);
		color: var(--brand-primary);
	}

	.save-badge {
		position: absolute;
		top: -8px;
		right: 8px;
		background: var(--status-success);
		color: #ffffff;
		font-size: 0.6rem;
		font-weight: 700;
		padding: 0.1rem 0.35rem;
		border-radius: var(--radius-full);
	}

	.payment-methods-box {
		margin-bottom: 1rem;
	}

	.sub-label {
		font-size: var(--text-caption);
		font-weight: 600;
		color: var(--text-secondary);
		display: block;
		margin-bottom: 0.35rem;
	}

	.pay-method-options {
		display: flex;
		flex-direction: column;
		gap: 0.35rem;
	}

	.pay-option {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.45rem 0.65rem;
		background: var(--surface-secondary);
		border: 1px solid var(--border-default);
		border-radius: var(--radius-sm);
		font-size: var(--text-caption);
		font-weight: 500;
		cursor: pointer;
		transition: all var(--transition-fast);
	}

	.pay-option.selected {
		border-color: var(--brand-primary);
		background: var(--brand-primary-soft);
		color: var(--brand-primary);
	}

	.checkout-total-box {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0.75rem;
		background: var(--surface-secondary);
		border-radius: var(--radius-md);
		border: 1px solid var(--border-default);
		margin-bottom: 1rem;
	}

	.total-price {
		font-size: 1.35rem;
	}

	.cycle-note {
		font-size: var(--text-caption);
		font-weight: 500;
		color: var(--text-secondary);
	}

	.checkout-actions {
		display: flex;
		flex-direction: column;
		gap: 0.6rem;
	}

	.pay-btn {
		width: 100%;
		font-weight: 700;
	}

	.trial-btn {
		width: 100%;
		font-size: var(--text-caption);
	}

	.core-free-box {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		background: var(--status-success-bg);
		border: 1px solid var(--status-success-border);
		color: var(--status-success);
		padding: 1rem;
		border-radius: var(--radius-md);
	}

	.core-icon {
		font-size: 1.5rem;
	}

	.active-badge-notice {
		display: flex;
		flex-direction: column;
		gap: 0.2rem;
		background: var(--status-success-bg);
		border: 1px solid var(--status-success-border);
		padding: 0.75rem;
		border-radius: var(--radius-md);
		margin-bottom: 0.5rem;
	}

	/* Catalog Grid View */
	.catalog-filter-bar {
		display: flex;
		align-items: center;
		gap: 0.65rem;
		flex-wrap: wrap;
	}

	.filter-label {
		font-size: var(--text-caption);
		font-weight: 600;
		color: var(--text-secondary);
	}

	.filter-pills {
		display: flex;
		gap: 0.35rem;
		flex-wrap: wrap;
	}

	.cat-pill {
		background: var(--surface-secondary);
		border: 1px solid var(--border-default);
		border-radius: var(--radius-full);
		padding: 0.25rem 0.65rem;
		font-size: var(--text-caption);
		font-weight: 500;
		color: var(--text-secondary);
		cursor: pointer;
		transition: all var(--transition-fast);
	}

	.cat-pill.active {
		background: var(--brand-primary);
		color: #ffffff;
		border-color: var(--brand-primary);
	}

	.modules-catalog-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
		gap: 1rem;
	}

	.module-catalog-card {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		padding: 1rem;
	}

	.module-catalog-card.card-active {
		border-color: var(--status-success-border);
	}

	.card-head {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.module-icon-sm {
		width: 36px;
		height: 36px;
		border-radius: var(--radius-sm);
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 1.25rem;
	}

	.card-title {
		font-size: var(--text-table);
		font-weight: 700;
		color: var(--text-primary);
		margin: 0;
	}

	.card-tagline {
		font-size: 0.75rem;
		color: var(--text-secondary);
		line-height: 1.35;
		margin: 0;
		display: -webkit-box;
		-webkit-line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
		flex: 1;
	}

	.card-footer-strip {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding-top: 0.5rem;
		border-top: 1px solid var(--border-default);
		margin-top: 0.25rem;
	}

	.price-tag {
		font-size: 0.85rem;
		color: var(--text-primary);
	}

	.price-tag .period {
		font-size: 0.6875rem;
		color: var(--text-muted);
	}

	/* Footer */
	.modal-footer {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0.85rem 1.5rem;
		border-top: 1px solid var(--border-default);
		background: var(--surface-secondary);
	}

	.footer-summary {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.footer-dot {
		color: var(--border-strong);
	}
</style>
