<script>
	import { onMount } from 'svelte';
	import { Search, Filter, Download, Calendar, Clock, Globe, AlertCircle, CheckCircle, XCircle } from 'lucide-svelte';

	// 搜索和过滤状态
	let searchQuery = '';
	let selectedMethod = 'all';
	let selectedStatus = 'all';
	let selectedService = 'all';
	let selectedComponent = 'all';
	let dateRange = 'today';

	// 分页状态
	let currentPage = 1;
	let pageSize = 20;
	let totalItems = 0;

	// 请求日志数据
	let requests = [];
	let loading = false;
	let error = null;

	// 可用的服务和组件
	let services = ['api', 'admin', 'gateway'];
	let components = [];

	// 统计数据
	let stats = null;

	// 构建查询参数
	function buildQueryParams() {
		const params = new URLSearchParams();

		// 搜索
		if (searchQuery) params.append('search', searchQuery);

		// 过滤器
		if (selectedMethod !== 'all') params.append('method', selectedMethod);
		if (selectedStatus !== 'all') {
			if (selectedStatus === 'success') {
				params.append('max_status', '399');
			} else if (selectedStatus === 'error') {
				params.append('min_status', '400');
			} else {
				params.append('min_status', selectedStatus);
				params.append('max_status', selectedStatus);
			}
		}
		if (selectedService !== 'all') params.append('services', selectedService);
		if (selectedComponent !== 'all') params.append('components', selectedComponent);

		// 时间范围
		const now = new Date();
		if (dateRange === 'today') {
			const startOfDay = new Date(now.getFullYear(), now.getMonth(), now.getDate());
			params.append('start_time', startOfDay.toISOString());
		} else if (dateRange === 'yesterday') {
			const yesterday = new Date(now.getTime() - 24 * 60 * 60 * 1000);
			const startOfYesterday = new Date(yesterday.getFullYear(), yesterday.getMonth(), yesterday.getDate());
			const endOfYesterday = new Date(yesterday.getFullYear(), yesterday.getMonth(), yesterday.getDate() + 1);
			params.append('start_time', startOfYesterday.toISOString());
			params.append('end_time', endOfYesterday.toISOString());
		} else if (dateRange === 'week') {
			const weekAgo = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000);
			params.append('start_time', weekAgo.toISOString());
		} else if (dateRange === 'month') {
			const monthAgo = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000);
			params.append('start_time', monthAgo.toISOString());
		}

		// 分页
		params.append('limit', pageSize.toString());
		params.append('offset', ((currentPage - 1) * pageSize).toString());

		// 排序
		params.append('order_by', 'timestamp');
		params.append('order_dir', 'desc');

		return params.toString();
	}

	onMount(() => {
		loadServices();
		loadComponents();
		loadStats();
		loadRequests();
	});

	async function loadServices() {
		try {
			const response = await fetch('/admin/logs/services');
			if (response.ok) {
				const data = await response.json();
				services = ['all', ...data.services];
			}
		} catch (err) {
			console.error('Failed to load services:', err);
		}
	}

	async function loadComponents() {
		try {
			const response = await fetch('/admin/logs/components');
			if (response.ok) {
				const data = await response.json();
				components = ['all', ...data.components];
			}
		} catch (err) {
			console.error('Failed to load components:', err);
		}
	}

	async function loadStats() {
		try {
			const response = await fetch('/admin/logs/stats');
			if (response.ok) {
				stats = await response.json();
			}
		} catch (err) {
			console.error('Failed to load stats:', err);
		}
	}

	async function loadRequests() {
		loading = true;
		error = null;

		try {
			const queryParams = buildQueryParams();
			const response = await fetch(`/admin/logs?${queryParams}`);

			if (!response.ok) {
				throw new Error(`HTTP ${response.status}: ${response.statusText}`);
			}

			const data = await response.json();
			requests = data.logs || [];
			totalItems = data.total || 0;
		} catch (err) {
			console.error('Failed to load requests:', err);
			error = err.message;
			requests = [];
			totalItems = 0;
		} finally {
			loading = false;
		}
	}

	function getStatusIcon(status) {
		if (status < 400) return CheckCircle;
		if (status < 500) return AlertCircle;
		return XCircle;
	}

	function getStatusClass(status) {
		if (status < 400) return 'status-success';
		if (status < 500) return 'status-warning';
		return 'status-error';
	}

	function formatTime(timestamp) {
		const date = new Date(timestamp);
		return date.toLocaleTimeString('zh-CN', {
			hour: '2-digit',
			minute: '2-digit',
			second: '2-digit'
		});
	}

	function formatDate(timestamp) {
		const date = new Date(timestamp);
		return date.toLocaleDateString('zh-CN');
	}

	function formatResponseTime(ms) {
		if (!ms) return '-';
		if (ms < 100) return `${ms}ms`;
		return `${(ms / 1000).toFixed(2)}s`;
	}

	function getMethodColorClass(method) {
		const methodColors = {
			'GET': 'method-get',
			'POST': 'method-post',
			'PUT': 'method-put',
			'DELETE': 'method-delete',
			'PATCH': 'method-patch'
		};
		return methodColors[method] || 'method-default';
	}

	// 当任何过滤条件改变时重新加载数据
	$: {
		if (typeof window !== 'undefined') {
			loadRequests();
		}
	}

	// 当页面改变时重新加载数据
	$: currentPage, loadRequests();

	function exportLogs() {
		const csv = [
			['时间', '级别', '消息', '服务', '组件', '方法', '路径', '状态', '响应时间', 'IP地址', '用户ID', '请求ID'],
			...requests.map(req => [
				req.timestamp,
				req.level,
				req.message,
				req.service || '',
				req.component || '',
				req.method || '',
				req.path || '',
				req.status || '',
				req.duration_ms || '',
				req.remote_addr || '',
				req.user_id || '',
				req.request_id || ''
			])
		].map(row => row.join(',')).join('\n');

		const blob = new Blob([csv], { type: 'text/csv' });
		const url = URL.createObjectURL(blob);
		const a = document.createElement('a');
		a.href = url;
		a.download = `logs_${new Date().toISOString().split('T')[0]}.csv`;
		a.click();
		URL.revokeObjectURL(url);
	}

	function resetFilters() {
		searchQuery = '';
		selectedMethod = 'all';
		selectedStatus = 'all';
		selectedService = 'all';
		selectedComponent = 'all';
		dateRange = 'today';
		currentPage = 1;
	}

	// 计算总页数
	$: totalPages = Math.ceil(totalItems / pageSize);
</script>

<div class="requests-page">
	<!-- 页面标题 -->
	<div class="page-header">
		<div class="header-left">
			<h1>请求日志</h1>
			<p class="page-description">HTTP请求记录和搜索分析</p>
		</div>
		<div class="header-right">
			<button class="btn btn-primary" on:click={exportLogs}>
				<Download size={16} />
				导出日志
			</button>
		</div>
	</div>

	<!-- 搜索和过滤器 -->
	<div class="filters-section">
		<div class="search-bar">
			<div class="search-input">
				<Search size={18} class="search-icon" />
				<input
					type="text"
					placeholder="搜索路径、方法、IP地址..."
					bind:value={searchQuery}
				/>
			</div>
		</div>

		<div class="filters-grid">
			<div class="filter-item">
				<label>服务</label>
				<select bind:value={selectedService}>
					{#each services as service}
						<option value={service}>{service}</option>
					{/each}
				</select>
			</div>

			<div class="filter-item">
				<label>组件</label>
				<select bind:value={selectedComponent}>
					{#each components as component}
						<option value={component}>{component}</option>
					{/each}
				</select>
			</div>

			<div class="filter-item">
				<label>请求方法</label>
				<select bind:value={selectedMethod}>
					<option value="all">全部</option>
					<option value="GET">GET</option>
					<option value="POST">POST</option>
					<option value="PUT">PUT</option>
					<option value="DELETE">DELETE</option>
					<option value="PATCH">PATCH</option>
				</select>
			</div>

			<div class="filter-item">
				<label>响应状态</label>
				<select bind:value={selectedStatus}>
					<option value="all">全部</option>
					<option value="success">成功 (2xx-3xx)</option>
					<option value="error">错误 (4xx-5xx)</option>
					<option value="200">200 OK</option>
					<option value="400">400 Bad Request</option>
					<option value="404">404 Not Found</option>
					<option value="500">500 Server Error</option>
				</select>
			</div>

			<div class="filter-item">
				<label>时间范围</label>
				<select bind:value={dateRange}>
					<option value="today">今天</option>
					<option value="yesterday">昨天</option>
					<option value="week">最近7天</option>
					<option value="month">最近30天</option>
				</select>
			</div>

			<div class="filter-item">
				<button class="btn btn-secondary" on:click={resetFilters}>
					重置过滤
				</button>
			</div>
		</div>
	</div>

	<!-- 统计信息 -->
	<div class="stats-section">
		<div class="stat-card">
			<div class="stat-icon success">
				<CheckCircle size={20} />
			</div>
			<div class="stat-content">
				<div class="stat-value">
					{stats?.logs_by_level?.INFO || 0}
				</div>
				<div class="stat-label">成功请求</div>
			</div>
		</div>

		<div class="stat-card">
			<div class="stat-icon error">
				<XCircle size={20} />
			</div>
			<div class="stat-content">
				<div class="stat-value">
					{(stats?.logs_by_level?.ERROR || 0) + (stats?.logs_by_level?.FATAL || 0)}
				</div>
				<div class="stat-label">错误请求</div>
			</div>
		</div>

		<div class="stat-card">
			<div class="stat-icon info">
				<Clock size={20} />
			</div>
			<div class="stat-content">
				<div class="stat-value">
					{Math.round(stats?.avg_response_time || 0)}ms
				</div>
				<div class="stat-label">平均响应时间</div>
			</div>
		</div>

		<div class="stat-card">
			<div class="stat-icon warning">
				<Globe size={20} />
			</div>
			<div class="stat-content">
				<div class="stat-value">
					{totalItems}
				</div>
				<div class="stat-label">筛选结果</div>
			</div>
		</div>
	</div>

	<!-- 请求表格 -->
	<div class="table-section">
		{#if loading}
			<div class="loading-state">
				<div class="loading-spinner"></div>
				<p>加载日志中...</p>
			</div>
		{:else if error}
			<div class="error-state">
				<AlertCircle size={32} />
				<p>加载日志失败: {error}</p>
				<button class="btn btn-primary" on:click={loadRequests}>重试</button>
			</div>
		{:else if requests.length === 0}
			<div class="empty-state">
				<Search size={32} />
				<p>没有找到日志记录</p>
				<button class="btn btn-secondary" on:click={resetFilters}>重置过滤条件</button>
			</div>
		{:else}
			<div class="table-wrapper">
				<table class="requests-table">
					<thead>
						<tr>
							<th>时间</th>
							<th>级别</th>
							<th>消息</th>
							<th>服务</th>
							<th>组件</th>
							<th>方法</th>
							<th>路径</th>
							<th>状态</th>
							<th>响应时间</th>
							<th>IP地址</th>
							<th>用户ID</th>
						</tr>
					</thead>
					<tbody>
						{#each requests as request}
							<tr class:status-success={request.status < 400} class:status-error={request.status >= 400}>
								<td>
									<div class="time-cell">
										<div class="time">{formatTime(request.timestamp)}</div>
										<div class="date">{formatDate(request.timestamp)}</div>
									</div>
								</td>
								<td>
									<span class="level-badge level-{request.level.toLowerCase()}">
										{request.level}
									</span>
								</td>
								<td class="message-cell" title={request.message}>{request.message}</td>
								<td>
									{#if request.service}
										<span class="service-badge">{request.service}</span>
									{:else}
										<span class="text-gray">-</span>
									{/if}
								</td>
								<td>
									{#if request.component}
										<span class="component-badge">{request.component}</span>
									{:else}
										<span class="text-gray">-</span>
									{/if}
								</td>
								<td>
									{#if request.method}
										<span class="method-badge {getMethodColorClass(request.method)}">
											{request.method}
										</span>
									{:else}
										<span class="text-gray">-</span>
									{/if}
								</td>
								<td class="path-cell">{request.path || '-'}</td>
								<td>
									{#if request.status > 0}
										<div class="status-cell">
											<svelte:component this={getStatusIcon(request.status)} size={16} class="status-icon {getStatusClass(request.status)}" />
											<span class="status-text">{request.status}</span>
										</div>
									{:else}
										<span class="text-gray">-</span>
									{/if}
								</td>
								<td>{formatResponseTime(request.duration_ms)}</td>
								<td>{request.remote_addr || '-'}</td>
								<td>
									{#if request.user_id}
										<span class="user-badge">{request.user_id}</span>
									{:else}
										<span class="text-gray">-</span>
									{/if}
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>

			<!-- 分页控制 -->
			<div class="pagination">
				<div class="pagination-info">
					显示 {((currentPage - 1) * pageSize) + 1}-{Math.min(currentPage * pageSize, totalItems)}
					共 {totalItems} 条记录
				</div>
				<div class="pagination-controls">
					<button
						class="btn btn-secondary btn-sm"
						disabled={currentPage === 1}
						on:click={() => currentPage = 1}
					>
						首页
					</button>
					<button
						class="btn btn-secondary btn-sm"
						disabled={currentPage === 1}
						on:click={() => currentPage -= 1}
					>
						上一页
					</button>
					<span class="page-info">
						第 {currentPage} / {totalPages} 页
					</span>
					<button
						class="btn btn-secondary btn-sm"
						disabled={currentPage === totalPages}
						on:click={() => currentPage += 1}
					>
						下一页
					</button>
					<button
						class="btn btn-secondary btn-sm"
						disabled={currentPage === totalPages}
						on:click={() => currentPage = totalPages}
					>
						末页
					</button>
				</div>
			</div>
		{/if}
	</div>
</div>

<style>
	.requests-page {
		display: flex;
		flex-direction: column;
		gap: 1.5rem;
	}

	.page-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.header-left h1 {
		margin: 0;
		font-size: 1.875rem;
		font-weight: 700;
		color: #111827;
	}

	.page-description {
		margin: 0.5rem 0 0 0;
		color: #6b7280;
		font-size: 0.875rem;
	}

	.filters-section {
		background: white;
		border-radius: 8px;
		border: 1px solid #e5e7eb;
		padding: 1.5rem;
	}

	.search-bar {
		margin-bottom: 1rem;
	}

	.search-input {
		position: relative;
		max-width: 400px;
	}

	.search-icon {
		position: absolute;
		left: 0.75rem;
		top: 50%;
		transform: translateY(-50%);
		color: #9ca3af;
	}

	.search-input input {
		width: 100%;
		padding: 0.75rem 1rem 0.75rem 2.5rem;
		border: 1px solid #d1d5db;
		border-radius: 6px;
		font-size: 0.875rem;
		transition: border-color 0.2s ease;
	}

	.search-input input:focus {
		outline: none;
		border-color: #3b82f6;
		box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
	}

	.filters-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
		gap: 1rem;
		align-items: end;
	}

	.filter-item label {
		display: block;
		font-size: 0.875rem;
		font-weight: 500;
		color: #374151;
		margin-bottom: 0.5rem;
	}

	.filter-item select {
		width: 100%;
		padding: 0.5rem;
		border: 1px solid #d1d5db;
		border-radius: 6px;
		font-size: 0.875rem;
		background: white;
	}

	.stats-section {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
		gap: 1rem;
	}

	.stat-card {
		background: white;
		border: 1px solid #e5e7eb;
		border-radius: 8px;
		padding: 1.5rem;
		display: flex;
		align-items: center;
		gap: 1rem;
	}

	.stat-icon {
		width: 40px;
		height: 40px;
		border-radius: 8px;
		display: flex;
		align-items: center;
		justify-content: center;
		color: white;
	}

	.stat-icon.success { background: #10b981; }
	.stat-icon.error { background: #ef4444; }
	.stat-icon.info { background: #3b82f6; }
	.stat-icon.warning { background: #f59e0b; }

	.stat-value {
		font-size: 1.5rem;
		font-weight: 700;
		color: #111827;
		line-height: 1;
	}

	.stat-label {
		font-size: 0.875rem;
		color: #6b7280;
		margin-top: 0.25rem;
	}

	.table-section {
		background: white;
		border: 1px solid #e5e7eb;
		border-radius: 8px;
		overflow: hidden;
	}

	.loading-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 4rem 2rem;
		color: #6b7280;
	}

	.loading-spinner {
		width: 32px;
		height: 32px;
		border: 3px solid #e5e7eb;
		border-top: 3px solid #3b82f6;
		border-radius: 50%;
		animation: spin 1s linear infinite;
		margin-bottom: 1rem;
	}

	@keyframes spin {
		0% { transform: rotate(0deg); }
		100% { transform: rotate(360deg); }
	}

	.table-wrapper {
		overflow-x: auto;
	}

	.requests-table {
		width: 100%;
		border-collapse: collapse;
	}

	.requests-table th {
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

	.requests-table td {
		padding: 0.75rem 1rem;
		border-bottom: 1px solid #f3f4f6;
		font-size: 0.875rem;
	}

	.requests-table tr:hover {
		background: #f9fafb;
	}

	.time-cell {
		line-height: 1.4;
	}

	.time {
		font-weight: 500;
		color: #111827;
	}

	.date {
		font-size: 0.75rem;
		color: #6b7280;
	}

	.method-badge {
		display: inline-flex;
		align-items: center;
		padding: 0.25rem 0.5rem;
		border-radius: 4px;
		font-size: 0.75rem;
		font-weight: 600;
		text-transform: uppercase;
	}

	.method-get { background: #dbeafe; color: #1d4ed8; }
	.method-post { background: #dcfce7; color: #166534; }
	.method-put { background: #fef3c7; color: #92400e; }
	.method-delete { background: #fee2e2; color: #991b1b; }
	.method-patch { background: #e0e7ff; color: #3730a3; }

	.level-badge {
		display: inline-flex;
		align-items: center;
		padding: 0.25rem 0.5rem;
		border-radius: 4px;
		font-size: 0.75rem;
		font-weight: 600;
		text-transform: uppercase;
	}

	.level-debug { background: #f3f4f6; color: #6b7280; }
	.level-info { background: #dbeafe; color: #1d4ed8; }
	.level-warn { background: #fef3c7; color: #92400e; }
	.level-error { background: #fee2e2; color: #991b1b; }
	.level-fatal { background: #991b1b; color: white; }

	.service-badge {
		display: inline-flex;
		align-items: center;
		padding: 0.125rem 0.5rem;
		background: #f0f9ff;
		color: #0369a1;
		border-radius: 999px;
		font-size: 0.75rem;
		font-weight: 500;
	}

	.component-badge {
		display: inline-flex;
		align-items: center;
		padding: 0.125rem 0.5rem;
		background: #fdf4ff;
		color: #a21caf;
		border-radius: 999px;
		font-size: 0.75rem;
		font-weight: 500;
	}

	.message-cell {
		max-width: 200px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.error-state, .empty-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 4rem 2rem;
		color: #6b7280;
		text-align: center;
	}

	.error-state, .empty-state {
		margin-bottom: 1rem;
	}

	.error-state p, .empty-state p {
		margin: 0.5rem 0;
	}

	.error-state svg, .empty-state svg {
		color: #9ca3af;
		margin-bottom: 1rem;
	}

	.path-cell {
		font-family: 'Monaco', 'Menlo', monospace;
		font-size: 0.8rem;
		color: #374151;
	}

	.status-cell {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.status-icon.status-success { color: #10b981; }
	.status-icon.status-warning { color: #f59e0b; }
	.status-icon.status-error { color: #ef4444; }

	.status-text {
		font-weight: 500;
	}

	.user-badge {
		display: inline-flex;
		align-items: center;
		padding: 0.125rem 0.5rem;
		background: #f3f4f6;
		color: #374151;
		border-radius: 999px;
		font-size: 0.75rem;
		font-weight: 500;
	}

	.text-gray {
		color: #9ca3af;
	}

	.pagination {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 1rem 1.5rem;
		border-top: 1px solid #e5e7eb;
	}

	.pagination-info {
		font-size: 0.875rem;
		color: #6b7280;
	}

	.pagination-controls {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.page-info {
		padding: 0 1rem;
		font-size: 0.875rem;
		color: #6b7280;
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
	}

	.btn-primary:hover {
		background-color: #2563eb;
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

	.btn-secondary:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.btn-sm {
		padding: 0.375rem 0.75rem;
		font-size: 0.75rem;
	}

	/* 响应式设计 */
	@media (max-width: 768px) {
		.page-header {
			flex-direction: column;
			align-items: flex-start;
			gap: 1rem;
		}

		.stats-section {
			grid-template-columns: repeat(2, 1fr);
		}

		.filters-grid {
			grid-template-columns: 1fr;
		}

		.pagination {
			flex-direction: column;
			gap: 1rem;
			align-items: stretch;
		}

		.pagination-controls {
			justify-content: center;
		}
	}
</style>