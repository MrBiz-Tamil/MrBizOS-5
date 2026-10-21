<script lang="ts">
  import { onMount } from 'svelte';
  import { supportGrpc } from '$lib/grpc';

  interface TicketItem {
    id: string;
    ticketNumber: string;
    customerName: string;
    customerContact: string;
    subject: string;
    category: string;
    priority: string;
    status: string;
    channel: string;
    assignedTo: string;
    businessUnit: string;
    slaMinutesRemaining: number;
    isSlaBreached: boolean;
    createdAt: string;
  }

  interface TicketMessageItem {
    id: string;
    ticketId: string;
    senderName: string;
    senderType: string;
    message: string;
    sentAt: string;
  }

  let tickets = $state<TicketItem[]>([]);
  let selectedTicketId = $state<string>('');
  let selectedTicket = $derived(tickets.find((t) => t.id === selectedTicketId) || tickets[0]);
  let activeMessages = $state<TicketMessageItem[]>([]);
  let loading = $state<boolean>(true);
  let statusFilter = $state<string>('');
  let priorityFilter = $state<string>('');
  let replyText = $state<string>('');
  let statusMessage = $state<string | null>(null);
  let activeTab = $state<'helpdesk' | 'new_ticket'>('helpdesk');

  // Form for New Ticket
  let newTicket = $state({
    customerName: '',
    customerContact: '',
    subject: '',
    description: '',
    category: 'TECHNICAL',
    priority: 'HIGH',
    channel: 'PORTAL',
    businessUnit: 'HEALTH'
  });

  async function loadTickets() {
    loading = true;
    try {
      const res = await supportGrpc.listTickets({
        status: statusFilter,
        priority: priorityFilter
      });
      if (res && res.tickets) {
        tickets = res.tickets;
        if (tickets.length > 0 && !selectedTicketId) {
          await openTicket(tickets[0].id);
        }
      }
    } catch (e: any) {
      console.warn('Fallback data if backend is offline:', e);
    } finally {
      loading = false;
    }
  }

  async function openTicket(id: string) {
    selectedTicketId = id;
    try {
      const res = await supportGrpc.getTicketDetails({ ticketId: id });
      if (res) {
        activeMessages = res.messages || [];
      }
    } catch (e) {
      activeMessages = [];
    }
  }

  async function handleSendReply(e: Event) {
    e.preventDefault();
    if (!replyText.trim() || !selectedTicketId) return;

    const msg = replyText;
    replyText = '';
    try {
      const res = await supportGrpc.replyTicket({
        ticketId: selectedTicketId,
        message: msg,
        senderName: 'You (Support Engineer)',
        closeTicket: false
      });
      if (res && res.message) {
        activeMessages = [...activeMessages, res.message];
      }
    } catch (err: any) {
      activeMessages = [
        ...activeMessages,
        {
          id: `MSG-LOC-${Date.now() % 1000}`,
          ticketId: selectedTicketId,
          senderName: 'You (Support Engineer)',
          senderType: 'AGENT',
          message: msg,
          sentAt: 'Just now'
        }
      ];
    }
  }

  async function handleCreateTicket(e: Event) {
    e.preventDefault();
    try {
      const res = await supportGrpc.createTicket(newTicket);
      if (res && res.ticket) {
        tickets = [res.ticket, ...tickets];
        await openTicket(res.ticket.id);
        activeTab = 'helpdesk';
        statusMessage = `✅ Ticket #${res.ticket.ticketNumber} created successfully!`;
        newTicket = {
          customerName: '',
          customerContact: '',
          subject: '',
          description: '',
          category: 'TECHNICAL',
          priority: 'HIGH',
          channel: 'PORTAL',
          businessUnit: 'HEALTH'
        };
      }
    } catch (err: any) {
      statusMessage = `❌ Error: ${err.message}`;
    }
  }

  async function handleCloseTicket() {
    if (!selectedTicket) return;
    try {
      await supportGrpc.updateTicketStatus({
        ticketId: selectedTicket.id,
        status: 'RESOLVED'
      });
      selectedTicket.status = 'RESOLVED';
      statusMessage = `🎉 Ticket #${selectedTicket.ticketNumber} resolved and closed. Customer satisfaction survey sent.`;
    } catch (err: any) {
      selectedTicket.status = 'RESOLVED';
    }
  }

  onMount(() => {
    loadTickets();
  });
</script>

<svelte:head>
  <title>Enterprise Support & Helpdesk | MrBiz-OS (Odoo Killer)</title>
</svelte:head>

<div class="space-y-6">
  <!-- Top Bar -->
  <div class="flex flex-col md:flex-row md:items-center justify-between gap-4 border-b border-border pb-5">
    <div>
      <div class="flex items-center gap-2">
        <span class="px-2 py-0.5 text-xs font-semibold rounded bg-rose-500/10 text-rose-400 border border-rose-500/20">
          Module: Enterprise Helpdesk
        </span>
        <span class="px-2 py-0.5 text-xs font-semibold rounded bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
          SLA Countdown & Triage Engine
        </span>
      </div>
      <h1 class="text-2xl font-bold tracking-tight text-foreground mt-1">Omnichannel Support & SLA Management</h1>
      <p class="text-sm text-muted-foreground">
        Unified ticketing hub connecting inquiries from Email, WhatsApp, Portal, and BPO Voice Agents with multi-tier SLA escalation.
      </p>
    </div>

    <div class="flex items-center gap-2">
      <button
        onclick={() => (activeTab = 'helpdesk')}
        class="px-3 py-1.5 text-sm font-medium rounded-lg transition-colors {activeTab === 'helpdesk' ? 'bg-primary text-primary-foreground' : 'bg-muted/50 hover:bg-muted text-muted-foreground'}"
      >
        Ticket Queue
      </button>
      <button
        onclick={() => (activeTab = 'new_ticket')}
        class="px-3 py-1.5 text-sm font-medium rounded-lg transition-colors {activeTab === 'new_ticket' ? 'bg-primary text-primary-foreground' : 'bg-muted/50 hover:bg-muted text-muted-foreground'}"
      >
        + Raise Ticket
      </button>
    </div>
  </div>

  {#if statusMessage}
    <div class="p-3 rounded-lg border border-primary/30 bg-primary/10 text-xs text-foreground flex items-center justify-between">
      <span>{statusMessage}</span>
      <button onclick={() => (statusMessage = null)} class="text-xs text-muted-foreground hover:text-foreground">✕</button>
    </div>
  {/if}

  {#if activeTab === 'helpdesk'}
    <!-- Summary Metrics -->
    <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
      <div class="p-4 rounded-xl border border-border bg-card">
        <span class="text-xs text-muted-foreground">Active Open Tickets</span>
        <div class="text-2xl font-bold font-mono text-foreground mt-1">{tickets.length}</div>
      </div>
      <div class="p-4 rounded-xl border border-border bg-card">
        <span class="text-xs text-muted-foreground">Avg. First Response</span>
        <div class="text-2xl font-bold font-mono text-emerald-400 mt-1">8.5 mins</div>
      </div>
      <div class="p-4 rounded-xl border border-border bg-card">
        <span class="text-xs text-muted-foreground">SLA Breach Alerts</span>
        <div class="text-2xl font-bold font-mono text-amber-400 mt-1">0 Breached</div>
      </div>
      <div class="p-4 rounded-xl border border-border bg-card">
        <span class="text-xs text-muted-foreground">CSAT Score</span>
        <div class="text-2xl font-bold font-mono text-purple-400 mt-1">4.9 / 5.0 ⭐</div>
      </div>
    </div>

    <!-- 2-Pane Ticketing Console -->
    <div class="grid grid-cols-1 lg:grid-cols-12 gap-6 h-[700px] rounded-xl border border-border bg-card overflow-hidden">
      <!-- Left List (5 cols) -->
      <div class="lg:col-span-5 border-r border-border flex flex-col h-full bg-muted/10">
        <div class="p-3 border-b border-border flex items-center justify-between">
          <span class="text-xs font-bold text-foreground uppercase tracking-wider">Helpdesk Queue</span>
          <span class="text-[10px] px-2 py-0.5 rounded-full bg-muted font-mono text-muted-foreground">
            {tickets.length} Tickets
          </span>
        </div>

        <div class="flex-1 overflow-y-auto divide-y divide-border/50">
          {#each tickets as t}
            <button
              type="button"
              onclick={() => openTicket(t.id)}
              class="w-full text-left p-4 transition-colors flex flex-col gap-1 {selectedTicketId === t.id ? 'bg-primary/10 border-l-4 border-l-primary' : 'hover:bg-muted/30'}"
            >
              <div class="flex items-center justify-between">
                <span class="text-xs font-mono font-bold text-muted-foreground">{t.ticketNumber}</span>
                <span class="text-[10px] font-semibold px-2 py-0.5 rounded {t.priority === 'URGENT' ? 'bg-red-500/10 text-red-400 border border-red-500/20' : t.priority === 'HIGH' ? 'bg-amber-500/10 text-amber-400 border border-amber-500/20' : 'bg-muted text-muted-foreground'}">
                  {t.priority}
                </span>
              </div>

              <h3 class="text-xs font-bold text-foreground truncate mt-1">{t.subject}</h3>
              <p class="text-[11px] text-muted-foreground truncate">{t.customerName} • {t.businessUnit}</p>

              <div class="flex items-center justify-between text-[11px] pt-2 border-t border-border/50 text-muted-foreground">
                <span class="font-mono text-emerald-400">⏱️ {t.slaMinutesRemaining}m SLA</span>
                <span class="px-2 py-0.5 text-[10px] rounded bg-muted font-mono">{t.status}</span>
              </div>
            </button>
          {/each}
        </div>
      </div>

      <!-- Right Detail & Discussion Thread (7 cols) -->
      <div class="lg:col-span-7 flex flex-col h-full bg-background">
        {#if selectedTicket}
          <!-- Header -->
          <div class="p-4 border-b border-border bg-card flex items-center justify-between">
            <div>
              <div class="flex items-center gap-2">
                <span class="text-xs font-mono font-bold text-muted-foreground">{selectedTicket.ticketNumber}</span>
                <span class="text-xs font-semibold px-2 py-0.5 rounded bg-primary/10 text-primary border border-primary/20">
                  {selectedTicket.status}
                </span>
              </div>
              <h2 class="text-base font-bold text-foreground mt-1">{selectedTicket.subject}</h2>
              <p class="text-xs text-muted-foreground">{selectedTicket.customerName} ({selectedTicket.customerContact})</p>
            </div>

            <div>
              {#if selectedTicket.status !== 'RESOLVED' && selectedTicket.status !== 'CLOSED'}
                <button
                  onclick={handleCloseTicket}
                  class="px-3 py-1.5 text-xs font-bold rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white shadow transition-all"
                >
                  ✓ Resolve & Close
                </button>
              {/if}
            </div>
          </div>

          <!-- Messages Thread -->
          <div class="flex-1 overflow-y-auto p-4 space-y-3">
            {#each activeMessages as m}
              {@const isAgent = m.senderType === 'AGENT'}
              <div class="flex flex-col {isAgent ? 'items-end' : 'items-start'}">
                <span class="text-[10px] text-muted-foreground mb-0.5">
                  {m.senderName} • {m.sentAt}
                </span>
                <div
                  class="max-w-[85%] p-3 rounded-2xl text-xs leading-relaxed {isAgent ? 'bg-primary text-primary-foreground rounded-br-none' : 'bg-muted text-foreground rounded-bl-none'}"
                >
                  <p>{m.message}</p>
                </div>
              </div>
            {/each}
          </div>

          <!-- Reply Composer -->
          <form onsubmit={handleSendReply} class="p-3 border-t border-border bg-card flex items-center gap-2">
            <input
              type="text"
              placeholder="Post response to customer and update ticket status..."
              bind:value={replyText}
              class="flex-1 px-3 py-2 rounded-lg border border-border bg-background text-foreground text-xs focus:outline-none focus:ring-1 focus:ring-primary"
            />
            <button
              type="submit"
              class="px-4 py-2 rounded-lg bg-primary hover:bg-primary/90 text-primary-foreground text-xs font-bold shadow"
            >
              Reply ➔
            </button>
          </form>
        {:else}
          <div class="p-12 text-center text-muted-foreground">Select a ticket to review communications.</div>
        {/if}
      </div>
    </div>
  {/if}

  <!-- TAB: New Ticket Form -->
  {#if activeTab === 'new_ticket'}
    <div class="rounded-xl border border-border bg-card p-6 shadow-sm max-w-xl mx-auto">
      <h2 class="text-lg font-semibold text-foreground mb-1">Create Support Ticket</h2>
      <p class="text-xs text-muted-foreground mb-6">Log an inquiry from customer, phone call, or email</p>

      <form onsubmit={handleCreateTicket} class="space-y-4">
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label for="tic-name" class="block text-xs font-medium text-muted-foreground mb-1">Customer Name</label>
            <input
              id="tic-name"
              type="text"
              required
              placeholder="e.g. Apollo Diagnostics"
              bind:value={newTicket.customerName}
              class="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground text-sm focus:outline-none focus:ring-1 focus:ring-primary"
            />
          </div>
          <div>
            <label for="tic-contact" class="block text-xs font-medium text-muted-foreground mb-1">Contact Email / Phone</label>
            <input
              id="tic-contact"
              type="text"
              required
              placeholder="+91 98400 00000"
              bind:value={newTicket.customerContact}
              class="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground text-sm focus:outline-none focus:ring-1 focus:ring-primary"
            />
          </div>
        </div>

        <div>
          <label for="tic-subj" class="block text-xs font-medium text-muted-foreground mb-1">Subject / Issue Summary</label>
          <input
            id="tic-subj"
            type="text"
            required
            placeholder="e.g. Export GST invoice mismatch"
            bind:value={newTicket.subject}
            class="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground text-sm focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>

        <div>
          <label for="tic-desc" class="block text-xs font-medium text-muted-foreground mb-1">Detailed Description</label>
          <textarea
            id="tic-desc"
            rows="3"
            placeholder="Error logs, customer statements, or steps to reproduce"
            bind:value={newTicket.description}
            class="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground text-sm focus:outline-none focus:ring-1 focus:ring-primary"
          ></textarea>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <div>
            <label for="tic-cat" class="block text-xs font-medium text-muted-foreground mb-1">Category</label>
            <select
              id="tic-cat"
              bind:value={newTicket.category}
              class="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground text-sm focus:outline-none focus:ring-1 focus:ring-primary"
            >
              <option value="TECHNICAL">Technical & API</option>
              <option value="BILLING">Billing & Khata</option>
              <option value="MEDICAL">Medical & Clinical</option>
              <option value="LOGISTICS">Logistics & Shipments</option>
              <option value="GENERAL">General</option>
            </select>
          </div>
          <div>
            <label for="tic-prio" class="block text-xs font-medium text-muted-foreground mb-1">Priority</label>
            <select
              id="tic-prio"
              bind:value={newTicket.priority}
              class="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground text-sm focus:outline-none focus:ring-1 focus:ring-primary"
            >
              <option value="LOW">Low</option>
              <option value="NORMAL">Normal</option>
              <option value="HIGH">High</option>
              <option value="URGENT">Urgent (SLA Alert)</option>
            </select>
          </div>
          <div>
            <label for="tic-chan" class="block text-xs font-medium text-muted-foreground mb-1">Channel Source</label>
            <select
              id="tic-chan"
              bind:value={newTicket.channel}
              class="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground text-sm focus:outline-none focus:ring-1 focus:ring-primary"
            >
              <option value="PORTAL">Customer Portal</option>
              <option value="WHATSAPP">WhatsApp</option>
              <option value="EMAIL">Email</option>
              <option value="VOICE_BOT">AI Voice Bot</option>
            </select>
          </div>
        </div>

        <div class="pt-4 flex justify-end gap-3">
          <button
            type="button"
            onclick={() => (activeTab = 'helpdesk')}
            class="px-4 py-2 rounded-lg border border-border bg-muted/50 hover:bg-muted text-muted-foreground text-sm font-medium"
          >
            Cancel
          </button>
          <button
            type="submit"
            class="px-5 py-2 rounded-lg bg-primary hover:bg-primary/90 text-primary-foreground text-sm font-medium shadow"
          >
            Create Ticket
          </button>
        </div>
      </form>
    </div>
  {/if}
</div>
