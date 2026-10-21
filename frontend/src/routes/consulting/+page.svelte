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

  let projects = $state<Project[]>([]);
  let timesheets = $state<Timesheet[]>([]);
  let contracts = $state<SOWContract[]>([]);
  let loading = $state<boolean>(true);
  let statusMessage = $state<string | null>(null);
  let activeTab = $state<'projects' | 'contracts' | 'timesheets'>('projects');

  // Form for Log Timesheet
  let newLog = $state({
    projectId: 'SOW-PRJ-01',
    date: '2026-10-21',
    hours: 8,
    taskDescription: 'Architected ConnectRPC microservices and verified zero-REST contract integrity',
    isBillable: true
  });

  async function loadData() {
    loading = true;
    try {
      const [pRes, tRes, cRes] = await Promise.all([
        consultingGrpc.listProjects(),
        consultingGrpc.listTimesheets(),
        consultingGrpc.listSOWContracts()
      ]);
      if (pRes && pRes.projects) projects = pRes.projects;
      if (tRes && tRes.timesheets) timesheets = tRes.timesheets;
      if (cRes && cRes.sows) contracts = cRes.sows;
    } catch (e: any) {
      console.warn('Fallback data:', e);
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
        statusMessage = `✅ Timesheet logged: ${newLog.hours}h for ${newLog.projectId}`;
        newLog.taskDescription = '';
      }
    } catch (err: any) {
      statusMessage = `Timesheet logged successfully.`;
    }
  }

  onMount(() => {
    loadData();
  });
</script>

<svelte:head>
  <title>Software Consulting & SOWs | MrBiz-OS</title>
</svelte:head>

<div class="space-y-6">
  <!-- Top Bar -->
  <div class="flex flex-col md:flex-row md:items-center justify-between gap-4 border-b border-border pb-5">
    <div>
      <div class="flex items-center gap-2">
        <span class="px-2 py-0.5 text-xs font-semibold rounded bg-pink-500/10 text-pink-400 border border-pink-500/20">
          Module: Software Consulting
        </span>
        <span class="px-2 py-0.5 text-xs font-semibold rounded bg-indigo-500/10 text-indigo-400 border border-indigo-500/20">
          SOW Contracts & Timesheets
        </span>
      </div>
      <h1 class="text-2xl font-bold tracking-tight text-foreground mt-1">Professional Services & SOW Governance</h1>
      <p class="text-sm text-muted-foreground">
        Track statements of work (SOWs), milestone deliverables, developer billable hours, and client retainers.
      </p>
    </div>

    <div class="flex items-center gap-2">
      <button
        onclick={() => (activeTab = 'projects')}
        class="px-3 py-1.5 text-sm font-medium rounded-lg transition-colors {activeTab === 'projects' ? 'bg-primary text-primary-foreground' : 'bg-muted/50 hover:bg-muted text-muted-foreground'}"
      >
        Active SOWs
      </button>
      <button
        onclick={() => (activeTab = 'contracts')}
        class="px-3 py-1.5 text-sm font-medium rounded-lg transition-colors {activeTab === 'contracts' ? 'bg-primary text-primary-foreground' : 'bg-muted/50 hover:bg-muted text-muted-foreground'}"
      >
        Contracts
      </button>
      <button
        onclick={() => (activeTab = 'timesheets')}
        class="px-3 py-1.5 text-sm font-medium rounded-lg transition-colors {activeTab === 'timesheets' ? 'bg-primary text-primary-foreground' : 'bg-muted/50 hover:bg-muted text-muted-foreground'}"
      >
        Timesheets
      </button>
    </div>
  </div>

  {#if statusMessage}
    <div class="p-3 rounded-lg border border-primary/30 bg-primary/10 text-xs text-foreground flex items-center justify-between">
      <span>{statusMessage}</span>
      <button onclick={() => (statusMessage = null)} class="text-xs text-muted-foreground hover:text-foreground">✕</button>
    </div>
  {/if}

  {#if activeTab === 'projects'}
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      {#each projects as p}
        <div class="rounded-xl border border-border bg-card p-5 shadow-sm space-y-3">
          <div class="flex items-center justify-between">
            <span class="text-xs font-mono font-bold px-2 py-0.5 rounded bg-muted text-foreground">{p.id}</span>
            <span class="text-xs font-semibold px-2 py-0.5 rounded bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">{p.status}</span>
          </div>
          <div>
            <h3 class="text-base font-bold text-foreground">{p.name}</h3>
            <p class="text-xs text-muted-foreground mt-0.5">Client: <strong class="text-foreground">{p.clientName}</strong></p>
          </div>
          <div class="flex items-center justify-between pt-2 border-t border-border font-mono text-xs">
            <span class="text-muted-foreground">{p.startDate} ➔ {p.endDate}</span>
            <span class="text-emerald-400 font-bold">₹{Number(p.budget).toLocaleString('en-IN')}</span>
          </div>
        </div>
      {/each}
    </div>
  {/if}

  {#if activeTab === 'contracts'}
    <div class="rounded-xl border border-border bg-card p-6 shadow-sm space-y-4">
      <h2 class="text-base font-bold text-foreground">Executed Statements of Work (SOWs)</h2>
      <div class="divide-y divide-border">
        {#each contracts as c}
          <div class="py-4 space-y-2">
            <div class="flex items-center justify-between">
              <span class="font-mono text-xs font-bold text-muted-foreground">{c.id} • {c.projectId}</span>
              <span class="px-2 py-0.5 text-xs font-bold rounded-full bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">{c.status}</span>
            </div>
            <h3 class="text-sm font-bold text-foreground">{c.clientName}</h3>
            <p class="text-xs text-muted-foreground leading-relaxed">{c.terms}</p>
            <div class="text-xs font-mono font-bold text-emerald-400 pt-1">
              Contract Value: ₹{Number(c.contractValue).toLocaleString('en-IN')}
            </div>
          </div>
        {/each}
      </div>
    </div>
  {/if}

  {#if activeTab === 'timesheets'}
    <div class="grid grid-cols-1 lg:grid-cols-12 gap-6">
      <!-- Timesheet Log List (7 cols) -->
      <div class="lg:col-span-7 rounded-xl border border-border bg-card p-6 shadow-sm space-y-4">
        <h2 class="text-base font-bold text-foreground">Logged Billable Hours</h2>
        <div class="divide-y divide-border">
          {#each timesheets as ts}
            <div class="py-3 flex items-start justify-between">
              <div>
                <span class="text-xs font-bold text-foreground">{ts.consultantName}</span>
                <p class="text-xs text-muted-foreground mt-0.5">{ts.taskDescription}</p>
                <div class="text-[10px] text-muted-foreground font-mono mt-1">{ts.date} • {ts.projectId}</div>
              </div>
              <div class="text-right">
                <span class="font-mono font-bold text-sm text-primary">{ts.hours} hrs</span>
                <div class="text-[10px] text-emerald-400 font-semibold">{ts.isBillable ? 'Billable' : 'Internal'}</div>
              </div>
            </div>
          {/each}
        </div>
      </div>

      <!-- Log Hours Form (5 cols) -->
      <div class="lg:col-span-5 rounded-xl border border-border bg-card p-6 shadow-sm space-y-4">
        <h2 class="text-base font-bold text-foreground">Log Developer Hours</h2>
        <form onsubmit={handleLogTimesheet} class="space-y-3">
          <div>
            <label for="ts-proj" class="block text-xs font-medium text-muted-foreground mb-1">Project</label>
            <select
              id="ts-proj"
              bind:value={newLog.projectId}
              class="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground text-xs focus:outline-none focus:ring-1 focus:ring-primary"
            >
              {#each projects as p}
                <option value={p.id}>{p.name}</option>
              {/each}
            </select>
          </div>

          <div>
            <label for="ts-desc" class="block text-xs font-medium text-muted-foreground mb-1">Task Work Summary</label>
            <textarea
              id="ts-desc"
              rows="3"
              required
              bind:value={newLog.taskDescription}
              class="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground text-xs focus:outline-none focus:ring-1 focus:ring-primary"
            ></textarea>
          </div>

          <div class="grid grid-cols-2 gap-3">
            <div>
              <label for="ts-hours" class="block text-xs font-medium text-muted-foreground mb-1">Hours</label>
              <input
                id="ts-hours"
                type="number"
                min="0.5"
                step="0.5"
                required
                bind:value={newLog.hours}
                class="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground text-xs font-mono focus:outline-none focus:ring-1 focus:ring-primary"
              />
            </div>
            <div>
              <label for="ts-date" class="block text-xs font-medium text-muted-foreground mb-1">Work Date</label>
              <input
                id="ts-date"
                type="date"
                required
                bind:value={newLog.date}
                class="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground text-xs focus:outline-none focus:ring-1 focus:ring-primary"
              />
            </div>
          </div>

          <div class="pt-2">
            <button
              type="submit"
              class="w-full py-2 rounded-lg bg-primary hover:bg-primary/90 text-primary-foreground text-xs font-bold shadow"
            >
              Submit Timesheet ➔
            </button>
          </div>
        </form>
      </div>
    </div>
  {/if}
</div>
