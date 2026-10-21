<script lang="ts">
	import { onMount } from 'svelte';
	import { api, type Tenant } from '$lib/api';

	let tenants = $state<Tenant[]>([]);
	let loading = $state(true);
	let error = $state('');

	// Provisioning Modal State
	let showCreateModal = $state(false);
	let name = $state('');
	let legalName = $state('');
	let gstin = $state('');
	let address = $state('');
	let city = $state('');
	let stateName = $state('Maharashtra');
	let pincode = $state('');
	let email = $state('');
	let phone = $state('');
	let invoicePrefix = $state('INV');
	let subscriptionPlan = $state('STARTER');
	let templateId = $state('1');
	let modalError = $state('');
	let submitting = $state(false);
	let updatingTenantId = $state<string | null>(null);

	// Subscription & Quotas Management Modal State
	let showSubModal = $state(false);
	let selectedTenant = $state<Tenant | null>(null);
	let subPlanCode = $state('STARTER');
	let subStatus = $state('ACTIVE');
	let subBillingCycle = $state('MONTHLY');
	let subMaxUsers = $state<number | null>(null);
	let subMaxInvoices = $state<number | null>(null);
	let subSaving = $state(false);
	let subModalMsg = $state<{ type: 'success' | 'error'; text: string } | null>(null);

	const templateList = [
		{ id: '1', name: '1. Modern Corporate', desc: 'Standard B2B dual-tone enterprise layout' },
		{ id: '2', name: '2. Classic Traditional', desc: 'Indian wholesale boxed grid format' },
		{ id: '3', name: '3. Minimalist Executive', desc: 'Whitespace & clean typography for consulting' },
		{ id: '4', name: '4. Retail Counter POS', desc: 'Space-saving counter receipt format' },
		{ id: '5', name: '5. Thermal 80mm Slip', desc: 'Continuous 3-inch roll receipt' },
		{ id: '6', name: '6. Creative Studio', desc: 'Vibrant agency scope of work layout' },
		{ id: '7', name: '7. Industrial & Dispatch', desc: 'Manufacturing with PO/E-Way/Vehicle' },
		{ id: '8', name: '8. Bold Tech SaaS', desc: 'Subscription receipt with cloud hero banner' },
		{ id: '9', name: '9. Wholesale Multi-Tax', desc: 'HSN/SAC summary with split CGST/SGST/IGST' },
		{ id: '10', name: '10. Simple Bill of Supply', desc: 'Non-GST / composition cash memo' },
		{ id: '11', name: '11. Industrial BlueMetals & Transport', desc: 'Heavy boxed grid with PO, Vehicle, Delivery Place, & E.& O.E' },
		{ id: '12', name: '12. Hostel / Lodging Guest Folio', desc: 'Room/bed, check-in/out, nights, GST slab summary' },
		{ id: '13', name: '13. Restaurant / Cafe Bill', desc: 'Table, waiter, KOT no, FSSAI licence, CGST/SGST split' },
		{ id: '14', name: '14. Pharmacy / Medical Store', desc: 'Drug licence, patient & doctor, batch/expiry/MRP, Schedule H notice' },
		{ id: '15', name: '15. Bakery / Sweet Shop', desc: 'FSSAI licence, veg/non-veg marks, best-before, order pickup, advance & balance' },
		{ id: '16', name: '16. Grocery / Supermarket', desc: 'MRP vs price, savings, weighed qty, GST summary incl. exempt staples' },
		{ id: '17', name: '17. Hardware / Building Materials', desc: 'Trade discount, site delivery, transport, HSN summary, credit terms' }
	];

	onMount(async () => {
		await loadTenants();
	});

	async function loadTenants() {
		loading = true;
		try {
			tenants = await api.listTenants();
		} catch (err: any) {
			error = err.message || 'Failed to load tenant companies';
		} finally {
			loading = false;
		}
	}

	async function handleUpdateTemplate(tId: string, newTpl: string) {
		updatingTenantId = tId;
		try {
			await api.updateTenant(tId, { template_id: newTpl });
			const found = tenants.find((t) => t.id === tId);
			if (found) found.template_id = newTpl;
		} catch (err: any) {
			alert('Failed to update tenant template: ' + (err.message || err));
		} finally {
			updatingTenantId = null;
		}
	}

	function openSubscriptionModal(t: Tenant) {
		selectedTenant = t;
		subPlanCode = (t.subscription_plan || 'STARTER').toUpperCase();
		subStatus = (t.subscription_status || 'ACTIVE').toUpperCase();
		subBillingCycle = (t.billing_cycle || 'MONTHLY').toUpperCase();
		subMaxUsers = t.max_users_override ?? null;
		subMaxInvoices = t.max_invoices_override ?? null;
		subModalMsg = null;
		showSubModal = true;
	}

	async function handleSaveSubscription(e: Event) {
		e.preventDefault();
		if (!selectedTenant) return;
		subSaving = true;
		subModalMsg = null;

		try {
			await api.updateTenantSubscription(selectedTenant.id, {
				plan_code: subPlanCode,
				status: subStatus,
				billing_cycle: subBillingCycle,
				max_users_override: subMaxUsers ? Number(subMaxUsers) : null,
				max_invoices_override: subMaxInvoices ? Number(subMaxInvoices) : null
			});

			subModalMsg = { type: 'success', text: `Subscription for ${selectedTenant.name} updated successfully!` };
			await loadTenants();
			setTimeout(() => {
				if (subModalMsg?.type === 'success') {
					showSubModal = false;
				}
			}, 1200);
		} catch (err: any) {
			subModalMsg = { type: 'error', text: err.message || 'Failed to update subscription' };
		} finally {
			subSaving = false;
		}
	}

	async function handleCreateTenant(e: Event) {
		e.preventDefault();
		modalError = '';
		const trimmedName = name.trim();
		const trimmedLegal = legalName.trim();
		const trimmedGSTIN = gstin.trim().toUpperCase();
		const trimmedAddress = address.trim();
		const trimmedCity = city.trim();
		const trimmedState = stateName.trim();
		const trimmedPincode = pincode.trim();
		const trimmedEmail = email.trim();
		const trimmedPhone = phone.trim();
		const trimmedPrefix = invoicePrefix.trim().toUpperCase() || 'INV';

		if (!trimmedName || !trimmedLegal || !trimmedGSTIN || !trimmedAddress || !trimmedCity || !trimmedState || !trimmedPincode || !trimmedEmail || !trimmedPhone) {
			modalError = 'Please fill out all required company master data fields';
			return;
		}
		if (trimmedGSTIN.length !== 15) {
			modalError = 'Indian GSTIN must be exactly 15 characters';
			return;
		}

		submitting = true;
		try {
			await api.createTenant({
				name: trimmedName,
				legal_name: trimmedLegal,
				gstin: trimmedGSTIN,
				address: trimmedAddress,
				city: trimmedCity,
				state: trimmedState,
				country: 'India',
				pincode: trimmedPincode,
				email: trimmedEmail,
				phone: trimmedPhone,
				invoice_prefix: trimmedPrefix,
				subscription_plan: subscriptionPlan,
				subscription_status: 'ACTIVE',
				template_id: templateId,
				is_active: true
			});
			showCreateModal = false;
			await loadTenants();
		} catch (err: any) {
			modalError = err.message || 'Failed to create tenant company';
		} finally {
			submitting = false;
		}
	}

	function getPlanBadgeColor(plan: string) {
		switch ((plan || '').toUpperCase()) {
			case 'FREE':
			case 'BASIC':
				return { bg: 'rgba(52, 211, 153, 0.15)', text: '#34d399', border: 'rgba(52, 211, 153, 0.3)' };
			case 'STARTER':
				return { bg: 'rgba(56, 189, 248, 0.15)', text: '#38bdf8', border: 'rgba(56, 189, 248, 0.3)' };
			case 'BUSINESS':
			case 'PRO':
				return { bg: 'rgba(129, 140, 248, 0.15)', text: '#818cf8', border: 'rgba(129, 140, 248, 0.3)' };
			case 'ENTERPRISE':
				return { bg: 'rgba(251, 191, 36, 0.15)', text: '#fbbf24', border: 'rgba(251, 191, 36, 0.3)' };
			default:
				return { bg: 'rgba(148, 163, 184, 0.15)', text: '#cbd5e1', border: 'rgba(148, 163, 184, 0.2)' };
		}
	}
</script>

<svelte:head>
	<title>Company Master Management - Super Admin</title>
</svelte:head>

<div style="display: flex; flex-direction: column; gap: 1.5rem;">
	<!-- Header Bar -->
	<div style="display: flex; justify-content: space-between; align-items: flex-start; flex-wrap: wrap; gap: 1rem;">
		<div>
			<div style="display: flex; gap: 1rem; align-items: center; margin-bottom: 0.25rem;">
				<a href="/admin" style="font-size: 0.8125rem; color: #818cf8; text-decoration: none;">
					← Back to Admin Console
				</a>
				<span style="color: var(--border);">|</span>
				<a href="/admin/subscriptions" style="font-size: 0.8125rem; color: #38bdf8; text-decoration: none; font-weight: 500;">
					💎 View Subscription Catalog & Quotas →
				</a>
			</div>
			<h1 style="font-size: 1.75rem;">Company Tenants Master</h1>
			<p style="font-size: 0.875rem; color: var(--text-secondary);">
				Provision, inspect, and configure multi-tenant company accounts, template locks, and subscription quotas.
			</p>
		</div>
		<div style="display: flex; gap: 0.75rem;">
			<a href="/admin/subscriptions" class="btn btn-secondary" style="display: flex; align-items: center; gap: 0.4rem;">
				💎 Subscription Plans
			</a>
			<button onclick={() => (showCreateModal = true)} class="btn btn-primary">
				+ Provision New Company
			</button>
		</div>
	</div>

	<!-- Tenants Table -->
	<div class="table-container card" style="padding: 0; overflow-x: auto;">
		<table class="data-table">
			<thead>
				<tr>
					<th>Company Name</th>
					<th>Legal Registered Name</th>
					<th>GSTIN</th>
					<th>Location</th>
					<th>Prefix</th>
					<th>Subscription Tier</th>
					<th>Fixed Template</th>
					<th>Status</th>
					<th>Actions</th>
				</tr>
			</thead>
			<tbody>
				{#if loading}
					<tr>
						<td colspan="9" class="text-center" style="padding: 3rem; color: var(--text-muted);">
							Loading tenant directories...
						</td>
					</tr>
				{:else if error}
					<tr>
						<td colspan="9" class="text-center" style="padding: 2rem; color: var(--rose);">
							⚠️ {error}
						</td>
					</tr>
				{:else if tenants.length === 0}
					<tr>
						<td colspan="9" class="text-center" style="padding: 3rem; color: var(--text-muted);">
							No tenant companies provisioned yet.
						</td>
					</tr>
				{:else}
					{#each tenants as t}
						{@const pColor = getPlanBadgeColor(t.subscription_plan)}
						<tr>
							<td style="font-weight: 600; min-width: 150px;">
								<div>{t.name}</div>
								{#if t.billing_cycle}
									<div style="font-size: 0.7rem; color: var(--text-muted); font-weight: normal;">Cycle: {t.billing_cycle}</div>
								{/if}
							</td>
							<td style="color: var(--text-secondary);">{t.legal_name}</td>
							<td style="font-family: var(--font-mono); color: #818cf8; font-size: 0.8125rem;">{t.gstin}</td>
							<td style="font-size: 0.8125rem;">{t.city}, {t.state}</td>
							<td style="font-family: var(--font-mono); font-weight: 600; color: #fbbf24;">{t.invoice_prefix}</td>
							<td>
								<span style="display: inline-flex; align-items: center; gap: 0.35rem; font-size: 0.75rem; font-weight: 600; background: {pColor.bg}; color: {pColor.text}; border: 1px solid {pColor.border}; padding: 0.2rem 0.6rem; border-radius: 9999px;">
									💎 {t.subscription_plan || 'FREE'}
								</span>
							</td>
							<td>
								<select
									class="form-select"
									style="width: auto; min-width: 170px; padding: 0.25rem 0.5rem; font-size: 0.75rem; background: #1e293b; color: #38bdf8; border-color: rgba(56, 189, 248, 0.4); border-radius: 4px;"
									value={t.template_id || '1'}
									disabled={updatingTenantId === t.id}
									onchange={(e) => handleUpdateTemplate(t.id, (e.target as HTMLSelectElement).value)}
									title="Super Admin: Lock this company's invoice template"
								>
									{#each templateList as tpl}
										<option value={tpl.id}>{tpl.name}</option>
									{/each}
								</select>
							</td>
							<td>
								{#if t.is_active && t.subscription_status === 'ACTIVE'}
									<span class="badge" style="background: rgba(16, 185, 129, 0.15); color: #34d399;">
										Active
									</span>
								{:else if t.subscription_status === 'SUSPENDED' || !t.is_active}
									<span class="badge" style="background: rgba(244, 63, 94, 0.15); color: #fb7185;">
										Suspended
									</span>
								{:else}
									<span class="badge" style="background: rgba(245, 158, 11, 0.15); color: #fbbf24;">
										{t.subscription_status || 'Trial'}
									</span>
								{/if}
							</td>
							<td>
								<button
									onclick={() => openSubscriptionModal(t)}
									class="btn btn-secondary"
									style="padding: 0.25rem 0.65rem; font-size: 0.75rem; display: inline-flex; align-items: center; gap: 0.3rem;"
									title="Manage plan tier, quotas and account status"
								>
									⚙️ Subscription
								</button>
							</td>
						</tr>
					{/each}
				{/if}
			</tbody>
		</table>
	</div>
</div>

<!-- Super Admin: Subscription & Quota Management Modal -->
{#if showSubModal && selectedTenant}
	<div class="modal-backdrop">
		<div class="modal-card animate-fade-in" style="max-width: 580px;">
			<div class="modal-header">
				<div>
					<h3 style="font-size: 1.125rem; margin: 0;">Manage Subscription & Quotas</h3>
					<div style="font-size: 0.8125rem; color: var(--text-secondary); margin-top: 0.2rem;">
						Tenant: <strong style="color: #fff;">{selectedTenant.name}</strong> ({selectedTenant.legal_name})
					</div>
				</div>
				<button onclick={() => (showSubModal = false)} style="background: none; border: none; color: var(--text-muted); cursor: pointer; font-size: 1.25rem;">✕</button>
			</div>

			<form onsubmit={handleSaveSubscription}>
				<div class="modal-body" style="display: flex; flex-direction: column; gap: 1.25rem;">
					{#if subModalMsg}
						<div
							style="border-radius: var(--radius-md); padding: 0.75rem; font-size: 0.8125rem; {subModalMsg.type === 'success'
								? 'background: rgba(16, 185, 129, 0.15); border: 1px solid rgba(16, 185, 129, 0.3); color: #34d399;'
								: 'background: rgba(244, 63, 94, 0.15); border: 1px solid rgba(244, 63, 94, 0.3); color: #fb7185;'}"
						>
							{subModalMsg.type === 'success' ? '✓' : '⚠️'} {subModalMsg.text}
						</div>
					{/if}

					<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="sub-plan">Subscription Plan Tier *</label>
							<select id="sub-plan" class="form-select" bind:value={subPlanCode} required>
								<option value="BASIC">🌱 Basic Plan (Billing, Customers & Credit/Debit Ledger - ₹499/mo)</option>
								<option value="STARTER">🌱 Starter Tier (Basic Features: Invoicing & Customer Balances)</option>
								<option value="BUSINESS">💎 Next Plan: Business (Full Accounting & Expense Management)</option>
								<option value="PRO">💎 Next Plan: Pro Tier (Accounting, Expenses & Audit)</option>
								<option value="ENTERPRISE">👑 Enterprise (Full Platform Unlimited)</option>
							</select>
						</div>

						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="sub-status">Account Access Status *</label>
							<select id="sub-status" class="form-select" bind:value={subStatus} required>
								<option value="ACTIVE">ACTIVE (Full Platform Access)</option>
								<option value="SUSPENDED">SUSPENDED (Lock Invoicing & Operations)</option>
							</select>
						</div>
					</div>

					<div class="form-group" style="margin-bottom: 0;">
						<label class="form-label" for="sub-cycle">Billing Frequency</label>
						<select id="sub-cycle" class="form-select" bind:value={subBillingCycle}>
							<option value="MONTHLY">Monthly Billing Cycle</option>
							<option value="QUARTERLY">Quarterly (3 Months)</option>
							<option value="ANNUAL">Annual Billing (12 Months)</option>
						</select>
					</div>

					<div style="background: rgba(15, 23, 42, 0.5); border: 1px solid var(--border); border-radius: var(--radius-md); padding: 1rem; display: flex; flex-direction: column; gap: 0.75rem;">
						<div style="font-size: 0.8125rem; font-weight: 600; color: #38bdf8;">
							🔧 Custom Quota Overrides (Optional)
						</div>
						<div style="font-size: 0.75rem; color: var(--text-muted);">
							Leave blank to use default plan limits. Set custom numbers to grant special allowances.
						</div>

						<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">
							<div class="form-group" style="margin-bottom: 0;">
								<label class="form-label" for="sub-users" style="font-size: 0.75rem;">Max Users Override</label>
								<input
									id="sub-users"
									type="number"
									min="1"
									class="form-input"
									bind:value={subMaxUsers}
									placeholder="e.g. 10 (Leave empty for default)"
								/>
							</div>

							<div class="form-group" style="margin-bottom: 0;">
								<label class="form-label" for="sub-inv" style="font-size: 0.75rem;">Max Invoices/Mo Override</label>
								<input
									id="sub-inv"
									type="number"
									min="1"
									class="form-input"
									bind:value={subMaxInvoices}
									placeholder="e.g. 1500 (Leave empty for default)"
								/>
							</div>
						</div>
					</div>
				</div>

				<div class="modal-footer" style="display: flex; justify-content: flex-end; gap: 0.75rem;">
					<button type="button" onclick={() => (showSubModal = false)} class="btn btn-secondary">
						Close
					</button>
					<button type="submit" class="btn btn-primary" disabled={subSaving}>
						{subSaving ? 'Saving Changes...' : 'Save Subscription Changes'}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}

<!-- Provision New Company Modal -->
{#if showCreateModal}
	<div class="modal-backdrop">
		<div class="modal-card animate-fade-in" style="max-width: 680px;">
			<div class="modal-header">
				<h3 style="font-size: 1.125rem;">Provision New Tenant Company</h3>
				<button onclick={() => (showCreateModal = false)} style="background: none; border: none; color: var(--text-muted); cursor: pointer; font-size: 1.25rem;">✕</button>
			</div>
			<form onsubmit={handleCreateTenant}>
				<div class="modal-body" style="display: flex; flex-direction: column; gap: 1rem;">
					{#if modalError}
						<div style="background: rgba(244, 63, 94, 0.15); border: 1px solid rgba(244, 63, 94, 0.3); border-radius: var(--radius-md); padding: 0.75rem; color: #fb7185; font-size: 0.8125rem;">
							⚠️ {modalError}
						</div>
					{/if}

					<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="t-name">Brand / Display Name *</label>
							<input id="t-name" type="text" class="form-input" bind:value={name} placeholder="e.g. Acme Tech" required />
						</div>

						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="t-legal">Legal Registered Entity *</label>
							<input id="t-legal" type="text" class="form-input" bind:value={legalName} placeholder="e.g. Acme Technologies Pvt Ltd" required />
						</div>
					</div>

					<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="t-gstin">GSTIN (15 characters) *</label>
							<input id="t-gstin" type="text" maxlength="15" class="form-input font-mono" bind:value={gstin} placeholder="e.g. 27ABCDE1234F1Z5" required />
						</div>

						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="t-prefix">Invoice Number Prefix *</label>
							<input id="t-prefix" type="text" maxlength="8" class="form-input font-mono" bind:value={invoicePrefix} placeholder="e.g. ACM" required />
						</div>
					</div>

					<div class="form-group" style="margin-bottom: 0;">
						<label class="form-label" for="t-addr">Registered Office Address *</label>
						<input id="t-addr" type="text" class="form-input" bind:value={address} placeholder="Plot 42, Tech Corridor" required />
					</div>

					<div style="display: grid; grid-template-columns: 1fr 1fr 1fr; gap: 1rem;">
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="t-city">City *</label>
							<input id="t-city" type="text" class="form-input" bind:value={city} placeholder="Mumbai" required />
						</div>

						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="t-state">State *</label>
							<input id="t-state" type="text" class="form-input" bind:value={stateName} placeholder="Maharashtra" required />
						</div>

						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="t-pin">Pincode *</label>
							<input id="t-pin" type="text" maxlength="6" class="form-input font-mono" bind:value={pincode} placeholder="400001" required />
						</div>
					</div>

					<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="t-email">Billing Email *</label>
							<input id="t-email" type="email" class="form-input" bind:value={email} placeholder="billing@acme.com" required />
						</div>

						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="t-phone">Phone Number *</label>
							<input id="t-phone" type="tel" class="form-input" bind:value={phone} placeholder="9876543210" required />
						</div>
					</div>

					<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="t-plan">Subscription Tier</label>
							<select id="t-plan" class="form-select" bind:value={subscriptionPlan}>
								<option value="BASIC">🌱 Basic Plan (Billing, Customers & Credit/Debit)</option>
								<option value="STARTER">🌱 Starter Tier (Basic Plan Features)</option>
								<option value="BUSINESS">💎 Next Plan: Business (Accounting + Expenses)</option>
								<option value="PRO">💎 Next Plan: Pro (Accounting + Expenses + Audit)</option>
								<option value="ENTERPRISE">👑 Enterprise Tier (Unlimited Quotas)</option>
							</select>
						</div>

						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="t-template">Assigned Invoice Template (Super Admin Fixed)</label>
							<select id="t-template" class="form-select" bind:value={templateId}>
								{#each templateList as tpl}
									<option value={tpl.id}>{tpl.name}</option>
								{/each}
							</select>
						</div>
					</div>
				</div>
				<div class="modal-footer">
					<button type="button" onclick={() => (showCreateModal = false)} class="btn btn-secondary">Cancel</button>
					<button type="submit" class="btn btn-primary" disabled={submitting}>
						{submitting ? 'Provisioning Tenant...' : 'Provision Tenant Company'}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}
