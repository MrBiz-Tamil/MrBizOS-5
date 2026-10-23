<script lang="ts">
	import { onMount } from 'svelte';
	import { projectGrpc } from '$lib/grpc';
	import Badge from '$lib/components/ui/Badge.svelte';
	import LockBanner from '$lib/components/ui/LockBanner.svelte';

	interface Project {
		id: string;
		code: string;
		name: string;
		description: string;
		businessUnit: string;
		clientName: string;
		status: 'ACTIVE' | 'IN_PROGRESS' | 'COMPLETED' | 'LOCKED' | 'ON_HOLD';
		budget: number;
		spent: number;
		startDate: string;
		targetDate: string;
		totalTasks: number;
		completedTasks: number;
		owner: string;
		isLocked?: boolean;
		lockReason?: string;
		lockScope?: string;
		isConfidential?: boolean;
	}

	interface UTMSTask {
		id: string;
		projectId: string;
		title: string;
		description: string;
		businessUnit: string;
		assignedTo: string;
		priority: 'URGENT' | 'HIGH' | 'MEDIUM' | 'LOW';
		status: 'BACKLOG' | 'TODO' | 'IN_PROGRESS' | 'REVIEW' | 'DONE';
		dueDate: string;
		slaHoursRemaining: number;
		entityResource: string;
		entityId: string;
		estimatedHours: number;
		actualHours: number;
	}

	interface Milestone {
		id: string;
		projectId: string;
		title: string;
		targetDate: string;
		invoiceTriggerAmount: number;
		status: 'PENDING' | 'READY_FOR_BILLING' | 'INVOICED';
	}

	interface TimesheetEntry {
		id: string;
		taskId: string;
		taskTitle: string;
		projectId: string;
		projectName: string;
		consultant: string;
		date: string;
		hours: number;
		rate: number;
		billable: boolean;
		status: 'DRAFT' | 'SUBMITTED' | 'APPROVED' | 'REJECTED' | 'LOCKED';
		notes: string;
	}

	// Projects Data with lock & confidentiality states
	let projects = $state<Project[]>([
		{
			id: 'PRJ-001',
			code: 'PRJ-SOW-2026',
			name: 'FinTech Enterprise ERP Rollout',
			description: 'Complete cloud transformation, accounting integration, and automated bank feeds',
			businessUnit: 'CONSULTING',
			clientName: 'Acme International Corp',
			status: 'IN_PROGRESS',
			budget: 1250000,
			spent: 420000,
			startDate: '2026-09-01',
			targetDate: '2026-12-15',
			totalTasks: 14,
			completedTasks: 8,
			owner: 'Arun Kumar',
			isLocked: false
		},
		{
			id: 'PRJ-002',
			code: 'PRJ-MED-SHIELD',
			name: 'Apollo Multi-Specialty Clinic Deployment',
			description: 'EMR onboarding, doctor consultation scheduling, and laboratory HL7 bridge',
			businessUnit: 'HEALTH',
			clientName: 'Apollo Diagnostics Network',
			status: 'IN_PROGRESS',
			budget: 850000,
			spent: 210000,
			startDate: '2026-10-01',
			targetDate: '2026-11-30',
			totalTasks: 9,
			completedTasks: 4,
			owner: 'Dr. Priya Sharma',
			isLocked: false
		},
		{
			id: 'PRJ-003',
			code: 'PRJ-TRAV-BALI',
			name: 'Global DMC Tour Operators Integration',
			description: 'Automated PNR holds, GDS airline webhooks, and VFS visa biometrics tracker',
			businessUnit: 'TRAVEL',
			clientName: 'Skyline Holiday Group Ltd',
			status: 'ACTIVE',
			budget: 620000,
			spent: 180000,
			startDate: '2026-10-10',
			targetDate: '2026-12-31',
			totalTasks: 11,
			completedTasks: 6,
			owner: 'Arunmozhi Varman',
			isLocked: false
		},
		{
			id: 'PRJ-004',
			code: 'PRJ-AUDIT-FY25',
			name: 'Statutory Financial Audit & Year-End Close',
			description: 'Closed period project with locked milestone budgets and immutable timesheets',
			businessUnit: 'FINANCE',
			clientName: 'Internal Audit Board',
			status: 'LOCKED',
			budget: 450000,
			spent: 450000,
			startDate: '2026-01-01',
			targetDate: '2026-03-31',
			totalTasks: 12,
			completedTasks: 12,
			owner: 'Priya Sundaram',
			isLocked: true,
			lockReason: 'Financial year-end books closed & audited by statutory board.',
			lockScope: 'Read-only financial ledgers, zero further timesheet postings permitted.'
		},
		{
			id: 'PRJ-005',
			code: 'PRJ-MNA-STEALTH',
			name: 'Project Sovereign (Restricted M&A Due Diligence)',
			description: 'Confidential strategic merger evaluation and proprietary vendor valuation',
			businessUnit: 'CONSULTING',
			clientName: 'Confidential Client (Scope A)',
			status: 'ACTIVE',
			budget: 2500000,
			spent: 980000,
			startDate: '2026-08-01',
			targetDate: '2026-11-15',
			totalTasks: 8,
			completedTasks: 5,
			owner: 'Executive Administrator',
			isConfidential: true
		}
	]);

	let tasks = $state<UTMSTask[]>([
		{
			id: 'TSK-101',
			projectId: 'PRJ-001',
			title: 'Deploy Automated Bank Reconciliation Stubs',
			description: 'Configure OFX/CSV statement parsers and verify ledger tolerance thresholds',
			businessUnit: 'CONSULTING',
			assignedTo: 'Arun Kumar (Lead Architect)',
			priority: 'HIGH',
			status: 'IN_PROGRESS',
			dueDate: '2026-10-25',
			slaHoursRemaining: 18,
			entityResource: 'project',
			entityId: 'PRJ-001',
			estimatedHours: 16,
			actualHours: 10
		},
		{
			id: 'TSK-102',
			projectId: 'PRJ-001',
			title: 'Client Review of 3-Way Procurement Workflow',
			description: 'Present automated Purchase Order vs GRN vs Vendor Bill reconciliation bench',
			businessUnit: 'FINANCE',
			assignedTo: 'Priya Sundaram (Controller)',
			priority: 'URGENT',
			status: 'TODO',
			dueDate: '2026-10-23',
			slaHoursRemaining: 6,
			entityResource: 'project',
			entityId: 'PRJ-001',
			estimatedHours: 8,
			actualHours: 2
		},
		{
			id: 'TSK-103',
			projectId: 'PRJ-002',
			title: 'Schedule H1 Antibiotic Compliance Audit Pad',
			description: 'Drug regulatory validation for pharmacy strip dispenser and doctor MCI register',
			businessUnit: 'HEALTH',
			assignedTo: 'Dr. Priya Sharma (Medical Dir)',
			priority: 'HIGH',
			status: 'IN_PROGRESS',
			dueDate: '2026-10-26',
			slaHoursRemaining: 34,
			entityResource: 'project',
			entityId: 'PRJ-002',
			estimatedHours: 12,
			actualHours: 7
		},
		{
			id: 'TSK-104',
			projectId: 'PRJ-003',
			title: 'Schengen Visa VFS Dossier Verification Checklist',
			description: 'Verify 6-month passport validity, travel insurance, and hotel DMC vouchers',
			businessUnit: 'TRAVEL',
			assignedTo: 'Arunmozhi Varman (Tour Ops)',
			priority: 'URGENT',
			status: 'IN_PROGRESS',
			dueDate: '2026-10-22',
			slaHoursRemaining: 8,
			entityResource: 'project',
			entityId: 'PRJ-003',
			estimatedHours: 6,
			actualHours: 4
		},
		{
			id: 'TSK-105',
			projectId: 'PRJ-001',
			title: 'Final SOW Milestone Sign-Off & Net-30 Invoicing',
			description: 'Approve sprint deliverables and generate Net-30 GST commercial invoice',
			businessUnit: 'CONSULTING',
			assignedTo: 'Executive Administrator',
			priority: 'MEDIUM',
			status: 'REVIEW',
			dueDate: '2026-10-28',
			slaHoursRemaining: 54,
			entityResource: 'project',
			entityId: 'PRJ-001',
			estimatedHours: 4,
			actualHours: 3
		},
		{
			id: 'TSK-106',
			projectId: 'PRJ-002',
			title: 'Nurse Triage Vitals Sync with Doctor Cabin 1',
			description: 'Ensure automated push of BP, pulse, temp and SpO2 into digital prescription pad',
			businessUnit: 'HEALTH',
			assignedTo: 'Nurse Divya (OP Triage)',
			priority: 'MEDIUM',
			status: 'DONE',
			dueDate: '2026-10-21',
			slaHoursRemaining: 0,
			entityResource: 'project',
			entityId: 'PRJ-002',
			estimatedHours: 8,
			actualHours: 8
		}
	]);

	let milestones = $state<Milestone[]>([
		{ id: 'MS-01', projectId: 'PRJ-001', title: 'Phase 1: Architecture Sign-Off', targetDate: '2026-09-30', invoiceTriggerAmount: 375000, status: 'INVOICED' },
		{ id: 'MS-02', projectId: 'PRJ-001', title: 'Phase 2: Core Ledger & Bank Sync', targetDate: '2026-10-31', invoiceTriggerAmount: 500000, status: 'READY_FOR_BILLING' },
		{ id: 'MS-03', projectId: 'PRJ-001', title: 'Phase 3: Production Go-Live', targetDate: '2026-12-15', invoiceTriggerAmount: 375000, status: 'PENDING' },
		{ id: 'MS-04', projectId: 'PRJ-002', title: 'HL7 Diagnostics Machine Interface', targetDate: '2026-11-15', invoiceTriggerAmount: 250000, status: 'READY_FOR_BILLING' }
	]);

	let timesheetEntries = $state<TimesheetEntry[]>([
		{
			id: 'TS-801',
			taskId: 'TSK-101',
			taskTitle: 'Deploy Automated Bank Reconciliation Stubs',
			projectId: 'PRJ-001',
			projectName: 'FinTech ERP Rollout',
			consultant: 'Arun Kumar',
			date: '2026-10-22',
			hours: 5.5,
			rate: 1500,
			billable: true,
			status: 'APPROVED',
			notes: 'Configured parser stubs and verified matching rule weights against mock bank feeds.'
		},
		{
			id: 'TS-802',
			taskId: 'TSK-102',
			taskTitle: 'Client Review of 3-Way Procurement Workflow',
			projectId: 'PRJ-001',
			projectName: 'FinTech ERP Rollout',
			consultant: 'Priya Sundaram',
			date: '2026-10-22',
			hours: 3.0,
			rate: 1800,
			billable: true,
			status: 'SUBMITTED',
			notes: 'Demonstrated automated PO matching tolerance to financial controller.'
		},
		{
			id: 'TS-803',
			taskId: 'TSK-104',
			taskTitle: 'Schengen Visa VFS Dossier Verification Checklist',
			projectId: 'PRJ-003',
			projectName: 'Global DMC Tour Operators Integration',
			consultant: 'Arunmozhi Varman',
			date: '2026-10-21',
			hours: 4.0,
			rate: 1200,
			billable: true,
			status: 'APPROVED',
			notes: 'Completed biometrics appointment slots verification.'
		},
		{
			id: 'TS-804',
			taskId: 'TSK-103',
			taskTitle: 'Schedule H1 Antibiotic Compliance Audit Pad',
			projectId: 'PRJ-002',
			projectName: 'Apollo Multi-Specialty Clinic Deployment',
			consultant: 'Dr. Priya Sharma',
			date: '2026-10-21',
			hours: 2.5,
			rate: 2200,
			billable: true,
			status: 'DRAFT',
			notes: 'Drafted regulatory formulary mapping table.'
		},
		{
			id: 'TS-805',
			taskId: 'TSK-105',
			taskTitle: 'Statutory Financial Audit 2025 Ledger Freeze',
			projectId: 'PRJ-004',
			projectName: 'Statutory Financial Audit & Year-End Close',
			consultant: 'Executive Administrator',
			date: '2026-03-30',
			hours: 8.0,
			rate: 2500,
			billable: true,
			status: 'LOCKED',
			notes: 'Financial ledger locked and verified against balance sheet filings.'
		}
	]);

	// View Navigation
	let activeTab = $state<'portfolio' | 'kanban' | 'timesheet' | 'milestones' | 'utms_creator'>('portfolio');
	let selectedProjectId = $state<string>('');
	let selectedBUFilter = $state<string>('ALL');
	let searchQuery = $state<string>('');
	let statusMessage = $state<string | null>(null);

	// Timesheet Form State
	let timesheetTask = $state('TSK-101');
	let timesheetHours = $state(4);
	let timesheetDate = $state(new Date().toISOString().split('T')[0]);
	let timesheetNote = $state('Integrated OFX bank feed parser and verified journal entries.');
	let timesheetBillableRate = $state(1500);

	// New Task Form State
	let newTask = $state({
		title: '',
		description: '',
		businessUnit: 'CONSULTING',
		assignedTo: 'Arun Kumar',
		priority: 'HIGH' as 'URGENT' | 'HIGH' | 'MEDIUM' | 'LOW',
		dueDate: '2026-10-30',
		estimatedHours: 8
	});

	// Delegation Modal State
	let showDelegateModal = $state(false);
	let taskToDelegate = $state<UTMSTask | null>(null);
	let targetBU = $state('HEALTH');
	let targetAssignee = $state('Dr. Priya Sharma');
	let delegationNote = $state('');

	const columns: Array<'BACKLOG' | 'TODO' | 'IN_PROGRESS' | 'REVIEW' | 'DONE'> = [
		'BACKLOG',
		'TODO',
		'IN_PROGRESS',
		'REVIEW',
		'DONE'
	];

	// Computed Metrics
	let totalBudget = $derived(projects.reduce((acc, p) => acc + p.budget, 0));
	let totalSpent = $derived(projects.reduce((acc, p) => acc + p.spent, 0));
	let overallProgress = $derived(totalBudget > 0 ? Math.round((totalSpent / totalBudget) * 100) : 0);
	let criticalSlaCount = $derived(tasks.filter(t => t.slaHoursRemaining > 0 && t.slaHoursRemaining <= 12 && t.status !== 'DONE').length);

	let totalTimesheetHours = $derived(timesheetEntries.reduce((acc, e) => acc + e.hours, 0));
	let totalBillableRealization = $derived(timesheetEntries.filter(e => e.billable).reduce((acc, e) => acc + (e.hours * e.rate), 0));

	let filteredTasks = $derived(
		tasks.filter(t => {
			const matchesProject = !selectedProjectId || t.projectId === selectedProjectId;
			const matchesBU = selectedBUFilter === 'ALL' || t.businessUnit === selectedBUFilter;
			const matchesSearch = !searchQuery ||
				t.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
				t.description.toLowerCase().includes(searchQuery.toLowerCase()) ||
				t.assignedTo.toLowerCase().includes(searchQuery.toLowerCase()) ||
				t.id.toLowerCase().includes(searchQuery.toLowerCase());
			return matchesProject && matchesBU && matchesSearch;
		})
	);

	let filteredProjects = $derived(
		projects.filter(p => {
			const matchesBU = selectedBUFilter === 'ALL' || p.businessUnit === selectedBUFilter;
			const matchesSearch = !searchQuery ||
				p.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
				p.code.toLowerCase().includes(searchQuery.toLowerCase()) ||
				p.clientName.toLowerCase().includes(searchQuery.toLowerCase());
			return matchesBU && matchesSearch;
		})
	);

	function getBUBadge(bu: string) {
		switch (bu) {
			case 'HEALTH': return { label: '🏥 Healthcare', accentVar: 'var(--accent-healthcare)' };
			case 'TRAVEL': return { label: '✈️ Travel Ops', accentVar: 'var(--accent-travel)' };
			case 'REALESTATE': return { label: '🏢 Real Estate', accentVar: 'var(--accent-realestate)' };
			case 'CONSULTING': return { label: '💻 Tech Consulting', accentVar: 'var(--accent-project)' };
			case 'COMMERCE': return { label: '🛍️ E-Commerce', accentVar: 'var(--accent-ecommerce)' };
			case 'FINANCE': return { label: '💰 Accounting & Tax', accentVar: 'var(--accent-finance)' };
			default: return { label: '📁 General', accentVar: 'var(--text-muted)' };
		}
	}

	async function loadData() {
		try {
			const [pRes, tRes] = await Promise.all([
				projectGrpc.listProjects(),
				projectGrpc.listTasks()
			]);
			if (pRes?.projects && pRes.projects.length > 0) {
				projects = pRes.projects as Project[];
			}
			if (tRes?.tasks && tRes.tasks.length > 0) {
				tasks = tRes.tasks;
			}
		} catch (e: any) {
			// Preserves responsive offline safety
		}
	}

	async function handleMoveStatus(task: UTMSTask, newStatus: UTMSTask['status']) {
		task.status = newStatus;
		statusMessage = `⚡ Task ${task.id} advanced to ${newStatus.replace('_', ' ')}!`;
		setTimeout(() => statusMessage = null, 3000);
		try {
			await projectGrpc.updateTaskStatus({ taskId: task.id, status: newStatus });
		} catch (e) {}
	}

	function handleCreateTask(e: Event) {
		e.preventDefault();
		if (!newTask.title.trim()) {
			statusMessage = '⚠️ Please enter task title.';
			return;
		}

		const created: UTMSTask = {
			id: `TSK-${Date.now().toString().slice(-4)}`,
			projectId: selectedProjectId || projects[0]?.id || 'PRJ-001',
			title: newTask.title,
			description: newTask.description || 'Deliverable assigned via UTMS Engine',
			businessUnit: newTask.businessUnit,
			assignedTo: newTask.assignedTo || 'Lead Associate',
			priority: newTask.priority,
			status: 'TODO',
			dueDate: newTask.dueDate,
			slaHoursRemaining: 48,
			entityResource: 'project',
			entityId: selectedProjectId || 'PRJ-001',
			estimatedHours: newTask.estimatedHours,
			actualHours: 0
		};

		tasks = [created, ...tasks];
		statusMessage = `✅ Created UTMS Task ${created.id}: ${created.title}`;
		newTask.title = '';
		newTask.description = '';
		activeTab = 'kanban';
		setTimeout(() => statusMessage = null, 4000);
	}

	function handleExecuteDelegation() {
		if (!taskToDelegate) return;
		taskToDelegate.businessUnit = targetBU;
		taskToDelegate.assignedTo = targetAssignee;
		statusMessage = `⚡ Cross-BU Delegated ${taskToDelegate.id} to ${targetBU} (${targetAssignee})!`;
		showDelegateModal = false;
		taskToDelegate = null;
		delegationNote = '';
		setTimeout(() => statusMessage = null, 4000);
	}

	function handleBillMilestone(m: Milestone) {
		m.status = 'INVOICED';
		statusMessage = `🧾 Milestone "${m.title}" converted to Invoice for ₹${m.invoiceTriggerAmount.toLocaleString('en-IN')}! Transferred to /billing.`;
		setTimeout(() => statusMessage = null, 4000);
	}

	function handleUnlockProject(p: Project) {
		p.isLocked = false;
		p.status = 'ACTIVE';
		statusMessage = `🔑 Authorized Unlock granted for project ${p.code} (${p.name}). Audit entry recorded.`;
		setTimeout(() => statusMessage = null, 4000);
	}

	function handleAddTimesheet(e: Event) {
		e.preventDefault();
		const matchTask = tasks.find(t => t.id === timesheetTask);
		const matchProject = projects.find(p => p.id === matchTask?.projectId);

		const entry: TimesheetEntry = {
			id: `TS-${Date.now().toString().slice(-4)}`,
			taskId: timesheetTask,
			taskTitle: matchTask?.title || 'General Engineering',
			projectId: matchTask?.projectId || 'PRJ-001',
			projectName: matchProject?.name || 'Enterprise Project',
			consultant: 'Arun Kumar',
			date: timesheetDate,
			hours: timesheetHours,
			rate: timesheetBillableRate,
			billable: true,
			status: 'SUBMITTED',
			notes: timesheetNote
		};

		timesheetEntries = [entry, ...timesheetEntries];
		statusMessage = `✅ Logged ${timesheetHours}h for ${entry.taskTitle} (₹${(timesheetHours * timesheetBillableRate).toLocaleString('en-IN')})! Submitted for approval.`;
		setTimeout(() => statusMessage = null, 4000);
	}

	function handleApproveTimesheet(entry: TimesheetEntry) {
		entry.status = 'APPROVED';
		statusMessage = `✓ Timesheet ${entry.id} approved for payroll & invoicing!`;
		setTimeout(() => statusMessage = null, 3000);
	}

	function handleRejectTimesheet(entry: TimesheetEntry) {
		entry.status = 'REJECTED';
		statusMessage = `✕ Timesheet ${entry.id} rejected. Correction requested from consultant.`;
		setTimeout(() => statusMessage = null, 3000);
	}

	function handleLockTimesheet(entry: TimesheetEntry) {
		entry.status = 'LOCKED';
		statusMessage = `🔒 Timesheet ${entry.id} permanently locked under closed accounting period.`;
		setTimeout(() => statusMessage = null, 3000);
	}

	onMount(() => {
		loadData();
	});
</script>

<svelte:head>
	<title>Project Management & Timesheets | MRBiz OS</title>
</svelte:head>

<div class="projects-application">
	<!-- Application Header with Accent Indicator -->
	<header class="app-header-strip">
		<div class="header-main">
			<div class="app-accent-tag">
				<span class="app-accent-dot"></span>
				<span class="app-accent-title">MRBiz OS · Project Management & Workforce</span>
			</div>
			<h1 class="page-title">Enterprise Project Portfolio & UTMS Operations</h1>
			<p class="page-subtitle">
				Odoo-inspired business views for project governance, sprint boards, milestone billing, and audited workforce timesheets.
			</p>
		</div>

		<div class="header-actions">
			<button
				type="button"
				class="btn btn-secondary"
				onclick={() => (activeTab = 'timesheet')}
			>
				⏱️ Log Timesheet
			</button>
			<button
				type="button"
				class="btn btn-primary"
				onclick={() => (activeTab = 'utms_creator')}
			>
				+ Create Task
			</button>
		</div>
	</header>

	{#if statusMessage}
		<div class="status-toast animate-fade-in" role="status">
			{statusMessage}
		</div>
	{/if}

	<!-- Executive KPI Summary Cockpit -->
	<section class="metrics-grid" aria-label="Project KPIs">
		<div class="metric-card">
			<div class="metric-title">
				<span>Committed Contracts</span>
				<span class="metric-icon">📁</span>
			</div>
			<div class="metric-value">₹{(totalBudget / 100000).toFixed(1)}L</div>
			<div class="metric-subtitle">{projects.length} Active Portfolio Projects</div>
		</div>

		<div class="metric-card">
			<div class="metric-title">
				<span>Budget Spent Ratio</span>
				<span class="metric-icon">📊</span>
			</div>
			<div class="metric-value">{overallProgress}%</div>
			<div class="metric-subtitle">₹{(totalSpent / 100000).toFixed(1)}L Total Burned</div>
		</div>

		<div class="metric-card">
			<div class="metric-title">
				<span>Critical SLA Deadlines</span>
				<span class="metric-icon">⏱️</span>
			</div>
			<div class="metric-value {criticalSlaCount > 0 ? 'text-warning' : 'text-success'}">
				{criticalSlaCount} Urgent
			</div>
			<div class="metric-subtitle">&lt;12 Hours SLA Remaining</div>
		</div>

		<div class="metric-card">
			<div class="metric-title">
				<span>Workforce Realization</span>
				<span class="metric-icon">💼</span>
			</div>
			<div class="metric-value text-brand">₹{(totalBillableRealization / 1000).toFixed(1)}k</div>
			<div class="metric-subtitle">{totalTimesheetHours}h Logged in Current Cycle</div>
		</div>
	</section>

	<!-- Filter & Search Toolbar -->
	<section class="toolbar-card">
		<div class="toolbar-filters">
			<span class="toolbar-label">Vertical Scope:</span>
			<div class="bu-pills">
				{#each ['ALL', 'CONSULTING', 'HEALTH', 'TRAVEL', 'FINANCE', 'REALESTATE'] as bu}
					<button
						type="button"
						class="bu-pill-btn"
						class:active={selectedBUFilter === bu}
						onclick={() => (selectedBUFilter = bu)}
					>
						{bu === 'ALL' ? '🌐 All Verticals' : getBUBadge(bu).label}
					</button>
				{/each}
			</div>
		</div>

		<div class="toolbar-search">
			<span class="search-glyph">🔍</span>
			<input
				type="text"
				class="toolbar-search-input"
				placeholder="Search projects, tasks, milestones, assignees..."
				bind:value={searchQuery}
			/>
		</div>
	</section>

	<!-- Odoo-Style View Switcher Tabs -->
	<nav class="view-tabs" aria-label="Project Management Views">
		<button
			type="button"
			class="view-tab"
			class:active={activeTab === 'portfolio'}
			onclick={() => (activeTab = 'portfolio')}
		>
			📁 Project Portfolio ({filteredProjects.length})
		</button>
		<button
			type="button"
			class="view-tab"
			class:active={activeTab === 'kanban'}
			onclick={() => (activeTab = 'kanban')}
		>
			📋 Kanban Sprint Board ({filteredTasks.length})
		</button>
		<button
			type="button"
			class="view-tab"
			class:active={activeTab === 'milestones'}
			onclick={() => (activeTab = 'milestones')}
		>
			🎯 Milestone Deliverables &amp; Invoicing ({milestones.length})
		</button>
		<button
			type="button"
			class="view-tab"
			class:active={activeTab === 'timesheet'}
			onclick={() => (activeTab = 'timesheet')}
		>
			⏱️ Timesheets &amp; Workforce Grid ({timesheetEntries.length})
		</button>
		<button
			type="button"
			class="view-tab"
			class:active={activeTab === 'utms_creator'}
			onclick={() => (activeTab = 'utms_creator')}
		>
			⚡ + New Task Dispatch
		</button>
	</nav>

	<!-- VIEW 1: PROJECT PORTFOLIO & WORKSPACE -->
	{#if activeTab === 'portfolio'}
		<div class="portfolio-view animate-fade-in">
			<div class="section-header-row">
				<div>
					<h2>Enterprise Client Contracts &amp; Project Workspaces</h2>
					<p>Track milestone completion, budget burn rates, client account owners, and security lock statuses.</p>
				</div>
			</div>

			<div class="portfolio-grid">
				{#each filteredProjects as prj (prj.id)}
					{@const percent = Math.min(100, Math.round((prj.spent / prj.budget) * 100))}
					{@const bu = getBUBadge(prj.businessUnit)}
					
					<div class="card project-card" class:is-locked={prj.isLocked}>
						<!-- Top Info Badges -->
						<div class="project-card-header">
							<span class="project-code font-mono">{prj.code}</span>
							<div class="card-status-badges">
								{#if prj.isConfidential}
									<Badge variant="warning" label="CONFIDENTIAL" icon="🔒" size="sm" />
								{/if}
								{#if prj.isLocked}
									<Badge variant="locked" label="LOCKED" icon="🔒" size="sm" />
								{:else if prj.status === 'ACTIVE'}
									<Badge variant="success" label="ACTIVE" icon="✓" size="sm" />
								{:else if prj.status === 'IN_PROGRESS'}
									<Badge variant="info" label="IN PROGRESS" icon="●" size="sm" />
								{:else}
									<Badge variant="neutral" label={prj.status} size="sm" />
								{/if}
							</div>
						</div>

						<!-- Title & Client Scope -->
						<div class="project-title-wrap">
							<h3 class="project-title">
								{prj.isConfidential ? 'Project Sovereign (Restricted M&A Due Diligence)' : prj.name}
							</h3>
							<span class="project-client">🏢 {prj.clientName} · Owner: <strong>{prj.owner}</strong></span>
						</div>

						<p class="project-desc">{prj.description}</p>

						<!-- Budget Progress -->
						<div class="project-budget-box">
							<div class="budget-meta">
								<span class="budget-label">Budget Burn</span>
								<strong class="budget-val font-mono">
									{percent}% (₹{(prj.spent).toLocaleString('en-IN')} / ₹{(prj.budget).toLocaleString('en-IN')})
								</strong>
							</div>
							<div class="progress-bar-track">
								<div
									class="progress-bar-fill"
									style={`width: ${percent}%; background: ${percent > 85 ? 'var(--status-warning)' : 'var(--brand-primary)'};`}
								></div>
							</div>
						</div>

						<!-- Timeline & Task Count -->
						<div class="project-meta-footer">
							<span class="meta-item">📅 {prj.startDate} → {prj.targetDate}</span>
							<span class="meta-item">✓ {prj.completedTasks} / {prj.totalTasks} Tasks</span>
						</div>

						<!-- Lock Banner if Protected -->
						{#if prj.isLocked}
							<div class="card-lock-notice">
								<LockBanner
									title={prj.name}
									reason={prj.lockReason}
									scope={prj.lockScope}
									canUnlock={true}
									onunlock={() => handleUnlockProject(prj)}
								/>
							</div>
						{/if}
					</div>
				{/each}
			</div>
		</div>
	{/if}

	<!-- VIEW 2: KANBAN SPRINT BOARD -->
	{#if activeTab === 'kanban'}
		<div class="kanban-view animate-fade-in">
			<div class="kanban-board">
				{#each columns as col}
					{@const colTasks = filteredTasks.filter(t => t.status === col)}
					{@const colName = col.replace('_', ' ')}
					<div class="kanban-column">
						<div class="kanban-column-header">
							<div class="column-title-wrap">
								<span class="col-indicator col-{col.toLowerCase()}"></span>
								<h3 class="column-name">{colName}</h3>
							</div>
							<span class="column-badge font-mono">{colTasks.length}</span>
						</div>

						<div class="kanban-tasks-list">
							{#if colTasks.length === 0}
								<div class="empty-column-msg">No tasks in {colName}</div>
							{/if}

							{#each colTasks as task (task.id)}
								{@const bu = getBUBadge(task.businessUnit)}
								<div class="card task-card">
									<div class="task-card-tags">
										<span class="task-bu-pill" style={`color: ${bu.accentVar};`}>
											{bu.label}
										</span>
										{#if task.priority === 'URGENT'}
											<Badge variant="error" label="URGENT" icon="🔴" size="sm" />
										{:else if task.priority === 'HIGH'}
											<Badge variant="warning" label="HIGH" icon="🟡" size="sm" />
										{:else}
											<Badge variant="neutral" label={task.priority} size="sm" />
										{/if}
									</div>

									<h4 class="task-title">{task.title}</h4>
									<p class="task-desc">{task.description}</p>

									<div class="task-card-meta">
										<div class="assignee-info">
											<div class="avatar-small">{task.assignedTo[0] || 'U'}</div>
											<span class="assignee-text">{task.assignedTo.split(' (')[0]}</span>
										</div>
										<div class="task-sla font-mono" class:sla-urgent={task.slaHoursRemaining <= 12 && task.status !== 'DONE'}>
											⏱️ {task.slaHoursRemaining}h SLA
										</div>
									</div>

									<div class="task-actions-footer">
										<button
											type="button"
											class="btn btn-outline btn-sm"
											onclick={() => {
												taskToDelegate = task;
												showDelegateModal = true;
											}}
											title="Cross-BU Task Handover"
										>
											🔁 Delegate
										</button>

										<div class="stage-step-group">
											{#if col !== 'BACKLOG'}
												<button
													type="button"
													class="btn btn-secondary btn-sm"
													onclick={() => handleMoveStatus(task, columns[columns.indexOf(col) - 1])}
													title="Previous stage"
												>
													◀
												</button>
											{/if}
											{#if col !== 'DONE'}
												<button
													type="button"
													class="btn btn-secondary btn-sm"
													onclick={() => handleMoveStatus(task, columns[columns.indexOf(col) + 1])}
													title="Advance stage"
												>
													▶
												</button>
											{/if}
										</div>
									</div>
								</div>
							{/each}
						</div>
					</div>
				{/each}
			</div>
		</div>
	{/if}

	<!-- VIEW 3: MILESTONE DELIVERABLES & AUTO-INVOICING -->
	{#if activeTab === 'milestones'}
		<div class="milestones-view animate-fade-in">
			<div class="card">
				<div class="card-header-bar">
					<div>
						<h2>Contract Milestones &amp; Automated Invoicing Triggers</h2>
						<p>Deliverable completion locks triggers for Net-30 commercial invoices and revenue recognition.</p>
					</div>
					<Badge variant="info" label="ConnectRPC Auto-Invoicing" icon="⚡" />
				</div>

				<div class="table-container">
					<table class="data-table">
						<thead>
							<tr>
								<th>Code</th>
								<th>Milestone Deliverable</th>
								<th>Target Date</th>
								<th class="text-right">Trigger Amount</th>
								<th>Status</th>
								<th class="text-right">Action</th>
							</tr>
						</thead>
						<tbody>
							{#each milestones as m (m.id)}
								<tr>
									<td class="font-mono text-muted">{m.id}</td>
									<td>
										<strong>{m.title}</strong>
										<span class="subtext">Project ID: {m.projectId}</span>
									</td>
									<td class="font-mono">{m.targetDate}</td>
									<td class="text-right font-mono text-success" style="font-weight: 700;">
										₹{m.invoiceTriggerAmount.toLocaleString('en-IN')}
									</td>
									<td>
										{#if m.status === 'INVOICED'}
											<Badge variant="success" label="INVOICED" icon="✓" size="sm" />
										{:else if m.status === 'READY_FOR_BILLING'}
											<Badge variant="warning" label="READY FOR BILLING" icon="⏳" size="sm" />
										{:else}
											<Badge variant="neutral" label="PENDING DELIVERABLE" icon="●" size="sm" />
										{/if}
									</td>
									<td class="text-right">
										{#if m.status === 'READY_FOR_BILLING'}
											<button
												type="button"
												class="btn btn-primary btn-sm"
												onclick={() => handleBillMilestone(m)}
											>
												🧾 Generate Invoice
											</button>
										{:else if m.status === 'INVOICED'}
											<span class="text-success font-mono" style="font-size: 0.75rem; font-weight: 600;">
												✓ Invoiced
											</span>
										{:else}
											<span class="text-muted" style="font-size: 0.75rem;">Sprint Ongoing</span>
										{/if}
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</div>
		</div>
	{/if}

	<!-- VIEW 4: TIMESHEETS & WORKFORCE GRID -->
	{#if activeTab === 'timesheet'}
		<div class="timesheet-view animate-fade-in">
			<!-- Timesheet KPI Bar -->
			<div class="timesheet-summary-strip">
				<div class="summary-pill">
					<span class="pill-title">Total Hours Logged</span>
					<strong class="pill-number font-mono">{totalTimesheetHours} hrs</strong>
				</div>
				<div class="summary-pill">
					<span class="pill-title">Billable Realization</span>
					<strong class="pill-number font-mono text-brand">₹{totalBillableRealization.toLocaleString('en-IN')}</strong>
				</div>
				<div class="summary-pill">
					<span class="pill-title">Approved Sessions</span>
					<strong class="pill-number font-mono text-success">
						{timesheetEntries.filter(e => e.status === 'APPROVED').length} Approved
					</strong>
				</div>
			</div>

			<div class="timesheet-layout-grid">
				<!-- Timesheet Entry Form -->
				<div class="card timesheet-form-card">
					<h3>Log Workforce Hours</h3>
					<p class="form-desc">Record consultant effort against universal tasks and billable rates.</p>

					<form onsubmit={handleAddTimesheet} class="form-body">
						<div class="form-group">
							<label class="form-label" for="ts-task-select">
								Universal Task <span class="form-required">*</span>
							</label>
							<select id="ts-task-select" class="form-select" bind:value={timesheetTask}>
								{#each tasks as t}
									<option value={t.id}>[{t.id}] {t.title}</option>
								{/each}
							</select>
						</div>

						<div class="form-row-2">
							<div class="form-group">
								<label class="form-label" for="ts-hours-input">Hours Worked</label>
								<input
									id="ts-hours-input"
									type="number"
									class="form-input font-mono"
									min="0.5"
									max="24"
									step="0.5"
									bind:value={timesheetHours}
									required
								/>
							</div>

							<div class="form-group">
								<label class="form-label" for="ts-date-input">Work Date</label>
								<input
									id="ts-date-input"
									type="date"
									class="form-input"
									bind:value={timesheetDate}
									required
								/>
							</div>
						</div>

						<div class="form-group">
							<label class="form-label" for="ts-rate-input">Billable Hourly Rate (INR)</label>
							<input
								id="ts-rate-input"
								type="number"
								class="form-input font-mono"
								bind:value={timesheetBillableRate}
								required
							/>
						</div>

						<div class="form-group">
							<label class="form-label" for="ts-note-input">Deliverable Summary &amp; Notes</label>
							<textarea
								id="ts-note-input"
								class="form-textarea"
								rows="3"
								bind:value={timesheetNote}
								required
							></textarea>
						</div>

						<div class="realization-box">
							<span class="realization-label">Session Realization:</span>
							<strong class="realization-val font-mono text-brand">
								₹{(timesheetHours * timesheetBillableRate).toLocaleString('en-IN')}
							</strong>
						</div>

						<button type="submit" class="btn btn-primary" style="width: 100%;">
							⏱️ Submit Timesheet to Approval Queue
						</button>
					</form>
				</div>

				<!-- Timesheet Entries Ledger Table -->
				<div class="card timesheet-table-card">
					<div class="table-header-row">
						<h3>Workforce Timesheet Ledger</h3>
						<span class="text-caption text-muted">Audited billing logs</span>
					</div>

					<div class="table-container">
						<table class="data-table">
							<thead>
								<tr>
									<th>ID</th>
									<th>Task &amp; Project</th>
									<th>Consultant</th>
									<th>Date</th>
									<th class="text-right">Hours</th>
									<th class="text-right">Realization</th>
									<th>Status</th>
									<th class="text-right">Approval Actions</th>
								</tr>
							</thead>
							<tbody>
								{#each timesheetEntries as entry (entry.id)}
									<tr>
										<td class="font-mono text-muted">{entry.id}</td>
										<td>
											<strong>{entry.taskTitle}</strong>
											<span class="subtext">{entry.projectName}</span>
										</td>
										<td>{entry.consultant}</td>
										<td class="font-mono">{entry.date}</td>
										<td class="text-right font-mono">{entry.hours}h</td>
										<td class="text-right font-mono" style="font-weight: 700;">
											₹{(entry.hours * entry.rate).toLocaleString('en-IN')}
										</td>
										<td>
											{#if entry.status === 'APPROVED'}
												<Badge variant="success" label="APPROVED" icon="✓" size="sm" />
											{:else if entry.status === 'SUBMITTED'}
												<Badge variant="warning" label="SUBMITTED" icon="⏳" size="sm" />
											{:else if entry.status === 'REJECTED'}
												<Badge variant="error" label="REJECTED" icon="✕" size="sm" />
											{:else if entry.status === 'LOCKED'}
												<Badge variant="locked" label="LOCKED" icon="🔒" size="sm" />
											{:else}
												<Badge variant="neutral" label="DRAFT" icon="●" size="sm" />
											{/if}
										</td>
										<td class="text-right">
											<div class="table-action-btns">
												{#if entry.status === 'SUBMITTED'}
													<button
														type="button"
														class="btn btn-sm btn-outline text-success"
														onclick={() => handleApproveTimesheet(entry)}
														title="Approve timesheet"
													>
														✓
													</button>
													<button
														type="button"
														class="btn btn-sm btn-outline text-error"
														onclick={() => handleRejectTimesheet(entry)}
														title="Reject and request correction"
													>
														✕
													</button>
												{:else if entry.status === 'APPROVED'}
													<button
														type="button"
														class="btn btn-sm btn-outline text-warning"
														onclick={() => handleLockTimesheet(entry)}
														title="Lock period"
													>
														🔒 Lock
													</button>
												{:else if entry.status === 'REJECTED'}
													<button
														type="button"
														class="btn btn-sm btn-secondary"
														onclick={() => (entry.status = 'SUBMITTED')}
														title="Re-submit correction"
													>
														↻ Resubmit
													</button>
												{:else if entry.status === 'LOCKED'}
													<span class="text-muted" style="font-size: 0.75rem;">Audited &amp; Closed</span>
												{/if}
											</div>
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				</div>
			</div>
		</div>
	{/if}

	<!-- VIEW 5: FAST UTMS TASK CREATOR -->
	{#if activeTab === 'utms_creator'}
		<div class="creator-view animate-fade-in">
			<div class="card creator-card">
				<div class="card-header-bar">
					<div>
						<h2>Dispatch Universal Task (UTMS)</h2>
						<p>Route cross-department deliverables with SLA deadlines and automatic stakeholder assignment.</p>
					</div>
					<Badge variant="info" label="Real-time Dispatch" icon="⚡" />
				</div>

				<form onsubmit={handleCreateTask} class="form-body">
					<div class="form-group">
						<label class="form-label" for="create-task-title">
							Task Title <span class="form-required">*</span>
						</label>
						<input
							id="create-task-title"
							type="text"
							class="form-input"
							placeholder="e.g. Verify Schengen Visa VFS Appointment Dossier"
							bind:value={newTask.title}
							required
						/>
					</div>

					<div class="form-group">
						<label class="form-label" for="create-task-desc">Actionable Deliverable Scope</label>
						<textarea
							id="create-task-desc"
							class="form-textarea"
							rows="3"
							placeholder="Provide detailed requirements, client SLA expectations, and verification references..."
							bind:value={newTask.description}
						></textarea>
					</div>

					<div class="form-row-2">
						<div class="form-group">
							<label class="form-label" for="create-task-bu">Operating Business Unit</label>
							<select id="create-task-bu" class="form-select" bind:value={newTask.businessUnit}>
								<option value="CONSULTING">💻 Software Consulting &amp; IT</option>
								<option value="HEALTH">🏥 Smart Healthcare &amp; Clinical</option>
								<option value="TRAVEL">✈️ Travel, Tours &amp; Visa Ops</option>
								<option value="REALESTATE">🏢 Real Estate Layouts &amp; Plots</option>
								<option value="COMMERCE">🛍️ E-Commerce &amp; POD Printing</option>
								<option value="FINANCE">💰 Corporate Finance &amp; Tax</option>
							</select>
						</div>

						<div class="form-group">
							<label class="form-label" for="create-task-priority">SLA Priority Tier</label>
							<select id="create-task-priority" class="form-select" bind:value={newTask.priority}>
								<option value="URGENT">🔴 URGENT (12-Hour SLA Clock)</option>
								<option value="HIGH">🟡 HIGH (24-Hour SLA Clock)</option>
								<option value="MEDIUM">🔵 MEDIUM (48-Hour SLA Clock)</option>
								<option value="LOW">⚪ LOW (Flexible Sprint Scope)</option>
							</select>
						</div>
					</div>

					<div class="form-row-2">
						<div class="form-group">
							<label class="form-label" for="create-task-lead">Assigned Lead Consultant</label>
							<input
								id="create-task-lead"
								type="text"
								class="form-input"
								bind:value={newTask.assignedTo}
								required
							/>
						</div>

						<div class="form-group">
							<label class="form-label" for="create-task-due">Due Date</label>
							<input
								id="create-task-due"
								type="date"
								class="form-input"
								bind:value={newTask.dueDate}
								required
							/>
						</div>
					</div>

					<button type="submit" class="btn btn-primary btn-lg" style="width: 100%; margin-top: 0.5rem;">
						⚡ Dispatch UTMS Task to Active Kanban Sprint
					</button>
				</form>
			</div>
		</div>
	{/if}

	<!-- MODAL: CROSS-BU TASK DELEGATION -->
	{#if showDelegateModal && taskToDelegate}
		<div class="modal-backdrop animate-fade-in" role="dialog" aria-modal="true">
			<div class="modal-card">
				<div class="modal-header">
					<div class="modal-title-wrap">
						<span style="font-size: 1.25rem;">🔁</span>
						<h3>Cross-BU Task Delegation</h3>
					</div>
					<button
						type="button"
						class="btn btn-secondary btn-sm"
						onclick={() => (showDelegateModal = false)}
					>
						✕
					</button>
				</div>

				<div class="modal-body">
					<div class="delegation-task-box">
						<span class="font-mono text-caption text-muted">{taskToDelegate.id}</span>
						<h4 style="margin: 0.2rem 0; font-size: 0.9375rem;">{taskToDelegate.title}</h4>
						<span class="text-caption text-secondary">
							Current Assignee: <strong>{taskToDelegate.assignedTo}</strong>
						</span>
					</div>

					<div class="form-group">
						<label class="form-label" for="modal-target-bu">Transfer to Department</label>
						<select id="modal-target-bu" class="form-select" bind:value={targetBU}>
							<option value="HEALTH">🏥 Healthcare (Doctor / Clinical Review)</option>
							<option value="TRAVEL">✈️ Travel (Visa Desk / Flight PNR)</option>
							<option value="REALESTATE">🏢 Real Estate (Plot Registration / EC Check)</option>
							<option value="FINANCE">💰 Finance (Tax Invoicing / GST)</option>
							<option value="CONSULTING">💻 Consulting (Architecture / Code)</option>
						</select>
					</div>

					<div class="form-group">
						<label class="form-label" for="modal-target-assignee">New Responsible Lead</label>
						<input
							id="modal-target-assignee"
							type="text"
							class="form-input"
							bind:value={targetAssignee}
						/>
					</div>

					<div class="form-group">
						<label class="form-label" for="modal-handover-notes">Handover Context &amp; SLA Brief</label>
						<textarea
							id="modal-handover-notes"
							class="form-textarea"
							rows="3"
							placeholder="Provide necessary background and acceptance requirements for receiving department..."
							bind:value={delegationNote}
						></textarea>
					</div>
				</div>

				<div class="modal-footer">
					<button
						type="button"
						class="btn btn-secondary"
						onclick={() => (showDelegateModal = false)}
					>
						Cancel
					</button>
					<button
						type="button"
						class="btn btn-primary"
						onclick={handleExecuteDelegation}
					>
						⚡ Confirm Cross-Department Transfer
					</button>
				</div>
			</div>
		</div>
	{/if}
</div>

<style>
	.projects-application {
		display: flex;
		flex-direction: column;
		gap: 1.5rem;
		padding-bottom: 3rem;
	}

	/* Header Strip */
	.app-header-strip {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		flex-wrap: wrap;
		gap: 1.25rem;
		padding-bottom: 1.25rem;
		border-bottom: 1px solid var(--border-default);
	}

	.header-main {
		display: flex;
		flex-direction: column;
		gap: 0.35rem;
	}

	.app-accent-tag {
		display: inline-flex;
		align-items: center;
		gap: 0.45rem;
		font-size: var(--text-caption);
		font-weight: 700;
		color: var(--accent-project);
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}

	.app-accent-dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		background: var(--accent-project);
	}

	.page-title {
		font-size: var(--text-page-title);
		font-weight: 700;
		color: var(--text-primary);
		line-height: 1.2;
	}

	.page-subtitle {
		font-size: var(--text-body);
		color: var(--text-secondary);
		max-width: 820px;
		margin: 0;
	}

	.header-actions {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}

	.status-toast {
		padding: 0.75rem 1rem;
		border-radius: var(--radius-md);
		background: var(--status-success-bg);
		border: 1px solid var(--status-success-border);
		color: var(--status-success);
		font-size: var(--text-table);
		font-weight: 600;
	}

	.metric-icon {
		font-size: 1.1rem;
	}

	/* Toolbar */
	.toolbar-card {
		display: flex;
		align-items: center;
		justify-content: space-between;
		flex-wrap: wrap;
		gap: 1rem;
		background: var(--surface-main);
		border: 1px solid var(--border-default);
		border-radius: var(--radius-lg);
		padding: 0.75rem 1rem;
		box-shadow: var(--shadow-xs);
	}

	.toolbar-filters {
		display: flex;
		align-items: center;
		gap: 0.65rem;
		flex-wrap: wrap;
	}

	.toolbar-label {
		font-size: var(--text-caption);
		font-weight: 600;
		color: var(--text-secondary);
	}

	.bu-pills {
		display: flex;
		align-items: center;
		gap: 0.35rem;
		flex-wrap: wrap;
	}

	.bu-pill-btn {
		background: var(--surface-secondary);
		border: 1px solid var(--border-default);
		border-radius: var(--radius-full);
		padding: 0.25rem 0.65rem;
		font-size: var(--text-caption);
		font-weight: 500;
		color: var(--text-secondary);
		cursor: pointer;
		transition: all var(--transition-fast);
	}

	.bu-pill-btn:hover {
		background: var(--border-default);
		color: var(--text-primary);
	}

	.bu-pill-btn.active {
		background: var(--brand-primary-soft);
		border-color: var(--brand-primary);
		color: var(--brand-primary);
		font-weight: 600;
	}

	.toolbar-search {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		background: var(--surface-secondary);
		border: 1px solid var(--border-default);
		border-radius: var(--radius-md);
		padding: 0.4rem 0.75rem;
		flex: 1;
		max-width: 380px;
	}

	.search-glyph {
		font-size: 0.85rem;
		opacity: 0.7;
	}

	.toolbar-search-input {
		background: transparent;
		border: none;
		outline: none;
		color: var(--text-primary);
		font-size: var(--text-table);
		width: 100%;
	}

	/* Portfolio Grid */
	.portfolio-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(360px, 1fr));
		gap: 1.25rem;
		margin-top: 1rem;
	}

	.project-card {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	.project-card-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.project-code {
		font-size: 0.75rem;
		font-weight: 700;
		padding: 0.15rem 0.45rem;
		background: var(--surface-secondary);
		border: 1px solid var(--border-default);
		border-radius: var(--radius-xs);
		color: var(--text-secondary);
	}

	.card-status-badges {
		display: flex;
		align-items: center;
		gap: 0.35rem;
	}

	.project-title-wrap {
		display: flex;
		flex-direction: column;
		gap: 0.2rem;
	}

	.project-title {
		font-size: var(--text-card-title);
		font-weight: 700;
		color: var(--text-primary);
		line-height: 1.3;
	}

	.project-client {
		font-size: var(--text-caption);
		color: var(--text-secondary);
	}

	.project-desc {
		font-size: var(--text-table);
		color: var(--text-secondary);
		line-height: 1.4;
		margin: 0;
		display: -webkit-box;
		-webkit-line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}

	.project-budget-box {
		display: flex;
		flex-direction: column;
		gap: 0.35rem;
		margin-top: auto;
		padding-top: 0.5rem;
	}

	.budget-meta {
		display: flex;
		justify-content: space-between;
		font-size: var(--text-caption);
	}

	.budget-label {
		color: var(--text-secondary);
	}

	.progress-bar-track {
		height: 6px;
		border-radius: var(--radius-full);
		background: var(--surface-secondary);
		overflow: hidden;
	}

	.progress-bar-fill {
		height: 100%;
		border-radius: var(--radius-full);
		transition: width 0.3s ease;
	}

	.project-meta-footer {
		display: flex;
		justify-content: space-between;
		padding-top: 0.5rem;
		border-top: 1px solid var(--border-default);
		font-size: var(--text-caption);
		color: var(--text-muted);
	}

	.card-lock-notice {
		margin-top: 0.5rem;
	}

	/* Kanban Sprint Board */
	.kanban-board {
		display: grid;
		grid-template-columns: repeat(5, minmax(260px, 1fr));
		gap: 1rem;
		overflow-x: auto;
		padding-bottom: 1rem;
	}

	.kanban-column {
		background: var(--surface-secondary);
		border: 1px solid var(--border-default);
		border-radius: var(--radius-lg);
		padding: 0.85rem;
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
		min-height: 520px;
	}

	.kanban-column-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding-bottom: 0.5rem;
		border-bottom: 1px solid var(--border-default);
	}

	.column-title-wrap {
		display: flex;
		align-items: center;
		gap: 0.45rem;
	}

	.col-indicator {
		width: 8px;
		height: 8px;
		border-radius: 50%;
	}
	.col-backlog { background: var(--status-neutral); }
	.col-todo { background: var(--status-warning); }
	.col-in_progress { background: var(--brand-primary); }
	.col-review { background: var(--accent-ai); }
	.col-done { background: var(--status-success); }

	.column-name {
		font-size: var(--text-caption);
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		color: var(--text-primary);
	}

	.column-badge {
		font-size: 0.75rem;
		font-weight: 700;
		color: var(--text-secondary);
		background: var(--surface-main);
		padding: 0.1rem 0.45rem;
		border-radius: var(--radius-xs);
		border: 1px solid var(--border-default);
	}

	.kanban-tasks-list {
		display: flex;
		flex-direction: column;
		gap: 0.65rem;
		flex: 1;
	}

	.empty-column-msg {
		font-size: var(--text-caption);
		color: var(--text-muted);
		text-align: center;
		padding: 2rem 0.5rem;
		border: 1px dashed var(--border-default);
		border-radius: var(--radius-md);
	}

	.task-card {
		padding: 0.85rem;
		display: flex;
		flex-direction: column;
		gap: 0.45rem;
	}

	.task-card-tags {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.task-bu-pill {
		font-size: 0.6875rem;
		font-weight: 700;
	}

	.task-title {
		font-size: 0.84375rem;
		font-weight: 600;
		color: var(--text-primary);
		line-height: 1.3;
	}

	.task-desc {
		font-size: 0.78125rem;
		color: var(--text-secondary);
		line-height: 1.35;
		margin: 0;
		display: -webkit-box;
		-webkit-line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}

	.task-card-meta {
		display: flex;
		align-items: center;
		justify-content: space-between;
		font-size: var(--text-caption);
		padding-top: 0.35rem;
		border-top: 1px solid var(--border-subtle);
	}

	.assignee-info {
		display: flex;
		align-items: center;
		gap: 0.35rem;
	}

	.avatar-small {
		width: 20px;
		height: 20px;
		border-radius: 50%;
		background: var(--brand-primary);
		color: #ffffff;
		font-size: 0.65rem;
		font-weight: 700;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.assignee-text {
		color: var(--text-secondary);
		font-weight: 500;
	}

	.task-sla {
		font-size: 0.72rem;
		color: var(--text-muted);
	}

	.sla-urgent {
		color: var(--status-error);
		font-weight: 700;
	}

	.task-actions-footer {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding-top: 0.35rem;
	}

	.stage-step-group {
		display: flex;
		gap: 0.25rem;
	}

	/* Timesheet View */
	.timesheet-summary-strip {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
		gap: 1rem;
		margin-bottom: 1.25rem;
	}

	.summary-pill {
		background: var(--surface-main);
		border: 1px solid var(--border-default);
		border-radius: var(--radius-lg);
		padding: 1rem;
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
		box-shadow: var(--shadow-xs);
	}

	.pill-title {
		font-size: var(--text-caption);
		font-weight: 600;
		color: var(--text-secondary);
	}

	.pill-number {
		font-size: 1.35rem;
		font-weight: 700;
		color: var(--text-primary);
	}

	.timesheet-layout-grid {
		display: grid;
		grid-template-columns: 340px 1fr;
		gap: 1.25rem;
	}

	@media (max-width: 960px) {
		.timesheet-layout-grid {
			grid-template-columns: 1fr;
		}
	}

	.form-desc {
		font-size: var(--text-caption);
		color: var(--text-muted);
		margin-bottom: 1rem;
	}

	.form-body {
		display: flex;
		flex-direction: column;
		gap: 0.85rem;
	}

	.form-row-2 {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 0.75rem;
	}

	.realization-box {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0.65rem 0.85rem;
		border-radius: var(--radius-md);
		background: var(--surface-secondary);
		border: 1px solid var(--border-default);
	}

	.realization-label {
		font-size: var(--text-caption);
		color: var(--text-secondary);
	}

	.realization-val {
		font-size: 1.05rem;
		font-weight: 700;
	}

	.table-header-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 1rem;
	}

	.subtext {
		display: block;
		font-size: var(--text-caption);
		color: var(--text-muted);
	}

	.table-action-btns {
		display: flex;
		align-items: center;
		justify-content: flex-end;
		gap: 0.35rem;
	}

	.creator-card {
		max-width: 680px;
		margin: 0 auto;
	}

	.card-header-bar {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 1rem;
		margin-bottom: 1.25rem;
		padding-bottom: 0.75rem;
		border-bottom: 1px solid var(--border-default);
	}

	.delegation-task-box {
		background: var(--surface-secondary);
		border: 1px solid var(--border-default);
		border-radius: var(--radius-md);
		padding: 0.75rem;
		margin-bottom: 1rem;
	}

	.modal-title-wrap {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}
</style>
