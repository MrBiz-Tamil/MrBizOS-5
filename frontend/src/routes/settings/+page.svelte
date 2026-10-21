<script lang="ts">
	import { onMount } from 'svelte';
	import { api, type Tenant, type User } from '$lib/api';

	let company = $state<Tenant | null>(null);
	let user = $state<User | null>(null);
	let loading = $state(true);
	let saving = $state(false);
	let message = $state('');
	let error = $state('');

	// Editable contact, branding, banking and template defaults
	let phone = $state('');
	let email = $state('');
	let website = $state('');
	let logoUrl = $state('');

	let bankName = $state('');
	let bankAccountNo = $state('');
	let bankIFSC = $state('');
	let bankBranch = $state('');
	let bankAccountHolder = $state('');
	let upiId = $state('');

	let businessSubtitle = $state('');
	let signatoryTitle = $state('Authorized Signatory');
	let signatureUrl = $state('');

	let defaultTerms = $state('Cash');
	let defaultJurisdiction = $state('Pune');
	let defaultInterestRate = $state('24%');
	let defaultReturnPolicy = $state('Goods once sold will not be taken back.');

	let drugLicenseNo = $state('');
	let fssaiNo = $state('');

	onMount(async () => {
		try {
			const [me, comp] = await Promise.all([api.getMe(), api.getCompany()]);
			user = me.user;
			company = comp;
			phone = comp.phone || '';
			email = comp.email || '';
			website = comp.website || '';
			logoUrl = comp.logo_url || '';

			bankName = comp.bank_name || '';
			bankAccountNo = comp.bank_account_no || '';
			bankIFSC = comp.bank_ifsc || '';
			bankBranch = comp.bank_branch || '';
			bankAccountHolder = comp.bank_account_holder || '';
			upiId = comp.upi_id || '';

			businessSubtitle = comp.business_subtitle || '';
			signatoryTitle = comp.signatory_title || 'Authorized Signatory';
			signatureUrl = comp.signature_url || '';

			defaultTerms = comp.default_terms || 'Cash';
			defaultJurisdiction = comp.default_jurisdiction || comp.city || 'Pune';
			defaultInterestRate = comp.default_interest_rate || '24%';
			defaultReturnPolicy = comp.default_return_policy || 'Goods once sold will not be taken back.';

			drugLicenseNo = comp.drug_license_no || '';
			fssaiNo = comp.fssai_no || '';
		} catch (err: any) {
			error = err.message || 'Failed to load company profile';
		} finally {
			loading = false;
		}
	});

	async function handleSave(e: Event) {
		e.preventDefault();
		message = '';
		error = '';
		saving = true;
		try {
			const updated = await api.updateCompany({
				phone,
				email,
				website,
				logo_url: logoUrl,
				bank_name: bankName,
				bank_account_no: bankAccountNo,
				bank_ifsc: bankIFSC,
				bank_branch: bankBranch,
				bank_account_holder: bankAccountHolder,
				upi_id: upiId,
				business_subtitle: businessSubtitle,
				signatory_title: signatoryTitle,
				signature_url: signatureUrl,
				default_terms: defaultTerms,
				default_jurisdiction: defaultJurisdiction,
				default_interest_rate: defaultInterestRate,
				default_return_policy: defaultReturnPolicy,
				drug_license_no: drugLicenseNo,
				fssai_no: fssaiNo
			});
			company = updated;
			if (logoUrl) {
				localStorage.setItem('tenant_brand_logo', logoUrl);
			} else {
				localStorage.removeItem('tenant_brand_logo');
			}
			message = 'Company profile, banking coordinates, and default terms updated successfully.';
		} catch (err: any) {
			error = err.message || 'Failed to save company profile';
		} finally {
			saving = false;
		}
	}
</script>

<svelte:head>
	<title>Company Profile & Legal Master - SaaS Billing</title>
</svelte:head>

<div style="display: flex; flex-direction: column; gap: 1.5rem; max-width: 900px;">
	<div style="display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 1rem;">
		<div>
			<h1 style="font-size: 1.75rem; margin-bottom: 0.25rem;">Company Profile & Tenant Master</h1>
			<p style="font-size: 0.875rem; color: var(--text-secondary);">
				Review legal registrations, configure banking settlement coordinates, visual branding, and invoice defaults.
			</p>
		</div>
		<a href="/settings/templates" class="btn btn-secondary" style="display: inline-flex; align-items: center; gap: 0.5rem;">
			<span>📑</span>
			<span>Specialized Templates ({company?.template_id ? `Template #${company.template_id}` : 'Manage'})</span>
		</a>
	</div>

	{#if message}
		<div class="card" style="background: rgba(16, 185, 129, 0.15); border-color: rgba(16, 185, 129, 0.3); color: #34d399; padding: 1rem;">
			✓ {message}
		</div>
	{/if}

	{#if error}
		<div class="card" style="background: rgba(244, 63, 94, 0.15); border-color: rgba(244, 63, 94, 0.3); color: #fb7185; padding: 1rem;">
			⚠️ {error}
		</div>
	{/if}

	{#if loading}
		<div style="padding: 3rem; text-align: center; color: var(--text-muted);">
			Loading company information...
		</div>
	{:else if company}
		<!-- Super Admin Protected Legal Master Box -->
		<div class="card" style="border: 1px solid rgba(245, 158, 11, 0.3); background: radial-gradient(circle at top left, rgba(245, 158, 11, 0.08), rgba(17, 24, 39, 0.9)); padding: 1.5rem;">
			<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1.25rem; flex-wrap: wrap; gap: 0.5rem;">
				<div style="display: flex; align-items: center; gap: 0.5rem;">
					<span style="font-size: 1.25rem;">🔒</span>
					<h3 style="font-size: 1.125rem;">Protected Legal Company Master Data</h3>
				</div>
				<span class="badge badge-protected">
					Restricted: Super Admin Only
				</span>
			</div>

			<p style="font-size: 0.8125rem; color: var(--text-secondary); margin-bottom: 1.25rem;">
				To prevent tax fraud and financial tampering, company legal name, GST identification number (GSTIN), and registered head office address are immutable by tenant users. Only a platform Super Administrator can modify these credentials.
			</p>

			<div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(240px, 1fr)); gap: 1.25rem;">
				<div class="form-group" style="margin-bottom: 0;">
					<label class="form-label" for="legal-name">Legal Registered Entity Name</label>
					<input id="legal-name" type="text" class="form-input" value={company.legal_name} disabled />
				</div>

				<div class="form-group" style="margin-bottom: 0;">
					<label class="form-label" for="gstin">GSTIN (Goods and Services Tax ID)</label>
					<input id="gstin" type="text" class="form-input font-mono" value={company.gstin} disabled />
				</div>

				<div class="form-group" style="margin-bottom: 0;">
					<label class="form-label" for="prefix">Invoice Sequence Prefix</label>
					<input id="prefix" type="text" class="form-input font-mono" value={company.invoice_prefix} disabled />
				</div>

				<div class="form-group" style="margin-bottom: 0;">
					<label class="form-label" for="fy-start">Financial Year Start</label>
					<input id="fy-start" type="text" class="form-input" value="April 1 (04-01)" disabled />
				</div>

				<div class="form-group" style="grid-column: 1 / -1; margin-bottom: 0;">
					<label class="form-label" for="reg-addr">Registered Head Office Address</label>
					<textarea id="reg-addr" class="form-textarea" rows="2" value={`${company.address}, ${company.city}, ${company.state} - ${company.pincode}`} disabled></textarea>
				</div>
			</div>
		</div>

		<!-- Editable Form for Contact, Banking, Branding & Invoice Defaults -->
		<form onsubmit={handleSave} class="card" style="padding: 1.5rem; display: flex; flex-direction: column; gap: 1.5rem;">
			<div>
				<h3 style="font-size: 1.125rem;">Operational & Contact Information</h3>
				<p style="font-size: 0.8125rem; color: var(--text-secondary); margin-top: 0.25rem;">
					Primary communications and corporate brand identity.
				</p>
			</div>

			<div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(240px, 1fr)); gap: 1.25rem;">
				<div class="form-group" style="margin-bottom: 0;">
					<label class="form-label" for="phone">Billing & Support Phone *</label>
					<input id="phone" type="tel" class="form-input" bind:value={phone} required />
				</div>

				<div class="form-group" style="margin-bottom: 0;">
					<label class="form-label" for="email">Invoicing Email *</label>
					<input id="email" type="email" class="form-input" bind:value={email} required />
				</div>

				<div class="form-group" style="margin-bottom: 0;">
					<label class="form-label" for="website">Company Website</label>
					<input id="website" type="url" class="form-input" bind:value={website} placeholder="https://..." />
				</div>

				<div class="form-group" style="margin-bottom: 0;">
					<label class="form-label" for="dl-no">Pharmacy Drug License No. (D.L. No)</label>
					<input id="dl-no" type="text" class="form-input font-mono" bind:value={drugLicenseNo} placeholder="e.g. 20B/21B-MH-TZ-12345" />
				</div>

				<div class="form-group" style="margin-bottom: 0;">
					<label class="form-label" for="fssai-no">FSSAI Food Safety License No.</label>
					<input id="fssai-no" type="text" class="form-input font-mono" bind:value={fssaiNo} placeholder="e.g. 11521012345678" />
				</div>

				<div class="form-group" style="grid-column: 1 / -1; margin-bottom: 0;">
					<label class="form-label" for="logo-url">Corporate Brand Logo URL (PNG / SVG / WebP)</label>
					<div style="display: flex; gap: 1rem; align-items: center;">
						<input id="logo-url" type="url" class="form-input" style="flex: 1;" bind:value={logoUrl} placeholder="https://example.com/logo.png" />
						{#if logoUrl}
							<div style="width: 44px; height: 44px; border-radius: 8px; background: rgba(255,255,255,0.05); border: 1px solid var(--border); display: flex; align-items: center; justify-content: center; padding: 3px; flex-shrink: 0;">
								<img src={logoUrl} alt="Logo preview" style="max-width: 100%; max-height: 100%; object-fit: contain;" />
							</div>
						{/if}
					</div>
				</div>
			</div>

			<hr style="border: 0; border-top: 1px solid var(--border-subtle); margin: 0;" />

			<!-- Banking & UPI Details -->
			<div>
				<h3 style="font-size: 1.125rem;">Direct Wire Settlement & UPI Remittance</h3>
				<p style="font-size: 0.8125rem; color: var(--text-secondary); margin-top: 0.25rem;">
					Rendered in the remittance coordinates box and UPI QR payment code on all customer invoices.
				</p>
			</div>

			<div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(240px, 1fr)); gap: 1.25rem;">
				<div class="form-group" style="margin-bottom: 0;">
					<label class="form-label" for="bank-name">Bank Name</label>
					<input id="bank-name" type="text" class="form-input" bind:value={bankName} placeholder="e.g. State Bank of India / HDFC Bank" />
				</div>

				<div class="form-group" style="margin-bottom: 0;">
					<label class="form-label" for="bank-acc">Account Number</label>
					<input id="bank-acc" type="text" class="form-input font-mono" bind:value={bankAccountNo} placeholder="e.g. 50200098765432" />
				</div>

				<div class="form-group" style="margin-bottom: 0;">
					<label class="form-label" for="bank-ifsc">IFSC Code</label>
					<input id="bank-ifsc" type="text" class="form-input font-mono" bind:value={bankIFSC} placeholder="e.g. HDFC0001234" />
				</div>

				<div class="form-group" style="margin-bottom: 0;">
					<label class="form-label" for="bank-branch">Branch Name</label>
					<input id="bank-branch" type="text" class="form-input" bind:value={bankBranch} placeholder="e.g. Nariman Point Branch" />
				</div>

				<div class="form-group" style="margin-bottom: 0;">
					<label class="form-label" for="bank-holder">Account Beneficiary Name</label>
					<input id="bank-holder" type="text" class="form-input" bind:value={bankAccountHolder} placeholder="Leave blank to use legal name" />
				</div>

				<div class="form-group" style="margin-bottom: 0;">
					<label class="form-label" for="upi-id">UPI ID / VPA (Instant QR)</label>
					<input id="upi-id" type="text" class="form-input font-mono" bind:value={upiId} placeholder="e.g. company@okhdfcbank" />
				</div>
			</div>

			<hr style="border: 0; border-top: 1px solid var(--border-subtle); margin: 0;" />

			<!-- Subtitle, Signatory & Default Policies -->
			<div>
				<h3 style="font-size: 1.125rem;">Signatory & Statutory Policies</h3>
				<p style="font-size: 0.8125rem; color: var(--text-secondary); margin-top: 0.25rem;">
					Designation headers, digital signatures, and invoice footer terms.
				</p>
			</div>

			<div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(240px, 1fr)); gap: 1.25rem;">
				<div class="form-group" style="margin-bottom: 0;">
					<label class="form-label" for="bus-sub">Business Subtitle / Tagline</label>
					<input id="bus-sub" type="text" class="form-input" bind:value={businessSubtitle} placeholder="e.g. Industrial Automation & Equipment" />
				</div>

				<div class="form-group" style="margin-bottom: 0;">
					<label class="form-label" for="sig-title">Signatory Title</label>
					<input id="sig-title" type="text" class="form-input" bind:value={signatoryTitle} placeholder="Authorized Signatory / Director" />
				</div>

				<div class="form-group" style="grid-column: 1 / -1; margin-bottom: 0;">
					<label class="form-label" for="sig-url">Digital Signature Image URL (Transparent PNG)</label>
					<div style="display: flex; gap: 1rem; align-items: center;">
						<input id="sig-url" type="url" class="form-input" style="flex: 1;" bind:value={signatureUrl} placeholder="https://example.com/signature.png" />
						{#if signatureUrl}
							<div style="max-height: 44px; padding: 4px; background: rgba(255,255,255,0.05); border: 1px solid var(--border); border-radius: 6px;">
								<img src={signatureUrl} alt="Signature Preview" style="max-height: 36px; object-fit: contain;" />
							</div>
						{/if}
					</div>
				</div>

				<div class="form-group" style="margin-bottom: 0;">
					<label class="form-label" for="def-terms">Default Payment Terms</label>
					<input id="def-terms" type="text" class="form-input" bind:value={defaultTerms} placeholder="e.g. Cash / Net 30 Days" />
				</div>

				<div class="form-group" style="margin-bottom: 0;">
					<label class="form-label" for="def-jur">Jurisdiction City</label>
					<input id="def-jur" type="text" class="form-input" bind:value={defaultJurisdiction} placeholder="e.g. Pune" />
				</div>

				<div class="form-group" style="margin-bottom: 0;">
					<label class="form-label" for="def-int">Overdue Interest Rate</label>
					<input id="def-int" type="text" class="form-input" bind:value={defaultInterestRate} placeholder="e.g. 24% p.a." />
				</div>

				<div class="form-group" style="margin-bottom: 0;">
					<label class="form-label" for="def-ret">Default Return Policy</label>
					<input id="def-ret" type="text" class="form-input" bind:value={defaultReturnPolicy} placeholder="e.g. Goods once sold..." />
				</div>
			</div>

			<div style="display: flex; justify-content: flex-end; margin-top: 0.5rem;">
				<button type="submit" class="btn btn-primary" disabled={saving}>
					{saving ? 'Saving Changes...' : 'Save Profile & Settings'}
				</button>
			</div>
		</form>
	{/if}
</div>
