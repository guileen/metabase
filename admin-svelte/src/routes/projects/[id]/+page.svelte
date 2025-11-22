<script>
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { metaBaseAPI } from '$lib/api';
	import { projectStore, projectActions } from '$lib/stores/projectStore';
	import { ArrowLeft, Settings, Users, Database, Activity, Shield, Globe, Code, FileText, BarChart3 } from 'lucide-svelte';
	import { currentProject } from '$lib/stores/projectStore';

	export let params = {};

	// State
	let project = null;
	let loading = false;
	let error = null;
	let stats = {
		totalUsers: 0,
		activeUsers: 0,
		apiCalls: 0,
		storage: 0,
		lastUpdate: null
	};

	// Mock data for demonstration
	let recentActivity = [
		{ id: 1, type: 'user_added', user: 'John Doe', timestamp: '2 hours ago', description: 'Joined the project' },
		{ id: 2, type: 'api_call', user: 'API Client', timestamp: '3 hours ago', description: 'Data sync completed' },
		{ id: 3, type: 'config_update', user: 'Jane Smith', timestamp: '1 day ago', description: 'Updated rate limits' },
		{ id: 4, type: 'deployment', user: 'System', timestamp: '2 days ago', description: 'Deployed to staging' }
	];

	onMount(async () => {
		await loadProject();
		await loadStats();
	});

	async function loadProject() {
		loading = true;
		error = null;

		try {
			const projectId = params.id;
			if (!projectId) {
				error = 'Project ID is required';
				return;
			}

			// Try to get from store first
			let currentProject = null;
			projectStore.subscribe(state => {
				currentProject = state.currentProject;
			})();

			if (currentProject && currentProject.project_id === projectId) {
				project = currentProject;
			} else {
				// Load from API
				const projectData = await metaBaseAPI.getProject(projectId);
				project = {
					id: projectData.id,
					project_id: projectData.name,
					tenant_id: projectData.tenant_id,
					effective_role: 'owner', // Mock role
					is_active: projectData.is_active,
					joined_at: projectData.created_at,
					can_manage: true,
					environment: projectData.environment,
					description: projectData.description,
					settings: projectData.settings
				};
			}

			// Switch to this project
			projectActions.switchProject(projectId);

		} catch (err) {
			error = err.message || 'Failed to load project';
			console.error('Error loading project:', err);
		} finally {
			loading = false;
		}
	}

	async function loadStats() {
		// Mock stats - in a real app, these would come from the API
		stats = {
			totalUsers: 24,
			activeUsers: 18,
			apiCalls: 15420,
			storage: 847,
			lastUpdate: new Date()
		};
	}

	function goBack() {
		window.history.back();
	}

	function openSettings() {
		// Navigate to project settings
		window.location.href = `/projects/${params.id}/settings`;
	}

	function openMembers() {
		// Navigate to project members
		window.location.href = `/projects/${params.id}/members`;
	}

	function getActivityIcon(type) {
		switch (type) {
			case 'user_added': return Users;
			case 'api_call': return Activity;
			case 'config_update': return Settings;
			case 'deployment': return Code;
			default: return FileText;
		}
	}

	function getActivityColor(type) {
		switch (type) {
			case 'user_added': return 'text-blue-600 bg-blue-100';
			case 'api_call': return 'text-green-600 bg-green-100';
			case 'config_update': return 'text-yellow-600 bg-yellow-100';
			case 'deployment': return 'text-purple-600 bg-purple-100';
			default: return 'text-gray-600 bg-gray-100';
		}
	}

	$: projectEnv = project?.environment || 'development';
	$: envColor = projectEnv === 'production' ? 'bg-green-100 text-green-800' :
					projectEnv === 'staging' ? 'bg-blue-100 text-blue-800' : 'bg-yellow-100 text-yellow-800';
</script>

<svelte:head>
	<title>{project?.project_id || 'Project'} - Dashboard - MetaBase</title>
</svelte:head>

{#if loading}
	<div class="loading-container">
		<div class="loading-spinner"></div>
		<p class="loading-text">正在加载项目数据...</p>
	</div>
{:else if error}
	<div class="error-container">
		<div class="error-icon">⚠️</div>
		<h2>加载失败</h2>
		<p>{error}</p>
		<button class="btn btn-primary" on:click={goBack}>
			<ArrowLeft size={20} />
			返回
		</button>
	</div>
{:else if project}
	<div class="dashboard-container">
		<!-- Header -->
		<div class="dashboard-header">
			<div class="header-left">
				<button class="btn btn-outline btn-sm" on:click={goBack}>
					<ArrowLeft size={16} />
					返回
				</button>
				<div class="project-info">
					<h1 class="project-title">{project.project_id}</h1>
					<p class="project-description">{project.description || '暂无描述'}</p>
					<div class="project-meta">
						<span class={`env-badge ${envColor}`}>
							{projectEnv === 'production' ? '生产环境' :
							 projectEnv === 'staging' ? '测试环境' : '开发环境'}
						</span>
						<span class="tenant-badge">
							<Globe size={14} />
							{project.tenant_id}
						</span>
						<span class="role-badge">
							<Users size={14} />
							所有者
						</span>
					</div>
				</div>
			</div>
			<div class="header-right">
				<button class="btn btn-secondary" on:click={openMembers}>
					<Users size={16} />
					成员管理
				</button>
				<button class="btn btn-primary" on:click={openSettings}>
					<Settings size={16} />
					项目设置
				</button>
			</div>
		</div>

		<!-- Stats Grid -->
		<div class="stats-grid">
			<div class="stat-card">
				<div class="stat-icon users">
					<Users size={24} />
				</div>
				<div class="stat-content">
					<div class="stat-value">{stats.totalUsers}</div>
					<div class="stat-label">总用户数</div>
					<div class="stat-detail">{stats.activeUsers} 活跃用户</div>
				</div>
			</div>
			<div class="stat-card">
				<div class="stat-icon api">
					<Activity size={24} />
				</div>
				<div class="stat-content">
					<div class="stat-value">{stats.apiCalls.toLocaleString()}</div>
					<div class="stat-label">API 调用</div>
					<div class="stat-detail">最近 30 天</div>
				</div>
			</div>
			<div class="stat-card">
				<div class="stat-icon storage">
					<Database size={24} />
				</div>
				<div class="stat-content">
					<div class="stat-value">{stats.storage} MB</div>
					<div class="stat-label">存储使用</div>
					<div class="stat-detail">本月使用</div>
				</div>
			</div>
			<div class="stat-card">
				<div class="stat-icon security">
					<Shield size={24} />
				</div>
				<div class="stat-content">
					<div class="stat-value">A+</div>
					<div class="stat-label">安全评级</div>
					<div class="stat-detail">所有检查通过</div>
				</div>
			</div>
		</div>

		<!-- Content Grid -->
		<div class="content-grid">
			<!-- Recent Activity -->
			<div class="content-card activity-card">
				<div class="card-header">
					<h3 class="card-title">
						<Activity size={18} />
						最近活动
					</h3>
					<button class="btn btn-outline btn-sm">查看全部</button>
				</div>
				<div class="activity-list">
					{#each recentActivity as activity}
						{@const Icon = getActivityIcon(activity.type)}
						<div class="activity-item">
							<div class="activity-icon-wrapper">
								<div class={`activity-icon ${getActivityColor(activity.type)}`}>
									<Icon size={16} />
								</div>
							</div>
							<div class="activity-content">
								<div class="activity-description">{activity.description}</div>
								<div class="activity-meta">
									<span class="activity-user">{activity.user}</span>
									<span class="activity-time">{activity.timestamp}</span>
								</div>
							</div>
						</div>
					{/each}
				</div>
			</div>

			<!-- Quick Actions -->
			<div class="content-card actions-card">
				<div class="card-header">
					<h3 class="card-title">
						<Settings size={18} />
						快速操作
					</h3>
				</div>
				<div class="quick-actions">
					<button class="action-item">
						<Users size={20} />
						<span>邀请成员</span>
					</button>
					<button class="action-item">
						<Code size={20} />
						<span>API 密钥</span>
					</button>
					<button class="action-item">
						<Database size={20} />
						<span>数据备份</span>
					</button>
					<button class="action-item">
						<BarChart3 size={20} />
						<span>使用统计</span>
					</button>
					<button class="action-item">
						<FileText size={20} />
						<span>项目日志</span>
					</button>
					<button class="action-item">
						<Shield size={20} />
						<span>安全设置</span>
					</button>
				</div>
			</div>
		</div>

		<!-- Project Settings Overview -->
		<div class="settings-overview">
			<div class="content-card">
				<div class="card-header">
					<h3 class="card-title">
						<Settings size={18} />
						项目配置
					</h3>
					<button class="btn btn-outline btn-sm" on:click={openSettings}>
						编辑设置
					</button>
				</div>
				<div class="settings-grid">
					<div class="setting-item">
						<div class="setting-label">身份验证</div>
						<div class="setting-value">
							读取：{project.settings?.require_auth_for_read ? '需要' : '不需要'}
						</div>
					</div>
					<div class="setting-item">
						<div class="setting-label">写入权限</div>
						<div class="setting-value">
							写入：{project.settings?.require_auth_for_write ? '需要' : '不需要'}
						</div>
					</div>
					<div class="setting-item">
						<div class="setting-label">数据库类型</div>
						<div class="setting-value">
							{project.settings?.database_type || 'SQLite'}
						</div>
					</div>
					<div class="setting-item">
						<div class="setting-label">速率限制</div>
						<div class="setting-value">
							{project.settings?.rate_limit?.enabled ?
								`${project.settings.rate_limit.requests_per_minute} 请求/分钟` :
								'未启用'}
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>
{/if}

<style>
	.dashboard-container {
		max-width: 1200px;
		margin: 0 auto;
		padding: 1rem;
	}

	/* Header */
	.dashboard-header {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		margin-bottom: 2rem;
		gap: 1rem;
	}

	.header-left {
		display: flex;
		align-items: flex-start;
		gap: 1rem;
	}

	.project-info {
		flex: 1;
	}

	.project-title {
		font-size: 1.875rem;
		font-weight: 700;
		color: #111827;
		margin: 0 0 0.5rem 0;
	}

	.project-description {
		color: #6b7280;
		margin: 0 0 1rem 0;
		font-size: 0.875rem;
	}

	.project-meta {
		display: flex;
		gap: 0.5rem;
		flex-wrap: wrap;
	}

	.env-badge, .tenant-badge, .role-badge {
		display: flex;
		align-items: center;
		gap: 0.25rem;
		padding: 0.25rem 0.75rem;
		border-radius: 12px;
		font-size: 0.75rem;
		font-weight: 500;
	}

	.tenant-badge {
		background-color: #f3f4f6;
		color: #374151;
	}

	.role-badge {
		background-color: #dbeafe;
		color: #1d4ed8;
	}

	.header-right {
		display: flex;
		gap: 0.5rem;
	}

	/* Stats Grid */
	.stats-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
		gap: 1rem;
		margin-bottom: 2rem;
	}

	.stat-card {
		background: white;
		padding: 1.5rem;
		border-radius: 8px;
		border: 1px solid #e5e7eb;
		display: flex;
		align-items: center;
		gap: 1rem;
		transition: transform 0.2s, box-shadow 0.2s;
	}

	.stat-card:hover {
		transform: translateY(-2px);
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
	}

	.stat-icon {
		width: 48px;
		height: 48px;
		border-radius: 8px;
		display: flex;
		align-items: center;
		justify-content: center;
		color: white;
	}

	.stat-icon.users { background-color: #3b82f6; }
	.stat-icon.api { background-color: #10b981; }
	.stat-icon.storage { background-color: #f59e0b; }
	.stat-icon.security { background-color: #8b5cf6; }

	.stat-content {
		flex: 1;
	}

	.stat-value {
		font-size: 1.5rem;
		font-weight: 700;
		color: #111827;
		line-height: 1;
		margin-bottom: 0.25rem;
	}

	.stat-label {
		font-size: 0.875rem;
		color: #374151;
		font-weight: 500;
		margin-bottom: 0.25rem;
	}

	.stat-detail {
		font-size: 0.75rem;
		color: #6b7280;
	}

	/* Content Grid */
	.content-grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 1rem;
		margin-bottom: 2rem;
	}

	.content-card {
		background: white;
		border-radius: 8px;
		border: 1px solid #e5e7eb;
		overflow: hidden;
	}

	.card-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 1rem 1.5rem;
		border-bottom: 1px solid #e5e7eb;
		background: #f9fafb;
	}

	.card-title {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 0.875rem;
		font-weight: 600;
		color: #111827;
		margin: 0;
	}

	/* Activity Card */
	.activity-card {
		min-height: 400px;
	}

	.activity-list {
		padding: 1rem;
	}

	.activity-item {
		display: flex;
		gap: 0.75rem;
		padding: 0.75rem 0;
		border-bottom: 1px solid #f3f4f6;
	}

	.activity-item:last-child {
		border-bottom: none;
	}

	.activity-icon-wrapper {
		flex-shrink: 0;
	}

	.activity-icon {
		width: 32px;
		height: 32px;
		border-radius: 50%;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.activity-content {
		flex: 1;
	}

	.activity-description {
		font-size: 0.875rem;
		color: #111827;
		margin-bottom: 0.25rem;
	}

	.activity-meta {
		display: flex;
		justify-content: space-between;
		align-items: center;
		font-size: 0.75rem;
		color: #6b7280;
	}

	/* Quick Actions */
	.actions-card {
		min-height: 400px;
	}

	.quick-actions {
		padding: 1rem;
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 0.5rem;
	}

	.action-item {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.5rem;
		padding: 1rem;
		border: 1px solid #e5e7eb;
		border-radius: 6px;
		background: white;
		color: #374151;
		font-size: 0.75rem;
		font-weight: 500;
		text-decoration: none;
		transition: all 0.2s;
		cursor: pointer;
	}

	.action-item:hover {
		background: #f9fafb;
		border-color: #d1d5db;
		transform: translateY(-1px);
	}

	/* Settings Overview */
	.settings-overview {
		margin-top: 2rem;
	}

	.settings-grid {
		padding: 1.5rem;
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
		gap: 1rem;
	}

	.setting-item {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}

	.setting-label {
		font-size: 0.75rem;
		color: #6b7280;
		font-weight: 500;
	}

	.setting-value {
		font-size: 0.875rem;
		color: #111827;
		font-weight: 500;
	}

	/* Loading */
	.loading-container {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		min-height: 400px;
		gap: 1rem;
	}

	.loading-spinner {
		width: 32px;
		height: 32px;
		border: 3px solid #e5e7eb;
		border-top: 3px solid #3b82f6;
		border-radius: 50%;
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		0% { transform: rotate(0deg); }
		100% { transform: rotate(360deg); }
	}

	.loading-text {
		color: #6b7280;
		font-size: 0.875rem;
	}

	/* Error */
	.error-container {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		min-height: 400px;
		gap: 1rem;
		text-align: center;
	}

	.error-icon {
		font-size: 3rem;
		margin-bottom: 1rem;
	}

	.error-container h2 {
		font-size: 1.25rem;
		font-weight: 600;
		color: #111827;
		margin: 0;
	}

	.error-container p {
		color: #6b7280;
		margin: 0 0 1rem 0;
	}

	/* Responsive */
	@media (max-width: 768px) {
		.dashboard-header {
			flex-direction: column;
			align-items: stretch;
		}

		.header-left {
			flex-direction: column;
			align-items: stretch;
		}

		.content-grid {
			grid-template-columns: 1fr;
		}

		.quick-actions {
			grid-template-columns: repeat(3, 1fr);
		}

		.stats-grid {
			grid-template-columns: 1fr;
		}

		.settings-grid {
			grid-template-columns: 1fr;
		}
	}
</style>