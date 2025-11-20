<script>
	import { onMount } from 'svelte';
	import { Activity, Cpu, HardDrive, Wifi, Server, AlertTriangle, TrendingUp, TrendingDown, RefreshCw, Download, Clock } from 'lucide-svelte';

	// 实时数据状态
	let loading = false;
	let autoRefresh = true;
	let refreshInterval = null;

	// 性能数据
	let performance = {
		system: {
			cpu: 0,
			memory: 0,
			disk: 0,
			network: 0
		},
		metrics: {
			responseTime: 0,
			requestsPerSecond: 0,
			errorRate: 0,
			uptime: 0
		},
		processes: [],
		alerts: [],
		historical: []
	};

	// 生成模拟实时数据
	function generateSystemData() {
		return {
			cpu: Math.floor(Math.random() * 30) + 20,
			memory: Math.floor(Math.random() * 40) + 40,
			disk: Math.floor(Math.random() * 20) + 15,
			network: Math.floor(Math.random() * 60) + 10
		};
	}

	function generateMetricsData() {
		return {
			responseTime: Math.floor(Math.random() * 200) + 50,
			requestsPerSecond: (Math.random() * 100 + 20).toFixed(1),
			errorRate: (Math.random() * 5).toFixed(2),
			uptime: '15天 8小时 32分钟'
		};
	}

	function generateProcesses() {
		const processes = [
			{ name: 'metabase', pid: 1234, cpu: 12.5, memory: 256, status: 'running' },
			{ name: 'nginx', pid: 5678, cpu: 2.3, memory: 64, status: 'running' },
			{ name: 'postgres', pid: 9012, cpu: 8.7, memory: 512, status: 'running' },
			{ name: 'redis', pid: 3456, cpu: 1.2, memory: 128, status: 'running' },
			{ name: 'node', pid: 7890, cpu: 15.8, memory: 384, status: 'running' }
		];

		return processes.map(process => ({
			...process,
			cpu: process.cpu + (Math.random() - 0.5) * 5,
			memory: process.memory + Math.floor((Math.random() - 0.5) * 50)
		})).sort((a, b) => b.cpu - a.cpu);
	}

	function generateAlerts() {
		const alertTypes = [
			{ level: 'warning', message: 'CPU使用率超过80%', time: '2分钟前' },
			{ level: 'error', message: '数据库连接超时', time: '5分钟前' },
			{ level: 'info', message: '新版本发布完成', time: '10分钟前' },
			{ level: 'warning', message: '内存使用率达到75%', time: '15分钟前' }
		];

		return alertTypes.map((alert, index) => ({
			id: index + 1,
			...alert,
			resolved: Math.random() > 0.7
		})).filter(alert => !alert.resolved);
	}

	function generateHistoricalData() {
		const now = new Date();
		const data = [];

		for (let i = 60; i >= 0; i--) {
			const timestamp = new Date(now - i * 60000);
			data.push({
				timestamp: timestamp.toISOString(),
				cpu: Math.floor(Math.random() * 40) + 20,
				memory: Math.floor(Math.random() * 30) + 50,
				responseTime: Math.floor(Math.random() * 150) + 50,
				rps: Math.floor(Math.random() * 80) + 20
			});
		}

		return data;
	}

	// 更新性能数据
	function updatePerformance() {
		performance.system = generateSystemData();
		performance.metrics = generateMetricsData();
		performance.processes = generateProcesses();
		performance.alerts = generateAlerts();
		performance.historical = generateHistoricalData();
	}

	onMount(() => {
		loadPerformance();
		startAutoRefresh();
	});

	function loadPerformance() {
		loading = true;
		setTimeout(() => {
			updatePerformance();
			loading = false;
		}, 600);
	}

	function startAutoRefresh() {
		if (autoRefresh) {
			refreshInterval = setInterval(() => {
				updatePerformance();
			}, 3000);
		}
	}

	function stopAutoRefresh() {
		if (refreshInterval) {
			clearInterval(refreshInterval);
			refreshInterval = null;
		}
	}

	function toggleAutoRefresh() {
		autoRefresh = !autoRefresh;
		if (autoRefresh) {
			startAutoRefresh();
		} else {
			stopAutoRefresh();
		}
	}

	function getStatusClass(value, type) {
		if (type === 'cpu' || type === 'memory') {
			if (value > 80) return 'critical';
			if (value > 60) return 'warning';
			return 'normal';
		}
		if (type === 'responseTime') {
			if (value > 500) return 'critical';
			if (value > 200) return 'warning';
			return 'normal';
		}
		if (type === 'errorRate') {
			if (value > 5) return 'critical';
			if (value > 1) return 'warning';
			return 'normal';
		}
		return 'normal';
	}

	function getAlertIcon(level) {
		switch (level) {
			case 'error': return AlertTriangle;
			case 'warning': return AlertTriangle;
			default: return Activity;
		}
	}

	function getAlertClass(level) {
		switch (level) {
			case 'error': return 'alert-error';
			case 'warning': return 'alert-warning';
			default: return 'alert-info';
		}
	}

	function formatUptime(uptime) {
		return uptime;
	}

	function exportPerformanceData() {
		const data = {
			timestamp: new Date().toISOString(),
			system: performance.system,
			metrics: performance.metrics,
			processes: performance.processes,
			alerts: performance.alerts
		};

		const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
		const url = URL.createObjectURL(blob);
		const a = document.createElement('a');
		a.href = url;
		a.download = `performance_${new Date().toISOString().split('T')[0]}.json`;
		a.click();
		URL.revokeObjectURL(url);
	}

	// 清理定时器
	import { onDestroy } from 'svelte';
	onDestroy(() => {
		stopAutoRefresh();
	});
</script>

<div class="performance-page">
	<!-- 页面标题 -->
	<div class="page-header">
		<div class="header-left">
			<h1>性能监控</h1>
			<p class="page-description">系统性能和资源监控</p>
		</div>
		<div class="header-right">
			<div class="refresh-controls">
				<label class="toggle-label">
					<input
						type="checkbox"
						bind:checked={autoRefresh}
						on:change={toggleAutoRefresh}
					/>
					<span class="toggle-slider"></span>
					自动刷新
				</label>
				<button class="btn btn-secondary" on:click={loadPerformance}>
					<div class:spinning={loading}>
						<RefreshCw size={16} />
					</div>
					刷新
				</button>
			</div>
			<button class="btn btn-primary" on:click={exportPerformanceData}>
				<Download size={16} />
				导出数据
			</button>
		</div>
	</div>

	{#if loading}
		<div class="loading-state">
			<div class="loading-spinner"></div>
			<p>加载性能数据中...</p>
		</div>
	{:else}
		<!-- 系统资源概览 -->
		<div class="resources-section">
			<div class="resource-card">
				<div class="resource-header">
					<div class="resource-icon cpu">
						<Cpu size={24} />
					</div>
					<div class="resource-title">CPU 使用率</div>
				</div>
				<div class="resource-value {getStatusClass(performance.system.cpu, 'cpu')}">
					{performance.system.cpu}%
				</div>
				<div class="resource-chart">
					<div class="progress-bar">
						<div class="progress-fill cpu" style="width: {performance.system.cpu}%"></div>
					</div>
				</div>
				<div class="resource-details">
					<span>8 核心 @ 2.4GHz</span>
					<span class="trend">
						<TrendingUp size={14} />
						+2.3%
					</span>
				</div>
			</div>

			<div class="resource-card">
				<div class="resource-header">
					<div class="resource-icon memory">
						<Activity size={24} />
					</div>
					<div class="resource-title">内存使用率</div>
				</div>
				<div class="resource-value {getStatusClass(performance.system.memory, 'memory')}">
					{performance.system.memory}%
				</div>
				<div class="resource-chart">
					<div class="progress-bar">
						<div class="progress-fill memory" style="width: {performance.system.memory}%"></div>
					</div>
				</div>
				<div class="resource-details">
					<span>12.3 GB / 16 GB</span>
					<span class="trend">
						<TrendingDown size={14} />
						-1.8%
					</span>
				</div>
			</div>

			<div class="resource-card">
				<div class="resource-header">
					<div class="resource-icon disk">
						<HardDrive size={24} />
					</div>
					<div class="resource-title">磁盘使用率</div>
				</div>
				<div class="resource-value {getStatusClass(performance.system.disk, 'disk')}">
					{performance.system.disk}%
				</div>
				<div class="resource-chart">
					<div class="progress-bar">
						<div class="progress-fill disk" style="width: {performance.system.disk}%"></div>
					</div>
				</div>
				<div class="resource-details">
					<span>125 GB / 500 GB</span>
					<span class="trend stable">
						<Activity size={14} />
						稳定
					</span>
				</div>
			</div>

			<div class="resource-card">
				<div class="resource-header">
					<div class="resource-icon network">
						<Wifi size={24} />
					</div>
					<div class="resource-title">网络使用率</div>
				</div>
				<div class="resource-value {getStatusClass(performance.system.network, 'network')}">
					{performance.system.network}%
				</div>
				<div class="resource-chart">
					<div class="progress-bar">
						<div class="progress-fill network" style="width: {performance.system.network}%"></div>
					</div>
				</div>
				<div class="resource-details">
					<span>↑ 2.3 MB/s ↓ 8.7 MB/s</span>
					<span class="trend">
						<TrendingUp size={14} />
						+12.5%
					</span>
				</div>
			</div>
		</div>

		<!-- 性能指标 -->
		<div class="metrics-section">
			<div class="metrics-grid">
				<div class="metric-card">
					<div class="metric-icon response-time">
						<Clock size={20} />
					</div>
					<div class="metric-content">
						<div class="metric-value {getStatusClass(performance.metrics.responseTime, 'responseTime')}">
							{performance.metrics.responseTime}ms
						</div>
						<div class="metric-label">平均响应时间</div>
					</div>
				</div>

				<div class="metric-card">
					<div class="metric-icon rps">
						<Server size={20} />
					</div>
					<div class="metric-content">
						<div class="metric-value">
							{performance.metrics.requestsPerSecond}
						</div>
						<div class="metric-label">每秒请求数</div>
					</div>
				</div>

				<div class="metric-card">
					<div class="metric-icon error-rate">
						<AlertTriangle size={20} />
					</div>
					<div class="metric-content">
						<div class="metric-value {getStatusClass(performance.metrics.errorRate, 'errorRate')}">
							{performance.metrics.errorRate}%
						</div>
						<div class="metric-label">错误率</div>
					</div>
				</div>

				<div class="metric-card">
					<div class="metric-icon uptime">
						<Activity size={20} />
					</div>
					<div class="metric-content">
						<div class="metric-value">{formatUptime(performance.metrics.uptime)}</div>
						<div class="metric-label">运行时间</div>
					</div>
				</div>
			</div>
		</div>

		<!-- 实时图表和进程监控 -->
		<div class="monitoring-section">
			<!-- 实时性能图表 -->
			<div class="chart-card large">
				<div class="card-header">
					<h3>实时性能监控</h3>
					<div class="chart-legend">
						<span class="legend-item">
							<span class="legend-color cpu"></span>
							CPU
						</span>
						<span class="legend-item">
							<span class="legend-color memory"></span>
							内存
						</span>
						<span class="legend-item">
							<span class="legend-color response"></span>
							响应时间
						</span>
					</div>
				</div>
				<div class="card-body">
					<div class="performance-chart">
						<div class="chart-grid">
							{#each performance.historical.slice(-20) as point}
								<div class="chart-point-container">
									<div class="chart-points">
										<div class="chart-point cpu" style="bottom: {(point.cpu / 100) * 100}%"></div>
										<div class="chart-point memory" style="bottom: {(point.memory / 100) * 100}%"></div>
										<div class="chart-point response" style="bottom: {(point.responseTime / 500) * 100}%"></div>
									</div>
								</div>
							{/each}
						</div>
					</div>
				</div>
			</div>

			<!-- 进程监控 -->
			<div class="process-card">
				<div class="card-header">
					<h3>进程监控</h3>
					<Server size={16} class="card-icon" />
				</div>
				<div class="card-body">
					<div class="process-list">
						{#each performance.processes.slice(0, 5) as process}
							<div class="process-item">
								<div class="process-info">
									<div class="process-name">{process.name}</div>
									<div class="process-pid">PID: {process.pid}</div>
								</div>
								<div class="process-metrics">
									<div class="process-metric">
										<span class="metric-label">CPU</span>
										<span class="metric-value">{process.cpu.toFixed(1)}%</span>
									</div>
									<div class="process-metric">
										<span class="metric-label">内存</span>
										<span class="metric-value">{process.memory}MB</span>
									</div>
								</div>
								<div class="process-status">
									<span class="status-dot running"></span>
								</div>
							</div>
						{/each}
					</div>
				</div>
			</div>

			<!-- 告警信息 -->
			<div class="alerts-card">
				<div class="card-header">
					<h3>告警信息</h3>
					<AlertTriangle size={16} class="card-icon" />
				</div>
				<div class="card-body">
					<div class="alerts-list">
						{#each performance.alerts.slice(0, 4) as alert}
							<div class="alert-item {getAlertClass(alert.level)}">
								<svelte:component this={getAlertIcon(alert.level)} size={16} class="alert-icon" />
								<div class="alert-content">
									<div class="alert-message">{alert.message}</div>
									<div class="alert-time">{alert.time}</div>
								</div>
							</div>
						{/each}
						{#if performance.alerts.length === 0}
							<div class="no-alerts">
								<Activity size={24} />
								<p>暂无告警信息</p>
							</div>
						{/if}
					</div>
				</div>
			</div>
		</div>
	{/if}
</div>

<style>
	.performance-page {
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

	.header-right {
		display: flex;
		align-items: center;
		gap: 1rem;
	}

	.refresh-controls {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}

	.toggle-label {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 0.875rem;
		color: #374151;
		cursor: pointer;
	}

	.toggle-label input[type="checkbox"] {
		display: none;
	}

	.toggle-slider {
		width: 44px;
		height: 24px;
		background: #d1d5db;
		border-radius: 12px;
		position: relative;
		transition: background 0.3s ease;
	}

	.toggle-slider::after {
		content: '';
		position: absolute;
		width: 18px;
		height: 18px;
		background: white;
		border-radius: 50%;
		top: 3px;
		left: 3px;
		transition: transform 0.3s ease;
	}

	input[type="checkbox"]:checked + .toggle-slider {
		background: #3b82f6;
	}

	input[type="checkbox"]:checked + .toggle-slider::after {
		transform: translateX(20px);
	}

	.resources-section {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
		gap: 1rem;
	}

	.resource-card {
		background: white;
		border: 1px solid #e5e7eb;
		border-radius: 8px;
		padding: 1.5rem;
	}

	.resource-header {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		margin-bottom: 1rem;
	}

	.resource-icon {
		width: 40px;
		height: 40px;
		border-radius: 8px;
		display: flex;
		align-items: center;
		justify-content: center;
		color: white;
	}

	.resource-icon.cpu { background: #3b82f6; }
	.resource-icon.memory { background: #10b981; }
	.resource-icon.disk { background: #f59e0b; }
	.resource-icon.network { background: #8b5cf6; }

	.resource-title {
		font-size: 0.875rem;
		font-weight: 600;
		color: #374151;
	}

	.resource-value {
		font-size: 2.5rem;
		font-weight: 700;
		color: #111827;
		margin-bottom: 1rem;
	}

	.resource-value.normal { color: #10b981; }
	.resource-value.warning { color: #f59e0b; }
	.resource-value.critical { color: #ef4444; }

	.progress-bar {
		width: 100%;
		height: 8px;
		background: #e5e7eb;
		border-radius: 4px;
		overflow: hidden;
		margin-bottom: 0.75rem;
	}

	.progress-fill {
		height: 100%;
		border-radius: 4px;
		transition: width 0.3s ease;
	}

	.progress-fill.cpu { background: linear-gradient(90deg, #3b82f6, #1d4ed8); }
	.progress-fill.memory { background: linear-gradient(90deg, #10b981, #059669); }
	.progress-fill.disk { background: linear-gradient(90deg, #f59e0b, #d97706); }
	.progress-fill.network { background: linear-gradient(90deg, #8b5cf6, #7c3aed); }

	.resource-details {
		display: flex;
		justify-content: space-between;
		align-items: center;
		font-size: 0.75rem;
		color: #6b7280;
	}

	.trend {
		display: flex;
		align-items: center;
		gap: 0.25rem;
		color: #10b981;
	}

	.trend.stable {
		color: #6b7280;
	}

	.metrics-section {
		background: white;
		border: 1px solid #e5e7eb;
		border-radius: 8px;
		padding: 1.5rem;
	}

	.metrics-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
		gap: 1rem;
	}

	.metric-card {
		display: flex;
		align-items: center;
		gap: 1rem;
		padding: 1rem;
		background: #f9fafb;
		border-radius: 6px;
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

	.metric-icon.response-time { background: #3b82f6; }
	.metric-icon.rps { background: #10b981; }
	.metric-icon.error-rate { background: #ef4444; }
	.metric-icon.uptime { background: #8b5cf6; }

	.metric-value {
		font-size: 1.5rem;
		font-weight: 700;
		color: #111827;
		line-height: 1;
	}

	.metric-value.normal { color: #10b981; }
	.metric-value.warning { color: #f59e0b; }
	.metric-value.critical { color: #ef4444; }

	.metric-label {
		font-size: 0.875rem;
		color: #6b7280;
		margin-top: 0.25rem;
	}

	.monitoring-section {
		display: grid;
		grid-template-columns: 2fr 1fr;
		gap: 1.5rem;
	}

	.chart-card.large {
		grid-column: 1 / -1;
		grid-row: 1;
	}

	.process-card {
		grid-column: 2;
		grid-row: 2;
	}

	.alerts-card {
		grid-column: 1;
		grid-row: 2;
	}

	.chart-card,
	.process-card,
	.alerts-card {
		background: white;
		border: 1px solid #e5e7eb;
		border-radius: 8px;
		overflow: hidden;
	}

	.card-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 1.5rem;
		border-bottom: 1px solid #e5e7eb;
	}

	.card-header h3 {
		margin: 0;
		font-size: 1rem;
		font-weight: 600;
		color: #111827;
	}

	.card-icon {
		color: #6b7280;
	}

	.chart-legend {
		display: flex;
		gap: 1rem;
	}

	.legend-item {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 0.875rem;
		color: #6b7280;
	}

	.legend-color {
		width: 12px;
		height: 12px;
		border-radius: 2px;
	}

	.legend-color.cpu { background: #3b82f6; }
	.legend-color.memory { background: #10b981; }
	.legend-color.response { background: #f59e0b; }

	.card-body {
		padding: 1.5rem;
	}

	.performance-chart {
		height: 200px;
		position: relative;
	}

	.chart-grid {
		display: flex;
		align-items: end;
		justify-content: space-between;
		height: 100%;
		gap: 2px;
		padding: 0 1rem;
	}

	.chart-point-container {
		flex: 1;
		display: flex;
		align-items: end;
		justify-content: center;
	}

	.chart-points {
		width: 100%;
		max-width: 30px;
		height: 100%;
		position: relative;
		display: flex;
		flex-direction: column;
		justify-content: end;
		gap: 1px;
	}

	.chart-point {
		width: 100%;
		background: currentColor;
		border-radius: 2px 2px 0 0;
		min-height: 2px;
	}

	.chart-point.cpu { color: #3b82f6; }
	.chart-point.memory { color: #10b981; }
	.chart-point.response { color: #f59e0b; }

	.process-list,
	.alerts-list {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	.process-item {
		display: flex;
		align-items: center;
		gap: 1rem;
		padding: 1rem;
		background: #f9fafb;
		border-radius: 6px;
	}

	.process-info {
		flex: 1;
	}

	.process-name {
		font-weight: 600;
		color: #111827;
		font-size: 0.875rem;
	}

	.process-pid {
		font-size: 0.75rem;
		color: #6b7280;
	}

	.process-metrics {
		display: flex;
		gap: 1.5rem;
	}

	.process-metric {
		display: flex;
		flex-direction: column;
		align-items: center;
	}

	.metric-label {
		font-size: 0.75rem;
		color: #6b7280;
		margin-bottom: 0.25rem;
	}

	.metric-value {
		font-weight: 600;
		color: #374151;
		font-size: 0.875rem;
	}

	.status-dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
	}

	.status-dot.running {
		background: #10b981;
	}

	.alert-item {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		padding: 1rem;
		border-radius: 6px;
	}

	.alert-item.alert-error {
		background: #fef2f2;
		color: #991b1b;
	}

	.alert-item.alert-warning {
		background: #fffbeb;
		color: #92400e;
	}

	.alert-item.alert-info {
		background: #eff6ff;
		color: #1d4ed8;
	}

	.alert-icon {
		flex-shrink: 0;
	}

	.alert-content {
		flex: 1;
	}

	.alert-message {
		font-weight: 500;
		font-size: 0.875rem;
	}

	.alert-time {
		font-size: 0.75rem;
		opacity: 0.7;
		margin-top: 0.25rem;
	}

	.no-alerts {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 2rem;
		color: #6b7280;
	}

	.no-alerts p {
		margin: 0.5rem 0 0 0;
		font-size: 0.875rem;
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

	.spinning {
		animation: spin 1s linear infinite;
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

	/* 响应式设计 */
	@media (max-width: 768px) {
		.page-header {
			flex-direction: column;
			align-items: flex-start;
			gap: 1rem;
		}

		.header-right {
			width: 100%;
			justify-content: space-between;
		}

		.resources-section {
			grid-template-columns: repeat(2, 1fr);
		}

		.metrics-grid {
			grid-template-columns: repeat(2, 1fr);
		}

		.monitoring-section {
			grid-template-columns: 1fr;
		}

		.chart-card.large,
		.process-card,
		.alerts-card {
			grid-column: 1;
			grid-row: auto;
		}

		.chart-legend {
			flex-wrap: wrap;
			gap: 0.5rem;
		}
	}

	@media (max-width: 480px) {
		.resources-section {
			grid-template-columns: 1fr;
		}

		.metrics-grid {
			grid-template-columns: 1fr;
		}

		.process-metrics {
			gap: 1rem;
		}
	}
</style>