import { writable, derived } from 'svelte/store';
import type { BuiltinProject, DeploymentRequest, DeploymentResult } from '$lib/types/builtin';
import { apiClient } from '$lib/api/client';

interface BuiltinStoreState {
	availableProjects: BuiltinProject[];
	installedProjects: BuiltinProject[];
	loading: boolean;
	error: string | null;
}

const initialState: BuiltinStoreState = {
	availableProjects: [],
	installedProjects: [],
	loading: false,
	error: null,
};

function createBuiltinStore() {
	const { subscribe, set, update } = writable<BuiltinStoreState>(initialState);

	return {
		subscribe,

		// Actions
		async loadAvailableProjects() {
			update(state => ({ ...state, loading: true, error: null }));

			try {
				const projects = await apiClient.get<BuiltinProject[]>('/builtin/projects/available');
				update(state => ({
					...state,
					availableProjects: projects,
					loading: false
				}));
			} catch (error) {
				update(state => ({
					...state,
					loading: false,
					error: error instanceof Error ? error.message : 'Failed to load projects'
				}));
			}
		},

		async loadInstalledProjects(tenantId: string) {
			update(state => ({ ...state, loading: true, error: null }));

			try {
				const projects = await apiClient.get<BuiltinProject[]>(`/builtin/projects/installed?tenant_id=${tenantId}`);
				update(state => ({
					...state,
					installedProjects: projects,
					loading: false
				}));
			} catch (error) {
				update(state => ({
					...state,
					loading: false,
					error: error instanceof Error ? error.message : 'Failed to load installed projects'
				}));
			}
		},

		async deployProject(request: DeploymentRequest): Promise<DeploymentResult> {
			update(state => ({ ...state, loading: true, error: null }));

			try {
				const result = await apiClient.post<DeploymentResult>('/builtin/projects/deploy', request);

				// Refresh installed projects after successful deployment
				if (result.success && request.tenant_id) {
					await this.loadInstalledProjects(request.tenant_id);
				}

				update(state => ({ ...state, loading: false }));
				return result;
			} catch (error) {
				const errorMessage = error instanceof Error ? error.message : 'Deployment failed';
				update(state => ({
					...state,
					loading: false,
					error: errorMessage
				}));
				throw error;
			}
		},

		async undeployProject(projectId: string, tenantId: string): Promise<void> {
			update(state => ({ ...state, loading: true, error: null }));

			try {
				await apiClient.delete(`/builtin/projects/${projectId}/undeploy?tenant_id=${tenantId}`);

				// Refresh installed projects after successful undeployment
				await this.loadInstalledProjects(tenantId);

				update(state => ({ ...state, loading: false }));
			} catch (error) {
				const errorMessage = error instanceof Error ? error.message : 'Undeployment failed';
				update(state => ({
					...state,
					loading: false,
					error: errorMessage
				}));
				throw error;
			}
		},

		async getProjectHealth(projectId: string, tenantId: string) {
			try {
				return await apiClient.get(`/builtin/projects/${projectId}/health?tenant_id=${tenantId}`);
			} catch (error) {
				console.error('Failed to get project health:', error);
				return null;
			}
		},

		async getProjectMetrics(projectId: string, tenantId: string) {
			try {
				return await apiClient.get(`/builtin/projects/${projectId}/metrics?tenant_id=${tenantId}`);
			} catch (error) {
				console.error('Failed to get project metrics:', error);
				return null;
			}
		},

		async getProjectTemplates(projectId: string) {
			try {
				return await apiClient.get(`/builtin/projects/${projectId}/templates`);
			} catch (error) {
				console.error('Failed to get project templates:', error);
				return [];
			}
		},

		// Utility methods
		getAvailableProjects(): BuiltinProject[] {
			let projects: BuiltinProject[] = [];
			subscribe(state => projects = state.availableProjects)();
			return projects;
		},

		getInstalledProjects(): BuiltinProject[] {
			let projects: BuiltinProject[] = [];
			subscribe(state => projects = state.installedProjects)();
			return projects;
		},

		getProjectById(projectId: string): BuiltinProject | undefined {
			let projects: BuiltinProject[] = [];
			subscribe(state => projects = state.availableProjects)();
			return projects.find(p => p.id === projectId);
		},

		getInstalledProjectById(projectId: string): BuiltinProject | undefined {
			let projects: BuiltinProject[] = [];
			subscribe(state => projects = state.installedProjects)();
			return projects.find(p => p.id === projectId);
		},

		isProjectInstalled(projectId: string): boolean {
			let installedProjects: BuiltinProject[] = [];
			subscribe(state => installedProjects = state.installedProjects)();
			return installedProjects.some(p => p.id === projectId);
		},

		clearError() {
			update(state => ({ ...state, error: null }));
		},

		reset() {
			set(initialState);
		}
	};
}

// Derived stores
export const projectStore = createBuiltinStore();

export const availableProjects = derived(
	projectStore,
	$state => $state.availableProjects
);

export const installedProjects = derived(
	projectStore,
	$state => $state.installedProjects
);

export const loading = derived(
	projectStore,
	$state => $state.loading
);

export const error = derived(
	projectStore,
	$state => $state.error
);

// Helper functions for filtering and searching
export function getProjectsByCategory(category: string): BuiltinProject[] {
	let projects: BuiltinProject[] = [];
	projectStore.subscribe(state => projects = state.availableProjects)();
	return projects.filter(p => p.category === category);
}

export function searchProjects(query: string): BuiltinProject[] {
	let projects: BuiltinProject[] = [];
	projectStore.subscribe(state => projects = state.availableProjects)();

	if (!query.trim()) {
		return projects;
	}

	const lowercaseQuery = query.toLowerCase();
	return projects.filter(p =>
		p.name.toLowerCase().includes(lowercaseQuery) ||
		p.description.toLowerCase().includes(lowercaseQuery) ||
		p.tags.some(tag => tag.toLowerCase().includes(lowercaseQuery))
	);
}

export function getProjectDependencies(projectId: string): string[] {
	let projects: BuiltinProject[] = [];
	projectStore.subscribe(state => projects = state.availableProjects)();

	const project = projects.find(p => p.id === projectId);
	return project?.dependencies || [];
}

export function canDeployProject(projectId: string): boolean {
	let installedProjects: BuiltinProject[] = [];
	projectStore.subscribe(state => installedProjects = state.installedProjects)();

	const isInstalled = installedProjects.some(p => p.id === projectId);
	return !isInstalled;
}