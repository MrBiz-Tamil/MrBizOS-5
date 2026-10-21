<script lang="ts">
	import { onMount } from 'svelte';
	import { api, type Unit } from '$lib/api';

	let units = $state<Unit[]>([]);
	let loading = $state(true);
	let error = $state('');

	// Converter Tool State
	let convertQty = $state('2');
	let fromCode = $state('TON');
	let toCode = $state('KG');
	let conversionResult = $state('');
	let conversionError = $state('');
	let converting = $state(false);

	onMount(async () => {
		try {
			units = await api.listUnits();
		} catch (err: any) {
			error = err.message || 'Failed to load units master';
		} finally {
			loading = false;
		}
	});

	async function runConversion() {
		conversionError = '';
		conversionResult = '';
		converting = true;
		try {
			const res = await api.convertUnit(convertQty, fromCode, toCode);
			conversionResult = res.result;
		} catch (err: any) {
			conversionError = err.message || 'Conversion error';
		} finally {
			converting = false;
		}
	}

	// Create Custom Unit State
	let showCreateModal = $state(false);
	let newUnit = $state({
		code: '',
		name: '',
		symbol: '',
		category: 'QUANTITY',
		base_unit_code: '',
		conversion_factor: '1',
		decimal_precision: 0
	});
	let creating = $state(false);
	let createError = $state('');

	async function handleCreateUnit() {
		creating = true;
		createError = '';
		try {
			await api.createUnit({
				code: newUnit.code.trim().toUpperCase(),
				name: newUnit.name.trim(),
				symbol: newUnit.symbol.trim() || newUnit.code.trim(),
				category: newUnit.category,
				base_unit_code: newUnit.base_unit_code.trim().toUpperCase() || undefined,
				conversion_factor: parseFloat(newUnit.conversion_factor) || 1,
				decimal_precision: Number(newUnit.decimal_precision) || 0
			});
			showCreateModal = false;
			newUnit = {
				code: '',
				name: '',
				symbol: '',
				category: 'QUANTITY',
				base_unit_code: '',
				conversion_factor: '1',
				decimal_precision: 0
			};
			units = await api.listUnits();
		} catch (err: any) {
			createError = err.message || 'Failed to create unit';
		} finally {
			creating = false;
		}
	}

	let unitsByCategory = $derived.by(() => {
		const map: Record<string, Unit[]> = {};
		for (const u of units) {
			if (!map[u.category]) map[u.category] = [];
			map[u.category].push(u);
		}
		return map;
	});
</script>

<svelte:head>
	<title>Units & Measurement Engine - SaaS Billing</title>
</svelte:head>

<div style="display: flex; flex-direction: column; gap: 2rem;">
	<div style="display: flex; justify-content: space-between; align-items: flex-start; flex-wrap: wrap; gap: 1rem;">
		<div>
			<h1 style="font-size: 1.75rem; margin-bottom: 0.25rem;">Centralized Units & Measurement Engine</h1>
			<p style="font-size: 0.875rem; color: var(--text-secondary);">
				Central system of standardized measurement units across Weight, Volume, Length, Area, and Quantity. Custom units can be added for packaging/trade.
			</p>
		</div>
		<button class="btn btn-primary" onclick={() => (showCreateModal = true)}>
			+ Add Custom Unit
		</button>
	</div>

	<!-- Interactive Live Unit Converter Tool -->
	<div class="card" style="border-color: var(--border-accent); background: radial-gradient(circle at top left, rgba(99, 102, 241, 0.12), rgba(17, 24, 39, 0.9)); padding: 1.75rem;">
		<div style="display: flex; align-items: center; gap: 0.5rem; margin-bottom: 1.25rem;">
			<span style="font-size: 1.5rem;">⚡</span>
			<div>
				<h3 style="font-size: 1.125rem;">Live Decimal Conversion Engine</h3>
				<p style="font-size: 0.75rem; color: var(--text-secondary);">Verify server-side decimal conversion math and category compatibility rules.</p>
			</div>
		</div>

		<form onsubmit={(e) => { e.preventDefault(); runConversion(); }} style="display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)) auto; gap: 1rem; align-items: flex-end;">
			<div class="form-group" style="margin-bottom: 0;">
				<label class="form-label" for="conv-qty">Quantity</label>
				<input id="conv-qty" type="number" step="any" class="form-input font-mono" bind:value={convertQty} required />
			</div>

			<div class="form-group" style="margin-bottom: 0;">
				<label class="form-label" for="from-unit">From Unit</label>
				<select id="from-unit" class="form-select" bind:value={fromCode}>
					{#each units as u}
						<option value={u.code}>{u.code} ({u.name} - {u.category})</option>
					{/each}
				</select>
			</div>

			<div class="form-group" style="margin-bottom: 0;">
				<label class="form-label" for="to-unit">To Unit</label>
				<select id="to-unit" class="form-select" bind:value={toCode}>
					{#each units as u}
						<option value={u.code}>{u.code} ({u.name} - {u.category})</option>
					{/each}
				</select>
			</div>

			<button type="submit" class="btn btn-primary" style="height: 42px;" disabled={converting}>
				{converting ? 'Converting...' : 'Convert →'}
			</button>
		</form>

		{#if conversionResult}
			<div style="margin-top: 1.25rem; padding: 1rem; background: rgba(16, 185, 129, 0.15); border: 1px solid rgba(16, 185, 129, 0.3); border-radius: var(--radius-md); display: flex; align-items: center; gap: 0.75rem;">
				<span style="font-size: 1.25rem;">✓</span>
				<div>
					<span style="font-size: 0.8125rem; color: var(--text-secondary);">Calculated Result:</span>
					<div style="font-size: 1.25rem; font-weight: 700; color: #34d399; font-family: var(--font-mono);">
						{convertQty} {fromCode} = {conversionResult} {toCode}
					</div>
				</div>
			</div>
		{/if}

		{#if conversionError}
			<div style="margin-top: 1.25rem; padding: 1rem; background: rgba(244, 63, 94, 0.15); border: 1px solid rgba(244, 63, 94, 0.3); border-radius: var(--radius-md); display: flex; align-items: center; gap: 0.75rem; color: #fb7185;">
				<span style="font-size: 1.25rem;">🚫</span>
				<div>
					<div style="font-weight: 600; font-size: 0.875rem;">Conversion Rejected by Server Engine</div>
					<div style="font-size: 0.8125rem;">{conversionError}</div>
				</div>
			</div>
		{/if}
	</div>

	<!-- Centralized Units Master List -->
	{#if loading}
		<div style="padding: 3rem; text-align: center; color: var(--text-muted);">
			Loading units master...
		</div>
	{:else}
		<div style="display: flex; flex-direction: column; gap: 1.5rem;">
			{#each Object.entries(unitsByCategory) as [category, list]}
				<div class="card" style="padding: 1.25rem;">
					<div style="display: flex; align-items: center; gap: 0.5rem; margin-bottom: 1rem;">
						<h3 style="font-size: 1.125rem;">{category} UNITS</h3>
						<span style="font-size: 0.75rem; background: rgba(255,255,255,0.06); padding: 0.2rem 0.5rem; border-radius: var(--radius-sm); color: var(--text-secondary);">
							Base: {list[0]?.base_unit_code}
						</span>
					</div>

					<div class="table-container" style="border: none;">
						<table class="data-table">
							<thead>
								<tr>
									<th>Code</th>
									<th>Unit Name</th>
									<th>Symbol</th>
									<th>Base Unit</th>
									<th class="text-right">Conversion Factor (to Base)</th>
									<th class="text-right">Precision</th>
									<th>Type</th>
								</tr>
							</thead>
							<tbody>
								{#each list as u}
									<tr>
										<td style="font-family: var(--font-mono); color: #818cf8; font-weight: 600;">{u.code}</td>
										<td style="font-weight: 500;">{u.name}</td>
										<td style="color: var(--text-secondary);">{u.symbol}</td>
										<td>{u.base_unit_code}</td>
										<td class="text-right font-mono">{parseFloat(u.conversion_factor)}</td>
										<td class="text-right font-mono">{u.decimal_precision} decimals</td>
										<td>
											{#if u.is_system}
												<span style="font-size: 0.6875rem; background: rgba(99, 102, 241, 0.15); color: #a5b4fc; padding: 0.15rem 0.4rem; border-radius: var(--radius-sm); font-weight: 600;">SYSTEM</span>
											{:else}
												<span style="font-size: 0.6875rem; background: rgba(16, 185, 129, 0.15); color: #34d399; padding: 0.15rem 0.4rem; border-radius: var(--radius-sm); font-weight: 600;">CUSTOM</span>
											{/if}
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				</div>
			{/each}
		</div>
	{/if}

	<!-- Add Custom Unit Modal -->
	{#if showCreateModal}
		<div style="position: fixed; inset: 0; background: rgba(0, 0, 0, 0.7); backdrop-filter: blur(4px); display: flex; align-items: center; justify-content: center; z-index: 50; padding: 1rem;">
			<div class="card" style="width: 100%; max-width: 520px; padding: 1.75rem; border-color: var(--border-accent); box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.5);">
				<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1.25rem;">
					<h3 style="font-size: 1.25rem; font-weight: 700;">Add Custom Unit</h3>
					<button
						class="btn btn-ghost"
						style="padding: 0.25rem 0.5rem;"
						onclick={() => (showCreateModal = false)}>✕</button
					>
				</div>

				{#if createError}
					<div style="background: rgba(244, 63, 94, 0.15); border: 1px solid rgba(244, 63, 94, 0.3); color: #fb7185; padding: 0.75rem; border-radius: var(--radius-md); font-size: 0.8125rem; margin-bottom: 1rem;">
						{createError}
					</div>
				{/if}

				<form onsubmit={(e) => { e.preventDefault(); handleCreateUnit(); }} style="display: flex; flex-direction: column; gap: 1rem;">
					<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="unit-code">Unit Code (e.g. BAG, BOX)</label>
							<input
								id="unit-code"
								type="text"
								class="form-input font-mono"
								placeholder="BAG"
								bind:value={newUnit.code}
								required
								style="text-transform: uppercase;"
							/>
						</div>
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="unit-name">Display Name</label>
							<input
								id="unit-name"
								type="text"
								class="form-input"
								placeholder="Bag (50kg)"
								bind:value={newUnit.name}
								required
							/>
						</div>
					</div>

					<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="unit-symbol">Symbol (Optional)</label>
							<input
								id="unit-symbol"
								type="text"
								class="form-input"
								placeholder="bag"
								bind:value={newUnit.symbol}
							/>
						</div>
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="unit-category">Measurement Category</label>
							<select id="unit-category" class="form-select" bind:value={newUnit.category}>
								<option value="QUANTITY">QUANTITY</option>
								<option value="WEIGHT">WEIGHT</option>
								<option value="VOLUME">VOLUME</option>
								<option value="LENGTH">LENGTH</option>
								<option value="AREA">AREA</option>
							</select>
						</div>
					</div>

					<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="unit-base">Base Unit (e.g. KG, PCS)</label>
							<input
								id="unit-base"
								type="text"
								class="form-input font-mono"
								placeholder="PCS"
								bind:value={newUnit.base_unit_code}
								style="text-transform: uppercase;"
							/>
						</div>
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="unit-factor">Conversion Factor</label>
							<input
								id="unit-factor"
								type="number"
								step="any"
								class="form-input font-mono"
								placeholder="1"
								bind:value={newUnit.conversion_factor}
								required
							/>
						</div>
					</div>

					<div class="form-group" style="margin-bottom: 0;">
						<label class="form-label" for="unit-prec">Decimal Precision (0 - 4)</label>
						<input
							id="unit-prec"
							type="number"
							min="0"
							max="4"
							class="form-input font-mono"
							bind:value={newUnit.decimal_precision}
							required
						/>
					</div>

					<div style="display: flex; justify-content: flex-end; gap: 0.75rem; margin-top: 0.5rem;">
						<button
							type="button"
							class="btn btn-ghost"
							onclick={() => (showCreateModal = false)}>Cancel</button
						>
						<button type="submit" class="btn btn-primary" disabled={creating}>
							{creating ? 'Saving...' : 'Create Unit'}
						</button>
					</div>
				</form>
			</div>
		</div>
	{/if}
</div>
