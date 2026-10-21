<script lang="ts">
	import { onMount } from 'svelte';
	import { api, type APIKey, type CustomFieldDefinition } from '$lib/api';
	import LiveLedgerStream from '$lib/components/LiveLedgerStream.svelte';

	let activeTab = $state<'api_keys' | 'custom_fields' | 'metering'>('api_keys');
	let loading = $state(true);
	let error = $state('');
	let successMsg = $state('');

	// API Keys State
	let apiKeys = $state<APIKey[]>([]);
	let showCreateKeyModal = $state(false);
	let newKeyName = $state('');
	let newKeyLiveMode = $state(true);
	let newKeyScopes = $state<string[]>(['billing:read', 'billing:write']);
	let createdSecret = $state<string | null>(null);
	let isCreatingKey = $state(false);

	// Custom Fields State
	let customFieldEntity = $state<'INVOICE' | 'CUSTOMER' | 'PRODUCT'>('INVOICE');
	let customFields = $state<CustomFieldDefinition[]>([]);
	let newFieldKey = $state('');
	let newFieldLabel = $state('');
	let newFieldType = $state<'STRING' | 'NUMBER' | 'BOOLEAN' | 'DATE' | 'ENUM'>('STRING');
	let newFieldRegex = $state('');
	let newFieldRequired = $state(false);
	let isCreatingField = $state(false);

	// Metering Simulator State
	let meterSlug = $state('api_requests');
	let meterValue = $state(100);
	let isSimulating = $state(false);

	onMount(async () => {
		await loadData();
	});

	async function loadData() {
		loading = true;
		error = '';
		try {
			const [keys, fields] = await Promise.all([
				api.listAPIKeys().catch(() => []),
				api.getCustomFields(customFieldEntity).catch(() => [])
			]);
			apiKeys = keys;
			customFields = fields;
		} catch (err: any) {
			error = err.message || 'Failed to load developer portal data';
		} finally {
			loading = false;
		}
	}

	async function handleSwitchFieldEntity(entity: 'INVOICE' | 'CUSTOMER' | 'PRODUCT') {
		customFieldEntity = entity;
		try {
			customFields = await api.getCustomFields(entity);
		} catch (e) {}
	}

	async function handleCreateKey(e: Event) {
		e.preventDefault();
		if (!newKeyName) return;
		isCreatingKey = true;
		error = '';
		try {
			const res = await api.createAPIKey({
				name: newKeyName,
				live_mode: newKeyLiveMode,
				scopes: newKeyScopes
			});
			createdSecret = res.secret;
			apiKeys = [res.api_key, ...apiKeys];
			newKeyName = '';
			successMsg = 'API Key generated successfully! Store secret securely; it will not be shown again.';
		} catch (err: any) {
			error = err.message || 'Failed to create API key';
		} finally {
			isCreatingKey = false;
		}
	}

	async function handleCreateCustomField(e: Event) {
		e.preventDefault();
		if (!newFieldKey || !newFieldLabel) return;
		isCreatingField = true;
		error = '';
		try {
			await api.createCustomField({
				entity_name: customFieldEntity,
				field_key: newFieldKey.toLowerCase().replace(/\s+/g, '_'),
				field_label: newFieldLabel,
				data_type: newFieldType,
				validation_regex: newFieldRegex,
				is_required: newFieldRequired
			});
			newFieldKey = '';
			newFieldLabel = '';
			newFieldRegex = '';
			newFieldRequired = false;
			successMsg = 'Custom field definition saved successfully!';
			customFields = await api.getCustomFields(customFieldEntity);
		} catch (err: any) {
			error = err.message || 'Failed to save custom field';
		} finally {
			isCreatingField = false;
		}
	}

	async function handleSimulateMetering() {
		isSimulating = true;
		error = '';
		try {
			await api.ingestMeteringEvent({
				meter_slug: meterSlug,
				customer_id: '00000000-0000-0000-0000-000000000000', // Simulator dummy
				value: meterValue,
				metadata: { source: 'developer_console_simulator', client: 'web' }
			});
			successMsg = `Queued ${meterValue} units to meter "${meterSlug}". Watch the live stream below!`;
		} catch (err: any) {
			error = err.message || 'Failed to ingest metering event';
		} finally {
			isSimulating = false;
		}
	}

	function copyToClipboard(text: string) {
		navigator.clipboard.writeText(text);
		alert('Copied to clipboard!');
	}
</script>

<svelte:head>
	<title>Developer & Extensibility Console - SaaS Billing</title>
</svelte:head>

<div class="space-y-6 max-w-6xl mx-auto p-2">
	<!-- Page Header -->
	<div class="flex items-center justify-between flex-wrap gap-4 border-b border-zinc-800 pb-5">
		<div>
			<h1 class="text-2xl font-bold tracking-tight text-white flex items-center gap-3">
				<span>⚡</span> Developer & Extensibility Engine
			</h1>
			<p class="text-sm text-zinc-400 mt-1">
				Manage developer API keys, custom invoice/customer metadata schemas, and real-time metering telemetry.
			</p>
		</div>

		<!-- Tab Navigation -->
		<div class="flex items-center gap-2 bg-zinc-900 border border-zinc-800 p-1 rounded-xl">
			<button
				class="px-4 py-2 text-xs font-semibold rounded-lg transition-all {activeTab === 'api_keys' ? 'bg-indigo-600 text-white shadow-lg' : 'text-zinc-400 hover:text-white'}"
				onclick={() => activeTab = 'api_keys'}
			>
				🔑 API Keys
			</button>
			<button
				class="px-4 py-2 text-xs font-semibold rounded-lg transition-all {activeTab === 'custom_fields' ? 'bg-indigo-600 text-white shadow-lg' : 'text-zinc-400 hover:text-white'}"
				onclick={() => activeTab = 'custom_fields'}
			>
				🧩 Custom Fields
			</button>
			<button
				class="px-4 py-2 text-xs font-semibold rounded-lg transition-all {activeTab === 'metering' ? 'bg-indigo-600 text-white shadow-lg' : 'text-zinc-400 hover:text-white'}"
				onclick={() => activeTab = 'metering'}
			>
				📈 Usage Metering
			</button>
		</div>
	</div>

	<!-- Flash Messages -->
	{#if error}
		<div class="p-4 rounded-xl bg-rose-500/10 border border-rose-500/20 text-rose-400 text-sm flex items-center justify-between">
			<span>{error}</span>
			<button onclick={() => error = ''} class="text-xs font-bold text-rose-300 hover:underline">Dismiss</button>
		</div>
	{/if}
	{#if successMsg}
		<div class="p-4 rounded-xl bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 text-sm flex items-center justify-between">
			<span>{successMsg}</span>
			<button onclick={() => successMsg = ''} class="text-xs font-bold text-emerald-300 hover:underline">Dismiss</button>
		</div>
	{/if}

	<!-- TAB 1: API KEYS -->
	{#if activeTab === 'api_keys'}
		<div class="space-y-6">
			<div class="flex items-center justify-between">
				<div>
					<h2 class="text-lg font-semibold text-white">Developer API Keys</h2>
					<p class="text-xs text-zinc-400">Use secret keys to authenticate server-to-server requests to the billing engine.</p>
				</div>
				<button
					class="px-4 py-2 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold flex items-center gap-2 shadow-lg shadow-indigo-600/30 transition-all"
					onclick={() => { showCreateKeyModal = true; createdSecret = null; }}
				>
					+ Create New Key
				</button>
			</div>

			<!-- Secret Reveal Callout -->
			{#if createdSecret}
				<div class="p-5 rounded-xl border border-amber-500/30 bg-amber-500/10 space-y-3">
					<div class="flex items-center justify-between">
						<span class="text-xs font-bold uppercase tracking-wider text-amber-400">⚠️ Save Your Key Secret</span>
						<button class="text-xs text-zinc-400 hover:text-white" onclick={() => createdSecret = null}>Close</button>
					</div>
					<div class="flex items-center gap-3">
						<input
							type="text"
							readonly
							value={createdSecret}
							class="w-full font-mono text-xs p-2.5 rounded-lg bg-black border border-amber-500/40 text-amber-200 select-all"
						/>
						<button
							class="px-4 py-2.5 rounded-lg bg-amber-600 hover:bg-amber-500 text-black font-semibold text-xs transition-all whitespace-nowrap"
							onclick={() => copyToClipboard(createdSecret!)}
						>
							Copy Secret
						</button>
					</div>
					<p class="text-xs text-amber-300/80">
						This secret will NEVER be displayed again. If lost, you must revoke it and generate a new key.
					</p>
				</div>
			{/if}

			<!-- API Keys Table -->
			<div class="rounded-xl border border-zinc-800 bg-zinc-950/60 overflow-hidden">
				<table class="w-full text-left font-mono text-xs">
					<thead class="border-b border-zinc-800 text-zinc-500 text-[11px] bg-zinc-900/50">
						<tr>
							<th class="p-3.5">NAME</th>
							<th class="p-3.5">PREFIX / TOKEN</th>
							<th class="p-3.5">SCOPES</th>
							<th class="p-3.5">CREATED</th>
							<th class="p-3.5 text-right">STATUS</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-zinc-900">
						{#if apiKeys.length === 0}
							<tr>
								<td colspan="5" class="p-8 text-center text-zinc-500 font-sans text-xs">
									No developer API keys created yet. Generate one to authenticate automated requests.
								</td>
							</tr>
						{:else}
							{#each apiKeys as key}
								<tr class="hover:bg-zinc-900/40 transition-colors">
									<td class="p-3.5 font-sans font-medium text-white">{key.name}</td>
									<td class="p-3.5 font-mono text-indigo-400">{key.key_prefix}_••••••••</td>
									<td class="p-3.5 font-sans">
										<div class="flex gap-1 flex-wrap">
											{#each key.scopes as scope}
												<span class="px-2 py-0.5 rounded text-[10px] bg-zinc-800 text-zinc-300 font-mono">
													{scope}
												</span>
											{/each}
										</div>
									</td>
									<td class="p-3.5 text-zinc-500">{new Date(key.created_at).toLocaleDateString()}</td>
									<td class="p-3.5 text-right">
										<span class="px-2 py-0.5 rounded text-[10px] font-bold {key.revoked ? 'bg-rose-500/10 text-rose-400' : 'bg-emerald-500/10 text-emerald-400'}">
											{key.revoked ? 'REVOKED' : 'ACTIVE'}
										</span>
									</td>
								</tr>
							{/each}
						{/if}
					</tbody>
				</table>
			</div>
		</div>
	{/if}

	<!-- TAB 2: CUSTOM FIELDS STUDIO -->
	{#if activeTab === 'custom_fields'}
		<div class="space-y-6">
			<div class="flex items-center justify-between flex-wrap gap-4">
				<div>
					<h2 class="text-lg font-semibold text-white">Custom Metadata Schema Engine</h2>
					<p class="text-xs text-zinc-400">Define tenant-specific fields with type constraints and regex validations.</p>
				</div>

				<div class="flex gap-1 p-1 bg-zinc-900 border border-zinc-800 rounded-lg text-xs">
					<button
						class="px-3 py-1.5 rounded font-medium {customFieldEntity === 'INVOICE' ? 'bg-zinc-800 text-white' : 'text-zinc-400 hover:text-white'}"
						onclick={() => handleSwitchFieldEntity('INVOICE')}
					>
						Invoices
					</button>
					<button
						class="px-3 py-1.5 rounded font-medium {customFieldEntity === 'CUSTOMER' ? 'bg-zinc-800 text-white' : 'text-zinc-400 hover:text-white'}"
						onclick={() => handleSwitchFieldEntity('CUSTOMER')}
					>
						Customers
					</button>
					<button
						class="px-3 py-1.5 rounded font-medium {customFieldEntity === 'PRODUCT' ? 'bg-zinc-800 text-white' : 'text-zinc-400 hover:text-white'}"
						onclick={() => handleSwitchFieldEntity('PRODUCT')}
					>
						Products
					</button>
				</div>
			</div>

			<!-- Add Custom Field Form -->
			<div class="p-5 rounded-xl border border-zinc-800 bg-zinc-900/40">
				<h3 class="text-sm font-semibold text-zinc-200 mb-3">Add Custom Field to {customFieldEntity}</h3>
				<form onsubmit={handleCreateCustomField} class="grid grid-cols-1 md:grid-cols-5 gap-3 items-end">
					<div>
						<label for="cf-label" class="block text-[11px] font-medium text-zinc-400 mb-1">Field Label</label>
						<input
							id="cf-label"
							type="text"
							placeholder="e.g. Purchase Order #"
							bind:value={newFieldLabel}
							class="w-full p-2 text-xs rounded-lg bg-zinc-950 border border-zinc-800 text-white focus:border-indigo-500"
							required
						/>
					</div>
					<div>
						<label for="cf-key" class="block text-[11px] font-medium text-zinc-400 mb-1">Field Key (JSON property)</label>
						<input
							id="cf-key"
							type="text"
							placeholder="e.g. po_number"
							bind:value={newFieldKey}
							class="w-full p-2 text-xs font-mono rounded-lg bg-zinc-950 border border-zinc-800 text-white focus:border-indigo-500"
							required
						/>
					</div>
					<div>
						<label for="cf-type" class="block text-[11px] font-medium text-zinc-400 mb-1">Data Type</label>
						<select
							id="cf-type"
							bind:value={newFieldType}
							class="w-full p-2 text-xs rounded-lg bg-zinc-950 border border-zinc-800 text-white focus:border-indigo-500"
						>
							<option value="STRING">STRING</option>
							<option value="NUMBER">NUMBER</option>
							<option value="BOOLEAN">BOOLEAN</option>
							<option value="DATE">DATE (YYYY-MM-DD)</option>
							<option value="ENUM">ENUM</option>
						</select>
					</div>
					<div>
						<label for="cf-regex" class="block text-[11px] font-medium text-zinc-400 mb-1">Validation Regex (Optional)</label>
						<input
							id="cf-regex"
							type="text"
							placeholder="^PO-[0-9]{5}$"
							bind:value={newFieldRegex}
							class="w-full p-2 text-xs font-mono rounded-lg bg-zinc-950 border border-zinc-800 text-white focus:border-indigo-500"
						/>
					</div>
					<div class="flex items-center gap-3">
						<label class="flex items-center gap-2 text-xs text-zinc-300 cursor-pointer">
							<input type="checkbox" bind:checked={newFieldRequired} class="rounded border-zinc-700 text-indigo-600" />
							Required
						</label>
						<button
							type="submit"
							disabled={isCreatingField}
							class="flex-1 p-2 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold transition-all disabled:opacity-50"
						>
							{isCreatingField ? 'Saving...' : '+ Add'}
						</button>
					</div>
				</form>
			</div>

			<!-- Fields Table -->
			<div class="rounded-xl border border-zinc-800 bg-zinc-950/60 overflow-hidden">
				<table class="w-full text-left text-xs">
					<thead class="border-b border-zinc-800 text-zinc-500 text-[11px] font-mono bg-zinc-900/50">
						<tr>
							<th class="p-3.5">LABEL</th>
							<th class="p-3.5">KEY</th>
							<th class="p-3.5">DATA TYPE</th>
							<th class="p-3.5">REGEX PATTERN</th>
							<th class="p-3.5 text-right">REQUIRED</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-zinc-900">
						{#if customFields.length === 0}
							<tr>
								<td colspan="5" class="p-8 text-center text-zinc-500 text-xs">
									No custom fields defined for {customFieldEntity}. Define one above to extend schemas.
								</td>
							</tr>
						{:else}
							{#each customFields as field}
								<tr class="hover:bg-zinc-900/40 transition-colors">
									<td class="p-3.5 font-medium text-white">{field.field_label}</td>
									<td class="p-3.5 font-mono text-indigo-400">{field.field_key}</td>
									<td class="p-3.5">
										<span class="px-2 py-0.5 rounded text-[10px] bg-zinc-800 text-zinc-300 font-mono">
											{field.data_type}
										</span>
									</td>
									<td class="p-3.5 font-mono text-zinc-400">{field.validation_regex || '—'}</td>
									<td class="p-3.5 text-right">
										<span class="px-2 py-0.5 rounded text-[10px] font-bold {field.is_required ? 'bg-amber-500/10 text-amber-400' : 'bg-zinc-800 text-zinc-400'}">
											{field.is_required ? 'YES' : 'NO'}
										</span>
									</td>
								</tr>
							{/each}
						{/if}
					</tbody>
				</table>
			</div>
		</div>
	{/if}

	<!-- TAB 3: USAGE METERING -->
	{#if activeTab === 'metering'}
		<div class="space-y-6">
			<div>
				<h2 class="text-lg font-semibold text-white">Usage-Based Metering Telemetry</h2>
				<p class="text-xs text-zinc-400">High-throughput event ingestion buffer with windowed rollups and live stream broadcasting.</p>
			</div>

			<!-- Simulator Form -->
			<div class="p-5 rounded-xl border border-zinc-800 bg-zinc-900/40">
				<h3 class="text-sm font-semibold text-zinc-200 mb-3">Simulate Metering Ingestion Event</h3>
				<div class="flex items-center gap-3 flex-wrap">
					<div class="flex-1 min-w-[200px]">
						<label for="meter-slug" class="block text-[11px] font-medium text-zinc-400 mb-1">Meter Slug</label>
						<select
							id="meter-slug"
							bind:value={meterSlug}
							class="w-full p-2 text-xs font-mono rounded-lg bg-zinc-950 border border-zinc-800 text-white"
						>
							<option value="api_requests">api_requests</option>
							<option value="ai_tokens">ai_tokens</option>
							<option value="storage_gb">storage_gb</option>
							<option value="active_seats">active_seats</option>
						</select>
					</div>
					<div class="w-32">
						<label for="meter-value" class="block text-[11px] font-medium text-zinc-400 mb-1">Units to Add</label>
						<input
							id="meter-value"
							type="number"
							bind:value={meterValue}
							class="w-full p-2 text-xs font-mono rounded-lg bg-zinc-950 border border-zinc-800 text-white"
						/>
					</div>
					<div class="pt-5">
						<button
							onclick={handleSimulateMetering}
							disabled={isSimulating}
							class="px-5 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-semibold text-xs transition-all shadow-lg shadow-emerald-600/20 disabled:opacity-50"
						>
							{isSimulating ? 'Queuing...' : '⚡ Ingest Event'}
						</button>
					</div>
				</div>
			</div>

			<!-- Embedded Live Financial & Metering Stream -->
			<LiveLedgerStream />
		</div>
	{/if}

	<!-- Create Key Modal -->
	{#if showCreateKeyModal}
		<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 backdrop-blur-sm p-4">
			<div class="w-full max-w-md rounded-xl border border-zinc-800 bg-zinc-950 p-6 shadow-2xl space-y-4">
				<div class="flex items-center justify-between border-b border-zinc-800 pb-3">
					<h3 class="text-sm font-semibold text-white">Generate Developer API Key</h3>
					<button class="text-zinc-500 hover:text-white" onclick={() => showCreateKeyModal = false}>✕</button>
				</div>

				<form onsubmit={handleCreateKey} class="space-y-3">
					<div>
						<label for="key-name" class="block text-xs font-medium text-zinc-300 mb-1">Key Name / Description</label>
						<input
							id="key-name"
							type="text"
							placeholder="e.g. Zapier Billing Integration"
							bind:value={newKeyName}
							class="w-full p-2.5 text-xs rounded-lg bg-zinc-900 border border-zinc-800 text-white focus:border-indigo-500"
							required
						/>
					</div>

					<div>
						<span class="block text-xs font-medium text-zinc-300 mb-1">Environment Mode</span>
						<div class="flex gap-4">
							<label class="flex items-center gap-2 text-xs text-zinc-300 cursor-pointer">
								<input type="radio" bind:group={newKeyLiveMode} value={true} />
								Live (sk_live_...)
							</label>
							<label class="flex items-center gap-2 text-xs text-zinc-300 cursor-pointer">
								<input type="radio" bind:group={newKeyLiveMode} value={false} />
								Test (sk_test_...)
							</label>
						</div>
					</div>

					<div class="flex justify-end gap-2 pt-3 border-t border-zinc-800">
						<button
							type="button"
							class="px-4 py-2 text-xs rounded-lg bg-zinc-800 text-zinc-300 hover:bg-zinc-700"
							onclick={() => showCreateKeyModal = false}
						>
							Cancel
						</button>
						<button
							type="submit"
							disabled={isCreatingKey}
							class="px-4 py-2 text-xs rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white font-semibold shadow-lg shadow-indigo-600/30 disabled:opacity-50"
						>
							{isCreatingKey ? 'Generating...' : 'Generate API Key'}
						</button>
					</div>
				</form>
			</div>
		</div>
	{/if}
</div>
