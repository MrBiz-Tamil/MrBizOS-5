<script lang="ts">
	import { onMount } from 'svelte';
	import QRCode from 'qrcode';
	import { api } from '$lib/api';

	type Status = { email: string; email_verified: boolean; two_factor_enabled: boolean; recovery_codes_remaining: number };
	let status = $state<Status | null>(null);
	let notice = $state('');
	let error = $state('');

	// change password
	let curPw = $state('');
	let newPw = $state('');

	// 2FA setup
	let setupPw = $state('');
	let secret = $state('');
	let qrDataUrl = $state('');
	let enableCode = $state('');
	let recoveryCodes = $state<string[]>([]);

	// 2FA disable
	let disablePw = $state('');
	let disableCode = $state('');

	async function load() {
		try {
			status = await api.getSecurityStatus();
		} catch (e: any) {
			error = e.message;
		}
	}
	onMount(load);

	function reset() {
		notice = '';
		error = '';
	}

	async function changePassword(e: Event) {
		e.preventDefault();
		reset();
		try {
			const r = await api.changePassword(curPw, newPw);
			notice = r.message || 'Password changed.';
			curPw = newPw = '';
		} catch (err: any) {
			error = err.message;
		}
	}

	async function resend() {
		reset();
		try {
			notice = (await api.resendVerification()).message;
		} catch (err: any) {
			error = err.message;
		}
	}

	async function startSetup(e: Event) {
		e.preventDefault();
		reset();
		try {
			const r = await api.twoFactorSetup(setupPw);
			secret = r.secret;
			// The QR code is generated in the browser; the secret never goes to a third-party service.
			qrDataUrl = await QRCode.toDataURL(r.otpauth_uri, { width: 200, margin: 1 });
			setupPw = '';
		} catch (err: any) {
			error = err.message;
		}
	}

	async function enable(e: Event) {
		e.preventDefault();
		reset();
		try {
			const r = await api.twoFactorEnable(enableCode.trim());
			recoveryCodes = r.recovery_codes;
			secret = qrDataUrl = enableCode = '';
			await load();
		} catch (err: any) {
			error = err.message;
		}
	}

	async function disable(e: Event) {
		e.preventDefault();
		reset();
		try {
			notice = (await api.twoFactorDisable(disablePw, disableCode.trim())).message;
			disablePw = disableCode = '';
			recoveryCodes = [];
			await load();
		} catch (err: any) {
			error = err.message;
		}
	}
</script>

<svelte:head><title>Account Security</title></svelte:head>

<div style="max-width: 720px;">
	<h1 style="margin-bottom: 0.25rem;">Account Security</h1>
	<p style="color: var(--text-secondary); margin-bottom: 1.5rem;">Manage your password, email verification and two-factor authentication.</p>

	{#if notice}<div class="card" style="padding: 0.75rem 1rem; margin-bottom: 1rem; color: #34d399;">{notice}</div>{/if}
	{#if error}<div class="card" style="padding: 0.75rem 1rem; margin-bottom: 1rem; color: #fb7185;">{error}</div>{/if}

	{#if status}
		<div class="card" style="padding: 1.25rem; margin-bottom: 1.25rem;">
			<h3 style="margin-bottom: 0.5rem;">Email address</h3>
			<p>{status.email}
				{#if status.email_verified}<span style="color: #34d399;"> — verified</span>{:else}<span style="color: #fbbf24;"> — not verified</span>{/if}
			</p>
			{#if !status.email_verified}
				<button class="btn btn-secondary" style="margin-top: 0.75rem;" onclick={resend}>Send verification email</button>
			{/if}
		</div>

		<div class="card" style="padding: 1.25rem; margin-bottom: 1.25rem;">
			<h3 style="margin-bottom: 0.75rem;">Change password</h3>
			<form onsubmit={changePassword}>
				<div class="form-group"><label class="form-label" for="cur">Current password</label>
					<input id="cur" type="password" class="form-input" bind:value={curPw} required autocomplete="current-password" /></div>
				<div class="form-group"><label class="form-label" for="new">New password (10+ characters, letter and number)</label>
					<input id="new" type="password" class="form-input" bind:value={newPw} required minlength="10" autocomplete="new-password" /></div>
				<button class="btn btn-primary">Change password</button>
			</form>
		</div>

		<div class="card" style="padding: 1.25rem;">
			<h3 style="margin-bottom: 0.5rem;">Two-factor authentication</h3>
			{#if recoveryCodes.length}
				<p style="color: #fbbf24; margin-bottom: 0.5rem;">Two-factor authentication is on. Save these recovery codes now — each works once and they are not shown again.</p>
				<pre id="recovery-codes" style="background: rgba(0,0,0,0.3); padding: 1rem; border-radius: 8px; columns: 2;">{recoveryCodes.join('\n')}</pre>
				<button class="btn btn-secondary" style="margin-top: 0.75rem;" onclick={() => (recoveryCodes = [])}>I have saved them</button>
			{:else if status.two_factor_enabled}
				<p style="margin-bottom: 0.75rem;">Enabled. Recovery codes remaining: <strong>{status.recovery_codes_remaining}</strong></p>
				<form onsubmit={disable}>
					<div class="form-group"><label class="form-label" for="dpw">Password</label>
						<input id="dpw" type="password" class="form-input" bind:value={disablePw} required autocomplete="current-password" /></div>
					<div class="form-group"><label class="form-label" for="dcode">Authenticator or recovery code</label>
						<input id="dcode" class="form-input" bind:value={disableCode} required autocomplete="one-time-code" /></div>
					<button class="btn btn-secondary">Turn off two-factor</button>
				</form>
			{:else if secret}
				<p style="color: var(--text-secondary); margin-bottom: 0.75rem;">Scan this QR code with an authenticator app (Google Authenticator, Authy, 1Password…), then enter the 6-digit code.</p>
				<img src={qrDataUrl} alt="Authenticator QR code" width="200" height="200" style="background: #fff; border-radius: 8px;" />
				<p style="font-size: 0.8125rem; margin: 0.75rem 0;">Or enter this key manually: <code id="totp-secret">{secret}</code></p>
				<form onsubmit={enable}>
					<div class="form-group"><label class="form-label" for="ecode">6-digit code</label>
						<input id="ecode" class="form-input" bind:value={enableCode} required inputmode="numeric" autocomplete="one-time-code" /></div>
					<button class="btn btn-primary">Turn on two-factor</button>
				</form>
			{:else}
				<p style="color: var(--text-secondary); margin-bottom: 0.75rem;">Adds a one-time code from your phone to every sign-in. Recommended for owners and admins.</p>
				<form onsubmit={startSetup}>
					<div class="form-group"><label class="form-label" for="spw">Confirm your password</label>
						<input id="spw" type="password" class="form-input" bind:value={setupPw} required autocomplete="current-password" /></div>
					<button class="btn btn-primary">Set up two-factor</button>
				</form>
			{/if}
		</div>
	{/if}

	<div class="card" style="padding: 1.25rem; margin-top: 1.25rem;">
		<h3 style="margin-bottom: 0.5rem;">Export company data</h3>
		<p style="color: var(--text-secondary); margin-bottom: 0.75rem; font-size: 0.875rem;">
			Download all of your company's records (customers, invoices, payments, ledger and more) as a JSON file. Available to the owner; limited to 5 exports per hour.
		</p>
		<a class="btn btn-secondary" href="/api/company/export" download>Download export</a>
	</div>
</div>
