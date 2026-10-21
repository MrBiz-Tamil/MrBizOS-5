<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';
	import { page } from '$app/stores';
	let phase = $state<'working' | 'ok' | 'error'>('working');
	let message = $state('');

	onMount(async () => {
		const token = $page.url.searchParams.get('token') || '';
		try {
			const r = await api.verifyEmail(token);
			message = r.message;
			phase = 'ok';
		} catch (err: any) {
			message = err.message || 'This verification link is invalid or has expired.';
			phase = 'error';
		}
	});
</script>

<svelte:head><title>Verify email</title></svelte:head>

<div style="min-height: 100vh; display: flex; align-items: center; justify-content: center; padding: 1.5rem; background: #0b0f19;"><div class="card" style="width: 100%; max-width: 440px; padding: 2rem; background: rgba(17, 24, 39, 0.9); border: 1px solid rgba(255,255,255,0.08); border-radius: var(--radius-lg);">
	<h1 style="font-size: 1.35rem; color: #fff; margin-bottom: 0.75rem;">Email verification</h1>
	{#if phase === 'working'}
		<p style="color: var(--text-secondary);">Verifying...</p>
	{:else if phase === 'ok'}
		<p style="color: #34d399;">{message}</p>
		<a href="/login" class="btn btn-primary" style="display: block; text-align: center; margin-top: 1.25rem;">Continue to sign in</a>
	{:else}
		<p style="color: #fb7185;">{message}</p>
		<p style="color: var(--text-secondary); font-size: 0.8125rem;">Sign in and use Account Security to send a new verification email.</p>
		<a href="/login" style="font-size: 0.8125rem;">Go to sign in</a>
	{/if}
</div></div>
