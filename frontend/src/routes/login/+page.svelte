<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';
	import { page } from '$app/stores';

	let email = $state('');
	let password = $state('');
	let errorMessage = $state('');
	let loading = $state(false);
	let challengeToken = $state('');
	let twoFactorCode = $state('');

	// Remembered Tenant Brand State
	let savedTenantName = $state('');
	let savedTenantLegal = $state('');
	let savedTenantLogo = $state('');
	let isCustomTenantView = $derived(!!savedTenantName);

	onMount(() => {
		// Check URL query param first e.g. /login?tenant=Acme
		const queryTenant = $page.url.searchParams.get('tenant');
		if (queryTenant) {
			savedTenantName = queryTenant;
		} else {
			// Read remembered tenant brand from previous session
			savedTenantName = localStorage.getItem('tenant_brand_name') || '';
			savedTenantLegal = localStorage.getItem('tenant_brand_legal') || '';
			savedTenantLogo = localStorage.getItem('tenant_brand_logo') || '';
		}

		// Pre-fill remembered email or default demo
		const savedEmail = localStorage.getItem('tenant_user_email');
		if (savedEmail) {
			email = savedEmail;
		}
	});

	function getInitials(name: string) {
		if (!name) return 'SB';
		const parts = name.trim().split(/\s+/);
		if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
		return (parts[0][0] + parts[1][0]).toUpperCase();
	}

	async function handleSubmit(e: Event) {
		e.preventDefault();
		errorMessage = '';
		loading = true;
		try {
			const res = await api.login(email, password);
			if (res.requires_2fa && res.challenge_token) {
				challengeToken = res.challenge_token;
				return;
			}
			await finishLogin();
		} catch (err: any) {
			errorMessage = err.message || 'Authentication failed';
		} finally {
			loading = false;
		}
	}

	async function handleVerify(e: Event) {
		e.preventDefault();
		errorMessage = '';
		loading = true;
		try {
			await api.verifyTwoFactor(challengeToken, twoFactorCode.trim());
			await finishLogin();
		} catch (err: any) {
			errorMessage = err.message || 'Verification failed';
			if (/expired/i.test(errorMessage)) {
				challengeToken = '';
				twoFactorCode = '';
			}
		} finally {
			loading = false;
		}
	}

	async function finishLogin() {
			// Save user email
			localStorage.setItem('tenant_user_email', email);

			// Fetch company context and save branding for next login
			try {
				const me = await api.getMe();
				if (me.tenant) {
					localStorage.setItem('tenant_brand_name', me.tenant.name || '');
					localStorage.setItem('tenant_brand_legal', me.tenant.legal_name || '');
					localStorage.setItem('tenant_brand_logo', me.tenant.logo_url || '');
				}
			} catch (e) {}

		window.location.href = '/';
	}

	function resetTenantBranding() {
		localStorage.removeItem('tenant_brand_name');
		localStorage.removeItem('tenant_brand_legal');
		localStorage.removeItem('tenant_brand_logo');
		savedTenantName = '';
		savedTenantLegal = '';
		savedTenantLogo = '';
		email = '';
		password = '';
	}
</script>

<svelte:head>
	<title>{savedTenantName ? `${savedTenantName} - Sign In` : 'Sign In - SaaS Billing & Accounting'}</title>
</svelte:head>

<div style="min-height: 100vh; display: flex; align-items: center; justify-content: center; padding: 1.5rem; background: radial-gradient(circle at top right, rgba(99, 102, 241, 0.18), transparent 50%), radial-gradient(circle at bottom left, rgba(236, 72, 153, 0.12), transparent 50%), #0b0f19;">
	<div class="card animate-fade-in" style="width: 100%; max-width: 460px; padding: 2.5rem 2rem; background: rgba(17, 24, 39, 0.9); box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.6); border: 1px solid rgba(255, 255, 255, 0.08); border-radius: var(--radius-lg);">
		
		<!-- Personalized Brand Header -->
		<div style="text-align: center; margin-bottom: 2rem;">
			{#if savedTenantLogo}
				<img
					src={savedTenantLogo}
					alt={savedTenantName}
					style="height: 54px; max-width: 180px; object-fit: contain; margin-bottom: 1rem; border-radius: var(--radius-sm);"
				/>
			{:else}
				<div style="width: 56px; height: 56px; border-radius: 14px; background: {isCustomTenantView ? 'linear-gradient(135deg, #4f46e5 0%, #06b6d4 100%)' : 'var(--primary-gradient)'}; display: inline-flex; align-items: center; justify-content: center; font-family: var(--font-heading); font-weight: 800; color: white; font-size: 1.35rem; box-shadow: 0 6px 20px rgba(99, 102, 241, 0.45); margin-bottom: 1rem;">
					{getInitials(savedTenantName || 'SaaS Billing')}
				</div>
			{/if}

			{#if isCustomTenantView}
				<div style="display: inline-flex; align-items: center; gap: 0.4rem; background: rgba(56, 189, 248, 0.12); color: #38bdf8; border: 1px solid rgba(56, 189, 248, 0.3); padding: 0.2rem 0.65rem; border-radius: 9999px; font-size: 0.75rem; font-weight: 600; margin-bottom: 0.5rem;">
					🏢 {savedTenantName} Workspace
				</div>
				<h1 style="font-size: 1.5rem; margin-bottom: 0.35rem; color: #fff;">
					Welcome Back
				</h1>
				<p style="font-size: 0.8125rem; color: var(--text-secondary); line-height: 1.4;">
					Sign in to your private financial operations & invoicing portal
					{#if savedTenantLegal}
						<br /><span style="font-size: 0.75rem; color: var(--text-muted);">({savedTenantLegal})</span>
					{/if}
				</p>
			{:else}
				<h1 style="font-size: 1.5rem; margin-bottom: 0.35rem; color: #fff;">SaaS Billing & Accounting</h1>
				<p style="font-size: 0.8125rem; color: var(--text-secondary);">Enterprise Multi-Tenant Financial Workspace</p>
			{/if}
		</div>

		{#if errorMessage}
			<div style="background: rgba(244, 63, 94, 0.15); border: 1px solid rgba(244, 63, 94, 0.3); border-radius: var(--radius-md); padding: 0.75rem 1rem; color: #fb7185; font-size: 0.8125rem; margin-bottom: 1.25rem; display: flex; align-items: center; gap: 0.5rem;">
				<span>⚠️</span> {errorMessage}
			</div>
		{/if}

		{#if challengeToken}
		<form onsubmit={handleVerify}>
			<p style="font-size: 0.8125rem; color: var(--text-secondary); margin-bottom: 1rem;">
				Enter the 6-digit code from your authenticator app, or one of your recovery codes.
			</p>
			<div class="form-group" style="margin-bottom: 1.5rem;">
				<label class="form-label" for="otp">Verification code</label>
				<input id="otp" class="form-input" bind:value={twoFactorCode} required autocomplete="one-time-code" placeholder="123456" />
			</div>
			<button type="submit" class="btn btn-primary" style="width: 100%; padding: 0.75rem; font-weight: 600;" disabled={loading}>
				{loading ? 'Verifying...' : 'Verify and sign in →'}
			</button>
			<div style="margin-top: 1rem; text-align: center;">
				<button type="button" onclick={() => { challengeToken = ''; twoFactorCode = ''; errorMessage = ''; }} style="background: none; border: none; font-size: 0.75rem; color: var(--text-muted); cursor: pointer; text-decoration: underline;">Back to sign in</button>
			</div>
		</form>
		{:else}
		<form onsubmit={handleSubmit}>
			<div class="form-group">
				<label class="form-label" for="email">Work Email</label>
				<input
					id="email"
					type="email"
					class="form-input"
					bind:value={email}
					required
					placeholder="name@company.com"
					autocomplete="username"
				/>
			</div>

			<div class="form-group" style="margin-bottom: 1.5rem;">
				<label class="form-label" for="password">Password</label>
				<input
					id="password"
					type="password"
					class="form-input"
					bind:value={password}
					required
					placeholder="••••••••••••"
					autocomplete="current-password"
				/>
			</div>

			<button
				type="submit"
				class="btn btn-primary"
				style="width: 100%; padding: 0.75rem; font-size: 0.9375rem; font-weight: 600;"
				disabled={loading}
			>
				{#if loading}
					Verifying credentials...
				{:else if isCustomTenantView}
					Sign In to {savedTenantName} →
				{:else}
					Sign In to Workspace →
				{/if}
			</button>
		</form>
		<div style="margin-top: 1rem; text-align: center;">
			<a href="/signup" style="font-size: 0.8125rem; color: #818cf8;">New here? Start a free 14-day trial</a>
		</div>
		<div style="margin-top: 1rem; text-align: center;">
			<a href="/forgot-password" style="font-size: 0.8125rem; color: var(--text-secondary);">Forgot password?</a>
		</div>
		{/if}

		<p style="margin-top: 1.25rem; text-align: center; font-size: 0.75rem; color: var(--text-muted);">
			By signing in you agree to our <a href="/terms">Terms</a> and <a href="/privacy">Privacy Policy</a>.
		</p>

		<!-- Switch / Clear Tenant Switcher -->
		{#if isCustomTenantView}
			<div style="margin-top: 1.25rem; text-align: center;">
				<button
					type="button"
					onclick={resetTenantBranding}
					style="background: none; border: none; font-size: 0.75rem; color: var(--text-muted); cursor: pointer; text-decoration: underline;"
				>
					Not {savedTenantName}? Switch company or sign in as Super Admin
				</button>
			</div>
		{/if}

	</div>
</div>
