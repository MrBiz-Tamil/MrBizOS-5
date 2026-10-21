<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';

	let profile = $state<any>({ iec_code: '', ad_code: '', bank_swift: '', lut_number: '', lut_valid_from: null, lut_valid_to: null, default_port_loading: '', default_port_code: '' });
	let currencies = $state<any[]>([]);
	let rates = $state<any[]>([]);
	let rateCurrency = $state('USD');
	let rateValue = $state('');
	let rateDate = $state(new Date().toISOString().slice(0, 10));
	let notice = $state('');
	let error = $state('');

	async function load() {
		try {
			[profile, currencies, rates] = await Promise.all([api.tradeProfile(), api.tradeCurrencies(), api.tradeRates()]);
		} catch (e: any) {
			error = e.message;
		}
	}
	onMount(load);

	async function saveProfile(e: Event) {
		e.preventDefault();
		error = notice = '';
		try {
			profile = await api.tradeSaveProfile({
				...profile,
				lut_valid_from: profile.lut_valid_from || null,
				lut_valid_to: profile.lut_valid_to || null
			});
			notice = 'Trade profile saved.';
		} catch (err: any) {
			error = err.message;
		}
	}

	async function saveRate(e: Event) {
		e.preventDefault();
		error = notice = '';
		try {
			await api.tradeSetRate(rateCurrency, rateValue, rateDate);
			rateValue = '';
			rates = await api.tradeRates();
			notice = 'Rate saved.';
		} catch (err: any) {
			error = err.message;
		}
	}
</script>

<svelte:head><title>Trade Settings & Rates</title></svelte:head>

<div style="max-width: 860px;">
	<h1 style="margin-bottom: 0.25rem;">Trade settings & exchange rates</h1>
	<p style="color: var(--text-secondary); margin-bottom: 1rem;">Details printed on export documents, and the INR rates used to value foreign-currency invoices.</p>
	{#if notice}<div class="card" style="padding: 0.6rem 1rem; margin-bottom: 1rem; color: #34d399;">{notice}</div>{/if}
	{#if error}<div class="card" style="padding: 0.6rem 1rem; margin-bottom: 1rem; color: #fb7185;">{error}</div>{/if}

	<form class="card" style="padding: 1.25rem; margin-bottom: 1.25rem;" onsubmit={saveProfile}>
		<h3 style="margin-bottom: 0.75rem;">Company trade profile</h3>
		<div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 0.75rem;">
			<div class="form-group"><label class="form-label" for="iec">IEC (10 characters)</label><input id="iec" class="form-input" bind:value={profile.iec_code} maxlength="10" /></div>
			<div class="form-group"><label class="form-label" for="ad">Bank AD code</label><input id="ad" class="form-input" bind:value={profile.ad_code} /></div>
			<div class="form-group"><label class="form-label" for="sw">Bank SWIFT / BIC</label><input id="sw" class="form-input" bind:value={profile.bank_swift} maxlength="11" /></div>
			<div class="form-group"><label class="form-label" for="lut">LUT number</label><input id="lut" class="form-input" bind:value={profile.lut_number} /></div>
			<div class="form-group"><label class="form-label" for="lf">LUT valid from</label><input id="lf" type="date" class="form-input" bind:value={profile.lut_valid_from} /></div>
			<div class="form-group"><label class="form-label" for="lt">LUT valid to</label><input id="lt" type="date" class="form-input" bind:value={profile.lut_valid_to} /></div>
			<div class="form-group"><label class="form-label" for="pl">Usual port of loading</label><input id="pl" class="form-input" bind:value={profile.default_port_loading} /></div>
			<div class="form-group"><label class="form-label" for="pc">Port code</label><input id="pc" class="form-input" bind:value={profile.default_port_code} /></div>
		</div>
		<button class="btn btn-primary">Save profile</button>
	</form>

	<div class="card" style="padding: 1.25rem;">
		<h3 style="margin-bottom: 0.25rem;">Exchange rates (INR per 1 unit)</h3>
		<p style="color: var(--text-secondary); font-size: 0.8125rem; margin-bottom: 0.75rem;">Enter the rate you use for invoicing (for example the customs-notified rate for the shipping date). Rates are not fetched automatically.</p>
		<form onsubmit={saveRate} style="display: flex; gap: 0.75rem; align-items: end; flex-wrap: wrap; margin-bottom: 1rem;">
			<div class="form-group" style="margin: 0;"><label class="form-label" for="rc">Currency</label>
				<select id="rc" class="form-input" bind:value={rateCurrency}>{#each currencies.filter((c) => c.code !== 'INR') as c}<option value={c.code}>{c.code} - {c.name}</option>{/each}</select></div>
			<div class="form-group" style="margin: 0;"><label class="form-label" for="rv">Rate</label><input id="rv" class="form-input" bind:value={rateValue} required placeholder="83.25" /></div>
			<div class="form-group" style="margin: 0;"><label class="form-label" for="rd">Date</label><input id="rd" type="date" class="form-input" bind:value={rateDate} required /></div>
			<button class="btn btn-primary">Save rate</button>
		</form>
		<table style="width: 100%; font-size: 0.875rem;">
			<thead><tr style="text-align: left; color: var(--text-secondary);"><th>Date</th><th>Currency</th><th>Rate</th><th>Source</th></tr></thead>
			<tbody>
				{#each rates as r}<tr><td>{r.rate_date}</td><td>{r.currency}</td><td>{r.rate}</td><td>{r.source}</td></tr>
				{:else}<tr><td colspan="4" style="color: var(--text-secondary);">No rates yet.</td></tr>{/each}
			</tbody>
		</table>
	</div>
</div>
