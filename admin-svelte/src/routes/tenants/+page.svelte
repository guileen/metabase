<script>
	import { onMount } from 'svelte';
	import { metaBaseAPI } from '$lib/api';
	import { Building, Users, Globe, Plus, Edit, Trash2, Search, Filter, Settings } from 'lucide-svelte';
	import { getTenantPlanColor, formatTenantLimits } from '$lib/api';

	// State
	let tenants = [];
	let loading = false;
	let error = null;
	let currentPage = 1;
	let totalPages = 1;
	let totalTenants = 0;
	let limit = 20;

	// Modal state
	let showCreateModal = false;
	let showEditModal = false;
	let showDeleteModal = false;
	let selectedTenant = null;
	let editingTenant = null;

	// Search and filter
	let searchTerm = '';
	let planFilter = 'all';
	let statusFilter = 'all';

	// Form data for create/edit
	let formData = {
		name: '',
		slug: '',
		domain: '',
		logo: '',
		description: '',
		plan: 'free',
		settings: {
			allow_user_registration: false,
			default_user_role: 'user',
			required_email_domains: [],
			require_email_verification: false,
			require_two_factor: false,
			session_timeout_minutes: 1440,
				},
		limits: {
			max_users: 10,
			max_projects: 5,
			max_storage_mb: 1024,
			max_api_requests_per_day: 10000,
		},
		metadata: {}
	};

	// Available plans
	const plans = [
		{ value: 'free', label: '免费版', color: 'bg-gray-100 text-gray-800', description: '适合个人用户和小团队' },
		{ value: 'pro', label: '专业版', color: 'bg-blue-100 text-blue-800', description: '适合成长型团队' },
		{ value: 'enterprise', label: '企业版', color: 'bg-purple-100 text-purple-800', description: '适合大型企业' }
	];

	onMount(() => {
		loadTenants();
	});

	// Methods
	async function loadTenants() {
		loading = true;
		error = null;

		try {
			// Apply filters
			const params = new URLSearchParams({
				page: currentPage.toString(),
				limit: limit.toString()
			});

			if (searchTerm) {
				// In a real app, you'd handle search on the server
				// For now, we'll search after loading
			}

			const response = await metaBaseAPI.getTenants(currentPage, limit);
			tenants = response.tenants;
			totalTenants = response.total;
			totalPages = Math.ceil(response.total / limit);

			// Apply client-side filters for demo purposes
			if (planFilter !== 'all') {
				tenants = tenants.filter(t => t.plan === planFilter);
			}
			if (statusFilter !== 'all') {
				tenants = tenants.filter(t => t.is_active === (statusFilter === 'active'));
			}
			if (searchTerm) {
				tenants = tenants.filter(t =>
					t.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
					t.slug.toLowerCase().includes(searchTerm.toLowerCase())
				);
			}

		} catch (err) {
			error = err.message || 'Failed to load tenants';
			console.error('Error loading tenants:', err);
		} finally {
			loading = false;
		}
	}

	// Modal functions
	function openCreateModal() {
		showCreateModal = true;
		formData = {
			name: '',
			slug: '',
			domain: '',
			logo: '',
			description: '',
			plan: 'free',
			settings: {
				allow_user_registration: false,
				default_user_role: 'user',
				required_email_domains: [],
				require_email_verification: false,
				require_two_factor: false,
				session_timeout_minutes: 1440,
			},
			limits: {
				max_users: 10,
				max_projects: 5,
				max_storage_mb: 1024,
				max_api_requests_per_day: 10000,
			},
			metadata: {}
		};
	}

	function openEditModal(tenant) {
		showEditModal = true;
		selectedTenant = tenant;
		editingTenant = { ...tenant };
	}

	// Close modal functions
	function closeCreateModal() {
		showCreateModal = false;
		formData = {
			name: '',
			slug: '',
			domain: '',
			logo: '',
			description: '',
			plan: 'free',
			settings: {
				allow_user_registration: false,
				default_user_role: 'user',
				required_email_domains: [],
				require_email_verification: false,
				require_two_factor: false,
				session_timeout_minutes: 1440,
			},
			limits: {
				max_users: 10,
				max_projects: 5,
				max_storage_mb: 1024,
				max_api_requests_per_day: 10000,
			},
			metadata: {}
		};
	}

		function closeEditModal() {
		showEditModal = false;
		selectedTenant = null;
		editingTenant = null;
	}

	// CRUD operations
	async function handleCreateTenant() {
		try {
			// Validate form
			if (!formData.name.trim()) {
				error = '租户名称不能为空';
				return;
			}

			if (!formData.slug.trim()) {
				// Generate slug from name if not provided
				formData.slug = formData.name.toLowerCase()
					.replace(/[^a-z0-9]+/g, '-')
					.replace(/-+/g, '-');
			}

			const newTenant = await metaBaseAPI.createTenant(formData);

			// Update limits based on plan
			if (newTenant.plan === 'pro') {
				newTenant.limits.max_users = 50;
				newTenant.limits.max_projects = 20;
				newTenant.limits.max_storage_mb = 5120;
				newTenant.limits.max_api_requests_per_day = 50000;
			} else if (newTenant.plan === 'enterprise') {
				newTenant.limits.max_users = 999;
				newTenant.limits.max_projects = 999;
				newTenant.limits.max_storage_mb = 99999;
				newTenant.limits.max_api_requests_per_day = 999999;
			}

			await metaBaseAPI.updateTenant(newTenant.id, newTenant);

			// Update local state
			tenants = [newTenant, ...tenants];
			totalTenants++;

			closeCreateModal();
		} catch (err) {
			error = err.message || 'Failed to create tenant';
		}
	}

	async function handleUpdateTenant() {
		if (!selectedTenant || !editingTenant) return;

		try {
			const updatedTenant = await metaBaseAPI.updateTenant(selectedTenant.id, editingTenant);

			// Update local state
			tenants = tenants.map(t => t.id === selectedTenant.id ? { ...t, ...editingTenant } : t);

			closeEditModal();
		} catch (err) {
			error = err.message || 'Failed to update tenant';
		}
	}

		function openDeleteModal(tenant) {
		selectedTenant = tenant;
		showDeleteModal = true;
		error = null;
	}

		async function handleDeleteTenant() {
		if (!selectedTenant) return;

		// Prevent deletion of system tenant
		if (selectedTenant.id === 'system') {
			error = '不能删除系统租户';
			return;
		}

		try {
			await metaBaseAPI.deleteTenant(selectedTenant.id);

			// Update local state
			tenants = tenants.filter(t => t.id !== selectedTenant.id);
			totalTenants--;

			closeDeleteModal();
		} catch (err) {
			error = err.message || 'Failed to delete tenant';
		}
	}

	// Pagination
	function changePage(page) {
		currentPage = page;
		loadTenants();
	}

	// Search and filter
	function handleSearch() {
		currentPage = 1;
		loadTenants();
	}

	// Plan management functions
	function updatePlanLimits(plan) {
		switch (plan) {
			case 'free':
				formData.limits = {
					max_users: 10,
					max_projects: 5,
					max_storage_mb: 1024,
					max_api_requests_per_day: 10000,
				};
				break;
			case 'pro':
				formData.limits = {
					max_users: 50,
					max_projects: 20,
					max_storage_mb: 5120,
					max_api_requests_per_day: 50000,
				};
				break;
			case 'enterprise':
				formData.limits = {
					max_users: 999,
					max_projects: 999,
					max_storage_mb: 99999,
					max_api_requests_per_day: 999999,
				};
				break;
		}
	}

	// Quick action buttons
	function quickEditTenant(tenant) {
		openEditModal(tenant);
	}

	function quickDeleteTenant(tenant) {
		openDeleteModal(tenant);
	}
</script>

<svelte:head>
	<title>租户管理 - MetaBase</title>
</svelte:head>

<div class="page-container">
	<!-- Page Header -->
	<div class="page-header">
		<div class="header-content">
			<div class="title-section">
				<div class="title-icon">
					<Globe size={28} />
				</div>
				<div class="title-info">
					<h1 class="page-title">租户管理</h1>
					<p class="page-description">多租户系统配置和管理中心</p>
				</div>
			</div>
			<div class="actions-section">
				<button class="btn btn-primary" on:click={openCreateModal}>
					<Plus size={20} />
					<span>创建租户</span>
				</button>
				<button class="btn btn-secondary" on:click={loadTenants}>
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
				<div class="stat-value">{totalTenants}</div>
				<div class="stat-label">总租户数</div>
			</div>
		</div>
		<div class="stat-card">
			<div class="stat-icon">
				<Users size={24} />
			</div>
			<div class="stat-content">
				<div class="stat-value">
					{tenants.reduce((sum, t) => sum + t.limits.max_users, 0).toLocaleString()}
				</div>
				<div class="stat-label">用户容量</div>
			</div>
		</div>
		<div class="stat-card">
			<div class="stat-icon">
				<Globe size={24} />
			</div>
			<div class="stat-content">
				<div class="stat-value">
					{tenants.filter(t => t.is_active).length}
				</div>
				<div class="stat-label">活跃租户</div>
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
					placeholder="搜索租户名称或标识符..."
					bind:value={searchTerm}
					on:input={handleSearch}
					class="search-input"
				/>
			</div>
		</div>
		<div class="filters-right">
			<div class="filter-group">
				<label for="plan-filter" class="filter-label">计划:</label>
				<select id="plan-filter" bind:value={planFilter} on:change={loadTenants} class="filter-select">
					<option value="all">全部</option>
					{#each plans as plan}
						<option value={plan.value}>{plan.label}</option>
					{/each}
				</select>
			</div>
			<div class="filter-group">
				<label for="status-filter" class="filter-label">状态:</label>
				<select id="status-filter" bind:value={statusFilter} on:change={loadTenants} class="filter-select">
					<option value="all">全部</option>
					<option value="active">活跃</option>
					<option value="inactive">未激活</option>
				</select>
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
			<p class="loading-text">正在加载租户数据...</p>
		</div>
	{/if}

	<!-- Tenants List -->
	{#if !loading && !error}
		<div class="table-container">
			<div class="table-responsive">
				<table class="data-table">
					<thead>
						<tr>
							<th>租户信息</th>
							<th>计划</th>
							<th>资源限制</th>
							<th>状态</th>
							<th>创建时间</th>
							<th>操作</th>
						</tr>
					</thead>
					<tbody>
						{#each tenants as tenant}
							<tr class:active={tenant.is_active} class:inactive={!tenant.is_active}>
								<td>
									<div class="tenant-info">
										<div class="tenant-name">
											{tenant.name}
											{#if tenant.slug}
												<span class="tenant-slug">({tenant.slug})</span>
											{/if}
										</div>
										{#if tenant.description}
											<div class="tenant-description">{tenant.description}</div>
										{/if}
										{#if tenant.domain}
											<div class="tenant-domain">域名: {tenant.domain}</div>
										{/if}
									</div>
								</td>
								<td>
									<span class={`plan-badge ${getTenantPlanColor(tenant.plan)}`}>
										{tenant.plan === 'free' ? '免费' : tenant.plan === 'pro' ? '专业' : '企业'}
									</span>
								</td>
								<td>
									<div class="limits-info">
										<div>{formatTenantLimits(tenant.limits)}</div>
									</div>
								</td>
								<td>
									<span class={`status-badge ${tenant.is_active ? 'status-active' : 'status-inactive'}`}>
										{tenant.is_active ? '激活' : '未激活'}
									</span>
								</td>
								<td>
									<div class="date-info">
										<div>{new Date(tenant.created_at).toLocaleDateString()}</div>
										<div class="date-time">{new Date(tenant.created_at).toLocaleTimeString()}</div>
									</div>
								</td>
								<td>
									<div class="action-buttons">
										<button
											class="btn-icon btn-sm"
											title="编辑"
											on:click={() => quickEditTenant(tenant)}
										>
											<Edit size={16} />
										</button>
										<button
											class="btn-icon btn-sm btn-danger"
											title="删除"
											on:click={() => quickDeleteTenant(tenant)}
											disabled={tenant.id === 'system'}
										>
											<Trash2 size={16} />
										</button>
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
						<span>显示第 {((currentPage - 1) * limit + 1)} - {Math.min(currentPage * limit, totalTenants)} 条，共 {totalTenants} 条</span>
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
	{#if !loading && !error && tenants.length === 0}
		<div class="empty-state">
			<div class="empty-icon">
				<Building size={48} />
			</div>
			<h3>暂无租户数据</h3>
			<p>开始创建您的第一个租户来使用多租户功能</p>
			<button class="btn btn-primary" on:click={openCreateModal}>
				<Plus size={20} />
				创建租户
			</button>
		</div>
	{/if}

	<!-- Create Tenant Modal -->
	{#if showCreateModal}
	<div class="modal-overlay" on:click={closeCreateModal}>
			<div class="modal-container" on:click|stopPropagation>
				<div class="modal-header">
					<h2 class="modal-title">创建新租户</h2>
					<button class="modal-close" on:click={closeCreateModal}>✕</button>
				</div>
				<div class="modal-body">
					<form on:submit|preventDefault={handleCreateTenant}>
						<div class="form-grid">
							<div class="form-group">
								<label for="tenant-name" class="form-label required">租户名称 *</label>
								<input
									id="tenant-name"
									type="text"
									bind:value={formData.name}
									class="form-input"
									placeholder="输入租户名称"
									required
								/>
							</div>
							<div class="form-group">
								<label for="tenant-slug" class="form-label">标识符</label>
								<input
									id="tenant-slug"
									type="text"
									bind:value={formData.slug}
									class="form-input"
									placeholder="自动生成或手动输入"
								/>
							</div>
							<div class="form-group">
								<label for="tenant-domain" class="form-label">域名</label>
								<input
									id="tenant-domain"
									type="text"
									bind:value={formData.domain}
									class="form-input"
									placeholder="example.com"
								/>
							</div>
							<div class="form-group">
								<label for="tenant-description" class="form-label">描述</label>
								<textarea
									id="tenant-description"
									bind:value={formData.description}
									class="form-input"
									rows="3"
									placeholder="描述租户的用途和特点"
								></textarea>
							</div>
							<div class="form-group">
								<label for="tenant-plan" class="form-label">订阅计划</label>
								<select
									id="tenant-plan"
									bind:value={formData.plan}
									on:change={() => updatePlanLimits(formData.plan)}
									class="form-select"
								>
									{#each plans as plan}
										<option value={plan.value}>{plan.label} - {plan.description}</option>
									{/each}
								</select>
							</div>
							<div class="form-group full-width">
								<label class="form-label">资源限制</label>
								<div class="limits-display">
															<div class="limit-item">
										<span>最大用户数:</span>
										<span class="limit-value">{formData.limits?.max_users || 10}</span>
									</div>
									<div class="limit-item">
										<span>最大项目数:</span>
										<span class="limit-value">{formData.limits?.max_projects || 5}</span>
									</div>
									<div class="limit-item">
										<span>存储空间:</span>
										<span class="limit-value">{formData.limits?.max_storage_mb || 1024} MB</span>
									</div>
									<div class="limit-item">
										<span>API请求/天:</span>
										<span class="limit-value">{(formData.limits?.max_api_requests_per_day || 10000).toLocaleString()}</span>
									</div>
								</div>
							</div>
						</div>
					</form>
				</div>
				<div class="modal-footer">
					<button type="button" class="btn btn-secondary" on:click={closeCreateModal}>
						取消
					</button>
					<button type="submit" class="btn btn-primary" disabled={!formData.name.trim()}>
						创建租户
					</button>
				</div>
			</div>
		</div>
	{/if}

	<!-- Edit Tenant Modal -->
	{#if showEditModal && selectedTenant && editingTenant}
		<div class="modal-overlay" on:click={closeEditModal}>
			<div class="modal-container" on:click|stopPropagation>
				<div class="modal-header">
					<h2 class="modal-title">编辑租户</h2>
					<button class="modal-close" on:click={closeEditModal}>✕</button>
				</div>
				<div class="modal-body">
					<form on:submit|preventDefault={handleUpdateTenant}>
						<div class="form-grid">
							<div class="form-group">
								<label for="edit-tenant-name" class="form-label required">租户名称 *</label>
								<input
									id="edit-tenant-name"
									type="text"
									bind:value={editingTenant.name}
									class="form-input"
									placeholder="输入租户名称"
									required
								/>
							</div>
							<div class="form-group">
								<label for="edit-tenant-slug" class="form-label">标识符</label>
								<input
									id="edit-tenant-slug"
									type="text"
									bind:value={editingTenant.slug}
									class="form-input"
									placeholder="唯一标识符"
								/>
							</div>
							<div class="form-group">
								<label for="edit-tenant-domain" class="form-label">域名</label>
								<input
									id="edit-tenant-domain"
									type="text"
									bind:value={editingTenant.domain}
									class="form-input"
									placeholder="example.com"
								/>
							</div>
							<div class="form-group">
								<label for="edit-tenant-description" class="form-label">描述</label>
								<textarea
									id="edit-tenant-description"
									bind:value={editingTenant.description}
									class="form-input"
									rows="3"
									placeholder="描述租户的用途和特点"
								></textarea>
							</div>
						</div>
					</form>
				</div>
				<div class="modal-footer">
					<button type="button" class="btn btn-secondary" on:click={closeEditModal}>
						取消
					</button>
					<button type="submit" class="btn btn-primary" disabled={!editingTenant.name?.trim()}>
						保存更改
					</button>
				</div>
			</div>
		</div>
	{/if}

	<!-- Delete Confirmation Modal -->
	{#if showDeleteModal && selectedTenant}
		<div class="modal-overlay" on:click={closeDeleteModal}>
			<div class="modal-container modal-sm" on:click|stopPropagation>
				<div class="modal-header">
					<h2 class="modal-title">确认删除</h2>
					<button class="modal-close" on:click={closeDeleteModal}>✕</button>
				</div>
				<div class="modal-body">
					<div class="confirmation-content">
						<div class="warning-icon">
							<Trash2 size={48} />
						</div>
						<div class="confirmation-text">
							<h3>确定要删除租户「{selectedTenant.name}」吗？</h3>
							<p>此操作不可撤销，租户下的所有项目和数据将被标记为已删除。</p>
							{#if selectedTenant.projects_count > 0}
								<p class="warning-text">
									⚠️ 注意：此租户下还有 {selectedTenant.projects_count} 个项目
								</p>
							{/if}
						</div>
					</div>
				</div>
				<div class="modal-footer">
					<button type="button" class="btn btn-secondary" on:click={closeDeleteModal}>
						取消
					</button>
					<button type="button" class="btn btn-danger" on:click={handleDeleteTenant}>
						确认删除
					</button>
				</div>
			</div>
		</div>
	{/if}
</div>

<style>
	/* Page Container */
	.page-container {
		max-width: 1200px;
		margin: 0 auto;
		padding: 0;
	}

	/* Page Header */
	.page-header {
		background: white;
		border-radius: 8px;
		padding: 2rem;
		margin-bottom: 2rem;
		border: 1px solid #e5e7eb;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
	}

	.header-content {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 2rem;
	}

	.title-section {
		display: flex;
		align-items: center;
		gap: 1rem;
	}

	.title-icon {
		width: 48px;
		height: 48px;
		background: linear-gradient(135deg, #3b82f6, #2563eb);
		border-radius: 8px;
		display: flex;
		align-items: center;
		justify-content: center;
		color: white;
	}

	.title-info h1 {
		margin: 0;
		font-size: 1.875rem;
		font-weight: 600;
		color: #111827;
		line-height: 1.2;
	}

	.title-info p {
			margin: 0;
			font-size: 0.875rem;
			color: #6b7280;
		line-height: 1.5;
	}

	.actions-section {
		display: flex;
		gap: 0.75rem;
	}

	/* Stats Grid */
	.stats-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
		gap: 1.5rem;
		margin-bottom: 2rem;
	}

	.stat-card {
		background: white;
		border-radius: 8px;
		padding: 1.5rem;
		border: 1px solid #e5e7eb;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
		display: flex;
		align-items: center;
		gap: 1rem;
		transition: all 0.2s ease;
	}

	.stat-card:hover {
		transform: translateY(-2px);
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
	}

	.stat-icon {
		width: 48px;
		height: 48px;
		background: linear-gradient(135deg, #f3f4f6, #e5e7eb);
		border-radius: 8px;
		display: flex;
		align-items: center;
		justify-content: center;
		color: #6b7280;
	}

	.stat-content {
		flex: 1;
	}

	.stat-value {
		font-size: 1.5rem;
		font-weight: 600;
		color: #111827;
		line-height: 1.2;
	}

	.stat-label {
		font-size: 0.875rem;
		color: #6b7280;
	}

	/* Filters Section */
	.filters-section {
		background: white;
		border-radius: 8px;
		padding: 1.5rem;
		margin-bottom: 2rem;
		border: 1px solid #e5e7eb;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
		flex-wrap: wrap;
	}

	.filters-left {
		flex: 1;
		display: flex;
		gap: 1rem;
		min-width: 300px;
	}

	.filters-right {
		display: flex;
		gap: 1rem;
		flex-wrap: wrap;
	}

	.search-box {
		position: relative;
		flex: 1;
		max-width: 400px;
	}

	.search-icon {
			position: absolute;
			left: 0.75rem;
			top: 50%;
			transform: translateY(-50%);
			color: #6b7280;
			pointer-events: none;
		}

	.search-input {
			width: 100%;
			padding-left: 2.5rem;
			padding-right: 1rem;
		border: 1px solid #d1d5db;
			border-radius: 6px;
			background: white;
			color: #374151;
			font-size: 0.875rem;
			transition: all 0.2s ease;
		}

	.search-input:focus {
			outline: none;
			border-color: #3b82f6;
			box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
		}

	.filter-group {
			display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.filter-label {
			font-size: 0.875rem;
			color: #374151;
			white-space: nowrap;
		}

	.filter-select {
			border: 1px solid #d1d5db;
			border-radius: 6px;
			background: white;
			color: #374151;
			font-size: 0.875rem;
			padding: 0.5rem;
			transition: all 0.2s ease;
		}

	.filter-select:focus {
			outline: none;
			border-color: #3b82f6;
			box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
		}

	/* Alerts */
	.alert {
		border-radius: 6px;
		padding: 1rem;
		margin-bottom: 2rem;
		border: 1px solid transparent;
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}

	.alert-error {
		background-color: #fef2f2;
		border-color: #fca5a5;
		color: #dc2626;
	}

	.alert-icon {
			font-size: 1.25rem;
		line-height: 1;
	}

	.alert-text {
			flex: 1;
			font-size: 0.875rem;
	}

	.alert-close {
			background: transparent;
			border: none;
			font-size: 1.25rem;
			line-height: 1;
			color: #dc2626;
			cursor: pointer;
			padding: 0.25rem;
			border-radius: 4px;
			transition: all 0.2s ease;
		}

	.alert-close:hover {
			background-color: #fca5a5;
		}

	/* Loading State */
	.loading-container {
		text-align: center;
		padding: 4rem 2rem;
	}

	.loading-spinner {
		width: 48px;
		height: 48px;
		border: 3px solid #e5e7eb;
		border-top: 3px solid #3b82f6;
		border-radius: 50%;
		animation: spin 1s linear infinite;
		margin: 0 auto 1rem;
	}

	.loading-text {
			color: #6b7280;
			font-size: 0.875rem;
		margin-top: 0.5rem;
	}

	@keyframes spin {
		0% { transform: rotate(0deg); }
	100% { transform: rotate(360deg); }
	}

	/* Table Container */
	.table-container {
		background: white;
		border-radius: 8px;
		border: 1px solid #e5e7eb;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
			overflow: hidden;
	}

	.table-responsive {
			width: 100%;
			overflow-x: auto;
		}

	/* Data Table */
	.data-table {
			width: 100%;
			border-collapse: collapse;
			font-size: 0.875rem;
		}

	.data-table th {
			background: #f9fafb;
			color: #374151;
			font-weight: 600;
			text-align: left;
			padding: 1rem;
			border-bottom: 2px solid #e5e7eb;
			white-space: nowrap;
		}

	.data-table td {
			color: #374151;
			padding: 1rem;
			border-bottom: 1px solid #f3f4f6;
			vertical-align: middle;
		}

	.data-table tr:hover {
			background-color: #f9fafb;
		}

	.data-table tr.active {
			background-color: #f0f9ff;
		}

	.data-table tr.inactive {
			background-color: #fafafa;
			opacity: 0.7;
		}

	/* Tenant Info */
	.tenant-info {
			min-width: 200px;
		}

	.tenant-name {
			font-weight: 600;
			color: #111827;
			font-size: 0.875rem;
			margin-bottom: 0.25rem;
		}

	.tenant-slug {
			color: #6b7280;
			font-size: 0.75rem;
		}

	.tenant-description {
			color: #6b7280;
			font-size: 0.75rem;
			margin-top: 0.25rem;
			max-width: 200px;
			display: -webkit-box;
			-webkit-line-clamp: 2;
			-webkit-box-orient: vertical;
	overflow: hidden;
		}

	.tenant-domain {
			color: #3b82f6;
			font-size: 0.75rem;
			margin-top: 0.25rem;
		}

	/* Badges */
	.plan-badge {
			padding: 0.25rem 0.75rem;
			border-radius: 12px;
			font-size: 0.75rem;
			font-weight: 500;
			text-transform: uppercase;
		}

	.status-badge {
			padding: 0.25rem 0.75rem;
			border-radius: 12px;
			font-size: 0.75rem;
			font-weight: 500;
		}

		.status-active {
			background-color: #f0fdf4;
			color: #166534;
			border-color: #15803d;
		}

		.status-inactive {
			background-color: #fef2f2;
			color: #dc2626;
			border-color: #fca5a5;
		}

	/* Date Info */
	.date-info {
			font-size: 0.75rem;
			color: #374151;
		}

	.date-time {
			color: #6b7280;
			font-size: 0.75rem;
			margin-top: 0.125rem;
		}

	/* Action Buttons */
	.action-buttons {
			display: flex;
			gap: 0.5rem;
		}

	.btn-icon {
			padding: 0.5rem;
			background: transparent;
			border: 1px solid #e5e7eb;
			border-radius: 6px;
			cursor: pointer;
			transition: all 0.2s ease;
			color: #6b7280;
			display: flex;
			align-items: center;
			justify-content: center;
		}

		.btn-icon:hover {
			background-color: #f3f4f6;
			color: #374151;
		}

		.btn-icon.btn-danger:hover {
			background-color: #fef2f2;
			color: #dc2626;
			border-color: #fca5a5;
		}

	.btn-icon.btn-sm {
			padding: 0.375rem;
		}

	/* Pagination */
	.pagination {
			display: flex;
			justify-content: space-between;
			align-items: center;
			padding: 1rem 0;
			border-top: 1px solid #e5e7eb;
			margin-top: 2rem;
		}

	.pagination-info {
			color: #6b7280;
			font-size: 0.875rem;
		}

	.pagination-controls {
			display: flex;
			gap: 0.5rem;
		align-items: center;
		}

	.pagination-current {
			font-weight: 500;
			color: #111827;
		}

	/* Empty State */
	.empty-state {
			text-align: center;
			padding: 4rem 2rem;
			color: #6b7280;
		}

		.empty-icon {
			width: 48px;
			height: 48px;
			background: #f3f4f6;
			border-radius: 50%;
			display: flex;
			align-items: center;
			justify-content: center;
			margin: 0 auto 1rem;
			color: #9ca3af;
		}

	.empty-state h3 {
			font-size: 1.25rem;
			font-weight: 600;
			color: #374151;
			margin: 1rem 0 0.5rem;
		}

	.empty-state p {
			color: #6b7280;
			font-size: 0.875rem;
			margin-bottom: 2rem;
		}

	/* Modals */
	.modal-overlay {
			position: fixed;
			top: 0;
			left: 0;
			right: 0;
			bottom: 0;
			background: rgba(0, 0, 0, 0.5);
			z-index: 999;
			display: flex;
			align-items: center;
			justify-content: center;
			padding: 2rem;
		}

	.modal-container {
			background: white;
			border-radius: 12px;
			width: 90%;
			max-width: 500px;
			max-height: 80vh;
			overflow-y: auto;
			box-shadow: 0 10px 25px rgba(0, 0, 0, 0.15);
		}

	.modal-sm {
			max-width: 400px;
		}

	.modal-header {
			display: flex;
			align-items: center;
			justify-content: space-between;
			padding: 1.5rem;
			border-bottom: 1px solid #e5e7eb;
		}

	.modal-title {
			font-size: 1.25rem;
			font-weight: 600;
			color: #111827;
			margin: 0;
		}

	.modal-close {
			background: transparent;
			border: none;
			font-size: 1.5rem;
			color: #6b7280;
			cursor: pointer;
			padding: 0.25rem;
			border-radius: 4px;
			transition: all 0.2s ease;
		}

	.modal-close:hover {
			background-color: #f3f4f6;
			color: #374151;
		}

	.modal-body {
			padding: 1.5rem;
		}

	.modal-footer {
			display: flex;
			justify-content: flex-end;
			gap: 0.75rem;
			padding: 1.5rem;
			border-top: 1px solid #e5e7eb;
		}

	/* Form Styles */
	.form-grid {
			display: grid;
			gap: 1rem;
			grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
		}

		.form-group {
			display: flex;
			flex-direction: column;
		}

		.form-group.full-width {
			grid-column: 1 / -1;
		}

		.form-label {
			font-size: 0.875rem;
			font-weight: 500;
			color: #374151;
			margin-bottom: 0.5rem;
		}

		.form-label.required::after {
			content: ' *';
			color: #ef4444;
			margin-left: 0.25rem;
		}

	.form-input,
		.form-select,
	.form-textarea {
			padding: 0.75rem;
			border: 1px solid #d1d5db;
			border-radius: 6px;
			background: white;
			color: #374151;
			font-size: 0.875rem;
			transition: all 0.2s ease;
		}

		.form-input:focus,
		.form-select:focus,
		.form-textarea:focus {
			outline: none;
			border-color: #3b82f6;
			box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
		}

		.form-textarea {
			resize: vertical;
			min-height: 80px;
		}

	/* Limits Display */
	.limits-display {
			background: #f9fafb;
			border: 1px solid #e5e7eb;
			border-radius: 6px;
			padding: 1rem;
			margin-top: 0.5rem;
		}

	.limit-item {
			display: flex;
			justify-content: space-between;
			margin-bottom: 0.5rem;
		}

		.limit-value {
			font-weight: 600;
			color: #111827;
		}

	/* Confirmation Content */
	.confirmation-content {
			text-align: center;
		}

	.warning-icon {
			width: 48px;
			height: 48px;
			background: #fef2f2;
			border-radius: 50%;
			display: flex;
			align-items: center;
			justify-content: center;
			margin: 0 auto 1rem;
			color: #dc2626;
		}

	.confirmation-text h3 {
			margin: 0 0 1rem 0;
			font-size: 1.125rem;
			font-weight: 600;
			color: #374151;
		}

	.confirmation-text p {
			margin: 0 0 1rem 0;
			line-height: 1.5;
			color: #6b7280;
		}

		.warning-text {
			color: #dc2626;
			font-weight: 500;
			margin-top: 1rem;
		}

	/* Buttons */
	.btn {
			display: inline-flex;
			align-items: center;
			justify-content: center;
			padding: 0.75rem 1.5rem;
			border: 1px solid transparent;
			border-radius: 6px;
			font-size: 0.875rem;
			font-weight: 500;
			text-decoration: none;
			cursor: pointer;
			transition: all 0.2s ease;
			gap: 0.5rem;
		}

	.btn:hover {
			transform: translateY(-1px);
		}

	.btn:focus {
			outline: 2px solid #3b82f6;
			outline-offset: 2px;
		}

	.btn:disabled {
			opacity: 0.5;
			cursor: not-allowed;
		}

		.btn-primary {
			background-color: #3b82f6;
			color: white;
			border-color: #3b82f6;
		}

		.btn-primary:hover:not(:disabled) {
			background-color: #2563eb;
			border-color: #2563eb;
		}

		.btn-secondary {
			background-color: white;
			color: #374151;
			border-color: #d1d5db;
		}

		.btn-secondary:hover:not(:disabled) {
			background-color: #f9fafb;
			border-color: #d1d5db;
		}

		.btn-outline {
			background-color: transparent;
			color: #3b82f6;
			border-color: #3b82f6;
		}

		.btn-outline:hover:not(:disabled) {
			background-color: #f0f9ff;
		}

		.btn-danger {
			background-color: #ef4444;
			color: white;
			border-color: #ef4444;
		}

		.btn-danger:hover:not(:disabled) {
			background-color: #dc2626;
			border-color: #dc2626;
		}

	.btn-sm {
			padding: 0.5rem 1rem;
			font-size: 0.875rem;
		}

	/* Responsive Design */
	@media (max-width: 768px) {
		.header-content {
			flex-direction: column;
			gap: 1rem;
			align-items: stretch;
		}

		.actions-section {
			width: 100%;
			justify-content: flex-start;
		}

		.stats-grid {
			grid-template-columns: 1fr;
		}

		.filters-section {
			flex-direction: column;
			gap: 1rem;
		}

		.filters-left,
		.filters-right {
			width: 100%;
		}

		.search-box {
			max-width: none;
		}

		.table-responsive {
			font-size: 0.75rem;
		}

		.data-table th,
		.data-table td {
			padding: 0.75rem;
		}

		.tenant-info {
			min-width: 150px;
		}

		.action-buttons {
			flex-direction: column;
		}

		.modal-container {
			width: 95%;
			margin: 1rem;
		}

		.form-grid {
			grid-template-columns: 1fr;
		}
	}
</style>