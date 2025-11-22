<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { toast } from 'svelte-sonner';
	import type { BuiltinProject, DeploymentRequest } from '$lib/types/builtin';
	import { projectStore } from '$lib/stores/builtinStore';
	import ProjectCard from '$lib/components/builtin/ProjectCard.svelte';
	import DeploymentModal from '$lib/components/builtin/DeploymentModal.svelte';
	import LoadingSpinner from '$lib/components/ui/LoadingSpinner.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Tabs, TabsContent, TabsList, TabsTrigger } from '$lib/components/ui/tabs';
	import { Badge } from '$lib/components/ui/badge';

	$: tenantId = $page.params.tenantId || '';

	// Reactive variables
	let availableProjects: BuiltinProject[] = [];
	let installedProjects: BuiltinProject[] = [];
	let loading = true;
	let error: string | null = null;
	let selectedProject: BuiltinProject | null = null;
	let showDeploymentModal = false;
	let activeTab = 'available';

	// Load data on mount
	onMount(async () => {
		await loadProjects();
	});

	async function loadProjects() {
		loading = true;
		error = null;

		try {
			// Load available projects
			const available = await projectStore.getAvailableProjects();
			availableProjects = available;

			// Load installed projects for the tenant
			if (tenantId) {
				const installed = await projectStore.getInstalledProjects(tenantId);
				installedProjects = installed;
			}
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load projects';
			toast.error('Failed to load projects');
		} finally {
			loading = false;
		}
	}

	function handleDeploy(project: BuiltinProject) {
		selectedProject = project;
		showDeploymentModal = true;
	}

	async function handleDeploymentComplete(result: any) {
		showDeploymentModal = false;
		selectedProject = null;

		if (result.success) {
			toast.success(`Project deployed successfully: ${result.message}`);
			await loadProjects(); // Refresh the list
		} else {
			toast.error(`Deployment failed: ${result.message}`);
		}
	}

	async function handleUndeploy(project: BuiltinProject) {
		if (!confirm(`Are you sure you want to undeploy ${project.name}?`)) {
			return;
		}

		try {
			await projectStore.undeployProject(project.id, tenantId);
			toast.success('Project undeployed successfully');
			await loadProjects(); // Refresh the list
		} catch (err) {
			toast.error('Failed to undeploy project');
		}
	}

	function getStatusColor(status: string) {
		switch (status) {
			case 'healthy':
				return 'bg-green-100 text-green-800';
			case 'unhealthy':
				return 'bg-red-100 text-red-800';
			case 'degraded':
				return 'bg-yellow-100 text-yellow-800';
			default:
				return 'bg-gray-100 text-gray-800';
		}
	}
</script>

<svelte:head>
	<title>Built-in Projects - MetaBase</title>
</svelte:head>

<div class="container mx-auto p-6">
	<div class="mb-6">
		<h1 class="text-3xl font-bold text-gray-900 mb-2">Built-in Projects</h1>
		<p class="text-gray-600">Deploy pre-built projects with a single click</p>
	</div>

	{#if error}
		<div class="bg-red-50 border border-red-200 rounded-md p-4 mb-6">
			<div class="flex">
				<div class="flex-shrink-0">
					<svg class="h-5 w-5 text-red-400" viewBox="0 0 20 20" fill="currentColor">
						<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd"/>
					</svg>
				</div>
				<div class="ml-3">
					<h3 class="text-sm font-medium text-red-800">Error loading projects</h3>
					<p class="mt-1 text-sm text-red-700">{error}</p>
				</div>
			</div>
		</div>
	{/if}

	<Tabs bind:value={activeTab} className="w-full">
		<TabsList class="grid w-full grid-cols-2">
			<TabsTrigger value="available">Available Projects ({availableProjects.length})</TabsTrigger>
			<TabsTrigger value="installed">Installed Projects ({installedProjects.length})</TabsTrigger>
		</TabsList>

		<TabsContent value="available" class="mt-6">
			{#if loading}
				<div class="flex justify-center items-center h-64">
					<LoadingSpinner size="lg" />
				</div>
			{:else if availableProjects.length === 0}
				<Card>
					<CardContent class="flex flex-col items-center justify-center py-12">
						<svg class="h-12 w-12 text-gray-400 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"/>
						</svg>
						<h3 class="text-lg font-medium text-gray-900 mb-2">No available projects</h3>
						<p class="text-gray-500 text-center">There are no built-in projects available for deployment.</p>
					</CardContent>
				</Card>
			{:else}
				<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
					{#each availableProjects as project (project.id)}
						<ProjectCard
							{project}
							{tenantId}
							on:deploy={() => handleDeploy(project)}
						/>
					{/each}
				</div>
			{/if}
		</TabsContent>

		<TabsContent value="installed" class="mt-6">
			{#if loading}
				<div class="flex justify-center items-center h-64">
					<LoadingSpinner size="lg" />
				</div>
			{:else if installedProjects.length === 0}
				<Card>
					<CardContent class="flex flex-col items-center justify-center py-12">
						<svg class="h-12 w-12 text-gray-400 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4"/>
						</svg>
						<h3 class="text-lg font-medium text-gray-900 mb-2">No installed projects</h3>
						<p class="text-gray-500 text-center mb-4">You haven't deployed any built-in projects yet.</p>
						<Button onclick={() => activeTab = 'available'}>
							Browse Available Projects
						</Button>
					</CardContent>
				</Card>
			{:else}
				<div class="space-y-6">
					{#each installedProjects as project (project.id)}
						<Card>
							<CardHeader>
								<div class="flex items-start justify-between">
									<div class="flex-1">
										<CardTitle class="flex items-center gap-2">
											{project.name}
											<Badge variant="secondary">v{project.version}</Badge>
											<Badge variant="outline">{project.category}</Badge>
										</CardTitle>
										<CardDescription class="mt-1">
											{project.description}
										</CardDescription>
									</div>
									<div class="flex items-center gap-2">
										{#if project.installedAt}
											<span class="text-sm text-gray-500">
												Installed {new Date(project.installedAt).toLocaleDateString()}
											</span>
										{/if}
										<Button
											variant="outline"
											size="sm"
											onclick={() => handleUndeploy(project)}
										>
											Undeploy
										</Button>
									</div>
								</div>
							</CardHeader>
							<CardContent>
								<div class="flex items-center gap-4">
									<div class="flex items-center gap-2">
										<svg class="h-4 w-4 text-green-500" fill="currentColor" viewBox="0 0 20 20">
											<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd"/>
										</svg>
										<span class="text-sm text-gray-600">Installed</span>
									</div>

									{#each project.tags as tag}
										<Badge variant="outline" class="text-xs">{tag}</Badge>
									{/each}

									<div class="ml-auto flex gap-2">
										{#if project.config?.endpoint}
											<Button variant="outline" size="sm">
												<a href={project.config.endpoint} target="_blank" rel="noopener noreferrer">
													Open Project
												</a>
											</Button>
										{/if}
										{#if project.config?.admin_url}
											<Button variant="outline" size="sm">
												<a href={project.config.admin_url} target="_blank" rel="noopener noreferrer">
													Admin Panel
												</a>
											</Button>
										{/if}
									</div>
								</div>
							</CardContent>
						</Card>
					{/each}
				</div>
			{/if}
		</TabsContent>
	</Tabs>
</div>

{#if showDeploymentModal && selectedProject}
	<DeploymentModal
		project={selectedProject}
		{tenantId}
		on:close={() => showDeploymentModal = false}
		on:deploy={handleDeploymentComplete}
	/>
{/if}