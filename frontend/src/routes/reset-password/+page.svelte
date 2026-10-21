<script lang="ts">
	import { api } from '$lib/api';
	import { page } from '$app/stores';
	let token = $derived($page.url.searchParams.get('token') || '');
	let password = $state('');
	let confirm = $state('');
	let done = $state(false);
	let loading = $state(false);
	let error = $state('');

	async function submit(e: Event) {
		e.preventDefault();
		error = '';
		if (password !== confirm) {
			error = 'Passwords do not match';
			return;
		}
		loading = true;
		try {
			await api.resetPassword(token, password);
			done = true;
		} catch (err: any) {
			error = err.message || 'Could not reset password';
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head><title>Choose a new password</title></svelte:head>

<div style="min-height: 100vh; display: flex; align-items: center; justify-content: center; padding: 1.5rem; background: #0b0f19;"><div class="card" style="width: 100%; max-width: 440px; padding: 2rem; background: rgba(17, 24, 39, 0.9); border: 1px solid rgba(255,255,255,0.08); border-radius: var(--radius-lg);">
	<h1 style="font-size: 1.35rem; color: #fff; margin-bottom: 0.5rem;">Choose a new password</h1>
	{#if done}
		<p style="color: var(--text-secondary); font-size: 0.875rem;">Your password was updated and all other sessions were signed out.</p>
		<a href="/login" class="btn btn-primary" style="display: block; text-align: center; margin-top: 1.25rem;">Sign in</a>
	{:else if !token}
		<p style="color: #fb7185; font-size: 0.875rem;">This link is missing its token. Request a new reset link.</p>
		<a href="/forgot-password" style="font-size: 0.8125rem;">Request a new link</a>
	{:else}
		<p style="color: var(--text-secondary); font-size: 0.8125rem; margin-bottom: 1.25rem;">At least 10 characters, including a letter and a number.</p>
		{#if error}<div style="color: #fb7185; font-size: 0.8125rem; margin-bottom: 1rem;">{error}</div>{/if}
		<form onsubmit={submit}>
			<div class="form-group">
				<label class="form-label" for="np">New password</label>
				<input id="np" type="password" class="form-input" bind:value={password} required minlength="10" autocomplete="new-password" />
			</div>
			<div class="form-group" style="margin-bottom: 1.25rem;">
				<label class="form-label" for="cp">Confirm password</label>
				<input id="cp" type="password" class="form-input" bind:value={confirm} required autocomplete="new-password" />
			</div>
			<button class="btn btn-primary" style="width: 100%;" disabled={loading}>{loading ? 'Saving...' : 'Update password'}</button>
		</form>
	{/if}
</div></div>
