import { toast } from './toast.svelte';

export interface ModuleLicense {
	moduleId: string;
	status: 'ACTIVE' | 'TRIAL' | 'INACTIVE';
	plan: 'MONTHLY' | 'ANNUAL' | 'CORE';
	activatedAt?: string;
	expiresAt?: string;
	paidAmount?: number;
}

export interface ModuleDescriptor {
	id: string;
	name: string;
	icon: string;
	category: 'VERTICAL' | 'CORE' | 'AI_GROWTH' | 'OPERATIONS';
	tagline: string;
	description: string;
	priceMonthly: number;
	priceAnnual: number;
	accentVar: string;
	routes: string[];
	features: string[];
	isCore?: boolean;
}

export const MODULE_CATALOG: ModuleDescriptor[] = [
	{
		id: 'billing',
		name: 'Express Billing & Tax Invoices',
		icon: '⚡',
		category: 'CORE',
		tagline: '3-second quick billing, UPI QR generation, and 17 GST print templates',
		description: 'Core commercial invoicing, multi-currency support, payment recording, customer khata balances, and automatic tax summaries.',
		priceMonthly: 0,
		priceAnnual: 0,
		accentVar: 'var(--accent-brand)',
		routes: ['/billing', '/invoices', '/quotations', '/payments', '/customers', '/suppliers', '/products', '/inventory', '/units', '/credit-notes', '/purchases'],
		features: [
			'GST & Non-GST compliant tax invoices',
			'UPI Dynamic QR codes for instant settlement',
			'17 custom printable invoice templates',
			'Customer debit/credit ledger & WhatsApp sharing'
		],
		isCore: true
	},
	{
		id: 'pos',
		name: 'Retail POS Terminal',
		icon: '🛒',
		category: 'CORE',
		tagline: 'High-speed barcode scanner checkout and daily cash drawer tally',
		description: 'Point-of-Sale checkout terminal built for retail counters, grocery stores, pharmacies, and hardware merchants.',
		priceMonthly: 699,
		priceAnnual: 6990,
		accentVar: 'var(--accent-crm)',
		routes: ['/pos'],
		features: [
			'Barcode scanner & thermal receipt printing',
			'Cash drawer float & end-of-day reconciliation',
			'Multi-counter fast switching & hold receipts',
			'Offline emergency billing queue'
		]
	},
	{
		id: 'healthcare',
		name: 'Smart Healthcare & Clinical Hub',
		icon: '🏥',
		category: 'VERTICAL',
		tagline: 'Doctor Rx pad, pharmacy dispenser, nurse triage & clinical HL7 lab bridge',
		description: 'Complete digital medical workstation for outpatient clinics, nursing homes, and diagnostic chains with Indian drug compliance.',
		priceMonthly: 1999,
		priceAnnual: 19990,
		accentVar: 'var(--accent-healthcare)',
		routes: ['/healthcare'],
		features: [
			'Digital Doctor Prescription (Rx) pad with drug database',
			'Pharmacy strip dispensing & schedule H1 compliance',
			'Nurse OP triage vitals synchronization',
			'Clinic director revenue, doctor payouts & bed status'
		]
	},
	{
		id: 'travel',
		name: 'Travel Agency OS & Visa Desk',
		icon: '✈️',
		category: 'VERTICAL',
		tagline: 'Live tour deck, airline PNR hold clocks, DMC vouchers & VFS tracking',
		description: 'Specialized enterprise operating desk for tour operators, DMC ground handlers, and international holiday planners.',
		priceMonthly: 1999,
		priceAnnual: 19990,
		accentVar: 'var(--accent-travel)',
		routes: ['/travel'],
		features: [
			'GDS Airline PNR hold timer & auto-release countdown',
			'Hotel & DMC transfer voucher generator with WhatsApp share',
			'Schengen/UK/US visa biometrics appointment dossier',
			'Markup margin calculator & traveler passport vault'
		]
	},
	{
		id: 'realestate',
		name: 'Real Estate Deals & Plot Matrix',
		icon: '🏢',
		category: 'VERTICAL',
		tagline: '24-plot visual layout grid, Indian land unit converter & cab site visits',
		description: 'Field-closer operating system for real estate promoters, township layout planners, and commercial brokers.',
		priceMonthly: 1999,
		priceAnnual: 19990,
		accentVar: 'var(--accent-realestate)',
		routes: ['/realestate'],
		features: [
			'Visual color-coded plot unit matrix (Available / Booked / Registered)',
			'Indian land unit converter (Sq.Ft, Guntha, Ground, Cent, Acre)',
			'Site visit cab dispatch & token advance receipt',
			'RERA compliant document tracking & commission splits'
		]
	},
	{
		id: 'ecommerce',
		name: 'E-Commerce & Print-on-Demand',
		icon: '🛍️',
		category: 'VERTICAL',
		tagline: 'Blank apparel matrix (Color/Size), print lab queue & courier NDR tracking',
		description: 'Order fulfillment and printing operations engine for custom apparel, merchandise stores, and drop-shippers.',
		priceMonthly: 1499,
		priceAnnual: 14990,
		accentVar: 'var(--accent-ecommerce)',
		routes: ['/ecommerce'],
		features: [
			'Matrix blank inventory (Color × Size SKU manager)',
			'Print production queue (DTF / Screen / Embroidery)',
			'Courier webhook sync (Delhivery, Shiprocket, BlueDart)',
			'Automated COD cash recovery & RTO loss reducer'
		]
	},
	{
		id: 'projects',
		name: 'Projects & UTMS Task Delivery',
		icon: '📁',
		category: 'OPERATIONS',
		tagline: 'Universal Task Management (UTMS) with SLA clocks & cross-BU delegation',
		description: 'Agile project delivery, sprint boards, milestone billing triggers, and cross-department delegation for professional teams.',
		priceMonthly: 1299,
		priceAnnual: 12990,
		accentVar: 'var(--accent-project)',
		routes: ['/projects'],
		features: [
			'5-stage Kanban sprint pipeline with SLA breach alarms',
			'Cross-business unit task handover & delegation audit',
			'Contract milestone deliverables & 1-click GST auto-invoicing',
			'Audited developer & consultant timesheet ledger'
		]
	},
	{
		id: 'accounting',
		name: 'Financial Accounting & General Ledger',
		icon: '📖',
		category: 'CORE',
		tagline: 'Audited double-entry general ledger, bank auto-sync & trial balance',
		description: 'Chartered accountant verified double-entry bookkeeping engine with OFX/CSV statement parsers and financial statements.',
		priceMonthly: 1499,
		priceAnnual: 14990,
		accentVar: 'var(--accent-finance)',
		routes: ['/accounting'],
		features: [
			'Full Chart of Accounts & immutable journal entries',
			'General Ledger, Trial Balance, Balance Sheet & P&L',
			'Automated bank feed reconciliation with tolerance rules',
			'GST-3B and GSTR-1 preparation ledger'
		]
	},
	{
		id: 'inbox',
		name: 'Omnichannel Social Inbox',
		icon: '💬',
		category: 'AI_GROWTH',
		tagline: 'Unified WhatsApp Business, Instagram DM, Facebook & Email tickets',
		description: 'Centralized customer conversation stream connecting all enterprise messaging channels in one collaborative workspace.',
		priceMonthly: 999,
		priceAnnual: 9990,
		accentVar: 'var(--accent-crm)',
		routes: ['/inbox'],
		features: [
			'Official Meta Cloud WhatsApp Business API integration',
			'Unified customer chat thread with shared team assignment',
			'Pre-approved HSM template dispatcher with payment links',
			'Conversation tagging & sentiment monitoring'
		]
	},
	{
		id: 'bpo-voice',
		name: 'AI Voice BPO Call Center',
		icon: '🎙️',
		category: 'AI_GROWTH',
		tagline: 'Autonomous AI phone agent with live audio warm transfer to staff',
		description: 'Telephony AI voice desk for automated customer booking, appointment confirmation, order status, and human handover.',
		priceMonthly: 2499,
		priceAnnual: 24990,
		accentVar: 'var(--accent-ai)',
		routes: ['/bpo-voice'],
		features: [
			'Ultra-low-latency voice AI with conversational interruptions',
			'Warm telephony transfer to human supervisor via WebRTC',
			'Live audio transcription & automated CRM note entry',
			'Inbound IVR routing & outbound campaign scheduler'
		]
	},
	{
		id: 'procurement',
		name: '3-Way Match Procurement OS',
		icon: '📦',
		category: 'OPERATIONS',
		tagline: 'Purchase Order vs GRN receipt vs Vendor Bill automated reconciliation',
		description: 'Fraud prevention and supply chain control reconciling purchase orders, warehouse delivery notes, and supplier invoices.',
		priceMonthly: 1199,
		priceAnnual: 11990,
		accentVar: 'var(--brand-slate)',
		routes: ['/procurement'],
		features: [
			'Automated 3-way line item price & quantity tolerance check',
			'Goods Receipt Note (GRN) warehouse receiving workflow',
			'Vendor discrepancy dispute resolution tracker',
			'Approved payment release certificates'
		]
	},
	{
		id: 'hrms',
		name: 'HRMS & Automated Payroll',
		icon: '👥',
		category: 'OPERATIONS',
		tagline: 'Employee biometric attendance, salary slips, and PF/ESI compliance',
		description: 'Human resource management with biometric machine sync, leave approvals, salary structure builder, and 1-click payslips.',
		priceMonthly: 1299,
		priceAnnual: 12990,
		accentVar: 'var(--accent-workforce)',
		routes: ['/hrms'],
		features: [
			'1-click monthly payroll processing with bank NEFT export',
			'PF, ESI, Professional Tax & TDS statutory deductions',
			'Biometric punch-in and Geo-fenced mobile attendance',
			'Employee self-service portal & downloadable PDF pay slips'
		]
	},
	{
		id: 'trade',
		name: 'Export & Import Trade OS',
		icon: '🚢',
		category: 'OPERATIONS',
		tagline: 'Customs documentation, container tracking & foreign exchange ledger',
		description: 'Cross-border trading system handling Shipping Bills, Bill of Lading, Certificate of Origin, and multi-currency exchange.',
		priceMonthly: 1999,
		priceAnnual: 19990,
		accentVar: 'var(--accent-travel)',
		routes: ['/trade'],
		features: [
			'Shipping Bill, Packing List & Commercial Export Invoice',
			'Container tracking (Port of Loading → Destination Port)',
			'Foreign currency realization & BRC certificate ledger',
			'Incoterms (FOB, CIF, EXW) rate calculators'
		]
	},
	{
		id: 'support',
		name: 'Support Helpdesk & SLAs',
		icon: '🎫',
		category: 'AI_GROWTH',
		tagline: 'Ticket lifecycle management with automated SLA breach alarms',
		description: 'Customer service desk with multi-tier ticket escalation, agent response time tracking, and customer satisfaction surveys.',
		priceMonthly: 899,
		priceAnnual: 8990,
		accentVar: 'var(--status-info)',
		routes: ['/support'],
		features: [
			'SLA policy engine with automated breach escalation',
			'Customer self-service portal & knowledge base',
			'Canned quick responses & internal team notes',
			'CSAT survey generator after ticket resolution'
		]
	},
	{
		id: 'journeys',
		name: 'Client Journey Lifecycle Engine',
		icon: '🛤️',
		category: 'AI_GROWTH',
		tagline: 'Visual client journey milestones tracking onboarding to renewal',
		description: 'High-touch client success pipeline ensuring deliverable execution, milestone check-ins, and automated renewal alerts.',
		priceMonthly: 999,
		priceAnnual: 9990,
		accentVar: 'var(--accent-ai)',
		routes: ['/journeys'],
		features: [
			'Multi-stage client journey visual pipeline',
			'Deliverable checklist & automated email milestones',
			'Customer Health Score & churn risk alarms',
			'Annual renewal contract triggers'
		]
	},
	{
		id: 'settings',
		name: 'Company Profile & Settings',
		icon: '⚙️',
		category: 'CORE',
		tagline: 'GSTIN, banking setup, digital signatures and corporate policies',
		description: 'Global organization parameters, master tax rates, staff permissions, and legal jurisdiction settings.',
		priceMonthly: 0,
		priceAnnual: 0,
		accentVar: 'var(--text-muted)',
		routes: ['/settings'],
		features: [
			'GSTIN, PAN, and corporate address management',
			'Bank account, IFSC, and UPI settlement setup',
			'Digital signature and logo white-labeling',
			'Terms of service, warranty and return policies'
		],
		isCore: true
	}
];

class ModuleStore {
	// Map of moduleId -> License object
	licenses = $state<Record<string, ModuleLicense>>({});
	initialized = $state(false);

	constructor() {
		if (typeof window !== 'undefined') {
			this.loadFromStorage();
		}
	}

	loadFromStorage() {
		try {
			const saved = localStorage.getItem('mrbiz_module_licenses');
			if (saved) {
				this.licenses = JSON.parse(saved);
			} else {
				// Initialize with baseline: Core modules active, plus Projects as starting workspace
				const baseline: Record<string, ModuleLicense> = {
					billing: { moduleId: 'billing', status: 'ACTIVE', plan: 'CORE' },
					settings: { moduleId: 'settings', status: 'ACTIVE', plan: 'CORE' },
					pos: { moduleId: 'pos', status: 'ACTIVE', plan: 'MONTHLY', paidAmount: 699 },
					projects: { moduleId: 'projects', status: 'ACTIVE', plan: 'MONTHLY', paidAmount: 1299 }
				};
				this.licenses = baseline;
				this.saveToStorage();
			}
		} catch (e) {
			this.licenses = {
				billing: { moduleId: 'billing', status: 'ACTIVE', plan: 'CORE' },
				settings: { moduleId: 'settings', status: 'ACTIVE', plan: 'CORE' },
				projects: { moduleId: 'projects', status: 'ACTIVE', plan: 'MONTHLY' }
			};
		} finally {
			this.initialized = true;
		}
	}

	saveToStorage() {
		if (typeof window !== 'undefined') {
			try {
				localStorage.setItem('mrbiz_module_licenses', JSON.stringify(this.licenses));
			} catch (e) {}
		}
	}

	isModuleActive(moduleId: string): boolean {
		// Core modules are always active
		if (moduleId === 'billing' || moduleId === 'settings') return true;
		const lic = this.licenses[moduleId];
		return lic?.status === 'ACTIVE' || lic?.status === 'TRIAL';
	}

	getLicense(moduleId: string): ModuleLicense | undefined {
		return this.licenses[moduleId];
	}

	getModuleDescriptor(moduleId: string): ModuleDescriptor | undefined {
		return MODULE_CATALOG.find((m) => m.id === moduleId);
	}

	getModuleForRoute(pathname: string): ModuleDescriptor | undefined {
		// Match longest matching prefix
		for (const m of MODULE_CATALOG) {
			for (const r of m.routes) {
				if (pathname === r || (pathname.startsWith(r + '/') && r !== '/')) {
					return m;
				}
			}
		}
		return undefined;
	}

	/**
	 * Pay and activate ONLY the selected module without mixing other modules.
	 */
	activateModule(moduleId: string, plan: 'MONTHLY' | 'ANNUAL' = 'MONTHLY', paymentMethod: string = 'Instant UPI'): void {
		const desc = this.getModuleDescriptor(moduleId);
		if (!desc) return;

		const price = plan === 'ANNUAL' ? desc.priceAnnual : desc.priceMonthly;
		const now = new Date();
		const expires = new Date();
		if (plan === 'ANNUAL') {
			expires.setFullYear(expires.getFullYear() + 1);
		} else {
			expires.setMonth(expires.getMonth() + 1);
		}

		// Activate ONLY this single module
		this.licenses[moduleId] = {
			moduleId,
			status: 'ACTIVE',
			plan,
			activatedAt: now.toISOString(),
			expiresAt: expires.toISOString(),
			paidAmount: price
		};

		this.saveToStorage();

		toast.success(
			`⚡ Module Activated: ${desc.name}`,
			`Paid ₹${price.toLocaleString('en-IN')} via ${paymentMethod}. Access unlocked for ${desc.name} exclusively.`
		);
	}

	/**
	 * Start a 14-day free trial for ONLY the selected module.
	 */
	startTrial(moduleId: string): void {
		const desc = this.getModuleDescriptor(moduleId);
		if (!desc) return;

		const now = new Date();
		const expires = new Date();
		expires.setDate(expires.getDate() + 14);

		this.licenses[moduleId] = {
			moduleId,
			status: 'TRIAL',
			plan: 'MONTHLY',
			activatedAt: now.toISOString(),
			expiresAt: expires.toISOString(),
			paidAmount: 0
		};

		this.saveToStorage();

		toast.info(
			`🎁 Free Trial Activated: ${desc.name}`,
			`14-day full access enabled until ${expires.toLocaleDateString('en-IN')}. No other modules affected.`
		);
	}

	/**
	 * Deactivate an individual module.
	 */
	deactivateModule(moduleId: string): void {
		const desc = this.getModuleDescriptor(moduleId);
		if (desc?.isCore) {
			toast.warning('Core Module Protected', 'Billing and System Settings cannot be deactivated.');
			return;
		}

		delete this.licenses[moduleId];
		this.saveToStorage();

		toast.warning(
			`Module Deactivated: ${desc?.name || moduleId}`,
			'Access to this module has been paused.'
		);
	}

	get activeCount(): number {
		return Object.values(this.licenses).filter((l) => l.status === 'ACTIVE' || l.status === 'TRIAL').length;
	}

	get totalMonthlyCommitment(): number {
		let total = 0;
		for (const lic of Object.values(this.licenses)) {
			if (lic.status === 'ACTIVE') {
				const desc = this.getModuleDescriptor(lic.moduleId);
				if (desc) {
					total += lic.plan === 'ANNUAL' ? Math.round(desc.priceAnnual / 12) : desc.priceMonthly;
				}
			}
		}
		return total;
	}
}

export const moduleStore = new ModuleStore();
