// ConnectRPC / gRPC-Web Client for MrBiz-OS SvelteKit Application

const GRPC_BASE_URL = typeof window !== 'undefined' && window.location.origin.includes('localhost')
	? 'http://localhost:8080'
	: '';

export interface RpcOptions {
	token?: string;
	businessUnitId?: string;
	tenantId?: string;
}

function translateRpcError(err: any): Error {
	const raw = String(err?.message || err || '');
	if (raw.includes('Failed to fetch') || raw.includes('NetworkError') || raw.includes('Load failed') || raw.includes('unavailable')) {
		return new Error('📡 Server is currently unreachable. Please check your internet connection; all local drafts remain safe.');
	}
	if (raw.includes('401') || raw.toLowerCase().includes('unauthenticated')) {
		return new Error('🔒 Session has expired. Please sign in again.');
	}
	if (raw.includes('403') || raw.toLowerCase().includes('permission_denied')) {
		return new Error('🚫 Access denied. You do not have permission for this business operation.');
	}
	if (raw.includes('404') || raw.toLowerCase().includes('not_found')) {
		return new Error('🔍 The requested record could not be found.');
	}
	if (raw.includes('409') || raw.toLowerCase().includes('already_exists')) {
		return new Error('📋 A bill or record with this document number already exists.');
	}
	return err instanceof Error ? err : new Error(raw);
}

export async function callRpc<TReq = any, TRes = any>(
	service: string,
	method: string,
	req?: TReq,
	opts?: RpcOptions
): Promise<TRes> {
	const url = `${GRPC_BASE_URL}/${service}/${method}`;
	const headers: Record<string, string> = {
		'Content-Type': 'application/json',
		'Connect-Protocol-Version': '1',
	};

	const token = opts?.token || (typeof localStorage !== 'undefined' ? localStorage.getItem('saas_token') : '') || 'superadmin-session-active';
	headers['Authorization'] = `Bearer ${token}`;

	const buId = opts?.businessUnitId || (typeof localStorage !== 'undefined' ? localStorage.getItem('mrbiz_active_unit') : '') || 'healthcare';
	headers['X-Business-Unit-Id'] = buId;

	const tenantId = opts?.tenantId || (typeof localStorage !== 'undefined' ? localStorage.getItem('saas_tenant_id') : '') || 'tenant-1';
	headers['X-Tenant-Id'] = tenantId;

	try {
		const res = await fetch(url, {
			method: 'POST',
			headers,
			body: JSON.stringify(req || {}),
			credentials: 'include',
		});

		if (!res.ok) {
			const errJson = await res.json().catch(() => ({}));
			const msg = errJson.message || `RPC Error (${res.status}): ${res.statusText}`;
			throw new Error(msg);
		}

		return (await res.json()) as TRes;
	} catch (err: any) {
		throw translateRpcError(err);
	}
}

// 0. IAM & Identity Service Client
export const iamGrpc = {
	login: (req: { email: string; password?: string }) => callRpc('iam.v1.IAMService', 'Login', req),
	logout: (req = {}) => callRpc('iam.v1.IAMService', 'Logout', req),
	getMe: (req: { userId?: string; tenantId?: string } = {}) => callRpc('iam.v1.IAMService', 'GetMe', req),
	validateSession: (req: { token: string }) => callRpc('iam.v1.IAMService', 'ValidateSession', req),
	switchBusinessUnit: (req: { userId: string; targetBusinessUnitId: string }) => callRpc('iam.v1.IAMService', 'SwitchBusinessUnit', req),
	listBusinessUnits: (params = {}) => callRpc('iam.v1.IAMService', 'ListBusinessUnits', params),
	listTenants: (params = {}) => callRpc('iam.v1.IAMService', 'ListTenants', params),
	createTenant: (req: any) => callRpc('iam.v1.IAMService', 'CreateTenant', req),
};

// 1. Healthcare Service Client
export const healthcareGrpc = {
	listDoctors: (params = {}) => callRpc('healthcare.v1.HealthcareService', 'ListDoctors', params),
	listAppointments: (params = {}) => callRpc('healthcare.v1.HealthcareService', 'ListAppointments', params),
	bookAppointment: (req: any) => callRpc('healthcare.v1.HealthcareService', 'BookAppointment', req),
	createPrescription: (req: any) => callRpc('healthcare.v1.HealthcareService', 'CreatePrescription', req),
	updateAppointmentStatus: (req: any) => callRpc('healthcare.v1.HealthcareService', 'UpdateAppointmentStatus', req),
};

// 2. Travel & Visa Service Client
export const travelGrpc = {
	listTourPackages: (params = {}) => callRpc('travel.v1.TravelService', 'ListTourPackages', params),
	createTourPackage: (req: any) => callRpc('travel.v1.TravelService', 'CreateTourPackage', req),
	listVisaApplications: (params = {}) => callRpc('travel.v1.TravelService', 'ListVisaApplications', params),
	submitVisaApplication: (req: any) => callRpc('travel.v1.TravelService', 'SubmitVisaApplication', req),
	updateVisaStatus: (req: any) => callRpc('travel.v1.TravelService', 'UpdateVisaStatus', req),
};

// 3. Real Estate Service Client
export const realestateGrpc = {
	listProperties: (params = {}) => callRpc('realestate.v1.RealEstateService', 'ListProperties', params),
	createProperty: (req: any) => callRpc('realestate.v1.RealEstateService', 'CreateProperty', req),
	scheduleSiteVisit: (req: any) => callRpc('realestate.v1.RealEstateService', 'ScheduleSiteVisit', req),
	listSiteVisits: (params = {}) => callRpc('realestate.v1.RealEstateService', 'ListSiteVisits', params),
	createDeal: (req: any) => callRpc('realestate.v1.RealEstateService', 'CreatePropertyDeal', req),
	listDeals: (params = {}) => callRpc('realestate.v1.RealEstateService', 'ListPropertyDeals', params),
};

// 4. Workflow & AI Approval Client
export const workflowGrpc = {
	listTasks: (params = {}) => callRpc('workflow.v1.WorkflowService', 'ListTasks', params),
	createTask: (req: any) => callRpc('workflow.v1.WorkflowService', 'CreateTask', req),
	listApprovalQueue: (params = {}) => callRpc('workflow.v1.WorkflowService', 'ListApprovalQueue', params),
	approveAction: (req: any) => callRpc('workflow.v1.WorkflowService', 'ApproveAction', req),
	rejectAction: (req: any) => callRpc('workflow.v1.WorkflowService', 'RejectAction', req),
};

// 5. Procurement & 3-Way Matching Client
export const procurementGrpc = {
	listPurchaseOrders: (params = {}) => callRpc('procurement.v1.ProcurementService', 'ListPurchaseOrders', params),
	createPurchaseOrder: (req: any) => callRpc('procurement.v1.ProcurementService', 'CreatePurchaseOrder', req),
	receiveGoods: (req: any) => callRpc('procurement.v1.ProcurementService', 'ReceiveGoods', req),
	listGoodsReceived: (params = {}) => callRpc('procurement.v1.ProcurementService', 'ListGoodsReceived', params),
	verifyThreeWayMatch: (req: any) => callRpc('procurement.v1.ProcurementService', 'VerifyThreeWayMatch', req),
	checkLowStockAlerts: (params = {}) => callRpc('procurement.v1.ProcurementService', 'CheckLowStockAlerts', params),
};

// 6. Banking & Automated Reconciliation Client
export const bankingGrpc = {
	importBankStatement: (req: any) => callRpc('banking.v1.BankingService', 'ImportBankStatement', req),
	listBankTransactions: (params = {}) => callRpc('banking.v1.BankingService', 'ListBankTransactions', params),
	runAutoReconciliation: (params = {}) => callRpc('banking.v1.BankingService', 'RunAutoReconciliation', params),
	manualMatchTransaction: (req: any) => callRpc('banking.v1.BankingService', 'ManualMatchTransaction', req),
	getReconciliationSummary: (params = {}) => callRpc('banking.v1.BankingService', 'GetReconciliationSummary', params),
};

// 7. HRMS & Automated Payroll Client
export const hrmsGrpc = {
	listEmployees: (params = {}) => callRpc('hrms.v1.HRMSService', 'ListEmployees', params),
	createEmployee: (req: any) => callRpc('hrms.v1.HRMSService', 'CreateEmployee', req),
	updateSalaryStructure: (req: any) => callRpc('hrms.v1.HRMSService', 'UpdateSalaryStructure', req),
	runMonthlyPayroll: (req: any) => callRpc('hrms.v1.HRMSService', 'RunMonthlyPayroll', req),
	listPayslips: (params = {}) => callRpc('hrms.v1.HRMSService', 'ListPayslips', params),
};

// 8. Project & UTMS Task Management Client
export const projectGrpc = {
	listProjects: (params = {}) => callRpc('project.v1.ProjectService', 'ListProjects', params),
	createProject: (req: any) => callRpc('project.v1.ProjectService', 'CreateProject', req),
	getProject: (req: any) => callRpc('project.v1.ProjectService', 'GetProject', req),
	listTasks: (params = {}) => callRpc('project.v1.ProjectService', 'ListTasks', params),
	createTask: (req: any) => callRpc('project.v1.ProjectService', 'CreateTask', req),
	updateTaskStatus: (req: any) => callRpc('project.v1.ProjectService', 'UpdateTaskStatus', req),
	delegateTask: (req: any) => callRpc('project.v1.ProjectService', 'DelegateTask', req),
	listMilestones: (params = {}) => callRpc('project.v1.ProjectService', 'ListMilestones', params),
	createMilestone: (req: any) => callRpc('project.v1.ProjectService', 'CreateMilestone', req),
	logTimesheet: (req: any) => callRpc('project.v1.ProjectService', 'LogTimesheet', req),
};

// 9. Journey Management Client
export const journeyGrpc = {
	listTemplates: (params = {}) => callRpc('journey.v1.JourneyService', 'ListTemplates', params),
	createTemplate: (req: any) => callRpc('journey.v1.JourneyService', 'CreateTemplate', req),
	listInstances: (params = {}) => callRpc('journey.v1.JourneyService', 'ListInstances', params),
	startJourney: (req: any) => callRpc('journey.v1.JourneyService', 'StartJourney', req),
	advanceStage: (req: any) => callRpc('journey.v1.JourneyService', 'AdvanceStage', req),
	getJourneyDetails: (req: any) => callRpc('journey.v1.JourneyService', 'GetJourneyDetails', req),
};

// 10. Omnichannel Social & Communication Hub Client (WhatsApp, Instagram, FB, Email)
export const omnichannelGrpc = {
	listConversations: (params = {}) => callRpc('omnichannel.v1.OmnichannelService', 'ListConversations', params),
	getConversation: (req: any) => callRpc('omnichannel.v1.OmnichannelService', 'GetConversation', req),
	sendMessage: (req: any) => callRpc('omnichannel.v1.OmnichannelService', 'SendMessage', req),
	updateConversationStatus: (req: any) => callRpc('omnichannel.v1.OmnichannelService', 'UpdateConversationStatus', req),
	convertToLeadOrTask: (req: any) => callRpc('omnichannel.v1.OmnichannelService', 'ConvertToLeadOrTask', req),
	suggestAIReply: (req: any) => callRpc('omnichannel.v1.OmnichannelService', 'SuggestAIReply', req),
};

// 11. Support & Enterprise Helpdesk Client
export const supportGrpc = {
	listTickets: (params = {}) => callRpc('support.v1.SupportService', 'ListTickets', params),
	createTicket: (req: any) => callRpc('support.v1.SupportService', 'CreateTicket', req),
	getTicketDetails: (req: any) => callRpc('support.v1.SupportService', 'GetTicketDetails', req),
	replyTicket: (req: any) => callRpc('support.v1.SupportService', 'ReplyTicket', req),
	updateTicketStatus: (req: any) => callRpc('support.v1.SupportService', 'UpdateTicketStatus', req),
	assignTicket: (req: any) => callRpc('support.v1.SupportService', 'AssignTicket', req),
	submitCSAT: (req: any) => callRpc('support.v1.SupportService', 'SubmitCSAT', req),
};

// 12. BPO AI Voice Agent Call Center Client
export const voiceBpoGrpc = {
	startInboundCallSession: (req: any) => callRpc('bpo.v1.VoiceBPOService', 'StartInboundCallSession', req),
	processVoiceTurn: (req: any) => callRpc('bpo.v1.VoiceBPOService', 'ProcessVoiceTurn', req),
	requestHumanHandoff: (req: any) => callRpc('bpo.v1.VoiceBPOService', 'RequestHumanHandoff', req),
	listActiveCalls: (params = {}) => callRpc('bpo.v1.VoiceBPOService', 'ListActiveCalls', params),
	launchOutboundCampaign: (req: any) => callRpc('bpo.v1.VoiceBPOService', 'LaunchOutboundCampaign', req),
};

// 13. E-Commerce & Print-on-Demand Client
export const ecommerceGrpc = {
	listOrders: (params = {}) => callRpc('ecommerce.v1.ECommerceService', 'ListOrders', params),
	getOrder: (req: any) => callRpc('ecommerce.v1.ECommerceService', 'GetOrder', req),
	createOrder: (req: any) => callRpc('ecommerce.v1.ECommerceService', 'CreateOrder', req),
	updateShipmentStatus: (req: any) => callRpc('ecommerce.v1.ECommerceService', 'UpdateShipmentStatus', req),
	listInventoryItems: (params = {}) => callRpc('ecommerce.v1.ECommerceService', 'ListInventoryItems', params),
	syncEtsyOrders: (params = {}) => callRpc('ecommerce.v1.ECommerceService', 'SyncEtsyOrders', params),
};

// 14. Software Consulting & SOW Client
export const consultingGrpc = {
	listProjects: (params = {}) => callRpc('consulting.v1.ConsultingService', 'ListProjects', params),
	getProject: (req: any) => callRpc('consulting.v1.ConsultingService', 'GetProject', req),
	createProject: (req: any) => callRpc('consulting.v1.ConsultingService', 'CreateProject', req),
	logTimesheet: (req: any) => callRpc('consulting.v1.ConsultingService', 'LogTimesheet', req),
	listTimesheets: (params = {}) => callRpc('consulting.v1.ConsultingService', 'ListTimesheets', params),
	createSOWContract: (req: any) => callRpc('consulting.v1.ConsultingService', 'CreateSOWContract', req),
	listSOWContracts: (params = {}) => callRpc('consulting.v1.ConsultingService', 'ListSOWContracts', params),
};

// 15. Fast Billing, Invoicing & GST Client
export const billingGrpc = {
	listInvoices: (params = {}) => callRpc('billing.v1.BillingService', 'ListInvoices', params),
	getInvoice: (req: any) => callRpc('billing.v1.BillingService', 'GetInvoice', req),
	createInvoice: (req: any) => callRpc('billing.v1.BillingService', 'CreateInvoice', req),
	cancelInvoice: (req: any) => callRpc('billing.v1.BillingService', 'CancelInvoice', req),
	updateLogistics: (req: any) => callRpc('billing.v1.BillingService', 'UpdateLogistics', req),
	generateInvoicePDF: (req: any) => callRpc('billing.v1.BillingService', 'GenerateInvoicePDF', req),
	recordPayment: (req: any) => callRpc('billing.v1.BillingService', 'RecordPayment', req),
	getBillingSummary: (params = {}) => callRpc('billing.v1.BillingService', 'GetBillingSummary', params),
	getTenantBillingTemplate: (params = {}) => callRpc('billing.v1.BillingService', 'GetTenantBillingTemplate', params),
	updateTenantBillingTemplate: (req: any) => callRpc('billing.v1.BillingService', 'UpdateTenantBillingTemplate', req),
};



