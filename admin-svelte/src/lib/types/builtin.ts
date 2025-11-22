export interface BuiltinProject {
	id: string;
	name: string;
	description: string;
	version: string;
	category: string;
	tags: string[];
	dependencies: string[];
	config: Record<string, any>;
	isInstalled: boolean;
	isEnabled: boolean;
	installedAt?: Date;
	endpoint?: string;
	adminUrl?: string;
	healthStatus?: 'healthy' | 'unhealthy' | 'degraded';
}

export interface DeploymentRequest {
	project_id: string;
	config: Record<string, any>;
	tenant_id: string;
	auto_start?: boolean;
}

export interface DeploymentResult {
	success: boolean;
	message: string;
	project_id: string;
	version: string;
	installed_at: Date;
	endpoint?: string;
	admin_url?: string;
	config?: Record<string, any>;
	metadata?: Record<string, any>;
}

export interface ProjectTemplate {
	id: string;
	project_id: string;
	name: string;
	description: string;
	category: string;
	config: Record<string, any>;
	is_default: boolean;
	is_active: boolean;
}

export interface ProjectDependency {
	id: string;
	project_id: string;
	dependency_id: string;
	dependency_version?: string;
	is_optional: boolean;
}

export interface ProjectHealthStatus {
	id: string;
	project_id: string;
	tenant_id: string;
	status: 'healthy' | 'unhealthy' | 'degraded';
	message: string;
	metrics?: Record<string, any>;
	last_checked: Date;
}

export interface ProjectMetrics {
	id: string;
	project_id: string;
	tenant_id: string;
	metric_name: string;
	metric_value: number;
	metric_unit?: string;
	tags?: Record<string, string>;
	recorded_at: Date;
}

export interface AuthGatewayConfig {
	enable_local_auth?: boolean;
	default_provider?: string;
	session_timeout_minutes?: number;
	refresh_token_expiry_hours?: number;
	max_login_attempts?: number;
	lockout_duration_minutes?: number;
	password_policy?: PasswordPolicy;
	enable_tenant_isolation?: boolean;
	shared_user_database?: boolean;
	cross_tenant_auth?: boolean;
	default_tenant_roles?: Record<string, string>;
	enable_oauth2?: boolean;
	google_oauth2?: OAuth2Config;
	github_oauth2?: OAuth2Config;
	microsoft_oauth2?: OAuth2Config;
	enable_mfa?: boolean;
	enable_password_reset?: boolean;
	enable_email_verification?: boolean;
	enable_audit_log?: boolean;
	create_admin_user?: boolean;
	admin_username?: string;
	admin_email?: string;
	admin_password?: string;
	admin_display_name?: string;
	email_settings?: EmailSettings;
}

export interface PasswordPolicy {
	min_length: number;
	require_uppercase: boolean;
	require_lowercase: boolean;
	require_numbers: boolean;
	require_symbols: boolean;
	forbidden_words: string[];
	history_count: number;
	max_age_days: number;
}

export interface OAuth2Config {
	client_id: string;
	client_secret: string;
	redirect_url: string;
	scopes: string[];
	auth_url: string;
	token_url: string;
	user_info_url: string;
	enabled: boolean;
}

export interface EmailSettings {
	smtp_host: string;
	smtp_port: number;
	from_email: string;
	from_name: string;
	use_tls: boolean;
	username: string;
	password: string;
}

export interface CMSConfig {
	site_title: string;
	site_description: string;
	site_url: string;
	admin_email: string;
	admin_password: string;
	timezone: string;
	language: string;
	enable_comments: boolean;
	enable_ratings: boolean;
	enable_search: boolean;
	enable_categories: boolean;
	enable_tags: boolean;
	enable_media: boolean;
	enable_seo: boolean;
}

export interface BuiltinProjectStats {
	total_projects: number;
	installed_projects: number;
	healthy_projects: number;
	unhealthy_projects: number;
	total_deployments: number;
	successful_deployments: number;
	failed_deployments: number;
	last_updated: Date;
}