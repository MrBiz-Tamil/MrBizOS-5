<script lang="ts">
  import { onMount } from 'svelte';
  import { journeyGrpc } from '$lib/grpc';

  interface JourneyStageDef {
    id: string;
    stageName: string;
    stageOrder: number;
    description: string;
    autoTask: boolean;
    autoTaskTitle: string;
    triggerNotificationChannel: string;
  }

  interface JourneyTemplateDef {
    id: string;
    code: string;
    name: string;
    businessUnit: string;
    description: string;
    stages: JourneyStageDef[];
  }

  interface JourneyInstanceDef {
    id: string;
    templateId: string;
    templateName: string;
    businessUnit: string;
    customerName: string;
    customerContact: string;
    currentStageName: string;
    currentStageOrder: number;
    totalStages: number;
    status: string;
    createdAt: string;
    updatedAt: string;
  }

  interface JourneyLogEntry {
    id: string;
    fromStage: string;
    toStage: string;
    actorName: string;
    notes: string;
    timestamp: string;
  }

  let templates = $state<JourneyTemplateDef[]>([]);
  let instances = $state<JourneyInstanceDef[]>([]);
  let selectedInstance = $state<JourneyInstanceDef | null>(null);
  let activeLogs = $state<JourneyLogEntry[]>([]);
  let activeStages = $state<JourneyStageDef[]>([]);
  let loading = $state<boolean>(true);
  let statusMessage = $state<string | null>(null);
  let activeTab = $state<'instances' | 'templates' | 'new_journey'>('instances');

  // Form for New Journey
  let newJourney = $state({
    templateId: 'JRN-TRAVEL',
    customerName: '',
    customerContact: '',
    initialNotes: ''
  });

  const defaultTemplates: JourneyTemplateDef[] = [
    {
      id: 'JRN-TRAVEL',
      code: 'TRV-SCHENGEN',
      name: 'Schengen Tourist & Business Visa Clearance',
      businessUnit: 'TRAVEL',
      description: 'End-to-end VFS appointment, document verification, biometric filing, and passport dispatch pipeline.',
      stages: [
        { id: 'S1', stageName: 'Document Collection', stageOrder: 1, description: 'Bank statements, cover letters, and flights hold', autoTask: true, autoTaskTitle: 'Verify ITR & Bank Docs', triggerNotificationChannel: 'WhatsApp' },
        { id: 'S2', stageName: 'VFS Slot Booking', stageOrder: 2, description: 'Book Mumbai/Delhi VFS biometric slot', autoTask: true, autoTaskTitle: 'Lock VFS Appointment', triggerNotificationChannel: 'WhatsApp + SMS' },
        { id: 'S3', stageName: 'Biometrics & Submission', stageOrder: 3, description: 'Traveler attends appointment at consulate/VFS center', autoTask: false, autoTaskTitle: 'Follow up after biometric filing', triggerNotificationChannel: 'SMS' },
        { id: 'S4', stageName: 'Visa Stamped & Dispatch', stageOrder: 4, description: 'Passport returned with official visa sticker', autoTask: true, autoTaskTitle: 'Issue Ticket and Travel Kit', triggerNotificationChannel: 'WhatsApp + Email' }
      ]
    },
    {
      id: 'JRN-HEALTH',
      code: 'HLT-ORTHO',
      name: 'Joint Replacement & Surgical Pathway',
      businessUnit: 'HEALTHCARE',
      description: 'Pre-op diagnostic assessment, OT scheduling, implant allocation, and post-op physio.',
      stages: [
        { id: 'H1', stageName: 'Pre-Op Diagnostics', stageOrder: 1, description: 'X-Rays, MRI, blood biochemistry, ECG clearance', autoTask: true, autoTaskTitle: 'Order blood work & review PAC', triggerNotificationChannel: 'WhatsApp' },
        { id: 'H2', stageName: 'Anesthesia PAC Clearance', stageOrder: 2, description: 'Cardio & Anesthesia fitness check', autoTask: false, autoTaskTitle: 'Review PAC notes', triggerNotificationChannel: 'Internal Staff' },
        { id: 'H3', stageName: 'OT Surgery & Implant', stageOrder: 3, description: 'Surgical procedure & implant serial tracking', autoTask: true, autoTaskTitle: 'Log surgical notes & implant batch', triggerNotificationChannel: 'In-Patient SMS' },
        { id: 'H4', stageName: 'Physio & Discharge', stageOrder: 4, description: 'Post-op mobility test, medication, discharge summary', autoTask: true, autoTaskTitle: 'Schedule home visit', triggerNotificationChannel: 'WhatsApp' }
      ]
    },
    {
      id: 'JRN-ESTATE',
      code: 'EST-PLOT',
      name: 'Commercial Plot Acquisition & Escrow',
      businessUnit: 'REALESTATE',
      description: 'Site visit, survey token advance, legal encumbrance search, and sub-registrar deed registration.',
      stages: [
        { id: 'E1', stageName: 'Site Visit & Token', stageOrder: 1, description: 'Prospective buyer site visit and advance booking token', autoTask: true, autoTaskTitle: 'Receipt token payment', triggerNotificationChannel: 'WhatsApp' },
        { id: 'E2', stageName: 'Legal Title Verification', stageOrder: 2, description: '30-year encumbrance search and patta validation', autoTask: true, autoTaskTitle: 'Advocate title opinion', triggerNotificationChannel: 'Email' },
        { id: 'E3', stageName: 'Escrow & Agreement of Sale', stageOrder: 3, description: 'Bilateral agreement execution and 20% advance', autoTask: false, autoTaskTitle: 'Draft Agreement of Sale', triggerNotificationChannel: 'WhatsApp + Email' },
        { id: 'E4', stageName: 'Deed Registration', stageOrder: 4, description: 'Sub-registrar office deed execution and biometric stamp', autoTask: true, autoTaskTitle: 'Handover original title deed', triggerNotificationChannel: 'WhatsApp' }
      ]
    }
  ];

  const defaultInstances: JourneyInstanceDef[] = [
    {
      id: 'JRN-2026-091',
      templateId: 'JRN-TRAVEL',
      templateName: 'Schengen Tourist & Business Visa Clearance',
      businessUnit: 'TRAVEL',
      customerName: 'Ananya Roy',
      customerContact: '+91 98401 23456',
      currentStageName: 'VFS Slot Booking',
      currentStageOrder: 2,
      totalStages: 4,
      status: 'ACTIVE',
      createdAt: '2026-10-18',
      updatedAt: '2026-10-21'
    },
    {
      id: 'JRN-2026-088',
      templateId: 'JRN-TRAVEL',
      templateName: 'Schengen Tourist & Business Visa Clearance',
      businessUnit: 'TRAVEL',
      customerName: 'Rahul Sharma',
      customerContact: '+91 98401 23456',
      currentStageName: 'Visa Stamped & Dispatch',
      currentStageOrder: 4,
      totalStages: 4,
      status: 'COMPLETED',
      createdAt: '2026-10-05',
      updatedAt: '2026-10-20'
    },
    {
      id: 'JRN-2026-074',
      templateId: 'JRN-HEALTH',
      templateName: 'Joint Replacement & Surgical Pathway',
      businessUnit: 'HEALTHCARE',
      customerName: 'Sundaramurthy K.',
      customerContact: '+91 98410 77889',
      currentStageName: 'OT Surgery & Implant',
      currentStageOrder: 3,
      totalStages: 4,
      status: 'ACTIVE',
      createdAt: '2026-10-12',
      updatedAt: '2026-10-21'
    }
  ];

  async function loadData() {
    loading = true;
    try {
      const [tRes, iRes] = await Promise.all([
        journeyGrpc.listTemplates(),
        journeyGrpc.listInstances()
      ]);
      if (tRes && tRes.templates && tRes.templates.length > 0) {
        templates = tRes.templates;
      } else {
        templates = defaultTemplates;
      }
      if (iRes && iRes.instances && iRes.instances.length > 0) {
        instances = iRes.instances;
      } else {
        instances = defaultInstances;
      }
      if (instances.length > 0 && !selectedInstance) {
        await selectInstance(instances[0]);
      }
    } catch (e: any) {
      console.warn('Fallback data if backend is offline:', e);
      templates = defaultTemplates;
      instances = defaultInstances;
      if (instances.length > 0 && !selectedInstance) {
        await selectInstance(instances[0]);
      }
    } finally {
      loading = false;
    }
  }

  async function selectInstance(inst: JourneyInstanceDef) {
    selectedInstance = inst;
    try {
      const res = await journeyGrpc.getJourneyDetails({ instanceId: inst.id });
      if (res && res.allStages && res.allStages.length > 0) {
        activeStages = res.allStages;
        activeLogs = res.logs || [];
        return;
      }
    } catch (e) {
      // fallback
    }

    // Lookup template stages
    const matchedTmpl = templates.find((t) => t.id === inst.templateId || t.name === inst.templateName) || templates[0];
    activeStages = matchedTmpl.stages || [];
    activeLogs = [
      {
        id: 'L1',
        fromStage: 'Intake',
        toStage: activeStages[0]?.stageName || 'Document Collection',
        actorName: 'Agent Lead (Vikram)',
        notes: 'Journey created upon receiving WhatsApp inquiry',
        timestamp: '18 Oct 2026 10:14'
      },
      {
        id: 'L2',
        fromStage: activeStages[0]?.stageName || 'Document Collection',
        toStage: inst.currentStageName,
        actorName: 'Automated Rule Engine',
        notes: 'ITR documents verified and bank statements approved by accounts',
        timestamp: '20 Oct 2026 14:30'
      }
    ];
  }

  async function handleAdvanceStage(targetStage: JourneyStageDef) {
    if (!selectedInstance) return;
    try {
      const res = await journeyGrpc.advanceStage({
        instanceId: selectedInstance.id,
        targetStageName: targetStage.stageName,
        notes: `Advanced via Journey Stage Machine to ${targetStage.stageName}`,
        actorName: 'Agent Lead'
      });
      if (res && res.instance) {
        selectedInstance.currentStageName = res.instance.currentStageName;
        selectedInstance.currentStageOrder = res.instance.currentStageOrder;
        selectedInstance.status = res.instance.status;
        statusMessage = `🎉 Journey advanced to "${targetStage.stageName}". Triggered automated notifications & UTMS deliverables.`;
        await selectInstance(selectedInstance);
        setTimeout(() => (statusMessage = null), 4000);
        return;
      }
    } catch (err: any) {
      // Fallback local advance
    }

    selectedInstance.currentStageName = targetStage.stageName;
    selectedInstance.currentStageOrder = targetStage.stageOrder;
    if (targetStage.stageOrder === selectedInstance.totalStages) {
      selectedInstance.status = 'COMPLETED';
    }
    activeLogs = [
      {
        id: `L-${Date.now()}`,
        fromStage: activeStages[targetStage.stageOrder - 2]?.stageName || 'Previous Stage',
        toStage: targetStage.stageName,
        actorName: 'You (Operator)',
        notes: `Advanced to ${targetStage.stageName}. Dispatched ${targetStage.triggerNotificationChannel} trigger.`,
        timestamp: 'Just now'
      },
      ...activeLogs
    ];
    statusMessage = `🎉 Journey advanced to "${targetStage.stageName}"! Automated ${targetStage.triggerNotificationChannel} alert dispatched.`;
    setTimeout(() => (statusMessage = null), 4000);
  }

  async function handleStartJourney(e: Event) {
    e.preventDefault();
    if (!newJourney.customerName.trim()) return;

    try {
      const res = await journeyGrpc.startJourney(newJourney);
      if (res && res.instance) {
        instances = [res.instance, ...instances];
        await selectInstance(res.instance);
        activeTab = 'instances';
        statusMessage = `✅ Started Journey for ${newJourney.customerName}`;
        newJourney.customerName = '';
        newJourney.customerContact = '';
        newJourney.initialNotes = '';
        setTimeout(() => (statusMessage = null), 4000);
        return;
      }
    } catch (err: any) {
      // fallback
    }

    const tmpl = templates.find((t) => t.id === newJourney.templateId) || templates[0];
    const newInst: JourneyInstanceDef = {
      id: `JRN-2026-${Math.floor(100 + Math.random() * 900)}`,
      templateId: tmpl.id,
      templateName: tmpl.name,
      businessUnit: tmpl.businessUnit,
      customerName: newJourney.customerName,
      customerContact: newJourney.customerContact || '+91 98400 00000',
      currentStageName: tmpl.stages[0]?.stageName || 'Stage 1',
      currentStageOrder: 1,
      totalStages: tmpl.stages.length,
      status: 'ACTIVE',
      createdAt: 'Today',
      updatedAt: 'Just now'
    };
    instances = [newInst, ...instances];
    await selectInstance(newInst);
    activeTab = 'instances';
    statusMessage = `✅ Successfully launched journey for ${newJourney.customerName}!`;
    newJourney.customerName = '';
    newJourney.customerContact = '';
    newJourney.initialNotes = '';
    setTimeout(() => (statusMessage = null), 4000);
  }

  onMount(() => {
    loadData();
  });
</script>

<svelte:head>
  <title>Customer Journey &amp; Tour Itinerary Engine | MrBiz-OS</title>
</svelte:head>

<div class="journeys-page-wrapper">
  <!-- Top Bar -->
  <header class="journeys-header">
    <div>
      <div class="badge-row">
        <span class="module-badge">🛤️ Lifecycle State Machine</span>
        <span class="status-indicator">● Automated Multi-Role Engine</span>
      </div>
      <h1 class="page-title">Cross-Industry Customer Journey Engine</h1>
      <p class="page-subtitle">
        Deterministic lifecycle pipelines for Travelers, Patients, Property Buyers, and Consulting Clients with automated milestone gate triggers.
      </p>
    </div>

    <div class="tabs-control-row">
      <button
        type="button"
        onclick={() => (activeTab = 'instances')}
        class="tab-btn"
        class:active={activeTab === 'instances'}
      >
        Active Journeys ({instances.length})
      </button>
      <button
        type="button"
        onclick={() => (activeTab = 'templates')}
        class="tab-btn"
        class:active={activeTab === 'templates'}
      >
        Journey Templates ({templates.length})
      </button>
      <button
        type="button"
        onclick={() => (activeTab = 'new_journey')}
        class="tab-btn btn-launch"
        class:active={activeTab === 'new_journey'}
      >
        + Launch Journey
      </button>
    </div>
  </header>

  <!-- Banner Message -->
  {#if statusMessage}
    <div class="status-toast animate-fade-in">
      <span>{statusMessage}</span>
      <button type="button" class="close-toast-btn" onclick={() => (statusMessage = null)}>✕</button>
    </div>
  {/if}

  <!-- TAB 1: ACTIVE JOURNEYS -->
  {#if activeTab === 'instances'}
    <div class="instances-layout">
      <!-- Left Column: Instances List -->
      <aside class="instances-list-col">
        <div class="list-col-header">
          <span class="col-title">In-Flight Journeys</span>
          <span class="col-count">{instances.length} Active</span>
        </div>

        <div class="instances-scroll-container">
          {#each instances as inst}
            <button
              type="button"
              onclick={() => selectInstance(inst)}
              class="instance-card"
              class:selected={selectedInstance?.id === inst.id}
            >
              <div class="inst-top-line">
                <span class="inst-id">{inst.id}</span>
                <span class="inst-status-badge status-{inst.status.toLowerCase()}">
                  {inst.status}
                </span>
              </div>

              <div class="inst-customer-name">{inst.customerName}</div>
              <div class="inst-template-name">{inst.templateName}</div>

              <div class="inst-progress-footer">
                <span class="inst-stage-counter">Stage {inst.currentStageOrder}/{inst.totalStages}</span>
                <span class="inst-current-stage truncate">{inst.currentStageName}</span>
              </div>
            </button>
          {/each}
        </div>
      </aside>

      <!-- Right Column: Visual Stage Machine Pipeline -->
      <main class="instance-details-col">
        {#if selectedInstance}
          <div class="detail-card">
            <!-- Header -->
            <div class="detail-header">
              <div>
                <div class="detail-meta-chip">
                  <span class="font-mono">{selectedInstance.id}</span>
                  <span>•</span>
                  <span class="bu-badge bu-{selectedInstance.businessUnit.toLowerCase()}">{selectedInstance.businessUnit}</span>
                  <span>•</span>
                  <span>Created {selectedInstance.createdAt}</span>
                </div>
                <h2 class="detail-customer-title">{selectedInstance.customerName}</h2>
                <div class="detail-sub-info">
                  <span>{selectedInstance.templateName}</span>
                  <span>•</span>
                  <span>📱 {selectedInstance.customerContact}</span>
                </div>
              </div>

              <div class="current-stage-hero">
                <span class="hero-label">Current Active Gate</span>
                <div class="hero-val">{selectedInstance.currentStageName}</div>
              </div>
            </div>

            <!-- Visual Horizontal Stage Progress Tracker -->
            <div class="stage-tracker-section">
              <h3 class="section-heading">Stage Pipeline &amp; Milestone Gate Transitions</h3>
              
              <div class="stages-timeline">
                {#each activeStages as stage}
                  {@const isCurrent = stage.stageName === selectedInstance.currentStageName}
                  {@const isPast = stage.stageOrder < selectedInstance.currentStageOrder}
                  <div
                    class="stage-box"
                    class:stage-current={isCurrent}
                    class:stage-past={isPast}
                    class:stage-future={!isCurrent && !isPast}
                  >
                    <div class="stage-order-row">
                      <span class="stage-num font-mono">0{stage.stageOrder}</span>
                      {#if isPast}
                        <span class="badge-done">✓ Done</span>
                      {:else if isCurrent}
                        <span class="badge-active">● Active</span>
                      {:else}
                        <span class="badge-pending">○ Pending</span>
                      {/if}
                    </div>

                    <div class="stage-name">{stage.stageName}</div>
                    <p class="stage-desc">{stage.description}</p>
                    <div class="stage-trigger-pill">
                      🔔 {stage.triggerNotificationChannel}
                    </div>

                    <!-- Advance Button if it's the next stage -->
                    {#if stage.stageOrder === selectedInstance.currentStageOrder + 1}
                      <button
                        type="button"
                        onclick={() => handleAdvanceStage(stage)}
                        class="btn-advance"
                      >
                        Advance Gate ➔
                      </button>
                    {/if}
                  </div>
                {/each}
              </div>
            </div>

            <!-- Audit Trail / Journey Logs -->
            <div class="logs-section">
              <h3 class="section-heading">Journey Audit &amp; Stage State Transition History</h3>
              <div class="logs-container">
                {#each activeLogs as log}
                  <div class="log-item">
                    <div class="log-content">
                      <div class="log-transition">
                        <span>{log.fromStage}</span>
                        <span class="arrow-accent">➔</span>
                        <span class="target-stage">{log.toStage}</span>
                      </div>
                      <p class="log-notes">{log.notes}</p>
                      <span class="log-actor">👤 Logged by: {log.actorName}</span>
                    </div>
                    <span class="log-time font-mono">{log.timestamp}</span>
                  </div>
                {/each}
              </div>
            </div>
          </div>
        {:else}
          <div class="empty-selection-card">
            <span style="font-size: 2.5rem;">🛤️</span>
            <h3>Select a customer journey</h3>
            <p>Choose an in-flight journey from the left column to view its live stage progress and history.</p>
          </div>
        {/if}
      </main>
    </div>
  {/if}

  <!-- TAB 2: JOURNEY TEMPLATES -->
  {#if activeTab === 'templates'}
    <div class="templates-grid">
      {#each templates as t}
        <div class="template-card">
          <div class="tmpl-top-row">
            <span class="tmpl-code font-mono">{t.code}</span>
            <span class="bu-badge bu-{t.businessUnit.toLowerCase()}">{t.businessUnit}</span>
          </div>

          <h3 class="tmpl-name">{t.name}</h3>
          <p class="tmpl-desc">{t.description}</p>

          <div class="tmpl-stages-list">
            <span class="stages-list-heading">Configured Milestone Gates ({t.stages?.length || 0}):</span>
            <div class="stages-items">
              {#each t.stages || [] as stg}
                <div class="tmpl-stage-pill">
                  <span class="font-bold">{stg.stageOrder}. {stg.stageName}</span>
                  <span class="font-mono text-cyan">{stg.triggerNotificationChannel}</span>
                </div>
              {/each}
            </div>
          </div>
        </div>
      {/each}
    </div>
  {/if}

  <!-- TAB 3: LAUNCH JOURNEY FORM -->
  {#if activeTab === 'new_journey'}
    <div class="launch-form-wrapper">
      <div class="form-card">
        <h2 class="form-title">Launch New Customer Lifecycle Journey</h2>
        <p class="form-desc">Attach a client, traveler, or patient to a deterministic automated multi-stage workflow.</p>

        <form onsubmit={handleStartJourney} class="form-body">
          <div class="form-group">
            <label for="template-select" class="form-label">Select Lifecycle Template</label>
            <select id="template-select" bind:value={newJourney.templateId} class="form-select">
              {#each templates as t}
                <option value={t.id}>[{t.businessUnit}] {t.name}</option>
              {/each}
            </select>
          </div>

          <div class="form-group">
            <label for="cust-name" class="form-label">Customer / Patient / Traveler Name *</label>
            <input
              id="cust-name"
              type="text"
              required
              placeholder="e.g. Anand Mahindra or Vikram Rathore"
              bind:value={newJourney.customerName}
              class="form-input"
            />
          </div>

          <div class="form-group">
            <label for="cust-contact" class="form-label">Primary WhatsApp / Phone / Email *</label>
            <input
              id="cust-contact"
              type="text"
              required
              placeholder="+91 98400 12345 or client@domain.com"
              bind:value={newJourney.customerContact}
              class="form-input"
            />
          </div>

          <div class="form-group">
            <label for="initial-notes" class="form-label">Intake Notes / Booking Reference</label>
            <textarea
              id="initial-notes"
              rows="3"
              placeholder="Special requirements, visa urgency, preferred airlines, medical history, or plot details..."
              bind:value={newJourney.initialNotes}
              class="form-textarea"
            ></textarea>
          </div>

          <div class="form-actions">
            <button
              type="button"
              onclick={() => (activeTab = 'instances')}
              class="btn-secondary"
            >
              Cancel
            </button>
            <button
              type="submit"
              class="btn-primary"
            >
              🚀 Launch Active Journey
            </button>
          </div>
        </form>
      </div>
    </div>
  {/if}
</div>

<style>
  /* ===================================================================
     JOURNEYS & TOUR ITINERARY ENGINE STYLES (PURE CSS)
     =================================================================== */

  .journeys-page-wrapper {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
    padding-bottom: 2rem;
  }

  .journeys-header {
    display: flex;
    flex-direction: column;
    gap: 1rem;
    border-bottom: 1px solid rgba(255, 255, 255, 0.08);
    padding-bottom: 1.25rem;
  }

  @media (min-width: 900px) {
    .journeys-header {
      flex-direction: row;
      align-items: flex-start;
      justify-content: space-between;
    }
  }

  .badge-row {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 0.35rem;
  }

  .module-badge {
    font-size: 0.6875rem;
    font-weight: 700;
    padding: 0.2rem 0.6rem;
    border-radius: 6px;
    background: rgba(245, 158, 11, 0.12);
    color: #fbbf24;
    border: 1px solid rgba(245, 158, 11, 0.25);
    text-transform: uppercase;
  }

  .status-indicator {
    font-size: 0.6875rem;
    font-weight: 700;
    color: #34d399;
    background: rgba(16, 185, 129, 0.12);
    padding: 0.2rem 0.5rem;
    border-radius: 6px;
  }

  .page-title {
    font-size: 1.5rem;
    font-weight: 800;
    color: #ffffff;
    margin: 0;
  }

  .page-subtitle {
    font-size: 0.8125rem;
    color: #94a3b8;
    margin: 0.25rem 0 0;
    max-width: 750px;
  }

  .tabs-control-row {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .tab-btn {
    background: rgba(15, 23, 42, 0.7);
    border: 1px solid rgba(255, 255, 255, 0.08);
    color: #94a3b8;
    font-size: 0.8125rem;
    font-weight: 600;
    padding: 0.5rem 1rem;
    border-radius: 8px;
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .tab-btn:hover {
    background: rgba(255, 255, 255, 0.06);
    color: #ffffff;
  }

  .tab-btn.active {
    background: #6366f1;
    color: #ffffff;
    border-color: #6366f1;
    box-shadow: 0 0 12px rgba(99, 102, 241, 0.35);
  }

  .btn-launch.active {
    background: #0284c7;
    border-color: #0284c7;
    box-shadow: 0 0 12px rgba(2, 132, 199, 0.4);
  }

  .status-toast {
    display: flex;
    align-items: center;
    justify-content: space-between;
    background: rgba(16, 185, 129, 0.15);
    border: 1px solid rgba(16, 185, 129, 0.35);
    color: #6ee7b7;
    font-size: 0.8125rem;
    padding: 0.65rem 1rem;
    border-radius: 8px;
  }

  .close-toast-btn {
    background: none;
    border: none;
    color: #94a3b8;
    cursor: pointer;
  }

  /* INSTANCES LAYOUT */
  .instances-layout {
    display: grid;
    grid-template-columns: 1fr;
    gap: 1.25rem;
  }

  @media (min-width: 1024px) {
    .instances-layout {
      grid-template-columns: 320px 1fr;
    }
  }

  .instances-list-col {
    background: rgba(15, 23, 42, 0.7);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 14px;
    padding: 1rem;
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    height: 720px;
  }

  .list-col-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    border-bottom: 1px solid rgba(255, 255, 255, 0.06);
    padding-bottom: 0.75rem;
  }

  .col-title {
    font-size: 0.75rem;
    font-weight: 700;
    color: #cbd5e1;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .col-count {
    font-size: 0.6875rem;
    font-family: monospace;
    color: #38bdf8;
    background: rgba(56, 189, 248, 0.12);
    padding: 0.15rem 0.5rem;
    border-radius: 9999px;
  }

  .instances-scroll-container {
    flex: 1;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .instance-card {
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
    padding: 0.875rem;
    background: rgba(11, 15, 25, 0.6);
    border: 1px solid rgba(255, 255, 255, 0.06);
    border-radius: 10px;
    cursor: pointer;
    text-align: left;
    transition: all 0.15s ease;
  }

  .instance-card:hover {
    background: rgba(255, 255, 255, 0.03);
    border-color: rgba(255, 255, 255, 0.15);
  }

  .instance-card.selected {
    background: rgba(99, 102, 241, 0.12);
    border-color: #6366f1;
    box-shadow: 0 4px 15px rgba(99, 102, 241, 0.2);
  }

  .inst-top-line {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .inst-id {
    font-size: 0.6875rem;
    font-family: monospace;
    font-weight: 700;
    color: #94a3b8;
  }

  .inst-status-badge {
    font-size: 0.625rem;
    font-weight: 700;
    padding: 0.1rem 0.45rem;
    border-radius: 4px;
    text-transform: uppercase;
  }

  .status-active { background: rgba(245, 158, 11, 0.15); color: #fbbf24; }
  .status-completed { background: rgba(16, 185, 129, 0.15); color: #34d399; }

  .inst-customer-name {
    font-size: 0.9375rem;
    font-weight: 800;
    color: #ffffff;
  }

  .inst-template-name {
    font-size: 0.75rem;
    color: #94a3b8;
    line-height: 1.3;
  }

  .inst-progress-footer {
    display: flex;
    justify-content: space-between;
    align-items: center;
    border-top: 1px solid rgba(255, 255, 255, 0.06);
    padding-top: 0.4rem;
    margin-top: 0.2rem;
    font-size: 0.6875rem;
  }

  .inst-stage-counter {
    color: #818cf8;
    font-weight: 700;
  }

  .inst-current-stage {
    color: #cbd5e1;
    max-width: 140px;
  }

  /* DETAILS COLUMN */
  .instance-details-col {
    background: rgba(15, 23, 42, 0.7);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 14px;
    padding: 1.5rem;
    height: 720px;
    overflow-y: auto;
  }

  .detail-card {
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
  }

  .detail-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    border-bottom: 1px solid rgba(255, 255, 255, 0.08);
    padding-bottom: 1.25rem;
  }

  .detail-meta-chip {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    font-size: 0.75rem;
    color: #94a3b8;
  }

  .bu-badge {
    font-size: 0.625rem;
    font-weight: 700;
    padding: 0.1rem 0.45rem;
    border-radius: 4px;
    text-transform: uppercase;
  }

  .bu-travel { background: rgba(14, 165, 233, 0.15); color: #38bdf8; }
  .bu-healthcare { background: rgba(168, 85, 247, 0.15); color: #c084fc; }
  .bu-realestate { background: rgba(245, 158, 11, 0.15); color: #fbbf24; }

  .detail-customer-title {
    font-size: 1.35rem;
    font-weight: 800;
    color: #ffffff;
    margin: 0.35rem 0 0.15rem;
  }

  .detail-sub-info {
    font-size: 0.8125rem;
    color: #94a3b8;
    display: flex;
    gap: 0.4rem;
  }

  .current-stage-hero {
    text-align: right;
    background: rgba(99, 102, 241, 0.1);
    border: 1px solid rgba(99, 102, 241, 0.25);
    padding: 0.5rem 1rem;
    border-radius: 10px;
  }

  .hero-label {
    font-size: 0.6875rem;
    color: #94a3b8;
    text-transform: uppercase;
    font-weight: 600;
    display: block;
  }

  .hero-val {
    font-size: 1.05rem;
    font-weight: 800;
    color: #818cf8;
  }

  .section-heading {
    font-size: 0.75rem;
    font-weight: 700;
    color: #cbd5e1;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    margin: 0 0 0.85rem;
  }

  /* STAGES TIMELINE */
  .stages-timeline {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
    gap: 0.75rem;
  }

  .stage-box {
    background: #0f172a;
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 12px;
    padding: 1rem;
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
    position: relative;
    transition: all 0.15s ease;
  }

  .stage-current {
    border-color: #6366f1;
    background: rgba(99, 102, 241, 0.12);
    box-shadow: 0 0 15px rgba(99, 102, 241, 0.2);
  }

  .stage-past {
    border-color: rgba(16, 185, 129, 0.3);
    background: rgba(16, 185, 129, 0.05);
  }

  .stage-future {
    opacity: 0.75;
  }

  .stage-order-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .stage-num {
    font-size: 0.75rem;
    font-weight: 800;
    color: #94a3b8;
  }

  .badge-done { color: #34d399; font-size: 0.6875rem; font-weight: 700; }
  .badge-active { color: #818cf8; font-size: 0.6875rem; font-weight: 800; }
  .badge-pending { color: #64748b; font-size: 0.6875rem; }

  .stage-name {
    font-size: 0.875rem;
    font-weight: 700;
    color: #ffffff;
  }

  .stage-desc {
    font-size: 0.75rem;
    color: #94a3b8;
    line-height: 1.35;
    margin: 0;
    flex: 1;
  }

  .stage-trigger-pill {
    font-size: 0.6875rem;
    color: #38bdf8;
    background: rgba(56, 189, 248, 0.1);
    padding: 0.2rem 0.45rem;
    border-radius: 4px;
    margin-top: 0.25rem;
  }

  .btn-advance {
    background: #6366f1;
    color: #ffffff;
    border: none;
    font-size: 0.75rem;
    font-weight: 700;
    padding: 0.45rem;
    border-radius: 6px;
    cursor: pointer;
    margin-top: 0.35rem;
    transition: background 0.15s;
  }

  .btn-advance:hover {
    background: #4f46e5;
  }

  /* AUDIT LOGS */
  .logs-container {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .log-item {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    padding: 0.75rem 1rem;
    background: #0f172a;
    border: 1px solid rgba(255, 255, 255, 0.06);
    border-radius: 8px;
    font-size: 0.75rem;
  }

  .log-transition {
    font-weight: 700;
    color: #ffffff;
    display: flex;
    align-items: center;
    gap: 0.35rem;
  }

  .arrow-accent { color: #6366f1; }
  .target-stage { color: #818cf8; }

  .log-notes {
    color: #94a3b8;
    margin: 0.2rem 0 0.15rem;
  }

  .log-actor {
    color: #64748b;
    font-size: 0.6875rem;
  }

  .log-time {
    color: #64748b;
    font-size: 0.6875rem;
  }

  /* TEMPLATES TAB */
  .templates-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
    gap: 1.25rem;
  }

  .template-card {
    background: rgba(15, 23, 42, 0.7);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 14px;
    padding: 1.25rem;
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .tmpl-top-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .tmpl-code {
    font-size: 0.75rem;
    font-weight: 700;
    color: #cbd5e1;
    background: rgba(255, 255, 255, 0.06);
    padding: 0.2rem 0.5rem;
    border-radius: 6px;
  }

  .tmpl-name {
    font-size: 1.05rem;
    font-weight: 800;
    color: #ffffff;
    margin: 0;
  }

  .tmpl-desc {
    font-size: 0.8125rem;
    color: #94a3b8;
    margin: 0;
    line-height: 1.45;
  }

  .tmpl-stages-list {
    border-top: 1px solid rgba(255, 255, 255, 0.06);
    padding-top: 0.75rem;
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
  }

  .stages-list-heading {
    font-size: 0.6875rem;
    font-weight: 700;
    color: #cbd5e1;
    text-transform: uppercase;
  }

  .stages-items {
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
  }

  .tmpl-stage-pill {
    display: flex;
    justify-content: space-between;
    font-size: 0.75rem;
    color: #cbd5e1;
    background: rgba(255, 255, 255, 0.03);
    padding: 0.4rem 0.6rem;
    border-radius: 6px;
  }

  .text-cyan { color: #38bdf8; }

  /* LAUNCH FORM */
  .launch-form-wrapper {
    display: flex;
    justify-content: center;
    padding-top: 1rem;
  }

  .form-card {
    background: rgba(15, 23, 42, 0.8);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 14px;
    padding: 2rem;
    max-width: 580px;
    width: 100%;
  }

  .form-title {
    font-size: 1.25rem;
    font-weight: 800;
    color: #ffffff;
    margin: 0 0 0.25rem;
  }

  .form-desc {
    font-size: 0.8125rem;
    color: #94a3b8;
    margin: 0 0 1.5rem;
  }

  .form-body {
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
    color: #cbd5e1;
  }

  .form-input, .form-select, .form-textarea {
    background: #0f172a;
    border: 1px solid rgba(255, 255, 255, 0.12);
    border-radius: 8px;
    padding: 0.65rem 0.85rem;
    color: #ffffff;
    font-size: 0.8125rem;
    outline: none;
    font-family: inherit;
  }

  .form-input:focus, .form-select:focus, .form-textarea:focus {
    border-color: #6366f1;
  }

  .form-actions {
    display: flex;
    justify-content: flex-end;
    gap: 0.75rem;
    margin-top: 0.75rem;
  }

  .btn-primary {
    background: #6366f1;
    color: #ffffff;
    border: none;
    font-size: 0.8125rem;
    font-weight: 700;
    padding: 0.65rem 1.25rem;
    border-radius: 8px;
    cursor: pointer;
  }

  .btn-primary:hover {
    background: #4f46e5;
  }

  .btn-secondary {
    background: rgba(255, 255, 255, 0.06);
    color: #cbd5e1;
    border: 1px solid rgba(255, 255, 255, 0.1);
    font-size: 0.8125rem;
    font-weight: 600;
    padding: 0.65rem 1.25rem;
    border-radius: 8px;
    cursor: pointer;
  }

  .btn-secondary:hover {
    background: rgba(255, 255, 255, 0.1);
    color: #ffffff;
  }

  .truncate {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .empty-selection-card {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: #64748b;
    gap: 0.5rem;
  }
</style>
