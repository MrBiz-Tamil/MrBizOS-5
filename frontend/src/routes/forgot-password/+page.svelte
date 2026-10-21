<script lang="ts">
	import { api } from '$lib/api';
	let email = $state('');
	let sent = $state(false);
	let loading = $state(false);
	let error = $state('');

	async function submit(e: Event) {
		e.preventDefault();
		error = '';
		loading = true;
		try {
			await api.forgotPassword(email.trim());
			sent = true;
		} catch (err: any) {
			error = err.message || 'Something went wrong';
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head><title>Forgot password</title></svelte:head>

<div style="min-height: 100vh; display: flex; align-items: center; justify-content: center; padding: 1.5rem; background: #0b0f19;"><div class="card" style="width: 100%; max-width: 440px; padding: 2rem; background: rgba(17, 24, 39, 0.9); border: 1px solid rgba(255,255,255,0.08); border-radius: var(--radius-lg);">
	<h1 style="font-size: 1.35rem; color: #fff; margin-bottom: 0.5rem;">Reset your password</h1>
	{#if sent}
		<p style="color: var(--text-secondary); font-size: 0.875rem; line-height: 1.5;">
			If that email address belongs to an account, we have sent a reset link. It is valid for 30 minutes and can be used once.
		</p>
	{:else}
		<p style="color: var(--text-secondary); font-size: 0.8125rem; margin-bottom: 1.25rem;">Enter your work email and we will send you a reset link.</p>
		{#if error}<div style="color: #fb7185; font-size: 0.8125rem; margin-bottom: 1rem;">{error}</div>{/if}
		<form onsubmit={submit}>
			<div class="form-group" style="margin-bottom: 1.25rem;">
				<label class="form-label" for="email">Work Email</label>
				<input id="email" type="email" class="form-input" bind:value={email} required autocomplete="username" />
			</div>
			<button class="btn btn-primary" style="width: 100%;" disabled={loading}>{loading ? 'Sending...' : 'Send reset link'}</button>
		</form>
	{/if}
	<div style="margin-top: 1.25rem; text-align: center;"><a href="/login" style="font-size: 0.8125rem; color: var(--text-secondary);">Back to sign in</a></div>
</div></div>
