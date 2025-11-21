<script>
	import { onMount } from 'svelte';
	import { Search, Filter, Download, RefreshCw, AlertCircle, CheckCircle, XCircle, Info } from 'lucide-svelte';

	// State
	let logs = [];
	let stats = null;
	let loading = false;
	let error = null;

	// Filters
	let searchQuery = '';
	let selectedLevel = 'ALL';
	let selectedService = 'ALL';
	let selectedComponent = 'ALL';
	let selectedTimeRange = '1h';

	// Pagination
	let currentPage = 1;
	let pageSize = 50;
	let totalLogs = 0;

	// Options
	let levels = ['DEBUG', 'INFO', 'WARN', 'ERROR', 'FATAL'];
	let services = [];
	let components = [];
	let timeRanges = [
		{ value: '15m', label: '15分钟' },
		{ value: '1h', label: '1小时' },
		{ value: '6h', label: '6小时' },
		{ value: '24h', label: '24小时' },
		{ value: '7d', label: '7天' }
	];

	// Level colors
	const levelColors = {
		DEBUG: 'text-gray-500 bg-gray-100',
		INFO: 'text-blue-600 bg-blue-100',
		WARN: 'text-yellow-600 bg-yellow-100',
		ERROR: 'text-red-600 bg-red-100',
		FATAL: 'text-purple-600 bg-purple-100'
	};

	const levelIcons = {
		DEBUG: Info,
		INFO: CheckCircle,
		WARN: AlertCircle,
		ERROR: XCircle,
		FATAL: XCircle
	};

	onMount(() => {
		loadLogs();
		loadStats();
		loadOptions();
	});

	async function loadLogs() {
		loading = true;
		error = null;

		try {
			const params = new URLSearchParams({
				limit: pageSize.toString(),
				offset: ((currentPage - 1) * pageSize).toString(),
				order_by: 'timestamp',
				order_dir: 'desc'
			});

			// Add filters
			if (searchQuery) params.append('search', searchQuery);
			if (selectedLevel !== 'ALL') params.append('levels', selectedLevel);
			if (selectedService !== 'ALL') params.append('services', selectedService);
			if (selectedComponent !== 'ALL') params.append('components', selectedComponent);

			// Add time range
			const now = new Date();
			let startTime;
			switch (selectedTimeRange) {
				case '15m':
					startTime = new Date(now.getTime() - 15 * 60 * 1000);
					break;
				case '1h':
					startTime = new Date(now.getTime() - 60 * 60 * 1000);
					break;
				case '6h':
					startTime = new Date(now.getTime() - 6 * 60 * 60 * 1000);
					break;
				case '24h':
					startTime = new Date(now.getTime() - 24 * 60 * 60 * 1000);
					break;
				case '7d':
					startTime = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000);
					break;
				default:
					startTime = new Date(now.getTime() - 60 * 60 * 1000);
			}
			params.append('start_time', startTime.toISOString());

			const response = await fetch(`/api/admin/logs?${params}`);
			if (!response.ok) {
				throw new Error(`Failed to load logs: ${response.statusText}`);
			}

			const data = await response.json();
			logs = data.logs || [];
			totalLogs = data.total || 0;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load logs';
			console.error('Error loading logs:', err);
		} finally {
			loading = false;
		}
	}

	async function loadStats() {
		try {
			const params = new URLSearchParams();
			if (selectedTimeRange !== '7d') {
				const now = new Date();
				let startTime;
				switch (selectedTimeRange) {
					case '15m':
						startTime = new Date(now.getTime() - 15 * 60 * 1000);
						break;
					case '1h':
						startTime = new Date(now.getTime() - 60 * 60 * 1000);
						break;
					case '6h':
						startTime = new Date(now.getTime() - 6 * 60 * 60 * 1000);
						break;
					case '24h':
						startTime = new Date(now.getTime() - 24 * 60 * 60 * 1000);
						break;
					default:
						startTime = new Date(now.getTime() - 60 * 60 * 1000);
				}
				params.append('time_range', startTime.toISOString());
			}

			const response = await fetch(`/api/admin/logs/stats?${params}`);
			if (!response.ok) {
				throw new Error(`Failed to load stats: ${response.statusText}`);
			}

			const data = await response.json();
			stats = data;
		} catch (err) {
			console.error('Error loading stats:', err);
		}
	}

	async function loadOptions() {
		try {
			const [servicesRes, componentsRes] = await Promise.all([
				fetch('/api/admin/logs/services'),
				fetch('/api/admin/logs/components')
			]);

			if (servicesRes.ok) {
				const servicesData = await servicesRes.json();
				services = servicesData.services || [];
			}

			if (componentsRes.ok) {
				const componentsData = await componentsRes.json();
				components = componentsData.components || [];
			}
		} catch (err) {
			console.error('Error loading options:', err);
		}
	}

	function refreshLogs() {
		currentPage = 1;
		loadLogs();
		loadStats();
	}

	function formatTimestamp(timestamp) {
		return new Date(timestamp).toLocaleString('zh-CN');
	}

	function formatDuration(ms) {
		if (!ms) return '-';
		if (ms < 1000) return `${ms.toFixed(2)}ms`;
		return `${(ms / 1000).toFixed(2)}s`;
	}

	function getStatusColor(status) {
		if (!status) return '';
		if (status >= 200 && status < 300) return 'text-green-600';
		if (status >= 300 && status < 400) return 'text-yellow-600';
		if (status >= 400 && status < 500) return 'text-orange-600';
		return 'text-red-600';
	}

	function truncateText(text, maxLength = 100) {
		if (text.length <= maxLength) return text;
		return text.substring(0, maxLength) + '...';
	}

	$: totalPages = Math.ceil(totalLogs / pageSize);
</script>

<div class="logs-page">
	<div class="page-header">
		<div class="header-content">
			<h1>系统日志</h1>
			<div class="header-actions">
				<button class="btn btn-primary" on:click={refreshLogs} disabled={loading}>
					<RefreshCw size={16} class={loading ? 'animate-spin' : ''} />
					刷新
				</button>
				<button class="btn btn-secondary">
					<Download size={16} />
					导出
				</button>
			</div>
		</div>
	</div>

	<!-- Stats Cards -->
	{#if stats}
		<div class="stats-grid">
			<div class="stat-card">
				<div class="stat-label">总日志数</div>
				<div class="stat-value">{stats.total_logs ? stats.total_logs.toLocaleString() : '0'}</div>
			</div>
			<div class="stat-card">
				<div class="stat-label">错误率</div>
				<div class="stat-value">{(stats.error_rate || 0).toFixed(2)}%</div>
			</div>
			<div class="stat-card">
				<div class="stat-label">99% 响应时间</div>
				<div class="stat-value">{formatDuration(stats.p99_response_time || stats.avg_response_time || 0)}</div>
			</div>
			<div class="stat-card">
				<div class="stat-label">活跃服务</div>
				<div class="stat-value">{stats.logs_by_service ? Object.keys(stats.logs_by_service).length : 0}</div>
			</div>
		</div>
	{/if}

	<!-- Filters -->
	<div class="filters-section">
		<div class="filters-grid">
			<div class="filter-group search-group">
				<label>搜索日志</label>
				<div class="search-input large">
					<Search size={18} />
					<input
						type="text"
						placeholder="搜索日志内容、服务、组件、路径等..."
						bind:value={searchQuery}
						on:keydown={(e) => e.key === 'Enter' && refreshLogs()}
					/>
				</div>
			</div>

			<div class="filter-group">
				<label>日志级别</label>
				<select bind:value={selectedLevel} on:change={refreshLogs}>
					<option value="ALL">全部</option>
					{#each levels as level}
						<option value={level}>{level}</option>
					{/each}
				</select>
			</div>

			<div class="filter-group">
				<label>服务</label>
				<select bind:value={selectedService} on:change={refreshLogs}>
					<option value="ALL">全部</option>
					{#each services as service}
						<option value={service}>{service}</option>
					{/each}
				</select>
			</div>

			<div class="filter-group">
				<label>组件</label>
				<select bind:value={selectedComponent} on:change={refreshLogs}>
					<option value="ALL">全部</option>
					{#each components as component}
						<option value={component}>{component}</option>
					{/each}
				</select>
			</div>

			<div class="filter-group">
				<label>时间范围</label>
				<select bind:value={selectedTimeRange} on:change={refreshLogs}>
					{#each timeRanges as range}
						<option value={range.value}>{range.label}</option>
					{/each}
				</select>
			</div>
		</div>
	</div>

	<!-- Error Message -->
	{#if error}
		<div class="error-message">
			<XCircle size={20} />
			<span>{error}</span>
		</div>
	{/if}

	<!-- Logs Table -->
	<div class="logs-container">
		<div class="table-wrapper">
			<table class="logs-table">
				<thead>
					<tr>
						<th>时间</th>
						<th>级别</th>
						<th>服务</th>
						<th>消息</th>
						<th>方法</th>
						<th>路径</th>
						<th>状态</th>
						<th>耗时</th>
						<th>用户</th>
					</tr>
				</thead>
				<tbody>
					{#each logs as log, index}
						{@const LevelIcon = levelIcons[log.level] || Info}
						<tr class="{index % 2 === 0 ? 'even' : 'odd'}">
							<td class="timestamp">{formatTimestamp(log.timestamp)}</td>
							<td class="level">
								<div class="level-badge {levelColors[log.level] || ''}">
									<LevelIcon size={14} />
									<span>{log.level}</span>
								</div>
							</td>
							<td class="service">{log.service || '-'}</td>
							<td class="message" title={log.message}>
								{truncateText(log.message, 80)}
							</td>
							<td class="method">{log.method || '-'}</td>
							<td class="path" title={log.path}>
								{truncateText(log.path || '', 50)}
							</td>
							<td class="status">
								{#if log.status}
									<span class="{getStatusColor(log.status)}">{log.status}</span>
								{:else}
									-
								{/if}
							</td>
							<td class="duration">{formatDuration(log.duration_ms)}</td>
							<td class="user">{log.user_id || '-'}</td>
						</tr>
					{/each}
				</tbody>
			</table>

			{#if loading}
				<div class="loading-state">
					<div class="spinner"></div>
					<span>加载中...</span>
				</div>
			{:else if logs.length === 0}
				<div class="empty-state">
					<Info size={48} />
					<h3>暂无日志</h3>
					<p>当前筛选条件下没有找到日志记录</p>
				</div>
			{/if}
		</div>

		<!-- Pagination -->
		{#if totalPages > 1}
			<div class="pagination">
				<div class="pagination-info">
					显示第 {(currentPage - 1) * pageSize + 1} - {Math.min(currentPage * pageSize, totalLogs)} 条，共 {totalLogs.toLocaleString()} 条
				</div>
				<div class="pagination-controls">
					<button
						class="btn btn-secondary"
						disabled={currentPage === 1 || loading}
						on:click={() => {
							currentPage--;
							loadLogs();
						}}
					>
						上一页
					</button>
					<span class="page-info">第 {currentPage} / {totalPages} 页</span>
					<button
						class="btn btn-secondary"
						disabled={currentPage === totalPages || loading}
						on:click={() => {
							currentPage++;
							loadLogs();
						}}
					>
						下一页
					</button>
				</div>
			</div>
		{/if}
	</div>
</div>

<style>
	.logs-page {
	display: flex;
	flex-direction: column;
	gap: 1.5rem;
	padding: 1.5rem;
	max-width: 1600px;
	margin: 0 auto;
	width: 100%;
}

	.page-header {
		background: white;
		border-radius: 8px;
		padding: 1.5rem;
		border: 1px solid #e5e7eb;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
	}

	.header-content {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.header-content h1 {
		margin: 0;
		font-size: 1.5rem;
		font-weight: 600;
		color: #111827;
	}

	.header-actions {
		display: flex;
		gap: 0.5rem;
	}

	.stats-grid {
		display: grid;
		grid-template-columns: repeat(4, 1fr);
		gap: 1rem;
	}

	.stat-card {
		background: white;
		border-radius: 8px;
		padding: 1.5rem;
		border: 1px solid #e5e7eb;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
		text-align: center;
	}

	.stat-label {
		font-size: 0.875rem;
		color: #6b7280;
		margin-bottom: 0.5rem;
		font-weight: 500;
	}

	.stat-value {
		font-size: 1.875rem;
		font-weight: 700;
		color: #111827;
	}

	.filters-section {
		background: white;
		border-radius: 8px;
		padding: 1.5rem;
		border: 1px solid #e5e7eb;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
	}

	.filters-grid {
		display: grid;
		grid-template-columns: 2fr 1fr 1fr 1fr;
		gap: 1rem;
		align-items: end;
	}

	.filter-group label {
		display: block;
		font-size: 0.875rem;
		font-weight: 500;
		color: #374151;
		margin-bottom: 0.5rem;
	}

	.search-input {
		position: relative;
		display: flex;
		align-items: center;
	}

	.search-input svg {
		position: absolute;
		left: 0.75rem;
		color: #6b7280;
		z-index: 1;
	}

	.search-input input {
		padding-left: 2.5rem;
		width: 100%;
		padding: 0.5rem 0.75rem 0.5rem 2.5rem;
		border: 1px solid #d1d5db;
		border-radius: 6px;
		font-size: 0.875rem;
	}

	.search-input.large {
		grid-column: span 1;
	}

	.search-input.large svg {
		left: 1rem;
		top: 50%;
		transform: translateY(-50%);
	}

	.search-input.large input {
		padding-left: 3rem;
		padding: 0.75rem 1rem 0.75rem 3rem;
		font-size: 0.9rem;
		border: 2px solid #d1d5db;
		transition: border-color 0.2s ease, box-shadow 0.2s ease;
	}

	.search-input.large input:focus {
		border-color: #3b82f6;
		box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
		outline: none;
	}

	.search-group {
		grid-column: span 2;
	}

	.search-input input:focus {
		outline: none;
		border-color: #3b82f6;
		box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
	}

	select {
		width: 100%;
		padding: 0.5rem 0.75rem;
		border: 1px solid #d1d5db;
		border-radius: 6px;
		font-size: 0.875rem;
		background: white;
	}

	select:focus {
		outline: none;
		border-color: #3b82f6;
		box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
	}

	.error-message {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 1rem;
		background: #fef2f2;
		border: 1px solid #fecaca;
		border-radius: 6px;
		color: #dc2626;
	}

	.logs-container {
	background: white;
	border-radius: 8px;
	border: 1px solid #e5e7eb;
	box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
	overflow: hidden;
	width: 100%;
}

.table-wrapper {
	overflow-x: auto;
	max-height: 70vh;
	width: 100%;
}

	.logs-table {
		width: 100%;
		border-collapse: collapse;
	}

	.logs-table th {
		background: #f9fafb;
		padding: 0.75rem 1rem;
		text-align: left;
		font-size: 0.75rem;
		font-weight: 600;
		color: #374151;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		border-bottom: 1px solid #e5e7eb;
		position: sticky;
		top: 0;
		z-index: 10;
	}

	.logs-table td {
		padding: 0.75rem 1rem;
		font-size: 0.875rem;
		border-bottom: 1px solid #f3f4f6;
		vertical-align: top;
	}

	.logs-table tr.even {
		background: #ffffff;
	}

	.logs-table tr.odd {
		background: #fafafa;
	}

	.logs-table tr:hover {
		background: #f3f4f6;
	}

	.timestamp {
		font-family: monospace;
		font-size: 0.75rem;
		color: #6b7280;
		white-space: nowrap;
		min-width: 120px;
	}

	.level-badge {
		display: inline-flex;
		align-items: center;
		gap: 0.25rem;
		padding: 0.25rem 0.5rem;
		border-radius: 9999px;
		font-size: 0.75rem;
		font-weight: 500;
	}

	.service {
		font-family: monospace;
		font-size: 0.75rem;
		color: #6b7280;
	}

	.message {
	max-width: 500px;
	word-break: break-word;
	line-height: 1.4;
	width: 35%;
}

.method {
	font-family: monospace;
	font-weight: 600;
	font-size: 0.75rem;
	width: 80px;
}

.path {
	font-family: monospace;
	font-size: 0.75rem;
	color: #6b7280;
	max-width: 350px;
	word-break: break-all;
	width: 25%;
}

	.status {
		font-family: monospace;
		font-weight: 600;
		font-size: 0.75rem;
	}

	.duration {
		font-family: monospace;
		font-size: 0.75rem;
		color: #6b7280;
		text-align: right;
	}

	.user {
		font-family: monospace;
		font-size: 0.75rem;
		color: #6b7280;
	}

	.loading-state,
	.empty-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 3rem;
		color: #6b7280;
		gap: 1rem;
	}

	.spinner {
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

	.empty-state h3 {
		margin: 0;
		font-size: 1.125rem;
		font-weight: 600;
		color: #374151;
	}

	.empty-state p {
		margin: 0.5rem 0 0 0;
		color: #6b7280;
	}

	.pagination {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 1rem 1.5rem;
		border-top: 1px solid #e5e7eb;
		background: #f9fafb;
	}

	.pagination-info {
		font-size: 0.875rem;
		color: #6b7280;
	}

	.pagination-controls {
		display: flex;
		align-items: center;
		gap: 1rem;
	}

	.page-info {
		font-size: 0.875rem;
		color: #374151;
		font-weight: 500;
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

	.btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.btn-primary {
		background-color: #3b82f6;
		color: white;
	}

	.btn-primary:hover:not(:disabled) {
		background-color: #2563eb;
	}

	.btn-secondary {
		background-color: white;
		color: #374151;
		border-color: #d1d5db;
	}

	.btn-secondary:hover:not(:disabled) {
		background-color: #f9fafb;
		border-color: #9ca3af;
	}

	/* 响应式设计 */
	@media (max-width: 1024px) {
		.filters-grid {
			grid-template-columns: 1fr 1fr;
			gap: 1rem;
		}

		.search-group {
			grid-column: span 2;
		}
	}

	@media (max-width: 768px) {
		.logs-page {
			padding: 1rem;
		}

		.header-content {
			flex-direction: column;
			gap: 1rem;
			align-items: stretch;
		}

		.stats-grid {
			grid-template-columns: repeat(2, 1fr);
		}

		.filters-grid {
			grid-template-columns: 1fr;
		}

		.search-group {
			grid-column: span 1;
		}

		.logs-table {
			font-size: 0.75rem;
		}

		.logs-table th,
		.logs-table td {
			padding: 0.5rem;
		}

		.message,
		.path {
			max-width: 150px;
		}

		.pagination {
			flex-direction: column;
			gap: 1rem;
			text-align: center;
		}
	}

	@media (max-width: 480px) {
		.stats-grid {
			grid-template-columns: 1fr;
		}

		.search-input.large input {
			padding: 0.6rem 0.8rem 0.6rem 2.5rem;
			font-size: 0.875rem;
		}
	}
</style>