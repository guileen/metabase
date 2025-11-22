<script>
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';

	// State
	let providers = [];
	let loading = true;
	let error = null;
	let showCreateModal = false;
	let editingProvider = null;
	let deletingProvider = null;
	let stats = null;

	// Form state
	let formData = {
		name: '',
		display_name: '',
		type: 'oauth2',
		enabled: false,
		client_id: '',
		client_secret: '',
		auth_url: '',
		token_url: '',
		user_info_url: '',
		scopes: [],
		config: {},
		features: ['login', 'register']
	};

	// Provider types
	const providerTypes = [
		{ value: 'local', label: 'Local Authentication' },
		{ value: 'oauth2', label: 'OAuth2 / OpenID Connect' },
		{ value: 'saml', label: 'SAML' },
		{ value: 'ldap', label: 'LDAP' }
	];

	// Pre-configured providers
	const predefinedProviders = {
		google: {
			name: 'google',
			display_name: 'Google',
			type: 'oauth2',
			auth_url: 'https://accounts.google.com/o/oauth2/v2/auth',
			token_url: 'https://oauth2.googleapis.com/token',
			user_info_url: 'https://www.googleapis.com/oauth2/v2/userinfo',
			scopes: ['openid', 'profile', 'email']
		},
		github: {
			name: 'github',
			display_name: 'GitHub',
			type: 'oauth2',
			auth_url: 'https://github.com/login/oauth/authorize',
			token_url: 'https://github.com/login/oauth/access_token',
			user_info_url: 'https://api.github.com/user',
			scopes: ['user:email']
		},
		wechat: {
			name: 'wechat',
			display_name: 'WeChat',
			type: 'oauth2',
			auth_url: 'https://open.weixin.qq.com/connect/qrconnect',
			token_url: 'https://api.weixin.qq.com/sns/oauth2/access_token',
			user_info_url: 'https://api.weixin.qq.com/sns/userinfo',
			scopes: ['snsapi_userinfo']
		},
		dingtalk: {
			name: 'dingtalk',
			display_name: 'DingTalk',
			type: 'oauth2',
			auth_url: 'https://login.dingtalk.com/oauth2/auth',
			token_url: 'https://api.dingtalk.com/v1.0/oauth2/userAccessToken',
			user_info_url: 'https://api.dingtalk.com/v1.0/contact/users/me',
			scopes: ['openid', 'contact:read']
		},
		feishu: {
			name: 'feishu',
			display_name: 'Feishu',
			type: 'oauth2',
			auth_url: 'https://open.feishu.cn/open-apis/authen/v1/authorize',
			token_url: 'https://open.feishu.cn/open-apis/authen/v1/access_token',
			user_info_url: 'https://open.feishu.cn/open-apis/authen/v1/user_info',
			scopes: ['contact:user.base:readonly', 'contact:user.email:readonly']
		},
		qq: {
			name: 'qq',
			display_name: 'QQ',
			type: 'oauth2',
			auth_url: 'https://graph.qq.com/oauth2.0/authorize',
			token_url: 'https://graph.qq.com/oauth2.0/token',
			user_info_url: 'https://graph.qq.com/user/get_user_info',
			scopes: ['get_user_info']
		}
	};

	onMount(() => {
		loadProviders();
		loadStats();
	});

	async function loadProviders() {
		try {
			loading = true;
			error = null;

			const response = await fetch('/api/v1/auth/providers');
			if (!response.ok) {
				throw new Error(`Failed to load providers: ${response.statusText}`);
			}

			const data = await response.json();
			providers = data.providers || [];
		} catch (err) {
			error = err.message;
			console.error('Error loading providers:', err);
		} finally {
			loading = false;
		}
	}

	async function loadStats() {
		try {
			const response = await fetch('/api/v1/auth/providers/stats');
			if (response.ok) {
				stats = await response.json();
			}
		} catch (err) {
			console.error('Error loading stats:', err);
		}
	}

	function openCreateModal() {
		editingProvider = null;
		resetForm();
		showCreateModal = true;
	}

	function openEditModal(provider) {
		editingProvider = provider;
		formData = {
			...formData,
			...provider,
			scopes: provider.scopes || [],
			config: provider.config || {},
			features: provider.features || ['login', 'register']
		};
		showCreateModal = true;
	}

	function closeCreateModal() {
		showCreateModal = false;
		editingProvider = null;
		resetForm();
	}

	function resetForm() {
		formData = {
			name: '',
			display_name: '',
			type: 'oauth2',
			enabled: false,
			client_id: '',
			client_secret: '',
			auth_url: '',
			token_url: '',
			user_info_url: '',
			scopes: [],
			config: {},
			features: ['login', 'register']
		};
	}

	function selectPredefinedProvider(providerName) {
		const predefined = predefinedProviders[providerName];
		if (predefined) {
			Object.assign(formData, predefined);
		}
	}

	async function saveProvider() {
		try {
			const url = editingProvider
				? `/api/v1/auth/providers/${editingProvider.id}`
				: '/api/v1/auth/providers';

			const method = editingProvider ? 'PUT' : 'POST';
			const response = await fetch(url, {
				method,
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify(formData)
			});

			if (!response.ok) {
				const errorData = await response.json();
				throw new Error(errorData.error || `Failed to save provider: ${response.statusText}`);
			}

			closeCreateModal();
			await loadProviders();
			await loadStats();
		} catch (err) {
			error = err.message;
			console.error('Error saving provider:', err);
		}
	}

	async function deleteProvider(provider) {
		if (!confirm(`Are you sure you want to delete the "${provider.display_name}" provider? This action cannot be undone.`)) {
			return;
		}

		try {
			const response = await fetch(`/api/v1/auth/providers/${provider.id}`, {
				method: 'DELETE'
			});

			if (!response.ok) {
				const errorData = await response.json();
				throw new Error(errorData.error || `Failed to delete provider: ${response.statusText}`);
			}

			await loadProviders();
			await loadStats();
		} catch (err) {
			error = err.message;
			console.error('Error deleting provider:', err);
		}
	}

	async function testProvider() {
		try {
			const response = await fetch('/api/v1/auth/providers/test', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify(formData)
			});

			const result = await response.json();

			if (result.valid) {
				alert('Configuration test passed!');
			} else {
				alert(`Configuration test failed: ${result.error}`);
			}
		} catch (err) {
			alert(`Test failed: ${err.message}`);
		}
	}

	function toggleProvider(provider) {
		const updatedProvider = { ...provider, enabled: !provider.enabled };
		saveProviderDirectly(updatedProvider);
	}

	async function saveProviderDirectly(provider) {
		try {
			const response = await fetch(`/api/v1/auth/providers/${provider.id}`, {
				method: 'PUT',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify(provider)
			});

			if (!response.ok) {
				throw new Error(`Failed to update provider: ${response.statusText}`);
			}

			await loadProviders();
		} catch (err) {
			error = err.message;
			console.error('Error updating provider:', err);
		}
	}

	function addScope() {
		const scopeInput = prompt('Enter scope:');
		if (scopeInput && formData.scopes.indexOf(scopeInput) === -1) {
			formData.scopes = [...formData.scopes, scopeInput];
		}
	}

	function removeScope(index) {
		formData.scopes = formData.scopes.filter((_, i) => i !== index);
	}
</script>

<div class="container mx-auto px-4 py-8">
	<div class="flex justify-between items-center mb-8">
		<div>
			<h1 class="text-3xl font-bold text-gray-900">Authentication Providers</h1>
			<p class="text-gray-600 mt-2">Manage authentication providers and OAuth integrations</p>
		</div>
		<button
			on:click={openCreateModal}
			class="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg flex items-center gap-2"
		>
			<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"></path>
			</svg>
			Add Provider
		</button>
	</div>

	<!-- Stats Cards -->
	{#if stats}
		<div class="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
			<div class="bg-white p-6 rounded-lg shadow-sm border">
				<div class="text-sm font-medium text-gray-500">Total Users</div>
				<div class="text-2xl font-bold text-gray-900">{stats.total_users}</div>
			</div>
			<div class="bg-white p-6 rounded-lg shadow-sm border">
				<div class="text-sm font-medium text-gray-500">OAuth Users</div>
				<div class="text-2xl font-bold text-gray-900">{stats.oauth_users}</div>
			</div>
			<div class="bg-white p-6 rounded-lg shadow-sm border">
				<div class="text-sm font-medium text-gray-500">Local Users</div>
				<div class="text-2xl font-bold text-gray-900">{stats.local_users}</div>
			</div>
			<div class="bg-white p-6 rounded-lg shadow-sm border">
				<div class="text-sm font-medium text-gray-500">Active Providers</div>
				<div class="text-2xl font-bold text-gray-900">{providers.filter(p => p.enabled).length}</div>
			</div>
		</div>
	{/if}

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
	{:else}
		<!-- Providers List -->
		<div class="bg-white shadow-sm rounded-lg overflow-hidden">
			<div class="overflow-x-auto">
				<table class="min-w-full divide-y divide-gray-200">
					<thead class="bg-gray-50">
						<tr>
							<th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Provider</th>
							<th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Type</th>
							<th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
							<th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Users</th>
							<th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Features</th>
							<th class="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
						</tr>
					</thead>
					<tbody class="bg-white divide-y divide-gray-200">
						{#each providers as provider}
							<tr class="hover:bg-gray-50">
								<td class="px-6 py-4 whitespace-nowrap">
									<div class="flex items-center">
										<div class="text-sm font-medium text-gray-900">{provider.display_name}</div>
										{#if provider.name === 'local'}
											<span class="ml-2 inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
												Default
											</span>
										{/if}
									</div>
								</td>
								<td class="px-6 py-4 whitespace-nowrap">
									<span class="text-sm text-gray-500">{provider.type}</span>
								</td>
								<td class="px-6 py-4 whitespace-nowrap">
									<button
										on:click={() => toggleProvider(provider)}
										class:disabled={provider.name === 'local'}
										class="relative inline-flex flex-shrink-0 h-6 w-11 border-2 border-transparent rounded-full cursor-pointer transition-colors ease-in-out duration-200 focus:outline-none {provider.enabled ? 'bg-blue-600' : 'bg-gray-200'}"
										disabled={provider.name === 'local'}
									>
										<span class="translate-x-0 inline-block h-5 w-5 rounded-full bg-white shadow transform ring-0 transition ease-in-out duration-200 {provider.enabled ? 'translate-x-5' : 'translate-x-0'}"></span>
									</button>
								</td>
								<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
									{#each stats?.provider_stats || [] as stat}
										{#if stat.provider === provider.name}
											{stat.user_count}
										{/if}
									{/each}
									{#if !stats?.provider_stats?.find(s => s.provider === provider.name)}
										0
									{/if}
								</td>
								<td class="px-6 py-4 whitespace-nowrap">
									<div class="flex flex-wrap gap-1">
										{#each (provider.features || []) as feature}
											<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800">
												{feature}
											</span>
										{/each}
									</div>
								</td>
								<td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
									<button
										on:click={() => openEditModal(provider)}
										class="text-indigo-600 hover:text-indigo-900 mr-4"
									>
										Edit
									</button>
									{#if provider.name !== 'local'}
										<button
											on:click={() => deleteProvider(provider)}
											class="text-red-600 hover:text-red-900"
										>
											Delete
										</button>
									{/if}
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</div>
	{/if}
</div>

<!-- Create/Edit Provider Modal -->
{#if showCreateModal}
	<div class="fixed inset-0 z-10 overflow-y-auto">
		<div class="flex items-end justify-center min-h-screen pt-4 px-4 pb-20 text-center sm:block sm:p-0">
			<div class="fixed inset-0 transition-opacity" on:click={closeCreateModal}>
				<div class="absolute inset-0 bg-gray-500 opacity-75"></div>
			</div>

			<div class="inline-block align-bottom bg-white rounded-lg text-left overflow-hidden shadow-xl transform transition-all sm:my-8 sm:align-middle sm:max-w-2xl sm:w-full">
				<div class="bg-white px-4 pt-5 pb-4 sm:p-6 sm:pb-4">
					<div class="mb-4">
						<h3 class="text-lg leading-6 font-medium text-gray-900">
							{editingProvider ? 'Edit Provider' : 'Add Authentication Provider'}
						</h3>
					</div>

					<form on:submit|preventDefault={saveProvider}>
						<div class="space-y-6">
							<!-- Predefined providers -->
							{#if !editingProvider}
								<div>
									<label class="block text-sm font-medium text-gray-700 mb-2">
										Quick Setup (Select a predefined provider)
									</label>
									<div class="grid grid-cols-3 gap-3">
										{#each Object.keys(predefinedProviders) as providerName}
											<button
												type="button"
												on:click={() => selectPredefinedProvider(providerName)}
												class="px-3 py-2 border border-gray-300 rounded-md text-sm hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500"
											>
												{predefinedProviders[providerName].display_name}
											</button>
										{/each}
									</div>
								</div>
							{/if}

							<!-- Basic Info -->
							<div class="grid grid-cols-2 gap-4">
								<div>
									<label class="block text-sm font-medium text-gray-700 mb-1">
										Name *
									</label>
									<input
										type="text"
										bind:value={formData.name}
										required
										disabled={!!editingProvider}
										class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
									/>
								</div>
								<div>
									<label class="block text-sm font-medium text-gray-700 mb-1">
										Display Name *
									</label>
									<input
										type="text"
										bind:value={formData.display_name}
										required
										class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
									/>
								</div>
							</div>

							<div class="grid grid-cols-2 gap-4">
								<div>
									<label class="block text-sm font-medium text-gray-700 mb-1">
										Type *
									</label>
									<select
										bind:value={formData.type}
										disabled={!!editingProvider}
										class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
									>
										{#each providerTypes as type}
											<option value={type.value}>{type.label}</option>
										{/each}
									</select>
								</div>
								<div>
									<label class="flex items-center">
										<input
											type="checkbox"
											bind:checked={formData.enabled}
											class="rounded border-gray-300 text-blue-600 shadow-sm focus:border-blue-300 focus:ring focus:ring-blue-200 focus:ring-opacity-50"
										/>
										<span class="ml-2 text-sm text-gray-700">Enabled</span>
									</label>
								</div>
							</div>

							<!-- OAuth2 Configuration -->
							{#if formData.type === 'oauth2'}
								<div class="border-t pt-6">
									<h4 class="text-md font-medium text-gray-900 mb-4">OAuth2 Configuration</h4>

									<div class="grid grid-cols-2 gap-4 mb-4">
										<div>
											<label class="block text-sm font-medium text-gray-700 mb-1">
												Client ID *
											</label>
											<input
												type="text"
												bind:value={formData.client_id}
												required
												class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
											/>
										</div>
										<div>
											<label class="block text-sm font-medium text-gray-700 mb-1">
												Client Secret *
											</label>
											<input
												type="password"
												bind:value={formData.client_secret}
												required={!!!editingProvider}
												placeholder={editingProvider ? 'Leave unchanged to keep current secret' : ''}
												class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
											/>
										</div>
									</div>

									<div class="space-y-4">
										<div>
											<label class="block text-sm font-medium text-gray-700 mb-1">
												Authorization URL *
											</label>
											<input
												type="url"
												bind:value={formData.auth_url}
												required
												class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
											/>
										</div>
										<div>
											<label class="block text-sm font-medium text-gray-700 mb-1">
												Token URL *
											</label>
											<input
												type="url"
												bind:value={formData.token_url}
												required
												class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
											/>
										</div>
										<div>
											<label class="block text-sm font-medium text-gray-700 mb-1">
												User Info URL *
											</label>
											<input
												type="url"
												bind:value={formData.user_info_url}
												required
												class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
											/>
										</div>
									</div>

									<div>
										<label class="block text-sm font-medium text-gray-700 mb-1">
											Scopes
										</label>
										<div class="flex flex-wrap gap-2 mb-2">
											{#each formData.scopes as scope, index}
												<span class="inline-flex items-center px-3 py-1 rounded-full text-sm bg-blue-100 text-blue-800">
													{scope}
													<button
														type="button"
														on:click={() => removeScope(index)}
														class="ml-2 text-blue-600 hover:text-blue-800"
													>
														×
													</button>
												</span>
											{/each}
										</div>
										<button
											type="button"
											on:click={addScope}
											class="px-3 py-1 border border-gray-300 rounded-md text-sm hover:bg-gray-50"
										>
											Add Scope
										</button>
									</div>
								</div>
							{/if}

							<!-- Features -->
							<div>
								<label class="block text-sm font-medium text-gray-700 mb-2">
									Features
								</label>
								<div class="space-y-2">
									{#each ['login', 'register', 'password_reset'] as feature}
										<label class="flex items-center">
											<input
												type="checkbox"
												bind:group={formData.features}
												value={feature}
												class="rounded border-gray-300 text-blue-600 shadow-sm focus:border-blue-300 focus:ring focus:ring-blue-200 focus:ring-opacity-50"
											/>
											<span class="ml-2 text-sm text-gray-700 capitalize">{feature.replace('_', ' ')}</span>
										</label>
									{/each}
								</div>
							</div>
						</div>

						<div class="mt-6 flex justify-end space-x-3">
							<button
								type="button"
								on:click={closeCreateModal}
								class="px-4 py-2 border border-gray-300 rounded-md text-sm font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500"
							>
								Cancel
							</button>
							<button
								type="button"
								on:click={testProvider}
								class="px-4 py-2 border border-gray-300 rounded-md text-sm font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500"
							>
								Test Configuration
							</button>
							<button
								type="submit"
								class="px-4 py-2 bg-blue-600 border border-transparent rounded-md shadow-sm text-sm font-medium text-white hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
							>
								{editingProvider ? 'Update' : 'Create'} Provider
							</button>
						</div>
					</form>
				</div>
			</div>
		</div>
	</div>
{/if}