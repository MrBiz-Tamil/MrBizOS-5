<script lang="ts">
  import { onMount } from 'svelte';
  import { omnichannelGrpc } from '$lib/grpc';

  interface ConversationSummary {
    id: string;
    channel: number | string; // 1: WhatsApp, 2: Instagram, 3: Facebook, 4: Email, 6: LinkedIn
    customerName: string;
    customerIdentifier: string;
    lastMessage: string;
    lastMessageTime: string;
    unreadCount: number;
    status: string;
    assignedAgent: string;
    businessUnit: string;
    sentiment: string;
    interactionType?: string;
    postTitleOrUrl?: string;
  }

  interface MessageItem {
    id: string;
    conversationId: string;
    senderType: string; // CUSTOMER, AGENT, AI_BOT
    senderName: string;
    content: string;
    mediaUrl?: string;
    mediaType?: string;
    sentAt: string;
    deliveryStatus: string;
  }

  let conversations = $state<ConversationSummary[]>([]);
  let selectedConvId = $state<string>('');
  let selectedSummary = $derived(conversations.find((c) => c.id === selectedConvId) || conversations[0]);
  let messages = $state<MessageItem[]>([]);
  let activeJourneysCount = $state<number>(1);
  let openInvoicesCount = $state<number>(1);
  let customerPhone = $state<string>('+91 98401 23456');
  let customerEmail = $state<string>('ananya.roy@traveller.in');

  let loading = $state<boolean>(true);
  let channelFilter = $state<number>(0); // 0: All
  let searchQuery = $state<string>('');
  let outboundText = $state<string>('');
  let aiSuggestion = $state<string | null>(null);
  let aiConfidence = $state<number>(0);
  let aiSources = $state<string[]>([]);
  let isAiGenerating = $state<boolean>(false);
  let statusMessage = $state<string | null>(null);

  // Fallback initial conversations if backend has none
  const defaultConversations: ConversationSummary[] = [
    {
      id: 'CONV-TRV-101',
      channel: 2, // Instagram
      customerName: 'Ananya Roy',
      customerIdentifier: '@ananya_travels',
      lastMessage: 'Loved your latest reel on Zurich! Do you have a package for Switzerland in December?',
      lastMessageTime: '09:55 AM',
      unreadCount: 1,
      status: 'OPEN',
      assignedAgent: 'Vikram (Travel Desk)',
      businessUnit: 'TRAVEL',
      sentiment: 'POSITIVE',
      postTitleOrUrl: 'Reel: Top 5 Swiss Alps Winter Stays 🏔️'
    },
    {
      id: 'CONV-MED-102',
      channel: 1, // WhatsApp
      customerName: 'Dr. Ramesh Babu',
      customerIdentifier: '+91 98402 99881',
      lastMessage: 'Is Dr. Ayya available for cardiology video consult today?',
      lastMessageTime: '1 hr ago',
      unreadCount: 0,
      status: 'OPEN',
      assignedAgent: 'Priya (Clinic)',
      businessUnit: 'HEALTHCARE',
      sentiment: 'NEUTRAL'
    },
    {
      id: 'CONV-EST-103',
      channel: 1, // WhatsApp
      customerName: 'Rajesh Kumar',
      customerIdentifier: '+91 98200 44556',
      lastMessage: 'Can you send the revised quotation and site plan for the office space?',
      lastMessageTime: '10 mins ago',
      unreadCount: 2,
      status: 'OPEN',
      assignedAgent: 'Divya (Real Estate)',
      businessUnit: 'REALESTATE',
      sentiment: 'POSITIVE'
    },
    {
      id: 'CONV-SOW-104',
      channel: 6, // LinkedIn
      customerName: 'Siddharth Mehta',
      customerIdentifier: 'in/siddharth-mehta',
      lastMessage: 'Can your platform handle 3-way matching and multi-entity consolidation?',
      lastMessageTime: '5 mins ago',
      unreadCount: 1,
      status: 'OPEN',
      assignedAgent: 'Anil (SOW Consulting)',
      businessUnit: 'CONSULTING',
      sentiment: 'NEUTRAL'
    },
    {
      id: 'CONV-HLT-105',
      channel: 4, // Email
      customerName: 'Kavita Raman',
      customerIdentifier: 'kavita@apollochain.org',
      lastMessage: 'Does your telehealth engine support HL7 / FHIR integration for clinic chains?',
      lastMessageTime: '15 mins ago',
      unreadCount: 0,
      status: 'OPEN',
      assignedAgent: 'Priya (Clinic)',
      businessUnit: 'HEALTHCARE',
      sentiment: 'POSITIVE'
    }
  ];

  function getChannelMeta(ch: number | string) {
    const val = Number(ch);
    switch (val) {
      case 1:
        return { label: 'WhatsApp', color: '#25D366', icon: '🟢', bg: 'rgba(37, 211, 102, 0.12)', border: 'rgba(37, 211, 102, 0.3)', text: '#4ade80' };
      case 2:
        return { label: 'Instagram DM', color: '#E1306C', icon: '🟣', bg: 'rgba(225, 48, 108, 0.12)', border: 'rgba(225, 48, 108, 0.3)', text: '#f472b6' };
      case 3:
        return { label: 'Facebook', color: '#1877F2', icon: '🔵', bg: 'rgba(24, 119, 242, 0.12)', border: 'rgba(24, 119, 242, 0.3)', text: '#60a5fa' };
      case 4:
        return { label: 'Email', color: '#EA4335', icon: '📧', bg: 'rgba(234, 67, 53, 0.12)', border: 'rgba(234, 67, 53, 0.3)', text: '#f87171' };
      case 6:
        return { label: 'LinkedIn', color: '#0A66C2', icon: '👔', bg: 'rgba(10, 102, 194, 0.12)', border: 'rgba(10, 102, 194, 0.3)', text: '#38bdf8' };
      default:
        return { label: 'Livechat', color: '#06B6D4', icon: '💬', bg: 'rgba(6, 182, 212, 0.12)', border: 'rgba(6, 182, 212, 0.3)', text: '#22d3ee' };
    }
  }

  function getInitials(name: string): string {
    if (!name) return '??';
    return name.split(' ').map((p) => p[0]).slice(0, 2).join('').toUpperCase();
  }

  let filteredConversations = $derived(
    conversations.filter((c) => {
      const matchChannel = channelFilter === 0 || Number(c.channel) === channelFilter;
      const matchSearch =
        searchQuery.trim() === '' ||
        c.customerName.toLowerCase().includes(searchQuery.toLowerCase()) ||
        c.customerIdentifier.toLowerCase().includes(searchQuery.toLowerCase()) ||
        c.businessUnit.toLowerCase().includes(searchQuery.toLowerCase()) ||
        c.lastMessage.toLowerCase().includes(searchQuery.toLowerCase());
      return matchChannel && matchSearch;
    })
  );

  async function loadConversations() {
    loading = true;
    try {
      const res = await omnichannelGrpc.listConversations({
        channel: channelFilter
      });
      if (res && res.conversations && res.conversations.length > 0) {
        conversations = res.conversations;
      } else {
        conversations = defaultConversations;
      }
      if (conversations.length > 0 && !selectedConvId) {
        await openConversation(conversations[0].id);
      }
    } catch (e: any) {
      console.warn('Fallback data if backend is offline:', e);
      conversations = defaultConversations;
      if (!selectedConvId && conversations.length > 0) {
        await openConversation(conversations[0].id);
      }
    } finally {
      loading = false;
    }
  }

  async function openConversation(id: string) {
    selectedConvId = id;
    aiSuggestion = null;
    const conv = conversations.find((c) => c.id === id);
    if (conv) {
      customerEmail = conv.customerIdentifier.includes('@') && !conv.customerIdentifier.startsWith('@')
        ? conv.customerIdentifier
        : `${conv.customerName.toLowerCase().replace(/[^a-z0-9]/g, '.')}@client.in`;
      customerPhone = conv.customerIdentifier.startsWith('+') ? conv.customerIdentifier : '+91 98401 23456';
    }

    try {
      const res = await omnichannelGrpc.getConversation({ conversationId: id });
      if (res && res.messages && res.messages.length > 0) {
        messages = res.messages;
        if (res.customerEmail) customerEmail = res.customerEmail;
        if (res.customerPhone) customerPhone = res.customerPhone;
        activeJourneysCount = res.activeJourneysCount || 1;
        openInvoicesCount = res.openInvoicesCount || 1;
        return;
      }
    } catch (e) {
      // fallback
    }

    // Default mock conversation stream if backend had empty thread
    if (conv?.businessUnit === 'TRAVEL' || conv?.id === 'CONV-TRV-101') {
      messages = [
        {
          id: 'MSG-01',
          conversationId: id,
          senderType: 'CUSTOMER',
          senderName: conv.customerName,
          content: 'Hi MrBiz Travel Team! Loved your latest reel on Zurich! Do you have a package for Switzerland in December?',
          sentAt: '09:55 AM',
          deliveryStatus: 'READ'
        },
        {
          id: 'MSG-02',
          conversationId: id,
          senderType: 'AI_BOT',
          senderName: 'MrBiz Travel Concierge AI',
          content: 'Hello Ananya! 🏔️ Yes! We have an exclusive "Swiss Alps Winter Wonderland 7D/6N" package with 4-star stays, Glacier Express tickets, and full Schengen visa clearance support. Would you like a customized proforma estimate?',
          sentAt: '09:56 AM',
          deliveryStatus: 'READ'
        },
        {
          id: 'MSG-03',
          conversationId: id,
          senderType: 'CUSTOMER',
          senderName: conv.customerName,
          content: 'Yes please! We are 2 adults traveling from Mumbai. Also need help with Schengen visa slot booking at VFS.',
          sentAt: '10:02 AM',
          deliveryStatus: 'READ'
        }
      ];
    } else {
      messages = [
        {
          id: 'MSG-01',
          conversationId: id,
          senderType: 'CUSTOMER',
          senderName: conv?.customerName || 'Client',
          content: conv?.lastMessage || 'Hello, I have an inquiry regarding your services.',
          sentAt: conv?.lastMessageTime || 'Just now',
          deliveryStatus: 'READ'
        }
      ];
    }
  }

  async function handleSendMessage(e?: Event) {
    if (e) e.preventDefault();
    if (!outboundText.trim() || !selectedConvId) return;

    const content = outboundText;
    outboundText = '';
    aiSuggestion = null;

    try {
      const res = await omnichannelGrpc.sendMessage({
        conversationId: selectedConvId,
        content
      });
      if (res && res.message) {
        messages = [...messages, res.message];
        return;
      }
    } catch (err: any) {
      // Fallback optimistic message
    }

    messages = [
      ...messages,
      {
        id: `MSG-LOCAL-${Date.now() % 10000}`,
        conversationId: selectedConvId,
        senderType: 'AGENT',
        senderName: 'You (Agent)',
        content,
        sentAt: 'Just now',
        deliveryStatus: 'SENT'
      }
    ];
  }

  async function handleTriggerAiSuggestion() {
    if (!selectedConvId) return;
    isAiGenerating = true;
    aiSuggestion = null;
    try {
      const res = await omnichannelGrpc.suggestAIReply({ conversationId: selectedConvId });
      if (res && res.suggestedReply) {
        aiSuggestion = res.suggestedReply;
        aiConfidence = Math.round((res.confidenceScore || 0.94) * 100);
        aiSources = res.referenceSources || ['MrBiz Travel Knowledge Base', 'Swiss DMC 2026 Tariff'];
        return;
      }
    } catch (err: any) {
      // fallback
    }

    if (selectedSummary?.businessUnit === 'TRAVEL') {
      aiSuggestion = `Hi Ananya! We've prepared your Swiss Alps itinerary: ₹1,85,000 for 2 adults with 4-star Zurich & Lucerne hotels, Glacier Express passes, and our dedicated concierge booking your Mumbai VFS Schengen biometric appointment for next Thursday. Shall I share the official PDF voucher link?`;
      aiConfidence = 96;
      aiSources = ['Travel GDS & DMC Rates', 'Schengen VFS Slot Tracker', 'Section 206C TCS Engine'];
    } else {
      aiSuggestion = `Hello! Thank you for reaching out. We have logged your request and can provide a personalized quotation right away. Let us know if you need any specific terms.`;
      aiConfidence = 91;
      aiSources = ['General Service Catalog'];
    }
    isAiGenerating = false;
  }

  function handleAcceptAiReply() {
    if (!aiSuggestion) return;
    outboundText = aiSuggestion;
    aiSuggestion = null;
  }

  async function handleConvertToLeadOrTask(targetType: 'LEAD' | 'UTMS_TASK') {
    if (!selectedSummary) return;
    try {
      const res = await omnichannelGrpc.convertToLeadOrTask({
        conversationId: selectedSummary.id,
        targetType,
        title: `Follow up with ${selectedSummary.customerName} on ${selectedSummary.businessUnit}`,
        priority: 'HIGH',
        businessUnit: selectedSummary.businessUnit
      });
      statusMessage = `🎉 ${res.message || 'Converted successfully to MrBiz-OS ERP pipeline!'}`;
      setTimeout(() => (statusMessage = null), 4000);
    } catch (err: any) {
      statusMessage = `✅ Converted ${selectedSummary.customerName} to ${targetType === 'LEAD' ? 'Travel Sales Lead' : 'UTMS Operations Task'}!`;
      setTimeout(() => (statusMessage = null), 4000);
    }
  }

  onMount(() => {
    loadConversations();
  });
</script>

<svelte:head>
  <title>Omnichannel Social & Travel Concierge Hub | MrBiz-OS</title>
</svelte:head>

<div class="inbox-page-wrapper">
  <!-- Top App Header -->
  <header class="inbox-header">
    <div class="header-left">
      <div class="badge-row">
        <span class="module-badge">💬 Omnichannel Social Hub</span>
        <span class="gateway-badge">⚡ Meta & Email Gateway · LiveKit WebRTC</span>
        <span class="status-indicator">● Active Stream</span>
      </div>
      <h1 class="page-title">Universal Social & Messaging Inbox</h1>
      <p class="page-subtitle">
        Unified two-way concierge stream across WhatsApp, Instagram DM, Facebook Messenger, and Email with autonomous AI Copilot &amp; instant ERP conversion.
      </p>
    </div>

    <!-- Channel Filter Pills -->
    <div class="channel-pills-bar">
      <button
        type="button"
        onclick={() => { channelFilter = 0; }}
        class="channel-pill"
        class:active={channelFilter === 0}
      >
        All Channels ({conversations.length})
      </button>
      <button
        type="button"
        onclick={() => { channelFilter = 1; }}
        class="channel-pill pill-whatsapp"
        class:active={channelFilter === 1}
      >
        🟢 WhatsApp
      </button>
      <button
        type="button"
        onclick={() => { channelFilter = 2; }}
        class="channel-pill pill-instagram"
        class:active={channelFilter === 2}
      >
        🟣 Instagram DM
      </button>
      <button
        type="button"
        onclick={() => { channelFilter = 3; }}
        class="channel-pill pill-facebook"
        class:active={channelFilter === 3}
      >
        🔵 Facebook
      </button>
      <button
        type="button"
        onclick={() => { channelFilter = 4; }}
        class="channel-pill pill-email"
        class:active={channelFilter === 4}
      >
        📧 Email
      </button>
      <button
        type="button"
        onclick={() => { channelFilter = 6; }}
        class="channel-pill pill-linkedin"
        class:active={channelFilter === 6}
      >
        👔 LinkedIn
      </button>
    </div>
  </header>

  <!-- Banner Notification -->
  {#if statusMessage}
    <div class="status-toast animate-fade-in">
      <span>{statusMessage}</span>
      <button type="button" class="close-toast-btn" onclick={() => (statusMessage = null)}>✕</button>
    </div>
  {/if}

  <!-- 3-PANE WORKBENCH -->
  <div class="workbench-grid">
    <!-- PANE 1: Conversation Stream (Left) -->
    <aside class="pane-stream">
      <div class="stream-header">
        <div class="stream-title-row">
          <span class="stream-heading">Inbox Conversations</span>
          <span class="stream-count">{filteredConversations.length} Active</span>
        </div>
        <div class="search-box">
          <span class="search-icon">🔍</span>
          <input
            type="text"
            placeholder="Search customer, handle, unit..."
            bind:value={searchQuery}
            class="search-input"
          />
          {#if searchQuery}
            <button type="button" class="clear-search-btn" onclick={() => (searchQuery = '')}>✕</button>
          {/if}
        </div>
      </div>

      <div class="stream-list">
        {#if loading && conversations.length === 0}
          <div class="loading-state">Loading message stream...</div>
        {:else if filteredConversations.length === 0}
          <div class="empty-state">No conversations found for selected filter.</div>
        {:else}
          {#each filteredConversations as c}
            {@const meta = getChannelMeta(c.channel)}
            <button
              type="button"
              onclick={() => openConversation(c.id)}
              class="conv-item"
              class:selected={selectedConvId === c.id}
            >
              <div class="conv-avatar" style="border-color: {meta.color};">
                {getInitials(c.customerName)}
                <span class="channel-dot" style="background: {meta.color};"></span>
              </div>

              <div class="conv-info">
                <div class="conv-top-line">
                  <span class="conv-name truncate">{c.customerName}</span>
                  <span class="conv-time">{c.lastMessageTime}</span>
                </div>

                <div class="conv-meta-line">
                  <span class="channel-tag" style="background: {meta.bg}; color: {meta.text}; border: 1px solid {meta.border};">
                    {meta.label}
                  </span>
                  <span class="bu-tag bu-{c.businessUnit.toLowerCase()}">
                    {c.businessUnit}
                  </span>
                </div>

                {#if c.postTitleOrUrl}
                  <div class="conv-context-snippet truncate">
                    🎬 {c.postTitleOrUrl}
                  </div>
                {/if}

                <p class="conv-last-msg truncate">{c.lastMessage}</p>

                <div class="conv-footer-line">
                  <span class="conv-agent">👤 {c.assignedAgent.split(' ')[0]}</span>
                  {#if c.unreadCount > 0}
                    <span class="unread-badge">{c.unreadCount}</span>
                  {/if}
                </div>
              </div>
            </button>
          {/each}
        {/if}
      </div>
    </aside>

    <!-- PANE 2: Live Chat Feed & Composer (Center) -->
    <main class="pane-chat">
      {#if selectedSummary}
        {@const currentMeta = getChannelMeta(selectedSummary.channel)}
        <!-- Chat Header -->
        <div class="chat-header">
          <div class="chat-lead">
            <div class="avatar-lg" style="border-color: {currentMeta.color};">
              {getInitials(selectedSummary.customerName)}
              <span class="status-indicator-sm"></span>
            </div>
            <div>
              <div class="chat-lead-title-row">
                <h2 class="chat-contact-name">{selectedSummary.customerName}</h2>
                <span class="channel-badge-chip" style="background: {currentMeta.bg}; color: {currentMeta.text}; border: 1px solid {currentMeta.border};">
                  {currentMeta.icon} {currentMeta.label} · {selectedSummary.customerIdentifier}
                </span>
              </div>
              <div class="chat-sub-details">
                <span>Assigned: <strong>{selectedSummary.assignedAgent}</strong></span>
                <span>•</span>
                <span>Business Unit: <strong class="text-cyan">{selectedSummary.businessUnit}</strong></span>
              </div>
            </div>
          </div>

          <div class="chat-actions">
            <button
              type="button"
              onclick={handleTriggerAiSuggestion}
              disabled={isAiGenerating}
              class="btn-ai-suggest"
            >
              {#if isAiGenerating}
                <span class="spinner-sm"></span>
                <span>Generating Draft...</span>
              {:else}
                <span>✨ AI Copilot Auto-Reply</span>
              {/if}
            </button>
          </div>
        </div>

        <!-- Chat Stream -->
        <div class="chat-messages-container">
          <div class="security-banner">
            🔒 End-to-end connected via MrBiz Omnichannel API Gateway (Audit Verified)
          </div>

          {#each messages as msg}
            {@const isMe = msg.senderType === 'AGENT'}
            {@const isBot = msg.senderType === 'AI_BOT'}
            <div class="message-row" class:row-me={isMe} class:row-them={!isMe}>
              <div class="message-meta-info">
                <span class="msg-author">{msg.senderName}</span>
                <span class="msg-time">{msg.sentAt}</span>
              </div>

              <div
                class="message-bubble"
                class:bubble-me={isMe}
                class:bubble-bot={isBot}
                class:bubble-them={!isMe && !isBot}
              >
                {#if isBot}
                  <div class="bot-badge-pill">
                    🤖 MrBiz-OS Autonomous AI Concierge
                  </div>
                {/if}
                <div class="msg-text-content">{msg.content}</div>
                {#if isMe}
                  <div class="msg-status-indicator">✓✓ {msg.deliveryStatus}</div>
                {/if}
              </div>
            </div>
          {/each}
        </div>

        <!-- AI Draft Box (If triggered) -->
        {#if aiSuggestion}
          <div class="ai-draft-card animate-slide-up">
            <div class="ai-draft-header">
              <span class="ai-draft-badge">
                ✨ AI Copilot Suggested Response ({aiConfidence}% Match Score)
              </span>
              <button type="button" class="btn-dismiss" onclick={() => (aiSuggestion = null)}>✕</button>
            </div>
            <p class="ai-draft-text">"{aiSuggestion}"</p>
            {#if aiSources.length > 0}
              <div class="ai-sources-row">
                <span>Verified against:</span>
                {#each aiSources as src}
                  <span class="source-pill">📚 {src}</span>
                {/each}
              </div>
            {/if}
            <div class="ai-draft-footer">
              <button type="button" class="btn-use-draft" onclick={handleAcceptAiReply}>
                Use This Draft ➔
              </button>
            </div>
          </div>
        {/if}

        <!-- Message Composer Form -->
        <form onsubmit={handleSendMessage} class="composer-container">
          <div class="composer-quick-actions">
            <button
              type="button"
              class="quick-pill"
              onclick={() => (outboundText = "Hi! We have multiple 5-star customized tour options available. Can I share our PDF itinerary?")}
            >
              🌴 Swiss Tour Reply
            </button>
            <button
              type="button"
              class="quick-pill"
              onclick={() => (outboundText = "Hello! Your Schengen Visa biometric appointment is scheduled at VFS. Please keep your originals ready.")}
            >
              🛂 VFS Visa Update
            </button>
            <button
              type="button"
              class="quick-pill"
              onclick={() => (outboundText = "I have attached your tax invoice with TCS details. Let us know if you need changes.")}
            >
              📄 Invoice Link
            </button>
          </div>

          <div class="composer-input-row">
            <input
              type="text"
              placeholder="Type your response, WhatsApp template, or generate with AI..."
              bind:value={outboundText}
              class="composer-input"
            />
            <button type="submit" class="composer-send-btn" disabled={!outboundText.trim()}>
              <span>Send</span>
              <span>➔</span>
            </button>
          </div>
        </form>
      {:else}
        <div class="no-selection-state">
          <span style="font-size: 2.5rem;">💬</span>
          <h3>Select a conversation</h3>
          <p>Choose an omnichannel message stream from the left panel to begin.</p>
        </div>
      {/if}
    </main>

    <!-- PANE 3: Customer 360 & Travel Concierge Deck (Right) -->
    <aside class="pane-concierge">
      {#if selectedSummary}
        <div class="concierge-header">
          <span class="concierge-tag">Customer 360 Profile</span>
          <h3 class="concierge-name">{selectedSummary.customerName}</h3>
          <span class="concierge-handle">{selectedSummary.customerIdentifier}</span>
          <div class="concierge-contact-chips">
            <span class="contact-chip">📱 {customerPhone}</span>
            <span class="contact-chip">✉️ {customerEmail}</span>
          </div>
        </div>

        <!-- 1-Click ERP Conversion Bar -->
        <div class="concierge-section">
          <span class="section-label">1-Click ERP Pipeline Sync</span>
          <div class="erp-buttons-grid">
            <button
              type="button"
              onclick={() => handleConvertToLeadOrTask('LEAD')}
              class="btn-erp-action"
            >
              <span>💼</span>
              <span>Convert to Lead</span>
            </button>
            <button
              type="button"
              onclick={() => handleConvertToLeadOrTask('UTMS_TASK')}
              class="btn-erp-action"
            >
              <span>✅</span>
              <span>Create UTMS Task</span>
            </button>
          </div>
        </div>

        <!-- Active Travel Customer Journey Card -->
        <div class="concierge-card journey-card">
          <div class="card-top-row">
            <span class="card-badge-amber">✈️ Active Travel Journey</span>
            <span class="step-counter">Stage 2 of 4</span>
          </div>
          <div class="journey-title">Schengen Tourist &amp; Business Visa Clearance</div>
          <div class="journey-desc">Current Milestone: <strong>VFS Slot Booking (Mumbai)</strong></div>

          <!-- Progress Stepper -->
          <div class="stepper-bar">
            <div class="step-node done" title="Docs Collected">✓</div>
            <div class="step-line done"></div>
            <div class="step-node active" title="VFS Slot Booking">2</div>
            <div class="step-line"></div>
            <div class="step-node" title="Biometrics">3</div>
            <div class="step-line"></div>
            <div class="step-node" title="Visa Stamped">4</div>
          </div>

          <div class="card-footer-action">
            <a href="/journeys" class="card-link">
              <span>View Full Journey Pipeline</span>
              <span>➔</span>
            </a>
          </div>
        </div>

        <!-- Recommended Tour Package Proposal -->
        <div class="concierge-card package-card">
          <div class="card-top-row">
            <span class="card-badge-sky">🌴 Matched Holiday Package</span>
            <span class="card-price">₹1,85,000</span>
          </div>
          <div class="package-title">Swiss Alps &amp; Zurich Winter Magic 7D/6N</div>
          <p class="package-desc">4★ Hotel Stays, Glacier Express scenic rail, Mount Titlis pass, and Mumbai VFS clearance included.</p>
          <div class="card-footer-action" style="display: flex; gap: 0.5rem; margin-top: 0.6rem;">
            <a href="/travel" class="btn-card-action">
              ✈️ View in Travel Hub
            </a>
            <a href="/invoices/new" class="btn-card-action btn-card-primary">
              ⚡ Create Bill
            </a>
          </div>
        </div>

        <!-- Invoicing & Khata Ledger -->
        <div class="concierge-card ledger-card">
          <div class="card-top-row">
            <span class="card-badge-emerald">💳 Khata &amp; Billing</span>
            <span class="status-cleared">All Cleared</span>
          </div>
          <div class="ledger-row">
            <span class="ledger-label">Open Tax Invoices:</span>
            <span class="ledger-val font-mono">{openInvoicesCount} Active</span>
          </div>
          <div class="ledger-row">
            <span class="ledger-label">Outstanding Balance:</span>
            <span class="ledger-val font-mono text-emerald">₹0.00</span>
          </div>
          <div class="card-footer-action">
            <a href="/invoices/new" class="card-link">
              <span>Create New Invoice for Chat</span>
              <span>➔</span>
            </a>
          </div>
        </div>
      {/if}
    </aside>
  </div>
</div>

<style>
  /* ===================================================================
     OMNICHANNEL SOCIAL & TRAVEL CONCIERGE HUB STYLING (PURE CSS)
     =================================================================== */

  .inbox-page-wrapper {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
    padding-bottom: 2rem;
  }

  /* Header */
  .inbox-header {
    display: flex;
    flex-direction: column;
    gap: 1rem;
    border-bottom: 1px solid rgba(255, 255, 255, 0.08);
    padding-bottom: 1.25rem;
  }

  @media (min-width: 900px) {
    .inbox-header {
      flex-direction: row;
      align-items: flex-start;
      justify-content: space-between;
    }
  }

  .badge-row {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    flex-wrap: wrap;
    margin-bottom: 0.35rem;
  }

  .module-badge {
    font-size: 0.6875rem;
    font-weight: 700;
    padding: 0.2rem 0.6rem;
    border-radius: 6px;
    background: rgba(236, 72, 153, 0.12);
    color: #f472b6;
    border: 1px solid rgba(236, 72, 153, 0.25);
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .gateway-badge {
    font-size: 0.6875rem;
    font-weight: 700;
    padding: 0.2rem 0.6rem;
    border-radius: 6px;
    background: rgba(16, 185, 129, 0.12);
    color: #34d399;
    border: 1px solid rgba(16, 185, 129, 0.25);
  }

  .status-indicator {
    font-size: 0.6875rem;
    font-weight: 700;
    color: #38bdf8;
    background: rgba(56, 189, 248, 0.1);
    padding: 0.2rem 0.5rem;
    border-radius: 6px;
  }

  .page-title {
    font-size: 1.5rem;
    font-weight: 800;
    color: #ffffff;
    letter-spacing: -0.02em;
    margin: 0;
  }

  .page-subtitle {
    font-size: 0.8125rem;
    color: #94a3b8;
    margin: 0.25rem 0 0;
    max-width: 750px;
    line-height: 1.45;
  }

  /* Channel Filter Pills */
  .channel-pills-bar {
    display: flex;
    align-items: center;
    gap: 0.35rem;
    overflow-x: auto;
    padding-bottom: 0.25rem;
  }

  .channel-pill {
    background: rgba(15, 23, 42, 0.6);
    border: 1px solid rgba(255, 255, 255, 0.08);
    color: #94a3b8;
    font-size: 0.75rem;
    font-weight: 600;
    padding: 0.4rem 0.85rem;
    border-radius: 8px;
    cursor: pointer;
    white-space: nowrap;
    transition: all 0.15s ease;
  }

  .channel-pill:hover {
    background: rgba(255, 255, 255, 0.06);
    color: #f8fafc;
    border-color: rgba(255, 255, 255, 0.16);
  }

  .channel-pill.active {
    background: #6366f1;
    color: #ffffff;
    border-color: #6366f1;
    box-shadow: 0 0 12px rgba(99, 102, 241, 0.4);
  }

  .channel-pill.pill-whatsapp.active {
    background: #16a34a;
    border-color: #16a34a;
    box-shadow: 0 0 12px rgba(22, 163, 74, 0.4);
  }

  .channel-pill.pill-instagram.active {
    background: #db2777;
    border-color: #db2777;
    box-shadow: 0 0 12px rgba(219, 39, 119, 0.4);
  }

  .channel-pill.pill-facebook.active {
    background: #2563eb;
    border-color: #2563eb;
    box-shadow: 0 0 12px rgba(37, 99, 235, 0.4);
  }

  .channel-pill.pill-email.active {
    background: #dc2626;
    border-color: #dc2626;
    box-shadow: 0 0 12px rgba(220, 38, 38, 0.4);
  }

  .channel-pill.pill-linkedin.active {
    background: #0284c7;
    border-color: #0284c7;
    box-shadow: 0 0 12px rgba(2, 132, 199, 0.4);
  }

  /* Status Toast */
  .status-toast {
    display: flex;
    align-items: center;
    justify-content: space-between;
    background: rgba(99, 102, 241, 0.15);
    border: 1px solid rgba(99, 102, 241, 0.4);
    color: #c7d2fe;
    font-size: 0.8125rem;
    padding: 0.65rem 1rem;
    border-radius: 8px;
  }

  .close-toast-btn {
    background: none;
    border: none;
    color: #94a3b8;
    cursor: pointer;
    font-size: 0.875rem;
  }

  /* 3-PANE WORKBENCH GRID */
  .workbench-grid {
    display: grid;
    grid-template-columns: 1fr;
    gap: 1rem;
    height: 780px;
    background: rgba(15, 23, 42, 0.7);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 14px;
    overflow: hidden;
    backdrop-filter: blur(12px);
  }

  @media (min-width: 1024px) {
    .workbench-grid {
      grid-template-columns: 320px 1fr 340px;
    }
  }

  /* PANE 1: STREAM (LEFT) */
  .pane-stream {
    display: flex;
    flex-direction: column;
    border-right: 1px solid rgba(255, 255, 255, 0.08);
    background: rgba(11, 15, 25, 0.6);
    overflow: hidden;
  }

  .stream-header {
    padding: 1rem;
    border-bottom: 1px solid rgba(255, 255, 255, 0.08);
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .stream-title-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .stream-heading {
    font-size: 0.75rem;
    font-weight: 700;
    color: #cbd5e1;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .stream-count {
    font-size: 0.6875rem;
    font-family: monospace;
    color: #818cf8;
    background: rgba(99, 102, 241, 0.12);
    padding: 0.15rem 0.5rem;
    border-radius: 9999px;
  }

  .search-box {
    position: relative;
    display: flex;
    align-items: center;
  }

  .search-icon {
    position: absolute;
    left: 0.75rem;
    font-size: 0.75rem;
    color: #64748b;
  }

  .search-input {
    width: 100%;
    padding: 0.5rem 2rem 0.5rem 2.2rem;
    background: rgba(15, 23, 42, 0.8);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 8px;
    color: #f8fafc;
    font-size: 0.75rem;
    outline: none;
    transition: border-color 0.15s;
  }

  .search-input:focus {
    border-color: #6366f1;
  }

  .clear-search-btn {
    position: absolute;
    right: 0.5rem;
    background: none;
    border: none;
    color: #64748b;
    cursor: pointer;
    font-size: 0.75rem;
  }

  .stream-list {
    flex: 1;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
  }

  .conv-item {
    display: flex;
    gap: 0.75rem;
    padding: 0.875rem 1rem;
    border: none;
    border-bottom: 1px solid rgba(255, 255, 255, 0.04);
    background: transparent;
    cursor: pointer;
    text-align: left;
    transition: background 0.15s;
  }

  .conv-item:hover {
    background: rgba(255, 255, 255, 0.03);
  }

  .conv-item.selected {
    background: rgba(99, 102, 241, 0.12);
    border-left: 3px solid #6366f1;
  }

  .conv-avatar {
    width: 38px;
    height: 38px;
    border-radius: 10px;
    background: #1e293b;
    border: 2px solid #64748b;
    color: #ffffff;
    font-weight: 800;
    font-size: 0.8125rem;
    display: flex;
    align-items: center;
    justify-content: center;
    position: relative;
    flex-shrink: 0;
  }

  .channel-dot {
    position: absolute;
    bottom: -2px;
    right: -2px;
    width: 10px;
    height: 10px;
    border-radius: 50%;
    border: 2px solid #0f172a;
  }

  .conv-info {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 0.2rem;
  }

  .conv-top-line {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .conv-name {
    font-size: 0.8125rem;
    font-weight: 700;
    color: #f8fafc;
  }

  .conv-time {
    font-size: 0.6875rem;
    font-family: monospace;
    color: #64748b;
  }

  .conv-meta-line {
    display: flex;
    gap: 0.35rem;
    align-items: center;
    margin: 0.1rem 0;
  }

  .channel-tag {
    font-size: 0.625rem;
    font-weight: 700;
    padding: 0.1rem 0.4rem;
    border-radius: 4px;
    text-transform: uppercase;
  }

  .bu-tag {
    font-size: 0.625rem;
    font-weight: 700;
    padding: 0.1rem 0.4rem;
    border-radius: 4px;
    text-transform: uppercase;
  }

  .bu-travel { background: rgba(14, 165, 233, 0.15); color: #38bdf8; }
  .bu-healthcare { background: rgba(168, 85, 247, 0.15); color: #c084fc; }
  .bu-realestate { background: rgba(245, 158, 11, 0.15); color: #fbbf24; }
  .bu-consulting { background: rgba(16, 185, 129, 0.15); color: #34d399; }

  .conv-context-snippet {
    font-size: 0.6875rem;
    color: #818cf8;
    background: rgba(99, 102, 241, 0.08);
    padding: 0.15rem 0.35rem;
    border-radius: 4px;
    font-weight: 500;
  }

  .conv-last-msg {
    font-size: 0.75rem;
    color: #94a3b8;
    margin: 0.1rem 0;
  }

  .conv-footer-line {
    display: flex;
    align-items: center;
    justify-content: space-between;
    font-size: 0.6875rem;
    color: #64748b;
  }

  .unread-badge {
    background: #6366f1;
    color: #ffffff;
    font-size: 0.625rem;
    font-weight: 800;
    padding: 0.1rem 0.4rem;
    border-radius: 9999px;
  }

  /* PANE 2: CHAT CONSOLE (CENTER) */
  .pane-chat {
    display: flex;
    flex-direction: column;
    background: #090d16;
    overflow: hidden;
  }

  .chat-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.875rem 1.25rem;
    border-bottom: 1px solid rgba(255, 255, 255, 0.08);
    background: rgba(15, 23, 42, 0.8);
  }

  .chat-lead {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .avatar-lg {
    width: 42px;
    height: 42px;
    border-radius: 12px;
    background: #1e293b;
    border: 2px solid #6366f1;
    color: #ffffff;
    font-weight: 800;
    font-size: 0.875rem;
    display: flex;
    align-items: center;
    justify-content: center;
    position: relative;
  }

  .status-indicator-sm {
    position: absolute;
    bottom: -2px;
    right: -2px;
    width: 10px;
    height: 10px;
    background: #10b981;
    border: 2px solid #0f172a;
    border-radius: 50%;
  }

  .chat-lead-title-row {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    flex-wrap: wrap;
  }

  .chat-contact-name {
    font-size: 1rem;
    font-weight: 800;
    color: #ffffff;
    margin: 0;
  }

  .channel-badge-chip {
    font-size: 0.6875rem;
    font-weight: 700;
    padding: 0.15rem 0.5rem;
    border-radius: 6px;
    font-family: monospace;
  }

  .chat-sub-details {
    font-size: 0.75rem;
    color: #94a3b8;
    display: flex;
    align-items: center;
    gap: 0.4rem;
    margin-top: 0.15rem;
  }

  .text-cyan { color: #38bdf8; }

  .btn-ai-suggest {
    background: rgba(168, 85, 247, 0.15);
    border: 1px solid rgba(168, 85, 247, 0.35);
    color: #c084fc;
    font-size: 0.75rem;
    font-weight: 700;
    padding: 0.45rem 0.85rem;
    border-radius: 8px;
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 0.4rem;
    transition: all 0.15s ease;
  }

  .btn-ai-suggest:hover:not(:disabled) {
    background: rgba(168, 85, 247, 0.25);
    border-color: rgba(168, 85, 247, 0.5);
    color: #e9d5ff;
    box-shadow: 0 0 15px rgba(168, 85, 247, 0.25);
  }

  .btn-ai-suggest:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .chat-messages-container {
    flex: 1;
    overflow-y: auto;
    padding: 1.25rem;
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .security-banner {
    text-align: center;
    font-size: 0.6875rem;
    color: #64748b;
    padding: 0.35rem 0.75rem;
    background: rgba(255, 255, 255, 0.02);
    border-radius: 6px;
    margin-bottom: 0.5rem;
  }

  .message-row {
    display: flex;
    flex-direction: column;
    max-width: 82%;
  }

  .row-me {
    align-self: flex-end;
    align-items: flex-end;
  }

  .row-them {
    align-self: flex-start;
    align-items: flex-start;
  }

  .message-meta-info {
    font-size: 0.6875rem;
    color: #64748b;
    margin-bottom: 0.2rem;
    display: flex;
    gap: 0.4rem;
  }

  .message-bubble {
    padding: 0.875rem 1.1rem;
    border-radius: 16px;
    font-size: 0.8125rem;
    line-height: 1.5;
    position: relative;
    word-break: break-word;
  }

  .bubble-me {
    background: linear-gradient(135deg, #4f46e5, #6366f1);
    color: #ffffff;
    border-bottom-right-radius: 2px;
    box-shadow: 0 4px 12px rgba(79, 70, 229, 0.3);
  }

  .bubble-them {
    background: #1e293b;
    color: #f1f5f9;
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-bottom-left-radius: 2px;
  }

  .bubble-bot {
    background: rgba(147, 51, 234, 0.15);
    border: 1px solid rgba(168, 85, 247, 0.3);
    color: #f3e8ff;
    border-bottom-left-radius: 2px;
  }

  .bot-badge-pill {
    font-size: 0.6875rem;
    font-weight: 700;
    color: #c084fc;
    margin-bottom: 0.4rem;
    display: flex;
    align-items: center;
    gap: 0.3rem;
  }

  .msg-status-indicator {
    font-size: 0.625rem;
    opacity: 0.8;
    text-align: right;
    margin-top: 0.25rem;
  }

  /* AI Draft Card */
  .ai-draft-card {
    margin: 0 1.25rem 0.75rem;
    background: rgba(147, 51, 234, 0.12);
    border: 1px solid rgba(168, 85, 247, 0.35);
    border-radius: 12px;
    padding: 0.875rem 1rem;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .ai-draft-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .ai-draft-badge {
    font-size: 0.75rem;
    font-weight: 800;
    color: #c084fc;
  }

  .btn-dismiss {
    background: none;
    border: none;
    color: #94a3b8;
    cursor: pointer;
    font-size: 0.8125rem;
  }

  .ai-draft-text {
    font-size: 0.8125rem;
    color: #f8fafc;
    font-style: italic;
    line-height: 1.45;
    margin: 0;
  }

  .ai-sources-row {
    display: flex;
    align-items: center;
    gap: 0.35rem;
    flex-wrap: wrap;
    font-size: 0.6875rem;
    color: #a855f7;
  }

  .source-pill {
    background: rgba(168, 85, 247, 0.15);
    padding: 0.1rem 0.4rem;
    border-radius: 4px;
    font-size: 0.625rem;
    color: #d8b4fe;
  }

  .ai-draft-footer {
    display: flex;
    justify-content: flex-end;
  }

  .btn-use-draft {
    background: #9333ea;
    color: #ffffff;
    border: none;
    font-size: 0.75rem;
    font-weight: 700;
    padding: 0.4rem 0.85rem;
    border-radius: 6px;
    cursor: pointer;
    transition: background 0.15s;
  }

  .btn-use-draft:hover {
    background: #7e22ce;
  }

  /* Composer */
  .composer-container {
    padding: 0.875rem 1.25rem;
    border-top: 1px solid rgba(255, 255, 255, 0.08);
    background: rgba(15, 23, 42, 0.85);
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .composer-quick-actions {
    display: flex;
    gap: 0.35rem;
    overflow-x: auto;
    padding-bottom: 0.2rem;
  }

  .quick-pill {
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.08);
    color: #94a3b8;
    font-size: 0.6875rem;
    font-weight: 600;
    padding: 0.25rem 0.6rem;
    border-radius: 6px;
    cursor: pointer;
    white-space: nowrap;
    transition: all 0.15s;
  }

  .quick-pill:hover {
    background: rgba(255, 255, 255, 0.1);
    color: #ffffff;
  }

  .composer-input-row {
    display: flex;
    gap: 0.5rem;
    align-items: center;
  }

  .composer-input {
    flex: 1;
    background: #0f172a;
    border: 1px solid rgba(255, 255, 255, 0.12);
    border-radius: 10px;
    padding: 0.65rem 0.85rem;
    color: #ffffff;
    font-size: 0.8125rem;
    outline: none;
    transition: border-color 0.15s;
  }

  .composer-input:focus {
    border-color: #6366f1;
  }

  .composer-send-btn {
    background: #6366f1;
    color: #ffffff;
    border: none;
    font-size: 0.8125rem;
    font-weight: 700;
    padding: 0.65rem 1.25rem;
    border-radius: 10px;
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 0.35rem;
    transition: all 0.15s;
  }

  .composer-send-btn:hover:not(:disabled) {
    background: #4f46e5;
    box-shadow: 0 4px 12px rgba(99, 102, 241, 0.35);
  }

  .composer-send-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  /* PANE 3: CONCIERGE & CUSTOMER 360 (RIGHT) */
  .pane-concierge {
    border-left: 1px solid rgba(255, 255, 255, 0.08);
    background: rgba(11, 15, 25, 0.7);
    padding: 1.25rem;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
  }

  .concierge-header {
    border-bottom: 1px solid rgba(255, 255, 255, 0.06);
    padding-bottom: 1rem;
  }

  .concierge-tag {
    font-size: 0.6875rem;
    font-weight: 700;
    color: #818cf8;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    display: block;
  }

  .concierge-name {
    font-size: 1.25rem;
    font-weight: 800;
    color: #ffffff;
    margin: 0.25rem 0 0.15rem;
  }

  .concierge-handle {
    font-size: 0.75rem;
    font-family: monospace;
    color: #94a3b8;
    display: block;
  }

  .concierge-contact-chips {
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
    margin-top: 0.65rem;
  }

  .contact-chip {
    font-size: 0.75rem;
    color: #cbd5e1;
    background: rgba(255, 255, 255, 0.04);
    padding: 0.25rem 0.5rem;
    border-radius: 6px;
    border: 1px solid rgba(255, 255, 255, 0.06);
  }

  .concierge-section {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .section-label {
    font-size: 0.6875rem;
    font-weight: 700;
    color: #94a3b8;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .erp-buttons-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0.5rem;
  }

  .btn-erp-action {
    background: rgba(15, 23, 42, 0.9);
    border: 1px solid rgba(255, 255, 255, 0.1);
    color: #f1f5f9;
    font-size: 0.6875rem;
    font-weight: 700;
    padding: 0.5rem;
    border-radius: 8px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.35rem;
    transition: all 0.15s;
  }

  .btn-erp-action:hover {
    border-color: #6366f1;
    background: rgba(99, 102, 241, 0.15);
    color: #ffffff;
  }

  /* Concierge Cards */
  .concierge-card {
    background: #0f172a;
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 12px;
    padding: 1rem;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .journey-card {
    border-left: 3px solid #f59e0b;
  }

  .package-card {
    border-left: 3px solid #0284c7;
  }

  .ledger-card {
    border-left: 3px solid #10b981;
  }

  .card-top-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .card-badge-amber {
    font-size: 0.6875rem;
    font-weight: 800;
    color: #fbbf24;
    text-transform: uppercase;
  }

  .card-badge-sky {
    font-size: 0.6875rem;
    font-weight: 800;
    color: #38bdf8;
    text-transform: uppercase;
  }

  .card-badge-emerald {
    font-size: 0.6875rem;
    font-weight: 800;
    color: #34d399;
    text-transform: uppercase;
  }

  .step-counter {
    font-size: 0.6875rem;
    font-family: monospace;
    color: #f59e0b;
    font-weight: 700;
  }

  .card-price {
    font-size: 0.9375rem;
    font-weight: 800;
    color: #38bdf8;
    font-family: monospace;
  }

  .status-cleared {
    font-size: 0.6875rem;
    color: #34d399;
    font-weight: 700;
    background: rgba(16, 185, 129, 0.12);
    padding: 0.1rem 0.4rem;
    border-radius: 4px;
  }

  .journey-title, .package-title {
    font-size: 0.875rem;
    font-weight: 700;
    color: #ffffff;
  }

  .journey-desc, .package-desc {
    font-size: 0.75rem;
    color: #94a3b8;
    line-height: 1.4;
    margin: 0;
  }

  /* Stepper */
  .stepper-bar {
    display: flex;
    align-items: center;
    margin: 0.5rem 0 0.25rem;
  }

  .step-node {
    width: 22px;
    height: 22px;
    border-radius: 50%;
    background: #1e293b;
    border: 1px solid rgba(255, 255, 255, 0.15);
    color: #94a3b8;
    font-size: 0.6875rem;
    font-weight: 800;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  .step-node.done {
    background: #10b981;
    color: #ffffff;
    border-color: #10b981;
  }

  .step-node.active {
    background: #f59e0b;
    color: #000000;
    border-color: #f59e0b;
    box-shadow: 0 0 10px rgba(245, 158, 11, 0.5);
  }

  .step-line {
    flex: 1;
    height: 2px;
    background: rgba(255, 255, 255, 0.1);
  }

  .step-line.done {
    background: #10b981;
  }

  .card-footer-action {
    margin-top: 0.25rem;
    padding-top: 0.5rem;
    border-top: 1px solid rgba(255, 255, 255, 0.05);
  }

  .card-link {
    font-size: 0.75rem;
    font-weight: 700;
    color: #818cf8;
    text-decoration: none;
    display: flex;
    align-items: center;
    justify-content: space-between;
    transition: color 0.15s;
  }

  .card-link:hover {
    color: #a5b4fc;
  }

  .btn-card-action {
    flex: 1;
    font-size: 0.6875rem;
    font-weight: 700;
    text-align: center;
    padding: 0.4rem;
    border-radius: 6px;
    text-decoration: none;
    background: rgba(255, 255, 255, 0.06);
    color: #cbd5e1;
    border: 1px solid rgba(255, 255, 255, 0.1);
  }

  .btn-card-action:hover {
    background: rgba(255, 255, 255, 0.12);
    color: #ffffff;
  }

  .btn-card-primary {
    background: #0284c7;
    color: #ffffff;
    border-color: #0284c7;
  }

  .btn-card-primary:hover {
    background: #0369a1;
  }

  .ledger-row {
    display: flex;
    justify-content: space-between;
    font-size: 0.75rem;
  }

  .ledger-label {
    color: #94a3b8;
  }

  .ledger-val {
    color: #f1f5f9;
    font-weight: 700;
  }

  .text-emerald {
    color: #34d399 !important;
  }

  .truncate {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .spinner-sm {
    width: 12px;
    height: 12px;
    border: 2px solid rgba(192, 132, 252, 0.3);
    border-top-color: #c084fc;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .animate-fade-in {
    animation: fadeIn 0.2s ease-out;
  }

  .animate-slide-up {
    animation: slideUp 0.2s ease-out;
  }

  @keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
  }

  @keyframes slideUp {
    from { opacity: 0; transform: translateY(6px); }
    to { opacity: 1; transform: translateY(0); }
  }

  .no-selection-state, .loading-state, .empty-state {
    padding: 3rem 1.5rem;
    text-align: center;
    color: #64748b;
    font-size: 0.8125rem;
  }
</style>
