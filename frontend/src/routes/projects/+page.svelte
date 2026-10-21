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
    priority: string;
    status: string;
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
    status: string;
  }

  let projects = $state<Project[]>([]);
  let tasks = $state<UTMSTask[]>([]);
  let milestones = $state<Milestone[]>([]);
  let selectedProjectId = $state<string>('');
  let loading = $state<boolean>(true);
  let activeTab = $state<'kanban' | 'projects' | 'utms_delegation' | 'timesheet'>('kanban');
  let statusMessage = $state<string | null>(null);

  // Form for New UTMS Task
  let newTask = $state({
    title: '',
    description: '',
    businessUnit: 'CONSULTING',
    assignedTo: '',
    priority: 'HIGH',
    dueDate: '2026-10-30',
    estimatedHours: 8
  });

  // Delegation Modal
  let showDelegateModal = $state(false);
  let taskToDelegate = $state<UTMSTask | null>(null);
  let targetBU = $state('HEALTH');
  let targetAssignee = $state('Dr. Ayya Senior Consultant');
  let delegationNote = $state('');

  const columns = ['BACKLOG', 'TODO', 'IN_PROGRESS', 'REVIEW', 'DONE'];

  async function loadData() {
    loading = true;
    try {
      const [pRes, tRes] = await Promise.all([
        projectGrpc.listProjects(),
        projectGrpc.listTasks()
      ]);
      if (pRes && pRes.projects) {
        projects = pRes.projects;
        if (projects.length > 0 && !selectedProjectId) {
          selectedProjectId = projects[0].id;
        }
      }
      if (tRes && tRes.tasks) {
        tasks = tRes.tasks;
      }
    } catch (e: any) {
      console.warn('Fallback sample data if backend is offline:', e);
    } finally {
      loading = false;
    }
  }

  async function handleMoveStatus(task: UTMSTask, newStatus: string) {
    try {
      await projectGrpc.updateTaskStatus({ taskId: task.id, status: newStatus });
      task.status = newStatus;
      statusMessage = `Task ${task.id} moved to ${newStatus}`;
    } catch (e: any) {
      task.status = newStatus;
    }
  }

  async function handleCreateTask(e: Event) {
    e.preventDefault();
    try {
      const res = await projectGrpc.createTask({
        projectId: selectedProjectId,
        ...newTask
      });
      tasks = [...tasks, res.task || {
        id: `TSK-${Date.now() % 1000}`,
        projectId: selectedProjectId,
        ...newTask,
        status: 'TODO',
        slaHoursRemaining: 48,
        entityResource: 'project',
        entityId: selectedProjectId,
        actualHours: 0
      }];
      statusMessage = `✅ Created UTMS Task: ${newTask.title}`;
      newTask.title = '';
      newTask.description = '';
    } catch (err: any) {
      statusMessage = `❌ Error: ${err.message || 'Failed to create task'}`;
    }
  }

  async function handleExecuteDelegation() {
    if (!taskToDelegate) return;
    try {
      await projectGrpc.delegateTask({
        taskId: taskToDelegate.id,
        targetBusinessUnit: targetBU,
        newAssignee: targetAssignee,
        delegationNote
      });
      taskToDelegate.businessUnit = targetBU;
      taskToDelegate.assignedTo = targetAssignee;
      statusMessage = `⚡ Cross-BU Delegated ${taskToDelegate.id} to ${targetBU} (${targetAssignee})`;
      showDelegateModal = false;
      taskToDelegate = null;
      delegationNote = '';
    } catch (err: any) {
      statusMessage = `❌ Delegation failed: ${err.message}`;
    }
  }

  onMount(() => {
    loadData();
  });
</script>

<svelte:head>
  <title>Projects & UTMS | MrBiz-OS (Odoo Killer)</title>
</svelte:head>

<div class="space-y-6">
  <!-- Top Bar -->
  <div class="flex flex-col md:flex-row md:items-center justify-between gap-4 border-b border-border pb-5">
    <div>
      <div class="flex items-center gap-2">
        <span class="px-2 py-0.5 text-xs font-semibold rounded bg-sky-500/10 text-sky-400 border border-sky-500/20">
          Module: Projects & UTMS
        </span>
        <span class="px-2 py-0.5 text-xs font-semibold rounded bg-indigo-500/10 text-indigo-400 border border-indigo-500/20">
          Cross-Department SLA Engine
        </span>
      </div>
      <h1 class="text-2xl font-bold tracking-tight text-foreground mt-1">Enterprise Project & Task Management</h1>
      <p class="text-sm text-muted-foreground">
        Unified Task Management System (UTMS) replacing Odoo Projects with cross-business unit task routing and SLA enforcement.
      </p>
    </div>

    <div class="flex items-center gap-2">
      <button
        onclick={() => (activeTab = 'kanban')}
        class="px-3 py-1.5 text-sm font-medium rounded-lg transition-colors {activeTab === 'kanban' ? 'bg-primary text-primary-foreground' : 'bg-muted/50 hover:bg-muted text-muted-foreground'}"
      >
        Kanban Pipeline
      </button>
      <button
        onclick={() => (activeTab = 'projects')}
        class="px-3 py-1.5 text-sm font-medium rounded-lg transition-colors {activeTab === 'projects' ? 'bg-primary text-primary-foreground' : 'bg-muted/50 hover:bg-muted text-muted-foreground'}"
      >
        Projects Register
      </button>
      <button
        onclick={() => (activeTab = 'utms_delegation')}
        class="px-3 py-1.5 text-sm font-medium rounded-lg transition-colors {activeTab === 'utms_delegation' ? 'bg-primary text-primary-foreground' : 'bg-muted/50 hover:bg-muted text-muted-foreground'}"
      >
        + New UTMS Task
      </button>
    </div>
  </div>

  {#if statusMessage}
    <div class="p-3 rounded-lg border border-primary/30 bg-primary/10 text-xs text-foreground flex items-center justify-between">
      <span>{statusMessage}</span>
      <button onclick={() => (statusMessage = null)} class="text-xs text-muted-foreground hover:text-foreground">✕</button>
    </div>
  {/if}

  <!-- TAB: Kanban Pipeline -->
  {#if activeTab === 'kanban'}
    <div class="space-y-4">
      <!-- Project Filter -->
      <div class="flex items-center justify-between bg-card p-4 rounded-xl border border-border">
        <div class="flex items-center gap-3">
          <label for="project-select" class="text-xs font-semibold text-muted-foreground uppercase">Project Filter:</label>
          <select
            id="project-select"
            bind:value={selectedProjectId}
            class="px-3 py-1.5 rounded-lg border border-border bg-background text-sm font-medium text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
          >
            <option value="">All Business Units & Projects</option>
            {#each projects as p}
              <option value={p.id}>[{p.businessUnit}] {p.name}</option>
            {/each}
          </select>
        </div>

        <div class="text-xs text-muted-foreground font-mono">
          Total UTMS Tasks: {tasks.length}
        </div>
      </div>

      <!-- Kanban Columns -->
      <div class="grid grid-cols-1 md:grid-cols-5 gap-4">
        {#each columns as col}
          {@const colTasks = tasks.filter(t => (t.status === col || (col === 'TODO' && !t.status)) && (!selectedProjectId || t.projectId === selectedProjectId))}
          <div class="rounded-xl border border-border bg-card/60 flex flex-col min-h-[500px]">
            <!-- Column Header -->
            <div class="p-3 border-b border-border flex items-center justify-between bg-muted/20 rounded-t-xl">
              <span class="text-xs font-bold text-foreground uppercase tracking-wider">{col.replace('_', ' ')}</span>
              <span class="text-xs px-2 py-0.5 rounded-full bg-muted text-muted-foreground font-mono">{colTasks.length}</span>
            </div>

            <!-- Task Cards -->
            <div class="p-2 space-y-3 flex-1 overflow-y-auto">
              {#each colTasks as task}
                <div class="p-3 rounded-lg border border-border bg-card shadow-sm hover:border-primary/50 transition-all space-y-2">
                  <div class="flex items-center justify-between">
                    <span class="text-[10px] font-bold px-1.5 py-0.5 rounded bg-muted text-muted-foreground font-mono">
                      {task.id}
                    </span>
                    <span class="text-[10px] font-semibold px-1.5 py-0.5 rounded {task.priority === 'URGENT' ? 'bg-red-500/10 text-red-400 border border-red-500/20' : task.priority === 'HIGH' ? 'bg-amber-500/10 text-amber-400 border border-amber-500/20' : 'bg-muted text-muted-foreground'}">
                      {task.priority}
                    </span>
                  </div>

                  <h3 class="text-xs font-semibold text-foreground line-clamp-2">{task.title}</h3>
                  <p class="text-[11px] text-muted-foreground line-clamp-2">{task.description}</p>

                  <div class="pt-2 border-t border-border flex items-center justify-between text-[11px] text-muted-foreground">
                    <span class="truncate max-w-[110px]" title={task.assignedTo}>👤 {task.assignedTo.split(' ')[0]}</span>
                    <span class="font-mono text-emerald-400">⏱️ {task.slaHoursRemaining}h SLA</span>
                  </div>

                  <!-- Quick Action Buttons -->
                  <div class="pt-1 flex items-center justify-between gap-1">
                    <button
                      onclick={() => {
                        taskToDelegate = task;
                        showDelegateModal = true;
                      }}
                      class="text-[10px] px-2 py-1 rounded bg-sky-500/10 hover:bg-sky-500/20 text-sky-400 transition-colors"
                      title="Delegate cross-department"
                    >
                      🔁 Delegate
                    </button>

                    <div class="flex items-center gap-1">
                      {#if col !== 'BACKLOG'}
                        <button
                          onclick={() => handleMoveStatus(task, columns[columns.indexOf(col) - 1])}
                          class="text-[10px] px-1.5 py-0.5 rounded bg-muted hover:bg-muted/80 text-muted-foreground"
                          title="Move Back"
                        >
                          ◀
                        </button>
                      {/if}
                      {#if col !== 'DONE'}
                        <button
                          onclick={() => handleMoveStatus(task, columns[columns.indexOf(col) + 1])}
                          class="text-[10px] px-1.5 py-0.5 rounded bg-muted hover:bg-muted/80 text-muted-foreground"
                          title="Move Forward"
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

  <!-- TAB: Projects Register -->
  {#if activeTab === 'projects'}
    <div class="rounded-xl border border-border bg-card p-6 shadow-sm">
      <h2 class="text-lg font-semibold text-foreground mb-1">Active Projects Portfolio</h2>
      <p class="text-xs text-muted-foreground mb-6">Cross-business unit projects with milestone billing and budget tracking</p>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        {#each projects as p}
          <div class="p-5 rounded-xl border border-border bg-muted/10 space-y-3">
            <div class="flex items-center justify-between">
              <span class="text-xs font-mono font-bold px-2 py-0.5 rounded bg-muted text-foreground">{p.code}</span>
              <span class="text-xs font-semibold px-2 py-0.5 rounded bg-primary/10 text-primary border border-primary/20">{p.status}</span>
            </div>
            <div>
              <h3 class="text-base font-bold text-foreground">{p.name}</h3>
              <p class="text-xs text-muted-foreground mt-0.5">{p.description}</p>
            </div>
            <div class="grid grid-cols-2 gap-2 text-xs font-mono pt-2 border-t border-border">
              <div>
                <span class="text-muted-foreground">Client:</span> <strong class="text-foreground">{p.clientName}</strong>
              </div>
              <div>
                <span class="text-muted-foreground">Budget:</span> <strong class="text-emerald-400">₹{p.budget.toLocaleString('en-IN')}</strong>
              </div>
              <div>
                <span class="text-muted-foreground">Unit:</span> <strong class="text-foreground">{p.businessUnit}</strong>
              </div>
              <div>
                <span class="text-muted-foreground">Target Date:</span> <strong class="text-foreground">{p.targetDate}</strong>
              </div>
            </div>
          </div>
        {/each}
      </div>
    </div>
  {/if}

  <!-- TAB: New UTMS Task Form -->
  {#if activeTab === 'utms_delegation'}
    <div class="rounded-xl border border-border bg-card p-6 shadow-sm max-w-2xl mx-auto">
      <h2 class="text-lg font-semibold text-foreground mb-1">Create Universal Task (UTMS)</h2>
      <p class="text-xs text-muted-foreground mb-6">Assign deliverables across Healthcare, Travel, Real Estate, or Finance</p>

      <form onsubmit={handleCreateTask} class="space-y-4">
        <div>
          <label for="task-title" class="block text-xs font-medium text-muted-foreground mb-1">Task Title</label>
          <input
            id="task-title"
            type="text"
            required
            placeholder="e.g. Verify Schengen Visa bank statements"
            bind:value={newTask.title}
            class="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground text-sm focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>

        <div>
          <label for="task-desc" class="block text-xs font-medium text-muted-foreground mb-1">Description & Requirements</label>
          <textarea
            id="task-desc"
            rows="3"
            placeholder="Detail the deliverable, acceptance criteria, and related customer entity"
            bind:value={newTask.description}
            class="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground text-sm focus:outline-none focus:ring-1 focus:ring-primary"
          ></textarea>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label for="task-bu" class="block text-xs font-medium text-muted-foreground mb-1">Business Unit</label>
            <select
              id="task-bu"
              bind:value={newTask.businessUnit}
              class="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground text-sm focus:outline-none focus:ring-1 focus:ring-primary"
            >
              <option value="CONSULTING">Consulting & Software</option>
              <option value="HEALTH">Healthcare & Clinics</option>
              <option value="TRAVEL">Travel & Visa OS</option>
              <option value="REALESTATE">Real Estate Deals</option>
              <option value="FINANCE">Finance & Accounting</option>
            </select>
          </div>
          <div>
            <label for="task-assignee" class="block text-xs font-medium text-muted-foreground mb-1">Assigned Specialist</label>
            <input
              id="task-assignee"
              type="text"
              required
              placeholder="e.g. Arun Kumar"
              bind:value={newTask.assignedTo}
              class="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground text-sm focus:outline-none focus:ring-1 focus:ring-primary"
            />
          </div>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <div>
            <label for="task-priority" class="block text-xs font-medium text-muted-foreground mb-1">Priority</label>
            <select
              id="task-priority"
              bind:value={newTask.priority}
              class="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground text-sm focus:outline-none focus:ring-1 focus:ring-primary"
            >
              <option value="LOW">Low</option>
              <option value="MEDIUM">Medium</option>
              <option value="HIGH">High</option>
              <option value="URGENT">Urgent (SLA Trigger)</option>
            </select>
          </div>
          <div>
            <label for="task-due" class="block text-xs font-medium text-muted-foreground mb-1">Due Date</label>
            <input
              id="task-due"
              type="date"
              bind:value={newTask.dueDate}
              class="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground text-sm focus:outline-none focus:ring-1 focus:ring-primary"
            />
          </div>
          <div>
            <label for="task-hours" class="block text-xs font-medium text-muted-foreground mb-1">Est. Hours</label>
            <input
              id="task-hours"
              type="number"
              min="1"
              bind:value={newTask.estimatedHours}
              class="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground text-sm font-mono focus:outline-none focus:ring-1 focus:ring-primary"
            />
          </div>
        </div>

        <div class="pt-4 flex justify-end gap-3">
          <button
            type="button"
            onclick={() => (activeTab = 'kanban')}
            class="px-4 py-2 rounded-lg border border-border bg-muted/50 hover:bg-muted text-muted-foreground text-sm font-medium"
          >
            Cancel
          </button>
          <button
            type="submit"
            class="px-5 py-2 rounded-lg bg-primary hover:bg-primary/90 text-primary-foreground text-sm font-medium shadow"
          >
            Save & Publish Task
          </button>
        </div>
      </form>
    </div>
  {/if}

  <!-- DELEGATION MODAL -->
  {#if showDelegateModal && taskToDelegate}
    <div class="fixed inset-0 bg-background/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
      <div class="rounded-xl border border-border bg-card p-6 max-w-md w-full shadow-2xl space-y-4">
        <div>
          <span class="text-xs px-2 py-0.5 rounded bg-sky-500/10 text-sky-400 font-bold">UTMS Cross-BU Routing</span>
          <h3 class="text-lg font-bold text-foreground mt-1">Delegate Task: {taskToDelegate.id}</h3>
          <p class="text-xs text-muted-foreground">{taskToDelegate.title}</p>
        </div>

        <div class="space-y-3">
          <div>
            <label for="target-bu" class="block text-xs font-medium text-muted-foreground mb-1">Target Department / BU</label>
            <select
              id="target-bu"
              bind:value={targetBU}
              class="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground text-sm focus:outline-none focus:ring-1 focus:ring-primary"
            >
              <option value="HEALTH">🏥 Smart Healthcare</option>
              <option value="TRAVEL">✈️ Travel & Visa OS</option>
              <option value="REALESTATE">🏢 Real Estate Deals</option>
              <option value="CONSULTING">💻 Software Consulting</option>
              <option value="FINANCE">💳 Finance & Accounting</option>
            </select>
          </div>

          <div>
            <label for="target-person" class="block text-xs font-medium text-muted-foreground mb-1">New Assignee Name</label>
            <input
              id="target-person"
              type="text"
              bind:value={targetAssignee}
              class="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground text-sm focus:outline-none focus:ring-1 focus:ring-primary"
            />
          </div>

          <div>
            <label for="delegation-note" class="block text-xs font-medium text-muted-foreground mb-1">Handover Instructions</label>
            <textarea
              id="delegation-note"
              rows="2"
              placeholder="e.g. Please verify compliance and contact customer"
              bind:value={delegationNote}
              class="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground text-sm focus:outline-none focus:ring-1 focus:ring-primary"
            ></textarea>
          </div>
        </div>

        <div class="flex justify-end gap-2 pt-2">
          <button
            onclick={() => (showDelegateModal = false)}
            class="px-4 py-2 rounded-lg border border-border text-sm text-muted-foreground hover:bg-muted"
          >
            Cancel
          </button>
          <button
            onclick={handleExecuteDelegation}
            class="px-4 py-2 rounded-lg bg-primary hover:bg-primary/90 text-primary-foreground text-sm font-semibold"
          >
            Confirm Delegation
          </button>
        </div>
      </div>
    </div>
  {/if}
</div>
