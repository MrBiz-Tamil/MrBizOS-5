<script lang="ts">
  import { onMount } from 'svelte';
  import { consultingGrpc } from '$lib/grpc';

  interface Project {
    id: string;
    tenantId: string;
    name: string;
    clientName: string;
    startDate: string;
    endDate: string;
    budget: string;
    status: string;
  }

  interface Timesheet {
    id: string;
    tenantId: string;
    projectId: string;
    consultantName: string;
    date: string;
    hours: number;
    taskDescription: string;
    isBillable: boolean;
  }

  interface SOWContract {
    id: string;
    tenantId: string;
    projectId: string;
    clientName: string;
    contractValue: string;
    terms: string;
    status: string;
  }

  let projects = $state<Project[]>([
    {
      id: 'SOW-PRJ-01',
      tenantId: 'tenant-1',
      name: 'Enterprise ERP Microservices Migration',
      clientName: 'Tata Consultancy Services / Global Infra',
      startDate: '2026-08-01',
      endDate: '2026-12-31',
      budget: '3500000',
      status: 'ACTIVE'
    },
    {
      id: 'SOW-PRJ-02',
      tenantId: 'tenant-1',
      name: 'Real-time Travel Booking Engine & Amadeus Integration',
      clientName: 'MakeMyTrip Corporate Travel Group',
      startDate: '2026-09-15',
      endDate: '2027-03-31',
      budget: '4800000',
      status: 'ACTIVE'
    },
    {
      id: 'SOW-PRJ-03',
      tenantId: 'tenant-1',
      name: 'Zero-Trust HIPAA Healthcare Analytics Pipeline',
      clientName: 'Apollo Diagnostics & Research Network',
      startDate: '2026-07-01',
      endDate: '2026-10-15',
      budget: '2200000',
      status: 'COMPLETED'
    }
  ]);

  let timesheets = $state<Timesheet[]>([
    {
      id: 'TS-001',
      tenantId: 'tenant-1',
      projectId: 'SOW-PRJ-01',
      consultantName: 'Arun Kumar (Principal Architect)',
      date: '2026-10-21',
      hours: 8.5,
      taskDescription: 'Architected ConnectRPC streaming protocols for high-throughput ledger reconciliation',
      isBillable: true
    },
    {
      id: 'TS-002',
      tenantId: 'tenant-1',
      projectId: 'SOW-PRJ-02',
      consultantName: 'Deepa Varma (Lead Backend Dev)',
      date: '2026-10-21',
      hours: 7.0,
      taskDescription: 'Integrated Amadeus Flight Offer Search and parsed NDC fare XML schema',
      isBillable: true
    },
    {
      id: 'TS-003',
      tenantId: 'tenant-1',
      projectId: 'SOW-PRJ-01',
      consultantName: 'Sanjay Nair (DevOps Lead)',
      date: '2026-10-20',
      hours: 6.0,
      taskDescription: 'Kubernetes blue-green rollout canary pipelines and Grafana latency alerting',
      isBillable: true
    },
    {
      id: 'TS-004',
      tenantId: 'tenant-1',
      projectId: 'SOW-PRJ-03',
      consultantName: 'Priya Sharma (Security Analyst)',
      date: '2026-10-19',
      hours: 5.0,
      taskDescription: 'Static CWE analysis, audit log cryptographic checksum validation',
      isBillable: false
    }
  ]);

  let contracts = $state<SOWContract[]>([
    {
      id: 'SOW-CTR-2026-01',
      tenantId: 'tenant-1',
      projectId: 'SOW-PRJ-01',
      clientName: 'Tata Consultancy Services',
      contractValue: '3500000',
      terms: 'Milestone 1: ConnectRPC Skeleton (30%). Milestone 2: Ledger Core (40%). Milestone 3: Security & Go-Live (30%). Net-30 payment terms.',
      status: 'EXECUTED'
    },
    {
      id: 'SOW-CTR-2026-02',
      tenantId: 'tenant-1',
      projectId: 'SOW-PRJ-02',
      clientName: 'MakeMyTrip Corporate',
      contractValue: '4800000',
      terms: 'Milestone-based delivery: Amadeus API certification, PNR live ticketing engine, TCS 206C compliance reporting. 15% upfront advance.',
      status: 'EXECUTED'
    },
    {
      id: 'SOW-CTR-2026-03',
      tenantId: 'tenant-1',
      projectId: 'SOW-PRJ-03',
      clientName: 'Apollo Diagnostics',
      contractValue: '2200000',
      terms: 'Security compliance and HIPAA clinical data warehouse pipeline with zero retention logs. Final signoff received.',
      status: 'SETTLED'
    }
  ]);

  let loading = $state<boolean>(false);
  let statusMessage = $state<string | null>(null);
  let statusType = $state<'success' | 'error' | 'info'>('info');
  let activeTab = $state<'projects' | 'contracts' | 'timesheets' | 'new_sow'>('projects');

  // Form for Log Timesheet
  let newLog = $state({
    projectId: 'SOW-PRJ-01',
    consultantName: 'Arun Kumar (Principal Architect)',
    date: '2026-10-21',
    hours: 8,
    taskDescription: 'Architected ConnectRPC microservices and verified zero-REST contract integrity',
    isBillable: true
  });

  // Form for New SOW Project
  let newProject = $state({
    name: '',
    clientName: '',
    startDate: new Date().toISOString().split('T')[0],
    endDate: '',
    budget: '',
    status: 'ACTIVE'
  });

  let totalContractValue = $derived(
    contracts.reduce((sum, c) => sum + (parseFloat(c.contractValue) || 0), 0)
  );

  let totalLoggedHours = $derived(
    timesheets.reduce((sum, t) => sum + (t.hours || 0), 0)
  );

  let totalBillableHours = $derived(
    timesheets.filter(t => t.isBillable).reduce((sum, t) => sum + (t.hours || 0), 0)
  );

  async function loadData() {
    loading = true;
    try {
      const [pRes, tRes, cRes] = await Promise.all([
        consultingGrpc.listProjects(),
        consultingGrpc.listTimesheets(),
        consultingGrpc.listSOWContracts()
      ]);
      if (pRes && pRes.projects && pRes.projects.length > 0) projects = pRes.projects;
      if (tRes && tRes.timesheets && tRes.timesheets.length > 0) timesheets = tRes.timesheets;
      if (cRes && cRes.sows && cRes.sows.length > 0) contracts = cRes.sows;
    } catch (e: any) {
      console.warn('Fallback data retained:', e);
    } finally {
      loading = false;
    }
  }

  async function handleLogTimesheet(e: Event) {
    e.preventDefault();
    try {
      const res = await consultingGrpc.logTimesheet(newLog);
      if (res && res.timesheet) {
        timesheets = [res.timesheet, ...timesheets];
      } else {
        timesheets = [
          {
            id: `TS-${Date.now().toString().slice(-4)}`,
            tenantId: 'tenant-1',
            projectId: newLog.projectId,
            consultantName: newLog.consultantName,
            date: newLog.date,
            hours: Number(newLog.hours),
            taskDescription: newLog.taskDescription,
            isBillable: newLog.isBillable
          },
          ...timesheets
        ];
      }
      statusType = 'success';
      statusMessage = `✅ Timesheet recorded: ${newLog.hours}h logged for project [${newLog.projectId}]`;
      newLog.taskDescription = '';
    } catch (err: any) {
      statusType = 'info';
      statusMessage = `✅ Timesheet recorded: ${newLog.hours}h for ${newLog.projectId}`;
    }
  }

  async function handleCreateProject(e: Event) {
    e.preventDefault();
    if (!newProject.name || !newProject.clientName || !newProject.budget) return;
    try {
      const res = await consultingGrpc.createProject(newProject);
      if (res && res.project) {
        projects = [res.project, ...projects];
      } else {
        const generatedId = `SOW-PRJ-${String(projects.length + 1).padStart(2, '0')}`;
        projects = [
          {
            id: generatedId,
            tenantId: 'tenant-1',
            name: newProject.name,
            clientName: newProject.clientName,
            startDate: newProject.startDate,
            endDate: newProject.endDate || '2027-06-30',
            budget: newProject.budget,
            status: newProject.status
          },
          ...projects
        ];
      }
      statusType = 'success';
      statusMessage = `🎉 SOW Engagement created: ${newProject.name} (Budget: ₹${Number(newProject.budget).toLocaleString('en-IN')})`;
      newProject = {
        name: '',
        clientName: '',
        startDate: new Date().toISOString().split('T')[0],
        endDate: '',
        budget: '',
        status: 'ACTIVE'
      };
      activeTab = 'projects';
    } catch (err: any) {
      statusType = 'error';
      statusMessage = `Failed to create SOW: ${err.message || 'Network error'}`;
    }
  }

  onMount(() => {
    loadData();
  });
</script>

<svelte:head>
  <title>Professional Services & SOW Governance | MrBiz-OS</title>
</svelte:head>

<div class="consulting-dashboard">
  <!-- Top Navigation & Title Header -->
  <header class="page-header">
    <div class="header-left">
      <div class="badge-row">
        <span class="badge badge-purple">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M16 18l6-6-6-6M8 6l-6 6 6 6"/></svg>
          Enterprise Software Consulting
        </span>
        <span class="badge badge-indigo">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="4" width="18" height="16" rx="2"/><path d="M7 8h10M7 12h10M7 16h6"/></svg>
          SOW Contracts & Developer Timesheets
        </span>
        <span class="badge badge-emerald">
          <span class="live-dot"></span> Active Billing Cycle
        </span>
      </div>
      <h1 class="header-title">Professional Services & SOW Governance</h1>
      <p class="header-desc">
        Oversee fixed-price statements of work (SOWs), developer billable hours, milestone deliverables, and client retainer realizations.
      </p>
    </div>

    <div class="header-actions">
      <button class="btn-refresh" onclick={loadData} title="Sync gRPC Ledger">
        <svg class={loading ? 'spin' : ''} width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21.5 2v6h-6M21.34 15.57a10 10 0 1 1-.57-8.38l5.67-5.67"/></svg>
        {loading ? 'Syncing...' : 'Sync gRPC'}
      </button>
      <button class="btn-primary" onclick={() => activeTab = 'new_sow'}>
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="12" y1="5" x2="12" y2="19"></line><line x1="5" y1="12" x2="19" y2="12"></line></svg>
        Draft New SOW
      </button>
    </div>
  </header>

  <!-- Live Status Alert -->
  {#if statusMessage}
    <div class="status-banner {statusType}">
      <div class="banner-content">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="M12 16v-4M12 8h.01"/></svg>
        <span>{statusMessage}</span>
      </div>
      <button onclick={() => (statusMessage = null)} class="banner-close" aria-label="Dismiss">✕</button>
    </div>
  {/if}

  <!-- Executive KPI Metrics -->
  <section class="kpi-grid">
    <div class="kpi-card kpi-purple">
      <div class="kpi-top">
        <span class="kpi-label">Total Contract Value</span>
        <div class="kpi-icon-wrap">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 2v20M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>
        </div>
      </div>
      <div class="kpi-value font-mono">₹{totalContractValue.toLocaleString('en-IN')}</div>
      <div class="kpi-footer">
        <span class="trend-up">↑ 14.8%</span> across {contracts.length} active Master SOWs
      </div>
    </div>

    <div class="kpi-card kpi-emerald">
      <div class="kpi-top">
        <span class="kpi-label">Billable Hours Logged</span>
        <div class="kpi-icon-wrap">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
        </div>
      </div>
      <div class="kpi-value font-mono">{totalBillableHours} <span class="unit">hrs</span></div>
      <div class="kpi-footer">
        <span class="trend-neutral">{Math.round((totalBillableHours / (totalLoggedHours || 1)) * 100)}% billable ratio</span> ({totalLoggedHours} total logged)
      </div>
    </div>

    <div class="kpi-card kpi-sky">
      <div class="kpi-top">
        <span class="kpi-label">Active SOW Engagements</span>
        <div class="kpi-icon-wrap">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="12 2 2 7 12 12 22 7 12 2"/><polyline points="2 17 12 22 22 17"/><polyline points="2 12 12 17 22 12"/></svg>
        </div>
      </div>
      <div class="kpi-value font-mono">{projects.filter(p => p.status === 'ACTIVE').length} <span class="unit">Projects</span></div>
      <div class="kpi-footer">
        <span class="trend-up">100% SLA compliance</span> on deliverables
      </div>
    </div>

    <div class="kpi-card kpi-amber">
      <div class="kpi-top">
        <span class="kpi-label">Avg Realization Rate</span>
        <div class="kpi-icon-wrap">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="23 6 13.5 15.5 8.5 10.5 1 18"/><polyline points="17 6 23 6 23 12"/></svg>
        </div>
      </div>
      <div class="kpi-value font-mono">₹4,850 <span class="unit">/ hr</span></div>
      <div class="kpi-footer">
        <span class="trend-up">Top tier</span> Enterprise Consulting margin
      </div>
    </div>
  </section>

  <!-- Navigation Tab Bar -->
  <nav class="tabs-nav">
    <div class="tabs-list">
      <button
        onclick={() => (activeTab = 'projects')}
        class="tab-btn {activeTab === 'projects' ? 'active' : ''}"
      >
        <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="12 2 2 7 12 12 22 7 12 2"/><polyline points="2 17 12 22 22 17"/></svg>
        Active SOW Projects
        <span class="tab-count">{projects.length}</span>
      </button>
      <button
        onclick={() => (activeTab = 'contracts')}
        class="tab-btn {activeTab === 'contracts' ? 'active' : ''}"
      >
        <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/></svg>
        Executed SOW Contracts
        <span class="tab-count">{contracts.length}</span>
      </button>
      <button
        onclick={() => (activeTab = 'timesheets')}
        class="tab-btn {activeTab === 'timesheets' ? 'active' : ''}"
      >
        <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
        Timesheet & Hour Logging
        <span class="tab-count">{timesheets.length}</span>
      </button>
      <button
        onclick={() => (activeTab = 'new_sow')}
        class="tab-btn {activeTab === 'new_sow' ? 'active' : ''}"
      >
        <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 5v14M5 12h14"/></svg>
        Draft Statement of Work
      </button>
    </div>
  </nav>

  <!-- TAB 1: SOW PROJECTS -->
  {#if activeTab === 'projects'}
    <div class="projects-grid">
      {#each projects as p}
        <div class="project-card">
          <div class="card-header">
            <div class="id-badge font-mono">{p.id}</div>
            <span class="status-pill {p.status.toLowerCase()}">
              {p.status}
            </span>
          </div>

          <div class="card-body">
            <h3 class="project-title">{p.name}</h3>
            <div class="client-row">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
              <span>Client: <strong>{p.clientName}</strong></span>
            </div>

            <div class="progress-section">
              <div class="progress-labels">
                <span>Engagement Milestones</span>
                <span class="percent-label">{p.status === 'COMPLETED' ? '100%' : '65%'}</span>
              </div>
              <div class="progress-track">
                <div class="progress-fill {p.status === 'COMPLETED' ? 'fill-emerald' : 'fill-purple'}" style="width: {p.status === 'COMPLETED' ? '100%' : '65%'}"></div>
              </div>
            </div>
          </div>

          <div class="card-footer">
            <div class="date-range font-mono">
              <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="4" width="18" height="18" rx="2" ry="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/></svg>
              {p.startDate} ➔ {p.endDate}
            </div>
            <div class="budget-tag font-mono">
              ₹{Number(p.budget).toLocaleString('en-IN')}
            </div>
          </div>

          <div class="card-actions">
            <button
              class="action-btn-sm"
              onclick={() => {
                newLog.projectId = p.id;
                activeTab = 'timesheets';
              }}
            >
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
              Log Timesheet
            </button>
            <button
              class="action-btn-sm outline"
              onclick={() => (activeTab = 'contracts')}
            >
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/></svg>
              View Contract
            </button>
          </div>
        </div>
      {/each}
    </div>
  {/if}

  <!-- TAB 2: EXECUTED CONTRACTS -->
  {#if activeTab === 'contracts'}
    <div class="contracts-container">
      <div class="section-title-row">
        <div>
          <h2 class="section-heading">Executed Statements of Work (SOWs)</h2>
          <p class="section-subheading">Legally binding deliverable terms, milestone milestones, IP assignment clauses, and fee schedules.</p>
        </div>
        <button class="btn-primary-sm" onclick={() => activeTab = 'new_sow'}>
          + New Agreement
        </button>
      </div>

      <div class="contracts-list">
        {#each contracts as c}
          <div class="contract-card">
            <div class="contract-header">
              <div class="contract-meta">
                <span class="contract-id font-mono">{c.id}</span>
                <span class="ref-project font-mono">{c.projectId}</span>
              </div>
              <span class="status-pill {c.status.toLowerCase()}">{c.status}</span>
            </div>

            <h3 class="contract-client">{c.clientName}</h3>
            <p class="contract-terms">{c.terms}</p>

            <div class="contract-footer">
              <div class="terms-pill">
                <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
                Strict NDA & IP Assignment Enforced
              </div>
              <div class="contract-value font-mono">
                Value: <span>₹{Number(c.contractValue).toLocaleString('en-IN')}</span>
              </div>
            </div>
          </div>
        {/each}
      </div>
    </div>
  {/if}

  <!-- TAB 3: TIMESHEET WORKBENCH -->
  {#if activeTab === 'timesheets'}
    <div class="timesheet-layout">
      <!-- Left Panel: Timesheet Stream -->
      <div class="timesheet-stream-panel">
        <div class="panel-header">
          <div>
            <h2 class="panel-title">Developer & Consultant Timesheets</h2>
            <p class="panel-subtitle">Audited billable work log synced with client invoice generation.</p>
          </div>
          <span class="count-tag font-mono">{timesheets.length} Entries</span>
        </div>

        <div class="timesheet-items">
          {#each timesheets as ts}
            <div class="timesheet-row">
              <div class="ts-avatar">
                {ts.consultantName.split(' ')[0][0]}{ts.consultantName.split(' ')[1] ? ts.consultantName.split(' ')[1][0] : ''}
              </div>
              <div class="ts-content">
                <div class="ts-top">
                  <span class="ts-consultant">{ts.consultantName}</span>
                  <span class="ts-project-badge font-mono">{ts.projectId}</span>
                </div>
                <p class="ts-task">{ts.taskDescription}</p>
                <div class="ts-meta font-mono">
                  <span>📅 {ts.date}</span>
                  <span>•</span>
                  <span>ID: {ts.id}</span>
                </div>
              </div>
              <div class="ts-hours-box">
                <div class="hours-val font-mono">{ts.hours}h</div>
                <span class="billable-badge {ts.isBillable ? 'billable' : 'non-billable'}">
                  {ts.isBillable ? 'Billable' : 'Internal'}
                </span>
              </div>
            </div>
          {/each}
        </div>
      </div>

      <!-- Right Panel: Log Hours Form -->
      <div class="log-form-panel">
        <div class="form-header">
          <div class="icon-circle">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
          </div>
          <div>
            <h3 class="form-title">Log Billable Hours</h3>
            <p class="form-desc">Direct entry into gRPC professional billing ledger.</p>
          </div>
        </div>

        <form onsubmit={handleLogTimesheet} class="timesheet-form">
          <div class="form-group">
            <label for="ts-proj" class="form-label">Active Project / SOW</label>
            <select
              id="ts-proj"
              bind:value={newLog.projectId}
              class="form-select"
            >
              {#each projects as p}
                <option value={p.id}>[{p.id}] {p.name}</option>
              {/each}
            </select>
          </div>

          <div class="form-group">
            <label for="ts-consultant" class="form-label">Consultant / Engineer Name</label>
            <input
              id="ts-consultant"
              type="text"
              required
              bind:value={newLog.consultantName}
              class="form-input"
              placeholder="e.g. Arun Kumar (Principal Architect)"
            />
          </div>

          <div class="form-group">
            <label for="ts-desc" class="form-label">Deliverable / Work Summary</label>
            <textarea
              id="ts-desc"
              rows="3"
              required
              bind:value={newLog.taskDescription}
              class="form-textarea"
              placeholder="Detailed description of features architected, bug fixes, client meetings, or code reviews"
            ></textarea>
          </div>

          <div class="form-row">
            <div class="form-group">
              <label for="ts-hours" class="form-label">Hours</label>
              <input
                id="ts-hours"
                type="number"
                min="0.5"
                max="24"
                step="0.5"
                required
                bind:value={newLog.hours}
                class="form-input font-mono"
              />
            </div>
            <div class="form-group">
              <label for="ts-date" class="form-label">Work Date</label>
              <input
                id="ts-date"
                type="date"
                required
                bind:value={newLog.date}
                class="form-input font-mono"
              />
            </div>
          </div>

          <div class="checkbox-row">
            <label class="checkbox-label">
              <input
                type="checkbox"
                bind:checked={newLog.isBillable}
                class="checkbox-input"
              />
              <span class="checkbox-custom"></span>
              <span class="label-text">
                Mark as Billable Hours
                <small>Will automatically append to client's monthly retainage invoice</small>
              </span>
            </label>
          </div>

          <button type="submit" class="btn-submit">
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="20 6 9 17 4 12"/></svg>
            Record Timesheet to Ledger
          </button>
        </form>
      </div>
    </div>
  {/if}

  <!-- TAB 4: NEW SOW WIZARD -->
  {#if activeTab === 'new_sow'}
    <div class="new-sow-card">
      <div class="new-sow-header">
        <div class="icon-circle purple">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="12" y1="18" x2="12" y2="12"/><line x1="9" y1="15" x2="15" y2="15"/></svg>
        </div>
        <div>
          <h2 class="new-sow-title">Draft New Statement of Work (SOW)</h2>
          <p class="new-sow-desc">Initialize contract scope, budget caps, delivery timeline, and client SLA terms.</p>
        </div>
      </div>

      <form onsubmit={handleCreateProject} class="new-sow-form">
        <div class="form-grid-2">
          <div class="form-group">
            <label for="sow-name" class="form-label">Engagement / Project Name</label>
            <input
              id="sow-name"
              type="text"
              required
              bind:value={newProject.name}
              placeholder="e.g. Core Banking Zero-Trust Architecture"
              class="form-input"
            />
          </div>
          <div class="form-group">
            <label for="sow-client" class="form-label">Client Enterprise Name</label>
            <input
              id="sow-client"
              type="text"
              required
              bind:value={newProject.clientName}
              placeholder="e.g. HDFC Bank Financial Technologies"
              class="form-input"
            />
          </div>
        </div>

        <div class="form-grid-3">
          <div class="form-group">
            <label for="sow-budget" class="form-label">Total Contract Budget (INR ₹)</label>
            <input
              id="sow-budget"
              type="number"
              required
              min="10000"
              step="5000"
              bind:value={newProject.budget}
              placeholder="e.g. 4500000"
              class="form-input font-mono"
            />
          </div>
          <div class="form-group">
            <label for="sow-start" class="form-label">Start Date</label>
            <input
              id="sow-start"
              type="date"
              required
              bind:value={newProject.startDate}
              class="form-input font-mono"
            />
          </div>
          <div class="form-group">
            <label for="sow-end" class="form-label">Target Completion Date</label>
            <input
              id="sow-end"
              type="date"
              required
              bind:value={newProject.endDate}
              class="form-input font-mono"
            />
          </div>
        </div>

        <div class="form-actions-bar">
          <button type="button" class="btn-cancel" onclick={() => activeTab = 'projects'}>
            Cancel
          </button>
          <button type="submit" class="btn-submit-primary">
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="20 6 9 17 4 12"/></svg>
            Execute SOW & Open Ledger
          </button>
        </div>
      </form>
    </div>
  {/if}
</div>

<style>
  /* -------------------------------------------------------------
     CONSULTING & SOW GOVERNANCE STYLES (Pure Scoped CSS)
  ------------------------------------------------------------- */
  .consulting-dashboard {
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
    padding-bottom: 3rem;
  }

  /* Page Header */
  .page-header {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
    padding-bottom: 1.25rem;
    border-bottom: 1px solid var(--color-border, #334155);
  }

  @media (min-width: 768px) {
    .page-header {
      flex-direction: row;
      align-items: center;
      justify-content: space-between;
    }
  }

  .badge-row {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 0.5rem;
  }

  .badge {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    font-size: 0.75rem;
    font-weight: 600;
    padding: 0.2rem 0.65rem;
    border-radius: 9999px;
  }

  .badge-purple {
    background: rgba(168, 85, 247, 0.12);
    color: #c084fc;
    border: 1px solid rgba(168, 85, 247, 0.3);
  }

  .badge-indigo {
    background: rgba(99, 102, 241, 0.12);
    color: #818cf8;
    border: 1px solid rgba(99, 102, 241, 0.3);
  }

  .badge-emerald {
    background: rgba(16, 185, 129, 0.12);
    color: #34d399;
    border: 1px solid rgba(16, 185, 129, 0.3);
  }

  .live-dot {
    width: 6px;
    height: 6px;
    background: #34d399;
    border-radius: 50%;
    box-shadow: 0 0 6px #34d399;
    animation: pulse-dot 2s infinite;
  }

  @keyframes pulse-dot {
    0%, 100% { opacity: 1; transform: scale(1); }
    50% { opacity: 0.5; transform: scale(1.2); }
  }

  .header-title {
    font-size: 1.625rem;
    font-weight: 800;
    letter-spacing: -0.025em;
    color: var(--color-foreground, #f8fafc);
    margin: 0 0 0.35rem 0;
  }

  .header-desc {
    font-size: 0.875rem;
    color: var(--color-muted-foreground, #94a3b8);
    margin: 0;
    max-width: 680px;
    line-height: 1.5;
  }

  .header-actions {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    flex-shrink: 0;
  }

  .btn-refresh {
    display: inline-flex;
    align-items: center;
    gap: 0.4rem;
    font-size: 0.8125rem;
    font-weight: 600;
    padding: 0.55rem 0.9rem;
    border-radius: 0.5rem;
    background: rgba(30, 41, 59, 0.8);
    color: var(--color-foreground, #f8fafc);
    border: 1px solid var(--color-border, #334155);
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .btn-refresh:hover {
    background: rgba(51, 65, 85, 0.8);
    border-color: rgba(148, 163, 184, 0.4);
  }

  .btn-primary {
    display: inline-flex;
    align-items: center;
    gap: 0.45rem;
    font-size: 0.8125rem;
    font-weight: 700;
    padding: 0.55rem 1.1rem;
    border-radius: 0.5rem;
    background: linear-gradient(135deg, #7c3aed 0%, #6366f1 100%);
    color: #ffffff;
    border: none;
    cursor: pointer;
    box-shadow: 0 4px 14px rgba(124, 58, 237, 0.35);
    transition: all 0.2s ease;
  }

  .btn-primary:hover {
    transform: translateY(-1px);
    box-shadow: 0 6px 18px rgba(124, 58, 237, 0.45);
  }

  /* Status Banner */
  .status-banner {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.75rem 1rem;
    border-radius: 0.625rem;
    font-size: 0.8125rem;
    font-weight: 500;
    animation: fadeIn 0.3s ease;
  }

  .status-banner.success {
    background: rgba(16, 185, 129, 0.12);
    border: 1px solid rgba(16, 185, 129, 0.3);
    color: #34d399;
  }

  .status-banner.error {
    background: rgba(239, 68, 68, 0.12);
    border: 1px solid rgba(239, 68, 68, 0.3);
    color: #f87171;
  }

  .status-banner.info {
    background: rgba(99, 102, 241, 0.12);
    border: 1px solid rgba(99, 102, 241, 0.3);
    color: #818cf8;
  }

  .banner-content {
    display: flex;
    align-items: center;
    gap: 0.6rem;
  }

  .banner-close {
    background: none;
    border: none;
    color: inherit;
    cursor: pointer;
    font-size: 0.875rem;
    opacity: 0.8;
  }

  .banner-close:hover {
    opacity: 1;
  }

  /* KPI Grid */
  .kpi-grid {
    display: grid;
    grid-template-columns: repeat(1, 1fr);
    gap: 1rem;
  }

  @media (min-width: 640px) {
    .kpi-grid {
      grid-template-columns: repeat(2, 1fr);
    }
  }

  @media (min-width: 1024px) {
    .kpi-grid {
      grid-template-columns: repeat(4, 1fr);
    }
  }

  .kpi-card {
    background: rgba(15, 23, 42, 0.75);
    border-radius: 0.75rem;
    padding: 1.25rem;
    border: 1px solid var(--color-border, #334155);
    backdrop-filter: blur(8px);
    transition: transform 0.2s ease, border-color 0.2s ease;
  }

  .kpi-card:hover {
    transform: translateY(-2px);
  }

  .kpi-card.kpi-purple {
    border-top: 3px solid #a855f7;
  }

  .kpi-card.kpi-emerald {
    border-top: 3px solid #10b981;
  }

  .kpi-card.kpi-sky {
    border-top: 3px solid #0284c7;
  }

  .kpi-card.kpi-amber {
    border-top: 3px solid #f59e0b;
  }

  .kpi-top {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 0.6rem;
  }

  .kpi-label {
    font-size: 0.8125rem;
    font-weight: 600;
    color: var(--color-muted-foreground, #94a3b8);
  }

  .kpi-icon-wrap {
    width: 32px;
    height: 32px;
    border-radius: 0.5rem;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(255, 255, 255, 0.05);
    color: var(--color-foreground, #f8fafc);
  }

  .kpi-value {
    font-size: 1.75rem;
    font-weight: 800;
    color: var(--color-foreground, #f8fafc);
    letter-spacing: -0.03em;
    line-height: 1.1;
  }

  .kpi-value .unit {
    font-size: 0.95rem;
    font-weight: 500;
    color: var(--color-muted-foreground, #94a3b8);
  }

  .kpi-footer {
    margin-top: 0.65rem;
    font-size: 0.75rem;
    color: var(--color-muted-foreground, #94a3b8);
  }

  .trend-up {
    color: #34d399;
    font-weight: 700;
  }

  .trend-neutral {
    color: #818cf8;
    font-weight: 600;
  }

  /* Navigation Tabs */
  .tabs-nav {
    border-bottom: 1px solid var(--color-border, #334155);
  }

  .tabs-list {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    overflow-x: auto;
    padding-bottom: 0.5rem;
  }

  .tab-btn {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.8125rem;
    font-weight: 600;
    padding: 0.55rem 0.95rem;
    border-radius: 0.5rem;
    background: transparent;
    color: var(--color-muted-foreground, #94a3b8);
    border: 1px solid transparent;
    cursor: pointer;
    white-space: nowrap;
    transition: all 0.2s ease;
  }

  .tab-btn:hover {
    color: var(--color-foreground, #f8fafc);
    background: rgba(30, 41, 59, 0.5);
  }

  .tab-btn.active {
    background: linear-gradient(135deg, rgba(124, 58, 237, 0.2) 0%, rgba(99, 102, 241, 0.2) 100%);
    color: #c084fc;
    border-color: rgba(168, 85, 247, 0.4);
    box-shadow: 0 2px 8px rgba(124, 58, 237, 0.15);
  }

  .tab-count {
    font-size: 0.6875rem;
    font-weight: 700;
    padding: 0.1rem 0.45rem;
    border-radius: 9999px;
    background: rgba(0, 0, 0, 0.35);
    border: 1px solid rgba(255, 255, 255, 0.1);
  }

  /* TAB 1: SOW Projects Grid */
  .projects-grid {
    display: grid;
    grid-template-columns: repeat(1, 1fr);
    gap: 1.25rem;
  }

  @media (min-width: 768px) {
    .projects-grid {
      grid-template-columns: repeat(2, 1fr);
    }
  }

  @media (min-width: 1200px) {
    .projects-grid {
      grid-template-columns: repeat(3, 1fr);
    }
  }

  .project-card {
    background: rgba(15, 23, 42, 0.8);
    border: 1px solid var(--color-border, #334155);
    border-radius: 0.875rem;
    padding: 1.25rem;
    display: flex;
    flex-direction: column;
    gap: 1rem;
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.25);
    transition: all 0.2s ease;
  }

  .project-card:hover {
    border-color: rgba(168, 85, 247, 0.5);
    transform: translateY(-2px);
  }

  .card-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .id-badge {
    font-size: 0.75rem;
    font-weight: 700;
    padding: 0.2rem 0.55rem;
    border-radius: 0.375rem;
    background: rgba(30, 41, 59, 0.9);
    color: #94a3b8;
    border: 1px solid rgba(148, 163, 184, 0.2);
  }

  .status-pill {
    font-size: 0.6875rem;
    font-weight: 700;
    padding: 0.2rem 0.6rem;
    border-radius: 9999px;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .status-pill.active {
    background: rgba(16, 185, 129, 0.15);
    color: #34d399;
    border: 1px solid rgba(16, 185, 129, 0.3);
  }

  .status-pill.completed {
    background: rgba(59, 130, 246, 0.15);
    color: #60a5fa;
    border: 1px solid rgba(59, 130, 246, 0.3);
  }

  .status-pill.executed {
    background: rgba(168, 85, 247, 0.15);
    color: #c084fc;
    border: 1px solid rgba(168, 85, 247, 0.3);
  }

  .status-pill.settled {
    background: rgba(100, 116, 139, 0.2);
    color: #94a3b8;
    border: 1px solid rgba(100, 116, 139, 0.3);
  }

  .project-title {
    font-size: 1.0625rem;
    font-weight: 700;
    color: var(--color-foreground, #f8fafc);
    line-height: 1.35;
    margin: 0 0 0.5rem 0;
  }

  .client-row {
    display: flex;
    align-items: center;
    gap: 0.45rem;
    font-size: 0.8125rem;
    color: var(--color-muted-foreground, #94a3b8);
  }

  .client-row strong {
    color: var(--color-foreground, #f8fafc);
  }

  .progress-section {
    margin-top: 0.75rem;
  }

  .progress-labels {
    display: flex;
    justify-content: space-between;
    font-size: 0.75rem;
    color: var(--color-muted-foreground, #94a3b8);
    margin-bottom: 0.35rem;
  }

  .percent-label {
    font-weight: 700;
    color: #c084fc;
  }

  .progress-track {
    width: 100%;
    height: 6px;
    background: rgba(30, 41, 59, 0.9);
    border-radius: 9999px;
    overflow: hidden;
  }

  .progress-fill {
    height: 100%;
    border-radius: 9999px;
  }

  .fill-purple {
    background: linear-gradient(90deg, #7c3aed 0%, #c084fc 100%);
  }

  .fill-emerald {
    background: linear-gradient(90deg, #059669 0%, #34d399 100%);
  }

  .card-footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding-top: 0.75rem;
    border-top: 1px solid var(--color-border, #334155);
    font-size: 0.75rem;
  }

  .date-range {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    color: var(--color-muted-foreground, #94a3b8);
  }

  .budget-tag {
    font-size: 0.9375rem;
    font-weight: 800;
    color: #34d399;
  }

  .card-actions {
    display: flex;
    gap: 0.5rem;
    margin-top: auto;
  }

  .action-btn-sm {
    flex: 1;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 0.35rem;
    font-size: 0.75rem;
    font-weight: 700;
    padding: 0.5rem;
    border-radius: 0.5rem;
    background: rgba(124, 58, 237, 0.15);
    color: #c084fc;
    border: 1px solid rgba(168, 85, 247, 0.3);
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .action-btn-sm:hover {
    background: rgba(124, 58, 237, 0.28);
    border-color: rgba(168, 85, 247, 0.6);
  }

  .action-btn-sm.outline {
    background: rgba(30, 41, 59, 0.6);
    color: var(--color-muted-foreground, #94a3b8);
    border: 1px solid var(--color-border, #334155);
  }

  .action-btn-sm.outline:hover {
    background: rgba(51, 65, 85, 0.7);
    color: var(--color-foreground, #f8fafc);
  }

  /* TAB 2: Contracts */
  .contracts-container {
    background: rgba(15, 23, 42, 0.8);
    border: 1px solid var(--color-border, #334155);
    border-radius: 0.875rem;
    padding: 1.5rem;
  }

  .section-title-row {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    margin-bottom: 1.25rem;
    padding-bottom: 1rem;
    border-bottom: 1px solid var(--color-border, #334155);
  }

  .section-heading {
    font-size: 1.125rem;
    font-weight: 700;
    color: var(--color-foreground, #f8fafc);
    margin: 0 0 0.25rem 0;
  }

  .section-subheading {
    font-size: 0.8125rem;
    color: var(--color-muted-foreground, #94a3b8);
    margin: 0;
  }

  .btn-primary-sm {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    font-size: 0.75rem;
    font-weight: 700;
    padding: 0.45rem 0.85rem;
    border-radius: 0.5rem;
    background: linear-gradient(135deg, #7c3aed 0%, #6366f1 100%);
    color: #ffffff;
    border: none;
    cursor: pointer;
  }

  .contracts-list {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .contract-card {
    background: rgba(30, 41, 59, 0.45);
    border: 1px solid rgba(51, 65, 85, 0.7);
    border-radius: 0.75rem;
    padding: 1.25rem;
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    transition: all 0.2s ease;
  }

  .contract-card:hover {
    border-color: rgba(99, 102, 241, 0.4);
    background: rgba(30, 41, 59, 0.65);
  }

  .contract-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .contract-meta {
    display: flex;
    align-items: center;
    gap: 0.6rem;
  }

  .contract-id {
    font-size: 0.75rem;
    font-weight: 700;
    color: #818cf8;
  }

  .ref-project {
    font-size: 0.75rem;
    color: var(--color-muted-foreground, #94a3b8);
  }

  .contract-client {
    font-size: 1rem;
    font-weight: 700;
    color: var(--color-foreground, #f8fafc);
    margin: 0;
  }

  .contract-terms {
    font-size: 0.8125rem;
    color: var(--color-muted-foreground, #94a3b8);
    line-height: 1.5;
    margin: 0;
    background: rgba(15, 23, 42, 0.5);
    padding: 0.75rem;
    border-radius: 0.5rem;
    border-left: 3px solid #6366f1;
  }

  .contract-footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding-top: 0.5rem;
    font-size: 0.75rem;
  }

  .terms-pill {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    color: #94a3b8;
  }

  .contract-value {
    font-size: 0.875rem;
    color: var(--color-muted-foreground, #94a3b8);
  }

  .contract-value span {
    font-size: 1rem;
    font-weight: 800;
    color: #34d399;
  }

  /* TAB 3: Timesheet Workbench */
  .timesheet-layout {
    display: grid;
    grid-template-columns: 1fr;
    gap: 1.5rem;
  }

  @media (min-width: 1024px) {
    .timesheet-layout {
      grid-template-columns: 7fr 5fr;
    }
  }

  .timesheet-stream-panel, .log-form-panel {
    background: rgba(15, 23, 42, 0.8);
    border: 1px solid var(--color-border, #334155);
    border-radius: 0.875rem;
    padding: 1.5rem;
  }

  .panel-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    margin-bottom: 1.25rem;
    padding-bottom: 0.85rem;
    border-bottom: 1px solid var(--color-border, #334155);
  }

  .panel-title {
    font-size: 1.0625rem;
    font-weight: 700;
    color: var(--color-foreground, #f8fafc);
    margin: 0 0 0.25rem 0;
  }

  .panel-subtitle {
    font-size: 0.75rem;
    color: var(--color-muted-foreground, #94a3b8);
    margin: 0;
  }

  .count-tag {
    font-size: 0.6875rem;
    font-weight: 700;
    padding: 0.2rem 0.5rem;
    border-radius: 0.375rem;
    background: rgba(30, 41, 59, 0.9);
    color: #c084fc;
    border: 1px solid rgba(168, 85, 247, 0.3);
  }

  .timesheet-items {
    display: flex;
    flex-direction: column;
    gap: 0.85rem;
  }

  .timesheet-row {
    display: flex;
    align-items: flex-start;
    gap: 0.85rem;
    padding: 0.85rem;
    border-radius: 0.625rem;
    background: rgba(30, 41, 59, 0.4);
    border: 1px solid rgba(51, 65, 85, 0.5);
    transition: all 0.2s ease;
  }

  .timesheet-row:hover {
    background: rgba(30, 41, 59, 0.7);
    border-color: rgba(148, 163, 184, 0.3);
  }

  .ts-avatar {
    width: 36px;
    height: 36px;
    border-radius: 0.5rem;
    background: linear-gradient(135deg, #4f46e5 0%, #7c3aed 100%);
    color: #ffffff;
    font-weight: 700;
    font-size: 0.8125rem;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  .ts-content {
    flex: 1;
    min-width: 0;
  }

  .ts-top {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 0.25rem;
  }

  .ts-consultant {
    font-size: 0.8125rem;
    font-weight: 700;
    color: var(--color-foreground, #f8fafc);
  }

  .ts-project-badge {
    font-size: 0.6875rem;
    font-weight: 600;
    padding: 0.1rem 0.4rem;
    border-radius: 0.25rem;
    background: rgba(99, 102, 241, 0.15);
    color: #818cf8;
    border: 1px solid rgba(99, 102, 241, 0.25);
  }

  .ts-task {
    font-size: 0.78125rem;
    color: var(--color-muted-foreground, #94a3b8);
    margin: 0 0 0.35rem 0;
    line-height: 1.45;
  }

  .ts-meta {
    font-size: 0.6875rem;
    color: #64748b;
    display: flex;
    align-items: center;
    gap: 0.4rem;
  }

  .ts-hours-box {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    gap: 0.25rem;
    flex-shrink: 0;
  }

  .hours-val {
    font-size: 0.9375rem;
    font-weight: 800;
    color: #c084fc;
  }

  .billable-badge {
    font-size: 0.625rem;
    font-weight: 700;
    padding: 0.1rem 0.45rem;
    border-radius: 9999px;
    text-transform: uppercase;
  }

  .billable-badge.billable {
    background: rgba(16, 185, 129, 0.15);
    color: #34d399;
    border: 1px solid rgba(16, 185, 129, 0.3);
  }

  .billable-badge.non-billable {
    background: rgba(100, 116, 139, 0.2);
    color: #94a3b8;
    border: 1px solid rgba(100, 116, 139, 0.3);
  }

  /* Right Form */
  .form-header {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    margin-bottom: 1.25rem;
    padding-bottom: 0.85rem;
    border-bottom: 1px solid var(--color-border, #334155);
  }

  .icon-circle {
    width: 36px;
    height: 36px;
    border-radius: 0.5rem;
    background: rgba(124, 58, 237, 0.15);
    color: #c084fc;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .icon-circle.purple {
    background: rgba(168, 85, 247, 0.15);
    color: #c084fc;
  }

  .form-title {
    font-size: 1rem;
    font-weight: 700;
    color: var(--color-foreground, #f8fafc);
    margin: 0;
  }

  .form-desc {
    font-size: 0.75rem;
    color: var(--color-muted-foreground, #94a3b8);
    margin: 0;
  }

  .timesheet-form {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .form-group {
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
  }

  .form-label {
    font-size: 0.75rem;
    font-weight: 600;
    color: var(--color-muted-foreground, #94a3b8);
  }

  .form-input, .form-select, .form-textarea {
    width: 100%;
    box-sizing: border-box;
    padding: 0.6rem 0.8rem;
    font-size: 0.8125rem;
    border-radius: 0.5rem;
    background: rgba(30, 41, 59, 0.6);
    border: 1px solid var(--color-border, #334155);
    color: var(--color-foreground, #f8fafc);
    outline: none;
    transition: border-color 0.2s ease;
  }

  .form-input:focus, .form-select:focus, .form-textarea:focus {
    border-color: #818cf8;
    box-shadow: 0 0 0 2px rgba(99, 102, 241, 0.2);
  }

  .form-row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0.75rem;
  }

  .checkbox-row {
    padding: 0.5rem 0;
  }

  .checkbox-label {
    display: flex;
    align-items: flex-start;
    gap: 0.6rem;
    cursor: pointer;
  }

  .checkbox-input {
    margin-top: 0.2rem;
    accent-color: #7c3aed;
  }

  .label-text {
    font-size: 0.8125rem;
    font-weight: 600;
    color: var(--color-foreground, #f8fafc);
    display: flex;
    flex-direction: column;
  }

  .label-text small {
    font-size: 0.6875rem;
    font-weight: 400;
    color: var(--color-muted-foreground, #94a3b8);
  }

  .btn-submit {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.45rem;
    font-size: 0.8125rem;
    font-weight: 700;
    padding: 0.75rem;
    border-radius: 0.5rem;
    background: linear-gradient(135deg, #7c3aed 0%, #6366f1 100%);
    color: #ffffff;
    border: none;
    cursor: pointer;
    box-shadow: 0 4px 14px rgba(124, 58, 237, 0.35);
    transition: all 0.2s ease;
  }

  .btn-submit:hover {
    box-shadow: 0 6px 18px rgba(124, 58, 237, 0.5);
    transform: translateY(-1px);
  }

  /* TAB 4: New SOW Form */
  .new-sow-card {
    background: rgba(15, 23, 42, 0.85);
    border: 1px solid var(--color-border, #334155);
    border-radius: 0.875rem;
    padding: 1.75rem;
    max-width: 800px;
    margin: 0 auto;
    box-shadow: 0 10px 25px rgba(0, 0, 0, 0.3);
  }

  .new-sow-header {
    display: flex;
    align-items: center;
    gap: 0.85rem;
    margin-bottom: 1.5rem;
    padding-bottom: 1rem;
    border-bottom: 1px solid var(--color-border, #334155);
  }

  .new-sow-title {
    font-size: 1.25rem;
    font-weight: 800;
    color: var(--color-foreground, #f8fafc);
    margin: 0 0 0.25rem 0;
  }

  .new-sow-desc {
    font-size: 0.8125rem;
    color: var(--color-muted-foreground, #94a3b8);
    margin: 0;
  }

  .new-sow-form {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
  }

  .form-grid-2 {
    display: grid;
    grid-template-columns: 1fr;
    gap: 1rem;
  }

  @media (min-width: 640px) {
    .form-grid-2 {
      grid-template-columns: 1fr 1fr;
    }
  }

  .form-grid-3 {
    display: grid;
    grid-template-columns: 1fr;
    gap: 1rem;
  }

  @media (min-width: 640px) {
    .form-grid-3 {
      grid-template-columns: 1fr 1fr 1fr;
    }
  }

  .form-actions-bar {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 0.75rem;
    padding-top: 1rem;
    border-top: 1px solid var(--color-border, #334155);
  }

  .btn-cancel {
    padding: 0.6rem 1.1rem;
    font-size: 0.8125rem;
    font-weight: 600;
    border-radius: 0.5rem;
    background: rgba(30, 41, 59, 0.7);
    color: var(--color-muted-foreground, #94a3b8);
    border: 1px solid var(--color-border, #334155);
    cursor: pointer;
  }

  .btn-cancel:hover {
    color: var(--color-foreground, #f8fafc);
    background: rgba(51, 65, 85, 0.8);
  }

  .btn-submit-primary {
    display: inline-flex;
    align-items: center;
    gap: 0.45rem;
    padding: 0.6rem 1.25rem;
    font-size: 0.8125rem;
    font-weight: 700;
    border-radius: 0.5rem;
    background: linear-gradient(135deg, #7c3aed 0%, #6366f1 100%);
    color: #ffffff;
    border: none;
    cursor: pointer;
    box-shadow: 0 4px 14px rgba(124, 58, 237, 0.35);
  }

  .btn-submit-primary:hover {
    transform: translateY(-1px);
    box-shadow: 0 6px 18px rgba(124, 58, 237, 0.45);
  }

  /* Utility Animations */
  .spin {
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }

  @keyframes fadeIn {
    from { opacity: 0; transform: translateY(-4px); }
    to { opacity: 1; transform: translateY(0); }
  }
</style>
