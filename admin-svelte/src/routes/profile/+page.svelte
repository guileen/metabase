<script>
	import { onMount } from 'svelte';
	import { page } from '$app/stores';

	// State
	let profile = null;
	let connectedAccounts = [];
	let loading = true;
	let error = null;
	let activeTab = 'profile';
	let showPasswordModal = false;
	let showSyncModal = false;

	// Form state
	let profileForm = {
		username: '',
		email: '',
		phone: '',
		first_name: '',
		last_name: '',
		display_name: '',
		avatar: '',
		metadata: {}
	};

	let passwordForm = {
		current_password: '',
		new_password: '',
		confirm_password: ''
	};

	onMount(() => {
		loadProfile();
		loadConnectedAccounts();
	});

	async function loadProfile() {
		try {
			loading = true;
			error = null;

			const response = await fetch('/api/v1/user/profile');
			if (!response.ok) {
				throw new Error(`Failed to load profile: ${response.statusText}`);
			}

			const data = await response.json();
			profile = data;

			// Populate form
			profileForm = {
				username: data.username || '',
				email: data.email || '',
				phone: data.phone || '',
				first_name: data.first_name || '',
				last_name: data.last_name || '',
				display_name: data.display_name || '',
				avatar: data.avatar || '',
				metadata: data.metadata || {}
			};
		} catch (err) {
			error = err.message;
			console.error('Error loading profile:', err);
		} finally {
			loading = false;
		}
	}

	async function loadConnectedAccounts() {
		try {
			const response = await fetch('/api/v1/user/connected-accounts');
			if (response.ok) {
				const data = await response.json();
				connectedAccounts = data.connected_accounts || [];
			}
		} catch (err) {
			console.error('Error loading connected accounts:', err);
		}
	}

	async function updateProfile() {
		try {
			error = null;

			const response = await fetch('/api/v1/user/profile', {
				method: 'PUT',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify(profileForm)
			});

			if (!response.ok) {
				const errorData = await response.json();
				throw new Error(errorData.error || `Failed to update profile: ${response.statusText}`);
			}

			await loadProfile();
			alert('Profile updated successfully!');
		} catch (err) {
			error = err.message;
			console.error('Error updating profile:', err);
		}
	}

	async function changePassword() {
		try {
			error = null;

			// Validate form
			if (passwordForm.new_password !== passwordForm.confirm_password) {
				throw new Error('New password and confirmation do not match');
			}

			if (passwordForm.new_password.length < 8) {
				throw new Error('New password must be at least 8 characters');
			}

			const response = await fetch('/api/v1/user/profile/password', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify(passwordForm)
			});

			if (!response.ok) {
				const errorData = await response.json();
				throw new Error(errorData.error || `Failed to change password: ${response.statusText}`);
			}

			closePasswordModal();
			alert('Password changed successfully!');
		} catch (err) {
			error = err.message;
			console.error('Error changing password:', err);
		}
	}

	async function syncProfile() {
		try {
			error = null;
			showSyncModal = true;

			const response = await fetch('/api/v1/user/profile/sync', {
				method: 'POST'
			});

			if (!response.ok) {
				throw new Error(`Failed to sync profile: ${response.statusText}`);
			}

			const result = await response.json();

			showSyncModal = false;
			await loadProfile();
			await loadConnectedAccounts();

			alert(`Profile synchronized from ${result.sync_count} providers: ${result.synced_providers.join(', ')}`);
		} catch (err) {
			error = err.message;
			showSyncModal = false;
			console.error('Error syncing profile:', err);
		}
	}

	async function disconnectAccount(account) {
		if (!confirm(`Are you sure you want to disconnect your ${account.provider} account?`)) {
			return;
		}

		try {
			const response = await fetch(`/api/v1/user/connected-accounts/${account.id}/disconnect`, {
				method: 'DELETE'
			});

			if (!response.ok) {
				const errorData = await response.json();
				throw new Error(errorData.error || `Failed to disconnect account: ${response.statusText}`);
			}

			await loadProfile();
			await loadConnectedAccounts();
			alert('Account disconnected successfully!');
		} catch (err) {
			error = err.message;
			console.error('Error disconnecting account:', err);
		}
	}

	function openPasswordModal() {
		passwordForm = {
			current_password: '',
			new_password: '',
			confirm_password: ''
		};
		showPasswordModal = true;
	}

	function closePasswordModal() {
		showPasswordModal = false;
		passwordForm = {
			current_password: '',
			new_password: '',
			confirm_password: ''
		};
	}

	function getProviderIcon(provider) {
		const icons = {
			google: '🔍',
			github: '🐙',
			wechat: '💬',
			dingtalk: '💼',
			feishu: '🚀',
			qq: '🐧'
		};
		return icons[provider] || '🔐';
	}

	function getProviderDisplayName(provider) {
		const names = {
			google: 'Google',
			github: 'GitHub',
			wechat: 'WeChat',
			dingtalk: 'DingTalk',
			feishu: 'Feishu',
			qq: 'QQ'
		};
		return names[provider] || provider;
	}
</script>

<div class="container mx-auto px-4 py-8 max-w-4xl">
	<div class="mb-8">
		<h1 class="text-3xl font-bold text-gray-900">My Profile</h1>
		<p class="text-gray-600 mt-2">Manage your profile information and connected accounts</p>
	</div>

	<!-- Error Alert -->
	{#if error}
		<div class="bg-red-50 border-l-4 border-red-400 p-4 mb-6">
			<div class="flex">
				<div class="flex-shrink-0">
					<svg class="h-5 w-5 text-red-400" viewBox="0 0 20 20" fill="currentColor">
						<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd"></path>
					</svg>
				</div>
				<div class="ml-3">
					<p class="text-sm text-red-700">{error}</p>
				</div>
			</div>
		</div>
	{/if}

	<!-- Loading State -->
	{#if loading}
		<div class="flex justify-center items-center h-64">
			<div class="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
		</div>
	{:else if profile}
		<!-- Tabs -->
		<div class="border-b border-gray-200 mb-8">
			<nav class="-mb-px flex space-x-8">
				<button
					on:click={() => activeTab = 'profile'}
					class="py-2 px-1 border-b-2 font-medium text-sm {activeTab === 'profile' ? 'border-blue-500 text-blue-600' : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'}"
				>
					Profile Information
				</button>
				<button
					on:click={() => activeTab = 'security'}
					class="py-2 px-1 border-b-2 font-medium text-sm {activeTab === 'security' ? 'border-blue-500 text-blue-600' : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'}"
				>
					Security
				</button>
				<button
					on:click={() => activeTab = 'accounts'}
					class="py-2 px-1 border-b-2 font-medium text-sm {activeTab === 'accounts' ? 'border-blue-500 text-blue-600' : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'}"
				>
					Connected Accounts
				</button>
			</nav>
		</div>

		<!-- Profile Information Tab -->
		{#if activeTab === 'profile'}
			<div class="bg-white shadow-sm rounded-lg p-6">
				<form on:submit|preventDefault={updateProfile}>
					<div class="space-y-6">
						<!-- Avatar -->
						<div class="flex items-center space-x-6">
							<div class="shrink-0">
								<img
									class="h-20 w-20 object-cover rounded-full"
									src={profileForm.avatar || `https://ui-avatars.com/api/?name=${encodeURIComponent(profileForm.display_name || profileForm.username)}&background=3b82f6&color=fff`}
									alt="Profile avatar"
								/>
							</div>
							<div>
								<button type="button" class="bg-white py-2 px-3 border border-gray-300 rounded-md shadow-sm text-sm leading-4 font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2">
									Change Avatar
								</button>
								<p class="mt-1 text-sm text-gray-500">JPG, GIF or PNG. 1MB max.</p>
							</div>
						</div>

						<!-- Basic Information -->
						<div>
							<h3 class="text-lg font-medium text-gray-900 mb-4">Basic Information</h3>
							<div class="grid grid-cols-1 md:grid-cols-2 gap-6">
								<div>
									<label class="block text-sm font-medium text-gray-700 mb-1">
										Username
									</label>
									<input
										type="text"
										bind:value={profileForm.username}
										class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
									/>
								</div>
								<div>
									<label class="block text-sm font-medium text-gray-700 mb-1">
										Display Name
									</label>
									<input
										type="text"
										bind:value={profileForm.display_name}
										class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
									/>
								</div>
								<div>
									<label class="block text-sm font-medium text-gray-700 mb-1">
										Email Address
									</label>
									<input
										type="email"
										bind:value={profileForm.email}
										class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
									/>
									{#if profile.email_verified}
										<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800 mt-1">
											✓ Verified
										</span>
									{/if}
								</div>
								<div>
									<label class="block text-sm font-medium text-gray-700 mb-1">
										Phone Number
									</label>
									<input
										type="tel"
										bind:value={profileForm.phone}
										class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
									/>
									{#if profile.phone_verified}
										<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800 mt-1">
											✓ Verified
										</span>
									{/if}
								</div>
								<div>
									<label class="block text-sm font-medium text-gray-700 mb-1">
										First Name
									</label>
									<input
										type="text"
										bind:value={profileForm.first_name}
										class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
									/>
								</div>
								<div>
									<label class="block text-sm font-medium text-gray-700 mb-1">
										Last Name
									</label>
									<input
										type="text"
										bind:value={profileForm.last_name}
										class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
									/>
								</div>
							</div>
						</div>

						<!-- Account Information -->
						<div>
							<h3 class="text-lg font-medium text-gray-900 mb-4">Account Information</h3>
							<div class="grid grid-cols-1 md:grid-cols-2 gap-6">
								<div>
									<label class="block text-sm font-medium text-gray-700 mb-1">
										Authentication Provider
									</label>
									<div class="flex items-center space-x-2">
										<span>{getProviderIcon(profile.provider)}</span>
										<span class="text-gray-900">{getProviderDisplayName(profile.provider)}</span>
									</div>
								</div>
								<div>
									<label class="block text-sm font-medium text-gray-700 mb-1">
										Member Since
									</label>
									<div class="text-gray-900">{new Date(profile.created_at).toLocaleDateString()}</div>
								</div>
								<div>
									<label class="block text-sm font-medium text-gray-700 mb-1">
										Last Sign In
									</label>
									<div class="text-gray-900">
										{profile.last_sign_in_at ? new Date(profile.last_sign_in_at).toLocaleString() : 'Never'}
									</div>
								</div>
								<div>
									<label class="block text-sm font-medium text-gray-700 mb-1">
										Account Status
									</label>
									<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium {profile.is_active ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'}">
										{profile.is_active ? 'Active' : 'Inactive'}
									</span>
								</div>
							</div>
						</div>

						<!-- Actions -->
						<div class="flex justify-end space-x-3">
							<button
								type="button"
								on:click={syncProfile}
								class="px-4 py-2 border border-gray-300 rounded-md shadow-sm text-sm font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500"
							>
								Sync Profile
							</button>
							<button
								type="submit"
								class="px-4 py-2 bg-blue-600 border border-transparent rounded-md shadow-sm text-sm font-medium text-white hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
							>
								Save Changes
							</button>
						</div>
					</div>
				</form>
			</div>
		{/if}

		<!-- Security Tab -->
		{#if activeTab === 'security'}
			<div class="bg-white shadow-sm rounded-lg p-6">
				<h3 class="text-lg font-medium text-gray-900 mb-6">Security Settings</h3>

				<!-- Password Change -->
				{#if profile.provider === 'local' || !profile.provider}
					<div class="border-b pb-6 mb-6">
						<div class="flex justify-between items-center mb-4">
							<div>
								<h4 class="text-md font-medium text-gray-900">Password</h4>
								<p class="text-sm text-gray-500">Change your account password</p>
							</div>
							<button
								on:click={openPasswordModal}
								class="px-4 py-2 border border-gray-300 rounded-md shadow-sm text-sm font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500"
							>
								Change Password
							</button>
						</div>
					</div>
				{:else}
					<div class="border-b pb-6 mb-6">
						<div class="bg-gray-50 p-4 rounded-lg">
							<h4 class="text-md font-medium text-gray-900 mb-2">Authentication Method</h4>
							<p class="text-sm text-gray-600">
								You're using {getProviderDisplayName(profile.provider)} for authentication.
								Password management is handled by your {getProviderDisplayName(profile.provider)} account.
							</p>
						</div>
					</div>
				{/if}

				<!-- Verification Status -->
				<div>
					<h4 class="text-md font-medium text-gray-900 mb-4">Verification Status</h4>
					<div class="space-y-4">
						<div class="flex items-center justify-between">
							<div class="flex items-center space-x-3">
								<div class="w-8 h-8 rounded-full bg-gray-200 flex items-center justify-center">
									<svg class="w-5 h-5 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"></path>
									</svg>
								</div>
								<div>
									<div class="text-sm font-medium text-gray-900">Email Address</div>
									<div class="text-sm text-gray-500">{profile.email}</div>
								</div>
							</div>
							<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium {profile.email_verified ? 'bg-green-100 text-green-800' : 'bg-yellow-100 text-yellow-800'}">
								{profile.email_verified ? 'Verified' : 'Not Verified'}
							</span>
						</div>

						{#if profile.phone}
							<div class="flex items-center justify-between">
								<div class="flex items-center space-x-3">
									<div class="w-8 h-8 rounded-full bg-gray-200 flex items-center justify-center">
										<svg class="w-5 h-5 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 5a2 2 0 012-2h3.28a1 1 0 01.948.684l1.498 4.493a1 1 0 01-.502 1.21l-2.257 1.13a11.042 11.042 0 005.516 5.516l1.13-2.257a1 1 0 011.21-.502l4.493 1.498a1 1 0 01.684.949V19a2 2 0 01-2 2h-1C9.716 21 3 14.284 3 6V5z"></path>
										</svg>
									</div>
									<div>
										<div class="text-sm font-medium text-gray-900">Phone Number</div>
										<div class="text-sm text-gray-500">{profile.phone}</div>
									</div>
								</div>
								<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium {profile.phone_verified ? 'bg-green-100 text-green-800' : 'bg-yellow-100 text-yellow-800'}">
									{profile.phone_verified ? 'Verified' : 'Not Verified'}
								</span>
							</div>
						{/if}
					</div>
				</div>
			</div>
		{/if}

		<!-- Connected Accounts Tab -->
		{#if activeTab === 'accounts'}
			<div class="bg-white shadow-sm rounded-lg p-6">
				<div class="flex justify-between items-center mb-6">
					<div>
						<h3 class="text-lg font-medium text-gray-900">Connected Accounts</h3>
						<p class="text-sm text-gray-500 mt-1">Manage your connected OAuth accounts</p>
					</div>
					<button
						on:click={syncProfile}
						class="px-4 py-2 border border-gray-300 rounded-md shadow-sm text-sm font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500"
					>
						Sync All Accounts
					</button>
				</div>

				{#if connectedAccounts.length === 0}
					<div class="text-center py-8">
						<div class="w-12 h-12 mx-auto bg-gray-200 rounded-full flex items-center justify-center mb-4">
							<svg class="w-6 h-6 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"></path>
							</svg>
						</div>
						<h3 class="text-lg font-medium text-gray-900 mb-2">No connected accounts</h3>
						<p class="text-gray-500">Connect your OAuth accounts for easier login and profile synchronization.</p>
					</div>
				{:else}
					<div class="space-y-4">
						{#each connectedAccounts as account}
							<div class="flex items-center justify-between p-4 border rounded-lg hover:bg-gray-50">
								<div class="flex items-center space-x-4">
									<div class="w-12 h-12 bg-gray-100 rounded-full flex items-center justify-center text-2xl">
										{getProviderIcon(account.provider)}
									</div>
									<div>
										<div class="text-sm font-medium text-gray-900">{getProviderDisplayName(account.provider)}</div>
										<div class="text-sm text-gray-500">Connected on {new Date(account.created_at).toLocaleDateString()}</div>
										{#if account.is_active}
											<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
												Active
											</span>
										{:else}
											<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800">
												Inactive
											</span>
										{/if}
									</div>
								</div>
								<button
									on:click={() => disconnectAccount(account)}
									class="px-3 py-1 border border-red-300 rounded-md text-sm font-medium text-red-700 hover:bg-red-50 focus:outline-none focus:ring-2 focus:ring-red-500"
								>
									Disconnect
								</button>
							</div>
						{/each}
					</div>
				{/if}

				<!-- Add New Account Section -->
				<div class="mt-8 pt-6 border-t">
					<h4 class="text-md font-medium text-gray-900 mb-4">Add New Account</h4>
					<div class="grid grid-cols-2 md:grid-cols-3 gap-4">
						{#each ['google', 'github', 'wechat'] as provider}
							<button class="p-4 border border-gray-300 rounded-lg hover:border-blue-500 hover:bg-blue-50 focus:outline-none focus:ring-2 focus:ring-blue-500">
								<div class="text-2xl mb-2">{getProviderIcon(provider)}</div>
								<div class="text-sm font-medium text-gray-900">{getProviderDisplayName(provider)}</div>
							</button>
						{/each}
					</div>
				</div>
			</div>
		{/if}
	{/if}
</div>

<!-- Password Change Modal -->
{#if showPasswordModal}
	<div class="fixed inset-0 z-10 overflow-y-auto">
		<div class="flex items-end justify-center min-h-screen pt-4 px-4 pb-20 text-center sm:block sm:p-0">
			<div class="fixed inset-0 transition-opacity" on:click={closePasswordModal}>
				<div class="absolute inset-0 bg-gray-500 opacity-75"></div>
			</div>

			<div class="inline-block align-bottom bg-white rounded-lg text-left overflow-hidden shadow-xl transform transition-all sm:my-8 sm:align-middle sm:max-w-lg sm:w-full">
				<div class="bg-white px-4 pt-5 pb-4 sm:p-6 sm:pb-4">
					<h3 class="text-lg leading-6 font-medium text-gray-900 mb-4">Change Password</h3>

					<form on:submit|preventDefault={changePassword}>
						<div class="space-y-4">
							<div>
								<label class="block text-sm font-medium text-gray-700 mb-1">
									Current Password
								</label>
								<input
									type="password"
									bind:value={passwordForm.current_password}
									required
									class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
								/>
							</div>
							<div>
								<label class="block text-sm font-medium text-gray-700 mb-1">
									New Password
								</label>
								<input
									type="password"
									bind:value={passwordForm.new_password}
									required
									minlength="8"
									class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
								/>
							</div>
							<div>
								<label class="block text-sm font-medium text-gray-700 mb-1">
									Confirm New Password
								</label>
								<input
									type="password"
									bind:value={passwordForm.confirm_password}
									required
									minlength="8"
									class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
								/>
							</div>
						</div>

						<div class="mt-6 flex justify-end space-x-3">
							<button
								type="button"
								on:click={closePasswordModal}
								class="px-4 py-2 border border-gray-300 rounded-md text-sm font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500"
							>
								Cancel
							</button>
							<button
								type="submit"
								class="px-4 py-2 bg-red-600 border border-transparent rounded-md shadow-sm text-sm font-medium text-white hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-red-500"
							>
								Change Password
							</button>
						</div>
					</form>
				</div>
			</div>
		</div>
	</div>
{/if}

<!-- Sync Modal -->
{#if showSyncModal}
	<div class="fixed inset-0 z-10 overflow-y-auto">
		<div class="flex items-end justify-center min-h-screen pt-4 px-4 pb-20 text-center sm:block sm:p-0">
			<div class="fixed inset-0 transition-opacity">
				<div class="absolute inset-0 bg-gray-500 opacity-75"></div>
			</div>

			<div class="inline-block align-bottom bg-white rounded-lg text-left overflow-hidden shadow-xl transform transition-all sm:my-8 sm:align-middle sm:max-w-sm sm:w-full">
				<div class="bg-white px-4 pt-5 pb-4 sm:p-6">
					<div class="flex items-center justify-center">
						<div class="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
					</div>
					<h3 class="text-lg leading-6 font-medium text-gray-900 text-center mt-4">
						Synchronizing Profile
					</h3>
					<p class="text-sm text-gray-500 text-center mt-2">
						Please wait while we sync your profile from connected accounts...
					</p>
				</div>
			</div>
		</div>
	</div>
{/if}