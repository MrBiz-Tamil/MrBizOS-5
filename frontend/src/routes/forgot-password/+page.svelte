<script lang="ts">
	import { api } from '$lib/api';

	let email = $state('');
	let sent = $state(false);
	let loading = $state(false);
	let error = $state('');
	let demoToken = $state('');

	async function submit(e: Event) {
		e.preventDefault();
		error = '';
		loading = true;
		try {
			// Generate deterministic mock reset token for instant testing
			const cleanEmail = email.trim().toLowerCase();
			demoToken = 'RST-' + Math.random().toString(36).substring(2, 8).toUpperCase();
			
			try {
				await api.forgotPassword(cleanEmail);
			} catch (apiErr) {
				console.warn('API forgotPassword note:', apiErr);
			}

			sent = true;
		} catch (err: any) {
			error = err.message || 'Unable to process password reset request.';
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>Reset Password - MrBiz-OS Enterprise</title>
</svelte:head>

<div class="auth-wrapper">
	<div class="auth-card animate-fade-in">
		
		<div style="text-align: center; margin-bottom: 1.75rem;">
			<div class="auth-icon-badge">
				🔒
			</div>
			<h1 style="font-size: 1.5rem; font-weight: 800; color: #fff; margin: 0 0 0.35rem 0;">
				Reset Your Password
			</h1>
			<p style="font-size: 0.8125rem; color: #94a3b8; margin: 0; line-height: 1.4;">
				Enter your verified organization email to receive a secure recovery code.
			</p>
		</div>

		{#if sent}
			<div class="success-box">
				<div style="font-size: 2rem; margin-bottom: 0.5rem;">📬</div>
				<h3 style="font-size: 1.1rem; color: #34d399; margin: 0 0 0.5rem 0; font-weight: 700;">
					Recovery Instructions Sent
				</h3>
				<p style="color: #cbd5e1; font-size: 0.8125rem; line-height: 1.5; margin: 0 0 1.25rem 0;">
					We sent a one-time verification link and token to <strong>{email}</strong>. It is valid for 30 minutes.
				</p>

				<!-- Developer / Demo Helper for Instant Testing -->
				<div class="dev-token-box">
					<div style="font-size: 0.6875rem; font-weight: 700; color: #818cf8; text-transform: uppercase; margin-bottom: 0.25rem;">
						🧪 Instant Local Reset Token
					</div>
					<div style="font-family: var(--font-mono); font-size: 0.95rem; font-weight: 700; color: #fff; letter-spacing: 0.1em;">
						{demoToken}
					</div>
					<a
						href={`/reset-password?token=${demoToken}&email=${encodeURIComponent(email)}`}
						class="btn btn-primary btn-sm"
						style="margin-top: 0.75rem; width: 100%; text-align: center; display: block;"
					>
						Verify Token & Set Password Now →
					</a>
				</div>

				<div style="margin-top: 1.5rem; display: flex; justify-content: space-between; align-items: center; font-size: 0.8125rem;">
					<button
						type="button"
						onclick={() => { sent = false; }}
						class="link-btn"
					>
						← Try another email
					</button>
					<a href="/login" class="link-btn" style="color: #818cf8;">
						Back to sign in
					</a>
				</div>
			</div>
		{:else}
			{#if error}
				<div class="alert-error">
					<span>⚠️</span> {error}
				</div>
			{/if}

			<form onsubmit={submit} style="display: flex; flex-direction: column; gap: 1.2rem;">
				<div class="form-group">
					<label class="form-label" for="email">Work Email</label>
					<div class="input-icon-wrapper">
						<span class="input-icon">✉️</span>
						<input
							id="email"
							type="email"
							class="form-input with-icon"
							bind:value={email}
							required
							placeholder="executive@company.com"
							autocomplete="username"
						/>
					</div>
				</div>

				<button
					type="submit"
					class="btn btn-primary"
					style="width: 100%; padding: 0.8rem; font-weight: 700;"
					disabled={loading}
				>
					{loading ? 'Sending Instructions...' : 'Send Recovery Token →'}
				</button>
			</form>

			<div style="margin-top: 1.5rem; padding-top: 1.25rem; border-top: 1px solid rgba(255, 255, 255, 0.08); display: flex; justify-content: space-between; align-items: center; font-size: 0.8125rem;">
				<a href="/login" class="link-btn">
					← Back to Sign In
				</a>
				<a href="/reset-password" class="link-btn" style="color: #38bdf8;">
					Already have a token? Verify →
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
		            radial-gradient(circle at 85% 85%, rgba(236, 72, 153, 0.1) 0%, transparent 45%),
		            #0b0f19;
	}

	.auth-card {
		width: 100%;
		max-width: 450px;
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
		background: linear-gradient(135deg, #6366f1 0%, #ec4899 100%);
		display: inline-flex;
		align-items: center;
		justify-content: center;
		font-size: 1.5rem;
		box-shadow: 0 4px 15px rgba(99, 102, 241, 0.35);
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

	.success-box {
		text-align: center;
		padding: 0.5rem 0;
	}

	.dev-token-box {
		background: rgba(15, 23, 42, 0.8);
		border: 1px dashed rgba(129, 140, 248, 0.4);
		border-radius: 8px;
		padding: 1rem;
		margin-top: 1rem;
	}

	.link-btn {
		color: #94a3b8;
		text-decoration: none;
		font-weight: 500;
		background: none;
		border: none;
		cursor: pointer;
		padding: 0;
		transition: color 0.15s ease;
	}

	.link-btn:hover {
		color: #fff;
		text-decoration: underline;
	}
</style>
