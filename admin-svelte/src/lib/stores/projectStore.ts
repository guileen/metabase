import { writable, derived } from 'svelte/store';
import { type UserTenantProject } from '$lib/api';
import { browser } from '$app/environment';

// Project state interface
interface ProjectState {
	currentProject: UserTenantProject | null;
	projects: UserTenantProject[];
	isLoading: boolean;
	error: string | null;
}

// Create writable store
const initialState: ProjectState = {
	currentProject: null,
	projects: [],
	isLoading: false,
	error: null,
};

export const projectStore = writable<ProjectState>(initialState);

// Save/load from localStorage
if (browser) {
	// Load saved project preference
	const savedProjectId = localStorage.getItem('metabase_current_project_id');
	if (savedProjectId) {
		// Will be used to restore project after projects are loaded
		projectStore.update(state => ({
			...state,
			_currentProjectId: savedProjectId,
		}));
	}
}

// Derived stores
export const currentProject = derived(projectStore, $projectStore => $projectStore.currentProject);
export const availableProjects = derived(projectStore, $projectStore => $projectStore.projects);
export const isLoading = derived(projectStore, $projectStore => $projectStore.isLoading);
export const projectError = derived(projectStore, $projectStore => $projectStore.error);

// Actions
export const projectActions = {
	// Set available projects
	setProjects: (projects: UserTenantProject[]) => {
		projectStore.update(state => {
			let currentProject = state.currentProject;

			// Try to restore previously selected project
			if (!currentProject && browser) {
				const savedProjectId = localStorage.getItem('metabase_current_project_id');
				if (savedProjectId) {
					currentProject = projects.find(p => p.project_id === savedProjectId) || null;
				}
			}

			// If still no current project, select the first one
			if (!currentProject && projects.length > 0) {
				currentProject = projects[0];
				if (browser) {
					localStorage.setItem('metabase_current_project_id', currentProject.project_id);
				}
			}

			return {
				...state,
				projects,
				currentProject,
				_currentProjectId: undefined, // Clear temporary storage
			};
		});
	},

	// Switch to a different project
	switchProject: (projectId: string) => {
		projectStore.update(state => {
			const project = state.projects.find(p => p.project_id === projectId);
			if (!project) {
				return {
					...state,
					error: `Project ${projectId} not found`,
				};
			}

			if (browser) {
				localStorage.setItem('metabase_current_project_id', projectId);
			}

			return {
				...state,
				currentProject: project,
				error: null,
			};
		});
	},

	// Add a new project to the list
	addProject: (project: UserTenantProject) => {
		projectStore.update(state => ({
			...state,
			projects: [...state.projects, project],
		}));
	},

	// Update a project in the list
	updateProject: (projectId: string, updates: Partial<UserTenantProject>) => {
		projectStore.update(state => ({
			...state,
			projects: state.projects.map(p =>
				p.project_id === projectId ? { ...p, ...updates } : p
			),
			currentProject: state.currentProject?.project_id === projectId
				? { ...state.currentProject, ...updates }
				: state.currentProject,
		}));
	},

	// Remove a project from the list
	removeProject: (projectId: string) => {
		projectStore.update(state => {
			const projects = state.projects.filter(p => p.project_id !== projectId);
			let currentProject = state.currentProject;

			// If we removed the current project, switch to another one
			if (currentProject?.project_id === projectId) {
				currentProject = projects.length > 0 ? projects[0] : null;
				if (browser && currentProject) {
					localStorage.setItem('metabase_current_project_id', currentProject.project_id);
				} else if (browser && !currentProject) {
					localStorage.removeItem('metabase_current_project_id');
				}
			}

			return {
				...state,
				projects,
				currentProject,
			};
		});
	},

	// Set loading state
	setLoading: (loading: boolean) => {
		projectStore.update(state => ({
			...state,
			isLoading: loading,
		}));
	},

	// Set error
	setError: (error: string | null) => {
		projectStore.update(state => ({
			...state,
			error,
		}));
	},

	// Clear current project (logout)
	clearCurrentProject: () => {
		projectStore.update(state => ({
			...state,
			currentProject: null,
		}));
		if (browser) {
			localStorage.removeItem('metabase_current_project_id');
		}
	},
};

// Helper functions
export const hasProjectAccess = () => {
	let accessible = false;
	projectStore.subscribe(state => {
		accessible = state.projects.length > 0;
	})();
	return accessible;
};

export const canManageCurrentProject = () => {
	let canManage = false;
	projectStore.subscribe(state => {
		canManage = state.currentProject?.can_manage || false;
	})();
	return canManage;
};

export const getCurrentProjectRole = () => {
	let role = '';
	projectStore.subscribe(state => {
		role = state.currentProject?.effective_role || '';
	})();
	return role;
};