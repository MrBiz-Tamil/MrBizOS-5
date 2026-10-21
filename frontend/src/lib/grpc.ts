// ConnectRPC / gRPC-Web Client for MrBiz-OS SvelteKit Application

const GRPC_BASE_URL = typeof window !== 'undefined' && window.location.origin.includes('localhost')
	? 'http://localhost:8080'
	: '';

export interface RpcOptions {
	token?: string;
	businessUnitId?: string;
	tenantId?: string;
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

	const token = opts?.token || (typeof localStorage !== 'undefined' ? localStorage.getItem('saas_token') : '');
	if (token) {
		headers['Authorization'] = `Bearer ${token}`;
	}

	const buId = opts?.businessUnitId || (typeof localStorage !== 'undefined' ? localStorage.getItem('mrbiz_active_unit') : '');
	if (buId) {
		headers['X-Business-Unit-Id'] = buId;
	}

	const tenantId = opts?.tenantId || (typeof localStorage !== 'undefined' ? localStorage.getItem('saas_tenant_id') : '');
	if (tenantId) {
		headers['X-Tenant-Id'] = tenantId;
	}

	const res = await fetch(url, {
		method: 'POST',
		headers,
		body: JSON.stringify(req || {}),
	});

	if (!res.ok) {
		const errJson = await res.json().catch(() => ({}));
		throw new Error(errJson.message || `RPC Error (${res.status}): ${res.statusText}`);
	}

	return (await res.json()) as TRes;
}

// 1. Healthcare Service Client
export const healthcareGrpc = {
	listDoctors: (params = {}) => callRpc('healthcare.v1.HealthcareService', 'ListDoctors', params),
	listAppointments: (params = {}) => callRpc('healthcare.v1.HealthcareService', 'ListAppointments', params),
	bookAppointment: (req: any) => callRpc('healthcare.v1.HealthcareService', 'BookAppointment', req),
	createPrescription: (req: any) => callRpc('healthcare.v1.HealthcareService', 'CreatePrescription', req),
};

// 2. Travel & Visa Service Client
export const travelGrpc = {
	listTourPackages: (params = {}) => callRpc('travel.v1.TravelService', 'ListTourPackages', params),
	listVisaApplications: (params = {}) => callRpc('travel.v1.TravelService', 'ListVisaApplications', params),
	submitVisaApplication: (req: any) => callRpc('travel.v1.TravelService', 'SubmitVisaApplication', req),
};

// 3. Real Estate Service Client
export const realestateGrpc = {
	listProperties: (params = {}) => callRpc('realestate.v1.RealEstateService', 'ListProperties', params),
	scheduleSiteVisit: (req: any) => callRpc('realestate.v1.RealEstateService', 'ScheduleSiteVisit', req),
	listSiteVisits: (params = {}) => callRpc('realestate.v1.RealEstateService', 'ListSiteVisits', params),
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
