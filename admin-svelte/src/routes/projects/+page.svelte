<script>
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { metaBaseAPI } from '$lib/api';
	import { projectActions } from '$lib/stores/projectStore';
	import { Building, Users, Plus, Edit, Trash2, Search, Filter, Settings, Folder, Globe, ArrowRight, Download, RefreshCw, X } from 'lucide-svelte';
	import { formatProjectRole, getRoleColor } from '$lib/api';

	// State
	let projects = [];
	let tenants = [];
	let loading = false;
	let error = null;
	let currentPage = 1;
	let totalPages = 1;
	let totalProjects = 0;
	let limit = 20;

	// Modal state
	let showCreateModal = false;
	let showEditModal = false;
	let showDeleteModal = false;
	let showMemberModal = false;
	let selectedProject = null;
	let selectedTenant = null;
	let editingProject = null;

	// Search and filter
	let searchTerm = '';
	let tenantFilter = 'all';
	let roleFilter = 'all';
	let statusFilter = 'all';
	let environmentFilter = 'all';
	let sortBy = 'updated_at';
	let sortOrder = 'desc';

	// Form data for create/edit
	let formData = {
		tenant_id: '',
		name: '',
		slug: '',
		description: '',
		logo: '',
		is_active: true,
		is_public: false,
		environment: 'development',
		settings: {
			database_name: '',
			database_type: 'sqlite',
			require_auth_for_read: true,
			require_auth_for_write: true,
			allowed_origins: [],
			enabled_features: [],
			rate_limit: {
				enabled: false,
				requests_per_minute: 60,
				burst_size: 10,
			},
			webhooks: {},
		},
		metadata: {}
	};

	// Available environments
	const environments = [
		{ value: 'development', label: '开发环境', color: 'bg-yellow-100 text-yellow-800' },
		{ value: 'staging', label: '测试环境', color: 'bg-blue-100 text-blue-800' },
		{ value: 'production', label: '生产环境', color: 'bg-green-100 text-green-800' }
	];

	onMount(async () => {
		await loadProjects();
		await loadTenants();
	});

	// Methods
	async function loadProjects() {
		loading = true;
		error = null;

		try {
			const response = await metaBaseAPI.getUserProjects(currentPage, limit);
			let filteredProjects = response.projects;

			// Apply filters
			if (tenantFilter !== 'all') {
				filteredProjects = filteredProjects.filter(p => p.tenant_id === tenantFilter);
			}
			if (roleFilter !== 'all') {
				filteredProjects = filteredProjects.filter(p => p.effective_role === roleFilter);
			}
			if (statusFilter !== 'all') {
				const isActive = statusFilter === 'active';
				filteredProjects = filteredProjects.filter(p => p.is_active === isActive);
			}
			if (environmentFilter !== 'all') {
				// In a real app, you'd have environment data from project details
				// For now, we'll simulate this
				filteredProjects = filteredProjects.filter(p => {
					// Mock environment assignment based on project name
					if (p.project_id.includes('prod')) return environmentFilter === 'production';
					if (p.project_id.includes('staging') || p.project_id.includes('stage')) return environmentFilter === 'staging';
					return environmentFilter === 'development';
				});
			}
			if (searchTerm) {
				const searchLower = searchTerm.toLowerCase();
				filteredProjects = filteredProjects.filter(p =>
					p.project_id.toLowerCase().includes(searchLower) ||
					p.tenant_id.toLowerCase().includes(searchLower) ||
					p.effective_role.toLowerCase().includes(searchLower)
				);
			}

			// Apply sorting
			filteredProjects.sort((a, b) => {
				let aValue = a[sortBy] || '';
				let bValue = b[sortBy] || '';

				// Handle date sorting
				if (sortBy.includes('_at')) {
					aValue = new Date(aValue).getTime();
					bValue = new Date(bValue).getTime();
				}

				// Handle string comparison
				if (typeof aValue === 'string') {
					aValue = aValue.toLowerCase();
					bValue = bValue.toLowerCase();
				}

				let comparison = 0;
				if (aValue > bValue) comparison = 1;
				if (aValue < bValue) comparison = -1;

				return sortOrder === 'desc' ? -comparison : comparison;
			});

			projects = filteredProjects;
			totalProjects = filteredProjects.length;
			totalPages = Math.ceil(totalProjects / limit);

		} catch (err) {
			error = err.message || 'Failed to load projects';
			console.error('Error loading projects:', err);
		} finally {
			loading = false;
		}
	}

	async function loadTenants() {
		try {
			const response = await metaBaseAPI.getTenants();
			tenants = response.tenants;
		} catch (err) {
			console.error('Error loading tenants:', err);
		}
	}

	// Modal functions
	function openCreateModal() {
		showCreateModal = true;
		formData = {
			tenant_id: '',
			name: '',
			slug: '',
			description: '',
			logo: '',
			is_active: true,
			is_public: false,
			environment: 'development',
			settings: {
				database_name: '',
				database_type: 'sqlite',
				require_auth_for_read: true,
				require_auth_for_write: true,
				allowed_origins: [],
				enabled_features: [],
				rate_limit: {
					enabled: false,
					requests_per_minute: 60,
					burst_size: 10,
				},
				webhooks: {},
			},
			metadata: {}
		};
	}

	function openEditModal(project) {
		showEditModal = true;
		selectedProject = project;
		// Load full project details for editing
		metaBaseAPI.getProject(project.project_id)
			.then(projectDetails => {
				editingProject = projectDetails;
			})
			.catch(err => {
				error = err.message || 'Failed to load project details';
			});
	}

	function openMemberModal(project) {
		showMemberModal = true;
		selectedProject = project;
		selectedTenant = tenants.find(t => t.id === project.tenant_id) || null;
	}

	// Close modal functions
	function closeCreateModal() {
		showCreateModal = false;
		formData = {
			tenant_id: '',
			name: '',
			slug: '',
			description: '',
			logo: '',
			is_active: true,
			is_public: false,
			environment: 'development',
			settings: {
				database_name: '',
				database_type: 'sqlite',
				require_auth_for_read: true,
				require_auth_for_write: true,
				allowed_origins: [],
				enabled_features: [],
				rate_limit: {
					enabled: false,
					requests_per_minute: 60,
					burst_size: 10,
				},
				webhooks: {},
			},
			metadata: {}
		};
	}

	function closeEditModal() {
		showEditModal = false;
		selectedProject = null;
		editingProject = null;
	}

	function closeMemberModal() {
		showMemberModal = false;
		selectedProject = null;
		selectedTenant = null;
	}

	// CRUD operations
	async function handleCreateProject() {
		try {
			if (!formData.name.trim()) {
				error = '项目名称不能为空';
				return;
			}

			if (!formData.tenant_id) {
				error = '请选择所属租户';
				return;
			}

			if (!formData.slug.trim()) {
				// Generate slug from name if not provided
				formData.slug = formData.name.toLowerCase()
					.replace(/[^a-z0-9]+/g, '-')
					.replace(/-+/g, '-');
			}

			const newProject = await metaBaseAPI.createProject(formData.tenant_id, formData);

			// Update local state - add to the list
			const newProjectEntry = {
				id: crypto.randomUUID(),
				user_id: 'current-user', // This would come from auth context
				tenant_id: formData.tenant_id,
				project_id: newProject.id,
				effective_role: 'creator',
				is_active: true,
				joined_at: new Date().toISOString(),
				left_at: undefined,
				tenant_role: 'owner',
				can_manage: true,
				metadata: {}
			};

			projects = [newProjectEntry, ...projects];
			totalProjects++;

			closeCreateModal();
		} catch (err) {
			error = err.message || 'Failed to create project';
		}
	}

	async function handleUpdateProject() {
		if (!selectedProject || !editingProject) return;

		try {
			const updatedProject = await metaBaseAPI.updateProject(selectedProject.project_id, editingProject);

			// Update local state
			projects = projects.map(p =>
				p.project_id === selectedProject.project_id
					? { ...p, /* Update project data if needed */ }
					: p
			);

			closeEditModal();
		} catch (err) {
			error = err.message || 'Failed to update project';
		}
	}

	function openDeleteModal(project) {
		selectedProject = project;
		showDeleteModal = true;
		error = null;
	}

	async function handleDeleteProject() {
		if (!selectedProject) return;

		try {
			await metaBaseAPI.deleteProject(selectedProject.project_id);

			// Update local state
			projects = projects.filter(p => p.project_id !== selectedProject.project_id);
			totalProjects--;

			closeDeleteModal();
		} catch (err) {
			error = err.message || 'Failed to delete project';
		}
	}

	// Pagination
	function changePage(page) {
		currentPage = page;
		loadProjects();
	}

	// Search and filter
	function handleSearch() {
		currentPage = 1;
		loadProjects();
	}

	// Quick action buttons
	function quickEditProject(project) {
		openEditModal(project);
	}

	function quickDeleteProject(project) {
		openDeleteModal(project);
	}

	function switchToProject(project) {
		// Use the project store to switch projects
		projectActions.switchProject(project.project_id);
		// Navigate to project dashboard
		window.location.href = `/projects/${project.project_id}`;
	}

	// Export and bulk actions
	async function exportProjects() {
		try {
			const csvContent = [
				['Project ID', 'Tenant', 'Role', 'Status', 'Joined Date'].join(','),
				...projects.map(p => [
					p.project_id,
					getTenantName(p.tenant_id),
					p.effective_role,
					p.is_active ? 'Active' : 'Inactive',
					new Date(p.joined_at).toLocaleDateString()
				].join(','))
			].join('\n');

			const blob = new Blob([csvContent], { type: 'text/csv' });
			const url = URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = `projects_${new Date().toISOString().split('T')[0]}.csv`;
			a.click();
			URL.revokeObjectURL(url);
		} catch (err) {
			error = 'Failed to export projects';
		}
	}

	function toggleSort(field) {
		if (sortBy === field) {
			sortOrder = sortOrder === 'asc' ? 'desc' : 'asc';
		} else {
			sortBy = field;
			sortOrder = 'asc';
		}
		loadProjects();
	}

	// Clear all filters
	function clearAllFilters() {
		searchTerm = '';
		tenantFilter = 'all';
		roleFilter = 'all';
		statusFilter = 'all';
		environmentFilter = 'all';
		sortBy = 'updated_at';
		sortOrder = 'desc';
		loadProjects();
	}

	// Get tenant name by ID
	function getTenantName(tenantId) {
		const tenant = tenants.find(t => t.id === tenantId);
		return tenant ? tenant.name : tenantId;
	}

	// Get environment color
	function getEnvironmentColor(environment) {
		switch (environment) {
			case 'development': return 'bg-yellow-100 text-yellow-800';
			case 'staging': return 'bg-blue-100 text-blue-800';
			case 'production': return 'bg-green-100 text-green-800';
			default: return 'bg-gray-100 text-gray-800';
		}
	}
</script>

<svelte:head>
	<title>项目管理 - MetaBase</title>
</svelte:head>

<div class="page-container">
	<!-- Page Header -->
	<div class="page-header">
		<div class="header-content">
			<div class="title-section">
				<div class="title-icon">
					<Building size={28} />
				</div>
				<div class="title-info">
					<h1 class="page-title">项目管理</h1>
					<p class="page-description">跨租户项目管理和协作中心</p>
				</div>
			</div>
			<div class="actions-section">
				<button class="btn btn-primary" on:click={openCreateModal}>
					<Plus size={20} />
					<span>创建项目</span>
				</button>
				<button class="btn btn-secondary" on:click={loadProjects}>
					<Settings size={20} />
					<span>刷新</span>
				</button>
			</div>
		</div>
	</div>

	<!-- Stats Cards -->
	<div class="stats-grid">
		<div class="stat-card">
			<div class="stat-icon">
				<Building size={24} />
			</div>
			<div class="stat-content">
				<div class="stat-value">{totalProjects}</div>
				<div class="stat-label">可访问项目</div>
			</div>
		</div>
		<div class="stat-card">
			<div class="stat-icon">
				<Users size={24} />
			</div>
			<div class="stat-content">
				<div class="stat-value">
					{projects.filter(p => p.can_manage).length}
				</div>
				<div class="stat-label">可管理项目</div>
			</div>
		</div>
		<div class="stat-card">
			<div class="stat-icon">
				<Globe size={24} />
			</div>
			<div class="stat-content">
				<div class="stat-value">
					{new Set(projects.map(p => p.tenant_id)).size}
				</div>
				<div class="stat-label">涉及租户</div>
			</div>
		</div>
	</div>

	<!-- Filters and Search -->
	<div class="filters-section">
		<div class="filters-left">
			<div class="search-box">
				<Search size={20} />
				<input
					type="text"
					placeholder="搜索项目ID、租户或角色..."
					bind:value={searchTerm}
					on:input={handleSearch}
					class="search-input"
				/>
			</div>
		</div>
		<div class="filters-right">
			<div class="filter-group">
				<label for="tenant-filter" class="filter-label">租户:</label>
				<select id="tenant-filter" bind:value={tenantFilter} on:change={loadProjects} class="filter-select">
					<option value="all">全部租户</option>
					{#each tenants as tenant}
						<option value={tenant.id}>{tenant.name}</option>
					{/each}
				</select>
			</div>
			<div class="filter-group">
				<label for="role-filter" class="filter-label">角色:</label>
				<select id="role-filter" bind:value={roleFilter} on:change={loadProjects} class="filter-select">
					<option value="all">全部角色</option>
					<option value="creator">创建者</option>
					<option value="owner">所有者</option>
					<option value="collaborator">协作者</option>
					<option value="viewer">查看者</option>
				</select>
			</div>
			<div class="filter-group">
				<label for="status-filter" class="filter-label">状态:</label>
				<select id="status-filter" bind:value={statusFilter} on:change={loadProjects} class="filter-select">
					<option value="all">全部状态</option>
					<option value="active">活跃</option>
					<option value="inactive">非活跃</option>
				</select>
			</div>
			<div class="filter-group">
				<label for="environment-filter" class="filter-label">环境:</label>
				<select id="environment-filter" bind:value={environmentFilter} on:change={loadProjects} class="filter-select">
					<option value="all">全部环境</option>
					<option value="development">开发环境</option>
					<option value="staging">测试环境</option>
					<option value="production">生产环境</option>
				</select>
			</div>
			<div class="filter-group">
				<label for="sort-by" class="filter-label">排序:</label>
				<div class="sort-controls">
					<select id="sort-by" bind:value={sortBy} on:change={loadProjects} class="filter-select">
						<option value="updated_at">更新时间</option>
						<option value="joined_at">加入时间</option>
						<option value="project_id">项目名称</option>
						<option value="tenant_id">租户</option>
						<option value="effective_role">角色</option>
					</select>
					<button class="btn btn-outline btn-sm" on:click={() => { sortOrder = sortOrder === 'asc' ? 'desc' : 'asc'; loadProjects(); }}>
						{sortOrder === 'asc' ? '↑' : '↓'}
					</button>
				</div>
			</div>
			<div class="filter-actions">
				<button class="btn btn-outline btn-sm" on:click={clearAllFilters} title="清除所有过滤器">
					<X size={16} />
				</button>
				<button class="btn btn-outline btn-sm" on:click={exportProjects} title="导出为CSV">
					<Download size={16} />
				</button>
				<button class="btn btn-outline btn-sm" on:click={loadProjects} title="刷新数据">
					<RefreshCw size={16} />
				</button>
			</div>
		</div>
	</div>

	<!-- Error Alert -->
	{#if error}
		<div class="alert alert-error">
			<span class="alert-icon">⚠</span>
			<span class="alert-text">{error}</span>
			<button class="alert-close" on:click={() => error = null} aria-label="关闭错误提示">✕</button>
		</div>
	{/if}

	<!-- Loading State -->
	{#if loading}
		<div class="loading-container">
			<div class="loading-spinner"></div>
			<p class="loading-text">正在加载项目数据...</p>
		</div>
	{/if}

	<!-- Projects List -->
	{#if !loading && !error}
		<div class="table-container">
			<div class="table-responsive">
				<table class="data-table">
					<thead>
						<tr>
							<th class="sortable" on:click={() => toggleSort('project_id')}>
								项目信息
								{#if sortBy === 'project_id'}
									<span class="sort-indicator">{sortOrder === 'asc' ? '↑' : '↓'}</span>
								{/if}
							</th>
							<th class="sortable" on:click={() => toggleSort('tenant_id')}>
								租户
								{#if sortBy === 'tenant_id'}
									<span class="sort-indicator">{sortOrder === 'asc' ? '↑' : '↓'}</span>
								{/if}
							</th>
							<th class="sortable" on:click={() => toggleSort('effective_role')}>
								角色
								{#if sortBy === 'effective_role'}
									<span class="sort-indicator">{sortOrder === 'asc' ? '↑' : '↓'}</span>
								{/if}
							</th>
							<th>环境</th>
							<th class="sortable" on:click={() => toggleSort('joined_at')}>
								加入时间
								{#if sortBy === 'joined_at'}
									<span class="sort-indicator">{sortOrder === 'asc' ? '↑' : '↓'}</span>
								{/if}
							</th>
							<th>操作</th>
						</tr>
					</thead>
					<tbody>
						{#each projects as project}
							{@const projectEnv = project.project_id.includes('prod') ? 'production' :
														 project.project_id.includes('staging') || project.project_id.includes('stage') ? 'staging' : 'development'}
							{@const envLabel = projectEnv === 'production' ? '生产环境' :
											projectEnv === 'staging' ? '测试环境' : '开发环境'}
							<tr class:active={project.is_active} class:inactive={!project.is_active}>
								<td>
									<div class="project-info">
										<div class="project-name">
											{project.project_id}
											{#if project.can_manage}
												<span class="manage-indicator">●</span>
											{/if}
										</div>
										<div class="project-id">ID: {project.id}</div>
									</div>
								</td>
								<td>
									<div class="tenant-info">
										<div class="tenant-name">{getTenantName(project.tenant_id)}</div>
										<div class="tenant-id">{project.tenant_id}</div>
									</div>
								</td>
								<td>
									<span class={`role-badge ${getRoleColor(project.effective_role)}`}>
										{formatProjectRole(project.effective_role)}
									</span>
								</td>
								<td>
									<span class={`env-badge ${getEnvironmentColor(projectEnv)}`}>
										{envLabel}
									</span>
								</td>
								<td>
									<div class="date-info">
										<div>{new Date(project.joined_at).toLocaleDateString()}</div>
										<div class="date-time">{new Date(project.joined_at).toLocaleTimeString()}</div>
									</div>
								</td>
								<td>
									<div class="action-buttons">
										<button
											class="btn-icon btn-sm btn-primary"
											title="切换到此项目"
											on:click={() => switchToProject(project)}
										>
											<ArrowRight size={16} />
										</button>
										<button
											class="btn-icon btn-sm"
											title="成员管理"
											on:click={() => openMemberModal(project)}
										>
											<Users size={16} />
										</button>
										{#if project.can_manage}
											<button
												class="btn-icon btn-sm"
												title="编辑"
												on:click={() => quickEditProject(project)}
											>
												<Edit size={16} />
											</button>
											<button
												class="btn-icon btn-sm btn-danger"
												title="删除"
												on:click={() => quickDeleteProject(project)}
											>
												<Trash2 size={16} />
											</button>
										{/if}
									</div>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>

			<!-- Pagination -->
			{#if totalPages > 1}
				<div class="pagination">
					<div class="pagination-info">
						<span>显示第 {((currentPage - 1) * limit + 1)} - {Math.min(currentPage * limit, totalProjects)} 条，共 {totalProjects} 条</span>
					</div>
					<div class="pagination-controls">
						<button
							class="btn btn-outline btn-sm"
							disabled={currentPage === 1}
							on:click={() => changePage(currentPage - 1)}
						>
							上一页
						</button>
						<span class="pagination-current">第 {currentPage} 页</span>
						<button
							class="btn btn-outline btn-sm"
							disabled={currentPage === totalPages}
							on:click={() => changePage(currentPage + 1)}
						>
							下一页
						</button>
					</div>
				</div>
			{/if}
		</div>
	{/if}

	<!-- Empty State -->
	{#if !loading && !error && projects.length === 0}
		<div class="empty-state">
			<div class="empty-icon">
				<Building size={48} />
			</div>
			<h3>暂无项目数据</h3>
			<p>您还没有访问任何项目，或者创建您的第一个项目</p>
			<button class="btn btn-primary" on:click={openCreateModal}>
				<Plus size={20} />
				创建项目
			</button>
		</div>
	{/if}
</div>

<!-- Create Project Modal -->
{#if showCreateModal}
	<div class="modal-overlay" on:click={closeCreateModal}>
		<div class="modal-container" on:click|stopPropagation>
			<div class="modal-header">
				<h2 class="modal-title">创建新项目</h2>
				<button class="modal-close" on:click={closeCreateModal}>✕</button>
			</div>
			<div class="modal-body">
				<form on:submit|preventDefault={handleCreateProject}>
					<div class="form-grid">
						<div class="form-group">
							<label for="project-name" class="form-label required">项目名称 *</label>
							<input
								id="project-name"
								type="text"
								bind:value={formData.name}
								class="form-input"
								placeholder="输入项目名称"
								required
							/>
						</div>
						<div class="form-group">
							<label for="project-slug" class="form-label">标识符</label>
							<input
								id="project-slug"
								type="text"
								bind:value={formData.slug}
								class="form-input"
								placeholder="自动生成或手动输入"
							/>
						</div>
						<div class="form-group">
							<label for="project-tenant" class="form-label required">所属租户 *</label>
							<select
								id="project-tenant"
								bind:value={formData.tenant_id}
								class="form-select"
								required
							>
								<option value="">选择租户</option>
								{#each tenants as tenant}
									<option value={tenant.id}>{tenant.name}</option>
								{/each}
							</select>
						</div>
						<div class="form-group">
							<label for="project-environment" class="form-label">环境</label>
							<select
								id="project-environment"
								bind:value={formData.environment}
								class="form-select"
							>
								{#each environments as env}
									<option value={env.value}>{env.label}</option>
								{/each}
							</select>
						</div>
						<div class="form-group full-width">
							<label for="project-description" class="form-label">描述</label>
							<textarea
								id="project-description"
								bind:value={formData.description}
								class="form-input"
								rows="3"
								placeholder="描述项目的用途和特点"
							></textarea>
						</div>
					</div>
				</form>
			</div>
			<div class="modal-footer">
				<button type="button" class="btn btn-secondary" on:click={closeCreateModal}>
					取消
				</button>
				<button type="submit" class="btn btn-primary" disabled={!formData.name.trim() || !formData.tenant_id}>
					创建项目
				</button>
			</div>
		</div>
	</div>
{/if}

<!-- Edit Project Modal -->
{#if showEditModal && selectedProject && editingProject}
	<div class="modal-overlay" on:click={closeEditModal}>
		<div class="modal-container" on:click|stopPropagation>
			<div class="modal-header">
				<h2 class="modal-title">编辑项目</h2>
				<button class="modal-close" on:click={closeEditModal}>✕</button>
			</div>
			<div class="modal-body">
				<form on:submit|preventDefault={handleUpdateProject}>
					<div class="form-grid">
						<div class="form-group">
							<label for="edit-project-name" class="form-label required">项目名称 *</label>
							<input
								id="edit-project-name"
								type="text"
								bind:value={editingProject.name}
								class="form-input"
								placeholder="输入项目名称"
								required
							/>
						</div>
						<div class="form-group">
							<label for="edit-project-slug" class="form-label">标识符</label>
							<input
								id="edit-project-slug"
								type="text"
								bind:value={editingProject.slug}
								class="form-input"
								placeholder="唯一标识符"
							/>
						</div>
						<div class="form-group">
							<label for="edit-project-environment" class="form-label">环境</label>
							<select
								id="edit-project-environment"
								bind:value={editingProject.environment}
								class="form-select"
							>
								{#each environments as env}
									<option value={env.value}>{env.label}</option>
								{/each}
							</select>
						</div>
						<div class="form-group full-width">
							<label for="edit-project-description" class="form-label">描述</label>
							<textarea
								id="edit-project-description"
								bind:value={editingProject.description}
								class="form-input"
								rows="3"
								placeholder="描述项目的用途和特点"
							></textarea>
						</div>
					</div>
				</form>
			</div>
			<div class="modal-footer">
				<button type="button" class="btn btn-secondary" on:click={closeEditModal}>
					取消
				</button>
				<button type="submit" class="btn btn-primary" disabled={!editingProject.name?.trim()}>
					保存更改
				</button>
			</div>
		</div>
	</div>
{/if}

<!-- Delete Confirmation Modal -->
{#if showDeleteModal && selectedProject}
	<div class="modal-overlay" on:click={() => showDeleteModal = false}>
		<div class="modal-container modal-sm" on:click|stopPropagation>
			<div class="modal-header">
				<h2 class="modal-title">确认删除</h2>
				<button class="modal-close" on:click={() => showDeleteModal = false}>✕</button>
			</div>
			<div class="modal-body">
				<div class="confirmation-content">
					<div class="warning-icon">
						<Trash2 size={48} />
					</div>
					<div class="confirmation-text">
						<h3>确定要删除项目「{selectedProject.project_id}」吗？</h3>
						<p>此操作不可撤销，项目相关的所有数据将被标记为已删除。</p>
					</div>
				</div>
			</div>
			<div class="modal-footer">
				<button type="button" class="btn btn-secondary" on:click={() => showDeleteModal = false}>
					取消
				</button>
				<button type="button" class="btn btn-danger" on:click={handleDeleteProject}>
					确认删除
				</button>
			</div>
		</div>
	</div>
{/if}

<!-- Project Members Modal -->
{#if showMemberModal && selectedProject && selectedTenant}
	<div class="modal-overlay" on:click={closeMemberModal}>
		<div class="modal-container modal-lg" on:click|stopPropagation>
			<div class="modal-header">
				<h2 class="modal-title">项目成员管理</h2>
				<button class="modal-close" on:click={closeMemberModal}>✕</button>
			</div>
			<div class="modal-body">
				<div class="project-info-summary">
					<div class="info-row">
						<span class="label">项目:</span>
						<span class="value">{selectedProject.project_id}</span>
					</div>
					<div class="info-row">
						<span class="label">租户:</span>
						<span class="value">{selectedTenant.name}</span>
					</div>
					<div class="info-row">
						<span class="label">您的角色:</span>
						<span class={`role-badge ${getRoleColor(selectedProject.effective_role)}`}>
							{formatProjectRole(selectedProject.effective_role)}
						</span>
					</div>
				</div>

				<div class="members-section">
					<h3>成员列表</h3>
					<p class="section-description">显示所有有权限访问此项目的租户成员</p>

					<!-- This would be populated with actual member data -->
					<div class="empty-members">
						<Users size={32} />
						<p>成员管理功能正在开发中</p>
					</div>
				</div>
			</div>
			<div class="modal-footer">
				<button type="button" class="btn btn-secondary" on:click={closeMemberModal}>
					关闭
				</button>
			</div>
		</div>
	</div>
{/if}

<style>
	/* Project-specific styles - common styles are inherited from layout */
	.page-container {
		max-width: 1200px;
		margin: 0 auto;
		padding: 0;
	}

	/* Project-specific additions */
	.manage-indicator {
		color: #10b981;
		font-size: 0.75rem;
		margin-left: 0.5rem;
	}

	.project-info {
		min-width: 200px;
	}

	.project-name {
		font-weight: 600;
		color: #111827;
		font-size: 0.875rem;
		margin-bottom: 0.25rem;
		display: flex;
		align-items: center;
	}

	.project-id {
		color: #6b7280;
		font-size: 0.75rem;
	}

	.tenant-info {
		min-width: 150px;
	}

	.tenant-name {
		font-weight: 500;
		color: #111827;
		font-size: 0.875rem;
		margin-bottom: 0.25rem;
	}

	.tenant-id {
		color: #6b7280;
		font-size: 0.75rem;
	}

	.role-badge {
		padding: 0.25rem 0.75rem;
		border-radius: 12px;
		font-size: 0.75rem;
		font-weight: 500;
	}

	.env-badge {
		padding: 0.25rem 0.75rem;
		border-radius: 12px;
		font-size: 0.75rem;
		font-weight: 500;
	}

	.btn-icon.btn-primary {
		background-color: #3b82f6;
		color: white;
		border-color: #3b82f6;
	}

	.btn-icon.btn-primary:hover {
		background-color: #2563eb;
	}

	/* Project Members Modal Styles */
	.modal-lg {
		max-width: 600px;
	}

	.project-info-summary {
		background: #f9fafb;
		border: 1px solid #e5e7eb;
		border-radius: 8px;
		padding: 1rem;
		margin-bottom: 1.5rem;
	}

	.info-row {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 0.5rem;
	}

	.info-row:last-child {
		margin-bottom: 0;
	}

	.info-row .label {
		font-weight: 500;
		color: #374151;
	}

	.info-row .value {
		color: #111827;
	}

	.members-section h3 {
		margin: 0 0 0.5rem 0;
		font-size: 1rem;
		font-weight: 600;
		color: #111827;
	}

	.section-description {
		margin: 0 0 1rem 0;
		color: #6b7280;
		font-size: 0.875rem;
	}

	.empty-members {
		text-align: center;
		padding: 2rem;
		color: #9ca3af;
		border: 2px dashed #e5e7eb;
		border-radius: 8px;
	}

	.empty-members svg {
		margin-bottom: 1rem;
	}

	/* Enhanced filter styles */
	.sort-controls {
		display: flex;
		align-items: center;
		gap: 0.25rem;
	}

	.sort-controls select {
		flex: 1;
		min-width: 120px;
	}

	.filter-actions {
		display: flex;
		gap: 0.25rem;
		align-items: center;
	}

	/* Sortable table headers */
	.data-table th.sortable {
		cursor: pointer;
		user-select: none;
		position: relative;
		padding-right: 1.5rem;
		transition: background-color 0.2s;
	}

	.data-table th.sortable:hover {
		background-color: #f3f4f6;
	}

	.sort-indicator {
		position: absolute;
		right: 0.5rem;
		top: 50%;
		transform: translateY(-50%);
		font-weight: bold;
		color: #3b82f6;
		font-size: 0.75rem;
	}

	/* Enhanced filter layout */
	.filters-section {
		background: white;
		padding: 1rem;
		border-radius: 8px;
		border: 1px solid #e5e7eb;
		margin-bottom: 1rem;
	}

	.filters-left {
		flex: 1;
		margin-bottom: 1rem;
	}

	.filters-right {
		display: flex;
		flex-wrap: wrap;
		gap: 1rem;
		align-items: end;
	}

	.filter-group {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
		min-width: 120px;
	}

	.filter-label {
		font-size: 0.75rem;
		font-weight: 500;
		color: #374151;
	}

	.filter-select {
		padding: 0.5rem;
		border: 1px solid #d1d5db;
		border-radius: 4px;
		font-size: 0.875rem;
		background: white;
		min-width: 120px;
	}

	.filter-select:focus {
		outline: none;
		border-color: #3b82f6;
		box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
	}

	/* Search box enhancements */
	.search-box {
		position: relative;
		flex: 1;
		max-width: 400px;
	}

	.search-input {
		width: 100%;
		padding: 0.5rem 1rem 0.5rem 2.5rem;
		border: 1px solid #d1d5db;
		border-radius: 6px;
		font-size: 0.875rem;
		transition: border-color 0.2s;
	}

	.search-input:focus {
		outline: none;
		border-color: #3b82f6;
		box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
	}

	.search-box svg {
		position: absolute;
		left: 0.75rem;
		top: 50%;
		transform: translateY(-50%);
		color: #6b7280;
	}

	/* Status indicators */
	.data-table tr.active {
		background-color: #f0fdf4;
	}

	.data-table tr.inactive {
		background-color: #f9fafb;
		opacity: 0.7;
	}

	.data-table tr.inactive .project-name {
		color: #6b7280;
	}

	/* Loading and transitions */
	.table-container {
		animation: fadeIn 0.3s ease-in;
	}

	@keyframes fadeIn {
		from {
			opacity: 0;
			transform: translateY(10px);
		}
		to {
			opacity: 1;
			transform: translateY(0);
		}
	}

	/* Button enhancements */
	.btn-sm {
		padding: 0.25rem 0.5rem;
		font-size: 0.75rem;
		line-height: 1rem;
	}

	.btn-outline {
		background: white;
		color: #374151;
		border-color: #d1d5db;
	}

	.btn-outline:hover {
		background: #f9fafb;
		border-color: #9ca3af;
	}

	.btn-outline:active {
		background: #f3f4f6;
	}

	/* Additional responsive adjustments */
	@media (max-width: 768px) {
		.action-buttons {
			flex-direction: column;
		}

		.filters-right {
			flex-direction: column;
			gap: 0.5rem;
		}

		.filter-group {
			width: 100%;
		}

		.filter-select {
			width: 100%;
		}

		.sort-controls {
			width: 100%;
		}

		.sort-controls select {
			flex: 1;
		}

		.filter-actions {
			margin-top: 0.5rem;
			justify-content: center;
		}

		.search-box {
			max-width: none;
		}

		.data-table th.sortable {
			padding-right: 1rem;
		}

		.sort-indicator {
			right: 0.25rem;
		}
	}

	@media (max-width: 640px) {
		.filters-section {
			padding: 0.75rem;
		}

		.filter-group {
			min-width: 100px;
		}

		.filter-select,
		.sort-controls select {
			min-width: 100px;
			font-size: 0.8rem;
		}

		.filter-label {
			font-size: 0.7rem;
		}
	}
</style>