<script lang="ts">
	import { onMount } from 'svelte';
	import { createEventDispatcher } from 'svelte';
	import type { BuiltinProject } from '$lib/types/builtin';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { Switch } from '$lib/components/ui/switch';
	import { Textarea } from '$lib/components/ui/textarea';
	import LoadingSpinner from '$lib/components/ui/LoadingSpinner.svelte';
	import { toast } from 'svelte-sonner';

	export let project: BuiltinProject;
	export let tenantId: string;

	const dispatch = createEventDispatcher();

	let config: Record<string, any> = {};
	let deploying = false;
	let showAdvanced = false;

	// Initialize config with project defaults
	onMount(() => {
		config = { ...project.config };
	});

	async function handleDeploy() {
		deploying = true;

		try {
			const response = await fetch('/api/v1/builtin/projects/deploy', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
				},
				body: JSON.stringify({
					project_id: project.id,
					tenant_id: tenantId,
					config: config,
					auto_start: true,
				}),
			});

			const result = await response.json();

			if (result.success) {
				toast.success('Project deployed successfully!');
				dispatch('deploy', result);
			} else {
				toast.error(result.message || 'Deployment failed');
				dispatch('deploy', result);
			}
		} catch (error) {
			console.error('Deployment error:', error);
			toast.error('Failed to deploy project');
			dispatch('deploy', { success: false, message: 'Failed to deploy project' });
		} finally {
			deploying = false;
		}
	}

	function handleClose() {
		if (!deploying) {
			dispatch('close');
		}
	}

	// Handle keyboard shortcuts
	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape' && !deploying) {
			handleClose();
		}
	}

	// Config field generators
	function renderConfigField(key: string, value: any, path: string = '') {
		const fullPath = path ? `${path}.${key}` : key;
		const fieldValue = path ? path.split('.').reduce((obj, k) => obj?.[k], config) : config[key];

		switch (typeof value) {
			case 'boolean':
				return `
					<div class="flex items-center space-x-2">
						<Switch
							id="${fullPath}"
							bind:checked={config${path ? '.' + path : ''}[key]}
						/>
						<Label for="${fullPath}" class="text-sm font-medium">
							${formatKeyName(key)}
						</Label>
					</div>
				`;
			case 'string':
				if (key.includes('password') || key.includes('secret')) {
					return `
						<div class="space-y-2">
							<Label for="${fullPath}" class="text-sm font-medium">
								${formatKeyName(key)}
							</Label>
							<Input
								id="${fullPath}"
								type="password"
								placeholder="Enter ${formatKeyName(key).toLowerCase()}"
								bind:value={config${path ? '.' + path : ''}[key]}
							/>
						</div>
					`;
				} else if (key.includes('email')) {
					return `
						<div class="space-y-2">
							<Label for="${fullPath}" class="text-sm font-medium">
								${formatKeyName(key)}
							</Label>
							<Input
								id="${fullPath}"
								type="email"
								placeholder="Enter email address"
								bind:value={config${path ? '.' + path : ''}[key]}
							/>
						</div>
					`;
				} else if (key.includes('url') || key.includes('uri')) {
					return `
						<div class="space-y-2">
							<Label for="${fullPath}" class="text-sm font-medium">
								${formatKeyName(key)}
							</Label>
							<Input
								id="${fullPath}"
								type="url"
								placeholder="https://example.com"
								bind:value={config${path ? '.' + path : ''}[key]}
							/>
						</div>
					`;
				} else {
					return `
						<div class="space-y-2">
							<Label for="${fullPath}" class="text-sm font-medium">
								${formatKeyName(key)}
							</Label>
							<Input
								id="${fullPath}"
								type="text"
								placeholder="Enter ${formatKeyName(key).toLowerCase()}"
								bind:value={config${path ? '.' + path : ''}[key]}
							/>
						</div>
					`;
				}
			case 'number':
				return `
					<div class="space-y-2">
						<Label for="${fullPath}" class="text-sm font-medium">
							${formatKeyName(key)}
						</Label>
						<Input
							id="${fullPath}"
							type="number"
							placeholder="Enter number"
							bind:value={config${path ? '.' + path : ''}[key]}
						/>
					</div>
				`;
			case 'object':
				if (value === null) {
					return '';
				}
				return `
					<div class="space-y-3">
						<h4 class="text-sm font-medium text-gray-700">${formatKeyName(key)}</h4>
						<div class="pl-4 border-l-2 border-gray-200 space-y-3">
							${Object.entries(value).map(([subKey, subValue]) => renderConfigField(subKey, subValue, fullPath)).join('')}
						</div>
					</div>
				`;
			default:
				return '';
		}
	}

	function formatKeyName(key: string): string {
		return key
			.split('_')
			.map(word => word.charAt(0).toUpperCase() + word.slice(1))
			.join(' ');
	}
</script>

<svelte:window on:keydown={handleKeydown} />

{#if deploying}
	<div class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
		<Card class="w-full max-w-md mx-4">
			<CardContent class="flex flex-col items-center justify-center py-8">
				<LoadingSpinner size="lg" />
				<h3 class="mt-4 text-lg font-semibold">Deploying {project.name}</h3>
				<p class="text-gray-600 text-center mt-2">
					This may take a few moments. Please don't close this window.
				</p>
			</CardContent>
		</Card>
	</div>
{:else}
	<div class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
		<Card class="w-full max-w-2xl max-h-[90vh] overflow-y-auto">
			<CardHeader>
				<div class="flex items-center justify-between">
					<div>
						<CardTitle>Deploy {project.name}</CardTitle>
						<CardDescription>
							Configure and deploy the {project.name} project to your tenant
						</CardDescription>
					</div>
					<Button variant="ghost" size="sm" onclick={handleClose}>
						<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
						</svg>
					</Button>
				</div>
			</CardHeader>

			<CardContent class="space-y-6">
				<!-- Project Info -->
				<div class="bg-gray-50 p-4 rounded-lg">
					<div class="flex items-start gap-3">
						<div class="flex-shrink-0">
							<div class="w-10 h-10 bg-blue-100 rounded-lg flex items-center justify-center">
								<svg class="h-6 w-6 text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"/>
								</svg>
							</div>
						</div>
						<div>
							<h3 class="font-medium text-gray-900">{project.name}</h3>
							<p class="text-sm text-gray-600 mt-1">{project.description}</p>
							<div class="flex items-center gap-2 mt-2">
								<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
									{project.category}
								</span>
								<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800">
									v{project.version}
								</span>
							</div>
						</div>
					</div>
				</div>

				<!-- Dependencies Check -->
				{#if project.dependencies && project.dependencies.length > 0}
					<div class="border-l-4 border-yellow-400 bg-yellow-50 p-4">
						<div class="flex">
							<div class="flex-shrink-0">
								<svg class="h-5 w-5 text-yellow-400" viewBox="0 0 20 20" fill="currentColor">
									<path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clip-rule="evenodd"/>
								</svg>
							</div>
							<div class="ml-3">
								<h3 class="text-sm font-medium text-yellow-800">
									Dependencies Required
								</h3>
								<div class="mt-2 text-sm text-yellow-700">
									This project requires: {project.dependencies.join(', ')}
								</div>
							</div>
						</div>
					</div>
				{/if}

				<!-- Configuration -->
				<div class="space-y-4">
					<div class="flex items-center justify-between">
						<h3 class="text-lg font-medium text-gray-900">Configuration</h3>
						<Button variant="outline" size="sm" onclick={() => showAdvanced = !showAdvanced}>
							{showAdvanced ? 'Hide' : 'Show'} Advanced
						</Button>
					</div>

					<!-- Basic Configuration -->
					<div class="space-y-4">
						{#each Object.entries(config).filter(([key]) => !isAdvancedKey(key)) as [key, value]}
							{@html renderConfigField(key, value)}
						{/each}
					</div>

					<!-- Advanced Configuration -->
					{#if showAdvanced}
						<div class="border-t pt-4">
							<h4 class="text-sm font-medium text-gray-700 mb-3">Advanced Settings</h4>
							<div class="space-y-4">
								{#each Object.entries(config).filter(([key]) => isAdvancedKey(key)) as [key, value]}
									{@html renderConfigField(key, value)}
								{/each}
							</div>
						</div>
					{/if}
				</div>

				<!-- Actions -->
				<div class="flex justify-end gap-3 pt-4 border-t">
					<Button variant="outline" onclick={handleClose}>
						Cancel
					</Button>
					<Button onclick={handleDeploy}>
						<svg class="h-4 w-4 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M9 19l3 3m0 0l3-3m-3 3V10"/>
						</svg>
						Deploy Project
					</Button>
				</div>
			</CardContent>
		</Card>
	</div>
{/if}

<script context="module">
	function isAdvancedKey(key: string): boolean {
		const advancedKeys = [
			'secret',
			'private_key',
			'client_secret',
			'webhook_secret',
			'encryption_key',
			'database_url',
			'redis_url',
		];
		return advancedKeys.some(advancedKey => key.toLowerCase().includes(advancedKey));
	}
</script>