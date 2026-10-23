// Data-driven navigation model for the application shell.
//
// Previously the sidebar nav was a ~700-line {#if activeWorkspacePreset}
// chain that re-listed the same shared routes with reworded labels for each
// of nine workspaces. This module encodes that content as data so the shell
// renders one grouped, collapsible nav from the active workspace's config —
// removing the duplication and giving every workspace consistent IA.
//
// The workspace ids and their `unit` values are kept identical to the old
// preset system so the existing `mrbiz_active_unit` contract (read by grpc.ts
// and several pages) and `mrbiz_workspace_preset` persistence keep working.

export type WorkspacePreset =
	| 'FULL'
	| 'COUNTER'
	| 'GROWTH'
	| 'OPERATIONS'
	| 'HEALTHCARE'
	| 'TRAVEL'
	| 'REALESTATE'
	| 'PROJECTS'
	| 'COMMERCE'
	| 'CUSTOM';

export interface NavItem {
	href: string;
	label: string;
	icon: string;
	/** The workspace's landing / headline destination, visually emphasised. */
	primary?: boolean;
	/** Active when currentPath === match; otherwise active when it startsWith match. */
	exact?: boolean;
	/** Path used for active-state matching when it differs from href (e.g. a "/new" link). */
	match?: string;
	/** Gated behind the Next (Business) plan — shows a lock and opens the upgrade modal. */
	gated?: boolean;
	lockTitle?: string;
	lockDesc?: string;
}

export interface NavGroup {
	/** Section header. Groups with a label are collapsible; headerless groups are not. */
	label?: string;
	badge?: string;
	/** Marks a group that only unlocks on the Next plan, so the shell can badge it. */
	nextPlan?: boolean;
	items: NavItem[];
}

export interface Workspace {
	id: WorkspacePreset;
	icon: string;
	name: string;
	badge: string;
	desc: string;
	/** Value written to `mrbiz_active_unit` / sent to the backend as the business unit. */
	unit?: string;
	/** Route to navigate to when this workspace is activated. */
	landing: string;
	groups: NavGroup[];
}

const ACCOUNTING_GATE = {
	gated: true,
	lockTitle: 'Audited Financial Accounting',
	lockDesc:
		'Full Double-Entry Accounting, General Ledger, Chart of Accounts, Trial Balance, and Profit & Loss statement.'
};
const EXPENSE_GATE = {
	gated: true,
	lockTitle: 'Company Expense Management',
	lockDesc:
		'Track and categorize company expenses, record vendor receipts, manage payment modes, and automate financial expense postings.'
};

export const workspaces: Record<Exclude<WorkspacePreset, 'CUSTOM'>, Workspace> = {
	FULL: {
		id: 'FULL',
		icon: '🌐',
		name: 'Enterprise ERP Suite',
		badge: 'All Modules',
		desc: 'Complete 16-module multi-industry ERP',
		unit: 'billing',
		landing: '/',
		groups: [
			{
				label: 'Business Applications',
				items: [
					{ href: '/services', label: 'All Services Hub', icon: '🌐', exact: true, primary: true },
					{ href: '/websites', label: 'Multi-Site CMS Builder', icon: '🖥️' },
					{ href: '/marketing', label: 'Omnichannel Marketing Hub', icon: '📣' },
					{ href: '/ecommerce', label: 'Universal E-Commerce', icon: '🛍️' },
					{ href: '/inbox', label: 'Omnichannel Social Inbox', icon: '💬' },
					{ href: '/bpo-voice', label: 'AI Voice Agent BPO Desk', icon: '🎙️' },
					{ href: '/support', label: 'Support Helpdesk & SLAs', icon: '🎫' },
					{ href: '/projects', label: 'Projects & UTMS Tasks', icon: '📁' },
					{ href: '/journeys', label: 'Journey Lifecycle Engine', icon: '🛤️' },
					{ href: '/procurement', label: 'Procurement & 3-Way Match', icon: '📦' },
					{ href: '/accounting/reconciliation', label: 'Bank Auto-Reconciliation', icon: '🏦' },
					{ href: '/hrms', label: 'HRMS & 1-Click Payroll', icon: '👥' },
					{ href: '/workflow', label: 'AI Autonomous Missions', icon: '🤖' },
					{ href: '/healthcare', label: 'Smart Healthcare Hub', icon: '🏥' },
					{ href: '/travel', label: 'Travel & Visa OS', icon: '✈️' },
					{ href: '/realestate', label: 'Real Estate Deals', icon: '🏢' }
				]
			},
			{
				label: 'Core Invoicing & Billing',
				items: [
					{ href: '/billing', label: 'Express Billing Counter', icon: '⚡' },
					{ href: '/', label: 'Dashboard', icon: '📊', exact: true },
					{ href: '/pos', label: 'Retail POS Terminal', icon: '🛒' },
					{ href: '/invoices', label: 'Sales Invoices', icon: '📄' },
					{ href: '/quotations', label: 'Quotations', icon: '📝' },
					{ href: '/trade/shipments', label: 'Export / Import Shipments', icon: '🚢' },
					{ href: '/trade/export-invoice/new', label: 'New Export Invoice', icon: '🌍', match: '/trade/export-invoice' },
					{ href: '/trade/settings', label: 'Trade Settings & Rates', icon: '💱', exact: true },
					{ href: '/invoices/recurring', label: 'Recurring Invoices', icon: '🔁', exact: true },
					{ href: '/purchases', label: 'Purchase Bills', icon: '📥' },
					{ href: '/credit-notes', label: 'Credit & Debit Notes', icon: '🔄' },
					{ href: '/payments', label: 'Payments Received', icon: '💳', exact: true },
					{ href: '/customers', label: 'Customers & Balances', icon: '👥', exact: true },
					{ href: '/suppliers', label: 'Suppliers & Khata', icon: '🏭' },
					{ href: '/products', label: 'Products & Pricing', icon: '📦', exact: true },
					{ href: '/inventory', label: 'Inventory Stock', icon: '📋', exact: true },
					{ href: '/units', label: 'Units & Converter', icon: '⚖️', exact: true }
				]
			},
			{
				label: 'Financial Suite',
				nextPlan: true,
				items: [
					{ href: '/accounting', label: 'Audited Accounting', icon: '📖', ...ACCOUNTING_GATE },
					{ href: '/expenses', label: 'Expense Management', icon: '💸', ...EXPENSE_GATE }
				]
			},
			{
				label: 'Administration',
				items: [
					{ href: '/settings', label: 'Company Profile', icon: '⚙️', exact: true },
					{ href: '/settings/team', label: 'Team & Staff Users', icon: '👥', exact: true },
					{ href: '/settings/security', label: 'Account Security', icon: '🔐', exact: true },
					{ href: '/settings/billing', label: 'Plan & Billing', icon: '💳', exact: true },
					{ href: '/accounting/exchange', label: 'Returns & Data Exchange', icon: '🔄', exact: true },
					{ href: '/settings/templates', label: 'Billing Templates', icon: '🎨', exact: true },
					{ href: '/settings/templates/studio', label: 'Bill Template Studio', icon: '🛠️', exact: true },
					{ href: '/settings/developer', label: 'Developer & API Keys', icon: '⚡', exact: true }
				]
			}
		]
	},
	COUNTER: {
		id: 'COUNTER',
		icon: '⚡',
		name: 'Express Retail Counter',
		badge: 'POS Terminal',
		desc: 'Ultra-fast retail counter with instant UPI QR',
		unit: 'billing',
		landing: '/billing',
		groups: [
			{
				label: 'Express Retail Counter',
				items: [
					{ href: '/billing', label: 'Express Billing Counter', icon: '⚡', primary: true },
					{ href: '/', label: 'Quick Dashboard', icon: '📊', exact: true },
					{ href: '/pos', label: 'Retail POS Terminal', icon: '🛒' },
					{ href: '/invoices', label: 'Sales Invoices', icon: '📄' },
					{ href: '/quotations', label: 'Quotations & Estimates', icon: '📝' },
					{ href: '/payments', label: 'Payments Received', icon: '💳', exact: true },
					{ href: '/customers', label: 'Customers & Balances', icon: '👥', exact: true },
					{ href: '/products', label: 'Products & Pricing', icon: '📦', exact: true },
					{ href: '/inventory', label: 'Inventory Stock', icon: '📋', exact: true }
				]
			},
			{
				label: 'Administration',
				items: [
					{ href: '/settings/templates', label: '17 Billing Templates', icon: '🎨', exact: true },
					{ href: '/settings', label: 'Company Profile & QR', icon: '⚙️', exact: true }
				]
			}
		]
	},
	HEALTHCARE: {
		id: 'HEALTHCARE',
		icon: '🏥',
		name: 'Smart Healthcare OS',
		badge: 'Clinical Suite',
		desc: 'Doctors, Telehealth, EMR Rx Pad, Pharmacy POS',
		unit: 'healthcare',
		landing: '/healthcare',
		groups: [
			{
				label: 'Clinical Suite',
				badge: '4 Workstations',
				items: [
					{ href: '/healthcare', label: 'Clinical Workstations (Rx/EMR)', icon: '🏥', primary: true },
					{ href: '/pos', label: 'Pharmacy POS Dispenser', icon: '🛒' },
					{ href: '/customers', label: 'Patient Records & History', icon: '👥', exact: true },
					{ href: '/products', label: 'Pharmacy Formulary / Drugs', icon: '📦', exact: true },
					{ href: '/inventory', label: 'Drug Batches & Expiry Dates', icon: '📋', exact: true },
					{ href: '/inbox', label: 'Patient WhatsApp Helpline', icon: '💬' },
					{ href: '/hrms', label: 'Doctors & Nursing Roster', icon: '👥' }
				]
			},
			{
				label: 'Finance & Billing',
				items: [
					{ href: '/billing', label: 'Quick Patient OPD Billing', icon: '⚡' },
					{ href: '/invoices', label: 'Invoices & Insurance Claims', icon: '📄' },
					{ href: '/accounting', label: 'Hospital Accounts & P&L', icon: '📖' }
				]
			},
			{
				label: 'Administration',
				items: [{ href: '/settings', label: 'Clinic Profile & Rx Header', icon: '⚙️', exact: true }]
			}
		]
	},
	TRAVEL: {
		id: 'TRAVEL',
		icon: '✈️',
		name: 'Travel & Tours OS',
		badge: 'Agency Operations',
		desc: 'Complete Tour Operator & Travel Agency OS',
		unit: 'travel',
		landing: '/travel',
		groups: [
			{
				label: 'Agency Operations',
				items: [
					{ href: '/travel', label: 'Dashboard', icon: '📊', exact: true, primary: true },
					{ href: '/travel/book', label: '⚡ Book a Trip (1-Page)', icon: '📝' },
					{ href: '/travel/booking', label: '🛒 Consumer Booking Wizard', icon: '🎫' },
					{ href: '/travel/crm', label: 'CRM & Leads (Odoo)', icon: '👥' },
					{ href: '/travel/packages', label: 'Tour Packages & Costing', icon: '📦' },
					{ href: '/travel/visa', label: 'Visa Processing & Fees', icon: '🛂' },
					{ href: '/travel?view=bookings', label: 'Bookings List', icon: '✈️', exact: true },
					{ href: '/travel?view=customers', label: 'Customers', icon: '👤', exact: true },
					{ href: '/travel?view=departures', label: 'Departures', icon: '🛫', exact: true },
					{ href: '/travel?view=itineraries', label: 'Itineraries & Planning', icon: '🗺️', exact: true },
					{ href: '/travel?view=tickets', label: 'Ticket Management', icon: '🎟️', exact: true }
				]
			},
			{
				label: 'Client & Partner Portals',
				items: [
					{ href: '/customer/services', label: 'Customer Concierge & PNR', icon: '🛎️' },
					{ href: '/travel/partners', label: 'B2B Partner Portal & Tiers', icon: '🤝' },
					{ href: '/login/partner', label: 'Partner Khata Ledger Login', icon: '🔐' },
					{ href: '/login/crew', label: 'Chauffeur / Guide Duty Slip', icon: '🚗' }
				]
			},
			{
				label: 'Fleet & Partner Logistics',
				items: [
					{ href: '/travel/drivers', label: 'Drivers & Car Agencies', icon: '🚗' },
					{ href: '/travel/guides', label: 'Local Tour Guides', icon: '🧭' },
					{ href: '/travel/agents', label: 'Tour Agents & Partners', icon: '🤝' },
					{ href: '/travel?view=vendors', label: 'Vendors & Suppliers', icon: '🏢', exact: true },
					{ href: '/travel?view=finance', label: 'Finance & Billing', icon: '💳', exact: true },
					{ href: '/travel?view=reports', label: 'Reports & Analytics', icon: '📈', exact: true },
					{ href: '/travel?view=settings', label: 'Settings', icon: '⚙️', exact: true }
				]
			}
		]
	},
	REALESTATE: {
		id: 'REALESTATE',
		icon: '🏢',
		name: 'Real Estate Matrix',
		badge: 'Plots & RERA',
		desc: '24-plot layout matrix, Indian land units & visits',
		unit: 'realestate',
		landing: '/realestate',
		groups: [
			{
				label: 'Deals Desk',
				badge: 'Plots & RERA',
				items: [
					{ href: '/realestate', label: '24-Plot Matrix & Site Visits', icon: '🏢', primary: true },
					{ href: '/bpo-voice', label: 'AI Voice Site Visit Dialer', icon: '🎙️' },
					{ href: '/inbox', label: 'Buyer WhatsApp & Inquiries', icon: '💬' },
					{ href: '/projects', label: 'Layout Development Tasks', icon: '📁' },
					{ href: '/customers', label: 'Property Buyers & Leads', icon: '👥', exact: true }
				]
			},
			{
				label: 'Finance & Billing',
				items: [
					{ href: '/billing', label: 'Plot Advance Token Receipt', icon: '⚡' },
					{ href: '/invoices', label: 'Sales Invoices & Agreements', icon: '📄' },
					{ href: '/quotations', label: 'Plot Cost Sheets & Quotations', icon: '📝' },
					{ href: '/accounting', label: 'Project Escrow & Land Ledger', icon: '📖' }
				]
			},
			{
				label: 'Administration',
				items: [{ href: '/settings', label: 'Builder & RERA Setup', icon: '⚙️', exact: true }]
			}
		]
	},
	PROJECTS: {
		id: 'PROJECTS',
		icon: '📁',
		name: 'UTMS Delivery OS',
		badge: 'SLA Engine',
		desc: 'SLA engine, Kanban pipeline & milestone billing',
		unit: 'consulting',
		landing: '/projects',
		groups: [
			{
				label: 'Delivery Desk',
				badge: 'SLA Engine',
				items: [
					{ href: '/projects', label: 'UTMS Tasks & SLA Command Center', icon: '📁', primary: true },
					{ href: '/support', label: 'Client SLA Helpdesk', icon: '🎫' },
					{ href: '/inbox', label: 'Omnichannel Client Inbox', icon: '💬' },
					{ href: '/workflow', label: 'Autonomous AI Workflows', icon: '🤖' },
					{ href: '/hrms', label: 'Timesheets & Team Payroll', icon: '👥' }
				]
			},
			{
				label: 'Finance & Billing',
				items: [
					{ href: '/billing', label: 'Milestone Billing Triggers', icon: '⚡' },
					{ href: '/invoices', label: 'Client Milestone Invoices', icon: '📄' },
					{ href: '/customers', label: 'Client SOW Accounts', icon: '👥', exact: true },
					{ href: '/accounting', label: 'Consulting SOW Ledger & P&L', icon: '📖' }
				]
			},
			{
				label: 'Administration',
				items: [{ href: '/settings', label: 'Agency Delivery Settings', icon: '⚙️', exact: true }]
			}
		]
	},
	COMMERCE: {
		id: 'COMMERCE',
		icon: '🛍️',
		name: 'E-Commerce & Multi-Site',
		badge: 'Store & Labs',
		desc: 'Online storefronts, social marketing & blank matrix',
		unit: 'ecommerce',
		landing: '/ecommerce',
		groups: [
			{
				label: 'Store & Marketing',
				badge: 'Online & Physical',
				items: [
					{ href: '/ecommerce', label: 'Universal Store & Catalog', icon: '🛍️', primary: true },
					{ href: '/websites', label: 'Multi-Business Web Builder', icon: '🖥️' },
					{ href: '/marketing', label: 'Omnichannel Social Marketing', icon: '📣' },
					{ href: '/pos', label: 'Retail Counter POS', icon: '🛒' },
					{ href: '/inventory', label: 'Garment Stock & Color/Size', icon: '📋', exact: true },
					{ href: '/trade/shipments', label: 'Courier Logistics & NDR', icon: '🚢' },
					{ href: '/inbox', label: 'Omnichannel Buyer Support', icon: '💬' }
				]
			},
			{
				label: 'Finance & Billing',
				items: [
					{ href: '/billing', label: 'Quick Order Invoicing', icon: '⚡' },
					{ href: '/procurement', label: '3-Way Blank Procurement', icon: '📦' },
					{ href: '/accounting', label: 'Store Financial Accounting', icon: '📖' }
				]
			},
			{
				label: 'Administration',
				items: [{ href: '/settings', label: 'Print Lab & Store Settings', icon: '⚙️', exact: true }]
			}
		]
	},
	GROWTH: {
		id: 'GROWTH',
		icon: '💬',
		name: 'Social Sales & AI Voice',
		badge: 'Growth Desk',
		desc: 'Omnichannel WhatsApp/IG, AI voice & deals',
		landing: '/inbox',
		groups: [
			{
				label: 'Growth & Omnichannel Sales',
				items: [
					{ href: '/inbox', label: 'Omnichannel Social Inbox', icon: '💬', primary: true },
					{ href: '/bpo-voice', label: 'AI Voice Agent BPO Desk', icon: '🎙️' },
					{ href: '/support', label: 'Support Helpdesk & SLAs', icon: '🎫' },
					{ href: '/journeys', label: 'Journey Lifecycle Engine', icon: '🛤️' },
					{ href: '/projects', label: 'Projects & UTMS Tasks', icon: '📁' }
				]
			},
			{
				label: 'Industry Desks',
				items: [
					{ href: '/realestate', label: 'Real Estate Deals', icon: '🏢' },
					{ href: '/travel', label: 'Travel & Visa OS', icon: '✈️' },
					{ href: '/healthcare', label: 'Smart Healthcare Hub', icon: '🏥' }
				]
			},
			{
				label: 'Finance & Billing',
				items: [
					{ href: '/billing', label: 'Quick Billing Counter', icon: '⚡' },
					{ href: '/customers', label: 'Customers & CRM', icon: '👥', exact: true }
				]
			}
		]
	},
	OPERATIONS: {
		id: 'OPERATIONS',
		icon: '📦',
		name: 'Supply Chain & Ops',
		badge: '3-Way Match',
		desc: '3-way procurement, batches & vendor khata',
		landing: '/procurement',
		groups: [
			{
				label: 'Supply Chain & Trade',
				items: [
					{ href: '/procurement', label: 'Procurement & 3-Way Match', icon: '📦', primary: true },
					{ href: '/inventory', label: 'Inventory Stock & Batches', icon: '📋', exact: true },
					{ href: '/trade/shipments', label: 'Export / Import Shipments', icon: '🚢' },
					{ href: '/trade/export-invoice/new', label: 'New Export Invoice', icon: '🌍', match: '/trade/export-invoice' },
					{ href: '/trade/settings', label: 'Trade Settings & Rates', icon: '💱', exact: true },
					{ href: '/suppliers', label: 'Suppliers & Khata', icon: '🏭' }
				]
			},
			{
				label: 'Finance & Billing',
				items: [
					{ href: '/purchases', label: 'Purchase Bills', icon: '📥' },
					{ href: '/products', label: 'Products & Pricing', icon: '📦', exact: true },
					{ href: '/units', label: 'Units & Converter', icon: '⚖️', exact: true },
					{ href: '/accounting/reconciliation', label: 'Bank Auto-Reconciliation', icon: '🏦' },
					{ href: '/accounting', label: 'Audited Accounting', icon: '📖' }
				]
			}
		]
	}
};

/** Order the switcher presents workspaces in. */
export const workspaceOrder: WorkspacePreset[] = [
	'FULL',
	'HEALTHCARE',
	'TRAVEL',
	'REALESTATE',
	'PROJECTS',
	'COUNTER',
	'COMMERCE',
	'GROWTH',
	'OPERATIONS'
];

export const workspaceList: Workspace[] = workspaceOrder
	.filter((id): id is Exclude<WorkspacePreset, 'CUSTOM'> => id !== 'CUSTOM')
	.map((id) => workspaces[id]);

/** Is a nav item the active route for the given path? */
export function isNavItemActive(item: NavItem, path: string, search = ''): boolean {
	const target = item.match ?? item.href;
	const fullCurrent = search ? `${path}${search}` : path;
	if (target.includes('?')) {
		return fullCurrent === target;
	}
	return item.exact ? (path === target && (!search || target !== '/travel')) : path.startsWith(target);
}
