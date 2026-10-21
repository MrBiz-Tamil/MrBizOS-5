<script lang="ts">
	import { onMount } from 'svelte';
	import { billingGrpc } from '$lib/grpc';
	import InvoiceTemplates from '$lib/components/InvoiceTemplates.svelte';
	import type { Tenant, Invoice as ApiInvoice } from '$lib/api';

	interface InvoiceItem {
		id?: string;
		productId: string;
		itemDescription: string;
		hsnSac: string;
		quantity: string;
		unitCode: string;
		unitPrice: string;
		discountRate: string;
		taxRate: string;
		taxableAmount?: string;
		cgstAmount?: string;
		sgstAmount?: string;
		igstAmount?: string;
		totalAmount?: string;
	}

	interface Invoice {
		id: string;
		tenantId: string;
		invoiceNumber: string;
		customerId: string;
		customerName: string;
		customerGstin?: string;
		invoiceDate: string;
		dueDate: string;
		status: string;
		subtotal: string;
		discountTotal: string;
		taxTotal: string;
		grandTotal: string;
		amountPaid: string;
		balanceDue: string;
		paymentMode: string;
		supplyState: string;
		isInterState: boolean;
		cgstTotal?: string;
		sgstTotal?: string;
		igstTotal?: string;
		templateId?: string;
		paperSize?: string;
		vehicleNumber?: string;
		deliveryPlace?: string;
		grossWeight?: string;
		netWeight?: string;
		poNumber?: string;
		lrNumber?: string;
		ewayBill?: string;
		items?: InvoiceItem[];
	}

	interface Receivable {
		customerId: string;
		customerName: string;
		phone: string;
		totalBilled: string;
		totalPaid: string;
		balanceDue: string;
		unpaidInvoicesCount: number;
	}

	interface BillingSummary {
		todaySales: string;
		monthSales: string;
		totalOutstanding: string;
		totalInvoicesCount: number;
		paidInvoicesCount: number;
		unpaidInvoicesCount: number;
		topReceivables: Receivable[];
	}

	interface TemplateInfo {
		id: string;
		name: string;
		category: string;
		desc: string;
		badge: string;
	}

	const templates: TemplateInfo[] = [
		{ id: '1', name: '1. Modern Corporate', category: 'Corporate B2B & Enterprise', desc: 'Contemporary indigo dual-tone layout with metadata pill cards, QR code, and structured GST breakdown.', badge: 'Popular' },
		{ id: '2', name: '2. Classic Traditional Boxed', category: 'Wholesale & Distribution', desc: 'Bordered cell outlines adhering strictly to standard Indian GST Rule 46 format with authorized signatory seal.', badge: 'GST Standard' },
		{ id: '3', name: '3. Minimalist Executive', category: 'Consulting & Legal', desc: 'High-end editorial typography, generous whitespace, and fine dividers for professional firms.', badge: 'Clean' },
		{ id: '4', name: '4. Retail Counter POS', category: 'Retail Stores & Supermarkets', desc: 'Space-saving counter receipt format with MRP, savings callouts, and cashier receipt headers.', badge: 'Retail' },
		{ id: '5', name: '5. Thermal 80mm Receipt', category: 'POS Thermal Roll Printers', desc: 'Continuous 3-inch/80mm roll paper layout with dashed dividers and monospace print formatting.', badge: '80mm POS' },
		{ id: '6', name: '6. Creative Studio & Agency', category: 'Agencies & Freelancers', desc: 'Vibrant accent colors with scope-of-work deliverable breakdowns for digital agencies.', badge: 'Creative' },
		{ id: '7', name: '7. Industrial & Dispatch B2B', category: 'Manufacturing & Logistics', desc: 'Dedicated transport fields: PO Number, Vehicle #, E-Way Bill #, Transporter Name, and Delivery Challan.', badge: 'Logistics' },
		{ id: '8', name: '8. Bold Tech SaaS', category: 'Cloud Subscriptions & IT', desc: 'Modern dark gradient banner with plan tiers, billing cycles, and software subscription terms.', badge: 'SaaS' },
		{ id: '9', name: '9. Wholesale Multi-Tax Slab', category: 'Trading & Multi-Tax Traders', desc: 'Includes an exhaustive HSN/SAC summary table dividing CGST, SGST, and IGST by rate slab.', badge: 'Tax Slab' },
		{ id: '10', name: '10. Simple Bill of Supply', category: 'Composition & Non-GST Traders', desc: 'No tax columns (Qty x Rate = Total), equipped with statutory Section 31(3)(c) composition notice.', badge: 'Composition' },
		{ id: '11', name: '11. Industrial BlueMetals & Transport', category: 'Mining, Heavy Materials & Transport', desc: 'Heavy boxed grid format with PO No, Vehicle No, Delivery Place, E.& O.E, and authentic Indian quarry/metals GST structure.', badge: 'Industry Favorite' },
		{ id: '12', name: '12. Hostel / Lodging Guest Folio', category: 'Hostels, PGs, Guest Houses & Lodges', desc: 'Guest folio with room/bed, booking ref, check-in/check-out, nights stayed, GST slab summary, advance & balance, and guest signature.', badge: 'Hospitality' },
		{ id: '13', name: '13. Restaurant / Cafe Bill', category: 'Restaurants, Cafes, Bakeries & Cloud Kitchens', desc: 'Compact dine-in bill with table, waiter, KOT/order no, FSSAI licence, CGST/SGST split, UPI QR and a customisable thank-you line.', badge: 'Food Service' },
		{ id: '14', name: '14. Pharmacy / Medical Store', category: 'Pharmacies, Chemists & Medical Stores', desc: 'Drug licence header, patient & prescribing doctor, per-line batch / expiry / MRP with short-expiry flag, MRP savings, GST slab summary, Schedule H notice and pharmacist signature.', badge: 'Pharma' },
		{ id: '15', name: '15. Bakery / Sweet Shop', category: 'Bakeries, Cake Shops, Sweet Marts & Cafes', desc: 'FSSAI licence header, veg / non-veg marks, best-before per item, cake-order block (pickup date, message on cake), per-rate GST split, advance paid & balance, UPI QR.', badge: 'Bakery' },
		{ id: '16', name: '16. Grocery / Supermarket', category: 'Kirana Stores, Supermarkets & Fresh Produce', desc: 'MRP vs selling price per line, total savings badge, weighed quantities (kg / L), item & quantity count, GST summary that keeps exempt staples visible, cash tendered & change, UPI QR.', badge: 'Grocery' },
		{ id: '17', name: '17. Hardware / Building Materials', category: 'Hardware, Paints, Plumbing, Electricals & Building Materials', desc: 'Trade-discount column, delivery / site block with transport details (vehicle, e-way, LR), HSN-wise tax summary, credit terms with due date, bank details, receiver and authorised signatures.', badge: 'Hardware' }
	];

	let activeTab = $state<'quickbill' | 'invoices' | 'khata' | 'templates'>('quickbill');
	let loading = $state(true);
	let billingOnlyMode = $state(false);

	// Template Preferences & Options State
	let selectedTemplateId = $state('1');
	let selectedPaperSize = $state<'A4' | 'A5' | 'A5_LANDSCAPE'>('A4');
	let selectedCopyType = $state<'ORIGINAL' | 'DUPLICATE' | 'TRIPLICATE' | 'OFFICE'>('ORIGINAL');
	let selectedTaxMode = $state<'WITH_GST' | 'WITHOUT_GST'>('WITH_GST');
	let brandColor = $state('#0284c7');
	let showBankDetails = $state(true);
	let showUPIQR = $state(true);
	let showTaxSummary = $state(true);
	let businessSubtitle = $state('Wholesale & Retail Commercial Trading');
	let signatoryTitle = $state('Authorized Signatory');
	let defaultTerms = $state('Payment within 15 days from invoice date.');
	let defaultJurisdiction = $state('Mumbai');
	let defaultInterestRate = $state('18% p.a.');
	let defaultReturnPolicy = $state('Goods once sold can only be replaced within 7 days in original condition.');

	// Company info for InvoiceTemplates component
	let company = $state<Tenant>({
		id: 'tenant-1',
		name: 'MrBiz Commercial Enterprises',
		legal_name: 'MrBiz Solutions LLP',
		gstin: '27AABCA1234F1Z6',
		address: 'Shop 14, Commercial Complex, MG Road',
		city: 'Mumbai',
		state: 'Maharashtra',
		country: 'India',
		pincode: '400001',
		email: 'billing@mrbiz.app',
		phone: '+91 98200 12345',
		invoice_prefix: 'INV-2026',
		financial_year_start: '2026-04-01',
		currency: 'INR',
		subscription_plan: 'BUSINESS',
		subscription_status: 'ACTIVE',
		bank_name: 'HDFC Bank Ltd',
		bank_account_no: '50200012345678',
		bank_ifsc: 'HDFC0000240',
		bank_branch: 'Fort, Mumbai',
		bank_account_holder: 'MrBiz Solutions LLP',
		upi_id: 'mrbiz@upi',
		business_subtitle: 'Wholesale & Retail Commercial Trading',
		signatory_title: 'Authorized Signatory',
		default_terms: 'Payment within 15 days from invoice date.',
		default_jurisdiction: 'Mumbai',
		default_interest_rate: '18% p.a.',
		default_return_policy: 'Goods once sold can only be replaced within 7 days in original condition.',
		drug_license_no: 'DL-MH-2026-9921',
		fssai_no: '11521001000452',
		is_active: true,
		created_at: new Date().toISOString()
	});

	// Summary KPI metrics
	let summary = $state<BillingSummary>({
		todaySales: '0.00',
		monthSales: '0.00',
		totalOutstanding: '0.00',
		totalInvoicesCount: 0,
		paidInvoicesCount: 0,
		unpaidInvoicesCount: 0,
		topReceivables: []
	});

	let invoices = $state<Invoice[]>([]);
	let statusFilter = $state<string>('ALL');

	// New Quick Bill State
	let customerName = $state('Cash Walk-in Customer');
	let customerPhone = $state('+91 98200 12345');
	let customerGstin = $state('');
	let supplyState = $state('Maharashtra (27)');
	let paymentMode = $state<'CASH' | 'UPI' | 'CARD' | 'CREDIT'>('UPI');
	let tenderedAmount = $state('');
	let billNotes = $state('Thank you for your business!');

	let items = $state<InvoiceItem[]>([
		{
			productId: 'PROD-01',
			itemDescription: 'Organic Cold-Pressed Coconut Oil 1L',
			hsnSac: '1513',
			quantity: '2',
			unitCode: 'BTL',
			unitPrice: '350.00',
			discountRate: '5',
			taxRate: '18'
		},
		{
			productId: 'PROD-02',
			itemDescription: 'Pure Forest Wild Honey 500g',
			hsnSac: '0409',
			quantity: '1',
			unitCode: 'JAR',
			unitPrice: '500.00',
			discountRate: '0',
			taxRate: '18'
		}
	]);

	// Payment Modal State
	let showPayModal = $state(false);
	let selectedInvoiceForPay = $state<Invoice | null>(null);
	let payAmount = $state('');
	let payMode = $state('UPI');
	let payRef = $state('');
	let payMessage = $state<string | null>(null);

	// Template Preview Modal State
	let showTemplatePreviewModal = $state(false);
	let previewInvoiceData = $state<ApiInvoice | null>(null);

	// Action notification banner
	let bannerMessage = $state<string | null>(null);

	// Live calculations for quick bill
	let isInterState = $derived(supplyState !== 'Maharashtra (27)');

	let grossSubtotal = $derived(
		items.reduce((sum, item) => {
			const q = parseFloat(item.quantity) || 0;
			const p = parseFloat(item.unitPrice) || 0;
			return sum + q * p;
		}, 0)
	);

	let totalDiscount = $derived(
		items.reduce((sum, item) => {
			const q = parseFloat(item.quantity) || 0;
			const p = parseFloat(item.unitPrice) || 0;
			const d = parseFloat(item.discountRate) || 0;
			return sum + (q * p * d) / 100.0;
		}, 0)
	);

	let taxableAmount = $derived(grossSubtotal - totalDiscount);

	let totalTax = $derived(
		items.reduce((sum, item) => {
			const q = parseFloat(item.quantity) || 0;
			const p = parseFloat(item.unitPrice) || 0;
			const d = parseFloat(item.discountRate) || 0;
			const lineTaxable = q * p - (q * p * d) / 100.0;
			const t = parseFloat(item.taxRate) || 0;
			return sum + (lineTaxable * t) / 100.0;
		}, 0)
	);

	let grandTotal = $derived(taxableAmount + totalTax);

	let payableAmount = $derived(Math.round(grandTotal));
	let roundOff = $derived((payableAmount - grandTotal).toFixed(2));

	let calculatedChange = $derived.by(() => {
		const tendered = parseFloat(tenderedAmount);
		if (isNaN(tendered) || tendered < payableAmount) return 0;
		return (tendered - payableAmount).toFixed(2);
	});

	let calculatedDue = $derived.by(() => {
		if (paymentMode === 'CREDIT') return payableAmount.toFixed(2);
		const tendered = parseFloat(tenderedAmount);
		if (isNaN(tendered)) return payableAmount.toFixed(2);
		if (tendered >= payableAmount) return '0.00';
		return (payableAmount - tendered).toFixed(2);
	});

	// UPI payment URI
	let upiUri = $derived(
		`upi://pay?pa=mrbiz@upi&pn=MrBizStore&am=${payableAmount}&cu=INR&tn=BillPayment`
	);

	// ── 🏗️ Blue Metal Quarry & Weighbridge System ──────────────────────────────
	let isQuarryMode = $state(false);
	let lorryNumber = $state('TN 38 AB 1234');
	let grossWeightKg = $state('28650');
	let tareWeightKg = $state('8450');
	let netWeightTons = $derived.by(() => {
		const g = parseFloat(grossWeightKg) || 0;
		const t = parseFloat(tareWeightKg) || 0;
		if (g > t) {
			return ((g - t) / 1000).toFixed(2);
		}
		return '20.20';
	});
	let deliveryPlace = $state('Bypass Flyover Pier #12');
	let poNumber = $state('PO-QUARRY-9821');
	let lrNumber = $state('LR-6621');
	let ewayBill = $state('331298457891');
	let driverName = $state('K. Murugan');
	let driverPhone = $state('+91 94431 23456');
	let transitPassNo = $state('TP-2026-0941');

	interface QuarryMaterial {
		name: string;
		hsn: string;
		rate: string;
		unit: string;
		taxRate: string;
		icon: string;
		desc: string;
	}

	const quarryMaterials: QuarryMaterial[] = [
		{ name: '20mm Blue Metal (Jally)', hsn: '25171010', rate: '550.00', unit: 'MT', taxRate: '5', icon: '🪨', desc: 'Concrete Slab & Columns' },
		{ name: '40mm Blue Metal (Ballast)', hsn: '25171010', rate: '480.00', unit: 'MT', taxRate: '5', icon: '🪨', desc: 'Railway & Heavy Foundation' },
		{ name: '12mm Blue Metal', hsn: '25171010', rate: '580.00', unit: 'MT', taxRate: '5', icon: '🪨', desc: 'Plastering & Interlock Pavers' },
		{ name: '6mm Chipped Gravel', hsn: '25171010', rate: '620.00', unit: 'MT', taxRate: '5', icon: '🪨', desc: 'Bitumen Road Works' },
		{ name: 'M-Sand (Manufactured Sand)', hsn: '25051010', rate: '650.00', unit: 'MT', taxRate: '5', icon: '🏖️', desc: 'IS 383 Concreting Sand' },
		{ name: 'P-Sand (Plastering Sand)', hsn: '25051010', rate: '750.00', unit: 'MT', taxRate: '5', icon: '🏖️', desc: 'Smooth Super-Fine Plaster' },
		{ name: 'GSB (Granular Sub Base)', hsn: '25171020', rate: '380.00', unit: 'MT', taxRate: '5', icon: '🛣️', desc: 'Highway Sub-Base Grading' },
		{ name: 'Wet Mix Macadam (WMM)', hsn: '25171020', rate: '420.00', unit: 'MT', taxRate: '5', icon: '🛣️', desc: 'Moist Base Mix' },
		{ name: 'Quarry Dust / Stone Powder', hsn: '25171010', rate: '320.00', unit: 'MT', taxRate: '5', icon: '🌪️', desc: 'Hollow Brick & Paver Base' }
	];

	function applyQuarryMaterial(mat: QuarryMaterial) {
		selectedTemplateId = '11';
		items = [
			{
				productId: `BM-${mat.hsn}`,
				itemDescription: mat.name,
				hsnSac: mat.hsn,
				quantity: netWeightTons,
				unitCode: mat.unit,
				unitPrice: mat.rate,
				discountRate: '0',
				taxRate: mat.taxRate
			}
		];
		billNotes = `Vehicle No: ${lorryNumber} | Gross: ${grossWeightKg} kg, Tare: ${tareWeightKg} kg, Net: ${netWeightTons} MT | Pass: ${transitPassNo} | Delivery: ${deliveryPlace}`;
		bannerMessage = `🪨 Loaded ${mat.name} (${netWeightTons} MT @ ₹${mat.rate}/MT) with Template 11!`;
		setTimeout(() => (bannerMessage = null), 4000);
	}

	function syncNetTonnageToItems() {
		if (items.length > 0) {
			items[0].quantity = netWeightTons;
			bannerMessage = `⚖️ Updated line item quantity to weighbridge net weight: ${netWeightTons} MT`;
			setTimeout(() => (bannerMessage = null), 3000);
		}
	}

	function toggleQuarryMode() {
		isQuarryMode = !isQuarryMode;
		if (isQuarryMode) {
			selectedTemplateId = '11';
			businessSubtitle = 'Quarry, Aggregates & Ready-Mix Transport';
			customerName = 'Bypass Infrastructure Pvt Ltd';
			supplyState = 'Tamil Nadu (33)';
			applyQuarryMaterial(quarryMaterials[0]);
			bannerMessage = '🏗️ Blue Metals & Quarry Weighbridge Mode activated (Template 11 Active)!';
		} else {
			selectedTemplateId = '1';
			businessSubtitle = 'Wholesale & Retail Commercial Trading';
			customerName = 'Cash Walk-in Customer';
			supplyState = 'Maharashtra (27)';
			bannerMessage = '🔄 General Retail / Express Mode restored.';
		}
		setTimeout(() => (bannerMessage = null), 4000);
	}

	onMount(async () => {
		if (typeof localStorage !== 'undefined') {
			billingOnlyMode = localStorage.getItem('mrbiz_billing_only_mode') === 'true';
		}
		if (typeof window !== 'undefined') {
			const params = new URLSearchParams(window.location.search);
			if (params.get('mode') === 'quarry' || params.get('template') === '11') {
				toggleQuarryMode();
			}
		}
		await loadBillingData();
		await loadTenantTemplate();
	});

	function toggleBillingOnlyMode() {
		billingOnlyMode = !billingOnlyMode;
		if (typeof localStorage !== 'undefined') {
			localStorage.setItem('mrbiz_billing_only_mode', String(billingOnlyMode));
		}
		bannerMessage = billingOnlyMode
			? '⚡ Focused Billing Mode activated! Non-essential modules hidden.'
			: '🌐 Full Business-OS Suite restored.';
		setTimeout(() => (bannerMessage = null), 4000);
	}

	async function loadTenantTemplate() {
		try {
			const res = await billingGrpc.getTenantBillingTemplate({ tenant_id: 'tenant-1' });
			if (res) {
				selectedTemplateId = res.template_id || '1';
				selectedPaperSize = (res.paper_size as any) || 'A4';
				selectedCopyType = (res.copy_type as any) || 'ORIGINAL';
				selectedTaxMode = (res.tax_mode as any) || 'WITH_GST';
				brandColor = res.brand_color || '#0284c7';
				showBankDetails = res.show_bank_details ?? true;
				showUPIQR = res.show_upi_qr ?? true;
				showTaxSummary = res.show_tax_summary ?? true;
				businessSubtitle = res.business_subtitle || businessSubtitle;
				signatoryTitle = res.signatory_title || signatoryTitle;
				defaultTerms = res.default_terms || defaultTerms;
				defaultJurisdiction = res.default_jurisdiction || defaultJurisdiction;
				defaultInterestRate = res.default_interest_rate || defaultInterestRate;
				defaultReturnPolicy = res.default_return_policy || defaultReturnPolicy;
			}
		} catch (e) {
			console.warn('Template config load fallback:', e);
		}
	}

	async function saveTemplateConfig() {
		try {
			loading = true;
			const res = await billingGrpc.updateTenantBillingTemplate({
				tenant_id: 'tenant-1',
				template_id: selectedTemplateId,
				paper_size: selectedPaperSize,
				copy_type: selectedCopyType,
				tax_mode: selectedTaxMode,
				brand_color: brandColor,
				show_bank_details: showBankDetails,
				show_upi_qr: showUPIQR,
				show_tax_summary: showTaxSummary,
				business_subtitle: businessSubtitle,
				signatory_title: signatoryTitle,
				default_terms: defaultTerms,
				default_jurisdiction: defaultJurisdiction,
				default_interest_rate: defaultInterestRate,
				default_return_policy: defaultReturnPolicy,
				template_options_json: JSON.stringify({
					primary_color: brandColor,
					show_bank: showBankDetails,
					show_upi: showUPIQR,
					show_tax: showTaxSummary
				})
			});
			if (res) {
				bannerMessage = `✅ ${res.message || 'Billing template preferences saved successfully!'}`;
			}
		} catch (e: any) {
			bannerMessage = `Error saving template: ${e.message}`;
		} finally {
			loading = false;
			setTimeout(() => (bannerMessage = null), 5000);
		}
	}

	async function loadBillingData() {
		loading = true;
		try {
			const [invRes, sumRes] = await Promise.all([
				billingGrpc.listInvoices({ tenant_id: 'tenant-1' }),
				billingGrpc.getBillingSummary({ tenant_id: 'tenant-1' })
			]);

			if (invRes && invRes.invoices) {
				invoices = invRes.invoices.map((i: any) => ({
					id: i.id,
					tenantId: i.tenant_id,
					invoiceNumber: i.invoice_number,
					customerId: i.customer_id,
					customerName: i.customer_name,
					customerGstin: i.customer_gstin,
					invoiceDate: i.invoice_date,
					dueDate: i.due_date,
					status: i.status,
					subtotal: i.subtotal,
					discountTotal: i.discount_total,
					taxTotal: i.tax_total,
					grandTotal: i.grand_total,
					amountPaid: i.amount_paid,
					balanceDue: i.balance_due,
					paymentMode: i.payment_mode,
					supplyState: i.supply_state,
					isInterState: i.is_inter_state,
					cgstTotal: i.cgst_total,
					sgstTotal: i.sgst_total,
					igstTotal: i.igst_total,
					templateId: i.template_id || '1',
					paperSize: i.paper_size || 'A4',
					vehicleNumber: i.vehicle_number,
					deliveryPlace: i.delivery_place,
					grossWeight: i.gross_weight,
					netWeight: i.net_weight,
					poNumber: i.po_number,
					lrNumber: i.lr_number,
					ewayBill: i.eway_bill,
					items: i.items?.map((it: any) => ({
						id: it.id,
						productId: it.product_id,
						itemDescription: it.item_description,
						hsnSac: it.hsn_sac,
						quantity: it.quantity,
						unitCode: it.unit_code,
						unitPrice: it.unit_price,
						discountRate: it.discount_rate,
						taxRate: it.tax_rate,
						taxableAmount: it.taxable_amount,
						cgstAmount: it.cgst_amount,
						sgstAmount: it.sgst_amount,
						igstAmount: it.igst_amount,
						totalAmount: it.total_amount
					}))
				}));
			}
			if (sumRes) {
				summary = {
					todaySales: sumRes.today_sales || '0.00',
					monthSales: sumRes.month_sales || '0.00',
					totalOutstanding: sumRes.total_outstanding || '0.00',
					totalInvoicesCount: sumRes.total_invoices_count || 0,
					paidInvoicesCount: sumRes.paid_invoices_count || 0,
					unpaidInvoicesCount: sumRes.unpaid_invoices_count || 0,
					topReceivables: sumRes.top_receivables || []
				};
			}
		} catch (e: any) {
			console.warn('Billing load error, fallback ready:', e);
		} finally {
			loading = false;
		}
	}

	function addItem() {
		items = [
			...items,
			{
				productId: `PROD-${items.length + 1}`,
				itemDescription: '',
				hsnSac: '9983',
				quantity: '1',
				unitCode: 'PCS',
				unitPrice: '100.00',
				discountRate: '0',
				taxRate: '18'
			}
		];
	}

	function removeItem(index: number) {
		if (items.length > 1) {
			items = items.filter((_, i) => i !== index);
		}
	}

	function setWalkinCustomer() {
		customerName = 'Cash Walk-in Customer';
		customerPhone = '+91 98200 12345';
		customerGstin = '';
		paymentMode = 'CASH';
		tenderedAmount = String(payableAmount);
	}

	async function generateBill() {
		try {
			loading = true;
			const res = await billingGrpc.createInvoice({
				tenant_id: 'tenant-1',
				customer_id: customerName,
				supply_state: supplyState,
				payment_mode: paymentMode,
				tendered_cash: paymentMode === 'CREDIT' ? '0' : tenderedAmount || String(payableAmount),
				notes: billNotes,
				template_id: selectedTemplateId,
				paper_size: selectedPaperSize,
				vehicle_number: isQuarryMode ? lorryNumber : '',
				delivery_place: isQuarryMode ? deliveryPlace : '',
				gross_weight: isQuarryMode ? grossWeightKg + ' kg' : '',
				net_weight: isQuarryMode ? netWeightTons + ' MT' : '',
				po_number: isQuarryMode ? poNumber : '',
				lr_number: isQuarryMode ? lrNumber : '',
				eway_bill: isQuarryMode ? ewayBill : '',
				template_options_json: JSON.stringify({
					primary_color: brandColor,
					show_bank: showBankDetails,
					show_upi: showUPIQR,
					show_tax: showTaxSummary
				}),
				items: items.map((i) => ({
					product_id: i.productId,
					item_description: i.itemDescription || 'General Goods/Service',
					hsn_sac: i.hsnSac,
					quantity: i.quantity,
					unit_code: i.unitCode,
					unit_price: i.unitPrice,
					discount_rate: i.discountRate,
					tax_rate: i.taxRate
				}))
			});

			if (res && res.invoice) {
				bannerMessage = `✅ Bill ${res.invoice.invoice_number} created successfully with Template ${selectedTemplateId}! Total: ₹${res.invoice.grand_total}`;
				await loadBillingData();
				activeTab = 'invoices';
			}
		} catch (err: any) {
			bannerMessage = `Error creating bill: ${err.message}`;
		} finally {
			loading = false;
			setTimeout(() => (bannerMessage = null), 5000);
		}
	}

	function shareBillOnWhatsApp(inv?: Invoice) {
		const targetInv = inv || {
			invoiceNumber: 'INV-NEW',
			customerName,
			grandTotal: String(payableAmount),
			balanceDue: calculatedDue,
			paymentMode
		};

		let text = '';
		if (isQuarryMode) {
			text = `🚚 *WEIGHMENT DELIVERY CHALLAN (DC)*\n*${company.name || 'Sri Krishna Blue Metals'}*\n------------------------------\n*Bill No:* ${targetInv.invoiceNumber}\n*Lorry No:* ${lorryNumber}\n*Driver:* ${driverName} (${driverPhone})\n*Material:* ${items[0]?.itemDescription || 'Blue Metal'}\n*Gross Wt:* ${grossWeightKg} kg\n*Tare Wt:* ${tareWeightKg} kg\n*Net Tonnage:* ${netWeightTons} MT\n*Total Amount:* ₹${targetInv.grandTotal} (Incl. GST)\n*Site / Unloading:* ${deliveryPlace}\n*E-Way Bill:* ${ewayBill}\n*Permit No:* ${transitPassNo}\n------------------------------\nThank you for choosing us!`;
		} else {
			text = `Hello ${targetInv.customerName}, here is your bill *${targetInv.invoiceNumber}* from MrBiz-OS Store.\nTotal Amount: ₹${targetInv.grandTotal}\nBalance Due: ₹${targetInv.balanceDue}\nPayment Link: https://mrbiz.app/pay/${targetInv.invoiceNumber}\nThank you!`;
		}
		const url = `https://wa.me/?text=${encodeURIComponent(text)}`;
		window.open(url, '_blank');
	}

	function printFormattedInvoice() {
		window.print();
	}

	function openPaymentModal(inv: Invoice) {
		selectedInvoiceForPay = inv;
		payAmount = inv.balanceDue;
		payMode = 'UPI';
		payRef = 'UPI-' + Date.now().toString().slice(-6);
		payMessage = null;
		showPayModal = true;
	}

	async function submitRecordPayment() {
		if (!selectedInvoiceForPay) return;
		try {
			const res = await billingGrpc.recordPayment({
				tenant_id: 'tenant-1',
				invoice_id: selectedInvoiceForPay.id,
				amount_paid: payAmount,
				payment_mode: payMode,
				reference_number: payRef,
				payment_date: new Date().toISOString().split('T')[0]
			});

			if (res) {
				bannerMessage = res.message || 'Payment recorded successfully!';
				showPayModal = false;
				await loadBillingData();
			}
		} catch (e: any) {
			payMessage = e.message;
		}
	}

	// Open Template Preview Modal for an Invoice or Live Draft
	function openInvoiceTemplatePreview(inv?: Invoice) {
		if (inv) {
			previewInvoiceData = {
				id: inv.id,
				tenant_id: inv.tenantId,
				invoice_number: inv.invoiceNumber,
				customer_id: inv.customerId,
				customer_name: inv.customerName,
				customer_gstin: inv.customerGstin || '',
				invoice_date: inv.invoiceDate,
				due_date: inv.dueDate,
				status: inv.status,
				subtotal: inv.subtotal,
				discount_total: inv.discountTotal,
				taxable_amount: (parseFloat(inv.subtotal) - parseFloat(inv.discountTotal)).toFixed(2),
				cgst_total: inv.cgstTotal || '0.00',
				sgst_total: inv.sgstTotal || '0.00',
				igst_total: inv.igstTotal || '0.00',
				round_off: '0.00',
				grand_total: inv.grandTotal,
				amount_paid: inv.amountPaid,
				balance_due: inv.balanceDue,
				supply_state: inv.supplyState,
				is_inter_state: inv.isInterState,
				billing_address: 'Main Market, Station Road, Mumbai 400001',
				shipping_address: 'Main Market, Station Road, Mumbai 400001',
				utgst_total: '0.00',
				notes: 'Thank you for choosing MrBiz-OS.',
				vehicle_number: inv.vehicleNumber || (isQuarryMode ? lorryNumber : ''),
				delivery_place: inv.deliveryPlace || (isQuarryMode ? deliveryPlace : ''),
				gross_weight: inv.grossWeight || (isQuarryMode ? grossWeightKg + ' kg' : ''),
				net_weight: inv.netWeight || (isQuarryMode ? netWeightTons + ' MT' : ''),
				po_number: inv.poNumber || (isQuarryMode ? poNumber : ''),
				lr_number: inv.lrNumber || (isQuarryMode ? lrNumber : ''),
				eway_bill: inv.ewayBill || (isQuarryMode ? ewayBill : ''),
				items: inv.items?.map((it, idx) => ({
					id: it.id || String(idx),
					invoice_id: inv.id,
					product_id: it.productId,
					item_description: it.itemDescription,
					hsn_sac: it.hsnSac,
					quantity: it.quantity,
					unit_code: it.unitCode,
					unit_price: it.unitPrice,
					discount_rate: it.discountRate,
					discount_amount: '0.00',
					taxable_amount: it.taxableAmount || (parseFloat(it.quantity) * parseFloat(it.unitPrice)).toFixed(2),
					tax_rate: it.taxRate,
					cgst_rate: (parseFloat(it.taxRate) / 2).toFixed(1),
					sgst_rate: (parseFloat(it.taxRate) / 2).toFixed(1),
					igst_rate: it.taxRate,
					cgst_amount: it.cgstAmount || '0.00',
					sgst_amount: it.sgstAmount || '0.00',
					igst_amount: it.igstAmount || '0.00',
					total_amount: it.totalAmount || (parseFloat(it.quantity) * parseFloat(it.unitPrice) * (1 + parseFloat(it.taxRate)/100)).toFixed(2),
					created_at: new Date().toISOString()
				})) || [],
				created_at: new Date().toISOString()
			};
			if (inv.templateId) {
				selectedTemplateId = inv.templateId;
			}
		} else {
			// Live Draft Invoice Preview
			previewInvoiceData = {
				id: 'draft-bill',
				tenant_id: 'tenant-1',
				invoice_number: 'INV-DRAFT-PREVIEW',
				customer_id: 'cust-draft',
				customer_name: customerName,
				customer_gstin: customerGstin,
				invoice_date: new Date().toISOString().split('T')[0],
				due_date: new Date(Date.now() + 15 * 86400000).toISOString().split('T')[0],
				status: 'DRAFT',
				subtotal: grossSubtotal.toFixed(2),
				discount_total: totalDiscount.toFixed(2),
				taxable_amount: taxableAmount.toFixed(2),
				cgst_total: (totalTax / 2).toFixed(2),
				sgst_total: (totalTax / 2).toFixed(2),
				igst_total: isInterState ? totalTax.toFixed(2) : '0.00',
				round_off: roundOff,
				grand_total: payableAmount.toFixed(2),
				amount_paid: tenderedAmount || '0.00',
				balance_due: calculatedDue,
				supply_state: supplyState,
				is_inter_state: isInterState,
				billing_address: isQuarryMode ? `Delivery Site: ${deliveryPlace}` : 'Walk-in Counter, MrBiz Store',
				shipping_address: isQuarryMode ? deliveryPlace : 'Walk-in Counter, MrBiz Store',
				utgst_total: '0.00',
				notes: billNotes,
				vehicle_number: isQuarryMode ? lorryNumber : '',
				delivery_place: isQuarryMode ? deliveryPlace : '',
				gross_weight: isQuarryMode ? grossWeightKg + ' kg' : '',
				net_weight: isQuarryMode ? netWeightTons + ' MT' : '',
				po_number: isQuarryMode ? poNumber : '',
				lr_number: isQuarryMode ? lrNumber : '',
				eway_bill: isQuarryMode ? ewayBill : '',
				items: items.map((it, idx) => ({
					id: String(idx + 1),
					invoice_id: 'draft',
					product_id: it.productId,
					item_description: it.itemDescription || 'General Goods/Service',
					hsn_sac: it.hsnSac,
					quantity: it.quantity,
					unit_code: it.unitCode,
					unit_price: it.unitPrice,
					discount_rate: it.discountRate,
					discount_amount: ((parseFloat(it.quantity) * parseFloat(it.unitPrice) * parseFloat(it.discountRate)) / 100).toFixed(2),
					taxable_amount: (parseFloat(it.quantity) * parseFloat(it.unitPrice) * (1 - parseFloat(it.discountRate)/100)).toFixed(2),
					tax_rate: it.taxRate,
					cgst_rate: (parseFloat(it.taxRate) / 2).toFixed(1),
					sgst_rate: (parseFloat(it.taxRate) / 2).toFixed(1),
					igst_rate: it.taxRate,
					cgst_amount: ((parseFloat(it.quantity) * parseFloat(it.unitPrice) * parseFloat(it.taxRate)) / 200).toFixed(2),
					sgst_amount: ((parseFloat(it.quantity) * parseFloat(it.unitPrice) * parseFloat(it.taxRate)) / 200).toFixed(2),
					igst_amount: isInterState ? ((parseFloat(it.quantity) * parseFloat(it.unitPrice) * parseFloat(it.taxRate)) / 100).toFixed(2) : '0.00',
					total_amount: (parseFloat(it.quantity) * parseFloat(it.unitPrice) * (1 + parseFloat(it.taxRate)/100)).toFixed(2),
					created_at: new Date().toISOString()
				})),
				created_at: new Date().toISOString()
			};
		}
		showTemplatePreviewModal = true;
	}

	let filteredInvoices = $derived.by(() => {
		if (statusFilter === 'ALL') return invoices;
		return invoices.filter((i) => i.status === statusFilter);
	});
</script>

<svelte:head>
	<title>Express Billing & Invoicing Counter | MrBiz-OS</title>
</svelte:head>

<div class="billing-container">
	<!-- Top Notification Banner -->
	{#if bannerMessage}
		<div class="alert-banner">
			<span>{bannerMessage}</span>
			<button class="banner-close" onclick={() => (bannerMessage = null)}>✕</button>
		</div>
	{/if}

	<!-- Header with Mode Switcher -->
	<header class="billing-header">
		<div class="title-group">
			<div class="title-row">
				<h1>⚡ Express Billing & Quick Invoicing Counter</h1>
				<span class="badge-tag">GST Ready</span>
				<span class="badge-template">Template {selectedTemplateId} Active</span>
				{#if billingOnlyMode}
					<span class="badge-mode">Focused Billing Only</span>
				{/if}
			</div>
			<p class="subtitle">
				Ultra-fast 3-second billing with 17 industry invoice templates. Auto GST, dynamic UPI QR code, 80mm thermal print, and WhatsApp sharing.
			</p>
		</div>

		<div class="header-actions">
			<button
				class="quarry-toggle-btn"
				class:active={isQuarryMode}
				onclick={toggleQuarryMode}
				title="Toggle Blue Metals & Quarry Weighbridge Station"
			>
				{isQuarryMode ? '🏗️ Quarry Mode: ON (Template 11)' : '🏗️ Blue Metals Quarry Mode'}
			</button>

			<button
				class="mode-toggle-btn"
				class:active={billingOnlyMode}
				onclick={toggleBillingOnlyMode}
				title="Toggle between focused billing-only mode and full enterprise suite"
			>
				{billingOnlyMode ? '⚙️ Switch to Full ERP Suite' : '⚡ Focused Billing-Only Mode'}
			</button>

			<button class="walkin-btn" onclick={setWalkinCustomer}>
				🚶 Cash Walk-in Customer
			</button>
		</div>
	</header>

	<!-- KPI Metric Cards -->
	<div class="metrics-grid">
		<div class="kpi-card sales-today">
			<div class="kpi-icon">💰</div>
			<div class="kpi-info">
				<span class="kpi-label">Today's Sales</span>
				<span class="kpi-value">₹{summary.todaySales}</span>
			</div>
		</div>

		<div class="kpi-card sales-month">
			<div class="kpi-icon">📈</div>
			<div class="kpi-info">
				<span class="kpi-label">This Month's Sales</span>
				<span class="kpi-value">₹{summary.monthSales}</span>
			</div>
		</div>

		<div class="kpi-card outstanding">
			<div class="kpi-icon">⏳</div>
			<div class="kpi-info">
				<span class="kpi-label">Total Outstanding Due</span>
				<span class="kpi-value warning">₹{summary.totalOutstanding}</span>
			</div>
		</div>

		<div class="kpi-card bill-counts">
			<div class="kpi-icon">🧾</div>
			<div class="kpi-info">
				<span class="kpi-label">Invoices Generated</span>
				<span class="kpi-value">
					{summary.paidInvoicesCount} <span class="sub">Paid</span> / {summary.unpaidInvoicesCount} <span class="sub">Pending</span>
				</span>
			</div>
		</div>
	</div>

	<!-- Navigation Tabs -->
	<nav class="billing-tabs" aria-label="Billing Views">
		<button
			class="tab-btn"
			class:active={activeTab === 'quickbill'}
			onclick={() => (activeTab = 'quickbill')}
		>
			⚡ Express Billing Counter
		</button>
		<button
			class="tab-btn"
			class:active={activeTab === 'invoices'}
			onclick={() => (activeTab = 'invoices')}
		>
			📜 All Bills & Invoices ({invoices.length})
		</button>
		<button
			class="tab-btn"
			class:active={activeTab === 'khata'}
			onclick={() => (activeTab = 'khata')}
		>
			👥 Receivables & Khata ({summary.topReceivables.length})
		</button>
		<button
			class="tab-btn"
			class:active={activeTab === 'templates'}
			onclick={() => (activeTab = 'templates')}
		>
			🎨 17 Billing Templates & Options
		</button>
	</nav>

	<!-- TAB 1: EXPRESS BILLING COUNTER -->
	{#if activeTab === 'quickbill'}
		<div class="bill-workbench">
			<!-- Left: Customer & Items -->
			<div class="bill-main">
				<!-- 🏗️ Dedicated Blue Metals & Weighbridge Station -->
				{#if isQuarryMode}
					<div class="card quarry-station-card">
						<div class="quarry-header">
							<div class="quarry-title">
								<div class="quarry-badge-row">
									<span class="quarry-badge">🏭 BLUE METALS WEIGHBRIDGE</span>
									<span class="quarry-subbadge">Template 11 Heavy Grid Active</span>
								</div>
								<h3>Quarry Weighbridge & Material Dispatch Station</h3>
								<p>Real-time Tonnage computation, Lorry Trip Pass, and authentic Indian quarry GST billing</p>
							</div>
							<div class="weighbridge-digital-display">
								<span class="display-label">NET TONNAGE</span>
								<span class="display-value">{netWeightTons} <small>MT</small></span>
								<span class="display-sub">Gross: {grossWeightKg} kg • Tare: {tareWeightKg} kg</span>
							</div>
						</div>

						<!-- Weighbridge Weight & Lorry Matrix -->
						<div class="quarry-form-grid">
							<div class="form-group">
								<label for="lorryNum">🚛 Lorry / Vehicle Number</label>
								<input id="lorryNum" type="text" bind:value={lorryNumber} placeholder="e.g. TN 38 AB 1234" />
							</div>

							<div class="form-group">
								<label for="grossWt">Gross Weight (Loaded kg)</label>
								<input id="grossWt" type="number" bind:value={grossWeightKg} oninput={syncNetTonnageToItems} placeholder="e.g. 28650" />
							</div>

							<div class="form-group">
								<label for="tareWt">Tare Weight (Empty kg)</label>
								<input id="tareWt" type="number" bind:value={tareWeightKg} oninput={syncNetTonnageToItems} placeholder="e.g. 8450" />
							</div>

							<div class="form-group">
								<label for="delPlace">📍 Unloading Site / Location</label>
								<input id="delPlace" type="text" bind:value={deliveryPlace} placeholder="e.g. Bypass Flyover Pier #12" />
							</div>

							<div class="form-group">
								<label for="transitPass">📜 Mineral Transit Pass / Permit No</label>
								<input id="transitPass" type="text" bind:value={transitPassNo} placeholder="e.g. TP-2026-0941" />
							</div>

							<div class="form-group">
								<label for="driverContact">👤 Driver Name & Mobile</label>
								<input id="driverContact" type="text" bind:value={driverName} placeholder="e.g. K. Murugan (+91 94431 23456)" />
							</div>

							<div class="form-group">
								<label for="ewayNum">📄 E-Way Bill Number</label>
								<input id="ewayNum" type="text" bind:value={ewayBill} placeholder="12-digit E-Way Bill" />
							</div>

							<div class="form-group">
								<label for="poNum">📋 P.O. / Work Order Ref</label>
								<input id="poNum" type="text" bind:value={poNumber} placeholder="e.g. PO-ROAD-442" />
							</div>
						</div>

						<!-- 1-Click Blue Metal Materials Palette -->
						<div class="materials-palette-wrapper">
							<div class="palette-header">
								<span class="palette-title">🪨 Quick-Add Quarry Materials (Auto-Calculates at {netWeightTons} MT):</span>
								<span class="palette-hint">Click any grade to instantly populate bill</span>
							</div>
							<div class="quarry-materials-grid">
								{#each quarryMaterials as mat}
									<button type="button" class="material-chip" onclick={() => applyQuarryMaterial(mat)}>
										<span class="mat-icon">{mat.icon}</span>
										<div class="mat-info">
											<span class="mat-name">{mat.name}</span>
											<span class="mat-meta">₹{mat.rate}/MT • HSN {mat.hsn} • {mat.taxRate}% GST</span>
										</div>
										<span class="mat-add-badge">+ Apply</span>
									</button>
								{/each}
							</div>
						</div>
					</div>
				{/if}

				<!-- Customer Details Card -->
				<div class="card customer-card">
					<div class="card-header">
						<h3>1. Customer Details & Place of Supply</h3>
						<span class="gst-state-badge">
							{isInterState ? 'IGST Applicable (Inter-State)' : 'CGST + SGST (Intra-State)'}
						</span>
					</div>

					<div class="form-row">
						<div class="form-group flex-2">
							<label for="custName">Customer / Company Name</label>
							<input
								id="custName"
								type="text"
								bind:value={customerName}
								placeholder="Enter customer name or select walk-in"
							/>
						</div>

						<div class="form-group flex-1">
							<label for="custPhone">Phone / WhatsApp</label>
							<input
								id="custPhone"
								type="text"
								bind:value={customerPhone}
								placeholder="+91 98..."
							/>
						</div>

						<div class="form-group flex-1">
							<label for="custGstin">GSTIN (Optional)</label>
							<input
								id="custGstin"
								type="text"
								bind:value={customerGstin}
								placeholder="27AABCA..."
							/>
						</div>

						<div class="form-group flex-1">
							<label for="supplyState">State of Supply</label>
							<select id="supplyState" bind:value={supplyState}>
								<option value="Maharashtra (27)">Maharashtra (27)</option>
								<option value="Gujarat (24)">Gujarat (24)</option>
								<option value="Karnataka (29)">Karnataka (29)</option>
								<option value="Delhi (07)">Delhi (07)</option>
								<option value="Tamil Nadu (33)">Tamil Nadu (33)</option>
								<option value="Telangana (36)">Telangana (36)</option>
								<option value="Uttar Pradesh (09)">Uttar Pradesh (09)</option>
							</select>
						</div>
					</div>
				</div>

				<!-- Items Table Card -->
				<div class="card items-card">
					<div class="card-header">
						<div class="items-header-left">
							<h3>2. Items & Services</h3>
							<div class="template-quick-picker">
								<label for="quickTmplSelect" class="small-label">Bill Template:</label>
								<select id="quickTmplSelect" class="small-select" bind:value={selectedTemplateId}>
									{#each templates as t}
										<option value={t.id}>{t.name}</option>
									{/each}
								</select>
							</div>
						</div>
						<button class="add-row-btn" onclick={addItem}>+ Add Item Line</button>
					</div>

					<div class="table-responsive">
						<table class="items-table">
							<thead>
								<tr>
									<th style="width: 35%;">Item Description</th>
									<th style="width: 10%;">HSN/SAC</th>
									<th style="width: 10%;">Qty</th>
									<th style="width: 10%;">Unit</th>
									<th style="width: 12%;">Price (₹)</th>
									<th style="width: 8%;">Disc %</th>
									<th style="width: 8%;">GST %</th>
									<th style="width: 12%;">Total (₹)</th>
									<th style="width: 5%;"></th>
								</tr>
							</thead>
							<tbody>
								{#each items as item, index}
									{@const q = parseFloat(item.quantity) || 0}
									{@const p = parseFloat(item.unitPrice) || 0}
									{@const d = parseFloat(item.discountRate) || 0}
									{@const t = parseFloat(item.taxRate) || 0}
									{@const lineTaxable = q * p - (q * p * d) / 100.0}
									{@const lineTotal = lineTaxable + (lineTaxable * t) / 100.0}
									<tr>
										<td>
											<input
												type="text"
												class="table-input"
												bind:value={item.itemDescription}
												placeholder="Item name / barcode scan"
											/>
										</td>
										<td>
											<input
												type="text"
												class="table-input center"
												bind:value={item.hsnSac}
												placeholder="HSN"
											/>
										</td>
										<td>
											<input
												type="number"
												min="1"
												class="table-input center"
												bind:value={item.quantity}
											/>
										</td>
										<td>
											<select class="table-input" bind:value={item.unitCode}>
												<option value="PCS">PCS</option>
												<option value="BOX">BOX</option>
												<option value="KG">KG</option>
												<option value="MTR">MTR</option>
												<option value="BTL">BTL</option>
												<option value="JAR">JAR</option>
											</select>
										</td>
										<td>
											<input
												type="number"
												step="0.01"
												class="table-input right"
												bind:value={item.unitPrice}
											/>
										</td>
										<td>
											<input
												type="number"
												min="0"
												max="100"
												class="table-input center"
												bind:value={item.discountRate}
											/>
										</td>
										<td>
											<select class="table-input center" bind:value={item.taxRate}>
												<option value="0">0%</option>
												<option value="5">5%</option>
												<option value="12">12%</option>
												<option value="18">18%</option>
												<option value="28">28%</option>
											</select>
										</td>
										<td class="line-total-cell right">
											₹{lineTotal.toFixed(2)}
										</td>
										<td class="center">
											<button
												class="del-row-btn"
												disabled={items.length === 1}
												onclick={() => removeItem(index)}
												title="Delete item"
											>
												✕
											</button>
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>

					<div class="table-footer-actions">
						<button class="add-row-btn-ghost" onclick={addItem}>
							+ Add Another Item
						</button>
						<span class="keyboard-hint">Tip: Enter barcode or type to autocomplete</span>
					</div>
				</div>
			</div>

			<!-- Right: Calculation & Instant Checkout -->
			<div class="bill-sidebar">
				<div class="card checkout-card">
					<div class="checkout-header">
						<h3>3. Payment & Settlement</h3>
						<span class="active-mode-badge">{paymentMode}</span>
					</div>

					<!-- Payment Mode Selector -->
					<div class="payment-mode-grid">
						<button
							class="pay-btn"
							class:selected={paymentMode === 'UPI'}
							onclick={() => {
								paymentMode = 'UPI';
								tenderedAmount = String(payableAmount);
							}}
						>
							📱 UPI QR
						</button>
						<button
							class="pay-btn"
							class:selected={paymentMode === 'CASH'}
							onclick={() => {
								paymentMode = 'CASH';
								tenderedAmount = String(payableAmount);
							}}
						>
							💵 Cash
						</button>
						<button
							class="pay-btn"
							class:selected={paymentMode === 'CARD'}
							onclick={() => {
								paymentMode = 'CARD';
								tenderedAmount = String(payableAmount);
							}}
						>
							💳 Card
						</button>
						<button
							class="pay-btn"
							class:selected={paymentMode === 'CREDIT'}
							onclick={() => {
								paymentMode = 'CREDIT';
								tenderedAmount = '0';
							}}
						>
							⏳ Credit / Due
						</button>
					</div>

					<!-- Dynamic UPI QR Scanner Display -->
					{#if paymentMode === 'UPI'}
						<div class="upi-box">
							<div class="upi-qr-wrapper">
								<div class="qr-placeholder">
									<svg viewBox="0 0 100 100" class="qr-svg">
										<rect x="5" y="5" width="30" height="30" fill="#0f172a" />
										<rect x="10" y="10" width="20" height="20" fill="#ffffff" />
										<rect x="14" y="14" width="12" height="12" fill="#0f172a" />

										<rect x="65" y="5" width="30" height="30" fill="#0f172a" />
										<rect x="70" y="10" width="20" height="20" fill="#ffffff" />
										<rect x="74" y="14" width="12" height="12" fill="#0f172a" />

										<rect x="5" y="65" width="30" height="30" fill="#0f172a" />
										<rect x="10" y="70" width="20" height="20" fill="#ffffff" />
										<rect x="14" y="74" width="12" height="12" fill="#0f172a" />

										<rect x="42" y="10" width="16" height="80" fill="#38bdf8" />
										<rect x="10" y="42" width="80" height="16" fill="#38bdf8" />
									</svg>
								</div>
								<div class="upi-text">
									<strong>Scan to Pay ₹{payableAmount}</strong>
									<span class="upi-id">UPI ID: mrbiz@upi</span>
								</div>
							</div>
						</div>
					{/if}

					<!-- Settlement Input -->
					<div class="tendered-group">
						<label for="tenderedCash">Amount Received (₹)</label>
						<input
							id="tenderedCash"
							type="number"
							step="1"
							bind:value={tenderedAmount}
							placeholder={String(payableAmount)}
						/>
					</div>

					<div class="balance-calc-row">
						{#if paymentMode !== 'CREDIT' && parseFloat(tenderedAmount) > payableAmount}
							<div class="change-tag return">
								<span>Change to Return:</span>
								<strong>₹{calculatedChange}</strong>
							</div>
						{:else if calculatedDue !== '0.00'}
							<div class="change-tag due">
								<span>Balance Remaining:</span>
								<strong>₹{calculatedDue}</strong>
							</div>
						{:else}
							<div class="change-tag full">
								<span>Payment Status:</span>
								<strong>Fully Settled</strong>
							</div>
						{/if}
					</div>

					<!-- Bill Breakdown Summary -->
					<div class="summary-breakdown">
						<div class="sum-row">
							<span>Gross Subtotal</span>
							<span>₹{grossSubtotal.toFixed(2)}</span>
						</div>
						<div class="sum-row">
							<span>Total Discount</span>
							<span class="disc">-₹{totalDiscount.toFixed(2)}</span>
						</div>
						<div class="sum-row">
							<span>Taxable Value</span>
							<span>₹{taxableAmount.toFixed(2)}</span>
						</div>
						{#if isInterState}
							<div class="sum-row">
								<span>IGST (Integrated Tax)</span>
								<span>₹{totalTax.toFixed(2)}</span>
							</div>
						{:else}
							<div class="sum-row">
								<span>CGST (Central Tax)</span>
								<span>₹{(totalTax / 2).toFixed(2)}</span>
							</div>
							<div class="sum-row">
								<span>SGST (State Tax)</span>
								<span>₹{(totalTax / 2).toFixed(2)}</span>
							</div>
						{/if}
						<div class="sum-row">
							<span>Round Off</span>
							<span>₹{roundOff}</span>
						</div>

						<div class="sum-total-row">
							<span>Net Payable</span>
							<span class="big-total">₹{payableAmount}</span>
						</div>
					</div>

					<!-- Fast Execution Action Buttons -->
					<div class="checkout-actions">
						<button class="primary-checkout-btn" disabled={loading} onclick={generateBill}>
							⚡ Generate & Save Bill
						</button>

						<button class="preview-btn" onclick={() => openInvoiceTemplatePreview()}>
							👁️ Live Template Preview ({selectedTemplateId})
						</button>

						<div class="action-btn-row">
							<button class="secondary-btn" onclick={printFormattedInvoice}>
								🖨️ 80mm Print
							</button>
							<button class="whatsapp-btn" onclick={() => shareBillOnWhatsApp()}>
								💬 WhatsApp Bill
							</button>
						</div>
					</div>
				</div>
			</div>
		</div>
	{/if}

	<!-- TAB 2: INVOICES & PAYMENT RECEIPTS -->
	{#if activeTab === 'invoices'}
		<div class="invoices-view">
			<div class="invoices-toolbar">
				<div class="filter-group">
					<button
						class="filter-pill"
						class:active={statusFilter === 'ALL'}
						onclick={() => (statusFilter = 'ALL')}
					>
						All ({invoices.length})
					</button>
					<button
						class="filter-pill"
						class:active={statusFilter === 'PAID'}
						onclick={() => (statusFilter = 'PAID')}
					>
						Paid
					</button>
					<button
						class="filter-pill"
						class:active={statusFilter === 'PARTIAL'}
						onclick={() => (statusFilter = 'PARTIAL')}
					>
						Partial
					</button>
					<button
						class="filter-pill"
						class:active={statusFilter === 'UNPAID'}
						onclick={() => (statusFilter = 'UNPAID')}
					>
						Unpaid
					</button>
					<button
						class="filter-pill"
						class:active={statusFilter === 'OVERDUE'}
						onclick={() => (statusFilter = 'OVERDUE')}
					>
						Overdue
					</button>
				</div>

				<button class="new-bill-shortcut" onclick={() => (activeTab = 'quickbill')}>
					+ New Quick Bill
				</button>
			</div>

			<div class="card table-card">
				<div class="table-responsive">
					<table class="data-table">
						<thead>
							<tr>
								<th>Bill #</th>
								<th>Customer</th>
								<th>Date</th>
								<th>Total Amount</th>
								<th>Paid</th>
								<th>Balance Due</th>
								<th>Template</th>
								<th>Status</th>
								<th>Actions</th>
							</tr>
						</thead>
						<tbody>
							{#each filteredInvoices as inv}
								<tr>
									<td class="bold">{inv.invoiceNumber}</td>
									<td>
										<div class="cust-cell">
											<span class="cust-title">{inv.customerName}</span>
											{#if inv.customerGstin}
												<span class="cust-gstin">{inv.customerGstin}</span>
											{/if}
										</div>
									</td>
									<td>{inv.invoiceDate}</td>
									<td class="bold">₹{inv.grandTotal}</td>
									<td class="green">₹{inv.amountPaid}</td>
									<td class={parseFloat(inv.balanceDue) > 0 ? 'red bold' : 'muted'}>
										₹{inv.balanceDue}
									</td>
									<td>
										<span class="tmpl-chip">T-{inv.templateId || '1'}</span>
									</td>
									<td>
										<span class="status-badge {inv.status.toLowerCase()}">
											{inv.status}
										</span>
									</td>
									<td>
										<div class="actions-cell">
											<button
												class="icon-action-btn view"
												onclick={() => openInvoiceTemplatePreview(inv)}
												title="View Full Formatted Template"
											>
												👁️
											</button>
											{#if parseFloat(inv.balanceDue) > 0}
												<button
													class="record-pay-btn"
													onclick={() => openPaymentModal(inv)}
													title="Record Payment"
												>
													💳 Pay
												</button>
											{/if}
											<button
												class="icon-action-btn"
												onclick={() => shareBillOnWhatsApp(inv)}
												title="Share on WhatsApp"
											>
												💬
											</button>
											<button
												class="icon-action-btn"
												onclick={() => openInvoiceTemplatePreview(inv)}
												title="Print Formatted Invoice"
											>
												🖨️
											</button>
										</div>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</div>
		</div>
	{/if}

	<!-- TAB 3: CUSTOMER RECEIVABLES & KHATA -->
	{#if activeTab === 'khata'}
		<div class="khata-view">
			<div class="card table-card">
				<div class="card-header">
					<h3>Customer Outstanding Khata Ledger</h3>
					<span class="sub-text">Real-time balances across customers</span>
				</div>

				<div class="table-responsive">
					<table class="data-table">
						<thead>
							<tr>
								<th>Customer Name</th>
								<th>Phone</th>
								<th>Total Invoiced</th>
								<th>Total Collected</th>
								<th>Outstanding Balance Due</th>
								<th>Unpaid Bills</th>
								<th>Payment Reminder</th>
							</tr>
						</thead>
						<tbody>
							{#each summary.topReceivables as rec}
								<tr>
									<td class="bold">{rec.customerName}</td>
									<td>{rec.phone}</td>
									<td>₹{rec.totalBilled}</td>
									<td class="green">₹{rec.totalPaid}</td>
									<td class="red bold">₹{rec.balanceDue}</td>
									<td>
										<span class="unpaid-badge">{rec.unpaidInvoicesCount} Bills</span>
									</td>
									<td>
										<button
											class="whatsapp-nudge-btn"
											onclick={() => {
												const msg = `Dear ${rec.customerName}, a gentle reminder that your pending balance of ₹${rec.balanceDue} is due. You can pay via UPI: mrbiz@upi. Thank you!`;
												window.open(`https://wa.me/?text=${encodeURIComponent(msg)}`, '_blank');
											}}
										>
											💬 Send WhatsApp Reminder
										</button>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</div>
		</div>
	{/if}

	<!-- TAB 4: 17 BILLING TEMPLATES & OPTIONS -->
	{#if activeTab === 'templates'}
		<div class="templates-hub">
			<!-- Top Options & Preferences Workbench -->
			<div class="card template-options-card">
				<div class="card-header">
					<div>
						<h3>🎨 Global Billing Template & Print Preferences</h3>
						<span class="sub-text">Configure print parameters, bank visibility, and branding across your 17 templates</span>
					</div>
					<button class="save-tmpl-btn" onclick={saveTemplateConfig}>
						💾 Save Preferences
					</button>
				</div>

				<div class="options-grid">
					<div class="opt-col">
						<label for="defaultTmplSelect">Active Template</label>
						<select id="defaultTmplSelect" bind:value={selectedTemplateId}>
							{#each templates as t}
								<option value={t.id}>{t.name} ({t.badge})</option>
							{/each}
						</select>
					</div>

					<div class="opt-col">
						<label for="defaultPaperSelect">Paper Size</label>
						<select id="defaultPaperSelect" bind:value={selectedPaperSize}>
							<option value="A4">A4 Standard Sheet</option>
							<option value="A5">A5 Portrait Sheet</option>
							<option value="A5_LANDSCAPE">A5 Landscape Sheet</option>
						</select>
					</div>

					<div class="opt-col">
						<label for="defaultCopySelect">Default Copy Type</label>
						<select id="defaultCopySelect" bind:value={selectedCopyType}>
							<option value="ORIGINAL">Original for Recipient</option>
							<option value="DUPLICATE">Duplicate for Transporter</option>
							<option value="TRIPLICATE">Triplicate for Supplier</option>
							<option value="OFFICE">Office Copy</option>
						</select>
					</div>

					<div class="opt-col">
						<label for="defaultTaxMode">Tax Mode</label>
						<select id="defaultTaxMode" bind:value={selectedTaxMode}>
							<option value="WITH_GST">Tax Invoice (With GST)</option>
							<option value="WITHOUT_GST">Bill of Supply (Composition/Non-GST)</option>
						</select>
					</div>

					<div class="opt-col">
						<label for="brandColorInput">Accent / Brand Color</label>
						<div class="color-picker-row">
							<input id="brandColorInput" type="color" bind:value={brandColor} />
							<span class="color-code">{brandColor}</span>
						</div>
					</div>
				</div>

				<!-- Toggles Row -->
				<div class="toggles-row">
					<label class="checkbox-label">
						<input type="checkbox" bind:checked={showBankDetails} />
						<span>Show Bank Account & IFSC</span>
					</label>

					<label class="checkbox-label">
						<input type="checkbox" bind:checked={showUPIQR} />
						<span>Show Dynamic UPI QR Code</span>
					</label>

					<label class="checkbox-label">
						<input type="checkbox" bind:checked={showTaxSummary} />
						<span>Show GST Slab Tax Summary</span>
					</label>
				</div>
			</div>

			<!-- All 17 Templates Gallery Grid -->
			<div class="templates-gallery-grid">
				{#each templates as t}
					<div
						class="template-card"
						class:active={selectedTemplateId === t.id}
						role="button"
						tabindex="0"
						onclick={() => (selectedTemplateId = t.id)}
						onkeydown={(e) => {
							if (e.key === 'Enter' || e.key === ' ') {
								selectedTemplateId = t.id;
							}
						}}
					>
						<div class="tmpl-card-header">
							<span class="tmpl-badge">{t.badge}</span>
							<span class="tmpl-category">{t.category}</span>
						</div>
						<h4 class="tmpl-title">{t.name}</h4>
						<p class="tmpl-desc">{t.desc}</p>
						<div class="tmpl-footer">
							<button
								class="select-tmpl-btn"
								class:selected={selectedTemplateId === t.id}
								onclick={(e) => {
									e.stopPropagation();
									selectedTemplateId = t.id;
								}}
							>
								{selectedTemplateId === t.id ? '✓ Selected Active' : 'Use Template'}
							</button>

							<button
								class="preview-tmpl-btn"
								onclick={(e) => {
									e.stopPropagation();
									selectedTemplateId = t.id;
									openInvoiceTemplatePreview();
								}}
							>
								👁️ Preview
							</button>
						</div>
					</div>
				{/each}
			</div>
		</div>
	{/if}

	<!-- LIVE INVOICE TEMPLATE PREVIEW & PRINT MODAL -->
	{#if showTemplatePreviewModal && previewInvoiceData}
		<div class="modal-backdrop">
			<div class="modal-card-full">
				<div class="modal-header-preview">
					<div class="preview-header-left">
						<h3>👁️ Formatted Template Preview: {templates.find(t => t.id === selectedTemplateId)?.name}</h3>
						<div class="modal-controls">
							<select class="control-select" bind:value={selectedTemplateId}>
								{#each templates as t}
									<option value={t.id}>{t.name}</option>
								{/each}
							</select>

							<select class="control-select" bind:value={selectedPaperSize}>
								<option value="A4">A4 Sheet</option>
								<option value="A5">A5 Portrait</option>
								<option value="A5_LANDSCAPE">A5 Landscape</option>
							</select>

							<select class="control-select" bind:value={selectedCopyType}>
								<option value="ORIGINAL">Original</option>
								<option value="DUPLICATE">Duplicate</option>
								<option value="TRIPLICATE">Triplicate</option>
								<option value="OFFICE">Office Copy</option>
							</select>

							<select class="control-select" bind:value={selectedTaxMode}>
								<option value="WITH_GST">With GST</option>
								<option value="WITHOUT_GST">Bill of Supply</option>
							</select>
						</div>
					</div>

					<div class="preview-header-actions">
						<button class="print-dialog-btn" onclick={printFormattedInvoice}>
							🖨️ Print Document
						</button>
						<button
							class="whatsapp-modal-btn"
							onclick={() => {
								const text = `Bill from MrBiz-OS: ${previewInvoiceData?.invoice_number} | Total: ₹${previewInvoiceData?.grand_total}`;
								window.open(`https://wa.me/?text=${encodeURIComponent(text)}`, '_blank');
							}}
						>
							💬 WhatsApp
						</button>
						<button class="modal-close" onclick={() => (showTemplatePreviewModal = false)}>✕</button>
					</div>
				</div>

				<!-- Render the full InvoiceTemplates component -->
				<div class="template-render-wrapper">
					<InvoiceTemplates
						invoice={previewInvoiceData}
						{company}
						templateId={selectedTemplateId}
						copyType={selectedCopyType}
						taxMode={selectedTaxMode}
						paperSize={selectedPaperSize}
						extraDetails={{
							showBankDetails,
							showUPIQR,
							showTaxSummary,
							signatoryTitle,
							terms: defaultTerms,
							jurisdictionCity: defaultJurisdiction,
							interestRate: defaultInterestRate,
							returnPolicy: defaultReturnPolicy,
							vehicleNumber: isQuarryMode ? lorryNumber : (previewInvoiceData?.vehicle_number || ''),
							deliveryPlace: isQuarryMode ? deliveryPlace : (previewInvoiceData?.delivery_place || ''),
							grossWeight: isQuarryMode ? grossWeightKg + ' kg' : (previewInvoiceData?.gross_weight || ''),
							netWeight: isQuarryMode ? netWeightTons + ' MT' : (previewInvoiceData?.net_weight || ''),
							poNumber: isQuarryMode ? poNumber : (previewInvoiceData?.po_number || ''),
							lrNumber: isQuarryMode ? lrNumber : (previewInvoiceData?.lr_number || ''),
							ewayBill: isQuarryMode ? ewayBill : (previewInvoiceData?.eway_bill || '')
						}}
					/>
				</div>
			</div>
		</div>
	{/if}

	<!-- Record Payment Modal -->
	{#if showPayModal && selectedInvoiceForPay}
		<div class="modal-backdrop">
			<div class="modal-card">
				<div class="modal-header">
					<h3>Record Payment for {selectedInvoiceForPay.invoiceNumber}</h3>
					<button class="modal-close" onclick={() => (showPayModal = false)}>✕</button>
				</div>

				{#if payMessage}
					<div class="modal-error">{payMessage}</div>
				{/if}

				<div class="modal-body">
					<div class="modal-info-row">
						<span>Customer:</span>
						<strong>{selectedInvoiceForPay.customerName}</strong>
					</div>
					<div class="modal-info-row">
						<span>Total Bill:</span>
						<strong>₹{selectedInvoiceForPay.grandTotal}</strong>
					</div>
					<div class="modal-info-row">
						<span>Balance Due:</span>
						<strong class="red">₹{selectedInvoiceForPay.balanceDue}</strong>
					</div>

					<div class="form-group" style="margin-top: 1rem;">
						<label for="payAmount">Amount Paid (₹)</label>
						<input id="payAmount" type="number" bind:value={payAmount} />
					</div>

					<div class="form-group">
						<label for="payMode">Payment Mode</label>
						<select id="payMode" bind:value={payMode}>
							<option value="UPI">UPI</option>
							<option value="CASH">Cash</option>
							<option value="CARD">Debit / Credit Card</option>
							<option value="NETBANKING">Net Banking</option>
							<option value="CHEQUE">Cheque</option>
						</select>
					</div>

					<div class="form-group">
						<label for="payRef">Reference / Transaction ID</label>
						<input id="payRef" type="text" bind:value={payRef} placeholder="UTR / Ref Number" />
					</div>
				</div>

				<div class="modal-footer">
					<button class="secondary-btn" onclick={() => (showPayModal = false)}>Cancel</button>
					<button class="primary-checkout-btn" onclick={submitRecordPayment}>
						Confirm Payment
					</button>
				</div>
			</div>
		</div>
	{/if}
</div>

<style>
	.billing-container {
		padding: 1.5rem;
		max-width: 1400px;
		margin: 0 auto;
		color: #e2e8f0;
	}

	.alert-banner {
		background: #065f46;
		color: #a7f3d0;
		padding: 0.75rem 1rem;
		border-radius: 8px;
		margin-bottom: 1rem;
		display: flex;
		justify-content: space-between;
		align-items: center;
		font-weight: 600;
	}

	.banner-close {
		background: transparent;
		border: none;
		color: inherit;
		cursor: pointer;
		font-size: 1rem;
	}

	.billing-header {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		margin-bottom: 1.5rem;
		gap: 1rem;
		flex-wrap: wrap;
	}

	.title-row {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		flex-wrap: wrap;
	}

	h1 {
		font-size: 1.625rem;
		font-weight: 800;
		color: #ffffff;
		margin: 0;
	}

	.badge-tag {
		background: rgba(56, 189, 248, 0.15);
		color: #38bdf8;
		padding: 0.2rem 0.6rem;
		border-radius: 6px;
		font-size: 0.75rem;
		font-weight: 700;
	}

	.badge-template {
		background: rgba(168, 85, 247, 0.15);
		color: #c084fc;
		padding: 0.2rem 0.6rem;
		border-radius: 6px;
		font-size: 0.75rem;
		font-weight: 700;
	}

	.badge-mode {
		background: rgba(16, 185, 129, 0.15);
		color: #34d399;
		padding: 0.2rem 0.6rem;
		border-radius: 6px;
		font-size: 0.75rem;
		font-weight: 700;
	}

	.subtitle {
		color: #94a3b8;
		font-size: 0.875rem;
		margin-top: 0.25rem;
	}

	.header-actions {
		display: flex;
		gap: 0.75rem;
		align-items: center;
	}

	.mode-toggle-btn {
		background: #1e293b;
		color: #94a3b8;
		border: 1px solid #334155;
		padding: 0.5rem 1rem;
		border-radius: 8px;
		font-weight: 600;
		cursor: pointer;
		transition: all 0.2s;
	}

	.mode-toggle-btn.active {
		background: rgba(16, 185, 129, 0.15);
		color: #34d399;
		border-color: #059669;
	}

	.walkin-btn {
		background: #0284c7;
		color: #ffffff;
		border: none;
		padding: 0.5rem 1rem;
		border-radius: 8px;
		font-weight: 600;
		cursor: pointer;
	}

	.metrics-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
		gap: 1rem;
		margin-bottom: 1.5rem;
	}

	.kpi-card {
		background: #0f172a;
		border: 1px solid #1e293b;
		padding: 1rem 1.25rem;
		border-radius: 12px;
		display: flex;
		align-items: center;
		gap: 1rem;
	}

	.kpi-icon {
		font-size: 2rem;
	}

	.kpi-info {
		display: flex;
		flex-direction: column;
	}

	.kpi-label {
		font-size: 0.75rem;
		color: #94a3b8;
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}

	.kpi-value {
		font-size: 1.375rem;
		font-weight: 800;
		color: #ffffff;
	}

	.kpi-value.warning {
		color: #f59e0b;
	}

	.kpi-value .sub {
		font-size: 0.875rem;
		color: #94a3b8;
		font-weight: 500;
	}

	.billing-tabs {
		display: flex;
		gap: 0.5rem;
		border-bottom: 1px solid #334155;
		margin-bottom: 1.5rem;
		overflow-x: auto;
	}

	.tab-btn {
		background: transparent;
		border: none;
		color: #94a3b8;
		padding: 0.75rem 1.25rem;
		font-size: 0.9375rem;
		font-weight: 600;
		cursor: pointer;
		border-bottom: 2px solid transparent;
		transition: all 0.2s;
		white-space: nowrap;
	}

	.tab-btn.active {
		color: #38bdf8;
		border-bottom-color: #38bdf8;
	}

	.bill-workbench {
		display: grid;
		grid-template-columns: 1fr 380px;
		gap: 1.5rem;
		align-items: start;
	}

	@media (max-width: 1024px) {
		.bill-workbench {
			grid-template-columns: 1fr;
		}
	}

	.card {
		background: #0f172a;
		border: 1px solid #1e293b;
		border-radius: 12px;
		padding: 1.25rem;
		margin-bottom: 1.5rem;
	}

	.card-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 1rem;
		flex-wrap: wrap;
		gap: 0.5rem;
	}

	.items-header-left {
		display: flex;
		align-items: center;
		gap: 1rem;
		flex-wrap: wrap;
	}

	.template-quick-picker {
		display: flex;
		align-items: center;
		gap: 0.4rem;
	}

	.small-label {
		font-size: 0.75rem;
		color: #94a3b8;
	}

	.small-select {
		background: #1e293b;
		border: 1px solid #334155;
		color: #38bdf8;
		font-size: 0.75rem;
		font-weight: 600;
		padding: 0.2rem 0.5rem;
		border-radius: 4px;
	}

	.card-header h3 {
		font-size: 1rem;
		font-weight: 700;
		color: #ffffff;
		margin: 0;
	}

	.gst-state-badge {
		font-size: 0.75rem;
		color: #38bdf8;
		background: rgba(56, 189, 248, 0.1);
		padding: 0.2rem 0.5rem;
		border-radius: 4px;
	}

	.form-row {
		display: flex;
		gap: 1rem;
		flex-wrap: wrap;
	}

	.form-group {
		display: flex;
		flex-direction: column;
		gap: 0.4rem;
	}

	.flex-1 {
		flex: 1;
		min-width: 160px;
	}

	.flex-2 {
		flex: 2;
		min-width: 220px;
	}

	label {
		font-size: 0.75rem;
		color: #94a3b8;
		font-weight: 600;
	}

	input, select {
		background: #1e293b;
		border: 1px solid #334155;
		color: #ffffff;
		padding: 0.5rem 0.75rem;
		border-radius: 6px;
		font-size: 0.875rem;
	}

	.table-responsive {
		overflow-x: auto;
	}

	.items-table, .data-table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.875rem;
	}

	th {
		text-align: left;
		padding: 0.6rem 0.75rem;
		color: #94a3b8;
		font-size: 0.75rem;
		text-transform: uppercase;
		border-bottom: 1px solid #334155;
	}

	td {
		padding: 0.6rem 0.75rem;
		border-bottom: 1px solid #1e293b;
	}

	.table-input {
		width: 100%;
		box-sizing: border-box;
		background: #1e293b;
		border: 1px solid #334155;
		color: #ffffff;
		padding: 0.4rem 0.5rem;
		border-radius: 4px;
	}

	.center {
		text-align: center;
	}

	.right {
		text-align: right;
	}

	.line-total-cell {
		font-weight: 700;
		color: #ffffff;
	}

	.del-row-btn {
		background: transparent;
		border: none;
		color: #ef4444;
		cursor: pointer;
		font-weight: 700;
	}

	.del-row-btn:disabled {
		opacity: 0.3;
		cursor: not-allowed;
	}

	.table-footer-actions {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-top: 1rem;
	}

	.add-row-btn {
		background: #0284c7;
		color: #ffffff;
		border: none;
		padding: 0.4rem 0.75rem;
		border-radius: 6px;
		font-weight: 600;
		cursor: pointer;
		font-size: 0.8125rem;
	}

	.add-row-btn-ghost {
		background: transparent;
		color: #38bdf8;
		border: 1px dashed #38bdf8;
		padding: 0.4rem 0.75rem;
		border-radius: 6px;
		font-weight: 600;
		cursor: pointer;
	}

	.keyboard-hint {
		font-size: 0.75rem;
		color: #64748b;
	}

	.checkout-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 1rem;
	}

	.active-mode-badge {
		background: #0284c7;
		color: #ffffff;
		font-size: 0.75rem;
		padding: 0.2rem 0.5rem;
		border-radius: 4px;
		font-weight: 700;
	}

	.payment-mode-grid {
		display: grid;
		grid-template-columns: repeat(2, 1fr);
		gap: 0.5rem;
		margin-bottom: 1rem;
	}

	.pay-btn {
		background: #1e293b;
		border: 1px solid #334155;
		color: #cbd5e1;
		padding: 0.6rem;
		border-radius: 8px;
		font-weight: 700;
		cursor: pointer;
		transition: all 0.2s;
	}

	.pay-btn.selected {
		background: #0284c7;
		color: #ffffff;
		border-color: #38bdf8;
	}

	.upi-box {
		background: #021a36;
		border: 1px dashed #38bdf8;
		border-radius: 8px;
		padding: 0.75rem;
		margin-bottom: 1rem;
	}

	.upi-qr-wrapper {
		display: flex;
		align-items: center;
		gap: 1rem;
	}

	.qr-placeholder {
		width: 70px;
		height: 70px;
		background: #ffffff;
		border-radius: 6px;
		padding: 4px;
	}

	.qr-svg {
		width: 100%;
		height: 100%;
	}

	.upi-text {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}

	.upi-id {
		font-size: 0.75rem;
		color: #94a3b8;
	}

	.tendered-group {
		margin-bottom: 0.75rem;
	}

	.tendered-group input {
		width: 100%;
		box-sizing: border-box;
		font-size: 1.125rem;
		font-weight: 700;
		text-align: right;
	}

	.balance-calc-row {
		margin-bottom: 1rem;
	}

	.change-tag {
		display: flex;
		justify-content: space-between;
		padding: 0.5rem 0.75rem;
		border-radius: 6px;
		font-size: 0.875rem;
	}

	.change-tag.return {
		background: rgba(16, 185, 129, 0.15);
		color: #34d399;
	}

	.change-tag.due {
		background: rgba(245, 158, 11, 0.15);
		color: #fbbf24;
	}

	.change-tag.full {
		background: rgba(56, 189, 248, 0.15);
		color: #38bdf8;
	}

	.summary-breakdown {
		border-top: 1px solid #1e293b;
		padding-top: 0.75rem;
		margin-bottom: 1.25rem;
	}

	.sum-row {
		display: flex;
		justify-content: space-between;
		font-size: 0.8125rem;
		color: #94a3b8;
		margin-bottom: 0.35rem;
	}

	.sum-row .disc {
		color: #34d399;
	}

	.sum-total-row {
		display: flex;
		justify-content: space-between;
		align-items: center;
		border-top: 1px dashed #334155;
		padding-top: 0.6rem;
		margin-top: 0.5rem;
		color: #ffffff;
		font-weight: 800;
	}

	.big-total {
		font-size: 1.5rem;
		color: #38bdf8;
	}

	.checkout-actions {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.primary-checkout-btn {
		background: #0284c7;
		color: #ffffff;
		border: none;
		padding: 0.75rem;
		border-radius: 8px;
		font-weight: 700;
		font-size: 1rem;
		cursor: pointer;
		transition: background 0.2s;
	}

	.primary-checkout-btn:hover {
		background: #0369a1;
	}

	.preview-btn {
		background: rgba(168, 85, 247, 0.15);
		border: 1px solid rgba(168, 85, 247, 0.4);
		color: #c084fc;
		padding: 0.65rem;
		border-radius: 8px;
		font-weight: 700;
		cursor: pointer;
		transition: all 0.2s;
	}

	.preview-btn:hover {
		background: rgba(168, 85, 247, 0.25);
	}

	.action-btn-row {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 0.5rem;
	}

	.secondary-btn {
		background: #1e293b;
		border: 1px solid #334155;
		color: #ffffff;
		padding: 0.6rem;
		border-radius: 8px;
		font-weight: 600;
		cursor: pointer;
	}

	.whatsapp-btn {
		background: #166534;
		border: none;
		color: #ffffff;
		padding: 0.6rem;
		border-radius: 8px;
		font-weight: 600;
		cursor: pointer;
	}

	/* Invoices Tab */
	.invoices-toolbar {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 1rem;
		flex-wrap: wrap;
		gap: 0.75rem;
	}

	.filter-group {
		display: flex;
		gap: 0.4rem;
	}

	.filter-pill {
		background: #1e293b;
		border: 1px solid #334155;
		color: #94a3b8;
		padding: 0.35rem 0.75rem;
		border-radius: 9999px;
		font-size: 0.8125rem;
		cursor: pointer;
	}

	.filter-pill.active {
		background: #0284c7;
		color: #ffffff;
		border-color: #38bdf8;
	}

	.new-bill-shortcut {
		background: #0284c7;
		color: #ffffff;
		border: none;
		padding: 0.4rem 0.8rem;
		border-radius: 6px;
		font-weight: 600;
		cursor: pointer;
	}

	.bold {
		font-weight: 700;
		color: #ffffff;
	}

	.green {
		color: #34d399;
	}

	.red {
		color: #f87171;
	}

	.muted {
		color: #64748b;
	}

	.cust-cell {
		display: flex;
		flex-direction: column;
	}

	.cust-title {
		font-weight: 600;
	}

	.cust-gstin {
		font-size: 0.75rem;
		color: #94a3b8;
	}

	.tmpl-chip {
		background: #1e293b;
		border: 1px solid #334155;
		color: #a5b4fc;
		font-size: 0.75rem;
		padding: 0.15rem 0.4rem;
		border-radius: 4px;
		font-weight: 600;
	}

	.status-badge {
		padding: 0.2rem 0.5rem;
		border-radius: 4px;
		font-size: 0.75rem;
		font-weight: 700;
	}

	.status-badge.paid {
		background: rgba(16, 185, 129, 0.15);
		color: #34d399;
	}

	.status-badge.partial {
		background: rgba(245, 158, 11, 0.15);
		color: #fbbf24;
	}

	.status-badge.unpaid {
		background: rgba(239, 68, 68, 0.15);
		color: #f87171;
	}

	.status-badge.overdue {
		background: rgba(239, 68, 68, 0.25);
		color: #ef4444;
	}

	.actions-cell {
		display: flex;
		gap: 0.4rem;
	}

	.record-pay-btn {
		background: #0284c7;
		border: none;
		color: #ffffff;
		padding: 0.25rem 0.5rem;
		border-radius: 4px;
		font-size: 0.75rem;
		cursor: pointer;
	}

	.icon-action-btn {
		background: #1e293b;
		border: 1px solid #334155;
		color: #ffffff;
		padding: 0.25rem 0.5rem;
		border-radius: 4px;
		cursor: pointer;
	}

	.icon-action-btn.view {
		background: rgba(168, 85, 247, 0.15);
		border-color: rgba(168, 85, 247, 0.3);
		color: #c084fc;
	}

	.unpaid-badge {
		background: rgba(239, 68, 68, 0.15);
		color: #f87171;
		padding: 0.2rem 0.5rem;
		border-radius: 4px;
		font-size: 0.75rem;
	}

	.whatsapp-nudge-btn {
		background: #166534;
		border: none;
		color: #ffffff;
		padding: 0.35rem 0.75rem;
		border-radius: 6px;
		font-size: 0.75rem;
		font-weight: 600;
		cursor: pointer;
	}

	/* TAB 4: TEMPLATES HUB */
	.template-options-card {
		margin-bottom: 2rem;
	}

	.options-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
		gap: 1rem;
		margin-bottom: 1.25rem;
	}

	.opt-col {
		display: flex;
		flex-direction: column;
		gap: 0.4rem;
	}

	.color-picker-row {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}

	.color-picker-row input[type="color"] {
		width: 44px;
		height: 38px;
		padding: 2px;
		cursor: pointer;
	}

	.color-code {
		font-family: monospace;
		font-weight: 700;
		color: #38bdf8;
	}

	.toggles-row {
		display: flex;
		gap: 1.5rem;
		flex-wrap: wrap;
		border-top: 1px solid #1e293b;
		padding-top: 1rem;
	}

	.checkbox-label {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 0.8125rem;
		color: #cbd5e1;
		cursor: pointer;
	}

	.save-tmpl-btn {
		background: #0284c7;
		color: #ffffff;
		border: none;
		padding: 0.5rem 1rem;
		border-radius: 6px;
		font-weight: 700;
		cursor: pointer;
	}

	.templates-gallery-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
		gap: 1.25rem;
	}

	.template-card {
		background: #0f172a;
		border: 1px solid #1e293b;
		border-radius: 12px;
		padding: 1.25rem;
		display: flex;
		flex-direction: column;
		cursor: pointer;
		transition: all 0.2s;
	}

	.template-card:hover {
		border-color: #38bdf8;
		transform: translateY(-2px);
	}

	.template-card.active {
		border-color: #38bdf8;
		background: rgba(2, 132, 199, 0.05);
		box-shadow: 0 0 0 1px #38bdf8;
	}

	.tmpl-card-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 0.5rem;
	}

	.tmpl-badge {
		background: rgba(168, 85, 247, 0.15);
		color: #c084fc;
		font-size: 0.75rem;
		font-weight: 700;
		padding: 0.15rem 0.5rem;
		border-radius: 4px;
	}

	.tmpl-category {
		font-size: 0.75rem;
		color: #94a3b8;
	}

	.tmpl-title {
		font-size: 1rem;
		font-weight: 700;
		color: #ffffff;
		margin: 0.25rem 0 0.5rem 0;
	}

	.tmpl-desc {
		font-size: 0.8125rem;
		color: #94a3b8;
		line-height: 1.4;
		flex: 1;
		margin: 0 0 1rem 0;
	}

	.tmpl-footer {
		display: flex;
		gap: 0.5rem;
	}

	.select-tmpl-btn {
		flex: 1;
		background: #1e293b;
		border: 1px solid #334155;
		color: #cbd5e1;
		padding: 0.4rem;
		border-radius: 6px;
		font-size: 0.8125rem;
		font-weight: 600;
		cursor: pointer;
	}

	.select-tmpl-btn.selected {
		background: #0284c7;
		color: #ffffff;
		border-color: #38bdf8;
	}

	.preview-tmpl-btn {
		background: rgba(56, 189, 248, 0.1);
		border: 1px solid rgba(56, 189, 248, 0.3);
		color: #38bdf8;
		padding: 0.4rem 0.75rem;
		border-radius: 6px;
		font-size: 0.8125rem;
		font-weight: 600;
		cursor: pointer;
	}

	/* Modal Common */
	.modal-backdrop {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.8);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 1000;
		padding: 1rem;
	}

	.modal-card {
		background: #0f172a;
		border: 1px solid #334155;
		border-radius: 12px;
		width: 100%;
		max-width: 440px;
		padding: 1.5rem;
	}

	.modal-card-full {
		background: #0b1120;
		border: 1px solid #334155;
		border-radius: 12px;
		width: 100%;
		max-width: 1100px;
		height: 90vh;
		display: flex;
		flex-direction: column;
		overflow: hidden;
	}

	.modal-header-preview {
		padding: 0.875rem 1.25rem;
		background: #0f172a;
		border-bottom: 1px solid #334155;
		display: flex;
		justify-content: space-between;
		align-items: center;
		flex-wrap: wrap;
		gap: 0.75rem;
	}

	.preview-header-left {
		display: flex;
		align-items: center;
		gap: 1rem;
		flex-wrap: wrap;
	}

	.preview-header-left h3 {
		font-size: 0.9375rem;
		margin: 0;
		color: #ffffff;
	}

	.modal-controls {
		display: flex;
		gap: 0.4rem;
	}

	.control-select {
		background: #1e293b;
		border: 1px solid #334155;
		color: #38bdf8;
		font-size: 0.75rem;
		padding: 0.25rem 0.5rem;
		border-radius: 4px;
	}

	.preview-header-actions {
		display: flex;
		align-items: center;
		gap: 0.6rem;
	}

	.print-dialog-btn {
		background: #0284c7;
		color: #ffffff;
		border: none;
		padding: 0.4rem 0.8rem;
		border-radius: 6px;
		font-weight: 700;
		font-size: 0.8125rem;
		cursor: pointer;
	}

	.whatsapp-modal-btn {
		background: #166534;
		color: #ffffff;
		border: none;
		padding: 0.4rem 0.8rem;
		border-radius: 6px;
		font-weight: 600;
		font-size: 0.8125rem;
		cursor: pointer;
	}

	.template-render-wrapper {
		flex: 1;
		overflow-y: auto;
		padding: 2rem;
		background: #1e293b;
		display: flex;
		justify-content: center;
	}

	.modal-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 1rem;
	}

	.modal-header h3 {
		margin: 0;
		font-size: 1.125rem;
		color: #ffffff;
	}

	.modal-close {
		background: transparent;
		border: none;
		color: #94a3b8;
		cursor: pointer;
		font-size: 1.125rem;
	}

	.modal-info-row {
		display: flex;
		justify-content: space-between;
		padding: 0.3rem 0;
		font-size: 0.875rem;
		color: #94a3b8;
	}

	.modal-footer {
		display: flex;
		justify-content: flex-end;
		gap: 0.75rem;
		margin-top: 1.5rem;
	}

	.modal-error {
		background: rgba(239, 68, 68, 0.15);
		color: #f87171;
		padding: 0.5rem;
		border-radius: 6px;
		font-size: 0.8125rem;
		margin-bottom: 1rem;
	}

	/* ── 🏗️ Blue Metals & Quarry Weighbridge Station UI ── */
	.quarry-toggle-btn {
		background: linear-gradient(135deg, rgba(14, 165, 233, 0.2), rgba(2, 132, 199, 0.3));
		color: #38bdf8;
		border: 1px solid rgba(56, 189, 248, 0.4);
		padding: 0.45rem 0.95rem;
		border-radius: 8px;
		font-size: 0.8125rem;
		font-weight: 700;
		cursor: pointer;
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		transition: all 0.2s cubic-bezier(0.16, 1, 0.3, 1);
		box-shadow: 0 2px 8px rgba(2, 132, 199, 0.2);
	}

	.quarry-toggle-btn:hover {
		background: linear-gradient(135deg, rgba(14, 165, 233, 0.35), rgba(2, 132, 199, 0.5));
		border-color: #38bdf8;
		transform: translateY(-1px);
		box-shadow: 0 4px 14px rgba(2, 132, 199, 0.35);
	}

	.quarry-toggle-btn.active {
		background: linear-gradient(135deg, #0284c7, #0369a1);
		color: #ffffff;
		border-color: #38bdf8;
		box-shadow: 0 0 15px rgba(56, 189, 248, 0.5);
	}

	.quarry-station-card {
		background: linear-gradient(135deg, rgba(15, 23, 42, 0.95) 0%, rgba(30, 41, 59, 0.9) 100%);
		border: 1px solid rgba(56, 189, 248, 0.35);
		border-radius: 12px;
		padding: 1.5rem;
		margin-bottom: 1.5rem;
		box-shadow: 0 10px 30px rgba(0, 0, 0, 0.45), 0 0 20px rgba(2, 132, 199, 0.15);
		position: relative;
		overflow: hidden;
	}

	.quarry-station-card::before {
		content: '';
		position: absolute;
		top: 0;
		left: 0;
		right: 0;
		height: 3px;
		background: linear-gradient(90deg, #0284c7, #38bdf8, #f59e0b, #10b981);
	}

	.quarry-header {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		gap: 1.5rem;
		flex-wrap: wrap;
		padding-bottom: 1.25rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
		margin-bottom: 1.25rem;
	}

	.quarry-title {
		flex: 1;
		min-width: 280px;
	}

	.quarry-badge-row {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		margin-bottom: 0.45rem;
	}

	.quarry-badge {
		background: rgba(2, 132, 199, 0.25);
		color: #38bdf8;
		border: 1px solid rgba(56, 189, 248, 0.5);
		padding: 0.2rem 0.6rem;
		border-radius: 999px;
		font-size: 0.7rem;
		font-weight: 800;
		letter-spacing: 0.05em;
		text-transform: uppercase;
	}

	.quarry-subbadge {
		background: rgba(245, 158, 11, 0.2);
		color: #fbbf24;
		border: 1px solid rgba(245, 158, 11, 0.4);
		padding: 0.2rem 0.55rem;
		border-radius: 999px;
		font-size: 0.7rem;
		font-weight: 700;
	}

	.quarry-title h3 {
		font-size: 1.25rem;
		font-weight: 800;
		color: #ffffff;
		margin: 0 0 0.25rem 0;
		letter-spacing: -0.02em;
	}

	.quarry-title p {
		font-size: 0.8125rem;
		color: #94a3b8;
		margin: 0;
	}

	/* Weighbridge Digital Phosphor Display */
	.weighbridge-digital-display {
		background: #020617;
		border: 2px solid #0284c7;
		border-radius: 8px;
		padding: 0.75rem 1.25rem;
		min-width: 240px;
		text-align: right;
		box-shadow: inset 0 2px 8px rgba(0, 0, 0, 0.8), 0 0 15px rgba(2, 132, 199, 0.3);
		position: relative;
	}

	.display-label {
		display: block;
		font-size: 0.6875rem;
		font-weight: 800;
		letter-spacing: 0.1em;
		color: #64748b;
		text-transform: uppercase;
	}

	.display-value {
		display: block;
		font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
		font-size: 2.25rem;
		font-weight: 900;
		color: #38bdf8;
		line-height: 1.1;
		margin: 0.15rem 0;
		text-shadow: 0 0 12px rgba(56, 189, 248, 0.6);
	}

	.display-value small {
		font-size: 1.125rem;
		color: #0284c7;
		font-weight: 700;
	}

	.display-sub {
		display: block;
		font-size: 0.6875rem;
		color: #94a3b8;
		font-family: ui-monospace, SFMono-Regular, monospace;
	}

	.quarry-form-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
		gap: 1rem;
		margin-bottom: 1.25rem;
	}

	.quarry-form-grid label {
		font-size: 0.75rem;
		font-weight: 700;
		color: #cbd5e1;
		margin-bottom: 0.35rem;
		display: flex;
		align-items: center;
		gap: 0.3rem;
	}

	.quarry-form-grid input {
		background: rgba(15, 23, 42, 0.85);
		border: 1px solid rgba(255, 255, 255, 0.12);
		color: #ffffff;
		padding: 0.55rem 0.75rem;
		border-radius: 6px;
		font-size: 0.875rem;
		font-weight: 600;
		width: 100%;
		transition: all 0.2s;
	}

	.quarry-form-grid input:focus {
		outline: none;
		border-color: #38bdf8;
		background: rgba(15, 23, 42, 1);
		box-shadow: 0 0 0 3px rgba(56, 189, 248, 0.2);
	}

	/* Materials Palette */
	.materials-palette-wrapper {
		background: rgba(15, 23, 42, 0.65);
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 8px;
		padding: 1rem;
	}

	.palette-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 0.75rem;
		flex-wrap: wrap;
		gap: 0.5rem;
	}

	.palette-title {
		font-size: 0.8125rem;
		font-weight: 700;
		color: #e2e8f0;
	}

	.palette-hint {
		font-size: 0.6875rem;
		color: #38bdf8;
		font-weight: 600;
	}

	.quarry-materials-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
		gap: 0.65rem;
	}

	.material-chip {
		background: rgba(30, 41, 59, 0.75);
		border: 1px solid rgba(255, 255, 255, 0.09);
		border-radius: 8px;
		padding: 0.65rem 0.85rem;
		display: flex;
		align-items: center;
		gap: 0.65rem;
		cursor: pointer;
		text-align: left;
		transition: all 0.2s cubic-bezier(0.16, 1, 0.3, 1);
		color: #ffffff;
		position: relative;
	}

	.material-chip:hover {
		background: rgba(30, 41, 59, 1);
		border-color: #38bdf8;
		transform: translateY(-2px);
		box-shadow: 0 4px 12px rgba(2, 132, 199, 0.25);
	}

	.mat-icon {
		font-size: 1.35rem;
		flex-shrink: 0;
	}

	.mat-info {
		flex: 1;
		min-width: 0;
	}

	.mat-name {
		display: block;
		font-size: 0.8125rem;
		font-weight: 700;
		color: #f8fafc;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.mat-meta {
		display: block;
		font-size: 0.6875rem;
		color: #94a3b8;
		margin-top: 0.1rem;
		font-family: ui-monospace, SFMono-Regular, monospace;
	}

	.mat-add-badge {
		background: rgba(56, 189, 248, 0.15);
		color: #38bdf8;
		font-size: 0.6875rem;
		font-weight: 700;
		padding: 0.2rem 0.45rem;
		border-radius: 4px;
		flex-shrink: 0;
		transition: all 0.2s;
	}

	.material-chip:hover .mat-add-badge {
		background: #0284c7;
		color: #ffffff;
	}
</style>
