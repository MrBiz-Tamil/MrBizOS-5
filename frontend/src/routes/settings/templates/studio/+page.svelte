<script lang="ts">
	import { onMount } from 'svelte';
	import { api, type Tenant, type TemplateStudioConfig } from '$lib/api';

	let company = $state<Tenant | null>(null);
	let loading = $state(true);
	let saving = $state(false);
	let saveSuccess = $state(false);
	let errorMessage = $state('');

	// Active Studio Configuration
	let config = $state<TemplateStudioConfig>({
		paper_size: 'thermal_80mm',
		theme: 'supermarket',
		font_family: 'monospace',
		font_size: 11,
		divider_style: 'dashed',
		show_logo: true,
		logo_height: 48,
		show_store_name: true,
		store_name_size: 18,
		show_address: true,
		show_dl_no: true,
		show_fssai_no: true,
		show_gstin: true,
		show_phone: true,
		show_cashier: true,
		show_patient_info: true,
		show_savings_banner: true,
		show_upi_qr: true,
		upi_qr_size: 120,
		show_barcode: true,
		custom_footer_message: 'THANK YOU! VISIT AGAIN • GOODS ONCE SOLD WILL NOT BE RETURNED',
		visible_columns: {
			sno: false,
			description: true,
			generic_name: true,
			batch_no: true,
			expiry: true,
			rack: false,
			mrp: true,
			rate: true,
			qty: true,
			discount: true,
			tax: false,
			total: true
		}
	});

	// Mock Demo Data for Live Preview
	const sampleBill = {
		invoice_number: 'POS/2026/00912',
		date: '09/09/2026 08:45 PM',
		cashier: 'Priya Sharma (Counter 02)',
		customer: 'Rahul Verma (+91 98765 43210)',
		patient: 'Aarav Verma (Age: 8)',
		doctor: 'Dr. S. Kulkarni, MBBS (PMC-4821)',
		items: [
			{
				sno: 1,
				name: 'Amoxicillin 500mg (Strip)',
				generic: 'Amoxicillin Trihydrate IP',
				batch: 'AMX-948',
				exp: '11/27',
				rack: 'Rack B-03',
				qty: 1,
				mrp: 112.50,
				rate: 98.00,
				disc: '12%',
				tax: '12%',
				total: 98.00
			},
			{
				sno: 2,
				name: 'Dolo 650mg Paracetamol',
				generic: 'Paracetamol 650mg',
				batch: 'DL-8821',
				exp: '03/28',
				rack: 'Rack A-01',
				qty: 2,
				mrp: 34.00,
				rate: 30.50,
				disc: '10%',
				tax: '12%',
				total: 61.00
			},
			{
				sno: 3,
				name: 'Organic Whole Milk 1L',
				generic: '',
				batch: 'MK-09',
				exp: '12/09/26',
				rack: 'Chiller-01',
				qty: 2,
				mrp: 68.00,
				rate: 68.00,
				disc: '0%',
				tax: '0%',
				total: 136.00
			},
			{
				sno: 4,
				name: 'Eco Shopping Bag (Large)',
				generic: '',
				batch: '',
				exp: '',
				rack: 'Counter',
				qty: 1,
				mrp: 15.00,
				rate: 10.00,
				disc: '33%',
				tax: '18%',
				total: 10.00
			}
		],
		subtotal_mrp: 341.50,
		net_total: 305.00,
		savings: 36.50
	};

	onMount(async () => {
		try {
			const data = await api.getCompany();
			company = data;
			if (data.template_studio_config && Object.keys(data.template_studio_config).length > 0) {
				config = {
					...config,
					...data.template_studio_config,
					visible_columns: {
						...config.visible_columns,
						...(data.template_studio_config.visible_columns || {})
					}
				};
			}
		} catch (e) {
			console.error('Failed loading template studio settings', e);
		} finally {
			loading = false;
		}
	});

	function applyPreset(preset: 'supermarket' | 'pharmacy' | 'compact58' | 'a4') {
		config.theme = preset;
		if (preset === 'supermarket') {
			config.paper_size = 'thermal_80mm';
			config.font_family = 'monospace';
			config.font_size = 11;
			config.divider_style = 'dashed';
			config.show_savings_banner = true;
			config.show_upi_qr = true;
			config.visible_columns = {
				sno: false,
				description: true,
				generic_name: false,
				batch_no: false,
				expiry: false,
				rack: false,
				mrp: true,
				rate: true,
				qty: true,
				discount: true,
				tax: false,
				total: true
			};
		} else if (preset === 'pharmacy') {
			config.paper_size = 'thermal_80mm';
			config.font_family = 'monospace';
			config.font_size = 10;
			config.divider_style = 'solid';
			config.show_dl_no = true;
			config.show_patient_info = true;
			config.show_savings_banner = true;
			config.visible_columns = {
				sno: false,
				description: true,
				generic_name: true,
				batch_no: true,
				expiry: true,
				rack: true,
				mrp: true,
				rate: true,
				qty: true,
				discount: false,
				tax: false,
				total: true
			};
		} else if (preset === 'compact58') {
			config.paper_size = 'thermal_58mm';
			config.font_family = 'monospace';
			config.font_size = 9;
			config.divider_style = 'dotted';
			config.store_name_size = 14;
			config.show_logo = false;
			config.visible_columns = {
				sno: false,
				description: true,
				generic_name: false,
				batch_no: false,
				expiry: false,
				rack: false,
				mrp: false,
				rate: false,
				qty: true,
				discount: false,
				tax: false,
				total: true
			};
		} else if (preset === 'a4') {
			config.paper_size = 'a4';
			config.font_family = 'sans';
			config.font_size = 12;
			config.divider_style = 'solid';
			config.visible_columns = {
				sno: true,
				description: true,
				generic_name: true,
				batch_no: true,
				expiry: true,
				rack: true,
				mrp: true,
				rate: true,
				qty: true,
				discount: true,
				tax: true,
				total: true
			};
		}
	}

	async function saveConfiguration() {
		saving = true;
		saveSuccess = false;
		errorMessage = '';
		try {
			await api.updateTemplateStudioConfig(config);
			saveSuccess = true;
			setTimeout(() => (saveSuccess = false), 4000);
		} catch (e: any) {
			errorMessage = e?.message || 'Failed to save bill template configuration';
		} finally {
			saving = false;
		}
	}

	function getDividerLine(style?: string) {
		if (style === 'solid') return '────────────────────────────────────────';
		if (style === 'double') return '════════════════════════════════════════';
		if (style === 'dotted') return '········································';
		return '----------------------------------------';
	}
</script>

<svelte:head>
	<title>Bill Template Studio | Visual Customizer & Receipt Layouts</title>
</svelte:head>

<div class="studio-container" id="template-studio-app">
	<!-- HEADER BAR -->
	<header class="studio-header">
		<div>
			<div class="header-breadcrumbs">
				<a href="/settings/templates">Billing Templates</a> &gt; <span>Template Studio Customizer</span>
			</div>
			<h2 class="studio-title">🛠️ Bill Template Studio</h2>
			<p class="studio-subtitle">
				Live visual receipt customizer for Supermarket, Grocery, and DPCO Pharmacy thermal bills
			</p>
		</div>

		<div class="header-actions">
			{#if saveSuccess}
				<span class="save-toast">✓ Settings saved & applied live!</span>
			{/if}
			{#if errorMessage}
				<span class="error-toast">{errorMessage}</span>
			{/if}
			<button
				type="button"
				class="save-btn"
				disabled={saving}
				onclick={saveConfiguration}
				id="btn-save-studio"
			>
				{#if saving}
					Saving Changes...
				{:else}
					💾 Save & Publish Layout
				{/if}
			</button>
		</div>
	</header>

	<!-- 2-COLUMN STUDIO WORKBENCH -->
	<div class="studio-workbench">
		<!-- LEFT SIDEBAR: CUSTOMIZER CONTROLS -->
		<aside class="controls-sidebar">
			<!-- PRESET SELECTOR -->
			<div class="control-section">
				<div class="section-label">⚡ Fast One-Click Industry Presets</div>
				<div class="preset-grid">
					<button
						type="button"
						class="preset-card"
						class:active={config.theme === 'supermarket'}
						onclick={() => applyPreset('supermarket')}
					>
						<strong>🛒 Supermarket</strong>
						<span>80mm Thermal, Savings banner, MRP vs Rate</span>
					</button>
					<button
						type="button"
						class="preset-card"
						class:active={config.theme === 'pharmacy'}
						onclick={() => applyPreset('pharmacy')}
					>
						<strong>💊 Pharmacy Rx</strong>
						<span>Batch, Expiry, DL #, Doctor & Patient info</span>
					</button>
					<button
						type="button"
						class="preset-card"
						class:active={config.theme === 'compact58'}
						onclick={() => applyPreset('compact58')}
					>
						<strong>🧾 Compact 58mm</strong>
						<span>Mini thermal rolls, clean minimal columns</span>
					</button>
					<button
						type="button"
						class="preset-card"
						class:active={config.theme === 'a4'}
						onclick={() => applyPreset('a4')}
					>
						<strong>📄 Full A4 / A5</strong>
						<span>Standard paper, all tax & logistics columns</span>
					</button>
				</div>
			</div>

			<!-- PAPER SIZE & TYPOGRAPHY -->
			<div class="control-section">
				<div class="section-label">📐 Paper Size & Dimensions</div>
				<div class="radio-group">
					<label class="radio-label">
						<input type="radio" bind:group={config.paper_size} value="thermal_80mm" />
						80mm Thermal POS
					</label>
					<label class="radio-label">
						<input type="radio" bind:group={config.paper_size} value="thermal_58mm" />
						58mm Mini POS
					</label>
					<label class="radio-label">
						<input type="radio" bind:group={config.paper_size} value="a4" />
						A4 Sheet
					</label>
					<label class="radio-label">
						<input type="radio" bind:group={config.paper_size} value="a5" />
						A5 Sheet
					</label>
				</div>

				<div class="field-row" style="margin-top: 0.75rem;">
					<label for="cfg-font-fam">Font Family:</label>
					<select id="cfg-font-fam" bind:value={config.font_family}>
						<option value="monospace">Thermal Monospace (Receipt Printer)</option>
						<option value="sans">Modern Sans-Serif</option>
						<option value="serif">Classic Serif</option>
					</select>
				</div>

				<div class="field-row">
					<label for="cfg-font-sz">Base Font Size: {config.font_size}px</label>
					<input
						id="cfg-font-sz"
						type="range"
						min="8"
						max="14"
						step="1"
						bind:value={config.font_size}
					/>
				</div>

				<div class="field-row">
					<label for="cfg-divider">Line Divider Style:</label>
					<select id="cfg-divider" bind:value={config.divider_style}>
						<option value="dashed">Dashed (----------------)</option>
						<option value="solid">Solid (────────────────)</option>
						<option value="double">Double (════════════════)</option>
						<option value="dotted">Dotted (················)</option>
					</select>
				</div>
			</div>

			<!-- HEADER & BRANDING -->
			<div class="control-section">
				<div class="section-label">🏪 Store Header & Regulatory Badges</div>
				<div class="toggle-list">
					<label class="toggle-row">
						<input type="checkbox" bind:checked={config.show_logo} />
						<span>Show Business Logo</span>
					</label>
					{#if config.show_logo}
						<div class="slider-subfield">
							<label for="cfg-logo-h">Logo Height: {config.logo_height}px</label>
							<input id="cfg-logo-h" type="range" min="24" max="80" bind:value={config.logo_height} />
						</div>
					{/if}

					<label class="toggle-row">
						<input type="checkbox" bind:checked={config.show_store_name} />
						<span>Show Store Name</span>
					</label>
					{#if config.show_store_name}
						<div class="slider-subfield">
							<label for="cfg-name-sz">Store Name Size: {config.store_name_size}px</label>
							<input id="cfg-name-sz" type="range" min="14" max="26" bind:value={config.store_name_size} />
						</div>
					{/if}

					<label class="toggle-row">
						<input type="checkbox" bind:checked={config.show_address} />
						<span>Show Store Address & Contact</span>
					</label>

					<label class="toggle-row">
						<input type="checkbox" bind:checked={config.show_dl_no} />
						<span>Show Drug License No. (Pharmacy compliance)</span>
					</label>

					<label class="toggle-row">
						<input type="checkbox" bind:checked={config.show_fssai_no} />
						<span>Show FSSAI No. (Grocery/Food compliance)</span>
					</label>

					<label class="toggle-row">
						<input type="checkbox" bind:checked={config.show_gstin} />
						<span>Show GSTIN Registration</span>
					</label>

					<label class="toggle-row">
						<input type="checkbox" bind:checked={config.show_cashier} />
						<span>Show Cashier & Counter ID</span>
					</label>

					<label class="toggle-row">
						<input type="checkbox" bind:checked={config.show_patient_info} />
						<span>Show Doctor & Patient Details (Rx compliance)</span>
					</label>
				</div>
			</div>

			<!-- TABLE COLUMNS VISIBILITY -->
			<div class="control-section">
				<div class="section-label">📋 Item Table Column Visibility</div>
				<p class="section-desc">Toggle columns to match your store layout and counter speed</p>
				<div class="column-switches-grid">
					{#if config.visible_columns}
						<label class="col-pill" class:active={config.visible_columns.sno}>
							<input type="checkbox" bind:checked={config.visible_columns.sno} />
							<span>S.No</span>
						</label>
						<label class="col-pill" class:active={config.visible_columns.description}>
							<input type="checkbox" bind:checked={config.visible_columns.description} />
							<span>Item Name</span>
						</label>
						<label class="col-pill" class:active={config.visible_columns.generic_name}>
							<input type="checkbox" bind:checked={config.visible_columns.generic_name} />
							<span>Generic Salt</span>
						</label>
						<label class="col-pill" class:active={config.visible_columns.batch_no}>
							<input type="checkbox" bind:checked={config.visible_columns.batch_no} />
							<span>Batch #</span>
						</label>
						<label class="col-pill" class:active={config.visible_columns.expiry}>
							<input type="checkbox" bind:checked={config.visible_columns.expiry} />
							<span>Expiry Date</span>
						</label>
						<label class="col-pill" class:active={config.visible_columns.rack}>
							<input type="checkbox" bind:checked={config.visible_columns.rack} />
							<span>Rack / Shelf</span>
						</label>
						<label class="col-pill" class:active={config.visible_columns.mrp}>
							<input type="checkbox" bind:checked={config.visible_columns.mrp} />
							<span>MRP (₹)</span>
						</label>
						<label class="col-pill" class:active={config.visible_columns.rate}>
							<input type="checkbox" bind:checked={config.visible_columns.rate} />
							<span>Sale Rate</span>
						</label>
						<label class="col-pill" class:active={config.visible_columns.qty}>
							<input type="checkbox" bind:checked={config.visible_columns.qty} />
							<span>Quantity</span>
						</label>
						<label class="col-pill" class:active={config.visible_columns.discount}>
							<input type="checkbox" bind:checked={config.visible_columns.discount} />
							<span>Disc %</span>
						</label>
						<label class="col-pill" class:active={config.visible_columns.tax}>
							<input type="checkbox" bind:checked={config.visible_columns.tax} />
							<span>GST %</span>
						</label>
						<label class="col-pill" class:active={config.visible_columns.total}>
							<input type="checkbox" bind:checked={config.visible_columns.total} />
							<span>Line Total</span>
						</label>
					{/if}
				</div>
			</div>

			<!-- FOOTER & QR CODE -->
			<div class="control-section">
				<div class="section-label">🎁 Footer, UPI QR & Customer Savings</div>
				<div class="toggle-list">
					<label class="toggle-row">
						<input type="checkbox" bind:checked={config.show_savings_banner} />
						<span>Show Customer Savings Banner ("You Saved ₹...")</span>
					</label>

					<label class="toggle-row">
						<input type="checkbox" bind:checked={config.show_upi_qr} />
						<span>Show Dynamic UPI Payment QR Code</span>
					</label>
					{#if config.show_upi_qr}
						<div class="slider-subfield">
							<label for="cfg-upi-sz">UPI QR Code Size: {config.upi_qr_size}px</label>
							<input id="cfg-upi-sz" type="range" min="80" max="180" bind:value={config.upi_qr_size} />
						</div>
					{/if}

					<label class="toggle-row">
						<input type="checkbox" bind:checked={config.show_barcode} />
						<span>Show Receipt Barcode at Bottom</span>
					</label>

					<div class="field-row" style="margin-top: 0.5rem;">
						<label for="cfg-footer-msg">Custom Footer Greeting:</label>
						<textarea
							id="cfg-footer-msg"
							rows="2"
							bind:value={config.custom_footer_message}
							placeholder="Enter return policy or thank you note..."
						></textarea>
					</div>
				</div>
			</div>
		</aside>

		<!-- RIGHT SIDE: REAL-TIME INTERACTIVE CANVAS PREVIEW -->
		<main class="preview-canvas-wrap">
			<div class="canvas-toolbar">
				<div class="canvas-mode-indicator">
					<span class="live-dot"></span>
					<span>Live Interactive Preview: <strong>{config.paper_size?.toUpperCase()}</strong></span>
				</div>
				<button type="button" class="test-print-btn" onclick={() => window.print()}>
					🖨️ Test Print
				</button>
			</div>

			<div class="paper-stage">
				<!-- THERMAL 80MM / 58MM PAPER PREVIEW -->
				<div
					class="live-receipt-paper"
					class:size-58mm={config.paper_size === 'thermal_58mm'}
					class:size-80mm={config.paper_size === 'thermal_80mm'}
					class:size-a4={config.paper_size === 'a4' || config.paper_size === 'a5'}
					style="
						font-family: {config.font_family === 'monospace' ? 'Courier New, Courier, monospace' : config.font_family === 'serif' ? 'Georgia, serif' : 'system-ui, sans-serif'};
						font-size: {config.font_size}px;
					"
					id="live-studio-receipt"
				>
					<!-- STORE HEADER -->
					<div class="receipt-header">
						{#if config.show_logo && company?.logo_url}
							<img
								src={company.logo_url}
								alt="Store Logo"
								class="store-logo"
								style="height: {config.logo_height}px;"
							/>
						{/if}

						{#if config.show_store_name}
							<h3 class="store-name" style="font-size: {config.store_name_size}px;">
								{company?.name || 'SUPERMARKET & PHARMACY'}
							</h3>
						{/if}

						{#if config.show_address}
							<p class="store-info">{company?.address || '124 Commercial Avenue, MG Road'}</p>
							<p class="store-info">{company?.city || 'Pune'}, {company?.state || 'Maharashtra'} - {company?.pincode || '411001'}</p>
						{/if}

						<div class="regulatory-row">
							{#if config.show_dl_no && company?.drug_license_no}
								<span>D.L. No: {company.drug_license_no}</span>
							{/if}
							{#if config.show_fssai_no && company?.fssai_no}
								<span>FSSAI: {company.fssai_no}</span>
							{/if}
							{#if config.show_gstin && company?.gstin}
								<span>GSTIN: {company.gstin}</span>
							{/if}
						</div>

						<div class="divider">{getDividerLine(config.divider_style)}</div>

						<!-- METADATA ROWS -->
						<div class="meta-line">
							<span>Bill: {sampleBill.invoice_number}</span>
							<span>{sampleBill.date}</span>
						</div>

						{#if config.show_cashier}
							<div class="meta-line">
								<span>{sampleBill.cashier}</span>
							</div>
						{/if}

						<div class="meta-line">
							<span>Customer: {sampleBill.customer}</span>
						</div>

						{#if config.show_patient_info}
							<div class="meta-line rx-compliance-line">
								<span>Pt: {sampleBill.patient}</span>
								<span>Dr: {sampleBill.doctor}</span>
							</div>
						{/if}

						<div class="divider">{getDividerLine(config.divider_style)}</div>
					</div>

					<!-- ITEM TABLE -->
					<div class="items-container">
						<!-- TABLE HEADER -->
						<div class="item-table-header">
							{#if config.visible_columns?.sno}<span class="col-sno">#</span>{/if}
							{#if config.visible_columns?.description}<span class="col-desc">ITEM</span>{/if}
							{#if config.visible_columns?.qty}<span class="col-qty">QTY</span>{/if}
							{#if config.visible_columns?.mrp}<span class="col-mrp">MRP</span>{/if}
							{#if config.visible_columns?.rate}<span class="col-rate">RATE</span>{/if}
							{#if config.visible_columns?.discount}<span class="col-disc">DISC</span>{/if}
							{#if config.visible_columns?.tax}<span class="col-tax">GST</span>{/if}
							{#if config.visible_columns?.total}<span class="col-tot">TOTAL</span>{/if}
						</div>

						<div class="divider">{getDividerLine(config.divider_style)}</div>

						<!-- ITEMS LIST -->
						{#each sampleBill.items as item}
							<div class="receipt-item-row">
								<div class="main-item-line">
									{#if config.visible_columns?.sno}<span class="col-sno">{item.sno}</span>{/if}
									{#if config.visible_columns?.description}
										<span class="col-desc item-name-bold">{item.name}</span>
									{/if}
									{#if config.visible_columns?.qty}<span class="col-qty">{item.qty}</span>{/if}
									{#if config.visible_columns?.mrp}<span class="col-mrp">₹{item.mrp.toFixed(2)}</span>{/if}
									{#if config.visible_columns?.rate}<span class="col-rate">₹{item.rate.toFixed(2)}</span>{/if}
									{#if config.visible_columns?.discount}<span class="col-disc">{item.disc}</span>{/if}
									{#if config.visible_columns?.tax}<span class="col-tax">{item.tax}</span>{/if}
									{#if config.visible_columns?.total}<span class="col-tot">₹{item.total.toFixed(2)}</span>{/if}
								</div>

								<!-- SECONDARY ROW FOR GENERIC & BATCH INFO -->
								{#if (config.visible_columns?.generic_name && item.generic) || (config.visible_columns?.batch_no && item.batch) || (config.visible_columns?.rack && item.rack)}
									<div class="sub-item-line">
										{#if config.visible_columns?.generic_name && item.generic}
											<span class="sub-generic">Salt: {item.generic}</span>
										{/if}
										{#if config.visible_columns?.batch_no && item.batch}
											<span class="sub-batch">B:{item.batch} Exp:{item.exp}</span>
										{/if}
										{#if config.visible_columns?.rack && item.rack}
											<span class="sub-rack">📍 {item.rack}</span>
										{/if}
									</div>
								{/if}
							</div>
						{/each}

						<div class="divider">{getDividerLine(config.divider_style)}</div>
					</div>

					<!-- TOTALS & NET PAYABLE -->
					<div class="receipt-totals">
						<div class="tot-row">
							<span>Subtotal (MRP Gross):</span>
							<span>₹{sampleBill.subtotal_mrp.toFixed(2)}</span>
						</div>

						{#if config.show_savings_banner}
							<div class="tot-row savings-highlight">
								<span>Customer Retail Discount:</span>
								<span>- ₹{sampleBill.savings.toFixed(2)}</span>
							</div>
						{/if}

						<div class="divider" style="margin: 0.2rem 0;">{getDividerLine(config.divider_style)}</div>

						<div class="tot-row grand-total-highlight">
							<strong>NET PAYABLE:</strong>
							<strong class="grand-num">₹{sampleBill.net_total.toFixed(2)}</strong>
						</div>

						<div class="divider" style="margin: 0.2rem 0;">{getDividerLine(config.divider_style)}</div>
					</div>

					<!-- SAVINGS BADGE -->
					{#if config.show_savings_banner}
						<div class="receipt-savings-banner">
							*** YOU SAVED ₹{sampleBill.savings.toFixed(2)} ON TODAY'S PURCHASE ***
						</div>
					{/if}

					<!-- DYNAMIC UPI QR CODE -->
					{#if config.show_upi_qr}
						<div class="upi-qr-section">
							<span class="upi-title">Scan to Pay via UPI</span>
							<div class="qr-mock-box" style="width: {config.upi_qr_size}px; height: {config.upi_qr_size}px;">
								<svg viewBox="0 0 100 100" class="qr-svg">
									<rect width="100" height="100" fill="#fff" />
									<rect x="10" y="10" width="25" height="25" fill="#000" />
									<rect x="15" y="15" width="15" height="15" fill="#fff" />
									<rect x="19" y="19" width="7" height="7" fill="#000" />

									<rect x="65" y="10" width="25" height="25" fill="#000" />
									<rect x="70" y="15" width="15" height="15" fill="#fff" />
									<rect x="74" y="19" width="7" height="7" fill="#000" />

									<rect x="10" y="65" width="25" height="25" fill="#000" />
									<rect x="15" y="70" width="15" height="15" fill="#fff" />
									<rect x="19" y="74" width="7" height="7" fill="#000" />

									<!-- QR Pattern Data -->
									<rect x="42" y="15" width="6" height="6" fill="#000" />
									<rect x="52" y="25" width="6" height="6" fill="#000" />
									<rect x="42" y="45" width="16" height="16" fill="#000" />
									<rect x="65" y="55" width="8" height="8" fill="#000" />
									<rect x="75" y="75" width="10" height="10" fill="#000" />
									<rect x="55" y="75" width="6" height="12" fill="#000" />
								</svg>
							</div>
							<span class="upi-id">{company?.upi_id || 'storename@okaxis'}</span>
						</div>
					{/if}

					<!-- RECEIPT BARCODE -->
					{#if config.show_barcode}
						<div class="receipt-barcode-wrap">
							<div class="barcode-lines">
								|||||| |||| |||||||| ||||| ||||||| |||||| |||||
							</div>
							<span class="barcode-num">*{sampleBill.invoice_number}*</span>
						</div>
					{/if}

					<!-- FOOTER MESSAGE -->
					{#if config.custom_footer_message}
						<div class="receipt-footer-msg">
							<p>{config.custom_footer_message}</p>
						</div>
					{/if}
				</div>
			</div>
		</main>
	</div>
</div>

<style>
	.studio-container {
		display: flex;
		flex-direction: column;
		height: calc(100vh - 4.5rem);
		background: #0b0f19;
		color: #f1f5f9;
		overflow: hidden;
	}

	.studio-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 0.875rem 1.5rem;
		background: #111827;
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
	}
	.header-breadcrumbs {
		font-size: 0.75rem;
		color: #94a3b8;
	}
	.header-breadcrumbs a {
		color: #38bdf8;
		text-decoration: none;
	}
	.studio-title {
		margin: 0.15rem 0 0 0;
		font-size: 1.25rem;
		font-weight: 800;
	}
	.studio-subtitle {
		margin: 0;
		font-size: 0.75rem;
		color: #94a3b8;
	}
	.header-actions {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}
	.save-toast {
		font-size: 0.8125rem;
		color: #34d399;
		font-weight: 600;
	}
	.error-toast {
		font-size: 0.8125rem;
		color: #ef4444;
		font-weight: 600;
	}
	.save-btn {
		background: #3b82f6;
		color: #fff;
		border: none;
		padding: 0.5rem 1.25rem;
		border-radius: 8px;
		font-weight: 700;
		font-size: 0.875rem;
		cursor: pointer;
		box-shadow: 0 0 15px rgba(59, 130, 246, 0.3);
		transition: all 0.2s;
	}
	.save-btn:hover:not(:disabled) {
		background: #2563eb;
		transform: translateY(-1px);
	}
	.save-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	/* WORKBENCH LAYOUT */
	.studio-workbench {
		display: grid;
		grid-template-columns: 420px 1fr;
		flex: 1;
		min-height: 0;
		overflow: hidden;
	}

	/* CONTROLS SIDEBAR */
	.controls-sidebar {
		background: #111827;
		border-right: 1px solid rgba(255, 255, 255, 0.08);
		overflow-y: auto;
		padding: 1.25rem;
		display: flex;
		flex-direction: column;
		gap: 1.5rem;
	}
	.control-section {
		display: flex;
		flex-direction: column;
		gap: 0.625rem;
		background: #1f2937;
		padding: 1rem;
		border-radius: 8px;
		border: 1px solid rgba(255, 255, 255, 0.05);
	}
	.section-label {
		font-size: 0.8125rem;
		font-weight: 700;
		color: #cbd5e1;
		display: flex;
		align-items: center;
		gap: 0.4rem;
	}
	.section-desc {
		margin: 0;
		font-size: 0.6875rem;
		color: #94a3b8;
	}

	.preset-grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 0.5rem;
	}
	.preset-card {
		background: #111827;
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: 6px;
		padding: 0.625rem;
		text-align: left;
		cursor: pointer;
		display: flex;
		flex-direction: column;
		gap: 0.2rem;
		transition: all 0.15s;
	}
	.preset-card:hover {
		border-color: #38bdf8;
		background: #1e293b;
	}
	.preset-card.active {
		border-color: #38bdf8;
		background: rgba(56, 189, 248, 0.12);
		box-shadow: 0 0 10px rgba(56, 189, 248, 0.2);
	}
	.preset-card strong {
		font-size: 0.75rem;
		color: #f1f5f9;
	}
	.preset-card span {
		font-size: 0.625rem;
		color: #94a3b8;
		line-height: 1.2;
	}

	.radio-group {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 0.4rem;
	}
	.radio-label {
		font-size: 0.75rem;
		display: flex;
		align-items: center;
		gap: 0.4rem;
		color: #cbd5e1;
		cursor: pointer;
	}
	.field-row {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}
	.field-row label {
		font-size: 0.6875rem;
		color: #94a3b8;
	}
	.field-row select, .field-row textarea {
		background: #111827;
		border: 1px solid rgba(255, 255, 255, 0.15);
		border-radius: 4px;
		color: #fff;
		padding: 0.4rem;
		font-size: 0.75rem;
	}

	.toggle-list {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}
	.toggle-row {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 0.75rem;
		color: #cbd5e1;
		cursor: pointer;
	}
	.slider-subfield {
		margin-left: 1.5rem;
		display: flex;
		flex-direction: column;
		gap: 0.15rem;
	}
	.slider-subfield label {
		font-size: 0.625rem;
		color: #94a3b8;
	}

	.column-switches-grid {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 0.4rem;
	}
	.col-pill {
		background: #111827;
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: 4px;
		padding: 0.35rem 0.4rem;
		font-size: 0.6875rem;
		display: flex;
		align-items: center;
		gap: 0.35rem;
		cursor: pointer;
		color: #94a3b8;
	}
	.col-pill.active {
		border-color: #38bdf8;
		color: #38bdf8;
		background: rgba(56, 189, 248, 0.08);
	}

	/* CANVAS PREVIEW */
	.preview-canvas-wrap {
		background: #030712;
		display: flex;
		flex-direction: column;
		overflow-y: auto;
		align-items: center;
		padding: 1.5rem;
	}
	.canvas-toolbar {
		width: 100%;
		max-width: 480px;
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 1rem;
	}
	.canvas-mode-indicator {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 0.75rem;
		color: #94a3b8;
	}
	.live-dot {
		width: 8px;
		height: 8px;
		background: #34d399;
		border-radius: 50%;
		box-shadow: 0 0 8px #34d399;
	}
	.test-print-btn {
		background: rgba(255, 255, 255, 0.08);
		border: 1px solid rgba(255, 255, 255, 0.15);
		color: #f1f5f9;
		padding: 0.3rem 0.65rem;
		border-radius: 4px;
		font-size: 0.75rem;
		cursor: pointer;
	}

	.paper-stage {
		display: flex;
		justify-content: center;
		width: 100%;
	}

	/* RECEIPT PAPER SIMULATOR */
	.live-receipt-paper {
		background: #ffffff;
		color: #000000;
		box-shadow: 0 20px 40px rgba(0, 0, 0, 0.6), 0 0 0 1px rgba(0, 0, 0, 0.1);
		border-radius: 2px;
		padding: 1.25rem 1rem;
		line-height: 1.3;
		box-sizing: border-box;
		transition: width 0.2s ease;
	}
	.live-receipt-paper.size-80mm {
		width: 340px; /* ~80mm at screen dpi */
	}
	.live-receipt-paper.size-58mm {
		width: 250px; /* ~58mm */
		padding: 0.75rem 0.5rem;
	}
	.live-receipt-paper.size-a4 {
		width: 520px;
		padding: 2rem 1.5rem;
	}

	.receipt-header {
		text-align: center;
	}
	.store-logo {
		max-width: 100%;
		object-fit: contain;
		margin-bottom: 0.25rem;
	}
	.store-name {
		margin: 0.1rem 0;
		font-weight: 800;
		text-transform: uppercase;
	}
	.store-info {
		margin: 0.05rem 0;
		font-size: 0.9em;
	}
	.regulatory-row {
		margin-top: 0.25rem;
		font-size: 0.85em;
		font-weight: 700;
		display: flex;
		flex-wrap: wrap;
		justify-content: center;
		gap: 0.5rem;
	}
	.divider {
		overflow: hidden;
		white-space: nowrap;
		margin: 0.35rem 0;
		letter-spacing: -1px;
	}
	.meta-line {
		display: flex;
		justify-content: space-between;
		font-size: 0.85em;
		margin: 0.1rem 0;
	}
	.rx-compliance-line {
		background: #f1f5f9;
		padding: 0.1rem 0.25rem;
		font-weight: 700;
	}

	/* ITEMS TABLE */
	.items-container {
		margin: 0.25rem 0;
	}
	.item-table-header {
		display: flex;
		font-weight: 800;
		font-size: 0.85em;
	}
	.receipt-item-row {
		margin: 0.35rem 0;
	}
	.main-item-line {
		display: flex;
		align-items: baseline;
	}
	.col-sno { width: 18px; }
	.col-desc { flex: 1; min-width: 0; }
	.col-qty { width: 35px; text-align: center; }
	.col-mrp { width: 50px; text-align: right; text-decoration: line-through; color: #555; }
	.col-rate { width: 50px; text-align: right; }
	.col-disc { width: 40px; text-align: right; color: #16a34a; }
	.col-tax { width: 35px; text-align: right; }
	.col-tot { width: 60px; text-align: right; font-weight: 700; }

	.item-name-bold {
		font-weight: 700;
	}
	.sub-item-line {
		font-size: 0.8em;
		color: #444;
		display: flex;
		flex-wrap: wrap;
		gap: 0.4rem;
		padding-left: 18px;
	}
	.sub-generic {
		font-style: italic;
	}

	/* TOTALS */
	.receipt-totals {
		margin-top: 0.25rem;
	}
	.tot-row {
		display: flex;
		justify-content: space-between;
		font-size: 0.9em;
		margin: 0.1rem 0;
	}
	.savings-highlight {
		color: #16a34a;
		font-weight: 700;
	}
	.grand-total-highlight {
		font-size: 1.15em;
		padding: 0.15rem 0;
	}
	.grand-num {
		font-size: 1.25em;
	}

	.receipt-savings-banner {
		text-align: center;
		font-weight: 800;
		font-size: 0.85em;
		padding: 0.35rem;
		border: 1px dashed #000;
		margin: 0.5rem 0;
	}

	.upi-qr-section {
		display: flex;
		flex-direction: column;
		align-items: center;
		margin: 0.5rem 0;
		text-align: center;
	}
	.upi-title {
		font-size: 0.8em;
		font-weight: 700;
		margin-bottom: 0.25rem;
	}
	.qr-mock-box {
		border: 1px solid #000;
		padding: 4px;
		background: #fff;
	}
	.qr-svg {
		width: 100%;
		height: 100%;
	}
	.upi-id {
		font-size: 0.75em;
		color: #333;
		margin-top: 0.2rem;
	}

	.receipt-barcode-wrap {
		text-align: center;
		margin: 0.5rem 0;
	}
	.barcode-lines {
		font-size: 1.1em;
		letter-spacing: 2px;
		transform: scaleY(1.4);
	}
	.barcode-num {
		font-size: 0.75em;
		display: block;
		margin-top: 0.25rem;
	}

	.receipt-footer-msg {
		text-align: center;
		font-size: 0.8em;
		margin-top: 0.5rem;
		line-height: 1.3;
	}
	.receipt-footer-msg p {
		margin: 0.1rem 0;
	}
</style>
