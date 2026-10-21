<script lang="ts">
	import { onMount } from 'svelte';
	import { api, type User } from '$lib/api';

	let users = $state<User[]>([]);
	let loading = $state(true);
	let error = $state('');
	let successMsg = $state('');

	// Create user modal state
	let showCreateModal = $state(false);
	let newEmail = $state('');
	let newFullName = $state('');
	let newRole = $state('BILLING');
	let newPassword = $state('');
	let isCreating = $state(false);

	const availableRoles = [
		{ value: 'OWNER', label: 'Owner (Full Access)', color: 'bg-amber-500/10 text-amber-400 border-amber-500/20' },
		{ value: 'ADMIN', label: 'Admin (Manage Business)', color: 'bg-indigo-500/10 text-indigo-400 border-indigo-500/20' },
		{ value: 'ACCOUNTANT', label: 'Accountant (Books & Audit)', color: 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20' },
		{ value: 'BILLING', label: 'Billing / Cashier', color: 'bg-sky-500/10 text-sky-400 border-sky-500/20' },
		{ value: 'SALES', label: 'Sales Clerk', color: 'bg-purple-500/10 text-purple-400 border-purple-500/20' },
		{ value: 'INVENTORY', label: 'Inventory Manager', color: 'bg-orange-500/10 text-orange-400 border-orange-500/20' },
		{ value: 'VIEWER', label: 'Viewer (Read Only)', color: 'bg-zinc-500/10 text-zinc-400 border-zinc-500/20' }
	];

	function getRoleBadge(role: string) {
		const r = availableRoles.find((item) => item.value === role);
		return r ? r.color : 'bg-zinc-800 text-zinc-300 border-zinc-700';
	}

	function generateRandomPassword() {
		const chars = 'ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789!@#$%';
		let res = '';
		for (let i = 0; i < 12; i++) {
			res += chars.charAt(Math.floor(Math.random() * chars.length));
		}
		newPassword = res;
	}

	async function loadUsers() {
		loading = true;
		error = '';
		try {
			users = await api.listUsers();
		} catch (err: any) {
			error = err.message || 'Failed to load team members';
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		loadUsers();
	});

	async function handleCreateUser(e: Event) {
		e.preventDefault();
		if (!newEmail || !newFullName || !newPassword) {
			error = 'Please fill out all required fields';
			return;
		}
		isCreating = true;
		error = '';
		successMsg = '';
		try {
			await api.createUser({
				email: newEmail,
				full_name: newFullName,
				role: newRole,
				password: newPassword
			});
			successMsg = `Staff member ${newFullName} added successfully!`;
			showCreateModal = false;
			newEmail = '';
			newFullName = '';
			newPassword = '';
			newRole = 'BILLING';
			await loadUsers();
		} catch (err: any) {
			error = err.message || 'Failed to create user';
		} finally {
			isCreating = false;
		}
	}

	async function handleToggleActive(u: User) {
		error = '';
		successMsg = '';
		try {
			const updated = await api.updateUser(u.id, { is_active: !u.is_active });
			users = users.map((item) => (item.id === u.id ? updated : item));
			successMsg = `User ${u.full_name} status updated.`;
		} catch (err: any) {
			error = err.message || 'Failed to update user status';
		}
	}

	async function handleRoleChange(u: User, newRole: string) {
		error = '';
		successMsg = '';
		try {
			const updated = await api.updateUser(u.id, { role: newRole });
			users = users.map((item) => (item.id === u.id ? updated : item));
			successMsg = `Role for ${u.full_name} updated to ${newRole}.`;
		} catch (err: any) {
			error = err.message || 'Failed to update user role';
		}
	}

	async function handleDeleteUser(u: User) {
		if (!confirm(`Are you sure you want to permanently remove staff member ${u.full_name}?`)) {
			return;
		}
		error = '';
		successMsg = '';
		try {
			await api.deleteUser(u.id);
			users = users.filter((item) => item.id !== u.id);
			successMsg = `Staff member ${u.full_name} removed.`;
		} catch (err: any) {
			error = err.message || 'Failed to remove user';
		}
	}
</script>

<svelte:head>
	<title>Team & Staff Management - SaaS Billing</title>
</svelte:head>

<div class="space-y-6 max-w-7xl mx-auto pb-12">
	<!-- Page Header -->
	<div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 border-b border-zinc-800/80 pb-5">
		<div>
			<h1 class="text-2xl font-bold tracking-tight text-white flex items-center gap-2.5">
				<span>👥</span> Team & Staff Management
			</h1>
			<p class="text-xs text-zinc-400 mt-1">
				Manage employee roles, access permissions, cashiers, accountants, and store staff.
			</p>
		</div>
		<div class="flex items-center gap-3">
			<a
				href="/settings"
				class="px-3.5 py-1.5 text-xs font-medium rounded-lg border border-zinc-800 bg-zinc-900 text-zinc-300 hover:bg-zinc-800 transition-colors"
			>
				← Back to Settings
			</a>
			<button
				onclick={() => {
					generateRandomPassword();
					showCreateModal = true;
				}}
				class="px-4 py-1.5 text-xs font-semibold rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white shadow-lg shadow-indigo-600/20 transition-all flex items-center gap-1.5"
			>
				<span>+</span> Add Staff Member
			</button>
		</div>
	</div>

	<!-- Alert Messages -->
	{#if error}
		<div class="p-3.5 rounded-xl border border-red-500/20 bg-red-500/10 text-red-400 text-xs flex items-center justify-between">
			<span>⚠️ {error}</span>
			<button onclick={() => (error = '')} class="text-red-400/80 hover:text-red-300">✕</button>
		</div>
	{/if}

	{#if successMsg}
		<div class="p-3.5 rounded-xl border border-emerald-500/20 bg-emerald-500/10 text-emerald-400 text-xs flex items-center justify-between">
			<span>✓ {successMsg}</span>
			<button onclick={() => (successMsg = '')} class="text-emerald-400/80 hover:text-emerald-300">✕</button>
		</div>
	{/if}

	<!-- Users Table -->
	<div class="rounded-xl border border-zinc-800/80 bg-zinc-900/30 overflow-hidden shadow-xl backdrop-blur-sm">
		<div class="p-4 border-b border-zinc-800/80 flex items-center justify-between">
			<div>
				<h2 class="text-sm font-semibold text-white">Active Staff Directory</h2>
				<p class="text-[11px] text-zinc-400">Total staff accounts: {users.length}</p>
			</div>
			<button
				onclick={loadUsers}
				class="text-xs text-zinc-400 hover:text-zinc-200 px-2 py-1 rounded bg-zinc-800/50"
			>
				↻ Refresh
			</button>
		</div>

		{#if loading}
			<div class="p-12 text-center text-xs text-zinc-500">Loading team members...</div>
		{:else if users.length === 0}
			<div class="p-12 text-center space-y-2">
				<p class="text-sm font-medium text-zinc-300">No staff members found</p>
				<p class="text-xs text-zinc-500">Add cashiers, accountants, and sales clerks to your team.</p>
			</div>
		{:else}
			<div class="overflow-x-auto">
				<table class="w-full text-left text-xs">
					<thead>
						<tr class="border-b border-zinc-800/60 bg-zinc-950/40 text-zinc-400 text-[11px] uppercase tracking-wider font-semibold">
							<th class="py-3 px-4">Staff Member</th>
							<th class="py-3 px-4">Role / Access</th>
							<th class="py-3 px-4">Account Status</th>
							<th class="py-3 px-4">Last Login</th>
							<th class="py-3 px-4">Joined Date</th>
							<th class="py-3 px-4 text-right">Actions</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-zinc-800/40 text-zinc-300">
						{#each users as u}
							<tr class="hover:bg-zinc-800/20 transition-colors">
								<td class="py-3.5 px-4">
									<div class="font-medium text-white">{u.full_name}</div>
									<div class="text-[11px] text-zinc-400 font-mono">{u.email}</div>
								</td>
								<td class="py-3.5 px-4">
									<select
										value={u.role}
										onchange={(e) => handleRoleChange(u, (e.target as HTMLSelectElement).value)}
										class="px-2 py-1 text-xs rounded border bg-zinc-950 font-medium {getRoleBadge(u.role)}"
									>
										{#each availableRoles as r}
											<option value={r.value}>{r.label}</option>
										{/each}
									</select>
								</td>
								<td class="py-3.5 px-4">
									{#if u.is_active}
										<span class="inline-flex items-center px-2 py-0.5 rounded-full text-[10px] font-semibold bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
											● Active
										</span>
									{:else}
										<span class="inline-flex items-center px-2 py-0.5 rounded-full text-[10px] font-semibold bg-rose-500/10 text-rose-400 border border-rose-500/20">
											○ Suspended
										</span>
									{/if}
								</td>
								<td class="py-3.5 px-4 text-zinc-400 text-[11px]">
									{u.last_login_at ? new Date(u.last_login_at).toLocaleString() : 'Never'}
								</td>
								<td class="py-3.5 px-4 text-zinc-400 text-[11px]">
									{u.created_at ? new Date(u.created_at).toLocaleDateString() : '—'}
								</td>
								<td class="py-3.5 px-4 text-right space-x-2">
									<button
										onclick={() => handleToggleActive(u)}
										class="px-2.5 py-1 text-[11px] rounded border border-zinc-800 bg-zinc-900 hover:bg-zinc-800 text-zinc-300 transition-colors"
									>
										{u.is_active ? 'Suspend' : 'Activate'}
									</button>
									<button
										onclick={() => handleDeleteUser(u)}
										class="px-2.5 py-1 text-[11px] rounded border border-red-500/20 bg-red-500/10 hover:bg-red-500/20 text-red-400 transition-colors"
									>
										Delete
									</button>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</div>

	<!-- Add Staff Modal -->
	{#if showCreateModal}
		<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 backdrop-blur-sm p-4">
			<div class="w-full max-w-md rounded-2xl border border-zinc-800 bg-zinc-950 p-6 shadow-2xl space-y-4">
				<div class="flex items-center justify-between border-b border-zinc-800 pb-3">
					<h3 class="text-sm font-semibold text-white">Add New Staff Member</h3>
					<button class="text-zinc-500 hover:text-white" onclick={() => (showCreateModal = false)}>✕</button>
				</div>

				<form onsubmit={handleCreateUser} class="space-y-3.5">
					<div>
						<label for="new-fullname" class="block text-xs font-medium text-zinc-300 mb-1">Full Name</label>
						<input
							id="new-fullname"
							type="text"
							placeholder="e.g. Rahul Sharma"
							bind:value={newFullName}
							class="w-full p-2 text-xs rounded-lg bg-zinc-900 border border-zinc-800 text-white focus:border-indigo-500"
							required
						/>
					</div>

					<div>
						<label for="new-email" class="block text-xs font-medium text-zinc-300 mb-1">Email Address</label>
						<input
							id="new-email"
							type="email"
							placeholder="e.g. rahul@company.com"
							bind:value={newEmail}
							class="w-full p-2 text-xs rounded-lg bg-zinc-900 border border-zinc-800 text-white focus:border-indigo-500"
							required
						/>
					</div>

					<div>
						<label for="new-role" class="block text-xs font-medium text-zinc-300 mb-1">Role / Permissions</label>
						<select
							id="new-role"
							bind:value={newRole}
							class="w-full p-2 text-xs rounded-lg bg-zinc-900 border border-zinc-800 text-white focus:border-indigo-500"
						>
							{#each availableRoles as r}
								<option value={r.value}>{r.label}</option>
							{/each}
						</select>
					</div>

					<div>
						<div class="flex items-center justify-between mb-1">
							<label for="new-password" class="block text-xs font-medium text-zinc-300">Temporary Password</label>
							<button
								type="button"
								onclick={generateRandomPassword}
								class="text-[10px] text-indigo-400 hover:text-indigo-300"
							>
								🎲 Generate New
							</button>
						</div>
						<input
							id="new-password"
							type="text"
							bind:value={newPassword}
							class="w-full p-2 text-xs font-mono rounded-lg bg-zinc-900 border border-zinc-800 text-white focus:border-indigo-500"
							required
						/>
					</div>

					<div class="flex items-center justify-end gap-2 pt-2 border-t border-zinc-800">
						<button
							type="button"
							onclick={() => (showCreateModal = false)}
							class="px-3 py-1.5 text-xs rounded-lg border border-zinc-800 text-zinc-400 hover:text-white"
						>
							Cancel
						</button>
						<button
							type="submit"
							disabled={isCreating}
							class="px-4 py-1.5 text-xs font-semibold rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white disabled:opacity-50"
						>
							{isCreating ? 'Adding...' : 'Create Account'}
						</button>
					</div>
				</form>
			</div>
		</div>
	{/if}
</div>
