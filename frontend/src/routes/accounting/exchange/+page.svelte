<script lang="ts">
	import { api } from '$lib/api';

	const today = new Date();
	let month = $state(today.toISOString().slice(0, 7));
	let gst = $state<any>(null);
	let gstError = $state('');
	let gstBusy = $state(false);

	let tallyFrom = $state(new Date(today.getFullYear(), today.getMonth(), 1).toISOString().slice(0, 10));
	let tallyTo = $state(today.toISOString().slice(0, 10));
	let tallyCompany = $state('');

	let eiFrom = $state(new Date(today.getFullYear(), today.getMonth(), 1).toISOString().slice(0, 10));
	let eiTo = $state(today.toISOString().slice(0, 10));
	let ei = $state<any>(null);
	let eiError = $state('');
	async function previewEInvoice() {
		eiError = '';
		try {
			ei = await api.getEInvoiceJson(eiFrom, eiTo);
		} catch (e: any) {
			eiError = e.message;
		}
	}

	let importKind = $state<'customers' | 'products'>('customers');
	let csvText = $state('');
	let importResult = $state<any>(null);
	let importError = $state('');
	let importBusy = $state(false);

	const templates = {
		customers: 'name,company_name,gstin,email,phone,address,city,state,pincode\n',
		products: 'name,sku,hsn_sac,unit,tax_rate,selling_price,purchase_price,opening_stock,category\n'
	};

	async function previewGST() {
		gstError = '';
		gstBusy = true;
		try {
			gst = await api.getGSTR1Json(month);
		} catch (e: any) {
			gstError = e.message;
		} finally {
			gstBusy = false;
		}
	}

	async function onFile(e: Event) {
		const f = (e.target as HTMLInputElement).files?.[0];
		if (f) csvText = await f.text();
		importResult = null;
	}

	async function runImport(dry: boolean) {
		importError = '';
		importBusy = true;
		try {
			importResult = await api.importCsv(importKind, csvText, dry);
		} catch (e: any) {
			importError = e.message;
		} finally {
			importBusy = false;
		}
	}
</script>

<svelte:head><title>Returns & Data Exchange</title></svelte:head>

<div style="max-width: 900px;">
	<h1 style="margin-bottom: 0.25rem;">Returns & data exchange</h1>
	<p style="color: var(--text-secondary); margin-bottom: 1.25rem;">Export for GST filing and Tally, and import customers and products from a spreadsheet.</p>

	<div class="card" style="padding: 1.25rem; margin-bottom: 1.25rem;">
		<h3 style="margin-bottom: 0.5rem;">GSTR-1 (JSON for the GST portal)</h3>
		<p style="color: var(--text-secondary); font-size: 0.8125rem; margin-bottom: 0.75rem;">
			Prepares B2B, B2C large, B2C small, credit notes, HSN summary and document counts for one month. Always load the file into the official GST Offline Tool or portal and review its validation report before filing.
		</p>
		<div style="display: flex; gap: 0.75rem; align-items: end; flex-wrap: wrap;">
			<div class="form-group" style="margin: 0;"><label class="form-label" for="m">Month</label><input id="m" type="month" class="form-input" bind:value={month} /></div>
			<button class="btn btn-secondary" onclick={previewGST} disabled={gstBusy}>Preview</button>
			<a class="btn btn-primary" href="/api/reports/gst/gstr-1/json?month={month}&download=1" download>Download JSON</a>
		</div>
		{#if gstError}<p style="color: #fb7185;">{gstError}</p>{/if}
		{#if gst}
			<p style="margin-top: 0.75rem; font-size: 0.875rem;">
				{gst.counts.Invoices} invoices · {gst.counts.B2B} B2B · {gst.counts.B2CL} B2C large · {gst.counts.B2CSRows} B2C small rows · {gst.counts.CreditNotes} credit notes · {gst.counts.HSNRows} HSN rows
			</p>
			{#if gst.warnings.length}
				<div style="margin-top: 0.5rem; color: #fbbf24; font-size: 0.8125rem;">
					<strong>Check before filing:</strong>
					<ul style="margin: 0.25rem 0 0 1.25rem;">{#each gst.warnings as w}<li>{w}</li>{/each}</ul>
				</div>
			{:else}
				<p style="color: #34d399; font-size: 0.8125rem;">No warnings.</p>
			{/if}
		{/if}
	</div>

	<div class="card" style="padding: 1.25rem; margin-bottom: 1.25rem;">
		<h3 style="margin-bottom: 0.5rem;">E-invoice file (for the IRP bulk upload)</h3>
		<p style="color: var(--text-secondary); font-size: 0.8125rem; margin-bottom: 0.75rem;">
			Builds the e-invoice JSON for your B2B invoices (customers with a GSTIN). Upload it on the government e-invoice portal to get the IRN and QR code; this software does not contact the portal itself, so the IRN is not stored here.
		</p>
		<div style="display: flex; gap: 0.75rem; align-items: end; flex-wrap: wrap;">
			<div class="form-group" style="margin: 0;"><label class="form-label" for="ef">From</label><input id="ef" type="date" class="form-input" bind:value={eiFrom} /></div>
			<div class="form-group" style="margin: 0;"><label class="form-label" for="et">To</label><input id="et" type="date" class="form-input" bind:value={eiTo} /></div>
			<button class="btn btn-secondary" onclick={previewEInvoice}>Check</button>
			<a class="btn btn-primary" href="/api/reports/einvoice/json?from={eiFrom}&to={eiTo}&download=1" download>Download JSON</a>
		</div>
		{#if eiError}<p style="color: #fb7185;">{eiError}</p>{/if}
		{#if ei}
			<p style="margin-top: 0.75rem; font-size: 0.875rem;">{ei.documents.length} invoice(s) ready, {ei.skipped_not_b2b} skipped (no customer GSTIN).</p>
			{#if ei.warnings.length}
				<ul style="color: #fbbf24; font-size: 0.8125rem; margin: 0.25rem 0 0 1.25rem; max-height: 180px; overflow: auto;">{#each ei.warnings.slice(0, 40) as w}<li>{w}</li>{/each}</ul>
			{/if}
		{/if}
	</div>

	<div class="card" style="padding: 1.25rem; margin-bottom: 1.25rem;">
		<h3 style="margin-bottom: 0.5rem;">Tally export</h3>
		<p style="color: var(--text-secondary); font-size: 0.8125rem; margin-bottom: 0.75rem;">
			Sales invoices, credit notes and customer receipts as a Tally "Import Data" XML file. Customer and tax ledgers are created if missing. Check Tally's import log afterwards.
		</p>
		<div style="display: flex; gap: 0.75rem; align-items: end; flex-wrap: wrap;">
			<div class="form-group" style="margin: 0;"><label class="form-label" for="tf">From</label><input id="tf" type="date" class="form-input" bind:value={tallyFrom} /></div>
			<div class="form-group" style="margin: 0;"><label class="form-label" for="tt">To</label><input id="tt" type="date" class="form-input" bind:value={tallyTo} /></div>
			<div class="form-group" style="margin: 0;"><label class="form-label" for="tc">Tally company name (optional)</label><input id="tc" class="form-input" bind:value={tallyCompany} /></div>
			<a class="btn btn-primary" href="/api/reports/tally/vouchers?from={tallyFrom}&to={tallyTo}&company={encodeURIComponent(tallyCompany)}" download>Download XML</a>
		</div>
	</div>

	<div class="card" style="padding: 1.25rem;">
		<h3 style="margin-bottom: 0.5rem;">Import from CSV</h3>
		<p style="color: var(--text-secondary); font-size: 0.8125rem; margin-bottom: 0.75rem;">
			Save your spreadsheet as CSV with this header row, choose the file, run a check first, then import. Existing records are skipped, never overwritten.
		</p>
		<div style="display: flex; gap: 0.75rem; align-items: end; flex-wrap: wrap; margin-bottom: 0.75rem;">
			<div class="form-group" style="margin: 0;"><label class="form-label" for="ik">Import</label>
				<select id="ik" class="form-input" bind:value={importKind} onchange={() => (importResult = null)}><option value="customers">Customers</option><option value="products">Products</option></select></div>
			<div class="form-group" style="margin: 0;"><label class="form-label" for="if">CSV file</label><input id="if" type="file" accept=".csv,text/csv" onchange={onFile} /></div>
		</div>
		<pre style="background: rgba(0,0,0,0.3); padding: 0.5rem; border-radius: 6px; font-size: 0.75rem; overflow-x: auto;">{templates[importKind]}</pre>
		<div style="display: flex; gap: 0.5rem; margin-top: 0.5rem;">
			<button class="btn btn-secondary" disabled={!csvText || importBusy} onclick={() => runImport(true)}>Check only</button>
			<button class="btn btn-primary" disabled={!csvText || importBusy} onclick={() => runImport(false)}>Import</button>
		</div>
		{#if importError}<p style="color: #fb7185;">{importError}</p>{/if}
		{#if importResult}
			<p style="margin-top: 0.75rem; font-size: 0.875rem;">
				{importResult.dry_run ? 'Check result' : 'Imported'}: <strong>{importResult.created}</strong> {importResult.dry_run ? 'would be created' : 'created'},
				{importResult.skipped_existing} already exist, {importResult.errors.length} row(s) with problems (of {importResult.rows}).
			</p>
			{#if importResult.errors.length}
				<ul style="color: #fb7185; font-size: 0.8125rem; margin-left: 1.25rem;">
					{#each importResult.errors.slice(0, 50) as er}<li>Row {er.row}: {er.error}</li>{/each}
				</ul>
			{/if}
		{/if}
	</div>
</div>
