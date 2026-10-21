<script lang="ts">
	import '../styles/templates.css';
	import type { Invoice, Tenant, InvoiceItem } from '$lib/api';

	interface ExtraDetails {
		poNumber?: string;
		vehicleNumber?: string;
		ewayBill?: string;
		transporterName?: string;
		lrNumber?: string;
		deliveryPlace?: string;
		terms?: string;
		jurisdictionCity?: string;
		interestRate?: string;
		returnPolicy?: string;
		reverseCharge?: boolean;
		subscriptionPeriodStart?: string;
		subscriptionPeriodEnd?: string;
		grossWeight?: string;
		netWeight?: string;
		cashierName?: string;
		counterNumber?: string;
		projectName?: string;
		showBankDetails?: boolean;
		showUPIQR?: boolean;
		showTaxSummary?: boolean;
		compositionNoticeType?: 'COMPOSITION' | 'UNREGISTERED' | 'EXEMPT';
		compositionNotice?: string;
		thermalFooter?: string;
		signatoryTitle?: string;
		signatureUrl?: string;
	}

	let {
		invoice,
		company,
		templateId = '1',
		copyType = 'ORIGINAL',
		taxMode = 'WITH_GST',
		paperSize = 'A4',
		extraDetails = {}
	}: {
		invoice: Invoice | null;
		company: Tenant | null;
		templateId?: string;
		copyType?: 'ORIGINAL' | 'DUPLICATE' | 'TRIPLICATE' | 'OFFICE';
		taxMode?: 'WITH_GST' | 'WITHOUT_GST';
		paperSize?: 'A4' | 'A5' | 'A5_LANDSCAPE';
		extraDetails?: ExtraDetails;
	} = $props();

	function getUPIQRUrl(upiId: string, payeeName: string, amount: string | number | undefined): string {
		const amt = amount ? parseFloat(amount.toString()).toFixed(2) : '0.00';
		const cleanUpi = upiId.trim();
		const cleanName = encodeURIComponent((payeeName || 'Merchant').trim());
		const upiUri = `upi://pay?pa=${encodeURIComponent(cleanUpi)}&pn=${cleanName}&am=${amt}&cu=INR`;
		return `https://api.qrserver.com/v1/create-qr-code/?size=120x120&data=${encodeURIComponent(upiUri)}&margin=1`;
	}

	let effectiveBankName = $derived(company?.bank_name || '');
	let effectiveAccountNo = $derived(company?.bank_account_no || '');
	let effectiveIFSC = $derived(company?.bank_ifsc || '');
	let effectiveBranch = $derived(company?.bank_branch || '');
	let effectiveAccountHolder = $derived(company?.bank_account_holder || company?.legal_name || company?.name || '');
	let effectiveUPI = $derived(company?.upi_id || '');

	let effectiveSubtitle = $derived(company?.business_subtitle || '');
	let effectiveSignatoryTitle = $derived(extraDetails?.signatoryTitle || company?.signatory_title || 'Authorized Signatory');
	let effectiveSignatureUrl = $derived(extraDetails?.signatureUrl || company?.signature_url || '');

	let effectiveTerms = $derived(extraDetails?.terms || invoice?.payment_mode || company?.default_terms || 'Cash');
	let effectiveJurisdiction = $derived(extraDetails?.jurisdictionCity || invoice?.jurisdiction_city || company?.default_jurisdiction || company?.city || '');
	let effectiveInterestRate = $derived(extraDetails?.interestRate || invoice?.interest_rate || company?.default_interest_rate || '');
	let effectiveReturnPolicy = $derived(extraDetails?.returnPolicy || invoice?.return_policy || company?.default_return_policy || 'Goods once sold will not be taken back.');

	let effectiveReverseCharge = $derived(extraDetails?.reverseCharge ?? invoice?.reverse_charge ?? false);
	let effectiveSubscriptionStart = $derived(extraDetails?.subscriptionPeriodStart || invoice?.subscription_period_start || '');
	let effectiveSubscriptionEnd = $derived(extraDetails?.subscriptionPeriodEnd || invoice?.subscription_period_end || '');
	let effectiveGrossWeight = $derived(extraDetails?.grossWeight || invoice?.gross_weight || '');
	let effectiveNetWeight = $derived(extraDetails?.netWeight || invoice?.net_weight || '');
	let effectiveCashierName = $derived(extraDetails?.cashierName || invoice?.cashier_name || '');
	let effectiveCounterNumber = $derived(extraDetails?.counterNumber || invoice?.counter_number || '');
	let effectiveProjectName = $derived(extraDetails?.projectName || invoice?.project_name || '');

	let showBank = $derived(
		extraDetails?.showBankDetails ??
		invoice?.template_options?.show_bank_details ??
		company?.template_options?.show_bank_details ??
		Boolean(effectiveBankName && effectiveAccountNo)
	);

	let showUPI = $derived(
		extraDetails?.showUPIQR ??
		invoice?.template_options?.show_upi_qr ??
		company?.template_options?.show_upi_qr ??
		Boolean(effectiveUPI)
	);

	let showTaxSummary = $derived(
		extraDetails?.showTaxSummary ??
		invoice?.template_options?.show_tax_summary ??
		company?.template_options?.show_tax_summary ??
		true
	);

	// Copy Title & Badge Map
	const copyLabels: Record<string, { title: string; desc: string; class: string }> = {
		ORIGINAL: { title: 'ORIGINAL FOR RECIPIENT', desc: 'Valid for Input Tax Credit (ITC)', class: 'copy-badge-original' },
		DUPLICATE: { title: 'DUPLICATE FOR TRANSPORTER', desc: 'Statutory Transit Copy', class: 'copy-badge-duplicate' },
		TRIPLICATE: { title: 'TRIPLICATE FOR SUPPLIER', desc: 'Accounts & Book of Record', class: 'copy-badge-triplicate' },
		OFFICE: { title: 'OFFICE / EXTRA COPY', desc: 'Internal Reference & Audit', class: 'copy-badge-office' }
	};

	let copyMeta = $derived(copyLabels[copyType] || copyLabels.ORIGINAL);
	let isTaxInvoice = $derived(taxMode === 'WITH_GST');

	function formatCurrency(val: string | number | undefined): string {
		if (!val && val !== 0) return '₹0.00';
		const num = typeof val === 'string' ? parseFloat(val) : val;
		if (isNaN(num)) return '₹0.00';
		return new Intl.NumberFormat('en-IN', { style: 'currency', currency: 'INR' }).format(num);
	}

	// Signed amount for round-off lines: +0.50 / -0.24
	function fmtSigned(val: string | number | undefined): string {
		const n = typeof val === 'string' ? parseFloat(val) : (val ?? 0);
		const v = isNaN(n as number) ? 0 : (n as number);
		return (v > 0 ? '+' : v < 0 ? '-' : '') + Math.abs(v).toFixed(2);
	}

	// Plain number with Indian digit grouping and 2 decimals (no currency symbol).
	function fmtNum(val: string | number | undefined): string {
		const n = typeof val === 'string' ? parseFloat(val) : (val ?? 0);
		return new Intl.NumberFormat('en-IN', { minimumFractionDigits: 2, maximumFractionDigits: 2 }).format(isNaN(n as number) ? 0 : (n as number));
	}

	function formatDate(val: string | undefined): string {
		if (!val) return new Date().toLocaleDateString('en-IN');
		return new Date(val).toLocaleDateString('en-IN', { day: '2-digit', month: 'short', year: 'numeric' });
	}

	// Indian Rupee Amount to Words Converter
	function numberToWords(amount: number | string | undefined): string {
		if (!amount && amount !== 0) return 'Zero Rupees Only';
		const num = Math.round(typeof amount === 'string' ? parseFloat(amount) : amount);
		if (isNaN(num) || num <= 0) return 'Zero Rupees Only';

		const a = [
			'', 'One ', 'Two ', 'Three ', 'Four ', 'Five ', 'Six ', 'Seven ', 'Eight ', 'Nine ', 'Ten ',
			'Eleven ', 'Twelve ', 'Thirteen ', 'Fourteen ', 'Fifteen ', 'Sixteen ', 'Seventeen ', 'Eighteen ', 'Nineteen '
		];
		const b = ['', '', 'Twenty', 'Thirty', 'Forty', 'Fifty', 'Sixty', 'Seventy', 'Eighty', 'Ninety'];

		function inWords(n: number): string {
			let str = '';
			if (n > 99) {
				str += a[Math.floor(n / 100)] + 'Hundred ';
				n %= 100;
			}
			if (n > 19) {
				str += b[Math.floor(n / 10)] + ' ' + a[n % 10];
			} else if (n > 0) {
				str += a[n];
			}
			return str;
		}

		let crore = Math.floor(num / 10000000);
		let lakh = Math.floor((num % 10000000) / 100000);
		let thousand = Math.floor((num % 100000) / 1000);
		let remainder = num % 1000;

		let result = '';
		if (crore > 0) result += inWords(crore) + 'Crore ';
		if (lakh > 0) result += inWords(lakh) + 'Lakh ';
		if (thousand > 0) result += inWords(thousand) + 'Thousand ';
		if (remainder > 0) result += inWords(remainder);

		return 'INR ' + result.trim() + ' Only';
	}

	let grandTotalWords = $derived(numberToWords(invoice?.grand_total));

	// Hardware helpers: credit terms, transport block visibility, trade-discount column, HSN-wise summary.
	let creditDue = $derived(
		Boolean(invoice?.due_date && invoice?.invoice_date) &&
			parseFloat(invoice?.balance_due || '0') > 0 &&
			new Date(invoice!.due_date).toDateString() !== new Date(invoice!.invoice_date).toDateString() &&
			new Date(invoice!.due_date).getTime() > new Date(invoice!.invoice_date).getTime()
	);

	let hardwareHasTransport = $derived(
		Boolean(
			invoice?.vehicle_number || invoice?.transporter_name || invoice?.lr_number || invoice?.eway_bill ||
				invoice?.gross_weight || invoice?.net_weight
		)
	);

	// Delivery site: a real shipping address, or a delivery place that is more than just the supply state
	// (the invoice form stores the state name when the field is left blank).
	let hardwareSite = $derived(
		invoice?.shipping_address?.trim() ||
			(invoice?.delivery_place &&
			invoice.delivery_place.trim().toLowerCase() !== (invoice?.supply_state || '').trim().toLowerCase()
				? invoice.delivery_place
				: '')
	);

	let hasDisc = $derived((invoice?.items || []).some((it) => parseFloat(it.discount_rate || '0') > 0));

	let hsnSummary = $derived.by(() => {
		const m = new Map<string, { hsn: string; rate: number; taxable: number; cgst: number; sgst: number; igst: number }>();
		for (const it of invoice?.items || []) {
			const hsn = (it.hsn_sac || '-').trim() || '-';
			const rate = parseFloat(it.tax_rate || '0') || 0;
			const key = hsn + '|' + rate;
			const e = m.get(key) || { hsn, rate, taxable: 0, cgst: 0, sgst: 0, igst: 0 };
			e.taxable += parseFloat(it.taxable_value || '0') || 0;
			e.cgst += parseFloat(it.cgst_amount || '0') || 0;
			e.sgst += parseFloat(it.sgst_amount || '0') || 0;
			e.igst += parseFloat(it.igst_amount || '0') || 0;
			m.set(key, e);
		}
		return [...m.values()];
	});

	// Quantity for bills: whole numbers as-is, weights trimmed to at most 3 decimals (1.250 -> 1.25, 10 -> 10).
	function fmtQty(val: string | number | undefined): string {
		const n = typeof val === 'string' ? parseFloat(val) : (val ?? 0);
		if (isNaN(n as number)) return '0';
		const v = n as number;
		return v % 1 === 0 ? String(v) : v.toFixed(3).replace(/0+$/, '').replace(/\.$/, '');
	}

	let hasMrp = $derived((invoice?.items || []).some((it) => parseFloat(it.mrp || '0') > 0));

	// Bakery helpers: veg / non-veg mark (from the item name) and order details.
	function isNonVegItem(desc: string | undefined): boolean {
		const d = (desc || '').toLowerCase();
		if (/eggless|egg-free|egg free/.test(d)) return false;
		return /\b(egg|eggs|chicken|mutton|beef|pork|fish|prawn|shrimp|meat|ham|bacon|tuna|salami|sausage|non[- ]?veg)\b/.test(d);
	}

	let bakeryDueLater = $derived.by(() => {
		if (!invoice?.due_date || !invoice?.invoice_date) return false;
		return new Date(invoice.due_date).toDateString() !== new Date(invoice.invoice_date).toDateString() &&
			new Date(invoice.due_date).getTime() > new Date(invoice.invoice_date).getTime();
	});
	let bakeryPlace = $derived(
		invoice?.delivery_place && invoice.delivery_place.trim().toLowerCase() !== (invoice?.supply_state || '').trim().toLowerCase()
			? invoice.delivery_place
			: ''
	);
	let bakeryIsOrder = $derived(Boolean(invoice?.project_name) || bakeryDueLater || Boolean(bakeryPlace));

	// Pharmacy helpers: expiry formatting / short-expiry flag / savings against printed MRP.
	function parseExpiry(exp: string | undefined): Date | null {
		if (!exp) return null;
		const v = exp.trim();
		let m = v.match(/^(\d{4})-(\d{2})(?:-(\d{2}))?/);
		if (m) {
			const y = +m[1], mo = +m[2];
			return m[3] ? new Date(y, mo - 1, +m[3]) : new Date(y, mo, 0); // no day => last day of month
		}
		m = v.match(/^(\d{1,2})[\/-](\d{2}|\d{4})$/);
		if (m) {
			const y = m[2].length === 2 ? 2000 + +m[2] : +m[2];
			return new Date(y, +m[1], 0);
		}
		const d = new Date(v);
		return isNaN(d.getTime()) ? null : d;
	}

	function formatExpiry(exp: string | undefined): string {
		const d = parseExpiry(exp);
		if (!d) return exp?.trim() || '-';
		return `${String(d.getMonth() + 1).padStart(2, '0')}/${String(d.getFullYear()).slice(-2)}`;
	}

	function isShortExpiry(exp: string | undefined): boolean {
		const d = parseExpiry(exp);
		if (!d) return false;
		const ref = invoice?.invoice_date ? new Date(invoice.invoice_date) : new Date();
		return d.getTime() - ref.getTime() <= 90 * 86400000;
	}

	let hasShortExpiry = $derived((invoice?.items || []).some((it) => isShortExpiry(it.expiry_date)));

	let mrpSavings = $derived(
		(invoice?.items || []).reduce((sum, it) => {
			const mrp = parseFloat(it.mrp || '0') || 0;
			const qty = parseFloat(it.quantity || '0') || 0;
			const paid = parseFloat(it.total_amount || '0') || 0;
			return mrp > 0 ? sum + Math.max(0, mrp * qty - paid) : sum;
		}, 0)
	);

	// "Street, City, State - PIN" without stray separators when parts are missing.
	let companyAddressLine = $derived(
		[company?.address, company?.city, [company?.state, company?.pincode].filter(Boolean).join(' - ')]
			.filter((p) => p && String(p).trim())
			.join(', ')
	);

	// Nights between check-in / check-out (hostel & lodging folios). Falls back to the first line's quantity.
	let nightsStayed = $derived.by(() => {
		const a = effectiveSubscriptionStart ? new Date(effectiveSubscriptionStart).getTime() : NaN;
		const b = effectiveSubscriptionEnd ? new Date(effectiveSubscriptionEnd).getTime() : NaN;
		if (!isNaN(a) && !isNaN(b) && b > a) return Math.round((b - a) / 86400000);
		const q = invoice?.items?.[0]?.quantity ? parseFloat(invoice.items[0].quantity) : 0;
		return q > 0 ? Math.round(q) : 0;
	});

	// GST slab summary (taxable value and CGST/SGST/IGST per rate) for hospitality templates.
	let taxSlabs = $derived.by(() => {
		const m = new Map<number, { rate: number; taxable: number; cgst: number; sgst: number; igst: number }>();
		for (const it of invoice?.items || []) {
			const rate = parseFloat(it.tax_rate || '0') || 0;
			const e = m.get(rate) || { rate, taxable: 0, cgst: 0, sgst: 0, igst: 0 };
			e.taxable += parseFloat(it.taxable_value || '0') || 0;
			e.cgst += parseFloat(it.cgst_amount || '0') || 0;
			e.sgst += parseFloat(it.sgst_amount || '0') || 0;
			e.igst += parseFloat(it.igst_amount || '0') || 0;
			m.set(rate, e);
		}
		return [...m.values()].sort((x, y) => x.rate - y.rate);
	});

	function getStateCode(gstin: string | undefined, stateName: string | undefined): string {
		if (gstin && gstin.trim().length >= 2 && !isNaN(parseInt(gstin.substring(0, 2)))) {
			return gstin.trim().substring(0, 2);
		}
		const map: Record<string, string> = {
			'TAMIL NADU': '33', 'TAMILNADU': '33', 'MAHARASHTRA': '27', 'KARNATAKA': '29',
			'KERALA': '32', 'ANDHRA PRADESH': '37', 'TELANGANA': '36', 'DELHI': '07',
			'GUJARAT': '24', 'RAJASTHAN': '08', 'UTTAR PRADESH': '09', 'WEST BENGAL': '19',
			'HARYANA': '06', 'PUNJAB': '03', 'MADHYA PRADESH': '23', 'ODISHA': '21',
			'BIHAR': '10', 'ASSAM': '18', 'JHARKHAND': '20', 'CHHATTISGARH': '22',
			'GOA': '30', 'UTTARAKHAND': '05', 'HIMACHAL PRADESH': '02', 'JAMMU AND KASHMIR': '01'
		};
		const key = (stateName || '').toUpperCase().trim();
		return map[key] || '';
	}

	function getPanNumber(gstin: string | undefined): string {
		if (gstin && gstin.trim().length >= 12) {
			return gstin.trim().substring(2, 12);
		}
		return '';
	}
</script>

{#snippet discRow()}
	{#if invoice && parseFloat(invoice.discount_total || '0') > 0}
		<div class="tpl-adj-row" style="color: #dc2626;"><span>Discount</span><span class="font-mono">- {formatCurrency(invoice.discount_total)}</span></div>
	{/if}
{/snippet}

{#snippet roundRow()}
	{#if invoice && parseFloat(invoice.round_off || '0') !== 0}
		<div class="tpl-adj-row" style="color: #64748b;"><span>Round Off</span><span class="font-mono">{fmtSigned(invoice.round_off)}</span></div>
	{/if}
{/snippet}

{#snippet payRows()}
	{#if invoice && parseFloat(invoice.amount_paid || '0') > 0}
		<div class="tpl-adj-row" style="color: #047857; margin-top: 4px;"><span>Amount Paid</span><span class="font-mono">{formatCurrency(invoice.amount_paid)}</span></div>
		<div class="tpl-adj-row" style="color: #b45309; font-weight: 800;"><span>Balance Due</span><span class="font-mono">{formatCurrency(invoice.balance_due)}</span></div>
	{/if}
{/snippet}

<div class="invoice-sheet-container" class:paper-a5={paperSize === 'A5'} class:paper-a5-landscape={paperSize === 'A5_LANDSCAPE'}>
	<!-- Background Watermark -->
	<div class="invoice-watermark">
		{copyMeta.title.split(' ')[0]}
	</div>

	<!-- ====================================================================
	     TEMPLATE 1: MODERN CORPORATE
	     ==================================================================== -->
	{#if templateId === '1'}
		<div class="tpl-corporate" style="position: relative; z-index: 1;">
			<div class="header">
				<div>
					<div style="margin-bottom: 0.35rem;">
						<span class={`copy-badge ${copyMeta.class}`}>{copyMeta.title}</span>
					</div>
					<h1 class="company-title">{company?.legal_name || company?.name || 'Enterprise Workspace'}</h1>
					{#if effectiveSubtitle}
						<div class="tpl-division-tag" style="color: #6366f1;">{effectiveSubtitle}</div>
					{/if}
					<p style="font-size: 11.5px; color: #64748b; max-width: 440px; margin-top: 4px;">
						{company?.address}, {company?.city}, {company?.state} - {company?.pincode}<br />
						{#if isTaxInvoice}<strong>GSTIN:</strong> {company?.gstin} | {/if}
						<strong>Email:</strong> {company?.email} | <strong>Phone:</strong> {company?.phone}
					</p>
				</div>
				<div style="text-align: right;">
					<div class="doc-type">{isTaxInvoice ? 'TAX INVOICE' : 'BILL OF SUPPLY'}</div>
					<div style="font-family: var(--font-mono); font-size: 14px; font-weight: 700; color: #1e293b; margin-top: 4px;">
						{invoice?.invoice_number || 'INV-PREVIEW'}
					</div>
					<div style="font-size: 11px; color: #64748b; margin-top: 2px;">
						Date: <strong>{formatDate(invoice?.invoice_date)}</strong><br />
						Due Date: <strong>{formatDate(invoice?.due_date)}</strong>
					</div>
				</div>
			</div>

			<!-- Meta Cards -->
			<div class="meta-grid">
				<div class="card-box">
					<div style="font-size: 10px; font-weight: 800; color: #6366f1; text-transform: uppercase; margin-bottom: 4px;">Billed To (Customer):</div>
					<div style="font-weight: 700; font-size: 13px;">{invoice?.customer_name || 'Customer Name'}</div>
					{#if isTaxInvoice && invoice?.customer_gstin}
						<div style="font-family: var(--font-mono); font-size: 11px; color: #4338ca;">GSTIN: {invoice.customer_gstin}</div>
					{/if}
					<div style="font-size: 11.5px; color: #475569; margin-top: 2px; white-space: pre-line;">{invoice?.billing_address}</div>
				</div>
				<div class="card-box">
					<div style="font-size: 10px; font-weight: 800; color: #6366f1; text-transform: uppercase; margin-bottom: 4px;">Supply & Logistics:</div>
					<div style="font-size: 12px;">Place of Supply: <strong>{invoice?.supply_state || company?.state}</strong></div>
					{#if isTaxInvoice}<div style="font-size: 12px; margin-top: 2px;">
						Supply Classification: 
						<strong style="color: {invoice?.is_inter_state ? '#0284c7' : '#059669'};">
							{invoice?.is_inter_state ? 'Inter-State (IGST)' : 'Intra-State (CGST + SGST)'}
						</strong>
					</div>{/if}
					<div style="font-size: 12px; margin-top: 2px;">Terms: <strong>{effectiveTerms}</strong></div>
					<div style="font-size: 12px; margin-top: 2px;">Status: <strong style="color: #047857;">{invoice?.status || 'ISSUED'}</strong></div>
				</div>
			</div>

			<!-- Line Items Table -->
			<table class="tpl-table">
				<thead>
					<tr>
						<th style="width: 5%;">#</th>
						<th style="width: 38%;">Item Description</th>
						{#if isTaxInvoice}<th class="text-center" style="width: 12%;">HSN/SAC</th>{/if}
						<th class="text-right" style="width: 12%;">Qty</th>
						<th class="text-right" style="width: 15%;">Unit Price</th>
						{#if isTaxInvoice}<th class="text-right" style="width: 10%;">GST Rate</th>{/if}
						<th class="text-right" style="width: 18%;">Amount</th>
					</tr>
				</thead>
				<tbody>
					{#if invoice?.items}
						{#each invoice.items as item, i}
							<tr>
								<td class="text-center">{i + 1}</td>
								<td>
									<strong>{item.item_description}</strong>
								</td>
								{#if isTaxInvoice}<td class="text-center font-mono">{item.hsn_sac || '-'}</td>{/if}
								<td class="text-right font-mono">{parseFloat(item.quantity).toFixed(2)} {item.unit_code}</td>
								<td class="text-right font-mono">{formatCurrency(item.unit_price)}</td>
								{#if isTaxInvoice}<td class="text-right font-mono">{parseFloat(item.tax_rate).toFixed(0)}%</td>{/if}
								<td class="text-right font-mono" style="font-weight: 700;">
									{isTaxInvoice ? formatCurrency(item.total_amount) : formatCurrency(item.taxable_value || item.total_amount)}
								</td>
							</tr>
						{/each}
					{/if}
				</tbody>
			</table>

			<!-- Totals Summary & Sign Off -->
			<div style="display: grid; grid-template-columns: 1.2fr 1fr; gap: 20px; margin-top: 14px; align-items: start;">
				<div>
					<div style="font-size: 11px; color: #64748b; margin-bottom: 6px;">
						<strong>Amount in Words:</strong><br />
						<span style="font-weight: 600; color: #1e293b;">{grandTotalWords}</span>
					</div>
					{#if invoice?.notes}
						<div style="font-size: 11px; color: #64748b; background: #f8fafc; padding: 8px 12px; border-radius: 4px; border: 1px solid #e2e8f0; margin-bottom: 8px;">
							<strong>Terms & Instructions:</strong> {invoice.notes}
						</div>
					{/if}

					<!-- Real Configurable Bank Settlement Box -->
					{#if showBank && (effectiveBankName || effectiveAccountNo)}
						<div class="tpl-bank-box" style="margin-bottom: 8px;">
							<div class="tpl-bank-title">🏦 Bank Settlement Coordinates</div>
							<div><strong>Bank:</strong> {effectiveBankName} {#if effectiveBranch}({effectiveBranch}){/if}</div>
							<div><strong>Account Number:</strong> <span class="font-mono" style="font-weight: 700;">{effectiveAccountNo}</span></div>
							<div><strong>IFSC Code:</strong> <span class="font-mono" style="font-weight: 700;">{effectiveIFSC}</span></div>
							{#if effectiveAccountHolder}<div><strong>Beneficiary:</strong> {effectiveAccountHolder}</div>{/if}
						</div>
					{/if}

					<!-- Real Configurable UPI QR Box -->
					{#if showUPI && effectiveUPI}
						<div style="display: flex; gap: 12px; align-items: center; background: #f8fafc; padding: 8px 12px; border-radius: 6px; border: 1px solid #e2e8f0;">
							<div class="tpl-upi-qr-box">
								<img class="tpl-upi-qr-img" src={getUPIQRUrl(effectiveUPI, company?.legal_name || company?.name || '', invoice?.grand_total)} alt="UPI QR" />
								<span class="tpl-upi-text">Scan & Pay</span>
							</div>
							<div style="font-size: 11px; color: #334155;">
								<div style="font-weight: 700; color: #4338ca; margin-bottom: 2px;">⚡ Instant UPI Payment</div>
								<div>UPI ID: <span class="font-mono" style="font-weight: 700; color: #1e293b;">{effectiveUPI}</span></div>
								<div style="font-size: 9.5px; color: #64748b; margin-top: 2px;">Google Pay • PhonePe • Paytm • BHIM</div>
							</div>
						</div>
					{/if}
				</div>
				<div style="background: #f8fafc; padding: 12px 16px; border-radius: 6px; border: 1px solid #e2e8f0;">
					<div style="display: flex; justify-content: space-between; margin-bottom: 4px;">
						<span>Subtotal:</span>
						<span class="font-mono">{formatCurrency(invoice?.subtotal)}</span>
					</div>
					{#if invoice && parseFloat(invoice.discount_total) > 0}
						<div style="display: flex; justify-content: space-between; margin-bottom: 4px; color: #dc2626;">
							<span>Discount:</span>
							<span class="font-mono">- {formatCurrency(invoice.discount_total)}</span>
						</div>
					{/if}
					{#if isTaxInvoice}
						<div style="display: flex; justify-content: space-between; margin-bottom: 4px;">
							<span>Taxable Amount:</span>
							<span class="font-mono">{formatCurrency(invoice?.taxable_amount)}</span>
						</div>
						{#if invoice?.is_inter_state}
							<div style="display: flex; justify-content: space-between; margin-bottom: 4px; color: #0284c7;">
								<span>Integrated GST (IGST):</span>
								<span class="font-mono">{formatCurrency(invoice?.igst_total)}</span>
							</div>
						{:else}
							<div style="display: flex; justify-content: space-between; margin-bottom: 4px; color: #059669;">
								<span>Central GST (CGST):</span>
								<span class="font-mono">{formatCurrency(invoice?.cgst_total)}</span>
							</div>
							<div style="display: flex; justify-content: space-between; margin-bottom: 4px; color: #059669;">
								<span>State GST (SGST):</span>
								<span class="font-mono">{formatCurrency(invoice?.sgst_total)}</span>
							</div>
						{/if}
					{/if}
					{#if invoice && parseFloat(invoice.round_off) !== 0}
						<div style="display: flex; justify-content: space-between; margin-bottom: 4px; color: #64748b; font-size: 11px;">
							<span>Round Off:</span>
							<span class="font-mono">{invoice.round_off}</span>
						</div>
					{/if}
					<div style="display: flex; justify-content: space-between; font-size: 16px; font-weight: 800; color: #312e81; border-top: 2px solid #cbd5e1; padding-top: 6px; margin-top: 6px;">
						<span>Grand Total:</span>
						<span class="font-mono">{formatCurrency(invoice?.grand_total)}</span>
					</div>
					<div style="display: flex; justify-content: space-between; font-size: 12px; margin-top: 4px; color: #047857; font-weight: 600;">
						<span>Amount Paid:</span>
						<span class="font-mono">{formatCurrency(invoice?.amount_paid)}</span>
					</div>
					<div style="display: flex; justify-content: space-between; font-size: 13px; margin-top: 2px; color: #b45309; font-weight: 700;">
						<span>Balance Due:</span>
						<span class="font-mono">{formatCurrency(invoice?.balance_due)}</span>
					</div>
				</div>
			</div>

			<!-- Footer Signatory -->
			<div style="display: flex; justify-content: space-between; align-items: flex-end; margin-top: 30px; padding-top: 14px; border-top: 1px solid #e2e8f0; font-size: 10.5px; color: #64748b;">
				<div>
					<div>* Subject to {effectiveJurisdiction} Jurisdiction.</div>
					<div>This is an authenticated computer-generated financial document.</div>
				</div>
				<div class="tpl-signature-box">
					<div style="font-weight: 700; color: #1e293b; margin-bottom: 4px;">For {company?.legal_name || company?.name}</div>
					{#if effectiveSignatureUrl}
						<div><img src={effectiveSignatureUrl} alt="Signature" class="tpl-signature-img" /></div>
					{:else}
						<div style="height: 32px;"></div>
					{/if}
					<div style="font-weight: 600;">{effectiveSignatoryTitle}</div>
				</div>
			</div>
		</div>

	<!-- ====================================================================
	     TEMPLATE 2: CLASSIC TRADITIONAL (INDIAN WHOLESALE BOXED)
	     ==================================================================== -->
	{:else if templateId === '2'}
		<div class="tpl-classic" style="position: relative; z-index: 1;">
			<div class="border-box">
				<div style="display: flex; justify-content: space-between; padding: 6px 12px; background: #000000; color: #ffffff; font-size: 11px; font-weight: 700;">
					<span>GST RULE 46 TAX INVOICE</span>
					<span>{copyMeta.title}</span>
				</div>

				<div class="header-banner">
					<h2>{company?.legal_name || company?.name}</h2>
					{#if effectiveSubtitle}
						<div style="font-size: 10px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.05em; margin-top: 2px;">{effectiveSubtitle}</div>
					{/if}
					<div style="font-size: 11px; margin-top: 2px;">
						{company?.address}, {company?.city}, {company?.state} - {company?.pincode}
					</div>
					<div style="font-size: 11px; font-weight: 700; margin-top: 2px;">
						{#if isTaxInvoice}GSTIN: {company?.gstin} | {/if}Email: {company?.email} | Tel: {company?.phone}
					</div>
				</div>

				<div class="doc-title-bar">
					{isTaxInvoice ? 'TAX INVOICE' : 'BILL OF SUPPLY (NON-GST)'}
				</div>

				<div class="split-row">
					<div class="split-cell">
						<div style="font-size: 10px; font-weight: 800; text-transform: uppercase;">Details of Receiver / Billed To:</div>
						<div style="font-size: 13px; font-weight: 800; margin-top: 2px;">{invoice?.customer_name}</div>
						{#if isTaxInvoice && invoice?.customer_gstin}
							<div style="font-family: var(--font-mono); font-size: 11px; font-weight: 700;">GSTIN / UIN: {invoice.customer_gstin}</div>
						{/if}
						<div style="font-size: 11px; margin-top: 2px; white-space: pre-line;">{invoice?.billing_address}</div>
						<div style="font-size: 11px; font-weight: 700; margin-top: 4px;">State & Code: {invoice?.supply_state}</div>
					</div>
					<div class="split-cell" style="font-size: 11.5px;">
						<div style="display: flex; justify-content: space-between; margin-bottom: 3px;">
							<span>Invoice Number:</span> <strong style="font-family: var(--font-mono);">{invoice?.invoice_number}</strong>
						</div>
						<div style="display: flex; justify-content: space-between; margin-bottom: 3px;">
							<span>Invoice Date:</span> <strong>{formatDate(invoice?.invoice_date)}</strong>
						</div>
						<div style="display: flex; justify-content: space-between; margin-bottom: 3px;">
							<span>Payment Due Date:</span> <strong>{formatDate(invoice?.due_date)}</strong>
						</div>
						<div style="display: flex; justify-content: space-between; margin-bottom: 3px;">
							<span>Place of Supply:</span> <strong>{invoice?.supply_state}</strong>
						</div>
						<div style="display: flex; justify-content: space-between; margin-bottom: 3px;">
							<span>Payment Terms:</span> <strong>{effectiveTerms}</strong>
						</div>
						<div style="display: flex; justify-content: space-between;">
							<span>Reverse Charge (Y/N):</span> <strong>{effectiveReverseCharge ? 'YES' : 'NO'}</strong>
						</div>
					</div>
				</div>

				<table class="tpl-table" style="margin: 0; border: none;">
					<thead>
						<tr>
							<th style="width: 5%;">S.No</th>
							<th style="width: 40%;">Description of Goods / Services</th>
							{#if isTaxInvoice}<th class="text-center" style="width: 12%;">HSN/SAC</th>{/if}
							<th class="text-right" style="width: 10%;">Qty</th>
							<th class="text-right" style="width: 13%;">Rate (₹)</th>
							{#if isTaxInvoice}<th class="text-right" style="width: 8%;">GST %</th>{/if}
							<th class="text-right" style="width: 16%;">Amount (₹)</th>
						</tr>
					</thead>
					<tbody>
						{#if invoice?.items}
							{#each invoice.items as it, i}
								<tr>
									<td class="text-center">{i + 1}</td>
									<td><strong>{it.item_description}</strong></td>
									{#if isTaxInvoice}<td class="text-center font-mono">{it.hsn_sac || '-'}</td>{/if}
									<td class="text-right font-mono">{parseFloat(it.quantity).toFixed(2)} {it.unit_code}</td>
									<td class="text-right font-mono">{fmtNum(it.unit_price)}</td>
									{#if isTaxInvoice}<td class="text-right font-mono">{parseFloat(it.tax_rate).toFixed(0)}%</td>{/if}
									<td class="text-right font-mono">
										{isTaxInvoice ? parseFloat(it.total_amount || '0').toFixed(2) : parseFloat(it.taxable_value || it.total_amount || '0').toFixed(2)}
									</td>
								</tr>
							{/each}
						{/if}
					</tbody>
				</table>

				<!-- Classic Totals -->
				<div style="display: grid; grid-template-columns: 1.5fr 1fr; border-top: 2px solid #000000;">
					<div style="padding: 10px 14px; border-right: 1px solid #000000;">
						<div style="font-size: 11px; margin-bottom: 6px;">
							<strong>Total Amount (in words):</strong><br />
							<span style="font-weight: 700;">{grandTotalWords}</span>
						</div>
						<div style="font-size: 10px; color: #475569; margin-top: 6px;">
							<strong>Declaration:</strong> We declare that this invoice shows actual price of the goods/services described and that all particulars are true and correct.
						</div>
						{#if showBank && (effectiveBankName || effectiveAccountNo)}
							<div style="font-size: 10px; color: #1e293b; margin-top: 8px; border-top: 1px dashed #64748b; padding-top: 5px;">
								<strong>Bank Settlement:</strong> {effectiveBankName} • A/C: <span class="font-mono" style="font-weight: 700;">{effectiveAccountNo}</span> • IFSC: <span class="font-mono" style="font-weight: 700;">{effectiveIFSC}</span>
								{#if effectiveUPI} • UPI: <span class="font-mono" style="font-weight: 700;">{effectiveUPI}</span>{/if}
							</div>
						{/if}
					</div>
					<div style="padding: 8px 12px; font-size: 11.5px;">
						<div style="display: flex; justify-content: space-between; margin-bottom: 3px;">
							<span>Subtotal:</span>
							<span class="font-mono">{formatCurrency(invoice?.subtotal)}</span>
						</div>
						{@render discRow()}
						{#if isTaxInvoice}
							{#if invoice?.is_inter_state}
								<div style="display: flex; justify-content: space-between; margin-bottom: 3px;">
									<span>IGST:</span>
									<span class="font-mono">{formatCurrency(invoice?.igst_total)}</span>
								</div>
							{:else}
								<div style="display: flex; justify-content: space-between; margin-bottom: 3px;">
									<span>CGST:</span>
									<span class="font-mono">{formatCurrency(invoice?.cgst_total)}</span>
								</div>
								<div style="display: flex; justify-content: space-between; margin-bottom: 3px;">
									<span>SGST:</span>
									<span class="font-mono">{formatCurrency(invoice?.sgst_total)}</span>
								</div>
							{/if}
						{/if}
						{@render roundRow()}
						<div style="display: flex; justify-content: space-between; font-size: 14px; font-weight: 900; border-top: 1px solid #000000; padding-top: 4px; margin-top: 4px;">
							<span>Total (Rounded):</span>
							<span class="font-mono">{formatCurrency(invoice?.grand_total)}</span>
						</div>
						{@render payRows()}
					</div>
				</div>

				<div class="sign-box">
					<div style="padding: 10px 14px; font-size: 10.5px; border-right: 1px solid #000000;">
						<strong>Terms & Conditions:</strong> * Subject to {effectiveJurisdiction} Jurisdiction. Interest @ {effectiveInterestRate} will be charged if bill is not settled on due date. {effectiveReturnPolicy}
					</div>
					<div style="padding: 10px 14px; text-align: center; font-size: 11px;">
						<div style="font-weight: 700;">For {company?.legal_name || company?.name}</div>
						{#if effectiveSignatureUrl}
							<div style="margin: 3px 0;"><img src={effectiveSignatureUrl} alt="Sign" class="tpl-signature-img" /></div>
						{:else}
							<div style="height: 35px;"></div>
						{/if}
						<div>{effectiveSignatoryTitle}</div>
					</div>
				</div>
			</div>
		</div>

	<!-- ====================================================================
	     TEMPLATE 3: MINIMALIST EXECUTIVE
	     ==================================================================== -->
	{:else if templateId === '3'}
		<div class="tpl-minimal" style="position: relative; z-index: 1;">
			<div style="margin-bottom: 1rem;">
				<span class={`copy-badge ${copyMeta.class}`}>{copyMeta.title}</span>
			</div>
			<div class="header">
				<div>
					<div class="company-name">{company?.legal_name || company?.name}</div>
					{#if effectiveSubtitle}
						<div style="font-size: 11px; color: #4f46e5; font-weight: 600; margin-top: 2px;">{effectiveSubtitle}</div>
					{/if}
					<div style="font-size: 11px; color: #64748b; margin-top: 4px;">
						{company?.city}, {company?.state} • {#if isTaxInvoice}GSTIN: {company?.gstin} • {/if}{company?.email}
					</div>
				</div>
				<div style="text-align: right;">
					<div class="doc-title">{isTaxInvoice ? 'INVOICE' : 'BILL OF SUPPLY'}</div>
					<div style="font-family: var(--font-mono); font-size: 15px; font-weight: 600; color: #0f172a;">{invoice?.invoice_number}</div>
					<div style="font-size: 11px; color: #94a3b8;">Issued: {formatDate(invoice?.invoice_date)}</div>
					<div style="font-size: 11px; color: #64748b; margin-top: 2px;">Terms: <strong>{effectiveTerms}</strong></div>
				</div>
			</div>

			<div style="margin-bottom: 24px; font-size: 12px; line-height: 1.6;">
				<span style="color: #94a3b8; text-transform: uppercase; font-size: 10px; font-weight: 700; letter-spacing: 0.05em; display: block; margin-bottom: 2px;">Invoiced To</span>
				<strong style="font-size: 14px; color: #0f172a;">{invoice?.customer_name}</strong><br />
				{#if isTaxInvoice && invoice?.customer_gstin}
					<span style="font-family: var(--font-mono); color: #475569;">GSTIN: {invoice.customer_gstin}</span><br />
				{/if}
				<span style="color: #475569;">{invoice?.billing_address}</span>
				{#if effectiveProjectName}
					<div style="margin-top: 4px; font-size: 11.5px; color: #4338ca;"><strong>Engagement / Project:</strong> {effectiveProjectName}</div>
				{/if}
			</div>

			<table class="tpl-table">
				<thead>
					<tr>
						<th style="width: 50%;">Description</th>
						<th class="text-right" style="width: 15%;">Qty</th>
						<th class="text-right" style="width: 15%;">Rate</th>
						<th class="text-right" style="width: 20%;">Total</th>
					</tr>
				</thead>
				<tbody>
					{#if invoice?.items}
						{#each invoice.items as it}
							<tr>
								<td>
									<div style="font-weight: 600; color: #0f172a;">{it.item_description}</div>
									{#if isTaxInvoice && it.hsn_sac}
										<div style="font-size: 10px; color: #64748b; font-family: var(--font-mono);">HSN: {it.hsn_sac} • Tax: {parseFloat(it.tax_rate).toFixed(0)}%</div>
									{/if}
								</td>
								<td class="text-right font-mono">{parseFloat(it.quantity).toFixed(2)}</td>
								<td class="text-right font-mono">{formatCurrency(it.unit_price)}</td>
								<td class="text-right font-mono" style="font-weight: 600; color: #0f172a;">{formatCurrency(it.total_amount)}</td>
							</tr>
						{/each}
					{/if}
				</tbody>
			</table>

			<div style="display: flex; justify-content: space-between; margin-top: 20px; align-items: flex-start;">
				<div style="max-width: 380px; font-size: 11px; color: #475569;">
					{#if showBank && (effectiveBankName || effectiveAccountNo)}
						<div class="tpl-bank-box" style="margin-bottom: 8px;">
							<div class="tpl-bank-title">Wire / Direct Remittance Coordinates</div>
							<div><strong>Bank:</strong> {effectiveBankName} {#if effectiveBranch}({effectiveBranch}){/if}</div>
							<div><strong>Account:</strong> <span class="font-mono" style="font-weight: 700;">{effectiveAccountNo}</span></div>
							<div><strong>IFSC:</strong> <span class="font-mono" style="font-weight: 700;">{effectiveIFSC}</span></div>
						</div>
					{/if}
					<div>* Subject to {effectiveJurisdiction} Jurisdiction. {effectiveReturnPolicy}</div>
				</div>
				<div style="width: 280px; font-size: 12px; display: flex; flex-direction: column; gap: 6px;">
					<div style="display: flex; justify-content: space-between; color: #64748b;">
						<span>Subtotal</span>
						<span class="font-mono">{formatCurrency(invoice?.subtotal)}</span>
					</div>
					{@render discRow()}
					{#if isTaxInvoice}
						<div style="display: flex; justify-content: space-between; color: #64748b;">
							<span>Taxes (GST)</span>
							<span class="font-mono">
								{formatCurrency(
									(parseFloat(invoice?.cgst_total || '0') + parseFloat(invoice?.sgst_total || '0') + parseFloat(invoice?.igst_total || '0')).toString()
								)}
							</span>
						</div>
					{/if}
					{@render roundRow()}
					<div style="display: flex; justify-content: space-between; font-size: 16px; font-weight: 700; color: #0f172a; border-top: 1px solid #0f172a; padding-top: 8px; margin-top: 4px;">
						<span>Total</span>
						<span class="font-mono">{formatCurrency(invoice?.grand_total)}</span>
					</div>
					{@render payRows()}
					<div style="text-align: right; margin-top: 16px; font-size: 11px;">
						<div style="color: #64748b; margin-bottom: 4px;">Authorized Signatory</div>
						{#if effectiveSignatureUrl}
							<div><img src={effectiveSignatureUrl} alt="Sign" class="tpl-signature-img" /></div>
						{/if}
						<div style="font-weight: 700; color: #0f172a;">{effectiveSignatoryTitle}</div>
					</div>
				</div>
			</div>
		</div>

	<!-- ====================================================================
	     TEMPLATE 4: RETAIL COUNTER POS
	     ==================================================================== -->
	{:else if templateId === '4'}
		<div class="tpl-retail" style="position: relative; z-index: 1;">
			<div class="store-header">
				<div>
					<div class="store-name">{company?.legal_name || company?.name}</div>
					{#if effectiveSubtitle}
						<div style="font-size: 11px; font-weight: 600; opacity: 0.95;">{effectiveSubtitle}</div>
					{/if}
					<div style="font-size: 11px; opacity: 0.9;">{company?.address}, {company?.city} • Ph: {company?.phone}</div>
				</div>
				<div style="text-align: right;">
					<span class={`copy-badge ${copyMeta.class}`}>{copyMeta.title}</span>
					<div style="font-size: 16px; font-weight: 800; margin-top: 4px;">{isTaxInvoice ? 'TAX INVOICE' : 'RETAIL CASH MEMO'}</div>
				</div>
			</div>

			<div class="bill-meta">
				<div>Bill No: <strong>{invoice?.invoice_number}</strong></div>
				<div>Date: <strong>{formatDate(invoice?.invoice_date)}</strong></div>
				<div>Customer: <strong>{invoice?.customer_name}</strong></div>
				{#if isTaxInvoice}<div>GSTIN: <strong>{company?.gstin}</strong></div>{/if}
				<div>Cashier: <strong>{effectiveCashierName}</strong></div>
				<div>Counter: <strong>{effectiveCounterNumber}</strong></div>
				<div>Pay Mode: <strong>{effectiveTerms}</strong></div>
			</div>

			<table class="tpl-table">
				<thead>
					<tr>
						<th style="width: 6%;">#</th>
						<th style="width: 44%;">Item</th>
						<th class="text-right" style="width: 12%;">Qty</th>
						<th class="text-right" style="width: 16%;">Rate</th>
						<th class="text-right" style="width: 22%;">Amount</th>
					</tr>
				</thead>
				<tbody>
					{#if invoice?.items}
						{#each invoice.items as it, i}
							<tr>
								<td>{i + 1}</td>
								<td>
									<strong>{it.item_description}</strong>
									{#if isTaxInvoice && it.hsn_sac}<span style="font-size: 10px; color: #64748b;"> (HSN: {it.hsn_sac})</span>{/if}
								</td>
								<td class="text-right font-mono">{parseFloat(it.quantity).toFixed(0)} {it.unit_code}</td>
								<td class="text-right font-mono">{formatCurrency(it.unit_price)}</td>
								<td class="text-right font-mono" style="font-weight: 700;">{formatCurrency(it.total_amount)}</td>
							</tr>
						{/each}
					{/if}
				</tbody>
			</table>

			<div style="display: flex; justify-content: space-between; align-items: center; border-top: 2px solid #047857; padding-top: 12px; margin-top: 12px; flex-wrap: wrap; gap: 12px;">
				<div>
					<span class="savings-pill">★ Thank you for shopping with us!</span>
					<div style="font-size: 10px; color: #64748b; margin-top: 6px;">
						* Policy: {effectiveReturnPolicy}
					</div>
				</div>
				<div style="display: flex; align-items: center; gap: 14px;">
					{#if showUPI && effectiveUPI}
						<div class="tpl-upi-qr-box" style="padding: 4px 6px;">
							<img class="tpl-upi-qr-img" style="width: 65px; height: 65px;" src={getUPIQRUrl(effectiveUPI, company?.name || '', invoice?.grand_total)} alt="UPI QR" />
							<span class="tpl-upi-text" style="font-size: 8.5px;">Scan UPI</span>
						</div>
					{/if}
					<div style="text-align: right;">
						<div style="font-size: 12px; color: #64748b;">Total Payable:</div>
						<div style="font-size: 22px; font-weight: 800; color: #065f46; font-family: var(--font-mono);">
							{formatCurrency(invoice?.grand_total)}
						</div>
						<div style="min-width: 190px;">{@render payRows()}</div>
					</div>
				</div>
			</div>
		</div>

	<!-- ====================================================================
	     TEMPLATE 5: THERMAL RECEIPT (80mm ROLL PAPER)
	     ==================================================================== -->
	{:else if templateId === '5'}
		<div class="tpl-thermal" style="position: relative; z-index: 1;">
			<div class="center-text">
				{#if company?.template_studio_config?.show_logo && company?.logo_url}
					<div style="margin-bottom: 4px;">
						<img src={company.logo_url} alt="Logo" style="max-height: {company.template_studio_config.logo_height || 40}px; display: inline-block;" />
					</div>
				{/if}
				<div style="font-size: 14px; font-weight: 900; text-transform: uppercase;">{company?.legal_name || company?.name}</div>
				{#if effectiveSubtitle}
					<div style="font-size: 10px; font-weight: 700; margin-top: 1px;">{effectiveSubtitle}</div>
				{/if}
				<div>{company?.address}, {company?.city}</div>
				<div>Tel: {company?.phone}</div>
				{#if company?.drug_license_no}
					<div style="font-size: 9.5px; font-weight: 700;">D.L. No: {company.drug_license_no}</div>
				{/if}
				{#if company?.fssai_no}
					<div style="font-size: 9.5px;">FSSAI Lic: {company.fssai_no}</div>
				{/if}
				{#if isTaxInvoice}<div>GSTIN: {company?.gstin}</div>{/if}
				<hr class="thermal-divider" />
				<div style="font-weight: bold; font-size: 12px;">{isTaxInvoice ? 'TAX INVOICE' : 'CASH RECEIPT'}</div>
				<div style="font-size: 10px;">({copyMeta.title})</div>
			</div>

			<hr class="thermal-divider" />
			<div style="display: flex; justify-content: space-between;">
				<span>Rcpt: {invoice?.invoice_number}</span>
				<span>{formatDate(invoice?.invoice_date)}</span>
			</div>
			<div>Cust: {invoice?.customer_name}</div>
			{#if invoice?.doctor_name || invoice?.patient_name}
				<div style="font-size: 9.5px; background: #f3f4f6; padding: 2px 4px; margin: 2px 0;">
					{#if invoice?.patient_name}<div>Pt: <strong>{invoice.patient_name}</strong> {#if invoice?.patient_age}(Age: {invoice.patient_age}){/if}</div>{/if}
					{#if invoice?.doctor_name}<div>Dr: <strong>{invoice.doctor_name}</strong> {#if invoice?.doctor_reg_no}(Reg: {invoice.doctor_reg_no}){/if}</div>{/if}
				</div>
			{/if}
			<div style="display: flex; justify-content: space-between; font-size: 10px;">
				<span>Cashier: {effectiveCashierName}</span>
				<span>{effectiveCounterNumber}</span>
			</div>
			<div style="font-size: 10px;">Mode: {effectiveTerms}</div>
			<hr class="thermal-divider" />

			<table class="thermal-table">
				<thead>
					<tr>
						<th style="width: 50%;">ITEM</th>
						<th style="width: 20%; text-align: right;">QTY</th>
						<th style="width: 30%; text-align: right;">TOTAL</th>
					</tr>
				</thead>
				<tbody>
					{#if invoice?.items}
						{#each invoice.items as it}
							<tr>
								<td>
									<div>{it.item_description.slice(0, 20)}</div>
									{#if it.batch_number}
										<div style="font-size: 8.5px; color: #4b5563;">B:{it.batch_number} Exp:{it.expiry_date}</div>
									{/if}
									{#if it.rack_location}
										<div style="font-size: 8px; color: #2563eb;">📍 {it.rack_location}</div>
									{/if}
								</td>
								<td style="text-align: right;">{parseFloat(it.quantity).toFixed(0)}</td>
								<td style="text-align: right; font-weight: bold;">{formatCurrency(it.total_amount)}</td>
							</tr>
						{/each}
					{/if}
				</tbody>
			</table>

			<hr class="thermal-divider-double" />
			<div style="display: flex; justify-content: space-between; font-weight: bold; font-size: 13px;">
				<span>NET TOTAL:</span>
				<span>{formatCurrency(invoice?.grand_total)}</span>
			</div>
			{#if invoice?.discount_total && parseFloat(invoice.discount_total) > 0}
				<div style="text-align: center; font-size: 9.5px; font-weight: 800; margin: 3px 0; border: 1px dashed #000; padding: 2px;">
					*** YOU SAVED {formatCurrency(invoice.discount_total)} ON MRP ***
				</div>
			{/if}
			<div style="display: flex; justify-content: space-between; font-size: 10px; margin-top: 2px;">
				<span>Paid: {formatCurrency(invoice?.amount_paid)}</span>
				<span>Due: {formatCurrency(invoice?.balance_due)}</span>
			</div>
			{#if invoice?.change_amount && parseFloat(invoice.change_amount) > 0}
				<div style="display: flex; justify-content: space-between; font-size: 9.5px; color: #16a34a; font-weight: 700;">
					<span>Change Returned:</span>
					<span>{formatCurrency(invoice.change_amount)}</span>
				</div>
			{/if}

			{#if showUPI && effectiveUPI}
				<div style="text-align: center; margin: 6px auto;">
					<img src={getUPIQRUrl(effectiveUPI, company?.name || '', invoice?.grand_total)} alt="UPI QR" style="width: {company?.template_studio_config?.upi_qr_size || 75}px; height: {company?.template_studio_config?.upi_qr_size || 75}px; display: inline-block;" />
					<div style="font-size: 9px; font-weight: 700;">UPI: {effectiveUPI}</div>
				</div>
			{/if}

			<hr class="thermal-divider" />
			<div class="center-text" style="font-size: 10px;">
				<div>{company?.template_studio_config?.custom_footer_message || company?.template_options?.thermal_footer || extraDetails?.thermalFooter || '*** THANK YOU VISIT AGAIN ***'}</div>
				{#if effectiveReturnPolicy}
					<div style="font-size: 8.5px; margin-top: 2px;">* {effectiveReturnPolicy}</div>
				{/if}
				<div style="margin-top: 4px; letter-spacing: 0.2em;">|||| | |||||| ||| |||||||</div>
			</div>
		</div>

	<!-- ====================================================================
	     TEMPLATE 6: CREATIVE STUDIO & AGENCY
	     ==================================================================== -->
	{:else if templateId === '6'}
		<div class="tpl-creative" style="position: relative; z-index: 1;">
			<div class="header">
				<div style="display: flex; justify-content: space-between; align-items: center;">
					<h1 class="company-title">{company?.legal_name || company?.name}</h1>
					<span class="doc-badge">{copyMeta.title}</span>
				</div>
				<p style="font-size: 12px; color: #6b7280; margin-top: 4px;">
					{#if effectiveSubtitle}
						<span style="font-weight: 600; color: #7c3aed;">{effectiveSubtitle}</span> •
					{:else}
						Digital Services, Design & Engineering •
					{/if}
					{company?.email} {#if isTaxInvoice && company?.gstin}• GSTIN: {company.gstin}{/if}
				</p>
			</div>

			<div style="display: flex; justify-content: space-between; margin-bottom: 24px; flex-wrap: wrap; gap: 12px;">
				<div>
					<div style="font-size: 11px; font-weight: 700; color: #7c3aed; text-transform: uppercase;">Client Information:</div>
					<div style="font-size: 14px; font-weight: 700; color: #111827;">{invoice?.customer_name}</div>
					<div style="font-size: 12px; color: #4b5563;">{invoice?.billing_address}</div>
					{#if isTaxInvoice && invoice?.customer_gstin}
						<div style="font-size: 11px; color: #6b7280; font-family: var(--font-mono); margin-top: 2px;">GSTIN: {invoice.customer_gstin}</div>
					{/if}
					{#if effectiveProjectName}
						<div style="margin-top: 4px; font-size: 12px; color: #6d28d9;">
							<strong>Engagement / Project:</strong> {effectiveProjectName}
						</div>
					{/if}
				</div>
				<div style="text-align: right; font-size: 12px;">
					<div style="font-size: 16px; font-weight: 800; color: #7c3aed;">{invoice?.invoice_number}</div>
					<div>Project Invoice • {formatDate(invoice?.invoice_date)}</div>
					<div style="font-size: 11px; color: #6b7280; margin-top: 2px;">Terms: <strong>{effectiveTerms}</strong></div>
				</div>
			</div>

			<table class="tpl-table">
				<thead>
					<tr>
						<th style="width: 50%;">Deliverable / Scope</th>
						<th class="text-right" style="width: 15%;">Units</th>
						<th class="text-right" style="width: 15%;">Rate</th>
						<th class="text-right" style="width: 20%;">Total</th>
					</tr>
				</thead>
				<tbody>
					{#if invoice?.items}
						{#each invoice.items as it}
							<tr>
								<td>
									<strong>{it.item_description}</strong>
									{#if isTaxInvoice && it.hsn_sac}
										<div style="font-size: 10px; color: #6b7280; font-family: var(--font-mono);">SAC: {it.hsn_sac}</div>
									{/if}
								</td>
								<td class="text-right font-mono">{parseFloat(it.quantity).toFixed(2)} {it.unit_code}</td>
								<td class="text-right font-mono">{formatCurrency(it.unit_price)}</td>
								<td class="text-right font-mono" style="font-weight: 700; color: #5b21b6;">{formatCurrency(it.total_amount)}</td>
							</tr>
						{/each}
					{/if}
				</tbody>
			</table>

			<div style="display: flex; justify-content: space-between; margin-top: 18px; align-items: flex-start; flex-wrap: wrap; gap: 14px;">
				<div style="max-width: 360px; font-size: 11px;">
					{#if showBank && (effectiveBankName || effectiveAccountNo)}
						<div class="tpl-bank-box" style="margin-bottom: 8px;">
							<div class="tpl-bank-title">Payment Coordinates</div>
							<div><strong>Bank:</strong> {effectiveBankName} {#if effectiveBranch}({effectiveBranch}){/if}</div>
							<div><strong>A/C No:</strong> <span class="font-mono" style="font-weight: 700;">{effectiveAccountNo}</span></div>
							<div><strong>IFSC:</strong> <span class="font-mono" style="font-weight: 700;">{effectiveIFSC}</span></div>
							{#if effectiveAccountHolder}<div><strong>Beneficiary:</strong> {effectiveAccountHolder}</div>{/if}
						</div>
					{/if}
					<div style="color: #6b7280;">
						* Payment is due as per <strong>{effectiveTerms}</strong> terms. Subject to {effectiveJurisdiction} Jurisdiction.
					</div>
				</div>

				<div style="display: flex; align-items: flex-start; gap: 14px;">
					{#if showUPI && effectiveUPI}
						<div class="tpl-upi-qr-box" style="padding: 6px 10px;">
							<img class="tpl-upi-qr-img" src={getUPIQRUrl(effectiveUPI, company?.name || '', invoice?.grand_total)} alt="UPI QR" />
							<span class="tpl-upi-text">Scan to Pay</span>
						</div>
					{/if}
					<div style="width: 250px; font-size: 12px; background: #faf5ff; padding: 14px; border-radius: 6px; border: 1px solid #e9d5ff;">
						<div style="display: flex; justify-content: space-between; margin-bottom: 4px;">
							<span>Fee:</span>
							<span class="font-mono">{formatCurrency(invoice?.subtotal)}</span>
						</div>
						{@render discRow()}
						{#if isTaxInvoice}
							<div style="display: flex; justify-content: space-between; margin-bottom: 4px; color: #7c3aed;">
								<span>GST:</span>
								<span class="font-mono">
									{formatCurrency(
										(parseFloat(invoice?.cgst_total || '0') + parseFloat(invoice?.sgst_total || '0') + parseFloat(invoice?.igst_total || '0')).toString()
									)}
								</span>
							</div>
						{/if}
						{@render roundRow()}
						<div style="display: flex; justify-content: space-between; font-size: 16px; font-weight: 800; color: #4c1d95; border-top: 1px solid #d8b4fe; padding-top: 6px; margin-top: 6px;">
							<span>Grand Total:</span>
							<span class="font-mono">{formatCurrency(invoice?.grand_total)}</span>
						</div>
						{@render payRows()}
						<div style="text-align: right; margin-top: 14px; font-size: 11px;">
							{#if effectiveSignatureUrl}
								<div><img src={effectiveSignatureUrl} alt="Sign" class="tpl-signature-img" /></div>
							{/if}
							<div style="font-weight: 700; color: #4c1d95;">{effectiveSignatoryTitle}</div>
						</div>
					</div>
				</div>
			</div>
		</div>

	<!-- ====================================================================
	     TEMPLATE 7: INDUSTRIAL / LOGISTICS / B2B DISPATCH
	     ==================================================================== -->
	{:else if templateId === '7'}
		<div class="tpl-industrial" style="position: relative; z-index: 1;">
			<div class="industrial-header">
				<div>
					<div style="font-size: 18px; font-weight: 900;">{company?.legal_name || company?.name}</div>
					<div style="font-size: 11px; opacity: 0.9;">{effectiveSubtitle || 'MANUFACTURING & INDUSTRIAL DISPATCH DIVISION'}</div>
					{#if company?.gstin}<div style="font-size: 10px; opacity: 0.85;">GSTIN: {company.gstin}</div>{/if}
				</div>
				<div style="text-align: right;">
					<span class="copy-badge copy-badge-triplicate">{copyMeta.title}</span>
					<div style="font-size: 15px; font-weight: 800; margin-top: 4px;">{isTaxInvoice ? 'TAX INVOICE' : 'DELIVERY CHALLAN / BILL'}</div>
				</div>
			</div>

			<!-- Dispatch & Logistics Grid -->
			<div class="dispatch-grid">
				<div class="dispatch-cell">
					<strong>Invoice #:</strong> {invoice?.invoice_number}<br />
					<strong>Date:</strong> {formatDate(invoice?.invoice_date)}<br />
					<strong>Terms:</strong> {effectiveTerms}
				</div>
				<div class="dispatch-cell">
					<strong>PO Number:</strong> {extraDetails.poNumber || invoice?.po_number || 'N/A'}<br />
					<strong>Vehicle #:</strong> {extraDetails.vehicleNumber || invoice?.vehicle_number || '-'}
				</div>
				<div class="dispatch-cell">
					<strong>E-Way Bill #:</strong> {extraDetails.ewayBill || invoice?.eway_bill || '-'}<br />
					<strong>LR / RR #:</strong> {extraDetails.lrNumber || invoice?.lr_number || '-'}
				</div>
				<div class="dispatch-cell">
					<strong>Transporter:</strong> {extraDetails.transporterName || invoice?.transporter_name || '-'}<br />
					<strong>Place:</strong> {extraDetails.deliveryPlace || invoice?.delivery_place || invoice?.supply_state || '-'}
				</div>
			</div>

			<table class="tpl-table" style="margin-top: 14px;">
				<thead>
					<tr>
						<th style="width: 5%;">#</th>
						<th style="width: 35%;">Product / Material Description</th>
						<th class="text-center" style="width: 12%;">HSN/SAC</th>
						<th class="text-right" style="width: 12%;">Dispatch Qty</th>
						<th class="text-right" style="width: 16%;">Unit Price</th>
						<th class="text-right" style="width: 20%;">Total Value</th>
					</tr>
				</thead>
				<tbody>
					{#if invoice?.items}
						{#each invoice.items as it, i}
							<tr>
								<td class="text-center">{i + 1}</td>
								<td><strong>{it.item_description}</strong></td>
								<td class="text-center font-mono">{it.hsn_sac || '-'}</td>
								<td class="text-right font-mono">{parseFloat(it.quantity).toFixed(2)} {it.unit_code}</td>
								<td class="text-right font-mono">{formatCurrency(it.unit_price)}</td>
								<td class="text-right font-mono" style="font-weight: 700;">{formatCurrency(it.total_amount)}</td>
							</tr>
						{/each}
					{/if}
				</tbody>
			</table>

			<div style="display: flex; justify-content: space-between; border: 1px solid #cbd5e1; padding: 10px 14px; background: #f8fafc; font-size: 11px; flex-wrap: wrap; gap: 8px;">
				<div>Total Quantity: <strong>{invoice?.items?.reduce((acc, it) => acc + parseFloat(it.quantity), 0).toFixed(2)} Units</strong></div>
				<div>Net Weight: <strong>{effectiveNetWeight || 'As Per Manifest'}</strong></div>
				{#if effectiveGrossWeight}
					<div>Gross Weight: <strong>{effectiveGrossWeight}</strong></div>
				{/if}
				<div style="font-size: 14px; font-weight: 900;">Grand Total: <span class="font-mono">{formatCurrency(invoice?.grand_total)}</span></div>
				{#if invoice && parseFloat(invoice.amount_paid || '0') > 0}
					<div style="width: 100%; display: flex; justify-content: flex-end; gap: 18px; font-size: 12px;">
						<span style="color: #047857;">Amount Paid: <strong class="font-mono">{formatCurrency(invoice.amount_paid)}</strong></span>
						<span style="color: #b45309;">Balance Due: <strong class="font-mono">{formatCurrency(invoice.balance_due)}</strong></span>
					</div>
				{/if}
			</div>

			<div style="display: flex; justify-content: space-between; margin-top: 14px; align-items: flex-start; flex-wrap: wrap; gap: 14px;">
				<div style="max-width: 420px; font-size: 11px; color: #334155;">
					{#if showBank && (effectiveBankName || effectiveAccountNo)}
						<div class="tpl-bank-box" style="margin-bottom: 6px;">
							<div class="tpl-bank-title">Bank Direct Credit Details</div>
							<div><strong>Bank:</strong> {effectiveBankName} {#if effectiveBranch}({effectiveBranch}){/if} | <strong>A/C:</strong> <span class="font-mono">{effectiveAccountNo}</span></div>
							<div><strong>IFSC:</strong> <span class="font-mono">{effectiveIFSC}</span> {#if effectiveAccountHolder}| <strong>Holder:</strong> {effectiveAccountHolder}{/if}</div>
						</div>
					{/if}
					<div>* Subject to {effectiveJurisdiction} Jurisdiction. Interest @ {effectiveInterestRate} will be charged on overdue payments.</div>
					<div>* {effectiveReturnPolicy}</div>
				</div>

				<div style="text-align: right; font-size: 11px;">
					<div style="color: #64748b; margin-bottom: 4px;">For {company?.legal_name || company?.name}</div>
					{#if effectiveSignatureUrl}
						<div><img src={effectiveSignatureUrl} alt="Sign" class="tpl-signature-img" /></div>
					{/if}
					<div style="font-weight: 700; color: #0f172a; margin-top: 4px;">{effectiveSignatoryTitle}</div>
				</div>
			</div>
		</div>

	<!-- ====================================================================
	     TEMPLATE 8: BOLD TECH SAAS
	     ==================================================================== -->
	{:else if templateId === '8'}
		<div class="tpl-saas" style="position: relative; z-index: 1;">
			<div class="hero-banner">
				<div>
					<div class="brand-name">{company?.legal_name || company?.name}</div>
					<div style="font-size: 12px; opacity: 0.85; margin-top: 2px;">{effectiveSubtitle || 'Cloud Platform & Software Billing'}</div>
				</div>
				<div style="text-align: right;">
					<span class="plan-tag">{copyMeta.title}</span>
					<div style="font-size: 16px; font-weight: 800; margin-top: 6px;">SUBSCRIPTION RECEIPT</div>
				</div>
			</div>

			<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 16px; margin-bottom: 20px; font-size: 12px;">
				<div style="background: #f8fafc; padding: 12px; border-radius: 6px; border: 1px solid #e2e8f0;">
					<span style="font-size: 10px; font-weight: 700; color: #6366f1; text-transform: uppercase;">Subscriber:</span>
					<div style="font-weight: 700; font-size: 13px;">{invoice?.customer_name}</div>
					<div style="color: #64748b;">{invoice?.billing_address}</div>
					{#if isTaxInvoice && invoice?.customer_gstin}
						<div style="font-family: var(--font-mono); font-size: 11px; color: #4f46e5; margin-top: 2px;">GSTIN: {invoice.customer_gstin}</div>
					{/if}
				</div>
				<div style="background: #f8fafc; padding: 12px; border-radius: 6px; border: 1px solid #e2e8f0;">
					<span style="font-size: 10px; font-weight: 700; color: #6366f1; text-transform: uppercase;">Billing Cycle & Terms:</span>
					<div style="font-weight: 700; font-size: 13px;">Invoice {invoice?.invoice_number}</div>
					<div style="color: #64748b;">Billed: {formatDate(invoice?.invoice_date)} • Terms: <strong>{effectiveTerms}</strong></div>
					{#if effectiveSubscriptionStart || effectiveSubscriptionEnd}
						<div style="margin-top: 4px; font-weight: 600; color: #4338ca; font-size: 11.5px;">
							Service Period: {formatDate(effectiveSubscriptionStart)} to {formatDate(effectiveSubscriptionEnd)}
						</div>
					{/if}
				</div>
			</div>

			<table class="tpl-table">
				<thead>
					<tr>
						<th style="width: 50%;">Plan / Subscription Item</th>
						<th class="text-right" style="width: 15%;">Seats/Qty</th>
						<th class="text-right" style="width: 15%;">Price</th>
						<th class="text-right" style="width: 20%;">Total</th>
					</tr>
				</thead>
				<tbody>
					{#if invoice?.items}
						{#each invoice.items as it}
							<tr>
								<td>
									<strong>{it.item_description}</strong>
									{#if isTaxInvoice && it.hsn_sac}
										<div style="font-size: 10px; color: #94a3b8; font-family: var(--font-mono);">SAC: {it.hsn_sac}</div>
									{/if}
								</td>
								<td class="text-right font-mono">{parseFloat(it.quantity).toFixed(0)}</td>
								<td class="text-right font-mono">{formatCurrency(it.unit_price)}</td>
								<td class="text-right font-mono" style="font-weight: 700;">{formatCurrency(it.total_amount)}</td>
							</tr>
						{/each}
					{/if}
				</tbody>
			</table>

			<div style="display: flex; justify-content: space-between; margin-top: 14px; align-items: flex-start; flex-wrap: wrap; gap: 14px;">
				<div style="max-width: 360px; font-size: 11px;">
					{#if showBank && (effectiveBankName || effectiveAccountNo)}
						<div class="tpl-bank-box" style="margin-bottom: 8px;">
							<div class="tpl-bank-title">Direct Wire Settlement</div>
							<div><strong>Bank:</strong> {effectiveBankName} {#if effectiveBranch}({effectiveBranch}){/if}</div>
							<div><strong>A/C No:</strong> <span class="font-mono" style="font-weight: 700;">{effectiveAccountNo}</span></div>
							<div><strong>IFSC:</strong> <span class="font-mono" style="font-weight: 700;">{effectiveIFSC}</span></div>
						</div>
					{/if}
					<div style="color: #64748b;">* Subject to {effectiveJurisdiction} Jurisdiction. {effectiveReturnPolicy}</div>
				</div>

				<div style="display: flex; align-items: center; gap: 14px;">
					{#if showUPI && effectiveUPI}
						<div class="tpl-upi-qr-box" style="padding: 6px 10px;">
							<img class="tpl-upi-qr-img" src={getUPIQRUrl(effectiveUPI, company?.name || '', invoice?.grand_total)} alt="UPI QR" />
							<span class="tpl-upi-text">Pay Instantly</span>
						</div>
					{/if}
					<div style="text-align: right;">
						<span style="font-size: 12px; color: #64748b;">Total Billed:</span>
						<div style="font-size: 24px; font-weight: 900; color: #312e81; font-family: var(--font-mono);">
							{formatCurrency(invoice?.grand_total)}
						</div>
						<div style="min-width: 220px; font-size: 12px;">{@render payRows()}</div>
						<div style="margin-top: 10px; font-size: 11px;">
							{#if effectiveSignatureUrl}
								<div><img src={effectiveSignatureUrl} alt="Sign" class="tpl-signature-img" /></div>
							{/if}
							<div style="font-weight: 700; color: #312e81;">{effectiveSignatoryTitle}</div>
						</div>
					</div>
				</div>
			</div>
		</div>

	<!-- ====================================================================
	     TEMPLATE 9: WHOLESALE MULTI-TAX SLAB TABLE
	     ==================================================================== -->
	{:else if templateId === '9'}
		<div class="tpl-wholesale" style="position: relative; z-index: 1;">
			<div class="company-box">
				<div style="display: flex; justify-content: space-between; align-items: center;">
					<h2 style="font-size: 20px; font-weight: 800; color: #115e59;">{company?.legal_name || company?.name}</h2>
					<span class={`copy-badge ${copyMeta.class}`}>{copyMeta.title}</span>
				</div>
				<div style="font-size: 11px; color: #134e4a; margin-top: 2px;">
					{effectiveSubtitle || 'WHOLESALE DISTRIBUTORS & TRADERS'} • GSTIN: {company?.gstin} • Invoice: <strong>{invoice?.invoice_number}</strong>
				</div>
			</div>

			<table class="tpl-table">
				<thead>
					<tr>
						<th>#</th>
						<th>Description</th>
						<th class="text-center">HSN</th>
						<th class="text-right">Qty</th>
						<th class="text-right">Rate</th>
						<th class="text-right">Taxable</th>
						<th class="text-right">Total</th>
					</tr>
				</thead>
				<tbody>
					{#if invoice?.items}
						{#each invoice.items as it, i}
							<tr>
								<td>{i + 1}</td>
								<td><strong>{it.item_description}</strong></td>
								<td class="text-center font-mono">{it.hsn_sac || '-'}</td>
								<td class="text-right font-mono">{parseFloat(it.quantity).toFixed(2)} {it.unit_code}</td>
								<td class="text-right font-mono">{formatCurrency(it.unit_price)}</td>
								<td class="text-right font-mono">{formatCurrency(it.taxable_value)}</td>
								<td class="text-right font-mono" style="font-weight: 700;">{formatCurrency(it.total_amount)}</td>
							</tr>
						{/each}
					{/if}
				</tbody>
			</table>

			<!-- Statutory HSN Tax Slab Summary Table -->
			{#if isTaxInvoice && showTaxSummary}
				<div style="margin-top: 14px;">
					<div style="font-size: 11px; font-weight: 700; color: #115e59; margin-bottom: 4px;">HSN/SAC Tax Slab Summary:</div>
					<table class="hsn-summary-table">
						<thead>
							<tr>
								<th>HSN/SAC</th>
								<th class="text-right">Taxable Amount</th>
								<th class="text-right">CGST Rate</th>
								<th class="text-right">CGST Amt</th>
								<th class="text-right">SGST Rate</th>
								<th class="text-right">SGST Amt</th>
								<th class="text-right">Total Tax</th>
							</tr>
						</thead>
						<tbody>
							{#if invoice?.items}
								{#each invoice.items as it}
									{@const halfRate = (parseFloat(it.tax_rate) / 2).toFixed(1)}
									{@const cgstAmt = parseFloat(it.cgst_amount || '0').toFixed(2)}
									{@const sgstAmt = parseFloat(it.sgst_amount || '0').toFixed(2)}
									<tr>
										<td class="font-mono">{it.hsn_sac || 'OTHER'}</td>
										<td class="text-right font-mono">{formatCurrency(it.taxable_value)}</td>
										<td class="text-right font-mono">{halfRate}%</td>
										<td class="text-right font-mono">{cgstAmt}</td>
										<td class="text-right font-mono">{halfRate}%</td>
										<td class="text-right font-mono">{sgstAmt}</td>
										<td class="text-right font-mono font-bold">{formatCurrency((parseFloat(cgstAmt) + parseFloat(sgstAmt)).toString())}</td>
									</tr>
								{/each}
							{/if}
						</tbody>
					</table>
				</div>
			{/if}

			<div style="display: flex; justify-content: space-between; align-items: flex-start; margin-top: 14px; flex-wrap: wrap; gap: 14px;">
				<div style="max-width: 380px; font-size: 11px;">
					{#if showBank && (effectiveBankName || effectiveAccountNo)}
						<div class="tpl-bank-box" style="margin-bottom: 6px;">
							<div class="tpl-bank-title">Bank RTGS / NEFT Particulars</div>
							<div><strong>Bank:</strong> {effectiveBankName} {#if effectiveBranch}({effectiveBranch}){/if}</div>
							<div><strong>A/C:</strong> <span class="font-mono" style="font-weight: 700;">{effectiveAccountNo}</span> | <strong>IFSC:</strong> <span class="font-mono">{effectiveIFSC}</span></div>
						</div>
					{/if}
					<div>* Terms: <strong>{effectiveTerms}</strong> • Jurisdiction: {effectiveJurisdiction}</div>
					<div>* {effectiveReturnPolicy}</div>
				</div>

				<div style="display: flex; align-items: flex-end; gap: 14px;">
					{#if showUPI && effectiveUPI}
						<div class="tpl-upi-qr-box">
							<img class="tpl-upi-qr-img" src={getUPIQRUrl(effectiveUPI, company?.name || '', invoice?.grand_total)} alt="UPI QR" />
							<span class="tpl-upi-text">Scan UPI</span>
						</div>
					{/if}
					<div style="text-align: right;">
						<div style="font-size: 16px; font-weight: 800; color: #0f766e;">
							Grand Total: <span class="font-mono">{formatCurrency(invoice?.grand_total)}</span>
						</div>
						<div style="min-width: 220px; font-size: 12px;">{@render payRows()}</div>
						<div style="margin-top: 12px; font-size: 11px;">
							{#if effectiveSignatureUrl}
								<div><img src={effectiveSignatureUrl} alt="Sign" class="tpl-signature-img" /></div>
							{/if}
							<div style="font-weight: 700; color: #0f766e;">{effectiveSignatoryTitle}</div>
						</div>
					</div>
				</div>
			</div>
		</div>

	<!-- ====================================================================
	     TEMPLATE 10: SIMPLE BILL OF SUPPLY / NON-GST CASH MEMO
	     ==================================================================== -->
	{:else if templateId === '10'}
		<div class="tpl-simple" style="position: relative; z-index: 1;">
			<div class="bill-head">
				<div style="margin-bottom: 4px;">
					<span class={`copy-badge ${copyMeta.class}`}>{copyMeta.title}</span>
				</div>
				<h2>{company?.legal_name || company?.name}</h2>
				{#if effectiveSubtitle}
					<div style="font-size: 12px; font-weight: 600; color: #475569; margin-top: 2px;">{effectiveSubtitle}</div>
				{/if}
				<div style="font-size: 12px; color: #64748b;">
					{company?.address}, {company?.city}, {company?.state} - {company?.pincode}
				</div>
				<div style="font-size: 16px; font-weight: 800; color: #0f172a; margin-top: 6px; letter-spacing: 0.05em;">
					BILL OF SUPPLY (NON-GST / CASH MEMO)
				</div>
			</div>

			<div class="composition-notice">
				{company?.template_options?.composition_notice || extraDetails?.compositionNotice || '⚠️ Composition Taxable Person, Not Eligible to Collect Tax on Supplies (Sec 31(3)(c) of CGST Act)'}
			</div>

			<div style="display: flex; justify-content: space-between; font-size: 12px; margin-bottom: 16px; flex-wrap: wrap; gap: 10px;">
				<div>
					<strong>Billed To:</strong> {invoice?.customer_name}<br />
					{invoice?.billing_address}
				</div>
				<div style="text-align: right;">
					<strong>Bill No:</strong> {invoice?.invoice_number}<br />
					<strong>Date:</strong> {formatDate(invoice?.invoice_date)}<br />
					<strong>Mode:</strong> {effectiveTerms}
				</div>
			</div>

			<table class="tpl-table">
				<thead>
					<tr>
						<th style="width: 8%;">S.No</th>
						<th style="width: 52%;">Description of Item / Goods</th>
						<th class="text-right" style="width: 15%;">Quantity</th>
						<th class="text-right" style="width: 15%;">Rate</th>
						<th class="text-right" style="width: 20%;">Total Amount</th>
					</tr>
				</thead>
				<tbody>
					{#if invoice?.items}
						{#each invoice.items as it, i}
							<tr>
								<td>{i + 1}</td>
								<td><strong>{it.item_description}</strong></td>
								<td class="text-right font-mono">{parseFloat(it.quantity).toFixed(0)} {it.unit_code}</td>
								<td class="text-right font-mono">{formatCurrency(it.unit_price)}</td>
								<td class="text-right font-mono" style="font-weight: 700;">
									{formatCurrency(parseFloat(it.quantity) * parseFloat(it.unit_price))}
								</td>
							</tr>
						{/each}
					{/if}
				</tbody>
			</table>

			<div style="display: flex; justify-content: space-between; align-items: flex-end; margin-top: 20px; flex-wrap: wrap; gap: 14px;">
				<div style="max-width: 360px; font-size: 11px; color: #64748b;">
					{#if showBank && (effectiveBankName || effectiveAccountNo)}
						<div class="tpl-bank-box" style="margin-bottom: 6px;">
							<div class="tpl-bank-title">Payment Settlement</div>
							<div><strong>Bank:</strong> {effectiveBankName} {#if effectiveBranch}({effectiveBranch}){/if}</div>
							<div><strong>A/C:</strong> {effectiveAccountNo} | <strong>IFSC:</strong> {effectiveIFSC}</div>
						</div>
					{/if}
					<div>Total Amount in Words: <strong style="color: #0f172a;">{grandTotalWords}</strong></div>
					<div style="margin-top: 4px;">* {effectiveReturnPolicy}</div>
				</div>

				<div style="display: flex; align-items: flex-end; gap: 14px;">
					{#if showUPI && effectiveUPI}
						<div class="tpl-upi-qr-box">
							<img class="tpl-upi-qr-img" src={getUPIQRUrl(effectiveUPI, company?.name || '', invoice?.grand_total)} alt="UPI QR" />
							<span class="tpl-upi-text">Scan & Pay</span>
						</div>
					{/if}
					<div style="text-align: right;">
						<div style="font-size: 12px; color: #64748b;">Net Amount Payable:</div>
						<div style="font-size: 22px; font-weight: 900; color: #0f172a; font-family: var(--font-mono);">
							{formatCurrency(invoice?.grand_total)}
						</div>
						<div style="min-width: 220px; font-size: 12px;">{@render payRows()}</div>
						<div style="margin-top: 12px; font-size: 11px;">
							{#if effectiveSignatureUrl}
								<div><img src={effectiveSignatureUrl} alt="Sign" class="tpl-signature-img" /></div>
							{/if}
							<div style="font-weight: 700; color: #0f172a;">{effectiveSignatoryTitle}</div>
						</div>
					</div>
				</div>
			</div>
		</div>

	<!-- ====================================================================
	     TEMPLATE 12: HOSTEL / LODGING GUEST FOLIO
	     Stay details (room/bed, check-in/out, nights) + room-charge & services table
	     Field mapping: Room/Bed = counter_number, Booking Ref = project_name,
	     Front Desk = cashier_name, Check-in/out = subscription_period_start/end
	     ==================================================================== -->
	{:else if templateId === '12'}
		<div class="tpl-hostel" style="position: relative; z-index: 1;">
			<div class="hostel-header">
				<div>
					<div class="hostel-name">{company?.legal_name || company?.name}</div>
					<div class="hostel-sub">{effectiveSubtitle || 'Hostel & Guest Accommodation'}</div>
					<div class="hostel-addr">{companyAddressLine}</div>
					<div class="hostel-addr">
						{#if company?.phone}Ph: {company.phone}{/if}{#if company?.email} &nbsp;|&nbsp; {company.email}{/if}
						{#if isTaxInvoice && company?.gstin} &nbsp;|&nbsp; GSTIN: {company.gstin}{/if}
					</div>
				</div>
				<div style="text-align: right;">
					<span class={`copy-badge ${copyMeta.class}`}>{copyMeta.title}</span>
					<div class="hostel-doc-title">{isTaxInvoice ? 'GUEST FOLIO · TAX INVOICE' : 'GUEST FOLIO · BILL OF SUPPLY'}</div>
					<div class="hostel-docno">{invoice?.invoice_number}</div>
					<div style="font-size: 11px; opacity: 0.9;">Billed on {formatDate(invoice?.invoice_date)}</div>
				</div>
			</div>

			<div class="hostel-stay-grid">
				<div class="hostel-cell">
					<span class="hostel-label">Guest</span>
					<strong>{invoice?.customer_name || 'Walk-in Guest'}</strong>
					{#if invoice?.billing_address}<div class="hostel-muted">{invoice.billing_address}</div>{/if}
					{#if isTaxInvoice && invoice?.customer_gstin}<div class="hostel-muted font-mono">GSTIN: {invoice.customer_gstin}</div>{/if}
				</div>
				<div class="hostel-cell">
					<span class="hostel-label">Room / Bed</span>
					<strong>{invoice?.counter_number || extraDetails?.counterNumber || '-'}</strong>
					<span class="hostel-label" style="margin-top: 6px;">Booking Ref</span>
					<strong>{invoice?.project_name || extraDetails?.projectName || '-'}</strong>
				</div>
				<div class="hostel-cell">
					<span class="hostel-label">Check-in</span>
					<strong>{effectiveSubscriptionStart ? formatDate(effectiveSubscriptionStart) : '-'}</strong>
					<span class="hostel-label" style="margin-top: 6px;">Check-out</span>
					<strong>{effectiveSubscriptionEnd ? formatDate(effectiveSubscriptionEnd) : '-'}</strong>
				</div>
				<div class="hostel-cell">
					<span class="hostel-label">Nights</span>
					<strong style="font-size: 18px;">{nightsStayed || '-'}</strong>
					<span class="hostel-label" style="margin-top: 6px;">Front Desk</span>
					<strong>{invoice?.cashier_name || extraDetails?.cashierName || '-'}</strong>
				</div>
			</div>

			<table class="tpl-table">
				<thead>
					<tr>
						<th style="width: 5%;">#</th>
						<th style="width: {isTaxInvoice ? '33%' : '43%'};">Description (Room / Services)</th>
						{#if isTaxInvoice}<th class="text-center" style="width: 10%;">SAC</th>{/if}
						<th class="text-right" style="width: 10%;">Nights / Qty</th>
						<th class="text-right" style="width: 14%;">Rate</th>
						{#if isTaxInvoice}<th class="text-right" style="width: 8%;">GST %</th>{/if}
						<th class="text-right" style="width: 16%;">Amount</th>
					</tr>
				</thead>
				<tbody>
					{#if invoice?.items}
						{#each invoice.items as it, i}
							<tr>
								<td class="text-center">{i + 1}</td>
								<td><strong>{it.item_description}</strong></td>
								{#if isTaxInvoice}<td class="text-center font-mono">{it.hsn_sac || '996311'}</td>{/if}
								<td class="text-right font-mono">{parseFloat(it.quantity).toFixed(0)} {it.unit_code}</td>
								<td class="text-right font-mono">{formatCurrency(it.unit_price)}</td>
								{#if isTaxInvoice}<td class="text-right font-mono">{parseFloat(it.tax_rate || '0').toFixed(0)}%</td>{/if}
								<td class="text-right font-mono" style="font-weight: 700;">{formatCurrency(isTaxInvoice ? it.taxable_value : it.total_amount)}</td>
							</tr>
						{/each}
					{:else}
						<tr><td colspan="7" class="text-center" style="padding: 20px; color: #94a3b8;">No charges posted to this folio.</td></tr>
					{/if}
				</tbody>
			</table>

			<div class="hostel-bottom">
				<div style="flex: 1; min-width: 240px; font-size: 11px; color: #334155;">
					{#if isTaxInvoice && taxSlabs.length > 0 && showTaxSummary}
						<table class="hostel-slab">
							<thead>
								<tr><th>GST Slab</th><th class="text-right">Taxable</th><th class="text-right">{invoice?.is_inter_state ? 'IGST' : 'CGST'}</th>{#if !invoice?.is_inter_state}<th class="text-right">SGST</th>{/if}</tr>
							</thead>
							<tbody>
								{#each taxSlabs as s}
									<tr>
										<td>{s.rate.toFixed(0)}%</td>
										<td class="text-right font-mono">{formatCurrency(s.taxable)}</td>
										<td class="text-right font-mono">{formatCurrency(invoice?.is_inter_state ? s.igst : s.cgst)}</td>
										{#if !invoice?.is_inter_state}<td class="text-right font-mono">{formatCurrency(s.sgst)}</td>{/if}
									</tr>
								{/each}
							</tbody>
						</table>
					{/if}
					{#if showBank && (effectiveBankName || effectiveAccountNo)}
						<div class="tpl-bank-box" style="margin-top: 8px;">
							<div class="tpl-bank-title">Bank Details</div>
							<div><strong>{effectiveBankName}</strong> {#if effectiveBranch}({effectiveBranch}){/if} · A/C <span class="font-mono">{effectiveAccountNo}</span> · IFSC <span class="font-mono">{effectiveIFSC}</span></div>
						</div>
					{/if}
					{#if showUPI && effectiveUPI}
						<div style="display: flex; gap: 10px; align-items: center; margin-top: 8px;">
							<img class="tpl-upi-qr-img" src={getUPIQRUrl(effectiveUPI, company?.legal_name || company?.name || '', invoice?.balance_due || invoice?.grand_total)} alt="UPI QR" />
							<div style="font-size: 10.5px;"><strong>Pay balance by UPI</strong><br /><span class="font-mono">{effectiveUPI}</span></div>
						</div>
					{/if}
					{#if invoice?.notes}<div style="margin-top: 8px;"><strong>Remarks:</strong> {invoice.notes}</div>{/if}
				</div>

				<div class="hostel-totals">
					<div class="hostel-trow"><span>Room & Service Charges</span><span class="font-mono">{formatCurrency(invoice?.subtotal)}</span></div>
					{#if invoice && parseFloat(invoice.discount_total) > 0}
						<div class="hostel-trow" style="color: #dc2626;"><span>Discount</span><span class="font-mono">- {formatCurrency(invoice.discount_total)}</span></div>
					{/if}
					{#if isTaxInvoice}
						{#if invoice?.is_inter_state}
							<div class="hostel-trow"><span>IGST</span><span class="font-mono">{formatCurrency(invoice?.igst_total)}</span></div>
						{:else}
							<div class="hostel-trow"><span>CGST</span><span class="font-mono">{formatCurrency(invoice?.cgst_total)}</span></div>
							<div class="hostel-trow"><span>SGST</span><span class="font-mono">{formatCurrency(invoice?.sgst_total)}</span></div>
						{/if}
					{/if}
					{#if invoice && parseFloat(invoice.round_off) !== 0}
						<div class="hostel-trow" style="color: #64748b; font-size: 11px;"><span>Round Off</span><span class="font-mono">{fmtSigned(invoice.round_off)}</span></div>
					{/if}
					<div class="hostel-trow hostel-grand"><span>Total</span><span class="font-mono">{formatCurrency(invoice?.grand_total)}</span></div>
					<div class="hostel-trow" style="color: #047857;"><span>Advance / Paid</span><span class="font-mono">{formatCurrency(invoice?.amount_paid)}</span></div>
					<div class="hostel-trow" style="color: #b45309; font-weight: 800;"><span>Balance Payable</span><span class="font-mono">{formatCurrency(invoice?.balance_due)}</span></div>
				</div>
			</div>

			<div style="font-size: 10.5px; color: #475569; margin-top: 8px;"><strong>In words:</strong> {grandTotalWords}</div>

			<div class="hostel-footer">
				<div style="font-size: 10px; color: #64748b; max-width: 360px;">
					<div>Payment terms: {effectiveTerms}. Standard check-out time applies; late check-out may be charged.</div>
					<div>{effectiveReturnPolicy}</div>
					<div>Subject to {effectiveJurisdiction} jurisdiction. Computer-generated folio.</div>
				</div>
				<div class="hostel-sign">
					<div class="hostel-sign-line">Guest Signature</div>
				</div>
				<div class="hostel-sign">
					{#if effectiveSignatureUrl}<img src={effectiveSignatureUrl} alt="Sign" class="tpl-signature-img" />{:else}<div style="height: 30px;"></div>{/if}
					<div class="hostel-sign-line">{effectiveSignatoryTitle}<br /><span style="font-weight: 500; font-size: 9.5px;">For {company?.legal_name || company?.name}</span></div>
				</div>
			</div>
		</div>

	<!-- ====================================================================
	     TEMPLATE 13: RESTAURANT / CAFE BILL
	     Dine-in bill with table, waiter, KOT/order no, FSSAI licence, GST split
	     Field mapping: Table = counter_number, Waiter = cashier_name,
	     KOT / Order No = project_name, Mode = payment_mode
	     ==================================================================== -->
	{:else if templateId === '13'}
		<div class="tpl-restaurant" style="position: relative; z-index: 1;">
			<div class="rest-head">
				<span class={`copy-badge ${copyMeta.class}`}>{copyMeta.title}</span>
				<div class="rest-name">{company?.legal_name || company?.name}</div>
				{#if effectiveSubtitle}<div class="rest-sub">{effectiveSubtitle}</div>{/if}
				<div class="rest-addr">{companyAddressLine}</div>
				<div class="rest-addr">
					{#if company?.phone}Ph: {company.phone}{/if}
					{#if isTaxInvoice && company?.gstin} &nbsp;·&nbsp; GSTIN: {company.gstin}{/if}
					{#if company?.fssai_no} &nbsp;·&nbsp; FSSAI Lic: {company.fssai_no}{/if}
				</div>
				<div class="rest-title">{isTaxInvoice ? 'TAX INVOICE' : 'BILL OF SUPPLY'}</div>
			</div>

			<div class="rest-meta">
				<div><span>Bill No</span><strong>{invoice?.invoice_number}</strong></div>
				<div><span>Date</span><strong>{formatDate(invoice?.invoice_date)}</strong></div>
				<div><span>Table</span><strong>{invoice?.counter_number || extraDetails?.counterNumber || '-'}</strong></div>
				<div><span>Waiter</span><strong>{invoice?.cashier_name || extraDetails?.cashierName || '-'}</strong></div>
				<div><span>KOT / Order</span><strong>{invoice?.project_name || extraDetails?.projectName || '-'}</strong></div>
				<div><span>Mode</span><strong>{effectiveTerms}</strong></div>
			</div>
			{#if invoice?.customer_name && invoice.customer_name.toLowerCase() !== 'cash customer' && invoice.customer_name.toLowerCase() !== 'walk-in customer'}
				<div class="rest-guest">Guest: <strong>{invoice.customer_name}</strong>{#if isTaxInvoice && invoice?.customer_gstin} · GSTIN <span class="font-mono">{invoice.customer_gstin}</span>{/if}</div>
			{/if}

			<table class="rest-table">
				<thead>
					<tr>
						<th style="width: 50%;">Item</th>
						<th class="text-right" style="width: 12%;">Qty</th>
						<th class="text-right" style="width: 18%;">Rate</th>
						<th class="text-right" style="width: 20%;">Amount</th>
					</tr>
				</thead>
				<tbody>
					{#if invoice?.items}
						{#each invoice.items as it}
							<tr>
								<td>{it.item_description}</td>
								<td class="text-right font-mono">{parseFloat(it.quantity).toFixed(parseFloat(it.quantity) % 1 === 0 ? 0 : 2)}</td>
								<td class="text-right font-mono">{formatCurrency(it.unit_price)}</td>
								<td class="text-right font-mono">{formatCurrency(isTaxInvoice ? it.taxable_value : it.total_amount)}</td>
							</tr>
						{/each}
					{/if}
				</tbody>
			</table>

			<div class="rest-totals">
				<div class="rest-trow"><span>Sub Total</span><span class="font-mono">{formatCurrency(invoice?.subtotal)}</span></div>
				{#if invoice && parseFloat(invoice.discount_total) > 0}
					<div class="rest-trow" style="color: #dc2626;"><span>Discount</span><span class="font-mono">- {formatCurrency(invoice.discount_total)}</span></div>
				{/if}
				{#if isTaxInvoice}
					{#if invoice?.is_inter_state}
						<div class="rest-trow"><span>IGST</span><span class="font-mono">{formatCurrency(invoice?.igst_total)}</span></div>
					{:else}
						{#each taxSlabs.filter((x) => x.rate > 0) as s}
							<div class="rest-trow"><span>CGST @ {(s.rate / 2).toFixed(1)}%</span><span class="font-mono">{formatCurrency(s.cgst)}</span></div>
							<div class="rest-trow"><span>SGST @ {(s.rate / 2).toFixed(1)}%</span><span class="font-mono">{formatCurrency(s.sgst)}</span></div>
						{/each}
					{/if}
				{/if}
				{#if invoice && parseFloat(invoice.round_off) !== 0}
					<div class="rest-trow" style="color: #64748b;"><span>Round Off</span><span class="font-mono">{fmtSigned(invoice.round_off)}</span></div>
				{/if}
				<div class="rest-trow rest-grand"><span>TOTAL</span><span class="font-mono">{formatCurrency(invoice?.grand_total)}</span></div>
				{#if invoice && parseFloat(invoice.amount_paid) > 0}
					<div class="rest-trow" style="color: #047857;"><span>Paid</span><span class="font-mono">{formatCurrency(invoice.amount_paid)}</span></div>
					{#if parseFloat(invoice.balance_due) > 0}
						<div class="rest-trow" style="color: #b45309; font-weight: 800;"><span>Balance</span><span class="font-mono">{formatCurrency(invoice.balance_due)}</span></div>
					{/if}
				{/if}
			</div>
			<div class="rest-words">{grandTotalWords}</div>

			{#if showUPI && effectiveUPI}
				<div class="rest-upi">
					<img class="tpl-upi-qr-img" src={getUPIQRUrl(effectiveUPI, company?.legal_name || company?.name || '', invoice?.balance_due || invoice?.grand_total)} alt="UPI QR" />
					<div><strong>Scan & Pay</strong><br /><span class="font-mono">{effectiveUPI}</span></div>
				</div>
			{/if}

			<div class="rest-foot">
				{#if invoice?.notes}<div>{invoice.notes}</div>{/if}
				<div style="font-weight: 700; margin-top: 4px;">{company?.template_options?.restaurant_footer || 'Thank you! Visit again.'}</div>
				<div style="font-size: 9.5px; color: #94a3b8; margin-top: 2px;">Subject to {effectiveJurisdiction} jurisdiction · Computer-generated bill</div>
			</div>
		</div>

	<!-- ====================================================================
	     TEMPLATE 14: PHARMACY / MEDICAL STORE TAX INVOICE
	     Drug licence header, patient & doctor block, per-line batch / expiry / MRP,
	     short-expiry flag, MRP savings, GST slab summary, Schedule H notice,
	     pharmacist signature. Uses invoice.patient_name/age, doctor_name/reg_no,
	     item batch_number/expiry_date/mrp/rack_location and company.drug_license_no.
	     ==================================================================== -->
	{:else if templateId === '14'}
		<div class="tpl-pharmacy" style="position: relative; z-index: 1;">
			<div class="rx-header">
				<div class="rx-brand">
					<div class="rx-cross">+</div>
					<div>
						<div class="rx-name">{company?.legal_name || company?.name}</div>
						<div class="rx-sub">{effectiveSubtitle || 'Pharmacy & Medical Store'}</div>
						<div class="rx-addr">{companyAddressLine}</div>
						<div class="rx-addr">
							{#if company?.phone}Ph: {company.phone}{/if}{#if company?.email} &nbsp;|&nbsp; {company.email}{/if}
						</div>
					</div>
				</div>
				<div class="rx-licence">
					{#if company?.drug_license_no}<div><strong>D.L. No:</strong> {company.drug_license_no}</div>{/if}
					{#if isTaxInvoice && company?.gstin}<div><strong>GSTIN:</strong> {company.gstin}</div>{/if}
					{#if company?.fssai_no}<div><strong>FSSAI:</strong> {company.fssai_no}</div>{/if}
					<span class={`copy-badge ${copyMeta.class}`} style="margin-top: 6px;">{copyMeta.title}</span>
				</div>
			</div>

			<div class="rx-titlebar">
				<span>{isTaxInvoice ? 'TAX INVOICE / CASH-CREDIT MEMO' : 'BILL OF SUPPLY / CASH-CREDIT MEMO'}</span>
				<span>Bill No: <strong class="font-mono">{invoice?.invoice_number}</strong> &nbsp; Date: <strong>{formatDate(invoice?.invoice_date)}</strong></span>
			</div>

			<div class="rx-party-grid">
				<div class="rx-cell">
					<span class="rx-label">Patient / Customer</span>
					<strong>{invoice?.patient_name || invoice?.customer_name || 'Cash Customer'}</strong>
					{#if invoice?.patient_age}<span class="rx-muted">Age {invoice.patient_age} yrs</span>{/if}
					{#if invoice?.billing_address}<div class="rx-muted">{invoice.billing_address}</div>{/if}
					{#if isTaxInvoice && invoice?.customer_gstin}<div class="rx-muted font-mono">GSTIN: {invoice.customer_gstin}</div>{/if}
				</div>
				<div class="rx-cell">
					<span class="rx-label">Prescribed By</span>
					<strong>{invoice?.doctor_name || '-'}</strong>
					{#if invoice?.doctor_reg_no}<div class="rx-muted">Reg. No: {invoice.doctor_reg_no}</div>{/if}
					<span class="rx-label" style="margin-top: 6px;">Mode of Payment</span>
					<strong>{effectiveTerms}</strong>
				</div>
				<div class="rx-cell">
					<span class="rx-label">Billed By</span>
					<strong>{invoice?.cashier_name || extraDetails?.cashierName || '-'}</strong>
					<span class="rx-label" style="margin-top: 6px;">Counter</span>
					<strong>{invoice?.counter_number || extraDetails?.counterNumber || '-'}</strong>
				</div>
			</div>

			<table class="tpl-table rx-table">
				<thead>
					<tr>
						<th style="width: 4%;">#</th>
						<th style="width: {isTaxInvoice ? '28%' : '33%'};">Product Description</th>
						<th class="text-center" style="width: 8%;">HSN</th>
						<th class="text-center" style="width: 10%;">Batch</th>
						<th class="text-center" style="width: 8%;">Exp</th>
						<th class="text-right" style="width: 9%;">MRP</th>
						<th class="text-right" style="width: 6%;">Qty</th>
						<th class="text-right" style="width: 9%;">Rate</th>
						<th class="text-right" style="width: 6%;">Disc%</th>
						{#if isTaxInvoice}<th class="text-right" style="width: 6%;">GST%</th>{/if}
						<th class="text-right" style="width: 10%;">Amount</th>
					</tr>
				</thead>
				<tbody>
					{#if invoice?.items}
						{#each invoice.items as it, i}
							<tr>
								<td class="text-center">{i + 1}</td>
								<td>
									<strong>{it.item_description}</strong>
									{#if it.rack_location}<div class="rx-rack">Rack: {it.rack_location}</div>{/if}
								</td>
								<td class="text-center font-mono">{it.hsn_sac || '3004'}</td>
								<td class="text-center font-mono">{it.batch_number || '-'}</td>
								<td class="text-center font-mono" class:rx-short-exp={isShortExpiry(it.expiry_date)}>{formatExpiry(it.expiry_date)}</td>
								<td class="text-right font-mono">{it.mrp && parseFloat(it.mrp) > 0 ? formatCurrency(it.mrp) : '-'}</td>
								<td class="text-right font-mono">{parseFloat(it.quantity).toFixed(parseFloat(it.quantity) % 1 === 0 ? 0 : 2)}</td>
								<td class="text-right font-mono">{formatCurrency(it.unit_price)}</td>
								<td class="text-right font-mono">{parseFloat(it.discount_rate || '0') > 0 ? parseFloat(it.discount_rate).toFixed(1) : '-'}</td>
								{#if isTaxInvoice}<td class="text-right font-mono">{parseFloat(it.tax_rate || '0').toFixed(0)}%</td>{/if}
								<td class="text-right font-mono" style="font-weight: 700;">{formatCurrency(it.total_amount)}</td>
							</tr>
						{/each}
					{:else}
						<tr><td colspan="11" class="text-center" style="padding: 18px; color: #94a3b8;">No items.</td></tr>
					{/if}
				</tbody>
			</table>

			{#if hasShortExpiry}
				<div class="rx-warn">⚠ Items marked in red are within 90 days of expiry. Please use before the printed date.</div>
			{/if}

			<div class="rx-bottom">
				<div style="flex: 1; min-width: 240px; font-size: 11px;">
					{#if isTaxInvoice && taxSlabs.length > 0 && showTaxSummary}
						<table class="rx-slab">
							<thead>
								<tr><th>GST</th><th class="text-right">Taxable</th><th class="text-right">{invoice?.is_inter_state ? 'IGST' : 'CGST'}</th>{#if !invoice?.is_inter_state}<th class="text-right">SGST</th>{/if}</tr>
							</thead>
							<tbody>
								{#each taxSlabs as s}
									<tr>
										<td>{s.rate.toFixed(0)}%</td>
										<td class="text-right font-mono">{formatCurrency(s.taxable)}</td>
										<td class="text-right font-mono">{formatCurrency(invoice?.is_inter_state ? s.igst : s.cgst)}</td>
										{#if !invoice?.is_inter_state}<td class="text-right font-mono">{formatCurrency(s.sgst)}</td>{/if}
									</tr>
								{/each}
							</tbody>
						</table>
					{/if}
					{#if mrpSavings > 0}
						<div class="rx-saving">You saved <strong class="font-mono">{formatCurrency(mrpSavings)}</strong> on MRP</div>
					{/if}
					{#if showUPI && effectiveUPI}
						<div style="display: flex; gap: 10px; align-items: center; margin-top: 8px;">
							<img class="tpl-upi-qr-img" src={getUPIQRUrl(effectiveUPI, company?.legal_name || company?.name || '', invoice?.balance_due || invoice?.grand_total)} alt="UPI QR" />
							<div style="font-size: 10.5px;"><strong>Scan & Pay</strong><br /><span class="font-mono">{effectiveUPI}</span></div>
						</div>
					{/if}
					{#if invoice?.notes}<div style="margin-top: 8px;"><strong>Remarks:</strong> {invoice.notes}</div>{/if}
				</div>

				<div class="rx-totals">
					<div class="rx-trow"><span>Gross Value</span><span class="font-mono">{formatCurrency(invoice?.subtotal)}</span></div>
					{#if invoice && parseFloat(invoice.discount_total) > 0}
						<div class="rx-trow" style="color: #dc2626;"><span>Discount</span><span class="font-mono">- {formatCurrency(invoice.discount_total)}</span></div>
					{/if}
					{#if isTaxInvoice}
						<div class="rx-trow"><span>Taxable Value</span><span class="font-mono">{formatCurrency(invoice?.taxable_amount)}</span></div>
						{#if invoice?.is_inter_state}
							<div class="rx-trow"><span>IGST</span><span class="font-mono">{formatCurrency(invoice?.igst_total)}</span></div>
						{:else}
							<div class="rx-trow"><span>CGST</span><span class="font-mono">{formatCurrency(invoice?.cgst_total)}</span></div>
							<div class="rx-trow"><span>SGST</span><span class="font-mono">{formatCurrency(invoice?.sgst_total)}</span></div>
						{/if}
					{/if}
					{#if invoice && parseFloat(invoice.round_off) !== 0}
						<div class="rx-trow" style="color: #64748b; font-size: 11px;"><span>Round Off</span><span class="font-mono">{fmtSigned(invoice.round_off)}</span></div>
					{/if}
					<div class="rx-trow rx-grand"><span>NET AMOUNT</span><span class="font-mono">{formatCurrency(invoice?.grand_total)}</span></div>
					{#if invoice && parseFloat(invoice.amount_paid) > 0}
						<div class="rx-trow" style="color: #047857;"><span>Paid</span><span class="font-mono">{formatCurrency(invoice.amount_paid)}</span></div>
						{#if parseFloat(invoice.balance_due) > 0}
							<div class="rx-trow" style="color: #b45309; font-weight: 800;"><span>Balance Due</span><span class="font-mono">{formatCurrency(invoice.balance_due)}</span></div>
						{/if}
					{/if}
					{#if invoice?.tendered_cash && parseFloat(invoice.tendered_cash) > 0}
						<div class="rx-trow" style="font-size: 11px; color: #475569;"><span>Cash Tendered</span><span class="font-mono">{formatCurrency(invoice.tendered_cash)}</span></div>
						<div class="rx-trow" style="font-size: 11px; color: #475569;"><span>Change Returned</span><span class="font-mono">{formatCurrency(invoice.change_amount)}</span></div>
					{/if}
				</div>
			</div>

			<div style="font-size: 10.5px; color: #475569; margin-top: 8px;"><strong>In words:</strong> {grandTotalWords}</div>

			<div class="rx-footer">
				<div class="rx-terms">
					<div><strong>Note:</strong> Schedule H / H1 / X drugs are sold only against the prescription of a Registered Medical Practitioner. Please check batch and expiry before leaving the counter.</div>
					<div>{company?.default_return_policy || 'Medicines once sold will not be taken back or exchanged. Cold-chain and opened items are non-returnable.'}</div>
					<div>Subject to {effectiveJurisdiction} jurisdiction. Computer-generated bill.</div>
				</div>
				<div class="rx-sign">
					{#if effectiveSignatureUrl}<img src={effectiveSignatureUrl} alt="Sign" class="tpl-signature-img" />{:else}<div style="height: 30px;"></div>{/if}
					<div class="rx-sign-line">Registered Pharmacist<br /><span style="font-weight: 500; font-size: 9.5px;">For {company?.legal_name || company?.name}</span></div>
				</div>
			</div>
		</div>

	<!-- ====================================================================
	     TEMPLATE 15: BAKERY / SWEET SHOP BILL
	     FSSAI licence header, order / pickup details, veg / non-veg marks, best-before per item,
	     per-rate GST split, advance & balance for cake orders.
	     Field mapping: Order / Token = counter_number, Staff = cashier_name,
	     Cake message / Order ref = project_name, Pickup / Delivery on = due_date,
	     Pickup / Delivery at = delivery_place, Best before = item expiry_date.
	     ==================================================================== -->
	{:else if templateId === '15'}
		<div class="tpl-bakery" style="position: relative; z-index: 1;">
			<div class="bk-head">
				<span class={`copy-badge ${copyMeta.class}`}>{copyMeta.title}</span>
				<div class="bk-emblem">🧁</div>
				<div class="bk-name">{company?.legal_name || company?.name}</div>
				<div class="bk-sub">{effectiveSubtitle || 'Fresh Bakery · Cakes · Pastries · Snacks'}</div>
				<div class="bk-addr">{companyAddressLine}</div>
				<div class="bk-addr">
					{#if company?.phone}Ph: {company.phone}{/if}
					{#if isTaxInvoice && company?.gstin} &nbsp;·&nbsp; GSTIN: {company.gstin}{/if}
				</div>
				{#if company?.fssai_no}
					<div class="bk-fssai"><span class="bk-fssai-mark">FSSAI</span> Lic. No: <strong class="font-mono">{company.fssai_no}</strong></div>
				{/if}
				<div class="bk-title">{isTaxInvoice ? 'TAX INVOICE' : 'BILL OF SUPPLY'}</div>
			</div>

			<div class="bk-meta">
				<div><span>Bill No</span><strong class="font-mono">{invoice?.invoice_number}</strong></div>
				<div><span>Date</span><strong>{formatDate(invoice?.invoice_date)}</strong></div>
				<div><span>Order / Token</span><strong>{invoice?.counter_number || extraDetails?.counterNumber || '-'}</strong></div>
				<div><span>Served By</span><strong>{invoice?.cashier_name || extraDetails?.cashierName || '-'}</strong></div>
				<div><span>Payment</span><strong>{effectiveTerms}</strong></div>
				<div><span>Customer</span><strong>{invoice?.customer_name || 'Walk-in'}</strong></div>
			</div>

			{#if bakeryIsOrder}
				<div class="bk-order">
					<div class="bk-order-title">🎂 Order details</div>
					<div class="bk-order-grid">
						{#if bakeryDueLater}<div><span>Pickup / Delivery on</span><strong>{formatDate(invoice?.due_date)}</strong></div>{/if}
						{#if bakeryPlace}<div><span>Pickup / Delivery at</span><strong>{bakeryPlace}</strong></div>{/if}
						{#if invoice?.project_name}<div style="grid-column: 1 / -1;"><span>Message on cake / Order ref</span><strong>{invoice.project_name}</strong></div>{/if}
					</div>
				</div>
			{/if}

			<table class="bk-table">
				<thead>
					<tr>
						<th style="width: 4%;">#</th>
						<th style="width: {isTaxInvoice ? '40%' : '48%'};">Item</th>
						{#if isTaxInvoice}<th class="text-center" style="width: 9%;">HSN</th>{/if}
						<th class="text-right" style="width: 10%;">Qty</th>
						<th class="text-right" style="width: 13%;">Rate</th>
						{#if isTaxInvoice}<th class="text-right" style="width: 7%;">GST</th>{/if}
						<th class="text-right" style="width: 15%;">Amount</th>
					</tr>
				</thead>
				<tbody>
					{#if invoice?.items}
						{#each invoice.items as it, i}
							<tr>
								<td class="text-center">{i + 1}</td>
								<td>
									<span class={`bk-dot ${isNonVegItem(it.item_description) ? 'bk-nonveg' : 'bk-veg'}`} title={isNonVegItem(it.item_description) ? 'Non-vegetarian' : 'Vegetarian'}></span>
									<strong>{it.item_description}</strong>
									{#if it.expiry_date}<div class="bk-bb">Best before {formatExpiry(it.expiry_date)}{#if it.batch_number}&nbsp;· Batch {it.batch_number}{/if}</div>{/if}
								</td>
								{#if isTaxInvoice}<td class="text-center font-mono">{it.hsn_sac || '1905'}</td>{/if}
								<td class="text-right font-mono">{fmtQty(it.quantity)} {it.unit_code}</td>
								<td class="text-right font-mono">{formatCurrency(it.unit_price)}</td>
								{#if isTaxInvoice}<td class="text-right font-mono">{parseFloat(it.tax_rate || '0').toFixed(0)}%</td>{/if}
								<td class="text-right font-mono" style="font-weight: 700;">{formatCurrency(isTaxInvoice ? it.taxable_value : it.total_amount)}</td>
							</tr>
						{/each}
					{:else}
						<tr><td colspan="7" class="text-center" style="padding: 18px; color: #94a3b8;">No items.</td></tr>
					{/if}
				</tbody>
			</table>

			<div class="bk-bottom">
				<div class="bk-left">
					{#if isTaxInvoice && taxSlabs.length > 0 && showTaxSummary}
						<table class="bk-slab">
							<thead>
								<tr><th>GST</th><th class="text-right">Taxable</th><th class="text-right">{invoice?.is_inter_state ? 'IGST' : 'CGST'}</th>{#if !invoice?.is_inter_state}<th class="text-right">SGST</th>{/if}</tr>
							</thead>
							<tbody>
								{#each taxSlabs as s}
									<tr>
										<td>{s.rate.toFixed(0)}%</td>
										<td class="text-right font-mono">{formatCurrency(s.taxable)}</td>
										<td class="text-right font-mono">{formatCurrency(invoice?.is_inter_state ? s.igst : s.cgst)}</td>
										{#if !invoice?.is_inter_state}<td class="text-right font-mono">{formatCurrency(s.sgst)}</td>{/if}
									</tr>
								{/each}
							</tbody>
						</table>
					{/if}
					{#if showUPI && effectiveUPI}
						<div class="bk-upi">
							<img class="tpl-upi-qr-img" src={getUPIQRUrl(effectiveUPI, company?.legal_name || company?.name || '', invoice?.balance_due || invoice?.grand_total)} alt="UPI QR" />
							<div><strong>Scan & Pay</strong><br /><span class="font-mono">{effectiveUPI}</span></div>
						</div>
					{/if}
					{#if invoice?.notes}<div class="bk-notes"><strong>Note:</strong> {invoice.notes}</div>{/if}
				</div>

				<div class="bk-totals">
					<div class="bk-trow"><span>Sub Total</span><span class="font-mono">{formatCurrency(invoice?.subtotal)}</span></div>
					{#if invoice && parseFloat(invoice.discount_total) > 0}
						<div class="bk-trow" style="color: #dc2626;"><span>Discount</span><span class="font-mono">- {formatCurrency(invoice.discount_total)}</span></div>
					{/if}
					{#if isTaxInvoice}
						{#if invoice?.is_inter_state}
							{#each taxSlabs.filter((x) => x.rate > 0) as s}
								<div class="bk-trow"><span>IGST @ {s.rate.toFixed(1)}%</span><span class="font-mono">{formatCurrency(s.igst)}</span></div>
							{/each}
						{:else}
							{#each taxSlabs.filter((x) => x.rate > 0) as s}
								<div class="bk-trow"><span>CGST @ {(s.rate / 2).toFixed(1)}%</span><span class="font-mono">{formatCurrency(s.cgst)}</span></div>
								<div class="bk-trow"><span>SGST @ {(s.rate / 2).toFixed(1)}%</span><span class="font-mono">{formatCurrency(s.sgst)}</span></div>
							{/each}
						{/if}
					{/if}
					{#if invoice && parseFloat(invoice.round_off) !== 0}
						<div class="bk-trow" style="color: #78716c;"><span>Round Off</span><span class="font-mono">{fmtSigned(invoice.round_off)}</span></div>
					{/if}
					<div class="bk-trow bk-grand"><span>TOTAL</span><span class="font-mono">{formatCurrency(invoice?.grand_total)}</span></div>
					{#if invoice && parseFloat(invoice.amount_paid) > 0}
						<div class="bk-trow" style="color: #047857;"><span>{parseFloat(invoice.balance_due) > 0 ? 'Advance Paid' : 'Paid'}</span><span class="font-mono">{formatCurrency(invoice.amount_paid)}</span></div>
						{#if parseFloat(invoice.balance_due) > 0}
							<div class="bk-trow" style="color: #b45309; font-weight: 800;"><span>Balance to Pay</span><span class="font-mono">{formatCurrency(invoice.balance_due)}</span></div>
						{/if}
					{/if}
				</div>
			</div>

			<div class="bk-words">{grandTotalWords}</div>

			<div class="bk-foot">
				<div class="bk-care">{company?.template_options?.bakery_note || 'Freshly baked. Best consumed within 24 hours. Store cakes refrigerated.'}</div>
				<div class="bk-thanks">{company?.template_options?.bakery_footer || 'Thank you! Sweet moments, baked fresh.'}</div>
				<div class="bk-small">{effectiveReturnPolicy} · Subject to {effectiveJurisdiction} jurisdiction · Computer-generated bill</div>
			</div>
		</div>

	<!-- ====================================================================
	     TEMPLATE 16: GROCERY / SUPERMARKET BILL
	     MRP vs price per line, total savings, weighed quantities (kg / g / L), item & quantity
	     count, full GST summary (exempt staples at 0% stay visible), cash tendered / change.
	     Field mapping: Counter = counter_number, Cashier = cashier_name,
	     Customer / Loyalty ref = project_name, MRP = item mrp, Tendered / Change = POS fields.
	     ==================================================================== -->
	{:else if templateId === '16'}
		<div class="tpl-grocery" style="position: relative; z-index: 1;">
			<div class="gr-head">
				<div>
					<div class="gr-name">{company?.legal_name || company?.name}</div>
					<div class="gr-sub">{effectiveSubtitle || 'Supermarket · Grocery · Fresh Produce'}</div>
					<div class="gr-addr">{companyAddressLine}</div>
					<div class="gr-addr">
						{#if company?.phone}Ph: {company.phone}{/if}
						{#if isTaxInvoice && company?.gstin} &nbsp;·&nbsp; GSTIN: {company.gstin}{/if}
						{#if company?.fssai_no} &nbsp;·&nbsp; FSSAI: {company.fssai_no}{/if}
					</div>
				</div>
				<div class="gr-docbox">
					<span class={`copy-badge ${copyMeta.class}`}>{copyMeta.title}</span>
					<div class="gr-doctitle">{isTaxInvoice ? 'TAX INVOICE' : 'BILL OF SUPPLY'}</div>
					<div class="gr-docno font-mono">{invoice?.invoice_number}</div>
					<div class="gr-docdate">{formatDate(invoice?.invoice_date)}</div>
				</div>
			</div>

			<div class="gr-meta">
				<div><span>Customer</span><strong>{invoice?.customer_name || 'Walk-in'}</strong></div>
				<div><span>Counter</span><strong>{invoice?.counter_number || extraDetails?.counterNumber || '-'}</strong></div>
				<div><span>Cashier</span><strong>{invoice?.cashier_name || extraDetails?.cashierName || '-'}</strong></div>
				<div><span>Payment</span><strong>{effectiveTerms}</strong></div>
				{#if invoice?.project_name}<div><span>Loyalty / Ref</span><strong>{invoice.project_name}</strong></div>{/if}
				{#if isTaxInvoice && invoice?.customer_gstin}<div><span>Customer GSTIN</span><strong class="font-mono">{invoice.customer_gstin}</strong></div>{/if}
			</div>

			<table class="gr-table">
				<thead>
					<tr>
						<th style="width: 4%;">#</th>
						<th style="width: {isTaxInvoice ? '33%' : '40%'};">Item</th>
						{#if hasMrp}<th class="text-right" style="width: 10%;">MRP</th>{/if}
						<th class="text-right" style="width: 11%;">Qty</th>
						<th class="text-right" style="width: 11%;">Rate</th>
						{#if isTaxInvoice}<th class="text-right" style="width: 7%;">GST</th>{/if}
						<th class="text-right" style="width: 14%;">Amount</th>
					</tr>
				</thead>
				<tbody>
					{#if invoice?.items}
						{#each invoice.items as it, i}
							<tr>
								<td class="text-center">{i + 1}</td>
								<td>
									<strong>{it.item_description}</strong>
									{#if isTaxInvoice && it.hsn_sac}<span class="gr-hsn"> HSN {it.hsn_sac}</span>{/if}
								</td>
								{#if hasMrp}<td class="text-right font-mono gr-mrp">{it.mrp && parseFloat(it.mrp) > 0 ? fmtNum(it.mrp) : ''}</td>{/if}
								<td class="text-right font-mono">{fmtQty(it.quantity)} {it.unit_code}</td>
								<td class="text-right font-mono">{fmtNum(it.unit_price)}</td>
								{#if isTaxInvoice}<td class="text-right font-mono">{parseFloat(it.tax_rate || '0') === 0 ? 'Nil' : parseFloat(it.tax_rate).toFixed(0) + '%'}</td>{/if}
								<td class="text-right font-mono" style="font-weight: 700;">{fmtNum(it.total_amount)}</td>
							</tr>
						{/each}
					{:else}
						<tr><td colspan="7" class="text-center" style="padding: 18px; color: #94a3b8;">No items.</td></tr>
					{/if}
				</tbody>
			</table>

			<div class="gr-count">
				<span>Items: <strong>{invoice?.items?.length || 0}</strong></span>
				<span>Total Qty: <strong>{fmtQty(String(invoice?.items?.reduce((a, it) => a + (parseFloat(it.quantity) || 0), 0) || 0))}</strong></span>
				{#if mrpSavings > 0}<span class="gr-saved">★ You saved <strong class="font-mono">{formatCurrency(mrpSavings)}</strong> on MRP</span>{/if}
			</div>

			<div class="gr-bottom">
				<div class="gr-left">
					{#if isTaxInvoice && taxSlabs.length > 0 && showTaxSummary}
						<div class="gr-gstlabel">GST Summary</div>
						<table class="gr-slab">
							<thead>
								<tr><th>Rate</th><th class="text-right">Taxable</th><th class="text-right">{invoice?.is_inter_state ? 'IGST' : 'CGST'}</th>{#if !invoice?.is_inter_state}<th class="text-right">SGST</th>{/if}</tr>
							</thead>
							<tbody>
								{#each taxSlabs as s}
									<tr>
										<td>{s.rate === 0 ? 'Exempt / Nil' : s.rate.toFixed(0) + '%'}</td>
										<td class="text-right font-mono">{fmtNum(s.taxable)}</td>
										<td class="text-right font-mono">{fmtNum(invoice?.is_inter_state ? s.igst : s.cgst)}</td>
										{#if !invoice?.is_inter_state}<td class="text-right font-mono">{fmtNum(s.sgst)}</td>{/if}
									</tr>
								{/each}
							</tbody>
						</table>
					{/if}
					{#if showUPI && effectiveUPI}
						<div class="gr-upi">
							<img class="tpl-upi-qr-img" src={getUPIQRUrl(effectiveUPI, company?.legal_name || company?.name || '', invoice?.balance_due || invoice?.grand_total)} alt="UPI QR" />
							<div><strong>Scan & Pay</strong><br /><span class="font-mono">{effectiveUPI}</span></div>
						</div>
					{/if}
					{#if invoice?.notes}<div class="gr-notes">{invoice.notes}</div>{/if}
				</div>

				<div class="gr-totals">
					<div class="gr-trow"><span>Sub Total</span><span class="font-mono">{formatCurrency(invoice?.subtotal)}</span></div>
					{#if invoice && parseFloat(invoice.discount_total) > 0}
						<div class="gr-trow" style="color: #dc2626;"><span>Discount</span><span class="font-mono">- {formatCurrency(invoice.discount_total)}</span></div>
					{/if}
					{#if isTaxInvoice}
						{#if invoice?.is_inter_state}
							<div class="gr-trow"><span>IGST</span><span class="font-mono">{formatCurrency(invoice?.igst_total)}</span></div>
						{:else}
							<div class="gr-trow"><span>CGST</span><span class="font-mono">{formatCurrency(invoice?.cgst_total)}</span></div>
							<div class="gr-trow"><span>SGST</span><span class="font-mono">{formatCurrency(invoice?.sgst_total)}</span></div>
						{/if}
					{/if}
					{#if invoice && parseFloat(invoice.round_off) !== 0}
						<div class="gr-trow" style="color: #64748b;"><span>Round Off</span><span class="font-mono">{fmtSigned(invoice.round_off)}</span></div>
					{/if}
					<div class="gr-trow gr-grand"><span>NET PAYABLE</span><span class="font-mono">{formatCurrency(invoice?.grand_total)}</span></div>
					{#if invoice?.tendered_cash && parseFloat(invoice.tendered_cash) > 0}
						<div class="gr-trow" style="font-size: 11px; color: #475569;"><span>Cash Tendered</span><span class="font-mono">{formatCurrency(invoice.tendered_cash)}</span></div>
						<div class="gr-trow" style="font-size: 11px; color: #475569;"><span>Change Returned</span><span class="font-mono">{formatCurrency(invoice.change_amount)}</span></div>
					{:else if invoice && parseFloat(invoice.amount_paid) > 0}
						<div class="gr-trow" style="color: #047857;"><span>Paid</span><span class="font-mono">{formatCurrency(invoice.amount_paid)}</span></div>
						{#if parseFloat(invoice.balance_due) > 0}
							<div class="gr-trow" style="color: #b45309; font-weight: 800;"><span>Balance Due</span><span class="font-mono">{formatCurrency(invoice.balance_due)}</span></div>
						{/if}
					{/if}
				</div>
			</div>

			<div class="gr-words">{grandTotalWords}</div>

			<div class="gr-foot">
				<div>{company?.template_options?.grocery_note || 'Please check your bill and items before leaving the counter. Fresh produce and loose items are non-returnable.'}</div>
				<div class="gr-thanks">{company?.template_options?.grocery_footer || 'Thank you for shopping with us! Visit again.'}</div>
				<div class="gr-small">{effectiveReturnPolicy} · Subject to {effectiveJurisdiction} jurisdiction · Computer-generated bill</div>
			</div>
		</div>

	<!-- ====================================================================
	     TEMPLATE 17: HARDWARE / BUILDING MATERIALS BILL
	     Trade-discount column, delivery / site block, transport details, HSN-wise tax summary,
	     credit terms with due date, receiver's signature.
	     Field mapping: Site / Project = project_name, PO / Work order = po_number,
	     Delivery address = shipping_address (else delivery_place), Vehicle / E-way / Transporter /
	     LR / weights = the transport fields, Credit due = due_date when a balance is outstanding.
	     ==================================================================== -->
	{:else if templateId === '17'}
		<div class="tpl-hardware" style="position: relative; z-index: 1;">
			<div class="hw-header">
				<div>
					<div class="hw-name">{company?.legal_name || company?.name}</div>
					<div class="hw-sub">{effectiveSubtitle || 'Hardware · Building Materials · Paints · Plumbing · Electricals'}</div>
					<div class="hw-addr">{companyAddressLine}</div>
					<div class="hw-addr">
						{#if company?.phone}Ph: {company.phone}{/if}{#if company?.email} &nbsp;|&nbsp; {company.email}{/if}
						{#if isTaxInvoice && company?.gstin} &nbsp;|&nbsp; GSTIN: {company.gstin}{/if}
					</div>
				</div>
				<div class="hw-docbox">
					<span class={`copy-badge ${copyMeta.class}`}>{copyMeta.title}</span>
					<div class="hw-doctitle">{isTaxInvoice ? 'TAX INVOICE' : 'BILL OF SUPPLY'}</div>
					<div class="hw-docno font-mono">{invoice?.invoice_number}</div>
					<div class="hw-docdate">Date: {formatDate(invoice?.invoice_date)}</div>
				</div>
			</div>
			<div class="hw-stripe"></div>

			<div class="hw-parties">
				<div class="hw-cell">
					<span class="hw-label">Billed To</span>
					<strong>{invoice?.customer_name || 'Cash Customer'}</strong>
					{#if invoice?.billing_address}<div class="hw-muted">{invoice.billing_address}</div>{/if}
					{#if isTaxInvoice && invoice?.customer_gstin}<div class="hw-muted font-mono">GSTIN: {invoice.customer_gstin}</div>{/if}
					{#if invoice?.supply_state}<div class="hw-muted">State: {invoice.supply_state}</div>{/if}
				</div>
				<div class="hw-cell">
					<span class="hw-label">Delivery / Site</span>
					<strong>{hardwareSite || 'Counter sale'}</strong>
					{#if invoice?.project_name}<div class="hw-muted">Project: {invoice.project_name}</div>{/if}
				</div>
				<div class="hw-cell">
					<span class="hw-label">Terms</span>
					<strong>{effectiveTerms}</strong>
					{#if invoice?.po_number || extraDetails?.poNumber}<div class="hw-muted">PO / Work Order: <span class="font-mono">{invoice?.po_number || extraDetails?.poNumber}</span></div>{/if}
					{#if creditDue}<div class="hw-credit">CREDIT · Due {formatDate(invoice?.due_date)}</div>{/if}
				</div>
			</div>

			{#if hardwareHasTransport}
				<div class="hw-transport">
					{#if invoice?.vehicle_number}<span>Vehicle: <strong class="font-mono">{invoice.vehicle_number}</strong></span>{/if}
					{#if invoice?.transporter_name}<span>Transporter: <strong>{invoice.transporter_name}</strong></span>{/if}
					{#if invoice?.lr_number}<span>LR No: <strong class="font-mono">{invoice.lr_number}</strong></span>{/if}
					{#if invoice?.eway_bill}<span>E-Way Bill: <strong class="font-mono">{invoice.eway_bill}</strong></span>{/if}
					{#if invoice?.gross_weight}<span>Gross Wt: <strong>{invoice.gross_weight}</strong></span>{/if}
					{#if invoice?.net_weight}<span>Net Wt: <strong>{invoice.net_weight}</strong></span>{/if}
				</div>
			{/if}

			<table class="hw-table">
				<thead>
					<tr>
						<th style="width: 4%;">#</th>
						<th style="width: {isTaxInvoice ? '31%' : '38%'};">Description of Goods</th>
						{#if isTaxInvoice}<th class="text-center" style="width: 8%;">HSN</th>{/if}
						<th class="text-right" style="width: 11%;">Qty</th>
						<th class="text-right" style="width: 10%;">Rate</th>
						{#if hasDisc}<th class="text-right" style="width: 7%;">Disc %</th>{/if}
						{#if isTaxInvoice}<th class="text-right" style="width: 8%;">GST</th>{/if}
						<th class="text-right" style="width: 14%;">Amount</th>
					</tr>
				</thead>
				<tbody>
					{#if invoice?.items}
						{#each invoice.items as it, i}
							<tr>
								<td class="text-center">{i + 1}</td>
								<td><strong>{it.item_description}</strong></td>
								{#if isTaxInvoice}<td class="text-center font-mono">{it.hsn_sac || '-'}</td>{/if}
								<td class="text-right font-mono">{fmtQty(it.quantity)} {it.unit_code}</td>
								<td class="text-right font-mono">{fmtNum(it.unit_price)}</td>
								{#if hasDisc}<td class="text-right font-mono">{parseFloat(it.discount_rate || '0') > 0 ? parseFloat(it.discount_rate).toFixed(1) : '-'}</td>{/if}
								{#if isTaxInvoice}<td class="text-right font-mono">{parseFloat(it.tax_rate || '0').toFixed(0)}%</td>{/if}
								<td class="text-right font-mono" style="font-weight: 700;">{fmtNum(isTaxInvoice ? it.taxable_value : it.total_amount)}</td>
							</tr>
						{/each}
					{:else}
						<tr><td colspan="8" class="text-center" style="padding: 18px; color: #94a3b8;">No items.</td></tr>
					{/if}
				</tbody>
			</table>

			<div class="hw-bottom">
				<div class="hw-left">
					{#if isTaxInvoice && hsnSummary.length > 0 && showTaxSummary}
						<div class="hw-gstlabel">HSN-wise Tax Summary</div>
						<table class="hw-slab">
							<thead>
								<tr><th>HSN</th><th class="text-right">Taxable</th><th class="text-right">Rate</th><th class="text-right">{invoice?.is_inter_state ? 'IGST' : 'CGST'}</th>{#if !invoice?.is_inter_state}<th class="text-right">SGST</th>{/if}</tr>
							</thead>
							<tbody>
								{#each hsnSummary as h}
									<tr>
										<td class="font-mono">{h.hsn}</td>
										<td class="text-right font-mono">{fmtNum(h.taxable)}</td>
										<td class="text-right">{h.rate.toFixed(0)}%</td>
										<td class="text-right font-mono">{fmtNum(invoice?.is_inter_state ? h.igst : h.cgst)}</td>
										{#if !invoice?.is_inter_state}<td class="text-right font-mono">{fmtNum(h.sgst)}</td>{/if}
									</tr>
								{/each}
							</tbody>
						</table>
					{/if}
					{#if showBank && (effectiveBankName || effectiveAccountNo)}
						<div class="tpl-bank-box" style="margin-top: 8px;">
							<div class="tpl-bank-title">Bank Details (NEFT / RTGS)</div>
							<div><strong>{effectiveBankName}</strong> {#if effectiveBranch}({effectiveBranch}){/if} · A/C <span class="font-mono">{effectiveAccountNo}</span> · IFSC <span class="font-mono">{effectiveIFSC}</span></div>
						</div>
					{/if}
					{#if showUPI && effectiveUPI}
						<div class="hw-upi">
							<img class="tpl-upi-qr-img" src={getUPIQRUrl(effectiveUPI, company?.legal_name || company?.name || '', invoice?.balance_due || invoice?.grand_total)} alt="UPI QR" />
							<div><strong>Pay by UPI</strong><br /><span class="font-mono">{effectiveUPI}</span></div>
						</div>
					{/if}
					{#if invoice?.notes}<div class="hw-notes"><strong>Remarks:</strong> {invoice.notes}</div>{/if}
				</div>

				<div class="hw-totals">
					<div class="hw-trow"><span>Gross Amount</span><span class="font-mono">{formatCurrency(invoice?.subtotal)}</span></div>
					{#if invoice && parseFloat(invoice.discount_total) > 0}
						<div class="hw-trow" style="color: #dc2626;"><span>Trade Discount</span><span class="font-mono">- {formatCurrency(invoice.discount_total)}</span></div>
					{/if}
					{#if isTaxInvoice}
						<div class="hw-trow"><span>Taxable Value</span><span class="font-mono">{formatCurrency(invoice?.taxable_amount)}</span></div>
						{#if invoice?.is_inter_state}
							<div class="hw-trow"><span>IGST</span><span class="font-mono">{formatCurrency(invoice?.igst_total)}</span></div>
						{:else}
							<div class="hw-trow"><span>CGST</span><span class="font-mono">{formatCurrency(invoice?.cgst_total)}</span></div>
							<div class="hw-trow"><span>SGST</span><span class="font-mono">{formatCurrency(invoice?.sgst_total)}</span></div>
						{/if}
					{/if}
					{#if invoice && parseFloat(invoice.round_off) !== 0}
						<div class="hw-trow" style="color: #64748b;"><span>Round Off</span><span class="font-mono">{fmtSigned(invoice.round_off)}</span></div>
					{/if}
					<div class="hw-trow hw-grand"><span>GRAND TOTAL</span><span class="font-mono">{formatCurrency(invoice?.grand_total)}</span></div>
					{@render payRows()}
				</div>
			</div>

			<div class="hw-words"><strong>Amount in words:</strong> {grandTotalWords}</div>

			<div class="hw-footer">
				<div class="hw-terms">
					<div><strong>Terms:</strong> {effectiveReturnPolicy}</div>
					{#if effectiveInterestRate}<div>Interest @ {effectiveInterestRate} p.a. will be charged on overdue payments. Goods are checked and accepted at delivery; breakage / shortage must be reported at the time of delivery.</div>{/if}
					<div>Subject to {effectiveJurisdiction} jurisdiction · Computer-generated bill</div>
				</div>
				<div class="hw-sign">
					<div class="hw-sign-line">Receiver's Signature</div>
				</div>
				<div class="hw-sign">
					{#if effectiveSignatureUrl}<img src={effectiveSignatureUrl} alt="Sign" class="tpl-signature-img" />{:else}<div style="height: 30px;"></div>{/if}
					<div class="hw-sign-line">{effectiveSignatoryTitle}<br /><span style="font-weight: 500; font-size: 9.5px;">For {company?.legal_name || company?.name}</span></div>
				</div>
			</div>
		</div>

	<!-- ====================================================================
	     TEMPLATE 11: INDUSTRIAL BLUEMETALS & TRANSPORT GST (HEAVY GRID)
	     Matches authentic Indian blue metals, quarry, transport & materials tax bill
	     ==================================================================== -->
	{:else if templateId === '11'}
		{@const compStateCode = getStateCode(company?.gstin, company?.state || 'Tamil Nadu')}
		{@const compPan = getPanNumber(company?.gstin)}
		{@const custStateCode = getStateCode(invoice?.customer_gstin, invoice?.supply_state || company?.state || 'Tamil Nadu')}

		<div class="tpl-bluemetals" style="position: relative; z-index: 1;">
			<div class="bm-outer-frame">
				<!-- Top Bar: Document Title & Copy Category -->
				<div class="bm-top-bar">
					<div style="flex: 1; text-align: center; font-size: 13px; font-weight: 800; letter-spacing: 0.05em; padding-left: 120px;">
						{isTaxInvoice ? 'TAX INVOICE' : 'BILL OF SUPPLY'}
					</div>
					<div style="font-size: 10px; font-weight: 700; text-transform: uppercase;">
						({copyMeta.title})
					</div>
				</div>

				<!-- Company Header Info -->
				<div class="bm-header-center">
					<div class="bm-company-name">
						{company?.legal_name || company?.name || 'COMPANY NAME'}
					</div>
					{#if effectiveSubtitle}
						<div style="font-size: 11px; font-weight: 700; color: #1e3a8a; margin-top: 1px;">
							{effectiveSubtitle}
						</div>
					{/if}
					<div class="bm-company-addr">
						{#if company?.address}{company.address}, {/if}
						{#if company?.city}{company.city}, {/if}
						{company?.state || ''} {#if company?.pincode}- {company.pincode}{/if}
					</div>
					<div class="bm-company-meta">
						{#if company?.phone}Mobile :- {company.phone} {/if}
						{#if company?.email}E-Mail : {company.email}{/if}
					</div>
					<div class="bm-company-meta" style="margin-top: 2px;">
						<strong>GSTIN :</strong> {company?.gstin || '-'} &nbsp;&nbsp;
						{#if compPan}<strong>PAN No :</strong> {compPan}{/if}
					</div>
					<div class="bm-company-meta" style="margin-top: 1px;">
						State : {company?.state || '-'} , Code : {compStateCode}
					</div>
					{#if company?.address}
						<div class="bm-company-addr" style="margin-top: 3px; font-size: 9.5px; opacity: 0.9;">
							H.O:- {company.address}
						</div>
					{/if}
				</div>

				<!-- 2x2 Logistics Matrix -->
				<!-- ROW 1: Billing Address vs Invoice Details -->
				<div class="bm-grid-row">
					<div class="bm-cell bm-cell-left">
						<div class="bm-heading">Billing Address Details :-</div>
						<div style="font-weight: 800; text-transform: uppercase;">{invoice?.customer_name || 'Cash Customer'}</div>
						<div>{invoice?.billing_address || '-'}</div>
						<div><strong>GSTIN :</strong> {invoice?.customer_gstin || 'Unregistered'}</div>
						<div>State : {invoice?.supply_state || company?.state || '-'} , Code : {custStateCode}</div>
					</div>

					<div class="bm-cell" style="padding: 0;">
						<div style="padding: 6px 10px;">
							<div style="display: flex; justify-content: space-between;">
								<span>Invoice No</span>
								<span>: <strong>{invoice?.invoice_number || '-'}</strong></span>
							</div>
							<div style="display: flex; justify-content: space-between; margin-top: 2px;">
								<span>Invoice Date</span>
								<span>: <strong>{formatDate(invoice?.invoice_date)}</strong></span>
							</div>
							<div style="display: flex; justify-content: space-between; margin-top: 2px;">
								<span>Terms</span>
								<span>: <strong>{effectiveTerms}</strong></span>
							</div>
						</div>
						<div class="bm-sub-split" style="padding: 5px 10px; margin-top: 0;">
							<div style="display: flex; justify-content: space-between;">
								<span>P.O. No.</span>
								<span>: <strong>{extraDetails?.poNumber || invoice?.po_number || '-'}</strong></span>
							</div>
							<div style="display: flex; justify-content: space-between; margin-top: 2px;">
								<span>Date</span>
								<span>: {formatDate(invoice?.invoice_date)}</span>
							</div>
						</div>
					</div>
				</div>

				<!-- ROW 2: Delivery Address vs Delivery Logistics -->
				<div class="bm-grid-row">
					<div class="bm-cell bm-cell-left">
						<div class="bm-heading">Delivery Address Details :-</div>
						<div style="font-weight: 800; text-transform: uppercase;">{invoice?.customer_name || 'Cash Customer'}</div>
						<div>{invoice?.shipping_address || invoice?.billing_address || '-'}</div>
						<div><strong>GSTIN :</strong> {invoice?.customer_gstin || 'Unregistered'}</div>
						<div>State : {invoice?.supply_state || company?.state || '-'} , Code : {custStateCode}</div>
					</div>

					<div class="bm-cell" style="padding: 0;">
						<div style="padding: 6px 10px;">
							<div style="display: flex; justify-content: space-between;">
								<span>Delivery Place</span>
								<span>: <strong>{extraDetails?.deliveryPlace || invoice?.delivery_place || invoice?.supply_state || '-'}</strong></span>
							</div>
							{#if effectiveGrossWeight || effectiveNetWeight}
								<div style="display: flex; justify-content: space-between; margin-top: 2px; font-size: 10px;">
									<span>Weight (Gross / Net)</span>
									<span>: <strong>{effectiveGrossWeight || '-'} / {effectiveNetWeight || '-'}</strong></span>
								</div>
							{/if}
						</div>
						<div class="bm-sub-split" style="padding: 5px 10px; margin-top: 0;">
							<div style="display: flex; justify-content: space-between;">
								<span>Vehicle No</span>
								<span>: <strong>{extraDetails?.vehicleNumber || invoice?.vehicle_number || '-'}</strong></span>
							</div>
							{#if extraDetails?.lrNumber || invoice?.lr_number}
								<div style="display: flex; justify-content: space-between; margin-top: 2px;">
									<span>LR / RR No</span>
									<span>: <strong>{extraDetails?.lrNumber || invoice?.lr_number}</strong></span>
								</div>
							{/if}
						</div>
					</div>
				</div>

				<!-- Line Items Table -->
				<table class="bm-table">
					<colgroup>
						<col style="width: 7%;" />
						<col style="width: 41%;" />
						<col style="width: 14%;" />
						<col style="width: 14%;" />
						<col style="width: 11%;" />
						<col style="width: 13%;" />
					</colgroup>
					<thead>
						<tr>
							<th style="text-align: center;">S.No.</th>
							<th>Product Name</th>
							<th style="text-align: center;">HSN Code</th>
							<th style="text-align: right;">Qty/Wt</th>
							<th style="text-align: right;">Rate</th>
							<th style="text-align: right;">Amount</th>
						</tr>
					</thead>
					<tbody>
						{#if invoice?.items && invoice.items.length > 0}
							{#each invoice.items as it, i}
								<tr>
									<td style="text-align: center;">{i + 1}</td>
									<td>
										<strong style="text-transform: uppercase;">{it.item_description}</strong>
									</td>
									<td style="text-align: center; font-family: monospace;">{it.hsn_sac || '-'}</td>
									<td style="text-align: right; font-family: monospace;">
										{parseFloat(it.quantity).toFixed(2)} {it.unit_code || ''}
									</td>
									<td style="text-align: right; font-family: monospace;">
										{parseFloat(it.unit_price).toFixed(2)}
									</td>
									<td style="text-align: right; font-family: monospace; font-weight: 700;">
										{parseFloat(it.taxable_value || (parseFloat(it.quantity) * parseFloat(it.unit_price)).toString()).toLocaleString('en-IN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
									</td>
								</tr>
							{/each}
						{:else if !invoice}
							<tr>
								<td style="text-align: center;">1</td>
								<td><strong>12 MM JALLY</strong></td>
								<td style="text-align: center; font-family: monospace;">25171010</td>
								<td style="text-align: right; font-family: monospace;">23.13 MT</td>
								<td style="text-align: right; font-family: monospace;">550.00</td>
								<td style="text-align: right; font-family: monospace; font-weight: 700;">12,721.50</td>
							</tr>
						{:else}
							<tr>
								<td colspan="6" style="text-align: center; padding: 1.5rem; color: #64748b; font-style: italic;">
									No items recorded on this invoice
								</td>
							</tr>
						{/if}
						<!-- Filler spacing rows to mimic industrial pre-printed continuous height if few items -->
						{#if !invoice?.items || invoice.items.length < 3}
							<tr>
								<td style="border-bottom: none; height: 35px;">&nbsp;</td>
								<td style="border-bottom: none;">&nbsp;</td>
								<td style="border-bottom: none;">&nbsp;</td>
								<td style="border-bottom: none;">&nbsp;</td>
								<td style="border-bottom: none;">&nbsp;</td>
								<td style="border-bottom: none;">&nbsp;</td>
							</tr>
						{/if}
					</tbody>
				</table>

				<!-- Sub-Totals & Tax Breakdown Section -->
				<div class="bm-tax-split">
					<div class="bm-eoe-box" style="display: flex; justify-content: space-between; align-items: flex-start; gap: 10px;">
						<div>
							<strong style="font-size: 11px;">E.& O.E</strong>
							{#if invoice?.notes}
								<div style="margin-top: 4px; font-size: 9.5px; opacity: 0.85; white-space: pre-wrap;">
									Note: {invoice.notes}
								</div>
							{/if}
							{#if showBank && (effectiveBankName || effectiveAccountNo)}
								<div style="font-size: 9.5px; color: #1e3a8a; margin-top: 6px; line-height: 1.4;">
									<strong>Bank:</strong> {effectiveBankName} {#if effectiveBranch}({effectiveBranch}){/if}<br />
									<strong>A/C:</strong> {effectiveAccountNo} • <strong>IFSC:</strong> {effectiveIFSC}
									{#if effectiveAccountHolder}<br /><strong>Holder:</strong> {effectiveAccountHolder}{/if}
								</div>
							{/if}
						</div>
						{#if showUPI && effectiveUPI}
							<div class="tpl-upi-qr-box" style="padding: 4px 6px; margin: 0;">
								<img class="tpl-upi-qr-img" style="width: 55px; height: 55px;" src={getUPIQRUrl(effectiveUPI, company?.name || '', invoice?.grand_total)} alt="UPI QR" />
								<span class="tpl-upi-text" style="font-size: 8px;">Scan UPI</span>
							</div>
						{/if}
					</div>

					<div class="bm-totals-box">
						{#if isTaxInvoice}
							<div class="bm-totals-row">
								<span>Taxable Value</span>
								<span class="font-mono">{fmtNum(invoice?.taxable_amount)}</span>
							</div>
							{#if invoice?.is_inter_state}
								{#each taxSlabs as sl}
									<div class="bm-totals-row">
										<span>IGST {sl.rate.toFixed(2)} % Amt</span>
										<span class="font-mono">{fmtNum(sl.igst)}</span>
									</div>
								{/each}
							{:else}
								{#each taxSlabs as sl}
									<div class="bm-totals-row">
										<span>CGST {(sl.rate / 2).toFixed(2)} % Amt</span>
										<span class="font-mono">{fmtNum(sl.cgst)}</span>
									</div>
									<div class="bm-totals-row">
										<span>SGST {(sl.rate / 2).toFixed(2)} % Amt</span>
										<span class="font-mono">{fmtNum(sl.sgst)}</span>
									</div>
								{/each}
							{/if}
						{/if}
						{#if invoice && parseFloat(invoice.round_off || '0') !== 0}
							<div class="bm-totals-row">
								<span>R.off</span>
								<span class="font-mono">{fmtNum(invoice.round_off)}</span>
							</div>
						{/if}
						<div class="bm-totals-row grand-net">
							<span>Net Amount</span>
							<span class="font-mono" style="font-size: 14px;">{fmtNum(invoice?.grand_total)}</span>
						</div>
						{#if invoice && parseFloat(invoice.amount_paid || '0') > 0}
							<div class="bm-totals-row"><span>Amount Paid</span><span class="font-mono">{fmtNum(invoice.amount_paid)}</span></div>
							<div class="bm-totals-row"><span><strong>Balance Due</strong></span><span class="font-mono"><strong>{fmtNum(invoice.balance_due)}</strong></span></div>
						{/if}
					</div>
				</div>

				<!-- Amount in Words Bar -->
				<div class="bm-words-bar">
					Rupees :- [ {grandTotalWords} ]
				</div>

				<!-- Terms & Signatory Section -->
				<div class="bm-footer-row">
					<div class="bm-terms-col">
						<div>* Subject to {effectiveJurisdiction} Jurisdiction.</div>
						<div>* Interest @ {effectiveInterestRate} will be charged on due date.</div>
						<div>* {effectiveReturnPolicy}</div>
					</div>

					<div class="bm-sign-col">
						<div style="font-weight: 800; font-size: 10.5px; text-transform: uppercase;">
							For {company?.legal_name || company?.name || 'Authorised Signatory'}
						</div>
						{#if effectiveSignatureUrl}
							<div style="text-align: right; margin-top: 4px;">
								<img src={effectiveSignatureUrl} alt="Sign" class="tpl-signature-img" style="max-height: 40px;" />
							</div>
						{:else}
							<div style="height: 38px;"></div>
						{/if}
						<div style="font-size: 10px; font-weight: 700; text-align: right;">
							{effectiveSignatoryTitle}
						</div>
					</div>
				</div>

				<!-- Computer Generated Disclaimer & Page Footer -->
				<div class="bm-computer-line">
					<span>*** This is Computer Generated Invoice ***</span>
					<span>Page 1 of 1</span>
				</div>
			</div>
		</div>
	{/if}
</div>
