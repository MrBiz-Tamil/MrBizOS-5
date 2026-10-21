<script lang="ts">
	import { onMount } from 'svelte';
	import { api, type Tenant, type Invoice } from '$lib/api';
	import InvoiceTemplates from '$lib/components/InvoiceTemplates.svelte';

	let company = $state<Tenant | null>(null);
	let isSuperAdmin = $state(false);
	let loading = $state(true);
	let saving = $state(false);
	let activeTemplate = $state('1');
	let activeCopy = $state<'ORIGINAL' | 'DUPLICATE' | 'TRIPLICATE' | 'OFFICE'>('ORIGINAL');
	let activeTaxMode = $state<'WITH_GST' | 'WITHOUT_GST'>('WITH_GST');
	let activePaperSize = $state<'A4' | 'A5' | 'A5_LANDSCAPE'>('A4');
	let defaultTemplate = $state('1');
	let savedSuccess = $state(false);
	let activeTab = $state<'catalog' | 'options'>('catalog');

	// Real Tenant Configurable Options
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
	let showBankDetails = $state(true);
	let showUPIQR = $state(true);
	let showTaxSummary = $state(true);

	const templates = [
		{
			id: '1',
			name: '1. Modern Corporate',
			category: 'Corporate B2B & Enterprise',
			desc: 'Contemporary indigo dual-tone layout with metadata pill cards, QR code, and structured GST breakdown.',
			badge: 'Popular'
		},
		{
			id: '2',
			name: '2. Classic Traditional Boxed',
			category: 'Wholesale & Distribution',
			desc: 'Bordered cell outlines adhering strictly to standard Indian GST Rule 46 format with authorized signatory seal.',
			badge: 'GST Standard'
		},
		{
			id: '3',
			name: '3. Minimalist Executive',
			category: 'Consulting & Legal',
			desc: 'High-end editorial typography, generous whitespace, and fine dividers for professional firms.',
			badge: 'Clean'
		},
		{
			id: '4',
			name: '4. Retail Counter POS',
			category: 'Retail Stores & Supermarkets',
			desc: 'Space-saving counter receipt format with MRP, savings callouts, and cashier receipt headers.',
			badge: 'Retail'
		},
		{
			id: '5',
			name: '5. Thermal 80mm Receipt',
			category: 'POS Thermal Roll Printers',
			desc: 'Continuous 3-inch/80mm roll paper layout with dashed dividers and monospace print formatting.',
			badge: '80mm POS'
		},
		{
			id: '6',
			name: '6. Creative Studio & Agency',
			category: 'Agencies & Freelancers',
			desc: 'Vibrant accent colors with scope-of-work deliverable breakdowns for digital agencies.',
			badge: 'Creative'
		},
		{
			id: '7',
			name: '7. Industrial & Dispatch B2B',
			category: 'Manufacturing & Logistics',
			desc: 'Dedicated transport fields: PO Number, Vehicle #, E-Way Bill #, Transporter Name, and Delivery Challan.',
			badge: 'Logistics'
		},
		{
			id: '8',
			name: '8. Bold Tech SaaS',
			category: 'Cloud Subscriptions & IT',
			desc: 'Modern dark gradient banner with plan tiers, billing cycles, and software subscription terms.',
			badge: 'SaaS'
		},
		{
			id: '9',
			name: '9. Wholesale Multi-Tax Slab',
			category: 'Trading & Multi-Tax Traders',
			desc: 'Includes an exhaustive HSN/SAC summary table dividing CGST, SGST, and IGST by rate slab.',
			badge: 'Tax Slab'
		},
		{
			id: '10',
			name: '10. Simple Bill of Supply',
			category: 'Composition & Non-GST Traders',
			desc: 'No tax columns (Qty × Rate = Total), equipped with statutory Section 31(3)(c) composition notice.',
			badge: 'Composition'
		},
		{
			id: '11',
			name: '11. Industrial BlueMetals & Transport',
			category: 'Mining, Heavy Materials & Transport',
			desc: 'Heavy boxed grid format with PO No, Vehicle No, Delivery Place, E.& O.E, and authentic Indian quarry/metals GST structure.',
			badge: 'Industry Favorite'
		},
		{
			id: '12',
			name: '12. Hostel / Lodging Guest Folio',
			category: 'Hostels, PGs, Guest Houses & Lodges',
			desc: 'Guest folio with room/bed, booking ref, check-in/check-out, nights stayed, GST slab summary, advance & balance, and guest signature.',
			badge: 'Hospitality'
		},
		{
			id: '13',
			name: '13. Restaurant / Cafe Bill',
			category: 'Restaurants, Cafes, Bakeries & Cloud Kitchens',
			desc: 'Compact dine-in bill with table, waiter, KOT/order no, FSSAI licence, CGST/SGST split, UPI QR and a customisable thank-you line.',
			badge: 'Food Service'
		},
		{
			id: '14',
			name: '14. Pharmacy / Medical Store',
			category: 'Pharmacies, Chemists & Medical Stores',
			desc: 'Drug licence header, patient & prescribing doctor, per-line batch / expiry / MRP with short-expiry flag, MRP savings, GST slab summary, Schedule H notice and pharmacist signature.',
			badge: 'Pharma'
		},
		{
			id: '15',
			name: '15. Bakery / Sweet Shop',
			category: 'Bakeries, Cake Shops, Sweet Marts & Cafes',
			desc: 'FSSAI licence header, veg / non-veg marks, best-before per item, cake-order block (pickup date, message on cake), per-rate GST split, advance paid & balance, UPI QR.',
			badge: 'Bakery'
		},
		{
			id: '16',
			name: '16. Grocery / Supermarket',
			category: 'Kirana Stores, Supermarkets & Fresh Produce',
			desc: 'MRP vs selling price per line, total savings badge, weighed quantities (kg / L), item & quantity count, GST summary that keeps exempt staples visible, cash tendered & change, UPI QR.',
			badge: 'Grocery'
		},
		{
			id: '17',
			name: '17. Hardware / Building Materials',
			category: 'Hardware, Paints, Plumbing, Electricals & Building Materials',
			desc: 'Trade-discount column, delivery / site block with transport details (vehicle, e-way, LR), HSN-wise tax summary, credit terms with due date, bank details, receiver and authorised signatures.',
			badge: 'Hardware'
		}
	];

	// Live Sample Mock Invoice for Live Showcase
	const sampleInvoice: Invoice = {
		id: 'sample-inv-preview',
		tenant_id: 'tenant-demo',
		invoice_number: 'INV-2026-00421',
		invoice_date: new Date().toISOString(),
		due_date: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toISOString(),
		customer_id: 'cust-demo',
		customer_name: 'Apex Global Enterprises LLP',
		customer_gstin: '27AABCA1234F1Z5',
		billing_address: 'Plot 42, Bandra Kurla Complex (BKC)\nBandra East, Mumbai, Maharashtra - 400051',
		shipping_address: 'Apex Warehouse Hub, Bhiwandi, Maharashtra - 421302',
		supply_state: 'Maharashtra',
		is_inter_state: false,
		subtotal: '25400.00',
		discount_total: '1000.00',
		taxable_amount: '24400.00',
		cgst_total: '2196.00',
		sgst_total: '2196.00',
		igst_total: '0.00',
		utgst_total: '0.00',
		round_off: '0.00',
		grand_total: '28792.00',
		amount_paid: '15000.00',
		balance_due: '13792.00',
		status: 'PARTIALLY_PAID',
		notes: 'Thank you for partnering with us. Please remit the remaining balance within 30 days via NEFT/RTGS or UPI.',
		created_at: new Date().toISOString(),
		project_name: 'Q3 Enterprise Architecture Setup',
		gross_weight: '14.80 MT',
		net_weight: '14.20 MT',
		cashier_name: 'Rajesh Sharma',
		counter_number: 'Counter B-02',
		subscription_period_start: new Date().toISOString(),
		subscription_period_end: new Date(Date.now() + 365 * 24 * 60 * 60 * 1000).toISOString(),
		items: [
			{
				id: 'item-1',
				item_description: 'Industrial Grade Sensor Assembly (Model X-200)',
				hsn_sac: '84713010',
				quantity: '10.00',
				unit_code: 'NOS',
				unit_price: '1500.00',
				discount_rate: '5.00',
				discount_amount: '750.00',
				taxable_value: '14250.00',
				tax_rate: '18.00',
				cgst_rate: '9.00',
				cgst_amount: '1282.50',
				sgst_rate: '9.00',
				sgst_amount: '1282.50',
				igst_rate: '0.00',
				igst_amount: '0.00',
				total_amount: '16815.00'
			},
			{
				id: 'item-2',
				item_description: 'Annual Maintenance & Firmware Cloud Subscription',
				hsn_sac: '998313',
				quantity: '1.00',
				unit_code: 'YEAR',
				unit_price: '10400.00',
				discount_rate: '2.40',
				discount_amount: '250.00',
				taxable_value: '10150.00',
				tax_rate: '18.00',
				cgst_rate: '9.00',
				cgst_amount: '913.50',
				sgst_rate: '9.00',
				sgst_amount: '913.50',
				igst_rate: '0.00',
				igst_amount: '0.00',
				total_amount: '11977.00'
			}
		]
	};

	const addDays = (d: number) => new Date(Date.now() + d * 86400000).toISOString();
	const baseItem = sampleInvoice.items![0];

	const sampleHostelInvoice: Invoice = {
		...sampleInvoice,
		id: 'sample-hostel-preview',
		invoice_number: 'HST-2026-00087',
		customer_name: 'Arjun Mehta',
		customer_gstin: '',
		billing_address: 'Flat 12, Lake View Apartments, Pune - 411014',
		supply_state: 'Maharashtra',
		subtotal: '7400.00',
		discount_total: '400.00',
		taxable_amount: '7000.00',
		cgst_total: '433.00',
		sgst_total: '433.00',
		round_off: '0.00',
		grand_total: '7866.00',
		amount_paid: '3000.00',
		balance_due: '4866.00',
		status: 'PARTIALLY_PAID',
		notes: 'Government ID verified at check-in. Locker key deposit collected separately.',
		project_name: 'BKG-20418',
		counter_number: 'Dorm A - Bed 4',
		cashier_name: 'Front Desk - Meera',
		subscription_period_start: addDays(-4),
		subscription_period_end: addDays(0),
		items: [
			{ ...baseItem, id: 'h1', item_description: 'Dorm bed (6-bed, AC) - 4 nights', hsn_sac: '996311', quantity: '4.00', unit_code: 'NIGHT', unit_price: '1200.00', discount_rate: '8.33', discount_amount: '400.00', taxable_value: '4400.00', tax_rate: '12.00', cgst_rate: '6.00', cgst_amount: '264.00', sgst_rate: '6.00', sgst_amount: '264.00', total_amount: '4928.00' },
			{ ...baseItem, id: 'h2', item_description: 'Breakfast plan - 4 mornings', hsn_sac: '996331', quantity: '4.00', unit_code: 'DAY', unit_price: '250.00', discount_rate: '0', discount_amount: '0.00', taxable_value: '1000.00', tax_rate: '5.00', cgst_rate: '2.50', cgst_amount: '25.00', sgst_rate: '2.50', sgst_amount: '25.00', total_amount: '1050.00' },
			{ ...baseItem, id: 'h3', item_description: 'Laundry & towel service', hsn_sac: '999711', quantity: '1.00', unit_code: 'SET', unit_price: '1600.00', discount_rate: '0', discount_amount: '0.00', taxable_value: '1600.00', tax_rate: '18.00', cgst_rate: '9.00', cgst_amount: '144.00', sgst_rate: '9.00', sgst_amount: '144.00', total_amount: '1888.00' }
		]
	};

	const sampleRestaurantInvoice: Invoice = {
		...sampleInvoice,
		id: 'sample-restaurant-preview',
		invoice_number: 'RST-2026-01342',
		customer_name: 'Cash Customer',
		customer_gstin: '',
		billing_address: '',
		subtotal: '1180.00',
		discount_total: '0.00',
		taxable_amount: '1180.00',
		cgst_total: '29.50',
		sgst_total: '29.50',
		round_off: '0.00',
		grand_total: '1239.00',
		amount_paid: '1239.00',
		balance_due: '0.00',
		status: 'PAID',
		notes: '',
		payment_mode: 'UPI',
		project_name: 'KOT-4471',
		counter_number: 'T-12',
		cashier_name: 'Ravi',
		items: [
			{ ...baseItem, id: 'r1', item_description: 'Paneer Butter Masala', hsn_sac: '996331', quantity: '2', unit_code: 'PLATE', unit_price: '260.00', discount_rate: '0', discount_amount: '0.00', taxable_value: '520.00', tax_rate: '5.00', cgst_amount: '13.00', sgst_amount: '13.00', total_amount: '546.00' },
			{ ...baseItem, id: 'r2', item_description: 'Butter Naan', hsn_sac: '996331', quantity: '6', unit_code: 'PC', unit_price: '55.00', discount_rate: '0', discount_amount: '0.00', taxable_value: '330.00', tax_rate: '5.00', cgst_amount: '8.25', sgst_amount: '8.25', total_amount: '346.50' },
			{ ...baseItem, id: 'r3', item_description: 'Fresh Lime Soda', hsn_sac: '996331', quantity: '3', unit_code: 'GLASS', unit_price: '70.00', discount_rate: '0', discount_amount: '0.00', taxable_value: '210.00', tax_rate: '5.00', cgst_amount: '5.25', sgst_amount: '5.25', total_amount: '220.50' },
			{ ...baseItem, id: 'r4', item_description: 'Service charge', hsn_sac: '996331', quantity: '1', unit_code: 'NOS', unit_price: '120.00', discount_rate: '0', discount_amount: '0.00', taxable_value: '120.00', tax_rate: '5.00', cgst_amount: '3.00', sgst_amount: '3.00', total_amount: '126.00' }
		]
	};

	const samplePharmacyInvoice: Invoice = {
		...sampleInvoice,
		id: 'sample-pharmacy-preview',
		invoice_number: 'PH-2026-03217',
		customer_name: 'Sunita Rao',
		customer_gstin: '',
		billing_address: '14, Green Park Colony, Pune - 411038',
		supply_state: 'Maharashtra',
		subtotal: '1062.00',
		discount_total: '53.10',
		taxable_amount: '962.43',
		cgst_total: '30.84',
		sgst_total: '30.84',
		igst_total: '0.00',
		round_off: '-0.11',
		grand_total: '1024.00',
		amount_paid: '1024.00',
		balance_due: '0.00',
		status: 'PAID',
		notes: '',
		payment_mode: 'Cash',
		doctor_name: 'Dr. Anil Kulkarni, MD',
		doctor_reg_no: 'MMC-2014/08817',
		patient_name: 'Sunita Rao',
		patient_age: 54,
		cashier_name: 'Pharmacist - Neha',
		counter_number: 'Counter 1',
		tendered_cash: '1100.00',
		change_amount: '76.00',
		items: [
			{ ...baseItem, id: 'p1', item_description: 'Metformin 500mg SR (Strip of 10)', hsn_sac: '3004', quantity: '6', unit_code: 'STRIP', unit_price: '22.00', discount_rate: '5.00', discount_amount: '6.60', taxable_value: '125.40', tax_rate: '5.00', cgst_amount: '3.14', sgst_amount: '3.13', total_amount: '131.67', batch_number: 'MF2407B', expiry_date: '2027-06-30', mrp: '25.50', rack_location: 'A-12' },
			{ ...baseItem, id: 'p2', item_description: 'Telmisartan 40mg (Strip of 15)', hsn_sac: '3004', quantity: '2', unit_code: 'STRIP', unit_price: '96.00', discount_rate: '5.00', discount_amount: '9.60', taxable_value: '182.40', tax_rate: '12.00', cgst_amount: '10.94', sgst_amount: '10.94', total_amount: '204.28', batch_number: 'TL9921', expiry_date: '2026-11-30', mrp: '112.00', rack_location: 'B-03' },
			{ ...baseItem, id: 'p3', item_description: 'Vitamin D3 60K Sachet (Box of 4)', hsn_sac: '3004', quantity: '3', unit_code: 'BOX', unit_price: '140.00', discount_rate: '5.00', discount_amount: '21.00', taxable_value: '399.00', tax_rate: '12.00', cgst_amount: '23.94', sgst_amount: '23.94', total_amount: '446.88', batch_number: 'VD1105', expiry_date: '2027-09-30', mrp: '165.00', rack_location: 'C-07' },
			{ ...baseItem, id: 'p4', item_description: 'Digital Thermometer', hsn_sac: '9025', quantity: '1', unit_code: 'PC', unit_price: '256.00', discount_rate: '5.00', discount_amount: '12.80', taxable_value: '243.20', tax_rate: '18.00', cgst_amount: '21.89', sgst_amount: '21.89', total_amount: '286.98', batch_number: 'TH-88', expiry_date: '', mrp: '299.00', rack_location: 'D-01' }
		]
	};

	const sampleBakeryInvoice: Invoice = {
		...sampleInvoice,
		id: 'sample-bakery-preview',
		invoice_number: 'BKR-2026-00518',
		customer_name: 'Meera Nair',
		customer_gstin: '',
		billing_address: '',
		due_date: addDays(2),
		delivery_place: 'Store pickup - Koregaon Park',
		supply_state: 'Maharashtra',
		subtotal: '2330.00',
		discount_total: '0.00',
		taxable_amount: '2330.00',
		cgst_total: '166.00',
		sgst_total: '166.00',
		igst_total: '0.00',
		round_off: '0.00',
		grand_total: '2662.00',
		amount_paid: '1000.00',
		balance_due: '1662.00',
		status: 'PARTIALLY_PAID',
		notes: 'Please keep refrigerated. Candles and knife included.',
		payment_mode: 'UPI',
		project_name: 'Happy Birthday Aarav!',
		counter_number: 'ORD-4821',
		cashier_name: 'Sneha',
		items: [
			{ ...baseItem, id: 'b1', item_description: 'Eggless Chocolate Truffle Cake (1 kg)', hsn_sac: '19059020', quantity: '1', unit_code: 'KG', unit_price: '950.00', discount_rate: '0', discount_amount: '0.00', taxable_value: '950.00', tax_rate: '18.00', cgst_rate: '9.00', cgst_amount: '85.50', sgst_rate: '9.00', sgst_amount: '85.50', total_amount: '1121.00', batch_number: 'CK-2210', expiry_date: addDays(3) },
			{ ...baseItem, id: 'b2', item_description: 'Chicken Puff', hsn_sac: '19059090', quantity: '6', unit_code: 'PC', unit_price: '45.00', discount_rate: '0', discount_amount: '0.00', taxable_value: '270.00', tax_rate: '5.00', cgst_rate: '2.50', cgst_amount: '6.75', sgst_rate: '2.50', sgst_amount: '6.75', total_amount: '283.50', batch_number: '', expiry_date: addDays(1) },
			{ ...baseItem, id: 'b3', item_description: 'Whole Wheat Bread', hsn_sac: '19051000', quantity: '2', unit_code: 'PC', unit_price: '55.00', discount_rate: '0', discount_amount: '0.00', taxable_value: '110.00', tax_rate: '0.00', cgst_rate: '0', cgst_amount: '0.00', sgst_rate: '0', sgst_amount: '0.00', total_amount: '110.00', batch_number: '', expiry_date: addDays(2) },
			{ ...baseItem, id: 'b4', item_description: 'Butter Cookies (Box of 12)', hsn_sac: '19053100', quantity: '3', unit_code: 'BOX', unit_price: '250.00', discount_rate: '0', discount_amount: '0.00', taxable_value: '750.00', tax_rate: '18.00', cgst_rate: '9.00', cgst_amount: '67.50', sgst_rate: '9.00', sgst_amount: '67.50', total_amount: '885.00', batch_number: 'CK-0987', expiry_date: addDays(30) },
			{ ...baseItem, id: 'b5', item_description: 'Rasmalai (500 g)', hsn_sac: '21069099', quantity: '0.5', unit_code: 'KG', unit_price: '500.00', discount_rate: '0', discount_amount: '0.00', taxable_value: '250.00', tax_rate: '5.00', cgst_rate: '2.50', cgst_amount: '6.25', sgst_rate: '2.50', sgst_amount: '6.25', total_amount: '262.50', batch_number: '', expiry_date: addDays(1) }
		]
	};

	const sampleGroceryInvoice: Invoice = {
		...sampleInvoice,
		id: 'sample-grocery-preview',
		invoice_number: 'GRC-2026-07731',
		customer_name: 'Walk-in Customer',
		customer_gstin: '',
		billing_address: '',
		supply_state: 'Maharashtra',
		subtotal: '1352.00',
		discount_total: '0.00',
		taxable_amount: '1352.00',
		cgst_total: '26.12',
		sgst_total: '26.12',
		igst_total: '0.00',
		round_off: '-0.24',
		grand_total: '1404.00',
		amount_paid: '1404.00',
		balance_due: '0.00',
		status: 'PAID',
		notes: '',
		payment_mode: 'Cash',
		counter_number: 'Counter 2',
		cashier_name: 'Prakash',
		project_name: 'Member 40217',
		tendered_cash: '1500.00',
		change_amount: '96.00',
		items: [
			{ ...baseItem, id: 'g1', item_description: 'Basmati Rice (loose)', hsn_sac: '10063020', quantity: '5', unit_code: 'KG', unit_price: '92.00', discount_rate: '0', discount_amount: '0.00', taxable_value: '460.00', tax_rate: '0.00', cgst_rate: '0', cgst_amount: '0.00', sgst_rate: '0', sgst_amount: '0.00', total_amount: '460.00', mrp: '110.00' },
			{ ...baseItem, id: 'g2', item_description: 'Whole Wheat Atta', hsn_sac: '11010000', quantity: '5', unit_code: 'KG', unit_price: '48.00', discount_rate: '0', discount_amount: '0.00', taxable_value: '240.00', tax_rate: '0.00', cgst_rate: '0', cgst_amount: '0.00', sgst_rate: '0', sgst_amount: '0.00', total_amount: '240.00', mrp: '55.00' },
			{ ...baseItem, id: 'g3', item_description: 'Toned Milk 1 L', hsn_sac: '04012000', quantity: '4', unit_code: 'L', unit_price: '27.00', discount_rate: '0', discount_amount: '0.00', taxable_value: '108.00', tax_rate: '0.00', cgst_rate: '0', cgst_amount: '0.00', sgst_rate: '0', sgst_amount: '0.00', total_amount: '108.00', mrp: '28.00' },
			{ ...baseItem, id: 'g4', item_description: 'Sunflower Oil 1 L', hsn_sac: '15121910', quantity: '2', unit_code: 'L', unit_price: '148.00', discount_rate: '0', discount_amount: '0.00', taxable_value: '296.00', tax_rate: '5.00', cgst_rate: '2.50', cgst_amount: '7.40', sgst_rate: '2.50', sgst_amount: '7.40', total_amount: '310.80', mrp: '165.00' },
			{ ...baseItem, id: 'g5', item_description: 'Glucose Biscuits', hsn_sac: '19053100', quantity: '10', unit_code: 'PCS', unit_price: '10.00', discount_rate: '0', discount_amount: '0.00', taxable_value: '100.00', tax_rate: '18.00', cgst_rate: '9.00', cgst_amount: '9.00', sgst_rate: '9.00', sgst_amount: '9.00', total_amount: '118.00', mrp: '10.00' },
			{ ...baseItem, id: 'g6', item_description: 'Tomato (loose)', hsn_sac: '07020000', quantity: '1.25', unit_code: 'KG', unit_price: '32.00', discount_rate: '0', discount_amount: '0.00', taxable_value: '40.00', tax_rate: '0.00', cgst_rate: '0', cgst_amount: '0.00', sgst_rate: '0', sgst_amount: '0.00', total_amount: '40.00', mrp: '0' },
			{ ...baseItem, id: 'g7', item_description: 'Bathing Soap 100 g', hsn_sac: '34011190', quantity: '3', unit_code: 'PCS', unit_price: '36.00', discount_rate: '0', discount_amount: '0.00', taxable_value: '108.00', tax_rate: '18.00', cgst_rate: '9.00', cgst_amount: '9.72', sgst_rate: '9.00', sgst_amount: '9.72', total_amount: '127.44', mrp: '40.00' }
		]
	};

	const sampleHardwareInvoice: Invoice = {
		...sampleInvoice,
		id: 'sample-hardware-preview',
		invoice_number: 'HW-2026-01184',
		customer_name: 'Sai Constructions',
		customer_gstin: '27AABCS5566K1Z2',
		billing_address: 'Office 3, Shivaji Nagar, Pune - 411005',
		shipping_address: 'Site: Plot 18, Sai Residency, Wakad, Pune - 411057',
		supply_state: 'Maharashtra',
		is_inter_state: false,
		subtotal: '56300.00',
		discount_total: '1278.00',
		taxable_amount: '55022.00',
		cgst_total: '4951.98',
		sgst_total: '4951.98',
		igst_total: '0.00',
		round_off: '0.04',
		grand_total: '64926.00',
		amount_paid: '20000.00',
		balance_due: '44926.00',
		status: 'PARTIALLY_PAID',
		notes: 'Delivery by truck to site. Cement stacked on pallets.',
		payment_mode: 'Credit - 15 days',
		due_date: new Date(Date.now() + 15 * 86400000).toISOString(),
		po_number: 'WO-2291',
		project_name: 'Sai Residency - Block B',
		vehicle_number: 'MH12AB3456',
		transporter_name: 'Shree Roadlines',
		lr_number: 'LR-8841',
		eway_bill: '271000987654',
		gross_weight: '2.85 MT',
		net_weight: '2.80 MT',
		items: [
			{ ...baseItem, id: 'h1', item_description: 'PPC Cement 50 kg bag (ISI)', hsn_sac: '2523', quantity: '20', unit_code: 'BAG', unit_price: '385.00', discount_rate: '2.00', discount_amount: '154.00', taxable_value: '7546.00', tax_rate: '18.00', cgst_rate: '9.00', cgst_amount: '679.14', sgst_rate: '9.00', sgst_amount: '679.14', total_amount: '8904.28' },
			{ ...baseItem, id: 'h2', item_description: 'TMT Steel Bar 12 mm Fe500D', hsn_sac: '7214', quantity: '500', unit_code: 'KG', unit_price: '64.00', discount_rate: '0', discount_amount: '0.00', taxable_value: '32000.00', tax_rate: '18.00', cgst_rate: '9.00', cgst_amount: '2880.00', sgst_rate: '9.00', sgst_amount: '2880.00', total_amount: '37760.00' },
			{ ...baseItem, id: 'h3', item_description: 'PVC Pipe 1 inch ISI', hsn_sac: '3917', quantity: '40', unit_code: 'MTR', unit_price: '58.00', discount_rate: '5.00', discount_amount: '116.00', taxable_value: '2204.00', tax_rate: '18.00', cgst_rate: '9.00', cgst_amount: '198.36', sgst_rate: '9.00', sgst_amount: '198.36', total_amount: '2600.72' },
			{ ...baseItem, id: 'h4', item_description: 'Tractor Emulsion Paint 20 L', hsn_sac: '3209', quantity: '3', unit_code: 'TIN', unit_price: '4200.00', discount_rate: '8.00', discount_amount: '1008.00', taxable_value: '11592.00', tax_rate: '18.00', cgst_rate: '9.00', cgst_amount: '1043.28', sgst_rate: '9.00', sgst_amount: '1043.28', total_amount: '13678.56' },
			{ ...baseItem, id: 'h5', item_description: 'Claw Hammer 500 g', hsn_sac: '8205', quantity: '6', unit_code: 'NOS', unit_price: '280.00', discount_rate: '0', discount_amount: '0.00', taxable_value: '1680.00', tax_rate: '18.00', cgst_rate: '9.00', cgst_amount: '151.20', sgst_rate: '9.00', sgst_amount: '151.20', total_amount: '1982.40' }
		]
	};

	let previewInvoice = $derived(
		activeTemplate === '12'
			? sampleHostelInvoice
			: activeTemplate === '13'
				? sampleRestaurantInvoice
				: activeTemplate === '14'
					? samplePharmacyInvoice
					: activeTemplate === '15'
						? sampleBakeryInvoice
						: activeTemplate === '16'
							? sampleGroceryInvoice
							: activeTemplate === '17'
								? sampleHardwareInvoice
								: sampleInvoice
	);

	onMount(async () => {
		try {
			const [meData, compData] = await Promise.all([
				api.getMe().catch(() => null),
				api.getCompany().catch(() => null)
			]);

			if (meData?.user) {
				isSuperAdmin = !!meData.user.is_super_admin;
			}

			if (compData) {
				company = compData;
				if (compData.template_id) {
					defaultTemplate = compData.template_id;
					activeTemplate = compData.template_id;
				}
				bankName = compData.bank_name || '';
				bankAccountNo = compData.bank_account_no || '';
				bankIFSC = compData.bank_ifsc || '';
				bankBranch = compData.bank_branch || '';
				bankAccountHolder = compData.bank_account_holder || '';
				upiId = compData.upi_id || '';
				businessSubtitle = compData.business_subtitle || '';
				signatoryTitle = compData.signatory_title || 'Authorized Signatory';
				signatureUrl = compData.signature_url || '';
				defaultTerms = compData.default_terms || 'Cash';
				defaultJurisdiction = compData.default_jurisdiction || compData.city || 'Pune';
				defaultInterestRate = compData.default_interest_rate || '24%';
				defaultReturnPolicy = compData.default_return_policy || 'Goods once sold will not be taken back.';
				showBankDetails = compData.template_options?.show_bank_details ?? true;
				showUPIQR = compData.template_options?.show_upi_qr ?? true;
				showTaxSummary = compData.template_options?.show_tax_summary ?? true;
			}
		} catch (e) {
			// Demo company fallback
			company = {
				id: 'demo-co',
				name: 'Vertex Technologies Private Limited',
				legal_name: 'Vertex Technologies Private Limited',
				gstin: '27AABCV9999P1Z8',
				address: 'Tower B, Tech Park, MIDC Industrial Area',
				city: 'Pune',
				state: 'Maharashtra',
				country: 'India',
				pincode: '411057',
				email: 'billing@vertextech.in',
				phone: '+91 98765 43210',
				invoice_prefix: 'VTX',
				financial_year_start: '04-01',
				currency: 'INR',
				subscription_plan: 'ENTERPRISE',
				subscription_status: 'ACTIVE',
				template_id: '1',
				is_active: true,
				created_at: new Date().toISOString()
			};
		} finally {
			loading = false;
		}
	});

	// Derived Live Preview Company Object reflecting user modifications in real time
	let previewCompany = $derived<Tenant | null>(company ? {
		...company,
		template_id: activeTemplate,
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
		template_options: {
			...(company.template_options || {}),
			show_bank_details: showBankDetails,
			show_upi_qr: showUPIQR,
			show_tax_summary: showTaxSummary
		}
	} : null);

	async function saveTemplateAndOptions() {
		if (!company) return;
		saving = true;
		try {
			const payload = {
				template_id: activeTemplate,
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
				template_options: {
					...(company.template_options || {}),
					show_bank_details: showBankDetails,
					show_upi_qr: showUPIQR,
					show_tax_summary: showTaxSummary
				}
			};

			const updated = await api.updateCompany(payload);
			company = updated;
			defaultTemplate = activeTemplate;
			savedSuccess = true;
			setTimeout(() => {
				savedSuccess = false;
			}, 3500);
		} catch (err: any) {
			alert('Failed to save template options: ' + (err.message || err));
		} finally {
			saving = false;
		}
	}
</script>

<svelte:head>
	<title>Specialized Billing Templates - SaaS Billing</title>
	{#if activePaperSize === 'A5'}
		<style>
			@media print {
				@page {
					size: 148mm 210mm !important;
					margin: 4mm !important;
				}
			}
		</style>
	{:else if activePaperSize === 'A5_LANDSCAPE'}
		<style>
			@media print {
				@page {
					size: 210mm 148mm !important;
					margin: 3.5mm !important;
				}
			}
		</style>
	{:else}
		<style>
			@media print {
				@page {
					size: 210mm 297mm !important;
					margin: 8mm !important;
				}
			}
		</style>
	{/if}
</svelte:head>

<div style="display: flex; flex-direction: column; gap: 1.5rem;">
	<!-- Page Header -->
	<div style="display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 1rem;">
		<div>
			<a href="/settings" style="font-size: 0.8125rem; color: #818cf8; text-decoration: none; margin-bottom: 0.25rem; display: inline-block;">
				← Back to Settings
			</a>
			<h1 style="font-size: 1.75rem; margin-bottom: 0.25rem;">Specialized Templates & Tenant Options</h1>
			<p style="font-size: 0.875rem; color: var(--text-secondary);">
				Select your default billing format, configure banking coordinates, UPI QR codes, statutory terms, and digital signatures.
			</p>
		</div>

		<div style="display: flex; gap: 0.75rem; align-items: center;">
			{#if savedSuccess}
				<span style="font-size: 0.8125rem; color: #34d399; font-weight: 700; background: rgba(16, 185, 129, 0.15); padding: 0.35rem 0.75rem; border-radius: 6px; border: 1px solid rgba(16, 185, 129, 0.3);">
					✓ Saved Template & Options!
				</span>
			{/if}
			<button id="btn-save-template-options" onclick={saveTemplateAndOptions} disabled={saving} class="btn btn-primary" style="display: inline-flex; align-items: center; gap: 0.5rem;">
				<span>★</span>
				<span>{saving ? 'Saving...' : `Set Template #${activeTemplate} & Options as Default`}</span>
			</button>
			<button id="btn-test-print" onclick={() => window.print()} class="btn btn-secondary">
				🖨️ Test Print ({activePaperSize.replace('_', ' ')})
			</button>
		</div>
	</div>

	<!-- Interactive Controls Toolbar -->
	<div class="card template-controls" style="padding: 1rem 1.25rem; background: rgba(17, 24, 39, 0.9); border-color: rgba(99, 102, 241, 0.35);">
		<div style="display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 1.25rem;">
			<!-- View Mode Tabs (Catalog vs Options Editor) -->
			<div style="display: flex; align-items: center; gap: 0.5rem;">
				<button
					id="tab-catalog"
					type="button"
					onclick={() => (activeTab = 'catalog')}
					class="btn btn-sm"
					style="font-size: 0.8125rem; padding: 0.4rem 0.85rem; background: {activeTab === 'catalog' ? 'var(--primary)' : 'rgba(255,255,255,0.06)'}; color: #fff; font-weight: 600;"
				>
					📋 11 Template Archetypes
				</button>
				<button
					id="tab-options"
					type="button"
					onclick={() => (activeTab = 'options')}
					class="btn btn-sm"
					style="font-size: 0.8125rem; padding: 0.4rem 0.85rem; background: {activeTab === 'options' ? 'var(--primary)' : 'rgba(255,255,255,0.06)'}; color: #fff; font-weight: 600;"
				>
					⚙️ Bank, UPI, Terms & Signatory Options
				</button>
			</div>

			<!-- Paper Size Selector -->
			<div style="display: flex; align-items: center; gap: 0.5rem;">
				<span style="font-size: 0.75rem; font-weight: 700; color: #a5b4fc; text-transform: uppercase;">Paper:</span>
				<div style="display: inline-flex; border: 1px solid var(--border-subtle); border-radius: var(--radius-md); overflow: hidden;">
					<button
						type="button"
						onclick={() => (activePaperSize = 'A4')}
						class="btn btn-sm"
						style="border-radius: 0; font-size: 0.75rem; padding: 0.3rem 0.65rem; background: {activePaperSize === 'A4' ? 'rgba(99, 102, 241, 0.3)' : 'transparent'}; color: {activePaperSize === 'A4' ? '#a5b4fc' : 'var(--text-secondary)'}; font-weight: {activePaperSize === 'A4' ? '700' : '500'};"
					>
						A4
					</button>
					<button
						type="button"
						onclick={() => (activePaperSize = 'A5')}
						class="btn btn-sm"
						style="border-radius: 0; font-size: 0.75rem; padding: 0.3rem 0.65rem; background: {activePaperSize === 'A5' ? 'rgba(16, 185, 129, 0.25)' : 'transparent'}; color: {activePaperSize === 'A5' ? '#34d399' : 'var(--text-secondary)'}; font-weight: {activePaperSize === 'A5' ? '700' : '500'};"
					>
						A5 Port
					</button>
					<button
						type="button"
						onclick={() => (activePaperSize = 'A5_LANDSCAPE')}
						class="btn btn-sm"
						style="border-radius: 0; font-size: 0.75rem; padding: 0.3rem 0.65rem; background: {activePaperSize === 'A5_LANDSCAPE' ? 'rgba(56, 189, 248, 0.25)' : 'transparent'}; color: {activePaperSize === 'A5_LANDSCAPE' ? '#38bdf8' : 'var(--text-secondary)'}; font-weight: {activePaperSize === 'A5_LANDSCAPE' ? '700' : '500'};"
					>
						A5 Land
					</button>
				</div>
			</div>

			<!-- Copy Type Selector -->
			<div style="display: flex; align-items: center; gap: 0.5rem;">
				<span style="font-size: 0.75rem; font-weight: 700; color: #a5b4fc; text-transform: uppercase;">Copy:</span>
				<div style="display: inline-flex; border: 1px solid var(--border-subtle); border-radius: var(--radius-md); overflow: hidden;">
					{#each [
						{ id: 'ORIGINAL', label: 'Original' },
						{ id: 'DUPLICATE', label: 'Duplicate' },
						{ id: 'TRIPLICATE', label: 'Triplicate' },
						{ id: 'OFFICE', label: 'Office' }
					] as cp}
						<button
							type="button"
							onclick={() => (activeCopy = cp.id as any)}
							class="btn btn-sm"
							style="border-radius: 0; font-size: 0.75rem; padding: 0.3rem 0.55rem; background: {activeCopy === cp.id ? 'rgba(99, 102, 241, 0.3)' : 'transparent'}; color: {activeCopy === cp.id ? '#a5b4fc' : 'var(--text-secondary)'}; font-weight: {activeCopy === cp.id ? '700' : '500'};"
						>
							{cp.label}
						</button>
					{/each}
				</div>
			</div>

			<!-- Tax Mode Selector -->
			<div style="display: flex; align-items: center; gap: 0.5rem;">
				<span style="font-size: 0.75rem; font-weight: 700; color: #a5b4fc; text-transform: uppercase;">GST Mode:</span>
				<div style="display: inline-flex; border: 1px solid var(--border-subtle); border-radius: var(--radius-md); overflow: hidden;">
					<button
						type="button"
						onclick={() => (activeTaxMode = 'WITH_GST')}
						class="btn btn-sm"
						style="border-radius: 0; font-size: 0.75rem; padding: 0.3rem 0.65rem; background: {activeTaxMode === 'WITH_GST' ? 'rgba(16, 185, 129, 0.25)' : 'transparent'}; color: {activeTaxMode === 'WITH_GST' ? '#34d399' : 'var(--text-secondary)'}; font-weight: {activeTaxMode === 'WITH_GST' ? '700' : '500'};"
					>
						Tax Invoice
					</button>
					<button
						type="button"
						onclick={() => (activeTaxMode = 'WITHOUT_GST')}
						class="btn btn-sm"
						style="border-radius: 0; font-size: 0.75rem; padding: 0.3rem 0.65rem; background: {activeTaxMode === 'WITHOUT_GST' ? 'rgba(245, 158, 11, 0.25)' : 'transparent'}; color: {activeTaxMode === 'WITHOUT_GST' ? '#fbbf24' : 'var(--text-secondary)'}; font-weight: {activeTaxMode === 'WITHOUT_GST' ? '700' : '500'};"
					>
						Bill of Supply
					</button>
				</div>
			</div>
		</div>
	</div>

	<!-- Main Showcase: Side Panel (Catalog or Options) + Live Canvas -->
	<div style="display: grid; grid-template-columns: 360px 1fr; gap: 1.5rem; align-items: start;">
		
		<!-- Left Panel: Template Cards Directory OR Template Options Editor -->
		{#if activeTab === 'catalog'}
			<div style="display: flex; flex-direction: column; gap: 0.75rem; max-height: 850px; overflow-y: auto; padding-right: 4px;">
				{#each templates as tpl}
					<button
						type="button"
						onclick={() => (activeTemplate = tpl.id)}
						class="card"
						style="text-align: left; padding: 1rem; cursor: pointer; border-color: {activeTemplate === tpl.id ? 'var(--primary)' : 'var(--border-subtle)'}; background: {activeTemplate === tpl.id ? 'rgba(99, 102, 241, 0.15)' : 'var(--bg-card)'}; transition: all 0.15s ease;"
					>
						<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.25rem;">
							<span style="font-weight: 700; font-size: 0.9375rem; color: {activeTemplate === tpl.id ? '#a5b4fc' : 'var(--text-primary)'};">
								{tpl.name}
							</span>
							<div style="display: flex; gap: 0.35rem;">
								{#if defaultTemplate === tpl.id}
									<span class="badge" style="background: rgba(16, 185, 129, 0.2); color: #34d399; font-size: 0.65rem;">
										Default
									</span>
								{/if}
								<span class="badge" style="background: rgba(255,255,255,0.06); font-size: 0.65rem; color: var(--text-muted);">
									{tpl.badge}
								</span>
							</div>
						</div>
						<div style="font-size: 0.75rem; color: #818cf8; font-weight: 600; margin-bottom: 0.35rem;">
							{tpl.category}
						</div>
						<p style="font-size: 0.75rem; color: var(--text-secondary); line-height: 1.35; margin: 0;">
							{tpl.desc}
						</p>
					</button>
				{/each}
			</div>
		{:else}
			<!-- Template & Company Options Panel -->
			<div class="card" style="display: flex; flex-direction: column; gap: 1.25rem; max-height: 850px; overflow-y: auto; padding: 1.25rem;">
				<div>
					<h3 style="font-size: 1rem; margin-bottom: 0.25rem; color: #a5b4fc;">💳 Bank & UPI Settlement</h3>
					<p style="font-size: 0.75rem; color: var(--text-muted);">
						Coordinates displayed in the payment box and dynamic UPI QR code on invoices.
					</p>
				</div>

				<div class="form-group" style="margin-bottom: 0;">
					<label class="form-label" for="opt-bank-name" style="font-size: 0.75rem;">Bank Name</label>
					<input id="opt-bank-name" type="text" class="form-input" bind:value={bankName} placeholder="e.g. HDFC Bank Ltd" />
				</div>

				<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 0.5rem;">
					<div class="form-group" style="margin-bottom: 0;">
						<label class="form-label" for="opt-bank-acc" style="font-size: 0.75rem;">A/C Number</label>
						<input id="opt-bank-acc" type="text" class="form-input" bind:value={bankAccountNo} placeholder="50200012345678" />
					</div>
					<div class="form-group" style="margin-bottom: 0;">
						<label class="form-label" for="opt-bank-ifsc" style="font-size: 0.75rem;">IFSC Code</label>
						<input id="opt-bank-ifsc" type="text" class="form-input font-mono" bind:value={bankIFSC} placeholder="HDFC0001234" />
					</div>
				</div>

				<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 0.5rem;">
					<div class="form-group" style="margin-bottom: 0;">
						<label class="form-label" for="opt-bank-branch" style="font-size: 0.75rem;">Branch Name</label>
						<input id="opt-bank-branch" type="text" class="form-input" bind:value={bankBranch} placeholder="Shivajinagar Branch" />
					</div>
					<div class="form-group" style="margin-bottom: 0;">
						<label class="form-label" for="opt-bank-holder" style="font-size: 0.75rem;">A/C Holder Name</label>
						<input id="opt-bank-holder" type="text" class="form-input" bind:value={bankAccountHolder} placeholder="Vertex Technologies Pvt Ltd" />
					</div>
				</div>

				<div class="form-group" style="margin-bottom: 0;">
					<label class="form-label" for="opt-upi-id" style="font-size: 0.75rem;">UPI ID / VPA (Instant Scan QR)</label>
					<input id="opt-upi-id" type="text" class="form-input font-mono" bind:value={upiId} placeholder="vertextech@okaxis" />
				</div>

				<hr style="border: 0; border-top: 1px solid var(--border-subtle); margin: 0.25rem 0;" />

				<div>
					<h3 style="font-size: 1rem; margin-bottom: 0.25rem; color: #a5b4fc;">✒️ Identity & Signatory</h3>
					<p style="font-size: 0.75rem; color: var(--text-muted);">
						Subtitle branding and digital signature authorization details.
					</p>
				</div>

				<div class="form-group" style="margin-bottom: 0;">
					<label class="form-label" for="opt-subtitle" style="font-size: 0.75rem;">Business Subtitle / Division Tagline</label>
					<input id="opt-subtitle" type="text" class="form-input" bind:value={businessSubtitle} placeholder="Enterprise Cloud & Automation Solutions" />
				</div>

				<div class="form-group" style="margin-bottom: 0;">
					<label class="form-label" for="opt-signatory-title" style="font-size: 0.75rem;">Signatory Designation Title</label>
					<input id="opt-signatory-title" type="text" class="form-input" bind:value={signatoryTitle} placeholder="Authorized Signatory / Partner" />
				</div>

				<div class="form-group" style="margin-bottom: 0;">
					<label class="form-label" for="opt-signature-url" style="font-size: 0.75rem;">Digital Signature Image URL</label>
					<input id="opt-signature-url" type="text" class="form-input" bind:value={signatureUrl} placeholder="https://example.com/signature.png" />
				</div>

				<hr style="border: 0; border-top: 1px solid var(--border-subtle); margin: 0.25rem 0;" />

				<div>
					<h3 style="font-size: 1rem; margin-bottom: 0.25rem; color: #a5b4fc;">📜 Statutory Terms & Policies</h3>
					<p style="font-size: 0.75rem; color: var(--text-muted);">
						Default terms applied to new invoices and template disclaimers.
					</p>
				</div>

				<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 0.5rem;">
					<div class="form-group" style="margin-bottom: 0;">
						<label class="form-label" for="opt-terms" style="font-size: 0.75rem;">Default Payment Terms</label>
						<input id="opt-terms" type="text" class="form-input" bind:value={defaultTerms} placeholder="Cash / Net 30" />
					</div>
					<div class="form-group" style="margin-bottom: 0;">
						<label class="form-label" for="opt-jurisdiction" style="font-size: 0.75rem;">Legal Jurisdiction City</label>
						<input id="opt-jurisdiction" type="text" class="form-input" bind:value={defaultJurisdiction} placeholder="Pune" />
					</div>
				</div>

				<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 0.5rem;">
					<div class="form-group" style="margin-bottom: 0;">
						<label class="form-label" for="opt-interest" style="font-size: 0.75rem;">Overdue Interest Rate</label>
						<input id="opt-interest" type="text" class="form-input" bind:value={defaultInterestRate} placeholder="24% p.a." />
					</div>
					<div class="form-group" style="margin-bottom: 0;">
						<label class="form-label" for="opt-return-policy" style="font-size: 0.75rem;">Return Policy</label>
						<input id="opt-return-policy" type="text" class="form-input" bind:value={defaultReturnPolicy} placeholder="Goods once sold..." />
					</div>
				</div>

				<hr style="border: 0; border-top: 1px solid var(--border-subtle); margin: 0.25rem 0;" />

				<div>
					<h3 style="font-size: 1rem; margin-bottom: 0.5rem; color: #a5b4fc;">👁️ Visual Display Toggles</h3>
					<div style="display: flex; flex-direction: column; gap: 0.5rem;">
						<label style="display: flex; align-items: center; gap: 0.5rem; font-size: 0.8125rem; cursor: pointer;">
							<input type="checkbox" bind:checked={showBankDetails} />
							<span>Display Bank Remittance Coordinates Box</span>
						</label>
						<label style="display: flex; align-items: center; gap: 0.5rem; font-size: 0.8125rem; cursor: pointer;">
							<input type="checkbox" bind:checked={showUPIQR} />
							<span>Display Scan-to-Pay UPI QR Code</span>
						</label>
						<label style="display: flex; align-items: center; gap: 0.5rem; font-size: 0.8125rem; cursor: pointer;">
							<input type="checkbox" bind:checked={showTaxSummary} />
							<span>Display HSN/SAC Tax Slab Summary Table (Template 9)</span>
						</label>
					</div>
				</div>
			</div>
		{/if}

		<!-- Right: Live Canvas Preview -->
		<div style="background: rgba(0, 0, 0, 0.2); border: 1px solid var(--border-subtle); border-radius: var(--radius-lg); padding: 1.5rem; display: flex; flex-direction: column; align-items: center; overflow-x: auto;">
			<div style="width: 100%; max-width: 900px; margin-bottom: 0.75rem; display: flex; justify-content: space-between; align-items: center;">
				<span style="font-size: 0.8125rem; font-weight: 600; color: var(--text-muted);">
					Previewing Template #{activeTemplate} • {activeCopy} • {activeTaxMode === 'WITH_GST' ? 'GST Mode' : 'Non-GST Mode'} • {activePaperSize}
				</span>
				<span style="font-size: 0.75rem; color: var(--text-muted);">
					Format: {activePaperSize === 'A5' ? 'A5 Portrait (148x210mm)' : activePaperSize === 'A5_LANDSCAPE' ? 'A5 Landscape (210x148mm)' : 'A4 Standard (210x297mm)'}
				</span>
			</div>

			<InvoiceTemplates
				invoice={previewInvoice}
				company={previewCompany}
				templateId={activeTemplate}
				copyType={activeCopy}
				taxMode={activeTaxMode}
				paperSize={activePaperSize}
			/>
		</div>
	</div>
</div>
