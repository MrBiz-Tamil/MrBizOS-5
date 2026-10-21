<script lang="ts">
	import { api } from '$lib/api';

	let companyName = $state('');
	let fullName = $state('');
	let email = $state('');
	let phone = $state('');
	let state_ = $state('');
	let password = $state('');
	let accept = $state(false);
	let loading = $state(false);
	let error = $state('');
	let done = $state(false);

	const states = [
		'Andhra Pradesh', 'Arunachal Pradesh', 'Assam', 'Bihar', 'Chhattisgarh', 'Goa', 'Gujarat', 'Haryana',
		'Himachal Pradesh', 'Jharkhand', 'Karnataka', 'Kerala', 'Madhya Pradesh', 'Maharashtra', 'Manipur', 'Meghalaya',
		'Mizoram', 'Nagaland', 'Odisha', 'Punjab', 'Rajasthan', 'Sikkim', 'Tamil Nadu', 'Telangana', 'Tripura',
		'Uttar Pradesh', 'Uttarakhand', 'West Bengal', 'Delhi', 'Jammu and Kashmir', 'Ladakh', 'Chandigarh',
		'Puducherry', 'Andaman and Nicobar Islands', 'Dadra and Nagar Haveli and Daman and Diu', 'Lakshadweep'
	];

	async function submit(e: Event) {
		e.preventDefault();
		error = '';
		loading = true;
		try {
			await api.signup({
				company_name: companyName,
				full_name: fullName,
				email,
				phone,
				state: state_,
				password,
				accept_terms: accept
			});
			done = true;
		} catch (err: any) {
			error = err.message || 'Could not create your account';
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head><title>Start your free trial</title></svelte:head>

<div style="min-height: 100vh; display: flex; align-items: center; justify-content: center; padding: 1.5rem; background: #0b0f19;">
	<div class="card" style="width: 100%; max-width: 480px; padding: 2rem; background: rgba(17, 24, 39, 0.9); border: 1px solid rgba(255,255,255,0.08); border-radius: var(--radius-lg);">
		{#if done}
			<h1 style="font-size: 1.35rem; color: #fff;">Check your email</h1>
			<p style="color: var(--text-secondary); line-height: 1.5; margin-top: 0.75rem;">
				We sent a link to <strong>{email}</strong>. Confirm your address to open your workspace and start your 14-day free trial.
			</p>
			<a href="/login" style="font-size: 0.8125rem;">Back to sign in</a>
		{:else}
			<h1 style="font-size: 1.35rem; color: #fff; margin-bottom: 0.25rem;">Start your 14-day free trial</h1>
			<p style="color: var(--text-secondary); font-size: 0.8125rem; margin-bottom: 1.25rem;">GST invoicing and accounting for your business. No card needed.</p>
			{#if error}<div style="color: #fb7185; font-size: 0.8125rem; margin-bottom: 1rem;">{error}</div>{/if}
			<form onsubmit={submit}>
				<div class="form-group"><label class="form-label" for="co">Business name</label>
					<input id="co" class="form-input" bind:value={companyName} required minlength="2" maxlength="120" autocomplete="organization" /></div>
				<div class="form-group"><label class="form-label" for="fn">Your name</label>
					<input id="fn" class="form-input" bind:value={fullName} required maxlength="120" autocomplete="name" /></div>
				<div class="form-group"><label class="form-label" for="em">Work email</label>
					<input id="em" type="email" class="form-input" bind:value={email} required autocomplete="username" /></div>
				<div class="form-group"><label class="form-label" for="ph">Phone (optional)</label>
					<input id="ph" class="form-input" bind:value={phone} maxlength="20" autocomplete="tel" /></div>
				<div class="form-group"><label class="form-label" for="st">State (decides CGST/SGST vs IGST)</label>
					<select id="st" class="form-input" bind:value={state_} required>
						<option value="" disabled>Select state</option>
						{#each states as s}<option value={s}>{s}</option>{/each}
					</select></div>
				<div class="form-group"><label class="form-label" for="pw">Password (10+ characters, letter and number)</label>
					<input id="pw" type="password" class="form-input" bind:value={password} required minlength="10" autocomplete="new-password" /></div>
				<label style="display: flex; gap: 0.5rem; font-size: 0.8125rem; color: var(--text-secondary); margin: 0.5rem 0 1.25rem;">
					<input type="checkbox" bind:checked={accept} required />
					<span>I accept the <a href="/terms" target="_blank" rel="noopener">Terms of Service</a> and <a href="/privacy" target="_blank" rel="noopener">Privacy Policy</a>.</span>
				</label>
				<button class="btn btn-primary" style="width: 100%;" disabled={loading}>{loading ? 'Creating...' : 'Create my account'}</button>
			</form>
			<p style="margin-top: 1rem; text-align: center; font-size: 0.8125rem;"><a href="/login">Already have an account? Sign in</a></p>
		{/if}
	</div>
</div>
