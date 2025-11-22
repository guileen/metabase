// API client for MetaBase tenant and project management

// Base API configuration
const API_BASE_URL = 'http://localhost:7610/admin/v1';

// Types
export interface Tenant {
	id: string;
	name: string;
	slug: string;
	domain?: string;
	logo?: string;
	description?: string;
	settings: TenantSettings;
	metadata?: Record<string, any>;
	is_active: boolean;
	plan: 'free' | 'pro' | 'enterprise';
	limits: TenantLimits;
	created_at: string;
	updated_at: string;
	deleted_at?: string;
}

export interface TenantSettings {
	allow_user_registration: boolean;
	default_user_role: string;
	required_email_domains: string[];
	require_email_verification: boolean;
	require_two_factor: boolean;
	session_timeout_minutes: number;
	enabled_features?: string[];
	theme?: ThemeSettings;
	custom_css?: string;
	custom_js?: string;
	webhook_url?: string;
	webhooks?: Record<string, string>;
}

export interface ThemeSettings {
	primary_color?: string;
	secondary_color?: string;
	logo_url?: string;
	favicon_url?: string;
	company_name?: string;
}

export interface TenantLimits {
	max_users: number;
	max_projects: number;
	max_storage_mb: number;
	max_api_requests_per_day: number;
}

export interface Project {
	id: string;
	tenant_id: string;
	name: string;
	slug: string;
	description?: string;
	logo?: string;
	settings: ProjectSettings;
	metadata?: Record<string, any>;
	is_active: boolean;
	is_public: boolean;
	environment: 'development' | 'staging' | 'production';
	owner_id: string;
	members?: ProjectMember[];
	created_at: string;
	updated_at: string;
	deleted_at?: string;
}

export interface ProjectSettings {
	database_name?: string;
	database_type?: string;
	require_auth_for_read: boolean;
	require_auth_for_write: boolean;
	allowed_origins?: string[];
	enabled_features?: string[];
	rate_limit?: RateLimitSettings;
	webhooks?: Record<string, string>;
}

export interface RateLimitSettings {
	enabled: boolean;
	requests_per_minute: number;
	burst_size: number;
}

export interface ProjectMember {
	id: string;
	project_id: string;
	tenant_id: string;
	role: 'creator' | 'owner' | 'collaborator' | 'viewer';
	is_active: boolean;
	is_creator: boolean;
	invited_by?: string;
	invited_at?: string;
	joined_at: string;
	left_at?: string;
	is_external_collaborator: boolean;
	can_invite: boolean;
	can_manage_members: boolean;
	contact_user_id?: string;
	custom_permissions?: string[];
	metadata?: Record<string, any>;
}

export interface UserTenantProject {
	id: string;
	user_id: string;
	tenant_id: string;
	project_id: string;
	effective_role: string;
	is_active: boolean;
	joined_at: string;
	left_at?: string;
	tenant_role: string;
	can_manage: boolean;
	metadata?: Record<string, any>;
}

export interface InviteUserRequest {
	user_id: string;
	email?: string;
	role: string;
	message?: string;
}

export interface TransferOwnershipRequest {
	to_user_id: string;
	message?: string;
}

// API client class
class MetaBaseAPI {
	private baseURL: string;
	private authToken: string | null = null;

	constructor(baseURL: string = API_BASE_URL) {
		this.baseURL = baseURL;
		// In a real app, you would get the auth token from localStorage or a store
		this.authToken = 'mock-jwt-token';
	}

	private async request<T>(
		endpoint: string,
		options: RequestInit = {}
	): Promise<T> {
		const url = `${this.baseURL}${endpoint}`;
		const headers: HeadersInit = {
			'Content-Type': 'application/json',
		};

		if (this.authToken) {
			headers.Authorization = `Bearer ${this.authToken}`;
		}

		const response = await fetch(url, {
			...options,
			headers,
		});

		if (!response.ok) {
			const error = await response.json();
			throw new Error(error.error || error.message || 'API request failed');
		}

		return response.json();
	}

	// Tenant Management APIs
	async getTenants(page = 1, limit = 20) {
		return this.request<{ tenants: Tenant[]; total: number; page: number; limit: number }>(`/tenants?page=${page}&limit=${limit}`);
	}

	async createTenant(tenant: Omit<Tenant, 'id' | 'created_at' | 'updated_at' | 'deleted_at'>) {
		return this.request<Tenant>('/tenants', {
			method: 'POST',
			body: JSON.stringify(tenant),
		});
	}

	async getTenant(tenantId: string) {
		return this.request<Tenant>(`/tenants/${tenantId}`);
	}

	async updateTenant(tenantId: string, tenant: Partial<Tenant>) {
		return this.request<Tenant>(`/tenants/${tenantId}`, {
			method: 'PUT',
			body: JSON.stringify(tenant),
		});
	}

	async deleteTenant(tenantId: string) {
		return this.request<{ message: string; id: string }>(`/tenants/${tenantId}`, {
			method: 'DELETE',
		});
	}

	// Project Management APIs
	async getUserProjects(page = 1, limit = 20) {
		return this.request<{ projects: UserTenantProject[]; total: number; page: number; limit: number }>(`/projects?page=${page}&limit=${limit}`);
	}

	async createProject(tenantId: string, project: Omit<Project, 'id' | 'created_at' | 'updated_at' | 'deleted_at' | 'owner_id'>) {
		return this.request<Project>(`/tenants/${tenantId}/projects`, {
			method: 'POST',
			body: JSON.stringify(project),
		});
	}

	async getProject(projectId: string) {
		return this.request<Project>(`/projects/${projectId}`);
	}

	async updateProject(projectId: string, project: Partial<Project>) {
		return this.request<Project>(`/projects/${projectId}`, {
			method: 'PUT',
			body: JSON.stringify(project),
		});
	}

	async deleteProject(projectId: string) {
		return this.request<{ message: string; id: string }>(`/projects/${projectId}`, {
			method: 'DELETE',
		});
	}

	// Project Member Management APIs
	async inviteUserToProject(projectId: string, invite: InviteUserRequest) {
		return this.request<{ message: string; user_id: string; project_id: string; role: string }>(`/projects/${projectId}/invite`, {
			method: 'POST',
			body: JSON.stringify(invite),
		});
	}

	async getProjectMembers(projectId: string) {
		return this.request<{ members: ProjectMember[]; total: number }>(`/projects/${projectId}/members`);
	}

	async removeUserFromProject(projectId: string, userId: string) {
		return this.request<{ message: string; user_id: string; project_id: string }>(`/projects/${projectId}/members/${userId}`, {
			method: 'DELETE',
		});
	}

	async transferOwnership(projectId: string, transfer: TransferOwnershipRequest) {
		return this.request<{ message: string; project_id: string; from_user: string; to_user: string }>(`/projects/${projectId}/transfer-ownership`, {
			method: 'POST',
			body: JSON.stringify(transfer),
		});
	}

	// Get projects by tenant
	async getTenantProjects(tenantId: string) {
		return this.request<{ projects: Project[]; total: number }>(`/tenants/${tenantId}/projects`);
	}
}

// Create a singleton instance
export const metaBaseAPI = new MetaBaseAPI();

// Utility functions
export const formatTenantLimits = (limits: TenantLimits): string => {
	return `用户: ${limits.max_users}, 项目: ${limits.max_projects}, 存储: ${limits.max_storage_mb}MB, API请求: ${limits.max_api_requests_per_day}`;
};

export const formatProjectRole = (role: string): string => {
	switch (role) {
		case 'creator': return '创建者';
		case 'owner': return '所有者';
		case 'collaborator': return '协作者';
		case 'viewer': return '查看者';
		default: return role;
	}
};

export const getRoleColor = (role: string): string => {
	switch (role) {
		case 'creator': return 'bg-purple-100 text-purple-800 border-purple-200';
		case 'owner': return 'bg-blue-100 text-blue-800 border-blue-200';
		case 'collaborator': return 'bg-green-100 text-green-800 border-green-200';
		case 'viewer': return 'bg-gray-100 text-gray-800 border-gray-200';
		default: return 'bg-gray-100 text-gray-800 border-gray-200';
	}
};

export const getTenantPlanColor = (plan: string): string => {
	switch (plan) {
		case 'free': return 'bg-gray-100 text-gray-800 border-gray-200';
		case 'pro': return 'bg-blue-100 text-blue-800 border-blue-200';
		case 'enterprise': return 'bg-purple-100 text-purple-800 border-purple-200';
		default: return 'bg-gray-100 text-gray-800 border-gray-200';
	}
};