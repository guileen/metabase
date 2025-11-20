<script>
	import { onMount } from 'svelte';
	import { Users, Plus, Search, Filter, MoreHorizontal, Edit, Trash2, Shield } from 'lucide-svelte';

	let users = [];
	let loading = false;
	let searchQuery = '';
	let selectedRole = 'all';
	let currentPage = 1;
	let pageSize = 10;

	onMount(async () => {
		await loadUsers();
	});

	async function loadUsers() {
		loading = true;
		try {
			// 模拟API调用
			await new Promise(resolve => setTimeout(resolve, 500));
			users = [
				{
					id: 1,
					username: 'admin',
					email: 'admin@metabase.com',
					role: '超级管理员',
					status: 'active',
					createdAt: '2024-01-15T10:30:00Z',
					lastLogin: '2024-01-20T14:22:00Z',
					tenant: 'default'
				},
				{
					id: 2,
					username: 'john_doe',
					email: 'john@example.com',
					role: '管理员',
					status: 'active',
					createdAt: '2024-01-16T09:15:00Z',
					lastLogin: '2024-01-20T12:45:00Z',
					tenant: 'tenant1'
				},
				{
					id: 3,
					username: 'jane_smith',
					email: 'jane@example.com',
					role: '用户',
					status: 'inactive',
					createdAt: '2024-01-17T11:20:00Z',
					lastLogin: '2024-01-19T16:30:00Z',
					tenant: 'tenant2'
				}
			];
		} catch (error) {
			console.error('Failed to load users:', error);
		} finally {
			loading = false;
		}
	}

	$: filteredUsers = users.filter(user => {
		const matchesSearch = user.username.toLowerCase().includes(searchQuery.toLowerCase()) ||
			user.email.toLowerCase().includes(searchQuery.toLowerCase());
		const matchesRole = selectedRole === 'all' || user.role === selectedRole;
		return matchesSearch && matchesRole;
	});

	$: paginatedUsers = filteredUsers.slice(
		(currentPage - 1) * pageSize,
		currentPage * pageSize
	);

	$: totalPages = Math.ceil(filteredUsers.length / pageSize);

	function getStatusBadge(status) {
		return status === 'active' ? 'success' : 'warning';
	}

	function getRoleBadgeClass(role) {
		switch (role) {
			case '超级管理员': return 'danger';
			case '管理员': return 'warning';
			default: return 'info';
		}
	}
</script>

<div class="users-page">
	<div class="page-header">
		<div class="header-content">
			<h1>用户管理</h1>
			<p>管理系统用户账户和权限配置</p>
		</div>
		<div class="header-actions">
			<button class="btn btn-primary">
				<Plus size={20} />
				添加用户
			</button>
		</div>
	</div>

	<div class="filters-section">
		<div class="filter-group">
			<div class="search-box">
				<Search size={20} />
				<input
					type="text"
					placeholder="搜索用户名或邮箱..."
					bind:value={searchQuery}
				/>
			</div>
			<div class="filter-select">
				<select bind:value={selectedRole}>
					<option value="all">所有角色</option>
					<option value="超级管理员">超级管理员</option>
					<option value="管理员">管理员</option>
					<option value="用户">用户</option>
				</select>
			</div>
		</div>
		<div class="filter-actions">
			<button class="btn btn-secondary">
				<Filter size={16} />
				高级筛选
			</button>
		</div>
	</div>

	{#if loading}
		<div class="loading-state">
			<div class="spinner"></div>
			<p>加载用户数据中...</p>
		</div>
	{:else}
		<div class="users-table">
			<table>
				<thead>
					<tr>
						<th>用户信息</th>
						<th>角色</th>
						<th>状态</th>
						<th>租户</th>
						<th>最后登录</th>
						<th>创建时间</th>
						<th class="text-right">操作</th>
					</tr>
				</thead>
				<tbody>
					{#each paginatedUsers as user}
						<tr>
							<td>
								<div class="user-info">
									<div class="user-avatar">
										{user.username.charAt(0).toUpperCase()}
									</div>
									<div class="user-details">
										<div class="user-name">{user.username}</div>
										<div class="user-email">{user.email}</div>
									</div>
								</div>
							</td>
							<td>
								<span class="badge badge-{getRoleBadgeClass(user.role)}">
									{user.role}
								</span>
							</td>
							<td>
								<span class="badge badge-{getStatusBadge(user.status)}">
									{user.status === 'active' ? '活跃' : '未激活'}
								</span>
							</td>
							<td>
								<span class="tenant-name">{user.tenant}</span>
							</td>
							<td>
								<div class="last-login">
									{new Date(user.lastLogin).toLocaleDateString('zh-CN')}
									<div class="time-ago">2小时前</div>
								</div>
							</td>
							<td>
								{new Date(user.createdAt).toLocaleDateString('zh-CN')}
							</td>
							<td class="text-right">
								<div class="action-menu">
									<button class="btn btn-sm btn-secondary">
										<Edit size={16} />
									</button>
									<button class="btn btn-sm btn-secondary">
										<Shield size={16} />
									</button>
									<button class="btn btn-sm btn-danger">
										<Trash2 size={16} />
									</button>
								</div>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>

		{#if totalPages > 1}
			<div class="pagination">
				<button class="btn btn-secondary" disabled={currentPage === 1}>
					上一页
				</button>
				<span class="page-info">
					第 {currentPage} 页，共 {totalPages} 页
				</span>
				<button class="btn btn-secondary" disabled={currentPage === totalPages}>
					下一页
				</button>
			</div>
		{/if}
	{/if}
</div>

<style>
	.users-page {
		display: flex;
		flex-direction: column;
		gap: 1.5rem;
	}

	.page-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
	}

	.header-content h1 {
		margin: 0 0 0.5rem 0;
		font-size: 1.875rem;
		font-weight: 700;
		color: #111827;
	}

	.header-content p {
		margin: 0;
		color: #6b7280;
		font-size: 0.875rem;
	}

	.header-actions {
		display: flex;
		gap: 0.5rem;
	}

	.filters-section {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
		padding: 1rem;
		background: white;
		border-radius: 8px;
		border: 1px solid #e5e7eb;
	}

	.filter-group {
		display: flex;
		align-items: center;
		gap: 1rem;
		flex: 1;
	}

	.search-box {
		display: flex;
		align-items: center;
		background: #f9fafb;
		border: 1px solid #e5e7eb;
		border-radius: 6px;
		padding: 0.5rem 0.75rem;
		gap: 0.5rem;
		flex: 1;
		max-width: 400px;
	}

	.search-box input {
		border: none;
		background: none;
		outline: none;
		flex: 1;
		font-size: 0.875rem;
	}

	.search-box input::placeholder {
		color: #9ca3af;
	}

	.filter-select select {
		padding: 0.5rem 0.75rem;
		border: 1px solid #e5e7eb;
		border-radius: 6px;
		background: white;
		font-size: 0.875rem;
	}

	.filter-actions {
		display: flex;
		gap: 0.5rem;
	}

	.btn {
		display: inline-flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.5rem 1rem;
		border: 1px solid transparent;
		border-radius: 6px;
		font-size: 0.875rem;
		font-weight: 500;
		text-decoration: none;
		cursor: pointer;
		transition: all 0.2s ease;
		background: none;
	}

	.btn-primary {
		background-color: #3b82f6;
		color: white;
		border-color: #3b82f6;
	}

	.btn-primary:hover {
		background-color: #2563eb;
		border-color: #2563eb;
	}

	.btn-secondary {
		background-color: white;
		color: #374151;
		border-color: #d1d5db;
	}

	.btn-secondary:hover {
		background-color: #f9fafb;
		border-color: #9ca3af;
	}

	.btn-danger {
		background-color: #ef4444;
		color: white;
		border-color: #ef4444;
	}

	.btn-danger:hover {
		background-color: #dc2626;
		border-color: #dc2626;
	}

	.btn-sm {
		padding: 0.375rem 0.75rem;
		font-size: 0.75rem;
	}

	.loading-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 3rem;
		background: white;
		border-radius: 8px;
		border: 1px solid #e5e7eb;
	}

	.spinner {
		width: 40px;
		height: 40px;
		border: 4px solid #e5e7eb;
		border-top: 4px solid #3b82f6;
		border-radius: 50%;
		animation: spin 1s linear infinite;
		margin-bottom: 1rem;
	}

	@keyframes spin {
		0% { transform: rotate(0deg); }
		100% { transform: rotate(360deg); }
	}

	.users-table {
		background: white;
		border-radius: 8px;
		border: 1px solid #e5e7eb;
		overflow: hidden;
	}

	.users-table table {
		width: 100%;
		border-collapse: collapse;
	}

	.users-table th {
		background: #f9fafb;
		padding: 0.75rem 1rem;
		text-align: left;
		font-size: 0.75rem;
		font-weight: 600;
		color: #6b7280;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		border-bottom: 1px solid #e5e7eb;
	}

	.users-table td {
		padding: 1rem;
		border-bottom: 1px solid #f3f4f6;
	}

	.users-table tr:hover {
		background-color: #f9fafb;
	}

	.user-info {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}

	.user-avatar {
		width: 40px;
		height: 40px;
		background: linear-gradient(135deg, #3b82f6, #2563eb);
		border-radius: 50%;
		display: flex;
		align-items: center;
		justify-content: center;
		color: white;
		font-weight: 600;
		font-size: 0.875rem;
	}

	.user-details {
		flex: 1;
	}

	.user-name {
		font-weight: 600;
		color: #111827;
		font-size: 0.875rem;
	}

	.user-email {
		color: #6b7280;
		font-size: 0.75rem;
	}

	.badge {
		display: inline-flex;
		align-items: center;
		padding: 0.125rem 0.5rem;
		border-radius: 9999px;
		font-size: 0.75rem;
		font-weight: 500;
	}

	.badge-success {
		background: #10b98120;
		color: #10b981;
	}

	.badge-warning {
		background: #f59e0b20;
		color: #f59e0b;
	}

	.badge-danger {
		background: #ef444420;
		color: #ef4444;
	}

	.badge-info {
		background: #3b82f620;
		color: #3b82f6;
	}

	.tenant-name {
		font-size: 0.875rem;
		color: #374151;
		font-weight: 500;
	}

	.last-login {
		display: flex;
		flex-direction: column;
		gap: 0.125rem;
	}

	.time-ago {
		font-size: 0.75rem;
		color: #6b7280;
	}

	.text-right {
		text-align: right;
	}

	.action-menu {
		display: flex;
		align-items: center;
		gap: 0.25rem;
		justify-content: flex-end;
	}

	.pagination {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 1rem;
		padding: 1rem;
		background: white;
		border-radius: 8px;
		border: 1px solid #e5e7eb;
	}

	.page-info {
		color: #6b7280;
		font-size: 0.875rem;
	}

	@media (max-width: 768px) {
		.page-header {
			flex-direction: column;
			align-items: flex-start;
		}

		.filters-section {
			flex-direction: column;
			align-items: stretch;
		}

		.filter-group {
			flex-direction: column;
		}

		.search-box {
			max-width: none;
		}

		.users-table {
			overflow-x: auto;
		}

		.action-menu {
			flex-direction: column;
		}
	}
</style>