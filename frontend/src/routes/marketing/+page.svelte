<script lang="ts">
	import { toast } from '$lib/stores/toast.svelte';

	interface CampaignItem {
		id: string;
		name: string;
		channel: 'whatsapp' | 'instagram' | 'linkedin' | 'email';
		businessUnit: string;
		audience: string;
		sentCount: number;
		deliveryRate: string;
		clickRate: string;
		status: 'COMPLETED' | 'SCHEDULED' | 'ACTIVE';
		date: string;
	}

	let campaigns = $state<CampaignItem[]>([
		{
			id: 'CMP-WA-01',
			name: 'Kashmir Early Winter Snow Early Bird (15% Off)',
			channel: 'whatsapp',
			businessUnit: 'TRAVEL',
			audience: 'Past Holiday Inquirers (1,420 Contacts)',
			sentCount: 1420,
			deliveryRate: '98.6%',
			clickRate: '34.2%',
			status: 'COMPLETED',
			date: '2026-10-21'
		},
		{
			id: 'CMP-IG-02',
			name: 'Reel Comment-to-DM: Switzerland Luxury Itinerary',
			channel: 'instagram',
			businessUnit: 'TRAVEL',
			audience: 'Reel Commenters ("SWISS" keyword)',
			sentCount: 312,
			deliveryRate: '100%',
			clickRate: '58.4%',
			status: 'ACTIVE',
			date: 'Live Bot Rule'
		},
		{
			id: 'CMP-LI-03',
			name: 'B2B SOW Architecture Audit Case Study & Lead Gen',
			channel: 'linkedin',
			businessUnit: 'CONSULTING',
			audience: 'CTOs & VPs of Engineering (LinkedIn Sponsored)',
			sentCount: 48,
			deliveryRate: '100%',
			clickRate: '22.9%',
			status: 'ACTIVE',
			date: '2026-10-22'
		},
		{
			id: 'CMP-EM-04',
			name: 'DrAyya Telehealth Weekly Wellness Newsletter',
			channel: 'email',
			businessUnit: 'HEALTHCARE',
			audience: 'Registered Clinic Patients (4,890 Patients)',
			sentCount: 4890,
			deliveryRate: '99.2%',
			clickRate: '41.2%',
			status: 'COMPLETED',
			date: '2026-10-20'
		}
	]);

	// Automated Comment-to-DM Rules
	let botRules = $state([
		{
			id: 'RULE-01',
			channel: 'Instagram',
			keyword: 'PRICE / BOOK',
			action: 'Public reply + Instant DM with 10% holiday voucher',
			active: true,
			triggeredCount: 418
		},
		{
			id: 'RULE-02',
			channel: 'WhatsApp',
			keyword: 'ABANDONED_CART_30M',
			action: 'Auto WhatsApp reminder with 1-click UPI checkout link',
			active: true,
			triggeredCount: 184
		},
		{
			id: 'RULE-03',
			channel: 'LinkedIn',
			keyword: 'LEAD_FORM_SUBMISSION',
			action: 'Ingest into CRM in <2s + Send Consulting SOW deck',
			active: true,
			triggeredCount: 62
		}
	]);

	// Channel Filter
	let channelFilter = $state<'all' | 'whatsapp' | 'instagram' | 'linkedin' | 'email'>('all');
	let filteredCampaigns = $derived(
		channelFilter === 'all' ? campaigns : campaigns.filter(c => c.channel === channelFilter)
	);

	// New Campaign Modal
	let showNewCampaignModal = $state(false);
	let newChannel = $state<'whatsapp' | 'instagram' | 'linkedin' | 'email'>('whatsapp');
	let campaignTitle = $state('Diwali Holiday Flash Sale 2026');
	let targetAudience = $state('Kashmir & Kerala Leads (2,100 contacts)');
	let campaignBody = $state('Celebrate Diwali with family! Get ₹5,000 instant credit voucher on luxury Srinagar and Munnar packages. Valid for next 48 hours.');
	let isDispatching = $state(false);

	function launchCampaign() {
		isDispatching = true;
		setTimeout(() => {
			campaigns.unshift({
				id: 'CMP-' + Math.floor(1000 + Math.random() * 9000),
				name: campaignTitle,
				channel: newChannel,
				businessUnit: 'TRAVEL',
				audience: targetAudience,
				sentCount: 2100,
				deliveryRate: '99.4%',
				clickRate: '38.1%',
				status: 'COMPLETED',
				date: 'Just Now'
			});
			isDispatching = false;
			showNewCampaignModal = false;
			toast?.success?.(`Campaign "${campaignTitle}" successfully dispatched across ${newChannel.toUpperCase()}!`);
		}, 900);
	}

	function toggleBotRule(ruleId: string) {
		const r = botRules.find(x => x.id === ruleId);
		if (r) {
			r.active = !r.active;
			toast?.info?.(`Rule ${r.keyword} is now ${r.active ? 'ACTIVE' : 'PAUSED'}`);
		}
	}
</script>

<svelte:head>
	<title>Omnichannel Marketing & Social Suite | MrBiz OS</title>
	<meta name="description" content="Centralized marketing automation across Email, WhatsApp Business Cloud, Instagram DMs, and LinkedIn B2B lead generation." />
</svelte:head>

<div class="marketing-page">
	<!-- Top Bar -->
	<header class="page-top-bar">
		<div>
			<div class="sub-pill">OMNICHANNEL MARKETING & SOCIAL AUTOMATION</div>
			<h1 class="page-title">Marketing, Broadcasts & Social Suite</h1>
			<p class="page-desc">Automate customer outreach, broadcast campaigns, and lead-generation bots across Email, WhatsApp, Instagram, and LinkedIn.</p>
		</div>

		<div class="top-actions">
			<a href="/inbox" class="btn-inbox-link">
				<span>💬 Open 2-Way Social Inbox</span>
				<span>➔</span>
			</a>
			<button type="button" class="btn-new-campaign" onclick={() => (showNewCampaignModal = true)}>
				<span>+ Launch New Campaign</span>
			</button>
		</div>
	</header>

	<!-- Four Channel Telemetry Grid -->
	<div class="channels-kpi-grid">
		<!-- WHATSAPP -->
		<div class="ch-kpi-card wa">
			<div class="ch-top">
				<div class="ch-icon">🟢</div>
				<span class="ch-status-dot"></span>
			</div>
			<div class="ch-title">WhatsApp Business Cloud</div>
			<div class="ch-metric font-mono">+91 98840 11223</div>
			<div class="ch-stats-row">
				<div><span>Broadcast Deliveries:</span> <strong>98.6%</strong></div>
				<div><span>Active Catalogs:</span> <strong>6 Modules</strong></div>
			</div>
			<div class="ch-badge">Verified Meta Green Tick</div>
		</div>

		<!-- INSTAGRAM -->
		<div class="ch-kpi-card ig">
			<div class="ch-top">
				<div class="ch-icon">🟣</div>
				<span class="ch-status-dot"></span>
			</div>
			<div class="ch-title">Instagram Direct & Reel Bots</div>
			<div class="ch-metric font-mono">@mrbiz_group (28.4k)</div>
			<div class="ch-stats-row">
				<div><span>Comment-to-DM:</span> <strong>Active</strong></div>
				<div><span>Inquiries MTD:</span> <strong>418 DMs</strong></div>
			</div>
			<div class="ch-badge">Comment-to-DM Bot Active</div>
		</div>

		<!-- LINKEDIN -->
		<div class="ch-kpi-card li">
			<div class="ch-top">
				<div class="ch-icon">👔</div>
				<span class="ch-status-dot"></span>
			</div>
			<div class="ch-title">LinkedIn B2B Lead Gen</div>
			<div class="ch-metric font-mono">14.8k Enterprise Followers</div>
			<div class="ch-stats-row">
				<div><span>Lead Forms Sync:</span> <strong>&lt;2s Ingest</strong></div>
				<div><span>B2B Deals MTD:</span> <strong>₹48.5L</strong></div>
			</div>
			<div class="ch-badge">Lead Gen Forms Webhook Live</div>
		</div>

		<!-- EMAIL -->
		<div class="ch-kpi-card em">
			<div class="ch-top">
				<div class="ch-icon">📧</div>
				<span class="ch-status-dot"></span>
			</div>
			<div class="ch-title">Enterprise Email Suite</div>
			<div class="ch-metric font-mono">concierge@mrbiz.group</div>
			<div class="ch-stats-row">
				<div><span>Avg Open Rate:</span> <strong>41.2%</strong></div>
				<div><span>2-Way Threading:</span> <strong>Auto-CRM</strong></div>
			</div>
			<div class="ch-badge">IMAP & SMTP 2-Way Synced</div>
		</div>
	</div>

	<!-- Automated Trigger Rules Engine -->
	<section class="rules-section">
		<div class="sec-header">
			<div>
				<h2 class="sec-title">Active Social Automation & Bot Rules</h2>
				<p class="sec-desc">Autonomous trigger workflows connecting social interactions directly to booking vouchers and CRM leads.</p>
			</div>
			<button class="btn-add-rule" onclick={() => toast?.info?.('New keyword rule builder launched')}>
				+ Create Trigger Rule
			</button>
		</div>

		<div class="rules-grid">
			{#each botRules as r}
				<div class="rule-card {r.active ? '' : 'paused'}">
					<div class="r-top">
						<span class="r-ch-badge {r.channel.toLowerCase()}">{r.channel}</span>
						<label class="toggle-switch">
							<input type="checkbox" checked={r.active} onchange={() => toggleBotRule(r.id)} />
							<span class="slider"></span>
						</label>
					</div>

					<div class="r-keyword">Trigger: <strong>"{r.keyword}"</strong></div>
					<div class="r-action">{r.action}</div>

					<div class="r-footer">
						<span>Total Triggers:</span>
						<strong class="font-mono">{r.triggeredCount} Executed</strong>
					</div>
				</div>
			{/each}
		</div>
	</section>

	<!-- Campaigns History & Register -->
	<section class="campaigns-section">
		<div class="sec-header">
			<div>
				<h2 class="sec-title">Outbound Marketing Campaigns Register</h2>
				<p class="sec-desc">Omnichannel broadcasts dispatched across WhatsApp, Instagram, LinkedIn, and Email.</p>
			</div>

			<!-- Channel Filter Buttons -->
			<div class="filter-pills">
				<button type="button" class="f-pill {channelFilter === 'all' ? 'active' : ''}" onclick={() => (channelFilter = 'all')}>All</button>
				<button type="button" class="f-pill {channelFilter === 'whatsapp' ? 'active' : ''}" onclick={() => (channelFilter = 'whatsapp')}>WhatsApp</button>
				<button type="button" class="f-pill {channelFilter === 'instagram' ? 'active' : ''}" onclick={() => (channelFilter = 'instagram')}>Instagram</button>
				<button type="button" class="f-pill {channelFilter === 'linkedin' ? 'active' : ''}" onclick={() => (channelFilter = 'linkedin')}>LinkedIn</button>
				<button type="button" class="f-pill {channelFilter === 'email' ? 'active' : ''}" onclick={() => (channelFilter = 'email')}>Email</button>
			</div>
		</div>

		<div class="campaigns-table-card">
			<div class="table-wrapper">
				<table class="data-table">
					<thead>
						<tr>
							<th>CAMPAIGN NAME</th>
							<th>CHANNEL</th>
							<th>BUSINESS UNIT</th>
							<th>TARGET AUDIENCE</th>
							<th>DELIVERIES</th>
							<th>CLICK / CTR</th>
							<th>STATUS</th>
							<th>DATE</th>
						</tr>
					</thead>
					<tbody>
						{#each filteredCampaigns as cmp}
							<tr>
								<td class="bold-title">{cmp.name}</td>
								<td>
									<span class="ch-tag {cmp.channel}">{cmp.channel.toUpperCase()}</span>
								</td>
								<td>
									<span class="unit-tag">{cmp.businessUnit}</span>
								</td>
								<td class="audience-cell">{cmp.audience}</td>
								<td class="font-mono">{cmp.sentCount.toLocaleString('en-IN')} ({cmp.deliveryRate})</td>
								<td class="font-mono ctr-val">{cmp.clickRate}</td>
								<td>
									<span class="status-chip {cmp.status.toLowerCase()}">{cmp.status}</span>
								</td>
								<td class="font-mono text-muted">{cmp.date}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</div>
	</section>
</div>

<!-- Launch Campaign Modal -->
{#if showNewCampaignModal}
	<div class="modal-backdrop" onclick={() => (showNewCampaignModal = false)}>
		<div class="campaign-modal-box" onclick={(e) => e.stopPropagation()}>
			<button type="button" class="btn-modal-close" onclick={() => (showNewCampaignModal = false)}>✕</button>

			<div class="modal-header">
				<div class="m-badge">OMNICHANNEL BROADCAST DISPATCHER</div>
				<h3 class="m-title">Launch Marketing Campaign</h3>
				<p class="m-sub">Send broadcast messages, e-newsletters, or B2B updates across your verified communication channels.</p>
			</div>

			<form onsubmit={(e) => { e.preventDefault(); launchCampaign(); }} class="modal-form">
				<!-- Channel Selector -->
				<div class="ch-select-row">
					<button
						type="button"
						class="ch-btn {newChannel === 'whatsapp' ? 'active wa' : ''}"
						onclick={() => (newChannel = 'whatsapp')}
					>
						<span>🟢 WhatsApp</span>
					</button>
					<button
						type="button"
						class="ch-btn {newChannel === 'instagram' ? 'active ig' : ''}"
						onclick={() => (newChannel = 'instagram')}
					>
						<span>🟣 Instagram</span>
					</button>
					<button
						type="button"
						class="ch-btn {newChannel === 'linkedin' ? 'active li' : ''}"
						onclick={() => (newChannel = 'linkedin')}
					>
						<span>👔 LinkedIn</span>
					</button>
					<button
						type="button"
						class="ch-btn {newChannel === 'email' ? 'active em' : ''}"
						onclick={() => (newChannel = 'email')}
					>
						<span>📧 Email</span>
					</button>
				</div>

				<div class="form-group">
					<label for="cmp-name">Campaign Title / Subject Line *</label>
					<input id="cmp-name" type="text" required bind:value={campaignTitle} class="form-input" />
				</div>

				<div class="form-group">
					<label for="cmp-aud">Target Audience Segment *</label>
					<select id="cmp-aud" bind:value={targetAudience} class="form-input">
						<option value="Kashmir & Kerala Leads (2,100 contacts)">Kashmir & Kerala Leads (2,100 contacts)</option>
						<option value="All Verified B2B Travel Agencies (1,240 partners)">All Verified B2B Travel Agencies (1,240 partners)</option>
						<option value="Registered Telehealth Patients (4,890 patients)">Registered Telehealth Patients (4,890 patients)</option>
						<option value="Enterprise CTOs & Consulting Prospects (1,850 leads)">Enterprise CTOs & Consulting Prospects (1,850 leads)</option>
					</select>
				</div>

				<div class="form-group">
					<label for="cmp-body">Message Content / Template Copy</label>
					<textarea id="cmp-body" rows="4" bind:value={campaignBody} class="form-input textarea"></textarea>
				</div>

				<div class="modal-actions">
					<button type="button" class="btn-cancel" onclick={() => (showNewCampaignModal = false)}>Cancel</button>
					<button type="submit" class="btn-dispatch-campaign" disabled={isDispatching}>
						{#if isDispatching}
							<span class="spinner"></span>
							<span>Dispatching Broadcast...</span>
						{:else}
							<span>Send Broadcast Now ➔</span>
						{/if}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}

<style>
	.marketing-page {
		min-height: 100vh;
		background: #090d16;
		color: #f8fafc;
		font-family: var(--font-sans, system-ui, sans-serif);
		padding: 2rem;
		box-sizing: border-box;
	}

	.page-top-bar {
		display: flex;
		justify-content: space-between;
		align-items: center;
		flex-wrap: wrap;
		gap: 1rem;
		padding-bottom: 1.5rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
		margin-bottom: 2rem;
	}

	.sub-pill {
		font-size: 0.72rem;
		font-weight: 800;
		color: #c084fc;
		letter-spacing: 0.08em;
	}

	.page-title {
		font-size: 1.8rem;
		font-weight: 900;
		color: #ffffff;
		margin: 0.2rem 0;
	}

	.page-desc {
		font-size: 0.9rem;
		color: #94a3b8;
		margin: 0;
	}

	.top-actions {
		display: flex;
		gap: 0.75rem;
	}

	.btn-inbox-link {
		display: flex;
		align-items: center;
		gap: 0.45rem;
		padding: 0.7rem 1.25rem;
		border-radius: 10px;
		background: rgba(255, 255, 255, 0.08);
		border: 1px solid rgba(255, 255, 255, 0.15);
		color: #ffffff;
		font-size: 0.85rem;
		font-weight: 700;
		text-decoration: none;
	}

	.btn-new-campaign {
		padding: 0.7rem 1.4rem;
		border-radius: 10px;
		background: linear-gradient(135deg, #a855f7 0%, #7c3aed 100%);
		border: none;
		color: #ffffff;
		font-size: 0.88rem;
		font-weight: 800;
		cursor: pointer;
		box-shadow: 0 4px 14px rgba(168, 85, 247, 0.4);
	}

	/* KPI Grid */
	.channels-kpi-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
		gap: 1.25rem;
		margin-bottom: 2.5rem;
	}

	.ch-kpi-card {
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 16px;
		padding: 1.25rem;
		display: flex;
		flex-direction: column;
		gap: 0.6rem;
	}

	.ch-kpi-card.wa { border-color: rgba(37, 211, 102, 0.3); }
	.ch-kpi-card.ig { border-color: rgba(225, 48, 108, 0.3); }
	.ch-kpi-card.li { border-color: rgba(10, 102, 194, 0.3); }
	.ch-kpi-card.em { border-color: rgba(234, 67, 53, 0.3); }

	.ch-top {
		display: flex;
		justify-content: space-between;
		align-items: center;
	}

	.ch-icon { font-size: 1.5rem; }
	.ch-status-dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		background: #34d399;
		box-shadow: 0 0 8px #34d399;
	}

	.ch-title {
		font-size: 0.95rem;
		font-weight: 800;
		color: #ffffff;
	}

	.ch-metric {
		font-size: 0.82rem;
		color: #38bdf8;
	}

	.ch-stats-row {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
		font-size: 0.74rem;
		color: #94a3b8;
		padding: 0.5rem 0;
		border-top: 1px solid rgba(255, 255, 255, 0.06);
		border-bottom: 1px solid rgba(255, 255, 255, 0.06);
	}

	.ch-stats-row strong { color: #f1f5f9; }

	.ch-badge {
		font-size: 0.68rem;
		font-weight: 700;
		color: #cbd5e1;
	}

	/* Rules section */
	.rules-section {
		margin-bottom: 2.5rem;
	}

	.sec-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 1.25rem;
		flex-wrap: wrap;
		gap: 0.75rem;
	}

	.sec-title {
		font-size: 1.25rem;
		font-weight: 800;
		color: #ffffff;
		margin: 0 0 0.25rem;
	}

	.sec-desc {
		font-size: 0.84rem;
		color: #94a3b8;
		margin: 0;
	}

	.btn-add-rule {
		padding: 0.55rem 1rem;
		border-radius: 8px;
		background: rgba(255, 255, 255, 0.06);
		border: 1px solid rgba(255, 255, 255, 0.12);
		color: #ffffff;
		font-size: 0.8rem;
		font-weight: 700;
		cursor: pointer;
	}

	.rules-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
		gap: 1rem;
	}

	.rule-card {
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 12px;
		padding: 1.1rem;
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.rule-card.paused { opacity: 0.5; }

	.r-top {
		display: flex;
		justify-content: space-between;
		align-items: center;
	}

	.r-ch-badge {
		font-size: 0.68rem;
		font-weight: 800;
		padding: 0.2rem 0.5rem;
		border-radius: 4px;
	}

	.r-ch-badge.instagram { background: rgba(225, 48, 108, 0.2); color: #f472b6; }
	.r-ch-badge.whatsapp { background: rgba(37, 211, 102, 0.2); color: #4ade80; }
	.r-ch-badge.linkedin { background: rgba(10, 102, 194, 0.2); color: #38bdf8; }

	.r-keyword { font-size: 0.88rem; color: #ffffff; }
	.r-action { font-size: 0.78rem; color: #94a3b8; line-height: 1.4; flex: 1; }
	.r-footer {
		display: flex;
		justify-content: space-between;
		font-size: 0.75rem;
		color: #64748b;
		padding-top: 0.5rem;
		border-top: 1px solid rgba(255, 255, 255, 0.06);
	}

	.r-footer strong { color: #34d399; }

	/* Toggle switch */
	.toggle-switch {
		position: relative;
		display: inline-block;
		width: 32px;
		height: 18px;
	}

	.toggle-switch input { opacity: 0; width: 0; height: 0; }
	.slider {
		position: absolute; cursor: pointer; inset: 0;
		background-color: #334155; border-radius: 20px; transition: 0.2s;
	}
	.slider:before {
		position: absolute; content: ""; height: 12px; width: 12px; left: 3px; bottom: 3px;
		background-color: white; border-radius: 50%; transition: 0.2s;
	}
	input:checked + .slider { background-color: #10b981; }
	input:checked + .slider:before { transform: translateX(14px); }

	/* Filter Pills */
	.filter-pills {
		display: flex;
		gap: 0.35rem;
	}

	.f-pill {
		padding: 0.4rem 0.75rem;
		border-radius: 6px;
		background: rgba(255, 255, 255, 0.05);
		border: 1px solid rgba(255, 255, 255, 0.08);
		color: #94a3b8;
		font-size: 0.78rem;
		font-weight: 700;
		cursor: pointer;
	}

	.f-pill.active {
		background: rgba(168, 85, 247, 0.2);
		border-color: #c084fc;
		color: #ffffff;
	}

	/* Campaigns Table */
	.campaigns-table-card {
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 14px;
		overflow: hidden;
	}

	.table-wrapper { overflow-x: auto; }
	.data-table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.82rem;
		text-align: left;
	}

	.data-table th {
		padding: 0.75rem 1rem;
		background: rgba(255, 255, 255, 0.02);
		color: #64748b;
		font-size: 0.7rem;
		font-weight: 800;
		border-bottom: 1px solid rgba(255, 255, 255, 0.08);
	}

	.data-table td {
		padding: 0.85rem 1rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.04);
		color: #cbd5e1;
	}

	.bold-title { font-weight: 700; color: #ffffff; }

	.ch-tag {
		font-size: 0.68rem;
		font-weight: 800;
		padding: 0.2rem 0.5rem;
		border-radius: 4px;
	}

	.ch-tag.whatsapp { background: rgba(37, 211, 102, 0.2); color: #4ade80; }
	.ch-tag.instagram { background: rgba(225, 48, 108, 0.2); color: #f472b6; }
	.ch-tag.linkedin { background: rgba(10, 102, 194, 0.2); color: #38bdf8; }
	.ch-tag.email { background: rgba(234, 67, 53, 0.2); color: #f87171; }

	.unit-tag {
		font-size: 0.7rem;
		background: rgba(255, 255, 255, 0.06);
		padding: 0.2rem 0.45rem;
		border-radius: 4px;
	}

	.audience-cell { font-size: 0.78rem; color: #94a3b8; }
	.ctr-val { color: #34d399; font-weight: 700; }

	.status-chip {
		font-size: 0.7rem;
		font-weight: 800;
		padding: 0.15rem 0.5rem;
		border-radius: 999px;
	}

	.status-chip.completed { background: rgba(16, 185, 129, 0.15); color: #34d399; }
	.status-chip.active { background: rgba(56, 189, 248, 0.15); color: #38bdf8; }

	.text-muted { color: #64748b; font-size: 0.75rem; }

	/* Modal */
	.modal-backdrop {
		position: fixed;
		inset: 0;
		z-index: 3000;
		background: rgba(0, 0, 0, 0.8);
		backdrop-filter: blur(8px);
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 1.5rem;
	}

	.campaign-modal-box {
		position: relative;
		width: 100%;
		max-width: 620px;
		background: #0f172a;
		border: 1px solid rgba(255, 255, 255, 0.15);
		border-radius: 18px;
		padding: 2rem;
		box-shadow: 0 25px 60px rgba(0, 0, 0, 0.8);
	}

	.btn-modal-close {
		position: absolute;
		top: 1.25rem;
		right: 1.25rem;
		background: rgba(255, 255, 255, 0.08);
		border: none;
		color: #ffffff;
		width: 32px;
		height: 32px;
		border-radius: 50%;
		cursor: pointer;
	}

	.m-badge { font-size: 0.7rem; font-weight: 800; color: #c084fc; margin-bottom: 0.25rem; }
	.m-title { font-size: 1.4rem; font-weight: 900; color: #ffffff; margin: 0 0 0.35rem; }
	.m-sub { font-size: 0.85rem; color: #94a3b8; margin: 0 0 1.5rem; }

	.ch-select-row {
		display: grid;
		grid-template-columns: repeat(4, 1fr);
		gap: 0.5rem;
		margin-bottom: 1.25rem;
	}

	.ch-btn {
		padding: 0.65rem 0.5rem;
		border-radius: 8px;
		background: #090d16;
		border: 2px solid rgba(255, 255, 255, 0.08);
		color: #cbd5e1;
		font-size: 0.8rem;
		font-weight: 700;
		cursor: pointer;
	}

	.ch-btn.active.wa { border-color: #25d366; background: rgba(37, 211, 102, 0.15); color: #ffffff; }
	.ch-btn.active.ig { border-color: #e1306c; background: rgba(225, 48, 108, 0.15); color: #ffffff; }
	.ch-btn.active.li { border-color: #0a66c2; background: rgba(10, 102, 194, 0.15); color: #ffffff; }
	.ch-btn.active.em { border-color: #ea4335; background: rgba(234, 67, 53, 0.15); color: #ffffff; }

	.form-group {
		display: flex;
		flex-direction: column;
		gap: 0.35rem;
		margin-bottom: 1rem;
	}

	.form-group label {
		font-size: 0.78rem;
		font-weight: 700;
		color: #cbd5e1;
	}

	.form-input {
		width: 100%;
		box-sizing: border-box;
		padding: 0.7rem 0.9rem;
		background: #090d16;
		border: 1px solid rgba(255, 255, 255, 0.12);
		border-radius: 8px;
		color: #ffffff;
		font-size: 0.88rem;
	}

	.form-input.textarea { resize: vertical; }

	.modal-actions {
		display: flex;
		justify-content: flex-end;
		gap: 0.75rem;
		margin-top: 1.5rem;
	}

	.btn-cancel {
		padding: 0.7rem 1.25rem;
		border-radius: 8px;
		background: rgba(255, 255, 255, 0.08);
		border: none;
		color: #ffffff;
		font-size: 0.85rem;
		cursor: pointer;
	}

	.btn-dispatch-campaign {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.7rem 1.4rem;
		border-radius: 8px;
		background: linear-gradient(135deg, #a855f7 0%, #7c3aed 100%);
		border: none;
		color: #ffffff;
		font-size: 0.88rem;
		font-weight: 800;
		cursor: pointer;
	}

	.spinner {
		width: 14px;
		height: 14px;
		border: 2px solid rgba(255, 255, 255, 0.3);
		border-top-color: #ffffff;
		border-radius: 50%;
		animation: spin 0.7s linear infinite;
	}

	@keyframes spin { to { transform: rotate(360deg); } }

	@media (max-width: 768px) {
		.ch-select-row {
			grid-template-columns: 1fr 1fr;
		}
	}
</style>
