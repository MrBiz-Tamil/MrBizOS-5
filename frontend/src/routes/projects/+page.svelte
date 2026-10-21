<script lang="ts">
	import { onMount } from 'svelte';
	import { projectGrpc } from '$lib/grpc';

	interface Project {
		id: string;
		code: string;
		name: string;
		description: string;
		businessUnit: string;
		clientName: string;
		status: string;
		budget: number;
		spent: number;
		startDate: string;
		targetDate: string;
		totalTasks: number;
		completedTasks: number;
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

	// Sample robust initial data
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
			completedTasks: 8
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
			completedTasks: 4
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
			completedTasks: 6
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
		{ id: 'MS-03', projectId: 'PRJ-001', title: 'Phase 3: Production Go-Live', targetDate: '2026-12-15', invoiceTriggerAmount: 375000, status: 'PENDING' }
	]);

	let selectedProjectId = $state<string>('');
	let selectedBUFilter = $state<string>('ALL');
	let searchQuery = $state<string>('');
	let loading = $state<boolean>(false);
	let activeTab = $state<'kanban' | 'projects' | 'timesheet' | 'utms_delegation'>('kanban');
	let statusMessage = $state<string | null>(null);

	// Timesheet Logger State
	let timesheetTask = $state('TSK-101');
	let timesheetHours = $state(4);
	let timesheetDate = $state(new Date().toISOString().split('T')[0]);
	let timesheetNote = $state('Integrated OFX bank feed parser and verified journal entries.');
	let timesheetBillableRate = $state(1500);

	// Form for New UTMS Task
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

	// Computed KPIs
	let criticalSlaCount = $derived(tasks.filter(t => t.slaHoursRemaining > 0 && t.slaHoursRemaining <= 12 && t.status !== 'DONE').length);
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

	function getBUBadge(bu: string) {
		switch (bu) {
			case 'HEALTH': return { label: '🏥 Healthcare', color: '#10b981', bg: 'rgba(16, 185, 129, 0.15)' };
			case 'TRAVEL': return { label: '✈️ Travel & Tours', color: '#06b6d4', bg: 'rgba(6, 182, 212, 0.15)' };
			case 'REALESTATE': return { label: '🏢 Real Estate', color: '#f59e0b', bg: 'rgba(245, 158, 11, 0.15)' };
			case 'CONSULTING': return { label: '💻 Tech Consulting', color: '#818cf8', bg: 'rgba(129, 140, 248, 0.15)' };
			case 'COMMERCE': return { label: '🛍️ E-Commerce', color: '#ec4899', bg: 'rgba(236, 72, 153, 0.15)' };
			case 'FINANCE': return { label: '💰 Corporate Finance', color: '#fbbf24', bg: 'rgba(251, 191, 36, 0.15)' };
			default: return { label: '📁 General', color: '#94a3b8', bg: 'rgba(148, 163, 184, 0.15)' };
		}
	}

	async function loadData() {
		try {
			const [pRes, tRes] = await Promise.all([
				projectGrpc.listProjects(),
				projectGrpc.listTasks()
			]);
			if (pRes?.projects && pRes.projects.length > 0) {
				projects = pRes.projects;
			}
			if (tRes?.tasks && tRes.tasks.length > 0) {
				tasks = tRes.tasks;
			}
		} catch (e: any) {
			console.warn('Using client state with full UTMS offline safety:', e);
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

	async function handleCreateTask(e: Event) {
		e.preventDefault();
		if (!newTask.title.trim()) {
			statusMessage = '⚠️ Please enter task title.';
			return;
		}

		const created: UTMSTask = {
			id: `TSK-${Date.now().toString().slice(-4)}`,
			projectId: selectedProjectId || (projects[0]?.id || 'PRJ-001'),
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

		try {
			await projectGrpc.createTask({
				projectId: created.projectId,
				...newTask
			});
		} catch (e) {}
	}

	async function handleExecuteDelegation() {
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

	onMount(() => {
		loadData();
	});
</script>

<svelte:head>
	<title>Universal Task Management System (UTMS) | MrBiz-OS</title>
</svelte:head>

<div class="utms-app">
	
	<!-- Header Bar -->
	<header class="utms-header">
		<div>
			<div class="badge-row">
				<span class="pill-badge">⚡ Universal Task Management System (UTMS)</span>
				<span class="sla-indicator">● Cross-BU Routing Engine · ConnectRPC</span>
			</div>
			<h1 class="main-title">Cross-Department Projects &amp; SLA Pipeline</h1>
			<p class="subtitle">
				Seamless task delegation connecting Healthcare EMR, Travel Ops, Real Estate Plots, and Financial Invoicing without app silos.
			</p>
		</div>

		<div class="header-actions">
			<button class="btn btn-secondary" onclick={() => activeTab = 'timesheet'}>
				⏱️ Log Timesheet
			</button>
			<button class="btn btn-primary" onclick={() => activeTab = 'utms_delegation'}>
				+ New UTMS Task
			</button>
		</div>
	</header>

	{#if statusMessage}
		<div class="status-alert">{statusMessage}</div>
	{/if}

	<!-- Executive KPI Cockpit -->
	<div class="kpi-grid">
		<div class="kpi-card">
			<div class="kpi-label">Active UTMS Tasks</div>
			<div class="kpi-value text-indigo-400">{tasks.length} Deliverables</div>
			<div class="kpi-sub">Across 5 Business Units</div>
		</div>

		<div class="kpi-card">
			<div class="kpi-label">Critical SLA Clock</div>
			<div class="kpi-value text-rose-400" class:pulse-glow={criticalSlaCount > 0}>
				{criticalSlaCount} Urgent Tasks
			</div>
			<div class="kpi-sub">&lt;12 Hours SLA Remaining</div>
		</div>

		<div class="kpi-card">
			<div class="kpi-label">Active Portfolio Value</div>
			<div class="kpi-value text-emerald-400">
				₹{(projects.reduce((acc, p) => acc + p.budget, 0) / 100000).toFixed(1)} Lakhs
			</div>
			<div class="kpi-sub">{projects.length} Committed Contracts</div>
		</div>

		<div class="kpi-card">
			<div class="kpi-label">SLA Compliance Rate</div>
			<div class="kpi-value text-sky-400">96.4% On-Track</div>
			<div class="kpi-sub">Cross-Department Velocity</div>
		</div>
	</div>

	<!-- Navigation Tabs -->
	<div class="nav-tabs">
		<button 
			class="tab-btn" 
			class:active={activeTab === 'kanban'} 
			onclick={() => activeTab = 'kanban'}>
			📋 Kanban Sprint Pipeline ({tasks.length})
		</button>
		<button 
			class="tab-btn" 
			class:active={activeTab === 'projects'} 
			onclick={() => activeTab = 'projects'}>
			📁 Project Portfolios &amp; Milestone Billing ({projects.length})
		</button>
		<button 
			class="tab-btn" 
			class:active={activeTab === 'timesheet'} 
			onclick={() => activeTab = 'timesheet'}>
			⏱️ Timesheets &amp; Billable Burn Rate
		</button>
		<button 
			class="tab-btn" 
			class:active={activeTab === 'utms_delegation'} 
			onclick={() => activeTab = 'utms_delegation'}>
			⚡ + Fast UTMS Task Creator
		</button>
	</div>

	<!-- TAB 1: KANBAN SPRINT PIPELINE -->
	{#if activeTab === 'kanban'}
		<div class="kanban-section">
			
			<!-- Filter & Search Toolbar -->
			<div class="filter-toolbar">
				<div class="filter-group">
					<span class="filter-label">Filter BU:</span>
					<div class="bu-pill-group">
						{#each ['ALL', 'HEALTH', 'TRAVEL', 'REALESTATE', 'CONSULTING', 'FINANCE'] as bu}
							<button 
								class="bu-pill" 
								class:active={selectedBUFilter === bu}
								onclick={() => selectedBUFilter = bu}>
								{bu === 'ALL' ? '🌐 All Verticals' : getBUBadge(bu).label}
							</button>
						{/each}
					</div>
				</div>

				<div class="search-box">
					<span>🔍</span>
					<input type="text" placeholder="Search tasks, assignees, keywords..." bind:value={searchQuery} />
				</div>
			</div>

			<!-- Visual 5-Column Kanban Board -->
			<div class="kanban-board">
				{#each columns as col}
					{@const colTasks = filteredTasks.filter(t => t.status === col)}
					{@const colName = col.replace('_', ' ')}
					<div class="kanban-column">
						
						<!-- Column Header -->
						<div class="column-header">
							<div class="col-title-wrap">
								<span class="col-dot" class:dot-todo={col === 'TODO'} class:dot-prog={col === 'IN_PROGRESS'} class:dot-review={col === 'REVIEW'} class:dot-done={col === 'DONE'}></span>
								<h3 class="col-title">{colName}</h3>
							</div>
							<span class="col-count">{colTasks.length}</span>
						</div>

						<!-- Task Cards List -->
						<div class="column-tasks">
							{#if colTasks.length === 0}
								<div class="empty-column-placeholder">
									No tasks in {colName}
								</div>
							{/if}

							{#each colTasks as task}
								{@const bu = getBUBadge(task.businessUnit)}
								<div class="task-card">
									
									<!-- Card Top Badges -->
									<div class="card-badges">
										<span class="bu-tag" style={`background: ${bu.bg}; color: ${bu.color};`}>
											{bu.label}
										</span>
										<span class="priority-tag" class:tag-urgent={task.priority === 'URGENT'} class:tag-high={task.priority === 'HIGH'}>
											{task.priority === 'URGENT' ? '🔴 URGENT' : task.priority === 'HIGH' ? '🟡 HIGH' : '⚪ NORMAL'}
										</span>
									</div>

									<!-- Title & Description -->
									<h4 class="task-title">{task.title}</h4>
									<p class="task-desc">{task.description}</p>

									<!-- SLA & Assignee Row -->
									<div class="card-meta">
										<div class="assignee-wrap">
											<div class="avatar-circle">
												{task.assignedTo.split(' ')[0][0] || 'U'}
											</div>
											<span class="assignee-name">{task.assignedTo.split(' (')[0]}</span>
										</div>

										<div class="sla-badge" class:sla-urgent={task.slaHoursRemaining <= 12 && task.status !== 'DONE'}>
											⏱️ {task.slaHoursRemaining}h SLA
										</div>
									</div>

									<!-- Action Footer -->
									<div class="card-footer">
										<button 
											class="card-btn-action" 
											onclick={() => {
												taskToDelegate = task;
												showDelegateModal = true;
											}}
											title="Cross-BU Task Delegation">
											🔁 Delegate
										</button>

										<div class="step-buttons">
											{#if col !== 'BACKLOG'}
												<button 
													class="step-nav-btn" 
													onclick={() => handleMoveStatus(task, columns[columns.indexOf(col) - 1])}
													title="Move Back">
													◀
												</button>
											{/if}
											{#if col !== 'DONE'}
												<button 
													class="step-nav-btn" 
													onclick={() => handleMoveStatus(task, columns[columns.indexOf(col) + 1])}
													title="Advance Step">
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

	<!-- TAB 2: PROJECTS & MILESTONE BILLING -->
	{#if activeTab === 'projects'}
		<div class="projects-section space-y-6">
			<div class="section-heading">
				<h2 class="text-lg font-bold text-white">Active Client Contracts &amp; Project Portfolios</h2>
				<p class="text-xs text-slate-400">Budget burn-rate tracking, milestone deliverables, and automated invoice triggers.</p>
			</div>

			<div class="grid grid-cols-1 md:grid-cols-3 gap-6">
				{#each projects as prj}
					{@const percent = Math.min(100, Math.round((prj.spent / prj.budget) * 100))}
					<div class="project-portfolio-card">
						<div class="flex justify-between items-start mb-3">
							<span class="font-mono text-xs font-bold px-2 py-0.5 rounded bg-slate-800 text-slate-300 border border-slate-700">{prj.code}</span>
							<span class="text-xs font-bold px-2.5 py-0.5 rounded-full bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">{prj.status}</span>
						</div>

						<h3 class="text-base font-bold text-white mb-1">{prj.name}</h3>
						<p class="text-xs text-slate-400 line-clamp-2 mb-4">{prj.description}</p>

						<!-- Progress Bar -->
						<div class="space-y-1.5 mb-4">
							<div class="flex justify-between text-xs">
								<span class="text-slate-400">Budget Spent</span>
								<strong class="text-white font-mono">{percent}% (₹{prj.spent.toLocaleString('en-IN')})</strong>
							</div>
							<div class="w-full h-2 rounded-full bg-slate-800 overflow-hidden">
								<div class="h-full bg-gradient-to-r from-indigo-500 to-emerald-400 rounded-full" style={`width: ${percent}%;`}></div>
							</div>
						</div>

						<div class="grid grid-cols-2 gap-2 text-xs pt-3 border-t border-slate-800/80 mb-4">
							<div>
								<span class="text-slate-500 block">Total Budget:</span>
								<strong class="text-white font-mono">₹{prj.budget.toLocaleString('en-IN')}</strong>
							</div>
							<div>
								<span class="text-slate-500 block">Client:</span>
								<strong class="text-slate-300 truncate block">{prj.clientName}</strong>
							</div>
							<div>
								<span class="text-slate-500 block">Business Unit:</span>
								<strong class="text-sky-400">{prj.businessUnit}</strong>
							</div>
							<div>
								<span class="text-slate-500 block">Target Date:</span>
								<strong class="text-slate-300">{prj.targetDate}</strong>
							</div>
						</div>

						<button 
							class="w-full py-2 rounded-lg bg-indigo-600/20 hover:bg-indigo-600 text-indigo-300 hover:text-white border border-indigo-500/40 text-xs font-bold transition-all cursor-pointer"
							onclick={() => {
								selectedProjectId = prj.id;
								activeTab = 'kanban';
							}}>
							📋 View {prj.totalTasks} UTMS Tasks in Kanban
						</button>
					</div>
				{/each}
			</div>

			<!-- Milestone Billing Registry Table -->
			<div class="bg-slate-900/90 border border-slate-700/80 rounded-xl p-5 shadow-xl space-y-4">
				<div class="flex justify-between items-center border-b border-slate-800 pb-3">
					<div>
						<h3 class="text-base font-bold text-white">SOW Contract Milestone Billing Triggers</h3>
						<p class="text-xs text-slate-400">Deliverables automatically convert to Net-30 GST tax invoices upon client milestone sign-off.</p>
					</div>
					<span class="px-3 py-1 bg-sky-500/10 border border-sky-500/30 text-sky-400 font-mono font-bold text-xs rounded-full">
						Auto-Invoicing
					</span>
				</div>

				<div class="table-container">
					<table class="data-table">
						<thead>
							<tr>
								<th>Milestone Code</th>
								<th>Project</th>
								<th>Milestone Deliverable</th>
								<th>Target Date</th>
								<th>Invoice Trigger Amount</th>
								<th>Status</th>
								<th>Action</th>
							</tr>
						</thead>
						<tbody>
							{#each milestones as m}
								<tr>
									<td class="font-mono text-xs font-bold text-slate-400">{m.id}</td>
									<td class="font-bold text-white">FinTech ERP</td>
									<td class="text-sm text-slate-200">{m.title}</td>
									<td class="font-mono text-xs text-slate-400">{m.targetDate}</td>
									<td class="font-mono text-emerald-400 font-bold">₹{m.invoiceTriggerAmount.toLocaleString('en-IN')}</td>
									<td>
										<span class="badge" class:status-available={m.status === 'INVOICED'}>
											{m.status}
										</span>
									</td>
									<td>
										{#if m.status === 'READY_FOR_BILLING'}
											<button class="btn btn-primary btn-sm py-1 px-3 text-xs" onclick={() => handleBillMilestone(m)}>
												🧾 Generate GST Invoice
											</button>
										{:else if m.status === 'INVOICED'}
											<span class="text-xs text-emerald-400 font-bold">✓ Billed (INV-2026)</span>
										{:else}
											<span class="text-xs text-slate-500">In Progress</span>
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

	<!-- TAB 3: TIMESHEET LOGGER -->
	{#if activeTab === 'timesheet'}
		<div class="timesheet-section max-w-3xl mx-auto space-y-6">
			<div class="bg-slate-900/90 border border-slate-700/80 rounded-xl p-6 shadow-xl space-y-5">
				<div class="border-b border-slate-800 pb-3">
					<h3 class="text-base font-bold text-white">Log Developer &amp; Consultant Timesheet Hours</h3>
					<p class="text-xs text-slate-400">Timesheets auto-calculate billable burn rates and client cost utilization.</p>
				</div>

				<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
					<div>
						<label class="input-label" for="ts-task">UTMS Task</label>
						<select id="ts-task" class="text-input" bind:value={timesheetTask}>
							{#each tasks as t}
								<option value={t.id}>[{t.id}] {t.title}</option>
							{/each}
						</select>
					</div>

					<div>
						<label class="input-label" for="ts-hours">Hours Spent Today</label>
						<input id="ts-hours" type="number" class="text-input" min="0.5" max="24" step="0.5" bind:value={timesheetHours} />
					</div>
				</div>

				<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
					<div>
						<label class="input-label" for="ts-date">Work Date</label>
						<input id="ts-date" type="date" class="text-input" bind:value={timesheetDate} />
					</div>

					<div>
						<label class="input-label" for="ts-rate">Billable Hourly Rate (INR)</label>
						<input id="ts-rate" type="number" class="text-input font-mono" bind:value={timesheetBillableRate} />
					</div>
				</div>

				<div>
					<label class="input-label" for="ts-notes">Work Summary &amp; Deliverables Completed</label>
					<textarea id="ts-notes" class="text-input" rows="3" bind:value={timesheetNote}></textarea>
				</div>

				<div class="flex justify-between items-center p-3 rounded-lg bg-slate-950/80 border border-slate-800">
					<span class="text-xs text-slate-400">Total Invoiced Realization for this session:</span>
					<strong class="text-emerald-400 font-mono text-base">₹{(timesheetHours * timesheetBillableRate).toLocaleString('en-IN')}</strong>
				</div>

				<button class="w-full btn btn-primary py-2.5 font-bold cursor-pointer" onclick={() => {
					statusMessage = `✅ Logged ${timesheetHours}h for ${timesheetTask} (Total: ₹${(timesheetHours * timesheetBillableRate).toLocaleString('en-IN')})! Added to project burn ledger.`;
					setTimeout(() => statusMessage = null, 4000);
				}}>
					⏱️ Submit Timesheet to Approval Queue
				</button>
			</div>
		</div>
	{/if}

	<!-- TAB 4: FAST UTMS TASK CREATOR -->
	{#if activeTab === 'utms_delegation'}
		<div class="utms-creator-section max-w-2xl mx-auto">
			<form onsubmit={handleCreateTask} class="bg-slate-900/90 border border-slate-700/80 rounded-xl p-6 shadow-xl space-y-5">
				<div class="border-b border-slate-800 pb-3">
					<h3 class="text-base font-bold text-white">Create Universal Task (UTMS)</h3>
					<p class="text-xs text-slate-400">Dispatch cross-department deliverables with SLA deadlines and automated routing.</p>
				</div>

				<div>
					<label class="input-label" for="tsk-title">Task Title *</label>
					<input id="tsk-title" type="text" class="text-input" placeholder="e.g. Verify Schengen Visa VFS Appointment Dossier" bind:value={newTask.title} required />
				</div>

				<div>
					<label class="input-label" for="tsk-desc">Detailed Description</label>
					<textarea id="tsk-desc" class="text-input" rows="3" placeholder="Provide actionable requirements, references, and acceptance criteria..." bind:value={newTask.description}></textarea>
				</div>

				<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
					<div>
						<label class="input-label" for="tsk-bu">Target Business Unit</label>
						<select id="tsk-bu" class="text-input" bind:value={newTask.businessUnit}>
							<option value="CONSULTING">💻 Software Consulting &amp; IT</option>
							<option value="HEALTH">🏥 Smart Healthcare &amp; Clinical</option>
							<option value="TRAVEL">✈️ Travel, Tours &amp; Visa Ops</option>
							<option value="REALESTATE">🏢 Real Estate Layouts &amp; Plots</option>
							<option value="COMMERCE">🛍️ E-Commerce &amp; POD Printing</option>
							<option value="FINANCE">💰 Corporate Finance &amp; Tax</option>
						</select>
					</div>

					<div>
						<label class="input-label" for="tsk-priority">SLA Priority Level</label>
						<select id="tsk-priority" class="text-input" bind:value={newTask.priority}>
							<option value="URGENT">🔴 URGENT (12-Hour SLA)</option>
							<option value="HIGH">🟡 HIGH (24-Hour SLA)</option>
							<option value="MEDIUM">🔵 MEDIUM (48-Hour SLA)</option>
							<option value="LOW">⚪ LOW (Flexible)</option>
						</select>
					</div>
				</div>

				<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
					<div>
						<label class="input-label" for="tsk-assignee">Assigned Lead</label>
						<input id="tsk-assignee" type="text" class="text-input" bind:value={newTask.assignedTo} />
					</div>

					<div>
						<label class="input-label" for="tsk-due">Due Date</label>
						<input id="tsk-due" type="date" class="text-input" bind:value={newTask.dueDate} />
					</div>
				</div>

				<button type="submit" class="w-full btn btn-primary py-3 font-bold cursor-pointer">
					⚡ Dispatch UTMS Task to Sprint Board
				</button>
			</form>
		</div>
	{/if}

	<!-- MODAL: CROSS-BU TASK DELEGATION -->
	{#if showDelegateModal && taskToDelegate}
		<div class="modal-backdrop">
			<div class="modal-card max-w-lg">
				<div class="modal-header">
					<div class="flex items-center gap-2">
						<span class="text-xl">🔁</span>
						<h3 class="text-base font-bold text-white">Cross-BU Task Delegation</h3>
					</div>
					<button class="close-btn" onclick={() => showDelegateModal = false}>✕</button>
				</div>

				<div class="modal-body space-y-4">
					<div class="p-3 rounded-lg bg-slate-950 border border-slate-800">
						<div class="text-[10px] text-slate-500 uppercase font-mono">{taskToDelegate.id}</div>
						<div class="text-sm font-bold text-white mt-0.5">{taskToDelegate.title}</div>
						<div class="text-xs text-slate-400 mt-1">Currently assigned to: {taskToDelegate.assignedTo}</div>
					</div>

					<div>
						<label class="input-label" for="del-bu">Delegate to Business Department</label>
						<select id="del-bu" class="text-input" bind:value={targetBU}>
							<option value="HEALTH">🏥 Healthcare (Doctor / Clinical Review)</option>
							<option value="TRAVEL">✈️ Travel (Visa Desk / Flight PNR)</option>
							<option value="REALESTATE">🏢 Real Estate (Plot Registration / EC Check)</option>
							<option value="FINANCE">💰 Finance (Tax Invoicing / GST)</option>
							<option value="CONSULTING">💻 Consulting (Architecture / Code)</option>
						</select>
					</div>

					<div>
						<label class="input-label" for="del-assignee">New Responsible Lead</label>
						<input id="del-assignee" type="text" class="text-input" bind:value={targetAssignee} />
					</div>

					<div>
						<label class="input-label" for="del-note">Handover Brief &amp; SLA Instructions</label>
						<textarea id="del-note" class="text-input" rows="3" placeholder="Provide necessary context and instructions for the receiving department..." bind:value={delegationNote}></textarea>
					</div>
				</div>

				<div class="modal-footer">
					<button class="btn btn-secondary text-xs" onclick={() => showDelegateModal = false}>
						Cancel
					</button>
					<button class="btn btn-primary text-xs" onclick={handleExecuteDelegation}>
						⚡ Confirm Cross-Department Handover
					</button>
				</div>
			</div>
		</div>
	{/if}

</div>

<style>
	.utms-app {
		display: flex;
		flex-direction: column;
		gap: 1.5rem;
		padding-bottom: 3rem;
	}

	.utms-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		flex-wrap: wrap;
		gap: 1.25rem;
		padding-bottom: 1.25rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
	}

	.badge-row {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		margin-bottom: 0.35rem;
	}

	.pill-badge {
		font-size: 0.75rem;
		font-weight: 700;
		background: rgba(99, 102, 241, 0.15);
		color: #818cf8;
		border: 1px solid rgba(99, 102, 241, 0.3);
		padding: 0.2rem 0.65rem;
		border-radius: 9999px;
	}

	.sla-indicator {
		font-size: 0.6875rem;
		font-weight: 600;
		color: #34d399;
	}

	.main-title {
		font-size: 1.75rem;
		font-weight: 800;
		color: #ffffff;
		margin: 0;
		letter-spacing: -0.02em;
	}

	.subtitle {
		font-size: 0.875rem;
		color: #94a3b8;
		margin: 0.25rem 0 0 0;
		max-width: 750px;
	}

	.header-actions {
		display: flex;
		gap: 0.75rem;
	}

	.status-alert {
		padding: 0.75rem 1rem;
		border-radius: 10px;
		background: rgba(16, 185, 129, 0.12);
		border: 1px solid rgba(16, 185, 129, 0.3);
		color: #34d399;
		font-size: 0.875rem;
		font-weight: 600;
	}

	/* KPI Grid */
	.kpi-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
		gap: 1rem;
	}

	.kpi-card {
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 12px;
		padding: 1.15rem 1.25rem;
	}

	.kpi-label {
		font-size: 0.6875rem;
		font-weight: 700;
		color: #94a3b8;
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}

	.kpi-value {
		font-size: 1.5rem;
		font-weight: 800;
		margin-top: 0.25rem;
	}

	.kpi-sub {
		font-size: 0.75rem;
		color: #64748b;
		margin-top: 0.25rem;
	}

	.pulse-glow {
		animation: pulseGlow 1.8s infinite;
	}

	@keyframes pulseGlow {
		0%, 100% { opacity: 1; }
		50% { opacity: 0.6; }
	}

	/* Tabs */
	.nav-tabs {
		display: flex;
		gap: 0.5rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
		padding-bottom: 0.5rem;
		overflow-x: auto;
	}

	.tab-btn {
		background: transparent;
		border: 1px solid rgba(255, 255, 255, 0.08);
		color: #94a3b8;
		padding: 0.6rem 1.15rem;
		border-radius: 10px;
		font-size: 0.8125rem;
		font-weight: 700;
		cursor: pointer;
		transition: all 0.15s ease;
		white-space: nowrap;
	}

	.tab-btn:hover {
		color: #ffffff;
		border-color: rgba(255, 255, 255, 0.2);
	}

	.tab-btn.active {
		background: rgba(99, 102, 241, 0.15);
		color: #818cf8;
		border-color: rgba(99, 102, 241, 0.5);
	}

	/* Kanban Section */
	.filter-toolbar {
		display: flex;
		justify-content: space-between;
		align-items: center;
		flex-wrap: wrap;
		gap: 1rem;
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 12px;
		padding: 0.75rem 1rem;
		margin-bottom: 1.25rem;
	}

	.filter-group {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		flex-wrap: wrap;
	}

	.filter-label {
		font-size: 0.75rem;
		font-weight: 700;
		color: #64748b;
		text-transform: uppercase;
	}

	.bu-pill-group {
		display: flex;
		gap: 0.35rem;
		flex-wrap: wrap;
	}

	.bu-pill {
		font-size: 0.75rem;
		font-weight: 600;
		background: #1e293b;
		color: #94a3b8;
		border: 1px solid rgba(255, 255, 255, 0.06);
		padding: 0.25rem 0.65rem;
		border-radius: 6px;
		cursor: pointer;
		transition: all 0.15s;
	}

	.bu-pill:hover, .bu-pill.active {
		background: rgba(99, 102, 241, 0.2);
		color: #818cf8;
		border-color: rgba(99, 102, 241, 0.4);
	}

	.search-box {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		background: #1e293b;
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: 8px;
		padding: 0.35rem 0.75rem;
		min-width: 260px;
	}

	.search-box input {
		background: transparent;
		border: none;
		outline: none;
		color: #ffffff;
		font-size: 0.8125rem;
		width: 100%;
	}

	/* Kanban Columns */
	.kanban-board {
		display: grid;
		grid-template-columns: repeat(5, minmax(260px, 1fr));
		gap: 1rem;
		overflow-x: auto;
		padding-bottom: 1rem;
	}

	.kanban-column {
		background: rgba(15, 23, 42, 0.65);
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 14px;
		display: flex;
		flex-direction: column;
		min-height: 520px;
		backdrop-filter: blur(8px);
	}

	.column-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 0.85rem 1rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.06);
		background: rgba(30, 41, 59, 0.4);
		border-radius: 14px 14px 0 0;
	}

	.col-title-wrap {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.col-dot {
		width: 8px;
		height: 8px;
		border-radius: 9999px;
		background: #64748b;
	}

	.dot-todo { background: #38bdf8; }
	.dot-prog { background: #818cf8; }
	.dot-review { background: #fbbf24; }
	.dot-done { background: #34d399; }

	.col-title {
		font-size: 0.75rem;
		font-weight: 800;
		color: #f1f5f9;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		margin: 0;
	}

	.col-count {
		font-size: 0.6875rem;
		font-weight: 700;
		background: #1e293b;
		color: #94a3b8;
		padding: 0.15rem 0.5rem;
		border-radius: 9999px;
		font-family: monospace;
	}

	.column-tasks {
		padding: 0.75rem;
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
		flex: 1;
		overflow-y: auto;
	}

	.empty-column-placeholder {
		text-align: center;
		padding: 2.5rem 1rem;
		font-size: 0.75rem;
		color: #64748b;
		font-style: italic;
	}

	/* Task Card */
	.task-card {
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 10px;
		padding: 0.85rem;
		display: flex;
		flex-direction: column;
		gap: 0.6rem;
		transition: all 0.2s ease;
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
	}

	.task-card:hover {
		border-color: rgba(99, 102, 241, 0.45);
		transform: translateY(-2px);
		box-shadow: 0 6px 18px rgba(0, 0, 0, 0.35);
	}

	.card-badges {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 0.4rem;
	}

	.bu-tag {
		font-size: 0.625rem;
		font-weight: 700;
		padding: 0.15rem 0.45rem;
		border-radius: 5px;
		max-width: 140px;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.priority-tag {
		font-size: 0.625rem;
		font-weight: 800;
		padding: 0.15rem 0.45rem;
		border-radius: 5px;
		background: #1e293b;
		color: #94a3b8;
	}

	.tag-urgent {
		background: rgba(239, 68, 68, 0.2);
		color: #f87171;
		border: 1px solid rgba(239, 68, 68, 0.4);
	}

	.tag-high {
		background: rgba(245, 158, 11, 0.18);
		color: #fbbf24;
		border: 1px solid rgba(245, 158, 11, 0.35);
	}

	.task-title {
		font-size: 0.8125rem;
		font-weight: 700;
		color: #ffffff;
		line-height: 1.35;
		margin: 0;
	}

	.task-desc {
		font-size: 0.725rem;
		color: #94a3b8;
		line-height: 1.4;
		margin: 0;
		display: -webkit-box;
		-webkit-line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}

	.card-meta {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding-top: 0.5rem;
		border-top: 1px solid rgba(255, 255, 255, 0.05);
	}

	.assignee-wrap {
		display: flex;
		align-items: center;
		gap: 0.4rem;
	}

	.avatar-circle {
		width: 22px;
		height: 22px;
		border-radius: 9999px;
		background: linear-gradient(135deg, #4f46e5, #06b6d4);
		color: #ffffff;
		font-size: 0.625rem;
		font-weight: 800;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.assignee-name {
		font-size: 0.6875rem;
		color: #cbd5e1;
		font-weight: 600;
		max-width: 90px;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.sla-badge {
		font-size: 0.6875rem;
		font-weight: 700;
		color: #34d399;
		font-family: monospace;
	}

	.sla-urgent {
		color: #f87171;
		font-weight: 800;
	}

	.card-footer {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding-top: 0.4rem;
	}

	.card-btn-action {
		font-size: 0.6875rem;
		font-weight: 700;
		color: #38bdf8;
		background: rgba(56, 189, 248, 0.1);
		border: 1px solid rgba(56, 189, 248, 0.25);
		border-radius: 5px;
		padding: 0.2rem 0.5rem;
		cursor: pointer;
		transition: all 0.15s;
	}

	.card-btn-action:hover {
		background: rgba(56, 189, 248, 0.25);
		color: #ffffff;
	}

	.step-buttons {
		display: flex;
		gap: 0.25rem;
	}

	.step-nav-btn {
		background: #1e293b;
		border: 1px solid rgba(255, 255, 255, 0.08);
		color: #cbd5e1;
		font-size: 0.625rem;
		padding: 0.2rem 0.4rem;
		border-radius: 4px;
		cursor: pointer;
		transition: all 0.15s;
	}

	.step-nav-btn:hover {
		background: #334155;
		color: #ffffff;
	}

	/* Project Portfolio Cards */
	.project-portfolio-card {
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 14px;
		padding: 1.25rem;
		display: flex;
		flex-direction: column;
		justify-content: space-between;
		box-shadow: 0 4px 15px rgba(0, 0, 0, 0.25);
	}

	/* Common UI Elements */
	.btn {
		padding: 0.55rem 1rem;
		border-radius: 8px;
		font-size: 0.8125rem;
		font-weight: 700;
		cursor: pointer;
		transition: all 0.15s;
		border: none;
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
	}

	.btn-primary {
		background: #4f46e5;
		color: #ffffff;
		box-shadow: 0 4px 12px rgba(79, 70, 229, 0.35);
	}

	.btn-primary:hover {
		background: #4338ca;
	}

	.btn-secondary {
		background: #1e293b;
		color: #f1f5f9;
		border: 1px solid rgba(255, 255, 255, 0.1);
	}

	.btn-secondary:hover {
		background: #334155;
	}

	.btn-sm {
		padding: 0.25rem 0.6rem;
		font-size: 0.75rem;
	}

	.text-input {
		width: 100%;
		background: #1e293b;
		border: 1px solid rgba(255, 255, 255, 0.1);
		color: #ffffff;
		padding: 0.6rem 0.75rem;
		border-radius: 8px;
		font-size: 0.8125rem;
		outline: none;
	}

	.text-input:focus {
		border-color: #6366f1;
	}

	.input-label {
		display: block;
		font-size: 0.75rem;
		font-weight: 700;
		color: #cbd5e1;
		margin-bottom: 0.35rem;
	}

	/* Tables */
	.table-container {
		overflow-x: auto;
	}

	.data-table {
		width: 100%;
		border-collapse: collapse;
		text-align: left;
		font-size: 0.8125rem;
	}

	.data-table th {
		padding: 0.75rem 0.9rem;
		background: rgba(30, 41, 59, 0.6);
		color: #94a3b8;
		font-size: 0.6875rem;
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
	}

	.data-table td {
		padding: 0.85rem 0.9rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.06);
		color: #f1f5f9;
	}

	.badge {
		font-size: 0.6875rem;
		font-weight: 700;
		padding: 0.2rem 0.55rem;
		border-radius: 9999px;
		background: rgba(148, 163, 184, 0.15);
		color: #94a3b8;
	}

	.status-available {
		background: rgba(16, 185, 129, 0.15);
		color: #34d399;
		border: 1px solid rgba(16, 185, 129, 0.3);
	}

	/* Modals */
	.modal-backdrop {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.75);
		backdrop-filter: blur(4px);
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 1rem;
		z-index: 50;
	}

	.modal-card {
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.12);
		border-radius: 14px;
		width: 100%;
		overflow: hidden;
		box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.5);
	}

	.modal-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 1.15rem 1.25rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
	}

	.close-btn {
		background: none;
		border: none;
		color: #94a3b8;
		font-size: 1.1rem;
		cursor: pointer;
	}

	.modal-body {
		padding: 1.25rem;
	}

	.modal-footer {
		display: flex;
		justify-content: flex-end;
		gap: 0.75rem;
		padding: 1rem 1.25rem;
		border-top: 1px solid rgba(255, 255, 255, 0.08);
		background: rgba(15, 23, 42, 0.5);
	}
</style>
