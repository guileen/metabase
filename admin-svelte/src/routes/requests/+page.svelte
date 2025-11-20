<script>
	import { onMount } from 'svelte';
	import { Search, Filter, Download, Calendar, Clock, Globe, AlertCircle, CheckCircle, XCircle } from 'lucide-svelte';

	// 搜索和过滤状态
	let searchQuery = '';
	let selectedMethod = 'all';
	let selectedStatus = 'all';
	let dateRange = 'today';

	// 分页状态
	let currentPage = 1;
	let pageSize = 20;
	let totalItems = 0;

	// 请求日志数据
	let requests = [];
	let loading = false;

	// 模拟数据生成
	function generateMockData() {
		const methods = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH'];
		const statuses = [200, 201, 400, 401, 404, 500, 502];
		const paths = ['/api/users', '/api/orders', '/api/products', '/admin/dashboard', '/api/auth/login', '/api/data/query'];

		const data = [];
		for (let i = 0; i < 150; i++) {
			const status = statuses[Math.floor(Math.random() * statuses.length)];
			const method = methods[Math.floor(Math.random() * methods.length)];
			const path = paths[Math.floor(Math.random() * paths.length)];
			const timestamp = new Date(Date.now() - Math.random() * 86400000); // 最近24小时

			data.push({
				id: `req_${i + 1}`,
				timestamp,
				method,
				path,
				status,
				responseTime: Math.floor(Math.random() * 1000) + 50,
				userAgent: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36',
				ip: `192.168.1.${Math.floor(Math.random() * 255)}`,
				user: Math.random() > 0.3 ? `user${Math.floor(Math.random() * 100)}` : null,
				size: Math.floor(Math.random() * 10000) + 100
			});
		}
		return data.sort((a, b) => b.timestamp - a.timestamp);
	}

	// 过滤数据
	$: filteredRequests = requests.filter(request => {
		const matchesSearch = !searchQuery ||
			request.path.toLowerCase().includes(searchQuery.toLowerCase()) ||
			request.method.toLowerCase().includes(searchQuery.toLowerCase()) ||
			(request.ip && request.ip.includes(searchQuery));

		const matchesMethod = selectedMethod === 'all' || request.method === selectedMethod;
		const matchesStatus = selectedStatus === 'all' ||
			(selectedStatus === 'success' && request.status < 400) ||
			(selectedStatus === 'error' && request.status >= 400) ||
			(request.status.toString() === selectedStatus);

		return matchesSearch && matchesMethod && matchesStatus;
	});

	// 分页数据
	$: totalPages = Math.ceil(filteredRequests.length / pageSize);
	$: paginatedRequests = filteredRequests.slice(
		(currentPage - 1) * pageSize,
		currentPage * pageSize
	);

	onMount(() => {
		loadRequests();
	});

	function loadRequests() {
		loading = true;
		setTimeout(() => {
			requests = generateMockData();
			totalItems = requests.length;
			loading = false;
		}, 500);
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

	function formatTime(date) {
		return date.toLocaleTimeString('zh-CN', {
			hour: '2-digit',
			minute: '2-digit',
			second: '2-digit'
		});
	}

	function formatDate(date) {
		return date.toLocaleDateString('zh-CN');
	}

	function formatResponseTime(ms) {
		if (ms < 100) return `${ms}ms`;
		return `${(ms / 1000).toFixed(2)}s`;
	}

	function formatSize(bytes) {
		if (bytes < 1024) return `${bytes}B`;
		if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)}KB`;
		return `${(bytes / 1024 / 1024).toFixed(1)}MB`;
	}

	function exportLogs() {
		const csv = [
			['时间', '方法', '路径', '状态', '响应时间', 'IP地址', '用户', '大小'],
			...filteredRequests.map(req => [
				req.timestamp.toISOString(),
				req.method,
				req.path,
				req.status,
				req.responseTime,
				req.ip,
				req.user || '',
				req.size
			])
		].map(row => row.join(',')).join('\n');

		const blob = new Blob([csv], { type: 'text/csv' });
		const url = URL.createObjectURL(blob);
		const a = document.createElement('a');
		a.href = url;
		a.download = `requests_${new Date().toISOString().split('T')[0]}.csv`;
		a.click();
		URL.revokeObjectURL(url);
	}

	function resetFilters() {
		searchQuery = '';
		selectedMethod = 'all';
		selectedStatus = 'all';
		dateRange = 'today';
		currentPage = 1;
	}
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
					{requests.filter(r => r.status < 400).length}
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
					{requests.filter(r => r.status >= 400).length}
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
					{Math.round(requests.reduce((sum, r) => sum + r.responseTime, 0) / requests.length)}ms
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
					{filteredRequests.length}
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
				<p>加载请求日志中...</p>
			</div>
		{:else}
			<div class="table-wrapper">
				<table class="requests-table">
					<thead>
						<tr>
							<th>时间</th>
							<th>方法</th>
							<th>路径</th>
							<th>状态</th>
							<th>响应时间</th>
							<th>大小</th>
							<th>IP地址</th>
							<th>用户</th>
						</tr>
					</thead>
					<tbody>
						{#each paginatedRequests as request}
							<tr class:status-success={request.status < 400} class:status-error={request.status >= 400}>
								<td>
									<div class="time-cell">
										<div class="time">{formatTime(request.timestamp)}</div>
										<div class="date">{formatDate(request.timestamp)}</div>
									</div>
								</td>
								<td>
									<span class="method-badge method-{request.method.toLowerCase()}">
										{request.method}
									</span>
								</td>
								<td class="path-cell">{request.path}</td>
								<td>
									<div class="status-cell">
										<svelte:component this={getStatusIcon(request.status)} size={16} class="status-icon {getStatusClass(request.status)}" />
										<span class="status-text">{request.status}</span>
									</div>
								</td>
								<td>{formatResponseTime(request.responseTime)}</td>
								<td>{formatSize(request.size)}</td>
								<td>{request.ip}</td>
								<td>
									{#if request.user}
										<span class="user-badge">{request.user}</span>
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
					显示 {((currentPage - 1) * pageSize) + 1}-{Math.min(currentPage * pageSize, filteredRequests.length)}
					共 {filteredRequests.length} 条记录
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