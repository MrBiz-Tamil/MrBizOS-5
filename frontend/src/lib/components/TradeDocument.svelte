<script lang="ts">
	import { amountInWords } from '$lib/words';

	let { type, data, extras = {} }: { type: string; data: any; extras?: Record<string, string> } = $props();

	const co = $derived(data.company || {});
	const prof = $derived(data.profile || {});
	const sh = $derived(data.shipment || {});
	const inv = $derived(data.invoice || null);
	const tr = $derived(data.trade || null);
	const pk = $derived(data.packages || []);
	const pt = $derived(data.packing_totals || { packages: 0, net_weight_kg: '0', gross_weight_kg: '0' });
	const lc = $derived(data.landed_cost || null);
	const cur = $derived(tr?.currency || 'USD');

	const fmt = (v: any, d = 2) => (v === null || v === undefined || v === '' ? '' : Number(v).toLocaleString('en-US', { minimumFractionDigits: d, maximumFractionDigits: d }));
	const dt = (s?: string | null) => (s ? new Date(s).toLocaleDateString('en-GB', { day: '2-digit', month: 'short', year: 'numeric' }) : '');
	const today = () => new Date().toLocaleDateString('en-GB', { day: '2-digit', month: 'short', year: 'numeric' });
	const coAddress = $derived([co.address, co.city, co.state, co.pincode].filter(Boolean).join(', '));
	const x = (k: string, fallback = '') => extras[k] || fallback;

	const titles: Record<string, string> = {
		'commercial-invoice': 'COMMERCIAL INVOICE', 'proforma-invoice': 'PROFORMA INVOICE', 'packing-list': 'PACKING LIST',
		'invoice-cum-packing-list': 'INVOICE CUM PACKING LIST', 'bill-of-exchange': 'BILL OF EXCHANGE', 'shipping-instructions': "SHIPPER'S LETTER OF INSTRUCTION",
		'bank-cover-letter': 'COVERING LETTER TO BANK', 'coo-application': 'APPLICATION FOR CERTIFICATE OF ORIGIN', 'purchase-order': 'PURCHASE ORDER',
		'remittance-request': 'REQUEST FOR OUTWARD REMITTANCE', grn: 'GOODS RECEIVED NOTE', 'landed-cost-sheet': 'LANDED COST SHEET'
	};

	const lineFc = (i: number) => (tr?.fc_lines || []).find((l: any) => l.line_no === i + 1) || {};
	const isTaxDoc = $derived(type !== 'proforma-invoice');
	const taxStatement = $derived(
		!tr
			? ''
			: tr.export_type === 'LUT' || tr.export_type === 'SEZ_LUT'
				? `SUPPLY MEANT FOR EXPORT UNDER LETTER OF UNDERTAKING WITHOUT PAYMENT OF INTEGRATED TAX${tr.lut_ref ? ' (LUT No. ' + tr.lut_ref + (prof.lut_valid_to ? ', valid up to ' + dt(prof.lut_valid_to) : '') + ')' : ''}`
				: `SUPPLY MEANT FOR EXPORT ON PAYMENT OF INTEGRATED TAX (IGST Rs. ${fmt(inv?.igst_total)} paid; refund claimed)`
	);
</script>

{#snippet seller()}
	<div class="box">
		<div class="lbl">Exporter</div>
		<strong>{co.legal_name || co.name}</strong><br />
		{coAddress}<br />
		{#if co.gstin}GSTIN: {co.gstin}<br />{/if}
		{#if prof.iec_code}IEC: {prof.iec_code}<br />{/if}
		{#if co.email}{co.email}{/if}{#if co.phone} · {co.phone}{/if}
	</div>
{/snippet}

{#snippet parties()}
	<div class="grid2">
		<div class="box">
			<div class="lbl">Consignee</div>
			<strong>{tr?.consignee_name || inv?.customer_name || sh.counterparty_name}</strong><br />
			{tr?.consignee_address || inv?.billing_address || ''}
			{#if tr?.buyer_country}<br />{tr.buyer_country}{/if}
			{#if tr?.buyer_tax_id}<br />Tax ID: {tr.buyer_tax_id}{/if}
		</div>
		<div class="box">
			<div class="lbl">Buyer (if other than consignee)</div>
			{#if tr?.consignee_name && tr.consignee_name !== inv?.customer_name}
				<strong>{inv?.customer_name}</strong><br />{inv?.billing_address}
			{:else}Same as consignee{/if}
			{#if tr?.notify_party}<div class="lbl" style="margin-top: 6px;">Notify party</div>{tr.notify_party}{/if}
		</div>
	</div>
{/snippet}

{#snippet shipmentBox()}
	<div class="grid4">
		<div class="box"><div class="lbl">Country of origin</div>{tr?.country_origin || 'India'}</div>
		<div class="box"><div class="lbl">Country of destination</div>{tr?.country_destination || ''}</div>
		<div class="box"><div class="lbl">Port of loading</div>{tr?.port_loading || sh.port_loading}</div>
		<div class="box"><div class="lbl">Port of discharge</div>{tr?.port_discharge || sh.port_discharge}</div>
		<div class="box"><div class="lbl">Final destination</div>{tr?.final_destination || ''}</div>
		<div class="box"><div class="lbl">Vessel / flight</div>{tr?.vessel_flight || sh.vessel_flight}</div>
		<div class="box"><div class="lbl">Incoterm</div>{tr?.incoterm || sh.incoterm}</div>
		<div class="box"><div class="lbl">Payment terms</div>{tr?.payment_terms || ''}</div>
		{#if sh.bl_awb_no}<div class="box"><div class="lbl">BL / AWB</div>{sh.bl_awb_no} {dt(sh.bl_awb_date)}</div>{/if}
		{#if sh.container_no}<div class="box"><div class="lbl">Container</div>{sh.container_no}</div>{/if}
		{#if sh.customs_ref_no}<div class="box"><div class="lbl">Shipping bill</div>{sh.customs_ref_no} {dt(sh.customs_ref_date)} {sh.port_code}</div>{/if}
	</div>
{/snippet}

{#snippet goods(showPrice: boolean)}
	<table>
		<thead>
			<tr><th>#</th><th style="text-align: left;">Description of goods</th><th>HS code</th><th>Qty</th>{#if showPrice}<th>Unit price ({cur})</th><th>Amount ({cur})</th>{/if}</tr>
		</thead>
		<tbody>
			{#each inv?.items || [] as it, i}
				<tr>
					<td>{i + 1}</td><td style="text-align: left;">{it.item_description}</td><td>{it.hsn_sac}</td>
					<td>{fmt(it.quantity, 3).replace(/\.?0+$/, '')} {it.unit_code}</td>
					{#if showPrice}<td style="text-align: right;">{fmt(lineFc(i).fc_unit_price, 4).replace(/0{1,2}$/, '')}</td><td style="text-align: right;">{fmt(lineFc(i).fc_amount)}</td>{/if}
				</tr>
			{/each}
		</tbody>
	</table>
{/snippet}

{#snippet totals()}
	{#if tr}
		<div class="totals">
			{#if Number(tr.fc_freight) > 0 || Number(tr.fc_insurance) > 0}
				<div><span>FOB value</span><span>{cur} {fmt(tr.fc_fob_value)}</span></div>
				{#if Number(tr.fc_freight) > 0}<div><span>Freight</span><span>{cur} {fmt(tr.fc_freight)}</span></div>{/if}
				{#if Number(tr.fc_insurance) > 0}<div><span>Insurance</span><span>{cur} {fmt(tr.fc_insurance)}</span></div>{/if}
			{/if}
			<div class="grand"><span>Total {tr.incoterm || ''}</span><span>{cur} {fmt(tr.fc_total)}</span></div>
		</div>
		<p style="margin-top: 6px;"><strong>Amount in words:</strong> {amountInWords(tr.fc_total, cur)}</p>
	{/if}
{/snippet}

{#snippet bank()}
	<div class="box" style="margin-top: 8px;">
		<div class="lbl">Bank details for remittance</div>
		{co.bank_account_holder || co.legal_name} · A/c {co.bank_account_no} · {co.bank_name} {co.bank_branch}<br />
		IFSC {co.bank_ifsc}{#if prof.bank_swift} · SWIFT {prof.bank_swift}{/if}{#if prof.ad_code} · AD code {prof.ad_code}{/if}
	</div>
{/snippet}

{#snippet signature(label = 'Authorised signatory')}
	<div style="margin-top: 36px; text-align: right;">
		For <strong>{co.legal_name || co.name}</strong><br /><br /><br />
		{label}
	</div>
{/snippet}

<div class="doc">
	<div class="head">
		<div>
			<div class="co">{co.legal_name || co.name}</div>
			<div class="sub">{coAddress}</div>
		</div>
		<div class="title">{titles[type] || type}</div>
	</div>

	<!-- COMMERCIAL / PROFORMA INVOICE, PACKING LIST, INVOICE-CUM-PACKING-LIST -->
	{#if ['commercial-invoice', 'proforma-invoice', 'packing-list', 'invoice-cum-packing-list'].includes(type)}
		{#if !inv || !tr}
			<p class="warn">Create the export invoice for this shipment first; this document is built from it.</p>
		{:else}
			<div class="grid2">
				{@render seller()}
				<div class="box">
					<div class="lbl">Invoice no. and date</div>
					<strong>{inv.invoice_number}</strong> · {dt(inv.invoice_date)}
					{#if sh.shipment_number}<div class="lbl" style="margin-top: 6px;">Shipment</div>{sh.shipment_number}{/if}
					{#if x('buyer_order_no')}<div class="lbl" style="margin-top: 6px;">Buyer's order no.</div>{x('buyer_order_no')}{/if}
					{#if x('lc_no')}<div class="lbl" style="margin-top: 6px;">Letter of credit</div>{x('lc_no')}{/if}
				</div>
			</div>
			{@render parties()}
			{@render shipmentBox()}
			{#if tr.marks_numbers}<div class="box"><div class="lbl">Marks and numbers</div>{tr.marks_numbers}</div>{/if}

			{#if type === 'commercial-invoice' || type === 'proforma-invoice' || type === 'invoice-cum-packing-list'}
				{@render goods(true)}
				{@render totals()}
				{#if type !== 'proforma-invoice'}
					<p class="stmt">{taxStatement}</p>
					<p class="small">Exchange rate used for INR value: Rs. {fmt(tr.exchange_rate, 4)} per {cur} · Taxable value Rs. {fmt(inv.taxable_amount)}{Number(inv.igst_total) > 0 ? ' · IGST Rs. ' + fmt(inv.igst_total) : ''}</p>
				{:else}
					<p class="small">This is a proforma invoice for quotation and approval purposes only. It is not a tax invoice and creates no payment obligation until confirmed.</p>
				{/if}
			{/if}

			{#if type === 'packing-list' || type === 'invoice-cum-packing-list'}
				<div style="margin-top: 14px;"><strong>Packing details</strong></div>
				<table>
					<thead><tr><th>Packages</th><th>Type</th><th style="text-align: left;">Contents</th><th>Qty</th><th>Net kg</th><th>Gross kg</th><th>Size</th></tr></thead>
					<tbody>
						{#each pk as p}
							<tr>
								<td>{p.package_from === p.package_to ? p.package_from : p.package_from + ' - ' + p.package_to}</td><td>{p.package_type}</td>
								<td style="text-align: left;">{p.description}</td><td>{Number(p.quantity) ? fmt(p.quantity, 3).replace(/\.?0+$/, '') + ' ' + p.unit : ''}</td>
								<td style="text-align: right;">{fmt(p.net_weight_kg, 3)}</td><td style="text-align: right;">{fmt(p.gross_weight_kg, 3)}</td><td>{p.dimensions}</td>
							</tr>
						{:else}
							<tr><td colspan="7" class="warn">Enter the packing list on the shipment page first.</td></tr>
						{/each}
					</tbody>
					<tfoot><tr><td colspan="4"><strong>Total {pt.packages} package(s)</strong></td><td style="text-align: right;"><strong>{fmt(pt.net_weight_kg, 3)}</strong></td><td style="text-align: right;"><strong>{fmt(pt.gross_weight_kg, 3)}</strong></td><td></td></tr></tfoot>
				</table>
			{/if}
			{#if type !== 'packing-list'}{@render bank()}{/if}
			<p class="small" style="margin-top: 10px;">We declare that this document shows the actual price of the goods described and that all particulars are true and correct.{type === 'commercial-invoice' ? ' Goods are of Indian origin.' : ''}</p>
			{@render signature()}
		{/if}

	<!-- BILL OF EXCHANGE -->
	{:else if type === 'bill-of-exchange'}
		{#if !inv || !tr}<p class="warn">Create the export invoice for this shipment first.</p>{:else}
			<div class="grid2">
				<div class="box"><div class="lbl">Exchange for</div><strong style="font-size: 1.2em;">{cur} {fmt(tr.fc_total)}</strong></div>
				<div class="box"><div class="lbl">Place and date</div>{co.city}, {x('bill_date', today())}</div>
			</div>
			<p style="margin-top: 14px; line-height: 1.8;">
				{x('tenor', 'At sight')} of this <strong>FIRST</strong> Bill of Exchange (Second of the same tenor and date being unpaid) pay to
				<strong>{x('payee', co.bank_name || 'ourselves or order')}</strong> or order the sum of
				<strong>{amountInWords(tr.fc_total, cur)}</strong> for value received against shipment of
				{(inv.items || []).map((i: any) => i.item_description).slice(0, 3).join(', ')} per Invoice No. {inv.invoice_number} dated {dt(inv.invoice_date)}
				{#if sh.bl_awb_no}, B/L or AWB No. {sh.bl_awb_no} dated {dt(sh.bl_awb_date)}{/if}.
			</p>
			{#if x('lc_no')}<p>Drawn under Letter of Credit No. <strong>{x('lc_no')}</strong> dated {x('lc_date')} issued by {x('issuing_bank')}.</p>{/if}
			<div class="grid2" style="margin-top: 14px;">
				<div class="box"><div class="lbl">To (drawee)</div><strong>{x('drawee', inv.customer_name)}</strong><br />{x('drawee_address', inv.billing_address)}</div>
				<div class="box"><div class="lbl">Drawer</div><strong>{co.legal_name || co.name}</strong><br />{coAddress}</div>
			</div>
			{@render signature('Drawer')}
		{/if}

	<!-- SHIPPER'S LETTER OF INSTRUCTION -->
	{:else if type === 'shipping-instructions'}
		<p>To: <strong>{x('forwarder', '(forwarder / shipping line)')}</strong><br />Date: {today()} · Our ref: {sh.shipment_number}</p>
		<p style="margin-top: 8px;">Please arrange shipment of the cargo below as per these instructions.</p>
		<div class="grid2">
			{@render seller()}
			<div class="box"><div class="lbl">Consignee</div><strong>{tr?.consignee_name || inv?.customer_name || sh.counterparty_name}</strong><br />{tr?.consignee_address || inv?.billing_address || ''}
				{#if tr?.notify_party}<div class="lbl" style="margin-top: 6px;">Notify party</div>{tr.notify_party}{/if}</div>
		</div>
		{@render shipmentBox()}
		<div class="grid4">
			<div class="box"><div class="lbl">Freight</div>{x('freight_terms', 'PREPAID')}</div>
			<div class="box"><div class="lbl">B/L type and originals</div>{x('bl_type', 'Original Bill of Lading, 3 originals')}</div>
			<div class="box"><div class="lbl">Invoice no.</div>{inv?.invoice_number || ''}</div>
			<div class="box"><div class="lbl">Total packages / gross weight</div>{pt.packages} / {fmt(pt.gross_weight_kg, 3)} kg</div>
		</div>
		<div class="box"><div class="lbl">Description of goods</div>{(inv?.items || []).map((i: any) => i.item_description).join('; ') || pk.map((p: any) => p.description).join('; ')}
			{#if tr?.marks_numbers}<div class="lbl" style="margin-top: 6px;">Marks and numbers</div>{tr.marks_numbers}{/if}</div>
		{#if x('special')}<div class="box"><div class="lbl">Special instructions</div>{x('special')}</div>{/if}
		<p class="small">Documents enclosed: commercial invoice, packing list{prof.iec_code ? '. IEC ' + prof.iec_code : ''}. Please confirm the booking and the cut-off dates.</p>
		{@render signature()}

	<!-- BANK COVERING LETTER -->
	{:else if type === 'bank-cover-letter'}
		<p>Date: {today()}</p>
		<p>The Manager<br /><strong>{x('bank_name', co.bank_name || '(bank)')}</strong><br />{x('bank_branch', co.bank_branch)}</p>
		<p><strong>Sub: Export documents for {x('mode', 'negotiation / collection')}: Invoice {inv?.invoice_number || ''}, {cur} {fmt(tr?.fc_total)}</strong></p>
		<p>Dear Sir,</p>
		<p style="line-height: 1.7;">
			We enclose the following export documents against shipment to <strong>{tr?.consignee_name || inv?.customer_name || sh.counterparty_name}</strong>,
			{tr?.buyer_country || ''}, for {cur} {fmt(tr?.fc_total)} ({amountInWords(tr?.fc_total || 0, cur)}), shipped from {tr?.port_loading || sh.port_loading}
			to {tr?.port_discharge || sh.port_discharge}{sh.bl_awb_no ? ' under B/L or AWB No. ' + sh.bl_awb_no + ' dated ' + dt(sh.bl_awb_date) : ''}.
			{#if x('lc_no')} The shipment is under Letter of Credit No. {x('lc_no')} dated {x('lc_date')}.{/if}
		</p>
		<table>
			<thead><tr><th style="text-align: left;">Document</th><th>Originals</th><th>Copies</th></tr></thead>
			<tbody>
				{#each (data.documents || []).filter((d: any) => d.status === 'READY' || d.status === 'SUBMITTED' || d.status === 'RECEIVED') as d}
					<tr><td style="text-align: left;">{d.label}{d.reference_no ? ' (' + d.reference_no + ')' : ''}</td><td>{x('originals', '1')}</td><td>{x('copies', '2')}</td></tr>
				{:else}
					<tr><td colspan="3" class="warn">Mark documents Ready on the shipment checklist; they are listed here.</td></tr>
				{/each}
			</tbody>
		</table>
		<p style="margin-top: 10px; line-height: 1.7;">
			Please {x('mode', 'negotiate / send for collection')} these documents and credit the proceeds to our account no. <strong>{co.bank_account_no}</strong> with you
			{#if prof.ad_code}(AD code {prof.ad_code}){/if}. Please issue the e-BRC / FIRC on realisation. {x('instructions')}
		</p>
		<p>Yours faithfully,</p>
		{@render signature()}

	<!-- CERTIFICATE OF ORIGIN APPLICATION -->
	{:else if type === 'coo-application'}
		<p>To: <strong>{x('chamber', 'The Chamber of Commerce / Issuing authority')}</strong><br />Date: {today()}</p>
		<p>We request a Certificate of Origin for the goods below.</p>
		<div class="grid2">
			{@render seller()}
			<div class="box"><div class="lbl">Consignee</div><strong>{tr?.consignee_name || inv?.customer_name || sh.counterparty_name}</strong><br />{tr?.consignee_address || inv?.billing_address || ''}<br />{tr?.buyer_country || ''}</div>
		</div>
		{@render shipmentBox()}
		<table>
			<thead><tr><th>#</th><th style="text-align: left;">Description</th><th>HS code</th><th>Qty</th><th>Value ({cur})</th></tr></thead>
			<tbody>{#each inv?.items || [] as it, i}<tr><td>{i + 1}</td><td style="text-align: left;">{it.item_description}</td><td>{it.hsn_sac}</td><td>{fmt(it.quantity, 3).replace(/\.?0+$/, '')} {it.unit_code}</td><td style="text-align: right;">{fmt(lineFc(i).fc_amount)}</td></tr>{/each}</tbody>
		</table>
		<div class="grid2">
			<div class="box"><div class="lbl">Invoice</div>{inv?.invoice_number} dated {dt(inv?.invoice_date)}</div>
			<div class="box"><div class="lbl">Origin criterion / remarks</div>{x('criterion', 'Wholly obtained / produced in India')}</div>
		</div>
		<p class="small" style="margin-top: 8px;">We declare that the goods described were produced or manufactured in {tr?.country_origin || 'India'} and that the particulars given are true and correct. Enclosed: commercial invoice, packing list.</p>
		{@render signature()}

	<!-- PURCHASE ORDER (import) -->
	{:else if type === 'purchase-order'}
		<div class="grid2">
			<div class="box"><div class="lbl">Buyer</div><strong>{co.legal_name || co.name}</strong><br />{coAddress}<br />{#if co.gstin}GSTIN: {co.gstin}<br />{/if}{#if prof.iec_code}IEC: {prof.iec_code}{/if}</div>
			<div class="box"><div class="lbl">Supplier</div><strong>{sh.counterparty_name}</strong><br />{x('supplier_address')}</div>
		</div>
		<div class="grid4">
			<div class="box"><div class="lbl">PO number</div>{x('po_no', sh.shipment_number)}</div>
			<div class="box"><div class="lbl">PO date</div>{x('po_date', today())}</div>
			<div class="box"><div class="lbl">Incoterm</div>{sh.incoterm}</div>
			<div class="box"><div class="lbl">Currency</div>{x('currency', 'USD')}</div>
			<div class="box"><div class="lbl">Port of loading</div>{sh.port_loading}</div>
			<div class="box"><div class="lbl">Port of discharge</div>{sh.port_discharge}</div>
			<div class="box"><div class="lbl">Mode</div>{sh.mode}</div>
			<div class="box"><div class="lbl">Delivery by</div>{x('delivery_by', dt(sh.eta))}</div>
		</div>
		<table>
			<thead><tr><th>#</th><th style="text-align: left;">Item</th><th>Qty</th><th>Unit price ({x('currency', 'USD')})</th></tr></thead>
			<tbody>
				{#each lc?.items || [] as it, i}
					<tr><td>{i + 1}</td><td style="text-align: left;">{it.description}</td><td>{fmt(it.quantity, 3).replace(/\.?0+$/, '')} {it.unit_code}</td><td style="text-align: right;">{x('price_' + i)}</td></tr>
				{:else}
					<tr><td colspan="4" class="warn">Link the supplier's purchase bill to this shipment to list the items here.</td></tr>
				{/each}
			</tbody>
		</table>
		<div class="box" style="margin-top: 8px;"><div class="lbl">Terms</div>Payment: {x('payment_terms', 'as agreed')}. Packing: export-worthy. Documents required: commercial invoice, packing list, bill of lading / airway bill, certificate of origin. {x('terms')}</div>
		{@render signature('Authorised signatory (buyer)')}

	<!-- REMITTANCE REQUEST (import) -->
	{:else if type === 'remittance-request'}
		<p>Date: {today()}</p>
		<p>The Manager (Foreign Exchange)<br /><strong>{x('bank_name', co.bank_name || '(bank)')}</strong></p>
		<p><strong>Sub: Request for outward remittance for imports, our ref {sh.shipment_number}</strong></p>
		<p style="line-height: 1.7;">Please remit <strong>{x('currency', 'USD')} {x('amount')}</strong> ({amountInWords(parseFloat(x('amount', '0')) || 0, x('currency', 'USD'))}) from our account no. <strong>{co.bank_account_no}</strong>{#if prof.ad_code} (AD code {prof.ad_code}){/if} towards payment for imported goods, against the documents below.</p>
		<div class="grid2">
			<div class="box"><div class="lbl">Beneficiary</div><strong>{x('ben_name', sh.counterparty_name)}</strong><br />{x('ben_address')}<br />{x('ben_country')}</div>
			<div class="box"><div class="lbl">Beneficiary bank</div><strong>{x('ben_bank')}</strong><br />SWIFT/BIC: {x('ben_swift')}<br />A/c / IBAN: {x('ben_account')}</div>
		</div>
		<div class="grid4">
			<div class="box"><div class="lbl">Supplier invoice</div>{x('supplier_invoice')} {x('supplier_invoice_date')}</div>
			<div class="box"><div class="lbl">BL / AWB</div>{sh.bl_awb_no} {dt(sh.bl_awb_date)}</div>
			<div class="box"><div class="lbl">Bill of entry</div>{sh.customs_ref_no} {dt(sh.customs_ref_date)}</div>
			<div class="box"><div class="lbl">Purpose code</div>{x('purpose_code')}</div>
		</div>
		<p class="small" style="margin-top: 8px;">Importer-exporter code: {prof.iec_code}. We confirm that the remittance is for the import of goods as per the enclosed documents, that the Bill of Entry has been filed, and that the details are true. Enclosed: supplier invoice, bill of lading / airway bill, bill of entry. {x('remarks')}</p>
		{@render signature()}

	<!-- GRN (import) -->
	{:else if type === 'grn'}
		<div class="grid4">
			<div class="box"><div class="lbl">GRN no.</div>{x('grn_no', 'GRN-' + sh.shipment_number)}</div>
			<div class="box"><div class="lbl">Received on</div>{x('received_on', today())}</div>
			<div class="box"><div class="lbl">Supplier</div>{sh.counterparty_name}</div>
			<div class="box"><div class="lbl">Shipment / BL</div>{sh.shipment_number} · {sh.bl_awb_no}</div>
			<div class="box"><div class="lbl">Bill of entry</div>{sh.customs_ref_no} {dt(sh.customs_ref_date)}</div>
			<div class="box"><div class="lbl">Container</div>{sh.container_no}</div>
			<div class="box"><div class="lbl">Received at</div>{x('location')}</div>
			<div class="box"><div class="lbl">Received by</div>{x('received_by')}</div>
		</div>
		<table>
			<thead><tr><th>#</th><th style="text-align: left;">Item</th><th>Ordered / invoiced</th><th>Received</th><th>Condition</th></tr></thead>
			<tbody>
				{#each lc?.items || [] as it, i}
					<tr><td>{i + 1}</td><td style="text-align: left;">{it.description}</td><td>{fmt(it.quantity, 3).replace(/\.?0+$/, '')} {it.unit_code}</td><td>{x('rcv_' + i, fmt(it.quantity, 3).replace(/\.?0+$/, '') + ' ' + it.unit_code)}</td><td>{x('cond_' + i, 'Good')}</td></tr>
				{:else}
					<tr><td colspan="5" class="warn">Link the supplier's purchase bill to this shipment to list the items here.</td></tr>
				{/each}
			</tbody>
		</table>
		<div class="box" style="margin-top: 8px;"><div class="lbl">Remarks (shortage / damage)</div>{x('remarks', 'Nil')}</div>
		<div class="grid2" style="margin-top: 30px;"><div>Received by: ____________________</div><div style="text-align: right;">Checked by: ____________________</div></div>

	<!-- LANDED COST SHEET (import) -->
	{:else if type === 'landed-cost-sheet'}
		{#if !lc}<p class="warn">Add the shipment's costs first.</p>{:else}
			<div class="grid4">
				<div class="box"><div class="lbl">Shipment</div>{sh.shipment_number}</div>
				<div class="box"><div class="lbl">Supplier</div>{sh.counterparty_name}</div>
				<div class="box"><div class="lbl">Supplier bill</div>{lc.purchase_bill_number || '-'}</div>
				<div class="box"><div class="lbl">Bill of entry</div>{sh.customs_ref_no} {dt(sh.customs_ref_date)}</div>
			</div>
			<table>
				<thead><tr><th style="text-align: left;">Cost</th><th style="text-align: left;">Detail</th><th>Amount (Rs.)</th><th>In landed cost?</th></tr></thead>
				<tbody>
					{#each lc.costs as c}<tr><td style="text-align: left;">{c.cost_type}</td><td style="text-align: left;">{c.description}</td><td style="text-align: right;">{fmt(c.amount_inr)}</td><td>{c.recoverable ? 'No (tax credit)' : 'Yes'}</td></tr>{/each}
				</tbody>
			</table>
			<div class="totals">
				<div><span>Goods value</span><span>Rs. {fmt(lc.goods_value_inr)}</span></div>
				<div><span>Added costs</span><span>Rs. {fmt(lc.added_costs_inr)}</span></div>
				<div class="grand"><span>Total landed cost</span><span>Rs. {fmt(lc.total_landed_inr)}</span></div>
				<div><span>Claimable IGST (not in cost)</span><span>Rs. {fmt(lc.recoverable_inr)}</span></div>
			</div>
			{#if lc.items.length}
				<div style="margin-top: 14px;"><strong>Cost per item</strong></div>
				<table>
					<thead><tr><th style="text-align: left;">Item</th><th>Qty</th><th>Value</th><th>Added cost</th><th>Landed</th><th>Per unit</th></tr></thead>
					<tbody>{#each lc.items as it}<tr><td style="text-align: left;">{it.description}</td><td>{fmt(it.quantity, 3).replace(/\.?0+$/, '')} {it.unit_code}</td><td style="text-align: right;">{fmt(it.value_inr)}</td><td style="text-align: right;">{fmt(it.allocated_cost_inr)}</td><td style="text-align: right;">{fmt(it.landed_cost_inr)}</td><td style="text-align: right;">{fmt(it.landed_unit_cost_inr, 4)}</td></tr>{/each}</tbody>
				</table>
			{/if}
			<p class="small" style="margin-top: 8px;">Added costs are spread over the items in proportion to their value. IGST paid on import is claimed as input tax credit and is excluded.</p>
		{/if}
	{/if}
</div>

<style>
	.doc { background: #fff; color: #111; padding: 22px 26px; max-width: 820px; margin: 0 auto; font-size: 12px; line-height: 1.45; font-family: Arial, Helvetica, sans-serif; }
	.head { display: flex; justify-content: space-between; align-items: flex-start; border-bottom: 2px solid #111; padding-bottom: 8px; margin-bottom: 10px; gap: 12px; }
	.co { font-size: 17px; font-weight: 700; }
	.sub { color: #444; max-width: 380px; }
	.title { font-size: 17px; font-weight: 800; letter-spacing: 0.04em; text-align: right; }
	.grid2 { display: grid; grid-template-columns: 1fr 1fr; gap: 0; }
	.grid4 { display: grid; grid-template-columns: repeat(4, 1fr); gap: 0; }
	.box { border: 1px solid #999; padding: 6px 8px; margin: -1px 0 0 -1px; }
	.lbl { font-size: 9.5px; text-transform: uppercase; color: #555; letter-spacing: 0.04em; }
	table { width: 100%; border-collapse: collapse; margin-top: 10px; }
	th, td { border: 1px solid #999; padding: 4px 6px; text-align: center; vertical-align: top; }
	th { background: #eee; font-size: 11px; }
	.totals { margin-left: auto; width: 320px; margin-top: 8px; }
	.totals div { display: flex; justify-content: space-between; padding: 2px 0; }
	.totals .grand { border-top: 2px solid #111; font-weight: 700; font-size: 13px; margin-top: 3px; padding-top: 3px; }
	.stmt { font-weight: 700; margin-top: 8px; }
	.small { font-size: 10.5px; color: #333; }
	.warn { color: #b45309; font-weight: 600; }
	p { margin: 6px 0; }
	@media print { .doc { padding: 0; } }
</style>
