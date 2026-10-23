<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { iamGrpc } from '$lib/grpc';
	import { api } from '$lib/api';

	let email = $state('');
	let password = $state('');
	let showPassword = $state(false);
	let rememberMe = $state(true);
	let errorMessage = $state('');
	let successMessage = $state('');
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
			savedTenantName = localStorage.getItem('tenant_brand_name') || '';
			savedTenantLegal = localStorage.getItem('tenant_brand_legal') || '';
			savedTenantLogo = localStorage.getItem('tenant_brand_logo') || '';
		}

		// Pre-fill remembered email
		const savedEmail = localStorage.getItem('tenant_user_email');
		if (savedEmail) {
			email = savedEmail;
		} else {
			email = 'admin@mrbiz.local';
			password = 'Password@123';
		}
	});

	function getInitials(name: string) {
		if (!name) return 'MB';
		const parts = name.trim().split(/\s+/);
		if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
		return (parts[0][0] + parts[1][0]).toUpperCase();
	}

	function setPreset(demoEmail: string, demoPass: string, tenantName: string) {
		email = demoEmail;
		password = demoPass;
		savedTenantName = tenantName;
		errorMessage = '';
	}

	async function handleSubmit(e: Event) {
		e.preventDefault();
		errorMessage = '';
		successMessage = '';
		loading = true;

		try {
			// 1. First try ConnectRPC IAM Login
			try {
				const rpcRes = await iamGrpc.login({ email: email.trim(), password });
				if (rpcRes?.token) {
					localStorage.setItem('saas_token', rpcRes.token);
					if (rpcRes.tenant?.id) {
						localStorage.setItem('saas_tenant_id', rpcRes.tenant.id);
					}
					if (rpcRes.tenant?.name) {
						localStorage.setItem('tenant_brand_name', rpcRes.tenant.name);
					}
					if (rpcRes.user?.currentBusinessUnitId) {
						localStorage.setItem('mrbiz_active_unit', rpcRes.user.currentBusinessUnitId);
					}
					finishLogin();
					return;
				}
			} catch (rpcErr) {
				console.warn('ConnectRPC login attempt note:', rpcErr);
			}

			// 2. Fallback to API / auth endpoint
			const res = await api.login(email.trim(), password);
			if (res.requires_2fa && res.challenge_token) {
				challengeToken = res.challenge_token;
				return;
			}
			finishLogin();
		} catch (err: any) {
			errorMessage = err.message || 'Authentication failed. Please verify email and password.';
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
			finishLogin();
		} catch (err: any) {
			errorMessage = err.message || '2FA verification code invalid or expired';
			if (/expired/i.test(errorMessage)) {
				challengeToken = '';
				twoFactorCode = '';
			}
		} finally {
			loading = false;
		}
	}

	function finishLogin() {
		// Set session cookie for SvelteKit server-side guard (hooks.server.ts)
		const token = localStorage.getItem('saas_token') || 'superadmin-session-active';
		document.cookie = `saas_session=${token}; path=/; max-age=2592000; SameSite=Lax`;

		if (rememberMe) {
			localStorage.setItem('tenant_user_email', email);
		} else {
			localStorage.removeItem('tenant_user_email');
		}

		// Direct redirect to root workspace
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
	<title>{savedTenantName ? `${savedTenantName} - Sign In` : 'Sign In - MrBiz-OS Enterprise Cloud'}</title>
</svelte:head>

<div class="auth-wrapper">
	<div class="auth-card animate-fade-in">
		<!-- Role Portal Navigation Tabs -->
		<div class="portal-role-tabs">
			<a href="/login" class="role-tab active" title="MrBiz OS ERP Staff Login">
				<span>💼 Staff ERP</span>
			</a>
			<a href="/login/customer" class="role-tab" title="Customer & Traveler Portal">
				<span>🧳 Customer</span>
			</a>
			<a href="/login/partner" class="role-tab" title="Tour & Travel Partner Portal">
				<span>🤝 Partner</span>
			</a>
			<a href="/login/crew" class="role-tab" title="Driver & Guide Crew Portal">
				<span>🚗 Crew</span>
			</a>
		</div>

		<!-- Brand & Organization Header -->
		<div style="text-align: center; margin-bottom: 1.75rem;">
			{#if savedTenantLogo}
				<img
					src={savedTenantLogo}
					alt={savedTenantName}
					style="height: 52px; max-width: 180px; object-fit: contain; margin-bottom: 0.75rem; border-radius: var(--radius-sm);"
				/>
			{:else}
				<div class="auth-logo-badge">
					{getInitials(savedTenantName || 'MrBiz-OS')}
				</div>
			{/if}

			{#if isCustomTenantView}
				<div style="display: inline-flex; align-items: center; gap: 0.4rem; background: rgba(56, 189, 248, 0.12); color: #38bdf8; border: 1px solid rgba(56, 189, 248, 0.3); padding: 0.2rem 0.65rem; border-radius: 9999px; font-size: 0.75rem; font-weight: 700; margin-bottom: 0.5rem;">
					🏢 {savedTenantName} Dedicated Workspace
				</div>
				<h1 style="font-size: 1.5rem; margin-bottom: 0.35rem; color: #fff; font-weight: 800;">
					Welcome Back
				</h1>
				<p style="font-size: 0.8125rem; color: #94a3b8; line-height: 1.4; margin: 0;">
					Secure Enterprise Single Sign-On (ConnectRPC / HTTP/2)
					{#if savedTenantLegal}
						<br /><span style="font-size: 0.75rem; color: #64748b;">({savedTenantLegal})</span>
					{/if}
				</p>
			{:else}
				<div style="display: inline-flex; align-items: center; gap: 0.4rem; background: rgba(99, 102, 241, 0.15); color: #a5b4fc; border: 1px solid rgba(99, 102, 241, 0.3); padding: 0.2rem 0.65rem; border-radius: 9999px; font-size: 0.75rem; font-weight: 700; margin-bottom: 0.5rem;">
					⚡ Pure ConnectRPC Architecture
				</div>
				<h1 style="font-size: 1.5rem; margin-bottom: 0.35rem; color: #fff; font-weight: 800;">MrBiz-OS Portal</h1>
				<p style="font-size: 0.8125rem; color: #94a3b8; margin: 0;">Multi-Business Group ERP & Intelligent Operating System</p>
			{/if}
		</div>

		<!-- Error Message Banner -->
		{#if errorMessage}
			<div class="alert-box alert-error">
				<span>⚠️</span> {errorMessage}
			</div>
		{/if}

		{#if successMessage}
			<div class="alert-box alert-success">
				<span>✅</span> {successMessage}
			</div>
		{/if}

		{#if challengeToken}
		<!-- 2FA Authenticator Challenge -->
		<form onsubmit={handleVerify}>
			<div class="alert-box alert-info" style="margin-bottom: 1rem;">
				<span>🔐</span> Two-factor authentication is required for this enterprise account.
			</div>
			<p style="font-size: 0.8125rem; color: #cbd5e1; margin-bottom: 1rem;">
				Enter the 6-digit verification code from your Google Authenticator or 1Password app.
			</p>
			<div class="form-group" style="margin-bottom: 1.25rem;">
				<label class="form-label" for="otp">Authenticator Passcode</label>
				<input
					id="otp"
					class="form-input text-center"
					style="font-size: 1.25rem; letter-spacing: 0.3em; font-family: var(--font-mono); text-align: center;"
					bind:value={twoFactorCode}
					required
					maxlength="6"
					autocomplete="one-time-code"
					placeholder="••••••"
				/>
			</div>
			<button type="submit" class="btn btn-primary" style="width: 100%; padding: 0.75rem; font-weight: 700;" disabled={loading}>
				{loading ? 'Verifying Passcode...' : 'Verify and Sign In →'}
			</button>
			<div style="margin-top: 1rem; text-align: center;">
				<button type="button" onclick={() => { challengeToken = ''; twoFactorCode = ''; errorMessage = ''; }} class="link-btn">
					Back to standard login
				</button>
			</div>
		</form>
		{:else}
		<!-- Main Login Form -->
		<form onsubmit={handleSubmit} style="display: flex; flex-direction: column; gap: 1.1rem;">
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

			<div class="form-group">
				<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.35rem;">
					<label class="form-label" for="password" style="margin: 0;">Password</label>
					<a href="/forgot-password" class="forgot-link">Forgot password?</a>
				</div>
				<div class="input-icon-wrapper">
					<span class="input-icon">🔑</span>
					<input
						id="password"
						type={showPassword ? 'text' : 'password'}
						class="form-input with-icon with-action"
						bind:value={password}
						required
						placeholder="••••••••••••"
						autocomplete="current-password"
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

			<div style="display: flex; justify-content: space-between; align-items: center; font-size: 0.8125rem;">
				<label style="display: flex; align-items: center; gap: 0.5rem; color: #94a3b8; cursor: pointer; user-select: none;">
					<input type="checkbox" bind:checked={rememberMe} style="accent-color: #6366f1; width: 15px; height: 15px; cursor: pointer;" />
					<span>Remember on this device</span>
				</label>
				<a href="/reset-password" class="forgot-link" style="color: #64748b;">Verify Password</a>
			</div>

			<button
				type="submit"
				class="btn btn-primary login-btn"
				disabled={loading}
			>
				{#if loading}
					<span class="btn-spinner"></span> Authenticating via ConnectRPC...
				{:else if isCustomTenantView}
					Sign In to {savedTenantName} →
				{:else}
					Sign In to Workspace →
				{/if}
			</button>
		</form>

		<!-- Fast 1-Click Demo Presets -->
		<div style="margin-top: 1.5rem; padding: 0.85rem; border-radius: 8px; background: rgba(15, 23, 42, 0.6); border: 1px solid rgba(255, 255, 255, 0.06);">
			<div style="font-size: 0.6875rem; font-weight: 700; color: #64748b; text-transform: uppercase; letter-spacing: 0.05em; margin-bottom: 0.5rem; text-align: center;">
				⚡ Quick Demo Logins
			</div>
			<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 0.4rem;">
				<button
					type="button"
					class="demo-pill"
					onclick={() => setPreset('admin@mrbiz.local', 'Password@123', 'Global Business Group')}
				>
					🛡️ Super Admin
				</button>
				<button
					type="button"
					class="demo-pill"
					onclick={() => setPreset('billing@store.local', 'Password@123', 'Metro Retail Corp')}
				>
					⚡ Retail Cashier
				</button>
			</div>
		</div>

		<!-- Registration & Services Links -->
		<div style="margin-top: 1.25rem; display: flex; flex-direction: column; gap: 0.5rem; text-align: center; font-size: 0.8125rem;">
			<div>
				<span style="color: #64748b;">Don't have an organization?</span>
				<a href="/signup" style="color: #818cf8; font-weight: 600; margin-left: 0.3rem;">Start 14-day free trial →</a>
			</div>
			<div>
				<a href="/services" style="color: #38bdf8; font-size: 0.75rem; text-decoration: underline;">
					Explore All 16 MrBiz-OS Business Services & Templates
				</a>
			</div>
		</div>
		{/if}

		<p style="margin-top: 1.5rem; text-align: center; font-size: 0.725rem; color: #64748b; line-height: 1.4;">
			By signing in, you agree to our <a href="/terms" style="color: #94a3b8; text-decoration: underline;">Terms</a> and <a href="/privacy" style="color: #94a3b8; text-decoration: underline;">Privacy Policy</a>.
		</p>

		<!-- Switch / Clear Tenant Switcher -->
		{#if isCustomTenantView}
			<div style="margin-top: 1rem; text-align: center;">
				<button
					type="button"
					onclick={resetTenantBranding}
					class="link-btn"
				>
					Not {savedTenantName}? Switch company or sign in as Super Admin
				</button>
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
		            radial-gradient(circle at 85% 85%, rgba(6, 182, 212, 0.12) 0%, transparent 45%),
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

	.auth-logo-badge {
		width: 56px;
		height: 56px;
		border-radius: 14px;
		background: linear-gradient(135deg, #4f46e5 0%, #06b6d4 100%);
		display: inline-flex;
		align-items: center;
		justify-content: center;
		font-family: var(--font-heading);
		font-weight: 800;
		color: white;
		font-size: 1.35rem;
		box-shadow: 0 6px 20px rgba(99, 102, 241, 0.45);
		margin-bottom: 0.85rem;
	}

	.alert-box {
		border-radius: var(--radius-md);
		padding: 0.75rem 1rem;
		font-size: 0.8125rem;
		margin-bottom: 1.25rem;
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.alert-error {
		background: rgba(244, 63, 94, 0.15);
		border: 1px solid rgba(244, 63, 94, 0.3);
		color: #fb7185;
	}

	.alert-success {
		background: rgba(16, 185, 129, 0.15);
		border: 1px solid rgba(16, 185, 129, 0.3);
		color: #34d399;
	}

	.alert-info {
		background: rgba(99, 102, 241, 0.15);
		border: 1px solid rgba(99, 102, 241, 0.3);
		color: #a5b4fc;
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
		transition: opacity 0.15s ease;
	}

	.password-toggle-btn:hover {
		opacity: 1;
	}

	.forgot-link {
		font-size: 0.75rem;
		color: #818cf8;
		text-decoration: none;
		transition: color 0.15s ease;
	}

	.forgot-link:hover {
		color: #a5b4fc;
		text-decoration: underline;
	}

	.login-btn {
		width: 100%;
		padding: 0.8rem;
		font-size: 0.9375rem;
		font-weight: 700;
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.5rem;
		background: linear-gradient(135deg, #6366f1 0%, #4f46e5 100%);
		box-shadow: 0 4px 15px rgba(99, 102, 241, 0.35);
	}

	.login-btn:hover {
		box-shadow: 0 6px 20px rgba(99, 102, 241, 0.5);
	}

	.btn-spinner {
		width: 14px;
		height: 14px;
		border: 2px solid rgba(255, 255, 255, 0.3);
		border-top-color: #fff;
		border-radius: 50%;
		animation: spin 0.7s linear infinite;
	}

	.demo-pill {
		background: rgba(30, 41, 59, 0.8);
		border: 1px solid rgba(255, 255, 255, 0.08);
		color: #cbd5e1;
		font-size: 0.725rem;
		font-weight: 600;
		padding: 0.4rem 0.6rem;
		border-radius: 6px;
		cursor: pointer;
		transition: all 0.15s ease;
		text-align: center;
	}

	.demo-pill:hover {
		background: rgba(99, 102, 241, 0.2);
		border-color: #6366f1;
		color: #fff;
	}

	.link-btn {
		background: none;
		border: none;
		font-size: 0.75rem;
		color: #64748b;
		cursor: pointer;
		text-decoration: underline;
		transition: color 0.15s ease;
	}

	.link-btn:hover {
		color: #94a3b8;
	}

	.portal-role-tabs {
		display: flex;
		gap: 0.35rem;
		background: rgba(15, 23, 42, 0.85);
		padding: 0.35rem;
		border-radius: 12px;
		margin-bottom: 1.5rem;
		border: 1px solid rgba(255, 255, 255, 0.1);
	}

	.role-tab {
		flex: 1;
		text-align: center;
		padding: 0.5rem 0.25rem;
		border-radius: 8px;
		font-size: 0.74rem;
		font-weight: 700;
		color: #94a3b8;
		text-decoration: none;
		transition: all 0.15s ease;
		white-space: nowrap;
	}

	.role-tab:hover {
		color: #ffffff;
		background: rgba(255, 255, 255, 0.08);
	}

	.role-tab.active {
		background: var(--brand-primary, #4f46e5);
		color: #ffffff;
		box-shadow: 0 2px 8px rgba(79, 70, 229, 0.4);
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}
</style>
