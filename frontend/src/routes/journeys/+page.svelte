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
    templateId: 'JRN-HEALTH',
    customerName: '',
    customerContact: '',
    initialNotes: ''
  });

  async function loadData() {
    loading = true;
    try {
      const [tRes, iRes] = await Promise.all([
        journeyGrpc.listTemplates(),
        journeyGrpc.listInstances()
      ]);
      if (tRes && tRes.templates) {
        templates = tRes.templates;
      }
      if (iRes && iRes.instances) {
        instances = iRes.instances;
        if (instances.length > 0 && !selectedInstance) {
          await selectInstance(instances[0]);
        }
      }
    } catch (e: any) {
      console.warn('Fallback data if backend is offline:', e);
    } finally {
      loading = false;
    }
  }

  async function selectInstance(inst: JourneyInstanceDef) {
    selectedInstance = inst;
    try {
      const res = await journeyGrpc.getJourneyDetails({ instanceId: inst.id });
      if (res) {
        activeStages = res.allStages || [];
        activeLogs = res.logs || [];
      }
    } catch (e) {
      activeStages = [];
      activeLogs = [];
    }
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
      }
    } catch (err: any) {
      statusMessage = `❌ Error: ${err.message}`;
    }
  }

  async function handleStartJourney(e: Event) {
    e.preventDefault();
    try {
      const res = await journeyGrpc.startJourney(newJourney);
      if (res && res.instance) {
        instances = [...instances, res.instance];
        await selectInstance(res.instance);
        activeTab = 'instances';
        statusMessage = `✅ Started Journey for ${newJourney.customerName}`;
        newJourney.customerName = '';
        newJourney.customerContact = '';
        newJourney.initialNotes = '';
      }
    } catch (err: any) {
      statusMessage = `❌ Error: ${err.message}`;
    }
  }

  onMount(() => {
    loadData();
  });
</script>

<svelte:head>
  <title>Journey Lifecycle Engine | MrBiz-OS (Odoo Killer)</title>
</svelte:head>

<div class="space-y-6">
  <!-- Top Bar -->
  <div class="flex flex-col md:flex-row md:items-center justify-between gap-4 border-b border-border pb-5">
    <div>
      <div class="flex items-center gap-2">
        <span class="px-2 py-0.5 text-xs font-semibold rounded bg-amber-500/10 text-amber-400 border border-amber-500/20">
          Module: Journey Lifecycle Engine
        </span>
        <span class="px-2 py-0.5 text-xs font-semibold rounded bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
          Automated State Machine & Triggers
        </span>
      </div>
      <h1 class="text-2xl font-bold tracking-tight text-foreground mt-1">Cross-Industry Customer Journey Engine</h1>
      <p class="text-sm text-muted-foreground">
        Deterministic lifecycle pipelines for Patients, Travelers, Property Buyers, and Consulting Clients with automated milestone triggers.
      </p>
    </div>

    <div class="flex items-center gap-2">
      <button
        onclick={() => (activeTab = 'instances')}
        class="px-3 py-1.5 text-sm font-medium rounded-lg transition-colors {activeTab === 'instances' ? 'bg-primary text-primary-foreground' : 'bg-muted/50 hover:bg-muted text-muted-foreground'}"
      >
        Active Journeys
      </button>
      <button
        onclick={() => (activeTab = 'templates')}
        class="px-3 py-1.5 text-sm font-medium rounded-lg transition-colors {activeTab === 'templates' ? 'bg-primary text-primary-foreground' : 'bg-muted/50 hover:bg-muted text-muted-foreground'}"
      >
        Journey Templates
      </button>
      <button
        onclick={() => (activeTab = 'new_journey')}
        class="px-3 py-1.5 text-sm font-medium rounded-lg transition-colors {activeTab === 'new_journey' ? 'bg-primary text-primary-foreground' : 'bg-muted/50 hover:bg-muted text-muted-foreground'}"
      >
        + Launch Journey
      </button>
    </div>
  </div>

  {#if statusMessage}
    <div class="p-3 rounded-lg border border-primary/30 bg-primary/10 text-xs text-foreground flex items-center justify-between">
      <span>{statusMessage}</span>
      <button onclick={() => (statusMessage = null)} class="text-xs text-muted-foreground hover:text-foreground">✕</button>
    </div>
  {/if}

  <!-- TAB: Active Journeys -->
  {#if activeTab === 'instances'}
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- Left List of Instances -->
      <div class="space-y-3">
        <h2 class="text-sm font-semibold text-muted-foreground uppercase tracking-wider">In-Flight Journeys ({instances.length})</h2>
        <div class="space-y-2">
          {#each instances as inst}
            <button
              type="button"
              onclick={() => selectInstance(inst)}
              class="w-full text-left p-4 rounded-xl border transition-all {selectedInstance?.id === inst.id ? 'border-primary bg-primary/5 shadow-sm' : 'border-border bg-card hover:border-border/80'}"
            >
              <div class="flex items-center justify-between">
                <span class="text-xs font-mono font-bold text-muted-foreground">{inst.id}</span>
                <span class="text-[10px] font-semibold px-2 py-0.5 rounded-full {inst.status === 'COMPLETED' ? 'bg-emerald-500/10 text-emerald-400' : 'bg-amber-500/10 text-amber-400'}">
                  {inst.status}
                </span>
              </div>
              <h3 class="text-sm font-bold text-foreground mt-1">{inst.customerName}</h3>
              <p class="text-xs text-muted-foreground">{inst.templateName}</p>

              <div class="mt-3 pt-2 border-t border-border flex items-center justify-between text-xs font-mono">
                <span class="text-primary font-medium">Stage {inst.currentStageOrder}/{inst.totalStages}</span>
                <span class="text-muted-foreground truncate max-w-[130px]">{inst.currentStageName}</span>
              </div>
            </button>
          {/each}
        </div>
      </div>

      <!-- Right Detail & Visual Stage Machine Pipeline -->
      <div class="lg:col-span-2 space-y-6">
        {#if selectedInstance}
          <div class="rounded-xl border border-border bg-card p-6 shadow-sm space-y-6">
            <!-- Header -->
            <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2 border-b border-border pb-4">
              <div>
                <span class="text-xs font-mono text-muted-foreground">{selectedInstance.id} • {selectedInstance.businessUnit}</span>
                <h2 class="text-xl font-bold text-foreground">{selectedInstance.customerName}</h2>
                <p class="text-xs text-muted-foreground">{selectedInstance.templateName} • {selectedInstance.customerContact}</p>
              </div>
              <div class="text-right">
                <span class="text-xs text-muted-foreground">Current Stage</span>
                <div class="text-base font-bold text-primary">{selectedInstance.currentStageName}</div>
              </div>
            </div>

            <!-- Visual Horizontal Stage Progress Tracker -->
            <div>
              <h3 class="text-xs font-semibold text-muted-foreground uppercase mb-4">Stage Progress & Gate Transition</h3>
              <div class="flex flex-col md:flex-row items-stretch md:items-center gap-2 overflow-x-auto pb-2">
                {#each activeStages as stage, i}
                  {@const isCurrent = stage.stageName === selectedInstance.currentStageName}
                  {@const isPast = stage.stageOrder < selectedInstance.currentStageOrder}
                  <div class="flex-1 min-w-[140px] p-3 rounded-lg border text-xs {isCurrent ? 'border-primary bg-primary/10 shadow-sm' : isPast ? 'border-emerald-500/30 bg-emerald-500/5 text-muted-foreground' : 'border-border bg-muted/20 text-muted-foreground'}">
                    <div class="flex items-center justify-between font-mono font-bold mb-1">
                      <span>0{stage.stageOrder}</span>
                      {#if isPast}
                        <span class="text-emerald-400">✓ Done</span>
                      {:else if isCurrent}
                        <span class="text-primary animate-pulse">● Active</span>
                      {:else}
                        <span>○ Pending</span>
                      {/if}
                    </div>
                    <div class="font-semibold text-foreground truncate" title={stage.stageName}>{stage.stageName}</div>
                    <div class="text-[10px] text-muted-foreground mt-1 truncate">
                      🔔 {stage.triggerNotificationChannel}
                    </div>

                    <!-- Advance Button if it's the next stage -->
                    {#if stage.stageOrder === selectedInstance.currentStageOrder + 1}
                      <button
                        onclick={() => handleAdvanceStage(stage)}
                        class="mt-2 w-full py-1 text-[11px] font-bold rounded bg-primary text-primary-foreground hover:bg-primary/90 transition-all shadow"
                      >
                        Advance ➔
                      </button>
                    {/if}
                  </div>
                {/each}
              </div>
            </div>

            <!-- Audit Trail / Journey Logs -->
            <div>
              <h3 class="text-xs font-semibold text-muted-foreground uppercase mb-3">Journey Audit & Transition History</h3>
              <div class="space-y-2 max-h-60 overflow-y-auto">
                {#each activeLogs as log}
                  <div class="p-3 rounded-lg border border-border bg-muted/20 text-xs flex items-start justify-between">
                    <div>
                      <div class="font-semibold text-foreground">
                        {log.fromStage} ➔ <span class="text-primary">{log.toStage}</span>
                      </div>
                      <div class="text-muted-foreground mt-0.5">{log.notes}</div>
                      <div class="text-[10px] text-muted-foreground/80 mt-1">Logged by: {log.actorName}</div>
                    </div>
                    <span class="font-mono text-[11px] text-muted-foreground">{log.timestamp}</span>
                  </div>
                {/each}
              </div>
            </div>
          </div>
        {:else}
          <div class="p-12 text-center text-muted-foreground border border-border rounded-xl">
            Select a journey to view lifecycle progression and stage transitions.
          </div>
        {/if}
      </div>
    </div>
  {/if}

  <!-- TAB: Journey Templates -->
  {#if activeTab === 'templates'}
    <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
      {#each templates as t}
        <div class="rounded-xl border border-border bg-card p-5 shadow-sm space-y-4">
          <div class="flex items-center justify-between">
            <span class="text-xs font-mono font-bold px-2 py-0.5 rounded bg-muted text-foreground">{t.code}</span>
            <span class="text-xs font-semibold text-primary">{t.businessUnit}</span>
          </div>
          <div>
            <h3 class="text-base font-bold text-foreground">{t.name}</h3>
            <p class="text-xs text-muted-foreground mt-1">{t.description}</p>
          </div>

          <div class="space-y-2 pt-2 border-t border-border">
            <span class="text-xs font-semibold text-muted-foreground">Configured Stages ({t.stages?.length || 0}):</span>
            <div class="space-y-1">
              {#each t.stages || [] as stg}
                <div class="text-xs text-muted-foreground flex items-center justify-between p-1.5 rounded bg-muted/30">
                  <span>{stg.stageOrder}. {stg.stageName}</span>
                  <span class="font-mono text-[10px] px-1.5 py-0.5 rounded bg-muted text-foreground">{stg.triggerNotificationChannel}</span>
                </div>
              {/each}
            </div>
          </div>
        </div>
      {/each}
    </div>
  {/if}

  <!-- TAB: Launch Journey Form -->
  {#if activeTab === 'new_journey'}
    <div class="rounded-xl border border-border bg-card p-6 shadow-sm max-w-xl mx-auto">
      <h2 class="text-lg font-semibold text-foreground mb-1">Launch New Customer Journey</h2>
      <p class="text-xs text-muted-foreground mb-6">Attach a prospect or client to a deterministic multi-stage workflow</p>

      <form onsubmit={handleStartJourney} class="space-y-4">
        <div>
          <label for="template-select" class="block text-xs font-medium text-muted-foreground mb-1">Select Journey Template</label>
          <select
            id="template-select"
            bind:value={newJourney.templateId}
            class="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground text-sm focus:outline-none focus:ring-1 focus:ring-primary"
          >
            {#each templates as t}
              <option value={t.id}>[{t.businessUnit}] {t.name}</option>
            {/each}
          </select>
        </div>

        <div>
          <label for="cust-name" class="block text-xs font-medium text-muted-foreground mb-1">Customer / Patient / Client Name</label>
          <input
            id="cust-name"
            type="text"
            required
            placeholder="e.g. Anand Mahindra"
            bind:value={newJourney.customerName}
            class="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground text-sm focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>

        <div>
          <label for="cust-contact" class="block text-xs font-medium text-muted-foreground mb-1">Primary WhatsApp / Phone / Email</label>
          <input
            id="cust-contact"
            type="text"
            required
            placeholder="+91 98400 12345"
            bind:value={newJourney.customerContact}
            class="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground text-sm focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>

        <div>
          <label for="initial-notes" class="block text-xs font-medium text-muted-foreground mb-1">Intake Notes</label>
          <textarea
            id="initial-notes"
            rows="3"
            placeholder="Special requirements, referral source, or initial medical/travel/real-estate briefs"
            bind:value={newJourney.initialNotes}
            class="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground text-sm focus:outline-none focus:ring-1 focus:ring-primary"
          ></textarea>
        </div>

        <div class="pt-4 flex justify-end gap-3">
          <button
            type="button"
            onclick={() => (activeTab = 'instances')}
            class="px-4 py-2 rounded-lg border border-border bg-muted/50 hover:bg-muted text-muted-foreground text-sm font-medium"
          >
            Cancel
          </button>
          <button
            type="submit"
            class="px-5 py-2 rounded-lg bg-primary hover:bg-primary/90 text-primary-foreground text-sm font-medium shadow"
          >
            Launch Journey
          </button>
        </div>
      </form>
    </div>
  {/if}
</div>
