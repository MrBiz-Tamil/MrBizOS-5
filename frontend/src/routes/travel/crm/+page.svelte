<script lang="ts">
	import { onMount } from 'svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import CreateBookingModal from '$lib/components/travel/CreateBookingModal.svelte';

	// Types
	interface TravelLead {
		id: string;
		leadNumber: string;
		name: string;
		groupTitle: string;
		phone: string;
		whatsapp: string;
		email: string;
		city: string;
		state: string;

		// Tour Specifications
		destination: string;
		tourType: 'Family' | 'Honeymoon' | 'Corporate' | 'Group' | 'Solo' | 'Pilgrimage';
		travelDate: string;
		durationNights: number;
		paxAdults: number;
		paxChildren: number;
		hotelPreference: '3 Star Standard' | '4 Star Luxury' | '5 Star Deluxe' | 'Houseboat / Heritage';
		cabRequirement: 'Sedan' | 'Innova Crysta AC' | 'Tempo Traveller' | 'Luxury Coach' | 'None';
		mealPlan: 'CP (Breakfast)' | 'MAP (Breakfast + Dinner)' | 'AP (All Meals)';

		// Commercials & Pipeline
		stage: 'new' | 'qualified' | 'proposition' | 'negotiation' | 'won' | 'lost';
		expectedRevenue: number;
		probability: number;
		priority: 1 | 2 | 3;
		assignedAgent: {
			name: string;
			avatar: string;
			initials: string;
		};
		tags: string[];

		// UTM Marketing Attribution
		utm: {
			source: string; // 'facebook' | 'instagram' | 'google' | 'whatsapp' | 'agent_referral' | 'organic'
			medium: string; // 'cpc' | 'reels_boost' | 'story_ad' | 'broadcast' | 'organic_search'
			campaign: string; // 'diwali_kashmir_2026' | 'dubai_winter_2026' | 'malaysia_business_q4' | 'kerala_monsoon'
			term?: string;
			content?: string;
			landingPageUrl: string;
			referrerDomain?: string;
			firstTouchDate: string;
		};

		// Activity & Chatter
		nextActivity?: {
			type: 'Call' | 'WhatsApp' | 'Send Itinerary' | 'Meeting' | 'Visa Follow-up';
			date: string;
			summary: string;
			status: 'planned' | 'today' | 'overdue';
		};
		lostReason?: string;
		notes: string;
		createdAt: string;
		activities: {
			id: string;
			user: string;
			type: string;
			text: string;
			timestamp: string;
		}[];
	}

	// Active View Mode: 'kanban' | 'list' | 'utm-pivot'
	let activeView = $state<'kanban' | 'list' | 'utm-pivot'>('kanban');

	// Search & Filter State
	let searchQuery = $state('');
	let filterStage = $state('all');
	let filterSource = $state('all');
	let filterCampaign = $state('all');
	let filterCity = $state('all');

	// Active Lead Drawer State
	let selectedLead = $state<TravelLead | null>(null);
	let showLeadDrawer = $state(false);
	let showBookingModal = $state(false);
	let drawerActiveTab = $state<'specs' | 'contact' | 'utm' | 'chatter'>('specs');

	// Quick Create Modal State
	let showNewLeadModal = $state(false);
	let newLeadStage = $state<'new' | 'qualified' | 'proposition' | 'negotiation'>('new');

	// New Activity Form State
	let newActivityType = $state<'Call' | 'WhatsApp' | 'Send Itinerary' | 'Meeting'>('WhatsApp');
	let newActivitySummary = $state('');
	let newActivityDate = $state(new Date().toISOString().split('T')[0]);
	let newChatterNote = $state('');

	// Stages Definition
	const stages = [
		{ id: 'new', label: 'New / Inbound', color: '#3b82f6', badgeClass: 'stage-blue', icon: '📥' },
		{ id: 'qualified', label: 'Qualified', color: '#06b6d4', badgeClass: 'stage-cyan', icon: '🎯' },
		{ id: 'proposition', label: 'Itinerary Sent', color: '#8b5cf6', badgeClass: 'stage-purple', icon: '🗺️' },
		{ id: 'negotiation', label: 'Negotiation', color: '#f59e0b', badgeClass: 'stage-amber', icon: '💬' },
		{ id: 'won', label: 'Won / Booked', color: '#10b981', badgeClass: 'stage-green', icon: '🏆' },
		{ id: 'lost', label: 'Lost', color: '#ef4444', badgeClass: 'stage-red', icon: '❌' }
	];

	// Initial Realistic Travel Leads Dataset with South India Origins & UTM Attribution
	let leads = $state<TravelLead[]>([
		{
			id: 'lead-1',
			leadNumber: 'LEAD-2026-101',
			name: 'Rahul Sharma & Family',
			groupTitle: 'Sharma Family Holiday (4 Pax)',
			phone: '+91 98840 11223',
			whatsapp: '+91 98840 11223',
			email: 'rahul.sharma@example.com',
			city: 'Chennai',
			state: 'Tamil Nadu',
			destination: 'Kashmir Valley Paradise',
			tourType: 'Family',
			travelDate: '2026-11-12',
			durationNights: 5,
			paxAdults: 2,
			paxChildren: 2,
			hotelPreference: '4 Star Luxury',
			cabRequirement: 'Innova Crysta AC',
			mealPlan: 'MAP (Breakfast + Dinner)',
			stage: 'new',
			expectedRevenue: 165000,
			probability: 80,
			priority: 3,
			assignedAgent: { name: 'Priya Menon', avatar: '👩‍💼', initials: 'PM' },
			tags: ['Family Tour', 'Flight Req', 'Snow / Gulmarg'],
			utm: {
				source: 'facebook',
				medium: 'cpc',
				campaign: 'diwali_kashmir_paradise_2026',
				term: 'kashmir tour package from chennai',
				content: 'video_snow_gulmarg_v2',
				landingPageUrl: 'https://mrbiz.tours/kashmir-valley?utm_source=facebook&utm_medium=cpc&utm_campaign=diwali_kashmir_paradise_2026',
				referrerDomain: 'l.facebook.com',
				firstTouchDate: '2026-10-22 14:15'
			},
			nextActivity: {
				type: 'Call',
				date: '2026-10-23',
				summary: 'Confirm flight timings from MAA to SXR with Indigo',
				status: 'today'
			},
			notes: 'Wants hotel in Pahalgam riverfront and Gulmarg Gandola Phase 1 & 2 tickets pre-booked.',
			createdAt: '2026-10-22T08:30:00Z',
			activities: [
				{ id: 'act-1', user: 'System Bot', type: 'inbound', text: 'Lead captured from Facebook Lead Ad Form (Diwali Kashmir Blast)', timestamp: '2026-10-22 14:15' },
				{ id: 'act-2', user: 'Priya Menon', type: 'call', text: 'Called customer; interested in 4-star hotels with heater and snow sledging for kids.', timestamp: '2026-10-22 15:40' }
			]
		},
		{
			id: 'lead-2',
			leadNumber: 'LEAD-2026-102',
			name: 'Karthik Subramanian',
			groupTitle: 'Corporate Tech Delegation (6 Pax)',
			phone: '+91 94441 55667',
			whatsapp: '+91 94441 55667',
			email: 'karthik.sub@techmadurai.in',
			city: 'Madurai',
			state: 'Tamil Nadu',
			destination: 'Malaysia Business & Twin Towers',
			tourType: 'Corporate',
			travelDate: '2026-11-20',
			durationNights: 4,
			paxAdults: 6,
			paxChildren: 0,
			hotelPreference: '5 Star Deluxe',
			cabRequirement: 'Luxury Coach',
			mealPlan: 'CP (Breakfast)',
			stage: 'qualified',
			expectedRevenue: 320000,
			probability: 70,
			priority: 3,
			assignedAgent: { name: 'Arun Kumar', avatar: '👨‍💼', initials: 'AK' },
			tags: ['Corporate', 'GST Invoice', 'Conference Hall'],
			utm: {
				source: 'google',
				medium: 'cpc',
				campaign: 'malaysia_corporate_q4',
				term: 'malaysia business corporate tour madurai',
				content: 'search_ad_corporate_gst',
				landingPageUrl: 'https://mrbiz.tours/malaysia-corporate?utm_source=google&utm_medium=cpc&utm_campaign=malaysia_corporate_q4',
				referrerDomain: 'google.com',
				firstTouchDate: '2026-10-21 11:20'
			},
			nextActivity: {
				type: 'Send Itinerary',
				date: '2026-10-23',
				summary: 'Submit formal quotation with GST and e-Visa checklist',
				status: 'today'
			},
			notes: 'Requires convention hall in KLCC and private transfer from KLIA Terminal 1.',
			createdAt: '2026-10-21T11:20:00Z',
			activities: [
				{ id: 'act-3', user: 'System Bot', type: 'inbound', text: 'Form submission on Google Search Ad Landing Page', timestamp: '2026-10-21 11:20' },
				{ id: 'act-4', user: 'Arun Kumar', type: 'whatsapp', text: 'Sent company profile and hotel options in KL Sentral.', timestamp: '2026-10-21 16:30' }
			]
		},
		{
			id: 'lead-3',
			leadNumber: 'LEAD-2026-103',
			name: 'Ananya & Vignesh',
			groupTitle: 'Honeymoon Package (2 Pax)',
			phone: '+91 97890 33445',
			whatsapp: '+91 97890 33445',
			email: 'vignesh.a@gmail.com',
			city: 'Kanniyakumari',
			state: 'Tamil Nadu',
			destination: 'Kerala Backwaters & Munnar',
			tourType: 'Honeymoon',
			travelDate: '2026-11-28',
			durationNights: 4,
			paxAdults: 2,
			paxChildren: 0,
			hotelPreference: 'Houseboat / Heritage',
			cabRequirement: 'Sedan',
			mealPlan: 'AP (All Meals)',
			stage: 'proposition',
			expectedRevenue: 98000,
			probability: 90,
			priority: 2,
			assignedAgent: { name: 'Deepa Lakshmi', avatar: '👩‍💼', initials: 'DL' },
			tags: ['Honeymoon', 'Private Houseboat', 'Flower Bed Decor'],
			utm: {
				source: 'instagram',
				medium: 'reels_boost',
				campaign: 'kerala_monsoon_retreat',
				term: 'kerala honeymoon package private boat',
				content: 'alleppey_drone_reel_cta',
				landingPageUrl: 'https://mrbiz.tours/kerala-honeymoon?utm_source=instagram&utm_medium=reels_boost&utm_campaign=kerala_monsoon_retreat',
				referrerDomain: 'instagram.com',
				firstTouchDate: '2026-10-20 18:45'
			},
			nextActivity: {
				type: 'WhatsApp',
				date: '2026-10-24',
				summary: 'Follow up on Alleppey AC Deluxe Houseboat choice',
				status: 'planned'
			},
			notes: 'Wants candle-light dinner on deck and flower decoration in bedroom.',
			createdAt: '2026-10-20T18:45:00Z',
			activities: [
				{ id: 'act-5', user: 'Deepa Lakshmi', type: 'quotation', text: 'Drafted 4N/5D Munnar + Thekkady + Alleppey itinerary with private AC Sedan.', timestamp: '2026-10-21 10:15' }
			]
		},
		{
			id: 'lead-4',
			leadNumber: 'LEAD-2026-104',
			name: 'Dr. S. Rajendran',
			groupTitle: 'Senior Doctors Leisure Tour (8 Pax)',
			phone: '+91 94220 88990',
			whatsapp: '+91 94220 88990',
			email: 'dr.rajendran@apollo.org',
			city: 'Coimbatore',
			state: 'Tamil Nadu',
			destination: 'Dubai Special Desert & City',
			tourType: 'Group',
			travelDate: '2026-12-05',
			durationNights: 5,
			paxAdults: 8,
			paxChildren: 0,
			hotelPreference: '5 Star Deluxe',
			cabRequirement: 'Tempo Traveller',
			mealPlan: 'MAP (Breakfast + Dinner)',
			stage: 'negotiation',
			expectedRevenue: 680000,
			probability: 85,
			priority: 3,
			assignedAgent: { name: 'Priya Menon', avatar: '👩‍💼', initials: 'PM' },
			tags: ['VIP Group', 'Visa Processing', 'Burj Khalifa Top'],
			utm: {
				source: 'whatsapp',
				medium: 'broadcast',
				campaign: 'dubai_winter_2026',
				term: 'dubai direct flights from coimbatore',
				content: 'vip_doctors_club_whatsapp_blast',
				landingPageUrl: 'https://mrbiz.tours/dubai?utm_source=whatsapp&utm_medium=broadcast&utm_campaign=dubai_winter_2026',
				referrerDomain: 'whatsapp.com',
				firstTouchDate: '2026-10-18 09:10'
			},
			nextActivity: {
				type: 'Visa Follow-up',
				date: '2026-10-22',
				summary: 'Collect passport first & last page copies for 8 pax UAE visa',
				status: 'today'
			},
			notes: 'Needs wheelchair assistance at Dubai International Airport (DXB) for 2 elderly guests.',
			createdAt: '2026-10-18T09:10:00Z',
			activities: [
				{ id: 'act-6', user: 'Priya Menon', type: 'call', text: 'Discussed Desert Safari 4x4 private hummer and Marina Dhow cruise with doctor.', timestamp: '2026-10-19 14:00' }
			]
		},
		{
			id: 'lead-5',
			leadNumber: 'LEAD-2026-105',
			name: 'Mohamed Farook',
			groupTitle: 'Farook Family Umrah & Dubai (6 Pax)',
			phone: '+91 98940 77112',
			whatsapp: '+91 98940 77112',
			email: 'farook.trichy@gmail.com',
			city: 'Tiruchirappalli',
			state: 'Tamil Nadu',
			destination: 'Dubai & Abu Dhabi Combo',
			tourType: 'Family',
			travelDate: '2026-11-25',
			durationNights: 6,
			paxAdults: 4,
			paxChildren: 2,
			hotelPreference: '4 Star Luxury',
			cabRequirement: 'Innova Crysta AC',
			mealPlan: 'MAP (Breakfast + Dinner)',
			stage: 'won',
			expectedRevenue: 485000,
			probability: 100,
			priority: 3,
			assignedAgent: { name: 'Arun Kumar', avatar: '👨‍💼', initials: 'AK' },
			tags: ['Won Deal', 'Advance Received', 'Flights Issued'],
			utm: {
				source: 'facebook',
				medium: 'cpc',
				campaign: 'dubai_winter_2026',
				term: 'family tour packages to dubai from trichy',
				content: 'carousel_abu_dhabi_mosque',
				landingPageUrl: 'https://mrbiz.tours/dubai?utm_source=facebook&utm_medium=cpc&utm_campaign=dubai_winter_2026',
				referrerDomain: 'l.facebook.com',
				firstTouchDate: '2026-10-15 16:30'
			},
			notes: 'Received advance ₹1,50,000 via IMPS / UTR. PNR issued on Air India Express.',
			createdAt: '2026-10-15T16:30:00Z',
			activities: [
				{ id: 'act-7', user: 'Arun Kumar', type: 'payment', text: 'Advance token of ₹1,50,000 recorded in Tour Operator Ledger.', timestamp: '2026-10-21 11:00' }
			]
		},
		{
			id: 'lead-6',
			leadNumber: 'LEAD-2026-106',
			name: 'Vijay Anand',
			groupTitle: 'Bangalore Solo Explorer',
			phone: '+91 99001 22334',
			whatsapp: '+91 99001 22334',
			email: 'vijay.trek@gmail.com',
			city: 'Bangalore',
			state: 'Karnataka',
			destination: 'Himachal & Spiti Circuit',
			tourType: 'Solo',
			travelDate: '2026-10-30',
			durationNights: 7,
			paxAdults: 1,
			paxChildren: 0,
			hotelPreference: '3 Star Standard',
			cabRequirement: 'None',
			mealPlan: 'CP (Breakfast)',
			stage: 'lost',
			expectedRevenue: 45000,
			probability: 0,
			priority: 1,
			assignedAgent: { name: 'Deepa Lakshmi', avatar: '👩‍💼', initials: 'DL' },
			tags: ['Lost', 'High Price Objection'],
			lostReason: 'Budget constraints; chose local hostel backpacking instead.',
			utm: {
				source: 'google',
				medium: 'organic_search',
				campaign: 'organic_seo_himachal',
				term: 'spiti valley solo group departure',
				landingPageUrl: 'https://mrbiz.tours/himachal-spiti',
				firstTouchDate: '2026-10-14 12:00'
			},
			notes: 'Follow up next summer for Ladakh bike expedition.',
			createdAt: '2026-10-14T12:00:00Z',
			activities: [
				{ id: 'act-8', user: 'Deepa Lakshmi', type: 'lost', text: 'Marked as Lost: Customer decided to do solo bike ride.', timestamp: '2026-10-19 16:00' }
			]
		}
	]);

	// Filtered Leads
	let filteredLeads = $derived.by(() => {
		return leads.filter((l) => {
			// Search Query
			if (searchQuery.trim()) {
				const q = searchQuery.toLowerCase();
				const matchName = l.name.toLowerCase().includes(q);
				const matchPhone = l.phone.includes(q);
				const matchDest = l.destination.toLowerCase().includes(q);
				const matchCity = l.city.toLowerCase().includes(q);
				const matchCamp = l.utm.campaign.toLowerCase().includes(q);
				const matchSource = l.utm.source.toLowerCase().includes(q);
				if (!matchName && !matchPhone && !matchDest && !matchCity && !matchCamp && !matchSource) return false;
			}
			// Stage Filter
			if (filterStage !== 'all' && l.stage !== filterStage) return false;
			// Source Filter
			if (filterSource !== 'all' && l.utm.source !== filterSource) return false;
			// Campaign Filter
			if (filterCampaign !== 'all' && l.utm.campaign !== filterCampaign) return false;
			// City Filter
			if (filterCity !== 'all' && l.city !== filterCity) return false;

			return true;
		});
	});

	// Pipeline Summary Stats
	let stats = $derived.by(() => {
		const total = leads.length;
		const totalRev = leads.reduce((sum, l) => sum + (l.stage !== 'lost' ? l.expectedRevenue : 0), 0);
		const wonRev = leads.filter(l => l.stage === 'won').reduce((sum, l) => sum + l.expectedRevenue, 0);
		const wonCount = leads.filter(l => l.stage === 'won').length;
		const winRate = total > 0 ? Math.round((wonCount / total) * 100) : 0;
		return { total, totalRev, wonRev, wonCount, winRate };
	});

	// UTM Attribution Matrix Data
	let utmAttribution = $derived.by(() => {
		const sources: Record<string, {
			leadsCount: number;
			qualified: number;
			wonCount: number;
			totalRevenue: number;
			wonRevenue: number;
			campaigns: Set<string>;
		}> = {};

		leads.forEach(l => {
			const s = l.utm.source || 'direct';
			if (!sources[s]) {
				sources[s] = { leadsCount: 0, qualified: 0, wonCount: 0, totalRevenue: 0, wonRevenue: 0, campaigns: new Set() };
			}
			sources[s].leadsCount += 1;
			if (['qualified', 'proposition', 'negotiation', 'won'].includes(l.stage)) {
				sources[s].qualified += 1;
			}
			if (l.stage === 'won') {
				sources[s].wonCount += 1;
				sources[s].wonRevenue += l.expectedRevenue;
			}
			if (l.stage !== 'lost') {
				sources[s].totalRevenue += l.expectedRevenue;
			}
			if (l.utm.campaign) {
				sources[s].campaigns.add(l.utm.campaign);
			}
		});

		return Object.entries(sources).map(([source, data]) => ({
			source,
			leadsCount: data.leadsCount,
			qualified: data.qualified,
			wonCount: data.wonCount,
			totalRevenue: data.totalRevenue,
			wonRevenue: data.wonRevenue,
			campaignCount: data.campaigns.size,
			conversionRate: data.leadsCount > 0 ? Math.round((data.wonCount / data.leadsCount) * 100) : 0
		}));
	});

	// Handlers
	function openLead(l: TravelLead) {
		selectedLead = JSON.parse(JSON.stringify(l)); // clone
		drawerActiveTab = 'specs';
		showLeadDrawer = true;
	}

	function updateStage(leadId: string, newStage: TravelLead['stage']) {
		const idx = leads.findIndex(l => l.id === leadId);
		if (idx !== -1) {
			leads[idx].stage = newStage;
			if (newStage === 'won') leads[idx].probability = 100;
			if (newStage === 'lost') leads[idx].probability = 0;
			leads[idx].activities.unshift({
				id: 'act-' + Date.now(),
				user: 'Current Agent',
				type: 'stage_change',
				text: `Changed stage to ${newStage.toUpperCase()}`,
				timestamp: new Date().toLocaleString('en-IN')
			});
			if (selectedLead && selectedLead.id === leadId) {
				selectedLead.stage = newStage;
			}
			toast.success(`Lead moved to ${newStage.toUpperCase()}`);
		}
	}

	function saveLeadChanges() {
		if (!selectedLead) return;
		const idx = leads.findIndex(l => l.id === selectedLead!.id);
		if (idx !== -1) {
			leads[idx] = JSON.parse(JSON.stringify(selectedLead));
			toast.success(`Lead ${selectedLead.leadNumber} updated successfully!`);
		}
		showLeadDrawer = false;
	}

	function addChatterNote() {
		if (!newChatterNote.trim() || !selectedLead) return;
		selectedLead.activities.unshift({
			id: 'act-' + Date.now(),
			user: 'Current Agent',
			type: 'note',
			text: newChatterNote.trim(),
			timestamp: new Date().toLocaleString('en-IN')
		});
		newChatterNote = '';
		toast.success('Internal note logged in Odoo Chatter');
	}

	function scheduleNewActivity() {
		if (!newActivitySummary.trim() || !selectedLead) return;
		selectedLead.nextActivity = {
			type: newActivityType,
			summary: newActivitySummary.trim(),
			date: newActivityDate,
			status: 'planned'
		};
		selectedLead.activities.unshift({
			id: 'act-' + Date.now(),
			user: 'Current Agent',
			type: 'activity',
			text: `Scheduled ${newActivityType}: "${newActivitySummary.trim()}" for ${newActivityDate}`,
			timestamp: new Date().toLocaleString('en-IN')
		});
		newActivitySummary = '';
		toast.success('Activity scheduled on Odoo timeline');
	}

	function setPriority(leadId: string, stars: 1 | 2 | 3, e: MouseEvent) {
		e.stopPropagation();
		const idx = leads.findIndex(l => l.id === leadId);
		if (idx !== -1) {
			leads[idx].priority = stars;
		}
	}

	// Formatters
	function formatCurrency(val: number) {
		return new Intl.NumberFormat('en-IN', { style: 'currency', currency: 'INR', maximumFractionDigits: 0 }).format(val);
	}

	function copyToClipboard(text: string) {
		if (typeof navigator !== 'undefined' && navigator.clipboard) {
			navigator.clipboard.writeText(text);
			toast.success('UTM link copied to clipboard!');
		}
	}
</script>

<svelte:head>
	<title>Odoo CRM Pipeline &amp; UTM Tracking | Travel &amp; Tours OS</title>
</svelte:head>

<div class="odoo-crm-canvas">
	<!-- =========================================================================
	     1. ODOO CONTROL PANEL TOP MASTHEAD
	     ========================================================================= -->
	<header class="odoo-control-panel">
		<!-- Top Breadcrumb & Actions Row -->
		<div class="cp-top-bar">
			<div class="cp-breadcrumbs">
				<a href="/travel" class="crumb-link">✈️ Travel &amp; Tours OS</a>
				<span class="crumb-separator">/</span>
				<span class="crumb-current">CRM &amp; Leads Pipeline</span>
				<span class="crm-odoo-badge">ODOO EDITION</span>
			</div>

			<!-- Pipeline KPI Highlights -->
			<div class="cp-stat-pills">
				<div class="stat-pill">
					<span class="pill-label">Pipeline Value</span>
					<span class="pill-val font-mono">{formatCurrency(stats.totalRev)}</span>
				</div>
				<div class="stat-pill highlight-green">
					<span class="pill-label">Won Revenue</span>
					<span class="pill-val font-mono">{formatCurrency(stats.wonRev)}</span>
				</div>
				<div class="stat-pill">
					<span class="pill-label">Total Leads</span>
					<span class="pill-val font-mono">{stats.total}</span>
				</div>
				<div class="stat-pill">
					<span class="pill-label">Win Rate</span>
					<span class="pill-val font-mono text-emerald">{stats.winRate}%</span>
				</div>
			</div>

			<!-- View Mode Switcher -->
			<div class="view-switcher-group">
				<button 
					type="button" 
					class="view-btn {activeView === 'kanban' ? 'active' : ''}" 
					onclick={() => (activeView = 'kanban')}
					title="Odoo Kanban Board"
				>
					<span>⊞</span> Kanban
				</button>
				<button 
					type="button" 
					class="view-btn {activeView === 'list' ? 'active' : ''}" 
					onclick={() => (activeView = 'list')}
					title="Table List View"
				>
					<span>☰</span> List
				</button>
				<button 
					type="button" 
					class="view-btn {activeView === 'utm-pivot' ? 'active' : ''}" 
					onclick={() => (activeView = 'utm-pivot')}
					title="UTM Marketing Attribution Matrix"
				>
					<span>📊</span> UTM Attribution
				</button>
			</div>
		</div>

		<!-- Bottom Filter & Search Command Bar -->
		<div class="cp-search-bar">
			<div class="search-input-wrap">
				<span class="search-icon">🔍</span>
				<input 
					type="text" 
					bind:value={searchQuery} 
					placeholder="Search by customer, phone (+91), destination, city (Chennai, Madurai), or UTM campaign..."
					class="odoo-search-input"
				/>
				{#if searchQuery}
					<button type="button" class="clear-search-btn" onclick={() => (searchQuery = '')}>✕</button>
				{/if}
			</div>

			<div class="filters-cluster">
				<!-- Source Filter -->
				<select bind:value={filterSource} class="odoo-filter-select">
					<option value="all">🌐 All UTM Sources</option>
					<option value="facebook">Meta / Facebook</option>
					<option value="instagram">Instagram Ads / Reels</option>
					<option value="google">Google CPC / Search</option>
					<option value="whatsapp">WhatsApp Direct Ads</option>
					<option value="organic">Organic / SEO</option>
				</select>

				<!-- Campaign Filter -->
				<select bind:value={filterCampaign} class="odoo-filter-select">
					<option value="all">🎯 All Campaigns</option>
					<option value="diwali_kashmir_paradise_2026">Diwali Kashmir 2026</option>
					<option value="dubai_winter_2026">Dubai Winter 2026</option>
					<option value="malaysia_corporate_q4">Malaysia Corporate Q4</option>
					<option value="kerala_monsoon_retreat">Kerala Monsoon Retreat</option>
				</select>

				<!-- City Filter -->
				<select bind:value={filterCity} class="odoo-filter-select">
					<option value="all">📍 All Cities</option>
					<option value="Chennai">Chennai</option>
					<option value="Madurai">Madurai</option>
					<option value="Kanniyakumari">Kanniyakumari</option>
					<option value="Coimbatore">Coimbatore</option>
					<option value="Tiruchirappalli">Tiruchirappalli</option>
					<option value="Bangalore">Bangalore</option>
				</select>

				<!-- Action Buttons -->
				<a href="/travel/book" class="btn btn-emerald btn-sm" style="font-weight: 700;">
					⚡ 1-Page Book Trip
				</a>
				<button 
					type="button" 
					class="btn btn-primary btn-sm new-lead-btn" 
					onclick={() => {
						newLeadStage = 'new';
						showNewLeadModal = true;
					}}
				>
					+ New Lead
				</button>
			</div>
		</div>
	</header>

	<!-- =========================================================================
	     2. MAIN VIEW AREA (KANBAN / LIST / UTM-PIVOT)
	     ========================================================================= -->
	{#if activeView === 'kanban'}
		<!-- Odoo-Style Kanban Board -->
		<div class="odoo-kanban-board">
			{#each stages as st}
				{@const stageLeads = filteredLeads.filter(l => l.stage === st.id)}
				{@const stageRev = stageLeads.reduce((sum, l) => sum + (st.id !== 'lost' ? l.expectedRevenue : 0), 0)}
				
				<div class="kanban-column" style="--stage-color: {st.color}">
					<!-- Column Header -->
					<div class="column-header">
						<div class="col-title-row">
							<span class="col-icon">{st.icon}</span>
							<span class="col-name">{st.label}</span>
							<span class="col-badge font-mono">{stageLeads.length}</span>
						</div>
						<div class="col-sum-row">
							<span class="col-rev font-mono">{formatCurrency(stageRev)}</span>
							<button 
								type="button" 
								class="quick-add-btn" 
								onclick={() => {
									newLeadStage = st.id as any;
									showNewLeadModal = true;
								}}
								title="Add lead in {st.label}"
							>
								+
							</button>
						</div>
						<!-- Progress bar under header -->
						<div class="stage-bar-line" style="background: {st.color}"></div>
					</div>

					<!-- Cards Container -->
					<div class="column-cards-container">
						{#if stageLeads.length === 0}
							<div class="empty-column-box">
								<span class="empty-icon">📭</span>
								<span class="empty-text">No leads in this stage</span>
							</div>
						{:else}
							{#each stageLeads as l}
								<!-- Odoo Lead Card -->
								<div 
									class="odoo-lead-card" 
									onclick={() => openLead(l)} 
									role="button" 
									tabindex="0"
									onkeydown={(e) => { if (e.key === 'Enter') openLead(l); }}
								>
									<!-- Card Top Row: Destination Tag & Priority Stars -->
									<div class="card-top-row">
										<span class="dest-badge">
											{l.destination}
										</span>
										<div class="stars-rating" title="Priority Rating">
											{#each [1, 2, 3] as star}
												<button 
													type="button" 
													class="star-btn {l.priority >= star ? 'star-filled' : 'star-empty'}"
													onclick={(e) => setPriority(l.id, star as any, e)}
												>
													★
												</button>
											{/each}
										</div>
									</div>

									<!-- Card Title & Contact -->
									<div class="card-body-section">
										<h4 class="lead-title">{l.name}</h4>
										<div class="lead-sub-info">
											<span class="origin-city">📍 {l.city}</span> • 
											<span class="pax-tag">👥 {l.paxAdults + l.paxChildren} Pax ({l.tourType})</span>
										</div>
									</div>

									<!-- UTM Marketing Attribution Tag (CRITICAL) -->
									<div class="card-utm-row" title="UTM Source & Campaign">
										<span class="utm-source-tag {l.utm.source}">
											{#if l.utm.source === 'facebook'}📘 Meta
											{:else if l.utm.source === 'instagram'}📸 Instagram
											{:else if l.utm.source === 'google'}🔍 Google Ads
											{:else if l.utm.source === 'whatsapp'}💬 WhatsApp
											{:else}🌐 Organic{/if}
										</span>
										<span class="utm-campaign-snippet">
											{l.utm.campaign.replace(/_/g, ' ')}
										</span>
									</div>

									<!-- Next Activity Reminder -->
									{#if l.nextActivity}
										<div class="card-activity-pill {l.nextActivity.status}">
											<span class="act-icon">
												{#if l.nextActivity.type === 'Call'}📞
												{:else if l.nextActivity.type === 'WhatsApp'}💬
												{:else if l.nextActivity.type === 'Send Itinerary'}🗺️
												{:else}🛂{/if}
											</span>
											<span class="act-text">{l.nextActivity.summary}</span>
										</div>
									{/if}

									<!-- Card Footer: Expected Revenue & Agent Avatar -->
									<div class="card-footer-row">
										<div class="rev-block">
											<span class="rev-amount font-mono">{formatCurrency(l.expectedRevenue)}</span>
											<span class="prob-tag font-mono">{l.probability}%</span>
										</div>
										<div class="agent-avatar-circle" title="Assigned Agent: {l.assignedAgent.name}">
											{l.assignedAgent.initials}
										</div>
									</div>
								</div>
							{/each}
						{/if}
					</div>
				</div>
			{/each}
		</div>
	{:else if activeView === 'list'}
		<!-- Odoo-Style Table List View -->
		<div class="odoo-list-view">
			<div class="table-responsive">
				<table class="odoo-table">
					<thead>
						<tr>
							<th style="width: 40px;"><input type="checkbox" /></th>
							<th>Lead / Contact</th>
							<th>Destination &amp; Pax</th>
							<th>Origin City</th>
							<th>Stage</th>
							<th>Expected Revenue</th>
							<th>Probability</th>
							<th>UTM Source</th>
							<th>UTM Campaign</th>
							<th>Next Activity</th>
							<th>Agent</th>
							<th>Actions</th>
						</tr>
					</thead>
					<tbody>
						{#each filteredLeads as l}
							<tr onclick={() => openLead(l)} class="table-row-clickable">
								<td onclick={(e) => e.stopPropagation()}><input type="checkbox" /></td>
								<td>
									<div class="lead-contact-cell">
										<span class="cell-lead-name">{l.name}</span>
										<span class="cell-lead-phone font-mono">{l.phone}</span>
									</div>
								</td>
								<td>
									<span class="cell-dest font-semibold">{l.destination}</span>
									<span class="cell-meta text-xs block">{l.tourType} • {l.paxAdults + l.paxChildren} Pax</span>
								</td>
								<td>📍 {l.city}</td>
								<td>
									<span class="stage-badge-pill {l.stage}">
										{l.stage.toUpperCase()}
									</span>
								</td>
								<td class="font-mono font-bold text-emerald">{formatCurrency(l.expectedRevenue)}</td>
								<td class="font-mono">{l.probability}%</td>
								<td>
									<span class="utm-source-tag {l.utm.source}">
										{l.utm.source}
									</span>
								</td>
								<td class="font-mono text-xs">{l.utm.campaign}</td>
								<td>
									{#if l.nextActivity}
										<span class="text-xs">{l.nextActivity.type}: {l.nextActivity.summary}</span>
									{:else}
										<span class="text-xs text-muted">—</span>
									{/if}
								</td>
								<td>
									<span class="agent-badge">{l.assignedAgent.name}</span>
								</td>
								<td onclick={(e) => e.stopPropagation()}>
									<button type="button" class="btn btn-xs btn-outline" onclick={() => openLead(l)}>
										Open ➔
									</button>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</div>
	{:else if activeView === 'utm-pivot'}
		<!-- Odoo-Style UTM Attribution Matrix & Analytics -->
		<div class="odoo-utm-view">
			<div class="utm-masthead-card">
				<div>
					<h3 class="utm-title">📈 Marketing Campaign &amp; UTM Attribution Intelligence</h3>
					<p class="utm-desc">
						Track which digital ad channels (Facebook, Instagram, Google Ads, WhatsApp Meta) are generating the highest qualified travel enquiries and booked tour packages.
					</p>
				</div>
				<button type="button" class="btn btn-secondary btn-sm" onclick={() => toast.success('UTM Analytics exported to Excel / CSV')}>
					📥 Export Attribution CSV
				</button>
			</div>

			<div class="utm-table-card">
				<table class="utm-analytics-table">
					<thead>
						<tr>
							<th>Channel / UTM Source</th>
							<th class="text-right">Active Campaigns</th>
							<th class="text-right">Total Inbound Leads</th>
							<th class="text-right">Qualified Enquiries</th>
							<th class="text-right">Won / Booked Deals</th>
							<th class="text-right">Win Conversion %</th>
							<th class="text-right">Pipeline Value</th>
							<th class="text-right">Realized Revenue</th>
						</tr>
					</thead>
					<tbody>
						{#each utmAttribution as row}
							<tr>
								<td>
									<div class="source-cell-title">
										<span class="utm-source-tag {row.source}">{row.source.toUpperCase()}</span>
									</div>
								</td>
								<td class="text-right font-mono">{row.campaignCount}</td>
								<td class="text-right font-mono font-semibold">{row.leadsCount}</td>
								<td class="text-right font-mono text-blue">{row.qualified}</td>
								<td class="text-right font-mono font-bold text-emerald">{row.wonCount}</td>
								<td class="text-right font-mono font-bold">
									<div class="conv-bar-wrap">
										<span>{row.conversionRate}%</span>
										<div class="conv-bar" style="width: {row.conversionRate}%"></div>
									</div>
								</td>
								<td class="text-right font-mono">{formatCurrency(row.totalRevenue)}</td>
								<td class="text-right font-mono font-bold text-emerald">{formatCurrency(row.wonRevenue)}</td>
							</tr>
						{/each}
					</tbody>
					<tfoot>
						<tr class="tfoot-totals">
							<td><strong>Total Agency Attribution</strong></td>
							<td class="text-right font-mono"><strong>4</strong></td>
							<td class="text-right font-mono font-bold"><strong>{stats.total}</strong></td>
							<td class="text-right font-mono font-bold"><strong>{leads.filter(l => ['qualified', 'proposition', 'negotiation', 'won'].includes(l.stage)).length}</strong></td>
							<td class="text-right font-mono font-bold text-emerald"><strong>{stats.wonCount}</strong></td>
							<td class="text-right font-mono font-bold"><strong>{stats.winRate}%</strong></td>
							<td class="text-right font-mono font-bold"><strong>{formatCurrency(stats.totalRev)}</strong></td>
							<td class="text-right font-mono font-bold text-emerald"><strong>{formatCurrency(stats.wonRev)}</strong></td>
						</tr>
					</tfoot>
				</table>
			</div>
		</div>
	{/if}
</div>

<!-- =========================================================================
     3. ODOO-STYLE LEAD DETAIL DRAWER & FULL SPECIFICATIONS
     ========================================================================= -->
{#if showLeadDrawer && selectedLead}
	<div class="odoo-drawer-backdrop" onclick={() => (showLeadDrawer = false)}>
		<div class="odoo-drawer-panel" onclick={(e) => e.stopPropagation()} role="dialog">
			<!-- Top Drawer Header with Stage Progress Chevrons -->
			<div class="drawer-header-bar">
				<div class="dh-left">
					<span class="dh-lead-num font-mono">{selectedLead.leadNumber}</span>
					<h2 class="dh-title">{selectedLead.name}</h2>
				</div>
				<button type="button" class="drawer-close-btn" onclick={() => (showLeadDrawer = false)}>✕</button>
			</div>

			<!-- Odoo Stage Chevron Bar -->
			<div class="odoo-chevron-pipeline">
				{#each stages as st}
					<button 
						type="button" 
						class="chevron-step {selectedLead.stage === st.id ? 'active' : ''}" 
						onclick={() => updateStage(selectedLead!.id, st.id as any)}
					>
						<span class="ch-icon">{st.icon}</span>
						<span class="ch-label">{st.label}</span>
					</button>
				{/each}
			</div>

			<!-- Quick Action Buttons -->
			<div class="drawer-actions-strip">
				<button type="button" class="btn btn-sm btn-emerald" onclick={() => updateStage(selectedLead!.id, 'won')}>
					🏆 Mark as Won
				</button>
				<button type="button" class="btn btn-sm btn-danger-outline" onclick={() => updateStage(selectedLead!.id, 'lost')}>
					❌ Mark as Lost
				</button>
				<a 
					href={`https://wa.me/${selectedLead.whatsapp.replace(/[^0-9]/g, '')}?text=${encodeURIComponent(`Hello ${selectedLead.name}, greetings from Travel & Tours OS! Regarding your ${selectedLead.destination} enquiry...`)}`} 
					target="_blank" 
					class="btn btn-sm btn-whatsapp"
				>
					📱 Send WhatsApp
				</a>
				<button type="button" class="btn btn-sm btn-outline" onclick={() => toast.success('Itinerary builder launched for ' + selectedLead?.destination)}>
					🗺️ Generate Itinerary
				</button>
				<button type="button" class="btn btn-sm btn-primary" onclick={() => (showBookingModal = true)}>
					✈️ Convert to Booking
				</button>
			</div>

			<!-- Tabs Navigation -->
			<div class="drawer-tabs-nav">
				<button 
					type="button" 
					class="dtab-btn {drawerActiveTab === 'specs' ? 'active' : ''}"
					onclick={() => (drawerActiveTab = 'specs')}
				>
					🧳 Tour &amp; Travel Specs
				</button>
				<button 
					type="button" 
					class="dtab-btn {drawerActiveTab === 'contact' ? 'active' : ''}"
					onclick={() => (drawerActiveTab = 'contact')}
				>
					👤 Customer &amp; Billing
				</button>
				<button 
					type="button" 
					class="dtab-btn {drawerActiveTab === 'utm' ? 'active' : ''}"
					onclick={() => (drawerActiveTab = 'utm')}
				>
					🎯 UTM &amp; Marketing Attribution
				</button>
				<button 
					type="button" 
					class="dtab-btn {drawerActiveTab === 'chatter' ? 'active' : ''}"
					onclick={() => (drawerActiveTab = 'chatter')}
				>
					💬 Odoo Chatter &amp; Activity
				</button>
			</div>

			<!-- Drawer Tab Content Body -->
			<div class="drawer-content-body">
				{#if drawerActiveTab === 'specs'}
					<div class="specs-grid">
						<div class="form-group">
							<label class="form-label">Destination Package</label>
							<input type="text" bind:value={selectedLead.destination} class="form-input" />
						</div>
						<div class="form-group">
							<label class="form-label">Tour Type</label>
							<select bind:value={selectedLead.tourType} class="form-select">
								<option value="Family">Family Tour</option>
								<option value="Honeymoon">Honeymoon Special</option>
								<option value="Corporate">Corporate / MICE</option>
								<option value="Group">Group Departure</option>
								<option value="Solo">Solo Traveler</option>
								<option value="Pilgrimage">Pilgrimage Circuit</option>
							</select>
						</div>
						<div class="form-group">
							<label class="form-label">Tentative Travel Date</label>
							<input type="date" bind:value={selectedLead.travelDate} class="form-input" />
						</div>
						<div class="form-group">
							<label class="form-label">Duration (Nights)</label>
							<input type="number" bind:value={selectedLead.durationNights} class="form-input" min="1" />
						</div>
						<div class="form-group">
							<label class="form-label">Adults Count</label>
							<input type="number" bind:value={selectedLead.paxAdults} class="form-input" min="1" />
						</div>
						<div class="form-group">
							<label class="form-label">Children Count</label>
							<input type="number" bind:value={selectedLead.paxChildren} class="form-input" min="0" />
						</div>
						<div class="form-group">
							<label class="form-label">Hotel Category</label>
							<select bind:value={selectedLead.hotelPreference} class="form-select">
								<option value="3 Star Standard">3 Star Standard</option>
								<option value="4 Star Luxury">4 Star Luxury</option>
								<option value="5 Star Deluxe">5 Star Deluxe</option>
								<option value="Houseboat / Heritage">Houseboat / Heritage</option>
							</select>
						</div>
						<div class="form-group">
							<label class="form-label">Cab / Chauffeur Transport</label>
							<select bind:value={selectedLead.cabRequirement} class="form-select">
								<option value="Innova Crysta AC">Innova Crysta AC</option>
								<option value="Sedan">Sedan (Dzire / Etios)</option>
								<option value="Tempo Traveller">Tempo Traveller (12/17 Seater)</option>
								<option value="Luxury Coach">Luxury Coach (Volvo)</option>
							</select>
						</div>
						<div class="form-group">
							<label class="form-label">Expected Revenue (₹)</label>
							<input type="number" bind:value={selectedLead.expectedRevenue} class="form-input font-mono font-bold" />
						</div>
						<div class="form-group">
							<label class="form-label">Probability (%)</label>
							<input type="number" bind:value={selectedLead.probability} class="form-input font-mono" min="0" max="100" />
						</div>
					</div>

					<div class="form-group full-width" style="margin-top: 1rem;">
						<label class="form-label">Customer Special Preferences &amp; Requirements</label>
						<textarea bind:value={selectedLead.notes} rows="3" class="form-textarea"></textarea>
					</div>
				{:else if drawerActiveTab === 'contact'}
					<div class="specs-grid">
						<div class="form-group">
							<label class="form-label">Full Name</label>
							<input type="text" bind:value={selectedLead.name} class="form-input" />
						</div>
						<div class="form-group">
							<label class="form-label">Group / Organization Name</label>
							<input type="text" bind:value={selectedLead.groupTitle} class="form-input" />
						</div>
						<div class="form-group">
							<label class="form-label">Primary Phone</label>
							<input type="text" bind:value={selectedLead.phone} class="form-input font-mono" />
						</div>
						<div class="form-group">
							<label class="form-label">WhatsApp Number</label>
							<input type="text" bind:value={selectedLead.whatsapp} class="form-input font-mono" />
						</div>
						<div class="form-group">
							<label class="form-label">Email Address</label>
							<input type="email" bind:value={selectedLead.email} class="form-input" />
						</div>
						<div class="form-group">
							<label class="form-label">Origin City</label>
							<input type="text" bind:value={selectedLead.city} class="form-input" />
						</div>
						<div class="form-group">
							<label class="form-label">Origin State</label>
							<input type="text" bind:value={selectedLead.state} class="form-input" />
						</div>
						<div class="form-group">
							<label class="form-label">Assigned Tour Planner</label>
							<input type="text" bind:value={selectedLead.assignedAgent.name} class="form-input" />
						</div>
					</div>
				{:else if drawerActiveTab === 'utm'}
					<!-- UTM Attribution Intelligence Breakdown -->
					<div class="utm-details-container">
						<div class="utm-info-alert">
							<span>🎯</span>
							<div>
								<strong>UTM Tracking Parameter Breakdown</strong>
								<p>These parameters allow your marketing team to identify the exact ad creative, search query, and platform that prompted this booking enquiry.</p>
							</div>
						</div>

						<div class="utm-grid-fields">
							<div class="utm-field-box">
								<span class="utm-param-label">UTM Source (Platform)</span>
								<div class="utm-param-val">
									<span class="utm-source-tag {selectedLead.utm.source}">{selectedLead.utm.source}</span>
								</div>
							</div>
							<div class="utm-field-box">
								<span class="utm-param-label">UTM Medium (Channel Type)</span>
								<div class="utm-param-val font-mono">{selectedLead.utm.medium}</div>
							</div>
							<div class="utm-field-box">
								<span class="utm-param-label">UTM Campaign</span>
								<div class="utm-param-val font-mono text-blue font-bold">{selectedLead.utm.campaign}</div>
							</div>
							<div class="utm-field-box">
								<span class="utm-param-label">UTM Search Keyword / Term</span>
								<div class="utm-param-val font-mono">{selectedLead.utm.term || '—'}</div>
							</div>
							<div class="utm-field-box">
								<span class="utm-param-label">UTM Ad Creative / Content</span>
								<div class="utm-param-val font-mono">{selectedLead.utm.content || '—'}</div>
							</div>
							<div class="utm-field-box">
								<span class="utm-param-label">Referrer Domain</span>
								<div class="utm-param-val font-mono">{selectedLead.utm.referrerDomain || 'Direct Inbound'}</div>
							</div>
						</div>

						<!-- Full Landing URL with Copy Button -->
						<div class="full-url-box">
							<span class="url-label">Landing Page Destination URL with Full UTM Query String</span>
							<div class="url-input-row">
								<input type="text" readonly value={selectedLead.utm.landingPageUrl} class="url-input font-mono" />
								<button type="button" class="btn btn-sm btn-secondary" onclick={() => copyToClipboard(selectedLead!.utm.landingPageUrl)}>
									📋 Copy URL
								</button>
							</div>
						</div>
					</div>
				{:else if drawerActiveTab === 'chatter'}
					<!-- Odoo Chatter & Activity Timeline -->
					<div class="chatter-container">
						<!-- Schedule Activity Box -->
						<div class="schedule-activity-box">
							<h4 class="box-title">⏰ Schedule Next Sales Activity</h4>
							<div class="activity-inputs-row">
								<select bind:value={newActivityType} class="act-select">
									<option value="WhatsApp">💬 WhatsApp Follow-up</option>
									<option value="Call">📞 Phone Call</option>
									<option value="Send Itinerary">🗺️ Send Itinerary</option>
									<option value="Meeting">🤝 Meeting / Video Call</option>
								</select>
								<input type="date" bind:value={newActivityDate} class="act-date-input" />
								<input type="text" bind:value={newActivitySummary} placeholder="Summary of next step..." class="act-summary-input" />
								<button type="button" class="btn btn-sm btn-primary" onclick={scheduleNewActivity}>
									Schedule
								</button>
							</div>
						</div>

						<!-- Add Note Box -->
						<div class="add-note-box">
							<textarea bind:value={newChatterNote} placeholder="Log internal tour notes, hotel quotes, or flight discussions..." rows="2" class="chatter-textarea"></textarea>
							<button type="button" class="btn btn-sm btn-secondary" onclick={addChatterNote}>
								📝 Log Internal Note
							</button>
						</div>

						<!-- Timeline History -->
						<div class="activities-timeline">
							<h4 class="timeline-title">Audit Trail &amp; Activity Stream</h4>
							{#each selectedLead.activities as act}
								<div class="timeline-item">
									<div class="tl-bullet"></div>
									<div class="tl-content">
										<div class="tl-meta">
											<strong class="tl-user">{act.user}</strong>
											<span class="tl-time">{act.timestamp}</span>
										</div>
										<p class="tl-text">{act.text}</p>
									</div>
								</div>
							{/each}
						</div>
					</div>
				{/if}
			</div>

			<!-- Drawer Footer -->
			<div class="drawer-footer-bar">
				<button type="button" class="btn btn-secondary" onclick={() => (showLeadDrawer = false)}>
					Cancel
				</button>
				<button type="button" class="btn btn-primary" onclick={saveLeadChanges}>
					💾 Save Changes
				</button>
			</div>
		</div>
	</div>
{/if}

<!-- =========================================================================
     4. QUICK NEW LEAD MODAL
     ========================================================================= -->
{#if showNewLeadModal}
	<div class="odoo-drawer-backdrop" onclick={() => (showNewLeadModal = false)}>
		<div class="odoo-modal-card" onclick={(e) => e.stopPropagation()} role="dialog">
			<div class="modal-card-header">
				<h3>+ Create New Travel Lead (Odoo Pipeline)</h3>
				<button type="button" class="drawer-close-btn" onclick={() => (showNewLeadModal = false)}>✕</button>
			</div>
			<div class="modal-card-body">
				<div class="specs-grid">
					<div class="form-group">
						<label class="form-label">Contact / Family Name *</label>
						<input type="text" id="nl-name" placeholder="e.g. Anand Kumar & Family" class="form-input" />
					</div>
					<div class="form-group">
						<label class="form-label">Phone / WhatsApp *</label>
						<input type="text" id="nl-phone" placeholder="+91 98840 12345" class="form-input font-mono" />
					</div>
					<div class="form-group">
						<label class="form-label">Destination *</label>
						<input type="text" id="nl-dest" placeholder="e.g. Kashmir Valley Paradise" class="form-input" />
					</div>
					<div class="form-group">
						<label class="form-label">Origin City</label>
						<input type="text" id="nl-city" placeholder="Chennai / Madurai" class="form-input" />
					</div>
					<div class="form-group">
						<label class="form-label">Expected Revenue (₹)</label>
						<input type="number" id="nl-rev" placeholder="150000" class="form-input font-mono" />
					</div>
					<div class="form-group">
						<label class="form-label">UTM Source (Attribution)</label>
						<select id="nl-source" class="form-select">
							<option value="facebook">Meta / Facebook</option>
							<option value="instagram">Instagram Ads / Reels</option>
							<option value="google">Google CPC / Search</option>
							<option value="whatsapp">WhatsApp Direct Campaign</option>
							<option value="organic">Organic / Direct</option>
						</select>
					</div>
				</div>
			</div>
			<div class="modal-card-footer">
				<button type="button" class="btn btn-secondary" onclick={() => (showNewLeadModal = false)}>
					Cancel
				</button>
				<button 
					type="button" 
					class="btn btn-primary" 
					onclick={() => {
						const nameEl = document.getElementById('nl-name') as HTMLInputElement;
						const phoneEl = document.getElementById('nl-phone') as HTMLInputElement;
						const destEl = document.getElementById('nl-dest') as HTMLInputElement;
						const cityEl = document.getElementById('nl-city') as HTMLInputElement;
						const revEl = document.getElementById('nl-rev') as HTMLInputElement;
						const srcEl = document.getElementById('nl-source') as HTMLSelectElement;

						if (!nameEl?.value) {
							toast.error('Customer name is required');
							return;
						}

						const newLead: TravelLead = {
							id: 'lead-' + Date.now(),
							leadNumber: 'LEAD-2026-' + (leads.length + 100),
							name: nameEl.value,
							groupTitle: nameEl.value + ' Tour',
							phone: phoneEl?.value || '+91 90000 00000',
							whatsapp: phoneEl?.value || '+91 90000 00000',
							email: 'contact@example.com',
							city: cityEl?.value || 'Chennai',
							state: 'Tamil Nadu',
							destination: destEl?.value || 'Kashmir Paradise',
							tourType: 'Family',
							travelDate: '2026-11-20',
							durationNights: 5,
							paxAdults: 2,
							paxChildren: 0,
							hotelPreference: '4 Star Luxury',
							cabRequirement: 'Innova Crysta AC',
							mealPlan: 'MAP (Breakfast + Dinner)',
							stage: newLeadStage,
							expectedRevenue: Number(revEl?.value) || 120000,
							probability: 60,
							priority: 2,
							assignedAgent: { name: 'Priya Menon', avatar: '👩‍💼', initials: 'PM' },
							tags: ['New Enquiry'],
							utm: {
								source: srcEl?.value || 'facebook',
								medium: 'cpc',
								campaign: 'direct_inbound_2026',
								landingPageUrl: 'https://mrbiz.tours/enquiry?utm_source=' + (srcEl?.value || 'facebook'),
								firstTouchDate: new Date().toISOString()
							},
							notes: 'Created via Odoo quick lead creator.',
							createdAt: new Date().toISOString(),
							activities: [
								{ id: 'act-' + Date.now(), user: 'Current Agent', type: 'create', text: 'Lead created in pipeline stage: ' + newLeadStage, timestamp: new Date().toLocaleString('en-IN') }
							]
						};

						leads.unshift(newLead);
						showNewLeadModal = false;
						toast.success('Lead created in Odoo Pipeline!');
					}}
				>
					Create Lead
				</button>
			</div>
		</div>
	</div>
{/if}

<CreateBookingModal 
	open={showBookingModal} 
	initialCustomerName={selectedLead?.name} 
	initialDestination={selectedLead?.destination} 
	initialAmount={selectedLead ? formatCurrency(selectedLead.expectedRevenue) : undefined} 
	onclose={() => (showBookingModal = false)} 
	onsave={(b) => { 
		if (selectedLead) updateStage(selectedLead.id, 'won'); 
		toast.success(`Booking ${b.id} confirmed for ${b.customerName}!`); 
	}} 
/>

<style>
	/* ==========================================================================
	   ODOO CRM PIPELINE STYLING (ENTERPRISE HIGH-CONVERTING DESIGN)
	   ========================================================================== */

	.odoo-crm-canvas {
		display: flex;
		flex-direction: column;
		min-height: calc(100vh - 64px);
		background: #f1f5f9;
		color: #0f172a;
		font-family: var(--font-sans, system-ui, -apple-system, sans-serif);
		padding: 1.25rem 1.75rem;
		gap: 1.25rem;
		box-sizing: border-box;
	}

	/* Odoo Control Panel Masthead */
	.odoo-control-panel {
		background: #ffffff;
		border-radius: 12px;
		border: 1px solid #e2e8f0;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
		padding: 1rem 1.25rem;
		display: flex;
		flex-direction: column;
		gap: 0.85rem;
	}

	.cp-top-bar {
		display: flex;
		justify-content: space-between;
		align-items: center;
		flex-wrap: wrap;
		gap: 1rem;
	}

	.cp-breadcrumbs {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 1.1rem;
		font-weight: 700;
	}

	.crumb-link {
		color: #64748b;
		text-decoration: none;
		transition: color 0.15s ease;
	}
	.crumb-link:hover {
		color: #2563eb;
	}

	.crumb-separator {
		color: #cbd5e1;
	}

	.crumb-current {
		color: #0f172a;
	}

	.crm-odoo-badge {
		background: #7c3aed;
		color: #ffffff;
		font-size: 0.65rem;
		font-weight: 800;
		padding: 2px 7px;
		border-radius: 4px;
		letter-spacing: 0.05em;
	}

	/* KPI Stat Pills */
	.cp-stat-pills {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		flex-wrap: wrap;
	}

	.stat-pill {
		background: #f8fafc;
		border: 1px solid #e2e8f0;
		border-radius: 8px;
		padding: 4px 10px;
		display: flex;
		flex-direction: column;
	}

	.stat-pill.highlight-green {
		background: #ecfdf5;
		border-color: #a7f3d0;
	}

	.pill-label {
		font-size: 0.65rem;
		text-transform: uppercase;
		color: #64748b;
		font-weight: 600;
	}

	.pill-val {
		font-size: 0.875rem;
		font-weight: 700;
		color: #0f172a;
	}

	/* View Switcher */
	.view-switcher-group {
		display: flex;
		background: #f1f5f9;
		border-radius: 8px;
		padding: 3px;
		border: 1px solid #e2e8f0;
	}

	.view-btn {
		background: transparent;
		border: none;
		padding: 5px 12px;
		font-size: 0.8125rem;
		font-weight: 600;
		color: #64748b;
		border-radius: 6px;
		cursor: pointer;
		display: flex;
		align-items: center;
		gap: 0.35rem;
		transition: all 0.15s ease;
	}

	.view-btn.active {
		background: #ffffff;
		color: #0f172a;
		box-shadow: 0 1px 2px rgba(0,0,0,0.06);
	}

	/* Search & Filters Bar */
	.cp-search-bar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		flex-wrap: wrap;
		gap: 0.75rem;
		border-top: 1px solid #f1f5f9;
		padding-top: 0.75rem;
	}

	.search-input-wrap {
		display: flex;
		align-items: center;
		position: relative;
		flex: 1;
		min-width: 280px;
	}

	.search-icon {
		position: absolute;
		left: 10px;
		font-size: 0.85rem;
		color: #94a3b8;
	}

	.odoo-search-input {
		width: 100%;
		padding: 7px 30px 7px 32px;
		border-radius: 8px;
		border: 1px solid #cbd5e1;
		font-size: 0.85rem;
		background: #ffffff;
		color: #0f172a;
		outline: none;
		transition: border-color 0.15s ease;
	}

	.odoo-search-input:focus {
		border-color: #2563eb;
		box-shadow: 0 0 0 2px rgba(37, 99, 235, 0.15);
	}

	.clear-search-btn {
		position: absolute;
		right: 8px;
		background: transparent;
		border: none;
		color: #94a3b8;
		font-size: 0.8rem;
		cursor: pointer;
	}

	.filters-cluster {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		flex-wrap: wrap;
	}

	.odoo-filter-select {
		padding: 6px 10px;
		border-radius: 6px;
		border: 1px solid #cbd5e1;
		background: #ffffff;
		font-size: 0.8125rem;
		color: #334155;
		font-weight: 500;
	}

	.new-lead-btn {
		font-weight: 700;
	}

	/* ==========================================================================
	     KANBAN BOARD COLUMNS & CARDS
	     ========================================================================= */

	.odoo-kanban-board {
		display: flex;
		gap: 1rem;
		overflow-x: auto;
		padding-bottom: 1rem;
		flex: 1;
		align-items: flex-start;
	}

	.kanban-column {
		width: 320px;
		min-width: 320px;
		background: #f8fafc;
		border: 1px solid #e2e8f0;
		border-radius: 10px;
		display: flex;
		flex-direction: column;
		max-height: calc(100vh - 210px);
		box-shadow: 0 1px 3px rgba(0,0,0,0.03);
	}

	.column-header {
		padding: 0.85rem 1rem 0.65rem 1rem;
		background: #ffffff;
		border-top-left-radius: 10px;
		border-top-right-radius: 10px;
		position: relative;
	}

	.col-title-row {
		display: flex;
		align-items: center;
		gap: 0.4rem;
	}

	.col-name {
		font-size: 0.9rem;
		font-weight: 800;
		color: #1e293b;
		flex: 1;
	}

	.col-badge {
		background: #f1f5f9;
		color: #475569;
		border-radius: 12px;
		padding: 1px 7px;
		font-size: 0.75rem;
		font-weight: 700;
	}

	.col-sum-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-top: 4px;
	}

	.col-rev {
		font-size: 0.75rem;
		font-weight: 700;
		color: #64748b;
	}

	.quick-add-btn {
		background: transparent;
		border: none;
		color: #64748b;
		font-size: 1rem;
		font-weight: 700;
		cursor: pointer;
		padding: 0 4px;
	}
	.quick-add-btn:hover {
		color: #0f172a;
	}

	.stage-bar-line {
		position: absolute;
		bottom: 0;
		left: 0;
		right: 0;
		height: 3px;
	}

	.column-cards-container {
		padding: 0.75rem;
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
		overflow-y: auto;
		flex: 1;
	}

	.empty-column-box {
		padding: 2rem 1rem;
		text-align: center;
		color: #94a3b8;
		display: flex;
		flex-direction: column;
		gap: 0.4rem;
	}

	/* Odoo Lead Card */
	.odoo-lead-card {
		background: #ffffff;
		border: 1px solid #e2e8f0;
		border-radius: 8px;
		padding: 0.85rem;
		cursor: pointer;
		display: flex;
		flex-direction: column;
		gap: 0.55rem;
		box-shadow: 0 1px 2px rgba(0,0,0,0.04);
		transition: transform 0.12s ease, box-shadow 0.12s ease, border-color 0.12s ease;
	}

	.odoo-lead-card:hover {
		transform: translateY(-2px);
		box-shadow: 0 4px 10px rgba(0,0,0,0.08);
		border-color: #cbd5e1;
	}

	.card-top-row {
		display: flex;
		justify-content: space-between;
		align-items: center;
	}

	.dest-badge {
		font-size: 0.7rem;
		font-weight: 700;
		color: #1e40af;
		background: #eff6ff;
		padding: 2px 7px;
		border-radius: 4px;
	}

	.stars-rating {
		display: flex;
		gap: 1px;
	}

	.star-btn {
		background: transparent;
		border: none;
		cursor: pointer;
		font-size: 0.85rem;
		padding: 0;
	}

	.star-filled {
		color: #eab308;
	}
	.star-empty {
		color: #cbd5e1;
	}

	.lead-title {
		font-size: 0.925rem;
		font-weight: 700;
		color: #0f172a;
		margin: 0;
	}

	.lead-sub-info {
		font-size: 0.75rem;
		color: #64748b;
		margin-top: 2px;
	}

	/* UTM Row inside Card */
	.card-utm-row {
		display: flex;
		align-items: center;
		gap: 0.4rem;
		flex-wrap: wrap;
	}

	.utm-source-tag {
		font-size: 0.65rem;
		font-weight: 700;
		padding: 1px 6px;
		border-radius: 4px;
		text-transform: uppercase;
	}

	.utm-source-tag.facebook { background: #dbeafe; color: #1e40af; }
	.utm-source-tag.instagram { background: #fce7f3; color: #9d174d; }
	.utm-source-tag.google { background: #fef3c7; color: #92400e; }
	.utm-source-tag.whatsapp { background: #dcfce7; color: #166534; }
	.utm-source-tag.organic { background: #f1f5f9; color: #475569; }

	.utm-campaign-snippet {
		font-size: 0.68rem;
		color: #64748b;
		font-family: monospace;
	}

	/* Next Activity Pill */
	.card-activity-pill {
		display: flex;
		align-items: center;
		gap: 0.35rem;
		padding: 3px 8px;
		border-radius: 6px;
		font-size: 0.7rem;
		font-weight: 600;
	}

	.card-activity-pill.today {
		background: #fef3c7;
		color: #92400e;
		border: 1px solid #fde68a;
	}
	.card-activity-pill.planned {
		background: #f0fdf4;
		color: #166534;
		border: 1px solid #bbf7d0;
	}

	.card-footer-row {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding-top: 0.4rem;
		border-top: 1px solid #f1f5f9;
	}

	.rev-amount {
		font-size: 0.85rem;
		font-weight: 800;
		color: #0f172a;
	}

	.prob-tag {
		font-size: 0.7rem;
		color: #64748b;
		margin-left: 4px;
	}

	.agent-avatar-circle {
		width: 26px;
		height: 26px;
		border-radius: 50%;
		background: #3b82f6;
		color: #ffffff;
		font-size: 0.7rem;
		font-weight: 700;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	/* ==========================================================================
	     LIST VIEW & UTM ATTRIBUTION TABLE
	     ========================================================================= */

	.odoo-list-view, .odoo-utm-view {
		background: #ffffff;
		border-radius: 12px;
		border: 1px solid #e2e8f0;
		padding: 1rem;
		box-shadow: 0 1px 3px rgba(0,0,0,0.03);
	}

	.table-responsive {
		overflow-x: auto;
	}

	.odoo-table, .utm-analytics-table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.8125rem;
	}

	.odoo-table th, .utm-analytics-table th {
		background: #f8fafc;
		padding: 0.65rem 0.75rem;
		text-align: left;
		font-weight: 700;
		color: #475569;
		border-bottom: 1px solid #cbd5e1;
	}

	.odoo-table td, .utm-analytics-table td {
		padding: 0.65rem 0.75rem;
		border-bottom: 1px solid #f1f5f9;
		vertical-align: middle;
	}

	.table-row-clickable {
		cursor: pointer;
	}
	.table-row-clickable:hover {
		background: #f8fafc;
	}

	.stage-badge-pill {
		font-size: 0.65rem;
		font-weight: 700;
		padding: 2px 7px;
		border-radius: 4px;
	}

	.stage-badge-pill.new { background: #dbeafe; color: #1e40af; }
	.stage-badge-pill.qualified { background: #cffafe; color: #0e7490; }
	.stage-badge-pill.proposition { background: #ede9fe; color: #6d28d9; }
	.stage-badge-pill.negotiation { background: #fef3c7; color: #b45309; }
	.stage-badge-pill.won { background: #dcfce7; color: #15803d; }
	.stage-badge-pill.lost { background: #fee2e2; color: #b91c1c; }

	.utm-masthead-card {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding-bottom: 1rem;
		border-bottom: 1px solid #e2e8f0;
		margin-bottom: 1rem;
	}

	.utm-title {
		font-size: 1.1rem;
		font-weight: 800;
		color: #0f172a;
		margin: 0;
	}

	.utm-desc {
		font-size: 0.8125rem;
		color: #64748b;
		margin: 4px 0 0 0;
	}

	.conv-bar-wrap {
		display: flex;
		align-items: center;
		justify-content: flex-end;
		gap: 0.5rem;
	}

	.conv-bar {
		height: 6px;
		background: #10b981;
		border-radius: 3px;
		min-width: 4px;
		max-width: 60px;
	}

	.tfoot-totals td {
		border-top: 2px solid #cbd5e1;
		background: #f8fafc;
	}

	/* ==========================================================================
	     ODOO SLIDE-OVER LEAD DRAWER
	     ========================================================================= */

	.odoo-drawer-backdrop {
		position: fixed;
		inset: 0;
		background: rgba(15, 23, 42, 0.45);
		backdrop-filter: blur(2px);
		z-index: 9999;
		display: flex;
		justify-content: flex-end;
	}

	.odoo-drawer-panel {
		width: 760px;
		max-width: 95vw;
		background: #ffffff;
		height: 100vh;
		display: flex;
		flex-direction: column;
		box-shadow: -4px 0 24px rgba(0,0,0,0.15);
		overflow: hidden;
	}

	.drawer-header-bar {
		padding: 1rem 1.5rem;
		display: flex;
		justify-content: space-between;
		align-items: center;
		border-bottom: 1px solid #e2e8f0;
	}

	.dh-lead-num {
		font-size: 0.75rem;
		font-weight: 700;
		color: #64748b;
	}

	.dh-title {
		font-size: 1.25rem;
		font-weight: 800;
		color: #0f172a;
		margin: 2px 0 0 0;
	}

	.drawer-close-btn {
		width: 32px;
		height: 32px;
		border-radius: 6px;
		background: transparent;
		border: 1px solid #cbd5e1;
		cursor: pointer;
	}

	/* Odoo Chevron Pipeline Bar */
	.odoo-chevron-pipeline {
		display: flex;
		background: #f8fafc;
		border-bottom: 1px solid #e2e8f0;
		overflow-x: auto;
	}

	.chevron-step {
		flex: 1;
		padding: 10px 14px;
		background: transparent;
		border: none;
		border-right: 1px solid #e2e8f0;
		font-size: 0.75rem;
		font-weight: 700;
		color: #64748b;
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.35rem;
		white-space: nowrap;
		transition: all 0.15s ease;
	}

	.chevron-step.active {
		background: #2563eb;
		color: #ffffff;
	}

	.drawer-actions-strip {
		padding: 0.75rem 1.5rem;
		background: #ffffff;
		border-bottom: 1px solid #e2e8f0;
		display: flex;
		gap: 0.5rem;
		flex-wrap: wrap;
	}

	/* Tabs */
	.drawer-tabs-nav {
		display: flex;
		background: #f8fafc;
		border-bottom: 1px solid #e2e8f0;
		padding: 0 1.5rem;
	}

	.dtab-btn {
		background: transparent;
		border: none;
		border-bottom: 2px solid transparent;
		padding: 10px 14px;
		font-size: 0.8125rem;
		font-weight: 700;
		color: #64748b;
		cursor: pointer;
	}

	.dtab-btn.active {
		border-bottom-color: #2563eb;
		color: #2563eb;
	}

	.drawer-content-body {
		padding: 1.5rem;
		overflow-y: auto;
		flex: 1;
	}

	.specs-grid {
		display: grid;
		grid-template-columns: repeat(2, 1fr);
		gap: 1rem;
	}

	.form-group {
		display: flex;
		flex-direction: column;
		gap: 0.3rem;
	}

	.form-label {
		font-size: 0.75rem;
		font-weight: 700;
		color: #475569;
	}

	.form-input, .form-select, .form-textarea {
		padding: 7px 10px;
		border-radius: 6px;
		border: 1px solid #cbd5e1;
		font-size: 0.85rem;
		background: #ffffff;
		color: #0f172a;
		outline: none;
	}

	.form-input:focus, .form-select:focus, .form-textarea:focus {
		border-color: #2563eb;
	}

	/* UTM Details Container */
	.utm-details-container {
		display: flex;
		flex-direction: column;
		gap: 1.25rem;
	}

	.utm-info-alert {
		background: #eff6ff;
		border: 1px solid #bfdbfe;
		border-radius: 8px;
		padding: 0.85rem 1rem;
		display: flex;
		gap: 0.75rem;
		font-size: 0.8125rem;
		color: #1e40af;
	}
	.utm-info-alert p {
		margin: 2px 0 0 0;
		font-size: 0.75rem;
		color: #3b82f6;
	}

	.utm-grid-fields {
		display: grid;
		grid-template-columns: repeat(2, 1fr);
		gap: 1rem;
	}

	.utm-field-box {
		background: #f8fafc;
		border: 1px solid #e2e8f0;
		border-radius: 8px;
		padding: 0.75rem 1rem;
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}

	.utm-param-label {
		font-size: 0.7rem;
		font-weight: 700;
		color: #64748b;
		text-transform: uppercase;
	}

	.utm-param-val {
		font-size: 0.85rem;
		color: #0f172a;
	}

	.full-url-box {
		background: #f8fafc;
		border: 1px solid #e2e8f0;
		border-radius: 8px;
		padding: 1rem;
		display: flex;
		flex-direction: column;
		gap: 0.4rem;
	}

	.url-label {
		font-size: 0.75rem;
		font-weight: 700;
		color: #475569;
	}

	.url-input-row {
		display: flex;
		gap: 0.5rem;
	}

	.url-input {
		flex: 1;
		padding: 7px 10px;
		border-radius: 6px;
		border: 1px solid #cbd5e1;
		font-size: 0.75rem;
		background: #ffffff;
		color: #334155;
	}

	/* Chatter Styles */
	.chatter-container {
		display: flex;
		flex-direction: column;
		gap: 1.25rem;
	}

	.schedule-activity-box, .add-note-box {
		background: #f8fafc;
		border: 1px solid #e2e8f0;
		border-radius: 8px;
		padding: 1rem;
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	.box-title {
		font-size: 0.85rem;
		font-weight: 700;
		color: #1e293b;
		margin: 0;
	}

	.activity-inputs-row {
		display: flex;
		gap: 0.5rem;
		flex-wrap: wrap;
	}

	.act-select, .act-date-input, .act-summary-input {
		padding: 6px 10px;
		border-radius: 6px;
		border: 1px solid #cbd5e1;
		font-size: 0.8125rem;
		background: #ffffff;
	}

	.act-summary-input {
		flex: 1;
		min-width: 180px;
	}

	.chatter-textarea {
		width: 100%;
		padding: 8px 10px;
		border-radius: 6px;
		border: 1px solid #cbd5e1;
		font-size: 0.8125rem;
		background: #ffffff;
	}

	.activities-timeline {
		display: flex;
		flex-direction: column;
		gap: 0.85rem;
		margin-top: 0.5rem;
	}

	.timeline-title {
		font-size: 0.85rem;
		font-weight: 700;
		color: #475569;
		margin: 0 0 0.5rem 0;
		text-transform: uppercase;
	}

	.timeline-item {
		display: flex;
		gap: 0.85rem;
		position: relative;
	}

	.tl-bullet {
		width: 10px;
		height: 10px;
		border-radius: 50%;
		background: #3b82f6;
		margin-top: 5px;
		border: 2px solid #ffffff;
		box-shadow: 0 0 0 2px #3b82f6;
	}

	.tl-content {
		background: #f8fafc;
		border: 1px solid #e2e8f0;
		border-radius: 8px;
		padding: 0.65rem 0.85rem;
		flex: 1;
	}

	.tl-meta {
		display: flex;
		justify-content: space-between;
		font-size: 0.75rem;
		margin-bottom: 2px;
	}

	.tl-user {
		color: #1e293b;
	}

	.tl-time {
		color: #94a3b8;
	}

	.tl-text {
		font-size: 0.8125rem;
		color: #334155;
		margin: 0;
	}

	/* Drawer Footer Bar */
	.drawer-footer-bar {
		padding: 1rem 1.5rem;
		background: #ffffff;
		border-top: 1px solid #e2e8f0;
		display: flex;
		justify-content: flex-end;
		gap: 0.75rem;
	}

	/* Quick New Lead Modal */
	.odoo-modal-card {
		background: #ffffff;
		border-radius: 12px;
		width: 600px;
		max-width: 90vw;
		margin: auto;
		box-shadow: 0 10px 30px rgba(0,0,0,0.2);
		overflow: hidden;
	}

	.modal-card-header {
		padding: 1rem 1.25rem;
		display: flex;
		justify-content: space-between;
		align-items: center;
		border-bottom: 1px solid #e2e8f0;
	}

	.modal-card-header h3 {
		margin: 0;
		font-size: 1.1rem;
		font-weight: 800;
	}

	.modal-card-body {
		padding: 1.25rem;
	}

	.modal-card-footer {
		padding: 0.85rem 1.25rem;
		background: #f8fafc;
		border-top: 1px solid #e2e8f0;
		display: flex;
		justify-content: flex-end;
		gap: 0.5rem;
	}

	/* General Buttons & Badges */
	.btn {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		padding: 0.5rem 0.9rem;
		border-radius: 6px;
		font-weight: 600;
		font-size: 0.8125rem;
		cursor: pointer;
		border: 1px solid transparent;
		transition: all 0.15s ease;
		text-decoration: none;
	}

	.btn-primary { background: #2563eb; color: #ffffff; }
	.btn-primary:hover { background: #1d4ed8; }

	.btn-secondary { background: #ffffff; border-color: #cbd5e1; color: #334155; }
	.btn-secondary:hover { background: #f8fafc; color: #0f172a; }

	.btn-outline { background: transparent; border-color: #cbd5e1; color: #475569; }
	.btn-outline:hover { background: #f1f5f9; color: #0f172a; }

	.btn-emerald { background: #10b981; color: #ffffff; }
	.btn-emerald:hover { background: #059669; }

	.btn-danger-outline { background: transparent; border-color: #fca5a5; color: #dc2626; }
	.btn-danger-outline:hover { background: #fee2e2; }

	.btn-whatsapp { background: #22c55e; color: #ffffff; }
	.btn-whatsapp:hover { background: #16a34a; }

	.btn-sm { padding: 0.35rem 0.7rem; font-size: 0.75rem; }
	.btn-xs { padding: 0.2rem 0.5rem; font-size: 0.7rem; }

	.font-mono { font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace; }
	.font-semibold { font-weight: 600; }
	.font-bold { font-weight: 700; }
	.text-emerald { color: #059669; }
	.text-blue { color: #2563eb; }
	.text-right { text-align: right; }
</style>
