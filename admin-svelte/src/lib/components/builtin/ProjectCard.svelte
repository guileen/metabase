<script lang="ts">
	import type { BuiltinProject } from '$lib/types/builtin';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { createEventDispatcher } from 'svelte';

	export let project: BuiltinProject;
	export let tenantId: string;

	const dispatch = createEventDispatcher();

	function handleDeploy() {
		dispatch('deploy');
	}

	function getCategoryColor(category: string) {
		const colors: Record<string, string> = {
			security: 'bg-red-100 text-red-800',
			content: 'bg-blue-100 text-blue-800',
			analytics: 'bg-green-100 text-green-800',
			utility: 'bg-gray-100 text-gray-800',
			communication: 'bg-purple-100 text-purple-800',
		};
		return colors[category] || 'bg-gray-100 text-gray-800';
	}

	function getDependencyStatus(): string {
		if (!project.dependencies || project.dependencies.length === 0) {
			return 'No dependencies';
		}
		return `${project.dependencies.length} dependenc${project.dependencies.length === 1 ? 'y' : 'ies'}`;
	}
</script>

<Card class="h-full flex flex-col">
	<CardHeader>
		<div class="flex items-start justify-between">
			<div class="flex-1">
				<CardTitle class="flex items-center gap-2 text-lg">
					{project.name}
					<Badge variant="secondary" class="text-xs">v{project.version}</Badge>
				</CardTitle>
				<div class="flex items-center gap-2 mt-2">
					<Badge class={getCategoryColor(project.category)}>
						{project.category}
					</Badge>
					{#if project.isInstalled}
						<Badge variant="default" class="bg-green-100 text-green-800">Installed</Badge>
					{/if}
				</div>
			</div>
			<div class="flex-shrink-0">
				{#if project.tags.includes('popular')}
					<svg class="h-5 w-5 text-yellow-500" fill="currentColor" viewBox="0 0 20 20">
						<path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z"/>
					</svg>
				{/if}
			</div>
		</div>
	</CardHeader>

	<CardContent class="flex-1">
		<CardDescription class="text-sm text-gray-600 mb-4">
			{project.description}
		</CardDescription>

		<!-- Features -->
		{#if project.tags && project.tags.length > 0}
			<div class="mb-4">
				<h4 class="text-sm font-medium text-gray-700 mb-2">Features:</h4>
				<div class="flex flex-wrap gap-1">
					{#each project.tags as tag}
						<Badge variant="outline" class="text-xs">{tag}</Badge>
					{/each}
				</div>
			</div>
		{/if}

		<!-- Dependencies -->
		<div class="mb-4">
			<div class="flex items-center justify-between text-sm">
				<span class="text-gray-600">Dependencies:</span>
				<span class="font-medium">{getDependencyStatus()}</span>
			</div>
			{#if project.dependencies && project.dependencies.length > 0}
				<div class="flex flex-wrap gap-1 mt-1">
					{#each project.dependencies as dep}
						<Badge variant="outline" class="text-xs bg-gray-50">{dep}</Badge>
					{/each}
				</div>
			{/if}
		</div>

		<!-- Configuration Preview -->
		{#if project.config}
			<div class="border-t pt-3">
				<h4 class="text-sm font-medium text-gray-700 mb-2">Quick Config:</h4>
				<div class="space-y-1 text-xs text-gray-600">
					{#if project.config.enable_local_auth !== undefined}
						<div class="flex justify-between">
							<span>Local Auth:</span>
							<span class="font-medium">{project.config.enable_local_auth ? 'Enabled' : 'Disabled'}</span>
						</div>
					{/if}
					{#if project.config.enable_tenant_isolation !== undefined}
						<div class="flex justify-between">
							<span>Tenant Isolation:</span>
							<span class="font-medium">{project.config.enable_tenant_isolation ? 'Enabled' : 'Disabled'}</span>
						</div>
					{/if}
					{#if project.config.session_timeout_minutes}
						<div class="flex justify-between">
							<span>Session Timeout:</span>
							<span class="font-medium">{project.config.session_timeout_minutes} min</span>
						</div>
					{/if}
				</div>
			</div>
		{/if}
	</CardContent>

	<CardFooter class="pt-3">
		<div class="flex gap-2 w-full">
			{#if project.isInstalled}
				<Button variant="outline" class="flex-1" disabled>
					<svg class="h-4 w-4 mr-2" fill="currentColor" viewBox="0 0 20 20">
						<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd"/>
					</svg>
					Installed
				</Button>
			{:else}
				<Button class="flex-1" onclick={handleDeploy}>
					<svg class="h-4 w-4 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M9 19l3 3m0 0l3-3m-3 3V10"/>
					</svg>
					Deploy
				</Button>
			{/if}
			<Button variant="outline" size="sm" onclick={() => window.open(`/docs/builtin/${project.id}`, '_blank')}>
				<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
				</svg>
			</Button>
		</div>
	</CardFooter>
</Card>