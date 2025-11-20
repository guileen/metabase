<script>
	import { onMount } from 'svelte';
	import { TrendingUp, TrendingDown, Users, Activity, Clock, Cpu } from 'lucide-svelte';

	let metrics = {
		totalRequests: 12543,
		totalUsers: 892,
		errorRate: 0.12,
		avgResponseTime: 142,
		qps: 45.2,
		uptime: 99.9
	};

	let systemStats = {
		cpu: 45,
		memory: 67,
		disk: 23,
		network: 12
	};

	$: qpsFormatted = metrics.qps.toFixed(1);
	$: responseTimeFormatted = metrics.avgResponseTime + 'ms';
	$: errorRateFormatted = metrics.errorRate + '%';
</script>

<div class="dashboard">
	<div class="metrics-grid">
		<div class="metric-card">
		<div class="metric-header">
			<h3 class="metric-title">总请求数</h3>
			<div class="metric-icon success">
				<Activity size={24} />
			</div>
		</div>
		<div class="metric-value">{metrics.totalRequests.toLocaleString()}</div>
		<div class="metric-change positive">
			<TrendingUp size={16} />
			<span>+12.5%</span>
			<span>较昨日</span>
		</div>
	</div>

		<div class="metric-card">
		<div class="metric-header">
			<h3 class="metric-title">活跃用户</h3>
			<div class="metric-icon info">
				<Users size={24} />
			</div>
		</div>
		<div class="metric-value">{metrics.totalUsers.toLocaleString()}</div>
		<div class="metric-change positive">
			<TrendingUp size={16} />
			<span>+8.2%</span>
			<span>较昨日</span>
		</div>
	</div>

		<div class="metric-card">
		<div class="metric-header">
			<h3 class="metric-title">错误率</h3>
			<div class="metric-icon warning">
				<Activity size={24} />
			</div>
		</div>
		<div class="metric-value">{errorRateFormatted}</div>
		<div class="metric-change negative">
			<TrendingDown size={16} />
			<span>-0.3%</span>
			<span>较昨日</span>
		</div>
	</div>

		<div class="metric-card">
		<div class="metric-header">
			<h3 class="metric-title">平均响应时间</h3>
			<div class="metric-icon primary">
				<Clock size={24} />
			</div>
		</div>
		<div class="metric-value">{responseTimeFormatted}</div>
		<div class="metric-change positive">
			<TrendingDown size={16} />
			<span>-15ms</span>
			<span>较昨日</span>
		</div>
	</div>
	</div>

	<div class="charts-grid">
		<div class="chart-card">
			<div class="card-header">
				<h3>请求趋势</h3>
				<button class="btn btn-secondary btn-sm">导出</button>
			</div>
			<div class="card-body">
				<div class="chart-placeholder">
					<p>📊 请求量趋势图</p>
					<p class="text-sm text-gray-500">实时显示最近24小时的请求量变化</p>
				</div>
			</div>
		</div>

		<div class="chart-card">
			<div class="card-header">
				<h3>系统性能</h3>
				<button class="btn btn-secondary btn-sm">刷新</button>
			</div>
			<div class="card-body">
				<div class="performance-grid">
					<div class="performance-item">
						<div class="performance-label">CPU 使用率</div>
						<div class="performance-value">{systemStats.cpu}%</div>
						<div class="progress-bar">
							<div class="progress-fill" style="width: {systemStats.cpu}%"></div>
						</div>
					</div>

					<div class="performance-item">
						<div class="performance-label">内存使用率</div>
						<div class="performance-value">{systemStats.memory}%</div>
						<div class="progress-bar">
							<div class="progress-fill" style="width: {systemStats.memory}%"></div>
						</div>
					</div>

					<div class="performance-item">
						<div class="performance-label">磁盘使用率</div>
						<div class="performance-value">{systemStats.disk}%</div>
						<div class="progress-bar">
							<div class="progress-fill" style="width: {systemStats.disk}%"></div>
						</div>
					</div>

					<div class="performance-item">
						<div class="performance-label">网络使用率</div>
						<div class="performance-value">{systemStats.network}%</div>
						<div class="progress-bar">
							<div class="progress-fill" style="width: {systemStats.network}%"></div>
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>

	<div class="info-card">
		<div class="card-header">
			<h3>系统信息</h3>
		</div>
		<div class="card-body">
			<div class="info-grid">
				<div class="info-item">
					<div class="info-label">版本</div>
					<div class="info-value">v1.0.0</div>
				</div>
				<div class="info-item">
					<div class="info-label">运行时间</div>
					<div class="info-value">15天</div>
				</div>
				<div class="info-item">
					<div class="info-label">数据库</div>
					<div class="info-value">5个</div>
				</div>
				<div class="info-item">
					<div class="info-label">数据表</div>
					<div class="info-value">42个</div>
				</div>
				<div class="info-item">
					<div class="info-label">活跃连接</div>
					<div class="info-value">23</div>
				</div>
				<div class="info-item">
					<div class="info-label">系统状态</div>
					<div class="info-value">
						<span class="status-badge status-success">正常</span>
					</div>
				</div>
				<div class="info-item">
					<div class="info-label">最后更新</div>
					<div class="info-value">刚刚</div>
				</div>
			</div>
		</div>
	</div>
</div>

<style>
	.dashboard {
		display: flex;
		flex-direction: column;
		gap: 1.5rem;
	}

	.metrics-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
		gap: 1rem;
	}

	.charts-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
		gap: 1rem;
	}

	.metric-card {
		background: white;
		border-radius: 8px;
		padding: 1.5rem;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
		border: 1px solid #e5e7eb;
		transition: transform 0.2s ease, box-shadow 0.2s ease;
	}

	.metric-card:hover {
		transform: translateY(-2px);
		box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
	}

	.metric-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 1rem;
	}

	.metric-title {
		font-size: 0.875rem;
		color: #6b7280;
		margin: 0;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}

	.metric-icon {
		width: 40px;
		height: 40px;
		border-radius: 8px;
		display: flex;
		align-items: center;
		justify-content: center;
		color: white;
	}

	.metric-icon.success {
		background: #10b981;
	}

	.metric-icon.info {
		background: #3b82f6;
	}

	.metric-icon.warning {
		background: #f59e0b;
	}

	.metric-icon.primary {
		background: #8b5cf6;
	}

	.metric-value {
		font-size: 2rem;
		font-weight: 700;
		color: #111827;
		line-height: 1;
		margin-bottom: 0.5rem;
	}

	.metric-change {
		display: flex;
		align-items: center;
		gap: 0.25rem;
		font-size: 0.875rem;
		font-weight: 500;
	}

	.metric-change.positive {
		color: #10b981;
	}

	.metric-change.negative {
		color: #ef4444;
	}

	.chart-card,
	.info-card {
		background: white;
		border-radius: 8px;
		border: 1px solid #e5e7eb;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
	}

	.card-header {
		padding: 1.5rem;
		border-bottom: 1px solid #e5e7eb;
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.card-header h3 {
		margin: 0;
		font-size: 1rem;
		font-weight: 600;
		color: #111827;
	}

	.card-body {
		padding: 1.5rem;
	}

	.chart-placeholder {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		min-height: 200px;
		text-align: center;
		color: #6b7280;
	}

	.performance-grid {
		display: grid;
		grid-template-columns: repeat(2, 1fr);
		gap: 1rem;
	}

	.performance-item {
		padding: 1rem;
		background: #f9fafb;
		border-radius: 6px;
	}

	.performance-label {
		font-size: 0.875rem;
		color: #6b7280;
		margin-bottom: 0.5rem;
		font-weight: 500;
	}

	.performance-value {
		font-size: 1.125rem;
		font-weight: 600;
		color: #111827;
		margin-bottom: 0.5rem;
	}

	.progress-bar {
		width: 100%;
		height: 4px;
		background: #e5e7eb;
		border-radius: 2px;
	overflow: hidden;
	}

	.progress-fill {
		height: 100%;
		background: linear-gradient(90deg, #3b82f6, #1d4ed8);
		border-radius: 2px;
		transition: width 0.3s ease;
	}

	.info-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
		gap: 1rem;
	}

	.info-item {
		text-align: center;
		padding: 1rem;
		background: #f9fafb;
		border-radius: 6px;
	}

	.info-label {
		font-size: 0.75rem;
		color: #6b7280;
		margin-bottom: 0.25rem;
		font-weight: 500;
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}

	.info-value {
		font-size: 0.875rem;
		font-weight: 600;
		color: #111827;
	}

	.status-badge {
		display: inline-flex;
		align-items: center;
		padding: 0.125rem 0.5rem;
		border-radius: 9999px;
		font-size: 0.75rem;
		font-weight: 500;
	}

	.status-badge.status-success {
		background: #10b98120;
		color: #10b981;
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

	.btn-secondary {
		background-color: white;
		color: #374151;
		border-color: #d1d5db;
	}

	.btn-secondary:hover {
		background-color: #f9fafb;
		border-color: #9ca3af;
	}

	.btn-sm {
		padding: 0.375rem 0.75rem;
		font-size: 0.75rem;
	}

	.text-sm {
		font-size: 0.875rem;
	}

	.text-gray-500 {
		color: #6b7280;
	}

	/* 响应式设计 */
	@media (max-width: 768px) {
		.metrics-grid {
			grid-template-columns: repeat(2, 1fr);
		}

		.charts-grid {
			grid-template-columns: 1fr;
		}

		.performance-grid {
			grid-template-columns: 1fr;
		}

		.info-grid {
			grid-template-columns: repeat(2, 1fr);
		}
	}

	@media (max-width: 480px) {
		.metrics-grid {
			grid-template-columns: 1fr;
		}

		.info-grid {
			grid-template-columns: 1fr;
		}
	}
</style>