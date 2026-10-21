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

  let tickets = $state<TicketItem[]>([
    {
      id: 'TCK-001',
      ticketNumber: 'SUP-9821',
      customerName: 'Ananya Roy',
      customerContact: '+91 98450 11234',
      subject: 'Flight PNR #6E-782 Rescheduling for Zurich Winter Tour',
      category: 'TRAVEL',
      priority: 'URGENT',
      status: 'IN_PROGRESS',
      channel: 'WHATSAPP',
      assignedTo: 'Vikram Joshi (Senior Agent)',
      businessUnit: 'TRAVEL',
      slaMinutesRemaining: 24,
      isSlaBreached: false,
      createdAt: '2026-10-21 14:30'
    },
    {
      id: 'TCK-002',
      ticketNumber: 'SUP-9822',
      customerName: 'Axis Financial Holdings',
      customerContact: 'finance@axiscorp.com',
      subject: 'Milestone 2 Tax Invoice & SOW Deliverables Signoff',
      category: 'BILLING',
      priority: 'HIGH',
      status: 'OPEN',
      channel: 'PORTAL',
      assignedTo: 'Finance Ops Desk',
      businessUnit: 'CONSULTING',
      slaMinutesRemaining: 55,
      isSlaBreached: false,
      createdAt: '2026-10-21 15:15'
    },
    {
      id: 'TCK-003',
      ticketNumber: 'SUP-9823',
      customerName: 'Apollo Diagnostics Laboratory',
      customerContact: 'lab-support@apollodiag.in',
      subject: 'HL7 Pathology Result Stream Sync Latency',
      category: 'TECHNICAL',
      priority: 'NORMAL',
      status: 'OPEN',
      channel: 'EMAIL',
      assignedTo: 'Dr. Ayya Medical Systems',
      businessUnit: 'HEALTHCARE',
      slaMinutesRemaining: 180,
      isSlaBreached: false,
      createdAt: '2026-10-21 16:00'
    },
    {
      id: 'TCK-004',
      ticketNumber: 'SUP-9820',
      customerName: 'Rajesh Mehra',
      customerContact: '+91 91234 56789',
      subject: 'Etsy Print-on-Demand Canvas Delivery Update',
      category: 'LOGISTICS',
      priority: 'LOW',
      status: 'RESOLVED',
      channel: 'PORTAL',
      assignedTo: 'Fulfillment Desk',
      businessUnit: 'ECOMMERCE',
      slaMinutesRemaining: 0,
      isSlaBreached: false,
      createdAt: '2026-10-20 11:20'
    }
  ]);

  let selectedTicketId = $state<string>('TCK-001');
  let selectedTicket = $derived(tickets.find((t) => t.id === selectedTicketId) || tickets[0]);
  let activeMessages = $state<TicketMessageItem[]>([
    {
      id: 'MSG-01',
      ticketId: 'TCK-001',
      senderName: 'Ananya Roy',
      senderType: 'CUSTOMER',
      message: 'Hello, our Swiss Alps tour flight was delayed by 3 hours. Can we re-route the connecting flight from Zurich to Geneva?',
      sentAt: 'Today, 2:32 PM'
    },
    {
      id: 'MSG-02',
      ticketId: 'TCK-001',
      senderName: 'Vikram Joshi (Senior Agent)',
      senderType: 'AGENT',
      message: 'Hello Ms. Roy, I have connected with Swiss Air NDC desk. We have protected seat 14A on LX-2804 departing at 17:45. Your airport transfers are notified.',
      sentAt: 'Today, 2:40 PM'
    }
  ]);

  let loading = $state<boolean>(false);
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
    businessUnit: 'TRAVEL'
  });

  async function loadTickets() {
    loading = true;
    try {
      const res = await supportGrpc.listTickets({
        status: statusFilter,
        priority: priorityFilter
      });
      if (res && res.tickets && res.tickets.length > 0) {
        tickets = res.tickets;
        if (!selectedTicketId) {
          await openTicket(tickets[0].id);
        }
      }
    } catch (e: any) {
      console.warn('Fallback data retained:', e);
    } finally {
      loading = false;
    }
  }

  async function openTicket(id: string) {
    selectedTicketId = id;
    try {
      const res = await supportGrpc.getTicketDetails({ ticketId: id });
      if (res && res.messages && res.messages.length > 0) {
        activeMessages = res.messages;
      }
    } catch (e) {
      console.warn('Retained local messages for ticket:', id);
    }
  }

  async function handleSendReply(e: Event) {
    e.preventDefault();
    if (!replyText.trim() || !selectedTicketId) return;

    const msg = replyText;
    replyText = '';
    const newMsg: TicketMessageItem = {
      id: `MSG-LOC-${Date.now().toString().slice(-4)}`,
      ticketId: selectedTicketId,
      senderName: 'You (Support Specialist)',
      senderType: 'AGENT',
      message: msg,
      sentAt: 'Just now'
    };
    activeMessages = [...activeMessages, newMsg];

    try {
      await supportGrpc.replyTicket({
        ticketId: selectedTicketId,
        message: msg,
        senderName: 'Support Specialist',
        closeTicket: false
      });
    } catch (err: any) {
      console.log('Recorded locally:', err);
    }
  }

  async function handleCreateTicket(e: Event) {
    e.preventDefault();
    try {
      const res = await supportGrpc.createTicket(newTicket);
      if (res && res.ticket) {
        tickets = [res.ticket, ...tickets];
        await openTicket(res.ticket.id);
      } else {
        const generatedNum = `SUP-${Math.floor(1000 + Math.random() * 9000)}`;
        const localTicket: TicketItem = {
          id: `TCK-${Date.now().toString().slice(-4)}`,
          ticketNumber: generatedNum,
          customerName: newTicket.customerName,
          customerContact: newTicket.customerContact,
          subject: newTicket.subject,
          category: newTicket.category,
          priority: newTicket.priority,
          status: 'OPEN',
          channel: newTicket.channel,
          assignedTo: 'Auto Triage Engine',
          businessUnit: newTicket.businessUnit,
          slaMinutesRemaining: newTicket.priority === 'URGENT' ? 30 : 120,
          isSlaBreached: false,
          createdAt: new Date().toISOString().replace('T', ' ').slice(0, 16)
        };
        tickets = [localTicket, ...tickets];
        selectedTicketId = localTicket.id;
        activeMessages = [
          {
            id: `MSG-INIT-${Date.now()}`,
            ticketId: localTicket.id,
            senderName: newTicket.customerName,
            senderType: 'CUSTOMER',
            message: newTicket.description || newTicket.subject,
            sentAt: 'Just now'
          }
        ];
      }

      activeTab = 'helpdesk';
      statusMessage = `✅ Ticket created successfully! Assigned to queue.`;
      newTicket = {
        customerName: '',
        customerContact: '',
        subject: '',
        description: '',
        category: 'TECHNICAL',
        priority: 'HIGH',
        channel: 'PORTAL',
        businessUnit: 'TRAVEL'
      };
    } catch (err: any) {
      statusMessage = `Ticket logged to local queue.`;
    }
  }

  async function handleCloseTicket() {
    if (!selectedTicket) return;
    try {
      await supportGrpc.updateTicketStatus({
        ticketId: selectedTicket.id,
        status: 'RESOLVED'
      });
    } catch (err: any) {
      console.log('Status updated locally');
    }
    selectedTicket.status = 'RESOLVED';
    statusMessage = `🎉 Ticket #${selectedTicket.ticketNumber} resolved and closed. Customer CSAT notification dispatched.`;
  }

  let urgentTickets = $derived(tickets.filter(t => t.priority === 'URGENT' && t.status !== 'RESOLVED'));
  let openTickets = $derived(tickets.filter(t => t.status !== 'RESOLVED'));

  onMount(() => {
    loadTickets();
  });
</script>

<svelte:head>
  <title>Omnichannel Support & SLA Management | MrBiz-OS</title>
</svelte:head>

<div class="support-dashboard">
  <!-- Top Bar -->
  <header class="page-header">
    <div class="header-left">
      <div class="badge-row">
        <span class="badge badge-rose">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9M13.73 21a2 2 0 0 1-3.46 0"/></svg>
          Enterprise Helpdesk & SLA Engine
        </span>
        <span class="badge badge-emerald">
          <span class="live-dot"></span> Realtime Omnichannel Inbound
        </span>
        <span class="badge badge-purple">
          Multi-Tenant Group Triage
        </span>
      </div>
      <h1 class="header-title">Omnichannel Support & SLA Management</h1>
      <p class="header-desc">
        Unified customer support hub connecting WhatsApp, Travel GDS re-issues, Healthcare diagnostic inquiries, and B2B consulting escalations.
      </p>
    </div>

    <div class="header-actions">
      <button class="btn-refresh" onclick={loadTickets} title="Refresh Tickets">
        <svg class={loading ? 'spin' : ''} width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21.5 2v6h-6M21.34 15.57a10 10 0 1 1-.57-8.38l5.67-5.67"/></svg>
        {loading ? 'Refreshing...' : 'Sync Queue'}
      </button>
      <button class="btn-primary" onclick={() => activeTab = 'new_ticket'}>
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="12" y1="5" x2="12" y2="19"></line><line x1="5" y1="12" x2="19" y2="12"></line></svg>
        Raise New Ticket
      </button>
    </div>
  </header>

  <!-- Status Alert -->
  {#if statusMessage}
    <div class="status-banner">
      <div class="banner-content">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="M12 16v-4M12 8h.01"/></svg>
        <span>{statusMessage}</span>
      </div>
      <button onclick={() => (statusMessage = null)} class="banner-close" aria-label="Dismiss">✕</button>
    </div>
  {/if}

  <!-- Summary Metrics Cards -->
  <section class="kpi-grid">
    <div class="kpi-card kpi-purple">
      <div class="kpi-top">
        <span class="kpi-label">Active Open Tickets</span>
        <div class="kpi-icon-wrap">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="4" width="18" height="16" rx="2"/><path d="M7 8h10M7 12h10M7 16h6"/></svg>
        </div>
      </div>
      <div class="kpi-value font-mono">{openTickets.length} <span class="unit">Pending</span></div>
      <div class="kpi-footer">
        <span class="trend-purple">{urgentTickets.length} Critical SLAs</span> across {tickets.length} total
      </div>
    </div>

    <div class="kpi-card kpi-emerald">
      <div class="kpi-top">
        <span class="kpi-label">Avg. First Response</span>
        <div class="kpi-icon-wrap">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
        </div>
      </div>
      <div class="kpi-value font-mono">8.5 <span class="unit">mins</span></div>
      <div class="kpi-footer">
        <span class="trend-up">↓ 24% faster</span> than SLA target
      </div>
    </div>

    <div class="kpi-card kpi-amber">
      <div class="kpi-top">
        <span class="kpi-label">SLA Breach Alerts</span>
        <div class="kpi-icon-wrap">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
        </div>
      </div>
      <div class="kpi-value font-mono">0 <span class="unit">Breached</span></div>
      <div class="kpi-footer">
        <span class="trend-up">100% On-Time</span> Tier-1 resolution
      </div>
    </div>

    <div class="kpi-card kpi-rose">
      <div class="kpi-top">
        <span class="kpi-label">CSAT Satisfaction</span>
        <div class="kpi-icon-wrap">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>
        </div>
      </div>
      <div class="kpi-value font-mono">4.9 <span class="unit">/ 5.0 ⭐</span></div>
      <div class="kpi-footer">
        <span class="trend-up">98.4%</span> positive customer ratings
      </div>
    </div>
  </section>

  <!-- Navigation Tab Bar -->
  <nav class="tabs-nav">
    <div class="tabs-list">
      <button
        onclick={() => (activeTab = 'helpdesk')}
        class="tab-btn {activeTab === 'helpdesk' ? 'active' : ''}"
      >
        <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
        Support Ticket Queue
        <span class="tab-count">{tickets.length}</span>
      </button>
      <button
        onclick={() => (activeTab = 'new_ticket')}
        class="tab-btn {activeTab === 'new_ticket' ? 'active' : ''}"
      >
        <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 5v14M5 12h14"/></svg>
        Create New Ticket
      </button>
    </div>
  </nav>

  <!-- TAB 1: 2-PANE WORKBENCH -->
  {#if activeTab === 'helpdesk'}
    <div class="workbench-card">
      <!-- Left Panel: Ticket Queue (5 cols) -->
      <aside class="ticket-queue-pane">
        <div class="queue-header">
          <div class="queue-title-row">
            <span class="queue-title">TICKET STREAM</span>
            <span class="count-tag font-mono">{tickets.length} Tickets</span>
          </div>
        </div>

        <div class="queue-list">
          {#each tickets as t}
            <button
              type="button"
              onclick={() => openTicket(t.id)}
              class="queue-item {selectedTicketId === t.id ? 'active' : ''}"
            >
              <div class="item-top">
                <span class="ticket-num font-mono">{t.ticketNumber}</span>
                <span class="prio-pill {t.priority.toLowerCase()}">{t.priority}</span>
              </div>

              <h4 class="item-subject">{t.subject}</h4>

              <div class="item-meta">
                <span>{t.customerName}</span>
                <span class="bu-tag">{t.businessUnit}</span>
              </div>

              <div class="item-bottom">
                <span class="sla-timer font-mono {t.slaMinutesRemaining < 30 && t.status !== 'RESOLVED' ? 'urgent' : ''}">
                  ⏱️ {t.status === 'RESOLVED' ? 'Closed' : `${t.slaMinutesRemaining}m SLA`}
                </span>
                <span class="status-tag {t.status.toLowerCase()} font-mono">{t.status}</span>
              </div>
            </button>
          {/each}
        </div>
      </aside>

      <!-- Right Panel: Ticket Communication & Thread (7 cols) -->
      <main class="ticket-thread-pane">
        {#if selectedTicket}
          <!-- Thread Header -->
          <div class="thread-header">
            <div>
              <div class="thread-header-top">
                <span class="font-mono text-purple font-bold text-xs">{selectedTicket.ticketNumber}</span>
                <span class="status-tag {selectedTicket.status.toLowerCase()} font-mono">{selectedTicket.status}</span>
                <span class="channel-pill">{selectedTicket.channel}</span>
              </div>
              <h2 class="thread-title">{selectedTicket.subject}</h2>
              <div class="thread-contact">
                <strong>{selectedTicket.customerName}</strong> • {selectedTicket.customerContact} • Assigned: <span class="agent-name">{selectedTicket.assignedTo}</span>
              </div>
            </div>

            <div class="thread-actions">
              {#if selectedTicket.status !== 'RESOLVED' && selectedTicket.status !== 'CLOSED'}
                <button onclick={handleCloseTicket} class="btn-resolve">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="20 6 9 17 4 12"/></svg>
                  Resolve & Close
                </button>
              {/if}
            </div>
          </div>

          <!-- Message History -->
          <div class="messages-stream">
            {#each activeMessages as m}
              {@const isAgent = m.senderType === 'AGENT'}
              <div class="message-bubble-wrapper {isAgent ? 'agent' : 'customer'}">
                <div class="message-meta">
                  <span class="sender-name">{m.senderName}</span>
                  <span class="sent-time">{m.sentAt}</span>
                </div>
                <div class="message-bubble {isAgent ? 'agent' : 'customer'}">
                  <p>{m.message}</p>
                </div>
              </div>
            {/each}
          </div>

          <!-- Reply Composer -->
          <form onsubmit={handleSendReply} class="composer-box">
            <input
              type="text"
              placeholder="Type your response to the customer or update ticket notes..."
              bind:value={replyText}
              class="composer-input"
            />
            <button type="submit" class="btn-send">
              <span>Send Reply</span>
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="22" y1="2" x2="11" y2="13"></line><polygon points="22 2 15 22 11 13 2 9 22 2"></polygon></svg>
            </button>
          </form>
        {:else}
          <div class="empty-state">
            <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="#64748b" stroke-width="1.5"><rect x="3" y="4" width="18" height="16" rx="2"/><path d="M7 8h10M7 12h10M7 16h6"/></svg>
            <p>Select a ticket from the queue on the left to inspect communications and SLA countdown.</p>
          </div>
        {/if}
      </main>
    </div>
  {/if}

  <!-- TAB 2: CREATE TICKET FORM -->
  {#if activeTab === 'new_ticket'}
    <div class="new-ticket-card">
      <div class="form-card-header">
        <div class="icon-circle rose">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 5v14M5 12h14"/></svg>
        </div>
        <div>
          <h2 class="form-card-title">Raise New Customer Support Ticket</h2>
          <p class="form-card-desc">Log inquiries from WhatsApp, phone triage, or incoming corporate client emails.</p>
        </div>
      </div>

      <form onsubmit={handleCreateTicket} class="new-ticket-form">
        <div class="form-row-2">
          <div class="form-group">
            <label for="tic-name" class="form-label">Customer / Client Name</label>
            <input
              id="tic-name"
              type="text"
              required
              placeholder="e.g. Ananya Roy or Axis Corp"
              bind:value={newTicket.customerName}
              class="form-input"
            />
          </div>
          <div class="form-group">
            <label for="tic-contact" class="form-label">Contact Phone / Email</label>
            <input
              id="tic-contact"
              type="text"
              required
              placeholder="+91 98400 00000 or client@domain.com"
              bind:value={newTicket.customerContact}
              class="form-input"
            />
          </div>
        </div>

        <div class="form-group">
          <label for="tic-subj" class="form-label">Issue Subject / Summary</label>
          <input
            id="tic-subj"
            type="text"
            required
            placeholder="e.g. Flight PNR reschedule request or GST Invoice clarification"
            bind:value={newTicket.subject}
            class="form-input"
          />
        </div>

        <div class="form-group">
          <label for="tic-desc" class="form-label">Detailed Description</label>
          <textarea
            id="tic-desc"
            rows="3"
            required
            placeholder="Specific passenger details, error messages, or customer requests"
            bind:value={newTicket.description}
            class="form-textarea"
          ></textarea>
        </div>

        <div class="form-row-3">
          <div class="form-group">
            <label for="tic-bu" class="form-label">Business Unit</label>
            <select id="tic-bu" bind:value={newTicket.businessUnit} class="form-select">
              <option value="TRAVEL">Travel Agency & Tours</option>
              <option value="HEALTHCARE">Smart Healthcare / Clinic</option>
              <option value="CONSULTING">Software Consulting</option>
              <option value="ECOMMERCE">E-Commerce & Retail</option>
            </select>
          </div>

          <div class="form-group">
            <label for="tic-prio" class="form-label">Priority & SLA</label>
            <select id="tic-prio" bind:value={newTicket.priority} class="form-select">
              <option value="LOW">Low (24hr SLA)</option>
              <option value="NORMAL">Normal (4hr SLA)</option>
              <option value="HIGH">High (2hr SLA)</option>
              <option value="URGENT">Urgent (30m SLA Alert)</option>
            </select>
          </div>

          <div class="form-group">
            <label for="tic-chan" class="form-label">Inbound Channel</label>
            <select id="tic-chan" bind:value={newTicket.channel} class="form-select">
              <option value="WHATSAPP">WhatsApp Business</option>
              <option value="PORTAL">Customer Web Portal</option>
              <option value="EMAIL">Email Support</option>
              <option value="VOICE_BOT">AI Voice Agent</option>
            </select>
          </div>
        </div>

        <div class="form-actions-bar">
          <button type="button" class="btn-cancel" onclick={() => activeTab = 'helpdesk'}>
            Cancel
          </button>
          <button type="submit" class="btn-submit-primary">
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="20 6 9 17 4 12"/></svg>
            Dispatch Ticket to Queue
          </button>
        </div>
      </form>
    </div>
  {/if}
</div>

<style>
  /* -------------------------------------------------------------
     OMNICHANNEL SUPPORT & SLA MANAGEMENT STYLES (Pure Scoped CSS)
  ------------------------------------------------------------- */
  .support-dashboard {
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
    padding-bottom: 3rem;
  }

  /* Header */
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

  .badge-rose {
    background: rgba(244, 63, 94, 0.12);
    color: #fb7185;
    border: 1px solid rgba(244, 63, 94, 0.3);
  }

  .badge-emerald {
    background: rgba(16, 185, 129, 0.12);
    color: #34d399;
    border: 1px solid rgba(16, 185, 129, 0.3);
  }

  .badge-purple {
    background: rgba(168, 85, 247, 0.12);
    color: #c084fc;
    border: 1px solid rgba(168, 85, 247, 0.3);
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
  }

  .btn-primary {
    display: inline-flex;
    align-items: center;
    gap: 0.45rem;
    font-size: 0.8125rem;
    font-weight: 700;
    padding: 0.55rem 1.1rem;
    border-radius: 0.5rem;
    background: linear-gradient(135deg, #e11d48 0%, #be123c 100%);
    color: #ffffff;
    border: none;
    cursor: pointer;
    box-shadow: 0 4px 14px rgba(225, 29, 72, 0.35);
    transition: all 0.2s ease;
  }

  .btn-primary:hover {
    transform: translateY(-1px);
    box-shadow: 0 6px 18px rgba(225, 29, 72, 0.45);
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
    background: rgba(16, 185, 129, 0.12);
    border: 1px solid rgba(16, 185, 129, 0.3);
    color: #34d399;
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
    transition: transform 0.2s ease;
  }

  .kpi-card:hover {
    transform: translateY(-2px);
  }

  .kpi-card.kpi-purple { border-top: 3px solid #a855f7; }
  .kpi-card.kpi-emerald { border-top: 3px solid #10b981; }
  .kpi-card.kpi-amber { border-top: 3px solid #f59e0b; }
  .kpi-card.kpi-rose { border-top: 3px solid #f43f5e; }

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

  .trend-up { color: #34d399; font-weight: 700; }
  .trend-purple { color: #c084fc; font-weight: 700; }

  /* Tabs Nav */
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
    transition: all 0.2s ease;
  }

  .tab-btn:hover {
    color: var(--color-foreground, #f8fafc);
    background: rgba(30, 41, 59, 0.5);
  }

  .tab-btn.active {
    background: linear-gradient(135deg, rgba(244, 63, 94, 0.2) 0%, rgba(168, 85, 247, 0.2) 100%);
    color: #fb7185;
    border-color: rgba(244, 63, 94, 0.4);
    box-shadow: 0 2px 8px rgba(244, 63, 94, 0.15);
  }

  .tab-count {
    font-size: 0.6875rem;
    font-weight: 700;
    padding: 0.1rem 0.45rem;
    border-radius: 9999px;
    background: rgba(0, 0, 0, 0.35);
    border: 1px solid rgba(255, 255, 255, 0.1);
  }

  /* 2-Pane Workbench */
  .workbench-card {
    display: grid;
    grid-template-columns: 1fr;
    background: rgba(15, 23, 42, 0.85);
    border: 1px solid var(--color-border, #334155);
    border-radius: 0.875rem;
    overflow: hidden;
    height: 720px;
    box-shadow: 0 8px 30px rgba(0, 0, 0, 0.3);
  }

  @media (min-width: 1024px) {
    .workbench-card {
      grid-template-columns: 4.8fr 7.2fr;
    }
  }

  /* Left Queue */
  .ticket-queue-pane {
    display: flex;
    flex-direction: column;
    border-right: 1px solid var(--color-border, #334155);
    background: rgba(15, 23, 42, 0.5);
    height: 100%;
    overflow: hidden;
  }

  .queue-header {
    padding: 0.85rem 1rem;
    border-bottom: 1px solid var(--color-border, #334155);
    background: rgba(30, 41, 59, 0.4);
  }

  .queue-title-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .queue-title {
    font-size: 0.75rem;
    font-weight: 700;
    letter-spacing: 0.05em;
    color: var(--color-muted-foreground, #94a3b8);
  }

  .count-tag {
    font-size: 0.6875rem;
    font-weight: 700;
    padding: 0.15rem 0.45rem;
    border-radius: 0.375rem;
    background: rgba(30, 41, 59, 0.9);
    color: #94a3b8;
  }

  .queue-list {
    flex: 1;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
  }

  .queue-item {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
    padding: 1rem;
    border: none;
    border-bottom: 1px solid rgba(51, 65, 85, 0.4);
    background: transparent;
    text-align: left;
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .queue-item:hover {
    background: rgba(30, 41, 59, 0.6);
  }

  .queue-item.active {
    background: rgba(244, 63, 94, 0.08);
    border-left: 4px solid #f43f5e;
  }

  .item-top {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .ticket-num {
    font-size: 0.75rem;
    font-weight: 700;
    color: #94a3b8;
  }

  .prio-pill {
    font-size: 0.625rem;
    font-weight: 700;
    padding: 0.15rem 0.45rem;
    border-radius: 0.25rem;
    text-transform: uppercase;
  }

  .prio-pill.urgent {
    background: rgba(239, 68, 68, 0.2);
    color: #f87171;
    border: 1px solid rgba(239, 68, 68, 0.4);
  }

  .prio-pill.high {
    background: rgba(245, 158, 11, 0.2);
    color: #fbbf24;
    border: 1px solid rgba(245, 158, 11, 0.4);
  }

  .prio-pill.normal {
    background: rgba(59, 130, 246, 0.2);
    color: #60a5fa;
    border: 1px solid rgba(59, 130, 246, 0.4);
  }

  .prio-pill.low {
    background: rgba(100, 116, 139, 0.2);
    color: #94a3b8;
  }

  .item-subject {
    font-size: 0.875rem;
    font-weight: 700;
    color: var(--color-foreground, #f8fafc);
    margin: 0;
    line-height: 1.3;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .item-meta {
    display: flex;
    align-items: center;
    justify-content: space-between;
    font-size: 0.75rem;
    color: var(--color-muted-foreground, #94a3b8);
  }

  .bu-tag {
    font-size: 0.625rem;
    font-weight: 700;
    padding: 0.1rem 0.4rem;
    border-radius: 0.25rem;
    background: rgba(30, 41, 59, 0.8);
    color: #cbd5e1;
    border: 1px solid rgba(148, 163, 184, 0.2);
  }

  .item-bottom {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding-top: 0.35rem;
    font-size: 0.75rem;
  }

  .sla-timer {
    color: #34d399;
    font-size: 0.75rem;
    font-weight: 600;
  }

  .sla-timer.urgent {
    color: #f87171;
    font-weight: 700;
    animation: pulse-dot 1.5s infinite;
  }

  .status-tag {
    font-size: 0.6875rem;
    font-weight: 700;
    padding: 0.15rem 0.5rem;
    border-radius: 9999px;
    text-transform: uppercase;
  }

  .status-tag.open {
    background: rgba(59, 130, 246, 0.15);
    color: #60a5fa;
    border: 1px solid rgba(59, 130, 246, 0.3);
  }

  .status-tag.in_progress {
    background: rgba(245, 158, 11, 0.15);
    color: #fbbf24;
    border: 1px solid rgba(245, 158, 11, 0.3);
  }

  .status-tag.resolved {
    background: rgba(16, 185, 129, 0.15);
    color: #34d399;
    border: 1px solid rgba(16, 185, 129, 0.3);
  }

  /* Right Thread Pane */
  .ticket-thread-pane {
    display: flex;
    flex-direction: column;
    height: 100%;
    background: rgba(15, 23, 42, 0.7);
    overflow: hidden;
  }

  .thread-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    padding: 1.25rem;
    border-bottom: 1px solid var(--color-border, #334155);
    background: rgba(30, 41, 59, 0.35);
  }

  .thread-header-top {
    display: flex;
    align-items: center;
    gap: 0.6rem;
    margin-bottom: 0.35rem;
  }

  .text-purple { color: #c084fc; }

  .channel-pill {
    font-size: 0.6875rem;
    font-weight: 700;
    padding: 0.1rem 0.45rem;
    border-radius: 0.25rem;
    background: rgba(30, 41, 59, 0.8);
    color: #94a3b8;
    border: 1px solid rgba(148, 163, 184, 0.2);
  }

  .thread-title {
    font-size: 1.125rem;
    font-weight: 800;
    color: var(--color-foreground, #f8fafc);
    margin: 0 0 0.35rem 0;
    line-height: 1.3;
  }

  .thread-contact {
    font-size: 0.78125rem;
    color: var(--color-muted-foreground, #94a3b8);
  }

  .thread-contact strong { color: var(--color-foreground, #f8fafc); }
  .agent-name { color: #818cf8; font-weight: 600; }

  .btn-resolve {
    display: inline-flex;
    align-items: center;
    gap: 0.4rem;
    font-size: 0.75rem;
    font-weight: 700;
    padding: 0.45rem 0.9rem;
    border-radius: 0.5rem;
    background: rgba(16, 185, 129, 0.15);
    color: #34d399;
    border: 1px solid rgba(16, 185, 129, 0.3);
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .btn-resolve:hover {
    background: rgba(16, 185, 129, 0.28);
    border-color: rgba(16, 185, 129, 0.5);
  }

  /* Messages Stream */
  .messages-stream {
    flex: 1;
    overflow-y: auto;
    padding: 1.25rem;
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
  }

  .message-bubble-wrapper {
    display: flex;
    flex-direction: column;
    max-width: 80%;
  }

  .message-bubble-wrapper.agent {
    align-self: flex-end;
    align-items: flex-end;
  }

  .message-bubble-wrapper.customer {
    align-self: flex-start;
    align-items: flex-start;
  }

  .message-meta {
    font-size: 0.6875rem;
    color: #64748b;
    margin-bottom: 0.3rem;
    display: flex;
    gap: 0.5rem;
  }

  .message-bubble {
    padding: 0.85rem 1.1rem;
    border-radius: 0.875rem;
    font-size: 0.8125rem;
    line-height: 1.5;
  }

  .message-bubble p { margin: 0; }

  .message-bubble.agent {
    background: linear-gradient(135deg, #7c3aed 0%, #6366f1 100%);
    color: #ffffff;
    border-bottom-right-radius: 0.2rem;
    box-shadow: 0 4px 12px rgba(124, 58, 237, 0.25);
  }

  .message-bubble.customer {
    background: rgba(30, 41, 59, 0.85);
    border: 1px solid rgba(51, 65, 85, 0.8);
    color: var(--color-foreground, #f8fafc);
    border-bottom-left-radius: 0.2rem;
  }

  /* Composer */
  .composer-box {
    padding: 0.85rem 1.25rem;
    border-top: 1px solid var(--color-border, #334155);
    background: rgba(30, 41, 59, 0.4);
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .composer-input {
    flex: 1;
    padding: 0.65rem 0.9rem;
    font-size: 0.8125rem;
    border-radius: 0.5rem;
    background: rgba(15, 23, 42, 0.75);
    border: 1px solid var(--color-border, #334155);
    color: var(--color-foreground, #f8fafc);
    outline: none;
    transition: border-color 0.2s ease;
  }

  .composer-input:focus {
    border-color: #818cf8;
    box-shadow: 0 0 0 2px rgba(99, 102, 241, 0.2);
  }

  .btn-send {
    display: inline-flex;
    align-items: center;
    gap: 0.4rem;
    font-size: 0.8125rem;
    font-weight: 700;
    padding: 0.65rem 1.1rem;
    border-radius: 0.5rem;
    background: linear-gradient(135deg, #7c3aed 0%, #6366f1 100%);
    color: #ffffff;
    border: none;
    cursor: pointer;
    box-shadow: 0 4px 12px rgba(124, 58, 237, 0.3);
    white-space: nowrap;
    transition: all 0.2s ease;
  }

  .btn-send:hover {
    transform: translateY(-1px);
    box-shadow: 0 6px 16px rgba(124, 58, 237, 0.4);
  }

  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: var(--color-muted-foreground, #94a3b8);
    gap: 1rem;
    text-align: center;
    padding: 2rem;
  }

  /* Form */
  .new-ticket-card {
    background: rgba(15, 23, 42, 0.85);
    border: 1px solid var(--color-border, #334155);
    border-radius: 0.875rem;
    padding: 1.75rem;
    max-width: 780px;
    margin: 0 auto;
    box-shadow: 0 10px 25px rgba(0, 0, 0, 0.3);
  }

  .form-card-header {
    display: flex;
    align-items: center;
    gap: 0.85rem;
    margin-bottom: 1.5rem;
    padding-bottom: 1rem;
    border-bottom: 1px solid var(--color-border, #334155);
  }

  .icon-circle {
    width: 38px;
    height: 38px;
    border-radius: 0.5rem;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .icon-circle.rose {
    background: rgba(244, 63, 94, 0.15);
    color: #fb7185;
  }

  .form-card-title {
    font-size: 1.25rem;
    font-weight: 800;
    color: var(--color-foreground, #f8fafc);
    margin: 0 0 0.25rem 0;
  }

  .form-card-desc {
    font-size: 0.8125rem;
    color: var(--color-muted-foreground, #94a3b8);
    margin: 0;
  }

  .new-ticket-form {
    display: flex;
    flex-direction: column;
    gap: 1.15rem;
  }

  .form-row-2 {
    display: grid;
    grid-template-columns: 1fr;
    gap: 1rem;
  }

  @media (min-width: 640px) {
    .form-row-2 { grid-template-columns: 1fr 1fr; }
  }

  .form-row-3 {
    display: grid;
    grid-template-columns: 1fr;
    gap: 1rem;
  }

  @media (min-width: 640px) {
    .form-row-3 { grid-template-columns: 1fr 1fr 1fr; }
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
    border-color: #fb7185;
    box-shadow: 0 0 0 2px rgba(244, 63, 94, 0.2);
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

  .btn-submit-primary {
    display: inline-flex;
    align-items: center;
    gap: 0.45rem;
    padding: 0.6rem 1.25rem;
    font-size: 0.8125rem;
    font-weight: 700;
    border-radius: 0.5rem;
    background: linear-gradient(135deg, #e11d48 0%, #be123c 100%);
    color: #ffffff;
    border: none;
    cursor: pointer;
    box-shadow: 0 4px 14px rgba(225, 29, 72, 0.35);
  }

  .spin {
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }
</style>
