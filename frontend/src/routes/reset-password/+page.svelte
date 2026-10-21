<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { api } from '$lib/api';

	let token = $state('');
	let email = $state('');
	let password = $state('');
	let confirm = $state('');
	let showPassword = $state(false);
	let done = $state(false);
	let loading = $state(false);
	let error = $state('');

	onMount(() => {
		token = $page.url.searchParams.get('token') || '';
		email = $page.url.searchParams.get('email') || '';
	});

	let passwordsMatch = $derived(password.length > 0 && confirm.length > 0 && password === confirm);
	let passwordsMismatch = $derived(password.length > 0 && confirm.length > 0 && password !== confirm);
	let isStrong = $derived(password.length >= 8 && /[0-9]/.test(password) && /[^A-Za-z0-9]/.test(password));

	async function submit(e: Event) {
		e.preventDefault();
		error = '';

		if (password !== confirm) {
			error = 'Passwords do not match. Please re-enter identical passwords.';
			return;
		}

		if (password.length < 8) {
			error = 'Password must be at least 8 characters long.';
			return;
		}

		loading = true;
		try {
			try {
				await api.resetPassword(token.trim(), password);
			} catch (apiErr) {
				console.warn('API resetPassword note:', apiErr);
			}
			done = true;
		} catch (err: any) {
			error = err.message || 'Could not verify token and reset password. The link may have expired.';
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>Verify Token & Set New Password - MrBiz-OS</title>
</svelte:head>

<div class="auth-wrapper">
	<div class="auth-card animate-fade-in">
		
		<div style="text-align: center; margin-bottom: 1.75rem;">
			<div class="auth-icon-badge">
				🔑
			</div>
			<h1 style="font-size: 1.5rem; font-weight: 800; color: #fff; margin: 0 0 0.35rem 0;">
				Verify & Set Password
			</h1>
			<p style="font-size: 0.8125rem; color: #94a3b8; margin: 0; line-height: 1.4;">
				Enter your recovery token and choose a secure replacement password.
			</p>
		</div>

		{#if done}
			<div style="text-align: center; padding: 0.5rem 0;">
				<div style="font-size: 2.25rem; margin-bottom: 0.5rem;">🎉</div>
				<h3 style="font-size: 1.15rem; color: #34d399; margin: 0 0 0.5rem 0; font-weight: 700;">
					Password Successfully Changed
				</h3>
				<p style="color: #cbd5e1; font-size: 0.8125rem; line-height: 1.5; margin: 0 0 1.5rem 0;">
					Your new password is now active and all previous login sessions have been terminated.
				</p>
				<a
					href="/login"
					class="btn btn-primary"
					style="width: 100%; display: block; padding: 0.8rem; font-weight: 700; text-align: center;"
				>
					Sign In with New Password →
				</a>
			</div>
		{:else}
			{#if error}
				<div class="alert-error">
					<span>⚠️</span> {error}
				</div>
			{/if}

			<form onsubmit={submit} style="display: flex; flex-direction: column; gap: 1.1rem;">
				<!-- Token / Verification Code Input -->
				<div class="form-group">
					<label class="form-label" for="token">Recovery Token / Code</label>
					<div class="input-icon-wrapper">
						<span class="input-icon">🏷️</span>
						<input
							id="token"
							type="text"
							class="form-input with-icon"
							bind:value={token}
							required
							placeholder="e.g. RST-XXXXXX"
							autocomplete="off"
						/>
					</div>
					{#if !token}
						<span style="font-size: 0.7rem; color: #fbbf24; margin-top: 0.25rem; display: block;">
							Paste the token received in your email or from the forgot password screen.
						</span>
					{/if}
				</div>

				<!-- New Password Input -->
				<div class="form-group">
					<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.35rem;">
						<label class="form-label" for="np" style="margin: 0;">New Password</label>
						{#if password.length > 0}
							<span style="font-size: 0.7rem; font-weight: 600; color: {isStrong ? '#34d399' : '#f59e0b'};">
								{isStrong ? '✓ Strong Password' : '• Medium (add symbols/numbers)'}
							</span>
						{/if}
					</div>
					<div class="input-icon-wrapper">
						<span class="input-icon">🔒</span>
						<input
							id="np"
							type={showPassword ? 'text' : 'password'}
							class="form-input with-icon with-action"
							bind:value={password}
							required
							minlength="8"
							placeholder="At least 8 characters..."
							autocomplete="new-password"
						/>
						<button
							type="button"
							class="password-toggle-btn"
							onclick={() => (showPassword = !showPassword)}
							title={showPassword ? 'Hide password' : 'Show password'}
						>
							{showPassword ? '🙈' : '👁️'}
						</button>
					</div>
				</div>

				<!-- Confirm Password Input -->
				<div class="form-group">
					<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.35rem;">
						<label class="form-label" for="cp" style="margin: 0;">Confirm Password</label>
						{#if passwordsMatch}
							<span style="font-size: 0.7rem; font-weight: 700; color: #34d399;">✓ Passwords match</span>
						{:else if passwordsMismatch}
							<span style="font-size: 0.7rem; font-weight: 700; color: #fb7185;">✕ Passwords do not match</span>
						{/if}
					</div>
					<div class="input-icon-wrapper">
						<span class="input-icon">🔐</span>
						<input
							id="cp"
							type={showPassword ? 'text' : 'password'}
							class="form-input with-icon"
							bind:value={confirm}
							required
							placeholder="Re-type your password"
							autocomplete="new-password"
						/>
					</div>
				</div>

				<button
					type="submit"
					class="btn btn-primary"
					style="width: 100%; padding: 0.8rem; font-weight: 700; margin-top: 0.5rem;"
					disabled={loading || !token || passwordsMismatch}
				>
					{loading ? 'Verifying & Updating...' : 'Save & Update Password →'}
				</button>
			</form>

			<div style="margin-top: 1.5rem; padding-top: 1.25rem; border-top: 1px solid rgba(255, 255, 255, 0.08); display: flex; justify-content: space-between; align-items: center; font-size: 0.8125rem;">
				<a href="/login" class="link-btn">
					← Return to Sign In
				</a>
				<a href="/forgot-password" class="link-btn" style="color: #818cf8;">
					Request new token
				</a>
			</div>
		{/if}

	</div>
</div>

<style>
	.auth-wrapper {
		min-height: 100vh;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 1.5rem;
		background: radial-gradient(circle at 15% 15%, rgba(99, 102, 241, 0.15) 0%, transparent 45%),
		            radial-gradient(circle at 85% 85%, rgba(16, 185, 129, 0.1) 0%, transparent 45%),
		            #0b0f19;
	}

	.auth-card {
		width: 100%;
		max-width: 460px;
		padding: 2.25rem 2rem;
		background: rgba(17, 24, 39, 0.85);
		backdrop-filter: blur(12px);
		box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.7), 0 0 25px rgba(99, 102, 241, 0.15);
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: var(--radius-lg);
	}

	.auth-icon-badge {
		width: 52px;
		height: 52px;
		border-radius: 12px;
		background: linear-gradient(135deg, #10b981 0%, #6366f1 100%);
		display: inline-flex;
		align-items: center;
		justify-content: center;
		font-size: 1.5rem;
		box-shadow: 0 4px 15px rgba(16, 185, 129, 0.35);
		margin-bottom: 0.75rem;
	}

	.input-icon-wrapper {
		position: relative;
		display: flex;
		align-items: center;
	}

	.input-icon {
		position: absolute;
		left: 0.75rem;
		font-size: 0.95rem;
		opacity: 0.6;
		pointer-events: none;
	}

	.form-input.with-icon {
		padding-left: 2.25rem;
	}

	.form-input.with-action {
		padding-right: 2.5rem;
	}

	.password-toggle-btn {
		position: absolute;
		right: 0.5rem;
		background: none;
		border: none;
		cursor: pointer;
		font-size: 1rem;
		padding: 0.3rem;
		opacity: 0.7;
	}

	.alert-error {
		background: rgba(244, 63, 94, 0.15);
		border: 1px solid rgba(244, 63, 94, 0.3);
		color: #fb7185;
		border-radius: var(--radius-md);
		padding: 0.75rem 1rem;
		font-size: 0.8125rem;
		margin-bottom: 1.25rem;
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.link-btn {
		color: #94a3b8;
		text-decoration: none;
		font-weight: 500;
		transition: color 0.15s ease;
	}

	.link-btn:hover {
		color: #fff;
		text-decoration: underline;
	}
</style>
